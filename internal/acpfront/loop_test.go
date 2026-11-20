package acpfront_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// TestMinimalChatLoop is the plan 04-06 TRACER (the phase's first vertical
// integration): one real SDK ACP client through acpport → acpfront → the
// Phase-3 turn machine → the zcode.Conn → the fake app-server, asserted on
// RECORDED wire transcripts at both legs. Staged asserts:
//
//	happy leg (setmodel-followup corpus):
//	  initialize honest D-02 shape (on the raw tee frame),
//	  session/new D-03 back-leg sequence (create → subscribe → setModel),
//	  prompt streams TWO message chunks BEFORE the response frame and
//	  answers stopReason end_turn,
//	cancel leg (extended cancel-mid-turn corpus from Task 1):
//	  cancel mid-turn answers exactly ONE prompt response with
//	  stopReason cancelled, NEVER a -32800 error anywhere on the tee,
//	  and a second prompt while the turn is active fails cleanly (an
//	  error frame — the exact class is plan 04-05's RED target).
//
// The corpus-locked synthetic selection (zai-api/GLM-5.3-Flash, level
// high) is configured on the Front — the setModel follow-up payload must
// match the recorded goldens byte for byte.

const (
	// tracerScenarioDir reaches the corpus from this package's test dir.
	tracerScenarioDir = "../zcode/fake/testdata/scenarios/"

	// tracerWait bounds each in-stage wait (turn start, prompt resolution);
	// the composed loop completes in milliseconds, so this fails fast.
	tracerWait = 5 * time.Second

	// The tee rides the CLIENT side of the composition (harness_test.go:
	// NewTee wraps the client's read/write legs), so its directions are the
	// client's: "out" = client→agent frames (requests), "in" =
	// agent→client frames (responses and session/update notifications).
	// The fake transcript directions are the host's: "in" = host→fake
	// (back-leg requests), "out" = fake→host.
)

// TestMinimalChatLoop runs the full minimal loop end-to-end under -race.
func TestMinimalChatLoop(t *testing.T) {
	t.Run("happy turn: honest initialize, D-03 session/new, chunks before stopReason", testHappyTurnStage)
	t.Run("cancel mid-turn: cancelled exactly once, no -32800, second prompt rejected", testCancelMidTurnStage)
}

// newTracerFront composes the full loop over a corpus scenario with the
// Front wrapped via acpport.NewAgent (the plan 04-02 harness's intended
// consumer shape — the Front owns the back-leg conn and is constructed
// inside the conn-carrying callback, before any frame flows).
func newTracerFront(t *testing.T, scenario string) *FrontHarness {
	t.Helper()

	sc := mustLoadScenario(t, scenario)

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(acpfront.ModelSelection{
				ProviderID:     "zai-api",
				ModelID:        "GLM-5.3-Flash",
				ReasoningLevel: "high",
			}),
		)

		return acpport.NewAgent(front)
	})

	return h
}

// mustLoadScenario loads a corpus scenario (the strict fail-loud loader).
func mustLoadScenario(t *testing.T, name string) *fake.Scenario {
	t.Helper()

	sc, err := fake.LoadScenario(tracerScenarioDir + name + ".json")
	if err != nil {
		t.Fatalf("load scenario %s: %v", name, err)
	}

	return sc
}

