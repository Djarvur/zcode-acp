// File patminter.go — the PAT mint execution tier (AUTH-02,
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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	// server-side default-org/default-project name literals (the D-06 pick
	// preference; the escapes keep the source ASCII, the VALUES are the
	// exact server literals).
	patDefaultOrgMarker  = "\u9ed8\u8ba4\u673a\u6784" // 默认机构
	patDefaultProjMarker = "\u9ed8\u8ba4\u9879\u76ee" // 默认项目

	// patTeamProjectType is the projectType wire value marking team projects,
	// which are never the personal plan's home (the D-06 filter;
	// String(projectType).trim() compared against it).
	patTeamProjectType = "2"

	// patPlanKind is the plan kind slot of the accountScope array.
	patPlanKind = "individual-coding-plan"

	// patCreateKeyBody is the byte-exact create-if-absent body (D-07): the
	// desktop's own create — name + the coding-plan usage scene (1 =
	// API_KEY_USAGE_SCENE.CODING_PLAN). A constant string, never a marshal
	// (key order is part of the wire shape).
	patCreateKeyBody = `{"name":"zcode-api-key","usageScene":1}`

	// The HTTP status markers of the transient classification (the
	// reference's isTransient: 429 and the 5xx band retry; 4xx does not).
	patHTTPTooManyRequests = 429
	patHTTPServerErrMin    = 500

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

	// patSecToMilli converts reply-second values into milliseconds (the
	// reference's *SECOND_MS expiry math).
	patSecToMilli = 1000.0

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

	// patTokenType is the access_tokens reply's required tokenType.
	patTokenType = "Bearer"

	// patContentType is the JSON content type every mint call carries.
	patContentType = "application/json"

	// patDefaultClientVersion is the desktop-version fallback (ref
	// installedAppVersion's "0.0.0"); production wiring injects the resolved
	// desktop install's version via WithPatClientVersion.
	patDefaultClientVersion = "0.0.0"

	// The mint-chain URL shapes (the official api.z.ai templates; org and
	// project ids are path-escaped).
	patCustomerInfoPath = "/api/biz/customer/getCustomerInfo"
	patKeyListURLFormat = "/api/biz/v1/organization/%s/projects/%s/api_keys"
	patAccessSuffix     = "/access_tokens"
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
// the login fingerprint guards the session so a desktop re-login drops the
// cached token (scope change — a stale token is never served under a new
// sign-in).
type patSession struct {
	// loginFP is sha256 hex of the stripped login token — a changed login
	// means a new session (cached artifacts dropped, scope changed).
	loginFP string

	// loc is the cached key location {organizationId, projectId, apiKeyId};
	// ONLY an access_tokens 404 drops it (Pitfall 8 — permission and network
	// errors never invalidate).
	loc *patKeyLocation

	// token is the cached Pat (nil = none).
	token *Pat

	// minting is non-nil while one mint is in flight; concurrent Ensure
	// callers join it (single-flight — exactly one HTTPS mint per burst).
	minting chan struct{}

	// mintID/doneID pair with lastPat/lastErr so joiners read exactly the
	// outcome of the mint they joined (a newer mint may have started by the
	// time a joiner wakes).
	mintID  uint64
	doneID  uint64
	lastPat *Pat
	lastErr error
}

// patKeyLocation is where the account key lives — an org/project/keyId triple
// resolved by the chain's first half (getCustomerInfo -> pick -> list).
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

	// mu guards the session (cache state).
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
	base, baseOK := http.DefaultTransport.(*http.Transport)
	if !baseOK {
		// Unreachable (http.DefaultTransport is exactly *http.Transport);
		// the honest fallback keeps the two default behaviors that matter
		// here — env proxy + HTTP/2 — without a custom dialer (Pitfall 7).
		base = &http.Transport{Proxy: http.ProxyFromEnvironment, ForceAttemptHTTP2: true}
	}

	transport := base.Clone()

	target := patOriginTarget(origin)
	proxyURL, proxyErr := http.ProxyFromEnvironment(&http.Request{URL: target})
	transport.Proxy = func(*http.Request) (*url.URL, error) {
		if proxyErr != nil {
			return nil, fmt.Errorf("pat minter proxy: %w", proxyErr)
		}

		return proxyURL, nil
	}

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

