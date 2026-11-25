// Package turn implements the turn state machine (BRIDGE-04, D-02): it
// lifts one fire-and-forget session/send into a synchronous turn resolved
// exactly once by the ordered signal union — a terminal event first, then
// the idle patch, then the quiet-window fallback (PITFALL 8: the machine
// owns the completion; it never synthesizes a signal — no heuristic
// resolution on absence of evidence).
//
// Wire truth: docs/protocol/app-server-dialect.md §5 (the shipped 3.18.2
// surface). The terminal envelope IS observed live on 3.18.2
// (EXP-VM-QA02-family dumps, 2026-11-23): the real desktop announces the
// completion on computer-use/operation-event (kind "turn-completed") and
// v4/telemetry/event (kind "turn.terminal" with status "success"), and the
// phase-1 EXP-TURN-01 degradation (terminal never observed) is resolved.
// The classifier is carrier-tolerant by design (RESEARCH Open Question 1
// resolution): it matches turnId + terminal kind across the known carriers
// — the bundle-taxonomy kinds stay classified for other runtime families.
//
// Completion timing (QA-03 field blocker): a turn whose start was announced
// on a terminal-proven carrier (the two carriers above) has a terminal
// signal coming — the quiet-window fallback may not end such a turn while
// the veto budget holds, because the real desktop can think silently for
// seconds past the send-latency floor before the terminal lands (the
// 2026-11-22 field runs: end_turn at the floor, 0.1 s after the first
// delta, reply text cut off). The budget bounds the fallback honestly: a
// start-announced turn gone silent for the whole budget resolves over the
// quiet window after all (the EXP-APIKEY-CYCLE-02 truth — a completed turn
// may present no terminal at all).
package turn

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	// D-02 fallback (dialect §5 :141), kept for runtime families that
	// present no terminal surface (EXP-APIKEY-CYCLE-02: a genuinely
	// completed turn may present no terminal envelope at all) and for
	// start-announced turns whose promised terminal never arrives within
	// the veto budget.
	SignalQuietWindow SignalKind = "quiet-window"

	// SignalCancel — the turn resolved through the bypass-lane
	// session/stop path: the stop's observed outcome (the cancelled state
	// patch, or the quiet settle while cancelling) while the machine is in
	// the cancelling state.
	SignalCancel SignalKind = "cancel"

	// SignalDegraded — the turn resolved from a recorded host-obligation
	// failure rather than a completion signal (D-05 record-honesty): a
	// tracked-no-answer reverse-request deadline expired mid-turn, the
	// expiry is carried on Resolution.Err, and the resolution is the honest
	// degraded state — never completed/failed (the BRIDGE-04 prohibition:
	// no resolution without a supporting completion signal).
	SignalDegraded SignalKind = "degraded"

	// SignalStall — the no-progress watchdog resolved the turn: a RUNNING
	// turn made no real model/tool progress past the configured threshold,
	// the bypass-lane session/stop was sent first, and the turn resolved
	// StopError carrying the recorded stall cause. A cancelling turn is
	// outside the watchdog's scope.
	SignalStall SignalKind = "stall"
)

// Watchdog budget defaults (the 2026-11-25 field incident: a 429 retry
// backoff kept one turn silent ~50 minutes and the wrapper had no progress
// bound). Base is the no-model-progress allowance; the larger tool allowance
// applies while a tool is active.
const (
	DefaultStallTimeout     = 15 * time.Minute
	DefaultStallToolTimeout = 30 * time.Minute

	// stallToolBudgetMultiple scales the tool allowance to an explicit base
	// override when no explicit tool override is configured (test
	// ergonomics: the tool budget stays a deterministic multiple of base).
	stallToolBudgetMultiple = 2
)

// Resolution is the exactly-once completion record: the StopReason, the
// resolving signal, and when the resolution fired. Err carries the recorded
// degradation cause for a degraded resolution (SignalDegraded — D-05
// record-honesty: the expiry is observable, never swallowed) and is nil for
// clean completions.
type Resolution struct {
	Reason StopReason
	Signal SignalKind
	At     time.Time
	Err    error
}

// Terminal and turn-start kind literals across the known carriers (dialect
// §5 signal tables): the session/event and v4/telemetry families spell the
// kinds with dots, the computer-use operation family with hyphens — each
// carrying the turnId. The classifier matches turnId + kind, so the same
// terminal classification fires whichever carrier delivers the signal.
//
// Live truth (EXP-VM-QA02-family dumps, 2026-11-23): the real 3.18.2
// desktop completes turns with the computer-use "turn-completed" literal
// AND the v4 telemetry "turn.terminal" literal (status/resultType
// "success") — the bundle taxonomy's "turn.completed"/"turn.failed" remain
// classified for other runtime families (D-02: keep both signal families
// armed).
const (
	KindTurnStarted   = "turn.started" // session/event + v4 telemetry
	KindTurnCompleted = "turn.completed"
	KindTurnFailed    = "turn.failed"

	// KindTurnTerminal is the real 3.18.2 desktop's v4/telemetry terminal
	// kind (raw3.out:84): the outcome rides the params "status" field —
	// "success" (the only value observed live) completes the turn; the
	// failure vocabulary resolves it failed.
	KindTurnTerminal = "turn.terminal"

	KindTurnStartedCU   = "turn-started" // computer-use operation events
	KindTurnCompletedCU = "turn-completed"
	KindTurnFailedCU    = "turn-failed"
)

// Carrier method constants this package classifies (dialect §5 turn-start
// signal table). They are not codec method-map entries, so they stay local
// to the classifier instead of growing the shared codec surface.
const (
	MethodV4Telemetry          = "v4/telemetry/event"
	MethodComputerUseOperation = "computer-use/operation-event"
)

// state.updated patch values the idle-patch latch consumes (dialect §5:
// only "running" was ever observed live — the recorded A4 assumption;
// "leaving running" implies running was seen). A cancelled status and a
// session_stopped reason are another turn's stop outcome: the stop was
// explicit, so the patch is the outcome of the turn the stop interrupted,
// never this running turn's completion.
const (
	statusRunningPatch        = "running"
	statusCancelledPatch      = "cancelled"
	reasonSessionStoppedPatch = "session_stopped"
)

// DefaultMaxQuietVetoes is the quiet-settle veto budget for start-announced
// turns (QA-03): how many consecutive quiet-window settlements — with NO
// intervening envelope observation, i.e. no evidence of life — a turn may
// veto while waiting for its promised terminal signal. Each vetoed settle
// is one full quiet window of the watcher's re-armed gate, so the default
// bounds the wait at ~2 minutes of TOTAL silence at the dialect defaults
// (2500 ms quiet) before the honest quiet fallback resolves. The real
// desktop's think pauses sit well inside it (the field runs: 4-5 s), while
// a start-announced turn whose runtime died mid-turn still ends.
const DefaultMaxQuietVetoes = 48

// machineState is the turn lifecycle (RESEARCH Pattern 6): idle → running →
// cancelling → done. The cancelling transitions land with 03-08's cancel;
// the state is declared now per the plan.
type machineState string

