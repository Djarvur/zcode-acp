package hostsvc_test

// The mode-aware PAT runtime-headers source locks (plan 06-07 Task 2,
// AUTH-02/D-05 — RESEARCH Pattern 4): every arm asserts on served answers
// and minter call counts (wire artifacts, never internals). The exactly-one
// re-mint semantics compose 06-02's single-flight Ensure; the race-heavy
// arms run under -race with concurrent asks.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Synthetic fixture material (plan 06-07; fixture data only, never a real
// credential — T-5-15; the fingerprints/scopes are 64-lowercase-hex shape
// constants the ask schema pins).
const (
	patFixtureTokenA = "zc-synthetic-pat-a-fixture"
	patFixtureTokenB = "zc-synthetic-pat-b-fixture"
	patFixtureKeyIDA = "key_pat_a"
	patFixtureKeyIDB = "key_pat_b"
	patFixtureFPA    = "1111111111111111111111111111111111111111111111111111111111111111" // sha256-shape of PAT-A
	patFixtureFPB    = "2222222222222222222222222222222222222222222222222222222222222222" // sha256-shape of PAT-B
	patFixtureScopeA = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	patFixtureScopeB = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	patFixtureBadHex = "not-hex-at-all"
	patFixtureModeID = "key_static_9"
	patFixtureModeTk = "zc-synthetic-static-fixture-key"
)

// errFixtureEnsureFailure is the scripted Ensure failure arm (err113).
var errFixtureEnsureFailure = errors.New("fixture ensure failure")

// scriptMinter is the patMintSource double: a warm cache field the source's
// Cached arm reads, a sequential Ensure script (a default outcome once the
// script drains), and invalidation counters — the exactly-once bookkeeping
// the arms assert on. Ensure is single-flight like the production PatMinter
// (WR-03): concurrent callers join one scripted mint, so a burst increments
// the mint counter exactly once.
type scriptMinter struct {
	mu           sync.Mutex
	cache        hostsvc.Pat
	hasCache     bool
	ensureScript []scriptedEnsure
	defaultHappy hostsvc.Pat
	ensures      int
	invalidates  int
	lastReason   string

	// Single-flight mint state mirroring PatMinter's leader/joiner shape:
	// minting != nil while one scripted mint is in flight; joiners wait on
	// it and read the published lastPat/lastErr outcome.
	minting chan struct{}
	mintID  uint64
	doneID  uint64
	lastPat *hostsvc.Pat
	lastErr error

	// ensureHold, when non-nil, parks the LEADER inside Ensure until the
	// channel closes — the deterministic-window hook the concurrent arm
	// uses to pin every rider into the join path. ensureStarts counts
	// cumulative Ensure entries (leader and joiners) — the arm's sync
	// point.
	ensureHold   chan struct{}
	ensureStarts int
}

// scriptedEnsure is one queued Ensure outcome: a PAT, or an error.
type scriptedEnsure struct {
	pat hostsvc.Pat
	err error
}

// Cached reports the scripted cache state (no network, no mint).
func (m *scriptMinter) Cached(_ context.Context) (hostsvc.Pat, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.cache, m.hasCache
}

