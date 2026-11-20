package interact

import (
	"context"

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
// params — resolves deny or the -32021 error reply, never allow.

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

// Forward answers one interaction/requestPermission ask.
//
// SKELETON (plan 05-02 Task 2 RED): the normalization and outcome
// translation land with GREEN — the skeleton answers the coded not-wired
// error so an unwired ask can never be mistaken for a deny or an allow.
func (b *PermissionBridge) Forward(_ context.Context, _ Request) (Answer, error) {
	return Answer{Err: &zcode.WireError{
		Code:    zcode.CodeInternalError,
		Message: "interaction/requestPermission bridge not wired (skeleton)",
	}}, nil
}
