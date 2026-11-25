// Package supervisor implements the ZCode-desktop back-leg plumbing: install
// discovery per the Phase 1 matrix, child spawning, and process supervision
// (BRIDGE-03). Plan 03-04 ships the discovery half; the seams on Probers keep
// the whole matrix testable without a real desktop install.
//
// Provenance: the probe order, probe set, locate commands, decoy rules, and
// the revision formula are a faithful translation of the Phase-1-verified
// matrix in docs/protocol/install-discovery.md (rows :19-25, entrypoint
// invariant :38, desktop-entry procedure :65-69, revision formula :82,
// recommended order :93-97).
package supervisor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Layout values of Install (RESEARCH Pattern 5; install-discovery.md rows).
const (
	layoutAppImage = "appimage"
)

// Matrix path facts (docs/protocol/install-discovery.md).
const (
	appImageSuffix  = ".AppImage"
	entrypointRel   = "resources/glm/zcode.cjs"
	builtinFileName = "zcode-builtin.json"

	// bundledProviderRel is the bundled builtin copy's directory under the
	// install root — <root>/resources/config/provider/zcode-builtin.json, the
	// copy every shipped layout carries (field-verified in the QA-02 VM's
	// squashfs-root, EXP-VM-QA02).
	bundledProviderRel = "resources/config/provider"

	// probeCount is the documented fixed probe set (install-discovery.md
	// :93-97) plus the runtime-provider-tree row and its install-side bundled
	// fallback row; the platform install stage adds its own rows.
	probeCount = 8
)

// ErrNoInstall is the empty-result outcome: no documented layout matched.
// install-discovery.md :34 makes the empty set a first-class result — the
// returned probe rows are the actionable diagnostics, never a guess.
var ErrNoInstall = errors.New("zcode: no ZCode desktop install found")

// Install describes one located ZCode desktop install. Field semantics follow
// docs/protocol/install-discovery.md: the relative entrypoint path
// resources/glm/zcode.cjs under Root is invariant across all layouts (:38).
type Install struct {
	Root        string // dir containing the Electron binary
	Binary      string // <root>/zcode
	Entrypoint  string // <root>/resources/glm/zcode.cjs (invariant relative path)
	BuiltinFile string // ZCODE_BUILTIN_PROVIDER_CONFIG_FILE value
	Layout      string // "unpacked-appimage" | "appimage" | "deb" | "rpm" | "explicit"
	// AppImage is the resolved packed AppImage path; non-empty means this
	// backend is mount-first (D-01/D-04): discovery is a pure read, so the
	// mount happens only after selection, and PrepareMount replaces the
	// empty Root/Binary/Entrypoint with the mounted tree.
	AppImage string
	// Version is the <ver> parsed from the AppImage filename or the unpacked
	// dir name (install-discovery.md :33) — the version the spawn line no
	// longer carries once the root lives under a volatile mount path.
	Version string
}

// RequiresMount reports whether the install is a packed AppImage that must be
// mounted before any spawn: AppImage set and Root not yet resolved to the
// mount (D-01/D-02). The mounted install (Root set) reports false, so the
// spawn seam only ever sees a mounted install.
func (i Install) RequiresMount() bool {
	return i.AppImage != "" && i.Root == ""
}

// Probe is one diagnostics row: the locate command a probe ran and what it
// returned, whether it hit or missed. The rows are the product of an empty
// discovery result (install-discovery.md :34).
type Probe struct {
	Name   string // documented probe name (fixed set, fixed order)
	Query  string // the locate command pattern from the matrix
	Result string // what the probe returned (hit, miss, or rejection)
}

