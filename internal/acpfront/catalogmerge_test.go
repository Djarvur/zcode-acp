package acpfront_test

import (
	"testing"

	"github.com/coder/acp-go-sdk"
)

// TestSessionNewKeepsFullCatalog — the catalog-preservation contract (the
// 2026-11-23 field report's root): the backend scopes settings.model.
// available to the session's CURRENT provider — the setModel rebind's reply
// narrows the scope to the bound model. session/new's configOptions must
// carry the CREATE snapshot's FULL entries (the picker's menu) with the
// rebind's CONFIRMED current (the model the session actually serves).
func TestSessionNewKeepsFullCatalog(t *testing.T) {
	h := newTracerFront(t, "catalog-merge")

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	sn, err := h.client.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        cpCycleWorkspace,
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
		t.Fatal("session/new carries no model select option")
	}

	if selectVariant.CurrentValue != acp.SessionConfigValueId(corpusModelSelection().ProviderID+"/"+corpusModelID) {
		t.Fatalf("currentValue = %q, want the rebind's confirmed model", selectVariant.CurrentValue)
	}

	ungrouped := *selectVariant.Options.Ungrouped

	values := make([]string, 0, len(ungrouped))

	for _, o := range ungrouped {
		values = append(values, string(o.Value))
	}

	if len(values) != 2 {
		t.Fatalf("model entries = %v, want BOTH catalog models (the rebind's narrowed reply "+
			"must not shrink the picker's menu)", values)
	}

	if values[0] != "zai-api/GLM-5.3" || values[1] != corpusModelSelection().ProviderID+"/"+corpusModelID {
		t.Fatalf("model entries = %v, want the create snapshot's catalog verbatim", values)
	}
}
