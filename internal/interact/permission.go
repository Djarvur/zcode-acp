package interact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// The backend half of the TOOL-02 permission bridge (plan 05-02): the
// Responder's PermissionForwarder. It normalizes the interaction/
// requestPermission ask (camelCase dig-out), maps the backend options onto
// the neutral vocabulary per the locked table (allow_once→allow_once,
// allow_project/allow_always→allow_always, deny→reject_once — names and
// optionIds verbatim), hands the ask to the injected front bridge, and
// translates the outcome: a selected DECLARED option forwards that option's
// own embedded response byte-verbatim (never re-serialized into a
// different shape); anything else — unknown or missing optionId, the ACP
// cancelled outcome, a cancelled-classified forwarder error, malformed
// params — resolves deny or the -32021 error reply, never allow (T-5-04,
// mirroring the backend's own unknown-optionId→deny posture,
// interaction-broker.ts:241-243).

// deniedByUserReason is the reason carried by every wrapper-synthesized
// deny (unknown or missing optionId, malformed params): our own honest
// text — the backend's PERMISSION_DENIED_BY_USER_CONTENT value is its own
// (cite-and-point, never extracted; consumers match decisions, not
// reasons).
const deniedByUserReason = "Permission denied by user"

// cancelledPermissionMessage is the stable message of the -32021 error
// reply (the corpus rows' expect_answer byte-locks it; the backend maps
// the numeric code to ToolCancelled — the code is the contract, the
// message our own constant).
const cancelledPermissionMessage = "Client request cancelled: interaction/requestPermission"

// The backend option-kind wire literals (zcodePermissionOptionSchema
// vocabulary — the CLI synthesizes exactly these three; unknown kinds pass
// through verbatim as UI hints).
const (
	kindAllowOnce    = "allow_once"
	kindAllowProject = "allow_project"
	kindAllowAlways  = "allow_always"
	kindDeny         = "deny"

	// decisionDeny is the deny decision literal (the response schema enum).
	decisionDeny = "deny"
)

// FrontBridge is the front-tier round-trip seam (implemented by the
// acpfront bridge, injected at composition): one normalized neutral ask in,
// the client's decoded outcome out.
type FrontBridge func(ctx context.Context, ask acpport.PermissionAsk) (acpport.PermissionOutcome, error)

// PermissionBridge implements the Responder's PermissionForwarder contract
// over the injected FrontBridge. Constructed via NewPermissionBridge and
// immutable afterwards; Forward is safe for concurrent use.
type PermissionBridge struct {
	front FrontBridge
}

// NewPermissionBridge builds the backend bridge over the front-tier
// round-trip seam.
func NewPermissionBridge(front FrontBridge) *PermissionBridge {
	return &PermissionBridge{front: front}
}

// Forward answers one interaction/requestPermission ask: normalize →
// forward → translate (deny-by-default on every outcome the backend's own
// options do not define).
func (b *PermissionBridge) Forward(ctx context.Context, req Request) (Answer, error) {
	if b.front == nil {
		return Answer{Err: &zcode.WireError{
			Code:    zcode.CodeInternalError,
			Message: "interaction/requestPermission not wired: no front bridge configured",
		}}, nil
	}

	ask, options := digPermissionAsk(req)

	if len(options) == 0 {
		// Malformed params (no options array, or entries without usable
		// ids): no option can be declared, so nothing may be allowed —
		// deny without forwarding (T-5-04 deny-by-default posture).
		slog.Warn("interact: permission ask carries no usable options — deny by default",
			"method", req.Method, "requestId", req.ReqID, "id", idKey(req))

		return denyAnswer(), nil
	}

	outcome, err := b.front(ctx, ask)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Answer{Err: cancelledWireError()}, nil
		}

		// A bare error wraps as the coded -32603 at the deliver seam —
		// never a fabricated decision.
		return Answer{}, fmt.Errorf("interaction/requestPermission: %w", err)
	}

	if outcome.Cancelled {
		return Answer{Err: cancelledWireError()}, nil
	}

	opt := findOption(options, outcome.OptionID)
	if opt == nil {
		// Unknown or missing optionId → deny with the denied-by-user
		// reason — never allow (the backend's own posture, mirrored).
		slog.Warn("interact: permission outcome names no declared option — deny by default",
			"method", req.Method, "requestId", req.ReqID, "id", idKey(req))

		return denyAnswer(), nil
	}

	// The selected option's OWN embedded response, byte-verbatim
	// (json.RawMessage passthrough — never re-serialized into a different
	// shape, never a synthesized decision/reason/modifiedInput).
	return Answer{Result: opt.response}, nil
}