// Probers is the seamed discovery unit: one injectable seam per OS contact.
// Tests stub the seams against t.TempDir() fixture trees and canned query
// outputs; production fills them with DefaultProbers, whose seams shell out
// per the matrix commands. A nil seam records an honest miss.
type Probers struct {
	// Override is order step 1: an explicit root or builtin file from the
	// wrapper's own configuration wins over any on-disk discovery.
	Override func() (Install, bool)
	// XDGQuery runs `xdg-mime query default x-scheme-handler/zcode` and
	// returns the default handler entry name.
	XDGQuery func(ctx context.Context) (string, error)
	// AppDirs returns the application-dir scan candidates: packed
	// ZCode*.AppImage files (incl. the user-facing symlink) and unpacked
	// zcode-*/squashfs-root dirs (matrix rows 1-2).
	AppDirs func() []string
	// DpkgQuery returns the deb file list for zcode (empty = rc remnant).
	DpkgQuery func(ctx context.Context) (string, error)
	// RpmQuery returns the rpm package identity for zcode (where applicable).
	RpmQuery func(ctx context.Context) (string, error)
	// Home is the user's home dir (~ for the applications scan and the
	// runtime-provider tree).
	Home string
	// OptDir is the package-family root (the deb's /opt/ZCode).
	OptDir string
	// Windows is the only Windows-specific seam group: the native install
	// directory sources and the uninstall-registry reader. It stays zero on
	// Linux and Darwin, where the platform stage supplies its own probes.
	Windows WindowsSeams
}

// UninstallEntry is one uninstall-registry record reduced to the fields the
// Windows install model reads. Key is the full hive path (for example
// HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\ZCode) and is what
// the diagnostics rows name.
//
// The type lives in this shared file so the Windows record rules are testable
// on every platform: only the windows-tagged seam reads a real hive.
type UninstallEntry struct {
	Key             string
	DisplayName     string
	DisplayVersion  string
	InstallLocation string
	DisplayIcon     string
}

// WindowsInstallDir is one native Windows install-directory source: the
// diagnostics Name and Query, the discovered Root ("" when the host does not
// define the source), and the Layout a hit records.
type WindowsInstallDir struct {
	Name   string
	Query  string
	Root   string
	Layout string
}

// WindowsSeams is the injectable Windows OS-contact surface. Dirs are the
// environment-derived install locations in probe order; Uninstall returns the
// uninstall-registry records. Tests stub both; the windows-tagged wiring
// fills them from the environment and the real registry.
type WindowsSeams struct {
	Dirs      []WindowsInstallDir
	Uninstall func() ([]UninstallEntry, error)
}

// Discover resolves the ZCode desktop install in the documented probe order
// (install-discovery.md §Recommended discovery order): override, desktop
// entry, AppImage paths, package paths, runtime provider tree. The first
// probe that yields a spawnable install wins and probes never merge; every
// probe appends a Probe row whether it hits or misses, and an empty result
// errors with ErrNoInstall carrying every row as diagnostics. Volatile
// /tmp/.mount_* decoys are recorded-and-rejected, never resolved (row 5).
func (p Probers) Discover(ctx context.Context) (Install, []Probe, error) {
	probes := make([]Probe, 0, probeCount)
	var winner Install
	found := false

	add := func(row Probe, inst Install, ok bool) {
		probes = append(probes, row)
		if ok && !found {
			winner, found = inst, true
		}
	}

	add(p.probeOverride())

	// The platform install-layout stage: Linux runs the XDG/AppImage/dpkg/
	// opt/rpm probes in the documented order; Windows runs the native install
	// stage (directory sources, then the uninstall registry); Darwin answers
	// an honest diagnostic row.
	stageRows, stageInstall, stageOK := p.probeInstallStage(ctx, found)
	probes = append(probes, stageRows...)
	if stageOK && !found {
		winner, found = stageInstall, true
	}

	row, builtin := p.probeBuiltinRuntime()
	probes = append(probes, row)
	if found && winner.BuiltinFile == "" {
		winner.BuiltinFile = builtin
	}
	bundledRow, bundled := p.probeBuiltinBundled(winner, found)
	probes = append(probes, bundledRow)
	if found && winner.BuiltinFile == "" {
		winner.BuiltinFile = bundled
	}
	if !found {
		return Install{}, probes, emptyResultError(probes)
	}

	return winner, probes, nil
}

// BuiltinRevision formats the zcode builtin revision string for an active
// zcode-builtin.json path and its recorded revision number:
// zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>
// (install-discovery.md :82). The hash input is the resolved absolute PATH
// STRING, not the file contents.
func BuiltinRevision(builtinFile string, rev int) string {
	resolved, err := filepath.EvalSymlinks(builtinFile)
	if err != nil {
		resolved = builtinFile
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		abs = resolved
	}
	sum := sha256.Sum256([]byte(abs))

	return fmt.Sprintf("zcode-builtin:%d:%s", rev, hex.EncodeToString(sum[:]))
}

