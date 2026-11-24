package acpfront_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Session/set_config_option contract locks (MODEL-02, plan 06-06): the
// validate-first gate (T-06-17 — the client's configId/value are untrusted
// input and every backend call is gated), the level-carry mutation mapping
// (Pitfall 2 on the locked OQ1 setModel path), and the reply-built
// full-list re-emit (D-10/T-06-19) — each asserted on the composed loop,
// the RECORDED back-leg transcript being the wire truth.
//
//   - TestSetConfigOptionInvalid (the plan Task-1 RED target): every
//     invalid arm answers coded -32602 naming the configId and the allowed
//     set/values, and NO backend mutation frame leaves the wrapper.
//   - TestSetConfigOptionDelegation rides the adapter tier (agent_test.go).
//   - TestSetConfigOptionModelSwitch / ThoughtSwitch / ReEmit (the plan
//     Task-2 RED targets): the mutation arms lock the setModel payloads
//     and the reply-only rebuild.

// cfgOptionScenario composes the per-arm fake for the set_config_option
// vectors: one create reply carrying the confirmed settings snapshot, one
// subscribe reply, then any extra rules (the picker's setModel budgets, the
// teardown close rule) in call order. Requests the scenario did not script
// fail loud (the -32601 unknown-method reply — a silent {} reply would hide
// contract regressions).
func cfgOptionScenario(createResult string, extra ...fake.RequestRule) *fake.Scenario {
	rules := make([]fake.RequestRule, 0, 2+len(extra))
	rules = append(rules,
		fake.RequestRule{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: json.RawMessage(createResult)}},
		fake.RequestRule{
			Method: zcode.MethodSessionSubscribe,
			Reply:  &fake.Reply{Result: json.RawMessage(`{"sessionId":"sess_cfg","eventSeq":0,"events":[]}`)},
		})
	rules = append(rules, extra...)

	return &fake.Scenario{
		Name:      "setconfigoption",
		OnRequest: rules,
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: fakeMethodNotFound},
		},
		EOFDrainMs: 100,
	}
}

// setCfgAsk builds the client's set_config_option request — the ValueId
// variant (the wire default; this build advertises select options only).
func setCfgAsk(sessionID, configID, value string) acp.SetSessionConfigOptionRequest {
	return acp.SetSessionConfigOptionRequest{
		ValueId: &acp.SetSessionConfigOptionValueId{
			SessionId: acp.SessionId(sessionID),
			ConfigId:  acp.SessionConfigId(configID),
			Value:     acp.SessionConfigValueId(value),
		},
	}
}

// newSetCfgFront composes the set_config_option harness with NO model
// configured — the picker's own session/setModel frames are then the only
// mutation frames the back leg can record.
func newSetCfgFront(t *testing.T, sc *fake.Scenario) *FrontHarness {
	t.Helper()

	return ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		return acpport.NewAgent(acpfront.New(conn))
	})
}

// openSetCfgSession runs the initialize + session/new preamble and returns
// the session id.
func openSetCfgSession(ctx context.Context, t *testing.T, h *FrontHarness) string {
	t.Helper()

	sid, _ := openSetCfgSessionWithOptions(ctx, t, h)

	return sid
}

// openSetCfgSessionWithOptions also returns the initial advertised options so
// omitted-select alias vectors can prove the translator hid thought_level.
func openSetCfgSessionWithOptions(
	ctx context.Context, t *testing.T, h *FrontHarness,
) (string, []acp.SessionConfigOption) {
	t.Helper()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	resp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if resp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	return string(resp.SessionId), resp.ConfigOptions
}

// assertNoBackendMutation asserts the transcript carries NO
// session/setModel frame and no dedicated thought-level method anywhere —
// validation gates every backend call (T-06-17), and the locked OQ1 keeps
// setThoughtLevel off the v1 wire entirely.
func assertNoBackendMutation(t *testing.T, tr fake.Transcript) {
	t.Helper()

	if setModels := transcriptIndexes(tr, "in", `"method":"session/setModel"`); len(setModels) != 0 {
		t.Fatalf("validation must gate every backend call (T-06-17) — "+
			"the back leg records %d session/setModel frame(s)", len(setModels))
	}

	for _, line := range tr.Lines {
		if strings.Contains(line.JSON, "setThoughtLevel") {
			t.Fatalf("no dedicated thought-level method exists (locked OQ1) — the back leg records: %s", line.JSON)
		}
	}
}

// assertInvalidParamsNaming asserts one coded -32602 whose message carries
// every required fragment — the configId, the allowed set/values, or the
// session id; the caller pins which. (The message-free class assert is
// lifecycle_test.go's assertCodedInvalidParams.)
func assertInvalidParamsNaming(t *testing.T, err error, mustContain ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("set_config_option must fail, got success")
	}

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("rejection must surface as a coded RequestError, got %T: %v", err, err)
	}

	if rerr.Code != acpport.CodeInvalidParams {
		t.Fatalf("error code = %d, want %d (invalid params class)", rerr.Code, acpport.CodeInvalidParams)
	}

	for _, frag := range mustContain {
		if !strings.Contains(rerr.Message, frag) {
			t.Fatalf("error message %q must name %q", rerr.Message, frag)
		}
	}
}

