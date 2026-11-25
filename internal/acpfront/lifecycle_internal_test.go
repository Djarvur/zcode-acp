package acpfront

// WR-05 (plan 07-01) in-package lifecycle locks. These two tests need the
// front's unexported close-gate state (closingMu/closing) and registry —
// the external-package suite (lifecycle_test.go) asserts the same contracts
// through the wire surfaces only. They ride the composed back leg directly
// (fake.Serve ⇄ zcode.Conn ⇄ Front — no SDK client leg: ResumeSession and
// CloseSession are front-level APIs; the direct registration mirrors
// NewSession's tail exactly).
//
//	TestResumeCloseInterleave — the stress interleave: a resume released
//	  together with a close over the same session id ends in exactly one
//	  of the two honest outcomes — the coded closing rejection, or a
//	  success with no teardown already completed under it. A success that
//	  registered into a mid-teardown (marked closing) session is the
//	  WR-05 leak arm.
//	TestRegisterIfOpen — the serialization contract lock: while the test
//	  goroutine holds closingMu with the session marked closing, a resume
//	  that has already passed its backend subscribe CANNOT complete
//	  registration — registry.Get keeps failing until the mutex is
//	  released — and the mark still set, the release answers the coded
//	  closing rejection. The liveness control re-runs the same machinery
//	  unmarked: registration lands and the registry holds the session.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/core"
	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

const (
	// interleaveSID is the scripted session id both legs race over (the
	// create/resume snapshot digs it back out unchanged).
	interleaveSID = "sess_stress"

	// interleaveIterations is the stress interleave's iteration count (the
	// plan's >=2000 bar, run under -race).
	interleaveIterations = 2000

	// interleaveWait bounds one racer's resolution; the composed loop
	// completes in milliseconds, so this fails fast.
	interleaveWait = 5 * time.Second

	// interleaveSubscribeDelayMs parks the serialization probe's resume on
	// its scripted subscribe reply — the deterministic "already passed its
	// subscribe" parking spot the held-mutex phase needs.
	interleaveSubscribeDelayMs = 50

	// interleaveHoldSamples is the held-mutex sampling count across the
	// hold span (the third sample crosses the subscribe reply: the resume
	// is parked INSIDE the closingMu-guarded section by then).
	interleaveHoldSamples = 3

	// interleaveHoldSampleGapMs spaces the held-mutex samples.
	interleaveHoldSampleGapMs = 30
)

// interleaveScenario scripts the four back-leg shapes the race needs
// (create/resume/subscribe/close), built to the fail-loud loader rules —
// every reply carries exactly one usable payload, and eof_drain_ms is set.
// Each reply rule answers unlimited repeats (no Times budget): both racers
// issue their own legs. subscribeDelayMs parks the resume on its scripted
// subscribe reply (0 = immediate, the stress arm).
func interleaveScenario(subscribeDelayMs int) *fake.Scenario {
	return &fake.Scenario{
		Name: "resume-close-interleave",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionCreate, Reply: &fake.Reply{
				Result: json.RawMessage(`{"session":{"sessionId":"` + interleaveSID + `"}}`),
			}},
			{Method: zcode.MethodSessionResume, Reply: &fake.Reply{
				Result: json.RawMessage(`{"session":{"sessionId":"` + interleaveSID + `"}}`),
			}},
			{
				Method:  zcode.MethodSessionSubscribe,
				DelayMs: subscribeDelayMs,
				Reply: &fake.Reply{Result: json.RawMessage(
					`{"sessionId":"` + interleaveSID + `","eventSeq":0,"events":[]}`)},
			},
			{Method: zcode.MethodSessionClose, Reply: &fake.Reply{
				Result: json.RawMessage(`{"closed":true}`),
			}},
		},
		EOFDrainMs: 1,
	}
}

// pipeEnd joins one io.Pipe half-pair into an io.ReadWriteCloser end (the
// composed harness's rwCloser shape, local to this in-package file).
type pipeEnd struct {
	r *io.PipeReader
	w *io.PipeWriter
}

// Read passes through to the pipe reader.
func (p *pipeEnd) Read(b []byte) (int, error) { return p.r.Read(b) }

// Write passes through to the pipe writer.
func (p *pipeEnd) Write(b []byte) (int, error) { return p.w.Write(b) }

