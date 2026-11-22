package hostsvc_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	patSecondToken        = "zc-synthetic-pat-token-b"
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

// patCustomerPath is the fixture's customer-info route.
const patCustomerPath = "/api/biz/customer/getCustomerInfo"

// SetCustomer replaces the customer-info replier (arm vectors reconfigure the
// origin mid-test).
func (f *patFixtureServer) SetCustomer(r patReplier) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.customer = r
}

// SetListKeys replaces the api_keys list replier.
func (f *patFixtureServer) SetListKeys(r patReplier) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.listKeys = r
}

// SetCreateKey replaces the api_keys create replier (nil resets to the
// unexpected-create 500).
func (f *patFixtureServer) SetCreateKey(r patReplier) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.createKey = r
}

// SetAccess replaces the access_tokens replier.
func (f *patFixtureServer) SetAccess(r patReplier) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.access = r
}

// SetNetFail toggles transport-level failure mode (hijack + drop).
func (f *patFixtureServer) SetNetFail(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.netFail = v
}

// Calls returns the recorded call slice (order-preserving snapshot).
func (f *patFixtureServer) Calls() []patFixtureCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]patFixtureCall, len(f.calls))
	copy(out, f.calls)

	return out
}

// CreateCall returns the recorded api_keys POST (ok=false when none fired).
func (f *patFixtureServer) CreateCall() (patFixtureCall, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for i := range f.calls {
		c := f.calls[i]
		if strings.HasSuffix(c.path, "/api_keys") && c.method == http.MethodPost {
			return c, true
		}
	}

	return patFixtureCall{}, false
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
	netFail := f.netFail
	f.mu.Unlock()

	if netFail {
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
	case r.URL.Path == patCustomerPath:
		rep = f.customer
	case strings.HasSuffix(r.URL.Path, "/api_keys") && r.Method == http.MethodGet:
		rep = f.listKeys
	case strings.HasSuffix(r.URL.Path, "/api_keys") && r.Method == http.MethodPost:
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

// patCallCount returns how many recorded calls hit the given stage kind
// ("customer" | "list" | "create" | "access"), matched by path/method.
func (f *patFixtureServer) patCallCount(kind string) int {
	calls := f.Calls()

	n := 0
	for _, c := range calls {
		switch {
		case kind == "customer" && c.path == patCustomerPath:
			n++
		case kind == "list" && strings.HasSuffix(c.path, "/api_keys") && c.method == http.MethodGet:
			n++
		case kind == "create" && strings.HasSuffix(c.path, "/api_keys") && c.method == http.MethodPost:
			n++
		case kind == "access" && strings.HasSuffix(c.path, "/access_tokens"):
			n++
		}
	}

	return n
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
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

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
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

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
			&fixturePatCreds{login: fixtureLoginToken, jwt: fixtureJWTToken, loginOK: true}, clk.Now)

		pat, err := m.Ensure(context.Background())
		if err != nil {
			t.Fatalf("Ensure = error %v, want a mint", err)
		}

		if pat.AccountScope != want {
			t.Fatalf("Pat.AccountScope = %q, want the string-JWT digest %q", pat.AccountScope, want)
		}
	})
}

// patCustomerEnvelope wraps an organizations JSON array in the success
// envelope (arm vectors build custom account shapes).
func patCustomerEnvelope(orgsJSON string) string {
	return `{"code":200,"data":{"organizations":` + orgsJSON + `}}`
}

// patOrgJSON renders one organization entry.
func patOrgJSON(id, name string, projects ...string) string {
	return fmt.Sprintf(`{"organizationId":%q,"organizationName":%q,"projects":[%s]}`,
		id, name, strings.Join(projects, ","))
}

// patProjJSON renders one project entry.
func patProjJSON(id, name, typ string) string {
	return fmt.Sprintf(`{"projectId":%q,"projectName":%q,"projectType":%q}`, id, name, typ)
}

// patKeyEntry renders one api_keys entry.
func patKeyEntry(name, apiKey string) string {
	return fmt.Sprintf(`{"name":%q,"apiKey":%q}`, name, apiKey)
}

// patKeyListEnvelope wraps key-entry JSON objects in the success envelope.
func patKeyListEnvelope(entries ...string) string {
	return `{"code":200,"data":[` + strings.Join(entries, ",") + `]}`
}

