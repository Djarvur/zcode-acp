// Process-group lifecycle tests (BRIDGE-03 part 2, plan 03-07 Task 2): the
// staged shutdown (stdin end → wait → grace → group SIGKILL → reap, probe
// precedent :638-653) kills the WHOLE process group with zero survivors —
// grandchildren included (T-3-09, PITFALL 6: never trust stdin-EOF on
// Electron) — and the crash-window restart policy (D-03: 3 attempts within a
// sliding window, fail loud past the limit, stable runs reset the counter —
// RESEARCH Pitfall 7, absent from both analogs, added here). The child is
// the re-executed test binary (reexec idiom, 03-06 precedent); the real
// Electron bundle never runs in CI (Pitfall 10).

package supervisor_test

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Lifecycle-facts constants under test.
const (
	// superviseHelperModeEnv selects the helper child's behavior.
	superviseHelperModeEnv = "ZCODE_ACP_SUPERVISE_HELPER"
	// superviseHelperStayEnv carries the stay-alive duration (ms).
	superviseHelperStayEnv = "ZCODE_ACP_SUPERVISE_STAY_MS"

	// helperModeCrash exits immediately — the always-crashing child.
	helperModeCrash = "crash"
	// helperModeStay sleeps then exits — a stable run for the reset test.
	helperModeStay = "stay"
	// helperModeGrandchild spawns a sleeping grandchild (inheriting the
	// process group), reports its pid, and stays alive until stdin EOF.
	helperModeGrandchild = "grandchild"

	// superviseHelperRunFlag targets the helper inside the re-executed binary.
	superviseHelperRunFlag = "-test.run=^TestSuperviseHelper$"

	// grandchildSleepSecs keeps the grandchild alive well past the test so
	// only the group SIGKILL can reap it (PITFALL 6: stdin-EOF must not be
	// trusted to clean the tree).
	grandchildSleepSecs = "30"
	// grandchildReadyPrefix is the helper's readiness line ("ready <pid>").
	grandchildReadyPrefix = "ready "

	// restartWindowShort is the injectable crash window for tests
	// (production default 60 s per D-03; RESEARCH Pitfall 7).
	restartWindowShort = 150 * time.Millisecond
	// restartBackoff << (crashes-1) from BaseBackoff — kept tiny so the
	// restart tests run in milliseconds.
	restartBackoff = 1 * time.Millisecond
	// restartGraceShort is the injectable shutdown grace for tests
	// (production default: the probe's 4 s GRACE_MS).
	restartGraceShort = 100 * time.Millisecond

	// resetStayMs keeps the second child alive LONGER than the crash window
	// — the stable run whose exit must reset the sliding counter.
	resetStayMs = 400
	// stableSpawnOrdinal is which spawn number stays alive (after one crash).
	stableSpawnOrdinal = 2
	// resetTripSpawnCount pins the reset arithmetic: crash(1) → stable reset →
	// crashes 1'..4' trip → 1 + 1 + 4 spawns. Without the reset the trip
	// would come at spawn 5 (crashes 1,2,3,4 with the counter never zeroed).
	resetTripSpawnCount = 6
	// tripSpawnCount pins the plain trip arithmetic: 3 attempts within the
	// window, then fail loud on the 4th consecutive crash.
	tripSpawnCount = 4
	// tripCrashCount is the crash count the loud failure must name.
	tripCrashCount = "4"

	// restartWatchdog bounds a Start that fails to trip the limit (a looping
	// policy must fail the test fast, not hang the suite).
	restartWatchdog = 10 * time.Second

	// shutdownWatchdog bounds the staged shutdown end-to-end.
	shutdownWatchdog = 8 * time.Second

	// spawnSettleDeadline bounds the wait for the SpawnFunc's delivery.
	spawnSettleDeadline = 5 * time.Second

	// groupGoneDeadline / groupGonePoll bound the ESRCH poll: after the
	// group SIGKILL, dead members exist as zombies until reaped (the
	// grandchild by init), so zero-survivors is proven by polling until
	// kill reports ESRCH — never by a single racy check.
	groupGoneDeadline = 3 * time.Second
	groupGonePoll     = 10 * time.Millisecond
)

// TestSuperviseHelper is the -test.run target the parent re-executes as the
// child: crash (exit immediately), stay (sleep then exit), or grandchild
// (spawn a sleeping grandchild in the same process group, report its pid on
// stdout, and stay alive until stdin EOF).
func TestSuperviseHelper(t *testing.T) {
	switch os.Getenv(superviseHelperModeEnv) {
	case helperModeCrash:
		os.Exit(1)
	case helperModeStay:
		stayMs, _ := strconv.Atoi(os.Getenv(superviseHelperStayEnv))
		time.Sleep(time.Duration(stayMs) * time.Millisecond)
		os.Exit(1)
	case helperModeGrandchild:
		runGrandchildHelper()
	}
}

