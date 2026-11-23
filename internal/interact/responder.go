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
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
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

// wireKeyReason is the camelCase reason wire key shared by the reply
// payloads that carry one (the permission deny answer, the officialMcp
// decline) — the goconst single-source for the repeated literal.
const wireKeyReason = "reason"

// Per-method deadline defaults (05-RESEARCH.md Pattern 1 policy table).
const (
	// DefaultPermissionDeadline is the OQ-2 disposition: the host-side
	// requestPermission deadline default — settable via WithMethodDeadline
	// — whose expiry answers deny (the turn continues honestly).
	DefaultPermissionDeadline = 300 * time.Second

	// DefaultRuntimeHeadersDeadline is the D-01 requestProviderRuntimeHeaders
	// deadline — the server holds the same 180 s window (dialect §3).
	DefaultRuntimeHeadersDeadline = 180 * time.Second

	// DefaultUserInputDeadline is the host policy for requestUserInput.
	DefaultUserInputDeadline = 15 * time.Second
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

// RuntimeHeadersKeySource supplies the credential material for the
// requestProviderRuntimeHeaders success arm (plan 05-06 injected the api-key
// source; plan 06-07 widens the signature with accountScope so the
// coding-plan source can hand over all three fields — Pattern 4/A9). A nil
// source answers a coded error naming the unconfigured key state — never the
// failure-shaped success arm (PITFALL 3). The scope is optional: an empty
// accountScope keeps the api-key two-field answer byte-compatibly identical.
type RuntimeHeadersKeySource func(ctx context.Context, req Request) (apiKey, apiKeyID, accountScope string, err error)

// Option configures a Responder at construction.
type Option func(*Responder)

// WithMethodDeadline overrides a policy-table method's deadline (0 = no
// deadline) — the injectable knob tests shorten to exercise expiry;
// requestPermission's 300 s default is settable per the OQ-2 disposition.
// Unknown methods (no policy entry) keep the -32601 floor regardless.
func WithMethodDeadline(method string, d time.Duration) Option {
	return func(r *Responder) {
		if pol, ok := r.methods[method]; ok {
			pol.Deadline = d
			r.methods[method] = pol
		}
	}
}

// WithUserInputAsker wires the ask-the-user bridge seam (v0.22.0): the
// requestUserInput policy row renders the decoded ask on this surface and
// maps the outcome into the answers merge content. Unset — the legacy
// honest decline stays (UIB-04).
func WithUserInputAsker(ask acpport.UserInputAsker) Option {
	return func(r *Responder) { r.userInputAsk = ask }
}

// WithPermissionForwarder wires the permission bridge seam (the TOOL-01
// translation: the backend's interaction/requestPermission forwards to the
// client's session/request_permission round-trip).
func WithPermissionForwarder(f PermissionForwarder) Option {
	return func(r *Responder) {
		r.permission = f
	}
}

// WithRuntimeHeadersKeySource injects the RuntimeHeadersKeySource seam
// (plan 05-06 injected the api-key source; plan 06-07 swaps in the
// mode-aware PAT source at the cmd wiring).
func WithRuntimeHeadersKeySource(src RuntimeHeadersKeySource) Option {
	return func(r *Responder) {
		r.keySource = src
	}
}

// WithRouterEvent installs the RouterEvent hook — deadline expiries are
// recorded through it, mirroring the conn's record-honesty contract
// (conn.go:436-450).
func WithRouterEvent(fn func(zcode.RouterEvent)) Option {
	return func(r *Responder) {
		r.onEvent = fn
	}
}

// WithCredentialDecrypt injects the D-06 decrypt seam (plan 05-07): the
// requestCredentialCipher ask gains a policy row dispatching decrypt ops to
// the CredentialDecrypt service (the field-verified {requestId,
// result:{ok...}} reply shapes). Unwired, the ask keeps the D-05
// tracked-no-answer fall-through — CI tiers never touch the keyring.
func WithCredentialDecrypt(svc *CredentialDecrypt) Option {
	return func(r *Responder) {
		r.cipher = svc
		r.methods[zcode.MethodRequestCredentialCipher] = MethodPolicy{
			Deadline: DefaultCredentialCipherDeadline,
			Handler:  svc.Handle,
		}
	}
}

// Responder is the D-01 policy object: one handler per server→host method,
// per-method deadlines, and the re-announce dedupe table. Constructed once
// via NewResponder and immutable afterwards; Serve is safe for concurrent
// use.
type Responder struct {
	methods    map[string]MethodPolicy
	dedupe     *dedupeTable
	permission PermissionForwarder
	keySource  RuntimeHeadersKeySource
	cipher     *CredentialDecrypt

	// userInputAsk is the ask-the-user bridge seam (v0.22.0, UIB-01):
	// nil — the legacy honest decline; wired — the ask renders on the
	// client surface and the outcome maps into the answers merge content.
	userInputAsk acpport.UserInputAsker
	onEvent      func(zcode.RouterEvent)
}

// NewResponder builds a Responder with the default D-01 policy table and
// applies the options (seams, deadline overrides, event hook).
func NewResponder(opts ...Option) *Responder {
	r := &Responder{dedupe: newDedupeTable()}
	r.methods = r.defaultPolicyTable()

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// PendingRequests reports the dedupe table's pending entry count — the
// bounded map's growth ceiling is one entry per distinct (method,
// params.requestId), evicted on answer and per-entry TTL (T-5-01).
func (r *Responder) PendingRequests() int {
	return r.dedupe.len()
}

// Serve answers one server→host request envelope (the conn.OnRequest
// signature): (result, wire error, handled). It returns handled=true for
// everything it routes — the unknown -32601 floor included — so the
// built-in router is never consulted behind it; the one exception is
// interaction/requestCredentialCipher while the D-06 decrypt seam is
// unwired, which falls through to the conn's tracked-no-answer tracker
// (D-05). With the seam injected the ask routes through its policy row
// like every other method.
func (r *Responder) Serve(ctx context.Context, env zcode.Envelope) (any, *zcode.WireError, bool) {
	req := normalize(env)

	// D-05 posture (Phase-3 locked): credential-cipher is tracked and
	// never answered while the decrypt seam is unwired — replying at all
	// (result, error, or bare id-echo) would claim an answer the wrapper
	// cannot give while the server holds the turn on it. The unwired
	// fall-through composes with the conn's base-context tracker; the D-06
	// injection (WithCredentialDecrypt) replaces it with the policy row.
	if req.Method == zcode.MethodRequestCredentialCipher && r.cipher == nil {
		return nil, nil, false
	}

	pol, known := r.methods[req.Method]

	if !known {
		// CORE-06 floor (conn.go:352-357 posture verbatim): -32601 naming
		// the method — never a crash, never a fabricated success.
		slog.Warn("interact: unsupported host request", "method", req.Method, "id", idKey(req))

		return nil, &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: "unsupported host request: " + req.Method,
		}, true
	}

	// Re-announce dedupe (05-RESEARCH.md Pattern 2): the first envelope of
	// a (method, params.requestId) set claims the entry; re-announces carry
	// NEW wire ids and the SAME requestId, so they join. Requests without a
	// requestId claim nothing — answered per envelope (empty-input probe).
	if req.ReqID != "" {
		if r.dedupe.claim(req.Method, req.ReqID, dedupeTTL(pol.Deadline)) {
			slog.Warn("interact: re-announce joined a pending entry",
				"method", req.Method, "requestId", req.ReqID, "id", idKey(req))

			// Park without framing anything until the conn closes: the
			// first reply resolves the whole pending set server-side
			// (cleanup deletes every wire id of the set), and a frame of
			// ANY shape to a still-registered id would reject the pending
			// ask — the server result-schema-parses reply payloads
			// (server.ts resolveClientRequest; a payload-less frame fails
			// parse → "Invalid response"). The silence is STRUCTURAL, not
			// a shutdown race (plan 05-10 WR-01): this arm returns
			// (nil, nil, true) and the conn's reply funnel drops a
			// neither-result-nor-error answer outright — a JSON-RPC
			// response carries exactly one of result/error — so the bare
			// id-echo cannot be framed even in the cancelBase→rw.Close
			// window. A parked goroutine adds no per-frame state the
			// conn has not already spent (the demux spawns one goroutine
			// per request frame regardless). The server silently drops
			// replies to already-cleaned ids — we simply never send them.
			<-ctx.Done()

			return nil, nil, true
		}

		defer r.dedupe.release(req.Method, req.ReqID)
	}

	result, werr := r.dispatch(ctx, pol, req)

	return result, werr, true
}

// defaultPolicyTable is the D-01 per-method policy table
// (05-RESEARCH.md Pattern 1): the migrated runtime-preferences static
// answer (byte-identical to conn.go:384-387 — the conn's built-in router
// keeps serving it only while the responder is unwired), the seam-backed
// permission and runtime-headers policies, the decline postures for
// requestUserInput and officialMcpAuthHeaders (the latter answered INSIDE
// the reply RESULT — the EXP-CODINGPLAN-CYCLE-01 attempt-1 unblock: a
// -32601 there stalls the plugin runtime materialization), and the -32601
// floor for the browser* pair (field-verified harmless —
// EXP-SUBSCRIBE-01: 6-7 asks per create answered -32601 without breaking
// the flow).
//
// interaction/requestCredentialCipher is deliberately ABSENT here: while
// the D-06 decrypt seam is unwired, Serve falls it through (handled=false)
// to the conn's built-in tracked-no-answer tracker — the Phase-3 locked
// D-05 posture (a reply of ANY shape would claim an answer the wrapper
// cannot give and the server holds the turn on it). WithCredentialDecrypt
// adds the policy row at injection time.
func (r *Responder) defaultPolicyTable() map[string]MethodPolicy {
	return map[string]MethodPolicy{
		zcode.MethodRequestRuntimePreferences: {
			Deadline: zcode.RouterDefaultDeadline,
			Handler:  answerRuntimePreferences,
		},
		MethodRequestPermission: {
			Deadline: DefaultPermissionDeadline,
			Handler:  r.handlePermission,
		},
		MethodRequestProviderRuntimeHeaders: {
			Deadline: DefaultRuntimeHeadersDeadline,
			Handler:  r.handleRuntimeHeaders,
		},
		MethodRequestUserInput: {
			// No policy deadline when the bridge is wired: a human answering
			// a dialog takes minutes (UIB-01); the legacy 15 s bound would
			// decline mid-thought. The nil-asker arm declines immediately
			// anyway, so the legacy posture loses nothing.
			Deadline: 0,
			Handler:  r.handleUserInput,
		},
		MethodRequestOfficialMcpAuthHeaders: {
			Handler: answerOfficialMcpAuthHeaders,
		},
		MethodBrowserList: {
			Handler: unsupportedHostMethod,
		},
		MethodBrowserExecute: {
			Handler: unsupportedHostMethod,
		},
	}
}

// dispatch runs the handler under the policy deadline (0 = none) and
// answers the method's expiry answer when the deadline elapses with the
// handler unanswered — the conn.go:365-395 deadline mechanics mirrored:
// errors.Is(ctx.Err(), context.DeadlineExceeded) distinguishes genuine
// expiry from conn shutdown, and every expiry is recorded via a RouterEvent
// plus a structural slog (method/id key/duration only — never params,
// T-5-03). A handler that answers past the expiry loses: the conn replies
// once, the late delivery lands in the buffered channel unread.
func (r *Responder) dispatch(ctx context.Context, pol MethodPolicy, req Request) (any, *zcode.WireError) {
	if pol.Deadline <= 0 {
		d := deliver(pol.Handler(ctx, req))

		return d.result, d.werr
	}

	ctx, cancel := context.WithTimeout(ctx, pol.Deadline)
	defer cancel()

	delivered := make(chan answerDelivery, 1)

	go func() {
		delivered <- deliver(pol.Handler(ctx, req))
	}()

	select {
	case d := <-delivered:
		return d.result, d.werr

	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// Conn shutdown owes nothing on the wire (the writer is
			// stopping; nothing lands).
			return nil, nil
		}

		r.recordDeadlineExpired(req, pol.Deadline, ctx.Err())

		return expiryAnswer(req.Method)
	}
}