// patAccessEnvelope wraps an access_tokens data object in the success
// envelope.
func patAccessEnvelope(dataJSON string) string {
	return `{"code":200,"data":` + dataJSON + `}`
}

// patAccessData renders a successful access_tokens data object (the pinned
// expiry sources: expiresIn 3600s, expiresAt base+1h epoch sec).
func patAccessData(token string) string {
	return patAccessDataFrom(fixtureBaseTime(), token)
}

// patAccessDataFrom renders the access_tokens data object with the expiry
// sources anchored at start (vectors that advance the clock re-anchor so the
// recovery mint's reply stays fresh relative to the fixture now).
func patAccessDataFrom(start time.Time, token string) string {
	return fmt.Sprintf(`{"accessToken":%q,"tokenType":"Bearer","expiresIn":3600,"expiresAt":%d}`,
		token, start.Add(time.Hour).Unix())
}

// TestPatMinterLoginRequired locks the login-required arm: an absent desktop
// login answers the coded error naming the desktop login, makes ZERO HTTP
// calls, and never fabricates success or poisons the cache (AUTH-02 "empty"
// probe posture).
func TestPatMinterLoginRequired(t *testing.T) {
	t.Run("absent login token answers the coded desktop-login arm with zero HTTP", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{loginOK: false}, clk.Now)

		_, err := m.Ensure(context.Background())
		if !errors.Is(err, hostsvc.ErrPatLoginRequired) {
			t.Fatalf("Ensure = error %v, want ErrPatLoginRequired", err)
		}

		if !strings.Contains(err.Error(), "log in via the ZCode desktop first") {
			t.Fatalf("error %q does not name the desktop login (the actionable cue)", err)
		}

		if calls := fx.Calls(); len(calls) != 0 {
			t.Fatalf("recorded %d calls, want ZERO HTTP before the login check: %+v", len(calls), calls)
		}

		if _, ok := m.Cached(context.Background()); ok {
			t.Fatal("cache populated on the login-required arm — partial poisoning")
		}
	})

	t.Run("whitespace-only login token answers the same arm", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: "   ", loginOK: true}, clk.Now)

		_, err := m.Ensure(context.Background())
		if !errors.Is(err, hostsvc.ErrPatLoginRequired) {
			t.Fatalf("Ensure = error %v, want ErrPatLoginRequired", err)
		}

		if calls := fx.Calls(); len(calls) != 0 {
			t.Fatalf("recorded %d calls, want ZERO HTTP: %+v", len(calls), calls)
		}
	})
}

// TestPatMinterPickUnavailable locks the project-unavailable arm: an account
// with no personal candidates (all-team projects, or zero organizations)
// answers the coded error after exactly one customer call and never
// fabricates success.
func TestPatMinterPickUnavailable(t *testing.T) {
	t.Run("all-team account answers project-unavailable", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		fx.SetCustomer(patReplyStatusBody(http.StatusOK, patCustomerEnvelope("["+
			patOrgJSON("org-team", "Team Org", patProjJSON("proj-team", "Team Proj", "2"))+"]")))
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

		_, err := m.Ensure(context.Background())
		if !errors.Is(err, hostsvc.ErrPatProjectUnavailable) {
			t.Fatalf("Ensure = error %v, want ErrPatProjectUnavailable "+
				"(team projects are never the personal home)", err)
		}

		if n := fx.patCallCount("customer"); n != 1 {
			t.Fatalf("customer calls = %d, want exactly 1 (the chain stops at the pick)", n)
		}

		if n := fx.patCallCount("access"); n != 0 {
			t.Fatalf("access_tokens calls = %d, want 0 (no fabricated success)", n)
		}

		if _, ok := m.Cached(context.Background()); ok {
			t.Fatal("cache populated on the pick-unavailable arm")
		}
	})

	t.Run("zero organizations answers project-unavailable", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		fx.SetCustomer(patReplyStatusBody(http.StatusOK, patCustomerEnvelope(`[]`)))
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

		_, err := m.Ensure(context.Background())
		if !errors.Is(err, hostsvc.ErrPatProjectUnavailable) {
			t.Fatalf("Ensure = error %v, want ErrPatProjectUnavailable", err)
		}

		if n := fx.patCallCount("list"); n != 0 {
			t.Fatalf("api_keys calls = %d, want 0", n)
		}
	})
}

