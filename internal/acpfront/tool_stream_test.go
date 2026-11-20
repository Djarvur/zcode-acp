package acpfront_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
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

	// Coalescing-aware composition lock (WR-03, same plan): the per-turn
	// pump may coalesce same-toolCallId patches keep-latest under drain
	// pressure, so the intermediate in_progress count is pressure-dependent
	// — the deterministic invariants are the ORDER (start card → patches →
	// same-turn text), exactly ONE start card (batch/subagent mirrors never
	// render), the terminal completed patch, the lossless text delta, and
	// the foreign delta's absence. The byte-level ORDER of the backend
	// envelopes is golden-locked at the fake tier.

	assertEditStartCard(t, updates, sid)

	assertEditPatchSequence(t, updates)
	assertForeignSuppression(t, h)
	assertToolFramesBeforeResponse(t, h, "leg 1")
}

// assertEditPatchSequence checks leg 1's patch stream in lifecycle order:
// any in_progress patches the drain delivered precede the terminal
// completed patch (keep-latest never drops the LAST same-id note — the
// completed result is always delivered, and it is always the last patch),
// with the rawOutput verbatim and the diff re-emitted (clients render
// early; ACP update content merges per spec).
func assertEditPatchSequence(t *testing.T, updates []ACPUpdate) {
	t.Helper()

	patches := toolPatchUpdates(updates)
	if len(patches) == 0 {
		t.Fatalf("leg 1 collected zero tool_call_update patches — at least the terminal patch must stream; got: %s",
			updateKinds(updates))
	}

	for i := range patches {
		got := patches[i].Update.ToolCallUpdate

		if got.ToolCallId != permToolCallID {
			t.Fatalf("patch[%d] toolCallId = %q, want %q", i, got.ToolCallId, permToolCallID)
		}

		want := acp.ToolCallStatusInProgress
		if i == len(patches)-1 {
			want = acp.ToolCallStatusCompleted
		}

		if status, _ := got.Status.Value(); status != want {
			t.Fatalf("patch[%d] status = %q, want %q (lifecycle order; completed is the last patch)", i, status, want)
		}
	}

	resultPatch := patches[len(patches)-1].Update.ToolCallUpdate

	raw, _ := resultPatch.RawOutput.Value()
	if string(raw) != `{"ok":true,"display":{"kind":"diff"}}` {
		t.Fatalf("result patch rawOutput = %s, want the backend result verbatim", raw)
	}

	content, _ := resultPatch.Content.Value()
	if len(content) != 1 || content[0].Diff == nil || content[0].Diff.NewText != "new body" {
		t.Fatalf("result patch must re-emit the diff (clients render early), got %s", updateKinds(updates))
	}
}

// assertForeignSuppression checks the same-turn text delta streamed
// losslessly and the foreign-turnId delta never reached the wire (IN-05).
func assertForeignSuppression(t *testing.T, h *FrontHarness) {
	t.Helper()

	updates := h.Updates()

	messages := messageUpdates(updates)
	if len(messages) != 1 || chunkText(messages[0].Update) != "done text" {
		t.Fatalf("message chunks = %s, want exactly the same-turn 'done text' delta", updateKinds(updates))
	}

	for _, line := range h.ACPLines() {
		if strings.Contains(line.JSON, "FOREIGN") {
			t.Fatalf("foreign-turnId delta reached the ACP wire (IN-05 violated): %s", line.JSON)
		}
	}
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

	assertWriteStartCard(t, h)

	assertWriteStartWire(t, h)

	assertToolFramesBeforeResponse(t, h, "leg 2")
}

