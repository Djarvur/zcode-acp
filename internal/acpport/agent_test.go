package acpport_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// acpport adapter contract tests (plan 04-02, D-01 seam lock): a scripted
// Handlers fake drives the adapter through the SDK's generated acp.Agent
// surface, and the wire frames it produces are asserted byte-for-byte —
// the honest D-02 initialize report, the -32601 defaults for the eight
// unadvertised stable methods, the stop-reason and error-code mappings,
// the delegation contracts, and the chunk shapes the sink emits.
//
// TestSDKImportBoundary pins the seam itself: no non-test file outside
// internal/acpport may import coder/acp-go-sdk (front-tier and cmd test
// files are exempt — RESEARCH Pattern 6: the SDK's client side is the
// honest counterparty).

// sdkModulePath is the import path prefix the seam lock forbids outside
// internal/acpport (production files).
const sdkModulePath = "github.com/coder/acp-go-sdk"

// frameWaitTimeout bounds each single-frame read in the sink wire test;
// a missing frame fails fast instead of hanging the exchange.
const frameWaitTimeout = 2 * time.Second

// Fixed synthetic ids for the contract tests.
const (
	testSessionID    = "sess_1"
	backendSessionID = "sess_backend"
	testCwd          = "/tmp/ws"
)

// errRawFront is the deliberately non-seam error in the coding table: a
// front failure that is NOT an *acpport.Error must still surface as a
// coded -32603, never raw (A5).
var errRawFront = errors.New("raw front failure")

// TestSDKImportBoundary walks every non-test .go file in the repo and
// fails if any file outside internal/acpport imports the SDK — the D-01
// seam lock, enforced structurally (T-4-SC containment).
func TestSDKImportBoundary(t *testing.T) {
	root := filepath.Join("..", "..")

	var offenders []string

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}

		if rel == "." {
			return nil // the repo root itself
		}

		if d.IsDir() {
			if prunedBoundaryDir(d.Name()) {
				return fs.SkipDir
			}

			return nil
		}

		if !isBoundaryCandidate(d.Name()) || inSeamDir(rel) {
			return nil
		}

		spec, parseErr := sdkImportSpec(path)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", rel, parseErr)
		}

		if spec != "" {
			offenders = append(offenders, rel+" -> "+spec)
		}

		return nil
	})

	if walkErr != nil {
		t.Fatalf("walk repo: %v", walkErr)
	}

	if len(offenders) > 0 {
		t.Fatalf("SDK imports outside internal/acpport (D-01 seam lock):\n%s", strings.Join(offenders, "\n"))
	}
}

// prunedBoundaryDir reports the directories the seam walk never descends
// into: VCS/testdata/vendor trees and hidden dirs.
func prunedBoundaryDir(name string) bool {
	switch name {
	case "testdata", "vendor", "node_modules":
		return true
	}

	return strings.HasPrefix(name, ".")
}

// isBoundaryCandidate reports whether the entry is a production Go file
// (test files are exempt — RESEARCH Pattern 6).
func isBoundaryCandidate(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// inSeamDir reports whether rel lives inside the sanctioned seam package.
func inSeamDir(rel string) bool {
	seam := filepath.Join("internal", "acpport")

	return rel == seam || strings.HasPrefix(rel, seam+string(filepath.Separator))
}

// sdkImportSpec returns the file's SDK import path, or "" when the file
// imports none.
func sdkImportSpec(path string) (string, error) {
	node, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return "", err
	}

	for _, imp := range node.Imports {
		spec, unquoteErr := strconv.Unquote(imp.Path.Value)
		if unquoteErr == nil && strings.HasPrefix(spec, sdkModulePath) {
			return spec, nil
		}
	}

	return "", nil
}

// honestReport builds the honest initialize payload (D-02,
// executed-probe shape): real agentInfo, both advertised auth methods,
// zero-value capabilities.
func honestReport() acpport.InitReport {
	return acpport.InitReport{
		Name:    "zcode-acp",
		Title:   "zcode-acp agent",
		Version: "0.0.0-dev",
		AuthMethods: []acpport.AuthMethodDef{
			{ID: "zai-api-key", Name: "z.ai API key"},
			{ID: "zai-individual-coding-plan", Name: "z.ai Individual Coding Plan"},
		},
	}
}

