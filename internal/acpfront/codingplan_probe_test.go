package acpfront_test

// The coding-plan probe round-3 locks (EXP-CODINGPLAN-CYCLE-01 attempts 1-3
// root cause): the authenticate probe's session/create + close alone never
// materializes the account provider — no model selection, no
// interaction/requestProviderRuntimeHeaders ask, no PAT answer, no
// account-model surface, and the read-back waits forever. The successor
// probe pokes session/setModel with the DETERMINISTIC want model after
// create-ready (the pick: the coding-plan account family from the want list,
// the -Flash preference, else the family's first; empty want pokes nothing),
// and BOTH poke outcomes are handled: ok → the watcher proceeds to Await
// (a subsequent ask is answered from the PAT cache the eager mint armed);
// a coded refusal is recorded as probe evidence — the verdict remains the
// watcher's Await, never the poke's own success or failure. The eager mint
// (D-04 round-3: mint at authenticate, after the entitlement gate) runs
// BEFORE any wire traffic so the materialization ask answers instantly from
// the cache. The api-key probe path stays untouched (locked here too).

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
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

// The probe arms' fixture constants (synthetic markers only).
const (
	// cpProbeSessionID is the session id the scripted probe create replies.
	cpProbeSessionID = "sess_cp_probe"

	// cpProbeAskID is the scripted runtime-headers ask's request id.
	cpProbeAskID = "cp-probe-hdr-1"

	// cpProbeModelFlash is the poke target the picker must derive from the
	// fixture want list (the family's -Flash entry).
	cpProbeModelFlash = cpFixAccountProvider + "/" + cpFixAccountModelB
)

// errCannedMintFailure is the eager-mint arm's injected mint verdict (err113:
// a static error, never dynamically constructed).
var errCannedMintFailure = errors.New("canned mint failure")

// failingPat is the mint seam double whose Ensure always fails (the eager
// mint's pre-traffic coded arm).
type failingPat struct{}

// Ensure replays the canned failure.
func (failingPat) Ensure(context.Context) (hostsvc.Pat, error) {
	return hostsvc.Pat{}, errCannedMintFailure
}

// TestProbeWantModel locks the deterministic poke-target picker (the want
// list's providerId/modelId entries): the coding-plan account family when
// present, within it the first modelId ending in the Flash suffix, else the
// family's first; an empty family selection falls back to the first
// well-formed want entry overall; an empty want list answers "" (no poke —
// nothing honest to bind).
func TestProbeWantModel(t *testing.T) {
	const flashSuffix = "-Flash"

	familyA := cpFixAccountProvider + "/" + cpFixAccountModelA     // GLM-5.3
	familyFlash := cpFixAccountProvider + "/" + cpFixAccountModelB // GLM-5.3-Flash
	apiFlash := "zai-api/GLM-5.3-Flash"
	apiPlain := "zai-api/GLM-5.3"

	tests := []struct {
		name string
		want []string
		exp  string
	}{
		{"flash preferred within the family", []string{familyA, familyFlash}, familyFlash},
		{"flash preference stays family-scoped", []string{apiFlash, familyA, familyFlash}, familyFlash},
		{"no flash in the family falls to the family's first", []string{familyA}, familyA},
		{"no flash and a foreign entry first still picks the family", []string{apiFlash, familyA}, familyA},
		{"empty family falls back to the first want entry", []string{apiFlash, apiPlain}, apiFlash},
		{"malformed no-slash entries are skipped in the fallback", []string{"garbage", apiPlain}, apiPlain},
		{"empty want list answers empty (no poke)", []string{}, ""},
		{"nil want list answers empty (no poke)", nil, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := acpfront.ProbeWantModel(tc.want)

			if got != tc.exp {
				t.Fatalf("ProbeWantModel(%v) = %q, want %q (deterministic pick; flash suffix %q)",
					tc.want, got, tc.exp, flashSuffix)
			}
		})
	}

	// Determinism: the same input answers identically across calls (the pick
	// is pure selection over the caller's order, never a map iteration).
	for range 5 {
		if got := acpfront.ProbeWantModel([]string{familyA, familyFlash}); got != familyFlash {
			t.Fatalf("ProbeWantModel lost determinism: %q", got)
		}
	}
}

// cpProbeLifecycleEmit builds one session/lifecycle/changed emit for the
// probe session (the create window's field shape).
func cpProbeLifecycleEmit(state string, revision int) fake.SideEffect {
	params := `{"sessionId":"` + cpProbeSessionID + `","state":"` + state +
		`","generation":1,"revision":` + strconv.Itoa(revision) + `,"lastError":null}`

	return fake.SideEffect{Emit: &fake.Emit{
		Method: zcode.MethodSessionLifecycleChanged,
		Params: json.RawMessage(params),
	}}
}

