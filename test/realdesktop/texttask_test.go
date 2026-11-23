//go:build realdesktop

// The guarded real-tier NON-INTERACTIVE TEXT-TASK probe (quick task
// 261124-154) — MANUAL-ONLY, never CI (the Phase-3/5 discipline). The
// realdesktop build tag excludes this file from default builds, vet, and CI;
// the ZCODE_ACP_REAL_E2E=1 runtime guard skips the body even when built with
// the tag (belt and braces), and a missing desktop install skips too.
//
// Canonical run command (the ORCHESTRATOR's end-of-task act — one tiny model
// call, ledgered in experiments/README.md; the executor never runs the
// guarded form):
//
//	ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestTextTaskLive -v ./test/realdesktop/
//
// Optional: ZCODE_ACP_REAL_MODEL=providerId/modelId overrides the selection;
// unset exercises the desktop's config.defaultModelSelection path (Orca's
// "Config default"). The prompt asks for a one-word reply, so the spend is
// one tiny generateText call.
//
// Expected: PASS — Orca's exact argv exits 0, stdout is non-whitespace plain
// text with no protocol frame, stderr carries no -32601 marker, and the
// measured elapsed time stays inside the wrapper's documented worst case
// (40 s run budget + 15 s shutdown watchdog = 55 s under the client's 60 s
// kill). Without the env var AND the tag the test is skipped/absent — CI
// never contacts the desktop or the account.

//nolint:testpackage // the guarded real-tier compositions live in-package (apicycle/codingplan precedent)
package realdesktop

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Text-probe knobs (names unique to this file — the realdesktop package's
// apicycle_test.go owns the API-key cycle's constants).
const (
	// envTextModel optionally overrides the selection (providerId/modelId);
	// unset exercises the provider_config default path.
	envTextModel = "ZCODE_ACP_REAL_MODEL"

	// textProbeKill is the client's own kill budget (Orca SIGKILLs the card
	// at 60 s).
	textProbeKill = 60 * time.Second

	// textProbeMaxElapsed is the wrapper's documented worst case: one 40 s
	// run budget plus the 15 s shutdown watchdog. A harness kill at 60 s is a
	// different, visibly distinct failure.
	textProbeMaxElapsed = 55 * time.Second

	// textProbeBuildDeadline bounds the wrapper build.
	textProbeBuildDeadline = 180 * time.Second

	// textProbePrompt is the one tiny model call: a one-word reply keeps the
	// spend minimal and the assertion non-whitespace.
	textProbePrompt = "Reply with the single word: ok"
)

// TestTextTaskLive runs Orca's exact non-interactive argv against the
// discovered real desktop and asserts the stdout/exit/elapsed contract.
// MANUAL-ONLY (see the file header).
func TestTextTaskLive(t *testing.T) {
	if os.Getenv(realE2EEnv) != "1" {
		t.Skip("manual-only: set ZCODE_ACP_REAL_E2E=1 AND build with -tags realdesktop — " +
			"CI never contacts the desktop or the account")
	}

	ctx, cancel := context.WithTimeout(context.Background(), textProbeKill)
	defer cancel()

	discoverTextProbeInstall(ctx, t)

	wrapper := buildTextProbeWrapper(t)
	workspace := t.TempDir()
	promptPath := writeTextProbePrompt(t, t.TempDir())

	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, wrapper, textProbeArgv(promptPath)...)
	cmd.Dir = workspace
	cmd.Env = os.Environ()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	started := time.Now()

	runErr := cmd.Run()
	elapsed := time.Since(started)

	assertTextProbeOutcome(ctx, t, runErr, elapsed, stdout.String(), stderr.String())
}

