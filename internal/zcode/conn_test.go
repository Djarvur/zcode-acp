package zcode_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/turn"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Test timing knobs: the positive assertions synchronize on channels (wire
// deliveries and call outcomes) and never on wall clock; graceWindow bounds
// only the negative assertions — a delivery that must NOT happen is checked
// after a quiet observation window.
const (
	testTimeout = 10 * time.Second
	graceWindow = 150 * time.Millisecond

	// deadStreamBound bounds the PROMPT-resolution assertions of the
	// stream-death and Close drain tests (review WR-01): a resolved call
	// must beat this bound by orders of magnitude, while the wedged RED
	// call — hung until its caller ctx — proves the hang with a bound far
	// below that ctx (testTimeout).
	deadStreamBound = 2 * time.Second

	// bigLineBytes drives the raised-buffer test: a 128 KiB line sits far
	// above the 64 KiB bufio.Scanner default (RESEARCH Pitfall 1) and well
	// below the conn's 16 MiB cap (T-3-02).
	bigLineBytes = 128 * 1024

	// pendingCalls is the in-flight call count for the WR-02 collision test:
	// integer ids 1..pendingCalls are assigned by the conn, and the colliding
	// reverse request carries the numeric id of the last one.
	pendingCalls = 7

	// testSessionID is the fixed session id for stop-frame shape assertions.
	testSessionID = "sess_x"
)

// errCorrelationBroken marks a cross-resolved call reply — a reply landing
// on the wrong pending call (the call-layer face of the WR-02 failure
// family).
var errCorrelationBroken = errors.New("id correlation broken")

// pipeEnd joins one io.Pipe half-pair into an io.ReadWriteCloser stream end.
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

// connPipes wires two io.Pipe half-pairs as the conn and peer stream ends:
// frames cross the real pipe boundary in both directions (D-01 mode A
// transport, without the fake — these tests script the peer by hand).
func connPipes() (connEnd, peerEnd *pipeEnd) {
	toPeerR, toPeerW := io.Pipe() // conn writes → peer reads
	toConnR, toConnW := io.Pipe() // peer writes → conn reads

	connEnd = &pipeEnd{r: toConnR, w: toPeerW}
	peerEnd = &pipeEnd{r: toPeerR, w: toConnW}

	return connEnd, peerEnd
}

// callOutcome is one settled Call: the result or the failure.
type callOutcome struct {
	res json.RawMessage
	err error
}

// issueCalls starts n concurrent Calls that the peer is expected NOT to
// answer; each outcome (result or error) is pushed to the returned channel
// when the call settles, so tests can assert pending stays pending.
func issueCalls(ctx context.Context, t *testing.T, conn *zcode.Conn, n int) <-chan callOutcome {
	t.Helper()

	outcomes := make(chan callOutcome, n)

	for range n {
		go func() {
			res, err := conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
				"workspace": map[string]any{"workspacePath": "/w", "workspaceKey": "/w"},
			})

			outcomes <- callOutcome{res: res, err: err}
		}()
	}

	return outcomes
}

// collectNotifications registers a conn notification observer that delivers
// notification envelopes on the returned channel (buffered; deliveries
// happen on the read goroutine in wire order).
func collectNotifications(conn *zcode.Conn) <-chan zcode.Envelope {
	ch := make(chan zcode.Envelope, 64)

	conn.AddObserver(func(env zcode.Envelope) {
		ch <- env
	})

	return ch
}

// collectRequests hooks Conn.OnRequest to deliver intercepted reverse
// requests on the returned channel (buffered; the hook runs on a
// per-request goroutine). The hook defers to the built-in router
// (handled=false) so the router's default answers stay exercised.
func collectRequests(conn *zcode.Conn) <-chan zcode.Envelope {
	ch := make(chan zcode.Envelope, 16)

	conn.OnRequest = func(_ context.Context, env zcode.Envelope) (any, *zcode.WireError, bool) {
		ch <- env

		return nil, nil, false
	}

	return ch
}

// drainPeer reads every line the conn writes to the peer side until the
// stream closes, pushing each complete wire line (newline stripped) to the
// returned channel — the recording peer.
func drainPeer(t *testing.T, peer *pipeEnd) <-chan string {
	t.Helper()

	lines := make(chan string, 256)

	go func() {
		defer close(lines)

		sc := bufio.NewScanner(peer.r)
		for sc.Scan() {
			lines <- string(sc.Bytes())
		}
	}()

	return lines
}

// readFrames reads exactly n wire lines from the peer side on a goroutine.
// io.Pipe's synchronous writes keep the conn's writer blocked until a reader
// appears, which is precisely the slow-peer premise of the bypass test.
func readFrames(peer *pipeEnd, n int) <-chan []string {
	lines := make(chan []string, 1)

	go func() {
		var got []string

		sc := bufio.NewScanner(peer.r)
		for len(got) < n && sc.Scan() {
			got = append(got, sc.Text())
		}

		lines <- got
	}()

	return lines
}

