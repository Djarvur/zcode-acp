package main

import (
	"errors"
	"strings"
	"testing"
)

// Orca 1.4.271's exact non-interactive text-task argv (the
// commit-message-agent spec): --print --no-session --no-tools
// --no-extensions --no-skills --no-rules --mode text, an optional --model
// (Orca omits it for "default"), an optional --thinking level, then the
// prompt tokens. ompTextArgs builds it with the arm token in front.
func ompTextArgs(model, level string, prompts ...string) []string {
	argv := []string{
		ompSubcommand,
		"--print",
		"--no-session",
		"--no-tools",
		"--no-extensions",
		"--no-skills",
		"--no-rules",
		"--mode", "text",
	}

	if model != "" {
		argv = append(argv, "--model", model)
	}

	if level != "" {
		argv = append(argv, "--thinking", level)
	}

	return append(argv, prompts...)
}

// TestOmpTextArgs locks the omp argv text-task surface: the exact Orca argv,
// the marker rule (the arm token and the `acp` card marker are never prompt
// tokens), @file collection order, unknown-flag tolerance, and the mode
// validation sentinel.
func TestOmpTextArgs(t *testing.T) {
	t.Run("the exact Orca text argv parses to the text-task surface", func(t *testing.T) {
		opts, err := ompParse(ompTextArgs("zai-api/GLM-5.3-Flash", "high", "@/tmp/prompt.txt"))
		if err != nil {
			t.Fatalf("ompParse: %v", err)
		}

		if !opts.printMode {
			t.Fatal("printMode = false, want true (--print)")
		}

		if opts.mode != ompModeText {
			t.Fatalf("mode = %q, want %q", opts.mode, ompModeText)
		}

		if opts.model != "zai-api/GLM-5.3-Flash" {
			t.Fatalf("model = %q, want the --model value", opts.model)
		}

		if opts.thinking != "high" {
			t.Fatalf("thinking = %q, want the --thinking value", opts.thinking)
		}

		if !opts.noSession || !opts.noTools || !opts.noExtensions || !opts.noSkills || !opts.noRules {
			t.Fatalf("no-* flags not all true: %+v", opts)
		}

		if len(opts.prompts) != 1 || opts.prompts[0] != "@/tmp/prompt.txt" {
			t.Fatalf("prompts = %v, want the one @file token", opts.prompts)
		}

		if !opts.textMode() {
			t.Fatal("textMode() = false, want true")
		}
	})

	t.Run("--print alone and --mode text alone both activate text mode", func(t *testing.T) {
		for _, argv := range [][]string{
			{ompSubcommand, "--print", "hello"},
			{ompSubcommand, "--mode", "text", "hello"},
		} {
			opts, err := ompParse(argv)
			if err != nil {
				t.Fatalf("%v: %v", argv, err)
			}

			if !opts.textMode() {
				t.Fatalf("%v: textMode() = false, want true", argv)
			}
		}
	})

	t.Run("omp acp keeps printMode false and collects no prompt tokens", func(t *testing.T) {
		for _, argv := range [][]string{
			{ompSubcommand, ompCardMarker},
			{ompSubcommand, ompCardMarker, ompFlagModel, ompModelShort},
		} {
			opts, err := ompParse(argv)
			if err != nil {
				t.Fatalf("%v: %v", argv, err)
			}

			if opts.printMode || opts.textMode() {
				t.Fatalf("%v: text mode selected for the ACP card argv: %+v", argv, opts)
			}

			if len(opts.prompts) != 0 {
				t.Fatalf("%v: prompts = %v, want none (acp is the card marker)", argv, opts.prompts)
			}
		}
	})

	t.Run("the arm token and the card marker are never prompt tokens", func(t *testing.T) {
		opts, err := ompParse([]string{ompSubcommand, ompCardToken, "hello"})
		if err != nil {
			t.Fatalf("ompParse: %v", err)
		}

		if len(opts.prompts) != 1 || opts.prompts[0] != "hello" {
			t.Fatalf("prompts = %v, want the single literal prompt", opts.prompts)
		}
	})

	t.Run("two @file tokens are collected in argv order", func(t *testing.T) {
		opts, err := ompParse(ompTextArgs("", "", "@/tmp/a.txt", "@/tmp/b.txt"))
		if err != nil {
			t.Fatalf("ompParse: %v", err)
		}

		want := []string{"@/tmp/a.txt", "@/tmp/b.txt"}
		if len(opts.prompts) != 2 || opts.prompts[0] != want[0] || opts.prompts[1] != want[1] {
			t.Fatalf("prompts = %v, want %v (argv order)", opts.prompts, want)
		}
	})

	t.Run("unknown flags stay tolerated", func(t *testing.T) {
		opts, err := ompParse([]string{ompSubcommand, "--print", "--future-flag", "v", "--model", ompModelShort})
		if err != nil {
			t.Fatalf("ompParse: %v", err)
		}

		if opts.unknownCount != 1 || opts.model != ompModelShort {
			t.Fatalf("unknownCount = %d model = %q, want 1 and the parsed model", opts.unknownCount, opts.model)
		}
	})

	t.Run("--mode plan answers the unsupported-mode sentinel", func(t *testing.T) {
		opts, err := ompParse([]string{ompSubcommand, "--mode", "plan", "hello"})
		if err != nil {
			t.Fatalf("ompParse: %v", err)
		}

		if verr := opts.validateMode(); !errors.Is(verr, errUnsupportedMode) {
			t.Fatalf("validateMode(--mode plan) = %v, want the errUnsupportedMode sentinel", verr)
		}

		if !strings.Contains(opts.validateMode().Error(), "plan") {
			t.Fatalf("validateMode error %q must name the got mode", opts.validateMode())
		}
	})

	t.Run("--print with no positional parses cleanly with zero prompt tokens", func(t *testing.T) {
		opts, err := ompParse([]string{ompSubcommand, "--print"})
		if err != nil {
			t.Fatalf("ompParse: %v", err)
		}

		if !opts.printMode || len(opts.prompts) != 0 {
			t.Fatalf("opts = %+v, want printMode true and zero prompts", opts)
		}
	})

	t.Run("textLevel prefers --thinking over --reasoning-level", func(t *testing.T) {
		if got := textLevel(ompOptions{thinking: "high", reasoningLevel: "low"}); got != "high" {
			t.Fatalf("textLevel = %q, want the --thinking value", got)
		}

		if got := textLevel(ompOptions{reasoningLevel: "low"}); got != "low" {
			t.Fatalf("textLevel = %q, want the reasoning-level fallback", got)
		}

		if got := textLevel(ompOptions{}); got != "" {
			t.Fatalf("textLevel = %q, want empty", got)
		}
	})
}
