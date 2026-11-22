package acpfront_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpport"
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

// slashPrompt is the client's /model prompt ask (the omp dialect's wire
// shape — one text block carrying "/model <name>").
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

// The pinned omp-dialect refusal strings (SESS-03, D-03): startsWith-
// matched by the client's matchers on code -32603 + data.details — byte-
// locked here as whole strings (the acceptance is string-exact).
const (
	wantUnknownACPModel = "Unknown ACP model: "
	wantNoModelSelected = "No model selected."
)

// teeErrorFrame is one decoded agent->client JSON-RPC error frame off the
// tee — the carriage the Hb matchers classify (Pitfall 1: any other code or
// field is invisible to them).
type teeErrorFrame struct {
	Error struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	} `json:"error"`
}

// teePromptError finds the one prompt error frame on the tee (client view:
// "in" = agent->client) and decodes it.
func teePromptError(t *testing.T, lines []TeeLine) teeErrorFrame {
	t.Helper()

	idx := teeLineIndex(t, lines, "in", `"error"`, "prompt error frame")

	var frame teeErrorFrame
	if err := json.Unmarshal([]byte(lines[idx].JSON), &frame); err != nil {
		t.Fatalf("decode prompt error frame: %v (%s)", err, lines[idx].JSON)
	}

	return frame
}

// assertRefusalCarriage byte-locks the refusal frame: code -32603 and
// data.details EXACTLY wantDetails (the pinned prefix with the selector, or
// the pinned sentence — no paraphrase anywhere).
func assertRefusalCarriage(t *testing.T, frame teeErrorFrame, wantDetails string) {
	t.Helper()

	if frame.Error.Code != acpport.CodeInternal {
		t.Fatalf("refusal code = %d, want %d (the Hb matchers read ONLY -32603)",
			frame.Error.Code, acpport.CodeInternal)
	}

	if len(frame.Error.Data) == 0 {
		t.Fatalf("refusal frame carries NO data — the matchers read data.details only (Pitfall 1); "+
			"message was %q", frame.Error.Message)
	}

	var data struct {
		Details string `json:"details"`
	}

	if err := json.Unmarshal(frame.Error.Data, &data); err != nil {
		t.Fatalf("decode error data: %v (%s)", err, frame.Error.Data)
	}

	if data.Details != wantDetails {
		t.Fatalf("data.details = %q, want EXACTLY %q (byte-exact; the client matches startsWith)",
			data.Details, wantDetails)
	}
}

// assertPromptErrorCode asserts one coded frame with the given code and a
// NULL data payload — the generic seam boundary (the detail seam widens
// ONLY detail-carrying errors, never a blanket data injection).
func assertPromptErrorCode(t *testing.T, frame teeErrorFrame, wantCode int) {
	t.Helper()

	if frame.Error.Code != wantCode {
		t.Fatalf("error code = %d, want %d", frame.Error.Code, wantCode)
	}

	if len(frame.Error.Data) != 0 {
		t.Fatalf("generic error frame must keep data null (the seam widens only "+
			"detail-carrying errors), got %s", frame.Error.Data)
	}
}

