// Windows install-model rules: the native directory sources, the
// uninstall-registry records, and the layout-invariant validation that admits
// a candidate root. The stage is pure path/string logic over injectable seams,
// which is what lets the Linux gate execute the Windows rules; the OS contact
// (environment, registry) lives in the windows-tagged file.

package supervisor

import (
	"fmt"
	"strings"
)

// windowsBinaryName is the Windows Electron binary; the native stage validates
// <root>\ZCode.exe beside the layout-invariant entrypoint.
const windowsBinaryName = "ZCode.exe"

// Windows install-stage layout labels.
const (
	layoutWindowsUser     = "windows-user"
	layoutWindowsMachine  = "windows-machine"
	layoutWindowsRegistry = "windows-registry"
)

// Windows stage diagnostics-row names and query shapes.
const (
	rowWindowsInstall       = "windows-install"
	rowWindowsRegistry      = "windows-registry"
	rowWindowsRegistryEntry = "windows-registry-entry"

	queryWindowsInstall = `%LOCALAPPDATA%\Programs\ZCode + %ProgramFiles%\ZCode + ` +
		`%ProgramFiles(x86)%\ZCode + the uninstall registry`
	queryWindowsRegistry = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall + ` +
		`HKLM 64-bit and 32-bit views`
)

// windowsBundleInstall validates the layout-invariant Windows spawn surface at
// root: <root>\ZCode.exe plus <root>\resources\glm\zcode.cjs. Passing the
// Windows binary name explicitly lets the rules run on a non-Windows host.
func windowsBundleInstall(root, layout string) (Install, bool) {
	return bundleInstallBinary(root, layout, windowsBinaryName)
}

// windowsInstallStage is the Windows install-layout stage: the per-user tree,
// the per-machine Program Files trees, then the uninstall-registry records in
// that order. found is the earlier-probe state: an earlier winner keeps the
// whole stage out of the way with one not-consulted row, reading neither the
// disk nor the registry. Every attempted directory source still appends its
// own diagnostics row; the first validated tree wins.
func windowsInstallStage(seams WindowsSeams, found bool) ([]Probe, Install, bool) {
	if found {
		return []Probe{{
			Name:   rowWindowsInstall,
			Query:  queryWindowsInstall,
			Result: "not consulted; an earlier probe resolved the install",
		}}, Install{}, false
	}

	rows, dirInstall, dirOK := windowsDirStage(seams.Dirs)
	registryRows, registryInstall, registryOK := windowsRegistryStage(seams.Uninstall)
	rows = append(rows, registryRows...)

	if registryOK && !dirOK {
		return rows, registryInstall, true
	}

	return rows, dirInstall, dirOK
}

// windowsDirStage evaluates the environment-derived directory sources in
// order. An empty root means the host does not define the source; a root
// without the spawn surface is a miss. Evaluation continues after a hit so
// every source has a row, but the first validated tree wins.
func windowsDirStage(dirs []WindowsInstallDir) ([]Probe, Install, bool) {
	if len(dirs) == 0 {
		return []Probe{{
			Name:   rowWindowsInstall,
			Query:  queryWindowsInstall,
			Result: "no native Windows directory sources are configured",
		}}, Install{}, false
	}

	rows := make([]Probe, 0, len(dirs))

	var winner Install

	ok := false

	for _, dir := range dirs {
		row := Probe{Name: dir.Name, Query: dir.Query}

		if dir.Root == "" {
			row.Result = "not defined on this host"
			rows = append(rows, row)

			continue
		}

		inst, valid := windowsBundleInstall(dir.Root, dir.Layout)
		if !valid {
			row.Result = "no spawnable tree at " + dir.Root + " (" +
				windowsBinaryName + " or " + entrypointRel + " missing)"
			rows = append(rows, row)

			continue
		}

		row.Result = "found: " + dir.Root
		rows = append(rows, row)

		if !ok {
			winner, ok = inst, true
		}
	}

	return rows, winner, ok
}

// windowsRegistryStage evaluates the uninstall-registry records: one summary
// row (scanned/matched counts, plus a partial-error note) always leads, then
// one row per ZCode-named record in seam order. Records whose DisplayName does
// not name ZCode produce no row — the uninstall hive holds hundreds of
// unrelated products. The first validated matching record wins.
func windowsRegistryStage(query func() ([]UninstallEntry, error)) ([]Probe, Install, bool) {
	summary := Probe{Name: rowWindowsRegistry, Query: queryWindowsRegistry}

	if query == nil {
		summary.Result = "registry seam not configured"

		return []Probe{summary}, Install{}, false
	}

	entries, err := query()
	if err != nil && len(entries) == 0 {
		summary.Result = "registry enumeration failed: " + err.Error()

		return []Probe{summary}, Install{}, false
	}

	rows := make([]Probe, 1, len(entries)+1)
	rows[0] = summary

	matched := 0

	var winner Install

	ok := false

	for _, entry := range entries {
		if !windowsDisplayNameMatches(entry.DisplayName) {
			continue
		}

		matched++

		row, inst, valid := windowsRegistryCandidate(entry)
		rows = append(rows, row)

		if valid && !ok {
			winner, ok = inst, true
		}
	}

	summary.Result = fmt.Sprintf("scanned %d uninstall records, %d matching ZCode", len(entries), matched)
	if err != nil {
		summary.Result += "; partial enumeration error: " + err.Error()
	}

	rows[0] = summary

	return rows, winner, ok
}

// windowsRegistryCandidate evaluates one ZCode-named record: the candidate
// root must carry the invariant tree before adoption, and the DisplayVersion
// becomes the install version.
func windowsRegistryCandidate(entry UninstallEntry) (Probe, Install, bool) {
	row := Probe{Name: rowWindowsRegistryEntry, Query: entry.Key}

	root := windowsInstallRoot(entry)
	if root == "" {
		row.Result = "miss: the record names no install location"

		return row, Install{}, false
	}

	inst, ok := windowsBundleInstall(root, layoutWindowsRegistry)
	if !ok {
		row.Result = "miss: " + root + " lacks the spawnable tree (" +
			windowsBinaryName + " + " + entrypointRel + ")"

		return row, Install{}, false
	}

	inst.Version = strings.TrimSpace(entry.DisplayVersion)
	row.Result = "found: " + root + describeVersion(inst.Version)

	return row, inst, true
}

// windowsInstallRoot resolves a record's candidate root: InstallLocation wins
// (trimmed of whitespace, surrounding quotes, and trailing separators), then
// the DisplayIcon's directory, then "".
func windowsInstallRoot(entry UninstallEntry) string {
	if location := trimWindowsPath(entry.InstallLocation); location != "" {
		return location
	}

	return windowsPathDir(windowsDisplayIconPath(entry.DisplayIcon))
}

// windowsDisplayIconPath normalizes a DisplayIcon value to the executable
// path: a quoted value is everything inside the quotes (the icon index after
// the closing quote is discarded), and an unquoted value loses a trailing
// comma-plus-digits icon index.
func windowsDisplayIconPath(value string) string {
	path := strings.TrimSpace(value)

	if rest, quoted := strings.CutPrefix(path, `"`); quoted {
		if inside, _, found := strings.Cut(rest, `"`); found {
			return strings.TrimSpace(inside)
		}

		path = rest
	}

	if idx := strings.LastIndex(path, ","); idx >= 0 && isDecimal(path[idx+1:]) {
		path = path[:idx]
	}

	return strings.TrimSpace(strings.Trim(path, `"`))
}

