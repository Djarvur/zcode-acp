//go:build !windows

package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Text-task fixture literals (the goconst discipline): the client's own flag
// spellings, its model/level values, the prompt-token shapes, and the fake
// scenarios' observable strings.
const (
	ompFlagPrint   = "--print"
	ompFlagMode    = "--mode"
	ompFlagFuture  = "--future-flag"
	ompModelFlash  = "zai-api/GLM-5.3-Flash"
	ompLevelLow    = "low"
	ompTextLiteral = "hello"
	ompPromptFile  = "@/tmp/prompt.txt"
	ompPromptOne   = "@/tmp/a.txt"
	ompPromptTwo   = "@/tmp/b.txt"

	ompTextModelPersonal   = "OpenCodeGo/deepseek-v4.1-flash"
	ompTextPromptBody      = "Name this chat."
	ompTextReply           = "Answer: done"
	ompTextQuerySource     = "zcode-acp-text-task"
	ompTextMaxOutputTokens = 1024
	ompTextAPIKeyFixture   = "fixture-key-not-a-credential"
	ompTextScenario        = "testdata/omp-text-e2e.json"
	ompTextAPIKeyScenario  = "testdata/omp-text-apikey-e2e.json"

	// The one-letter prompt chunks the multi-chunk arm concatenates.
	ompTextPromptHead = "Name this "
	ompTextPromptTail = "chat."
)

// Ad-hoc scenarios (whitespace text and a scripted wire error) live as
// runtime temp files — the committed testdata set stays the two documented
// fixtures.
const (
	ompTextEmptyScenarioJSON = `{"name":"omp-text-empty","on_request":[` +
		`{"method":"workspace/generateText","reply":{"result":{"text":"  \n\t",` +
		`"selection":{"providerId":"p","modelId":"m"},"finishReason":"stop"}}}],` +
		`"eof_drain_ms":100}`

	ompTextWireErrorScenarioJSON = `{"name":"omp-text-wire-error","on_request":[` +
		`{"method":"workspace/generateText","reply":{"error":{"code":-32031,` +
		`"message":"provider headers not applied"}}}],"eof_drain_ms":100}`
)

// ompTextArgs builds Orca 1.4.271's exact non-interactive text-task argv (the
// commit-message-agent spec): --print --no-session --no-tools
// --no-extensions --no-skills --no-rules --mode text, an optional --model
// (Orca omits it for "default"), an optional --thinking level, then the
// prompt tokens — with the arm token in front.
func ompTextArgs(model, level string, prompts ...string) []string {
	argv := []string{
		ompSubcommand,
		ompFlagPrint,
		"--no-session",
		"--no-tools",
		"--no-extensions",
		"--no-skills",
		"--no-rules",
		ompFlagMode, ompModeText,
	}

	if model != "" {
		argv = append(argv, ompFlagModel, model)
	}

	if level != "" {
		argv = append(argv, "--thinking", level)
	}

	return append(argv, prompts...)
}

// TestOmpTextArgs locks the omp argv text-task surface: the exact Orca argv,
// the marker rule (the arm token and the `acp` card marker are never prompt
// tokens), @file collection order, unknown-flag tolerance, and the mode
// validation sentinel.
func TestOmpTextArgs(t *testing.T) {
	t.Run("the exact Orca text argv parses to the text-task surface", testOmpTextExactArgv)

	t.Run("--print alone and --mode text alone both activate text mode", testOmpTextActivation)

	t.Run("omp acp keeps printMode false and collects no prompt tokens", testOmpTextAcpMarkers)

	t.Run("the arm token and the card marker are never prompt tokens", testOmpTextMarkerRule)

	t.Run("two @file tokens are collected in argv order", testOmpTextPromptOrder)

	t.Run("unknown flags stay tolerated", testOmpTextUnknownTolerance)

	t.Run("--mode plan answers the unsupported-mode sentinel", testOmpTextUnsupportedMode)

	t.Run("--print with no positional parses cleanly with zero prompt tokens", testOmpTextPrintNoPrompt)

	t.Run("textLevel prefers --thinking over --reasoning-level", testOmpTextLevel)
}

