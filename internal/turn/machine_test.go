package turn_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Turn machine contract tests (03-05, BRIDGE-04, TDD): the happy-turn
// exactly-once completion and the poll-watcher delivery/stop contract,
// driven over the in-process fake (the programmatic option per D-01 —
// scenarios built in Go with fixed synthetic ids and injectable
// millisecond timings, so no test waits on wall-clock defaults).

const (
	fixSessionID = "sess_fix1"
	fixTurnID    = "turn_fix1"

	// turnTestTimeout bounds each exchange; the GREEN run completes in
	// milliseconds and a RED run fails fast on its own assertion.
	turnTestTimeout = 10 * time.Second

	// stopSilence is how long the watcher test waits past several poll
	// intervals to prove no delivery happens after stop returns.
	stopSilence = 60 * time.Millisecond

	// variantWindow bounds one completion-variant Run: GREEN resolves the
	// quiet variant in ~150 ms (floor 80 ms + settle 40 ms at the injected
	// knobs); a RED run fails fast on its planned assertion.
	variantWindow = 2 * time.Second

	// deliveryKind — every session/event envelope on the wire carried it
	// (EXP-TURN-01).
	deliveryKind = "desktop-continuous"

	// state patch reasons (capture shapes, EXP-TURN-01.jsonl:35 and the
	// completion-idle-patch corpus row).
	reasonPromptStarted   = "prompt_started"
	reasonPromptCompleted = "prompt_completed"

	// wire keys of the turn-scoped carrier params (dialect §5: every
	// carrier carries the turnId; session envelopes are session-scoped).
	wireKeySessionID = "sessionId"
	wireKeyTurnID    = "turnId"

	// unknownMethodReply is the fake's unknown-method -32601 reply message
	// (goconst: the literal repeats across the programmatic scenarios).
	unknownMethodReply = "MethodNotFound"
)

// state.updated patch.status values (dialect §5: only "running" was ever
// observed live; "idle" is the corpus row's by-construction leaving value).
const (
	statusRunning = "running"
	statusIdle    = "idle"
)

// errUnexpectedSession is the stub poll source's sentinel for a poll
// addressed to a foreign session.
var errUnexpectedSession = errors.New("poll source: unexpected session")

// rawOf marshals one scenario payload for the wire.
func rawOf(t *testing.T, v any) json.RawMessage {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal scenario payload: %v", err)
	}

	return b
}

// emit builds one notification side effect.
func emit(t *testing.T, method string, params any) fake.SideEffect {
	t.Helper()

	return fake.SideEffect{Emit: &fake.Emit{Method: method, Params: rawOf(t, params)}}
}

// sendRule is the session/send rule: the fast ack with the EXP-TURN-01
// verbatim result keys (accepted, sessionId, stateRevision) and the rule's
// ordered side effects (the fake's documented ordering: side effects
// before reply — the wire order the capture records).
func sendRule(t *testing.T, effects ...fake.SideEffect) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionSend,
		Reply: &fake.Reply{Result: rawOf(t, map[string]any{
			"accepted":       true,
			wireKeySessionID: fixSessionID,
			"stateRevision":  1,
		})},
		SideEffects: effects,
	}
}

// messagesRule answers session/messages polls with stable content — the
// quiet-window settling surface (the completion-quiet-window corpus shape:
// stable polls, presence/time evidence only).
func messagesRule(t *testing.T) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionMessages,
		Reply: &fake.Reply{Result: rawOf(t, map[string]any{
			"messages":       []any{},
			wireKeySessionID: fixSessionID,
		})},
	}
}

// statePatch is a state.updated notification with a patch.status for the
// fixed session (the capture-verbatim envelope shape).
func statePatch(t *testing.T, status, reason string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodStateUpdated, map[string]any{
		"patch":          map[string]any{"status": status},
		"reason":         reason,
		"revision":       1,
		"scope":          "session",
		wireKeySessionID: fixSessionID,
		"type":           "state.updated",
	})
}

// sessionEvent is a session/event turn-scoped carrier envelope.
func sessionEvent(t *testing.T, kind, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodSessionEvent, map[string]any{
		"type":         kind,
		"seq":          4,
		wireKeyTurnID:  turnID,
		"traceId":      "trace_fix1",
		"deliveryKind": deliveryKind,
	})
}

