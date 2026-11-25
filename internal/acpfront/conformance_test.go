package acpfront_test

// TestConformanceV1Arc is the phase-7 conformance suite (QA-01, D-04) —
// the ONE named scenario the done-v1 bar names: the full v1 arc composed
// continuously over ONE harness in the real client's order —
//
//	initialize → authenticate → session/new → prompt streaming
//	  (tool calls + permission decision) → cancel
//
// Phases 3-6 proved every leg separately (the 34-scenario back-leg corpus,
// TestFullCycle's composed arc, the cancel rows, the permission bridge
// locks); this suite proves they COMPOSE, byte-locked on the recorded wire
// artifacts (the ACP tee, the fake transcript — never internals), and it
// is the regression anchor the milestone closes on.
//
// The composition rides the proven full-cycle corpus row for legs 1-4 and
// extends it, entirely within this file, with the cancel leg's rules (the
// proven cancel-mid-turn row's shapes) — the fake's own rule-takeover
// semantics route the arc's SECOND send to the appended rule once the
// corpus row's times:1 budget is spent. No corpus or shared harness file
// is touched: the arc is additive by construction (D-04's prohibition).

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The arc's byte-lock surface: tee-frame probe substring constants naming
// each leg's load-bearing frames (the fullcycle_test.go pattern), plus the
// back-leg wire probes for the appended cancel rules.
const (
	// conformanceScenario is the proven corpus row the arc rides for legs
	// 1-4 (the honest auth chain, the D-03 preamble, the featured prompt
	// turn); the cancel-leg rules append programmatically below.
	conformanceScenario = "full-cycle"

	// arcTurn2ID is the second prompt's scripted turn id — distinct from
	// turn 1's corpus turn_fix1.
	arcTurn2ID = "turn_fix2"

	// Leg 1 — initialize's honest capability reply: protocolVersion 1, the
	// options and resume sessionCapabilities opt-ins, and both authMethod ids.
	arcTeeVersionOne  = `"protocolVersion":1`
	arcTeeSessionCaps = `"sessionCapabilities":{"options":{},"resume":{}}`
	arcTeeAuthKeyID   = `"zai-api-key"`
	arcTeeAuthPlanID  = `"zai-individual-coding-plan"`

	// Legs 4/5 — the two prompt resolutions: turn 1 answers end_turn,
	// turn 2 answers cancelled (the CORE-05 lock carried into the arc).
	arcTeeEndTurn   = `"end_turn"`
	arcTeeCancelled = `"cancelled"`

	// Back-leg wire probes — the appended rules' shapes and the stop's
	// honest cancelled outcome patch.
	arcWireSendProbe      = `"method":"session/send"`
	arcWireStopProbe      = `"method":"session/stop"`
	arcWireCancelledPatch = `"status":"cancelled"`
)

// TestConformanceV1Arc runs the named conformance suite end-to-end under
// -race: ONE harness, ONE continuous client story, legs in exactly the
// order a real client plays them. Every leg asserts its load-bearing wire
// frames — a missing or out-of-order leg fails the suite, never a silent
// pass (a zero-frame run is a reported failure, the TestEOFSignature
// posture).
func TestConformanceV1Arc(t *testing.T) {
	h := newConformanceFront(t)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	// Leg 1: initialize — the honest capability reply on the raw tee frame.
	assertConformanceInitialize(ctx, t, h)

	// Leg 2: authenticate — the honest chain answers {} on the wire (the
	// read-back evidence is a transcript-level lock at the arc's end).
	fullCycleAuthenticate(ctx, t, h)

	// Leg 3: session/new — the D-03 gate (create → subscribe → setModel)
	// is likewise a transcript-level lock at the arc's end (the harness's
	// transcript accessor closes the back leg, so transcript locks run
	// once, after the whole story).
	sid := fullCycleOpenSession(ctx, t, h)

	// Leg 4: the featured prompt turn — streamed text deltas, the
	// tool_call/tool_call_update pair with title/kind/status, the bridged
	// session/request_permission answered ALLOW, flush-before-end_turn
	// (the proven full-cycle leg asserts, ridden whole).
	fullCyclePrompt(ctx, t, h, sid)

	// Leg 5: a second prompt cancelled mid-turn — exactly-once cancelled
	// resolution and no -32800 anywhere on the tee.
	promptDone := startActiveTurn(ctx, t, h, sid)

	if err := h.client.Cancel(ctx, acp.CancelNotification{SessionId: acp.SessionId(sid)}); err != nil {
		t.Fatalf("cancel: %v", err)
	}

	assertConformanceCancelOutcome(t, h, waitPromptOutcome(t, promptDone))

	// The recorded-wire audit (closes the back leg once, last).
	conformanceTranscriptAudit(t, h, sid)
}

