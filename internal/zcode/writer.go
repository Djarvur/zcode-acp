package zcode

import (
	"io"
	"sync"
)

// writer owns every outbound byte on the stream: ONE goroutine serializes
// all writes (RESEARCH Pitfall 7 — interleaved/goroutine-per-message writes
// reorder frames and corrupt the child's stdin), fed by two lanes per the
// dialect §1 scheduling facts: the ordered queue for ordinary calls and a
// bypass lane that jumps the queue for session/stop and reverse-request
// replies (answering a reverse request from the queue would deadlock the
// pairing — dialect §1).
type writer struct {
	w      io.Writer
	done   chan struct{}
	stopWG sync.WaitGroup
}

// newWriter starts the single writer goroutine over w.
func newWriter(w io.Writer) *writer {
	cw := &writer{
		w:    w,
		done: make(chan struct{}),
	}
	cw.stopWG.Add(1)

	go cw.run()

	return cw
}

// run is the writer goroutine. Skeleton stage: the two-lane loop from
// RESEARCH Pattern 2 lands with the codec leaf commit (plan Task 2); until
// then the writer only services shutdown so the tracer RED fails on its own
// assertion instead of hanging the pipe.
func (w *writer) run() {
	defer w.stopWG.Done()

	<-w.done
}

// stop ends the writer goroutine and waits for it to exit.
func (w *writer) stop() {
	close(w.done)
	w.stopWG.Wait()
}
