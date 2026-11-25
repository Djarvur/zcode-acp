package acpport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/coder/acp-go-sdk"
)

// agentAdapter implements the SDK's 12-method acp.Agent surface over a
// Handlers impl (RESEARCH Pattern 1, the D-01 seam): the advertised
// methods, the SESS-01 lifecycle pair (ResumeSession/CloseSession —
// plain requests per the SDK's dispatcher, no capability flag), the
// config-options mutation (SetSessionConfigOption — plan 06-06, the
// sessionCapabilities.options advertisement's serviced method), and the
// SESS-02 delete arm (plan 07-03 — registered ids run the full teardown,
// unknown ids answer the silent-success {}) translate the neutral
// vocabulary, the three remaining unadvertised stable methods answer
// acp.NewMethodNotFound (-32601, Pitfall 6) — matching the SDK's
// own optional-interface behavior — and authenticate (advertised per
// D-02) delegates through Handlers: the front's honest chain (AUTH-01,
// plan 05-07) is the only success path, and its errors code through
// wireError (D-02 honesty). Capability omission stops legitimate clients;
// the dispatcher still routes here, so the defaults must be clean.
//
// Construction discipline: NewAgentSideConnection binds the update sink
// before it returns, and no frame can flow before that (the pipes are
// empty at construction), so Handlers always see a usable sink — the
// assign-before-traffic contract every conn hook in this repo follows.
type agentAdapter struct {
	h Handlers

	// sink is the connection-bound update sink, held as an atomic pointer
	// (04-REVIEW WR-02, batched with this file's seam touch per the D-04
	// disposition): the publish (bindSink) and every prompt-time load
	// (sinkForPrompt) are synchronized, so a binding that lands while
	// traffic flows can never tear the func value.
	sink atomic.Pointer[UpdateSink]
}

// sinkBinder is the hook NewAgentSideConnection uses to hand the
// connection-bound sink to our adapter before any frame flows.
type sinkBinder interface {
	bindSink(sink UpdateSink)
}

type updateSinkBinder interface {
	bindUpdateSink(sink UpdateSink)
}

// permissionBinder is the adapter-side one-shot construction hook: it
// hands the connection-bound PermissionSender to the adapter, which
// forwards it into the Handlers impl when it opts in via the exported
// PermissionBinder (bind-at-construction, before any frame flows — the
// sinkBinder discipline).
type permissionBinder interface {
	bindPermissionSender(sender PermissionSender)
}

// PermissionBinder is the Handlers-side opt-in for the sender binding. The
// method is exported because Go qualifies unexported method names by their
// declaring package — an unexported-method interface can never be
// satisfied by another package's type, and the bridge lives in acpfront.
type PermissionBinder interface {
	BindPermissionSender(sender PermissionSender)
}

// UpdateSinkBinder opts a Handlers implementation into the construction-time
// connection sink binding used for front-originated asynchronous updates.
type UpdateSinkBinder interface {
	BindUpdateSink(sink UpdateSink)
}

// CatalogRefresher opts a Handlers implementation into a post-authenticate
// catalog refresh request. The request is advisory and cannot turn confirmed
// authentication into a failure.
type CatalogRefresher interface {
	RequestCatalogRefresh(ctx context.Context)
}

// NewAgent wraps a Handlers impl as the acp.Agent implementation — the
// D-01 translation point (RESEARCH Pattern 1): production wiring passes
// the result to NewAgentSideConnection; the adapter carries no state
// beyond the Handlers reference and the connection-bound sink.
func NewAgent(h Handlers) *agentAdapter {
	return &agentAdapter{h: h}
}