// Ensure pops the script (the default happy outcome once drained) and
// installs its PAT as the cache — the mint-side state flow. Single-flight
// like the production PatMinter: the first caller becomes the leader,
// runs the scripted mint, and publishes the outcome; the concurrent callers
// that joined the in-flight mint wake with the SAME outcome, so a burst
// increments the mint counter exactly once. A non-nil ensureHold parks the
// leader mid-mint until it closes (the concurrent arm's window pin); the
// joiner's ctx wait mirrors the real minter's cancellation arm.
func (m *scriptMinter) Ensure(ctx context.Context) (hostsvc.Pat, error) {
	m.mu.Lock()
	m.ensureStarts++
	m.mu.Unlock()

	for {
		m.mu.Lock()

		if m.minting == nil {
			m.mu.Unlock()

			return m.ensureLead(ctx)
		}

		// Join the in-flight mint (single-flight: N callers, one scripted
		// mint).
		done := m.minting
		id := m.mintID
		m.mu.Unlock()

		select {
		case <-done:
		case <-ctx.Done():
			return hostsvc.Pat{}, fmt.Errorf("scripted mint join aborted: %w", ctx.Err())
		}

		m.mu.Lock()
		joined := m.doneID == id

		var joinPat hostsvc.Pat

		if m.lastPat != nil {
			joinPat = *m.lastPat
		}

		joinErr := m.lastErr
		m.mu.Unlock()

		if joined {
			return joinPat, joinErr
		}

		// The joined slot moved on (a newer mint started) — retry the loop.
	}
}

// Invalidate drops the cached token and records the reason.
func (m *scriptMinter) Invalidate(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hasCache = false
	m.invalidates++
	m.lastReason = reason
}

// counts reads the recorded counters (ensures counts LEADER mints —
// concurrent joiners coalesce into them).
func (m *scriptMinter) counts() (ensures, invalidates int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ensures, m.invalidates
}

// startedEnsures reads the cumulative Ensure entry count (leader and joiner
// asks) — the concurrent arm's window sync point.
func (m *scriptMinter) startedEnsures() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ensureStarts
}

// ensureLead runs the leader half of the single-flight Ensure (the caller
// released mu; this method re-locks): pop the script now, park on the hold
// when the concurrent arm pinned the window, then publish the outcome for
// the joiners and install the cache on success.
func (m *scriptMinter) ensureLead(ctx context.Context) (hostsvc.Pat, error) {
	m.mu.Lock()
	m.ensures++

	out := scriptedEnsure{pat: m.defaultHappy}
	if len(m.ensureScript) > 0 {
		out = m.ensureScript[0]
		m.ensureScript = m.ensureScript[1:]
	}

	done := make(chan struct{})
	m.minting = done
	m.mintID++
	id := m.mintID
	hold := m.ensureHold
	m.mu.Unlock()

	if hold != nil {
		select {
		case <-hold:
		case <-ctx.Done():
			out = scriptedEnsure{err: fmt.Errorf("scripted mint aborted: %w", ctx.Err())}
		}
	}

	m.mu.Lock()
	if out.err == nil {
		m.cache, m.hasCache = out.pat, true

		stored := out.pat
		m.lastPat, m.lastErr = &stored, nil
	} else {
		m.lastPat, m.lastErr = nil, out.err
	}

	m.doneID = id
	m.minting = nil
	close(done)
	m.mu.Unlock()

	if out.err != nil {
		return hostsvc.Pat{}, out.err
	}

	return out.pat, nil
}

// The concurrency-window poll bounds (the core01 waitForTeeLine pattern: a
// deadline loop that fails at the bound, never a bare sleep-guess).
const (
	patWindowWait = 5 * time.Second
	patPollStep   = time.Millisecond
)

// waitForPatState polls cond until it holds or the window bound expires —
// the deterministic-window sequencer the concurrent arm pins the held mint
// with.
func waitForPatState(t *testing.T, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(patWindowWait)

	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(patPollStep)
	}

	t.Fatal("the expected concurrency state never settled within the bound")
}

// patFixture builds one fixture Pat.
func patFixture(token, keyID, fingerprint, scope string) hostsvc.Pat {
	return hostsvc.Pat{
		Token:        token,
		APIKeyID:     keyID,
		Fingerprint:  fingerprint,
		AccountScope: scope,
	}
}

// patAskExtra builds the ask params' rejection-field suffix ("" = absent;
// the JSON rides verbatim into the params object).
func patAskExtra(scope, fingerprint string) string {
	out := ""
	if scope != "" {
		out += fmt.Sprintf(`,"expectedAccountScope":%q`, scope)
	}

	if fingerprint != "" {
		out += fmt.Sprintf(`,"rejectedProjectTokenFingerprint":%q`, fingerprint)
	}

	return out
}

