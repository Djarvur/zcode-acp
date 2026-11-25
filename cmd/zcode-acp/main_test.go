package main

// TestAgentBinaryEndToEnd (plan 04-05): the BUILT zcode-acp binary carries
// the full minimal chat loop over REAL stdio — exec pipes, not io.Pipe —
// proven in CI without a desktop bundle (T-3-13's front-leg twin):
//
//	an SDK ACP client spawns the binary with --app-server pointing at the
//	fake-scenario install shape the discovery Override seam accepts (the
//	production discovery order runs; the override wins step 1)
//	  → initialize (agentInfo.version = the binary's version var, D-02)
//	  → session/new (the model selection rides --model/--reasoning-level;
//	    the setModel payload is asserted on the FAKE's own transcript)
//	  → loop 1: streamed prompt → end_turn with the corpus deltas
//	  → loop 2: prompt → cancel mid-turn (synced on the streamed delta)
//	    → cancelled exactly once as a SUCCESS
//	  → client disconnect → staged shutdown → clean exit 0
//
// Both binaries are built once per test run into the run's temp dir (the
// 03-09 precedent: one build per parent-test invocation — a process-global
// sync.Once cache would point at a deleted TempDir under -count repeats).

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Ride deadlines. The builds dominate the wall clock; the protocol exchanges
// and the staged shutdown complete in seconds.
const (
	e2eBuildDeadline  = 180 * time.Second
	e2eCycleDeadline  = 60 * time.Second
	e2eLoopWait       = 15 * time.Second
	e2eUpdateWait     = 10 * time.Second
	e2eExitWatchdog   = 15 * time.Second
	e2eTranscriptWait = 10 * time.Second

	// corpus row: the two-loop scenario (loop 1 completes end_turn; loop 2
	// stays active until the stop's cancelled patch) — the SAME fixture the
	// golden corpus pins (path relative to this test's dir).
	e2eScenario = "../../internal/zcode/fake/testdata/scenarios/complete-then-cancel.json"

	// The responder leg's scenario (path relative to this test's dir): the
	// create fires ONE interaction reverse mid-create whose expect
	// semantics the built binary must satisfy. interaction/requestUserInput
	// is the deliberate discriminator — the conn's built-in router answers
	// it -32601 (an expect mismatch or an unanswered timeout notes the
	// transcript), while the wired D-01 responder answers the schema-valid
	// decline. Both binaries ride the build-once seam; no second build path.
	e2eResponderScenario = "testdata/responder-e2e.json"

	// The model selection the --model/--reasoning-level flags carry; the
	// fake-transcript assert locks the setModel payload verbatim.
	e2eProvider = "zai-api"
	e2eModel    = "GLM-5.3-Flash"
	e2eLevel    = "high"

	// wiringVersionSegment is the version-looking segment the
	// patClientVersion best-effort dig finds in the fixture install tree.
	wiringVersionSegment = "3.18.2"
)

// wiringFixtureInstall is the explicit fixture install whose tree carries a
// version-looking directory segment (the unpacked layout shape).
func wiringFixtureInstall() supervisor.Install {
	base := "/tmp/wiring-fixture/zcode-" + wiringVersionSegment + "/unpacked"

	return supervisor.Install{
		Root:        base,
		Binary:      base + "/zcode",
		Entrypoint:  base + "/resources/glm/zcode.cjs",
		BuiltinFile: base + "/zcode-builtin.json",
		Layout:      "explicit",
	}
}

// TestWiringCodingPlanSeams is the plan 06-07 cmd wiring vector: the
// production machinery constructs LAZILY (no keyring/mint/exec I/O at
// construction — D-13), buildFront wires all four AuthConfig seams, and the
// responder serves runtime-headers through the mode-aware source (the
// unauthenticated mode delegates to the static arm's unconfigured posture;
// the coding-plan arm's behavior is the corpus row's proof).
func TestWiringCodingPlanSeams(t *testing.T) {
	t.Run("machinery constructs lazily with every seam present", testWiringMachineryLazily)

	t.Run("patClientVersion digs the install tree best-effort with the documented fallback", testWiringClientVersion)

	t.Run("buildFront wires the coding-plan seams into AuthConfig", testWiringBuildFrontSeams)

	t.Run("wireResponder serves runtime-headers through the mode-aware source", testWiringModeAwareServing)
}

// Transcript-parsing sentinels (err113): the poll treats any of them as
// "not ready yet".
var (
	errTranscriptRecord = errors.New("transcript record without a direction tag")
	errTranscriptEmpty  = errors.New("transcript file is empty")
)

// TestAgentBinaryEndToEnd runs the built binary through the full loop.
func TestAgentBinaryEndToEnd(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("version flag unchanged", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), e2eUpdateWait)
		defer cancel()

		out, err := exec.CommandContext(ctx, agentBin, ompFlagVersion).Output()
		if err != nil {
			t.Fatalf("--version: %v", err)
		}

		if got := strings.TrimSpace(string(out)); got != "zcode-acp "+version {
			t.Fatalf("--version = %q, want %q", got, "zcode-acp "+version)
		}
	})

	t.Run("two prompt loops over real stdio", func(t *testing.T) {
		e2eRunTwoLoopRide(t, agentBin, fakeBin)
	})

	t.Run("wired responder answers a scripted reverse", func(t *testing.T) {
		e2eRunResponderLeg(t, agentBin, fakeBin)
	})
}

// e2eBuildBinary builds one package's binary once per parent-test invocation
// into the run's temp dir (the built-binary fallback — never an in-process
// substitute and never the desktop bundle at the E2E tier).
func e2eBuildBinary(t *testing.T, pkg, name string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), e2eBuildDeadline)
	defer cancel()

	binPath := filepath.Join(t.TempDir(), name)

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, pkg)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("go build %s: %v", pkg, err)
	}

	return binPath
}

// e2eRunTwoLoopRide drives the spawned binary through both prompt loops and
// the clean staged shutdown, then asserts the fake's transcript.
func e2eRunTwoLoopRide(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(e2eScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eScenario, err)
	}

	transcriptPath := filepath.Join(t.TempDir(), "fake-transcript.txt")
	installRoot := t.TempDir()
	e2eWriteFakeInstallRoot(t, installRoot, fakeBin, scenarioAbs, transcriptPath)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin,
		ompFlagAppServer, installRoot,
		ompFlagModel, e2eProvider+"/"+e2eModel,
		ompFlagLevel, e2eLevel)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // own group: the cleanup can take the whole tree down

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	e2eRideTwoLoops(rideCtx, t, wrapper, stderrBuf, nil)
	e2eAssertFakeTranscript(t, transcriptPath)
}

// e2eRideTwoLoops starts wrapper (its stderr captured into stderrBuf), drives
// initialize, two session/new calls, and both prompt loops through the ACP SDK
// client over the wrapper's exec pipes, then closes stdin for the clean staged
// shutdown and requires exit 0. onStarted, when set, runs after Start and
// before the first frame — the mount e2e's window to observe the acquisition.
func e2eRideTwoLoops(
	ctx context.Context, t *testing.T, wrapper *exec.Cmd, stderrBuf *e2eSyncBuffer, onStarted func(),
) {
	t.Helper()

	stdin, stdout := e2eStartWrapper(t, wrapper, stderrBuf)

	if onStarted != nil {
		onStarted()
	}

	// The test client rides the wrapper's exec pipes — the production
	// transport (stdout-is-protocol is proven by construction: any stray
	// stdout byte would desync this parser).
	updates := &e2eCollector{}
	client := acp.NewClientSideConnection(updates, stdin, stdout)

	e2eInitialize(ctx, t, client)

	sid1 := e2eNewSession(ctx, t, client)
	e2eLoopOne(ctx, t, client, sid1)

	sid2 := e2eNewSession(ctx, t, client)
	e2eLoopTwo(ctx, t, client, updates, sid2)

	// Clean shutdown: the client side closes stdin — the wrapper's SDK conn
	// Done fires, the staged shutdown takes the fake down, exit 0.
	if err := stdin.Close(); err != nil {
		t.Logf("close wrapper stdin: %v", err)
	}

	e2eWaitExit(t, wrapper)
}

// e2eStartWrapper wires the wrapper's stdin/stdout pipes, starts it (own
// process group), and registers the group cleanup; the pipes are returned for
// the SDK client.
func e2eStartWrapper(t *testing.T, wrapper *exec.Cmd, stderrBuf *e2eSyncBuffer) (io.WriteCloser, io.ReadCloser) {
	t.Helper()

	stdin, err := wrapper.StdinPipe()
	if err != nil {
		t.Fatalf("wrapper stdin pipe: %v", err)
	}

	stdout, err := wrapper.StdoutPipe()
	if err != nil {
		t.Fatalf("wrapper stdout pipe: %v", err)
	}

	if err := wrapper.Start(); err != nil {
		t.Fatalf("spawn %s: %v", wrapper.Path, err)
	}

	t.Cleanup(e2eKillGroup(t, wrapper, stderrBuf))

	return stdin, stdout
}

// Fake packed-AppImage fixture constants (the documented packed shape,
// install-discovery.md :33 — version parsing is exercised by the name).
const (
	e2eFakeAppImageName = "ZCode-4.0.0-linux-x64.AppImage"
	e2eFakeMountRecord  = "mount-path.txt"
	e2eFakeHelperPID    = "helper.pid"
	e2eFakeAppServerPID = "app-server.pid"
	e2eFakeTermRecord   = "term-record.txt"
	e2eFakeLeakPID      = "leak.pid"

	e2eDetachedLeakLauncherEnv = "ZCODE_ACP_TEST_DETACHED_LEAK_LAUNCHER"
	e2eDetachedLeakHelperEnv   = "ZCODE_ACP_TEST_DETACHED_LEAK_HELPER"
	e2eDetachedLeakPIDEnv      = "ZCODE_ACP_TEST_DETACHED_LEAK_PID"
	e2eDetachedLeakLifetime    = 30 * time.Second

	// e2eMountRecordWait bounds the wait for the helper's mount line; a
	// wrapper that treats the AppImage as the app-server binary never mounts
	// (the RED shape this test pins).
	e2eMountRecordWait = 10 * time.Second
	// e2ePIDGoneWait bounds the ESRCH poll; the wrapper reaps both children
	// before it exits, so the first sample normally succeeds.
	e2ePIDGoneWait = 5 * time.Second
	// e2eFixturePoll is the fixture poll interval.
	e2eFixturePoll = 20 * time.Millisecond
	// e2eSignalExitWatchdog bounds the signal/teardown-failure exits: the
	// production MountOpts grace (5 s) plus the mount-gone deadline (10 s)
	// dominate on the teardown-failure arm.
	e2eSignalExitWatchdog = 45 * time.Second
	// e2eMountLossWatchdog bounds the mount-loss exit.
	e2eMountLossWatchdog = 20 * time.Second
)

// TestAppImageMountEndToEnd (plan 10-01) proves the mount-first acquisition
// end to end with no FUSE: the built wrapper receives a scripted fake
// .AppImage through --app-server, mounts it through the production
// --appimage-mount path, validates the mounted tree, serves ACP from it, and
// tears the mount down after the app-server. RED before the change: the
// AppImage is treated as the app-server binary, so no mount is ever created.
func TestAppImageMountEndToEnd(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	scenarioAbs, err := filepath.Abs(e2eScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eScenario, err)
	}

	fixture := t.TempDir()
	transcriptPath := filepath.Join(fixture, "fake-transcript.txt")
	appImage := e2eWriteFakeAppImage(t, fixture, fakeBin, scenarioAbs, transcriptPath)
	appImageSHA := e2eFileSHA256(t, appImage)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin,
		ompFlagAppServer, appImage,
		ompFlagModel, e2eProvider+"/"+e2eModel,
		ompFlagLevel, e2eLevel)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	t.Cleanup(e2eCleanupMountFixture(t, fixture))

	mountPath := ""
	onStarted := func() { mountPath = e2eWaitMountRecord(t, fixture, stderrBuf) }

	e2eRideTwoLoops(rideCtx, t, wrapper, stderrBuf, onStarted)
	e2eAssertFakeTranscript(t, transcriptPath)

	// Ordered teardown: the helper's TERM trap records whether the app-server
	// was still alive — it must have been gone (the supervisor's staged
	// shutdown reaped it before Mount.Stop signalled the helper).
	e2eAssertOrderedTeardown(t, fixture)

	// The mount and the private base it lived in are gone.
	e2eAssertGonePath(t, mountPath, "the managed mount")
	e2eAssertGonePath(t, filepath.Dir(mountPath), "the private mount base")

	// Both supervised children were reaped — never an orphan.
	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeHelperPID), "mount helper")
	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeAppServerPID), "app-server")

	// MNT-06: the AppImage bytes are untouched and no extraction copy exists.
	if got := e2eFileSHA256(t, appImage); got != appImageSHA {
		t.Fatalf("the AppImage changed during the run: sha256 %s, want %s", got, appImageSHA)
	}
	e2eAssertNoExtraction(t, fixture)
}

