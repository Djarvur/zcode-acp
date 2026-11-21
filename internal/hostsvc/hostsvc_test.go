package hostsvc_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

// peerRule scripts one inbound method at the scripted peer: the canned
// result reply and, optionally, one notification emitted afterMs after the
// reply (the happy arm's read-back surface arrives strictly AFTER the
// refresh reply registers — the live poll path the real wire shows).
type peerRule struct {
	result      json.RawMessage
	emitAfterMs int
	emitMethod  string
	emitParams  json.RawMessage
}

// scriptedPeer is the minimal scripted app-server double: it answers each
// scripted inbound request with its rule's canned result and plays the
// rule's post-reply emit. Writes serialize behind mu (the peer serves on a
// goroutine while the test writes nothing — but the mutex keeps the encode+
// write pair atomic all the same).
type scriptedPeer struct {
	mu    sync.Mutex
	w     io.Writer
	rules map[string]peerRule
}

// serve reads host lines until the stream closes, answering per rule.
func (p *scriptedPeer) serve(r io.Reader) {
	sc := bufio.NewScanner(r)

	for sc.Scan() {
		env, err := zcode.Decode(sc.Bytes())
		if err != nil || env.ID == nil || env.Method == "" {
			continue
		}

		p.answer(&env)
	}
}

// answer replies with the rule's canned result and plays its post-reply
// emit.
func (p *scriptedPeer) answer(env *zcode.Envelope) {
	rule, ok := p.rules[env.Method]
	if !ok {
		return
	}

	if rule.result != nil {
		p.write(&zcode.Envelope{ID: env.ID, Result: rule.result})
	}

	if rule.emitAfterMs > 0 {
		time.Sleep(time.Duration(rule.emitAfterMs) * time.Millisecond)
		p.write(&zcode.Envelope{Method: rule.emitMethod, Params: rule.emitParams})
	}
}

// write encodes and writes one peer→host envelope.
func (p *scriptedPeer) write(env *zcode.Envelope) {
	line, err := zcode.Encode(env)
	if err != nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	_, _ = p.w.Write(line)
}

// peerPipes joins one io.Pipe half-pair into an io.ReadWriteCloser end.
type peerPipes struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (p *peerPipes) Read(b []byte) (int, error)  { return p.r.Read(b) }
func (p *peerPipes) Write(b []byte) (int, error) { return p.w.Write(b) }
func (p *peerPipes) Close() error                { return errors.Join(p.r.Close(), p.w.Close()) }

// newPeerConn wires a real zcode.Conn to a scripted peer over crossed
// io.Pipes (the roundtrip_test pattern): the conn reads what the peer
// writes and vice versa. The returned cleanup closes both ends.
func newPeerConn(t *testing.T, rules map[string]peerRule) (*zcode.Conn, *scriptedPeer, func()) {
	t.Helper()

	toPeerR, toHostW := io.Pipe() // host writes → peer reads
	toHostR, toPeerW := io.Pipe() // peer writes → host reads

	conn := zcode.NewConn(&peerPipes{r: toHostR, w: toHostW})
	peer := &scriptedPeer{w: toPeerW, rules: rules}

	done := make(chan struct{})
	go func() {
		peer.serve(toPeerR)
		close(done)
	}()

	cleanup := func() {
		_ = conn.Close()
		_ = toPeerR.Close()
		<-done
	}

	t.Cleanup(cleanup)

	return conn, peer, cleanup
}

// readbackWant returns the wanted provider model identifiers the read-back
// arms await (providerId/modelId — the dig-out normalization).
func readbackWant() []string {
	return []string{"zai-api/GLM-5.3-Flash", "zai-api/GLM-5.3"}
}

// TestAuthReadback locks the PITFALL-1 read-back verdict in four arms:
// (a) the happy registry surface — the helper returns applied with the
// observed revision echo when a state.updated carries the expected provider
// models; (b) the ack-mismatch negative — the ack says received but the
// registry never carries the models within the deadline, so applied=false
// with the coded actionable error (the verdict is the registry's, never the
// ack's); (c) the create-reply arm — the helper also accepts the
// settings.model.available surface via OfferCreateReply; (d) the Applied
// gate — false before any surface, true once one latches (the corrected
// chain's probe gate, plan 05-11).
func TestAuthReadback(t *testing.T) {
	want := readbackWant()

	t.Run("happy registry surface", func(t *testing.T) {
		assertReadbackHappy(t, want)
	})

	t.Run("ack without registry confirmation is not application", func(t *testing.T) {
		assertReadbackAckNegative(t, want)
	})

	t.Run("create-reply surface", func(t *testing.T) {
		assertReadbackCreateReply(t, want)
	})

	t.Run("Applied gate before and after a surface", func(t *testing.T) {
		assertReadbackAppliedGate(t, want)
	})
}

