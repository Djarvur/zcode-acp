package interact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// macOS Keychain credential reader (Phase 14, MAUTH-02/MAUTH-03).
//
// The desktop seals credential values as enc:v2:os:<fp24>:<iv>.<tag>.<ct> and
// stores the AES key as a generic-password item:
//
//	service = "ZCode Credential Encryption"
//	account = hex(sha256(resolved credentials.json path))
//
// The bridge plays the desktop host, so it reads the same item with a bounded
// /usr/bin/security invocation. Key material exists only in memory; the argv
// carries the service/account lookup keys (the account is a PATH hash, never a
// secret), and failure details never reach logs or the wire.

// DefaultSecurityBin is the macOS system keychain CLI.
const DefaultSecurityBin = "/usr/bin/security"

// DefaultSecretToolBin is the Linux Secret Service CLI.
const DefaultSecretToolBin = "secret-tool"

// SecurityLookupTimeout bounds one security invocation even when the caller's
// context is longer: a locked/ACL-prompted Keychain must degrade to a coded
// error, never hang a turn.
const SecurityLookupTimeout = 15 * time.Second

// errSecurityLookup is the Keychain lookup failure sentinel.
var errSecurityLookup = errors.New("macOS Keychain lookup failed")

// KeyLookupExec is the injectable subprocess seam for the security reader.
type KeyLookupExec func(ctx context.Context, argv, env []string) (stdout []byte, err error)

// SecurityKeyLoader reads the ZCode credential-encryption key from the macOS
// Keychain. It implements KeyLoader.
type SecurityKeyLoader struct {
	bin      string
	credPath string
	env      []string
	run      KeyLookupExec
}

// NewSecurityKeyLoader returns the production Keychain reader.
func NewSecurityKeyLoader(bin, credentialsPath string, env []string) *SecurityKeyLoader {
	return &SecurityKeyLoader{
		bin:      bin,
		credPath: credentialsPath,
		env:      env,
		run:      defaultSecurityExec,
	}
}

// NewSecurityKeyLoaderWithExec returns the reader over an injected exec seam.
// Tests use it to lock the argv/output contract without a real keychain.
func NewSecurityKeyLoaderWithExec(bin, credentialsPath string, env []string,
	run KeyLookupExec,
) *SecurityKeyLoader {
	loader := NewSecurityKeyLoader(bin, credentialsPath, env)
	loader.run = run

	return loader
}

// LookupCommand builds the Keychain lookup contract:
//
//	security find-generic-password -s "ZCode Credential Encryption" -a <hex> -w
//
// The account is the sha256 hex of the RESOLVED credentials.json path — the
// same identity the desktop writes under.
func (l *SecurityKeyLoader) LookupCommand() (argv, env []string, err error) {
	if l.bin == "" || l.credPath == "" {
		return nil, nil, errEmptyCredentialsPath
	}

	link, err := resolveRealPath(l.credPath)
	if err != nil {
		return nil, nil, err
	}

	sum := sha256.Sum256([]byte(link))
	account := hex.EncodeToString(sum[:])

	argv = []string{
		l.bin, "find-generic-password",
		"-s", credentialKeyringService,
		"-a", account,
		"-w",
	}

	return argv, l.env, nil
}

// LoadKey executes the lookup and decodes the base64url 32-byte key.
func (l *SecurityKeyLoader) LoadKey(ctx context.Context) ([]byte, error) {
	argv, env, err := l.LookupCommand()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSecurityLookup, err)
	}

	out, err := l.run(ctx, argv, env)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errSecurityLookup, err)
	}

	return decodeLoaderKey(out)
}

// defaultSecurityExec runs the bounded lookup with stdout captured.
func defaultSecurityExec(ctx context.Context, argv, env []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, SecurityLookupTimeout)
	defer cancel()

	//nolint:gosec // argv is the built Keychain contract, never user input
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run %s: %w", argv[0], err)
	}

	return out, nil
}