// assertWriteStartCard checks leg 2's single tc2 tool_call start: the
// Write mapping row (kind edit, pending) and the new-file diff recipe —
// absolute path, newText only, oldText nil on the decoded surface.
func assertWriteStartCard(t *testing.T, h *FrontHarness) {
	t.Helper()

	// The collector spans both prompts of this harness — the leg-2 starts
	// are the tc2 rows (the warm-up prompt's tc1 start stays in the same
	// collection, so the filter is by id, not by count).
	var starts []ACPUpdate

	tc2Starts := toolUpdates(h.Updates())
	for i := range tc2Starts {
		if tc2Starts[i].Update.ToolCall.ToolCallId == "tc2" {
			starts = append(starts, tc2Starts[i])
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

// WR-03 backpressure locks (plan 05-04 Task 3): the per-turn pump's three
// legs, driven through a back-leg-only composition (fake ⇄ conn ⇄ Front)
// with the adversarial client pace injected AT the Front.Prompt sink
// boundary — a stalled sink is exactly the condition the pump design must
// survive without wedging the backend leg.
//
//	backpressureSettle bounds the fake's scripted exchange (milliseconds in
//	practice) — everything the scenario emits is enqueued behind a parked
//	drain before the leg's next step runs.
//	backpressureWait bounds the prompt's resolution: with the pump, a
//	stalled sink cannot hold the turn past release's bounded drain; an
//	inline emitter (the WR-03 defect) parks the read loop forever.
const (
	backpressureSettle = 500 * time.Millisecond
	backpressureWait   = 8 * time.Second
)

// backLegHarness composes the back leg only: fake.Serve ⇄ zcode.Conn ⇄
// acpfront.Front — no ACP leg (the WR-03 legs own the sink).
type backLegHarness struct {
	front  *acpfront.Front
	conn   *zcode.Conn
	served <-chan fakeServeResult
}

// composeBackLeg wires the back leg with the corpus scenario and optional
// Front options; everything tears down via t.Cleanup.
func composeBackLeg(t *testing.T, scenario string, opts ...acpfront.Option) *backLegHarness {
	t.Helper()

	sc := mustLoadScenario(t, scenario)

	h := &backLegHarness{}

	toFakeR, toFakeW := io.Pipe()
	toHostR, toHostW := io.Pipe()

	hostEnd := &rwCloser{r: toHostR, w: toFakeW}
	fakeEnd := &rwCloser{r: toFakeR, w: toHostW}

	t.Cleanup(func() { _ = fakeEnd.Close() })

	serveDone := make(chan fakeServeResult, 1)

	go func() {
		tr, serveErr := fake.Serve(fakeEnd, sc)
		serveDone <- fakeServeResult{transcript: tr, err: serveErr}
	}()

	h.conn = zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = h.conn.Close() })

	h.served = serveDone
	h.front = acpfront.New(h.conn, opts...)

	return h
}

// openSession runs the D-03 contract on the back leg and returns the id.
func (h *backLegHarness) openSession(ctx context.Context, t *testing.T) string {
	t.Helper()

	sid, err := h.front.NewSession(ctx, acpport.NewSessionParams{Cwd: toolLifecycleCwd})
	if err != nil {
		t.Fatalf("front.NewSession: %v", err)
	}

	return sid
}

// TestStreamBackpressure locks the WR-03 designed policy (the OQ3
// disposition):
//
//	stalled sink: with the sink parked on a gate, the BACKEND read loop
//	  keeps consuming envelopes — the machine resolves and Prompt returns
//	  end_turn WHILE the gate is still closed (the WR-03 defect — the
//	  inline ACP write on the read goroutine — parks the read loop and the
//	  turn never resolves).
//	coalescing: tool_call_update patches for the SAME toolCallId coalesce
//	  keep-latest while the drain is parked — no intermediate update frame
//	  for that toolCallId reaches the client.
//	ceiling: a tiny injectable byte ceiling makes the first oversized delta
//	  FAIL THE TURN LOUD — Prompt returns the coded prompt error naming the
//	  cause; nothing partial reaches the client, nothing hangs.
func TestStreamBackpressure(t *testing.T) {
	t.Run("stalled sink: the backend read loop keeps consuming", testBackpressureReadLoopLive)
	t.Run("same-tool updates coalesce keep-latest behind a parked drain", testBackpressureCoalesce)
	t.Run("byte ceiling fails the turn loud", testBackpressureCeiling)
}

// testBackpressureReadLoopLive parks the sink on a gate BEFORE the prompt
// and asserts the prompt resolves end_turn while the gate is still closed
// — the machine saw the terminal event, so the read loop never parked on
// the stalled ACP write.
func testBackpressureReadLoopLive(t *testing.T) {
	h := composeBackLeg(t, "part-delta-streaming")

	ctx, cancel := context.WithTimeout(context.Background(), backpressureWait)
	defer cancel()

	sid := h.openSession(ctx, t)

	gate := make(chan struct{})
	defer close(gate) // unwind the parked emit at leg end

	var invoked atomic.Int32

	sink := func(_ context.Context, _ acpport.SessionNote) error {
		invoked.Add(1)
		<-gate // the adversarial stalled client: the first write parks here

		return nil
	}

	promptDone := make(chan gatedPromptResult, 1)

	go func() {
		promptDone <- driveGatedPrompt(ctx, t, h, sid, sink)
	}()

	time.Sleep(backpressureSettle) // the whole scripted exchange enqueues behind the park

	// The proof: resolved while the gate is still closed.
	res := <-promptDone
	if res.timedOut {
		t.Fatal("prompt never resolved while the sink gate stayed closed — " +
			"the backend read loop parked on the stalled ACP write (the WR-03 defect)")
	}

	if res.err != nil {
		t.Fatalf("stalled-sink turn errored: %v", res.err)
	}

	if res.reason != acpport.StopEndTurn {
		t.Fatalf("stalled-sink turn resolved %q, want end_turn (the machine consumed the terminal event)", res.reason)
	}

	if got := invoked.Load(); got != 1 {
		t.Fatalf("sink invoked %d times while parked, want exactly 1 (the drain holds at the park)", got)
	}
}

// testBackpressureCoalesce parks the drain on the tc1 start card, lets the
// full Edit lifecycle enqueue behind it, then releases: of tc1's three
// patches (in_progress, in_progress, completed) only the LAST reaches the
// client — keep-latest, no intermediate update frame for that toolCallId.
func testBackpressureCoalesce(t *testing.T) {
	h := composeBackLeg(t, "tool-lifecycle")

	ctx, cancel := context.WithTimeout(context.Background(), backpressureWait)
	defer cancel()

	sid := h.openSession(ctx, t)

	gate := make(chan struct{})

	collector := newStartParkingCollector(gate)

	promptDone := make(chan gatedPromptResult, 1)

	go func() {
		promptDone <- driveGatedPrompt(ctx, t, h, sid, collector.sink)
	}()

	time.Sleep(backpressureSettle) // scheduled + started + progress + result all enqueue
	close(gate)                    // release the drain: keep-latest delivers

	var res gatedPromptResult

	select {
	case res = <-promptDone:
	case <-time.After(backpressureWait):
		t.Fatal("prompt never resolved after the gate opened")
	}

	if res.err != nil {
		t.Fatalf("coalescing turn errored: %v", res.err)
	}

	if res.reason != acpport.StopEndTurn {
		t.Fatalf("coalescing turn resolved %q, want end_turn", res.reason)
	}

	var patches []string

	for _, note := range collector.snapshot() {
		if !isTc1Patch(&note) {
			continue
		}

		patches = append(patches, note.Tool.Status)
	}

	if len(patches) != 1 {
		t.Fatalf("tc1 delivered %d tool_call_update frames (%v), want exactly 1 — intermediate "+
			"same-toolCallId updates must coalesce keep-latest behind a parked drain", len(patches), patches)
	}

	if patches[0] != acpport.ToolStatusCompleted {
		t.Fatalf("the surviving tc1 patch status = %q, want completed (the last update wins)", patches[0])
	}
}

// testBackpressureCeiling configures a tiny injectable ceiling; the first
// oversized delta ("Answer:", 7 bytes > 4) must fail the turn LOUD — a
// coded -32603 prompt error naming the ceiling cause, nothing delivered,
// nothing hanging.
func testBackpressureCeiling(t *testing.T) {
	h := composeBackLeg(t, "part-delta-streaming", acpfront.WithStreamCeiling(4))

	ctx, cancel := context.WithTimeout(context.Background(), backpressureWait)
	defer cancel()

	sid := h.openSession(ctx, t)

	var delivered atomic.Int32

	sink := func(_ context.Context, _ acpport.SessionNote) error {
		delivered.Add(1)

		return nil
	}

	reason, err := h.front.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: "hi"}, sink)
	if err == nil {
		t.Fatalf("ceiling breach resolved %q with nil error — the breach must FAIL THE TURN LOUD", reason)
	}

	var seamErr *acpport.Error
	if !errors.As(err, &seamErr) {
		t.Fatalf("ceiling breach error = %T (%v), want a coded *acpport.Error", err, err)
	}

	if seamErr.Code != acpport.CodeInternal {
		t.Fatalf("ceiling breach error code = %d, want %d (-32603 internal)", seamErr.Code, acpport.CodeInternal)
	}

	if !strings.Contains(seamErr.Message, "ceiling") {
		t.Fatalf("ceiling breach error must name the cause, got %q", seamErr.Message)
	}

	if got := delivered.Load(); got != 0 {
		t.Fatalf("%d notes were delivered past a breached ceiling — the breach must fail the turn, "+
			"never stream a partial", got)
	}
}