// textProbeArgv is Orca's exact text-task argv: the mode flags, --model only
// when ZCODE_ACP_REAL_MODEL is set (otherwise the provider_config default is
// exercised), then the one @file prompt token.
func textProbeArgv(promptPath string) []string {
	argv := []string{
		"omp",
		"--print",
		"--no-session",
		"--no-tools",
		"--no-extensions",
		"--no-skills",
		"--no-rules",
		"--mode", "text",
	}

	if model := os.Getenv(envTextModel); model != "" {
		argv = append(argv, "--model", model)
	}

	return append(argv, "@"+promptPath)
}

// discoverTextProbeInstall runs the REAL production discovery (the default
// prober matrix over the user's home) and logs the resolved binary and
// layout — the diagnostics are the product. A host without an install SKIPS.
func discoverTextProbeInstall(ctx context.Context, t *testing.T) supervisor.Install {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home for discovery: %v", err)
	}

	inst, probes, err := supervisor.DefaultProbers(home, optDir).Discover(ctx)
	if err != nil {
		for _, row := range probes {
			t.Logf("probe %-16s %s -> %s", row.Name, row.Query, row.Result)
		}

		if errors.Is(err, supervisor.ErrNoInstall) {
			t.Skip("no ZCode desktop install found on this host — the guarded text probe needs one")
		}

		t.Fatalf("discover: %v", err)
	}

	t.Logf("resolved install: layout=%s root=%s binary=%s entrypoint=%s builtin=%s",
		inst.Layout, inst.Root, inst.Binary, inst.Entrypoint, inst.BuiltinFile)

	return inst
}

// buildTextProbeWrapper builds the wrapper binary into the test's temp dir.
func buildTextProbeWrapper(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), textProbeBuildDeadline)
	defer cancel()

	binPath := filepath.Join(t.TempDir(), "zcode-acp")

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"github.com/Djarvur/zcode-acp/cmd/zcode-acp")
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("go build wrapper: %v", err)
	}

	return binPath
}

// writeTextProbePrompt writes the BOM-prefixed prompt file (Orca's delivery
// shape) and returns its absolute path.
func writeTextProbePrompt(t *testing.T, dir string) string {
	t.Helper()

	path := filepath.Join(dir, "prompt-0.txt")
	if err := os.WriteFile(path, []byte("\ufeff"+textProbePrompt), 0o600); err != nil {
		t.Fatalf("write prompt file %s: %v", path, err)
	}

	return path
}

// assertTextProbeOutcome asserts the full contract and fails with the
// captured stderr plus the measured elapsed time so a wire refusal is
// diagnosable.
func assertTextProbeOutcome(
	ctx context.Context, t *testing.T, runErr error, elapsed time.Duration, stdout, stderr string,
) {
	t.Helper()

	if ctx.Err() != nil {
		t.Fatalf("the 60 s client kill fired after %s (the wrapper's documented worst case is %s) — stderr: %s",
			elapsed.Truncate(time.Millisecond), textProbeMaxElapsed, stderr)
	}

	if runErr != nil {
		t.Fatalf("wrapper exit = %v after %s — stderr: %s", runErr, elapsed.Truncate(time.Millisecond), stderr)
	}

	if strings.TrimSpace(stdout) == "" {
		t.Fatalf("stdout is whitespace-only after %s — stderr: %s", elapsed.Truncate(time.Millisecond), stderr)
	}

	if strings.Contains(stdout, `"jsonrpc"`) {
		t.Fatalf("stdout carries a protocol frame, not plain text: %q", stdout)
	}

	if strings.Contains(stderr, "-32601") {
		t.Fatalf("stderr carries the -32601 method-not-found marker "+
			"(workspace/generateText is unusable on this desktop): %s", stderr)
	}

	if elapsed > textProbeMaxElapsed {
		t.Fatalf("elapsed %s exceeds the %s worst case (40 s run budget + 15 s shutdown watchdog)",
			elapsed.Truncate(time.Millisecond), textProbeMaxElapsed)
	}

	t.Logf("text task ok: exit 0, %d stdout bytes, %s elapsed (one tiny model call)",
		len(stdout), elapsed.Truncate(time.Millisecond))
}
