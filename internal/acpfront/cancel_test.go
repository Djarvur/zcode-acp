package acpfront_test

// CORE-05 cancel locks (plan 04-05, D-05) — the three cancel-contract tests
// the tracer built the shapes for, named and sharpened:
//
//	TestCancelMidTurn: cancel mid-turn resolves the prompt EXACTLY once
//	  with {"stopReason":"cancelled"} as a SUCCESS result; no -32800
//	  error frame ever appears on the ACP wire (the SDK maps a handler's
//	  context.Canceled to a RequestCancelled ERROR — RESEARCH Pitfall 2);
//	  any post-cancel updates precede the response frame (the spec rule:
//	  updates may follow session/cancel but MUST precede the response).
//	TestSecondPromptRejected: a second prompt while a turn is active
//	  answers ONE clean invalid-request (-32600) error with the stable
//	  concurrent-prompt message — from BOTH guard layers (the front-wide
//	  turn slot and the machine's ErrTurnActive, whose backend-refusal
//	  leg is the numeric -32010 mapping, message text never matched) —
//	  never a hang, never -32603 internal noise, never a second
//	  concurrent turn on the wire.
//	TestCancelAfterCompleteNoOp: a cancel arriving after the prompt
//	  already answered produces no second response and no error frame —
//	  the SDK notification is consumed and the machine's idempotent
//	  Cancel writes nothing (Phase 3: a resolved turn's cancel no-ops).
//
// The tests ride the plan 04-02 harness as composed by plan 04-06
// (newTracerFront) over the corpus rows; the backend-refusal leg needs a
// refusal the front's own slot cannot pre-empt, hence the
// send-refused-32010 corpus row.

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	// turnActiveMessage is the ONE stable rejection surface the
	// second-prompt guard answers with, whatever guard layer fires (the
	// front-wide slot or the machine's ErrTurnActive) — asserted verbatim.
	// The WR-05 scope-accurate text (plan 05-05): the slot is front-wide,
	// and the message says so. The constant change and the lock change land
	// in the same commit (the message is a deliberate stable surface).
	turnActiveMessage = "another prompt is already running (this agent runs one turn at a time)"

	// cancelSettle bounds the absence window that proves
	// cancel-after-complete stays silent: no response frame exists to await,
	// so the settle is the honest quiet before the silence assert.
	cancelSettle = 300 * time.Millisecond
)

// assertNoRequestCancelled scans the whole tee (both directions) for a
// -32800 frame: the SDK maps a handler's context.Canceled to a
// RequestCancelled ERROR response, while the contract requires cancelled as
// the SUCCESS stopReason (Pitfall 2).
func assertNoRequestCancelled(t *testing.T, lines []TeeLine) {
	t.Helper()

	for _, line := range lines {
		if strings.Contains(line.JSON, "-32800") {
			t.Fatalf("found a -32800 frame on the ACP tee (cancelled is a success stopReason): %s", line.JSON)
		}
	}
}

// assertNoAgentRequestCancelled is the scoped D-05 lock for flows with an
// in-flight PERMISSION round-trip: it scans the agent→client direction
// only. Pitfall 2 governs the AGENT's answers (the prompt must never
// surface its cancellation as a RequestCancelled ERROR). The client→agent
// direction is exempt by contract: when the fan-in cancels the displaced
// turn's pending bridge, the session/request_permission round-trip dies
// mid-flight and the client-side SDK answers the cancelled request with
// its OWN -32800 — the honest client-side mirror of the -32021 arm,
// present exactly when the fan-in fires. The blanket scan in
// assertNoRequestCancelled stays for flows with no pending ask, where the
// client never writes one either.
func assertNoAgentRequestCancelled(t *testing.T, lines []TeeLine) {
	t.Helper()

	for _, line := range lines {
		if line.Dir == "in" && strings.Contains(line.JSON, "-32800") {
			t.Fatalf("found an agent→client -32800 frame (cancelled is a success stopReason — "+
				"the agent must never surface its cancellation as a RequestCancelled ERROR): %s", line.JSON)
		}
	}
}

