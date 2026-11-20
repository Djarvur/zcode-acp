package fake_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
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

	// reannounceAnswerDelay is how long the OnRequest stub waits before
	// answering the permission reverse: long enough that both scripted
	// re-announce envelopes (interval 40 ms, count 2) are on the wire first —
	// they join the same pending ask and earn no reply — but well inside the
	// reverse's 500 ms timeout window.
	reannounceAnswerDelay = 300 * time.Millisecond

	// reannounceSettle lets the re-announce envelopes' honest timeout notes
	// (interval 40 ms + 500 ms timeout ≈ 580 ms after the reverse fires) land
	// before stdin closes — the deterministic-golden bounded quiet window
	// (stopSettle pattern).
	reannounceSettle = 650 * time.Millisecond

	// requestPermissionMethod is the dialect's interaction/requestPermission
	// method string (the corpus row's scripted reverse).
	requestPermissionMethod = "interaction/requestPermission"

	// reannounceRegisterID is the deterministic wire id of the permission
	// reverse's first envelope (the fake-N counter: fake-1 goes to the
	// create rule's runtime-preferences reverse); fake-3/fake-4 are its two
	// re-announces.
	reannounceRegisterID = "fake-2"

	// reannounceJoinID1/reannounceJoinID2 are the deterministic wire ids of
	// the two re-announce envelopes.
	reannounceJoinID1 = "fake-3"
	reannounceJoinID2 = "fake-4"
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
		"send-refused-32010":         driveSendRefused32010,
		"complete-then-cancel":       driveCompleteThenCancel,
		"reannounce-dedupe":          driveReannounceDedupe,
	}
}

