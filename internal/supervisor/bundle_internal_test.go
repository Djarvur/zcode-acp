//go:build darwin

package supervisor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// macAppFixture is a disposable app bundle tree for discovery tests.
type macAppFixture struct {
	App        string
	Binary     string
	Entrypoint string
	Provider   string
}

// macCanonicalPath mirrors production's EvalSymlinks-resolved expectations
// (macOS /var -> /private/var).
func macCanonicalPath(t *testing.T, path string) string {
	t.Helper()

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs %s: %v", path, err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}

	return resolved
}

// writeMacApp lays down a complete ZCode.app bundle with an XML Info.plist.
func writeMacApp(t *testing.T) macAppFixture {
	t.Helper()

	return writeMacAppAt(t, filepath.Join(t.TempDir(), "ZCode.app"))
}

// writeMacAppAt lays down a complete bundle at app.
func writeMacAppAt(t *testing.T, app string) macAppFixture {
	t.Helper()

	fixture := macAppFixture{
		App:        app,
		Binary:     filepath.Join(app, "Contents", "MacOS", "ZCode"),
		Entrypoint: filepath.Join(app, "Contents", "Resources", "glm", "zcode.cjs"),
		Provider: filepath.Join(app, "Contents", "Resources", "config", "provider",
			"zcode-builtin.json"),
	}

	for _, dir := range []string{
		filepath.Dir(fixture.Binary),
		filepath.Dir(fixture.Entrypoint),
		filepath.Dir(fixture.Provider),
	} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	writeMacInfo(t, app, "ZCode", "4.0.0", "4.0.0.10870")
	writeMacFile(t, fixture.Binary, 0o755)
	writeMacFile(t, fixture.Entrypoint, 0o644)
	writeMacFile(t, fixture.Provider, 0o644)

	return fixture
}

