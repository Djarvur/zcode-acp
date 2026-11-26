//go:build linux

package interact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os/exec"
)

// SecretToolKeyLoader is the production KeyLoader: it shells the
// field-verified lookup `secret-tool lookup service "ZCode Credential
// Encryption" username <sha256(resolved credentials path)>` on the
// caller-provided scrubbed env, captures the stdout (the 43-char base64url
// key), and releases it into the reply scope. Real tier only: CI never
// constructs nor executes it against a keyring (D-06) — the loader runs
// only when a cipher ask actually arrives on a wired binary.
type SecretToolKeyLoader struct {
	bin      string
	credPath string
	env      []string
}

// NewSecretToolKeyLoader returns the loader over the secret-tool binary
// name, the desktop credentials.json path (resolved at lookup time), and
// the scrubbed subprocess env (see ScrubbedEnv).
func NewSecretToolKeyLoader(bin, credentialsPath string, env []string) *SecretToolKeyLoader {
	return &SecretToolKeyLoader{bin: bin, credPath: credentialsPath, env: env}
}

// LookupCommand builds the lookup subprocess contract the loader executes:
// the argv and the env it runs under. Exported for the hygiene locks —
// tests inspect the contract without executing the binary. The username
// argument is the sha256 hex derivation over the RESOLVED credentials path
// (EvalSymlinks + Abs — the realpath discipline), derived from a PATH
// STRING, never key material: the loader holds no key at argv-build time.
func (l *SecretToolKeyLoader) LookupCommand() (argv, env []string, err error) {
	if l.bin == "" || l.credPath == "" {
		return nil, nil, errEmptyCredentialsPath
	}

	link, err := resolveRealPath(l.credPath)
	if err != nil {
		return nil, nil, err
	}

	sum := sha256.Sum256([]byte(link))
	username := hex.EncodeToString(sum[:])

	argv = []string{l.bin, "lookup", "service", credentialKeyringService, "username", username}

	return argv, l.env, nil
}

// LoadKey shells the lookup and decodes the captured stdout: trimmed
// base64url, exactly 32 bytes. cmd.Output captures and drains stdout (the
// key never touches a pipe the process doesn't own) and carries stderr on
// the exec error — which this package wraps and never logs (hygiene: the
// failure detail rides nowhere; the decline's generic message is the wire
// surface).
func (l *SecretToolKeyLoader) LoadKey(ctx context.Context) ([]byte, error) {
	argv, env, err := l.LookupCommand()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoaderLookup, err)
	}

	//nolint:gosec // the argv is the built field-verified contract, not user input
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoaderLookup, err)
	}

	key, err := decodeLoaderKey(out)
	if err != nil {
		return nil, err
	}

	return key, nil
}
