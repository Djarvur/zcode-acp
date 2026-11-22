package acpfront_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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

// Front.Authenticate contract locks (plan 05-07, AUTH-01 composition tier):
// the honest chain (record key → push snapshot → refresh registry →
// registry read-back → {}) proven on the FAKE TRANSCRIPT (PITFALL 3 — the
// ack is never the verdict, and the success arm asserts the SEQUENCE, not
// the return value alone), the OQ4 no-key arm naming ZCODE_ACP_API_KEY,
// the unknown-methodId invalid-params arm, the read-back-negative arm
// (ack OK but the registry never confirms → coded error, never {}), and
// the interruptibility probe (cancel between push and read-back → coded
// error, no success recorded, immediate re-authenticate completes cleanly).
//
// The harness composes the full loop (SDK client ⇄ acpport ⇄ Front ⇄
// zcode.Conn ⇄ fake.Serve) with IN-CODE auth scenarios (the push/refresh
// rules per arm); every key material is the synthetic fixture marker.

const (
	// syntheticAuthKey is the in-test fixture key (never a real credential;
	// the scan-secrets gate stays green).
	syntheticAuthKey = "zc-synthetic-test-key"

	// authFixKeyID is the fixture apiKeyId.
	authFixKeyID = "key_auth_fix_1"

	// authRefreshDelayMs is the interrupt arm's refresh-reply delay: the
	// cancel lands well inside it, after the push reply registered.
	authRefreshDelayMs = 800

	// authCancelAfter is when the interrupt arm fires its cancel (well
	// after the fast push leg, well inside the refresh delay).
	authCancelAfter = 200 * time.Millisecond

	// authReadbackWait bounds the negative arm's authenticate call (the
	// read-back window the ctx caps).
	authReadbackWait = 400 * time.Millisecond
)

// authRefreshStateUpdated is the registry patch the read-back surfaces
// carry (the corpus row's shape).
const authRefreshStateUpdated = `{"scope":"registry","reason":"registry_refresh","revision":2,` +
	`"patch":{"providers":{"zai-api":{"models":["GLM-5.3-Flash","GLM-5.3"]}}}}`

// authFixWantModels is the read-back want list (the corpus-locked zai-api
// pair — the fixtures' registry patch carries exactly these).
func authFixWantModels() []string {
	return []string{corpusModelFlashID, corpusModelPlainID}
}

// stubAuthKey is the in-test KeySource double (the hostsvc binding).
type stubAuthKey struct {
	apiKey string
	keyID  string
	ok     bool
}

// APIKey replays the canned material.
func (k *stubAuthKey) APIKey(_ context.Context) (apiKey, keyID string, ok bool) {
	return k.apiKey, k.keyID, k.ok
}

// configuredAuthKey is the arms' configured key source.
func configuredAuthKey() *stubAuthKey {
	return &stubAuthKey{apiKey: syntheticAuthKey, keyID: authFixKeyID, ok: true}
}

// authFixtureInstall writes a temp install whose zcode-builtin.json records
// revision 44 (the snapshot formula's probe-time read).
func authFixtureInstall(t *testing.T) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(`{"revision":44}`), 0o600); err != nil {
		t.Fatalf("write builtin fixture: %v", err)
	}

	return supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: builtin,
		Layout:      "explicit",
	}
}

// authAckRule is the canned provider/updateAccountConfig ack (receipt
// evidence only).
func authAckRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodProviderUpdateAccountConfig,
		Reply: &fake.Reply{Result: json.RawMessage(
			`{"receivedRevision":"account-bridge:auth-fix1","providerCount":0,"status":"received"}`)},
	}
}

// authRefreshRule builds one provider/refreshRegistry rule: times caps the
// match budget (0 = unlimited), delayMs holds the reply, withEmit arms the
// registry patch (the read-back surface).
func authRefreshRule(times, delayMs int, withEmit bool) fake.RequestRule {
	rule := fake.RequestRule{
		Method:  zcode.MethodProviderRefreshRegistry,
		Times:   times,
		DelayMs: delayMs,
		Reply:   &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)},
	}

	if withEmit {
		rule.SideEffects = []fake.SideEffect{{Emit: &fake.Emit{
			Method: zcode.MethodStateUpdated,
			Params: json.RawMessage(authRefreshStateUpdated),
		}}}
	}

	return rule
}

// authScenario wraps per-arm rules into the fail-loud scenario shape.
func authScenario(rules []fake.RequestRule) *fake.Scenario {
	return &fake.Scenario{
		Name:      "auth-test",
		OnRequest: rules,
		UnknownMethod: &fake.Reply{
			Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: fakeMethodNotFound},
		},
		EOFDrainMs: 100,
	}
}

// authReadbackDeadline is the arms' shortened read-back window: the
// negative arm's coded read-back error must reach the client WELL before
// the client's own bounded context dies (a client ctx expiry would surface
// the SDK's cancelled-request mapping instead of the server's verdict).
const authReadbackDeadline = 150 * time.Millisecond

// newAuthFront composes the full loop over per-arm rules with the auth
// machinery injected.
func newAuthFront(t *testing.T, rules []fake.RequestRule, keys hostsvc.KeySource) *FrontHarness {
	t.Helper()

	return newAuthFrontWith(t, rules, acpfront.AuthConfig{
		Keys:             keys,
		Install:          authFixtureInstall(t),
		WantModels:       authFixWantModels(),
		ReadbackDeadline: authReadbackDeadline,
	})
}

