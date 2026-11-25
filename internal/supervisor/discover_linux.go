//go:build linux

package supervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Linux install-layout facts (docs/protocol/install-discovery.md).
const (
	layoutUnpackedAppImage = "unpacked-appimage"
	layoutDeb              = "deb"
	layoutRpm              = "rpm"

	unpackedDirName    = "squashfs-root"
	binaryName         = "zcode"
	builtinProviderRel = ".zcode/v2/runtime/provider/linux-x86_64"

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

// DefaultProbers returns the production seam set: every OS contact shells out
// per the matrix commands (xdg-mime, dpkg, rpm); directory scans derive from
// home and optDir. Tests build their own Probers with stubbed seams instead.
func DefaultProbers(home, optDir string) Probers {
	return Probers{
		Override:  defaultOverride,
		XDGQuery:  defaultXDGQuery,
		AppDirs:   defaultAppDirs(home),
		DpkgQuery: defaultDpkgQuery,
		RpmQuery:  defaultRpmQuery,
		Home:      home,
		OptDir:    optDir,
	}
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

// probeInstallStage runs the Linux install-layout probes in the documented
// order (XDG entry, ranked AppImage stage, dpkg, /opt, rpm) and returns the
// first winning install plus every diagnostic row.
func (p Probers) probeInstallStage(ctx context.Context, found bool) ([]Probe, Install, bool) {
	var rows []Probe
	var winner Install
	ok := false

	add := func(row Probe, inst Install, probOK bool) {
		rows = append(rows, row)
		if probOK && !ok {
			winner, ok = inst, true
		}
	}

	xdgRow, xdgInstall, xdgSeeds, xdgOK := p.probeXDGEntry(ctx)
	add(xdgRow, xdgInstall, xdgOK)

	stageRows, stageInstall, stageOK := p.probeAppImageStage(xdgSeeds, found || xdgOK)
	rows = append(rows, stageRows...)
	if stageOK && !ok {
		winner, ok = stageInstall, true
	}

	add(p.probeDpkg(ctx))
	add(p.probeOptPath())
	add(p.probeRpm(ctx))

	return rows, winner, ok
}

// builtinProviderRelForPlatform names the Linux runtime-provider cache dir.
func builtinProviderRelForPlatform() (string, bool) {
	return builtinProviderRel, true
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

// DefaultOptDir is the Linux package-family discovery root (the deb layout).
func DefaultOptDir() string { return "/opt/ZCode" }
