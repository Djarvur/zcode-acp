package acpfront

import (
	"context"
	"sync"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/core"
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
	baseCtx  context.Context
	conn     *zcode.Conn
	registry *core.SessionRegistry
	model    *ModelSelection
	version  string

	// turnMu guards the front-wide turn slot (the single-active-turn
	// invariant) and the active turn's tap-facing state: the sink the tap
	// emits through and the turn context it writes on. Held for the turn's
	// whole lifetime — from Prompt entry to its return.
	turnMu     sync.Mutex
	turnActive bool
	turnCancel context.CancelFunc
	turnCtx    context.Context
	turnSink   acpport.UpdateSink
	turnSessID string
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

// New returns a Front owning conn and registers the delta tap on the
// conn's ordered observer fan-out BEFORE any machine exists — a machine
// constructed later (per NewSession) lands after the tap in the delivery
// order (RESEARCH Pattern 3). Production wiring wraps the result via
// acpport.NewAgent; the acpport connection binds the update sink that
// Prompt hands back into the front's turn slot.
func New(conn *zcode.Conn, opts ...Option) *Front {
	f := &Front{
		baseCtx:  context.Background(),
		conn:     conn,
		registry: core.NewSessionRegistry(),
		version:  "0.0.0-dev",
	}

	for _, opt := range opts {
		opt(f)
	}

	if conn != nil {
		conn.AddObserver(f.observeEnvelope) // the tap — BEFORE any machine
	}

	return f
}

// Initialize returns the honest D-02 report (RESEARCH Code Examples:
// executed-probe shape — protocolVersion 1 and the zero-value capabilities
// are assembled by the adapter).
func (f *Front) Initialize(ctx context.Context) (acpport.InitReport, error) {
	_ = ctx

	return acpport.InitReport{}, nil // TODO(04-06 GREEN): the honest report
}

// NewSession runs the D-03 contract on the back leg.
func (f *Front) NewSession(ctx context.Context, params acpport.NewSessionParams) (string, error) {
	_, _ = ctx, params

	return "", nil // TODO(04-06 GREEN): create → subscribe → setModel → register
}

// Prompt runs one turn (D-04/D-05).
func (f *Front) Prompt(ctx context.Context, params acpport.PromptParams, sink acpport.UpdateSink) (acpport.StopReason, error) {
	_, _, _ = ctx, params, sink

	return "", nil // TODO(04-06 GREEN): turn slot → front-owned ctx → machine.Run
}

// Cancel cancels the session's active turn (D-05).
func (f *Front) Cancel(ctx context.Context, sessionID string) error {
	_, _ = ctx, sessionID

	return nil // TODO(04-06 GREEN): registry lookup → machine.Cancel
}
