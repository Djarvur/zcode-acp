package interact

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"runtime"
	"strings"
)

// AUTH-02 credential-store base (plan 06-01): a read-only reader over the
// desktop's ~/.zcode/v2/credentials.json — a JSON object of key → string
// value whose values are plaintext, enc:v1: blobs, or enc:v2:os: blobs.
//
// Port attribution (Apache-2.0): the store layout, the dispatch ladder,
// and the never-plaintext rule are ported from william0wang/zcode-acp
// v0.89.0 src/backend/credential-store.ts (Apache-2.0; no NOTICE file —
// attribution per docs/research/reference-adapters.md).
//
// Dispatch ladder (06-RESEARCH.md Pattern 1, wire formats verbatim):
//
//   - no enc: prefix → plaintext passthrough verbatim;
//   - enc:v1:<iv>.<tag>.<ct> → the legacy fallback decrypt, key =
//     sha256(secret) (D-03);
//   - enc:v2:os:<fp24>:<iv>.<tag>.<ct> → the composed CredentialDecrypt
//     seam service over the injected KeyLoader (the fp24 checksum is
//     verified before any GCM open — the seam is the only keyring path,
//     D-01);
//   - any other enc: prefix → the entry is SKIPPED with a structural log
//     (the hasUnhandledEncPrefix never-plaintext rule — a silent plaintext
//     downgrade of encrypted credentials is impossible; the upstream
//     v2-ENV degradation of 2026-11-12 is the locked failure mode).
//
// Per-entry error isolation: one undecryptable entry skips that entry
// alone; a missing credentials.json yields an empty result with nil error
// (a fresh machine is simply not logged in); an undecodable store yields
// an empty result with a coded wrapped error. The reader never panics and
// never hangs.
//
// Hygiene (T-5-17 inheritance, test-locked): decrypted values exist only
// in reply-construction scope; log statements carry only entry-key names,
// shapes, and lengths — never a value.

// EncV1Prefix is the legacy credential-blob layout prefix
// (enc:v1:<iv>.<tag>.<ct>; segments base64url — the same encoding choice
// as the v2 loader's established RawURLEncoding discipline).
const EncV1Prefix = "enc:v1:"

// encPrefix is the encrypted-value family prefix: any value carrying it is
// encrypted under SOME layout, so only the handled layouts below may ever
// decrypt it — every other member skips (never plaintext).
const encPrefix = "enc:"

// The store-level structural-log codes (the skip arms plus the whole-store
// failures — short coded tokens, never messages, never values).
const (
	storeCodeUnhandledEncPrefix = "unhandled-enc-prefix"
	storeCodeUnreadable         = "store-unreadable"
	storeCodeUndecodable        = "store-undecodable"
)

// The store entry keys the chain consumes (RESEARCH Pattern 1) and the
// Bearer form the login token carries (ref project-token.ts:71-83).
const (
	oauthAccessTokenKeyPrefix = "oauth:"
	oauthAccessTokenKeySuffix = ":access_token"
	zcodeJWTEntryKey          = "zcodejwttoken"
	bearerPrefix              = "Bearer "
)

// v1SecretEnvKey is the env key the enc:v1 fallback resolves first and
// v1SecretFallbackPrefix the machine-string fallback's prefix (ref
// credential-store.ts:33-43 — the trimmed env wins, else the machine).
const (
	v1SecretEnvKey         = "ZCODE_CREDENTIAL_SECRET"
	v1SecretFallbackPrefix = "zcode-credential-fallback:"
)

// The store reader's coded posture: a missing file is NOT an error (a
// fresh machine is simply not logged in); an undecodable store answers
// the coded wrapped error; any other read failure answers the coded
// unreadable error. Consumers match the sentinel, never the message (the
// package's coded-decline discipline).
var (
	ErrStoreUndecodable = errors.New("credential store is not a decodable JSON object")
	ErrStoreUnreadable  = errors.New("credential store is unreadable")
)

// errV1SecretInputs is the loud-validation sentinel: an unresolvable
// machine input (home, username) never derives a silently-empty machine
// string — that would hash the wrong key and fail GCM auth with no cue.
var errV1SecretInputs = errors.New("enc:v1 fallback machine inputs are unresolved")

// CredentialStore is the D-01/D-03 reader service. Constructed once via
// NewCredentialStore; Entries is safe for concurrent use (the v2 loader's
// exec contract is the only shared resource and is bounded by the ctx).
type CredentialStore struct {
	path string
	v2   KeyLoader
	// decrypt composes the existing v2 seam service (parse → fp24 verify →
	// GCM open) — the store never forks the layout parser (D-01: one seam).
	decrypt *CredentialDecrypt
	// v1 is the legacy fallback key source (nil = enc:v1 entries skip with
	// a structural log — an unavailable fallback must not sink the store).
	v1 KeyLoader
}