// newAuthFrontWith composes the loop over an explicit auth config.
func newAuthFrontWith(t *testing.T, rules []fake.RequestRule, cfg acpfront.AuthConfig) *FrontHarness {
	t.Helper()

	if cfg.Install.Root == "" {
		cfg.Install = authFixtureInstall(t)
	}

	if len(cfg.WantModels) == 0 {
		cfg.WantModels = authFixWantModels()
	}

	h := ComposeFront(t, authScenario(rules), func(conn *zcode.Conn) acp.Agent {
		front := acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithAuth(cfg),
		)

		return acpport.NewAgent(front)
	})

	return h
}

// authenticate runs one client.authenticate{methodId} on the harness with a
// fresh bounded context.
func authenticate(h *FrontHarness, methodID string) (acp.AuthenticateResponse, error) {
	ctx, cancel := h.ExchangeCtx()
	defer cancel()

	return h.client.Authenticate(ctx, acp.AuthenticateRequest{MethodId: acp.AuthMethodId(methodID)})
}

// transcriptMethodIndexes returns the transcript indexes of every host→fake
// frame naming the method, in wire order.
func transcriptMethodIndexes(tr fake.Transcript, method string) []int {
	indexes := []int{}
	for i, line := range tr.Lines {
		if line.Dir == "in" && strings.Contains(line.JSON, `"method":"`+method+`"`) {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

// wireErrOf extracts the SDK RequestError from an authenticate failure
// (failing the test when the error is something else — the adapter must
// always answer coded request errors).
func wireErrOf(t *testing.T, err error) *acp.RequestError {
	t.Helper()

	if err == nil {
		t.Fatal("expected an authenticate error, got success")
	}

	var rerr *acp.RequestError
	if !errors.As(err, &rerr) {
		t.Fatalf("authenticate error %v (%T) is not a coded *acp.RequestError — "+
			"every arm answers a coded error envelope", err, err)
	}

	return rerr
}

// TestAuthenticate locks the AUTH-01 composition tier's arms.
func TestAuthenticate(t *testing.T) {
	t.Run("happy chain is read-back proven", testAuthHappyChain)
	t.Run("no key answers the coded error naming ZCODE_ACP_API_KEY", testAuthNoKey)
	t.Run("unknown methodId answers coded invalid-params", testAuthUnknownMethod)
	t.Run("ack without registry confirmation errors, never {}", testAuthReadbackNegative)
	t.Run("interrupted authenticate is a clean retry", testAuthInterruptRetry)
	t.Run("probe create surface latches the field-verified verdict", testAuthProbeCreateSurface)
}

// authProbeCreateRule scripts the probe session/create reply the branch-(c)
// lock arms against: the create reply carries settings.model.available with
// BOTH want models — the field-verified read-back surface (plan 05-11,
// EXP-READBACK-DIAG-01: the real desktop 3.18.2 emits NO state.updated
// patch for push+refresh; the create reply IS the registry read-back).
func authProbeCreateRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionCreate,
		Reply: &fake.Reply{Result: json.RawMessage(
			`{"session":{"sessionId":"sess_auth_probe"},` +
				`"settings":{"model":{"available":[` +
				`{"ref":{"providerId":"zai-api","modelId":"GLM-5.3-Flash"}},` +
				`{"ref":{"providerId":"zai-api","modelId":"GLM-5.3"}}]}}}`)},
	}
}

// authProbeCloseRule scripts the probe session/close reply (the
// field-verified {closed:true} shape the diagnostic recorded).
func authProbeCloseRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodSessionClose,
		Reply:  &fake.Reply{Result: json.RawMessage(`{"closed":true}`)},
	}
}

// testAuthProbeCreateSurface is the 05-11 branch-(c) corrected-surface lock:
// the fake acks but emits NO state.updated (the field-verified real-desktop
// shape) — authenticate must succeed ONLY via the probe create's
// settings.model.available surface, with the probe pair (create → close)
// on the transcript between the refresh and the chain's return. The ack and
// the probe's transport success are never the verdict; the watcher's latch
// is.
func testAuthProbeCreateSurface(t *testing.T) {
	h := newAuthFront(t, []fake.RequestRule{
		authAckRule(),
		authRefreshRule(0, 0, false),
		authProbeCreateRule(),
		authProbeCloseRule(),
	}, configuredAuthKey())

	if _, err := authenticate(h, acpfront.AuthMethodZaiAPIKey); err != nil {
		t.Fatalf("authenticate via the probe-create read-back surface: %v", err)
	}

	// PITFALL 3 discipline: the probe ran on the wire — exactly one create
	// and one close, ordered refresh → create → close (the probe never
	// precedes its traffic and the close never precedes its create).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	refreshIdx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry)
	createIdx := transcriptMethodIndexes(tr, zcode.MethodSessionCreate)
	closeIdx := transcriptMethodIndexes(tr, zcode.MethodSessionClose)

	if len(createIdx) != 1 || len(closeIdx) != 1 {
		t.Fatalf("transcript records probe create×%d close×%d, want exactly one probe pair",
			len(createIdx), len(closeIdx))
	}

	if len(refreshIdx) != 1 || refreshIdx[0] >= createIdx[0] || createIdx[0] >= closeIdx[0] {
		t.Fatalf("probe order violated: refresh(%v) create(%v) close(%v) — "+
			"the probe pair must run between the refresh and the chain's return",
			refreshIdx, createIdx, closeIdx)
	}
}

