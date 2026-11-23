package texttask_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/texttask"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Test-fixture literals shared across this package's test files (the
// goconst discipline): the personal provider pair the request fixtures use,
// the builtin z.ai pair the rebind/selection fixtures use, and the
// assertion surfaces.
const (
	testProviderPersonal = "OpenCodeGo"
	testModelPersonal    = "deepseek-v4.1-flash"
	testProviderZaiAPI   = "zai-api"
	testModelFlash       = "GLM-5.3-Flash"
	testLevelHigh        = "high"
	testWorkspacePath    = "/repo"
	testLiteralPrompt    = "hello"
)

// generateReply is the scripted success reply (the desktop's result shape).
const generateReply = `{"text":"Answer: done","selection":` +
	`{"providerId":"OpenCodeGo","modelId":"deepseek-v4.1-flash"},"finishReason":"stop"}`

// generateScenario scripts one workspace/generateText reply.
func generateScenario(result string) *fake.Scenario {
	return &fake.Scenario{
		Name: "texttask-generate",
		OnRequest: []fake.RequestRule{{
			Method: zcode.MethodWorkspaceGenerateText,
			Reply:  &fake.Reply{Result: json.RawMessage(result)},
		}},
	}
}

// generateFrame is one decoded transcript request frame.
type generateFrame struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// generateParams is the decoded generateText params surface the assertions
// read.
type generateParams struct {
	Workspace struct {
		WorkspacePath string `json:"workspacePath"`
		WorkspaceKey  string `json:"workspaceKey"`
	} `json:"workspace"`
	Selection struct {
		ProviderID string `json:"providerId"`
		ModelID    string `json:"modelId"`
		Options    *struct {
			ReasoningLevel string `json:"reasoningLevel"`
		} `json:"options"`
	} `json:"selection"`
	Prompt      string `json:"prompt"`
	QuerySource string `json:"querySource"`
}

// rideGenerate runs one Run call against a scripted fake over net.Pipe and
// returns the recorded transcript plus Run's error. The host stream closes
// after Run returns, which is what ends the fake's scan loop.
func rideGenerate(
	ctx context.Context, t *testing.T, sc *fake.Scenario, req texttask.Request,
) (fake.Transcript, error) {
	t.Helper()

	hostEnd, fakeEnd := net.Pipe()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	//nolint:contextcheck // NewConn owns its base context (ctx only bounds the call below)
	conn := zcode.NewConn(hostEnd)

	done := make(chan fake.Transcript, 1)
	serveErrCh := make(chan error, 1)

	go func() {
		tr, err := fake.Serve(fakeEnd, sc)
		done <- tr
		serveErrCh <- err
	}()

	_, runErr := texttask.Run(ctx, conn, req)
	_ = conn.Close()

	select {
	case tr := <-done:
		if err := <-serveErrCh; err != nil {
			t.Fatalf("fake.Serve: %v", err)
		}

		return tr, runErr
	case <-time.After(10 * time.Second):
		t.Fatal("fake.Serve did not return after the host stream closed")

		return fake.Transcript{}, runErr
	}
}

// singleGenerateFrame returns the one inbound generateText frame, failing
// on any other count (the one-shot contract).
func singleGenerateFrame(t *testing.T, tr fake.Transcript) generateFrame {
	t.Helper()

	frames := make([]generateFrame, 0, 1)

	for _, line := range tr.Lines {
		if line.Dir != fake.DirIn {
			continue
		}

		var frame generateFrame
		if err := json.Unmarshal([]byte(line.JSON), &frame); err != nil {
			t.Fatalf("decode inbound frame %q: %v", line.JSON, err)
		}

		if frame.Method == zcode.MethodWorkspaceGenerateText {
			frames = append(frames, frame)
		}
	}

	if len(frames) != 1 {
		t.Fatalf("transcript carries %d %s frames, want exactly 1",
			len(frames), zcode.MethodWorkspaceGenerateText)
	}

	return frames[0]
}

// decodeGenerateParams decodes the frame's params into the typed view and
// the raw key set (the tools/messages absence assertion).
func decodeGenerateParams(
	t *testing.T, frame generateFrame,
) (params generateParams, raw map[string]json.RawMessage) {
	t.Helper()

	if err := json.Unmarshal(frame.Params, &params); err != nil {
		t.Fatalf("decode generateText params: %v", err)
	}

	if err := json.Unmarshal(frame.Params, &raw); err != nil {
		t.Fatalf("decode generateText params key set: %v", err)
	}

	return params, raw
}

