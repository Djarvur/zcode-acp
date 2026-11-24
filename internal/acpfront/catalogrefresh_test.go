package acpfront_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

func liveCatalogCreate(sid, current string, entries ...string) json.RawMessage {
	available := make([]any, 0, len(entries))
	for _, id := range entries {
		parts := strings.SplitN(id, "/", 2)
		available = append(available, map[string]any{
			"ref":   map[string]any{"providerId": parts[0], "modelId": parts[1]},
			"label": strings.TrimPrefix(parts[1], "GLM-"),
		})
	}
	ref := func(id string) any {
		parts := strings.SplitN(id, "/", 2)

		return map[string]any{"providerId": parts[0], "modelId": parts[1]}
	}
	raw, err := json.Marshal(map[string]any{"session": map[string]any{"sessionId": sid},
		"settings": map[string]any{"model": map[string]any{
			"current": ref(current), "available": available,
		}}})
	if err != nil {
		panic(err) // fixture contains only JSON-safe scalar/map/slice values
	}

	return raw
}

func liveCatalogScenario() *fake.Scenario {
	patch := json.RawMessage(`{"patch":{"providers":{"zai-api":{"models":["GLM-5.3","GLM-5.4"]}}}}`)
	registryEffects := []fake.SideEffect{
		{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: patch}},
		{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: patch}},
		{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(
			`{"sessionId":"sess_a","type":"turn.started","turnId":"turn_live"}`)}},
		{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(
			`{"sessionId":"sess_a","type":"turn.completed","turnId":"turn_live"}`)}},
	}

	return &fake.Scenario{
		Name: "live-catalog-refresh",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionCreate, Times: 1,
				Reply: &fake.Reply{Result: liveCatalogCreate("sess_a", "zai-api/GLM-5.2")}},
			{Method: zcode.MethodSessionCreate, Times: 1,
				Reply: &fake.Reply{Result: liveCatalogCreate("sess_b", "zai-api/GLM-5.3")}},
			{Method: zcode.MethodSessionCreate, Times: 1, DelayMs: 50,
				Reply: &fake.Reply{Result: liveCatalogCreate("sess_probe", "zai-api/GLM-5.3",
					"zai-api/GLM-5.3", "zai-api/GLM-5.4")}},
			{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{Result: json.RawMessage(`{"events":[]}`)}},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{Result: json.RawMessage(`{"closed":true}`)}},
			{Method: zcode.MethodProviderRefreshRegistry, DelayMs: 50,
				Reply: &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)}},
			{Method: zcode.MethodSessionSend, Reply: &fake.Reply{Result: json.RawMessage(`{"accepted":true}`)},
				SideEffects: registryEffects},
		},
		EOFDrainMs: 1,
	}
}

func liveCatalogExpected(sid string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":%q,`+
		`"update":{"configOptions":[{"category":"model","currentValue":"zai-api/GLM-5.3",`+
		`"id":"model","name":"Model","options":[{"name":"5.3","value":"zai-api/GLM-5.3"},`+
		`{"name":"5.4","value":"zai-api/GLM-5.4"}],"type":"select"}],`+
		`"sessionUpdate":"config_option_update"}}}`, sid)
}

//nolint:gocognit,gocyclo,cyclop,funlen // one composed backend-to-two-client proof keeps causal ordering explicit
func TestLiveCatalogRefresh(t *testing.T) {
	h := ComposeFront(t, liveCatalogScenario(), func(conn *zcode.Conn) acp.Agent {
		return acpport.NewAgent(acpfront.New(conn))
	})
	ctx, cancel := h.ExchangeCtx()
	defer cancel()
	if _, err := h.client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	a, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatalf("session A: %v", err)
	}
	b, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatalf("session B: %v", err)
	}
	if _, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: a.SessionId, Prompt: []acp.ContentBlock{acp.TextBlock("change catalog")},
	}); err != nil {
		t.Fatalf("prompt: %v", err)
	}

	want := map[string]string{string(a.SessionId): liveCatalogExpected(string(a.SessionId)),
		string(b.SessionId): liveCatalogExpected(string(b.SessionId))}
	deadline := time.Now().Add(5 * time.Second)
	for len(want) > 0 && time.Now().Before(deadline) {
		for _, line := range h.ACPLines() {
			if line.Dir != "in" || !strings.Contains(line.JSON, `"config_option_update"`) {
				continue
			}
			for sid, expected := range want {
				if line.JSON == expected {
					delete(want, sid)
				}
			}
		}
		if len(want) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing exact catalog updates for sessions %v; ACP lines: %v", want, h.ACPLines())
	}

	tr, err := h.Transcript(t)
	if err != nil {
		t.Fatalf("transcript: %v", err)
	}
	counts := make(map[string]int)
	for _, line := range tr.Lines {
		var env struct {
			Method string `json:"method"`
		}
		if json.Unmarshal([]byte(line.JSON), &env) == nil && env.Method != "" {
			counts[env.Method]++
		}
	}
	if counts[zcode.MethodProviderRefreshRegistry] != 1 || counts[zcode.MethodSessionCreate] != 3 {
		t.Fatalf("backend calls = refresh %d, create %d; want 1 and 3 (duplicates coalesced)",
			counts[zcode.MethodProviderRefreshRegistry], counts[zcode.MethodSessionCreate])
	}
	refreshAt := transcriptNthMethod(tr, zcode.MethodProviderRefreshRegistry, 1)
	probeAt := transcriptNthMethod(tr, zcode.MethodSessionCreate, 3)
	closeAt := transcriptNthMethod(tr, zcode.MethodSessionClose, 1)
	if refreshAt >= probeAt || probeAt >= closeAt {
		t.Fatalf("probe order = refresh %d, create %d, close %d; want refresh < create < close",
			refreshAt, probeAt, closeAt)
	}
}

func transcriptNthMethod(tr fake.Transcript, method string, nth int) int {
	seen := 0
	for i, line := range tr.Lines {
		var env struct {
			Method string `json:"method"`
		}
		if json.Unmarshal([]byte(line.JSON), &env) == nil && env.Method == method {
			seen++
			if seen == nth {
				return i
			}
		}
	}

	return -1
}