const (
	stateIdle       machineState = "idle"
	stateRunning    machineState = "running"
	stateCancelling machineState = "cancelling"
	stateDone       machineState = "done"
)

// ErrTurnActive is the -32010 concurrency guard mapped as a clean error
// (D-02, PITFALL 5): a second concurrent session/send while a turn is
// running is rejected with it wrapper-side — never silently queued — and
// the same sentinel surfaces when the SERVER refuses a send with dialect
// §6 CodeConcurrency (-32010). The mapping keys on the numeric code only;
// message text is never compared (dialect §6: messages are not a stable
// contract).
var ErrTurnActive = errors.New("turn: a prompt is already running for this session")

// ErrOrphanedBackend is the distinct non-retryable sentinel for a backend
// that still reports an active prompt AFTER the orphan-recovery stop and
// single retry: the front-wide slot proves no local turn was active, so the
// backend is stuck on an orphan and the client must not keep retrying the
// retryable -32003 session_busy class.
var ErrOrphanedBackend = errors.New(
	"turn: the backend still reports an active prompt after stop and retry")

// DefaultOrphanClearTimeout bounds the orphan-recovery wait for
// session-scoped clear evidence (a terminal or an idle state.updated patch)
// before the single retry proceeds.
const DefaultOrphanClearTimeout = 5 * time.Second

// errObligationExpired is the static degradation sentinel wrapped into a
// degraded Resolution.Err (D-05 record-honesty): a tracked-no-answer
// reverse-request deadline elapsed mid-turn with no reply written — the
// expiry is recorded, never swallowed.
var errObligationExpired = errors.New(
	"turn: host obligation deadline expired without an answer (recorded, never answered)")

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

	// MaxQuietVetoes overrides the quiet-settle veto budget for
	// start-announced turns (QA-03; DefaultMaxQuietVetoes when zero).
	// Assign before any turn runs, like OnResolve.
	MaxQuietVetoes int

	// cancelGrace is how long a cancelled turn may stay unresolved before
	// the client-requested cancellation is answered anyway (the ACP cancel
	// contract; DefaultCancelGrace when zero).
	cancelGrace time.Duration

	// cancelTimer is the armed cancel grace (guarded by mu): Cancel arms it
	// on the running→cancelling transition, resolveLocked stops it, and
	// finishTurn clears it between turns.
	cancelTimer *time.Timer

	// StallTimeout bounds a RUNNING turn's total no-progress silence
	// (DefaultStallTimeout when zero). Assign before any turn runs. The
	// watchdog fires only while the turn is running — a cancelling turn
	// keeps the existing quiet/cancel contract.
	StallTimeout time.Duration

	// StallToolTimeout is the larger allowance applied while a tool is
	// active (DefaultStallToolTimeout when zero). An explicit StallTimeout
	// with no StallToolTimeout scales the tool allowance to 2x base.
	StallToolTimeout time.Duration

	// OrphanRecovery enables the bounded orphan-recovery path: a numeric
	// -32010 session/send rejection triggers ONE bypass-lane stop, a bounded
	// wait for session-scoped clear evidence, and ONE retry. It is an opt-in
	// flag because a direct machine embedding cannot prove no LOCAL turn is
	// active; the front enables it on every machine it registers for prompts
	// (its front-wide slot is that proof). Default false keeps the -32010 ->
	// ErrTurnActive -> -32003 mapping.
	OrphanRecovery bool

	// OrphanClearTimeout bounds the orphan-recovery clear wait
	// (DefaultOrphanClearTimeout when zero). Assign before any turn runs.
	OrphanClearTimeout time.Duration

	mu          sync.Mutex
	state       machineState
	turnID      string
	runningSeen bool
	// retiredTurnID remembers the turnId the last turn resolved with (or a
	// late duplicate terminal observed after resolution). It PERSISTS across
	// turns: beginTurnLocked never clears it, so a prompt accepted after a
	// prior turn resolved cannot be bound by that turn's late terminal — the
	// 2026-11-25 field incident's false completion.
	retiredTurnID string
	// runningRevision/runningRevisionSeen/runningReason record the running
	// state.updated patch that armed the idle latch: an idle patch completes
	// the turn only when both revisions are present and the idle revision is
	// strictly greater (dialect §5: idle patches are not reliably
	// turn-scoped, so ordering must be proven by the session-scoped revision).
	runningRevision     int64
	runningRevisionSeen bool
	runningReason       string

	// The no-progress watchdog state (per turn): lastProgress is the last
	// real model/tool progress instant, toolActive the tool-allowance
	// selector, stallWake the capacity-1 poke channel, and the messages
	// pair the poll-surface content baseline (a stable echo is not
	// progress).
	lastProgress    time.Time
	toolActive      bool
	stallWake       chan struct{}
	messagesSeen    bool
	lastMessagesRaw json.RawMessage

	// The orphan-recovery wait state (per turn): while orphanWaiting, the
	// one observe path inspects inbound envelopes ONLY for explicitly
	// session-scoped clear evidence and signals orphanClear (capacity 1).
	orphanWaiting bool
	orphanClear   chan struct{}

	startProven bool
	quietVetoes int
	resolved    bool
	resolution  Resolution
	resolvedCh  chan struct{}
	ackAt       time.Time

	// removeNotificationObserver / removeRouterObserver deregister the
	// machine from the conn's two ordered fan-outs (WR-07: exposed for
	// session/close teardown — plan 05-05 calls them). Set by NewMachine;
	// nil without a conn. CloseObservers drives both, idempotently.
	removeNotificationObserver func()
	removeRouterObserver       func()
}

// DefaultCancelGrace is the production bound on a cancelled turn's answer:
// long enough to let the live stop→terminal latency (~300 ms in the field
// log, 2026-11-25 17:20:28.882 → 17:20:29.181) resolve the turn through its
// own signal, short enough that a client block on the prompt answer (Orca
// holds a steer until the active prompt settles) never sees a stall.
const DefaultCancelGrace = 2 * time.Second

// MachineOption configures a Machine at construction, before any turn runs.
type MachineOption func(*Machine)

// WithCancelGrace overrides DefaultCancelGrace. Zero keeps the default; tests
// inject milliseconds so no test waits on the production bound.
func WithCancelGrace(d time.Duration) MachineOption {
	return func(m *Machine) { m.cancelGrace = d }
}

// NewMachine returns a machine bound to sessionID on conn with the given
// watcher, and wires the machine's one observation surface: every inbound
// notification envelope funnels into it via conn.AddObserver and
// reverse-request router policy events via conn.AddRouterObserver — both
// ordered fan-outs (WR-01: a second machine no longer steals the first's
// router hook). NewMachine keeps BOTH remove funcs on the Machine; plan
// 05-05's session/close teardown calls them (WR-07: teardown can finally
// release observers). Registration happens before any frame flows (the
// Conn hook contract).
func NewMachine(conn *zcode.Conn, sessionID string, watcher TurnWatcher, opts ...MachineOption) *Machine {
	m := &Machine{
		conn:       conn,
		sessID:     sessionID,
		watcher:    watcher,
		state:      stateIdle,
		resolvedCh: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(m)
	}

	if conn != nil {
		m.removeNotificationObserver = conn.AddObserver(m.ObserveEnvelope)
		m.removeRouterObserver = conn.AddRouterObserver(m.observeRouterEvent)
	}

	return m
}

