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
		"keyring probe: no desktop session available " +
			"(the keyring bus cannot autolaunch headless — run inside your desktop session)")

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

// keyLookupCommander is the one argv source both desktop loaders expose.
type keyLookupCommander interface {
	LookupCommand() (argv, env []string, err error)
}

// keyringProbeMessages carries the coded sentinels for one platform family.
type keyringProbeMessages struct {
	binaryMissing    error
	noDesktopSession error
	noEntry          error
	timeout          error
	locked           error
}

func linuxProbeMessages() keyringProbeMessages {
	return keyringProbeMessages{
		binaryMissing:    ErrKeyringBinaryMissing,
		noDesktopSession: ErrKeyringNoDesktopSession,
		noEntry:          ErrKeyringNoEntry,
		timeout:          ErrKeyringLookupTimeout,
	}
}

// macOS Keychain sentinels: missing/locked items are distinct actionable
// errors, and no key material is ever named.
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

func keychainProbeMessages() keyringProbeMessages {
	return keyringProbeMessages{
		binaryMissing: ErrKeychainBinaryMissing,
		noEntry:       ErrKeychainNoEntry,
		timeout:       ErrKeychainLookupTimeout,
		locked:        ErrKeychainLocked,
	}
}

// keyringProber carries the probe's injectable pieces. Constructed via
// NewKeyringProber; the seams default to production behavior.
type keyringProber struct {
	bin           string
	loader        keyLookupCommander
	lookPath      func(string) (string, error)
	exec          KeyringProbeExec
	timeout       time.Duration
	messages      keyringProbeMessages
	lockedMarkers []string
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
		messages: linuxProbeMessages(),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p.probe
}

// NewKeychainProber returns the macOS probe over /usr/bin/security and the
// desktop credentials path. The lookup argv is SecurityKeyLoader's contract;
// locked and missing items classify to distinct coded errors.
func NewKeychainProber(bin, credentialsPath string, env []string,
	opts ...KeyringProbeOption,
) KeyringProber {
	p := &keyringProber{
		bin:      bin,
		loader:   NewSecurityKeyLoader(bin, credentialsPath, env),
		lookPath: exec.LookPath,
		exec:     defaultKeyringProbeExec,
		timeout:  KeyringProbeTimeout,
		messages: keychainProbeMessages(),
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

// The structural-log codes for the probe's arms (the class name only —
// latency and class are the ONLY probe facts ever logged; no value, no
// stderr content).
const (
	probeClassOK               = "ok"
	probeClassBinaryMissing    = "binary-missing"
	probeClassNoDesktopSession = "no-desktop-session"
	probeClassNoEntry          = "no-entry"
	probeClassLocked           = "locked"
	probeClassTimeout          = "timeout"
	probeClassCancelled        = "cancelled"
)

// probe runs one proactive probe: LookPath gate → the loader's LookupCommand
// argv (one argv source) under a bounded context with the scrubbed env →
// coarse-signal classification into the coded arms. The arm class and
// latency are the only logged facts (T-5-17 inheritance — never a value,
// never stderr content).
func (p *keyringProber) probe(ctx context.Context) error {
	started := time.Now()

	class, err := p.classify(ctx)

	slog.Info("interact: keyring probe", "class", class,
		"latency_ms", time.Since(started).Milliseconds())

	return err
}

// classify runs the probed sequence and maps each failure mode to its coded
// arm (06-RESEARCH.md Pattern 5's failure table). Every non-nil error wraps
// one of the exported sentinels — the authenticate arm's client-facing
// actionable text.
func (p *keyringProber) classify(ctx context.Context) (string, error) {
	// Tests construct the struct directly and may omit the message set;
	// production constructors always fill it. Default to the Linux family.
	msgs := p.messages
	if msgs.binaryMissing == nil {
		msgs = linuxProbeMessages()
	}

	// 1. LookPath FIRST (D-02): an absent binary is the cheapest honest
	// verdict and never spawns a subprocess.
	if _, err := p.lookPath(p.bin); err != nil {
		return probeClassBinaryMissing, fmt.Errorf("%w: %w", msgs.binaryMissing, err)
	}

	// 2. The loader's exact argv — one argv source (the probe never
	// re-derives the lookup shape). An unresolvable credentials path fails
	// HERE, before any subprocess: a fresh machine is simply not logged in
	// (the log-in hint).
	argv, env, err := p.loader.LookupCommand()
	if err != nil {
		return probeClassNoEntry,
			fmt.Errorf("%w: no readable desktop credentials (%w)", msgs.noEntry, err)
	}

	// 3. The bounded exec: the ≤5s ceiling contains any prompt path nobody
	// can answer headless (Pitfall 5; A3 belt-and-braces).
	pctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	stdout, stderr, execErr := p.exec(pctx, argv, env)

	// 4. Classification, coarsest signals first, the SUCCESS arm ahead of
	// the deadline arm (D-08 IN-03 / T-07-06): a lookup that answers
	// successfully exactly at the boundary must classify ok — the deadline
	// arm below it only claims the false-timeout verdict for lookups that
	// did NOT hand back a usable answer. The probe's OWN bounded ctx state
	// is authoritative for the timeout arm (the exec error's wording is an
	// implementation detail). A cancelled PARENT ctx surfaces the
	// cancellation, not a probe-class verdict. The autolaunch marker in
	// stderr is the no-desktop-session class; everything else (rc≠0 or an
	// empty answer) is the log-in hint. Success = rc 0 with a non-empty
	// answer — the answer bytes themselves are discarded unread beyond the
	// emptiness check (the probe proves availability, it is not the key
	// reader).
	switch {
	case ctx.Err() != nil:
		return probeClassCancelled, fmt.Errorf("keyring probe cancelled: %w", ctx.Err())

	case execErr == nil && len(bytes.TrimSpace(stdout)) > 0:
		return probeClassOK, nil

	case errors.Is(pctx.Err(), context.DeadlineExceeded):
		return probeClassTimeout, fmt.Errorf("%w (%s ceiling)", msgs.timeout, p.timeout)

	case execErr != nil && msgs.noDesktopSession != nil &&
		bytes.Contains(stderr, []byte(autolaunchMarker)):
		return probeClassNoDesktopSession,
			fmt.Errorf("%w: %w", msgs.noDesktopSession, execErr)

	case execErr != nil && containsAny(stderr, p.lockedMarkers):
		return probeClassLocked, fmt.Errorf("%w: %w", msgs.locked, execErr)

	default:
		return probeClassNoEntry,
			fmt.Errorf("%w (keyring lookup answered empty or rc≠0)", msgs.noEntry)
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