// patOriginTarget renders the proxy-resolution target for origin (the
// scheme+host pair; an unparseable origin falls back to https + the raw
// string — the constructor is the only caller and the default origin is a
// constant).
func patOriginTarget(origin string) *url.URL {
	if parsed, err := url.Parse(origin); err == nil && parsed.Host != "" {
		return &url.URL{Scheme: parsed.Scheme, Host: parsed.Host}
	}

	return &url.URL{Scheme: "https", Host: origin}
}

// Ensure returns a current PAT for the source's login, minting through the
// official chain when the cache is stale or absent. A cached token serves
// while its RefreshAt is in the future (zero HTTP); past it the next Ensure
// refreshes (the proactive D-05 arm — the reactive fingerprint arm lives on
// the 06-07 source adapter over Invalidate). Concurrent callers share one
// in-flight mint (single-flight). A transient failure with a still-valid
// cached token serves the cached token and defers the retry past the 5s
// floor; every other failure is a coded stable error and never leaves a
// partially-poisoned cache.
func (m *PatMinter) Ensure(ctx context.Context) (Pat, error) {
	if m.creds == nil {
		// No credential source configured — the desktop-login arm is the
		// honest posture (an unconfigured minter is simply not logged in).
		return Pat{}, ErrPatLoginRequired
	}

	raw, ok := m.creds.LoginToken(ctx, patFamily)
	login := strings.TrimSpace(raw)
	if !ok || login == "" {
		// The absent-login arm: the coded error naming the desktop login,
		// BEFORE any network I/O (zero HTTP calls, no fabricated success).
		return Pat{}, ErrPatLoginRequired
	}

	// The session (per-login cache scope) is established BEFORE the mint —
	// the reference's LoginSession guard: a changed login drops every cached
	// artifact up front, and the in-flight chain can store what it resolves.
	fp := patSHA256Hex(login)

	for {
		m.mu.Lock()
		if m.sess == nil || m.sess.loginFP != fp {
			m.sess = &patSession{loginFP: fp}
		}

		s := m.sess

		if s.minting == nil {
			now := m.clock()
			if s.token != nil && now.Before(s.token.RefreshAt) {
				// Fast path: the cached token serves until its refresh
				// point — zero HTTP (D-05).
				pat := *s.token
				m.mu.Unlock()

				return pat, nil
			}

			// Become the single-flight leader.
			done := make(chan struct{})
			s.minting = done
			s.mintID++
			id := s.mintID
			m.mu.Unlock()

			return m.mintAsLeader(ctx, login, s, id, done)
		}

		// Join the in-flight mint (single-flight: N callers, one mint).
		done := s.minting
		id := s.mintID
		m.mu.Unlock()

		select {
		case <-done:
		case <-ctx.Done():
			return Pat{}, fmt.Errorf("%w: %w", ErrPatRequestFailed, ctx.Err())
		}

		m.mu.Lock()
		joined := s.doneID == id
		joinPat, joinErr := patJoinResult(s.lastPat, s.lastErr)
		m.mu.Unlock()

		if joined {
			return joinPat, joinErr
		}
		// The joined slot moved on (a newer mint started) — retry the loop.
	}
}

// Cached returns the in-process cached PAT without any network I/O; ok=false
// when no token is cached. The 06-07 source adapter uses it to serve
// runtime-headers asks without forcing a mint.
func (m *PatMinter) Cached(_ context.Context) (Pat, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sess == nil || m.sess.token == nil {
		return Pat{}, false
	}

	return *m.sess.token, true
}

// Invalidate drops the cached token (never the key location) so the next
// Ensure re-mints — the D-05 reactive re-mint trigger surface the 06-07
// source adapter drives on a rejectedProjectTokenFingerprint match.
func (m *PatMinter) Invalidate(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sess == nil || m.sess.token == nil {
		return
	}

	m.sess.token = nil

	slog.Info("hostsvc: pat cache invalidated", "reason", reason)
}

// patJoinResult copies the joined mint's outcome.
func patJoinResult(lastPat *Pat, lastErr error) (Pat, error) {
	if lastErr != nil || lastPat == nil {
		return Pat{}, lastErr
	}

	return *lastPat, nil
}

