//go:build linux

package interact_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/interact"
)

// TestKeyringProbe locks the proactive keyring probe (AUTH-04, D-02, plan
// 06-05) arm by arm — 06-RESEARCH.md Pattern 5's probed failure table made
// executable:
//
//	(a) the secret-tool binary is absent → the coded install hint NAMING
//	    the package (libsecret-tools) with NO subprocess attempted;
//	(b) the D-Bus bus cannot autolaunch → the coded no-desktop-session
//	    hint, classified from the coarse marker — never stderr verbatim;
//	(c) an empty keyring answer → the coded log-in-via-desktop hint;
//	(d) the argv contract: the probe runs the LOADER'S LookupCommand argv
//	    (one argv source) under a bounded context with the loader's
//	    scrubbed env;
//	(e) a lookup path that cannot resolve (no credentials.json) → the
//	    log-in hint with no subprocess attempted;
//	(f) the never-hangs lock: a hung subprocess seam answers the coded
//	    timeout well inside the test's own watchdog bound.
//
// All subprocess material is synthetic fixture data — the scan-secrets gate
// stays green.
func TestKeyringProbe(t *testing.T) {
	t.Run("missing binary answers the install hint with no exec", testKeyringProbeMissingBinary)
	t.Run("no D-Bus session answers the desktop-session hint, never stderr verbatim",
		testKeyringProbeNoDesktopSession)
	t.Run("empty keyring answer answers the log-in-via-desktop hint", testKeyringProbeNoEntry)
	t.Run("argv comes from the loader's LookupCommand under a bounded ctx", testKeyringProbeArgvContract)
	t.Run("unresolvable credentials path answers the log-in hint with no exec",
		testKeyringProbeUnresolvablePath)
	t.Run("hung lookup answers the coded timeout inside the bound", testKeyringProbeHungExec)
}

// The arms' static fixture errors (err113: wrapped static errors, never
// dynamically constructed).
var (
	// errProbeLookPathAbsent replays the exec.LookPath miss verbatim.
	errProbeLookPathAbsent = errors.New(`exec: "secret-tool": executable file not found in $PATH`)

	// errProbeExitStatus1 replays the probed rc=1 failure modes.
	errProbeExitStatus1 = errors.New("exit status 1")
)

const fixtureHomeEnv = "HOME=/fixture-home"

// probeEnv is the fixture scrubbed env (synthetic allowlist members only).
func probeEnv() []string {
	return []string{"PATH=/usr/bin", fixtureHomeEnv}
}

// probeFixtureCredentials writes the fixture credentials.json the loader's
// LookupCommand realpath-derivation needs (contents are irrelevant to the
// argv contract — only the resolved path is hashed).
func probeFixtureCredentials(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, []byte(`{"fixture":"not-a-credential"}`), 0o600); err != nil {
		t.Fatalf("write credentials fixture: %v", err)
	}

	return path
}

// probeLookPathAvailable is the fixture success seam for every arm that
// exercises the exec path: the host's real secret-tool presence must never
// decide a test arm (the CI runner does not install libsecret-tools). Only
// arm (a) scripts LookPath absent.
func probeLookPathAvailable(string) (string, error) {
	return "/usr/bin/secret-tool", nil
}

// recordedProbeExec records every exec-seam invocation (argv/env/deadline
// facts) — the injection discipline the arms use to prove ordering and
// non-execution. Verdict scripting rides the per-arm WithKeyringProbeExec
// closures; this double's own verdict is the synthetic success.
type recordedProbeExec struct {
	mu    sync.Mutex
	calls []probeExecCall
}

type probeExecCall struct {
	argv []string
	env  []string

	// deadline/hasDeadline carry the bounded-ctx facts the argv-contract
	// arm asserts — the ctx itself is never stored (containedctx).
	deadline    time.Time
	hasDeadline bool
}

// exec is the seam: records the invocation (deadline facts only — never
// the ctx), then returns the synthetic success verdict (an entry exists).
func (r *recordedProbeExec) exec(
	ctx context.Context, argv, env []string,
) (stdout, stderr []byte, err error) {
	dl, dlOK := ctx.Deadline()

	r.mu.Lock()
	r.calls = append(r.calls, probeExecCall{argv: argv, env: env, deadline: dl, hasDeadline: dlOK})
	r.mu.Unlock()

	return []byte("fixture-synthetic-keyring-answer"), nil, nil
}

