package acpport

import (
	"context"
	"fmt"
)

// Handlers is the neutral ACP-agent vocabulary (RESEARCH Pattern 1, the
// D-01 seam): the agent policy layer (internal/acpfront) implements this
// interface and never imports the SDK; agent.go translates it onto the
// SDK's generated acp.Agent surface, so a v0.x SDK break is a
// one-package migration.
//
// Method coverage is the v1 minimal chat loop: Initialize assembles the
// honest D-02 capability report, NewSession/Prompt/Cancel carry the
// D-03/D-04/D-05 contracts, and every unadvertised stable method is
// answered with the -32601 default inside the adapter (Pitfall 6) — the
// Handlers vocabulary never grows for methods this build does not offer.
type Handlers interface {
	// Initialize returns the honest agent identity and the auth methods
	// this build advertises (D-02); the adapter assembles the wire
	// response.
	Initialize(ctx context.Context) (InitReport, error)

	// NewSession opens one backend session and returns its ACP session id
	// (the D-03 create/subscribe/setModel contract lives in the front).
	NewSession(ctx context.Context, params NewSessionParams) (string, error)

	// Prompt runs one prompt turn: content deltas stream through sink as
	// they arrive (D-04), and the returned StopReason is the turn's
	// terminal outcome. Front errors return as Go errors — the adapter
	// codes them; a failure is never a synthetic stop reason (A5).
	Prompt(ctx context.Context, params PromptParams, sink UpdateSink) (StopReason, error)

	// Cancel cancels the session's active turn exactly once (D-05).
	Cancel(ctx context.Context, sessionID string) error
}

// InitReport is the honest initialize payload (D-02): agent identity plus
// the auth methods this build advertises. Wire capabilities are assembled
// by the adapter as the SDK zero value — there is nothing to configure.
type InitReport struct {
	Name        string
	Title       string
	Version     string
	AuthMethods []AuthMethodDef
}

// AuthMethodDef is one advertised authentication method; an empty
// Description omits the optional wire field.
type AuthMethodDef struct {
	ID          string
	Name        string
	Description string
}

// NewSessionParams carries the decoded session/open request.
type NewSessionParams struct {
	// Cwd is the working directory the client requested (spec
	// MUST-be-absolute; the front validates before the backend call).
	Cwd string

	// McpServers is the client's MCP list — accepted and ignored in
	// Phase 4 (D-03) — shallow-copied into the neutral shape so SDK types
	// never cross the seam.
	McpServers []McpServerDef
}

// McpServerDef is the neutral shallow view of one client-declared MCP
// server (name plus transport kind). Phase 4 ignores the list; later
// phases may widen the vocabulary.
type McpServerDef struct {
	Name string
	Kind string // "stdio" | "http" | "sse" | "acp"
}

// PromptParams is the decoded prompt turn request: the session id and the
// prompt content joined into one string (text blocks verbatim;
// resource_link blocks contribute their URI — Assumption A3).
type PromptParams struct {
	SessionID string
	Text      string
}

// SessionNoteKind discriminates the streaming update kinds Phase 4 emits
// (D-04).
type SessionNoteKind string

// The two backend-sourced delta kinds: message text and reasoning text.
const (
	NoteMessage SessionNoteKind = "message"
	NoteThought SessionNoteKind = "thought"
)

// SessionNote is one streaming content delta (D-04): one backend
// part.delta envelope maps to exactly one note — nothing here is ever
// synthesized (every chunk the client sees is traceable to one received
// backend event).
type SessionNote struct {
	SessionID string
	Kind      SessionNoteKind
	Delta     string
}

// UpdateSink streams one SessionNote to the ACP client; the adapter hands
// a connection-bound sink into Handlers.Prompt.
type UpdateSink func(ctx context.Context, note SessionNote) error

// StopReason is the neutral terminal outcome of one prompt turn; the
// values are the ACP stop-reason strings the adapter maps onto the SDK
// constants. The minimal loop has exactly two: a completed turn and a
// cancelled one (D-05) — every other terminal state is an error (A5).
type StopReason string

// The two mapped stop reasons.
const (
	StopEndTurn   StopReason = "end_turn"
	StopCancelled StopReason = "cancelled"
)

// JSON-RPC error codes the error helpers map front errors onto. Plan
// 04-05 sharpens the concurrency mapping (-32010) on top of these.
const (
	CodeInvalidRequest = -32600
	CodeInternal       = -32603
)

// Error is a seam-carried front error with its wire code; the adapter
// translates it onto the SDK's RequestError so the client sees the coded
// JSON-RPC error.
type Error struct {
	Code    int
	Message string
}

// Error implements the error interface with the plain message.
func (e *Error) Error() string {
	return e.Message
}

// NewInvalidRequest builds a -32600 front error (protocol-level request
// violation — e.g. the D-05 second-prompt guard until 04-05 sharpens the
// concurrency mapping).
func NewInvalidRequest(format string, args ...any) *Error {
	return &Error{Code: CodeInvalidRequest, Message: fmt.Sprintf(format, args...)}
}

// NewInternal builds a -32603 front error (backend failure surfaced
// honestly — A5).
func NewInternal(format string, args ...any) *Error {
	return &Error{Code: CodeInternal, Message: fmt.Sprintf(format, args...)}
}
