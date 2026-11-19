// Package turn implements the turn state machine (BRIDGE-04, D-02): it
// lifts one fire-and-forget session/send into a synchronous turn resolved
// exactly once by the ordered signal union — a terminal event first, then
// the idle patch, then the quiet-window fallback (PITFALL 8: the machine
// owns the completion; it never synthesizes a signal — no heuristic
// resolution on absence of evidence).
//
// Wire truth: docs/protocol/app-server-dialect.md §5 (the shipped 3.18.2
// surface). The completion half of §5 is V-SRC-only on the real wire, so
// the terminal classifier is carrier-tolerant by design (RESEARCH Open
// Question 1 resolution): it matches turnId + terminal kind across the
// known carriers; live reconciliation is Phase 5/6 work.
package turn

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// StopReason is the terminal outcome of one synchronous turn (D-02): the
// value Run/Wait answer the (future Phase 4) session/prompt with, resolved
// exactly once per turn.
type StopReason string

// Stop reasons (RESEARCH Pattern 6). Cancelled is resolved by the
// bypass-lane cancel path (03-08); the constant is declared now so the
// completion surface is complete.
const (
	StopCompleted StopReason = "completed"
	StopFailed    StopReason = "failed"
	StopCancelled StopReason = "cancelled"
	StopError     StopReason = "error"
)

// SignalKind names the signal that resolved a turn — the resolving-signal
// record (D-05-style observability: the honest answer to "how did this
// turn end").
type SignalKind string

const (
	// SignalTerminalEvent — a turnId-matched terminal event
	// (turn.completed/turn.failed) across the known carriers; the D-02
	// primary completion signal.
	SignalTerminalEvent SignalKind = "terminal-event"

	// SignalIdlePatch — a state.updated patch whose status left "running";
	// the D-02 secondary (dialect §5 reliability column: do not rely on it
	// alone).
	SignalIdlePatch SignalKind = "idle-patch"

	// SignalQuietWindow — the poll watcher's quiet-window settlement; the
	// D-02 fallback and the only completion path proven end-to-end
	// (dialect §5 :141).
	SignalQuietWindow SignalKind = "quiet-window"

	// SignalCancel — the bypass-lane session/stop path (declared now; its
	// transition logic lands with 03-08's cancel handling).
	SignalCancel SignalKind = "cancel"
)

// Resolution is the exactly-once completion record: the StopReason, the
// resolving signal, and when the resolution fired.
type Resolution struct {
	Reason StopReason
	Signal SignalKind
	At     time.Time
}

// Terminal and turn-start kind literals across the known carriers (dialect
// §5 signal tables): the session/event and v4/telemetry families spell the
// kinds with dots, the computer-use operation family with hyphens — each
// carrying the turnId. The classifier matches turnId + kind, so the same
// terminal classification fires whichever carrier delivers the signal.
const (
	KindTurnStarted   = "turn.started" // session/event + v4 telemetry
	KindTurnCompleted = "turn.completed"
	KindTurnFailed    = "turn.failed"

	KindTurnStartedCU   = "turn-started" // computer-use operation events
	KindTurnCompletedCU = "turn-completed"
	KindTurnFailedCU    = "turn-failed"
)

// Carrier method constants this package classifies (dialect §5 turn-start
// signal table). They are not codec method-map entries, so they stay local
// to the classifier instead of growing the shared codec surface.
const (
	MethodV4Telemetry         = "v4/telemetry/event"
	MethodComputerUseOperation = "computer-use/operation-event"
)

// machineState is the turn lifecycle (RESEARCH Pattern 6): idle → running →
// cancelling → done. The cancelling transitions land with 03-08's cancel;
// the state is declared now per the plan.
type machineState string

const (
	stateIdle       machineState = "idle"
	stateRunning    machineState = "running"
	stateCancelling machineState = "cancelling" // transitions land with 03-08
	stateDone       machineState = "done"
)

// ErrTurnActive is the wrapper-side concurrency guard (D-02): a second
// concurrent session/send while a turn is running is rejected with a clean
// error, never silently queued — the wrapper-side analog of the dialect §6
// -32010 concurrency rejection (match the condition, not any message text).
var ErrTurnActive = errors.New("turn: a prompt is already running for this session")

// Machine lifts fire-and-forget session/send turns into synchronous
// completions for ONE app-server sessionId on ONE shared Conn (D-04: one
// app-server child per wrapper process; the machine binds one session; ACP
// sessionId maps 1:1). The machine owns the completion (PITFALL 8): every
// inbound envelope funnels through ONE observe path, and the turn resolves
// exactly once.
type Machine struct {
	conn    *zcode.Conn
	sessID  string
	watcher TurnWatcher

	// OnResolve, when non-nil, is invoked exactly once per turn at the
	// resolve point — the observable face of the exactly-once contract
	// (the resolution counter/hook the tests observe). Must not block: it
	// runs on the observing goroutine.
	OnResolve func(Resolution)

	mu          sync.Mutex
	state       machineState
	turnID      string
	runningSeen bool
	resolved    bool
	resolution  Resolution
	resolvedCh  chan struct{}
	ackAt       time.Time
}

// NewMachine returns a machine bound to sessionID on conn with the given
// watcher, and takes over conn's single OnNotification hook — every inbound
// envelope funnels into the machine's one observe path. Assign happens
// before any frame flows (the Conn hook contract).
func NewMachine(conn *zcode.Conn, sessionID string, watcher TurnWatcher) *Machine {
	return &Machine{
		conn:       conn,
		sessID:     sessionID,
		watcher:    watcher,
		state:      stateIdle,
		resolvedCh: make(chan struct{}),
	}
}

// Run lifts one fire-and-forget session/send into a synchronous turn: it
// returns a StopReason exactly once, after the backend emitted a completion
// signal — never before, never twice (PITFALL 8). A second concurrent Run
// on the same session is rejected ErrTurnActive (the -32010 guard,
// wrapper-side). RED stub: the contract surface is declared; the body
// lands with the GREEN commit (03-05 Task 2).
func (m *Machine) Run(ctx context.Context, content string) (StopReason, error) {
	if err := m.guardTurnLocked(); err != nil {
		return "", err
	}

	_, _ = ctx, content

	return "", errNotImplemented
}

// Wait returns the current turn's StopReason, blocking until the turn
// resolves; a second wait returns the SAME StopReason without re-resolving.
// RED stub: returns errNotImplemented.
func (m *Machine) Wait(ctx context.Context) (StopReason, error) {
	_ = ctx

	return "", errNotImplemented
}

// guardTurnLocked rejects a concurrent second send (D-02: the -32010
// concurrency guard is enforced wrapper-side, never silently queued).
// Declared with the RED stub; the GREEN Run reuses it.
func (m *Machine) guardTurnLocked() error {
	switch m.state {
	case stateRunning, stateCancelling:
		return ErrTurnActive
	case stateIdle, stateDone:
		return nil
	}

	return nil
}