// cpProbeCreateRule is the probe create: the field-shaped reply carrying ONLY
// the builtin personal model (the account provider is NOT yet materialized —
// the gap the poke closes), plus the lifecycle emits.
func cpProbeCreateRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionCreate,
		Reply: &fake.Reply{Result: json.RawMessage(`{"session":{"sessionId":"` + cpProbeSessionID + `"},` +
			`"settings":{"model":{"available":[` +
			`{"ref":{"providerId":"zai-api","modelId":"GLM-5.3-Flash"}}]}}}`)},
		SideEffects: []fake.SideEffect{
			cpProbeLifecycleEmit("starting", 1),
			cpProbeLifecycleEmit("ready", 2),
		},
	}
}

// cpProbeHeadersAsk is the materialization ask the poke fires (the 06-07
// machinery's shape): the post-poke model selection, the PAT three-field
// expect_answer the mode-aware source serves from the armed cache.
func cpProbeHeadersAsk() fake.SideEffect {
	return fake.SideEffect{Reverse: &fake.Reverse{
		Method: interact.MethodRequestProviderRuntimeHeaders,
		Params: json.RawMessage(`{"requestId":"` + cpProbeAskID + `","sessionId":"` + cpProbeSessionID + `",` +
			`"workspace":{"workspacePath":"/tmp/happy-turn-ws"},` +
			`"modelSelection":{"providerId":"` + cpFixAccountProvider + `","modelId":"` + cpFixAccountModelB + `"},` +
			`"providerId":"zhipu-account","reason":"model-request"}`),
		TimeoutMs: 2000,
		ExpectAnswer: &fake.Reply{Result: json.RawMessage(
			`{"headersApplied":true,"requestAuth":{"apiKey":"` + cpFixPatToken +
				`","apiKeyId":"` + cpFixPatKeyID + `","accountScope":"` + strings.Repeat("f", 64) + `"}}`)},
	}}
}

// cpProbeSetModelRule is the poke's rule: ok reply plus the materialization
// side effects (the runtime-headers ask, then the account-models registry
// patch — the surface the watcher latches).
func cpProbeSetModelRule() fake.RequestRule {
	ref := func(modelID string) string {
		return `{"ref":{"providerId":"` + cpFixAccountProvider + `","modelId":"` + modelID + `"}}`
	}

	reply := `{"session":{"sessionId":"` + cpProbeSessionID + `"},` +
		`"settings":{"model":{"current":` + ref(cpFixAccountModelB) + `,` +
		`"available":[` + ref(cpFixAccountModelA) + `,` + ref(cpFixAccountModelB) + `]}}}`

	return fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply:  &fake.Reply{Result: json.RawMessage(reply)},
		SideEffects: []fake.SideEffect{
			cpProbeHeadersAsk(),
			{Emit: &fake.Emit{
				Method: zcode.MethodStateUpdated,
				Params: json.RawMessage(cpRefreshStateUpdated),
			}},
		},
	}
}

// cpProbeRefusedSetModelRule is the poke's coded refusal (model not in the
// registry — -32002 resource-not-found class): NO side effects, the
// provider never materializes.
func cpProbeRefusedSetModelRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionSetModel,
		Reply: &fake.Reply{Error: &zcode.WireError{
			Code:    zcode.CodeResourceNotFound,
			Message: "model not in registry: " + cpProbeModelFlash,
		}},
	}
}

// cpProbeCloseRule scripts the probe session/close (the field-verified shape).
func cpProbeCloseRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionClose,
		Reply:  &fake.Reply{Result: json.RawMessage(`{"closed":true}`)},
	}
}

// newCodingPlanProbeFront composes the loop over the probe rules with the
// PRODUCTION responder composition: the mode-aware PAT source over the SAME
// cache-semantic double the front mints through (the 06-07 machinery the
// arming posture depends on).
func newCodingPlanProbeFront(t *testing.T, rules []fake.RequestRule) (*FrontHarness, *acpfront.Front, *scriptedPat) {
	t.Helper()

	cfg, _, pat := codingPlanConfig(t)
	cfg.Install = codingPlanFixtureInstall(t)
	cfg.ReadbackDeadline = authReadbackDeadline

	var front *acpfront.Front

	h := ComposeFront(t, authScenario(rules), func(conn *zcode.Conn) acp.Agent {
		front = acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithAuth(cfg),
		)

		responder := interact.NewResponder(interact.WithRuntimeHeadersKeySource(
			hostsvc.PatHeadersSource(hostsvc.PatHeadersSourceDeps{
				Minter:         pat,
				Mode:           front.AuthMode,
				APIKeyFallback: hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey("", "")),
			})))
		conn.OnRequest = responder.Serve // hook contract: before any frame flows

		return acpport.NewAgent(front)
	})

	return h, front, pat
}