// CloseObservers deregisters the machine from both conn fan-outs
// (WR-07 teardown). Idempotent: repeated or double calls are no-ops, so
// the teardown owner can call it without owning the only reference.
func (m *Machine) CloseObservers() {
	if m.removeNotificationObserver != nil {
		m.removeNotificationObserver()
		m.removeNotificationObserver = nil
	}

	if m.removeRouterObserver != nil {
		m.removeRouterObserver()
		m.removeRouterObserver = nil
	}
}

// Run lifts one fire-and-forget session/send into a synchronous turn: it
// returns a StopReason exactly once, after the backend emitted a completion
// signal — never before, never twice (PITFALL 8). The send's fast ack
// ({accepted, sessionId, stateRevision} — EXP-TURN-01.jsonl:36) starts the
// quiet-window floor clock; the watcher runs for the session until the turn
// resolves and is joined before Run returns. A second concurrent Run is
// rejected ErrTurnActive (the -32010 guard, wrapper-side); a send that does
// proceed and is refused -32010 by the server surfaces the same mapped
// error. A transport failure or ctx cancellation returns an error — never a
// synthesized StopReason (the machine never resolves on absence of evidence).
func (m *Machine) Run(ctx context.Context, content string) (StopReason, error) {
	m.mu.Lock()

	if err := m.guardTurnLocked(); err != nil {
		m.mu.Unlock()

		return "", err
	}

	m.beginTurnLocked()
	m.mu.Unlock()

	if sendErr := m.sendTurn(ctx, content); sendErr != nil {
		if retried, retryErr := m.recoverOrphanedSend(ctx, content, sendErr); retried {
			sendErr = retryErr
		}

		if sendErr != nil {
			m.finishTurn()

			return "", m.mapSendError(sendErr)
		}
	}

	m.mu.Lock()
	m.ackAt = time.Now()
	m.lastProgress = m.ackAt
	m.mu.Unlock()

	stop, err := m.watcher.Watch(ctx, m.sessID, m.onWatcherObservation)
	if err != nil {
		m.finishTurn()

		return "", fmt.Errorf("turn: start watcher: %w", err)
	}

	defer stop()

	defer m.startStallWatchdog(ctx)()

	reason, waitErr := m.Wait(ctx)
	m.finishTurn()

	if waitErr != nil {
		return "", waitErr
	}

	return reason, nil
}

// Wait returns the current turn's StopReason, blocking until the turn
// resolves; a second wait returns the SAME StopReason without re-resolving.
// Cancellation returns an error — never a synthesized StopReason.
func (m *Machine) Wait(ctx context.Context) (StopReason, error) {
	m.mu.Lock()
	resolvedCh := m.resolvedCh
	m.mu.Unlock()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("turn: wait: %w", ctx.Err())
	case <-resolvedCh:
		m.mu.Lock()
		defer m.mu.Unlock()

		return m.resolution.Reason, nil
	}
}

// Cancel requests cancellation of the current turn (D-02 lane 3, PITFALL 4):
// the first call on a running turn transitions it to cancelling and sends
// session/stop for the session through the conn's bypass lane — immediately,
// never queued behind ordinary traffic (dialect §1: the bypass set is
// exactly session/stop and friends). Further calls are no-ops: the stop is
// sent exactly once per turn, a cancel after the turn resolved writes
// nothing, and a cancel with no turn running is a no-op. The resolution
// itself is signal-driven through the one observe path — the stop outcome
// (the cancelled state patch, or the quiet settle while cancelling) resolves
// StopCancelled exactly once via the existing sync.Once completion path; a
// cancel racing a terminal event resolves whichever signal lands first,
// still exactly once.
func (m *Machine) Cancel() {
	m.mu.Lock()

	if m.resolved || m.state != stateRunning {
		m.mu.Unlock()

		return
	}

	m.state = stateCancelling

	// The ACP cancel contract (quick task 261125-s7k): the active
	// session/prompt MUST answer the client's cancel. The backend's own
	// stop outcome stays the primary signal (terminal, cancelled patch, or
	// quiet settle); the grace is the bound that keeps a silent or
	// chattering backend from holding the client's prompt — and, for Orca,
	// its steer queue, which waits for exactly this answer — open. Armed
	// under the same lock section as the state transition, so no resolution
	// can slip between them.
	grace := m.cancelGrace
	if grace <= 0 {
		grace = DefaultCancelGrace
	}

	if m.cancelTimer == nil {
		m.cancelTimer = time.AfterFunc(grace, m.resolveCancelGrace)
	}

	m.mu.Unlock()

	if err := m.conn.Stop(m.sessID); err != nil {
		// The bypass write failed (a dead stream): record it structurally —
		// the watcher's quiet settle while cancelling still ends the turn
		// honestly cancelled.
		slog.Warn("turn: bypass stop failed", "session", m.sessID, "err", err)
	}
}

// AckAt returns when the current turn's session/send ack arrived (zero
// before the ack) — the floor clock's origin for the quiet-window gates.
func (m *Machine) AckAt() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ackAt
}

// Resolution returns the current turn's exactly-once completion record (the
// zero Resolution before the first turn resolves). The front reads it to
// surface a recorded stall cause as the coded client error.
func (m *Machine) Resolution() Resolution {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.resolution
}