// v4Telemetry is the v4/telemetry/event carrier envelope (dialect §5:
// dot-spelled kind literals, turnId carried, eventSeq sequence key).
func v4Telemetry(t *testing.T, kind, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodV4Telemetry, map[string]any{
		"kind":           kind,
		wireKeyTurnID:    turnID,
		"eventSeq":       2,
		wireKeySessionID: fixSessionID,
		"version":        1,
	})
}

// computerUse is the computer-use/operation-event carrier envelope
// (dialect §5: hyphen-spelled kind literals, turnId carried).
func computerUse(t *testing.T, kind, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodComputerUseOperation, map[string]any{
		"kind":           kind,
		wireKeyTurnID:    turnID,
		"sequenceNumber": 1,
		wireKeySessionID: fixSessionID,
		"eventId":        "ev_fix1",
		"timestamp":      "2026-01-01T00:00:00Z",
	})
}

// happyTurnScenario is the programmatic happy-turn script: send ack with
// the running patch, turn.started, and the terminal turn.completed session
// event (turnId-matched), plus stable session/messages for the watcher's
// polls.
func happyTurnScenario(t *testing.T) *fake.Scenario {
	t.Helper()

	return &fake.Scenario{
		Name: "turn-happy",
		OnRequest: []fake.RequestRule{
			sendRule(t,
				statePatch(t, statusRunning, reasonPromptStarted),
				sessionEvent(t, turn.KindTurnStarted, fixTurnID),
				sessionEvent(t, turn.KindTurnCompleted, fixTurnID),
			),
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// pipeEnd joins one io.Pipe half-pair into an io.ReadWriteCloser end.
type pipeEnd struct {
	r *io.PipeReader
	w *io.PipeWriter
}

// Read implements io.Reader.
func (p *pipeEnd) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

// Write implements io.Writer.
func (p *pipeEnd) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

// Close tears down both halves; the peer sees EOF.
func (p *pipeEnd) Close() error {
	return errors.Join(p.r.Close(), p.w.Close())
}

// roundTripPipes wires the two io.Pipe half-pairs as host and fake stream
// ends: frames on the pipe are the real boundary (D-01 mode A).
func roundTripPipes() (hostEnd, fakeEnd *pipeEnd) {
	toFakeR, toFakeW := io.Pipe()
	toHostR, toHostW := io.Pipe()

	hostEnd = &pipeEnd{r: toHostR, w: toFakeW}
	fakeEnd = &pipeEnd{r: toFakeR, w: toHostW}

	return hostEnd, fakeEnd
}

// serveResult carries the fake's transcript and error across the goroutine
// boundary (channel ordering replaces a mutex).
type serveResult struct {
	transcript fake.Transcript
	err        error
}

// startFake runs fake.Serve on a goroutine and returns its completion
// channel. The stream end is an io.ReadWriteCloser so harness wrappers (the
// cancel tier's read gate) can wrap the raw pipe end.
func startFake(fakeEnd io.ReadWriteCloser, sc *fake.Scenario) <-chan serveResult {
	serveDone := make(chan serveResult, 1)

	go func() {
		tr, serveErr := fake.Serve(fakeEnd, sc)
		serveDone <- serveResult{transcript: tr, err: serveErr}
	}()

	return serveDone
}

// waitServed waits for fake.Serve to return after the host closed its end.
func waitServed(t *testing.T, serveDone <-chan serveResult) serveResult {
	t.Helper()

	var served serveResult

	select {
	case served = <-serveDone:
	case <-time.After(turnTestTimeout):
		t.Fatal("fake.Serve did not return after stdin EOF")
	}

	return served
}

// resolutionSpy is the exactly-once observation hook: a synchronized
// counter/record of every OnResolve delivery (the hook runs on observing
// goroutines; the test reads it under the same lock).
type resolutionSpy struct {
	mu  sync.Mutex
	got []turn.Resolution
}

// record implements the OnResolve hook signature.
func (s *resolutionSpy) record(res turn.Resolution) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.got = append(s.got, res)
}

// snapshot returns the recorded resolutions.
func (s *resolutionSpy) snapshot() []turn.Resolution {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.got)
}

// turnExchange bundles one machine exchange against the fake: the conn,
// the machine with millisecond quiet knobs, and the resolution spy.
type turnExchange struct {
	conn      *zcode.Conn
	machine   *turn.Machine
	spy       *resolutionSpy
	serveDone <-chan serveResult
}

// startTurnExchange wires the fake, the conn, a machine bound to the fixed
// session with injectable millisecond quiet knobs, and the resolution spy.
// Cleanup joins both ends.
func startTurnExchange(t *testing.T, sc *fake.Scenario) *turnExchange {
	t.Helper()

	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	serveDone := startFake(fakeEnd, sc)

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	m := turn.NewMachine(conn, fixSessionID, turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))

	spy := &resolutionSpy{}
	m.OnResolve = spy.record

	return &turnExchange{conn: conn, machine: m, spy: spy, serveDone: serveDone}
}