// runGrandchildHelper is the grandchild branch: spawn `sleep` WITHOUT a new
// process group (it inherits the helper's — that is exactly the tree the
// staged shutdown must reap), report its pid, and stay alive until stdin EOF.
func runGrandchildHelper() {
	grandchild := exec.CommandContext(context.Background(), "sleep", grandchildSleepSecs)
	if err := grandchild.Start(); err != nil {
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stdout, "%s%d\n", grandchildReadyPrefix, grandchild.Process.Pid)
	_, _ = io.Copy(io.Discard, os.Stdin) // stay alive until stdin EOF

	os.Exit(0)
}

// TestGroupKillNoSurvivors proves the staged shutdown leaves ZERO surviving
// group members: the supervisor starts the helper (which spawns a sleeping
// grandchild in its group), ctx cancel drives the staged shutdown, and the
// test asserts the child was reaped (Done closed, Exited delivered) AND both
// the group and the grandchild report ESRCH — PITFALL 6: stdin-EOF is never
// trusted, the group SIGKILL backstop is unconditional.
func TestGroupKillNoSurvivors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	spawnCh := make(chan *supervisor.Child, 1)

	sup := supervisor.NewSupervisor(func(ctx context.Context) (*supervisor.Child, error) {
		child, err := spawnSuperviseChild(ctx, helperModeGrandchild, 0)
		if err == nil {
			spawnCh <- child
		}

		return child, err
	}, supervisor.DefaultPolicy())

	runDone := make(chan error, 1)

	go func() {
		runDone <- sup.Start(ctx)
	}()

	child := waitSpawnedChild(t, spawnCh)
	grandchildPid := readGrandchildPid(t, child)

	cancel()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("staged shutdown Start: %v", err)
		}
	case <-time.After(shutdownWatchdog):
		t.Fatalf("staged shutdown did not complete within %v", shutdownWatchdog)
	}

	assertReaped(t, child)
	assertGone(t, "the process group", -child.PgID)
	assertGone(t, "the grandchild", grandchildPid)
}

// TestRestartWindow pins the crash-window restart policy (D-03, ARCHITECTURE
// Pattern 4, RESEARCH Pitfall 7) with injectable window knobs.
func TestRestartWindow(t *testing.T) {
	t.Run("trips loud past the limit", testRestartWindowTrip)
	t.Run("stable run resets the sliding window", testRestartWindowReset)
}

// testRestartWindowTrip: a helper child that exits immediately consumes the
// 3 in-window attempts and Start fails LOUD naming the crash count and the
// window — never a loop past the limit.
func testRestartWindowTrip(t *testing.T) {
	policy := restartTestPolicy()

	spawns := 0

	sup := supervisor.NewSupervisor(func(ctx context.Context) (*supervisor.Child, error) {
		spawns++

		return spawnSuperviseChild(ctx, helperModeCrash, 0)
	}, policy)

	ctx, cancel := context.WithTimeout(context.Background(), restartWatchdog)
	defer cancel()

	err := sup.Start(ctx)

	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the restart policy looped past the limit without failing loud within %v", restartWatchdog)
	}

	if err == nil {
		t.Fatal("Start must fail loud past the crash limit, not keep restarting")
	}

	if spawns != tripSpawnCount {
		t.Fatalf("spawn attempts = %d, want %d (3 attempts within the window, then fail loud)", spawns, tripSpawnCount)
	}

	assertNamesCrashAndWindow(t, err, policy)
}

// testRestartWindowReset: after one crash, a child that stays alive LONGER
// than the window resets the sliding counter (RESEARCH Pitfall 7 — absent
// from both analogs), so the limit trips only after FOUR further crashes —
// six spawns total; a five-spawn trip would prove the counter never reset.
func testRestartWindowReset(t *testing.T) {
	policy := restartTestPolicy()

	spawns := 0

	sup := supervisor.NewSupervisor(func(ctx context.Context) (*supervisor.Child, error) {
		spawns++

		if spawns == stableSpawnOrdinal {
			return spawnSuperviseChild(ctx, helperModeStay, resetStayMs)
		}

		return spawnSuperviseChild(ctx, helperModeCrash, 0)
	}, policy)

	ctx, cancel := context.WithTimeout(context.Background(), restartWatchdog)
	defer cancel()

	err := sup.Start(ctx)

	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the restart policy looped past the limit without failing loud within %v", restartWatchdog)
	}

	if err == nil {
		t.Fatal("Start must eventually fail loud even with the sliding reset")
	}

	if spawns != resetTripSpawnCount {
		t.Fatalf("spawn attempts = %d, want %d — a stable run longer than the window must reset the counter",
			spawns, resetTripSpawnCount)
	}

	assertNamesCrashAndWindow(t, err, policy)
}

