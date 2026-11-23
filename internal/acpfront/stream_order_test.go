package acpfront_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
)

// TestStreamBeforeStopReason locks CORE-03/D-04 flush-before-stopReason
// (plan 04-04) on RAW ACP wire lines — the ordering proof across BOTH
// completion variants plus the zero-delta honesty probe:
//
//	terminal-event (part-delta-streaming): every chunk frame (message AND
//	thought) strictly precedes the prompt response frame on the recorded
//	tee, and NO session/update frame appears after it.
//	quiet-window (completion-quiet-window): completion without any terminal
//	event still answers only after the (zero) pending updates — the
//	response frame is the last protocol frame of the exchange.
//	zero-delta (completion-quiet-window): a turn whose send emits no
//	part.delta rows yields zero chunks yet answers stopReason end_turn —
//	the completion signal alone suffices.
//
// This is a LOCK suite on tracer-built behavior: the registration order
// (tap before machine, front.go New) and the SDK's FIFO write mutex make
// the ordering structural, so all three subtests are expected green on
// first run — the suite's value is pinning the ordering so later refactors
// cannot silently reorder it. The assertions run on raw tee lines (string
// containment of the wire keys), not decoded structs — the wire truth is
// the contract.
func TestStreamBeforeStopReason(t *testing.T) {
	t.Run("terminal-event: chunk frames precede the response, none follow", testStreamOrderTerminalEvent)
	t.Run("quiet-window: the response frame is the last protocol frame", testStreamOrderQuietWindow)
	t.Run("zero-delta: no deltas yields zero chunks yet answers end_turn", testStreamOrderZeroDelta)
}

// quietWindowExchangeTimeout bounds the quiet-window rides: the Phase-3
// settle gates put resolution at ~10s wall clock (floor = 3x 2.5s from the
// send ack, then a 2.5s quiet settle — watcher.go dialect defaults), so
// these subtests use a bounded 20s exchange instead of the 10s harness
// default — still fail-fast, safely above the wall-clock floor.
const quietWindowExchangeTimeout = 20 * time.Second

// testStreamOrderTerminalEvent rides part-delta-streaming: the
// turn.completed terminal event resolves the turn only after the tap
// drained the interleaved delta envelopes ahead of it (fan-out order), so
// every chunk frame precedes the response frame.
func testStreamOrderTerminalEvent(t *testing.T) {
	h := newTracerFront(t, "part-delta-streaming")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openTracerSession(ctx, t, h)

	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}

	assertChunksBeforeResponse(t, h)
}

// testStreamOrderQuietWindow rides completion-quiet-window: no turn.completed
// ever arrives — the poll watcher's quiet settle resolves the turn, and the
// prompt still answers only after the (zero) pending updates: the response
// frame is the last protocol frame of the exchange.
func testStreamOrderQuietWindow(t *testing.T) {
	h := newTracerFront(t, "completion-quiet-window")

	ctx, cancel := context.WithTimeout(context.Background(), quietWindowExchangeTimeout)
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openTracerSession(ctx, t, h)

	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (the quiet-window completion must still answer end_turn)",
			promptResp.StopReason, acp.StopReasonEndTurn)
	}

	lines := h.ACPLines()
	respIdx := teeLineIndex(t, lines, "in", `"end_turn"`, "prompt response")

	for i, line := range lines[respIdx+1:] {
		if line.Dir == "in" {
			t.Fatalf("agent frame at line %d streams after the response — completion without a terminal event "+
				"must still answer only after the flush: %s", respIdx+1+i, line.JSON)
		}
	}
}

// testStreamOrderZeroDelta rides completion-quiet-window as the empty
// probe: the send rule emits NO part.delta rows, so the turn yields zero
// message chunks yet still answers stopReason end_turn — the completion
// signal alone suffices, nothing synthesized.
func testStreamOrderZeroDelta(t *testing.T) {
	h := newTracerFront(t, "completion-quiet-window")

	ctx, cancel := context.WithTimeout(context.Background(), quietWindowExchangeTimeout)
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openTracerSession(ctx, t, h)

	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (a zero-delta turn must still answer end_turn)",
			promptResp.StopReason, acp.StopReasonEndTurn)
	}

	if updates := h.Updates(); len(updates) != 0 {
		t.Fatalf("zero-delta turn produced %d updates, want 0 — the completion signal must never synthesize chunks",
			len(updates))
	}

	lines := h.ACPLines()
	// The available_commands_update catalog frame is a legitimate non-chunk
	// session/update (v0.21.0) — the zero-frame lock covers CHUNK kinds.
	chunks := len(teeLineIndexes(lines, "in", `"agent_message_chunk"`)) +
		len(teeLineIndexes(lines, "in", `"agent_thought_chunk"`))
	if chunks != 0 {
		t.Fatalf("tee records %d chunk frames on a zero-delta turn, want 0", chunks)
	}
}

// assertChunksBeforeResponse checks the CORE-03 wire order on the raw tee:
// every chunk frame (message AND thought) strictly precedes the prompt
// response frame, and NO session/update frame appears after it
// (flush-before-stopReason, D-04).
func assertChunksBeforeResponse(t *testing.T, h *FrontHarness) {
	t.Helper()

	lines := h.ACPLines()
	respIdx := teeLineIndex(t, lines, "in", `"end_turn"`, "prompt response")

	msgIdxs := teeLineIndexes(lines, "in", `"agent_message_chunk"`)
	if len(msgIdxs) == 0 {
		t.Fatal("no agent_message_chunk frames on the tee — nothing streamed before the stopReason")
	}

	for i, idx := range msgIdxs {
		if idx >= respIdx {
			t.Fatalf("message chunk %d (line %d) not before the response (line %d) — flush-before-stopReason violated",
				i, idx, respIdx)
		}
	}

	for i, idx := range teeLineIndexes(lines, "in", `"agent_thought_chunk"`) {
		if idx >= respIdx {
			t.Fatalf("thought chunk %d (line %d) not before the response (line %d) — flush-before-stopReason violated",
				i, idx, respIdx)
		}
	}

	for _, line := range lines[respIdx+1:] {
		if line.Dir == "in" && strings.Contains(line.JSON, `"sessionUpdate"`) {
			t.Fatalf("update frame streams after the prompt response — stopReason must land only after the flush: %s",
				line.JSON)
		}
	}
}
