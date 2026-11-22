package acpport

import (
	"context"
	"encoding/json"
	"fmt"
)

// Handlers is the neutral ACP-agent vocabulary (RESEARCH Pattern 1, the
// D-01 seam): the agent policy layer (internal/acpfront) implements this
// interface and never imports the SDK; agent.go translates it onto the
// SDK's generated acp.Agent surface, so a v0.x SDK break is a
// one-package migration.
//
// Method coverage is the v1 minimal chat loop plus the SESS-01 lifecycle
// pair: Initialize assembles the honest D-02 capability report,
// NewSession/Prompt/Cancel carry the D-03/D-04/D-05 contracts, Resume/
// Close carry SESS-01 (plan 05-05 — the adapter's unadvertised -32601
// stubs became real delegates), and every remaining unadvertised stable
// method is answered with the -32601 default inside the adapter (Pitfall
// 6) — the Handlers vocabulary never grows for methods this build does
// not offer.
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

	// ResumeSession reattaches one backend session and returns its id
	// (SESS-01, plan 05-05): the backend reply is the create-shaped full
	// snapshot, so the id the front returns digs out through the same
	// path session/new uses. No setModel follow-up, no replay of
	// pre-resume history (D-04) — the contracts live in the front. The
	// cwd rides the SDK ResumeSessionRequest.Cwd and MUST be absolute
	// (the session-setup MUST — the front validates it before any
	// backend call, the T-4-01 parity guard, and records it so the
	// resumed session's tool diff/location paths resolve absolute;
	// plan 05-10 WR-02).
	ResumeSession(ctx context.Context, sessionID string, cwd string) (string, error)

	// CloseSession runs the session teardown (SESS-01, plan 05-05):
	// cancel-then-close-then-release, best-effort on the backend leg —
	// the ordered contract lives in the front.
	CloseSession(ctx context.Context, sessionID string) error

	// Authenticate runs one advertised auth method (AUTH-01, plan 05-07):
	// the front validates the method id against the advertised set and
	// runs the honest chain (record key → push snapshot → refresh registry
	// → registry read-back) — the read-back verdict is the only success.
	// No credentials ride the wire (the key enters via the wrapper's own
	// config); every non-success answers a coded error.
	Authenticate(ctx context.Context, methodID string) error
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

// Neutral config-option categories (MODEL-01, plan 06-04): the ACP wire
// strings the adapter maps onto the SDK's SessionConfigOptionCategory —
// the same wire-string vocabulary discipline as ToolKind*/PermKind*.
const (
	ConfigCategoryModel        = "model"
	ConfigCategoryThoughtLevel = "thought_level"
)

// SelectOptionDef is one selectable value of a config-option select list
// (MODEL-01, D-08 verbatim mirror): the backend's label and description
// ride exactly as reported — the wrapper never filters, renames, or
// enriches the list (a disabledReason-carrying entry stays present, as
// reported).
type SelectOptionDef struct {
	Value       string
	Name        string
	Description string
}

