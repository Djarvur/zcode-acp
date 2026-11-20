package acpfront_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"
)

// CORE-01 concurrency backstop (plan 04-03's `verification: backstop`
// resolution, discharged here for the Phase-4 verifier's `insufficient_spec`
// abstention): concurrent ACP request dispatch over ONE connection — the SDK
// dispatcher serializes them so each response matches its request id with no
// crossed frames. Concurrent PROMPTS are excluded by design (the
// single-active-turn invariant, D-05/CORE-05, proven by
// TestMinimalChatLoop's rejection stage); the genuinely concurrent pair this
// backstop drives is session/new + initialize, both request/response. The
// registry half was already proven by TestRegistryConcurrentAccess (-race);
// the front-tier half — the CONCURRENT session's machine resolving and
// running — is asserted here through a sequential prompt.

const (
	// dispatchPollStep bounds the overlap sequencer's tee polling; the
	// composed loop completes in milliseconds, so this fails fast.
	dispatchPollStep = 2 * time.Millisecond
)

// newSessionOutcome is one in-flight session/new's resolution (the goroutine
// boundary channel payload, mirroring promptOutcome in loop_test.go).
type newSessionOutcome struct {
	resp acp.NewSessionResponse
	err  error
}

// wireFrame is one raw JSON-RPC 2.0 frame decoded generically off the ACP
// tee: id, method, result, error raw — enough to pair every response with
// its request and nothing more (the wire truth is the contract). The wire
// keys stay camelCase (the dialect's spelling; tagliatelle is relaxed for
// front-tier wire-shape tests).
type wireFrame struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  *wireErrorShape `json:"error"`
}

// wireErrorShape is the error object of a JSON-RPC error response frame.
type wireErrorShape struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// TestConcurrentRequestDispatch — the CORE-01 concurrency-dispatch backstop.
// session/new is launched in flight; once its request frame is provably on
// the ACP wire, initialize rides the SAME connection while session/new is
// still unanswered (it cannot answer before three back-leg round trips).
// Assertions, three layers:
//
//  1. both concurrent calls succeed and each CALLER received its own
//     method's typed result (protocolVersion 1 + real agentInfo for
//     initialize; a non-empty session id for session/new) — a crossed frame
//     would decode as zero values in the wrong caller;
//  2. the front's session registry resolves the CONCURRENT session's
//     machine: a sequential prompt on that session runs to end_turn (a
//     registry miss answers invalid-request, never end_turn);
//  3. the raw tee pairs by wire id: exactly one response per request id, no
//     error frames, and each response carries ITS OWN method's result shape
//     — the decisive no-crossed-frames proof at the wire level.
func TestConcurrentRequestDispatch(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	initResp, newOut := driveConcurrentRequests(ctx, t, h)
	assertTypedIdentities(t, initResp, newOut)

	sid := string(newOut.resp.SessionId)

	assertConcurrentSessionRegistry(ctx, t, h, sid)
	assertNoCrossedFrames(t, h, sid)
}

// driveConcurrentRequests overlaps the two ACP requests on the harness's one
// connection: session/new starts in flight, initialize is issued only after
// session/new's request frame is recorded on the tee — so both are
// outstanding simultaneously (the dispatcher serialization property under
// test). Both must resolve inside the bounded exchange window.
func driveConcurrentRequests(
	ctx context.Context, t *testing.T, h *FrontHarness,
) (acp.InitializeResponse, newSessionOutcome) {
	t.Helper()

	newDone := make(chan newSessionOutcome, 1)

	go func() {
		resp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
		newDone <- newSessionOutcome{resp: resp, err: err}
	}()

	waitForTeeLine(t, h, "out", `"method":"session/new"`)

	initResp, initErr := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1})
	if initErr != nil {
		t.Fatalf("initialize while session/new in flight: %v", initErr)
	}

	var newOut newSessionOutcome

	select {
	case newOut = <-newDone:
	case <-time.After(tracerWait):
		t.Fatal("session/new never resolved while initialize was in flight — the dispatcher hung a concurrent request")

		return acp.InitializeResponse{}, newSessionOutcome{} // unreachable
	}

	return initResp, newOut
}

// assertTypedIdentities checks layer 1: both concurrent calls succeeded and
// each caller decoded ITS OWN method's result — the initialize caller sees
// protocolVersion 1 with the real agentInfo, the session/new caller a
// non-empty session id. A crossed or duplicated frame surfaces here as a
// zero-value decode in the wrong caller.
func assertTypedIdentities(t *testing.T, initResp acp.InitializeResponse, newOut newSessionOutcome) {
	t.Helper()

	if newOut.err != nil {
		t.Fatalf("concurrent session/new: %v", newOut.err)
	}

	if newOut.resp.SessionId == "" {
		t.Fatal("concurrent session/new returned an empty session id")
	}

	if initResp.ProtocolVersion != 1 {
		t.Fatalf("initialize response protocolVersion = %d, want 1 (crossed-frame identity)",
			initResp.ProtocolVersion)
	}

	if initResp.AgentInfo == nil || initResp.AgentInfo.Name != "zcode-acp" {
		t.Fatalf("initialize agentInfo = %+v, want name zcode-acp (crossed-frame identity)", initResp.AgentInfo)
	}
}

// assertConcurrentSessionRegistry checks layer 2 — the registry half on the
// composed loop: a sequential prompt on the CONCURRENT session resolves the
// front's registry (registry.Get runs before the turn slot) and drives the
// exact machine session/new registered to a clean end_turn. A registry miss
// or a wrongly-resolved machine answers invalid-request or hangs — never a
// successful streamed turn.
func assertConcurrentSessionRegistry(ctx context.Context, t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	promptResp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("registry probe")},
	})
	if err != nil {
		t.Fatalf("prompt on the concurrent session (registry resolution): %v", err)
	}

	if promptResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("registry-probe prompt stopReason = %q, want %q", promptResp.StopReason, acp.StopReasonEndTurn)
	}
}

