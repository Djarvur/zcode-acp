//nolint:testpackage // the Windows stage rules are unexported and tested in place on every platform
package supervisor

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Windows install-model contract tests (quick item 261126-2s9). The install
// stage is platform-neutral and seam-fed, so the Windows discovery rules run
// in the Linux gate; the windows-tagged suite in discovery_windows_test.go
// proves the production wiring and the real registry read.

// The stable probe-row names the Windows stage appends; diagnostics stay
// readable because every directory source carries its own name.
const (
	testWindowsLocalAppDataRow  = "windows-localappdata"
	testWindowsProgramFilesRow  = "windows-programfiles"
	testWindowsInstallRow       = "windows-install"
	testWindowsRegistryRow      = "windows-registry"
	testWindowsRegistryEntryRow = "windows-registry-entry"
)

// Canned registry-seam failures for the honest-error arms.
var (
	errTestHiveUnreadable = errors.New("hive unreadable")
	errTestHKCULocked     = errors.New("hkcu locked")
)

// Fixture spellings: the version, the ZCode display name, and the Windows
// path shapes the record rules must normalize.
const (
	testWindowsVersion     = "3.18.2"
	testWindowsDisplayName = "ZCode"
	testWindowsInstallPath = `C:\ZCode`
	testWindowsBinaryPath  = `C:\ZCode\ZCode.exe`
)

