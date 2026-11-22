package hostsvc_test

// The mode-aware PAT runtime-headers source locks (plan 06-07 Task 2,
// AUTH-02/D-05 — RESEARCH Pattern 4): every arm asserts on served answers
// and minter call counts (wire artifacts, never internals). The exactly-one
// re-mint semantics compose 06-02's single-flight Ensure; the race-heavy
// arms run under -race with concurrent asks.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// Synthetic fixture material (plan 06-07; fixture data only, never a real
// credential — T-5-15; the fingerprints/scopes are 64-lowercase-hex shape
// constants the ask schema pins).
const (
	patFixtureTokenA  = "zc-synthetic-pat-a-fixture"
	patFixtureTokenB  = "zc-synthetic-pat-b-fixture"
	patFixtureKeyIDA  = "key_pat_a"
	patFixtureKeyIDB  = "key_pat_b"
	patFixtureFPA     = "1111111111111111111111111111111111111111111111111111111111111111" // sha256-shape of PAT-A
	patFixtureFPB     = "2222222222222222222222222222222222222222222222222222222222222222" // sha256-shape of PAT-B
	patFixtureScopeA  = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	patFixtureScopeB  = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	patFixtureBadHex  = "not-hex-at-all"
	patFixtureModeKey = "zc-synthetic-static-fixture-key"
	patFixtureModeID  = "key_static_9"
)

// errFixtureEnsureFailure is the scripted Ensure failure arm (err113).
var errFixtureEnsureFailure = errors.New("fixture ensure failure")

