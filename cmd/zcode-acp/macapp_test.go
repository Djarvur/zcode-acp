//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ompTextMacAppInstall prepares a fake ZCode.app for the text-task E2E: the
// macOS spawn contract (Contents/MacOS/ZCode + Contents/Resources/glm/zcode.cjs),
// the bundled provider table, and a shim that drops the entrypoint argument
// before exec'ing the fake app-server.
func ompTextMacAppInstall(t *testing.T, fakeBin, scenario string) (appPath, transcriptPath string) {
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

	shim := fmt.Sprintf("#!/bin/sh\nshift\nexec %s --scenario %s --transcript %s \"$@\"\n",
		fakeBin, scenarioAbs, transcriptPath)
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

// TestOmpTextMacAppEndToEnd proves Orca's exact text argv with --app-server
// pointing at a ZCode.app: discovery/override resolve the app, the canonical
// macOS argv reaches the fake backend, and the stdout/exit contract holds.
func TestOmpTextMacAppEndToEnd(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	appPath, transcriptPath := ompTextMacAppInstall(t, fakeBin, ompTextScenario)
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, appPath),
		textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	assertTextStdout(t, res, 0)

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

	ompTextAssertNoACP(t, res.lines)
}