// assertSingleCancelledResponse checks the exactly-once cancelled answer on
// the tee: exactly one prompt response frame carrying stopReason cancelled,
// and every streamed update strictly precedes it (the spec ordering rule).
func assertSingleCancelledResponse(t *testing.T, lines []TeeLine) {
	t.Helper()

	stopIdxs := teeLineIndexes(lines, "in", `"stopReason"`)
	if len(stopIdxs) != 1 {
		t.Fatalf("tee records %d prompt response frames, want exactly 1 (exactly-once)", len(stopIdxs))
	}

	if !strings.Contains(lines[stopIdxs[0]].JSON, `"cancelled"`) {
		t.Fatalf("the prompt response = %s, want stopReason cancelled", lines[stopIdxs[0]].JSON)
	}

	for i, line := range lines {
		if i > stopIdxs[0] && line.Dir == "in" && strings.Contains(line.JSON, `"sessionUpdate"`) {
			t.Fatalf("update frame at %d streams after the prompt response (line %d) — "+
				"post-cancel updates MUST precede the response", i, stopIdxs[0])
		}
	}
}

// assertSingleSend checks the back leg saw exactly one session/send — the
// rejected or cancelled flows never put a second wire turn in flight.
func assertSingleSend(t *testing.T, tr fake.Transcript, what string) {
	t.Helper()

	if sends := len(transcriptIndexes(tr, "in", `"method":"session/send"`)); sends != 1 {
		t.Fatalf("back leg records %d session/send frames, want exactly 1 (%s)", sends, what)
	}
}

// TestCancelMidTurn locks the exactly-once cancelled contract over the
// cancel-mid-turn corpus: prompt starts → turn.started → client cancels →
// exactly ONE success response {"stopReason":"cancelled"}, no -32800
// anywhere on the wire.
func TestCancelMidTurn(t *testing.T) {
	h := newTracerFront(t, "cancel-mid-turn")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openCancelStageSession(ctx, t, h)

	promptDone := startActiveTurn(ctx, t, h, sid)

	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	outcome := waitPromptOutcome(t, promptDone)
	if outcome.err != nil {
		t.Fatalf("cancelled prompt must answer as a success response, got error: %v", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stopReason = %q, want %q", outcome.resp.StopReason, acp.StopReasonCancelled)
	}

	assertSingleCancelledResponse(t, h.ACPLines())
	assertNoRequestCancelled(t, h.ACPLines())

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertSingleSend(t, tr, "the turn ran once")
}

// TestSecondPromptRejected locks the ONE invalid-request rejection surface
// for concurrent prompts (D-05): whatever guard layer fires, the client sees
// the same -32600 class and the same stable message — never a hang, never
// -32603 internal noise, never a second concurrent turn on the wire.
func TestSecondPromptRejected(t *testing.T) {
	t.Run("front turn slot held", testFrontSlotRejection)
	t.Run("backend -32010 refusal mapped", testBackendRefusalRejection)
}

// assertRejectedInvalidRequest checks one prompt error against the stable
// rejection surface: a coded JSON-RPC error in the invalid-request class
// carrying the concurrent-prompt message verbatim.
func assertRejectedInvalidRequest(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("a second prompt while a turn is active must return an error, got success")
	}

	var reqErr *acp.RequestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("rejection error = %T (%v), want a coded JSON-RPC RequestError", err, err)
	}

	if reqErr.Code != acpport.CodeInvalidRequest {
		t.Fatalf("rejection code = %d, want %d (invalid-request class); message %q",
			reqErr.Code, acpport.CodeInvalidRequest, reqErr.Message)
	}

	if reqErr.Message != turnActiveMessage {
		t.Fatalf("rejection message = %q, want the one stable surface %q", reqErr.Message, turnActiveMessage)
	}
}

// testFrontSlotRejection rides the cancel-mid-turn corpus: prompt #1 holds
// the front-wide turn slot mid-turn (the tracer's turn.started sync proves
// the slot is held), prompt #2 for the same session is rejected at the slot
// before any second wire turn.
func testFrontSlotRejection(t *testing.T) {
	h := newTracerFront(t, "cancel-mid-turn")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openCancelStageSession(ctx, t, h)

	promptDone := startActiveTurn(ctx, t, h, sid)

	_, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("second")},
	})
	assertRejectedInvalidRequest(t, err)

	// The displaced prompt still resolves as the cancelled SUCCESS (the SDK
	// cancels it when the second prompt registers — registerPrompt — and the
	// front's fan-in turns that into exactly one idempotent machine.Cancel).
	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	outcome := waitPromptOutcome(t, promptDone)
	if outcome.err != nil || outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("displaced prompt outcome = (%v, %v), want a cancelled success",
			outcome.resp.StopReason, outcome.err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertSingleSend(t, tr, "the rejected prompt never reached the app-server")
}

