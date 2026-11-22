package hostsvc_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
	"github.com/Djarvur/zcode-acp/internal/zcode"
)

// AUTH-02 execution-tier contract locks (plan 06-03, RESEARCH Pattern 2):
// BuildAccount produces the NON-empty account-provider overlay mirrored from
// the desktop's OWN bundled builtin table (D-08's snapshot-side twin) —
// providers one entry per zhipu-account rule ({builtinModelIds verbatim,
// access {type, accountType, mode, entitled}} — the COMPLETE access schema:
// the runtime registry validates each pushed provider against the strict
// completeZhipuAccountAccessDataSchema and SILENTLY EXCLUDES a provider with
// validation issues (EXP-CODINGPLAN-CYCLE-01 attempt-4 poke "Provider
// Registry 中不存在 Model")), states per provider {availability, entitled,
// current}. The Phase-5 empty-overlay builder (TestSnapshotEnvelope) stays
// byte-untouched. The entitlement gate is a decision the builder RECEIVES —
// it may only omit families, never edit entries (T-06-09).

// accountFixtureTable is the fixture builtin table in the desktop's own
// schema (providerId / providerName / config.group / config.builtinModelIds /
// config.access{type,mode,accountType} — shape verified against this
// machine's revision-44 table, V-EXP 2026-11-22). It mixes: two zhipu-account
// rules (different families — the gate arms need them), one apiKey rule and
// one other-type rule (the leak guards), a mixed-case model id and a
// slash-free provider name (the no-transformation locks, AUTH-02 encoding).
const accountFixtureTable = `{
  "schemaVersion": 1,
  "revision": 44,
  "config": {
    "providerConfigRules": {
      "templateRules": [],
      "providerRules": [
        {
          "providerId": "account:zai-individual-coding-plan",
          "providerName": "Z.AI Individual Coding Plan",
          "config": {
            "group": "zai-family",
            "builtinModelIds": ["GLM-5.3", "GLM-5.3-Flash", "gLM-MiXeD-Case"],
            "access": {"type": "zhipu-account", "mode": "individual-coding-plan", "accountType": "zai"}
          }
        },
        {
          "providerId": "account:bigmodel-individual-coding-plan",
          "providerName": "BigModel Individual Coding Plan",
          "config": {
            "group": "bigmodel-family",
            "builtinModelIds": ["GLM-5.2"],
            "access": {"type": "zhipu-account", "mode": "individual-coding-plan", "accountType": "bigmodel"}
          }
        },
        {
          "providerId": "zai-api",
          "providerName": "Z.AI API Key",
          "config": {
            "builtinModelIds": ["GLM-5.3"],
            "access": {"type": "apiKey"}
          }
        },
        {
          "providerId": "account:trial-access",
          "providerName": "Trial Access",
          "config": {
            "builtinModelIds": ["M-1"],
            "access": {"type": "trial"}
          }
        }
      ]
    }
  }
}`

// wantZhipuProviders is the exact providers/states shape the fixture's
// zhipu-account rules must build (all-entitled gate): model lists are the
// table's arrays verbatim, entitlement true. (A function keeps the fixture
// gochecknoglobals-clean — the accountFixtureTable precedent.)
func wantZhipuProviders() map[string][]string {
	return map[string][]string{
		"account:zai-individual-coding-plan":      {"GLM-5.3", "GLM-5.3-Flash", "gLM-MiXeD-Case"},
		"account:bigmodel-individual-coding-plan": {"GLM-5.2"},
	}
}

// wantZhipuAccess is the pushed access's accountType+mode pair per fixture
// zhipu-account rule — capture-verbatim values from the fixture table's
// access block. The runtime registry validates each pushed provider against
// the COMPLETE access schema ({type, accountType, mode, entitled}, strict
// zod): an access missing accountType or mode is a validation issue, and a
// provider with validation issues is silently excluded from the registry —
// exactly the EXP-CODINGPLAN-CYCLE-01 attempt-4 setModel poke symptom
// ("Provider Registry 中不存在 Model", state.updated never carrying the
// account models).
func wantZhipuAccess() map[string][2]string {
	return map[string][2]string{
		"account:zai-individual-coding-plan":      {"zai", "individual-coding-plan"},
		"account:bigmodel-individual-coding-plan": {"bigmodel", "individual-coding-plan"},
	}
}

// accountFixtureInstall writes a temp install whose zcode-builtin.json is the
// full account fixture table. Two calls never share a path, so the
// basedOnZCodeBuiltinRevision path-sensitivity lock bites (the formula hashes
// the resolved absolute PATH STRING, not the contents).
func accountFixtureInstall(t *testing.T) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(accountFixtureTable), 0o600); err != nil {
		t.Fatalf("write builtin fixture: %v", err)
	}

	return supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: builtin,
		Layout:      fixtureLayoutExplicit,
	}
}