// testAuthHappyChain is arm (a): the configured-key chain answers the empty
// {} AND the transcript proves the full sequence ran (push → refresh —
// PITFALL 3: never the return value alone).
func testAuthHappyChain(t *testing.T) {
	h := newAuthFront(t, []fake.RequestRule{authAckRule(), authRefreshRule(0, 0, true)}, configuredAuthKey())

	if _, err := authenticate(h, acpfront.AuthMethodZaiAPIKey); err != nil {
		t.Fatalf("authenticate with a configured key: %v", err)
	}

	// PITFALL 3 discipline: the success arm asserts the TRANSCRIPT
	// SEQUENCE — push then refresh actually ran on the back leg — never
	// the return value alone.
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	pushIdx := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig)
	refreshIdx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry)

	if len(pushIdx) != 1 || len(refreshIdx) != 1 {
		t.Fatalf("transcript records push×%d refresh×%d, want exactly one honest chain run",
			len(pushIdx), len(refreshIdx))
	}

	if pushIdx[0] >= refreshIdx[0] {
		t.Fatalf("chain order violated: push(%d) must precede refresh(%d)", pushIdx[0], refreshIdx[0])
	}

	// The ACP tee answers a RESULT frame (the empty {}), never an error.
	lines := h.ACPLines()
	reqIdx := teeLineIndex(t, lines, "out", `"authenticate"`, "authenticate request")
	respLine := authTeeResponse(t, lines, reqIdx)

	if strings.Contains(respLine.JSON, `"error"`) {
		t.Fatalf("authenticate response carries an error frame: %s", respLine.JSON)
	}

	if !strings.Contains(respLine.JSON, `"result":{}`) {
		t.Fatalf("authenticate response = %s, want the empty {} result", respLine.JSON)
	}
}

// testAuthNoKey is arm (b): the OQ4 disposition — the honest coded error
// names ZCODE_ACP_API_KEY, and the rejection precedes every backend call
// (never a fallback read of the desktop's config store).
func testAuthNoKey(t *testing.T) {
	h := newAuthFront(t, []fake.RequestRule{authAckRule(), authRefreshRule(0, 0, true)},
		&stubAuthKey{ok: false})

	_, err := authenticate(h, acpfront.AuthMethodZaiAPIKey)

	rerr := wireErrOf(t, err)
	if !strings.Contains(rerr.Message, "ZCODE_ACP_API_KEY") {
		t.Fatalf("no-key error message %q must name ZCODE_ACP_API_KEY exactly (OQ4)", rerr.Message)
	}

	// The honest rejection happens BEFORE any backend call — never a
	// push without a key, never a fallback read of the desktop's config
	// store (OQ4).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if idx := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig); len(idx) != 0 {
		t.Fatalf("no-key authenticate touched the back leg (push at %v) — "+
			"the honest rejection must precede every backend call", idx)
	}
}

// testAuthUnknownMethod is arm (c): an id outside the advertised set
// answers coded invalid-params naming the offending id and the set.
func testAuthUnknownMethod(t *testing.T) {
	h := newAuthFront(t, []fake.RequestRule{authAckRule(), authRefreshRule(0, 0, true)}, configuredAuthKey())

	_, err := authenticate(h, "zai-oauth")

	rerr := wireErrOf(t, err)
	if rerr.Code != acpport.CodeInvalidParams {
		t.Fatalf("unknown methodId code = %d, want %d (-32602)", rerr.Code, acpport.CodeInvalidParams)
	}

	if !strings.Contains(rerr.Message, "zai-oauth") ||
		!strings.Contains(rerr.Message, acpfront.AuthMethodZaiAPIKey) {
		t.Fatalf("unknown-methodId message %q must name the offending id and the advertised set",
			rerr.Message)
	}
}

// testAuthReadbackNegative is arm (d): the ack says received but the
// registry never confirms — the coded read-back error answers, never {}
// (PITFALL 1: the verdict is the registry's).
func testAuthReadbackNegative(t *testing.T) {
	h := newAuthFront(t, []fake.RequestRule{authAckRule(), authRefreshRule(0, 0, false)}, configuredAuthKey())

	ctx, cancel := context.WithTimeout(context.Background(), authReadbackWait)
	defer cancel()

	_, err := h.client.Authenticate(ctx, acp.AuthenticateRequest{
		MethodId: acp.AuthMethodId(acpfront.AuthMethodZaiAPIKey),
	})

	rerr := wireErrOf(t, err)
	if !strings.Contains(rerr.Message, "read-back") {
		t.Fatalf("read-back-negative message %q must name the read-back step", rerr.Message)
	}

	// The negative's point: the chain RAN (push and refresh are on the
	// transcript — the ack said received) and the verdict was still the
	// registry's, never the ack's (PITFALL 1).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if idx := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig); len(idx) != 1 {
		t.Fatalf("transcript records push×%d, want the chain's honest run", len(idx))
	}

	if idx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry); len(idx) != 1 {
		t.Fatalf("transcript records refresh×%d, want the chain's honest run", len(idx))
	}
}

