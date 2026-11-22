package fake_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
		"happy-turn":                     driveHappyTurnCorpus,
		"cancel-mid-turn":                driveCancelMidTurn,
		"error-32010":                    driveError32010,
		"unknown-host-method":            driveUnknownHostMethod,
		"official-mcp-auth-decline":      driveOfficialMcpAuthDecline,
		"runtime-preferences":            driveRuntimePreferences,
		"completion-idle-patch":          driveCompletionIdlePatch,
		"completion-quiet-window":        driveCompletionQuietWindow,
		"eof-drain":                      driveEOFDrain,
		"envelope-negatives":             driveEnvelopeNegatives,
		"credential-cipher-scripted":     driveCredentialCipherScripted,
		"setmodel-followup":              driveSetModelFollowup,
		"part-delta-streaming":           drivePartDeltaStreaming,
		"desktop-delta-streaming":        driveDesktopDeltaStreaming,
		"desktop-nodelta-messages":       driveDesktopNodeltaMessages,
		"desktop-delta-messages":         driveDesktopDeltaMessages,
		"desktop-nodelta-pollerror":      driveDesktopNodeltaPollError,
		"desktop-nodelta-slowpoll":       driveDesktopNodeltaSlowPoll,
		"desktop-nodelta-noassistant":    driveDesktopNodeltaNoAssistant,
		"no-reasoning":                   driveNoReasoning,
		"send-refused-32010":             driveSendRefused32010,
		"complete-then-cancel":           driveCompleteThenCancel,
		"reannounce-dedupe":              driveReannounceDedupe,
		"permission-allow":               drivePermissionAllow,
		"permission-cancel":              drivePermissionCancel,
		"user-input-decline":             driveUserInputDecline,
		"tool-lifecycle":                 driveToolLifecycle,
		"session-lifecycle":              driveSessionLifecycle,
		"newsession-orphan-close":        driveNewSessionOrphanClose,
		"auth-apikey":                    driveAuthApikey,
		"coding-plan-auth-cycle":         driveCodingPlanAuthCycle,
		cpCorpusProbeFailName:            driveCodingPlanProbeFail,
		cpCorpusProbeSetModelName:        driveCodingPlanProbeSetModel,
		cpCorpusProbeSetModelRefusedName: driveCodingPlanProbeSetModelRefused,
		"runtime-headers-answer":         driveRuntimeHeadersAnswer,
		"runtime-headers-pat":            driveRuntimeHeadersPat,
		"full-cycle":                     driveFullCycle,
		"lingering-latch-32010":          driveLingeringLatch32010,
		"config-option-set-model":        driveConfigOptionSetModel,
		"config-option-thought":          driveConfigOptionThought,
	}
}

// goldenPostAsserts maps corpus rows to post-replay transcript assertions —
// semantic locks that ride the recorded conversation beyond the byte-golden
// (the fake records both directions, so reply addressing is assertable).
func goldenPostAsserts() map[string]func(t *testing.T, tr fake.Transcript) {
	return map[string]func(t *testing.T, tr fake.Transcript){
		"reannounce-dedupe":              assertReannounceDedupeTranscript,
		cpCorpusProbeFailName:            assertCodingPlanProbeFailTranscript,
		cpCorpusProbeSetModelName:        assertCodingPlanProbeSetModelTranscript,
		cpCorpusProbeSetModelRefusedName: assertCodingPlanProbeSetModelRefusedTranscript,
	}
}

// goldenNormalizers maps corpus rows to pre-golden transcript prep: the
// coding-plan rows drive the FRONT's production chain, whose push revision
// (account-bridge:<unix-nano>) and builtin-table hash
// (zcode-builtin:44:<sha256 of the resolved fixture path>) are
// machine-derived BY FORMULA DESIGN (T-5-16: the formula is never
// hard-coded, never cached). The normalizer substitutes their fixture
// tokens so the byte-lock stays cross-machine deterministic — the same
// determinism the auth-apikey row gets from its fixed synthetic builder
// path. Every OTHER byte stays locked: frame order, methods, params
// shapes, and the ack/echo pairing all remain byte-exact.
func goldenNormalizers() map[string]func([]byte) []byte {
	return map[string]func([]byte) []byte{
		"coding-plan-auth-cycle":         normalizeCodingPlanCorpus,
		cpCorpusProbeFailName:            normalizeCodingPlanCorpus,
		cpCorpusProbeSetModelName:        normalizeCodingPlanCorpus,
		cpCorpusProbeSetModelRefusedName: normalizeCodingPlanCorpus,
		"runtime-headers-pat":            normalizeCodingPlanCorpus,
	}
}

// The fixture tokens the normalizer substitutes (the ack's scripted echo
// pairs with the substituted push revision — the golden then SHOWS the
// receipt pairing). cpCorpusProbeDirToken substitutes the probe sessions'
// os.MkdirTemp random suffix (the round-3 poke rows' only other
// machine-derived byte run).
const (
	cpCorpusRevisionToken = "account-bridge:cp-fix1"
	cpCorpusBuiltinToken  = "zcode-builtin:44:cp-corpus-fixture-hash"
	cpCorpusProbeDirToken = "zcode-acp-auth-probe-<probe-dir>"
)

var (
	cpCorpusRevPattern     = regexp.MustCompile(`account-bridge:\d+`)
	cpCorpusBuiltinPattern = regexp.MustCompile(`zcode-builtin:44:[0-9a-f]{64}`)
	cpCorpusProbeDir       = regexp.MustCompile(`zcode-acp-auth-probe-\d+`)
)

// normalizeCodingPlanCorpus substitutes the machine-derived tokens.
func normalizeCodingPlanCorpus(b []byte) []byte {
	b = cpCorpusRevPattern.ReplaceAll(b, []byte(cpCorpusRevisionToken))
	b = cpCorpusProbeDir.ReplaceAll(b, []byte(cpCorpusProbeDirToken))

	return cpCorpusBuiltinPattern.ReplaceAll(b, []byte(cpCorpusBuiltinToken))
}

