//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ompTextMacAppInstall prepares a fake ZCode.app for the text-task E2E: the
// macOS spawn contract (Contents/MacOS/ZCode + Contents/Resources/glm/zcode.cjs),
// the bundled provider table, and an Electron shim that records its selection
// before dropping the entrypoint argument and exec'ing the fake app-server.
func ompTextMacAppInstall(
	t *testing.T, fakeBin, scenario, electronMarker string,
) (appPath, transcriptPath string) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(scenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", scenario, err)
	}

	transcriptPath = filepath.Join(t.TempDir(), "fake-transcript.txt")
	appPath = filepath.Join(t.TempDir(), "ZCode.app")

	contents := filepath.Join(appPath, "Contents")
	macosDir := filepath.Join(contents, "MacOS")
	resources := filepath.Join(contents, "Resources")
	for _, dir := range []string{macosDir, filepath.Join(resources, "glm"),
		filepath.Join(resources, "config", "provider")} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("create app dir %s: %v", dir, err)
		}
	}

	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>dev.zcode.app</string>
<key>CFBundleExecutable</key><string>ZCode</string>
<key>CFBundleShortVersionString</key><string>4.0.0</string>
</dict></plist>
`
	if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatalf("write Info.plist: %v", err)
	}

	shim := fmt.Sprintf("#!/bin/sh\n: > %q\nshift\nexec %q --scenario %q --transcript %q \"$@\"\n",
		electronMarker, fakeBin, scenarioAbs, transcriptPath)
	if err := os.WriteFile(filepath.Join(macosDir, "ZCode"), []byte(shim), 0o750); err != nil {
		t.Fatalf("write app executable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resources, "glm", "zcode.cjs"), []byte("\n"), 0o640); err != nil {
		t.Fatalf("write entrypoint: %v", err)
	}
	if err := os.WriteFile(filepath.Join(resources, "config", "provider", "zcode-builtin.json"),
		[]byte(`{"revision":44}`), 0o600); err != nil {
		t.Fatalf("write bundled provider: %v", err)
	}

	return appPath, transcriptPath
}

// ompTextHermeticShell writes a login-shell shim named sh. It ignores the
// production -lc/script arguments, records that it ran, and prints only the
// inherited fixture PATH: /etc/profile and path_helper are never consulted.
func ompTextHermeticShell(t *testing.T, dir, marker string) string {
	t.Helper()
	path := filepath.Join(dir, "sh")
	body := fmt.Sprintf("#!/bin/sh\n: > %q\nprintf '%%s\\n' \"$PATH\"\n", marker)
	if err := os.WriteFile(path, []byte(body), 0o750); err != nil {
		t.Fatalf("write hermetic login-shell shim: %v", err)
	}

	return path
}

// ompTextNodeFixture writes the shipping Node launcher fixture. It records the
// exact argv, drops the app entrypoint, and execs the built fake app-server.
func ompTextNodeFixture(t *testing.T, dir, marker, fakeBin, scenario, transcript string) {
	t.Helper()
	scenarioAbs, err := filepath.Abs(scenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", scenario, err)
	}
	body := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nshift\nexec %q --scenario %q --transcript %q \"$@\"\n",
		marker, fakeBin, scenarioAbs, transcript)
	if err := os.WriteFile(filepath.Join(dir, "node"), []byte(body), 0o750); err != nil {
		t.Fatalf("write fake node: %v", err)
	}
}

// TestOmpTextMacAppEndToEnd proves both shipping launcher choices independently:
// Node is the default, and Electron runs only through the explicit fallback.
func TestOmpTextMacAppEndToEnd(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("shipping Node default", func(t *testing.T) {
		testOmpTextMacAppNodeDefault(t, agentBin, fakeBin)
	})
	t.Run("explicit Electron fallback", func(t *testing.T) {
		testOmpTextMacAppElectronFallback(t, agentBin, fakeBin)
	})
}

func testOmpTextMacAppNodeDefault(t *testing.T, agentBin, fakeBin string) {
	t.Helper()
	fixtureDir := t.TempDir()
	electronMarker := filepath.Join(fixtureDir, "electron.marker")
	nodeMarker := filepath.Join(fixtureDir, "node.marker")
	shellMarker := filepath.Join(fixtureDir, "shell.marker")
	appPath, transcriptPath := ompTextMacAppInstall(t, fakeBin, ompTextScenario, electronMarker)
	ompTextNodeFixture(t, fixtureDir, nodeMarker, fakeBin, ompTextScenario, transcriptPath)
	shell := ompTextHermeticShell(t, fixtureDir, shellMarker)
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, appPath), textRide{
		dir:            t.TempDir(),
		transcriptPath: transcriptPath,
		path:           fixtureDir,
		extraEnv:       []string{"SHELL=" + shell},
	})

	assertTextStdout(t, res, 0)
	entrypoint, err := filepath.EvalSymlinks(filepath.Join(appPath, "Contents", "Resources", "glm", "zcode.cjs"))
	if err != nil {
		t.Fatalf("resolve app entrypoint: %v", err)
	}
	assertFixtureLines(t, nodeMarker, []string{entrypoint, "app-server", "--stdio"})
	assertFixtureAbsent(t, electronMarker)
	assertDarwinShellMarker(t, shellMarker)
	assertOmpTextMacAppWire(t, res, transcriptPath)
}

func testOmpTextMacAppElectronFallback(t *testing.T, agentBin, fakeBin string) {
	t.Helper()
	fixtureDir := t.TempDir()
	electronMarker := filepath.Join(fixtureDir, "electron.marker")
	nodeMarker := filepath.Join(fixtureDir, "node.marker")
	shellMarker := filepath.Join(fixtureDir, "shell.marker")
	appPath, transcriptPath := ompTextMacAppInstall(t, fakeBin, ompTextScenario, electronMarker)
	shell := ompTextHermeticShell(t, fixtureDir, shellMarker)
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, appPath), textRide{
		dir:                 t.TempDir(),
		transcriptPath:      transcriptPath,
		path:                fixtureDir,
		disableNodeLauncher: true,
		extraEnv:            []string{"SHELL=" + shell},
	})

	assertTextStdout(t, res, 0)
	assertFixtureExists(t, electronMarker)
	assertFixtureAbsent(t, nodeMarker)
	assertDarwinShellMarker(t, shellMarker)
	assertOmpTextMacAppWire(t, res, transcriptPath)
}

func assertOmpTextMacAppWire(t *testing.T, res textRun, transcriptPath string) {
	t.Helper()
	if res.linesErr != nil {
		t.Fatalf("fake transcript: %v (stderr %q)", res.linesErr, res.stderr)
	}
	params, _ := ompTextOnlyGenerateFrame(t, res.lines)
	if params.Prompt != ompTextPromptBody {
		t.Fatalf("prompt = %q, want %q", params.Prompt, ompTextPromptBody)
	}
	if params.Selection.ProviderID != "OpenCodeGo" || params.Selection.ModelID != "deepseek-v4.1-flash" {
		t.Fatalf("selection = %+v, want the --model ids", params.Selection)
	}
	if methods := ompTextInboundMethods(t, res.lines); len(methods) != 1 {
		t.Fatalf("inbound methods = %v, want exactly one generateText", methods)
	}
	ompTextAssertNoACP(t, res.lines)
}

func assertFixtureLines(t *testing.T, path string, want []string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture marker %s: %v", path, err)
	}
	got := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("fixture marker %s = %q, want %q", path, got, want)
	}
}

func assertFixtureExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("fixture marker %s is not a regular file: %v", path, err)
	}
}

func assertFixtureAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("fixture marker %s must not exist (stat error %v)", path, err)
	}
}

func assertDarwinShellMarker(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "darwin" {
		assertFixtureExists(t, path)
	}
}
