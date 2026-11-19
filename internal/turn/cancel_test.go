package turn_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Cancel + double-send guard contract tests (03-08, BRIDGE-04, TDD): cancel
// mid-turn sends session/stop through the codec's bypass lane (never queued
// behind prompts — D-02 lane 3) and resolves StopCancelled exactly once; a
// second concurrent session/send is rejected with the clean error mapping
// the dialect §6 code -32010 (matched by numeric code, never message text).
// Driven over the in-process fake — the programmatic option per D-01, with
// the JSON corpus variants replayed in the 03-08 corpus tier.

const (
	// cancelWindow bounds one cancel-flow Run: GREEN resolves in
	// milliseconds, the RED no-op stub never resolves and fails fast on the
	// planned resolution assertion.
	cancelWindow = 3 * time.Second

	// gateSettle bounds the pause-point wait: longer than two poll ticks
	// (10 ms), so by the time the queued markers are issued the conn writer
	// is deterministically parked in the in-flight write — the occupancy the
	// exchange-level bypass assertion needs.
	gateSettle = 30 * time.Millisecond

	// enqueueSettle lets a goroutine-issued call reach the writer queue.
	enqueueSettle = 5 * time.Millisecond

	// ackWaitTimeout bounds the wait for the send ack — non-zero AckAt is
	// the proof the turn is running (the fake's side effects precede the
	// ack, so the turn id and the running latch are already recorded).
	ackWaitTimeout = 2 * time.Second

	// statusCancelled / reasonSessionStopped are the corpus cancel-mid-turn
	// stop-rule patch values (stop was never exercised live —
	// by-construction valid like turn.completed).
	statusCancelled      = "cancelled"
	reasonSessionStopped = "session_stopped"

	// marker values embedded in the params of the queued ordinary calls so
	// the transcript can identify the frames the stop must precede.
	markerQueued1 = "queued-1"
	markerQueued2 = "queued-2"
)

// cancelScenario is the programmatic cancel-mid-turn script (the corpus
// cancel-mid-turn row's shape): the send ack with the running patch and
// turn.started, the session/stop rule whose cancelled patch precedes its ack
// (side effects before reply — the wire order the fake documents), and
// stable session/messages for the watcher's polls.
func cancelScenario(t *testing.T) *fake.Scenario {
	t.Helper()

	return &fake.Scenario{
		Name: "turn-cancel-mid-turn",
		OnRequest: []fake.RequestRule{
			sendRule(t,
				statePatch(t, statusRunning, reasonPromptStarted),
				sessionEvent(t, turn.KindTurnStarted, fixTurnID),
			),
			{
				Method:      zcode.MethodSessionStop,
				Reply:       &fake.Reply{Result: rawOf(t, map[string]any{})},
				SideEffects: []fake.SideEffect{statePatch(t, statusCancelled, reasonSessionStopped)},
			},
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// runResult carries one Machine.Run outcome across the goroutine boundary.
type runResult struct {
	reason turn.StopReason
	err    error
}

// runAsync drives Machine.Run on its own goroutine — the test cancels or
// double-sends from outside while the turn runs.
func runAsync(ctx context.Context, m *turn.Machine, content string) <-chan runResult {
	done := make(chan runResult, 1)

	go func() {
		reason, err := m.Run(ctx, content)
		done <- runResult{reason: reason, err: err}
	}()

	return done
}

// waitAck blocks until the current turn's send ack landed, proving the turn
// is running (fail-loud: a missing ack means the exchange never started).
func waitAck(t *testing.T, m *turn.Machine) {
	t.Helper()

	deadline := time.After(ackWaitTimeout)

	for m.AckAt().IsZero() {
		select {
		case <-deadline:
			t.Fatal("the send ack never landed — the turn never started")
		case <-time.After(time.Millisecond):
		}
	}
}

// readGate is a pausable io.Reader over the fake's pipe end: while paused,
// reads park before reaching the pipe, so the conn writer's next Write
// blocks in-flight and later enqueues park in the ordered queue — the
// deterministic occupancy the exchange-level bypass assertion needs (the
// writer-tier TestStopJumpsQueue mechanism, driven through the real
// codec + fake exchange).
type readGate struct {
	mu     sync.Mutex
	paused bool
	wake   chan struct{}
	r      io.Reader
}

// newReadGate wraps r with a pause gate.
func newReadGate(r io.Reader) *readGate {
	return &readGate{r: r, wake: make(chan struct{})}
}

// Read implements io.Reader with the pause gate (the fake's single scanner
// goroutine is the only reader, so at most one waiter parks per gate).
func (g *readGate) Read(b []byte) (int, error) {
	for {
		g.mu.Lock()
		paused, wake := g.paused, g.wake
		g.mu.Unlock()

		if !paused {
			break
		}

		<-wake
	}

	return g.r.Read(b)
}

// pause holds every subsequent Read until resume.
func (g *readGate) pause() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.paused {
		return
	}

	g.paused = true
	g.wake = make(chan struct{})
}

// resume releases held reads.
func (g *readGate) resume() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.paused {
		return
	}

	g.paused = false
	close(g.wake)
}

// gatedFakeEnd is the fake's stream end with the pausable read side; Write
// and Close pass through untouched.
type gatedFakeEnd struct {
	*readGate

	end *pipeEnd
}

// Write implements io.Writer.
func (g *gatedFakeEnd) Write(b []byte) (int, error) {
	return g.end.Write(b)
}

// Close tears down the underlying pipe pair; the peer sees EOF.
func (g *gatedFakeEnd) Close() error {
	return g.end.Close()
}

// startGatedTurnExchange wires the cancel-tier exchange: the
// startTurnExchange wiring with the fake's read side behind a
// test-controlled gate.
func startGatedTurnExchange(t *testing.T, sc *fake.Scenario) (*turnExchange, *readGate) {
	t.Helper()

	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	gate := newReadGate(fakeEnd)
	serveDone := startFake(&gatedFakeEnd{readGate: gate, end: fakeEnd}, sc)

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	m := turn.NewMachine(conn, fixSessionID, turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))

	spy := &resolutionSpy{}
	m.OnResolve = spy.record

	return &turnExchange{conn: conn, machine: m, spy: spy, serveDone: serveDone}, gate
}

