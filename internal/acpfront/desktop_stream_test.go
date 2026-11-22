package acpfront_test

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// TestMain registers the golden-update flag on this package's test binary —
// fake.CompareGolden looks the flag up by name (the fake package's own
// TestMain registers it for ITS binary; each package needs its own).
func TestMain(m *testing.M) {
	flag.Bool(fake.UpdateFlagName, false, "rewrite golden fixtures from the current run")

	os.Exit(m.Run())
}

// TestDesktopDeltaStreaming locks QA-03 on the composed loop over the
// desktop-delta-streaming corpus — the REAL desktop's model-response delta
// carrier (session/event params.type "model.streaming" with desktop-
// continuous delivery, EXP-APIKEY-CYCLE-03 frames 176+ shapes verbatim):
//
//	positive: every payload text_delta becomes exactly ONE agent_message_chunk
//	and every payload reasoning_delta exactly ONE agent_thought_chunk, in
//	arrival order (the same CORE-03/CORE-04 1:1 discipline the part.delta
//	fake-tier surface is locked by — the real wire's carrier, where the
//	field bug lived: the wrapper translated only part.delta, so real
//	desktop turns completed end_turn with ZERO client-visible chunks).
//	scope guard: the tool_input_start/delta/end trio and the turn.* /
//	session.* desktop-UI facts produce NOTHING client-visible — tool cards
//	ride the tool.updated stream, never synthesized.
//
// Assertions run on both surfaces: the client's collected updates (chunk
// kind + text, in order) and the byte-locked ACP-leg golden (the wire
// truth: the chunk frame sequence plus the final end_turn response).
func TestDesktopDeltaStreaming(t *testing.T) {
	t.Run("model.streaming deltas translate to client chunks in arrival order", testDesktopDeltaChunks)
	t.Run("client-visible golden: the chunk sequence and the final end_turn", testDesktopDeltaGolden)
}

// desktopDeltaWantUpdates is the expected client-visible chunk sequence over
// the desktop-delta-streaming corpus: the scripted model.streaming payload
// deltas verbatim (reasoning_delta → thought chunks, text_delta → message
// chunks), the tool_input_* trio and turn.started/turn.completed absent.
var desktopDeltaWantUpdates = []struct {
	thought bool
	text    string
}{
	{true, "Thinking"},
	{true, " about it"},
	{false, "Answer:"},
	{false, " done"},
}

// testDesktopDeltaChunks drives the composed loop and asserts the QA-03
// translation on the collected updates: exactly the four corpus deltas as
// chunks (never merged, never split, nothing synthesized), the tool-args
// trio invisible, and every chunk frame on the wire BEFORE the end_turn
// response (flush-before-stopReason).
func testDesktopDeltaChunks(t *testing.T) {
	h := newTracerFront(t, "desktop-delta-streaming")

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
		t.Fatalf("stopReason = %q, want %q (the turn.completed terminal still resolves the turn)",
			promptResp.StopReason, acp.StopReasonEndTurn)
	}

	updates := h.Updates()
	if len(updates) != len(desktopDeltaWantUpdates) {
		t.Fatalf("client collected %d updates, want exactly %d "+
			"(one chunk per model.streaming delta payload; the tool_input_* trio and turn.* desktop facts never translate)",
			len(updates), len(desktopDeltaWantUpdates))
	}

	for i, want := range desktopDeltaWantUpdates {
		if updates[i].SessionID != sid {
			t.Fatalf("update[%d] session = %q, want %q", i, updates[i].SessionID, sid)
		}

		if thought := updates[i].Update.AgentThoughtChunk != nil; thought != want.thought {
			t.Fatalf("update[%d] thought=%v, want thought=%v (arrival order must mirror the delta stream)",
				i, thought, want.thought)
		}

		if got := chunkText(updates[i].Update); got != want.text {
			t.Fatalf("update[%d] text = %q, want %q", i, got, want.text)
		}
	}

	// The tool-args trio renders no tool card on this row (no tool.updated
	// frames are scripted): a tool_call/tool_call_update update here would be
	// synthesized from the held kinds — never.
	for i, update := range updates {
		if update.Update.ToolCall != nil || update.Update.ToolCallUpdate != nil {
			t.Fatalf("update[%d] carries a tool surface — tool-args model.streaming kinds must stay held from the client", i)
		}
	}

	assertChunksBeforeResponse(t, h)
}

// testDesktopDeltaGolden byte-locks the client-visible stream: every
// agent→client session/update frame plus the prompt response, one JSON line
// per frame in recorded wire order, against the recorded golden fixture.
// The fixture is recorded from the GREEN run with -update — never committed
// from a red run (a missing fixture FAILS; record-honesty discipline).
func testDesktopDeltaGolden(t *testing.T) {
	h := newTracerFront(t, "desktop-delta-streaming")

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

	fake.CompareGolden(t, "testdata/golden/desktop-delta-streaming.acp.golden", desktopACPBytes(h))
}

// desktopACPBytes renders the client-visible ACP frames of one composed run
// for the golden fixture: every agent→client frame that is a session/update
// notification plus the prompt response, one JSON line per frame, in
// recorded (wire) order. Setup responses (initialize/session-new) and all
// client→agent frames stay out — the golden locks the STREAM the client
// sees, not the setup conversation.
func desktopACPBytes(h *FrontHarness) []byte {
	var buf bytes.Buffer

	for _, line := range h.ACPLines() {
		if line.Dir != "in" {
			continue
		}

		if !strings.Contains(line.JSON, `"sessionUpdate"`) && !strings.Contains(line.JSON, `"end_turn"`) {
			continue
		}

		buf.WriteString(line.JSON)
		buf.WriteByte('\n')
	}

	return buf.Bytes()
}
