package fake_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// Golden corpus replay (03-03 Task 2): every scenario in testdata/scenarios/
// is driven through a REAL zcode.Conn over io.Pipe (the codec import is the
// point — the fake is exercised against the same codec the wrapper runs) and
// the direction-tagged transcript is compared byte-for-byte to its recorded
// golden fixture. Fixed synthetic ids keep replays byte-deterministic (D-01,
// RESEARCH Open Question 3).

// Harness timing knobs. Exchanges complete in milliseconds; the timeouts
// fail fast, and the one quiet window (stopSettle) bounds a fire-and-forget
// flush rather than racing wall clocks (the 03-02 bounded-quiet-window
// pattern).
const (
	exchangeTimeout   = 10 * time.Second
	noticeWaitTimeout = 5 * time.Second

	// stopSettle lets a fire-and-forget bypass frame (and the frames that
	// follow it) reach the fake's scanner before the host closes stdin —
	// io.Pipe writes complete in microseconds once the reader parks.
	stopSettle = 100 * time.Millisecond

	// messagePolls fixes the session/messages poll count for the
	// completion-quiet-window row so the transcript stays byte-deterministic.
	messagePolls = 3

	// oversizedLineBytes exceeds the 64 KiB bufio.Scanner default the dialect
	// warns about (PITFALL 7); both fake and codec buffers are raised, so the
	// line is recorded as undecodable and the stream continues.
	oversizedLineBytes = 70000

	// wireKeySessionID is the dialect's camelCase session-id params key
	// (replayed wire payloads keep the capture-verbatim spelling).
	wireKeySessionID = "sessionId"
)

// errZeroFrameExchange is the exit-0-with-zero-lines failure signature
// (dialect §1 EOF facts): an exchange that ends with zero recorded frames is
// a FAILURE, never an empty-but-successful result.
var errZeroFrameExchange = errors.New("exchange recorded zero frames: " +
	"exit-0-with-zero-lines is the EOF failure signature, never success")

// checkExchangeRecorded enforces the zero-frame failure signature on a
// recorded transcript.
func checkExchangeRecorded(tr fake.Transcript) error {
	if len(tr.Lines) == 0 {
		return errZeroFrameExchange
	}

	return nil
}

// driverFunc drives the host side of one scripted exchange; test failures
// go through the driver's t (the happyTurnHarness pattern — no *testing.T
// parameter, so the helper carries its harness instead). The replay harness
// closes the conn and joins fake.Serve afterwards.
type driverFunc func(ctx context.Context, d *exchangeDriver)

// exchangeDriver drives the host side through the zcode.Conn and carries the
// raw host pipe end for the envelope negatives (lines that must NOT pass
// through the codec, dialect §1 -32700/-32600 territory).
type exchangeDriver struct {
	t       *testing.T
	conn    *zcode.Conn
	host    *pipeEnd
	notices chan zcode.Envelope
}

// newExchangeDriver registers a conn notification observer wired to a
// buffered channel — registered immediately after NewConn, before any frame
// flows (the Conn hook contract).
func newExchangeDriver(t *testing.T, conn *zcode.Conn, host *pipeEnd) *exchangeDriver {
	t.Helper()

	d := &exchangeDriver{
		t:       t,
		conn:    conn,
		host:    host,
		notices: make(chan zcode.Envelope, 64),
	}

	conn.AddObserver(func(env zcode.Envelope) {
		d.notices <- env
	})

	return d
}

// call issues one request whose scripted wire-error reply the caller asserts
// on; only transport failures fail the exchange here.
func (d *exchangeDriver) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	d.t.Helper()

	return d.conn.Call(ctx, method, params)
}

// rawLine writes one raw host→fake wire line, bypassing the conn writer.
// io.Pipe's synchronous write returns once the fake's scanner consumed the
// line, pinning the transcript order deterministically.
func (d *exchangeDriver) rawLine(line string) {
	d.t.Helper()

	if _, err := d.host.Write([]byte(line + "\n")); err != nil {
		d.t.Fatalf("raw write: %v", err)
	}
}

// waitStatePatch waits for the next state.updated notification whose
// patch.status equals want (the wire-level sync signal for cancel flows).
func (d *exchangeDriver) waitStatePatch(status string) {
	d.t.Helper()

	deadline := time.After(noticeWaitTimeout)

	for {
		select {
		case env := <-d.notices:
			if env.Method == zcode.MethodStateUpdated && patchStatus(env) == status {
				return
			}
		case <-deadline:
			d.t.Fatalf("no state.updated patch with status %q within %v", status, noticeWaitTimeout)
		}
	}
}

