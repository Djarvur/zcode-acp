package acpfront_test

// TestCodingPlanFullCycleFake is plan 06-08's composed full cycle at the
// fake tier — the 05-08 composition discipline over the full coding-plan
// stack, every phase capability exercised end-to-end in ONE continuous
// client story over ONE harness, ZERO spend by construction (no real
// account, no real keyring — synthetic fixtures only):
//
//	authenticate{zai-individual-coding-plan}
//	  (the D-02 probe -> the D-03 login read -> the account-overlay push ->
//	  the registry read-back verdict -> the D-04 mint-once),
//	session/new returning configOptions with BOTH entries and the
//	  reply-confirmed currentValue (MODEL-01/D-10),
//	session/set_config_option performing a model switch whose response
//	  re-emits the REPLY-BUILT list (MODEL-02/T-06-19),
//	one tool-less prompt turn streaming to stopReason with the
//	  interaction/requestProviderRuntimeHeaders ask answered from the
//	  scripted PAT source mid-turn (the 06-07 mode-aware composition),
//	and the recorded-wire ordering audit: the auth sequence before
//	  session/new, the setModel mutation before the prompt, and the ask
//	  answered exactly once.
//
// The responder carries the PRODUCTION composition (the mode-aware
// PatHeadersSource over the SAME Pat double the front minted through, plus
// the D-06 cipher seam over the fixture loader) — the cycle proves those
// pieces COMPOSE, the per-piece contracts staying the prior plans' locks.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
	"github.com/Djarvur/zcode-acp/internal/zcode/fake"
)

// The composed-cycle fixture constants (synthetic markers only — the
// scan-secrets gate stays green; the account provider/family/model/PAT
// literals reuse the 06-05 arms' cpFix* fixtures from auth_test.go).
const (
	// cpCycleSessionID/cpCycleTurnID are the scripted session and turn ids.
	cpCycleSessionID = "sess_cpc1"
	cpCycleTurnID    = "turn_cpc1"

	// cpCycleModelPlain/Flash are the provider-qualified value ids the
	// configOptions mirror carries (providerId + "/" + modelId).
	cpCycleModelPlain = cpFixAccountProvider + "/" + cpFixAccountModelA
	cpCycleModelFlash = cpFixAccountProvider + "/" + cpFixAccountModelB

	// cpCycleCipherReqID/cpCycleHdrReqID are the reverse asks' request ids.
	cpCycleCipherReqID = "cp-cycle-cipher-1"
	cpCycleHdrReqID    = "cp-cycle-hdr-1"

	// cpCycleCipherKey/blob/plaintext are the sealed fixture triple the
	// coding-plan-auth-cycle corpus row already proven (the responder's
	// CredentialDecrypt seam decrypts the blob via the fixture loader; the
	// kid is sha256(key) hex[:24] — computed once and frozen there).
	cpCycleCipherKey       = "0123456789abcdef0123456789abcdef"
	cpCycleCipherPlaintext = "fixture-synthetic-credential-not-real"

	// cpCycleCipherBlob is the enc:v2:os fixture blob sealed under
	// cpCycleCipherKey (lll: one concatenated literal).
	cpCycleCipherBlob = "enc:v2:os:3eb1bd439947eb762998e566:MDEyMzQ1Njc4OWFi.Rwp6ooipuq_S1a9bRUi96w." +
		"_8CfTja4_TaGfE3J22EITiGrt71yrA8FUOaudeSW9t0HDQIYng"

	// cpCycleWorkspace is the scripted workspace path (corpus-verbatim).
	cpCycleWorkspace = "/tmp/happy-turn-ws"

	// cpCycleProviderLabel is the ask frame's providerId literal
	// (capture-verbatim — the desktop asks with the zhipu-account class).
	cpCycleProviderLabel = "zhipu-account"

	// cpCycleDeltaText is the streamed reply text (one chunk, tool-less).
	cpCycleDeltaText = "ok then"
)

// errCycleSecondMint guards the mint-once contract at the cycle double (the
// scriptedPat discipline: a second Ensure is a contract violation).
var errCycleSecondMint = errors.New("cycle PAT double: second Ensure is a contract violation")

// cpCycleAccountScope is the double's fixture accountScope (64 hex chars —
// the ask-schema digest shape; strings.Repeat keeps it obviously synthetic).
func cpCycleAccountScope() string {
	return strings.Repeat("f", 64)
}

