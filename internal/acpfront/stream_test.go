package acpfront_test

import (
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"
)

// TestThoughtChunks locks CORE-04 (plan 04-04) on the composed loop over
// the two corpus rows authored for it:
//
//	positive (part-delta-streaming): EVERY backend field-reasoning part.delta
//	becomes exactly ONE agent_thought_chunk in arrival order, interleaved
//	correctly with the text message chunks — chunks mirror envelopes 1:1
//	(consecutive deltas never merge, one delta never splits).
//	negative (no-reasoning): a turn with zero reasoning deltas produces ZERO
//	thought chunks while the message chunks stay intact — thought chunks are
//	backend-sourced ONLY, never synthesized (the D-04 honesty prohibition;
//	this subtest is its enforcement test).
//
// Assertions run on both surfaces: the client's collected updates (chunk
// kind + text, in order) and the raw ACP tee frames (the wire truth).
func TestThoughtChunks(t *testing.T) {
	t.Run("positive: reasoning deltas stream as thought chunks interleaved with text", testThoughtChunksPositive)
	t.Run("negative: zero reasoning deltas yield zero thought chunks", testThoughtChunksNegative)
}

// testThoughtChunksPositive drives the part-delta-streaming corpus — the
// send rule's ordered side effects interleave text "Answer:", reasoning
// "Thinking about it", text " done" — and asserts the 1:1 envelope→chunk
// mirror on both the collector and the raw tee.
func testThoughtChunksPositive(t *testing.T) {
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

	updates := h.Updates()
	if len(updates) != 3 {
		t.Fatalf("client collected %d updates, want exactly 3 "+
			"(one chunk per part.delta envelope, never merged or split)", len(updates))
	}

	// CORE-04 positive: every field-reasoning delta becomes exactly ONE
	// thought chunk (RED target — the tracer tap skipped reasoning, so
	// zero arrived before the mapping landed).
	thoughts := thoughtUpdates(updates)
	if len(thoughts) != 1 {
		t.Fatalf("collected %d agent_thought_chunk updates, want exactly 1 "+
			"(one per field-reasoning delta, never synthesized)", len(thoughts))
	}

	if thoughts[0].SessionID != sid || chunkText(thoughts[0].Update) != "Thinking about it" {
		t.Fatalf("thought chunk = (%s, %q), want (%s, %q)",
			thoughts[0].SessionID, chunkText(thoughts[0].Update), sid, "Thinking about it")
	}

	assertTextChunksIntact(t, updates)
	assertThoughtChunkWire(t, h, sid)
}

// assertTextChunksIntact checks the message chunks (exactly the two corpus
// text deltas) and the cross-kind arrival order (message → thought →
// message) on the collected updates.
func assertTextChunksIntact(t *testing.T, updates []ACPUpdate) {
	t.Helper()

	// Text chunks unaffected: exactly the two corpus text deltas.
	messages := messageUpdates(updates)
	if len(messages) != 2 {
		t.Fatalf("collected %d agent_message_chunk updates, want exactly 2", len(messages))
	}

	wantText := []string{"Answer:", " done"}
	for i, want := range wantText {
		if got := chunkText(messages[i].Update); got != want {
			t.Fatalf("message chunk[%d] = %q, want %q", i, got, want)
		}
	}

	// Arrival order preserved across kinds: message → thought → message.
	wantThought := []bool{false, true, false}
	for i, want := range wantThought {
		if got := updates[i].Update.AgentThoughtChunk != nil; got != want {
			t.Fatalf("update[%d] thought=%v, want thought=%v (arrival order must mirror the envelope stream)",
				i, got, want)
		}
	}
}

// testThoughtChunksNegative drives the no-reasoning corpus (two text
// deltas, zero field-reasoning envelopes): ZERO thought chunks collected
// and zero on the wire while the message chunks stay intact and the prompt
// answers end_turn — the never-synthesized lock.
func testThoughtChunksNegative(t *testing.T) {
	h := newTracerFront(t, "no-reasoning")

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

	updates := h.Updates()
	if got := len(thoughtUpdates(updates)); got != 0 {
		t.Fatalf("no-reasoning turn produced %d agent_thought_chunk updates — "+
			"thought chunks must be backend-sourced ONLY, never synthesized", got)
	}

	messages := messageUpdates(updates)
	if len(messages) != 2 {
		t.Fatalf("collected %d agent_message_chunk updates, want exactly 2 (message chunks intact)", len(messages))
	}

	wantText := []string{"Plain", " answer"}
	for i, want := range wantText {
		if got := chunkText(messages[i].Update); got != want {
			t.Fatalf("message chunk[%d] = %q, want %q", i, got, want)
		}
	}

	lines := h.ACPLines()
	if got := len(teeLineIndexes(lines, "in", `"agent_thought_chunk"`)); got != 0 {
		t.Fatalf("tee records %d agent_thought_chunk frames, want 0 on a zero-reasoning turn (never synthesized)", got)
	}

	if got := len(teeLineIndexes(lines, "in", `"agent_message_chunk"`)); got != 2 {
		t.Fatalf("tee records %d agent_message_chunk frames, want exactly 2", got)
	}
}

// thoughtUpdates filters the collected updates down to the thought chunks.
func thoughtUpdates(updates []ACPUpdate) []ACPUpdate {
	return kindUpdates(updates, func(u acp.SessionUpdate) bool { return u.AgentThoughtChunk != nil })
}

// messageUpdates filters the collected updates down to the message chunks.
func messageUpdates(updates []ACPUpdate) []ACPUpdate {
	return kindUpdates(updates, func(u acp.SessionUpdate) bool { return u.AgentMessageChunk != nil })
}

// kindUpdates returns the updates matching keep, in arrival order.
func kindUpdates(updates []ACPUpdate, keep func(acp.SessionUpdate) bool) []ACPUpdate {
	out := []ACPUpdate{}
	for i := range updates {
		if keep(updates[i].Update) {
			out = append(out, updates[i])
		}
	}

	return out
}

// assertThoughtChunkWire checks the raw ACP tee frames: exactly one
// agent_thought_chunk frame carrying a text content block for this
// session, strictly between the two agent_message_chunk frames — the
// wire-order interleaving proof.
func assertThoughtChunkWire(t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	lines := h.ACPLines()

	thoughtIdxs := teeLineIndexes(lines, "in", `"agent_thought_chunk"`)
	if len(thoughtIdxs) != 1 {
		t.Fatalf("tee records %d agent_thought_chunk frames, want exactly 1", len(thoughtIdxs))
	}

	msgIdxs := teeLineIndexes(lines, "in", `"agent_message_chunk"`)
	if len(msgIdxs) != 2 {
		t.Fatalf("tee records %d agent_message_chunk frames, want exactly 2", len(msgIdxs))
	}

	if msgIdxs[0] >= thoughtIdxs[0] || thoughtIdxs[0] >= msgIdxs[1] {
		t.Fatalf("thought frame (line %d) not interleaved between the message frames (%v) — arrival order violated",
			thoughtIdxs[0], msgIdxs)
	}

	raw := lines[thoughtIdxs[0]].JSON

	if !strings.Contains(raw, `"type":"text"`) {
		t.Fatalf("thought frame carries no text content block: %s", raw)
	}

	if !strings.Contains(raw, sid) {
		t.Fatalf("thought frame does not carry session id %q: %s", sid, raw)
	}
}
