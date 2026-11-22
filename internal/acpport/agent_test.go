package acpport_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpport"
)

// acpport adapter contract tests (plan 04-02, D-01 seam lock): a scripted
// Handlers fake drives the adapter through the SDK's generated acp.Agent
// surface, and the wire frames it produces are asserted byte-for-byte —
// the honest D-02 initialize report, the -32601 defaults for the eight
// unadvertised stable methods, the stop-reason and error-code mappings,
// the delegation contracts, and the chunk shapes the sink emits.
//
// TestSDKImportBoundary pins the seam itself: no non-test file outside
// internal/acpport may import coder/acp-go-sdk (front-tier and cmd test
// files are exempt — RESEARCH Pattern 6: the SDK's client side is the
// honest counterparty).

// sdkModulePath is the import path prefix the seam lock forbids outside
// internal/acpport (production files).
const sdkModulePath = "github.com/coder/acp-go-sdk"

// frameWaitTimeout bounds each single-frame read in the sink wire test;
// a missing frame fails fast instead of hanging the exchange.
const frameWaitTimeout = 2 * time.Second

// Fixed synthetic ids for the contract tests.
const (
	testSessionID    = "sess_1"
	backendSessionID = "sess_backend"
	testCwd          = "/tmp/ws"
)

// errRawFront is the deliberately non-seam error in the coding table: a
// front failure that is NOT an *acpport.Error must still surface as a
// coded -32603, never raw (A5).
var errRawFront = errors.New("raw front failure")

// errAuthScriptedFailure is the authenticate delegation test's canned
// Handlers failure (the scripted counterparty's sentinel).
var errAuthScriptedFailure = errors.New("scripted auth failure")

// authDelegationMethodID is the delegation tests' method id (the
// advertised zai-api-key string, verbatim on the wire).
const authDelegationMethodID = "zai-api-key"

// TestSDKImportBoundary walks every non-test .go file in the repo and
// fails if any file outside internal/acpport imports the SDK — the D-01
// seam lock, enforced structurally (T-4-SC containment).
func TestSDKImportBoundary(t *testing.T) {
	root := filepath.Join("..", "..")

	var offenders []string

	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}

		if rel == "." {
			return nil // the repo root itself
		}

		if d.IsDir() {
			if prunedBoundaryDir(d.Name()) {
				return fs.SkipDir
			}

			return nil
		}

		if !isBoundaryCandidate(d.Name()) || inSeamDir(rel) {
			return nil
		}

		spec, parseErr := sdkImportSpec(path)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", rel, parseErr)
		}

		if spec != "" {
			offenders = append(offenders, rel+" -> "+spec)
		}

		return nil
	})

	if walkErr != nil {
		t.Fatalf("walk repo: %v", walkErr)
	}

	if len(offenders) > 0 {
		t.Fatalf("SDK imports outside internal/acpport (D-01 seam lock):\n%s", strings.Join(offenders, "\n"))
	}
}

// prunedBoundaryDir reports the directories the seam walk never descends
// into: VCS/testdata/vendor trees and hidden dirs.
func prunedBoundaryDir(name string) bool {
	switch name {
	case "testdata", "vendor", "node_modules":
		return true
	}

	return strings.HasPrefix(name, ".")
}

// isBoundaryCandidate reports whether the entry is a production Go file
// (test files are exempt — RESEARCH Pattern 6).
func isBoundaryCandidate(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// inSeamDir reports whether rel lives inside the sanctioned seam package.
func inSeamDir(rel string) bool {
	seam := filepath.Join("internal", "acpport")

	return rel == seam || strings.HasPrefix(rel, seam+string(filepath.Separator))
}

// sdkImportSpec returns the file's SDK import path, or "" when the file
// imports none.
func sdkImportSpec(path string) (string, error) {
	node, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return "", err
	}

	for _, imp := range node.Imports {
		spec, unquoteErr := strconv.Unquote(imp.Path.Value)
		if unquoteErr == nil && strings.HasPrefix(spec, sdkModulePath) {
			return spec, nil
		}
	}

	return "", nil
}

// honestReport builds the honest initialize payload (D-02,
// executed-probe shape): real agentInfo, both advertised auth methods,
// zero-value capabilities.
func honestReport() acpport.InitReport {
	return acpport.InitReport{
		Name:    "zcode-acp",
		Title:   "zcode-acp agent",
		Version: "0.0.0-dev",
		AuthMethods: []acpport.AuthMethodDef{
			{ID: "zai-api-key", Name: "z.ai API key"},
			{ID: "zai-individual-coding-plan", Name: "z.ai Individual Coding Plan"},
		},
	}
}

// honestInitializeFrame is the byte-exact wire shape the executed probe
// against v0.20.0 produced (RESEARCH Code Examples), WIDENED deliberately
// by plan 06-04 (MODEL-01, D-11): sessionCapabilities gains the options
// empty-struct opt-in — clients gate session/set_config_option on it. The
// agent-variant authMethods still carry no type discriminator, and no
// fs/terminal/loadSession surface appears anywhere.
const honestInitializeFrame = `{"agentCapabilities":{"auth":{},"mcpCapabilities":{},` +
	`"promptCapabilities":{},"sessionCapabilities":{"options":{}}},` +
	`"agentInfo":{"name":"zcode-acp","title":"zcode-acp agent","version":"0.0.0-dev"},` +
	`"authMethods":[{"id":"zai-api-key","name":"z.ai API key"},` +
	`{"id":"zai-individual-coding-plan","name":"z.ai Individual Coding Plan"}],"protocolVersion":1}`