// TestAppImageOverride (plan 10-04) closes the real user surface around the
// packed override: the env-only form mounts and serves, the flag still wins
// over the env, a bad packed value fails loud at the mount step instead of
// falling through to discovery, and a directory override keeps its tree
// diagnostics.
func TestAppImageOverride(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	scenarioAbs, err := filepath.Abs(e2eScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eScenario, err)
	}

	t.Run("env-only packed override mounts and serves", func(t *testing.T) {
		e2eRunEnvPackedOverride(t, agentBin, fakeBin, scenarioAbs)
	})

	t.Run("flag beats a broken env value", func(t *testing.T) {
		e2eRunFlagBeatsEnvOverride(t, agentBin, fakeBin, scenarioAbs)
	})

	t.Run("dangling packed override fails loud", func(t *testing.T) {
		e2eRunDanglingOverride(t, agentBin)
	})

	t.Run("directory override keeps the tree shape", func(t *testing.T) {
		e2eRunDirectoryOverride(t, agentBin, fakeBin, scenarioAbs)
	})
}

// e2eWriteOverrideFixture writes the fake AppImage and its ride transcript in
// a fresh fixture dir, registers the failed-run cleanup, and returns the
// AppImage path.
func e2eWriteOverrideFixture(t *testing.T, fakeBin, scenarioAbs string) string {
	t.Helper()

	fixture := t.TempDir()
	transcript := filepath.Join(fixture, "fake-transcript.txt")
	appImage := e2eWriteFakeAppImage(t, fixture, fakeBin, scenarioAbs, transcript)
	t.Cleanup(e2eCleanupMountFixture(t, fixture))

	return appImage
}

// e2eRunEnvPackedOverride proves the env-only configuration path: no flag, a
// single ZCODE_ACP_APP_SERVER value, a mounted backend serving ACP.
func e2eRunEnvPackedOverride(t *testing.T, agentBin, fakeBin, scenarioAbs string) {
	t.Helper()

	appImage := e2eWriteOverrideFixture(t, fakeBin, scenarioAbs)

	stderr := e2eRideOverride(t, agentBin, nil, e2eOverrideEnv("ZCODE_ACP_APP_SERVER="+appImage))

	e2eAssertPackedSelection(t, stderr, appImage)
}

// e2eRunFlagBeatsEnvOverride proves flag-over-env precedence: the env names a
// missing AppImage and the flag names the working fixture, so a reversed
// precedence would fail the mount before any session exists.
func e2eRunFlagBeatsEnvOverride(t *testing.T, agentBin, fakeBin, scenarioAbs string) {
	t.Helper()

	appImage := e2eWriteOverrideFixture(t, fakeBin, scenarioAbs)
	broken := filepath.Join(t.TempDir(), "ZCode-9.9.9-linux-x64.AppImage")

	stderr := e2eRideOverride(t, agentBin,
		[]string{ompFlagAppServer, appImage},
		e2eOverrideEnv("ZCODE_ACP_APP_SERVER="+broken))

	e2eAssertPackedSelection(t, stderr, appImage)
}

// e2eRunDanglingOverride proves the fail-loud contract: a missing packed
// override exits 1 naming the path and the mount step, never falls through to
// discovery, starts no session, and leaves no extraction copy.
func e2eRunDanglingOverride(t *testing.T, agentBin string) {
	t.Helper()

	dir := t.TempDir()
	missing := filepath.Join(dir, "ZCode-9.9.9-linux-x64.AppImage")

	ctx, cancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer cancel()

	wrapper := exec.CommandContext(ctx, agentBin, ompFlagAppServer, missing)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	wrapper.Env = append(os.Environ(), e2eOverrideEnv("")...)

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	err := wrapper.Run()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		stderrBuf.dump(t)

		t.Fatalf("wrapper exit = %v, want exit code 1 for a dangling packed override", err)
	}

	stderr := stderrBuf.String()

	for _, want := range []string{missing, "mount the app-server AppImage", "spawn the helper"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("the dangling-override stderr must carry %q: %s", want, stderr)
		}
	}
	if strings.Contains(stderr, "no ZCode desktop install found") ||
		strings.Contains(stderr, "serving ACP v1 over stdio") {
		t.Fatalf("a bad packed override must fail at the mount step, never fall through to "+
			"discovery or a session: %s", stderr)
	}

	e2eAssertNoExtraction(t, dir)
}

// e2eRunDirectoryOverride proves the directory form is unchanged: the tree
// resolves through the explicit layout, reports its root, mounts nothing, and
// still serves the session.
func e2eRunDirectoryOverride(t *testing.T, agentBin, fakeBin, scenarioAbs string) {
	t.Helper()

	transcript := filepath.Join(t.TempDir(), "fake-transcript.txt")
	installRoot := t.TempDir()
	e2eWriteFakeInstallRoot(t, installRoot, fakeBin, scenarioAbs, transcript)

	stderr := e2eRideOverride(t, agentBin,
		[]string{ompFlagAppServer, installRoot},
		e2eOverrideEnv(""))

	if !strings.Contains(stderr, "layout=explicit") || !strings.Contains(stderr, "root="+installRoot) {
		t.Fatalf("a directory override must keep its tree diagnostics (layout/root): %s", stderr)
	}
	if strings.Contains(stderr, "appimage backend mounted") {
		t.Fatalf("a directory override must not mount anything: %s", stderr)
	}
}

// e2eOverrideEnv blanks every wrapper env fallback and appends extra, so a
// subtest's environment is exactly what it configured.
func e2eOverrideEnv(extra string) []string {
	env := []string{"ZCODE_ACP_APP_SERVER=", "ZCODE_ACP_MODEL=", "ZCODE_ACP_REASONING_LEVEL=", "ZCODE_ACP_API_KEY="}
	if extra != "" {
		env = append(env, extra)
	}

	return env
}

// e2eRideOverride starts the built wrapper with argv over the caller's env,
// drives initialize and session/new, closes stdin for the clean staged
// shutdown, and returns the captured stderr.
func e2eRideOverride(t *testing.T, agentBin string, argv, extraEnv []string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer cancel()

	wrapper := exec.CommandContext(ctx, agentBin, argv...)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	wrapper.Env = append(os.Environ(), extraEnv...)

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	stdin, stdout := e2eStartWrapper(t, wrapper, stderrBuf)

	client := acp.NewClientSideConnection(&e2eCollector{}, stdin, stdout)
	e2eInitialize(ctx, t, client)
	e2eNewSession(ctx, t, client)

	if err := stdin.Close(); err != nil {
		t.Logf("close wrapper stdin: %v", err)
	}

	e2eWaitExit(t, wrapper)

	return stderrBuf.String()
}

// e2eCanonicalPath matches production's EvalSymlinks-resolved diagnostics.
// macOS exposes t.TempDir under /var/folders while its real path is
// /private/var/folders; missing fixture paths retain their absolute spelling.
func e2eCanonicalPath(t *testing.T, path string) string {
	t.Helper()

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs %s: %v", path, err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}

	return resolved
}

// e2eAssertPackedSelection asserts the startup diagnostics name the packed
// layout, its version, and the resolved AppImage path.
func e2eAssertPackedSelection(t *testing.T, stderr, appImage string) {
	t.Helper()

	for _, want := range []string{
		"layout=appimage",
		"version=" + mountedFixtureVersion,
		"appimage=" + e2eCanonicalPath(t, appImage),
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("the packed-override startup diagnostics must carry %q: %s", want, stderr)
		}
	}
}

// Fast failure fixture variants for the binary tier (MNT-04): each fails
// before any ACP session exists, so the test can assert exit code, stderr,
// and the absence of a mount/base without driving a ride.
const (
	e2eFailureEarlyExit   = "early-exit"
	e2eFailureBogusLine   = "bogus-line"
	e2eFailureMissingTree = "missing-tree"
	e2eFailureNoFuse      = "no-fuse"
	e2eFailureNoLine      = "no-line"
	e2eFailureIgnoreTerm  = "ignore-term"
)

// Binary-tier failure-want constants.
const (
	e2eWantStepRead     = "read the mount line"
	e2eWantStepValidate = "validate the mount tree"
)

// e2eMountFailureCase is one binary-tier failure family.
type e2eMountFailureCase struct {
	name    string
	variant string
	want    []string
}

// e2eMountFailureCases is the binary-tier fast failure matrix.
func e2eMountFailureCases() []e2eMountFailureCase {
	return []e2eMountFailureCase{
		{
			name:    "early-exit",
			variant: e2eFailureEarlyExit,
			want:    []string{e2eWantStepRead, "exit status 127"},
		},
		{
			name:    "bogus-line",
			variant: e2eFailureBogusLine,
			want:    []string{e2eWantStepRead, "is not one .mount_* component"},
		},
		{
			name:    "missing-tree",
			variant: e2eFailureMissingTree,
			want:    []string{e2eWantStepValidate, "zcode"},
		},
		{
			name:    "no-fuse",
			variant: e2eFailureNoFuse,
			want: []string{
				e2eWantStepRead, "dlopen(): error loading libfuse.so.2", "libfuse2t64", "/dev/fuse",
			},
		},
	}
}

// TestAppImageMountFailures drives the BUILT wrapper through the fast mount
// failure families: every arm exits 1 (a mount failure is a runtime failure,
// never a usage error), writes the AppImage path, the failing step, the
// observed output, and a remediation to stderr, never starts an ACP session,
// and leaves no mount path or private base behind (MNT-04, D-03). The
// readiness-timeout arm lives at the supervisor tier, where MountOpts can
// shorten ReadyTimeout.
func TestAppImageMountFailures(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")

	for _, tc := range e2eMountFailureCases() {
		t.Run(tc.name, func(t *testing.T) {
			fixture := t.TempDir()
			tmpRoot := t.TempDir()
			appImage := e2eWriteFakeAppImageFailure(t, fixture, tc.variant)

			ctx, cancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
			defer cancel()

			wrapper := exec.CommandContext(ctx, agentBin, ompFlagAppServer, appImage)
			wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			wrapper.Env = append(os.Environ(), "TMPDIR="+tmpRoot)

			stderrBuf := &e2eSyncBuffer{}
			wrapper.Stderr = stderrBuf

			err := wrapper.Run()

			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				stderrBuf.dump(t)

				t.Fatalf("wrapper exit = %v, want exit code 1 (a mount failure is a runtime failure)", err)
			}

			stderr := stderrBuf.String()

			if !strings.Contains(stderr, appImage) {
				t.Fatalf("stderr %q must name the AppImage %q", stderr, appImage)
			}

			if !strings.Contains(stderr, "remediation:") {
				t.Fatalf("stderr %q must carry the remediation", stderr)
			}

			for _, want := range tc.want {
				if !strings.Contains(stderr, want) {
					t.Fatalf("stderr %q must carry %q", stderr, want)
				}
			}

			if record, readErr := os.ReadFile(filepath.Join(fixture, e2eFakeMountRecord)); readErr == nil {
				path := strings.TrimSpace(string(record))
				if path != "" {
					e2eAssertGonePath(t, path, "the recorded mount")
				}
			}

			e2eAssertNoMountBase(t, tmpRoot)
		})
	}
}

// e2eWriteFakeAppImageFailure writes a fast failure variant of the scripted
// runtime; it records its PID so a leaked helper is visible on failure.
func e2eWriteFakeAppImageFailure(t *testing.T, fixture, variant string) string {
	t.Helper()

	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("resolve the current test binary: %v", err)
	}

	body := strings.NewReplacer(
		"@@PID@@", filepath.Join(fixture, e2eFakeHelperPID),
		"@@LEAK_PID@@", filepath.Join(fixture, e2eFakeLeakPID),
		"@@TEST_BINARY@@", testBinary,
	).Replace(e2eFakeAppImageFailureScript(variant))

	appImage := filepath.Join(fixture, e2eFakeAppImageName)
	if err := os.WriteFile(appImage, []byte(body), 0o750); err != nil {
		t.Fatalf("write the fake AppImage failure runtime: %v", err)
	}

	return appImage
}

// e2eFakeAppImageFailureScript returns the fast failure script body.
func e2eFakeAppImageFailureScript(variant string) string {
	const preamble = `#!/bin/sh
set -eu
if [ "${1:-}" != "--appimage-mount" ]; then
  echo "fake-appimage: unsupported argv: $*" >&2
  exit 64
fi
echo $$ > "@@PID@@"
base=${TMPDIR:?TMPDIR must be set by the wrapper}
`

	switch variant {
	case e2eFailureNoLine:
		// Creates a valid tree but never prints the mount line: the wrapper
		// sits in PrepareMount until a signal cancels it (the startup-signal
		// arm), so teardown must stop and reap the helper and leave no base.
		return preamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
while :; do sleep 0.05; done
`
	case e2eFailureIgnoreTerm:
		// The readiness-teardown arm (CR-01): creates a tree but never prints
		// the mount line, ignores TERM, and detaches a descendant into its own
		// session. The descendant keeps the inherited stderr pipe open, so the
		// helper's process tree can never be reaped within ReapDeadline and
		// the signal-time teardown fails.
		return preamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
ZCODE_ACP_TEST_DETACHED_LEAK_LAUNCHER=1 \
ZCODE_ACP_TEST_DETACHED_LEAK_PID="@@LEAK_PID@@" \
"@@TEST_BINARY@@" '-test.run=^TestE2EDetachedLeakLauncherHelper$' >/dev/null &
trap '' TERM
while :; do sleep 0.05; done
`
	case e2eFailureBogusLine:
		return preamble + `echo /tmp/zcode-acp-not-ours
while :; do sleep 0.05; done
`
	case e2eFailureMissingTree:
		return preamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm"
: > "$dir/resources/glm/zcode.cjs"
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	case e2eFailureNoFuse:
		return preamble + `echo ""
echo "dlopen(): error loading libfuse.so.2"
echo "Cannot mount AppImage, please check your FUSE setup."
exit 127
`
	default:
		return preamble + "exit 127\n"
	}
}

