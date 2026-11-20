// Package zcode implements the JSONL codec for the ZCode desktop app-server
// dialect: one newline-delimited JSON envelope per wire line, string-or-int
// ids, no JSON-RPC framing key, a single-writer output stage with a bypass
// lane, and a pending-call table keyed by type-tagged id.
//
// Wire truth is docs/protocol/app-server-dialect.md (the shipped 3.18.2
// surface); the 3.17.x source clone is corroborating only and is never cited
// as wire truth here (dialect §9 drift baseline).
package zcode

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Canonical id-key prefixes: the type tag keeps "5" (string) and 5 (int)
// from ever colliding in the pending-call table (RESEARCH Pitfall 8, WR-02).
const (
	intKeyPrefix = "i:"
	strKeyPrefix = "s:"
)

// ID is a wire request id: string-or-int per dialect §1. The two forms are
// distinct wire values ("5" and 5 never correlate), so consumers key on the
// type-tagged canonical Key.
type ID struct {
	Int   int64
	Str   string
	IsInt bool
}

// Key returns the canonical type-tagged correlation key: "i:<n>" for integer
// ids, "s:<s>" for string ids. Replies echo the request id verbatim (dialect
// §1), so correlation is by exact id value AND type.
func (id *ID) Key() string {
	if id.IsInt {
		return intKeyPrefix + strconv.FormatInt(id.Int, 10)
	}

	return strKeyPrefix + id.Str
}

// MarshalJSON renders the id as a bare JSON number or string, per the
// dialect §1 envelope grammar (ids are never quoted numbers).
func (id *ID) MarshalJSON() ([]byte, error) {
	if id.IsInt {
		b, err := json.Marshal(id.Int)
		if err != nil {
			return nil, fmt.Errorf("zcode: marshal int id: %w", err)
		}

		return b, nil
	}

	b, err := json.Marshal(id.Str)
	if err != nil {
		return nil, fmt.Errorf("zcode: marshal string id: %w", err)
	}

	return b, nil
}

// UnmarshalJSON accepts either a JSON number or a JSON string and records
// which form arrived (dialect §1 — ids are string-or-int on the wire; the
// probe saw integer client ids and string server ids).
func (id *ID) UnmarshalJSON(b []byte) error {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		id.Int, id.Str, id.IsInt = n, "", true

		return nil
	}

	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("zcode: id is neither number nor string: %w", err)
	}

	id.Int, id.Str, id.IsInt = 0, s, false

	return nil
}

// Envelope is one wire line (dialect §1 grammar). The struct has no JSON-RPC
// framing field by construction: a jsonrpc key is unrepresentable in
// outbound bytes — the structural defense against the child's -32600
// unrecognized_keys rejection (dialect §1).
type Envelope struct {
	ID     *ID             `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *WireError      `json:"error,omitempty"`
}

// WireError is the error payload of a reply envelope (dialect §1: numeric
// code and message, optional structured data — observed live as the zod
// issue arrays behind -32600/-32603).
type WireError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Encode marshals env to one newline-terminated wire line. The typed struct
// cannot emit a JSON-RPC framing key — there is no such field, the
// structural defense against the child's -32600 unrecognized_keys rejection
// (dialect §1).
func Encode(env *Envelope) ([]byte, error) {
	frame, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("zcode: encode envelope: %w", err)
	}

	line := make([]byte, 0, len(frame)+1)
	line = append(line, frame...)
	line = append(line, '\n')

	return line, nil
}

// Decode parses one wire line into an envelope. Undecodable JSON returns an
// error — the caller logs and skips the line, and the stream continues (the
// host-side analog of the child's -32700 parse-error path, dialect §1).
// Params and result stay byte-exact as json.RawMessage.
func Decode(line []byte) (Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return Envelope{}, fmt.Errorf("zcode: decode envelope: %w", err)
	}

	return env, nil
}

// Method constants: verbatim strings from the dialect §2 method map, named
// after the dialect method strings (PascalCase) per D-06. The map names
// survive minification in the shipped bundle; 3.18.2 is the truth source.
const (
	MethodSessionCreate             = "session/create"
	MethodSessionSend               = "session/send"
	MethodSessionStop               = "session/stop"
	MethodSessionMessages           = "session/messages"
	MethodRequestRuntimePreferences = "session/requestRuntimePreferences"
	MethodRequestCredentialCipher   = "interaction/requestCredentialCipher"
	MethodStateUpdated              = "state.updated"
	MethodSessionEvent              = "session/event"
	MethodSessionLifecycleChanged   = "session/lifecycle/changed"
	MethodSessionSubscribe          = "session/subscribe"
	MethodSessionSetModel           = "session/setModel"
)

// Error-code constants carry the dialect §6 numeric values verbatim.
// Consumers match the numeric code, never the message text (dialect §6
// matching rule: messages are not a stable contract). -32042 and -32002 are
// 3.18.2-only (dialect §9 drift baseline).
const (
	CodeParseError                      = -32700
	CodeInvalidRequest                  = -32600
	CodeMethodNotFound                  = -32601
	CodeInvalidParams                   = -32602
	CodeInternalError                   = -32603
	CodeResourceNotFound                = -32002
	CodeConcurrency                     = -32010
	CodeHeaderMismatch                  = -32020
	CodeMissingRequiredClientCapability = -32021
	CodeUnsupportedProtocolVersion      = -32022
	CodeProviderHeadersNotApplied       = -32031
	CodeURLElicitationRequired          = -32042
)

// Client-request-error codes: the dialect's client-request error namespace
// (server-types.ts PROTOCOL_CLIENT_REQUEST_ERROR_CODES values verbatim —
// the guarded-mode codes the interaction broker maps to
// ConfigurationError / ToolCancelled / PermissionTimeout). The numbers
// deliberately collide with the header-space constants above
// (CodeHeaderMismatch -32020, CodeMissingRequiredClientCapability -32021,
// CodeUnsupportedProtocolVersion -32022): the dialect itself reuses these
// numbers across namespaces — consumers match the code within their own
// namespace, never the message text (dialect §6 matching rule).
const (
	CodeHostNoClientAttached = -32020
	CodeHostRequestCancelled = -32021
	CodeHostRequestTimedOut  = -32022
)