// emptyResultError wraps ErrNoInstall with one line per probe row — the
// actionable diagnostics (install-discovery.md :34: the empty set must be
// reported with every probe attempted, never guessed away).
func emptyResultError(probes []Probe) error {
	lines := make([]string, 0, len(probes))
	for _, row := range probes {
		lines = append(lines, "  - "+row.Name+": "+row.Result)
	}

	return fmt.Errorf("%w; probes attempted:\n%s", ErrNoInstall, strings.Join(lines, "\n"))
}

// mountDecoy reports whether path matches the volatile running-mount decoy
// pattern /tmp/.mount_* (install-discovery.md row 5: do-not-rely — recorded
// and rejected, never resolved as a spawn root; localhost listeners are
// never probed at all).
func mountDecoy(path string) bool {
	if path == "" {
		return false
	}

	return strings.HasPrefix(filepath.Clean(path), "/tmp/.mount_")
}

// volatileInstallPath names the first install path matching the volatile
// running-mount decoy pattern (install-discovery.md row 5) — the root of a
// tree install or the AppImage of a packed one; "" when neither is volatile.
// The override probe rejects the whole override rather than resolving it.
func volatileInstallPath(inst Install) string {
	for _, path := range []string{inst.Root, inst.AppImage} {
		if mountDecoy(path) {
			return path
		}
	}

	return ""
}

// bundleInstall validates the layout-invariant spawn surface at root and
// returns the Install for it (binary <root>/zcode, entrypoint
// <root>/resources/glm/zcode.cjs — install-discovery.md :38).
func bundleInstall(root, layout string) (Install, bool) {
	return bundleInstallBinary(root, layout, platformBinaryName())
}

// bundleInstallBinary is bundleInstall with an explicit binary name: the
// Windows stage validates <root>\ZCode.exe on hosts whose
// platformBinaryName() answers a different name, so the same invariant tree
// check serves every platform.
func bundleInstallBinary(root, layout, exeName string) (Install, bool) {
	if root == "" || mountDecoy(root) {
		return Install{}, false
	}
	inst := Install{
		Root:       root,
		Binary:     filepath.Join(root, exeName),
		Entrypoint: filepath.Join(root, entrypointRel),
		Layout:     layout,
	}
	if _, err := os.Stat(inst.Binary); err != nil {
		return Install{}, false
	}
	if _, err := os.Stat(inst.Entrypoint); err != nil {
		return Install{}, false
	}

	return inst, true
}

// ExplicitRootInstall builds the Install for an explicitly configured root —
// the discovery order-step-1 override (install-discovery.md :93): the same
// layout-invariant spawn surface every discovered bundle must carry
// (<root>/zcode + <root>/resources/glm/zcode.cjs, :38) is validated here, so
// an override that names a non-spawnable tree is an honest miss the caller
// reports, never a spawn-time guess. Layout "explicit" marks the override.
func ExplicitRootInstall(root string) (Install, bool) {
	return bundleInstall(root, "explicit")
}

// OverrideInstall maps an --app-server / ZCODE_ACP_APP_SERVER value into the
// discovery Install shape (D-04, precedence unchanged): a directory keeps the
// existing explicit-root behavior (validated when the invariant tree is
// present, the same unvalidated shape otherwise); a path whose literal
// basename or EvalSymlinks target ends in .AppImage is a packed install that
// goes through the managed mount; any other regular file keeps the existing
// app-server-binary shape. An empty path reports false so the override row
// stays an honest "no override configured".
//
// The stored AppImage is always absolute: exec.CommandContext would otherwise
// PATH-resolve a bare basename (a relative literal is absolutized with
// filepath.Abs, never a PATH lookup), and a dangling relative override is
// still stored absolute so the mount step fails loud naming the real path
// instead of silently falling through to discovery.
func OverrideInstall(path string) (Install, bool) {
	if path == "" {
		return Install{}, false
	}

	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		if inst, ok := ExplicitRootInstall(path); ok {
			return inst, true
		}

		return Install{
			Root:   path,
			Binary: filepath.Join(path, platformBinaryName()),
			Layout: "explicit",
		}, true
	}

	literal := path
	if abs, absErr := filepath.Abs(path); absErr == nil {
		literal = abs
	}

	resolved := literal
	if target, evalErr := filepath.EvalSymlinks(literal); evalErr == nil {
		resolved = target
	}

	if strings.HasSuffix(literal, appImageSuffix) || strings.HasSuffix(resolved, appImageSuffix) {
		return Install{
			AppImage: resolved,
			Version:  appImageVersion(resolved),
			Layout:   layoutAppImage,
		}, true
	}

	root := filepath.Dir(path)

	return Install{
		Root:       root,
		Binary:     path,
		Entrypoint: filepath.Join(root, filepath.FromSlash(entrypointRel)),
		Layout:     "explicit",
	}, true
}

