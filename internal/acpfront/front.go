package acpfront

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/core"
	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Front implements acpport.Handlers (RESEARCH Pattern 1, the D-01 seam):
// the agent policy layer — the honest D-02 initialize report, the D-03
// session/new contract (bare create + subscribe + setModel follow-up), the
// D-04 streamed prompt turn over the Phase-3 turn machine, and the D-05
// exactly-once cancel. It owns the back-leg conn and registers the delta
// tap (stream.go) BEFORE any machine is constructed — fan-out registration
// order is delivery order, so the tap classifies every envelope ahead of
// the machine and a terminal event can never resolve a turn ahead of
// earlier delta envelopes (RESEARCH Pattern 3, flush-before-stopReason).
//
// Locked contracts (04-CONTEXT D-02..D-05):
//   - D-02 honesty: initialize advertises exactly what exists; the
//     capability report is assembled by the adapter from the neutral
//     InitReport with the SDK zero value — nothing to suppress.
//   - D-03: session/new = bare create (workspaceKey = cwd, no model —
//     Pitfall 4) → subscribe {deliveryKind:desktop-continuous} BEFORE any
//     turn (subscribe-before-any-turn, A6 — event pushes are gated on the
//     deliveryKind record, Pitfall 1) → setModel follow-up iff configured
//     (Open Question 4; create drops options.reasoningLevel).
//   - D-04: every backend part.delta (field text) becomes exactly ONE
//     message chunk in arrival order — nothing synthesized (prohibition).
//   - D-05: cancelled is a SUCCESS stopReason resolved exactly once —
//     Prompt never returns ctx.Canceled (the SDK would map it to -32800,
//     Pitfall 2); ONE turn at a time across the whole front (the
//     app-server event stream routes by turnId only — dialect §5 — so
//     single-active-turn is the delta-routing precondition).
type Front struct {
	conn     *zcode.Conn
	registry *core.SessionRegistry
	model    *ModelSelection
	version  string

	// newTurnCtx derives a front-owned turn context from the front's base
	// context (closure-captured in New — the containedctx house rule:
	// contexts never live in struct fields, conn.go captures the same way).
	newTurnCtx func() (context.Context, context.CancelFunc)

	// perm is the TOOL-02 permission bridge (plan 05-02): the pending table
	// for in-flight client round-trips plus the connection-bound sender
	// bound at acpport construction (permission.go).
	perm *permissionBridge

	// turnMu guards the front-wide turn slot (the single-active-turn
	// invariant) and the active turn's tap-facing emitter — the closure
	// binding the turn's sink, session id, and context into one call the
	// stream.go tap makes per delta envelope. Nil when no turn is active.
	// The slot is held for the turn's whole lifetime: from Prompt entry to
	// its return. turnSession/turnDone carry the active turn's session id
	// and its lifecycle done channel (closed by release after the flush) —
	// the SESS-01 close teardown waits on them for its bounded
	// cancel-then-drain (lifecycle.go). The remaining slot fields are the
	// tap's turn-scoped classification state (plan 05-04): turnID is the
	// active turn's backend turnId (IN-05 — record-on-first-sight,
	// foreign-turnId deltas are skipped), turnCwd the session cwd the
	// diff/location paths resolve absolute against (Pitfall 5), turnDiffs
	// the scheduled diffs the result arm re-emits (clients render early;
	// update content merges per spec).
	turnMu      sync.Mutex
	turnActive  bool
	turnSession string
	turnDone    chan struct{}
	turnEmit    func(note acpport.SessionNote) error
	turnID      string
	turnCwd     string
	turnDiffs   map[string]acpport.ToolDiff
	turnPump    *turnPump

	// closingMu guards the closing set (SESS-01, lifecycle.go): the session
	// ids mid-teardown. The gate makes the close-vs-prompt race
	// deterministic (T-5-12): once close marks an id, a prompt for it can
	// never acquire the slot mid-teardown — it lands exactly one clean
	// coded rejection instead of racing the release steps.
	closingMu sync.Mutex
	closing   map[string]bool

	// cwdMu guards the session-cwd registry (session id → the validated
	// absolute cwd from session/new) — acquireTurn copies the active
	// session's cwd into the turn slot.
	cwdMu sync.Mutex
	cwds  map[string]string

	// streamCeiling is the configured per-turn stream queue byte ceiling
	// (WithStreamCeiling; zero → the documented default at pump build).
	streamCeiling int

	// closeWaitOverride overrides the close teardown's bounded active-turn
	// wait (lifecycle.go closeTurnWait) when set — the plan-05-10
	// deadline-arm test seam (WithCloseTurnWait); zero keeps the
	// production default.
	closeWaitOverride time.Duration

	// auth is the injected authenticate machinery (plan 05-07, WithAuth):
	// the configured key binding, the resolved install, and the read-back
	// want list. Nil until wired — authenticate answers the honest no-key
	// coded error.
	auth *AuthConfig
}