// decodeEnvelope marshals a built Snapshot's params and decodes the strict
// envelope object: exactly the four schema keys as raw JSON members.
func decodeEnvelope(t *testing.T, snap hostsvc.Snapshot) map[string]json.RawMessage {
	t.Helper()

	wire, err := json.Marshal(snap.Params)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(wire, &got); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, wire)
	}

	wantKeys := envelopeKeys()
	if len(got) != len(wantKeys) {
		t.Fatalf("envelope keys = %v, want exactly %v (strict schema — extra fields are parse errors)",
			keysOf(got), wantKeys)
	}

	for _, k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Fatalf("envelope missing key %q (strict schema requires all four)", k)
		}
	}

	return got
}

// assertModelIDsVerbatim asserts one provider's builtinModelIds equals the
// fixture table's array element-for-element — every emitted string traces to
// the table byte-for-byte (T-06-09: no wrapper-side renaming or editing).
func assertModelIDsVerbatim(t *testing.T, what string, got any, want []string) {
	t.Helper()

	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("%s builtinModelIds = %v (%T), want a JSON array", what, got, got)
	}

	if len(arr) != len(want) {
		t.Fatalf("%s builtinModelIds = %v, want the table's array verbatim (%d elements): %v",
			what, arr, len(want), want)
	}

	for i, el := range arr {
		if s, _ := el.(string); s != want[i] {
			t.Fatalf("%s builtinModelIds[%d] = %v, want the table's %q verbatim (no transformation)",
				what, i, el, want[i])
		}
	}
}

