package acpfront_test

// SESS-01 lifecycle locks (plan 05-05 Task 2, RED→GREEN over the
// session-lifecycle corpus row and the WR-04 orphan row):
//
//	TestResume: session/resume reattaches — the same id comes back through
//	  the create-shaped snapshot, the back leg shows resume → subscribe
//	  (re-issued, deliveryKind desktop-continuous) and NO setModel on the
//	  resume leg (model binding persists backend-side), a post-resume
//	  prompt resolves on the reattached machine, and the client receives
//	  NO session/update frames between the resume response and the next
//	  prompt (no replay — the D-04 prohibition: the wrapper never
//	  synthesizes pre-resume history).
//	TestCloseTeardown: close runs the ordered teardown — cancel any active
//	  turn first (session/stop precedes session/close on the wire), the
//	  registry entry is gone (a follow-up prompt fails the coded
//	  ErrNoSession-class rejection and the back leg shows no session/send
//	  for it), both conn observer counts return to baseline (the plan
//	  05-03 exposed removes — WR-07), and a second close is a clean coded
//	  rejection that never touches the wire (idempotent teardown, never a
//	  panic, never a zombie session).
//	TestCloseWhileTurnActiveRace: close cancels the active turn first and
//	  waits bounded; a close racing a second prompt lands exactly one
//	  clean outcome (the coded rejection or the cancelled resolution) —
//	  never a hang, never a resolve-after-removal, never a third outcome
//	  frame on the wire (the machine's resolved-exactly-once semantics at
//	  the composed tier).
//	TestNewSessionOrphanClosed (WR-04): a session/new whose subscribe
//	  fails after create succeeded closes the backend session best-effort
//	  (session/close on the back leg) and answers the coded error — no
//	  error-path orphan (T-5-12).
//
// All tests ride the plan 04-02 harness as composed by newTracerFront
// (harness_test.go) over the plan 05-05 corpus rows.

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// newLifecycleSession opens one session over the composed loop on a fresh
// temp cwd and returns the id (the cwd is client-supplied; the front
// validates it absolute — the T-4-01 guard).
func newLifecycleSession(ctx context.Context, t *testing.T, h *FrontHarness) string {
	t.Helper()

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	return string(newResp.SessionId)
}

// lifecyclePrompt builds one prompt request for a lifecycle test.
func lifecyclePrompt(sid, text string) acp.PromptRequest {
	return acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock(text)},
	}
}

// assertCodedInvalidRequest asserts one coded JSON-RPC error in the
// invalid-request class (the ErrNoSession/concurrency rejection surface)
// without pinning the message text.
func assertCodedInvalidRequest(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected a rejection, got success")
	}

	var reqErr *acp.RequestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("rejection error = %T (%v), want a coded JSON-RPC RequestError", err, err)
	}

	if reqErr.Code != acpport.CodeInvalidRequest {
		t.Fatalf("rejection code = %d, want %d (invalid-request class); message %q",
			reqErr.Code, acpport.CodeInvalidRequest, reqErr.Message)
	}
}