// testAuthInterruptRetry is arm (e) — the AUTH-01 concurrency probe: a
// cancelled authenticate surfaces a coded error with no success recorded,
// and the immediate re-authenticate completes cleanly (no
// half-authenticated state observable).
func testAuthInterruptRetry(t *testing.T) {
	// Run 1: the refresh reply holds (authRefreshDelayMs); the cancel
	// fires inside the hold — after the push leg, before any read-back.
	h := newAuthFront(t, []fake.RequestRule{
		authAckRule(),
		authRefreshRule(1, authRefreshDelayMs, false), // spent by run 1
		authRefreshRule(0, 0, true),                   // run 2's registry surface
	}, configuredAuthKey())

	ctx1, cancel1 := context.WithCancel(context.Background())

	type authResult struct {
		resp acp.AuthenticateResponse
		err  error
	}

	done1 := make(chan authResult, 1)

	go func() {
		resp, err := h.client.Authenticate(ctx1, acp.AuthenticateRequest{
			MethodId: acp.AuthMethodId(acpfront.AuthMethodZaiAPIKey),
		})
		done1 <- authResult{resp: resp, err: err}
	}()

	time.Sleep(authCancelAfter)
	cancel1()

	res1 := <-done1
	if res1.err == nil {
		t.Fatal("interrupted authenticate answered success — a cancelled chain must surface a coded error")
	}

	var rerr1 *acp.RequestError
	if !errors.As(res1.err, &rerr1) {
		t.Fatalf("interrupted authenticate error %v is not a coded request error", res1.err)
	}

	// Run 2: the immediate re-authenticate completes cleanly — no
	// half-authenticated state is observable (the AUTH-01 concurrency
	// probe). The empty-{} wire shape is the happy arm's lock; here the
	// clean completion is the point.
	if _, err2 := authenticate(h, acpfront.AuthMethodZaiAPIKey); err2 != nil {
		t.Fatalf("re-authenticate after interruption: %v (the retry must be clean)", err2)
	}

	// The retry ran its OWN full honest chain (a second push+refresh on
	// the transcript — the retry is not a replay of cached success).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if idx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry); len(idx) != 2 {
		t.Fatalf("transcript records refresh×%d, want both runs' honest chains", len(idx))
	}
}

// authTeeResponse finds the agent→client response frame matching the
// request at reqIdx (by JSON-RPC id) — the authenticate reply assertion.
func authTeeResponse(t *testing.T, lines []TeeLine, reqIdx int) TeeLine {
	t.Helper()

	var req struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal([]byte(lines[reqIdx].JSON), &req); err != nil {
		t.Fatalf("decode authenticate request frame: %v (%s)", err, lines[reqIdx].JSON)
	}

	idKey := `"id":` + string(req.ID)

	for _, line := range lines {
		if line.Dir == "in" && !strings.Contains(line.JSON, `"method"`) && strings.Contains(line.JSON, idKey) {
			return line
		}
	}

	t.Fatalf("no response frame for the authenticate request (%s on the tee)", idKey)

	return TeeLine{}
}

// TestCodingPlanAuth locks the composed coding-plan authenticate tier
// (AUTH-02/03/04, plan 06-05) — the official chain's arms over the fake
// transcript:
//
//	(a) the happy arm: probe ran, push carries the ACCOUNT overlay, refresh
//	    precedes the verdict, the registry patch carries exactly the
//	    snapshot-derived models (the D-08 verdict-side mirror), the PAT
//	    mint ran exactly once after the verdict, the auth mode recorded;
//	(b) the probe-fail arm: a canned probe error answers the coded probe
//	    error and NOTHING touches the wire (D-02 ordering, composed-tier
//	    proof — zero push, zero refresh, zero mint);
//	(c) the no-login arm: an empty store answers the coded desktop-first
//	    error before any traffic, no mint;
//	(d) the read-back negative: the ack says received but the registry
//	    never confirms — the coded read-back error and NO mint (the mint
//	    happens only after the verdict — the PITFALL-1 discipline's mint
//	    half);
//	(e) the cancel-interrupt arm: cancel between push and verdict → coded
//	    error, no success recorded, an immediate retry completes cleanly;
//	(f) the api-key regression: authenticate{zai-api-key} still passes with
//	    the coding-plan seams present (AUTH-03 both-methods).
//
// All fixture material is synthetic (the scan-secrets gate stays green).
func TestCodingPlanAuth(t *testing.T) {
	t.Run("happy chain is probe-proven, overlay-pushed, mint-once", testCodingPlanHappy)
	t.Run("probe failure answers coded before any wire traffic", testCodingPlanProbeFail)
	t.Run("no desktop login answers the coded desktop-first error", testCodingPlanNoLogin)
	t.Run("read-back negative mints nothing", testCodingPlanReadbackNegative)
	t.Run("interrupted coding-plan authenticate is a clean retry", testCodingPlanCancelRetry)
	t.Run("api-key arm stays green with the coding-plan seams present", testCodingPlanAPIKeyRegression)
}

