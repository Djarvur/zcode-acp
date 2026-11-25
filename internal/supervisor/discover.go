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
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Layout values of Install (RESEARCH Pattern 5; install-discovery.md rows).
const (
	layoutUnpackedAppImage = "unpacked-appimage"
	layoutAppImage         = "appimage"
	layoutDeb              = "deb"
	layoutRpm              = "rpm"
)

// Matrix path facts (docs/protocol/install-discovery.md).
const (
	appImageSuffix      = ".AppImage"
	unpackedDirName     = "squashfs-root"
	binaryName          = "zcode"
	entrypointRel       = "resources/glm/zcode.cjs"
	builtinFileName     = "zcode-builtin.json"
	builtinProviderRoot = ".zcode/v2/runtime/provider"

	// bundledProviderRel is the bundled builtin copy's directory under the
	// install root — <root>/resources/config/provider/zcode-builtin.json, the
	// copy every shipped layout carries (field-verified in the QA-02 VM's
	// squashfs-root, EXP-VM-QA02).
	bundledProviderRel = "resources/config/provider"

	// probeCount is the documented fixed probe set (install-discovery.md
	// :93-97) with the two extraction-era AppImage probes replaced by the
	// single ranked AppImage stage, plus the macOS .app probe, the
	// runtime-provider-tree row, and its install-side bundled fallback row.
	// The variable appimage-candidate and mac-app-candidate rows add one row
	// per evaluated candidate.
	probeCount = 9

	// appExecMask is the executable-bit mask a packed candidate must carry
	// before discovery adopts it (D-12's file-level usability checks).
	appExecMask = 0o111
)

// AppImage candidate ranks: the D-08 tie-break at equal versions, in
// preference order — a spawnable unpacked tree needs no mount, then the
// user-facing symlink, then a versioned packed regular file.
const (
	rankUnpackedTree = iota
	rankPackedSymlink
	rankPackedFile
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
	Layout      string // "unpacked-appimage" | "appimage" | "deb" | "rpm" | "explicit" | "mac-app"
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
	// MacApps returns the macOS ZCode*.app candidates from
	// /Applications and $HOME/Applications.
	MacApps func() []string
	// DpkgQuery returns the deb file list for zcode (empty = rc remnant).
	DpkgQuery func(ctx context.Context) (string, error)
	// RpmQuery returns the rpm package identity for zcode (where applicable).
	RpmQuery func(ctx context.Context) (string, error)
	// Home is the user's home dir (~ for the applications scan and the
	// runtime-provider tree).
	Home string
	// OptDir is the package-family root (the deb's /opt/ZCode).
	OptDir string
	// RuntimeProviderPlatform overrides the runtime-provider cache platform
	// directory. Empty selects the current GOOS/GOARCH mapping.
	RuntimeProviderPlatform string
}