// testOmpTextExactArgv locks Orca's exact argv.
func testOmpTextExactArgv(t *testing.T) {
	t.Helper()

	opts, err := ompParse(ompTextArgs(ompModelFlash, e2eLevel, ompPromptFile))
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if !opts.printMode {
		t.Fatal("printMode = false, want true (--print)")
	}

	if opts.mode != ompModeText {
		t.Fatalf("mode = %q, want %q", opts.mode, ompModeText)
	}

	if opts.model != ompModelFlash {
		t.Fatalf("model = %q, want the --model value", opts.model)
	}

	if opts.thinking != e2eLevel {
		t.Fatalf("thinking = %q, want the --thinking value", opts.thinking)
	}

	if !opts.noSession || !opts.noTools || !opts.noExtensions || !opts.noSkills || !opts.noRules {
		t.Fatalf("no-* flags not all true: %+v", opts)
	}

	if len(opts.prompts) != 1 || opts.prompts[0] != ompPromptFile {
		t.Fatalf("prompts = %v, want the one @file token", opts.prompts)
	}

	if !opts.textMode() {
		t.Fatal("textMode() = false, want true")
	}
}

// testOmpTextActivation locks the either-flag activation rule.
func testOmpTextActivation(t *testing.T) {
	t.Helper()

	for _, argv := range [][]string{
		{ompSubcommand, ompFlagPrint, ompTextLiteral},
		{ompSubcommand, ompFlagMode, ompModeText, ompTextLiteral},
	} {
		opts, err := ompParse(argv)
		if err != nil {
			t.Fatalf("%v: %v", argv, err)
		}

		if !opts.textMode() {
			t.Fatalf("%v: textMode() = false, want true", argv)
		}
	}
}

// testOmpTextAcpMarkers locks the ACP card argv: no text mode, no prompts.
func testOmpTextAcpMarkers(t *testing.T) {
	t.Helper()

	for _, argv := range [][]string{
		{ompSubcommand, ompCardMarker},
		{ompSubcommand, ompCardMarker, ompFlagModel, ompModelShort},
	} {
		opts, err := ompParse(argv)
		if err != nil {
			t.Fatalf("%v: %v", argv, err)
		}

		if opts.printMode || opts.textMode() {
			t.Fatalf("%v: text mode selected for the ACP card argv: %+v", argv, opts)
		}

		if len(opts.prompts) != 0 {
			t.Fatalf("%v: prompts = %v, want none (acp is the card marker)", argv, opts.prompts)
		}
	}
}

// testOmpTextMarkerRule locks the arm-token/card-marker exclusion.
func testOmpTextMarkerRule(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompCardToken, ompTextLiteral})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if len(opts.prompts) != 1 || opts.prompts[0] != ompTextLiteral {
		t.Fatalf("prompts = %v, want the single literal prompt", opts.prompts)
	}
}

// testOmpTextPromptOrder locks argv-order @file collection.
func testOmpTextPromptOrder(t *testing.T) {
	t.Helper()

	opts, err := ompParse(ompTextArgs("", "", ompPromptOne, ompPromptTwo))
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	want := []string{ompPromptOne, ompPromptTwo}
	if len(opts.prompts) != 2 || opts.prompts[0] != want[0] || opts.prompts[1] != want[1] {
		t.Fatalf("prompts = %v, want %v (argv order)", opts.prompts, want)
	}
}

// testOmpTextUnknownTolerance locks the unknown-flag tolerance arm.
func testOmpTextUnknownTolerance(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompFlagPrint, ompFlagFuture, "v", ompFlagModel, ompModelShort})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if opts.unknownCount != 1 || opts.model != ompModelShort {
		t.Fatalf("unknownCount = %d model = %q, want 1 and the parsed model", opts.unknownCount, opts.model)
	}
}

// testOmpTextUnsupportedMode locks the unsupported-mode sentinel.
func testOmpTextUnsupportedMode(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompFlagMode, "plan", ompTextLiteral})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	verr := opts.validateMode()
	if !errors.Is(verr, errUnsupportedMode) {
		t.Fatalf("validateMode(--mode plan) = %v, want the errUnsupportedMode sentinel", verr)
	}

	if !strings.Contains(verr.Error(), "plan") {
		t.Fatalf("validateMode error %q must name the got mode", verr)
	}
}