// testBackendRefusalRejection rides the send-refused-32010 corpus: the front
// slot is FREE but the app-server refuses the send with the dialect §6
// concurrency guard (-32010); the machine maps the numeric code to
// ErrTurnActive (never the message text) and the front must surface the same
// ONE invalid-request rejection the slot path answers — the RED target: the
// internal-class mapping answers -32603 with the wrapped transport noise.
func testBackendRefusalRejection(t *testing.T) {
	h := newTracerFront(t, "send-refused-32010")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openCancelStageSession(ctx, t, h)

	_, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("refused")},
	})
	assertRejectedInvalidRequest(t, err)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertSingleSend(t, tr, "the refused turn made no further wire traffic")
}

// TestCancelAfterCompleteNoOp locks the no-op leg over the setmodel-followup
// corpus: the prompt completes (end_turn), THEN a cancel arrives — no second
// response, no error frame, and the machine's idempotent Cancel writes
// nothing (no session/stop on the back leg).
func TestCancelAfterCompleteNoOp(t *testing.T) {
	h := newTracerFront(t, "setmodel-followup")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openCancelStageSession(ctx, t, h)

	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("complete first")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (the turn completed before the cancel)",
			promptResp.StopReason, acp.StopReasonEndTurn)
	}

	before := inFrameCount(h.ACPLines())

	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel after complete: %v", err)
	}

	// No response frame exists to await — the settle is the honest absence
	// window before asserting the silence.
	time.Sleep(cancelSettle)

	lines := h.ACPLines()
	if got := inFrameCount(lines); got != before {
		t.Fatalf("cancel-after-complete produced %d new agent→client frames, want 0", got-before)
	}

	assertNoRequestCancelled(t, lines)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if stops := len(transcriptIndexes(tr, "in", `"method":"session/stop"`)); stops != 0 {
		t.Fatalf("back leg records %d session/stop frames, want 0 (a resolved turn's cancel writes nothing)", stops)
	}
}

// inFrameCount counts the agent→client frames on a tee snapshot — the side
// the silence asserts read (the client's own cancel notification is an "out"
// frame and must not count).
func inFrameCount(lines []TeeLine) int {
	n := 0
	for _, line := range lines {
		if line.Dir == "in" {
			n++
		}
	}

	return n
}

// newSlotStage composes the slot-gate drive (plan 05-10 WR-04): the
// permission-cancel row over the REAL front bridge, the gate-held client
// script, and the front captured for direct exported-seam Cancel drives.
func newSlotStage(t *testing.T, gate chan struct{}) (*FrontHarness, *acpfront.Front) {
	t.Helper()

	sc := mustLoadScenario(t, "permission-cancel")

	var front *acpfront.Front

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front = acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
		)

		opts := []interact.Option{
			interact.WithPermissionForwarder(interact.NewPermissionBridge(front.PermissionForward()).Forward),
		}

		responder := interact.NewResponder(opts...)
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	h.rec.SetPermissionScript(PermissionScript{Gate: gate})

	return h, front
}