// assertEditStartCard checks leg 1's single tc1 tool_call start: the
// title/kind/status mapping row and the diff recipe (absolute path,
// old/new text) with the location — the Pattern-4 mapping table at the
// composed tier.
func assertEditStartCard(t *testing.T, updates []ACPUpdate, sid string) {
	t.Helper()

	starts := toolUpdates(updates)
	if len(starts) != 1 {
		t.Fatalf("leg 1 collected %d tool_call starts, want exactly 1 "+
			"(batch/subagent mirrors never render)", len(starts))
	}

	start := starts[0].Update.ToolCall

	if starts[0].SessionID != sid {
		t.Fatalf("tool_call session id = %q, want %q", starts[0].SessionID, sid)
	}

	if start.ToolCallId != permToolCallID {
		t.Fatalf("tool_call toolCallId = %q, want %q", start.ToolCallId, permToolCallID)
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

	assertEditStartDiff(t, start, updates)
}

// assertEditStartDiff checks leg 1's start-card diff recipe: the absolute
// session-cwd-resolved path, the old/new texts, and the location.
func assertEditStartDiff(t *testing.T, start *acp.SessionUpdateToolCall, updates []ACPUpdate) {
	t.Helper()

	wantPath := toolLifecycleCwd + "/src/main.go"

	if len(start.Content) != 1 || start.Content[0].Diff == nil {
		t.Fatalf("tool_call carries no diff content: %s", updateKinds(updates))
	}

	diff := start.Content[0].Diff

	if diff.Path != wantPath {
		t.Fatalf("diff path = %q, want %q (absolute, resolved against the session cwd)", diff.Path, wantPath)
	}

	if diff.NewText != "new body" || diff.OldText == nil || *diff.OldText != "old body" {
		t.Fatalf("diff fields = (new %q, old %v), want (new %q, old %q)",
			diff.NewText, diff.OldText, "new body", "old body")
	}

	if len(start.Locations) != 1 || start.Locations[0].Path != wantPath {
		t.Fatalf("tool_call locations = %v, want [{path %q}]", start.Locations, wantPath)
	}
}

// assertWriteStartWire checks the raw tee for the tc2 tool_call frame: the
// new-file shape carries newText and NOT oldText — the omission is a wire
// property, not just a decoded nil.
func assertWriteStartWire(t *testing.T, h *FrontHarness) {
	t.Helper()

	var startFrame string

	for _, line := range h.ACPLines() {
		isToolStart := line.Dir == "in" &&
			strings.Contains(line.JSON, `"toolCallId":"tc2"`) &&
			strings.Contains(line.JSON, `"tool_call"`)

		if isToolStart {
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
}

// gatedPromptResult is one stalled-sink prompt's resolution.
type gatedPromptResult struct {
	reason   acpport.StopReason
	err      error
	timedOut bool
}

// driveGatedPrompt runs one prompt on its own goroutine and waits bounded
// for its resolution — the stalled-sink legs' shared driver.
func driveGatedPrompt(
	ctx context.Context,
	t *testing.T,
	h *backLegHarness,
	sid string,
	sink acpport.UpdateSink,
) gatedPromptResult {
	t.Helper()

	done := make(chan gatedPromptResult, 1)

	go func() {
		reason, err := h.front.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: "hi"}, sink)
		done <- gatedPromptResult{reason: reason, err: err}
	}()

	select {
	case res := <-done:
		return res
	case <-time.After(backpressureWait):
		return gatedPromptResult{timedOut: true}
	}
}

// isTc1Patch reports whether the note is one of tc1's tool_call_update
// patches (the coalescing leg's filter).
func isTc1Patch(note *acpport.SessionNote) bool {
	return note.Kind == acpport.NoteTool &&
		note.Tool != nil &&
		!note.Tool.IsStart &&
		note.Tool.ToolCallID == permToolCallID
}

// startParkingCollector is the coalescing leg's sink: it records every
// note it is handed and PARKS the drain on the first start card (the
// adversarial client holds the wire while the lifecycle enqueues).
type startParkingCollector struct {
	gate <-chan struct{}

	mu   sync.Mutex
	seen []acpport.SessionNote
}

// newStartParkingCollector builds the collector around one gate channel.
func newStartParkingCollector(gate <-chan struct{}) *startParkingCollector {
	return &startParkingCollector{gate: gate}
}

// sink is the acpport.UpdateSink: record, park on the start card.
func (c *startParkingCollector) sink(_ context.Context, note acpport.SessionNote) error {
	c.mu.Lock()
	c.seen = append(c.seen, note)
	c.mu.Unlock()

	if note.Kind == acpport.NoteTool && note.Tool != nil && note.Tool.IsStart {
		<-c.gate // park the drain on the start card: everything else queues
	}

	return nil
}

// snapshot copies the recorded notes in arrival order.
func (c *startParkingCollector) snapshot() []acpport.SessionNote {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]acpport.SessionNote, len(c.seen))
	copy(out, c.seen)

	return out
}