// testOmpTextPrintNoPrompt locks the parse-clean, zero-prompt shape (the
// missing-prompt exit-2 class is the run-time gate's case).
func testOmpTextPrintNoPrompt(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompFlagPrint})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if !opts.printMode || len(opts.prompts) != 0 {
		t.Fatalf("opts = %+v, want printMode true and zero prompts", opts)
	}
}

// testOmpTextLevel locks the level precedence rule.
func testOmpTextLevel(t *testing.T) {
	t.Helper()

	if got := textLevel(ompOptions{thinking: e2eLevel, reasoningLevel: ompLevelLow}); got != e2eLevel {
		t.Fatalf("textLevel = %q, want the --thinking value", got)
	}

	if got := textLevel(ompOptions{reasoningLevel: ompLevelLow}); got != ompLevelLow {
		t.Fatalf("textLevel = %q, want the reasoning-level fallback", got)
	}

	if got := textLevel(ompOptions{}); got != "" {
		t.Fatalf("textLevel = %q, want empty", got)
	}
}

// textRide is one built-binary text run's inputs: the hermetic HOME (a fresh
// temp dir unless the provider_config arm supplies its own), the optional
// wrapper api key, the command's Dir (the workspace ref), and the fake
// transcript path to read back ("" = none).
type textRide struct {
	home                string
	apiKey              string
	dir                 string
	transcriptPath      string
	path                string
	disableNodeLauncher bool
	extraEnv            []string
}

// textRun is one completed text run's observable output plus the fake
// transcript when one was written.
type textRun struct {
	stdout   string
	stderr   string
	exit     int
	lines    []e2eTranscriptLine
	linesErr error
}

// ompTextInstall prepares the fake install root for one text run: the
// invariant spawn surface with the fake shim, plus the bundled builtin
// fixture at resources/config/provider/zcode-builtin.json. bundleInstall
// leaves BuiltinFile empty, so discovery's bundled probe is the only way
// authenticateAPIKey's snapshot Build can read a revision.
func ompTextInstall(t *testing.T, fakeBin, scenario string) (installRoot, transcriptPath string) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(scenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", scenario, err)
	}

	transcriptPath = filepath.Join(t.TempDir(), "fake-transcript.txt")
	installRoot = t.TempDir()

	e2eWriteFakeInstallRoot(t, installRoot, fakeBin, scenarioAbs, transcriptPath)

	builtinDir := filepath.Join(installRoot, "resources", "config", "provider")
	if err := os.MkdirAll(builtinDir, 0o750); err != nil {
		t.Fatalf("create bundled builtin dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(builtinDir, "zcode-builtin.json"),
		[]byte(`{"revision":44}`), 0o600); err != nil {
		t.Fatalf("write bundled builtin fixture: %v", err)
	}

	return installRoot, transcriptPath
}

// writeTextScenario writes an ad-hoc scenario JSON to the test's temp dir.
func writeTextScenario(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "scenario.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write scenario %s: %v", path, err)
	}

	return path
}

// writeTextPrompt writes a BOM-prefixed prompt chunk file (Orca's delivery
// shape) and returns its absolute path.
func writeTextPrompt(t *testing.T, dir, name, body string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("\ufeff"+body), 0o600); err != nil {
		t.Fatalf("write prompt file %s: %v", path, err)
	}

	return path
}

// ompTextRunArgv is Orca's exact text argv with the harness's --app-server
// override appended last (Orca has no such flag; the override is discovery
// step 1).
func ompTextRunArgv(model, promptPath, installRoot string) []string {
	argv := ompTextArgs(model, "", "@"+promptPath)

	return append(argv, ompFlagAppServer, installRoot)
}

