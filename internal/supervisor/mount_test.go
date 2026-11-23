// Mount lifecycle tests (plan 10-01; MNT-01..03, MNT-06; VER-01): a scripted
// fake AppImage runtime drives the PRODUCTION PrepareMount/Stop path with no
// FUSE. The fake answers only --appimage-mount, creates a .mount_* directory
// under the wrapper's private TMPDIR with the invariant spawn surface, prints
// the mount path, and stays alive — the same line/wait protocol the real
// runtime implements (RESEARCH Pattern 2), so the exec, readiness, validation,
// and teardown code under test is the production code, not a test double.

package supervisor_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Fake-runtime facts: the packed shape carries the version in the filename
// (install-discovery.md :33), and every test timeout is injected in
// milliseconds — no test waits on the production wall-clock defaults.
const (
	fakeAppImageName = "ZCode-4.0.0-linux-x64.AppImage"
	fakeAppImageVer  = "4.0.0"
	fakeMountDirGlob = ".mount_Fake"

	// Fake-runtime variants (separate script bodies, never env-controlled:
	// the helper env is a closed allowlist, RESEARCH Code Examples).
	fakeVariantOK        = "ok"
	fakeVariantBadTree   = "bad-tree"
	fakeVariantStubborn  = "stubborn"
	fakeVariantEarlyExit = "early-exit"

	testAppImageLayout = "appimage"

	testMountReadyTimeout = 5 * time.Second
	testMountStopGrace    = 150 * time.Millisecond
	testMountReapDeadline = 5 * time.Second
	testMountPollInterval = 5 * time.Millisecond
	testMountLossPoll     = 10 * time.Millisecond
)

// testMountOpts is the millisecond-scale MountOpts every test uses.
func testMountOpts() supervisor.MountOpts {
	return supervisor.MountOpts{
		ReadyTimeout: testMountReadyTimeout,
		StopGrace:    testMountStopGrace,
		ReapDeadline: testMountReapDeadline,
		PollInterval: testMountPollInterval,
		LossPoll:     testMountLossPoll,
	}
}

// writeFakeAppImage writes the scripted runtime next to the fixture (the
// documented packed filename so version parsing is exercised) and returns its
// path. The script creates the mount tree under the private TMPDIR the wrapper
// exports — the same relocation the real runtime performs.
func writeFakeAppImage(t *testing.T, dir, variant string) string {
	t.Helper()

	appImage := filepath.Join(dir, fakeAppImageName)
	if err := os.WriteFile(appImage, []byte(fakeAppImageScript(variant)), 0o750); err != nil {
		t.Fatalf("write the fake AppImage runtime: %v", err)
	}

	return appImage
}