// The arms' static fixture errors (err113: static errors, never dynamic).
var (
	// errCannedProbeFailure is the probe-fail arm's injected verdict.
	errCannedProbeFailure = errors.New("canned probe failure")
)

// Coding-plan fixture material — synthetic markers only.
const (
	cpFixLoginToken = "fixture-synthetic-not-a-real-login-token"

	cpFixPatToken = "fixture-synthetic-not-a-real-pat"

	cpFixPatKeyID = "key_cp_fix_1"

	// cpFixAccountProvider is the fixture builtin table's zhipu-account rule.
	cpFixAccountProvider = "account:zai-individual-coding-plan"

	// cpFixAccountFamily is the rule's entitlement family.
	cpFixAccountFamily = "zai"

	// cpFixAccountModels are the rule's builtinModelIds (the live-probed
	// revision-44 pair, synthetic in the fixture).
	cpFixAccountModelA = "GLM-5.3"

	cpFixAccountModelB = "GLM-5.3-Flash"
)

// cpRefreshStateUpdated is the registry patch the happy arm's read-back
// surfaces carry (the account models — exactly the snapshot-derived want).
const cpRefreshStateUpdated = `{"scope":"registry","reason":"registry_refresh","revision":3,` +
	`"patch":{"providers":{` + `"` + cpFixAccountProvider + `":{"models":["` +
	cpFixAccountModelA + `","` + cpFixAccountModelB + `"]}}}}`

// cpAckRule is the canned provider/updateAccountConfig ack for the account
// overlay: providerCount 1 — the overlay's provider is visible in the
// receipt (receipt evidence only; the verdict stays the registry's).
func cpAckRule() fake.RequestRule {
	return fake.RequestRule{
		Method: zcode.MethodProviderUpdateAccountConfig,
		Reply: &fake.Reply{Result: json.RawMessage(
			`{"receivedRevision":"account-bridge:cp-fix1","providerCount":1,"status":"received"}`)},
	}
}

// cpRefreshRule builds one provider/refreshRegistry rule for the
// coding-plan arms (withEmit arms the ACCOUNT-models registry patch).
func cpRefreshRule(times, delayMs int, withEmit bool) fake.RequestRule {
	rule := fake.RequestRule{
		Method:  zcode.MethodProviderRefreshRegistry,
		Times:   times,
		DelayMs: delayMs,
		Reply:   &fake.Reply{Result: json.RawMessage(`{"refreshed":true}`)},
	}

	if withEmit {
		rule.SideEffects = []fake.SideEffect{{Emit: &fake.Emit{
			Method: zcode.MethodStateUpdated,
			Params: json.RawMessage(cpRefreshStateUpdated),
		}}}
	}

	return rule
}

// codingPlanFixtureInstall writes a temp install whose zcode-builtin.json
// carries revision 44 AND one zhipu-account provider rule (the account
// overlay's source table — BuildAccount reads the rules verbatim).
func codingPlanFixtureInstall(t *testing.T) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	doc := `{"revision":44,"config":{"providerConfigRules":{"providerRules":[` +
		`{"providerId":"` + cpFixAccountProvider + `","config":{` +
		`"builtinModelIds":["` + cpFixAccountModelA + `","` + cpFixAccountModelB + `"],` +
		`"access":{"type":"zhipu-account","accountType":"` + cpFixAccountFamily + `"}}}]}}}`

	if err := os.WriteFile(builtin, []byte(doc), 0o600); err != nil {
		t.Fatalf("write builtin fixture: %v", err)
	}

	return supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: builtin,
		Layout:      "explicit",
	}
}

// cpFixtureCredentials writes the fixture credentials.json and returns the
// 06-01 store over it. entries == nil writes a fresh empty store (the
// no-login arm).
func cpFixtureCredentials(t *testing.T, entries map[string]string) *interact.CredentialStore {
	t.Helper()

	if entries == nil {
		entries = map[string]string{}
	}

	doc, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal credentials fixture: %v", err)
	}

	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, doc, 0o600); err != nil {
		t.Fatalf("write credentials fixture: %v", err)
	}

	// v2 loader nil: the fixture entries are plaintext (the store's default
	// dispatch arm — the reader never touches a keyring for them).
	return interact.NewCredentialStore(path, nil)
}

// recordingProbe is the injected probe seam double: it records its
// invocations and replays a canned verdict (the KeyLoader injection
// discipline).
type recordingProbe struct {
	mu    sync.Mutex
	calls int
	err   error
}

// probeKeyring adapts the double to the interact.KeyringProber func seam.
func (r *recordingProbe) probeKeyring(_ context.Context) error {
	r.mu.Lock()
	r.calls++
	fn := r.err
	r.mu.Unlock()

	return fn
}

// count reports the recorded invocation count.
func (r *recordingProbe) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.calls
}

// scriptedPat is the PatEnsurer double with the production cache semantics
// (the minter owns caching + single-flight): the FIRST Ensure mints and warms
// the cache; any later Ensure dedupes against the warm cache (the eager-mint
// + ask-side single-flight shape); a mint while warm would be a contract
// violation — the double serves cached instead and the mint counter stays,
// so the mint-once locks read exactly one mint per authenticate lifetime.
type scriptedPat struct {
	mu    sync.Mutex
	mints int
	warm  bool
}

