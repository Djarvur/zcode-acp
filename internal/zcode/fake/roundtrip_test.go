package fake_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Fixed synthetic ids pin byte-deterministic goldens (RESEARCH Open Question
// 3, resolved): scrubbed captures remain seed documentation, never goldens.
const (
	fixSessionID = "sess_fix1"
	fixTurnID    = "turn_fix1"

	// happyTurnTimeout bounds the whole exchange; the GREEN run completes
	// in milliseconds and a RED run fails fast on its own assertion.
	happyTurnTimeout = 10 * time.Second

	// serveWaitTimeout bounds how long the test waits for fake.Serve to
	// return after the host closes its end of the pipe.
	serveWaitTimeout = 5 * time.Second

	// jsonRPCFramingKey must never appear on the wire (dialect §1: the
	// child rejects the JSON-RPC framing key with -32600).
	jsonRPCFramingKey = "jsonrpc"
)

func TestMain(m *testing.M) {
	flag.Bool(fake.UpdateFlagName, false, "rewrite golden fixtures from the current run")
	flag.Parse()

	os.Exit(m.Run())
}

// TestHappyTurnRoundTrip is the tracer RED target: a real zcode.Conn driven
// against the in-process fake over io.Pipe completes the happy-turn scenario
// end-to-end — create (answered via the runtime-preferences reverse
// request), send ack, and the scripted notifications in wire order.
func TestHappyTurnRoundTrip(t *testing.T) {
	res := runHappyTurn(t)

	assertCreateReply(t, res)
	assertSendAck(t, res)
	assertNotificationOrder(t, res.notices)
	assertTranscript(t, res.transcript)
}

// TestGoldenHappyTurn replays the happy-turn exchange against the recorded
// golden fixture — the protocol-spec-made-executable (D-01). Any frame drift
// or missing frame FAILS; -update re-records (never auto-creates).
func TestGoldenHappyTurn(t *testing.T) {
	res := runHappyTurn(t)

	fake.CompareGolden(t, "testdata/golden/happy-turn.golden", goldenBytes(res.transcript))
}

// goldenBytes renders a transcript in the golden fixture format: one
// direction tag, a tab, and the raw wire line per transcript entry.
func goldenBytes(tr fake.Transcript) []byte {
	var buf bytes.Buffer

	for _, line := range tr.Lines {
		buf.WriteString(string(line.Dir))
		buf.WriteByte('\t')
		buf.WriteString(line.JSON)
		buf.WriteByte('\n')
	}

	return buf.Bytes()
}

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

// roundTripPipes wires the two io.Pipe half-pairs as host and fake stream
// ends: frames on the pipe are the real boundary (D-01 mode A).
func roundTripPipes() (hostEnd, fakeEnd *pipeEnd) {
	toFakeR, toFakeW := io.Pipe()
	toHostR, toHostW := io.Pipe()

	hostEnd = &pipeEnd{r: toHostR, w: toFakeW}
	fakeEnd = &pipeEnd{r: toFakeR, w: toHostW}

	return hostEnd, fakeEnd
}

// serveResult carries the fake's transcript and error across the goroutine
// boundary (channel ordering replaces a mutex).
type serveResult struct {
	transcript fake.Transcript
	err        error
}

// happyTurnResult carries everything the round-trip assertions need.
type happyTurnResult struct {
	createRes  json.RawMessage
	sendRes    json.RawMessage
	notices    []zcode.Envelope
	transcript fake.Transcript
}

// runHappyTurn drives the full happy-turn exchange over io.Pipe: a fresh
// zcode.Conn against fake.Serve with the happy-turn scenario. It fails the
// test if any call errors; wire-order and transcript content are asserted
// by the callers.
func runHappyTurn(t *testing.T) happyTurnResult {
	t.Helper()

	sc, err := fake.LoadScenario("testdata/scenarios/happy-turn.json")
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}

	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	serveDone := startFake(fakeEnd, sc)

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	snapshotNotices := recordNotices(conn)

	ctx, cancel := context.WithTimeout(context.Background(), happyTurnTimeout)
	defer cancel()

	h := &happyTurnHarness{t: t, conn: conn}
	createRes := h.create(ctx)
	sendRes := h.send(ctx)
	notices := snapshotNotices()

	if err := conn.Close(); err != nil {
		t.Logf("conn close: %v", err)
	}

	served := waitServed(t, serveDone)

	if served.err != nil {
		t.Fatalf("fake.Serve: %v", served.err)
	}

	return happyTurnResult{
		createRes:  createRes,
		sendRes:    sendRes,
		notices:    notices,
		transcript: served.transcript,
	}
}

