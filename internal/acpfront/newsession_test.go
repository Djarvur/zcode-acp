package acpfront_test

import (
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

// Front-tier session/new contract locks (plan 04-03, CORE-02/D-03/V5): each
// leg composes its own loop over the setmodel-followup corpus (its
// create/subscribe/setModel preamble IS the D-03 wire contract), asserting
// on the RECORDED back-leg fake transcript.
//
//   - TestNewSessionRejectsRelativeCwd is the RED target (T-4-01, RESEARCH
//     Security Domain V5): a RELATIVE cwd must fail with an
//     invalid-params-class error BEFORE any backend call — no session/create
//     frame on the wire. RED baseline: the tracer's Front passes the cwd
//     through unchecked and the create IS attempted.
//   - TestNewSessionAcceptsMcpServers locks D-03 accept-and-ignored: a
//     non-empty mcpServers list succeeds and never reaches the backend wire.
//   - TestNewSessionCreatesAndSetsModel locks the CORE-02 payload sequence:
//     create bare → subscribe desktop-continuous → setModel carrying
//     model.options.reasoningLevel + persistAsWorkspaceLastUsed:false.
//   - TestSetModelOmitsOptionsWithoutLevel locks the IN-04 omit-shape
//     (plan 05-05): a level-less model selection emits the documented wire
//     shape — model.providerId/modelId with NO options key at all.

// TestNewSessionRejectsRelativeCwd — the relative-cwd rejection (T-4-01):
// the back-leg transcript shows NO session/create frame and the client gets
// an invalid-params-class error naming the requirement (spec session-setup:
// cwd MUST be absolute).
func TestNewSessionRejectsRelativeCwd(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	_, newErr := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: "relative/path"})

	// The transcript assertion runs FIRST (deliberately): at RED the create
	// frame is on the wire and this is the planned failure point; at GREEN
	// the rejection happens before anything touches the back leg.
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if creates := transcriptIndexes(tr, "in", `"method":"session/create"`); len(creates) != 0 {
		t.Fatalf("relative-cwd session/new must be rejected BEFORE any backend call (T-4-01) — "+
			"back leg records %d session/create frame(s)", len(creates))
	}

	// The rejection IS an invalid-params-class JSON-RPC error naming the
	// requirement.
	if newErr == nil {
		t.Fatal("session/new with a relative cwd must fail, got success")
	}

	var rerr *acp.RequestError
	if !errors.As(newErr, &rerr) {
		t.Fatalf("relative-cwd rejection must surface as a coded RequestError, got %T: %v", newErr, newErr)
	}

	if rerr.Code != -32602 {
		t.Fatalf("relative-cwd error code = %d, want -32602 (invalid params class)", rerr.Code)
	}

	if !strings.Contains(rerr.Message, "absolute") {
		t.Fatalf("relative-cwd error message %q must name the requirement (cwd must be absolute)", rerr.Message)
	}
}

