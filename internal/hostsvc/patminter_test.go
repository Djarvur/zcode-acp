package hostsvc_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
)

// AUTH-02 PAT mint execution-tier contract locks (plan 06-02): the OFFICIAL
// api.z.ai chain over httptest vectors — getCustomerInfo with the RAW login
// token (NO Bearer; a Bearer is misjudged login-invalid), the api_keys
// list/create and access_tokens calls with "Bearer " + token, byte-exact
// bodies, the byte-parity accountScope digest, and the expiry/refresh math.
// CI never touches the real api.z.ai (the origin is injectable ONLY for
// these vectors).
//
// Hygiene: every fixture value is synthetic ("zc-synthetic-*", "key-*") —
// the scan-secrets gate stays green.

// The synthetic fixture material (never a real credential).
const (
	fixtureLoginToken     = "zc-synthetic-login-token"
	fixtureJWTToken       = "zc-synthetic-jwt-token"
	fixturePATToken       = "zc-synthetic-pat-token"
	fixtureDesktopVersion = "9.9.9-fixture"
	fixtureOrgID          = "org-1"
	fixtureProjID         = "proj-1"
	fixtureKeyID          = "key-7"
	fixtureKeyListPath    = "/api/biz/v1/organization/org-1/projects/proj-1/api_keys"
	fixtureAccessPath     = "/api/biz/v1/organization/org-1/projects/proj-1/api_keys/key-7/access_tokens"
)

// fixtureBaseTime returns the pinned clock base: the expiry/refresh math
// asserts against deterministic time, never wall time. (A function keeps the
// fixture gochecknoglobals-clean.)
func fixtureBaseTime() time.Time {
	return time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
}

// pinnedAccountScopeDigest is the sha256 hex over the COMPACT JSON array
// ["zai","individual-coding-plan",null,"zc-synthetic-login-token",null]
// (JSON.stringify parity — no whitespace). The test recomputes the digest
// from the same array construction AND pins the literal, so either side
// drifting (encoding change, stdlib change) trips.
const pinnedAccountScopeDigest = "8280bb48d1bf3bb1cd726406d2393769174ca9e940228c2d9f1fb617cad162c9"

// The default org/project markers are the desktop's server-side literals
// (the D-06 pick preference); the escapes keep the fixture source ASCII
// (gosmopolitan's Han-script literal gate) while the VALUES are the exact
// server literals.
const (
	fixtureDefaultOrgMarker  = "\u9ed8\u8ba4\u673a\u6784" // 默认机构
	fixtureDefaultProjMarker = "\u9ed8\u8ba4\u9879\u76ee" // 默认项目
)

// The happy-path reply envelopes (the server-side shapes of Pattern 3): one
// personal org carrying the default markers, one personal project of a
// non-team type; the key list carries an unrelated entry plus the candidate.
func fixtureCustomerReply() string {
	return fmt.Sprintf(
		`{"code":200,"data":{"organizations":[{"organizationId":"org-1",`+
			`"organizationName":%q,"projects":[{"projectId":"proj-1",`+
			`"projectName":%q,"projectType":"1"}]}]}}`,
		fixtureDefaultOrgMarker, fixtureDefaultProjMarker)
}

func fixtureKeyListReply() string {
	return `{"code":200,"data":[` +
		`{"name":"other-key","apiKey":"key-other"},` +
		`{"name":"zcode-api-key","apiKey":"key-7"}]}`
}

// fixtureAccessReply builds the access_tokens success envelope with the
// pinned expiry sources (expiresIn 3600s; expiresAt = base+1h in epoch sec).
func fixtureAccessReply() string {
	return fmt.Sprintf(
		`{"code":200,"data":{"accessToken":%q,"tokenType":"Bearer","expiresIn":3600,"expiresAt":%d}}`,
		fixturePATToken, fixtureBaseTime().Add(time.Hour).Unix())
}

// patFixtureCall is one recorded mint-chain request: the wire shape the
// vectors assert (method, path, Authorization, Content-Type, body).
type patFixtureCall struct {
	method      string
	path        string
	auth        string
	contentType string
	body        string
}

// patReplier produces one stage's HTTP status + body. A nil createKey replier
// means the vector does NOT expect a create POST — one arriving is a coded
// 500 the request-count assertions catch.
type patReplier func() (int, string)

