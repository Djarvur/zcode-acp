package texttask

import (
	"context"
	"errors"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Request is the one-shot text-task request: the resolved selection, the
// prompt text (already BOM-stripped and concatenated), and the workspace
// ref the app-server needs.
type Request struct {
	Selection     acpfront.ModelSelection
	Prompt        string
	WorkspacePath string
	WorkspaceKey  string
}

// Result is the generateText reply: the final text and the backend's
// finishReason (optional on the wire, empty when absent).
type Result struct {
	Text         string
	FinishReason string
}

// Generate sentinels.
var (
	// ErrEmptyText reports a reply whose trimmed text is empty.
	ErrEmptyText = errors.New("text task: the backend returned no text")
	// ErrInvalidWorkspace reports a missing, relative, or non-directory
	// workspace path.
	ErrInvalidWorkspace = errors.New("text task: invalid workspace path")
)

// textTaskQuerySource is the wrapper querySource literal (never the
// desktop's git_commit_message profile).
const textTaskQuerySource = "zcode-acp-text-task"

// camelCase wire-key constants (capture-verbatim dialect spelling).
const (
	wireKeyWorkspace     = "workspace"
	wireKeyWorkspacePath = "workspacePath"
	wireKeyWorkspaceKey  = "workspaceKey"
	wireKeySelection     = "selection"
	wireKeyProviderID    = "providerId"
	wireKeyModelID       = "modelId"
	wireKeyOptions       = "options"
	wireKeyReasoning     = "reasoningLevel"
	wireKeyPrompt        = "prompt"
	wireKeyQuerySource   = "querySource"
	wireKeyText          = "text"
	wireKeyFinishReason  = "finishReason"
)

// Run performs one workspace/generateText call and decodes the reply (stub:
// RED skeleton).
func Run(ctx context.Context, conn *zcode.Conn, req Request) (Result, error) {
	_ = ctx
	_ = conn
	_ = req

	return Result{}, nil
}

// WorkspaceRef validates the workspace directory and derives the wire ref
// (stub: RED skeleton).
func WorkspaceRef(cwd, identity string) (string, string, error) {
	_ = cwd
	_ = identity

	return "", "", nil
}