// TestPatMinterMultiPickWarn locks D-06: several personal orgs/projects pick
// deterministically (marker preference first, deterministic first otherwise)
// and a slog warn lists the candidate ids — the user is never blocked, and a
// multi-candidate account is never silently collapsed.
func TestPatMinterMultiPickWarn(t *testing.T) {
	t.Run("several candidates pick deterministic first and warn the ids", patRunMultiPickFirst)
	t.Run("the default-marker org and project win over the first", patRunMultiPickMarkers)
}

// patRunMultiPickFirst runs the deterministic-first D-06 vector: several
// personal orgs/projects still mint (never block) against the FIRST pair,
// with a warn listing every candidate id.
func patRunMultiPickFirst(t *testing.T) {
	fx := newPatFixtureServer(t)
	warns := capturePatSlogWarns(t)
	clk := newPatClock(fixtureBaseTime())
	org1 := patOrgJSON("org-1", "Org One",
		patProjJSON("proj-1a", "Proj A", "1"), patProjJSON("proj-1b", "Proj B", "1"))
	org2 := patOrgJSON("org-2", "Org Two", patProjJSON("proj-2", "Proj Two", "1"))
	fx.SetCustomer(patReplyStatusBody(http.StatusOK,
		patCustomerEnvelope("["+org1+","+org2+"]")))
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	pat, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure = error %v, want the deterministic-first mint (D-06: never block)", err)
	}

	calls := fx.Calls()
	if len(calls) < 3 || !strings.HasSuffix(calls[1].path, "/organization/org-1/projects/proj-1a/api_keys") {
		t.Fatalf("list path = %+v, want the deterministic-first org-1/proj-1a pair", calls)
	}

	if !strings.HasSuffix(calls[2].path, "/api_keys/key-7/access_tokens") {
		t.Fatalf("access path = %q, want the picked pair's key", calls[2].path)
	}

	if pat.APIKeyID != fixtureKeyID {
		t.Fatalf("Pat.APIKeyID = %q, want the candidate under the picked pair", pat.APIKeyID)
	}

	for _, id := range []string{"org-1", "org-2", "proj-1a", "proj-1b"} {
		if !strings.Contains(warns.String(), id) {
			t.Fatalf("warn log %q does not list candidate id %q (D-06)", warns.String(), id)
		}
	}
}

// patRunMultiPickMarkers runs the marker-preference D-06 vector: the
// default-marker org and project win over the deterministic first.
func patRunMultiPickMarkers(t *testing.T) {
	fx := newPatFixtureServer(t)
	clk := newPatClock(fixtureBaseTime())
	orgA := patOrgJSON("org-a", "Org A", patProjJSON("proj-a", "Proj A", "1"))
	orgB := patOrgJSON("org-b", fixtureDefaultOrgMarker,
		patProjJSON("proj-b1", "Proj B1", "1"),
		patProjJSON("proj-b2", fixtureDefaultProjMarker, "1"))
	fx.SetCustomer(patReplyStatusBody(http.StatusOK,
		patCustomerEnvelope("["+orgA+","+orgB+"]")))
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	pat, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure = error %v, want the marker-preference mint", err)
	}

	calls := fx.Calls()
	if len(calls) < 3 || !strings.HasSuffix(calls[1].path, "/organization/org-b/projects/proj-b2/api_keys") {
		t.Fatalf("list path = %+v, want the marker org-b/proj-b2 pair (default-marker preference)", calls)
	}

	if !strings.HasSuffix(calls[2].path, "/api_keys/key-7/access_tokens") {
		t.Fatalf("access path = %q, want the marker pair's key", calls[2].path)
	}

	if pat.APIKeyID != fixtureKeyID {
		t.Fatalf("Pat.APIKeyID = %q, want the marker pair's candidate", pat.APIKeyID)
	}
}

// TestPatMinterCreateIfAbsent locks D-07: an account with no zcode-api-key
// creates it through the official API (byte-exact body, usageScene 1), uses
// the created key, and a cache-missed re-mint lists FIRST and reuses the
// existing entry (check-then-create idempotency — never a duplicate).
func TestPatMinterCreateIfAbsent(t *testing.T) {
	t.Run("empty list creates the key with the byte-exact body", patRunCreateAbsent)
	t.Run("created entry is used as the keyId and a cache-missed re-mint lists first (idempotent)",
		patRunCreateIdempotent)
}

