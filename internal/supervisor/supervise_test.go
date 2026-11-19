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
	// helperModeGrandchildCrash spawns a sleeping grandchild (inheriting the
	// process group), reports its pid, then exits 1 immediately — the crash
	// that strands a LIVE grandchild inside the group (the CR-01 fixture:
	// the fail-loud crash-loop exit must still clean the tree).
	helperModeGrandchildCrash = "grandchild-crash"

	// superviseHelperRunFlag targets the helper inside the re-executed binary.
	superviseHelperRunFlag = "-test.run=^TestSuperviseHelper$"

	// grandchildSleepSecs keeps the grandchild alive well past the test so
	// only the group SIGKILL can reap it (PITFALL 6: stdin-EOF must not be
	// trusted to clean the tree).
	grandchildSleepSecs = "30"
	// grandchildReadyPrefix is the helper's readiness line ("ready <pid>").
	grandchildReadyPrefix = "ready"

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
	// resetTripSpawnCount pins the reset arithmetic: crash(1) → the 400 ms
	// stable run resets the counter, so its own crash counts 1' → crashes
	// 2', 3' restart → 4' trips → 1 + 1 + 3 = 5 spawns. Without the reset the
	// trip would come at spawn 4 (crashes 1,2,3,4 with the counter never
	// zeroed).
	resetTripSpawnCount = 5
	// noResetSpawnCount is what the trip arithmetic would be WITHOUT the
	// sliding reset — documented for the failure message.
	noResetSpawnCount = 4
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
	case helperModeGrandchildCrash:
		runGrandchildCrashHelper()
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

	_, _ = fmt.Fprintf(os.Stdout, "%s %d\n", grandchildReadyPrefix, grandchild.Process.Pid)
	_, _ = io.Copy(io.Discard, os.Stdin) // stay alive until stdin EOF

	os.Exit(0)
}

// runGrandchildCrashHelper is the grandchild-crash branch: spawn a sleeping
// grandchild in the same process group (no new group — exactly the tree the
// staged shutdown must reap), report its pid, and exit 1 immediately. The
// sequencing is the fixture's existence proof: the grandchild is ALREADY
// running in the group at the moment the helper crashes.
func runGrandchildCrashHelper() {
	grandchild := exec.CommandContext(context.Background(), "sleep", grandchildSleepSecs)
	if err := grandchild.Start(); err != nil {
		os.Exit(1)
	}

	_, _ = fmt.Fprintf(os.Stdout, "%s %d\n", grandchildReadyPrefix, grandchild.Process.Pid)

	os.Exit(1)
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
	t.Run("crash-loop trip leaves zero group survivors", testRestartWindowTripZeroSurvivors)
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
// from both analogs): the stable run's own crash then counts from zero, so
// the limit trips on that count's fourth crash — five spawns total; a
// four-spawn trip would prove the counter never reset.
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
		t.Fatalf("spawn attempts = %d, want %d (no-reset trip: %d) — a stable run must reset the sliding counter",
			spawns, resetTripSpawnCount, noResetSpawnCount)
	}

	assertNamesCrashAndWindow(t, err, policy)
}

// testRestartWindowTripZeroSurvivors: the fail-loud crash-loop exit owes the
// same staged shutdown as every other exit path (supervise.go's "staged
// shutdown ALWAYS" invariant, PITFALL 6): every crashing helper strands a
// LIVE sleeping grandchild inside its process group, so after Start returns
// the crash-loop error EVERY spawned group — the finally-crashed child's
// included — must report ESRCH. Without the shutdown on the trip path, the
// last group's grandchild survives as an orphan — exactly the leak the
// supervisor exists to contain (review CR-01).
func testRestartWindowTripZeroSurvivors(t *testing.T) {
	policy := restartTestPolicy()

	var children []*supervisor.Child // written by the Start goroutine; read only after runDone (channel sync)

	sup := supervisor.NewSupervisor(func(ctx context.Context) (*supervisor.Child, error) {
		child, err := spawnSuperviseChild(ctx, helperModeGrandchildCrash, 0)
		if err != nil {
			return child, err
		}

		children = append(children, child)

		// The readiness line prints only AFTER the grandchild started, and
		// exec.Wait (the monitor's reap) closes the parent-side stdout pipe
		// — discarding anything unread — so the line must be read HERE,
		// while the helper is alive. Reading it in the spawn path is also
		// the fixture's synchronization: the grandchild EXISTS inside the
		// group before its helper's crash can count.
		_, perr := readGrandchildReady(child)
		if perr != nil {
			t.Errorf("crashing child %d never proved its grandchild: %v", len(children), perr)
		}

		return child, nil
	}, policy)

	ctx, cancel := context.WithTimeout(context.Background(), restartWatchdog)
	defer cancel()

	runDone := make(chan error, 1)

	go func() {
		runDone <- sup.Start(ctx)
	}()

	var err error

	select {
	case err = <-runDone:
	case <-time.After(restartWatchdog):
		t.Fatalf("the crash-loop trip did not return within %v", restartWatchdog)
	}

	if err == nil {
		t.Fatal("Start must fail loud past the crash limit even with a grandchild in every group")
	}

	if len(children) != tripSpawnCount {
		t.Fatalf("spawn attempts = %d, want %d (3 attempts within the window, then fail loud)",
			len(children), tripSpawnCount)
	}

	assertNamesCrashAndWindow(t, err, policy)

	// Zero survivors — each child's WHOLE group (the child crash-reaped by
	// its own monitor; the grandchild only the group SIGKILL can reach).
	for i, child := range children {
		assertGone(t, fmt.Sprintf("crashing child %d's process group (grandchild included)", i+1), -child.PgID)
	}
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

	pid, err := readGrandchildReady(child)
	if err != nil {
		t.Fatalf("read the helper's readiness line: %v", err)
	}

	return pid
}

// readGrandchildReady is readGrandchildPid's test-goroutine-free core: it
// returns the grandchild pid (or the failure) so the supervisor's spawn
// goroutine can call it too — t.Fatalf is only legal on the test goroutine,
// and the spawn path runs on the Start goroutine.
func readGrandchildReady(child *supervisor.Child) (int, error) {
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
			return 0, fmt.Errorf("%w (line %q)", res.err, res.line)
		}

		return res.pid, nil

	case <-time.After(spawnSettleDeadline):
		return 0, fmt.Errorf("%w within %v", errReadyTimeout, spawnSettleDeadline)
	}
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

// errNoReadyLine, errUnexpectedReadyLine and errReadyTimeout are
// readGrandchildReady's failure sentinels.
var (
	errNoReadyLine         = errors.New("child stdout reached EOF without a readiness line")
	errUnexpectedReadyLine = errors.New("unexpected readiness line")
	errReadyTimeout        = errors.New("no readiness line from the helper")
)
