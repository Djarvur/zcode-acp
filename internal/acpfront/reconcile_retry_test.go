package acpfront_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// The reconcile retry-window fix cycle's review findings (the 2026-11-23
// delta review of e30cb97): the retry loop's three warnings and the breach
// re-check, each locked by the deterministic corpus replay.

// TestReconcileTwoTurnStaleGuard — WR-01: the retry loop must never emit a
// PRIOR turn's stored reply as this turn's reconcile text. Turn 1 streams
// its reply ("first-reply"); turn 2 completes zero-delta while the
// projection still lags (its poll one carries ONLY turn 1's assistant
// entry, created BEFORE turn 2 started). The turn-scoped selection rejects
// the stale entry (its projection time precedes this turn's start stamp)
// and keeps polling; poll two reads the caught-up projection and recovers
// turn 2's own reply ("second-reply").
func TestReconcileTwoTurnStaleGuard(t *testing.T) {
	h := newTracerFront(t, "desktop-twoturn-lateprojection")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openTracerSession(ctx, t, h)

	resp1, err := h.client.Prompt(
		ctx, acp.PromptRequest{SessionId: acp.SessionId(sid), Prompt: []acp.ContentBlock{acp.TextBlock("one")}})
	if err != nil {
		t.Fatalf("prompt one: %v", err)
	}
	if resp1.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("turn one stopReason = %q, want end_turn", resp1.StopReason)
	}

	resp2, err := h.client.Prompt(
		ctx, acp.PromptRequest{SessionId: acp.SessionId(sid), Prompt: []acp.ContentBlock{acp.TextBlock("two")}})
	if err != nil {
		t.Fatalf("prompt two: %v", err)
	}
	if resp2.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("turn two stopReason = %q, want end_turn", resp2.StopReason)
	}

	updates := h.Updates()
	if len(updates) != 2 {
		t.Fatalf("client collected %d updates, want exactly 2 (turn one's streamed delta + "+
			"turn two's recovered reply), got: %+v", len(updates), updates)
	}

	if got := chunkText(updates[0].Update); got != "first-reply" {
		t.Fatalf("turn one text = %q, want %q (the streamed delta)", got, "first-reply")
	}

	if got := chunkText(updates[1].Update); got != "second-reply" {
		t.Fatalf("turn two reconcile text = %q, want %q — a prior turn's stored reply must never ride as "+
			"this turn's recovery", got, "second-reply")
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if got := len(transcriptIndexes(tr, "in", `"method":"session/messages"`)); got != 2 {
		t.Fatalf("back leg records %d session/messages requests, want exactly 2 (turn one streamed and never polls; "+
			"turn two polls twice — poll one rejects the stale prior-turn entry, poll two recovers the reply)", got)
	}
}

// TestReconcileLateDeltaSingleDelivery — WR-03's mid-window arm: a text
// delta landing DURING the reconcile window (the messages rule's side
// effect fires at poll arrival, its reply answers 1.5s later) means the
// client already has the text live — the reconcile must stay silent, so the
// turn delivers exactly ONE message chunk (the delta), never delta + stored
// text. Mutation lock: deleting the message-count guard makes this red.
func TestReconcileLateDeltaSingleDelivery(t *testing.T) {
	h, _, resp := runReconcileRide(t, "desktop-nodelta-latedelta")

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
	}

	updates := h.Updates()
	if len(updates) != 1 {
		t.Fatalf("client collected %d updates, want exactly ONE (the late delta — "+
			"a stored-text chunk on top = DUPLICATE)", len(updates))
	}

	if got := chunkText(updates[0].Update); got != "late-ok" {
		t.Fatalf("the delivered chunk = %q, want %q (the late delta, not the stored text)", got, "late-ok")
	}
}

// TestReconcileWindowBreachFailsLoud — IN-01: a stream-ceiling breach caused
// by a late delta DURING the reconcile window must fail the turn loud (the
// coded ceiling error), never answer a success end_turn for a turn whose
// output was dropped. The tiny injectable ceiling makes the window's 4 KiB
// delta breach on arrival; the cancelled window unwinds the reconcile, and
// the prompt tail must re-check the breach before answering.
func TestReconcileWindowBreachFailsLoud(t *testing.T) {
	h := composeBackLeg(t, "desktop-nodelta-breach", acpfront.WithStreamCeiling(1024))

	ctx, cancel := context.WithTimeout(context.Background(), backpressureWait)
	defer cancel()

	sid := h.openSession(ctx, t)

	var delivered atomic.Int32

	sink := func(_ context.Context, _ acpport.SessionNote) error {
		delivered.Add(1)

		return nil
	}

	_, err := h.front.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: "hi"}, sink)
	if err == nil {
		t.Fatal("a window breach resolved with nil error — the dropped output must FAIL THE TURN LOUD")
	}

	var seamErr *acpport.Error
	if !errors.As(err, &seamErr) {
		t.Fatalf("window breach error = %T (%v), want a coded *acpport.Error", err, err)
	}

	if !strings.Contains(seamErr.Message, "ceiling") {
		t.Fatalf("window breach error must name the cause, got %q", seamErr.Message)
	}

	if got := delivered.Load(); got != 0 {
		t.Fatalf("sink delivered %d notes after a window breach, want 0 (the oversized delta was refused)", got)
	}
}
