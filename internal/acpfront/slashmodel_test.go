package acpfront_test

import (
	"encoding/json"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The /model slash-prompt layer (SESS-03, plan 07-05, D-03): the omp
// dialect sends model switches as PROMPT turns — session/prompt text
// `/model <name>` is a command, not a literal prompt. The fixtures and
// helpers mirror setconfigoption_test.go (the picker switch the intercept
// mirrors), and the assertions ride the same surfaces: the recorded
// back-leg transcript is the wire truth, the tee the ACP wire truth.
//
//   - TestModelSlashSwitchAppliesViaSetModel (the plan Task-1 tracer RED
//     target): a listed selector applies through the SAME setModel
//     machinery the picker uses (confirmed level, persist-false), the
//     turn resolves end_turn, and the literal slash text never reaches
//     the backend.
//   - TestModelSlashUnknownRefusalByteLocked / TestModelSlashNoModelRefusal
//     (plan Task 2): refusals ride -32603 + data.details byte-exactly.
//   - TestModelSlashRefusalHoldsNoTurnSlot / TestModelSlashSelectorByteEquality /
//     TestModelSlashCarriesConfirmedLevel (plan Task 3): the edge arms.

// slashPrompt is the client's /model prompt ask (the omp dialect's
// `text:e=>`/model ${e}`` wire shape — one text block).
func slashPrompt(sessionID, text string) acp.PromptRequest {
	return acp.PromptRequest{
		SessionId: acp.SessionId(sessionID),
		Prompt:    []acp.ContentBlock{acp.TextBlock(text)},
	}
}

// TestModelSlashSwitchAppliesViaSetModel — the Task-1 tracer RED target
// (D-03): a session/prompt whose text is `/model <listed-value>` applies
// through the SAME setModel machinery the picker uses and resolves as a
// normal completed turn (stopReason end_turn); the literal slash text never
// reaches the backend (the intercept consumed it), and the setModel reply
// becomes the confirmed store — the same settings update the picker
// performs (D-10).
//
//nolint:funlen // each subtest is a full harness lock; splitting would scatter its sequencing
func TestModelSlashSwitchAppliesViaSetModel(t *testing.T) {
	t.Run("a listed selector applies via the picker arm and resolves end_turn", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult, fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
		}))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		resp, err := h.client.Prompt(ctx, slashPrompt(sid, "/model "+corpusModelFlashID))
		if err != nil {
			t.Fatalf("the /model switch must resolve as a normal completed turn: %v", err)
		}

		if resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("stopReason = %q, want end_turn", resp.StopReason)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// The payload byte-locks the picker arm's shape: the target ids from
		// the listed value, the CURRENT confirmed level, persist-false.
		assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), corpusProviderID, corpusModelID, reasoningLevelLow)
	})

	t.Run("the literal slash text never reaches the backend", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult, fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
		}))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		if _, err := h.client.Prompt(ctx, slashPrompt(sid, "/model "+corpusModelFlashID)); err != nil {
			t.Fatalf("the /model switch must resolve as a normal completed turn: %v", err)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// The scenario scripts no session/send rule at all: the slash turn
		// must leave ZERO backend prompt frames — any send proves the
		// intercept let the literal text through.
		if sends := transcriptIndexes(tr, "in", `"method":"session/send"`); len(sends) != 0 {
			t.Fatalf("the intercept must consume the /model prompt (the literal text never "+
				"reaches the backend) — the back leg records %d session/send frame(s)", len(sends))
		}
	})

	t.Run("the setModel reply becomes the confirmed store (the picker's capture)", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult,
			fake.RequestRule{
				Method: zcode.MethodSessionSetModel, Times: 1,
				Reply: &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
			},
			fake.RequestRule{
				Method: zcode.MethodSessionSetModel,
				Reply:  &fake.Reply{Result: json.RawMessage(cfgThoughtReply)},
			}))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		if _, err := h.client.Prompt(ctx, slashPrompt(sid, "/model "+corpusModelFlashID)); err != nil {
			t.Fatalf("the /model switch must resolve as a normal completed turn: %v", err)
		}

		// "high" is listed ONLY in the reply-confirmed set (the pre-ask
		// fixture's plain binding renders [off, low]; the Flash binding's own
		// reasoning.levels [high, low] arrive only via the reply): the ask
		// succeeding proves the store updated from the reply — the same
		// settings update the picker performs (D-10).
		if _, err := h.client.SetSessionConfigOption(
			ctx, setCfgAsk(sid, optionIDThoughtLevel, corpusReasoningLevel)); err != nil {
			t.Fatalf("the follow-up must validate against the reply-confirmed set: %v", err)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// The follow-up rides the same setModel path: exactly the switch
		// frame and the follow-up frame — nothing else mutated the backend.
		if setModels := transcriptIndexes(tr, "in", `"method":"session/setModel"`); len(setModels) != 2 {
			t.Fatalf("the back leg records %d session/setModel frame(s), want exactly 2 "+
				"(the slash switch + the picker follow-up)", len(setModels))
		}
	})
}