// writeWindowsBundleTree lays down the layout-invariant Windows spawn surface
// at root: ZCode.exe plus resources/glm/zcode.cjs.
func writeWindowsBundleTree(t *testing.T, root string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Join(root, "resources", "glm"), 0o750); err != nil {
		t.Fatalf("mkdir windows bundle tree: %v", err)
	}
	for _, rel := range []string{windowsBinaryName, filepath.FromSlash(entrypointRel)} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("#!zcode-bundle-marker\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// emptyWindowsUninstall is the "no records on this host" registry seam.
func emptyWindowsUninstall() ([]UninstallEntry, error) {
	return []UninstallEntry{}, nil
}

// windowsRowNames returns every probe-row name in order.
func windowsRowNames(probes []Probe) []string {
	names := make([]string, 0, len(probes))
	for _, row := range probes {
		names = append(names, row.Name)
	}

	return names
}

// windowsRows returns every probe row carrying name.
func windowsRows(probes []Probe, name string) []Probe {
	rows := make([]Probe, 0, 1)
	for _, row := range probes {
		if row.Name == name {
			rows = append(rows, row)
		}
	}

	return rows
}

// windowsRow finds the first probe row for name.
func windowsRow(probes []Probe, name string) (Probe, bool) {
	rows := windowsRows(probes, name)
	if len(rows) == 0 {
		return Probe{}, false
	}

	return rows[0], true
}

// windowsMachineDir builds one machine-scope source fixture.
func windowsMachineDir(root string) WindowsInstallDir {
	return WindowsInstallDir{
		Name:   testWindowsProgramFilesRow,
		Query:  `%ProgramFiles%\ZCode`,
		Root:   root,
		Layout: layoutWindowsMachine,
	}
}

// windowsUserDir builds one user-scope source fixture.
func windowsUserDir(root string) WindowsInstallDir {
	return WindowsInstallDir{
		Name:   testWindowsLocalAppDataRow,
		Query:  `%LOCALAPPDATA%\Programs\ZCode`,
		Root:   root,
		Layout: layoutWindowsUser,
	}
}

// TestWindowsInstallStageLocalAppData pins the first documented source: a
// valid per-user tree wins with layout windows-user, and every later source
// still appends its own diagnostics row.
func TestWindowsInstallStageLocalAppData(t *testing.T) {
	userRoot := t.TempDir()
	machineRoot := t.TempDir()
	writeWindowsBundleTree(t, userRoot)
	writeWindowsBundleTree(t, machineRoot)

	probes, install, ok := windowsInstallStage(WindowsSeams{
		Dirs:      []WindowsInstallDir{windowsUserDir(userRoot), windowsMachineDir(machineRoot)},
		Uninstall: emptyWindowsUninstall,
	}, false)
	if !ok {
		t.Fatalf("a valid local-appdata tree must win: %+v", probes)
	}

	want := Install{
		Root:       userRoot,
		Binary:     filepath.Join(userRoot, windowsBinaryName),
		Entrypoint: filepath.Join(userRoot, filepath.FromSlash(entrypointRel)),
		Layout:     layoutWindowsUser,
	}
	if install != want {
		t.Fatalf("install mismatch:\n got %+v\nwant %+v", install, want)
	}

	wantNames := []string{testWindowsLocalAppDataRow, testWindowsProgramFilesRow, testWindowsRegistryRow}
	if names := windowsRowNames(probes); len(names) != len(wantNames) {
		t.Fatalf("rows = %v, want %v", names, wantNames)
	} else {
		for i, name := range wantNames {
			if names[i] != name {
				t.Fatalf("row %d = %q, want %q (documented source order)", i, names[i], name)
			}
		}
	}

	for _, name := range []string{testWindowsLocalAppDataRow, testWindowsProgramFilesRow} {
		row, found := windowsRow(probes, name)
		if !found || !strings.Contains(row.Result, "found: ") {
			t.Errorf("%s row must report the validated tree: %+v", name, row)
		}
	}
}

// TestWindowsInstallStageProgramFilesFallback pins the machine source: an
// absent or invalid first source is a miss row, and a valid second source wins
// with layout windows-machine.
func TestWindowsInstallStageProgramFilesFallback(t *testing.T) {
	machineRoot := t.TempDir()
	writeWindowsBundleTree(t, machineRoot)

	probes, install, ok := windowsInstallStage(WindowsSeams{
		Dirs:      []WindowsInstallDir{windowsUserDir(t.TempDir()), windowsMachineDir(machineRoot)},
		Uninstall: emptyWindowsUninstall,
	}, false)
	if !ok {
		t.Fatalf("the valid program-files tree must win: %+v", probes)
	}
	if install.Layout != layoutWindowsMachine || install.Root != machineRoot {
		t.Fatalf("winner = %+v, want the machine tree %q", install, machineRoot)
	}

	miss, found := windowsRow(probes, testWindowsLocalAppDataRow)
	if !found || strings.Contains(miss.Result, "found: ") {
		t.Fatalf("the invalid user source must be a miss row: %+v", miss)
	}
	if !strings.Contains(miss.Result, "spawnable tree") {
		t.Errorf("the miss row must name the missing spawn surface: %+v", miss)
	}

	hit, found := windowsRow(probes, testWindowsProgramFilesRow)
	if !found || !strings.Contains(hit.Result, "found: "+machineRoot) {
		t.Fatalf("the machine row must record its hit: %+v", hit)
	}
}

// TestWindowsInstallStageEmptySources pins the empty-source arms: an undefined
// environment variable records "not defined on this host" and never wins, and
// an empty source list records the honest no-sources row.
func TestWindowsInstallStageEmptySources(t *testing.T) {
	t.Run("root not defined on this host", func(t *testing.T) {
		probes, install, ok := windowsInstallStage(WindowsSeams{
			Dirs:      []WindowsInstallDir{windowsUserDir("")},
			Uninstall: emptyWindowsUninstall,
		}, false)
		if ok || install != (Install{}) {
			t.Fatalf("an empty Root must never win: install %+v, ok %v", install, ok)
		}

		row, found := windowsRow(probes, testWindowsLocalAppDataRow)
		if !found || !strings.Contains(row.Result, "not defined on this host") {
			t.Fatalf("the empty-root row must name the missing source: %+v", row)
		}
	})

	t.Run("no directory sources configured", func(t *testing.T) {
		probes, install, ok := windowsInstallStage(WindowsSeams{Uninstall: emptyWindowsUninstall}, false)
		if ok || install != (Install{}) {
			t.Fatalf("no sources must never win: install %+v, ok %v", install, ok)
		}

		row, found := windowsRow(probes, testWindowsInstallRow)
		if !found || !strings.Contains(row.Result, "no native Windows directory sources") {
			t.Fatalf("the empty stage row must say no sources are configured: %+v", row)
		}
	})
}

// TestWindowsInstallStageInvalidTree pins the validation surface: a root that
// exists but lacks ZCode.exe or resources/glm/zcode.cjs is a miss, never an
// adopted backend.
func TestWindowsInstallStageInvalidTree(t *testing.T) {
	t.Run("missing entrypoint", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, windowsBinaryName), []byte("fixture"), 0o600); err != nil {
			t.Fatalf("write binary: %v", err)
		}
		assertWindowsInvalidTree(t, root)
	})

	t.Run("missing binary", func(t *testing.T) {
		root := t.TempDir()
		entrypoint := filepath.Join(root, filepath.FromSlash(entrypointRel))
		if err := os.MkdirAll(filepath.Dir(entrypoint), 0o750); err != nil {
			t.Fatalf("mkdir entrypoint dir: %v", err)
		}
		if err := os.WriteFile(entrypoint, []byte("fixture"), 0o600); err != nil {
			t.Fatalf("write entrypoint: %v", err)
		}
		assertWindowsInvalidTree(t, root)
	})
}

