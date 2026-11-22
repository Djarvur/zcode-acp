package fake_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
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

	// wireKeyDeliveryKind is the subscribe gate's camelCase params key and
	// deliveryDesktopContinuous its record value (the session/event push
	// gate — dialect §4).
	wireKeyDeliveryKind = "deliveryKind"

	// deliveryDesktopContinuous is the subscribe delivery kind value.
	deliveryDesktopContinuous = "desktop-continuous"

	// reannounceAnswerDelay is how long the OnRequest stub waits before
	// answering the permission reverse: long enough that both scripted
	// re-announce envelopes (interval 40 ms, count 2) are on the wire first —
	// they join the same pending ask and earn no reply — but well inside the
	// reverse's 500 ms timeout window.
	reannounceAnswerDelay = 300 * time.Millisecond

	// permCancelSettle lets the cancel row's reverse reply and the send ack
	// land deterministically AFTER the scripted client's released answer and
	// BEFORE the driver drives the session cancel — every frame the golden
	// locks sits in a causally forced position.
	permCancelSettle = 150 * time.Millisecond

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

	// corpusAuthInstall is the synthetic absolute install the auth-apikey
	// corpus row pushes against. The revision formula hashes the resolved
	// PATH STRING (the file need not exist), so the fixed string keeps the
	// recorded push-envelope golden bytes deterministic across machines; the
	// formula itself is proven path-sensitive at the unit tier
	// (hostsvc.TestSnapshotEnvelope).
	corpusAuthBuiltinPath = "/zcode-acp-corpus-fixture/zcode-builtin.json"

	// corpusAuthBuiltinRev is the fixture's recorded builtin revision int
	// (the 3.18.2-recorded value — EXP-DISCOVERY-01 style probe-time read).
	corpusAuthBuiltinRev = 44

	// corpusAuthRevision is the injected fixed push revision (the corpus
	// fixture's synthetic value; the fake's static ack echoes it verbatim).
	corpusAuthRevision = "account-bridge:corpus-fix1"

	// syntheticFixtureKey/syntheticFixtureKeyID are the runtime-headers-answer
	// row's synthetic KeySource material — fixture data only, never a real
	// credential (T-5-15; the scan-secrets gate stays green).
	syntheticFixtureKey = "zc-synthetic-fixture-not-a-real-key"

	syntheticFixtureKeyID = "key_fixture_1"

	// authReadbackWait bounds the auth-apikey row's read-back Await (the
	// surfaces are already on the wire when the driver awaits).
	authReadbackWait = 5 * time.Second

	// lingeringLatchSettle is how long the lingering-latch driver waits out
	// the desktop's one-turn-at-a-time latch before the drained retry: the
	// scenario's 2000 ms latch window plus scheduling margin (≈ the real
	// drain's latch-plus-margin discipline). The sleep sits between
	// wire-gated steps, so the frame sequence stays byte-deterministic.
	lingeringLatchSettle = 2500 * time.Millisecond
)

