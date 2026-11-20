package acpport

import (
	"context"
	"io"

	"github.com/coder/acp-go-sdk"
)

// agentAdapter implements the SDK's 12-method acp.Agent surface over a
// Handlers impl (RESEARCH Pattern 1, the D-01 seam): the four advertised
// methods translate the neutral vocabulary, and the eight unadvertised
// stable methods answer acp.NewMethodNotFound (-32601, Pitfall 6) —
// matching the SDK's own optional-interface behavior. Capability omission
// stops legitimate clients; the dispatcher still routes here, so the
// defaults must be clean.
//
// Construction discipline: NewAgentSideConnection binds the update sink
// before it returns, and no frame can flow before that (the pipes are
// empty at construction), so Handlers always see a usable sink — the
// assign-before-traffic contract every conn hook in this repo follows.
type agentAdapter struct {
	h    Handlers
	sink UpdateSink
}

// sinkBinder is the hook NewAgentSideConnection uses to hand the
// connection-bound sink to our adapter before any frame flows.
type sinkBinder interface {
	bindSink(UpdateSink)
}

// bindSink attaches the connection-bound update sink; called once by
// NewAgentSideConnection during construction.
func (a *agentAdapter) bindSink(sink UpdateSink) {
	a.sink = sink
}

// NewAgentSideConnection wires any acp.Agent implementation (typically
// acpport.NewAgent over a Handlers impl) into an agent-side connection
// and returns the update sink bound to it. The SDK owns framing and
// dispatch; this wrapper owns nothing but the sink binding.
func NewAgentSideConnection(
	agent acp.Agent,
	peerInput io.Writer,
	peerOutput io.Reader,
) (*acp.AgentSideConnection, UpdateSink) {
	conn := acp.NewAgentSideConnection(agent, peerInput, peerOutput)

	sink := NewUpdateSink(conn)
	if binder, ok := agent.(sinkBinder); ok {
		binder.bindSink(sink)
	}

	return conn, sink
}

// NewAgent wraps a Handlers impl as an acp.Agent — the D-01 translation
// point (RESEARCH Pattern 1): production wiring passes the result to
// NewAgentSideConnection; the adapter carries no state beyond the
// Handlers reference and the connection-bound sink.
func NewAgent(h Handlers) acp.Agent {
	return &agentAdapter{h: h}
}

// RED stub (04-02 TDD): behavior lands in the GREEN step; the methods
// below compile against the full acp.Agent surface and fail the contract
// tests on their planned assertions.

// Initialize returns the zero response at RED — the honest D-02 assembly
// is the GREEN behavior under test.
func (a *agentAdapter) Initialize(ctx context.Context, params acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{}, nil
}

// NewSession returns an empty id at RED — delegation is the GREEN
// behavior under test.
func (a *agentAdapter) NewSession(ctx context.Context, params acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	return acp.NewSessionResponse{}, nil
}

// Prompt returns the zero response at RED — stop-reason mapping and sink
// handoff are the GREEN behaviors under test.
func (a *agentAdapter) Prompt(ctx context.Context, params acp.PromptRequest) (acp.PromptResponse, error) {
	return acp.PromptResponse{}, nil
}

// Cancel returns nil at RED — delegation is the GREEN behavior under test.
func (a *agentAdapter) Cancel(ctx context.Context, params acp.CancelNotification) error {
	return nil
}

// Authenticate answers the generic unadvertised default in Phase 4
// (deliberate -32601 exception, Open Question 2); plan 04-03 flips it to
// the honest not-available error.
func (a *agentAdapter) Authenticate(ctx context.Context, params acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

// Logout is unadvertised in Phase 4 — RED stub, GREEN answers -32601.
func (a *agentAdapter) Logout(ctx context.Context, params acp.LogoutRequest) (acp.LogoutResponse, error) {
	return acp.LogoutResponse{}, nil
}

// CloseSession is unadvertised in Phase 4 — RED stub, GREEN answers -32601.
func (a *agentAdapter) CloseSession(ctx context.Context, params acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	return acp.CloseSessionResponse{}, nil
}

// DeleteSession is unadvertised in Phase 4 — RED stub, GREEN answers -32601.
func (a *agentAdapter) DeleteSession(ctx context.Context, params acp.DeleteSessionRequest) (acp.DeleteSessionResponse, error) {
	return acp.DeleteSessionResponse{}, nil
}

// ListSessions is unadvertised in Phase 4 — RED stub, GREEN answers -32601.
func (a *agentAdapter) ListSessions(ctx context.Context, params acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, nil
}

// ResumeSession is unadvertised in Phase 4 — RED stub, GREEN answers -32601.
func (a *agentAdapter) ResumeSession(ctx context.Context, params acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, nil
}

// SetSessionConfigOption is unadvertised in Phase 4 — RED stub, GREEN answers -32601.
func (a *agentAdapter) SetSessionConfigOption(ctx context.Context, params acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, nil
}

// SetSessionMode is unadvertised in Phase 4 — RED stub, GREEN answers -32601.
func (a *agentAdapter) SetSessionMode(ctx context.Context, params acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}
