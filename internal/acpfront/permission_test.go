package acpfront_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The TOOL-02 permission bridge locks (plan 05-02 tracer): one full
// round-trip — backend interaction/requestPermission → the Responder's
// PermissionForwarder seam → the front bridge → the connection-bound
// acpport sender → the ACP client's session/request_permission → the
// client's scripted outcome → the option's own embedded response back to
// the backend — asserted on recorded wire artifacts (the RecordingClient's
// captured asks, the raw ACP tee, the fake transcript), never internals.

// permissionFixtures: the corpus row's canonical values (the fake's
// deterministic ids: fake-1 goes to the create rule's runtime-preferences
// reverse; fake-2 is the permission reverse).
const (
	permBackendSessionID = "sess_fix1"
	permToolCallID       = "tc1"
	permToolName         = "Bash"
	permRequestID        = "perm-1"
	permReverseID        = "fake-2"

	// permAllowReason is the allow_once option's embedded response reason
	// (the corpus row's expect_answer payload).
	permAllowReason = "Approved once"

	// permAllowOptionID is the scripted client selection (the corpus row's
	// allow_once option).
	permAllowOptionID = "allow_once"
)

// newPermissionFront composes the full loop over a permission corpus row
// with the Responder wired through the REAL front bridge (interact.
// NewPermissionBridge over the Front's PermissionForward func — the seam
// the production cmd wiring (plan 05-07) composes the same way). The
// connection-bound sender binds at acpport construction (inside
// ComposeFront, before any frame flows).
func newPermissionFront(t *testing.T, scenario string) *FrontHarness {
	t.Helper()

	sc := mustLoadScenario(t, scenario)

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
		)

		responder := interact.NewResponder(interact.WithPermissionForwarder(
			interact.NewPermissionBridge(front.PermissionForward()).Forward,
		))
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	h.rec.SetPermissionScript(PermissionScript{SelectedOptionID: permAllowOptionID})

	return h
}

// TestPermissionBridge is the Task 2 RED target: the orchestrator-pinned
// tracer — request arrives → policy answers → client sees effect →
// backend proceeds, over the permission-allow corpus row.
func TestPermissionBridge(t *testing.T) {
	h := newPermissionFront(t, "permission-allow")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openPermissionSession(ctx, t, h)

	// The prompt runs the scripted turn mid-permission-ask and must return
	// end_turn: the client's allow decision forwarded to the backend lets
	// the turn complete (the corpus row emits turn.completed after the
	// reverse answers).
	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want end_turn (the allow decision must complete the turn)",
			promptResp.StopReason)
	}

	// Stage (a): the client captured exactly one session/request_permission
	// carrying the MAPPED options verbatim and the backend toolCallId.
	assertCapturedPermissionAsk(t, h)

	// Stage (c): the request_permission frame is on the ACP tee mid-turn —
	// BEFORE the prompt response (the client sees the ask while the turn
	// is still running).
	assertPermissionFrameMidTurn(t, h)

	// Stage (b): the back leg's expect_answer is satisfied — the forwarded
	// decision answered the reverse with the option's embedded response
	// ({decision:allow, reason:Approved once}).
	assertPermissionBackLeg(t, h, sid)
}

// openPermissionSession opens one session over the composed loop.
func openPermissionSession(ctx context.Context, t *testing.T, h *FrontHarness) string {
	t.Helper()

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	return string(newResp.SessionId)
}

// assertCapturedPermissionAsk checks stage (a): exactly one captured ask,
// options mapped per the locked table (allow_once→allow_once, deny→
// reject_once; names and optionIds verbatim), toolCall carrying the
// backend toolCallId, session id the backend's.
func assertCapturedPermissionAsk(t *testing.T, h *FrontHarness) {
	t.Helper()

	asks := h.rec.PermissionAsks()
	if len(asks) != 1 {
		t.Fatalf("client captured %d session/request_permission asks, want exactly 1", len(asks))
	}

	ask := asks[0]

	if ask.SessionId != acp.SessionId(permBackendSessionID) {
		t.Fatalf("ask sessionId = %q, want %q (the backend session id rides the ask)",
			ask.SessionId, permBackendSessionID)
	}

	if ask.ToolCall.ToolCallId != acp.ToolCallId(permToolCallID) {
		t.Fatalf("ask toolCall.toolCallId = %q, want %q (the backend toolCallId must carry)",
			ask.ToolCall.ToolCallId, permToolCallID)
	}

	want := []struct {
		kind, name, optionID string
	}{
		{"allow_once", "Allow once", permAllowOptionID},
		{"reject_once", "Deny", "deny"},
	}

	if len(ask.Options) != len(want) {
		t.Fatalf("ask options = %d entries, want %d (mapped per the locked table, in backend order)",
			len(ask.Options), len(want))
	}

	for i, w := range want {
		got := ask.Options[i]

		if string(got.Kind) != w.kind || got.Name != w.name || string(got.OptionId) != w.optionID {
			t.Fatalf("ask options[%d] = kind %q name %q optionId %q, want kind %q name %q optionId %q",
				i, got.Kind, got.Name, got.OptionId, w.kind, w.name, w.optionID)
		}
	}
}

