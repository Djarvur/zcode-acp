package interact

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// The D-06 CredentialDecrypt seam (plan 05-07): the requestCredentialCipher
// row's decrypt service. The field-verified shapes (05-RESEARCH.md Pattern
// 6) are the contract: the ask params are {requestId, operation:
// "encrypt"|"decrypt", value}; a decrypt of an enc:v2:os credential blob
// loads the login-keyring key through the injectable KeyLoader and answers
// {requestId, result:{ok:true, value:<decrypted>}}; every other arm —
// encrypt ops, malformed blobs, loader failures, key mismatch, GCM auth
// failure — answers the {ok:false, error:{code, message}} decline inside
// the reply RESULT (never a fabricated success, never a panic, never a
// stall). CI tiers never touch the keyring: the production loader executes
// only when a real-tier cipher ask actually arrives.
//
// Key-material hygiene (T-5-17, test-locked): key bytes and decrypted
// plaintext live in-memory only, in the reply-construction scope — never
// persisted, never logged (log fields carry shapes and lengths, never
// values), never on the loader subprocess's argv (the username argument is
// the sha256 hex derivation over the RESOLVED credentials path), and the
// subprocess runs on the caller-provided scrubbed env with its stdout
// captured and drained. The GCM auth failure's decline carries a generic
// message — the crypto error text is never echoed into the reply.

// EncV2OSPrefix is the field-verified credential-blob layout prefix
// (enc:v2:os:<kid>:<iv>.<tag>.<ct>; segments iv/tag/ct base64url).
const EncV2OSPrefix = "enc:v2:os:"

// The cipher ask's operation param values (dialect §3).
const (
	CipherOpDecrypt = "decrypt"
	CipherOpEncrypt = "encrypt"
)

// DefaultCredentialCipherDeadline is the wired policy row's answer deadline:
// the same 15 s window the conn's tracked-no-answer tracker holds (the D-05
// posture this row replaces when the seam is injected). A loader failure
// answers the {ok:false} decline well inside it — never a stall (T-5-19).
const DefaultCredentialCipherDeadline = zcode.RouterDefaultDeadline

// Decline codes for the ok:false arm — coded and stable; consumers match
// the code, never the message (dialect §6 posture carried into the union).
const (
	CipherCodeUnsupportedOperation = "unsupported-operation"
	CipherCodeInvalidValue         = "invalid-value"
	CipherCodeKeyUnavailable       = "key-unavailable"
	CipherCodeKeyMismatch          = "key-mismatch"
	CipherCodeDecryptFailed        = "decrypt-failed"
)

// cipherCannotConstruct is the shared generic message for the arms where
// the AES-256-GCM primitive itself cannot be constructed (goconst).
const cipherCannotConstruct = "credential cipher cannot be constructed"

// secretToolService is the field-verified Secret Service attribute value
// the login-keyring credential entry is keyed under (a lookup key, never a
// credential — gosec's hardcoded-credentials shape match is a false hit).
//
//nolint:gosec // a Secret Service lookup key string, not a credential value
const secretToolService = "ZCode Credential Encryption"

// loaderKeyLen is the keyring key's decoded length: the field-verified
// 43-char base64url form decodes to exactly 32 AES-256 bytes.
const loaderKeyLen = 32

// kidHexLen is the blob kid's length: the first 24 hex chars of sha256(key).
const kidHexLen = 24

// cipherSegmentCount is the blob's dot-separated segment count: iv.tag.ct.
const cipherSegmentCount = 3

// loaderEnvAllowlist is the scrubbed-env allowlist posture for the loader
// subprocess (the supervisor.ChildEnv discipline): only these keys cross
// from the parent env — the D-Bus session address the keyring lives on,
// plus PATH/HOME for the binary and its config resolution. Everything else
// (proxies, tokens, caller-specific env) never crosses. A fresh slice per
// call keeps the contract immutable.
func loaderEnvAllowlist() []string {
	return []string{"PATH", "HOME", "DBUS_SESSION_BUS_ADDRESS", "XDG_RUNTIME_DIR"}
}

