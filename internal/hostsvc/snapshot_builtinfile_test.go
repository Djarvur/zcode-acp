package hostsvc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/hostsvc"
	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// The FIELD-truth revision lock (round-3, EXP-CODINGPLAN-CYCLE-01 attempts
// 1-3 + the desktop GUI host's own registry trace): the runtime's registry
// revision is the ACTIVE-cache path hash — the same zcode-builtin.json copy
// discovery records into install.BuiltinFile — even when the install root
// ALSO carries a same-revision (or different-revision) bundled resources
// copy. Attempts 1-2 pushed exactly the recorded copy's hash and the ack
// matched the runtime's revision; the 2026-11-15 host-log trace carries it
// verbatim ("Provider Registry 已就绪", configRevision
// ["zcode-builtin:44:e9c0674…", "<personal-hash>"]). The two-candidate
// tie-to-bundled selection (zcode-src 3.17.1 clone, corroborating tier only)
// does NOT hold on the shipped 3.18.2 runtime: the snapshot builders must
// hash install.BuiltinFile and NEVER promote the install's own resources
// copy — a push of the bundled hash cannot match the runtime's registry.

// builtinFileFixtureInstall writes a temp install whose discovery-recorded
// zcode-builtin.json carries recordedRev while the install root ALSO carries
// its own resources copy at rootRev — the real unpacked-AppImage shape
// (discovery records the runtime-tree copy; the root ships its own bundled
// one; the two differ in bytes, so their path hashes differ).
func builtinFileFixtureInstall(t *testing.T, recordedDoc, rootDoc string) supervisor.Install {
	t.Helper()

	root := t.TempDir()
	builtin := filepath.Join(root, "zcode-builtin.json")

	if err := os.WriteFile(builtin, []byte(recordedDoc), 0o600); err != nil {
		t.Fatalf("write recorded builtin: %v", err)
	}

	bundled := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	if err := os.MkdirAll(filepath.Dir(bundled), 0o750); err != nil {
		t.Fatalf("mkdir resources dir: %v", err)
	}
	if err := os.WriteFile(bundled, []byte(rootDoc), 0o600); err != nil {
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

// assertBasedOnIsRecordedFile asserts the snapshot's basedOnZCodeBuiltinRevision
// is the DISCOVERY-RECORDED file's formula output, never the root resources
// copy's — on the Snapshot.BasedOn identity, the strict envelope member, and
// the negative arm (a promoted resources-copy hash is the field-refuted
// hypothesis).
func assertBasedOnIsRecordedFile(t *testing.T, what string, snap hostsvc.Snapshot, inst supervisor.Install) {
	t.Helper()

	want := supervisor.BuiltinRevision(inst.BuiltinFile, 44)
	notWant := supervisor.BuiltinRevision(
		filepath.Join(inst.Root, "resources", "config", "provider", "zcode-builtin.json"), 44)

	if want == notWant {
		t.Fatal("fixture degenerated: recorded file and resources copy hash equal")
	}

	if snap.BasedOn != want {
		t.Fatalf("%s basedOn = %q, want the recorded-file hash %q "+
			"(the active-cache copy the runtime's own registry carries — V-EXP)", what, snap.BasedOn, want)
	}

	if snap.BasedOn == notWant {
		t.Fatalf("%s basedOn = %q — the resources-copy hash leaked "+
			"(the tie-to-bundled hypothesis the field evidence refuted)", what, snap.BasedOn)
	}

	if paramsBasedOn, _ := snap.Params["basedOnZCodeBuiltinRevision"].(string); paramsBasedOn != want {
		t.Fatalf("%s params basedOnZCodeBuiltinRevision = %q, want %q", what, paramsBasedOn, want)
	}
}

// TestSnapshotBuildUsesRecordedBuiltinFile locks the api-key arm (Build, the
// phase-5 empty overlay): the push envelope hashes the discovery-recorded
// file even when the install root carries its own same-revision resources
// copy with different bytes.
func TestSnapshotBuildUsesRecordedBuiltinFile(t *testing.T) {
	inst := builtinFileFixtureInstall(t, `{"revision":44,"note":"recorded"}`, `{"revision":44,"note":"bundled"}`)

	snap, err := hostsvc.NewSnapshotBuilder(nil).Build(inst)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}

	assertBasedOnIsRecordedFile(t, "build:", snap, inst)
	assertStrictEnvelope(t, snap)
}

// TestAccountBuildUsesRecordedBuiltinFile locks the coding-plan arm
// (BuildAccount, the account overlay): the revision hashes the recorded file
// while the OVERLAY still mirrors that same table's zhipu-account rules
// verbatim — one file is both the overlay source and the revision hash input.
func TestAccountBuildUsesRecordedBuiltinFile(t *testing.T) {
	inst := builtinFileFixtureInstall(t, accountFixtureTable, `{"revision":44,"note":"bundled"}`)

	fixed := "account-bridge:recorded-fix1"
	snap, err := hostsvc.NewSnapshotBuilder(nil,
		hostsvc.WithRevisionSource(func() string { return fixed })).
		BuildAccount(inst, func(string) bool { return true })
	if err != nil {
		t.Fatalf("build account snapshot: %v", err)
	}

	if snap.Revision != fixed {
		t.Fatalf("revision = %q, want the injected push revision %q", snap.Revision, fixed)
	}

	assertBasedOnIsRecordedFile(t, "account build:", snap, inst)
}
