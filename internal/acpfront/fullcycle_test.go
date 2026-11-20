package acpfront_test

// TestFullCycle is the plan 05-08 composed E2E — success criterion 5's
// composed-tier rehearsal: authenticate (read-back proven) → session/new →
// prompt with streamed tool cards AND a permission ask bridged to the
// client's decision → end_turn → session/resume (no replay) → session/close
// (teardown), as ONE continuous composition over ONE harness, every leg
// asserted on recorded wire artifacts (the ACP tee, the RecordingClient
// drain, the fake transcript — never internals).
//
// The composition rides the full-cycle corpus row; the per-leg contracts
// are the prior plans' locks (05-02 permission bridge, 05-04 tool stream +
// pump, 05-05 resume/close, 05-06/07 auth chain) — this test proves they
// compose, with the ordering interactions the phase's truths pin:
// the permission ask lands BETWEEN the started and progress tool frames,
// the end_turn response lands only after every tool frame (flush-before-
// stopReason through the pump), nothing streams after the response, the
// resume leg replays nothing, and the close teardown releases exactly the
// session machine's observers before rejecting a post-close prompt clean.

import (
	"context"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Tee-frame probe substrings for the composed-tier placement asserts (the
// SDK's sessionUpdate discriminants — the exact `"tool_call"` value is
// distinct from `"tool_call_update"` by its closing quote).
const (
	teeToolStart  = `"sessionUpdate":"tool_call"`
	teeToolPatch  = `"sessionUpdate":"tool_call_update"`
	teePermFrame  = `"session/request_permission"`
	teeUpdateMark = `"sessionUpdate"`
	teeInProgress = `"in_progress"`
)

// TestFullCycle runs the composed cycle end-to-end under -race: one
// harness, one continuous client story, legs in exactly the order the real
// client plays them.
func TestFullCycle(t *testing.T) {
	h := newFullCycleFront(t)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	fullCycleAuthenticate(ctx, t, h)

	sid := fullCycleOpenSession(ctx, t, h)

	fullCyclePrompt(ctx, t, h, sid)

	quietAfterResume := fullCycleResume(ctx, t, h, sid)

	fullCycleClose(ctx, t, h, sid, quietAfterResume)

	fullCycleTranscript(t, h, sid)
}

// newFullCycleFront composes the full loop over the full-cycle corpus row
// with the complete phase stack the production wiring carries: the model
// selection (the corpus-locked setModel preamble), the authenticate
// machinery over the synthetic fixture key, and the responder whose
// PermissionForwarder is the real front bridge — assigned before any frame
// flows. The scripted client answers allow_once (the cycle's permission
// decision).
func newFullCycleFront(t *testing.T) *FrontHarness {
	t.Helper()

	h := ComposeFront(t, mustLoadScenario(t, "full-cycle"), func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
			acpfront.WithAuth(acpfront.AuthConfig{
				Keys:       hostsvc.NewStaticKey(syntheticAuthKey, authFixKeyID),
				Install:    authFixtureInstall(t),
				WantModels: authFixWantModels(),
			}),
		)

		responder := interact.NewResponder(interact.WithPermissionForwarder(
			interact.NewPermissionBridge(front.PermissionForward()).Forward,
		))
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	h.rec.SetPermissionScript(PermissionScript{SelectedOptionID: permAllowOptionID})

	return h
}

// fullCycleAuthenticate is the cycle's authenticate leg: the honest chain
// answers the empty {} success on the wire (the read-back evidence on the
// recorded transcript is asserted at the cycle's end — the harness's
// transcript accessor closes the back leg, so transcript-level locks run
// once, after the whole story).
func fullCycleAuthenticate(ctx context.Context, t *testing.T, h *FrontHarness) {
	t.Helper()

	if _, err := h.client.Authenticate(ctx, acp.AuthenticateRequest{
		MethodId: acp.AuthMethodId(acpfront.AuthMethodZaiAPIKey),
	}); err != nil {
		t.Fatalf("authenticate: %v", err)
	}

	lines := h.ACPLines()
	reqIdx := teeLineIndex(t, lines, "out", `"authenticate"`, "authenticate request")
	respLine := authTeeResponse(t, lines, reqIdx)

	if strings.Contains(respLine.JSON, `"error"`) {
		t.Fatalf("authenticate response carries an error frame: %s", respLine.JSON)
	}

	if !strings.Contains(respLine.JSON, `"result":{}`) {
		t.Fatalf("authenticate response = %s, want the empty {} result", respLine.JSON)
	}
}

// fullCycleOpenSession opens the cycle's session with the tool tests' fixed
// cwd (the absolute-path anchor the Edit diff/location asserts resolve
// against) and returns the id.
func fullCycleOpenSession(ctx context.Context, t *testing.T, h *FrontHarness) string {
	t.Helper()

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: toolLifecycleCwd})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	sid := string(newResp.SessionId)
	if sid == "" {
		t.Fatal("session/new returned an empty session id")
	}

	return sid
}

