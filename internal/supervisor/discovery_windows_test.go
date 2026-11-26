//go:build windows

//nolint:testpackage // the windows-tagged wiring proves unexported seams in place
package supervisor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Windows production-wiring tests (quick item 261126-2s9). The platform-neutral
// rules execute in the Linux gate (discovery_windows_model_test.go); these
// tests prove the windows-tagged wiring, the Discover integration, the
// override precedence, and the real read-only registry enumeration. They
// compile locally via GOOS=windows vet / go test -c and execute on a Windows
// host (the CI windows-latest leg or scripts/windows-vm).

// writeWindowsBundledBuiltin plants the bundled zcode-builtin.json inside the
// Windows install root and returns its path.
func writeWindowsBundledBuiltin(t *testing.T, root string) string {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(bundledProviderRel), builtinFileName)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir bundled provider dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"revision":44}`), 0o600); err != nil {
		t.Fatalf("write bundled zcode-builtin.json: %v", err)
	}

	return path
}

// TestWindowsDefaultProbers pins the production seam wiring: DefaultProbers
// fills the environment-derived directory sources in order and a non-nil
// uninstall seam beside the override.
func TestWindowsDefaultProbers(t *testing.T) {
	probers := DefaultProbers(t.TempDir(), "")

	if probers.Override == nil {
		t.Fatal("DefaultProbers must wire the override seam")
	}
	if probers.Windows.Uninstall == nil {
		t.Fatal("DefaultProbers must wire the uninstall-registry seam")
	}
	if len(probers.Windows.Dirs) < 2 {
		t.Fatalf("want at least the user and machine directory sources, got %+v", probers.Windows.Dirs)
	}

	first := probers.Windows.Dirs[0]
	if first.Name != testWindowsLocalAppDataRow || first.Layout != layoutWindowsUser {
		t.Fatalf("first source = %+v, want %s/%s", first, testWindowsLocalAppDataRow, layoutWindowsUser)
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		want := filepath.Join(localAppData, "Programs", "ZCode")
		if first.Root != want {
			t.Fatalf("local-appdata root = %q, want %q", first.Root, want)
		}
	}

	second := probers.Windows.Dirs[1]
	if second.Name != testWindowsProgramFilesRow || second.Layout != layoutWindowsMachine {
		t.Fatalf("second source = %+v, want %s/%s", second, testWindowsProgramFilesRow, layoutWindowsMachine)
	}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		want := filepath.Join(programFiles, "ZCode")
		if second.Root != want {
			t.Fatalf("program-files root = %q, want %q", second.Root, want)
		}
	}
}

// TestWindowsDiscoverNativeInstall pins the Discover integration: a native
// winner carries the layout, the tree, the ZCode.exe spawn surface, and the
// bundled builtin file (the fresh-machine pairing contract).
func TestWindowsDiscoverNativeInstall(t *testing.T) {
	root := t.TempDir()
	writeWindowsBundleTree(t, root)
	builtin := writeWindowsBundledBuiltin(t, root)

	probers := Probers{
		Override: func() (Install, bool) { return Install{}, false },
		Home:     t.TempDir(),
		Windows: WindowsSeams{
			Dirs:      []WindowsInstallDir{windowsUserDir(root)},
			Uninstall: emptyWindowsUninstall,
		},
	}

	install, probes, err := probers.Discover(context.Background())
	if err != nil {
		t.Fatalf("a native Windows tree must resolve: %v", err)
	}

	if install.Layout != layoutWindowsUser || install.Root != root {
		t.Fatalf("winner = %+v, want the local-appdata tree %q", install, root)
	}
	if install.Binary != filepath.Join(root, windowsBinaryName) {
		t.Fatalf("binary = %q, want %q", install.Binary, filepath.Join(root, windowsBinaryName))
	}
	if install.Entrypoint != filepath.Join(root, filepath.FromSlash(entrypointRel)) {
		t.Fatalf("entrypoint = %q, want the invariant entrypoint under %q", install.Entrypoint, root)
	}
	if install.BuiltinFile != builtin {
		t.Fatalf("builtin file = %q, want the bundled copy %q", install.BuiltinFile, builtin)
	}

	sourceRow, found := windowsRow(probes, testWindowsLocalAppDataRow)
	if !found || !strings.Contains(sourceRow.Result, "found: "+root) {
		t.Fatalf("the source row must record the hit: %+v", sourceRow)
	}

	bundledRow, found := windowsRow(probes, "builtin-bundled")
	if !found || !strings.Contains(bundledRow.Result, "found: "+builtin) {
		t.Fatalf("the bundled builtin row must record the hit: %+v", bundledRow)
	}

	runtimeRow, found := windowsRow(probes, "builtin-runtime")
	if !found || !strings.Contains(runtimeRow.Result, "not probed on this platform") {
		t.Fatalf("the runtime-provider probe must report the platform: %+v", runtimeRow)
	}
}

// TestWindowsDiscoverOverridePrecedence pins unchanged precedence: an explicit
// override wins over a valid native tree and the Windows stage records one
// not-consulted row.
func TestWindowsDiscoverOverridePrecedence(t *testing.T) {
	overrideRoot := t.TempDir()
	nativeRoot := t.TempDir()
	writeWindowsBundleTree(t, overrideRoot)
	writeWindowsBundleTree(t, nativeRoot)

	probers := Probers{
		Override: func() (Install, bool) { return ExplicitRootInstall(overrideRoot) },
		Home:     t.TempDir(),
		Windows: WindowsSeams{
			Dirs:      []WindowsInstallDir{windowsUserDir(nativeRoot)},
			Uninstall: emptyWindowsUninstall,
		},
	}

	install, probes, err := probers.Discover(context.Background())
	if err != nil {
		t.Fatalf("the override fixture must resolve: %v", err)
	}
	if install.Layout != "explicit" || install.Root != overrideRoot {
		t.Fatalf("winner = %+v, want the explicit override root %q", install, overrideRoot)
	}

	rows := windowsRows(probes, testWindowsInstallRow)
	if len(rows) != 1 {
		t.Fatalf("want exactly one %s row, got %+v", testWindowsInstallRow, rows)
	}
	if !strings.Contains(rows[0].Result, "earlier probe") {
		t.Fatalf("the not-consulted row must say why it did not run: %+v", rows[0])
	}

	if nativeRow, found := windowsRow(probes, testWindowsLocalAppDataRow); found {
		t.Fatalf("the native stage must not read the disk when the override won: %+v", nativeRow)
	}
}

// TestWindowsRealRegistrySanity walks the real read-only enumeration on a
// Windows host: every entry carries a Key and at least one usable value, and
// the call returns promptly. A locked or absent hive is a legitimate host
// state, so the partial error is logged, never failed.
func TestWindowsRealRegistrySanity(t *testing.T) {
	start := time.Now()

	entries, err := DefaultProbers(t.TempDir(), "").Windows.Uninstall()

	elapsed := time.Since(start)
	if err != nil {
		t.Logf("real uninstall-registry enumeration reported a partial error: %v", err)
	}
	t.Logf("enumerated %d uninstall records in %s", len(entries), elapsed)

	for _, entry := range entries {
		if entry.Key == "" {
			t.Errorf("uninstall record with an empty Key: %+v", entry)
		}
		if entry.DisplayName == "" && entry.InstallLocation == "" && entry.DisplayIcon == "" {
			t.Errorf("uninstall record %q carries no usable value", entry.Key)
		}
	}

	if elapsed > time.Minute {
		t.Errorf("the uninstall-registry enumeration took %s; discovery must stay bounded", elapsed)
	}
}
