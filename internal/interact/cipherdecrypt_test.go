package interact_test

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// D-06 CredentialDecrypt seam contract locks (plan 05-07): the decrypt
// engine over the field-verified enc:v2:os layout, the encrypt decline, the
// honest loader-failure degradation, and the hygiene locks (no key material
// on the loader argv, scrubbed subprocess env, no key/plaintext in any log
// line). Every vector is SYNTHETIC — the key is generated in-test and the
// ciphertext blobs are sealed in-test with stdlib AES-256-GCM; no real
// keyring material exists anywhere in this package (scan-secrets green).

// cipherTestTimeout bounds every arm — a seam that stalls (the T-5-19
// regression) fails loud instead of hanging the suite.
const cipherTestTimeout = 2 * time.Second

// syntheticCipherPlaintext is the in-test plaintext payload (fixture data —
// never a real credential; its synthetic marker stays under the scan gate's
// 20-char token-value class the way every corpus fixture does).
const syntheticCipherPlaintext = `{"token":"zc-fix-not-real","scopes":["read"]}`

// cipherFixRequestID is the arms' fixed params.requestId (the reply echo
// assertion's expected value).
const cipherFixRequestID = "cred-cipher-fix1"

// errKeyringLocked is the test loader's failure sentinel (the D-06
// keyring-locked degradation arm).
var errKeyringLocked = errors.New("secret-tool lookup failed: keyring locked")

// stubLoader is the in-test KeyLoader double: a fixed key or a fixed error.
type stubLoader struct {
	key []byte
	err error
}

// LoadKey replays the canned key or error.
func (l *stubLoader) LoadKey(_ context.Context) ([]byte, error) {
	return l.key, l.err
}

// randomTestKey generates the arm's synthetic 32-byte key.
func randomTestKey(t *testing.T) []byte {
	t.Helper()

	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		t.Fatalf("generate synthetic key: %v", err)
	}

	return key
}

// kidFor renders the field-verified kid: the first 24 hex chars of
// sha256(key).
func kidFor(key []byte) string {
	sum := sha256.Sum256(key)

	return hex.EncodeToString(sum[:])[:24]
}

// cipherSha256 hashes one string (the shared hash helper the argv
// assertions use for the username derivation).
func cipherSha256(s string) []byte {
	sum := sha256.Sum256([]byte(s))

	return sum[:]
}

