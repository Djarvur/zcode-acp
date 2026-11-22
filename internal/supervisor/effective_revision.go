package supervisor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// The effective builtin revision: the revision string the runtime's OWN
// registry carries for a spawn — zcode-builtin:<winnerRev>:<sha256(resolved
// winner path)> over the two-candidate release selection the desktop's
// provider stack performs at read time.
//
// Wire truth (V-SRC zcode-src 3.17.1 clone, corroborating tier for the
// shipped 3.18.2 family; recorded in docs/protocol/install-discovery.md):
//
//   - BUNDLED candidate: the install's own
//     <root>/resources/config/provider/zcode-builtin.json (the source's
//     bundledFilePath — resolveBundledZCodeBuiltinProviderConfig).
//   - ACTIVE candidate: the desktop's runtime cache
//     <dataBaseDir>/.zcode/v2/runtime/provider/<platform>/<appVersion>/endpoint-<32-hex>/zcode-builtin.json
//     (resolveZCodeBuiltinCachePaths; platform linux-x86_64 is this wrapper's
//     v1 OS scope; the endpoint key is derived from the zcode endpoint origin,
//     so on disk it is DISCOVERED by glob, never computed).
//   - Selection (selectReleaseCandidate): only a STRICTLY greater active
//     revision replaces the bundled baseline; every tie falls to bundled —
//     including the equal-revision/different-bytes conflict, which the source
//     resolves to bundled explicitly ("同 revision 冲突属于 Active 缓存失效").
//   - The registry match is exact-string: an account push whose
//     basedOnZCodeBuiltinRevision differs from the runtime's own
//     zcodeBuiltinRevision is schema-accepted and silently ignored
//     (V-SRC registry-service.ts — the mismatch arm keeps serving the
//     previous registry), which is why the raw discovery-recorded revision
//     (hash over the runtime-tree copy) never applied.
//
// Deterministic, no network, no runtime spawn. Missing active candidate →
// bundled-only — byte-identical to the pre-change BuiltinRevision output, so
// every CI fixture and golden digest (fake tier has no runtime cache) stays
// unchanged.

// builtinConfigRel is the install-relative dir of the desktop's own bundled
// builtin table (the BUNDLED candidate under Install.Root).
const builtinConfigRel = "resources/config/provider"

// ErrNoBuiltinCandidate is the neither-candidate-readable outcome: no usable
// builtin revision exists for the install (bundled and active both missing
// or undecodable). The error wraps both candidates' failures — actionable
// diagnostics, never a silent empty revision string.
var ErrNoBuiltinCandidate = errors.New("zcode: no readable builtin revision candidate (bundled or active)")

// installAppVersionPattern digs the version-looking segment out of an
// install path segment (the layout names carry it — zcode-3.18.2/...,
// ZCode-3.18.2.AppImage; the reference's installedAppVersion(appBundlePath())
// best-effort port). Best-effort by design: a missed segment leaves
// appVersion "" and the active candidate unreachable (bundled-only).
var installAppVersionPattern = regexp.MustCompile(`\d+\.\d+(\.\d+)?`)

// InstallAppVersion digs the desktop version out of the resolved install
// paths (deepest segment first — e.g. zcode-3.18.2/squashfs-root), the
// appVersion the runtime-cache active candidate is scoped by; "" when no
// segment matches.
func InstallAppVersion(install Install) string {
	paths := []string{install.Binary, install.Root, install.Entrypoint}

	for _, path := range paths {
		segs := strings.Split(path, string(filepath.Separator))

		for i := len(segs) - 1; i >= 0; i-- {
			if v := installAppVersionPattern.FindString(segs[i]); v != "" {
				return v
			}
		}
	}

	return ""
}