// assertCodingPlanProbeFailTranscript is the probe-fail row's semantic
// belt to the byte-lock braces: a probe failure must leave NO push and NO
// refresh frame on the wire (D-02 ordering, AUTH-04).
func assertCodingPlanProbeFailTranscript(t *testing.T, tr fake.Transcript) {
	t.Helper()

	for _, line := range tr.Lines {
		if strings.Contains(line.JSON, `"method":"provider/`) {
			t.Fatalf("probe-fail transcript carries a provider frame: %s", line.JSON)
		}
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

			golden := goldenBytes(tr)
			if norm, ok := goldenNormalizers()[name]; ok {
				golden = norm(golden)
			}

			fake.CompareGolden(t, "testdata/golden/"+name+".golden", golden)
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

// driveUnknownHostMethod: create; the fake's reverse of a GENUINELY unknown
// method (interaction/requestFoo — the -32601 arm stays scoped to methods
// with no dialect §3 surface) is answered -32601 naming the method by the
// conn's router and the flow continues to the create reply.
func driveUnknownHostMethod(ctx context.Context, d *exchangeDriver) {
	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.create(ctx)
}

// driveOfficialMcpAuthDecline: the official-MCP auth corpus row (the
// EXP-CODINGPLAN-CYCLE-01 attempt-1 unblock) — create whose rule fires the
// interaction/requestOfficialMcpAuthHeaders reverse (dialect §3, the
// attempt-1 capture's params shape with a fixed synthetic workspace) and
// expects the wrapper's honest structured decline {ok:false,
// reason:"official_auth_unavailable"} INSIDE the reply RESULT (never
// -32601 — the attempt-1 capture recorded that error stalling the plugin
// runtime materialization), then the session continues to the ready state
// and the create reply. The Responder alone wires the hook (the static
// decline needs no seam).
func driveOfficialMcpAuthDecline(ctx context.Context, d *exchangeDriver) {
	d.conn.OnRequest = interact.NewResponder().Serve // hook contract: before any frame flows

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
		corpusWireKeyModel: map[string]any{
			"providerId": "zai-api",
			"modelId":    "GLM-5.3-Flash",
			"options":    map[string]any{"reasoningLevel": corpusCfgLevelHigh},
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

// driveDesktopDeltaStreaming: the QA-03 corpus row — the REAL desktop's
// model-response delta carrier (EXP-APIKEY-CYCLE-03 frames 176+, scrubbed
// synthetic ids/text). The send rule's ordered side effects replay the
// capture-verbatim frame shapes: turn.started with the desktop's full
// envelope surface (deliveryKind/eventId/seq/sessionId/timestamp/traceId/
// turnId), interleaved model.streaming deltas (payload.kind reasoning_delta
// and text_delta carrying payload.delta per assistantMessageId), the
// tool_input_start/delta/end trio (tool-args streaming — held from the
// client surface by contract), and turn.completed. The golden byte-locks
// the ordered emit stream the front's tap consumes in arrival order; the
// client-visible chunk lock lives in acpfront's TestDesktopDeltaStreaming
// (the ACP-leg golden over the same corpus row).
func driveDesktopDeltaStreaming(ctx context.Context, d *exchangeDriver) {
	driveFrontPreamble(ctx, d)

	h := happyTurnHarness{t: d.t, conn: d.conn}
	h.send(ctx)
}

// driveDesktopNodeltaMessages: the QA-03 completion-side reconcile's
// flagship corpus row (EXP-CODINGPLAN-CYCLE-01 poll-24 projection shape) —
// the front preamble, the no-delta send (turn.started → turn.completed with
// ZERO model.streaming rows), then ONE session/messages poll replaying the
// reconcile's completion-time read at the back tier: the golden byte-locks
// the messages projection whose LAST role:assistant entry's type:text parts
// carry the stored reply (beside reasoning/step-*/timeline parts). The
// client-visible reconcile chunk lock lives in acpfront's
// TestNoDeltaReplyReconcile (the ACP-leg golden over the same corpus row).
func driveDesktopNodeltaMessages(ctx context.Context, d *exchangeDriver) {
	driveDesktopReconcileTurn(ctx, d)

	if _, err := d.call(ctx, zcode.MethodSessionMessages, map[string]any{wireKeySessionID: fixSessionID}); err != nil {
		d.t.Fatalf("session/messages: %v", err)
	}
}

// driveDesktopNodeltaNoAssistant: the reconcile's unrecoverable-text row —
// the same turn and poll as the flagship row, but the projection carries no
// assistant reply (timeline + user entries only): nothing extractable, so
// nothing is ever emitted.
func driveDesktopNodeltaNoAssistant(ctx context.Context, d *exchangeDriver) {
	driveDesktopReconcileTurn(ctx, d)

	if _, err := d.call(ctx, zcode.MethodSessionMessages, map[string]any{wireKeySessionID: fixSessionID}); err != nil {
		d.t.Fatalf("session/messages: %v", err)
	}
}

// driveDesktopDeltaMessages: the reconcile's no-duplicate row — the same
// no-delta turn shape with TWO text_delta rows streaming before the
// terminal, and the messages rule present (the projection data available):
// a front that lost the streamed-deltas counter would duplicate the reply
// here. The poll itself is locked by the flagship row's replay.
func driveDesktopDeltaMessages(ctx context.Context, d *exchangeDriver) {
	driveDesktopReconcileTurn(ctx, d)
}

// driveDesktopNodeltaPollError: the reconcile's honest-failure row — the
// no-delta turn whose session/messages poll answers a dialect §6 internal
// error: the reconcile read fails, nothing is fabricated, the stopReason
// still leaves.
func driveDesktopNodeltaPollError(ctx context.Context, d *exchangeDriver) {
	driveDesktopReconcileTurn(ctx, d)

	_, err := d.call(ctx, zcode.MethodSessionMessages, map[string]any{wireKeySessionID: fixSessionID})

	var callErr *zcode.CallError
	if !errors.As(err, &callErr) {
		d.t.Fatalf("the refused session/messages must answer a wire error, got %v", err)
	}

	if callErr.Err.Code != zcode.CodeInternalError {
		d.t.Fatalf("session/messages error code = %d, want %d (-32603)", callErr.Err.Code, zcode.CodeInternalError)
	}
}

// driveDesktopNodeltaSlowPoll: the reconcile's bounded-read row — the
// no-delta turn whose messages rule answers 3s past the front's bounded
// read. The replay drives preamble + send only: a poll here would stall the
// exchange on the scripted delay for no back-tier fact (the delay semantics
// are the acpfront tier's lock).
func driveDesktopNodeltaSlowPoll(ctx context.Context, d *exchangeDriver) {
	driveDesktopReconcileTurn(ctx, d)
}

// driveDesktopReconcileTurn is the shared reconcile-row body: the front
// preamble plus the no-delta send (the rule's ordered side effects replay
// state.running → turn.started → [deltas] → turn.completed with the
// desktop envelope surface verbatim).
func driveDesktopReconcileTurn(ctx context.Context, d *exchangeDriver) {
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
// The coding-plan corpus fixtures (plan 06-05). All material is synthetic
// fixture data — never a real credential (T-5-15; the scan-secrets gate
// stays green).
const (
	// cpCorpusSessionID is the coding-plan row's fixed session id.
	cpCorpusSessionID = "sess_cp1"

	// cpCorpusProbeFailName is the probe-fail row's corpus name (the three
	// map keys the row registers under).
	cpCorpusProbeFailName = "coding-plan-probe-fail"

	// The round-3 poke rows' corpus names (the three map keys each
	// registers under): the happy setModel-poke row and its coded-refusal
	// twin (EXP-CODINGPLAN-CYCLE-01 round-3 root cause).
	cpCorpusProbeSetModelName        = "coding-plan-probe-setmodel"
	cpCorpusProbeSetModelRefusedName = "coding-plan-probe-setmodel-refused"

	// cpCorpusProbeReadbackDeadline is the poke rows' injected read-back
	// window: the refused row's Await must time out fast (the honest coded
	// read-back failure) without racing the harness timeout.
	cpCorpusProbeReadbackDeadline = 500 * time.Millisecond

	// corpusLayoutExplicit is the fixture install layout string.
	corpusLayoutExplicit = "explicit"

	// corpusWorkspacePath is the fixed synthetic workspace the corpus
	// preambles create against (byte-deterministic).
	corpusWorkspacePath = "/tmp/happy-turn-ws"

	// corpusCfgOptionModel/corpusCfgOptionThought are the config-option
	// rows' configId values; corpusCfgLevelHigh the shared confirmed level
	// (goconst: the repeated capture-verbatim spellings extract to one
	// source each).
	corpusCfgOptionModel   = "model"
	corpusCfgOptionThought = "thought_level"
	corpusCfgLevelHigh     = "high"

	// corpusWireKeyModel is the setModel params' model-object wire key.
	corpusWireKeyModel = "model"

	// cpCorpusAccountProvider/Family are the fixture builtin table's
	// zhipu-account rule (the account overlay's single provider).
	cpCorpusAccountProvider = "account:zai-individual-coding-plan"
	cpCorpusAccountFamily   = "zai"

	// cpCorpusLoginToken is the fixture store's oauth login (the 06-01
	// reader serves it Bearer-stripped to the front's coding-plan arm).
	cpCorpusLoginToken = "fixture-synthetic-not-a-real-login-token"

	// cpCorpusCipherKey is the fixture v2 key (32 bytes) — the blob baked
	// into the coding-plan-auth-cycle scenario is sealed under exactly this
	// key with a fixed synthetic IV (its kid is sha256(key) hex[:24],
	// computed once and frozen; the responder's CredentialDecrypt seam
	// decrypts it via the fixture loader below, and the scenario's
	// expect_answer carries the plaintext).
	cpCorpusCipherKey = "0123456789abcdef0123456789abcdef"
)

// errCannedProbeFailure is the probe-fail row's injected probe verdict
// (err113: a static error, never dynamically constructed).
var errCannedProbeFailure = errors.New("canned probe failure")

// corpusFixtureKeyLoader is the injected KeyLoader the cipher seam decrypts
// with (the KeyLoader injection discipline — no keyring on CI).
type corpusFixtureKeyLoader struct{}

// LoadKey returns the fixture 32-byte key.
func (corpusFixtureKeyLoader) LoadKey(_ context.Context) ([]byte, error) {
	return []byte(cpCorpusCipherKey), nil
}

// corpusPat is the PatEnsurer double: counts Ensure calls, returns the
// synthetic fixture PAT (the mint-once lock reads the counter).
type corpusPat struct {
	mu    sync.Mutex
	calls int
}

// Ensure replays the fixture PAT and counts the call.
func (p *corpusPat) Ensure(_ context.Context) (hostsvc.Pat, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()

	return hostsvc.Pat{
		Token:        syntheticFixtureKey,
		APIKeyID:     syntheticFixtureKeyID,
		Fingerprint:  strings.Repeat("0", 64),
		AccountScope: strings.Repeat("f", 64),
	}, nil
}

// count reports the recorded Ensure count.
func (p *corpusPat) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.calls
}

// codingPlanCorpusInstall writes the fixture builtin table (revision 44 +
// one zhipu-account provider rule — the account overlay's source) into a
// temp install. The resolved PATH STRING feeds the exact-revision formula;
// the corpus normalizer substitutes the machine-derived hash so the golden
// stays cross-machine deterministic.
func codingPlanCorpusInstall(t *testing.T) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	doc := `{"revision":44,"config":{"providerConfigRules":{"providerRules":[` +
		`{"providerId":"` + cpCorpusAccountProvider + `","config":{` +
		`"builtinModelIds":["GLM-5.3","GLM-5.3-Flash"],` +
		`"access":{"type":"zhipu-account","accountType":"` + cpCorpusAccountFamily + `"}}}]}}}`

	if err := os.WriteFile(builtin, []byte(doc), 0o600); err != nil {
		t.Fatalf("write corpus builtin fixture: %v", err)
	}

	return supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: builtin,
		Layout:      corpusLayoutExplicit,
	}
}

// corpusCreds writes the fixture credentials.json (one oauth login entry)
// and returns the 06-01 store over it — the SAME reader the production
// coding-plan arm composes.
func corpusCreds(t *testing.T) *interact.CredentialStore {
	t.Helper()

	path := filepath.Join(t.TempDir(), "credentials.json")

	doc := `{"oauth:` + cpCorpusAccountFamily + `:access_token":"` + cpCorpusLoginToken + `"}`

	if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
		t.Fatalf("write corpus credentials fixture: %v", err)
	}

	// v2 loader nil: the fixture entry is plaintext (the store's default
	// dispatch arm — the reader never touches a keyring for it).
	return interact.NewCredentialStore(path, nil)
}

// driveCodingPlanAuthCycle replays the coding-plan auth cycle (plan 06-05,
// AUTH-02): the FRONT's production chain over the scripted wire — the
// probe-pass (in-process, canned) → the account-overlay push (the ack's
// providerCount 1 receipt) → the refresh whose registry patch carries the
// account models (the verdict surface — the watcher latches it, so the
// probe-create read-back arm never runs and the create below is the
// post-auth preamble's) → the mint-once (counter-asserted) → the
// create/subscribe/setModel preamble whose settings carry the account
// models + the thoughtLevel block and whose reverse asks answer with the
// scripted cipher decrypt (the CredentialDecrypt seam over the fixture
// loader — the blob/kid pair is baked into the scenario) and the scripted
// PAT source. The push revision + builtin hash are normalized to fixture
// tokens in the golden assembly (goldenNormalizers) — every other byte
// stays locked.
func driveCodingPlanAuthCycle(ctx context.Context, d *exchangeDriver) {
	pat := &corpusPat{}

	//nolint:contextcheck // the Front derives its own process-lifetime base
	// context by design (front.go New — the containedctx rule); the driver's
	// ctx bounds only the back-leg calls.
	front := acpfront.New(d.conn, acpfront.WithAuth(acpfront.AuthConfig{
		Install: codingPlanCorpusInstall(d.t),
		Probe:   func(_ context.Context) error { return nil }, // canned probe-pass
		Creds:   corpusCreds(d.t),
		Pat:     pat,
		Entitle: func(_ context.Context, family string) bool { return family == cpCorpusAccountFamily },
	}))

	// Hook contract: the responder (scripted PAT source + the cipher seam)
	// is assigned before any frame flows. The scripted source carries NO
	// scope: the coding-plan-auth-cycle row's expect_answer stays the
	// byte-locked two-field shape (the widening's byte-compat regression —
	// plan 06-07 Task 1 keeps this row's reply bytes identical).
	responder := interact.NewResponder(
		interact.WithRuntimeHeadersKeySource(
			func(_ context.Context, _ interact.Request) (string, string, string, error) {
				return syntheticFixtureKey, syntheticFixtureKeyID, "", nil
			}),
		interact.WithCredentialDecrypt(interact.NewCredentialDecrypt(corpusFixtureKeyLoader{})),
	)
	d.conn.OnRequest = responder.Serve

	if err := front.Authenticate(ctx, acpfront.AuthMethodZaiCodingPlan); err != nil {
		d.t.Fatalf("authenticate coding-plan: %v", err)
	}

	if calls := pat.count(); calls != 1 {
		d.t.Fatalf("PAT Ensure ran %d time(s), want exactly one (mint-once)", calls)
	}

	// The post-auth preamble: create → subscribe → setModel (the
	// driveFrontPreamble shape over the coding-plan session; the create
	// reply carries the account models in settings.model.available and the
	// thoughtLevel block, and its reverse asks answer the cipher + PAT
	// arms with expect_answer byte-compare).
	if _, err := d.call(ctx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{"workspacePath": corpusWorkspacePath, "workspaceKey": corpusWorkspacePath},
	}); err != nil {
		d.t.Fatalf("session/create: %v", err)
	}

	if _, err := d.call(ctx, zcode.MethodSessionSubscribe, map[string]any{
		wireKeySessionID:    cpCorpusSessionID,
		wireKeyDeliveryKind: deliveryDesktopContinuous,
	}); err != nil {
		d.t.Fatalf("session/subscribe: %v", err)
	}

	if _, err := d.call(ctx, zcode.MethodSessionSetModel, map[string]any{
		wireKeySessionID: cpCorpusSessionID,
		corpusWireKeyModel: map[string]any{
			"providerId": cpCorpusAccountProvider,
			"modelId":    "GLM-5.3",
			"options":    map[string]any{"reasoningLevel": corpusCfgLevelHigh},
		},
		"persistAsWorkspaceLastUsed": false,
	}); err != nil {
		d.t.Fatalf("session/setModel: %v", err)
	}
}

