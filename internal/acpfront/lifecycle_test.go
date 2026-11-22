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
	"path/filepath"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/interact"
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

// assertCodedInvalidParams asserts one coded JSON-RPC error in the
// invalid-params class (the -32602 session-setup MUST rejections) without
// pinning the message text.
func assertCodedInvalidParams(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected a rejection, got success")
	}

	var reqErr *acp.RequestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("rejection error = %T (%v), want a coded JSON-RPC RequestError", err, err)
	}

	if reqErr.Code != acpport.CodeInvalidParams {
		t.Fatalf("rejection code = %d, want %d (invalid-params class); message %q",
			reqErr.Code, acpport.CodeInvalidParams, reqErr.Message)
	}
}

// TestResumeRecordsCwd locks WR-02 (plan 05-10): a resume carrying an
// absolute Cwd records it for the reattached id, so the resumed session's
// FIRST turn resolves an ABSOLUTE diff/location path in the tool card (the
// Pitfall-5 shape on the resume path). The client cwd rides the SDK
// ResumeSessionRequest.Cwd; at HEAD the adapter forwards only the session
// id and the front never records the cwd, so the post-resume Edit card
// renders the backend's relative path verbatim.
func TestResumeRecordsCwd(t *testing.T) {
	h := newPermissionFrontOpts(t, "full-cycle", PermissionScript{SelectedOptionID: permAllowOptionID}, nil)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	openCwd := t.TempDir()

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: openCwd})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	sid := string(newResp.SessionId)

	// The resume re-cwds the session: the post-resume turn must resolve
	// against the RESUMED cwd, not the open one.
	resumeCwd := t.TempDir()

	if _, err := h.client.ResumeSession(ctx, acp.ResumeSessionRequest{
		SessionId: acp.SessionId(sid),
		Cwd:       resumeCwd,
	}); err != nil {
		t.Fatalf("session/resume: %v", err)
	}

	// The first post-resume turn runs the corpus row's tool legs: the
	// scheduled Edit card must carry the diff path RESOLVED against the
	// resumed cwd (absolute — Pitfall 5).
	if _, err := h.client.Prompt(ctx, lifecyclePrompt(sid, "after resume")); err != nil {
		t.Fatalf("post-resume prompt: %v", err)
	}

	var start *acp.SessionUpdateToolCall
	for _, u := range h.Updates() {
		if u.Update.ToolCall != nil && u.Update.ToolCall.ToolCallId == acp.ToolCallId("tc1") {
			start = u.Update.ToolCall
		}
	}

	if start == nil {
		t.Fatal("the post-resume turn streamed no tool_call start for tc1")
	}

	wantPath := filepath.Join(resumeCwd, "src", "main.go")

	if len(start.Content) == 0 || start.Content[0].Diff == nil {
		t.Fatalf("the tc1 start carries no diff content: %+v", start)
	}

	if got := start.Content[0].Diff.Path; got != wantPath {
		t.Fatalf("post-resume edit diff path = %q, want %q (absolute, resolved against the RESUMED cwd — "+
			"the resume must record the client cwd)", got, wantPath)
	}
}

// resumeSettingsScenario composes the WR-01 pure-resume harness:
// cfgOptionScenario's create rule never fires (no session/new is driven);
// the settings snapshot rides the resume reply verbatim (create-shaped)
// and the setModel rule answers the picker/slash mutation.
func resumeSettingsScenario() *fake.Scenario {
	return cfgOptionScenario(cfgCreateResult,
		fake.RequestRule{
			Method: zcode.MethodSessionResume,
			Reply:  &fake.Reply{Result: json.RawMessage(cfgCreateResult)},
		},
		fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
		})
}

// resumeSetCfgSession runs the initialize + PURE resume preamble (no
// session/new — the snapshot must come from the resume reply alone) and
// returns the resumed session id.
func resumeSetCfgSession(ctx context.Context, t *testing.T, h *FrontHarness) string {
	t.Helper()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	if _, err := h.client.ResumeSession(ctx, acp.ResumeSessionRequest{
		SessionId: acp.SessionId("sess_cfg"),
		Cwd:       t.TempDir(),
	}); err != nil {
		t.Fatalf("session/resume: %v", err)
	}

	return "sess_cfg"
}

