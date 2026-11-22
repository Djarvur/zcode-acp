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
	rules := []fake.RequestRule{
		{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: json.RawMessage(createResult)}},
		{
			Method: zcode.MethodSessionSubscribe,
			Reply:  &fake.Reply{Result: json.RawMessage(`{"sessionId":"sess_cfg","eventSeq":0,"events":[]}`)},
		},
	}

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

	return string(resp.SessionId)
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

		if _, closeErr := h.client.CloseSession(ctx, acp.CloseSessionRequest{SessionId: acp.SessionId(sid)}); closeErr != nil {
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