// corpusAuthModels returns the personal zai-api models the auth-apikey
// row's registry surfaces must carry for the read-back verdict
// (providerId/modelId).
func corpusAuthModels() []string {
	return []string{"zai-api/GLM-5.3-Flash", "zai-api/GLM-5.3"}
}

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
		"permission-allow":           drivePermissionAllow,
		"permission-cancel":          drivePermissionCancel,
		"user-input-decline":         driveUserInputDecline,
		"tool-lifecycle":             driveToolLifecycle,
		"session-lifecycle":          driveSessionLifecycle,
		"newsession-orphan-close":    driveNewSessionOrphanClose,
		"auth-apikey":                driveAuthApikey,
		"runtime-headers-answer":     driveRuntimeHeadersAnswer,
		"full-cycle":                 driveFullCycle,
		"lingering-latch-32010":      driveLingeringLatch32010,
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
// Plan 05-01 re-points this row at the D-01 responder (D-01 migration): the
// responder's static answer serves both asks through the OnRequest seam —
// byte-identical to the built-in router's, which is why the recorded
// golden bytes are unchanged; the golden staying green IS the migration's
// byte-identity proof at the corpus tier. (The params carry no requestId,
// so the responder answers per envelope — the empty-input posture.)
func driveRuntimePreferences(ctx context.Context, d *exchangeDriver) {
	d.conn.OnRequest = interact.NewResponder().Serve // hook contract: before any frame flows

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
		wireKeySessionID:    fixSessionID,
		wireKeyDeliveryKind: deliveryDesktopContinuous,
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

// drivePermissionAllow: the TOOL-02 tracer corpus row (plan 05-02) — the
// front preamble (copied verbatim from setmodel-followup), then send whose
// rule emits one tool.updated scheduled event and fires the
// interaction/requestPermission reverse (requestId perm-1, the canonical
// allow_once/deny options with embedded responses, reannounce count 2
// interval 40) expecting the allow answer. The OnRequest hook carries the
// FULL composition the plan pins: the Responder whose PermissionForwarder
// is the real front bridge (acpfront bridge over the acpport sender bound
// at acpport construction), with a scripted SDK client (permissionClient)
// answering selected allow_once on the ACP leg — the client's decision
// rides the option's own embedded response back to the reverse. The
// scripted answer lands in milliseconds, so the re-announce schedule stops
// before its first envelope (the done close in runReannounce) — the
// golden byte-locks the answered round-trip: the reverse line, the reply
// carrying the option's embedded response bytes verbatim, tool.updated,
// turn.completed, and the send ack.
func drivePermissionAllow(ctx context.Context, d *exchangeDriver) {
	t := d.t

	// ACP leg pipes: the agent (acpport) side reads toAgent and writes
	// toClient; the scripted client reads toClient and writes toAgent.
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{toAgentR, toAgentW, toClientR, toClientW} {
			_ = c.Close()
		}
	})

	//nolint:contextcheck // the Front derives its own process-lifetime base
	// context by design (front.go New — the containedctx rule); the driver's
	// ctx bounds only the back-leg calls.
	front := acpfront.New(d.conn)

	// Hook contract: the responder (whose PermissionForwarder is the real
	// front bridge) is assigned before any frame flows — the first server
	// frame only moves when the preamble's create runs below.
	responder := interact.NewResponder(interact.WithPermissionForwarder(
		interact.NewPermissionBridge(front.PermissionForward()).Forward,
	))
	d.conn.OnRequest = responder.Serve

	client := &permissionClient{}

	// The construction order completes both bindings (sink + sender) before
	// any frame flows: the sender binds into the front's bridge here, the
	// client only ever sees the request_permission frames the bridge sends.
	// Pipe ownership: the agent connection writes the agent→client pipe
	// (toClientW) and reads the client→agent pipe (toAgentR); the client is
	// wired the mirror way (writes toAgentW, reads toClientR).
	//
	//nolint:dogsled // the front carries all three bindings; none is needed here
	_, _, _ = acpport.NewAgentSideConnection(acpport.NewAgent(front), toClientW, toAgentR)
	_ = acp.NewClientSideConnection(client, toAgentW, toClientR)

	driveFrontPreamble(ctx, d)

	// The send fires the row's tracer exchange: tool.updated, the
	// permission reverse (answered by the scripted client through the full
	// bridge chain), turn.completed, then the ack — the golden byte-locks
	// the whole answered round-trip.
	h := happyTurnHarness{t: t, conn: d.conn}
	h.send(ctx)
}

