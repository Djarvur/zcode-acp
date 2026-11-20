// Package interact implements the D-01 interaction responder policy object
// (CORE-06): every server→host request reaching a conn wired with the
// Responder is answered exactly once — known methods per their per-method
// policy (deadline, handler), unknown methods with the -32601 protocol
// error naming the method (never a crash, never a hang), re-announced
// requests deduped on (method, params.requestId) — and every reply rides
// the conn's bypass lane (the conn's Reply/ReplyError path; the responder
// returns values and wire errors to the conn, it never writes envelopes
// itself).
//
// Production wiring assigns Responder.Serve to conn.OnRequest before any
// frame flows (the intercept seam, conn.go:91-101); Serve answers for
// everything it routes, so the conn's built-in router stays the unwired
// fallback (D-01: Phase-3 test locks untouched).
package interact

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Interaction method-name constants: verbatim wire strings from the
// dialect §3 server→host surface.
const (
	MethodRequestPermission             = "interaction/requestPermission"
	MethodRequestUserInput              = "interaction/requestUserInput"
	MethodRequestProviderRuntimeHeaders = "interaction/requestProviderRuntimeHeaders"
	MethodRequestOfficialMcpAuthHeaders = "interaction/requestOfficialMcpAuthHeaders"
	MethodBrowserList                   = "interaction/browserList"
	MethodBrowserExecute                = "interaction/browserExecute"
)

// Per-method deadline defaults (05-RESEARCH.md Pattern 1 policy table).
const (
	// DefaultPermissionDeadline is the OQ-2 disposition: the host-side
	// requestPermission deadline default — settable via WithMethodDeadline
	// — whose expiry answers deny (the turn continues honestly).
	DefaultPermissionDeadline = 300 * time.Second
)

// Answer is one handler's reply payload: Result marshals into the reply
// envelope, Err (non-nil) turns it into an error reply. Consumers match
// the numeric code, never the message text (dialect §6).
type Answer struct {
	Result any
	Err    *zcode.WireError
}

// Request is the normalized reverse request; ReqID/SessionID are dug out
// of params once, at serve time (the wire keys are camelCase — the
// machine.go paramsString dig-out discipline).
type Request struct {
	Method    string
	ID        *zcode.ID // the wire id the reply must echo
	Params    json.RawMessage
	ReqID     string // params.requestId, "" when absent
	SessionID string // params.sessionId, "" when absent
}

// Handler answers one normalized reverse request under its policy
// deadline. A returned error (with no Answer.Err) wraps as the -32603
// internal code.
type Handler func(ctx context.Context, req Request) (Answer, error)

// MethodPolicy is one method's row in the policy table: the answer
// deadline (0 = no deadline) and its handler.
type MethodPolicy struct {
	Deadline time.Duration
	Handler  Handler
}

// PermissionForwarder forwards an interaction/requestPermission ask to the
// ACP client (the TOOL-02 bridge — plan 05-02 implements it). A nil
// forwarder answers a coded not-wired error.
type PermissionForwarder func(ctx context.Context, req Request) (Answer, error)

// RuntimeHeadersKeySource supplies the api-key material for the
// requestProviderRuntimeHeaders success arm (plan 05-06 injects the real
// source). A nil source answers a coded error naming the unconfigured key
// state — never the failure-shaped success arm (PITFALL 3).
type RuntimeHeadersKeySource func(ctx context.Context, req Request) (apiKey, apiKeyID string, err error)

// Option configures a Responder at construction.
type Option func(*Responder)

// WithMethodDeadline overrides a policy-table method's deadline (0 = no
// deadline) — the injectable knob tests shorten to exercise expiry;
// requestPermission's 300 s default is settable per the OQ-2 disposition.
func WithMethodDeadline(_ string, _ time.Duration) Option {
	return func(*Responder) {}
}

// WithPermissionForwarder injects the PermissionForwarder seam.
func WithPermissionForwarder(_ PermissionForwarder) Option {
	return func(*Responder) {}
}

// WithRuntimeHeadersKeySource injects the RuntimeHeadersKeySource seam.
func WithRuntimeHeadersKeySource(_ RuntimeHeadersKeySource) Option {
	return func(*Responder) {}
}

// WithRouterEvent installs the RouterEvent hook (deadline expiries are
// recorded through it, mirroring the conn's record-honesty contract).
func WithRouterEvent(_ func(zcode.RouterEvent)) Option {
	return func(*Responder) {}
}

// Responder is the D-01 policy object: one handler per server→host method,
// per-method deadlines, and the re-announce dedupe table. Constructed once
// via NewResponder; Serve is safe for concurrent use.
type Responder struct{}

// NewResponder builds a Responder with the default D-01 policy table and
// applies the options.
func NewResponder(_ ...Option) *Responder {
	return &Responder{}
}

// PendingRequests reports the dedupe table's pending entry count — the
// bounded map's growth ceiling is one entry per distinct (method,
// params.requestId), evicted on answer and per-entry TTL (T-5-01).
func (r *Responder) PendingRequests() int {
	return 0
}

// Serve answers one server→host request envelope (the conn.OnRequest
// signature): (result, wire error, handled).
func (r *Responder) Serve(_ context.Context, _ zcode.Envelope) (any, *zcode.WireError, bool) {
	// RED skeleton: zero values so the contract test compiles and fails on
	// its first unmet assertion (never a build failure) — handled=false
	// falls through to the conn's built-in router.
	return nil, nil, false
}