// driveCodingPlanProbeFail replays the probe-fail ordering row (plan 06-05,
// AUTH-04): the front's chain with a FAILING prober answers the coded
// keyring error and NOTHING touches the wire — the transcript carries only
// the scripted canary (whose unknown-method reply proves the exchange
// lived; a zero-frame transcript is a harness failure signature). The
// golden byte-locks the push-absence; the post-assert is the semantic belt.
func driveCodingPlanProbeFail(ctx context.Context, d *exchangeDriver) {
	//nolint:contextcheck // the Front derives its own process-lifetime base
	// context by design (front.go New — the containedctx rule); the driver's
	// ctx bounds only the back-leg calls.
	front := acpfront.New(d.conn, acpfront.WithAuth(acpfront.AuthConfig{
		Install: codingPlanCorpusInstall(d.t),
		Probe:   func(_ context.Context) error { return errCannedProbeFailure },
		Creds:   corpusCreds(d.t),
		Pat:     &corpusPat{},
		Entitle: func(_ context.Context, family string) bool { return family == cpCorpusAccountFamily },
	}))

	if err := front.Authenticate(ctx, acpfront.AuthMethodZaiCodingPlan); err == nil {
		d.t.Fatal("authenticate answered success on a failing probe — never a fabricated success")
	} else if !strings.Contains(err.Error(), "keyring") {
		d.t.Fatalf("probe-fail error %v must name the keyring step", err)
	}

	// The canary: the row's only recorded exchange (the unknown-method
	// reply the scenario scripts) — every push/refresh frame's ABSENCE is
	// the lock.
	if _, err := d.call(ctx, "corpus/probe-fail-canary", map[string]any{}); err == nil {
		d.t.Fatal("the canary must record the scripted unknown-method error")
	}
}