// fullCyclePrompt is the cycle's prompt leg: the tool stream, the bridged
// permission decision, and the end_turn resolution — asserted on the
// collector and the raw tee.
func fullCyclePrompt(ctx context.Context, t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	resp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("edit the file")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (the allowed permission must complete the turn)",
			resp.StopReason, acp.StopReasonEndTurn)
	}

	updates := h.Updates()

	assertEditStartCard(t, updates, sid)
	assertEditPatchSequence(t, updates)
	assertForeignSuppression(t, h)
	assertCapturedPermissionAsk(t, h)
	assertFullCyclePermissionPlacement(t, h)
	assertToolFramesBeforeResponse(t, h, "full cycle")
	assertFullCycleQuietAfterResponse(t, h)
}

// assertFullCyclePermissionPlacement pins the composition's ordering truth
// on the raw ACP tee: the request_permission frame lands BETWEEN the
// started and progress tool frames — after the tc1 start card and its
// started patch, before the progress patch and the terminal completed
// patch (the completed patch is causally locked after the ask: the fake
// emits it only after the client's answer came back through the bridge).
func assertFullCyclePermissionPlacement(t *testing.T, h *FrontHarness) {
	t.Helper()

	lines := h.ACPLines()

	permIdx := teeLineIndex(t, lines, "in", teePermFrame, "request_permission frame")
	startIdx := teeLineIndex(t, lines, "in", teeToolStart, "tc1 tool_call start card")

	patchIdxs := teeLineIndexes(lines, "in", teeToolPatch)
	if len(patchIdxs) < 2 {
		t.Fatalf("tee records %d tool_call_update frames, want the started and progress patches at least: %s",
			len(patchIdxs), updateKinds(h.Updates()))
	}

	if permIdx < startIdx || permIdx < patchIdxs[0] {
		t.Fatalf("request_permission frame (line %d) not after the tool stream's start "+
			"(start card %d, first patch %d) — the ask must land between the started and progress frames",
			permIdx, startIdx, patchIdxs[0])
	}

	progressIdx := -1
	for _, idx := range patchIdxs {
		if idx > permIdx {
			progressIdx = idx

			break
		}
	}

	if progressIdx < 0 {
		t.Fatalf("no tool patch lands after the request_permission frame (line %d) — "+
			"the progress patch must follow the bridged decision; patches at %v", permIdx, patchIdxs)
	}

	if !strings.Contains(lines[progressIdx].JSON, teeInProgress) {
		t.Fatalf("first post-ask patch (line %d) is not in_progress: %s", progressIdx, lines[progressIdx].JSON)
	}

	completedIdx := patchIdxs[len(patchIdxs)-1]
	if completedIdx <= permIdx {
		t.Fatalf("terminal completed patch (line %d) not after the ask (line %d)", completedIdx, permIdx)
	}
}

// assertFullCycleQuietAfterResponse locks the response frame as the cycle's
// quiet point: exactly ONE prompt response on the tee (exactly-once), and
// NOTHING streams after it — no update frame of any kind (flush-before-
// stopReason through the pump at the composed tier).
func assertFullCycleQuietAfterResponse(t *testing.T, h *FrontHarness) {
	t.Helper()

	lines := h.ACPLines()

	respIdxs := teeLineIndexes(lines, "in", `"stopReason"`)
	if len(respIdxs) != 1 {
		t.Fatalf("tee records %d prompt response frames, want exactly 1 (one prompt, resolved exactly once)",
			len(respIdxs))
	}

	respIdx := respIdxs[0]

	for i, line := range lines[respIdx+1:] {
		if line.Dir == "in" && strings.Contains(line.JSON, teeUpdateMark) {
			t.Fatalf("update frame at line %d streams after the prompt response (line %d) — "+
				"stopReason must land only after the flush", respIdx+1+i, respIdx)
		}
	}
}

// fullCycleResume is the cycle's resume leg: reattach, then ZERO client
// updates through the leg (no replay — the completed turn's frames stayed
// behind the prompt response that already flushed them). Returns the
// collector's count so the close leg can assert the quiet holds through
// teardown.
func fullCycleResume(ctx context.Context, t *testing.T, h *FrontHarness, sid string) int {
	t.Helper()

	before := len(h.Updates())

	if _, err := h.client.ResumeSession(ctx, acp.ResumeSessionRequest{
		SessionId: acp.SessionId(sid),
		Cwd:       toolLifecycleCwd,
	}); err != nil {
		t.Fatalf("session/resume: %v", err)
	}

	if after := len(h.Updates()); after != before {
		t.Fatalf("client collected %d new updates across the resume leg, want 0 "+
			"(no replay of pre-resume history — D-04)", after-before)
	}

	return before
}