// TestInitCapabilities — plan 06-04 (MODEL-01, D-11): initialize advertises
// sessionCapabilities.options, the SDK's empty-struct opt-in — clients MUST
// only send session/set_config_option when it is present. The advertisement
// is UNCONDITIONAL (D-11: the picker is auth-path-independent), and every
// phase-4 honesty lock (protocol version, real agentInfo, both auth
// methods, the byte-exact frame with zero fs/terminal) stays asserted
// alongside the widened shape.
func TestInitCapabilities(t *testing.T) {
	h := newScriptedHandlers()
	h.initReport = honestReport()

	resp, err := acpport.NewAgent(h).Initialize(
		context.Background(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// The SDK carries the capabilities block as a VALUE struct — the opt-in
	// is AgentCapabilities.SessionCapabilities.Options (the empty-struct
	// pointer; nil means unadvertised).
	if resp.AgentCapabilities.SessionCapabilities.Options == nil {
		t.Fatalf("initialize must advertise sessionCapabilities.options (the empty-struct opt-in, D-11): %+v",
			resp.AgentCapabilities.SessionCapabilities)
	}

	if resp.ProtocolVersion != acp.ProtocolVersionNumber {
		t.Fatalf("protocolVersion = %d, want %d", resp.ProtocolVersion, acp.ProtocolVersionNumber)
	}

	assertAgentInfo(t, resp.AgentInfo)
	assertAuthMethods(t, resp.AuthMethods)
	assertHonestFrame(t, resp)
}

// newSessionConfigOptionsFrame is the byte-exact configOptions assembly:
// each neutral option becomes the SDK select variant with the "type":
// "select" discriminant INJECTED BY THE SDK's MarshalJSON (never hand-
// marshaled, T-06-12), category/currentValue/id/name translating verbatim,
// and the ungrouped option entries as the flat array (description omitted
// when empty).
const newSessionConfigOptionsFrame = `{"configOptions":[` +
	`{"category":"model","currentValue":"zai-api/GLM-5.3-Flash","id":"model","name":"Model","options":[` +
	`{"description":"The flagship reasoning model","name":"GLM-5.3-Flash","value":"zai-api/GLM-5.3-Flash"},` +
	`{"name":"GLM-5 Turbo","value":"zai-account/GLM-5-turbo"}],"type":"select"},` +
	`{"category":"thought_level","currentValue":"high","id":"thought_level","name":"Thought level","options":[` +
	`{"description":"deep reasoning","name":"High","value":"high"},` +
	`{"name":"Low","value":"low"}],"type":"select"}],"sessionId":"sess_backend"}`

// TestNewSessionAssemblesConfigOptions — plan 06-04: the adapter assembles
// NewSessionResponse.ConfigOptions from the neutral options the front
// returned, exclusively via the SDK select variant, and the assembled union
// passes the SDK's own Validate (exactly one variant, name required).
func TestNewSessionAssemblesConfigOptions(t *testing.T) {
	// The canned neutral option list the scripted Handlers replay — the
	// adapter's select-assembly counterparty (model + thought_level, the
	// translation table's two entries; the provider ids mix zai-api and the
	// account prefix — D-11's both-paths posture).
	newSessionConfigOptions := []acpport.ConfigOptionDef{
		{
			ID:           "model",
			Name:         "Model",
			Category:     acpport.ConfigCategoryModel,
			CurrentValue: "zai-api/GLM-5.3-Flash",
			Options: []acpport.SelectOptionDef{
				{Value: "zai-api/GLM-5.3-Flash", Name: "GLM-5.3-Flash", Description: "The flagship reasoning model"},
				{Value: "zai-account/GLM-5-turbo", Name: "GLM-5 Turbo"},
			},
		},
		{
			ID:           "thought_level",
			Name:         "Thought level",
			Category:     acpport.ConfigCategoryThoughtLevel,
			CurrentValue: "high",
			Options: []acpport.SelectOptionDef{
				{Value: "high", Name: "High", Description: "deep reasoning"},
				{Value: "low", Name: "Low"},
			},
		},
	}

	h := newScriptedHandlers()
	h.newSessionOptions = newSessionConfigOptions

	resp, err := acpport.NewAgent(h).NewSession(context.Background(), acp.NewSessionRequest{Cwd: testCwd})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if len(resp.ConfigOptions) != 2 {
		t.Fatalf("NewSession ConfigOptions = %d entries, want the front's 2 (model + thought_level): %+v",
			len(resp.ConfigOptions), resp.ConfigOptions)
	}

	// The SDK union's own Validate: exactly one variant set, name present.
	for i := range resp.ConfigOptions {
		if err := resp.ConfigOptions[i].Validate(); err != nil {
			t.Fatalf("ConfigOptions[%d] fails the SDK Validate: %v", i, err)
		}
	}

	frame, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal NewSessionResponse: %v", err)
	}

	if string(frame) != newSessionConfigOptionsFrame {
		t.Fatalf("configOptions assembly drift:\n got: %s\nwant: %s", frame, newSessionConfigOptionsFrame)
	}
}

// TestInitializeAssemblesHonestReport drives the adapter's Initialize and
// byte-locks the response frame: protocolVersion 1, agentInfo
// name/title/version, both authMethods in contract order, zero-value
// capabilities, and NO fs/terminal/loadSession/type surface (D-02 —
// honesty by construction; assert on absent features, A7).
func TestInitializeAssemblesHonestReport(t *testing.T) {
	h := newScriptedHandlers()
	h.initReport = honestReport()

	adapter := acpport.NewAgent(h)

	initReq := acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber}

	resp, err := adapter.Initialize(context.Background(), initReq)
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if resp.ProtocolVersion != acp.ProtocolVersionNumber {
		t.Fatalf("protocolVersion = %d, want %d", resp.ProtocolVersion, acp.ProtocolVersionNumber)
	}

	assertAgentInfo(t, resp.AgentInfo)
	assertAuthMethods(t, resp.AuthMethods)
	assertHonestFrame(t, resp)
}

// assertAgentInfo checks the agentInfo identity block.
func assertAgentInfo(t *testing.T, got *acp.Implementation) {
	t.Helper()

	want := honestReport()

	if got == nil {
		t.Fatal("agentInfo missing — the honest report carries real identity (D-02)")
	}

	if got.Name != want.Name || got.Version != want.Version {
		t.Fatalf("agentInfo = %+v, want name/version from InitReport", got)
	}

	if got.Title == nil || *got.Title != want.Title {
		t.Fatalf("agentInfo title = %v, want %q", got.Title, want.Title)
	}
}

// assertAuthMethods checks both advertised methods in contract order.
func assertAuthMethods(t *testing.T, got []acp.AuthMethod) {
	t.Helper()

	if len(got) != 2 {
		t.Fatalf("authMethods count = %d, want 2 (both advertised per D-02)", len(got))
	}

	if first := got[0].Agent; first == nil || string(first.Id) != "zai-api-key" || first.Name != "z.ai API key" {
		t.Fatalf("authMethods[0] = %+v, want the zai-api-key agent variant", got[0])
	}

	if second := got[1].Agent; second == nil || string(second.Id) != "zai-individual-coding-plan" {
		t.Fatalf("authMethods[1] = %+v, want the zai-individual-coding-plan agent variant", got[1])
	}
}

// assertHonestFrame byte-locks the marshaled response against the
// executed probe shape and the absent-feature guarantees.
func assertHonestFrame(t *testing.T, resp acp.InitializeResponse) {
	t.Helper()

	got, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal initialize response: %v", err)
	}

	if string(got) != honestInitializeFrame {
		t.Fatalf("initialize frame drift:\n got: %s\nwant: %s", got, honestInitializeFrame)
	}

	for _, banned := range []string{"loadSession", "readTextFile", "writeTextFile", "terminal", `"type"`} {
		if strings.Contains(string(got), banned) {
			t.Fatalf("honest initialize frame must not carry %q:\n%s", banned, got)
		}
	}
}

// TestUnadvertisedMethodsAnswerMethodNotFound drives the five unadvertised
// stable methods and asserts each answers the clean -32601 default naming
// the method (Pitfall 6) — and that the Handlers fake was never touched
// (the defaults are adapter-side). Authenticate is NOT in this table: it is
// advertised per D-02 and answers the deliberate honest exception, locked
// by TestAuthenticateDelegation (plan 05-07, superseding plan 04-03's
// TestAuthenticateHonestError). SessionResume/SessionClose
// left the table with plan 05-05: the SESS-01 lifecycle pair became real
// delegates (TestLifecycleDelegation locks the delegation contract).
func TestUnadvertisedMethodsAnswerMethodNotFound(t *testing.T) {
	h := newScriptedHandlers()
	adapter := acpport.NewAgent(h)
	ctx := context.Background()

	tests := []struct {
		method string
		call   func() error
	}{
		{acp.AgentMethodLogout, func() error {
			_, err := adapter.Logout(ctx, acp.LogoutRequest{})

			return err
		}},
		{acp.AgentMethodSessionDelete, func() error {
			_, err := adapter.DeleteSession(ctx, acp.DeleteSessionRequest{})

			return err
		}},
		{acp.AgentMethodSessionList, func() error {
			_, err := adapter.ListSessions(ctx, acp.ListSessionsRequest{})

			return err
		}},
		{acp.AgentMethodSessionSetConfigOption, func() error {
			_, err := adapter.SetSessionConfigOption(ctx, acp.SetSessionConfigOptionRequest{})

			return err
		}},
		{acp.AgentMethodSessionSetMode, func() error {
			_, err := adapter.SetSessionMode(ctx, acp.SetSessionModeRequest{})

			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			assertMethodNotFound(t, tt.call(), tt.method)
		})
	}

	if calls := h.totalCalls(); calls != 0 {
		t.Fatalf("unadvertised methods reached Handlers %d times, want 0 (defaults are adapter-side)", calls)
	}
}

// assertMethodNotFound asserts one coded -32601 error naming the method.
func assertMethodNotFound(t *testing.T, err error, method string) {
	t.Helper()

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("unadvertised method must answer a coded RequestError, got %v", err)
	}

	if rerr.Code != -32601 {
		t.Fatalf("error code = %d, want -32601 (MethodNotFound)", rerr.Code)
	}

	var data map[string]string
	if err := json.Unmarshal(rerr.Data, &data); err != nil {
		t.Fatalf("decode error data: %v (%s)", err, rerr.Data)
	}

	if data["method"] != method {
		t.Fatalf("error data method = %q, want %q", data["method"], method)
	}
}