// Ensure serves cache-first: one mint, later calls dedupe.
func (s *scriptedPat) Ensure(_ context.Context) (hostsvc.Pat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.warm {
		return s.fixturePat(), nil
	}

	s.mints++
	s.warm = true

	return s.fixturePat(), nil
}

// Cached reports the cache without minting (the ask-side read).
func (s *scriptedPat) Cached(_ context.Context) (hostsvc.Pat, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.fixturePat(), s.warm
}

// Invalidate drops the cached token (unused on these arms' scripts — an
// Invalidate here would mean a fingerprint rejection the scenarios never
// script).
func (s *scriptedPat) Invalidate(string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.warm = false
}

// fixturePat renders the invariant fixture PAT (every serve is the same
// cached value — the cache identity the asks compare against).
func (s *scriptedPat) fixturePat() hostsvc.Pat {
	return hostsvc.Pat{
		Token:        cpFixPatToken,
		APIKeyID:     cpFixPatKeyID,
		Fingerprint:  strings.Repeat("0", 64),
		AccountScope: strings.Repeat("f", 64),
	}
}

// count reports the recorded mint count.
func (s *scriptedPat) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.mints
}

// codingPlanConfig composes the arms' default coding-plan seams: the
// canned-pass recording probe, the fixture store (login present), the
// scripted mint double, and the canned-true entitlement gate.
func codingPlanConfig(t *testing.T) (acpfront.AuthConfig, *recordingProbe, *scriptedPat) {
	t.Helper()

	probe := &recordingProbe{}
	pat := &scriptedPat{}

	cfg := acpfront.AuthConfig{
		Creds: cpFixtureCredentials(t, map[string]string{
			"oauth:" + cpFixAccountFamily + ":access_token": cpFixLoginToken,
		}),
		Pat:   pat,
		Probe: probe.probeKeyring,
		Entitle: func(_ context.Context, family string) bool {
			return family == cpFixAccountFamily
		},
	}

	return cfg, probe, pat
}

// newCodingPlanFront composes the loop over coding-plan rules and returns
// the harness WITH the front (the arms assert the recorded auth mode on it).
func newCodingPlanFront(
	t *testing.T, rules []fake.RequestRule, cfg acpfront.AuthConfig,
) (*FrontHarness, *acpfront.Front) {
	t.Helper()

	if cfg.Install.Root == "" {
		cfg.Install = codingPlanFixtureInstall(t)
	}

	if cfg.ReadbackDeadline == 0 {
		cfg.ReadbackDeadline = authReadbackDeadline
	}

	var front *acpfront.Front

	h := ComposeFront(t, authScenario(rules), func(conn *zcode.Conn) acp.Agent {
		front = acpfront.New(conn,
			acpfront.WithVersion("0.0.0-dev"),
			acpfront.WithAuth(cfg),
		)

		return acpport.NewAgent(front)
	})

	return h, front
}

// assertNoCodingPlanTraffic fails the test when any push or refresh frame
// touched the transcript (the D-02 ordering lock's shared assertion).
func assertNoCodingPlanTraffic(t *testing.T, tr fake.Transcript) {
	t.Helper()

	if idx := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig); len(idx) != 0 {
		t.Fatalf("transcript records push at %v — the probe-gated chain must not touch the wire", idx)
	}

	if idx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry); len(idx) != 0 {
		t.Fatalf("transcript records refresh at %v — the probe-gated chain must not touch the wire", idx)
	}
}

// testCodingPlanHappy is arm (a): the coding-plan chain answers {} with the
// FULL sequence on the transcript — push (the account overlay visible in
// the frame), refresh, the registry patch carrying exactly the
// snapshot-derived models (the D-08 verdict-side mirror asserted against
// the PUSHED builtinModelIds, never a literal catalog), exactly one mint,
// and the auth mode recorded.
func testCodingPlanHappy(t *testing.T) {
	cfg, probe, pat := codingPlanConfig(t)
	h, front := newCodingPlanFront(t, []fake.RequestRule{cpAckRule(), cpRefreshRule(0, 0, true)}, cfg)

	if _, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan); err != nil {
		t.Fatalf("authenticate coding-plan: %v", err)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertCodingPlanHappySequence(t, tr)

	// The probe RAN (one proactive call — the in-process step the
	// transcript cannot see; arm (b) proves its ordering).
	if calls := probe.count(); calls != 1 {
		t.Fatalf("probe ran %d time(s), want exactly one", calls)
	}

	// D-04: exactly one mint.
	if calls := pat.count(); calls != 1 {
		t.Fatalf("PAT Ensure ran %d time(s), want exactly one (mint-once)", calls)
	}

	// D-11: the success recorded the coding-plan mode.
	if mode := front.AuthMode(); mode != acpfront.AuthModeCodingPlan {
		t.Fatalf("auth mode = %q, want %q after a coding-plan success", mode, acpfront.AuthModeCodingPlan)
	}
}