// The runtime-headers-pat row's fixture constants (plan 06-07 Task 3): the
// synthetic PAT pair, their key ids, the account scope (unchanged across the
// re-ask — the scope digest covers the login, not the token), and
// fingerprint(PAT-A) as a FIXTURE CONSTANT (not secret-derived — the shape
// is what the byte-lock pins, never a real credential, T-5-15).
const (
	patCorpusTokenA = "zc-synthetic-pat-a-corpus-fixture"
	patCorpusTokenB = "zc-synthetic-pat-b-corpus-fixture"
	patCorpusKeyA   = "key_pat_corpus_a"
	patCorpusKeyB   = "key_pat_corpus_b"

	patCorpusScopeA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	patCorpusFPA    = "4444444444444444444444444444444444444444444444444444444444444444"

	// patCorpusFPB is fingerprint(PAT-B) — fixture constant like PAT-A's.
	patCorpusFPB = "5555555555555555555555555555555555555555555555555555555555555555"

	// patCorpusSessionID is the row's fixed session id (goconst: the
	// driver's repeated capture-verbatim spelling extracts to one source).
	patCorpusSessionID = "sess_rhp1"

	// The wire-key literals the driver's params maps carry (goconst: each
	// repeated capture-verbatim spelling extracts to one source — the
	// corpusWireKeyModel precedent).
	corpusWireKeyWorkspace     = "workspace"
	corpusWireKeyWorkspacePath = "workspacePath"
	corpusWireKeyWorkspaceKey  = "workspaceKey"
	corpusWireKeyProviderID    = "providerId"
	corpusWireKeyModelID       = "modelId"
	corpusWireKeyOptions       = "options"
	corpusWireKeyLevel         = "reasoningLevel"
	corpusWireKeyPersist       = "persistAsWorkspaceLastUsed"
)