// patWantAnswer is the strict success-arm wire shape for the given material
// (codec-sorted requestAuth keys; an empty scope OMITS the accountScope key
// — the byte-compat two-field rule).
func patWantAnswer(scope, token, keyID string) string {
	fields := ""
	if scope != "" {
		fields += fmt.Sprintf(`"accountScope":%q,`, scope)
	}

	fields += fmt.Sprintf(`"apiKey":%q,"apiKeyId":%q`, token, keyID)

	return `{"headersApplied":true,"requestAuth":{` + fields + `}}`
}

// patEnvelope builds one runtime-headers ask; extra JSON fields (the
// rejection pair) ride verbatim.
func patEnvelope(reqID, extra string) zcode.Envelope {
	params := fmt.Sprintf(
		`{"requestId":%q,"sessionId":"sess_cp1","providerId":"zhipu-account","reason":"model-request"%s}`,
		reqID, extra)

	return zcode.Envelope{
		ID:     &zcode.ID{Str: "server-" + reqID},
		Method: interact.MethodRequestProviderRuntimeHeaders,
		Params: json.RawMessage(params),
	}
}

// servePatAsk runs one ask through the mode-aware source wired into a real
// responder and returns the marshaled wire payload plus the wire error.
func servePatAsk(
	t *testing.T, src interact.RuntimeHeadersKeySource, env zcode.Envelope,
) (wire []byte, werr *zcode.WireError) {
	t.Helper()

	resp := interact.NewResponder(interact.WithRuntimeHeadersKeySource(src))

	result, serveErr, handled := resp.Serve(context.Background(), env)
	if !handled {
		t.Fatal("the runtime-headers ask must be handled by the responder, never silence")
	}

	out, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	return out, serveErr
}

// unconfiguredFallback is the static-key source every coding-plan arm wires
// as the fallback: any wrong delegation surfaces as the api-key
// unconfigured error, failing the arm loud.
func unconfiguredFallback() interact.RuntimeHeadersKeySource {
	return hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey("", ""))
}

// codingPlanMode returns the mode selector pinned to coding-plan.
func codingPlanMode() func() string {
	return func() string { return "coding-plan" }
}

// newPatSource wires the source under test with the given minter, mode, and
// fallback (every field populated — the exhaustruct discipline).
func newPatSource(
	minter *scriptMinter, mode func() string, fallback interact.RuntimeHeadersKeySource,
) interact.RuntimeHeadersKeySource {
	return hostsvc.PatHeadersSource(hostsvc.PatHeadersSourceDeps{
		Minter:         minter,
		Mode:           mode,
		APIKeyFallback: fallback,
	})
}

// TestPatHeadersSource is the RED target: arms a-g over the mode-aware
// source (plan 06-07 Task 2).
func TestPatHeadersSource(t *testing.T) {
	t.Run("happy arm answers from the warm cache with zero mints", testPatHappyArm)

	t.Run("fingerprint match drops the cache and re-mints exactly once", testPatFingerprintArm)

	t.Run("second consecutive rejection surfaces the coded error and stops", testPatSecondRejectionArm)

	t.Run("a failed re-mint passes the minter's coded error through", testPatEnsureFailureArm)

	t.Run("scope mismatch refuses before any token is served", testPatScopeRefusalArm)

	t.Run("scope gate outranks the fingerprint re-mint", testPatScopeFirstArm)

	t.Run("empty cache routes through Ensure and answers the coded unauthenticated state", testPatEmptyCacheArm)

	t.Run("api-key mode delegates byte-compatibly to the static source", testPatAPIKeyDelegationArm)

	t.Run("malformed rejection field digests as absent and serves the cache", testPatMalformedFieldArm)

	t.Run("concurrent asks pinned in the re-mint window join one coalesced mint", testPatConcurrentArm)
}