// ModelSelection is the configured backend model the session/new leg binds
// with the setModel follow-up (D-03). A nil selection skips the follow-up
// (Open Question 4 resolution: setModel iff configured); tests always
// configure one, matching the corpus-locked payload values.
type ModelSelection struct {
	ProviderID     string
	ModelID        string
	ReasoningLevel string
}

// Option configures the Front at construction, before any frame flows.
type Option func(*Front)

// WithVersion sets the agentInfo version (D-02 real agentInfo). The cmd
// binary's version var arrives in plan 04-05; unset stays "0.0.0-dev".
func WithVersion(version string) Option {
	return func(f *Front) { f.version = version }
}

// WithModel sets the model selection the session/new leg binds via the
// setModel follow-up (D-03, Open Question 4 resolution).
func WithModel(sel ModelSelection) Option {
	return func(f *Front) { f.model = &sel }
}

// WithStreamCeiling sets the per-turn stream queue's byte ceiling (the WR-03
// OQ3 disposition: a per-turn MiB-scale memory ceiling). Zero (unset) uses
// the documented default. The ceiling is enforced by the per-turn pump
// (turnctx.go): a breach fails the turn loud with a coded prompt error —
// never a hang, never a silent drop.
func WithStreamCeiling(bytes int) Option {
	return func(f *Front) { f.streamCeiling = bytes }
}

// WithCloseTurnWait overrides the close teardown's bounded active-turn wait
// (lifecycle.go closeTurnWait). Plan-05-10 test seam: the deadline arm of
// drainActiveTurn (a turn still active past the wait reports sawTurn and the
// permission fan-in fires best-effort) is exercised with a shrunk wait;
// production wiring never sets it.
func WithCloseTurnWait(d time.Duration) Option {
	return func(f *Front) { f.closeWaitOverride = d }
}

// New returns a Front owning conn and registers the delta tap on the
// conn's ordered observer fan-out BEFORE any machine exists — a machine
// constructed later (per NewSession) lands after the tap in the delivery
// order (RESEARCH Pattern 3). Production wiring wraps the result via
// acpport.NewAgent; the acpport connection binds the update sink that
// Prompt hands back into the front's turn slot.
func New(conn *zcode.Conn, opts ...Option) *Front {
	// The front's base context — the parent of every front-owned turn
	// context. Deliberately process-lifetime (IN-03 truth refresh): turn
	// contexts are cancelled by release (turnctx.go), and an SDK
	// disconnect reaches the turn through the cancel fan-in
	// → machine.Cancel — so there is no cancellable-base override seam
	// for the cmd wiring to derive.
	baseCtx := context.Background()

	f := &Front{
		conn:     conn,
		registry: core.NewSessionRegistry(),
		version:  "0.0.0-dev",
		perm:     newPermissionBridge(),
		cwds:     make(map[string]string),
		newTurnCtx: func() (context.Context, context.CancelFunc) {
			return context.WithCancel(baseCtx)
		},
	}

	for _, opt := range opts {
		opt(f)
	}

	if conn != nil {
		conn.AddObserver(f.observeEnvelope) // the tap — BEFORE any machine
	}

	return f
}

// Initialize returns the honest D-02 report: real agent identity and the
// two advertised auth methods (zai-api-key functional in Phase 5, the
// coding plan in Phase 6). Wire capabilities are assembled by the adapter
// as the SDK zero value — honesty by construction, nothing to suppress
// (RESEARCH Code Examples: executed-probe shape). The advertised list is
// the single source the authenticate validation reads (auth.go).
func (f *Front) Initialize(ctx context.Context) (acpport.InitReport, error) {
	_ = ctx

	return acpport.InitReport{
		Name:        "zcode-acp",
		Title:       "zcode-acp agent",
		Version:     f.version,
		AuthMethods: advertisedAuthMethods(),
	}, nil
}

