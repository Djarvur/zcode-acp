package interact_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The D-01 Responder contract locks (plan 05-01 tracer, CORE-06): staged
// asserts over a REAL zcode.Conn on io.Pipe with OnRequest assigned to
// responder.Serve BEFORE any frame flows (the hook contract), plus
// dedicated expiry and claim-race tests. Every assertion lands on recorded
// wire artifacts (conn replies, RouterEvent payloads, fake transcript
// lines) — never on internals.

// Test timing knobs: exchanges complete in milliseconds; the timeouts fail
// fast (the happy-turn harness pattern).
const (
	contractTimeout = 10 * time.Second

	// corpusScenarioPath is the reannounce-dedupe corpus row (Task 1) —
	// the end-to-end dedupe leg of this tracer replays it with the
	// Responder wired.
	corpusScenarioPath = "../zcode/fake/testdata/scenarios/reannounce-dedupe.json"

	// corpusAnswerDelay is how long the scripted forwarder waits before
	// answering the permission ask: long enough that both scripted
	// re-announces (interval 40 ms, count 2) land while the dedupe entry is
	// pending (they join and stay silent), well inside the reverse's
	// 500 ms timeout window.
	corpusAnswerDelay = 200 * time.Millisecond

	// The corpus row's deterministic fake-N wire ids (the fake-N counter:
	// fake-1 goes to the create rule's runtime-preferences reverse).
	corpusRegisterID = "fake-2"
	corpusJoinID1    = "fake-3"
	corpusJoinID2    = "fake-4"

	// migratedRuntimePrefsJSON is the migrated static answer byte-for-byte
	// — the same map literal conn.go:384-387 marshals (codec map keys are
	// sorted), so the bytes are the byte-identity proof.
	migratedRuntimePrefsJSON = `{"memoryEnabled":false,"nativeSearchEnhancementsEnabled":false}`
)

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

// pipePair wires two io.Pipe half-pairs (the roundtrip_test crossing):
// hostEnd drives a zcode.Conn, peerEnd is the app-server side (the test or
// fake.Serve). Each io.Pipe call makes ONE direction: host→peer and
// peer→host.
func pipePair(t *testing.T) (hostEnd, peerEnd *pipeEnd) {
	t.Helper()

	toFakeR, toFakeW := io.Pipe() // host writes, peer reads
	toHostR, toHostW := io.Pipe() // peer writes, host reads

	return &pipeEnd{r: toHostR, w: toFakeW}, &pipeEnd{r: toFakeR, w: toHostW}
}

// peerServer is the test-side app-server end for the staged conn tiers:
// it writes request envelopes and reads the conn's reply lines.
type peerServer struct {
	end *pipeEnd
	br  *bufio.Reader
}

// newPeerServer wraps a peer pipe end.
func newPeerServer(end *pipeEnd) *peerServer {
	return &peerServer{end: end, br: bufio.NewReader(end)}
}

// sendRequest writes one request envelope and returns the reply envelope
// the conn writes back.
func (p *peerServer) sendRequest(t *testing.T, env zcode.Envelope) zcode.Envelope {
	t.Helper()

	line, err := zcode.Encode(&env)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}

	if _, err := p.end.w.Write(line); err != nil {
		t.Fatalf("write request: %v", err)
	}

	replyLine, err := p.br.ReadString('\n')
	if err != nil {
		t.Fatalf("read reply line: %v", err)
	}

	reply, err := zcode.Decode([]byte(replyLine))
	if err != nil {
		t.Fatalf("decode reply: %v", err)
	}

	return reply
}

// newResponderConn wires a real zcode.Conn against the test-side peer with
// resp.Serve assigned to OnRequest BEFORE any frame flows (the hook
// contract).
func newResponderConn(t *testing.T, resp *interact.Responder) (*zcode.Conn, *peerServer) {
	t.Helper()

	hostEnd, peerEnd := pipePair(t)
	t.Cleanup(func() { _ = peerEnd.Close() })

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	conn.OnRequest = resp.Serve

	return conn, newPeerServer(peerEnd)
}

// fakeServeResult carries the fake's transcript and error across the
// goroutine boundary (channel ordering replaces a mutex).
type fakeServeResult struct {
	transcript fake.Transcript
	err        error
}