// sealTestBlob builds an enc:v2:os credential blob in the field-verified
// layout (prefix, kid, base64url iv.tag.ct) over the synthetic key. The
// layout's iv carries 12 GCM nonce bytes; the sealed output is split into
// ciphertext and tag segments exactly as the layout demands.
func sealTestBlob(t *testing.T, key []byte, plaintext string) string {
	t.Helper()

	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("aes cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}

	iv := make([]byte, gcm.NonceSize())

	if _, err := rand.Read(iv); err != nil {
		t.Fatalf("generate iv: %v", err)
	}

	sealed := gcm.Seal(nil, iv, []byte(plaintext), nil)
	split := len(sealed) - gcm.Overhead()
	ct, tag := sealed[:split], sealed[split:]

	return interact.EncV2OSPrefix + kidFor(key) + ":" +
		base64.RawURLEncoding.EncodeToString(iv) + "." +
		base64.RawURLEncoding.EncodeToString(tag) + "." +
		base64.RawURLEncoding.EncodeToString(ct)
}

// tamperCiphertext flips one bit inside the blob's ciphertext segment (the
// GCM auth failure vector).
func tamperCiphertext(t *testing.T, blob string) string {
	t.Helper()

	idx := strings.LastIndex(blob, ".")
	if idx < 0 {
		t.Fatal("blob carries no ct segment")
	}

	head, ctB64 := blob[:idx+1], blob[idx+1:]

	ct, err := base64.RawURLEncoding.DecodeString(ctB64)
	if err != nil || len(ct) == 0 {
		t.Fatalf("decode ct segment: %v", err)
	}

	ct[0] ^= 0x01

	return head + base64.RawURLEncoding.EncodeToString(ct)
}

// cipherEnvelope builds one interaction/requestCredentialCipher ask over
// the given operation and value.
func cipherEnvelope(t *testing.T, operation, value string) (interact.Request, zcode.Envelope) {
	t.Helper()

	params, err := json.Marshal(map[string]any{
		"requestId": cipherFixRequestID,
		"operation": operation,
		"value":     value,
	})
	if err != nil {
		t.Fatalf("marshal cipher params: %v", err)
	}

	req := interact.Request{
		Method: zcode.MethodRequestCredentialCipher,
		ID:     &zcode.ID{Str: "server-cipher-1"},
		Params: params,
		ReqID:  cipherFixRequestID,
	}

	return req, zcode.Envelope{ID: req.ID, Method: req.Method, Params: params}
}

// serveCipherAnswer runs one ask through the service and decodes the reply
// result into its wire map (nil-safe).
func serveCipherAnswer(
	t *testing.T, svc *interact.CredentialDecrypt, operation, value string,
) map[string]any {
	t.Helper()

	req, _ := cipherEnvelope(t, operation, value)

	ctx, cancel := context.WithTimeout(context.Background(), cipherTestTimeout)
	defer cancel()

	ans, err := svc.Handle(ctx, req)
	if err != nil {
		t.Fatalf("Handle returned a handler error (%v) — every arm answers inside the result union", err)
	}

	wire, err := json.Marshal(ans.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(wire, &got); err != nil {
		t.Fatalf("decode result %s: %v", wire, err)
	}

	assertCipherReplyShape(t, got)

	return got
}

// assertCipherReplyShape locks the field-verified reply envelope: exactly
// {requestId, result}; result is the ok-discriminated union —
// {ok:true, value} or {ok:false, error:{code, message}} — never both arms,
// never extra keys (the server result-schema-parses every reply, Pitfall 4).
func assertCipherReplyShape(t *testing.T, got map[string]any) {
	t.Helper()

	if got["requestId"] != cipherFixRequestID {
		t.Fatalf("reply requestId = %v, want the params.requestId echo", got["requestId"])
	}

	if len(got) != 2 {
		t.Fatalf("reply keys = %v, want exactly [requestId result]", keysOfCipher(got))
	}

	result, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatalf("result = %v, want the ok-discriminated union object", got["result"])
	}

	okFlag, present := result["ok"]
	if !present {
		t.Fatalf("result keys = %v, want the ok discriminant", keysOfCipher(result))
	}

	switch okFlag {
	case true:
		if len(result) != 2 || result["value"] == nil {
			t.Fatalf("ok:true union = %v, want exactly [ok value]", keysOfCipher(result))
		}
	case false:
		errObj, isMap := result["error"].(map[string]any)
		if !isMap || len(result) != 2 {
			t.Fatalf("ok:false union = %v, want exactly [ok error]", keysOfCipher(result))
		}

		if errObj["code"] == "" || errObj["message"] == "" || len(errObj) != 2 {
			t.Fatalf("error object = %v, want exactly [code message] with non-empty values", errObj)
		}
	default:
		t.Fatalf("ok discriminant = %v, want a boolean", okFlag)
	}
}

// cipherDeclineOf digs the ok:false arm's error object out of a decoded
// reply (failing the test when the reply is not the decline arm).
func cipherDeclineOf(t *testing.T, got map[string]any) map[string]any {
	t.Helper()

	if okFlag, _ := got["result"].(map[string]any)["ok"].(bool); okFlag {
		t.Fatalf("arm answered ok:true — a decline was required: %v", got["result"])
	}

	errObj, _ := got["result"].(map[string]any)["error"].(map[string]any)

	return errObj
}

func keysOfCipher(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

// TestCredentialDecryptDecrypt locks the success engine: a well-formed
// synthetic blob decrypts to the exact plaintext through the injected
// loader's key (the field-verified layout round-trips; the reply carries
// the discriminated-union success arm).
func TestCredentialDecryptDecrypt(t *testing.T) {
	key := randomTestKey(t)
	svc := interact.NewCredentialDecrypt(&stubLoader{key: key})

	got := serveCipherAnswer(t, svc, interact.CipherOpDecrypt, sealTestBlob(t, key, syntheticCipherPlaintext))

	result, isMap := got["result"].(map[string]any)
	if !isMap {
		t.Fatalf("result = %v, want the union object", got["result"])
	}

	if okFlag, _ := result["ok"].(bool); !okFlag {
		t.Fatalf("decrypt arm answered the decline %v — want the {ok:true, value} success arm", result)
	}

	if result["value"] != syntheticCipherPlaintext {
		t.Fatalf("decrypted value = %v, want the sealed plaintext verbatim", result["value"])
	}
}

// TestCredentialDecryptTamper locks the GCM auth-failure arm: a bit-flipped
// ciphertext answers the coded decrypt-failed decline — never a panic,
// never a fabricated value (V6: stdlib GCM is the integrity contract).
func TestCredentialDecryptTamper(t *testing.T) {
	key := randomTestKey(t)
	svc := interact.NewCredentialDecrypt(&stubLoader{key: key})

	blob := tamperCiphertext(t, sealTestBlob(t, key, syntheticCipherPlaintext))

	errObj := cipherDeclineOf(t, serveCipherAnswer(t, svc, interact.CipherOpDecrypt, blob))

	if errObj["code"] != interact.CipherCodeDecryptFailed {
		t.Fatalf("tampered blob answered code %v, want %q", errObj["code"], interact.CipherCodeDecryptFailed)
	}
}

// TestCredentialDecryptEncryptDeclines locks the encrypt decline (the
// D-06 never-fabricate posture): an encrypt op answers the coded
// unsupported-operation decline naming the operation — the wrapper never
// encrypts keyring material.
func TestCredentialDecryptEncryptDeclines(t *testing.T) {
	key := randomTestKey(t)
	svc := interact.NewCredentialDecrypt(&stubLoader{key: key})

	errObj := cipherDeclineOf(t, serveCipherAnswer(
		t, svc, interact.CipherOpEncrypt, sealTestBlob(t, key, syntheticCipherPlaintext)))

	if errObj["code"] != interact.CipherCodeUnsupportedOperation {
		t.Fatalf("encrypt answered code %v, want %q", errObj["code"], interact.CipherCodeUnsupportedOperation)
	}

	if msg, _ := errObj["message"].(string); !strings.Contains(msg, interact.CipherOpEncrypt) {
		t.Fatalf("decline message %q must name the unsupported operation encrypt", msg)
	}
}

// TestCredentialDecryptLoaderFailure locks the D-06 degradation: a failing
// loader (keyring locked) yields the honest key-unavailable decline inside
// the bounded window — never a stall, never a fabricated success (T-5-19).
func TestCredentialDecryptLoaderFailure(t *testing.T) {
	key := randomTestKey(t)
	svc := interact.NewCredentialDecrypt(&stubLoader{err: errKeyringLocked})

	blob := sealTestBlob(t, key, syntheticCipherPlaintext)

	started := time.Now()
	errObj := cipherDeclineOf(t, serveCipherAnswer(t, svc, interact.CipherOpDecrypt, blob))

	if elapsed := time.Since(started); elapsed > cipherTestTimeout {
		t.Fatalf("loader failure answered after %v — the degradation must never stall", elapsed)
	}

	if errObj["code"] != interact.CipherCodeKeyUnavailable {
		t.Fatalf("loader failure answered code %v, want %q", errObj["code"], interact.CipherCodeKeyUnavailable)
	}
}

// TestCredentialDecryptKeyMismatch locks the kid verification: a blob
// keyed by a different key answers the coded key-mismatch decline (the kid
// is first-24-hex of sha256(key) — the loader's key must be THE key).
func TestCredentialDecryptKeyMismatch(t *testing.T) {
	sealed := randomTestKey(t)
	loaded := randomTestKey(t)
	svc := interact.NewCredentialDecrypt(&stubLoader{key: loaded})

	blob := sealTestBlob(t, sealed, syntheticCipherPlaintext)

	errObj := cipherDeclineOf(t, serveCipherAnswer(t, svc, interact.CipherOpDecrypt, blob))

	if errObj["code"] != interact.CipherCodeKeyMismatch {
		t.Fatalf("kid mismatch answered code %v, want %q", errObj["code"], interact.CipherCodeKeyMismatch)
	}
}

// TestCredentialDecryptMalformedValue locks the layout parser's coded
// negative arms: every malformed shape answers invalid-value — never a
// panic, never a fabricated arm.
func TestCredentialDecryptMalformedValue(t *testing.T) {
	key := randomTestKey(t)
	svc := interact.NewCredentialDecrypt(&stubLoader{key: key})

	wellFormed := sealTestBlob(t, key, syntheticCipherPlaintext)

	wrongSegCount := interact.EncV2OSPrefix + kidFor(key) + ":" +
		base64.RawURLEncoding.EncodeToString([]byte("only-two-segments.here"))

	badBase64 := interact.EncV2OSPrefix + kidFor(key) + ":!!!.???.*!*"

	tests := []struct {
		name  string
		value string
	}{
		{"no prefix", "deadbeef-decrypt-me"},
		{"empty value", ""},
		{"no kid separator", interact.EncV2OSPrefix + "nocolonhere"},
		{"wrong segment count", wrongSegCount},
		{"non-base64url segment", badBase64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errObj := cipherDeclineOf(t, serveCipherAnswer(t, svc, interact.CipherOpDecrypt, tt.value))

			if errObj["code"] != interact.CipherCodeInvalidValue {
				t.Fatalf("malformed value answered code %v, want %q", errObj["code"], interact.CipherCodeInvalidValue)
			}
		})
	}

	// The well-formed control stays green behind the negatives: the decline
	// arms exist for broken shapes, not for the valid one.
	got := serveCipherAnswer(t, svc, interact.CipherOpDecrypt, wellFormed)

	if okFlag, _ := got["result"].(map[string]any)["ok"].(bool); !okFlag {
		t.Fatalf("well-formed control answered the decline %v — the parser broke", got["result"])
	}
}

// TestCredentialDecryptLogHygiene is the T-5-17 lock: driving a full
// decrypt through the service emits NO log line carrying key bytes or
// plaintext — every credential field rides as shapes and lengths. The
// capture asserts absence across the base64url/base64std/hex encodings of
// the key and the plaintext itself.
func TestCredentialDecryptLogHygiene(t *testing.T) {
	key := randomTestKey(t)

	var buf bytes.Buffer

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	svc := interact.NewCredentialDecrypt(&stubLoader{key: key})

	got := serveCipherAnswer(t, svc, interact.CipherOpDecrypt, sealTestBlob(t, key, syntheticCipherPlaintext))

	if okFlag, _ := got["result"].(map[string]any)["ok"].(bool); !okFlag {
		t.Fatalf("hygiene drive needs the success arm, got %v", got["result"])
	}

	logged := buf.String()
	if logged == "" {
		t.Fatal("the seam answered no structural log line " +
			"(the answer must be observably recorded: shapes/lengths only)")
	}

	forbidden := []string{
		syntheticCipherPlaintext,
		base64.RawURLEncoding.EncodeToString(key),
		base64.StdEncoding.EncodeToString(key),
		hex.EncodeToString(key),
	}

	for _, needle := range forbidden {
		if strings.Contains(logged, needle) {
			t.Fatalf("log output carries key/plaintext material (%s…): %s", needle[:8], logged)
		}
	}
}

// TestSecretToolLoaderHygiene locks the production loader's subprocess
// contract (T-5-17): the argv carries the field-verified lookup shape with
// the username derived from the RESOLVED credentials path (never key
// material — the loader holds no key yet), and the env is exactly the
// caller-provided scrubbed set.
func TestSecretToolLoaderHygiene(t *testing.T) {
	target := filepath.Join(t.TempDir(), "v2", "credentials.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	link := filepath.Join(t.TempDir(), "link-credentials.json")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	scrubbed := []string{"PATH=/usr/bin", "HOME=/home/u"}
	loader := interact.NewSecretToolKeyLoader("secret-tool", link, scrubbed)

	argv, env, err := loader.LookupCommand()
	if err != nil {
		t.Fatalf("LookupCommand: %v", err)
	}

	if len(argv) < 2 || argv[0] != "secret-tool" || argv[1] != "lookup" {
		t.Fatalf("argv = %v, want the secret-tool lookup invocation", argv)
	}

	joined := strings.Join(argv, " ")
	if !strings.Contains(joined, "service") || !strings.Contains(joined, "ZCode Credential Encryption") {
		t.Fatalf("argv %q lacks the field-verified service attribute", joined)
	}

	// The username derivation hashes the RESOLVED (realpath) path string:
	// the symlinked fixture must hash the target, not the link.
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("resolve fixture: %v", err)
	}

	wantUsername := hex.EncodeToString(cipherSha256(resolved))
	wantArgv := "secret-tool lookup service ZCode Credential Encryption username " + wantUsername

	if joined != wantArgv {
		t.Fatalf("argv = %q, want the exact field-verified lookup shape %q", joined, wantArgv)
	}

	if len(env) != len(scrubbed) {
		t.Fatalf("env = %v, want exactly the caller-provided scrubbed set %v", env, scrubbed)
	}

	for i, kv := range scrubbed {
		if env[i] != kv {
			t.Fatalf("env[%d] = %q, want %q (the scrubbed set passes through verbatim)", i, env[i], kv)
		}
	}
}

// TestSecretToolLoaderEmptyPath locks the loader's fail-loud input
// validation: an empty credentials path answers an error at both the
// command build and the load — never a phantom-path lookup.
func TestSecretToolLoaderEmptyPath(t *testing.T) {
	loader := interact.NewSecretToolKeyLoader("secret-tool", "", nil)

	if _, _, err := loader.LookupCommand(); err == nil {
		t.Fatal("empty credentials path must fail loud, got nil error")
	}

	if _, err := loader.LoadKey(context.Background()); err == nil {
		t.Fatal("empty credentials path must fail loud at LoadKey, got nil error")
	}
}

// TestResponderCredentialCipherSeam locks the D-06 injection at the
// responder surface: with the seam injected the cipher ask routes through
// the policy row (handled, the decrypted answer rides the reply); unwired,
// the D-05 tracked-no-answer fall-through stays exactly as Phase 3 locked
// it (handled=false, zero reply bytes).
func TestResponderCredentialCipherSeam(t *testing.T) {
	key := randomTestKey(t)
	blob := sealTestBlob(t, key, syntheticCipherPlaintext)

	t.Run("unwired falls through to the tracked-no-answer posture", func(t *testing.T) {
		_, env := cipherEnvelope(t, interact.CipherOpDecrypt, blob)

		_, _, handled := interact.NewResponder().Serve(context.Background(), env)
		if handled {
			t.Fatal("unwired responder must fall the cipher ask through (handled=false), got handled=true")
		}
	})

	t.Run("injected seam answers the decrypt through the policy row", func(t *testing.T) {
		svc := interact.NewCredentialDecrypt(&stubLoader{key: key})
		responder := interact.NewResponder(interact.WithCredentialDecrypt(svc))

		_, env := cipherEnvelope(t, interact.CipherOpDecrypt, blob)

		result, werr, handled := responder.Serve(context.Background(), env)
		if !handled {
			t.Fatal("injected seam must handle the cipher ask, got handled=false (the turn would stall)")
		}

		if werr != nil {
			t.Fatalf("decrypt answered a wire error: %v — the decline rides the result union", werr)
		}

		wire, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal result: %v", err)
		}

		var got map[string]any
		if err := json.Unmarshal(wire, &got); err != nil {
			t.Fatalf("decode result %s: %v", wire, err)
		}

		assertCipherReplyShape(t, got)

		resultMap, isMap := got["result"].(map[string]any)
		if !isMap {
			t.Fatalf("result = %v, want the union object", got["result"])
		}

		if okFlag, _ := resultMap["ok"].(bool); !okFlag || resultMap["value"] != syntheticCipherPlaintext {
			t.Fatalf("wired reply = %v, want the {ok:true, value} success arm", resultMap)
		}
	})
}
