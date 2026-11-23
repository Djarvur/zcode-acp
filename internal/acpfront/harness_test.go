package acpfront_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Front-tier test harness (plan 04-02, RESEARCH Pattern 6): the
// in-process composition every CORE test and plan 04-06's tracer rides —
//
//	test client (SDK ClientSideConnection, the honest counterparty)
//	  ⇄ recording tee (raw ACP wire, both directions)
//	  ⇄ acpport connection over ANY caller-supplied acp.Agent impl
//	  ⇄ zcode.Conn ⇄ fake.Serve (the scripted corpus back leg)
//
// The helper deliberately does NOT import the acpfront package itself:
// plan 04-06 constructs the Front, wraps it via acpport.NewAgent, and
// becomes this harness's first full-loop consumer.
//
// Compile-only at this plan's boundary — no test drives the composition
// yet (the harness's own verify run filters to TestNothingPlaceholder);
// the Phase-3 discipline still binds: construction completes before any
// frame flows, exchanges run under bounded timeouts, and everything
// tears down via t.Cleanup. The surface is exported because it is this
// phase's shared test infrastructure: plans 04-03..04-06 add test files
// to this package and drive it.

const (
	// ComposeTimeout bounds a whole composed exchange; the composed loop
	// completes in milliseconds and a stuck exchange fails fast.
	ComposeTimeout = 10 * time.Second

	// serveWaitTimeout bounds fake.Serve's return after the host closes
	// its stdin (the Phase-3 runExchange bound).
	serveWaitTimeout = 5 * time.Second
)

// errServeTimeout is the sentinel for fake.Serve not returning after
// stdin EOF within the wait bound.
var errServeTimeout = errors.New("fake.Serve did not return after stdin EOF")

// TeeLine is one direction-tagged ACP frame: "out" lines are frames the
// teed side wrote, "in" lines the frames it received.
type TeeLine struct {
	Dir  string
	JSON string
}

// TeeRW is the recording-pipe tee (RESEARCH Pattern 6, ~40 LOC): an
// io.ReadWriter over one connection side's read/write legs that appends
// every completed newline-delimited ACP frame to a shared mutex-guarded
// list — the wire-order assertion surface (chunk frames before the
// prompt response, exactly-once answers).
type TeeRW struct {
	mu    sync.Mutex
	lines *[]TeeLine

	r    io.Reader
	w    io.Writer
	rbuf []byte
	wbuf []byte
}

// NewTee wraps one connection side: r carries the frames it receives, w
// the frames it writes; both record into the shared lines list.
func NewTee(r io.Reader, w io.Writer, lines *[]TeeLine) *TeeRW {
	return &TeeRW{lines: lines, r: r, w: w}
}

// Read passes frames through and records complete received lines.
func (t *TeeRW) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)

	t.record("in", p[:n], &t.rbuf)

	return n, err
}

// Write records the outgoing bytes and passes them through.
func (t *TeeRW) Write(p []byte) (int, error) {
	t.record("out", p, &t.wbuf)

	return t.w.Write(p)
}

// Recorded snapshots the recorded frames in recorded (wire) order.
func (t *TeeRW) Recorded() []TeeLine {
	t.mu.Lock()
	defer t.mu.Unlock()

	lines := make([]TeeLine, len(*t.lines))
	copy(lines, *t.lines)

	return lines
}

// record appends p to the direction's line buffer and moves every
// completed newline-delimited frame onto the shared list.
func (t *TeeRW) record(dir string, p []byte, buf *[]byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	*buf = append(*buf, p...)

	for {
		idx := bytes.IndexByte(*buf, '\n')
		if idx < 0 {
			return
		}

		line := string((*buf)[:idx])
		*buf = (*buf)[idx+1:]

		*t.lines = append(*t.lines, TeeLine{Dir: dir, JSON: line})
	}
}

// ACPUpdate is one collected session update with its session id.
type ACPUpdate struct {
	SessionID string
	Update    acp.SessionUpdate
}

// PermissionScript configures the outcome every session/request_permission
// gets answered with (plan 05-02 TOOL-02 harness): the selected option's
// id, the cancelled arm, an optional hold before answering (the expiry
// arm's unanswered-client shape), and an optional gate the test closes to
// release the answer at a chosen moment (the cancel-row's deterministic
// sequencing — the ask stays pending until the gate opens). Cancelled wins
// when both arms are set; the Delay and Gate holds both honor the request
// context.
type PermissionScript struct {
	SelectedOptionID string
	Cancelled        bool
	Delay            time.Duration
	Gate             <-chan struct{}
}