// newConformanceFront composes the arc harness: the full-cycle corpus row
// carrying legs 1-4, extended with the cancel-leg rules appended
// programmatically (conformanceCancelLegRules), over the complete phase
// stack the production wiring carries — the model selection, the
// authenticate machinery over the synthetic fixture key, and the responder
// whose PermissionForwarder is the real front bridge, assigned BEFORE any
// frame flows (the phase-3 construction rule). The scripted client answers
// allow_once (the arc's permission decision).
func newConformanceFront(t *testing.T) *FrontHarness {
	t.Helper()

	sc := mustLoadScenario(t, conformanceScenario)
	sc.OnRequest = append(sc.OnRequest, conformanceCancelLegRules()...)

	h := ComposeFront(t, sc, func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithModel(corpusModelSelection()),
			acpfront.WithAuth(acpfront.AuthConfig{
				Keys:       hostsvc.NewStaticKey(syntheticAuthKey, authFixKeyID),
				Install:    authFixtureInstall(t),
				WantModels: authFixWantModels(),
			}),
		)

		responder := interact.NewResponder(interact.WithPermissionForwarder(
			interact.NewPermissionBridge(front.PermissionForward()).Forward,
		))
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	h.rec.SetPermissionScript(PermissionScript{SelectedOptionID: permAllowOptionID})

	return h
}

// conformanceCancelLegRules builds the appended cancel-leg rules: the
// second send (running + turn.started on arcTurn2ID, NO terminal event —
// the cancel-mid-turn row's parkable shape, routed here by the fake's
// rule-takeover semantics once the corpus row's times:1 budget is spent)
// and the session/stop rule ({} reply + the cancelled state patch — the
// corpus cancel row's honest cancelled outcome). Payloads are the
// capture-verbatim compact shapes; one line per envelope.
func conformanceCancelLegRules() []fake.RequestRule {
	return []fake.RequestRule{
		{
			Method: zcode.MethodSessionSend,
			Reply: &fake.Reply{Result: json.RawMessage(
				`{"accepted":true,"sessionId":"sess_fix1","stateRevision":2}`)},
			SideEffects: []fake.SideEffect{
				{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: json.RawMessage(
					`{"patch":{"status":"running"},"reason":"prompt_started","revision":2,` +
						`"scope":"session","sessionId":"sess_fix1","type":"state.updated"}`)}},
				{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(
					`{"type":"turn.started","seq":4,"turnId":"` + arcTurn2ID +
						`","deliveryKind":"desktop-continuous"}`)}},
			},
		},
		{
			Method: zcode.MethodSessionStop,
			Reply:  &fake.Reply{Result: json.RawMessage(`{}`)},
			SideEffects: []fake.SideEffect{
				{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: json.RawMessage(
					`{"patch":{"status":"cancelled"},"reason":"session_stopped","revision":3,` +
						`"scope":"session","sessionId":"sess_fix1","type":"state.updated"}`)}},
			},
		},
	}
}

// assertConformanceInitialize runs the arc's initialize leg and locks the
// honest capability reply on the raw tee frame: the D-02 probe shape
// (assertInitializeReport, ridden whole), plus the three surfaces the arc
// names — protocolVersion 1, the options and resume sessionCapabilities
// opt-ins, and both authMethod ids.
func assertConformanceInitialize(ctx context.Context, t *testing.T, h *FrontHarness) {
	t.Helper()

	assertInitializeReport(ctx, t, h)

	lines := h.ACPLines()
	initIdx := teeLineIndex(t, lines, "in", arcTeeVersionOne, "initialize response")
	initRaw := lines[initIdx].JSON

	if !strings.Contains(initRaw, arcTeeSessionCaps) {
		t.Fatalf("initialize response carries no options-plus-resume sessionCapabilities opt-ins: %s", initRaw)
	}

	if !strings.Contains(initRaw, arcTeeAuthKeyID) || !strings.Contains(initRaw, arcTeeAuthPlanID) {
		t.Fatalf("initialize response does not advertise both authMethod ids: %s", initRaw)
	}
}

