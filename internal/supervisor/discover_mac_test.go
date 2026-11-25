package supervisor_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

const (
	fixtureMacAppLayout  = "mac-app"
	fixtureMacAppVersion = "4.0.0"
)

// writeMacBundleFixture lays down a complete ZCode.app fixture and returns its
// app path.
func writeMacBundleFixture(t *testing.T, app string) string {
	t.Helper()

	binary := filepath.Join(app, "Contents", "MacOS", "ZCode")
	entrypoint := filepath.Join(app, "Contents", "Resources", "glm", "zcode.cjs")
	provider := filepath.Join(app, "Contents", "Resources", "config", "provider",
		"zcode-builtin.json")

	for _, path := range []string{binary, entrypoint, provider} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
	}
	for _, path := range []string{binary, entrypoint, provider} {
		if err := os.WriteFile(path, []byte("fixture\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	if err := os.Chmod(binary, 0o755); err != nil {
		t.Fatalf("chmod %s: %v", binary, err)
	}

	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>dev.zcode.app</string>
<key>CFBundleExecutable</key><string>ZCode</string>
<key>CFBundleShortVersionString</key><string>%s</string>
</dict></plist>
`, fixtureMacAppVersion)
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatalf("write Info.plist: %v", err)
	}

	return app
}

// discoverMacApps runs discovery with the macOS scan seam and no Linux hits.
func discoverMacApps(t *testing.T, apps []string) (supervisor.Install, []supervisor.Probe, error) {
	t.Helper()

	p := supervisor.Probers{
		Home:    t.TempDir(),
		OptDir:  filepath.Join(t.TempDir(), "ZCode"),
		MacApps: func() []string { return apps },
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}

	return p.Discover(context.Background())
}

func TestMacAppDiscovery(t *testing.T) {
	app := writeMacBundleFixture(t, filepath.Join(t.TempDir(), "Applications", "ZCode.app"))
	homeApp := writeMacBundleFixture(t, filepath.Join(t.TempDir(), "ZCode.app"))

	install, probes, err := discoverMacApps(t, []string{app, homeApp})
	if err != nil {
		t.Fatalf("discover mac apps: %v", err)
	}

	if install.Layout != fixtureMacAppLayout || install.Version != fixtureMacAppVersion {
		t.Fatalf("install = %+v, want mac-app version 4.0.0", install)
	}
	if install.AppImage != "" || install.RequiresMount() {
		t.Fatalf("mac app must not require a mount: %+v", install)
	}
	if install.BuiltinFile == "" || !strings.Contains(install.BuiltinFile, "zcode-builtin.json") {
		t.Fatalf("mac app must resolve the bundled provider table: %+v", install)
	}

	selected, ok := probeRow(probes, fixtureMacAppLayout)
	if !ok || !strings.Contains(selected.Result, "selected: mac-app version 4.0.0") {
		t.Fatalf("mac-app row = %+v, want a selected row", selected)
	}
	if rows := probeRows(probes, "mac-app-candidate"); len(rows) != 2 {
		t.Fatalf("mac-app-candidate rows = %d, want 2: %+v", len(rows), rows)
	}
	appImage, ok := probeRow(probes, "appimage-rank")
	if !ok || !strings.Contains(appImage.Result, "not consulted") {
		t.Fatalf("appimage stage must not be consulted after a mac win: %+v", appImage)
	}
	bundled, ok := probeRow(probes, "builtin-bundled")
	if !ok || !strings.Contains(bundled.Query, "Contents/Resources/config/provider") {
		t.Fatalf("builtin-bundled row must name the app-relative provider path: %+v", bundled)
	}
}

func TestMacAppDiscoveryRejectsInvalid(t *testing.T) {
	app := writeMacBundleFixture(t, filepath.Join(t.TempDir(), "ZCode.app"))
	if err := os.Remove(filepath.Join(app, "Contents", "Resources", "config", "provider",
		"zcode-builtin.json")); err != nil {
		t.Fatalf("remove provider: %v", err)
	}

	_, probes, err := discoverMacApps(t, []string{app})
	if !errors.Is(err, supervisor.ErrNoInstall) {
		t.Fatalf("discover invalid app err = %v, want ErrNoInstall", err)
	}

	candidates := probeRows(probes, "mac-app-candidate")
	if len(candidates) != 1 || !strings.Contains(candidates[0].Result, "zcode-builtin.json") {
		t.Fatalf("invalid app must produce an actionable candidate row: %+v", candidates)
	}
}

//nolint:cyclop,funlen // each override spelling is a separate acceptance branch
func TestMacAppOverrideForms(t *testing.T) {
	app := writeMacBundleFixture(t, filepath.Join(t.TempDir(), "ZCode.app"))
	canonical := canonicalTestPath(t, app)

	want, ok := supervisor.OverrideInstall(app)
	if !ok {
		t.Fatal("the .app override must resolve")
	}
	if want.Layout != fixtureMacAppLayout || want.Root != canonical || want.Version != fixtureMacAppVersion {
		t.Fatalf("app override = %+v", want)
	}

	for _, tc := range []struct {
		name string
		path string
	}{
		{"Contents", filepath.Join(app, "Contents")},
		{"executable", filepath.Join(app, "Contents", "MacOS", "ZCode")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := supervisor.OverrideInstall(tc.path)
			if !ok {
				t.Fatalf("%s override did not resolve", tc.name)
			}
			if got != want {
				t.Fatalf("%s override = %+v, want %+v", tc.name, got, want)
			}
		})
	}

	t.Run("relative .app", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("getwd: %v", err)
		}
		rel, err := filepath.Rel(wd, app)
		if err != nil {
			t.Fatalf("rel %s: %v", app, err)
		}

		got, ok := supervisor.OverrideInstall(rel)
		if !ok || got != want {
			t.Fatalf("relative app override = %+v ok=%v, want %+v", got, ok, want)
		}
	})

	t.Run("symlinked .app", func(t *testing.T) {
		link := filepath.Join(t.TempDir(), "ZCode-current.app")
		if err := os.Symlink(app, link); err != nil {
			t.Fatalf("symlink %s: %v", link, err)
		}

		got, ok := supervisor.OverrideInstall(link)
		if !ok || got != want {
			t.Fatalf("symlinked app override = %+v ok=%v, want %+v", got, ok, want)
		}
	})

	t.Run("invalid app", func(t *testing.T) {
		broken := filepath.Join(t.TempDir(), "Broken.app")
		if err := os.MkdirAll(filepath.Join(broken, "Contents"), 0o750); err != nil {
			t.Fatalf("mkdir broken app: %v", err)
		}

		if inst, ok := supervisor.OverrideInstall(broken); ok {
			t.Fatalf("invalid .app override must be rejected: %+v", inst)
		}
	})
}

func TestMacAppOverrideProbeRow(t *testing.T) {
	app := writeMacBundleFixture(t, filepath.Join(t.TempDir(), "ZCode.app"))

	p := supervisor.Probers{
		Home:    t.TempDir(),
		OptDir:  filepath.Join(t.TempDir(), "ZCode"),
		AppDirs: func() []string { return nil },
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}
	p.Override = func() (supervisor.Install, bool) {
		return supervisor.OverrideInstall(app)
	}

	_, probes, err := p.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover with override: %v", err)
	}

	row, ok := probeRow(probes, "override")
	if !ok {
		t.Fatal("override row missing")
	}
	for _, want := range []string{"found: mac-app", canonicalTestPath(t, app), "(version 4.0.0)"} {
		if !strings.Contains(row.Result, want) {
			t.Fatalf("override row %q missing %q", row.Result, want)
		}
	}
}

// writeMacRuntimeTree plants the Darwin runtime-provider cache for an app
// version and one newer version, returning the provider root.
func writeMacRuntimeTree(t *testing.T, home, platform string, versions []string, revs []int) string {
	t.Helper()

	provider := filepath.Join(home, ".zcode", "v2", "runtime", "provider", platform)
	for i, version := range versions {
		dir := filepath.Join(provider, version, fixtureEndpointDir)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir runtime %s: %v", version, err)
		}
		body := fmt.Sprintf(`{"schemaVersion":1,"revision":%d}`, revs[i])
		if err := os.WriteFile(filepath.Join(dir, "zcode-builtin.json"), []byte(body), 0o600); err != nil {
			t.Fatalf("write runtime builtin %s: %v", version, err)
		}
	}

	return provider
}

func TestMacAppProviderCache(t *testing.T) {
	home := t.TempDir()
	app := writeMacBundleFixture(t, filepath.Join(home, "Applications", "ZCode.app"))
	provider := writeMacRuntimeTree(t, home, "darwin-aarch64", []string{"4.9.0", fixtureMacAppVersion}, []int{99, 48})

	p := supervisor.Probers{
		Home:                    home,
		OptDir:                  filepath.Join(t.TempDir(), "ZCode"),
		MacApps:                 func() []string { return []string{app} },
		RuntimeProviderPlatform: "darwin-aarch64",
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}

	install, probes, err := p.Discover(context.Background())
	if err != nil {
		t.Fatalf("discover mac app with provider cache: %v", err)
	}

	want := filepath.Join(provider, fixtureMacAppVersion, fixtureEndpointDir, "zcode-builtin.json")
	if install.BuiltinFile != want {
		t.Fatalf("BuiltinFile = %q, want the app-version cache path %q", install.BuiltinFile, want)
	}

	row, ok := probeRow(probes, "builtin-runtime")
	if !ok || !strings.Contains(row.Query, "darwin-aarch64") {
		t.Fatalf("builtin-runtime row must name the Darwin platform: %+v", row)
	}
}

func TestMacAppSpawnContract(t *testing.T) {
	app := writeMacBundleFixture(t, filepath.Join(t.TempDir(), "ZCode.app"))
	install, _, err := discoverMacApps(t, []string{app})
	if err != nil {
		t.Fatalf("discover mac app: %v", err)
	}

	root := canonicalTestPath(t, app)
	got := supervisor.SpawnArgv(install)
	want := []string{
		filepath.Join(root, "Contents", "MacOS", "ZCode"),
		filepath.Join(root, "Contents", "Resources", "glm", "zcode.cjs"),
		"app-server",
		"--stdio",
	}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("SpawnArgv = %v, want %v", got, want)
	}

	env := map[string]string{}
	for _, kv := range supervisor.ChildEnv(install, []string{
		"PATH=/caller/bin:/usr/bin",
		"HOME=/home/fixture",
		"HTTP_PROXY=http://192.0.2.1:9",
		"SECRET=must-not-cross",
	}) {
		key, value, ok := strings.Cut(kv, "=")
		if ok {
			env[key] = value
		}
	}

	if env["ELECTRON_RUN_AS_NODE"] != "1" {
		t.Fatalf("ELECTRON_RUN_AS_NODE = %q, want 1", env["ELECTRON_RUN_AS_NODE"])
	}
	if env["ZCODE_BUILTIN_PROVIDER_CONFIG_FILE"] != install.BuiltinFile {
		t.Fatalf("builtin env = %q, want %q",
			env["ZCODE_BUILTIN_PROVIDER_CONFIG_FILE"], install.BuiltinFile)
	}
	if env["PATH"] != "/caller/bin:/usr/bin" {
		t.Fatalf("PATH = %q, want the caller PATH copied", env["PATH"])
	}
	for _, key := range []string{"HTTP_PROXY", "SECRET"} {
		if _, present := env[key]; present {
			t.Fatalf("%s crossed into the child env", key)
		}
	}
}
