// End-to-end exec chain (BRIDGE-03 + BRIDGE-04 composition, plan 03-09):
// the FULL phase chain composed in one test — the supervisor spawns the
// standalone fake-app-server BINARY through the real spawn path (process
// group, allowlisted env, drained stderr), a zcode.Conn rides the exec
// pipes, and the turn machine drives turns over them. Subtest 1 replays the
// happy-turn corpus row: the conn's router answers the create-time
// runtime-preferences obligation (D-05) and Machine.Run resolves completed
// exactly once. Subtest 2 replays the cancel-mid-turn corpus row: a machine
// bound to the same corpus session id has its turn cancelled mid-flight
// through the bypass lane and resolves cancelled exactly once. Each subtest
// ends with the supervisor's staged shutdown and the poll-verified
// zero-survivors proof (kill(-pgid, 0) ESRCH — the ROADMAP criteria 2 + 4
// composition, RESEARCH System Architecture Diagram executed).
//
// CI isolation (T-3-13): the child is the built FAKE binary only — the
// real-desktop path stays behind realdesktop_test.go's build tag + env
// guard, and this test never contacts a desktop bundle. The binary is
// produced once via `go build` into a temp dir (the RESEARCH A2-blessed
// built-binary fallback; no module deps, zero new dependencies — T-3-SC).

package supervisor_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Corpus rows replayed over the exec chain — the fake package's own
// testdata, i.e. the SAME fixtures the in-process golden tests pin (the
// corpus is the protocol spec made executable, D-01).
const (
	e2eHappyScenario  = "../zcode/fake/testdata/scenarios/happy-turn.json"
	e2eCancelScenario = "../zcode/fake/testdata/scenarios/cancel-mid-turn.json"
)

// e2eSessionID is the corpus scenarios' fixed synthetic session id
// (sess_fix1): the happy create reply must carry it, and the cancel cycle
// binds its machine to it directly (the cancel corpus row has no create
// rule — the machine starts at the send, the shape 03-08's corpus proofs
// use).
const e2eSessionID = "sess_fix1"

// Watcher knobs (millisecond-scale, the 03-05 test pattern): no test waits
// on the wall-clock dialect defaults, and the terminal/cancel signals win
// deterministically against the quiet-window settle floor
// (FloorMultiple x quiet + quiet ≈ 120 ms — the signals land in ~ms).
const (
	e2eQuietMs        = 40
	e2eFloorMultiple  = 2
	e2ePollIntervalMs = 15
)

// Cycle deadlines: one subtest's full pass (spawn → protocol exchange →
// turn → staged shutdown → ESRCH) fits well inside the cycle deadline; the
// shutdown watchdog bounds only the staged teardown; the build deadline
// bounds the one-time `go build` of the fake binary.
const (
	e2eCycleDeadline    = 30 * time.Second
	e2eShutdownWatchdog = 15 * time.Second
	e2eBuildDeadline    = 120 * time.Second

	// e2eAckWait / e2eAckPoll bound the wait for the send ack — non-zero
	// AckAt proves the turn is running (the fake's side effects precede the
	// ack, so the running latch is already recorded when it returns).
	e2eAckWait = 2 * time.Second
	e2eAckPoll = time.Millisecond
)

// e2eRunContent is the prompt content the machine sends — arbitrary (the
// fake matches requests by method), named so no magic string repeats.
const e2eRunContent = "hello from the e2e exec chain"

// e2eScenarioFlag is the fake binary's scenario argv flag (cmd/fake-app-
// server's stdlib flag).
const e2eScenarioFlag = "--scenario"

// TestEndToEndExecChain runs the full phase chain end-to-end on real exec
// pipes: supervisor → built fake binary child → codec Conn → turn machine —
// the happy turn, the mid-turn cancel, then the staged shutdown with zero
// surviving group members. The fake binary is built once per test run and
// shared by both subtests.
func TestEndToEndExecChain(t *testing.T) {
	fakeBinary := e2eBuildFakeBinary(t)

	t.Run("happy turn completes exactly once", func(t *testing.T) {
		e2eRunHappyTurn(t, fakeBinary)
	})

	t.Run("cancel mid-turn resolves cancelled exactly once", func(t *testing.T) {
		e2eRunCancelTurn(t, fakeBinary)
	})
}