// finishWithTranscript closes the conn (the fake drains), asserts the run
// ended clean, and returns the recorded transcript.
func finishWithTranscript(t *testing.T, ex *turnExchange) fake.Transcript {
	t.Helper()

	_ = ex.conn.Close()

	served := waitServed(t, ex.serveDone)

	if served.err != nil {
		t.Fatalf("fake.Serve: %v", served.err)
	}

	return served.transcript
}

// inLineCount counts host→fake frames whose wire line contains needle.
func inLineCount(tr fake.Transcript, needle string) int {
	n := 0

	for _, l := range tr.Lines {
		if l.Dir == fake.DirIn && strings.Contains(l.JSON, needle) {
			n++
		}
	}

	return n
}

// inLineIndex returns the transcript index of the first host→fake frame
// whose wire line contains needle, or -1.
func inLineIndex(tr fake.Transcript, needle string) int {
	for i, l := range tr.Lines {
		if l.Dir == fake.DirIn && strings.Contains(l.JSON, needle) {
			return i
		}
	}

	return -1
}

// queueQueuedCall fires one identifiable ordinary call on its own goroutine:
// while the gate holds the fake's reads, its write parks in the ordered
// queue behind the in-flight frame (the caller reads the wire order off the
// transcript at the end). Errors are deliberately discarded — the exchange's
// own assertions are the verdict.
func queueQueuedCall(ctx context.Context, ex *turnExchange, marker string) {
	go func() {
		_, _ = ex.conn.Call(ctx, zcode.MethodSessionMessages, map[string]any{
			wireKeySessionID: fixSessionID,
			"marker":         marker,
		})
	}()
}

// assertCancelWireContract asserts the cancel tier's transcript contract:
// exactly one session/stop reached the wire and it preceded BOTH ordinary
// calls parked behind the in-flight write — the bypass lane held (the
// stop never queued behind the parked traffic).
func assertCancelWireContract(t *testing.T, tr fake.Transcript) {
	t.Helper()

	if stops := inLineCount(tr, `"method":"session/stop"`); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1", stops)
	}

	idxStop := inLineIndex(tr, `"method":"session/stop"`)

	for _, marker := range []string{markerQueued1, markerQueued2} {
		needle := `"marker":"` + marker + `"`

		idxMarker := inLineIndex(tr, needle)
		if idxMarker < 0 {
			t.Fatalf("queued call %s never reached the wire", marker)
		}

		if idxStop > idxMarker {
			t.Fatalf("session/stop queued behind ordinary call %s — bypass lane violated", marker)
		}
	}
}

