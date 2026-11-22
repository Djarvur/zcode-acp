// Package hostsvc file patminter.go — the PAT mint execution tier (AUTH-02,
// plan 06-02): the OFFICIAL api.z.ai chain (getCustomerInfo -> personal pick
// -> api_keys list -> create-if-absent -> access_tokens) that trades the
// desktop's OAuth login for a short-lived project access token, with the
// in-process-only cache (D-04), single-flight mints, and the reactive
// refresh surface (D-05) the runtime-headers source (06-07) consumes.
//
// Port attribution (Apache-2.0): the chain, cache semantics, and the coded
// project_token-class error family are ported from william0wang/zcode-acp
// v0.89.0 src/config/project-token.ts (Apache-2.0; no NOTICE file —
// attribution per docs/research/reference-adapters.md), with the desktop
// client's wire shapes (zcode-src 3.17.x
// packages/shared/src/project-access-token-client.ts). D-07 (create-if-absent)
// is the explicit user decision superseding the reference's read-only posture.
//
// Wire truth (06-RESEARCH.md Pattern 3, byte-verbatim): getCustomerInfo takes
// the RAW login token in Authorization (NO Bearer — a Bearer here is misjudged
// as login-invalid); the api_keys list/create and access_tokens calls take
// "Bearer " + login token. Business envelope {code, data} with success codes
// {200, 0, "200", "0"}. HTTP 404 / business code 404 on access_tokens drops
// ONLY the cached key location (Pitfall 8: permission/network errors never
// invalidate). accountScope = sha256 over the compact JSON array [family,
// plan kind, null, loginToken, zcodeJwtOrNil].
//
// Hygiene (D-04, T-5-17 extension, test-locked by TestPatMemoryOnly): the PAT,
// the login token, and every other credential class live in memory only —
// nothing in this file writes to disk, no constructor accepts a path, and the
// Pat struct carries no String()/GoString()/Marshal rendering. Log statements
// carry codes, ids, and lengths — never a value, never a response body.
package hostsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// The mint chain's pinned constants (06-RESEARCH.md Pattern 3; the literals
// are the reference's own — never re-derived).
const (
	// patFamily is the coding-plan family this minter serves (the official
	// api.z.ai path; bigmodel.cn parity is out of this phase's scope).
	patFamily = "zai"

	// patDefaultOrigin is the official business-API origin for zai. Tests
	// inject an httptest origin through WithPatOrigin — production never
	// overrides it.
	patDefaultOrigin = "https://api.z.ai"

	// patKeyName is the account key the desktop provisions per account and
	// the chain selects (create-if-absent per D-07).
	patKeyName = "zcode-api-key"

	// patDefaultOrgMarker and patDefaultProjMarker are the desktop's
	// server-side default-org/default-project name markers (D-06 pick).
	patDefaultOrgMarker  = "默认机构"
	patDefaultProjMarker = "默认项目"

	// patTeamProjectType is the projectType string marking team projects,
	// which are never the personal plan's home (D-06 filter).
	patTeamProjectType = "2"

	// patPlanKind is the plan kind slot of the accountScope array.
	patPlanKind = "individual-coding-plan"

	// patRequestTimeout bounds every mint HTTPS call (ref REQUEST_TIMEOUT_MS),
	// set once on the client at construction (Pitfall 7: bounded I/O, never a
	// hang).
	patRequestTimeout = 15 * time.Second

	// patRefreshLead refreshes this long before expiry so an in-flight turn
	// never carries a dead token (ref REFRESH_LEAD_MS).
	patRefreshLead = 120 * time.Second

	// patTransientBackoff defers the refresh retry after a transient failure
	// while the cached token is still valid (ref TRANSIENT_RETRY_BACKOFF_MS,
	// D-05 reactive posture).
	patTransientBackoff = 5 * time.Second

	// patMaxBodyBytes bounds the response-body read — a hostile or broken
	// backend cannot balloon the minter's memory (the body itself is never
	// logged).
	patMaxBodyBytes = 1 << 20

	// The success-envelope code set {200, 0, "200", "0"} (ref
	// projectTokenSuccessCodeSchema), numeric and string forms.
	patCodeOK      = 200
	patCodeZero    = 0
	patCodeOKStr   = "200"
	patCodeZeroStr = "0"

	// patCodeNotFound is the business/HTTP code that — and ONLY on
	// access_tokens — drops the cached key location (Pitfall 8).
	patCodeNotFound = 404

	// patHTTPMultiMin/patHTTPMultiMax delimit the 2xx transport band the
	// envelope decode accepts.
	patHTTPMultiMin = 200
	patHTTPMultiMax = 299

	// patBearerPrefix is the Bearer form the list/create and access_tokens
	// calls carry (zai form); getCustomerInfo carries the RAW token.
	patBearerPrefix = "Bearer "

	// patDefaultClientVersion is the desktop-version fallback (ref
	// installedAppVersion's "0.0.0"); production wiring injects the resolved
	// desktop install's version via WithPatClientVersion.
	patDefaultClientVersion = "0.0.0"
)

