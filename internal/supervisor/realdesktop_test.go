//go:build realdesktop && linux

// Shared real-tier helper module for the REAL mount-first acceptance
// (VER-02, plan 10-05) — MANUAL-ONLY, never CI (Pitfall 10; the Phase 2
// blocker: CI must never spawn the real desktop). The realdesktop build tag
// excludes this file from default builds, vet, and CI; the
// ZCODE_ACP_REAL_E2E=1 runtime guard skips the body even when someone builds
// with the tag (belt and braces).
//
// The single real-machine positive path is TestRealAppImageMount in
// appimage_realdesktop_test.go: mount a packed AppImage through the
// production mount owner, spawn the real app-server from the mount, complete
// session/create, and tear down with zero leftovers. A live prompt on ZCode
// 4.0.0 is explicitly out of scope (D-14: protocol drift is a separate
// milestone); the gap is documented, not papered over. This file carries only
// the helpers that test shares.
//
// Canonical run command (a human runs this ONCE on a FUSE-capable machine
// with a real packed AppImage):
//
//	ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/
//
// Expected: PASS with the probe rows, selected layout/version/AppImage, mount
// path, helper/app-server PIDs, and teardown facts logged (capture-header
// discipline: record, never assume) and zero orphaned processes afterwards.
// Without the env var AND the tag the test is skipped/absent — CI never
// contacts the desktop.

package supervisor_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Real-desktop manual-tier knobs shared by TestRealAppImageMount.
const (
	// realE2EEnv is the runtime guard: the body never runs without it.
	realE2EEnv = "ZCODE_ACP_REAL_E2E"
	// realOptDir is the package-family root the discovery matrix probes
	// (the deb layout's /opt/ZCode).
	realOptDir = "/opt/ZCode"

	// realShutdownWatchdog bounds the staged shutdown wait in the test's
	// ordered teardown (the supervisor's stdin-grace-groupkill-reap chain).
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

// logObservedInstall records the probe-time install facts (capture-header
// discipline: record the layout and the builtin revision, never assume).
// The packed fields (version, appimage) print when present; a mounted install
// reports the mount root, a pre-mount packed install reports an empty root.
func logObservedInstall(t *testing.T, inst supervisor.Install) {
	t.Helper()

	fields := fmt.Sprintf("layout=%s", inst.Layout)
	if inst.Version != "" {
		fields += " version=" + inst.Version
	}

	fields += fmt.Sprintf(" root=%s binary=%s", inst.Root, inst.Binary)

	if inst.AppImage != "" {
		fields += " appimage=" + inst.AppImage
	}

	revision, revErr := readBuiltinRevision(inst.BuiltinFile)
	if revErr != nil {
		t.Logf("observed install: %s builtinRevision=unknown (%v)", fields, revErr)

		return
	}

	t.Logf("observed install: %s builtinRevision=%s", fields, supervisor.BuiltinRevision(inst.BuiltinFile, revision))
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

// assertCreateRoundTrip runs session/create through the connected real
// desktop: the real server blocks on the create-time
// session/requestRuntimePreferences obligation, which the Conn's router
// answers within its D-05 deadline (the Phase 1 discovery: unanswered creates
// fail -32603). The CALLER owns the Conn and must keep it open until the
// supervisor's staged shutdown returns — closing stdin earlier lets the
// restart loop spawn a second child before the test cancels.
func assertCreateRoundTrip(ctx context.Context, t *testing.T, conn *zcode.Conn) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve workspace dir: %v", err)
	}

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
