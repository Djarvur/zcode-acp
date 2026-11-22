package hostsvc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Effective-revision wiring locks: the snapshot builders must push the
// EFFECTIVE builtin revision — the runtime's own two-candidate selection —
// not the raw discovery-recorded one. Field evidence (EXP-CODINGPLAN-CYCLE-01
// attempts 1-2): a push hashed over the runtime-tree copy is schema-accepted
// and silently ignored by the registry (V-SRC registry-service.ts), because
// the runtime's bundled candidate is the install's OWN
// <root>/resources/config/provider/zcode-builtin.json. Fixtures are
// machine-independent by construction: temp paths carry no version-looking
// segment, so the default source's active-cache lookup is unreachable
// (empty appVersion) — the bundled-candidate promotion is the observable
// effective-resolution effect, and it is exactly the field gap.

// effectiveFixtureInstall writes a temp install whose discovery-recorded
// zcode-builtin.json is flatTable (revision 44) while the install root ALSO
// carries its own resources copy at rootCopyRev — the two-candidate shape of
// a real unpacked-AppImage machine (discovery records the runtime-tree copy;
// the runtime selects over the root copy and the cache).
func effectiveFixtureInstall(t *testing.T, flatTable, rootCopy string) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(flatTable), 0o600); err != nil {
		t.Fatalf("write flat builtin: %v", err)
	}

	bundled := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	if err := os.MkdirAll(filepath.Dir(bundled), 0o750); err != nil {
		t.Fatalf("mkdir bundled dir: %v", err)
	}
	if err := os.WriteFile(bundled, []byte(rootCopy), 0o600); err != nil {
		t.Fatalf("write root resources copy: %v", err)
	}

	return supervisor.Install{
		Root:        root,
		Binary:      filepath.Join(root, "zcode"),
		Entrypoint:  filepath.Join(root, "resources", "glm", "zcode.cjs"),
		BuiltinFile: builtin,
		Layout:      fixtureLayoutExplicit,
	}
}

// rootCopyRev45 is the install's own resources copy one revision ahead of
// the recorded one ({"revision":45}).
const rootCopyRev45 = `{"revision":45,"note":"bundled"}`

// flatBuiltinRev44 is the discovery-recorded builtin (revision 44).
const flatBuiltinRev44 = `{"revision":44}`

// assertBasedOnIsEffective asserts the snapshot's basedOnZCodeBuiltinRevision
// is the ROOT copy's hash (the effective bundled candidate), never the raw
// recorded-file hash — on BOTH the Snapshot.BasedOn identity and the strict
// envelope member.
func assertBasedOnIsEffective(t *testing.T, what string, snap hostsvc.Snapshot, inst supervisor.Install) {
	t.Helper()

	want := supervisor.BuiltinRevision(
		filepath.Join(inst.Root, "resources", "config", "provider", "zcode-builtin.json"), 45)
	notWant := supervisor.BuiltinRevision(inst.BuiltinFile, 44)

	if snap.BasedOn != want {
		t.Fatalf("%s basedOn = %q, want the effective (root resources copy) hash %q", what, snap.BasedOn, want)
	}

	if snap.BasedOn == notWant {
		t.Fatalf("%s basedOn = %q — the raw discovery-recorded hash leaked (the field gap this wiring closes)", what, snap.BasedOn)
	}

	if paramsBasedOn, _ := snap.Params["basedOnZCodeBuiltinRevision"].(string); paramsBasedOn != want {
		t.Fatalf("%s params basedOnZCodeBuiltinRevision = %q, want %q", what, paramsBasedOn, want)
	}
}

// TestSnapshotBuildUsesEffectiveRevision locks the api-key arm (Build, the
// phase-5 empty overlay): the push envelope carries the effective revision —
// the install's own resources copy — even when the discovery-recorded file
// disagrees in bytes and path.
func TestSnapshotBuildUsesEffectiveRevision(t *testing.T) {
	inst := effectiveFixtureInstall(t, flatBuiltinRev44, rootCopyRev45)

	snap, err := hostsvc.NewSnapshotBuilder(nil).Build(inst)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}

	assertBasedOnIsEffective(t, "build:", snap, inst)
	assertStrictEnvelope(t, snap)
}

// TestAccountBuildUsesEffectiveRevision locks the coding-plan arm
// (BuildAccount, the account overlay): the effective revision rides the
// push while the OVERLAY still mirrors the recorded table's zhipu-account
// rules verbatim — the revision is scoped plumbing, the rules source is
// untouched (phase-6 scope: revision plumbing only).
func TestAccountBuildUsesEffectiveRevision(t *testing.T) {
	inst := effectiveFixtureInstall(t, accountFixtureTable, rootCopyRev45)

	fixed := "account-bridge:effective-fix1"
	snap, err := hostsvc.NewSnapshotBuilder(nil,
		hostsvc.WithRevisionSource(func() string { return fixed })).
		BuildAccount(inst, func(string) bool { return true })
	if err != nil {
		t.Fatalf("build account snapshot: %v", err)
	}

	if snap.Revision != fixed {
		t.Fatalf("revision = %q, want the injected push revision %q", snap.Revision, fixed)
	}

	assertBasedOnIsEffective(t, "account build:", snap, inst)

	providers, _ := snap.Params["providers"].(map[string]any)
	if len(providers) != len(wantZhipuProviders()) {
		t.Fatalf("providers = %d entries, want the table's %d zhipu-account rules (rules source unchanged)",
			len(providers), len(wantZhipuProviders()))
	}

	for providerID, modelIDs := range wantZhipuProviders() {
		entry, ok := providers[providerID]
		if !ok {
			t.Fatalf("providers missing %q — the overlay must still mirror the recorded table", providerID)
		}
		assertModelIDsVerbatim(t, providerID, entry.(map[string]any)["builtinModelIds"], modelIDs)
	}
}