// honestInitializeFrame is the byte-exact wire shape the executed probe
// against v0.20.0 produced (RESEARCH Code Examples): capabilities are
// empty objects, the agent-variant authMethods carry no type
// discriminator, and no fs/terminal/loadSession surface appears anywhere.
const honestInitializeFrame = `{"agentCapabilities":{"auth":{},"mcpCapabilities":{},` +
	`"promptCapabilities":{},"sessionCapabilities":{}},` +
	`"agentInfo":{"name":"zcode-acp","title":"zcode-acp agent","version":"0.0.0-dev"},` +
	`"authMethods":[{"id":"zai-api-key","name":"z.ai API key"},` +
	`{"id":"zai-individual-coding-plan","name":"z.ai Individual Coding Plan"}],"protocolVersion":1}`

// TestInitializeAssemblesHonestReport drives the adapter's Initialize and
// byte-locks the response frame: protocolVersion 1, agentInfo
// name/title/version, both authMethods in contract order, zero-value
// capabilities, and NO fs/terminal/loadSession/type surface (D-02 —
// honesty by construction; assert on absent features, A7).
func TestInitializeAssemblesHonestReport(t *testing.T) {
	h := newScriptedHandlers()
	h.initReport = honestReport()

	adapter := acpport.NewAgent(h)

	initReq := acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber}

	resp, err := adapter.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if resp.ProtocolVersion != acp.ProtocolVersionNumber {
		t.Fatalf("protocolVersion = %d, want %d", resp.ProtocolVersion, acp.ProtocolVersionNumber)
	}

	assertAgentInfo(t, resp.AgentInfo)
	assertAuthMethods(t, resp.AuthMethods)
	assertHonestFrame(t, resp)
}

// assertAgentInfo checks the agentInfo identity block.
func assertAgentInfo(t *testing.T, got *acp.Implementation) {
	t.Helper()

	want := honestReport()

	if got == nil {
		t.Fatal("agentInfo missing — the honest report carries real identity (D-02)")
	}

	if got.Name != want.Name || got.Version != want.Version {
		t.Fatalf("agentInfo = %+v, want name/version from InitReport", got)
	}

	if got.Title == nil || *got.Title != want.Title {
		t.Fatalf("agentInfo title = %v, want %q", got.Title, want.Title)
	}
}

// assertAuthMethods checks both advertised methods in contract order.
func assertAuthMethods(t *testing.T, got []acp.AuthMethod) {
	t.Helper()

	if len(got) != 2 {
		t.Fatalf("authMethods count = %d, want 2 (both advertised per D-02)", len(got))
	}

	if first := got[0].Agent; first == nil || string(first.Id) != "zai-api-key" || first.Name != "z.ai API key" {
		t.Fatalf("authMethods[0] = %+v, want the zai-api-key agent variant", got[0])
	}

	if second := got[1].Agent; second == nil || string(second.Id) != "zai-individual-coding-plan" {
		t.Fatalf("authMethods[1] = %+v, want the zai-individual-coding-plan agent variant", got[1])
	}
}

// assertHonestFrame byte-locks the marshaled response against the
// executed probe shape and the absent-feature guarantees.
func assertHonestFrame(t *testing.T, resp acp.InitializeResponse) {
	t.Helper()

	got, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal initialize response: %v", err)
	}

	if string(got) != honestInitializeFrame {
		t.Fatalf("initialize frame drift:\n got: %s\nwant: %s", got, honestInitializeFrame)
	}

	for _, banned := range []string{"loadSession", "readTextFile", "writeTextFile", "terminal", `"type"`} {
		if strings.Contains(string(got), banned) {
			t.Fatalf("honest initialize frame must not carry %q:\n%s", banned, got)
		}
	}
}