// TestCancelFanInOnlyWhenSlotHeld locks the WR-06 displaced-prompt fan-in
// contract (plan 05-15): when the SDK's second-prompt registration
// displaces an active turn whose permission ask is pending, the fan-in
// goroutine's mine arm cancels the turn AND fans its pending bridge in —
// the back leg observes EXACTLY ONE reply envelope for the registering
// reverse and it is the -32021 cancelled error, and no unanswered-at note
// for that reverse ever rides the transcript (the 05-02 prohibition
// restored: a permission bridge never outlives its turn — on all three
// cancel arms: close (CR-01), ACP cancel (TestPermissionCancel), and this,
// the displacement arm).
//
// ENSHRINING-TEST REWRITE (plan 05-15, WR-06 — legitimacy stated): the
// previous revision of this test locked the DANGLING artifact as the
// expected wire shape — it asserted the surviving ask ("the ask was never
// cancelled by anything in this flow", assertDanglingAskNote) because the
// mine arm at HEAD fired machine.Cancel only. Both the delta review
// (05-REVIEW WR-06) and the verifier (05-VERIFICATION gap 3, truth 7
// FAILED) classified that dangle as a contract violation of the 05-02
// prohibition. This rewrite INVERTS the assertions — from asserting the
// dangle to asserting the fan-in — and weakens nothing: the exactly-once
// cancelled SUCCESS response, the single send, and the no--32800 locks all
// remain. The test fails at HEAD with the genuine missing-fan-in signal
// (no -32021 reply within the bound; the fake's honest unanswered note
// present instead).
//
// Re-homing note: the previous revision's tail drove a direct slot-free
// front.Cancel and asserted the fan-in's NEGATIVE log (the WR-04/CR-01
// slot-gate skip). Post-fix the displacement leaves NO dangling ask, so
// that tail's table is empty and its non-fire assertion would be trivially
// true — the tail is dropped here and the slot-free-skip lock is RE-HOMED
// with full discriminating power (a pending actually present while the
// slot is free) in TestCancelFanInSkipAndScope's slot-free subtest (plan
// 05-15 Task 2). No lock is lost.
//
// Drive: turn 1 parks on the pending ask (the client holds the gate); the
// second prompt's registration displaces it (the SDK cancels turn 1's ctx
// — and the second prompt itself is rejected by the held slot); the fan-in
// goroutine's mine arm (the WR-04 pump-identity re-check under the slot
// lock) cancels the turn and fans the pendings in.
func TestCancelFanInOnlyWhenSlotHeld(t *testing.T) {
	gate := make(chan struct{})

	h, _ := newSlotStage(t, gate)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openPermissionSession(ctx, t, h)

	// Turn 1 parks on the pending ask (the client holds the gate).
	prompt1Done := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		prompt1Done <- promptOutcome{resp: resp, err: err}
	}()

	awaitPermissionAsk(t, h)

	// The SDK displacement: the second prompt's registration cancels turn
	// 1's SDK ctx — the fan-in goroutine machine.Cancels turn 1 (its
	// genuine arm) AND, under the fixed contract, fans its pending bridge
	// in; the second prompt itself is rejected by the held slot (one clean
	// invalid-request).
	_, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("second")},
	})
	assertRejectedInvalidRequest(t, err)

	// The displaced turn resolved cancelled — exactly once, as the SUCCESS
	// stopReason (D-05; the fan-in adds no second reply and no -32800).
	outcome := waitPermissionOutcome(t, prompt1Done)

	if outcome.err != nil || outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("displaced turn outcome = (%v, %v), want a cancelled success",
			outcome.resp.StopReason, outcome.err)
	}

	// Bounded settle before the single transcript snapshot: the fake
	// materializes its transcript only at Serve return (EOF drain —
	// dialect §1), so the back leg cannot be polled live. The bound is
	// keyed to the corpus row's own reverse deadline (timeout_ms 500): by
	// the time it elapses the -32021 reply has landed (the fan-in fired —
	// millisecond scale) or the fake recorded the honest unanswered note
	// (the bridge dangled — the RED signal this test exists to fail on).
	time.Sleep(backLegSettle)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// The contract: the displaced turn's pending bridge fanned in at the
	// displacement boundary — exactly one -32021 reply, no dangling ask.
	assertCancelledFanInBackLeg(t, tr)

	// Exactly-once preserved (D-05): ONE success response with stopReason
	// cancelled, no agent-sent -32800 anywhere (the client-side -32800 for
	// the cancelled permission round-trip is the honest mirror of the
	// cancelled bridge — see assertNoAgentRequestCancelled), exactly one
	// session/send on the back leg (the rejected second prompt never
	// reached the app-server).
	assertSingleCancelledResponse(t, h.ACPLines())
	assertNoAgentRequestCancelled(t, h.ACPLines())
	assertSingleSend(t, tr, "the displaced turn ran once")
}

// safeLogBuffer is a mutex-guarded slog sink (the fan-in log fires on the
// cancel caller's goroutine; the read is the test goroutine — same
// goroutine in this test, but the guard keeps -race honest).
type safeLogBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

// contains reports whether the captured log carries the substring.
func (b *safeLogBuffer) contains(substr string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return strings.Contains(b.buf.String(), substr)
}

// count reports how many captured log LINES carry the substring — the
// discriminator asserts counts, never boolean-contains.
func (b *safeLogBuffer) count(substr string) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	n := 0

	for _, line := range strings.Split(b.buf.String(), "\n") {
		if strings.Contains(line, substr) {
			n++
		}
	}

	return n
}

