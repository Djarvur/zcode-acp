package acpfront_test

// Steer contract regression (quick task 261125-s7k): Orca dispatches a
// steered message only when the ACTIVE session/prompt request settles
// (1.4.274 src/main/acp/turns/acp-structured-turns.ts: dispatch queues the
// send, cancelForSteer sends session/cancel, and finish() — driven by the
// prompt's promise — starts the next prompt; "Steers wait for the active
// prompt to answer cancellation"). The wrapper must therefore answer the
// cancelled prompt with stopReason cancelled, whatever signal the backend
// reports for the stop, and its next prompt must reach the backend.
//
// The two arms ride the composed fake back leg:
//
//	TestSteerAfterCancelStartsNewTurn — session/new path; the stop's outcome
//	  is the real desktop's turn.failed (not a cancelled state patch), so the
//	  test exercises the terminal-mapping fix end-to-end, and prompt B then
//	  completes normally.
//	TestSteerAfterCancelOnResumedSession — a session/resume-registered machine
//	  whose backend acknowledges the stop with NOTHING; only the cancel grace
//	  can answer (the quiet-window fallback sits at 7.5 s+), proving the
//	  shared construction helper wires the seam into resumed sessions too.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// steerWait bounds the resumed arm's cancelled answer: the injected grace is
// 60 ms and the quiet fallback is 7.5 s+, so only the grace can answer inside
// this bound.
const steerWait = 2 * time.Second

// steerScenarioDir holds the steer-arm wire fixtures. They are front-tier
// fixtures, not back-leg corpus rows: the fake corpus demands a replay
// driver and golden for every scenario it owns, while these exist to drive
// one composed ACP sequence.
const steerScenarioDir = "testdata/steer/"

// mustLoadSteerScenario loads one steer-arm fixture with the strict
// fail-loud loader.
func mustLoadSteerScenario(t *testing.T, name string) *fake.Scenario {
	t.Helper()

	sc, err := fake.LoadScenario(steerScenarioDir + name + ".json")
	if err != nil {
		t.Fatalf("load steer scenario %s: %v", name, err)
	}

	return sc
}

// newSteerFront composes the loop over a scenario with caller-supplied front
// options (the corpus-locked model selection stays bound for create paths).
func newSteerFront(t *testing.T, scenario string, opts ...acpfront.Option) *FrontHarness {
	t.Helper()

	sc := mustLoadSteerScenario(t, scenario)

	return ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		frontOpts := make([]acpfront.Option, 0, len(opts)+2)
		frontOpts = append(frontOpts,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
		)
		frontOpts = append(frontOpts, opts...)

		return acpport.NewAgent(acpfront.New(conn, frontOpts...))
	})
}