// TestNewSessionAcceptsMcpServers — D-03 accept-and-ignored: a non-empty
// mcpServers list (one HTTP entry, name + url shape) succeeds and returns a
// session id, and the backend create frame carries NO mcp fields — the list
// never reaches the backend wire.
func TestNewSessionAcceptsMcpServers(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	newResp, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd: t.TempDir(),
		McpServers: []acp.McpServer{
			{Http: &acp.McpServerHttpInline{Name: "docs-mcp", Url: "https://mcp.example.invalid/s"}},
		},
	})
	if err != nil {
		t.Fatalf("session/new with a non-empty mcpServers list must succeed: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	createIdx := transcriptIndex(t, tr, "in", `"method":"session/create"`, "create request")
	createRaw := tr.Lines[createIdx].JSON

	if strings.Contains(createRaw, "mcp") {
		t.Fatalf("mcp content must never reach the backend wire (D-03 accept-and-ignored): %s", createRaw)
	}
}

// TestNewSessionCreatesAndSetsModel — the CORE-02 lock over the tracer's
// D-03 sequence: session/new with a model configured drives create →
// subscribe → setModel on the back leg in wire order; the setModel IN frame
// carries model.options.reasoningLevel and explicit
// persistAsWorkspaceLastUsed:false (Open Question 3); the create IN frame
// stays BARE — no model object (Pitfall 4 regression lock).
func TestNewSessionCreatesAndSetsModel(t *testing.T) {
	h := newTracerFront(t, setModelScenario)

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

	sid := string(newResp.SessionId)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	createIdx := transcriptIndex(t, tr, "in", `"method":"session/create"`, "create request")
	subIdx := transcriptIndex(t, tr, "in", `"method":"session/subscribe"`, "subscribe request")
	setModelIdx := transcriptIndex(t, tr, "in", `"method":"session/setModel"`, "setModel request")

	if createIdx >= subIdx || subIdx >= setModelIdx {
		t.Fatalf("back-leg wire order violated: create(%d) subscribe(%d) setModel(%d)",
			createIdx, subIdx, setModelIdx)
	}

	assertCreateFrame(t, tr.Lines[createIdx].JSON)      // bare create — no model key (Pitfall 4)
	assertSubscribeFrame(t, tr.Lines[subIdx].JSON, sid) // desktop-continuous, this session
	assertSetModelFrame(t, tr.Lines[setModelIdx].JSON)  // reasoningLevel + persist:false
}

// TestSetModelOmitsOptionsWithoutLevel locks the IN-04 omit-shape (plan
// 05-05): a model selection configured WITHOUT a reasoning level emits the
// documented setModel shape — providerId/modelId ride, and the options
// object is OMITTED entirely (never `"options":{"reasoningLevel":""}`, the
// shape that is neither the documented omit-shape nor a valid level). The
// backend's own actionable refusal then surfaces for level-strict
// providers. The configured-WITH-level preamble stays byte-locked by the
// corpus (TestGolden/setmodel rows assert those bytes at the fake tier;
// TestNewSessionCreatesAndSetsModel asserts the same payload here).
func TestSetModelOmitsOptionsWithoutLevel(t *testing.T) {
	sc := mustLoadScenario(t, setModelScenario)

	var front *acpfront.Front

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front = acpfront.New(conn, acpfront.WithModel(acpfront.ModelSelection{
			ProviderID: corpusProviderID,
			ModelID:    corpusModelID,
			// ReasoningLevel deliberately empty — the omit-shape arm.
		}))

		return acpport.NewAgent(front)
	})

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	if _, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()}); err != nil {
		t.Fatalf("session/new: %v", err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	setModelIdx := transcriptIndex(t, tr, "in", `"method":"session/setModel"`, "setModel request")

	var frame struct {
		Params struct {
			Model struct {
				ProviderID string          `json:"providerId"`
				ModelID    string          `json:"modelId"`
				Options    json.RawMessage `json:"options"`
			} `json:"model"`
		} `json:"params"`
	}

	raw := tr.Lines[setModelIdx].JSON
	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode setModel frame: %v (%s)", err, raw)
	}

	if len(frame.Params.Model.Options) != 0 {
		t.Fatalf("level-less setModel carries an options object (IN-04: the omit-shape is the contract): %s", raw)
	}

	model := frame.Params.Model
	if model.ProviderID != corpusProviderID || model.ModelID != corpusModelID {
		t.Fatalf("setModel model = %s/%s, want %s/%s (the selection rides without options)",
			model.ProviderID, model.ModelID, corpusProviderID, corpusModelID)
	}
}

// settingsCreateResult is the config-options create reply: a full settings
// snapshot (RESEARCH Pattern 6 schema) whose model.current binds a model
// DIFFERENT from the follow-up selection — the D-10 follow-up-overrides
// vector needs the two snapshots to disagree. The available list carries
// per-model reasoning levels on the Flash entry, a plain entry, and an
// account-prefixed disabled entry (the D-11 both-paths fixture), plus a
// thoughtLevel block.
const settingsCreateResult = `{
  "session": {"sessionId": "sess_opts", "protocol": {"name": "ZCode Protocol", "version": 1}},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash",
         "description": "The flagship reasoning model",
         "reasoning": {"levels": [
            {"value": "high", "label": "High", "description": "deep reasoning"},
            {"value": "low", "label": "Low"}], "defaultLevel": "high"}},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"},
        {"ref": {"providerId": "zai-account", "modelId": "GLM-5-turbo"}, "label": "GLM-5 Turbo",
         "description": "Legacy plan model", "disabledReason": "deprecated on this account"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "low", "defaultLevel": "low",
      "available": [{"value": "off", "label": "Off"}, {"value": "low", "label": "Low"}]
    }
  }
}`

// settingsSetModelResult is the setModel FOLLOW-UP reply: the CONFIRMED
// snapshot — model.current now binds the configured selection
// (zai-api/GLM-5.3-Flash) and thoughtLevel.current moved to high. This is
// the reply the D-10 currentValue must read (the create reply's older
// current must NOT survive it).
const settingsSetModelResult = `{
  "session": {"sessionId": "sess_opts"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash",
         "description": "The flagship reasoning model",
         "reasoning": {"levels": [
            {"value": "high", "label": "High", "description": "deep reasoning"},
            {"value": "low", "label": "Low"}], "defaultLevel": "high"}},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"},
        {"ref": {"providerId": "zai-account", "modelId": "GLM-5-turbo"}, "label": "GLM-5 Turbo",
         "description": "Legacy plan model", "disabledReason": "deprecated on this account"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "high", "defaultLevel": "high",
      "available": [{"value": "off", "label": "Off"}, {"value": "high", "label": "High"}]
    }
  }
}`

