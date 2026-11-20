package acpfront

import (
	"context"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// Per-turn context ownership (RESEARCH Pattern 5, CORE-05): each prompt
// turn runs on a FRONT-OWNED context derived from the Front's base
// context — never the SDK request context. The SDK maps a handler's
// context.Canceled to a -32800 RequestCancelled ERROR response, while ACP
// requires session/cancel answered with stopReason:"cancelled" as a
// SUCCESS result — so the turn context is cancelled only when the prompt
// returns (release), and cancellation reaches the turn exclusively through
// machine.Cancel (idempotent, exactly once: Phase 3), which resolves
// StopCancelled via the observed stop outcome. Prompt's cancel fan-in
// goroutine selects {SDK ctx.Done → machine.Cancel; turn ctx.Done →
// return} so both cancel sources trigger exactly one machine.Cancel, and
// post-cancel delta drain still reaches the client because the tap writes
// on the turn context (alive until the prompt returns).

// acquireTurn takes the front-wide turn slot for one prompt turn (the
// single-active-turn invariant). It returns the front-owned turn context
// and the slot release func; release is nil when the slot is already held
// — a second prompt while a turn is active, the D-05 guard surface (the
// exact error class is plan 04-05's RED target). The tap-facing emitter
// (the closure binding the turn's sink, session id, and context) publishes
// under the same lock, so the stream.go tap observes either the complete
// turn state or none.
func (f *Front) acquireTurn(sessionID string, sink acpport.UpdateSink) (turnCtx context.Context, release func()) {
	f.turnMu.Lock()
	defer f.turnMu.Unlock()

	if f.turnActive {
		return nil, nil
	}

	// Front-owned: derived from the front's base context, NOT the SDK
	// request context (the type doc — cancelling on the SDK ctx would turn
	// the cancelled stopReason into a -32800 error response).
	turnCtx, turnCancel := f.newTurnCtx()

	f.turnActive = true
	f.turnEmit = func(note acpport.SessionNote) error {
		note.SessionID = sessionID

		return sink(turnCtx, note)
	}

	// The tap's turn-scoped classification state (plan 05-04): the slot's
	// turnId starts empty (record-on-first-sight at the tap, IN-05), the
	// cwd anchors the absolute diff/location paths (Pitfall 5), and the
	// diff registry starts fresh per turn.
	f.turnID = ""
	f.turnCwd = f.sessionCwdLocked(sessionID)
	f.turnDiffs = make(map[string]acpport.ToolDiff)

	release = func() {
		f.turnMu.Lock()

		f.turnActive = false
		f.turnEmit = nil
		f.turnID = ""
		f.turnCwd = ""
		f.turnDiffs = nil

		f.turnMu.Unlock()

		turnCancel() // outside the lock: wakes the fan-in goroutine's turnCtx.Done arm
	}

	return turnCtx, release
}

// sessionCwdLocked reads the session-cwd registry; the caller holds turnMu
// (a different mutex than the registry's cwdMu — no lock ordering issue,
// this is a leaf read).
func (f *Front) sessionCwdLocked(sessionID string) string {
	f.cwdMu.Lock()
	defer f.cwdMu.Unlock()

	return f.cwds[sessionID]
}