// drivePermissionCancel: the TOOL-02 cancel corpus row (plan 05-02) — the
// full composition (Responder + real front bridge + acpport sender over a
// real Front session), a REAL prompt turn through the front's machine, and
// a gated permissionClient scripted with the cancelled outcome arm. The
// driver releases the gate only after the ask reached the client, settles
// (the -32021 error reply and the send ack land — the reply rides the
// bypass lane), then drives the landed Front.Cancel: the stop-bypass
// session/stop fires while the machine resolves the cancelled patch — the
// CORE-05 cancel path the plan pins — and the prompt returns. The golden
// byte-locks the cancel flow: reverse, -32021 reply (expect_answer locks
// the code AND the bridge's message constant), send ack, stop, cancelled
// patch, stop ack — every frame causally ordered.
func drivePermissionCancel(ctx context.Context, d *exchangeDriver) {
	t := d.t

	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{toAgentR, toAgentW, toClientR, toClientW} {
			_ = c.Close()
		}
	})

	//nolint:contextcheck // the Front derives its own process-lifetime base
	// context by design (front.go New — the containedctx rule); the driver's
	// ctx bounds only the back-leg calls.
	front := acpfront.New(d.conn)

	responder := interact.NewResponder(interact.WithPermissionForwarder(
		interact.NewPermissionBridge(front.PermissionForward()).Forward,
	))
	d.conn.OnRequest = responder.Serve // hook contract: before any frame flows

	client := &permissionClient{gate: make(chan struct{}), cancel: true}

	// Pipe ownership mirrors drivePermissionAllow: the agent writes
	// toClientW and reads toAgentR; the client writes toAgentW, reads
	// toClientR.
	agentConn, _, _ := acpport.NewAgentSideConnection(acpport.NewAgent(front), toClientW, toAgentR)
	_ = acp.NewClientSideConnection(client, toAgentW, toClientR)

	sid, _, err := front.NewSession(ctx, acpport.NewSessionParams{Cwd: "/tmp/perm-cancel-ws"})
	if err != nil {
		t.Fatalf("front.NewSession: %v", err)
	}

	// The REAL prompt turn: the front's machine runs the send, the reverse
	// fires, and the ask parks on the client's gate.
	promptDone := make(chan acpport.StopReason, 1)

	go func() {
		reason, promptErr := front.Prompt(ctx, acpport.PromptParams{
			SessionID: sid,
			Text:      sendContent,
		}, acpport.NewUpdateSink(agentConn))
		if promptErr != nil {
			t.Errorf("front.Prompt: %v", promptErr)
		}

		promptDone <- reason
	}()

	client.awaitAsk(t)
	close(client.gate)           // the cancelled outcome answer releases → -32021 reply
	time.Sleep(permCancelSettle) // reply + send ack land deterministically

	if cancelErr := front.Cancel(ctx, sid); cancelErr != nil {
		t.Fatalf("front.Cancel: %v", cancelErr)
	}

	waitStopReason(t, promptDone, acpport.StopCancelled)

	time.Sleep(stopSettle) // the stop ack lands before stdin closes
}

// driveUserInputDecline: the CORE-06 decline corpus row (plan 05-02) — the
// front preamble, then send whose rule fires an interaction/requestUserInput
// reverse expecting {action:decline}. The Responder alone wires the hook
// (no bridge involvement — the policy handler answers the decline
// directly, the honest no-UI-channel posture; elicitation stays Phase 7).
func driveUserInputDecline(ctx context.Context, d *exchangeDriver) {
	d.conn.OnRequest = interact.NewResponder().Serve // hook contract: before any frame flows

	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}

// driveToolLifecycle: the TOOL-01 corpus row (plan 05-04) — the front
// preamble (copied verbatim from setmodel-followup), then TWO sends: the
// first (times budget 1) emits the full Edit tool lifecycle
// (scheduled→started→progress→batch→subagent-mirror→result), a same-turn
// text delta, and a FOREIGN-turnId delta (IN-05's suppression fixture at
// the fake tier); the second drives the Write-input scheduled event (the
// new-file diff shape). The golden byte-locks the ordered emit stream the
// front's tap consumes; the composed-tier wire-order lock lives in
// acpfront's TestToolLifecycle (the same corpus row through the harness).
func driveToolLifecycle(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx) // rule 1 (times 1): the Edit lifecycle + foreign delta
	h.send(ctx) // rule 2 (unlimited): the Write lifecycle
}