// Hygiene sentinels — wrapped with %w context at the failure site, never
// constructed dynamically (err113).
var (
	errEmptyCredentialsPath = errors.New("credentials path is empty")
	errLoaderKeyShape       = errors.New("keyring key is not 32 bytes of base64url")
	errLoaderLookup         = errors.New("secret-tool lookup failed")
	// errV1SecretInputs is the enc:v1 loud-validation sentinel: an
	// unresolvable machine input (platform, home, username) never derives
	// a silently-empty machine string.
	errV1SecretInputs = errors.New("enc:v1 fallback machine inputs are unresolved")
)

// cipherFailure is one decline arm's coded payload: the stable code plus a
// generic message. Failure detail never rides the reply — loader and GCM
// errors carry no key/plaintext material into any message or log (T-5-17).
type cipherFailure struct {
	code    string
	message string
}

// KeyLoader loads a 32-byte credential-decryption key. Two layouts join
// the seam (D-01/D-03 — one seam, two layouts): the v2:os keyring loader
// shells the field-verified secret-tool lookup (SecretToolKeyLoader), and
// the legacy enc:v1 fallback derives its key from the resolved secret by
// pure compute (V1FallbackKeyLoader — no exec). A Phase-7 godbus/dbus swap
// must be a local seam edit — no business logic may bypass the seam (the
// interface is the upgrade boundary, D-06 recommend-then-upgrade posture).
type KeyLoader interface {
	LoadKey(ctx context.Context) ([]byte, error)
}

// CredentialDecrypt is the D-06 seam service: the requestCredentialCipher
// policy row's handler. Constructed once via NewCredentialDecrypt; Handle
// is safe for concurrent use.
type CredentialDecrypt struct {
	loader KeyLoader
}

// NewCredentialDecrypt returns the service over the injectable key loader.
func NewCredentialDecrypt(loader KeyLoader) *CredentialDecrypt {
	return &CredentialDecrypt{loader: loader}
}

// Handle answers one interaction/requestCredentialCipher ask (the
// interact.Handler signature): every arm — success included — answers
// inside the field-verified reply union {requestId, result:{ok...}}; the
// handler never errors (a decline IS the answer, never a transport
// failure).
func (s *CredentialDecrypt) Handle(ctx context.Context, req Request) (Answer, error) {
	var params struct {
		Operation string `json:"operation"`
		Value     string `json:"value"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		slog.Warn("interact: credential cipher declined (undecodable params)",
			"requestId", req.ReqID, "params_len", len(req.Params), "code", CipherCodeInvalidValue)

		//nolint:nilerr // the decline IS the answer: every arm — malformed
		// params included — rides the reply result union, never the error lane.
		return Answer{Result: cipherDecline(req, CipherCodeInvalidValue,
			"credential cipher params are malformed")}, nil
	}

	switch params.Operation {
	case CipherOpDecrypt:
		value, fail := s.decryptOne(ctx, params.Value)
		if fail != nil {
			slog.Warn("interact: credential cipher decrypt declined",
				"requestId", req.ReqID, "value_len", len(params.Value),
				"code", fail.code)

			return Answer{Result: cipherDecline(req, fail.code, fail.message)}, nil
		}

		slog.Info("interact: credential cipher decrypt answered",
			"requestId", req.ReqID, "value_len", len(params.Value), "plaintext_len", len(value))

		return Answer{Result: cipherSuccess(req, value)}, nil

	case CipherOpEncrypt:
		slog.Warn("interact: credential cipher encrypt declined",
			"requestId", req.ReqID, "value_len", len(params.Value), "code", CipherCodeUnsupportedOperation)

		return Answer{Result: cipherDecline(req, CipherCodeUnsupportedOperation,
			`operation "encrypt" is not supported: the wrapper decrypts `+
				`the desktop's credential blobs but never encrypts`)}, nil

	default:
		slog.Warn("interact: credential cipher declined (unknown operation)",
			"requestId", req.ReqID, "value_len", len(params.Value), "code", CipherCodeInvalidValue)

		return Answer{Result: cipherDecline(req, CipherCodeInvalidValue,
			"credential cipher operation must be decrypt or encrypt")}, nil
	}
}