// patRunCreateAbsent runs the D-07 create vector: an empty candidate list
// answers exactly one POST with the byte-exact create body, Bearer form.
func patRunCreateAbsent(t *testing.T) {
	fx := newPatFixtureServer(t)
	clk := newPatClock(fixtureBaseTime())
	fx.SetListKeys(patReplyStatusBody(http.StatusOK, patKeyListEnvelope()))
	fx.SetCreateKey(patReplyStatusBody(http.StatusOK,
		`{"code":200,"data":{"name":"zcode-api-key","apiKey":"key-created"}}`))
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	_, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure = error %v, want the create-if-absent mint (D-07)", err)
	}

	if n := fx.patCallCount("create"); n != 1 {
		t.Fatalf("create POSTs = %d, want exactly 1", n)
	}

	create, ok := fx.CreateCall()
	if !ok {
		t.Fatal("create call missing")
	}

	const wantCreateBody = `{"name":"zcode-api-key","usageScene":1}`
	if create.body != wantCreateBody {
		t.Fatalf("create body = %q, want the byte-exact %s", create.body, wantCreateBody)
	}

	if create.auth != "Bearer "+fixtureLoginToken {
		t.Fatalf("create Authorization = %q, want Bearer %s", create.auth, fixtureLoginToken)
	}
}

// patRunCreateIdempotent runs the D-07 idempotency vector: the created entry
// becomes the keyId, and a cache-missed re-mint (a fresh minter against the
// same account) lists FIRST and reuses the existing entry — never a duplicate.
func patRunCreateIdempotent(t *testing.T) {
	fx := newPatFixtureServer(t)
	clk := newPatClock(fixtureBaseTime())
	fx.SetListKeys(patReplyStatusBody(http.StatusOK, patKeyListEnvelope()))
	fx.SetCreateKey(patReplyStatusBody(http.StatusOK,
		`{"code":200,"data":{"name":"zcode-api-key","apiKey":"key-created"}}`))
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	pat, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure = error %v, want the create-if-absent mint", err)
	}

	if pat.APIKeyID != "key-created" {
		t.Fatalf("Pat.APIKeyID = %q, want the created entry's id", pat.APIKeyID)
	}

	// A cache-missed re-mint (a fresh minter against the same account)
	// lists FIRST and finds the existing entry — no second create.
	fx.SetListKeys(patReplyStatusBody(http.StatusOK,
		patKeyListEnvelope(patKeyEntry("zcode-api-key", "key-created"))))
	m2 := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	pat2, err := m2.Ensure(context.Background())
	if err != nil {
		t.Fatalf("second Ensure = error %v, want the reuse mint", err)
	}

	if pat2.APIKeyID != "key-created" {
		t.Fatalf("second Pat.APIKeyID = %q, want the existing entry reused", pat2.APIKeyID)
	}

	if n := fx.patCallCount("create"); n != 1 {
		t.Fatalf("create POSTs = %d after the cache-missed re-mint, want 1 (check-then-create idempotency)", n)
	}

	if n := fx.patCallCount("list"); n != 2 {
		t.Fatalf("list calls = %d, want 2 (the candidate search precedes every mint)", n)
	}
}

// TestPatMinterDisabled locks the disabled arm: a reply carrying
// enable === false answers the coded disabled error, never a fabricated
// success, and the cache stays empty.
func TestPatMinterDisabled(t *testing.T) {
	fx := newPatFixtureServer(t)
	clk := newPatClock(fixtureBaseTime())
	fx.SetAccess(patReplyStatusBody(http.StatusOK, patAccessEnvelope(`{"enable":false}`)))
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	_, err := m.Ensure(context.Background())
	if !errors.Is(err, hostsvc.ErrPatDisabled) {
		t.Fatalf("Ensure = error %v, want ErrPatDisabled", err)
	}

	if _, ok := m.Cached(context.Background()); ok {
		t.Fatal("cache populated on the disabled arm")
	}
}

