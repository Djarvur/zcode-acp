package zcode

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
)

// Read-loop line buffer sizing (T-3-02 mitigation): raised but capped per
// ARCHITECTURE Pattern 2 — large tool-output lines must decode, while a
// pathological line ends the scan instead of growing memory unboundedly.
const (
	readBufInitial = 1 << 20
	readBufMax     = 16 << 20
)

// Conn drives one app-server JSONL stream over rw: a read loop that demuxes
// reply / reverse-request / notification frames, a single writer goroutine
// with a bypass lane, and a pending-call table keyed by type-tagged id
// (ARCHITECTURE Pattern 2). Conn is safe for concurrent use.
type Conn struct {
	rw        io.ReadWriteCloser
	out       *writer
	pending   *pendingTable
	nextID    atomic.Int64
	readDone  chan struct{}
	closeOnce sync.Once

	// serveRequest serves one server→host request on a fresh goroutine,
	// deriving its context from the conn's base context. The context itself
	// is closure-captured in NewConn, never stored on the struct.
	serveRequest func(env Envelope)

	// cancelBase cancels the conn's base request context (NewConn-owned).
	cancelBase context.CancelFunc

	// OnNotification, when non-nil, receives every notification/event
	// envelope (method, no id) in wire order. Deliveries happen on the read
	// goroutine: the hook must not block. Assign hooks immediately after
	// NewConn, before any frame flows — assigning while traffic is in
	// flight is a data race.
	OnNotification func(env Envelope)

	// OnRequest, when non-nil, intercepts a server→host request before the
	// built-in router; returning handled=false falls through to the
	// router's defaults (D-05 minimum: the runtime-preferences answer and
	// -32601 for everything else). Invoked on a per-request goroutine.
	OnRequest func(ctx context.Context, env Envelope) (result any, werr *WireError, handled bool)
}

// NewConn returns a Conn over rw and starts its writer and read-loop
// goroutines. rw.Close must unblock a blocked Read on rw — Close relies on
// it to end the read loop (io.Pipe, os/exec pipes, and net.Conn all do).
func NewConn(rw io.ReadWriteCloser) *Conn {
	baseCtx, cancel := context.WithCancel(context.Background())

	c := &Conn{
		rw:         rw,
		out:        newWriter(rw),
		pending:    newPendingTable(),
		readDone:   make(chan struct{}),
		cancelBase: cancel,
	}

	c.serveRequest = func(env Envelope) {
		ctx, cancelReq := context.WithCancel(baseCtx)
		defer cancelReq()

		c.handleRequest(ctx, env)
	}

	go c.readLoop()

	return c
}

// Close stops the stream and the writer, in that order: the peer sees EOF
// and runs its drain window (dialect §1 EOF facts), and closing rw first
// unblocks any writeFrame parked on the stream so the writer goroutine can
// exit. Request goroutines are cancelled via the base context; the read
// loop is joined. Idempotent.
func (c *Conn) Close() error {
	var closeErr error

	c.closeOnce.Do(func() {
		c.cancelBase()

		if err := c.rw.Close(); err != nil {
			closeErr = fmt.Errorf("zcode: close conn stream: %w", err)
		}

		c.out.stop()
		<-c.readDone
	})

	return closeErr
}

// readLoop scans wire lines until EOF, demuxing each decoded envelope
// (ARCHITECTURE Pattern 2). Undecodable lines are logged structurally —
// method names, id keys and byte sizes only, never params or results, which
// carry prompt content — and skipped; the stream continues.
func (c *Conn) readLoop() {
	defer close(c.readDone)

	sc := bufio.NewScanner(c.rw)
	sc.Buffer(make([]byte, 0, readBufInitial), readBufMax)

	for sc.Scan() {
		line := sc.Bytes()

		env, err := Decode(line)
		if err != nil {
			slog.Warn("zcode: skipping undecodable line", "err", err, "bytes", len(line))

			continue
		}

		c.demux(env)
	}

	if err := sc.Err(); err != nil {
		slog.Warn("zcode: read loop ended", "err", err)
	}
}

// demux classifies one inbound envelope per the RESEARCH Pattern 1 rule
// (the WR-02 structural fix): a method-bearing frame NEVER enters the
// pending table — only result/error reply frames resolve calls.
func (c *Conn) demux(env Envelope) {
	switch {
	case isReply(env):
		c.pending.resolve(env)

	case env.ID != nil && env.Method != "":
		go c.serveRequest(env)

	case env.Method != "":
		c.deliverNotification(env)

	default:
		slog.Warn("zcode: dropping non-conforming envelope (no id, no method)")
	}
}

// isReply reports whether env is a reply envelope: id present, no method,
// and a result or error payload (RESEARCH Pattern 1 classification — the
// ARCHITECTURE sketch's looser first case predates the WR-02 fix and must
// also exclude method-bearing frames).
func isReply(env Envelope) bool {
	return env.ID != nil && env.Method == "" && (env.Result != nil || env.Error != nil)
}

// deliverNotification hands a notification to the OnNotification hook on the
// read goroutine — wire order preserved; the hook must not block.
func (c *Conn) deliverNotification(env Envelope) {
	if c.OnNotification != nil {
		c.OnNotification(env)
	}
}

// handleRequest serves one server→host request on the caller's goroutine —
// for the read loop that is always a fresh goroutine, never inline
// (ARCHITECTURE Anti-Pattern 3: the app-server blocks mid-turn on answers;
// a blocked read loop deadlocks everything). The request context derives
// from the conn's base context, cancelled on Close.
func (c *Conn) handleRequest(ctx context.Context, env Envelope) {
	if c.OnRequest != nil {
		result, werr, handled := c.OnRequest(ctx, env)
		if handled {
			c.replyRequest(env.ID, result, werr)

			return
		}
	}

	c.routeRequest(env)
}

// routeRequest is the built-in reverse-request router minimum (RESEARCH
// Pattern 3, D-05): the runtime-preferences static answer and -32601 for
// every other method (probe precedent — dialect §3). Per-method deadlines
// and their expiry recording are the plan-03-02 hardening seam.
func (c *Conn) routeRequest(env Envelope) {
	switch env.Method {
	case MethodRequestRuntimePreferences:
		c.replyRequest(env.ID, map[string]any{
			"nativeSearchEnhancementsEnabled": false,
			"memoryEnabled":                   false,
		}, nil)

	default:
		c.replyRequest(env.ID, nil, &WireError{
			Code:    CodeMethodNotFound,
			Message: "unsupported host request: " + env.Method,
		})
	}
}

// replyRequest delivers a reverse-request answer on the bypass lane
// (dialect §1: answering from the queue would deadlock the pairing).
func (c *Conn) replyRequest(id *ID, result any, werr *WireError) {
	var err error

	if werr != nil {
		err = c.ReplyError(id, werr)
	} else {
		err = c.Reply(id, result)
	}

	if err != nil {
		slog.Warn("zcode: reverse-request reply failed", "id", id.Key(), "err", err)
	}
}