// NewSession runs the D-03 contract on the back leg: bare session/create
// (workspaceKey = cwd; no model object — the create path serializes the
// selection to providerId/modelId and drops options.reasoningLevel,
// Pitfall 4), session/subscribe {deliveryKind:desktop-continuous} BEFORE
// any turn (A6 — session/event pushes are gated on the deliveryKind
// record, Pitfall 1), then the session/setModel follow-up ONLY when a
// model selection is configured (Open Question 4 resolution) with explicit
// persistAsWorkspaceLastUsed:false (Open Question 3 resolution). The
// session's turn machine registers in the core registry before the id
// returns. mcpServers are accepted and ignored (D-03).
func (f *Front) NewSession(ctx context.Context, params acpport.NewSessionParams) (string, error) {
	// Spec input validation (RESEARCH Security Domain V5, T-4-01): the ACP
	// session-setup contract makes cwd MUST-be-absolute — validate BEFORE
	// any backend call so a rejected session/new never touches the back-leg
	// wire (the fake transcript must show no session/create frame). The
	// empty-cwd case folds in: an empty string is not absolute either.
	if !filepath.IsAbs(params.Cwd) {
		return "", acpport.NewInvalidParams(
			"session/new: cwd must be absolute (spec session-setup MUST), got %q", params.Cwd)
	}

	res, err := f.conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{"workspacePath": params.Cwd, "workspaceKey": params.Cwd},
	})
	if err != nil {
		return "", acpport.NewInternal("session/create: %v", err)
	}

	sid, err := createSessionID(res, "session/create")
	if err != nil {
		return "", err
	}

	f.recordCwd(sid, params.Cwd)

	if _, err := f.conn.Call(ctx, zcode.MethodSessionSubscribe, map[string]any{
		wireKeySessionID: sid,
		"deliveryKind":   "desktop-continuous",
	}); err != nil {
		// WR-04 (plan 05-05): the create succeeded — tear the backend
		// session down best-effort instead of orphaning it (T-5-12).
		f.closeOrphan(ctx, sid, "subscribe", err)

		return "", acpport.NewInternal("session/subscribe: %v", err)
	}

	if f.model != nil {
		// IN-04 (plan 05-05): the options object rides ONLY when a level is
		// configured — an empty level emits the documented omit-shape (never
		// `"options":{"reasoningLevel":""}`), so the backend's own actionable
		// refusal surfaces for level-strict providers. The configured-with-
		// level preamble stays byte-locked by the corpus.
		model := map[string]any{
			"providerId": f.model.ProviderID,
			"modelId":    f.model.ModelID,
		}

		if f.model.ReasoningLevel != "" {
			model["options"] = map[string]any{"reasoningLevel": f.model.ReasoningLevel}
		}

		if _, err := f.conn.Call(ctx, zcode.MethodSessionSetModel, map[string]any{
			wireKeySessionID:             sid,
			"model":                      model,
			"persistAsWorkspaceLastUsed": false,
		}); err != nil {
			// WR-04: same orphan discipline for the setModel refusal —
			// every post-create failure closes the backend session.
			f.closeOrphan(ctx, sid, "setModel", err)

			return "", acpport.NewInternal("session/setModel: %v", err)
		}
	}

	// The machine registers on the conn's fan-out AFTER the tap (constructed
	// earlier, front.go New) — the flush-before-stopReason registration
	// order. Phase-3 watcher defaults; corpus-grade terminal/cancel signals
	// resolve turns well inside the quiet-window gates.
	machine := turn.NewMachine(f.conn, sid, turn.NewPollWatcher(f.conn, turn.PollConfig{}))
	f.registry.Register(core.Session{ID: sid, Machine: machine})

	slog.Info("acpfront: session opened", "session", sid)

	return sid, nil
}

// wireKeySessionID is the dialect's camelCase session-id params key (the
// capture-verbatim wire spelling — the corpus discipline's
// wireKeySessionID twin on the production side).
const wireKeySessionID = "sessionId"

// turnActiveMessage is the ONE stable rejection surface for concurrent
// prompts (D-05, CORE-05): whatever guard layer fires — the front-wide turn
// slot here, or the machine's ErrTurnActive below (whose backend-refusal leg
// is the machine's numeric -32010 mapping, message text never matched) — the
// client sees the same invalid-request class carrying this message. No
// internal locations or content ever ride the error (the machine-path rule).
// The text names the FRONT-WIDE scope accurately (WR-05, plan 05-05): the
// slot spans every session of this agent, so the message must not send a
// cross-session prompt hunting for a stuck turn in the wrong conversation.
const turnActiveMessage = "another prompt is already running (this agent runs one turn at a time)"