// waitLines reads exactly n wire lines from ch, failing the test on timeout
// or premature stream close.
func waitLines(t *testing.T, ch <-chan string, n int, timeout time.Duration) []string {
	t.Helper()

	got := make([]string, 0, n)

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for len(got) < n {
		select {
		case line, ok := <-ch:
			if !ok {
				t.Fatalf("peer stream closed after %d of %d lines", len(got), n)
			}

			got = append(got, line)
		case <-timer.C:
			t.Fatalf("timed out waiting for %d peer lines (got %d)", n, len(got))
		}
	}

	return got
}

// decodeWireLine parses one wire line into an envelope, failing the test on
// any corruption — an interleaved or truncated frame is not valid JSON
// (RESEARCH Pitfall 7, single-writer contract).
func decodeWireLine(t *testing.T, line string) zcode.Envelope {
	t.Helper()

	env, err := zcode.Decode([]byte(line))
	if err != nil {
		t.Fatalf("corrupt frame on the wire: %v (%s)", err, line)
	}

	return env
}

// injectLine writes one wire line into the peer side (peer→conn direction);
// io.Pipe's synchronous write returns once the conn's read loop consumed it.
func injectLine(t *testing.T, peer *pipeEnd, line string) {
	t.Helper()

	if _, err := peer.Write([]byte(line + "\n")); err != nil {
		t.Fatalf("inject line: %v", err)
	}
}

// waitEnvelope waits for one hook delivery, failing the test on timeout.
func waitEnvelope(t *testing.T, ch <-chan zcode.Envelope, what string) zcode.Envelope {
	t.Helper()

	select {
	case env := <-ch:
		return env
	case <-time.After(testTimeout):
		t.Fatalf("%s hook never fired", what)

		return zcode.Envelope{}
	}
}

// waitOutcome waits for one settled call, failing the test on timeout.
func waitOutcome(t *testing.T, ch <-chan callOutcome) callOutcome {
	t.Helper()

	select {
	case oc := <-ch:
		return oc
	case <-time.After(testTimeout):
		t.Fatal("pending call never settled")

		return callOutcome{}
	}
}

// assertCount fails the test unless the recorder holds exactly n deliveries
// after a short quiet window — negative assertions need a bounded
// observation, never an instant check.
func assertCount(t *testing.T, rec *hookRecorder, n int, what string) {
	t.Helper()

	time.Sleep(graceWindow)

	if got := rec.count(); got != n {
		t.Fatalf("%s deliveries = %d, want %d", what, got, n)
	}
}

// hookRecorder collects hook deliveries, preserving arrival order; each
// delivery also signals ch so tests can wait deterministically without
// consuming the record.
type hookRecorder struct {
	mu   sync.Mutex
	envs []zcode.Envelope
	ch   chan struct{}
}

// newHookRecorder returns an empty recorder.
func newHookRecorder() *hookRecorder {
	return &hookRecorder{ch: make(chan struct{}, 256)}
}

// record appends one delivery, then signals (order matters: the record is
// visible before the signal arrives).
func (h *hookRecorder) record(env zcode.Envelope) {
	h.mu.Lock()
	h.envs = append(h.envs, env)
	h.mu.Unlock()

	h.ch <- struct{}{}
}

// count snapshots the delivery count.
func (h *hookRecorder) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.envs)
}

