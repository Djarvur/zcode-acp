//go:build linux

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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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

	// The fixture helper records its own PID here so the external test
	// package can prove the reap without dereferencing a Child handle (the
	// external package cannot see Child.Done, and PrepareMount returns nil on
	// every failure).
	fakeHelperPIDFile = "helper.pid"
	// fakeAppServerPIDFile is the release signal the die-after/vanish
	// variants wait for: the test writes it after PrepareMount returned a
	// validated Mount, so the loss happens on an already-installed mount.
	fakeAppServerPIDFile = "app-server.pid"
	// testZcodeName is the invariant binary name the mounted tree must carry.
	testZcodeName = "zcode"

	// Fake-runtime variants (separate script bodies, never env-controlled:
	// the helper env is a closed allowlist, RESEARCH Code Examples).
	fakeVariantOK          = "ok"
	fakeVariantBadTree     = "bad-tree"
	fakeVariantStubborn    = "stubborn"
	fakeVariantEarlyExit   = "early-exit"
	fakeVariantNoLine      = "no-line"
	fakeVariantBogusLine   = "bogus-line"
	fakeVariantMissingTree = "missing-tree"
	fakeVariantNoFuse      = "no-fuse"
	fakeVariantLateText    = "late-text"
	fakeVariantLeaky       = "leaky"
	fakeVariantDieAfter    = "die-after"
	fakeVariantVanish      = "vanish"

	testAppImageLayout = "appimage"

	testMountReadyTimeout = 5 * time.Second
	testMountStopGrace    = 150 * time.Millisecond
	testMountReapDeadline = 5 * time.Second
	testMountPollInterval = 5 * time.Millisecond
	testMountLossPoll     = 10 * time.Millisecond

	mountCleanerRunFlag      = "-test.run=^TestDetachedMountCleanerHelper$"
	mountCleanerEnv          = "ZCODE_ACP_TEST_DETACHED_MOUNT_CLEANER"
	mountRootEnv             = "ZCODE_ACP_TEST_DETACHED_MOUNT_ROOT"
	testDetachedCleanerDelay = 500 * time.Millisecond
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
// exports — the same relocation the real runtime performs — and records its
// own PID beside the script so the caller can assert the reap.
func writeFakeAppImage(t *testing.T, dir, variant string) string {
	t.Helper()

	appImage := filepath.Join(dir, fakeAppImageName)
	body := strings.ReplaceAll(fakeAppImageScript(variant), "@@PID@@", filepath.Join(dir, fakeHelperPIDFile))
	body = strings.ReplaceAll(body, "@@APP_PID@@", filepath.Join(dir, fakeAppServerPIDFile))

	if err := os.WriteFile(appImage, []byte(body), 0o750); err != nil {
		t.Fatalf("write the fake AppImage runtime: %v", err)
	}

	return appImage
}

// fakeMountPreamble is the shared script prefix: argv check plus the PID
// record (the external test package's reap proof).
const fakeMountPreamble = `#!/bin/sh
set -eu
[ "${1:-}" = "--appimage-mount" ] || { echo "fake-appimage: unsupported argv: $*" >&2; exit 64; }
echo $$ > "@@PID@@"
base=${TMPDIR:?TMPDIR must be set by the wrapper}
`

// fakeTreeBody creates the invariant spawn surface inside a fresh .mount_*
// directory and leaves the directory path in $dir.
const fakeTreeBody = `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
`