// TestResumeRecordsConfirmedSettings locks WR-01 (07-REVIEW): the resume
// reply IS a confirmed settings snapshot — createSessionID's own contract
// states session/create and session/resume carry the SAME full snapshot
// shape, and D-10 names the reply the confirmation source — so
// ResumeSession must record it beside the cwd. Without the record every
// resumed session answers the picker's model arm the misleading -32602
// "no confirmed settings (unknown session)" and the /model slash the
// pinned "No model selected." refusal despite a backend-confirmed
// binding. The harness carries NO session/new leg: the snapshot arrives
// ONLY through the resume reply, so a pass proves the resume path
// recorded it.
func TestResumeRecordsConfirmedSettings(t *testing.T) {
	t.Run("the picker's model arm validates against the resume reply", func(t *testing.T) {
		h := newSetCfgFront(t, resumeSettingsScenario())

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := resumeSetCfgSession(ctx, t, h)

		// The ask targets a value the RESUME reply lists: success proves the
		// store recorded the resume snapshot; the -32602 unknown-session
		// answer is the WR-01 bug this test pins.
		ask := setCfgAsk(sid, optionIDModel, corpusModelFlashID)

		if _, err := h.client.SetSessionConfigOption(ctx, ask); err != nil {
			t.Fatalf("the resumed session must validate the picker ask against the "+
				"resume reply's confirmed snapshot (the reply IS the D-10 source): %v", err)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// The mutation payload byte-locks the picker arm's shape over the
		// resumed session: the listed target ids and the RESUME reply's
		// confirmed level.
		assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), corpusProviderID, corpusModelID, reasoningLevelLow)
	})

	t.Run("the /model slash resolves the resume reply's binding", func(t *testing.T) {
		h := newSetCfgFront(t, resumeSettingsScenario())

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := resumeSetCfgSession(ctx, t, h)

		resp, err := h.client.Prompt(ctx, slashPrompt(sid, "/model "+corpusModelFlashID))
		if err != nil {
			t.Fatalf("the /model switch on a resumed session must resolve against the "+
				"resume reply's confirmed binding (never the pinned no-model refusal): %v", err)
		}

		if resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("stopReason = %q, want end_turn", resp.StopReason)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// The intercept consumed the command: exactly the ONE setModel
		// mutation, zero literal sends.
		assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), corpusProviderID, corpusModelID, reasoningLevelLow)

		if sends := transcriptIndexes(tr, "in", `"method":"session/send"`); len(sends) != 0 {
			t.Fatalf("the intercept must consume the /model prompt (the literal text never "+
				"reaches the backend) — the back leg records %d session/send frame(s)", len(sends))
		}
	})
}

// TestResumeRejectsRelativeCwd locks WR-02's validation arm (plan 05-10):
// a non-absolute (or empty) resume cwd answers the coded invalid-params
// rejection BEFORE any backend call — a rejected resume never touches the
// wire (the T-4-01 parity arm; the SDK contract makes resume cwd
// MUST-be-absolute exactly like session/new's).
func TestResumeRejectsRelativeCwd(t *testing.T) {
	h := newTracerFront(t, "session-lifecycle")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid := newLifecycleSession(ctx, t, h)

	for name, cwd := range map[string]string{
		"relative": "relative/path",
		"empty":    "",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := h.client.ResumeSession(ctx, acp.ResumeSessionRequest{
				SessionId: acp.SessionId(sid),
				Cwd:       cwd,
			})
			assertCodedInvalidParams(t, err)
		})
	}

	// The parity arm on the back leg: neither rejected resume touched the
	// backend wire — no session/resume frame at all.
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if idxs := transcriptIndexes(tr, "in", `"method":"session/resume"`); len(idxs) != 0 {
		t.Fatalf("back leg records %d session/resume frames for rejected resumes, want 0 "+
			"(the absolute-cwd validation must precede any backend call)", len(idxs))
	}
}

