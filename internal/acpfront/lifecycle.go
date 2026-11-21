package acpfront

import (
	"context"
	"log/slog"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/core"
	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// SESS-01 session lifecycle (plan 05-05, D-04-routed teardown work):
// Front.ResumeSession and Front.CloseSession — the contracts Task 2's
// RED→GREEN locked in lifecycle_test.go over the session-lifecycle corpus
// row.
//
//   - Resume reattaches a backend session: backend session/resume (the
//     reply is the SAME full snapshot shape as create — createSessionID
//     digs result.session.sessionId unchanged) → RE-ISSUED
//     session/subscribe {deliveryKind:desktop-continuous} (the
//     subscribe-before-any-turn gate applies to a reattached session
//     exactly as to a new one) → machine + registry entry registered →
//     {sessionId} returned. NO setModel follow-up (model binding persists
//     backend-side) and NO replay — the wrapper never synthesizes updates
//     for pre-resume history; the client sees only future turns (D-04).
//   - Close runs the teardown ORDER: cancel any active turn
//     (machine.Cancel; bounded wait for the slot to drain) → best-effort
//     backend session/close (teardown continues even when the backend
//     call fails — structurally logged) → registry Remove (the first
//     production consumer of core.Registry.Remove) → both observer
//     removes (the plan-05-03 exposed Machine.CloseObservers) → reply.
//     A post-close prompt fails cleanly with the ErrNoSession class
//     (never a panic, never a zombie session); a second close is the
//     clean coded rejection (idempotent teardown by the Get gate).
//
// The close-vs-prompt race is deterministic (T-5-12, Pitfall 6): close
// marks the session in the closing set for the whole teardown span, so a
// racing prompt lands exactly one clean outcome — the coded closing/
// closed rejection, or its already-running turn resolves cancelled (the
// drain re-cancels and waits the turn out before the release steps).

// closeTurnWait bounds the close teardown's wait for an active turn to
// unwind after its machine.Cancel (the cancel fires first; the stop's
// observed outcome resolves the turn in milliseconds, so the bound has
// orders of magnitude of margin while keeping close bounded — the
// spawnWait/shutdownWatchdog scale, defined locally per plan).
const closeTurnWait = 10 * time.Second

// ResumeSession reattaches one backend session (SESS-01). The id comes
// back through the create-shaped snapshot dig-out; the caller-supplied id
// is validated passable (a non-empty string — a rejected resume never
// touches the backend wire, T-5-13).
func (f *Front) ResumeSession(ctx context.Context, sessionID string) (string, error) {
	if sessionID == "" {
		return "", acpport.NewInvalidParams("session/resume: sessionId must not be empty")
	}

	res, err := f.conn.Call(ctx, zcode.MethodSessionResume, map[string]any{
		wireKeySessionID: sessionID,
	})
	if err != nil {
		return "", acpport.NewInternal("session/resume: %v", err)
	}

	sid, err := createSessionID(res, zcode.MethodSessionResume)
	if err != nil {
		return "", err
	}

	// The re-issued subscribe (the D-03 gate applies to a reattached
	// session exactly as to a new one — session/event pushes are gated on
	// the deliveryKind record, Pitfall 1). No setModel: the binding
	// persists backend-side; the resume params carry no fabricated hints.
	if _, err := f.conn.Call(ctx, zcode.MethodSessionSubscribe, map[string]any{
		wireKeySessionID: sid,
		"deliveryKind":   "desktop-continuous",
	}); err != nil {
		return "", acpport.NewInternal("session/resume subscribe: %v", err)
	}

	// In-process reattach (the registry still holds the id from an earlier
	// open): release the previous machine's observers before Register
	// replaces it — the WR-07 discipline (Register is last-wins; without
	// this the old machine's fan-out entries would leak).
	if prev, getErr := f.registry.Get(sid); getErr == nil && prev.Machine != nil {
		prev.Machine.CloseObservers()
	}

	// The machine registers on the conn's fan-out AFTER the tap — the same
	// flush-before-stopReason registration order NewSession relies on.
	machine := turn.NewMachine(f.conn, sid, turn.NewPollWatcher(f.conn, turn.PollConfig{}))
	f.registry.Register(core.Session{ID: sid, Machine: machine})

	slog.Info("acpfront: session resumed", "session", sid)

	return sid, nil
}

// CloseSession runs the ordered teardown (SESS-01). Unregistered ids fail
// clean before anything touches the backend wire (T-5-13).
func (f *Front) CloseSession(ctx context.Context, sessionID string) error {
	st, err := f.registry.Get(sessionID)
	if err != nil {
		return acpport.NewInvalidRequest("session/close: %v: %s", err, sessionID)
	}

	f.closingMu.Lock()

	if f.closing[sessionID] {
		f.closingMu.Unlock()

		return acpport.NewInvalidRequest("session/close: %s: close already in progress", sessionID)
	}

	if f.closing == nil {
		f.closing = make(map[string]bool)
	}

	// The race gate (T-5-12): from here until the teardown completes, a
	// racing prompt lands exactly one clean coded rejection instead of
	// acquiring the slot mid-close.
	f.closing[sessionID] = true

	f.closingMu.Unlock()

	// Teardown order, step 1: cancel any active turn and wait bounded for
	// it to drain (the stop's observed outcome resolves it; the drain also
	// re-cancels a turn that sneaked in at the slot).
	if f.drainActiveTurn(sessionID, st) {
		// Pitfall 2: the cancelled turn's pending permission bridges must
		// not dangle — one turn per front, so the whole pending table was
		// that turn's (the same fan-in Front.Cancel performs).
		f.perm.cancelPendingForTurn()
	}

	// Step 2: best-effort backend close — teardown continues even when the
	// backend call fails (structurally logged, never a zombie release
	// skip).
	if _, err := f.conn.Call(ctx, zcode.MethodSessionClose, map[string]any{
		wireKeySessionID: sessionID,
	}); err != nil {
		slog.Warn("acpfront: backend session/close failed (teardown continues)",
			"session", sessionID, "err", err)
	}

	// A turn can only have sneaked in between the first drain and here via
	// the slot BEFORE the closing gate — impossible: the gate predates both
	// drains. The second drain is the cheap re-check that keeps the
	// release steps strictly behind an unwound turn.
	f.drainActiveTurn(sessionID, st)

	// Steps 3-5: registry release, then both observer removes (WR-07 — the
	// plan-05-03 exposed CloseObservers drives the notification and router
	// fan-outs, idempotently), then the cwd entry (the map stays bounded
	// with the registry).
	f.registry.Remove(sessionID)
	st.Machine.CloseObservers()
	f.dropCwd(sessionID)

	f.closingMu.Lock()
	delete(f.closing, sessionID)
	f.closingMu.Unlock()

	slog.Info("acpfront: session closed", "session", sessionID)

	return nil
}

// drainActiveTurn cancels the session's active turn (machine.Cancel —
// idempotent, no-op when idle) and waits bounded for the turn to unwind
// (the slot's done channel closes when release has flushed). It reports
// whether a turn of THIS session was observed and cancelled (sawTurn):
// true on BOTH the successful-drain arm and the deadline-elapsed arm (a
// turn WAS observed; the caller's permission fan-in fires best-effort past
// the deadline), false only when no turn of this session was ever observed
// — so the caller only cancels permission bridges that belong to it
// (CR-01, plan 05-10: the fan-in fires exactly when a turn of the session
// was cancelled, never on an idle close). A turn of another session (the
// front-wide slot) never blocks this session's close beyond the check.
func (f *Front) drainActiveTurn(sessionID string, st core.Session) bool {
	deadline := time.Now().Add(f.closeWait())

	sawTurn := false

	for {
		f.turnMu.Lock()

		var done chan struct{}
		if f.turnActive && f.turnSession == sessionID {
			done = f.turnDone
		}

		f.turnMu.Unlock()

		if done == nil {
			return sawTurn // false = no turn of this session ever ran; true = observed, cancelled, drained
		}

		sawTurn = true

		// Cancel fires first (re-cancels a sneaked second turn —
		// machine.Cancel is idempotent per turn and a new turn is a new
		// resolve target).
		st.Machine.Cancel()

		remain := time.Until(deadline)
		if remain <= 0 {
			slog.Warn("acpfront: close turn wait elapsed; proceeding best-effort",
				"session", sessionID)

			return true
		}

		select {
		case <-done:
		case <-time.After(remain):
		}
	}
}

// closeWait returns the effective close-teardown wait for an active turn:
// the injected override (WithCloseTurnWait — the plan-05-10 deadline-arm
// test seam) or the closeTurnWait production default.
func (f *Front) closeWait() time.Duration {
	if f.closeWaitOverride > 0 {
		return f.closeWaitOverride
	}

	return closeTurnWait
}
