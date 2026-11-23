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
	appImageSuffix     = ".AppImage"
	unpackedDirName    = "squashfs-root"
	binaryName         = "zcode"
	entrypointRel      = "resources/glm/zcode.cjs"
	builtinFileName    = "zcode-builtin.json"
	builtinProviderRel = ".zcode/v2/runtime/provider/linux-x86_64"

	// bundledProviderRel is the bundled builtin copy's directory under the
	// install root — <root>/resources/config/provider/zcode-builtin.json, the
	// copy every shipped layout carries (field-verified in the QA-02 VM's
	// squashfs-root, EXP-VM-QA02).
	bundledProviderRel = "resources/config/provider"

	// probeCount is the documented probe set (install-discovery.md :93-97
	// plus the runtime-provider-tree row and its install-side bundled
	// fallback row).
	probeCount = 9

	// dirPerm is the permission for directories discovery creates (the
	// AppImage extraction working dir).
	dirPerm = 0o750
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
	// ExtractAppImage unpacks a packed AppImage (--appimage-extract route)
	// and returns the extracted bundle root.
	ExtractAppImage func(ctx context.Context, appImagePath, workDir string) (string, error)
	// Home is the user's home dir (~ for the applications scan and the
	// runtime-provider tree).
	Home string
	// OptDir is the package-family root (the deb's /opt/ZCode).
	OptDir string
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
	add(p.probeXDGEntry(ctx))
	add(p.probeAppImageGlob(ctx, found))
	add(p.probeUnpackedGlob())
	add(p.probeDpkg(ctx))
	add(p.probeOptPath())
	add(p.probeRpm(ctx))
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
	if mountDecoy(inst.Root) {
		row.Result = "rejected volatile mount root " + inst.Root

		return row, Install{}, false
	}
	if inst.RequiresMount() {
		row.Result = "found: packed AppImage " + inst.AppImage + describeVersion(inst.Version)

		return row, inst, true
	}
	row.Result = "found: explicit root " + inst.Root

	return row, inst, true
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
// (install-discovery.md :65-69).
func (p Probers) probeXDGEntry(ctx context.Context) (Probe, Install, bool) {
	row := Probe{Name: "xdg-entry", Query: "xdg-mime query default x-scheme-handler/zcode"}
	if p.XDGQuery == nil {
		row.Result = "xdg-mime seam not configured"

		return row, Install{}, false
	}
	entryName, err := p.XDGQuery(ctx)
	if err != nil {
		row.Result = fmt.Sprintf("xdg-mime query failed: %v", err)

		return row, Install{}, false
	}
	entryName = strings.TrimSpace(entryName)
	if entryName == "" {
		row.Result = "no default handler for x-scheme-handler/zcode"

		return row, Install{}, false
	}
	entryPath, ok := findDesktopEntry(p.xdgEntryDirs(), entryName)
	if !ok {
		row.Result = fmt.Sprintf("default handler %q not found in the xdg entry search order", entryName)

		return row, Install{}, false
	}
	execBinary, ok := desktopEntryExec(entryPath)
	if !ok {
		row.Result = "entry " + entryPath + " has no Exec line"

		return row, Install{}, false
	}
	inst, ok, missReason := resolveExecTarget(execBinary)
	if !ok {
		row.Result = missReason

		return row, Install{}, false
	}
	row.Result = "found: default handler " + entryName + " -> " + execBinary

	return row, inst, true
}

