package texttask_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/texttask"
)

// bom is the UTF-8 BOM Orca prefixes to every prompt chunk file.
const bom = "\ufeff"

// writePromptFile writes content under the test's temp dir and returns the
// absolute path.
func writePromptFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write prompt file %s: %v", path, err)
	}

	return path
}

// TestReadPromptArgChunks covers the happy prompt-delivery contract:
// literal tokens, @file chunks with one BOM stripped per file, and
// argv-order concatenation with no separator.
func TestReadPromptArgChunks(t *testing.T) {
	t.Run("a single literal token is the prompt", func(t *testing.T) {
		got, err := texttask.ReadPromptArg([]string{testLiteralPrompt})
		if err != nil {
			t.Fatalf("ReadPromptArg: %v", err)
		}

		if got != testLiteralPrompt {
			t.Fatalf("ReadPromptArg = %q, want %q", got, testLiteralPrompt)
		}
	})

	t.Run("a single @file strips exactly one leading BOM", func(t *testing.T) {
		path := writePromptFile(t, "prompt-0.txt", bom+testLiteralPrompt)

		got, err := texttask.ReadPromptArg([]string{"@" + path})
		if err != nil {
			t.Fatalf("ReadPromptArg: %v", err)
		}

		if got != testLiteralPrompt {
			t.Fatalf("ReadPromptArg = %q, want the BOM-free %q", got, testLiteralPrompt)
		}
	})

	t.Run("two @file chunks concatenate in argv order with no separator", func(t *testing.T) {
		first := writePromptFile(t, "prompt-0.txt", bom+"Answer: ")
		second := writePromptFile(t, "prompt-1.txt", bom+"done")

		got, err := texttask.ReadPromptArg([]string{"@" + first, "@" + second})
		if err != nil {
			t.Fatalf("ReadPromptArg: %v", err)
		}

		if got != "Answer: done" {
			t.Fatalf("ReadPromptArg = %q, want the argv-order concatenation %q", got, "Answer: done")
		}
	})
}

// TestReadPromptArgErrors covers the prompt-delivery sentinel classes and
// the no-prompt-bytes-in-errors rule.
func TestReadPromptArgErrors(t *testing.T) {
	t.Run("a missing file is ErrPromptUnreadable naming the path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "absent.txt")

		_, err := texttask.ReadPromptArg([]string{"@" + path})
		if !errors.Is(err, texttask.ErrPromptUnreadable) {
			t.Fatalf("ReadPromptArg(missing) error = %v, want ErrPromptUnreadable", err)
		}

		if !strings.Contains(err.Error(), path) {
			t.Fatalf("error %q must name the path %q", err, path)
		}
	})

	t.Run("a relative @ path is rejected", func(t *testing.T) {
		_, err := texttask.ReadPromptArg([]string{"@prompt.txt"})
		if !errors.Is(err, texttask.ErrPromptUnreadable) {
			t.Fatalf("ReadPromptArg(relative) error = %v, want ErrPromptUnreadable", err)
		}
	})

	t.Run("a non-regular path is rejected", func(t *testing.T) {
		dir := t.TempDir()

		_, err := texttask.ReadPromptArg([]string{"@" + dir})
		if !errors.Is(err, texttask.ErrPromptUnreadable) {
			t.Fatalf("ReadPromptArg(directory) error = %v, want ErrPromptUnreadable", err)
		}
	})

	t.Run("an oversize file is ErrPromptTooLarge without echoing content", func(t *testing.T) {
		path := writePromptFile(t, "big.txt", strings.Repeat("S", (16<<20)+1))

		_, err := texttask.ReadPromptArg([]string{"@" + path})
		if !errors.Is(err, texttask.ErrPromptTooLarge) {
			t.Fatalf("ReadPromptArg(oversize) error = %v, want ErrPromptTooLarge", err)
		}

		if strings.Contains(err.Error(), "SSSS") {
			t.Fatalf("error %q must not carry prompt bytes", err)
		}
	})

	t.Run("an oversize literal token is ErrPromptTooLarge without echoing content", func(t *testing.T) {
		_, err := texttask.ReadPromptArg([]string{strings.Repeat("x", (16<<20)+1)})
		if !errors.Is(err, texttask.ErrPromptTooLarge) {
			t.Fatalf("ReadPromptArg(oversize literal) error = %v, want ErrPromptTooLarge", err)
		}

		if strings.Contains(err.Error(), "xxxx") {
			t.Fatalf("error %q must not carry prompt bytes", err)
		}
	})
}

// TestReadPromptArgEmpty covers the empty/missing prompt classes.
func TestReadPromptArgEmpty(t *testing.T) {
	t.Run("a whitespace-only result is ErrPromptEmpty", func(t *testing.T) {
		_, err := texttask.ReadPromptArg([]string{"  \n\t"})
		if !errors.Is(err, texttask.ErrPromptEmpty) {
			t.Fatalf("ReadPromptArg(whitespace) error = %v, want ErrPromptEmpty", err)
		}
	})

	t.Run("no tokens is ErrPromptMissing", func(t *testing.T) {
		_, err := texttask.ReadPromptArg(nil)
		if !errors.Is(err, texttask.ErrPromptMissing) {
			t.Fatalf("ReadPromptArg(nil) error = %v, want ErrPromptMissing", err)
		}
	})
}