// TestPatMinterBizFail locks the business-envelope failure class: non-success
// codes (numeric and string forms) answer the coded invalid-response error,
// the string success forms still mint, and no response body ever leaks into
// error text.
func TestPatMinterBizFail(t *testing.T) {
	t.Run("numeric non-success code answers a coded error without leaking the body", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		body := `{"code":1001,"data":{"secret-marker-zc-synthetic-leak-probe":true}}`
		fx.SetAccess(patReplyStatusBody(http.StatusOK, body))
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

		_, err := m.Ensure(context.Background())
		if !errors.Is(err, hostsvc.ErrPatInvalidResponse) {
			t.Fatalf("Ensure = error %v, want ErrPatInvalidResponse for the non-success envelope", err)
		}

		if strings.Contains(err.Error(), "secret-marker") || strings.Contains(err.Error(), "1001") {
			t.Fatalf("error text %q leaks response material (bodies never appear in errors)", err)
		}
	})

	t.Run("string non-success code answers the same coded class", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		fx.SetAccess(patReplyStatusBody(http.StatusOK,
			`{"code":"500","data":{"accessToken":"x","tokenType":"Bearer","expiresIn":1,"expiresAt":1}}`))
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

		_, err := m.Ensure(context.Background())
		if !errors.Is(err, hostsvc.ErrPatInvalidResponse) {
			t.Fatalf("Ensure = error %v, want ErrPatInvalidResponse for the string code form", err)
		}
	})

	t.Run("string success codes still mint (the success set is numeric AND string)", func(t *testing.T) {
		fx := newPatFixtureServer(t)
		clk := newPatClock(fixtureBaseTime())
		fx.SetAccess(patReplyStatusBody(http.StatusOK,
			fmt.Sprintf(`{"code":"200","data":%s}`, patAccessData(fixturePATToken))))
		m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

		pat, err := m.Ensure(context.Background())
		if err != nil {
			t.Fatalf("Ensure = error %v, want a mint on the string success form", err)
		}

		if pat.Token != fixturePATToken {
			t.Fatalf("Pat.Token = %q, want the minted token", pat.Token)
		}
	})
}

// TestPatMinterKey404 locks Pitfall 8's discrimination: ONLY an access_tokens
// 404 (HTTP or business code) drops the cached key location so the next mint
// re-resolves via getCustomerInfo and re-lists; a permission (403) error and
// a network error leave the cached location intact — and no failure ever
// poisons the cache with a partial entry.
func TestPatMinterKey404(t *testing.T) {
	t.Run("404 drops the location and the next mint re-resolves and re-lists", patRunKey404Drop)
	t.Run("403 permission error leaves the cached location intact", patRunKey403Intact)
	t.Run("network error leaves the cached location intact", patRunNetIntact)
}

// patRunKey404Drop runs the 404 vector: access_tokens 404 answers the coded
// key-missing error, drops the cached token AND location, and the next mint
// re-resolves via getCustomerInfo and re-lists against the new key.
func patRunKey404Drop(t *testing.T) {
	fx := newPatFixtureServer(t)
	clk := newPatClock(fixtureBaseTime())
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	if _, err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("first Ensure = error %v, want a mint", err)
	}

	if _, ok := m.Cached(context.Background()); !ok {
		t.Fatal("first mint did not populate the cache")
	}

	// Past the refresh lead: the next Ensure re-mints, and the key has
	// moved (404 at access_tokens).
	clk.Advance(59 * time.Minute)
	fx.SetAccess(patReplyStatusBody(http.StatusNotFound, `{"code":404,"data":null}`))

	_, err := m.Ensure(context.Background())
	if !errors.Is(err, hostsvc.ErrPatKeyMissing) {
		t.Fatalf("Ensure = error %v, want ErrPatKeyMissing (404 is the ONLY location-invalidating class)", err)
	}

	if _, ok := m.Cached(context.Background()); ok {
		t.Fatal("cache kept the token after the 404 — the dead key location survived")
	}

	// The key re-appeared under a new id: the next mint re-resolves via
	// getCustomerInfo, re-lists, and succeeds against the new key.
	fx.SetListKeys(patReplyStatusBody(http.StatusOK,
		patKeyListEnvelope(patKeyEntry("zcode-api-key", "key-8"))))
	fx.SetAccess(patReplyStatusBody(http.StatusOK, patAccessEnvelope(patAccessData(patSecondToken))))

	pat, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("recovery Ensure = error %v, want a mint against the re-resolved location", err)
	}

	patAssertKey404Counts(t, fx, 2)

	calls := fx.Calls()
	var access404, accessOK *patFixtureCall

	for i := range calls {
		if !strings.HasSuffix(calls[i].path, "/access_tokens") {
			continue
		}

		switch {
		case strings.HasSuffix(calls[i].path, "/api_keys/key-7/access_tokens"):
			access404 = &calls[i]
		case strings.HasSuffix(calls[i].path, "/api_keys/key-8/access_tokens"):
			accessOK = &calls[i]
		}
	}

	if access404 == nil || accessOK == nil {
		t.Fatalf("access calls = %+v, want key-7 (404) then key-8 (success)", calls)
	}

	if pat.APIKeyID != "key-8" {
		t.Fatalf("Pat.APIKeyID = %q, want the re-resolved key-8", pat.APIKeyID)
	}
}