// finish closes the conn (the fake sees stdin EOF and drains) and asserts
// the scripted run ended clean.
func (ex *turnExchange) finish(t *testing.T) {
	t.Helper()

	_ = ex.conn.Close()

	served := waitServed(t, ex.serveDone)

	if served.err != nil {
		t.Fatalf("fake.Serve: %v", served.err)
	}
}

// assertResolution asserts exactly one resolution carrying the expected
// reason and resolving-signal record.
func assertResolution(t *testing.T, spy *resolutionSpy, reason turn.StopReason, signal turn.SignalKind) {
	t.Helper()

	got := spy.snapshot()

	if len(got) != 1 {
		t.Fatalf("resolution hook fired %d times, want exactly 1", len(got))
	}

	if got[0].Reason != reason {
		t.Fatalf("resolution reason = %q, want %q", got[0].Reason, reason)
	}

	if got[0].Signal != signal {
		t.Fatalf("resolving signal = %q, want %q", got[0].Signal, signal)
	}
}

// The happy-turn contract: the machine lifts one fire-and-forget
// session/send into a synchronous turn — Run returns StopCompleted exactly
// once after the backend emitted the terminal signal, the resolving-signal
// record names the terminal event, and a second wait returns the SAME
// StopReason without re-resolving (PITFALL 8: the machine owns the
// completion).
func TestHappyTurnCompletedExactlyOnce(t *testing.T) {
	ex := startTurnExchange(t, happyTurnScenario(t))

	ctx, cancel := context.WithTimeout(context.Background(), turnTestTimeout)
	defer cancel()

	reason, err := ex.machine.Run(ctx, "Reply with exactly: OK")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	// Exactly-once: a second wait returns the SAME StopReason and the
	// resolution hook has still fired exactly once (no re-resolution).
	reason2, err := ex.machine.Wait(ctx)
	if err != nil {
		t.Fatalf("second Wait: %v", err)
	}

	if reason2 != turn.StopCompleted {
		t.Fatalf("second Wait reason = %q, want %q (idempotent)", reason2, turn.StopCompleted)
	}

	assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

	ex.finish(t)
}

// stubPollSource returns scripted poll pages; calls counts the polls the
// watcher issued.
type stubPollSource struct {
	mu    sync.Mutex
	pages [][]zcode.Envelope
	calls int
}

// Poll implements PollSource.
func (s *stubPollSource) Poll(_ context.Context, sessionID string) ([]zcode.Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sessionID != fixSessionID {
		return nil, errUnexpectedSession
	}

	i := s.calls
	if i >= len(s.pages) {
		i = len(s.pages) - 1
	}

	s.calls++

	return s.pages[i], nil
}

// pollPage builds one session-scoped session/messages poll page carrying
// the given message ids (opaque payloads — presence evidence only).
func pollPage(t *testing.T, messages ...string) []zcode.Envelope {
	t.Helper()

	b, err := json.Marshal(map[string]any{
		"messages":       messages,
		wireKeySessionID: fixSessionID,
	})
	if err != nil {
		t.Fatalf("marshal poll page: %v", err)
	}

	return []zcode.Envelope{{
		Method: zcode.MethodSessionMessages,
		Params: b,
	}}
}

// collectDeliveries reads want envelope deliveries off the watcher channel.
func collectDeliveries(t *testing.T, deliveries <-chan turn.Observation, want int) []string {
	t.Helper()

	saw := make([]string, 0, want)
	deadline := time.After(2 * time.Second)

	for len(saw) < want {
		select {
		case obs := <-deliveries:
			if obs.Env != nil {
				saw = append(saw, string(obs.Env.Params))
			}
		case <-deadline:
			t.Fatalf("expected %d distinct deliveries, got %d", want, len(saw))
		}
	}

	return saw
}