// probeOverride is order step 1: the explicit override wins over any on-disk
// discovery (install-discovery.md :93).
func (p Probers) probeOverride() (Probe, Install, bool) {
	row := Probe{Name: "override", Query: "wrapper config: explicit root or ZCODE_BUILTIN_PROVIDER_CONFIG_FILE"}
	if p.Override == nil {
		row.Result = "no override configured"

		return row, Install{}, false
	}
	inst, ok := p.Override()
	if !ok {
		row.Result = "no override configured"

		return row, Install{}, false
	}
	if decoy := volatileInstallPath(inst); decoy != "" {
		row.Result = "rejected volatile mount " + decoy

		return row, Install{}, false
	}
	if inst.RequiresMount() {
		row.Result = "found: packed AppImage " + inst.AppImage + describeVersion(inst.Version)

		return row, inst, true
	}
	row.Result = "found: explicit root " + inst.Root

	return row, inst, true
}

// probeBuiltinRuntime is order step 5's data row: the newest zcode-builtin.json
// under the runtime-provider tree feeds ZCODE_BUILTIN_PROVIDER_CONFIG_FILE
// (the spawn line, install-discovery.md :50). It never yields a root install.
// On a fresh machine the tree does not exist yet and this probe honestly
// misses — the winner's BuiltinFile would stay empty and the child would boot
// on the unpaired personal-only env (the QA-02 field failure, EXP-VM-QA02);
// probeBuiltinBundled is the install-side completion of the same step.
func (p Probers) probeBuiltinRuntime() (row Probe, builtinFile string) {
	row = Probe{Name: "builtin-runtime", Query: "newest " + builtinFileName + " under the runtime-provider tree"}

	rel, supported := builtinProviderRelForPlatform()
	if !supported {
		row.Result = "not probed on this platform; the bundled copy fills the builtin env"

		return row, ""
	}

	row.Query = "newest " + builtinFileName + " under ~/" + rel + "/<ver>/endpoint-*/"
	provider := filepath.Join(p.Home, rel)
	versions, err := versionDirs(provider)
	if err != nil {
		row.Result = "no runtime provider tree under " + provider

		return row, ""
	}

	slices.SortFunc(versions, func(a, b string) int { return compareVersions(b, a) })
	for _, ver := range versions {
		file, ok := newestEndpointBuiltin(filepath.Join(provider, ver))
		if !ok {
			continue
		}
		row.Result = "found: " + file + describeRevision(file)

		return row, file
	}
	row.Result = "provider tree present but no " + builtinFileName + " in any version dir"

	return row, ""
}