// corpusPatMinter is the runtime-headers-pat row's test-side minter double:
// the ONE surface both the front's PatEnsurer seam and the responder's
// mode-aware source orchestrate (one cache, one lifecycle — the production
// composition). Ensure scripts the mint sequence (authenticate installs
// PAT-A; the re-ask's Ensure installs PAT-B) and Invalidate drops the cache
// — the invalidate-then-remint sequence the row byte-locks (the 05-07
// test-injection discipline; synthetic shapes only).
type corpusPatMinter struct {
	mu    sync.Mutex
	cache hostsvc.Pat
	warm  bool
	mints int
}

// Cached reports the scripted cache state.
func (m *corpusPatMinter) Cached(_ context.Context) (hostsvc.Pat, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.cache, m.warm
}

// Ensure mints the scripted sequence: call 1 (authenticate) installs PAT-A,
// call 2 (the fingerprint re-ask) installs PAT-B.
func (m *corpusPatMinter) Ensure(_ context.Context) (hostsvc.Pat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mints++

	if m.mints == 1 {
		m.cache = hostsvc.Pat{
			Token:        patCorpusTokenA,
			APIKeyID:     patCorpusKeyA,
			Fingerprint:  patCorpusFPA,
			AccountScope: patCorpusScopeA,
		}
	} else {
		m.cache = hostsvc.Pat{
			Token:        patCorpusTokenB,
			APIKeyID:     patCorpusKeyB,
			Fingerprint:  patCorpusFPB,
			AccountScope: patCorpusScopeA,
		}
	}

	m.warm = true

	return m.cache, nil
}

// Invalidate drops the cached token (the source's re-mint trigger).
func (m *corpusPatMinter) Invalidate(string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.warm = false
}

// mintsCount reports the recorded Ensure count.
func (m *corpusPatMinter) mintsCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.mints
}

