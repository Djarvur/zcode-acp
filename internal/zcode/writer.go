package zcode

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

// Writer lane depths: bounded queues so a stalled reader applies
// backpressure instead of unbounded memory (RESEARCH Security Domain:
// unbounded memory via huge lines or stalled readers is a DoS vector).
const (
	writerQueueDepth  = 64
	writerBypassDepth = 16
)

// errWriterStopped reports enqueue attempts after the writer stopped.
var errWriterStopped = errors.New("zcode: writer stopped")

// writer owns every outbound byte on the stream: ONE goroutine serializes
// all writes (RESEARCH Pitfall 7 — interleaved/goroutine-per-message writes
// reorder frames and corrupt the child's stdin), fed by two lanes per the
// dialect §1 scheduling facts: the ordered queue for ordinary calls and a
// bypass lane that jumps the queue for session/stop and reverse-request
// replies (answering a reverse request from the queue would deadlock the
// pairing — dialect §1).
type writer struct {
	w        io.Writer
	queue    chan *Envelope
	bypass   chan *Envelope
	done     chan struct{}
	stopWG   sync.WaitGroup
	errMu    sync.Mutex
	firstErr error
}

// newWriter starts the single writer goroutine over w.
func newWriter(w io.Writer) *writer {
	cw := &writer{
		w:      w,
		queue:  make(chan *Envelope, writerQueueDepth),
		bypass: make(chan *Envelope, writerBypassDepth),
		done:   make(chan struct{}),
	}
	cw.stopWG.Add(1)

	go cw.run()

	return cw
}

// run is the two-lane writer loop (RESEARCH Pattern 2, final shape): the
// bypass channel is checked non-blockingly before each queued write, and
// the loop re-checks the queue after a bypass frame so the bypass lane can
// never starve ordinary traffic (dialect §1: ordinary requests execute
// serially on one queue; stop/cancel and reverse-request replies bypass).
func (w *writer) run() {
	defer w.stopWG.Done()

	for {
		select {
		case env := <-w.bypass:
			w.writeFrame(env)

			continue
		case <-w.done:
			return
		default:
		}

		select {
		case env := <-w.bypass:
			w.writeFrame(env)
		case env := <-w.queue:
			w.writeFrame(env)
		case <-w.done:
			return
		}
	}
}

// writeFrame marshals env to one newline-terminated line and writes it.
func (w *writer) writeFrame(env *Envelope) {
	line, err := Encode(env)
	if err != nil {
		w.recordErr(err)

		return
	}

	if _, err := w.w.Write(line); err != nil {
		w.recordErr(fmt.Errorf("zcode: write frame: %w", err))
	}
}

// recordErr keeps the first write error; the writer keeps draining so one
// failed write does not wedge callers — the read-loop drain surfaces
// firstErr to in-flight calls as the stream's actual cause (WR-01) and
// Close ends the loop.
func (w *writer) recordErr(err error) {
	w.errMu.Lock()
	defer w.errMu.Unlock()

	if w.firstErr == nil {
		w.firstErr = err
	}
}

// firstError returns the first write error the writer recorded, or nil —
// the conn's read-loop drain wraps it as the cause of the in-flight-call
// failure so callers see the stream's actual last error (review WR-01:
// firstErr was write-only).
func (w *writer) firstError() error {
	w.errMu.Lock()
	defer w.errMu.Unlock()

	return w.firstErr
}

// enqueue sends env on the ordered queue lane (ordinary calls and
// notifications).
func (w *writer) enqueue(env *Envelope) error {
	return w.send(w.queue, env)
}

// enqueueBypass sends env on the bypass lane (session/stop and
// reverse-request replies — dialect §1 bypass set).
func (w *writer) enqueueBypass(env *Envelope) error {
	return w.send(w.bypass, env)
}

// send delivers env to a lane, failing fast once the writer stopped. The
// done check runs first so a stopped writer never races a lane send.
func (w *writer) send(ch chan *Envelope, env *Envelope) error {
	select {
	case <-w.done:
		return errWriterStopped
	default:
	}

	select {
	case <-w.done:
		return errWriterStopped
	case ch <- env:
		return nil
	}
}

// stop ends the writer goroutine and waits for it to exit. Callers must
// close the underlying stream first so a blocked Write can unblock
// (Conn.Close orders it that way).
func (w *writer) stop() {
	close(w.done)
	w.stopWG.Wait()
}