// Variant bodies. Each is a static script; writeFakeAppImage substitutes the
// PID tokens at fixture creation.
const (
	// fakeScriptBadTree prints a valid mount line but leaves
	// resources/glm/zcode.cjs out: the tree validation must reject it. The
	// stderr marker proves the tap carries the helper's stderr into the
	// failure diagnostics.
	fakeScriptBadTree = fakeMountPreamble + `echo "fake-appimage: cannot mount (stderr marker)" >&2
dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
trap 'rm -rf "$dir"; exit 143' TERM
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	// fakeScriptEarlyExit exits immediately with the legacy runtime's
	// missing-helper status: the error must name the AppImage, the step, the
	// exit, the observed output, and a remediation.
	fakeScriptEarlyExit = fakeMountPreamble + "exit 127\n"
	// fakeScriptNoLine creates a tree but never prints the mount line:
	// readiness times out (millisecond ReadyTimeout) and teardown must still
	// reap the helper and remove the private base.
	fakeScriptNoLine = fakeMountPreamble + fakeTreeBody + "while :; do sleep 0.05; done\n"
	// fakeScriptBogusLine prints an absolute line outside the private base:
	// rejected immediately as an untrusted path-looking line, naming the
	// observed line.
	fakeScriptBogusLine = fakeMountPreamble + `echo /tmp/zcode-acp-not-ours
while :; do sleep 0.05; done
`
	// fakeScriptMissingTree prints a valid mount directory without the
	// invariant zcode entry: tree validation fails naming the missing path.
	fakeScriptMissingTree = fakeMountPreamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm"
: > "$dir/resources/glm/zcode.cjs"
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	// fakeScriptNoFuse prints the legacy runtime's FUSE failure text to
	// STDOUT (notify(), RESEARCH Pattern 2): the blank line must be ignored
	// and the two classification strings must reach the error with both
	// remediation families.
	fakeScriptNoFuse = fakeMountPreamble + `echo ""
echo "dlopen(): error loading libfuse.so.2"
echo "Cannot mount AppImage, please check your FUSE setup."
exit 127
`
	// fakeScriptLateText writes its final diagnostic line and exits
	// immediately: the stdout tap is drained to EOF, so that exact line is
	// in the failure error.
	fakeScriptLateText = fakeMountPreamble + `echo "fake-appimage: late failure text before exit"
exit 1
`
	// fakeScriptLeaky is a valid tree that ignores SIGTERM and installs no
	// cleanup trap: the tree outlives the helper, so a teardown that cannot
	// prove the mount gone must fail loud and retain the private base.
	fakeScriptLeaky = fakeMountPreamble + fakeTreeBody + `trap '' TERM
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	// fakeScriptDieAfter prints the path, waits for the app-server PID file
	// (the test writes it after PrepareMount returned a validated Mount), then
	// removes the tree and exits: watcher-driven loss on an installed mount.
	fakeScriptDieAfter = fakeMountPreamble + fakeTreeBody + `printf '%s\n' "$dir"
while [ ! -f "@@APP_PID@@" ]; do sleep 0.02; done
rm -rf "$dir"
exit 0
`
	// fakeScriptVanish prints the path, waits for the app-server PID file,
	// removes the tree but stays alive: the poll-driven loss arm.
	fakeScriptVanish = fakeMountPreamble + fakeTreeBody + `printf '%s\n' "$dir"
while [ ! -f "@@APP_PID@@" ]; do sleep 0.02; done
rm -rf "$dir"
while :; do sleep 0.05; done
`
	// fakeScriptStubborn ignores SIGTERM and keeps the tree alive; the test
	// starts a detached Go helper that removes the tree after the helper is
	// killed — the fake analogue of the real keepalive-pipe unmount.
	fakeScriptStubborn = fakeMountPreamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
trap '' TERM
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	// fakeScriptHappy cleans its tree on TERM/INT and carries the bundled
	// builtin copy (the QA-02 fresh-machine completion).
	fakeScriptHappy = fakeMountPreamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
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
)