// count reports the recorded invocation count.
func (r *recordedProbeExec) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.calls)
}

// testKeyringProbeMissingBinary is arm (a): LookPath answers absent — the
// probe must answer the coded install hint naming libsecret-tools WITHOUT
// attempting any subprocess (D-02: LookPath first).
func testKeyringProbeMissingBinary(t *testing.T) {
	exec := &recordedProbeExec{}

	probe := interact.NewKeyringProber("secret-tool", probeFixtureCredentials(t), probeEnv(),
		interact.WithKeyringProbeLookPath(func(string) (string, error) {
			return "", errProbeLookPathAbsent
		}),
		interact.WithKeyringProbeExec(exec.exec),
	)

	err := probe(context.Background())

	if !errors.Is(err, interact.ErrKeyringBinaryMissing) {
		t.Fatalf("missing-binary probe error = %v, want the coded ErrKeyringBinaryMissing", err)
	}

	if !strings.Contains(err.Error(), "libsecret-tools") {
		t.Fatalf("missing-binary error %q must name the package to install (libsecret-tools)", err.Error())
	}

	if calls := exec.count(); calls != 0 {
		t.Fatalf("missing-binary probe attempted %d subprocess call(s) — LookPath must gate the exec (D-02)", calls)
	}
}

// testKeyringProbeNoDesktopSession is arm (b): the subprocess stderr
// carries the probed Cannot-autolaunch D-Bus class — the coded
// no-desktop-session hint answers, and the stderr text itself never
// reaches the error (coarse-signal classification only).
func testKeyringProbeNoDesktopSession(t *testing.T) {
	probe := interact.NewKeyringProber("secret-tool", probeFixtureCredentials(t), probeEnv(),
		interact.WithKeyringProbeLookPath(probeLookPathAvailable),
		interact.WithKeyringProbeExec(func(
			_ context.Context, _ []string, _ []string,
		) ([]byte, []byte, error) {
			// The probed failure mode (V-EXP 2026-11-22): autolaunch refused.
			return nil, []byte("secret-tool: Cannot autolaunch D-Bus without X11 $DISPLAY"),
				errProbeExitStatus1
		}),
	)

	err := probe(context.Background())

	if !errors.Is(err, interact.ErrKeyringNoDesktopSession) {
		t.Fatalf("autolaunch probe error = %v, want the coded ErrKeyringNoDesktopSession", err)
	}

	if !strings.Contains(err.Error(), "desktop session") {
		t.Fatalf("no-desktop-session error %q must carry the desktop-session hint", err.Error())
	}

	if strings.Contains(err.Error(), "X11 $DISPLAY") {
		t.Fatalf("no-desktop-session error %q carries stderr verbatim — "+
			"classification must use the coarse marker only", err.Error())
	}
}

// testKeyringProbeNoEntry is arm (c): the keyring answers empty (rc=1, no
// output) — the coded log-in-via-desktop hint.
func testKeyringProbeNoEntry(t *testing.T) {
	probe := interact.NewKeyringProber("secret-tool", probeFixtureCredentials(t), probeEnv(),
		interact.WithKeyringProbeLookPath(probeLookPathAvailable),
		interact.WithKeyringProbeExec(func(
			_ context.Context, _ []string, _ []string,
		) ([]byte, []byte, error) {
			// The probed failure mode (V-EXP 2026-11-22): no matching entry.
			return nil, nil, errProbeExitStatus1
		}),
	)

	err := probe(context.Background())

	if !errors.Is(err, interact.ErrKeyringNoEntry) {
		t.Fatalf("empty-answer probe error = %v, want the coded ErrKeyringNoEntry", err)
	}

	if !strings.Contains(err.Error(), "log in via the ZCode desktop") {
		t.Fatalf("no-entry error %q must carry the log-in-via-desktop hint", err.Error())
	}
}