// answerDelivery is the handler outcome reduced to the conn's reply
// vocabulary.
type answerDelivery struct {
	result any
	werr   *zcode.WireError
}

// deliver reduces a handler's (Answer, error) to the reply payload: an
// explicit Answer.Err wins; a bare error wraps as the -32603 internal
// code (numeric-code matching — dialect §6).
func deliver(ans Answer, err error) answerDelivery {
	if err != nil && ans.Err == nil {
		return answerDelivery{werr: &zcode.WireError{Code: zcode.CodeInternalError, Message: err.Error()}}
	}

	return answerDelivery{result: ans.Result, werr: ans.Err}
}

// expiryAnswer is the per-method answer when a deadline elapsed with the
// handler unanswered: requestPermission expiry answers deny with the
// denied-by-user reason (the OQ-2 disposition — the turn continues
// honestly, never hangs, never auto-allows), every other deadline answers
// the coded -32022 timeout error (consumers match the code, never the
// message — dialect §6).
func expiryAnswer(method string) (any, *zcode.WireError) {
	if method == MethodRequestPermission {
		return map[string]any{"decision": decisionDeny, wireKeyReason: deniedByUserReason}, nil
	}

	return nil, &zcode.WireError{
		Code:    zcode.CodeHostRequestTimedOut,
		Message: "Client request timed out: " + method,
	}
}

