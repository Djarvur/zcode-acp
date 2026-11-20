package acpfront_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/acp-go-sdk"

	"github.com/Djarvur/zcode-acp/internal/acpfront"
	"github.com/Djarvur/zcode-acp/internal/acpport"
	"github.com/Djarvur/zcode-acp/internal/hostsvc"
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
	return []string{"zai-api/GLM-5.3-Flash", "zai-api/GLM-5.3"}
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
		Name:          "auth-test",
		OnRequest:     rules,
		UnknownMethod: &fake.Reply{Error: &zcode.WireError{Code: zcode.CodeMethodNotFound, Message: "MethodNotFound"}},
		EOFDrainMs:    100,
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

// TestAuthenticate locks the AUTH-01 composition tier's five arms.
func TestAuthenticate(t *testing.T) {
	t.Run("happy chain is read-back proven", testAuthHappyChain)
	t.Run("no key answers the coded error naming ZCODE_ACP_API_KEY", testAuthNoKey)
	t.Run("unknown methodId answers coded invalid-params", testAuthUnknownMethod)
	t.Run("ack without registry confirmation errors, never {}", testAuthReadbackNegative)
	t.Run("interrupted authenticate is a clean retry", testAuthInterruptRetry)
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
