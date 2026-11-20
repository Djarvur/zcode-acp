package acpfront_test

import (
	"context"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"
)

// TOOL-01 composed-tier locks (plan 05-04): the tool-lifecycle corpus row
// driven through the composed harness — the client sees the ordered
// tool_call / tool_call_update sequence with kind/status/title mapping,
// Edit/Write diffs at session-cwd-ABSOLUTE paths, batch/raw/subagent
// mirrors staying log-only (never client cards), and foreign-turnId
// deltas never reaching the client (IN-05).
//
// The fixed session cwd pins the absolute-path resolution (Pitfall 5) —
// the corpus file_path values are workspace-relative on purpose.

// toolLifecycleCwd is the fixed absolute session cwd the tool tests open
// sessions with; every diff/location assert resolves against it.
const toolLifecycleCwd = "/tmp/tool-lifecycle-ws"

// TestToolLifecycle drives the tool-lifecycle corpus row through the
// composed harness in two staged legs:
//
//	leg 1 (Edit): scheduled→started→progress→result produces exactly the
//	  ordered tool_call (pending, diff attached, kind edit, title from the
//	  description) → tool_call_update (in_progress) → tool_call_update
//	  (in_progress) → tool_call_update (completed with rawOutput and the
//	  diff re-emitted) sequence, every tool frame before the prompt
//	  response; the batch and subagent-mirror envelopes produce NO client
//	  tool cards; the same-turn text delta streams and the FOREIGN-turnId
//	  delta never does (IN-05).
//	leg 2 (Write): a Write-input scheduled event renders the diff with
//	  newText only — oldText ABSENT on the raw wire (the new-file shape).
func TestToolLifecycle(t *testing.T) {
	t.Run("edit lifecycle: ordered frames, diff at absolute path, foreign delta suppressed", testToolLifecycleEdit)
	t.Run("write lifecycle: new-file diff carries newText only", testToolLifecycleWrite)
}

