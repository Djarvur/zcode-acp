//go:build realdesktop && linux

// Mount-first real-machine acceptance (VER-02, MNT-01, MNT-03, MNT-06; D-14)
// — MANUAL-ONLY, never CI. The realdesktop build tag excludes this file from
// default builds, vet, and CI; the ZCODE_ACP_REAL_E2E=1 runtime guard skips
// the body even when someone builds with the tag (belt and braces).
//
// Canonical run command (a human runs this ONCE on a FUSE-capable machine with
// a real packed AppImage):
//
//	ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/
//
// Optional pin (otherwise discovery selects the newest usable install and this
// test skips when that selection is not a packed AppImage):
//
//	ZCODE_ACP_REAL_APPIMAGE=/path/to/ZCode-<ver>-linux-x64.AppImage
//
// SCOPE (D-14): mount the real packed AppImage through the production mount
// owner, spawn the real app-server from the mount through the production
// spawn contract, complete session/create, and tear down with zero leftover
// processes or mounts. A LIVE PROMPT ON ZCODE 4.0.0 IS EXPLICITLY OUT OF
// SCOPE — protocol drift is a separate milestone — so this test stops at
// session/create and documents the gap instead of papering over it. It never
// contacts api.z.ai.
//
// Read-only guarantee (T-10-18/T-10-19): the AppImage is mounted, never
// extracted or patched; sha256 is recorded before and after and asserted
// equal, and the AppImage's parent directory is snapshot and asserted to gain
// no new squashfs-root/zcode-<ver> entry. PRE-EXISTING EXTRACTS ARE ALLOWED
// AND NEVER DELETED: this host already carries
// ~/Applications/zcode-3.18.2/squashfs-root, so a blanket "parent is clean"
// assertion would be a false failure.

package supervisor_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Real mount-tier knobs: the optional AppImage pin, the mount+startup+create
// deadline, and the bounded zero-survivor poll used by the teardown closure.
const (
	// realAppImageEnv optionally pins the packed AppImage under test.
	realAppImageEnv = "ZCODE_ACP_REAL_APPIMAGE"

	// realMountDeadline bounds mount plus Electron startup plus session/create
	// (the field probe: ~23 ms to the mount line, 9.0 s to session/create; the
	// margin covers a cold FUSE/Electron start).
	realMountDeadline = 120 * time.Second

	// realGoneDeadline/realGonePoll drive the non-fatal zero-survivor poll in
	// the teardown closure (t.Errorf, never t.Fatalf — a body failure must not
	// be masked and a cleanup must not abort early).
	realGoneDeadline = 30 * time.Second
	realGonePoll     = 100 * time.Millisecond
)