// TestSetConfigOptionInvalid — the Task-1 RED target (T-06-17): every
// invalid arm answers coded -32602 naming the configId and the allowed
// set/values (the CONTEXT discretion lock: coded invalid-params, never a
// crash path), and NO backend frame leaves the wrapper — validation gates
// every backend call.
//
//nolint:funlen // each subtest is a full harness lock; splitting would scatter its sequencing
func TestSetConfigOptionInvalid(t *testing.T) {
	t.Run("unknown configId names it and the allowed set", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(settingsCreateResult))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		_, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, "volume", "loud"))

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertNoBackendMutation(t, tr)
		assertInvalidParamsNaming(t, err, "volume", "model", "thought_level")
	})

	t.Run("a value outside the option's own listed set names both", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(settingsCreateResult))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		_, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDModel, "zai-api/GLM-9"))

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertNoBackendMutation(t, tr)
		assertInvalidParamsNaming(t, err, optionIDModel, "zai-api/GLM-9", corpusModelFlashID)
	})

	t.Run("a session with no confirmed settings answers -32602, never a panic", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(settingsCreateResult))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		openSetCfgSession(ctx, t, h) // a real session opens; the ask targets a different id

		_, err := h.client.SetSessionConfigOption(ctx, setCfgAsk("sess_never", optionIDModel, corpusModelPlainID))

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertNoBackendMutation(t, tr)
		assertInvalidParamsNaming(t, err, "sess_never")
	})

	t.Run("a closed session answers the established coded posture", func(t *testing.T) {
		sc := cfgOptionScenario(settingsCreateResult, fake.RequestRule{
			Method: zcode.MethodSessionClose,
			Reply:  &fake.Reply{Result: json.RawMessage(`{"closed":true}`)},
		})
		h := newSetCfgFront(t, sc)

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		closeReq := acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}
		if _, closeErr := h.client.CloseSession(ctx, closeReq); closeErr != nil {
			t.Fatalf("close: %v", closeErr)
		}

		_, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDModel, corpusModelPlainID))

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertNoBackendMutation(t, tr)
		assertInvalidParamsNaming(t, err, sid)
	})
}

// cfgCreateResult is the mutation vectors' create snapshot: the PLAIN
// GLM-5.3 binds (its entry carries no reasoning block, so the thought
// picker falls back to thoughtLevel.available [off, low]) and the
// confirmed level is low — the level-carry vector's CURRENT confirmed
// level.
const cfgCreateResult = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash",
         "description": "The flagship reasoning model",
         "reasoning": {"levels": [
            {"value": "high", "label": "High", "description": "deep reasoning"},
            {"value": "low", "label": "Low"}], "defaultLevel": "high"}},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "low", "defaultLevel": "low",
      "available": [{"value": "off", "label": "Off"}, {"value": "low", "label": "Low"}]
    }
  }
}`

// cfgSetModelReply is the model switch's afterStateMutation reply: the
// CONFIRMED snapshot with the NEW current (Flash) and a CHANGED available
// list (GLM-5.3 first now) — the re-emit vector's reply-only source, kept
// deliberately different from cfgCreateResult on BOTH surfaces so a
// pre-ask rebuild cannot pass.
const cfgSetModelReply = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash",
         "description": "The flagship reasoning model",
         "reasoning": {"levels": [
            {"value": "high", "label": "High", "description": "deep reasoning"},
            {"value": "low", "label": "Low"}], "defaultLevel": "high"}}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "low", "defaultLevel": "low",
      "available": [{"value": "off", "label": "Off"}, {"value": "low", "label": "Low"}]
    }
  }
}`

// cfgNoLevelCreateResult is the omit-shape arm's snapshot: NO thoughtLevel
// block at all — no confirmed level exists, so a model switch must emit the
// documented omit-shape (never an empty-level options object, IN-04).
const cfgNoLevelCreateResult = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash"},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
      ]
    }
  }
}`

// cfgFlashBoundCreateResult is the thought-switch vector's snapshot: the
// Flash binds (the thought picker renders ITS reasoning.levels [high, low],
// so "high" is listed) and the confirmed level is low.
const cfgFlashBoundCreateResult = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash",
         "description": "The flagship reasoning model",
         "reasoning": {"levels": [
            {"value": "high", "label": "High", "description": "deep reasoning"},
            {"value": "low", "label": "Low"}], "defaultLevel": "high"}},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "low", "defaultLevel": "low",
      "available": [{"value": "off", "label": "Off"}, {"value": "low", "label": "Low"}]
    }
  }
}`