// TestUnadvertisedMethodsAnswerMethodNotFound drives the eight
// unadvertised stable methods and asserts each answers the clean
// -32601 default naming the method (Pitfall 6) — and that the Handlers
// fake was never touched (the defaults are adapter-side).
func TestUnadvertisedMethodsAnswerMethodNotFound(t *testing.T) {
	h := newScriptedHandlers()
	adapter := acpport.NewAgent(h)
	ctx := context.Background()

	tests := []struct {
		method string
		call   func() error
	}{
		{acp.AgentMethodAuthenticate, func() error {
			_, err := adapter.Authenticate(ctx, acp.AuthenticateRequest{})

			return err
		}},
		{acp.AgentMethodLogout, func() error {
			_, err := adapter.Logout(ctx, acp.LogoutRequest{})

			return err
		}},
		{acp.AgentMethodSessionClose, func() error {
			_, err := adapter.CloseSession(ctx, acp.CloseSessionRequest{})

			return err
		}},
		{acp.AgentMethodSessionDelete, func() error {
			_, err := adapter.DeleteSession(ctx, acp.DeleteSessionRequest{})

			return err
		}},
		{acp.AgentMethodSessionList, func() error {
			_, err := adapter.ListSessions(ctx, acp.ListSessionsRequest{})

			return err
		}},
		{acp.AgentMethodSessionResume, func() error {
			_, err := adapter.ResumeSession(ctx, acp.ResumeSessionRequest{})

			return err
		}},
		{acp.AgentMethodSessionSetConfigOption, func() error {
			_, err := adapter.SetSessionConfigOption(ctx, acp.SetSessionConfigOptionRequest{})

			return err
		}},
		{acp.AgentMethodSessionSetMode, func() error {
			_, err := adapter.SetSessionMode(ctx, acp.SetSessionModeRequest{})

			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			assertMethodNotFound(t, tt.call(), tt.method)
		})
	}

	if calls := h.totalCalls(); calls != 0 {
		t.Fatalf("unadvertised methods reached Handlers %d times, want 0 (defaults are adapter-side)", calls)
	}
}

// assertMethodNotFound asserts one coded -32601 error naming the method.
func assertMethodNotFound(t *testing.T, err error, method string) {
	t.Helper()

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("unadvertised method must answer a coded RequestError, got %v", err)
	}

	if rerr.Code != -32601 {
		t.Fatalf("error code = %d, want -32601 (MethodNotFound)", rerr.Code)
	}

	var data map[string]string
	if err := json.Unmarshal(rerr.Data, &data); err != nil {
		t.Fatalf("decode error data: %v (%s)", err, rerr.Data)
	}

	if data["method"] != method {
		t.Fatalf("error data method = %q, want %q", data["method"], method)
	}
}

// promptJoinProbe is the prompt request the content-join assertions ride:
// two text blocks around one resource_link (Assumption A3 — the link
// contributes its URI to the joined content).
func promptJoinProbe() acp.PromptRequest {
	return acp.PromptRequest{
		SessionId: testSessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Reply"),
			{ResourceLink: &acp.ContentBlockResourceLink{Uri: "file:///x", Name: "x"}},
			acp.TextBlock("OK"),
		},
	}
}

// TestPromptMapsStopReasonsAndCodesErrors drives the adapter's Prompt
// over the scripted fake: StopEndTurn/StopCancelled map onto the SDK stop
// reasons; front errors surface as coded RequestErrors (-32600 seam
// invalid-request, -32603 internal — never raw, never synthetic stop
// reasons, A5); an unmapped stop reason must not leak.
func TestPromptMapsStopReasonsAndCodesErrors(t *testing.T) {
	req := promptJoinProbe()

	t.Run("stop-reason mapping", func(t *testing.T) {
		assertPromptStopReasons(t, req)
	})

	t.Run("content join and delegation", func(t *testing.T) {
		assertPromptContentJoin(t, req)
	})

	t.Run("front errors are coded", func(t *testing.T) {
		assertPromptCodedErrors(t, req)
	})

	t.Run("unmapped stop reason must not leak", func(t *testing.T) {
		assertUnmappedReasonLoud(t, req)
	})
}

// assertPromptStopReasons checks both neutral stop reasons map onto the
// SDK's wire stop reasons.
func assertPromptStopReasons(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	for reason, want := range map[acpport.StopReason]acp.StopReason{
		acpport.StopEndTurn:   acp.StopReasonEndTurn,
		acpport.StopCancelled: acp.StopReasonCancelled,
	} {
		h := newScriptedHandlers()
		h.promptReason = reason

		resp, err := acpport.NewAgent(h).Prompt(context.Background(), req)
		if err != nil {
			t.Fatalf("Prompt(%s): %v", reason, err)
		}

		if resp.StopReason != want {
			t.Fatalf("Prompt(%s) stopReason = %q, want %q", reason, resp.StopReason, want)
		}
	}
}