// Prompt runs one prompt turn (D-04/D-05): the front-wide turn slot is
// acquired for the turn's whole lifetime (the single-active-turn
// invariant — the delta-routing precondition, see the type doc), the turn
// context is front-owned (NOT the SDK request ctx — turnctx.go), and the
// machine's outcome maps onto the neutral stop reasons. Cancelled resolves
// as a SUCCESS stopReason exactly once; the concurrency guard fires as ONE
// invalid-request rejection (both the slot and the machine's ErrTurnActive);
// other backend failures surface as coded -32603 internal errors (A5).
func (f *Front) Prompt(
	sdkCtx context.Context, params acpport.PromptParams, sink acpport.UpdateSink,
) (acpport.StopReason, error) {
	st, err := f.registry.Get(params.SessionID)
	if err != nil {
		return "", acpport.NewInvalidRequest("session/prompt: %v: %s", err, params.SessionID)
	}

	// SESS-01 gate (lifecycle.go): a session mid-teardown takes prompts as
	// the clean ErrNoSession-class rejection — never a mid-close slot
	// acquisition racing the release steps (T-5-12).
	if f.isClosing(params.SessionID) {
		return "", acpport.NewInvalidRequest("session/prompt: %s: the session is closing or closed", params.SessionID)
	}

	turnCtx, release, err := f.acquireTurn(params.SessionID, sink)
	if err != nil {
		return "", err // the pre-validated closing rejection (acquireTurn gate)
	}

	if release == nil {
		return "", acpport.NewInvalidRequest(turnActiveMessage)
	}

	defer release()

	// WR-04 (plan 05-10): the fan-in carries the turn's identity — the pump
	// pointer acquireTurn published under the slot lock. Captured right
	// after acquisition: the slot is ours until the deferred release runs,
	// so this is THIS turn's pump by construction.
	pump := f.turnPumpFor()

	go f.watchCancelFanIn(sdkCtx, turnCtx, pump, st, params.SessionID)

	//nolint:contextcheck // the turn ctx is front-owned BY CONTRACT (Pattern 5,
	// turnctx.go): deriving it from sdkCtx would surface the SDK's -32800
	// error instead of the cancelled success stopReason.
	reason, runErr := st.Machine.Run(turnCtx, params.Text)

	// WR-03 fail-loud gate (plan 05-04): a stream-ceiling breach fails the
	// turn REGARDLESS of how the machine resolved — the streamed output
	// dropped at the ceiling, so a success stopReason would be dishonest.
	// The breach error is the coded prompt error naming the cause.
	if breach := f.turnBreach(); breach != nil {
		return "", breach
	}

	if runErr != nil {
		// The concurrency guard (CORE-05): the machine's ErrTurnActive — its
		// backend-refusal leg being the numeric -32010 mapping — is the same
		// rejection the front-wide slot answers, so both surfaces share the
		// one invalid-request constructor and message (never the wrapped
		// transport noise, which carries the server's wire error verbatim).
		if errors.Is(runErr, turn.ErrTurnActive) {
			return "", acpport.NewInvalidRequest(turnActiveMessage)
		}

		return "", acpport.NewInternal("session/prompt: %v", runErr)
	}

	switch reason {
	case turn.StopCompleted:
		return acpport.StopEndTurn, nil
	case turn.StopCancelled:
		return acpport.StopCancelled, nil
	case turn.StopFailed, turn.StopError:
		// A5: ACP v1 has no failure stopReason — a backend failure surfaces
		// as the coded internal error, never a synthetic result.
		return "", acpport.NewInternal("session/prompt: turn ended: %s", reason)
	default:
		return "", acpport.NewInternal("session/prompt: turn ended: %s", reason)
	}
}