// RecordingClient implements the SDK's Client interface for tests:
// SessionUpdate collects (sessionId, update) pairs under a mutex (drained
// by assertions); RequestPermission records every received ask and answers
// the scripted outcome (plan 05-02 — the TOOL-02 client counterparty);
// every other client-side request/notification callback is a no-op
// returning zero values — Phase 5 bridges no further client surfaces
// (elicitation/fs/terminal stay Phases 6+).
type RecordingClient struct {
	mu         sync.Mutex
	updates    []ACPUpdate
	permScript PermissionScript
	permAsks   []acp.RequestPermissionRequest
}

// SetPermissionScript installs the scripted request_permission outcome.
func (c *RecordingClient) SetPermissionScript(script PermissionScript) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.permScript = script
}

// PermissionAsks snapshots every received session/request_permission in
// arrival order.
func (c *RecordingClient) PermissionAsks() []acp.RequestPermissionRequest {
	c.mu.Lock()
	defer c.mu.Unlock()

	asks := make([]acp.RequestPermissionRequest, len(c.permAsks))
	copy(asks, c.permAsks)

	return asks
}

// SessionUpdate collects one streamed update.
func (c *RecordingClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.updates = append(c.updates, ACPUpdate{
		SessionID: string(params.SessionId),
		Update:    params.Update,
	})

	return nil
}

// DrainUpdates snapshots the collected CHUNK updates in arrival order —
// message/thought chunks and tool calls (the catalog's
// available_commands_update frame is a session-level nicety, not a turn
// update: it rides the ACPLines tee and the goldens instead, so the
// chunk-count contracts read unchanged). Everything collected stays
// reachable through the raw tee.
func (c *RecordingClient) DrainUpdates() []ACPUpdate {
	c.mu.Lock()
	defer c.mu.Unlock()

	updates := make([]ACPUpdate, 0, len(c.updates))
	for i := range c.updates {
		u := &c.updates[i]
		if u.Update.AgentMessageChunk != nil || u.Update.AgentThoughtChunk != nil ||
			u.Update.ToolCall != nil || u.Update.ToolCallUpdate != nil {
			updates = append(updates, *u)
		}
	}

	return updates
}

// CompleteElicitation is a no-op (elicitation completes are Phase 5+).
func (c *RecordingClient) CompleteElicitation(ctx context.Context, params acp.CompleteElicitationNotification) error {
	return nil
}

// CreateElicitation is a no-op (elicitation is Phase 5+).
func (c *RecordingClient) CreateElicitation(
	ctx context.Context,
	params acp.CreateElicitationRequest,
) (acp.CreateElicitationResponse, error) {
	return acp.CreateElicitationResponse{}, nil
}

// ReadTextFile is a no-op (fs surfaces are never advertised).
func (c *RecordingClient) ReadTextFile(
	ctx context.Context,
	params acp.ReadTextFileRequest,
) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, nil
}

// WriteTextFile is a no-op (fs surfaces are never advertised).
func (c *RecordingClient) WriteTextFile(
	ctx context.Context,
	params acp.WriteTextFileRequest,
) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, nil
}

// RequestPermission records the received ask and answers the scripted
// outcome: the selected arm ({outcome:selected, optionId}) or the
// cancelled arm ({outcome:cancelled} — the ACP v1 MUST on turn cancel);
// the optional Delay/Gate holds the answer open and honors the request
// context while held.
func (c *RecordingClient) RequestPermission(
	ctx context.Context,
	params acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
	c.mu.Lock()
	c.permAsks = append(c.permAsks, params)
	script := c.permScript
	c.mu.Unlock()

	if script.Delay > 0 || script.Gate != nil {
		// Nil-channel hold semantics: a zero Delay arms no timer (nil fires
		// never), an absent gate blocks forever — the answer releases on
		// whichever source fires first, or on the request context dying.
		var timer <-chan time.Time
		if script.Delay > 0 {
			timer = time.After(script.Delay)
		}

		select {
		case <-timer:
		case <-script.Gate:
		case <-ctx.Done():
			return acp.RequestPermissionResponse{}, ctx.Err()
		}
	}

	if script.Cancelled {
		return acp.RequestPermissionResponse{
			Outcome: acp.RequestPermissionOutcome{
				Cancelled: &acp.RequestPermissionOutcomeCancelled{Outcome: "cancelled"},
			},
		}, nil
	}

	return acp.RequestPermissionResponse{
		Outcome: acp.RequestPermissionOutcome{
			Selected: &acp.RequestPermissionOutcomeSelected{
				Outcome:  "selected",
				OptionId: acp.PermissionOptionId(script.SelectedOptionID),
			},
		},
	}, nil
}

// CreateTerminal is a no-op (terminal surfaces are never advertised).
func (c *RecordingClient) CreateTerminal(
	ctx context.Context,
	params acp.CreateTerminalRequest,
) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, nil
}

// KillTerminal is a no-op (terminal surfaces are never advertised).
func (c *RecordingClient) KillTerminal(
	ctx context.Context,
	params acp.KillTerminalRequest,
) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, nil
}

