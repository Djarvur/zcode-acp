package acpfront_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

	// permCancelJoinID is the cancel row's SECOND send reverse (the tombstone
	// join the second prompt parks — it must never earn a reply).
	permCancelJoinID = "fake-3"

	// permAllowReason is the allow_once option's embedded response reason
	// (the corpus row's expect_answer payload).
	permAllowReason = "Approved once"

	// permAllowOptionID is the scripted client selection (the corpus row's
	// allow_once option).
	permAllowOptionID = "allow_once"

	// permCancelledMessage is the bridge's stable -32021 message constant
	// (the cancel row's expect_answer byte-locks it).
	permCancelledMessage = "Client request cancelled: interaction/requestPermission"

	// permDenyReason is the wrapper-synthesized denied-by-user reason
	// (interact.deniedByUserReason's value — byte-locked here).
	permDenyReason = "Permission denied by user"
)

// newPermissionFront composes the full loop over a permission corpus row
// with the Responder wired through the REAL front bridge (interact.
// NewPermissionBridge over the Front's PermissionForward func — the seam
// the production cmd wiring (plan 05-07) composes the same way) and the
// scripted allow-once client outcome. The connection-bound sender binds at
// acpport construction (inside ComposeFront, before any frame flows).
func newPermissionFront(t *testing.T, scenario string) *FrontHarness {
	t.Helper()

	return newPermissionFrontOpts(t, scenario,
		PermissionScript{SelectedOptionID: permAllowOptionID}, nil)
}

// newPermissionFrontOpts is the parameterized composition: a caller-supplied
// client outcome script and extra Responder options (deadline overrides for
// the expiry arms) layered over the forwarder seam.
func newPermissionFrontOpts(
	t *testing.T, scenario string, script PermissionScript, responderOpts []interact.Option,
) *FrontHarness {
	t.Helper()

	sc := mustLoadScenario(t, scenario)

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
		)

		opts := append([]interact.Option{
			interact.WithPermissionForwarder(interact.NewPermissionBridge(front.PermissionForward()).Forward),
		}, responderOpts...)

		responder := interact.NewResponder(opts...)
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	h.rec.SetPermissionScript(script)

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

// permWait bounds each cancel/expiry-stage wait (the tracerWait pattern);
// the composed loop completes in milliseconds and a stuck stage fails fast.
const permWait = 5 * time.Second

// permAnswerHold is the client hold the cancel-race probe scripts: long
// enough that the test's cancel (issued the moment the ask is captured)
// reliably races the answer window, short enough that the client-win arm
// still resolves quickly when it happens.
const permAnswerHold = 20 * time.Millisecond

// permExpiryHold is the unanswered-client hold for the expiry arm — far
// beyond the injected 50 ms permission deadline, bounded by the request
// context at teardown.
const permExpiryHold = 30 * time.Second

// permExpiryDeadline is the injected requestPermission policy deadline the
// expiry arm runs under (the OQ-2 knob — 300 s default, settable).
const permExpiryDeadline = 50 * time.Millisecond

// TestPermissionCancel is the Task 3 cancel lock over the composed harness
// (permission-cancel row): cancel mid-pending → the backend receives the
// -32021 error reply (the bridge's stable message constant, byte-locked by
// the row's expect_answer), the RecordingClient's captured ask is exactly
// one, NO reply lands after the error reply for the same requestId (the
// tombstone join parks silently — reply-once), the pending table drains
// (a second prompt on the session works — no dangling bridge), and the
// turn resolves cancelled as a SUCCESS stopReason.
func TestPermissionCancel(t *testing.T) {
	gate := make(chan struct{})
	h := newPermissionFrontOpts(t, "permission-cancel", PermissionScript{Gate: gate}, nil)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openPermissionSession(ctx, t, h)

	promptDone := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		promptDone <- promptOutcome{resp: resp, err: err}
	}()

	awaitPermissionAsk(t, h)

	// Cancel mid-pending (D-05 + the TOOL-02 fan-in): machine.Cancel rides
	// the bypass lane while CancelPendingForTurn resolves the in-flight
	// bridge → the -32021 error reply.
	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	outcome := waitPermissionOutcome(t, promptDone)

	if outcome.err != nil {
		t.Fatalf("cancelled prompt must answer as a success response, got error: %v", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stopReason = %q, want cancelled", outcome.resp.StopReason)
	}

	// The pending table drains: a SECOND prompt on the session works — its
	// reverse joins the answered set's tombstone (parks silently, the fake
	// records the honest timeout note) and the turn completes end_turn.
	secondResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("second")},
	})
	if err != nil {
		t.Fatalf("second prompt after cancel (no dangling bridge): %v", err)
	}

	if secondResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("second prompt stopReason = %q, want end_turn", secondResp.StopReason)
	}

	tr, _ := h.Transcript(t)
	assertCancelledBackLeg(t, tr)
}