// startFake runs fake.Serve on a goroutine and returns its completion
// channel (channel ordering replaces a mutex for the transcript handoff).
func startFake(fakeEnd *pipeEnd, sc *fake.Scenario) <-chan serveResult {
	serveDone := make(chan serveResult, 1)

	go func() {
		tr, serveErr := fake.Serve(fakeEnd, sc)
		serveDone <- serveResult{transcript: tr, err: serveErr}
	}()

	return serveDone
}

// recordNotices hooks Conn.OnNotification to collect envelopes in wire
// order; the returned func snapshots the collected slice.
func recordNotices(conn *zcode.Conn) func() []zcode.Envelope {
	var (
		mu      sync.Mutex
		notices []zcode.Envelope
	)

	conn.OnNotification = func(env zcode.Envelope) {
		mu.Lock()
		defer mu.Unlock()

		notices = append(notices, env)
	}

	return func() []zcode.Envelope {
		mu.Lock()
		defer mu.Unlock()

		return slices.Clone(notices)
	}
}

// happyTurnHarness bundles the driver state for the scripted calls.
type happyTurnHarness struct {
	t    *testing.T
	conn *zcode.Conn
}

// create issues session/create with the workspace path/key pair (dialect
// §7; workspaceKey is a host-opaque nonEmptyString — EXP-WORKSPACEKEY-01).
func (h *happyTurnHarness) create(ctx context.Context) json.RawMessage {
	h.t.Helper()

	res, err := h.conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{
			"workspacePath": "/tmp/happy-turn-ws",
			"workspaceKey":  "/tmp/happy-turn-ws",
		},
	})
	if err != nil {
		h.t.Fatalf("session/create: %v", err)
	}

	return res
}

// send issues session/send for the fixed session.
func (h *happyTurnHarness) send(ctx context.Context) json.RawMessage {
	h.t.Helper()

	res, err := h.conn.Call(ctx, zcode.MethodSessionSend, map[string]any{
		"sessionId": fixSessionID,
		"content":   "Reply with exactly: OK",
	})
	if err != nil {
		h.t.Fatalf("session/send: %v", err)
	}

	return res
}

// waitServed waits for fake.Serve to return after the host closed its end.
func waitServed(t *testing.T, serveDone <-chan serveResult) serveResult {
	t.Helper()

	var served serveResult

	select {
	case served = <-serveDone:
	case <-time.After(serveWaitTimeout):
		t.Fatal("fake.Serve did not return after stdin EOF")
	}

	return served
}

// assertCreateReply checks the create result's fixed session id.
func assertCreateReply(t *testing.T, res happyTurnResult) {
	t.Helper()

	sessID := jsonPathString(t, res.createRes, "session", "sessionId")
	if sessID != fixSessionID {
		t.Fatalf("session/create sessionId = %q, want %q", sessID, fixSessionID)
	}
}

// assertSendAck checks the send ack shape (EXP-TURN-01.jsonl:36 verbatim
// keys: accepted, sessionId, stateRevision).
func assertSendAck(t *testing.T, res happyTurnResult) {
	t.Helper()

	var ack map[string]any
	if err := json.Unmarshal(res.sendRes, &ack); err != nil {
		t.Fatalf("decode send ack: %v", err)
	}

	accepted, ok := ack["accepted"].(bool)
	if !ok || !accepted {
		t.Fatalf("send ack accepted = %v, want true", ack["accepted"])
	}

	if sid, _ := ack["sessionId"].(string); sid != fixSessionID {
		t.Fatalf("send ack sessionId = %v, want %q", ack["sessionId"], fixSessionID)
	}

	rev, ok := ack["stateRevision"].(float64)
	if !ok || rev != 1 {
		t.Fatalf("send ack stateRevision = %v, want 1", ack["stateRevision"])
	}
}