// Discover resolves the ZCode desktop install in the documented probe order
// (install-discovery.md §Recommended discovery order), with the macOS .app
// stage before the Linux probes: override, macOS app bundle, desktop entry,
// AppImage paths, package paths, runtime provider tree. The first probe that
// yields a spawnable install wins and probes never merge; every probe appends
// a Probe row whether it hits or misses, and an empty result errors with
// ErrNoInstall carrying every row as diagnostics. Volatile /tmp/.mount_*
// decoys are recorded-and-rejected, never resolved (row 5).
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

	macRows, macInstall, macOK := p.probeMacApp(ctx, found)
	probes = append(probes, macRows...)
	if macOK && !found {
		winner, found = macInstall, true
	}

	xdgRow, xdgInstall, xdgSeeds, xdgOK := p.probeXDGEntry(ctx)
	add(xdgRow, xdgInstall, xdgOK)

	// Order step 3 is one version-ranked stage: an AppImage-family XDG target
	// joins the candidate set (D-13), the stage appends the rank row and one
	// candidate row per evaluated candidate, then the winner.
	stageRows, stageInstall, stageOK := p.probeAppImageStage(xdgSeeds, found)
	probes = append(probes, stageRows...)
	if stageOK && !found {
		winner, found = stageInstall, true
	}

	add(p.probeDpkg(ctx))
	add(p.probeOptPath())
	add(p.probeRpm(ctx))
	row, builtin := p.probeBuiltinRuntime(winner, found)
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
	if root == "" || mountDecoy(root) {
		return Install{}, false
	}
	inst := Install{
		Root:       root,
		Binary:     filepath.Join(root, binaryName),
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
// discovery Install shape (D-04, precedence unchanged): a macOS .app bundle,
// its Contents directory, or its Contents/MacOS executable resolves through
// macAppInstall; a directory keeps the existing explicit-root behavior
// (validated when the invariant tree is present, the same unvalidated shape
// otherwise); a path whose literal basename or EvalSymlinks target ends in
// .AppImage is a packed install that goes through the managed mount; any other
// regular file keeps the existing app-server-binary shape. An empty path
// reports false so the override row stays an honest "no override configured".
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

	if appPath, ok := macAppOverridePath(path); ok {
		inst, reason := macAppInstall(context.Background(), appPath)
		if reason != "" {
			return Install{}, false
		}

		return inst, true
	}

	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		if inst, ok := ExplicitRootInstall(path); ok {
			return inst, true
		}

		return Install{
			Root:   path,
			Binary: filepath.Join(path, binaryName),
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
	row := Probe{
		Name:  "override",
		Query: "wrapper config: explicit root, macOS .app/Contents/executable, or ZCODE_BUILTIN_PROVIDER_CONFIG_FILE",
	}
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
	if inst.Layout == layoutMacApp {
		row.Result = "found: mac-app " + inst.Root + describeVersion(inst.Version)

		return row, inst, true
	}
	row.Result = "found: explicit root " + inst.Root

	return row, inst, true
}

// probeMacApp is the macOS order step before every Linux probe: evaluate
// /Applications and $HOME/Applications candidates, keep one diagnostics row
// per candidate, and select the newest valid bundle. An earlier override
// wins and the stage records that it was not consulted.
func (p Probers) probeMacApp(ctx context.Context, found bool) ([]Probe, Install, bool) {
	row := Probe{Name: "mac-app", Query: "/Applications/ZCode*.app + $HOME/Applications/ZCode*.app"}
	if found {
		row.Result = "not consulted; an earlier probe resolved the install"

		return []Probe{row}, Install{}, false
	}
	if p.MacApps == nil {
		row.Result = "macOS app seam not configured (or not a macOS host)"

		return []Probe{row}, Install{}, false
	}

	apps := dedupeStrings(p.MacApps())
	rows := make([]Probe, 0, len(apps)+1)
	rows = append(rows, row)

	var winner Install
	for _, app := range apps {
		inst, reason := macAppInstall(ctx, app)
		if reason != "" {
			rows = append(rows, Probe{Name: "mac-app-candidate", Query: app,
				Result: "miss: " + app + ": " + reason})

			continue
		}

		rows = append(rows, Probe{Name: "mac-app-candidate", Query: app,
			Result: "hit: mac-app version " + inst.Version + " " + inst.Root})
		if winner.Root == "" || compareVersions(inst.Version, winner.Version) > 0 {
			winner = inst
		}
	}

	if winner.Root == "" {
		if len(apps) == 0 {
			rows[0].Result = "no ZCode*.app in the standard application dirs"
		} else {
			rows[0].Result = "no valid ZCode*.app; see mac-app-candidate rows"
		}

		return rows, Install{}, false
	}

	rows[0].Result = "selected: mac-app version " + winner.Version + " " + winner.Root

	return rows, winner, true
}

// dedupeStrings preserves first-seen order.
func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))

	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}

