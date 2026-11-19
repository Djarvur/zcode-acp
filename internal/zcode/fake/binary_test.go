// Exec-plumbing tier for the standalone fake binary (BRIDGE-02, D-01 mode
// B): the helper-process reexec idiom (GOROOT-verified convention — the
// net/http/httptest/server_test.go:426,435 shape) drives a REAL child
// process over os.Pipe stdin/stdout with the exact spawn plumbing the
// supervisor (03-07) formalizes — child in its own process group (Setpgid),
// stderr captured, exit reaped. Frames flow both directions, the EOF drain
// window is observable rather than instant, and the exit-0-with-zero-lines
// failure signature (dialect §1 :30) is enforced at this tier against a real
// child.
package fake_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Reexec contract: a test binary re-executed with GO_WANT_HELPER_PROCESS=1
// IS the fake binary — fake.Main serves the scenario at FAKE_SCENARIO over
// the inherited stdio and exits with its code (the same call cmd/fake-app-
// server's main makes: one scripting core, two hosting modes, D-01).
const (
	helperEnv   = "GO_WANT_HELPER_PROCESS"
	scenarioEnv = "FAKE_SCENARIO"

	// happyTurnScenario is the corpus row both exec-tier tests serve.
	happyTurnScenario = "testdata/scenarios/happy-turn.json"

	// runtimePrefsAnswer is the reverse-request answer the host side echoes
	// back (the happy-turn scenario's expect_answer, semantically compared
	// by the fake).
	runtimePrefsAnswer = `{"nativeSearchEnhancementsEnabled":false,"memoryEnabled":false}`

	// frameDeadline bounds each scripted read on the child's stdout pipe;
	// exchanges complete in milliseconds and the deadline fails fast.
	frameDeadline = 5 * time.Second

	// exitDeadline bounds the child's exit wait after stdin close: the drain
	// window (100 ms) plus generous scheduling slack, so a hung child fails
	// fast instead of stalling the suite.
	exitDeadline = 2 * time.Second

	// drainLowerBound is the observable-drain assertion's floor: the child
	// sleeps its eof_drain_ms (100 ms, dialect §1 :30) between stdin EOF and
	// exit, so an instant exit would break the drain contract.
	drainLowerBound = 90 * time.Millisecond

	// lineChanBuffer carries the child's stdout lines from the reader
	// goroutine; sized so the reader never blocks a test that stops reading.
	lineChanBuffer = 32

	// createCallID and sendCallID are the parent's integer call ids.
	createCallID = 1
	sendCallID   = 2

	// parentFrameCount and childFrameCount pin the happy-turn exchange
	// volume over the exec pipes: the parent writes three lines (create,
	// the reverse answer, send); the child answers with eight (two lifecycle
	// emissions, the reverse request, the create reply, the state patch, two
	// session events, the send ack).
	parentFrameCount = 3
	childFrameCount  = 8
)

// fakeChild is one re-executed fake-binary child: the protocol pipes, the
// child's stdout line stream, its captured stderr, and the process-group
// handle (Setpgid — the spawn hygiene T-3-07 mandates and 03-07 formalizes).
type fakeChild struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	pgid   int
	stdin  io.WriteCloser
	lines  chan string
	stderr strings.Builder
	wrote  int
	read   int
	waited bool
}

