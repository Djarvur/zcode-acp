package turn_test

// No-progress stall watchdog contract tests (quick task 261125-r6o, Task 2):
// the 2026-11-25 field incident's second defect — a 429 retry backoff kept
// one turn silent ~50 minutes because the quiet watcher counted retry/model-
// request telemetry as activity and no independent progress bound existed.
//
// The watchdog is a RUNNING-turn guard: it fires only past the configured
// no-progress budget, sends exactly one bypass-lane session/stop BEFORE
// resolving, resolves StopError/SignalStall with a recorded cause, and is
// joined on every Run return path. Real model/tool progress (deltas, tool
// lifecycle, turn starts/terminals, session/messages content changes)
// resets the clock; retry telemetry and stable poll echoes do not. A
// cancelling turn keeps the existing quiet/cancel contract — the watchdog
// never fires and never sends a second stop.
//
// Schedules are scaled so each discriminating boundary keeps ~20-30 ms of
// margin (the assertions, not the literal millisecond, are the contract).

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// stallQuietMs/stallFloorMultiple keep the quiet fallback far away from a
// watchdog decision: the watchdog budgets below are hundreds of
// milliseconds at most, while the fallback's first eligible settle sits at
// >= 2 x stallQuietMs.
const (
	stallQuietMs       = 200
	stallFloorMultiple = 2
	stallPollMs        = 10
)

// stallPollRule is one unit-budget session/messages poll rule: the stable
// poll-surface reply, an optional reply delay, and the ordered side effects
// the fake writes BEFORE the delayed reply (so an effect's time is the
// poll-request time, and delay_ms shifts the reply — and therefore every
// later poll).
func stallPollRule(t *testing.T, delayMs int, effects ...fake.SideEffect) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method:      zcode.MethodSessionMessages,
		Times:       1,
		DelayMs:     delayMs,
		Reply:       qa03StableReply(t),
		SideEffects: effects,
	}
}

// stallPollRuleReply is stallPollRule with an explicit reply payload.
func stallPollRuleReply(t *testing.T, delayMs int, reply *fake.Reply, effects ...fake.SideEffect) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method:      zcode.MethodSessionMessages,
		Times:       1,
		DelayMs:     delayMs,
		Reply:       reply,
		SideEffects: effects,
	}
}

// stallToolEvent is one session/event tool.updated lifecycle envelope for
// the fixed turn (the live carrier shape).
func stallToolEvent(t *testing.T, kind string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodSessionEvent, map[string]any{
		wireKeyType:      "tool.updated",
		wireKeyTurnID:    fixTurnID,
		wireKeySessionID: fixSessionID,
		wireKeySeq:       3,
		wireKeyPayload: map[string]any{
			wireKeyKind:  kind,
			"toolCallId": "tc_stall",
			"toolName":   "Read",
		},
	})
}

// stallModelStream is one session/event model.streaming delta envelope (the
// reply-text carrier whose payload.kind names the delta class).
func stallModelStream(t *testing.T, deltaKind string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodSessionEvent, map[string]any{
		wireKeyType:      "model.streaming",
		wireKeyTurnID:    fixTurnID,
		wireKeySessionID: fixSessionID,
		wireKeySeq:       5,
		wireKeyPayload: map[string]any{
			wireKeyKind: deltaKind,
			"delta":     "x",
		},
	})
}

// stallPartDelta is one session/event part.delta envelope (the alternative
// text/reasoning progress carrier).
func stallPartDelta(t *testing.T, field string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodSessionEvent, map[string]any{
		wireKeyType:      "part.delta",
		wireKeyTurnID:    fixTurnID,
		wireKeySessionID: fixSessionID,
		"field":          field,
		wireKeyDelta:     "x",
	})
}

// stallTelemetry is the retry/model-request telemetry pair observed live
// (EXP-APIKEY-CYCLE-03.jsonl): a v4 model.request.status event and a
// session/event session.updated payload carrying the same lifecycle fact.
// Neither is model progress.
func stallTelemetry(t *testing.T) []fake.SideEffect {
	t.Helper()

	return []fake.SideEffect{
		emit(t, turn.MethodV4Telemetry, map[string]any{
			wireKeyKind:      "model.request.status",
			wireKeyStatus:    "model_request_started",
			wireKeySessionID: fixSessionID,
			wireKeyEventSeq:  1,
		}),
		emit(t, zcode.MethodSessionEvent, map[string]any{
			wireKeyType:      "session.updated",
			wireKeySessionID: fixSessionID,
			wireKeySeq:       2,
			"payload": map[string]any{
				"kind":        "model.request.status",
				wireKeyStatus: "retry_scheduled",
			},
		}),
	}
}