// mustInitialize performs the composed loop's initialize leg.
func mustInitialize(ctx context.Context, t *testing.T, h *FrontHarness) {
	t.Helper()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

// TestResume locks the SESS-01 resume contract over the session-lifecycle
// corpus: reattach returns the same id through the create-shaped snapshot,
// the wire order is resume → re-issued subscribe with no setModel on the
// leg, the post-resume prompt resolves on the reattached machine, and
// nothing replays between the resume response and the next prompt.
func TestResume(t *testing.T) {
	h := newTracerFront(t, "session-lifecycle")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid := newLifecycleSession(ctx, t, h)

	if _, err := h.client.ResumeSession(ctx, acp.ResumeSessionRequest{
		SessionId: acp.SessionId(sid),
		Cwd:       t.TempDir(),
	}); err != nil {
		t.Fatalf("session/resume: %v", err)
	}

	// No replay (D-04): the client collected ZERO updates between the
	// resume response and the next prompt — the wrapper never synthesizes
	// pre-resume history.
	if updates := h.Updates(); len(updates) != 0 {
		t.Fatalf("client collected %d updates after resume, want 0 (no replay of pre-resume history)",
			len(updates))
	}

	// The post-resume prompt resolves on the reattached machine (send rule
	// 1: the completing shape) — the registry entry + fresh machine work.
	promptResp, err := h.client.Prompt(ctx, lifecyclePrompt(sid, "after resume"))
	if err != nil {
		t.Fatalf("post-resume prompt: %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("post-resume stopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}

	assertCollectedUpdates(t, h, sid)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertResumeLegOrder(t, tr, sid)
}

// assertResumeLegOrder locks the resume leg's back-leg wire shape: exactly
// one session/resume, exactly one subscribe AFTER it (the re-issue,
// desktop-continuous for the reattached session), and NO setModel after
// the resume (model binding persists backend-side).
func assertResumeLegOrder(t *testing.T, tr fake.Transcript, sid string) {
	t.Helper()

	resumeIdxs := transcriptIndexes(tr, "in", `"method":"session/resume"`)
	if len(resumeIdxs) != 1 {
		t.Fatalf("back leg records %d session/resume frames, want exactly 1", len(resumeIdxs))
	}

	resumeIdx := resumeIdxs[0]

	var afterResume []int
	for _, idx := range transcriptIndexes(tr, "in", `"method":"session/subscribe"`) {
		if idx > resumeIdx {
			afterResume = append(afterResume, idx)
		}
	}

	if len(afterResume) != 1 {
		t.Fatalf("resume leg re-issued %d subscribes, want exactly 1 (resume at %d)",
			len(afterResume), resumeIdx)
	}

	assertSubscribeFrame(t, tr.Lines[afterResume[0]].JSON, sid)

	setModels := transcriptIndexes(tr, "in", `"method":"session/setModel"`)
	if len(setModels) != 1 || setModels[0] > resumeIdx {
		t.Fatalf("resume leg carries setModel frames after the resume (setModels at %v, resume at %d) — "+
			"model binding persists backend-side, the resume leg must not re-bind",
			setModels, resumeIdx)
	}
}

// lifecycleStage is the close-probe prelude: capture the observer
// baseline, open the session (+1/+1 on both fan-outs), run the completing
// stage turn (consumes send rule 1), and start the ACTIVE turn (send rule
// 2 — no terminal until cancel). Returns the session id, the baseline,
// and the active prompt's resolution channel.
func lifecycleStage(ctx context.Context, t *testing.T, h *FrontHarness) (string, int, int, <-chan promptOutcome) {
	t.Helper()

	notifBase, routerBase := h.conn.ObserverCounts()

	sid := newLifecycleSession(ctx, t, h)

	if notif, router := h.conn.ObserverCounts(); notif != notifBase+1 || router != routerBase+1 {
		t.Fatalf("post-open observer counts = (%d, %d), want baseline+1 (%d, %d)",
			notif, router, notifBase, routerBase)
	}

	// Stage 1: a completing turn consumes send rule 1 (times 1) — the
	// machine resolved end_turn before any teardown.
	if _, err := h.client.Prompt(ctx, lifecyclePrompt(sid, "complete first")); err != nil {
		t.Fatalf("stage prompt: %v", err)
	}

	// Stage 2: an ACTIVE turn (send rule 2 — no terminal until cancel).
	promptDone := startActiveTurn(ctx, t, h, sid)

	return sid, notifBase, routerBase, promptDone
}

// assertTeardownWire locks the post-close back-leg shape: the active
// turn's cancel (session/stop) precedes the ONE backend session/close,
// and the send count matches what legitimately ran.
func assertTeardownWire(t *testing.T, tr fake.Transcript, wantSends int) {
	t.Helper()

	stopIdx := transcriptIndex(t, tr, "in", `"method":"session/stop"`, "stop request")
	closeIdxs := transcriptIndexes(tr, "in", `"method":"session/close"`)

	if len(closeIdxs) != 1 {
		t.Fatalf("back leg records %d session/close frames, want exactly 1 (repeats never touch the wire)",
			len(closeIdxs))
	}

	if stopIdx > closeIdxs[0] {
		t.Fatalf("wire order violated: session/stop at %d must precede session/close at %d", stopIdx, closeIdxs[0])
	}

	if sends := len(transcriptIndexes(tr, "in", `"method":"session/send"`)); sends != wantSends {
		t.Fatalf("back leg records %d session/send frames, want exactly %d (rejected prompts never reach the wire)",
			sends, wantSends)
	}
}

// TestCloseTeardown locks the SESS-01 close contract: the ordered
// teardown (cancel first, then the backend close), the registry release
// with clean post-close rejections, the observer-count baseline (WR-07),
// and the idempotent second close.
func TestCloseTeardown(t *testing.T) {
	h := newTracerFront(t, "session-lifecycle")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid, notifBase, routerBase, promptDone := lifecycleStage(ctx, t, h)

	// Close while the turn runs: cancel fires first, the bounded wait
	// drains the turn, THEN the backend close lands.
	if _, err := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("session/close: %v", err)
	}

	outcome := waitPromptOutcome(t, promptDone)
	if outcome.err != nil {
		t.Fatalf("cancelled-by-close prompt must answer as a success response, got error: %v", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stopReason = %q, want %q", outcome.resp.StopReason, acp.StopReasonCancelled)
	}

	// The registry entry is gone: a follow-up prompt fails the coded
	// ErrNoSession-class rejection — never a panic, never a zombie.
	_, err := h.client.Prompt(ctx, lifecyclePrompt(sid, "post-close"))
	assertCodedInvalidRequest(t, err)

	// The second close is a clean coded rejection (idempotent teardown —
	// the Get gate answers before anything touches the wire again).
	_, err = h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)})
	assertCodedInvalidRequest(t, err)

	// Both conn observer counts return to baseline (WR-07: the plan 05-03
	// exposed removes — CloseObservers drives both fan-outs).
	if notif, router := h.conn.ObserverCounts(); notif != notifBase || router != routerBase {
		t.Fatalf("post-close observer counts = (%d, %d), want baseline (%d, %d)",
			notif, router, notifBase, routerBase)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// Exactly the stage-1 and stage-2 sends: the rejected post-close
	// prompt never touched the back leg.
	assertTeardownWire(t, tr, 2)
}

