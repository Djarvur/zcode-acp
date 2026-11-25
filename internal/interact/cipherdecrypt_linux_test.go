//go:build linux

package interact_test

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/interact"
)

// TestSecretToolLoaderHygiene locks the production loader's subprocess
// contract (T-5-17): the argv carries the field-verified lookup shape with
// the username derived from the RESOLVED credentials path (never key
// material — the loader holds no key yet), and the env is exactly the
// caller-provided scrubbed set.
func TestSecretToolLoaderHygiene(t *testing.T) {
	target := filepath.Join(t.TempDir(), "v2", "credentials.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	link := filepath.Join(t.TempDir(), "link-credentials.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	scrubbed := []string{"PATH=/usr/bin", "HOME=/home/u"}
	loader := interact.NewSecretToolKeyLoader("secret-tool", link, scrubbed)

	argv, env, err := loader.LookupCommand()
	if err != nil {
		t.Fatalf("LookupCommand: %v", err)
	}

	if len(argv) < 2 || argv[0] != "secret-tool" || argv[1] != "lookup" {
		t.Fatalf("argv = %v, want the secret-tool lookup invocation", argv)
	}

	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "service") || !strings.Contains(joined, "ZCode Credential Encryption") {
		t.Fatalf("argv %q lacks the field-verified service attribute", joined)
	}

	// The username derivation hashes the RESOLVED (realpath) path string:
	// the symlinked fixture must hash the target, not the link.
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}

	wantUsername := hex.EncodeToString(cipherSha256(resolved))
	wantArgv := "secret-tool lookup service ZCode Credential Encryption username " + wantUsername

	if joined != wantArgv {
		t.Fatalf("argv = %q, want the exact field-verified lookup shape %q", joined, wantArgv)
	}

	if len(env) != len(scrubbed) {
		t.Fatalf("env = %v, want exactly the caller-provided scrubbed set %v", env, scrubbed)
	}

	for i, kv := range scrubbed {
		if env[i] != kv {
			t.Fatalf("env[%d] = %q, want %q (the scrubbed set passes through verbatim)", i, env[i], kv)
		}
	}
}

// TestSecretToolLoaderEmptyPath locks the loader's fail-loud input
// validation: an empty credentials path answers an error at both the
// command build and the load — never a phantom-path lookup.
func TestSecretToolLoaderEmptyPath(t *testing.T) {
	loader := interact.NewSecretToolKeyLoader("secret-tool", "", nil)

	if _, _, err := loader.LookupCommand(); err == nil {
		t.Fatal("empty credentials path must fail loud, got nil error")
	}

	if _, err := loader.LoadKey(context.Background()); err == nil {
		t.Fatal("empty credentials path must fail loud at LoadKey, got nil error")
	}
}