// xdgEntryDirs lists the desktop-entry search order: user-level entries win
// over the system dirs (install-discovery.md :63, :68).
func (p Probers) xdgEntryDirs() []string {
	dirs := []string{"/usr/local/share/applications", "/usr/share/applications"}
	if p.Home != "" {
		user := filepath.Join(p.Home, ".local", "share", "applications")
		dirs = append([]string{user}, dirs...)
	}

	return dirs
}

// probeXDGEntry is order step 2: the desktop entry's Exec target instantly
// identifies the owning layout (deb /opt/ZCode/zcode vs an AppImage path)
// (install-discovery.md :65-69). A package-tree target wins here unchanged
// (D-10); an AppImage-family target is returned as a seed for the version
// ranking instead, so a stale entry can never pin an older backend (D-13).
func (p Probers) probeXDGEntry(ctx context.Context) (Probe, Install, []string, bool) {
	row := Probe{Name: "xdg-entry", Query: "xdg-mime query default x-scheme-handler/zcode"}
	if p.XDGQuery == nil {
		row.Result = "xdg-mime seam not configured"

		return row, Install{}, nil, false
	}
	entryName, err := p.XDGQuery(ctx)
	if err != nil {
		row.Result = fmt.Sprintf("xdg-mime query failed: %v", err)

		return row, Install{}, nil, false
	}
	entryName = strings.TrimSpace(entryName)
	if entryName == "" {
		row.Result = "no default handler for x-scheme-handler/zcode"

		return row, Install{}, nil, false
	}
	entryPath, ok := findDesktopEntry(p.xdgEntryDirs(), entryName)
	if !ok {
		row.Result = fmt.Sprintf("default handler %q not found in the xdg entry search order", entryName)

		return row, Install{}, nil, false
	}
	execBinary, ok := desktopEntryExec(entryPath)
	if !ok {
		row.Result = "entry " + entryPath + " has no Exec line"

		return row, Install{}, nil, false
	}
	inst, seeds, ok, missReason := resolveExecTarget(execBinary)
	if !ok {
		row.Result = missReason

		return row, Install{}, seeds, false
	}
	row.Result = "found: default handler " + entryName + " -> " + execBinary

	return row, inst, nil, true
}

// resolveExecTarget turns the entry's Exec binary into the install it
// identifies (deb vs AppImage dichotomy, install-discovery.md :68). A
// package tree returns the step-2 install; a packed AppImage or an unpacked
// squashfs-root tree returns a seed for the version-ranked stage; ok=false
// carries the honest miss reason for the diagnostics row.
func resolveExecTarget(execBinary string) (inst Install, seeds []string, ok bool, missReason string) {
	resolved, err := filepath.EvalSymlinks(execBinary)
	if err != nil {
		return Install{}, nil, false, fmt.Sprintf("Exec target %s unresolvable: %v", execBinary, err)
	}
	if mountDecoy(resolved) {
		return Install{}, nil, false, "rejected volatile mount Exec target " + resolved
	}
	if strings.HasSuffix(resolved, appImageSuffix) {
		return Install{}, []string{resolved}, false, "default handler points at packed AppImage " +
			resolved + " (fed to the AppImage ranking)"
	}
	root := filepath.Dir(resolved)
	if filepath.Base(root) == unpackedDirName {
		return Install{}, []string{root}, false, "default handler points at an unpacked " +
			unpackedDirName + " tree " + root + " (fed to the AppImage ranking)"
	}
	inst, ok = bundleInstall(root, layoutDeb)
	if !ok {
		return Install{}, nil, false, "Exec target " + resolved + " does not carry the spawnable tree (" +
			binaryName + " + " + entrypointRel + ")"
	}

	return inst, nil, true, ""
}

// findDesktopEntry resolves an entry name through the xdg search order.
func findDesktopEntry(dirs []string, entryName string) (string, bool) {
	for _, dir := range dirs {
		path := filepath.Join(dir, entryName)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, true
		}
	}

	return "", false
}