// ConfigOptionDef is one neutral session config option (MODEL-01): a
// select list the client renders as the model / thought-level picker.
// CurrentValue is SERVER-CONFIRMED ONLY (D-10) — when the backend reports
// no confirmed current the whole option is omitted from the list (locked
// OQ2: a smaller honest list beats a fabricated selection). The adapter
// alone assembles the SDK select variant from this vocabulary; the neutral
// shape never carries SDK types across the seam (D-01).
type ConfigOptionDef struct {
	ID           string
	Name         string
	Category     string
	CurrentValue string
	Options      []SelectOptionDef
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

// The backend-sourced streaming kinds: message text, reasoning text, and
// the tool-call lifecycle notes (TOOL-01).
const (
	NoteMessage SessionNoteKind = "message"
	NoteThought SessionNoteKind = "thought"
	NoteTool    SessionNoteKind = "tool"
)

// SessionNote is one streaming content delta (D-04): one backend
// part.delta envelope maps to exactly one note — nothing here is ever
// synthesized (every chunk the client sees is traceable to one received
// backend event). Tool lifecycle envelopes map onto Kind NoteTool with a
// non-nil Tool payload (TOOL-01).
type SessionNote struct {
	SessionID string
	Kind      SessionNoteKind
	Delta     string

	// Tool carries the neutral tool-call vocabulary when Kind is NoteTool.
	Tool *ToolNote
}

// ACP tool-call kinds (05-RESEARCH.md Pattern 4 mapping table, the SDK's
// ToolKind vocabulary): the neutral vocabulary carries the wire strings —
// SDK types never cross the seam. Unknown backend tool names decode as
// ToolKindOther at the mapping site (the tap), never here.
const (
	ToolKindRead    = "read"
	ToolKindEdit    = "edit"
	ToolKindDelete  = "delete"
	ToolKindMove    = "move"
	ToolKindSearch  = "search"
	ToolKindExecute = "execute"
	ToolKindThink   = "think"
	ToolKindFetch   = "fetch"
	ToolKindOther   = "other"
)

// ACP tool-call statuses (05-RESEARCH.md Pattern 4): the lifecycle
// transitions the tool notes carry — pending→in_progress→completed/failed.
const (
	ToolStatusPending    = "pending"
	ToolStatusInProgress = "in_progress"
	ToolStatusCompleted  = "completed"
	ToolStatusFailed     = "failed"
)

// ToolDiff is one rendered file diff (TOOL-01, D-03): the path is the
// session-cwd-resolved ABSOLUTE path (the tap resolves before emitting —
// Pitfall 5); an empty OldText omits oldText on the wire (the new-file
// Write shape per the ACP tool-calls spec).
type ToolDiff struct {
	Path    string
	OldText string
	NewText string
}

// ToolLocation is one affected file location (the ACP tool-calls
// locations follow-along surface); the path is absolute like ToolDiff.
type ToolLocation struct {
	Path string
}

// ToolNote is one neutral tool-call lifecycle note (TOOL-01, the
// prescription vocabulary): one backend tool.updated envelope maps to
// exactly one note — nothing synthesized (the TOOL-01 honesty carry-over
// of D-04). IsStart selects the wire shape: a start note renders the
// tool_call frame (StartToolCall arms), an update note renders the
// tool_call_update patch (only the populated fields ride — patch
// semantics per spec).
type ToolNote struct {
	ToolCallID string
	IsStart    bool
	Title      string
	Name       string
	Kind       string // one of the ToolKind* wire strings
	Status     string // one of the ToolStatus* wire strings
	Diff       *ToolDiff
	Locations  []ToolLocation
	RawInput   json.RawMessage
	RawOutput  json.RawMessage
}

// UpdateSink streams one SessionNote to the ACP client; the adapter hands
// a connection-bound sink into Handlers.Prompt.
type UpdateSink func(ctx context.Context, note SessionNote) error

// ACP permission-option kinds the backend bridge's mapping produces
// (05-RESEARCH.md Pattern 3 table): the neutral permission vocabulary
// carries whatever kind string the ask builder supplied — the sender
// passes it verbatim onto the SDK — and these constants name the mapped
// ACP values (the backend's deny kind surfaces as the ACP reject_once UI
// hint; kind is a UI hint only per the ACP v1 spec).
const (
	PermKindAllowOnce   = "allow_once"
	PermKindAllowAlways = "allow_always"
	PermKindRejectOnce  = "reject_once"
)

// PermissionOptionDef is one option of a backend permission ask, verbatim
// from the interaction/requestPermission params (TOOL-02): the backend's
// kind string, the human-readable name, and the option id the client's
// selected outcome echoes back. Nothing here is ever rewritten by the
// adapter — the mapping (allow_project→allow_always, deny→reject_once)
// happens ask-side, before the neutral vocabulary is built.
type PermissionOptionDef struct {
	Kind     string
	Name     string
	OptionID string
}

// PermissionAsk is one normalized backend permission ask on the neutral
// vocabulary: the ACP session the ask belongs to, the backend's
// correlation id (params.requestId — the front bridge's pending-table
// key), the tool call the ask renders on, and the verbatim options.
type PermissionAsk struct {
	SessionID  string
	RequestID  string
	ToolCallID string
	ToolName   string
	Options    []PermissionOptionDef
}

// PermissionOutcome is one decoded ACP client decision (the SDK's outcome
// union decoded into the neutral vocabulary): either the prompt turn was
// cancelled while the ask was pending — the client MUST answer cancelled
// on turn cancel (ACP v1 tool-calls cancellation) — or the user selected
// one of the ask's options, by its verbatim OptionID.
type PermissionOutcome struct {
	Cancelled bool
	OptionID  string
}

// PermissionSender forwards one permission ask to the ACP client as
// session/request_permission and returns the client's decoded outcome;
// the adapter binds it at construction (sinkBinder discipline — before
// any frame flows) and the front bridge calls it per ask.
type PermissionSender func(ctx context.Context, ask PermissionAsk) (PermissionOutcome, error)

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
	CodeInvalidParams  = -32602
	CodeInternal       = -32603
)

// Error is a seam-carried front error with its wire code; the adapter
// translates it onto the SDK's RequestError so the client sees the coded
// JSON-RPC error.
type Error struct {
	Code    int
	Message string
}

// NewInvalidRequest builds a -32600 front error (protocol-level request
// violation — e.g. the D-05 second-prompt guard until 04-05 sharpens the
// concurrency mapping).
func NewInvalidRequest(format string, args ...any) *Error {
	return &Error{Code: CodeInvalidRequest, Message: fmt.Sprintf(format, args...)}
}

// NewInvalidParams builds a -32602 front error (the request's params are
// malformed — e.g. the session/new relative-cwd rejection, T-4-01: the
// spec's session-setup contract makes cwd MUST-be-absolute).
func NewInvalidParams(format string, args ...any) *Error {
	return &Error{Code: CodeInvalidParams, Message: fmt.Sprintf(format, args...)}
}

// NewInternal builds a -32603 front error (backend failure surfaced
// honestly — A5).
func NewInternal(format string, args ...any) *Error {
	return &Error{Code: CodeInternal, Message: fmt.Sprintf(format, args...)}
}

// Error implements the error interface with the plain message.
func (e *Error) Error() string {
	return e.Message
}