// driveRuntimeHeadersPat replays the runtime-headers PAT row (plan 06-07
// Task 3, AUTH-02/D-05): the front's coding-plan chain (probe-pass canned →
// overlay push → registry verdict → mint-once through the corpusPatMinter)
// warms the cache, then the turn's send fires TWO scripted runtime-headers
// asks — ask 1 plain (the cached PAT-A answer, zero mints) and ask 2 the
// rejected-fingerprint re-ask (Invalidate + EXACTLY ONE Ensure → PAT-B,
// same scope). The responder wires the mode-aware source over the SAME
// double the front minted through; the mode record selects at ask time.
func driveRuntimeHeadersPat(ctx context.Context, d *exchangeDriver) {
	t := d.t
	patMinter := &corpusPatMinter{}

	//nolint:contextcheck // the Front derives its own process-lifetime base
	// context by design (front.go New — the containedctx rule); the driver's
	// ctx bounds only the back-leg calls.
	front := acpfront.New(d.conn, acpfront.WithAuth(acpfront.AuthConfig{
		Install: codingPlanCorpusInstall(t),
		Probe:   func(_ context.Context) error { return nil }, // canned probe-pass
		Creds:   corpusCreds(t),
		Pat:     patMinter,
		Entitle: func(_ context.Context, family string) bool { return family == cpCorpusAccountFamily },
	}))

	// Hook contract: the mode-aware source is assigned before any frame
	// flows; the unauthenticated None mode would delegate to the static
	// arm's unconfigured posture until authenticate records coding-plan.
	responder := interact.NewResponder(interact.WithRuntimeHeadersKeySource(
		hostsvc.PatHeadersSource(hostsvc.PatHeadersSourceDeps{
			Minter:         patMinter,
			Mode:           front.AuthMode,
			APIKeyFallback: hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey("", "")),
		})))
	d.conn.OnRequest = responder.Serve

	if err := front.Authenticate(ctx, acpfront.AuthMethodZaiCodingPlan); err != nil {
		t.Fatalf("authenticate coding-plan: %v", err)
	}

	if calls := patMinter.mintsCount(); calls != 1 {
		t.Fatalf("authenticate minted %d time(s), want exactly one (mint-once, D-04)", calls)
	}

	// The post-auth preamble over the coding-plan session (create reply
	// carries the account-provider settings; setModel binds the account
	// model).
	drivePatCorpusPreamble(ctx, d)

	if _, err := d.call(ctx, zcode.MethodSessionSend, map[string]any{
		wireKeySessionID: patCorpusSessionID,
		"content":        "Reply with exactly: OK",
	}); err != nil {
		t.Fatalf("session/send: %v", err)
	}

	// The semantic belt: the two asks left exactly the D-05 footprint —
	// authenticate's mint + the one re-mint (expect_answer already
	// byte-compared PAT-A then PAT-B per ask).
	if calls := patMinter.mintsCount(); calls != 2 {
		t.Fatalf("the re-ask left %d total mints, want exactly 2 (authenticate + the one re-mint)", calls)
	}
}

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

// configOptionCorpusModelSelection is the configured selection both
// config-option rows bind with (the session/new setModel follow-up fires;
// its reply IS the confirmed snapshot the picker validates against).
func configOptionCorpusModelSelection() acpfront.ModelSelection {
	return acpfront.ModelSelection{
		ProviderID:     "zai-api",
		ModelID:        "GLM-5.3-Flash",
		ReasoningLevel: corpusCfgLevelHigh,
	}
}

// driveConfigOptionSetModel: the MODEL-02 model-switch corpus row (plan
// 06-06) — the full front composition, the standard preamble, then the SDK
// client's session/set_config_option with a LISTED model value: the
// picker's setModel carries the CURRENT confirmed level (Pitfall 2 — a
// switch must never silently drop it) and the reply confirms the new
// current. The golden byte-locks both setModel payloads (the follow-up's
// and the picker's — model object + the carried options.reasoningLevel +
// explicit persistAsWorkspaceLastUsed:false) and the reply-triggered
// sequencing.
func driveConfigOptionSetModel(ctx context.Context, d *exchangeDriver) {
	driveConfigOptionCorpus(ctx, d, corpusCfgOptionModel, "zai-api/GLM-5.3")
}

// driveConfigOptionThought: the MODEL-02 thought-level corpus row (plan
// 06-06, locked OQ1) — the same preamble, then the thought-level ask: the
// setModel path is the ONLY mutation frame (no dedicated setThoughtLevel
// exists on the v1 wire), the confirmed model rides unchanged, and the
// requested level travels in options.reasoningLevel; the reply confirms the
// new thoughtLevel.current. The golden byte-locks that payload shape.
func driveConfigOptionThought(ctx context.Context, d *exchangeDriver) {
	driveConfigOptionCorpus(ctx, d, corpusCfgOptionThought, "medium")
}

// driveConfigOptionCorpus is the shared body of both config-option rows:
// the FULL front composition (the Front with the configured selection
// wrapped via acpport over a real SDK client — pipe ownership mirrors
// drivePermissionAllow: the agent connection writes toClientW and reads
// toAgentR; the client mirrors), the client-leg preamble (initialize →
// session/new — create, subscribe, the setModel follow-up whose reply IS
// the confirmed snapshot), then the session/set_config_option ask.
func driveConfigOptionCorpus(ctx context.Context, d *exchangeDriver, configID, value string) {
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
	front := acpfront.New(d.conn, acpfront.WithModel(configOptionCorpusModelSelection()))

	// The construction order completes both bindings before any frame
	// flows; the scripted permissionClient never sees an ask on these rows.
	//nolint:dogsled // the front carries all three bindings; none is needed here
	_, _, _ = acpport.NewAgentSideConnection(acpport.NewAgent(front), toClientW, toAgentR)

	client := acp.NewClientSideConnection(&permissionClient{}, toAgentW, toClientR)

	if _, err := client.Initialize(ctx, acp.InitializeRequest{ProtocolVersion: 1}); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	newResp, err := client.NewSession(ctx, acp.NewSessionRequest{Cwd: corpusWorkspacePath})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if newResp.SessionId == "" {
		t.Fatal("session/new returned an empty session id")
	}

	_, err = client.SetSessionConfigOption(ctx, acp.SetSessionConfigOptionRequest{
		ValueId: &acp.SetSessionConfigOptionValueId{
			SessionId: newResp.SessionId,
			ConfigId:  acp.SessionConfigId(configID),
			Value:     acp.SessionConfigValueId(value),
		},
	})
	if err != nil {
		t.Fatalf("set_config_option %s: %v", configID, err)
	}
}