// Close tears down both halves; the peer sees EOF.
func (p *pipeEnd) Close() error { return errors.Join(p.r.Close(), p.w.Close()) }

// newInterleaveFront composes one fresh back leg (fake.Serve ⇄ zcode.Conn)
// with a Front over it and returns the closer the caller MUST run at
// iteration end — 2000+ fresh legs per test must not stack up until the
// test's cleanup phase. The closer returns the fake's recorded transcript
// (serve fully drained — the wire-order assertion surface for the flows
// that need it). Options configure the Front at construction (the stall /
// orphan seam arms); existing call sites stay source-compatible.
func newInterleaveFront(
	t *testing.T, sc *fake.Scenario, opts ...Option,
) (front *Front, closeLegs func() fake.Transcript) {
	t.Helper()

	toFakeR, toFakeW := io.Pipe()
	toHostR, toHostW := io.Pipe()

	hostEnd := &pipeEnd{r: toHostR, w: toFakeW}
	fakeEnd := &pipeEnd{r: toFakeR, w: toHostW}

	serveDone := make(chan fake.Transcript, 1)

	go func() {
		tr, _ := fake.Serve(fakeEnd, sc)
		serveDone <- tr
	}()

	conn := zcode.NewConn(hostEnd)
	front = New(conn, opts...)

	var once sync.Once

	closeLegs = func() fake.Transcript {
		var tr fake.Transcript

		once.Do(func() {
			_ = conn.Close()
			_ = fakeEnd.Close()
			tr = <-serveDone
		})

		return tr
	}

	return front, closeLegs
}

// openInterleaveSession registers the raced-over session exactly the way
// NewSession's tail does (machine on the conn fan-outs + registry entry +
// recorded cwd) without the acpport layer — CloseSession's Get gate then
// sees a live registered session.
func openInterleaveSession(t *testing.T, f *Front, cwd string) {
	t.Helper()

	machine := turn.NewMachine(f.conn, interleaveSID, turn.NewPollWatcher(f.conn, turn.PollConfig{}))
	f.recordCwd(interleaveSID, cwd)
	f.registry.Register(core.Session{ID: interleaveSID, Machine: machine})
}

// awaitRacer waits for one racer's resolution inside the wait bound.
func awaitRacer(t *testing.T, done <-chan error, what string, iteration int) error {
	t.Helper()

	select {
	case err := <-done:
		return err
	case <-time.After(interleaveWait):
		t.Fatalf("iteration %d: %s never resolved within the wait bound", iteration, what)

		return nil // unreachable
	}
}

// assertClosingRejection requires the honest close-wins arm: the coded
// invalid-request closing rejection, never another error class.
func assertClosingRejection(t *testing.T, err error, iteration int) {
	t.Helper()

	var seamErr *acpport.Error
	if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInvalidRequest {
		t.Fatalf("iteration %d: resume error = %v, want the coded invalid-request closing rejection", iteration, err)
	}
}

// TestResumeCloseInterleave is the WR-05 stress probe: >=2000 iterations of
// a resume released together with a close over the same session id, each
// ending in exactly one of the two honest outcomes — the coded closing
// rejection, or a resume success. A success that registered its machine
// into a session whose teardown was already completed (the WR-05 leak arm:
// the machine escapes close's snapshot-based release) surfaces as a resume
// success the session's closed state contradicts — the registerIfOpen
// section makes the whole surface deterministic.
func TestResumeCloseInterleave(t *testing.T) {
	cwd := t.TempDir()
	ctx := context.Background()

	for i := range interleaveIterations {
		f, closeLegs := newInterleaveFront(t, interleaveScenario(0))
		openInterleaveSession(t, f, cwd)

		start := make(chan struct{})
		resumeDone := make(chan error, 1)
		closeDone := make(chan error, 1)

		go func() {
			<-start
			_, err := f.ResumeSession(ctx, interleaveSID, cwd)
			resumeDone <- err
		}()

		go func() {
			<-start
			closeDone <- f.CloseSession(ctx, interleaveSID)
		}()

		close(start)

		closeErr := awaitRacer(t, closeDone, "close", i)
		resumeErr := awaitRacer(t, resumeDone, "resume", i)

		if closeErr != nil {
			t.Fatalf("iteration %d: close must always succeed over a registered session, got: %v", i, closeErr)
		}

		if resumeErr != nil {
			assertClosingRejection(t, resumeErr, i)
		}

		closeLegs()
	}
}

