package turn_test

import (
	"context"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// QA-03 field-blocker tests (the turn machine vs the REAL 3.18.2 desktop's
// terminal surface, EXP-VM-QA02-family dumps 2026-11-23 — /tmp/qa03-raw3.out
// and /tmp/qa03-raw4.out, green106): the real terminal signal arrives on
// computer-use/operation-event (kind "turn-completed", hyphen-spelled) AND
// v4/telemetry/event (kind "turn.terminal" with status/resultType "success"
// — NOT the bundle taxonomy's "turn.completed"), each carrying the turnId.
//
// The field failure these tests pin: the model thinks several seconds before
// the first delta, the poll watcher's quiet floor (3x quiet from the send
// ack) releases the settle DURING the think, and the machine resolved the
// turn off the floor — 0.1 s after the first delta, cutting the reply —
// while the real terminal signal was still seconds away. The machine must
// treat a start-announced turn (the desktop's turn-started operation event)
// as a promise of a terminal signal: the quiet settle may not end such a
// turn while the budget holds, so the real signal resolves it instead.

// qa03CUStarted is the raw4.out:34 operation-event shape verbatim (scrubbed
// ids): eventId, sequenceNumber, sessionId, ms-epoch timestamp, hyphen kind,
// turnId — the REAL desktop's turn-start announcement.
func qa03CUStarted(t *testing.T, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodComputerUseOperation, map[string]any{
		wireKeyEventID:        "evt_cu_start",
		wireKeySequenceNumber: 4,
		wireKeySessionID:      fixSessionID,
		wireKeyTimestamp:      1795371946926,
		wireKeyKind:           turn.KindTurnStartedCU,
		wireKeyTurnID:         turnID,
	})
}

// qa03CUCompleted is the raw4.out:66 operation-event shape verbatim
// (scrubbed ids): the REAL terminal signal the machine must resolve turns
// by, with the same ms-epoch timestamp shape the dump records.
func qa03CUCompleted(t *testing.T, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodComputerUseOperation, map[string]any{
		wireKeyEventID:        "evt_cu_done",
		wireKeySequenceNumber: 30,
		wireKeySessionID:      fixSessionID,
		wireKeyTimestamp:      1795371951553,
		wireKeyKind:           turn.KindTurnCompletedCU,
		wireKeyTurnID:         turnID,
	})
}

// qa03V4Terminal is the raw3.out:84 v4/telemetry/event shape verbatim
// (scrubbed ids): kind "turn.terminal" with status/resultType "success" —
// the real desktop's v4-carrier terminal kind, which is NOT the bundle
// taxonomy's "turn.completed" the classifier grew up on.
func qa03V4Terminal(t *testing.T, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodV4Telemetry, map[string]any{
		"memoryEnabled":  false,
		"version":        1,
		wireKeyEventID:   "evt_cu_done",
		"eventSeq":       46,
		"occurredAt":     1795371856008,
		wireKeySessionID: fixSessionID,
		wireKeyTurnID:    turnID,
		wireKeyKind:      "turn.terminal",
		"status":         "success",
		"resultType":     "success",
		"durationMs":     5701,
		"tokenCount":     35396,
		"toolCallCount":  0,
	})
}

// qa03Delta is a model.streaming delta on the session/event carrier (the
// raw4.out:43 shape verbatim, scrubbed): the reply-text carrier whose
// orphaned chunk is the QA-03 symptom.
func qa03Delta(t *testing.T, deltaKind, text string, seq int) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodSessionEvent, map[string]any{
		"deliveryKind": deliveryKind,
		wireKeyEventID: "evt_delta_" + text,
		"payload": map[string]any{
			"assistantMessageId": "msg_fix1",
			"delta":              text,
			"done":               false,
			"kind":               deltaKind,
		},
		"seq":            seq,
		wireKeySessionID: fixSessionID,
		wireKeyTimestamp: 1795371951072,
		"traceId":        "trace_fix1",
		wireKeyTurnID:    fixTurnID,
		wireKeyType:      "model.streaming",
	})
}

// qa03PollRule builds one session/messages poll rule for the delayed-
// terminal script: times caps its matches (0 = unlimited), delayMs holds
// the reply (the scripted silent think), sideEffects ride the rule — the
// ordered emission shape the watcher's next poll consumes. STABLE reply
// bytes every poll — the think-pause poll surface: identical bytes each
// time, so the poll-surface quiet gate arms exactly as it did in the field
// runs.
func qa03PollRule(t *testing.T, times int, sideEffects []fake.SideEffect) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method:      zcode.MethodSessionMessages,
		Times:       times,
		Reply:       qa03StableReply(t),
		SideEffects: sideEffects,
	}
}