// ompRunText runs the built binary with an explicit hermetic env (PATH and a
// fresh temp HOME only, plus ZCODE_ACP_API_KEY for the auth arm), closes
// stdin immediately (the text arm never reads it — Orca ends the child's
// stdin after handing over the record stream), waits for exit, and returns
// the captured streams plus the fake transcript when one appeared.
func ompRunText(t *testing.T, bin string, argv []string, ride textRide) textRun {
	t.Helper()

	home := ride.home
	if home == "" {
		home = t.TempDir()
	}

	ctx, cancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer cancel()

	stdoutBuf := &e2eSyncBuffer{}
	stderrBuf := &e2eSyncBuffer{}

	cmd := exec.CommandContext(ctx, bin, argv...)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "ZCODE_ACP_DISABLE_NODE_LAUNCHER=1"}
	if ride.apiKey != "" {
		cmd.Env = append(cmd.Env, "ZCODE_ACP_API_KEY="+ride.apiKey)
	}

	cmd.Dir = ride.dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("wrapper stdin pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn %s: %v", bin, err)
	}

	t.Cleanup(e2eKillGroup(t, cmd, stderrBuf))

	if err := stdin.Close(); err != nil {
		t.Logf("close wrapper stdin: %v", err)
	}

	exit := 0

	if waitErr := cmd.Wait(); waitErr != nil {
		exitErr := &exec.ExitError{}
		if !errors.As(waitErr, &exitErr) {
			t.Fatalf("%v: %v", argv, waitErr)
		}

		exit = exitErr.ExitCode()
	}

	res := textRun{stdout: stdoutBuf.String(), stderr: stderrBuf.String(), exit: exit}

	if ride.transcriptPath != "" {
		res.lines, res.linesErr = e2eLoadTranscript(ride.transcriptPath)
	}

	return res
}

// ompTextInboundMethods returns the inbound request methods in transcript
// order (replies carry no method and are skipped).
func ompTextInboundMethods(t *testing.T, lines []e2eTranscriptLine) []string {
	t.Helper()

	methods := make([]string, 0, len(lines))

	for _, line := range lines {
		if line.dir != "in" {
			continue
		}

		var frame struct {
			Method string `json:"method"`
		}

		if err := json.Unmarshal([]byte(line.json), &frame); err != nil {
			continue
		}

		if frame.Method != "" {
			methods = append(methods, frame.Method)
		}
	}

	return methods
}

// ompTextParams is the decoded generateText params surface.
type ompTextParams struct {
	Workspace struct {
		WorkspacePath string `json:"workspacePath"`
		WorkspaceKey  string `json:"workspaceKey"`
	} `json:"workspace"`
	Selection struct {
		ProviderID string `json:"providerId"`
		ModelID    string `json:"modelId"`
		Options    struct {
			ReasoningLevel string `json:"reasoningLevel"`
		} `json:"options"`
	} `json:"selection"`
	Prompt          string `json:"prompt"`
	QuerySource     string `json:"querySource"`
	MaxOutputTokens int    `json:"maxOutputTokens"`
}

// ompTextOnlyGenerateFrame asserts exactly one inbound generateText frame
// and returns its params in both the typed and raw-key-set forms.
func ompTextOnlyGenerateFrame(
	t *testing.T, lines []e2eTranscriptLine,
) (params ompTextParams, raw map[string]json.RawMessage) {
	t.Helper()

	frameJSON := ""
	count := 0

	for _, line := range lines {
		if line.dir != "in" || !strings.Contains(line.json, `"method":"`+zcode.MethodWorkspaceGenerateText+`"`) {
			continue
		}

		count++
		frameJSON = line.json
	}

	if count != 1 {
		t.Fatalf("%s frames = %d, want exactly 1", zcode.MethodWorkspaceGenerateText, count)
	}

	var frame struct {
		Params json.RawMessage `json:"params"`
	}

	if err := json.Unmarshal([]byte(frameJSON), &frame); err != nil {
		t.Fatalf("decode generateText frame: %v", err)
	}

	if err := json.Unmarshal(frame.Params, &params); err != nil {
		t.Fatalf("decode generateText params: %v", err)
	}

	if err := json.Unmarshal(frame.Params, &raw); err != nil {
		t.Fatalf("decode generateText params key set: %v", err)
	}

	return params, raw
}

// ompTextAssertNoACP asserts the text run never spoke ACP: no initialize and
// no session/* frame in either direction.
func ompTextAssertNoACP(t *testing.T, lines []e2eTranscriptLine) {
	t.Helper()

	for _, line := range lines {
		if strings.Contains(line.json, `"method":"initialize"`) ||
			strings.Contains(line.json, `"method":"session/`) {
			t.Fatalf("text run leaked an ACP frame: %s", line.json)
		}
	}
}

