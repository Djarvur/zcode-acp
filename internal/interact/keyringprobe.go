package interact

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// The proactive keyring probe (AUTH-04, D-02, plan 06-05): the production
// prober answers whether the desktop keyring path is usable BEFORE the
// coding-plan authenticate chain pushes any snapshot traffic. Every failure
// arm is a coded actionable error naming the missing piece (06-RESEARCH.md
// Pattern 5's probed failure table — V-EXP 2026-11-22):
//
//   - secret-tool binary absent  → the install hint naming the package
//     (Ubuntu: libsecret-tools) — no subprocess is attempted;
//   - the D-Bus bus cannot autolaunch (no desktop session) → the
//     no-desktop-session hint — classified from a coarse stderr marker,
//     never stderr verbatim;
//   - no matching keyring entry (empty answer, rc≠0) → the log-in-via-
//     desktop hint;
//   - the lookup subprocess outlives its bounded context → the coded
//     timeout — the chain never hangs a turn (Pitfall 5; A3 belt-and-braces
//     against a locked-keyring prompt path nobody can answer headless).
//
// Hygiene posture (T-5-17 inheritance): the probe reads keyring state
// read-only, logs the arm class and latency only — never a value, never
// stderr content. The lookup argv comes from the SecretToolKeyLoader's
// LookupCommand (one argv source — the probe never re-derives the lookup
// shape) and runs under a bounded context with the loader's scrubbed env.

// errKeyringProbeStub is the compiling skeleton's marker (plan 06-05 RED):
// the GREEN step replaces the probe body with the real implementation.
var errKeyringProbeStub = errors.New("keyring probe skeleton: not implemented")

// The probed failure classes' sentinels — the coded messages ARE the
// client-facing actionable text (the authenticate arm wraps these verbatim;
// err113: wrapped with %w at the failure site, never constructed
// dynamically).
var (
	// ErrKeyringBinaryMissing: the secret-tool binary is not on PATH — the
	// error names the package to install (Ubuntu: libsecret-tools).
	ErrKeyringBinaryMissing = errors.New(
		"keyring probe: secret-tool not found (install libsecret-tools on Ubuntu: sudo apt install libsecret-tools)")

	// ErrKeyringNoDesktopSession: the keyring bus cannot autolaunch — the
	// coded no-desktop-session hint (the coarse autolaunch marker class).
	ErrKeyringNoDesktopSession = errors.New(
		"keyring probe: no desktop session available (the keyring bus cannot autolaunch headless — run inside your desktop session)")

	// ErrKeyringNoEntry: the keyring holds no matching ZCode credential —
	// the coded log-in-via-desktop hint.
	ErrKeyringNoEntry = errors.New(
		"keyring probe: no ZCode desktop login found in the keyring (log in via the ZCode desktop first)")

	// ErrKeyringLookupTimeout: the bounded lookup outlived its context —
	// the never-hangs coded timeout.
	ErrKeyringLookupTimeout = errors.New(
		"keyring probe: keyring lookup did not answer within its bound")
)

// KeyringProbeTimeout is the production bound on the lookup subprocess
// (06-RESEARCH.md Pattern 5's recommendation — belt-and-braces against a
// locked-keyring prompt path nobody can answer headless; A3).
const KeyringProbeTimeout = 5 * time.Second

// autolaunchMarker is the coarse stderr signal of the no-desktop-session
// class (the probed `secret-tool: Cannot autolaunch D-Bus …` failure mode).
// Classification checks CONTAINMENT of this marker only — the stderr text
// itself never reaches an error or a log.
const autolaunchMarker = "Cannot autolaunch"

// KeyringProber is the proactive keyring availability seam (OQ3 lock): a
// function type on the authenticate config. The production constructor
// builds it over the secret-tool binary + the credential loader; tests
// inject canned verdicts (the KeyLoader injection discipline). nil error =
// the keyring path is usable; every non-nil error is one of the coded
// sentinels above (possibly wrapped with coarse context — never stderr
// verbatim, never a value).
type KeyringProber func(ctx context.Context) error

// KeyringProbeExec is the injectable execution seam signature: run the
// lookup argv under the given (already bounded) context and env, returning
// stdout/stderr for classification. Production shells exec.CommandContext
// with capture buffers; tests script verdicts.
type KeyringProbeExec func(ctx context.Context, argv, env []string) (stdout, stderr []byte, err error)

// keyringProber carries the probe's injectable pieces. Constructed via
// NewKeyringProber; the seams default to production behavior.
type keyringProber struct {
	bin      string
	loader   *SecretToolKeyLoader
	lookPath func(string) (string, error)
	exec     KeyringProbeExec
	timeout  time.Duration
}

// KeyringProbeOption configures a KeyringProber at construction (tests
// inject canned verdicts; production keeps every default).
type KeyringProbeOption func(*keyringProber)

// WithKeyringProbeLookPath overrides the binary lookup seam (tests inject
// the missing-binary verdict; production uses exec.LookPath).
func WithKeyringProbeLookPath(fn func(string) (string, error)) KeyringProbeOption {
	return func(p *keyringProber) { p.lookPath = fn }
}

// WithKeyringProbeExec overrides the subprocess execution seam (tests
// script verdicts per arm; production runs the loader's argv).
func WithKeyringProbeExec(fn KeyringProbeExec) KeyringProbeOption {
	return func(p *keyringProber) { p.exec = fn }
}

// WithKeyringProbeTimeout overrides the bounded-lookup deadline (tests
// shorten it — the established deadline-knob posture).
func WithKeyringProbeTimeout(d time.Duration) KeyringProbeOption {
	return func(p *keyringProber) { p.timeout = d }
}

// NewKeyringProber returns the production probe over the secret-tool binary
// name, the desktop credentials.json path, and the scrubbed subprocess env
// (ScrubbedEnv). The lookup argv is the loader's LookupCommand contract —
// one argv source (the probe never re-derives the lookup shape).
func NewKeyringProber(bin, credentialsPath string, env []string, opts ...KeyringProbeOption) KeyringProber {
	p := &keyringProber{
		bin:      bin,
		loader:   NewSecretToolKeyLoader(bin, credentialsPath, env),
		lookPath: exec.LookPath,
		exec:     defaultKeyringProbeExec,
		timeout:  KeyringProbeTimeout,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p.probe
}

// defaultKeyringProbeExec is the production execution seam: the lookup argv
// under the caller's bounded context with capture buffers for stdout/stderr
// (the classification reads coarse signals off them; neither is ever logged
// or carried verbatim).
func defaultKeyringProbeExec(ctx context.Context, argv, env []string) (stdout, stderr []byte, err error) {
	//nolint:gosec // the argv is the loader's field-verified LookupCommand contract, not user input
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env

	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf

	err = cmd.Run()

	return out.Bytes(), errBuf.Bytes(), err
}

// probe runs one proactive probe (the skeleton arm — GREEN replaces this
// body with the real classification).
func (p *keyringProber) probe(ctx context.Context) error {
	return errKeyringProbeStub
}