// driveSessionLifecycle: the SESS-01 corpus row (plan 05-05) — the front
// preamble (copied verbatim from setmodel-followup), then the lifecycle
// legs the front-tier contracts replay at the corpus tier (the
// driveFrontPreamble discipline: the front-tier contract calls driven
// verbatim so the golden byte-locks their wire shape): session/resume
// {sessionId} answered with the SAME create-shaped snapshot (the reply
// body session.sessionId dig-out createSessionID performs unchanged),
// the RE-ISSUED session/subscribe {deliveryKind:desktop-continuous}
// (the resume leg's subscribe, no setModel — model binding persists
// backend-side), and session/close {sessionId} answered {closed:true}.
// The golden locks the wire order create→subscribe→setModel→resume→
// subscribe→close; the composed-tier resume/close contracts live in
// acpfront's TestResume/TestCloseTeardown (the same row through the
// harness). The row also carries the send legs the composed tier drives
// (a completing send on times-1 rule 1, the active-turn send + stop rule
// for the close race probe) — unused here, matched on demand.
func driveSessionLifecycle(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	res, err := d.call(ctx, zcode.MethodSessionResume, map[string]any{
		wireKeySessionID: fixSessionID,
	})
	if err != nil {
		d.t.Fatalf("session/resume: %v", err)
	}

	assertResumeSnapshot(d.t, res)

	subParams := map[string]any{
		wireKeySessionID:    fixSessionID,
		wireKeyDeliveryKind: deliveryDesktopContinuous,
	}

	if _, err := d.call(ctx, zcode.MethodSessionSubscribe, subParams); err != nil {
		d.t.Fatalf("session/subscribe (resume leg): %v", err)
	}

	closeRes, err := d.call(ctx, zcode.MethodSessionClose, map[string]any{
		wireKeySessionID: fixSessionID,
	})
	if err != nil {
		d.t.Fatalf("session/close: %v", err)
	}

	assertCloseResult(d.t, closeRes)
}

// driveNewSessionOrphanClose: the WR-04 corpus row (plan 05-05) — create
// succeeds, the subscribe leg refuses (the times-1 -32603 rule), and the
// best-effort orphan close lands (session/close answered {closed:true}).
// The golden byte-locks the orphan-close wire shape at the corpus tier;
// the composed-tier lock lives in acpfront's TestNewSessionOrphanClosed
// (the front's actual WR-04 error path over the same row).
func driveNewSessionOrphanClose(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)

	_, err := d.call(ctx, zcode.MethodSessionSubscribe, map[string]any{
		wireKeySessionID:    fixSessionID,
		wireKeyDeliveryKind: deliveryDesktopContinuous,
	})

	var callErr *zcode.CallError
	if !errors.As(err, &callErr) {
		d.t.Fatalf("the refused session/subscribe must answer a wire error, got %v", err)
	}

	if callErr.Err.Code != zcode.CodeInternalError {
		d.t.Fatalf("session/subscribe error code = %d, want %d (-32603)", callErr.Err.Code, zcode.CodeInternalError)
	}

	closeRes, err := d.call(ctx, zcode.MethodSessionClose, map[string]any{wireKeySessionID: fixSessionID})
	if err != nil {
		d.t.Fatalf("session/close: %v", err)
	}

	assertCloseResult(d.t, closeRes)
}