// desktopEntryExec extracts the Exec line's binary field from a freedesktop
// entry file (first unquoted field, or the quoted path when present).
func desktopEntryExec(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		value, ok := strings.CutPrefix(line, "Exec=")
		if !ok {
			continue
		}
		if before, rest, had := strings.Cut(value, `"`); had && before == "" {
			binary, _, _ := strings.Cut(rest, `"`)

			return binary, binary != ""
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			return "", false
		}

		return fields[0], true
	}

	return "", false
}

// appCandidate is one AppImage-layout backend evaluated by the ranked stage:
// the path as scanned (the symlink display is kept for diagnostics), the
// EvalSymlinks-resolved path, its layout and parsed version, the D-08 rank,
// the Install a hit yields, and the miss reason ("" for a hit).
type appCandidate struct {
	display  string
	resolved string
	layout   string
	version  string
	rank     int
	install  Install
	miss     string
}

// probeAppImageStage is order step 3 (D-06..D-09): collect packed AppImage
// files, unpacked squashfs-root trees, and the XDG AppImage-family seeds;
// evaluate every candidate individually (a bad one is a miss, never an
// abort); rank by version, then the D-08 tie-break; and return the newest
// usable candidate as at most one Install. found is the earlier-probe state:
// the documented first-match order keeps an earlier winner, so the stage
// records that it was not consulted and evaluates nothing. Discovery is a
// pure read — no mount and no directory creation happens here (D-12).
func (p Probers) probeAppImageStage(seeds []string, found bool) ([]Probe, Install, bool) {
	row := Probe{Name: "appimage-rank", Query: "~/Applications/ZCode*.AppImage + ~/Applications/zcode-*/" +
		unpackedDirName + " + the desktop entry target, version-ranked"}
	if found {
		row.Result = "not consulted; an earlier probe resolved the install"

		return []Probe{row}, Install{}, false
	}

	candidates := dedupeAppCandidates(p.collectAppCandidates(seeds))
	sortAppCandidates(candidates)

	winner, ok := appImageWinner(candidates)
	if ok {
		row.Result = winner.selection()
	} else {
		row.Result = "no AppImage-layout candidate"
	}

	rows := make([]Probe, 0, len(candidates)+1)
	rows = append(rows, row)

	for i := range candidates {
		cand := candidates[i]
		rows = append(rows, Probe{Name: "appimage-candidate", Query: cand.display, Result: cand.result()})
	}

	if ok {
		return rows, winner.install, true
	}

	return rows, Install{}, false
}

// collectAppCandidates gathers the AppDirs scan plus the XDG seeds and
// evaluates each path independently.
func (p Probers) collectAppCandidates(seeds []string) []appCandidate {
	paths := make([]string, 0, len(seeds))
	if p.AppDirs != nil {
		paths = append(paths, p.AppDirs()...)
	}
	paths = append(paths, seeds...)

	candidates := make([]appCandidate, 0, len(paths))
	for _, path := range paths {
		candidates = append(candidates, evaluateAppCandidate(path))
	}

	return candidates
}

// evaluateAppCandidate classifies one scanned path and evaluates it in
// isolation: an unknown shape, a decoy, an unresolvable link, a non-regular
// or non-executable file, a missing tree, and an unparseable version are all
// miss reasons (D-07/D-12), never an abort.
func evaluateAppCandidate(path string) appCandidate {
	switch {
	case mountDecoy(path):
		return appCandidate{display: path, layout: layoutAppImage, miss: "rejected volatile mount " + path}
	case strings.HasSuffix(path, appImageSuffix):
		return evaluatePackedCandidate(path)
	case filepath.Base(path) == unpackedDirName:
		return evaluateUnpackedCandidate(path)
	default:
		return appCandidate{display: path, miss: "not an AppImage or " + unpackedDirName + " candidate"}
	}
}

