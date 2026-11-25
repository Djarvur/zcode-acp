package turn_test

// Cancel-outcome contract tests (quick task 261125-s7k): a client cancel is
// answered by the turn it interrupted — as StopCancelled — whatever signal the
// backend reports for the stop, and within a bounded grace when it reports
// none at all.
//
// Why this is load-bearing (Orca 1.4.274, src/main/acp/turns/
// acp-structured-turns.ts): a steer queues the new message, sends
// session/cancel and dispatches the new prompt only when the ACTIVE
// session/prompt request settles — "Steers wait for the active prompt to
// answer cancellation." The real desktop reports the stop as turn.failed
// (field log 2026-11-25 17:20:29.181: "the prior turn emitted turn.failed
// (cancelled)"), which the machine used to map to StopFailed — the front
// then answered -32603 instead of stopReason cancelled — and a silent backend
// left the prompt open behind the 7.5 s quiet-window floor. Both shapes hold
// the client's steer queue: the current answer stops, the new one never
// starts.
//
// The locks below ride the shared turn-tier fake exchange with a watcher
// whose quiet window is far longer than each test bound, so only the signal
// or grace path under test can resolve the turn.

import (
	"context"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Wire keys this file uses beyond the shared machine_test set (goconst: one
// definition, verbatim spelling).
const (
	wireKeyStatus   = "status"
	wireKeyEventSeq = "eventSeq"
	wireKeyVersion  = "version"
)

// longQuietWatcher is the quiet-window config these tests inject: a floor of
// 2 × 10 s keeps the watcher's settle (the production fallback) out of every
// test bound, so the resolution under test is the only one that can fire.
const longQuietWatcher = 10000

// stopOutcomeScenario scripts one turn whose session/stop rule emits the
// supplied stop-outcome effects (none for the silent-stop arm).
func stopOutcomeScenario(t *testing.T, effects ...fake.SideEffect) *fake.Scenario {
	t.Helper()

	return &fake.Scenario{
		Name: "turn-cancel-outcome",
		OnRequest: []fake.RequestRule{
			sendRule(t,
				statePatch(t, statusRunning, reasonPromptStarted),
				sessionEvent(t, turn.KindTurnStarted, fixTurnID),
			),
			{
				Method:      zcode.MethodSessionStop,
				Reply:       &fake.Reply{Result: rawOf(t, map[string]any{})},
				SideEffects: effects,
			},
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// v4TelemetryFailed is the v4 telemetry terminal carrying the failed status
// (the dialect's failure vocabulary — the real carrier may report the stop
// this way).
func v4TelemetryFailed(t *testing.T) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodV4Telemetry, map[string]any{
		wireKeyKind:      turn.KindTurnTerminal,
		wireKeyTurnID:    fixTurnID,
		wireKeyStatus:    "failed",
		wireKeyEventSeq:  2,
		wireKeySessionID: fixSessionID,
		wireKeyVersion:   1,
	})
}

// startOutcomeExchange wires the turn-tier exchange with the long-quiet
// watcher and the caller's machine options.
func startOutcomeExchange(
	t *testing.T, sc *fake.Scenario, opts ...turn.MachineOption,
) *turnExchange {
	t.Helper()

	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	serveDone := startFake(fakeEnd, sc)

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	m := turn.NewMachine(conn, fixSessionID, turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        longQuietWatcher,
		FloorMultiple:  2,
		PollIntervalMs: longQuietWatcher,
	}), opts...)

	spy := &resolutionSpy{}
	m.OnResolve = spy.record

	return &turnExchange{conn: conn, machine: m, spy: spy, serveDone: serveDone}
}

// cancelRunningTurn starts the turn, waits for its ack, and cancels it.
func cancelRunningTurn(ctx context.Context, t *testing.T, ex *turnExchange) <-chan runResult {
	t.Helper()

	runDone := runAsync(ctx, ex.machine, "Reply with exactly: OK")
	waitAck(t, ex.machine)

	ex.machine.Cancel()

	return runDone
}

// TestCancelResolvesCancelledOnFailedTerminal locks the real-desktop cancel
// outcome: turn.failed is the stop's observed result once the client asked,
// so the turn answers StopCancelled — never StopFailed, which the front would
// surface to the client as an internal error instead of the ACP cancelled
// stopReason.
func TestCancelResolvesCancelledOnFailedTerminal(t *testing.T) {
	t.Parallel()

	t.Run("session event turn.failed", func(t *testing.T) {
		t.Parallel()

		ex := startOutcomeExchange(t, stopOutcomeScenario(t,
			sessionEvent(t, turn.KindTurnFailed, fixTurnID)))

		ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
		defer cancel()

		got := <-cancelRunningTurn(ctx, t, ex)

		if got.err != nil {
			t.Fatalf("Run after Cancel returned error %v, want the cancelled outcome", got.err)
		}

		if got.reason != turn.StopCancelled {
			t.Fatalf("Run reason = %q, want %q (a failed terminal is the stop's outcome)",
				got.reason, turn.StopCancelled)
		}

		assertResolution(t, ex.spy, turn.StopCancelled, turn.SignalCancel)

		ex.finish(t)
	})

	t.Run("v4 telemetry turn.terminal failed", func(t *testing.T) {
		t.Parallel()

		ex := startOutcomeExchange(t, stopOutcomeScenario(t,
			v4TelemetryFailed(t)))

		ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
		defer cancel()

		got := <-cancelRunningTurn(ctx, t, ex)

		if got.err != nil {
			t.Fatalf("Run after Cancel returned error %v, want the cancelled outcome", got.err)
		}

		if got.reason != turn.StopCancelled {
			t.Fatalf("Run reason = %q, want %q (a failed terminal is the stop's outcome)",
				got.reason, turn.StopCancelled)
		}

		assertResolution(t, ex.spy, turn.StopCancelled, turn.SignalCancel)

		ex.finish(t)
	})
}

// TestCancelRacingCompletedStaysCompleted locks the other side of the
// honesty rule: a turn that genuinely completed keeps StopCompleted even
// when the cancel raced its terminal — the machine must not claim a
// cancellation that did not happen.
func TestCancelRacingCompletedStaysCompleted(t *testing.T) {
	t.Parallel()

	ex := startOutcomeExchange(t, stopOutcomeScenario(t,
		sessionEvent(t, turn.KindTurnCompleted, fixTurnID)))

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	got := <-cancelRunningTurn(ctx, t, ex)

	if got.err != nil {
		t.Fatalf("Run after Cancel returned error %v, want the completed outcome", got.err)
	}

	if got.reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q (the turn really completed)", got.reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	ex.finish(t)
}

// TestCancelGraceResolvesWithoutSignal locks the bounded answer: a backend
// that acknowledges the stop with no turn-scoped signal at all may not hold
// the client's cancelled prompt behind the quiet-window fallback. The grace
// resolves the client-requested outcome.
func TestCancelGraceResolvesWithoutSignal(t *testing.T) {
	t.Parallel()

	ex := startOutcomeExchange(t, stopOutcomeScenario(t), turn.WithCancelGrace(40*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	started := time.Now()
	got := <-cancelRunningTurn(ctx, t, ex)

	if got.err != nil {
		t.Fatalf("Run after Cancel returned error %v, want the cancelled outcome", got.err)
	}

	if got.reason != turn.StopCancelled {
		t.Fatalf("Run reason = %q, want %q", got.reason, turn.StopCancelled)
	}

	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("cancel answered after %v, want the configured grace to bound it", elapsed)
	}

	assertResolution(t, ex.spy, turn.StopCancelled, turn.SignalCancel)

	ex.finish(t)
}

// TestCancelGraceNeverDoubleResolves locks exactly-once around the grace
// timer: a turn resolved by its terminal before the grace elapses never
// resolves again, and the production default stays the documented two
// seconds.
func TestCancelGraceNeverDoubleResolves(t *testing.T) {
	t.Parallel()

	ex := startOutcomeExchange(t,
		stopOutcomeScenario(t, sessionEvent(t, turn.KindTurnFailed, fixTurnID)),
		turn.WithCancelGrace(30*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	got := <-cancelRunningTurn(ctx, t, ex)

	if got.err != nil {
		t.Fatalf("Run after Cancel returned error %v, want the cancelled outcome", got.err)
	}

	// Past the grace and the previous resolution: nothing more may resolve.
	time.Sleep(120 * time.Millisecond)

	if resolutions := ex.spy.snapshot(); len(resolutions) != 1 {
		t.Fatalf("resolution hook fired %d times, want exactly 1 around the grace", len(resolutions))
	}

	if turn.DefaultCancelGrace != 2*time.Second {
		t.Fatalf("DefaultCancelGrace = %v, want the documented 2s", turn.DefaultCancelGrace)
	}

	ex.finish(t)
}

// TestCancelGraceDoesNotResolveIdleTurn locks the guard: a cancel with no
// local turn writes the stop (the machine's existing no-op arm keeps it
// silent on the resolution path) and never resolves a later turn through a
// stale timer.
func TestCancelGraceDoesNotResolveIdleTurn(t *testing.T) {
	t.Parallel()

	ex := startOutcomeExchange(t, stopOutcomeScenario(t), turn.WithCancelGrace(20*time.Millisecond))

	ex.machine.Cancel()

	time.Sleep(80 * time.Millisecond)

	if resolutions := ex.spy.snapshot(); len(resolutions) != 0 {
		t.Fatalf("idle-machine cancel resolved %d turns, want 0", len(resolutions))
	}

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	runDone := runAsync(ctx, ex.machine, "fresh turn")
	waitAck(t, ex.machine)

	ex.machine.Cancel()

	got := <-runDone

	if got.err != nil || got.reason != turn.StopCancelled {
		t.Fatalf("fresh turn after idle cancel = (%q, %v), want cancelled", got.reason, got.err)
	}

	ex.finish(t)
}