// driveAuthApikey: the AUTH-01 corpus row (plan 05-06) — the push →
// refreshRegistry → read-back sequence against the fake, with the verdict
// driven by the hostsvc read-back helper (the PITFALL-1 discipline: the ack
// is receipt evidence, the registry is the verdict). The driver registers
// the watcher BEFORE any traffic (the hook contract — the refresh rule's
// side-effect emit precedes its reply on the wire), pushes the strict
// snapshot envelope built by hostsvc over the synthetic fixture install
// (injected fixed revision — the fake's static ack echoes it verbatim, the
// pairing the unit tier cannot show), asserts the ack echo pairing,
// refreshes the registry, and awaits the helper's applied verdict; then a
// fresh session/create whose reply is offered to the watcher explicitly
// (the dialect §8 create-reply read-back surface). The golden locks the
// push envelope bytes — the exact formula revision string computed at
// runtime over the fixture path — and the whole read-back sequence.
func driveAuthApikey(ctx context.Context, d *exchangeDriver) {
	// D-01 posture: the responder answers the create framing's
	// runtime-preferences reverse (byte-identical to the built-in router).
	d.conn.OnRequest = interact.NewResponder().Serve

	// Hook contract: registration before traffic.
	watcher, _ := hostsvc.WatchRegistry(d.conn, corpusAuthModels())

	install := supervisor.Install{
		Root:        "/zcode-acp-corpus-fixture",
		Binary:      "/zcode-acp-corpus-fixture/zcode",
		Entrypoint:  "/zcode-acp-corpus-fixture/resources/glm/zcode.cjs",
		BuiltinFile: corpusAuthBuiltinPath,
		Layout:      "explicit",
	}

	b := hostsvc.NewSnapshotBuilder(nil, hostsvc.WithRevisionSource(func() string { return corpusAuthRevision }))
	snap := b.BuildRevision(install, corpusAuthBuiltinRev)

	ack, err := hostsvc.PushEnvelope(ctx, d.conn, snap.Params)
	if err != nil {
		d.t.Fatalf("provider/updateAccountConfig: %v", err)
	}

	// The ack echo pairing: the reply's receivedRevision must echo the
	// pushed revision verbatim. Receipt evidence only — the verdict below
	// is the registry's (PITFALL 1).
	if ack.ReceivedRevision != corpusAuthRevision {
		d.t.Fatalf("ack receivedRevision = %q, want the pushed revision %q echoed",
			ack.ReceivedRevision, corpusAuthRevision)
	}

	if ack.Status != "received" {
		d.t.Fatalf("ack status = %q, want %q", ack.Status, "received")
	}

	if ack.ProviderCount != 0 {
		d.t.Fatalf("ack providerCount = %d, want 0 (the empty overlay)", ack.ProviderCount)
	}

	if err := hostsvc.RefreshRegistry(ctx, d.conn); err != nil {
		d.t.Fatalf("provider/refreshRegistry: %v", err)
	}

	// The read-back verdict: applied ONLY when the registry surface carried
	// the personal zai-api models intact.
	readbackCtx, cancel := context.WithTimeout(ctx, authReadbackWait)
	defer cancel()

	rb, err := watcher.Await(readbackCtx)
	if err != nil {
		d.t.Fatalf("registry read-back: %v", err)
	}

	if !rb.Applied {
		d.t.Fatalf("read-back verdict applied = false (revision %q) — the ack is never the verdict", rb.Revision)
	}

	// The fresh create-reply surface: a bare session/create whose reply is
	// offered to the watcher explicitly; the verdict stays applied.
	h := happyTurnHarness{t: d.t, conn: d.conn}
	watcher.OfferCreateReply(h.create(ctx))

	if _, err := watcher.Await(readbackCtx); err != nil {
		d.t.Fatalf("create-reply read-back: %v", err)
	}
}

// driveRuntimeHeadersAnswer: the AUTH-01 reply-vector corpus row (plan
// 05-06 Task 2) — the front preamble, then send whose rule fires ONE
// interaction/requestProviderRuntimeHeaders reverse (params per the
// research's request shape with a synthetic workspace/modelSelection; the
// 180 s-scale window configured small at timeout_ms 500; reannounce count 1
// — the dedupe key carries params.requestId, and the instant answer closes
// the schedule before its first envelope, which the golden locks: an
// answered ask earns no re-announce). The OnRequest hook carries the FULL
// composition the plan pins: the Responder whose RuntimeHeadersKeySource is
// the hostsvc handler over a synthetic StaticKey. The expect_answer locks
// the EXACT strict reply {headersApplied:true, requestAuth:{apiKey,
// apiKeyId}} (Pitfall 4: an invalid reply REJECTS the pending server-side
// request, so the shape is byte-load-bearing). The synthetic key material
// is fixture data only — never a real credential (T-5-15).
func driveRuntimeHeadersAnswer(ctx context.Context, d *exchangeDriver) {
	responder := interact.NewResponder(interact.WithRuntimeHeadersKeySource(
		hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey(syntheticFixtureKey, syntheticFixtureKeyID))))
	d.conn.OnRequest = responder.Serve // hook contract: before any frame flows

	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}