// TestPermissionCancelRace is the concurrency probe: fire the client's
// selection and the session/cancel concurrently — exactly one outcome
// reaches the backend (the selected response OR the -32021 error, never
// both, never neither), the reply-once discipline under -race.
func TestPermissionCancelRace(t *testing.T) {
	h := newPermissionFrontOpts(t, "permission-allow", PermissionScript{
		SelectedOptionID: permAllowOptionID,
		Delay:            permAnswerHold,
	}, nil)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openPermissionSession(ctx, t, h)

	promptDone := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		promptDone <- promptOutcome{resp: resp, err: err}
	}()

	awaitPermissionAsk(t, h)

	// The race: the client's answer window (20 ms) vs the cancel fan-in —
	// whichever lands first owns the single outcome.
	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	outcome := waitPermissionOutcome(t, promptDone)

	// The prompt must resolve cleanly whatever arm won: end_turn (the
	// client's answer completed the turn) or cancelled (the fan-in won) —
	// never a hang, never an error.
	if outcome.err != nil {
		t.Fatalf("race prompt failed: %v", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonEndTurn && outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("race prompt stopReason = %q, want end_turn or cancelled (never neither)", outcome.resp.StopReason)
	}

	tr, _ := h.Transcript(t)
	assertSingleReply(t, tr)
}