// ObserveEnvelope is the ONE classification path for every inbound envelope
// (the T-3-06 mitigation): the ordered D-02 signal union — terminal kinds
// with turnId match across the known carriers first, then the idle patch
// (state.updated leaving running). It is registered on the conn's
// notification observer list — delivery runs on the read goroutine and must
// not block (classification is bounded: small params only), and it is safe
// to call before Run (pre-turn envelopes are observation only).
func (m *Machine) ObserveEnvelope(env zcode.Envelope) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != stateRunning && m.state != stateCancelling {
		return
	}

	// While the orphan-recovery wait is active, the envelope is inspected
	// ONLY for explicitly session-scoped clear evidence; nothing classifies
	// or resolves until the single retry starts a fresh turn.
	if m.orphanWaiting {
		m.observeOrphanEvidenceLocked(env)

		return
	}

	// Activity feed (QA-03): a watcher whose settle gate consumes live wire
	// activity hears every mid-turn envelope through the machine's one
	// observe funnel — the poll watcher resets its gate on session-scoped
	// notifications, so a streaming turn never settles mid-generation.
	if af, ok := m.watcher.(ActivityFeed); ok {
		af.ObserveActivity(env)
	}

	// Any REAL notification observed mid-turn is evidence of life on the
	// turn's wire: the veto budget (quietVetoes) counts only TOTAL silence,
	// so a streaming turn never exhausts it. The watcher's poll echo (the
	// synthetic session/messages envelope each poll delivers) is presence
	// evidence, not wire life — it must not refill the budget, or a silent
	// think polling a byte-stable surface would veto forever.
	if env.Method != zcode.MethodSessionMessages {
		m.quietVetoes = 0
	}

	// The no-progress watchdog's clock: real model/tool progress resets it,
	// telemetry and stable poll echoes do not (the narrow scope list).
	m.noteProgressLocked(env)

	reason, terminal := m.classifyTerminalLocked(env)
	if terminal {
		// The stop's observed outcome (quick task 261125-s7k): once the
		// client asked for cancellation, a FAILED terminal — turn.failed, or
		// turn.terminal with a failed status, which is exactly what the live
		// desktop emits after session/stop — resolves the turn CANCELLED,
		// not failed. Answering failed would surface -32603 to a client that
		// expects the ACP cancelled stopReason. A genuinely COMPLETED
		// terminal keeps its completed outcome: the turn really did finish,
		// and the machine must not claim a cancellation that never
		// happened (PITFALL 4 holds in both directions).
		if m.state == stateCancelling && reason == StopFailed {
			m.resolveLocked(Resolution{Reason: StopCancelled, Signal: SignalCancel, At: time.Now()})

			return
		}

		m.resolveLocked(Resolution{Reason: reason, Signal: SignalTerminalEvent, At: time.Now()})

		return
	}

	if m.classifyIdlePatchLocked(env) {
		// A patch leaving running while cancelling is the stop's observed
		// outcome (the corpus cancel row's cancelled patch) — the honest
		// cancelled resolution, never a completed one (PITFALL 4: the
		// machine must not lie about how the turn ended).
		if m.state == stateCancelling {
			m.resolveLocked(Resolution{Reason: StopCancelled, Signal: SignalCancel, At: time.Now()})

			return
		}

		m.resolveLocked(Resolution{Reason: StopCompleted, Signal: SignalIdlePatch, At: time.Now()})
	}
}

// startStallWatchdog starts the no-progress watchdog goroutine on a derived
// context and returns the join function: cancelling the derived context
// unblocks the loop and the join waits until it exits, so Run never returns
// with the goroutine still running (the single-stop-after-resolution
// guarantee).
func (m *Machine) startStallWatchdog(ctx context.Context) func() {
	stallCtx, stallCancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		m.stallLoop(stallCtx)
	}()

	return func() {
		stallCancel()
		<-done
	}
}

// sendTurn issues one fire-and-forget session/send and returns the wire
// failure verbatim (the caller maps it).
func (m *Machine) sendTurn(ctx context.Context, content string) error {
	_, err := m.conn.Call(ctx, zcode.MethodSessionSend, map[string]any{
		"sessionId": m.sessID,
		"content":   content,
	})

	return err //nolint:wrapcheck // the caller maps the wire failure and wraps it in context
}

// recoverOrphanedSend handles a session/send rejection. The numeric -32010
// concurrency refusal is mapped to ErrTurnActive by default; when the
// machine's front-slot orphan recovery is enabled, it instead sends ONE
// bypass-lane stop, waits a bounded interval for explicitly session-scoped
// clear evidence, resets the per-turn signal state, and retries the send
// exactly once. A second -32010 returns ErrOrphanedBackend; any other retry
// failure returns the normal transport error. retried reports whether the
// recovery path owned the failure (so the caller maps accordingly).
func (m *Machine) recoverOrphanedSend(ctx context.Context, content string, sendErr error) (bool, error) {
	var callErr *zcode.CallError
	if !errors.As(sendErr, &callErr) || callErr.Err.Code != zcode.CodeConcurrency {
		return false, nil
	}

	if !m.OrphanRecovery {
		return false, nil
	}

	slog.Warn("turn: backend reports an orphaned active prompt; stopping and retrying once",
		"session", m.sessID)

	m.mu.Lock()
	m.startOrphanWaitLocked()
	clearCh := m.orphanClear
	window := m.orphanClearWindowLocked()
	m.mu.Unlock()

	// The stop goes through the bypass lane; a failure is recorded and the
	// bounded wait still runs (the retry is attempted either way).
	if err := m.conn.Stop(m.sessID); err != nil {
		slog.Warn("turn: orphan stop failed", "session", m.sessID, "err", err)
	}

	timer := time.NewTimer(window)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-clearCh:
	case <-timer.C:
	}

	m.mu.Lock()
	m.endOrphanWaitLocked()
	m.resetTurnSignalsLocked()
	m.mu.Unlock()

	retryErr := m.sendTurn(ctx, content)
	if retryErr == nil {
		return true, nil
	}

	var retryCallErr *zcode.CallError
	if errors.As(retryErr, &retryCallErr) && retryCallErr.Err.Code == zcode.CodeConcurrency {
		return true, fmt.Errorf("turn: session/send after orphan stop: %w: %w", retryErr, ErrOrphanedBackend)
	}

	return true, retryErr
}

// mapSendError maps a session/send failure onto the machine's public error:
// the orphan-recovery verdict passes through, the default numeric -32010
// mapping stays ErrTurnActive (matched by code, never message text), and
// everything else is the wrapped transport error.
func (m *Machine) mapSendError(err error) error {
	if errors.Is(err, ErrOrphanedBackend) {
		return err
	}

	var callErr *zcode.CallError
	if errors.As(err, &callErr) && callErr.Err.Code == zcode.CodeConcurrency {
		return fmt.Errorf("turn: session/send: %w: %w", callErr, ErrTurnActive)
	}

	return fmt.Errorf("turn: session/send: %w", err)
}

// startOrphanWaitLocked opens the orphan clear-evidence window with a fresh
// capacity-1 channel (evidence before the select is buffered, never lost).
func (m *Machine) startOrphanWaitLocked() {
	m.orphanWaiting = true
	m.orphanClear = make(chan struct{}, 1)
}

// endOrphanWaitLocked closes the orphan wait (idempotent).
func (m *Machine) endOrphanWaitLocked() {
	m.orphanWaiting = false
	m.orphanClear = nil
}

// signalOrphanClearLocked delivers clear evidence non-blockingly.
func (m *Machine) signalOrphanClearLocked() {
	if m.orphanClear == nil {
		return
	}

	select {
	case m.orphanClear <- struct{}{}:
	default:
	}
}

// orphanClearWindowLocked resolves the bounded clear wait: the injected
// override when positive, else the production default.
func (m *Machine) orphanClearWindowLocked() time.Duration {
	if m.OrphanClearTimeout > 0 {
		return m.OrphanClearTimeout
	}

	return DefaultOrphanClearTimeout
}

