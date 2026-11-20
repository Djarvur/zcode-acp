package acpfront

import (
	"context"
	"sync"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// The front half of the TOOL-02 permission bridge (plan 05-02): the
// in-flight pending table and the client round-trip. One entry per backend
// requestId holds the round-trip's cancel func (published under one mutex —
// the turnctx.go acquireTurn discipline: complete-or-none state); Front.
// Cancel's fan-in cancels every pending entry so a bridge never outlives
// its turn (→ -32021 replies, T-5-05), and the entry is removed under the
// same lock on return (delete-under-lock exactly-once, the call.go
// pendingTable discipline).

// permissionBridge is the Front-owned bridge component; the connection-
// bound PermissionSender binds at acpport construction (the sinkBinder
// discipline — before any frame flows).
type permissionBridge struct {
	mu      sync.Mutex
	pending map[string]context.CancelFunc
	send    acpport.PermissionSender
}

// newPermissionBridge builds the bridge with an unbound sender.
//
// SKELETON (plan 05-02 Task 2 RED): the pending table and round-trip land
// with GREEN.
func newPermissionBridge() *permissionBridge {
	return &permissionBridge{}
}

// bindSender attaches the connection-bound sender; called once by
// acpport.NewAgentSideConnection during construction.
func (b *permissionBridge) bindSender(_ acpport.PermissionSender) {}

// forward performs the client round-trip for one normalized ask.
//
// SKELETON: fails loud — an unwired round-trip must never read as a
// cancelled or selected outcome.
func (b *permissionBridge) forward(_ context.Context, _ acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
	return acpport.PermissionOutcome{}, acpport.NewInternal("acpfront: permission bridge not wired (skeleton)")
}

// cancelPendingForTurn cancels every in-flight round-trip.
func (b *permissionBridge) cancelPendingForTurn() {}

// PermissionForward returns the front bridge func the interact Responder
// wiring injects as its PermissionForwarder seam (interact.
// NewPermissionBridge over this func).
func (f *Front) PermissionForward() func(ctx context.Context, ask acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
	return f.perm.forward
}

// bindPermissionSender is the acpport permissionBinder hook: the adapter
// forwards the connection-bound sender here during construction, before
// any frame flows.
func (f *Front) bindPermissionSender(sender acpport.PermissionSender) {
	f.perm.bindSender(sender)
}