// evaluatePackedCandidate applies D-12's file-level checks to one packed
// AppImage: resolvable, regular, executable, non-volatile, and with a
// parseable <ver>.
func evaluatePackedCandidate(display string) appCandidate {
	cand := appCandidate{display: display, layout: layoutAppImage, rank: rankPackedFile}

	info, err := os.Lstat(display)
	if err != nil {
		cand.miss = "unresolvable path: " + err.Error()

		return cand
	}

	if info.Mode()&os.ModeSymlink != 0 {
		cand.rank = rankPackedSymlink
	}

	resolved, err := filepath.EvalSymlinks(display)
	if err != nil {
		cand.miss = "unresolvable symlink: " + err.Error()

		return cand
	}

	cand.resolved = resolved

	if mountDecoy(resolved) {
		cand.miss = "rejected volatile mount " + resolved

		return cand
	}

	stat, err := os.Stat(resolved)
	if err != nil {
		cand.miss = "cannot stat the resolved path: " + err.Error()

		return cand
	}

	if !stat.Mode().IsRegular() {
		cand.miss = "not a regular file"

		return cand
	}

	if stat.Mode().Perm()&appExecMask == 0 {
		cand.miss = "not executable; remediation: chmod +x " + resolved

		return cand
	}

	version := appImageVersion(resolved)
	if version == "" {
		cand.miss = "version unknown: the resolved name is not ZCode-<ver>-linux-<arch>" + appImageSuffix

		return cand
	}

	cand.version = version
	cand.install = Install{AppImage: resolved, Version: version, Layout: layoutAppImage}

	return cand
}

// evaluateUnpackedCandidate resolves one squashfs-root tree and applies the
// same file-level bar: the documented zcode-<ver> parent and the invariant
// bundle tree.
func evaluateUnpackedCandidate(display string) appCandidate {
	cand := appCandidate{display: display, layout: layoutUnpackedAppImage, rank: rankUnpackedTree}

	resolved, err := filepath.EvalSymlinks(display)
	if err != nil {
		cand.miss = "unresolvable tree: " + err.Error()

		return cand
	}

	cand.resolved = resolved

	if mountDecoy(resolved) {
		cand.miss = "rejected volatile mount " + resolved

		return cand
	}

	version := dirVersion(resolved)
	if version == "" {
		cand.miss = "version unknown: the parent dir is not zcode-<ver>"

		return cand
	}

	inst, ok := bundleInstall(resolved, layoutUnpackedAppImage)
	if !ok {
		cand.miss = "missing the spawnable tree (" + binaryName + " + " + entrypointRel + ")"

		return cand
	}

	inst.Version = version
	cand.version = version
	cand.install = inst

	return cand
}

// dedupeAppCandidates keeps one candidate per resolved path: a symlink and
// its target are one backend (A14), a hit beats a miss, the D-08 symlink rank
// beats a regular file, and the lexicographically smaller display breaks the
// remaining tie. Unresolved misses are keyed by display so distinct broken
// paths stay distinct rows.
func dedupeAppCandidates(candidates []appCandidate) []appCandidate {
	index := make(map[string]int, len(candidates))
	deduped := make([]appCandidate, 0, len(candidates))

	for i := range candidates {
		cand := candidates[i]
		key := cand.resolved
		if key == "" {
			key = "display:" + cand.display
		}

		existing, seen := index[key]
		if !seen {
			index[key] = len(deduped)
			deduped = append(deduped, cand)

			continue
		}

		if preferAppCandidate(cand, deduped[existing]) {
			deduped[existing] = cand
		}
	}

	return deduped
}

// preferAppCandidate reports whether cand represents the same backend better
// than the incumbent.
func preferAppCandidate(cand, incumbent appCandidate) bool {
	if (cand.miss == "") != (incumbent.miss == "") {
		return cand.miss == ""
	}

	if cand.rank != incumbent.rank {
		return cand.rank < incumbent.rank
	}

	return cand.display < incumbent.display
}