// decryptOne runs one decrypt: layout parse → key load → kid verification →
// stdlib AES-256-GCM open. Every failure renders as a coded decline; the
// plaintext exists only inside this call's reply-construction scope.
func (s *CredentialDecrypt) decryptOne(ctx context.Context, value string) (string, *cipherFailure) {
	kid, iv, tag, ct, fail := parseEncV2OS(value)
	if fail != nil {
		return "", fail
	}

	if s.loader == nil {
		return "", &cipherFailure{code: CipherCodeKeyUnavailable, message: "no key loader configured"}
	}

	key, err := s.loader.LoadKey(ctx)
	if err != nil {
		// The loader's failure detail (exec noise, keyring state) never
		// rides the reply or the log — the generic coded message is the
		// honest degradation arm (D-06 keyring-locked posture, T-5-17).
		return "", &cipherFailure{code: CipherCodeKeyUnavailable,
			message: "the credential key is unavailable (keyring locked or missing)"}
	}

	if len(key) != loaderKeyLen {
		return "", &cipherFailure{code: CipherCodeKeyUnavailable,
			message: "the credential key has the wrong length"}
	}

	if kid != kidFor(key) {
		return "", &cipherFailure{code: CipherCodeKeyMismatch,
			message: "credential blob kid does not match the keyring key"}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", &cipherFailure{code: CipherCodeDecryptFailed,
			message: cipherCannotConstruct}
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", &cipherFailure{code: CipherCodeDecryptFailed,
			message: cipherCannotConstruct}
	}

	sealed := make([]byte, 0, len(ct)+len(tag))
	sealed = append(sealed, ct...)
	sealed = append(sealed, tag...)

	plaintext, err := gcm.Open(nil, iv, sealed, nil)
	if err != nil {
		// The crypto error text is generic, but the decline never echoes it
		// — the auth-failure arm is the coded generic message (hygiene).
		return "", &cipherFailure{code: CipherCodeDecryptFailed,
			message: "credential blob failed AES-256-GCM authentication"}
	}

	return string(plaintext), nil
}

// parseEncV2OS parses the field-verified layout: the enc:v2:os prefix, the
// kid segment (colon-separated), then the base64url iv.tag.ct segments.
// Malformed shapes answer the coded invalid-value decline — never a panic.
func parseEncV2OS(value string) (kid string, iv, tag, ct []byte, fail *cipherFailure) {
	if !strings.HasPrefix(value, EncV2OSPrefix) {
		return "", nil, nil, nil, &cipherFailure{code: CipherCodeInvalidValue,
			message: "credential value is not an enc:v2:os blob"}
	}

	body := strings.TrimPrefix(value, EncV2OSPrefix)

	kid, rest, ok := strings.Cut(body, ":")
	if !ok || kid == "" {
		return "", nil, nil, nil, &cipherFailure{code: CipherCodeInvalidValue,
			message: "credential blob carries no kid segment"}
	}

	segs := strings.Split(rest, ".")
	if len(segs) != cipherSegmentCount {
		return "", nil, nil, nil, &cipherFailure{code: CipherCodeInvalidValue,
			message: "credential blob layout is malformed (want iv.tag.ct)"}
	}

	iv, errIV := base64.RawURLEncoding.DecodeString(segs[0])
	tag, errTag := base64.RawURLEncoding.DecodeString(segs[1])
	ct, errCT := base64.RawURLEncoding.DecodeString(segs[2])

	if errIV != nil || errTag != nil || errCT != nil {
		return "", nil, nil, nil, &cipherFailure{code: CipherCodeInvalidValue,
			message: "credential blob segments are not base64url"}
	}

	return kid, iv, tag, ct, nil
}

// kidFor renders the field-verified kid: the first 24 hex chars of
// sha256(key) — the same derivation the desktop uses to bind a blob to its
// keyring key.
func kidFor(key []byte) string {
	sum := sha256.Sum256(key)

	return hex.EncodeToString(sum[:])[:kidHexLen]
}

// cipherSuccess renders the {ok:true, value} arm: the requestId echo plus
// the decrypted plaintext, keyed verbatim per the field-verified reply
// schema (credentialCipherReplySchema = {requestId, result:
// discriminatedUnion("ok")}).
func cipherSuccess(req Request, value string) map[string]any {
	return map[string]any{
		"requestId": req.ReqID,
		"result": map[string]any{
			"ok":    true,
			"value": value,
		},
	}
}

// cipherDecline renders the {ok:false, error{code, message}} arm.
func cipherDecline(req Request, code, message string) map[string]any {
	return map[string]any{
		"requestId": req.ReqID,
		"result": map[string]any{
			"ok":    false,
			"error": map[string]any{"code": code, "message": message},
		},
	}
}