// mintAsLeader runs the chain as the burst's single minter, publishes the
// outcome for joiners, and advances the cache. The transient serve arm
// (still-valid cached token + the 5s retry floor) and the honest-failure
// cache clearing both land here so the leader and every joiner see ONE
// consistent outcome (the reference's shared-pending semantics).
func (m *PatMinter) mintAsLeader(
	ctx context.Context,
	login string,
	s *patSession,
	id uint64,
	done chan struct{},
) (Pat, error) {
	pat, merr := m.mint(ctx, login)

	m.mu.Lock()
	s.minting = nil
	s.doneID = id

	result, resultErr := pat, merr

	switch {
	case merr == nil:
		stored := pat
		s.lastPat = &stored
		s.lastErr = nil

		if m.sess == s {
			s.token = &stored
		}

	case isTransientMintError(merr) && s.token != nil && m.clock().Before(s.token.ExpiresAt):
		// Keep serving the still-valid token; the retry defers past the 5s
		// floor (the reference's onRefreshDeferred posture, D-05).
		served := *s.token
		floor := m.clock().Add(patTransientBackoff)
		if served.ExpiresAt.Before(floor) {
			floor = served.ExpiresAt
		}

		served.RefreshAt = floor
		s.token = &served
		s.lastPat = &served
		s.lastErr = nil
		result, resultErr = served, nil

		slog.Warn("hostsvc: pat mint deferred", "code", "pat-refresh-deferred")

	default:
		// Honest failure: never a dead or partial token in the cache.
		s.lastPat = nil
		s.lastErr = merr

		if m.sess == s {
			s.token = nil
		}
	}

	close(done)
	m.mu.Unlock()

	return result, resultErr
}

// isTransientMintError classifies mint failures the reference's way: the
// timeout and network coded classes, plus the 429 and 5xx transport bands,
// are transient (retryable); every other failure (permission, disabled,
// business codes, shape errors) is not.
func isTransientMintError(err error) bool {
	if errors.Is(err, ErrPatTimeout) || errors.Is(err, ErrPatNetworkError) {
		return true
	}

	var status *patHTTPStatusError
	if errors.As(err, &status) {
		return status.status == patHTTPTooManyRequests || status.status >= patHTTPServerErrMin
	}

	return false
}

// mint runs the official chain once: the accountScope digest over the current
// sign-in, the key location (cached per session, or resolved through
// getCustomerInfo -> pick -> list/create), and the access_tokens exchange.
// An access_tokens 404 drops ONLY the cached location (Pitfall 8) so the
// next mint re-resolves.
func (m *PatMinter) mint(ctx context.Context, login string) (Pat, error) {
	jwt, _ := m.creds.ZcodeJWT(ctx)
	scope := patAccountScope(patFamily, login, jwt)

	loc := m.cachedLocation(login)
	if loc == nil {
		resolved, rerr := m.resolveLocation(ctx, login)
		if rerr != nil {
			return Pat{}, rerr
		}

		m.storeLocation(login, resolved)
		loc = resolved
	}

	pat, err := m.issueToken(ctx, login, loc, m.clock(), scope)
	if err != nil && errors.Is(err, ErrPatKeyMissing) {
		// The key moved (access_tokens 404): drop ONLY the cached location
		// — the next mint re-resolves via getCustomerInfo and re-lists.
		m.dropLocation(login)
	}

	return pat, err
}

// cachedLocation returns the session's cached key location (nil = none),
// scoped to the login the caller is minting for.
func (m *PatMinter) cachedLocation(login string) *patKeyLocation {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sess == nil || m.sess.loginFP != patSHA256Hex(login) {
		return nil
	}

	return m.sess.loc
}

// storeLocation caches the resolved key location under the current session.
func (m *PatMinter) storeLocation(login string, loc *patKeyLocation) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sess != nil && m.sess.loginFP == patSHA256Hex(login) {
		m.sess.loc = loc
	}
}

// dropLocation clears the cached key location (the 404 arm).
func (m *PatMinter) dropLocation(login string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sess != nil && m.sess.loginFP == patSHA256Hex(login) {
		m.sess.loc = nil
	}
}