// assertPromptContentJoin checks the delegation contract: one Handlers
// call with the joined content (text verbatim, resource_link as URI) and
// a non-nil sink.
func assertPromptContentJoin(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	h := newScriptedHandlers()
	h.promptReason = acpport.StopEndTurn

	if _, err := acpport.NewAgent(h).Prompt(context.Background(), req); err != nil {
		t.Fatalf("Prompt: %v", err)
	}

	if len(h.promptCalls) != 1 {
		t.Fatalf("Handlers.Prompt calls = %d, want 1", len(h.promptCalls))
	}

	got := h.promptCalls[0]

	if got.SessionID != testSessionID {
		t.Fatalf("PromptParams.SessionID = %q, want %s", got.SessionID, testSessionID)
	}

	if got.Text != "Reply file:///x OK" {
		t.Fatalf("PromptParams.Text = %q, want joined content %q", got.Text, "Reply file:///x OK")
	}

	if h.sinks[0] == nil {
		t.Fatal("Handlers.Prompt must receive a non-nil update sink")
	}
}

// assertPromptCodedErrors checks the error-coding table: seam
// invalid-request codes -32600, seam and raw internal errors code -32603.
func assertPromptCodedErrors(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	tests := []struct {
		name     string
		frontErr error
		wantCode int
	}{
		{"invalid request", acpport.NewInvalidRequest("a prompt is already running for this session"), -32600},
		{"internal", acpport.NewInternal("backend session send failed"), -32603},
		{"raw error is coded internal", errRawFront, -32603},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newScriptedHandlers()
			h.promptErr = tt.frontErr

			resp, err := acpport.NewAgent(h).Prompt(context.Background(), req)
			if err == nil {
				t.Fatalf("Prompt must fail, got response %+v", resp)
			}

			var rerr *acp.RequestError
			if !errors.As(err, &rerr) {
				t.Fatalf("front error must surface as a coded RequestError, got %T: %v", err, err)
			}

			if rerr.Code != tt.wantCode {
				t.Fatalf("error code = %d, want %d", rerr.Code, tt.wantCode)
			}
		})
	}
}

// assertUnmappedReasonLoud checks a foreign stop reason never becomes a
// result — it surfaces as a coded internal error (A5).
func assertUnmappedReasonLoud(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	h := newScriptedHandlers()
	h.promptReason = acpport.StopReason("refusal")

	resp, err := acpport.NewAgent(h).Prompt(context.Background(), req)
	if err == nil {
		t.Fatalf("unmapped stop reason %q must not become a response, got %+v", h.promptReason, resp)
	}

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) || rerr.Code != -32603 {
		t.Fatalf("unmapped stop reason must surface as -32603, got %v", err)
	}
}

// TestFrontErrorsCodedOnAllAdvertisedMethods pins the coding contract on
// every advertised method: a seam error returned by Handlers surfaces as
// the coded RequestError on the wire, never raw — the SDK's toReqErr
// type-asserts *RequestError, so an untranslated seam error would degrade
// to a generic -32603 with the message string.
func TestFrontErrorsCodedOnAllAdvertisedMethods(t *testing.T) {
	ctx := context.Background()

	seamErr := acpport.NewInvalidRequest("session is not available for this operation")

	t.Run("initialize", func(t *testing.T) {
		h := newScriptedHandlers()
		h.initErr = seamErr

		_, err := acpport.NewAgent(h).Initialize(ctx, acp.InitializeRequest{})
		assertSeamCoded(t, err, seamErr)
	})

	t.Run("new_session", func(t *testing.T) {
		h := newScriptedHandlers()
		h.newSessionErr = seamErr

		_, err := acpport.NewAgent(h).NewSession(ctx, acp.NewSessionRequest{Cwd: testCwd})
		assertSeamCoded(t, err, seamErr)
	})

	t.Run("cancel", func(t *testing.T) {
		h := newScriptedHandlers()
		h.cancelErr = seamErr

		err := acpport.NewAgent(h).Cancel(ctx, acp.CancelNotification{SessionId: testSessionID})
		assertSeamCoded(t, err, seamErr)
	})
}