// patFixtureServer is the httptest origin standing in for api.z.ai: it
// records every mint-chain call (order-preserving) and serves the stage
// replies its hooks configure (nil hook = the happy default).
type patFixtureServer struct {
	t   *testing.T
	srv *httptest.Server

	mu        sync.Mutex
	calls     []patFixtureCall
	customer  patReplier
	listKeys  patReplier
	createKey patReplier
	access    patReplier
	netFail   bool // hijack+close every request while set (transport-level failure)
}

// newPatFixtureServer starts the fixture origin with the happy defaults and
// stops it at cleanup.
func newPatFixtureServer(t *testing.T) *patFixtureServer {
	t.Helper()

	fx := &patFixtureServer{
		t:         t,
		customer:  patReplyStatusBody(http.StatusOK, fixtureCustomerReply()),
		listKeys:  patReplyStatusBody(http.StatusOK, fixtureKeyListReply()),
		createKey: nil,
		access:    patReplyStatusBody(http.StatusOK, fixtureAccessReply()),
	}
	fx.srv = httptest.NewServer(fx)

	t.Cleanup(fx.srv.Close)

	return fx
}

// patReplyStatusBody is a constant status+body replier.
func patReplyStatusBody(status int, body string) patReplier {
	return func() (int, string) { return status, body }
}

// Calls returns the recorded call slice (order-preserving snapshot).
func (f *patFixtureServer) Calls() []patFixtureCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]patFixtureCall, len(f.calls))
	copy(out, f.calls)

	return out
}

// ServeHTTP records the wire shape and serves the configured stage reply.
func (f *patFixtureServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))

	f.mu.Lock()
	f.calls = append(f.calls, patFixtureCall{
		method:      r.Method,
		path:        r.URL.Path,
		auth:        r.Header.Get("Authorization"),
		contentType: r.Header.Get("Content-Type"),
		body:        string(body),
	})
	f.mu.Unlock()

	if f.netFail {
		// Transport-level failure: hijack and drop the connection so the
		// client observes a reset (a network error, not an HTTP status).
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, err := hj.Hijack()
			if err == nil {
				_ = conn.Close()
			}
		}

		return
	}

	var rep patReplier

	switch {
	case r.URL.Path == "/api/biz/customer/getCustomerInfo":
		rep = f.customer
	case r.URL.Path == fixtureKeyListPath && r.Method == http.MethodGet:
		rep = f.listKeys
	case r.URL.Path == fixtureKeyListPath && r.Method == http.MethodPost:
		rep = f.createKey
		if rep == nil {
			rep = patReplyStatusBody(http.StatusNotImplemented,
				`{"code":500,"data":null}`)
		}
	case strings.HasSuffix(r.URL.Path, "/access_tokens"):
		rep = f.access
	default:
		rep = patReplyStatusBody(http.StatusNotFound, `{"code":404,"data":null}`)
	}

	status, reply := rep()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(reply))
}

// fixturePatCreds is the credential seam fixture: the 06-01 accessor contract
// (login token Bearer-stripped; JWT present only when set).
type fixturePatCreds struct {
	login   string
	jwt     string
	loginOK bool
}

// LoginToken returns the fixture login (already Bearer-stripped per the seam
// contract).
func (c *fixturePatCreds) LoginToken(_ context.Context, _ string) (string, bool) {
	return c.login, c.loginOK
}

// ZcodeJWT returns the fixture JWT; ok=false when unset (folds as null).
func (c *fixturePatCreds) ZcodeJWT(_ context.Context) (string, bool) {
	return c.jwt, c.jwt != ""
}

// patClock is the mutable fixture clock: pinned at fixtureBaseTime until a
// vector advances it (the expiry/refresh/backoff math stays deterministic).
type patClock struct {
	mu  sync.Mutex
	now time.Time
}

// newPatClock pins the clock at start.
func newPatClock(start time.Time) *patClock {
	return &patClock{now: start}
}