// patchStatus digs patch.status out of a state.updated params payload.
func patchStatus(env zcode.Envelope) string {
	var params struct {
		Patch map[string]any `json:"patch"`
	}

	if err := json.Unmarshal(env.Params, &params); err != nil {
		return ""
	}

	s, _ := params.Patch["status"].(string)

	return s
}

// goldenDrivers maps every corpus scenario to its host-side driver. A corpus
// entry without a registered driver fails loud (TestGolden): adding a
// scenario means adding its replay driver — the corpus must stay replayable.
func goldenDrivers() map[string]driverFunc {
	return map[string]driverFunc{
		"happy-turn":                 driveHappyTurnCorpus,
		"cancel-mid-turn":            driveCancelMidTurn,
		"error-32010":                driveError32010,
		"unknown-host-method":        driveUnknownHostMethod,
		"runtime-preferences":        driveRuntimePreferences,
		"completion-idle-patch":      driveCompletionIdlePatch,
		"completion-quiet-window":    driveCompletionQuietWindow,
		"eof-drain":                  driveEOFDrain,
		"envelope-negatives":         driveEnvelopeNegatives,
		"credential-cipher-scripted": driveCredentialCipherScripted,
		"setmodel-followup":          driveSetModelFollowup,
		"part-delta-streaming":       drivePartDeltaStreaming,
		"no-reasoning":               driveNoReasoning,
	}
}

// TestGolden replays EVERY scenario in the corpus against a zcode.Conn over
// io.Pipe and compares each transcript to its golden fixture — the fake IS
// the protocol spec made executable (D-01). Drift, a missing fixture, or a
// driver-less scenario FAILS; -update re-records.
func TestGolden(t *testing.T) {
	entries, err := os.ReadDir("testdata/scenarios")
	if err != nil {
		t.Fatalf("read scenario corpus: %v", err)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")

		t.Run(name, func(t *testing.T) {
			drive, ok := goldenDrivers()[name]
			if !ok {
				t.Fatalf("scenario %s has no replay driver — every corpus entry must be replayable (fail-loud)", name)
			}

			tr := replayScenario(t, name, drive)
			fake.CompareGolden(t, "testdata/golden/"+name+".golden", goldenBytes(tr))
		})
	}
}

// replayScenario loads the corpus scenario, drives the scripted exchange,
// and enforces the zero-frame failure signature on the transcript.
func replayScenario(t *testing.T, name string, drive driverFunc) fake.Transcript {
	t.Helper()

	sc, err := fake.LoadScenario("testdata/scenarios/" + name + ".json")
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}

	tr := runExchange(t, sc, drive)

	if err := checkExchangeRecorded(tr); err != nil {
		t.Fatalf("scenario %s: %v", name, err)
	}

	return tr
}

// runExchange drives one scripted exchange: fake.Serve on one io.Pipe end, a
// fresh zcode.Conn on the other, the driver's scripted traffic, then conn
// close — the fake sees stdin EOF and drains (dialect §1 EOF facts). Serve
// must return clean; a run error fails the exchange.
func runExchange(t *testing.T, sc *fake.Scenario, drive driverFunc) fake.Transcript {
	t.Helper()

	hostEnd, fakeEnd := roundTripPipes()
	t.Cleanup(func() { _ = fakeEnd.Close() })

	serveDone := startFake(fakeEnd, sc)

	conn := zcode.NewConn(hostEnd)
	t.Cleanup(func() { _ = conn.Close() })

	d := newExchangeDriver(t, conn, hostEnd)

	ctx, cancel := context.WithTimeout(context.Background(), exchangeTimeout)
	defer cancel()

	drive(ctx, d)

	if err := conn.Close(); err != nil {
		t.Logf("conn close: %v", err)
	}

	served := waitServed(t, serveDone)

	if served.err != nil {
		t.Fatalf("fake.Serve: %v", served.err)
	}

	return served.transcript
}

// TestEOFSignature pins the harness failure signature (dialect §1 EOF facts,
// probe guard app-server-probe.mjs:661-673): an exchange that ends with zero
// recorded frames reports FAILURE — never an empty-but-successful result.
// Proven both at the helper level and against a real no-traffic exchange.
func TestEOFSignature(t *testing.T) {
	if err := checkExchangeRecorded(fake.Transcript{}); err == nil {
		t.Fatal("a zero-frame transcript must report failure, never success")
	}

	sc, err := fake.LoadScenario("testdata/scenarios/happy-turn.json")
	if err != nil {
		t.Fatalf("load scenario: %v", err)
	}

	tr := runExchange(t, sc, func(ctx context.Context, d *exchangeDriver) {
		// No traffic: stdin closes immediately — the exit-0-with-zero-lines
		// signature the probe guard treats as failure, never success.
		_ = ctx
		_ = d
	})

	if len(tr.Lines) != 0 {
		t.Fatalf("no-traffic exchange recorded %d frames, want 0", len(tr.Lines))
	}

	err = checkExchangeRecorded(tr)
	if err == nil {
		t.Fatal("the harness must report FAILURE on a zero-frame exchange, never success")
	}

	if !errors.Is(err, errZeroFrameExchange) {
		t.Fatalf("zero-frame failure must be the EOF-signature sentinel, got: %v", err)
	}
}