// settingsScenario composes the per-arm fake: one create reply + one
// subscribe reply, and the setModel reply ONLY when setModelResult is
// non-empty (the no-model arm never issues the follow-up). Requests the
// scenario did not script fail loud (the -32601 unknown-method reply — a
// silent {} reply would hide contract regressions).
func settingsScenario(createResult, setModelResult string) *fake.Scenario {
	rules := []fake.RequestRule{
		{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{Result: json.RawMessage(createResult)}},
		{
			Method: zcode.MethodSessionSubscribe,
			Reply:  &fake.Reply{Result: json.RawMessage(`{"sessionId":"sess_opts","eventSeq":0,"events":[]}`)},
		},
	}

	if setModelResult != "" {
		rules = append(rules, fake.RequestRule{
			Method: zcode.MethodSessionSetModel,
			Reply:  &fake.Reply{Result: json.RawMessage(setModelResult)},
		})
	}

	return &fake.Scenario{
		Name:      "configoptions",
		OnRequest: rules,
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: fakeMethodNotFound},
		},
		EOFDrainMs: 100,
	}
}

// assertModelOptionCurrent digs the model option's select variant out of a
// client NewSession response and locks its currentValue + entry values
// verbatim (the wire is the contract).
func assertModelOptionCurrent(t *testing.T, resp acp.NewSessionResponse, wantCurrent string, wantValues []string) {
	t.Helper()

	if len(resp.ConfigOptions) == 0 {
		t.Fatalf("session/new returned no configOptions (the confirmed-reply snapshot never "+
			"reached the picker): %+v", resp)
	}

	model := resp.ConfigOptions[0].Select
	if model == nil {
		t.Fatalf("configOptions[0] carries no select variant: %+v", resp.ConfigOptions[0])
	}

	if model.Id != optionIDModel {
		t.Fatalf("configOptions[0].id = %q, want model", model.Id)
	}

	if model.CurrentValue != acp.SessionConfigValueId(wantCurrent) {
		t.Fatalf("model currentValue = %q, want the server-confirmed %q (D-10)", model.CurrentValue, wantCurrent)
	}

	var got []string
	if model.Options.Ungrouped != nil {
		for _, o := range *model.Options.Ungrouped {
			got = append(got, string(o.Value))
		}
	}

	if len(got) != len(wantValues) {
		t.Fatalf("model entries = %v, want %v", got, wantValues)
	}

	for i := range wantValues {
		if got[i] != wantValues[i] {
			t.Fatalf("model entries[%d] = %q, want %q (verbatim mirror, D-08)", i, got[i], wantValues[i])
		}
	}
}