// cpCycleCreateResult is the create reply's confirmed snapshot: the ACCOUNT
// provider's pair, the per-model reasoning block on GLM-5.3 (the thought
// picker's D-09 source), and the thoughtLevel block present (A4's shape) —
// current = account GLM-5.3, thought level high.
func cpCycleCreateResult() string {
	return fmt.Sprintf(`{
  "session": {"sessionId": %q},
  "protocol": {"name": "ZCode Protocol", "version": 1},
  "settings": {
    "model": {
      "current": {"providerId": %q, "modelId": %q},
      "available": [
        {"ref": {"providerId": %q, "modelId": %q}, "label": %q, "providerLabel": "zai",
         "reasoning": {"levels": [{"value": "high", "label": "High"}, {"value": "medium", "label": "Medium"}],
                       "defaultLevel": "high"}},
        {"ref": {"providerId": %q, "modelId": %q}, "label": %q, "providerLabel": "zai"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "high", "defaultLevel": "high",
      "available": [{"value": "high", "label": "High"}, {"value": "medium", "label": "Medium"}]
    }
  }
}`, cpCycleSessionID, cpFixAccountProvider, cpFixAccountModelA,
		cpFixAccountProvider, cpFixAccountModelA, cpFixAccountModelA,
		cpFixAccountProvider, cpFixAccountModelB, cpFixAccountModelB)
}

// cpCycleSetModelReply is the model switch's afterStateMutation reply — the
// CONFIRMED post-switch snapshot, deliberately different from the create
// snapshot on BOTH surfaces (current = Flash; GLM-5.3 first in the list) so
// a pre-ask rebuild can never pass the re-emit comparison.
func cpCycleSetModelReply() string {
	return fmt.Sprintf(`{
  "session": {"sessionId": %q},
  "settings": {
    "model": {
      "current": {"providerId": %q, "modelId": %q},
      "available": [
        {"ref": {"providerId": %q, "modelId": %q}, "label": %q, "providerLabel": "zai",
         "reasoning": {"levels": [{"value": "high", "label": "High"}, {"value": "medium", "label": "Medium"}],
                       "defaultLevel": "high"}},
        {"ref": {"providerId": %q, "modelId": %q}, "label": %q, "providerLabel": "zai"}
      ]
    },
    "thoughtLevel": {
      "enabled": true, "current": "high", "defaultLevel": "high",
      "available": [{"value": "high", "label": "High"}, {"value": "medium", "label": "Medium"}]
    }
  }
}`, cpCycleSessionID, cpFixAccountProvider, cpFixAccountModelB,
		cpFixAccountProvider, cpFixAccountModelA, cpFixAccountModelA,
		cpFixAccountProvider, cpFixAccountModelB, cpFixAccountModelB)
}

// cpCycleLifecycleEmit builds one session/lifecycle/changed emit.
func cpCycleLifecycleEmit(state string, revision int) fake.SideEffect {
	params := fmt.Sprintf(
		`{"sessionId":%q,"state":%q,"generation":1,"revision":%d,"lastError":null}`,
		cpCycleSessionID, state, revision)

	return fake.SideEffect{Emit: &fake.Emit{
		Method: zcode.MethodSessionLifecycleChanged,
		Params: json.RawMessage(params),
	}}
}

// cpCycleScenario composes the cycle's fake: the coding-plan auth chain's
// push/refresh-with-patch rules, the settings-bearing create (whose reverse
// asks answer the runtime-preferences and cipher arms), the subscribe, the
// switch's setModel reply, and the send whose side effects fire the
// runtime-headers ask MID-TURN with the PAT three-field expect_answer.
func cpCycleScenario() *fake.Scenario {
	return &fake.Scenario{
		Name: "coding-plan-cycle",
		OnRequest: []fake.RequestRule{
			cpAckRule(),
			cpRefreshRule(0, 0, true), // the registry patch latches the watcher (no probe create)
			cpCycleCreateRule(),
			{
				Method: zcode.MethodSessionSubscribe,
				Reply: &fake.Reply{Result: json.RawMessage(fmt.Sprintf(
					`{"sessionId":%q,"eventSeq":0,"events":[]}`, cpCycleSessionID))},
			},
			{
				Method: zcode.MethodSessionSetModel,
				Reply:  &fake.Reply{Result: json.RawMessage(cpCycleSetModelReply())},
			},
			cpCycleSendRule(),
		},
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: fakeMethodNotFound},
		},
		EOFDrainMs: 100,
	}
}