// TestLifecycleDelegation (SESS-01, plan 05-05) locks the delegation
// contract of the two former unadvertised stubs: the adapter translates
// the wire request onto the neutral Handlers vocabulary, the session id
// rides verbatim, the resume response stays the SDK zero value (the SDK
// resume response carries no session id — the reattached id is the
// request's own), and a front seam error surfaces coded via wireError —
// never raw (A5).
func TestLifecycleDelegation(t *testing.T) {
	t.Run("resume delegates the session id and cwd", func(t *testing.T) {
		h := newScriptedHandlers()
		adapter := acpport.NewAgent(h)

		resp, err := adapter.ResumeSession(context.Background(), acp.ResumeSessionRequest{
			SessionId: acp.SessionId(testSessionID),
			Cwd:       testCwd,
		})
		if err != nil {
			t.Fatalf("ResumeSession: %v", err)
		}

		if len(h.resumeCalls) != 1 || h.resumeCalls[0] != testSessionID {
			t.Fatalf("Handlers.ResumeSession calls = %v, want exactly [%s]", h.resumeCalls, testSessionID)
		}

		// The client cwd rides through verbatim (plan 05-10 WR-02): the
		// front validates it absolute and records it for the reattached id.
		if len(h.resumeCwdCalls) != 1 || h.resumeCwdCalls[0] != testCwd {
			t.Fatalf("Handlers.ResumeSession cwd calls = %v, want exactly [%s]", h.resumeCwdCalls, testCwd)
		}

		marshaled, marshalErr := json.Marshal(resp)
		if marshalErr != nil {
			t.Fatalf("marshal resume response: %v", marshalErr)
		}

		if string(marshaled) != "{}" {
			t.Fatalf("resume response = %s, want the empty object (no id echo on the wire)", marshaled)
		}
	})

	t.Run("close delegates the session id", func(t *testing.T) {
		h := newScriptedHandlers()
		adapter := acpport.NewAgent(h)

		if _, err := adapter.CloseSession(context.Background(), acp.CloseSessionRequest{
			SessionId: acp.SessionId(testSessionID),
		}); err != nil {
			t.Fatalf("CloseSession: %v", err)
		}

		if len(h.closeCalls) != 1 || h.closeCalls[0] != testSessionID {
			t.Fatalf("Handlers.CloseSession calls = %v, want exactly [%s]", h.closeCalls, testSessionID)
		}
	})

	t.Run("resume seam error surfaces coded", func(t *testing.T) {
		h := newScriptedHandlers()
		h.mu.Lock()
		h.resumeErr = acpport.NewInternal("backend refused the reattach")
		h.mu.Unlock()

		_, err := acpport.NewAgent(h).ResumeSession(context.Background(), acp.ResumeSessionRequest{
			SessionId: acp.SessionId(testSessionID),
		})
		assertCodedInternal(t, err, "backend refused the reattach")
	})
}

// TestLifecycleDelegationCoded locks the error leg of the SESS-01
// delegation: a Handlers-seam error (and an invalid-request one) surfaces
// through wireError with its own code and message — never raw (A5), never
// the -32601 stub default the methods answered before plan 05-05.
func TestLifecycleDelegationCoded(t *testing.T) {
	t.Run("close seam error surfaces coded", func(t *testing.T) {
		h := newScriptedHandlers()
		h.mu.Lock()
		h.closeErr = acpport.NewInvalidRequest("no session registered under id")
		h.mu.Unlock()

		_, err := acpport.NewAgent(h).CloseSession(context.Background(), acp.CloseSessionRequest{
			SessionId: acp.SessionId(testSessionID),
		})

		var rerr *acp.RequestError
		if !errors.As(err, &rerr) {
			t.Fatalf("close error = %T (%v), want a coded RequestError", err, err)
		}

		if rerr.Code != acpport.CodeInvalidRequest {
			t.Fatalf("close error code = %d, want %d", rerr.Code, acpport.CodeInvalidRequest)
		}

		if rerr.Message != "no session registered under id" {
			t.Fatalf("close error message = %q, want the seam's own text", rerr.Message)
		}
	})
}

// assertCodedInternal asserts one coded -32603 RequestError whose message
// carries the front's own text (wireError keeps seam messages verbatim,
// never the wrapped transport noise).
func assertCodedInternal(t *testing.T, err error, message string) {
	t.Helper()

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("error = %T (%v), want a coded RequestError", err, err)
	}

	if rerr.Code != acpport.CodeInternal {
		t.Fatalf("error code = %d, want %d (internal class)", rerr.Code, acpport.CodeInternal)
	}

	if rerr.Message != message {
		t.Fatalf("error message = %q, want %q", rerr.Message, message)
	}
}

// TestLifecycleDelegationWire drives the SESS-01 pair through a REAL
// agent-side connection with raw JSON-RPC frames: session/resume and
// session/close answer RESULT frames on the actual wire (no -32601
// remains for these two methods — the plan 04-03 unadvertised lock
// shrank to five), and the Handlers impl received the wire session ids.
func TestLifecycleDelegationWire(t *testing.T) {
	h := newScriptedHandlers()

	probe := newConnectionProbe(t, h)

	tests := []struct {
		method string
		params string
	}{
		{acp.AgentMethodSessionResume, `{"sessionId":"` + testSessionID + `","cwd":"` + testCwd + `"}`},
		{acp.AgentMethodSessionClose, `{"sessionId":"` + testSessionID + `"}`},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			raw := probe(t, tt.method, tt.params)

			var frame struct {
				Error json.RawMessage `json:"error"`
			}

			if err := json.Unmarshal([]byte(raw), &frame); err != nil {
				t.Fatalf("decode %s response frame: %v (%s)", tt.method, err, raw)
			}

			if len(frame.Error) != 0 && string(frame.Error) != "null" {
				t.Fatalf("%s answered an error frame, want a result: %s", tt.method, raw)
			}
		})
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.resumeCalls) != 1 || h.resumeCalls[0] != testSessionID {
		t.Fatalf("Handlers.ResumeSession calls = %v, want exactly [%s]", h.resumeCalls, testSessionID)
	}

	// The wire cwd rode through verbatim (plan 05-10 WR-02).
	if len(h.resumeCwdCalls) != 1 || h.resumeCwdCalls[0] != testCwd {
		t.Fatalf("Handlers.ResumeSession cwd calls = %v, want exactly [%s]", h.resumeCwdCalls, testCwd)
	}

	if len(h.closeCalls) != 1 || h.closeCalls[0] != testSessionID {
		t.Fatalf("Handlers.CloseSession calls = %v, want exactly [%s]", h.closeCalls, testSessionID)
	}
}

// connProbe is one raw JSON-RPC exchange through a live agent-side
// connection: write the request frame, read back the raw response frame —
// the SDK dispatcher's actual wire answer, not just the adapter method.
type connProbe func(t *testing.T, method, params string) string

// newConnectionProbe wires acpport.NewAgentSideConnection over io pipes
// (construction completes before any frame flows — the Phase-3 discipline)
// and returns the raw-frame exchange probe. Everything tears down via
// t.Cleanup; response frames arrive on the drained channel.
func newConnectionProbe(t *testing.T, h *scriptedHandlers) connProbe {
	t.Helper()

	toAgentR, toAgentW := io.Pipe()
	fromAgentR, fromAgentW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{toAgentR, toAgentW, fromAgentR, fromAgentW} {
			_ = c.Close()
		}
	})

	acpport.NewAgentSideConnection(acpport.NewAgent(h), fromAgentW, toAgentR)

	lines := make(chan string, 16)
	go drainFrames(fromAgentR, lines)

	return func(t *testing.T, method, params string) string {
		t.Helper()

		frame := fmt.Sprintf(`{"jsonrpc":"2.0","id":7,"method":%q,"params":%s}`+"\n", method, params)
		if _, err := toAgentW.Write([]byte(frame)); err != nil {
			t.Fatalf("write %s request frame: %v", method, err)
		}

		return readFrame(t, lines)
	}
}