// TestCloseWhileTurnActiveRace is the T-5-12 concurrency probe: close
// cancels the active turn first and waits bounded; a second prompt racing
// the close lands exactly ONE clean outcome — the coded rejection
// (closing/closed/turn-active) or its own cancelled resolution — never a
// hang (both waiters bound), never a resolve-after-removal (the outcome
// frame count assert).
func TestCloseWhileTurnActiveRace(t *testing.T) {
	h := newTracerFront(t, "session-lifecycle")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid, _, _, promptDone := lifecycleStage(ctx, t, h)

	// The race: a second prompt fired WHILE close tears the session down.
	raceDone := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, lifecyclePrompt(sid, "racing"))
		raceDone <- promptOutcome{resp: resp, err: err}
	}()

	if _, err := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("session/close: %v", err)
	}

	// The active turn resolves cancelled exactly once — the close's cancel
	// fired first and the bounded drain waited it out.
	outcome := waitPromptOutcome(t, promptDone)
	if outcome.err != nil || outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("active turn outcome = (%v, %v), want a cancelled success",
			outcome.resp.StopReason, outcome.err)
	}

	assertRaceOutcome(t, raceDone)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// Exactly-once at the composed tier: three prompts ran (stage, active,
	// racing) — the wire carries exactly three outcome frames (stopReason
	// responses or error rejections), never a duplicate or a lost
	// resolution (a machine resolve after removal would show up as a
	// missing or duplicated frame here).
	outcomes := len(transcriptIndexes(tr, "in", `"stopReason"`)) +
		len(transcriptIndexes(tr, "in", `"error"`))
	if want := 3; outcomes != want {
		t.Fatalf("wire records %d outcome frames for 3 prompts, want exactly %d (resolved exactly once, never after removal)",
			outcomes, want)
	}

	assertTeardownWire(t, tr, 3)
}

