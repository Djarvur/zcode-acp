package zcode

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// Conn drives one app-server JSONL stream over rw: a read loop that demuxes
// reply / reverse-request / notification frames, a single writer goroutine
// with a bypass lane, and a pending-call table keyed by type-tagged id
// (ARCHITECTURE Pattern 2). Conn is safe for concurrent use.
type Conn struct {
	rw        io.ReadWriteCloser
	out       *writer
	closeOnce sync.Once

	// OnNotification, when non-nil, receives every notification/event
	// envelope (method, no id) in wire order. Deliveries happen on the read
	// goroutine: the hook must not block.
	OnNotification func(env Envelope)

	// OnRequest, when non-nil, intercepts a server→host request before the
	// built-in router; returning handled=false falls through to the
	// router's defaults (D-05 minimum: the runtime-preferences answer and
	// -32601 for everything else).
	OnRequest func(ctx context.Context, env Envelope) (result any, werr *WireError, handled bool)
}

// NewConn returns a Conn over rw and starts its writer goroutine. The read
// loop starts with the conn stage (plan Task 3).
func NewConn(rw io.ReadWriteCloser) *Conn {
	c := &Conn{
		rw:  rw,
		out: newWriter(rw),
	}

	return c
}

// Close stops the writer and closes rw — the peer sees EOF and runs its
// drain window before exiting (dialect §1 EOF facts). Idempotent.
func (c *Conn) Close() error {
	var closeErr error

	c.closeOnce.Do(func() {
		c.out.stop()

		if err := c.rw.Close(); err != nil {
			closeErr = fmt.Errorf("zcode: close conn stream: %w", err)
		}
	})

	return closeErr
}