// assertReadbackAppliedGate is arm (d): the watcher's Applied gate reads
// false before ANY surface latched (the corrected chain must run its probe
// observation then) and true once a surface carried the want models (the
// probe is then unnecessary — the observer stays for patches the desktop
// does emit, EXP-READBACK-DIAG-01).
func assertReadbackAppliedGate(t *testing.T, want []string) {
	t.Helper()

	conn, _, _ := newPeerConn(t, map[string]peerRule{})

	w, remove := hostsvc.WatchRegistry(conn, want)
	defer remove()

	if w.Applied() {
		t.Fatal("Applied = true before any surface — the probe gate would skip its observation wrongly")
	}

	w.OfferCreateReply(json.RawMessage(
		`{"session":{"sessionId":"sess_gate1"},` +
			`"settings":{"model":{"available":[` +
			`{"ref":{"providerId":"zai-api","modelId":"GLM-5.3-Flash"}},` +
			`{"ref":{"providerId":"zai-api","modelId":"GLM-5.3"}}]}}}`))

	if !w.Applied() {
		t.Fatal("Applied = false after the create-reply surface carried the want models")
	}
}

// peerAckRule is the canned provider/updateAccountConfig ack the read-back
// arms push against (receipt evidence only).
func peerAckRule() peerRule {
	return peerRule{result: json.RawMessage(
		`{"receivedRevision":"account-bridge:peer-fix1","providerCount":0,"status":"received"}`)}
}

// assertReadbackHappy is arm (a): PushSnapshot → RefreshRegistry →
// AwaitRegistryContains, the peer emitting the registry patch 30ms after
// the refresh reply (strictly after the helper registers) — applied=true
// with the observed revision echo.
func assertReadbackHappy(t *testing.T, want []string) {
	t.Helper()

	rules := map[string]peerRule{
		zcode.MethodProviderUpdateAccountConfig: peerAckRule(),
		zcode.MethodProviderRefreshRegistry: {
			result:      json.RawMessage(`{"refreshed":true}`),
			emitAfterMs: 30,
			emitMethod:  zcode.MethodStateUpdated,
			emitParams: json.RawMessage(`{"scope":"registry","reason":"registry_refresh","revision":2,` +
				`"patch":{"providers":{"zai-api":{"models":["GLM-5.3-Flash","GLM-5.3"]}}}}`),
		},
	}

	conn, _, _ := newPeerConn(t, rules)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := hostsvc.PushSnapshot(ctx, conn, fixtureInstall(t, 44)); err != nil {
		t.Fatalf("push: %v", err)
	}

	if err := hostsvc.RefreshRegistry(ctx, conn); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	rb, err := hostsvc.AwaitRegistryContains(ctx, conn, want)
	if err != nil {
		t.Fatalf("read-back: %v", err)
	}

	if !rb.Applied {
		t.Fatal("read-back verdict applied = false, want true (the registry surface carried the models)")
	}

	if rb.Revision != "2" {
		t.Fatalf("observed revision = %q, want the surface's revision echo %q", rb.Revision, "2")
	}
}

// assertReadbackAckNegative is arm (b): the ack says received but the
// registry never carries the models within the deadline — applied=false
// with the coded actionable error (the PITFALL-1 discipline: the verdict is
// the registry's, not the ack's).
func assertReadbackAckNegative(t *testing.T, want []string) {
	t.Helper()

	rules := map[string]peerRule{
		zcode.MethodProviderUpdateAccountConfig: peerAckRule(),
		zcode.MethodProviderRefreshRegistry:     {result: json.RawMessage(`{"refreshed":true}`)},
	}

	conn, _, _ := newPeerConn(t, rules)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	ack, err := hostsvc.PushSnapshot(ctx, conn, fixtureInstall(t, 44))
	if err != nil {
		t.Fatalf("push: %v", err)
	}

	if ack.Status != "received" {
		t.Fatalf("ack status = %q, want received (the ack SAYS received — the negative's point)", ack.Status)
	}

	if err := hostsvc.RefreshRegistry(ctx, conn); err != nil {
		t.Fatalf("refresh: %v", err)
	}

	rb, err := hostsvc.AwaitRegistryContains(ctx, conn, want)
	if rb.Applied {
		t.Fatal("applied = true on ack-only evidence — the verdict must be the registry's, never the ack's")
	}

	if !errors.Is(err, hostsvc.ErrRegistryNotConfirmed) {
		t.Fatalf("error = %v, want the coded ErrRegistryNotConfirmed sentinel", err)
	}
}

// assertReadbackCreateReply is arm (c): a bare session/create whose reply
// is offered to the watcher explicitly — the settings.model.available
// dig-out.
func assertReadbackCreateReply(t *testing.T, want []string) {
	t.Helper()

	rules := map[string]peerRule{
		zcode.MethodSessionCreate: {result: json.RawMessage(
			`{"session":{"sessionId":"sess_fix1"},` +
				`"settings":{"model":{"available":[` +
				`{"ref":{"providerId":"zai-api","modelId":"GLM-5.3-Flash"}},` +
				`{"ref":{"providerId":"zai-api","modelId":"GLM-5.3"}}]}}}`)},
	}

	conn, _, _ := newPeerConn(t, rules)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w, remove := hostsvc.WatchRegistry(conn, want)
	defer remove()

	res, err := conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
		"workspace": map[string]any{"workspacePath": "/tmp/readback-ws", "workspaceKey": "/tmp/readback-ws"},
	})
	if err != nil {
		t.Fatalf("session/create: %v", err)
	}

	w.OfferCreateReply(res)

	rb, err := w.Await(ctx)
	if err != nil {
		t.Fatalf("read-back: %v", err)
	}

	if !rb.Applied {
		t.Fatal("read-back verdict applied = false, want true (the create-reply surface carried the models)")
	}
}