// NewAgentSideConnection wires any acp.Agent implementation (typically
// acpport.NewAgent over a Handlers impl) into an agent-side connection
// and returns the update sink and the permission sender bound to it. The
// SDK owns framing and dispatch; this wrapper owns nothing but the two
// bindings — both complete before the call returns, so no frame can flow
// before they exist (the pipes are empty at construction).
func NewAgentSideConnection(
	agent acp.Agent,
	peerInput io.Writer,
	peerOutput io.Reader,
) (*acp.AgentSideConnection, UpdateSink, PermissionSender) {
	conn := acp.NewAgentSideConnection(agent, peerInput, peerOutput)

	sink := NewUpdateSink(conn)
	if binder, ok := agent.(sinkBinder); ok {
		binder.bindSink(sink)
	}

	if binder, ok := agent.(updateSinkBinder); ok {
		binder.bindUpdateSink(sink)
	}

	sender := newPermissionSender(conn)
	if binder, ok := agent.(permissionBinder); ok {
		binder.bindPermissionSender(sender)
	}

	return conn, sink, sender
}

// newPermissionSender builds the connection-bound PermissionSender over
// the SDK's AgentSideConnection.RequestPermission (Don't-Hand-Roll: the
// SDK owns the request_permission round-trip and its pending table — this
// wrapper only translates the neutral ask onto the wire request and
// decodes the outcome union back into the neutral vocabulary).
//
// Options pass through verbatim (kind/name/optionId — the backend ask's
// mapping already happened ask-side; these are ACP kind strings here), the
// ToolCall patch carries the backend toolCallId (clients render the ask on
// the tool card), and the outcome union decodes both arms: cancelled →
// neutral Cancelled, selected → neutral Selected{OptionID}. An outcome
// carrying neither arm is an SDK contract violation — it fails loud, never
// silently reads as cancelled (a synthesized cancel would answer the
// backend's pending ask with -32021 the client never sent).
func newPermissionSender(conn *acp.AgentSideConnection) PermissionSender {
	return func(ctx context.Context, ask PermissionAsk) (PermissionOutcome, error) {
		options := make([]acp.PermissionOption, 0, len(ask.Options))
		for _, opt := range ask.Options {
			options = append(options, acp.PermissionOption{
				Kind:     acp.PermissionOptionKind(opt.Kind),
				Name:     opt.Name,
				OptionId: acp.PermissionOptionId(opt.OptionID),
			})
		}

		resp, err := conn.RequestPermission(ctx, acp.RequestPermissionRequest{
			SessionId: acp.SessionId(ask.SessionID),
			Options:   options,
			ToolCall:  acp.ToolCallUpdate{ToolCallId: acp.ToolCallId(ask.ToolCallID)},
		})
		if err != nil {
			return PermissionOutcome{}, err
		}

		switch {
		case resp.Outcome.Cancelled != nil:
			return PermissionOutcome{Cancelled: true}, nil
		case resp.Outcome.Selected != nil:
			return PermissionOutcome{OptionID: string(resp.Outcome.Selected.OptionId)}, nil
		default:
			return PermissionOutcome{}, NewInternal(
				"acpport: session/request_permission outcome carried neither cancelled nor selected arm")
		}
	}
}

// Initialize translates the neutral InitReport onto the honest D-02 wire
// response; front errors are coded via wireError, never passed raw (the
// SDK's toReqErr type-asserts *RequestError — an untranslated seam error
// would degrade to a generic -32603).
func (a *agentAdapter) Initialize(ctx context.Context, params acp.InitializeRequest) (acp.InitializeResponse, error) {
	report, err := a.h.Initialize(ctx)
	if err != nil {
		return acp.InitializeResponse{}, wireError(err)
	}

	return initializeResponse(report), nil
}

// initializeResponse assembles the honest wire report from the neutral
// InitReport: protocol version 1, real agentInfo, the advertised auth
// methods as agent-variant entries, the capabilities block with the
// sessionCapabilities.options empty-struct opt-in (MODEL-01, plan 06-04 —
// UNCONDITIONAL per D-11: the config-options picker is auth-path-
// independent; clients gate session/set_config_option on it), and zero
// fs/terminal surface (the SDK capability structs carry no such fields at
// all — honesty by construction, D-02/A7; omission is spec-correct).
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
		AgentCapabilities: acp.AgentCapabilities{
			SessionCapabilities: acp.SessionCapabilities{
				Options: &acp.SessionOptionsCapabilities{},
			},
		},
	}
}