// resolveExecTarget turns the entry's Exec binary into the install it
// identifies (deb vs AppImage dichotomy, install-discovery.md :68). ok=false
// carries the honest miss reason for the diagnostics row.
func resolveExecTarget(execBinary string) (inst Install, ok bool, missReason string) {
	resolved, err := filepath.EvalSymlinks(execBinary)
	if err != nil {
		return Install{}, false, fmt.Sprintf("Exec target %s unresolvable: %v", execBinary, err)
	}
	if mountDecoy(resolved) {
		return Install{}, false, "rejected volatile mount Exec target " + resolved
	}
	if strings.HasSuffix(resolved, appImageSuffix) {
		// The packed AppImage route belongs to the appimage-glob probe
		// (extraction), not to the entry resolution (:41).
		return Install{}, false, "default handler points at packed AppImage " +
			resolved + " (the appimage-glob probe resolves it)"
	}
	root := filepath.Dir(resolved)
	layout := layoutDeb
	if filepath.Base(root) == unpackedDirName {
		layout = layoutUnpackedAppImage
	}
	inst, ok = bundleInstall(root, layout)
	if !ok {
		return Install{}, false, "Exec target " + resolved + " does not carry the spawnable tree (" +
			binaryName + " + " + entrypointRel + ")"
	}

	return inst, true, ""
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

// probeAppImageGlob is order step 3a: packed ZCode*.AppImage file + symlink,
// resolved via EvalSymlinks; the user-facing symlink is preferred (matrix row
// 2). Extraction runs only when a packed AppImage is all that exists (:95).
func (p Probers) probeAppImageGlob(ctx context.Context, resolvedAlready bool) (Probe, Install, bool) {
	row := Probe{Name: "appimage-glob", Query: "~/Applications/ZCode*.AppImage" +
		" (file + symlink, resolved via EvalSymlinks)"}
	if p.AppDirs == nil {
		row.Result = "applications scan seam not configured"

		return row, Install{}, false
	}
	packed, unpacked, rejected := classifyAppDirs(p.AppDirs())
	notes := rejectionNotes(rejected)
	if len(packed) == 0 {
		row.Result = joinNotes(notes, "no ZCode*.AppImage under the applications scan")

		return row, Install{}, false
	}
	if len(unpacked) > 0 {
		row.Result = joinNotes(notes, "packed AppImage present but an unpacked "+unpackedDirName+
			" exists — extraction skipped (matrix step 3: extract only when a packed AppImage is all that exists)")

		return row, Install{}, false
	}
	if resolvedAlready {
		row.Result = joinNotes(notes, "packed AppImage present; extraction skipped — "+
			"an earlier probe resolved the install")

		return row, Install{}, false
	}
	sortAppImageCandidates(packed)
	resolved, err := filepath.EvalSymlinks(packed[0])
	if err != nil {
		row.Result = joinNotes(notes, fmt.Sprintf("AppImage %s unresolvable: %v", packed[0], err))

		return row, Install{}, false
	}
	if mountDecoy(resolved) {
		row.Result = joinNotes(notes, "rejected volatile mount AppImage "+resolved)

		return row, Install{}, false
	}
	if p.ExtractAppImage == nil {
		row.Result = joinNotes(notes, "extraction seam not configured for packed AppImage "+resolved)

		return row, Install{}, false
	}
	root, err := p.ExtractAppImage(ctx, resolved, "")
	if err != nil {
		row.Result = joinNotes(notes, fmt.Sprintf("extraction of %s failed: %v", resolved, err))

		return row, Install{}, false
	}
	inst, ok := bundleInstall(root, layoutAppImage)
	if !ok {
		row.Result = joinNotes(notes, "extracted tree at "+root+" is not spawnable ("+
			binaryName+" or "+entrypointRel+" missing)")

		return row, Install{}, false
	}
	row.Result = joinNotes(notes, "found: extracted "+resolved+" -> "+root)

	return row, inst, true
}

// classifyAppDirs sorts application-scan candidates into packed AppImage
// paths, unpacked squashfs-root dirs, and volatile-mount rejections.
func classifyAppDirs(candidates []string) (packed, unpacked, rejected []string) {
	for _, cand := range candidates {
		switch {
		case mountDecoy(cand):
			rejected = append(rejected, cand)
		case strings.HasSuffix(cand, appImageSuffix):
			packed = append(packed, cand)
		case filepath.Base(cand) == unpackedDirName:
			unpacked = append(unpacked, cand)
		}
	}

	return packed, unpacked, rejected
}

// probeUnpackedGlob is order step 3b: unpacked squashfs-root copies under the
// applications dirs, newest version first (matrix row 1).
func (p Probers) probeUnpackedGlob() (Probe, Install, bool) {
	row := Probe{Name: "unpacked-glob", Query: "~/Applications/zcode-*/" + unpackedDirName + "/" + binaryName}
	if p.AppDirs == nil {
		row.Result = "applications scan seam not configured"

		return row, Install{}, false
	}
	_, trees, rejected := classifyAppDirs(p.AppDirs())
	notes := rejectionNotes(rejected)
	slices.SortFunc(trees, func(a, b string) int { return compareVersions(dirVersion(b), dirVersion(a)) })
	for _, tree := range trees {
		inst, ok := bundleInstall(tree, layoutUnpackedAppImage)
		if ok {
			row.Result = joinNotes(notes, "found: "+tree)

			return row, inst, true
		}
	}
	row.Result = joinNotes(notes, "no "+unpackedDirName+" tree carrying "+
		binaryName+" + "+entrypointRel+" under the applications scan")

	return row, Install{}, false
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

// probeBuiltinRuntime is order step 5's data row: the newest zcode-builtin.json
// under the runtime-provider tree feeds ZCODE_BUILTIN_PROVIDER_CONFIG_FILE
// (the spawn line, install-discovery.md :50). It never yields a root install.
// On a fresh machine the tree does not exist yet and this probe honestly
// misses — the winner's BuiltinFile would stay empty and the child would boot
// on the unpaired personal-only env (the QA-02 field failure, EXP-VM-QA02);
// probeBuiltinBundled is the install-side completion of the same step.
func (p Probers) probeBuiltinRuntime() (row Probe, builtinFile string) {
	row = Probe{Name: "builtin-runtime", Query: "newest " + builtinFileName +
		" under ~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/endpoint-*/"}
	provider := filepath.Join(p.Home, builtinProviderRel)
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
		// is validated inside the mounted tree by PrepareMount (Pitfall 15).
		row.Result = "deferred until the packed AppImage is mounted"

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

// rejectionNotes renders mount-decoy rejections for a probe row.
func rejectionNotes(rejected []string) []string {
	notes := make([]string, 0, len(rejected))
	for _, mount := range rejected {
		notes = append(notes, "rejected volatile mount "+mount)
	}

	return notes
}

// joinNotes combines a row's rejection notes with its main result.
func joinNotes(notes []string, result string) string {
	if len(notes) == 0 {
		return result
	}

	return strings.Join(notes, "; ") + "; " + result
}

// sortAppImageCandidates orders packed candidates symlinks-first (the
// unversioned symlink is the user-facing entry, matrix row 2), names within
// each group.
func sortAppImageCandidates(paths []string) {
	symlinkRank := func(path string) int {
		info, err := os.Lstat(path)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return 0
		}

		return 1
	}
	slices.SortFunc(paths, func(a, b string) int {
		if ra, rb := symlinkRank(a), symlinkRank(b); ra != rb {
			return ra - rb
		}

		return strings.Compare(a, b)
	})
}

// dirVersion extracts the <ver> of a zcode-<ver>-named parent dir (the
// unpacked-copy glob shape); "" when the name does not match.
func dirVersion(path string) string {
	name := filepath.Base(filepath.Dir(path))
	ver, _ := strings.CutPrefix(name, "zcode-")

	return ver
}

// appImageVersion extracts the <ver> of a packed AppImage filename in the
// documented shape ZCode-<ver>-linux-<arch>.AppImage (install-discovery.md
// :33); "" when the name does not match. The symlink form (ZCode.AppImage)
// carries no version — callers pass the EvalSymlinks-resolved target.
func appImageVersion(path string) string {
	name, ok := strings.CutPrefix(filepath.Base(path), "ZCode-")
	if !ok {
		return ""
	}

	ver, _, ok := strings.Cut(name, "-linux-")
	if !ok {
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
		Override:        defaultOverride,
		XDGQuery:        defaultXDGQuery,
		AppDirs:         defaultAppDirs(home),
		DpkgQuery:       defaultDpkgQuery,
		RpmQuery:        defaultRpmQuery,
		ExtractAppImage: defaultExtractAppImage,
		Home:            home,
		OptDir:          optDir,
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
	if !strings.Contains(string(state), " install ok installed") {
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

// defaultExtractAppImage runs the documented extraction route: the resolved
// AppImage with --appimage-extract in a stable working dir next to the image
// (install-discovery.md :41 — extraction is the reproducible route). The
// production path is exercised only outside CI; tests stub the seam.
func defaultExtractAppImage(ctx context.Context, appImagePath, workDir string) (string, error) {
	if workDir == "" {
		workDir = filepath.Dir(appImagePath)
	}
	if err := os.MkdirAll(workDir, dirPerm); err != nil {
		return "", fmt.Errorf("prepare extraction dir %s: %w", workDir, err)
	}
	cmd := exec.CommandContext(ctx, appImagePath, "--appimage-extract")
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s --appimage-extract: %w", appImagePath, err)
	}

	return filepath.Join(workDir, unpackedDirName), nil
}