// waitDelivery waits for one delivery signal and returns the full record.
func (h *hookRecorder) waitDelivery(t *testing.T, what string) []zcode.Envelope {
	t.Helper()

	select {
	case <-h.ch:
	case <-time.After(testTimeout):
		t.Fatalf("%s hook never fired", what)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return slices.Clone(h.envs)
}

// WR-02 (02-REVIEW, routed here by disposition): a server→host request
// bearing a numeric id that collides with an in-flight call's id routes to
// the request router and NEVER resolves the pending call — the false-accept
// the probe's reply classifier could produce must be structurally
// impossible in the real demux.
func TestDemuxReverseRequestNeverResolvesPendingCall(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	requests := collectRequests(conn)
	peerLines := drainPeer(t, peerEnd)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outcomes := issueCalls(ctx, t, conn, pendingCalls)

	// All request frames on the wire ⇒ all pending entries registered
	// (registration precedes enqueue in Call).
	waitLines(t, peerLines, pendingCalls, testTimeout)

	// The collision: a method-bearing frame with the NUMERIC id of the last
	// in-flight call (dialect §3 interaction family).
	collide := fmt.Sprintf(`{"id":%d,"method":"interaction/requestPermission","params":{"sessionId":%q}}`,
		pendingCalls, testSessionID)
	injectLine(t, peerEnd, collide)

	// The router, not the reply table, owns the frame.
	got := waitEnvelope(t, requests, "request")

	if got.ID == nil || !got.ID.IsInt || got.ID.Int != pendingCalls {
		t.Fatalf("OnRequest id = %+v, want integer %d", got.ID, pendingCalls)
	}

	if got.Method != "interaction/requestPermission" {
		t.Fatalf("OnRequest method = %q, want interaction/requestPermission", got.Method)
	}

	// Negative half: after a quiet window past the demux, none of the seven
	// calls may have resolved.
	time.Sleep(graceWindow)

	for i := range pendingCalls {
		select {
		case oc := <-outcomes:
			t.Fatalf("call %d resolved by a method-bearing frame (WR-02 regression): %+v", i+1, oc)
		default:
		}
	}
}

// Ids correlate by exact value AND type (dialect §1: replies echo the
// request id verbatim; "1" the string and 1 the int are different wire
// values): a reply keyed with the other type is a no-op, and only the
// own-type echo resolves the pending call.
func TestDemuxIdTypeNonConflation(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	peerLines := drainPeer(t, peerEnd)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	outcomes := issueCalls(ctx, t, conn, 1)
	waitLines(t, peerLines, 1, testTimeout) // pending entry registered (int id 1)

	// Foreign TYPE: the string id "1" never resolves the integer-id call.
	injectLine(t, peerEnd, `{"id":"1","result":{"foreign":true}}`)

	time.Sleep(graceWindow)

	select {
	case oc := <-outcomes:
		t.Fatalf("string-typed reply resolved an integer-id call: %+v", oc)
	default:
	}

	// Own type: the integer id 1 echo resolves it (verbatim echo, §1).
	injectLine(t, peerEnd, `{"id":1,"result":{"accepted":true}}`)

	select {
	case oc := <-outcomes:
		if oc.err != nil {
			t.Fatalf("own-type reply errored the call: %v", oc.err)
		}

		if string(oc.res) != `{"accepted":true}` {
			t.Fatalf("own-type reply result = %s, want {\"accepted\":true}", oc.res)
		}
	case <-time.After(testTimeout):
		t.Fatal("verbatim-id reply never resolved the pending call")
	}
}

// Each inbound frame class lands on exactly one demux path: a reply resolves
// the pending table, a reverse request (id AND method) fires OnRequest, a
// notification (method, no id) reaches the notification observers — and none
// crosses into another path.
func TestDemuxClassification(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	notices := newHookRecorder()
	conn.AddObserver(notices.record)

	requests := newHookRecorder()
	conn.OnRequest = func(_ context.Context, env zcode.Envelope) (any, *zcode.WireError, bool) {
		requests.record(env)

		return nil, nil, false
	}

	peerLines := drainPeer(t, peerEnd)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outcomes := issueCalls(ctx, t, conn, 1)
	waitLines(t, peerLines, 1, testTimeout)

	// Notification: the notification observers only.
	injectLine(t, peerEnd, `{"method":"state.updated","params":{"patch":{"status":"running"}}}`)

	n := lastEnvelope(t, notices.waitDelivery(t, "notification"))

	if n.Method != zcode.MethodStateUpdated {
		t.Fatalf("notification method = %q, want %q", n.Method, zcode.MethodStateUpdated)
	}

	assertCount(t, requests, 0, "request")

	// Reverse request: OnRequest only (string id — the live server's
	// reverse-request id shape, EXP-WORKSPACEKEY-01.jsonl:8).
	injectLine(t, peerEnd,
		`{"id":"server-1","method":"session/requestRuntimePreferences","params":{"scope":"runtime-materialization"}}`)

	r := lastEnvelope(t, requests.waitDelivery(t, "request"))

	if r.ID == nil || r.ID.IsInt || r.ID.Str != "server-1" {
		t.Fatalf("OnRequest id = %+v, want string server-1", r.ID)
	}

	assertCount(t, notices, 1, "notification")

	// Reply: the pending table only; hooks stay at exactly one delivery.
	injectLine(t, peerEnd, `{"id":1,"result":{"session":{"sessionId":"sess_x"}}}`)

	oc := waitOutcome(t, outcomes)

	if oc.err != nil {
		t.Fatalf("reply path failed the pending call: %v", oc.err)
	}

	if string(oc.res) != `{"session":{"sessionId":"sess_x"}}` {
		t.Fatalf("reply result = %s, want the injected session snapshot", oc.res)
	}

	assertCount(t, notices, 1, "notification")
	assertCount(t, requests, 1, "request")
}

// lastEnvelope returns the final delivery of a hook record.
func lastEnvelope(t *testing.T, envs []zcode.Envelope) zcode.Envelope {
	t.Helper()

	if len(envs) == 0 {
		t.Fatal("hook record is empty")

		return zcode.Envelope{}
	}

	return envs[len(envs)-1]
}

// Raised-line behavior (PITFALL 7 / T-3-02): a 128 KiB valid line survives
// intact — far above the 64 KiB bufio default, far below the 16 MiB cap.
func TestLineBufferLargeLineIntact(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	notices := collectNotifications(conn)

	blob := strings.Repeat("a", bigLineBytes)
	injectLine(t, peerEnd, fmt.Sprintf(`{"method":"state.updated","params":{"blob":%q}}`, blob))

	select {
	case env := <-notices:
		var params struct {
			Blob string `json:"blob"`
		}

		if err := json.Unmarshal(env.Params, &params); err != nil {
			t.Fatalf("decode big-line params: %v", err)
		}

		if len(params.Blob) != bigLineBytes {
			t.Fatalf("128 KiB line arrived truncated: blob = %d bytes, want %d", len(params.Blob), bigLineBytes)
		}
	case <-time.After(testTimeout):
		t.Fatal("128 KiB line never delivered — the raised buffer is not raised")
	}
}

// Malformed-line behavior: a non-JSON line between two valid lines is
// skipped with a logged structural warning while the stream continues —
// wire order of the valid lines is preserved across the skip.
func TestLineBufferMalformedLineBetweenValidLines(t *testing.T) {
	var logBuf bytes.Buffer

	prevLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuf, nil)))
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	notices := collectNotifications(conn)

	injectLine(t, peerEnd, `{"method":"state.updated","params":{"seq":1}}`)
	injectLine(t, peerEnd, `{{{not-json-at-all`)
	injectLine(t, peerEnd, `{"method":"state.updated","params":{"seq":2}}`)

	first := waitEnvelope(t, notices, "notification")
	second := waitEnvelope(t, notices, "notification")

	if got := noticeSeq(t, first); got != 1 {
		t.Fatalf("first valid line seq = %d, want 1", got)
	}

	if got := noticeSeq(t, second); got != 2 {
		t.Fatalf("second valid line seq = %d, want 2 (stream did not continue past the malformed line)", got)
	}

	if !bytes.Contains(logBuf.Bytes(), []byte("skipping undecodable line")) {
		t.Fatalf("malformed line skip was not logged: %q", logBuf.String())
	}
}