// jsonRPCErrorFrame decodes the error object of one raw response frame;
// nil when the frame is a result (never expected in these probes).
func jsonRPCErrorFrame(t *testing.T, raw, method string) *struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Method string `json:"method"`
	} `json:"data"`
} {
	t.Helper()

	var frame struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				Method string `json:"method"`
			} `json:"data"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode %s response frame: %v (%s)", method, err, raw)
	}

	return frame.Error
}

// TestUnadvertisedStableMethodsAnswerNotFound (plan 04-03 lock) drives the
// five unadvertised stable methods through a REAL agent-side connection
// with raw JSON-RPC frames: each must answer the clean -32601 default
// naming the method on the actual wire (Pitfall 6 — capability omission
// stops legitimate clients, but the dispatcher still routes), and the
// Handlers impl must never be touched (the defaults are adapter-side).
// Authenticate is NOT in this table: it is advertised per D-02 and answers
// the authenticate delegation (TestAuthenticateDelegation, plan 05-07).
// SessionResume/SessionClose left the table with plan 05-05 (the SESS-01
// delegates; TestLifecycleDelegationWire locks the wire shape).
func TestUnadvertisedStableMethodsAnswerNotFound(t *testing.T) {
	h := newScriptedHandlers()

	probe := newConnectionProbe(t, h)

	// Minimal params per method — session/set_config_option is a union whose
	// default (value-id) variant needs its keys; everything else decodes the
	// empty object.
	tests := []struct {
		method string
		params string
	}{
		{acp.AgentMethodLogout, "{}"},
		{acp.AgentMethodSessionDelete, "{}"},
		{acp.AgentMethodSessionList, "{}"},
		{acp.AgentMethodSessionSetConfigOption, `{"configId":"c","sessionId":"s","value":"v"}`},
		{acp.AgentMethodSessionSetMode, "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			errObj := jsonRPCErrorFrame(t, probe(t, tt.method, tt.params), tt.method)

			if errObj == nil {
				t.Fatalf("%s must answer a JSON-RPC error, got a result frame", tt.method)
			}

			if errObj.Code != -32601 {
				t.Fatalf("%s error code = %d, want -32601 (MethodNotFound)", tt.method, errObj.Code)
			}

			if errObj.Data.Method != tt.method {
				t.Fatalf("%s error data method = %q, want %q", tt.method, errObj.Data.Method, tt.method)
			}
		})
	}

	if calls := h.totalCalls(); calls != 0 {
		t.Fatalf("unadvertised methods reached Handlers %d times, want 0 (defaults are adapter-side)", calls)
	}
}

// TestAuthenticateDelegation (plan 05-07, superseding plan 04-03's
// TestAuthenticateHonestError lock): authenticate is ADVERTISED per D-02
// (both z.ai auth methods ride the initialize report), and the adapter now
// DELEGATES through the neutral Handlers surface — the front's honest chain
// (AUTH-01: record key → push snapshot → refresh registry → read-back) is
// the only success path. The lock keeps plan 04-03's honesty floor and adds
// the delegation contract: every Handlers error codes through wireError on
// the actual wire (never raw, never the misleading -32601 default, never a
// fake success), a seam-carried code rides verbatim, and success answers
// the empty result {}.
func TestAuthenticateDelegation(t *testing.T) {
	t.Run("handlers error codes as the internal-class wire error", testAuthDelegationPlainError)
	t.Run("seam-carried code rides the wire verbatim", testAuthDelegationSeamCode)
	t.Run("success answers the empty result", testAuthDelegationSuccess)
}

// testAuthDelegationPlainError is the A5 floor arm: a plain Handlers error
// codes internal on the wire — detail in data, never raw, never -32601,
// never a fake success — and the client's methodId reached Handlers
// verbatim.
func testAuthDelegationPlainError(t *testing.T) {
	h := newScriptedHandlers()
	h.authErr = errAuthScriptedFailure

	probe := newConnectionProbe(t, h)

	raw := probe(t, acp.AgentMethodAuthenticate, `{"methodId":"`+authDelegationMethodID+`"}`)

	errObj := authErrorFrame(t, raw)

	if errObj == nil {
		t.Fatalf("authenticate must answer a JSON-RPC ERROR (never fake success), got result frame: %s", raw)
	}

	if errObj.Code == -32601 {
		t.Fatalf("authenticate answers the misleading MethodNotFound default (-32601): %s", raw)
	}

	if errObj.Code != acpport.CodeInternal {
		t.Fatalf("authenticate error code = %d, want %d (a plain error codes internal): %s",
			errObj.Code, acpport.CodeInternal, raw)
	}

	if !strings.Contains(string(errObj.Data), "scripted auth failure") {
		t.Fatalf("error data %s must carry the Handlers error text (the A5 internal mapping): %s",
			string(errObj.Data), raw)
	}

	if len(h.authCalls) != 1 || h.authCalls[0] != authDelegationMethodID {
		t.Fatalf("authCalls = %v, want exactly the client's methodId verbatim", h.authCalls)
	}
}

// testAuthDelegationSeamCode is the seam arm: a seam-carried code and
// message ride the wire verbatim (the RequestError stays unwrapped).
func testAuthDelegationSeamCode(t *testing.T) {
	h := newScriptedHandlers()
	h.authErr = acpport.NewInvalidParams("authenticate: unknown methodId %q", "zai-nope")

	probe := newConnectionProbe(t, h)

	raw := probe(t, acp.AgentMethodAuthenticate, `{"methodId":"zai-nope"}`)

	errObj := authErrorFrame(t, raw)

	if errObj == nil {
		t.Fatalf("seam error must answer a JSON-RPC ERROR, got result frame: %s", raw)
	}

	if errObj.Code != acpport.CodeInvalidParams {
		t.Fatalf("error code = %d, want the seam's %d riding verbatim: %s",
			errObj.Code, acpport.CodeInvalidParams, raw)
	}

	if !strings.Contains(errObj.Message, "zai-nope") {
		t.Fatalf("error message %q must carry the seam text verbatim (the RequestError stays unwrapped): %s",
			errObj.Message, raw)
	}
}

// testAuthDelegationSuccess is the success arm: the empty result {}.
func testAuthDelegationSuccess(t *testing.T) {
	h := newScriptedHandlers()

	probe := newConnectionProbe(t, h)

	raw := probe(t, acp.AgentMethodAuthenticate, `{"methodId":"`+authDelegationMethodID+`"}`)

	errObj := authErrorFrame(t, raw)
	if errObj != nil {
		t.Fatalf("authenticated authenticate must answer a result, got error frame: %s", raw)
	}

	var frame struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode result frame: %v (%s)", err, raw)
	}

	if trimmed := strings.TrimSpace(string(frame.Result)); trimmed != "{}" {
		t.Fatalf("authenticate result = %s, want the empty result {}", trimmed)
	}
}

// authErrorFrame decodes the error object of one raw authenticate response
// with a raw Data payload — the SDK's internal mapping carries the Handlers
// detail as a JSON string in data (never the strict {method} shape the
// unadvertised defaults carry).
func authErrorFrame(t *testing.T, raw string) *struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
} {
	t.Helper()

	var frame struct {
		Error *struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		} `json:"error"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode authenticate response frame: %v (%s)", err, raw)
	}

	return frame.Error
}

// promptJoinProbe is the prompt request the content-join assertions ride:
// two text blocks around one resource_link (Assumption A3 — the link
// contributes its URI to the joined content).
func promptJoinProbe() acp.PromptRequest {
	return acp.PromptRequest{
		SessionId: testSessionID,
		Prompt: []acp.ContentBlock{
			acp.TextBlock("Reply"),
			{ResourceLink: &acp.ContentBlockResourceLink{Uri: "file:///x", Name: "x"}},
			acp.TextBlock("OK"),
		},
	}
}

// TestPromptMapsStopReasonsAndCodesErrors drives the adapter's Prompt
// over the scripted fake: StopEndTurn/StopCancelled map onto the SDK stop
// reasons; front errors surface as coded RequestErrors (-32600 seam
// invalid-request, -32603 internal — never raw, never synthetic stop
// reasons, A5); an unmapped stop reason must not leak.
func TestPromptMapsStopReasonsAndCodesErrors(t *testing.T) {
	req := promptJoinProbe()

	t.Run("stop-reason mapping", func(t *testing.T) {
		assertPromptStopReasons(t, req)
	})

	t.Run("content join and delegation", func(t *testing.T) {
		assertPromptContentJoin(t, req)
	})

	t.Run("front errors are coded", func(t *testing.T) {
		assertPromptCodedErrors(t, req)
	})

	t.Run("unmapped stop reason must not leak", func(t *testing.T) {
		assertUnmappedReasonLoud(t, req)
	})
}

// assertPromptStopReasons checks both neutral stop reasons map onto the
// SDK's wire stop reasons.
func assertPromptStopReasons(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	for reason, want := range map[acpport.StopReason]acp.StopReason{
		acpport.StopEndTurn:   acp.StopReasonEndTurn,
		acpport.StopCancelled: acp.StopReasonCancelled,
	} {
		h := newScriptedHandlers()
		h.promptReason = reason

		resp, err := acpport.NewAgent(h).Prompt(context.Background(), req)
		if err != nil {
			t.Fatalf("Prompt(%s): %v", reason, err)
		}

		if resp.StopReason != want {
			t.Fatalf("Prompt(%s) stopReason = %q, want %q", reason, resp.StopReason, want)
		}
	}
}