// backLegSettle bounds the settle before a single transcript snapshot (the
// fake materializes its transcript only at Serve return — the back leg
// cannot be polled live, dialect §1 EOF facts). The bound is keyed to the
// permission-cancel row's reverse deadline (timeout_ms 500): past it, the
// -32021 reply (fan-in fired — millisecond scale) or the honest unanswered
// note (the dangle) is final on the transcript either way.
const backLegSettle = 700 * time.Millisecond

// assertCancelledFanInBackLeg locks the WR-06 fan-in's wire artifact: the
// registering permission reverse earned EXACTLY ONE reply envelope and it
// is the -32021 cancelled error carrying the bridge's stable message, and
// no unanswered-at note for the reverse survives (a dangling ask is the
// contract violation the 05-02 prohibition forbids — the fake records the
// note when the bridge rides to its deadline with no reply).
func assertCancelledFanInBackLeg(t *testing.T, tr fake.Transcript) {
	t.Helper()

	replies := 0
	dangling := false

	for _, line := range tr.Lines {
		if line.Dir == fake.DirNote && strings.Contains(line.JSON, "unanswered at") &&
			strings.Contains(line.JSON, permReverseID) {
			dangling = true
		}

		probe := probeLineJSON(line.JSON)

		if line.Dir != fake.DirIn || !isReplyTo(probe, permReverseID) {
			continue
		}

		if probe["error"] == nil {
			t.Fatalf("the displaced turn's permission reverse earned a result reply (%s), want the "+
				"-32021 cancelled error (a deny after the turn was cancelled is the dangling-bridge shape)", line.JSON)
		}

		replies++
		assertCancelledError(t, probe, line.JSON)
	}

	if replies != 1 {
		t.Fatalf("the displaced ask earned %d reply envelopes within the bound, want exactly 1 (-32021 "+
			"cancelled) — the mine arm omitted the pendings fan-in (WR-06), so the pending bridge dangled; "+
			"unanswered-at note present on the transcript: %v", replies, dangling)
	}
}

// TestCancelFanInStaleArmSkips locks the displaced-prompt cancel's
// containment (plan 05-10 WR-04, the observable contract): turn 1's SDK ctx
// is cancelled by the second prompt's registration while turn 1 is active —
// the fan-in's genuine arm cancels exactly THAT turn — and a LATER turn on
// the same machine completes untouched: the identity re-check (and the
// release-time goroutine reap) keeps the stale arm from ever reaching a
// later turn. The genuine displaced-prompt cancel itself stays locked by
// testFrontSlotRejection; this lock adds the later-turn containment over
// the corpus row's completing second send.
//
// Drive note (plan-05-10 refinement): the HEAD bug's harmful interleaving
// (a descheduled arm waking past release + re-acquisition) is a scheduler
// race with no deterministic drive through the public surface — the arm's
// own turn context closes at release, so a cancel arriving after the
// boundary finds the goroutine already reaped. This test locks the
// reachable contract end-to-end; the skip arm's identity check is locked
// deterministically by TestCancelFanInOnlyWhenSlotHeld's slot-gate (the
// same f.turnPump/f.turnSession identity semantics).
func TestCancelFanInStaleArmSkips(t *testing.T) {
	gate := make(chan struct{})

	h := newPermissionFrontOpts(t, "permission-cancel", PermissionScript{Gate: gate}, nil)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openPermissionSession(ctx, t, h)

	// Turn 1 parks on the pending ask; its SDK ctx dies at the second
	// prompt's registration (the SDK displacement).
	prompt1Done := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		prompt1Done <- promptOutcome{resp: resp, err: err}
	}()

	awaitPermissionAsk(t, h)

	// Displace turn 1: the second prompt is rejected by the held slot, and
	// turn 1 resolves cancelled through the genuine displaced arm.
	_, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("second")},
	})
	assertRejectedInvalidRequest(t, err)

	outcome := waitPermissionOutcome(t, prompt1Done)

	if outcome.err != nil || outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("displaced turn outcome = (%v, %v), want a cancelled success",
			outcome.resp.StopReason, outcome.err)
	}

	// A LATER turn on the same machine completes untouched: whatever the
	// displaced fan-in arm did, it must never reach past the turn boundary.
	prompt3, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("third")},
	})
	if err != nil {
		t.Fatalf("later prompt: %v", err)
	}

	if prompt3.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("later prompt stopReason = %q, want end_turn (the stale arm must not cancel a later turn)",
			prompt3.StopReason)
	}

	// Exactly ONE session/stop on the back leg — the displaced turn's own
	// cancel; a stale cancel of the later turn would fire a second.
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if stops := len(transcriptIndexes(tr, "in", `"method":"session/stop"`)); stops != 1 {
		t.Fatalf("back leg records %d session/stop frames, want exactly 1 (the displaced turn's cancel — "+
			"a stale arm firing past the boundary would stop the later turn too)", stops)
	}
}