// patAssertKey404Counts locks the re-resolution counts: the customer probe
// and the key list each ran once per RESOLUTION (the 404 re-resolved, the
// 403/network recoveries did not).
func patAssertKey404Counts(t *testing.T, fx *patFixtureServer, wantResolutions int) {
	t.Helper()

	if n := fx.patCallCount("customer"); n != wantResolutions {
		t.Fatalf("customer calls = %d, want %d", n, wantResolutions)
	}

	if n := fx.patCallCount("list"); n != wantResolutions {
		t.Fatalf("list calls = %d, want %d", n, wantResolutions)
	}
}

// patRunKey403Intact runs the permission vector: an access_tokens 403
// answers the coded http-status class, drops the stale TOKEN, but the
// cached LOCATION survives — the recovery mint goes straight back to the
// same key with no re-resolution and no re-list.
func patRunKey403Intact(t *testing.T) {
	fx := newPatFixtureServer(t)
	clk := newPatClock(fixtureBaseTime())
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	if _, err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("first Ensure = error %v, want a mint", err)
	}

	clk.Advance(59 * time.Minute)
	fx.SetAccess(patReplyStatusBody(http.StatusForbidden, `{"code":403,"data":null}`))

	_, err := m.Ensure(context.Background())
	if err == nil || !strings.Contains(err.Error(), "project_token_http_403") {
		t.Fatalf("Ensure = error %v, want the coded project_token_http_403 class", err)
	}

	if _, ok := m.Cached(context.Background()); ok {
		t.Fatal("cache kept the token after a non-transient failure (never a partial serve)")
	}

	// Recovery: the permission problem resolves. The location was NEVER
	// dropped — the mint goes straight back to access_tokens with the
	// SAME key, no re-resolution, no re-list.
	fx.SetAccess(patReplyStatusBody(http.StatusOK, patAccessEnvelope(patAccessData(patSecondToken))))

	pat, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("recovery Ensure = error %v, want a mint against the intact location", err)
	}

	patAssertKey404Counts(t, fx, 1)

	if pat.APIKeyID != fixtureKeyID {
		t.Fatalf("Pat.APIKeyID = %q, want the SAME key-7 (no re-create)", pat.APIKeyID)
	}
}

// patRunNetIntact runs the network vector: a transport-level failure past
// token expiry answers the honest coded network error, and the cached
// location survives — the recovery mint goes straight back to access_tokens.
func patRunNetIntact(t *testing.T) {
	fx := newPatFixtureServer(t)
	clk := newPatClock(fixtureBaseTime())
	m := newFixtureMinter(t, fx.srv.URL, &fixturePatCreds{login: fixtureLoginToken, loginOK: true}, clk.Now)

	if _, err := m.Ensure(context.Background()); err != nil {
		t.Fatalf("first Ensure = error %v, want a mint", err)
	}

	// Past expiry: no still-valid token can serve, so the network failure
	// surfaces as the honest coded error.
	clk.Advance(2 * time.Hour)
	fx.SetAccess(patReplyStatusBody(http.StatusOK,
		patAccessEnvelope(patAccessDataFrom(clk.Now(), patSecondToken))))
	fx.SetNetFail(true)

	_, err := m.Ensure(context.Background())
	if !errors.Is(err, hostsvc.ErrPatNetworkError) {
		t.Fatalf("Ensure = error %v, want ErrPatNetworkError", err)
	}

	fx.SetNetFail(false)

	pat, err := m.Ensure(context.Background())
	if err != nil {
		t.Fatalf("recovery Ensure = error %v, want a mint against the intact location", err)
	}

	patAssertKey404Counts(t, fx, 1)

	if pat.Token != patSecondToken {
		t.Fatalf("Pat.Token = %q, want the recovery mint", pat.Token)
	}
}