// NewSession delegates to Handlers and wraps the front's session id plus
// its confirmed config options in the wire response (D-03 contract lives
// in the front; the options translate through the SDK select variant
// exclusively).
func (a *agentAdapter) NewSession(ctx context.Context, params acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	sessionID, options, err := a.h.NewSession(ctx, newSessionParams(params))
	if err != nil {
		return acp.NewSessionResponse{}, wireError(err)
	}

	a.emitCommandCatalog(ctx, sessionID)

	return acp.NewSessionResponse{
		SessionId:     acp.SessionId(sessionID),
		ConfigOptions: configOptionSelects(options),
	}, nil
}

// CatalogProvider is the Handlers-side opt-in for the slash-command
// catalog (v0.21.0, CAT-01): the adapter asks for the session's items
// after session/new and emits ONE available_commands_update through the
// connection-bound sink. Optional — Handlers impls without it simply
// never emit a catalog (the client's composer stays empty, the session
// unaffected).
type CatalogProvider interface {
	CommandCatalog(ctx context.Context, sessionID string) []CommandItem
}

// configOptionSelects assembles the SDK select variants from the neutral
// options vocabulary (MODEL-01, plan 06-04): id/name/category/currentValue
// translate verbatim, entries ride as the SDK's ungrouped select-options
// array, and the "type":"select" discriminant is INJECTED BY THE SDK's
// union MarshalJSON — nothing here hand-marshals wire bytes (T-06-12),
// and the assembled unions satisfy the SDK's own required-field Validate.
// A nil/empty list renders no configOptions at all (the D-10 honest
// smaller-list posture).
func configOptionSelects(defs []ConfigOptionDef) []acp.SessionConfigOption {
	if len(defs) == 0 {
		return nil
	}

	out := make([]acp.SessionConfigOption, 0, len(defs))

	for _, def := range defs {
		opts := make(acp.SessionConfigSelectOptionsUngrouped, 0, len(def.Options))

		for _, o := range def.Options {
			entry := acp.SessionConfigSelectOption{
				Value: acp.SessionConfigValueId(o.Value),
				Name:  o.Name,
			}
			if o.Description != "" {
				description := o.Description
				entry.Description = &description
			}

			opts = append(opts, entry)
		}

		category := acp.SessionConfigOptionCategory(def.Category)

		out = append(out, acp.SessionConfigOption{
			Select: &acp.SessionConfigOptionSelect{
				Id:           acp.SessionConfigId(def.ID),
				Name:         def.Name,
				Category:     &category,
				CurrentValue: acp.SessionConfigValueId(def.CurrentValue),
				Options:      acp.SessionConfigSelectOptions{Ungrouped: &opts},
				Type:         "select",
			},
		})
	}

	return out
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
// blocks verbatim, resource_link blocks as their URI (A3), image/audio/
// embedded-resource blocks skipped (this build advertises the text
// baseline only). The drop site warns STRUCTURALLY (WR-06, plan 05-04):
// kinds and counts only — never block content.
func joinContentBlocks(blocks []acp.ContentBlock) string {
	parts := make([]string, 0, len(blocks))
	held := make(map[string]int)

	for _, block := range blocks {
		switch {
		case block.Text != nil:
			parts = append(parts, block.Text.Text)
		case block.ResourceLink != nil:
			parts = append(parts, block.ResourceLink.Uri)
		case block.Image != nil:
			held["image"]++
		case block.Audio != nil:
			held["audio"]++
		case block.Resource != nil:
			held["resource"]++
		default:
			held["unknown"]++
		}
	}

	if len(held) > 0 {
		slog.Warn("acpport: unsupported prompt content blocks held from the turn text",
			"kinds", heldSummary(held))
	}

	return strings.Join(parts, " ")
}

// heldSummary renders the held-block counts as sorted kind=count pairs —
// the WR-06 warn carries kinds and counts only.
func heldSummary(held map[string]int) string {
	keys := make([]string, 0, len(held))
	for kind := range held {
		keys = append(keys, kind)
	}

	slices.Sort(keys)

	pairs := make([]string, 0, len(keys))
	for _, kind := range keys {
		pairs = append(pairs, kind+"="+strconv.Itoa(held[kind]))
	}

	return strings.Join(pairs, ",")
}

// Cancel forwards the session id to Handlers (D-05 — the front owns the
// exactly-once cancel machinery).
func (a *agentAdapter) Cancel(ctx context.Context, params acp.CancelNotification) error {
	return wireError(a.h.Cancel(ctx, string(params.SessionId)))
}

// Authenticate delegates the advertised auth method through the neutral
// Handlers surface (AUTH-01, plan 05-07 — the Phase-4 not-available stub is
// replaced by the front's honest chain: record key → push snapshot →
// refresh registry → read-back → {}). Success answers the empty result;
// every front error is coded via wireError — never a fake success, never
// the misleading -32601 default (D-02 honesty). The client-supplied method
// id passes through verbatim; its validation against the advertised set
// lives in the front.
func (a *agentAdapter) Authenticate(
	ctx context.Context,
	params acp.AuthenticateRequest,
) (acp.AuthenticateResponse, error) {
	if err := a.h.Authenticate(ctx, string(params.MethodId)); err != nil {
		return acp.AuthenticateResponse{}, wireError(err)
	}

	if refresher, ok := a.h.(CatalogRefresher); ok {
		refresher.RequestCatalogRefresh(ctx)
	}

	return acp.AuthenticateResponse{}, nil
}

// Logout is unadvertised in Phase 4 — clean -32601 default (Pitfall 6).
func (a *agentAdapter) Logout(ctx context.Context, params acp.LogoutRequest) (acp.LogoutResponse, error) {
	return acp.LogoutResponse{}, unadvertised(acp.AgentMethodLogout)
}

// CloseSession delegates the teardown to Handlers (SESS-01, plan 05-05)
// and answers the empty result — the ordered cancel-close-release contract
// lives in the front. Front errors are coded via wireError, never passed
// raw.
func (a *agentAdapter) CloseSession(
	ctx context.Context,
	params acp.CloseSessionRequest,
) (acp.CloseSessionResponse, error) {
	if err := a.h.CloseSession(ctx, string(params.SessionId)); err != nil {
		return acp.CloseSessionResponse{}, wireError(err)
	}

	return acp.CloseSessionResponse{}, nil
}

// DeleteSession delegates to Handlers (SESS-02, plan 07-03 — the -32601
// unadvertised default replaced, the CloseSession delegation shape): a
// registered id runs the front's full SESS-01 teardown and every unknown /
// already-closed id answers the spec's silent-success {} without touching
// the back leg (the front owns both arms — D-01). Success answers the
// empty result; front errors code via wireError, never raw. The method
// stays UNADVERTISED (the plan Task-3 honesty lock): the graceful arm
// serves spec-loose clients only.
func (a *agentAdapter) DeleteSession(
	ctx context.Context,
	params acp.DeleteSessionRequest,
) (acp.DeleteSessionResponse, error) {
	if err := a.h.DeleteSession(ctx, string(params.SessionId)); err != nil {
		return acp.DeleteSessionResponse{}, wireError(err)
	}

	return acp.DeleteSessionResponse{}, nil
}

// UnstableForkSession is the EXPLICIT fork decline arm (SESS-02, plan
// 07-03, T-07-10): implementing the SDK's optional AgentUnstableForkSession
// interface answers session/fork with the SAME graceful decline the
// dispatcher's type-assert default produces — unadvertised(method) is
// acp.NewMethodNotFound, the exact -32601 "Method not found" +
// {"method":"session/fork"} frame — so the honest shape is self-documenting
// and testable without relying on the default path, and can never drift
// into a crash path or a faked success. Fork support is v2 scope; the
// capability stays UNADVERTISED (Orca gates the call client-side and must
// never start calling it — Pitfall 4).
func (a *agentAdapter) UnstableForkSession(
	ctx context.Context,
	params acp.UnstableForkSessionRequest,
) (acp.UnstableForkSessionResponse, error) {
	return acp.UnstableForkSessionResponse{}, unadvertised(acp.AgentMethodSessionFork)
}

// ListSessions is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) ListSessions(
	ctx context.Context,
	params acp.ListSessionsRequest,
) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, unadvertised(acp.AgentMethodSessionList)
}