// WR-07 session-scoped fan-in locks (plan 05-15 Task 2). Constants local
// to TestCancelFanInSkipAndScope's drives.
const (
	// foreignScopeSessionID / foreignScopeRequestID arm the session-scope
	// subtest's FOREIGN ask: a distinct session id string drives the scope
	// discriminator deterministically (registry-distinct sessions are
	// impossible over one fake — every corpus create returns the same
	// backend id — but a foreign SessionID in the pending table is exactly
	// the shape the scoping must spare).
	foreignScopeSessionID = "sess_foreign_other"
	foreignScopeRequestID = "perm-foreign-1"

	// slotFreeRequestID arms the slot-free subtest's parked round-trip.
	slotFreeRequestID = "perm-slotfree-1"

	// fanInLogLine is the discriminator log's stable substring (the
	// fan-in's structural log — occurrences are COUNTED, never
	// boolean-contains).
	fanInLogLine = "permission pendings cancelled by turn cancel"
)

// TestCancelFanInSkipAndScope locks the WR-07 session-scoped fan-in (plan
// 05-15 Task 2) on the two arms the pending-table re-key must preserve:
//
//   - session scope spares foreign sessions: with a turn of session S
//     active and its own ask pending, a second round-trip parked directly
//     through the exported PermissionForward seam for a FOREIGN session id
//     survives front.Cancel(S) — the fan-in answers S's own pending ask
//     -32021 and never touches the foreign round-trip, which resolves as
//     the client decision on gate release (its derived context never
//     cancelled). At HEAD the front-wide sweep cancels the foreign
//     round-trip — the RED.
//   - slot free spares parked pendings: the RE-HOMED WR-04/CR-01 skip lock
//     (formerly TestCancelFanInOnlyWhenSlotHeld's negative-log tail,
//     dropped in the plan-05-15 enshrining rewrite) — with NO active turn,
//     a round-trip parked for the session's OWN id survives front.Cancel:
//     the discriminator log count is unchanged and the round-trip resolves
//     normally on gate release. A pending actually present while the slot
//     is free gives the lock back its discriminating power.
func TestCancelFanInSkipAndScope(t *testing.T) {
	t.Run("session scope spares foreign sessions", testCancelFanInSessionScope)
	t.Run("slot free spares parked pendings", testCancelFanInSlotFreeSkip)
}

// testCancelFanInSessionScope drives the session-scope discriminator: turn
// 1 parks on its own ask; a FOREIGN session's round-trip parks behind the
// same scripted client gate through the exported seam; front.Cancel(sid)
// (the genuine slot-held arm) must fan in exactly the session's own bridge.
func testCancelFanInSessionScope(t *testing.T) {
	gate := make(chan struct{})

	h, front := newSlotStage(t, gate)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openPermissionSession(ctx, t, h)

	// Turn 1 parks on its own pending ask (the session's bridge — the one
	// the genuine arm must fan in).
	prompt1Done := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		prompt1Done <- promptOutcome{resp: resp, err: err}
	}()

	awaitPermissionAsk(t, h)

	// The FOREIGN session's round-trip parks directly through the exported
	// PermissionForward seam (the same seam the interact Responder wiring
	// drives), behind the same scripted client gate as the real ask. The
	// drive captures both the outcome and its own context.
	foreignCtx, foreignCancel := context.WithCancel(ctx)
	defer foreignCancel()

	foreignOutcome := make(chan acpport.PermissionOutcome, 1)
	foreignErr := make(chan error, 1)

	go func() {
		outcome, err := front.PermissionForward()(foreignCtx, acpport.PermissionAsk{
			SessionID:  foreignScopeSessionID,
			RequestID:  foreignScopeRequestID,
			ToolCallID: "tc-foreign",
			ToolName:   "Inert",
		})
		foreignOutcome <- outcome
		foreignErr <- err
	}()

	awaitSecondPermissionAsk(t, h)

	// The genuine arm: the session holds the slot — the fan-in must answer
	// THE SESSION'S ask -32021 and spare the foreign round-trip (WR-07).
	if err := front.Cancel(ctx, sid); err != nil {
		t.Fatalf("cancel (slot held): %v", err)
	}

	// Whatever survived parks no more — the foreign round-trip resolves as
	// the client decision.
	close(gate)

	outcome := <-foreignOutcome
	if err := <-foreignErr; err != nil {
		t.Fatalf("foreign round-trip errored: %v", err)
	}

	if foreignCtx.Err() != nil {
		t.Fatalf("the foreign drive context was cancelled — a sweep reached past the session scope")
	}

	if outcome.Cancelled {
		t.Fatalf("the FOREIGN session's pending round-trip read cancelled — the fan-in swept past "+
			"its session (the front-wide table, WR-07); want the client decision "+
			"(session scope spares foreign sessions)")
	}

	// The displaced turn still resolves exactly once as the cancelled
	// success (the genuine arm's machine.Cancel landed).
	promptRes := waitPermissionOutcome(t, prompt1Done)

	if promptRes.err != nil || promptRes.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("displaced turn outcome = (%v, %v), want a cancelled success",
			promptRes.resp.StopReason, promptRes.err)
	}

	// The real ask's fan-in artifact on the back leg: exactly one -32021
	// reply (bounded settle — the fake materializes its transcript only at
	// Serve return; backLegSettle outlasts the row's 500ms reverse
	// deadline either way).
	time.Sleep(backLegSettle)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertCancelledFanInBackLeg(t, tr)
}