// permissionOption is one dug-out backend option: the correlation fields
// plus its embedded response kept as the original bytes.
type permissionOption struct {
	optionID string
	kind     string
	name     string
	response json.RawMessage
}

// digPermissionAsk normalizes the reverse request's params into the
// neutral ask (mapped options included) and the flat option list the
// outcome lookup scans. The dig-out is map-based — the wire keys are the
// dialect's camelCase (tagliatelle-safe, machine.go paramsString
// discipline) — and the embedded responses stay byte-verbatim via
// RawMessage. Undecodable fields classify as absent.
func digPermissionAsk(req Request) (acpport.PermissionAsk, []permissionOption) {
	var params map[string]json.RawMessage
	_ = json.Unmarshal(req.Params, &params)

	ask := acpport.PermissionAsk{
		SessionID:  req.SessionID,
		RequestID:  req.ReqID,
		ToolCallID: rawString(params["toolCallId"]),
		ToolName:   rawString(params["toolName"]),
	}

	var optionsRaw []json.RawMessage
	if len(params["options"]) > 0 {
		_ = json.Unmarshal(params["options"], &optionsRaw)
	}

	options := make([]permissionOption, 0, len(optionsRaw))

	for _, raw := range optionsRaw {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			continue
		}

		opt := permissionOption{
			optionID: rawString(obj["optionId"]),
			kind:     rawString(obj["kind"]),
			name:     rawString(obj["name"]),
			response: obj["response"],
		}

		if opt.optionID == "" {
			continue
		}

		options = append(options, opt)

		ask.Options = append(ask.Options, acpport.PermissionOptionDef{
			Kind:     permissionKindToACP(opt.kind),
			Name:     opt.name,
			OptionID: opt.optionID,
		})
	}

	return ask, options
}

// permissionKindToACP maps the backend option kind onto the ACP kind per
// the locked table: allow_once→allow_once, allow_project/allow_always→
// allow_always, deny→reject_once. Unknown kinds pass through verbatim
// (kind is a UI hint only per the ACP v1 spec — the optionId is the
// contract).
func permissionKindToACP(kind string) string {
	switch kind {
	case kindAllowOnce:
		return acpport.PermKindAllowOnce
	case kindAllowProject, kindAllowAlways:
		return acpport.PermKindAllowAlways
	case kindDeny:
		return acpport.PermKindRejectOnce
	default:
		return kind
	}
}

// findOption scans the dug-out options for the selected id; nil when the
// outcome names no declared option.
func findOption(options []permissionOption, optionID string) *permissionOption {
	for i := range options {
		if options[i].optionID == optionID {
			return &options[i]
		}
	}

	return nil
}

// rawString decodes one raw JSON string value; anything else is absent.
func rawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}

	return s
}

// denyAnswer is the wrapper-synthesized deny (unknown/missing optionId,
// malformed params, deadline expiry via expiryAnswer): the denied-by-user
// reason, never a bare decision.
func denyAnswer() Answer {
	return Answer{Result: map[string]any{"decision": decisionDeny, wireKeyReason: deniedByUserReason}}
}

// cancelledWireError is the -32021 error reply for a cancelled permission
// round-trip (the ACP cancelled outcome or a cancelled-classified
// forwarder error): the backend maps the numeric code to ToolCancelled and
// ends the turn — the pending ask resolves exactly once.
func cancelledWireError() *zcode.WireError {
	return &zcode.WireError{Code: zcode.CodeHostRequestCancelled, Message: cancelledPermissionMessage}
}