// servePatAskInGoroutine is the goroutine-safe ask runner (no *testing.T —
// failures come back as data).
func servePatAskInGoroutine(src interact.RuntimeHeadersKeySource, env zcode.Envelope) ([]byte, *zcode.WireError) {
	resp := interact.NewResponder(interact.WithRuntimeHeadersKeySource(src))

	result, serveErr, handled := resp.Serve(context.Background(), env)
	if !handled {
		return nil, &zcode.WireError{}
	}

	out, err := json.Marshal(result)
	if err != nil {
		return nil, &zcode.WireError{}
	}

	return out, serveErr
}

// wellShapedPatAnswer reports whether the marshaled payload is the strict
// success shape with all three fields present and non-empty (the shape is
// deterministic across the pre-/post-re-mint timing; the exact token is not).
func wellShapedPatAnswer(wire []byte) bool {
	var parsed struct {
		HeadersApplied bool `json:"headersApplied"`
		RequestAuth    struct {
			APIKey       string `json:"apiKey"`
			APIKeyID     string `json:"apiKeyId"`
			AccountScope string `json:"accountScope"`
		} `json:"requestAuth"`
	}

	if err := json.Unmarshal(wire, &parsed); err != nil {
		return false
	}

	return parsed.HeadersApplied &&
		parsed.RequestAuth.APIKey != "" &&
		parsed.RequestAuth.APIKeyID != "" &&
		parsed.RequestAuth.AccountScope != ""
}

// testPatHappyArm locks the warm-cache answer: the cached three-field PAT serves with zero mints (Cached consumed).
func testPatHappyArm(t *testing.T) {
	minter := &scriptMinter{
		cache:        patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache:     true,
		defaultHappy: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPB, patFixtureScopeA),
	}

	wire, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-a", patAskExtra("", "")))

	if werr != nil {
		t.Fatalf("happy arm errored: %+v", werr)
	}

	want := patWantAnswer(patFixtureScopeA, patFixtureTokenA, patFixtureKeyIDA)
	if string(wire) != want {
		t.Fatalf("happy answer = %s, want the cached three-field answer %s", wire, want)
	}

	if ensures, _ := minter.counts(); ensures != 0 {
		t.Fatalf("happy arm minted %d time(s), want 0 (Cached consumed)", ensures)
	}
}

// testPatFingerprintArm locks the D-05 one-shot: a rejected fingerprint equal to the cached one drops the cache and
// re-mints EXACTLY once, answering the fresh token under the unchanged scope.
func testPatFingerprintArm(t *testing.T) {
	// The realistic rejection shape: the backend rejected THIS token
	// (expiry/revocation) while the sign-in is UNCHANGED — the echoed
	// expectedAccountScope equals the cached scope and the re-minted
	// token keeps it (the scope digest covers the login, not the token).
	minter := &scriptMinter{
		cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache: true,
		ensureScript: []scriptedEnsure{
			{pat: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPB, patFixtureScopeA)},
		},
	}

	wire, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-b", patAskExtra(patFixtureScopeA, patFixtureFPA)))

	if werr != nil {
		t.Fatalf("fingerprint arm errored: %+v", werr)
	}

	want := patWantAnswer(patFixtureScopeA, patFixtureTokenB, patFixtureKeyIDB)
	if string(wire) != want {
		t.Fatalf("re-mint answer = %s, want the fresh PAT-B answer %s", wire, want)
	}

	ensures, invalidates := minter.counts()
	if ensures != 1 {
		t.Fatalf("fingerprint arm minted %d time(s), want exactly 1 (the D-05 one-shot)", ensures)
	}

	if invalidates != 1 {
		t.Fatalf("fingerprint arm invalidated %d time(s), want exactly 1", invalidates)
	}
}