// recordDeadlineExpired records a deadline expiry observably — the
// injected RouterEvent hook when installed, plus a structural slog warning
// (record-honesty, mirroring conn.go:436-450). Method names, canonical id
// keys and durations only — never params (T-5-03).
func (r *Responder) recordDeadlineExpired(req Request, deadline time.Duration, err error) {
	if r.onEvent != nil {
		r.onEvent(zcode.RouterEvent{
			Kind:     zcode.RouterDeadlineExpired,
			Method:   req.Method,
			IDKey:    idKey(req),
			Deadline: deadline,
			Err:      err,
		})
	}

	slog.Warn("interact: reverse-request deadline expired without an answer",
		"method", req.Method,
		"id", idKey(req),
		"deadline", deadline.String(),
	)
}

// handlePermission forwards the ask to the ACP client via the injected
// PermissionForwarder (plan 05-02 implements the bridge). The nil default
// answers a coded not-wired error — never a fabricated allow, never
// silence (CORE-06: every ask is answered exactly once).
func (r *Responder) handlePermission(ctx context.Context, req Request) (Answer, error) {
	if r.permission == nil {
		return Answer{Err: &zcode.WireError{
			Code:    zcode.CodeInternalError,
			Message: "interaction/requestPermission not wired: no permission forwarder configured",
		}}, nil
	}

	return r.permission(ctx, req)
}