// assertPromptContentJoin checks the delegation contract: one Handlers
// call with the joined content (text verbatim, resource_link as URI) and
// a non-nil sink.
func assertPromptContentJoin(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	h := newScriptedHandlers()
	h.promptReason = acpport.StopEndTurn

	if _, err := acpport.NewAgent(h).Prompt(context.Background(), req); err != nil {
		t.Fatalf("Prompt: %v", err)
	}

	if len(h.promptCalls) != 1 {
		t.Fatalf("Handlers.Prompt calls = %d, want 1", len(h.promptCalls))
	}

	got := h.promptCalls[0]

	if got.SessionID != testSessionID {
		t.Fatalf("PromptParams.SessionID = %q, want %s", got.SessionID, testSessionID)
	}

	if got.Text != "Reply file:///x OK" {
		t.Fatalf("PromptParams.Text = %q, want joined content %q", got.Text, "Reply file:///x OK")
	}

	if h.sinks[0] == nil {
		t.Fatal("Handlers.Prompt must receive a non-nil update sink")
	}
}

// assertPromptCodedErrors checks the error-coding table: seam
// invalid-request codes -32600, seam and raw internal errors code -32603.
func assertPromptCodedErrors(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	tests := []struct {
		name     string
		frontErr error
		wantCode int
	}{
		{"invalid request", acpport.NewInvalidRequest("a prompt is already running for this session"), -32600},
		{"internal", acpport.NewInternal("backend session send failed"), -32603},
		{"raw error is coded internal", errRawFront, -32603},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newScriptedHandlers()
			h.promptErr = tt.frontErr

			resp, err := acpport.NewAgent(h).Prompt(context.Background(), req)
			if err == nil {
				t.Fatalf("Prompt must fail, got response %+v", resp)
			}

			var rerr *acp.RequestError
			if !errors.As(err, &rerr) {
				t.Fatalf("front error must surface as a coded RequestError, got %T: %v", err, err)
			}

			if rerr.Code != tt.wantCode {
				t.Fatalf("error code = %d, want %d", rerr.Code, tt.wantCode)
			}
		})
	}
}

// assertUnmappedReasonLoud checks a foreign stop reason never becomes a
// result — it surfaces as a coded internal error (A5).
func assertUnmappedReasonLoud(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	h := newScriptedHandlers()
	h.promptReason = acpport.StopReason("refusal")

	resp, err := acpport.NewAgent(h).Prompt(context.Background(), req)
	if err == nil {
		t.Fatalf("unmapped stop reason %q must not become a response, got %+v", h.promptReason, resp)
	}

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) || rerr.Code != -32603 {
		t.Fatalf("unmapped stop reason must surface as -32603, got %v", err)
	}
}

// TestFrontErrorsCodedOnAllAdvertisedMethods pins the coding contract on
// every advertised method: a seam error returned by Handlers surfaces as
// the coded RequestError on the wire, never raw — the SDK's toReqErr
// type-asserts *RequestError, so an untranslated seam error would degrade
// to a generic -32603 with the message string.
func TestFrontErrorsCodedOnAllAdvertisedMethods(t *testing.T) {
	ctx := context.Background()

	seamErr := acpport.NewInvalidRequest("session is not available for this operation")

	t.Run("initialize", func(t *testing.T) {
		h := newScriptedHandlers()
		h.initErr = seamErr

		_, err := acpport.NewAgent(h).Initialize(ctx, acp.InitializeRequest{})
		assertSeamCoded(t, err, seamErr)
	})

	t.Run("new_session", func(t *testing.T) {
		h := newScriptedHandlers()
		h.newSessionErr = seamErr

		_, err := acpport.NewAgent(h).NewSession(ctx, acp.NewSessionRequest{Cwd: testCwd})
		assertSeamCoded(t, err, seamErr)
	})

	t.Run("cancel", func(t *testing.T) {
		h := newScriptedHandlers()
		h.cancelErr = seamErr

		err := acpport.NewAgent(h).Cancel(ctx, acp.CancelNotification{SessionId: testSessionID})
		assertSeamCoded(t, err, seamErr)
	})
}

// assertSeamCoded asserts err is the coded RequestError carrying the seam
// error's code and message.
func assertSeamCoded(t *testing.T, err error, seam *acpport.Error) {
	t.Helper()

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("seam error must surface as a coded RequestError, got %T: %v", err, err)
	}

	if rerr.Code != seam.Code || rerr.Message != seam.Message {
		t.Fatalf("coded error = %d/%q, want %d/%q", rerr.Code, rerr.Message, seam.Code, seam.Message)
	}
}

// TestNewSessionAndCancelDelegate drives the two remaining advertised
// methods over the scripted fake: NewSession returns the front's session
// id and passes cwd + the neutral MCP view through; Cancel forwards the
// session id (D-03/D-05 delegation contracts).
func TestNewSessionAndCancelDelegate(t *testing.T) {
	h := newScriptedHandlers()
	adapter := acpport.NewAgent(h)
	ctx := context.Background()

	req := acp.NewSessionRequest{
		Cwd: testCwd,
		McpServers: []acp.McpServer{
			{Stdio: &acp.McpServerStdio{Name: "mcp-fs", Command: "uvx"}},
		},
	}

	resp, err := adapter.NewSession(ctx, req)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	if resp.SessionId != acp.SessionId(h.newSessionID) {
		t.Fatalf("NewSession sessionId = %q, want %q (front id passes through)", resp.SessionId, h.newSessionID)
	}

	if len(h.newSessionCalls) != 1 {
		t.Fatalf("Handlers.NewSession calls = %d, want 1", len(h.newSessionCalls))
	}

	assertNeutralNewSessionParams(t, h.newSessionCalls[0])

	cancelNote := acp.CancelNotification{SessionId: backendSessionID}

	if err := adapter.Cancel(ctx, cancelNote); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	if len(h.cancelCalls) != 1 || h.cancelCalls[0] != backendSessionID {
		t.Fatalf("Handlers.Cancel calls = %v, want exactly [%s]", h.cancelCalls, backendSessionID)
	}
}

// assertNeutralNewSessionParams checks the neutral DTO translation of the
// create request (cwd verbatim, MCP list shallow-copied out of SDK types).
func assertNeutralNewSessionParams(t *testing.T, params acpport.NewSessionParams) {
	t.Helper()

	if params.Cwd != testCwd {
		t.Fatalf("NewSessionParams.Cwd = %q, want %s", params.Cwd, testCwd)
	}

	if len(params.McpServers) != 1 || params.McpServers[0].Kind != "stdio" || params.McpServers[0].Name != "mcp-fs" {
		t.Fatalf("NewSessionParams.McpServers = %+v, want the neutral stdio view of mcp-fs", params.McpServers)
	}
}

// TestUpdateSinkEmitsChunkShapes wires a real agent-side connection and
// asserts the sink's wire shapes byte-for-byte: message notes become
// agent_message_chunk notifications, thought notes become
// agent_thought_chunk notifications, and an unknown note kind fails loud
// without emitting anything (D-04 honesty).
func TestUpdateSinkEmitsChunkShapes(t *testing.T) {
	clientToAgentR, clientToAgentW := io.Pipe()
	agentToClientR, agentToClientW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{clientToAgentR, clientToAgentW, agentToClientR, agentToClientW} {
			_ = c.Close()
		}
	})

	h := newScriptedHandlers()
	conn, _, _ := acpport.NewAgentSideConnection(acpport.NewAgent(h), agentToClientW, clientToAgentR)
	sink := acpport.NewUpdateSink(conn)

	lines := make(chan string, 16)

	go drainFrames(agentToClientR, lines)

	ctx := context.Background()

	messageNote := acpport.SessionNote{SessionID: testSessionID, Kind: acpport.NoteMessage, Delta: "hello"}

	if err := sink(ctx, messageNote); err != nil {
		t.Fatalf("sink(message): %v", err)
	}

	thoughtNote := acpport.SessionNote{SessionID: testSessionID, Kind: acpport.NoteThought, Delta: "thinking"}

	if err := sink(ctx, thoughtNote); err != nil {
		t.Fatalf("sink(thought): %v", err)
	}

	// The full wire frame: the SDK wraps the notification in the JSON-RPC
	// envelope — the executed-probe shapes (RESEARCH Code Examples) are
	// the params payload inside it.
	wantMessage := `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` + testSessionID +
		`","update":{"content":{"text":"hello","type":"text"},"sessionUpdate":"agent_message_chunk"}}}`
	wantThought := `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` + testSessionID +
		`","update":{"content":{"text":"thinking","type":"text"},"sessionUpdate":"agent_thought_chunk"}}}`

	if got := readFrame(t, lines); got != wantMessage {
		t.Fatalf("message chunk frame drift:\n got: %s\nwant: %s", got, wantMessage)
	}

	if got := readFrame(t, lines); got != wantThought {
		t.Fatalf("thought chunk frame drift:\n got: %s\nwant: %s", got, wantThought)
	}

	err := sink(ctx, acpport.SessionNote{SessionID: testSessionID, Kind: "bogus", Delta: "?"})
	if err == nil {
		t.Fatal("unknown note kind must fail loud, got nil error")
	}

	expectNoFrame(t, lines, "unknown note kind must not emit a frame")
}