// testPatSecondRejectionArm locks the second-failure refusal: the re-minted fingerprint rejected again answers the
// coded ErrPatRejectionLoop class and stops (two total mints, never a loop).
func testPatSecondRejectionArm(t *testing.T) {
	minter := &scriptMinter{
		cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache: true,
		ensureScript: []scriptedEnsure{
			// The re-mint produced the SAME fingerprint again — the
			// backend rejects it a second time.
			{pat: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPA, patFixtureScopeA)},
		},
	}

	wire, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-c", patAskExtra("", patFixtureFPA)))

	if werr == nil {
		t.Fatalf("second rejection answered %s — want the coded error, never a loop", wire)
	}

	if !strings.Contains(werr.Message, hostsvc.ErrPatRejectionLoop.Error()) {
		t.Fatalf("second rejection message = %q, want the %s coded class",
			werr.Message, hostsvc.ErrPatRejectionLoop.Error())
	}

	if ensures, _ := minter.counts(); ensures != 1 {
		t.Fatalf("second-rejection arm minted %d time(s), want exactly 1 "+
			"(2 total with the original — never an infinite loop)", ensures)
	}
}

// testPatEnsureFailureArm locks the pass-through: a failed re-mint answers the minter's coded error class, never a
// fabricated success, with no silent retry.
func testPatEnsureFailureArm(t *testing.T) {
	minter := &scriptMinter{
		cache:        patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache:     true,
		ensureScript: []scriptedEnsure{{err: errFixtureEnsureFailure}},
	}

	wire, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-c2", patAskExtra("", patFixtureFPA)))

	if werr == nil {
		t.Fatalf("failed re-mint answered %s — want the coded error envelope, never a fabricated success", wire)
	}

	if !strings.Contains(werr.Message, errFixtureEnsureFailure.Error()) {
		t.Fatalf("re-mint failure message = %q, want the minter's coded class passed through", werr.Message)
	}

	if ensures, _ := minter.counts(); ensures != 1 {
		t.Fatalf("failed re-mint arm minted %d time(s), want exactly 1 (no silent retry)", ensures)
	}
}

// testPatScopeRefusalArm locks T-06-22: an expectedAccountScope mismatch answers the coded scope-invalidated error
// before any token is served, minting nothing.
func testPatScopeRefusalArm(t *testing.T) {
	minter := &scriptMinter{
		cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache: true,
	}

	wire, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-d", patAskExtra(patFixtureScopeB, "")))

	if werr == nil {
		t.Fatalf("scope mismatch answered %s — want the coded scope-invalidated error, never the stale token", wire)
	}

	if !strings.Contains(werr.Message, hostsvc.ErrPatScopeInvalidated.Error()) {
		t.Fatalf("scope error message = %q, want the %s coded class",
			werr.Message, hostsvc.ErrPatScopeInvalidated.Error())
	}

	ensures, invalidates := minter.counts()
	if ensures != 0 || invalidates != 0 {
		t.Fatalf("scope arm minted/invalidated (%d/%d), want (0/0) — refusal only", ensures, invalidates)
	}
}

// testPatScopeFirstArm locks the trigger order: the scope refusal outranks the fingerprint re-mint when both ride one
// ask.
func testPatScopeFirstArm(t *testing.T) {
	// Both triggers at once: the scope refusal must answer BEFORE the
	// re-mint fires (T-06-22 — the stale PAT would ride the old
	// entitlement; the re-mint arm only runs against the CURRENT scope).
	minter := &scriptMinter{
		cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache: true,
		ensureScript: []scriptedEnsure{
			{pat: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPB, patFixtureScopeB)},
		},
	}

	_, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-d2", patAskExtra(patFixtureScopeB, patFixtureFPA)))

	if werr == nil || !strings.Contains(werr.Message, hostsvc.ErrPatScopeInvalidated.Error()) {
		t.Fatalf("combined ask error = %+v, want the scope refusal to outrank the re-mint", werr)
	}

	if ensures, _ := minter.counts(); ensures != 0 {
		t.Fatalf("combined arm minted %d time(s), want 0 (scope gate first)", ensures)
	}
}