// ResumeSession delegates the reattach to Handlers (SESS-01, plan 05-05).
// The SDK's resume response carries no session id — the reattached id IS
// the request's own — so the adapter delegates and discards the
// Handlers-returned id (D-08 IN-02: no non-empty check happens here; the
// front owns the validation — the sessionId must not be empty, the cwd must
// be absolute, and the reattached id derives from the reply via the front's
// createSessionID — with every refusal coded via wireError) and the
// response stays the SDK zero value. The request's Cwd rides through
// verbatim (plan 05-10 WR-02 — the SDK contract makes it MUST-be-absolute;
// the front validates and records it). Front errors are coded via
// wireError, never passed raw.
func (a *agentAdapter) ResumeSession(
	ctx context.Context,
	params acp.ResumeSessionRequest,
) (acp.ResumeSessionResponse, error) {
	if _, err := a.h.ResumeSession(ctx, string(params.SessionId), params.Cwd); err != nil {
		return acp.ResumeSessionResponse{}, wireError(err)
	}

	return acp.ResumeSessionResponse{}, nil
}

// SetSessionConfigOption delegates the advertised config-options mutation
// through the neutral Handlers surface (MODEL-02, plan 06-06 — the -32601
// unadvertised default replaced, the Authenticate delegation shape): the
// wire triple (sessionId/configId/value) rides verbatim, the full
// re-emitted list assembles through the SDK select variant exclusively, and
// every front error codes via wireError — never raw, never the misleading
// -32601 default (D-02 honesty). A boolean-variant ask translates to the
// empty value id: this build advertises select options only, so the front's
// membership validation answers the coded -32602 naming the option's own
// listed set.
func (a *agentAdapter) SetSessionConfigOption(
	ctx context.Context,
	params acp.SetSessionConfigOptionRequest,
) (acp.SetSessionConfigOptionResponse, error) {
	var (
		sessionID string
		configID  string
		value     string
	)

	switch {
	case params.ValueId != nil:
		sessionID = string(params.ValueId.SessionId)
		configID = string(params.ValueId.ConfigId)
		value = string(params.ValueId.Value)
	case params.Boolean != nil:
		sessionID = string(params.Boolean.SessionId)
		configID = string(params.Boolean.ConfigId)
	default:
		return acp.SetSessionConfigOptionResponse{}, acp.NewRequestError(
			CodeInvalidParams, "session/set_config_option: params carry neither a value id nor a boolean variant", nil)
	}

	options, err := a.h.SetSessionConfigOption(ctx, sessionID, configID, value)
	if err != nil {
		return acp.SetSessionConfigOptionResponse{}, wireError(err)
	}

	selects := configOptionSelects(options)
	if selects == nil {
		selects = []acp.SessionConfigOption{} // the spec MUST + the SDK Validate: the list is required
	}

	return acp.SetSessionConfigOptionResponse{ConfigOptions: selects}, nil
}

