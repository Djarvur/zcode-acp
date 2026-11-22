package acpfront

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

	// authMode records the authenticate outcome the runtime-headers source
	// selects on (D-11, plan 06-05): AuthModeNone until an authenticate
	// arm's read-back-proven success stores its mode (a failed or cancelled
	// authenticate records nothing — no half-authenticated state is
	// observable). Atomic: reads ride any goroutine; the Authenticate
	// dispatch is the single write point.
	authMode atomic.Value

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

	// settingsMu guards the per-session confirmed-settings store (session
	// id → the latest confirmed backend reply settings snapshot, plan
	// 06-04): recorded at session/new from the create reply and AGAIN from
	// the setModel follow-up reply when it fired (the later reply
	// overwrites — the follow-up IS the D-10 confirmation), dropped at
	// teardown beside dropCwd (the WR-07 bounding — the map stays bounded
	// with the registry).
	settingsMu sync.Mutex
	settings   map[string]json.RawMessage

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
		settings: make(map[string]json.RawMessage),
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
// returns. mcpServers are accepted and ignored (D-03). The widened
// Handlers contract (plan 06-04) returns the session's config options —
// the confirmed-reply capture that fills them lands in Task 3 (06-04
// sequencing: the signature widens here, the settings store follows).
func (f *Front) NewSession(
	ctx context.Context, params acpport.NewSessionParams,
) (string, []acpport.ConfigOptionDef, error) {
	// Spec input validation (RESEARCH Security Domain V5, T-4-01): the ACP
	// session-setup contract makes cwd MUST-be-absolute — validate BEFORE
	// any backend call so a rejected session/new never touches the back-leg
	// wire (the fake transcript must show no session/create frame). The
	// empty-cwd case folds in: an empty string is not absolute either.
	if !filepath.IsAbs(params.Cwd) {
		return "", nil, acpport.NewInvalidParams(
			"session/new: cwd must be absolute (spec session-setup MUST), got %q", params.Cwd)
	}

	res, err := f.conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		wireKeyWorkspace: map[string]any{wireKeyWorkspacePath: params.Cwd, wireKeyWorkspaceKey: params.Cwd},
	})
	if err != nil {
		return "", nil, acpport.NewInternal("session/create: %v", err)
	}

	sid, err := createSessionID(res, "session/create")
	if err != nil {
		return "", nil, err
	}

	f.recordCwd(sid, params.Cwd)

	if _, err := f.conn.Call(ctx, zcode.MethodSessionSubscribe, map[string]any{
		wireKeySessionID: sid,
		"deliveryKind":   "desktop-continuous",
	}); err != nil {
		// WR-04 (plan 05-05): the create succeeded — tear the backend
		// session down best-effort instead of orphaning it (T-5-12).
		f.closeOrphan(ctx, sid, "subscribe", err)

		return "", nil, acpport.NewInternal("session/subscribe: %v", err)
	}

	// The create reply IS the first confirmed snapshot (Pattern 7's
	// reply-snapshot fact): capture it BEFORE the follow-up so a snapshot-
	// carrying reply renders even with no model configured (the no-model
	// arm).
	f.recordSettings(sid, res)

	if err := f.applyConfiguredModel(ctx, sid); err != nil {
		return "", nil, err
	}

	// The machine registers on the conn's fan-out AFTER the tap (constructed
	// earlier, front.go New) — the flush-before-stopReason registration
	// order. Phase-3 watcher defaults; corpus-grade terminal/cancel signals
	// resolve turns well inside the quiet-window gates.
	machine := turn.NewMachine(f.conn, sid, turn.NewPollWatcher(f.conn, turn.PollConfig{}))
	f.registry.Register(core.Session{ID: sid, Machine: machine})

	// The confirmed snapshot -> configOptions (MODEL-01): the latest
	// recorded snapshot — the setModel follow-up's when it fired, the
	// create reply's otherwise — translates through the verbatim mirror;
	// an unbound session renders no options at all (D-10 + locked OQ2:
	// never a fabricated currentValue). The reply IS the confirmation
	// source (Pitfall 6): nothing here reads any other state.
	confirmed, ok := f.latestSettings(sid)

	var options []acpport.ConfigOptionDef

	if ok {
		if s, parsed := ParseSettings(confirmed); parsed {
			options = BuildConfigOptions(s)
		}
	}

	slog.Info("acpfront: session opened", "session", sid)

	return sid, options, nil
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

	// SESS-03 slash intercept (plan 07-05, D-03): the omp dialect sends
	// model switches as prompt turns — the command form is consumed HERE,
	// before acquireTurn, so a refused switch never holds the front-wide
	// turn slot (a refusal and a normal prompt never serialize behind it).
	if stop, handled, slashErr := f.promptModelSlash(sdkCtx, params, sink); handled {
		return stop, slashErr
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

// modelSlashSelector detects the omp dialect's /model command form in one
// prompt text (SESS-03, plan 07-05, D-03) and returns the selector: the
// command alone (or with an empty remainder) yields the empty selector —
// the no-model refusal arm; the remainder after ONE separating space is the
// selector verbatim (no trimming, no case folding — the byte-equality
// contract). "/modelfoo" is NOT the command: it reports not-a-command and
// the caller treats the prompt as a literal turn.
func modelSlashSelector(text string) (selector string, isCommand bool) {
	if text == modelSlashCommand {
		return "", true
	}

	if rest, ok := strings.CutPrefix(text, modelSlashSpace); ok {
		return rest, true
	}

	return "", false
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

// The config-option ids the translator renders (configoptions.go) — the
// picker's switch keys, the validation whitelist's vocabulary. The model
// id's string doubles as the setModel params' wire key (the dialect spells
// both the same).
const (
	optionIDModel   = "model"
	optionIDThought = "thought_level"
)

// The omp dialect's /model slash-prompt command form (SESS-03, plan 07-05,
// D-03): model switching is a prompt convention, not a method — the prompt
// text is exactly the command or the command plus ONE separating space
// (case-sensitive; "/modelfoo" is NOT the command and falls through as a
// normal prompt). The happy switch's one short confirmation text chunk is
// worded freely (A2 — the client classifies only refusals, never success
// text); the refusal strings carry their own pinned contracts (plan Task 2).
const (
	modelSlashCommand = "/model"
	modelSlashSpace   = modelSlashCommand + " "

	modelSwitchedConfirmationPrefix = "Model switched to "
)

// The pinned omp-dialect refusal strings (SESS-03, plan 07-05, D-03; the
// dialect doc's byte-for-byte contracts — startsWith-matched by the
// client's matchers on code -32603 + data.details). Each lives in exactly
// ONE site here; the refusal arms reference them verbatim — never a
// paraphrase (SESS-03's acceptance is string-exact).
const (
	unknownACPModelFormat = "Unknown ACP model: %s"
	noModelSelectedDetail = "No model selected."
)

// The setModel params' remaining camelCase wire keys (the capture-verbatim
// dialect spellings — the wireKeySessionID family).
const (
	wireKeyProviderID      = "providerId"
	wireKeyModelID         = "modelId"
	wireKeyReasoningLevel  = "reasoningLevel"
	wireKeyPersistLastUsed = "persistAsWorkspaceLastUsed"

	// wireKeyWorkspace/wireKeyWorkspacePath/wireKeyWorkspaceKey are the
	// session-create params' workspace-object keys (the probe sessions'
	// temp-dir cwd shape).
	wireKeyWorkspace     = "workspace"
	wireKeyWorkspacePath = "workspacePath"
	wireKeyWorkspaceKey  = "workspaceKey"
)

// SetSessionConfigOption applies one client config-option selection
// (MODEL-02, plan 06-06): validate FIRST against the CURRENT confirmed
// option set — the client's configId/value are untrusted input, and the
// -32602 arms name the configId and the allowed set/values so the client
// can re-render its picker (T-06-17: no backend call ever carries an
// unvalidated value). The closed-session gate rides the established SESS-01
// posture (isClosing — the T-5-12 race-gate parity with prompt/resume), and
// the confirmed-settings store is the validation source: a session with no
// confirmed snapshot (never opened, or torn down — dropSettings rode the
// SESS-01 teardown beside dropCwd) answers -32602, never a panic. A valid
// ask maps onto the locked OQ1 wire path (setConfigMutation), the store
// updates FROM THE MUTATION REPLY (D-10 — the reply IS the confirmation
// source, so a fabricated currentValue is impossible by construction: the
// re-emit consumes the store the reply just overwrote), and the FULL
// rebuilt list returns — the spec MUST the adapter's response carries.
func (f *Front) SetSessionConfigOption(
	ctx context.Context, sessionID, configID, value string,
) ([]acpport.ConfigOptionDef, error) {
	if f.isClosing(sessionID) {
		return nil, acpport.NewInvalidRequest(
			"session/set_config_option: %s: the session is closing or closed", sessionID)
	}

	raw, ok := f.latestSettings(sessionID)
	if !ok {
		return nil, acpport.NewInvalidParams(
			"session/set_config_option: %s: no confirmed settings to validate against (unknown session)", sessionID)
	}

	s, parsed := ParseSettings(raw)
	if !parsed {
		return nil, acpport.NewInternal(
			"session/set_config_option: %s: the confirmed settings snapshot does not decode", sessionID)
	}

	options := BuildConfigOptions(s)

	def := configOptionByID(options, configID)
	if def == nil {
		return nil, acpport.NewInvalidParams(
			"session/set_config_option: unknown configId %q (allowed: %s)", configID, allowedOptionIDs(options))
	}

	if !configOptionListed(def, value) {
		return nil, acpport.NewInvalidParams(
			"session/set_config_option: configId %q: value %q is not one of the listed values (%s)",
			configID, value, listedOptionValues(def))
	}

	params, err := setConfigMutation(sessionID, s, configID, value)
	if err != nil {
		return nil, err
	}

	res, err := f.conn.Call(ctx, zcode.MethodSessionSetModel, params)
	if err != nil {
		// WR-01 (plan 07-01): the full SESS-01 teardown — never a zombie
		// front session against a dead mutation target.
		f.teardownAfterPickerRefusal(ctx, sessionID, err)

		return nil, acpport.NewInternal("session/setModel: %v", err)
	}

	// The reply IS the confirmed state (D-10): the store overwrites from the
	// reply BEFORE the rebuild — the response-before-store ordering is
	// impossible by construction (the rebuild reads the store back).
	f.recordSettings(sessionID, res)

	confirmed, ok := f.latestSettings(sessionID)
	if !ok {
		return nil, acpport.NewInternal(
			"session/set_config_option: %s: the mutation reply snapshot did not record", sessionID)
	}

	cs, parsed := ParseSettings(confirmed)
	if !parsed {
		return nil, acpport.NewInternal(
			"session/set_config_option: %s: the mutation reply settings do not decode", sessionID)
	}

	return BuildConfigOptions(cs), nil
}

// DeleteSession answers the dialect's session/delete (SESS-02, plan 07-03,
// D-01): a REGISTERED id delegates to the SESS-01 teardown (CloseSession —
// cancel-drain → backend close → registry release → observer close → map
// drops, the 07-01 contract; the reply on success is nil and the teardown's
// coded errors surface verbatim), and an UNKNOWN or already-closed id
// answers success IMMEDIATELY — the spec's silent-success arm — without any
// backend call (the runtime owns real deletion; there is nothing on the
// back leg to delete: "session/delete" occurs ×0 in the app-server bundle).
// The method stays unadvertised (the plan Task-3 honesty lock): the graceful
// arm serves spec-loose clients only — Orca gates the call client-side and
// must never start calling it.
func (f *Front) DeleteSession(ctx context.Context, sessionID string) error {
	if _, err := f.registry.Get(sessionID); err != nil {
		//nolint:nilerr // the spec's silent-success arm BY CONTRACT (D-01): an
		// unknown/already-closed id answers the graceful success — the registry
		// miss IS the arm's trigger, not an error to surface.
		return nil
	}

	return f.CloseSession(ctx, sessionID)
}

// promptModelSlash runs the /model slash-prompt intercept (SESS-03, plan
// 07-05, D-03): a prompt turn whose text is the command form resolves the
// selector against the session's confirmed configOptions listed values
// (validate-first, the T-06-17 posture — a refusal is emitted before any
// backend call and no backend setModel ever carries an unvalidated string)
// and, on a match, applies it through the SAME machinery the picker's model
// arm uses (setConfigMutation: the confirmed reasoning level, explicit
// persist-false) — then resolves as a normal completed turn with one short
// confirmation text chunk (A2). The handled return carries the turn's
// outcome; a not-a-command text reports handled=false and falls through to
// the normal turn path.
//
// The intercept runs BEFORE acquireTurn (the Prompt call site), so a
// refused switch never holds the front-wide turn slot; the busy gate below
// keeps that placement from becoming a bypass — mid-turn commands still
// take the CORE-05 rejection instead of racing the slot.
func (f *Front) promptModelSlash(
	ctx context.Context, params acpport.PromptParams, sink acpport.UpdateSink,
) (acpport.StopReason, bool, error) {
	selector, isCommand := modelSlashSelector(params.Text)
	if !isCommand {
		return "", false, nil
	}

	// WR-02 (07-REVIEW): the session-scoped busy gate. A command prompt
	// issued while THIS session's turn is active takes the CORE-05
	// turnActiveMessage rejection — the same surface any literal prompt
	// gets from the slot — instead of issuing a mid-turn setModel mutation
	// whose refusal would run the picker-refusal teardown (CloseSession →
	// drainActiveTurn) and cancel the user's RUNNING turn. Session-scoped
	// under the slot lock (the Cancel WR-07 discipline): another session's
	// turn on the front-wide slot does not block this session's switch.
	f.turnMu.Lock()
	busy := f.turnActive && f.turnSession == params.SessionID
	f.turnMu.Unlock()

	if busy {
		return "", true, acpport.NewInvalidRequest(turnActiveMessage)
	}

	if selector == "" {
		// The no-argument arm: nothing to resolve — the pinned no-model
		// sentence rides the detail carriage (-32603 + data.details).
		return "", true, acpport.NewInternalWithData(noModelSelectedDetail)
	}

	// The validation source is the SAME confirmed-settings read the picker
	// uses (latestSettings -> BuildConfigOptions): the listed values are the
	// providerId/modelId serializations (configoptions.go), compared byte-
	// exact — no case folding, no trimming of the selector itself.
	raw, ok := f.latestSettings(params.SessionID)
	if !ok {
		return "", true, acpport.NewInternalWithData(noModelSelectedDetail)
	}

	s, parsed := ParseSettings(raw)
	if !parsed {
		return "", true, acpport.NewInternal(
			"session/prompt /model: %s: the confirmed settings snapshot does not decode", params.SessionID)
	}

	def := configOptionByID(BuildConfigOptions(s), optionIDModel)
	if def == nil {
		// The snapshot binds no model (D-10: an unbound session renders no
		// model option) — the pinned no-model sentence rides the detail
		// carriage.
		return "", true, acpport.NewInternalWithData(noModelSelectedDetail)
	}

	if !configOptionListed(def, selector) {
		// The validate-first refusal: never a backend call — the pinned
		// unknown-selector detail rides the detail carriage (the Hb
		// matchers read ONLY -32603 + data.details, Pitfall 1).
		return "", true, acpport.NewInternalWithData(unknownACPModelFormat, selector)
	}

	mutation, err := setConfigMutation(params.SessionID, s, optionIDModel, selector)
	if err != nil {
		return "", true, err
	}

	res, err := f.conn.Call(ctx, zcode.MethodSessionSetModel, mutation)
	if err != nil {
		// WR-01 parity (plan 07-01): a refused slash setModel runs the same
		// full SESS-01 teardown as the picker refusal — never a zombie front
		// session against a dead mutation target.
		f.teardownAfterPickerRefusal(ctx, params.SessionID, err)

		return "", true, acpport.NewInternal("session/setModel: %v", err)
	}

	// The reply IS the confirmed state (D-10) — the same settings update the
	// picker performs.
	f.recordSettings(params.SessionID, res)

	// A2: one short confirmation text chunk — the client classifies only
	// refusals (error frames), never success text.
	if err := sink(ctx, acpport.SessionNote{
		SessionID: params.SessionID,
		Kind:      acpport.NoteMessage,
		Delta:     modelSwitchedConfirmationPrefix + selector,
	}); err != nil {
		return "", true, acpport.NewInternal("session/prompt: %v", err)
	}

	return acpport.StopEndTurn, true, nil
}

// teardownAfterPickerRefusal runs the WR-01 full SESS-01 teardown after a
// refused picker setModel on an ESTABLISHED session (the wire-only orphan
// close left the front session registered against a dead mutation target —
// the zombie: later prompts sent to a dead backend, later picker asks
// re-failed). The teardown's own failure logs structurally — never a
// silent skip. The client-facing refusal error is unchanged.
func (f *Front) teardownAfterPickerRefusal(ctx context.Context, sessionID string, cause error) {
	if err := f.CloseSession(ctx, sessionID); err != nil {
		slog.Warn("acpfront: picker-refusal teardown failed",
			"session", sessionID, "stage", "set_config_option setModel", "cause", cause, "err", err)
	}
}

// setConfigMutation maps a VALIDATED config-option ask onto the locked OQ1
// wire path: BOTH switch kinds ride session/setModel. A model switch parses
// the value with strings.Cut on the FIRST slash only (no composite forms —
// the backend's display-only $ separator stays out of the wire contract)
// and carries options.reasoningLevel = the CURRENT confirmed level unless
// the ask changes it (Pitfall 2: a switch must never silently drop the
// level — the field-proven binding path per EXP-TURN-01); a thought-level
// switch carries the CONFIRMED current model unchanged with the requested
// level. persistAsWorkspaceLastUsed stays false on both paths (the Open
// Question 3 posture), and the options object emits the documented
// omit-shape whenever no confirmed level exists (IN-04).
func setConfigMutation(
	sessionID string, s Settings, configID, value string,
) (map[string]any, error) {
	switch configID {
	case optionIDModel:
		providerID, modelID, cut := strings.Cut(value, "/")
		if !cut {
			// Unreachable through the validated path (the model option's
			// listed values are the providerId/modelId serialization) —
			// fail loud if the invariant ever breaks.
			return nil, acpport.NewInternal(
				"session/set_config_option: configId %q: value %q is not the providerId/modelId serialization",
				configID, value)
		}

		model := map[string]any{wireKeyProviderID: providerID, wireKeyModelID: modelID}

		if level := confirmedLevel(s); level != "" {
			model["options"] = map[string]any{wireKeyReasoningLevel: level}
		}

		return map[string]any{
			wireKeySessionID:       sessionID,
			optionIDModel:          model,
			wireKeyPersistLastUsed: false,
		}, nil

	case optionIDThought:
		bound := s.Model.Current
		if bound == nil {
			bound = s.Model.LastUsed
		}

		if bound == nil {
			// The thought option renders only for a confirmed-level session;
			// with no confirmed model binding there is nothing honest to
			// carry on the locked setModel path — never an empty model
			// object on the wire.
			return nil, acpport.NewInternal(
				"session/set_config_option: configId %q: no confirmed model binding to carry (locked OQ1)", configID)
		}

		return map[string]any{
			wireKeySessionID: sessionID,
			optionIDModel: map[string]any{
				wireKeyProviderID: bound.ProviderID,
				wireKeyModelID:    bound.ModelID,
				"options":         map[string]any{wireKeyReasoningLevel: value},
			},
			wireKeyPersistLastUsed: false,
		}, nil

	default:
		// Unreachable through the validated path — fail loud, never guess.
		return nil, acpport.NewInternal("session/set_config_option: unmapped configId %q", configID)
	}
}

// confirmedLevel reads the CURRENT confirmed reasoning level off the
// snapshot (settings.thoughtLevel.current under enabled — the D-10 source);
// empty when the session carries none (the omit-shape arm, IN-04).
func confirmedLevel(s Settings) string {
	if !s.ThoughtLevel.Enabled {
		return ""
	}

	return s.ThoughtLevel.Current
}

// configOptionByID finds one rendered option def by its wire id.
func configOptionByID(options []acpport.ConfigOptionDef, configID string) *acpport.ConfigOptionDef {
	for i := range options {
		if options[i].ID == configID {
			return &options[i]
		}
	}

	return nil
}

// allowedOptionIDs renders the option ids the client may pick — the unknown
// configId error's allowed set ("none" for a snapshot that renders no
// options at all — the honest smaller-list posture).
func allowedOptionIDs(options []acpport.ConfigOptionDef) string {
	ids := make([]string, 0, len(options))
	for i := range options {
		ids = append(ids, options[i].ID)
	}

	if len(ids) == 0 {
		return "none"
	}

	return strings.Join(ids, ", ")
}

// configOptionListed reports whether value is a member of the option's own
// listed set (the spec MUST: the value must be one of the option's options
// array).
func configOptionListed(def *acpport.ConfigOptionDef, value string) bool {
	for _, o := range def.Options {
		if o.Value == value {
			return true
		}
	}

	return false
}

// listedOptionValues renders the option's own listed values — the
// out-of-set value error's allowed set.
func listedOptionValues(def *acpport.ConfigOptionDef) string {
	values := make([]string, 0, len(def.Options))
	for _, o := range def.Options {
		values = append(values, o.Value)
	}

	return strings.Join(values, ", ")
}

// applyConfiguredModel runs the session/setModel follow-up ONLY when a
// model selection is configured (Open Question 4 resolution) with explicit
// persistAsWorkspaceLastUsed:false (Open Question 3 resolution), then lets
// the follow-up reply overwrite the create snapshot as the CONFIRMED state
// (D-10 — an unconfirmed target is never shown selected). A no-op answer
// (nil error, nothing recorded) when no model is bound — the create reply's
// snapshot stays the confirmed one. A refused follow-up fails the caller
// with the coded internal error exactly as the inline arm did.
func (f *Front) applyConfiguredModel(ctx context.Context, sid string) error {
	if f.model == nil {
		return nil
	}

	// The reasoning level's default chain (plan 07-04, D-03): the explicit
	// configured level first, else the target model's backend-confirmed
	// defaultLevel from the create-reply snapshot (recorded just above the
	// follow-up), else "" — the IN-04 omit-shape whose backend refusal is
	// the honest answer for level-strict providers. Never a fabricated
	// level: an unlisted target answers "".
	level := f.model.ReasoningLevel
	if level == "" {
		level = f.snapshotDefaultLevel(sid, f.model.ProviderID, f.model.ModelID)
	}

	// IN-04 (plan 05-05): the options object rides ONLY when a level is
	// configured — an empty level emits the documented omit-shape (never
	// `"options":{"reasoningLevel":""}`), so the backend's own actionable
	// refusal surfaces for level-strict providers. The configured-with-
	// level preamble stays byte-locked by the corpus.
	model := map[string]any{
		"providerId":   f.model.ProviderID,
		wireKeyModelID: f.model.ModelID,
	}

	if level != "" {
		model["options"] = map[string]any{wireKeyReasoningLevel: level}
	}

	setModelRes, err := f.conn.Call(ctx, zcode.MethodSessionSetModel, map[string]any{
		wireKeySessionID:       sid,
		optionIDModel:          model,
		wireKeyPersistLastUsed: false,
	})
	if err != nil {
		// WR-04 (plan 05-05): the create succeeded — the same orphan
		// discipline as the subscribe refusal: every post-create failure
		// closes the backend session best-effort instead of orphaning it
		// (T-5-12).
		f.closeOrphan(ctx, sid, "setModel", err)

		return acpport.NewInternal("session/setModel: %v", err)
	}

	// The follow-up reply is the CONFIRMED state (D-10): it overwrites
	// the create snapshot — an unconfirmed target is never shown
	// selected.
	f.recordSettings(sid, setModelRes)

	return nil
}

// snapshotDefaultLevel derives the target model's backend-confirmed default
// reasoning level from the session's latest recorded snapshot (plan 07-04,
// D-03's level-default chain): the create reply is recorded before the
// setModel follow-up fires, so a target the snapshot lists answers its
// defaultLevel; an unlisted target, a level-less entry, or an undecodable
// snapshot answers "" — the omit-shape arm, never a fabricated level.
func (f *Front) snapshotDefaultLevel(sid, providerID, modelID string) string {
	raw, ok := f.latestSettings(sid)
	if !ok {
		return ""
	}

	s, parsed := ParseSettings(raw)
	if !parsed {
		return ""
	}

	for _, entry := range s.Model.Available {
		if entry.Ref.ProviderID == providerID && entry.Ref.ModelID == modelID {
			return entry.DefaultLevel
		}
	}

	return ""
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

// recordSettings stores one session's latest confirmed settings snapshot
// (plan 06-04): session/new records the create reply's snapshot and again
// the setModel follow-up's — the later reply overwrites, because the
// follow-up reply IS the server-confirmed state (D-10). The store is
// initialized by the ONLY constructor (New) and never reassigned — no
// re-initialization lives here (IN-04, plan 07-01: the dead nil-guard is
// gone; the sibling lazy-init of closing in CloseSession is load-bearing
// and stays).
func (f *Front) recordSettings(sessionID string, raw json.RawMessage) {
	f.settingsMu.Lock()
	defer f.settingsMu.Unlock()

	f.settings[sessionID] = raw
}

// latestSettings copies one session's confirmed settings snapshot out
// under the store lock; an unknown session (or one whose replies carried
// no snapshot) answers absent — the omission arms render honestly.
func (f *Front) latestSettings(sessionID string) (json.RawMessage, bool) {
	f.settingsMu.Lock()
	defer f.settingsMu.Unlock()

	raw, ok := f.settings[sessionID]

	return raw, ok
}

// dropSettings removes one session's confirmed-settings entry (SESS-01
// teardown, beside dropCwd — the map stays bounded with the registry, the
// WR-07 discipline).
func (f *Front) dropSettings(sessionID string) {
	f.settingsMu.Lock()
	defer f.settingsMu.Unlock()

	delete(f.settings, sessionID)
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
// It DOES drop the session's recorded cwd/settings entries (WR-04, plan
// 07-01): closeOrphan is the failure-path closer, so the WR-07 bounding
// invariant belongs here — every post-create failure path leaves zero map
// entries behind. The drops ride BEFORE the wire attempt so a failed
// best-effort close cannot skip them.
func (f *Front) closeOrphan(ctx context.Context, sessionID, stage string, cause error) {
	f.dropCwd(sessionID)
	f.dropSettings(sessionID)

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