// TestCodingPlanProbePoke locks the successor probe's wire behavior.
func TestCodingPlanProbePoke(t *testing.T) {
	t.Run("create -> setModel(account Flash) -> close latches the verdict", testCodingPlanProbePokeHappy)
	t.Run("a refused poke is evidence, never a verdict", testCodingPlanProbePokeRefused)
	t.Run("the eager mint precedes every wire traffic", testCodingPlanMintEager)
	t.Run("the api-key probe stays create -> close, no poke", testAPIKeyProbeUnchanged)
}

// assertProbePokeSequence is the probe-shape lock shared by the poke arms:
// exactly one refresh → create → setModel → close in wire order (the
// create's reply IS the ready gate; the close still trails the poke — the
// probe session never leaks). Returns the setModel frame's transcript index.
func assertProbePokeSequence(t *testing.T, tr fake.Transcript) int {
	t.Helper()

	refreshIdx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry)
	createIdx := transcriptMethodIndexes(tr, zcode.MethodSessionCreate)
	setModelIdx := transcriptMethodIndexes(tr, zcode.MethodSessionSetModel)
	closeIdx := transcriptMethodIndexes(tr, zcode.MethodSessionClose)

	if len(createIdx) != 1 || len(setModelIdx) != 1 || len(closeIdx) != 1 {
		t.Fatalf("transcript records create×%d setModel×%d close×%d, want exactly one probe poke",
			len(createIdx), len(setModelIdx), len(closeIdx))
	}

	if len(refreshIdx) != 1 || refreshIdx[0] >= createIdx[0] ||
		createIdx[0] >= setModelIdx[0] || setModelIdx[0] >= closeIdx[0] {
		t.Fatalf("probe order violated: refresh(%v) create(%v) setModel(%v) close(%v)",
			refreshIdx, createIdx, setModelIdx, closeIdx)
	}

	return setModelIdx[0]
}