// fullCycleClose is the cycle's close leg: the ordered teardown releases
// exactly the session machine's observers (the plan-05-03 surface), the
// resume-leg quiet holds through teardown, and the post-close prompt fails
// clean — the coded ErrNoSession-class rejection, never a zombie session.
func fullCycleClose(ctx context.Context, t *testing.T, h *FrontHarness, sid string, quietCount int) {
	t.Helper()

	notifPre, routerPre := h.conn.ObserverCounts()

	if _, err := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("session/close: %v", err)
	}

	if notif, router := h.conn.ObserverCounts(); notif != notifPre-1 || router != routerPre-1 {
		t.Fatalf("post-close observer counts = (%d, %d), want exactly the machine's entries removed (%d, %d)",
			notif, router, notifPre-1, routerPre-1)
	}

	if updates := len(h.Updates()); updates != quietCount {
		t.Fatalf("client collected %d updates after resume+close, want %d (no replay through teardown)",
			updates, quietCount)
	}

	_, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("post-close")},
	})
	assertCodedInvalidRequest(t, err)
}

// fullCycleTranscript is the cycle's recorded-wire audit: once the story is
// told, the harness's transcript (which closes the back leg) carries every
// leg's evidence — the authenticate chain's honest run, the D-03 preamble,
// the resume leg's re-subscribe, the teardown wire, and the permission
// reply's embedded-response bytes.
func fullCycleTranscript(t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertFullCycleAuthEvidence(t, tr)
	assertBackLegNewSessionSequence(t, tr, sid)
	assertResumeLegOrder(t, tr, sid)
	assertFullCycleTeardownWire(t, tr)
	assertPermissionReply(t, tr)
}

// assertFullCycleAuthEvidence locks the authenticate leg on the transcript
// (PITFALL 3: the success arm asserts the SEQUENCE, never the return value
// alone): exactly one push and one refresh in order, both BEFORE the
// cycle's session/create, and the registry surface the read-back verdict
// consumed is on the wire.
func assertFullCycleAuthEvidence(t *testing.T, tr fake.Transcript) {
	t.Helper()

	pushIdx := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig)
	refreshIdx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry)

	if len(pushIdx) != 1 || len(refreshIdx) != 1 {
		t.Fatalf("transcript records push×%d refresh×%d, want exactly one honest chain run",
			len(pushIdx), len(refreshIdx))
	}

	if pushIdx[0] >= refreshIdx[0] {
		t.Fatalf("chain order violated: push(%d) must precede refresh(%d)", pushIdx[0], refreshIdx[0])
	}

	createIdx := transcriptIndex(t, tr, "in", `"method":"session/create"`, "create request")
	if refreshIdx[0] >= createIdx {
		t.Fatalf("cycle order violated: authenticate's refresh(%d) must precede session/new's create(%d)",
			refreshIdx[0], createIdx)
	}

	if len(transcriptIndexes(tr, string(fake.DirOut), `"scope":"registry"`)) == 0 {
		t.Fatal("no registry state.updated surface on the transcript — the read-back verdict has no recorded evidence")
	}
}

// assertFullCycleTeardownWire locks the close leg's back-leg shape: exactly
// one session/close after the cycle's single send, NO session/stop (the
// close tears down an idle session — no cancel leg), and the post-close
// prompt never touched the wire.
func assertFullCycleTeardownWire(t *testing.T, tr fake.Transcript) {
	t.Helper()

	closeIdxs := transcriptIndexes(tr, "in", `"method":"session/close"`)
	if len(closeIdxs) != 1 {
		t.Fatalf("back leg records %d session/close frames, want exactly 1", len(closeIdxs))
	}

	sends := transcriptIndexes(tr, "in", `"method":"session/send"`)
	if len(sends) != 1 {
		t.Fatalf("back leg records %d session/send frames, want exactly 1 (the cycle's one prompt; "+
			"the post-close prompt never reached the wire)", len(sends))
	}

	if sends[0] >= closeIdxs[0] {
		t.Fatalf("wire order violated: the prompt send(%d) must precede the teardown close(%d)",
			sends[0], closeIdxs[0])
	}

	if stops := transcriptIndexes(tr, "in", `"method":"session/stop"`); len(stops) != 0 {
		t.Fatalf("back leg records %d session/stop frames, want 0 (an idle-session close runs no cancel leg)",
			len(stops))
	}
}