// assertCodingPlanHappySequence is the PITFALL-3 transcript lock: exactly
// one push then one refresh, the PUSH FRAME carrying the account overlay
// (Pattern 2's strict shape, mirrored verbatim from the fixture table), and
// the registry patch (the state.updated emit) carrying exactly the
// providerId/models pairs derived from the PUSHED builtinModelIds — the
// D-08 verdict-side mirror. The happy arm's success already proves the
// watcher latched the derived want; this pins the wire surfaces themselves
// (never a literal catalog).
func assertCodingPlanHappySequence(t *testing.T, tr fake.Transcript) {
	t.Helper()

	pushIdx := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig)
	refreshIdx := transcriptMethodIndexes(tr, zcode.MethodProviderRefreshRegistry)

	if len(pushIdx) != 1 || len(refreshIdx) != 1 {
		t.Fatalf("transcript records push×%d refresh×%d, want exactly one honest chain run",
			len(pushIdx), len(refreshIdx))
	}

	if pushIdx[0] >= refreshIdx[0] {
		t.Fatalf("chain order violated: push(%d) must precede refresh(%d)", pushIdx[0], refreshIdx[0])
	}

	pushJSON := tr.Lines[pushIdx[0]].JSON

	for _, marker := range []string{
		cpFixAccountProvider, `"builtinModelIds"`, `"zhipu-account"`, `"entitled":true`, `"states"`,
	} {
		if !strings.Contains(pushJSON, marker) {
			t.Fatalf("push frame %s lacks the overlay marker %s", pushJSON, marker)
		}
	}

	patchFound := false

	for _, line := range tr.Lines {
		if strings.Contains(line.JSON, `"method":"`+zcode.MethodStateUpdated+`"`) &&
			strings.Contains(line.JSON, `"`+cpFixAccountProvider+`"`) &&
			strings.Contains(line.JSON, `"`+cpFixAccountModelA+`"`) &&
			strings.Contains(line.JSON, `"`+cpFixAccountModelB+`"`) {
			patchFound = true

			break
		}
	}

	if !patchFound {
		t.Fatal("no account-models registry patch on the transcript — the verdict would wait forever")
	}
}