// e2eBuildFakeBinary builds cmd/fake-app-server ONCE per test run via
// `go build` into the run's temp dir (the RESEARCH A2-blessed built-binary
// fallback; no module deps, zero new dependencies — the pinned mise
// toolchain builds the repo's own package). The exec-chain child is this
// REAL binary, never an in-process Serve and never the desktop bundle
// (T-3-13). One build per parent-test invocation (not per process) so
// -count repeats each get a live binary — a prior run's TempDir is gone.
func e2eBuildFakeBinary(t *testing.T) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), e2eBuildDeadline)
	defer cancel()

	binPath := filepath.Join(t.TempDir(), "fake-app-server")

	cmd := exec.CommandContext(ctx, "go", "build", "-o", binPath,
		"github.com/Djarvur/zcode-acp/cmd/fake-app-server")
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("go build cmd/fake-app-server: %v", err)
	}

	return binPath
}

// e2eRunHappyTurn replays corpus row 1 over the exec chain: session/create
// (the router answers the create-time runtime-preferences obligation), then
// Machine.Run resolves completed exactly once via the turnId-matched
// terminal event.
func e2eRunHappyTurn(t *testing.T, fakeBinary string) {
	t.Helper()

	ctx, cancelCtx := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer cancelCtx()

	child, conn, supCancel, startDone := e2eStartCycle(ctx, t, fakeBinary, e2eHappyScenario)

	// The machine is constructed BEFORE any frame flows — the Conn hook
	// contract (NewMachine assigns the conn's OnNotification/OnRouterEvent
	// hooks, which must be in place before the demux and the router can
	// read them). The binding is the corpus fixture id; the create reply is
	// asserted against it below.
	spy := &e2eResolutionSpy{}
	machine := e2eNewMachine(conn, e2eSessionID, spy)

	// session/create over the exec pipes: the fake blocks the create on the
	// runtime-preferences reverse request, which the conn's router answers
	// within its D-05 deadline (unanswered creates fail -32603 — the host
	// obligation the front leg inherits).
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve workspace dir: %v", err)
	}

	result, err := conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{"workspacePath": cwd, "workspaceKey": cwd},
	})
	if err != nil {
		t.Fatalf("session/create over the exec pipes: %v", err)
	}

	if sessID := e2eDecodeSessionID(t, result); sessID != e2eSessionID {
		t.Fatalf("create reply session id = %q, want the machine's binding %q", sessID, e2eSessionID)
	}

	reason, err := machine.Run(ctx, e2eRunContent)
	if err != nil {
		t.Fatalf("happy turn: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("stop reason = %q, want %q", reason, turn.StopCompleted)
	}

	spy.assertSingle(t, turn.StopCompleted, turn.SignalTerminalEvent)

	_ = conn.Close() // the staged shutdown owns stdin from here
	e2eShutdownAndAssertGone(t, child, supCancel, startDone)
}

// e2eRunCancelTurn replays corpus row 2 over the exec chain: a machine bound
// to the corpus session id starts a turn, the test cancels mid-turn once
// the send ack proves it is running, and the stop's observed outcome (the
// cancelled state patch through the fake's stop rule) resolves cancelled
// exactly once.
func e2eRunCancelTurn(t *testing.T, fakeBinary string) {
	t.Helper()

	ctx, cancelCtx := context.WithTimeout(context.Background(), e2eCycleDeadline)
	defer cancelCtx()

	child, conn, supCancel, startDone := e2eStartCycle(ctx, t, fakeBinary, e2eCancelScenario)

	spy := &e2eResolutionSpy{}
	machine := e2eNewMachine(conn, e2eSessionID, spy)

	outcome := e2eRunAsync(ctx, machine, e2eRunContent)
	e2eWaitAck(t, machine)
	machine.Cancel() // session/stop through the bypass lane, mid-turn

	got := <-outcome
	if got.err != nil {
		t.Fatalf("cancelled turn: %v", got.err)
	}

	if got.reason != turn.StopCancelled {
		t.Fatalf("stop reason = %q, want %q", got.reason, turn.StopCancelled)
	}

	spy.assertSingle(t, turn.StopCancelled, turn.SignalCancel)

	_ = conn.Close()
	e2eShutdownAndAssertGone(t, child, supCancel, startDone)
}