// sortAppCandidates orders candidates newest version first (an empty version
// sorts last), then by the D-08 rank, then by display path — the
// deterministic selection DISC-04 requires.
func sortAppCandidates(candidates []appCandidate) {
	slices.SortFunc(candidates, func(a, b appCandidate) int {
		if order := compareVersions(a.version, b.version); order != 0 {
			return -order
		}

		if a.rank != b.rank {
			return a.rank - b.rank
		}

		return strings.Compare(a.display, b.display)
	})
}

// appImageWinner returns the first non-miss candidate — the newest usable
// backend after ranking (D-06/D-12).
func appImageWinner(candidates []appCandidate) (appCandidate, bool) {
	for i := range candidates {
		if candidates[i].miss == "" {
			return candidates[i], true
		}
	}

	return appCandidate{}, false
}

// selection renders the rank row for a winning candidate: layout, version,
// and resolved path, plus the symlink display when it differs (D-09).
func (c appCandidate) selection() string {
	path := "resolved " + c.resolved
	if c.display != c.resolved {
		path += " (display " + c.display + ")"
	}

	return "selected: " + c.layout + " version " + c.version + " " + path
}

// result renders one candidate's diagnostics row: a hit names the layout,
// version, and resolved path; a miss names the display, the known version,
// and the failing step (D-09).
func (c appCandidate) result() string {
	version := c.version
	if version == "" {
		version = "unknown"
	}

	if c.miss != "" {
		return "miss: " + c.display + " (version " + version + "): " + c.miss
	}

	path := "resolved " + c.resolved
	if c.display != c.resolved {
		path += " (display " + c.display + ")"
	}

	return "hit: " + c.layout + " version " + version + " " + path
}

// probeDpkg is order step 4a: the dpkg database (matrix row 3). An rc-state
// remnant answers with an EMPTY file list and falls through to opt-path.
func (p Probers) probeDpkg(ctx context.Context) (Probe, Install, bool) {
	row := Probe{Name: "dpkg", Query: "dpkg-query -W -f='${Version} ${Status}' zcode; dpkg -L zcode"}
	if p.DpkgQuery == nil {
		row.Result = "dpkg seam not configured"

		return row, Install{}, false
	}
	out, err := p.DpkgQuery(ctx)
	if err != nil {
		row.Result = fmt.Sprintf("dpkg query failed: %v", err)

		return row, Install{}, false
	}
	files := pathList(out)
	if len(files) == 0 {
		row.Result = "rc remnant or absent (empty file list) — falling through to opt-path"

		return row, Install{}, false
	}
	inst, ok := p.packageTreeInstall()
	if !ok {
		row.Result = fmt.Sprintf("package file list has %d entries but no spawnable tree at %s", len(files), p.OptDir)

		return row, Install{}, false
	}
	entrypoint := filepath.Join(p.OptDir, entrypointRel)
	if !slices.Contains(files, entrypoint) {
		row.Result = fmt.Sprintf("package file list has %d entries, none naming %s", len(files), entrypoint)

		return row, Install{}, false
	}
	row.Result = "found: package files present under " + p.OptDir

	return row, inst, true
}

// probeOptPath is order step 4b: the package-family root checked directly —
// the documented fallback when only an rc remnant remains in the dpkg
// database (:42).
func (p Probers) probeOptPath() (Probe, Install, bool) {
	row := Probe{Name: "opt-path", Query: "/opt/ZCode/" + entrypointRel + " + /opt/ZCode/" + binaryName}
	inst, ok := p.packageTreeInstall()
	if !ok {
		row.Result = "no spawnable tree at " + p.OptDir + " (" + binaryName + " or " + entrypointRel + " missing)"

		return row, Install{}, false
	}
	row.Result = "found: opt tree at " + p.OptDir

	return row, inst, true
}