// goldenPostAsserts maps corpus rows to post-replay transcript assertions —
// semantic locks that ride the recorded conversation beyond the byte-golden
// (the fake records both directions, so reply addressing is assertable).
func goldenPostAsserts() map[string]func(t *testing.T, tr fake.Transcript) {
	return map[string]func(t *testing.T, tr fake.Transcript){
		"reannounce-dedupe": assertReannounceDedupeTranscript,
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

			if post, ok := goldenPostAsserts()[name]; ok {
				post(t, tr)
			}

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
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
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
		wireKeyContent:   sendContent,
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

// driveCompletionQuietWindow: the front preamble (plan 04-04 Rule 3 — the
// front-tier ordering lock rides this row and the front cannot open a
// session without it; rules copied verbatim from the setmodel-followup
// preamble), then send, then a fixed count of session/messages polls the
// fake answers with stable content — the poll-watcher settling surface
// (03-05/03-08 key link: neither terminal event nor idle patch arrives;
// the quiet window is the proven fallback).
func driveCompletionQuietWindow(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

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
// of the scripted rule is the golden's job, not the scenario's. The
// preamble itself is the shared driveFrontPreamble helper (one source of
// truth for the identical front-tier preamble every corpus row locks).
func driveSetModelFollowup(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}

// driveFrontPreamble drives the front-tier session/new contract calls
// verbatim — bare create, subscribe (the session/event push gate), the
// setModel follow-up with the corpus-locked synthetic selection. It is the
// ONE source of truth for the identical front-tier preamble the corpus
// rows byte-lock (setmodel-followup and cancel-mid-turn since 04-06;
// part-delta-streaming and no-reasoning since 04-04; completion-quiet-window
// since 04-04's Rule-3 extension).
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

// driveSendRefused32010: the CORE-05 backend-refusal corpus row (plan 04-05)
// — the front preamble (copied verbatim from cancel-mid-turn; the front
// cannot open a session without it), then ONE session/send the fake refuses
// with the dialect §6 concurrency guard -32010 (times budget 1). The
// machine maps the numeric code to ErrTurnActive (message text never
// matched) and the front surfaces ONE invalid-request rejection — the wire
// shape the front-tier TestSecondPromptRejected locks.
func driveSendRefused32010(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	params := map[string]any{
		wireKeySessionID: fixSessionID,
		wireKeyContent:   sendContent,
	}

	_, err := d.call(ctx, zcode.MethodSessionSend, params)

	var callErr *zcode.CallError
	if !errors.As(err, &callErr) {
		d.t.Fatalf("the refused session/send must answer a wire error, got %v", err)
	}

	if callErr.Err.Code != zcode.CodeConcurrency {
		d.t.Fatalf("session/send error code = %d, want %d (-32010)", callErr.Err.Code, zcode.CodeConcurrency)
	}
}

// driveCompleteThenCancel: the binary-E2E corpus row (plan 04-05) — the
// front preamble, then the FIRST send completing with its scripted deltas
// (times budget 1: the loop-1 end_turn turn), then a SECOND send whose
// stream stays active (the slow delta, no terminal), then the bypass
// session/stop whose cancelled patch is the observed stop outcome — the two
// prompt loops the built zcode-acp binary drives: end_turn, then cancelled.
func driveCompleteThenCancel(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx) // rule 1 (times 1): deltas + turn.completed — loop 1 end_turn
	h.send(ctx) // rule 2 (unlimited): slow delta, no terminal — loop 2 in flight

	if err := d.conn.Stop(fixSessionID); err != nil {
		d.t.Fatalf("session/stop: %v", err)
	}

	d.waitStatePatch("cancelled")

	// Bounded quiet window: the cancelled patch precedes the stop ack on the
	// wire by microseconds; settle so the ack is always recorded before the
	// host closes stdin.
	time.Sleep(stopSettle)
}

// driveReannounceDedupe: the CORE-06 dedupe corpus row (plan 05-01) — the
// create framing (its runtime-preferences reverse falls through the stub to
// the conn's built-in router), then send whose rule fires ONE
// interaction/requestPermission reverse carrying params.requestId perm-1
// with reannounce {count:2, interval_ms:40}. The OnRequest stub — a
// scripted in-test double, the real bridge being plan 05-02 — answers the
// permission reverse EXACTLY ONCE with the allow decision (the first
// envelope; first-reply-wins) after a fixed delay long enough that both
// re-announce envelopes are on the wire first. Join envelopes are HELD
// without framing anything until the conn closes: the wire must carry no
// reply for a re-announce (a reply envelope to a still-pending set would
// reject it server-side — 05-RESEARCH.md Pattern 2), and the fake's honest
// timeout notes are the fixture's point. The bounded settle window then
// lets those notes land before stdin closes (deterministic golden).
func driveReannounceDedupe(ctx context.Context, d *exchangeDriver) {
	var (
		reqMu    sync.Mutex
		answered bool
	)

	// Hook contract: assigned before any frame flows — the first server
	// frame only moves when create() below runs.
	d.conn.OnRequest = func(reqCtx context.Context, env zcode.Envelope) (any, *zcode.WireError, bool) {
		if env.Method != requestPermissionMethod {
			return nil, nil, false // fall through to the built-in router
		}

		reqMu.Lock()

		if answered {
			reqMu.Unlock()

			// Join envelope: hold without framing — releasing early would
			// let the conn write a reply envelope to a re-announce id.
			<-reqCtx.Done()

			return nil, nil, true
		}

		answered = true

		reqMu.Unlock()

		time.Sleep(reannounceAnswerDelay)

		return map[string]any{"decision": "allow", "reason": "Approved once"}, nil, true
	}

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)
	h.send(ctx)

	time.Sleep(reannounceSettle)
}

// assertReannounceDedupeTranscript asserts the dedupe semantics on the
// recorded wire conversation (the fake records both directions; the
// fake-N ids are deterministic): exactly one reply envelope exists for the
// reverse set, addressed to the registering wire id — never to a
// re-announce id (first-reply-wins; Pitfall 1's redundant-bytes window).
// Reply envelopes travel host→fake (DirIn); a reply envelope carries a
// result or error payload — bare id-echo frames are not replies
// (dialect §1).
func assertReannounceDedupeTranscript(t *testing.T, tr fake.Transcript) {
	t.Helper()

	replies := 0

	for _, line := range tr.Lines {
		if line.Dir != fake.DirIn {
			continue
		}

		probe := probeLine(line.JSON)
		if probe == nil {
			continue
		}

		id, _ := probe["id"].(string)

		if _, hasMethod := probe["method"]; hasMethod || id == "" {
			continue
		}

		_, hasPayload := probe["result"]
		if _, hasErr := probe["error"]; !hasPayload && !hasErr {
			continue
		}

		switch id {
		case reannounceRegisterID:
			replies++
		case reannounceJoinID1, reannounceJoinID2:
			t.Fatalf("re-announce envelope %s earned a reply (first-reply-wins violated): %s", id, line.JSON)
		}
	}

	if replies != 1 {
		t.Fatalf("permission reverse set earned %d reply envelopes, want exactly 1 addressed to %s",
			replies, reannounceRegisterID)
	}
}
