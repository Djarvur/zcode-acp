package acpfront_test

// Front-tier orphan-recovery tests (quick task 261125-r6o, Task 3).
//
// The front-wide slot proves no LOCAL turn is active while a prompt holds
// it, so a backend session/send refused with the numeric -32010 concurrency
// code can only name an ORPHANED backend turn. Every machine the front
// registers for prompts therefore carries the bounded one-stop/one-retry
// recovery: a healthy backend clears on the stop's evidence and the retry
// completes; a stuck one surfaces the stable non-retryable -32603 orphan
// phrase — never the retryable -32003 session_busy class that produced the
// 2026-11-25 unbounded-retry incident.
//
// SUPERSESSION: the old "backend -32010 refusal mapped" arm of
// TestSecondPromptRejected asserted the exact slot-free shape as a -32003
// session_busy surface. That behavior WAS the bug, so the arm is removed
// (cancel_test.go keeps the genuine local-busy slot lock) and this contract
// replaces it. The send-refused-32010 corpus row, its golden, and its
// fake-tier driver are untouched wire fixtures.

import (
	"context"
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
	orphanFrontSessionID = "sess_orphan"
	orphanFrontRetryTurn = "turn_orphan_retry"
)

// orphanFrontPreamble is the create/subscribe/messages projection preamble
// both arms share (the stored reply keeps the completion reconcile prompt).
func orphanFrontPreamble() []fake.RequestRule {
	return []fake.RequestRule{
		{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: json.RawMessage(
			`{"session":{"sessionId":"` + orphanFrontSessionID + `"},` +
				`"protocol":{"name":"ZCode Protocol","version":1}}`)}},
		{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{Result: json.RawMessage(
			`{"sessionId":"` + orphanFrontSessionID + `","eventSeq":0,"events":[]}`)}},
		{Method: zcode.MethodSessionMessages, Reply: &fake.Reply{Result: json.RawMessage(
			`{"messages":[{"info":{"role":"assistant"},"parts":[{"type":"text","text":"ok"}]}],` +
				`"sessionId":"` + orphanFrontSessionID + `"}`)}},
	}
}

// orphanRefusedSend is one -32010 send-refusal rule.
func orphanRefusedSend(t *testing.T, times int) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionSend,
		Times:  times,
		Reply: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeConcurrency,
			Message: "active prompt exists",
		}},
	}
}

// orphanRetrySend is the retried send's ack plus its running patch and
// terminal.
func orphanRetrySend(t *testing.T) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionSend,
		Reply: &fake.Reply{Result: json.RawMessage(
			`{"accepted":true,"sessionId":"` + orphanFrontSessionID + `","stateRevision":2}`)},
		SideEffects: []fake.SideEffect{
			{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: json.RawMessage(
				`{"patch":{"status":"running"},"reason":"prompt_started","revision":1,` +
					`"scope":"session","sessionId":"` + orphanFrontSessionID + `","type":"state.updated"}`)}},
			{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(
				`{"type":"turn.completed","seq":9,"turnId":"` + orphanFrontRetryTurn + `",` +
					`"deliveryKind":"desktop-continuous"}`)}},
		},
	}
}

// orphanStopWithIdle is the stop's ack with the session-scoped idle patch —
// the clear evidence that ends the bounded wait early.
func orphanStopWithIdle(t *testing.T) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionStop,
		Reply:  &fake.Reply{Result: json.RawMessage(`{}`)},
		SideEffects: []fake.SideEffect{
			{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: json.RawMessage(
				`{"patch":{"status":"idle"},"reason":"prompt_completed","revision":2,` +
					`"scope":"session","sessionId":"` + orphanFrontSessionID + `","type":"state.updated"}`)}},
		},
	}
}

// orphanStopSilent is the stop's ack with no clear evidence.
func orphanStopSilent(t *testing.T) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionStop,
		Reply:  &fake.Reply{Result: json.RawMessage(`{}`)},
	}
}

// newOrphanFront composes one programmatic orphan scenario.
func newOrphanFront(t *testing.T, rules []fake.RequestRule, opts ...acpfront.Option) *FrontHarness {
	t.Helper()

	sc := &fake.Scenario{
		Name:      "front-turn-orphan",
		OnRequest: append(orphanFrontPreamble(), rules...),
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: "MethodNotFound",
		}},
		EOFDrainMs: 50,
	}

	return ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		return acpport.NewAgent(acpfront.New(conn, opts...))
	})
}

// openOrphanFrontSession initializes and opens the scripted session.
func openOrphanFrontSession(ctx context.Context, t *testing.T, h *FrontHarness) string {
	t.Helper()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	resp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	return string(resp.SessionId)
}

// assertOrphanWireCounts asserts the two-sends/one-stop contract.
func assertOrphanWireCounts(t *testing.T, tr fake.Transcript) {
	t.Helper()

	if sends := len(transcriptIndexes(tr, "in", `"method":"session/send"`)); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2 (the refused send + the retry)", sends)
	}

	if stops := len(transcriptIndexes(tr, "in", `"method":"session/stop"`)); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1", stops)
	}
}

// TestOrphanRecoveryFrontMapping locks the two front-tier outcomes.
//
//nolint:funlen // the two composed arms share one preamble and wire-count contract
func TestOrphanRecoveryFrontMapping(t *testing.T) {
	t.Run("self-heals: one stop, one retry, end_turn", func(t *testing.T) {
		rules := []fake.RequestRule{
			orphanRefusedSend(t, 1),
			orphanStopWithIdle(t),
			orphanRetrySend(t),
		}

		h := newOrphanFront(t, rules)

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openOrphanFrontSession(ctx, t, h)

		resp, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("heal")},
		})
		if err != nil {
			t.Fatalf("self-healing prompt: %v", err)
		}

		if resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("self-healing prompt stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertOrphanWireCounts(t, tr)
	})

	t.Run("stuck: the stable non-retryable -32603 orphan phrase", func(t *testing.T) {
		rules := []fake.RequestRule{
			orphanRefusedSend(t, 0), // unlimited: both sends are refused
			orphanStopSilent(t),
		}

		h := newOrphanFront(t, rules, acpfront.WithOrphanClearTimeout(40*time.Millisecond))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openOrphanFrontSession(ctx, t, h)

		_, err := h.client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("stuck")},
		})
		if err == nil {
			t.Fatal("a stuck orphan must surface a coded error, got a success response")
		}

		var reqErr *acp.RequestError
		if !errors.As(err, &reqErr) {
			t.Fatalf("stuck orphan error = %T (%v), want a coded JSON-RPC RequestError", err, err)
		}

		if reqErr.Code != acpport.CodeInternal {
			t.Fatalf("stuck orphan code = %d, want %d (the non-retryable internal class)",
				reqErr.Code, acpport.CodeInternal)
		}

		if !strings.Contains(reqErr.Message, "still reports an active prompt after stop and retry") {
			t.Fatalf("stuck orphan message = %q, want the stable orphan phrase", reqErr.Message)
		}

		if reqErr.Code == acpport.CodeSessionBusy {
			t.Fatalf("stuck orphan surfaced the retryable session_busy class (%d)", reqErr.Code)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertOrphanWireCounts(t, tr)
	})
}