// assertNoDeliveryAfterStop drains pre-stop buffered deliveries and proves
// silence past several poll intervals — no delivery may happen after stop
// returned (no watcher leak).
func assertNoDeliveryAfterStop(t *testing.T, deliveries <-chan turn.Observation) {
	t.Helper()

drain:
	for {
		select {
		case <-deliveries:
		default:
			break drain
		}
	}

	violations := 0
	silence := time.After(stopSilence)

	for {
		select {
		case <-deliveries:
			violations++
		case <-silence:
			if violations > 0 {
				t.Fatalf("%d deliveries arrived after stop returned", violations)
			}

			return
		}
	}
}

// The watcher contract: the PollWatcher delivers session-scoped envelopes
// to the callback and honors stop — stop is synchronous and after it
// returns no delivery happens.
func TestTurnWatcherDeliversUntilStopped(t *testing.T) {
	source := &stubPollSource{
		pages: [][]zcode.Envelope{
			pollPage(t, "a"),
			pollPage(t, "a", "b"),
			pollPage(t, "a"),
		},
	}

	w := turn.NewPollWatcher(nil, turn.PollConfig{
		Source:         source,
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), turnTestTimeout)
	defer cancel()

	deliveries := make(chan turn.Observation, 64)
	stop, err := w.Watch(ctx, fixSessionID, func(obs turn.Observation) {
		deliveries <- obs
	})
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	// Session-scoped envelopes arrive at the callback, advancing across the
	// scripted poll pages.
	saw := collectDeliveries(t, deliveries, 2)

	if saw[0] == saw[1] {
		t.Fatalf("deliveries did not advance across poll pages: %v", saw)
	}

	stop()

	assertNoDeliveryAfterStop(t, deliveries)
}

// completionScenario is the completion-variant script base: the send
// rule's side effects ARE the variant's signal set, and stable
// session/messages keeps the watcher's polls quiet unless the variant IS
// the quiet window.
func completionScenario(t *testing.T, effects ...fake.SideEffect) *fake.Scenario {
	t.Helper()

	return &fake.Scenario{
		Name: "turn-completion-variant",
		OnRequest: []fake.RequestRule{
			sendRule(t, effects...),
			messagesRule(t),
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: unknownMethodReply,
		}},
		EOFDrainMs: 50,
	}
}

// assertQuietAfterFloor asserts the quiet-window resolution fired strictly
// after the send-latency floor elapsed post-ack (the floor = FloorMultiple
// x QuietMs measured from the send ack — the probe's first live-verified
// gate; a quiet fire before the floor is first-token latency masquerading
// as silence).
func assertQuietAfterFloor(t *testing.T, ex *turnExchange) {
	t.Helper()

	got := ex.spy.snapshot()

	if len(got) != 1 {
		t.Fatalf("resolution hook fired %d times, want exactly 1", len(got))
	}

	floor := time.Duration(2) * time.Duration(40) * time.Millisecond // the injected knobs

	if got[0].At.Before(ex.machine.AckAt().Add(floor)) {
		t.Fatalf("quiet resolution at %v fired before ack %v + floor %v elapsed",
			got[0].At, ex.machine.AckAt(), floor)
	}
}

// The completion-variant contract (D-02 ordered union): terminal event,
// idle patch, and quiet window each resolve the turn exactly once with the
// correct StopReason and resolving-signal record. The idle patch resolves
// only when the status LEAVES running (the running patch itself never
// resolves); the quiet window resolves only after the send-latency floor
// and the settle gate (no test scenario resolves on absence of evidence).
func TestCompletionVariants(t *testing.T) {
	variants := []struct {
		name     string
		scenario func(*testing.T) *fake.Scenario
		want     turn.StopReason
		signal   turn.SignalKind
	}{
		{
			name: "terminal-event",
			scenario: func(t *testing.T) *fake.Scenario {
				t.Helper()

				return completionScenario(t,
					statePatch(t, statusRunning, reasonPromptStarted),
					sessionEvent(t, turn.KindTurnStarted, fixTurnID),
					sessionEvent(t, turn.KindTurnCompleted, fixTurnID),
				)
			},
			want:   turn.StopCompleted,
			signal: turn.SignalTerminalEvent,
		},
		{
			name: "idle-patch",
			scenario: func(t *testing.T) *fake.Scenario {
				t.Helper()

				return completionScenario(t,
					statePatch(t, statusRunning, reasonPromptStarted),
					statePatch(t, statusIdle, reasonPromptCompleted),
				)
			},
			want:   turn.StopCompleted,
			signal: turn.SignalIdlePatch,
		},
		{
			name: "quiet-window",
			scenario: func(t *testing.T) *fake.Scenario {
				t.Helper()

				return completionScenario(t,
					statePatch(t, statusRunning, reasonPromptStarted),
					sessionEvent(t, turn.KindTurnStarted, fixTurnID),
				)
			},
			want:   turn.StopCompleted,
			signal: turn.SignalQuietWindow,
		},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			runCompletionVariant(t, v.name, v.scenario(t), v.want, v.signal)
		})
	}
}