// probeRpm is order step 4c: the rpm database where applicable (matrix row
// 4, DEB-GENERALIZED: the rpm layout mirrors the deb family).
func (p Probers) probeRpm(ctx context.Context) (Probe, Install, bool) {
	row := Probe{Name: "rpm", Query: "rpm -q zcode (where applicable)"}
	if p.RpmQuery == nil {
		row.Result = "rpm seam not configured (or not an rpm host)"

		return row, Install{}, false
	}
	out, err := p.RpmQuery(ctx)
	if err != nil {
		row.Result = fmt.Sprintf("rpm query failed: %v", err)

		return row, Install{}, false
	}
	out = strings.TrimSpace(out)
	if out == "" {
		row.Result = "rpm package absent"

		return row, Install{}, false
	}
	inst, ok := p.packageTreeInstall()
	if !ok {
		row.Result = "rpm " + out + " but no spawnable tree at " + p.OptDir

		return row, Install{}, false
	}
	row.Result = "found: rpm " + out + " serving " + p.OptDir

	return row, inst, true
}

// packageTreeInstall builds the deb-family Install from the OptDir root after
// guarding the volatile-mount decoy and validating the spawn surface.
func (p Probers) packageTreeInstall() (Install, bool) {
	return bundleInstall(p.OptDir, layoutDeb)
}