// probeBuiltinBundled is the install-side completion of order step 5: when the
// resolved install carries no BuiltinFile from the runtime-provider tree — a
// fresh machine has no ~/.zcode/v2/runtime/provider cache — discovery falls
// back to the BUNDLED copy inside the install,
// <root>/resources/config/provider/zcode-builtin.json. Field-proven on the
// clean QA-02 VM (EXP-VM-QA02): the squashfs-root layout ships the bundled
// file, and the paired builtin+personal env pass boots the app-server with
// account models publishing. It never yields a root install and never
// displaces a runtime-cache hit (the runtime cache wins — install-discovery.md
// :91's active-cache rule); with neither source present BuiltinFile honestly
// stays empty (:34).
func (p Probers) probeBuiltinBundled(winner Install, found bool) (row Probe, bundledFile string) {
	row = Probe{
		Name:  "builtin-bundled",
		Query: "<root>/" + bundledProviderRel + "/" + builtinFileName,
	}
	if !found {
		row.Result = "no install resolved; no bundled builtin to check"

		return row, ""
	}
	if winner.BuiltinFile != "" {
		row.Result = "not consulted; the runtime cache resolved the builtin file"

		return row, ""
	}
	if winner.RequiresMount() {
		// A packed install is not readable before the mount: the bundled copy
		// is read inside the mounted tree by PrepareMount (Pitfall 15) and
		// fills BuiltinFile only when discovery left it empty (10-01).
		row.Result = "deferred until mount; <mount>/" + bundledProviderRel + "/" +
			builtinFileName + " is read after PrepareMount"

		return row, ""
	}
	bundled := filepath.Join(winner.Root, filepath.FromSlash(bundledProviderRel), builtinFileName)
	if _, err := os.Stat(bundled); err != nil {
		row.Result = "no bundled " + builtinFileName + " under " + winner.Root +
			" (" + bundledProviderRel + "/" + builtinFileName + " missing)"

		return row, ""
	}
	row.Result = "found: " + bundled + describeRevision(bundled)

	return row, bundled
}

// versionDirs lists the directory names under dir (the <ver> dirs of the
// runtime-provider tree).
func versionDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

// newestEndpointBuiltin finds zcode-builtin.json under the version dir's
// endpoint-* dirs (all version dirs share one endpoint dir name, :77).
func newestEndpointBuiltin(verDir string) (string, bool) {
	endpoints, err := versionDirs(verDir)
	if err != nil {
		return "", false
	}
	slices.Sort(endpoints)
	for _, name := range endpoints {
		if !strings.HasPrefix(name, "endpoint-") {
			continue
		}
		cand := filepath.Join(verDir, name, builtinFileName)
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand, true
		}
	}

	return "", false
}

// describeRevision reads the recorded revision number from a
// zcode-builtin.json for the diagnostics row (" (revision 44)"; per-version
// revisions are probe-time state, install-discovery.md :77, :87).
func describeRevision(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var doc struct {
		Revision int `json:"revision"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}

	return fmt.Sprintf(" (revision %d)", doc.Revision)
}

// appImageVersion extracts the <ver> of a packed AppImage filename in the
// documented shape ZCode-<ver>-linux-<arch>.AppImage (install-discovery.md
// :33); "" when the name does not match. The symlink form (ZCode.AppImage)
// carries no version — callers pass the EvalSymlinks-resolved target.
func appImageVersion(path string) string {
	name, ok := strings.CutSuffix(filepath.Base(path), appImageSuffix)
	if !ok {
		return ""
	}

	rest, ok := strings.CutPrefix(name, "ZCode-")
	if !ok {
		return ""
	}

	ver, arch, ok := strings.Cut(rest, "-linux-")
	if !ok || ver == "" || arch == "" {
		return ""
	}

	return ver
}

// describeVersion renders a parsed <ver> for a diagnostics row ("" when the
// filename carried none).
func describeVersion(version string) string {
	if version == "" {
		return ""
	}

	return " (version " + version + ")"
}

// compareVersions orders dotted version strings by numeric component
// (install-discovery.md :32's scheme: 3.18.2 > 3.17.1 > 0.0.0-dev — the dev
// marker parses to (0,0,0) via its leading digits; <build> suffixes compare
// as text only after equal numeric prefixes).
func compareVersions(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	for i := range max(len(as), len(bs)) {
		av, bv := versionComponent(as, i), versionComponent(bs, i)
		if av != bv {
			if av < bv {
				return -1
			}

			return 1
		}
	}

	return strings.Compare(a, b)
}

// versionComponent parses the leading digits of component i (0 when absent
// or non-numeric, e.g. the "-dev" suffix of 0.0.0-dev).
func versionComponent(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	digits := parts[i]
	for j, r := range digits {
		if r < '0' || r > '9' {
			digits = digits[:j]

			break
		}
	}
	n, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}

	return n
}

// defaultOverride is the order-step-1 seam. The wrapper's explicit
// root/builtin configuration surface is wired by the spawn half (plan 03-07);
// until then discovery always runs the on-disk matrix.
func defaultOverride() (Install, bool) {
	return Install{}, false
}
