package zcode

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Read-loop line buffer sizing (T-3-02 mitigation): raised but capped per
// ARCHITECTURE Pattern 2 — large tool-output lines must decode, while a
// pathological line ends the scan instead of growing memory unboundedly.
const (
	readBufInitial = 1 << 20
	readBufMax     = 16 << 20
)

// Router policy surface (D-05): the reverse-request router answers the
// session-establishment obligations inside per-method deadlines and records
// what it did — answered under its deadline, or a tracked-no-answer method's
// deadline expiring with no reply written.

// RouterDefaultDeadline is the D-05 per-method answer deadline: the
// session/requestRuntimePreferences obligation is answered within 15 s
// (live precedent: unanswered creates fail -32603 — dialect §3).
const RouterDefaultDeadline = 15 * time.Second

// RouterEventKind classifies one router policy observation.
type RouterEventKind string

const (
	// RouterAnswered marks a policy-table method the router answered inside
	// its deadline.
	RouterAnswered RouterEventKind = "answered"

	// RouterDeadlineExpired marks a tracked-no-answer method whose deadline
	// elapsed without any reply written (record-honesty: the expiry is
	// observed, never swallowed — D-05).
	RouterDeadlineExpired RouterEventKind = "deadline-expired"
)

// RouterEvent is one reverse-request router policy observation.
type RouterEvent struct {
	Kind     RouterEventKind // answered, or deadline-expired with no reply
	Method   string          // the server→host request method
	IDKey    string          // canonical type-tagged id key of the request
	Deadline time.Duration   // the per-method deadline that governed it
	Err      error           // expiry cause (context.DeadlineExceeded); nil when answered
}

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

	// trackRequest registers a tracked-no-answer request against the conn's
	// base context (closure-captured in NewConn): the tracker must outlive
	// the per-request handler goroutine — serveRequest cancels the request
	// context when its handler returns, which would cut the tracker's
	// deadline short before it could elapse.
	trackRequest func(env Envelope)

	// cancelBase cancels the conn's base request context (NewConn-owned).
	cancelBase context.CancelFunc

	// obsMu guards the ordered observer list. Deliveries iterate the list
	// under the lock: registration happens before traffic per the hook
	// contract, so the lock never contends with an in-flight delivery on
	// the real path.
	obsMu     sync.Mutex
	observers []*observerEntry

	// OnRequest, when non-nil, intercepts a server→host request before the
	// built-in router; returning handled=false falls through to the
	// router's defaults (D-05 minimum: the runtime-preferences answer and
	// -32601 for everything else). Invoked on a per-request goroutine.
	OnRequest func(ctx context.Context, env Envelope) (result any, werr *WireError, handled bool)

	// RouterDeadline, when non-nil, overrides the per-method policy deadline
	// for reverse-request routing (default RouterDefaultDeadline). Tests
	// inject millisecond deadlines; assign before any frame flows, like the
	// hooks.
	RouterDeadline func(method string) time.Duration

	// routerObservers is the ordered router-event fan-out (WR-01): every
	// registered machine keeps its own policy-event path — a later machine's
	// construction no longer steals an earlier machine's degraded-resolution
	// hook. Delivery is registration order under routerObsMu; mid-traffic
	// registration is memory-safe (new observers see only subsequent events —
	// the contract is delivery ORDERING, not replay).
	routerObservers []*routerObserverEntry

	// routerObsMu guards routerObservers registration and delivery.
	routerObsMu sync.Mutex
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

	c.trackRequest = func(env Envelope) {
		c.trackUnanswered(baseCtx, env)
	}

	go c.readLoop()

	return c
}

// AddRouterObserver registers fn on the ordered router-event fan-out and
// returns its remove func. Registration is memory-safe mid-traffic under
// routerObsMu; the delivery contract is ORDERING — a new observer sees
// only subsequent events (no replay of earlier policy observations), and
// every registered observer receives each event in registration order
// (IN-01: the mid-traffic-registration sentence in the old single-slot doc
// overstated the contract; the mutex made registration data-race-free,
// which is the weaker claim this doc now states precisely).
func (c *Conn) AddRouterObserver(fn func(ev RouterEvent)) (remove func()) {
	entry := &routerObserverEntry{fn: fn}

	c.routerObsMu.Lock()
	c.routerObservers = append(c.routerObservers, entry)
	c.routerObsMu.Unlock()

	return func() {
		c.routerObsMu.Lock()
		defer c.routerObsMu.Unlock()

		for i, obs := range c.routerObservers {
			if obs == entry {
				c.routerObservers = slices.Delete(c.routerObservers, i, i+1)

				return
			}
		}
	}
}

