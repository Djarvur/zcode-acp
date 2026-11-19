package turn_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	// deliveryKind — every session/event envelope on the wire carried it
	// (EXP-TURN-01).
	deliveryKind = "desktop-continuous"

	// state patch reasons (capture shapes, EXP-TURN-01.jsonl:35 and the
	// completion-idle-patch corpus row).
	reasonPromptStarted   = "prompt_started"
	reasonPromptCompleted = "prompt_completed"
)

// state.updated patch.status values (dialect §5: only "running" was ever
// observed live; "idle" is the corpus row's by-construction leaving value).
const (
	statusRunning = "running"
	statusIdle    = "idle"
)

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
// verbatim result keys (accepted, sessionId, stateRevision) and the
// rule's ordered side effects (the fake's documented ordering: side
// effects before reply — the wire order the capture records).
func sendRule(t *testing.T, effects ...fake.SideEffect) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionSend,
		Reply: &fake.Reply{Result: rawOf(t, map[string]any{
			"accepted":      true,
			"sessionId":     fixSessionID,
			"stateRevision": 1,
		})},
		SideEffects: effects,
	}
}

// messagesRule answers session/messages polls with stable content — the
// quiet-window settling surface (the completion-quiet-window corpus
// shape: stable polls, presence/time evidence only).
func messagesRule(t *testing.T) fake.RequestRule {
	t.Helper()

	return fake.RequestRule{
		Method: zcode.MethodSessionMessages,
		Reply: &fake.Reply{Result: rawOf(t, map[string]any{
			"messages":  []any{},
			"sessionId": fixSessionID,
		})},
	}
}

// statePatch is a state.updated notification with a patch.status for the
// fixed session (the capture-verbatim envelope shape).
func statePatch(t *testing.T, status, reason string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodStateUpdated, map[string]any{
		"patch":     map[string]any{"status": status},
		"reason":    reason,
		"revision":  1,
		"scope":     "session",
		"sessionId": fixSessionID,
		"type":      "state.updated",
	})
}

// sessionEvent is a session/event turn-scoped carrier envelope.
func sessionEvent(t *testing.T, kind, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, zcode.MethodSessionEvent, map[string]any{
		"type":         kind,
		"seq":          4,
		"turnId":       turnID,
		"traceId":      "trace_fix1",
		"deliveryKind": deliveryKind,
	})
}

// v4Telemetry is the v4/telemetry/event carrier envelope (dialect §5:
// kind literals with dots, turnId carried).
func v4Telemetry(t *testing.T, kind, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodV4Telemetry, map[string]any{
		"kind":      kind,
		"turnId":    turnID,
		"eventSeq":  2,
		"sessionId": fixSessionID,
		"version":   1,
	})
}

