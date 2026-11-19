// Package fake provides a scriptable in-process ZCode app-server (D-01):
// Serve speaks the same newline-delimited envelope grammar as the codec over
// any io.ReadWriteCloser (io.Pipe in-process, exec pipes for the standalone
// binary), replays a declarative Scenario, and records a direction-tagged
// Transcript that golden tests replay byte-deterministically.
package fake

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Read-buffer sizing mirrors the codec read loop (ARCHITECTURE Pattern 2,
// T-3-02): raised but capped — large scripted payloads decode, while a
// pathological line ends the scan instead of growing memory unboundedly.
const (
	serveBufInitial = 1 << 20
	serveBufMax     = 16 << 20
)

// errReverseMismatch marks a reverse answer that deviates from the
// scenario's declared expect_answer (fail-loud scenario assertion).
var errReverseMismatch = errors.New("fake: reverse answer mismatch")

// Dir tags which way a transcript frame travelled relative to the fake.
type Dir string

const (
	// DirIn marks a host→fake frame: requests and reverse-request replies.
	DirIn Dir = "in"

	// DirOut marks a fake→host frame: replies, notifications, and the
	// fake's own reverse requests.
	DirOut Dir = "out"

	// DirNote marks a non-frame transcript annotation — e.g. a reverse
	// request recorded honestly as unanswered at its timeout (record-
	// honesty discipline: never normalize a missing frame away).
	DirNote Dir = "note"
)

// Line is one transcript entry: the direction tag plus the raw wire line,
// without its newline terminator (both directions record it identically).
type Line struct {
	Dir  Dir
	JSON string
}

// Transcript is the recorded wire conversation in true wire order — the
// protocol-spec-made-executable fixture (D-01). Fixed scenario ids keep
// replays byte-deterministic.
type Transcript struct {
	Lines []Line
}

// server is one Serve run: the scan loop, the pending reverse-request table,
// and the direction-tagged transcript. Stream writes serialize behind trMu —
// one writer per sink (RESEARCH goroutine-topology gotcha) — and every frame
// is recorded at the boundary-crossing moment, so the transcript order is
// the wire order.
type server struct {
	rw   io.ReadWriteCloser
	sc   *Scenario
	wg   sync.WaitGroup
	trMu sync.Mutex
	tr   Transcript

	revMu    sync.Mutex
	reverse  map[string]chan zcode.Envelope
	nextRev  atomic.Int64
	closed   chan struct{}
	ruleMu   sync.Mutex
	errMu    sync.Mutex
	firstErr error
}

// Serve runs the scripted app-server over rw against sc until stdin EOF,
// then drains sc.EOFDrainMs (dialect §1 EOF facts) and returns the recorded
// transcript. Requests are matched against sc.on_request by method: the
// rule's side effects run in order — emit writes one notification, reverse
// writes a reverse request and blocks until the host's answer or timeout_ms
// — and the reply, echoing the request id verbatim (dialect §1), is written
// after them, so a rule's notifications are on the wire before its reply
// (the wire truth the captures record: the create reply follows the
// runtime-preferences exchange; the send ack follows the state patch). An
// expect_answer, when declared, is checked semantically — decoded JSON
// compare, so key order never matters. A reverse request that times out or
// is orphaned by EOF is recorded honestly as a transcript note — never
// normalized away.
//
// Serve does not close rw; the caller owns the stream lifecycle.
func Serve(rw io.ReadWriteCloser, sc *Scenario) (Transcript, error) {
	s := &server{
		rw:      rw,
		sc:      sc,
		reverse: make(map[string]chan zcode.Envelope),
		closed:  make(chan struct{}),
	}

	return s.run()
}

// run scans host lines, dispatches requests to handler goroutines, and
// resolves reverse-request answers; on scan end it drains the EOF window,
// joins in-flight handlers, and returns the transcript.
func (s *server) run() (Transcript, error) {
	sc := bufio.NewScanner(s.rw)
	sc.Buffer(make([]byte, 0, serveBufInitial), serveBufMax)

	for sc.Scan() {
		line := sc.Bytes()

		env, err := zcode.Decode(line)
		if err != nil {
			s.record(DirNote, fmt.Sprintf("undecodable host line (%d bytes): %v", len(line), err))

			continue
		}

		s.record(DirIn, string(line))
		s.dispatch(env)
	}

	close(s.closed)

	if err := sc.Err(); err != nil && !isCleanEOF(err) {
		s.setErr(fmt.Errorf("fake: scan host stream: %w", err))
	}

	if s.sc.EOFDrainMs > 0 {
		time.Sleep(time.Duration(s.sc.EOFDrainMs) * time.Millisecond)
	}

	s.wg.Wait()

	return s.snapshot()
}

// isCleanEOF reports whether a scanner error is an expected shutdown signal:
// a real EOF, the io.Pipe closed-pipe error, or a closed exec pipe — the
// host closing its end is the normal end of every scripted run (dialect §1
// client-EOF facts).
func isCleanEOF(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, os.ErrClosed)
}