// TerminalOutput is a no-op (terminal surfaces are never advertised).
func (c *RecordingClient) TerminalOutput(
	ctx context.Context,
	params acp.TerminalOutputRequest,
) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, nil
}

// ReleaseTerminal is a no-op (terminal surfaces are never advertised).
func (c *RecordingClient) ReleaseTerminal(
	ctx context.Context,
	params acp.ReleaseTerminalRequest,
) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}

// WaitForTerminalExit is a no-op (terminal surfaces are never advertised).
func (c *RecordingClient) WaitForTerminalExit(
	ctx context.Context,
	params acp.WaitForTerminalExitRequest,
) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, nil
}

// rwCloser joins one io.Pipe half-pair into an io.ReadWriteCloser end.
type rwCloser struct {
	r *io.PipeReader
	w *io.PipeWriter
}

// Read passes through to the pipe reader.
func (p *rwCloser) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

// Write passes through to the pipe writer.
func (p *rwCloser) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

// Close tears down both halves; the peer sees EOF.
func (p *rwCloser) Close() error {
	return errors.Join(p.r.Close(), p.w.Close())
}

// fakeServeResult carries the fake's transcript and error across the
// goroutine boundary (channel ordering replaces a mutex).
type fakeServeResult struct {
	transcript fake.Transcript
	err        error
}

// FrontHarness carries the composed loop's assertion surfaces.
type FrontHarness struct {
	client    *acp.ClientSideConnection
	rec       *RecordingClient
	tee       *TeeRW
	conn      *zcode.Conn
	serveDone <-chan fakeServeResult

	closeServe sync.Once
	served     fakeServeResult
	serveErr   error
}

// ExchangeCtx returns the bounded context one composed exchange runs on.
func (h *FrontHarness) ExchangeCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), ComposeTimeout)
}

// ACPLines snapshots the recorded ACP wire frames (both directions, in
// recorded order) — the wire-order assertion surface.
func (h *FrontHarness) ACPLines() []TeeLine {
	return h.tee.Recorded()
}

// Updates drains the client's collected session updates.
func (h *FrontHarness) Updates() []ACPUpdate {
	return h.rec.DrainUpdates()
}

// Transcript closes the back leg, waits for fake.Serve to drain (dialect
// §1 EOF facts), and returns its recorded transcript. Idempotent.
func (h *FrontHarness) Transcript(t *testing.T) (fake.Transcript, error) {
	t.Helper()

	h.closeServe.Do(func() {
		if err := h.conn.Close(); err != nil {
			t.Logf("conn close: %v", err)
		}

		select {
		case result := <-h.serveDone:
			h.served = result
			h.serveErr = result.err
		case <-time.After(serveWaitTimeout):
			h.serveErr = errServeTimeout
		}
	})

	return h.served.transcript, h.serveErr
}

// ComposeFront wires the full in-process loop and returns the harness.
// newAgent receives the back-leg conn so caller-supplied acp.Agent impls
// that own it (plan 04-06's Front wrapped via acpport.NewAgent) are
// constructed before any frame flows — the Phase-3 construction
// discipline. Everything tears down via t.Cleanup.
func ComposeFront(t *testing.T, sc *fake.Scenario, newAgent func(conn *zcode.Conn) acp.Agent) *FrontHarness {
	t.Helper()

	h := &FrontHarness{}

	// Back leg: fake.Serve on one pipe pair end, zcode.Conn on the other
	// (the Phase-3 runExchange composition shape).
	toFakeR, toFakeW := io.Pipe()
	toHostR, toHostW := io.Pipe()

	hostEnd := &rwCloser{r: toHostR, w: toFakeW}
	fakeEnd := &rwCloser{r: toFakeR, w: toHostW}

	t.Cleanup(func() { _ = fakeEnd.Close() })

	serveDone := make(chan fakeServeResult, 1)

	go func() {
		tr, serveErr := fake.Serve(fakeEnd, sc)
		serveDone <- fakeServeResult{transcript: tr, err: serveErr}
	}()

	h.conn = zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = h.conn.Close() })

	h.serveDone = serveDone

	// ACP front leg: client ⇄ recording tee ⇄ acpport connection over the
	// caller-supplied agent. Construction completes before any frame
	// flows — no frame exists until the test speaks first.
	clientToAgentR, clientToAgentW := io.Pipe()
	agentToClientR, agentToClientW := io.Pipe()

	t.Cleanup(func() {
		for _, closer := range []io.Closer{clientToAgentR, clientToAgentW, agentToClientR, agentToClientW} {
			_ = closer.Close()
		}
	})

	var lines []TeeLine

	h.tee = NewTee(agentToClientR, clientToAgentW, &lines)
	h.rec = &RecordingClient{}

	agent := newAgent(h.conn)

	acpport.NewAgentSideConnection(agent, agentToClientW, clientToAgentR)
	h.client = acp.NewClientSideConnection(h.rec, h.tee, h.tee)

	return h
}
