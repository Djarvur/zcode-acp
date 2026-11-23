// Package texttask implements the one-shot text-task engine behind the omp
// card's non-interactive surface (--print / --mode text): the @file prompt
// reader (one BOM stripped per chunk, argv-order concatenation), the
// selection resolver over the desktop's provider_config.json, the
// workspace/generateText request/response, and the minimal auth decision.
// Every exported error is a sentinel: consumers match with errors.Is, never
// the message text.
package texttask

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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

const (
	// maxPromptBytes caps each chunk and the accumulated prompt
	// (T-261124-04: an oversize prompt is a usage error before any backend
	// spawn, never an OOM vector).
	maxPromptBytes = 16 << 20

	// promptTokenPrefix marks one @file chunk token.
	promptTokenPrefix = "@"

	// bomPrefix is the UTF-8 BOM Orca prefixes to every prompt chunk file:
	// exactly one is stripped per chunk (bytes.TrimPrefix removes one
	// occurrence), never more.
	bomPrefix = "\ufeff"
)

// ReadPromptArg resolves the text-task prompt from the positional tokens in
// argv order: a token starting with @ names a prompt chunk file (absolute,
// regular, size-capped, one leading BOM stripped), every other token is a
// literal prompt chunk. Chunks are concatenated with no separator and the
// trimmed result must be non-empty. Every error names the path and byte
// sizes only — never prompt bytes.
func ReadPromptArg(tokens []string) (string, error) {
	if len(tokens) == 0 {
		return "", fmt.Errorf("%w: pass a literal prompt or one or more @file tokens", ErrPromptMissing)
	}

	var prompt strings.Builder

	for _, token := range tokens {
		chunk, err := promptChunk(token)
		if err != nil {
			return "", err
		}

		if prompt.Len()+len(chunk) > maxPromptBytes {
			return "", fmt.Errorf("%w: the concatenated prompt exceeds %d bytes", ErrPromptTooLarge, maxPromptBytes)
		}

		prompt.Write(chunk)
	}

	resolved := prompt.String()
	if strings.TrimSpace(resolved) == "" {
		return "", fmt.Errorf("%w: the prompt carries only whitespace", ErrPromptEmpty)
	}

	return resolved, nil
}

// promptChunk resolves one argv token: a literal prompt chunk verbatim, or
// one @file chunk read through a limiting reader (a growing file cannot
// exceed the cap) with exactly one leading BOM stripped.
func promptChunk(token string) ([]byte, error) {
	if !strings.HasPrefix(token, promptTokenPrefix) {
		if len(token) > maxPromptBytes {
			return nil, fmt.Errorf("%w: the literal token carries %d bytes (cap %d)",
				ErrPromptTooLarge, len(token), maxPromptBytes)
		}

		return []byte(token), nil
	}

	path := strings.TrimPrefix(token, promptTokenPrefix)
	if path == "" {
		return nil, fmt.Errorf("%w: the @ token names no path", ErrPromptUnreadable)
	}

	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("%w: %q is not absolute", ErrPromptUnreadable, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: stat %s: %w", ErrPromptUnreadable, path, err)
	}

	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %q is not a regular file", ErrPromptUnreadable, path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: open %s: %w", ErrPromptUnreadable, path, err)
	}

	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, maxPromptBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %w", ErrPromptUnreadable, path, err)
	}

	if len(data) > maxPromptBytes {
		return nil, fmt.Errorf("%w: %s carries more than %d bytes", ErrPromptTooLarge, path, maxPromptBytes)
	}

	return bytes.TrimPrefix(data, []byte(bomPrefix)), nil
}
