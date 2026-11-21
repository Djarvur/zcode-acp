package acpfront

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// The front half of the TOOL-02 permission bridge (plan 05-02): the
// in-flight pending table and the client round-trip. The table is
// SESSION-SCOPED (plan 05-15 WR-07): map[sessionID]map[requestID]cancel —
// one inner entry per backend requestId holds the round-trip's cancel
// func (published under one mutex — the turnctx.go acquireTurn
// discipline: complete-or-none state); the cancel fan-ins take exactly
// the named session's bucket so a bridge never outlives its turn (→
// -32021 replies, T-5-05) and never crosses sessions (a foreign
// session's in-flight ask survives a cancel/close of another session),
// and each entry is deleted under the same lock on return
// (delete-under-lock exactly-once, the call.go pendingTable discipline;
// empty buckets are deleted so the table stays bounded). Growth is capped
// by one entry per in-flight ask — bounded further by the responder's
// per-method deadline.
//
// Lock ordering (T-5-38, plan 05-15): the one-way order turnMu → b.mu.
// The cancel fan-in callers take pendings under the SAME turnMu section
// that owns the slot observation / pump-identity re-check and fire the
// cancels outside any lock; every bridge method below acquires b.mu ONLY
// and never turnMu (PermissionForward releases turnMu before entering
// forward), so no cycle can form.

// permissionBridge is the Front-owned bridge component; the connection-
// bound PermissionSender binds at acpport construction (the sinkBinder
// discipline — before any frame flows).
type permissionBridge struct {
	mu      sync.Mutex
	pending map[string]map[string]context.CancelFunc
	send    acpport.PermissionSender
}

// newPermissionBridge builds the bridge with an unbound sender and an
// empty pending table.
func newPermissionBridge() *permissionBridge {
	return &permissionBridge{pending: make(map[string]map[string]context.CancelFunc)}
}

// bindSender attaches the connection-bound sender; called once by
// acpport.NewAgentSideConnection during construction.
func (b *permissionBridge) bindSender(send acpport.PermissionSender) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.send = send
}

// boundSender returns the bound sender under the lock; nil until
// construction binds it.
func (b *permissionBridge) boundSender() acpport.PermissionSender {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.send
}

// forward performs the client round-trip for one normalized ask: the
// in-flight entry joins the pending table (keyed by the backend
// requestId), the bound sender runs on the entry's own derived context, and
// the entry is removed under the lock before the outcome returns. A
// locally-cancelled round-trip (the turn's cancel fan-in) reads as the
// cancelled outcome — never an error the backend would misread as a
// transport failure; every other send failure surfaces as-is.
func (b *permissionBridge) forward(ctx context.Context, ask acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
	send := b.boundSender()
	if send == nil {
		return acpport.PermissionOutcome{}, acpport.NewInternal(
			"acpfront: no permission sender bound — construct via acpport.NewAgentSideConnection")
	}

	pctx, cancel := context.WithCancel(ctx)

	tracked := ask.RequestID != ""
	if tracked {
		b.addPending(ask.SessionID, ask.RequestID, cancel)
	}

	outcome, err := send(pctx, ask)

	if tracked {
		b.removePending(ask.SessionID, ask.RequestID)
	}

	cancel() // outside any lock: releases the entry's derived context either way

	if err != nil && pctx.Err() != nil {
		// The round-trip was cancelled locally (the turn's cancel fan-in,
		// or the responder deadline the caller owns): resolve as the
		// cancelled outcome — the backend answers its pending ask -32021
		// and ends the turn honestly. Returning a nil error beside the
		// send error is the cancelled classification itself (the caller
		// matches the outcome, not the transport failure).
		//
		//nolint:nilerr // deliberate cancelled classification — the outcome IS the contract
		return acpport.PermissionOutcome{Cancelled: true}, nil
	}

	if err != nil {
		return acpport.PermissionOutcome{}, err
	}

	return outcome, nil
}