// noticeSeq digs the integer seq field out of a test notification's params.
func noticeSeq(t *testing.T, env zcode.Envelope) int {
	t.Helper()

	var params struct {
		Seq int `json:"seq"`
	}

	if err := json.Unmarshal(env.Params, &params); err != nil {
		t.Fatalf("decode notice params: %v", err)
	}

	return params.Seq
}

// D-05 router policy — RED target: the mid-turn credential-cipher obligation
// is known-but-NOT-answered in this phase (needs the Phase 6 keyring vault).
// The router must write ZERO reply bytes (a -32601 would violate D-05 —
// replying at all would claim an answer the wrapper cannot give) and record
// the deadline expiry observably, exactly once per tracked request.
func TestRouterCredentialCipherNeverAnsweredExpiryRecorded(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	const cipherDeadline = 60 * time.Millisecond

	conn.RouterDeadline = func(string) time.Duration { return cipherDeadline }

	events := collectRouterEvents(conn)
	peerLines := drainPeer(t, peerEnd)

	// EXP-TURN-01.jsonl:50 params shape: requestId, operation "decrypt",
	// value (a 3.18-only surface — dialect §3/§9 drift baseline).
	injectLine(t, peerEnd,
		`{"id":"server-1","method":"interaction/requestCredentialCipher",`+
			`"params":{"requestId":"req-1","operation":"decrypt","value":"enc:v2:os:c2VjcmV0"}}`)

	// The tracked-no-answer deadline elapses; a quiet window beyond it must
	// hold no reply bytes at all.
	time.Sleep(cipherDeadline + graceWindow)

	select {
	case line := <-peerLines:
		t.Fatalf("credential-cipher request drew a reply — D-05 violation: %s", line)
	default:
	}

	// Exactly one recorded expiry observation per deadline, naming the
	// tracked request and the governing deadline.
	ev := waitEvent(t, events, zcode.RouterDeadlineExpired)

	if ev.Method != zcode.MethodRequestCredentialCipher {
		t.Fatalf("expiry event method = %q, want %q", ev.Method, zcode.MethodRequestCredentialCipher)
	}

	if ev.IDKey != "s:server-1" {
		t.Fatalf("expiry event id key = %q, want s:server-1", ev.IDKey)
	}

	if ev.Deadline != cipherDeadline {
		t.Fatalf("expiry event deadline = %v, want the injected %v", ev.Deadline, cipherDeadline)
	}

	if !errors.Is(ev.Err, context.DeadlineExceeded) {
		t.Fatalf("expiry event err = %v, want context.DeadlineExceeded", ev.Err)
	}

	assertNoEvents(t, events, "expiry")
}

