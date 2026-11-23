package acpfront_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const cpChatSessionID = "sess_cp_chat"

// The chat-session legs for the catalog-binding test: the chat create
// carries the SAME account catalog snapshot the probe create does (the
// level source for the launch binding), and the chat setModel answers the
// rebind (the backend's provider-scoped view).
// cpAccountEntry renders one account catalog entry with the levels.
func cpAccountEntry(modelID string) string {
	levels := `"reasoning":{"levels":[{"value":"low"},{"value":"high"},{"value":"max"}],"defaultLevel":"low"}`

	return `{"ref":{"providerId":"` + cpFixAccountProvider + `","modelId":"` + modelID + `"},` + levels + `}`
}

func cpChatCreateRule() fake.RequestRule {
	reply := `{"session":{"sessionId":"` + cpChatSessionID + `"},` +
		`"settings":{"model":{"available":[` +
		cpAccountEntry(cpFixAccountModelA) + `,` + cpAccountEntry(cpFixAccountModelB) + `]}}}`

	return fake.RequestRule{
		Method: zcode.MethodSessionCreate,
		Reply:  &fake.Reply{Result: json.RawMessage(reply)},
		SideEffects: []fake.SideEffect{
			cpProbeLifecycleEmit("starting", 1),
			cpProbeLifecycleEmit("ready", 2),
		},
	}
}

func cpChatSetModelRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply:  &fake.Reply{Result: json.RawMessage(`{}`)},
	}
}

// TestCodingPlanCatalogBinding — the 2026-11-23 field report: a coding-plan
// chat bound the builtin zai-api model, and the backend's provider-scoped
// settings view shrank the /model menu to that single model. The launch
// binding must map the builtin serialization to the SAME model through the
// account provider on a coding-plan-authenticated session — the create-time
// full catalog then survives the follow-up.
func TestCodingPlanCatalogBinding(t *testing.T) {
	h, _, _ := newCodingPlanProbeFront(t, cpCatalogBindingRules(),
		acpfront.WithModel(acpfront.ModelSelection{
			ProviderID: "zai-api",
			ModelID:    cpFixAccountModelB,
		}))

	if _, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan); err != nil {
		t.Fatalf("authenticate coding-plan: %v", err)
	}

	newSession(t, h)

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	chatBinding := ""

	for _, l := range tr.Lines {
		if l.Dir == "in" && strings.Contains(l.JSON, `"session/setModel"`) &&
			strings.Contains(l.JSON, cpChatSessionID) {
			chatBinding = l.JSON
		}
	}

	if chatBinding == "" {
		var all []string

		for _, l := range tr.Lines {
			if strings.Contains(l.JSON, "setModel") {
				snippet := l.JSON
				if len(snippet) > 300 {
					snippet = snippet[:300]
				}

				all = append(all, string(l.Dir)+": "+snippet)
			}
		}

		t.Fatal("the chat session recorded no chat setModel — setModel frames seen: " +
			strings.Join(all, " || "))
	}

	if !strings.Contains(chatBinding, `"providerId":"`+cpFixAccountProvider+`"`) {
		t.Fatalf("the chat binding = %s, want the account provider — "+
			"the builtin rebind shrinks the /model catalog to the builtin scope", chatBinding)
	}

	if !strings.Contains(chatBinding, `"modelId":"`+cpFixAccountModelB+`"`) {
		t.Fatalf("the chat binding = %s, want the same modelId verbatim", chatBinding)
	}

	if !strings.Contains(chatBinding, `"reasoningLevel":"low"`) {
		t.Fatalf("the chat binding = %s, want the snapshot-derived reasoning level", chatBinding)
	}
}

// newSession opens the harness's chat session (the catalog-binding tests'
// shared leg).
func newSession(t *testing.T, h *FrontHarness) {
	t.Helper()

	if _, err := h.client.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd:        cpCycleWorkspace,
		McpServers: []acp.McpServer{},
	}); err != nil {
		t.Fatalf("session/new: %v", err)
	}
}

// cpCatalogBindingRules assembles the two-session leg set: the authenticate
// probe chain plus the chat session's create/subscribe/rebind.
func cpCatalogBindingRules() []fake.RequestRule {
	chatSubscribe := fake.RequestRule{
		Method: zcode.MethodSessionSubscribe,
		Reply: &fake.Reply{Result: json.RawMessage(
			`{"sessionId":"` + cpChatSessionID + `","eventSeq":0,"events":[]}`)},
	}

	return []fake.RequestRule{
		cpAckRule(),
		cpRefreshRule(0, 0, false),
		cpProbeCreateRule(true),
		cpProbeSetModelRule(),
		cpProbeCloseRule(),
		cpChatCreateRule(),
		chatSubscribe,
		cpChatSetModelRule(),
	}
}