// Now renders the current fixture time (the minter's clock seam).
func (c *patClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

// Advance moves the fixture clock forward by d.
func (c *patClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

// newFixtureMinter builds the minter over the fixture origin, creds, clock,
// and the pinned fixture desktop version.
func newFixtureMinter(
	t *testing.T,
	origin string,
	creds hostsvc.PatCredentialSource,
	clock func() time.Time,
) *hostsvc.PatMinter {
	t.Helper()

	return hostsvc.NewPatMinter(
		hostsvc.WithPatOrigin(origin),
		hostsvc.WithPatClock(clock),
		hostsvc.WithPatClientVersion(func() string { return fixtureDesktopVersion }),
		hostsvc.WithPatCredentials(creds),
	)
}

// capturePatSlogWarns redirects the default slog to a Warn-level buffer and
// restores it at cleanup — the D-06 multi-pick warn arms assert through it,
// and the happy path asserts its ABSENCE (a single candidate never warns).
func capturePatSlogWarns(t *testing.T) *bytes.Buffer {
	t.Helper()

	prev := slog.Default()
	buf := &bytes.Buffer{}

	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return buf
}

// patSHA256HexTest is the test-side sha256-hex helper (the fingerprint and
// digest form).
func patSHA256HexTest(s string) string {
	sum := sha256.Sum256([]byte(s))

	return hex.EncodeToString(sum[:])
}

// assertPatCustomerCall locks call 1's wire shape: the RAW login token in
// Authorization (a Bearer prefix is the misjudged-login-invalid trap),
// application/json, the official path.
func assertPatCustomerCall(t *testing.T, call patFixtureCall) {
	t.Helper()

	const customerPath = "/api/biz/customer/getCustomerInfo"
	if call.path != customerPath {
		t.Fatalf("call 1 path = %q, want %s", call.path, customerPath)
	}

	if call.auth != fixtureLoginToken {
		t.Fatalf("call 1 Authorization = %q, want the RAW login token %q", call.auth, fixtureLoginToken)
	}

	if strings.HasPrefix(call.auth, "Bearer") {
		t.Fatalf("call 1 carried a Bearer prefix on getCustomerInfo — the server misjudges it "+
			"login-invalid (Pattern 3 step 2); auth = %q", call.auth)
	}

	if call.contentType != "application/json" {
		t.Fatalf("call 1 Content-Type = %q, want application/json", call.contentType)
	}
}

// assertPatListCall locks call 2's wire shape: Bearer form (zai) on the
// api_keys collection URL, application/json.
func assertPatListCall(t *testing.T, call patFixtureCall) {
	t.Helper()

	if call.path != fixtureKeyListPath {
		t.Fatalf("call 2 path = %q, want %s", call.path, fixtureKeyListPath)
	}

	if call.auth != "Bearer "+fixtureLoginToken {
		t.Fatalf("call 2 Authorization = %q, want Bearer %s", call.auth, fixtureLoginToken)
	}

	if call.contentType != "application/json" {
		t.Fatalf("call 2 Content-Type = %q, want application/json", call.contentType)
	}
}

// assertPatAccessCall locks call 3's wire shape: Bearer POST on the key's
// access_tokens URL with the byte-exact clientType/clientVersion body.
func assertPatAccessCall(t *testing.T, call patFixtureCall) {
	t.Helper()

	if call.path != fixtureAccessPath {
		t.Fatalf("call 3 path = %q, want %s", call.path, fixtureAccessPath)
	}

	if call.method != http.MethodPost {
		t.Fatalf("call 3 method = %q, want POST", call.method)
	}

	if call.auth != "Bearer "+fixtureLoginToken {
		t.Fatalf("call 3 Authorization = %q, want Bearer %s", call.auth, fixtureLoginToken)
	}

	wantBody := fmt.Sprintf(`{"clientType":"zcode","clientVersion":%q}`, fixtureDesktopVersion)
	if call.body != wantBody {
		t.Fatalf("call 3 body = %q, want the byte-exact %s", call.body, wantBody)
	}
}

// assertHappyPat locks the Pat field contract: minted token, the listed key
// id, the token's sha256 fingerprint, the pinned accountScope, and the
// min/max expiry + refresh math over the pinned clock.
func assertHappyPat(t *testing.T, pat hostsvc.Pat) {
	t.Helper()

	if pat.Token != fixturePATToken {
		t.Fatalf("Pat.Token = %q, want the minted fixture PAT", pat.Token)
	}

	if pat.APIKeyID != fixtureKeyID {
		t.Fatalf("Pat.APIKeyID = %q, want the listed candidate %s", pat.APIKeyID, fixtureKeyID)
	}

	if pat.Fingerprint != patSHA256HexTest(fixturePATToken) {
		t.Fatalf("Pat.Fingerprint = %q, want sha256 hex of the token", pat.Fingerprint)
	}

	if pat.AccountScope != pinnedAccountScopeDigest {
		t.Fatalf("Pat.AccountScope = %q, want the pinned digest %s", pat.AccountScope, pinnedAccountScopeDigest)
	}

	// Expiry math: expiresAt = min(expiresAtSec*1000, started+expiresIn*1000)
	// — both sources agree at base+1h in this fixture.
	wantExpiry := fixtureBaseTime().Add(time.Hour)
	if !pat.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("Pat.ExpiresAt = %v, want %v (min of both sources)", pat.ExpiresAt, wantExpiry)
	}

	// refreshAt = max(now, expiresAt-120s) — the refresh lead, floored at now.
	wantRefresh := wantExpiry.Add(-120 * time.Second)
	if !pat.RefreshAt.Equal(wantRefresh) {
		t.Fatalf("Pat.RefreshAt = %v, want %v (expiry minus the 120s refresh lead)", pat.RefreshAt, wantRefresh)
	}

	if !pat.RefreshAt.After(fixtureBaseTime()) {
		t.Fatalf("Pat.RefreshAt = %v is not in the future relative to the pinned now", pat.RefreshAt)
	}
}

// TestPatMinterHappyPath locks the tracer slice: the five-call official chain
// mints a PAT end-to-end against the fixture origin with every wire shape
// asserted in order — the RAW-token getCustomerInfo (a Bearer here is
// misjudged login-invalid), the Bearer api_keys list, the Bearer access_tokens
// with the byte-exact body — plus the Pat field contract and the single
// personal org/project picking a candidate with NO warn log (D-06).
func TestPatMinterHappyPath(t *testing.T) {
	fx := newPatFixtureServer(t)
	warns := capturePatSlogWarns(t)
	clk := newPatClock(fixtureBaseTime())
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken}, clk.Now)

	pat, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure = error %v, want the official chain to mint a PAT", err)
	}

	calls := fx.Calls()
	if len(calls) != 3 {
		t.Fatalf("fixture recorded %d calls, want exactly 3 (getCustomerInfo, api_keys list, "+
			"access_tokens): %+v", len(calls), calls)
	}

	assertPatCustomerCall(t, calls[0])
	assertPatListCall(t, calls[1])
	assertPatAccessCall(t, calls[2])
	assertHappyPat(t, pat)

	// A single personal org/project picks deterministically — never a warn.
	if warns.String() != "" {
		t.Fatalf("warn log recorded for a single-candidate pick (D-06: multi-candidates only): %s", warns.String())
	}
}