// resolveLocation resolves where the account key lives: getCustomerInfo with
// the RAW token, the personal pick (D-06), then the api_keys list.
func (m *PatMinter) resolveLocation(ctx context.Context, login string) (*patKeyLocation, error) {
	orgID, projID, err := m.pickPersonal(ctx, login)
	if err != nil {
		return nil, err
	}

	keyID, err := m.locateOrCreateKey(ctx, login, orgID, projID)
	if err != nil {
		return nil, err
	}

	return &patKeyLocation{organizationID: orgID, projectID: projID, apiKeyID: keyID}, nil
}

// patPersonalOrg/patPersonalProject are the D-06 pick candidates: the
// personal layer after the team filter, ids non-empty.
type patPersonalOrg struct {
	id       string
	name     string
	projects []patPersonalProject
}

type patPersonalProject struct {
	id   string
	name string
}

// pickPersonal digs the customer-info envelope and picks the personal org and
// project (D-06): team projects (projectType "2") are never candidates, the
// default-marker org/project win, the deterministic first pair otherwise —
// and multiple candidates answer a slog warn listing the ids (the user is
// never blocked). No personal candidates at all answers the coded
// project-unavailable arm.
func (m *PatMinter) pickPersonal(ctx context.Context, login string) (orgID, projID string, err error) {
	obj, err := m.callJSON(ctx, http.MethodGet, m.origin+patCustomerInfoPath, "", login)
	if err != nil {
		return "", "", err
	}

	orgs, err := patCustomerOrganizations(obj)
	if err != nil {
		return "", "", err
	}

	personal := patCollectPersonal(orgs)
	picked, proj, ok := patPickPair(personal)
	if !ok {
		return "", "", ErrPatProjectUnavailable
	}

	patWarnMultiPick(personal, picked, proj)

	return picked.id, proj.id, nil
}

// patCustomerOrganizations digs the organizations array out of the decoded
// getCustomerInfo envelope (the desktop's customerSchema shape).
func patCustomerOrganizations(obj map[string]any) ([]any, error) {
	data, err := patEnvelopeData(obj)
	if err != nil {
		return nil, err
	}

	dataObj, dataOK := data.(map[string]any)
	if !dataOK {
		return nil, ErrPatInvalidResponse
	}

	orgs, ok := dataObj["organizations"].([]any)
	if !ok {
		return nil, ErrPatInvalidResponse
	}

	return orgs, nil
}

// patCollectPersonal filters the organizations array down to the personal
// candidates: non-empty org id, team projects removed, orgs without any
// remaining project skipped (the reference's personal layer build).
func patCollectPersonal(orgs []any) []patPersonalOrg {
	personal := make([]patPersonalOrg, 0, len(orgs))

	for _, rawOrg := range orgs {
		org, ok := rawOrg.(map[string]any)
		if !ok {
			continue
		}

		id := patNonEmpty(org["organizationId"])
		if id == "" {
			continue
		}

		rawProjects, _ := org["projects"].([]any)
		projects := patCollectProjects(rawProjects)
		if len(projects) == 0 {
			continue
		}

		personal = append(personal, patPersonalOrg{
			id:       id,
			name:     patNonEmpty(org["organizationName"]),
			projects: projects,
		})
	}

	return personal
}

// patCollectProjects filters one org's projects: team-typed entries removed,
// non-empty ids kept.
func patCollectProjects(rawProjects []any) []patPersonalProject {
	projects := make([]patPersonalProject, 0, len(rawProjects))

	for _, rawProj := range rawProjects {
		proj, ok := rawProj.(map[string]any)
		if !ok || patProjectTypeIsTeam(proj["projectType"]) {
			continue
		}

		id := patNonEmpty(proj["projectId"])
		if id == "" {
			continue
		}

		projects = append(projects, patPersonalProject{id: id, name: patNonEmpty(proj["projectName"])})
	}

	return projects
}