// spawnFakeChild re-executes the test binary as the fake binary child
// (RESEARCH.md :535-547 idiom): os.Args[0] targeted at this test's name, the
// scenario in FAKE_SCENARIO, os.Pipe stdin/stdout as the protocol, stderr
// captured to a buffer exec drains itself, and the child started in its own
// process group. The cleanup backstop kills the group and reaps a child the
// test left behind — no orphan survives a failure path (T-3-07).
func spawnFakeChild(t *testing.T, scenarioPath string) *fakeChild {
	t.Helper()

	scenarioAbs, err := filepath.Abs(scenarioPath)
	if err != nil {
		t.Fatalf("resolve scenario path: %v", err)
	}

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	spawnCtx, cancel := context.WithCancel(context.Background())

	child := &fakeChild{
		cmd:    exec.CommandContext(spawnCtx, os.Args[0], "-test.run=^"+t.Name()+"$"),
		cancel: cancel,
		stdin:  stdinW,
		lines:  make(chan string, lineChanBuffer),
	}

	child.cmd.Env = append(os.Environ(), helperEnv+"=1", scenarioEnv+"="+scenarioAbs)
	child.cmd.Stdin = stdinR
	child.cmd.Stdout = stdoutW
	child.cmd.Stderr = &child.stderr
	child.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := child.cmd.Start(); err != nil {
		t.Fatalf("start fake child: %v", err)
	}

	child.pgid = child.cmd.Process.Pid

	// The parent keeps exactly one end per pipe: closing the child's ends
	// here is what makes the stdin-EOF and stdout-EOF semantics work.
	_ = stdinR.Close()
	_ = stdoutW.Close()

	go func() {
		sc := bufio.NewScanner(stdoutR)

		for sc.Scan() {
			child.lines <- sc.Text()
		}

		close(child.lines)
	}()

	t.Cleanup(func() { child.terminate(t) })

	return child
}

// terminate is the cleanup backstop: a child the test did not reap is killed
// as a process group and waited — the full group-kill assertion is 03-07's.
func (c *fakeChild) terminate(t *testing.T) {
	t.Helper()

	c.cancel()
	_ = c.stdin.Close()

	if c.waited {
		return
	}

	t.Logf("fake child stderr at kill: %s", c.stderr.String())
	_ = syscall.Kill(-c.pgid, syscall.SIGKILL)
	_ = c.cmd.Wait()

	c.waited = true
}

// writeFrame encodes one parent→child protocol frame and writes its wire
// line to the child's stdin pipe (the parent is the exec-pipe client side).
func (c *fakeChild) writeFrame(t *testing.T, env *zcode.Envelope) {
	t.Helper()

	line, err := zcode.Encode(env)
	if err != nil {
		t.Fatalf("encode parent frame: %v", err)
	}

	if _, err := c.stdin.Write(line); err != nil {
		t.Fatalf("write to child stdin: %v", err)
	}

	c.wrote++
}

// readFrame reads the next protocol line from the child's stdout, failing
// the test on premature EOF or a frameDeadline stall.
func (c *fakeChild) readFrame(t *testing.T) string {
	t.Helper()

	select {
	case line, ok := <-c.lines:
		if !ok {
			t.Fatal("child stdout closed before the scripted exchange completed")

			return ""
		}

		c.read++

		return line

	case <-time.After(frameDeadline):
		t.Fatalf("no frame from the child within %v", frameDeadline)

		return ""
	}
}

// readEnvelope reads and decodes the next child→parent frame.
func (c *fakeChild) readEnvelope(t *testing.T) zcode.Envelope {
	t.Helper()

	line := c.readFrame(t)

	env, err := zcode.Decode([]byte(line))
	if err != nil {
		t.Fatalf("decode child frame %q: %v", line, err)
	}

	return env
}

// wantMethod reads the next child frame and asserts its method — wire-order
// breakage fails here.
func (c *fakeChild) wantMethod(t *testing.T, method string) zcode.Envelope {
	t.Helper()

	env := c.readEnvelope(t)

	if env.Method != method {
		t.Fatalf("child frame method = %q, want %q", env.Method, method)
	}

	return env
}

// wantReply reads the next child frame and asserts it is the reply to the
// given integer call id, echoed verbatim (dialect §1).
func (c *fakeChild) wantReply(t *testing.T, id int64) zcode.Envelope {
	t.Helper()

	env := c.readEnvelope(t)

	if env.Method != "" || env.ID == nil || !env.ID.IsInt || env.ID.Int != id {
		t.Fatalf("expected the reply to call id %d, got method=%q id=%v", id, env.Method, env.ID)
	}

	return env
}