// SetSessionMode is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) SetSessionMode(
	ctx context.Context,
	params acp.SetSessionModeRequest,
) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, unadvertised(acp.AgentMethodSessionSetMode)
}

// emitCommandCatalog publishes the session's command catalog as ONE
// available_commands_update (the Orca composer's discovery scan waits ≤9 s
// from connect). The catalog is a nicety — an emission failure (or a
// missing binding) logs structurally and NEVER fails session/new.
func (a *agentAdapter) emitCommandCatalog(ctx context.Context, sessionID string) {
	provider, ok := a.h.(CatalogProvider)
	if !ok {
		return
	}

	sink := a.sink.Load()
	if sink == nil {
		return
	}

	note := SessionNote{SessionID: sessionID, Kind: NoteCatalog, Catalog: provider.CommandCatalog(ctx, sessionID)}
	if err := (*sink)(ctx, note); err != nil {
		slog.Warn("acpport: command catalog emission failed", "session", sessionID, "err", err)
	}
}

// bindSink attaches the connection-bound update sink (WR-02: the publish
// is an atomic store); called once by NewAgentSideConnection during
// construction.
func (a *agentAdapter) bindSink(sink UpdateSink) {
	a.sink.Store(&sink)
}

// bindUpdateSink forwards the same connection-bound sink to the Handlers
// implementation for front-originated asynchronous notifications.
func (a *agentAdapter) bindUpdateSink(sink UpdateSink) {
	if binder, ok := a.h.(UpdateSinkBinder); ok {
		binder.BindUpdateSink(sink)
	}
}