// addPending registers one in-flight round-trip's cancel func under its
// session's bucket (the ask's SessionID is the WR-07 scope key).
func (b *permissionBridge) addPending(sessionID, requestID string, cancel context.CancelFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	bucket, ok := b.pending[sessionID]
	if !ok {
		bucket = make(map[string]context.CancelFunc)
		b.pending[sessionID] = bucket
	}

	bucket[requestID] = cancel
}

// removePending deletes one in-flight entry (delete-under-lock — the
// exactly-once discipline: a cancelled entry that raced the answer can
// never fire twice) and the session's bucket when it empties (the
// bounded-table house discipline).
func (b *permissionBridge) removePending(sessionID, requestID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	bucket, ok := b.pending[sessionID]
	if !ok {
		return
	}

	delete(bucket, requestID)

	if len(bucket) == 0 {
		delete(b.pending, sessionID)
	}
}

// takePendingCancels snapshots and deletes ONLY the named session's
// pending cancels (delete-under-lock, exactly-once — a cancelled entry
// that raced the answer can never fire twice; empty-bucket cleanup keeps
// the table bounded; the caller FIRES the cancels outside any lock, each
// entry's forward observing the cancellation through its own derived
// context exactly once). Session scoping is the WR-07 closure: a fan-in
// for one session can never sweep a foreign session's (or a later turn's
// fresh) ask, and the takes on the Front.Cancel / mine-arm call sites run
// under the SAME turnMu section as the slot observation / pump-identity
// re-check — check-and-take is atomic, so no deschedule window exists.
func (b *permissionBridge) takePendingCancels(sessionID string) []context.CancelFunc {
	b.mu.Lock()
	defer b.mu.Unlock()

	bucket, ok := b.pending[sessionID]
	if !ok {
		return nil
	}

	cancels := make([]context.CancelFunc, 0, len(bucket))
	for id, cancel := range bucket {
		cancels = append(cancels, cancel)
		delete(bucket, id)
	}

	delete(b.pending, sessionID)

	return cancels
}

// fireTakenPendings logs and fires the cancels a take returned — the ONE
// home of the fan-in's discriminator log and the fire loop (takes happen
// under whatever lock section owns the observation; the fires always run
// outside any lock).
func fireTakenPendings(cancels []context.CancelFunc) {
	if len(cancels) > 0 {
		slog.Info("acpfront: permission pendings cancelled by turn cancel", "count", len(cancels))
	}

	for _, cancel := range cancels {
		cancel()
	}
}

// cancelPendingForSession cancels every in-flight round-trip of ONE
// session (the close-path fan-in — pending bridges never outlive their
// turn; session scoping contains the sweep to the closing session,
// WR-07). Delegates to take + fire so the log/message have exactly one
// home.
func (b *permissionBridge) cancelPendingForSession(sessionID string) {
	fireTakenPendings(b.takePendingCancels(sessionID))
}

// permissionAskDrainWait bounds the wire-order barrier the ask barrier
// waits through before forwarding (a stalled pump must not stall the ask).
const permissionAskDrainWait = pumpReleaseWait

// PermissionForward returns the front bridge func the interact Responder
// wiring injects as its PermissionForwarder seam (interact.
// NewPermissionBridge over this func). Before the round-trip it barriers
// on the turn pump: the ask frame must land AFTER the tool frames the
// backend emitted before its interaction request (the started card precedes
// the ask — the fullcycle wire-order lock); the pump and the direct
// permission send are otherwise independent wires with no shared ordering.
func (f *Front) PermissionForward() func(
	ctx context.Context, ask acpport.PermissionAsk,
) (acpport.PermissionOutcome, error) {
	return func(ctx context.Context, ask acpport.PermissionAsk) (acpport.PermissionOutcome, error) {
		f.turnMu.Lock()
		pump := f.turnPump
		f.turnMu.Unlock()

		if pump != nil {
			pump.barrier(permissionAskDrainWait)
		}

		return f.perm.forward(ctx, ask)
	}
}

// BindPermissionSender is the acpport.PermissionBinder hook: the adapter
// forwards the connection-bound sender here during construction, before
// any frame flows.
func (f *Front) BindPermissionSender(sender acpport.PermissionSender) {
	f.perm.bindSender(sender)
}