// D-05 router policy — RED target: the runtime-preferences handler runs
// under its own context deadline (default 15 s, injectable) and answers with
// the static defaults (EXP-WORKSPACEKEY-01.jsonl:9 shape); the governing
// deadline is recorded on the router event.
func TestRouterRuntimePreferencesDeadline(t *testing.T) {
	if zcode.RouterDefaultDeadline != 15*time.Second {
		t.Fatalf("RouterDefaultDeadline = %v, want the D-05 15 s value", zcode.RouterDefaultDeadline)
	}

	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	const injected = 40 * time.Millisecond

	conn.RouterDeadline = func(string) time.Duration { return injected }

	events := collectRouterEvents(conn)
	peerLines := drainPeer(t, peerEnd)

	injectLine(t, peerEnd,
		`{"id":"server-1","method":"session/requestRuntimePreferences",`+
			`"params":{"sessionId":"sess_x","scope":"runtime-materialization"}}`)

	env := decodeWireLine(t, waitLines(t, peerLines, 1, testTimeout)[0])
	assertRuntimePreferencesReply(t, env, "server-1")

	ev := waitEvent(t, events, zcode.RouterAnswered)

	if ev.Method != zcode.MethodRequestRuntimePreferences {
		t.Fatalf("answered event method = %q, want %q", ev.Method, zcode.MethodRequestRuntimePreferences)
	}

	if ev.IDKey != "s:server-1" {
		t.Fatalf("answered event id key = %q, want s:server-1", ev.IDKey)
	}

	if ev.Deadline != injected {
		t.Fatalf("answered event deadline = %v, want the injected %v", ev.Deadline, injected)
	}
}

// Unknown server→host methods are still answered -32601 naming the method
// (probe precedent — dialect §3; the tracer default, locked here).
func TestRouterUnknownMethodAnswered32601(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	peerLines := drainPeer(t, peerEnd)

	injectLine(t, peerEnd,
		`{"id":"server-9","method":"interaction/requestPermission","params":{"sessionId":"sess_x"}}`)

	env := decodeWireLine(t, waitLines(t, peerLines, 1, testTimeout)[0])

	if env.Error == nil || env.Error.Code != zcode.CodeMethodNotFound {
		t.Fatalf("unknown method answered %+v, want -32601", env.Error)
	}

	if env.Result != nil {
		t.Fatalf("unknown method answered with a result: %s", env.Result)
	}

	if !strings.Contains(env.Error.Message, "interaction/requestPermission") {
		t.Fatalf("-32601 message %q does not name the method", env.Error.Message)
	}
}

// collectRouterEvents registers on Conn's router-event fan-out to deliver
// router policy observations on the returned channel (buffered; the
// observer runs on the request or tracker goroutine).
func collectRouterEvents(conn *zcode.Conn) <-chan zcode.RouterEvent {
	ch := make(chan zcode.RouterEvent, 16)

	conn.AddRouterObserver(func(ev zcode.RouterEvent) {
		ch <- ev
	})

	return ch
}

// waitEvent waits for the next router event of the given kind, skipping
// other observations, failing the test on timeout.
func waitEvent(t *testing.T, ch <-chan zcode.RouterEvent, kind zcode.RouterEventKind) zcode.RouterEvent {
	t.Helper()

	timer := time.NewTimer(testTimeout)
	defer timer.Stop()

	for {
		select {
		case ev := <-ch:
			if ev.Kind == kind {
				return ev
			}
		case <-timer.C:
			t.Fatalf("router never recorded a %s event", kind)

			return zcode.RouterEvent{}
		}
	}
}

// assertNoEvents fails the test if any further router events arrive after a
// quiet window — the exactly-one-observation half of the honesty contract.
func assertNoEvents(t *testing.T, ch <-chan zcode.RouterEvent, what string) {
	t.Helper()

	time.Sleep(graceWindow)

	if got := len(ch); got != 0 {
		t.Fatalf("unexpected extra %s observations: %d", what, got)
	}
}

// assertRuntimePreferencesReply checks a reply envelope for the verbatim
// static runtime-preference defaults (EXP-WORKSPACEKEY-01.jsonl:9) echoed to
// the string id wantID — decoded as a map so extra keys fail.
func assertRuntimePreferencesReply(t *testing.T, env zcode.Envelope, wantID string) {
	t.Helper()

	if env.ID == nil || env.ID.IsInt || env.ID.Str != wantID {
		t.Fatalf("reply id = %+v, want string %s", env.ID, wantID)
	}

	var got map[string]any

	if err := json.Unmarshal(env.Result, &got); err != nil {
		t.Fatalf("decode runtime-preferences result: %v", err)
	}

	want := map[string]any{
		"nativeSearchEnhancementsEnabled": false,
		"memoryEnabled":                   false,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime-preferences result = %v, want exactly %v", got, want)
	}
}

