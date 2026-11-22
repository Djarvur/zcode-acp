package acpfront_test

import (
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The shared delta-stream literals (goconst): the two corpus text deltas
// the streaming corpus rows replay (part-delta-streaming and the
// desktop-delta rows) — extracted so the package stays under the literal
// repetition threshold.
const (
	deltaTextAnswer = "Answer:"
	deltaTextDone   = " done"
)

// TestNoDeltaReplyReconcile locks the QA-03 completion-side reconcile (the
// second half of the field gap): the real desktop's turns VARY run-to-run —
// some stream model.streaming deltas (the landed translation covers those),
// some complete end_turn with ZERO deltas while the reply text sits in the
// backend's message store (the clean-build field turn emitted zero client
// frames). The contract under lock:
//
//	no-delta turn: at completion, before the stopReason leaves, the front
//	reads the turn's final assistant text from the session/messages
//	projection (the poll surface the machine already uses) and emits ONE
//	agent_message_chunk with it — the chunk rides the same flush as the
//	stopReason (flush-before-stopReason).
//	deltas streamed: the reconcile stays silent — no duplicate chunk.
//	poll failure / unrecoverable text: honest empty reply + end_turn —
//	never a fabricated chunk, never a hang.
//	slow poll: the reconcile read is bounded — a projection that answers
//	beyond the bound cannot delay the stopReason into a hang nor leak a
//	late chunk.
//
// Corpus rows replay the desktop shapes verbatim (the desktop-delta-
// streaming row's envelope surface) plus the session/messages projection
// wire truth (EXP-CODINGPLAN-CYCLE-01 poll 24: the completed turn's reply
// is the LAST assistant message, its text in the type:"text" parts; the
// reasoning/step-start/step-finish parts are never message text).
func TestNoDeltaReplyReconcile(t *testing.T) {
	t.Run("no-delta turn: ONE reconcile chunk with the stored reply, then end_turn", testReconcileNoDeltaChunk)
	t.Run("streamed deltas: the reconcile stays silent (no duplicate)", testReconcileDeltaSuppressed)
	t.Run("poll failure: honest empty reply + end_turn, never a hang", testReconcilePollFailure)
	t.Run("slow poll: the bounded reconcile gives up — end_turn without the chunk", testReconcileSlowPollBound)
	t.Run("projection without an assistant reply: no fabricated chunk", testReconcileNoAssistantText)
	t.Run("client-visible golden: the reconcile stream and the suppressed stream", testReconcileGolden)
}

// runReconcileRide drives one composed prompt turn over a reconcile corpus
// row and returns the harness, the session id, and the prompt response.
func runReconcileRide(t *testing.T, scenario string) (*FrontHarness, string, acp.PromptResponse) {
	t.Helper()

	h := newTracerFront(t, scenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openTracerSession(ctx, t, h)

	resp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	return h, sid, resp
}

// testReconcileNoDeltaChunk — the flagship arm: a turn that completes with
// NO model deltas but WITH a final assistant message in the messages
// projection answers exactly ONE agent_message_chunk carrying the FULL
// stored text ("ok"), strictly before the end_turn response frame. The
// projection's reasoning part never rides (no thought chunk), and the
// back leg records the reconcile's ONE session/messages read AFTER the
// send (the completion-time read, not a mid-turn poll).
func testReconcileNoDeltaChunk(t *testing.T) {
	h, sid, resp := runReconcileRide(t, "desktop-nodelta-messages")

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (the reconcile must not change the turn outcome)",
			resp.StopReason, acp.StopReasonEndTurn)
	}

	updates := h.Updates()
	if len(updates) != 1 {
		t.Fatalf("client collected %d updates, want exactly ONE reconcile chunk carrying the stored reply",
			len(updates))
	}

	if updates[0].SessionID != sid {
		t.Fatalf("update session = %q, want %q", updates[0].SessionID, sid)
	}

	if updates[0].Update.AgentThoughtChunk != nil {
		t.Fatal("the projection's reasoning part rode as a thought chunk — only type:text parts are message text")
	}

	if got := chunkText(updates[0].Update); got != "ok" {
		t.Fatalf("reconcile chunk text = %q, want %q (the last assistant message's type:text parts)", got, "ok")
	}

	assertChunksBeforeResponse(t, h)

	// The reconcile read is the completion-time session/messages poll: one
	// request, strictly after the send (asserted on the fake transcript —
	// Transcript closes the back leg, so it rides last).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	sendIdx := transcriptIndex(t, tr, "in", `"method":"session/send"`, "send request")

	msgIdxs := transcriptIndexes(tr, "in", `"method":"session/messages"`)
	if len(msgIdxs) != 1 {
		t.Fatalf("back leg records %d session/messages requests, want exactly 1 (the bounded reconcile read)",
			len(msgIdxs))
	}

	if msgIdxs[0] < sendIdx {
		t.Fatalf("the reconcile read (line %d) precedes the send (line %d) — it must poll at completion, not mid-turn",
			msgIdxs[0], sendIdx)
	}
}