// TestAccountScopeDigest locks the accountScope byte-parity: sha256 over the
// COMPACT JSON array [family, plan kind, null, loginToken, zcodeJwtOrNil]
// (JSON.stringify parity — no whitespace, lowercase 64-hex), recomputed from
// the same array construction AND pinned against the literal vector; the nil
// JWT folds as null, a present JWT as a string.
func TestAccountScopeDigest(t *testing.T) {
	t.Run("pinned vector matches the same-array construction", func(t *testing.T) {
		payload, merr := json.Marshal([]any{"zai", "individual-coding-plan", nil, fixtureLoginToken, nil})
		if merr != nil {
			t.Fatalf("marshal scope array: %v", merr)
		}

		sum := sha256.Sum256(payload)
		if got := hex.EncodeToString(sum[:]); got != pinnedAccountScopeDigest {
			t.Fatalf("recomputed digest = %s, want the pinned %s (the array encoding drifted)",
				got, pinnedAccountScopeDigest)
		}
	})

	t.Run("nil JWT folds as null and the mint carries the pinned scope", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken}, clk.Now)

		pat, err := m.Ensure(context.Background())
		if err != nil {
			t.Fatalf("Ensure = error %v, want a mint", err)
		}

		if pat.AccountScope != pinnedAccountScopeDigest {
			t.Fatalf("Pat.AccountScope = %q, want the pinned nil-JWT digest %q",
				pat.AccountScope, pinnedAccountScopeDigest)
		}
	})

	t.Run("present JWT folds as a string in the array", func(t *testing.T) {
		payload, merr := json.Marshal([]any{"zai", "individual-coding-plan", nil, fixtureLoginToken, fixtureJWTToken})
		if merr != nil {
			t.Fatalf("marshal scope array: %v", merr)
		}

		sum := sha256.Sum256(payload)
		want := hex.EncodeToString(sum[:])

		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		m := newFixtureMinter(t, fx.srv.URL,
			&fixturePatCreds{login: fixtureLoginToken, jwt: fixtureJWTToken}, clk.Now)

		pat, err := m.Ensure(context.Background())
		if err != nil {
			t.Fatalf("Ensure = error %v, want a mint", err)
		}

		if pat.AccountScope != want {
			t.Fatalf("Pat.AccountScope = %q, want the string-JWT digest %q", pat.AccountScope, want)
		}
	})
}
