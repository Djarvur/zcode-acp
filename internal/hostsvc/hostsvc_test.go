package hostsvc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/interact"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// AUTH-01 execution-tier contract locks (plan 05-06): the snapshot envelope
// carries the exact builtin-revision formula (reused from
// supervisor.BuiltinRevision, never re-derived, never hard-coded) over an
// EMPTY personal-layer overlay, and the runtime-headers answer serves the
// D-02 success shape from an injected KeySource with coded errors otherwise
// — the failure-shaped success arm (headersApplied:false) never leaves the
// process.

// syntheticTestKey is the in-test key material (fixture data only — never a
// real credential; the scan-secrets gate stays green).
const syntheticTestKey = "zc-synthetic-test-key"

// fixtureInstall writes a temp install whose zcode-builtin.json records the
// given revision int. The formula input is the resolved absolute PATH
// STRING, so two temp dirs with equal revision ints must produce different
// basedOn strings (the path-sensitivity lock).
func fixtureInstall(t *testing.T, rev int) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(fmt.Sprintf(`{"revision":%d}`, rev)), 0o600); err != nil {
		t.Fatalf("write builtin fixture: %v", err)
	}

	return supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: builtin,
		Layout:      "explicit",
	}
}

// TestSnapshotEnvelope locks the strict push envelope: the generated
// revision (stable account-bridge form), basedOnZCodeBuiltinRevision
// exactly tracking supervisor.BuiltinRevision for the same resolved path
// (and CHANGING when the path changes — the no-hard-code lock), and empty
// providers/states records decoding as the strict shape.
func TestSnapshotEnvelope(t *testing.T) {
	instA := fixtureInstall(t, 44)
	instB := fixtureInstall(t, 44)

	b := hostsvc.NewSnapshotBuilder(nil)

	snapA, err := b.Build(instA)
	if err != nil {
		t.Fatalf("build install A: %v", err)
	}

	if snapA.Revision == "" {
		t.Fatal("snapshot revision is empty, want a generated non-empty string")
	}

	if !strings.HasPrefix(snapA.Revision, "account-bridge:") {
		t.Fatalf("snapshot revision %q lacks the account-bridge: stable form (reference-adapter precedent)",
			snapA.Revision)
	}

	wantA := supervisor.BuiltinRevision(instA.BuiltinFile, 44)
	if snapA.BasedOn != wantA {
		t.Fatalf("basedOn = %q, want the exact supervisor.BuiltinRevision output %q", snapA.BasedOn, wantA)
	}

	snapB, err := b.Build(instB)
	if err != nil {
		t.Fatalf("build install B: %v", err)
	}

	wantB := supervisor.BuiltinRevision(instB.BuiltinFile, 44)
	if snapB.BasedOn != wantB {
		t.Fatalf("basedOn (install B) = %q, want %q", snapB.BasedOn, wantB)
	}

	if snapB.BasedOn == snapA.BasedOn {
		t.Fatalf("basedOn is path-insensitive (%q for both installs) — "+
			"the formula must be computed at runtime, never hard-coded", snapA.BasedOn)
	}

	assertStrictEnvelope(t, snapA)

	fixed := "account-bridge:corpus-fix1"
	bf := hostsvc.NewSnapshotBuilder(nil, hostsvc.WithRevisionSource(func() string { return fixed }))
	snapF := bf.BuildRevision(instA, 44)

	if snapF.Revision != fixed {
		t.Fatalf("injected revision = %q, want %q", snapF.Revision, fixed)
	}

	if snapF.BasedOn != wantA {
		t.Fatalf("BuildRevision basedOn = %q, want %q (same formula path)", snapF.BasedOn, wantA)
	}
}

// assertStrictEnvelope decodes the built params and asserts the strict
// envelope shape: exactly the four schema keys, providers and states
// marshaling as empty records (the verified personal-layer no-op overlay).
func assertStrictEnvelope(t *testing.T, snap hostsvc.Snapshot) {
	t.Helper()

	wire, err := json.Marshal(snap.Params)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(wire, &got); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, wire)
	}

	wantKeys := []string{"revision", "basedOnZCodeBuiltinRevision", "providers", "states"}
	if len(got) != len(wantKeys) {
		t.Fatalf("envelope keys = %v, want exactly %v (strict schema — extra fields are parse errors)",
			keysOf(got), wantKeys)
	}

	for _, k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Fatalf("envelope missing key %q (strict schema requires all four)", k)
		}
	}

	for _, k := range []string{"providers", "states"} {
		assertEmptyRecord(t, k, got[k])
	}

	var revision string
	if err := json.Unmarshal(got["revision"], &revision); err != nil || revision == "" {
		t.Fatalf("revision = %s, want a non-empty string", got["revision"])
	}

	var basedOn string
	if err := json.Unmarshal(got["basedOnZCodeBuiltinRevision"], &basedOn); err != nil || basedOn == "" {
		t.Fatalf("basedOnZCodeBuiltinRevision = %s, want a non-empty string", got["basedOnZCodeBuiltinRevision"])
	}
}

// assertEmptyRecord asserts one envelope member decodes as the empty record.
func assertEmptyRecord(t *testing.T, key string, raw json.RawMessage) {
	t.Helper()

	var rec map[string]any
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("%s does not decode as a record: %v (%s)", key, err, raw)
	}

	if len(rec) != 0 {
		t.Fatalf("%s = %s, want the empty record {} (the personal-layer no-op overlay)", key, raw)
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}