// Tool-note wire fixture ids and paths (the goconst contract — reused
// across the table rows).
const (
	toolNoteCallID    = "tc1"
	toolNoteCallID2   = "tc2"
	toolNoteDiffPath  = "/w/main.go"
	toolNoteDiffPath2 = "/w/new.go"
)

// toolNoteCase is one wire-shape table row: a note and its byte-exact
// expected frame (the SDK wraps each in the JSON-RPC notification
// envelope — the existing chunk-shape test's contract).
type toolNoteCase struct {
	name string
	note acpport.SessionNote
	want string
}

// toolNoteFrame wraps one session/update payload in the JSON-RPC
// notification envelope for the fixture session — the rows compose their
// expected update payloads through it.
func toolNoteFrame(update string) string {
	return `{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"` +
		testSessionID + `","update":{` + update + `}}}`
}

// toolNoteCases is the wire-shape table: one note per row, the byte-exact
// expected frame.
func toolNoteCases() []toolNoteCase {
	return append(editToolNoteCases(), writeToolNoteCase())
}

// editToolNoteCases covers the tc1 lifecycle: the start card with diff,
// locations, and verbatim raw input; the patch-semantics update; the
// result re-emitting the diff with rawOutput.
func editToolNoteCases() []toolNoteCase {
	return []toolNoteCase{
		{
			name: "start note renders tool_call with diff, locations, raw input",
			note: acpport.SessionNote{
				SessionID: testSessionID,
				Kind:      acpport.NoteTool,
				Tool: &acpport.ToolNote{
					ToolCallID: toolNoteCallID,
					IsStart:    true,
					Title:      "Edit main.go",
					Name:       "Edit",
					Kind:       acpport.ToolKindEdit,
					Status:     acpport.ToolStatusPending,
					RawInput: json.RawMessage(
						`{"file_path":"` + toolNoteDiffPath + `","old_string":"a","new_string":"b"}`),
					Diff:      &acpport.ToolDiff{Path: toolNoteDiffPath, OldText: "a", NewText: "b"},
					Locations: []acpport.ToolLocation{{Path: toolNoteDiffPath}},
				},
			},
			want: toolNoteFrame(
				`"content":[{"newText":"b","oldText":"a","path":"` + toolNoteDiffPath + `","type":"diff"}],` +
					`"kind":"edit","locations":[{"path":"` + toolNoteDiffPath + `"}],` +
					`"name":"Edit","rawInput":{"file_path":"` + toolNoteDiffPath + `",` +
					`"old_string":"a","new_string":"b"},"sessionUpdate":"tool_call",` +
					`"status":"pending","title":"Edit main.go","toolCallId":"` + toolNoteCallID + `"`),
		},
		{
			name: "update note carries only the changed field (patch semantics)",
			note: acpport.SessionNote{
				SessionID: testSessionID,
				Kind:      acpport.NoteTool,
				Tool: &acpport.ToolNote{
					ToolCallID: toolNoteCallID,
					Status:     acpport.ToolStatusInProgress,
				},
			},
			want: toolNoteFrame(
				`"sessionUpdate":"tool_call_update","status":"in_progress",` +
					`"toolCallId":"` + toolNoteCallID + `"`),
		},
		{
			name: "result note re-emits the diff and completes with raw output",
			note: acpport.SessionNote{
				SessionID: testSessionID,
				Kind:      acpport.NoteTool,
				Tool: &acpport.ToolNote{
					ToolCallID: toolNoteCallID,
					Status:     acpport.ToolStatusCompleted,
					RawOutput:  json.RawMessage(`{"ok":true}`),
					Diff:       &acpport.ToolDiff{Path: toolNoteDiffPath, OldText: "a", NewText: "b"},
				},
			},
			want: toolNoteFrame(
				`"content":[{"newText":"b","oldText":"a","path":"` + toolNoteDiffPath + `","type":"diff"}],` +
					`"rawOutput":{"ok":true},"sessionUpdate":"tool_call_update",` +
					`"status":"completed","toolCallId":"` + toolNoteCallID + `"`),
		},
	}
}

// writeToolNoteCase covers the tc2 new-file shape: oldText absent on the
// wire (empty OldText renders no old text).
func writeToolNoteCase() toolNoteCase {
	return toolNoteCase{
		name: "new-file write shape omits oldText on the wire",
		note: acpport.SessionNote{
			SessionID: testSessionID,
			Kind:      acpport.NoteTool,
			Tool: &acpport.ToolNote{
				ToolCallID: toolNoteCallID2,
				IsStart:    true,
				Title:      "Write new.go",
				Name:       "Write",
				Kind:       acpport.ToolKindEdit,
				Status:     acpport.ToolStatusPending,
				RawInput:   json.RawMessage(`{"content":"file body","file_path":"` + toolNoteDiffPath2 + `"}`),
				Diff:       &acpport.ToolDiff{Path: toolNoteDiffPath2, NewText: "file body"},
			},
		},
		want: toolNoteFrame(
			`"content":[{"newText":"file body","path":"` + toolNoteDiffPath2 + `","type":"diff"}],` +
				`"kind":"edit","name":"Write","rawInput":{"content":"file body",` +
				`"file_path":"` + toolNoteDiffPath2 + `"},"sessionUpdate":"tool_call",` +
				`"status":"pending","title":"Write new.go","toolCallId":"` + toolNoteCallID2 + `"`),
	}
}

// syncBuffer is a mutex-guarded bytes.Buffer — the write side serves the
// default slog handler (including other tests' background goroutines),
// the read side is the assertion snapshot.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends p under the mutex (io.Writer).
func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buf.Write(p)
}

// String snapshots the buffer under the mutex.
func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.buf.String()
}

// TestToolNote (plan 05-04 Task 1 RED target) locks the TOOL-01 sink
// translation on recorded session/update wire frames: the neutral ToolNote
// vocabulary maps onto the SDK's tool vocabulary byte-exactly —
//
//	start note with diff, locations, and raw input → the tool_call frame
//	  carrying toolCallId+title+name+kind+status pending, the diff content
//	  (type diff with path/newText/oldText exactly as the SDK helper sets
//	  them), and the location path — with rawInput carried VERBATIM (the
//	  D-04 honesty carry-over: the sink transports, never augments);
//	update note with a status → the tool_call_update frame with ONLY the
//	  changed field (patch semantics — no empty patch objects on the wire);
//	result note (status completed + rawOutput + diff re-emitted) → the
//	  patch carrying content, rawOutput, and the completed status;
//	new-file Write shape (empty OldText) → oldText ABSENT on the wire.
//
// The empty-kind and payload-less arms keep the loud coded error — an
// unknown or unusable note never emits a frame, never silently drops.
func TestToolNote(t *testing.T) {
	clientToAgentR, clientToAgentW := io.Pipe()
	agentToClientR, agentToClientW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{clientToAgentR, clientToAgentW, agentToClientR, agentToClientW} {
			_ = c.Close()
		}
	})

	h := newScriptedHandlers()
	conn, _, _ := acpport.NewAgentSideConnection(acpport.NewAgent(h), agentToClientW, clientToAgentR)
	sink := acpport.NewUpdateSink(conn)

	lines := make(chan string, 16)

	go drainFrames(agentToClientR, lines)

	ctx := context.Background()

	for _, tc := range toolNoteCases() {
		if err := sink(ctx, tc.note); err != nil {
			t.Fatalf("%s: sink: %v", tc.name, err)
		}

		if got := readFrame(t, lines); got != tc.want {
			t.Fatalf("%s: wire frame drift:\n got: %s\nwant: %s", tc.name, got, tc.want)
		}
	}

	// The loud defaults: an empty-kind note and a tool note without its
	// payload fail loud, never emit, never silently drop.
	for _, note := range []acpport.SessionNote{
		{SessionID: testSessionID, Kind: "", Delta: "?"},
		{SessionID: testSessionID, Kind: acpport.NoteTool},
	} {
		if err := sink(ctx, note); err == nil {
			t.Fatalf("note kind %q without usable payload must fail loud, got nil error", note.Kind)
		}
	}

	expectNoFrame(t, lines, "loud-default notes must not emit frames")
}