// assertTextStdout asserts the exact stdout contract and the exit class.
func assertTextStdout(t *testing.T, res textRun, wantExit int) {
	t.Helper()

	if res.exit != wantExit {
		t.Fatalf("exit = %d, want %d (stdout %q stderr %q)", res.exit, wantExit, res.stdout, res.stderr)
	}

	if res.exit == 0 && res.stdout != ompTextReply+"\n" {
		t.Fatalf("stdout = %q, want exactly %q", res.stdout, ompTextReply+"\n")
	}

	if strings.Contains(res.stderr, `"jsonrpc"`) {
		t.Fatalf("stderr carries a protocol frame: %q", res.stderr)
	}
}

// TestOmpTextEndToEnd is the built-binary contract proof: Orca's exact argv
// produces exact stdout, exit 0, exactly one workspace/generateText, and
// zero ACP frames; the multi-chunk arm proves BOM-free argv-order prompt
// concatenation on the wire.
func TestOmpTextEndToEnd(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("Orca's exact argv yields exact stdout and one generateText", func(t *testing.T) {
		testOmpTextExactRide(t, agentBin, fakeBin)
	})

	t.Run("multi-chunk @file prompts concatenate in argv order", func(t *testing.T) {
		testOmpTextMultiChunk(t, agentBin, fakeBin)
	})
}

// testOmpTextExactRide drives the exact Orca argv and asserts the whole
// wire/stdout contract.
func testOmpTextExactRide(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextScenario)
	workspace := t.TempDir()
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, installRoot),
		textRide{dir: workspace, transcriptPath: transcriptPath})

	assertTextStdout(t, res, 0)

	if res.linesErr != nil {
		t.Fatalf("fake transcript: %v (stderr %q)", res.linesErr, res.stderr)
	}

	params, raw := ompTextOnlyGenerateFrame(t, res.lines)

	wantWS, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", workspace, err)
	}

	if params.Workspace.WorkspacePath != wantWS || params.Workspace.WorkspaceKey != wantWS {
		t.Fatalf("workspace params = %+v, want path/key %q", params.Workspace, wantWS)
	}

	if params.Selection.ProviderID != "OpenCodeGo" || params.Selection.ModelID != "deepseek-v4.1-flash" {
		t.Fatalf("selection params = %+v, want the --model ids", params.Selection)
	}

	if params.Prompt != ompTextPromptBody {
		t.Fatalf("prompt = %q, want the BOM-free prompt %q", params.Prompt, ompTextPromptBody)
	}

	if params.QuerySource != ompTextQuerySource {
		t.Fatalf("querySource = %q, want %q", params.QuerySource, ompTextQuerySource)
	}

	if params.MaxOutputTokens != ompTextMaxOutputTokens {
		t.Fatalf("maxOutputTokens = %d, want %d (the strict option validator rejects an undefined value)",
			params.MaxOutputTokens, ompTextMaxOutputTokens)
	}

	for _, forbidden := range []string{"tools", "messages"} {
		if _, ok := raw[forbidden]; ok {
			t.Fatalf("params carry a %q key; --no-tools must omit it", forbidden)
		}
	}

	if methods := ompTextInboundMethods(t, res.lines); len(methods) != 1 {
		t.Fatalf("inbound methods = %v, want exactly one generateText", methods)
	}

	ompTextAssertNoACP(t, res.lines)
}

// testOmpTextMultiChunk proves the two-chunk BOM-stripped concatenation on
// the wire.
func testOmpTextMultiChunk(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextScenario)
	promptDir := t.TempDir()
	first := writeTextPrompt(t, promptDir, "prompt-0.txt", ompTextPromptHead)
	second := writeTextPrompt(t, promptDir, "prompt-1.txt", ompTextPromptTail)

	argv := ompTextArgs(ompTextModelPersonal, "", "@"+first, "@"+second)
	argv = append(argv, ompFlagAppServer, installRoot)

	res := ompRunText(t, agentBin, argv, textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	assertTextStdout(t, res, 0)

	if res.linesErr != nil {
		t.Fatalf("fake transcript: %v (stderr %q)", res.linesErr, res.stderr)
	}

	params, _ := ompTextOnlyGenerateFrame(t, res.lines)
	if params.Prompt != ompTextPromptBody {
		t.Fatalf("prompt = %q, want the concatenated %q", params.Prompt, ompTextPromptBody)
	}
}