// Close stops the stream and the writer, in that order: the peer sees EOF
// and runs its drain window (dialect §1 EOF facts), and closing rw first
// unblocks any writeFrame parked on the stream so the writer goroutine can
// exit. Request goroutines are cancelled via the base context; pending
// calls are drained with an error matching ErrConnClosed (WR-01 — the base
// context never reaches them); the read loop is joined. Idempotent.
func (c *Conn) Close() error {
	var closeErr error

	c.closeOnce.Do(func() {
		c.cancelBase()

		if err := c.rw.Close(); err != nil {
			closeErr = fmt.Errorf("zcode: close conn stream: %w", err)
		}

		c.out.stop()
		<-c.readDone
		c.drainPending() // sweep entries registered after the read loop's drain; idempotent
	})

	return closeErr
}

// AddObserver registers fn as a notification observer and returns its
// remove func. Every notification/event envelope (method, no id) is
// delivered to ALL registered observers — synchronously, in registration
// order, on the read goroutine (wire order preserved). The hook contract
// from the single slot carries over: deliveries run on the read goroutine,
// so an observer must not block. Mid-traffic registration and removal are
// memory-safe under the fan-out mutex; the real contract is delivery
// ORDERING — a newly registered observer sees only subsequent envelopes,
// never a replay of earlier ones. An empty observer list is a silent
// drop, like the nil single slot this fan-out replaces (RESEARCH
// Pattern 4 — a second machine could silently steal that slot, leaving
// the first session's turns hanging, PITFALL 3).
func (c *Conn) AddObserver(fn func(Envelope)) (remove func()) {
	entry := &observerEntry{fn: fn}

	c.obsMu.Lock()
	c.observers = append(c.observers, entry)
	c.obsMu.Unlock()

	return func() {
		c.obsMu.Lock()
		defer c.obsMu.Unlock()

		for i, obs := range c.observers {
			if obs == entry {
				c.observers = slices.Delete(c.observers, i, i+1)

				return
			}
		}
	}
}

// observerEntry pins one registered observer's identity: the remove func
// matches by entry pointer, so double-removes and index shifts stay safe
// (function values are not comparable in Go).
type observerEntry struct {
	fn func(Envelope)
}

// ObserverCounts returns the live notification and router observer counts
// (diagnostics and the teardown-baseline test seam).
func (c *Conn) ObserverCounts() (notification, router int) {
	return len(c.observers), len(c.routerObservers)
}

// routerObserverEntry is the router-event twin of observerEntry — the
// carrier type differs (RouterEvent policy observations, not envelopes).
type routerObserverEntry struct {
	fn func(ev RouterEvent)
}

// readLoop scans wire lines until EOF, demuxing each decoded envelope
// (ARCHITECTURE Pattern 2). Undecodable lines are logged structurally —
// method names, id keys and byte sizes only, never params or results, which
// carry prompt content — and skipped; the stream continues. On exit the
// pending table is drained: no reply can ever arrive on a dead stream, so
// in-flight calls fail promptly instead of hanging until their own ctx
// expires (WR-01).
func (c *Conn) readLoop() {
	defer func() {
		c.drainPending()
		close(c.readDone)
	}()

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

// drainPending fails every in-flight call — the stream died or the conn is
// closing, so no reply can ever arrive. The writer's first recorded write
// error is the cause when one exists (the stream's last observable
// failure), wrapped by ErrConnClosed; the sentinel alone otherwise. Called
// on read-loop exit and again at the end of Close — idempotent (a drained
// table is empty), and exact-once per entry: drainAll and resolve both
// delete under the table lock, so a call can never receive both a drain
// failure and a wire reply (the table's once semantics, WR-01).
func (c *Conn) drainPending() {
	err := c.out.firstError()
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrConnClosed, err)
	} else {
		err = ErrConnClosed
	}

	c.pending.drainAll(err)
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