// driveFullCycle: the full-cycle corpus row (plan 05-08) — the ONE-scenario
// composition of the four leg rows' call sequences: authenticate (the
// driveAuthApikey push → refresh → read-back verdict over the injected
// fixed revision), the front preamble (driveFrontPreamble verbatim), the
// prompt leg (send rule 1: the Edit tool stream with the permission reverse
// fired BETWEEN the started and progress emits, answered by the scripted
// client through the FULL bridge chain — the drivePermissionAllow
// composition), then the SESS-01 legs (session/resume answered with the
// create-shaped snapshot + the re-issued subscribe, session/close answered
// {closed:true}) — the driveSessionLifecycle resume/close sequence. The
// golden byte-locks the whole interleaved wire order; the composed-tier
// locks live in acpfront's TestFullCycle (the same row through the
// harness).
func driveFullCycle(ctx context.Context, d *exchangeDriver) {
	t := d.t

	// ACP leg pipes: the drivePermissionAllow composition — the responder
	// (whose PermissionForwarder is the real front bridge) answers the
	// create framing's runtime-preferences reverse AND the mid-turn
	// permission reverse; the scripted SDK client answers allow_once on the
	// ACP leg.
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()

	t.Cleanup(func() {
		for _, c := range []io.Closer{toAgentR, toAgentW, toClientR, toClientW} {
			_ = c.Close()
		}
	})

	//nolint:contextcheck // the Front derives its own process-lifetime base
	// context by design (front.go New — the containedctx rule); the driver's
	// ctx bounds only the back-leg calls.
	front := acpfront.New(d.conn)

	// Hook contract: assigned before any frame flows — the first server
	// frame only moves when the auth chain's push runs below.
	responder := interact.NewResponder(interact.WithPermissionForwarder(
		interact.NewPermissionBridge(front.PermissionForward()).Forward,
	))
	d.conn.OnRequest = responder.Serve

	client := &permissionClient{}

	// Pipe ownership mirrors drivePermissionAllow: the agent writes
	// toClientW and reads toAgentR; the client writes toAgentW, reads
	// toClientR.
	//
	//nolint:dogsled // the front carries all three bindings; none is needed here
	_, _, _ = acpport.NewAgentSideConnection(acpport.NewAgent(front), toClientW, toAgentR)
	_ = acp.NewClientSideConnection(client, toAgentW, toClientR)

	// AUTH leg (the driveAuthApikey sequence): watcher registered BEFORE
	// any traffic, push the strict snapshot, refresh, read-back verdict.
	fullCycleAuthLeg(ctx, d)

	// PREAMBLE leg (driveFrontPreamble verbatim): create + subscribe +
	// setModel — the corpus-locked front-tier contract.
	driveFrontPreamble(ctx, d)

	// PROMPT leg: the send fires the row's composed exchange — tool.updated
	// scheduled/started, the permission reverse (answered by the scripted
	// client through the full bridge chain), progress, result, the
	// same-turn text delta, turn.completed, then the ack.
	h := happyTurnHarness{t: t, conn: d.conn}
	h.send(ctx)

	// RESUME leg (the driveSessionLifecycle sequence): reattach via the
	// create-shaped snapshot, then the re-issued subscribe (no setModel).
	res, err := d.call(ctx, zcode.MethodSessionResume, map[string]any{
		wireKeySessionID: fixSessionID,
	})
	if err != nil {
		t.Fatalf("session/resume: %v", err)
	}

	assertResumeSnapshot(t, res)

	subParams := map[string]any{
		wireKeySessionID:    fixSessionID,
		wireKeyDeliveryKind: deliveryDesktopContinuous,
	}

	if _, err := d.call(ctx, zcode.MethodSessionSubscribe, subParams); err != nil {
		t.Fatalf("session/subscribe (resume leg): %v", err)
	}

	// CLOSE leg: the teardown close, answered {closed:true}.
	closeRes, err := d.call(ctx, zcode.MethodSessionClose, map[string]any{
		wireKeySessionID: fixSessionID,
	})
	if err != nil {
		t.Fatalf("session/close: %v", err)
	}

	assertCloseResult(t, closeRes)
}