// assertWindowsInvalidTree fails unless root is a miss naming the spawn
// surface and yields no install.
func assertWindowsInvalidTree(t *testing.T, root string) {
	t.Helper()

	probes, install, ok := windowsInstallStage(WindowsSeams{
		Dirs:      []WindowsInstallDir{windowsUserDir(root)},
		Uninstall: emptyWindowsUninstall,
	}, false)
	if ok || install != (Install{}) {
		t.Fatalf("an invalid tree must never be adopted: install %+v, ok %v", install, ok)
	}

	row, found := windowsRow(probes, testWindowsLocalAppDataRow)
	if !found || !strings.Contains(row.Result, "spawnable tree") {
		t.Fatalf("the invalid-tree row must name the missing invariant: %+v", row)
	}
}

// TestWindowsRegistryStageHit pins the registry hit: a ZCode-named record
// whose InstallLocation carries the tree wins with layout windows-registry and
// the DisplayVersion, and the summary row reports scanned and matched counts.
func TestWindowsRegistryStageHit(t *testing.T) {
	root := t.TempDir()
	writeWindowsBundleTree(t, root)

	const key = `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\ZCode`
	entry := UninstallEntry{
		Key:             key,
		DisplayName:     testWindowsDisplayName + " " + testWindowsVersion,
		DisplayVersion:  testWindowsVersion,
		InstallLocation: root,
	}

	probes, install, ok := windowsRegistryStage(func() ([]UninstallEntry, error) {
		return []UninstallEntry{entry}, nil
	})
	if !ok {
		t.Fatalf("a valid registry record must win: %+v", probes)
	}

	want := Install{
		Root:       root,
		Binary:     filepath.Join(root, windowsBinaryName),
		Entrypoint: filepath.Join(root, filepath.FromSlash(entrypointRel)),
		Layout:     layoutWindowsRegistry,
		Version:    testWindowsVersion,
	}
	if install != want {
		t.Fatalf("install mismatch:\n got %+v\nwant %+v", install, want)
	}

	if names := windowsRowNames(probes); len(names) != 2 ||
		names[0] != testWindowsRegistryRow || names[1] != testWindowsRegistryEntryRow {
		t.Fatalf("rows = %v, want the summary then the entry row", names)
	}

	summary, _ := windowsRow(probes, testWindowsRegistryRow)
	if !strings.Contains(summary.Result, "scanned 1 uninstall records, 1 matching ZCode") {
		t.Errorf("summary row must report the counts: %+v", summary)
	}

	entryRow, _ := windowsRow(probes, testWindowsRegistryEntryRow)
	if entryRow.Query != key {
		t.Errorf("entry row Query = %q, want the full key path %q", entryRow.Query, key)
	}
	if !strings.Contains(entryRow.Result, "found: "+root) || !strings.Contains(entryRow.Result, "(version 3.18.2)") {
		t.Errorf("entry row must name the root and version: %+v", entryRow)
	}
}

// TestWindowsRegistryStageMisses pins the registry misses: a matching record
// with no install location and one whose root lacks the tree are miss rows,
// and a non-ZCode record appends no entry row at all.
func TestWindowsRegistryStageMisses(t *testing.T) {
	entries := []UninstallEntry{
		{Key: `HKCU\...\ZCodeWriter`, DisplayName: "ZCodeWriter", InstallLocation: t.TempDir()},
		{Key: `HKLM\...\ZCode`, DisplayName: testWindowsDisplayName, DisplayVersion: testWindowsVersion},
		{Key: `HKLM\...\ZCode 3.17.1`, DisplayName: "ZCode 3.17.1", InstallLocation: t.TempDir()},
	}

	probes, install, ok := windowsRegistryStage(func() ([]UninstallEntry, error) {
		return entries, nil
	})
	if ok || install != (Install{}) {
		t.Fatalf("no usable registry record must win: install %+v, ok %v", install, ok)
	}

	rows := windowsRows(probes, testWindowsRegistryEntryRow)
	if len(rows) != 2 {
		t.Fatalf("one entry row per matching record (the non-ZCode record stays silent): %+v", rows)
	}
	for _, row := range rows {
		if !strings.Contains(row.Result, "miss:") {
			t.Errorf("matching but unusable records must be miss rows: %+v", row)
		}
		if strings.Contains(row.Query, "ZCodeWriter") {
			t.Errorf("a non-ZCode record must produce no entry row: %+v", row)
		}
	}

	summary, _ := windowsRow(probes, testWindowsRegistryRow)
	if !strings.Contains(summary.Result, "scanned 3 uninstall records, 2 matching ZCode") {
		t.Errorf("summary row must report the counts: %+v", summary)
	}

	noLocation, _ := windowsRow(probes, testWindowsRegistryEntryRow)
	if !strings.Contains(noLocation.Result, "no install location") {
		t.Errorf("the location-less record must name the miss reason: %+v", noLocation)
	}
}