// TestE2EDetachedLeakLauncherHelper runs inside the fake mount helper's process
// group and starts only the sleeping descendant in a detached session. Returning
// after Start leaves that descendant alive with this process's inherited stderr.
func TestE2EDetachedLeakLauncherHelper(t *testing.T) {
	if os.Getenv(e2eDetachedLeakLauncherEnv) != "1" {
		return
	}

	pidPath := os.Getenv(e2eDetachedLeakPIDEnv)
	if pidPath == "" {
		t.Fatalf("%s is missing", e2eDetachedLeakPIDEnv)
	}

	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestE2EDetachedLeakHelper$")
	cmd.Env = append(os.Environ(),
		e2eDetachedLeakHelperEnv+"=1",
		e2eDetachedLeakPIDEnv+"="+pidPath,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start the detached leak helper: %v", err)
	}
}

// TestE2EDetachedLeakHelper records its PID and holds the inherited wrapper
// stderr pipe open for the fixture's bounded lifetime. Its dedicated marker keeps
// it inert in the ordinary suite.
func TestE2EDetachedLeakHelper(t *testing.T) {
	if os.Getenv(e2eDetachedLeakHelperEnv) != "1" {
		return
	}

	pidPath := os.Getenv(e2eDetachedLeakPIDEnv)
	if pidPath == "" {
		t.Fatalf("%s is missing", e2eDetachedLeakPIDEnv)
	}

	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatalf("record the detached leak helper PID: %v", err)
	}

	time.Sleep(e2eDetachedLeakLifetime)
}

// e2eAssertNoMountBase requires no zcode-acp-mount-* base to remain under root.
func e2eAssertNoMountBase(t *testing.T, root string) {
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

// TestAppImageMountSignal is the OS-signal exit path (MNT-03, D-02): SIGTERM,
// SIGINT, and SIGHUP run the staged teardown (app-server first, helper second),
// exit 0, leave zero orphans and zero stale mounts, and the fake transcript —
// written only after the app-server's clean stdin-EOF drain — proves a
// signal-derived spawn context never SIGKILLed the child. A signal during
// mount readiness is a clean exit 0 when the teardown succeeds; a teardown
// that cannot reap the helper or prove the mount gone exits 1 even when a
// signal was delivered (CR-01).
func TestAppImageMountSignal(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	signals := []struct {
		name string
		sig  os.Signal
	}{
		{"SIGTERM", syscall.SIGTERM},
		{"SIGINT", syscall.SIGINT},
		{"SIGHUP", syscall.SIGHUP},
	}

	for _, tc := range signals {
		t.Run(tc.name, func(t *testing.T) {
			e2eSignalShutdownRide(t, agentBin, fakeBin, tc.sig)
		})
	}

	t.Run("signal during mount readiness", func(t *testing.T) {
		e2eSignalDuringReadiness(t, agentBin)
	})

	t.Run("failed teardown during mount readiness outranks the signal", func(t *testing.T) {
		e2eReadinessTeardownFailureBeatsSignal(t, agentBin)
	})

	t.Run("teardown failure outranks the signal", func(t *testing.T) {
		e2eTeardownFailureBeatsSignal(t, agentBin, fakeBin)
	})
}

// e2eSignalShutdownRide drives one signal through a live ride and asserts the
// clean staged teardown.
func e2eSignalShutdownRide(t *testing.T, agentBin, fakeBin string, sig os.Signal) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(e2eScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eScenario, err)
	}

	fixture := t.TempDir()
	transcriptPath := filepath.Join(fixture, "fake-transcript.txt")
	appImage := e2eWriteFakeAppImage(t, fixture, fakeBin, scenarioAbs, transcriptPath)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin, ompFlagAppServer, appImage)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	t.Cleanup(e2eCleanupMountFixture(t, fixture))

	stdin, stdout := e2eStartWrapper(t, wrapper, stderrBuf)

	mountPath := e2eWaitMountRecord(t, fixture, stderrBuf)

	client := acp.NewClientSideConnection(&e2eCollector{}, stdin, stdout)
	e2eInitialize(rideCtx, t, client)
	e2eNewSession(rideCtx, t, client)

	if sigErr := wrapper.Process.Signal(sig); sigErr != nil {
		t.Fatalf("send %v to the wrapper: %v", sig, sigErr)
	}

	e2eWaitExitWithin(t, wrapper, e2eSignalExitWatchdog)

	e2eAssertGonePath(t, mountPath, "the managed mount")
	e2eAssertGonePath(t, filepath.Dir(mountPath), "the private mount base")
	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeHelperPID), "mount helper")
	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeAppServerPID), "app-server")
	e2eAssertOrderedTeardown(t, fixture)
	e2eAssertServedTranscript(t, transcriptPath)
}

// e2eAssertServedTranscript proves the app-server reached its clean EOF-drain
// exit: the transcript exists and carries a served session/create. A
// signal-derived spawn context would SIGKILL the child mid-flight and never
// produce it.
func e2eAssertServedTranscript(t *testing.T, transcriptPath string) {
	t.Helper()

	lines := e2eWaitTranscript(t, transcriptPath)

	for _, line := range lines {
		if line.dir == "in" && strings.Contains(line.json, `"method":"session/create"`) {
			return
		}
	}

	t.Fatalf("the fake transcript carries no served session/create: the app-server was killed " +
		"before its stdin-EOF drain (the spawn context must not be signal-derived)")
}

// e2eSignalDuringReadiness signals the wrapper while PrepareMount waits on a
// helper that never prints: the run must stop and reap the helper and exit 0
// (a signal during acquisition is an intentional shutdown, not a mount
// failure), leaving no base residue.
func e2eSignalDuringReadiness(t *testing.T, agentBin string) {
	t.Helper()

	fixture := t.TempDir()
	tmpRoot := t.TempDir()
	appImage := e2eWriteFakeAppImageFailure(t, fixture, e2eFailureNoLine)

	ctx, cancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer cancel()

	wrapper := exec.CommandContext(ctx, agentBin, ompFlagAppServer, appImage)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	wrapper.Env = append(os.Environ(), "TMPDIR="+tmpRoot)

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	t.Cleanup(e2eCleanupMountFixture(t, fixture))

	_, _ = e2eStartWrapper(t, wrapper, stderrBuf)

	e2eWaitPIDFile(t, filepath.Join(fixture, e2eFakeHelperPID), stderrBuf)

	if sigErr := wrapper.Process.Signal(syscall.SIGTERM); sigErr != nil {
		t.Fatalf("send SIGTERM during readiness: %v", sigErr)
	}

	e2eWaitExitWithin(t, wrapper, e2eSignalExitWatchdog)

	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeHelperPID), "mount helper")
	e2eAssertNoMountBase(t, tmpRoot)
	e2eAssertGonePath(t, filepath.Join(fixture, e2eFakeMountRecord), "the mount record")
}

// e2eReadinessTeardownFailureBeatsSignal proves the acquisition-failure
// precedence (CR-01): the helper ignores TERM and detaches a descendant into
// its own session, so the inherited stderr pipe never closes and the
// readiness-time teardown cannot reap the helper within ReapDeadline. The
// joined readiness+teardown failure must surface — exit 1 naming the AppImage,
// the failing step, the remediation, and the incomplete teardown — even though
// the signal context is cancelled; the teardown error is never swallowed.
func e2eReadinessTeardownFailureBeatsSignal(t *testing.T, agentBin string) {
	t.Helper()

	fixture := t.TempDir()
	tmpRoot := t.TempDir()
	appImage := e2eWriteFakeAppImageFailure(t, fixture, e2eFailureIgnoreTerm)
	leakPID := filepath.Join(fixture, e2eFakeLeakPID)

	t.Cleanup(func() { e2eKillRecordedPID(leakPID) })

	ctx, cancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer cancel()

	wrapper := exec.CommandContext(ctx, agentBin, ompFlagAppServer, appImage)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	wrapper.Env = append(os.Environ(), "TMPDIR="+tmpRoot)

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	t.Cleanup(e2eCleanupMountFixture(t, fixture))

	_, _ = e2eStartWrapper(t, wrapper, stderrBuf)

	e2eWaitPIDFile(t, filepath.Join(fixture, e2eFakeHelperPID), stderrBuf)
	// The detached descendant must exist before the signal: it is what keeps
	// the helper's stderr pipe open past the group SIGKILL.
	e2eWaitPIDFile(t, leakPID, stderrBuf)

	if sigErr := wrapper.Process.Signal(syscall.SIGTERM); sigErr != nil {
		t.Fatalf("send SIGTERM during readiness: %v", sigErr)
	}

	e2eWaitExitCodeWithin(t, wrapper, e2eSignalExitWatchdog, 1)

	stderr := stderrBuf.String()
	for _, want := range []string{
		appImage, e2eWantStepRead, "remediation:", "teardown incomplete",
	} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q must carry %q (a failed readiness-time teardown must beat the signal)",
				stderr, want)
		}
	}

	e2eAssertNoMountBase(t, tmpRoot)
}

// e2eKillRecordedPID kills a PID the fixture recorded, when present. The
// ignore-term fixture's detached descendant deliberately outlives the wrapper,
// so its cleanup is unconditional (not only on test failure).
func e2eKillRecordedPID(path string) {
	pid := e2eReadPID(path)
	if pid <= 0 {
		return
	}

	_ = syscall.Kill(pid, syscall.SIGKILL)
}

// e2eTeardownFailureBeatsSignal proves the cause precedence: with SIGTERM
// already delivered and the leaky helper's tree unable to disappear, the run
// exits 1 with the mount and the manual-unmount remediation — never a clean 0
// — after the app-server is torn down first, and retains the base.
func e2eTeardownFailureBeatsSignal(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(e2eScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eScenario, err)
	}

	fixture := t.TempDir()
	tmpRoot := t.TempDir()
	transcriptPath := filepath.Join(fixture, "fake-transcript.txt")
	appImage := e2eWriteFakeAppImageVariant(t, fixture, fakeBin, scenarioAbs, transcriptPath, e2eVariantLeaky)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin, ompFlagAppServer, appImage)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	wrapper.Env = append(os.Environ(), "TMPDIR="+tmpRoot)

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	t.Cleanup(e2eCleanupMountFixture(t, fixture))

	stdin, stdout := e2eStartWrapper(t, wrapper, stderrBuf)

	mountPath := e2eWaitMountRecord(t, fixture, stderrBuf)
	base := filepath.Dir(mountPath)

	t.Cleanup(func() { _ = os.RemoveAll(base) }) // the retained base, cleaned after the asserts

	client := acp.NewClientSideConnection(&e2eCollector{}, stdin, stdout)
	e2eInitialize(rideCtx, t, client)
	e2eNewSession(rideCtx, t, client)

	if sigErr := wrapper.Process.Signal(syscall.SIGTERM); sigErr != nil {
		t.Fatalf("send SIGTERM while the mount is stuck: %v", sigErr)
	}

	e2eWaitExitCodeWithin(t, wrapper, e2eSignalExitWatchdog, 1)
	e2eAssertTeardownFailure(t, fixture, base, mountPath, stderrBuf)
}

// e2eAssertTeardownFailure asserts the teardown-error arm: the exit-1 stderr
// names the mount and remediation, the app-server went first, the base and its
// .mount_* child are retained, and both children are reaped.
func e2eAssertTeardownFailure(t *testing.T, fixture, base, mountPath string, stderrBuf *e2eSyncBuffer) {
	t.Helper()

	stderr := stderrBuf.String()
	for _, want := range []string{mountPath, "teardown", "fusermount -u"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q must carry %q (the teardown error must beat the signal)", stderr, want)
		}
	}

	e2eAssertOrderedTeardown(t, fixture)

	entries, readErr := os.ReadDir(base)
	if readErr != nil {
		t.Fatalf("the retained private base %s must exist: %v", base, readErr)
	}

	foundMount := false

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".mount_") {
			foundMount = true
		}
	}

	if !foundMount {
		t.Fatalf("the retained private base %s lost its .mount_* child", base)
	}

	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeHelperPID), "mount helper")
	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeAppServerPID), "app-server")
}

// TestAppImageMountLost proves mid-session loss (MNT-05, D-05): the helper
// removes the tree and exits after the app-server started, the run exits 1
// with the mount cause, and the app-server records no completed ride — the
// supervisor never respawns against the vanished root.
func TestAppImageMountLost(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	scenarioAbs, err := filepath.Abs(e2eScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eScenario, err)
	}

	fixture := t.TempDir()
	tmpRoot := t.TempDir()
	transcriptPath := filepath.Join(fixture, "fake-transcript.txt")
	appImage := e2eWriteFakeAppImageVariant(t, fixture, fakeBin, scenarioAbs, transcriptPath, e2eVariantDieAfter)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin, ompFlagAppServer, appImage)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	wrapper.Env = append(os.Environ(), "TMPDIR="+tmpRoot)

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	t.Cleanup(e2eCleanupMountFixture(t, fixture))

	_, _ = e2eStartWrapper(t, wrapper, stderrBuf)

	mountPath := e2eWaitMountRecord(t, fixture, stderrBuf)
	e2eWaitPIDFile(t, filepath.Join(fixture, e2eFakeAppServerPID), stderrBuf)

	e2eWaitExitCodeWithin(t, wrapper, e2eMountLossWatchdog, 1)

	stderr := stderrBuf.String()
	for _, want := range []string{"mount lost", appImage} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr %q must carry the mount cause %q", stderr, want)
		}
	}

	e2eAssertGonePath(t, mountPath, "the lost mount")
	e2eAssertGonePath(t, filepath.Dir(mountPath), "the private mount base")
	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeHelperPID), "mount helper")
	e2eAssertPIDGone(t, filepath.Join(fixture, e2eFakeAppServerPID), "app-server")

	if data, readErr := os.ReadFile(transcriptPath); readErr == nil {
		if strings.Contains(string(data), `"method":"session/send"`) ||
			strings.Contains(string(data), `"method":"session/create"`) {
			t.Fatalf("the app-server transcript records a completed ride after the mount was lost: %s", data)
		}
	}
}