// defaultRequest is the canonical happy-path request.
func defaultRequest(level string) texttask.Request {
	return texttask.Request{
		Selection: acpfront.ModelSelection{
			ProviderID: testProviderPersonal, ModelID: testModelPersonal, ReasoningLevel: level,
		},
		Prompt:        "Name this chat.",
		WorkspacePath: testWorkspacePath,
		WorkspaceKey:  testWorkspacePath,
	}
}

// frameRaw re-marshals the raw params for failure messages (order-stable by
// encoding/json's sorted map keys).
func frameRaw(raw map[string]json.RawMessage) string {
	out, err := json.Marshal(raw)
	if err != nil {
		return "<undecodable>"
	}

	return string(out)
}

// TestRunGenerateRequest locks the request assembly and the reply decode
// over a real zcode.Conn: the method, every params field, the wrapper
// querySource, and the absence of tools/messages/maxOutputTokens.
func TestRunGenerateRequest(t *testing.T) {
	tr, runErr := rideGenerate(
		context.Background(), t, generateScenario(generateReply), defaultRequest(testLevelHigh))
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}

	params, raw := decodeGenerateParams(t, singleGenerateFrame(t, tr))

	if params.Workspace.WorkspacePath != testWorkspacePath ||
		params.Workspace.WorkspaceKey != testWorkspacePath {
		t.Fatalf("workspace params = %+v, want the request's path/key verbatim", params.Workspace)
	}

	if params.Selection.ProviderID != testProviderPersonal || params.Selection.ModelID != testModelPersonal {
		t.Fatalf("selection params = %+v, want the request's ids", params.Selection)
	}

	if params.Prompt != "Name this chat." {
		t.Fatalf("prompt = %q, want the request prompt verbatim", params.Prompt)
	}

	if params.QuerySource != "zcode-acp-text-task" {
		t.Fatalf("querySource = %q, want the wrapper literal", params.QuerySource)
	}

	for _, forbidden := range []string{"tools", "messages", "maxOutputTokens"} {
		if _, ok := raw[forbidden]; ok {
			t.Fatalf("params carry a %q key; --no-tools must omit it (params %s)", forbidden, frameRaw(raw))
		}
	}

	if params.Selection.Options == nil || params.Selection.Options.ReasoningLevel != testLevelHigh {
		t.Fatalf("selection.options = %+v, want the configured level", params.Selection.Options)
	}
}

// TestRunGenerateOptionsOmitted locks the omit-shape: an empty level emits
// no options object at all.
func TestRunGenerateOptionsOmitted(t *testing.T) {
	tr, runErr := rideGenerate(context.Background(), t, generateScenario(generateReply), defaultRequest(""))
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}

	params, raw := decodeGenerateParams(t, singleGenerateFrame(t, tr))

	if params.Selection.Options != nil {
		t.Fatalf("selection.options = %+v, want omitted for an empty level", params.Selection.Options)
	}

	var selectionRaw map[string]json.RawMessage
	if err := json.Unmarshal(raw["selection"], &selectionRaw); err != nil {
		t.Fatalf("decode selection key set: %v", err)
	}

	if _, ok := selectionRaw["options"]; ok {
		t.Fatalf("selection carries an options key for an empty level: %s", raw["selection"])
	}
}

// TestRunGenerateCodingPlanRebindWire is the wire proof for the rebind: the
// coding-plan auth mode turns the builtin zai-api selection into the account
// provider before the request is sent.
func TestRunGenerateCodingPlanRebindWire(t *testing.T) {
	req := defaultRequest("")
	req.Selection = texttask.ApplyCodingPlanRebind(
		acpfront.ModelSelection{ProviderID: testProviderZaiAPI, ModelID: testModelFlash},
		acpfront.AuthModeCodingPlan)

	tr, runErr := rideGenerate(context.Background(), t, generateScenario(generateReply), req)
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}

	params, _ := decodeGenerateParams(t, singleGenerateFrame(t, tr))

	if params.Selection.ProviderID != "account:zai-individual-coding-plan" {
		t.Fatalf("selection.providerId = %q, want the rebound account provider", params.Selection.ProviderID)
	}
}

// TestRunGenerateEmptyText locks the empty-text sentinel: whitespace-only
// generated text is an honest exit-1-class failure, never an empty success.
func TestRunGenerateEmptyText(t *testing.T) {
	reply := `{"text":"  \n\t","selection":{"providerId":"p","modelId":"m"},"finishReason":"stop"}`

	_, runErr := rideGenerate(context.Background(), t, generateScenario(reply), defaultRequest(""))
	if !errors.Is(runErr, texttask.ErrEmptyText) {
		t.Fatalf("Run(whitespace) error = %v, want ErrEmptyText", runErr)
	}
}