// TestWindowsRegistryStageDisplayIcon pins the DisplayIcon fallback: with an
// empty InstallLocation, a quoted DisplayIcon ending in an icon index resolves
// the candidate root to the executable's directory.
func TestWindowsRegistryStageDisplayIcon(t *testing.T) {
	root := t.TempDir()
	writeWindowsBundleTree(t, root)

	tests := []struct {
		name string
		icon string
	}{
		{"quoted with icon index", `"` + filepath.Join(root, windowsBinaryName) + `",0`},
		{"unquoted with icon index", filepath.Join(root, windowsBinaryName) + `,0`},
		{"plain executable path", filepath.Join(root, windowsBinaryName)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			probes, install, ok := windowsRegistryStage(func() ([]UninstallEntry, error) {
				return []UninstallEntry{{
					Key:         `HKCU\...\ZCode`,
					DisplayName: testWindowsDisplayName,
					DisplayIcon: test.icon,
				}}, nil
			})
			if !ok {
				t.Fatalf("a DisplayIcon record must resolve its directory: %+v", probes)
			}
			if install.Layout != layoutWindowsRegistry || install.Root != root {
				t.Fatalf("winner = %+v, want the icon's directory %q", install, root)
			}
		})
	}
}

// TestWindowsRegistryStageSeamStates pins the honest seam arms: a nil seam, an
// error with no entries, and a partial error whose entries still evaluate.
func TestWindowsRegistryStageSeamStates(t *testing.T) {
	t.Run("nil seam", func(t *testing.T) {
		probes, install, ok := windowsRegistryStage(nil)
		if ok || install != (Install{}) {
			t.Fatalf("a nil seam must never win: install %+v, ok %v", install, ok)
		}
		if names := windowsRowNames(probes); len(names) != 1 || names[0] != testWindowsRegistryRow {
			t.Fatalf("rows = %v, want one registry summary row", names)
		}

		if !strings.Contains(probes[0].Result, "not configured") {
			t.Errorf("the nil-seam row must say the seam is unconfigured: %+v", probes[0])
		}
	})

	t.Run("error without entries", func(t *testing.T) {
		probes, install, ok := windowsRegistryStage(func() ([]UninstallEntry, error) {
			return nil, errTestHiveUnreadable
		})
		if ok || install != (Install{}) {
			t.Fatalf("a failed enumeration must never win: install %+v, ok %v", install, ok)
		}
		if len(probes) != 1 || !strings.Contains(probes[0].Result, "hive unreadable") {
			t.Fatalf("the error row must name the failure: %+v", probes)
		}
	})

	t.Run("partial error still evaluates entries", func(t *testing.T) {
		root := t.TempDir()
		writeWindowsBundleTree(t, root)

		probes, install, ok := windowsRegistryStage(func() ([]UninstallEntry, error) {
			return []UninstallEntry{{
				Key:             `HKLM\...\ZCode`,
				DisplayName:     testWindowsDisplayName,
				DisplayVersion:  testWindowsVersion,
				InstallLocation: root,
			}}, errTestHKCULocked
		})
		if !ok || install.Root != root {
			t.Fatalf("entries must still evaluate after a partial error: %+v", install)
		}

		summary, _ := windowsRow(probes, testWindowsRegistryRow)
		if !strings.Contains(summary.Result, "hkcu locked") {
			t.Errorf("the summary row must name the partial error: %+v", summary)
		}
	})
}