// closeStdin ends the protocol stream: the child sees stdin EOF, drains its
// eof_drain_ms window, and exits (dialect §1 :30).
func (c *fakeChild) closeStdin(t *testing.T) {
	t.Helper()

	if err := c.stdin.Close(); err != nil {
		t.Fatalf("close child stdin: %v", err)
	}
}

// waitExit waits for the child to exit (bounded by exitDeadline), reaps it,
// and logs its captured stderr; the return is the process exit code. A hung
// child is killed as a group before the test fails — no orphan left behind.
func (c *fakeChild) waitExit(t *testing.T) int {
	t.Helper()

	done := make(chan error, 1)
	waitDone := make(chan struct{})

	go func() {
		done <- c.cmd.Wait()
		close(waitDone)
	}()

	select {
	case waitErr := <-done:
		c.waited = true

		t.Logf("fake child stderr: %s", c.stderr.String())

		if waitErr == nil {
			return 0
		}

		var exitErr *exec.ExitError

		if !errors.As(waitErr, &exitErr) {
			t.Fatalf("child wait failed (no exit status): %v", waitErr)
		}

		return exitErr.ExitCode()

	case <-time.After(exitDeadline):
		c.cancel()
		_ = syscall.Kill(-c.pgid, syscall.SIGKILL)
		<-waitDone // the original Wait reaps after the kill — never a double Wait

		t.Fatalf("child did not exit within %v (killed, group reaped)", exitDeadline)
	}

	return 0 // unreachable: both select arms end the test
}

// collectStdout drains every stdout line the child produced — after exit,
// the reader goroutine flushes the pipe buffer and closes the channel.
func (c *fakeChild) collectStdout(t *testing.T) []string {
	t.Helper()

	var lines []string

	deadline := time.After(frameDeadline)

	for {
		select {
		case line, ok := <-c.lines:
			if !ok {
				return lines
			}

			lines = append(lines, line)

		case <-deadline:
			t.Fatalf("child stdout did not reach EOF within %v", frameDeadline)
		}
	}
}

// TestFakeBinarySpawnPath drives the happy-turn exchange through the REAL
// exec pipes of a re-executed fake child: create answered via the scripted
// runtime-preferences reverse exchange, send ack, notifications in wire
// order — then stdin close, the observable EOF drain, and a clean exit 0.
func TestFakeBinarySpawnPath(t *testing.T) {
	if os.Getenv(helperEnv) == "1" {
		// Helper branch: this process IS the fake binary — serve the
		// scenario over the inherited stdio (the same fake.Main call the
		// cmd main makes) and exit with its code.
		os.Exit(fake.Main(os.Getenv(scenarioEnv)))
	}

	child := spawnFakeChild(t, happyTurnScenario)

	driveSpawnCreate(t, child)
	driveSpawnSend(t, child)

	if child.wrote != parentFrameCount || child.read != childFrameCount {
		t.Fatalf("frames through the exec pipes: wrote %d (want %d), read %d (want %d)",
			child.wrote, parentFrameCount, child.read, childFrameCount)
	}

	// stdin EOF: the child drains its eof_drain_ms window (dialect §1 :30)
	// and exits 0 — observable, never instant.
	drainStart := time.Now()

	child.closeStdin(t)

	if code := child.waitExit(t); code != 0 {
		t.Fatalf("child exit code = %d, want 0 after the EOF drain", code)
	}

	drained := time.Since(drainStart)
	if drained < drainLowerBound {
		t.Fatalf("child exited %v after stdin EOF — the drain window was not observable (want >= %v)",
			drained, drainLowerBound)
	}
}