// driveHappyTurnCorpus: create (answered via the runtime-preferences reverse
// exchange) then send — the exact call sequence the tracer round-trip drives,
// so corpus row 1's recorded golden stays stable across both tests.
func driveHappyTurnCorpus(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)
	h.send(ctx)
}

// driveCancelMidTurn: the front preamble (plan 04-06 — the front tier
// cannot reach send without a session: create, subscribe, setModel copied
// verbatim from the setmodel-followup row), then send ack, then
// session/stop on the bypass lane (D-02); the stop rule's cancelled patch
// doubles as the wire-level sync signal that the fake processed the stop
// frame before the exchange closes.
func driveCancelMidTurn(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)

	subParams := map[string]any{
		wireKeySessionID: fixSessionID,
		"deliveryKind":   "desktop-continuous",
	}

	if _, err := d.call(ctx, zcode.MethodSessionSubscribe, subParams); err != nil {
		d.t.Fatalf("session/subscribe: %v", err)
	}

	setModelParams := map[string]any{
		wireKeySessionID: fixSessionID,
		"model": map[string]any{
			"providerId": "zai-api",
			"modelId":    "GLM-5.3-Flash",
			"options":    map[string]any{"reasoningLevel": "high"},
		},
		"persistAsWorkspaceLastUsed": false,
	}

	if _, err := d.call(ctx, zcode.MethodSessionSetModel, setModelParams); err != nil {
		d.t.Fatalf("session/setModel: %v", err)
	}

	h.send(ctx)

	if err := d.conn.Stop(fixSessionID); err != nil {
		d.t.Fatalf("session/stop: %v", err)
	}

	d.waitStatePatch("cancelled")

	// Bounded quiet window: the cancelled patch precedes the stop ack on the
	// wire by microseconds; settle so the ack is always recorded before the
	// host closes stdin.
	time.Sleep(stopSettle)
}

// driveError32010: the first send is acked (its rule's times budget spent);
// the second send while the turn runs is answered -32010, the dialect §6
// concurrency guard (bundle literal message).
func driveError32010(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)

	params := map[string]any{
		wireKeySessionID: fixSessionID,
		"content":        "Reply with exactly: OK",
	}

	_, err := d.call(ctx, zcode.MethodSessionSend, params)

	var callErr *zcode.CallError
	if !errors.As(err, &callErr) {
		d.t.Fatalf("second session/send must be refused with a wire error, got %v", err)
	}

	if callErr.Err.Code != zcode.CodeConcurrency {
		d.t.Fatalf("second session/send error code = %d, want %d (-32010)", callErr.Err.Code, zcode.CodeConcurrency)
	}
}

// driveUnknownHostMethod: create; the fake's reverse
// interaction/requestOfficialMcpAuthHeaders (dialect §3, capture-verbatim
// params shape with fixed synthetic workspace values) is answered -32601 by
// the conn's router and the flow continues to the create reply.
func driveUnknownHostMethod(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)
}

// driveRuntimePreferences: create fires the runtime-materialization scope
// reverse and send fires the user-execution scope — both answered with the
// minimal defaults (EXP-WORKSPACEKEY-01.jsonl:8-9, EXP-TURN-01.jsonl:33-34).
func driveRuntimePreferences(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)
	h.send(ctx)
}

// driveCompletionIdlePatch: send; no terminal event — the state.updated
// patch whose patch.status leaves "running" is the completion signal under
// test (dialect §5 reliability column: do not rely on it alone).
func driveCompletionIdlePatch(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}

// driveCompletionQuietWindow: send, then a fixed count of session/messages
// polls the fake answers with stable content — the poll-watcher settling
// surface (03-05/03-08 key link: neither terminal event nor idle patch
// arrives; the quiet window is the proven fallback).
func driveCompletionQuietWindow(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)

	pollParams := map[string]any{wireKeySessionID: fixSessionID}

	for i := range messagePolls {
		if _, err := d.call(ctx, zcode.MethodSessionMessages, pollParams); err != nil {
			d.t.Fatalf("session/messages poll %d: %v", i+1, err)
		}
	}
}