// e2eWaitPIDFile polls for a recorded PID file and returns its pid.
func e2eWaitPIDFile(t *testing.T, path string, stderrBuf *e2eSyncBuffer) int {
	t.Helper()

	deadline := time.After(e2eMountRecordWait)

	for {
		if pid := e2eReadPID(path); pid > 0 {
			return pid
		}

		select {
		case <-deadline:
			stderrBuf.dump(t)

			t.Fatalf("the fixture PID record %s never appeared within %v", path, e2eMountRecordWait)
		case <-time.After(e2eFixturePoll):
		}
	}
}

// e2eWaitExitWithin waits for a clean exit 0 within the given bound.
func e2eWaitExitWithin(t *testing.T, wrapper *exec.Cmd, within time.Duration) {
	t.Helper()

	if err := e2eAwaitWrapperExit(t, wrapper, within); err != nil {
		t.Fatalf("wrapper exit = %v, want a clean exit 0", err)
	}
}

// e2eWaitExitCodeWithin waits for the wrapper's exit and requires want.
func e2eWaitExitCodeWithin(t *testing.T, wrapper *exec.Cmd, within time.Duration, want int) {
	t.Helper()

	err := e2eAwaitWrapperExit(t, wrapper, within)

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != want {
		t.Fatalf("wrapper exit = %v, want exit code %d", err, want)
	}
}

// e2eAwaitWrapperExit waits for the wrapper's exit within the bound.
func e2eAwaitWrapperExit(t *testing.T, wrapper *exec.Cmd, within time.Duration) error {
	t.Helper()

	waitCh := make(chan error, 1)

	go func() { waitCh <- wrapper.Wait() }()

	select {
	case err := <-waitCh:
		return err
	case <-time.After(within):
		t.Fatalf("wrapper did not exit within %v", within)

		return nil
	}
}

// e2eWriteFakeAppImage writes the scripted AppImage runtime: it answers only
// --appimage-mount, creates a .mount_* directory under the wrapper's private
// TMPDIR with the invariant spawn surface, records its own PID and the mount
// path, prints the path, and stays alive. A TERM trap records whether the
// app-server was still alive and removes the tree. The zcode shim bakes the
// fixture's absolute fake-binary/scenario/transcript paths in (its own dirname
// is the mount root, never the fixture dir) and records the app-server PID
// before exec-ing it.
func e2eWriteFakeAppImage(t *testing.T, fixture, fakeBin, scenarioAbs, transcript string) string {
	t.Helper()

	return e2eWriteFakeAppImageVariant(t, fixture, fakeBin, scenarioAbs, transcript, e2eVariantHappy)
}

// Fast lifecycle variant names for the loss/signal tests.
const (
	e2eVariantHappy    = "happy"
	e2eVariantDieAfter = "die-after"
	e2eVariantLeaky    = "leaky"
)

// e2eWriteFakeAppImageVariant writes one lifecycle variant of the scripted
// runtime (the @@ tokens are substituted with the fixture's absolute paths).
func e2eWriteFakeAppImageVariant(t *testing.T, fixture, fakeBin, scenarioAbs, transcript, variant string) string {
	t.Helper()

	replacements := strings.NewReplacer(
		"@@HELPER_PID@@", filepath.Join(fixture, e2eFakeHelperPID),
		"@@APP_PID@@", filepath.Join(fixture, e2eFakeAppServerPID),
		"@@MOUNT_RECORD@@", filepath.Join(fixture, e2eFakeMountRecord),
		"@@TERM_RECORD@@", filepath.Join(fixture, e2eFakeTermRecord),
		"@@FAKE_BIN@@", fakeBin,
		"@@SCENARIO@@", scenarioAbs,
		"@@TRANSCRIPT@@", transcript,
	)

	appImage := filepath.Join(fixture, e2eFakeAppImageName)
	if err := os.WriteFile(appImage, []byte(replacements.Replace(e2eFakeAppImageScript(variant))), 0o750); err != nil {
		t.Fatalf("write the fake AppImage runtime: %v", err)
	}

	return appImage
}

// e2eFakeAppImageTree is the shared mount-tree creation body: the record, the
// invariant spawn surface, and the app-server shim (which records its PID
// before exec).
const e2eFakeAppImageTree = `base=${TMPDIR:?TMPDIR must be set by the wrapper}
dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
printf '%s\n' "$dir" > "@@MOUNT_RECORD@@"
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
{
  echo '#!/bin/sh'
  echo 'echo $$ > "@@APP_PID@@"'
  echo 'exec "@@FAKE_BIN@@" --scenario "@@SCENARIO@@" --transcript "@@TRANSCRIPT@@" "$@"'
} > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
`

// e2eFakeAppImageScript is the scripted runtime body (the @@ tokens are
// substituted with the fixture's absolute paths).
func e2eFakeAppImageScript(variant string) string {
	const preamble = `#!/bin/sh
set -eu
if [ "${1:-}" != "--appimage-mount" ]; then
  echo "fake-appimage: unsupported argv: $*" >&2
  exit 64
fi
echo $$ > "@@HELPER_PID@@"
`

	const reportTerm = `report_term() {
  app=""
  if [ -s "@@APP_PID@@" ]; then
    app=$(cat "@@APP_PID@@")
  fi
  if [ -n "$app" ] && kill -0 "$app" 2>/dev/null; then
    echo app-server-alive-on-term >> "@@TERM_RECORD@@"
  else
    echo app-server-gone-on-term >> "@@TERM_RECORD@@"
  fi
}
`

	switch variant {
	case e2eVariantDieAfter:
		// The loss arm: prints the path, waits for the app-server PID file,
		// then removes the tree and exits (watcher-driven loss).
		return preamble + e2eFakeAppImageTree + `printf '%s\n' "$dir"
while [ ! -s "@@APP_PID@@" ]; do sleep 0.02; done
rm -rf "$dir"
exit 0
`
	case e2eVariantLeaky:
		// The teardown-failure arm: ignores TERM (recording whether the
		// app-server was alive), installs no cleanup trap, so the tree
		// outlives the helper.
		return preamble + e2eFakeAppImageTree + reportTerm + `trap report_term TERM
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	default:
		return preamble + e2eFakeAppImageTree + reportTerm + `report_term_and_cleanup() {
  report_term
  rm -rf "$dir"
  exit 143
}
trap report_term_and_cleanup TERM
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
	}
}

// e2eWaitMountRecord polls for the mount path the fake helper records before
// it prints the line; a wrapper that never mounts the AppImage fails here.
func e2eWaitMountRecord(t *testing.T, fixture string, stderrBuf *e2eSyncBuffer) string {
	t.Helper()

	path := filepath.Join(fixture, e2eFakeMountRecord)
	deadline := time.After(e2eMountRecordWait)

	for {
		data, err := os.ReadFile(path)
		if err == nil && strings.TrimSpace(string(data)) != "" {
			return strings.TrimSpace(string(data))
		}

		select {
		case <-deadline:
			stderrBuf.dump(t)

			t.Fatalf("the fake AppImage was never mounted (no record at %s within %v): "+
				"the --app-server .AppImage value reached the spawn seam as the app-server "+
				"binary instead of the mount path", path, e2eMountRecordWait)
		case <-time.After(e2eFixturePoll):
		}
	}
}

// e2eAssertOrderedTeardown asserts the helper observed the app-server already
// gone when it received TERM (app-server first, helper second).
func e2eAssertOrderedTeardown(t *testing.T, fixture string) {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(fixture, e2eFakeTermRecord))
	if err != nil {
		t.Fatalf("read the helper's TERM record: %v", err)
	}

	record := string(data)
	if !strings.Contains(record, "app-server-gone-on-term") {
		t.Fatalf("teardown order violated: the helper saw the app-server alive on TERM (%q); "+
			"the app-server must be stopped and reaped before the mount helper", record)
	}
}

// e2eAssertGonePath requires path to be gone.
func e2eAssertGonePath(t *testing.T, path, what string) {
	t.Helper()

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s still exists at %s (stat error %v)", what, path, err)
	}
}

// e2eAssertPIDGone requires the recorded PID to be gone (ESRCH) within the
// bounded poll — a zombie would still answer kill(pid, 0), so this proves the
// wrapper reaped the child, not merely killed it.
func e2eAssertPIDGone(t *testing.T, pidPath, label string) {
	t.Helper()

	data, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatalf("read the %s pid record: %v", label, err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		t.Fatalf("the %s pid record %q is not a pid: %v", label, string(data), err)
	}

	deadline := time.After(e2ePIDGoneWait)

	for {
		killErr := syscall.Kill(pid, 0)
		if errors.Is(killErr, syscall.ESRCH) {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("the %s process %d still exists (kill 0: %v)", label, pid, killErr)
		case <-time.After(e2eFixturePoll):
		}
	}
}

// e2eAssertNoExtraction asserts no squashfs-root or zcode-{ver} copy appeared
// under the fixture dir (where the old --appimage-extract route would write).
func e2eAssertNoExtraction(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read the fixture dir: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "squashfs-root" || strings.HasPrefix(name, "zcode-") {
			t.Fatalf("the run created an extraction copy %q in %s (MNT-06: no on-disk duplicate)", name, dir)
		}
	}
}

// e2eFileSHA256 returns the hex sha256 of path.
func e2eFileSHA256(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s for hashing: %v", path, err)
	}

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// e2eCleanupMountFixture is the failed-run backstop: the helper and the
// app-server live in their own process groups, so the wrapper's group kill
// never reaches them. Kill both recorded groups and remove the recorded mount
// directory — a failing test must not leak children or mounts into CI.
func e2eCleanupMountFixture(t *testing.T, fixture string) func() {
	t.Helper()

	return func() {
		if !t.Failed() {
			return
		}

		for _, name := range []string{e2eFakeHelperPID, e2eFakeAppServerPID} {
			pid := e2eReadPID(filepath.Join(fixture, name))
			if pid <= 0 {
				continue
			}
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
				t.Logf("kill the %s group: %v", name, err)
			}
		}

		if data, err := os.ReadFile(filepath.Join(fixture, e2eFakeMountRecord)); err == nil {
			mountPath := strings.TrimSpace(string(data))
			if mountPath != "" {
				if err := os.RemoveAll(mountPath); err != nil {
					t.Logf("remove the leaked mount %s: %v", mountPath, err)
				}
			}
		}
	}
}

// e2eReadPID reads a recorded PID file (0 when absent or unparseable).
func e2eReadPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}

	return pid
}

// e2eRunResponderLeg is the plan 05-07 wiring proof at the binary tier:
// the BUILT binary's wired responder answers a scripted interaction
// reverse through production stdio. The create fires one
// interaction/requestUserInput reverse mid-create; the leg passes only
// when the fake's transcript records the schema-valid decline reply and no
// unanswered-reverse note (the differential discriminator: the conn's
// built-in router would answer -32601, failing the expect semantics).
func e2eRunResponderLeg(t *testing.T, agentBin, fakeBin string) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(e2eResponderScenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", e2eResponderScenario, err)
	}

	transcriptPath := filepath.Join(t.TempDir(), "fake-transcript.txt")
	installRoot := t.TempDir()
	e2eWriteFakeInstallRoot(t, installRoot, fakeBin, scenarioAbs, transcriptPath)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin, ompFlagAppServer, installRoot)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	stdin, err := wrapper.StdinPipe()
	if err != nil {
		t.Fatalf("wrapper stdin pipe: %v", err)
	}

	stdout, err := wrapper.StdoutPipe()
	if err != nil {
		t.Fatalf("wrapper stdout pipe: %v", err)
	}

	if err := wrapper.Start(); err != nil {
		t.Fatalf("spawn %s: %v", agentBin, err)
	}

	t.Cleanup(e2eKillGroup(t, wrapper, stderrBuf))

	updates := &e2eCollector{}
	client := acp.NewClientSideConnection(updates, stdin, stdout)

	e2eInitialize(rideCtx, t, client)

	// session/new: the create fires the reverse mid-create — the wired
	// responder must answer it for the leg to complete inside the cycle
	// deadline (an unanswered reverse holds the create's side-effect chain
	// for the reverse's own timeout first).
	e2eNewSession(rideCtx, t, client)

	if err := stdin.Close(); err != nil {
		t.Logf("close wrapper stdin: %v", err)
	}

	e2eWaitExit(t, wrapper)
	e2eAssertResponderTranscript(t, transcriptPath)

	// The ask-the-user bridge (v0.22.0): the scripted requestUserInput
	// reverse must have crossed as a FORM elicitation on this client —
	// the collector records it before the decline rides back.
	updates.mu.Lock()
	elicitations := len(updates.elicitations)
	updates.mu.Unlock()

	if elicitations == 0 {
		t.Fatal("the bridge built no elicitation — the scripted ask never rendered on the client surface")
	}
}