// restartTestPolicy is the injectable-knobs policy for the restart tests:
// production shape (3 attempts, sliding window, exponential backoff), test
// timescales.
func restartTestPolicy() supervisor.Policy {
	policy := supervisor.DefaultPolicy()
	policy.CrashWindow = restartWindowShort
	policy.BaseBackoff = restartBackoff
	policy.Grace = restartGraceShort

	return policy
}

// assertNamesCrashAndWindow asserts the loud failure names the crash count
// and the window (the actionable diagnostics the policy owes its caller).
func assertNamesCrashAndWindow(t *testing.T, err error, policy supervisor.Policy) {
	t.Helper()

	for _, want := range []string{"crash", tripCrashCount, policy.CrashWindow.String()} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("crash-loop error %q must name %q", err, want)
		}
	}
}

// spawnSuperviseChild re-executes the test binary as the helper child through
// the exported SpawnChild path (the Task-1 spawn contract: Setpgid, piped
// stdio) with the requested helper mode.
func spawnSuperviseChild(ctx context.Context, mode string, stayMs int) (*supervisor.Child, error) {
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		superviseHelperModeEnv + "=" + mode,
		superviseHelperStayEnv + "=" + strconv.Itoa(stayMs),
	}

	return supervisor.SpawnChild(ctx, []string{os.Args[0], superviseHelperRunFlag}, env, supervisor.SpawnOpts{})
}

// waitSpawnedChild receives the SpawnFunc's child delivery (race-safe handoff
// from the Start goroutine).
func waitSpawnedChild(t *testing.T, spawnCh <-chan *supervisor.Child) *supervisor.Child {
	t.Helper()

	select {
	case child := <-spawnCh:
		return child
	case <-time.After(spawnSettleDeadline):
		t.Fatalf("no child delivered by the spawn path within %v", spawnSettleDeadline)

		return nil
	}
}

// readGrandchildPid reads the helper's "ready <pid>" line — the
// synchronization point proving the grandchild EXISTS before the shutdown.
func readGrandchildPid(t *testing.T, child *supervisor.Child) int {
	t.Helper()

	type result struct {
		pid  int
		line string
		err  error
	}

	done := make(chan result, 1)

	go func() {
		scanner := bufio.NewScanner(child.Stdout)
		if !scanner.Scan() {
			done <- result{err: errNoReadyLine}

			return
		}

		line := scanner.Text()

		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != grandchildReadyPrefix {
			done <- result{line: line, err: errUnexpectedReadyLine}

			return
		}

		pid, convErr := strconv.Atoi(fields[1])
		done <- result{pid: pid, err: convErr}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("read the helper's readiness line: %v (line %q)", res.err, res.line)
		}

		return res.pid

	case <-time.After(spawnSettleDeadline):
		t.Fatalf("no readiness line from the helper within %v", spawnSettleDeadline)
	}

	return 0 // unreachable: both select arms end the test
}

// assertReaped asserts the monitor reaped the child (Done closed, Exited
// delivered) — the "Wait reaps the child" half of the zero-survivors proof.
func assertReaped(t *testing.T, child *supervisor.Child) {
	t.Helper()

	select {
	case <-child.Done:
	case <-time.After(groupGoneDeadline):
		t.Fatalf("child was not reaped within %v after the staged shutdown", groupGoneDeadline)
	}

	select {
	case <-child.Exited:
	case <-time.After(groupGoneDeadline):
		t.Fatal("the exit monitor never delivered the exit info")
	}
}

// assertGone polls kill(signo target, 0) until ESRCH — zero survivors is
// proven by polling (zombies exist until reaped), never by one racy check.
func assertGone(t *testing.T, what string, pid int) {
	t.Helper()

	deadline := time.Now().Add(groupGoneDeadline)

	for {
		err := syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("%s still alive after the staged shutdown: kill(%d, 0) = %v", what, pid, err)
		}

		time.Sleep(groupGonePoll)
	}
}

// errNoReadyLine and errUnexpectedReadyLine are readGrandchildPid's failure
// sentinels.
var (
	errNoReadyLine         = errors.New("child stdout reached EOF without a readiness line")
	errUnexpectedReadyLine = errors.New("unexpected readiness line")
)