// dispatch classifies one inbound envelope: a reply (id, no method) answers
// a pending reverse request; a request (id AND method) runs on its own
// handler goroutine so a blocking reverse side effect never stalls the scan
// loop; a host notification (method, no id) needs no reply (dialect §1).
func (s *server) dispatch(env zcode.Envelope) {
	switch {
	case env.ID != nil && env.Method == "":
		s.resolveReverse(env)

	case env.ID != nil && env.Method != "":
		s.wg.Add(1)

		go s.handleRequest(env)
	}
}

// handleRequest serves one host request: the rule's side effects in order,
// then the reply — echoing the request id verbatim (dialect §1).
func (s *server) handleRequest(env zcode.Envelope) {
	defer s.wg.Done()

	rule := s.ruleFor(env.Method)

	if rule == nil {
		s.replyUnknown(env.ID)

		return
	}

	s.runSideEffects(rule.SideEffects)
	s.writeReply(env.ID, rule.Reply, rule.DelayMs)
}

// ruleFor finds the scenario's request rule for a method with remaining
// match budget, or nil. A rule's times, when positive, caps its matches —
// a later on_request entry for the same method takes over once the budget
// is spent. Guarded by ruleMu: handler goroutines are concurrent (one per
// dispatched request).
func (s *server) ruleFor(method string) *RequestRule {
	s.ruleMu.Lock()
	defer s.ruleMu.Unlock()

	for i := range s.sc.OnRequest {
		r := &s.sc.OnRequest[i]
		if r.Method != method {
			continue
		}

		if r.Times > 0 && r.matched >= r.Times {
			continue
		}

		r.matched++

		return r
	}

	return nil
}

// runSideEffects executes a rule's side effects in order: emit writes one
// notification; reverse writes a reverse request and blocks for the answer.
func (s *server) runSideEffects(effects []SideEffect) {
	for i := range effects {
		switch {
		case effects[i].Emit != nil:
			s.write(&zcode.Envelope{
				Method: effects[i].Emit.Method,
				Params: effects[i].Emit.Params,
			})

		case effects[i].Reverse != nil:
			s.runReverse(effects[i].Reverse)
		}
	}
}

// runReverse issues one server→host request and blocks until the host's
// answer arrives, timeout_ms elapses, or the stream closes — whichever is
// first. An answered reverse whose payload mismatches the declared
// expect_answer fails the run; an unanswered one is recorded honestly as a
// transcript note.
func (s *server) runReverse(r *Reverse) {
	id := &zcode.ID{Str: fmt.Sprintf("fake-%d", s.nextRev.Add(1))}

	answer := make(chan zcode.Envelope, 1)

	s.revMu.Lock()
	s.reverse[id.Key()] = answer
	s.revMu.Unlock()

	s.write(&zcode.Envelope{ID: id, Method: r.Method, Params: r.Params})

	timer := time.NewTimer(time.Duration(r.TimeoutMs) * time.Millisecond)
	defer timer.Stop()

	select {
	case env := <-answer:
		s.forgetReverse(id)
		s.checkExpectation(r, env)

	case <-timer.C:
		s.forgetReverse(id)
		s.record(DirNote, fmt.Sprintf("reverse %s %s unanswered at %dms timeout", r.Method, id.Key(), r.TimeoutMs))

	case <-s.closed:
		s.forgetReverse(id)
		s.drainOrphan(r, id, answer)
	}
}

// drainOrphan records an unanswered reverse at stream close, unless the
// answer landed in the buffer as the scan loop ended (select readiness is
// random when both channels are ready — the buffer is checked first).
func (s *server) drainOrphan(r *Reverse, id *zcode.ID, answer chan zcode.Envelope) {
	select {
	case env := <-answer:
		s.checkExpectation(r, env)

	default:
		s.record(DirNote, fmt.Sprintf("reverse %s %s unanswered (host stream closed)", r.Method, id.Key()))
	}
}

// resolveReverse delivers a host reply to the pending reverse request with
// the same canonical id key. Unsolicited replies are dropped.
func (s *server) resolveReverse(env zcode.Envelope) {
	s.revMu.Lock()
	answer, ok := s.reverse[env.ID.Key()]
	s.revMu.Unlock()

	if !ok {
		return
	}

	select {
	case answer <- env:
	default:
		s.record(DirNote, fmt.Sprintf("duplicate reply for reverse %s dropped", env.ID.Key()))
	}
}

// forgetReverse deregisters a pending reverse request.
func (s *server) forgetReverse(id *zcode.ID) {
	s.revMu.Lock()
	defer s.revMu.Unlock()

	delete(s.reverse, id.Key())
}