// The coded stable error family (the reference's project_token-class codes;
// consumers match the sentinels, never the message — the package's
// coded-decline discipline). Every mint failure answers one of these, never a
// raw transport noise and never a response body.
var (
	// ErrPatLoginRequired: the credential source holds no desktop login — the
	// coded arm names the desktop login (the user's fix).
	ErrPatLoginRequired = errors.New("project_token_login_required " +
		"(log in via the ZCode desktop first)")

	// ErrPatProjectUnavailable: no personal organization/project survives the
	// D-06 pick (all-team account or empty org list).
	ErrPatProjectUnavailable = errors.New("project_token_project_unavailable")

	// ErrPatKeyMissing: the selected key no longer exists at its cached
	// location (access_tokens 404) — the location is dropped and the next
	// mint re-resolves (Pitfall 8's ONLY invalidating error class).
	ErrPatKeyMissing = errors.New("project_token_key_missing")

	// ErrPatDisabled: the access_tokens reply carries enable === false — the
	// key is switched off account-side.
	ErrPatDisabled = errors.New("project_token_disabled")

	// ErrPatInvalidResponse: an envelope/reply failed its shape validation
	// (non-success code, wrong types, dead expiry) — never a partial serve.
	ErrPatInvalidResponse = errors.New("project_token_invalid_response")

	// ErrPatScopeInvalidated: the sign-in changed relative to the credential
	// context — a stale token is never served (the D-05/T-06-06 posture; the
	// expectedAccountScope check itself lives on the 06-07 source adapter).
	ErrPatScopeInvalidated = errors.New("project_token_scope_invalidated")

	// ErrPatNetworkError: a transport-level failure (connection refused/reset,
	// unreachable host) — transient per the reference's classification.
	ErrPatNetworkError = errors.New("project_token_network_error")

	// ErrPatTimeout: the per-call bound fired — transient per the reference's
	// classification.
	ErrPatTimeout = errors.New("project_token_timeout")

	// ErrPatRequestFailed: the uncategorized fallback code (the reference's
	// project_token_request_failed) — e.g. a caller's ctx dying mid-mint.
	ErrPatRequestFailed = errors.New("project_token_request_failed")
)

// patHTTPStatusError is the non-2xx transport status the envelope decode
// refuses (the reference's project_token_http_<status> family). The status is
// not a credential; the body is never carried.
type patHTTPStatusError struct {
	status int
}

// Error renders the coded family form project_token_http_<status>.
func (e *patHTTPStatusError) Error() string {
	return fmt.Sprintf("project_token_http_%d", e.status)
}

// Pat is one minted project access token and its cache identity. The fields
// are exactly the plan's cache spec. Deliberately NO String()/GoString()/
// Marshal* rendering (D-04/T-5-17 hygiene, test-locked): the token and its
// material must never reach a log or an error through accidental formatting.
type Pat struct {
	// Token is the minted project access token (the requestAuth.apiKey).
	Token string

	// APIKeyID is the account key id the token was issued against (the
	// requestAuth.apiKeyId).
	APIKeyID string

	// Fingerprint is sha256 hex of Token — the form the backend's
	// rejectedProjectTokenFingerprint speaks (D-05 re-mint trigger).
	Fingerprint string

	// AccountScope is the opaque sign-in identity the scope check consumes
	// (06-07's expectedAccountScope comparison; T-06-06 mitigation).
	AccountScope string

	// ExpiresAt is the conservative expiry: min(expiresAtSec*1000,
	// startedAt+expiresIn*1000) — network time never extends validity.
	ExpiresAt time.Time

	// RefreshAt is max(startedAt, ExpiresAt-120s): a cached token is served
	// only while now < RefreshAt; past it the next Ensure refreshes (the
	// proactive arm of D-05; the reactive fingerprint arm lives on 06-07).
	RefreshAt time.Time
}

// PatCredentialSource is the credential seam (06-01's locked accessor
// surface): the login token arrives Bearer-stripped (the CredentialStore
// contract) and the ZCode JWT folds into accountScope as null when absent.
// Tests inject fixtures; production injects *interact.CredentialStore.
type PatCredentialSource interface {
	// LoginToken returns the family's oauth access token with the Bearer
	// prefix already stripped; ok=false when the desktop holds no login.
	LoginToken(ctx context.Context, family string) (string, bool)

	// ZcodeJWT returns the zcodejwttoken entry; ok=false when absent.
	ZcodeJWT(ctx context.Context) (string, bool)
}

// PatOption configures a PatMinter at construction.
type PatOption func(*PatMinter)

// WithPatOrigin overrides the business-API origin (httptest vectors only —
// production keeps the official https://api.z.ai default; T-06-05: the origin
// is injectable ONLY for tests).
func WithPatOrigin(origin string) PatOption {
	return func(m *PatMinter) { m.origin = origin }
}

// WithPatHTTPClient overrides the HTTP client (tests inject transport
// variants; the default is a fresh client over a cloned http.DefaultTransport
// with the effective proxy resolved and logged once at construction and the
// 15s per-call Timeout — Pitfall 7).
func WithPatHTTPClient(client *http.Client) PatOption {
	return func(m *PatMinter) { m.client = client }
}