// driveEofDrain: a request issued just before stdin EOF is never answered
// (dialect §1 — the drain window beats request processing): the bypass stop
// goes out, the exchange closes before its delayed reply, and the fake
// records the in-flight reply honestly as unanswered on a clean run.
func driveEOFDrain(ctx context.Context, d *exchangeDriver) {
	if err := d.conn.Stop(fixSessionID); err != nil {
		d.t.Fatalf("session/stop: %v", err)
	}

	// Bounded quiet window: let the bypass frame reach the fake's scanner
	// before stdin closes, so the unanswered request is always recorded.
	time.Sleep(stopSettle)
}

// driveEnvelopeNegatives: a malformed JSON line and an oversized line (above
// the 64 KiB bufio default) interleave with valid traffic — the stream
// continues and the valid frames flow (the codec-side analog: conn_test.go
// TestLineBufferMalformedLineBetweenValidLines).
func driveEnvelopeNegatives(ctx context.Context, d *exchangeDriver) {
	d.rawLine("{definitely not json")
	d.rawLine(strings.Repeat("x", oversizedLineBytes))

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)
	h.send(ctx)
}

// driveCredentialCipherScripted: send; the fake's mid-turn reverse
// interaction/requestCredentialCipher (EXP-TURN-01.jsonl:50 params shape,
// re-serialized with a fixed synthetic value) gets NO answer — the conn
// router tracks it without replying (D-05) — the timeout elapses and the
// exchange records the expiry honestly. The scenario declares no
// expect_answer: the timeout path is the fixture's point.
func driveCredentialCipherScripted(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}

// driveSetModelFollowup: the front-tier session/new contract replayed at the
// back tier (plan 04-02 corpus row the tracer rides) — bare create (no model
// object: create drops options.reasoningLevel, PITFALL 4), subscribe (the
// session/event push gate, dialect §4), the setModel follow-up carrying
// model.options.reasoningLevel and persistAsWorkspaceLastUsed:false (D-03;
// Open Question 3 resolution), then send whose scripted stream emits the
// two part.delta text rows the prompt leg consumes. The golden
// byte-asserts the setModel request payload verbatim — the wire acceptance
// of the scripted rule is the golden's job, not the scenario's.
func driveSetModelFollowup(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)

	subParams := map[string]any{
		wireKeySessionID: fixSessionID,
		"deliveryKind":   "desktop-continuous",
	}

	if _, err := d.call(ctx, zcode.MethodSessionSubscribe, subParams); err != nil {
		d.t.Fatalf("session/subscribe: %v", err)
	}

	setModelParams := map[string]any{
		wireKeySessionID: fixSessionID,
		"model": map[string]any{
			"providerId": "zai-api",
			"modelId":    "GLM-5.3-Flash",
			"options":    map[string]any{"reasoningLevel": "high"},
		},
		"persistAsWorkspaceLastUsed": false,
	}

	if _, err := d.call(ctx, zcode.MethodSessionSetModel, setModelParams); err != nil {
		d.t.Fatalf("session/setModel: %v", err)
	}

	h.send(ctx)
}

// driveFrontPreamble drives the front-tier session/new contract calls
// verbatim (plan 04-04: the part-delta-streaming / no-reasoning rows ride
// the identical preamble the cancel-mid-turn row byte-locked in 04-06) —
// bare create, subscribe (the session/event push gate), the setModel
// follow-up with the corpus-locked synthetic selection.
func driveFrontPreamble(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)

	subParams := map[string]any{
		wireKeySessionID: fixSessionID,
		"deliveryKind":   "desktop-continuous",
	}

	if _, err := d.call(ctx, zcode.MethodSessionSubscribe, subParams); err != nil {
		d.t.Fatalf("session/subscribe: %v", err)
	}

	setModelParams := map[string]any{
		wireKeySessionID: fixSessionID,
		"model": map[string]any{
			"providerId": "zai-api",
			"modelId":    "GLM-5.3-Flash",
			"options":    map[string]any{"reasoningLevel": "high"},
		},
		"persistAsWorkspaceLastUsed": false,
	}

	if _, err := d.call(ctx, zcode.MethodSessionSetModel, setModelParams); err != nil {
		d.t.Fatalf("session/setModel: %v", err)
	}
}

// drivePartDeltaStreaming: the CORE-04 positive corpus row (plan 04-04) —
// the front preamble, then send whose scripted stream interleaves text and
// reasoning part.delta rows (text "Answer:", reasoning "Thinking about it",
// text " done") before turn.completed. The golden byte-locks the
// interleaved emit order the front's tap consumes in arrival order.
func drivePartDeltaStreaming(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}

// driveNoReasoning: the CORE-04 negative corpus row (plan 04-04) — the
// front preamble, then send whose scripted stream carries TEXT deltas only
// (zero field-reasoning envelopes): a client-side thought chunk here would
// be synthesized, never backend-sourced.
func driveNoReasoning(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}