// trimWindowsPath strips whitespace, surrounding double quotes, and trailing
// path separators (both Windows and POSIX forms) from a registry path value.
func trimWindowsPath(value string) string {
	path := strings.Trim(strings.TrimSpace(value), `"`)
	path = strings.TrimSpace(strings.TrimRight(path, `\/`))

	return strings.Trim(strings.TrimSpace(path), `"`)
}

// windowsPathDir returns the directory component of a Windows path literal,
// splitting on both separators so the rule is testable on any host; "" when
// the value carries no separator.
func windowsPathDir(path string) string {
	idx := strings.LastIndexAny(path, `\/`)
	if idx < 0 {
		return ""
	}

	if idx == 0 {
		return path[:1]
	}

	return path[:idx]
}

// isDecimal reports whether value is a non-empty run of ASCII digits (the icon
// index of a DisplayIcon value).
func isDecimal(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// windowsDisplayNameMatches reports whether a DisplayName names ZCode: the
// trimmed, lowercased value is exactly "zcode" or starts with "zcode ".
// Compounds such as "ZCodeWriter" and "Other ZCode" never match — the name
// rule is only a prefilter that keeps the diagnostics actionable; adoption
// still requires the invariant tree.
func windowsDisplayNameMatches(name string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(name))

	return trimmed == "zcode" || strings.HasPrefix(trimmed, "zcode ")
}