// TestAccountSnapshotVerbatimMirror locks the account overlay mirror over the
// fixture table with an all-entitled gate: providers contains EXACTLY the
// zhipu-account rules (never the apiKey/trial rows), each entry mirrors the
// table verbatim with access {type zhipu-account, entitled true}, states
// carries the strict {availability available, entitled true, current true}
// per entry, basedOnZCodeBuiltinRevision is the exact
// supervisor.BuiltinRevision output for the fixture path AND changes when the
// path changes (T-06-08), and the revision is the builder's non-empty
// account-bridge form.
//
//nolint:funlen,gocognit,gocyclo,cyclop // one mirror flow over one snapshot; splitting would rebuild it per arm
func TestAccountSnapshotVerbatimMirror(t *testing.T) {
	instA := accountFixtureInstall(t)
	instB := accountFixtureInstall(t) // same table bytes, different resolved path

	fixed := "account-bridge:account-fix1"
	b := hostsvc.NewSnapshotBuilder(nil, hostsvc.WithRevisionSource(func() string { return fixed }))

	snap, err := b.BuildAccount(instA, func(string) bool { return true })
	if err != nil {
		t.Fatalf("build account snapshot: %v", err)
	}

	if snap.Revision != fixed || !strings.HasPrefix(snap.Revision, "account-bridge:") {
		t.Fatalf("revision = %q, want the builder's non-empty account-bridge form %q", snap.Revision, fixed)
	}

	wantA := supervisor.BuiltinRevision(instA.BuiltinFile, 44)
	if snap.BasedOn != wantA {
		t.Fatalf("basedOn = %q, want the exact supervisor.BuiltinRevision output %q", snap.BasedOn, wantA)
	}

	snapB, err := b.BuildAccount(instB, func(string) bool { return true })
	if err != nil {
		t.Fatalf("build account snapshot (path B): %v", err)
	}

	wantB := supervisor.BuiltinRevision(instB.BuiltinFile, 44)
	if snapB.BasedOn != wantB {
		t.Fatalf("basedOn (path B) = %q, want %q", snapB.BasedOn, wantB)
	}

	if snapB.BasedOn == snap.BasedOn {
		t.Fatalf("basedOn is path-insensitive (%q for both installs) — "+
			"the formula must be computed at runtime over the resolved path, never hard-coded", snap.BasedOn)
	}

	got := decodeEnvelope(t, snap)

	var providers map[string]any
	if err := json.Unmarshal(got["providers"], &providers); err != nil {
		t.Fatalf("providers does not decode as a record: %v (%s)", err, got["providers"])
	}

	if len(providers) != len(wantZhipuProviders()) {
		t.Fatalf("providers keys = %v, want exactly the zhipu-account rules %v "+
			"(a non-zhipu rule leaked in or a family went missing)",
			keysOfAny(providers), keysOfStr(wantZhipuProviders()))
	}

	for pid, models := range wantZhipuProviders() {
		entry, ok := providers[pid].(map[string]any)
		if !ok {
			t.Fatalf("providers[%s] missing or not an object: %v", pid, providers[pid])
		}

		if len(entry) != 2 {
			t.Fatalf("providers[%s] keys = %v, want exactly [access builtinModelIds] (strict zod shape)",
				pid, keysOfAny(entry))
		}

		assertModelIDsVerbatim(t, "providers["+pid+"]", entry["builtinModelIds"], models)

		access, ok := entry["access"].(map[string]any)
		if !ok {
			t.Fatalf("providers[%s].access missing or not an object: %v", pid, entry["access"])
		}

		if len(access) != 4 {
			t.Fatalf("providers[%s].access keys = %v, want exactly [accountType entitled mode type] "+
				"(the complete access schema — an access missing accountType/mode fails the runtime's "+
				"strict zhipu-account validation and the provider is silently excluded from the registry)",
				pid, keysOfAny(access))
		}

		if access["type"] != "zhipu-account" {
			t.Fatalf("providers[%s].access.type = %v, want the table's zhipu-account verbatim", pid, access["type"])
		}

		if access["entitled"] != true {
			t.Fatalf("providers[%s].access.entitled = %v, want true (entitled account entry)", pid, access["entitled"])
		}

		wantAccess := wantZhipuAccess()[pid]

		if access["accountType"] != wantAccess[0] {
			t.Fatalf("providers[%s].access.accountType = %v, want the table's %q verbatim "+
				"(capture-verbatim — never a wrapper-side literal)", pid, access["accountType"], wantAccess[0])
		}

		if access["mode"] != wantAccess[1] {
			t.Fatalf("providers[%s].access.mode = %v, want the table's %q verbatim "+
				"(capture-verbatim — never a wrapper-side literal)", pid, access["mode"], wantAccess[1])
		}
	}

	var states map[string]any
	if err := json.Unmarshal(got["states"], &states); err != nil {
		t.Fatalf("states does not decode as a record: %v (%s)", err, got["states"])
	}

	if len(states) != len(wantZhipuProviders()) {
		t.Fatalf("states keys = %v, want exactly the entitled providers %v",
			keysOfAny(states), keysOfStr(wantZhipuProviders()))
	}

	for pid := range wantZhipuProviders() {
		st, ok := states[pid].(map[string]any)
		if !ok {
			t.Fatalf("states[%s] missing or not an object: %v", pid, states[pid])
		}

		if len(st) != 3 {
			t.Fatalf("states[%s] keys = %v, want exactly [availability current entitled] (strict zod shape)",
				pid, keysOfAny(st))
		}

		if st["availability"] != "available" {
			t.Fatalf("states[%s].availability = %v, want available (entitled zhipu-account entry)",
				pid, st["availability"])
		}

		if st["entitled"] != true {
			t.Fatalf("states[%s].entitled = %v, want true", pid, st["entitled"])
		}

		if st["current"] != true {
			t.Fatalf("states[%s].current = %v, want true (required for entitled zhipu-account entries)",
				pid, st["current"])
		}
	}
}

// keysOfStr lists the keys of a string-slice map (sorted — deterministic).
func keysOfStr(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}

	return out
}

// accountFixtureTableNoZhipu is a builtin table with ZERO zhipu-account
// rules (only the apiKey personal-layer row and another type) — the
// misconfiguration arm: an account push with nothing to entitle.
const accountFixtureTableNoZhipu = `{
  "revision": 44,
  "config": {
    "providerConfigRules": {
      "providerRules": [
        {
          "providerId": "zai-api",
          "providerName": "Z.AI API Key",
          "config": {
            "builtinModelIds": ["GLM-5.3"],
            "access": {"type": "apiKey"}
          }
        },
        {
          "providerId": "account:trial-access",
          "providerName": "Trial Access",
          "config": {
            "builtinModelIds": ["M-1"],
            "access": {"type": "trial"}
          }
        }
      ]
    }
  }
}`

// accountTableUndecodable is not JSON — the table tampering arm.
const accountTableUndecodable = `{"revision":44,"config":{"providerConfigRules"`

// accountTableMissingRules carries the config envelope but no providerRules
// key — the malformed-shape arm.
const accountTableMissingRules = `{"revision":44,"config":{"providerConfigRules":{"templateRules":[]}}}`