// TestRealAppImageMount runs the REAL mount-first positive path on the host:
// discovery/override of a packed AppImage, PrepareMount through the production
// mount owner, spawn through the production SpawnArgv/ChildEnv/SpawnChild
// contract, session/create through a zcode.Conn, then the ordered teardown
// (supervisor staged shutdown, exec backstop, mount Stop) with the zero-
// leftover and read-only proofs. MANUAL-ONLY (see the file header).
func TestRealAppImageMount(t *testing.T) {
	if os.Getenv(realE2EEnv) != "1" {
		t.Skip("manual-only: set ZCODE_ACP_REAL_E2E=1 AND build with -tags realdesktop — CI never contacts the desktop")
	}

	ctx, cancel := context.WithTimeout(context.Background(), realMountDeadline)
	defer cancel()

	inst := resolveRealAppImage(ctx, t)
	logObservedInstall(t, inst)
	t.Logf("selected packed install: layout=%s version=%s appimage=%s", inst.Layout, inst.Version, inst.AppImage)

	// Baseline BEFORE the mount: the AppImage bytes and every extraction-shaped
	// entry already residing next to it (pre-existing extracts are allowed).
	hashBefore := fileSHA256(t, inst.AppImage)
	entriesBefore := extractionEntries(t, inst.AppImage)
	t.Logf("appimage baseline: sha256=%s extractionEntries=%v", hashBefore, entriesBefore)

	m, err := supervisor.PrepareMount(ctx, inst, supervisor.DefaultMountOpts())
	if err != nil {
		t.Fatalf("mount the packed AppImage %s (step: PrepareMount): %v", inst.AppImage, err)
	}

	mounted := m.Install()
	t.Logf("appimage mounted: layout=%s version=%s root=%s helperPID=%d",
		mounted.Layout, mounted.Version, m.Root(), m.HelperPID())

	// The production two-context discipline: supCtx drives Supervisor.Start
	// (its staged shutdown), spawnCtx is detached from the supervision cancel
	// and is the exec-level kill backstop, cancelled only after Start returns.
	supCtx, cancelSup := context.WithCancel(context.Background())
	spawnCtx, cancelSpawn := context.WithCancel(context.WithoutCancel(supCtx))

	var (
		child     *supervisor.Child
		startDone chan error
		conn      *zcode.Conn
	)

	// ONE ordered teardown closure, guarded by sync.Once and registered before
	// the first assertion that depends on the spawn, so a body failure between
	// registration and assignment cannot leak the helper, the app-server, or
	// the mount. Every wait and close is nil-guarded.
	var (
		teardownOnce sync.Once
		stopErr      error
	)

	teardown := func() error {
		teardownOnce.Do(func() {
			// Stage 1: cancel the supervision context and wait for the
			// supervisor's staged shutdown (stdin close, grace, group kill, reap).
			cancelSup()

			if startDone != nil {
				select {
				case err := <-startDone:
					if err != nil {
						t.Errorf("the supervisor returned an error on the intentional shutdown: %v", err)
					}
				case <-time.After(realShutdownWatchdog):
					t.Errorf("the supervisor's staged shutdown did not complete within %v", realShutdownWatchdog)
				}
			} else {
				t.Logf("no supervisor start channel (the body failed before the spawn); skipping the staged-shutdown wait")
			}

			// Stage 2: the exec backstop, then release the Conn's read loop only
			// after Start returned — closing stdin earlier would let the
			// supervisor restart a second child before the cancel lands.
			cancelSpawn()

			if conn != nil {
				_ = conn.Close()
			}

			// Stage 3: the mount owner's staged helper teardown.
			stopErr = m.Stop()
			if stopErr != nil {
				t.Errorf("stop the mount for %s: %v", inst.AppImage, stopErr)
			}

			// Zero leftovers, asserted even when the body already failed. The
			// child is nil only when the body failed before the spawn.
			if child != nil {
				requireGroupGone(t, "the real app-server process group", -child.PgID)
			} else {
				t.Logf("no app-server child was spawned before the failure; skipping the app-server ESRCH assertion")
			}

			if helperPID := m.HelperPID(); helperPID > 0 {
				requireGroupGone(t, "the mount helper process group", -helperPID)
			} else {
				t.Errorf("the mount helper PID is unset after a successful PrepareMount — refusing to signal group 0")
			}
		})

		return stopErr
	}
	t.Cleanup(func() { _ = teardown() })

	child, startDone = startMountedChild(t, supCtx, spawnCtx, mounted, m)
	t.Logf("app-server spawned: pgid=%d binary=%s", child.PgID, mounted.Binary)

	//nolint:contextcheck // NewConn owns its base context (ctx only bounds the session/create call)
	conn = zcode.NewConn(childDuplex{reader: child.Stdout, writer: child.Stdin})
	assertCreateRoundTrip(ctx, t, conn)

	// The body's final step invokes the SAME teardown closure (the Once makes
	// the deferred invocation a no-op) and requires a clean Stop. D-14: no
	// prompt or turn is sent after the create.
	if err := teardown(); err != nil {
		t.Fatalf("teardown the mount for %s: %v", inst.AppImage, err)
	}

	assertReaped(t, child)
	assertGone(t, "the real app-server process group", -child.PgID)
	assertMountGone(t, m)

	if hashAfter := fileSHA256(t, inst.AppImage); hashAfter != hashBefore {
		t.Errorf("the AppImage sha256 changed during the mount: before=%s after=%s", hashBefore, hashAfter)
	} else {
		t.Logf("AppImage unchanged: sha256=%s", hashAfter)
	}

	entriesAfter := extractionEntries(t, inst.AppImage)
	assertNoNewExtractionEntries(t, inst.AppImage, entriesBefore, entriesAfter)

	t.Log("real mount acceptance complete: mount -> app-server -> session/create -> zero leftovers " +
		"(no live prompt on 4.0.0 — D-14 scope)")
}