// assertRaceOutcome requires the racing prompt's exactly-one clean
// outcome: the coded rejection or the cancelled resolution.
func assertRaceOutcome(t *testing.T, raceDone <-chan promptOutcome) {
	t.Helper()

	race := waitPromptOutcome(t, raceDone)

	switch {
	case race.err != nil:
		assertCodedInvalidRequest(t, race.err)
	case race.resp.StopReason == acp.StopReasonCancelled:
	default:
		t.Fatalf("racing prompt outcome = (%v, %v), want the cancelled success or the coded rejection",
			race.resp.StopReason, race.err)
	}
}

// TestNewSessionOrphanClosed locks WR-04 over the orphan corpus row
// (newsession-orphan-close): session/create succeeds, session/subscribe
// refuses (times 1) — NewSession answers the coded internal error AND the
// just-created backend session is closed best-effort on the back leg (no
// error-path orphan, T-5-12).
func TestNewSessionOrphanClosed(t *testing.T) {
	var front *acpfront.Front

	h := ComposeFront(t, mustLoadScenario(t, "newsession-orphan-close"), func(conn *zcode.Conn) acp.Agent {
		// No model configured: the failure point is the subscribe leg
		// (the create already succeeded — the orphan exists).
		front = acpfront.New(conn)

		return acpport.NewAgent(front)
	})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid, err := front.NewSession(ctx, acpport.NewSessionParams{Cwd: t.TempDir()})
	if err == nil {
		t.Fatalf("session/new must fail when the subscribe leg refuses, got session %q", sid)
	}

	var seamErr *acpport.Error
	if !errors.As(err, &seamErr) {
		t.Fatalf("session/new error = %T (%v), want a coded seam error", err, err)
	}

	if seamErr.Code != acpport.CodeInternal {
		t.Fatalf("session/new error code = %d, want %d (the backend refusal surfaced honestly)",
			seamErr.Code, acpport.CodeInternal)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertOrphanClosed(t, tr)
}

// assertOrphanClosed asserts the WR-04 back-leg shape: exactly one
// session/close, after the refused subscribe, carrying the created
// session id.
func assertOrphanClosed(t *testing.T, tr fake.Transcript) {
	t.Helper()

	subIdx := transcriptIndex(t, tr, "in", `"method":"session/subscribe"`, "refused subscribe")
	closeIdxs := transcriptIndexes(tr, "in", `"method":"session/close"`)

	if len(closeIdxs) != 1 {
		t.Fatalf("back leg records %d session/close frames, want exactly 1 (WR-04: the error-path orphan closes)",
			len(closeIdxs))
	}

	if closeIdxs[0] < subIdx {
		t.Fatalf("orphan close at %d precedes the refused subscribe at %d", closeIdxs[0], subIdx)
	}

	var frame struct {
		Params struct {
			SessionID string `json:"sessionId"`
		} `json:"params"`
	}

	raw := tr.Lines[closeIdxs[0]].JSON
	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode close frame: %v (%s)", err, raw)
	}

	if frame.Params.SessionID != "sess_fix1" {
		t.Fatalf("orphan close sessionId = %q, want the created session (sess_fix1): %s",
			frame.Params.SessionID, raw)
	}
}
