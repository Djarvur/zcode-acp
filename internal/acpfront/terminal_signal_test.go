package acpfront_test

import (
	"context"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// TestDesktopTerminalSignalTiming locks the QA-03 field blocker on the
// composed loop over the desktop-terminal-signal corpus — the REAL desktop's
// terminal surface (EXP-VM-QA02-family dumps 2026-11-23, green106:
// computer-use/operation-event kind "turn-completed" + v4/telemetry/event
// kind "turn.terminal" status success, each turnId-scoped, shapes verbatim
// scrubbed) — with the field failure's timing replayed against the
// production quiet-window gates:
//
// the model thinks SILENTLY for 7 s (the second session/messages rule holds
// the poll watcher off for 7000 ms on a byte-stable poll surface — no frame
// for the quiet window to count), so the send-latency floor (3x 2.5 s from
// the send ack, watcher.go dialect defaults) releases the settle DURING the
// think — the exact instant the field runs resolved the turn off the floor
// and replied end_turn with an orphan chunk. The floor-instant poll carries
// NO reply text (the field truth the completion-side salvage read); the
// orphan reasoning delta rode the send-time surface, suppressing the
// zero-chunk salvage exactly as it did in the field. The REAL terminal
// signals and the reply-text delta ride a later poll, after the floor.
//
// The contract: the deltas that stream after the floor still reach the
// client as chunks, and the turn resolves by the real terminal signal —
// end_turn leaves only after the desktop said turn-completed, never off the
// floor (the QA-03 report: the Orca card showed an empty reply because the
// wrapper ended the turn at the floor with the reply still seconds away).
//
// RED (the field bug): the floor's settle fires during the think, the turn
// resolves, and the reply text never reaches the client — the assertion
// counts ONE orphan chunk where the wire carried two deltas.
func TestDesktopTerminalSignalTiming(t *testing.T) {
	t.Run("post-floor deltas reach the client; end_turn after the terminal", testDesktopTerminalTiming)
	t.Run("client-visible golden: the chunk sequence and the final end_turn", testDesktopTerminalGolden)
}

// desktopTerminalWantUpdates is the expected client-visible chunk sequence
// over the desktop-terminal-signal corpus: the orphan reasoning delta (the
// field bug's one delivered chunk) and the reply text that streamed after
// the floor — the chunk the floor-resolved wrapper never delivered.
func desktopTerminalWantUpdates() []struct {
	thought bool
	text    string
} {
	return []struct {
		thought bool
		text    string
	}{
		{true, "Thinking"},
		{false, "ok"},
	}
}

// testDesktopTerminalTiming drives the composed loop and asserts the timing
// semantics: exactly the corpus's two model.streaming payloads as chunks
// (the orphan thought, then the reply text), end_turn, and the
// flush-before-stopReason order.
func testDesktopTerminalTiming(t *testing.T) {
	wantUpdates := desktopTerminalWantUpdates()

	h := newTracerFront(t, "desktop-terminal-signal")

	ctx, cancel := context.WithTimeout(context.Background(), terminalSignalExchangeTimeout)
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
		t.Fatalf("stopReason = %q, want %q (the real terminal signal resolves the turn)",
			promptResp.StopReason, acp.StopReasonEndTurn)
	}

	// The post-floor deltas must reach the client: one chunk each, in
	// arrival order. A floor-resolved turn delivers only the orphan — the
	// watcher's settle beat the poll that carried the reply text.
	updates := h.Updates()
	if len(updates) != len(wantUpdates) {
		t.Fatalf("client collected %d updates, want exactly %d "+
			"(the deltas that streamed after the quiet floor must still reach the client — "+
			"the turn may not end off the floor while the real terminal signal is pending)",
			len(updates), len(wantUpdates))
	}

	for i, want := range wantUpdates {
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

	// Flush-before-stopReason (D-04): the chunks leave before the response,
	// and nothing client-visible streams after it.
	assertChunksBeforeResponse(t, h)
}

// testDesktopTerminalGolden byte-locks the client-visible stream: the two
// chunk frames plus the prompt response, one JSON line per frame in
// recorded wire order, against the recorded golden fixture. The fixture is
// recorded from the GREEN run with -update — never committed from a red
// run (record-honesty discipline).
func testDesktopTerminalGolden(t *testing.T) {
	h := newTracerFront(t, "desktop-terminal-signal")

	ctx, cancel := context.WithTimeout(context.Background(), terminalSignalExchangeTimeout)
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

	if updates := h.Updates(); len(updates) != len(desktopTerminalWantUpdates()) {
		t.Fatalf("golden run collected %d updates, want %d — recording from a broken run is forbidden",
			len(updates), len(desktopTerminalWantUpdates()))
	}

	fake.CompareGolden(t, "testdata/golden/desktop-terminal-signal.acp.golden", desktopACPBytes(h))
}

// terminalSignalExchangeTimeout bounds the timing replay: resolution lands
// at ~8s wall clock (the scripted 7s think hold plus the front's poll
// cadence — the floor fires at 7.5s and the real terminal signal must
// outlive it), so the bound is the quiet-window rides' 20s, not the 10s
// harness default.
const terminalSignalExchangeTimeout = quietWindowExchangeTimeout
