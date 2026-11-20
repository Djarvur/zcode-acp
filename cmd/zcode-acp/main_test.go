package main

// TestAgentBinaryEndToEnd (plan 04-05): the BUILT zcode-acp binary carries
// the full minimal chat loop over REAL stdio — exec pipes, not io.Pipe —
// proven in CI without a desktop bundle (T-3-13's front-leg twin):
//
//	an SDK ACP client spawns the binary with --app-server pointing at the
//	fake-scenario install shape the discovery Override seam accepts (the
//	production discovery order runs; the override wins step 1)
//	  → initialize (agentInfo.version = the binary's version var, D-02)
//	  → session/new (the model selection rides --model/--reasoning-level;
//	    the setModel payload is asserted on the FAKE's own transcript)
//	  → loop 1: streamed prompt → end_turn with the corpus deltas
//	  → loop 2: prompt → cancel mid-turn (synced on the streamed delta)
//	    → cancelled exactly once as a SUCCESS
//	  → client disconnect → staged shutdown → clean exit 0
//
// Both binaries are built once per test run into the run's temp dir (the
// 03-09 precedent: one build per parent-test invocation — a process-global
// sync.Once cache would point at a deleted TempDir under -count repeats).

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
)

// Ride deadlines. The builds dominate the wall clock; the protocol exchanges
// and the staged shutdown complete in seconds.
const (
	e2eBuildDeadline  = 180 * time.Second
	e2eCycleDeadline  = 60 * time.Second
	e2eLoopWait       = 15 * time.Second
	e2eUpdateWait     = 10 * time.Second
	e2eExitWatchdog   = 15 * time.Second
	e2eTranscriptWait = 10 * time.Second

	// corpus row: the two-loop scenario (loop 1 completes end_turn; loop 2
	// stays active until the stop's cancelled patch) — the SAME fixture the
	// golden corpus pins (path relative to this test's dir).
	e2eScenario = "../../internal/zcode/fake/testdata/scenarios/complete-then-cancel.json"

	// The model selection the --model/--reasoning-level flags carry; the
	// fake-transcript assert locks the setModel payload verbatim.
	e2eProvider = "zai-api"
	e2eModel    = "GLM-5.3-Flash"
	e2eLevel    = "high"
)

// Transcript-parsing sentinels (err113): the poll treats any of them as
// "not ready yet".
var (
	errTranscriptRecord = errors.New("transcript record without a direction tag")
	errTranscriptEmpty  = errors.New("transcript file is empty")
)

// TestAgentBinaryEndToEnd runs the built binary through the full loop.
func TestAgentBinaryEndToEnd(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("version flag unchanged", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), e2eUpdateWait)
		defer cancel()

		out, err := exec.CommandContext(ctx, agentBin, "--version").Output()
		if err != nil {
			t.Fatalf("--version: %v", err)
		}

		if got := strings.TrimSpace(string(out)); got != "zcode-acp "+version {
			t.Fatalf("--version = %q, want %q", got, "zcode-acp "+version)
		}
	})

	t.Run("two prompt loops over real stdio", func(t *testing.T) {
		e2eRunTwoLoopRide(t, agentBin, fakeBin)
	})
}

// e2eBuildBinary builds one package's binary once per parent-test invocation
// into the run's temp dir (the built-binary fallback — never an in-process
// substitute and never the desktop bundle at the E2E tier).
func e2eBuildBinary(t *testing.T, pkg, name string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), e2eBuildDeadline)
	defer cancel()

	binPath := filepath.Join(t.TempDir(), name)

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, pkg)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("go build %s: %v", pkg, err)
	}

	return binPath
}