// observeOrphanEvidenceLocked inspects one envelope while the orphan wait is
// active. Clear evidence must be EXPLICITLY session-scoped: the machine's
// ObserveEnvelope is conn-wide, so a terminal or idle patch carrying a
// foreign OR ABSENT sessionId neither clears the wait nor resolves — the
// bound expires and the single retry still happens. Scoped evidence only
// signals the wait (nothing classifies or resolves): a terminal's non-empty
// turnId is retired so a late duplicate cannot bind the retried turn.
func (m *Machine) observeOrphanEvidenceLocked(env zcode.Envelope) {
	sid := paramsString(env.Params, "sessionId")
	if sid != m.sessID {
		slog.Debug("turn: orphan clear evidence ignored (not session-scoped)",
			"session", m.sessID, "evidence_session", sid)

		return
	}

	if kind, terminal := terminalSignalKind(env); terminal {
		if turnID := paramsString(env.Params, "turnId"); turnID != "" {
			m.retiredTurnID = turnID
		}

		m.signalOrphanClearLocked()

		slog.Debug("turn: orphan cleared by terminal evidence", "session", m.sessID, "kind", kind)

		return
	}

	if env.Method == zcode.MethodStateUpdated {
		if obs, ok := decodeStatePatch(env, m.sessID); ok && obs.status != "" &&
			obs.status != statusRunningPatch {
			m.signalOrphanClearLocked()

			slog.Debug("turn: orphan cleared by idle patch evidence",
				"session", m.sessID, "status", obs.status)
		}
	}
}

// terminalSignalKind reports the terminal kind an envelope carries on a
// known carrier (turn starts are not terminal evidence).
func terminalSignalKind(env zcode.Envelope) (string, bool) {
	field, ok := carrierKindField(env.Method)
	if !ok {
		return "", false
	}

	kind := paramsString(env.Params, field)

	switch kind {
	case KindTurnCompleted, KindTurnFailed, KindTurnTerminal,
		KindTurnCompletedCU, KindTurnFailedCU:
		return kind, true
	}

	return "", false
}

// guardTurnLocked rejects a concurrent second send (D-02: the -32010
// concurrency guard is enforced wrapper-side, never silently queued).
func (m *Machine) guardTurnLocked() error {
	switch m.state {
	case stateRunning, stateCancelling:
		return ErrTurnActive
	case stateIdle, stateDone:
		return nil
	}

	return nil
}

// beginTurnLocked starts a fresh turn: the per-turn signal state (the
// learned turnId, the running-patch latch and its revision, the
// start-announcement record, the veto budget, the resolution record)
// resets; the lifetime resolution counter does not — it is the exactly-once
// proof — and the retired turnId does not either (it is the memory that
// keeps a prior turn's late terminal from binding this one).
func (m *Machine) beginTurnLocked() {
	m.state = stateRunning
	m.resetTurnSignalsLocked()
	m.resolved = false
	m.resolution = Resolution{}
	m.ackAt = time.Time{}
	m.resolvedCh = make(chan struct{})
	m.stallWake = make(chan struct{}, 1)
}

// resetTurnSignalsLocked clears the per-turn signal and watchdog state — the
// state an orphan-recovery retry must start clean (running latch, bound
// turnId, start-proven, veto budget, progress clock, messages baseline)
// while the RETIRED turnId is deliberately preserved.
func (m *Machine) resetTurnSignalsLocked() {
	m.turnID = ""
	m.runningSeen = false
	m.runningRevision = 0
	m.runningRevisionSeen = false
	m.runningReason = ""
	m.startProven = false
	m.quietVetoes = 0
	m.lastProgress = time.Time{}
	m.toolActive = false
	m.messagesSeen = false
	m.lastMessagesRaw = nil
}

// finishTurn marks the turn lifecycle done — the machine's resting state
// between turns; the next Run starts a fresh turn (Phase 4 calls Run per
// session/prompt).
func (m *Machine) finishTurn() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Between turns there is nothing for the cancel grace to answer: a turn
	// that ended without resolving (a ctx-cancelled wait) must not have its
	// timer resolve a later turn on this machine.
	if m.cancelTimer != nil {
		m.cancelTimer.Stop()
		m.cancelTimer = nil
	}

	m.state = stateDone
}

// onWatcherObservation funnels watcher deliveries into the one observe
// path. The quiet-window settlement resolves completed — the settle gate
// (after the send-latency floor, both owned by the watcher) is the
// end-of-turn verdict of the fallback path (dialect §5 :141; the probe
// harness's quiet-window end mode) — EXCEPT for a start-announced turn:
// when the active turn's start was announced on a terminal-proven carrier
// (computer-use/operation-event or v4/telemetry/event, both live-proven on
// 3.18.2), a terminal signal is coming, and the settle may not end the turn
// while the veto budget holds (QA-03: the real desktop can think silently
// past the floor; ending the turn at the floor cut the reply off). Each
// vetoed settle spends one budget unit; the budget refills on ANY observed
// envelope (evidence of life). A cancelling turn never vetoes — the
// user-requested stop keeps its early cancelled resolution.
func (m *Machine) onWatcherObservation(obs Observation) {
	if obs.Quiet {
		m.mu.Lock()
		defer m.mu.Unlock()

		// The quiet settle while cancelling is the stop's observed outcome
		// when no patch follows the stop: the turn winds down cancelled,
		// never claiming a completed turn the stop interrupted.
		if m.state == stateCancelling {
			m.resolveLocked(Resolution{Reason: StopCancelled, Signal: SignalCancel, At: time.Now()})

			return
		}

		// The start-announced veto (QA-03): the promised terminal signal has
		// not landed yet — observation only, no resolution. The budget is
		// the honest bound: after maxQuietVetoes consecutive silent settles
		// the quiet fallback resolves after all (EXP-APIKEY-CYCLE-02: a
		// completed turn may present no terminal at all).
		if m.startProven && m.quietVetoes < m.maxQuietVetoes() {
			m.quietVetoes++

			return
		}

		m.resolveLocked(Resolution{Reason: StopCompleted, Signal: SignalQuietWindow, At: time.Now()})

		return
	}

	if obs.Env != nil {
		m.ObserveEnvelope(*obs.Env)
	}
}

// maxQuietVetoes resolves the veto budget: the injected override wins when
// positive, else the dialect-scaled default.
func (m *Machine) maxQuietVetoes() int {
	if m.MaxQuietVetoes > 0 {
		return m.MaxQuietVetoes
	}

	return DefaultMaxQuietVetoes
}

// noteProgressLocked feeds one observed envelope into the no-progress clock.
// The scope is deliberately narrow (the 2026-11-25 field incident: retry
// telemetry and stable poll echoes kept the quiet watcher alive forever):
//
//   - session/event part.delta with field text/reasoning;
//   - session/event model.streaming payload.kind text_delta/reasoning_delta;
//   - session/event tool.updated scheduled/started/progress (tool active)
//     and result/error (tool inactive);
//   - turn-start/terminal kinds on any carrier;
//   - a session/messages payload whose messages array is non-empty on first
//     sight or whose raw bytes changed since the previous poll.
//
// Everything else — model-request/retry telemetry, session.updated
// lifecycle payloads, byte-stable poll echoes — is a lifecycle fact, not
// model progress. Progress scopes to the machine's session when the
// envelope carries a sessionId.
func (m *Machine) noteProgressLocked(env zcode.Envelope) {
	if sid := paramsString(env.Params, "sessionId"); sid != "" && sid != m.sessID {
		return
	}

	progress := false

	switch env.Method {
	case zcode.MethodSessionEvent:
		progress = m.sessionEventProgressLocked(env)
	case MethodV4Telemetry, MethodComputerUseOperation:
		field, _ := carrierKindField(env.Method)
		progress = isTurnSignalKind(paramsString(env.Params, field))
	case zcode.MethodSessionMessages:
		progress = m.messagesProgressLocked(env)
	}

	if !progress {
		return
	}

	m.lastProgress = time.Now()
	m.pokeStallLocked()
}