// testCodingPlanProbePokeHappy is the happy poke lock: refresh emits nothing
// (the watcher is unlatched), the probe runs create → setModel → close, the
// poke carries the deterministic account Flash model with
// persistAsWorkspaceLastUsed:false and NO options object (the probe session
// has no confirmed level — the omit-shape), the materialization ask is
// answered from the armed cache (the arming posture proven in-flight), the
// account patch latches the watcher, and authenticate answers nil.
func testCodingPlanProbePokeHappy(t *testing.T) {
	h, front, pat := newCodingPlanProbeFront(t, []fake.RequestRule{
		cpAckRule(),
		cpRefreshRule(0, 0, false),
		cpProbeCreateRule(),
		cpProbeSetModelRule(),
		cpProbeCloseRule(),
	})

	if _, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan); err != nil {
		t.Fatalf("authenticate coding-plan with the poke probe: %v", err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// The poke's params: the deterministic account Flash model, the probe
	// session id, persistAsWorkspaceLastUsed:false, and NO options object
	// (the probe session has no confirmed level — omit-shape).
	poke := tr.Lines[assertProbePokeSequence(t, tr)].JSON

	for _, marker := range []string{
		`"sessionId":"` + cpProbeSessionID + `"`,
		`"providerId":"` + cpFixAccountProvider + `"`,
		`"modelId":"` + cpFixAccountModelB + `"`,
		`"persistAsWorkspaceLastUsed":false`,
	} {
		if !strings.Contains(poke, marker) {
			t.Fatalf("setModel frame %s lacks the poke marker %s", poke, marker)
		}
	}

	if strings.Contains(poke, `"options"`) {
		t.Fatalf("setModel frame %s carries an options object — the probe session "+
			"has no confirmed level, the omit-shape must ride", poke)
	}

	// The materialization ask fired exactly once and was ANSWERED (no
	// transcript notes — an unanswered or timed-out ask must fail loud).
	asks := transcriptIndexes(tr, string(fake.DirOut), `"method":"`+interact.MethodRequestProviderRuntimeHeaders+`"`)
	if len(asks) != 1 {
		t.Fatalf("transcript records %d runtime-headers ask(s), want exactly one", len(asks))
	}

	if notes := transcriptIndexes(tr, string(fake.DirNote), "requestProviderRuntimeHeaders"); len(notes) != 0 {
		t.Fatalf("the materialization ask left %d transcript note(s) — the mode-aware source "+
			"must answer it from the armed cache during the probe", len(notes))
	}

	// The eager mint armed the cache: exactly one mint, and the success
	// recorded the coding-plan mode.
	if calls := pat.count(); calls != 1 {
		t.Fatalf("PAT minted %d time(s), want exactly one (the eager mint; the ask served from the cache)", calls)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeCodingPlan {
		t.Fatalf("auth mode = %q, want %q after the poke-proven success", mode, acpfront.AuthModeCodingPlan)
	}
}

// testCodingPlanProbePokeRefused is the honest-negative lock: the poke
// answers a coded model-not-in-registry refusal — the probe records the
// evidence and STILL closes the probe session; the verdict remains the
// watcher's Await (which times out): the coded read-back failure answers,
// the prior mode is restored (nothing recorded), and the eager mint stays
// exactly one. The poke's failure never fabricates success or failure by
// itself.
func testCodingPlanProbePokeRefused(t *testing.T) {
	h, front, pat := newCodingPlanProbeFront(t, []fake.RequestRule{
		cpAckRule(),
		cpRefreshRule(0, 0, false),
		cpProbeCreateRule(),
		cpProbeRefusedSetModelRule(),
		cpProbeCloseRule(),
	})

	_, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan)

	rerr := wireErrOf(t, err)
	if !strings.Contains(rerr.Message, "read-back") {
		t.Fatalf("refused-poke message %q must name the read-back step (the verdict is the watcher's, "+
			"never the poke's)", rerr.Message)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	// Both outcomes handled: the full sequence held, the close STILL
	// trailing the refused poke.
	assertProbePokeSequence(t, tr)

	// A refused poke never materializes: no runtime-headers ask on the wire.
	asks := transcriptIndexes(tr, string(fake.DirOut), `"method":"`+interact.MethodRequestProviderRuntimeHeaders+`"`)
	if len(asks) != 0 {
		t.Fatalf("transcript records %d runtime-headers ask(s) after a refused poke, want zero", len(asks))
	}

	if calls := pat.count(); calls != 1 {
		t.Fatalf("PAT minted %d time(s), want exactly one (the eager mint stands even when the verdict fails)", calls)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeNone {
		t.Fatalf("auth mode = %q after the refused-poke failure, want none recorded", mode)
	}
}

// testCodingPlanMintEager is the D-04 round-3 ordering lock: the mint runs
// BEFORE any wire traffic — a mint failure answers the coded mint error with
// NOTHING on the transcript (no push, no refresh, no probe session), after
// the probe+login prologue ran.
func testCodingPlanMintEager(t *testing.T) {
	cfg, probe, _ := codingPlanConfig(t)
	cfg.Pat = failingPat{}
	cfg.Install = codingPlanFixtureInstall(t)
	cfg.ReadbackDeadline = authReadbackDeadline

	h, front := newCodingPlanFront(t, []fake.RequestRule{cpAckRule(), cpRefreshRule(0, 0, false)}, cfg)

	_, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan)

	rerr := wireErrOf(t, err)
	if !strings.Contains(rerr.Message, "PAT mint") {
		t.Fatalf("mint-failure message %q must name the PAT mint step", rerr.Message)
	}

	// The prologue RAN (the probe precedes the mint — D-02's ordering).
	if calls := probe.count(); calls != 1 {
		t.Fatalf("keyring probe ran %d time(s), want exactly one", calls)
	}

	// The eager mint precedes every wire traffic: the failure left the
	// transcript EMPTY of chain frames.
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertNoCodingPlanTraffic(t, tr)

	if idx := transcriptMethodIndexes(tr, zcode.MethodSessionCreate); len(idx) != 0 {
		t.Fatalf("transcript records a probe create at %v — the mint failure must answer "+
			"before the probe session ever spawns", idx)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeNone {
		t.Fatalf("auth mode = %q after a mint failure, want none recorded", mode)
	}
}

// testAPIKeyProbeUnchanged locks the regression face: the api-key arm's probe
// stays the bare create → close pair (NO setModel poke — the poke is
// coding-plan-only), succeeding via the create reply's surfaces as before.
func testAPIKeyProbeUnchanged(t *testing.T) {
	h := newAuthFront(t, []fake.RequestRule{
		authAckRule(),
		authRefreshRule(0, 0, false),
		authProbeCreateRule(),
		authProbeCloseRule(),
	}, configuredAuthKey())

	if _, err := authenticate(h, acpfront.AuthMethodZaiAPIKey); err != nil {
		t.Fatalf("authenticate api-key via the unchanged probe: %v", err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	createIdx := transcriptMethodIndexes(tr, zcode.MethodSessionCreate)
	setModelIdx := transcriptMethodIndexes(tr, zcode.MethodSessionSetModel)
	closeIdx := transcriptMethodIndexes(tr, zcode.MethodSessionClose)

	if len(createIdx) != 1 || len(closeIdx) != 1 {
		t.Fatalf("transcript records create×%d close×%d, want the bare probe pair", len(createIdx), len(closeIdx))
	}

	if len(setModelIdx) != 0 {
		t.Fatalf("transcript records setModel×%d on the api-key probe — the poke must stay coding-plan-only",
			len(setModelIdx))
	}
}
