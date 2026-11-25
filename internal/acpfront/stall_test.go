package acpfront_test

// Front-tier no-progress watchdog tests (quick task 261125-r6o, Task 2).
//
// The machine-tier watchdog resolves StopError/SignalStall with a recorded
// cause; the client-visible contract is a coded -32603 naming the stall —
// never a silent end_turn — and the front-wide slot plus the watchdog
// goroutine must be released so the next prompt on the same session runs
// normally.
//
// RED (recorded at HEAD without WithTurnStallTimeout): with the production
// 15-minute default budget the watchdog cannot fire, so the first prompt
// resolves over the production quiet window (~7.5 s) as a successful
// end_turn with ZERO session/stop frames and no coded error. GREEN composes
// with WithTurnStallTimeout(60 ms, 120 ms), far inside the exchange bound.

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	stallFrontSessionID = "sess_stall"

	// stallWatchdogBudget/stallWatchdogToolBudget are the GREEN arm's
	// shot-clock budgets: far inside the composed exchange bound.
	stallWatchdogBudget     = 60 * time.Millisecond
	stallWatchdogToolBudget = 120 * time.Millisecond
)

// stallFrontScenario scripts the two-prompt stall exchange on session/new:
// the bare create + subscribe, a first send answered only by its ack (the
// silent backend turn), the stop, a second send whose running patch and
// terminal complete normally, and the stored-reply messages projection (so
// the completion reconcile returns promptly).
func stallFrontScenario() *fake.Scenario {
	create := json.RawMessage(`{"session":{"sessionId":"` + stallFrontSessionID +
		`"},"protocol":{"name":"ZCode Protocol","version":1}}`)
	subscribe := json.RawMessage(`{"sessionId":"` + stallFrontSessionID + `","eventSeq":0,"events":[]}`)
	firstAck := json.RawMessage(`{"accepted":true,"sessionId":"` + stallFrontSessionID + `","stateRevision":1}`)
	secondAck := json.RawMessage(`{"accepted":true,"sessionId":"` + stallFrontSessionID + `","stateRevision":2}`)
	running := json.RawMessage(`{"patch":{"status":"running"},"reason":"prompt_started","revision":1,` +
		`"scope":"session","sessionId":"` + stallFrontSessionID + `","type":"state.updated"}`)
	completed := json.RawMessage(`{"type":"turn.completed","seq":9,"turnId":"turn_stall2",` +
		`"deliveryKind":"desktop-continuous"}`)
	messages := json.RawMessage(`{"messages":[{"info":{"role":"assistant"},` +
		`"parts":[{"type":"text","text":"ok"}]}],"sessionId":"` + stallFrontSessionID + `"}`)

	return &fake.Scenario{
		Name: "front-turn-stall",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: create}},
			{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{Result: subscribe}},
			{
				Method: zcode.MethodSessionSend,
				Times:  1,
				Reply:  &fake.Reply{Result: firstAck},
			},
			{Method: zcode.MethodSessionStop, Reply: &fake.Reply{Result: json.RawMessage(`{}`)}},
			{
				Method: zcode.MethodSessionSend,
				Reply:  &fake.Reply{Result: secondAck},
				SideEffects: []fake.SideEffect{
					{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: running}},
					{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: completed}},
				},
			},
			{Method: zcode.MethodSessionMessages, Reply: &fake.Reply{Result: messages}},
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: "MethodNotFound",
		}},
		EOFDrainMs: 50,
	}
}

// newStallFront composes the programmatic scenario through the full loop
// (the newTracerFront shape minus corpus loading: no model is configured,
// so session/new skips the setModel follow-up).
func newStallFront(t *testing.T, opts ...acpfront.Option) *FrontHarness {
	t.Helper()

	return ComposeFront(t, stallFrontScenario(), func(conn *zcode.Conn) acp.Agent {
		return acpport.NewAgent(acpfront.New(conn, opts...))
	})
}

// TestStalledTurnSurfacesCodedErrorAndReleasesSlot: the silent turn resolves
// as a coded -32603 naming the stall (never end_turn, never session_busy),
// and the next prompt on the same session completes — the watchdog goroutine
// joined and the front-wide slot released.
//
// RED (recorded without the option before the front seam existed): the first
// prompt resolved as a successful end_turn over the production quiet window
// at ~7.5 s with zero session/stop frames and no coded error. GREEN composes
// the seam.
//
//nolint:funlen // one composed lock drives init/new/prompt/assert end-to-end
func TestStalledTurnSurfacesCodedErrorAndReleasesSlot(t *testing.T) {
	h := newStallFront(t, acpfront.WithTurnStallTimeout(stallWatchdogBudget, stallWatchdogToolBudget))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	sid := string(newResp.SessionId)

	_, err = h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("stall")},
	})
	if err == nil {
		t.Fatal("a stalled turn must surface a coded error, got a success response (the silent end_turn shape)")
	}

	var reqErr *acp.RequestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("stalled prompt error = %T (%v), want a coded JSON-RPC RequestError", err, err)
	}

	if reqErr.Code != acpport.CodeInternal {
		t.Fatalf("stalled prompt code = %d, want %d (internal class); message %q",
			reqErr.Code, acpport.CodeInternal, reqErr.Message)
	}

	if !strings.Contains(reqErr.Message, "turn stalled") {
		t.Fatalf("stalled prompt message = %q, want it to name the stall", reqErr.Message)
	}

	if strings.Contains(reqErr.Message, "session_busy") || reqErr.Code == acpport.CodeSessionBusy {
		t.Fatalf("stalled prompt surfaced the session_busy class (%d: %q) — the stall must be non-retryable",
			reqErr.Code, reqErr.Message)
	}

	// The slot and watchdog were released: the next prompt completes.
	resp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("after stall")},
	})
	if err != nil {
		t.Fatalf("second prompt after the stall: %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("second prompt stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if sends := len(transcriptIndexes(tr, "in", `"method":"session/send"`)); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2", sends)
	}

	if stops := len(transcriptIndexes(tr, "in", `"method":"session/stop"`)); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1 (the watchdog stop)", stops)
	}
}
