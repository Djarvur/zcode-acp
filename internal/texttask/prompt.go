// Package texttask implements the one-shot text-task engine behind the omp
// card's non-interactive surface (--print / --mode text): the @file prompt
// reader (one BOM stripped per chunk, argv-order concatenation), the
// selection resolver over the desktop's provider_config.json, the
// workspace/generateText request/response, and the minimal auth decision.
// Every exported error is a sentinel: consumers match with errors.Is, never
// the message text.
package texttask

import "errors"

// Prompt-reader sentinels. The messages name the path and byte sizes only —
// never prompt bytes (T-261124-02: prompt text is private user content).
var (
	// ErrPromptMissing reports an argv with no literal and no @file token.
	ErrPromptMissing = errors.New("text task: no prompt supplied")
	// ErrPromptUnreadable reports a non-absolute, missing, or non-regular
	// @file path.
	ErrPromptUnreadable = errors.New("text task: prompt file is unreadable")
	// ErrPromptTooLarge reports a chunk or the concatenated prompt above the
	// 16 MiB cap.
	ErrPromptTooLarge = errors.New("text task: prompt exceeds the size cap")
	// ErrPromptEmpty reports a prompt whose trimmed value is empty.
	ErrPromptEmpty = errors.New("text task: prompt is empty")
)

// maxPromptBytes caps each chunk and the accumulated prompt (T-261124-04:
// an oversize prompt is a usage error before any backend spawn, never an
// OOM vector).
const maxPromptBytes = 16 << 20

// promptTokenPrefix marks one @file chunk token.
const promptTokenPrefix = "@"

// ReadPromptArg resolves the text-task prompt from the positional tokens
// (stub: RED skeleton).
func ReadPromptArg(tokens []string) (string, error) {
	_ = tokens

	return "", nil
}