// TestEOFSignatureChild pins the exit-0-with-zero-lines failure signature at
// the exec tier (dialect §1 :30): a real child that exits 0 having produced
// ZERO frames must be reported as FAILURE by the harness classifier — never
// an empty success. The in-process half of the contract is golden_test.go's
// TestEOFSignature (03-03); this test adds the child-process proof, and the
// plan's -run regex 'TestFakeBinary|TestEOFSignature' selects both.
func TestEOFSignatureChild(t *testing.T) {
	if os.Getenv(helperEnv) == "1" {
		os.Exit(fake.Main(os.Getenv(scenarioEnv)))
	}

	child := spawnFakeChild(t, happyTurnScenario)

	// No traffic: stdin closes immediately — the exact exchange shape whose
	// empty capture the probe guard records as failure, never success.
	child.closeStdin(t)

	if code := child.waitExit(t); code != 0 {
		t.Fatalf("child exit code = %d, want 0 (the signature exchange is a CLEAN child exit)", code)
	}

	captured := child.collectStdout(t)
	if len(captured) != 0 {
		t.Fatalf("zero-frame exchange captured %d stdout lines, want 0", len(captured))
	}

	err := checkExchangeRecorded(fake.Transcript{})
	if err == nil {
		t.Fatal("the harness must report FAILURE on a real child's exit-0-with-zero-lines, never success")
	}

	if !errors.Is(err, errZeroFrameExchange) {
		t.Fatalf("zero-lines failure must be the EOF-signature sentinel, got: %v", err)
	}
}

// driveSpawnCreate writes session/create and reads the scripted reverse
// exchange: lifecycle starting, the reverse request, the answer's effects,
// lifecycle ready, then the create reply.
func driveSpawnCreate(t *testing.T, child *fakeChild) {
	t.Helper()

	child.writeFrame(t, &zcode.Envelope{
		ID:     &zcode.ID{Int: createCallID, IsInt: true},
		Method: zcode.MethodSessionCreate,
		Params: json.RawMessage(`{"workspace":{"workspacePath":"/w","workspaceKey":"/w"}}`),
	})

	starting := child.wantMethod(t, zcode.MethodSessionLifecycleChanged)

	if got := paramsKey(t, starting, "state"); got != "starting" {
		t.Fatalf("lifecycle state = %q, want starting", got)
	}

	reverse := child.wantMethod(t, zcode.MethodRequestRuntimePreferences)

	if reverse.ID == nil || reverse.ID.IsInt {
		t.Fatalf("reverse request id = %v, want a string id (dialect §1)", reverse.ID)
	}

	child.writeFrame(t, &zcode.Envelope{
		ID:     reverse.ID,
		Result: json.RawMessage(runtimePrefsAnswer),
	})

	ready := child.wantMethod(t, zcode.MethodSessionLifecycleChanged)

	if got := paramsKey(t, ready, "state"); got != "ready" {
		t.Fatalf("lifecycle state = %q, want ready", got)
	}

	createReply := child.wantReply(t, createCallID)

	if got := jsonPathString(t, createReply.Result, "session", "sessionId"); got != fixSessionID {
		t.Fatalf("create reply sessionId = %q, want %q", got, fixSessionID)
	}
}

// driveSpawnSend writes session/send and reads the scripted turn: the state
// patch, the two session events, then the accepted ack.
func driveSpawnSend(t *testing.T, child *fakeChild) {
	t.Helper()

	child.writeFrame(t, &zcode.Envelope{
		ID:     &zcode.ID{Int: sendCallID, IsInt: true},
		Method: zcode.MethodSessionSend,
		Params: json.RawMessage(`{"sessionId":"` + fixSessionID + `","content":"Reply with exactly: OK"}`),
	})

	child.wantMethod(t, zcode.MethodStateUpdated) // the running patch
	child.wantMethod(t, zcode.MethodSessionEvent) // turn.started
	child.wantMethod(t, zcode.MethodSessionEvent) // turn.completed

	assertAccepted(t, child.wantReply(t, sendCallID))
}

// assertAccepted checks a send-ack reply's accepted flag — the exec tier
// pins the plumbing, not the full payload shape (TestHappyTurnRoundTrip
// owns that).
func assertAccepted(t *testing.T, env zcode.Envelope) {
	t.Helper()

	var ack struct {
		Accepted bool `json:"accepted"`
	}

	if err := json.Unmarshal(env.Result, &ack); err != nil {
		t.Fatalf("decode send ack: %v (%s)", err, env.Result)
	}

	if !ack.Accepted {
		t.Fatalf("send ack accepted = false: %s", env.Result)
	}
}