// TestResumeRejectedWhileClosing locks WR-03 (plan 05-10): session/resume
// racing a close lands exactly one coded-clean rejection — the same
// closing/closed family the prompt gate answers — and no new machine's
// observers register (the leak arm: a Register after close's snapshot+
// Remove would strand the new machine's fan-out entries). The shrunk
// WithCloseTurnWait holds the closing window open deterministically (the
// active turn outlives the deadline — the completion-quiet-window row has
// no stop rule, so the close parks in the drain for the whole wait).
func TestResumeRejectedWhileClosing(t *testing.T) {
	sc := mustLoadScenario(t, "completion-quiet-window")

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
			acpfront.WithCloseTurnWait(300*time.Millisecond), // hold the closing window open
		)

		return acpport.NewAgent(front)
	})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid := newLifecycleSession(ctx, t, h)

	// The active turn parks the close in its drain for the whole wait —
	// the closing set stays marked across the window.
	promptDone := startActiveTurn(ctx, t, h, sid)
	_ = promptDone // resolves cancelled well after this test's asserts

	notifPre, routerPre := h.conn.ObserverCounts()

	closeDone := make(chan error, 1)

	go func() {
		_, err := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)})
		closeDone <- err
	}()

	// Land the resume INSIDE the closing window (the close marked the set
	// within microseconds of starting; its teardown spans >= the 600ms of
	// drain deadlines).
	time.Sleep(100 * time.Millisecond)

	_, err := h.client.ResumeSession(ctx, acp.ResumeSessionRequest{
		SessionId: acp.SessionId(sid),
		Cwd:       t.TempDir(),
	})
	assertCodedInvalidRequest(t, err) // the closing/closed rejection — same family as the prompt gate

	if closeErr := <-closeDone; closeErr != nil {
		t.Fatalf("session/close: %v", closeErr)
	}

	// WR-03's leak arm: exactly the machine's entries removed — a mid-close
	// resume must never have registered a machine whose observers close's
	// snapshot-based teardown cannot see.
	if notif, router := h.conn.ObserverCounts(); notif != notifPre-1 || router != routerPre-1 {
		t.Fatalf("post-close observer counts = (%d, %d), want exactly the machine's entries removed (%d, %d) — "+
			"a resume racing the close registered a zombie machine", notif, router, notifPre-1, routerPre-1)
	}

	// The registry entry is released: a post-close prompt fails the coded
	// ErrNoSession-class rejection — never a zombie session.
	_, err = h.client.Prompt(ctx, lifecyclePrompt(sid, "post-close"))
	assertCodedInvalidRequest(t, err)
}

// lifecycleStage is the close-probe prelude: open the session (+1/+1 on
// both fan-outs over the tap-only baseline), run the completing stage
// turn (consumes send rule 1), and start the ACTIVE turn (send rule 2 —
// no terminal until cancel). Returns the session id and the active
// prompt's resolution channel; the close tests capture their own
// pre-close observer snapshot (the WR-07 lock asserts exactly the
// machine's (1,1) removed).
func lifecycleStage(
	ctx context.Context, t *testing.T, h *FrontHarness,
) (sid string, promptDone <-chan promptOutcome) {
	t.Helper()

	notifTap, routerTap := h.conn.ObserverCounts()

	sid = newLifecycleSession(ctx, t, h)

	notifOpen, routerOpen := h.conn.ObserverCounts()
	if notifOpen != notifTap+1 || routerOpen != routerTap+1 {
		t.Fatalf("post-open observer counts = (%d, %d), want tap+1 (%d, %d)",
			notifOpen, routerOpen, notifTap, routerTap)
	}

	// Stage 1: a completing turn consumes send rule 1 (times 1) — the
	// machine resolved end_turn before any teardown.
	if _, err := h.client.Prompt(ctx, lifecyclePrompt(sid, "complete first")); err != nil {
		t.Fatalf("stage prompt: %v", err)
	}

	// Stage 2: an ACTIVE turn (send rule 2 — no terminal until cancel).
	promptDone = startActiveTurn(ctx, t, h, sid)

	return sid, promptDone
}