// testReconcileDeltaSuppressed — the no-duplicate arm: the turn streamed
// text deltas AND the projection carries the full reply, yet the client
// receives exactly the streamed chunks — the reconcile stayed silent
// (deltas did stream, so the client already has the text).
func testReconcileDeltaSuppressed(t *testing.T) {
	h, sid, resp := runReconcileRide(t, "desktop-delta-messages")

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
	}

	updates := h.Updates()
	if len(updates) != 2 {
		t.Fatalf("client collected %d updates, want exactly the 2 streamed deltas (a reconcile chunk = DUPLICATE)",
			len(updates))
	}

	wantDeltas := []string{deltaTextAnswer, deltaTextDone}
	for i, want := range wantDeltas {
		if updates[i].SessionID != sid {
			t.Fatalf("update[%d] session = %q, want %q", i, updates[i].SessionID, sid)
		}

		if got := chunkText(updates[i].Update); got != want {
			t.Fatalf("update[%d] text = %q, want %q (the streamed deltas, verbatim)", i, got, want)
		}
	}

	lines := h.ACPLines()
	if got := len(teeLineIndexes(lines, "in", `"agent_message_chunk"`)); got != 2 {
		t.Fatalf("tee records %d agent_message_chunk frames, want exactly 2 (the deltas; no full-reply duplicate)",
			got)
	}
}

// testReconcilePollFailure — the honest-failure arm: the reconcile poll
// answers a wire error, so NO chunk is fabricated and the stopReason still
// leaves (the client sees an empty reply + end_turn). The prompt resolving
// within the bounded exchange IS the never-a-hang proof.
func testReconcilePollFailure(t *testing.T) {
	h, _, resp := runReconcileRide(t, "desktop-nodelta-pollerror")

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (a failed reconcile poll must still answer the turn)",
			resp.StopReason, acp.StopReasonEndTurn)
	}

	if updates := h.Updates(); len(updates) != 0 {
		t.Fatalf("client collected %d updates after a failed reconcile poll, want 0 (nothing fabricated)", len(updates))
	}

	lines := h.ACPLines()
	if got := len(teeLineIndexes(lines, "in", `"sessionUpdate"`)); got != 0 {
		t.Fatalf("tee records %d session/update frames after a failed reconcile poll, want 0", got)
	}
}

// testReconcileSlowPollBound — the bounded-read arm: the projection carries
// the reply text but answers 3s late, past the reconcile's poll bound. The
// turn still answers end_turn with ZERO chunks — the bound gave up before
// the reply arrived (a reconcile that waited would emit the chunk and the
// zero-chunk assert here would fail; the prompt resolving inside the
// bounded exchange IS the never-a-hang proof).
func testReconcileSlowPollBound(t *testing.T) {
	h, _, resp := runReconcileRide(t, "desktop-nodelta-slowpoll")

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (a slow reconcile poll must not hold the turn open)",
			resp.StopReason, acp.StopReasonEndTurn)
	}

	if updates := h.Updates(); len(updates) != 0 {
		t.Fatalf("client collected %d updates after the reconcile poll missed its bound, want 0 (no late-chunk leak)",
			len(updates))
	}

	lines := h.ACPLines()
	if got := len(teeLineIndexes(lines, "in", `"agent_message_chunk"`)); got != 0 {
		t.Fatalf("tee records %d agent_message_chunk frames after the reconcile poll missed its bound, want 0", got)
	}
}

// testReconcileNoAssistantText — the unrecoverable-text arm: the poll
// succeeds but the projection carries no assistant reply (the timeline and
// user entries only). Nothing is fabricated; the stopReason still leaves.
func testReconcileNoAssistantText(t *testing.T) {
	h, _, resp := runReconcileRide(t, "desktop-nodelta-noassistant")

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (an unrecoverable reply text must still answer the turn)",
			resp.StopReason, acp.StopReasonEndTurn)
	}

	if updates := h.Updates(); len(updates) != 0 {
		t.Fatalf("client collected %d updates with no assistant reply in the projection, want 0 (nothing fabricated)",
			len(updates))
	}
}

// testReconcileGolden byte-locks the client-visible streams of the two
// happy arms: the reconcile row (ONE chunk + end_turn) and the suppressed
// row (the two deltas + end_turn, nothing more). Fixtures are recorded
// from the GREEN run with -update — never committed from a red run.
func testReconcileGolden(t *testing.T) {
	h, _, _ := runReconcileRide(t, "desktop-nodelta-messages")
	fake.CompareGolden(t, "testdata/golden/desktop-nodelta-reconcile.acp.golden", desktopACPBytes(h))

	hSuppressed, _, _ := runReconcileRide(t, "desktop-delta-messages")
	fake.CompareGolden(t, "testdata/golden/desktop-delta-suppressed.acp.golden", desktopACPBytes(hSuppressed))
}