// SecretToolKeyLoader is the production KeyLoader: it shells the
// field-verified lookup `secret-tool lookup service "ZCode Credential
// Encryption" username <sha256(resolved credentials path)>` on the
// caller-provided scrubbed env, captures the stdout (the 43-char base64url
// key), and releases it into the reply scope. Real tier only: CI never
// constructs nor executes it against a keyring (D-06) — the loader runs
// only when a cipher ask actually arrives on a wired binary.
type SecretToolKeyLoader struct {
	bin      string
	credPath string
	env      []string
}

// NewSecretToolKeyLoader returns the loader over the secret-tool binary
// name, the desktop credentials.json path (resolved at lookup time), and
// the scrubbed subprocess env (see ScrubbedEnv).
func NewSecretToolKeyLoader(bin, credentialsPath string, env []string) *SecretToolKeyLoader {
	return &SecretToolKeyLoader{bin: bin, credPath: credentialsPath, env: env}
}

// LookupCommand builds the lookup subprocess contract the loader executes:
// the argv and the env it runs under. Exported for the hygiene locks —
// tests inspect the contract without executing the binary. The username
// argument is the sha256 hex derivation over the RESOLVED credentials path
// (EvalSymlinks + Abs — the realpath discipline), derived from a PATH
// STRING, never key material: the loader holds no key at argv-build time.
func (l *SecretToolKeyLoader) LookupCommand() (argv, env []string, err error) {
	if l.bin == "" || l.credPath == "" {
		return nil, nil, errEmptyCredentialsPath
	}

	link, err := resolveRealPath(l.credPath)
	if err != nil {
		return nil, nil, err
	}

	sum := sha256.Sum256([]byte(link))
	username := hex.EncodeToString(sum[:])

	argv = []string{l.bin, "lookup", "service", secretToolService, "username", username}

	return argv, l.env, nil
}

// resolveRealPath resolves one path the realpath discipline: EvalSymlinks +
// Abs over the PATH STRING (the supervisor.BuiltinRevision formula's input
// contract). A missing file fails loud — a lookup hashed against a phantom
// path would answer a wrong key silently.
func resolveRealPath(path string) (string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve credentials path %q: %w", path, err)
	}

	link, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve credentials path %q: %w", path, err)
	}

	return link, nil
}

// LoadKey shells the lookup and decodes the captured stdout: trimmed
// base64url, exactly 32 bytes. cmd.Output captures and drains stdout (the
// key never touches a pipe the process doesn't own) and carries stderr on
// the exec error — which this package wraps and never logs (hygiene: the
// failure detail rides nowhere; the decline's generic message is the wire
// surface).
func (l *SecretToolKeyLoader) LoadKey(ctx context.Context) ([]byte, error) {
	argv, env, err := l.LookupCommand()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoaderLookup, err)
	}

	//nolint:gosec // the argv is the built field-verified contract, not user input
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = env

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoaderLookup, err)
	}

	key, err := decodeLoaderKey(out)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// decodeLoaderKey decodes the captured stdout: surrounding whitespace
// trimmed, base64url, exactly 32 bytes.
func decodeLoaderKey(raw []byte) ([]byte, error) {
	key, err := base64.RawURLEncoding.DecodeString(string(bytes.TrimSpace(raw)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLoaderKeyShape, err)
	}

	if len(key) != loaderKeyLen {
		return nil, fmt.Errorf("%w: got %d bytes", errLoaderKeyShape, len(key))
	}

	return key, nil
}

// V1SecretEnvKey is the env key the enc:v1 fallback resolves first (the
// reference's ZCODE_CREDENTIAL_SECRET — the trimmed value wins) and
// v1SecretFallbackPrefix the machine-string fallback's prefix (ref
// credential-store.ts:33-43).
const (
	//nolint:gosec // an env KEY name, not a credential value
	V1SecretEnvKey = "ZCODE_CREDENTIAL_SECRET"
	//nolint:gosec // the fallback secret's prefix literal, not a credential value
	v1SecretFallbackPrefix = "zcode-credential-fallback:"
)