// bindPermissionSender forwards the connection-bound sender into the
// Handlers impl when it opts in via the exported PermissionBinder hook
// (the front's bridge); called once by NewAgentSideConnection during
// construction.
func (a *agentAdapter) bindPermissionSender(sender PermissionSender) {
	if binder, ok := a.h.(PermissionBinder); ok {
		binder.BindPermissionSender(sender)
	}
}

// sinkForPrompt returns the bound connection sink (WR-02: an atomic load —
// always the complete func value or nil, never torn); the construction
// discipline makes the bound path the only real path, but a direct
// NewAgent misuse gets a loud error instead of a nil-deref panic.
func (a *agentAdapter) sinkForPrompt() UpdateSink {
	if bound := a.sink.Load(); bound != nil {
		return *bound
	}

	return func(ctx context.Context, note SessionNote) error {
		return NewInternal("acpport: no connection bound — construct via NewAgentSideConnection")
	}
}

// wireError maps a front error onto the SDK's coded RequestError: seam
// errors carry their own code; anything else is an internal error (A5).
// A seam error carrying a non-empty Details widens the frame with
// data:{"details": ...} (NewInternalWithData — the omp dialect's string
// contracts, plan 07-05), and one carrying a non-empty Reason widens it
// with data:{"reason": ...} (NewSessionBusy — the concurrent-prompt refusal
// Orca's retryBusyPrompts gate retries); a seam error with neither keeps
// data null — the payload is opt-in per error. The returned *RequestError
// must stay unwrapped — the SDK's toReqErr type-asserts the concrete type
// (no errors.As), so %w wrapping would silently degrade every coded seam
// error to a generic -32603.
func wireError(err error) error {
	if err == nil {
		return nil
	}

	var seamErr *Error
	if errors.As(err, &seamErr) {
		var data any
		if seamErr.Details != "" || seamErr.Reason != "" {
			payload := make(map[string]string)
			if seamErr.Details != "" {
				payload["details"] = seamErr.Details
			}

			if seamErr.Reason != "" {
				payload["reason"] = seamErr.Reason
			}

			data = payload
		}

		return acp.NewRequestError(seamErr.Code, seamErr.Message, data)
	}

	return acp.NewInternalError(err.Error())
}

// unadvertised is the -32601 default for the stable methods this build
// does not advertise (Pitfall 6): capability omission stops legitimate
// clients, but the dispatcher still routes here — answer cleanly, naming
// the method.
func unadvertised(method string) error {
	return acp.NewMethodNotFound(method)
}