// resolveRealAppImage resolves the packed AppImage under test: the optional
// ZCODE_ACP_REAL_APPIMAGE pin through the shared supervisor.OverrideInstall
// mapper, or production discovery. A host whose discovery does not select a
// packed AppImage skips with every probe row logged (the rows ARE the
// diagnostics; install-discovery.md :34).
func resolveRealAppImage(ctx context.Context, t *testing.T) supervisor.Install {
	t.Helper()

	if pinned := os.Getenv(realAppImageEnv); pinned != "" {
		inst, ok := supervisor.OverrideInstall(pinned)
		if !ok || !inst.RequiresMount() {
			t.Fatalf("%s=%q did not map to a packed AppImage install; the pin must name a packed ZCode*.AppImage",
				realAppImageEnv, pinned)
		}

		t.Logf("appimage pinned: %s -> %s", pinned, inst.AppImage)

		return inst
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("resolve home for discovery: %v", err)
	}

	inst, probes, err := supervisor.DefaultProbers(home, realOptDir).Discover(ctx)
	for _, row := range probes {
		t.Logf("probe %-16s %s -> %s", row.Name, row.Query, row.Result)
	}

	if err != nil {
		if errors.Is(err, supervisor.ErrNoInstall) {
			t.Skip("no ZCode desktop install found on this host — the real mount acceptance needs a packed AppImage")
		}

		t.Fatalf("discover: %v", err)
	}

	if !inst.RequiresMount() {
		t.Skipf("discovery selected the non-packed install layout=%s root=%s; "+
			"the real mount acceptance needs a packed AppImage (pin one with %s)",
			inst.Layout, inst.Root, realAppImageEnv)
	}

	return inst
}

// startMountedChild spawns the REAL app-server through the production path
// under the two-context discipline and returns both the first spawned child
// and the supervisor's completion channel. The channel lets the ordered
// teardown wait for Start to return before closing the Conn — closing stdin
// first would let the restart loop spawn a second child.
func startMountedChild(
	t *testing.T, supCtx, spawnCtx context.Context, inst supervisor.Install, m *supervisor.Mount,
) (*supervisor.Child, chan error) {
	t.Helper()

	spawnCh := make(chan *supervisor.Child, 1)
	startDone := make(chan error, 1)

	sup := supervisor.NewSupervisor(func(_ context.Context) (*supervisor.Child, error) {
		// The mount guard runs before EVERY spawn (production's discipline): a
		// vanished root fails loud instead of respawning against it (MNT-05).
		if guardErr := m.Guard(); guardErr != nil {
			return nil, fmt.Errorf("the appimage mount is not usable: %w", guardErr)
		}

		child, spawnErr := supervisor.SpawnChild(spawnCtx,
			supervisor.SpawnArgv(inst),
			supervisor.ChildEnv(inst, os.Environ()),
			supervisor.SpawnOpts{})
		if spawnErr == nil {
			spawnCh <- child
		}

		return child, spawnErr
	}, supervisor.DefaultPolicy())

	go func() { startDone <- sup.Start(supCtx) }()

	return waitSpawnedChild(t, spawnCh), startDone
}

