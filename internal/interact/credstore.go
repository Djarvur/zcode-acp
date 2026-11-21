package interact

import (
	"context"
	"errors"
)

// The store reader's coded posture: a missing file is NOT an error (a
// fresh machine is simply not logged in); an undecodable store answers
// the coded wrapped error. Consumers match the sentinel, never the
// message (the package's coded-decline discipline).
var ErrStoreUndecodable = errors.New("credential store is not a decodable JSON object")

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
type CredentialStore struct {
	path string
	v2   KeyLoader
}

// NewCredentialStore returns the reader over the credentials.json path and
// the injectable v2 keyring loader (D-01: the seam is the only keyring
// path; the v2:os arm composes the existing CredentialDecrypt service).
func NewCredentialStore(path string, v2 KeyLoader) *CredentialStore {
	return &CredentialStore{path: path, v2: v2}
}

// Entries reads and decrypts the whole store with per-entry error
// isolation: plaintext passes through verbatim, v1/v2 blobs decrypt, an
// unknown enc: prefix skips its entry, one broken entry never sinks the
// rest.
func (s *CredentialStore) Entries(_ context.Context) (map[string]string, error) {
	return map[string]string{}, nil // RED skeleton (06-01): the dispatch ladder lands in GREEN
}

// LoginToken returns the family's oauth access token with exactly one
// leading "Bearer " ASCII prefix stripped (ref project-token.ts:71-83);
// a value without the prefix passes unchanged. A missing entry answers
// ok=false — the coded login-required arm is the consumer's (06-02).
func (s *CredentialStore) LoginToken(_ context.Context, _ string) (string, bool) {
	return "", false // RED skeleton (06-01)
}

// ZcodeJWT returns the zcodejwttoken entry — the ZCode JWT the chain
// folds into accountScope (ref project-token.ts:76,113-118). A missing
// entry answers ok=false.
func (s *CredentialStore) ZcodeJWT(_ context.Context) (string, bool) {
	return "", false // RED skeleton (06-01)
}
