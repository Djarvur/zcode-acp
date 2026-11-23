package acpfront_test

import (
	"context"
	"testing"

	"encoding/json"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// TestSilentAuthFillsCatalogOnNewSession — the discovery sessions' catalog
// fix (UIB follow-up, the 2026-11-23 field report): a session/new WITHOUT
// the card's explicit authenticate (the client's command/model catalog
// probes run in fresh wrapper processes) still serves the ACCOUNT catalog —
// the silent authenticate chain runs (all-local: the keyring + the
// entitlement + the PAT + the read-back) before the create, so the probe
// sessions carry the same model option the chat does. The picker's menu
// never shrinks to the builtin scope.
func TestSilentAuthFillsCatalogOnNewSession(t *testing.T) {
	h, _, _ := newCodingPlanProbeFront(t, cpCatalogBindingRules())

	if _, err := h.client.Initialize(context.Background(), acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sn, err := h.client.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd:        "/tmp/happy-turn-ws",
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	var selectVariant *acp.SessionConfigOptionSelect

	for i := range sn.ConfigOptions {
		if sn.ConfigOptions[i].Select != nil && sn.ConfigOptions[i].Select.Id == optionIDModel {
			selectVariant = sn.ConfigOptions[i].Select
		}
	}

	if selectVariant == nil {
		t.Fatalf("session/new carries no model select option (%d config options, %d tee lines)",
			len(sn.ConfigOptions), len(h.ACPLines()))
	}

	if selectVariant.CurrentValue != "zai-api/GLM-5.3-Flash" {
		t.Fatalf("currentValue = %q, want the builtin binding (the silent auth carries no model rebind)",
			selectVariant.CurrentValue)
	}

	ungrouped := selectVariant.Options.Ungrouped

	values := make([]string, 0, len(*ungrouped))

	for _, o := range *ungrouped {
		values = append(values, string(o.Value))
	}

	if len(values) != 2 || values[0] != "account:zai-individual-coding-plan/GLM-5.3" ||
		values[1] != "account:zai-individual-coding-plan/GLM-5.3-Flash" {
		t.Fatalf("model entries = %v, want the ACCOUNT catalog (the silent authenticate "+
			"registered the family before the create)", values)
	}
}

// TestSilentAuthKeyringFailureStillOpens — the best-effort arm: the keyring
// probe failing means the silent authenticate fails; the session STILL
// opens with the builtin scope (never a session/open failure).
func TestSilentAuthKeyringFailureStillOpens(t *testing.T) {
	current := `{"providerId":"zai-api","modelId":"GLM-5.3-Flash","options":{"reasoningLevel":"max"}}`
	lastUsed := `{"providerId":"zai-api","modelId":"GLM-5.3-Flash"}`
	available := `[{"ref":{"providerId":"zai-api","modelId":"GLM-5.3-Flash"}}]`

	chatCreate := fake.RequestRule{
		Method: zcode.MethodSessionCreate,
		Reply: &fake.Reply{Result: json.RawMessage(`{"session":{"sessionId":"` + cpChatSessionID + `"},` +
			`"settings":{"model":{"current":` + current + `,"lastUsed":` + lastUsed + `,"available":` + available + `}}}`)},
		SideEffects: []fake.SideEffect{
			cpProbeLifecycleEmit("starting", 1),
			cpProbeLifecycleEmit("ready", 2),
		},
	}

	// The keyring seam fails: the coding-plan chain cannot even start.
	failing := &recordingProbe{err: context.DeadlineExceeded}

	cfg, _, _ := codingPlanConfig(t)
	cfg.Probe = failing.probeKeyring

	h, _, _ := newCodingPlanProbeFront(t, []fake.RequestRule{
		chatCreate,
		fake.RequestRule{
			Method: zcode.MethodSessionSubscribe,
			Reply: &fake.Reply{Result: json.RawMessage(
				`{"sessionId":"` + cpChatSessionID + `","eventSeq":0,"events":[]}`)},
		},
	}, acpfront.WithAuth(cfg))

	if _, err := h.client.Initialize(context.Background(), acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sn, err := h.client.NewSession(context.Background(), acp.NewSessionRequest{
		Cwd:        "/tmp/happy-turn-ws",
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("session/new after a failed silent auth: %v", err)
	}

	var modelSeen bool

	for i := range sn.ConfigOptions {
		if sn.ConfigOptions[i].Select != nil && sn.ConfigOptions[i].Select.Id == optionIDModel {
			modelSeen = true
		}
	}

	if !modelSeen {
		t.Fatal("the model option vanished after a failed silent auth — the builtin scope must still ride")
	}
}