// cpCycleCreateRule is the cycle's create rule: the settings-bearing reply
// plus the create window's side effects — lifecycle starting, the
// runtime-preferences reverse ask, the cipher decrypt reverse ask (the
// fixture blob's expect_answer carries the plaintext), lifecycle ready.
func cpCycleCreateRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionCreate,
		Reply:  &fake.Reply{Result: json.RawMessage(cpCycleCreateResult())},
		SideEffects: []fake.SideEffect{
			cpCycleLifecycleEmit("starting", 1),
			{
				Reverse: &fake.Reverse{
					Method: zcode.MethodRequestRuntimePreferences,
					Params: json.RawMessage(fmt.Sprintf(
						`{"sessionId":%q,"scope":"runtime-materialization"}`,
						cpCycleSessionID)),
					TimeoutMs: 15000,
					ExpectAnswer: &fake.Reply{Result: json.RawMessage(
						`{"nativeSearchEnhancementsEnabled":false,"memoryEnabled":false}`)},
				},
			},
			{
				Reverse: &fake.Reverse{
					Method: zcode.MethodRequestCredentialCipher,
					Params: json.RawMessage(fmt.Sprintf(
						`{"requestId":%q,"operation":"decrypt","value":%q}`,
						cpCycleCipherReqID, cpCycleCipherBlob)),
					TimeoutMs: 15000,
					ExpectAnswer: &fake.Reply{Result: json.RawMessage(fmt.Sprintf(
						`{"requestId":%q,"result":{"ok":true,"value":%q}}`,
						cpCycleCipherReqID, cpCycleCipherPlaintext))},
				},
			},
			cpCycleLifecycleEmit("ready", 2),
		},
	}
}

// cpCycleSendRule is the cycle's send rule: the accepted ack plus the
// turn's side effects — the running patch, turn.started, the
// runtime-headers reverse ask MID-TURN (the PAT three-field expect_answer),
// the reply chunk, turn.completed.
func cpCycleSendRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionSend,
		Reply: &fake.Reply{Result: json.RawMessage(fmt.Sprintf(
			`{"accepted":true,"sessionId":%q,"stateRevision":1}`, cpCycleSessionID))},
		SideEffects: []fake.SideEffect{
			{Emit: &fake.Emit{Method: zcode.MethodStateUpdated, Params: json.RawMessage(fmt.Sprintf(
				`{"patch":{"status":"running"},"reason":"prompt_started","revision":1,`+
					`"scope":"session","sessionId":%q,"type":"state.updated"}`, cpCycleSessionID))}},
			{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(fmt.Sprintf(
				`{"type":"turn.started","seq":4,"turnId":%q,"deliveryKind":"desktop-continuous"}`,
				cpCycleTurnID))}},
			{Reverse: cpCycleHeadersAsk()},
			{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(fmt.Sprintf(
				`{"type":"part.delta","seq":5,"turnId":%q,"messageId":"msg_1","partId":"part_1",`+
					`"field":"text","delta":%q}`, cpCycleTurnID, cpCycleDeltaText))}},
			{Emit: &fake.Emit{Method: zcode.MethodSessionEvent, Params: json.RawMessage(fmt.Sprintf(
				`{"type":"turn.completed","seq":6,"turnId":%q,"deliveryKind":"desktop-continuous"}`,
				cpCycleTurnID))}},
		},
	}
}

// cpCycleHeadersAsk is the mid-turn runtime-headers ask: the post-switch
// model selection and the PAT three-field expect_answer the mode-aware
// source serves from the cache (semantic compare — key order never matters).
func cpCycleHeadersAsk() *fake.Reverse {
	return &fake.Reverse{
		Method: interact.MethodRequestProviderRuntimeHeaders,
		Params: json.RawMessage(fmt.Sprintf(
			`{"requestId":%q,"sessionId":%q,"workspace":{"workspacePath":%q},`+
				`"modelSelection":{"providerId":%q,"modelId":%q},`+
				`"providerId":%q,"reason":"model-request"}`,
			cpCycleHdrReqID, cpCycleSessionID, cpCycleWorkspace,
			cpFixAccountProvider, cpFixAccountModelB, cpCycleProviderLabel)),
		TimeoutMs: 15000,
		ExpectAnswer: &fake.Reply{Result: json.RawMessage(fmt.Sprintf(
			`{"headersApplied":true,"requestAuth":{"apiKey":%q,"apiKeyId":%q,"accountScope":%q}}`,
			cpFixPatToken, cpFixPatKeyID, cpCycleAccountScope()))},
	}
}