// assertCancelledSteerAnswer asserts prompt A answered as the ACP cancelled
// stopReason: a success response, exactly one prompt response frame on the
// tee, and no error frame of the internal (-32603) or cancelled-request
// (-32800) class from the agent.
func assertCancelledSteerAnswer(t *testing.T, h *FrontHarness, outcome promptOutcome) {
	t.Helper()

	if outcome.err != nil {
		t.Fatalf("the cancelled prompt answered with an error (%v), want the cancelled success stopReason", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stopReason = %q, want %q", outcome.resp.StopReason, acp.StopReasonCancelled)
	}

	for _, line := range h.ACPLines() {
		if line.Dir != "in" {
			continue
		}

		if strings.Contains(line.JSON, `"code":-32603`) {
			t.Fatalf("the cancelled prompt carried a -32603 frame: %s", line.JSON)
		}

		if strings.Contains(line.JSON, `"code":-32800`) {
			t.Fatalf("the cancelled prompt carried a -32800 frame: %s", line.JSON)
		}
	}
}

// assertSteerSendOrder asserts the back leg recorded exactly two
// session/send frames and that prompt A's response preceded the second send
// on the ACP wire (the steer was dispatched only after the cancel answered).
func assertSteerSendOrder(t *testing.T, h *FrontHarness, tr fake.Transcript) {
	t.Helper()

	if sends := len(transcriptIndexes(tr, string(fake.DirIn), `"method":"session/send"`)); sends != 2 {
		t.Fatalf("back leg recorded %d session/send frames, want exactly 2 (prompt A + steered prompt B)", sends)
	}

	lines := h.ACPLines()
	answerAt, steerAt := -1, -1

	for i, line := range lines {
		if line.Dir == "in" && strings.Contains(line.JSON, `"stopReason":"cancelled"`) && answerAt < 0 {
			answerAt = i
		}

		if line.Dir == "out" && strings.Contains(line.JSON, `"method":"session/prompt"`) && answerAt >= 0 {
			steerAt = i
		}
	}

	if answerAt < 0 {
		t.Fatal("the cancelled prompt answer is missing from the ACP tee")
	}

	if steerAt < 0 {
		t.Fatal("the steered prompt request is missing from the ACP tee after the cancelled answer")
	}

	if steerAt < answerAt {
		t.Fatal("the steered prompt was dispatched before the cancelled prompt answered")
	}
}

// TestSteerAfterCancelStartsNewTurn drives Orca's exact steer sequence with
// the real desktop's cancel outcome: prompt A streams, the client cancels,
// the backend reports turn.failed, A answers cancelled, and prompt B — the
// steer — starts and completes.
func TestSteerAfterCancelStartsNewTurn(t *testing.T) {
	h := newSteerFront(t, "cancel-turn-failed")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	sid := openCancelStageSession(ctx, t, h)
	promptDone := startActiveTurn(ctx, t, h, sid)

	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	assertCancelledSteerAnswer(t, h, waitPromptOutcome(t, promptDone))

	steerCtx, steerCancel := context.WithTimeout(ctx, tracerWait)
	defer steerCancel()

	steerResp, err := h.client.Prompt(steerCtx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("steered message")},
	})
	if err != nil {
		t.Fatalf("the steered prompt must start after the cancelled answer, got: %v", err)
	}

	if steerResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("steered prompt stopReason = %q, want %q", steerResp.StopReason, acp.StopReasonEndTurn)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertSteerSendOrder(t, h, tr)
}

// TestSteerAfterCancelOnResumedSession proves the cancel grace reaches a
// session/resume-registered machine: the backend acknowledges the stop with
// no outcome at all, and the cancelled prompt still answers inside the
// configured grace, after which the steer starts.
func TestSteerAfterCancelOnResumedSession(t *testing.T) {
	h := newSteerFront(t, "resume-silent-stop", acpfront.WithCancelGrace(60*time.Millisecond))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	const resumedID = "sess_resumed1"

	if _, err := h.client.ResumeSession(ctx, acp.ResumeSessionRequest{
		SessionId: acp.SessionId(resumedID),
		Cwd:       t.TempDir(),
	}); err != nil {
		t.Fatalf("session/resume: %v", err)
	}

	sid := resumedID
	promptDone := startActiveTurn(ctx, t, h, sid)

	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	select {
	case outcome := <-promptDone:
		assertCancelledSteerAnswer(t, h, outcome)
	case <-time.After(steerWait):
		t.Fatal("the resumed session's cancelled prompt never answered inside the grace bound")
	}

	steerCtx, steerCancel := context.WithTimeout(ctx, tracerWait)
	defer steerCancel()

	steerResp, err := h.client.Prompt(steerCtx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("steered message")},
	})
	if err != nil {
		t.Fatalf("the steered prompt must start on the resumed session, got: %v", err)
	}

	if steerResp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("steered prompt stopReason = %q, want %q", steerResp.StopReason, acp.StopReasonEndTurn)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if sends := len(transcriptIndexes(tr, string(fake.DirIn), `"method":"session/send"`)); sends != 2 {
		t.Fatalf("back leg recorded %d session/send frames, want 2 (the steer reached the backend)", sends)
	}
}
