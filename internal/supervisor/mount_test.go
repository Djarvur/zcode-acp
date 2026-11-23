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
	"errors"
	"os"
	"path/filepath"
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
	fakeVariantOK       = "ok"
	fakeVariantBadTree  = "bad-tree"
	fakeVariantStubborn = "stubborn"

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
		// the tree validation must reject it.
		return `#!/bin/sh
set -eu
[ "${1:-}" = "--appimage-mount" ] || { echo "fake-appimage: unsupported argv: $*" >&2; exit 64; }
base=${TMPDIR:?TMPDIR must be set by the wrapper}
dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
trap 'rm -rf "$dir"; exit 143' TERM
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
printf '%s\n' "$dir"
while :; do sleep 0.05; done
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

	inst := supervisor.Install{AppImage: appImage, Version: fakeAppImageVer, Layout: "appimage"}

	mount, err := supervisor.PrepareMount(context.Background(), inst, testMountOpts())
	if err != nil {
		t.Fatalf("PrepareMount: %v", err)
	}

	t.Cleanup(func() { _ = mount.Stop() })

	root := mount.Root()
	base := filepath.Dir(root)

	if !strings.HasPrefix(base, filepath.Join(tmpRoot, "zcode-acp-mount-")) {
		t.Fatalf("the private base %q is not under the wrapper temp root %q", base, tmpRoot)
	}

	if filepath.Base(root) == "" || !strings.HasPrefix(filepath.Base(root), fakeMountDirGlob) {
		t.Fatalf("the mount root %q is not a %s* directory", root, fakeMountDirGlob)
	}

	for _, rel := range []string{"zcode", filepath.Join("resources", "glm", "zcode.cjs")} {
		if _, statErr := os.Stat(filepath.Join(root, rel)); statErr != nil {
			t.Fatalf("the validated mounted tree misses %s: %v", rel, statErr)
		}
	}

	installed := mount.Install()
	want := supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json"),
		Layout:      "appimage",
		AppImage:    appImage,
		Version:     fakeAppImageVer,
	}

	if installed != want {
		t.Fatalf("mount.Install() mismatch:\n got %+v\nwant %+v", installed, want)
	}

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
		t.Fatalf("HelperPID() = %d after Stop, want the recorded %d (never cleared: ESRCH is asserted, never group 0)", got, pid)
	}

	assertMountPathGone(t, root, "the mount root")
	assertMountPathGone(t, base, "the private base")
	assertProcessGone(t, pid, "the mount helper")
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