// extractionEntries lists the extraction-shaped entries in the AppImage's
// parent directory: an entry named squashfs-root or matching zcode-<ver>.
func extractionEntries(t *testing.T, appImage string) []string {
	t.Helper()

	dir := filepath.Dir(appImage)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("list the AppImage parent directory %s: %v", dir, err)
	}

	shaped := make([]string, 0, len(entries))
	for _, entry := range entries {
		if extractionShaped(entry.Name()) {
			shaped = append(shaped, entry.Name())
		}
	}

	sort.Strings(shaped)

	return shaped
}

// extractionShaped reports whether name is an extraction-shaped entry:
// "squashfs-root" or "zcode-<ver>" (case-insensitive, <ver> = dotted digits).
// The packed AppImage filename itself never matches: it carries a
// -linux-<arch>.AppImage suffix after the version.
func extractionShaped(name string) bool {
	lower := strings.ToLower(name)
	if lower == "squashfs-root" {
		return true
	}

	rest, ok := strings.CutPrefix(lower, "zcode-")
	if !ok || rest == "" {
		return false
	}

	parts := strings.Split(rest, ".")
	if len(parts) < 2 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			return false
		}

		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}

	return true
}

// assertNoNewExtractionEntries proves the mount added no extraction copy: every
// baseline entry is still present and no new entry appeared. A pre-existing
// extract is explicitly allowed and must never be deleted (MNT-06).
func assertNoNewExtractionEntries(t *testing.T, appImage string, before, after []string) {
	t.Helper()

	dir := filepath.Dir(appImage)
	beforeSet := make(map[string]struct{}, len(before))
	for _, name := range before {
		beforeSet[name] = struct{}{}
	}

	afterSet := make(map[string]struct{}, len(after))
	for _, name := range after {
		afterSet[name] = struct{}{}
	}

	for _, name := range before {
		if _, ok := afterSet[name]; !ok {
			t.Errorf("pre-existing extraction entry %q under %s disappeared — a pre-existing extract is never deleted",
				name, dir)
		}
	}

	added := 0

	for _, name := range after {
		if _, ok := beforeSet[name]; !ok {
			added++
			t.Errorf("the mount added extraction entry %q under %s — the wrapper never extracts", name, dir)
		}
	}

	t.Logf("extraction check under %s: %d pre-existing entries unchanged, %d added", dir, len(before), added)
}

// assertMountGone proves the mount is gone after Stop: the mount root's zcode
// entry no longer stats (ENOENT/ENOTDIR — the mountpoint may linger) and the
// private base was removed.
func assertMountGone(t *testing.T, m *supervisor.Mount) {
	t.Helper()

	zcodePath := filepath.Join(m.Root(), "zcode")

	if _, err := os.Stat(zcodePath); err == nil || (!errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR)) {
		t.Errorf("the mounted zcode entry still stats after Stop: %s: %v", zcodePath, err)
	}

	base := filepath.Dir(m.Root())

	if _, err := os.Stat(base); err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the private mount base %s still exists after Stop: %v", base, err)
	}

	t.Logf("mount gone: no zcode entry at %s and no private base at %s", zcodePath, base)
}

// requireGroupGone polls kill(pgid, 0) until ESRCH. It is the teardown
// closure's NON-FATAL zero-survivor proof: t.Errorf at the deadline, so a body
// failure is never masked and the remaining cleanup still runs.
func requireGroupGone(t *testing.T, what string, pgid int) {
	t.Helper()

	deadline := time.Now().Add(realGoneDeadline)

	for {
		err := syscall.Kill(pgid, 0)
		if errors.Is(err, syscall.ESRCH) {
			t.Logf("%s: kill(%d, 0) reports ESRCH — zero survivors", what, pgid)

			return
		}

		if time.Now().After(deadline) {
			t.Errorf("%s still alive after the staged teardown: kill(%d, 0) = %v", what, pgid, err)

			return
		}

		time.Sleep(realGonePoll)
	}
}
