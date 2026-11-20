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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	// turnActiveMessage is the ONE stable rejection surface the
	// second-prompt guard answers with, whatever guard layer fires (the
	// front-wide slot or the machine's ErrTurnActive) — asserted verbatim.
	turnActiveMessage = "a prompt is already running for this session"

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
			t.Fatalf("update frame at %d streams after the prompt response (line %d) — post-cancel updates MUST precede the response",
				i, stopIdxs[0])
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
