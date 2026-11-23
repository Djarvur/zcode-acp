package main

import (
	"errors"
	"strings"
	"testing"
)

// Text-task fixture literals (the goconst discipline): the client's own flag
// spellings, its model/level values, and the prompt-token shapes.
const (
	ompFlagPrint   = "--print"
	ompFlagMode    = "--mode"
	ompFlagFuture  = "--future-flag"
	ompModelFlash  = "zai-api/GLM-5.3-Flash"
	ompLevelLow    = "low"
	ompTextLiteral = "hello"
	ompPromptFile  = "@/tmp/prompt.txt"
	ompPromptOne   = "@/tmp/a.txt"
	ompPromptTwo   = "@/tmp/b.txt"
)

// ompTextArgs builds Orca 1.4.271's exact non-interactive text-task argv (the
// commit-message-agent spec): --print --no-session --no-tools
// --no-extensions --no-skills --no-rules --mode text, an optional --model
// (Orca omits it for "default"), an optional --thinking level, then the
// prompt tokens — with the arm token in front.
func ompTextArgs(model, level string, prompts ...string) []string {
	argv := []string{
		ompSubcommand,
		ompFlagPrint,
		"--no-session",
		"--no-tools",
		"--no-extensions",
		"--no-skills",
		"--no-rules",
		ompFlagMode, ompModeText,
	}

	if model != "" {
		argv = append(argv, ompFlagModel, model)
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
	t.Run("the exact Orca text argv parses to the text-task surface", testOmpTextExactArgv)

	t.Run("--print alone and --mode text alone both activate text mode", testOmpTextActivation)

	t.Run("omp acp keeps printMode false and collects no prompt tokens", testOmpTextAcpMarkers)

	t.Run("the arm token and the card marker are never prompt tokens", testOmpTextMarkerRule)

	t.Run("two @file tokens are collected in argv order", testOmpTextPromptOrder)

	t.Run("unknown flags stay tolerated", testOmpTextUnknownTolerance)

	t.Run("--mode plan answers the unsupported-mode sentinel", testOmpTextUnsupportedMode)

	t.Run("--print with no positional parses cleanly with zero prompt tokens", testOmpTextPrintNoPrompt)

	t.Run("textLevel prefers --thinking over --reasoning-level", testOmpTextLevel)
}

// testOmpTextExactArgv locks Orca's exact argv.
func testOmpTextExactArgv(t *testing.T) {
	t.Helper()

	opts, err := ompParse(ompTextArgs(ompModelFlash, e2eLevel, ompPromptFile))
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if !opts.printMode {
		t.Fatal("printMode = false, want true (--print)")
	}

	if opts.mode != ompModeText {
		t.Fatalf("mode = %q, want %q", opts.mode, ompModeText)
	}

	if opts.model != ompModelFlash {
		t.Fatalf("model = %q, want the --model value", opts.model)
	}

	if opts.thinking != e2eLevel {
		t.Fatalf("thinking = %q, want the --thinking value", opts.thinking)
	}

	if !opts.noSession || !opts.noTools || !opts.noExtensions || !opts.noSkills || !opts.noRules {
		t.Fatalf("no-* flags not all true: %+v", opts)
	}

	if len(opts.prompts) != 1 || opts.prompts[0] != ompPromptFile {
		t.Fatalf("prompts = %v, want the one @file token", opts.prompts)
	}

	if !opts.textMode() {
		t.Fatal("textMode() = false, want true")
	}
}

// testOmpTextActivation locks the either-flag activation rule.
func testOmpTextActivation(t *testing.T) {
	t.Helper()

	for _, argv := range [][]string{
		{ompSubcommand, ompFlagPrint, ompTextLiteral},
		{ompSubcommand, ompFlagMode, ompModeText, ompTextLiteral},
	} {
		opts, err := ompParse(argv)
		if err != nil {
			t.Fatalf("%v: %v", argv, err)
		}

		if !opts.textMode() {
			t.Fatalf("%v: textMode() = false, want true", argv)
		}
	}
}

// testOmpTextAcpMarkers locks the ACP card argv: no text mode, no prompts.
func testOmpTextAcpMarkers(t *testing.T) {
	t.Helper()

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
}

// testOmpTextMarkerRule locks the arm-token/card-marker exclusion.
func testOmpTextMarkerRule(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompCardToken, ompTextLiteral})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if len(opts.prompts) != 1 || opts.prompts[0] != ompTextLiteral {
		t.Fatalf("prompts = %v, want the single literal prompt", opts.prompts)
	}
}

// testOmpTextPromptOrder locks argv-order @file collection.
func testOmpTextPromptOrder(t *testing.T) {
	t.Helper()

	opts, err := ompParse(ompTextArgs("", "", ompPromptOne, ompPromptTwo))
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	want := []string{ompPromptOne, ompPromptTwo}
	if len(opts.prompts) != 2 || opts.prompts[0] != want[0] || opts.prompts[1] != want[1] {
		t.Fatalf("prompts = %v, want %v (argv order)", opts.prompts, want)
	}
}

// testOmpTextUnknownTolerance locks the unknown-flag tolerance arm.
func testOmpTextUnknownTolerance(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompFlagPrint, ompFlagFuture, "v", ompFlagModel, ompModelShort})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if opts.unknownCount != 1 || opts.model != ompModelShort {
		t.Fatalf("unknownCount = %d model = %q, want 1 and the parsed model", opts.unknownCount, opts.model)
	}
}

// testOmpTextUnsupportedMode locks the unsupported-mode sentinel.
func testOmpTextUnsupportedMode(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompFlagMode, "plan", ompTextLiteral})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	verr := opts.validateMode()
	if !errors.Is(verr, errUnsupportedMode) {
		t.Fatalf("validateMode(--mode plan) = %v, want the errUnsupportedMode sentinel", verr)
	}

	if !strings.Contains(verr.Error(), "plan") {
		t.Fatalf("validateMode error %q must name the got mode", verr)
	}
}

// testOmpTextPrintNoPrompt locks the parse-clean, zero-prompt shape (the
// missing-prompt exit-2 class is the run-time gate's case).
func testOmpTextPrintNoPrompt(t *testing.T) {
	t.Helper()

	opts, err := ompParse([]string{ompSubcommand, ompFlagPrint})
	if err != nil {
		t.Fatalf("ompParse: %v", err)
	}

	if !opts.printMode || len(opts.prompts) != 0 {
		t.Fatalf("opts = %+v, want printMode true and zero prompts", opts)
	}
}

// testOmpTextLevel locks the level precedence rule.
func testOmpTextLevel(t *testing.T) {
	t.Helper()

	if got := textLevel(ompOptions{thinking: e2eLevel, reasoningLevel: ompLevelLow}); got != e2eLevel {
		t.Fatalf("textLevel = %q, want the --thinking value", got)
	}

	if got := textLevel(ompOptions{reasoningLevel: ompLevelLow}); got != ompLevelLow {
		t.Fatalf("textLevel = %q, want the reasoning-level fallback", got)
	}

	if got := textLevel(ompOptions{}); got != "" {
		t.Fatalf("textLevel = %q, want empty", got)
	}
}