// runCompletionVariant drives one completion variant: Run must resolve
// within the exchange window with the expected reason and resolving-signal
// record; the quiet variant additionally asserts its fire is gated strictly
// after the send-latency floor post-ack.
func runCompletionVariant(t *testing.T, name string, sc *fake.Scenario, want turn.StopReason, signal turn.SignalKind) {
	t.Helper()

	ex := startTurnExchange(t, sc)

	ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
	defer cancel()

	reason, err := ex.machine.Run(ctx, "Reply with exactly: OK")
	if err != nil {
		// The planned assertion: this variant must resolve within the
		// exchange window.
		t.Fatalf("Run (%s): turn did not resolve within the exchange window: %v", name, err)
	}

	if reason != want {
		t.Fatalf("Run (%s) reason = %q, want %q", name, reason, want)
	}

	assertResolution(t, ex.spy, want, signal)

	if signal == turn.SignalQuietWindow {
		assertQuietAfterFloor(t, ex)
	}

	ex.finish(t)
}

// carrierFunc builds one carrier envelope with a kind literal and a turnId.
type carrierFunc func(*testing.T, string, string) fake.SideEffect

// carrierScenario is the carrier-tolerance script base: turn.started via
// the session/event carrier records the turn's id, then a FOREIGN-turnId
// terminal (which must NOT resolve the turn) precedes the turnId-matched
// terminal on the carrier under test.
func carrierScenario(t *testing.T, carrier carrierFunc, terminalKind string) *fake.Scenario {
	t.Helper()

	return completionScenario(t,
		statePatch(t, statusRunning, reasonPromptStarted),
		sessionEvent(t, turn.KindTurnStarted, fixTurnID),
		carrier(t, terminalKind, "turn_foreign1"),
		carrier(t, terminalKind, fixTurnID),
	)
}

// The carrier-tolerance contract (RESEARCH Open Question 1 resolution):
// the same terminal classification fires whichever carrier delivers the
// turnId-matched terminal signal — session/event (type turn.completed),
// v4/telemetry/event (kind turn.completed), and computer-use/operation-
// event (kind turn-completed). A foreign turnId never resolves the turn.
func TestTerminalCarrierTolerance(t *testing.T) {
	carriers := []struct {
		name    string
		carrier carrierFunc
		kind    string
	}{
		{"session-event", sessionEvent, turn.KindTurnCompleted},
		{"v4-telemetry", v4Telemetry, turn.KindTurnCompleted},
		{"computer-use", computerUse, turn.KindTurnCompletedCU},
	}

	for _, c := range carriers {
		t.Run(c.name, func(t *testing.T) {
			ex := startTurnExchange(t, carrierScenario(t, c.carrier, c.kind))

			ctx, cancel := context.WithTimeout(context.Background(), variantWindow)
			defer cancel()

			reason, err := ex.machine.Run(ctx, "Reply with exactly: OK")
			if err != nil {
				t.Fatalf("Run (%s): turn did not resolve within the exchange window: %v", c.name, err)
			}

			if reason != turn.StopCompleted {
				t.Fatalf("Run (%s) reason = %q, want %q", c.name, reason, turn.StopCompleted)
			}

			// Exactly once: the spy count also proves the foreign-turnId
			// terminal never resolved the turn.
			assertResolution(t, ex.spy, turn.StopCompleted, turn.SignalTerminalEvent)

			ex.finish(t)
		})
	}
}