// assertPermissionFrameMidTurn checks stage (c) on the raw ACP tee: the
// session/request_permission frame (agent→client, "in") precedes the
// prompt response frame — the client decides while the turn is live.
func assertPermissionFrameMidTurn(t *testing.T, h *FrontHarness) {
	t.Helper()

	lines := h.ACPLines()

	permIdx := teeLineIndex(t, lines, "in", `"session/request_permission"`, "request_permission frame")
	respIdx := teeLineIndex(t, lines, "in", `"end_turn"`, "prompt response")

	if permIdx >= respIdx {
		t.Fatalf("request_permission frame (line %d) not before the prompt response (line %d) — "+
			"the client must see the ask mid-turn", permIdx, respIdx)
	}
}

// assertPermissionBackLeg checks stage (b) on the fake transcript: the
// run is clean (expect_answer satisfied) and exactly one reply envelope
// addresses the registering wire id carrying the embedded response bytes
// ({decision:allow, reason:Approved once}).
func assertPermissionBackLeg(t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v (the back-leg expect_answer must be satisfied)", trErr)
	}

	assertPermissionReply(t, tr)
	assertPermissionFlow(t, tr, sid)
}

// assertPermissionReply scans the transcript's reply envelopes: exactly one
// reply addresses the registering wire id and carries the embedded
// response bytes ({decision:allow, reason:Approved once}); no ERROR reply
// exists for the reverse at all.
func assertPermissionReply(t *testing.T, tr fake.Transcript) {
	t.Helper()

	replies := 0

	for _, line := range tr.Lines {
		if line.Dir != fake.DirIn {
			continue
		}

		var frame struct {
			ID     string `json:"id"`
			Method string `json:"method"`
			Result struct {
				Decision string `json:"decision"`
				Reason   string `json:"reason"`
			} `json:"result"`
			Error json.RawMessage `json:"error"`
		}

		if err := json.Unmarshal([]byte(line.JSON), &frame); err != nil {
			continue
		}

		if frame.Method != "" || frame.ID == "" {
			continue
		}

		if frame.Error != nil {
			t.Fatalf("the permission reverse earned an ERROR reply (the allow decision must forward "+
				"as the embedded response): %s", line.JSON)
		}

		if frame.ID != permReverseID {
			continue
		}

		replies++

		if frame.Result.Decision != "allow" || frame.Result.Reason != permAllowReason {
			t.Fatalf("reply to %s = %s, want the embedded response {decision:allow, reason:%s}",
				permReverseID, line.JSON, permAllowReason)
		}
	}

	if replies != 1 {
		t.Fatalf("permission reverse %s earned %d reply envelopes, want exactly 1", permReverseID, replies)
	}
}

// assertPermissionFlow pins the scripted flow on the transcript: one send,
// the tool.updated scheduled event, and exactly one reverse envelope
// carrying the corpus requestId.
func assertPermissionFlow(t *testing.T, tr fake.Transcript, sid string) {
	t.Helper()

	// The mapped options crossed the back leg verbatim inside the reverse
	// params (the corpus row's own scripted options — the golden locks the
	// bytes; here the scripted correlation keys pin the flow).
	if len(transcriptIndexes(tr, "in", `"method":"session/send"`)) != 1 {
		t.Fatalf("back leg must record exactly one session/send (session %s)", sid)
	}

	if len(transcriptIndexes(tr, string(fake.DirOut), `"type":"tool.updated"`)) != 1 {
		t.Fatalf("back leg must record the tool.updated scheduled event (toolCallId %s)", permToolCallID)
	}

	if len(transcriptIndexes(tr, string(fake.DirOut), permRequestID)) != 1 {
		t.Fatalf("back leg must record exactly one reverse envelope carrying requestId %s", permRequestID)
	}
}