// concurrentCalls fans out n concurrent calls, each verified to receive its
// OWN reply (the echoed params carry the caller's marker) — per-call
// correlation made visible. Failures are collected on the returned channel.
func concurrentCalls(ctx context.Context, t *testing.T, conn *zcode.Conn, n int) <-chan error {
	t.Helper()

	errs := make(chan error, n)

	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)

		go func(marker int) {
			defer wg.Done()

			res, err := conn.Call(ctx, zcode.MethodSessionMessages, map[string]any{"marker": marker})
			if err != nil {
				errs <- fmt.Errorf("call %d: %w", marker, err)

				return
			}

			var echo struct {
				Marker int `json:"marker"`
			}

			if err := json.Unmarshal(res, &echo); err != nil {
				errs <- fmt.Errorf("call %d: decode echo: %w", marker, err)

				return
			}

			if echo.Marker != marker {
				errs <- fmt.Errorf("call %d received call %d's reply: %w", marker, echo.Marker, errCorrelationBroken)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	return errs
}

// drainCallErrors fails the test if any fanned-out call reported an error.
func drainCallErrors(t *testing.T, errs <-chan error) {
	t.Helper()

	for err := range errs {
		t.Error(err)
	}
}

// fanoutTurnWindow bounds one interleaved machine turn in the fan-out
// two-machines subtest: the GREEN run resolves in milliseconds; a RED
// slot-theft hang fails the subtest on this bound.
const fanoutTurnWindow = 2 * time.Second

// The observer fan-out contract (Phase 4, RESEARCH Pattern 4): notifications
// fan out to ALL registered observers — synchronously, in registration
// order, on the read goroutine (wire order preserved) — with a remove func
// that deregisters, and with two machines on one conn each resolving their
// own interleaved turns exactly once (the single slot this replaces let a
// second machine silently steal the hook, PITFALL 3: the first session's
// turns hang).
func TestObserverFanOut(t *testing.T) {
	t.Run("ordering", testObserverFanOutOrdering)
	t.Run("remove", testObserverFanOutRemove)
	t.Run("two-machines", testObserverFanOutTwoMachines)
}

// testObserverFanOutOrdering locks the delivery contract: registration
// order per envelope, synchronous on the read goroutine (a parked observer
// pins the read loop — nothing reads ahead), and wire order across
// envelopes.
func testObserverFanOutOrdering(t *testing.T) {
	t.Helper()

	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	var (
		mu    sync.Mutex
		trace []string
		sigs  = make(chan struct{}, 8)
	)

	record := func(tag string) func(zcode.Envelope) {
		return func(env zcode.Envelope) {
			mu.Lock()
			trace = append(trace, tag+":"+env.Method)
			mu.Unlock()

			sigs <- struct{}{}
		}
	}

	// The first observer parks inside its delivery until released.
	release := make(chan struct{})
	conn.AddObserver(func(env zcode.Envelope) {
		<-release
		record("a")(env)
	})
	conn.AddObserver(record("b"))

	// Envelope 1 parks the first observer inside the delivery.
	injectLine(t, peerEnd, `{"method":"state.updated","params":{"patch":{"status":"running"}}}`)

	// While the first observer is parked, a second envelope cannot be
	// consumed: delivery is synchronous on the read goroutine, so the read
	// loop is inside the observer — not reading ahead (the grace window
	// bounds the observation; an early write completion fails the subtest).
	secondLine := `{"method":"session/event","params":{"type":"turn.completed","seq":1,"turnId":"t1"}}`

	secondWritten := make(chan error, 1)
	go func() {
		_, err := peerEnd.Write([]byte(secondLine + "\n"))
		secondWritten <- err
	}()

	select {
	case err := <-secondWritten:
		t.Fatalf("second envelope consumed while the first observer was still parked (err=%v) — "+
			"delivery is not synchronous on the read goroutine", err)
	case <-time.After(graceWindow):
	}

	close(release)

	// Four deliveries: two observers x two envelopes, registration order
	// per envelope, wire order across envelopes.
	want := []string{
		"a:" + zcode.MethodStateUpdated, // registration order: first-registered first
		"b:" + zcode.MethodStateUpdated,
		"a:" + zcode.MethodSessionEvent, // wire order across envelopes preserved
		"b:" + zcode.MethodSessionEvent,
	}

	waitFanoutTrace(t, &mu, &trace, sigs, want)

	if err := <-secondWritten; err != nil {
		t.Fatalf("second envelope write failed: %v", err)
	}
}

// waitFanoutTrace waits for one delivery signal per expected entry and
// asserts the recorded trace equals want (order-sensitive).
func waitFanoutTrace(t *testing.T, mu *sync.Mutex, trace *[]string, sigs <-chan struct{}, want []string) {
	t.Helper()

	for range want {
		select {
		case <-sigs:
		case <-time.After(testTimeout):
			mu.Lock()
			got := len(*trace)
			mu.Unlock()

			t.Fatalf("only %d deliveries arrived within %v", got, testTimeout)
		}
	}

	mu.Lock()
	got := slices.Clone(*trace)
	mu.Unlock()

	if !slices.Equal(got, want) {
		t.Fatalf("delivery trace = %v, want %v", got, want)
	}
}

// testObserverFanOutRemove locks the deregistration contract: the remove
// func is idempotent, the removed observer receives nothing from a later
// envelope, and the remaining observers are unaffected.
func testObserverFanOutRemove(t *testing.T) {
	t.Helper()

	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	removed := newHookRecorder()
	kept := newHookRecorder()

	removeRemoved := conn.AddObserver(removed.record)
	conn.AddObserver(kept.record)

	// Deregister the first observer (a repeated call is a no-op — the
	// remove func is idempotent).
	removeRemoved()
	removeRemoved()

	injectLine(t, peerEnd, `{"method":"state.updated","params":{"patch":{"status":"running"}}}`)

	n := lastEnvelope(t, kept.waitDelivery(t, "kept observer"))

	if n.Method != zcode.MethodStateUpdated {
		t.Fatalf("kept observer method = %q, want %q", n.Method, zcode.MethodStateUpdated)
	}

	assertCount(t, removed, 0, "removed observer")
}

// testObserverFanOutTwoMachines locks the multi-session contract: two
// machines on one conn run interleaved (sequential) turns, each resolving
// exactly once via its own terminal event.
func testObserverFanOutTwoMachines(t *testing.T) {
	t.Helper()

	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	// Both machines register BEFORE any frame flows (the Conn hook
	// contract). The second construction must not steal the first's
	// notifications (PITFALL 3).
	spyA := newFanoutResolutionSpy()
	machineA := turn.NewMachine(conn, "sess_a", turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))
	machineA.OnResolve = spyA.record

	spyB := newFanoutResolutionSpy()
	machineB := turn.NewMachine(conn, "sess_b", turn.NewPollWatcher(conn, turn.PollConfig{
		QuietMs:        40,
		FloorMultiple:  2,
		PollIntervalMs: 10,
	}))
	machineB.OnResolve = spyB.record

	// The peer auto-answers every request the turns generate:
	// session/send acks and the poll watchers' session/messages pages. The
	// id echoes verbatim (the codec's type-tagged id rule).
	sends := make(chan string, 8)
	go autoAnswerPeer(peerEnd, sends)

	// Interleaved (sequential) turns on the shared conn: each machine
	// resolves its own turn exactly once — the first session's turns still
	// complete after the second machine registered (no hook theft), and
	// neither machine observes the other's resolution.
	runFanoutTurn(t, peerEnd, sends, machineA, spyA, "sess_a", "turn_a")
	assertFanoutResolution(t, spyB, 0, "machine B before its turn")

	runFanoutTurn(t, peerEnd, sends, machineB, spyB, "sess_b", "turn_b")
	assertFanoutResolution(t, spyA, 1, "machine A after B's turn")
	assertFanoutResolution(t, spyB, 1, "machine B")
}

