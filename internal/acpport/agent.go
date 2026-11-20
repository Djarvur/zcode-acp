package acpport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

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

// NewAgent wraps a Handlers impl as an acp.Agent — the D-01 translation
// point (RESEARCH Pattern 1): production wiring passes the result to
// NewAgentSideConnection; the adapter carries no state beyond the
// Handlers reference and the connection-bound sink.
func NewAgent(h Handlers) acp.Agent {
	return &agentAdapter{h: h}
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

// Initialize translates the neutral InitReport onto the honest D-02 wire
// response via initializeResponse; front errors surface as Go errors and
// are coded at the dispatch layer.
func (a *agentAdapter) Initialize(ctx context.Context, params acp.InitializeRequest) (acp.InitializeResponse, error) {
	report, err := a.h.Initialize(ctx)
	if err != nil {
		return acp.InitializeResponse{}, err
	}

	return initializeResponse(report), nil
}

// initializeResponse assembles the honest wire report from the neutral
// InitReport: protocol version 1, real agentInfo, the advertised auth
// methods as agent-variant entries, and the zero-value AgentCapabilities
// (the SDK struct has no fs/terminal fields at all — honesty by
// construction, D-02/A7; omission is spec-correct).
func initializeResponse(report InitReport) acp.InitializeResponse {
	title := report.Title

	authMethods := make([]acp.AuthMethod, 0, len(report.AuthMethods))
	for _, m := range report.AuthMethods {
		agent := &acp.AuthMethodAgent{Id: acp.AuthMethodId(m.ID), Name: m.Name}
		if m.Description != "" {
			description := m.Description
			agent.Description = &description
		}

		authMethods = append(authMethods, acp.AuthMethod{Agent: agent})
	}

	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentInfo: &acp.Implementation{
			Name:    report.Name,
			Title:   &title,
			Version: report.Version,
		},
		AuthMethods: authMethods,
	}
}

// NewSession delegates to Handlers and wraps the front's session id in
// the wire response (D-03 contract lives in the front).
func (a *agentAdapter) NewSession(ctx context.Context, params acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	sessionID, err := a.h.NewSession(ctx, newSessionParams(params))
	if err != nil {
		return acp.NewSessionResponse{}, err
	}

	return acp.NewSessionResponse{SessionId: acp.SessionId(sessionID)}, nil
}

// newSessionParams translates the create request into the neutral DTO
// (cwd verbatim; the MCP list shallow-copied out of SDK types — the front
// accepts and ignores it in Phase 4, D-03).
func newSessionParams(req acp.NewSessionRequest) NewSessionParams {
	servers := make([]McpServerDef, 0, len(req.McpServers))
	for _, srv := range req.McpServers {
		servers = append(servers, mcpServerDef(srv))
	}

	return NewSessionParams{Cwd: req.Cwd, McpServers: servers}
}

// mcpServerDef shallow-copies one client-declared MCP server; the
// transport kind derives from whichever union variant is present.
func mcpServerDef(srv acp.McpServer) McpServerDef {
	def := McpServerDef{}

	switch {
	case srv.Stdio != nil:
		def.Kind, def.Name = "stdio", srv.Stdio.Name
	case srv.Http != nil:
		def.Kind, def.Name = "http", srv.Http.Name
	case srv.Sse != nil:
		def.Kind, def.Name = "sse", srv.Sse.Name
	case srv.Acp != nil:
		def.Kind, def.Name = "acp", srv.Acp.Name
	}

	return def
}

// Prompt delegates the turn to Handlers with the joined content and the
// connection-bound sink (D-04 streaming), then maps the neutral stop
// reason onto the wire. Front errors are coded via wireError; unmapped
// stop reasons never leak as results (A5).
func (a *agentAdapter) Prompt(ctx context.Context, params acp.PromptRequest) (acp.PromptResponse, error) {
	reason, err := a.h.Prompt(ctx, PromptParams{
		SessionID: string(params.SessionId),
		Text:      joinContentBlocks(params.Prompt),
	}, a.sinkForPrompt())
	if err != nil {
		return acp.PromptResponse{}, wireError(err)
	}

	switch reason {
	case StopEndTurn:
		return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
	case StopCancelled:
		return acp.PromptResponse{StopReason: acp.StopReasonCancelled}, nil
	default:
		return acp.PromptResponse{}, acp.NewInternalError(
			fmt.Sprintf("acpport: unmapped stop reason %q (the minimal loop maps end_turn/cancelled only)", reason),
		)
	}
}