// TestOmpTextConfigDefault proves the live Orca "Config default" path: with
// no --model, the provider_config defaultModelSelection (idS + reasoning
// level) reaches the generateText params.
func TestOmpTextConfigDefault(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextScenario)
	home := t.TempDir()

	configDir := filepath.Join(home, ".zcode", "v2")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatalf("create provider config dir: %v", err)
	}

	configJSON := `{"config":{"defaultModelSelection":{"providerId":"zai-api",` +
		`"modelId":"GLM-5.3-Flash","options":{"reasoningLevel":"high"}}},"schemaVersion":1}`
	if err := os.WriteFile(filepath.Join(configDir, "provider_config.json"), []byte(configJSON), 0o600); err != nil {
		t.Fatalf("write provider config: %v", err)
	}

	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv("", promptPath, installRoot),
		textRide{home: home, dir: t.TempDir(), transcriptPath: transcriptPath})

	assertTextStdout(t, res, 0)

	if res.linesErr != nil {
		t.Fatalf("fake transcript: %v (stderr %q)", res.linesErr, res.stderr)
	}

	params, _ := ompTextOnlyGenerateFrame(t, res.lines)

	if params.Selection.ProviderID != providerZaiAPI || params.Selection.ModelID != e2eModel {
		t.Fatalf("selection params = %+v, want the provider_config default ids", params.Selection)
	}

	if params.Selection.Options.ReasoningLevel != e2eLevel {
		t.Fatalf("reasoningLevel = %q, want the default selection's %q",
			params.Selection.Options.ReasoningLevel, e2eLevel)
	}
}

// TestOmpTextSelectionRebind is the unit table over the single request
// assembly point: the coding-plan auth mode rebinds zai-api to the account
// provider, while the api-key and none modes leave it untouched.
func TestOmpTextSelectionRebind(t *testing.T) {
	sel := acpfront.ModelSelection{ProviderID: providerZaiAPI, ModelID: e2eModel, ReasoningLevel: e2eLevel}

	cases := []struct {
		name     string
		authMode string
		want     string
	}{
		{"coding plan", acpfront.AuthModeCodingPlan, "account:zai-individual-coding-plan"},
		{"api key", acpfront.AuthModeAPIKey, providerZaiAPI},
		{"none", acpfront.AuthModeNone, providerZaiAPI},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := textRequest(sel, tc.authMode, "prompt", "/ws", "/key")

			if req.Selection.ProviderID != tc.want {
				t.Fatalf("providerId = %q, want %q", req.Selection.ProviderID, tc.want)
			}

			if req.Selection.ModelID != e2eModel || req.Selection.ReasoningLevel != e2eLevel {
				t.Fatalf("model/level lost in the rebind: %+v", req.Selection)
			}

			if req.Prompt != "prompt" || req.WorkspacePath != "/ws" || req.WorkspaceKey != "/key" {
				t.Fatalf("request fields not verbatim: %+v", req)
			}
		})
	}

	t.Run("level-less coding-plan selection gets the account default", func(t *testing.T) {
		levelLess := acpfront.ModelSelection{ProviderID: providerZaiAPI, ModelID: e2eModel}
		req := textRequest(levelLess, acpfront.AuthModeCodingPlan, "prompt", "/ws", "/key")

		if req.Selection.ProviderID != "account:zai-individual-coding-plan" ||
			req.Selection.ReasoningLevel != defaultAccountReasoningLevel {
			t.Fatalf("account fallback = %+v, want provider account:… and level %q",
				req.Selection, defaultAccountReasoningLevel)
		}

		apiKey := textRequest(levelLess, acpfront.AuthModeAPIKey, "prompt", "/ws", "/key")
		if apiKey.Selection.ProviderID != providerZaiAPI || apiKey.Selection.ReasoningLevel != "" {
			t.Fatalf("api-key level-less selection = %+v, want no fabricated level", apiKey.Selection)
		}
	})
}