// assertNotificationOrder checks the scripted notification stream on the
// hook, in wire order: lifecycle starting→ready, the state patch, then the
// turn.started / turn.completed session events carrying the fixed turn id.
func assertNotificationOrder(t *testing.T, notices []zcode.Envelope) {
	t.Helper()

	wantMethods := []string{
		zcode.MethodSessionLifecycleChanged, // starting
		zcode.MethodSessionLifecycleChanged, // ready
		zcode.MethodStateUpdated,            // running patch
		zcode.MethodSessionEvent,            // turn.started
		zcode.MethodSessionEvent,            // turn.completed
	}

	if len(notices) != len(wantMethods) {
		t.Fatalf("got %d notifications, want %d: %+v", len(notices), len(wantMethods), notices)
	}

	for i, env := range notices {
		if env.Method != wantMethods[i] {
			t.Fatalf("notification %d method = %q, want %q (wire order broken)", i, env.Method, wantMethods[i])
		}
	}

	if state := paramsKey(t, notices[0], "state"); state != "starting" {
		t.Fatalf("lifecycle[0] state = %q, want starting", state)
	}

	if state := paramsKey(t, notices[1], "state"); state != "ready" {
		t.Fatalf("lifecycle[1] state = %q, want ready", state)
	}

	started := paramsKey(t, notices[3], "type")

	if started != "turn.started" {
		t.Fatalf("session/event type = %q, want turn.started", started)
	}

	if turnID := paramsKey(t, notices[3], "turnId"); turnID != fixTurnID {
		t.Fatalf("turn.started turnId = %q, want %q", turnID, fixTurnID)
	}

	completed := paramsKey(t, notices[4], "type")

	if completed != "turn.completed" {
		t.Fatalf("session/event type = %q, want turn.completed", completed)
	}

	if turnID := paramsKey(t, notices[4], "turnId"); turnID != fixTurnID {
		t.Fatalf("turn.completed turnId = %q, want %q", turnID, fixTurnID)
	}
}

// assertTranscript checks the recorded wire conversation: non-empty, both
// sides of the runtime-preferences exchange present — the fake's reverse
// request travels fake→host (DirOut), the host's answer host→fake (DirIn) —
// and every host→server line valid JSON carrying no JSON-RPC framing key
// (dialect §1).
func assertTranscript(t *testing.T, tr fake.Transcript) {
	t.Helper()

	if len(tr.Lines) == 0 {
		t.Fatal("transcript is empty — nothing was recorded on the pipe")
	}

	sawReverseRequest := false
	sawReverseAnswer := false

	for _, line := range tr.Lines {
		if line.Dir == fake.DirIn {
			if strings.Contains(line.JSON, jsonRPCFramingKey) {
				t.Fatalf("host→server line carries the JSON-RPC framing key: %s", line.JSON)
			}

			var validity map[string]any
			if err := json.Unmarshal([]byte(line.JSON), &validity); err != nil {
				t.Fatalf("host→server line is not JSON: %v (%s)", err, line.JSON)
			}
		}

		probe := probeLine(line.JSON)

		if method, _ := probe["method"].(string); method == zcode.MethodRequestRuntimePreferences {
			sawReverseRequest = true

			if params, ok := probe["params"].(map[string]any); ok {
				if scope, _ := params["scope"].(string); scope != "runtime-materialization" {
					t.Fatalf("runtime-preferences scope = %q, want runtime-materialization", scope)
				}
			}
		}

		if res, ok := probe["result"].(map[string]any); ok {
			if _, has := res["memoryEnabled"]; has {
				sawReverseAnswer = true
			}
		}
	}

	if !sawReverseRequest || !sawReverseAnswer {
		t.Fatalf("runtime-preferences exchange missing from transcript (request=%v answer=%v)",
			sawReverseRequest, sawReverseAnswer)
	}
}

// probeLine best-effort decodes a transcript line for content assertions;
// note lines (prose) decode to nil.
func probeLine(line string) map[string]any {
	var probe map[string]any
	if err := json.Unmarshal([]byte(line), &probe); err != nil {
		return nil
	}

	return probe
}

// jsonPathString digs a nested string value out of a decoded result.
func jsonPathString(t *testing.T, raw json.RawMessage, path ...string) string {
	t.Helper()

	cur := decodeJSONAny(t, raw)

	for _, key := range path {
		obj, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("result path %v: %T is not an object", path, cur)
		}

		cur, ok = obj[key]
		if !ok {
			t.Fatalf("result path %v: key %q missing", path, key)
		}
	}

	s, ok := cur.(string)
	if !ok {
		t.Fatalf("result path %v: %T is not a string", path, cur)
	}

	return s
}

// paramsKey digs one top-level string key out of an envelope's params.
func paramsKey(t *testing.T, env zcode.Envelope, key string) string {
	t.Helper()

	if len(env.Params) == 0 {
		t.Fatalf("envelope %s has no params", env.Method)
	}

	obj, ok := decodeJSONAny(t, env.Params).(map[string]any)
	if !ok {
		t.Fatalf("envelope %s params is not an object", env.Method)
	}

	s, _ := obj[key].(string)

	return s
}

// decodeJSONAny unmarshals raw, failing the test on invalid JSON.
func decodeJSONAny(t *testing.T, raw json.RawMessage) any {
	t.Helper()

	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("decode json: %v (%s)", err, raw)
	}

	return v
}