// openToolSession opens one session over the composed loop with the fixed
// toolLifecycleCwd (the absolute-path anchor) and returns the id.
func openToolSession(ctx context.Context, t *testing.T, h *FrontHarness) string {
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

// toolUpdates filters the collected updates down to the tool_call starts.
func toolUpdates(updates []ACPUpdate) []ACPUpdate {
	return kindUpdates(updates, func(u acp.SessionUpdate) bool { return u.ToolCall != nil })
}

// toolPatchUpdates filters the collected updates down to the
// tool_call_update patches.
func toolPatchUpdates(updates []ACPUpdate) []ACPUpdate {
	return kindUpdates(updates, func(u acp.SessionUpdate) bool { return u.ToolCallUpdate != nil })
}

// testToolLifecycleEdit drives prompt 1 (the Edit lifecycle) and asserts
// the ordered client-side sequence, the log-only posture for batch and
// subagent mirrors, the absolute diff path, and the IN-05 foreign-delta
// suppression — on both the collector and the raw tee.
func testToolLifecycleEdit(t *testing.T) {
	h := newTracerFront(t, "tool-lifecycle")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openToolSession(ctx, t, h)

	resp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("edit the file")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
	}

	updates := h.Updates()

	// Exactly the client-visible frames: 4 tool frames + 1 same-turn text
	// chunk. A batch card, a subagent child card, or a foreign delta would
	// break this count.
	if got := len(updates); got != 5 {
		t.Fatalf("leg 1 collected %d updates, want exactly 5 (4 tool frames + 1 text chunk) — "+
			"batch/subagent mirrors must stay log-only and foreign deltas suppressed; got: %s",
			got, updateKinds(updates))
	}

	starts := toolUpdates(updates)
	if len(starts) != 1 {
		t.Fatalf("leg 1 collected %d tool_call starts, want exactly 1 (tc1; batch/subagent never render)", len(starts))
	}

	start := starts[0].Update.ToolCall

	if starts[0].SessionID != sid {
		t.Fatalf("tool_call session id = %q, want %q", starts[0].SessionID, sid)
	}

	if start.ToolCallId != "tc1" {
		t.Fatalf("tool_call toolCallId = %q, want tc1", start.ToolCallId)
	}

	if start.Title != "Edit main.go" {
		t.Fatalf("tool_call title = %q, want %q (title from description)", start.Title, "Edit main.go")
	}

	if start.Name == nil || *start.Name != "Edit" {
		t.Fatalf("tool_call name = %v, want Edit", start.Name)
	}

	if start.Kind != acp.ToolKindEdit {
		t.Fatalf("tool_call kind = %q, want edit (the Edit mapping row)", start.Kind)
	}

	if start.Status != acp.ToolCallStatusPending {
		t.Fatalf("tool_call status = %q, want pending (scheduled starts pending)", start.Status)
	}

	wantPath := toolLifecycleCwd + "/src/main.go"

	if len(start.Content) != 1 || start.Content[0].Diff == nil {
		t.Fatalf("tool_call carries no diff content: %s", updateKinds(updates))
	}

	diff := start.Content[0].Diff

	if diff.Path != wantPath {
		t.Fatalf("diff path = %q, want %q (absolute, resolved against the session cwd)", diff.Path, wantPath)
	}

	if diff.NewText != "new body" || diff.OldText == nil || *diff.OldText != "old body" {
		t.Fatalf("diff fields = (new %q, old %v), want (new %q, old %q)", diff.NewText, diff.OldText, "new body", "old body")
	}

	if len(start.Locations) != 1 || start.Locations[0].Path != wantPath {
		t.Fatalf("tool_call locations = %v, want [{path %q}]", start.Locations, wantPath)
	}

	// The ordered patch sequence: in_progress (started), in_progress
	// (progress), completed (result, rawOutput + diff re-emitted).
	patches := toolPatchUpdates(updates)
	if len(patches) != 3 {
		t.Fatalf("leg 1 collected %d tool_call_update patches, want exactly 3", len(patches))
	}

	wantStatus := []acp.ToolCallStatus{acp.ToolCallStatusInProgress, acp.ToolCallStatusInProgress, acp.ToolCallStatusCompleted}
	for i, want := range wantStatus {
		got := patches[i].Update.ToolCallUpdate

		if got.ToolCallId != "tc1" {
			t.Fatalf("patch[%d] toolCallId = %q, want tc1", i, got.ToolCallId)
		}

		if status, _ := got.Status.Value(); status != want {
			t.Fatalf("patch[%d] status = %v, want %q", i, got.Status, want)
		}
	}

	resultPatch := patches[2].Update.ToolCallUpdate

	raw, _ := resultPatch.RawOutput.Value()
	if string(raw) != `{"ok":true,"display":{"kind":"diff"}}` {
		t.Fatalf("result patch rawOutput = %s, want the backend result verbatim", raw)
	}

	content, _ := resultPatch.Content.Value()
	if len(content) != 1 || content[0].Diff == nil || content[0].Diff.NewText != "new body" {
		t.Fatalf("result patch must re-emit the diff (clients render early), got %s", updateKinds(updates))
	}

	// The same-turn text delta streamed; the foreign one did not.
	messages := messageUpdates(updates)
	if len(messages) != 1 || chunkText(messages[0].Update) != "done text" {
		t.Fatalf("message chunks = %s, want exactly the same-turn 'done text' delta", updateKinds(updates))
	}

	for _, line := range h.ACPLines() {
		if strings.Contains(line.JSON, "FOREIGN") {
			t.Fatalf("foreign-turnId delta reached the ACP wire (IN-05 violated): %s", line.JSON)
		}
	}

	assertToolFramesBeforeResponse(t, h, "leg 1")
}