// e2eAssertResponderTranscript asserts the responder leg's transcript: the
// requestUserInput reverse was asked, a decline reply answered it AFTER the
// ask, and no unanswered-reverse note was recorded (record-honesty: the
// wiring proof never normalizes a missing answer away).
func e2eAssertResponderTranscript(t *testing.T, path string) {
	t.Helper()

	lines := e2eWaitTranscript(t, path)

	reqIdx := -1
	for i, line := range lines {
		if line.dir == "out" && strings.Contains(line.json, `"method":"interaction/requestUserInput"`) {
			reqIdx = i

			break
		}
	}

	if reqIdx < 0 {
		t.Fatal("fake transcript records no requestUserInput reverse — the scripted ask never fired")
	}

	replyIdx := -1
	for i := reqIdx + 1; i < len(lines); i++ {
		if lines[i].dir == "in" && strings.Contains(lines[i].json, `"action":"decline"`) {
			replyIdx = i

			break
		}
	}

	if replyIdx < 0 {
		t.Fatalf("no decline reply after the requestUserInput reverse (ask at %d) — "+
			"the wired responder did not answer through production stdio", reqIdx)
	}

	for _, line := range lines {
		if line.dir == "note" && strings.Contains(line.json, "requestUserInput") {
			t.Fatalf("unanswered-reverse note recorded for requestUserInput: %s", line.json)
		}
	}
}

// e2eInitialize runs the initialize leg and asserts the agent identity: the
// version var IS the agentInfo version source (D-02 real agentInfo).
func e2eInitialize(ctx context.Context, t *testing.T, client *acp.ClientSideConnection) {
	t.Helper()

	resp, err := client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}

	if resp.ProtocolVersion != 1 {
		t.Fatalf("protocolVersion = %d, want 1", resp.ProtocolVersion)
	}

	if resp.AgentInfo == nil || resp.AgentInfo.Name != "zcode-acp" {
		t.Fatalf("agentInfo = %+v, want name zcode-acp", resp.AgentInfo)
	}

	if resp.AgentInfo.Version != version {
		t.Fatalf("agentInfo.version = %q, want the binary's version var %q", resp.AgentInfo.Version, version)
	}
}

// e2eNewSession opens one session over the spawned binary.
func e2eNewSession(ctx context.Context, t *testing.T, client *acp.ClientSideConnection) string {
	t.Helper()

	resp, err := client.NewSession(ctx, acp.NewSessionRequest{Cwd: t.TempDir()})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if resp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	return string(resp.SessionId)
}

// e2eLoopOne drives the completed loop: streamed text deltas then end_turn.
func e2eLoopOne(ctx context.Context, t *testing.T, client *acp.ClientSideConnection, sid string) {
	t.Helper()

	resp, err := client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("loop one")},
	})
	if err != nil {
		t.Fatalf("loop 1 prompt: %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("loop 1 stopReason = %q, want %q", resp.StopReason, acp.StopReasonEndTurn)
	}
}

// e2eLoopTwo drives the cancelled loop: prompt in flight (synced on the
// streamed slow delta — the tap proves the turn is running), cancel
// mid-turn, cancelled exactly once as a SUCCESS response (never a -32800
// error — the SDK trap the front's turn-ctx ownership exists for).
func e2eLoopTwo(
	ctx context.Context, t *testing.T, client *acp.ClientSideConnection, updates *e2eCollector, sid string,
) {
	t.Helper()

	promptDone := make(chan acp.PromptResponse, 1)
	promptErr := make(chan error, 1)

	go func() {
		resp, err := client.Prompt(ctx, acp.PromptRequest{
			SessionId: acp.SessionId(sid),
			Prompt:    []acp.ContentBlock{acp.TextBlock("loop two")},
		})
		if err != nil {
			promptErr <- err

			return
		}

		promptDone <- resp
	}()

	e2eWaitUpdateText(t, updates, "slow") // loop 2's scripted delta — the turn is in flight

	if err := client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	select {
	case resp := <-promptDone:
		if resp.StopReason != acp.StopReasonCancelled {
			t.Fatalf("loop 2 stopReason = %q, want %q", resp.StopReason, acp.StopReasonCancelled)
		}
	case err := <-promptErr:
		t.Fatalf("cancelled loop must answer as a success response, got error: %v", err)
	case <-time.After(e2eLoopWait):
		t.Fatal("cancelled loop never resolved within the wait bound")
	}
}

// e2eWaitUpdateText polls the test client's collected updates for one
// carrying want — the wire-level sync signal that the scripted turn is
// streaming (chunk frames cross the real binary boundary only when the tap
// is live and the turn is running).
func e2eWaitUpdateText(t *testing.T, updates *e2eCollector, want string) {
	t.Helper()

	deadline := time.After(e2eUpdateWait)

	for {
		if updates.hasUpdateText(want) {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("update text %q never arrived; collected %d updates", want, len(updates.DrainUpdates()))
		case <-time.After(5 * time.Millisecond):
		}
	}
}

// e2eWaitExit waits for the wrapper's clean exit after the client
// disconnected (the staged shutdown path — exit 0).
func e2eWaitExit(t *testing.T, wrapper *exec.Cmd) {
	t.Helper()

	waitCh := make(chan error, 1)

	go func() { waitCh <- wrapper.Wait() }()

	select {
	case err := <-waitCh:
		if err != nil {
			t.Fatalf("wrapper exit = %v, want clean exit 0 on client disconnect", err)
		}
	case <-time.After(e2eExitWatchdog):
		t.Fatal("wrapper did not exit after the client disconnected")
	}
}

// e2eWriteFakeInstallRoot fabricates the fake-scenario install shape the
// Override seam accepts: the invariant spawn surface (<root>/zcode +
// <root>/resources/glm/zcode.cjs, install-discovery.md :38) where the
// "zcode" binary is a shim exec-ing the built fake binary with its scenario
// (and transcript sink) baked in — the production discovery order and the
// canonical SpawnArgv line run against it unchanged.
func e2eWriteFakeInstallRoot(t *testing.T, root, fakeBin, scenarioAbs, transcriptPath string) {
	t.Helper()

	glmDir := filepath.Join(root, "resources", "glm")
	if err := os.MkdirAll(glmDir, 0o750); err != nil {
		t.Fatalf("create install tree: %v", err)
	}

	if err := os.WriteFile(filepath.Join(glmDir, "zcode.cjs"), []byte("\n"), 0o640); err != nil {
		t.Fatalf("create entrypoint stub: %v", err)
	}

	shim := "#!/bin/sh\nexec " + fakeBin + " --scenario " + scenarioAbs +
		" --transcript " + transcriptPath + " \"$@\"\n"

	if err := os.WriteFile(filepath.Join(root, "zcode"), []byte(shim), 0o750); err != nil {
		t.Fatalf("create shim binary: %v", err)
	}
}

// e2eTranscriptScan is the classified transcript summary the asserts read.
type e2eTranscriptScan struct {
	createIdx, subIdx, setModelIdx int
	setModelJSON                   string
	sends, stops                   int
}

// e2eAssertFakeTranscript loads the fake's transcript file (written at its
// EOF-drain exit — the binary hosting mode's window into its own wire) and
// asserts the D-03 model contract rode the REAL spawn: the setModel payload
// verbatim from the --model/--reasoning-level flags, the preamble wire
// order, exactly the two prompt sends, and the one bypass stop.
func e2eAssertFakeTranscript(t *testing.T, path string) {
	t.Helper()

	lines := e2eWaitTranscript(t, path)
	scan := e2eScanTranscript(t, lines)

	if scan.createIdx < 0 || scan.subIdx < 0 || scan.setModelIdx < 0 {
		t.Fatalf("fake transcript misses the preamble: create(%d) subscribe(%d) setModel(%d)",
			scan.createIdx, scan.subIdx, scan.setModelIdx)
	}

	if scan.createIdx >= scan.subIdx || scan.subIdx >= scan.setModelIdx {
		t.Fatalf("back-leg wire order violated: create(%d) subscribe(%d) setModel(%d)",
			scan.createIdx, scan.subIdx, scan.setModelIdx)
	}

	if scan.sends != 2 {
		t.Fatalf("fake transcript records %d session/send frames, want exactly 2 (two loops)", scan.sends)
	}

	if scan.stops != 1 {
		t.Fatalf("fake transcript records %d session/stop frames, want exactly 1 (the loop-2 bypass)",
			scan.stops)
	}
}