// probeJSON best-effort decodes a transcript line; note lines decode nil.
func probeJSON(line string) map[string]any {
	var probe map[string]any
	if err := json.Unmarshal([]byte(line), &probe); err != nil {
		return nil
	}

	return probe
}

// TestResponderContract is the RED target: one test, staged asserts.
func TestResponderContract(t *testing.T) {
	contractCtx, cancel := context.WithTimeout(context.Background(), contractTimeout)
	defer cancel()

	// Stages 1-2 run over a real Conn with the conn's built-in router NOT
	// wired (OnRequest is set to the responder).
	defaultResp := interact.NewResponder()
	_, peer := newResponderConn(t, defaultResp)

	// Stage 1 — the unknown-method floor: interaction/requestFoo answers
	// -32601 naming the method (never a crash, never a fabricated success).
	reply := peer.sendRequest(t, zcode.Envelope{
		ID:     &zcode.ID{Str: "s-1"},
		Method: "interaction/requestFoo",
	})

	if reply.Error == nil || reply.Error.Code != zcode.CodeMethodNotFound {
		t.Fatalf("unknown interaction method must answer -32601, got %+v", reply.Error)
	}

	if reply.Error.Message != "unsupported host request: interaction/requestFoo" {
		t.Fatalf("unknown-method message = %q, want the verbatim unsupported-host-request floor", reply.Error.Message)
	}

	// Stage 2 — the follow-up (proving the process is alive and answering):
	// session/requestRuntimePreferences serves the migrated static answer
	// byte-identical to conn.go:384-387's marshaled bytes, from the policy
	// table.
	reply = peer.sendRequest(t, zcode.Envelope{
		ID:     &zcode.ID{Str: "s-2"},
		Method: zcode.MethodRequestRuntimePreferences,
		Params: json.RawMessage(`{"sessionId":"sess_fix1","scope":"user-execution"}`),
	})

	if reply.Error != nil {
		t.Fatalf("runtime-preferences ask errored: %+v", reply.Error)
	}

	if string(reply.Result) != migratedRuntimePrefsJSON {
		t.Fatalf("migrated runtime-prefs answer = %s, want byte-identical %s", reply.Result, migratedRuntimePrefsJSON)
	}

	// Stage 3 — re-announce dedupe end-to-end: the reannounce-dedupe corpus
	// row through a real Conn with the Responder wired and a scripted
	// forwarder answering allow (the real bridge is plan 05-02).
	var forwardCalls atomic.Int64

	dedupeResp := interact.NewResponder(interact.WithPermissionForwarder(
		func(_ context.Context, _ interact.Request) (interact.Answer, error) {
			forwardCalls.Add(1)

			time.Sleep(corpusAnswerDelay)

			return interact.Answer{Result: map[string]any{"decision": "allow", "reason": "Approved once"}}, nil
		},
	))

	sc, err := fake.LoadScenario(corpusScenarioPath)
	if err != nil {
		t.Fatalf("load corpus scenario: %v", err)
	}

	hostEnd, peerEnd := pipePair(t)
	t.Cleanup(func() { _ = peerEnd.Close() })

	serveDone := make(chan fakeServeResult, 1)

	go func() {
		tr, serveErr := fake.Serve(peerEnd, sc)
		serveDone <- fakeServeResult{transcript: tr, err: serveErr}
	}()

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	conn.OnRequest = dedupeResp.Serve // the hook contract: before any frame flows

	if _, err := conn.Call(contractCtx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{"workspacePath": "/tmp/happy-turn-ws", "workspaceKey": "/tmp/happy-turn-ws"},
	}); err != nil {
		t.Fatalf("corpus create: %v", err)
	}

	if _, err := conn.Call(contractCtx, zcode.MethodSessionSend, map[string]any{
		"sessionId": "sess_fix1",
		"content":   "Reply with exactly: OK",
	}); err != nil {
		t.Fatalf("corpus send: %v", err)
	}

	if err := conn.Close(); err != nil {
		t.Logf("conn close: %v", err)
	}

	var served fakeServeResult

	select {
	case served = <-serveDone:
	case <-time.After(contractTimeout):
		t.Fatal("fake.Serve did not return after stdin EOF")
	}

	// expect_answer satisfied: the fake's semantic compare accepted the one
	// reply (a mismatch or an unanswered registering envelope fails the
	// run).
	if served.err != nil {
		t.Fatalf("corpus replay: fake.Serve error (expect_answer must be satisfied): %v", served.err)
	}

	assertCorpusDedupe(t, served.transcript)

	if n := forwardCalls.Load(); n != 1 {
		t.Fatalf("permission forwarder ran %d times, want exactly 1 (first-reply-wins)", n)
	}

	// Adjacency probe — equal merges: the corpus row's re-announces joined
	// the one pending entry (asserted above: one forwarder run, zero
	// replies to the join ids). Adjacency probe — touching separates: two
	// reverses with DISTINCT requestIds through the same responder are
	// answered independently.
	for _, rid := range []string{"perm-a", "perm-b"} {
		res, werr, handled := dedupeResp.Serve(contractCtx, zcode.Envelope{
			ID:     &zcode.ID{Str: "adj-" + rid},
			Method: interact.MethodRequestPermission,
			Params: json.RawMessage(`{"requestId":"` + rid + `","sessionId":"sess_fix1"}`),
		})

		if !handled || werr != nil || res == nil {
			t.Fatalf("distinct-requestId reverse %s must be answered independently (handled=%v werr=%v res=%v)",
				rid, handled, werr, res)
		}
	}

	// Stage 4 — empty-input probe: a reverse whose params carry NO
	// requestId claims no dedupe entry and is answered per envelope, twice
	// in a row.
	var noReqCalls atomic.Int64

	emptyResp := interact.NewResponder(interact.WithPermissionForwarder(
		func(_ context.Context, _ interact.Request) (interact.Answer, error) {
			noReqCalls.Add(1)

			return interact.Answer{Result: map[string]any{"decision": "allow"}}, nil
		},
	))

	for i := range 2 {
		res, werr, handled := emptyResp.Serve(contractCtx, zcode.Envelope{
			ID:     &zcode.ID{Str: fmt.Sprintf("e-%d", i)},
			Method: interact.MethodRequestPermission,
			Params: json.RawMessage(`{"sessionId":"sess_fix1"}`),
		})

		if !handled || werr != nil || res == nil {
			t.Fatalf("no-requestId reverse #%d must be answered per envelope (handled=%v werr=%v res=%v)",
				i, handled, werr, res)
		}
	}

	if n := noReqCalls.Load(); n != 2 {
		t.Fatalf("no-requestId reverses ran the handler %d times, want 2 (per-envelope)", n)
	}

	if n := emptyResp.PendingRequests(); n != 0 {
		t.Fatalf("no-requestId reverses claimed %d dedupe entries, want 0 (empty-input probe)", n)
	}

	// Post-answer eviction: an answered requestId entry releases — the
	// bounded map's answer eviction (T-5-01).
	if _, werr, handled := emptyResp.Serve(contractCtx, zcode.Envelope{
		ID:     &zcode.ID{Str: "e-evict"},
		Method: interact.MethodRequestPermission,
		Params: json.RawMessage(`{"requestId":"perm-evict","sessionId":"sess_fix1"}`),
	}); !handled || werr != nil {
		t.Fatalf("requestId reverse must be answered (handled=%v werr=%v)", handled, werr)
	}

	if n := emptyResp.PendingRequests(); n != 0 {
		t.Fatalf("pending dedupe entries after answer = %d, want 0 (answer eviction)", n)
	}

	// Stage 7 — requestUserInput declines (the honest no-UI-channel
	// posture; elicitation deferred to Phase 7).
	res, werr, handled := defaultResp.Serve(contractCtx, zcode.Envelope{
		ID:     &zcode.ID{Str: "s-7"},
		Method: interact.MethodRequestUserInput,
		Params: json.RawMessage(`{"requestId":"ui-1","sessionId":"sess_fix1"}`),
	})

	if !handled || werr != nil {
		t.Fatalf("requestUserInput must be answered with a result (handled=%v werr=%v)", handled, werr)
	}

	decline, ok := res.(map[string]any)

	if !ok || decline["action"] != "decline" {
		t.Fatalf("requestUserInput result = %v, want {action:decline}", res)
	}

	// Stage 8 — interaction/requestOfficialMcpAuthHeaders: the
	// field-verified harmless -32601 default (EXP-SUBSCRIBE-01).
	_, werr, handled = defaultResp.Serve(contractCtx, zcode.Envelope{
		ID:     &zcode.ID{Str: "s-8"},
		Method: interact.MethodRequestOfficialMcpAuthHeaders,
	})

	if !handled || werr == nil || werr.Code != zcode.CodeMethodNotFound {
		t.Fatalf("requestOfficialMcpAuthHeaders must answer -32601 (handled=%v werr=%v)", handled, werr)
	}
}