// handleRuntimeHeaders answers the account-provider credential delegation
// from the injected RuntimeHeadersKeySource (plan 05-06 injects the real
// source). The nil default answers a coded internal error naming the
// unconfigured key state — NEVER the failure-shaped success arm
// (headersApplied:false is a guaranteed turn death for account providers,
// PITFALL 3); on any failure the coded ERROR envelope is the honest arm.
func (r *Responder) handleRuntimeHeaders(ctx context.Context, req Request) (Answer, error) {
	if r.keySource == nil {
		return Answer{Err: &zcode.WireError{
			Code:    zcode.CodeInternalError,
			Message: "interaction/requestProviderRuntimeHeaders unavailable: no api-key source configured",
		}}, nil
	}

	apiKey, apiKeyID, accountScope, keyErr := r.keySource(ctx, req)
	if keyErr != nil {
		// A bare handler error wraps as the coded -32603 internal error at
		// the deliver seam — the honest coded-ERROR arm, never
		// headersApplied:false (PITFALL 3).
		return Answer{}, fmt.Errorf("interaction/requestProviderRuntimeHeaders: %w", keyErr)
	}

	auth := map[string]any{"apiKey": apiKey}

	if apiKeyID != "" {
		auth["apiKeyId"] = apiKeyID
	}

	// The widened success arm (Pattern 4/A9): accountScope joins requestAuth
	// ONLY when the source supplies it — an empty scope keeps the api-key
	// two-field answer byte-compatibly identical (the Phase-5 corpus bytes).
	if accountScope != "" {
		auth["accountScope"] = accountScope
	}

	// The success arm verbatim (dialect §3; the server zod-parses the
	// discriminatedUnion("headersApplied") reply schema — Pitfall 4).
	return Answer{Result: map[string]any{
		"headersApplied": true,
		"requestAuth":    auth,
	}}, nil
}