// probeBuiltinRuntime is order step 5's data row: the zcode-builtin.json under
// the host's runtime-provider cache feeds ZCODE_BUILTIN_PROVIDER_CONFIG_FILE
// (the spawn line, install-discovery.md :50). It never yields a root install.
// On a fresh machine the tree does not exist yet and this probe honestly
// misses — the winner's BuiltinFile would stay empty and the child would boot
// on the unpaired personal-only env (the QA-02 field failure, EXP-VM-QA02);
// probeBuiltinBundled is the install-side completion of the same step.
//
// The platform directory is the host GOOS/GOARCH mapping (darwin-aarch64 on
// this Mac), never the Linux literal. An exact version match with the selected
// install wins over a newer cache entry, so the builtin revision the bridge
// pushes is derived from the same provider path the backend was launched with.
func (p Probers) probeBuiltinRuntime(winner Install, found bool) (row Probe, builtinFile string) {
	platform := p.RuntimeProviderPlatform
	if platform == "" {
		platform = hostProviderPlatform()
	}
	row = Probe{Name: "builtin-runtime", Query: "newest " + builtinFileName +
		" under " + builtinProviderRoot + "/" + platform + "/<ver>/endpoint-*/"}
	if platform == "" {
		row.Result = "no runtime-provider platform mapping for this host"

		return row, ""
	}

	provider := filepath.Join(p.Home, filepath.FromSlash(builtinProviderRoot), platform)
	versions, err := versionDirs(provider)
	if err != nil {
		row.Result = "no runtime provider tree under " + provider

		return row, ""
	}

	slices.SortFunc(versions, func(a, b string) int { return compareVersions(b, a) })
	if found && winner.Version != "" {
		versions = preferVersion(versions, winner.Version)
	}
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

// preferVersion moves preferred to the front when present.
func preferVersion(versions []string, preferred string) []string {
	for i, ver := range versions {
		if ver != preferred {
			continue
		}
		ordered := make([]string, 0, len(versions))
		ordered = append(ordered, ver)
		ordered = append(ordered, versions[:i]...)
		ordered = append(ordered, versions[i+1:]...)

		return ordered
	}

	return versions
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
	query := "<root>/" + bundledProviderRel + "/" + builtinFileName
	if found && winner.Layout == layoutMacApp {
		query = "<app>/Contents/Resources/" + appProviderDir + "/" + builtinFileName
	}
	row = Probe{
		Name:  "builtin-bundled",
		Query: query,
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
	if winner.Layout == layoutMacApp {
		bundled = macBundledBuiltinPath(winner.Root)
	}
	if _, err := os.Stat(bundled); err != nil {
		row.Result = "no bundled " + builtinFileName + " under " + winner.Root +
			" (" + query + " missing)"

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

// dirVersion extracts the <ver> of a zcode-<ver>-named parent dir (the
// unpacked-copy glob shape); "" when the name does not match the documented
// shape. D-12 lists a parseable version among the file-level usability
// checks, so an arbitrary parent name is a version-unknown miss, never a
// bogus version string.
func dirVersion(path string) string {
	name := filepath.Base(filepath.Dir(path))
	ver, ok := strings.CutPrefix(name, "zcode-")
	if !ok || ver == "" {
		return ""
	}

	return ver
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

// pathList splits command output into its non-empty trimmed lines.
func pathList(out string) []string {
	lines := make([]string, 0, strings.Count(out, "\n")+1)
	for line := range strings.SplitSeq(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

// DefaultProbers returns the production seam set: every OS contact shells out
// per the matrix commands (xdg-mime, dpkg, rpm); directory scans derive from
// home and optDir. Tests build their own Probers with stubbed seams instead.
func DefaultProbers(home, optDir string) Probers {
	return Probers{
		Override:  defaultOverride,
		XDGQuery:  defaultXDGQuery,
		AppDirs:   defaultAppDirs(home),
		MacApps:   defaultMacApps(home),
		DpkgQuery: defaultDpkgQuery,
		RpmQuery:  defaultRpmQuery,
		Home:      home,
		OptDir:    optDir,
	}
}

// defaultOverride is the order-step-1 seam. The wrapper's explicit
// root/builtin configuration surface is wired by the spawn half (plan 03-07);
// until then discovery always runs the on-disk matrix.
func defaultOverride() (Install, bool) {
	return Install{}, false
}

// defaultXDGQuery shells out the matrix step-2 command.
func defaultXDGQuery(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "xdg-mime", "query", "default", "x-scheme-handler/zcode").Output()
	if err != nil {
		return "", fmt.Errorf("xdg-mime query default x-scheme-handler/zcode: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// defaultAppDirs scans the user's Applications dir for the documented glob
// shapes (matrix rows 1-2).
func defaultAppDirs(home string) func() []string {
	return func() []string {
		apps := filepath.Join(home, "Applications")
		packed, _ := filepath.Glob(filepath.Join(apps, "ZCode*"+appImageSuffix))
		unpacked, _ := filepath.Glob(filepath.Join(apps, "zcode-*/"+unpackedDirName))
		candidates := make([]string, 0, len(packed)+len(unpacked))
		candidates = append(candidates, packed...)
		candidates = append(candidates, unpacked...)

		return candidates
	}
}

// defaultDpkgQuery shells out the matrix row-3 commands: the install state
// line, then the package file list. An unknown or removed install (rc-state
// remnant) answers as an empty list — the probe falls through to opt-path.
func defaultDpkgQuery(ctx context.Context) (string, error) {
	state, err := exec.CommandContext(ctx, "dpkg-query", "-W", "-f=${Status}", "zcode").Output()
	if err != nil {
		return "", fmt.Errorf("dpkg-query -W zcode: %w", err)
	}

	// The real output for an installed package is exactly
	// "install ok installed" — no leading space — so a space-prefixed
	// substring can never match (WR-01). Compare the status fields instead:
	// the first word is the desired state and the last is the current one.
	fields := strings.Fields(string(state))
	if len(fields) < 3 || fields[0] != "install" || fields[2] != "installed" {
		// rc-state remnant (deinstall ok config-files) or otherwise removed —
		// the empty file list makes the probe fall through to opt-path (:30).
		return "", nil
	}

	files, err := exec.CommandContext(ctx, "dpkg", "-L", "zcode").Output()
	if err != nil {
		return "", fmt.Errorf("dpkg -L zcode: %w", err)
	}

	return string(files), nil
}

// defaultRpmQuery shells out the matrix row-4 command where applicable; an
// absent rpm or package is an honest miss, not an error.
func defaultRpmQuery(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "rpm", "-q", "zcode").Output()
	if err != nil {
		return "", fmt.Errorf("rpm -q zcode: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