// assertCorpusDedupe asserts the dedupe semantics on the recorded wire
// conversation: both re-announce envelopes are on the wire (same params,
// NEW wire ids — the schedule fired), exactly one reply envelope exists
// for the reverse set, addressed to the registering wire id — never to a
// join id (first-reply-wins; Pitfall 1's redundant-bytes window). A reply
// envelope carries a result or error payload — bare id-echo frames are
// not replies (dialect §1).
func assertCorpusDedupe(t *testing.T, tr fake.Transcript) {
	t.Helper()

	announced := map[string]int{corpusJoinID1: 0, corpusJoinID2: 0}
	replies := 0

	for _, line := range tr.Lines {
		probe := probeJSON(line.JSON)

		switch line.Dir {
		case fake.DirOut:
			if m, _ := probe["method"].(string); m == interact.MethodRequestPermission {
				if id, _ := probe["id"].(string); id == corpusJoinID1 || id == corpusJoinID2 {
					announced[id]++
				}
			}
		case fake.DirIn:
			id, _ := probe["id"].(string)

			if _, hasMethod := probe["method"]; hasMethod || id == "" {
				continue
			}

			_, hasPayload := probe["result"]
			_, hasErr := probe["error"]

			if !hasPayload && !hasErr {
				continue
			}

			switch id {
			case corpusRegisterID:
				replies++
			case corpusJoinID1, corpusJoinID2:
				t.Fatalf("re-announce envelope %s earned a reply (first-reply-wins violated): %s", id, line.JSON)
			}
		}
	}

	for id, n := range announced {
		if n != 1 {
			t.Fatalf("re-announce envelope %s recorded %d times, want exactly 1 (the schedule must fire)", id, n)
		}
	}

	if replies != 1 {
		t.Fatalf("permission reverse set earned %d reply envelopes, want exactly 1 addressed to %s",
			replies, corpusRegisterID)
	}
}

