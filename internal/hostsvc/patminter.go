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

	// patPlanKind is the plan kind slot of the accountScope array.
	patPlanKind = "individual-coding-plan"

	// patRequestTimeout bounds every mint HTTPS call (ref REQUEST_TIMEOUT_MS),
	// set once on the client at construction (Pitfall 7: bounded I/O, never a
	// hang).
	patRequestTimeout = 15 * time.Second

	// patRefreshLead refreshes this long before expiry so an in-flight turn
	// never carries a dead token (ref REFRESH_LEAD_MS).
	patRefreshLead = 120 * time.Second

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

	// token is the cached Pat (nil = none).
	token *Pat
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
// official chain when needed. Every failure is a coded stable error and the
// cache is only ever advanced by a successful mint (never a partial or
// poisoned entry).
func (m *PatMinter) Ensure(ctx context.Context) (Pat, error) {
	if m.creds == nil {
		// No credential source configured — the desktop-login arm is the
		// honest posture (an unconfigured minter is simply not logged in).
		return Pat{}, ErrPatLoginRequired
	}

	raw, _ := m.creds.LoginToken(ctx, patFamily)
	login := strings.TrimSpace(raw)

	pat, err := m.mint(ctx, login)
	if err != nil {
		return Pat{}, err
	}

	m.mu.Lock()
	fp := patSHA256Hex(login)
	if m.sess == nil || m.sess.loginFP != fp {
		m.sess = &patSession{loginFP: fp}
	}

	m.sess.token = &pat
	m.mu.Unlock()

	return pat, nil
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

// mint runs the official chain once: the accountScope digest over the current
// sign-in, the key-location resolution (getCustomerInfo -> pick -> list), and
// the access_tokens exchange.
func (m *PatMinter) mint(ctx context.Context, login string) (Pat, error) {
	jwt, _ := m.creds.ZcodeJWT(ctx)
	scope := patAccountScope(patFamily, login, jwt)

	loc, err := m.resolveLocation(ctx, login)
	if err != nil {
		return Pat{}, err
	}

	return m.issueToken(ctx, login, loc, m.clock(), scope)
}

// resolveLocation resolves where the account key lives: getCustomerInfo with
// the RAW token, the personal pick (D-06), then the api_keys list. The result
// is recomputed per mint in this slice (the cross-mint location cache with
// the 404 drop rule is the edge-arm slice's subject).
func (m *PatMinter) resolveLocation(ctx context.Context, login string) (*patKeyLocation, error) {
	orgID, projID, err := m.pickPersonal(ctx, login)
	if err != nil {
		return nil, err
	}

	keyID, err := m.locateKey(ctx, login, orgID, projID)
	if err != nil {
		return nil, err
	}

	return &patKeyLocation{organizationID: orgID, projectID: projID, apiKeyID: keyID}, nil
}

// pickPersonal digs the customer-info envelope and picks the personal org and
// project: personal-layer ids must be non-empty; this slice takes the
// deterministic first pair (the D-06 team filter, marker preference,
// multi-candidate warn, and the coded unavailable arm land with their
// vectors).
func (m *PatMinter) pickPersonal(ctx context.Context, login string) (orgID, projID string, err error) {
	obj, err := m.callJSON(ctx, http.MethodGet, m.origin+patCustomerInfoPath, "", login)
	if err != nil {
		return "", "", err
	}

	data, err := patEnvelopeData(obj)
	if err != nil {
		return "", "", err
	}

	dataObj, dataOK := data.(map[string]any)
	if !dataOK {
		return "", "", ErrPatInvalidResponse
	}

	orgs, ok := dataObj["organizations"].([]any)
	if !ok {
		return "", "", ErrPatInvalidResponse
	}

	for _, rawOrg := range orgs {
		org, ok := rawOrg.(map[string]any)
		if !ok {
			continue
		}

		orgID := patNonEmpty(org["organizationId"])
		if orgID == "" {
			continue
		}

		orgProjects, ok := org["projects"].([]any)
		if !ok {
			continue
		}

		for _, rawProj := range orgProjects {
			proj, ok := rawProj.(map[string]any)
			if !ok {
				continue
			}

			projID := patNonEmpty(proj["projectId"])
			if projID == "" {
				continue
			}

			return orgID, projID, nil
		}
	}

	return "", "", ErrPatInvalidResponse
}

// locateKey lists the org/project api_keys (Bearer form, zai) and returns the
// id of the zcode-api-key candidate. A missing candidate answers the
// reference's read-only coded arm in this slice (the D-07 create-if-absent
// lands with its vector).
func (m *PatMinter) locateKey(ctx context.Context, login, orgID, projID string) (string, error) {
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
			keyID := patNonEmpty(entry["apiKey"])
			if keyID == "" {
				return "", ErrPatInvalidResponse
			}

			return keyID, nil
		}
	}

	return "", ErrPatKeyMissing
}

// issueToken exchanges the login for the project access token: POST
// access_tokens with the Bearer header and the byte-exact body, validate the
// reply shape, and apply the conservative expiry math (expiresAt =
// min(expiresAtSec*1000, startedAt+expiresIn*1000); refreshAt =
// max(startedAt, expiresAt-120s)).
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

	data, err := patEnvelopeData(obj)
	if err != nil {
		return Pat{}, err
	}

	dataObj, dataOK := data.(map[string]any)
	if !dataOK {
		return Pat{}, ErrPatInvalidResponse
	}

	token := patNonEmpty(dataObj["accessToken"])

	tokenType, _ := dataObj["tokenType"].(string)
	expiresIn, expiresInOK := patPositiveNumber(dataObj["expiresIn"])
	expiresAtSec, expiresAtSecOK := patPositiveNumber(dataObj["expiresAt"])

	if token == "" || tokenType != patTokenType || !expiresInOK || !expiresAtSecOK {
		return Pat{}, ErrPatInvalidResponse
	}

	fromExpiresAt := time.UnixMilli(int64(expiresAtSec * patSecToMilli))
	fromExpiresIn := startedAt.Add(time.Duration(expiresIn * float64(time.Second)))

	expiresAt := fromExpiresAt
	if fromExpiresIn.Before(expiresAt) {
		expiresAt = fromExpiresIn
	}

	if !expiresAt.After(m.clock()) {
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