// sessionEventProgressLocked classifies one session/event envelope against
// the progress scope list.
func (m *Machine) sessionEventProgressLocked(env zcode.Envelope) bool {
	kind := paramsString(env.Params, "type")

	if isTurnSignalKind(kind) {
		return true
	}

	switch kind {
	case "part.delta":
		field := paramsString(env.Params, "field")

		return field == "text" || field == "reasoning"
	case "model.streaming":
		payloadKind := nestedPayloadKind(env.Params)

		return payloadKind == "text_delta" || payloadKind == "reasoning_delta"
	case "tool.updated":
		switch nestedPayloadKind(env.Params) {
		case "scheduled", "started", "progress":
			m.toolActive = true

			return true
		case "result", "error":
			m.toolActive = false

			return true
		}
	}

	return false
}

// isTurnSignalKind reports whether a carrier kind is a turn start or
// terminal — both are turn lifecycle facts and count as progress.
func isTurnSignalKind(kind string) bool {
	switch kind {
	case KindTurnStarted, KindTurnCompleted, KindTurnFailed, KindTurnTerminal,
		KindTurnStartedCU, KindTurnCompletedCU, KindTurnFailedCU:
		return true
	}

	return false
}

// nestedPayloadKind digs payload.kind out of a session/event params map
// (model.streaming/tool.updated carry their discriminator there).
func nestedPayloadKind(raw json.RawMessage) string {
	var params struct {
		Payload struct {
			Kind string `json:"kind"`
		} `json:"payload"`
	}

	if err := json.Unmarshal(raw, &params); err != nil {
		return ""
	}

	return params.Payload.Kind
}

// messagesProgressLocked treats a session/messages content change as
// progress: a non-empty messages array on first sight, or raw bytes that
// differ from the previous poll. A byte-stable poll echo is presence
// evidence, not progress.
func (m *Machine) messagesProgressLocked(env zcode.Envelope) bool {
	var params struct {
		Messages json.RawMessage `json:"messages"`
	}

	if err := json.Unmarshal(env.Params, &params); err != nil || len(params.Messages) == 0 {
		return false
	}

	var messages []json.RawMessage
	if err := json.Unmarshal(params.Messages, &messages); err != nil {
		return false
	}

	nonEmpty := len(messages) > 0
	changed := m.lastMessagesRaw != nil && !bytes.Equal(m.lastMessagesRaw, params.Messages)
	m.lastMessagesRaw = bytes.Clone(params.Messages)

	if nonEmpty && !m.messagesSeen {
		m.messagesSeen = true

		return true
	}

	return nonEmpty && changed
}

// pokeStallLocked wakes the watchdog non-blockingly to recompute the
// remaining budget (progress moved the deadline).
func (m *Machine) pokeStallLocked() {
	if m.stallWake == nil {
		return
	}

	select {
	case m.stallWake <- struct{}{}:
	default:
	}
}

// stallLoop is the watchdog goroutine: it computes the remaining no-progress
// budget, wakes on progress or the deadline, and exits as soon as the turn
// is resolved or leaves the running state — a cancelling turn is outside the
// watchdog's scope, so the loop never busy-fires a non-positive remainder
// while the quiet/cancel path winds the turn down.
func (m *Machine) stallLoop(ctx context.Context) {
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {
		m.mu.Lock()
		if m.resolved || m.state != stateRunning {
			m.mu.Unlock()

			return
		}

		remaining := m.stallRemainingLocked(time.Now())
		wake := m.stallWake
		m.mu.Unlock()

		if remaining <= 0 {
			if m.fireStallIfExpired() {
				return
			}

			continue
		}

		timer.Reset(remaining)

		select {
		case <-ctx.Done():
			stallTimerStop(timer)

			return
		case <-wake:
			stallTimerStop(timer)
		case <-timer.C:
		}
	}
}

