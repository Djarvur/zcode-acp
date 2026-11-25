//go:build linux

package interact

// IN-03 (plan 07-02) in-package classify-order locks. The classify switch is
// a pure function of its scripted inputs — (parent-cancel, deadline, exec
// verdict, stderr marker) — so the arm evaluation ORDER is asserted directly
// on the unexported classify; the external-package suite
// (keyringprobe_test.go) stays the client-facing arm-contract locks through
// the public KeyringProber seam. All subprocess material is synthetic
// fixture data — the scan-secrets gate stays green.
//
//	TestKeyringProbeClassifyDeadlineBoundarySuccess — a lookup that
//	  SUCCEEDS exactly at the deadline boundary (execErr nil, non-blank
//	  stdout, the probe's own bounded ctx already past its deadline)
//	  classifies ok, never a false timeout (the RED arm: the success arm
//	  must be evaluated BEFORE the deadline arm — D-08 IN-03 / T-07-06).
//	The sibling locks pin the rest of the order: parent-cancel wins first
//	even over a success-shaped verdict, a bound-killed EMPTY lookup still
//	classifies timeout, and the autolaunch-marker and default arms are
//	unchanged.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// errClassifyFixtureExit replays the probed rc=1 failure modes (a static
// fixture error — err113: wrapped static errors, never constructed
// dynamically).
var errClassifyFixtureExit = errors.New("exit status 1")

// classifyFixtureCredentials is the fixture credentials.json the loader's
// LookupCommand needs to resolve (contents irrelevant to the argv contract —
// only the resolved path matters).
const classifyFixtureCredentials = `{"fixture":"not-a-credential"}`

// classifyFixtureAnswer is the synthetic non-blank keyring answer (the
// success shape; never a real credential value).
const classifyFixtureAnswer = "fixture-synthetic-keyring-answer"

// classifyFixtureEnv is the fixture scrubbed env (synthetic allowlist
// members only).
func classifyFixtureEnv() []string {
	return []string{"PATH=/usr/bin", "HOME=/fixture-home"}
}

// classifyFixtureProber builds a prober over fully scripted seams: LookPath
// answers present, the credentials path resolves, exec answers the caller's
// verdict — so the classify inputs are exactly the scripted (parent ctx,
// deadline, exec result, stderr marker) tuple.
func classifyFixtureProber(t *testing.T, exec KeyringProbeExec, timeout time.Duration) *keyringProber {
	t.Helper()

	credPath := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(credPath, []byte(classifyFixtureCredentials), 0o600); err != nil {
		t.Fatalf("write credentials fixture: %v", err)
	}

	return &keyringProber{
		bin:      "secret-tool",
		loader:   NewSecretToolKeyLoader("secret-tool", credPath, classifyFixtureEnv()),
		lookPath: func(string) (string, error) { return "/usr/bin/secret-tool", nil },
		exec:     exec,
		timeout:  timeout,
	}
}

// TestKeyringProbeClassifyDeadlineBoundarySuccess is the IN-03 arm (T-07-06):
// the lookup succeeds exactly at the deadline boundary — execErr nil,
// non-blank stdout, the probe's own bounded ctx already past its deadline —
// and must classify ok, never a false timeout: the success arm is evaluated
// BEFORE the deadline arm.
func TestKeyringProbeClassifyDeadlineBoundarySuccess(t *testing.T) {
	p := classifyFixtureProber(t, func(ctx context.Context, _, _ []string) ([]byte, []byte, error) {
		<-ctx.Done() // the subprocess answers exactly at the boundary...

		return []byte(classifyFixtureAnswer), nil, nil // ...but answers SUCCESS
	}, 80*time.Millisecond)

	class, err := p.classify(context.Background())

	if class != probeClassOK || err != nil {
		t.Fatalf("boundary-timed success classified %q (err %v) — want %q with nil error "+
			"(IN-03: the success arm must win over the deadline arm)", class, err, probeClassOK)
	}
}

// TestKeyringProbeClassifyParentCancelWinsFirst locks the coarseness order
// above IN-03: a cancelled PARENT ctx classifies cancelled FIRST — even when
// the exec verdict would look like a success (the caller's cancellation is
// the verdict, never a probe-class misread).
func TestKeyringProbeClassifyParentCancelWinsFirst(t *testing.T) {
	p := classifyFixtureProber(t, func(_ context.Context, _, _ []string) ([]byte, []byte, error) {
		return []byte(classifyFixtureAnswer), nil, nil
	}, KeyringProbeTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	class, err := p.classify(ctx)

	if class != probeClassCancelled || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled parent classified %q (err %v) — want %q wrapping context.Canceled "+
			"(the parent cancellation is authoritative)", class, err, probeClassCancelled)
	}
}

// TestKeyringProbeClassifyTimeoutOnEmptyStdout locks the genuine timeout arm
// through the reorder: the lookup killed at its bound with NO answer bytes
// still classifies timeout — the IN-03 success-first order must not lose the
// coded never-hangs verdict.
func TestKeyringProbeClassifyTimeoutOnEmptyStdout(t *testing.T) {
	p := classifyFixtureProber(t, func(ctx context.Context, _, _ []string) ([]byte, []byte, error) {
		<-ctx.Done()

		return nil, nil, ctx.Err()
	}, 80*time.Millisecond)

	class, err := p.classify(context.Background())

	if class != probeClassTimeout || !errors.Is(err, ErrKeyringLookupTimeout) {
		t.Fatalf("bound-killed empty lookup classified %q (err %v) — want %q wrapping ErrKeyringLookupTimeout",
			class, err, probeClassTimeout)
	}
}

// TestKeyringProbeClassifyMarkerAndDefaultArms locks the coarse-marker and
// default arms verbatim: the autolaunch marker in stderr answers
// no-desktop-session (never stderr verbatim in the error); rc≠0 without the
// marker answers the no-entry log-in hint.
func TestKeyringProbeClassifyMarkerAndDefaultArms(t *testing.T) {
	t.Run("autolaunch marker answers no-desktop-session", func(t *testing.T) {
		p := classifyFixtureProber(t, func(_ context.Context, _, _ []string) ([]byte, []byte, error) {
			return nil, []byte("secret-tool: Cannot autolaunch D-Bus without X11 $DISPLAY"), errClassifyFixtureExit
		}, KeyringProbeTimeout)

		class, err := p.classify(context.Background())

		if class != probeClassNoDesktopSession || !errors.Is(err, ErrKeyringNoDesktopSession) {
			t.Fatalf("autolaunch-marker lookup classified %q (err %v) — want %q wrapping ErrKeyringNoDesktopSession",
				class, err, probeClassNoDesktopSession)
		}

		if err != nil && strings.Contains(err.Error(), "X11 $DISPLAY") {
			t.Fatalf("no-desktop-session error %q carries stderr verbatim — "+
				"coarse-marker classification only", err.Error())
		}
	})

	t.Run("rc nonzero without the marker answers no-entry", func(t *testing.T) {
		p := classifyFixtureProber(t, func(_ context.Context, _, _ []string) ([]byte, []byte, error) {
			return nil, nil, errClassifyFixtureExit
		}, KeyringProbeTimeout)

		class, err := p.classify(context.Background())

		if class != probeClassNoEntry || !errors.Is(err, ErrKeyringNoEntry) {
			t.Fatalf("empty rc≠0 lookup classified %q (err %v) — want %q wrapping ErrKeyringNoEntry",
				class, err, probeClassNoEntry)
		}
	})
}