// TestPermissionExpiry is the expiry lock (OQ-2 disposition): with the
// responder's permission deadline injected short and the client scripted
// unanswered beyond it, the backend reply resolves as the deny result with
// the denied-by-user reason, the turn continues (the machine still resolves
// — the prompt returns end_turn), and exactly one reply lands for the
// reverse (the late client answer is dropped — reply-once).
func TestPermissionExpiry(t *testing.T) {
	h := newPermissionFrontOpts(t, "permission-allow", PermissionScript{
		SelectedOptionID: permAllowOptionID,
		Delay:            permExpiryHold,
	}, []interact.Option{
		interact.WithMethodDeadline(interact.MethodRequestPermission, permExpiryDeadline),
	})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openPermissionSession(ctx, t, h)

	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	if err != nil {
		t.Fatalf("prompt: %v (the expiry deny must let the turn continue)", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want end_turn (the machine still resolves the turn)", promptResp.StopReason)
	}

	if asks := h.rec.PermissionAsks(); len(asks) != 1 {
		t.Fatalf("client captured %d asks, want exactly 1", len(asks))
	}

	tr, _ := h.Transcript(t)
	assertExpiryDenyReply(t, tr)
}

// awaitPermissionAsk blocks until the client captured its first
// request_permission ask (bounded) — the sync point every cancel/expiry
// stage gates on.
func awaitPermissionAsk(t *testing.T, h *FrontHarness) {
	t.Helper()

	deadline := time.After(permWait)

	for {
		if len(h.rec.PermissionAsks()) > 0 {
			return
		}

		select {
		case <-time.After(2 * time.Millisecond):
		case <-deadline:
			t.Fatal("no request_permission ask captured within the wait bound")
		}
	}
}

// waitPermissionOutcome blocks for the in-flight prompt's resolution.
func waitPermissionOutcome(t *testing.T, done <-chan promptOutcome) promptOutcome {
	t.Helper()

	select {
	case outcome := <-done:
		return outcome
	case <-time.After(permWait):
		t.Fatal("prompt never resolved within the wait bound")

		return promptOutcome{} // unreachable
	}
}

// assertCancelledBackLeg checks the cancel row's recorded conversation: the
// -32021 error reply to the registering wire id carries the bridge's
// stable message, the tombstone join (second send's reverse) earned NO
// reply (reply-once after the error), and the stop-bypass session/stop
// rode the bypass lane.
func assertCancelledBackLeg(t *testing.T, tr fake.Transcript) {
	t.Helper()

	errReplies, joinReplies, stops := 0, 0, 0

	for _, line := range tr.Lines {
		probe := probeLineJSON(line.JSON)

		switch {
		case line.Dir == fake.DirIn && isReplyTo(probe, permReverseID):
			if hasError := probe["error"] != nil; !hasError {
				t.Fatalf("cancel reverse earned a result reply (%s), want the -32021 error", line.JSON)
			}

			errReplies++
			assertCancelledError(t, probe, line.JSON)
		case line.Dir == fake.DirIn && isReplyTo(probe, permCancelJoinID):
			joinReplies++
		case line.Dir == fake.DirIn && frameMethod(probe) == "session/stop":
			stops++
		}
	}

	if errReplies != 1 {
		t.Fatalf("permission reverse earned %d error replies, want exactly 1 (-32021)", errReplies)
	}

	if joinReplies != 0 {
		t.Fatalf("tombstone join %s earned %d replies, want 0 (no reply after the error)",
			permCancelJoinID, joinReplies)
	}

	if stops != 1 {
		t.Fatalf("stop-bypass session/stop recorded %d times, want exactly 1 (CORE-05 cancel path)", stops)
	}
}

// assertCancelledError checks the error reply's code and message bytes
// (the corpus row's expect_answer locks them; the assert re-locks
// semantically).
func assertCancelledError(t *testing.T, probe map[string]any, line string) {
	t.Helper()

	errObj, _ := probe["error"].(map[string]any)

	code, _ := errObj["code"].(float64)
	if code != -32021 {
		t.Fatalf("cancel reply error code = %v, want -32021: %s", code, line)
	}

	msg, _ := errObj["message"].(string)
	if msg != permCancelledMessage {
		t.Fatalf("cancel reply message = %q, want the bridge's stable constant %q", msg, permCancelledMessage)
	}
}

// assertSingleReply checks the race probe's exactly-one discipline: exactly
// one reply envelope addresses the registering wire id, and its shape is
// one of the two race arms — the allow decision or the -32021 error —
// never both, never neither, never anything else. The prompt's own outcome
// is NOT coupled to the reply shape: in the narrow SDK race the client's
// answer can land while the sender's context is already dead, so the
// backend's single outcome is the -32021 while the machine still completes
// the turn through the corpus turn.completed — the invariant the probe
// guards is the ONE reply on the backend wire.
func assertSingleReply(t *testing.T, tr fake.Transcript) {
	t.Helper()

	replies := 0

	for _, line := range tr.Lines {
		probe := probeLineJSON(line.JSON)

		if line.Dir != fake.DirIn || !isReplyTo(probe, permReverseID) {
			continue
		}

		replies++

		switch {
		case probe["error"] != nil:
			errObj, _ := probe["error"].(map[string]any)
			code, _ := errObj["code"].(float64)

			if code != -32021 {
				t.Fatalf("race reply error code = %v, want -32021: %s", code, line.JSON)
			}
		case probe["result"] != nil:
			result, _ := probe["result"].(map[string]any)

			if result["decision"] != "allow" {
				t.Fatalf("race reply result = %s, want the allow decision", line.JSON)
			}
		default:
			t.Fatalf("race reply carries neither result nor error: %s", line.JSON)
		}
	}

	if replies != 1 {
		t.Fatalf("race produced %d reply envelopes for the reverse, want exactly 1 (never both, never neither)",
			replies)
	}
}

// assertExpiryDenyReply checks the expiry lock: exactly one reply for the
// reverse carrying the deny decision WITH the denied-by-user reason, and
// no second reply after it (the late client answer is dropped).
func assertExpiryDenyReply(t *testing.T, tr fake.Transcript) {
	t.Helper()

	replies := 0

	for _, line := range tr.Lines {
		probe := probeLineJSON(line.JSON)

		if line.Dir != fake.DirIn || !isReplyTo(probe, permReverseID) {
			continue
		}

		replies++

		result, _ := probe["result"].(map[string]any)

		if result == nil || result["decision"] != "deny" || result["reason"] != permDenyReason {
			t.Fatalf("expiry reply = %s, want {decision:deny, reason:%s}", line.JSON, permDenyReason)
		}
	}

	if replies != 1 {
		t.Fatalf("expiry produced %d reply envelopes, want exactly 1 (reply-once)", replies)
	}
}

// probeLineJSON best-effort decodes one transcript line; note lines decode
// nil.
func probeLineJSON(line string) map[string]any {
	var probe map[string]any
	if err := json.Unmarshal([]byte(line), &probe); err != nil {
		return nil
	}

	return probe
}

// isReplyTo reports whether the decoded frame is a reply envelope (id, no
// method, result or error payload) addressed to the given wire id.
func isReplyTo(probe map[string]any, id string) bool {
	frameID, _ := probe["id"].(string)

	if _, hasMethod := probe["method"]; hasMethod || frameID == "" || frameID != id {
		return false
	}

	_, hasResult := probe["result"]
	_, hasErr := probe["error"]

	return hasResult || hasErr
}

// frameMethod digs the decoded frame's method string ("" when absent).
func frameMethod(probe map[string]any) string {
	m, _ := probe["method"].(string)

	return m
}