// computerUse is the computer-use/operation-event carrier envelope
// (dialect §5: hyphenated kind literals, turnId carried).
func computerUse(t *testing.T, kind, turnID string) fake.SideEffect {
	t.Helper()

	return emit(t, turn.MethodComputerUseOperation, map[string]any{
		"kind":           kind,
		"turnId":         turnID,
		"sequenceNumber": 1,
		"sessionId":      fixSessionID,
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
			Message: "MethodNotFound",
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
// channel.
func startFake(fakeEnd *pipeEnd, sc *fake.Scenario) <-chan serveResult {
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

// The happy-turn contract: the machine lifts one fire-and-forget
// session/send into a synchronous turn — Run returns StopCompleted exactly
// once after the backend emitted the terminal signal, the resolving-signal
// record names the terminal event, and a second wait returns the SAME
// StopReason without re-resolving (PITFALL 8: the machine owns the
// completion).
func TestHappyTurnCompletedExactlyOnce(t *testing.T) {
	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	serveDone := startFake(fakeEnd, happyTurnScenario(t))

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	// Millisecond knobs: the terminal event resolves the turn long before
	// any quiet gate could arm — no test waits on wall-clock defaults.
	m := turn.NewMachine(conn, fixSessionID, turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))

	resolutions := make(chan turn.Resolution, 8)
	m.OnResolve = func(res turn.Resolution) { resolutions <- res }

	ctx, cancel := context.WithTimeout(context.Background(), turnTestTimeout)
	defer cancel()

	reason, err := m.Run(ctx, "Reply with exactly: OK")
	if err != nil {
		t.Fatalf("Run: %v", err) // RED: the stub fails this planned assertion
	}

	if reason != turn.StopCompleted {
		t.Fatalf("Run reason = %q, want %q", reason, turn.StopCompleted)
	}

	// The resolving-signal record is honest: the turnId-matched terminal
	// event is the D-02 primary completion signal.
	select {
	case res := <-resolutions:
		if res.Signal != turn.SignalTerminalEvent {
			t.Fatalf("resolving signal = %q, want %q", res.Signal, turn.SignalTerminalEvent)
		}

		if res.Reason != turn.StopCompleted {
			t.Fatalf("resolution reason = %q, want %q", res.Reason, turn.StopCompleted)
		}
	default:
		t.Fatal("no resolution delivered to the OnResolve hook")
	}

	// Exactly-once: a second wait returns the SAME StopReason and the
	// resolution hook has fired exactly once (no re-resolution).
	reason2, err := m.Wait(ctx)
	if err != nil {
		t.Fatalf("second Wait: %v", err)
	}

	if reason2 != turn.StopCompleted {
		t.Fatalf("second Wait reason = %q, want %q (idempotent)", reason2, turn.StopCompleted)
	}

	if n := len(resolutions); n != 1 {
		t.Fatalf("resolution hook fired %d times, want exactly 1", n)
	}

	_ = conn.Close()

	served := waitServed(t, serveDone)

	if served.err != nil {
		t.Fatalf("fake.Serve: %v", served.err)
	}
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
		return nil, fmt.Errorf("poll source: unexpected session %q", sessionID)
	}

	i := s.calls
	if i >= len(s.pages) {
		i = len(s.pages) - 1
	}

	s.calls++

	return s.pages[i], nil
}

// TestTurnWatcherDeliversUntilStopped: the PollWatcher delivers
// session-scoped envelopes to the callback and honors stop — stop is
// synchronous and after it returns no delivery happens (no watcher leak).
func TestTurnWatcherDeliversUntilStopped(t *testing.T) {
	pageA := []zcode.Envelope{{
		Method: zcode.MethodSessionMessages,
		Params: json.RawMessage(`{"messages":["a"],"sessionId":"sess_fix1"}`),
	}}
	pageB := []zcode.Envelope{{
		Method: zcode.MethodSessionMessages,
		Params: json.RawMessage(`{"messages":["a","b"],"sessionId":"sess_fix1"}`),
	}}

	source := &stubPollSource{pages: [][]zcode.Envelope{pageA, pageB, pageA}}

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
		t.Fatalf("Watch: %v", err) // RED: the stub fails this planned assertion
	}

	// Session-scoped envelopes arrive at the callback, advancing across the
	// scripted poll pages.
	saw := make([]string, 0, 2)
	deadline := time.After(2 * time.Second)

	for len(saw) < 2 {
		select {
		case obs := <-deliveries:
			if obs.Env != nil {
				saw = append(saw, string(obs.Env.Params))
			}
		case <-deadline:
			t.Fatalf("expected 2 distinct deliveries, got %d", len(saw))
		}
	}

	if saw[0] == saw[1] {
		t.Fatalf("deliveries did not advance across poll pages: %v", saw)
	}

	// Stop is synchronous: the watcher goroutine is joined before it
	// returns.
	stop()

	// Drain anything buffered from BEFORE the stop, then prove silence
	// past several poll intervals — no delivery may happen after stop
	// returned.
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