// patPickPair applies the D-06 preference: the org whose name carries the
// default-org marker else the deterministic first; within it the project
// carrying the default-project marker else the first.
func patPickPair(personal []patPersonalOrg) (patPersonalOrg, patPersonalProject, bool) {
	if len(personal) == 0 {
		return patPersonalOrg{}, patPersonalProject{}, false
	}

	orgIdx := 0
	for i, org := range personal {
		if strings.Contains(org.name, patDefaultOrgMarker) {
			orgIdx = i

			break
		}
	}

	picked := personal[orgIdx]

	projIdx := 0
	for i, proj := range picked.projects {
		if strings.Contains(proj.name, patDefaultProjMarker) {
			projIdx = i

			break
		}
	}

	return picked, picked.projects[projIdx], true
}

// patWarnMultiPick emits the D-06 structural warn listing the candidate ids
// when several personal orgs (or several projects of the picked org) exist —
// ids only, never names-with-material, never a block.
func patWarnMultiPick(orgs []patPersonalOrg, picked patPersonalOrg, proj patPersonalProject) {
	if len(orgs) > 1 {
		ids := make([]string, 0, len(orgs))
		for _, org := range orgs {
			ids = append(ids, org.id)
		}

		slog.Warn("hostsvc: pat minter multiple personal organizations, picking deterministic first",
			"code", "pat-multipick-org", "picked", picked.id, "candidates", ids)
	}

	if len(picked.projects) > 1 {
		ids := make([]string, 0, len(picked.projects))
		for _, p := range picked.projects {
			ids = append(ids, p.id)
		}

		slog.Warn("hostsvc: pat minter multiple personal projects, picking deterministic first",
			"code", "pat-multipick-project", "picked", proj.id, "candidates", ids)
	}
}

// locateOrCreateKey resolves the account key id: list the org/project
// api_keys (Bearer form, zai) and return the zcode-api-key candidate's id;
// no candidate → create-if-absent (D-07: the desktop's own posture,
// superseding the reference's read-only key_missing).
func (m *PatMinter) locateOrCreateKey(ctx context.Context, login, orgID, projID string) (string, error) {
	obj, err := m.callJSON(ctx, http.MethodGet,
		patKeyListURL(m.origin, orgID, projID), "", patBearerPrefix+login)
	if err != nil {
		return "", err
	}

	data, err := patEnvelopeData(obj)
	if err != nil {
		return "", err
	}

	keys, ok := data.([]any)
	if !ok {
		return "", ErrPatInvalidResponse
	}

	for _, rawKey := range keys {
		entry, ok := rawKey.(map[string]any)
		if !ok {
			continue
		}

		if name, _ := entry["name"].(string); name == patKeyName {
			return patKeyIDFrom(entry)
		}
	}

	// Check-then-create idempotency: the candidate search precedes every
	// mint, so the create fires only when the list carried no candidate.
	return m.createKey(ctx, login, orgID, projID)
}

// createKey POSTs the byte-exact create body to the same api_keys URL and
// returns the created entry's id (the reply is the created key entry).
func (m *PatMinter) createKey(ctx context.Context, login, orgID, projID string) (string, error) {
	obj, err := m.callJSON(ctx, http.MethodPost,
		patKeyListURL(m.origin, orgID, projID), patCreateKeyBody, patBearerPrefix+login)
	if err != nil {
		return "", err
	}

	data, err := patEnvelopeData(obj)
	if err != nil {
		return "", err
	}

	entry, ok := data.(map[string]any)
	if !ok {
		return "", ErrPatInvalidResponse
	}

	return patKeyIDFrom(entry)
}

// patKeyIDFrom digs the non-empty apiKey id out of a key entry (a
// candidate/created entry without a usable id is the invalid-response arm).
func patKeyIDFrom(entry map[string]any) (string, error) {
	keyID := patNonEmpty(entry["apiKey"])
	if keyID == "" {
		return "", ErrPatInvalidResponse
	}

	return keyID, nil
}