// Cancel cancels the session's active turn exactly once (D-05): the
// machine's bypass-lane Cancel is idempotent — repeats and
// cancel-after-complete are no-ops (Phase 3 proven). The prompt call
// resolves stopReason cancelled as a SUCCESS response; this handler never
// touches the turn context — cancelling it would surface the SDK's -32800
// error instead of the success stopReason (Pitfall 2, turnctx.go).
//
// TOOL-02 cancel fan-in (plan 05-02): the in-flight permission bridges are
// cancelled with the turn (→ -32021 replies to the backend's pending asks)
// — a cancelled turn leaves no pending bridge silently dangling while the
// server re-announces forever (T-5-05, Pitfall 2). WR-04/CR-01 alignment
// (plan 05-10): the fan-in fires only while a turn of THIS session holds
// the front-wide slot — the same semantics drainActiveTurn's sawTurn
// reports on the close path; a cancel for a slot-free session fans
// nothing in. WR-07 (plan 05-15): the fan-in is SESSION-SCOPED and atomic
// with the observation — the slotHeld check and the pendings take run
// under ONE turnMu section, so a descheduled handler can never sweep a
// later turn's (or another session's) fresh ask; the cancels fire outside
// any lock, after machine.Cancel.
func (f *Front) Cancel(ctx context.Context, sessionID string) error {
	_ = ctx

	st, err := f.registry.Get(sessionID)
	if err != nil {
		return acpport.NewInvalidRequest("session/cancel: %v: %s", err, sessionID)
	}

	// Observe the slot BEFORE the cancel (WR-04/CR-01, plan 05-10): release
	// nils the fields the moment the turn resolves, so a post-cancel
	// observation would miss a turn the cancel itself resolved. The take
	// rides the SAME section (WR-07 — check-and-take is atomic; lock
	// ordering one-way turnMu → bridge.mu, T-5-38).
	f.turnMu.Lock()
	slotHeld := f.turnActive && f.turnSession == sessionID
	take := []context.CancelFunc(nil)
	if slotHeld {
		take = f.perm.takePendingCancels(sessionID)
	}
	f.turnMu.Unlock()

	st.Machine.Cancel()

	if slotHeld {
		fireTakenPendings(take)
	}

	return nil
}

// watchCancelFanIn is the prompt's cancel fan-in goroutine body (D-05):
// either cancel source triggers exactly one machine.Cancel. The contexts
// ride as arguments, never struct fields (containedctx); the SDK ctx is
// only selected on for its Done arm — deriving anything from it would
// surface the SDK's -32800 error instead of the cancelled success
// (Pattern 5, turnctx.go).
//
// The identity re-check under the slot lock (plan 05-10 WR-04): a
// descheduled arm that runs past the turn boundary (release already nilled
// the slot, a later turn owns it) must never cancel the later turn as
// cancelled — skip silently.
//
// WR-06 (plan 05-15): when the turn IS ours, its pending permission
// bridges fan in WITH it (the 05-02 prohibition — a cancelled turn leaves
// every pending bridge answered -32021, never silently dangling while the
// server re-announces). The take rides the SAME turnMu section as the
// identity re-check, so check-and-take is atomic — no deschedule window
// can sweep a later turn's fresh ask. Lock ordering is one-way
// turnMu → bridge.mu (T-5-38); the cancels fire outside any lock, after
// machine.Cancel (idempotent, Phase 3: repeats no-op).
func (f *Front) watchCancelFanIn(
	sdkCtx context.Context,
	turnCtx context.Context,
	pump *turnPump,
	st core.Session,
	sessionID string,
) {
	select {
	case <-sdkCtx.Done():
		f.turnMu.Lock()
		mine := f.turnPump == pump
		take := []context.CancelFunc(nil)
		if mine {
			take = f.perm.takePendingCancels(sessionID)
		}
		f.turnMu.Unlock()

		if mine {
			st.Machine.Cancel() // idempotent (Phase 3: repeats no-op)
			fireTakenPendings(take)
		}
	case <-turnCtx.Done():
	}
}

// turnPumpFor returns the active turn's pump pointer under the slot lock —
// the per-turn identity the cancel fan-in re-checks (plan 05-10 WR-04).
// Prompt captures it right after acquisition; only Prompt's own deferred
// release can nil the slot before the turn's end, so the capture is this
// turn's pump by construction.
func (f *Front) turnPumpFor() *turnPump {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	return f.turnPump
}

// recordCwd stores one session's validated absolute cwd (the T-4-01 guard
// upstream guarantees absoluteness) for the tap's diff/location resolution.
func (f *Front) recordCwd(sessionID, cwd string) {
	f.cwdMu.Lock()
	defer f.cwdMu.Unlock()

	f.cwds[sessionID] = cwd
}

// dropCwd removes one session's cwd entry (SESS-01 teardown — the map
// stays bounded with the registry, the WR-07 discipline).
func (f *Front) dropCwd(sessionID string) {
	f.cwdMu.Lock()
	defer f.cwdMu.Unlock()

	delete(f.cwds, sessionID)
}