// NewCredentialStore returns the reader over the credentials.json path and
// the injectable v2 keyring loader (D-01: the seam is the only keyring
// path; the v2:os arm composes the existing CredentialDecrypt service).
func NewCredentialStore(path string, v2 KeyLoader) *CredentialStore {
	return &CredentialStore{
		path:    path,
		v2:      v2,
		decrypt: NewCredentialDecrypt(v2),
		v1:      newStoreV1Loader(),
	}
}

// Entries reads and decrypts the whole store with per-entry error
// isolation: plaintext passes through verbatim, v1/v2 blobs decrypt, an
// unknown enc: prefix skips its entry (never plaintext), and one broken
// entry never sinks the rest. A missing file answers the empty-nil
// posture; an undecodable store answers the coded wrapped error.
func (s *CredentialStore) Entries(ctx context.Context) (map[string]string, error) {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// A fresh machine is simply not logged in — the honest empty
			// posture, never an error (AUTH-02 base posture).
			return map[string]string{}, nil
		}

		slog.Warn("interact: credential store unread", "path_len", len(s.path),
			"code", storeCodeUnreadable)

		return map[string]string{}, fmt.Errorf("%w: %w", ErrStoreUnreadable, err)
	}

	var store map[string]string
	if err := json.Unmarshal(raw, &store); err != nil {
		slog.Warn("interact: credential store undecodable", "store_len", len(raw),
			"code", storeCodeUndecodable)

		return map[string]string{}, fmt.Errorf("%w: %w", ErrStoreUndecodable, err)
	}

	out := make(map[string]string, len(store))

	for entryKey, value := range store {
		switch {
		case strings.HasPrefix(value, EncV2OSPrefix):
			decrypted, fail := s.decrypt.decryptOne(ctx, value)
			if fail != nil {
				slog.Warn("interact: credential store entry skipped",
					"key", entryKey, "value_len", len(value), "code", fail.code)

				continue
			}

			out[entryKey] = decrypted

		case strings.HasPrefix(value, EncV1Prefix):
			decrypted, fail := s.decryptV1(ctx, value)
			if fail != nil {
				slog.Warn("interact: credential store entry skipped",
					"key", entryKey, "value_len", len(value), "code", fail.code)

				continue
			}

			out[entryKey] = decrypted

		case hasUnhandledEncPrefix(value):
			// The never-plaintext rule: an unknown enc: layout must never
			// degrade to plaintext (the upstream v2-ENV failure mode).
			slog.Warn("interact: credential store entry skipped (unhandled enc prefix)",
				"key", entryKey, "value_len", len(value), "code", storeCodeUnhandledEncPrefix)

		default:
			out[entryKey] = value
		}
	}

	return out, nil
}

// LoginToken returns the family's oauth access token with exactly one
// leading "Bearer " ASCII prefix stripped (ref project-token.ts:71-83);
// a value without the prefix passes unchanged. A missing entry answers
// ok=false — the coded login-required arm is the consumer's (06-02).
func (s *CredentialStore) LoginToken(ctx context.Context, family string) (string, bool) {
	value, ok := s.entry(ctx, oauthAccessTokenKeyPrefix+family+oauthAccessTokenKeySuffix)
	if !ok {
		return "", false
	}

	return strings.TrimPrefix(value, bearerPrefix), true
}

// ZcodeJWT returns the zcodejwttoken entry — the ZCode JWT the chain
// folds into accountScope (ref project-token.ts:76,113-118). A missing
// entry answers ok=false.
func (s *CredentialStore) ZcodeJWT(ctx context.Context) (string, bool) {
	return s.entry(ctx, zcodeJWTEntryKey)
}

// entry reads one decrypted entry through the whole-store read (the
// accessors share the per-entry isolation posture).
func (s *CredentialStore) entry(ctx context.Context, key string) (string, bool) {
	entries, err := s.Entries(ctx)
	if err != nil {
		return "", false
	}

	value, ok := entries[key]

	return value, ok
}

// decryptV1 runs one legacy decrypt: the fallback key (pure compute — the
// derivation performs no exec, unlike the v2 keyring read) → layout parse
// → stdlib AES-256-GCM open. Every failure renders as a coded skip in the
// dispatch ladder; the plaintext exists only in the reply-construction
// scope of the caller.
func (s *CredentialStore) decryptV1(ctx context.Context, value string) (string, *cipherFailure) {
	if s.v1 == nil {
		return "", &cipherFailure{code: CipherCodeKeyUnavailable,
			message: "no enc:v1 fallback key configured"}
	}

	key, err := s.v1.LoadKey(ctx)
	if err != nil {
		return "", &cipherFailure{code: CipherCodeKeyUnavailable,
			message: "the enc:v1 fallback key is unavailable"}
	}

	if len(key) != loaderKeyLen {
		return "", &cipherFailure{code: CipherCodeKeyUnavailable,
			message: "the enc:v1 fallback key has the wrong length"}
	}

	return openEncV1(key, value)
}

