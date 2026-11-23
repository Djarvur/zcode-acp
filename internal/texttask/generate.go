package texttask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Request is the one-shot text-task request: the resolved (and rebound)
// selection, the prompt text (already BOM-stripped and concatenated), and
// the workspace ref the app-server needs.
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
	// ErrEmptyText reports a reply whose trimmed text is empty: a coded
	// exit-1 failure beats an exit-0 with empty stdout (Orca's own
	// "returned an empty ..." failure either way, but ours is diagnosable).
	ErrEmptyText = errors.New("text task: the backend returned no text")
	// ErrInvalidWorkspace reports a missing, relative, or non-directory
	// workspace path.
	ErrInvalidWorkspace = errors.New("text task: invalid workspace path")
)

// textTaskQuerySource is the wrapper querySource literal: never the
// desktop's git_commit_message profile (a special commit-message binding on
// the app-server side).
const textTaskQuerySource = "zcode-acp-text-task"

// textMaxOutputTokens is the explicit generation cap the request carries.
// The desktop's strict model-option validator requires a maxOutputTokens
// value for any custom querySource: the git_commit_message profile is the
// only path that binds a model profile supplying the missing options, and a
// wrapper querySource gets no such profile. 1024 is the smallest cap the
// builtin registry allows (the glm-4v-flash rule; every other builtin rule
// is >= 4096) and is ample for Orca's text tasks (chat/branch names and
// commit messages capped at 1000 characters).
const textMaxOutputTokens = 1024

// camelCase wire-key constants (capture-verbatim dialect spelling — the
// request is assembled as a map, never a tagged struct, so the production
// decode structs stay tagliatelle snake-safe where they exist).
const (
	wireKeyWorkspace       = "workspace"
	wireKeyWorkspacePath   = "workspacePath"
	wireKeyWorkspaceKey    = "workspaceKey"
	wireKeySelection       = "selection"
	wireKeyProviderID      = "providerId"
	wireKeyModelID         = "modelId"
	wireKeyOptions         = "options"
	wireKeyReasoning       = "reasoningLevel"
	wireKeyPrompt          = "prompt"
	wireKeyQuerySource     = "querySource"
	wireKeyMaxOutputTokens = "maxOutputTokens"
)

// Run performs one workspace/generateText call and decodes the reply text.
// No tools or messages ride the request (--no-tools is satisfied by
// omission), while maxOutputTokens MUST ride it: the strict model-option
// validator rejects an undefined value for a non-git_commit_message
// querySource. The selection options object appears only when a level is
// set. A wire error surfaces as *zcode.CallError; a cancelled context
// surfaces as a wrapped context error; whitespace-only text is ErrEmptyText.
func Run(ctx context.Context, conn *zcode.Conn, req Request) (Result, error) {
	params := map[string]any{
		wireKeyWorkspace: map[string]any{
			wireKeyWorkspacePath: req.WorkspacePath,
			wireKeyWorkspaceKey:  req.WorkspaceKey,
		},
		wireKeySelection:       selectionParams(req.Selection),
		wireKeyPrompt:          req.Prompt,
		wireKeyQuerySource:     textTaskQuerySource,
		wireKeyMaxOutputTokens: textMaxOutputTokens,
	}

	res, err := conn.Call(ctx, zcode.MethodWorkspaceGenerateText, params)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Result{}, fmt.Errorf("text task: generate: %w", ctxErr)
		}

		return Result{}, fmt.Errorf("text task: generate: %w", err)
	}

	var decoded struct {
		Text         string `json:"text"`
		FinishReason string `json:"finishReason"` //nolint:tagliatelle // capture-verbatim
	}

	if err := json.Unmarshal(res, &decoded); err != nil {
		return Result{}, fmt.Errorf("text task: decode the generateText reply: %w", err)
	}

	if strings.TrimSpace(decoded.Text) == "" {
		return Result{}, fmt.Errorf("%w (finishReason %q)", ErrEmptyText, decoded.FinishReason)
	}

	return Result{Text: decoded.Text, FinishReason: decoded.FinishReason}, nil
}

// selectionParams renders the wire selection object: the ids verbatim and
// the options object only when a reasoning level is set (the omit-shape).
func selectionParams(sel acpfront.ModelSelection) map[string]any {
	selection := map[string]any{
		wireKeyProviderID: sel.ProviderID,
		wireKeyModelID:    sel.ModelID,
	}

	if sel.ReasoningLevel != "" {
		selection[wireKeyOptions] = map[string]any{wireKeyReasoning: sel.ReasoningLevel}
	}

	return selection
}

// WorkspaceRef validates the workspace directory for the generateText ref:
// the path must be absolute and name an existing directory; the returned
// path is the symlink-resolved absolute form. workspaceKey is the trimmed
// identity when set, else the path (docs/protocol/workspace-key.md's
// convention).
func WorkspaceRef(cwd, identity string) (path, key string, err error) {
	if cwd == "" || !filepath.IsAbs(cwd) {
		return "", "", fmt.Errorf("%w: %q must be an absolute path", ErrInvalidWorkspace, cwd)
	}

	info, err := os.Stat(cwd)
	if err != nil {
		return "", "", fmt.Errorf("%w: stat %s: %w", ErrInvalidWorkspace, cwd, err)
	}

	if !info.IsDir() {
		return "", "", fmt.Errorf("%w: %q is not a directory", ErrInvalidWorkspace, cwd)
	}

	resolved, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return "", "", fmt.Errorf("%w: resolve %s: %w", ErrInvalidWorkspace, cwd, err)
	}

	if strings.TrimSpace(identity) != "" {
		return resolved, identity, nil
	}

	return resolved, resolved, nil
}