// stubKey is the in-test KeySource double.
type stubKey struct {
	apiKey string
	keyID  string
	ok     bool
}

func (s *stubKey) APIKey(_ context.Context) (apiKey, keyID string, ok bool) {
	return s.apiKey, s.keyID, s.ok
}

// runtimeHeadersEnvelope builds one interaction/requestProviderRuntimeHeaders
// reverse-request envelope with a distinct requestId per arm.
func runtimeHeadersEnvelope(reqID string) zcode.Envelope {
	return zcode.Envelope{
		ID:     &zcode.ID{Str: "server-" + reqID},
		Method: "interaction/requestProviderRuntimeHeaders",
		Params: json.RawMessage(fmt.Sprintf(
			`{"requestId":%q,"sessionId":"sess_fix1","providerId":"zhipu-account","reason":"model-refresh"}`,
			reqID)),
	}
}

// serveRuntimeHeaders builds a responder over a stub key source (the 05-01
// seam wired with the hostsvc handler) and serves one ask; the result is
// returned as its marshaled wire payload (nil result marshals "null").
func serveRuntimeHeaders(
	t *testing.T,
	reqID string,
	src hostsvc.KeySource,
) (wire []byte, werr *zcode.WireError) {
	t.Helper()

	responder := interact.NewResponder(interact.WithRuntimeHeadersKeySource(
		hostsvc.RuntimeHeadersHandler(src)))

	result, serveErr, handled := responder.Serve(context.Background(), runtimeHeadersEnvelope(reqID))
	if !handled {
		t.Fatal("the runtime-headers ask must be handled by the responder, never silence")
	}

	out, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	return out, serveErr
}

// decodeResult decodes a served result payload into its object shape.
func decodeResult(t *testing.T, wire []byte) map[string]any {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal(wire, &got); err != nil {
		t.Fatalf("decode result: %v (%s)", err, wire)
	}

	return got
}

// TestRuntimeHeaders locks both reply arms over the 05-01 seam: with a key
// present the strict D-02 success shape marshals from the injected source
// (both keys optional strings — apiKeyId omitted when empty); with no key
// the coded error envelope answers (NEVER the failure-shaped
// headersApplied:false arm).
func TestRuntimeHeaders(t *testing.T) {
	t.Run("configured", func(t *testing.T) {
		wire, werr := serveRuntimeHeaders(t, "rh-fix-1",
			&stubKey{apiKey: syntheticTestKey, keyID: "key_fix_1", ok: true})

		if werr != nil {
			t.Fatalf("configured arm answered a wire error: %v", werr)
		}

		assertHeadersSuccess(t, wire, syntheticTestKey, "key_fix_1")
	})

	t.Run("apiKeyId omitted when empty", func(t *testing.T) {
		wire, werr := serveRuntimeHeaders(t, "rh-fix-2",
			&stubKey{apiKey: syntheticTestKey, ok: true})

		if werr != nil {
			t.Fatalf("configured arm (no key id) answered a wire error: %v", werr)
		}

		auth, ok := decodeResult(t, wire)["requestAuth"].(map[string]any)
		if !ok {
			t.Fatalf("requestAuth missing: %s", wire)
		}

		if len(auth) != 1 || auth["apiKey"] == "" {
			t.Fatalf("requestAuth = %v, want exactly {apiKey} "+
				"(apiKeyId is an optional string, omitted when empty)", keysOfAny(auth))
		}
	})

	t.Run("unconfigured answers coded error", func(t *testing.T) {
		wire, werr := serveRuntimeHeaders(t, "rh-fix-3", &stubKey{ok: false})
		assertHeadersCodedError(t, wire, werr)
	})
}

// assertHeadersSuccess locks the strict D-02 success shape: the
// headersApplied discriminant true and the requestAuth object carrying the
// injected material verbatim.
func assertHeadersSuccess(t *testing.T, wire []byte, wantKey, wantID string) {
	t.Helper()

	got := decodeResult(t, wire)

	if applied, _ := got["headersApplied"].(bool); !applied {
		t.Fatalf("headersApplied = %v, want true (the discriminant)", got["headersApplied"])
	}

	auth, ok := got["requestAuth"].(map[string]any)
	if !ok {
		t.Fatalf("requestAuth missing or not an object: %v (strict requestAuth object)", got["requestAuth"])
	}

	if len(auth) != 2 {
		t.Fatalf("requestAuth keys = %v, want exactly [apiKey apiKeyId]", keysOfAny(auth))
	}

	if auth["apiKey"] != wantKey || auth["apiKeyId"] != wantID {
		t.Fatalf("requestAuth = %v, want the injected key material verbatim", auth)
	}
}

// assertHeadersCodedError locks the unconfigured arm: nil result, coded
// -32603 wire error — never the failure-shaped success arm.
func assertHeadersCodedError(t *testing.T, wire []byte, werr *zcode.WireError) {
	t.Helper()

	if string(wire) != "null" {
		t.Fatalf("unconfigured arm answered a result %s — "+
			"the failure-shaped success arm must never leave the process", wire)
	}

	if werr == nil {
		t.Fatal("unconfigured arm answered no wire error, want the coded error envelope")
	}

	if werr.Code != zcode.CodeInternalError {
		t.Fatalf("error code = %d, want %d (-32603 coded internal)", werr.Code, zcode.CodeInternalError)
	}
}

func keysOfAny(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
