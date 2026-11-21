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

	"github.com/Djarvur/zcode-acp/internal/interact"
)

// AUTH-02 credential-store base locks (plan 06-01): the read-only reader
// over the desktop's ~/.zcode/v2/credentials.json — the plaintext /
// enc:v1 / enc:v2:os dispatch ladder with per-entry error isolation, the
// unknown-enc: never-plaintext skip (the upstream v2-ENV degradation is
// the locked failure mode), the missing-file empty-nil posture, and the
// chain's two accessors (the Bearer-strip and the JWT entry). Every
// vector is SYNTHETIC — the store file, secrets, keys and blobs are
// produced in-test over a t.TempDir(); no real credential material exists
// anywhere in this package (scan-secrets green).

// v1TestSecret is the in-test enc:v1 secret (fixture marker, never a real
// credential); the store resolves it through the trimmed
// ZCODE_CREDENTIAL_SECRET env seam.
const v1TestSecret = "zc-fix-v1-secret"

// Synthetic store payloads (fixture markers, all under the scan gate's
// 20-char token-value class).
const (
	fixAccessToken = "zc-fix-oauth-tok"
	fixBearerless  = "zc-fix-bearerless"
	fixJWT         = "zc-fix-jwt-payload"
	fixRefresh     = "zc-fix-refresh-val"
	fixIDToken     = "zc-fix-idtok"
)

// sealV1TestBlob builds an enc:v1 credential blob in the legacy layout
// (prefix, exactly 3 base64url dot-separated iv.tag.ct segments) under the
// sha256(secret) key the v1 fallback derives — the in-test encrypt half of
// the round-trip vectors.
func sealV1TestBlob(t *testing.T, secret, plaintext string) string {
	t.Helper()

	sum := sha256.Sum256([]byte(secret))
	key := sum[:]

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

	return "enc:v1:" +
		base64.RawURLEncoding.EncodeToString(iv) + "." +
		base64.RawURLEncoding.EncodeToString(tag) + "." +
		base64.RawURLEncoding.EncodeToString(ct)
}

// writeFixtureStore marshals the entry map and writes the store file
// (0600 under 0750 dirs — the desktop's own file posture).
func writeFixtureStore(t *testing.T, path string, entries map[string]string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	raw, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal store: %v", err)
	}

	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write store: %v", err)
	}
}