// testKeyringProbeArgvContract is arm (d): the probe builds its lookup by
// calling the LOADER'S LookupCommand (one argv source — never re-derived)
// and runs it under a bounded context (the ≤5s ceiling) with the loader's
// scrubbed env.
func testKeyringProbeArgvContract(t *testing.T) {
	credPath := probeFixtureCredentials(t)
	env := probeEnv()

	loader := interact.NewSecretToolKeyLoader("secret-tool", credPath, env)
	wantArgv, wantEnv, err := loader.LookupCommand()
	if err != nil {
		t.Fatalf("loader LookupCommand: %v", err)
	}

	exec := &recordedProbeExec{}

	probe := interact.NewKeyringProber("secret-tool", credPath, env,
		interact.WithKeyringProbeLookPath(probeLookPathAvailable),
		interact.WithKeyringProbeExec(exec.exec),
	)

	if perr := probe(context.Background()); perr != nil {
		t.Fatalf("probe over an available keyring: %v", perr)
	}

	if calls := exec.count(); calls != 1 {
		t.Fatalf("probe attempted %d subprocess call(s), want exactly one lookup", calls)
	}

	call := exec.calls[0]

	if strings.Join(call.argv, " ") != strings.Join(wantArgv, " ") {
		t.Fatalf("probe argv = %v, want the loader's LookupCommand argv %v (one argv source)", call.argv, wantArgv)
	}

	if strings.Join(call.env, " ") != strings.Join(wantEnv, " ") {
		t.Fatalf("probe env = %v, want the loader's scrubbed env %v", call.env, wantEnv)
	}

	if !call.hasDeadline {
		t.Fatal("probe exec ctx carries no deadline — the lookup must be bounded (AUTH-04)")
	}

	remaining := time.Until(call.deadline)
	if remaining > interact.KeyringProbeTimeout || remaining < interact.KeyringProbeTimeout-time.Second {
		t.Fatalf("probe exec ctx deadline in %v, want the %s bound (±1s)", remaining, interact.KeyringProbeTimeout)
	}
}

// testKeyringProbeUnresolvablePath is arm (e): the credentials path cannot
// resolve (a fresh machine) — the LookupCommand build itself fails, so the
// log-in hint answers with NO subprocess attempted.
func testKeyringProbeUnresolvablePath(t *testing.T) {
	exec := &recordedProbeExec{}

	missing := filepath.Join(t.TempDir(), "missing", "credentials.json")

	probe := interact.NewKeyringProber("secret-tool", missing, probeEnv(),
		interact.WithKeyringProbeLookPath(probeLookPathAvailable),
		interact.WithKeyringProbeExec(exec.exec),
	)

	err := probe(context.Background())

	if !errors.Is(err, interact.ErrKeyringNoEntry) {
		t.Fatalf("unresolvable-path probe error = %v, want the coded ErrKeyringNoEntry", err)
	}

	if !strings.Contains(err.Error(), "log in via the ZCode desktop") {
		t.Fatalf("unresolvable-path error %q must carry the log-in-via-desktop hint", err.Error())
	}

	if calls := exec.count(); calls != 0 {
		t.Fatalf("unresolvable-path probe attempted %d subprocess call(s) — "+
			"the log-in hint must answer before any exec", calls)
	}
}

// testKeyringProbeHungExec is arm (f) — the never-hangs lock (Pitfall 5 /
// A3): a subprocess seam that blocks until its bounded ctx dies must answer
// the coded timeout WELL inside the test's own watchdog bound.
func testKeyringProbeHungExec(t *testing.T) {
	probe := interact.NewKeyringProber("secret-tool", probeFixtureCredentials(t), probeEnv(),
		interact.WithKeyringProbeLookPath(probeLookPathAvailable),
		interact.WithKeyringProbeTimeout(80*time.Millisecond),
		interact.WithKeyringProbeExec(func(
			ctx context.Context, _ []string, _ []string,
		) ([]byte, []byte, error) {
			<-ctx.Done() // hang until the bounded ctx kills the lookup

			return nil, nil, ctx.Err()
		}),
	)

	type outcome struct {
		watchdog bool
	}

	done := make(chan outcome, 1)

	watchdog := time.AfterFunc(3*time.Second, func() { done <- outcome{watchdog: true} })
	defer watchdog.Stop()

	started := time.Now()

	err := probe(context.Background())

	watchdog.Stop()

	elapsed := time.Since(started)

	select {
	case o := <-done:
		if o.watchdog {
			t.Fatal("probe hit the test watchdog — a hung lookup must never hang authenticate (Pitfall 5)")
		}
	default:
	}

	if err == nil || !errors.Is(err, interact.ErrKeyringLookupTimeout) {
		t.Fatalf("hung-lookup probe error = %v, want the coded ErrKeyringLookupTimeout", err)
	}

	if elapsed > 2*time.Second {
		t.Fatalf("hung-lookup probe answered in %v — must return well inside the bound", elapsed)
	}
}