// RuntimeHeadersRejection digs the D-05 re-mint trigger fields out of a
// runtime-headers ask's params (plan 06-07, the mode-aware PAT source's
// inputs): expectedAccountScope and rejectedProjectTokenFingerprint dig as
// camelCase top-level strings, each validated as 64 LOWERCASE hex before
// being returned — a malformed or absent field digs as empty, and
// undecodable params classify as absent outright (the package's
// paramsString discipline). Neither value is ever logged here: the caller
// may log digests/scope hashes (they are digests), never token material.
func RuntimeHeadersRejection(req Request) (expectedScope, rejectedFingerprint string) {
	return paramsHex64(req.Params, "expectedAccountScope"),
		paramsHex64(req.Params, "rejectedProjectTokenFingerprint")
}

// hexFingerprintLen is the strict fingerprint/scope shape the ask schema
// pins (/^[a-f0-9]{64}$/) — sha256 hex length.
const hexFingerprintLen = 64

// paramsHex64 digs one top-level string value out of a params payload and
// gates it on the strict 64-lowercase-hex shape the ask schema pins
// (/^[a-f0-9]{64}$/) — anything else classifies as absent.
func paramsHex64(raw json.RawMessage, key string) string {
	s := paramsString(raw, key)
	if !isLowerHex64(s) {
		return ""
	}

	return s
}

// isLowerHex64 reports whether s is exactly 64 lowercase hex characters.
func isLowerHex64(s string) bool {
	if len(s) != hexFingerprintLen {
		return false
	}

	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}

// answerRuntimePreferences replies with the static runtime-preference
// defaults — the D-01 migration of conn.go:384-387's answer into the
// policy table, byte-identical (the same map literal marshals to the same
// wire bytes; the existing runtime-preferences corpus row re-pointed at
// the responder stays green).
func answerRuntimePreferences(_ context.Context, _ Request) (Answer, error) {
	return Answer{Result: map[string]any{
		"nativeSearchEnhancementsEnabled": false,
		"memoryEnabled":                   false,
	}}, nil
}

// The dialect's response literals (the ask-the-user reply's members).
const (
	wireActionOutcome = "action"
	wireActionDecline = "decline"
)

// handleUserInput is the ask-the-user policy row (v0.22.0, UIB-01..04):
// with the bridge wired, the decoded ask renders on the client surface
// (ONE form elicitation) and the outcome maps into the dialect's answers
// merge content; without it — the legacy honest decline (the schema-valid
// decline keeps the turn alive). Structural logs: id keys only, never
// params (T-5-03).
func (r *Responder) handleUserInput(ctx context.Context, req Request) (Answer, error) {
	if r.userInputAsk == nil {
		slog.Info("interact: requestUserInput declined (no UI channel; the ask bridge is not wired)",
			"method", req.Method,
			"requestId", req.ReqID,
			"id", idKey(req),
		)

		return Answer{Result: map[string]any{wireActionOutcome: wireActionDecline}}, nil
	}

	ask, decodeErr := acpport.DecodeUserInputAsk(req.Params)
	if decodeErr != nil {
		slog.Warn("interact: requestUserInput params undecodable; declining",
			"method", req.Method,
			"requestId", req.ReqID,
			"id", idKey(req),
			"err", decodeErr,
		)

		return Answer{Result: map[string]any{wireActionOutcome: wireActionDecline}}, nil
	}

	outcome, askErr := r.userInputAsk(ctx, ask)
	if askErr != nil {
		slog.Warn("interact: requestUserInput surface failed; declining",
			"method", req.Method, "requestId", req.ReqID, "id", idKey(req), "err", askErr,
		)

		return declineAnswer("the ask surface failed: " + askErr.Error()), nil
	}

	if outcome.Action != "accept" {
		slog.Info("interact: requestUserInput resolved by the user",
			"method", req.Method, "requestId", req.ReqID, "id", idKey(req), "action", outcome.Action,
		)

		return Answer{Result: map[string]any{wireActionOutcome: outcome.Action}}, nil
	}

	slog.Info("interact: requestUserInput answered",
		"method", req.Method, "requestId", req.ReqID, "id", idKey(req),
	)

	return acceptAnswer(outcome.Answers), nil
}