// joinContentBlocks flattens the prompt content into one string: text
// blocks verbatim, resource_link blocks as their URI (A3), image/audio
// blocks skipped (Phase 4 advertises the text baseline only).
func joinContentBlocks(blocks []acp.ContentBlock) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		switch {
		case block.Text != nil:
			parts = append(parts, block.Text.Text)
		case block.ResourceLink != nil:
			parts = append(parts, block.ResourceLink.Uri)
		}
	}

	return strings.Join(parts, " ")
}

// sinkForPrompt returns the bound connection sink; the construction
// discipline makes the bound path the only real path, but a direct
// NewAgent misuse gets a loud error instead of a nil-deref panic.
func (a *agentAdapter) sinkForPrompt() UpdateSink {
	if a.sink != nil {
		return a.sink
	}

	return func(ctx context.Context, note SessionNote) error {
		return NewInternal("acpport: no connection bound — construct via NewAgentSideConnection")
	}
}

// Cancel forwards the session id to Handlers (D-05 — the front owns the
// exactly-once cancel machinery).
func (a *agentAdapter) Cancel(ctx context.Context, params acp.CancelNotification) error {
	return a.h.Cancel(ctx, string(params.SessionId))
}

// wireError maps a front error onto the SDK's coded RequestError: seam
// errors carry their own code; anything else is an internal error (A5).
func wireError(err error) error {
	var seamErr *Error
	if errors.As(err, &seamErr) {
		return acp.NewRequestError(seamErr.Code, seamErr.Message, nil)
	}

	return acp.NewInternalError(err.Error())
}

// Authenticate answers the generic unadvertised default in Phase 4
// (deliberate -32601 exception, Open Question 2 — authMethods are
// advertised per D-02, the flow lands in Phases 5-6); plan 04-03 flips it
// to the honest not-available error.
func (a *agentAdapter) Authenticate(ctx context.Context, params acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, unadvertised(acp.AgentMethodAuthenticate)
}

// Logout is unadvertised in Phase 4 — clean -32601 default (Pitfall 6).
func (a *agentAdapter) Logout(ctx context.Context, params acp.LogoutRequest) (acp.LogoutResponse, error) {
	return acp.LogoutResponse{}, unadvertised(acp.AgentMethodLogout)
}

// CloseSession is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) CloseSession(ctx context.Context, params acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	return acp.CloseSessionResponse{}, unadvertised(acp.AgentMethodSessionClose)
}

// DeleteSession is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) DeleteSession(ctx context.Context, params acp.DeleteSessionRequest) (acp.DeleteSessionResponse, error) {
	return acp.DeleteSessionResponse{}, unadvertised(acp.AgentMethodSessionDelete)
}

// ListSessions is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) ListSessions(ctx context.Context, params acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, unadvertised(acp.AgentMethodSessionList)
}

// ResumeSession is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) ResumeSession(ctx context.Context, params acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, unadvertised(acp.AgentMethodSessionResume)
}

// SetSessionConfigOption is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) SetSessionConfigOption(ctx context.Context, params acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, unadvertised(acp.AgentMethodSessionSetConfigOption)
}

// SetSessionMode is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) SetSessionMode(ctx context.Context, params acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, unadvertised(acp.AgentMethodSessionSetMode)
}

// unadvertised is the -32601 default for the stable methods this build
// does not advertise (Pitfall 6): capability omission stops legitimate
// clients, but the dispatcher still routes here — answer cleanly, naming
// the method.
func unadvertised(method string) error {
	return acp.NewMethodNotFound(method)
}