// accountTableRuleNoAccess is the fixture table with the first zhipu rule's
// access block stripped — the per-rule malformed arm (a rule the wrapper
// cannot classify).
const accountTableRuleNoAccess = `{
  "revision": 44,
  "config": {
    "providerConfigRules": {
      "providerRules": [
        {
          "providerId": "account:zai-individual-coding-plan",
          "providerName": "Z.AI Individual Coding Plan",
          "config": {
            "group": "zai-family",
            "builtinModelIds": ["GLM-5.3"]
          }
        },
        {
          "providerId": "account:bigmodel-individual-coding-plan",
          "providerName": "BigModel Individual Coding Plan",
          "config": {
            "builtinModelIds": ["GLM-5.2"],
            "access": {"type": "zhipu-account", "accountType": "bigmodel"}
          }
        }
      ]
    }
  }
}`

// accountTableRuleNoModels is the fixture table with the first zhipu rule's
// model list stripped (its access block stays COMPLETE) — the same coded
// class (a rule that cannot be mirrored).
const accountTableRuleNoModels = `{
  "revision": 44,
  "config": {
    "providerConfigRules": {
      "providerRules": [
        {
          "providerId": "account:zai-individual-coding-plan",
          "providerName": "Z.AI Individual Coding Plan",
          "config": {
            "access": {"type": "zhipu-account", "mode": "individual-coding-plan", "accountType": "zai"}
          }
        }
      ]
    }
  }
}`

// accountTableRuleNoMode is the fixture table with the first zhipu rule's
// access mode stripped (models + accountType stay) — the same coded
// missing-access class: a zhipu-account rule without mode cannot build a
// COMPLETE access schema push, and the runtime registry would silently
// exclude the provider (EXP-CODINGPLAN-CYCLE-01 attempt-4 symptom) — the
// wrapper must refuse the build, never push the incomplete access.
const accountTableRuleNoMode = `{
  "revision": 44,
  "config": {
    "providerConfigRules": {
      "providerRules": [
        {
          "providerId": "account:zai-individual-coding-plan",
          "providerName": "Z.AI Individual Coding Plan",
          "config": {
            "group": "zai-family",
            "builtinModelIds": ["GLM-5.3"],
            "access": {"type": "zhipu-account", "accountType": "zai"}
          }
        }
      ]
    }
  }
}`

// writeBuiltinTable writes one raw table body to a temp install's builtin
// file (the malformed arms need raw control over the bytes).
func writeBuiltinTable(t *testing.T, body string) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(body), 0o600); err != nil {
		t.Fatalf("write builtin fixture: %v", err)
	}

	return supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: builtin,
		Layout:      fixtureLayoutExplicit,
	}
}

// allEntitled is the all-pass entitlement gate.
func allEntitled(string) bool { return true }

// assertNoPartialSnapshot locks the coded-error arm's shape: the zero
// Snapshot returns nothing pushable — never a partial overlay.
func assertNoPartialSnapshot(t *testing.T, snap hostsvc.Snapshot, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("BuildAccount answered success (params %v) — want the coded build error", snap.Params)
	}

	if snap.Params != nil {
		t.Fatalf("error arm returned a non-nil Params map %v — never a partial push", snap.Params)
	}
}

// TestAccountSnapshotGateOmission locks the gate's omission-only discipline
// (D-08/T-06-09): a gate rejecting the bigmodel family omits EXACTLY that
// family from providers AND states while the zai entry stays intact — the
// gate may only omit, never edit an entry.
func TestAccountSnapshotGateOmission(t *testing.T) {
	inst := accountFixtureInstall(t)

	fixed := "account-bridge:gate-fix1"
	b := hostsvc.NewSnapshotBuilder(nil, hostsvc.WithRevisionSource(func() string { return fixed }))

	snap, err := b.BuildAccount(inst, func(family string) bool { return family == fixtureFamily })
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	got := decodeEnvelope(t, snap)

	var providers map[string]any
	if err := json.Unmarshal(got["providers"], &providers); err != nil {
		t.Fatalf("providers does not decode: %v (%s)", err, got["providers"])
	}

	if len(providers) != 1 {
		t.Fatalf("providers keys = %v, want exactly [account:zai-individual-coding-plan] "+
			"(the gate-rejected family must be omitted entirely)", keysOfAny(providers))
	}

	entry, ok := providers["account:zai-individual-coding-plan"].(map[string]any)
	if !ok {
		t.Fatalf("providers[account:zai-individual-coding-plan] missing: %v", providers)
	}

	assertModelIDsVerbatim(t, "providers[account:zai-individual-coding-plan]",
		entry["builtinModelIds"], wantZhipuProviders()["account:zai-individual-coding-plan"])

	// The surviving entry's access stays COMPLETE and unedited: the gate's
	// omission of the bigmodel family must not strip or rewrite the zai
	// entry's accountType/mode (the gate may only omit, never edit — T-06-09).
	access, ok := entry["access"].(map[string]any)
	if !ok {
		t.Fatalf("providers[account:zai-individual-coding-plan].access missing: %v", entry["access"])
	}

	wantAccess := wantZhipuAccess()["account:zai-individual-coding-plan"]

	if len(access) != 4 || access["accountType"] != wantAccess[0] || access["mode"] != wantAccess[1] ||
		access["type"] != "zhipu-account" || access["entitled"] != true {
		t.Fatalf("providers[account:zai-individual-coding-plan].access = %v, want the table-verbatim "+
			"complete schema {type zhipu-account, accountType %q, mode %q, entitled true}",
			keysOfAny(access), wantAccess[0], wantAccess[1])
	}

	var states map[string]any
	if err := json.Unmarshal(got["states"], &states); err != nil {
		t.Fatalf("states does not decode: %v (%s)", err, got["states"])
	}

	if len(states) != 1 {
		t.Fatalf("states keys = %v, want exactly [account:zai-individual-coding-plan] "+
			"(omission covers providers AND states)", keysOfAny(states))
	}

	if _, ok := states["account:zai-individual-coding-plan"]; !ok {
		t.Fatal("states[account:zai-individual-coding-plan] missing — the entitled family must stay intact")
	}
}