// isClosing reports whether the session is mid-teardown (the SESS-01
// close-vs-prompt race gate, lifecycle.go).
func (f *Front) isClosing(sessionID string) bool {
	f.closingMu.Lock()
	defer f.closingMu.Unlock()

	return f.closing[sessionID]
}

// closeOrphan is the WR-04 error-path close (plan 05-05): a session/new
// failure after session/create succeeded tears the just-created backend
// session down best-effort and logs structurally. It is the wire-level
// close — NOT Front.CloseSession: the orphan deliberately holds no
// registry entry (nothing to release; the machine was never constructed).
func (f *Front) closeOrphan(ctx context.Context, sessionID, stage string, cause error) {
	if _, err := f.conn.Call(ctx, zcode.MethodSessionClose, map[string]any{"sessionId": sessionID}); err != nil {
		slog.Warn("acpfront: orphan backend session close failed",
			"session", sessionID, "stage", stage, "err", err)

		return
	}

	slog.Warn("acpfront: closed the error-path backend session",
		"session", sessionID, "stage", stage, "cause", cause)
}

// turnScope copies the active turn's emitter out under the slot lock and
// applies the IN-05 turn-id guard (record-on-first-sight; a foreign-turnId
// envelope is not scoped). The emit call itself happens OUTSIDE the lock —
// the lock is never held across a wire write.
func (f *Front) turnScope(turnID string) (emit func(note acpport.SessionNote) error, scoped bool) {
	emit, _, scoped = f.turnScopeWithCwd(turnID)

	return emit, scoped
}

// turnScopeWithCwd is turnScope plus the session cwd copy the tool
// classification resolves diff/location paths against (Pitfall 5).
func (f *Front) turnScopeWithCwd(turnID string) (
	emit func(note acpport.SessionNote) error,
	cwd string,
	scoped bool,
) {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	if !f.turnActive || f.turnEmit == nil {
		return nil, "", false
	}

	if turnID != "" {
		switch f.turnID {
		case "":
			f.turnID = turnID // record-on-first-sight (machine.go discipline)
		case turnID:
		default:
			return nil, "", false // IN-05: a foreign turn's envelope never reaches the client
		}
	}

	return f.turnEmit, f.turnCwd, true
}

// recordTurnDiff stores one scheduled tool's diff recipe in the turn slot
// so the result arm re-emits it (TOOL-01: the diff rides BOTH the
// scheduled tool_call and the result update). A no-op outside an active
// turn.
func (f *Front) recordTurnDiff(toolCallID string, diff acpport.ToolDiff) {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	if !f.turnActive || f.turnDiffs == nil {
		return
	}

	f.turnDiffs[toolCallID] = diff
}

// turnDiffFor snapshots one tool's recorded diff under the slot lock.
func (f *Front) turnDiffFor(toolCallID string) (acpport.ToolDiff, bool) {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	diff, ok := f.turnDiffs[toolCallID]

	return diff, ok
}

// turnBreach returns the active turn's recorded stream-ceiling breach, if
// any (the WR-03 fail-loud path). Prompt gates on this BEFORE mapping the
// machine outcome: a turn that dropped streamed output at the ceiling must
// never answer a success stopReason.
func (f *Front) turnBreach() error {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	if f.turnPump == nil {
		return nil
	}

	return f.turnPump.breachErr()
}

// createSessionID digs result.session.sessionId out of a session/setup
// reply (the happy-turn.json corpus shape — session/create and the SESS-01
// session/resume carry the SAME full snapshot shape, 05-RESEARCH.md
// Pattern 7); a reply without the id fails loud — the session registry
// must never hold an empty key. The id digs out through a map because the
// wire key is the dialect's camelCase (the tagliatelle contract:
// production decode structs stay snake-safe). method names the calling
// wire method in the error surface.
func createSessionID(res json.RawMessage, method string) (string, error) {
	var result struct {
		Session json.RawMessage `json:"session"`
	}

	if err := json.Unmarshal(res, &result); err != nil {
		return "", acpport.NewInternal("%s: decode reply: %v", method, err)
	}

	var session map[string]any
	if err := json.Unmarshal(result.Session, &session); err != nil {
		return "", acpport.NewInternal("%s: decode session: %v", method, err)
	}

	sid, _ := session["sessionId"].(string)
	if sid == "" {
		return "", acpport.NewInternal("%s: reply carries no session.sessionId", method)
	}

	return sid, nil
}