// checkExpectation validates a reverse answer against the declared
// expect_answer; a mismatch fails the run — the scenario assertion is the
// point of the fixture.
func (s *server) checkExpectation(r *Reverse, env zcode.Envelope) {
	if r.ExpectAnswer == nil {
		return
	}

	if err := matchExpectAnswer(r.ExpectAnswer, env); err != nil {
		s.setErr(err)
		s.record(DirNote, err.Error())
	}
}

// matchExpectAnswer compares a declared expectation against the observed
// answer envelope semantically — decoded JSON deep compare, so key order
// never matters (the codec marshals maps sorted; captures author payloads
// in wire order).
func matchExpectAnswer(exp *Reply, env zcode.Envelope) error {
	if exp.Error != nil {
		if env.Error == nil || env.Error.Code != exp.Error.Code || env.Error.Message != exp.Error.Message {
			return fmt.Errorf("%w: want error %d %q, got %v",
				errReverseMismatch, exp.Error.Code, exp.Error.Message, env.Error)
		}

		return nil
	}

	var want, got any

	if err := json.Unmarshal(exp.Result, &want); err != nil {
		return fmt.Errorf("fake: expect_answer.result undecodable: %w", err)
	}

	if err := json.Unmarshal(env.Result, &got); err != nil {
		return fmt.Errorf("fake: reverse answer undecodable: %w", err)
	}

	if !reflect.DeepEqual(want, got) {
		return fmt.Errorf("%w: want %s, got %s", errReverseMismatch, exp.Result, env.Result)
	}

	return nil
}

// replyUnknown answers an unmatched method: the scenario's unknown_method
// error when declared, else the dialect §6 MethodNotFound default (the same
// -32601 the conn's router answers with — probe precedent, dialect §3).
func (s *server) replyUnknown(id *zcode.ID) {
	werr := &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: "MethodNotFound"}

	if s.sc.UnknownMethod != nil && s.sc.UnknownMethod.Error != nil {
		werr = s.sc.UnknownMethod.Error
	}

	s.write(&zcode.Envelope{ID: id, Error: werr})
}

// writeReply writes the rule's reply after the configured delay, echoing
// the request id verbatim (dialect §1). Exactly one of result or error was
// validated at load.
func (s *server) writeReply(id *zcode.ID, r *Reply, delayMs int) {
	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	env := &zcode.Envelope{ID: id}

	if r.Error != nil {
		env.Error = r.Error
	} else {
		env.Result = r.Result
	}

	s.write(env)
}

// write encodes one fake→host envelope and writes it to the stream,
// recording it in the transcript only after the bytes hit the wire. The
// trMu critical section covers write and record together, so DirOut order
// equals wire order. Inbound and note lines go through record. A write that
// fails because the host stream closed at EOF is the drain-window shutdown
// (dialect §1: a request issued just before stdin EOF is never answered) —
// the frame is recorded honestly as an unanswered note, never a run error.
func (s *server) write(env *zcode.Envelope) {
	line, err := zcode.Encode(env)
	if err != nil {
		s.setErr(fmt.Errorf("fake: encode %s frame: %w", env.Method, err))

		return
	}

	s.trMu.Lock()
	defer s.trMu.Unlock()

	if _, err := s.rw.Write(line); err != nil {
		if isCleanEOF(err) {
			s.appendLine(DirNote, unansweredNote(env))

			return
		}

		s.setErr(fmt.Errorf("fake: write %s frame: %w", env.Method, err))

		return
	}

	s.appendLine(DirOut, string(line[:len(line)-1]))
}

// appendLine appends one transcript line; the caller holds trMu (record is
// the locking entry point — write's critical section appends directly, since
// a mutex is not reentrant).
func (s *server) appendLine(dir Dir, line string) {
	s.tr.Lines = append(s.tr.Lines, Line{Dir: dir, JSON: line})
}

// unansweredNote renders the honest transcript note for a frame whose write
// failed because the host stream closed: labeled by method, or by the id key
// for replies. Record-honesty: never normalized away, never a silent drop.
func unansweredNote(env *zcode.Envelope) string {
	label := env.Method
	if label == "" && env.ID != nil {
		label = "reply " + env.ID.Key()
	}

	return label + " unanswered (host stream closed)"
}

// record appends one transcript line (any direction) under trMu.
func (s *server) record(dir Dir, line string) {
	s.trMu.Lock()
	defer s.trMu.Unlock()

	s.appendLine(dir, line)
}

// setErr keeps the first run error.
func (s *server) setErr(err error) {
	s.errMu.Lock()
	defer s.errMu.Unlock()

	if s.firstErr == nil {
		s.firstErr = err
	}
}

// snapshot returns the transcript and the first recorded error.
func (s *server) snapshot() (Transcript, error) {
	s.trMu.Lock()
	tr := s.tr
	s.trMu.Unlock()

	s.errMu.Lock()
	defer s.errMu.Unlock()

	return tr, s.firstErr
}
