package acpport_test

import (
	"context"
	"io"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/coder/acp-go-sdk"
)

const configOptionUpdateFrame = `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"sess_1","update":{"configOptions":[{"category":"model","currentValue":"zai-api/GLM-5.3-Flash","id":"model","name":"Model","options":[{"description":"The flagship reasoning model","name":"GLM-5.3-Flash","value":"zai-api/GLM-5.3-Flash"},{"name":"GLM-5.3","value":"zai-api/GLM-5.3"}],"type":"select"},{"category":"thought_level","currentValue":"high","id":"thought_level","name":"Thought level","options":[{"description":"deep reasoning","name":"High","value":"high"},{"name":"Low","value":"low"}],"type":"select"}],"sessionUpdate":"config_option_update"}}}`

// catalogHandlers opts into the front-originated asynchronous bindings while
// retaining the scripted Handlers behavior.
type catalogHandlers struct {
	*scriptedHandlers
	boundSink acpport.UpdateSink
	refreshes int
}

func (h *catalogHandlers) BindUpdateSink(sink acpport.UpdateSink) {
	h.boundSink = sink
}

func (h *catalogHandlers) RequestCatalogRefresh(context.Context) {
	h.refreshes++
}

func TestConfigOptionUpdateWire(t *testing.T) {
	clientToAgentR, clientToAgentW := io.Pipe()
	agentToClientR, agentToClientW := io.Pipe()
	t.Cleanup(func() {
		for _, c := range []io.Closer{clientToAgentR, clientToAgentW, agentToClientR, agentToClientW} {
			_ = c.Close()
		}
	})

	h := &catalogHandlers{scriptedHandlers: newScriptedHandlers()}
	_, sink, _ := acpport.NewAgentSideConnection(acpport.NewAgent(h), agentToClientW, clientToAgentR)
	lines := make(chan string, 4)
	go drainFrames(agentToClientR, lines)

	options := []acpport.ConfigOptionDef{
		{
			ID: "model", Name: "Model", Category: acpport.ConfigCategoryModel,
			CurrentValue: "zai-api/GLM-5.3-Flash",
			Options: []acpport.SelectOptionDef{
				{Value: "zai-api/GLM-5.3-Flash", Name: "GLM-5.3-Flash", Description: "The flagship reasoning model"},
				{Value: "zai-api/GLM-5.3", Name: "GLM-5.3"},
			},
		},
		{
			ID: "thought_level", Name: "Thought level", Category: acpport.ConfigCategoryThoughtLevel,
			CurrentValue: "high",
			Options: []acpport.SelectOptionDef{
				{Value: "high", Name: "High", Description: "deep reasoning"},
				{Value: "low", Name: "Low"},
			},
		},
	}
	if err := sink(context.Background(), acpport.SessionNote{
		SessionID: testSessionID, Kind: acpport.NoteConfigOptions, ConfigOptions: options,
	}); err != nil {
		t.Fatalf("sink(config options): %v", err)
	}

	if got := readFrame(t, lines); got != configOptionUpdateFrame {
		t.Fatalf("config option frame drift:\n got: %s\nwant: %s", got, configOptionUpdateFrame)
	}
}

func TestAsyncSinkBinding(t *testing.T) {
	h := &catalogHandlers{scriptedHandlers: newScriptedHandlers()}
	toAgentR, toAgentW := io.Pipe()
	fromAgentR, fromAgentW := io.Pipe()
	t.Cleanup(func() {
		for _, c := range []io.Closer{toAgentR, toAgentW, fromAgentR, fromAgentW} {
			_ = c.Close()
		}
	})

	_, sink, _ := acpport.NewAgentSideConnection(acpport.NewAgent(h), fromAgentW, toAgentR)
	if h.boundSink == nil {
		t.Fatal("construction returned without binding the asynchronous update sink")
	}

	lines := make(chan string, 1)
	go drainFrames(fromAgentR, lines)
	if err := h.boundSink(context.Background(), acpport.SessionNote{
		SessionID: testSessionID, Kind: acpport.NoteMessage, Delta: "bound",
	}); err != nil {
		t.Fatalf("bound sink: %v", err)
	}
	if sink == nil {
		t.Fatal("returned connection sink is nil")
	}
	readFrame(t, lines)
}

func TestAuthenticateRequestsCatalogRefresh(t *testing.T) {
	h := &catalogHandlers{scriptedHandlers: newScriptedHandlers()}
	adapter := acpport.NewAgent(h)
	ctx := context.Background()

	if _, err := adapter.Authenticate(ctx, acp.AuthenticateRequest{MethodId: authDelegationMethodID}); err != nil {
		t.Fatalf("successful Authenticate: %v", err)
	}
	if h.refreshes != 1 {
		t.Fatalf("refresh requests = %d, want 1 after successful authenticate", h.refreshes)
	}

	h.authErr = errAuthScriptedFailure
	if _, err := adapter.Authenticate(ctx, acp.AuthenticateRequest{MethodId: authDelegationMethodID}); err == nil {
		t.Fatal("scripted authenticate failure must remain an error")
	}
	if h.refreshes != 1 {
		t.Fatalf("refresh requests = %d, want no additional request after failed authenticate", h.refreshes)
	}
}