// qa03StableReply is the stable poll-surface reply payload.
func qa03StableReply(t *testing.T) *fake.Reply {
	t.Helper()

	return &fake.Reply{Result: rawOf(t, map[string]any{
		wireKeyMessages:  []any{},
		wireKeySessionID: fixSessionID,
	})}
}

// qa03Run drives one scenario through the machine exchange and asserts the
// single resolution carries the expected reason and resolving-signal record
// — the shared body of the real-shape tests. budget overrides the
// quiet-settle veto budget (0 = the default); the quiet-resolving arms use
// a small budget so the fallback lands well inside the exchange window.
func qa03Run(t *testing.T, sc *fake.Scenario, budget int, want turn.SignalKind) {
	t.Helper()

	ex := startTurnExchange(t, sc)
	ex.machine.MaxQuietVetoes = budget

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	reason, err := ex.machine.Run(ctx, "Reply with exactly: OK")
	if err != nil {
		t.Fatalf("Run: turn did not resolve within the exchange window: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, want)

	ex.finish(t)
}

// The REAL terminal shapes resolve the turn (the signal table's live rows):
// the computer-use turn-completed operation event with a matching turnId
// resolves COMPLETED (the raw4.out:66 shape verbatim), and the v4 telemetry
// turn.terminal event with status success resolves COMPLETED (the
// raw3.out:84 shape verbatim — the kind the bundle taxonomy never named).
func TestRealTerminalShapesResolve(t *testing.T) {
	t.Run("computer-use turn-completed raw shape", func(t *testing.T) {
		sc := completionScenario(t,
			statePatch(t, statusRunning, reasonPromptStarted),
			qa03CUStarted(t, fixTurnID),
			qa03CUCompleted(t, fixTurnID),
		)

		qa03Run(t, sc, 0, turn.SignalTerminalEvent)
	})

	t.Run("v4 telemetry turn.terminal success raw shape", func(t *testing.T) {
		sc := completionScenario(t,
			statePatch(t, statusRunning, reasonPromptStarted),
			sessionEvent(t, turn.KindTurnStarted, fixTurnID),
			qa03V4Terminal(t, fixTurnID),
		)

		// The live v4 terminal kind must complete the turn — never the
		// quiet floor it fell back to before the kind was classified.
		qa03Run(t, sc, 0, turn.SignalTerminalEvent)
	})
}

// A foreign turnId on the real carriers never resolves (IN-05: a foreign
// turn's envelope is observation only) — proven observably: the turn still
// ends over the quiet window, so the resolution record names the quiet
// window, never the foreign signals.
func TestRealTerminalForeignTurnIDNeverResolves(t *testing.T) {
	sc := completionScenario(t,
		statePatch(t, statusRunning, reasonPromptStarted),
		qa03CUStarted(t, fixTurnID),
		qa03CUCompleted(t, "turn_foreign1"),
		qa03V4Terminal(t, "turn_foreign2"),
	)

	qa03Run(t, sc, 2, turn.SignalQuietWindow)
}

// delayedTerminalScenario is the field bug as a script: the send carries the
// running patch and the desktop's turn-started operation event (the promise
// of a terminal signal), the poll surface stays byte-stable through the
// think, the SECOND poll holds the watcher off for 100 ms — longer than the
// injected floor (2x40 ms from the ack) — so the quiet gates release during
// the think. Its side effect is the orphan reasoning delta: the one chunk
// that landed before the floor in the field runs (suppressing the wrapper's
// zero-chunk salvage and leaving the reply text undelivered). Poll 3 is the
// floor instant — the poll surface still carries NO reply text (the field
// truth the salvage path read). The REAL terminal signals and the reply
// text ride the FOURTH poll, after the floor. This is raw4's arc with the
// wall clock compressed: turn-started → orphan delta → silent think past
// the floor → floor instant with no text → text delta + terminal signal.
func delayedTerminalScenario(t *testing.T) *fake.Scenario {
	t.Helper()

	return &fake.Scenario{
		Name: "turn-qa03-terminal-signal",
		OnRequest: []fake.RequestRule{
			sendRule(t,
				statePatch(t, statusRunning, reasonPromptStarted),
				qa03CUStarted(t, fixTurnID),
			),
			qa03PollRule(t, 1, nil), // poll 1 — the think's stable surface
			{
				Method:  zcode.MethodSessionMessages, // poll 2 — the silent think outlasts the floor
				Times:   1,
				DelayMs: 100,
				Reply:   qa03StableReply(t),
				SideEffects: []fake.SideEffect{
					qa03Delta(t, "reasoning_delta", "Thinking", 7), // the orphan chunk's source
				},
			},
			qa03PollRule(t, 1, nil), // poll 3 — the floor instant: NO reply text yet (the field truth)
			{
				Method: zcode.MethodSessionMessages, // poll 4 — the REAL terminal lands after the floor
				Reply:  qa03StableReply(t),
				SideEffects: []fake.SideEffect{
					qa03Delta(t, "text_delta", "ok", 8), // the reply text, still pending at the floor
					qa03CUCompleted(t, fixTurnID),
					qa03V4Terminal(t, fixTurnID),
				},
			},
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// The QA-03 premature-end contract: a start-announced turn whose real
// terminal signal arrives AFTER the quiet floor resolves by that terminal
// signal — NEVER off the floor's settle. RED (the field bug): the machine
// resolves off the floor during the think (signal quiet-window) and the
// real signal lands against a finished turn, exactly the QA-03 report —
// the reply cut off 0.1 s after the first delta.
func TestTerminalSignalBeatsQuietFloor(t *testing.T) {
	ex := startTurnExchange(t, delayedTerminalScenario(t))

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	reason, err := ex.machine.Run(ctx, "Reply with exactly: OK")
	if err != nil {
		t.Fatalf("Run: turn did not resolve within the exchange window: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	// The discriminating assertion: the resolving signal is the REAL
	// terminal event, not the quiet floor. RED resolves quiet-window here —
	// the floor's settle fired during the silent think, before the terminal
	// signals on the fourth poll.
	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	// The resolution fired only after the delayed poll returned — the
	// terminal signals' emission instant (poll 4 starts after poll 2's
	// 100 ms hold); a floor-resolved turn would record an earlier instant.
	if got := ex.spy.snapshot(); got[0].At.Before(ex.machine.AckAt().Add(100 * time.Millisecond)) {
		t.Fatalf("resolution at %v precedes the delayed poll's return (ack %v + 100 ms) — "+
			"the turn ended before the real terminal signal was even on the wire",
			got[0].At, ex.machine.AckAt())
	}

	ex.finish(t)
}

// silentStartAnnouncedScenario is the budget script: the desktop's
// turn-started operation event announces the turn, the poll surface stays
// byte-stable, and NO terminal EVER arrives — the EXP-APIKEY-CYCLE-02 truth
// (a genuinely completed turn may present no terminal envelope at all),
// played against a start-announced turn whose runtime went silent.
func silentStartAnnouncedScenario(t *testing.T) *fake.Scenario {
	t.Helper()

	return &fake.Scenario{
		Name: "turn-qa03-veto-budget",
		OnRequest: []fake.RequestRule{
			sendRule(t,
				statePatch(t, statusRunning, reasonPromptStarted),
				qa03CUStarted(t, fixTurnID),
			),
			qa03PollRule(t, 0, nil), // the byte-stable surface, forever
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// The veto budget is the fallback's honest bound (the no-hang contract): a
// start-announced turn vetoes quiet settles only while the budget holds —
// budget consecutive silent settles — and then the quiet window resolves
// the turn after all. A start-announced turn whose promised terminal never
// lands (a runtime gone silent mid-turn) still ends; it never hangs.
func TestQuietVetoBudgetEndsSilentStartAnnouncedTurn(t *testing.T) {
	sc := silentStartAnnouncedScenario(t)

	ex := startTurnExchange(t, sc)
	ex.machine.MaxQuietVetoes = 2

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	reason, err := ex.machine.Run(ctx, "Reply with exactly: OK")
	if err != nil {
		t.Fatalf("Run: turn did not resolve within the exchange window: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalQuietWindow)

	// The budget held the floor's first settles off: with quiet 40 ms and
	// floor 2x40 ms, the first settle is eligible at ~80 ms and each veto
	// re-arms a full fresh window — a budget of 2 resolves on the THIRD
	// settle, never before two full silent windows past the first eligible
	// settle.
	floor := 2 * 40 * time.Millisecond
	twoWindows := 2 * 40 * time.Millisecond

	if got := ex.spy.snapshot(); got[0].At.Before(ex.machine.AckAt().Add(floor + twoWindows)) {
		t.Fatalf("quiet resolution at %v fired before the veto budget elapsed "+
			"(ack %v + floor %v + two vetoed settle windows %v) — the budget did not hold",
			got[0].At, ex.machine.AckAt(), floor, twoWindows)
	}

	ex.finish(t)
}

// The no-signal regression (the quiet fallback's own family, task 3e): a
// turn with NO start announcement keeps the legacy settle timing — the
// floor's first eligible settle resolves it, nothing vetoed.
func TestQuietWindowUnchangedWithoutStartSignal(t *testing.T) {
	sc := completionScenario(t,
		statePatch(t, statusRunning, reasonPromptStarted),
	)

	qa03Run(t, sc, 0, turn.SignalQuietWindow)
}