// deliverNotification hands a notification envelope to every registered
// observer — synchronously, in registration order, on the read goroutine
// (wire order preserved); observers must not block. Zero observers is a
// silent drop.
func (c *Conn) deliverNotification(env Envelope) {
	c.obsMu.Lock()
	defer c.obsMu.Unlock()

	for _, obs := range c.observers {
		obs.fn(env)
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

	c.routeRequest(ctx, env)
}

// routeRequest is the built-in reverse-request router (RESEARCH Pattern 3,
// D-05): the runtime-preferences static answer inside the method's deadline,
// the mid-turn credential-cipher request tracked-but-never-answered (its
// Phase 6 keyring scope), and -32601 naming the method for everything else
// (probe precedent — dialect §3/§6: numeric codes are the stable contract,
// message text is not).
func (c *Conn) routeRequest(ctx context.Context, env Envelope) {
	switch env.Method {
	case MethodRequestRuntimePreferences:
		c.answerRuntimePreferences(ctx, env)

	case MethodRequestCredentialCipher:
		c.trackRequest(env)

	default:
		c.replyRequest(env.ID, nil, &WireError{
			Code:    CodeMethodNotFound,
			Message: "unsupported host request: " + env.Method,
		})
	}
}

// answerRuntimePreferences replies with the static runtime-preference
// defaults (EXP-WORKSPACEKEY-01.jsonl:9 shape, proven live) inside the
// method's D-05 deadline; the governing deadline is recorded on the router
// event so the policy is observable. Phase 5 replaces the static defaults
// with real preference computation under the same bound.
func (c *Conn) answerRuntimePreferences(ctx context.Context, env Envelope) {
	deadline := c.deadlineFor(env.Method)

	ctx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	select {
	case <-ctx.Done():
		// The parent died (conn closing) or the deadline elapsed first: a
		// genuine deadline expiry is recorded honestly; conn shutdown owes
		// nothing on the wire.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			c.recordDeadlineExpired(env, deadline, ctx.Err())
		}

		return
	default:
	}

	c.replyRequest(env.ID, map[string]any{
		"nativeSearchEnhancementsEnabled": false,
		"memoryEnabled":                   false,
	}, nil)

	c.recordRouterEvent(RouterEvent{
		Kind:     RouterAnswered,
		Method:   env.Method,
		IDKey:    env.ID.Key(),
		Deadline: deadline,
	})
}

// trackUnanswered registers the known-but-NOT-answered credential-cipher
// request (D-05: answering it needs the Phase 6 keyring vault) against the
// conn's base context and records its deadline expiry when it fires — never
// a reply, never a fabricated -32601: replying at all would claim an answer
// the wrapper cannot give, and the server holds the turn on it. The tracker
// goroutine is the registered entry; it dies with the conn (base-context
// cancellation) without recording — nothing is owed on a closed stream.
func (c *Conn) trackUnanswered(ctx context.Context, env Envelope) {
	deadline := c.deadlineFor(env.Method)

	ctx, cancel := context.WithTimeout(ctx, deadline)

	go func() {
		defer cancel()

		<-ctx.Done()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			c.recordDeadlineExpired(env, deadline, ctx.Err())
		}
	}()
}

// deadlineFor resolves the per-method policy deadline: the injectable
// override wins when it returns a positive duration, else the D-05 default.
func (c *Conn) deadlineFor(method string) time.Duration {
	if c.RouterDeadline != nil {
		if d := c.RouterDeadline(method); d > 0 {
			return d
		}
	}

	return RouterDefaultDeadline
}

// recordDeadlineExpired records a tracked-no-answer expiry observably — the
// OnRouterEvent hook when installed, plus a structural slog warning to
// stderr (record-honesty: D-05 forbids swallowing the expiry). Method names,
// canonical id keys and durations only — never params.
func (c *Conn) recordDeadlineExpired(env Envelope, deadline time.Duration, err error) {
	c.recordRouterEvent(RouterEvent{
		Kind:     RouterDeadlineExpired,
		Method:   env.Method,
		IDKey:    env.ID.Key(),
		Deadline: deadline,
		Err:      err,
	})

	slog.Warn("zcode: reverse-request deadline expired without an answer",
		"method", env.Method,
		"id", env.ID.Key(),
		"deadline", deadline.String(),
	)
}

// recordRouterEvent delivers one policy observation through the ordered
// router-event fan-out under routerObsMu — one mechanism (WR-01: the
// single-slot read discipline folded into the fan-out mutex; no
// double-lock, no mid-traffic registration race). The snapshot-then-call
// shape keeps callbacks free to register or remove observers mid-delivery.
func (c *Conn) recordRouterEvent(ev RouterEvent) {
	c.routerObsMu.Lock()
	observers := slices.Clone(c.routerObservers)
	c.routerObsMu.Unlock()

	for _, obs := range observers {
		obs.fn(ev)
	}
}

// replyRequest delivers a reverse-request answer on the bypass lane
// (dialect §1: answering from the queue would deadlock the pairing).
//
// WR-01 (plan 05-10): an answer carrying NEITHER result NOR error is
// dropped structurally — a JSON-RPC response carries exactly one of
// result/error, so the payload-less bare id-echo (Reply with a nil result
// marshals {"id":X}) is forbidden: framing it at a still-registered pending
// ask would reject it ("Invalid response" at the server's result-schema
// parse). The parked dedupe-join arm's (nil, nil, true) is therefore
// SILENCE BY CONSTRUCTION at conn close — no shutdown race the writer must
// lose.
func (c *Conn) replyRequest(id *ID, result any, werr *WireError) {
	if result == nil && werr == nil {
		// Structural silence: the id key only — request params never log.
		slog.Warn("zcode: reverse-request answer carried neither result nor error; dropped (structural silence)",
			"id", id.Key())

		return
	}

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