// issueToken exchanges the login for the project access token: POST
// access_tokens with the Bearer header and the byte-exact body; the reply
// validation and expiry math live in patFromReply.
func (m *PatMinter) issueToken(
	ctx context.Context,
	login string,
	loc *patKeyLocation,
	startedAt time.Time,
	scope string,
) (Pat, error) {
	body := fmt.Sprintf(`{"clientType":"zcode","clientVersion":%q}`, m.version())
	accessURL := patKeyListURL(m.origin, loc.organizationID, loc.projectID) +
		"/" + url.PathEscape(loc.apiKeyID) + patAccessSuffix

	obj, err := m.callJSON(ctx, http.MethodPost, accessURL, body, patBearerPrefix+login)
	if err != nil {
		return Pat{}, err
	}

	if patCodeIs404(obj["code"]) {
		// The ONLY location-invalidating failure: the key no longer exists
		// at its cached location (HTTP 404 arrives as the pseudo envelope,
		// so both forms land here — Pitfall 8). Permission and network
		// errors never reach this arm.
		return Pat{}, ErrPatKeyMissing
	}

	data, err := patEnvelopeData(obj)
	if err != nil {
		return Pat{}, err
	}

	dataObj, dataOK := data.(map[string]any)
	if !dataOK {
		return Pat{}, ErrPatInvalidResponse
	}

	return patFromReply(dataObj, loc, startedAt, scope, m.clock())
}

// patFromReply validates the access_tokens reply shape (the reference's
// guards: non-empty token, tokenType Bearer, positive expiresIn/expiresAt,
// enable !== false) and applies the conservative expiry math (expiresAt =
// min(expiresAtSec*1000, startedAt+expiresIn*1000); refreshAt =
// max(startedAt, expiresAt-120s)). A dead expiry is the invalid-response arm
// — never a partial serve.
func patFromReply(
	dataObj map[string]any,
	loc *patKeyLocation,
	startedAt time.Time,
	scope string,
	now time.Time,
) (Pat, error) {
	token := patNonEmpty(dataObj["accessToken"])

	tokenType, _ := dataObj["tokenType"].(string)
	expiresIn, expiresInOK := patPositiveNumber(dataObj["expiresIn"])
	expiresAtSec, expiresAtSecOK := patPositiveNumber(dataObj["expiresAt"])

	if v, present := dataObj["enable"]; present {
		if disabled, isBool := v.(bool); isBool && !disabled {
			return Pat{}, ErrPatDisabled
		}
	}

	if token == "" || tokenType != patTokenType || !expiresInOK || !expiresAtSecOK {
		return Pat{}, ErrPatInvalidResponse
	}

	fromExpiresAt := time.UnixMilli(int64(expiresAtSec * patSecToMilli))
	fromExpiresIn := startedAt.Add(time.Duration(expiresIn * float64(time.Second)))

	expiresAt := fromExpiresAt
	if fromExpiresIn.Before(expiresAt) {
		expiresAt = fromExpiresIn
	}

	if !expiresAt.After(now) {
		return Pat{}, ErrPatInvalidResponse
	}

	refreshAt := expiresAt.Add(-patRefreshLead)
	if startedAt.After(refreshAt) {
		refreshAt = startedAt
	}

	return Pat{
		Token:        token,
		APIKeyID:     loc.apiKeyID,
		Fingerprint:  patSHA256Hex(token),
		AccountScope: scope,
		ExpiresAt:    expiresAt,
		RefreshAt:    refreshAt,
	}, nil
}

// callJSON runs one bounded mint HTTPS call: explicit Content-Type and
// Authorization headers (the caller composes raw vs Bearer), the 2xx band
// only, and the {code, data} business envelope decoded — HTTP 404 renders as
// the pseudo envelope {code:404} so the access_tokens location rule can
// discriminate it from other failures (the reference's requestJson shape).
// Transport errors classify into the coded timeout/network family; the body
// is size-bounded and never logged.
func (m *PatMinter) callJSON(
	ctx context.Context,
	method, rawURL, body, auth string,
) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, strings.NewReader(body))
	if err != nil {
		return nil, ErrPatInvalidResponse
	}

	req.Header.Set("Content-Type", patContentType)
	req.Header.Set("Authorization", auth)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, patClassifyTransportError(err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == patCodeNotFound {
		// The reference's 404 form: a pseudo envelope the caller's code-404
		// check reads uniformly, whether the 404 came from HTTP or business.
		return map[string]any{"code": float64(patCodeNotFound)}, nil
	}

	if resp.StatusCode < patHTTPMultiMin || resp.StatusCode > patHTTPMultiMax {
		return nil, &patHTTPStatusError{status: resp.StatusCode}
	}

	payload, err := io.ReadAll(io.LimitReader(resp.Body, patMaxBodyBytes))
	if err != nil {
		return nil, ErrPatNetworkError
	}

	var obj map[string]any
	if err := json.Unmarshal(payload, &obj); err != nil {
		return nil, ErrPatInvalidResponse
	}

	return obj, nil
}