// cyclePat is the composed cycle's PAT double: the ONE surface both the
// front's PatEnsurer seam and the responder's mode-aware source orchestrate
// (one cache, one lifecycle — the production composition). The FIRST Ensure
// warms the cache with the fixture PAT; a SECOND Ensure is a contract
// violation (the mint-once lock fails loud — the mid-turn ask must serve
// from the cache with zero extra mints).
type cyclePat struct {
	mu    sync.Mutex
	mints int
	pat   hostsvc.Pat
	warm  bool
}

// Cached reports the cache without minting (the ask-side read).
func (p *cyclePat) Cached(_ context.Context) (hostsvc.Pat, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.pat, p.warm
}

// Ensure mints exactly once, then refuses (the mint-once guard).
func (p *cyclePat) Ensure(_ context.Context) (hostsvc.Pat, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.mints++
	if p.mints > 1 {
		return hostsvc.Pat{}, errCycleSecondMint
	}

	p.pat = hostsvc.Pat{
		Token:        cpFixPatToken,
		APIKeyID:     cpFixPatKeyID,
		Fingerprint:  strings.Repeat("0", 64),
		AccountScope: cpCycleAccountScope(),
	}
	p.warm = true

	return p.pat, nil
}

// Invalidate drops the cached token (unused on the happy script — an
// Invalidate here would mean a fingerprint rejection the scenario never
// scripted).
func (p *cyclePat) Invalidate(string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.warm = false
}

// mintsCount reports the recorded Ensure count.
func (p *cyclePat) mintsCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.mints
}

// cycleCipherLoader is the injected KeyLoader the cipher seam decrypts with
// (the corpus row's fixture key — no keyring anywhere).
type cycleCipherLoader struct{}

// LoadKey returns the fixture 32-byte key.
func (cycleCipherLoader) LoadKey(_ context.Context) ([]byte, error) {
	return []byte(cpCycleCipherKey), nil
}

// newCodingPlanCycleFront composes the full loop over the cycle scenario
// with the complete production stack: the coding-plan auth seams (probe,
// store, mint double, entitlement gate) and the responder whose
// runtime-headers source is the mode-aware PatHeadersSource over the SAME
// double — assigned before any frame flows.
func newCodingPlanCycleFront(t *testing.T) (*FrontHarness, *cyclePat) {
	t.Helper()

	pat := &cyclePat{}

	cfg, _, _ := codingPlanConfig(t)
	cfg.Pat = pat
	cfg.Install = codingPlanFixtureInstall(t)
	cfg.ReadbackDeadline = authReadbackDeadline

	h := ComposeFront(t, cpCycleScenario(), func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithAuth(cfg),
		)

		responder := interact.NewResponder(
			interact.WithPermissionForwarder(
				interact.NewPermissionBridge(front.PermissionForward()).Forward),
			interact.WithRuntimeHeadersKeySource(hostsvc.PatHeadersSource(hostsvc.PatHeadersSourceDeps{
				Minter:         pat,
				Mode:           front.AuthMode,
				APIKeyFallback: hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey("", "")),
			})),
			interact.WithCredentialDecrypt(interact.NewCredentialDecrypt(cycleCipherLoader{})),
		)
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	return h, pat
}

// TestCodingPlanFullCycleFake runs the composed cycle end-to-end under
// -race: one harness, one continuous client story, zero spend.
func TestCodingPlanFullCycleFake(t *testing.T) {
	h, pat := newCodingPlanCycleFront(t)

	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	mustInitialize(ctx, t, h)

	cpCycleAuthenticateLeg(t, h, pat)

	sid := cpCycleSessionNewLeg(ctx, t, h)

	cpCycleSwitchLeg(ctx, t, h, sid)

	cpCyclePromptLeg(ctx, t, h, sid)

	cpCycleOrdering(t, h)
}

