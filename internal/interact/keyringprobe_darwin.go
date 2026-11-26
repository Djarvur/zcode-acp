//go:build darwin

package interact

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// The macOS Keychain probe: the desktop stores the enc:v2:os key as a generic
// password item; the wrapper checks it with a bounded /usr/bin/security call.
// Missing and locked items are distinct coded arms, and stderr is classified
// by coarse markers only — never echoed.

// macOS Keychain sentinels.
var (
	ErrKeychainBinaryMissing = errors.New(
		"keychain probe: /usr/bin/security not found (macOS system security tool missing)")
	ErrKeychainNoEntry = errors.New(
		"keychain probe: no ZCode desktop credential found in the macOS Keychain " +
			"(log in to the ZCode desktop and retry)")
	ErrKeychainLocked = errors.New(
		"keychain probe: the macOS Keychain is locked or cannot prompt " +
			"(unlock the login keychain and retry)")
	ErrKeychainLookupTimeout = errors.New(
		"keychain probe: keychain lookup did not answer within its bound")
)

// KeyringProbeTimeout bounds the Keychain lookup subprocess.
const KeyringProbeTimeout = 5 * time.Second

// KeyringProbeExec is the injectable execution seam for the security lookup.
type KeyringProbeExec func(ctx context.Context, argv, env []string) (stdout, stderr []byte, err error)

// keyLookupCommander is the one argv source both desktop loaders expose.
type keyLookupCommander interface {
	LookupCommand() (argv, env []string, err error)
}

// keyringProber carries the macOS probe's injectable pieces.
type keyringProber struct {
	bin           string
	loader        keyLookupCommander
	lookPath      func(string) (string, error)
	exec          KeyringProbeExec
	timeout       time.Duration
	lockedMarkers []string
}

// KeyringProbeOption configures a KeyringProber at construction.
type KeyringProbeOption func(*keyringProber)

// WithKeyringProbeLookPath overrides the binary lookup seam.
func WithKeyringProbeLookPath(fn func(string) (string, error)) KeyringProbeOption {
	return func(p *keyringProber) { p.lookPath = fn }
}

// WithKeyringProbeExec overrides the subprocess execution seam.
func WithKeyringProbeExec(fn KeyringProbeExec) KeyringProbeOption {
	return func(p *keyringProber) { p.exec = fn }
}

// WithKeyringProbeTimeout overrides the bounded-lookup deadline.
func WithKeyringProbeTimeout(d time.Duration) KeyringProbeOption {
	return func(p *keyringProber) { p.timeout = d }
}

// NewKeychainProber returns the macOS probe over /usr/bin/security and the
// desktop credentials path.
func NewKeychainProber(bin, credentialsPath string, env []string,
	opts ...KeyringProbeOption,
) KeyringProber {
	p := &keyringProber{
		bin:      bin,
		loader:   NewSecurityKeyLoader(bin, credentialsPath, env),
		lookPath: exec.LookPath,
		exec:     defaultKeyringProbeExec,
		timeout:  KeyringProbeTimeout,
		lockedMarkers: []string{
			"User interaction is not allowed",
			"passphrase you entered is not correct",
			"could not be unlocked",
			"keychain is locked",
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p.probe
}

// defaultKeyringProbeExec runs the lookup with captured stdout/stderr.
func defaultKeyringProbeExec(ctx context.Context, argv, env []string) (stdout, stderr []byte, err error) {
	//nolint:gosec // argv is the built Keychain contract, never user input
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env

	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	err = cmd.Run()

	return out.Bytes(), errBuf.Bytes(), err
}

// The structural-log codes for the probe's arms.
const (
	probeClassBinaryMissing = "binary-missing"
	probeClassLocked        = "locked"
	probeClassTimeout       = "timeout"
	probeClassCancelled     = "cancelled"
)

// probe runs one proactive probe.
func (p *keyringProber) probe(ctx context.Context) error {
	started := time.Now()

	class, err := p.classify(ctx)

	slog.Info("interact: keyring probe", "class", class,
		"latency_ms", time.Since(started).Milliseconds())

	return err
}

// classify maps the Keychain sequence to the coded arms.
func (p *keyringProber) classify(ctx context.Context) (string, error) {
	if _, err := p.lookPath(p.bin); err != nil {
		return probeClassBinaryMissing, fmt.Errorf("%w: %w", ErrKeychainBinaryMissing, err)
	}

	argv, env, err := p.loader.LookupCommand()
	if err != nil {
		return probeClassNoEntry,
			fmt.Errorf("%w: no readable desktop credentials (%w)", ErrKeychainNoEntry, err)
	}

	pctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	stdout, stderr, execErr := p.exec(pctx, argv, env)

	switch {
	case ctx.Err() != nil:
		return probeClassCancelled, fmt.Errorf("keyring probe cancelled: %w", ctx.Err())

	case execErr == nil && len(bytes.TrimSpace(stdout)) > 0:
		return probeClassOK, nil

	case errors.Is(pctx.Err(), context.DeadlineExceeded):
		return probeClassTimeout, fmt.Errorf("%w (%s ceiling)", ErrKeychainLookupTimeout, p.timeout)

	case execErr != nil && containsAny(stderr, p.lockedMarkers):
		return probeClassLocked, fmt.Errorf("%w: %w", ErrKeychainLocked, execErr)

	default:
		return probeClassNoEntry,
			fmt.Errorf("%w (keychain lookup answered empty or rc≠0)", ErrKeychainNoEntry)
	}
}

// containsAny reports whether stderr contains one of the coarse markers.
func containsAny(stderr []byte, markers []string) bool {
	for _, marker := range markers {
		if bytes.Contains(stderr, []byte(marker)) {
			return true
		}
	}

	return false
}