// fakeAppImageScript returns the variant's script body.
func fakeAppImageScript(variant string) string {
	switch variant {
	case fakeVariantBadTree:
		// Prints a valid mount line but leaves resources/glm/zcode.cjs out:
		// the tree validation must reject it. The stderr marker proves the
		// tap carries the helper's stderr into the failure diagnostics.
		return `#!/bin/sh
set -eu
[ "${1:-}" = "--appimage-mount" ] || { echo "fake-appimage: unsupported argv: $*" >&2; exit 64; }
echo "fake-appimage: cannot mount (stderr marker)" >&2
base=${TMPDIR:?TMPDIR must be set by the wrapper}
dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
trap 'rm -rf "$dir"; exit 143' TERM
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	case fakeVariantEarlyExit:
		// Writes its diagnostic line and exits immediately: the line must
		// survive in the failure error (the stdout tap is drained to EOF by
		// exec's Wait, never discarded by WaitDelay).
		return `#!/bin/sh
set -eu
[ "${1:-}" = "--appimage-mount" ] || { echo "fake-appimage: unsupported argv: $*" >&2; exit 64; }
echo "fake-appimage: helper exploded before the mount line"
exit 1
`
	case fakeVariantStubborn:
		// Ignores SIGTERM and keeps the tree alive; a detached (setsid)
		// short-lived process removes the tree after the helper is killed —
		// the fake analogue of the real keepalive-pipe unmount.
		return `#!/bin/sh
set -eu
[ "${1:-}" = "--appimage-mount" ] || { echo "fake-appimage: unsupported argv: $*" >&2; exit 64; }
base=${TMPDIR:?TMPDIR must be set by the wrapper}
dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
trap '' TERM
setsid sh -c 'sleep 0.5; rm -rf "$1"' _ "$dir" >/dev/null 2>&1 &
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	default:
		// The happy runtime: cleans its tree on TERM/INT and carries the
		// bundled builtin copy (the QA-02 fresh-machine completion).
		return `#!/bin/sh
set -eu
[ "${1:-}" = "--appimage-mount" ] || { echo "fake-appimage: unsupported argv: $*" >&2; exit 64; }
base=${TMPDIR:?TMPDIR must be set by the wrapper}
dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
trap 'rm -rf "$dir"; exit 143' TERM
trap 'rm -rf "$dir"; exit 130' INT
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
printf '{"schemaVersion":1,"revision":44}\n' > "$dir/resources/config/provider/zcode-builtin.json"
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	}
}

// TestMountHappyPath covers PrepareMount/Install/Stop directly: the private
// base lives under the wrapper temp root, the tree is validated (the bad-tree
// arm proves rejection), Install carries the mounted root with AppImage and
// Version preserved, HelperPID is non-zero before and after Stop, Stop removes
// the mount and the base, and a discovery-provided builtin path is preserved.
func TestMountHappyPath(t *testing.T) {
	t.Run("mount, install, stop", testMountHappyPath)
	t.Run("invalid tree rejected", testMountBadTree)
	t.Run("discovery-builtin preserved", testMountPreservesBuiltin)
}

// testMountHappyPath proves the happy lifecycle end to end.
func testMountHappyPath(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantOK)

	inst := supervisor.Install{AppImage: appImage, Version: fakeAppImageVer, Layout: testAppImageLayout}

	mount, err := supervisor.PrepareMount(context.Background(), inst, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	t.Cleanup(func() { _ = mount.Stop() })

	assertMountBaseLocation(t, mount.Root(), tmpRoot)
	assertMountTreeValidated(t, mount.Root())
	assertMountedInstall(t, mount, appImage)

	pid := mount.HelperPID()
	if pid == 0 {
		t.Fatal("HelperPID() = 0 after a successful PrepareMount")
	}

	if killErr := syscall.Kill(pid, 0); killErr != nil {
		t.Fatalf("the helper %d must be alive before Stop: %v", pid, killErr)
	}

	if stopErr := mount.Stop(); stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}

	if got := mount.HelperPID(); got != pid {
		t.Fatalf("HelperPID() = %d after Stop, want the recorded %d "+
			"(never cleared: ESRCH is asserted, never group 0)", got, pid)
	}

	assertMountPathGone(t, mount.Root(), "the mount root")
	assertMountPathGone(t, filepath.Dir(mount.Root()), "the private base")
	assertProcessGone(t, pid, "the mount helper")
}

// assertMountBaseLocation requires the mount root to live one component under
// a private base created in the wrapper temp root.
func assertMountBaseLocation(t *testing.T, root, tmpRoot string) {
	t.Helper()

	base := filepath.Dir(root)
	if !strings.HasPrefix(base, filepath.Join(tmpRoot, "zcode-acp-mount-")) {
		t.Fatalf("the private base %q is not under the wrapper temp root %q", base, tmpRoot)
	}

	if !strings.HasPrefix(filepath.Base(root), fakeMountDirGlob) {
		t.Fatalf("the mount root %q is not a %s* directory", root, fakeMountDirGlob)
	}
}

// assertMountTreeValidated requires the invariant spawn surface to be present
// in the accepted mount.
func assertMountTreeValidated(t *testing.T, root string) {
	t.Helper()

	for _, rel := range []string{"zcode", filepath.Join("resources", "glm", "zcode.cjs")} {
		if _, statErr := os.Stat(filepath.Join(root, rel)); statErr != nil {
			t.Fatalf("the validated mounted tree misses %s: %v", rel, statErr)
		}
	}
}

// assertMountedInstall requires Install() to carry the mounted root with the
// packed AppImage and its version preserved.
func assertMountedInstall(t *testing.T, mount *supervisor.Mount, appImage string) {
	t.Helper()

	root := mount.Root()
	want := supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json"),
		Layout:      testAppImageLayout,
		AppImage:    appImage,
		Version:     fakeAppImageVer,
	}

	if installed := mount.Install(); installed != want {
		t.Fatalf("mount.Install() mismatch:\n got %+v\nwant %+v", installed, want)
	}
}

// testMountBadTree proves the mounted tree is validated: a mount line that
// resolves but lacks the invariant entrypoint fails loud and leaves nothing
// behind.
func testMountBadTree(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantBadTree)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err == nil {
		_ = mount.Stop()

		t.Fatal("PrepareMount accepted a mount without resources/glm/zcode.cjs")
	}

	if mount != nil {
		t.Fatalf("PrepareMount returned a mount alongside the error: %+v", mount)
	}

	if !strings.Contains(err.Error(), "zcode.cjs") {
		t.Fatalf("the tree error %q must name the missing invariant entrypoint", err)
	}

	entries, readErr := os.ReadDir(tmpRoot)
	if readErr != nil {
		t.Fatalf("read the wrapper temp root: %v", readErr)
	}

	if len(entries) != 0 {
		t.Fatalf("a failed PrepareMount leaked %d entries under %s: %v", len(entries), tmpRoot, entries)
	}
}

// testMountPreservesBuiltin proves a discovery-provided builtin path is
// carried through the mount unchanged (the runtime cache wins; the bundled
// fill only applies when discovery left it empty).
func testMountPreservesBuiltin(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantOK)
	preset := filepath.Join(t.TempDir(), "runtime", "zcode-builtin.json")

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer, BuiltinFile: preset}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	t.Cleanup(func() { _ = mount.Stop() })

	if got := mount.Install().BuiltinFile; got != preset {
		t.Fatalf("BuiltinFile = %q, want the discovery-provided %q preserved", got, preset)
	}

	if err := mount.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// assertMountPathGone requires path to be gone.
func assertMountPathGone(t *testing.T, path, what string) {
	t.Helper()

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("%s still exists at %s (stat error %v)", what, path, err)
	}
}

// assertProcessGone requires pid to answer ESRCH (reaped, not merely killed).
func assertProcessGone(t *testing.T, pid int, what string) {
	t.Helper()

	deadline := time.After(testMountReapDeadline)

	for {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("%s %d still exists after teardown", what, pid)
		case <-time.After(testMountPollInterval):
		}
	}
}

// TestMountStop proves the staged stop against a helper that ignores SIGTERM:
// the escalation to a group SIGKILL happens only after StopGrace, the helper is
// reaped, the tree disappears (the detached cleaner is the fake analogue of
// the keepalive-pipe unmount), the base is removed, and Stop is idempotent.
func TestMountStop(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantStubborn)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	pid := mount.HelperPID()
	if pid == 0 {
		t.Fatal("HelperPID() = 0 after a successful PrepareMount")
	}

	if killErr := syscall.Kill(pid, 0); killErr != nil {
		t.Fatalf("the stubborn helper %d must be alive before Stop: %v", pid, killErr)
	}

	start := time.Now()
	stopErr := mount.Stop()
	elapsed := time.Since(start)

	if stopErr != nil {
		t.Fatalf("Stop: %v (the stubborn helper must be escalated to the group SIGKILL and reaped)", stopErr)
	}

	if elapsed < testMountStopGrace {
		t.Fatalf("Stop returned after %v, want at least the %v grace: a TERM-ignoring helper "+
			"must reach the group SIGKILL escalation", elapsed, testMountStopGrace)
	}

	assertProcessGone(t, pid, "the stubborn mount helper")
	assertMountPathGone(t, mount.Root(), "the mount root")
	assertMountPathGone(t, filepath.Dir(mount.Root()), "the private base")

	if err := mount.Stop(); err != nil {
		t.Fatalf("second Stop = %v, want nil (idempotent)", err)
	}
}

// TestMountSpacedTmpdir proves the whole mount line is the path: a TMPDIR that
// contains a space produces a mount path with a space, and a fields-splitting
// parser would truncate it at the first space.
func TestMountSpacedTmpdir(t *testing.T) {
	root := t.TempDir()
	spaced := filepath.Join(root, "tmp dir with spaces")

	if err := os.MkdirAll(spaced, 0o700); err != nil {
		t.Fatalf("create the spaced TMPDIR: %v", err)
	}

	t.Setenv("TMPDIR", spaced)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantOK)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount with a spaced TMPDIR: %v", err)
	}

	t.Cleanup(func() { _ = mount.Stop() })

	if !strings.Contains(mount.Root(), "tmp dir with spaces") {
		t.Fatalf("mount root %q does not carry the spaced TMPDIR — the line was field-split", mount.Root())
	}

	if err := mount.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

// TestMountRejectBadTmpdir proves a newline-bearing private base is rejected
// before any spawn: the error names TMPDIR and the base, nothing was spawned,
// and the just-created base was removed.
func TestMountRejectBadTmpdir(t *testing.T) {
	root := t.TempDir()
	bad := root + "/bad\ntmpdir"

	if err := os.MkdirAll(bad, 0o700); err != nil {
		t.Fatalf("create the newline-bearing TMPDIR: %v", err)
	}

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantOK)

	t.Setenv("TMPDIR", bad)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())

	if mount != nil {
		t.Fatalf("PrepareMount returned a mount for a newline-bearing TMPDIR: %+v", mount)
	}

	if err == nil {
		t.Fatal("PrepareMount accepted a newline-bearing TMPDIR")
	}

	if !strings.Contains(err.Error(), "TMPDIR") {
		t.Fatalf("the rejection %q must name TMPDIR", err)
	}

	// The base is rendered with %q (a raw newline would forge log lines), so
	// the offending segment appears escaped.
	if !strings.Contains(err.Error(), "bad\\ntmpdir") {
		t.Fatalf("the rejection %q must name the offending base path %q (quoted)", err, bad)
	}

	entries, readErr := os.ReadDir(bad)
	if readErr != nil {
		t.Fatalf("read the rejected TMPDIR: %v", readErr)
	}

	if len(entries) != 0 {
		t.Fatalf("the rejected base was not removed (nothing may be spawned): %v", entries)
	}
}

// TestMountNoExtraction proves a full PrepareMount/Stop cycle writes no
// squashfs-root/zcode-{ver} copy and leaves the AppImage bytes untouched
// (MNT-06).
func TestMountNoExtraction(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantOK)

	beforeSHA := fileSHA256(t, appImage)
	beforeEntries := fixtureEntries(t, fixture)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	if err := mount.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if got := fileSHA256(t, appImage); got != beforeSHA {
		t.Fatalf("the AppImage changed during the mount cycle: sha256 %s, want %s", got, beforeSHA)
	}

	afterEntries := fixtureEntries(t, fixture)
	if !slices.Equal(beforeEntries, afterEntries) {
		t.Fatalf("the mount cycle changed the fixture dir: before %v, after %v", beforeEntries, afterEntries)
	}

	for _, name := range afterEntries {
		if name == "squashfs-root" || strings.HasPrefix(name, "zcode-") {
			t.Fatalf("the mount cycle created an extraction copy %q (MNT-06)", name)
		}
	}
}

// TestMountStderrTap proves SpawnOpts.StderrTap mirrors the helper's stderr
// into the caller's diagnostics (and the capped forward keeps its default
// behavior): the fake writes its marker to stderr before the mount line, and
// the tree-validation failure must carry it.
func TestMountStderrTap(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantBadTree)

	_, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err == nil {
		t.Fatal("PrepareMount accepted the bad tree")
	}

	if !strings.Contains(err.Error(), "cannot mount (stderr marker)") {
		t.Fatalf("the failure %q does not carry the helper's stderr — the tap is not wired", err)
	}
}

// TestMountEarlyExitOutput proves the helper's last stdout line survives: a
// helper that prints its diagnostic and exits immediately still has that line
// in the failure error, because exec's copier drains the tap to EOF and Wait
// joins it (WaitDelay is only the close backstop).
func TestMountEarlyExitOutput(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantEarlyExit)

	_, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err == nil {
		t.Fatal("PrepareMount accepted a helper that never printed a mount line")
	}

	if !strings.Contains(err.Error(), "helper exploded before the mount line") {
		t.Fatalf("the early-exit failure %q lost the line the helper wrote immediately before exiting "+
			"(the stdout tap must be drained to EOF, never discarded by WaitDelay)", err)
	}

	if !strings.Contains(err.Error(), "exited before the mount line") {
		t.Fatalf("the early-exit failure %q must name the readiness step", err)
	}
}

// fileSHA256 is the AppImage-unchanged proof's hash helper.
func fileSHA256(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s for hashing: %v", path, err)
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// fixtureEntries lists and sorts a directory's entry names.
func fixtureEntries(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	slices.Sort(names)

	return names
}