// cpCycleAuthenticateLeg is stage 1: authenticate{coding-plan} answers {}
// through the honest chain (the read-back/mint evidence is asserted at the
// cycle's end on the transcript — the harness closes the back leg once).
// The call rides its own bounded context (the authenticate helper's shape —
// the cycle's ctx stays reserved for the story legs that follow).
func cpCycleAuthenticateLeg(t *testing.T, h *FrontHarness, pat *cyclePat) {
	t.Helper()

	authCtx, cancel := h.ExchangeCtx()
	defer cancel()

	if _, err := h.client.Authenticate(authCtx, acp.AuthenticateRequest{
		MethodId: acp.AuthMethodId(acpfront.AuthMethodZaiCodingPlan),
	}); err != nil {
		t.Fatalf("authenticate coding-plan: %v", err)
	}

	if calls := pat.mintsCount(); calls != 1 {
		t.Fatalf("PAT Ensure ran %d time(s) at auth, want exactly one (mint-once after the verdict)", calls)
	}
}

// cpCycleSessionNewLeg is stage 2: session/new returns configOptions with
// BOTH entries and the reply-confirmed currentValues (the create reply's
// snapshot is the confirmed state — no model is configured on this front).
func cpCycleSessionNewLeg(ctx context.Context, t *testing.T, h *FrontHarness) string {
	t.Helper()

	resp, err := h.client.NewSession(ctx, acp.NewSessionRequest{Cwd: toolLifecycleCwd})
	if err != nil {
		t.Fatalf("session/new: %v", err)
	}

	if sid := string(resp.SessionId); sid != cpCycleSessionID {
		t.Fatalf("session/new id = %q, want the scripted %q", sid, cpCycleSessionID)
	}

	// The model option: the confirmed current (the create reply's) and the
	// verbatim account pair (D-08/D-10).
	assertModelOptionCurrent(t, resp, cpCycleModelPlain, []string{cpCycleModelPlain, cpCycleModelFlash})

	// The thought-level option: the BOUND model's reasoning.levels (D-09)
	// and the confirmed high (D-10).
	if len(resp.ConfigOptions) != 2 {
		t.Fatalf("configOptions = %d entries, want 2 (model + thought_level)", len(resp.ConfigOptions))
	}

	thought := resp.ConfigOptions[1].Select
	if thought == nil || thought.Id != optionIDThoughtLevel {
		t.Fatalf("configOptions[1] = %+v, want the thought_level select", resp.ConfigOptions[1])
	}

	if thought.CurrentValue != corpusReasoningLevel {
		t.Fatalf("thought_level currentValue = %q, want the confirmed high (D-10)", thought.CurrentValue)
	}

	if thought.Options.Ungrouped == nil || len(*thought.Options.Ungrouped) != 2 {
		t.Fatalf("thought_level entries = %+v, want the bound model's high/medium levels (D-09)", thought.Options)
	}

	return cpCycleSessionID
}

// cpCycleSwitchLeg is stage 3: the model switch maps to exactly one
// session/setModel carrying the confirmed level, and the response re-emits
// the REPLY-BUILT list (never the pre-ask snapshot).
func cpCycleSwitchLeg(ctx context.Context, t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	resp, err := h.client.SetSessionConfigOption(ctx, setCfgAsk(sid, optionIDModel, cpCycleModelFlash))
	if err != nil {
		t.Fatalf("model switch: %v", err)
	}

	replySettings, parsed := acpfront.ParseSettings(json.RawMessage(cpCycleSetModelReply()))
	if !parsed {
		t.Fatal("the fixture reply must parse")
	}

	assertConfigOptionsMatch(t, resp.ConfigOptions, acpfront.BuildConfigOptions(replySettings))
}

// cpCyclePromptLeg is stage 4: one tool-less turn streams to end_turn with
// the runtime-headers ask answered mid-turn from the PAT cache (the
// expect_answer byte-compare already failed the run on any mismatch — the
// chunk and tool-frame audit here is the streamed-turn shape).
func cpCyclePromptLeg(ctx context.Context, t *testing.T, h *FrontHarness, sid string) {
	t.Helper()

	resp, err := h.client.Prompt(ctx, acp.PromptRequest{
		SessionId: acp.SessionId(sid),
		Prompt:    []acp.ContentBlock{acp.TextBlock("reply ok")},
	})
	if err != nil {
		t.Fatalf("prompt: %v", err)
	}

	if resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stopReason = %q, want %q (the tool-less turn must complete)", resp.StopReason, acp.StopReasonEndTurn)
	}

	chunks, tools := 0, 0

	updates := h.Updates()

	for i := range updates {
		switch {
		case updates[i].Update.AgentMessageChunk != nil:
			chunks++
		case updates[i].Update.ToolCall != nil || updates[i].Update.ToolCallUpdate != nil:
			tools++
		}
	}

	if chunks == 0 {
		t.Fatal("the turn streamed no message chunk — the scripted reply never reached the client")
	}

	if tools != 0 {
		t.Fatalf("the tool-less turn observed %d tool frame(s), want zero", tools)
	}
}

