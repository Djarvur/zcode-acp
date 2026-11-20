package acpfront

import (
	"context"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// SESS-01 session lifecycle (plan 05-05) — Task-1 compile-presence stage.
// The neutral acpport.Handlers vocabulary widened with ResumeSession/
// CloseSession (the adapter's unadvertised -32601 stubs became real
// delegates in the same commit), which makes the Front the compile-time
// implementor of both methods. The bodies below are the honest
// not-available answer — the established Authenticate posture (agent.go):
// the wire surface routes and the adapter delegates, the function answers
// a coded error instead of a fake success, and no backend frame ever
// leaves for these methods yet (a rejected lifecycle call never touches
// the wire — T-5-13).
//
// Plan 05-05 Task 2 replaces these bodies with the real contracts
// (RED→GREEN, locked by TestResume/TestCloseTeardown over the
// session-lifecycle corpus row):
//   - ResumeSession: backend session/resume (the reply is the SAME full
//     snapshot shape as create — createSessionID digs result.session.sessionId
//     unchanged) → RE-ISSUED session/subscribe {deliveryKind:desktop-continuous}
//     → machine + registry registration → id returned. NO setModel follow-up
//     (model binding persists backend-side) and NO replay — the wrapper never
//     synthesizes updates for pre-resume history (D-04 prohibition).
//   - CloseSession: the ordered teardown — cancel any active turn
//     (machine.Cancel; bounded wait) → best-effort backend session/close →
//     registry Remove → both observer removes (the plan-05-03 exposed
//     Machine.CloseObservers) → reply. A post-close prompt fails cleanly
//     with ErrNoSession (never a panic, never a zombie session).

// ResumeSession reattaches one backend session (SESS-01). Task-1 stage:
// the honest not-available answer (see the file comment).
func (f *Front) ResumeSession(ctx context.Context, sessionID string) (string, error) {
	_ = ctx
	_ = sessionID

	return "", acpport.NewInternal(
		"session/resume is not available yet (SESS-01 lifecycle lands with plan 05-05 Task 2)")
}

// CloseSession tears one session down (SESS-01). Task-1 stage: the honest
// not-available answer (see the file comment).
func (f *Front) CloseSession(ctx context.Context, sessionID string) error {
	_ = ctx
	_ = sessionID

	return acpport.NewInternal(
		"session/close is not available yet (SESS-01 lifecycle lands with plan 05-05 Task 2)")
}