// testCodingPlanProbeFail is arm (b) — the D-02 ordering lock at the
// composed tier: a canned probe error answers the coded probe error and
// NOTHING touched the wire (zero push, zero refresh), nothing minted, and
// no success was recorded.
func testCodingPlanProbeFail(t *testing.T) {
	cfg, probe, pat := codingPlanConfig(t)
	probe.err = errCannedProbeFailure

	h, front := newCodingPlanFront(t, []fake.RequestRule{cpAckRule(), cpRefreshRule(0, 0, true)}, cfg)

	_, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan)

	rerr := wireErrOf(t, err)
	if !strings.Contains(rerr.Message, "keyring") {
		t.Fatalf("probe-fail message %q must name the keyring step", rerr.Message)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertNoCodingPlanTraffic(t, tr)

	if calls := probe.count(); calls != 1 {
		t.Fatalf("probe ran %d time(s), want exactly one", calls)
	}

	if calls := pat.count(); calls != 0 {
		t.Fatalf("PAT Ensure ran %d time(s) on a probe failure — the mint must never run", calls)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeNone {
		t.Fatalf("auth mode = %q after a probe failure, want none recorded", mode)
	}
}

// testCodingPlanNoLogin is arm (c): an empty store answers the coded
// desktop-first error BEFORE any traffic (T-06-16 — a missing desktop login
// never fabricates), no mint.
func testCodingPlanNoLogin(t *testing.T) {
	cfg, probe, pat := codingPlanConfig(t)
	cfg.Creds = cpFixtureCredentials(t, nil) // fresh machine: no oauth entry

	h, front := newCodingPlanFront(t, []fake.RequestRule{cpAckRule(), cpRefreshRule(0, 0, true)}, cfg)

	_, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan)

	rerr := wireErrOf(t, err)
	if !strings.Contains(rerr.Message, "desktop") {
		t.Fatalf("no-login message %q must carry the desktop-first hint", rerr.Message)
	}

	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	assertNoCodingPlanTraffic(t, tr)

	if calls := probe.count(); calls != 1 {
		t.Fatalf("probe ran %d time(s), want exactly one (the probe precedes the login read)", calls)
	}

	if calls := pat.count(); calls != 0 {
		t.Fatalf("PAT Ensure ran %d time(s) on a no-login failure — the mint must never run", calls)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeNone {
		t.Fatalf("auth mode = %q after a no-login failure, want none recorded", mode)
	}
}

// testCodingPlanReadbackNegative is arm (d): the ack says received but the
// registry never confirms within the shortened deadline — the coded
// read-back error answers, and the eager mint ALREADY RAN (D-04 round-3:
// the mint arms the cache at authenticate, before the probe — the verdict
// gates the SUCCESS RECORD, never the mint; the failure path restores the
// prior mode so nothing new is recorded).
func testCodingPlanReadbackNegative(t *testing.T) {
	cfg, _, pat := codingPlanConfig(t)
	h, front := newCodingPlanFront(t, []fake.RequestRule{cpAckRule(), cpRefreshRule(0, 0, false)}, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), authReadbackWait)
	defer cancel()

	_, err := h.client.Authenticate(ctx, acp.AuthenticateRequest{
		MethodId: acp.AuthMethodId(acpfront.AuthMethodZaiCodingPlan),
	})

	rerr := wireErrOf(t, err)
	if !strings.Contains(rerr.Message, "read-back") {
		t.Fatalf("read-back-negative message %q must name the read-back step", rerr.Message)
	}

	// The chain RAN (push and refresh on the transcript — the ack said
	// received) and the verdict was still the registry's (PITFALL 1).
	tr, trErr := h.Transcript(t)
	if trErr != nil {
		t.Fatalf("fake transcript: %v", trErr)
	}

	if idx := transcriptMethodIndexes(tr, zcode.MethodProviderUpdateAccountConfig); len(idx) != 1 {
		t.Fatalf("transcript records push×%d, want the chain's honest run", len(idx))
	}

	if calls := pat.count(); calls != 1 {
		t.Fatalf("PAT minted %d time(s), want exactly one — the eager mint "+
			"arms the cache before the probe (D-04 round-3) even when the verdict fails", calls)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeNone {
		t.Fatalf("auth mode = %q after a read-back failure, want none recorded", mode)
	}
}

// testCodingPlanCancelRetry is arm (e): cancel between push and verdict →
// coded error, no success recorded, the eager mint already armed the cache;
// the immediate re-authenticate completes cleanly (the
// interrupted-authenticate concurrency disposition, re-asserted for the new
// chain).
func testCodingPlanCancelRetry(t *testing.T) {
	cfg, probe, pat := codingPlanConfig(t)

	h, front := newCodingPlanFront(t, []fake.RequestRule{
		cpAckRule(),
		cpRefreshRule(1, authRefreshDelayMs, false), // spent by run 1
		cpRefreshRule(0, 0, true),                   // run 2's registry surface
	}, cfg)

	cancelMidChainRun(t, h)

	if calls := pat.count(); calls != 1 {
		t.Fatalf("PAT minted %d time(s) on the cancelled run, want exactly one — "+
			"the eager mint arms the cache before the traffic (D-04 round-3); "+
			"a cancelled run records no success, never a second mint", calls)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeNone {
		t.Fatalf("auth mode = %q after the cancelled run, want none recorded", mode)
	}

	// Run 2: the immediate re-authenticate completes cleanly.
	if _, err := authenticate(h, acpfront.AuthMethodZaiCodingPlan); err != nil {
		t.Fatalf("re-authenticate after interruption: %v (the retry must be clean)", err)
	}

	if calls := pat.count(); calls != 1 {
		t.Fatalf("PAT minted %d time(s) total, want still one — the retry's Ensure "+
			"dedupes against the cache the eager mint armed (the minter owns caching)", calls)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeCodingPlan {
		t.Fatalf("auth mode = %q after the clean retry, want %q", mode, acpfront.AuthModeCodingPlan)
	}

	if calls := probe.count(); calls != 2 {
		t.Fatalf("probe ran %d time(s), want one per authenticate attempt", calls)
	}
}

// cancelMidChainRun drives one coding-plan authenticate on its own context
// and cancels it mid-chain (inside the held refresh reply — after the push
// leg, before any verdict): a coded request error must surface, never a
// fabricated success.
func cancelMidChainRun(t *testing.T, h *FrontHarness) {
	t.Helper()

	ctx1, cancel1 := context.WithCancel(context.Background())

	type authResult struct {
		resp acp.AuthenticateResponse
		err  error
	}

	done1 := make(chan authResult, 1)

	go func() {
		resp, err := h.client.Authenticate(ctx1, acp.AuthenticateRequest{
			MethodId: acp.AuthMethodId(acpfront.AuthMethodZaiCodingPlan),
		})
		done1 <- authResult{resp: resp, err: err}
	}()

	time.Sleep(authCancelAfter)
	cancel1()

	res1 := <-done1
	if res1.err == nil {
		t.Fatal("interrupted coding-plan authenticate answered success — " +
			"a cancelled chain must surface a coded error")
	}

	var rerr1 *acp.RequestError
	if !errors.As(res1.err, &rerr1) {
		t.Fatalf("interrupted authenticate error %v is not a coded request error", res1.err)
	}
}

// testCodingPlanAPIKeyRegression is arm (f): with the coding-plan seams
// present, authenticate{zai-api-key} still passes its honest chain and
// records the api-key mode (AUTH-03 both-methods).
func testCodingPlanAPIKeyRegression(t *testing.T) {
	cfg, _, pat := codingPlanConfig(t)
	cfg.Keys = configuredAuthKey()
	cfg.Install = authFixtureInstall(t) // the api-key fixture table (no provider rules needed)
	cfg.WantModels = authFixWantModels()

	h, front := newCodingPlanFront(t, []fake.RequestRule{authAckRule(), authRefreshRule(0, 0, true)}, cfg)

	if _, err := authenticate(h, acpfront.AuthMethodZaiAPIKey); err != nil {
		t.Fatalf("authenticate api-key with the coding-plan seams present: %v", err)
	}

	if mode := front.AuthMode(); mode != acpfront.AuthModeAPIKey {
		t.Fatalf("auth mode = %q, want %q after an api-key success", mode, acpfront.AuthModeAPIKey)
	}

	if calls := pat.count(); calls != 0 {
		t.Fatalf("PAT Ensure ran %d time(s) on the api-key arm — the mint is coding-plan-only", calls)
	}
}