// TestModelSlashUnknownRefusalByteLocked — the Task-2 RED target (T-07-15,
// Pitfall 1): an unknown selector refuses with code -32603 whose
// data.details is EXACTLY "Unknown ACP model: <name>" — the pinned prefix
// with its trailing space — and the refusal never touches the backend
// (validate-first, T-06-17).
func TestModelSlashUnknownRefusalByteLocked(t *testing.T) {
	h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid := openSetCfgSession(ctx, t, h)

	_, err := h.client.Prompt(ctx, slashPrompt(sid, "/model nope"))
	if err == nil {
		t.Fatal("an unknown selector must refuse, got success")
	}

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) || rerr.Code != acpport.CodeInternal {
		t.Fatalf("refusal must surface as coded -32603, got %v", err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertRefusalCarriage(t, teePromptError(t, h.ACPLines()), wantUnknownACPModel+"nope")

	if setModels := transcriptIndexes(tr, "in", `"method":"session/setModel"`); len(setModels) != 0 {
		t.Fatalf("the refusal must be emitted before any backend call (validate-first) — "+
			"the back leg records %d session/setModel frame(s)", len(setModels))
	}
}

// slashNoModelCreateResult is the no-model-bound arm's snapshot: an
// available list but NO current/lastUsed binding — the model option renders
// nothing (D-10 + locked OQ2), so the no-model refusal class fires.
const slashNoModelCreateResult = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "low", "defaultLevel": "low",
      "available": [{"value": "off", "label": "Off"}, {"value": "low", "label": "Low"}]
    }
  }
}`

// TestModelSlashNoModelRefusal — the Task-2 RED target (D-03): both no-
// model arms — the command with no argument (bare and empty-name forms) and
// a session with no model binding — refuse with data.details starting
// "No model selected." on code -32603.
func TestModelSlashNoModelRefusal(t *testing.T) {
	askRefusal := func(t *testing.T, sc *fake.Scenario, text string) {
		t.Helper()

		h := newSetCfgFront(t, sc)

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		_, err := h.client.Prompt(ctx, slashPrompt(sid, text))
		if err == nil {
			t.Fatalf("%q must refuse (no model selected), got success", text)
		}

		if _, trErr := h.Transcript(t); trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertRefusalCarriage(t, teePromptError(t, h.ACPLines()), wantNoModelSelected)
	}

	t.Run("the bare command refuses the pinned sentence", func(t *testing.T) {
		askRefusal(t, cfgOptionScenario(cfgCreateResult), "/model")
	})

	t.Run("the command with a trailing space (empty name) refuses the pinned sentence", func(t *testing.T) {
		askRefusal(t, cfgOptionScenario(cfgCreateResult), "/model ")
	})

	t.Run("a session with no model binding refuses the pinned sentence", func(t *testing.T) {
		askRefusal(t, cfgOptionScenario(slashNoModelCreateResult), "/model "+corpusModelFlashID)
	})

	t.Run("generic errors keep data null (the seam widens only detail-carrying errors)", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		openSetCfgSession(ctx, t, h) // a real session; the ask targets an unknown id

		_, err := h.client.Prompt(ctx, slashPrompt("sess_never", "/model "+corpusModelFlashID))
		if err == nil {
			t.Fatal("an unknown session must refuse, got success")
		}

		if _, trErr := h.Transcript(t); trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertPromptErrorCode(t, teePromptError(t, h.ACPLines()), acpport.CodeInvalidRequest)
	})
}

// slashSendRule scripts one unlimited completed literal-prompt turn (the
// edge arms' backend leg): accepted:true + the running patch + turn.started
// + turn.completed on the corpus session id — the same shape the
// setmodel-followup corpus row byte-locks (unlimited budget: repeats
// re-fire for each literal prompt, the 03-09 determinism).
func slashSendRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionSend,
		Reply: &fake.Reply{Result: json.RawMessage(
			`{"accepted":true,"sessionId":"sess_cfg","stateRevision":2}`)},
		SideEffects: []fake.SideEffect{
			{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: json.RawMessage(
				`{"patch":{"status":"running"},"reason":"prompt_started","revision":2,` +
					`"scope":"session","sessionId":"sess_cfg","type":"state.updated"}`)}},
			{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(
				`{"type":"turn.started","seq":4,"turnId":"turn_slash","deliveryKind":"desktop-continuous"}`)}},
			{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(
				`{"type":"turn.completed","seq":5,"turnId":"turn_slash","deliveryKind":"desktop-continuous"}`)}},
		},
	}
}

// TestModelSlashRefusalHoldsNoTurnSlot — the Task-3 edge lock (the
// concurrency probe): a refused /model ask NEVER holds the front-wide turn
// slot — the immediate follow-up prompt on the same session is accepted
// first-try (no invalid-request/turn-active refusal), proving the intercept
// consumed the refusal BEFORE slot acquisition.
func TestModelSlashRefusalHoldsNoTurnSlot(t *testing.T) {
	h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult, slashSendRule()))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid := openSetCfgSession(ctx, t, h)

	_, err := h.client.Prompt(ctx, slashPrompt(sid, "/model nope"))
	if err == nil {
		t.Fatal("the unknown selector must refuse, got success")
	}

	// The follow-up rides first-try: accepted, completed — the refusal never
	// serialized behind (or as) a turn slot.
	resp, err := h.client.Prompt(ctx, slashPrompt(sid, "hello"))
	if err != nil {
		t.Fatalf("the normal prompt must be accepted first-try after a refused "+
			"switch (the refusal never holds the slot): %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want end_turn", resp.StopReason)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// Exactly the refused selector's zero backend frames + the follow-up's
	// ONE send — the refusal added nothing to the wire.
	if sends := transcriptIndexes(tr, "in", `"method":"session/send"`); len(sends) != 1 {
		t.Fatalf("the back leg records %d session/send frame(s), want exactly 1 (the follow-up)", len(sends))
	}

	if setModels := transcriptIndexes(tr, "in", `"method":"session/setModel"`); len(setModels) != 0 {
		t.Fatalf("the refused switch must never touch the backend — "+
			"the back leg records %d session/setModel frame(s)", len(setModels))
	}
}

// TestModelSlashSelectorByteEquality — the Task-3 edge lock (the encoding
// probe): the selector matches the listed values byte-exactly — no case
// folding, no trimming of the name itself; only the ONE separating space is
// consumed. Non-command texts ("/modes ...", mid-text "/model", the
// "/modelfoo" prefix form) fall through as literal backend prompts.
//
//nolint:funlen // each subtest is a full harness lock; splitting would scatter its sequencing
func TestModelSlashSelectorByteEquality(t *testing.T) {
	askUnknown := func(t *testing.T, text, wantDetails string) {
		t.Helper()

		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		_, err := h.client.Prompt(ctx, slashPrompt(sid, text))
		if err == nil {
			t.Fatalf("%q must refuse (byte-exact matching), got success", text)
		}

		if _, trErr := h.Transcript(t); trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertRefusalCarriage(t, teePromptError(t, h.ACPLines()), wantDetails)
	}

	t.Run("a case-mismatched selector answers the Unknown refusal", func(t *testing.T) {
		askUnknown(t, "/model "+corpusModelPlainUpper, wantUnknownACPModel+corpusModelPlainUpper)
	})

	t.Run("a trailing space on the NAME answers the Unknown refusal", func(t *testing.T) {
		nameWithSpace := corpusModelPlainID + " "
		askUnknown(t, "/model "+nameWithSpace, wantUnknownACPModel+nameWithSpace)
	})

	t.Run("non-command texts pass through as literal backend prompts", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult, slashSendRule()))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		for _, literal := range []string{"/modes low", "please /model switch", "/modelfoo"} {
			resp, err := h.client.Prompt(ctx, slashPrompt(sid, literal))
			if err != nil {
				t.Fatalf("the non-command %q must pass through as a normal prompt: %v", literal, err)
			}

			if resp.StopReason != acp.StopReasonEndTurn {
				t.Fatalf("pass-through %q stopReason = %q, want end_turn", literal, resp.StopReason)
			}
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// Every literal reached the backend verbatim — the intercept matches
		// the command form only at the start.
		for _, literal := range []string{"/modes low", "please /model switch", "/modelfoo"} {
			if idx := transcriptIndexes(tr, "in", `"content":"`+literal+`"`); len(idx) != 1 {
				t.Fatalf("the literal %q must ride one session/send frame verbatim, got %d",
					literal, len(idx))
			}
		}

		if setModels := transcriptIndexes(tr, "in", `"method":"session/setModel"`); len(setModels) != 0 {
			t.Fatalf("non-command texts must never mutate the backend — "+
				"the back leg records %d session/setModel frame(s)", len(setModels))
		}
	})
}

// corpusModelPlainUpper is the case-mismatched spelling of the plain model
// id (the listed value "zai-api/GLM-5.3" is lowercase — byte-equality
// refuses the uppercase spelling).
const corpusModelPlainUpper = "ZAI-API/GLM-5.3"

// TestModelSlashCarriesConfirmedLevel — the Task-3 edge lock (the level
// carry, picker parity): a successful switch carries the session's CURRENT
// confirmed reasoning level (never empty, never a fabricated one); a
// session with no confirmed level emits the IN-04 omit-shape (the backend's
// own actionable refusal then surfaces — the field-proven honest behavior).
func TestModelSlashCarriesConfirmedLevel(t *testing.T) {
	t.Run("the confirmed level rides the switch payload", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult, fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
		}))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		if _, err := h.client.Prompt(ctx, slashPrompt(sid, "/model "+corpusModelFlashID)); err != nil {
			t.Fatalf("the switch must succeed: %v", err)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), corpusProviderID, corpusModelID, reasoningLevelLow)
	})

	t.Run("no confirmed level emits the omit-shape (IN-04)", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgNoLevelCreateResult, fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
		}))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		if _, err := h.client.Prompt(ctx, slashPrompt(sid, "/model "+corpusModelFlashID)); err != nil {
			t.Fatalf("the switch must succeed: %v", err)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// wantLevel "" = the options key must be ABSENT — never an empty
		// level object.
		assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), corpusProviderID, corpusModelID, "")
	})
}