// fakeAppImageScript returns the variant's script body.
func fakeAppImageScript(variant string) string {
	switch variant {
	case fakeVariantBadTree:
		return fakeScriptBadTree
	case fakeVariantEarlyExit:
		return fakeScriptEarlyExit
	case fakeVariantNoLine:
		return fakeScriptNoLine
	case fakeVariantBogusLine:
		return fakeScriptBogusLine
	case fakeVariantMissingTree:
		return fakeScriptMissingTree
	case fakeVariantNoFuse:
		return fakeScriptNoFuse
	case fakeVariantLateText:
		return fakeScriptLateText
	case fakeVariantLeaky:
		return fakeScriptLeaky
	case fakeVariantDieAfter:
		return fakeScriptDieAfter
	case fakeVariantVanish:
		return fakeScriptVanish
	case fakeVariantStubborn:
		return fakeScriptStubborn
	default:
		return fakeScriptHappy
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

	for _, rel := range []string{testZcodeName, filepath.Join("resources", "glm", "zcode.cjs")} {
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

// TestDetachedMountCleanerHelper is the bounded, detached remover selected by
// TestMountStop. The dedicated marker keeps it inert in the ordinary suite.
func TestDetachedMountCleanerHelper(t *testing.T) {
	if os.Getenv(mountCleanerEnv) != "1" {
		return
	}

	root := os.Getenv(mountRootEnv)
	if root == "" {
		fmt.Fprintf(os.Stderr, "detached mount cleaner: %s is missing\n", mountRootEnv)
		os.Exit(1)
	}

	time.Sleep(testDetachedCleanerDelay)

	if err := os.RemoveAll(root); err != nil {
		fmt.Fprintf(os.Stderr, "detached mount cleaner: remove %q: %v\n", root, err)
		os.Exit(1)
	}
}

// startDetachedMountCleaner launches the bounded remover in its own session and
// registers unconditional PID cleanup before returning it to the test.
func startDetachedMountCleaner(t *testing.T, root string) *exec.Cmd {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testMountReapDeadline)
	t.Cleanup(cancel)

	cleaner := exec.CommandContext(ctx, os.Args[0], mountCleanerRunFlag)
	cleaner.Env = append(os.Environ(), mountCleanerEnv+"=1", mountRootEnv+"="+root)
	cleaner.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cleaner.Stderr = os.Stderr

	if err := cleaner.Start(); err != nil {
		t.Fatalf("start the detached mount cleaner: %v", err)
	}

	cleanerPID := cleaner.Process.Pid
	t.Cleanup(func() {
		if cleaner.ProcessState != nil {
			return
		}

		_ = syscall.Kill(cleanerPID, syscall.SIGKILL)
		_ = cleaner.Wait()
	})

	return cleaner
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

	cleaner := startDetachedMountCleaner(t, mount.Root())
	cleanerPID := cleaner.Process.Pid

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

	if err := cleaner.Wait(); err != nil {
		t.Fatalf("reap the detached mount cleaner: %v", err)
	}

	assertProcessGone(t, cleanerPID, "the detached mount cleaner")

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
	// The harness's own PID record is not part of the mount cycle's output.
	afterEntries = slices.DeleteFunc(afterEntries, func(name string) bool { return name == fakeHelperPIDFile })

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
	appImage := writeFakeAppImage(t, fixture, fakeVariantLateText)

	_, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err == nil {
		t.Fatal("PrepareMount accepted a helper that never printed a mount line")
	}

	if !strings.Contains(err.Error(), "late failure text before exit") {
		t.Fatalf("the early-exit failure %q lost the line the helper wrote immediately before exiting "+
			"(the stdout tap must be drained to EOF, never discarded by WaitDelay)", err)
	}

	if !strings.Contains(err.Error(), "read the mount line") {
		t.Fatalf("the early-exit failure %q must name the readiness step", err)
	}
}

// mountFailureCase is one readiness/validation failure family: the variant,
// the ReadyTimeout override, and the substrings the actionable error must
// carry (the AppImage path and the remediation prefix are asserted for every
// case by TestMountFailures).
type mountFailureCase struct {
	name         string
	variant      string
	readyTimeout time.Duration
	want         []string
}

// Failure-want constants (the actionable-error contract's fixed substrings).
const (
	wantStepReadMountLine = "read the mount line"
	wantStepValidateTree  = "validate the mount tree"
	wantObservedOutput    = "observed output"
	wantRemediation       = "remediation:"
)

// mountFailureCases is the failure-classification matrix.
func mountFailureCases() []mountFailureCase {
	return []mountFailureCase{
		{
			name:    "early-exit",
			variant: fakeVariantEarlyExit,
			want:    []string{wantStepReadMountLine, "exit status 127", wantObservedOutput, wantRemediation},
		},
		{
			name:         "no-line",
			variant:      fakeVariantNoLine,
			readyTimeout: 400 * time.Millisecond,
			want:         []string{wantStepReadMountLine, "no mount line", wantObservedOutput, wantRemediation},
		},
		{
			name:    "bogus-line",
			variant: fakeVariantBogusLine,
			want: []string{
				wantStepReadMountLine, "is not one .mount_* component", "/tmp/zcode-acp-not-ours",
				wantRemediation,
			},
		},
		{
			name:    "missing-tree",
			variant: fakeVariantMissingTree,
			want:    []string{wantStepValidateTree, testZcodeName, wantObservedOutput, wantRemediation},
		},
		{
			name:    "no-fuse",
			variant: fakeVariantNoFuse,
			want: []string{
				wantStepReadMountLine, "dlopen(): error loading libfuse.so.2", "libfuse2t64", "/dev/fuse",
				wantRemediation,
			},
		},
		{
			name:    "late-text",
			variant: fakeVariantLateText,
			want: []string{
				wantStepReadMountLine, "late failure text before exit", wantObservedOutput, wantRemediation,
			},
		},
	}
}

// TestMountFailures is the failure-classification matrix (MNT-04, D-03): every
// arm fails loud with the AppImage, the failing step, the capped observed
// output, and a remediation; the helper is reaped, the private base removed,
// and no extraction copy appears. The readiness-timeout arm runs at the
// supervisor tier with a millisecond ReadyTimeout so CI never waits the 15 s
// production default.
func TestMountFailures(t *testing.T) {
	for _, tc := range mountFailureCases() {
		t.Run(tc.name, func(t *testing.T) {
			tmpRoot := t.TempDir()
			t.Setenv("TMPDIR", tmpRoot)

			fixture := t.TempDir()
			appImage := writeFakeAppImage(t, fixture, tc.variant)

			opts := testMountOpts()
			if tc.readyTimeout > 0 {
				opts.ReadyTimeout = tc.readyTimeout
			}

			mount, err := supervisor.PrepareMount(context.Background(),
				supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, opts)

			if mount != nil {
				_ = mount.Stop()

				t.Fatalf("PrepareMount returned a mount alongside the error: %+v", mount)
			}

			if err == nil {
				t.Fatal("PrepareMount accepted a failing helper")
			}

			if !strings.Contains(err.Error(), appImage) {
				t.Fatalf("the failure %q must name the AppImage %q", err, appImage)
			}

			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("the failure %q must carry %q", err, want)
				}
			}

			assertProcessGone(t, readFixturePID(t, fixture, fakeHelperPIDFile), "the mount helper")
			assertNoMountBase(t, tmpRoot)
			assertNoExtractionCopy(t, fixture)
		})
	}
}