// drivePatCorpusPreamble drives the runtime-headers-pat row's post-auth
// preamble: create (the reply carries the account-provider settings),
// subscribe, and the setModel binding the account model.
func drivePatCorpusPreamble(ctx context.Context, d *exchangeDriver) {
	t := d.t

	if _, err := d.call(ctx, zcode.MethodSessionCreate, map[string]any{
		corpusWireKeyWorkspace: map[string]any{
			corpusWireKeyWorkspacePath: corpusWorkspacePath,
			corpusWireKeyWorkspaceKey:  corpusWorkspacePath,
		},
	}); err != nil {
		t.Fatalf("session/create: %v", err)
	}

	if _, err := d.call(ctx, zcode.MethodSessionSubscribe, map[string]any{
		wireKeySessionID:    patCorpusSessionID,
		wireKeyDeliveryKind: deliveryDesktopContinuous,
	}); err != nil {
		t.Fatalf("session/subscribe: %v", err)
	}

	if _, err := d.call(ctx, zcode.MethodSessionSetModel, map[string]any{
		wireKeySessionID: patCorpusSessionID,
		corpusWireKeyModel: map[string]any{
			corpusWireKeyProviderID: cpCorpusAccountProvider,
			corpusWireKeyModelID:    "GLM-5.3",
			corpusWireKeyOptions:    map[string]any{corpusWireKeyLevel: corpusCfgLevelHigh},
		},
		corpusWireKeyPersist: false,
	}); err != nil {
		t.Fatalf("session/setModel: %v", err)
	}
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

// The round-3 poke rows (plans 06-08's composed chain, EXP-CODINGPLAN-CYCLE-01
// round-3 root cause): the coding-plan probe's create → setModel poke → close
// materializes the account provider — the ask the poke fires is answered by
// the mode-aware source from the cache the EAGER mint armed (D-04 round-3),
// and the account-models patch is the verdict surface the watcher latches.
// The refused twin records the poke's coded refusal as probe evidence — the
// verdict stays the watcher's Await, never the poke's own outcome.

// corpusProbePokeFront composes the poke rows' front + responder: the
// production chain (probe-pass canned, fixture store, the cache-semantic
// corpusPatMinter the front eagerly mints through, the entitlement gate)
// and the mode-aware PAT source over the SAME minter — the arming posture
// the materialization ask depends on.
func corpusProbePokeFront(d *exchangeDriver, patMinter *corpusPatMinter) *acpfront.Front {
	front := acpfront.New(d.conn, acpfront.WithAuth(acpfront.AuthConfig{
		Install:          codingPlanCorpusInstall(d.t),
		Probe:            func(_ context.Context) error { return nil }, // canned probe-pass
		Creds:            corpusCreds(d.t),
		Pat:              patMinter,
		Entitle:          func(_ context.Context, family string) bool { return family == cpCorpusAccountFamily },
		ReadbackDeadline: cpCorpusProbeReadbackDeadline,
	}))

	responder := interact.NewResponder(interact.WithRuntimeHeadersKeySource(
		hostsvc.PatHeadersSource(hostsvc.PatHeadersSourceDeps{
			Minter:         patMinter,
			Mode:           front.AuthMode,
			APIKeyFallback: hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey("", "")),
		})))
	d.conn.OnRequest = responder.Serve // hook contract: before any frame flows

	return front
}

// driveCodingPlanProbeSetModel replays the round-3 happy poke row: the
// front's production chain — push → refresh (the row scripts NO registry
// patch, so the probe runs) → the probe's create → the deterministic
// setModel POKE (the model-selection materialization) whose side effects
// fire the runtime-headers ask (answered by the armed mode-aware source
// from the eagerly minted cache) and the account-models patch (the verdict
// surface the watcher latches) → close. Authenticate answers nil with
// exactly one eager mint.
func driveCodingPlanProbeSetModel(ctx context.Context, d *exchangeDriver) {
	patMinter := &corpusPatMinter{}

	//nolint:contextcheck // the Front derives its own process-lifetime base context by design (front.go New)
	front := corpusProbePokeFront(d, patMinter)

	if err := front.Authenticate(ctx, acpfront.AuthMethodZaiCodingPlan); err != nil {
		d.t.Fatalf("authenticate coding-plan (poke probe): %v", err)
	}

	if calls := patMinter.mintsCount(); calls != 1 {
		d.t.Fatalf("authenticate minted %d time(s), want exactly one "+
			"(the eager mint; the ask served from the cache)", calls)
	}
}

// driveCodingPlanProbeSetModelRefused replays the round-3 negative poke
// row: the poke answers a coded model-not-in-registry refusal — the probe
// records the evidence and STILL closes the probe session; no ask fires,
// the watcher never latches, and authenticate surfaces the honest coded
// read-back failure (never a fabricated outcome from the poke's own
// error).
func driveCodingPlanProbeSetModelRefused(ctx context.Context, d *exchangeDriver) {
	patMinter := &corpusPatMinter{}

	//nolint:contextcheck // the Front derives its own process-lifetime base context by design (front.go New)
	front := corpusProbePokeFront(d, patMinter)

	err := front.Authenticate(ctx, acpfront.AuthMethodZaiCodingPlan)
	if err == nil {
		d.t.Fatal("authenticate answered success on a refused poke — never a fabricated success")
	}

	if !strings.Contains(err.Error(), "read-back") {
		d.t.Fatalf("refused-poke error %v must name the read-back step (the verdict is the watcher's)", err)
	}

	if calls := patMinter.mintsCount(); calls != 1 {
		d.t.Fatalf("authenticate minted %d time(s), want exactly one (the eager mint stands)", calls)
	}
}