// EffectiveBuiltinRevision computes the revision string the runtime's own
// registry carries for install: the two-candidate selection over the bundled
// and active zcode-builtin.json copies, formatted with the exact
// BuiltinRevision formula (zcode-builtin:<rev>:<sha256(resolved path)>) over
// the WINNER's resolved path. appVersion scopes the runtime-cache active
// tree (InstallAppVersion digs it); dataBaseDir is the desktop data base
// (the user home in production, injectable for tests — "" disables the
// active lookup entirely). Missing active candidate → bundled-only, the
// pre-change behavior preserved byte-for-byte.
func EffectiveBuiltinRevision(install Install, appVersion, dataBaseDir string) (string, error) {
	bundled := bundledBuiltinPath(install)
	active := activeBuiltinPath(dataBaseDir, appVersion)

	bundledRev, bundledErr := builtinRevisionNumber(bundled)
	activeRev, activeErr := builtinRevisionNumber(active)

	switch {
	case bundledErr == nil && activeErr == nil && activeRev > bundledRev:
		// Strictly-newer active cache wins (selectReleaseCandidate's reduce
		// replaces only on strictly-greater).
		return BuiltinRevision(active, activeRev), nil
	case bundledErr == nil:
		// Ties (same revision, any bytes) always fall here — the bundled
		// baseline is the trustworthy copy (V-SRC: the equal-revision
		// conflict branch returns bundled explicitly).
		return BuiltinRevision(bundled, bundledRev), nil
	case activeErr == nil:
		// Single-candidate arm of the same reduce: an unusable bundled copy
		// never blocks a usable active cache.
		return BuiltinRevision(active, activeRev), nil
	default:
		return "", fmt.Errorf("%w: bundled %q: %w; active %q: %w",
			ErrNoBuiltinCandidate, bundled, bundledErr, active, activeErr)
	}
}

// DefaultEffectiveRevision is the production input set: the appVersion dug
// from the install's own paths and the user home as dataBaseDir. A home
// resolution failure degrades to bundled-only (the active candidate needs a
// base dir; the bundled baseline is always local).
func DefaultEffectiveRevision(install Install) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	return EffectiveBuiltinRevision(install, InstallAppVersion(install), home)
}

// bundledBuiltinPath resolves the BUNDLED candidate: the install's own
// resources copy when the tree carries it (the runtime's bundledFilePath —
// the copy the spawn root itself ships), else the discovery-recorded
// BuiltinFile (CI fixtures, explicit overrides and override-only trees have
// no resources copy; the recorded file stays the bundled baseline so the
// existing digests hold byte-for-byte).
func bundledBuiltinPath(install Install) string {
	if install.Root != "" {
		cand := filepath.Join(install.Root, builtinConfigRel, builtinFileName)
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand
		}
	}

	return install.BuiltinFile
}

// activeBuiltinPath resolves the runtime-cache ACTIVE candidate:
// <dataBaseDir>/<builtinProviderRel>/<appVersion>/endpoint-*/zcode-builtin.json.
// The endpoint dir name is the runtime's origin-derived key — discovered by
// listing, never computed. Multiple endpoint dirs under one version dir:
// the lexicographically-last sorted name that carries the file (documented
// assumption — one endpoint dir per origin in practice). "" when any level
// is absent (no base dir, no version dir, no endpoint copy).
func activeBuiltinPath(dataBaseDir, appVersion string) string {
	if dataBaseDir == "" || appVersion == "" {
		return ""
	}

	verDir := filepath.Join(dataBaseDir, builtinProviderRel, appVersion)
	endpoints, err := versionDirs(verDir)
	if err != nil {
		return ""
	}

	slices.Sort(endpoints)
	slices.Reverse(endpoints)

	for _, name := range endpoints {
		if !strings.HasPrefix(name, "endpoint-") {
			continue
		}
		cand := filepath.Join(verDir, name, builtinFileName)
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand
		}
	}

	return ""
}

// errNoCandidatePath is the empty-input arm of builtinRevisionNumber (no
// candidate resolved at that level).
var errNoCandidatePath = errors.New("no candidate path resolved")

// errNoRevisionNumber is the missing-number arm (the release decode requires
// the top-level revision; its absence invalidates the candidate).
var errNoRevisionNumber = errors.New("no top-level revision number")

// builtinRevisionNumber reads the top-level revision number out of a
// zcode-builtin.json release document; a document without the number is an
// invalid candidate (V-SRC: the release decode requires it), never a zero.
func builtinRevisionNumber(path string) (int, error) {
	if path == "" {
		return 0, errNoCandidatePath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}

	var doc struct {
		Revision *int `json:"revision"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Revision == nil {
		return 0, fmt.Errorf("parse %s: %w", path, errNoRevisionNumber)
	}

	return *doc.Revision, nil
}