// patClassifyTransportError maps a client.Do failure onto the coded family:
// deadline/timeout shapes answer ErrPatTimeout, everything else
// ErrPatNetworkError (the reference's transient classes; the original error
// text never carries into the coded answer).
func patClassifyTransportError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || patIsTimeoutShape(err) {
		return ErrPatTimeout
	}

	return ErrPatNetworkError
}

// patIsTimeoutShape reports whether the transport error carries a timeout
// marker (net.Error's Timeout() shape).
func patIsTimeoutShape(err error) bool {
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}

// patKeyListURL renders the api_keys collection URL for the org/project pair
// (ids path-escaped, the reference's encodeURIComponent form).
func patKeyListURL(origin, orgID, projID string) string {
	return origin + fmt.Sprintf(patKeyListURLFormat, url.PathEscape(orgID), url.PathEscape(projID))
}

// patAccountScope computes the byte-parity accountScope: sha256 hex over the
// COMPACT JSON array [family, plan kind, null, loginToken, zcodeJwtOrNil] —
// encoding/json's compact marshal is the JSON.stringify parity (no
// whitespace), lowercase 64-hex. The JWT folds as null when absent.
func patAccountScope(family, loginToken, zcodeJWT string) string {
	var jwtSlot any
	if zcodeJWT != "" {
		jwtSlot = zcodeJWT
	}

	payload, err := json.Marshal([]any{family, patPlanKind, nil, loginToken, jwtSlot})
	if err != nil {
		// Unreachable for the string/nil array — the honest empty answer
		// rather than a panic (an empty scope can never match an
		// expectedAccountScope, so a failure cannot slip through).
		return ""
	}

	sum := sha256.Sum256(payload)

	return hex.EncodeToString(sum[:])
}

// patSHA256Hex is the shared sha256-hex form (fingerprints, login guard).
func patSHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))

	return hex.EncodeToString(sum[:])
}

// patNonEmpty digs a non-empty trimmed string out of a decoded JSON value
// (the reference's nonEmpty: strings trim to null otherwise).
func patNonEmpty(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(s)
}

// patPositiveNumber digs a finite positive number out of a decoded JSON value
// (the reference's typeof-number-and-positive reply guards).
func patPositiveNumber(v any) (float64, bool) {
	n, ok := v.(float64)
	if !ok || !(n > 0) {
		return 0, false
	}

	return n, true
}

// patEnvelopeData extracts the `data` of a {code, data} business envelope,
// refusing any non-success code (the success set {200, 0, "200", "0"}) with
// the coded invalid-response error — the reference's envelopeData.
func patEnvelopeData(obj map[string]any) (any, error) {
	if !patSuccessCode(obj["code"]) {
		return nil, ErrPatInvalidResponse
	}

	return obj["data"], nil
}

// patSuccessCode reports whether an envelope code is in the success set
// {200, 0, "200", "0"} (numeric and string forms).
func patSuccessCode(code any) bool {
	switch v := code.(type) {
	case float64:
		return v == patCodeOK || v == patCodeZero
	case string:
		return v == patCodeOKStr || v == patCodeZeroStr
	default:
		return false
	}
}

// patCodeIs404 reports whether an envelope code is the numeric 404 — the
// key-missing marker on access_tokens (the pseudo envelope from an HTTP 404
// and a business-code 404 both land here; the reference's strict numeric
// check).
func patCodeIs404(code any) bool {
	n, ok := code.(float64)

	return ok && int(n) == patCodeNotFound
}

// patProjectTypeIsTeam reports whether a projectType value (string, number,
// or null on the wire) marks a team project — String(projectType).trim()
// compared against the team marker (the D-06 filter).
func patProjectTypeIsTeam(v any) bool {
	return strings.TrimSpace(patAnyToString(v)) == patTeamProjectType
}

// patAnyToString renders a decoded JSON value the way the reference's
// String() coercion does (numbers render without trailing zeros; nil is the
// empty string).
func patAnyToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}
