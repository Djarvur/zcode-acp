package interact

import (
	"context"
	"errors"

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
// values), never on the loader subprocess's argv, and the subprocess runs
// on the caller-provided scrubbed env with its stdout captured and drained.

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

// Hygiene sentinels — wrapped with %w context at the failure site, never
// constructed dynamically (err113).
var (
	errEmptyCredentialsPath = errors.New("credentials path is empty")
	errLoaderKeyShape       = errors.New("keyring key is not 32 bytes of base64url")
	errCipherNotAssembled   = errors.New("credential decrypt unavailable: the seam is not assembled in this build")
)

// KeyLoader loads the 32-byte login-keyring credential key. The production
// implementation shells the field-verified secret-tool lookup
// (SecretToolKeyLoader); Phase 6's godbus vault implements the same seam —
// the interface is the upgrade boundary (D-06 recommend-then-upgrade
// posture).
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
// interact.Handler signature). The reply result is the field-verified
// discriminated union {requestId, result:{ok...}}.
//
// Skeleton posture (RED floor): the seam answers the honest decline for
// every ask — never a fabricated success — while the layout engine, loader
// and hygiene contract land in GREEN.
func (s *CredentialDecrypt) Handle(_ context.Context, req Request) (Answer, error) {
	return Answer{Result: cipherDecline(req, CipherCodeInvalidValue, errCipherNotAssembled.Error())}, nil
}

// cipherDecline renders the {ok:false} arm: the requestId echo plus the
// coded error union, keyed verbatim per the field-verified reply schema
// (credentialCipherReplySchema = {requestId, result: discriminatedUnion("ok")}).
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
// caller-provided scrubbed env (ChildEnv-style allowlist posture — no key
// material can cross; the argv carries the path derivation, never key
// bytes), captures the stdout (the 43-char base64url key), and releases it
// into the reply scope. Real tier only: CI never constructs nor executes it
// against a keyring (D-06).
type SecretToolKeyLoader struct {
	bin      string
	credPath string
	env      []string
}

// NewSecretToolKeyLoader returns the loader over the secret-tool binary
// name, the desktop credentials.json path (resolved at lookup time), and
// the scrubbed subprocess env.
func NewSecretToolKeyLoader(bin, credentialsPath string, env []string) *SecretToolKeyLoader {
	return &SecretToolKeyLoader{bin: bin, credPath: credentialsPath, env: env}
}

// LookupCommand builds the lookup subprocess contract the loader executes:
// the argv (secret-tool lookup service <name> username <hash>) and the env
// it runs under. Exported for the hygiene locks — tests inspect the
// contract without executing the binary. The username argument is the
// sha256 hex derivation over the RESOLVED credentials path (EvalSymlinks +
// Abs — the realpath discipline), derived from a PATH STRING, never key
// material.
//
// Skeleton posture (RED floor): returns the not-assembled error.
func (l *SecretToolKeyLoader) LookupCommand() ([]string, []string, error) {
	return nil, nil, errEmptyCredentialsPath
}

// LoadKey shells the lookup and decodes the captured stdout: trimmed
// base64url, exactly 32 bytes. The captured stderr rides the exec error
// and is never logged (hygiene: the key never appears in any log path).
//
// Skeleton posture (RED floor): returns the not-assembled error.
func (l *SecretToolKeyLoader) LoadKey(_ context.Context) ([]byte, error) {
	return nil, errLoaderKeyShape
}