// fullCycleAuthLeg is the driveAuthApikey auth sequence the full-cycle row's
// authenticate leg composes: watcher registered BEFORE any traffic (the
// refresh rule's side-effect emit precedes its reply), push the strict
// snapshot over the synthetic fixture install (the injected fixed revision
// — the fake's static ack echoes it verbatim), refresh, then the read-back
// verdict (PITFALL 1 — never the ack).
func fullCycleAuthLeg(ctx context.Context, d *exchangeDriver) {
	t := d.t

	watcher, _ := hostsvc.WatchRegistry(d.conn, corpusAuthModels())

	install := supervisor.Install{
		Root:        "/zcode-acp-corpus-fixture",
		Binary:      "/zcode-acp-corpus-fixture/zcode",
		Entrypoint:  "/zcode-acp-corpus-fixture/resources/glm/zcode.cjs",
		BuiltinFile: corpusAuthBuiltinPath,
		Layout:      "explicit",
	}

	b := hostsvc.NewSnapshotBuilder(nil, hostsvc.WithRevisionSource(func() string { return corpusAuthRevision }))
	snap := b.BuildRevision(install, corpusAuthBuiltinRev)

	ack, err := hostsvc.PushEnvelope(ctx, d.conn, snap.Params)
	if err != nil {
		t.Fatalf("provider/updateAccountConfig: %v", err)
	}

	if ack.ReceivedRevision != corpusAuthRevision {
		t.Fatalf("ack receivedRevision = %q, want the pushed revision %q echoed",
			ack.ReceivedRevision, corpusAuthRevision)
	}

	if ack.Status != "received" {
		t.Fatalf("ack status = %q, want %q", ack.Status, "received")
	}

	if err := hostsvc.RefreshRegistry(ctx, d.conn); err != nil {
		t.Fatalf("provider/refreshRegistry: %v", err)
	}

	readbackCtx, cancel := context.WithTimeout(ctx, authReadbackWait)
	defer cancel()

	rb, err := watcher.Await(readbackCtx)
	if err != nil {
		t.Fatalf("registry read-back: %v", err)
	}

	if !rb.Applied {
		t.Fatalf("read-back verdict applied = false (revision %q) — the ack is never the verdict", rb.Revision)
	}
}

// driveLingeringLatch32010: the lingering-latch corpus row (plan 05-13) —
// the front preamble, then send-1 whose rule acks with EARLY completion
// surfaces (running→idle state patches, reason prompt_completed) and arms
// the desktop's one-turn-at-a-time latch for latch_ms; the immediate send-2
// inside the window is refused -32010 (the EXP-APIKEY-CYCLE-02.jsonl:119
// refusal verbatim: "A prompt is already running for this session") WITHOUT
// consuming rule budget; after latch release send-3 matches the next rule
// fresh and is accepted with turn-2 completion surfaces — the drained
// recovery attempt 3's test demands.
func driveLingeringLatch32010(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx) // send-1: ack + early completion surfaces, arms the latch

	d.waitStatePatch("idle")

	params := map[string]any{
		wireKeySessionID: fixSessionID,
		wireKeyContent:   sendContent,
	}

	_, err := d.call(ctx, zcode.MethodSessionSend, params)

	var callErr *zcode.CallError
	if !errors.As(err, &callErr) {
		d.t.Fatalf("send-2 inside the latch window must be refused with a wire error, got %v", err)
	}

	if callErr.Err.Code != zcode.CodeConcurrency {
		d.t.Fatalf("send-2 error code = %d, want %d (-32010)", callErr.Err.Code, zcode.CodeConcurrency)
	}

	// Latch drain: wait out the window plus margin before the retry — the
	// corpus rehearsal of the real-tier turnWithLatchDrain discipline.
	time.Sleep(lingeringLatchSettle)

	h.send(ctx) // send-3 after release: accepted, turn-2 completion surfaces

	d.waitStatePatch("idle")
}

// assertResumeSnapshot digs result.session.sessionId out of the resume
// reply — the SAME decode path createSessionID runs on the create reply
// (the backend resume snapshot shape, 05-RESEARCH.md Pattern 7): a reply
// without the id fails loud.
func assertResumeSnapshot(t *testing.T, res json.RawMessage) {
	t.Helper()

	var result struct {
		Session json.RawMessage `json:"session"`
	}

	if err := json.Unmarshal(res, &result); err != nil {
		t.Fatalf("session/resume: decode reply: %v (%s)", err, res)
	}

	var session map[string]any
	if err := json.Unmarshal(result.Session, &session); err != nil {
		t.Fatalf("session/resume: decode session: %v (%s)", err, res)
	}

	if sid, _ := session["sessionId"].(string); sid != fixSessionID {
		t.Fatalf("session/resume snapshot sessionId = %q, want %q (the create-shaped snapshot)", sid, fixSessionID)
	}
}

// assertCloseResult asserts the session/close reply carries the dialect's
// {closed:true} result (05-RESEARCH.md Pattern 7 close params/result).
func assertCloseResult(t *testing.T, res json.RawMessage) {
	t.Helper()

	var result struct {
		Closed bool `json:"closed"`
	}

	if err := json.Unmarshal(res, &result); err != nil {
		t.Fatalf("session/close: decode reply: %v (%s)", err, res)
	}

	if !result.Closed {
		t.Fatalf("session/close result = %s, want {\"closed\":true}", res)
	}
}