// stallTimerStop stops t and drains its channel so the next Reset starts
// from a clean timer.
func stallTimerStop(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

// stallRemainingLocked is the time left before the active budget expires
// (non-positive when the budget is already spent).
func (m *Machine) stallRemainingLocked(now time.Time) time.Duration {
	budget := m.stallBudgetLocked()

	if m.lastProgress.IsZero() {
		return budget
	}

	return m.lastProgress.Add(budget).Sub(now)
}

// stallBudgetLocked resolves the active budget: the tool allowance while a
// tool is active, the base allowance otherwise.
func (m *Machine) stallBudgetLocked() time.Duration {
	if m.toolActive {
		return m.stallToolBudgetLocked()
	}

	if m.StallTimeout > 0 {
		return m.StallTimeout
	}

	return DefaultStallTimeout
}

// stallToolBudgetLocked resolves the tool allowance: the explicit override,
// else 2x an explicit base (test ergonomics), else the production default.
func (m *Machine) stallToolBudgetLocked() time.Duration {
	if m.StallToolTimeout > 0 {
		return m.StallToolTimeout
	}

	if m.StallTimeout > 0 {
		return stallToolBudgetMultiple * m.StallTimeout
	}

	return DefaultStallToolTimeout
}

// fireStallIfExpired sends the bypass-lane stop BEFORE resolving the stall
// (so no successor prompt can race a late stop against its own send) and
// reports whether the watchdog loop is done. The state is re-checked under
// the mutex on both sides of the wire write: a cancel that landed in between
// owns the outcome, and progress that moved the deadline keeps the loop
// waiting.
func (m *Machine) fireStallIfExpired() bool {
	m.mu.Lock()
	if m.resolved || m.state != stateRunning {
		m.mu.Unlock()

		return true
	}

	now := time.Now()
	if remaining := m.stallRemainingLocked(now); remaining > 0 {
		m.mu.Unlock()

		return false
	}

	budget := m.stallBudgetLocked()
	toolActive := m.toolActive
	m.mu.Unlock()

	if err := m.conn.Stop(m.sessID); err != nil {
		slog.Warn("turn: stall stop failed", "session", m.sessID, "err", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.resolved || m.state != stateRunning {
		return true
	}

	if remaining := m.stallRemainingLocked(time.Now()); remaining > 0 {
		return false
	}

	err := stallCause(toolActive, budget)
	m.resolveLocked(Resolution{Reason: StopError, Signal: SignalStall, At: time.Now(), Err: err})

	slog.Warn("turn: stalled turn stopped", "session", m.sessID,
		"tool_active", toolActive, "budget", budget.String(), "err", err)

	return true
}

// errStallNoModelProgress / errStallNoToolProgress are the two stable stall
// causes recorded on Resolution.Err and surfaced by the front as the coded
// client error (the budget rides as the wrapper's detail, never the text).
var (
	errStallNoModelProgress = errors.New("turn stalled: no model progress (provider may be rate-limiting)")
	errStallNoToolProgress  = errors.New(
		"turn stalled: no tool progress (provider may be rate-limiting; a tool was active)")
)

// stallCause renders the recorded stall cause (the short stable wording the
// front surfaces as a coded -32603).
func stallCause(toolActive bool, budget time.Duration) error {
	if toolActive {
		return fmt.Errorf("%w: budget %s", errStallNoToolProgress, budget)
	}

	return fmt.Errorf("%w: budget %s", errStallNoModelProgress, budget)
}

// observeRouterEvent is the machine's face on the conn's reverse-request
// router policy events (D-05 record-honesty): a tracked-no-answer deadline
// expiring mid-turn — on the real path the scripted credential-cipher
// obligation the wrapper must NOT answer in Phase 3 (T-3-12) — is surfaced,
// never swallowed. The turn resolves the honest degraded state (StopError /
// SignalDegraded) exactly once, with the expiry recorded on Resolution.Err,
// and NEVER completed/failed from it: an obligation expiry is evidence the
// turn cannot complete, not a completion signal (the BRIDGE-04 prohibition).
// It is conn.OnRouterEvent's receiver — runs on the request/tracker
// goroutine and must not block.
func (m *Machine) observeRouterEvent(ev zcode.RouterEvent) {
	if ev.Kind != zcode.RouterDeadlineExpired {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != stateRunning && m.state != stateCancelling {
		return
	}

	m.resolveLocked(Resolution{
		Reason: StopError,
		Signal: SignalDegraded,
		At:     time.Now(),
		Err:    fmt.Errorf("turn: %s: %w after %s", ev.Method, errObligationExpired, ev.Deadline),
	})
}

// classifyIdlePatchLocked reports whether env is the idle patch: a
// state.updated whose patch.status leaves "running" for this session — the
// D-02 secondary signal (dialect §5 reliability column: "do not rely on it
// alone"). The running patch itself never resolves: the latch requires a
// running patch to have been observed first, and the leaving patch must be
// provably NEWER than the running patch that armed the turn.
//
// The idle patch is not reliably turn-scoped (dialect §5), so the binding is
// the session-scoped revision: the running patch records the MAX revision
// (and its reason) it was armed with, and a leaving patch completes the
// turn only when both revisions are present and the idle revision is
// strictly greater. A missing, equal, or lower revision is observation
// only — the turn resolves through its terminal, quiet window, or watchdog.
// While running, a cancelled status or a session_stopped reason is another
// turn's stop outcome (the stop was explicit) and never completes this one;
// while cancelling, any leaving-running patch is the stop's observed
// outcome.
func (m *Machine) classifyIdlePatchLocked(env zcode.Envelope) bool {
	if env.Method != zcode.MethodStateUpdated {
		return false
	}

	obs, ok := decodeStatePatch(env, m.sessID)
	if !ok {
		return false
	}

	slog.Debug("turn: state.updated patch", "session", m.sessID, "status", obs.status,
		"reason", obs.reason, "revision", obs.revision, "armed_revision", m.runningRevision,
		"state", m.state)

	if obs.status == statusRunningPatch {
		m.armRunningPatchLocked(obs)

		return false
	}

	if !m.runningSeen || obs.status == "" {
		return false
	}

	return m.leavingRunningCompletesLocked(obs)
}

// statePatchObservation is one decoded state.updated patch: the nested
// status plus the top-level reason/revision pair the ordering gate reads.
type statePatchObservation struct {
	status      string
	reason      string
	revision    int64
	hasRevision bool
}

// decodeStatePatch decodes one state.updated params payload scoped to
// sessID. Wire payloads keep the dialect's camelCase keys verbatim (the
// tagliatelle contract), so the session id digs out of the params map and
// only the snake-safe inner patch shape is struct-decoded.
func decodeStatePatch(env zcode.Envelope, sessID string) (statePatchObservation, bool) {
	var obs statePatchObservation

	sid := paramsString(env.Params, "sessionId")
	if sid != "" && sid != sessID {
		return obs, false
	}

	var params struct {
		Patch json.RawMessage `json:"patch"`
	}

	if err := json.Unmarshal(env.Params, &params); err != nil || len(params.Patch) == 0 {
		return obs, false
	}

	var patch struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(params.Patch, &patch); err != nil {
		return obs, false
	}

	obs.status = patch.Status
	obs.reason = paramsString(env.Params, "reason")
	obs.revision, obs.hasRevision = paramsInt64(env.Params, "revision")

	return obs, true
}

// armRunningPatchLocked records the running patch that armed the turn,
// keeping the MAX revision bound: a late lower-revision running echo must
// not lower the bar the idle patch has to clear.
func (m *Machine) armRunningPatchLocked(obs statePatchObservation) {
	m.runningSeen = true

	if obs.hasRevision && (!m.runningRevisionSeen || obs.revision > m.runningRevision) {
		m.runningRevision = obs.revision
		m.runningRevisionSeen = true
		m.runningReason = obs.reason
	}
}

// leavingRunningCompletesLocked applies the leaving-running gate: while
// cancelling, the patch is the stop's explicit outcome; while running, a
// cancelled status or session_stopped reason belongs to another turn's stop
// and the revision must be present on both sides and strictly newer.
func (m *Machine) leavingRunningCompletesLocked(obs statePatchObservation) bool {
	if m.state == stateCancelling {
		return true
	}

	if obs.status == statusCancelledPatch || obs.reason == reasonSessionStoppedPatch {
		return false
	}

	return obs.hasRevision && m.runningRevisionSeen && obs.revision > m.runningRevision
}

// classifyTerminalLocked reports the terminal StopReason when env is a
// turnId-matched terminal signal across the known carriers (RESEARCH Open
// Question 1 resolution — carrier-tolerant by design; the real 3.18.2
// surface is now live-observed: computer-use "turn-completed" and v4
// telemetry "turn.terminal", EXP-VM-QA02-family dumps 2026-11-23). Turn
// -start kinds record the turn's id; a start on a TERMINAL-PROVEN carrier
// additionally arms the quiet-settle veto (QA-03: the family promised a
// terminal signal). A terminal or turn-start with an EMPTY turnId never
// binds (all shipped carriers are turnId-scoped); a signal carrying the
// RETIRED turnId never binds a successor; a signal for a foreign bound turn
// is observation only (IN-05: a foreign turn's envelope must never resolve
// this turn, and a foreign start must not arm its veto).
func (m *Machine) classifyTerminalLocked(env zcode.Envelope) (StopReason, bool) {
	field, ok := carrierKindField(env.Method)
	if !ok {
		return "", false
	}

	kind := paramsString(env.Params, field)
	turnID := paramsString(env.Params, "turnId")

	switch kind {
	case KindTurnStarted, KindTurnStartedCU:
		if !m.bindTurnStartLocked(turnID) {
			return "", false
		}

		if m.startProvenCarrier(env.Method) {
			m.startProven = true
		}

		return "", false
	case KindTurnCompleted, KindTurnCompletedCU:
		if !m.bindTerminalTurnLocked(turnID) {
			return "", false
		}

		return StopCompleted, true
	case KindTurnFailed, KindTurnFailedCU:
		if !m.bindTerminalTurnLocked(turnID) {
			return "", false
		}

		return StopFailed, true
	case KindTurnTerminal:
		if !m.bindTerminalTurnLocked(turnID) {
			return "", false
		}

		return terminalStatusReason(paramsString(env.Params, "status")), true
	}

	return "", false
}

// startProvenCarrier reports whether a start signal on this carrier proves
// a terminal signal will follow: the two carriers whose terminal envelopes
// are live-observed on the real 3.18.2 desktop (computer-use/operation-
// event turn-completed AND v4/telemetry/event turn.terminal, same turnId —
// EXP-VM-QA02-family dumps). The session/event carrier's start does NOT
// arm the veto: its turn-started observation predates any observed
// terminal on that carrier, and the capture-family turns it characterizes
// are exactly the ones the quiet fallback is proven to end (dialect §5).
func (m *Machine) startProvenCarrier(method string) bool {
	return method == MethodComputerUseOperation || method == MethodV4Telemetry
}

// terminalStatusReason maps a v4 turn.terminal params status to the
// StopReason (raw3.out:84 verbatim: status/resultType "success" — the only
// value observed live). "success" (or an absent status — the terminal kind
// itself is the completion announcement) completes the turn; the known
// failure vocabulary resolves it failed; any other unobserved value
// completes — the terminal fired, and only the recorded failure literals
// may claim otherwise.
func terminalStatusReason(status string) StopReason {
	switch status {
	case "failed", "error":
		return StopFailed
	}

	return StopCompleted
}

// carrierKindField maps a carrier method to the params field that carries
// its kind literal (dialect §5 signal tables): session/event uses "type",
// the v4 telemetry and computer-use operation families use "kind" (live
// truth: the v4 family's terminal literal is "turn.terminal", the
// computer-use family's "turn-completed" — both turnId-scoped).
func carrierKindField(method string) (string, bool) {
	switch method {
	case zcode.MethodSessionEvent:
		return "type", true
	case MethodV4Telemetry, MethodComputerUseOperation:
		return "kind", true
	}

	return "", false
}

// bindTerminalTurnLocked decides whether a TERMINAL signal's turnId belongs
// to the live turn, and records it on first sight. The binder is
// conservative by construction (the 2026-11-25 false-completion race):
//
//   - an EMPTY turnId never binds (all shipped carriers are turnId-scoped;
//     the quiet window remains the honest fallback);
//   - a turnId equal to the RETIRED id never binds a successor — the prior
//     turn's late terminal is ignored while this turn is unbound;
//   - when unbound, the first other non-empty turnId is the live turn
//     (record-on-first-sight survives for the legitimate first signal);
//   - when bound, only the exact bound turnId matches;
//   - after resolution, a terminal that belongs to the resolved turn (or
//     arrives while nothing is bound) records its id as retired and never
//     resolves, so a late duplicate can never falsely complete a successor.
func (m *Machine) bindTerminalTurnLocked(turnID string) bool {
	if turnID == "" {
		return false
	}

	if m.resolved {
		if m.turnID == "" || turnID == m.turnID {
			m.retiredTurnID = turnID
		}

		return false
	}

	if m.turnID == "" {
		if turnID == m.retiredTurnID {
			return false
		}

		m.turnID = turnID

		return true
	}

	return turnID == m.turnID
}

// bindTurnStartLocked binds a turn-start signal. A start carrying the
// retired turnId is ignored while the successor is unbound AND no running
// patch has armed this turn yet: that is the stale-prior-turn shape (the
// prior turn's start reached the wire before the new send, so a late
// start+terminal pair cannot claim the successor). A retired-id start
// observed AFTER this turn's own running patch is the successor's own start
// under a backend that recycles turn ids (the legacy corpus fixtures and
// any runtime that does not mint per-turn ids): binding it is safe because
// the running patch proves this turn's send was already accepted.
func (m *Machine) bindTurnStartLocked(turnID string) bool {
	if turnID == "" {
		return false
	}

	if m.resolved {
		if m.turnID == "" || turnID == m.turnID {
			m.retiredTurnID = turnID
		}

		return false
	}

	if m.turnID == "" {
		if turnID == m.retiredTurnID && !m.runningSeen {
			return false
		}

		m.turnID = turnID

		return true
	}

	return turnID == m.turnID
}

// resolveCancelGrace is the cancel grace's callback (quick task 261125-s7k):
// when the client's cancel is still unanswered past the configured bound, the
// turn resolves with the client-requested outcome. This is not a synthesized
// completion — the client demanded the cancellation and the stop was already
// written; the grace only guarantees the active session/prompt ANSWERS. Every
// backend signal that lands first still wins (exactly-once, resolveLocked).
// The state guard keeps a timer that outlived its turn (a resolved turn or a
// fresh turn on the same machine) from ever resolving anything.
func (m *Machine) resolveCancelGrace() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.resolved || m.state != stateCancelling {
		return
	}

	m.resolveLocked(Resolution{Reason: StopCancelled, Signal: SignalCancel, At: time.Now()})
}

// resolveLocked resolves the turn exactly once (sync.Once semantics: the
// per-turn resolved flag under the machine mutex), closes the wait channel,
// and fires the OnResolve hook — the observable face of exactly-once. The
// first supporting signal wins; later signals are observations only. A
// resolved turn's bound turnId is retired for the successor's binder.
func (m *Machine) resolveLocked(res Resolution) {
	if m.resolved {
		return
	}

	// The turn is answered: the cancel grace must never fire for it.
	if m.cancelTimer != nil {
		m.cancelTimer.Stop()
	}

	m.resolved = true
	m.resolution = res

	if m.turnID != "" {
		m.retiredTurnID = m.turnID
	}

	slog.Debug("turn: resolved", "session", m.sessID, "reason", res.Reason,
		"signal", res.Signal, "turnId", m.turnID, "err", res.Err)

	close(m.resolvedCh)

	if m.OnResolve != nil {
		m.OnResolve(res)
	}
}

// paramsInt64 digs one top-level integer value out of an envelope params
// payload; undecodable params or an absent key classify as absent (the
// idle-binding conservative fallback).
func paramsInt64(raw json.RawMessage, key string) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return 0, false
	}

	switch v := obj[key].(type) {
	case float64:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}

		return n, true
	}

	return 0, false
}

// paramsString digs one top-level string value out of an envelope params
// payload; undecodable params classify as absent.
func paramsString(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}

	s, _ := obj[key].(string)

	return s
}