// TestWindowsInstallStageOverrideNotConsulted pins unchanged precedence: when
// an earlier probe already won, the Windows stage records exactly one
// not-consulted row and reads neither the disk nor the registry.
func TestWindowsInstallStageOverrideNotConsulted(t *testing.T) {
	seams := WindowsSeams{
		Dirs: []WindowsInstallDir{windowsUserDir(filepath.Join(t.TempDir(), "absent"))},
		Uninstall: func() ([]UninstallEntry, error) {
			t.Fatal("the registry seam must not be read once an earlier probe won")

			return []UninstallEntry{}, nil
		},
	}

	probes, install, ok := windowsInstallStage(seams, true)
	if ok || install != (Install{}) {
		t.Fatalf("the not-consulted arm must yield no install: install %+v, ok %v", install, ok)
	}
	if names := windowsRowNames(probes); len(names) != 1 || names[0] != testWindowsInstallRow {
		t.Fatalf("rows = %v, want exactly one %s row", names, testWindowsInstallRow)
	}
	if !strings.Contains(probes[0].Result, "not consulted") {
		t.Errorf("the row must record why it did not run: %+v", probes[0])
	}
}

// TestWindowsInstallStageEmptyMachine pins the honest empty result: no
// directory hit and no registry hit yields no install.
func TestWindowsInstallStageEmptyMachine(t *testing.T) {
	probes, install, ok := windowsInstallStage(WindowsSeams{
		Dirs: []WindowsInstallDir{
			windowsUserDir(""),
			windowsMachineDir(filepath.Join(t.TempDir(), "absent")),
		},
		Uninstall: emptyWindowsUninstall,
	}, false)
	if ok || install != (Install{}) {
		t.Fatalf("an empty machine must yield no install: install %+v, ok %v", install, ok)
	}
	if len(probes) != 3 {
		t.Fatalf("one row per attempted source plus the registry summary: %+v", probes)
	}
}

// TestWindowsDisplayNameMatches pins the prefilter: "ZCode" and versioned
// names match case-insensitively; compounds and empty names do not.
func TestWindowsDisplayNameMatches(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{testWindowsDisplayName, true},
		{"ZCode 3.18.2", true},
		{"zcode", true},
		{"  ZCode 3.18.2 ", true},
		{"ZCodeWriter", false},
		{"Other ZCode", false},
		{"", false},
	}

	for _, test := range tests {
		if got := windowsDisplayNameMatches(test.name); got != test.want {
			t.Errorf("windowsDisplayNameMatches(%q) = %v, want %v", test.name, got, test.want)
		}
	}
}

// TestWindowsInstallRoot pins the record-to-root rule: InstallLocation wins,
// surrounding quotes/index suffixes/trailing separators are stripped, and an
// empty record yields "".
func TestWindowsInstallRoot(t *testing.T) {
	tests := []struct {
		name  string
		entry UninstallEntry
		want  string
	}{
		{
			name:  "install location wins over the icon",
			entry: UninstallEntry{InstallLocation: testWindowsInstallPath, DisplayIcon: `C:\Other\ZCode.exe`},
			want:  testWindowsInstallPath,
		},
		{
			name:  "quoted install location",
			entry: UninstallEntry{InstallLocation: `"C:\Program Files\ZCode"`},
			want:  `C:\Program Files\ZCode`,
		},
		{
			name:  "trailing separators stripped",
			entry: UninstallEntry{InstallLocation: testWindowsInstallPath + `\`},
			want:  testWindowsInstallPath,
		},
		{
			name:  "display icon quoted with icon index",
			entry: UninstallEntry{DisplayIcon: `"` + testWindowsBinaryPath + `",0`},
			want:  testWindowsInstallPath,
		},
		{
			name:  "display icon unquoted with icon index",
			entry: UninstallEntry{DisplayIcon: testWindowsBinaryPath + `,0`},
			want:  testWindowsInstallPath,
		},
		{
			name:  "empty record",
			entry: UninstallEntry{},
			want:  "",
		},
	}

	for _, test := range tests {
		if got := windowsInstallRoot(test.entry); got != test.want {
			t.Errorf("%s: windowsInstallRoot(%+v) = %q, want %q", test.name, test.entry, got, test.want)
		}
	}
}

// TestWindowsDisplayIconPath pins the icon-string normalization on its own.
func TestWindowsDisplayIconPath(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"quoted with index", `"` + testWindowsBinaryPath + `",0`, testWindowsBinaryPath},
		{"quoted without index", `"` + testWindowsBinaryPath + `"`, testWindowsBinaryPath},
		{"unquoted with index", testWindowsBinaryPath + `,0`, testWindowsBinaryPath},
		{"comma in a directory name", `C:\ZCode, Inc\ZCode.exe`, `C:\ZCode, Inc\ZCode.exe`},
		{"empty", "", ""},
	}

	for _, test := range tests {
		if got := windowsDisplayIconPath(test.value); got != test.want {
			t.Errorf("%s: windowsDisplayIconPath(%q) = %q, want %q", test.name, test.value, got, test.want)
		}
	}
}