// e2eRunTwoLoopRide drives the spawned binary through both prompt loops and
// the clean staged shutdown, then asserts the fake's transcript.
func e2eRunTwoLoopRide(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(e2eScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eScenario, err)
	}

	transcriptPath := filepath.Join(t.TempDir(), "fake-transcript.txt")
	installRoot := t.TempDir()
	e2eWriteFakeInstallRoot(t, installRoot, fakeBin, scenarioAbs, transcriptPath)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin,
		"--app-server", installRoot,
		"--model", e2eProvider+"/"+e2eModel,
		"--reasoning-level", e2eLevel)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // own group: the cleanup can take the whole tree down

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	stdin, err := wrapper.StdinPipe()
	if err != nil {
		t.Fatalf("wrapper stdin pipe: %v", err)
	}

	stdout, err := wrapper.StdoutPipe()
	if err != nil {
		t.Fatalf("wrapper stdout pipe: %v", err)
	}

	if err := wrapper.Start(); err != nil {
		t.Fatalf("spawn %s: %v", agentBin, err)
	}

	t.Cleanup(e2eKillGroup(t, wrapper, stderrBuf))

	// The test client rides the wrapper's exec pipes — the production
	// transport (stdout-is-protocol is proven by construction: any stray
	// stdout byte would desync this parser).
	updates := &e2eCollector{}
	client := acp.NewClientSideConnection(updates, stdin, stdout)

	ctx := rideCtx

	e2eInitialize(ctx, t, client)

	sid1 := e2eNewSession(ctx, t, client)
	e2eLoopOne(ctx, t, client, sid1)

	sid2 := e2eNewSession(ctx, t, client)
	e2eLoopTwo(ctx, t, client, updates, sid2)

	// Clean shutdown: the client side closes stdin — the wrapper's SDK conn
	// Done fires, the staged shutdown takes the fake down, exit 0.
	if err := stdin.Close(); err != nil {
		t.Logf("close wrapper stdin: %v", err)
	}

	e2eWaitExit(t, wrapper)
	e2eAssertFakeTranscript(t, transcriptPath)
}

// e2eInitialize runs the initialize leg and asserts the agent identity: the
// version var IS the agentInfo version source (D-02 real agentInfo).
func e2eInitialize(ctx context.Context, t *testing.T, client *acp.ClientSideConnection) {
	t.Helper()

	resp, err := client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}

	if resp.ProtocolVersion != 1 {
		t.Fatalf("protocolVersion = %d, want 1", resp.ProtocolVersion)
	}

	if resp.AgentInfo == nil || resp.AgentInfo.Name != "zcode-acp" {
		t.Fatalf("agentInfo = %+v, want name zcode-acp", resp.AgentInfo)
	}

	if resp.AgentInfo.Version != version {
		t.Fatalf("agentInfo.version = %q, want the binary's version var %q", resp.AgentInfo.Version, version)
	}
}

// e2eNewSession opens one session over the spawned binary.
func e2eNewSession(ctx context.Context, t *testing.T, client *acp.ClientSideConnection) string {
	t.Helper()

	resp, err := client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if resp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	return string(resp.SessionId)
}

// e2eLoopOne drives the completed loop: streamed text deltas then end_turn.
func e2eLoopOne(ctx context.Context, t *testing.T, client *acp.ClientSideConnection, sid string) {
	t.Helper()

	resp, err := client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("loop one")},
	})
	if err != nil {
		t.Fatalf("loop 1 prompt: %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("loop 1 stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
	}
}

// e2eLoopTwo drives the cancelled loop: prompt in flight (synced on the
// streamed slow delta — the tap proves the turn is running), cancel
// mid-turn, cancelled exactly once as a SUCCESS response (never a -32800
// error — the SDK trap the front's turn-ctx ownership exists for).
func e2eLoopTwo(
	ctx context.Context, t *testing.T, client *acp.ClientSideConnection, updates *e2eCollector, sid string,
) {
	t.Helper()

	promptDone := make(chan acp.PromptResponse, 1)
	promptErr := make(chan error, 1)

	go func() {
		resp, err := client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("loop two")},
		})
		if err != nil {
			promptErr <- err

			return
		}

		promptDone <- resp
	}()

	e2eWaitUpdateText(t, updates, "slow") // loop 2's scripted delta — the turn is in flight

	if err := client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	select {
	case resp := <-promptDone:
		if resp.StopReason != acp.StopReasonCancelled {
			t.Fatalf("loop 2 stopReason = %q, want %q", resp.StopReason, acp.StopReasonCancelled)
		}
	case err := <-promptErr:
		t.Fatalf("cancelled loop must answer as a success response, got error: %v", err)
	case <-time.After(e2eLoopWait):
		t.Fatal("cancelled loop never resolved within the wait bound")
	}
}