// runFanoutTurn drives one machine turn against the hand-scripted peer: it
// starts Run, waits for the send to reach the peer (proof the machine is
// running), injects the turnId-matched started/completed signals, and
// asserts the run resolved exactly once via the terminal EVENT — the poll
// watcher is machine-owned and never stolen, so a stolen notification hook
// still resolves the turn via the watcher's quiet window instead; the wrong
// signal is the theft's fingerprint (RESEARCH Pattern 4).
func runFanoutTurn(
	t *testing.T,
	peer *pipeEnd,
	sends <-chan string,
	machine *turn.Machine,
	spy *fanoutResolutionSpy,
	sessionID, turnID string,
) {
	t.Helper()

	type runResult struct {
		reason turn.StopReason
		err    error
	}

	done := make(chan runResult, 1)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fanoutTurnWindow)
		defer cancel()

		reason, err := machine.Run(ctx, "Reply with exactly: OK")
		done <- runResult{reason: reason, err: err}
	}()

	select {
	case got := <-sends:
		if got != sessionID {
			t.Fatalf("send session = %q, want %q", got, sessionID)
		}
	case <-time.After(fanoutTurnWindow):
		t.Fatalf("%s send never reached the peer", sessionID)
	}

	injectLine(t, peer, sessionEventLine(turn.KindTurnStarted, 1, turnID))
	injectLine(t, peer, sessionEventLine(turn.KindTurnCompleted, 2, turnID))

	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("%s machine run: %v", sessionID, res.err)
		}

		if res.reason != turn.StopCompleted {
			t.Fatalf("%s machine reason = %q, want %q", sessionID, res.reason, turn.StopCompleted)
		}
	case <-time.After(testTimeout):
		t.Fatalf("%s machine never resolved its turn", sessionID)
	}

	got := spy.snapshot()

	if len(got) != 1 || got[0].Signal != turn.SignalTerminalEvent {
		t.Fatalf("%s resolved %d times, last signal %q — want exactly once via %q "+
			"(a stolen hook resolves via the watcher's quiet window instead)",
			sessionID, len(got), got[len(got)-1].Signal, turn.SignalTerminalEvent)
	}
}

// sessionEventLine renders one session/event carrier line for the
// turn-scoped signals the two-machines subtest injects.
func sessionEventLine(kind string, seq int, turnID string) string {
	return fmt.Sprintf(`{"method":"session/event","params":{"type":%q,"seq":%d,"turnId":%q}}`, kind, seq, turnID)
}