// TestRunGenerateWireError locks the coded wire-error surface: the numeric
// code is matchable through *zcode.CallError.
func TestRunGenerateWireError(t *testing.T) {
	sc := &fake.Scenario{
		Name: "texttask-generate-error",
		OnRequest: []fake.RequestRule{{
			Method: zcode.MethodWorkspaceGenerateText,
			Reply:  &fake.Reply{Error: &zcode.WireError{Code: zcode.CodeInternalError, Message: "boom"}},
		}},
	}

	_, runErr := rideGenerate(context.Background(), t, sc, defaultRequest(""))
	if runErr == nil {
		t.Fatal("Run(wire error) succeeded, want the coded error")
	}

	var callErr *zcode.CallError
	if !errors.As(runErr, &callErr) || callErr.Err.Code != zcode.CodeInternalError {
		t.Fatalf("Run(wire error) = %v, want a *zcode.CallError carrying %d", runErr, zcode.CodeInternalError)
	}
}

// TestRunGenerateCancelledContext locks the cancellation surface: a
// cancelled context surfaces as a wrapped context error.
func TestRunGenerateCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, runErr := rideGenerate(ctx, t, generateScenario(generateReply), defaultRequest(""))
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("Run(cancelled) error = %v, want a wrapped context.Canceled", runErr)
	}
}

// TestRunGenerateResult is the typed-result assertion (Run's return value,
// not the transcript): the text and finishReason decode from the reply.
func TestRunGenerateResult(t *testing.T) {
	hostEnd, fakeEnd := net.Pipe()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	conn := zcode.NewConn(hostEnd)

	done := make(chan struct{})

	go func() {
		_, _ = fake.Serve(fakeEnd, generateScenario(generateReply))
		close(done)
	}()

	res, err := texttask.Run(context.Background(), conn, defaultRequest(""))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	_ = conn.Close()
	<-done

	if res.Text != "Answer: done" || res.FinishReason != "stop" {
		t.Fatalf("Result = %+v, want text %q and finishReason %q", res, "Answer: done", "stop")
	}
}

// TestWorkspaceRef locks the workspace-ref derivation: the absolute
// existing directory, the trimmed identity rule, and the invalid sentinel.
func TestWorkspaceRef(t *testing.T) {
	t.Run("an existing absolute directory resolves with the identity as key", func(t *testing.T) {
		dir := t.TempDir()

		path, key, err := texttask.WorkspaceRef(dir, " ws-identity ")
		if err != nil {
			t.Fatalf("WorkspaceRef: %v", err)
		}

		wantPath, err := filepath.EvalSymlinks(dir)
		if err != nil {
			t.Fatalf("EvalSymlinks(%s): %v", dir, err)
		}

		if path != wantPath {
			t.Fatalf("workspacePath = %q, want the resolved directory %q", path, wantPath)
		}

		if key != " ws-identity " {
			t.Fatalf("workspaceKey = %q, want the raw identity verbatim", key)
		}
	})

	t.Run("an empty identity means the path is the key", func(t *testing.T) {
		dir := t.TempDir()

		path, key, err := texttask.WorkspaceRef(dir, "  ")
		if err != nil {
			t.Fatalf("WorkspaceRef: %v", err)
		}

		if key != path {
			t.Fatalf("workspaceKey = %q, want the path %q for a blank identity", key, path)
		}
	})

	t.Run("a relative cwd is ErrInvalidWorkspace", func(t *testing.T) {
		if _, _, err := texttask.WorkspaceRef("relative/dir", ""); !errors.Is(err, texttask.ErrInvalidWorkspace) {
			t.Fatalf("WorkspaceRef(relative) error = %v, want ErrInvalidWorkspace", err)
		}
	})

	t.Run("a nonexistent cwd is ErrInvalidWorkspace", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "absent")

		if _, _, err := texttask.WorkspaceRef(dir, ""); !errors.Is(err, texttask.ErrInvalidWorkspace) {
			t.Fatalf("WorkspaceRef(absent) error = %v, want ErrInvalidWorkspace", err)
		}
	})

	t.Run("a regular file is ErrInvalidWorkspace", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file.txt")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}

		if _, _, err := texttask.WorkspaceRef(file, ""); !errors.Is(err, texttask.ErrInvalidWorkspace) {
			t.Fatalf("WorkspaceRef(file) error = %v, want ErrInvalidWorkspace", err)
		}
	})
}