// TestUnsupportedPromptBlocksWarnStructural locks WR-06 (plan 05-04
// Task 2) at the drop site: image/audio/embedded-resource prompt blocks
// are held from the joined turn text (the text baseline intact) and the
// drop emits ONE structural slog warn carrying kinds and counts — never
// the block content (the base64 payload must not appear anywhere in the
// log record).
func TestUnsupportedPromptBlocksWarnStructural(t *testing.T) {
	// The log buffer is mutex-guarded: the swapped-in default handler ALSO
	// receives the SDK's background connection logs from other tests'
	// goroutines, and the test's own reads must not race those writes.
	var logs syncBuffer

	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

	h := newScriptedHandlers()
	h.promptReason = acpport.StopEndTurn

	const secretPixel = "c2VjcmV0LXBpeGVsLXBheWxvYWQ=" // scrubbed fixture bytes

	adapter := acpport.NewAgent(h)

	_, err := adapter.Prompt(context.Background(), acp.PromptRequest{
		SessionId: acp.SessionId(testSessionID),
		Prompt: []acp.ContentBlock{
			acp.TextBlock("before"),
			acp.ImageBlock(secretPixel, "image/png"),
			acp.AudioBlock(secretPixel, "audio/wav"),
			acp.TextBlock("after"),
		},
	})
	if err != nil {
		t.Fatalf("prompt with unsupported blocks: %v", err)
	}

	// The text baseline joined intact around the held blocks.
	prompted := h.promptCalls
	if len(prompted) != 1 || prompted[0].Text != "before after" {
		t.Fatalf("joined prompt text = %q, want %q (held blocks contribute nothing)", prompted[0].Text, "before after")
	}

	recorded := logs.String()

	if !strings.Contains(recorded, "unsupported prompt content blocks held") {
		t.Fatalf("no drop-site warn recorded:\n%s", recorded)
	}

	if !strings.Contains(recorded, "image=1") || !strings.Contains(recorded, "audio=1") {
		t.Fatalf("drop-site warn must carry kinds and counts (image=1, audio=1):\n%s", recorded)
	}

	if strings.Contains(recorded, secretPixel) {
		t.Fatal("drop-site warn leaked block content — the warn carries kinds and counts ONLY")
	}
}

// TestPermissionSender (plan 05-02 Task 1 RED target) drives the bound
// PermissionSender over a real agent-side connection and asserts the wire
// round-trip on the recorded frame: the outgoing session/request_permission
// carries the ask's options verbatim (kind/name/optionId — the sender never
// rewrites them; the mapping happens ask-side), the toolCall patch carries
// the backend toolCallId, and the client's outcome union decodes both arms
// into the neutral vocabulary — {outcome:{outcome:selected, optionId}}
// becomes Selected{OptionID} and {outcome:{outcome:cancelled}} becomes
// Cancelled (TOOL-02; Don't-Hand-Roll: the SDK owns framing and pending,
// the sender only translates).
func TestPermissionSender(t *testing.T) {
	t.Run("selected outcome decodes neutral", func(t *testing.T) {
		outcome := drivePermissionOutcome(t, `{"outcome":{"outcome":"selected","optionId":"allow_once"}}`)
		assertPermissionOutcome(t, outcome, acpport.PermissionOutcome{OptionID: "allow_once"})
	})

	t.Run("cancelled outcome decodes neutral", func(t *testing.T) {
		outcome := drivePermissionOutcome(t, `{"outcome":{"outcome":"cancelled"}}`)
		assertPermissionOutcome(t, outcome, acpport.PermissionOutcome{Cancelled: true})
	})
}

// permissionAsk is the canonical scripted ask the sender drives: two
// options with the mapped ACP kinds (allow_once stays allow_once; deny
// arrives as reject_once after the ask-side mapping), names and optionIds
// verbatim from the backend.
func permissionAsk() acpport.PermissionAsk {
	return acpport.PermissionAsk{
		SessionID:  "sess_p1",
		RequestID:  "perm-1",
		ToolCallID: "tc1",
		ToolName:   "Bash",
		Options: []acpport.PermissionOptionDef{
			{Kind: acpport.PermKindAllowOnce, Name: "Allow once", OptionID: "allow_once"},
			{Kind: acpport.PermKindRejectOnce, Name: "Deny", OptionID: "deny"},
		},
	}
}

// drivePermissionOutcome composes the connection, drives the bound sender
// with the canonical ask, plays the client for the request_permission
// frame (asserting its wire shape), answers with the rawClientOutcome
// result payload, and returns the decoded neutral outcome.
func drivePermissionOutcome(t *testing.T, rawClientOutcome string) acpport.PermissionOutcome {
	t.Helper()

	toAgentR, toAgentW := io.Pipe()
	fromAgentR, fromAgentW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{toAgentR, toAgentW, fromAgentR, fromAgentW} {
			_ = c.Close()
		}
	})

	_, _, sender := acpport.NewAgentSideConnection(acpport.NewAgent(newScriptedHandlers()), fromAgentW, toAgentR)

	lines := make(chan string, 16)

	go drainFrames(fromAgentR, lines)

	type result struct {
		outcome acpport.PermissionOutcome
		err     error
	}

	done := make(chan result, 1)

	go func() {
		outcome, err := sender(context.Background(), permissionAsk())
		done <- result{outcome: outcome, err: err}
	}()

	raw := readFrame(t, lines)
	assertRequestPermissionFrame(t, raw)

	// The client's answer: echo the request's id verbatim so the SDK's
	// pending correlation resolves (the id type is SDK-internal — replay
	// the decoded raw token).
	var frame struct {
		ID json.RawMessage `json:"id"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode request_permission frame id: %v (%s)", err, raw)
	}

	resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`+"\n", frame.ID, rawClientOutcome)

	if _, err := toAgentW.Write([]byte(resp)); err != nil {
		t.Fatalf("write request_permission response: %v", err)
	}

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("sender round-trip: %v", r.err)
		}

		return r.outcome
	case <-time.After(frameWaitTimeout):
		t.Fatal("sender round-trip never resolved after the client answered")
	}

	return acpport.PermissionOutcome{} // unreachable
}

