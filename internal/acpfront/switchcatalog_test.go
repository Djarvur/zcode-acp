package acpfront_test

import (
	"encoding/json"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The switch-path catalog fixtures: a two-model create snapshot, and a
// setModel reply whose settings view is PROVIDER-SCOPED (only the switched
// model's entry) — the 2026-11-23 field report's exact shape.
const (
	switchCreateResult = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3"}, "label": "GLM-5.3"},
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash"}
      ]
    },
    "thoughtLevel": {"enabled": true, "current": "high", "available": [{"value": "low"}, {"value": "high"}]}
  }
}`

	switchNarrowReply = `{
  "session": {"sessionId": "sess_cfg"},
  "settings": {
    "model": {
      "current": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"},
      "available": [
        {"ref": {"providerId": "zai-api", "modelId": "GLM-5.3-Flash"}, "label": "GLM-5.3-Flash"}
      ]
    },
    "thoughtLevel": {"enabled": true, "current": "low", "available": [{"value": "low"}, {"value": "high"}]}
  }
}`
)

// TestSetConfigOptionKeepsFullCatalog — the switch-path catalog contract
// (the 2026-11-23 field report's root): the backend scopes the setModel
// reply's model.available to the newly bound model's provider — the reply
// the client adopts must carry the PREVIOUS full catalog with the CONFIRMED
// new current, never the narrowed view (the picker shrank to one model on
// every switch).
func TestSetConfigOptionKeepsFullCatalog(t *testing.T) {
	h := newSetCfgFront(t, cfgOptionScenario(switchCreateResult, fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply:  &fake.Reply{Result: json.RawMessage(switchNarrowReply)},
	}))

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	sid := openSetCfgSession(ctx, t, h)

	ask := setCfgAsk(sid, optionIDModel, corpusModelFlashID)

	resp, err := h.client.SetSessionConfigOption(ctx, ask)
	if err != nil {
		t.Fatalf("model switch: %v", err)
	}

	var selectVariant *acp.SessionConfigOptionSelect

	for i := range resp.ConfigOptions {
		if resp.ConfigOptions[i].Select != nil && resp.ConfigOptions[i].Select.Id == optionIDModel {
			selectVariant = resp.ConfigOptions[i].Select
		}
	}

	if selectVariant == nil {
		t.Fatal("the switch reply carries no model select option")
	}

	if selectVariant.CurrentValue != corpusModelFlashID {
		t.Fatalf("currentValue = %q, want the confirmed switch target", selectVariant.CurrentValue)
	}

	if selectVariant.Options.Ungrouped == nil {
		t.Fatal("the model option carries no ungrouped options")
	}

	values := make([]string, 0, len(*selectVariant.Options.Ungrouped))

	for _, o := range *selectVariant.Options.Ungrouped {
		values = append(values, string(o.Value))
	}

	if len(values) != 2 || values[0] != corpusModelPlainID || values[1] != corpusModelFlashID {
		t.Fatalf("model entries = %v, want the FULL catalog under the confirmed current "+
			"(the narrow reply must not shrink the picker)", values)
	}
}