// e2eStartCycle runs one supervised child cycle: the REAL discovery resolves
// the synthetic install through the order-step-1 override, the supervisor's
// production Start machinery (health watch, crash window, staged shutdown)
// owns the child while the SpawnFunc seam hands SpawnChild the fake's argv
// ([fake, --scenario, <corpus row>]), and the conn rides the child's pipes.
// The supervision ctx cancel IS the staged-shutdown trigger.
func e2eStartCycle(ctx context.Context, t *testing.T, fakeBinary, scenario string,
) (*supervisor.Child, *zcode.Conn, context.CancelFunc, <-chan error) {
	t.Helper()

	scenarioAbs, err := filepath.Abs(scenario)
	if err != nil {
		t.Fatalf("resolve %s: %v", scenario, err)
	}

	inst := e2eDiscoverOverride(ctx, t, fakeBinary)

	supCtx, supCancel := context.WithCancel(ctx)
	t.Cleanup(supCancel) // an early Fatal still tears the child down

	spawnCh := make(chan *supervisor.Child, 1)
	startDone := make(chan error, 1)

	sup := supervisor.NewSupervisor(func(spawnCtx context.Context) (*supervisor.Child, error) {
		child, spawnErr := supervisor.SpawnChild(spawnCtx,
			[]string{inst.Binary, e2eScenarioFlag, scenarioAbs},
			supervisor.ChildEnv(inst, os.Environ()), supervisor.SpawnOpts{})
		if spawnErr == nil {
			spawnCh <- child
		}

		return child, spawnErr
	}, supervisor.DefaultPolicy())

	go func() { startDone <- sup.Start(supCtx) }()

	child := waitSpawnedChild(t, spawnCh)

	//nolint:contextcheck // NewConn owns its base context (ctx only bounds the calls below)
	conn := zcode.NewConn(e2eDuplex{reader: child.Stdout, writer: child.Stdin})
	t.Cleanup(func() { _ = conn.Close() })

	return child, conn, supCancel, startDone
}

// e2eDiscoverOverride resolves the synthetic install through the REAL
// Discover — order step 1: the override wins; every other probe runs to
// completion against an empty fixture home and misses, so the diagnostics
// stay complete on the success path (the discovery-adjacency decision) and
// the test stays hermetic (nil seams record misses, never exec host tools).
func e2eDiscoverOverride(ctx context.Context, t *testing.T, fakeBinary string) supervisor.Install {
	t.Helper()

	probers := supervisor.Probers{
		Override: func() (supervisor.Install, bool) {
			return supervisor.Install{
				Root:   filepath.Dir(fakeBinary),
				Binary: fakeBinary,
				Layout: "explicit",
			}, true
		},
		Home:   t.TempDir(),
		OptDir: filepath.Join(t.TempDir(), "opt"),
	}

	inst, probes, err := probers.Discover(ctx)
	if err != nil {
		t.Fatalf("discover the synthetic override install: %v", err)
	}

	if inst.Binary != fakeBinary {
		t.Fatalf("override binary = %q, want %q", inst.Binary, fakeBinary)
	}

	t.Logf("discovery override resolved the fake binary install (%d probe rows recorded)", len(probes))

	return inst
}

// e2eShutdownAndAssertGone drives the supervisor's staged shutdown (the
// supervision ctx cancel), asserts Start returns nil on the intentional
// path, and proves zero surviving group members: the monitor reaped the
// child (Done closed, Exited delivered) and kill(-pgid, 0) reports ESRCH by
// bounded poll — stdin-EOF is never trusted (PITFALL 6) and zombies exist
// until reaped, so zero survivors is proven by polling, never one racy
// check.
func e2eShutdownAndAssertGone(t *testing.T, child *supervisor.Child,
	supCancel context.CancelFunc, startDone <-chan error,
) {
	t.Helper()

	supCancel()

	select {
	case err := <-startDone:
		if err != nil {
			t.Fatalf("supervisor Start returned %v; want nil on intentional shutdown", err)
		}
	case <-time.After(e2eShutdownWatchdog):
		t.Fatalf("the staged shutdown did not complete within %v", e2eShutdownWatchdog)
	}

	assertReaped(t, child)
	assertGone(t, "the fake app-server process group", -child.PgID)
}

