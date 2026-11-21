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

// TestCancelFanInOnlyWhenSlotHeld locks the WR-04/CR-01 cancel alignment
// (plan 05-10): Front.Cancel's permission fan-in fires only while a turn of
// THIS session holds the front-wide slot — the same semantics drainActive-
// Turn's sawTurn reports on the close path. The deterministic drive stages
// the slot-free state with a pending ask still registered: turn 1 parks on
// the pending ask, the SDK's second-prompt displacement cancels it (the
// fan-in goroutine's machine.Cancel — that arm never fans pendings), the
// turn resolves and RELEASES the slot, leaving the ask dangling with the
// slot free. The session cancel must then SKIP the fan-in — the pending
// bridge survives (the discriminator is the fan-in's own structural log,
// which fires synchronously inside the cancel call; on the wire the ask
// rides to the fake's honest unanswered note). At HEAD the fan-in is
// unconditional and the log fires. With the slot held, the existing
// TestPermissionCancel behavior is unchanged (the genuine arm).
//
// Note on the plan's "another session" flavor: every corpus create returns
// the SAME deterministic backend id (sess_fix1), so two registry-distinct
// sessions are impossible over one fake — the slot-free-with-dangling-pending
// state exercises the identical slotHeld=false skip arm.
func TestCancelFanInOnlyWhenSlotHeld(t *testing.T) {
	// Capture the default logger: the fan-in's structural log is the
	// deterministic discriminator (it fires synchronously inside the
	// cancel call, before any teardown race can interfere).
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
	// genuine arm), the turn resolves cancelled and the slot RELEASES. The
	// second prompt itself is rejected by the held slot (one clean
	// invalid-request).
	_, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("second")},
	})
	assertRejectedInvalidRequest(t, err)

	// Turn 1 resolved cancelled (the displacement's machine.Cancel landed
	// through the stop rule's cancelled patch) — the slot is FREE and the
	// pending ask dangles.
	outcome := waitPermissionOutcome(t, prompt1Done)

	if outcome.err != nil || outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("displaced turn outcome = (%v, %v), want a cancelled success",
			outcome.resp.StopReason, outcome.err)
	}

	// The cancel with NO slot held, driven DIRECTLY on the front (the
	// exported seam — no SDK prompt-registration state in the way): the
	// fan-in must be SKIPPED — the dangling ask survives.
	if err := front.Cancel(ctx, sid); err != nil {
		t.Fatalf("cancel (slot free): %v", err)
	}

	// The discriminator: the fan-in's structural log must NOT have fired —
	// a cancel for a session that holds no slot fans no pendings in
	// (WR-04/CR-01; at HEAD the unconditional fan-in logs count=1 here).
	if fired := buf.contains("permission pendings cancelled by turn cancel"); fired {
		t.Fatalf("the slot-free cancel fanned the pending bridge in (fan-in log fired) — " +
			"Front.Cancel's fan-in must fire only while a turn of THIS session holds the slot (WR-04/CR-01)")
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// The wire artifact of the surviving ask: the fake's honest note — the
	// reverse rode to its 500ms deadline unanswered (the ask was never
	// cancelled by anything in this flow).
	assertDanglingAskNote(t, tr)
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

// assertDanglingAskNote locks the surviving-ask wire artifact: the fake
// recorded the reverse's honest unanswered note (it rode to its deadline
// with no reply — nothing cancelled the dangling bridge).
func assertDanglingAskNote(t *testing.T, tr fake.Transcript) {
	t.Helper()

	for _, line := range tr.Lines {
		if line.Dir == fake.DirNote && strings.Contains(line.JSON, "unanswered at") &&
			strings.Contains(line.JSON, permReverseID) {
			return
		}
	}

	t.Fatalf("the dangling ask left no unanswered note on the transcript — " +
		"the surviving-ask artifact is missing (was the ask cancelled after all?)")
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
