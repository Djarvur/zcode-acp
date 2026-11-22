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

// stateRunningPatch is the state.updated patch.status value the idle-patch
// latch consumes (dialect §5: only "running" was ever observed live — the
// recorded A4 assumption; "leaving running" implies running was seen).
const statusRunningPatch = "running"

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

	mu          sync.Mutex
	state       machineState
	turnID      string
	runningSeen bool
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

// NewMachine returns a machine bound to sessionID on conn with the given
// watcher, and wires the machine's one observation surface: every inbound
// notification envelope funnels into it via conn.AddObserver and
// reverse-request router policy events via conn.AddRouterObserver — both
// ordered fan-outs (WR-01: a second machine no longer steals the first's
// router hook). NewMachine keeps BOTH remove funcs on the Machine; plan
// 05-05's session/close teardown calls them (WR-07: teardown can finally
// release observers). Registration happens before any frame flows (the
// Conn hook contract).
func NewMachine(conn *zcode.Conn, sessionID string, watcher TurnWatcher) *Machine {
	m := &Machine{
		conn:       conn,
		sessID:     sessionID,
		watcher:    watcher,
		state:      stateIdle,
		resolvedCh: make(chan struct{}),
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

	if _, err := m.conn.Call(ctx, zcode.MethodSessionSend, map[string]any{
		"sessionId": m.sessID,
		"content":   content,
	}); err != nil {
		m.finishTurn()

		// The server-side -32010 rejection (PITFALL 5): when the send
		// reached the wire and the server refused it with dialect §6
		// CodeConcurrency, surface the same mapped clean error the
		// wrapper-side guard uses — matched by the numeric code, never the
		// message text.
		var callErr *zcode.CallError
		if errors.As(err, &callErr) && callErr.Err.Code == zcode.CodeConcurrency {
			return "", fmt.Errorf("turn: session/send: %w: %w", callErr, ErrTurnActive)
		}

		return "", fmt.Errorf("turn: session/send: %w", err)
	}

	m.mu.Lock()
	m.ackAt = time.Now()
	m.mu.Unlock()

	stop, err := m.watcher.Watch(ctx, m.sessID, m.onWatcherObservation)
	if err != nil {
		m.finishTurn()

		return "", fmt.Errorf("turn: start watcher: %w", err)
	}

	defer stop()

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

	reason, terminal := m.classifyTerminalLocked(env)
	if terminal {
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
// learned turnId, the running-patch latch, the start-announcement record,
// the veto budget, the resolution record) resets; the lifetime resolution
// counter does not — it is the exactly-once proof.
func (m *Machine) beginTurnLocked() {
	m.state = stateRunning
	m.turnID = ""
	m.runningSeen = false
	m.startProven = false
	m.quietVetoes = 0
	m.resolved = false
	m.resolution = Resolution{}
	m.ackAt = time.Time{}
	m.resolvedCh = make(chan struct{})
}

// finishTurn marks the turn lifecycle done — the machine's resting state
// between turns; the next Run starts a fresh turn (Phase 4 calls Run per
// session/prompt).
func (m *Machine) finishTurn() {
	m.mu.Lock()
	defer m.mu.Unlock()

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
// running patch to have been observed first, and any later non-running
// status counts as leaving running.
func (m *Machine) classifyIdlePatchLocked(env zcode.Envelope) bool {
	if env.Method != zcode.MethodStateUpdated {
		return false
	}

	// Wire payloads keep the dialect's camelCase keys verbatim (the
	// tagliatelle contract), so the session id digs out of the params map
	// and only the snake-safe inner patch shape is struct-decoded.
	sessID := paramsString(env.Params, "sessionId")
	if sessID != "" && sessID != m.sessID {
		return false
	}

	var params struct {
		Patch json.RawMessage `json:"patch"`
	}

	if err := json.Unmarshal(env.Params, &params); err != nil || len(params.Patch) == 0 {
		return false
	}

	var patch struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(params.Patch, &patch); err != nil {
		return false
	}

	if patch.Status == statusRunningPatch {
		m.runningSeen = true

		return false
	}

	return m.runningSeen && patch.Status != ""
}

// classifyTerminalLocked reports the terminal StopReason when env is a
// turnId-matched terminal signal across the known carriers (RESEARCH Open
// Question 1 resolution — carrier-tolerant by design; the real 3.18.2
// surface is now live-observed: computer-use "turn-completed" and v4
// telemetry "turn.terminal", EXP-VM-QA02-family dumps 2026-11-23). Turn
// -start kinds record the turn's id; a start on a TERMINAL-PROVEN carrier
// additionally arms the quiet-settle veto (QA-03: the family promised a
// terminal signal). A terminal whose turnId does not match the turn's id is
// NOT a completion signal (a foreign turn's envelope must never resolve
// this turn).
func (m *Machine) classifyTerminalLocked(env zcode.Envelope) (StopReason, bool) {
	field, ok := carrierKindField(env.Method)
	if !ok {
		return "", false
	}

	kind := paramsString(env.Params, field)
	turnID := paramsString(env.Params, "turnId")

	switch kind {
	case KindTurnStarted, KindTurnStartedCU:
		if m.turnID == "" {
			m.turnID = turnID
		}

		if m.startProvenCarrier(env.Method) {
			m.startProven = true
		}

		return "", false
	case KindTurnCompleted, KindTurnCompletedCU:
		if !m.turnIDMatchesLocked(turnID) {
			return "", false
		}

		return StopCompleted, true
	case KindTurnFailed, KindTurnFailedCU:
		if !m.turnIDMatchesLocked(turnID) {
			return "", false
		}

		return StopFailed, true
	case KindTurnTerminal:
		if !m.turnIDMatchesLocked(turnID) {
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

// turnIDMatchesLocked reports whether a turn-scoped signal belongs to the
// active turn. Record-on-first-sight: exactly one turn runs per session at
// a time (the -32010 guard), so the first turn-scoped signal defines the
// turn's id; later signals must match it.
func (m *Machine) turnIDMatchesLocked(turnID string) bool {
	if m.turnID == "" {
		m.turnID = turnID

		return true
	}

	return turnID == m.turnID
}

// resolveLocked resolves the turn exactly once (sync.Once semantics: the
// per-turn resolved flag under the machine mutex), closes the wait channel,
// and fires the OnResolve hook — the observable face of exactly-once. The
// first supporting signal wins; later signals are observations only.
func (m *Machine) resolveLocked(res Resolution) {
	if m.resolved {
		return
	}

	m.resolved = true
	m.resolution = res
	close(m.resolvedCh)

	if m.OnResolve != nil {
		m.OnResolve(res)
	}
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
