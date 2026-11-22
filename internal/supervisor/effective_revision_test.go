package supervisor_test

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Contract tests for the effective builtin revision (the runtime's own
// registry revision string). Source of truth: the two-candidate
// selectReleaseCandidate over the install's bundled zcode-builtin.json and
// the runtime-cache active copy (V-SRC zcode-builtin-provider-config-source.ts
// + zcode-builtin-cache-paths.ts, corroborated in
// docs/protocol/install-discovery.md). Every fixture is a t.TempDir() tree —
// no test reads the real machine layout; the tie vector mirrors THIS
// machine's recorded field case (both revisions 44, different bytes → the
// bundled path hash wins) without naming any user-specific path.

// effFixture is one two-candidate fixture tree: an install root optionally
// carrying its own resources builtin copy and a dataBase dir optionally
// carrying the runtime-cache active tree. Written files are tiny release
// documents ({"revision":N,...}); the extra note key keeps the two copies'
// BYTES different so a content-blind resolver could never pass the tie arm.
type effFixture struct {
	root       string // the install root ("" for a bare-BuiltinFile install)
	builtin    string // the install's discovery-recorded BuiltinFile
	dataBase   string // the runtime-cache base dir ("" = no cache at all)
	bundledRev int    // rev recorded in the root resources copy (0 = not written)
	activeRev  int    // rev recorded in the cache copy (0 = not written)
	// activeVerDir is the cache <appVersion> dir name ("" = no version dir).
	activeVerDir string
	// activeEndpoint is the endpoint dir name under the version dir
	// ("" = the version dir exists but is empty).
	activeEndpoint string
	// activeEndpointEmpty keeps the endpoint dir but omits its
	// zcode-builtin.json (the dir-exists-file-missing vector).
	activeEndpointEmpty bool
	// extraEndpoints are additional endpoint dir names written BEFORE the
	// primary one (multi-endpoint vectors).
	extraEndpoints map[string]int // name → rev written inside (0 = dir only)
}

// writeEffFixture materializes the fixture tree and returns the install to
// resolve. The discovery-recorded BuiltinFile always points at the runtime
// cache copy when one exists (the field shape: discovery records the
// runtime-tree builtin, NOT the install's bundled copy — exactly the gap
// this resolver fixes).
func writeEffFixture(t *testing.T, fx effFixture) supervisor.Install {
	t.Helper()

	inst := supervisor.Install{BuiltinFile: fx.builtin}

	if fx.root != "" {
		inst.Root = fx.root
		inst.Binary = filepath.Join(fx.root, "zcode")
		inst.Entrypoint = filepath.Join(fx.root, "resources", "glm", "zcode.cjs")
	}

	if fx.bundledRev > 0 {
		bundled := filepath.Join(fx.root, "resources", "config", "provider", "zcode-builtin.json")
		if err := os.MkdirAll(filepath.Dir(bundled), 0o750); err != nil {
			t.Fatalf("mkdir bundled dir: %v", err)
		}
		if err := os.WriteFile(bundled,
			[]byte(`{"revision":`+strconv.Itoa(fx.bundledRev)+`,"note":"bundled"}`), 0o600); err != nil {
			t.Fatalf("write bundled builtin: %v", err)
		}
	}

	if fx.dataBase != "" && fx.activeVerDir != "" {
		verDir := filepath.Join(fx.dataBase, ".zcode", "v2", "runtime", "provider",
			"linux-x86_64", fx.activeVerDir)
		endpoints := map[string]int{}
		for name, rev := range fx.extraEndpoints {
			endpoints[name] = rev
		}
		if fx.activeEndpoint != "" {
			endpoints[fx.activeEndpoint] = fx.activeRev
		}
		for name, rev := range endpoints {
			dir := filepath.Join(verDir, name)
			if err := os.MkdirAll(dir, 0o750); err != nil {
				t.Fatalf("mkdir endpoint dir %s: %v", dir, err)
			}
			emptyVector := fx.activeEndpointEmpty && name == fx.activeEndpoint
			if emptyVector || rev <= 0 {
				continue // dir-exists-file-missing vector (or a bare extra dir)
			}
			if err := os.WriteFile(filepath.Join(dir, "zcode-builtin.json"),
				[]byte(`{"revision":`+strconv.Itoa(rev)+`,"note":"active"}`), 0o600); err != nil {
				t.Fatalf("write active builtin: %v", err)
			}
		}
		if fx.activeEndpoint != "" && !fx.activeEndpointEmpty {
			inst.BuiltinFile = filepath.Join(verDir, fx.activeEndpoint, "zcode-builtin.json")
		}
	}

	return inst
}

// TestEffectiveRevisionTieFallsToBundled locks vector (a) — THIS machine's
// recorded field case: bundled and active both carry revision 44 with
// different bytes, so the strictly-greater rule never fires and the BUNDLED
// path hash wins. The discovery-recorded BuiltinFile points at the active
// copy (the field shape) — the resolver must NOT hash it.
func TestEffectiveRevisionTieFallsToBundled(t *testing.T) {
	root := t.TempDir()
	dataBase := t.TempDir()

	inst := writeEffFixture(t, effFixture{
		root:           root,
		dataBase:       dataBase,
		bundledRev:     44,
		activeRev:      44,
		activeVerDir:   "3.18.2",
		activeEndpoint: "endpoint-78d7c3bef4024722642626fe3669a799",
	})

	bundledPath := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	want := supervisor.BuiltinRevision(bundledPath, 44)
	notWant := supervisor.BuiltinRevision(inst.BuiltinFile, 44)

	if want == notWant {
		t.Fatal("fixture degenerated: bundled and active path hashes equal")
	}

	got, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", dataBase)
	if err != nil {
		t.Fatalf("effective revision: %v", err)
	}

	if got != want {
		t.Fatalf("effective revision = %q, want the bundled candidate hash %q "+
			"(a revision tie must fall to bundled — the runtime's selectReleaseCandidate)", got, want)
	}
}

// TestEffectiveRevisionStrictlyNewerActive locks vector (b): the active
// cache copy at a strictly greater revision replaces the bundled baseline —
// the winner path hash is the ACTIVE copy's.
func TestEffectiveRevisionStrictlyNewerActive(t *testing.T) {
	root := t.TempDir()
	dataBase := t.TempDir()

	inst := writeEffFixture(t, effFixture{
		root:           root,
		dataBase:       dataBase,
		bundledRev:     43,
		activeRev:      44,
		activeVerDir:   "3.18.2",
		activeEndpoint: "endpoint-78d7c3bef4024722642626fe3669a799",
	})

	want := supervisor.BuiltinRevision(inst.BuiltinFile, 44)

	got, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", dataBase)
	if err != nil {
		t.Fatalf("effective revision: %v", err)
	}

	if got != want {
		t.Fatalf("effective revision = %q, want the strictly-newer active hash %q", got, want)
	}
}

// TestEffectiveRevisionNoActiveTree locks vector (c): no runtime cache at
// all (the CI/fake-tier shape — and every current test fixture) degrades to
// bundled-only. Two arms: an install WITH its own resources copy (the copy
// is the bundled candidate) and a bare-BuiltinFile install without one (the
// recorded file stays the candidate — the exact pre-change digest surface
// the existing goldens lock).
func TestEffectiveRevisionNoActiveTree(t *testing.T) {
	root := t.TempDir()

	inst := writeEffFixture(t, effFixture{root: root, bundledRev: 44, builtin: filepath.Join(root, "zcode-builtin.json")})

	bundledPath := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	got, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", t.TempDir())
	if err != nil {
		t.Fatalf("effective revision (resources copy): %v", err)
	}

	if want := supervisor.BuiltinRevision(bundledPath, 44); got != want {
		t.Fatalf("effective revision = %q, want bundled-only %q", got, want)
	}

	// Bare-BuiltinFile arm: no resources copy → the recorded file IS the
	// bundled candidate (old behavior byte-preserved).
	flat := t.TempDir()
	flatBuiltin := filepath.Join(flat, "zcode-builtin.json")
	if err := os.WriteFile(flatBuiltin, []byte(`{"revision":44}`), 0o600); err != nil {
		t.Fatalf("write flat builtin: %v", err)
	}

	bare := supervisor.Install{
		BuiltinFile: flatBuiltin,
		Root:        t.TempDir(), // a root WITHOUT the resources copy
	}
	got, err = supervisor.EffectiveBuiltinRevision(bare, "3.18.2", "")
	if err != nil {
		t.Fatalf("effective revision (bare builtin): %v", err)
	}

	if want := supervisor.BuiltinRevision(bare.BuiltinFile, 44); got != want {
		t.Fatalf("effective revision = %q, want the recorded-file digest %q (old behavior)", got, want)
	}
}

// TestEffectiveRevisionActiveDirWithoutFile locks vector (d): the version
// dir and endpoint dir exist but carry no zcode-builtin.json — bundled-only.
func TestEffectiveRevisionActiveDirWithoutFile(t *testing.T) {
	root := t.TempDir()
	dataBase := t.TempDir()

	inst := writeEffFixture(t, effFixture{
		root:                root,
		dataBase:            dataBase,
		bundledRev:          44,
		activeRev:           50,
		activeVerDir:        "3.18.2",
		activeEndpoint:      "endpoint-78d7c3bef4024722642626fe3669a799",
		activeEndpointEmpty: true,
	})

	bundledPath := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	got, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", dataBase)
	if err != nil {
		t.Fatalf("effective revision: %v", err)
	}

	if want := supervisor.BuiltinRevision(bundledPath, 44); got != want {
		t.Fatalf("effective revision = %q, want bundled-only %q (dir without file is no candidate)", got, want)
	}
}

// TestEffectiveRevisionMultipleEndpoints locks the documented assumption:
// several endpoint dirs under one version dir resolve to the
// lexicographically-LAST sorted name that carries the file — here the
// rev-45 copy under endpoint-zzz, whose strictly-newer revision then wins
// the selection outright (proving WHICH file was picked via its hash).
func TestEffectiveRevisionMultipleEndpoints(t *testing.T) {
	root := t.TempDir()
	dataBase := t.TempDir()

	inst := writeEffFixture(t, effFixture{
		root:           root,
		dataBase:       dataBase,
		bundledRev:     44,
		activeRev:      45,
		activeVerDir:   "3.18.2",
		activeEndpoint: "endpoint-zzz",
		extraEndpoints: map[string]int{"endpoint-aaa": 45},
	})

	want := supervisor.BuiltinRevision(
		filepath.Join(dataBase, ".zcode", "v2", "runtime", "provider", "linux-x86_64",
			"3.18.2", "endpoint-zzz", "zcode-builtin.json"), 45)

	got, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", dataBase)
	if err != nil {
		t.Fatalf("effective revision: %v", err)
	}

	if got != want {
		t.Fatalf("effective revision = %q, want the lexicographically-last endpoint's hash %q", got, want)
	}
}

// TestEffectiveRevisionEndpointLastWithoutFile locks the multi-endpoint
// fallback: the lexicographically-last endpoint dir exists but carries no
// file, so the earlier dir's copy is the active candidate.
func TestEffectiveRevisionEndpointLastWithoutFile(t *testing.T) {
	root := t.TempDir()
	dataBase := t.TempDir()

	inst := writeEffFixture(t, effFixture{
		root:           root,
		dataBase:       dataBase,
		bundledRev:     44,
		activeRev:      45,
		activeVerDir:   "3.18.2",
		activeEndpoint: "endpoint-aaa", // carries the file
		extraEndpoints: map[string]int{"endpoint-zzz": 0},
	})

	want := supervisor.BuiltinRevision(
		filepath.Join(dataBase, ".zcode", "v2", "runtime", "provider", "linux-x86_64",
			"3.18.2", "endpoint-aaa", "zcode-builtin.json"), 45)

	got, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", dataBase)
	if err != nil {
		t.Fatalf("effective revision: %v", err)
	}

	if got != want {
		t.Fatalf("effective revision = %q, want the last endpoint WITH a file %q", got, want)
	}
}

// TestEffectiveRevisionNoCandidateErrors locks the honest-error arm: neither
// candidate readable → a coded error (errors.Is ErrNoBuiltinCandidate),
// never a silently empty revision string.
func TestEffectiveRevisionNoCandidateErrors(t *testing.T) {
	root := t.TempDir()

	inst := writeEffFixture(t, effFixture{root: root, builtin: filepath.Join(root, "missing", "zcode-builtin.json")})

	_, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", t.TempDir())
	if !errors.Is(err, supervisor.ErrNoBuiltinCandidate) {
		t.Fatalf("err = %v, want ErrNoBuiltinCandidate", err)
	}
}

// TestEffectiveRevisionMalformedBundledFallsToActive locks the candidate
// validity rule: a bundled copy without a usable top-level revision number
// is not a candidate (V-SRC: the decode step rejects it) — the active copy
// serves alone.
func TestEffectiveRevisionMalformedBundledFallsToActive(t *testing.T) {
	root := t.TempDir()
	dataBase := t.TempDir()

	bundled := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	if err := os.MkdirAll(filepath.Dir(bundled), 0o750); err != nil {
		t.Fatalf("mkdir bundled dir: %v", err)
	}
	if err := os.WriteFile(bundled, []byte(`{"schemaVersion":1}`), 0o600); err != nil {
		t.Fatalf("write malformed bundled: %v", err)
	}

	inst := writeEffFixture(t, effFixture{
		root:           root,
		dataBase:       dataBase,
		activeRev:      44,
		activeVerDir:   "3.18.2",
		activeEndpoint: "endpoint-78d7c3bef4024722642626fe3669a799",
	})

	want := supervisor.BuiltinRevision(inst.BuiltinFile, 44)

	got, err := supervisor.EffectiveBuiltinRevision(inst, "3.18.2", dataBase)
	if err != nil {
		t.Fatalf("effective revision: %v", err)
	}

	if got != want {
		t.Fatalf("effective revision = %q, want the active hash %q (malformed bundled is no candidate)", got, want)
	}
}

// TestInstallAppVersionVersion locks the appVersion dig: the version-looking
// segment of the install paths, deepest segment first — and "" when no
// segment matches (which disables the active candidate entirely).
func TestInstallAppVersionVersion(t *testing.T) {
	root := filepath.Join(t.TempDir(), "zcode-3.18.2", "squashfs-root")
	inst := supervisor.Install{
		Root:       root,
		Binary:     filepath.Join(root, "zcode"),
		Entrypoint: filepath.Join(root, "resources", "glm", "zcode.cjs"),
	}

	if got := supervisor.InstallAppVersion(inst); got != "3.18.2" {
		t.Fatalf("InstallAppVersion = %q, want 3.18.2 (digged from the root's version dir)", got)
	}

	plain := supervisor.Install{Root: t.TempDir()}
	if got := supervisor.InstallAppVersion(plain); got != "" {
		t.Fatalf("InstallAppVersion = %q, want \"\" (no version-looking segment)", got)
	}
}

// TestDefaultEffectiveRevisionBundledOnly locks the production composition:
// the default source derives the appVersion from the install and the
// dataBaseDir from the user home. For a fixture whose paths carry no
// version-looking segment the active candidate is unreachable BY CONSTRUCTION
// (empty appVersion — no cache lookup at all, machine-independent), so the
// default resolves bundled-only over the install's resources copy.
func TestDefaultEffectiveRevisionBundledOnly(t *testing.T) {
	root := t.TempDir()

	inst := writeEffFixture(t, effFixture{root: root, bundledRev: 44, builtin: filepath.Join(root, "zcode-builtin.json")})

	bundledPath := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	got, err := supervisor.DefaultEffectiveRevision(inst)
	if err != nil {
		t.Fatalf("default effective revision: %v", err)
	}

	if want := supervisor.BuiltinRevision(bundledPath, 44); got != want {
		t.Fatalf("default effective revision = %q, want bundled-only %q", got, want)
	}

	if !strings.HasPrefix(got, "zcode-builtin:44:") {
		t.Fatalf("effective revision %q lacks the zcode-builtin:<rev>: form", got)
	}
}
