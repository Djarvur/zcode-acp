package turn_test

// Orphan-recovery contract tests (quick task 261125-r6o, Task 3): the
// 2026-11-25 field incident's third defect — the wrapper falsely freed its
// front-wide slot and forwarded 867 client retries into a still-running
// backend turn for 79 minutes because a slot-free -32010 mapped to the
// retryable -32003 session_busy class.
//
// The machine cannot see other machines; the FRONT-wide slot is the proof
// that no local turn is active, so orphan recovery is an opt-in machine flag
// the Front enables on every prompt machine. With it on, a numeric -32010
// send rejection triggers exactly one bypass-lane stop, a bounded wait for
// EXPLICITLY session-scoped clear evidence, and exactly one retry. A second
// rejection is the distinct non-retryable ErrOrphanedBackend. A terminal
// observed during the wait is retired so a late duplicate cannot bind or
// falsely complete the retried turn.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	// orphanTurnID is the backend orphan's turnId (the one the bounded wait
	// retires) and retryTurnID the retried turn's own id.
	orphanTurnID = "turn_orphan"
	retryTurnID  = "turn_retry"

	// foreignSID is a different conversation's session id — evidence scoped
	// to it must never clear this machine's wait.
	foreignSID = "sess_foreign_orphan"
)

// orphanRefusalRule answers one session/send with the numeric -32010
// concurrency refusal (the desktop's "active prompt exists" shape; message
// text is deliberately non-contract).
func orphanRefusalRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionSend,
		Reply: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeConcurrency,
			Message: "active prompt exists",
		}},
	}
}

// orphanStopRule answers the orphan stop, optionally emitting clear
// evidence first (side effects precede the reply).
func orphanStopRule(t *testing.T, effects ...fake.SideEffect) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method:      zcode.MethodSessionStop,
		Reply:       &fake.Reply{Result: rawOf(t, map[string]any{})},
		SideEffects: effects,
	}
}

// orphanStatePatch is a session-scoped state.updated patch for an arbitrary
// session id (the clear-evidence and foreign-evidence shapes).
func orphanStatePatch(t *testing.T, sid, status, reason string, revision int) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodStateUpdated, map[string]any{
		"patch":          map[string]any{wireKeyStatus: status},
		"reason":         reason,
		"revision":       revision,
		"scope":          "session",
		wireKeySessionID: sid,
		wireKeyType:      "state.updated",
	})
}

// orphanTerminal is a session-scoped terminal event for an arbitrary session
// id and turnId.
func orphanTerminal(t *testing.T, sid, kind, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodSessionEvent, map[string]any{
		wireKeyType:      kind,
		wireKeySessionID: sid,
		wireKeyTurnID:    turnID,
		wireKeySeq:       4,
	})
}