// testPatEmptyCacheArm locks the PAT-less arm: an empty cache routes through
// the single-flight Ensure (WR-02) and — with no desktop login — answers the
// coded unauthenticated state, the minter's coded error passed through
// unwrapped.
func testPatEmptyCacheArm(t *testing.T) {
	// The no-login script: the real minter's Ensure answers the coded
	// login_required BEFORE any network I/O when the credential store holds
	// no login (locked at the minter tier by TestPatMinterLoginRequired) —
	// the double mirrors that outcome.
	minter := &scriptMinter{
		ensureScript: []scriptedEnsure{{err: hostsvc.ErrPatLoginRequired}},
	}

	wire, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-e", patAskExtra("", "")))

	if werr == nil {
		t.Fatalf("empty cache answered %s — want the coded error naming the "+
			"unauthenticated state, never an empty success arm", wire)
	}

	if !strings.Contains(werr.Message, hostsvc.ErrPatLoginRequired.Error()) {
		t.Fatalf("PAT-less message = %q, want the %s coded class",
			werr.Message, hostsvc.ErrPatLoginRequired.Error())
	}

	if ensures, _ := minter.counts(); ensures != 1 {
		t.Fatalf("cache-miss arm minted %d time(s), want exactly 1 — the ask routes "+
			"through the single-flight Ensure (WR-02: a re-mint in flight is joined, "+
			"never answered login_required from a known-empty cache)", ensures)
	}
}

// testPatAPIKeyDelegationArm locks D-11's other half: api-key (and unauthenticated None) mode delegates byte-
// compatibly to the static source.
func testPatAPIKeyDelegationArm(t *testing.T) {
	static := hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey(patFixtureModeTk, patFixtureModeID))
	mode := func() string { return "api-key" }

	wire, werr := servePatAsk(t, newPatSource(&scriptMinter{}, mode, static),
		patEnvelope("pat-f", patAskExtra("", "")))
	if werr != nil {
		t.Fatalf("api-key delegation errored: %+v", werr)
	}

	// Byte-compat: the SAME ask through the static handler alone must
	// marshal identical bytes.
	direct, directErr := servePatAsk(t, static, patEnvelope("pat-f-direct", patAskExtra("", "")))
	if directErr != nil {
		t.Fatalf("static handler errored: %+v", directErr)
	}

	if !bytes.Equal(wire, direct) {
		t.Fatalf("api-key delegation = %s, want byte-identical %s (the static arm's answer)", wire, direct)
	}

	want := patWantAnswer("", patFixtureModeTk, patFixtureModeID)
	if string(wire) != want {
		t.Fatalf("api-key delegation = %s, want the two-field static answer %s", wire, want)
	}

	// The unauthenticated mode delegates too — the static arm's
	// unconfigured posture covers the pre-auth ask (the recorded mode is
	// empty until authenticate succeeds).
	noneMode := func() string { return "" }

	_, unauthErr := servePatAsk(t, newPatSource(&scriptMinter{}, noneMode, unconfiguredFallback()),
		patEnvelope("pat-f2", patAskExtra("", "")))
	if unauthErr == nil || !strings.Contains(unauthErr.Message, "ZCODE_ACP_API_KEY") {
		t.Fatalf("unauthenticated mode must delegate to the static arm's "+
			"unconfigured posture, got %+v", unauthErr)
	}
}