// V1FallbackKeyLoader is the legacy enc:v1 sibling KeyLoader (D-03) joining
// the SecretToolKeyLoader family behind the same seam — one seam, two
// layouts. Port attribution (Apache-2.0): the v1 secret resolution and
// layout are ported from william0wang/zcode-acp v0.89.0
// src/backend/credential-store.ts (docs/research/reference-adapters.md).
//
// The derivation is pure compute — unlike the v2 keyring read it performs
// NO exec: secret = the trimmed ZCODE_CREDENTIAL_SECRET env value when
// set, else the machine string
// `zcode-credential-fallback:<GOOS>:<home>:<username>`; key = sha256(secret).
type V1FallbackKeyLoader struct {
	env      map[string]string
	platform string
	home     string
	username string
}

// NewV1FallbackKeyLoader returns the loader over the caller-provided
// secret inputs. The machine inputs resolve through the provided seams
// (nil = the OS defaults: runtime.GOOS, os.UserHomeDir, user.Current) and
// validate loud: an empty platform, home, or username is the sentinel
// error — never a silently-empty machine string (that would hash the
// wrong key and fail every GCM open with no cue). env == nil consults the
// process env at LoadKey time (the reference reads process.env at decrypt
// time).
func NewV1FallbackKeyLoader(
	env map[string]string, platform, home, username func() string,
) (*V1FallbackKeyLoader, error) {
	if platform == nil {
		platform = func() string { return runtime.GOOS }
	}

	if home == nil {
		home = func() string {
			h, _ := os.UserHomeDir()

			return h
		}
	}

	if username == nil {
		username = func() string {
			u, err := user.Current()
			if err != nil {
				return ""
			}

			return u.Username
		}
	}

	loader := &V1FallbackKeyLoader{
		env:      env,
		platform: platform(),
		home:     home(),
		username: username(),
	}

	if _, err := ResolveV1CredentialSecret(nil, loader.platform, loader.home, loader.username); err != nil {
		return nil, err
	}

	return loader, nil
}

// ResolveV1CredentialSecret is the pure secret-resolution function (the
// env-then-machine-string order with trim semantics, exposed for the
// parity vectors; ref credential-store.ts:33-43): the trimmed non-empty
// ZCODE_CREDENTIAL_SECRET env value wins; otherwise the machine-bound
// fallback string `zcode-credential-fallback:<platform>:<home>:<username>`.
// Empty machine inputs fail loud (the sentinel) — never a silently-empty
// machine string.
func ResolveV1CredentialSecret(
	env map[string]string, platform, home, username string,
) (string, error) {
	if platform == "" || home == "" || username == "" {
		return "", fmt.Errorf("%w: platform/home/username must be non-empty "+
			"(got %d/%d/%d chars)", errV1SecretInputs, len(platform), len(home), len(username))
	}

	if v, ok := env[V1SecretEnvKey]; ok {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed, nil
		}
	}

	return v1SecretFallbackPrefix + platform + ":" + home + ":" + username, nil
}

// LoadKey resolves the secret (env-else-machine-string) and returns
// sha256(secret) — the exact 32-byte key the enc:v1 layout encrypts
// under. Pure compute: no exec, no subprocess, no env scrubbing needed.
func (l *V1FallbackKeyLoader) LoadKey(_ context.Context) ([]byte, error) {
	secret, err := ResolveV1CredentialSecret(l.envMap(), l.platform, l.home, l.username)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256([]byte(secret))

	return sum[:], nil
}

// envMap resolves the env seam: the injected map when present, else a
// snapshot of the process env at load time (the reference reads
// process.env at decrypt time).
func (l *V1FallbackKeyLoader) envMap() map[string]string {
	if l.env != nil {
		return l.env
	}

	out := make(map[string]string)

	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			out[k] = v
		}
	}

	return out
}

// ScrubbedEnv builds the loader subprocess env from a parent env via the
// allowlist (the ChildEnv posture): only the allowlisted keys cross, in
// allowlist order, each at most once. The cmd wiring passes
// ScrubbedEnv(os.Environ()) — the loader subprocess never sees a proxy
// var, a token, or any other ambient secret.
func ScrubbedEnv(parentEnv []string) []string {
	parent := make(map[string]string, len(parentEnv))
	for _, kv := range parentEnv {
		if k, v, ok := strings.Cut(kv, "="); ok {
			parent[k] = v
		}
	}

	allow := loaderEnvAllowlist()

	env := make([]string, 0, len(allow))
	for _, key := range allow {
		if v, ok := parent[key]; ok {
			env = append(env, key+"="+v)
		}
	}

	return env
}
