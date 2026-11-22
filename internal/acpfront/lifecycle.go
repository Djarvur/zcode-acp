package acpfront

import (
	"context"
	"log/slog"
	"path/filepath"
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
// touches the backend wire, T-5-13). The cwd rides the SDK
// ResumeSessionRequest.Cwd and MUST be absolute (the session-setup MUST —
// validated BEFORE any backend call, the T-4-01 parity guard, plan 05-10
// WR-02) and is recorded for the reattached id so the session's tool
// diff/location paths resolve absolute (Pitfall 5 off the resume path).
// A resume racing a close is rejected coded-clean at entry and re-checked
// inside the registerIfOpen section — the re-check and the Register are ONE
// closingMu-guarded critical section (plan 07-01 WR-05, hardening the
// plan-05-10 WR-03 gate): no machine observer leak, no zombie registration.
func (f *Front) ResumeSession(ctx context.Context, sessionID, cwd string) (string, error) {
	if sessionID == "" {
		return "", acpport.NewInvalidParams("session/resume: sessionId must not be empty")
	}

	// WR-02 (plan 05-10): the spec's session-setup contract makes cwd
	// MUST-be-absolute — the empty case folds in, exactly like the
	// session/new T-4-01 guard. A rejected resume never touches the
	// backend wire.
	if !filepath.IsAbs(cwd) {
		return "", acpport.NewInvalidParams(
			"session/resume: cwd must be absolute (spec session-setup MUST), got %q", cwd)
	}

	// WR-03 (plan 05-10): the SESS-01 race gate at entry — a session
	// mid-teardown takes the resume as the clean coded rejection (the same
	// message family as the prompt gate), never a mid-close reattach
	// racing the release steps.
	if f.isClosing(sessionID) {
		return "", acpport.NewInvalidRequest("session/resume: %s: the session is closing or closed", sessionID)
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

	// WR-02 (plan 05-10): the client cwd anchors the reattached id the
	// moment createSessionID confirms it — the first post-resume turn
	// resolves absolute diff/location paths (acquireTurn's
	// sessionCwdLocked read).
	f.recordCwd(sid, cwd)

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

	// The machine registers on the conn's fan-out AFTER the tap — the same
	// flush-before-stopReason registration order NewSession relies on. The
	// construction stays OUTSIDE the close gate (its observer registrations
	// are conn-mutex work; only the close-gate section below is atomic).
	machine := turn.NewMachine(f.conn, sid, turn.NewPollWatcher(f.conn, turn.PollConfig{}))

	// WR-05 (plan 07-01): the mandatory WR-03 re-check, the reattach
	// release, and the Register are ONE closingMu-guarded section — a
	// resume that entered before close marked the set can never register
	// its machine into a session mid-teardown (the Register after close's
	// snapshot+Remove window is the observer-leak arm), and its reattach
	// release can never race the teardown's own CloseObservers on the same
	// machine.
	if !f.registerIfOpen(sid, machine) {
		return "", acpport.NewInvalidRequest("session/resume: %s: the session is closing or closed", sessionID)
	}

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
		// not dangle — the session-scoped sweep (plan 05-15 WR-07) cancels
		// exactly the closing session's pendings, never another session's
		// in-flight asks. No atomic take is needed here: the closing gate
		// predates the drain, so no same-session turn can register after
		// it — scoping contains the cross-session case.
		f.perm.cancelPendingForSession(sessionID)
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
	f.dropSettings(sessionID)

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

// registerIfOpen is the resume path's ONE closingMu-guarded section (WR-05,
// plan 07-01): the closing re-check, the in-process reattach's previous
// machine release, and the registry Register run under the SAME critical
// section, so a resume that survived the entry gate can never register its
// machine into a session mid-teardown — either the mark is already set (the
// caller answers the coded closing rejection and the registered machine is
// left untouched) or the machine registers before any later close can mark,
// and that close's snapshot-based teardown sees and releases it. The
// reattach release (Register is last-wins — the WR-07 discipline: without
// the release the previous machine's fan-out entries would leak) lives
// inside the section so it can never race the teardown's own CloseObservers
// on the same machine (a data race the WR-05 stress probe caught on the
// unguarded pre-check placement). The closing set is read inline under the
// held lock — no nested isClosing call — and Register runs only while the
// mutex is still held.
//
// Lock-order discipline (one-way nesting): closingMu -> registry lock and
// closingMu -> conn fan-out mutexes (CloseObservers' remove funcs) here; no
// path takes either and then closingMu. CloseSession releases closingMu
// before its registry Get/Remove and its teardown CloseObservers (its
// marking section holds closingMu alone — so a section-holder excludes it
// from marking until the section completes, and its CloseObservers either
// preceded the section's unmarked read or follows it), the registry's own
// methods never call back into the front, and the conn's delivery side
// invokes observers under the fan-out mutexes the remove funcs also take —
// none of those callback paths re-enter closingMu.
func (f *Front) registerIfOpen(sessionID string, machine *turn.Machine) bool {
	f.closingMu.Lock()
	defer f.closingMu.Unlock()

	if f.closing[sessionID] {
		return false
	}

	// In-process reattach (the registry still holds the id from an earlier
	// open): release the previous machine's observers before Register
	// replaces it.
	if prev, getErr := f.registry.Get(sessionID); getErr == nil && prev.Machine != nil {
		prev.Machine.CloseObservers()
	}

	f.registry.Register(core.Session{ID: sessionID, Machine: machine})

	return true
}