// WR-01 router-event fan-out (05-03 TDD RED, machine tier): the FIRST
// machine's degraded-resolution path must survive a second machine's
// construction. RED: NewMachine's SetOnRouterEvent registration is
// last-wins — machine2's construction steals the hook and turn 2 on
// machine 1 never resolves (assertion failure at the bound).
func TestTwoMachinesDegradedSignalNotStolen(t *testing.T) {
	cipherReverse := fake.SideEffect{Reverse: &fake.Reverse{
		Method:    "interaction/requestCredentialCipher",
		Params:    rawOf(t, map[string]any{"requestId": "req-x", "operation": "decrypt", "value": "enc:v2:os:c2VjcmV0"}),
		TimeoutMs: 200,
	}}

	sc := &fake.Scenario{
		Name: "turn-cipher-expiry-x2",
		OnRequest: []fake.RequestRule{
			{
				Method:      zcode.MethodSessionSend,
				Reply:       sendRule(t).Reply,
				SideEffects: []fake.SideEffect{cipherReverse},
			},
			messagesRule(t),
		},
	}

	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	startFake(fakeEnd, sc)

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	// The deadline expiry must land well inside the turn bound.
	conn.RouterDeadline = func(string) time.Duration { return 60 * time.Millisecond }

	spy1 := &resolutionSpy{}
	machine1 := turn.NewMachine(conn, fixSessionID, turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))
	machine1.OnResolve = spy1.record

	// Turn 1: the expiry reaches machine 1 — degraded, exactly once.
	ctx1, cancel1 := context.WithTimeout(context.Background(), turnTestTimeout)
	defer cancel1()

	reason1, err1 := machine1.Run(ctx1, "prompt one")

	if err1 != nil {
		t.Fatalf("turn 1 Run: %v", err1)
	}

	if reason1 != turn.StopError {
		t.Fatalf("turn 1 reason = %q, want %q (degraded expiry)", reason1, turn.StopError)
	}

	res1 := spy1.snapshot()

	if len(res1) != 1 || res1[0].Signal != turn.SignalDegraded {
		t.Fatalf("turn 1 resolutions = %+v, want exactly one degraded", res1)
	}

	// The defect shape: a second session's machine construction steals the
	// single router-event slot.
	machine2 := turn.NewMachine(conn, "sess_2", turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))

	spy2 := &resolutionSpy{}
	machine2.OnResolve = spy2.record

	// Turn 2 on machine 1: GREEN delivers the expiry through the fan-out —
	// machine 1 resolves degraded again, machine 2 early-returns (idle).
	// RED: the slot holds machine 2's observer, so machine 1 hangs.
	ctx2, cancel2 := context.WithTimeout(context.Background(), turnTestTimeout)
	defer cancel2()

	reason2, err2 := machine1.Run(ctx2, "prompt two")

	if err2 != nil {
		t.Fatalf("turn 2 Run: %v", err2)
	}

	if reason2 != turn.StopError {
		t.Fatalf("turn 2 reason = %q, want %q — the first machine's degraded-resolution path was stolen (WR-01)", reason2, turn.StopError)
	}

	res2 := spy2.snapshot()

	if len(res2) != 0 {
		t.Fatalf("machine 2 recorded %+v, want zero (idle early-return)", res2)
	}
}

// WR-07 teardown baseline (05-03 Task 2): constructing a machine registers
// two conn observers; CloseObservers must release both back to baseline —
// idempotently — so session/close teardown can finally release observers.
func TestMachineCloseObserversReturnsToBaseline(t *testing.T) {
	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	startFake(fakeEnd, &fake.Scenario{Name: "teardown-baseline"})

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	notifBase, routerBase := conn.ObserverCounts()

	m := turn.NewMachine(conn, fixSessionID, turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))

	notifNow, routerNow := conn.ObserverCounts()

	if notifNow != notifBase+1 || routerNow != routerBase+1 {
		t.Fatalf("post-construction counts = (%d, %d), want baseline+1 each (%d, %d)", notifNow, routerNow, notifBase, routerBase)
	}

	m.CloseObservers()
	m.CloseObservers() // idempotent

	notifAfter, routerAfter := conn.ObserverCounts()

	if notifAfter != notifBase || routerAfter != routerBase {
		t.Fatalf("post-teardown counts = (%d, %d), want the (%d, %d) baseline", notifAfter, routerAfter, notifBase, routerBase)
	}
}