// TestCredStore locks the read path end-to-end over one fixture store
// carrying every value class: plaintext passthrough verbatim, an enc:v1
// blob decrypting under the trimmed env secret, an enc:v2:os blob
// decrypting through the injected stub loader only after its fp24
// checksum matches, a mismatched-fp24 blob isolating to a per-entry skip,
// an unknown enc: prefix NEVER passing through as plaintext — and the
// chain's two accessors (the Bearer-strip and the JWT entry).
func TestCredStore(t *testing.T) {
	// The trimmed-env seam: the surrounding whitespace proves the trim
	// end-to-end (the v1 secret resolution order is env-then-machine).
	t.Setenv("ZCODE_CREDENTIAL_SECRET", "  "+v1TestSecret+"  ")

	sealKey := randomTestKey(t)
	otherKey := randomTestKey(t)

	storePath := filepath.Join(t.TempDir(), "v2", "credentials.json")
	writeFixtureStore(t, storePath, map[string]string{
		"oauth:zai:access_token":   "Bearer " + fixAccessToken,
		"oauth:other:access_token": fixBearerless,
		"zcodejwttoken":            sealV1TestBlob(t, v1TestSecret, fixJWT),
		"oauth:zai:refresh_token":  sealTestBlob(t, sealKey, fixRefresh),
		"oauth:zai:id_token":       sealTestBlob(t, otherKey, fixIDToken),
		"mystery":                  "enc:v3:os:AAAA.BBBB.CCCC",
	})

	store := interact.NewCredentialStore(storePath, &stubLoader{key: sealKey})

	ctx, cancel := context.WithTimeout(context.Background(), cipherTestTimeout)
	defer cancel()

	got, err := store.Entries(ctx)
	if err != nil {
		t.Fatalf("Entries returned %v — per-entry isolation keeps the read green", err)
	}

	// Plaintext passes through verbatim (the Bearer strip is the
	// accessor's job, never the reader's).
	if got["oauth:zai:access_token"] != "Bearer "+fixAccessToken {
		t.Fatalf("plaintext entry = %q, want the verbatim passthrough", got["oauth:zai:access_token"])
	}

	if got["oauth:other:access_token"] != fixBearerless {
		t.Fatalf("plaintext entry = %q, want the verbatim passthrough", got["oauth:other:access_token"])
	}

	// The enc:v1 blob decrypts under the trimmed test secret.
	if got["zcodejwttoken"] != fixJWT {
		t.Fatalf("enc:v1 entry = %q, want the decrypted %q", got["zcodejwttoken"], fixJWT)
	}

	// The enc:v2:os blob decrypts through the injected stub loader (the
	// fp24 checksum matches by construction).
	if got["oauth:zai:refresh_token"] != fixRefresh {
		t.Fatalf("enc:v2:os entry = %q, want the decrypted %q", got["oauth:zai:refresh_token"], fixRefresh)
	}

	// The fp24-mismatched blob isolates: skipped, matching entries intact.
	if _, present := got["oauth:zai:id_token"]; present {
		t.Fatalf("fp24-mismatched entry served %q — the checksum must skip the entry", got["oauth:zai:id_token"])
	}

	// The unknown enc: prefix NEVER passes through as plaintext.
	if _, present := got["mystery"]; present {
		t.Fatalf("unknown-enc: entry served %q — the never-plaintext rule is broken", got["mystery"])
	}

	if len(got) != 4 {
		t.Fatalf("Entries returned %d entries %v, want exactly the 4 handled ones", len(got), got)
	}

	// LoginToken strips exactly one leading "Bearer " ASCII prefix.
	tok, ok := store.LoginToken(ctx, "zai")
	if !ok || tok != fixAccessToken {
		t.Fatalf("LoginToken(zai) = (%q, %v), want the Bearer-stripped %q", tok, ok, fixAccessToken)
	}

	// A value without the prefix passes unchanged.
	tok, ok = store.LoginToken(ctx, "other")
	if !ok || tok != fixBearerless {
		t.Fatalf("LoginToken(other) = (%q, %v), want %q unchanged", tok, ok, fixBearerless)
	}

	// A missing family answers the ok=false posture (never a panic).
	if tok, ok = store.LoginToken(ctx, "absent"); ok || tok != "" {
		t.Fatalf("LoginToken(absent) = (%q, %v), want the ok=false posture", tok, ok)
	}

	// ZcodeJWT serves the decrypted zcodejwttoken entry.
	if jwt, ok := store.ZcodeJWT(ctx); !ok || jwt != fixJWT {
		t.Fatalf("ZcodeJWT() = (%q, %v), want (%q, true)", jwt, ok, fixJWT)
	}

	// Log hygiene (T-5-17 inheritance): the skips' structural lines carry
	// entry-key names, shapes, and lengths — never a decrypted value nor
	// key material. The fixture guarantees both skip arms fire on every
	// read (the fp24 mismatch and the unknown prefix).
	var buf bytes.Buffer

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	defer slog.SetDefault(prev)

	if _, err := store.Entries(ctx); err != nil {
		t.Fatalf("hygiene re-read: %v", err)
	}

	logged := buf.String()
	if logged == "" {
		t.Fatal("the store read emitted no structural log (the skip arms must be observable)")
	}

	if !strings.Contains(logged, "unhandled-enc-prefix") {
		t.Fatalf("the unknown-prefix skip line is missing its code: %s", logged)
	}

	forbidden := []string{
		fixAccessToken,
		fixBearerless,
		fixJWT,
		fixRefresh,
		fixIDToken,
		v1TestSecret,
		base64.RawURLEncoding.EncodeToString(sealKey),
		base64.StdEncoding.EncodeToString(sealKey),
		hex.EncodeToString(sealKey),
	}

	for _, needle := range forbidden {
		if strings.Contains(logged, needle) {
			t.Fatalf("log output carries credential material (%s…): %s", needle[:8], logged)
		}
	}
}

// TestCredStoreMissingAndEmpty locks the base posture (AUTH-02, the
// probe-ledger's "empty" resolution): a missing credentials.json is simply
// not-logged-in (empty result, nil error); a decodable-but-empty store is
// empty-nil; an undecodable store is the coded wrapped error — never a
// panic, never a fabricated entry, never a hang.
func TestCredStoreMissingAndEmpty(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), cipherTestTimeout)
	defer cancel()

	t.Run("missing file is empty result with nil error", func(t *testing.T) {
		store := interact.NewCredentialStore(
			filepath.Join(t.TempDir(), "v2", "credentials.json"), &stubLoader{})

		got, err := store.Entries(ctx)
		if err != nil {
			t.Fatalf("missing store answered %v — a fresh machine is simply not logged in", err)
		}

		if len(got) != 0 {
			t.Fatalf("missing store answered %d entries, want empty", len(got))
		}
	})

	t.Run("empty object is empty result with nil error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "v2", "credentials.json")
		writeFixtureStore(t, path, map[string]string{})

		store := interact.NewCredentialStore(path, &stubLoader{})

		got, err := store.Entries(ctx)
		if err != nil {
			t.Fatalf("empty store answered %v — {} decodes to zero entries", err)
		}

		if len(got) != 0 {
			t.Fatalf("empty store answered %d entries, want empty", len(got))
		}
	})

	t.Run("undecodable store is the coded wrapped error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "v2", "credentials.json")

		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(path, []byte("this is not a credential store"), 0o600); err != nil {
			t.Fatalf("write fixture: %v", err)
		}

		store := interact.NewCredentialStore(path, &stubLoader{})

		got, err := store.Entries(ctx)
		if !errors.Is(err, interact.ErrStoreUndecodable) {
			t.Fatalf("undecodable store answered (%v, %v), want the ErrStoreUndecodable coded error", got, err)
		}

		if len(got) != 0 {
			t.Fatalf("undecodable store answered %d entries, want empty", len(got))
		}
	})
}