// assertSeamCoded asserts err is the coded RequestError carrying the seam
// error's code and message.
func assertSeamCoded(t *testing.T, err error, seam *acpport.Error) {
	t.Helper()

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("seam error must surface as a coded RequestError, got %T: %v", err, err)
	}

	if rerr.Code != seam.Code || rerr.Message != seam.Message {
		t.Fatalf("coded error = %d/%q, want %d/%q", rerr.Code, rerr.Message, seam.Code, seam.Message)
	}
}

// TestNewSessionAndCancelDelegate drives the two remaining advertised
// methods over the scripted fake: NewSession returns the front's session
// id and passes cwd + the neutral MCP view through; Cancel forwards the
// session id (D-03/D-05 delegation contracts).
func TestNewSessionAndCancelDelegate(t *testing.T) {
	h := newScriptedHandlers()
	adapter := acpport.NewAgent(h)
	ctx := context.Background()

	req := acp.NewSessionRequest{
		Cwd: testCwd,
		McpServers: []acp.McpServer{
			{Stdio: &acp.McpServerStdio{Name: "mcp-fs", Command: "uvx"}},
		},
	}

	resp, err := adapter.NewSession(ctx, req)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if resp.SessionId != acp.SessionId(h.newSessionID) {
		t.Fatalf("NewSession sessionId = %q, want %q (front id passes through)", resp.SessionId, h.newSessionID)
	}

	if len(h.newSessionCalls) != 1 {
		t.Fatalf("Handlers.NewSession calls = %d, want 1", len(h.newSessionCalls))
	}

	assertNeutralNewSessionParams(t, h.newSessionCalls[0])

	cancelNote := acp.CancelNotification{SessionId: backendSessionID}

	if err := adapter.Cancel(ctx, cancelNote); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	if len(h.cancelCalls) != 1 || h.cancelCalls[0] != backendSessionID {
		t.Fatalf("Handlers.Cancel calls = %v, want exactly [%s]", h.cancelCalls, backendSessionID)
	}
}

// assertNeutralNewSessionParams checks the neutral DTO translation of the
// create request (cwd verbatim, MCP list shallow-copied out of SDK types).
func assertNeutralNewSessionParams(t *testing.T, params acpport.NewSessionParams) {
	t.Helper()

	if params.Cwd != testCwd {
		t.Fatalf("NewSessionParams.Cwd = %q, want %s", params.Cwd, testCwd)
	}

	if len(params.McpServers) != 1 || params.McpServers[0].Kind != "stdio" || params.McpServers[0].Name != "mcp-fs" {
		t.Fatalf("NewSessionParams.McpServers = %+v, want the neutral stdio view of mcp-fs", params.McpServers)
	}
}

// TestUpdateSinkEmitsChunkShapes wires a real agent-side connection and
// asserts the sink's wire shapes byte-for-byte: message notes become
// agent_message_chunk notifications, thought notes become
// agent_thought_chunk notifications, and an unknown note kind fails loud
// without emitting anything (D-04 honesty).
func TestUpdateSinkEmitsChunkShapes(t *testing.T) {
	clientToAgentR, clientToAgentW := io.Pipe()
	agentToClientR, agentToClientW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{clientToAgentR, clientToAgentW, agentToClientR, agentToClientW} {
			_ = c.Close()
		}
	})

	h := newScriptedHandlers()
	conn, _ := acpport.NewAgentSideConnection(acpport.NewAgent(h), agentToClientW, clientToAgentR)
	sink := acpport.NewUpdateSink(conn)

	lines := make(chan string, 16)

	go drainFrames(agentToClientR, lines)

	ctx := context.Background()

	messageNote := acpport.SessionNote{SessionID: testSessionID, Kind: acpport.NoteMessage, Delta: "hello"}

	if err := sink(ctx, messageNote); err != nil {
		t.Fatalf("sink(message): %v", err)
	}

	thoughtNote := acpport.SessionNote{SessionID: testSessionID, Kind: acpport.NoteThought, Delta: "thinking"}

	if err := sink(ctx, thoughtNote); err != nil {
		t.Fatalf("sink(thought): %v", err)
	}

	// The full wire frame: the SDK wraps the notification in the JSON-RPC
	// envelope — the executed-probe shapes (RESEARCH Code Examples) are
	// the params payload inside it.
	wantMessage := `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` + testSessionID +
		`","update":{"content":{"text":"hello","type":"text"},"sessionUpdate":"agent_message_chunk"}}}`
	wantThought := `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` + testSessionID +
		`","update":{"content":{"text":"thinking","type":"text"},"sessionUpdate":"agent_thought_chunk"}}}`

	if got := readFrame(t, lines); got != wantMessage {
		t.Fatalf("message chunk frame drift:\n got: %s\nwant: %s", got, wantMessage)
	}

	if got := readFrame(t, lines); got != wantThought {
		t.Fatalf("thought chunk frame drift:\n got: %s\nwant: %s", got, wantThought)
	}

	err := sink(ctx, acpport.SessionNote{SessionID: testSessionID, Kind: "bogus", Delta: "?"})
	if err == nil {
		t.Fatal("unknown note kind must fail loud, got nil error")
	}

	expectNoFrame(t, lines, "unknown note kind must not emit a frame")
}