// e2eWaitUpdateText polls the test client's collected updates for one
// carrying want — the wire-level sync signal that the scripted turn is
// streaming (chunk frames cross the real binary boundary only when the tap
// is live and the turn is running).
func e2eWaitUpdateText(t *testing.T, updates *e2eCollector, want string) {
	t.Helper()

	deadline := time.After(e2eUpdateWait)

	for {
		if updates.hasUpdateText(want) {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("update text %q never arrived; collected %d updates", want, len(updates.DrainUpdates()))
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// e2eWaitExit waits for the wrapper's clean exit after the client
// disconnected (the staged shutdown path — exit 0).
func e2eWaitExit(t *testing.T, wrapper *exec.Cmd) {
	t.Helper()

	waitCh := make(chan error, 1)

	go func() { waitCh <- wrapper.Wait() }()

	select {
	case err := <-waitCh:
		if err != nil {
			t.Fatalf("wrapper exit = %v, want clean exit 0 on client disconnect", err)
		}
	case <-time.After(e2eExitWatchdog):
		t.Fatal("wrapper did not exit after the client disconnected")
	}
}

// e2eWriteFakeInstallRoot fabricates the fake-scenario install shape the
// Override seam accepts: the invariant spawn surface (<root>/zcode +
// <root>/resources/glm/zcode.cjs, install-discovery.md :38) where the
// "zcode" binary is a shim exec-ing the built fake binary with its scenario
// (and transcript sink) baked in — the production discovery order and the
// canonical SpawnArgv line run against it unchanged.
func e2eWriteFakeInstallRoot(t *testing.T, root, fakeBin, scenarioAbs, transcriptPath string) {
	t.Helper()

	glmDir := filepath.Join(root, "resources", "glm")
	if err := os.MkdirAll(glmDir, 0o750); err != nil {
		t.Fatalf("create install tree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(glmDir, "zcode.cjs"), []byte("\n"), 0o640); err != nil {
		t.Fatalf("create entrypoint stub: %v", err)
	}

	shim := "#!/bin/sh\nexec " + fakeBin + " --scenario " + scenarioAbs +
		" --transcript " + transcriptPath + " \"$@\"\n"

	if err := os.WriteFile(filepath.Join(root, "zcode"), []byte(shim), 0o750); err != nil {
		t.Fatalf("create shim binary: %v", err)
	}
}

// e2eTranscriptScan is the classified transcript summary the asserts read.
type e2eTranscriptScan struct {
	createIdx, subIdx, setModelIdx int
	setModelJSON                   string
	sends, stops                   int
}

// e2eAssertFakeTranscript loads the fake's transcript file (written at its
// EOF-drain exit — the binary hosting mode's window into its own wire) and
// asserts the D-03 model contract rode the REAL spawn: the setModel payload
// verbatim from the --model/--reasoning-level flags, the preamble wire
// order, exactly the two prompt sends, and the one bypass stop.
func e2eAssertFakeTranscript(t *testing.T, path string) {
	t.Helper()

	lines := e2eWaitTranscript(t, path)
	scan := e2eScanTranscript(t, lines)

	if scan.createIdx < 0 || scan.subIdx < 0 || scan.setModelIdx < 0 {
		t.Fatalf("fake transcript misses the preamble: create(%d) subscribe(%d) setModel(%d)",
			scan.createIdx, scan.subIdx, scan.setModelIdx)
	}

	if scan.createIdx >= scan.subIdx || scan.subIdx >= scan.setModelIdx {
		t.Fatalf("back-leg wire order violated: create(%d) subscribe(%d) setModel(%d)",
			scan.createIdx, scan.subIdx, scan.setModelIdx)
	}

	if scan.sends != 2 {
		t.Fatalf("fake transcript records %d session/send frames, want exactly 2 (two loops)", scan.sends)
	}

	if scan.stops != 1 {
		t.Fatalf("fake transcript records %d session/stop frames, want exactly 1 (the loop-2 bypass)",
			scan.stops)
	}
}

// e2eWaitTranscript polls for the transcript file until the fake's
// EOF-drain exit lands it (bounded).
func e2eWaitTranscript(t *testing.T, path string) []e2eTranscriptLine {
	t.Helper()

	deadline := time.After(e2eTranscriptWait)

	for {
		loaded, err := e2eLoadTranscript(path)
		if err == nil {
			return loaded
		}

		select {
		case <-deadline:
			t.Fatalf("fake transcript never appeared at %s: %v", path, err)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// e2eScanTranscript classifies the transcript lines and asserts the
// setModel payload at its wire position.
func e2eScanTranscript(t *testing.T, lines []e2eTranscriptLine) e2eTranscriptScan {
	t.Helper()

	scan := e2eTranscriptScan{createIdx: -1, subIdx: -1, setModelIdx: -1}

	for i, line := range lines {
		switch {
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/create"`):
			scan.createIdx = i
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/subscribe"`):
			scan.subIdx = i
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/setModel"`):
			scan.setModelIdx = i
			scan.setModelJSON = line.json
			e2eAssertSetModelPayload(t, line.json)
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/send"`):
			scan.sends++
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/stop"`):
			scan.stops++
		}
	}

	return scan
}

// e2eAssertSetModelPayload decodes the setModel request frame and asserts
// the payload verbatim: the --model/--reasoning-level flags landed as
// model.options.reasoningLevel (create drops it — the refusal trap), with
// explicit persistAsWorkspaceLastUsed:false.
func e2eAssertSetModelPayload(t *testing.T, raw string) {
	t.Helper()

	var frame struct {
		Params struct {
			Model struct {
				ProviderID string `json:"providerId"`
				ModelID    string `json:"modelId"`
				Options    struct {
					ReasoningLevel string `json:"reasoningLevel"`
				} `json:"options"`
			} `json:"model"`
			PersistAsWorkspaceLastUsed *bool `json:"persistAsWorkspaceLastUsed"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode setModel frame: %v", err)
	}

	model := frame.Params.Model
	if model.ProviderID != e2eProvider || model.ModelID != e2eModel {
		t.Fatalf("setModel model = %s/%s, want %s/%s (the --model flags): %s",
			model.ProviderID, model.ModelID, e2eProvider, e2eModel, raw)
	}

	if model.Options.ReasoningLevel != e2eLevel {
		t.Fatalf("setModel reasoningLevel = %q, want %q (the --reasoning-level flag): %s",
			model.Options.ReasoningLevel, e2eLevel, raw)
	}

	if frame.Params.PersistAsWorkspaceLastUsed == nil || *frame.Params.PersistAsWorkspaceLastUsed {
		t.Fatalf("setModel persistAsWorkspaceLastUsed must be explicitly false: %s", raw)
	}
}

// e2eTranscriptLine is one dir-tagged line of the fake's transcript file
// (the golden format: dir + TAB + raw JSON).
type e2eTranscriptLine struct {
	dir  string
	json string
}

// e2eLoadTranscript reads the transcript file; a missing file (the fake has
// not exited yet) reports the error so the caller can poll.
func e2eLoadTranscript(path string) ([]e2eTranscriptLine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := []e2eTranscriptLine{}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		dir, jsonLine, ok := strings.Cut(scanner.Text(), "\t")
		if !ok {
			return nil, errTranscriptRecord
		}

		lines = append(lines, e2eTranscriptLine{dir: dir, json: jsonLine})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, errTranscriptEmpty
	}

	return lines, nil
}

// e2eKillGroup is the best-effort cleanup: SIGKILL the wrapper's process
// group (it leads its own group — Setpgid above) so an early failure leaves
// neither the wrapper nor its fake child behind. ESRCH is the normal-path
// no-op (the wrapper already exited cleanly).
func e2eKillGroup(t *testing.T, wrapper *exec.Cmd, stderrBuf *e2eSyncBuffer) func() {
	t.Helper()

	return func() {
		if t.Failed() {
			stderrBuf.dump(t)
		}

		if wrapper.Process == nil {
			return
		}

		if err := syscall.Kill(-wrapper.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Logf("kill wrapper group: %v", err)
		}
	}
}

// e2eSyncBuffer is a goroutine-safe stderr capture for the wrapper; dumped
// to the test log only on failure (CI stays quiet on green).
type e2eSyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends to the buffer.
func (b *e2eSyncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

// dump logs the captured stderr.
func (b *e2eSyncBuffer) dump(t *testing.T) {
	t.Helper()

	b.mu.Lock()
	defer b.mu.Unlock()

	t.Logf("wrapper stderr:\n%s", b.buf.String())
}

// e2eCollector collects (sessionId, update) pairs under a mutex — the
// package-main twin of the acpfront harness's RecordingClient, stripped to
// what the ride asserts.
type e2eCollector struct {
	mu      sync.Mutex
	updates []e2eUpdate
}

// e2eUpdate is one collected update.
type e2eUpdate struct {
	sessionID string
	text      string
}

// SessionUpdate collects one streamed update.
func (c *e2eCollector) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	_ = ctx

	text := ""
	switch {
	case params.Update.AgentMessageChunk != nil:
		if block := params.Update.AgentMessageChunk.Content.Text; block != nil {
			text = block.Text
		}
	case params.Update.AgentThoughtChunk != nil:
		if block := params.Update.AgentThoughtChunk.Content.Text; block != nil {
			text = block.Text
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.updates = append(c.updates, e2eUpdate{sessionID: string(params.SessionId), text: text})

	return nil
}

// CompleteElicitation is a no-op (elicitation is Phase 5+).
func (c *e2eCollector) CompleteElicitation(ctx context.Context, params acp.CompleteElicitationNotification) error {
	_, _ = ctx, params

	return nil
}

// CreateElicitation is a no-op (elicitation is Phase 5+).
func (c *e2eCollector) CreateElicitation(
	ctx context.Context,
	params acp.CreateElicitationRequest,
) (acp.CreateElicitationResponse, error) {
	_, _ = ctx, params

	return acp.CreateElicitationResponse{}, nil
}

// ReadTextFile is a no-op (fs surfaces are never advertised).
func (c *e2eCollector) ReadTextFile(
	ctx context.Context,
	params acp.ReadTextFileRequest,
) (acp.ReadTextFileResponse, error) {
	_, _ = ctx, params

	return acp.ReadTextFileResponse{}, nil
}

// WriteTextFile is a no-op (fs surfaces are never advertised).
func (c *e2eCollector) WriteTextFile(
	ctx context.Context,
	params acp.WriteTextFileRequest,
) (acp.WriteTextFileResponse, error) {
	_, _ = ctx, params

	return acp.WriteTextFileResponse{}, nil
}

// RequestPermission is a no-op (permission bridging is Phase 5).
func (c *e2eCollector) RequestPermission(
	ctx context.Context,
	params acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
	_, _ = ctx, params

	return acp.RequestPermissionResponse{}, nil
}

// CreateTerminal is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) CreateTerminal(
	ctx context.Context,
	params acp.CreateTerminalRequest,
) (acp.CreateTerminalResponse, error) {
	_, _ = ctx, params

	return acp.CreateTerminalResponse{}, nil
}

// KillTerminal is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) KillTerminal(
	ctx context.Context,
	params acp.KillTerminalRequest,
) (acp.KillTerminalResponse, error) {
	_, _ = ctx, params

	return acp.KillTerminalResponse{}, nil
}

// TerminalOutput is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) TerminalOutput(
	ctx context.Context,
	params acp.TerminalOutputRequest,
) (acp.TerminalOutputResponse, error) {
	_, _ = ctx, params

	return acp.TerminalOutputResponse{}, nil
}

// ReleaseTerminal is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) ReleaseTerminal(
	ctx context.Context,
	params acp.ReleaseTerminalRequest,
) (acp.ReleaseTerminalResponse, error) {
	_, _ = ctx, params

	return acp.ReleaseTerminalResponse{}, nil
}

// WaitForTerminalExit is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) WaitForTerminalExit(
	ctx context.Context,
	params acp.WaitForTerminalExitRequest,
) (acp.WaitForTerminalExitResponse, error) {
	_, _ = ctx, params

	return acp.WaitForTerminalExitResponse{}, nil
}

// DrainUpdates snapshots the collected updates.
func (c *e2eCollector) DrainUpdates() []e2eUpdate {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]e2eUpdate, len(c.updates))
	copy(out, c.updates)

	return out
}

// hasUpdateText reports whether any collected update carries want.
func (c *e2eCollector) hasUpdateText(want string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, u := range c.updates {
		if u.text == want {
			return true
		}
	}

	return false
}