// TestAccountSnapshotNoRules locks the misconfiguration arm (AUTH-02 empty
// probe, resolved explicit): a builtin table with ZERO zhipu-account rules
// answers a coded error NAMING the check — an account push with nothing to
// entitle is a misconfiguration, never an empty overlay silently pushed on
// the coding-plan path (that would push the api-key shape onto the
// coding-plan path and unentitle every account model).
func TestAccountSnapshotNoRules(t *testing.T) {
	inst := writeBuiltinTable(t, accountFixtureTableNoZhipu)

	b := hostsvc.NewSnapshotBuilder(nil)

	snap, err := b.BuildAccount(inst, allEntitled)
	assertNoPartialSnapshot(t, snap, err)

	if !strings.Contains(err.Error(), "zhipu-account") {
		t.Fatalf("error %q does not name the zhipu-account check", err)
	}
}

// TestAccountSnapshotMalformedTable locks the tampering arm: undecodable
// JSON, a missing providerRules key, a rule missing its access block (and
// the same coded class for a rule missing its model list or its access
// mode — a zhipu-account rule without mode cannot build the COMPLETE access
// schema push) each answer the coded build error — never a partial push,
// never a panic.
func TestAccountSnapshotMalformedTable(t *testing.T) {
	arms := map[string]string{
		"undecodable JSON":              accountTableUndecodable,
		"missing providerRules key":     accountTableMissingRules,
		"rule missing its access block": accountTableRuleNoAccess,
		"rule missing its model list":   accountTableRuleNoModels,
		"rule missing its access mode":  accountTableRuleNoMode,
	}

	for name, table := range arms {
		t.Run(name, func(t *testing.T) {
			inst := writeBuiltinTable(t, table)

			b := hostsvc.NewSnapshotBuilder(nil)

			snap, err := b.BuildAccount(inst, allEntitled)
			assertNoPartialSnapshot(t, snap, err)
		})
	}
}

// TestPushEnvelopeCompat locks the 06-05 consumer contract: BuildAccount's
// Snapshot feeds the EXISTING PushEnvelope unchanged — the params map
// contract the fake-tier corpus in 06-05 consumes is exactly this shape.
func TestPushEnvelopeCompat(t *testing.T) {
	inst := accountFixtureInstall(t)

	b := hostsvc.NewSnapshotBuilder(nil,
		hostsvc.WithRevisionSource(func() string { return "account-bridge:compat-fix1" }))

	snap, err := b.BuildAccount(inst, allEntitled)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	conn, _, _ := newPeerConn(t, map[string]peerRule{
		zcode.MethodProviderUpdateAccountConfig: peerAckRule(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ack, err := hostsvc.PushEnvelope(ctx, conn, snap.Params)
	if err != nil {
		t.Fatalf("push: %v", err)
	}

	if ack.Status != "received" {
		t.Fatalf("ack status = %q, want received — PushEnvelope must consume the account "+
			"Snapshot's Params unchanged", ack.Status)
	}

	if ack.ReceivedRevision != "account-bridge:peer-fix1" {
		t.Fatalf("ack receivedRevision = %q, want the scripted peer's echo", ack.ReceivedRevision)
	}
}
