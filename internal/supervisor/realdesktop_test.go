//go:build realdesktop

// Real-desktop positive path (BRIDGE-03 final test-map row) — MANUAL-ONLY,
// never CI (Pitfall 10; the Phase 2 blocker: CI must never spawn the real
// desktop). The realdesktop build tag excludes this file from default
// builds, vet, and CI; the ZCODE_ACP_REAL_E2E=1 runtime guard skips the body
// even when someone builds with the tag (belt and braces).
//
// Canonical run command (a human runs this ONCE after the phase's plans
// complete, on a machine WITH a ZCode desktop install — 03-VALIDATION.md
// Manual-Only Verifications row; this machine has ZCode 3.18.2):
//
//	ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop ./internal/supervisor/
//
// Expected: PASS with the observed install layout + builtin revision logged
// at probe time (capture-header discipline: record, never assume) and zero
// orphaned processes afterwards (`pgrep -f zcode.cjs` empty). Without the
// env var AND the tag the test is skipped/absent — CI never contacts the
// desktop.

package supervisor_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Real-desktop manual-tier knobs.
const (
	// realE2EEnv is the runtime guard: the body never runs without it.
	realE2EEnv = "ZCODE_ACP_REAL_E2E"
	// realOptDir is the package-family root the discovery matrix probes
	// (the deb layout's /opt/ZCode).
	realOptDir = "/opt/ZCode"

	// realDesktopDeadline bounds one full positive-path pass (locate →
	// spawn → create → close): Electron startup takes seconds, never minutes.
	realDesktopDeadline = 60 * time.Second
	// realShutdownWatchdog bounds the staged shutdown wait in the test.
	realShutdownWatchdog = 30 * time.Second
)

// childDuplex joins the child's stdout read side and stdin write side into
// the one io.ReadWriteCloser the codec Conn consumes (ARCHITECTURE internal
// boundary: the supervisor owns the pipes and hands the Conn a
// ReadWriteCloser). Close unblocks a blocked Read (os/exec pipes do), which
// is what Conn.Close relies on to join its read loop.
type childDuplex struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (d childDuplex) Read(p []byte) (int, error) {
	return d.reader.Read(p)
}

func (d childDuplex) Write(p []byte) (int, error) {
	return d.writer.Write(p)
}

func (d childDuplex) Close() error {
	return errors.Join(d.writer.Close(), d.reader.Close())
}

// TestRealDesktop runs the REAL positive path on the host machine:
// Probers.Discover per the Phase 1 matrix, spawn through the production
// SpawnArgv/ChildEnv/SpawnChild path, a zcode.Conn carrying session/create
// through the happy-path exchange (the router answers the create-time
// runtime-preferences obligation per D-05), then the staged shutdown with
// the ESRCH-verified zero-orphan check. MANUAL-ONLY (see the file header).
func TestRealDesktop(t *testing.T) {
	if os.Getenv(realE2EEnv) != "1" {
		t.Skip("manual-only: set ZCODE_ACP_REAL_E2E=1 AND build with -tags realdesktop — CI never contacts the desktop")
	}

	ctx, cancel := context.WithTimeout(context.Background(), realDesktopDeadline)
	defer cancel()

	inst := discoverRealInstall(ctx, t)
	logObservedInstall(t, inst)

	child := startRealChild(ctx, t, inst)
	assertCreateRoundTrip(ctx, t, child)

	// Staged shutdown via the supervisor, then the zero-orphan proof.
	cancel()

	assertRealShutdown(t, child)
	assertReaped(t, child)
	assertGone(t, "the real desktop process group", -child.PgID)

	t.Log("no orphaned group members: kill(-pgid, 0) reports ESRCH after the staged shutdown")
}

// discoverRealInstall runs the REAL Probers.Discover on the host machine and
// logs every probe row (the diagnostics ARE the product, install-discovery.md
// :34). A host without an install SKIPS — the positive path needs one.
func discoverRealInstall(ctx context.Context, t *testing.T) supervisor.Install {
	t.Helper()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home for discovery: %v", err)
	}

	inst, probes, err := supervisor.DefaultProbers(home, realOptDir).Discover(ctx)
	if err != nil {
		for _, row := range probes {
			t.Logf("probe %-16s %s -> %s", row.Name, row.Query, row.Result)
		}

		if errors.Is(err, supervisor.ErrNoInstall) {
			t.Skip("no ZCode desktop install found on this host — the manual positive path needs one")
		}

		t.Fatalf("discover: %v", err)
	}

	return inst
}