// scriptMinter is the patMintSource double: a warm cache field the source's
// Cached arm reads, a sequential Ensure script (a default outcome once the
// script drains), and invalidation counters — the exactly-once bookkeeping
// the arms assert on.
type scriptMinter struct {
	mu           sync.Mutex
	cache        hostsvc.Pat
	hasCache     bool
	ensureScript []scriptedEnsure
	defaultHappy hostsvc.Pat
	ensures      int
	invalidates  int
	lastReason   string
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
// installs its PAT as the cache — the mint-side state flow.
func (m *scriptMinter) Ensure(_ context.Context) (hostsvc.Pat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ensures++

	if len(m.ensureScript) > 0 {
		out := m.ensureScript[0]
		m.ensureScript = m.ensureScript[1:]

		if out.err == nil {
			m.cache, m.hasCache = out.pat, true
		}

		return out.pat, out.err
	}

	m.cache, m.hasCache = m.defaultHappy, true

	return m.defaultHappy, nil
}

// Invalidate drops the cached token and records the reason.
func (m *scriptMinter) Invalidate(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hasCache = false
	m.invalidates++
	m.lastReason = reason
}

// counts reads the recorded counters.
func (m *scriptMinter) counts() (ensures, invalidates int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ensures, m.invalidates
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

// patEnvelope builds one runtime-headers ask; extra JSON fields (the
// rejection pair) ride verbatim.
func patEnvelope(reqID, extra string) zcode.Envelope {
	params := fmt.Sprintf(`{"requestId":%q,"sessionId":"sess_cp1","providerId":"zhipu-account","reason":"model-request"%s}`,
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
func newPatSource(minter *scriptMinter, mode func() string, fallback interact.RuntimeHeadersKeySource) interact.RuntimeHeadersKeySource {
	return hostsvc.PatHeadersSource(hostsvc.PatHeadersSourceDeps{
		Minter:         minter,
		Mode:           mode,
		APIKeyFallback: fallback,
	})
}

// TestPatHeadersSource is the RED target: arms a-g over the mode-aware
// source (plan 06-07 Task 2).
func TestPatHeadersSource(t *testing.T) {
	t.Run("happy arm answers from the warm cache with zero mints", func(t *testing.T) {
		minter := &scriptMinter{
			cache:        patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
			hasCache:     true,
			defaultHappy: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPB, patFixtureScopeB),
		}

		wire, werr := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-a", ""))

		if werr != nil {
			t.Fatalf("happy arm errored: %+v", werr)
		}

		want := `{"headersApplied":true,"requestAuth":{"accountScope":"` + patFixtureScopeA +
			`","apiKey":"` + patFixtureTokenA + `","apiKeyId":"` + patFixtureKeyIDA + `"}}`
		if string(wire) != want {
			t.Fatalf("happy answer = %s, want the cached three-field answer %s", wire, want)
		}

		ensures, _ := minter.counts()
		if ensures != 0 {
			t.Fatalf("happy arm minted %d time(s), want 0 (Cached consumed)", ensures)
		}
	})

	t.Run("fingerprint match drops the cache and re-mints exactly once", func(t *testing.T) {
		minter := &scriptMinter{
			cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
			hasCache: true,
			ensureScript: []scriptedEnsure{
				{pat: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPB, patFixtureScopeB)},
			},
		}

		extra := `,"expectedAccountScope":"` + patFixtureScopeA + `","rejectedProjectTokenFingerprint":"` + patFixtureFPA + `"`
		wire, werr := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-b", extra))

		if werr != nil {
			t.Fatalf("fingerprint arm errored: %+v", werr)
		}

		want := `{"headersApplied":true,"requestAuth":{"accountScope":"` + patFixtureScopeB +
			`","apiKey":"` + patFixtureTokenB + `","apiKeyId":"` + patFixtureKeyIDB + `"}}`
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
	})

	t.Run("second consecutive rejection surfaces the coded error and stops", func(t *testing.T) {
		minter := &scriptMinter{
			cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
			hasCache: true,
			ensureScript: []scriptedEnsure{
				// The re-mint produced the SAME fingerprint again — the
				// backend rejects it a second time.
				{pat: patFixture("zc-synthetic-pat-c-fixture", "key_pat_c", patFixtureFPA, patFixtureScopeA)},
			},
		}

		extra := `,"rejectedProjectTokenFingerprint":"` + patFixtureFPA + `"`
		wire, werr := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-c", extra))

		if werr == nil {
			t.Fatalf("second rejection answered %s — want the coded error, never a loop", wire)
		}

		if !strings.Contains(werr.Message, hostsvc.ErrPatRejectionLoop.Error()) {
			t.Fatalf("second rejection message = %q, want the %s coded class", werr.Message, hostsvc.ErrPatRejectionLoop.Error())
		}

		ensures, _ := minter.counts()
		if ensures != 1 {
			t.Fatalf("second-rejection arm minted %d time(s), want exactly 1 (2 total with the original — never an infinite loop)", ensures)
		}
	})

	t.Run("scope mismatch refuses before any token is served", func(t *testing.T) {
		minter := &scriptMinter{
			cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
			hasCache: true,
		}

		extra := `,"expectedAccountScope":"` + patFixtureScopeB + `"`
		wire, werr := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-d", extra))

		if werr == nil {
			t.Fatalf("scope mismatch answered %s — want the coded scope-invalidated error, never the stale token", wire)
		}

		if !strings.Contains(werr.Message, hostsvc.ErrPatScopeInvalidated.Error()) {
			t.Fatalf("scope error message = %q, want the %s coded class", werr.Message, hostsvc.ErrPatScopeInvalidated.Error())
		}

		ensures, invalidates := minter.counts()
		if ensures != 0 || invalidates != 0 {
			t.Fatalf("scope arm minted/invalidated (%d/%d), want (0/0) — refusal only", ensures, invalidates)
		}
	})

	t.Run("scope gate outranks the fingerprint re-mint", func(t *testing.T) {
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

		extra := `,"expectedAccountScope":"` + patFixtureScopeB + `","rejectedProjectTokenFingerprint":"` + patFixtureFPA + `"`
		_, werr := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-d2", extra))

		if werr == nil || !strings.Contains(werr.Message, hostsvc.ErrPatScopeInvalidated.Error()) {
			t.Fatalf("combined ask error = %+v, want the scope refusal to outrank the re-mint", werr)
		}

		ensures, _ := minter.counts()
		if ensures != 0 {
			t.Fatalf("combined arm minted %d time(s), want 0 (scope gate first)", ensures)
		}
	})

	t.Run("empty cache answers the coded unauthenticated state", func(t *testing.T) {
		minter := &scriptMinter{} // no cache — pre-auth / post-invalidate

		wire, werr := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-e", ""))

		if werr == nil {
			t.Fatalf("empty cache answered %s — want the coded error naming the unauthenticated state, never an empty success arm", wire)
		}

		if !strings.Contains(werr.Message, hostsvc.ErrPatLoginRequired.Error()) {
			t.Fatalf("PAT-less message = %q, want the %s coded class", werr.Message, hostsvc.ErrPatLoginRequired.Error())
		}

		ensures, _ := minter.counts()
		if ensures != 0 {
			t.Fatalf("PAT-less arm minted %d time(s), want 0 (the ask-side never mints into an unauthenticated state)", ensures)
		}
	})

	t.Run("api-key mode delegates byte-compatibly to the static source", func(t *testing.T) {
		static := hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey(patFixtureModeKey, patFixtureModeID))

		mode := func() string { return "api-key" }

		wire, werr := servePatAsk(t, newPatSource(&scriptMinter{}, mode, static), patEnvelope("pat-f", ""))
		if werr != nil {
			t.Fatalf("api-key delegation errored: %+v", werr)
		}

		// Byte-compat: the SAME ask through the static handler alone must
		// marshal identical bytes.
		direct, directErr := servePatAsk(t, static, patEnvelope("pat-f-direct", ""))
		if directErr != nil {
			t.Fatalf("static handler errored: %+v", directErr)
		}

		if string(wire) != string(direct) {
			t.Fatalf("api-key delegation = %s, want byte-identical %s (the static arm's answer)", wire, direct)
		}

		want := `{"headersApplied":true,"requestAuth":{"apiKey":"` + patFixtureModeKey + `","apiKeyId":"` + patFixtureModeID + `"}}`
		if string(wire) != want {
			t.Fatalf("api-key delegation = %s, want the two-field static answer %s", wire, want)
		}

		// The unauthenticated mode delegates too — the static arm's
		// unconfigured posture covers the pre-auth ask (the recorded mode is
		// empty until authenticate succeeds).
		noneMode := func() string { return "" }

		_, unauthErr := servePatAsk(t, newPatSource(&scriptMinter{}, noneMode, unconfiguredFallback()), patEnvelope("pat-f2", ""))
		if unauthErr == nil || !strings.Contains(unauthErr.Message, "ZCODE_ACP_API_KEY") {
			t.Fatalf("unauthenticated mode must delegate to the static arm's unconfigured posture, got %+v", unauthErr)
		}
	})

	t.Run("malformed rejection field digests as absent and serves the cache", func(t *testing.T) {
		minter := &scriptMinter{
			cache:    patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
			hasCache: true,
		}

		extra := `,"rejectedProjectTokenFingerprint":"` + patFixtureBadHex + `"`
		wire, werr := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-g", extra))

		if werr != nil {
			t.Fatalf("malformed-field arm errored: %+v", werr)
		}

		want := `{"headersApplied":true,"requestAuth":{"accountScope":"` + patFixtureScopeA +
			`","apiKey":"` + patFixtureTokenA + `","apiKeyId":"` + patFixtureKeyIDA + `"}}`
		if string(wire) != want {
			t.Fatalf("malformed-field answer = %s, want the plain cached answer %s", wire, want)
		}

		ensures, _ := minter.counts()
		if ensures != 0 {
			t.Fatalf("malformed field triggered %d mint(s), want 0 (a malformed field classifies as absent)", ensures)
		}

		// A well-formed fingerprint that does NOT match the cache is equally
		// not our token — the plain cached answer, no re-mint.
		extra = `,"rejectedProjectTokenFingerprint":"` + patFixtureFPB + `"`
		wire2, werr2 := servePatAsk(t,
			newPatSource(minter, codingPlanMode(), unconfiguredFallback()),
			patEnvelope("pat-g2", extra))

		if werr2 != nil || string(wire2) != want {
			t.Fatalf("mismatching-fingerprint arm = (%s, %+v), want the plain cached answer %s", wire2, werr2, want)
		}
	})

	t.Run("concurrent fingerprint asks each re-mint exactly once", func(t *testing.T) {
		const askers = 8

		minter := &scriptMinter{
			cache:        patFixture(patFixtureTokenA, patFixtureKeyIDA, patFixtureFPA, patFixtureScopeA),
			hasCache:     true,
			defaultHappy: patFixture(patFixtureTokenB, patFixtureKeyIDB, patFixtureFPB, patFixtureScopeB),
		}

		src := newPatSource(minter, codingPlanMode(), unconfiguredFallback())
		extra := `,"rejectedProjectTokenFingerprint":"` + patFixtureFPA + `"`

		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			badAsks int
		)

		for i := range askers {
			wg.Add(1)

			go func(n int) {
				defer wg.Done()

				wire, werr := servePatAskInGoroutine(src, patEnvelope(fmt.Sprintf("pat-race-%d", n), extra))
				mu.Lock()
				defer mu.Unlock()

				if werr != nil || string(wire) != patWantB() {
					badAsks++
				}
			}(i)
		}

		wg.Wait()

		if badAsks != 0 {
			t.Fatalf("%d/%d concurrent asks got a wrong answer", badAsks, askers)
		}

		ensures, invalidates := minter.counts()
		if ensures != askers || invalidates != askers {
			t.Fatalf("concurrent asks minted/invalidated (%d/%d), want (%d/%d) — exactly one re-mint per ask",
				ensures, invalidates, askers, askers)
		}
	})
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

// patWantB is the fresh-PAT-B wire shape the race arm asserts on.
func patWantB() string {
	return `{"headersApplied":true,"requestAuth":{"accountScope":"` + patFixtureScopeB +
		`","apiKey":"` + patFixtureTokenB + `","apiKeyId":"` + patFixtureKeyIDB + `"}}`
}