// cfgThoughtReply is the thought switch's afterStateMutation reply: the
// CONFIRMED snapshot with thoughtLevel.current moved to high — the
// re-emit's confirmation source.
const cfgThoughtReply = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash",
         "description": "The flagship reasoning model",
         "reasoning": {"levels": [
            {"value": "high", "label": "High", "description": "deep reasoning"},
            {"value": "low", "label": "Low"}], "defaultLevel": "high"}},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "high", "defaultLevel": "high",
      "available": [{"value": "off", "label": "Off"}, {"value": "high", "label": "High"}]
    }
  }
}`

// setCfgModelFrame is the setModel request frame's payload shape (the
// dialect's camelCase wire keys, capture-verbatim).
type setCfgModelFrame struct {
	Params struct {
		Model struct {
			ProviderID string          `json:"providerId"`
			ModelID    string          `json:"modelId"`
			Options    json.RawMessage `json:"options"`
		} `json:"model"`
		PersistAsWorkspaceLastUsed *bool `json:"persistAsWorkspaceLastUsed"`
	} `json:"params"`
}

// assertSetCfgModelFrame byte-locks one setModel request frame: the
// provider/model ids, the options object (wantLevel "" = the omit-shape —
// the options key must be ABSENT, never an empty level), and explicit
// persistAsWorkspaceLastUsed:false.
func assertSetCfgModelFrame(t *testing.T, raw, providerID, modelID, wantLevel string) {
	t.Helper()

	var frame setCfgModelFrame
	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode setModel frame: %v (%s)", err, raw)
	}

	model := frame.Params.Model
	if model.ProviderID != providerID || model.ModelID != modelID {
		t.Fatalf("setModel model = %s/%s, want %s/%s: %s", model.ProviderID, model.ModelID, providerID, modelID, raw)
	}

	if frame.Params.PersistAsWorkspaceLastUsed == nil || *frame.Params.PersistAsWorkspaceLastUsed {
		t.Fatalf("setModel persistAsWorkspaceLastUsed must be explicitly false: %s", raw)
	}

	if wantLevel == "" {
		if len(model.Options) != 0 {
			t.Fatalf("setModel must carry the omit-shape (no options key), got %s: %s", model.Options, raw)
		}

		return
	}

	var opts struct {
		ReasoningLevel string `json:"reasoningLevel"`
	}

	if err := json.Unmarshal(model.Options, &opts); err != nil {
		t.Fatalf("decode setModel options: %v (%s: %s)", err, model.Options, raw)
	}

	if opts.ReasoningLevel != wantLevel {
		t.Fatalf("setModel options.reasoningLevel = %q, want %q (the level must ride options): %s",
			opts.ReasoningLevel, wantLevel, raw)
	}
}

// assertSingleSetModel asserts exactly one session/setModel frame on the
// back leg and returns it.
func assertSingleSetModel(t *testing.T, tr fake.Transcript) string {
	t.Helper()

	setModels := transcriptIndexes(tr, "in", `"method":"session/setModel"`)
	if len(setModels) != 1 {
		t.Fatalf("back leg records %d session/setModel frame(s), want exactly 1", len(setModels))
	}

	return tr.Lines[setModels[0]].JSON
}

// TestSetConfigOptionModelSwitch — the Task-2 RED target (Pitfall 2): a
// valid model value maps to exactly one session/setModel whose model object
// parses with strings.Cut on the FIRST slash only and whose options carry
// the CURRENT confirmed level when one exists (the level-carry vector — a
// switch must never silently drop the level) or omit options entirely when
// none does (the IN-04 omit-shape); persistAsWorkspaceLastUsed stays false,
// and a fatal/dead-target setModel refusal tears the backend session down
// best-effort (the WR-04 discipline — never an orphan).
//
//nolint:funlen // each subtest is a full harness lock; splitting would scatter its sequencing
func TestSetConfigOptionModelSwitch(t *testing.T) {
	t.Run("the setModel carries the confirmed level (level-carry vector)", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult, fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
		}))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		ask := setCfgAsk(sid, optionIDModel, corpusModelFlashID)
		if _, err := h.client.SetSessionConfigOption(ctx, ask); err != nil {
			t.Fatalf("model switch: %v", err)
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

		ask := setCfgAsk(sid, optionIDModel, corpusModelFlashID)
		if _, err := h.client.SetSessionConfigOption(ctx, ask); err != nil {
			t.Fatalf("model switch: %v", err)
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), corpusProviderID, corpusModelID, "")
	})

	t.Run("a fatal setModel refusal closes the backend session (WR-04, never an orphan)", func(t *testing.T) {
		h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult,
			fake.RequestRule{
				Method: zcode.MethodSessionSetModel,
				Reply: &fake.Reply{Error: &zcode.WireError{
					Code: zcode.CodeResourceNotFound, Message: "session not found",
				}},
			},
			fake.RequestRule{
				Method: zcode.MethodSessionClose,
				Reply:  &fake.Reply{Result: json.RawMessage(`{"closed":true}`)},
			}))

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		sid := openSetCfgSession(ctx, t, h)

		_, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDModel, corpusModelFlashID))
		if err == nil {
			t.Fatal("a refused setModel must fail the ask, got success")
		}

		tr, trErr := h.Transcript(t)
		if trErr != nil {
			t.Fatalf("fake transcript: %v", trErr)
		}

		// The refusal surfaced honestly AND the just-mutated backend session
		// was torn down best-effort — the transcript shows BOTH frames.
		if closes := transcriptIndexes(tr, "in", `"method":"session/close"`); len(closes) != 1 {
			t.Fatalf("a refused setModel must close the backend session best-effort (WR-04) — "+
				"the back leg records %d session/close frame(s)", len(closes))
		}
	})
}

// TestSetConfigOptionThoughtSwitch — the Task-2 RED target (locked OQ1): a
// valid thought_level value travels on the SAME proven setModel path — the
// CONFIRMED current model rides unchanged and options.reasoningLevel = the
// requested level; no dedicated setThoughtLevel frame may exist anywhere on
// the wire.
func TestSetConfigOptionThoughtSwitch(t *testing.T) {
	h := newSetCfgFront(t, cfgOptionScenario(cfgFlashBoundCreateResult, fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply:  &fake.Reply{Result: json.RawMessage(cfgThoughtReply)},
	}))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid := openSetCfgSession(ctx, t, h)

	ask := setCfgAsk(sid, optionIDThoughtLevel, corpusReasoningLevel)
	if _, err := h.client.SetSessionConfigOption(ctx, ask); err != nil {
		t.Fatalf("thought switch: %v", err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// The setModel path is the ONLY mutation frame, and the confirmed model
	// rides unchanged with the requested level.
	assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), corpusProviderID, corpusModelID, corpusReasoningLevel)

	for _, line := range tr.Lines {
		if strings.Contains(line.JSON, "setThoughtLevel") {
			t.Fatalf("the dedicated setThoughtLevel method is locked out of the v1 wire (OQ1): %s", line.JSON)
		}
	}
}

// TestSetConfigOptionReEmitReplyWins — the Task-2 RED target (D-10/T-06-19):
// the response re-emits the FULL option list rebuilt FROM THE MUTATION
// REPLY's settings snapshot — never the pre-ask one.
func TestSetConfigOptionReEmitReplyWins(t *testing.T) {
	h := newSetCfgFront(t, cfgOptionScenario(cfgCreateResult, fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply:  &fake.Reply{Result: json.RawMessage(cfgSetModelReply)},
	}))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid := openSetCfgSession(ctx, t, h)

	resp, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDModel, corpusModelFlashID))
	if err != nil {
		t.Fatalf("model switch: %v", err)
	}

	// The merged expectation (the 2026-11-23 catalog-preservation
	// contract): the CONFIRMED current and thought level ride the
	// reply; the AVAILABLE catalog rides the previous (create)
	// snapshot — the reply's provider-scoped view must not shrink the
	// picker's entries. Fixtures differ on both sides on purpose: a
	// pre-ask rebuild fails on the current, a raw-reply rebuild fails
	// on the catalog.
	replySettings, parsed := acpfront.ParseSettings(json.RawMessage(cfgSetModelReply))
	if !parsed {
		t.Fatal("the fixture reply must parse")
	}

	createSettings, createParsed := acpfront.ParseSettings(json.RawMessage(cfgCreateResult))
	if !createParsed {
		t.Fatal("the create fixture must parse")
	}

	createSettings.Model.Current = replySettings.Model.Current
	createSettings.Model.LastUsed = replySettings.Model.LastUsed
	createSettings.ThoughtLevel = replySettings.ThoughtLevel

	want := acpfront.BuildConfigOptions(createSettings)

	assertConfigOptionsMatch(t, resp.ConfigOptions, want)
}

// TestSetConfigOptionReEmitStoreUpdated — the confirmed store updates from
// the reply before the rebuild (a follow-up ask validates against the
// reply-confirmed set).
func TestSetConfigOptionReEmitStoreUpdated(t *testing.T) {
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

	ask := setCfgAsk(sid, optionIDModel, corpusModelFlashID)
	if _, err := h.client.SetSessionConfigOption(ctx, ask); err != nil {
		t.Fatalf("model switch: %v", err)
	}

	// "high" is listed ONLY in the reply-confirmed set (the pre-ask
	// fixture's thought list is [off, low] — the plain bound model has
	// no reasoning block): the ask succeeding proves the store updated
	// from the reply BEFORE this validation ran.
	resp, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDThoughtLevel, corpusReasoningLevel))
	if err != nil {
		t.Fatalf("the follow-up must validate against the REPLY-confirmed set: %v", err)
	}

	replySettings, parsed := acpfront.ParseSettings(json.RawMessage(cfgThoughtReply))
	if !parsed {
		t.Fatal("the fixture reply must parse")
	}

	assertConfigOptionsMatch(t, resp.ConfigOptions, acpfront.BuildConfigOptions(replySettings))
}

// assertConfigOptionsMatch deep-compares the client response's option list
// against the expected neutral defs — id/name/category/currentValue and
// every entry's value/name/description (the adapter's own select-assembly
// bytes are locked at the acpport tier; this is the rebuild-source lock).
//
//nolint:cyclop // one flat field-by-field deep-compare; splitting would hide the lock
func assertConfigOptionsMatch(t *testing.T, got []acp.SessionConfigOption, want []acpport.ConfigOptionDef) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("response list = %d entries, want the reply-built %d", len(got), len(want))
	}

	for i, def := range want {
		sel := got[i].Select
		if sel == nil {
			t.Fatalf("response[%d] carries no select variant: %+v", i, got[i])
		}

		category := ""
		if sel.Category != nil {
			category = string(*sel.Category)
		}

		if string(sel.Id) != def.ID || sel.Name != def.Name ||
			category != def.Category || string(sel.CurrentValue) != def.CurrentValue {
			t.Fatalf("response[%d] = id %q name %q category %q current %q, want %q/%q/%q/%q (reply-built, D-10)",
				i, sel.Id, sel.Name, category, sel.CurrentValue, def.ID, def.Name, def.Category, def.CurrentValue)
		}

		var entries []acp.SessionConfigSelectOption
		if sel.Options.Ungrouped != nil {
			entries = *sel.Options.Ungrouped
		}

		if len(entries) != len(def.Options) {
			t.Fatalf("response[%d] entries = %d, want %d (the reply's list verbatim)",
				i, len(entries), len(def.Options))
		}

		for j, wantOpt := range def.Options {
			gotOpt := entries[j]
			desc := ""
			if gotOpt.Description != nil {
				desc = *gotOpt.Description
			}

			if string(gotOpt.Value) != wantOpt.Value || gotOpt.Name != wantOpt.Name || desc != wantOpt.Description {
				t.Fatalf("response[%d] entry %d = %q/%q/%q, want %q/%q/%q (D-08 verbatim mirror)",
					i, j, gotOpt.Value, gotOpt.Name, desc, wantOpt.Value, wantOpt.Name, wantOpt.Description)
			}
		}
	}
}

const (
	reasoningSnapshotProvider       = "zai-api"
	reasoningSnapshotSource         = "DeepSeek"
	reasoningSnapshotTargetProvider = "OpenCodeGo"
	reasoningSnapshotTargetModel    = "hungerg-free"
	reasoningSnapshotTarget         = reasoningSnapshotTargetProvider + "/" + reasoningSnapshotTargetModel
	reasoningLevelMax               = "max"
	reasoningLevelOff               = "off"
	reasoningLabelMax               = "Max"
	reasoningLabelHigh              = "High"
	reasoningLabelLow               = "Low"
	reasoningLabelOff               = "Off"
	reasoningSnapshotLabelKey       = "label"
	reasoningSnapshotCurrentKey     = "current"
	reasoningSnapshotLastUsedKey    = "lastUsed"
	reasoningBindingCurrent         = "current"
	reasoningBindingLastUsed        = "lastUsed"
	orcaThoughtEnabled              = "enabled"
	orcaThoughtDisabled             = "disabled"
)

// cfgReasoningEntry builds one catalog entry without duplicating wire JSON.
// Labels mirror the values; the resolver and wire assertions key on values.
func cfgReasoningEntry(ref acpfront.ModelRef, levels ...string) acpfront.ModelEntry {
	entry := acpfront.ModelEntry{Ref: ref, Label: ref.ModelID}
	entry.ReasoningLevels = make([]acpfront.ReasoningLevel, 0, len(levels))

	for _, value := range levels {
		entry.ReasoningLevels = append(entry.ReasoningLevels, acpfront.ReasoningLevel{Value: value, Label: value})
	}

	return entry
}

func cfgReasoningRefWire(ref acpfront.ModelRef) map[string]any {
	return map[string]any{"providerId": ref.ProviderID, "modelId": ref.ModelID}
}

func cfgReasoningCatalogWire(available []acpfront.ModelEntry) []any {
	entries := make([]any, 0, len(available))
	for _, entry := range available {
		wireEntry := map[string]any{
			"ref": cfgReasoningRefWire(entry.Ref), reasoningSnapshotLabelKey: entry.Label,
		}
		if len(entry.ReasoningLevels) > 0 || entry.DefaultLevel != "" {
			levels := make([]any, 0, len(entry.ReasoningLevels))
			for _, level := range entry.ReasoningLevels {
				levels = append(levels, map[string]any{
					"value": level.Value, reasoningSnapshotLabelKey: level.Label, "description": level.Description,
				})
			}

			wireEntry["reasoning"] = map[string]any{
				"levels": levels, "defaultLevel": entry.DefaultLevel,
			}
		}

		entries = append(entries, wireEntry)
	}

	return entries
}

func cfgReasoningFallbackWire(globalLevels []acpfront.LevelEntry) []any {
	fallback := make([]any, 0, len(globalLevels))
	for _, level := range globalLevels {
		fallback = append(fallback, map[string]any{
			"value": level.Value, reasoningSnapshotLabelKey: level.Label, "description": level.Description,
		})
	}

	return fallback
}

// cfgReasoningCreateResult builds a backend-confirmed settings snapshot with
// independently controlled bindings, exact catalog entries, and global fallback
// values. It stays test-local so later ACP wire vectors reuse the same shape.
func cfgReasoningCreateResult(
	t *testing.T,
	current, lastUsed *acpfront.ModelRef,
	currentLevel string,
	thoughtEnabled bool,
	available []acpfront.ModelEntry,
	globalLevels []acpfront.LevelEntry,
) string {
	t.Helper()

	model := map[string]any{"available": cfgReasoningCatalogWire(available)}
	if current != nil {
		model[reasoningSnapshotCurrentKey] = cfgReasoningRefWire(*current)
	}

	if lastUsed != nil {
		model[reasoningSnapshotLastUsedKey] = cfgReasoningRefWire(*lastUsed)
	}

	raw, err := json.Marshal(map[string]any{
		"session": map[string]any{"sessionId": "sess_cfg"},
		"settings": map[string]any{
			"model": model,
			"thoughtLevel": map[string]any{
				"enabled": thoughtEnabled, reasoningSnapshotCurrentKey: currentLevel,
				"defaultLevel": currentLevel, "available": cfgReasoningFallbackWire(globalLevels),
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal reasoning snapshot: %v", err)
	}

	return string(raw)
}

// TestSetConfigOptionTargetReasoningResolution locks the exact-target order at
// the ACP wire: preserve an advertised current level, else advertised default,
// else first listed level; no source level still selects a target level, and a
// level-less target omits options.
//
//nolint:funlen // the table is one ACP wire contract; splitting would scatter its cases
func TestSetConfigOptionTargetReasoningResolution(t *testing.T) {
	source := acpfront.ModelRef{ProviderID: reasoningSnapshotProvider, ModelID: reasoningSnapshotSource}
	target := acpfront.ModelRef{ProviderID: reasoningSnapshotTargetProvider, ModelID: reasoningSnapshotTargetModel}

	for _, tc := range []struct {
		name          string
		current       string
		targetLevels  []string
		targetDefault string
		sameModel     bool
		want          string
	}{
		{
			name:          "current max advertised by target is preserved",
			current:       reasoningLevelMax,
			targetLevels:  []string{reasoningLevelLow, corpusReasoningLevel, reasoningLevelMax},
			targetDefault: corpusReasoningLevel, want: reasoningLevelMax,
		},
		{
			name:          "same-model max carry is preserved",
			current:       reasoningLevelMax,
			targetLevels:  []string{reasoningLevelLow, corpusReasoningLevel, reasoningLevelMax},
			targetDefault: corpusReasoningLevel, sameModel: true, want: reasoningLevelMax,
		},
		{
			name:    "stale max falls back to advertised target default",
			current: reasoningLevelMax, targetLevels: []string{reasoningLevelLow, corpusReasoningLevel},
			targetDefault: corpusReasoningLevel, want: corpusReasoningLevel,
		},
		{
			name:    "absent target default falls back to first low",
			current: reasoningLevelMax, targetLevels: []string{reasoningLevelLow, corpusReasoningLevel},
			want: reasoningLevelLow,
		},
		{
			name:    "unadvertised target default falls back to first low",
			current: reasoningLevelMax, targetLevels: []string{reasoningLevelLow, corpusReasoningLevel},
			targetDefault: reasoningLevelMax, want: reasoningLevelLow,
		},
		{
			name:          "no source confirmed level selects advertised target default",
			targetLevels:  []string{reasoningLevelLow, corpusReasoningLevel},
			targetDefault: corpusReasoningLevel, want: corpusReasoningLevel,
		},
		{
			name:    "source max into level-less target omits options",
			current: reasoningLevelMax, want: "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceEntry := cfgReasoningEntry(source, reasoningLevelLow, corpusReasoningLevel, reasoningLevelMax)
			sourceEntry.DefaultLevel = reasoningLevelMax
			targetEntry := cfgReasoningEntry(target, tc.targetLevels...)
			targetEntry.DefaultLevel = tc.targetDefault
			available := []acpfront.ModelEntry{sourceEntry}

			askTarget := target
			if tc.sameModel {
				targetEntry.Ref = source
				available[0] = targetEntry
				askTarget = source
			} else {
				available = append(available, targetEntry)
			}

			create := cfgReasoningCreateResult(t, &source, nil, tc.current, true, available,
				[]acpfront.LevelEntry{{Value: reasoningLevelOff, Label: reasoningLabelOff}})
			reply := cfgReasoningCreateResult(t, &askTarget, nil, tc.want, tc.want != "",
				[]acpfront.ModelEntry{targetEntry}, nil)
			h := newSetCfgFront(t, cfgOptionScenario(create, fake.RequestRule{
				Method: zcode.MethodSessionSetModel,
				Reply:  &fake.Reply{Result: json.RawMessage(reply)},
			}))

			ctx, cancel := h.ExchangeCtx()
			defer cancel()

			sid := openSetCfgSession(ctx, t, h)
			value := askTarget.ProviderID + "/" + askTarget.ModelID
			if _, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDModel, value)); err != nil {
				t.Fatalf("model switch: %v", err)
			}

			tr, trErr := h.Transcript(t)
			if trErr != nil {
				t.Fatalf("fake transcript: %v", trErr)
			}

			assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), askTarget.ProviderID, askTarget.ModelID, tc.want)
			if closes := transcriptIndexes(tr, "in", `"method":"session/close"`); len(closes) != 0 {
				t.Fatalf("a successful target resolution must not close the session: %+v", closes)
			}
		})
	}
}

// TestModelSlashTargetReasoningResolution proves /model shares the picker's
// exact-target resolver rather than carrying the source level independently.
func TestModelSlashTargetReasoningResolution(t *testing.T) {
	source := acpfront.ModelRef{ProviderID: reasoningSnapshotProvider, ModelID: reasoningSnapshotSource}
	target := acpfront.ModelRef{ProviderID: reasoningSnapshotTargetProvider, ModelID: reasoningSnapshotTargetModel}
	sourceEntry := cfgReasoningEntry(source, reasoningLevelLow, corpusReasoningLevel, reasoningLevelMax)
	sourceEntry.DefaultLevel = reasoningLevelMax
	targetEntry := cfgReasoningEntry(target, reasoningLevelLow, corpusReasoningLevel)
	targetEntry.DefaultLevel = corpusReasoningLevel

	create := cfgReasoningCreateResult(t, &source, nil, reasoningLevelMax, true,
		[]acpfront.ModelEntry{sourceEntry, targetEntry},
		[]acpfront.LevelEntry{{Value: reasoningLevelOff, Label: reasoningLabelOff}})
	reply := cfgReasoningCreateResult(t, &target, nil, corpusReasoningLevel, true,
		[]acpfront.ModelEntry{targetEntry}, nil)
	h := newSetCfgFront(t, cfgOptionScenario(create, fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply:  &fake.Reply{Result: json.RawMessage(reply)},
	}))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid := openSetCfgSession(ctx, t, h)
	if _, err := h.client.Prompt(ctx, slashPrompt(sid, "/model "+reasoningSnapshotTarget)); err != nil {
		t.Fatalf("the /model switch must succeed: %v", err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), target.ProviderID, target.ModelID, corpusReasoningLevel)
}

const unknownAliasConfigID = "unknown"

type thoughtAliasSuccessCase struct {
	name           string
	alias          string
	current        string
	levels         []string
	defaultLevel   string
	thoughtEnabled bool
	omitted        string
	binding        string
	want           string
}

func assertThoughtOptionOmitted(t *testing.T, options []acp.SessionConfigOption) {
	t.Helper()

	for _, option := range options {
		if option.Select != nil && option.Select.Id == optionIDThoughtLevel {
			t.Fatalf("thought_level must be omitted before alias normalization: %+v", option)
		}

		if option.Boolean != nil && option.Boolean.Id == optionIDThoughtLevel {
			t.Fatalf("thought_level must not advertise a boolean variant: %+v", option)
		}
	}
}

func runThoughtAliasSuccess(t *testing.T, tc thoughtAliasSuccessCase) {
	t.Helper()

	bound := acpfront.ModelRef{ProviderID: reasoningSnapshotTargetProvider, ModelID: reasoningSnapshotTargetModel}
	entry := cfgReasoningEntry(bound, tc.levels...)
	entry.DefaultLevel = tc.defaultLevel

	var current, lastUsed *acpfront.ModelRef
	if tc.binding == reasoningBindingLastUsed {
		lastUsed = &bound
	} else {
		current = &bound
	}

	create := cfgReasoningCreateResult(t, current, lastUsed, tc.current, tc.thoughtEnabled,
		[]acpfront.ModelEntry{entry},
		[]acpfront.LevelEntry{
			{Value: reasoningLevelOff, Label: reasoningLabelOff},
			{Value: reasoningLevelMax, Label: reasoningLabelMax},
		})
	reply := cfgReasoningCreateResult(t, &bound, nil, tc.want, true, []acpfront.ModelEntry{entry}, nil)
	h := newSetCfgFront(t, cfgOptionScenario(create, fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply:  &fake.Reply{Result: json.RawMessage(reply)},
	}))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid, options := openSetCfgSessionWithOptions(ctx, t, h)
	if tc.omitted != "" {
		assertThoughtOptionOmitted(t, options)
	}

	if _, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDThoughtLevel, tc.alias)); err != nil {
		t.Fatalf("thought alias %q: %v", tc.alias, err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertSetCfgModelFrame(t, assertSingleSetModel(t, tr), bound.ProviderID, bound.ModelID, tc.want)
	if closes := transcriptIndexes(tr, "in", `"method":"session/close"`); len(closes) != 0 {
		t.Fatalf("a successful thought alias must not close the session: %+v", closes)
	}
}

type thoughtAliasRefusalCase struct {
	name           string
	configID       string
	value          string
	current        string
	thoughtEnabled bool
	binding        string
	available      []acpfront.ModelEntry
	mustContain    []string
	wantMessage    string
}

func runThoughtAliasRefusal(t *testing.T, tc thoughtAliasRefusalCase) {
	t.Helper()

	bound := acpfront.ModelRef{ProviderID: reasoningSnapshotTargetProvider, ModelID: reasoningSnapshotTargetModel}
	var current, lastUsed *acpfront.ModelRef
	switch tc.binding {
	case reasoningBindingCurrent:
		current = &bound
	case reasoningBindingLastUsed:
		lastUsed = &bound
	}

	create := cfgReasoningCreateResult(t, current, lastUsed, tc.current, tc.thoughtEnabled, tc.available,
		[]acpfront.LevelEntry{
			{Value: reasoningLevelOff, Label: reasoningLabelOff},
			{Value: corpusReasoningLevel, Label: reasoningLabelHigh},
		})
	h := newSetCfgFront(t, cfgOptionScenario(create))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid := openSetCfgSession(ctx, t, h)
	_, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, tc.configID, tc.value))

	assertInvalidParamsNaming(t, err, tc.mustContain...)
	if tc.wantMessage != "" {
		var requestErr *acp.RequestError
		if !errors.As(err, &requestErr) {
			t.Fatalf("refusal must remain a RequestError: %v", err)
		}

		if requestErr.Message != tc.wantMessage {
			t.Fatalf("refusal message = %q, want existing shape %q", requestErr.Message, tc.wantMessage)
		}
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertNoBackendMutation(t, tr)
	if closes := transcriptIndexes(tr, "in", `"method":"session/close"`); len(closes) != 0 {
		t.Fatalf("a pre-wire alias refusal must not close the session: %+v", closes)
	}
}

// TestSetConfigOptionOrcaThoughtAliases locks exact enabled/disabled normalization
// before option lookup and membership validation, including omitted selects and
// fail-closed metadata/validation controls.
//
//nolint:funlen // the success/refusal matrices are one alias trust-boundary contract
func TestSetConfigOptionOrcaThoughtAliases(t *testing.T) {
	bound := acpfront.ModelRef{ProviderID: reasoningSnapshotTargetProvider, ModelID: reasoningSnapshotTargetModel}
	lowHigh := cfgReasoningEntry(bound, reasoningLevelLow, corpusReasoningLevel)
	lowHigh.DefaultLevel = corpusReasoningLevel
	offHigh := cfgReasoningEntry(bound, reasoningLevelOff, corpusReasoningLevel)
	offHigh.DefaultLevel = corpusReasoningLevel
	levelLess := cfgReasoningEntry(bound)
	otherRef := acpfront.ModelRef{ProviderID: reasoningSnapshotProvider, ModelID: reasoningSnapshotSource}
	other := cfgReasoningEntry(otherRef, corpusReasoningLevel)

	for _, tc := range []thoughtAliasSuccessCase{
		{
			name: "enabled preserves supported current low", alias: orcaThoughtEnabled, current: reasoningLevelLow,
			levels: []string{reasoningLevelLow, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			thoughtEnabled: true, want: reasoningLevelLow,
		},
		{
			name: "enabled replaces stale max with advertised default", alias: orcaThoughtEnabled,
			current: reasoningLevelMax,
			levels:  []string{reasoningLevelLow, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			thoughtEnabled: true, want: corpusReasoningLevel,
		},
		{
			name: "enabled without default uses first low", alias: orcaThoughtEnabled, current: reasoningLevelMax,
			levels: []string{reasoningLevelLow, corpusReasoningLevel}, thoughtEnabled: true, want: reasoningLevelLow,
		},
		{
			name: "disabled maps to advertised off", alias: orcaThoughtDisabled, current: corpusReasoningLevel,
			levels: []string{reasoningLevelOff, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			thoughtEnabled: true, want: reasoningLevelOff,
		},
		{
			name: "enabled works with thought-disabled omitted select", alias: orcaThoughtEnabled,
			current: reasoningLevelMax,
			levels:  []string{reasoningLevelOff, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			omitted: "disabled", want: corpusReasoningLevel,
		},
		{
			name: "disabled works with thought-disabled omitted select", alias: orcaThoughtDisabled,
			current: reasoningLevelMax,
			levels:  []string{reasoningLevelOff, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			omitted: "disabled", want: reasoningLevelOff,
		},
		{
			name: "enabled works with current-empty omitted select", alias: orcaThoughtEnabled,
			levels: []string{reasoningLevelOff, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			thoughtEnabled: true, omitted: "no-current", want: corpusReasoningLevel,
		},
		{
			name: "disabled works with current-empty omitted select", alias: orcaThoughtDisabled,
			levels: []string{reasoningLevelOff, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			thoughtEnabled: true, omitted: "no-current", want: reasoningLevelOff,
		},
		{
			name: "enabled resolves the lastUsed binding", alias: orcaThoughtEnabled, current: reasoningLevelLow,
			levels: []string{reasoningLevelLow, corpusReasoningLevel}, defaultLevel: corpusReasoningLevel,
			thoughtEnabled: true, binding: reasoningBindingLastUsed, want: reasoningLevelLow,
		},
	} {
		t.Run(tc.name, func(t *testing.T) { runThoughtAliasSuccess(t, tc) })
	}

	for _, tc := range []thoughtAliasRefusalCase{
		{
			name:     "disabled without advertised off keeps listed-value refusal",
			configID: optionIDThoughtLevel, value: orcaThoughtDisabled, current: corpusReasoningLevel,
			thoughtEnabled: true, binding: reasoningBindingCurrent, available: []acpfront.ModelEntry{lowHigh},
			mustContain: []string{optionIDThoughtLevel, orcaThoughtDisabled, reasoningLevelLow, corpusReasoningLevel},
			wantMessage: `session/set_config_option: configId "thought_level": value "disabled" ` +
				`is not one of the listed values (low, high)`,
		},
		{
			name:     "enabled missing bound catalog entry fails closed",
			configID: optionIDThoughtLevel, value: orcaThoughtEnabled, current: reasoningLevelMax,
			thoughtEnabled: true, binding: reasoningBindingCurrent, available: []acpfront.ModelEntry{other},
			mustContain: []string{optionIDThoughtLevel, orcaThoughtEnabled, reasoningSnapshotTarget},
		},
		{
			name:     "enabled bound entry without levels fails closed",
			configID: optionIDThoughtLevel, value: orcaThoughtEnabled, current: reasoningLevelMax,
			thoughtEnabled: true, binding: reasoningBindingCurrent, available: []acpfront.ModelEntry{levelLess},
			mustContain: []string{optionIDThoughtLevel, orcaThoughtEnabled, reasoningSnapshotTarget},
		},
		{
			name:     "enabled missing binding fails closed",
			configID: optionIDThoughtLevel, value: orcaThoughtEnabled, current: reasoningLevelMax,
			thoughtEnabled: true, available: []acpfront.ModelEntry{lowHigh},
			mustContain: []string{optionIDThoughtLevel, orcaThoughtEnabled},
		},
		{
			name:     "disabled missing bound catalog entry fails closed",
			configID: optionIDThoughtLevel, value: orcaThoughtDisabled, current: corpusReasoningLevel,
			thoughtEnabled: true, binding: reasoningBindingCurrent, available: []acpfront.ModelEntry{other},
			mustContain: []string{optionIDThoughtLevel, orcaThoughtDisabled, reasoningSnapshotTarget},
		},
		{
			name:     "disabled bound entry without levels fails closed",
			configID: optionIDThoughtLevel, value: orcaThoughtDisabled, current: corpusReasoningLevel,
			thoughtEnabled: true, binding: reasoningBindingCurrent, available: []acpfront.ModelEntry{levelLess},
			mustContain: []string{optionIDThoughtLevel, orcaThoughtDisabled, reasoningSnapshotTarget},
		},
		{
			name:     "disabled missing binding fails closed",
			configID: optionIDThoughtLevel, value: orcaThoughtDisabled, current: corpusReasoningLevel,
			thoughtEnabled: true, available: []acpfront.ModelEntry{offHigh},
			mustContain: []string{optionIDThoughtLevel, orcaThoughtDisabled},
		},
		{
			name:     "non-alias unlisted thought value keeps listed-value validation",
			configID: optionIDThoughtLevel, value: reasoningLevelMax, current: reasoningLevelLow,
			thoughtEnabled: true, binding: reasoningBindingCurrent, available: []acpfront.ModelEntry{lowHigh},
			mustContain: []string{optionIDThoughtLevel, reasoningLevelMax, reasoningLevelLow, corpusReasoningLevel},
			wantMessage: `session/set_config_option: configId "thought_level": value "max" ` +
				`is not one of the listed values (low, high)`,
		},
		{
			name:     "unknown config id with alias-looking value keeps unknown-id validation",
			configID: unknownAliasConfigID, value: orcaThoughtEnabled, current: reasoningLevelLow,
			thoughtEnabled: true, binding: reasoningBindingCurrent, available: []acpfront.ModelEntry{lowHigh},
			mustContain: []string{unknownAliasConfigID, optionIDModel, optionIDThoughtLevel},
			wantMessage: `session/set_config_option: unknown configId "unknown" (allowed: model, thought_level)`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) { runThoughtAliasRefusal(t, tc) })
	}
}

// TestThoughtLevelAdvertisementRemainsVerbatimSelect proves the alias acceptance
// path does not enrich the ACP option: backend values stay verbatim in a select,
// with no aliases or boolean variant.
func TestThoughtLevelAdvertisementRemainsVerbatimSelect(t *testing.T) {
	bound := acpfront.ModelRef{ProviderID: reasoningSnapshotTargetProvider, ModelID: reasoningSnapshotTargetModel}
	entry := cfgReasoningEntry(bound, reasoningLevelOff, corpusReasoningLevel)
	entry.DefaultLevel = corpusReasoningLevel
	create := cfgReasoningCreateResult(t, &bound, nil, corpusReasoningLevel, true, []acpfront.ModelEntry{entry},
		[]acpfront.LevelEntry{
			{Value: reasoningLevelLow, Label: reasoningLabelLow},
			{Value: reasoningLevelMax, Label: reasoningLabelMax},
		})
	h := newSetCfgFront(t, cfgOptionScenario(create))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	_, options := openSetCfgSessionWithOptions(ctx, t, h)

	var thought *acp.SessionConfigOptionSelect
	for _, option := range options {
		if option.Boolean != nil && option.Boolean.Id == optionIDThoughtLevel {
			t.Fatalf("thought_level must not advertise a boolean variant: %+v", option.Boolean)
		}

		if option.Select != nil && option.Select.Id == optionIDThoughtLevel {
			thought = option.Select
		}
	}

	if thought == nil {
		t.Fatalf("thought_level select is missing: %+v", options)
	}

	var entries []acp.SessionConfigSelectOption
	if thought.Options.Ungrouped != nil {
		entries = *thought.Options.Ungrouped
	}

	want := []string{reasoningLevelOff, corpusReasoningLevel}
	if len(entries) != len(want) {
		t.Fatalf("thought select = %+v, want exactly %v", entries, want)
	}

	for i, value := range want {
		if string(entries[i].Value) != value {
			t.Fatalf("thought select[%d] = %q, want verbatim %q", i, entries[i].Value, value)
		}
	}
}