// assertRequestPermissionFrame byte-asserts the outgoing session/
// request_permission params on the recorded frame: the mapped options
// verbatim in ask order and the toolCall patch carrying the backend
// toolCallId.
func assertRequestPermissionFrame(t *testing.T, raw string) {
	t.Helper()

	var frame struct {
		Method string `json:"method"`
		Params struct {
			SessionID string `json:"sessionId"`
			Options   []struct {
				Kind     string `json:"kind"`
				Name     string `json:"name"`
				OptionID string `json:"optionId"`
			} `json:"options"`
			ToolCall struct {
				ToolCallID string `json:"toolCallId"`
			} `json:"toolCall"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode outgoing frame: %v (%s)", err, raw)
	}

	if frame.Method != acp.ClientMethodSessionRequestPermission {
		t.Fatalf("outgoing method = %q, want session/request_permission", frame.Method)
	}

	ask := permissionAsk()

	if frame.Params.SessionID != ask.SessionID {
		t.Fatalf("request_permission sessionId = %q, want %q", frame.Params.SessionID, ask.SessionID)
	}

	if frame.Params.ToolCall.ToolCallID != ask.ToolCallID {
		t.Fatalf("request_permission toolCall.toolCallId = %q, want %q (the backend toolCallId must ride ToolCall)",
			frame.Params.ToolCall.ToolCallID, ask.ToolCallID)
	}

	if len(frame.Params.Options) != len(ask.Options) {
		t.Fatalf("request_permission options = %d entries, want %d (verbatim, in ask order)",
			len(frame.Params.Options), len(ask.Options))
	}

	for i, want := range ask.Options {
		got := frame.Params.Options[i]

		if got.Kind != want.Kind || got.Name != want.Name || got.OptionID != want.OptionID {
			t.Fatalf("request_permission options[%d] = %+v, want kind %q name %q optionId %q (verbatim passthrough)",
				i, got, want.Kind, want.Name, want.OptionID)
		}
	}
}

// assertPermissionOutcome compares the decoded neutral outcome.
func assertPermissionOutcome(t *testing.T, got, want acpport.PermissionOutcome) {
	t.Helper()

	if got != want {
		t.Fatalf("decoded outcome = %+v, want %+v", got, want)
	}
}

// TestSinkBindIsAtomic locks the WR-02 sink-publish contract (04-REVIEW,
// batched with the plan 05-02 agent.go seam touch): an adapter whose
// connection bound the sink hands every concurrent Prompt the BOUND sink
// (each sink write succeeds through the connection), while a directly
// constructed NewAgent adapter keeps the loud unbound fallback — its sink
// write fails with the stable no-connection-bound message, never a nil
// deref. Run under -race: every Prompt loads the sink pointer concurrently.
func TestSinkBindIsAtomic(t *testing.T) {
	h := newScriptedHandlers()
	h.promptReason = acpport.StopEndTurn

	toAgentR, toAgentW := io.Pipe()
	fromAgentR, fromAgentW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{toAgentR, toAgentW, fromAgentR, fromAgentW} {
			_ = c.Close()
		}
	})

	adapter := acpport.NewAgent(h)

	//nolint:dogsled // the adapter carries all three bindings; none is needed here
	_, _, _ = acpport.NewAgentSideConnection(adapter, fromAgentW, toAgentR)

	go func() { _, _ = io.Copy(io.Discard, fromAgentR) }() // drain the bound sinks' chunk writes

	const prompts = 16

	var wg sync.WaitGroup

	req := acp.PromptRequest{
		SessionId: testSessionID,
		Prompt:    []acp.ContentBlock{acp.TextBlock("sink-bind probe")},
	}

	for range prompts {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := adapter.Prompt(context.Background(), req); err != nil {
				t.Errorf("bound-adapter prompt: %v", err)
			}
		}()
	}

	wg.Wait()

	if len(h.sinkErrs) != prompts {
		t.Fatalf("recorded %d sink writes, want %d", len(h.sinkErrs), prompts)
	}

	for i, sinkErr := range h.sinkErrs {
		if sinkErr != nil {
			t.Fatalf("prompt %d's sink failed — the adapter handed over a non-bound sink: %v", i, sinkErr)
		}
	}

	// The unbound fallback: a directly constructed adapter's sink fails
	// loud with the stable message (no nil deref, no fake success).
	assertUnboundSinkFallback(t, req)
}

// assertUnboundSinkFallback drives a directly constructed NewAgent adapter
// and asserts its handed-over sink fails loud with the stable
// no-connection-bound message.
func assertUnboundSinkFallback(t *testing.T, req acp.PromptRequest) {
	t.Helper()

	loneH := newScriptedHandlers()
	loneH.promptReason = acpport.StopEndTurn
	lone := acpport.NewAgent(loneH)

	if _, err := lone.Prompt(context.Background(), req); err != nil {
		t.Fatalf("unbound prompt itself must answer normally (the reason is canned): %v", err)
	}

	if len(loneH.sinkErrs) == 0 || !strings.Contains(loneH.sinkErrs[0].Error(), "no connection bound") {
		t.Fatalf("unbound adapter's sink must fail loud naming the missing connection, got %v", loneH.sinkErrs)
	}
}

// drainFrames reads newline-delimited frames into lines until EOF.
func drainFrames(r io.Reader, lines chan<- string) {
	sc := bufio.NewScanner(r)

	for sc.Scan() {
		lines <- sc.Text()
	}
}

// readFrame waits for one wire frame, failing fast on timeout.
func readFrame(t *testing.T, lines <-chan string) string {
	t.Helper()

	select {
	case line := <-lines:
		return line
	case <-time.After(frameWaitTimeout):
		t.Fatalf("no frame arrived on the agent wire within %v", frameWaitTimeout)

		return ""
	}
}

// expectNoFrame asserts silence: no frame arrives within the wait window.
func expectNoFrame(t *testing.T, lines <-chan string, why string) {
	t.Helper()

	select {
	case line := <-lines:
		t.Fatalf("%s — got: %s", why, line)
	case <-time.After(frameWaitTimeout):
	}
}

// scriptedHandlers is the test-local Handlers fake: it records every call
// and replays canned results — the adapter contract's counterparty.
type scriptedHandlers struct {
	mu sync.Mutex

	initReport        acpport.InitReport
	newSessionID      string
	newSessionOptions []acpport.ConfigOptionDef
	promptReason      acpport.StopReason
	promptErr         error
	initErr           error
	newSessionErr     error
	cancelErr         error
	resumeID          string
	resumeErr         error
	closeErr          error
	authErr           error

	initCalls       int
	newSessionCalls []acpport.NewSessionParams
	promptCalls     []acpport.PromptParams
	sinks           []acpport.UpdateSink
	sinkErrs        []error
	cancelCalls     []string
	resumeCalls     []string
	resumeCwdCalls  []string
	closeCalls      []string
	authCalls       []string
}

// newScriptedHandlers returns the fake with the canned backend session id.
func newScriptedHandlers() *scriptedHandlers {
	return &scriptedHandlers{newSessionID: backendSessionID, resumeID: backendSessionID}
}

// Initialize records the call and replays the canned report.
func (s *scriptedHandlers) Initialize(ctx context.Context) (acpport.InitReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.initCalls++

	if s.initErr != nil {
		return acpport.InitReport{}, s.initErr
	}

	return s.initReport, nil
}

// NewSession records the params and replays the canned session id plus
// the canned neutral options (plan 06-04 — the adapter's select-assembly
// counterparty).
func (s *scriptedHandlers) NewSession(
	ctx context.Context, params acpport.NewSessionParams,
) (string, []acpport.ConfigOptionDef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.newSessionCalls = append(s.newSessionCalls, params)

	if s.newSessionErr != nil {
		return "", nil, s.newSessionErr
	}

	return s.newSessionID, s.newSessionOptions, nil
}

// Prompt records the params and sink, then answers with the canned reason
// and error. The sink is invoked once per prompt with a probe note and the
// write error recorded — the sink-binding contract asserts WHICH sink the
// adapter handed over (the bound one writes through the connection; the
// unbound fallback fails loud).
func (s *scriptedHandlers) Prompt(
	ctx context.Context,
	params acpport.PromptParams,
	sink acpport.UpdateSink,
) (acpport.StopReason, error) {
	s.mu.Lock()
	s.promptCalls = append(s.promptCalls, params)
	s.sinks = append(s.sinks, sink)
	s.mu.Unlock()

	probe := acpport.SessionNote{SessionID: params.SessionID, Kind: acpport.NoteMessage, Delta: "sink-probe"}
	sinkErr := sink(ctx, probe)

	s.mu.Lock()
	s.sinkErrs = append(s.sinkErrs, sinkErr)
	s.mu.Unlock()

	return s.promptReason, s.promptErr
}

// Cancel records the session id.
func (s *scriptedHandlers) Cancel(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelCalls = append(s.cancelCalls, sessionID)

	return s.cancelErr
}

// ResumeSession records the session id and the client cwd (plan 05-10
// WR-02 — the adapter forwards the SDK request's Cwd verbatim) and replays
// the canned reattached id (SESS-01, plan 05-05 — the lifecycle pair
// delegate through the same translate-call-code path as the other
// advertised methods).
func (s *scriptedHandlers) ResumeSession(ctx context.Context, sessionID, cwd string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resumeCalls = append(s.resumeCalls, sessionID)
	s.resumeCwdCalls = append(s.resumeCwdCalls, cwd)

	if s.resumeErr != nil {
		return "", s.resumeErr
	}

	return s.resumeID, nil
}

// CloseSession records the session id.
func (s *scriptedHandlers) CloseSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closeCalls = append(s.closeCalls, sessionID)

	return s.closeErr
}

// Authenticate records the method id and replays the canned auth outcome
// (AUTH-01, plan 05-07 — the adapter delegates; the honest chain lives in
// the front, this fake is its scripted counterparty).
func (s *scriptedHandlers) Authenticate(_ context.Context, methodID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authCalls = append(s.authCalls, methodID)

	return s.authErr
}

// totalCalls reports the number of Handlers invocations recorded.
func (s *scriptedHandlers) totalCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.initCalls + len(s.newSessionCalls) + len(s.promptCalls) + len(s.cancelCalls) +
		len(s.resumeCalls) + len(s.closeCalls) + len(s.authCalls)
}