// assertTeardownWire locks the post-close back-leg shape: the active
// turn's cancel (session/stop) precedes the ONE backend session/close,
// and the send count matches what legitimately ran.
func assertTeardownWire(t *testing.T, tr fake.Transcript, wantSends int) {
	t.Helper()

	assertCloseAfterStop(t, tr)

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

	sid, promptDone := lifecycleStage(ctx, t, h)

	// The pre-close observer snapshot: everything registered so far (the
	// tap, the machine, the stage's own turn.started observer).
	notifPre, routerPre := h.conn.ObserverCounts()

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

	// Exactly the machine's (notification + router) entries removed — the
	// WR-07 lock: the plan 05-03 exposed CloseObservers drives both
	// fan-outs, and nothing else leaks or lingers.
	if notif, router := h.conn.ObserverCounts(); notif != notifPre-1 || router != routerPre-1 {
		t.Fatalf("post-close observer counts = (%d, %d), want exactly the machine's entries removed (%d, %d)",
			notif, router, notifPre-1, routerPre-1)
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

	sid, promptDone := lifecycleStage(ctx, t, h)

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
	// racing) — the ACP tee carries exactly three outcome frames (the
	// prompt responses' stopReason or the rejections' error), never a
	// duplicate or a lost resolution (a machine resolve after removal
	// would show up as a missing or duplicated frame here).
	lines := h.ACPLines()
	outcomes := len(teeLineIndexes(lines, "in", `"stopReason"`)) +
		len(teeLineIndexes(lines, "in", `"error"`))
	if want := 3; outcomes != want {
		t.Fatalf("ACP tee records %d outcome frames for 3 prompts, want exactly %d "+
			"(resolved exactly once, never after removal)", outcomes, want)
	}

	// The racing prompt lands one clean outcome BEFORE any send (the
	// common arm: the slot/closing gate rejects it) or after running (the
	// drain cancels it) — so the back leg carries the stage + active sends
	// plus at most the racing send.
	sends := len(transcriptIndexes(tr, "in", `"method":"session/send"`))
	if sends != 2 && sends != 3 {
		t.Fatalf("back leg records %d session/send frames, want 2 (racer rejected pre-send) "+
			"or 3 (racer cancelled mid-turn)", sends)
	}

	// The teardown wire shape with the observed racer arm: the stop
	// precedes the ONE close; the racer's send, when it ran, precedes them.
	assertCloseAfterStop(t, tr)
}

// TestCloseWithPendingPermissionCancelled is the CR-01 lock (plan 05-10,
// gap 2 of 05-VERIFICATION.md): a session whose active turn holds a pending
// interaction/requestPermission bridge — the client parks on the answer
// gate — is closed mid-ask. The CORE-06 close-path truth demands the fan-in:
// the pending table drains (the bridge's round-trip classifies cancelled)
// and the backend observes the -32021 cancelled reply on the transcript —
// a permission bridge never outlives its turn on the close path (the 05-02
// prohibition). At HEAD the drain reports false after a successful drain,
// so CloseSession skips cancelPendingForTurn exactly when a turn was
// cancelled, and the pending dangles to the responder deadline instead.
func TestCloseWithPendingPermissionCancelled(t *testing.T) {
	gate := make(chan struct{})

	h := newPermissionFrontOpts(t, "permission-cancel", PermissionScript{Gate: gate}, nil)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid := openPermissionSession(ctx, t, h)

	promptDone := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		promptDone <- promptOutcome{resp: resp, err: err}
	}()

	// The ask is pending: the client captured it and parks on the gate —
	// the bridge's round-trip holds until the turn's cancel fan-in (or the
	// responder deadline) resolves it.
	awaitPermissionAsk(t, h)

	// Close while the ask is pending: the ordered teardown cancels the
	// active turn first, and the cancelled turn's permission bridge must
	// fan in with it (the -32021 cancelled arm — the disposition ledger's
	// REAL RUN gate).
	if _, err := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("session/close: %v", err)
	}

	// The cancelled turn resolves as the SUCCESS cancelled stopReason (the
	// close's machine.Cancel landed; the prompt never hangs).
	outcome := waitPromptOutcome(t, promptDone)
	if outcome.err != nil {
		t.Fatalf("cancelled-by-close prompt must answer as a success response, got error: %v", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stopReason = %q, want %q", outcome.resp.StopReason, acp.StopReasonCancelled)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// The close-path artifact: the pending reverse earned EXACTLY ONE reply
	// and it is the -32021 cancelled error (the bridge's pending table
	// drained — the fan-in fired), with the teardown wire order intact
	// (stop precedes the ONE backend close).
	assertCloseWithPendingBackLeg(t, tr)
}

// assertCloseWithPendingBackLeg locks the CR-01 close-path shape on the
// fake transcript: the registering permission reverse earned exactly one
// reply, that reply is the -32021 cancelled error carrying the bridge's
// stable message (the corpus row's expect_answer byte-locks both), and the
// teardown order holds (the turn's session/stop precedes the ONE backend
// session/close — the close cancelled the turn, not skipped it).
func assertCloseWithPendingBackLeg(t *testing.T, tr fake.Transcript) {
	t.Helper()

	errReplies := 0

	for _, line := range tr.Lines {
		probe := probeLineJSON(line.JSON)

		if line.Dir != fake.DirIn || !isReplyTo(probe, permReverseID) {
			continue
		}

		if probe["error"] == nil {
			t.Fatalf("close-path reverse earned a result reply (%s), want the -32021 cancelled error "+
				"(a deny after the turn was cancelled is the dangling-bridge shape CR-01 forbids)", line.JSON)
		}

		errReplies++
		assertCancelledError(t, probe, line.JSON)
	}

	if errReplies != 1 {
		t.Fatalf("close-path permission reverse earned %d replies, want exactly 1 (-32021 cancelled — "+
			"the close must fan the pending bridge in with its turn)", errReplies)
	}

	assertCloseAfterStop(t, tr)
}

// TestCloseDeadlineArmFansPendingIn locks the deadline arm of the fixed
// drainActiveTurn contract (plan 05-10): a turn still active past the close
// wait reports sawTurn=true (a turn WAS observed — the review's fix shape
// returns true on both arms) and the permission fan-in still fires
// best-effort — the -32021 cancelled reply reaches the backend even though
// the teardown proceeded before the turn fully unwound. The shrunk
// WithCloseTurnWait (1ns) makes the deadline arm deterministic: the first
// drain iteration finds the active turn, cancels, and the elapsed deadline
// returns true before any unwind could win the race.
func TestCloseDeadlineArmFansPendingIn(t *testing.T) {
	gate := make(chan struct{})

	sc := mustLoadScenario(t, "permission-cancel")

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
			acpfront.WithCloseTurnWait(time.Nanosecond),
		)

		opts := []interact.Option{
			interact.WithPermissionForwarder(interact.NewPermissionBridge(front.PermissionForward()).Forward),
		}

		responder := interact.NewResponder(opts...)
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	h.rec.SetPermissionScript(PermissionScript{Gate: gate})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid := openPermissionSession(ctx, t, h)

	promptDone := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		promptDone <- promptOutcome{resp: resp, err: err}
	}()

	awaitPermissionAsk(t, h)

	if _, err := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("session/close: %v", err)
	}

	// The deadline-arm close still cancels the turn (best-effort): the
	// machine cancel landed and the prompt resolves cancelled as a SUCCESS.
	outcome := waitPromptOutcome(t, promptDone)

	if outcome.err != nil || outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("deadline-arm close outcome = (%v, %v), want the cancelled success "+
			"(the deadline arm still cancels the turn)", outcome.resp.StopReason, outcome.err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// The same close-path artifact as the successful-drain arm: the fan-in
	// fired best-effort past the deadline — the -32021 cancelled reply with
	// the teardown wire order intact.
	assertCloseWithPendingBackLeg(t, tr)
}

// assertCloseAfterStop locks the teardown wire order: exactly one
// session/close, preceded by exactly one session/stop.
func assertCloseAfterStop(t *testing.T, tr fake.Transcript) {
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

	sid, _, err := front.NewSession(ctx, acpport.NewSessionParams{Cwd: t.TempDir()})
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