// assertConformanceCancelOutcome locks the arc's cancel leg on the tee
// (CORE-05 carried into the arc): the second prompt resolved exactly once
// as the cancelled SUCCESS, after turn 1's end_turn; exactly one response
// per arc turn; no update frame after the cancelled response (the spec
// rule: post-cancel updates MUST precede the response); and no -32800
// anywhere on the wire — cancelled is a success stopReason, never a
// RequestCancelled error.
func assertConformanceCancelOutcome(t *testing.T, h *FrontHarness, outcome promptOutcome) {
	t.Helper()

	if outcome.err != nil {
		t.Fatalf("cancelled prompt must answer as a success response, got error: %v", outcome.err)
	}

	if outcome.resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stopReason = %q, want %q", outcome.resp.StopReason, acp.StopReasonCancelled)
	}

	lines := h.ACPLines()

	endIdxs := teeLineIndexes(lines, "in", arcTeeEndTurn)
	cancelIdxs := teeLineIndexes(lines, "in", arcTeeCancelled)

	if len(endIdxs) != 1 || len(cancelIdxs) != 1 {
		t.Fatalf("tee records end_turn×%d cancelled×%d prompt responses, want exactly one of each "+
			"(the arc's two turns each resolve exactly once)", len(endIdxs), len(cancelIdxs))
	}

	if cancelIdxs[0] < endIdxs[0] {
		t.Fatalf("the cancelled response (line %d) precedes the end_turn response (line %d) — "+
			"the arc's turns must resolve in the client's order", cancelIdxs[0], endIdxs[0])
	}

	if stopIdxs := teeLineIndexes(lines, "in", `"stopReason"`); len(stopIdxs) != 2 {
		t.Fatalf("tee records %d prompt response frames, want exactly 2 (one per arc turn, exactly-once each)",
			len(stopIdxs))
	}

	for i, line := range lines[cancelIdxs[0]+1:] {
		if line.Dir == "in" && strings.Contains(line.JSON, teeUpdateMark) {
			t.Fatalf("update frame at line %d streams after the cancelled response (line %d) — "+
				"post-cancel updates MUST precede the response", cancelIdxs[0]+1+i, cancelIdxs[0])
		}
	}

	assertNoRequestCancelled(t, lines)
}

// conformanceTranscriptAudit is the arc's recorded-wire audit: the honest
// authenticate chain (push → refresh, before the session), the D-03
// session/new sequence (create → subscribe → setModel → send), the
// permission reply's embedded response bytes, and the arc wire — exactly
// the two prompt sends with one session/stop cancelling the second turn.
func conformanceTranscriptAudit(t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertFullCycleAuthEvidence(t, tr)
	assertBackLegNewSessionSequence(t, tr, sid)
	assertPermissionReply(t, tr)
	assertConformanceArcWire(t, tr)
}

// assertConformanceArcWire locks the arc's back-leg story: exactly the two
// prompt sends, exactly one session/stop (the cancel leg) following the
// SECOND send, and exactly one cancelled state patch — the honest
// cancelled outcome the tee's stopReason mirrors.
func assertConformanceArcWire(t *testing.T, tr fake.Transcript) {
	t.Helper()

	sends := transcriptIndexes(tr, "in", arcWireSendProbe)
	if len(sends) != 2 {
		t.Fatalf("back leg records %d session/send frames, want exactly 2 (the arc's two prompts)", len(sends))
	}

	stops := transcriptIndexes(tr, "in", arcWireStopProbe)
	if len(stops) != 1 {
		t.Fatalf("back leg records %d session/stop frames, want exactly 1 (the cancel leg)", len(stops))
	}

	if stops[0] < sends[1] {
		t.Fatalf("wire order violated: the cancel leg's stop(%d) must follow the second send(%d)",
			stops[0], sends[1])
	}

	if patches := transcriptIndexes(tr, string(fake.DirOut), arcWireCancelledPatch); len(patches) != 1 {
		t.Fatalf("back leg emits %d cancelled state patches, want exactly 1 (the honest cancelled outcome)",
			len(patches))
	}
}
