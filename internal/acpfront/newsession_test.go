package acpfront_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"
)

// Front-tier session/new contract locks (plan 04-03, CORE-02/D-03/V5): each
// leg composes its own loop over the setmodel-followup corpus (its
// create/subscribe/setModel preamble IS the D-03 wire contract), asserting
// on the RECORDED back-leg fake transcript.
//
//   - TestNewSessionRejectsRelativeCwd is the RED target (T-4-01, RESEARCH
//     Security Domain V5): a RELATIVE cwd must fail with an
//     invalid-params-class error BEFORE any backend call — no session/create
//     frame on the wire. RED baseline: the tracer's Front passes the cwd
//     through unchecked and the create IS attempted.
//   - TestNewSessionAcceptsMcpServers locks D-03 accept-and-ignored: a
//     non-empty mcpServers list succeeds and never reaches the backend wire.
//   - TestNewSessionCreatesAndSetsModel locks the CORE-02 payload sequence:
//     create bare → subscribe desktop-continuous → setModel carrying
//     model.options.reasoningLevel + persistAsWorkspaceLastUsed:false.

// TestNewSessionRejectsRelativeCwd — the relative-cwd rejection (T-4-01):
// the back-leg transcript shows NO session/create frame and the client gets
// an invalid-params-class error naming the requirement (spec session-setup:
// cwd MUST be absolute).
func TestNewSessionRejectsRelativeCwd(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	_, newErr := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: "relative/path"})

	// The transcript assertion runs FIRST (deliberately): at RED the create
	// frame is on the wire and this is the planned failure point; at GREEN
	// the rejection happens before anything touches the back leg.
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if creates := transcriptIndexes(tr, "in", `"method":"session/create"`); len(creates) != 0 {
		t.Fatalf("relative-cwd session/new must be rejected BEFORE any backend call (T-4-01) — "+
			"back leg records %d session/create frame(s)", len(creates))
	}

	// The rejection IS an invalid-params-class JSON-RPC error naming the
	// requirement.
	if newErr == nil {
		t.Fatal("session/new with a relative cwd must fail, got success")
	}

	var rerr *acp.RequestError
	if !errors.As(newErr, &rerr) {
		t.Fatalf("relative-cwd rejection must surface as a coded RequestError, got %T: %v", newErr, newErr)
	}

	if rerr.Code != -32602 {
		t.Fatalf("relative-cwd error code = %d, want -32602 (invalid params class)", rerr.Code)
	}

	if !strings.Contains(rerr.Message, "absolute") {
		t.Fatalf("relative-cwd error message %q must name the requirement (cwd must be absolute)", rerr.Message)
	}
}

// TestNewSessionAcceptsMcpServers — D-03 accept-and-ignored: a non-empty
// mcpServers list (one HTTP entry, name + url shape) succeeds and returns a
// session id, and the backend create frame carries NO mcp fields — the list
// never reaches the backend wire.
func TestNewSessionAcceptsMcpServers(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd: t.TempDir(),
		McpServers: []acp.McpServer{
			{Http: &acp.McpServerHttpInline{Name: "docs-mcp", Url: "https://mcp.example.invalid/s"}},
		},
	})
	if err != nil {
		t.Fatalf("session/new with a non-empty mcpServers list must succeed: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	createIdx := transcriptIndex(t, tr, "in", `"method":"session/create"`, "create request")
	createRaw := tr.Lines[createIdx].JSON

	if strings.Contains(createRaw, "mcp") {
		t.Fatalf("mcp content must never reach the backend wire (D-03 accept-and-ignored): %s", createRaw)
	}
}

// TestNewSessionCreatesAndSetsModel — the CORE-02 lock over the tracer's
// D-03 sequence: session/new with a model configured drives create →
// subscribe → setModel on the back leg in wire order; the setModel IN frame
// carries model.options.reasoningLevel and explicit
// persistAsWorkspaceLastUsed:false (Open Question 3); the create IN frame
// stays BARE — no model object (Pitfall 4 regression lock).
func TestNewSessionCreatesAndSetsModel(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	sid := string(newResp.SessionId)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	createIdx := transcriptIndex(t, tr, "in", `"method":"session/create"`, "create request")
	subIdx := transcriptIndex(t, tr, "in", `"method":"session/subscribe"`, "subscribe request")
	setModelIdx := transcriptIndex(t, tr, "in", `"method":"session/setModel"`, "setModel request")

	if createIdx >= subIdx || subIdx >= setModelIdx {
		t.Fatalf("back-leg wire order violated: create(%d) subscribe(%d) setModel(%d)",
			createIdx, subIdx, setModelIdx)
	}

	assertCreateFrame(t, tr.Lines[createIdx].JSON)      // bare create — no model key (Pitfall 4)
	assertSubscribeFrame(t, tr.Lines[subIdx].JSON, sid) // desktop-continuous, this session
	assertSetModelFrame(t, tr.Lines[setModelIdx].JSON)  // reasoningLevel + persist:false
}