// WithPatClock overrides the wall clock (tests pin/advance time for the
// expiry/refresh/backoff math; the default is time.Now).
func WithPatClock(clock func() time.Time) PatOption {
	return func(m *PatMinter) { m.clock = clock }
}

// WithPatClientVersion overrides the clientVersion sent with the exchange
// (the reference sends the DESKTOP's version with a "0.0.0" fallback;
// production injects the resolved desktop install's version).
func WithPatClientVersion(version func() string) PatOption {
	return func(m *PatMinter) { m.version = version }
}

// WithPatCredentials overrides the credential source (the 06-01 store seam;
// tests inject fixtures).
func WithPatCredentials(creds PatCredentialSource) PatOption {
	return func(m *PatMinter) { m.creds = creds }
}

// patSession is one login's in-memory state (the reference's LoginSession):
// the login fingerprint guards the session so a desktop re-login drops every
// cached artifact (scope change), and the single-flight fields make
// concurrent Ensure calls share one in-flight mint.
type patSession struct {
	// loginFP is sha256 hex of the stripped login token — a changed login
	// means a new session (location + token dropped, scope changed).
	loginFP string

	// loc is the cached key location {organizationId, projectId, apiKeyId}.
	// Only an access_tokens 404 drops it (Pitfall 8); permission/network
	// errors never invalidate.
	loc *patKeyLocation

	// token is the cached Pat (nil = none).
	token *Pat

	// minting is non-nil while one mint is in flight; waiters join it
	// (single-flight).
	minting chan struct{}

	// mintID/doneID pair with lastPat/lastErr so joiners read exactly the
	// outcome of the mint they joined.
	mintID  uint64
	doneID  uint64
	lastPat *Pat
	lastErr error
}

// patKeyLocation is where the account key lives — an org/project/keyId triple
// resolved by the chain's first half (getCustomerInfo -> pick -> list/create).
type patKeyLocation struct {
	organizationID string
	projectID      string
	apiKeyID       string
}

// PatMinter is the AUTH-02 execution tier service. Constructed once via
// NewPatMinter; Ensure/Cached/Invalidate are safe for concurrent use.
type PatMinter struct {
	origin  string
	client  *http.Client
	clock   func() time.Time
	version func() string
	creds   PatCredentialSource

	// mu guards the session (cache + single-flight state).
	mu   sync.Mutex
	sess *patSession
}

// NewPatMinter returns the minter with options applied. The default HTTP
// client clones http.DefaultTransport (the default Happy-Eyeballs dialer —
// never a custom IPv6-first dialer, Pitfall 7), resolves the effective proxy
// for the origin ONCE at construction and logs it once (a dead proxy is
// surfaced at startup, never discovered per call), and carries the 15s
// per-call Timeout.
func NewPatMinter(opts ...PatOption) *PatMinter {
	m := &PatMinter{
		origin:  patDefaultOrigin,
		clock:   time.Now,
		version: func() string { return patDefaultClientVersion },
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.client == nil {
		m.client = newPatHTTPClient(m.origin)
	}

	return m
}

// newPatHTTPClient builds the default client: cloned default transport, the
// effective proxy for origin resolved once and logged once (Pitfall 7).
func newPatHTTPClient(origin string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	target := &url.URL{Scheme: "https", Host: origin}
	if parsed, perr := url.Parse(origin); perr == nil && parsed.Host != "" {
		target = &url.URL{Scheme: parsed.Scheme, Host: parsed.Host}
	}

	proxyURL, proxyErr := http.ProxyFromEnvironment(&http.Request{URL: target})
	transport.Proxy = func(*http.Request) (*url.URL, error) { return proxyURL, proxyErr }

	proxyHost := ""
	if proxyURL != nil {
		proxyHost = proxyURL.Host
	}

	slog.Info("hostsvc: pat minter transport ready",
		"origin_host", target.Host,
		"proxy_configured", proxyURL != nil,
		"proxy_host", proxyHost,
		"timeout", patRequestTimeout.String())

	return &http.Client{Transport: transport, Timeout: patRequestTimeout}
}

// Ensure returns a current PAT for the source's login, minting through the
// official chain when the cache is stale or absent. Concurrent callers share
// one in-flight mint (single-flight). A transient failure with a still-valid
// cached token serves the cached token and defers the retry (D-05); every
// other failure is a coded stable error and never poisons the cache.
func (m *PatMinter) Ensure(_ context.Context) (Pat, error) {
	return Pat{}, nil // skeleton: zero-value body (plan 06-02, compile-first RED)
}

// Cached returns the in-process cached PAT without any network I/O; ok=false
// when no token is cached. The 06-07 source adapter uses it to serve
// runtime-headers asks without forcing a mint.
func (m *PatMinter) Cached(_ context.Context) (Pat, bool) {
	return Pat{}, false // skeleton: zero-value body
}

// Invalidate drops the cached token (never the key location) so the next
// Ensure re-mints — the D-05 reactive re-mint trigger surface the 06-07
// source adapter drives on a rejectedProjectTokenFingerprint match.
func (m *PatMinter) Invalidate(_ string) {}