// parseEncV1 parses the legacy layout: the enc:v1 prefix (guaranteed by
// the dispatch ladder), then the exactly-3 base64url dot-separated
// iv.tag.ct segments. Malformed shapes answer the coded invalid-value
// failure — never a panic, never a partial decrypt.
func parseEncV1(value string) (iv, tag, ct []byte, fail *cipherFailure) {
	segs := strings.Split(strings.TrimPrefix(value, EncV1Prefix), ".")
	if len(segs) != cipherSegmentCount {
		return nil, nil, nil, &cipherFailure{code: CipherCodeInvalidValue,
			message: "credential blob layout is malformed (want iv.tag.ct)"}
	}

	iv, errIV := base64.RawURLEncoding.DecodeString(segs[0])
	tag, errTag := base64.RawURLEncoding.DecodeString(segs[1])
	ct, errCT := base64.RawURLEncoding.DecodeString(segs[2])

	if errIV != nil || errTag != nil || errCT != nil {
		return nil, nil, nil, &cipherFailure{code: CipherCodeInvalidValue,
			message: "credential blob segments are not base64url"}
	}

	return iv, tag, ct, nil
}

// openEncV1 decrypts one legacy blob over the sha256(secret) key: parse →
// stdlib AES-256-GCM open (the same primitive the v2 engine uses). A GCM
// auth failure answers the coded generic message — the crypto error text
// never carries material.
func openEncV1(key []byte, value string) (string, *cipherFailure) {
	iv, tag, ct, fail := parseEncV1(value)
	if fail != nil {
		return "", fail
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", &cipherFailure{code: CipherCodeDecryptFailed,
			message: "credential cipher cannot be constructed"}
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", &cipherFailure{code: CipherCodeDecryptFailed,
			message: "credential cipher cannot be constructed"}
	}

	sealed := make([]byte, 0, len(ct)+len(tag))
	sealed = append(sealed, ct...)
	sealed = append(sealed, tag...)

	plaintext, err := gcm.Open(nil, iv, sealed, nil)
	if err != nil {
		return "", &cipherFailure{code: CipherCodeDecryptFailed,
			message: "credential blob failed AES-256-GCM authentication"}
	}

	return string(plaintext), nil
}

// hasUnhandledEncPrefix is the never-plaintext rule (RESEARCH Pattern 1,
// ref credential-store.ts:166-175): a value in the enc: family that is
// neither handled layout must never pass through as plaintext — the entry
// skips instead.
func hasUnhandledEncPrefix(value string) bool {
	if !strings.HasPrefix(value, encPrefix) {
		return false
	}

	return !strings.HasPrefix(value, EncV1Prefix) && !strings.HasPrefix(value, EncV2OSPrefix)
}

// storeV1Loader is the store's internal enc:v1 key source (D-03): pure
// compute — no exec, unlike the v2 keyring read. The secret resolves
// env-then-machine-string at load time; the key is sha256(secret).
type storeV1Loader struct{}

// newStoreV1Loader returns the internal fallback key source (never nil —
// the loud validation lives at load time so a malformed machine never
// pre-degrades every enc:v1 entry silently).
func newStoreV1Loader() KeyLoader {
	return storeV1Loader{}
}

// LoadKey derives the fallback key: resolve the secret, hash it. The
// machine inputs validate loud — an unresolvable home or username is the
// sentinel error, never a silently-empty machine string (that would hash
// the wrong key and fail every GCM open with no cue).
func (storeV1Loader) LoadKey(_ context.Context) ([]byte, error) {
	secret, err := resolveStoreV1Secret()
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256([]byte(secret))

	return sum[:], nil
}

// resolveStoreV1Secret resolves the enc:v1 secret (RESEARCH Pattern 1,
// ref credential-store.ts:33-43): the trimmed ZCODE_CREDENTIAL_SECRET env
// value wins when set and non-blank; otherwise the machine-bound fallback
// string over GOOS/home/username.
func resolveStoreV1Secret() (string, error) {
	if trimmed := strings.TrimSpace(os.Getenv(v1SecretEnvKey)); trimmed != "" {
		return trimmed, nil
	}

	home, herr := os.UserHomeDir()
	if herr != nil || home == "" {
		return "", fmt.Errorf("%w: home directory unresolved", errV1SecretInputs)
	}

	username := ""

	if u, uerr := user.Current(); uerr == nil {
		username = u.Username
	}

	if username == "" {
		return "", fmt.Errorf("%w: username unresolved", errV1SecretInputs)
	}

	return v1SecretFallbackPrefix + runtime.GOOS + ":" + home + ":" + username, nil
}