// drainFrames reads newline-delimited frames into lines until EOF.
func drainFrames(r io.Reader, lines chan<- string) {
	sc := bufio.NewScanner(r)

	for sc.Scan() {
		lines <- sc.Text()
	}
}

// readFrame waits for one wire frame, failing fast on timeout.
func readFrame(t *testing.T, lines <-chan string) string {
	t.Helper()

	select {
	case line := <-lines:
		return line
	case <-time.After(frameWaitTimeout):
		t.Fatalf("no frame arrived on the agent wire within %v", frameWaitTimeout)

		return ""
	}
}

// expectNoFrame asserts silence: no frame arrives within the wait window.
func expectNoFrame(t *testing.T, lines <-chan string, why string) {
	t.Helper()

	select {
	case line := <-lines:
		t.Fatalf("%s — got: %s", why, line)
	case <-time.After(frameWaitTimeout):
	}
}

// scriptedHandlers is the test-local Handlers fake: it records every call
// and replays canned results — the adapter contract's counterparty.
type scriptedHandlers struct {
	mu sync.Mutex

	initReport    acpport.InitReport
	newSessionID  string
	promptReason  acpport.StopReason
	promptErr     error
	initErr       error
	newSessionErr error
	cancelErr     error

	initCalls       int
	newSessionCalls []acpport.NewSessionParams
	promptCalls     []acpport.PromptParams
	sinks           []acpport.UpdateSink
	cancelCalls     []string
}

// newScriptedHandlers returns the fake with the canned backend session id.
func newScriptedHandlers() *scriptedHandlers {
	return &scriptedHandlers{newSessionID: backendSessionID}
}

// Initialize records the call and replays the canned report.
func (s *scriptedHandlers) Initialize(ctx context.Context) (acpport.InitReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initCalls++

	if s.initErr != nil {
		return acpport.InitReport{}, s.initErr
	}

	return s.initReport, nil
}

// NewSession records the params and replays the canned session id.
func (s *scriptedHandlers) NewSession(ctx context.Context, params acpport.NewSessionParams) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.newSessionCalls = append(s.newSessionCalls, params)

	if s.newSessionErr != nil {
		return "", s.newSessionErr
	}

	return s.newSessionID, nil
}

// Prompt records the params and sink, then answers with the canned reason
// and error.
func (s *scriptedHandlers) Prompt(
	ctx context.Context,
	params acpport.PromptParams,
	sink acpport.UpdateSink,
) (acpport.StopReason, error) {
	s.mu.Lock()
	s.promptCalls = append(s.promptCalls, params)
	s.sinks = append(s.sinks, sink)
	s.mu.Unlock()

	return s.promptReason, s.promptErr
}

// Cancel records the session id.
func (s *scriptedHandlers) Cancel(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelCalls = append(s.cancelCalls, sessionID)

	return s.cancelErr
}

// totalCalls reports the number of Handlers invocations recorded.
func (s *scriptedHandlers) totalCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.initCalls + len(s.newSessionCalls) + len(s.promptCalls) + len(s.cancelCalls)
}
