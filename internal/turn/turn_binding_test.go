package turn_test

// Turn-binding contract tests (quick task 261125-r6o, Task 1): completion is
// bound to the LIVE turn, never to a retired predecessor's late signal.
//
// The 2026-11-25 field incident: a new turn was falsely completed ~4 s in by
// a prior turn's late idle/terminal signal. HEAD's record-on-first-sight
// turnId latch has no memory across turns (beginTurnLocked clears it), and
// the idle-patch latch accepts ANY later non-running status — its revision
// and reason were decoded away. These tests pin the two defenses:
//
//   - the retired turnId persists across turns: an unbound successor ignores
//     a late terminal/turn-start carrying the predecessor's id, and a late
//     idle patch whose revision is not strictly greater than the running
//     patch that armed the successor never completes it;
//   - the positive path survives: the turn's OWN later-revision idle patch
//     still resolves StopCompleted/SignalIdlePatch.

import (
	"context"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// latePriorSignalsScenario scripts two sequential turns on one machine:
// turn 1 announces the running patch revision 1, the turn.started for
// fixTurnID, and the turn.completed that resolves it; turn 2 announces the
// running patch revision 3 and then receives LATE signals from turn 1 — the
// retired turnId's turn.failed and an idle patch revision 2 — before its own
// ack. Neither may resolve turn 2; it ends over the quiet window instead.
func latePriorSignalsScenario(t *testing.T) *fake.Scenario {
	t.Helper()

	first := sendRule(t,
		statePatch(t, statusRunning, reasonPromptStarted),
		sessionEvent(t, turn.KindTurnStarted, fixTurnID),
		sessionEvent(t, turn.KindTurnCompleted, fixTurnID),
	)
	first.Times = 1

	second := sendRule(t,
		statePatchRev(t, statusRunning, reasonPromptStarted, 3),
		sessionEvent(t, turn.KindTurnFailed, fixTurnID),
		statePatchRev(t, statusIdle, reasonPromptCompleted, 2),
	)
	second.Times = 1

	return &fake.Scenario{
		Name: "turn-late-prior-signals",
		OnRequest: []fake.RequestRule{
			first,
			second,
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// TestLatePriorTurnSignalsDoNotResolveNewTurn is the false-completion
// regression: turn 1 resolves by its own terminal; turn 2 (still unbound —
// the send ack has not even landed when the side effects hit the wire) must
// ignore the retired turnId's late terminal and any idle patch not strictly
// newer than its own running patch, then resolve over the quiet window
// exactly once.
func TestLatePriorTurnSignalsDoNotResolveNewTurn(t *testing.T) {
	ex := startTurnExchange(t, latePriorSignalsScenario(t))

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	reason, err := ex.machine.Run(ctx, "first turn")
	if err != nil {
		t.Fatalf("first Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("first Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	reason, err = ex.machine.Run(ctx, "second turn")
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("second Run reason = %q, want %q (the late prior-turn signals must not resolve it)",
			reason, turn.StopCompleted)
	}

	got := ex.spy.snapshot()

	if len(got) != 2 {
		t.Fatalf("resolution hook fired %d times across two turns, want exactly 2: %+v", len(got), got)
	}

	if got[1].Reason != turn.StopCompleted || got[1].Signal != turn.SignalQuietWindow {
		t.Fatalf("second turn resolution = %+v, want completed/quiet-window "+
			"(the retired turn's late terminal and stale idle patch produced no resolution)", got[1])
	}

	// The quiet fallback cannot fire before ack + the injected floor
	// (FloorMultiple 2 x QuietMs 40 = the 80 ms floor).
	floor := 2 * 40 * time.Millisecond
	if got[1].At.Before(ex.machine.AckAt().Add(floor)) {
		t.Fatalf("second turn resolution at %v fired before ack %v + floor %v elapsed",
			got[1].At, ex.machine.AckAt(), floor)
	}

	ex.finish(t)
}

// TestLatePriorTurnStartBeforeRunningPatchIsIgnored locks the narrow
// carve-out boundary: a retired-id turn-start observed BEFORE this turn's
// own running patch is the stale prior-turn shape and must not bind — its
// following late terminal must not resolve the successor. (A retired-id
// start observed AFTER the successor's running patch is the successor's own
// start under a backend that recycles turn ids; the front-tier corpus
// fixtures exercise that path.)
func TestLatePriorTurnStartBeforeRunningPatchIsIgnored(t *testing.T) {
	first := sendRule(t,
		statePatch(t, statusRunning, reasonPromptStarted),
		sessionEvent(t, turn.KindTurnStarted, fixTurnID),
		sessionEvent(t, turn.KindTurnCompleted, fixTurnID),
	)
	first.Times = 1

	second := sendRule(t,
		sessionEvent(t, turn.KindTurnStarted, fixTurnID),
		sessionEvent(t, turn.KindTurnCompleted, fixTurnID),
		statePatchRev(t, statusRunning, reasonPromptStarted, 3),
	)
	second.Times = 1

	sc := &fake.Scenario{
		Name: "turn-late-prior-start",
		OnRequest: []fake.RequestRule{
			first,
			second,
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}

	ex := startTurnExchange(t, sc)

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	if _, err := ex.machine.Run(ctx, "first turn"); err != nil {
		t.Fatalf("first Run: %v", err)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	reason, err := ex.machine.Run(ctx, "second turn")
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("second Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	// The late start must not have bound: had it, the late terminal would
	// have resolved this turn by the terminal event instead of the quiet
	// fallback.
	got := ex.spy.snapshot()

	if len(got) != 2 {
		t.Fatalf("resolution hook fired %d times across two turns, want exactly 2: %+v", len(got), got)
	}

	if got[1].Reason != turn.StopCompleted || got[1].Signal != turn.SignalQuietWindow {
		t.Fatalf("second turn resolution = %+v, want completed/quiet-window "+
			"(the stale prior-turn start must not bind the successor)", got[1])
	}

	ex.finish(t)
}

// TestIdlePatchBindingRequiresLaterRevision pins the revision/status/reason
// idle gate: a leaving-running patch completes the turn ONLY when both
// revisions are present and the idle revision is strictly greater than the
// revision of the running patch that armed the turn, and never when it
// carries another turn's stop outcome (cancelled/session_stopped) while the
// turn is still running.
func TestIdlePatchBindingRequiresLaterRevision(t *testing.T) {
	variants := []struct {
		name     string
		scenario func(*testing.T) *fake.Scenario
		signal   turn.SignalKind
	}{
		{
			name: "own later revision resolves",
			scenario: func(t *testing.T) *fake.Scenario {
				t.Helper()

				return completionScenario(t,
					statePatchRev(t, statusRunning, reasonPromptStarted, 1),
					statePatchRev(t, statusIdle, reasonPromptCompleted, 2),
				)
			},
			signal: turn.SignalIdlePatch,
		},
		{
			name: "equal revision falls through to the quiet window",
			scenario: func(t *testing.T) *fake.Scenario {
				t.Helper()

				return completionScenario(t,
					statePatchRev(t, statusRunning, reasonPromptStarted, 2),
					statePatchRev(t, statusIdle, reasonPromptCompleted, 2),
				)
			},
			signal: turn.SignalQuietWindow,
		},
		{
			name: "absent revision falls through to the quiet window",
			scenario: func(t *testing.T) *fake.Scenario {
				t.Helper()

				return completionScenario(t,
					statePatchRev(t, statusRunning, reasonPromptStarted, 0),
					statePatchRev(t, statusIdle, reasonPromptCompleted, 0),
				)
			},
			signal: turn.SignalQuietWindow,
		},
		{
			name: "cancelled stop outcome while running is ignored",
			scenario: func(t *testing.T) *fake.Scenario {
				t.Helper()

				return completionScenario(t,
					statePatchRev(t, statusRunning, reasonPromptStarted, 1),
					statePatchRev(t, statusCancelled, reasonSessionStopped, 2),
				)
			},
			signal: turn.SignalQuietWindow,
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			runCompletionVariant(t, v.name, v.scenario(t), turn.StopCompleted, v.signal)
		})
	}
}