// assertNoCrossedFrames checks layer 3 — the wire-level pairing on the raw
// recorded tee: exactly the three client requests (initialize, session/new,
// session/prompt), exactly one response per request id, no error frames
// anywhere, and every response carrying its own method's result shape.
func assertNoCrossedFrames(t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	requests, responses, errFrames := collectWireFrames(t, h)

	if len(errFrames) != 0 {
		t.Fatalf("error frames on the ACP wire, want none: %v", errFrames)
	}

	if len(requests) != 3 {
		t.Fatalf("tee records %d distinct request ids, want exactly 3 (initialize, session/new, session/prompt): %v",
			len(requests), requests)
	}

	for _, method := range []string{"initialize", "session/new", "session/prompt"} {
		if !wireMethodRequestedOnce(requests, method) {
			t.Fatalf("method %q is not on the tee exactly once as a request: %v", method, requests)
		}
	}

	if len(responses) != len(requests) {
		t.Fatalf("tee records %d responses for %d requests — exactly-once pairing violated: %v",
			len(responses), len(requests), responses)
	}

	for id, method := range requests {
		result, ok := responses[id]
		if !ok {
			t.Fatalf("request id %s (%s) has NO response on the tee — a hung concurrent request", id, method)
		}

		assertResultShape(t, id, method, result, sid)
	}
}

// collectWireFrames classifies the recorded ACP tee frames into the request
// table (wire id → method), the response table (wire id → raw result), and
// the raw error frames. Notifications carry no id and take no part in
// request/response pairing.
func collectWireFrames(t *testing.T, h *FrontHarness) (
	requests map[string]string, responses map[string]json.RawMessage, errFrames []string,
) {
	t.Helper()

	requests = make(map[string]string)
	responses = make(map[string]json.RawMessage)

	for _, line := range h.ACPLines() {
		var frame wireFrame
		if err := json.Unmarshal([]byte(line.JSON), &frame); err != nil {
			t.Fatalf("decode tee frame: %v: %s", err, line.JSON)
		}

		switch {
		case len(frame.ID) == 0:
			// Notification — no id, nothing to pair.
		case frame.Method != "":
			requests[string(frame.ID)] = frame.Method
		default:
			if frame.Error != nil {
				errFrames = append(errFrames, line.JSON)
			}

			responses[string(frame.ID)] = frame.Result
		}
	}

	return requests, responses, errFrames
}

// wireMethodRequestedOnce reports whether method appears exactly once in the
// request table (distinct wire ids are guaranteed by the map).
func wireMethodRequestedOnce(requests map[string]string, method string) bool {
	count := 0

	for _, got := range requests {
		if got == method {
			count++
		}
	}

	return count == 1
}

// assertResultShape checks that the response under wire id carries ITS OWN
// method's result shape: protocolVersion for the initialize id, the
// top-level sessionId the caller received for the session/new id, end_turn
// for the session/prompt id. A crossed frame shows here as a foreign shape
// under another request's id.
func assertResultShape(t *testing.T, id, method string, result json.RawMessage, sid string) {
	t.Helper()

	switch method {
	case "initialize":
		var initResult struct {
			ProtocolVersion int `json:"protocolVersion"`
		}

		if err := json.Unmarshal(result, &initResult); err != nil {
			t.Fatalf("initialize result (id %s) undecodable: %v: %s", id, err, result)
		}

		if initResult.ProtocolVersion != 1 {
			t.Fatalf("initialize result (id %s) protocolVersion = %d, want 1 — crossed frame",
				id, initResult.ProtocolVersion)
		}
	case "session/new":
		var newResult struct {
			SessionID string `json:"sessionId"`
		}

		if err := json.Unmarshal(result, &newResult); err != nil {
			t.Fatalf("session/new result (id %s) undecodable: %v: %s", id, err, result)
		}

		if newResult.SessionID != sid {
			t.Fatalf("session/new result (id %s) sessionId = %q, want the caller's %q — crossed frame",
				id, newResult.SessionID, sid)
		}
	case "session/prompt":
		var promptResult struct {
			StopReason string `json:"stopReason"`
		}

		if err := json.Unmarshal(result, &promptResult); err != nil {
			t.Fatalf("session/prompt result (id %s) undecodable: %v: %s", id, err, result)
		}

		if promptResult.StopReason != string(acp.StopReasonEndTurn) {
			t.Fatalf("session/prompt result (id %s) stopReason = %q, want %q — crossed frame",
				id, promptResult.StopReason, acp.StopReasonEndTurn)
		}
	default:
		t.Fatalf("unexpected method %q under wire id %s", method, id)
	}
}

// waitForTeeLine polls the recorded ACP wire until a frame in dir whose raw
// JSON contains substr is recorded, failing at the bound — the overlap
// sequencer proving the first request is on the wire before the second is
// issued.
func waitForTeeLine(t *testing.T, h *FrontHarness, dir, substr string) {
	t.Helper()

	deadline := time.Now().Add(tracerWait)

	for time.Now().Before(deadline) {
		for _, line := range h.ACPLines() {
			if line.Dir == dir && strings.Contains(line.JSON, substr) {
				return
			}
		}

		time.Sleep(dispatchPollStep)
	}

	t.Fatalf("no %s frame matching %q on the ACP tee within %s", dir, substr, tracerWait)
}