// transcriptMethodCount counts the host→fake frames naming method.
func transcriptMethodCount(tr fake.Transcript, method string) int {
	count := 0

	for _, line := range tr.Lines {
		if line.Dir == fake.DirIn && strings.Contains(line.JSON, `"method":"`+method+`"`) {
			count++
		}
	}

	return count
}

// transcriptFirstIndex returns the first transcript index of a frame in dir
// carrying substr (fails loud when absent).
func transcriptFirstIndex(t *testing.T, tr fake.Transcript, dir fake.Dir, substr, what string) int {
	t.Helper()

	for i, line := range tr.Lines {
		if line.Dir == dir && strings.Contains(line.JSON, substr) {
			return i
		}
	}

	t.Fatalf("transcript carries no %s frame (%s)", what, substr)

	return 0
}

// transcriptFirstMethod returns the first transcript index of a host→fake
// frame naming method (fails loud when absent).
func transcriptFirstMethod(t *testing.T, tr fake.Transcript, method string) int {
	t.Helper()

	return transcriptFirstIndex(t, tr, fake.DirIn, `"method":"`+method+`"`, method)
}

// assertProbePokeWireOrder is the shared probe-chain shape: exactly one
// push/refresh/create/setModel/close, ordered refresh → create → setModel →
// close (the create's reply IS the ready gate; the close trails the poke —
// the probe session never leaks). Returns the setModel frame's index.
func assertProbePokeWireOrder(t *testing.T, tr fake.Transcript) int {
	t.Helper()

	for _, method := range []string{
		zcode.MethodProviderUpdateAccountConfig,
		zcode.MethodProviderRefreshRegistry,
		zcode.MethodSessionCreate,
		zcode.MethodSessionSetModel,
		zcode.MethodSessionClose,
	} {
		if count := transcriptMethodCount(tr, method); count != 1 {
			t.Fatalf("transcript records %s ×%d, want exactly one honest probe chain", method, count)
		}
	}

	refreshIdx := transcriptFirstMethod(t, tr, zcode.MethodProviderRefreshRegistry)
	createIdx := transcriptFirstMethod(t, tr, zcode.MethodSessionCreate)
	setModelIdx := transcriptFirstMethod(t, tr, zcode.MethodSessionSetModel)
	closeIdx := transcriptFirstMethod(t, tr, zcode.MethodSessionClose)

	if refreshIdx >= createIdx || createIdx >= setModelIdx || setModelIdx >= closeIdx {
		t.Fatalf("probe order violated: refresh(%d) create(%d) setModel(%d) close(%d)",
			refreshIdx, createIdx, setModelIdx, closeIdx)
	}

	return setModelIdx
}

// assertNoHeadersAsk fails the test when any runtime-headers ask frame is
// on the transcript (the refused-poke lock: a provider that never
// materialized never asks).
func assertNoHeadersAsk(t *testing.T, tr fake.Transcript) {
	t.Helper()

	for _, line := range tr.Lines {
		if strings.Contains(line.JSON, `"method":"interaction/requestProviderRuntimeHeaders"`) {
			t.Fatalf("transcript records a runtime-headers ask after a refused poke: %s", line.JSON)
		}
	}
}

// assertHeadersAskAnswered is the happy row's ask lock: the materialization
// ask fired exactly once and left no unanswered note (an unanswered or
// timed-out ask must fail the row, never pass silently).
func assertHeadersAskAnswered(t *testing.T, tr fake.Transcript) {
	t.Helper()

	asks := 0

	for _, line := range tr.Lines {
		if line.Dir == fake.DirOut &&
			strings.Contains(line.JSON, `"method":"interaction/requestProviderRuntimeHeaders"`) {
			asks++
		}
	}

	if asks != 1 {
		t.Fatalf("transcript records %d runtime-headers ask(s), want exactly one", asks)
	}

	for _, line := range tr.Lines {
		if line.Dir == fake.DirNote && strings.Contains(line.JSON, "requestProviderRuntimeHeaders") {
			t.Fatalf("the materialization ask left a transcript note %s", line.JSON)
		}
	}
}

// assertCodingPlanProbeSetModelTranscript is the happy poke row's semantic
// belt to the byte-lock braces: the shared probe-chain shape, the poke
// carrying the probe session id, the deterministic account Flash model, and
// persistAsWorkspaceLastUsed:false (no options object — the omit-shape),
// and the materialization ask answered exactly once.
func assertCodingPlanProbeSetModelTranscript(t *testing.T, tr fake.Transcript) {
	t.Helper()

	poke := tr.Lines[assertProbePokeWireOrder(t, tr)].JSON

	for _, marker := range []string{
		`"sessionId":"` + cpCorpusSessionID + `"`,
		`"providerId":"` + cpCorpusAccountProvider + `"`,
		`"modelId":"GLM-5.3-Flash"`,
		`"persistAsWorkspaceLastUsed":false`,
	} {
		if !strings.Contains(poke, marker) {
			t.Fatalf("setModel frame %s lacks the poke marker %s", poke, marker)
		}
	}

	if strings.Contains(poke, `"options"`) {
		t.Fatalf("setModel frame %s carries an options object — the omit-shape must ride", poke)
	}

	assertHeadersAskAnswered(t, tr)
}

// assertCodingPlanProbeSetModelRefusedTranscript is the negative row's
// semantic belt: the full probe sequence STILL ran (the refused poke did
// not skip the close) and NO runtime-headers ask fired (the provider never
// materialized).
func assertCodingPlanProbeSetModelRefusedTranscript(t *testing.T, tr fake.Transcript) {
	t.Helper()

	assertProbePokeWireOrder(t, tr)
	assertNoHeadersAsk(t, tr)
}