// waitStopReason blocks for the in-flight prompt's resolution and requires
// the given stop reason.
func waitStopReason(t *testing.T, done <-chan acpport.StopReason, want acpport.StopReason) {
	t.Helper()

	select {
	case got := <-done:
		if got != want {
			t.Fatalf("prompt resolved %q, want %q", got, want)
		}
	case <-time.After(noticeWaitTimeout):
		t.Fatal("prompt never resolved within the wait bound")
	}
}

// permissionClient is the fake_test-local SDK Client double the corpus
// replays sit on the ACP leg: it answers every session/request_permission
// with its scripted mode — the canonical allow decision (selected
// allow_once) or the cancelled outcome arm (the ACP v1 MUST on turn
// cancel) — optionally holding the answer behind a gate the driver closes
// at a chosen moment, and records every ask it received.
type permissionClient struct {
	gate chan struct{}

	mu     sync.Mutex
	cancel bool
	asks   []acp.RequestPermissionRequest
}

// RequestPermission records the ask and answers the scripted outcome.
func (c *permissionClient) RequestPermission(
	ctx context.Context,
	params acp.RequestPermissionRequest,
) (acp.RequestPermissionResponse, error) {
	c.mu.Lock()
	c.asks = append(c.asks, params)
	cancelled := c.cancel
	gate := c.gate
	c.mu.Unlock()

	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return acp.RequestPermissionResponse{}, ctx.Err()
		}
	}

	if cancelled {
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
				OptionId: "allow_once",
			},
		},
	}, nil
}

// SessionUpdate accepts streamed updates (none flow on this leg).
func (c *permissionClient) SessionUpdate(ctx context.Context, params acp.SessionNotification) error {
	return nil
}

// CompleteElicitation is a no-op (elicitation is Phase 7).
func (c *permissionClient) CompleteElicitation(
	ctx context.Context,
	params acp.CompleteElicitationNotification,
) error {
	return nil
}

// CreateElicitation is a no-op (elicitation is Phase 7).
func (c *permissionClient) CreateElicitation(
	ctx context.Context,
	params acp.CreateElicitationRequest,
) (acp.CreateElicitationResponse, error) {
	return acp.CreateElicitationResponse{}, nil
}

// ReadTextFile is a no-op (fs surfaces are never advertised).
func (c *permissionClient) ReadTextFile(
	ctx context.Context,
	params acp.ReadTextFileRequest,
) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, nil
}

// WriteTextFile is a no-op (fs surfaces are never advertised).
func (c *permissionClient) WriteTextFile(
	ctx context.Context,
	params acp.WriteTextFileRequest,
) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, nil
}

// CreateTerminal is a no-op (terminal surfaces are never advertised).
func (c *permissionClient) CreateTerminal(
	ctx context.Context,
	params acp.CreateTerminalRequest,
) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{}, nil
}

// KillTerminal is a no-op (terminal surfaces are never advertised).
func (c *permissionClient) KillTerminal(
	ctx context.Context,
	params acp.KillTerminalRequest,
) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, nil
}

// TerminalOutput is a no-op (terminal surfaces are never advertised).
func (c *permissionClient) TerminalOutput(
	ctx context.Context,
	params acp.TerminalOutputRequest,
) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{}, nil
}

// ReleaseTerminal is a no-op (terminal surfaces are never advertised).
func (c *permissionClient) ReleaseTerminal(
	ctx context.Context,
	params acp.ReleaseTerminalRequest,
) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}

// WaitForTerminalExit is a no-op (terminal surfaces are never advertised).
func (c *permissionClient) WaitForTerminalExit(
	ctx context.Context,
	params acp.WaitForTerminalExitRequest,
) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, nil
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

// awaitAsk blocks until the client received its first request_permission
// ask (bounded) — the wire-level sync signal the drivers gate on.
func (c *permissionClient) awaitAsk(t *testing.T) {
	t.Helper()

	deadline := time.After(noticeWaitTimeout)

	for {
		c.mu.Lock()
		n := len(c.asks)
		c.mu.Unlock()

		if n > 0 {
			return
		}

		select {
		case <-time.After(2 * time.Millisecond):
		case <-deadline:
			t.Fatal("no request_permission ask reached the scripted client within the wait bound")
		}
	}
}