// testFuseSuperblockMagic is the Linux FUSE superblock type the teardown
// presence predicate compares against (0x65735546).
const testFuseSuperblockMagic = 0x65735546

// errUnguardedSpawn marks a guarded spawn closure that was reached with a
// lost mount (the crash-loop respawn the Guard exists to forbid, T-10-08).
var errUnguardedSpawn = errors.New("guarded spawn ran despite a lost mount")

// TestMountLost is the loss-lifecycle matrix (MNT-05, D-05): Lost closes
// exactly once after an already-validated mount is lost, Accessible is false,
// Guard wraps ErrMountLost, and an intentional Stop never masquerades as a
// loss.
func TestMountLost(t *testing.T) {
	t.Run("helper dies after the mount is installed", testMountLostHelperDies)
	t.Run("tree vanishes while the helper lives", testMountLostTreeVanishes)
	t.Run("intentional stop is not a loss", testMountStopIsNotLoss)
}

// testMountLostHelperDies covers the watcher-driven loss: the helper removes
// the tree and exits (die-after) only after the app-server PID release, so
// PrepareMount returned a validated Mount first.
func testMountLostHelperDies(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantDieAfter)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	t.Cleanup(func() { _ = mount.Stop() })

	releaseMountLoss(t, fixture)
	waitMountLost(t, mount)

	if mount.Accessible() {
		t.Fatal("Accessible() = true after the mount tree was removed")
	}

	guardErr := mount.Guard()
	if !errors.Is(guardErr, supervisor.ErrMountLost) {
		t.Fatalf("Guard() = %v, want a wrap of ErrMountLost", guardErr)
	}

	if !strings.Contains(guardErr.Error(), appImage) {
		t.Fatalf("the guard error %q must name the AppImage", guardErr)
	}

	if err := mount.Stop(); err != nil {
		t.Fatalf("Stop after loss: %v", err)
	}

	assertMountPathGone(t, mount.Root(), "the lost mount root")
	assertNoMountBase(t, tmpRoot)
}