// teeLineIndexes returns the indexes of every line in one direction whose
// raw JSON contains substr, in recorded (wire) order.
func teeLineIndexes(lines []TeeLine, dir, substr string) []int {
	indexes := []int{}
	for i, line := range lines {
		if line.Dir == dir && strings.Contains(line.JSON, substr) {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

// teeLineIndex returns the FIRST matching line index, failing the test when
// absent.
func teeLineIndex(t *testing.T, lines []TeeLine, dir, substr, what string) int {
	t.Helper()

	indexes := teeLineIndexes(lines, dir, substr)
	if len(indexes) == 0 {
		t.Fatalf("no %s frame (%s lines matching %q) on the ACP tee", what, dir, substr)
	}

	return indexes[0]
}

// testHappyTurnStage — the streaming leg over the setmodel-followup corpus.
func testHappyTurnStage(t *testing.T) {
	h := newTracerFront(t, "setmodel-followup")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	// -- initialize: the honest D-02 report, asserted on the RECORDED tee
	// frame (not the decoded struct — the wire truth is the contract).
	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	lines := h.ACPLines()
	initIdx := teeLineIndex(t, lines, "in", `"protocolVersion"`, "initialize response")
	initRaw := lines[initIdx].JSON

	// Honesty by omission (D-02, A7): assert on the ABSENT features, not on
	// empty-object cosmetics. fs/terminal are client-side capabilities and
	// loadSession is simply never advertised by this build.
	for _, banned := range []string{`"loadSession"`, `"terminal"`, `"fs"`, `"readTextFile"`, `"writeTextFile"`} {
		if strings.Contains(initRaw, banned) {
			t.Fatalf("initialize response advertises %s (D-02 honesty violation): %s", banned, initRaw)
		}
	}

	// promptCapabilities carries NO modes: the empty object is the whole
	// surface this build offers (the executed-probe marshal shape).
	if !strings.Contains(initRaw, `"promptCapabilities":{}`) {
		t.Fatalf("promptCapabilities must marshal empty (no modes advertised): %s", initRaw)
	}

	var initFrame struct {
		Result struct {
			ProtocolVersion int `json:"protocolVersion"`
			AgentInfo       struct {
				Name string `json:"name"`
			} `json:"agentInfo"`
			AuthMethods []map[string]any `json:"authMethods"`
		} `json:"result"`
	}

	if err := json.Unmarshal([]byte(initRaw), &initFrame); err != nil {
		t.Fatalf("decode initialize response frame: %v", err)
	}

	if initFrame.Result.ProtocolVersion != 1 {
		t.Fatalf("protocolVersion = %d, want 1", initFrame.Result.ProtocolVersion)
	}

	if initFrame.Result.AgentInfo.Name != "zcode-acp" {
		t.Fatalf("agentInfo.name = %q, want %q", initFrame.Result.AgentInfo.Name, "zcode-acp")
	}

	wantAuth := []struct{ id, name string }{
		{"zai-api-key", "z.ai API key"},
		{"zai-individual-coding-plan", "z.ai Individual Coding Plan"},
	}

	if len(initFrame.Result.AuthMethods) != len(wantAuth) {
		t.Fatalf("authMethods = %d entries, want %d", len(initFrame.Result.AuthMethods), len(wantAuth))
	}

	for i, want := range wantAuth {
		entry := initFrame.Result.AuthMethods[i]

		if entry["id"] != want.id || entry["name"] != want.name {
			t.Fatalf("authMethods[%d] = %v, want id %q name %q", i, entry, want.id, want.name)
		}

		// Agent variant: marshals WITHOUT a type field (the executed-probe
		// shape — the agent variant is the default).
		if _, hasType := entry["type"]; hasType {
			t.Fatalf("authMethods[%d] carries a type field (agent variant must omit it): %v", i, entry)
		}
	}

	// -- session/new: D-03 — accepted with an MCP list (ignored) and the
	// absolute cwd; the D-03 back-leg sequence is asserted on the fake
	// transcript below.
	cwd := t.TempDir()

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd: cwd,
		McpServers: []acp.McpServer{{
			Stdio: &acp.McpServerStdio{Name: "tracer-mcp", Command: "noop"},
		}},
	})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	// -- prompt: two live text chunks, then stopReason end_turn.
	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}

	// The client's collected updates: exactly the two text deltas in
	// arrival order, zero thought chunks (CORE-04 negative half: no
	// reasoning deltas in this corpus → no thought chunks, nothing
	// synthesized).
	updates := h.Updates()
	if len(updates) != 2 {
		t.Fatalf("client collected %d updates, want exactly 2 message chunks", len(updates))
	}

	wantDeltas := []string{"Hello", " world"}
	for i, want := range wantDeltas {
		if updates[i].SessionID != string(newResp.SessionId) {
			t.Fatalf("update[%d] session = %q, want %q", i, updates[i].SessionID, string(newResp.SessionId))
		}

		if got := chunkText(updates[i].Update); got != want {
			t.Fatalf("update[%d] text = %q, want %q", i, got, want)
		}
	}

	// Wire order on the ACP tee (CORE-03/D-04): the two chunk frames are
	// on the wire BEFORE the prompt response frame, and NOTHING streams
	// after the response (flush-before-stopReason).
	lines = h.ACPLines()

	chunkIdxs := teeLineIndexes(lines, "in", `"agent_message_chunk"`)
	if len(chunkIdxs) != 2 {
		t.Fatalf("tee records %d agent_message_chunk frames, want 2", len(chunkIdxs))
	}

	respIdx := teeLineIndex(t, lines, "in", `"end_turn"`, "prompt response")
	for i, idx := range chunkIdxs {
		if idx >= respIdx {
			t.Fatalf("chunk frame %d (line %d) is not before the response frame (line %d) — flush-before-stopReason violated", i, idx, respIdx)
		}
	}

	afterResp := 0
	for _, line := range lines[respIdx+1:] {
		if line.Dir == "in" && strings.Contains(line.JSON, `"sessionUpdate"`) {
			afterResp++
		}
	}

	if afterResp != 0 {
		t.Fatalf("%d update frames stream after the prompt response frame — stopReason must land only after the flush", afterResp)
	}

	if thoughtIdxs := teeLineIndexes(lines, "in", `"agent_thought_chunk"`); len(thoughtIdxs) != 0 {
		t.Fatalf("tee records %d agent_thought_chunk frames, want 0 (no reasoning deltas in this corpus — none may be synthesized)", len(thoughtIdxs))
	}

	// -- back leg: the D-03 sequence in wire order, asserted on the fake
	// transcript (the app-server's view of the front's session/new).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	createIdx := transcriptIndex(t, tr, "in", `"method":"session/create"`, "create request")
	subIdx := transcriptIndex(t, tr, "in", `"method":"session/subscribe"`, "subscribe request")
	setModelIdx := transcriptIndex(t, tr, "in", `"method":"session/setModel"`, "setModel request")
	sendIdx := transcriptIndex(t, tr, "in", `"method":"session/send"`, "send request")

	if !(createIdx < subIdx && subIdx < setModelIdx && setModelIdx < sendIdx) {
		t.Fatalf("back-leg wire order violated: create(%d) subscribe(%d) setModel(%d) send(%d) — want create → subscribe → setModel → send", createIdx, subIdx, setModelIdx, sendIdx)
	}

	var createFrame struct {
		Params struct {
			Workspace struct {
				WorkspacePath string `json:"workspacePath"`
				WorkspaceKey  string `json:"workspaceKey"`
			} `json:"workspace"`
			Model json.RawMessage `json:"model"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(tr.Lines[createIdx].JSON), &createFrame); err != nil {
		t.Fatalf("decode create frame: %v", err)
	}

	if createFrame.Params.Workspace.WorkspacePath != cwd || createFrame.Params.Workspace.WorkspaceKey != cwd {
		t.Fatalf("create workspace = %s (path %q key %q), want both %q", tr.Lines[createIdx].JSON, createFrame.Params.Workspace.WorkspacePath, createFrame.Params.Workspace.WorkspaceKey, cwd)
	}

	// Bare create (Pitfall 4): create serializes the selection to
	// providerId/modelId and drops options.reasoningLevel — the front must
	// not send a model object at all.
	if len(createFrame.Params.Model) != 0 && !jsonIsNull(createFrame.Params.Model) {
		t.Fatalf("bare create must carry no model object (Pitfall 4): %s", tr.Lines[createIdx].JSON)
	}

	assertSubscribeFrame(t, tr.Lines[subIdx].JSON)
	assertSetModelFrame(t, tr.Lines[setModelIdx].JSON)
}

// testCancelMidTurnStage — the cancel leg over the extended cancel-mid-turn
// corpus (Task 1): cancel mid-turn, cancelled exactly once, no -32800
// anywhere, second-prompt-while-active rejected cleanly.
func testCancelMidTurnStage(t *testing.T) {
	h := newTracerFront(t, "cancel-mid-turn")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	// The stage's own back-leg observer — registered AFTER the front's tap
	// and the session machine (ComposeFront constructed both), so the
	// moment turn.started reaches this channel the machine is already
	// classifying a running turn and the front-wide slot is held.
	turnStarted := make(chan struct{}, 1)

	removeObserver := h.conn.AddObserver(func(env zcode.Envelope) {
		if env.Method == zcode.MethodSessionEvent && strings.Contains(string(env.Params), `"turn.started"`) {
			select {
			case turnStarted <- struct{}{}:
			default:
			}
		}
	})
	defer removeObserver()

	// Prompt #1 in flight — the turn stays active on this corpus (no
	// terminal event until the cancel).
	type promptOutcome struct {
		resp acp.PromptResponse
		err  error
	}

	promptDone := make(chan promptOutcome, 1)

	go func() {
		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: newResp.SessionId,
			Prompt:    []acp.ContentBlock{acp.TextBlock("hi")},
		})
		promptDone <- promptOutcome{resp: resp, err: err}
	}()

	select {
	case <-turnStarted:
	case <-time.After(tracerWait):
		t.Fatal("turn never started — no turn.started on the back leg within the wait bound")
	}

	// The second prompt while the first is active fails CLEANLY: a JSON-RPC
	// error response, never a hang and never a second concurrent turn on
	// the wire (asserted below as exactly one session/send). The exact
	// error class is plan 04-05's RED target — here it only must BE an
	// error.
	if _, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: newResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("second")},
	}); err == nil {
		t.Fatal("second prompt while a turn is active must return an error, got success")
	}

	// Cancel mid-turn (D-05).
	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: newResp.SessionId}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	var outcome promptOutcome

	select {
	case outcome = <-promptDone:
	case <-time.After(tracerWait):
		t.Fatal("cancelled prompt never resolved within the wait bound")
	}

	// Cancelled is a SUCCESS stopReason — never a -32800 error (Pitfall 2).
	if outcome.err != nil {
		t.Fatalf("cancelled prompt must answer as a success response, got error: %v", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stopReason = %q, want %q", outcome.resp.StopReason, acp.StopReasonCancelled)
	}

	// Exactly ONE prompt response on the whole tee (exactly-once), and it
	// is the cancelled one.
	lines := h.ACPLines()

	stopIdxs := teeLineIndexes(lines, "in", `"stopReason"`)
	if len(stopIdxs) != 1 {
		t.Fatalf("tee records %d stopReason frames, want exactly 1 (the prompt answer resolves exactly once)", len(stopIdxs))
	}

	if !strings.Contains(lines[stopIdxs[0]].JSON, `"cancelled"`) {
		t.Fatalf("the prompt response = %s, want stopReason cancelled", lines[stopIdxs[0]].JSON)
	}

	// NO -32800 error anywhere on the ACP wire — neither direction, no
	// frame.
	for _, line := range lines {
		if strings.Contains(line.JSON, "-32800") {
			t.Fatalf("found a -32800 frame on the tee (cancel must resolve as a success stopReason): %s", line.JSON)
		}
	}

	// The second prompt's rejection IS an error frame on the wire.
	if len(teeLineIndexes(lines, "in", `"error"`)) == 0 {
		t.Fatal("the rejected second prompt left no error frame on the tee")
	}

	// Back leg: exactly ONE session/send — the rejected prompt never
	// reached the app-server (single-active-turn invariant).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if sends := len(transcriptIndexes(tr, "in", `"method":"session/send"`)); sends != 1 {
		t.Fatalf("back leg records %d session/send frames, want exactly 1 (the second prompt must be rejected before the wire)", sends)
	}
}

// chunkText digs the text out of a collected session update (both chunk
// helpers carry the text in a text content block).
func chunkText(update acp.SessionUpdate) string {
	switch {
	case update.AgentMessageChunk != nil:
		if block := update.AgentMessageChunk.Content.Text; block != nil {
			return block.Text
		}
	case update.AgentThoughtChunk != nil:
		if block := update.AgentThoughtChunk.Content.Text; block != nil {
			return block.Text
		}
	}

	return ""
}

// transcriptIndex returns the first transcript line index in one direction
// whose JSON contains substr, failing the test when absent.
func transcriptIndex(t *testing.T, tr fake.Transcript, dir, substr, what string) int {
	t.Helper()

	indexes := transcriptIndexes(tr, dir, substr)
	if len(indexes) == 0 {
		t.Fatalf("no %s (%s lines matching %q) on the fake transcript", what, dir, substr)
	}

	return indexes[0]
}

// transcriptIndexes returns every matching transcript line index in wire
// order.
func transcriptIndexes(tr fake.Transcript, dir, substr string) []int {
	indexes := []int{}
	for i, line := range tr.Lines {
		if string(line.Dir) == dir && strings.Contains(line.JSON, substr) {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

// jsonIsNull reports whether a raw JSON value is the null literal.
func jsonIsNull(raw json.RawMessage) bool {
	return string(raw) == "null"
}

// assertSubscribeFrame checks the subscribe request's wire payload
// (dialect §4): the desktop-continuous delivery kind is the session/event
// push gate (RESEARCH Pitfall 1).
func assertSubscribeFrame(t *testing.T, raw string) {
	t.Helper()

	var frame struct {
		Params struct {
			SessionID    string `json:"sessionId"`
			DeliveryKind string `json:"deliveryKind"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode subscribe frame: %v", err)
	}

	if frame.Params.DeliveryKind != "desktop-continuous" {
		t.Fatalf("subscribe deliveryKind = %q, want desktop-continuous: %s", frame.Params.DeliveryKind, raw)
	}
}

// assertSetModelFrame checks the setModel follow-up's corpus-locked payload
// (D-03; Open Question 3 resolution: explicit persistAsWorkspaceLastUsed
// false; the reasoning level rides model.options — create drops it, the
// refusal trap of dialect §2).
func assertSetModelFrame(t *testing.T, raw string) {
	t.Helper()

	var frame struct {
		Params struct {
			SessionID string `json:"sessionId"`
			Model     struct {
				ProviderID string `json:"providerId"`
				ModelID    string `json:"modelId"`
				Options    struct {
					ReasoningLevel string `json:"reasoningLevel"`
				} `json:"options"`
			} `json:"model"`
			PersistAsWorkspaceLastUsed *bool `json:"persistAsWorkspaceLastUsed"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode setModel frame: %v", err)
	}

	if frame.Params.Model.ProviderID != "zai-api" || frame.Params.Model.ModelID != "GLM-5.3-Flash" {
		t.Fatalf("setModel model = %s/%s, want zai-api/GLM-5.3-Flash (corpus-locked): %s", frame.Params.Model.ProviderID, frame.Params.Model.ModelID, raw)
	}

	if frame.Params.Model.Options.ReasoningLevel != "high" {
		t.Fatalf("setModel reasoningLevel = %q, want high (the refusal trap — level rides model.options): %s", frame.Params.Model.Options.ReasoningLevel, raw)
	}

	if frame.Params.PersistAsWorkspaceLastUsed == nil || *frame.Params.PersistAsWorkspaceLastUsed {
		t.Fatalf("setModel persistAsWorkspaceLastUsed must be explicitly false (Open Question 3): %s", raw)
	}
}