// e2eNewMachine builds the turn machine over the conn with the fast watcher
// knobs and the exactly-once spy as the OnResolve observer.
func e2eNewMachine(conn *zcode.Conn, sessID string, spy *e2eResolutionSpy) *turn.Machine {
	machine := turn.NewMachine(conn, sessID, turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        e2eQuietMs,
		FloorMultiple:  e2eFloorMultiple,
		PollIntervalMs: e2ePollIntervalMs,
	}))
	machine.OnResolve = spy.record

	return machine
}

// e2eDecodeSessionID reads the create reply's session id through a generic
// decode: the wire key is the dialect's camelCase session.sessionId (a
// snake_case struct tag would misname the wire contract).
func e2eDecodeSessionID(t *testing.T, result json.RawMessage) string {
	t.Helper()

	var created struct {
		Session map[string]any `json:"session"`
	}

	if err := json.Unmarshal(result, &created); err != nil {
		t.Fatalf("decode session/create result %s: %v", result, err)
	}

	sessID, _ := created.Session["sessionId"].(string)
	if sessID == "" {
		t.Fatalf("session/create result carries no session.sessionId: %s", result)
	}

	return sessID
}

// e2eResolutionSpy is the synchronized OnResolve recorder — the exactly-once
// counter that survives the machine's observing goroutines (the
// resolutionSpy pattern 03-05 established).
type e2eResolutionSpy struct {
	mu          sync.Mutex
	resolutions []turn.Resolution
}

// record implements the machine's OnResolve hook.
func (s *e2eResolutionSpy) record(res turn.Resolution) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resolutions = append(s.resolutions, res)
}

// assertSingle asserts the turn resolved exactly once, with the expected
// reason and resolving signal.
func (s *e2eResolutionSpy) assertSingle(t *testing.T, reason turn.StopReason, signal turn.SignalKind) {
	t.Helper()

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.resolutions) != 1 {
		t.Fatalf("resolution count = %d, want exactly 1: %+v", len(s.resolutions), s.resolutions)
	}

	got := s.resolutions[0]
	if got.Reason != reason || got.Signal != signal {
		t.Fatalf("resolution = %s/%s, want %s/%s", got.Reason, got.Signal, reason, signal)
	}
}

// e2eRunOutcome carries one Machine.Run outcome across the goroutine
// boundary — the test cancels from outside while the turn runs.
type e2eRunOutcome struct {
	reason turn.StopReason
	err    error
}

// e2eRunAsync drives Machine.Run on its own goroutine.
func e2eRunAsync(ctx context.Context, machine *turn.Machine, content string) <-chan e2eRunOutcome {
	done := make(chan e2eRunOutcome, 1)

	go func() {
		reason, err := machine.Run(ctx, content)
		done <- e2eRunOutcome{reason: reason, err: err}
	}()

	return done
}

// e2eWaitAck blocks until the current turn's send ack landed, proving the
// turn is running (fail-loud: a missing ack means the exchange never
// started).
func e2eWaitAck(t *testing.T, machine *turn.Machine) {
	t.Helper()

	deadline := time.After(e2eAckWait)

	for machine.AckAt().IsZero() {
		select {
		case <-deadline:
			t.Fatal("the send ack never landed — the turn never started")
		case <-time.After(e2eAckPoll):
		}
	}
}

// e2eDuplex joins the child's stdout read side and stdin write side into the
// one io.ReadWriteCloser the codec Conn consumes (ARCHITECTURE internal
// boundary: the supervisor owns the pipes and hands the Conn a
// ReadWriteCloser). Close unblocks a blocked Read (os/exec pipes do), which
// is what Conn.Close relies on to join its read loop.
type e2eDuplex struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

// Read implements io.Reader over the child's stdout.
func (d e2eDuplex) Read(p []byte) (int, error) {
	return d.reader.Read(p)
}

// Write implements io.Writer to the child's stdin.
func (d e2eDuplex) Write(p []byte) (int, error) {
	return d.writer.Write(p)
}

// Close ends both pipe halves, joining the conn's read loop.
func (d e2eDuplex) Close() error {
	return errors.Join(d.writer.Close(), d.reader.Close())
}