// TestOmpTextAuth proves the minimal-auth contract on the wire: the personal
// provider emits zero provider/* frames, and the api-key arm runs its
// documented push/refresh chain strictly BEFORE the generateText call.
func TestOmpTextAuth(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("a personal provider emits no auth frames", func(t *testing.T) {
		testOmpTextPersonalAuth(t, agentBin, fakeBin)
	})

	t.Run("the api-key arm pushes and refreshes before the call", func(t *testing.T) {
		testOmpTextAPIKeyAuth(t, agentBin, fakeBin)
	})
}

// testOmpTextPersonalAuth asserts the no-auth arm.
func testOmpTextPersonalAuth(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextScenario)
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, installRoot),
		textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	assertTextStdout(t, res, 0)

	if res.linesErr != nil {
		t.Fatalf("fake transcript: %v (stderr %q)", res.linesErr, res.stderr)
	}

	for _, method := range ompTextInboundMethods(t, res.lines) {
		if strings.HasPrefix(method, "provider/") {
			t.Fatalf("personal-provider arm emitted an auth frame: %s", method)
		}
	}

	ompTextOnlyGenerateFrame(t, res.lines)
}

// testOmpTextAPIKeyAuth asserts the api-key chain and its wire order.
func testOmpTextAPIKeyAuth(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextAPIKeyScenario)
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompModelFlash, promptPath, installRoot),
		textRide{apiKey: ompTextAPIKeyFixture, dir: t.TempDir(), transcriptPath: transcriptPath})

	assertTextStdout(t, res, 0)

	if res.linesErr != nil {
		t.Fatalf("fake transcript: %v (stderr %q)", res.linesErr, res.stderr)
	}

	methods := ompTextInboundMethods(t, res.lines)

	pushIdx := indexOfMethod(methods, zcode.MethodProviderUpdateAccountConfig)
	refreshIdx := indexOfMethod(methods, zcode.MethodProviderRefreshRegistry)
	generateIdx := indexOfMethod(methods, zcode.MethodWorkspaceGenerateText)

	if pushIdx < 0 || refreshIdx < 0 || generateIdx < 0 {
		t.Fatalf("auth chain incomplete: push=%d refresh=%d generate=%d (%v)",
			pushIdx, refreshIdx, generateIdx, methods)
	}

	if pushIdx >= refreshIdx || refreshIdx >= generateIdx {
		t.Fatalf("wire order violated: push(%d) refresh(%d) generate(%d)", pushIdx, refreshIdx, generateIdx)
	}

	params, _ := ompTextOnlyGenerateFrame(t, res.lines)
	if params.Selection.ProviderID != providerZaiAPI {
		t.Fatalf("api-key arm selection = %q, want zai-api (no coding-plan rebind)", params.Selection.ProviderID)
	}
}

// indexOfMethod returns the first index of method in methods, -1 when
// absent.
func indexOfMethod(methods []string, method string) int {
	for i := range methods {
		if methods[i] == method {
			return i
		}
	}

	return -1
}

// TestOmpTextExitCodes covers the documented exit classes: 2 for usage-class
// argv/prompt errors (no backend spawn) and 1 for runtime/generation
// failures.
func TestOmpTextExitCodes(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("a missing prompt exits 2 before any spawn", func(t *testing.T) {
		testOmpTextMissingPrompt(t, agentBin, fakeBin)
	})

	t.Run("an unsupported mode exits 2 before any spawn", func(t *testing.T) {
		testOmpTextBadMode(t, agentBin, fakeBin)
	})

	t.Run("an unreadable @file exits 2 before any spawn", func(t *testing.T) {
		testOmpTextUnreadablePrompt(t, agentBin, fakeBin)
	})

	t.Run("whitespace-only generated text exits 1", func(t *testing.T) {
		testOmpTextEmptyOutput(t, agentBin, fakeBin)
	})

	t.Run("a scripted wire error exits 1 naming the numeric code", func(t *testing.T) {
		testOmpTextWireError(t, agentBin, fakeBin)
	})

	t.Run("a nonexistent install exits 1", func(t *testing.T) {
		testOmpTextNoInstall(t, agentBin)
	})
}

// assertNoSpawn asserts the usage-class failure happened before the backend
// spawn: the fake transcript file was never written.
func assertNoSpawn(t *testing.T, res textRun) {
	t.Helper()

	if res.linesErr == nil {
		t.Fatalf("fake transcript exists for a usage-class failure: %v", res.lines)
	}
}