// writeMacInfo writes a minimal XML Info.plist.
func writeMacInfo(t *testing.T, app, executable, version, build string) {
	t.Helper()

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	body.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" ` +
		`"http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	body.WriteString(`<plist version="1.0"><dict>` + "\n")
	body.WriteString(`<key>CFBundleIdentifier</key><string>dev.zcode.app</string>` + "\n")
	if executable != "" {
		body.WriteString(`<key>CFBundleExecutable</key><string>` + executable + `</string>` + "\n")
	}
	if version != "" {
		body.WriteString(`<key>CFBundleShortVersionString</key><string>` + version + `</string>` + "\n")
	}
	if build != "" {
		body.WriteString(`<key>CFBundleVersion</key><string>` + build + `</string>` + "\n")
	}
	body.WriteString(`</dict></plist>` + "\n")

	path := filepath.Join(app, "Contents", "Info.plist")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err != nil {
		t.Fatalf("write Info.plist: %v", err)
	}
}

// writeMacFile creates a regular file with mode.
func writeMacFile(t *testing.T, path string, mode os.FileMode) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("fixture\n"), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestReadBundleInfoBinaryContext proves a canceled caller context reaches the
// binary-plist fallback. The local fixture is deliberately binary so the XML
// parser cannot accept it; an already-canceled context prevents plutil launch.
func TestReadBundleInfoBinaryContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Info.plist")
	if err := os.WriteFile(path, []byte("bplist00\x00\x08"), 0o600); err != nil {
		t.Fatalf("write binary plist fixture: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := readBundleInfo(ctx, path)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("readBundleInfo() error = %v, want context.Canceled", err)
	}
	if elapsed > time.Second {
		t.Fatalf("readBundleInfo() took %v with an already-canceled context", elapsed)
	}
}

func TestScanMacApps(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"ZCode.app", "ZCode Preview.app", "Other.app"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
	}
	writeMacFile(t, filepath.Join(dir, "ZCode.app", "marker"), 0o644)

	got := scanMacApps([]string{dir, dir})
	if len(got) != 2 {
		t.Fatalf("scanMacApps = %v, want two deduplicated ZCode*.app dirs", got)
	}
	for _, app := range got {
		if filepath.Base(app) == "Other.app" {
			t.Fatalf("scanMacApps selected an unrelated app: %v", got)
		}
	}
}

//nolint:cyclop,funlen // the table of small rejection branches is the test.s point
func TestMacAppInstall(t *testing.T) {
	t.Run("valid bundle", func(t *testing.T) {
		fixture := writeMacApp(t)

		inst, reason := macAppInstall(context.Background(), fixture.App)
		if reason != "" {
			t.Fatalf("valid app rejected: %s", reason)
		}

		root := macCanonicalPath(t, fixture.App)
		if inst.Root != root || inst.Binary != filepath.Join(root, "Contents", "MacOS", "ZCode") ||
			inst.Entrypoint != filepath.Join(root, "Contents", "Resources", "glm", "zcode.cjs") ||
			inst.Layout != "mac-app" || inst.Version != "4.0.0" {
			t.Fatalf("unexpected install: %+v", inst)
		}
		if inst.AppImage != "" || inst.RequiresMount() {
			t.Fatalf("a mac app must never require a mount: %+v", inst)
		}
	})

	t.Run("build version fallback", func(t *testing.T) {
		fixture := writeMacApp(t)
		writeMacInfo(t, fixture.App, "ZCode", "", "4.0.0.10870")

		inst, reason := macAppInstall(context.Background(), fixture.App)
		if reason != "" {
			t.Fatalf("build-version fallback rejected: %s", reason)
		}
		if inst.Version != "4.0.0.10870" {
			t.Fatalf("version = %q, want CFBundleVersion fallback", inst.Version)
		}
	})

	t.Run("missing Info.plist", func(t *testing.T) {
		fixture := writeMacApp(t)
		if err := os.Remove(filepath.Join(fixture.App, "Contents", "Info.plist")); err != nil {
			t.Fatalf("remove Info.plist: %v", err)
		}

		assertMacAppMiss(t, fixture.App, "Info.plist")
	})

	t.Run("malformed Info.plist", func(t *testing.T) {
		fixture := writeMacApp(t)
		if err := os.WriteFile(filepath.Join(fixture.App, "Contents", "Info.plist"),
			[]byte("not a plist"), 0o644); err != nil {
			t.Fatalf("write malformed plist: %v", err)
		}

		assertMacAppMiss(t, fixture.App, "Info.plist")
	})

	t.Run("missing executable key", func(t *testing.T) {
		fixture := writeMacApp(t)
		writeMacInfo(t, fixture.App, "", "4.0.0", "4.0.0.10870")

		assertMacAppMiss(t, fixture.App, "CFBundleExecutable")
	})

	t.Run("missing executable file", func(t *testing.T) {
		fixture := writeMacApp(t)
		if err := os.Remove(fixture.Binary); err != nil {
			t.Fatalf("remove binary: %v", err)
		}

		assertMacAppMiss(t, fixture.App, "bundle executable")
	})

	t.Run("non-executable binary", func(t *testing.T) {
		fixture := writeMacApp(t)
		if err := os.Chmod(fixture.Binary, 0o644); err != nil {
			t.Fatalf("chmod binary: %v", err)
		}

		assertMacAppMiss(t, fixture.App, "not executable")
	})

	t.Run("missing entrypoint", func(t *testing.T) {
		fixture := writeMacApp(t)
		if err := os.Remove(fixture.Entrypoint); err != nil {
			t.Fatalf("remove entrypoint: %v", err)
		}

		assertMacAppMiss(t, fixture.App, "zcode.cjs")
	})

	t.Run("missing bundled provider table", func(t *testing.T) {
		fixture := writeMacApp(t)
		if err := os.Remove(fixture.Provider); err != nil {
			t.Fatalf("remove provider: %v", err)
		}

		assertMacAppMiss(t, fixture.App, "zcode-builtin.json")
	})

	t.Run("missing version", func(t *testing.T) {
		fixture := writeMacApp(t)
		writeMacInfo(t, fixture.App, "ZCode", "", "")

		assertMacAppMiss(t, fixture.App, "CFBundleShortVersionString")
	})

	t.Run("executable traversal", func(t *testing.T) {
		fixture := writeMacApp(t)
		writeMacInfo(t, fixture.App, "../ZCode", "4.0.0", "")

		assertMacAppMiss(t, fixture.App, "escapes")
	})

	t.Run("not an app bundle", func(t *testing.T) {
		dir := t.TempDir()

		assertMacAppMiss(t, dir, "not a .app bundle")
	})
}

// assertMacAppMiss requires a rejection whose reason contains want.
func assertMacAppMiss(t *testing.T, app, want string) {
	t.Helper()

	_, reason := macAppInstall(context.Background(), app)
	if reason == "" {
		t.Fatalf("invalid app %s accepted", app)
	}
	if !strings.Contains(reason, want) {
		t.Fatalf("reason %q does not name %q", reason, want)
	}
}