// TestRegisterIfOpen is the WR-05 serialization contract lock. Phase 1
// parks a resume on its scripted subscribe reply, then holds closingMu with
// the session marked closing: the resume — past its subscribe once the
// reply lands mid-hold — cannot complete registration while the mutex is
// held (registry.Get keeps failing across every held-span sample), and
// releasing it with the mark still set answers the coded closing rejection
// with nothing ever registered. Phase 2 is the liveness control: the same
// machinery with no mark registers and the registry holds the session.
//
//nolint:funlen // the two-phase lock is one story; splitting would scatter the serialization contract
func TestRegisterIfOpen(t *testing.T) {
	cwd := t.TempDir()
	ctx := context.Background()

	f, closeLegs := newInterleaveFront(t, interleaveScenario(interleaveSubscribeDelayMs))

	resumeDone := make(chan error, 1)

	go func() {
		_, err := f.ResumeSession(ctx, interleaveSID, cwd)
		resumeDone <- err
	}()

	// The resume is inside its backend leg (the subscribe reply parks it
	// for the whole hold span) — grab the close gate and mark the session
	// closing while it is parked.
	time.Sleep(5 * time.Millisecond)

	f.closingMu.Lock()

	if f.closing == nil {
		f.closing = make(map[string]bool)
	}

	f.closing[interleaveSID] = true

	for sample := range interleaveHoldSamples {
		if _, getErr := f.registry.Get(interleaveSID); getErr == nil {
			f.closingMu.Unlock()
			closeLegs()
			t.Fatalf("sample %d: registry holds the session while closingMu is held with the mark set — "+
				"registration escaped the close-gate critical section (WR-05)", sample)
		}

		time.Sleep(interleaveHoldSampleGapMs * time.Millisecond)
	}

	// The last sample crossed the subscribe reply: the resume received it
	// and parked on THIS goroutine's held mutex — still unregistered.

	f.closingMu.Unlock() // the mark STAYS set

	resumeErr := awaitRacer(t, resumeDone, "resume", 0)
	assertClosingRejection(t, resumeErr, 0)

	if _, getErr := f.registry.Get(interleaveSID); getErr == nil {
		closeLegs()
		t.Fatal("the rejected resume left a registration behind — the closing rejection must register nothing")
	}

	closeLegs()

	// Phase 2 — liveness control: unmarked, the same machinery registers.
	f2, closeLegs2 := newInterleaveFront(t, interleaveScenario(0))
	defer closeLegs2()

	resumeDone2 := make(chan error, 1)

	go func() {
		_, err := f2.ResumeSession(ctx, interleaveSID, cwd)
		resumeDone2 <- err
	}()

	if err := awaitRacer(t, resumeDone2, "unmarked resume", 0); err != nil {
		t.Fatalf("the unmarked control resume must succeed, got: %v", err)
	}

	if _, getErr := f2.registry.Get(interleaveSID); getErr != nil {
		t.Fatalf("the successful control resume must leave the session registered, got: %v", getErr)
	}
}

// resumedStallWindow bounds the resumed-session stall exchange. RED (without
// the seam) needed room for the production quiet floor (7.5 s) plus the
// reconcile; GREEN bounds the exchange at 2 s — the 60 ms watchdog must fire
// far inside it, and the 5 s production default would break the bound.
const resumedStallWindow = 2 * time.Second

// The resumed arm's watchdog budgets (the same seam the session/new arm
// composes).
const (
	resumedStallBudget     = 60 * time.Millisecond
	resumedStallToolBudget = 120 * time.Millisecond
)