// testOmpTextMissingPrompt locks the missing-prompt usage class.
func testOmpTextMissingPrompt(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextScenario)
	argv := []string{ompSubcommand, ompFlagPrint, ompFlagMode, ompModeText, ompFlagAppServer, installRoot}

	res := ompRunText(t, agentBin, argv, textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	if res.exit != exitInvalidConfig {
		t.Fatalf("exit = %d, want %d (stderr %q)", res.exit, exitInvalidConfig, res.stderr)
	}

	if !strings.Contains(res.stderr, "no prompt supplied") {
		t.Fatalf("stderr %q must name the missing-prompt sentinel", res.stderr)
	}

	assertNoSpawn(t, res)
}

// testOmpTextBadMode locks the unsupported-mode usage class.
func testOmpTextBadMode(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextScenario)
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	argv := []string{
		ompSubcommand, ompFlagMode, "plan", "@" + promptPath, ompFlagAppServer, installRoot,
	}

	res := ompRunText(t, agentBin, argv, textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	if res.exit != exitInvalidConfig {
		t.Fatalf("exit = %d, want %d (stderr %q)", res.exit, exitInvalidConfig, res.stderr)
	}

	if !strings.Contains(res.stderr, "unsupported --mode") {
		t.Fatalf("stderr %q must name the unsupported-mode sentinel", res.stderr)
	}

	assertNoSpawn(t, res)
}

// testOmpTextUnreadablePrompt locks the unreadable-@file usage class.
func testOmpTextUnreadablePrompt(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, ompTextScenario)
	missing := filepath.Join(t.TempDir(), "absent.txt")

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, missing, installRoot),
		textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	if res.exit != exitInvalidConfig {
		t.Fatalf("exit = %d, want %d (stderr %q)", res.exit, exitInvalidConfig, res.stderr)
	}

	if !strings.Contains(res.stderr, "prompt file is unreadable") {
		t.Fatalf("stderr %q must name the unreadable-prompt sentinel", res.stderr)
	}

	assertNoSpawn(t, res)
}

// testOmpTextEmptyOutput locks the whitespace-only-text failure class.
func testOmpTextEmptyOutput(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, writeTextScenario(t, ompTextEmptyScenarioJSON))
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, installRoot),
		textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	if res.exit != 1 {
		t.Fatalf("exit = %d, want 1 (stdout %q stderr %q)", res.exit, res.stdout, res.stderr)
	}

	if res.stdout != "" {
		t.Fatalf("stdout = %q, want empty on an empty-text failure", res.stdout)
	}

	if !strings.Contains(res.stderr, "returned no text") {
		t.Fatalf("stderr %q must name the empty-text sentinel", res.stderr)
	}
}

// testOmpTextWireError locks the coded wire-error failure class.
func testOmpTextWireError(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	installRoot, transcriptPath := ompTextInstall(t, fakeBin, writeTextScenario(t, ompTextWireErrorScenarioJSON))
	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, installRoot),
		textRide{dir: t.TempDir(), transcriptPath: transcriptPath})

	if res.exit != 1 {
		t.Fatalf("exit = %d, want 1 (stdout %q stderr %q)", res.exit, res.stdout, res.stderr)
	}

	if !strings.Contains(res.stderr, "-32031") {
		t.Fatalf("stderr %q must name the numeric wire code -32031", res.stderr)
	}
}

// testOmpTextNoInstall locks the discovery/spawn failure class.
func testOmpTextNoInstall(t *testing.T, agentBin string) {
	t.Helper()

	promptPath := writeTextPrompt(t, t.TempDir(), "prompt-0.txt", ompTextPromptBody)
	noSuchInstall := filepath.Join(t.TempDir(), "no-such-install")

	res := ompRunText(t, agentBin, ompTextRunArgv(ompTextModelPersonal, promptPath, noSuchInstall),
		textRide{dir: t.TempDir()})

	if res.exit != 1 {
		t.Fatalf("exit = %d, want 1 (stderr %q)", res.exit, res.stderr)
	}

	if !strings.Contains(res.stderr, "backend start failed") {
		t.Fatalf("stderr %q must carry the backend-start failure framing", res.stderr)
	}
}