// testMountLostTreeVanishes covers the poll-driven loss: the helper stays
// alive but removes the tree (vanish).
func testMountLostTreeVanishes(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantVanish)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	t.Cleanup(func() { _ = mount.Stop() })

	releaseMountLoss(t, fixture)
	waitMountLost(t, mount)

	if mount.Accessible() {
		t.Fatal("Accessible() = true after the tree vanished under a live helper")
	}

	if err := mount.Stop(); err != nil {
		t.Fatalf("Stop after loss: %v", err)
	}

	assertProcessGone(t, readFixturePID(t, fixture, fakeHelperPIDFile), "the mount helper")
	assertNoMountBase(t, tmpRoot)
}

// testMountStopIsNotLoss proves the stopping flag: an intentional teardown
// closes the helper (the happy fixture removes its tree on TERM) but must
// never close Lost — a teardown is not a mid-session loss (D-05).
func testMountStopIsNotLoss(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantOK)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	if err := mount.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	select {
	case <-mount.Lost():
		t.Fatal("an intentional Stop closed Lost (the teardown masqueraded as a mid-session loss)")
	case <-time.After(200 * time.Millisecond):
	}
}

// TestMountGuard proves the guarded spawn seam (T-10-08): after the mount is
// lost, Guard refuses the spawn and the supervisor's crash-loop seam runs
// exactly once — never a respawn against a vanished root.
func TestMountGuard(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantDieAfter)

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	t.Cleanup(func() { _ = mount.Stop() })

	releaseMountLoss(t, fixture)
	waitMountLost(t, mount)

	spawnCount := 0

	sup := supervisor.NewSupervisor(func(context.Context) (*supervisor.Child, error) {
		spawnCount++

		if guardErr := mount.Guard(); guardErr != nil {
			return nil, guardErr
		}

		return nil, errUnguardedSpawn
	}, supervisor.Policy{
		CrashLimit:  1,
		CrashWindow: 100 * time.Millisecond,
		BaseBackoff: 10 * time.Millisecond,
		Grace:       50 * time.Millisecond,
	})

	startErr := sup.Start(context.Background())
	if !errors.Is(startErr, supervisor.ErrMountLost) {
		t.Fatalf("Start() = %v, want a wrap of ErrMountLost", startErr)
	}

	if spawnCount != 1 {
		t.Fatalf("the spawn seam ran %d times after the loss, want exactly 1 (no respawn)", spawnCount)
	}
}

// waitMountLost waits for the Lost channel to close.
func waitMountLost(t *testing.T, mount *supervisor.Mount) {
	t.Helper()

	select {
	case <-mount.Lost():
	case <-time.After(testMountReapDeadline):
		t.Fatal("Lost never closed after the mount was lost")
	}
}

// releaseMountLoss writes the release file the die-after/vanish fixtures wait
// for (the test owns the app-server PID file at the supervisor tier).
func releaseMountLoss(t *testing.T, fixture string) {
	t.Helper()

	path := filepath.Join(fixture, fakeAppServerPIDFile)
	if err := os.WriteFile(path, []byte("1\n"), 0o600); err != nil {
		t.Fatalf("write the loss release file %s: %v", path, err)
	}
}