// testPatMalformedFieldArm locks the dig discipline: a malformed (or valid-but-mismatching) rejection field
// classifies as absent and serves the plain cached answer.
func testPatMalformedFieldArm(t *testing.T) {
	minter := &scriptMinter{
		cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache: true,
	}

	want := patWantAnswer(patFixtureScopeA, patFixtureTokenA, patFixtureKeyIDA)

	wire, werr := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-g", patAskExtra("", patFixtureBadHex)))

	if werr != nil {
		t.Fatalf("malformed-field arm errored: %+v", werr)
	}

	if string(wire) != want {
		t.Fatalf("malformed-field answer = %s, want the plain cached answer %s", wire, want)
	}

	if ensures, _ := minter.counts(); ensures != 0 {
		t.Fatalf("malformed field triggered %d mint(s), want 0 (a malformed field classifies as absent)", ensures)
	}

	// A well-formed fingerprint that does NOT match the cache is equally
	// not our token — the plain cached answer, no re-mint.
	wire2, werr2 := servePatAsk(t,
		newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
		patEnvelope("pat-g2", patAskExtra("", patFixtureFPB)))

	if werr2 != nil || string(wire2) != want {
		t.Fatalf("mismatching-fingerprint arm = (%s, %+v), want the plain cached answer %s", wire2, werr2, want)
	}
}

// testPatConcurrentArm locks the WR-02/WR-03 race posture: the ONE
// fingerprint ask (the D-05 staleness event) invalidates and re-mints while
// seven plain asks are pinned INSIDE the invalidate→ensure window — every
// rider routes through the single-flight Ensure and WAITS for the in-flight
// mint, answering the fresh token (never the misleading login_required a
// known-empty cache handed out before the WR-02 fix). The scripted double's
// held mint makes the window deterministic (the old free-race shape was
// flaky under -race, WR-03): every ask answers a well-shaped success and
// the counters show exactly one Invalidate+Ensure — one coalesced re-mint
// per staleness event.
func testPatConcurrentArm(t *testing.T) {
	// plainAskers ride the held window; askers is the total including the
	// one fingerprint ask.
	const plainAskers, fingerprinters, askers = 7, 1, 8

	minter := &scriptMinter{
		cache:        patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
		hasCache:     true,
		defaultHappy: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPB, patFixtureScopeA),
		ensureHold:   make(chan struct{}),
	}

	src := newPatSource(minter, codingPlanMode(), unconfiguredFallback())

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		badAsks int
	)

	// runAsk serves one ask per goroutine and counts it down (failures are
	// data, never a t.Fatal).
	runAsk := func(n int, extra string) {
		defer wg.Done()
		wire, werr := servePatAskInGoroutine(src, patEnvelope(fmt.Sprintf("pat-race-%d", n), extra))
		mu.Lock()
		defer mu.Unlock()

		if werr != nil || !wellShapedPatAnswer(wire) {
			badAsks++
		}
	}

	// 1. The fingerprint asker invalidates the stale cache and parks as the
	//    held leader — the re-mint window stays open.
	wg.Add(1)
	go runAsk(0, patAskExtra("", patFixtureFPA))

	// 2. Wait for the window: invalidation landed, the mint is parked.
	waitForPatState(t, func() bool {
		ensures, invalidates := minter.counts()

		return ensures == 1 && invalidates == 1 && minter.startedEnsures() == 1
	})

	// 3. The riders launch INTO the held window: each Cached sees the
	//    invalidated cache and joins the parked mint through Ensure.
	for i := 1; i <= plainAskers; i++ {
		wg.Add(1)
		go runAsk(i, "")
	}

	// 4. Once every ask is inside Ensure, release the mint — all wake with
	//    the SAME fresh token.
	waitForPatState(t, func() bool {
		return minter.startedEnsures() == askers
	})

	close(minter.ensureHold)
	wg.Wait()

	if badAsks != 0 {
		t.Fatalf("%d/%d concurrent asks got a wrong answer", badAsks, askers)
	}
	ensures, invalidates := minter.counts()
	if ensures != fingerprinters || invalidates != fingerprinters {
		t.Fatalf("concurrent asks minted/invalidated (%d/%d), want (%d/%d) — "+
			"exactly one re-mint per staleness event, every rider coalesced",
			ensures, invalidates, fingerprinters, fingerprinters)
	}
}