// stallMessagesPage is one session/messages poll reply carrying the given
// raw messages array (byte-stable across reuse — the poll-surface echo).
func stallMessagesPage(t *testing.T, messages string) *fake.Reply {
	t.Helper()

	return &fake.Reply{Result: rawOf(t, map[string]any{
		wireKeyMessages:  jsonRaw(t, messages),
		wireKeySessionID: fixSessionID,
	})}
}

// jsonRaw turns one raw JSON literal into a json.RawMessage for embedding
// in a scenario payload (the non-empty messages array shapes).
func jsonRaw(t *testing.T, literal string) []any {
	t.Helper()

	var v []any
	if err := json.Unmarshal([]byte(literal), &v); err != nil {
		t.Fatalf("decode messages literal %q: %v", literal, err)
	}

	return v
}

// stallScenario builds one watchdog scenario: the send ack with the running
// patch (plus any send-time effects) and the ordered poll rules.
func stallScenario(t *testing.T, sendEffects []fake.SideEffect, polls ...fake.RequestRule) *fake.Scenario {
	t.Helper()

	onRequest := make([]fake.RequestRule, 0, 1+len(polls)+1)
	send := sendRule(t, append([]fake.SideEffect{
		statePatch(t, statusRunning, reasonPromptStarted),
	}, sendEffects...)...)
	onRequest = append(onRequest, send)
	onRequest = append(onRequest, polls...)
	onRequest = append(onRequest, messagesRule(t))

	return &fake.Scenario{
		Name:      "turn-stall",
		OnRequest: onRequest,
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// driveStall runs one watchdog turn with the configured budgets.
func driveStall(t *testing.T, ex *turnExchange, base, tool time.Duration) (turn.StopReason, error) {
	t.Helper()

	ex.machine.StallTimeout = base
	ex.machine.StallToolTimeout = tool

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	return ex.machine.Run(ctx, "stall prompt")
}

// TestStallWatchdogTripsOnSilentTurn is the field-incident regression: a
// RUNNING turn with no progress past the base budget sends exactly one
// bypass-lane stop, resolves StopError/SignalStall with a recorded model
// budget cause, and the watchdog goroutine is joined before Run returns (no
// second stop ever reaches the wire).
func TestStallWatchdogTripsOnSilentTurn(t *testing.T) {
	ex := startTurnExchangeKnobs(t, stallScenario(t, nil), stallQuietMs, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 120*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopError {
		t.Fatalf("Run reason = %q, want %q (the silent turn is an honest coded failure)", reason, turn.StopError)
	}

	assertResolution(t, ex.spy, turn.StopError, turn.SignalStall)

	res := ex.spy.snapshot()[0]
	if res.Err == nil || !strings.Contains(res.Err.Error(), "no model progress") {
		t.Fatalf("recorded stall cause = %v, want a message naming the missing model progress", res.Err)
	}

	// Join proof: the watchdog is done before Run returns, so no stop fires
	// in the window after resolution.
	time.Sleep(2 * 120 * time.Millisecond)

	tr := finishWithTranscript(t, ex)

	if stops := inLineCount(tr, `"method":"session/stop"`); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1 (one watchdog stop, joined)", stops)
	}
}

// TestStallWatchdogProgressResetsOnDelta: a model.streaming text_delta that
// lands before the base budget moves the deadline, letting the turn's own
// terminal win. Without the reset the budget fires first.
func TestStallWatchdogProgressResetsOnDelta(t *testing.T) {
	sc := stallScenario(t, nil,
		stallPollRule(t, 50),
		stallPollRule(t, 90, stallModelStream(t, "text_delta")),
		stallPollRule(t, 0, sessionEvent(t, turn.KindTurnCompleted, fixTurnID)),
	)

	ex := startTurnExchangeKnobs(t, sc, stallQuietMs, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 120*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q (the delta reset the no-progress clock)", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	ex.finish(t)
}

// TestStallWatchdogProgressResetsOnPartDelta (residual R1): the part.delta
// carrier with field text/reasoning is the alternative text-progress shape;
// ignoring it would let the base budget fire before the terminal.
func TestStallWatchdogProgressResetsOnPartDelta(t *testing.T) {
	sc := stallScenario(t, nil,
		stallPollRule(t, 50),
		stallPollRule(t, 90, stallPartDelta(t, "text")),
		stallPollRule(t, 0, sessionEvent(t, turn.KindTurnCompleted, fixTurnID)),
	)

	ex := startTurnExchangeKnobs(t, sc, stallQuietMs, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 120*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q (the part.delta reset the no-progress clock)", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	ex.finish(t)
}

// TestStallWatchdogTurnStartIsProgress (residual R1): a turn-start signal
// alone keeps a silent pre-budget turn alive past the base budget — the
// turn-start at ~50 ms moves the 120 ms deadline to ~170 ms. Without the
// reset the foster fires at ~120 ms.
func TestStallWatchdogTurnStartIsProgress(t *testing.T) {
	sc := stallScenario(t, nil,
		stallPollRule(t, 40),
		stallPollRule(t, 0, sessionEvent(t, turn.KindTurnStarted, fixTurnID)),
	)

	ex := startTurnExchangeKnobs(t, sc, stallQuietMs, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 120*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopError {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopError)
	}

	assertResolution(t, ex.spy, turn.StopError, turn.SignalStall)

	got := ex.spy.snapshot()[0]
	if got.At.Before(ex.machine.AckAt().Add(150 * time.Millisecond)) {
		t.Fatalf("stall fired at %v, before the turn-start reset window elapsed (ack %v + 150 ms) — "+
			"the turn-start did not count as progress", got.At, ex.machine.AckAt())
	}

	ex.finish(t)
}

// TestStallWatchdogActiveToolAllowance: a scheduled/started tool with no
// result gets the larger tool allowance — the base budget never fires.
func TestStallWatchdogActiveToolAllowance(t *testing.T) {
	ex := startTurnExchangeKnobs(t, stallScenario(t, []fake.SideEffect{
		stallToolEvent(t, "scheduled"),
	}), stallQuietMs, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 60*time.Millisecond, 180*time.Millisecond)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopError {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopError)
	}

	assertResolution(t, ex.spy, turn.StopError, turn.SignalStall)

	got := ex.spy.snapshot()[0]
	if got.At.Before(ex.machine.AckAt().Add(150 * time.Millisecond)) {
		t.Fatalf("stall fired at %v, inside the base budget (ack %v + 150 ms) — "+
			"the active tool did not get the larger allowance", got.At, ex.machine.AckAt())
	}

	if got.Err == nil || !strings.Contains(got.Err.Error(), "no tool progress") {
		t.Fatalf("recorded stall cause = %v, want the tool-budget message", got.Err)
	}

	ex.finish(t)
}

// TestStallWatchdogToolCompletionRestoresBaseBudget: a tool result/error
// ends the tool allowance; the remaining silence is then bounded by the
// MODEL budget (the timing and the message both discriminate — a missed
// end-of-tool would fire the much larger tool budget with tool wording).
func TestStallWatchdogToolCompletionRestoresBaseBudget(t *testing.T) {
	sc := stallScenario(t, []fake.SideEffect{
		stallToolEvent(t, "scheduled"),
	},
		stallPollRule(t, 30),
		stallPollRule(t, 0, stallToolEvent(t, "result")),
	)

	ex := startTurnExchangeKnobs(t, sc, stallQuietMs, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 60*time.Millisecond, 300*time.Millisecond)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopError {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopError)
	}

	assertResolution(t, ex.spy, turn.StopError, turn.SignalStall)

	got := ex.spy.snapshot()[0]

	// The result lands at ~40 ms and the 60 ms model budget then fires at
	// ~100 ms. A missed end-of-tool would fire the 300 ms tool budget.
	if got.At.Before(ex.machine.AckAt().Add(70*time.Millisecond)) ||
		got.At.After(ex.machine.AckAt().Add(250*time.Millisecond)) {
		t.Fatalf("stall fired at %v (ack %v), want ~ack+100 ms — the tool completion did not restore the base budget",
			got.At, ex.machine.AckAt())
	}

	if got.Err == nil || !strings.Contains(got.Err.Error(), "no model progress") {
		t.Fatalf("recorded stall cause = %v, want the model-budget message", got.Err)
	}

	ex.finish(t)
}

// TestStallWatchdogIgnoresRetryTelemetry: model-request/retry telemetry is a
// lifecycle fact, not progress — a telemetry-only silence still fires the
// watchdog before the later terminal arrives.
func TestStallWatchdogIgnoresRetryTelemetry(t *testing.T) {
	sc := stallScenario(t, nil,
		stallPollRule(t, 50),
		stallPollRule(t, 90, stallTelemetry(t)...),
		stallPollRule(t, 0, sessionEvent(t, turn.KindTurnCompleted, fixTurnID)),
	)

	ex := startTurnExchangeKnobs(t, sc, stallQuietMs, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 120*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopError {
		t.Fatalf("Run reason = %q, want %q (retry telemetry must not reset the clock)", reason, turn.StopError)
	}

	assertResolution(t, ex.spy, turn.StopError, turn.SignalStall)

	ex.finish(t)
}

// TestStallWatchdogMessagesContentChangeIsProgress: a real session/messages
// content change resets the clock (observed at ~60 ms), while the byte-
// identical echo that follows does not — so the ~180 ms terminal wins
// instead of the 150 ms budget.
func TestStallWatchdogMessagesContentChangeIsProgress(t *testing.T) {
	contentPage := stallMessagesPage(t, `[{"id":"m1"}]`)

	sc := stallScenario(t, nil,
		stallPollRuleReply(t, 0, qa03StableReply(t)),
		stallPollRuleReply(t, 40, contentPage),
		stallPollRuleReply(t, 120, contentPage),
		stallPollRuleReply(t, 0, contentPage, sessionEvent(t, turn.KindTurnCompleted, fixTurnID)),
	)

	ex := startTurnExchangeKnobs(t, sc, 500, stallFloorMultiple, stallPollMs)

	reason, err := driveStall(t, ex, 150*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q (the messages content change reset the clock)", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	ex.finish(t)
}

// TestStallWatchdogDoesNotFireWhileCancelling (residual R3): a cancelling
// turn is outside the watchdog's scope — the 60 ms budget expires while the
// turn is cancelling, the watchdog parks/exits without a fire or a second
// stop, and the quiet settle resolves the stop's outcome at the cancel
// floor. The turn lives past the budget.
func TestStallWatchdogDoesNotFireWhileCancelling(t *testing.T) {
	sc := &fake.Scenario{
		Name: "turn-stall-cancelling",
		OnRequest: []fake.RequestRule{
			sendRule(t, statePatch(t, statusRunning, reasonPromptStarted)),
			{
				Method:  zcode.MethodSessionStop,
				Times:   1,
				DelayMs: 300,
				Reply:   &fake.Reply{Result: rawOf(t, map[string]any{})},
			},
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}

	// Quiet 200 ms / floor 2 = the settle at ~400 ms; the stall budget is
	// 60 ms, so a cancelling-scope leak would fire long before the settle.
	ex := startTurnExchangeKnobs(t, sc, 200, 2, stallPollMs)
	ex.machine.StallTimeout = 60 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	runDone := runAsync(ctx, ex.machine, "stall prompt")
	waitAck(t, ex.machine)

	ex.machine.Cancel()

	got := <-runDone
	if got.err != nil {
		t.Fatalf("Run: %v", got.err)
	}

	if got.reason != turn.StopCancelled {
		t.Fatalf("Run reason = %q, want %q (the cancel contract owns the turn, not the watchdog)",
			got.reason, turn.StopCancelled)
	}

	assertResolution(t, ex.spy, turn.StopCancelled, turn.SignalCancel)

	// The turn outlived the 60 ms stall budget by construction (the quiet
	// settle is at ~400 ms) and still resolved cancelled.
	res := ex.spy.snapshot()[0]
	if res.At.Before(ex.machine.AckAt().Add(90 * time.Millisecond)) {
		t.Fatalf("cancelled resolution at %v predates the stall budget (ack %v + 90 ms) — "+
			"the turn did not actually outlive it", res.At, ex.machine.AckAt())
	}

	tr := finishWithTranscript(t, ex)

	if stops := inLineCount(tr, `"method":"session/stop"`); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1 (the cancel stop; no watchdog stop)",
			stops)
	}
}