// orphanScenario assembles one orphan-recovery script with the standard
// unknown-method reply and EOF drain.
func orphanScenario(t *testing.T, rules ...fake.RequestRule) *fake.Scenario {
	t.Helper()

	onRequest := make([]fake.RequestRule, 0, len(rules)+1)
	onRequest = append(onRequest, rules...)
	onRequest = append(onRequest, messagesRule(t))

	return &fake.Scenario{
		Name:      "turn-orphan",
		OnRequest: onRequest,
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// orphanWireIndexes returns the host→fake line indexes of one wire method.
func orphanWireIndexes(tr fake.Transcript, method string) []int {
	needle := `"method":"` + method + `"`

	indexes := []int{}

	for i, line := range tr.Lines {
		if line.Dir == fake.DirIn && strings.Contains(line.JSON, needle) {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

// runOrphan drives one orphan-recovery turn on an armed machine.
func runOrphan(t *testing.T, ex *turnExchange, clearTimeout time.Duration) (turn.StopReason, error) {
	t.Helper()

	ex.machine.OrphanRecovery = true
	ex.machine.OrphanClearTimeout = clearTimeout

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	return ex.machine.Run(ctx, "orphan prompt")
}

// TestOrphanedBackendSendSelfHeals: a slot-free -32010 triggers exactly one
// stop, the session-scoped idle patch clears the wait, and the single retry
// completes normally — the wire order is send, stop, send.
func TestOrphanedBackendSendSelfHeals(t *testing.T) {
	first := orphanRefusalRule()
	first.Times = 1

	sc := orphanScenario(t,
		first,
		orphanStopRule(t, orphanStatePatch(t, fixSessionID, statusIdle, reasonPromptCompleted, 2)),
		sendRule(t,
			orphanStatePatch(t, fixSessionID, statusRunning, reasonPromptStarted, 3),
			orphanTerminal(t, fixSessionID, turn.KindTurnCompleted, retryTurnID),
		),
	)

	ex := startTurnExchange(t, sc)

	reason, err := runOrphan(t, ex, 2*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	tr := finishWithTranscript(t, ex)

	sends := orphanWireIndexes(tr, zcode.MethodSessionSend)
	stops := orphanWireIndexes(tr, zcode.MethodSessionStop)

	if len(sends) != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2 (the rejected send + the retry)", len(sends))
	}

	if len(stops) != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1", len(stops))
	}

	if sends[0] >= stops[0] || stops[0] >= sends[1] {
		t.Fatalf("wire order = send@%v stop@%v, want send, stop, send", sends, stops)
	}
}

// TestOrphanRecoverySecondRejectionIsNotRetryable: a retry refused again is
// the distinct non-retryable ErrOrphanedBackend (never ErrTurnActive), with
// exactly two sends and one stop on the wire.
func TestOrphanRecoverySecondRejectionIsNotRetryable(t *testing.T) {
	sc := orphanScenario(t,
		orphanRefusalRule(), // unlimited: both sends are refused
		orphanStopRule(t),   // no clear evidence
	)

	ex := startTurnExchange(t, sc)

	_, err := runOrphan(t, ex, 40*time.Millisecond)
	if !errors.Is(err, turn.ErrOrphanedBackend) {
		t.Fatalf("Run error = %v, want ErrOrphanedBackend", err)
	}

	if errors.Is(err, turn.ErrTurnActive) {
		t.Fatalf("Run error = %v, must not match ErrTurnActive (the retryable session_busy class)", err)
	}

	tr := finishWithTranscript(t, ex)

	if sends := len(orphanWireIndexes(tr, zcode.MethodSessionSend)); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2", sends)
	}

	if stops := len(orphanWireIndexes(tr, zcode.MethodSessionStop)); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1", stops)
	}
}

// TestOrphanClearWaitTimeoutStillRetriesOnce: missing clear evidence does
// not suppress the single retry — the bounded wait expires, the retry runs,
// and the turn completes.
func TestOrphanClearWaitTimeoutStillRetriesOnce(t *testing.T) {
	first := orphanRefusalRule()
	first.Times = 1

	sc := orphanScenario(t,
		first,
		orphanStopRule(t), // no evidence: the bound decides
		sendRule(t,
			orphanStatePatch(t, fixSessionID, statusRunning, reasonPromptStarted, 3),
			orphanTerminal(t, fixSessionID, turn.KindTurnCompleted, retryTurnID),
		),
	)

	ex := startTurnExchange(t, sc)

	started := time.Now()
	reason, err := runOrphan(t, ex, 50*time.Millisecond)
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	if elapsed < 45*time.Millisecond || elapsed >= 2*time.Second {
		t.Fatalf("Run elapsed = %v, want the bounded wait (~50 ms) then the retry", elapsed)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	tr := finishWithTranscript(t, ex)

	if sends := len(orphanWireIndexes(tr, zcode.MethodSessionSend)); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2", sends)
	}

	if stops := len(orphanWireIndexes(tr, zcode.MethodSessionStop)); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1", stops)
	}
}

// TestOrphanClearWaitTerminalEvidenceRetiresOrphan: the orphan's own
// session-scoped terminal ends the wait early AND is retired, so a late
// duplicate cannot bind the unbound retried turn (which would resolve
// StopFailed). The retried turn's own terminal resolves exactly once.
func TestOrphanClearWaitTerminalEvidenceRetiresOrphan(t *testing.T) {
	first := orphanRefusalRule()
	first.Times = 1

	sc := orphanScenario(t,
		first,
		orphanStopRule(t, orphanTerminal(t, fixSessionID, turn.KindTurnFailed, orphanTurnID)),
		sendRule(t,
			orphanStatePatch(t, fixSessionID, statusRunning, reasonPromptStarted, 3),
			orphanTerminal(t, fixSessionID, turn.KindTurnFailed, orphanTurnID),   // late duplicate
			orphanTerminal(t, fixSessionID, turn.KindTurnCompleted, retryTurnID), // the retried turn's own
			orphanTerminal(t, fixSessionID, turn.KindTurnFailed, orphanTurnID),   // one more duplicate
		),
	)

	ex := startTurnExchange(t, sc)

	started := time.Now()
	reason, err := runOrphan(t, ex, 2*time.Second)
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q (a late orphan duplicate must not resolve it failed)",
			reason, turn.StopCompleted)
	}

	// The terminal evidence ended the wait; the generous bound never
	// elapsed.
	if elapsed >= 1500*time.Millisecond {
		t.Fatalf("Run elapsed = %v — the wait did not end on the terminal evidence", elapsed)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	tr := finishWithTranscript(t, ex)

	if sends := len(orphanWireIndexes(tr, zcode.MethodSessionSend)); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2", sends)
	}

	if stops := len(orphanWireIndexes(tr, zcode.MethodSessionStop)); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1", stops)
	}
}

// TestOrphanClearWaitIgnoresForeignSessionEvidence: a foreign-session
// terminal and a foreign idle patch are not clear evidence — the wait runs
// its bound, then the single retry completes with the retried turn's own
// terminal.
func TestOrphanClearWaitIgnoresForeignSessionEvidence(t *testing.T) {
	first := orphanRefusalRule()
	first.Times = 1

	sc := orphanScenario(t,
		first,
		orphanStopRule(t,
			orphanTerminal(t, foreignSID, turn.KindTurnFailed, orphanTurnID),
			orphanStatePatch(t, foreignSID, statusIdle, reasonPromptCompleted, 2),
		),
		sendRule(t,
			orphanStatePatch(t, fixSessionID, statusRunning, reasonPromptStarted, 3),
			orphanTerminal(t, fixSessionID, turn.KindTurnCompleted, retryTurnID),
		),
	)

	ex := startTurnExchange(t, sc)

	started := time.Now()
	reason, err := runOrphan(t, ex, 80*time.Millisecond)
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	// The foreign evidence did not clear the wait early (an early clear
	// would retry within milliseconds) and no foreign terminal resolved the
	// turn.
	if elapsed < 70*time.Millisecond || elapsed >= 2*time.Second {
		t.Fatalf("Run elapsed = %v, want the full 80 ms bound (foreign evidence cleared it early)", elapsed)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	tr := finishWithTranscript(t, ex)

	if sends := len(orphanWireIndexes(tr, zcode.MethodSessionSend)); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2", sends)
	}

	if stops := len(orphanWireIndexes(tr, zcode.MethodSessionStop)); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1", stops)
	}
}
