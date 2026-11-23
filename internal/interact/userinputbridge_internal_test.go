package interact

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// TestHandleUserInputBridge — UIB-01/03/04: the policy row with a wired
// asker decodes the params, renders the ask on the asker, and maps the
// outcome into the dialect's answers merge content; an asker error declines
// with the reason; no asker — the legacy honest decline.
const bridgeAskParams = `{
	"requestId": "req-1", "sessionId": "sess-1",
	"prompt": "Pick one",
	"questions": [{"question": "Which phase?", "header": "Phase",
		"options": [{"value": "Verify 1", "label": "Verify 1"}], "multiSelect": false}]
}`

func TestHandleUserInputBridge(t *testing.T) {
	t.Run("the bridged ask renders and the outcome maps", testBridgeAcceptArm)
	t.Run("the asker error declines with the reason", testBridgeErrorArm)
	t.Run("no asker — the legacy decline", testBridgeLegacyArm)
}

func testBridgeAcceptArm(t *testing.T) {
	var gotAsk acpport.UserInputAsk
	asker := func(_ context.Context, ask acpport.UserInputAsk) (acpport.UserInputOutcome, error) {
		gotAsk = ask

		return acpport.UserInputOutcome{Action: "accept", Answers: map[string]string{"Which phase?": "Verify 1"}}, nil
	}

	r := NewResponder(WithUserInputAsker(asker))
	row := r.defaultPolicyTable()[MethodRequestUserInput]

	ans, err := row.Handler(context.Background(), Request{
		Method: MethodRequestUserInput, ReqID: "req-1", SessionID: "sess-1", Params: json.RawMessage(bridgeAskParams),
	})
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	if gotAsk.Prompt != "Pick one" || len(gotAsk.Questions) != 1 {
		t.Fatalf("the asker received %+v, want the decoded ask", gotAsk)
	}

	result, ok := ans.Result.(map[string]any)
	if !ok {
		t.Fatalf("answer result = %+v, want the action/content map", ans.Result)
	}

	if result["action"] != "accept" {
		t.Fatalf("action = %v, want accept", result["action"])
	}

	answers, ok := result["content"].(map[string]any)["answers"].(map[string]any)
	if !ok || answers["Which phase?"] != "Verify 1" {
		t.Fatalf("content.answers = %+v, want the question-keyed label", result["content"])
	}
}

func testBridgeErrorArm(t *testing.T) {
	asker := func(_ context.Context, _ acpport.UserInputAsk) (acpport.UserInputOutcome, error) {
		return acpport.UserInputOutcome{}, context.DeadlineExceeded
	}

	failing := NewResponder(WithUserInputAsker(asker))
	row := failing.defaultPolicyTable()[MethodRequestUserInput]

	ans, err := row.Handler(
		context.Background(),
		Request{Method: MethodRequestUserInput, ReqID: "r", Params: json.RawMessage(bridgeAskParams)},
	)
	if err != nil {
		t.Fatalf("the asker error must not fail the handler: %v", err)
	}

	if result, ok := ans.Result.(map[string]any); !ok || result["action"] != "decline" {
		t.Fatalf("the error arm answered %+v, want decline", ans.Result)
	}
}

func testBridgeLegacyArm(t *testing.T) {
	legacy := NewResponder()
	row := legacy.defaultPolicyTable()[MethodRequestUserInput]

	ans, err := row.Handler(
		context.Background(),
		Request{Method: MethodRequestUserInput, ReqID: "r", Params: json.RawMessage(bridgeAskParams)},
	)
	if err != nil {
		t.Fatalf("legacy handler: %v", err)
	}

	if result, ok := ans.Result.(map[string]any); !ok || result["action"] != "decline" {
		t.Fatalf("the legacy row answered %+v, want decline", ans.Result)
	}
}