// TestMountTeardownFailure proves a live mount is never traversed by
// RemoveAll: with the Statfs seam reporting the FUSE magic while the tree's
// zcode entry is absent, the mount never disappears, Stop fails at the
// deadline naming the mount and the manual-unmount remediation, the private
// base is retained with its .mount_* child, and the helper is still reaped.
func TestMountTeardownFailure(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	fixture := t.TempDir()
	appImage := writeFakeAppImage(t, fixture, fakeVariantLeaky)

	opts := testMountOpts()
	opts.ReapDeadline = 300 * time.Millisecond
	opts.Statfs = func(path string) (int64, error) {
		if _, err := os.Stat(path); err != nil {
			return 0, err
		}

		return testFuseSuperblockMagic, nil
	}

	mount, err := supervisor.PrepareMount(context.Background(),
		supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, opts)
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	root := mount.Root()
	base := filepath.Dir(root)

	t.Cleanup(func() {
		_ = os.RemoveAll(base) // the retained FUSE-magic base, removed after asserts
	})

	// The tree that failed validation has no zcode; the FUSE predicate is the
	// fallback that must keep Stop from traversing the live mount.
	if err := os.Remove(filepath.Join(root, "zcode")); err != nil {
		t.Fatalf("remove the fixture zcode: %v", err)
	}

	stopErr := mount.Stop()
	if stopErr == nil {
		t.Fatal("Stop accepted a still-present FUSE mount")
	}

	for _, want := range []string{"still present", root, "fusermount -u", base} {
		if !strings.Contains(stopErr.Error(), want) {
			t.Fatalf("the teardown error %q must carry %q", stopErr, want)
		}
	}

	assertRetainedBase(t, base)
	assertProcessGone(t, readFixturePID(t, fixture, fakeHelperPIDFile), "the leaky mount helper")
}

// assertRetainedBase requires the private base and its .mount_* child to
// survive, with the child's tree untouched (RemoveAll never traversed it).
func assertRetainedBase(t *testing.T, base string) {
	t.Helper()

	entries, readErr := os.ReadDir(base)
	if readErr != nil {
		t.Fatalf("the private base %s must be retained: %v", base, readErr)
	}

	foundMount := false

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), fakeMountDirGlob) {
			continue
		}

		foundMount = true

		child := filepath.Join(base, entry.Name())
		if _, statErr := os.Stat(filepath.Join(child, "resources", "glm", "zcode.cjs")); statErr != nil {
			t.Fatalf("RemoveAll traversed the retained mount %s: %v", child, statErr)
		}
	}

	if !foundMount {
		t.Fatalf("the retained private base %s lost its %s* child", base, fakeMountDirGlob)
	}
}

// readFixturePID reads a PID the fixture helper recorded.
func readFixturePID(t *testing.T, fixture, name string) int {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(fixture, name))
	if err != nil {
		t.Fatalf("read the %s record: %v", name, err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("the %s record %q is not a pid: %v", name, string(data), err)
	}

	return pid
}

// assertNoMountBase requires no zcode-acp-mount-* base to remain under root.
func assertNoMountBase(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read the wrapper temp root %s: %v", root, err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "zcode-acp-mount-") {
			t.Fatalf("the failure path left the private base %q behind", filepath.Join(root, entry.Name()))
		}
	}
}

// assertNoExtractionCopy requires no squashfs-root/zcode-<ver> copy under dir
// (MNT-06: the failure paths never fall back to extraction).
func assertNoExtractionCopy(t *testing.T, dir string) {
	t.Helper()

	for _, name := range fixtureEntries(t, dir) {
		if name == "squashfs-root" || strings.HasPrefix(name, "zcode-") {
			t.Fatalf("a failure path created an extraction copy %q in %s (MNT-06)", name, dir)
		}
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