// TestNewSessionConfigOptions — plan 06-04 Task 3 (MODEL-01 at the
// harness): session/new returns the configOptions built from the live
// backend reply snapshots — the setModel FOLLOW-UP reply is the D-10
// confirmation source when a model is configured, the create reply's
// snapshot when not, and an unconfirmed (current-less) snapshot omits the
// model entry entirely.
//
//nolint:funlen,gocognit,cyclop // each subtest is a full harness lock; splitting would scatter its sequencing
func TestNewSessionConfigOptions(t *testing.T) {
	t.Run("the setModel follow-up reply is the confirmation (D-10)", func(t *testing.T) {
		h := ComposeFront(t, settingsScenario(settingsCreateResult, settingsSetModelResult),
			func(conn *zcode.Conn) acp.Agent {
				front := acpfront.New(conn,
					acpfront.WithVersion("0.0.0-dev"),
					acpfront.WithModel(acpfront.ModelSelection{
						ProviderID: corpusProviderID, ModelID: corpusModelID, ReasoningLevel: corpusReasoningLevel,
					}))

				return acpport.NewAgent(front)
			})

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
			t.Fatalf("initialize: %v", err)
		}

		resp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("session/new: %v", err)
		}

		// The FOLLOW-UP reply's confirmed current — the create reply's
		// zai-api/GLM-5.3 must NOT survive it (the later reply overwrites).
		assertModelOptionCurrent(t, resp, corpusModelFlashID,
			[]string{corpusModelFlashID, corpusModelPlainID, corpusAccountModelID})

		// The thought-level option: entries from the BOUND model's
		// reasoning.levels (D-09), currentValue = the follow-up's confirmed
		// high — the create reply's low must NOT survive it.
		if len(resp.ConfigOptions) != 2 {
			t.Fatalf("configOptions = %d entries, want 2 (model + thought_level)", len(resp.ConfigOptions))
		}

		thought := resp.ConfigOptions[1].Select
		if thought == nil || thought.Id != optionIDThoughtLevel {
			t.Fatalf("configOptions[1] = %+v, want the thought_level select", resp.ConfigOptions[1])
		}

		if thought.CurrentValue != corpusReasoningLevel {
			t.Fatalf("thought_level currentValue = %q, want the confirmed high (D-10)", thought.CurrentValue)
		}

		if thought.Options.Ungrouped == nil || len(*thought.Options.Ungrouped) != 2 {
			t.Fatalf("thought_level entries = %+v, want the bound model's high/low levels (D-09)", thought.Options)
		}

		if (*thought.Options.Ungrouped)[0].Name != "High" || (*thought.Options.Ungrouped)[0].Description == nil ||
			*(*thought.Options.Ungrouped)[0].Description != "deep reasoning" {
			t.Fatalf("thought_level entries[0] = %+v, want {high, High, deep reasoning} verbatim",
				(*thought.Options.Ungrouped)[0])
		}
	})

	t.Run("the no-model arm renders from the create reply", func(t *testing.T) {
		h := ComposeFront(t, settingsScenario(settingsCreateResult, ""),
			func(conn *zcode.Conn) acp.Agent {
				return acpport.NewAgent(acpfront.New(conn))
			})

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
			t.Fatalf("initialize: %v", err)
		}

		resp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("session/new: %v", err)
		}

		// No follow-up fired — the create reply's own current is the
		// confirmed state.
		assertModelOptionCurrent(t, resp, corpusModelPlainID,
			[]string{corpusModelFlashID, corpusModelPlainID, corpusAccountModelID})
	})

	t.Run("a reply with no current and no lastUsed omits the model entry (D-10 + locked OQ2)", func(t *testing.T) {
		create := `{
		  "session": {"sessionId": "sess_unbound"},
		  "settings": {
		    "model": {
		      "available": [
		        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"}
		      ]
		    }
		  }
		}`

		h := ComposeFront(t, settingsScenario(create, ""),
			func(conn *zcode.Conn) acp.Agent {
				return acpport.NewAgent(acpfront.New(conn))
			})

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
			t.Fatalf("initialize: %v", err)
		}

		resp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("session/new: %v", err)
		}

		if len(resp.ConfigOptions) != 0 {
			t.Fatalf("an unbound session must answer NO configOptions entries (a smaller honest list "+
				"beats a fabricated currentValue, locked OQ2): %+v", resp.ConfigOptions)
		}
	})

	t.Run("the confirmed store is per-session — the second session renders its own snapshot", func(t *testing.T) {
		// Two create rules (budget 1 each, consumed in order): session A's
		// snapshot binds zai-api/GLM-5.3, session B's binds the Flash —
		// each response must reflect its OWN session's snapshot, never the
		// other's (the store is keyed by session id, the WR-07 bounding).
		createA := strings.Replace(settingsCreateResult, `"sess_opts"`, `"sess_optsA"`, 1)
		createB := strings.Replace(settingsSetModelResult, `"sess_opts"`, `"sess_optsB"`, 1)

		sc := &fake.Scenario{
			Name: "configoptions-two-sessions",
			OnRequest: []fake.RequestRule{
				{Method: zcode.MethodSessionCreate, Times: 1, Reply: &fake.Reply{Result: json.RawMessage(createA)}},
				{Method: zcode.MethodSessionCreate, Times: 1, Reply: &fake.Reply{Result: json.RawMessage(createB)}},
				{
					Method: zcode.MethodSessionSubscribe,
					Reply:  &fake.Reply{Result: json.RawMessage(`{"eventSeq":0,"events":[]}`)},
				},
			},
			UnknownMethod: &fake.Reply{
				Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: fakeMethodNotFound},
			},
			EOFDrainMs: 100,
		}

		h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
			return acpport.NewAgent(acpfront.New(conn))
		})

		ctx, cancel := h.ExchangeCtx()
		defer cancel()

		if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
			t.Fatalf("initialize: %v", err)
		}

		respA, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("session/new A: %v", err)
		}

		respB, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
		if err != nil {
			t.Fatalf("session/new B: %v", err)
		}

		assertModelOptionCurrent(t, respA, corpusModelPlainID,
			[]string{corpusModelFlashID, corpusModelPlainID, corpusAccountModelID})

		assertModelOptionCurrent(t, respB, corpusModelFlashID,
			[]string{corpusModelFlashID, corpusModelPlainID, corpusAccountModelID})
	})
}