// logObservedInstall records the probe-time install facts (capture-header
// discipline: record the layout and the builtin revision, never assume).
func logObservedInstall(t *testing.T, inst supervisor.Install) {
	t.Helper()

	revision, revErr := readBuiltinRevision(inst.BuiltinFile)
	if revErr != nil {
		t.Logf("observed install: layout=%s root=%s binary=%s builtinRevision=unknown (%v)",
			inst.Layout, inst.Root, inst.Binary, revErr)

		return
	}

	t.Logf("observed install: layout=%s root=%s binary=%s builtinRevision=%s",
		inst.Layout, inst.Root, inst.Binary, supervisor.BuiltinRevision(inst.BuiltinFile, revision))
}

// readBuiltinRevision reads the recorded revision number out of the active
// zcode-builtin.json (probe-time state — a live desktop may update mid-probe;
// the value is recorded, never assumed).
func readBuiltinRevision(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	var parsed struct {
		Revision int `json:"revision"`
	}

	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, err
	}

	return parsed.Revision, nil
}

// startRealChild spawns the REAL app-server through the production path and
// hands back the child (race-safe delivery from the Start goroutine).
func startRealChild(ctx context.Context, t *testing.T, inst supervisor.Install) *supervisor.Child {
	t.Helper()

	spawnCh := make(chan *supervisor.Child, 1)

	sup := supervisor.NewSupervisor(func(ctx context.Context) (*supervisor.Child, error) {
		child, err := supervisor.SpawnChild(ctx, supervisor.SpawnArgv(inst),
			supervisor.ChildEnv(inst, os.Environ()), supervisor.SpawnOpts{})
		if err == nil {
			spawnCh <- child
		}

		return child, err
	}, supervisor.DefaultPolicy())

	startDone := make(chan error, 1)

	go func() {
		startDone <- sup.Start(ctx)
	}()

	child := waitSpawnedChild(t, spawnCh)

	return child
}

// assertCreateRoundTrip brings up a zcode.Conn over the real pipes and runs
// session/create through the happy-path exchange: the real server blocks on
// the create-time session/requestRuntimePreferences obligation, which the
// Conn's router answers within its D-05 deadline (the Phase 1 discovery:
// unanswered creates fail -32603).
func assertCreateRoundTrip(ctx context.Context, t *testing.T, child *supervisor.Child) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve workspace dir: %v", err)
	}

	//nolint:contextcheck // NewConn owns its base context (ctx only bounds the Call below)
	conn := zcode.NewConn(childDuplex{reader: child.Stdout, writer: child.Stdin})
	defer func() { _ = conn.Close() }() // the staged shutdown owns stdin from here; Close is joined by defer

	result, err := conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{"workspacePath": cwd, "workspaceKey": cwd},
	})
	if err != nil {
		t.Fatalf("session/create through the real desktop: %v", err)
	}

	// The create reply's session id is read through a generic decode: the
	// wire key is the dialect's camelCase session.sessionId (a snake_case
	// struct tag would misname the wire contract).
	var created struct {
		Session map[string]any `json:"session"`
	}

	if err := json.Unmarshal(result, &created); err != nil {
		t.Fatalf("decode session/create result %s: %v", result, err)
	}

	sessionID, _ := created.Session["sessionId"].(string)

	if sessionID == "" {
		t.Fatalf("session/create result carries no session.sessionId: %s", result)
	}

	t.Logf("session/create ok: sessionId=%s (runtime-preferences answered by the router)", sessionID)
}

// assertRealShutdown waits for the supervisor's staged shutdown (driven by
// the ctx cancel) to complete.
func assertRealShutdown(t *testing.T, child *supervisor.Child) {
	t.Helper()

	select {
	case <-child.Done:
	case <-time.After(realShutdownWatchdog):
		t.Fatalf("the staged shutdown did not complete within %v", realShutdownWatchdog)
	}
}