// declineAnswer is the schema-valid decline with an optional reason.
func declineAnswer(reason string) Answer {
	result := map[string]any{wireActionOutcome: wireActionDecline}
	if reason != "" {
		result["reason"] = reason
	}

	return Answer{Result: result}
}

// acceptAnswer wraps the question-keyed answers into the dialect's
// content.answers merge shape.
func acceptAnswer(answers map[string]string) Answer {
	content := map[string]any{}
	for k, v := range answers {
		content[k] = v
	}

	return Answer{Result: map[string]any{
		"action":  "accept",
		"content": map[string]any{"answers": content},
	}}
}

// answerOfficialMcpAuthHeaders declines the official-plugin MCP auth ask
// INSIDE the reply RESULT with the desktop's own honest no-resolver shape
// {ok:false, reason:"official_auth_unavailable"} (dialect §3 — the host's
// three RESULT replies; the plugin proceeds header-less on ok:false, its
// designed survivable path). The wrapper has no official-MCP auth
// resolver: never {ok:true} (no fabricated headers — an invented success
// would poison the plugin's auth state), never the -32601 error envelope
// (EXP-CODINGPLAN-CYCLE-01 attempt-1: the -32601 reply stalled the probe
// session's runtime materialization — the create reply's settings carried
// only the builtin registry, the read-back verdict timed out, and the
// authenticate attempt aborted), never silence. No deadline policy: the
// static decline answers immediately (a zero deadline dispatches the
// handler undelimited; the dedupe entry still ages out through the
// zero-deadline router-default TTL fallback). Structural log: id keys
// only, never params (T-5-03).
func answerOfficialMcpAuthHeaders(_ context.Context, req Request) (Answer, error) {
	slog.Info("interact: requestOfficialMcpAuthHeaders declined (no official-MCP auth resolver)",
		"method", req.Method,
		"requestId", req.ReqID,
		"id", idKey(req),
	)

	return Answer{Result: map[string]any{
		"ok":          false,
		wireKeyReason: "official_auth_unavailable",
	}}, nil
}

// unsupportedHostMethod is the -32601 floor for known-but-unanswered
// methods (the browser* pair): the same verbatim posture as the unknown
// default — a protocol error naming the method, never a fabricated
// success.
func unsupportedHostMethod(_ context.Context, req Request) (Answer, error) {
	return Answer{Err: &zcode.WireError{
		Code:    zcode.CodeMethodNotFound,
		Message: "unsupported host request: " + req.Method,
	}}, nil
}

// normalize digs the dedupe/session correlation keys out of params once,
// at serve time (the camelCase wire-key dig-out — machine.go paramsString
// discipline, tagliatelle-safe).
func normalize(env zcode.Envelope) Request {
	return Request{
		Method:    env.Method,
		ID:        env.ID,
		Params:    env.Params,
		ReqID:     paramsString(env.Params, "requestId"),
		SessionID: paramsString(env.Params, "sessionId"),
	}
}

// paramsString digs one top-level string value out of a params payload;
// undecodable params classify as absent.
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

// idKey renders the request's canonical id key; a nil id (never produced
// by the conn's demux) renders empty rather than panicking — CORE-06's
// never-crash floor extends to malformed envelopes.
func idKey(req Request) string {
	if req.ID == nil {
		return ""
	}

	return req.ID.Key()
}