// resumedStallScenario scripts a resumed session's two-turn stall exchange:
// resume + subscribe, a first send answered only by its ack (the silent
// backend turn), a stop, and a second send whose running patch + terminal
// complete normally; the messages projection carries the stored reply so
// the completion reconcile returns promptly.
func resumedStallScenario() *fake.Scenario {
	return &fake.Scenario{
		Name: "resume-stall-seam",
		OnRequest: []fake.RequestRule{
			{Method: zcode.MethodSessionResume, Reply: &fake.Reply{Result: json.RawMessage(
				`{"session":{"sessionId":"` + interleaveSID + `"}}`)}},
			{Method: zcode.MethodSessionSubscribe, Reply: &fake.Reply{Result: json.RawMessage(
				`{"sessionId":"` + interleaveSID + `","eventSeq":0,"events":[]}`)}},
			{
				Method: zcode.MethodSessionSend,
				Times:  1,
				Reply: &fake.Reply{Result: json.RawMessage(`{"accepted":true,"sessionId":"` +
					interleaveSID + `","stateRevision":1}`)},
			},
			{Method: zcode.MethodSessionStop, Reply: &fake.Reply{Result: json.RawMessage(`{}`)}},
			{
				Method: zcode.MethodSessionSend,
				Reply: &fake.Reply{Result: json.RawMessage(`{"accepted":true,"sessionId":"` +
					interleaveSID + `","stateRevision":2}`)},
				SideEffects: []fake.SideEffect{
					{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: json.RawMessage(
						`{"patch":{"status":"running"},"reason":"prompt_started","revision":1,` +
							`"scope":"session","sessionId":"` + interleaveSID + `","type":"state.updated"}`)}},
					{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(
						`{"type":"turn.completed","seq":9,"turnId":"turn_resume_stall",` +
							`"deliveryKind":"desktop-continuous"}`)}},
				},
			},
			{Method: zcode.MethodSessionMessages, Reply: &fake.Reply{Result: json.RawMessage(
				`{"messages":[{"info":{"role":"assistant"},"parts":[{"type":"text","text":"ok"}]}],` +
					`"sessionId":"` + interleaveSID + `"}`)}},
		},
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeMethodNotFound,
			Message: failureMethodNotFound,
		}},
		EOFDrainMs: 1,
	}
}

// TestResumedSessionStallSeam proves the stall seam reaches a
// session/resume machine, not only session/new's: the first resumed prompt
// surfaces the coded -32603 naming the stall with exactly one session/stop,
// and a second prompt completes normally (the slot released, the watchdog
// joined).
func TestResumedSessionStallSeam(t *testing.T) {
	f, closeLegs := newInterleaveFront(t, resumedStallScenario(),
		WithTurnStallTimeout(resumedStallBudget, resumedStallToolBudget))

	ctx, cancel := context.WithTimeout(context.Background(), resumedStallWindow)
	defer cancel()

	cwd := t.TempDir()

	sid, err := f.ResumeSession(ctx, interleaveSID, cwd)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}

	_, err = f.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: "stall"}, pickerNoopSink())
	if err == nil {
		t.Fatal("a stalled resumed turn must surface a coded error, got a success answer")
	}

	var seamErr *acpport.Error
	if !errors.As(err, &seamErr) || seamErr.Code != acpport.CodeInternal {
		t.Fatalf("stalled resumed prompt error = %v, want a coded %d internal error", err, acpport.CodeInternal)
	}

	if !strings.Contains(seamErr.Message, "turn stalled") {
		t.Fatalf("stalled resumed prompt message = %q, want it to name the stall", seamErr.Message)
	}

	resp, err := f.Prompt(ctx, acpport.PromptParams{SessionID: sid, Text: "after stall"}, pickerNoopSink())
	if err != nil {
		t.Fatalf("second resumed prompt: %v", err)
	}

	if resp != acpport.StopEndTurn {
		t.Fatalf("second resumed prompt stop = %q, want %q", resp, acpport.StopEndTurn)
	}

	tr := closeLegs()

	if sends := resumedTranscriptMethodCount(tr, zcode.MethodSessionSend); sends != 2 {
		t.Fatalf("session/send frames on the wire = %d, want exactly 2", sends)
	}

	if stops := resumedTranscriptMethodCount(tr, zcode.MethodSessionStop); stops != 1 {
		t.Fatalf("session/stop frames on the wire = %d, want exactly 1 (the watchdog stop)", stops)
	}
}

// resumedTranscriptMethodCount counts one method's host→fake frames in a
// drained transcript.
func resumedTranscriptMethodCount(tr fake.Transcript, method string) int {
	count := 0

	for _, line := range tr.Lines {
		if line.Dir != fake.DirIn {
			continue
		}

		var env struct {
			Method string `json:"method"`
		}
		if json.Unmarshal([]byte(line.JSON), &env) == nil && env.Method == method {
			count++
		}
	}

	return count
}