// The cancel contract (D-02 lane 3, PITFALL 4): Cancel mid-turn sends
// session/stop through the bypass lane — on the wire ahead of the ordinary
// traffic parked behind the in-flight write — and the turn resolves
// StopCancelled exactly once; a second Cancel is a no-op (no second stop on
// the wire, no state change), a cancel after completion resolves nothing
// new, and a fresh Run after done is a new turn (the fresh turn here
// resolves over the quiet window — no terminal in this script).
func TestCancelMidTurnAnswersOnce(t *testing.T) {
	ex, gate := startGatedTurnExchange(t, cancelScenario(t))

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	runDone := runAsync(ctx, ex.machine, "Reply with exactly: OK")
	waitAck(t, ex.machine)

	// Deterministic queue occupancy: pause the fake's reads — the conn
	// writer parks in the in-flight write within two poll ticks — then
	// queue two identifiable ordinary calls behind it.
	gate.pause()
	time.Sleep(gateSettle)

	queueQueuedCall(ctx, ex, markerQueued1)
	queueQueuedCall(ctx, ex, markerQueued2)
	time.Sleep(enqueueSettle)

	// Cancel mid-turn: the stop must jump the parked queue; the immediate
	// repeat is the idempotence probe.
	ex.machine.Cancel()
	ex.machine.Cancel()

	gate.resume()

	got := <-runDone

	if got.err != nil {
		t.Fatalf("Run did not resolve cancelled after Cancel: %v", got.err)
	}

	if got.reason != turn.StopCancelled {
		t.Fatalf("Run reason = %q, want %q", got.reason, turn.StopCancelled)
	}

	assertResolution(t, ex.spy, turn.StopCancelled, turn.SignalCancel)

	// Cancel after completion: nothing new resolves.
	ex.machine.Cancel()
	assertResolution(t, ex.spy, turn.StopCancelled, turn.SignalCancel)

	// A fresh Run after done is a new turn: the send rule answers again and
	// the quiet window resolves it.
	reason, err := ex.machine.Run(ctx, "fresh turn after done")
	if err != nil {
		t.Fatalf("fresh Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("fresh Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	gotAll := ex.spy.snapshot()

	if len(gotAll) != 2 {
		t.Fatalf("resolution hook fired %d times across two turns, want 2", len(gotAll))
	}

	if gotAll[1].Reason != turn.StopCompleted || gotAll[1].Signal != turn.SignalQuietWindow {
		t.Fatalf("fresh turn resolution = %v, want completed/quiet-window", gotAll[1])
	}

	ex.machine.Cancel()

	tr := finishWithTranscript(t, ex)
	assertCancelWireContract(t, tr)
}

// racingTerminalScenario is the cancel script whose stop rule emits the
// turnId-matched terminal event BEFORE the cancelled patch: a cancel racing
// a terminal resolves whichever signal lands first, still exactly once.
func racingTerminalScenario(t *testing.T) *fake.Scenario {
	t.Helper()

	sc := cancelScenario(t)
	sc.OnRequest[1].SideEffects = []fake.SideEffect{
		sessionEvent(t, turn.KindTurnCompleted, fixTurnID),
		statePatch(t, statusCancelled, reasonSessionStopped),
	}

	return sc
}

// The cancel-races-terminal contract (PITFALL 4): the stop reaches the wire,
// the terminal signal lands first, and the turn resolves completed exactly
// once — never twice, never hung.
func TestCancelRacingTerminalResolvesOnce(t *testing.T) {
	ex := startTurnExchange(t, racingTerminalScenario(t))

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	runDone := runAsync(ctx, ex.machine, "Reply with exactly: OK")
	waitAck(t, ex.machine)

	ex.machine.Cancel()
	ex.machine.Cancel()

	got := <-runDone

	if got.err != nil {
		t.Fatalf("Run: %v", got.err)
	}

	if got.reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q (the terminal signal landed first)", got.reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	tr := finishWithTranscript(t, ex)

	if inLineCount(tr, `"method":"session/stop"`) != 1 {
		t.Fatal("the racing-cancel stop never reached the wire")
	}
}

// serverRejectScenario: the first session/send is acked (its rule's budget
// spent), the second is answered -32010 carrying a message that is NOT the
// bundle literal — the mapping must key on the numeric code, never the
// message text (dialect §6: messages are not a stable contract).
func serverRejectScenario(t *testing.T) *fake.Scenario {
	t.Helper()

	first := sendRule(t,
		statePatch(t, statusRunning, reasonPromptStarted),
		sessionEvent(t, turn.KindTurnStarted, fixTurnID),
	)
	first.Times = 1

	return &fake.Scenario{
		Name: "turn-double-send-rejected",
		OnRequest: []fake.RequestRule{
			first,
			{
				Method: zcode.MethodSessionSend,
				Reply: &fake.Reply{Error: &zcode.WireError{
					Code:    zcode.CodeConcurrency,
					Message: "custom refusal text that must never be matched",
				}},
			},
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// The -32010 double-send guard, wrapper side (D-02, PITFALL 5): a second
// concurrent Run on the same machine is rejected with the mapped clean error
// without writing session/send on the wire.
func TestDoubleSendRejectedWrapperSide(t *testing.T) {
	ex := startTurnExchange(t, cancelScenario(t))

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	runDone := runAsync(ctx, ex.machine, "Reply with exactly: OK")
	waitAck(t, ex.machine)

	// The wrapper-side guard: rejected before any wire write.
	reason, err := ex.machine.Run(ctx, "second while running")
	if !errors.Is(err, turn.ErrTurnActive) {
		t.Fatalf("second Run error = %v, want ErrTurnActive (-32010 mapped)", err)
	}

	if reason != "" {
		t.Fatalf("rejected Run reason = %q, want empty", reason)
	}

	got := <-runDone

	if got.err != nil {
		t.Fatalf("first Run: %v", got.err)
	}

	if got.reason != turn.StopCompleted {
		t.Fatalf("first Run reason = %q, want %q", got.reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalQuietWindow)

	tr := finishWithTranscript(t, ex)

	if sends := inLineCount(tr, `"method":"session/send"`); sends != 1 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 1 (the rejected run wrote nothing)", sends)
	}
}

// The -32010 double-send guard, server side: a send that does reach the
// wire (a second machine on the same session — its own guard is idle) and
// is refused -32010 surfaces the same mapped error, matched by the numeric
// code, never the message text.
func TestDoubleSendRejectedServerSide(t *testing.T) {
	ex := startTurnExchange(t, serverRejectScenario(t))

	ctx, cancel := context.WithTimeout(context.Background(), cancelWindow)
	defer cancel()

	runDone := runAsync(ctx, ex.machine, "Reply with exactly: OK")
	waitAck(t, ex.machine)

	// A second machine bound to the same session has its own idle guard, so
	// the send proceeds to the wire — the server's -32010 reply must surface
	// as the same mapped error.
	m2 := turn.NewMachine(ex.conn, fixSessionID, turn.NewPollWatcher(ex.conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))

	reason, err := m2.Run(ctx, "second on the wire")
	if !errors.Is(err, turn.ErrTurnActive) {
		t.Fatalf("server-rejected Run error = %v, want ErrTurnActive (-32010 mapped by code)", err)
	}

	if reason != "" {
		t.Fatalf("rejected Run reason = %q, want empty", reason)
	}

	var callErr *zcode.CallError
	if !errors.As(err, &callErr) || callErr.Err.Code != zcode.CodeConcurrency {
		t.Fatalf("rejection must carry the -32010 wire code, got %v", err)
	}

	got := <-runDone

	if got.err != nil {
		t.Fatalf("first Run: %v", got.err)
	}

	if got.reason != turn.StopCompleted {
		t.Fatalf("first Run reason = %q, want %q", got.reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalQuietWindow)

	tr := finishWithTranscript(t, ex)

	if sends := inLineCount(tr, `"method":"session/send"`); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want 2 (both machines sent)", sends)
	}
}