// TestResponderDeadlineExpiry: with a policy deadline shortened via the
// injectable knob, an unanswered handler expiry produces the method's
// expiry answer (deny for requestPermission; coded error for
// requestProviderRuntimeHeaders) and a RouterEvent recording the expiry —
// method/id key/duration only, never params (structural-logging rule).
func TestResponderDeadlineExpiry(t *testing.T) {
	events := make(chan zcode.RouterEvent, 8)

	resp := interact.NewResponder(
		interact.WithMethodDeadline(interact.MethodRequestPermission, 50*time.Millisecond),
		interact.WithPermissionForwarder(func(ctx context.Context, _ interact.Request) (interact.Answer, error) {
			<-ctx.Done() // never answers before the deadline

			return interact.Answer{}, nil
		}),
		interact.WithRouterEvent(func(ev zcode.RouterEvent) { events <- ev }),
	)

	ctx, cancel := context.WithTimeout(context.Background(), contractTimeout)
	defer cancel()

	res, werr, handled := resp.Serve(ctx, zcode.Envelope{
		ID:     &zcode.ID{Str: "exp-1"},
		Method: interact.MethodRequestPermission,
		Params: json.RawMessage(`{"requestId":"perm-exp","sessionId":"sess_fix1"}`),
	})

	if !handled || werr != nil {
		t.Fatalf("expiry must answer handled with a result (handled=%v werr=%v)", handled, werr)
	}

	deny, ok := res.(map[string]any)

	if !ok || deny["decision"] != "deny" {
		t.Fatalf("permission expiry answer = %v, want {decision:deny} (OQ-2 disposition)", res)
	}

	select {
	case ev := <-events:
		if ev.Kind != zcode.RouterDeadlineExpired ||
			ev.Method != interact.MethodRequestPermission ||
			ev.IDKey != "s:exp-1" ||
			ev.Deadline != 50*time.Millisecond ||
			!errors.Is(ev.Err, context.DeadlineExceeded) {
			t.Fatalf("expiry RouterEvent = %+v, want deadline-expired naming the method/id key/duration", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no RouterEvent recorded the expiry")
	}

	// The OQ-2 default: requestPermission's deadline default is 300 s,
	// settable via WithMethodDeadline (the 50 ms override above governed
	// the run).
	if interact.DefaultPermissionDeadline != 300*time.Second {
		t.Fatalf("DefaultPermissionDeadline = %v, want 300s (OQ-2 disposition)", interact.DefaultPermissionDeadline)
	}

	// Runtime-headers expiry answers the coded -32022 timeout error —
	// never the failure-shaped success arm (PITFALL 3).
	headers := interact.NewResponder(
		interact.WithMethodDeadline(interact.MethodRequestProviderRuntimeHeaders, 50*time.Millisecond),
		interact.WithRuntimeHeadersKeySource(func(ctx context.Context, _ interact.Request) (string, string, error) {
			<-ctx.Done()

			return "", "", ctx.Err()
		}),
	)

	_, werr, handled = headers.Serve(ctx, zcode.Envelope{
		ID:     &zcode.ID{Str: "exp-2"},
		Method: interact.MethodRequestProviderRuntimeHeaders,
		Params: json.RawMessage(`{"requestId":"rh-exp","sessionId":"sess_fix1"}`),
	})

	if !handled || werr == nil || werr.Code != zcode.CodeHostRequestTimedOut {
		t.Fatalf("runtime-headers expiry must answer the coded timeout error (handled=%v werr=%v)", handled, werr)
	}
}

// TestResponderDedupeRace fires the same (method, params.requestId)
// envelope concurrently at the responder: the handler runs exactly once
// (delete-under-lock claim) and exactly one reply payload is produced —
// -race clean.
func TestResponderDedupeRace(t *testing.T) {
	var runs atomic.Int64

	resp := interact.NewResponder(interact.WithPermissionForwarder(
		func(_ context.Context, _ interact.Request) (interact.Answer, error) {
			runs.Add(1)

			time.Sleep(20 * time.Millisecond) // widen the claim race window

			return interact.Answer{Result: map[string]any{"decision": "deny"}}, nil
		},
	))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const racers = 8

	type served struct {
		res     any
		werr    *zcode.WireError
		handled bool
	}

	results := make(chan served, racers)

	env := zcode.Envelope{
		ID:     &zcode.ID{Str: "race-1"},
		Method: interact.MethodRequestPermission,
		Params: json.RawMessage(`{"requestId":"perm-race","sessionId":"sess_fix1"}`),
	}

	for range racers {
		go func() {
			res, werr, handled := resp.Serve(ctx, env)
			results <- served{res: res, werr: werr, handled: handled}
		}()
	}

	// Mid-answer: the primary holds the one claimed entry.
	time.Sleep(5 * time.Millisecond)

	if n := resp.PendingRequests(); n != 1 {
		t.Fatalf("pending dedupe entries mid-answer = %d, want 1 (the claimed entry)", n)
	}

	// Post-answer: the entry released (the joiners parked it out).
	time.Sleep(200 * time.Millisecond)

	if n := runs.Load(); n != 1 {
		t.Fatalf("handler ran %d times for one (method, requestId), want exactly 1", n)
	}

	if n := resp.PendingRequests(); n != 0 {
		t.Fatalf("pending dedupe entries after answer = %d, want 0 (release on answer)", n)
	}

	cancel() // release the parked joiners

	replies := 0

	for range racers {
		select {
		case r := <-results:
			if r.handled && r.res != nil {
				replies++
			}
		case <-time.After(2 * time.Second):
			t.Fatal("a racer never returned after ctx cancellation")
		}
	}

	if replies != 1 {
		t.Fatalf("concurrent same-key envelopes produced %d replies, want exactly 1", replies)
	}
}