// testCancelFanInSlotFreeSkip is the re-homed WR-04/CR-01 skip lock with
// full discriminating power: the session is registered but IDLE (no prompt
// ever ran — the slot is free) and a round-trip is actually parked through
// the exported seam; front.Cancel must skip the fan-in entirely (the
// discriminator log count unchanged) and the parked round-trip resolves
// normally on gate release.
func testCancelFanInSlotFreeSkip(t *testing.T) {
	// Capture the default logger: the fan-in's structural log is the
	// discriminator — occurrences are counted, never boolean-contains.
	var buf safeLogBuffer

	origLog := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(origLog) })

	gate := make(chan struct{})

	h, front := newSlotStage(t, gate)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	// The session is open but IDLE — no prompt, the slot is free.
	sid := openPermissionSession(ctx, t, h)

	// A round-trip parks for the session's OWN id (a pending actually
	// present while the slot is free — the shape the old negative-log
	// tail could never construct).
	outcomeCh := make(chan acpport.PermissionOutcome, 1)
	errCh := make(chan error, 1)

	go func() {
		outcome, err := front.PermissionForward()(ctx, acpport.PermissionAsk{
			SessionID:  sid,
			RequestID:  slotFreeRequestID,
			ToolCallID: "tc-idle",
			ToolName:   "Inert",
		})
		outcomeCh <- outcome
		errCh <- err
	}()

	awaitPermissionAsk(t, h)

	before := buf.count(fanInLogLine)

	if err := front.Cancel(ctx, sid); err != nil {
		t.Fatalf("cancel (slot free): %v", err)
	}

	if after := buf.count(fanInLogLine); after != before {
		t.Fatalf("the slot-free cancel fanned pendings in (fan-in log fired %d → %d occurrences) — "+
			"Front.Cancel's fan-in must fire only while a turn of THIS session holds the slot (WR-04/CR-01)",
			before, after)
	}

	// The parked round-trip survives: on gate release it resolves as the
	// client decision (its derived context never cancelled).
	close(gate)

	outcome := <-outcomeCh
	if err := <-errCh; err != nil {
		t.Fatalf("parked round-trip errored: %v", err)
	}

	if outcome.Cancelled {
		t.Fatalf("the parked round-trip read cancelled — a slot-free cancel must never sweep (WR-04/CR-01)")
	}
}

// awaitSecondPermissionAsk blocks until the client captured its SECOND
// request_permission ask (bounded) — the foreign round-trip's park point.
func awaitSecondPermissionAsk(t *testing.T, h *FrontHarness) {
	t.Helper()

	deadline := time.After(permWait)

	for {
		if len(h.rec.PermissionAsks()) >= 2 {
			return
		}

		select {
		case <-time.After(2 * time.Millisecond):
		case <-deadline:
			t.Fatal("the foreign ask never reached the client within the wait bound")
		}
	}
}