// cpCycleOrdering is stage 5: the composition's ordering truths on the
// recorded wires — the tee's client story in order, the back leg's
// auth-before-new and mutation-before-prompt sequence, the overlay on the
// push frame, and the runtime-headers ask answered exactly once.
func cpCycleOrdering(t *testing.T, h *FrontHarness) {
	t.Helper()

	cpCycleTeeOrder(t, h)
	cpCycleTranscript(t, h)
}

// cpCycleTeeOrder locks the CLIENT story order on the ACP tee:
// authenticate -> session/new -> set_config_option -> prompt.
func cpCycleTeeOrder(t *testing.T, h *FrontHarness) {
	t.Helper()

	lines := h.ACPLines()

	authIdx := teeLineIndex(t, lines, "out", `"method":"authenticate"`, "authenticate request")
	newIdx := teeLineIndex(t, lines, "out", `"method":"session/new"`, "session/new request")
	cfgIdx := teeLineIndex(t, lines, "out", `"method":"session/set_config_option"`, "set_config_option request")
	promptIdx := teeLineIndex(t, lines, "out", `"method":"session/prompt"`, "prompt request")

	if authIdx >= newIdx || newIdx >= cfgIdx || cfgIdx >= promptIdx {
		t.Fatalf("client story order violated: authenticate(%d) -> session/new(%d) -> "+
			"set_config_option(%d) -> prompt(%d) must be strictly increasing",
			authIdx, newIdx, cfgIdx, promptIdx)
	}
}

// cpCycleTranscript locks the BACK leg's recorded sequence: exactly one
// push/refresh with the account overlay on the push frame, the auth chain
// completing before the cycle's create, exactly one setModel (the switch —
// no session/new preamble setModel) preceding the send, and the
// runtime-headers ask fired exactly once (the expect_answer byte-compare
// already proved the answer's content; a mismatch fails the run outright).
func cpCycleTranscript(t *testing.T, h *FrontHarness) {
	t.Helper()

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertCodingPlanHappySequence(t, tr)

	refreshIdx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry)
	createIdx := transcriptIndex(t, tr, "in", `"method":"session/create"`, "create request")
	if refreshIdx[0] >= createIdx {
		t.Fatalf("cycle order violated: authenticate's refresh(%d) must precede session/new's create(%d)",
			refreshIdx[0], createIdx)
	}

	assertSingleSetModel(t, tr) // exactly one mutation frame, BEFORE the send

	setModelIdx := transcriptMethodIndexes(tr, zcode.MethodSessionSetModel)
	sendIdx := transcriptIndex(t, tr, "in", `"method":"session/send"`, "send request")
	if setModelIdx[0] >= sendIdx {
		t.Fatalf("cycle order violated: the switch's setModel(%d) must precede the prompt's send(%d)",
			setModelIdx[0], sendIdx)
	}

	asks := transcriptIndexes(tr, string(fake.DirOut), `"method":"`+interact.MethodRequestProviderRuntimeHeaders+`"`)
	if len(asks) != 1 {
		t.Fatalf("transcript records %d runtime-headers ask(s), want exactly one "+
			"(answered once, from the cache — never re-asked)", len(asks))
	}

	if notes := transcriptIndexes(tr, string(fake.DirNote), "requestProviderRuntimeHeaders"); len(notes) != 0 {
		t.Fatalf("the runtime-headers ask left %d transcript note(s) — an unanswered or timed-out ask "+
			"must fail the cycle, never pass silently", len(notes))
	}

	if mints := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig); len(mints) != 1 {
		t.Fatalf("transcript records push×%d, want exactly one honest chain run", len(mints))
	}
}