// testToolLifecycleWrite drives prompt 2 (the Write lifecycle) and asserts
// the new-file diff shape on the raw wire: newText present, oldText
// ABSENT (the ACP new-file convention).
func testToolLifecycleWrite(t *testing.T) {
	h := newTracerFront(t, "tool-lifecycle")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sid := openToolSession(ctx, t, h)

	// Turn 1 first (the corpus's times-budgeted Edit row), so the second
	// send rule — the Write lifecycle — is the one this leg prompts.
	if _, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("edit the file")},
	}); err != nil {
		t.Fatalf("warm-up prompt: %v", err)
	}

	resp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("write the file")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
	}

	// The collector spans both prompts of this harness — the leg-2 starts
	// are the tc2 rows (the warm-up prompt's tc1 start stays in the same
	// collection, so the filter is by id, not by count).
	var starts []ACPUpdate

	for _, u := range toolUpdates(h.Updates()) {
		if u.Update.ToolCall.ToolCallId == "tc2" {
			starts = append(starts, u)
		}
	}

	if len(starts) != 1 {
		t.Fatalf("leg 2 collected %d tc2 tool_call starts, want exactly 1", len(starts))
	}

	start := starts[0].Update.ToolCall

	if start.Name == nil || *start.Name != "Write" {
		t.Fatalf("leg 2 start name = %v, want Write", start.Name)
	}

	if start.Title != "Write new.go" || start.Kind != acp.ToolKindEdit || start.Status != acp.ToolCallStatusPending {
		t.Fatalf("leg 2 start = (title %q, kind %q, status %q), want (Write new.go, edit, pending)",
			start.Title, start.Kind, start.Status)
	}

	wantPath := toolLifecycleCwd + "/docs/new.md"

	if len(start.Content) != 1 || start.Content[0].Diff == nil {
		t.Fatalf("leg 2 start carries no diff content")
	}

	diff := start.Content[0].Diff

	if diff.Path != wantPath {
		t.Fatalf("write diff path = %q, want %q (absolute, resolved against the session cwd)", diff.Path, wantPath)
	}

	if diff.NewText != "file body" {
		t.Fatalf("write diff newText = %q, want %q", diff.NewText, "file body")
	}

	if diff.OldText != nil {
		t.Fatalf("write diff oldText = %q, want nil (a new file renders no old text)", *diff.OldText)
	}

	// The raw wire truth: the tc2 tool_call frame carries newText and NOT
	// oldText — the omission is a wire property, not just a decoded nil.
	var startFrame string

	for _, line := range h.ACPLines() {
		if line.Dir == "in" && strings.Contains(line.JSON, `"toolCallId":"tc2"`) && strings.Contains(line.JSON, `"tool_call"`) {
			startFrame = line.JSON

			break
		}
	}

	if startFrame == "" {
		t.Fatal("no tool_call frame for tc2 on the tee")
	}

	if !strings.Contains(startFrame, `"newText":"file body"`) {
		t.Fatalf("tc2 tool_call frame carries no newText: %s", startFrame)
	}

	if strings.Contains(startFrame, `"oldText"`) {
		t.Fatalf("tc2 tool_call frame carries oldText — the new-file shape must omit it: %s", startFrame)
	}

	assertToolFramesBeforeResponse(t, h, "leg 2")
}

// assertToolFramesBeforeResponse checks that every tool_call /
// tool_call_update frame on the tee strictly precedes THIS leg's prompt
// response frame — the LAST stopReason line on the tee (the collector
// spans both prompts of the harness, so the anchor is the final response).
func assertToolFramesBeforeResponse(t *testing.T, h *FrontHarness, leg string) {
	t.Helper()

	lines := h.ACPLines()

	respIdxs := teeLineIndexes(lines, "in", `"stopReason"`)
	if len(respIdxs) == 0 {
		t.Fatal("no prompt response frame on the tee")
	}

	respIdx := respIdxs[len(respIdxs)-1]

	for i, line := range lines {
		if line.Dir == "in" && strings.Contains(line.JSON, `"tool_call"`) && i >= respIdx {
			t.Fatalf("%s: tool frame at line %d not before the response (line %d): %s", leg, i, respIdx, line.JSON)
		}
	}
}

// updateKinds renders the collected updates' discriminants for failure
// messages (no payload content — the structural log discipline).
func updateKinds(updates []ACPUpdate) string {
	parts := make([]string, 0, len(updates))
	for i := range updates {
		switch {
		case updates[i].Update.ToolCall != nil:
			parts = append(parts, "tool_call("+string(updates[i].Update.ToolCall.ToolCallId)+")")
		case updates[i].Update.ToolCallUpdate != nil:
			parts = append(parts, "tool_call_update("+string(updates[i].Update.ToolCallUpdate.ToolCallId)+")")
		case updates[i].Update.AgentMessageChunk != nil:
			parts = append(parts, "message")
		case updates[i].Update.AgentThoughtChunk != nil:
			parts = append(parts, "thought")
		default:
			parts = append(parts, "other")
		}
	}

	return "[" + strings.Join(parts, " ") + "]"
}