// e2eWaitTranscript polls for the transcript file until the fake's
// EOF-drain exit lands it (bounded).
func e2eWaitTranscript(t *testing.T, path string) []e2eTranscriptLine {
	t.Helper()

	deadline := time.After(e2eTranscriptWait)

	for {
		loaded, err := e2eLoadTranscript(path)
		if err == nil {
			return loaded
		}

		select {
		case <-deadline:
			t.Fatalf("fake transcript never appeared at %s: %v", path, err)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// e2eScanTranscript classifies the transcript lines and asserts the
// setModel payload at its wire position.
func e2eScanTranscript(t *testing.T, lines []e2eTranscriptLine) e2eTranscriptScan {
	t.Helper()

	scan := e2eTranscriptScan{createIdx: -1, subIdx: -1, setModelIdx: -1}

	for i, line := range lines {
		switch {
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/create"`):
			scan.createIdx = i
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/subscribe"`):
			scan.subIdx = i
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/setModel"`):
			scan.setModelIdx = i
			scan.setModelJSON = line.json
			e2eAssertSetModelPayload(t, line.json)
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/send"`):
			scan.sends++
		case line.dir == "in" && strings.Contains(line.json, `"method":"session/stop"`):
			scan.stops++
		}
	}

	return scan
}

// e2eAssertSetModelPayload decodes the setModel request frame and asserts
// the payload verbatim: the --model/--reasoning-level flags landed as
// model.options.reasoningLevel (create drops it — the refusal trap), with
// explicit persistAsWorkspaceLastUsed:false.
func e2eAssertSetModelPayload(t *testing.T, raw string) {
	t.Helper()

	var frame struct {
		Params struct {
			Model struct {
				ProviderID string `json:"providerId"`
				ModelID    string `json:"modelId"`
				Options    struct {
					ReasoningLevel string `json:"reasoningLevel"`
				} `json:"options"`
			} `json:"model"`
			PersistAsWorkspaceLastUsed *bool `json:"persistAsWorkspaceLastUsed"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode setModel frame: %v", err)
	}

	model := frame.Params.Model
	if model.ProviderID != e2eProvider || model.ModelID != e2eModel {
		t.Fatalf("setModel model = %s/%s, want %s/%s (the --model flags): %s",
			model.ProviderID, model.ModelID, e2eProvider, e2eModel, raw)
	}

	if model.Options.ReasoningLevel != e2eLevel {
		t.Fatalf("setModel reasoningLevel = %q, want %q (the --reasoning-level flag): %s",
			model.Options.ReasoningLevel, e2eLevel, raw)
	}

	if frame.Params.PersistAsWorkspaceLastUsed == nil || *frame.Params.PersistAsWorkspaceLastUsed {
		t.Fatalf("setModel persistAsWorkspaceLastUsed must be explicitly false: %s", raw)
	}
}

// e2eTranscriptLine is one dir-tagged line of the fake's transcript file
// (the golden format: dir + TAB + raw JSON).
type e2eTranscriptLine struct {
	dir  string
	json string
}

// e2eLoadTranscript reads the transcript file; a missing file (the fake has
// not exited yet) reports the error so the caller can poll.
func e2eLoadTranscript(path string) ([]e2eTranscriptLine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := []e2eTranscriptLine{}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		dir, jsonLine, ok := strings.Cut(scanner.Text(), "\t")
		if !ok {
			return nil, errTranscriptRecord
		}

		lines = append(lines, e2eTranscriptLine{dir: dir, json: jsonLine})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, errTranscriptEmpty
	}

	return lines, nil
}

// e2eKillGroup is the best-effort cleanup: SIGKILL the wrapper's process
// group (it leads its own group — Setpgid above) so an early failure leaves
// neither the wrapper nor its fake child behind. ESRCH is the normal-path
// no-op (the wrapper already exited cleanly).
func e2eKillGroup(t *testing.T, wrapper *exec.Cmd, stderrBuf *e2eSyncBuffer) func() {
	t.Helper()

	return func() {
		if t.Failed() {
			stderrBuf.dump(t)
		}

		if wrapper.Process == nil {
			return
		}

		if err := syscall.Kill(-wrapper.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			t.Logf("kill wrapper group: %v", err)
		}
	}
}

// e2eSyncBuffer is a goroutine-safe stderr capture for the wrapper; dumped
// to the test log only on failure (CI stays quiet on green).
type e2eSyncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends to the buffer.
func (b *e2eSyncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

// String snapshots the captured stderr.
func (b *e2eSyncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

// dump logs the captured stderr.
func (b *e2eSyncBuffer) dump(t *testing.T) {
	t.Helper()

	b.mu.Lock()
	defer b.mu.Unlock()

	t.Logf("wrapper stderr:\n%s", b.buf.String())
}

// e2eCollector collects (sessionId, update) pairs under a mutex — the
// package-main twin of the acpfront harness's RecordingClient, stripped to
// what the ride asserts.
type e2eCollector struct {
	mu      sync.Mutex
	updates []e2eUpdate
	// elicitations records the ask-the-user bridge requests in arrival order.
	elicitations []acp.CreateElicitationRequest
}

// e2eUpdate is one collected update.
type e2eUpdate struct {
	sessionID string
	text      string
}

// SessionUpdate collects one streamed update.
func (c *e2eCollector) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	_ = ctx

	text := ""
	switch {
	case params.Update.AgentMessageChunk != nil:
		if block := params.Update.AgentMessageChunk.Content.Text; block != nil {
			text = block.Text
		}
	case params.Update.AgentThoughtChunk != nil:
		if block := params.Update.AgentThoughtChunk.Content.Text; block != nil {
			text = block.Text
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.updates = append(c.updates, e2eUpdate{sessionID: string(params.SessionId), text: text})

	return nil
}

// CompleteElicitation is a no-op (elicitation is Phase 5+).
func (c *e2eCollector) CompleteElicitation(ctx context.Context, params acp.CompleteElicitationNotification) error {
	_, _ = ctx, params

	return nil
}

// CreateElicitation answers the v0.22.0 ask-the-user bridge: the request
// is recorded (the form mapping assert) and DECLINED — the scripted fake
// expects the schema-valid decline reply, which now crosses the bridge
// instead of the legacy stub.
func (c *e2eCollector) CreateElicitation(
	ctx context.Context,
	params acp.CreateElicitationRequest,
) (acp.CreateElicitationResponse, error) {
	_ = ctx

	c.mu.Lock()
	c.elicitations = append(c.elicitations, params)
	c.mu.Unlock()

	return acp.CreateElicitationResponse{Decline: &acp.CreateElicitationDecline{Action: "decline"}}, nil
}

// ReadTextFile is a no-op (fs surfaces are never advertised).
func (c *e2eCollector) ReadTextFile(
	ctx context.Context,
	params acp.ReadTextFileRequest,
) (acp.ReadTextFileResponse, error) {
	_, _ = ctx, params

	return acp.ReadTextFileResponse{}, nil
}

// WriteTextFile is a no-op (fs surfaces are never advertised).
func (c *e2eCollector) WriteTextFile(
	ctx context.Context,
	params acp.WriteTextFileRequest,
) (acp.WriteTextFileResponse, error) {
	_, _ = ctx, params

	return acp.WriteTextFileResponse{}, nil
}

// RequestPermission is a no-op (permission bridging is Phase 5).
func (c *e2eCollector) RequestPermission(
	ctx context.Context,
	params acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
	_, _ = ctx, params

	return acp.RequestPermissionResponse{}, nil
}

// CreateTerminal is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) CreateTerminal(
	ctx context.Context,
	params acp.CreateTerminalRequest,
) (acp.CreateTerminalResponse, error) {
	_, _ = ctx, params

	return acp.CreateTerminalResponse{}, nil
}

// KillTerminal is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) KillTerminal(
	ctx context.Context,
	params acp.KillTerminalRequest,
) (acp.KillTerminalResponse, error) {
	_, _ = ctx, params

	return acp.KillTerminalResponse{}, nil
}

// TerminalOutput is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) TerminalOutput(
	ctx context.Context,
	params acp.TerminalOutputRequest,
) (acp.TerminalOutputResponse, error) {
	_, _ = ctx, params

	return acp.TerminalOutputResponse{}, nil
}

// ReleaseTerminal is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) ReleaseTerminal(
	ctx context.Context,
	params acp.ReleaseTerminalRequest,
) (acp.ReleaseTerminalResponse, error) {
	_, _ = ctx, params

	return acp.ReleaseTerminalResponse{}, nil
}

// WaitForTerminalExit is a no-op (terminal surfaces are never advertised).
func (c *e2eCollector) WaitForTerminalExit(
	ctx context.Context,
	params acp.WaitForTerminalExitRequest,
) (acp.WaitForTerminalExitResponse, error) {
	_, _ = ctx, params

	return acp.WaitForTerminalExitResponse{}, nil
}

// DrainUpdates snapshots the collected updates.
func (c *e2eCollector) DrainUpdates() []e2eUpdate {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]e2eUpdate, len(c.updates))
	copy(out, c.updates)

	return out
}

// hasUpdateText reports whether any collected update carries want.
func (c *e2eCollector) hasUpdateText(want string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, u := range c.updates {
		if u.text == want {
			return true
		}
	}

	return false
}

// Test-surface string constants (the goconst discipline): the omp argv
// tokens and the matrix's model ids repeat across the ride/scan vectors.
const (
	ompFlagVersion   = "--version"
	ompFlagAppServer = "--app-server"
	ompFlagModel     = "--model"
	ompFlagLevel     = "--reasoning-level"
	ompCardMarker    = "acp"
	ompModelShort    = "zai-api/GLM"
	ompModelLower    = "zai-api/glm-5.3"
	ompModelBadShape = "no-slash"
)

// semverTriple is the parsed major.minor.patch of a version answer.
type semverTriple struct {
	major, minor, patch int
}

// parseSemverLine parses the probe's one output line — the exact format the
// probe parses around (`zcode-acp <semver>`, build metadata tolerated) —
// and reports the triple; ok=false on any format deviation.
func parseSemverLine(line string) (semverTriple, bool) {
	prefix, ver, ok := strings.Cut(line, " ")
	if !ok || prefix != "zcode-acp" {
		return semverTriple{}, false
	}

	ver, _, _ = strings.Cut(ver, "+") // build metadata never compares

	majorStr, rest, ok := strings.Cut(ver, ".")
	if !ok {
		return semverTriple{}, false
	}

	minorStr, patchStr, ok := strings.Cut(rest, ".")
	if !ok {
		return semverTriple{}, false
	}

	// A prerelease suffix parses away (the client's semver reader reads
	// `0.0.0-dev` as 0.0.0 — the tooOld shape the phase replaces).
	patchStr, _, _ = strings.Cut(patchStr, "-")

	major, majorErr := strconv.Atoi(majorStr)
	minor, minorErr := strconv.Atoi(minorStr)
	patch, patchErr := strconv.Atoi(patchStr)
	if majorErr != nil || minorErr != nil || patchErr != nil {
		return semverTriple{}, false
	}

	return semverTriple{major: major, minor: minor, patch: patch}, true
}

// ompVersionAtLeast reports whether the got triple meets the compatibility
// floor the phase stamps (plan 07-04, D-03): the client's probe blocks the
// card below 17.0.5 outright and model selectors resolve only from 18.1.9
// — the locked stamp decision prints the 18.1.9 floor.
func ompVersionAtLeast(got semverTriple) bool {
	const (
		floorMajor = 18
		floorMinor = 1
		floorPatch = 9
	)

	if got.major != floorMajor {
		return got.major > floorMajor
	}

	if got.minor != floorMinor {
		return got.minor > floorMinor
	}

	return got.patch >= floorPatch
}

// ompRunVersionProbe runs the built binary with argv and returns its stdout
// (the client probe reads stdout with a 4096-byte cap inside a 5s bound —
// the wait bound mirrors that). The app-server env override names a
// nonexistent path so a pre-dispatch fall-through to acp mode fails fast at
// discovery — deterministically side-effect-free (no real desktop spawn,
// the CI blocker rule), which is exactly the pre-implementation shape.
func ompRunVersionProbe(t *testing.T, bin string, argv []string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), e2eUpdateWait)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, argv...)
	cmd.Env = append(os.Environ(), "ZCODE_ACP_APP_SERVER="+filepath.Join(t.TempDir(), "no-such-install"))

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%v: %v (stdout %q)", argv, err, string(out))
	}

	return strings.TrimSpace(string(out))
}

// ompAssertVersionFloor asserts the probe's one-line answer: the
// `zcode-acp <semver>` format and a semver >= the 18.1.9 floor.
func ompAssertVersionFloor(t *testing.T, out string) {
	t.Helper()

	lines := strings.Split(out, "\n")
	if len(lines) != 1 {
		t.Fatalf("version answer = %q, want exactly one stdout line", out)
	}

	got, ok := parseSemverLine(lines[0])
	if !ok {
		t.Fatalf("version answer %q does not parse as the `zcode-acp <semver>` line", lines[0])
	}

	if !ompVersionAtLeast(got) {
		t.Fatalf("version answer %q parses to %d.%d.%d, want >= the 18.1.9 floor",
			lines[0], got.major, got.minor, got.patch)
	}
}

// TestOmpVersionProbeFloor (plan 07-04 task 1): the built binary answers the
// client's version probe under the omp subcommand in ANY position —
// `omp --version` (the probe shape: prefix args + --version, card args
// absent) and `omp acp --version` (the any-position rule) — with a semver
// >= the 18.1.9 floor on one stdout line. Pre-implementation every omp
// shape falls through to root acp mode (stdlib flag stops at the `omp`
// positional, Pitfall 3), so the probe never answers.
func TestOmpVersionProbeFloor(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")

	for _, argv := range [][]string{
		{ompSubcommand, ompFlagVersion},
		{ompSubcommand, ompCardMarker, ompFlagVersion},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			ompAssertVersionFloor(t, ompRunVersionProbe(t, agentBin, argv))
		})
	}
}

// TestRootVersionFloor locks the root arm's floor parity: the root
// `--version` answer carries the SAME >= 18.1.9 semver (the probe also hits
// the root shape when no prefix args are configured) and the format stays
// `zcode-acp <semver>`.
func TestRootVersionFloor(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")

	ompAssertVersionFloor(t, ompRunVersionProbe(t, agentBin, []string{ompFlagVersion}))
}

// ompBindingScenario is the omp launch-binding ride's scenario (path
// relative to this test's dir): the create reply carries the account
// settings snapshot whose GLM-5.3-Flash entry pins defaultLevel "high" —
// the backend-confirmed default the launch binding derives when no explicit
// level rides (D-03's level-default chain). The GLM-5.3 entry carries no
// reasoning block — the derivation's absent-level arm.
const ompBindingScenario = "testdata/omp-binding-e2e.json"

// ompSetModelPayload is the decoded setModel assertion surface: the bound
// ids, whether an options object rode at all (the never-fabricated arm),
// its level when present, and the explicit persistAsWorkspaceLastUsed:false
// lock.
type ompSetModelPayload struct {
	providerID     string
	modelID        string
	optionsPresent bool
	reasoningLevel string
	persistFalse   bool
}

// ompRunAcpRide spawns the built binary with argv (the ride appends the
// --app-server override), drives initialize + session/new — the launch
// binding fires inside session/new's post-create follow-up — then closes
// stdin for the clean staged shutdown and returns the fake transcript plus
// the wrapper's captured stderr (the dispatch discriminator: only the omp
// arm logs the "zcode-acp omp:" namespace). The fake install root rides
// BOTH the --app-server flag and the env override — the flag wins
// post-dispatch (root-tested precedence), while the env keeps any
// pre-dispatch fall-through hermetic (the fake, never a real desktop).
func ompRunAcpRide(
	t *testing.T, agentBin, fakeBin string, argv []string, scenario string,
) (lines []e2eTranscriptLine, stderr string) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(scenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", scenario, err)
	}

	transcriptPath := filepath.Join(t.TempDir(), "fake-transcript.txt")
	installRoot := t.TempDir()
	e2eWriteFakeInstallRoot(t, installRoot, fakeBin, scenarioAbs, transcriptPath)

	rideCtx, rideCancel := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer rideCancel()

	wrapper := exec.CommandContext(rideCtx, agentBin, append(argv, ompFlagAppServer, installRoot)...)
	wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	wrapper.Env = append(os.Environ(),
		"ZCODE_ACP_APP_SERVER="+installRoot,
		"ZCODE_ACP_MODEL=", "ZCODE_ACP_REASONING_LEVEL=", "ZCODE_ACP_API_KEY=")

	stderrBuf := &e2eSyncBuffer{}
	wrapper.Stderr = stderrBuf

	stdin, err := wrapper.StdinPipe()
	if err != nil {
		t.Fatalf("wrapper stdin pipe: %v", err)
	}

	stdout, err := wrapper.StdoutPipe()
	if err != nil {
		t.Fatalf("wrapper stdout pipe: %v", err)
	}

	if err := wrapper.Start(); err != nil {
		t.Fatalf("spawn %s: %v", agentBin, err)
	}

	t.Cleanup(e2eKillGroup(t, wrapper, stderrBuf))

	updates := &e2eCollector{}
	client := acp.NewClientSideConnection(updates, stdin, stdout)

	e2eInitialize(rideCtx, t, client)
	e2eNewSession(rideCtx, t, client)

	if err := stdin.Close(); err != nil {
		t.Logf("close wrapper stdin: %v", err)
	}

	e2eWaitExit(t, wrapper)

	return e2eWaitTranscript(t, transcriptPath), stderrBuf.String()
}

// TestOmpFlagSet is the omp argv contract's unit matrix over ompParse (the
// ompFlagSet pass): the card surface re-tokenizes after the `acp`
// positional — tolerance arms answer no error, and the config arms compose
// into parseConfig's validation (the SAME sentinels as the root arm).
func TestOmpFlagSet(t *testing.T) {
	t.Run("unknown flags tolerated and counted", testOmpFlagSetUnknownTolerated)

	t.Run("unknown flag without a separate value tolerated", testOmpFlagSetUnknownValueless)

	t.Run("the bare acp positional is ignored", testOmpFlagSetBarePositional)

	t.Run("plugin-dir and session-dir accepted and ignored", testOmpFlagSetCardFlagsIgnored)

	t.Run("version answers in any position", testOmpFlagSetVersionAnyPosition)

	t.Run("bad model shape answers the honest errModelShape error", testOmpFlagSetBadShape)

	t.Run("level without model answers the existing sentinel", testOmpFlagSetLevelWithoutModel)

	t.Run("the omp parse composes into parseConfig with env fallbacks", testOmpFlagSetEnvCompose)

	t.Run("app-server help names the packed AppImage shape", testOmpFlagSetAppServerHelp)
}

// testOmpFlagSetAppServerHelp locks the user-facing flag description to the
// shipped override contract: a directory, an app-server binary, OR a packed
// AppImage. A help string that omits the packed arm misdescribes the feature.
func testOmpFlagSetAppServerHelp(t *testing.T) {
	fs, _ := ompFlagSet()

	flag := fs.Lookup("app-server")
	if flag == nil {
		t.Fatal("the omp flag set carries no --app-server flag")
	}

	for _, want := range []string{"install root", "app-server binary", "packed AppImage"} {
		if !strings.Contains(flag.Usage, want) {
			t.Errorf("--app-server help = %q, want it to name %q", flag.Usage, want)
		}
	}
}

// testOmpFlagSetUnknownTolerated locks the tolerance contract: an unknown
// flag plus its value token is skipped and counted, and the following
// known flag still parses.
func testOmpFlagSetUnknownTolerated(t *testing.T) {
	opts, err := ompParse([]string{ompCardMarker, "--future-flag", "v", ompFlagModel, ompModelShort})
	if err != nil {
		t.Fatalf("the unknown-flag tolerance contract broke: %v", err)
	}

	if opts.unknownCount != 1 {
		t.Fatalf("unknownCount = %d, want 1 (one tolerated flag)", opts.unknownCount)
	}

	if opts.model != ompModelShort {
		t.Fatalf("model = %q, want the flag parsed after the tolerated unknown", opts.model)
	}
}

// testOmpFlagSetUnknownValueless locks the value-less unknown arm: a dash
// token following an unknown flag is never consumed as its value.
func testOmpFlagSetUnknownValueless(t *testing.T) {
	opts, err := ompParse([]string{"--future-flag", ompFlagModel, ompModelShort})
	if err != nil {
		t.Fatalf("value-less unknown flag must tolerate: %v", err)
	}

	if opts.unknownCount != 1 || opts.model != ompModelShort {
		t.Fatalf("unknownCount = %d model = %q, want 1 and the parsed model", opts.unknownCount, opts.model)
	}
}

// testOmpFlagSetBarePositional locks the card-marker arm: the bare `acp`
// positional parses clean and leaks nothing into the config.
func testOmpFlagSetBarePositional(t *testing.T) {
	opts, err := ompParse([]string{ompCardMarker})
	if err != nil {
		t.Fatalf("the card's acp marker must parse clean: %v", err)
	}

	if opts.version || opts.model != "" || opts.appServer != "" {
		t.Fatalf("the bare positional leaked into the config: %+v", opts)
	}
}

// testOmpFlagSetCardFlagsIgnored locks the accepted-and-ignored arm: the
// card's own flags register without error.
func testOmpFlagSetCardFlagsIgnored(t *testing.T) {
	opts, err := ompParse([]string{ompCardMarker, "--plugin-dir", "/plugins", "--session-dir", "/sessions"})
	if err != nil {
		t.Fatalf("the card's own flags must parse clean (accepted-and-ignored): %v", err)
	}

	if !opts.pluginDirSet || !opts.sessionDirSet {
		t.Fatalf("card flags not registered as accepted-ignored: %+v", opts)
	}
}

// testOmpFlagSetVersionAnyPosition locks the any-position version answer
// at the parse tier.
func testOmpFlagSetVersionAnyPosition(t *testing.T) {
	for _, argv := range [][]string{
		{ompSubcommand, ompFlagVersion},
		{ompSubcommand, ompCardMarker, ompFlagVersion},
	} {
		opts, err := ompParse(argv)
		if err != nil || !opts.version {
			t.Fatalf("%v: version = %v err = %v, want the any-position answer", argv, opts.version, err)
		}
	}
}

// testOmpFlagSetBadShape locks the honest-shape arm: the re-tokenizing
// parse succeeds and config validation answers the errModelShape wrap
// naming the got value.
func testOmpFlagSetBadShape(t *testing.T) {
	opts, err := ompParse([]string{ompCardMarker, ompFlagModel, ompModelBadShape})
	if err != nil {
		t.Fatalf("the shape error belongs to config validation, not the re-tokenizing parse: %v", err)
	}

	_, cfgErr := parseConfig(opts.appServer, opts.model, opts.reasoningLevel, opts.apiKey)
	if !errors.Is(cfgErr, errModelShape) {
		t.Fatalf("config err = %v, want the errModelShape sentinel", cfgErr)
	}

	if !strings.Contains(cfgErr.Error(), ompModelBadShape) {
		t.Fatalf("config err %q must name the got value", cfgErr)
	}
}

// testOmpFlagSetLevelWithoutModel locks the level-without-model arm: the
// parse succeeds and validation answers the existing sentinel.
func testOmpFlagSetLevelWithoutModel(t *testing.T) {
	opts, err := ompParse([]string{ompCardMarker, ompFlagLevel, "high"})
	if err != nil {
		t.Fatalf("the parse must succeed and let validation reject: %v", err)
	}

	_, cfgErr := parseConfig(opts.appServer, opts.model, opts.reasoningLevel, opts.apiKey)
	if !errors.Is(cfgErr, errLevelWithoutModel) {
		t.Fatalf("config err = %v, want the errLevelWithoutModel sentinel", cfgErr)
	}
}

// testOmpFlagSetEnvCompose locks the composition arm: the omp parse feeds
// parseConfig whose env fallbacks ride unchanged.
func testOmpFlagSetEnvCompose(t *testing.T) {
	t.Setenv(envModel, "zai-api/GLM-5.3-Flash")

	opts, err := ompParse([]string{ompCardMarker})
	if err != nil {
		t.Fatalf("bare card argv must parse clean: %v", err)
	}

	cfg, cfgErr := parseConfig(opts.appServer, opts.model, opts.reasoningLevel, opts.apiKey)
	if cfgErr != nil || cfg.model == nil ||
		cfg.model.ProviderID != providerZaiAPI || cfg.model.ModelID != e2eModel {
		t.Fatalf("env fallback lost under the omp arm: cfg %+v err %v", cfg, cfgErr)
	}
}

// ompFindSetModelLine returns the transcript line carrying the backend
// session/setModel request — ok=false when none fired (the silent-drop
// RED shape, Pitfall 3's warning sign).
func ompFindSetModelLine(lines []e2eTranscriptLine) (e2eTranscriptLine, bool) {
	for _, line := range lines {
		if line.dir == "in" && strings.Contains(line.json, `"method":"session/setModel"`) {
			return line, true
		}
	}

	return e2eTranscriptLine{}, false
}

// ompDecodeSetModel decodes one setModel frame into its assertion payload.
func ompDecodeSetModel(t *testing.T, raw string) ompSetModelPayload {
	t.Helper()

	var frame struct {
		Params struct {
			Model struct {
				ProviderID string `json:"providerId"`
				ModelID    string `json:"modelId"`
				Options    *struct {
					ReasoningLevel string `json:"reasoningLevel"`
				} `json:"options"`
			} `json:"model"`
			PersistAsWorkspaceLastUsed *bool `json:"persistAsWorkspaceLastUsed"`
		} `json:"params"`
	}

	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatalf("decode setModel frame: %v", err)
	}

	payload := ompSetModelPayload{
		providerID:   frame.Params.Model.ProviderID,
		modelID:      frame.Params.Model.ModelID,
		persistFalse: frame.Params.PersistAsWorkspaceLastUsed != nil && !*frame.Params.PersistAsWorkspaceLastUsed,
	}

	if frame.Params.Model.Options != nil {
		payload.optionsPresent = true
		payload.reasoningLevel = frame.Params.Model.Options.ReasoningLevel
	}

	return payload
}

// ompAssertBound asserts one decoded payload against the want ids and the
// option-shape expectation (wantLevel "" with wantOptions false = the
// omit-shape arm).
func ompAssertBound(
	t *testing.T, payload ompSetModelPayload, raw, provider, model string, wantOptions bool, wantLevel string,
) {
	t.Helper()

	if payload.providerID != provider || payload.modelID != model {
		t.Fatalf("setModel model = %s/%s, want %s/%s (the omp --model): %s",
			payload.providerID, payload.modelID, provider, model, raw)
	}

	if !payload.persistFalse {
		t.Fatalf("setModel persistAsWorkspaceLastUsed must be explicitly false: %s", raw)
	}

	if payload.optionsPresent != wantOptions {
		t.Fatalf("setModel options present = %v, want %v: %s", payload.optionsPresent, wantOptions, raw)
	}

	if payload.reasoningLevel != wantLevel {
		t.Fatalf("setModel reasoningLevel = %q, want %q: %s", payload.reasoningLevel, wantLevel, raw)
	}
}

// TestOmpLaunchBinding (plan 07-04 task 2): the built binary binds the omp
// card's --model at launch — the setModel follow-up fires on the backend
// wire after create + subscribe with the parsed provider/model (RED
// pre-wiring: the stub drops the flag and no setModel ever fires), the
// reasoning level derives from the create-reply snapshot's defaultLevel
// when no explicit level rides (never a fabricated level), and an explicit
// level wins.
func TestOmpLaunchBinding(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("the omp --model rides the setModel follow-up verbatim", func(t *testing.T) {
		lines, _ := ompRunAcpRide(t, agentBin, fakeBin,
			[]string{ompSubcommand, ompCardMarker, ompFlagModel, ompModelLower}, e2eScenario)

		line, ok := ompFindSetModelLine(lines)
		if !ok {
			t.Fatal("no session/setModel fired — the omp --model was silently dropped (Pitfall 3)")
		}

		// The settings-less scenario's snapshot lists nothing and no
		// explicit level rode: the omit-shape is the honest answer.
		ompAssertBound(t, ompDecodeSetModel(t, line.json), line.json, providerZaiAPI, "glm-5.3", false, "")
	})

	t.Run("the reasoning default derives from the create-reply snapshot", func(t *testing.T) {
		lines, _ := ompRunAcpRide(t, agentBin, fakeBin,
			[]string{ompSubcommand, ompCardMarker, ompFlagModel, "zai-api/GLM-5.3-Flash"}, ompBindingScenario)

		line, ok := ompFindSetModelLine(lines)
		if !ok {
			t.Fatal("no session/setModel fired — the omp --model was silently dropped (Pitfall 3)")
		}

		ompAssertBound(t, ompDecodeSetModel(t, line.json), line.json,
			providerZaiAPI, e2eModel, true, "high")
	})

	t.Run("an explicit level wins over the snapshot default", func(t *testing.T) {
		lines, _ := ompRunAcpRide(t, agentBin, fakeBin, []string{
			ompSubcommand, ompCardMarker, ompFlagModel,
			"zai-api/GLM-5.3-Flash", ompFlagLevel, "low"}, ompBindingScenario)

		line, ok := ompFindSetModelLine(lines)
		if !ok {
			t.Fatal("no session/setModel fired — the omp --model was silently dropped (Pitfall 3)")
		}

		ompAssertBound(t, ompDecodeSetModel(t, line.json), line.json,
			providerZaiAPI, e2eModel, true, "low")
	})
}

// ompRunExpectExitCode runs the built binary with argv (hermetic: the env
// override names a nonexistent path so no arm can fall into production
// discovery), asserts the exact exit class, and returns the wrapper's
// stderr for the honest-message assertions.
func ompRunExpectExitCode(t *testing.T, bin string, argv []string, wantCode int) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), e2eUpdateWait)
	defer cancel()

	stderrBuf := &e2eSyncBuffer{}

	cmd := exec.CommandContext(ctx, bin, argv...)
	cmd.Env = append(os.Environ(), "ZCODE_ACP_APP_SERVER="+filepath.Join(t.TempDir(), "no-such-install"))
	cmd.Stderr = stderrBuf

	err := cmd.Run()

	exitErr := &exec.ExitError{}
	if !errors.As(err, &exitErr) {
		t.Fatalf("%v: err = %v, want an ExitError carrying the %d class", argv, err, wantCode)
	}

	if code := exitErr.ExitCode(); code != wantCode {
		t.Fatalf("%v: exit = %d, want %d (stderr %q)", argv, code, wantCode, stderrBuf.String())
	}

	return stderrBuf.String()
}

// TestRootModeUnchanged (plan 07-04 task 3): the pre-phase root vectors
// still pass with the omp arm present in the binary — root --version, root
// --model acp mode (the setModel follow-up binds at the root path too, with
// no omp-arm log lines on stderr), and the usage exit class on a bad root
// config. The wiring seams test (TestWiringCodingPlanSeams) runs in-suite
// unchanged.
func TestRootModeUnchanged(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("root --version prints the stamped floor line", func(t *testing.T) {
		ompAssertVersionFloor(t, ompRunVersionProbe(t, agentBin, []string{ompFlagVersion}))
	})

	t.Run("root --model acp mode still binds via the setModel follow-up", func(t *testing.T) {
		lines, stderr := ompRunAcpRide(t, agentBin, fakeBin,
			[]string{ompFlagModel, e2eProvider + "/" + e2eModel, ompFlagLevel, e2eLevel}, ompBindingScenario)

		line, ok := ompFindSetModelLine(lines)
		if !ok {
			t.Fatal("no session/setModel fired — the root --model path regressed")
		}

		ompAssertBound(t, ompDecodeSetModel(t, line.json), line.json, e2eProvider, e2eModel, true, e2eLevel)

		if strings.Contains(stderr, "zcode-acp omp:") {
			t.Fatalf("the root path logged omp-arm lines — the dispatch switched arms: %s", stderr)
		}
	})

	t.Run("bad root config exits with the usage class", func(t *testing.T) {
		stderr := ompRunExpectExitCode(t, agentBin, []string{ompFlagModel, ompModelBadShape}, exitInvalidConfig)

		if !strings.Contains(stderr, "--model must be providerId/modelId") ||
			!strings.Contains(stderr, ompModelBadShape) {
			t.Fatalf("root usage error %q must name the shape sentinel and the got value", stderr)
		}
	})
}

// TestOmpRootCoexistence locks the dispatch switch: the first positional is
// the ONLY switch — the same --model binding rides either arm, the unit
// scan proves the value-token rule (a flag value equal to omp never
// hijacks the arm), and the omp arm's stderr namespace discriminates the
// two paths.
func TestOmpRootCoexistence(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	fakeBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/fake-app-server", "fake-app-server")

	t.Run("first-positional dispatch is the only switch (unit scan)", func(t *testing.T) {
		for _, tc := range []struct {
			args []string
			want bool
		}{
			{[]string{ompSubcommand, ompCardMarker, ompFlagModel, ompModelShort}, true},
			{[]string{ompSubcommand, ompFlagVersion}, true},
			{[]string{ompFlagModel, ompModelShort}, false},
			{[]string{ompFlagModel, ompSubcommand}, false}, // the flag's value token — never a positional
			{[]string{ompFlagAppServer, "/x", ompSubcommand, ompFlagVersion}, true},
			{[]string{}, false},
			{[]string{ompCardMarker, ompFlagModel, ompModelShort}, false}, // the bare card marker alone is not omp
			// after the terminator everything is positional
			{[]string{"--", ompSubcommand}, true},
		} {
			if got := ompRequested(tc.args); got != tc.want {
				t.Fatalf("ompRequested(%v) = %v, want %v (first-positional dispatch is the only switch)",
					tc.args, got, tc.want)
			}
		}
	})

	t.Run("root --model rides the root path", func(t *testing.T) {
		lines, stderr := ompRunAcpRide(t, agentBin, fakeBin,
			[]string{ompFlagModel, ompModelLower}, e2eScenario)

		line, ok := ompFindSetModelLine(lines)
		if !ok {
			t.Fatal("no session/setModel fired — the root --model path regressed")
		}

		ompAssertBound(t, ompDecodeSetModel(t, line.json), line.json, providerZaiAPI, "glm-5.3", false, "")

		if strings.Contains(stderr, "zcode-acp omp:") {
			t.Fatalf("root argv must not log omp-arm lines: %s", stderr)
		}
	})

	t.Run("omp acp --model rides the omp arm", func(t *testing.T) {
		lines, stderr := ompRunAcpRide(t, agentBin, fakeBin,
			[]string{ompSubcommand, ompCardMarker, ompFlagModel, ompModelLower}, e2eScenario)

		line, ok := ompFindSetModelLine(lines)
		if !ok {
			t.Fatal("no session/setModel fired — the omp arm's binding regressed")
		}

		ompAssertBound(t, ompDecodeSetModel(t, line.json), line.json, providerZaiAPI, "glm-5.3", false, "")

		if !strings.Contains(stderr, "zcode-acp omp: model selection bound at launch") {
			t.Fatalf("the omp arm must announce the launch binding (ids/shapes only): %s", stderr)
		}
	})
}

// TestOmpExitCodes locks the exit-code classes: a malformed omp config
// exits with the SAME usage class as the root path (exitInvalidConfig,
// distinct from the runtime exit 1) and the honest message names the
// contract; runtime failures stay the exit-1 class on both arms.
func TestOmpExitCodes(t *testing.T) {
	agentBin := e2eBuildBinary(t, "github.com/Djarvur/zcode-acp/cmd/zcode-acp", "zcode-acp")
	noSuchInstall := filepath.Join(t.TempDir(), "no-such-install")

	t.Run("malformed omp model shape exits with the usage class", func(t *testing.T) {
		stderr := ompRunExpectExitCode(t, agentBin,
			[]string{ompSubcommand, ompCardMarker, ompFlagModel, ompModelBadShape}, exitInvalidConfig)

		// The slog text handler escapes quotes inside the err value —
		// assert the sentinel and the got value by their stable fragments.
		if !strings.Contains(stderr, "--model must be providerId/modelId") ||
			!strings.Contains(stderr, ompModelBadShape) {
			t.Fatalf("omp usage error %q must name the shape sentinel and the got value", stderr)
		}
	})

	t.Run("omp level without model exits with the usage class", func(t *testing.T) {
		stderr := ompRunExpectExitCode(t, agentBin,
			[]string{ompSubcommand, ompCardMarker, ompFlagLevel, "high"}, exitInvalidConfig)

		if !strings.Contains(stderr, "--reasoning-level set without --model") {
			t.Fatalf("omp usage error %q must name the level-without-model sentinel", stderr)
		}
	})

	t.Run("root bad config exits with the same usage class", func(t *testing.T) {
		stderr := ompRunExpectExitCode(t, agentBin, []string{ompFlagModel, ompModelBadShape}, exitInvalidConfig)

		if !strings.Contains(stderr, "--model must be providerId/modelId") {
			t.Fatalf("root usage error %q must name the shape sentinel", stderr)
		}
	})

	t.Run("omp runtime failure stays the distinct exit-1 class", func(t *testing.T) {
		stderr := ompRunExpectExitCode(t, agentBin,
			[]string{ompSubcommand, ompCardMarker, ompFlagAppServer, noSuchInstall}, 1)

		// The nonexistent-install override rides discovery's step 1 and
		// fails loud at spawn — the honest runtime failure framing (the
		// usage class never fires; exit 1 is the distinct runtime class).
		if !strings.Contains(stderr, "zcode-acp: exiting after failure") {
			t.Fatalf("runtime failure %q must carry the exiting-after-failure framing", stderr)
		}
	})

	t.Run("root runtime failure stays the distinct exit-1 class", func(t *testing.T) {
		stderr := ompRunExpectExitCode(t, agentBin, []string{ompFlagAppServer, noSuchInstall}, 1)

		if !strings.Contains(stderr, "zcode-acp: exiting after failure") {
			t.Fatalf("runtime failure %q must carry the exiting-after-failure framing", stderr)
		}
	})
}

// testWiringMachineryLazily locks D-13's construction posture: every coding-plan seam constructs without executing
// (no keyring, no mint, no gate read).
func testWiringMachineryLazily(t *testing.T) {
	machinery := buildCodingPlanMachinery(wiringFixtureInstall())

	if machinery.creds == nil || machinery.prober == nil || machinery.gate == nil || machinery.minter == nil {
		t.Fatalf("a coding-plan seam is nil: %+v", machinery)
	}
}

// testWiringClientVersion locks the A6 clientVersion seam: the install tree's version segment wins, the documented
// fallback covers the version-less layouts.
func testWiringClientVersion(t *testing.T) {
	if got := patClientVersion(wiringFixtureInstall())(); got != wiringVersionSegment {
		t.Fatalf("patClientVersion = %q, want the fixture tree's %q", got, wiringVersionSegment)
	}

	bare := supervisor.Install{Root: optDir, Binary: optDir + "/zcode"} // the deb layout carries no version segment
	if got := patClientVersion(bare)(); got != patDesktopVersionFallback {
		t.Fatalf("patClientVersion(bare) = %q, want the documented %q fallback", got, patDesktopVersionFallback)
	}
}

// mountedFixtureVersion is the version the mount-first fixtures carry.
const mountedFixtureVersion = "4.0.0"

// TestDesktopVersionMounted locks the mount-safe PAT client-version seam
// (MNT-04, D-04): a mounted root lives under a versionless /tmp .mount_* path,
// so the version must come from Install.Version or the packed AppImage
// basename — never the documented 0.0.0 fallback once a version is knowable.
func TestDesktopVersionMounted(t *testing.T) {
	t.Run("mounted install reports Install.Version", func(t *testing.T) {
		mountRoot := "/tmp/zcode-acp-mount-1700000000/.mount_ZCode-AbCdEf"
		inst := supervisor.Install{
			Root:       mountRoot,
			Binary:     filepath.Join(mountRoot, "zcode"),
			Entrypoint: filepath.Join(mountRoot, "resources", "glm", "zcode.cjs"),
			Layout:     "appimage",
			Version:    mountedFixtureVersion,
		}

		if got := desktopVersion(inst); got != mountedFixtureVersion {
			t.Fatalf("desktopVersion(mounted) = %q, want Install.Version %s (never the 0.0.0 fallback)",
				got, mountedFixtureVersion)
		}
		if got := patClientVersion(inst)(); got != mountedFixtureVersion {
			t.Fatalf("patClientVersion(mounted) = %q, want %s", got, mountedFixtureVersion)
		}
	})

	t.Run("pre-mount packed install reports the basename version", func(t *testing.T) {
		packed := supervisor.Install{
			AppImage: "/tmp/apps/ZCode-" + mountedFixtureVersion + "-linux-x64.AppImage",
			Layout:   "appimage",
		}
		if got := desktopVersion(packed); got != mountedFixtureVersion {
			t.Fatalf("desktopVersion(packed) = %q, want the AppImage basename %s",
				got, mountedFixtureVersion)
		}
		packed.Version = mountedFixtureVersion
		if got := desktopVersion(packed); got != mountedFixtureVersion {
			t.Fatalf("desktopVersion(packed with Version) = %q, want %s", got, mountedFixtureVersion)
		}
	})

	t.Run("bare install keeps the documented fallback", func(t *testing.T) {
		bare := supervisor.Install{Root: optDir, Binary: optDir + "/zcode"}
		if got := desktopVersion(bare); got != "" {
			t.Fatalf("desktopVersion(bare) = %q, want no version", got)
		}
		if got := patClientVersion(bare)(); got != patDesktopVersionFallback {
			t.Fatalf("patClientVersion(bare) = %q, want the documented %q fallback", got, patDesktopVersionFallback)
		}
	})
}

// testWiringBuildFrontSeams locks buildFront's AuthConfig wiring: the front constructs over the machinery with the
// mode record still unset.
func testWiringBuildFrontSeams(t *testing.T) {
	hostEnd, peerEnd := net.Pipe()
	t.Cleanup(func() {
		_ = hostEnd.Close()
		_ = peerEnd.Close()
	})

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	machinery := buildCodingPlanMachinery(wiringFixtureInstall())
	keys := hostsvc.NewStaticKey("", "")

	front := buildFront(conn, config{}, wiringFixtureInstall(), keys, machinery)
	if front == nil {
		t.Fatal("buildFront returned nil")
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeNone {
		t.Fatalf("fresh front auth mode = %q, want %q (record-once discipline)", mode, acpfront.AuthModeNone)
	}
}

// testWiringModeAwareServing locks wireResponder's runtime-headers option: the mode-aware source is in the chain (the
// unauthenticated mode delegates to the static arm's unconfigured posture; the coding-plan arm's positives are the
// corpus row's proof).
func testWiringModeAwareServing(t *testing.T) {
	hostEnd, peerEnd := net.Pipe()
	t.Cleanup(func() {
		_ = hostEnd.Close()
		_ = peerEnd.Close()
	})

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	machinery := buildCodingPlanMachinery(wiringFixtureInstall())
	keys := hostsvc.NewStaticKey("", "")

	front := buildFront(conn, config{}, wiringFixtureInstall(), keys, machinery)
	wireResponder(conn, front, keys, machinery,
		interact.NewPermissionBridge(front.PermissionForward()).Forward, nil)

	if conn.OnRequest == nil {
		t.Fatal("wireResponder left conn.OnRequest unwired")
	}

	// The unauthenticated mode (None until authenticate succeeds)
	// delegates to the static arm's unconfigured posture: the coded
	// error naming ZCODE_ACP_API_KEY — proving the ask routes through
	// the mode-aware source (the old wiring served the same error only
	// via the raw static handler; the discriminator here is that the
	// mode-aware source is IN the chain, and the coding-plan arm's
	// positive answers are the corpus row's byte-locked proof).
	env := zcode.Envelope{
		ID:     &zcode.ID{Str: "wire-1"},
		Method: interact.MethodRequestProviderRuntimeHeaders,
		Params: json.RawMessage(`{"requestId":"wire-1","sessionId":"sess_w1","reason":"model-request"}`),
	}

	_, werr, handled := conn.OnRequest(context.Background(), env)
	if !handled {
		t.Fatal("the runtime-headers ask must be handled by the responder, never silence")
	}

	if werr == nil || werr.Code != zcode.CodeInternalError ||
		!strings.Contains(werr.Message, "ZCODE_ACP_API_KEY") {
		t.Fatalf("unauthenticated-mode ask answered (%v, %+v), want the static "+
			"arm's coded unconfigured posture", werr == nil, werr)
	}
}