// autoAnswerPeer answers every request the conn writes to the peer end:
// session/send acks and session/messages pages (the machine never inspects
// the result bodies). Runs until the peer stream closes; the test's own
// injectLine calls share the writer safely (io.Pipe serializes writes).
func autoAnswerPeer(peer *pipeEnd, sends chan<- string) {
	sc := bufio.NewScanner(peer.r)

	for sc.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params map[string]any  `json:"params"`
		}

		if err := json.Unmarshal(sc.Bytes(), &req); err != nil || req.ID == nil {
			continue
		}

		if req.Method == zcode.MethodSessionSend {
			// Wire payloads keep the dialect's camelCase keys verbatim (the
			// tagliatelle contract), so the session id digs out of the
			// params map (the machine.go classifyIdlePatch precedent).
			if sessionID, _ := req.Params["sessionId"].(string); sessionID != "" {
				sends <- sessionID
			}
		}

		result := `{"accepted":true}`
		if req.Method == zcode.MethodSessionMessages {
			result = `{"messages":[]}`
		}

		if _, err := peer.Write([]byte(fmt.Sprintf(`{"id":%s,"result":%s}`, req.ID, result) + "\n")); err != nil {
			close(sends)

			return
		}
	}

	close(sends)
}

// fanoutResolutionSpy is the exactly-once observation hook for the
// two-machines subtest: a synchronized record of every OnResolve delivery
// (the hook runs on the read goroutine; the test reads it under the same
// lock).
type fanoutResolutionSpy struct {
	mu  sync.Mutex
	got []turn.Resolution
}

// newFanoutResolutionSpy returns an empty spy.
func newFanoutResolutionSpy() *fanoutResolutionSpy {
	return &fanoutResolutionSpy{}
}

// record implements the OnResolve hook signature.
func (s *fanoutResolutionSpy) record(res turn.Resolution) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.got = append(s.got, res)
}

// count snapshots the resolution count.
func (s *fanoutResolutionSpy) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.got)
}

// snapshot returns the recorded resolutions.
func (s *fanoutResolutionSpy) snapshot() []turn.Resolution {
	s.mu.Lock()
	defer s.mu.Unlock()

	return slices.Clone(s.got)
}

// assertFanoutResolution fails the test unless the spy holds exactly n
// resolutions.
func assertFanoutResolution(t *testing.T, spy *fanoutResolutionSpy, n int, what string) {
	t.Helper()

	if got := spy.count(); got != n {
		t.Fatalf("%s resolutions = %d, want %d", what, got, n)
	}
}

// WR-01 router-event fan-out (05-03 TDD RED): two router observers
// registered A then B must receive every router event in registration
// order, and removing A must stop A's deliveries without affecting B.
// RED: AddRouterObserver is a compiling stub over the single slot
// (last-wins, no-op remove) — both subtests fail on their ordering /
// delivery assertions until the fan-out lands.
func TestRouterFanout(t *testing.T) {
	connEnd, peerEnd := connPipes()
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(connEnd)
	t.Cleanup(func() { _ = conn.Close() })

	const deadline = 60 * time.Millisecond

	conn.RouterDeadline = func(string) time.Duration { return deadline }

	peerLines := drainPeer(t, peerEnd)

	var recMu sync.Mutex
	var order []string

	removeA := conn.AddRouterObserver(func(zcode.RouterEvent) {
		recMu.Lock()
		defer recMu.Unlock()

		order = append(order, "A")
	})

	_ = conn.AddRouterObserver(func(zcode.RouterEvent) {
		recMu.Lock()
		defer recMu.Unlock()

		order = append(order, "B")
	})

	injectLine(t, peerEnd,
		`{"id":"server-1","method":"interaction/requestCredentialCipher",`+
			`"params":{"requestId":"req-1","operation":"decrypt","value":"enc:v2:os:c2VjcmV0"}}`)

	time.Sleep(deadline + graceWindow)

	select {
	case line := <-peerLines:
		t.Fatalf("credential-cipher request drew a reply — D-05 violation: %s", line)
	default:
	}

	recMu.Lock()
	gotOrder := slices.Clone(order)
	order = order[:0]
	recMu.Unlock()

	if len(gotOrder) != 2 || gotOrder[0] != "A" || gotOrder[1] != "B" {
		t.Fatalf("first event delivery order = %v, want [A B] (registration order, both observers)", gotOrder)
	}

	removeA()

	injectLine(t, peerEnd,
		`{"id":"server-2","method":"interaction/requestCredentialCipher",`+
			`"params":{"requestId":"req-2","operation":"decrypt","value":"enc:v2:os:c2VjcmV0"}}`)

	time.Sleep(deadline + graceWindow)

	recMu.Lock()
	gotAfterRemove := slices.Clone(order)
	recMu.Unlock()

	if len(gotAfterRemove) != 1 || gotAfterRemove[0] != "B" {
		t.Fatalf("post-removal delivery = %v, want [B] only (A was removed)", gotAfterRemove)
	}
}
