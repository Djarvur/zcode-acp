package supervisor_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Contract tests for the seamed install discovery (BRIDGE-03 part 1, phase 10
// plan 03's version-ranked AppImage stage). Source of truth:
// docs/protocol/install-discovery.md — §Layout matrix (:15-35), the
// entrypoint invariant (:38), the desktop-entry procedure (:65-69), the
// version formats (:33), the revision formula (:82), §Recommended discovery
// order (:89-97). Every fixture is a t.TempDir() tree and every OS contact is
// a stubbed seam — no test reads the real machine layout.

// fixtureEndpointDir mirrors the documented runtime-provider fact that every
// version dir shares ONE endpoint dir name (install-discovery.md :77).
const fixtureEndpointDir = "endpoint-78d7c3bef4024722642626fe3669a799"

// fixtureUnpackedLayout and fixtureUnpackedVer name the stale research-copy
// layout the version ranking must beat (D-06).
const (
	fixtureUnpackedLayout = "unpacked-appimage"
	fixtureUnpackedVer    = "3.18.2"
)

// documentedProbeNames returns the fixed probe set in the documented order
// (install-discovery.md :93-97) extended with the runtime-provider-tree row
// the spawn line needs and its install-side bundled fallback (the QA-02
// fresh-machine completion, EXP-VM-QA02); diagnostics preserve this order.
// The appimage-candidate rows are variable and live between appimage-rank and
// dpkg, so they are excluded here.
func documentedProbeNames() []string {
	return []string{
		"override", "xdg-entry", "appimage-rank",
		"dpkg", "opt-path", "rpm", "builtin-runtime", "builtin-bundled",
	}
}

// writeBundleTree lays down the layout-invariant spawn surface at root: the
// Electron binary marker and the glm entrypoint. The relative entrypoint path
// resources/glm/zcode.cjs under Root holds across all layouts
// (install-discovery.md :38).
func writeBundleTree(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "resources", "glm"), 0o750); err != nil {
		t.Fatalf("mkdir bundle tree: %v", err)
	}
	for _, rel := range []string{"zcode", filepath.Join("resources", "glm", "zcode.cjs")} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("#!zcode-bundle-marker\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

// writePackedAppImage plants an executable regular packed AppImage at path —
// the file-level checks D-12 requires before a candidate can win.
func writePackedAppImage(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir packed AppImage dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("appimage-payload"), 0o755); err != nil {
		t.Fatalf("write packed AppImage %s: %v", path, err)
	}
}

// writeAppImageSymlink plants a symlink at link pointing at target (a
// relative target mirrors the user-facing ZCode.AppImage entry,
// install-discovery.md row 2).
func writeAppImageSymlink(t *testing.T, link, target string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %s -> %s: %v", link, target, err)
	}
}

// writeDesktopEntry plants a freedesktop entry naming execBinary as its Exec
// target under applicationsDir — the documented URL-handler clash surface
// (install-discovery.md :57-69).
func writeDesktopEntry(t *testing.T, applicationsDir, entryName, execBinary string) {
	t.Helper()
	if err := os.MkdirAll(applicationsDir, 0o750); err != nil {
		t.Fatalf("mkdir applications dir: %v", err)
	}
	const entryFormat = "[Desktop Entry]\nType=Application\nName=ZCode\nExec=%s %%U\nMimeType=x-scheme-handler/zcode;\n"
	entryBody := fmt.Sprintf(entryFormat, execBinary)
	entryPath := filepath.Join(applicationsDir, entryName)
	if err := os.WriteFile(entryPath, []byte(entryBody), 0o600); err != nil {
		t.Fatalf("write desktop entry: %v", err)
	}
}

// writeRuntimeTree plants the runtime-provider tree
// (~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/<endpoint>/zcode-builtin.json)
// with one version dir per version and the matching revision values
// (install-discovery.md :77). Returns the provider root.
func writeRuntimeTree(t *testing.T, home string, versions []string, revs []int) string {
	t.Helper()
	provider := filepath.Join(home, ".zcode", "v2", "runtime", "provider", "linux-x86_64")
	for i, ver := range versions {
		dir := filepath.Join(provider, ver, fixtureEndpointDir)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatalf("mkdir runtime version dir %s: %v", ver, err)
		}
		body := fmt.Sprintf(`{"schemaVersion":1,"revision":%d}`, revs[i])
		if err := os.WriteFile(filepath.Join(dir, "zcode-builtin.json"), []byte(body), 0o600); err != nil {
			t.Fatalf("write zcode-builtin.json for %s: %v", ver, err)
		}
	}

	return provider
}

// writeBundledBuiltin plants the bundled zcode-builtin.json inside the
// install itself (resources/config/provider/zcode-builtin.json under the
// bundle root) — the copy every shipped layout carries; field-verified in the
// QA-02 VM's squashfs-root (EXP-VM-QA02). Returns the planted path.
func writeBundledBuiltin(t *testing.T, root string, rev int) string {
	t.Helper()
	path := filepath.Join(root, "resources", "config", "provider", "zcode-builtin.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir bundled provider dir: %v", err)
	}
	body := fmt.Sprintf(`{"schemaVersion":1,"revision":%d}`, rev)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write bundled zcode-builtin.json: %v", err)
	}

	return path
}

// probeRows returns every diagnostics row carrying name.
func probeRows(probes []supervisor.Probe, name string) []supervisor.Probe {
	rows := make([]supervisor.Probe, 0, 1)
	for _, row := range probes {
		if row.Name == name {
			rows = append(rows, row)
		}
	}

	return rows
}

// probeRow finds the first diagnostics row for one probe name.
func probeRow(probes []supervisor.Probe, name string) (supervisor.Probe, bool) {
	rows := probeRows(probes, name)
	if len(rows) == 0 {
		return supervisor.Probe{}, false
	}

	return rows[0], true
}

// assertProbeOrder fails unless the diagnostics rows carry exactly the fixed
// probe set in the documented order. The appimage-candidate block is variable
// but must remain contiguous between appimage-rank and dpkg (D-09).
func assertProbeOrder(t *testing.T, probes []supervisor.Probe) {
	t.Helper()

	names := documentedProbeNames()
	fixed := make([]string, 0, len(names))
	rankIndex, dpkgIndex := -1, -1
	candidates := make([]int, 0, len(probes))

	for i, row := range probes {
		switch row.Name {
		case "appimage-candidate":
			candidates = append(candidates, i)
		case "appimage-rank":
			rankIndex = i
			fixed = append(fixed, row.Name)
		case "dpkg":
			dpkgIndex = i
			fixed = append(fixed, row.Name)
		default:
			fixed = append(fixed, row.Name)
		}
	}

	if len(fixed) != len(names) {
		t.Fatalf("want %d fixed probe rows, got %d: %+v", len(names), len(fixed), probes)
	}

	for i, name := range names {
		if fixed[i] != name {
			t.Fatalf("diagnostics row %d = %q, want %q (documented order is stable)", i, fixed[i], name)
		}
	}

	for _, index := range candidates {
		if index <= rankIndex || index >= dpkgIndex {
			t.Fatalf("appimage-candidate row %d must sit between appimage-rank (%d) and dpkg (%d)",
				index, rankIndex, dpkgIndex)
		}
	}
}

// assertCandidateMiss fails unless a candidate row names display as a miss
// whose failing step contains step.
func assertCandidateMiss(t *testing.T, probes []supervisor.Probe, display, step string) {
	t.Helper()

	for _, row := range probeRows(probes, "appimage-candidate") {
		if strings.Contains(row.Result, display) && strings.Contains(row.Result, "miss:") &&
			strings.Contains(row.Result, step) {
			return
		}
	}

	t.Fatalf("a candidate miss row must name %q and the failing step %q: %+v", display, step, probes)
}

// snapshotTree records every path under root with its mode and size — the
// pure-read assertion's before/after comparison.
func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()

	snapshot := make(map[string]string)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		snapshot[path] = fmt.Sprintf("%v|%d", info.Mode(), info.Size())

		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}

	return snapshot
}

// noHitSeams returns mutators stubbing every OS-query seam to an empty
// machine: no override, no default handler, no packages. AppDirs is left to
// the caller (it is the injected-candidates surface, not an OS query).
func noHitSeams() []func(*supervisor.Probers) {
	return []func(*supervisor.Probers){
		func(p *supervisor.Probers) {
			p.Override = func() (supervisor.Install, bool) { return supervisor.Install{}, false }
		},
		func(p *supervisor.Probers) { p.XDGQuery = func(context.Context) (string, error) { return "", nil } },
		func(p *supervisor.Probers) { p.DpkgQuery = func(context.Context) (string, error) { return "", nil } },
		func(p *supervisor.Probers) { p.RpmQuery = func(context.Context) (string, error) { return "", nil } },
	}
}

// discoverApps runs discovery over an empty machine seeded with the given
// application-dir candidates under a fresh home.
func discoverApps(t *testing.T, apps []string) (supervisor.Install, []supervisor.Probe, error) {
	t.Helper()

	return discoverAppsHome(t, t.TempDir(), apps)
}

// discoverAppsHome is discoverApps with an explicit home (the runtime-tree
// fixture).
func discoverAppsHome(t *testing.T, home string, apps []string) (supervisor.Install, []supervisor.Probe, error) {
	t.Helper()

	p := supervisor.Probers{
		Home:    home,
		OptDir:  filepath.Join(t.TempDir(), "ZCode"), // absent opt tree
		AppDirs: func() []string { return apps },
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}

	return p.Discover(context.Background())
}

// discoverUnpackedFixture seeds the unpacked-AppImage layout (matrix row 1:
// ~/Applications/zcode-*/squashfs-root) and asserts it resolves with the
// version parsed from the zcode-<ver> parent.
func discoverUnpackedFixture(t *testing.T) {
	t.Helper()

	root := filepath.Join(t.TempDir(), "Applications", "zcode-3.18.2", "squashfs-root")
	writeBundleTree(t, root)

	install, probes, err := discoverApps(t, []string{root})
	if err != nil {
		t.Fatalf("unpacked squashfs-root tree must resolve: %v", err)
	}

	want := supervisor.Install{
		Root:       root,
		Binary:     filepath.Join(root, "zcode"),
		Entrypoint: filepath.Join(root, "resources", "glm", "zcode.cjs"),
		Layout:     fixtureUnpackedLayout,
		Version:    fixtureUnpackedVer,
	}
	if install != want {
		t.Fatalf("install mismatch:\n got %+v\nwant %+v", install, want)
	}

	row, ok := probeRow(probes, "appimage-rank")
	if !ok || !strings.Contains(row.Result, root) || !strings.Contains(row.Result, fixtureUnpackedVer) {
		t.Errorf("appimage-rank row must name the resolved root and version: %+v", row)
	}
}

// discoverAppImageFixture seeds the packed-AppImage layout (matrix row 2:
// versioned file + unversioned symlink) and asserts the resolved packed
// install wins without extraction.
func discoverAppImageFixture(t *testing.T) {
	t.Helper()

	apps := filepath.Join(t.TempDir(), "Applications")
	versioned := filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage")
	writePackedAppImage(t, versioned)

	symlink := filepath.Join(apps, "ZCode.AppImage")
	writeAppImageSymlink(t, symlink, filepath.Base(versioned))

	install, probes, err := discoverApps(t, []string{versioned, symlink})
	if err != nil {
		t.Fatalf("packed AppImage must resolve as a mount-first install: %v", err)
	}

	want := supervisor.Install{
		AppImage: versioned,
		Version:  fixtureUnpackedVer,
		Layout:   testAppImageLayout,
	}
	if install != want {
		t.Fatalf("packed install mismatch:\n got %+v\nwant %+v", install, want)
	}
	if install.Root != "" || install.Binary != "" || install.Entrypoint != "" {
		t.Errorf("a packed install must carry empty Root/Binary/Entrypoint: %+v", install)
	}
	if !install.RequiresMount() {
		t.Error("a packed install must require the managed mount")
	}

	row, ok := probeRow(probes, "appimage-rank")
	if !ok || !strings.Contains(row.Result, "selected: appimage version 3.18.2") {
		t.Errorf("appimage-rank row must name layout and version: %+v", row)
	}
}

// discoverDebFixture seeds the deb layout (matrix row 3: dpkg file list
// naming the opt tree) and asserts it resolves with the invariant entrypoint.
func discoverDebFixture(t *testing.T) {
	t.Helper()

	optDir := filepath.Join(t.TempDir(), "ZCode")
	writeBundleTree(t, optDir)
	fileList := strings.Join([]string{
		optDir,
		filepath.Join(optDir, "zcode"),
		filepath.Join(optDir, "resources", "glm", "zcode.cjs"),
	}, "\n")

	p := supervisor.Probers{
		Home:      t.TempDir(),
		OptDir:    optDir,
		DpkgQuery: func(context.Context) (string, error) { return fileList, nil },
	}
	install, probes, err := p.Discover(context.Background())
	if err != nil {
		t.Fatalf("deb-style opt tree must resolve: %v", err)
	}

	want := supervisor.Install{
		Root:       optDir,
		Binary:     filepath.Join(optDir, "zcode"),
		Entrypoint: filepath.Join(optDir, "resources", "glm", "zcode.cjs"),
		Layout:     fixtureLayout,
	}
	if install != want {
		t.Fatalf("install mismatch:\n got %+v\nwant %+v", install, want)
	}

	row, ok := probeRow(probes, "dpkg")
	if !ok || !strings.Contains(row.Result, optDir) {
		t.Errorf("dpkg row must name the opt tree: %+v", row)
	}
}

func TestDiscoverFixtureTree(t *testing.T) {
	t.Run(fixtureUnpackedLayout, discoverUnpackedFixture)
	t.Run(testAppImageLayout, discoverAppImageFixture)
	t.Run(fixtureLayout, discoverDebFixture)
}

// TestDiscoverRankPackedOverStaleUnpacked pins DISC-01/D-06: the version, not
// the layout order, decides — a current packed 4.0.0 beats a stale unpacked
// 3.18.2 tree sitting beside it.
func TestDiscoverRankPackedOverStaleUnpacked(t *testing.T) {
	apps := filepath.Join(t.TempDir(), "Applications")
	packed := filepath.Join(apps, "ZCode-4.0.0-linux-x64.AppImage")
	staleRoot := filepath.Join(apps, "zcode-3.18.2", "squashfs-root")
	writePackedAppImage(t, packed)
	writeBundleTree(t, staleRoot)

	install, probes, err := discoverApps(t, []string{staleRoot, packed})
	if err != nil {
		t.Fatalf("both AppImage-layout candidates must resolve: %v", err)
	}

	want := supervisor.Install{AppImage: packed, Version: fakeAppImageVer, Layout: testAppImageLayout}
	if install != want {
		t.Fatalf("the newer packed candidate must win:\n got %+v\nwant %+v", install, want)
	}
	if !install.RequiresMount() {
		t.Error("the packed winner must require the managed mount")
	}

	row, ok := probeRow(probes, "appimage-rank")
	if !ok || !strings.Contains(row.Result, "selected: appimage version 4.0.0") ||
		!strings.Contains(row.Result, packed) {
		t.Fatalf("appimage-rank must name the selected layout, version, and resolved path: %+v", row)
	}
}

// TestDiscoverPerCandidateMisses pins DISC-02/D-07: every candidate is
// evaluated; a dangling symlink or invalid tree is a miss row and the scan
// continues.
func TestDiscoverPerCandidateMisses(t *testing.T) {
	t.Run("dangling symlink", func(t *testing.T) {
		apps := filepath.Join(t.TempDir(), "Applications")
		packed := filepath.Join(apps, "ZCode-4.0.0-linux-x64.AppImage")
		writePackedAppImage(t, packed)
		dangling := filepath.Join(apps, "ZCode.AppImage")
		writeAppImageSymlink(t, dangling, filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage"))

		install, probes, err := discoverApps(t, []string{dangling, packed})
		if err != nil {
			t.Fatalf("a dangling candidate must not abort discovery: %v", err)
		}
		if install.AppImage != packed || install.Version != fakeAppImageVer {
			t.Fatalf("the versioned 4.0.0 file must win: %+v", install)
		}
		assertCandidateMiss(t, probes, dangling, "unresolvable")
	})

	t.Run("invalid tree", func(t *testing.T) {
		apps := filepath.Join(t.TempDir(), "Applications")
		packed := filepath.Join(apps, "ZCode-4.0.0-linux-x64.AppImage")
		writePackedAppImage(t, packed)
		invalidRoot := filepath.Join(apps, "zcode-3.18.2", "squashfs-root")
		if err := os.MkdirAll(invalidRoot, 0o750); err != nil {
			t.Fatalf("mkdir invalid tree: %v", err)
		}

		install, probes, err := discoverApps(t, []string{invalidRoot, packed})
		if err != nil {
			t.Fatalf("an invalid tree must not abort discovery: %v", err)
		}
		if install.AppImage != packed || install.Version != fakeAppImageVer {
			t.Fatalf("the packed 4.0.0 file must win: %+v", install)
		}
		assertCandidateMiss(t, probes, invalidRoot, "spawnable tree")
	})
}

// TestDiscoverUnpackedOnly pins the unpacked-only path: it resolves through
// bundleInstall with the version from the zcode-<ver> parent dir.
func TestDiscoverUnpackedOnly(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Applications", "zcode-4.0.0", "squashfs-root")
	writeBundleTree(t, root)

	install, _, err := discoverApps(t, []string{root})
	if err != nil {
		t.Fatalf("an unpacked-only fixture must resolve: %v", err)
	}

	want := supervisor.Install{
		Root:       root,
		Binary:     filepath.Join(root, "zcode"),
		Entrypoint: filepath.Join(root, "resources", "glm", "zcode.cjs"),
		Layout:     fixtureUnpackedLayout,
		Version:    fakeAppImageVer,
	}
	if install != want {
		t.Fatalf("unpacked install mismatch:\n got %+v\nwant %+v", install, want)
	}
	if install.RequiresMount() {
		t.Error("an unpacked install must not require a mount")
	}
}

// TestDiscoverTieBreak pins all four D-08/DISC-04 rules at equal versions:
// unpacked beats packed, a packed symlink beats a packed regular file, two
// regular files break lexicographically, and a symlink plus its target are
// one candidate keyed by the resolved path.
func TestDiscoverTieBreak(t *testing.T) {
	t.Run("unpacked beats packed", tieBreakUnpacked)
	t.Run("symlink beats regular file", tieBreakSymlink)
	t.Run("regular files break lexicographically", tieBreakLexicographic)
	t.Run("symlink and target are one candidate", tieBreakDedupe)
}

// tieBreakUnpacked pins rule 1: at equal versions the spawnable unpacked tree
// wins because it needs no mount.
func tieBreakUnpacked(t *testing.T) {
	t.Helper()

	apps := filepath.Join(t.TempDir(), "Applications")
	packed := filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage")
	unpacked := filepath.Join(apps, "zcode-3.18.2", "squashfs-root")
	writePackedAppImage(t, packed)
	writeBundleTree(t, unpacked)

	install, _, err := discoverApps(t, []string{packed, unpacked})
	if err != nil {
		t.Fatalf("equal-version candidates must resolve: %v", err)
	}
	if install.Root != unpacked || install.Layout != fixtureUnpackedLayout {
		t.Fatalf("the unpacked tree must win at equal versions: %+v", install)
	}
}

// tieBreakSymlink pins rule 2: the user-facing symlink candidate outranks a
// packed regular file at the same version.
func tieBreakSymlink(t *testing.T) {
	t.Helper()

	apps := filepath.Join(t.TempDir(), "Applications")
	x64 := filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage")
	arm64 := filepath.Join(apps, "ZCode-3.18.2-linux-arm64.AppImage")
	link := filepath.Join(apps, "ZCode.AppImage")
	writePackedAppImage(t, x64)
	writePackedAppImage(t, arm64)
	writeAppImageSymlink(t, link, filepath.Base(x64))

	install, probes, err := discoverApps(t, []string{arm64, x64, link})
	if err != nil {
		t.Fatalf("equal-version packed candidates must resolve: %v", err)
	}
	if install.AppImage != x64 {
		t.Fatalf("the symlink's resolved target must win over the regular arm64 file: %+v", install)
	}

	rows := probeRows(probes, "appimage-candidate")
	if len(rows) != 2 {
		t.Fatalf("a symlink and its target must collapse into one candidate, plus arm64: got %d rows: %+v",
			len(rows), rows)
	}
	if !strings.Contains(rows[0].Result, link) || !strings.Contains(rows[0].Result, "hit:") {
		t.Fatalf("the winning candidate must be represented by the symlink display: %+v", rows[0])
	}
}

// tieBreakLexicographic pins rule 3: two regular files at the same version
// break by display path.
func tieBreakLexicographic(t *testing.T) {
	t.Helper()

	apps := filepath.Join(t.TempDir(), "Applications")
	arm64 := filepath.Join(apps, "ZCode-3.18.2-linux-arm64.AppImage")
	x64 := filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage")
	writePackedAppImage(t, arm64)
	writePackedAppImage(t, x64)

	install, _, err := discoverApps(t, []string{x64, arm64})
	if err != nil {
		t.Fatalf("equal-version regular files must resolve: %v", err)
	}
	if install.AppImage != arm64 {
		t.Fatalf("the lexicographically smaller display must win: got %q, want %q",
			install.AppImage, arm64)
	}
}

// tieBreakDedupe pins A14: a symlink and its resolved target are one
// candidate row.
func tieBreakDedupe(t *testing.T) {
	t.Helper()

	apps := filepath.Join(t.TempDir(), "Applications")
	target := filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage")
	link := filepath.Join(apps, "ZCode.AppImage")
	writePackedAppImage(t, target)
	writeAppImageSymlink(t, link, filepath.Base(target))

	install, probes, err := discoverApps(t, []string{link, target})
	if err != nil {
		t.Fatalf("a symlink plus its target must resolve: %v", err)
	}
	if install.AppImage != target {
		t.Fatalf("the symlink and target are one backend: %+v", install)
	}
	if rows := probeRows(probes, "appimage-candidate"); len(rows) != 1 {
		t.Fatalf("the symlink and target must be one candidate row: %+v", rows)
	}
}

// TestDiscoverVersionUnknown pins D-12's parseable-version requirement: an
// unversioned packed file and an unpacked tree under a non-zcode-<ver> parent
// are misses, never a last-resort winner.
func TestDiscoverVersionUnknown(t *testing.T) {
	t.Run("renamed packed file", func(t *testing.T) {
		renamed := filepath.Join(t.TempDir(), "Applications", "Renamed.AppImage")
		writePackedAppImage(t, renamed)

		_, probes, err := discoverApps(t, []string{renamed})
		if !errors.Is(err, supervisor.ErrNoInstall) {
			t.Fatalf("an unversioned packed file must not win: err %v", err)
		}
		if !strings.Contains(err.Error(), "version unknown") || !strings.Contains(err.Error(), renamed) {
			t.Fatalf("the diagnostics must name the version-unknown miss: %v", err)
		}
		assertCandidateMiss(t, probes, renamed, "version unknown")
	})

	t.Run("unpacked tree outside the zcode-<ver> shape", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), "Applications", "mystery", "squashfs-root")
		writeBundleTree(t, root)

		_, probes, err := discoverApps(t, []string{root})
		if !errors.Is(err, supervisor.ErrNoInstall) {
			t.Fatalf("an unpacked tree without a zcode-<ver> parent must not win: err %v", err)
		}
		if !strings.Contains(err.Error(), "version unknown") {
			t.Fatalf("the diagnostics must record the version-unknown miss: %v", err)
		}
		assertCandidateMiss(t, probes, root, "version unknown")
	})
}

// TestDiscoverPureRead pins D-12: discovery mounts nothing and creates no
// directory — the scanned tree is byte-identical before and after, and no
// .mount_* entry appears.
func TestDiscoverPureRead(t *testing.T) {
	apps := filepath.Join(t.TempDir(), "Applications")
	packed := filepath.Join(apps, "ZCode-4.0.0-linux-x64.AppImage")
	staleRoot := filepath.Join(apps, "zcode-3.18.2", "squashfs-root")
	writePackedAppImage(t, packed)
	writeBundleTree(t, staleRoot)

	before := snapshotTree(t, apps)

	install, _, err := discoverApps(t, []string{packed, staleRoot})
	if err != nil {
		t.Fatalf("the pure-read fixture must resolve: %v", err)
	}
	if install.AppImage != packed {
		t.Fatalf("the packed 4.0.0 winner must resolve: %+v", install)
	}

	after := snapshotTree(t, apps)
	if !maps.Equal(before, after) {
		t.Fatalf("discovery must not mutate the scanned tree:\n before %v\n after  %v", before, after)
	}

	for path := range after {
		if strings.HasPrefix(filepath.Base(path), ".mount_") {
			t.Fatalf("discovery must not create a mount entry: %s", path)
		}
	}
}

func TestDiscoverNoInstallListsEveryProbe(t *testing.T) {
	p := supervisor.Probers{
		Home:    t.TempDir(),
		OptDir:  filepath.Join(t.TempDir(), "ZCode"), // absent opt tree
		AppDirs: func() []string { return nil },      // empty applications scan
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}

	_, probes, err := p.Discover(context.Background())
	if err == nil {
		t.Fatal("empty machine is a first-class outcome: Discover must error with diagnostics")
	}

	assertProbeOrder(t, probes)

	for _, name := range documentedProbeNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error diagnostics must name probe %q: %v", name, err)
		}
	}
}

func TestDiscoverOrderFirstMatchWins(t *testing.T) {
	home := t.TempDir()
	entryTree := t.TempDir()                       // the opt-style tree the desktop entry points at (probe 2)
	optTree := filepath.Join(t.TempDir(), "ZCode") // the directly present opt tree (probe 6)
	writeBundleTree(t, entryTree)
	writeBundleTree(t, optTree)
	userEntries := filepath.Join(home, ".local", "share", "applications")
	writeDesktopEntry(t, userEntries, "zcode.desktop", filepath.Join(entryTree, "zcode"))

	p := supervisor.Probers{
		Home:      home,
		OptDir:    optTree,
		XDGQuery:  func(context.Context) (string, error) { return "zcode.desktop", nil },
		DpkgQuery: func(context.Context) (string, error) { return "", nil }, // rc remnant: empty list falls through
	}
	install, probes, err := p.Discover(context.Background())
	if err != nil {
		t.Fatalf("two adjacent probes yield valid installs: %v", err)
	}
	// The earlier probe in the documented order wins — xdg-entry is probe 2,
	// opt-path probe 6 — and adjacency never merges the two yields.
	if install.Root != entryTree {
		t.Fatalf("xdg-entry must win over opt-path: got root %q, want %q", install.Root, entryTree)
	}
	wantBinary := filepath.Join(entryTree, "zcode")
	wantEntrypoint := filepath.Join(entryTree, "resources", "glm", "zcode.cjs")
	if install.Binary != wantBinary || install.Entrypoint != wantEntrypoint || install.Layout != fixtureLayout {
		t.Fatalf("winner must be the entry-pointed tree:\n got %+v", install)
	}
	// Both attempts stay in the diagnostics (probes never merge).
	entryRow, ok := probeRow(probes, "xdg-entry")
	if !ok || !strings.Contains(entryRow.Result, "found") {
		t.Errorf("xdg-entry row must record its hit: %+v", entryRow)
	}
	optRow, ok := probeRow(probes, "opt-path")
	if !ok || !strings.Contains(optRow.Result, "found") {
		t.Errorf("opt-path row must record its hit too: %+v", optRow)
	}
}

func TestDiscoverRejectsVolatileMounts(t *testing.T) {
	// install-discovery.md row 5: /tmp/.mount_ZCode-<random> is do-not-rely —
	// volatile, exists only while the app runs. Synthetic decoy name; the
	// guard is prefix-based and no such mount is read.
	const mountDecoy = "/tmp/.mount_ZCode-e1f2a3b4"

	p := supervisor.Probers{
		Home:    t.TempDir(),
		OptDir:  filepath.Join(t.TempDir(), "ZCode"),
		AppDirs: func() []string { return []string{mountDecoy} },
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}

	install, probes, err := p.Discover(context.Background())
	if !errors.Is(err, supervisor.ErrNoInstall) || install.Root != "" {
		t.Fatalf("a volatile mount must never resolve as the install: install %+v, err %v", install, err)
	}

	assertProbeOrder(t, probes)

	row, ok := probeRow(probes, "appimage-rank")
	if !ok || !strings.Contains(row.Result, "no AppImage-layout candidate") {
		t.Fatalf("appimage-rank must record that no usable candidate exists: %+v", row)
	}

	found := false

	for _, cand := range probeRows(probes, "appimage-candidate") {
		if strings.Contains(cand.Result, mountDecoy) && strings.Contains(cand.Result, "reject") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the volatile mount must be a rejection candidate row: %+v", probes)
	}
}

// xdgDiscover runs discovery with a user-level desktop entry naming
// execTarget as its Exec target.
func xdgDiscover(
	t *testing.T, home, optDir string, apps []string, entryName, execTarget string,
) (supervisor.Install, []supervisor.Probe, error) {
	t.Helper()

	writeDesktopEntry(t, filepath.Join(home, ".local", "share", "applications"), entryName, execTarget)

	p := supervisor.Probers{
		Home:    home,
		OptDir:  optDir,
		AppDirs: func() []string { return apps },
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}
	p.XDGQuery = func(context.Context) (string, error) { return entryName, nil }

	return p.Discover(context.Background())
}

// TestDiscoverDiagnostics pins DISC-03/D-09: the rank row carries layout,
// version, and resolved path; one ordered candidate row per evaluated
// candidate carries the display path, version, and the failing step.
func TestDiscoverDiagnostics(t *testing.T) {
	apps := filepath.Join(t.TempDir(), "Applications")
	packed := filepath.Join(apps, "ZCode-4.0.0-linux-x64.AppImage")
	dangling := filepath.Join(apps, "ZCode.AppImage")
	invalidRoot := filepath.Join(apps, "zcode-3.18.2", "squashfs-root")
	writePackedAppImage(t, packed)
	writeAppImageSymlink(t, dangling, filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage"))
	if err := os.MkdirAll(invalidRoot, 0o750); err != nil {
		t.Fatalf("mkdir invalid tree: %v", err)
	}

	install, probes, err := discoverApps(t, []string{dangling, invalidRoot, packed})
	if err != nil {
		t.Fatalf("the mixed fixture must still resolve the packed hit: %v", err)
	}
	if install.AppImage != packed || install.Version != fakeAppImageVer {
		t.Fatalf("the packed 4.0.0 file must win: %+v", install)
	}

	rank, ok := probeRow(probes, "appimage-rank")
	if !ok {
		t.Fatal("the appimage-rank row must exist")
	}
	for _, want := range []string{"selected:", testAppImageLayout, fakeAppImageVer, packed} {
		if !strings.Contains(rank.Result, want) {
			t.Errorf("appimage-rank row must carry %q: %+v", want, rank)
		}
	}

	rows := probeRows(probes, "appimage-candidate")
	if len(rows) != 3 {
		t.Fatalf("one candidate row per evaluated candidate, got %d: %+v", len(rows), rows)
	}
	if !strings.HasPrefix(rows[0].Result, "hit:") || !strings.Contains(rows[0].Result, packed) ||
		!strings.Contains(rows[0].Result, fakeAppImageVer) {
		t.Fatalf("the first row must be the winning hit with version: %+v", rows[0])
	}
	for _, row := range rows[1:] {
		if !strings.HasPrefix(row.Result, "miss:") || !strings.Contains(row.Result, "version") {
			t.Fatalf("skipped candidates must be miss rows naming the version: %+v", row)
		}
	}
	assertCandidateMiss(t, probes, dangling, "unresolvable")
	assertCandidateMiss(t, probes, invalidRoot, "spawnable tree")
}

// TestDiscoverXDGAppImageSeed pins D-13: an XDG Exec target that resolves to
// an AppImage-family path joins the version ranking, while a package tree
// still wins at order step 2.
func TestDiscoverXDGAppImageSeed(t *testing.T) {
	t.Run("packed target outside the applications dir", xdgSeedPacked)
	t.Run("package tree keeps step-2 precedence", xdgSeedPackagePrecedence)
	t.Run("unpacked seed wins when it is the only candidate", xdgSeedUnpacked)
	t.Run("unpacked seed loses to a newer packed candidate", xdgSeedLosesToVersion)
}

// xdgSeedPacked pins the packed arm: an XDG target outside ~/Applications is
// reachable because it joins the ranked stage.
func xdgSeedPacked(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	packed := filepath.Join(home, "Downloads", "ZCode-4.0.0-linux-x64.AppImage")
	writePackedAppImage(t, packed)

	install, probes, err := xdgDiscover(t, home, filepath.Join(t.TempDir(), "ZCode"),
		nil, "zcode.desktop", packed)
	if err != nil {
		t.Fatalf("an XDG AppImage target must be a ranked candidate: %v", err)
	}
	if install.AppImage != packed || install.Version != fakeAppImageVer || install.Layout != testAppImageLayout {
		t.Fatalf("the seeded packed target must win: %+v", install)
	}

	row, ok := probeRow(probes, "xdg-entry")
	if !ok || !strings.Contains(row.Result, "fed to the AppImage ranking") {
		t.Fatalf("xdg-entry must record that the target joined the ranking: %+v", row)
	}
}

// xdgSeedPackagePrecedence pins D-10: a package tree still wins at step 2 and
// the AppImage stage is not consulted.
func xdgSeedPackagePrecedence(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	optDir := filepath.Join(t.TempDir(), "ZCode")
	writeBundleTree(t, optDir)
	packed := filepath.Join(home, "Applications", "ZCode-4.0.0-linux-x64.AppImage")
	writePackedAppImage(t, packed)

	install, probes, err := xdgDiscover(t, home, optDir, []string{packed},
		"zcode.desktop", filepath.Join(optDir, "zcode"))
	if err != nil {
		t.Fatalf("the package-tree entry must resolve at step 2: %v", err)
	}
	if install.Root != optDir || install.Layout != fixtureLayout || install.AppImage != "" {
		t.Fatalf("the package tree must beat the newer packed candidate: %+v", install)
	}

	rank, ok := probeRow(probes, "appimage-rank")
	if !ok || !strings.Contains(rank.Result, "not consulted; an earlier probe resolved the install") {
		t.Fatalf("the stage must record the earlier-probe precedence: %+v", rank)
	}
	if rows := probeRows(probes, "appimage-candidate"); len(rows) != 0 {
		t.Fatalf("a not-consulted stage must evaluate no candidates: %+v", rows)
	}
}

// xdgSeedUnpacked pins the unpacked arm: an XDG squashfs-root target outside
// ~/Applications is a candidate and wins when it is the only one.
func xdgSeedUnpacked(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	unpacked := filepath.Join(home, "Applications", "zcode-3.18.2", "squashfs-root")
	writeBundleTree(t, unpacked)

	install, probes, err := xdgDiscover(t, home, filepath.Join(t.TempDir(), "ZCode"),
		nil, "zcode.desktop", filepath.Join(unpacked, "zcode"))
	if err != nil {
		t.Fatalf("an XDG squashfs-root target must be a ranked candidate: %v", err)
	}
	if install.Root != unpacked || install.Layout != fixtureUnpackedLayout || install.Version != fixtureUnpackedVer {
		t.Fatalf("the seeded unpacked tree must win: %+v", install)
	}

	row, ok := probeRow(probes, "xdg-entry")
	if !ok || !strings.Contains(row.Result, "fed to the AppImage ranking") {
		t.Fatalf("xdg-entry must record that the tree joined the ranking: %+v", row)
	}
}

// xdgSeedLosesToVersion pins DISC-01/D-13: an older entry seed cannot pin an
// older backend when a newer packed candidate exists.
func xdgSeedLosesToVersion(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	unpacked := filepath.Join(home, "Applications", "zcode-3.18.2", "squashfs-root")
	packed := filepath.Join(home, "Applications", "ZCode-4.0.0-linux-x64.AppImage")
	writeBundleTree(t, unpacked)
	writePackedAppImage(t, packed)

	install, _, err := xdgDiscover(t, home, filepath.Join(t.TempDir(), "ZCode"),
		[]string{packed}, "zcode.desktop", filepath.Join(unpacked, "zcode"))
	if err != nil {
		t.Fatalf("both candidates must resolve: %v", err)
	}
	if install.AppImage != packed || install.Version != fakeAppImageVer {
		t.Fatalf("version ranking must beat the entry's older unpacked seed: %+v", install)
	}
}

// TestDiscoverBuiltinDeferredForPacked pins the two packed builtin arms: a
// runtime-cache hit keeps its literal, and no cache defers the bundled read
// until the mount (Pitfall 15), naming the mount-relative path.
func TestDiscoverBuiltinDeferredForPacked(t *testing.T) {
	t.Run("runtime cache wins", func(t *testing.T) {
		home := t.TempDir()
		packed := filepath.Join(home, "Applications", "ZCode-4.0.0-linux-x64.AppImage")
		writePackedAppImage(t, packed)
		writeRuntimeTree(t, home, []string{fakeAppImageVer}, []int{51})

		install, probes, err := discoverAppsHome(t, home, []string{packed})
		if err != nil {
			t.Fatalf("packed install plus runtime tree must resolve: %v", err)
		}

		wantBuiltin := filepath.Join(home, ".zcode", "v2", "runtime", "provider",
			"linux-x86_64", fakeAppImageVer, fixtureEndpointDir, "zcode-builtin.json")
		if install.BuiltinFile != wantBuiltin {
			t.Fatalf("the runtime-cache copy must win:\n got %q\nwant %q", install.BuiltinFile, wantBuiltin)
		}

		row, ok := probeRow(probes, "builtin-bundled")
		if !ok || !strings.Contains(row.Result, "not consulted; the runtime cache resolved the builtin file") {
			t.Fatalf("a cache hit keeps the pre-phase literal: %+v", row)
		}
		if strings.Contains(row.Result, "deferred until mount") {
			t.Fatalf("a cache hit must not defer: %+v", row)
		}
	})

	t.Run("no runtime cache defers to the mount", func(t *testing.T) {
		home := t.TempDir()
		packed := filepath.Join(home, "Applications", "ZCode-4.0.0-linux-x64.AppImage")
		writePackedAppImage(t, packed)

		install, probes, err := discoverAppsHome(t, home, []string{packed})
		if err != nil {
			t.Fatalf("the packed install must resolve without a runtime cache: %v", err)
		}
		if install.BuiltinFile != "" {
			t.Fatalf("a packed install cannot read the bundled copy before the mount: got %q",
				install.BuiltinFile)
		}

		row, ok := probeRow(probes, "builtin-bundled")
		if !ok || !strings.Contains(row.Result, "deferred until mount") {
			t.Fatalf("the packed empty arm must defer until the mount: %+v", row)
		}
		if !strings.Contains(row.Result, "<mount>/resources/config/provider/zcode-builtin.json") {
			t.Fatalf("the deferred row must name the mount-relative bundled path: %+v", row)
		}
	})
}

// TestDiscoverProbeOrder pins the fixed probe order with a variable
// appimage-candidate block between appimage-rank and dpkg.
func TestDiscoverProbeOrder(t *testing.T) {
	apps := filepath.Join(t.TempDir(), "Applications")
	packed := filepath.Join(apps, "ZCode-4.0.0-linux-x64.AppImage")
	staleRoot := filepath.Join(apps, "zcode-3.18.2", "squashfs-root")
	writePackedAppImage(t, packed)
	writeBundleTree(t, staleRoot)

	_, probes, err := discoverApps(t, []string{staleRoot, packed})
	if err != nil {
		t.Fatalf("the ordered fixture must resolve: %v", err)
	}

	assertProbeOrder(t, probes)

	if rows := probeRows(probes, "appimage-candidate"); len(rows) != 2 {
		t.Fatalf("both candidates must produce rows: %+v", rows)
	}
}

func TestBuiltinRevisionFormula(t *testing.T) {
	// install-discovery.md :82 — revision = zcode-builtin:<rev>:<sha256(
	// path.resolve(activeFilePath))>; the hash input is the resolved absolute
	// PATH STRING, never the file contents.
	dir := t.TempDir()
	builtin := filepath.Join(dir, "zcode-builtin.json")
	const body = `{"schemaVersion":1,"revision":44}`
	if err := os.WriteFile(builtin, []byte(body), 0o600); err != nil {
		t.Fatalf("write builtin file: %v", err)
	}

	got := supervisor.BuiltinRevision(builtin, 44)
	resolved, err := filepath.EvalSymlinks(builtin)
	if err != nil {
		t.Fatalf("eval path: %v", err)
	}
	pathSum := sha256.Sum256([]byte(resolved))
	want := fmt.Sprintf("zcode-builtin:%d:%s", 44, hex.EncodeToString(pathSum[:]))
	if got != want {
		t.Fatalf("revision formula: got %q, want %q (sha256 over resolved path string)", got, want)
	}
	contentSum := sha256.Sum256([]byte(body))
	if got == fmt.Sprintf("zcode-builtin:%d:%s", 44, hex.EncodeToString(contentSum[:])) {
		t.Fatal("revision must hash the path STRING, never the file contents")
	}

	// A symlinked active file resolves: the hash covers the REAL path.
	linkFile := symlinkedBuiltinFixture(t, dir, body)
	resolvedLink, err := filepath.EvalSymlinks(linkFile)
	if err != nil {
		t.Fatalf("eval symlink: %v", err)
	}
	linkSum := sha256.Sum256([]byte(resolvedLink))
	wantLink := fmt.Sprintf("zcode-builtin:%d:%s", 44, hex.EncodeToString(linkSum[:]))
	if got := supervisor.BuiltinRevision(linkFile, 44); got != wantLink {
		t.Fatalf("symlinked builtin must hash the resolved real path: got %q, want %q", got, wantLink)
	}
}

// symlinkedBuiltinFixture plants a real zcode-builtin.json plus a symlinked
// alias in a sibling endpoint dir and returns the symlink path.
func symlinkedBuiltinFixture(t *testing.T, dir, body string) string {
	t.Helper()
	realFile := filepath.Join(dir, "endpoint-real", "zcode-builtin.json")
	linkFile := filepath.Join(dir, "endpoint-link", "zcode-builtin.json")
	for _, path := range []string{realFile, linkFile} {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir endpoint dir: %v", err)
		}
	}
	if err := os.WriteFile(realFile, []byte(body), 0o600); err != nil {
		t.Fatalf("write real builtin: %v", err)
	}
	if err := os.Symlink(realFile, linkFile); err != nil {
		t.Fatalf("symlink builtin: %v", err)
	}

	return linkFile
}

func TestDiscoverFindsNewestBuiltin(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "Applications", "zcode-3.18.2", "squashfs-root")
	writeBundleTree(t, root)
	writeRuntimeTree(t, home, []string{"0.0.0-dev", "3.17.1", fixtureUnpackedVer}, []int{1, 43, 44})

	install, probes, err := discoverAppsHome(t, home, []string{root})
	if err != nil {
		t.Fatalf("root install plus runtime tree must resolve: %v", err)
	}

	wantBuiltin := filepath.Join(home, ".zcode", "v2", "runtime", "provider",
		"linux-x86_64", fixtureUnpackedVer, fixtureEndpointDir, "zcode-builtin.json")
	if install.BuiltinFile != wantBuiltin {
		t.Fatalf("builtin file under the highest version dir must win:\n got %q\nwant %q",
			install.BuiltinFile, wantBuiltin)
	}
	if install.Root != root || install.Layout != fixtureUnpackedLayout {
		t.Fatalf("root install must still resolve alongside the runtime tree: %+v", install)
	}

	row, ok := probeRow(probes, "builtin-runtime")
	if !ok || !strings.Contains(row.Result, fixtureUnpackedVer) {
		t.Fatalf("builtin-runtime row must name the picked version: %+v", row)
	}
}

func TestDiscoverBundledBuiltinFallback(t *testing.T) {
	// The QA-02 field failure (EXP-VM-QA02): a fresh machine has no
	// ~/.zcode/v2/runtime/provider cache, the builtin-runtime probe honestly
	// misses, and the winner's BuiltinFile came back empty — ChildEnv then
	// passes the personal path alone, the unpaired env form, and
	// resolveNodeProviderRuntimePaths kills the worker at startup. Discovery
	// completes order step 5 install-side: the BUNDLED copy under the winning
	// root (resources/config/provider/zcode-builtin.json — present in the
	// QA-02 VM's squashfs-root) feeds the builtin env path.
	home := t.TempDir()
	root := filepath.Join(home, "Applications", "zcode-3.18.2", "squashfs-root")
	writeBundleTree(t, root)
	bundled := writeBundledBuiltin(t, root, 44)

	install, probes, err := discoverAppsHome(t, home, []string{root})
	if err != nil {
		t.Fatalf("unpacked tree with bundled builtin must resolve: %v", err)
	}
	if install.BuiltinFile != bundled {
		t.Fatalf("fresh machine (no runtime cache) must fall back to the bundled builtin:\n got %q\nwant %q",
			install.BuiltinFile, bundled)
	}

	row, ok := probeRow(probes, "builtin-bundled")
	if !ok || !strings.Contains(row.Result, "found") || !strings.Contains(row.Result, bundled) {
		t.Errorf("builtin-bundled row must record the bundled hit naming the path: %+v", row)
	}
}

func TestDiscoverBuiltinStaysEmptyWithoutAnySource(t *testing.T) {
	// Neither source exists — no runtime-provider tree, no bundled copy under
	// the root: BuiltinFile stays empty and the install still resolves. The
	// empty result is first-class and honest (install-discovery.md :34): the
	// builtin-bundled diagnostics row carries the miss, never a guess.
	home := t.TempDir()
	root := filepath.Join(home, "Applications", "zcode-3.18.2", "squashfs-root")
	writeBundleTree(t, root)

	install, probes, err := discoverAppsHome(t, home, []string{root})
	if err != nil {
		t.Fatalf("the install itself must still resolve: %v", err)
	}
	if install.BuiltinFile != "" {
		t.Fatalf("no runtime cache and no bundled copy: BuiltinFile must stay empty, got %q",
			install.BuiltinFile)
	}

	row, ok := probeRow(probes, "builtin-bundled")
	if !ok || strings.Contains(row.Result, "found") {
		t.Errorf("builtin-bundled row must record the honest miss: %+v", row)
	}
}

func TestDiscoverRuntimeCacheWinsOverBundled(t *testing.T) {
	// Both sources present: the runtime-cache copy wins — the existing
	// precedence is unchanged and now locked (install-discovery.md :91: the
	// runtime's registry revision is the ACTIVE-cache path hash, never the
	// bundled resources copy's).
	home := t.TempDir()
	root := filepath.Join(home, "Applications", "zcode-3.18.2", "squashfs-root")
	writeBundleTree(t, root)
	writeBundledBuiltin(t, root, 44)
	writeRuntimeTree(t, home, []string{fixtureUnpackedVer}, []int{44})

	install, probes, err := discoverAppsHome(t, home, []string{root})
	if err != nil {
		t.Fatalf("root install plus runtime tree must resolve: %v", err)
	}

	wantBuiltin := filepath.Join(home, ".zcode", "v2", "runtime", "provider",
		"linux-x86_64", fixtureUnpackedVer, fixtureEndpointDir, "zcode-builtin.json")
	if install.BuiltinFile != wantBuiltin {
		t.Fatalf("the runtime-cache copy must win over the bundled copy:\n got %q\nwant %q",
			install.BuiltinFile, wantBuiltin)
	}

	row, ok := probeRow(probes, "builtin-bundled")
	if !ok || strings.Contains(row.Result, "found") {
		t.Errorf("bundled row must not record a hit when the runtime cache resolved: %+v", row)
	}
}

// TestOverrideInstall pins the full --app-server mapper (D-04, MNT-02): a
// directory keeps the explicit-root shape, a packed .AppImage (literal,
// symlinked target, or dangling) becomes an absolute mount-first install, and
// any other regular file keeps the app-server-binary shape. The probe rows
// must name the packed override and reject a volatile /tmp/.mount_* path.
func TestOverrideInstall(t *testing.T) {
	t.Run("empty path", testOverrideEmpty)
	t.Run("valid directory", testOverrideDirectoryValid)
	t.Run("unvalidated directory", testOverrideDirectoryUnvalidated)
	t.Run("literal AppImage", testOverrideLiteralAppImage)
	t.Run("relative literal AppImage", testOverrideRelativeAppImage)
	t.Run("symlink to AppImage", testOverrideSymlinkAppImage)
	t.Run("dangling AppImage", testOverrideDanglingAppImage)
	t.Run("non-AppImage binary", testOverrideBinary)
	t.Run("directory named like an AppImage", testOverrideAppImageDirectory)
	t.Run("probe row names the packed override", testOverrideProbePacked)
	t.Run("probe rejects a volatile AppImage path", testOverrideProbeVolatile)
}

func testOverrideEmpty(t *testing.T) {
	t.Helper()

	if inst, ok := supervisor.OverrideInstall(""); ok {
		t.Fatalf("an empty override must not resolve: %+v", inst)
	}
}

func testOverrideDirectoryValid(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	writeBundleTree(t, root)

	inst, ok := supervisor.OverrideInstall(root)
	if !ok {
		t.Fatal("a valid install root must resolve")
	}

	want := supervisor.Install{
		Root:       root,
		Binary:     filepath.Join(root, "zcode"),
		Entrypoint: filepath.Join(root, "resources", "glm", "zcode.cjs"),
		Layout:     "explicit",
	}
	if inst != want {
		t.Fatalf("directory override = %+v, want %+v", inst, want)
	}
}

func testOverrideDirectoryUnvalidated(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	inst, ok := supervisor.OverrideInstall(root)
	if !ok {
		t.Fatal("an existing directory must keep the unvalidated fallback shape")
	}
	if inst.Root != root || inst.Binary != filepath.Join(root, "zcode") ||
		inst.Entrypoint != "" || inst.Layout != "explicit" || inst.AppImage != "" {
		t.Fatalf("unvalidated directory override = %+v", inst)
	}
}

func testOverrideLiteralAppImage(t *testing.T) {
	t.Helper()

	packed := filepath.Join(t.TempDir(), "ZCode-4.0.0-linux-x64.AppImage")
	writePackedAppImage(t, packed)

	inst, ok := supervisor.OverrideInstall(packed)
	if !ok {
		t.Fatal("a literal AppImage override must resolve")
	}
	assertPackedOverride(t, inst, packed)
}

func testOverrideRelativeAppImage(t *testing.T) {
	t.Helper()

	packed := filepath.Join(t.TempDir(), "sub", "ZCode-4.0.0-linux-x64.AppImage")
	writePackedAppImage(t, packed)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	rel, err := filepath.Rel(wd, packed)
	if err != nil {
		t.Fatalf("rel %s: %v", packed, err)
	}

	inst, ok := supervisor.OverrideInstall(rel)
	if !ok {
		t.Fatal("a relative literal AppImage override must resolve")
	}
	assertPackedOverride(t, inst, packed)
}

func testOverrideSymlinkAppImage(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	target := filepath.Join(dir, "ZCode-4.0.0-linux-x64.AppImage")
	writePackedAppImage(t, target)
	link := filepath.Join(dir, "zcode-current")
	writeAppImageSymlink(t, link, target)

	inst, ok := supervisor.OverrideInstall(link)
	if !ok {
		t.Fatal("a symlink whose target is a packed AppImage must resolve")
	}
	assertPackedOverride(t, inst, target)
}

func testOverrideDanglingAppImage(t *testing.T) {
	t.Helper()

	missing := filepath.Join(t.TempDir(), "ZCode-4.0.0-linux-x64.AppImage")
	inst, ok := supervisor.OverrideInstall(missing)
	if !ok {
		t.Fatal("a missing AppImage must still map packed so the mount step fails loud")
	}
	assertPackedOverride(t, inst, missing)
}

func testOverrideBinary(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	bin := filepath.Join(dir, "zcode-app-server")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o750); err != nil {
		t.Fatalf("write binary override: %v", err)
	}

	inst, ok := supervisor.OverrideInstall(bin)
	if !ok {
		t.Fatal("a non-AppImage regular file override must resolve")
	}

	want := supervisor.Install{
		Root:       dir,
		Binary:     bin,
		Entrypoint: filepath.Join(dir, "resources", "glm", "zcode.cjs"),
		Layout:     "explicit",
	}
	if inst != want {
		t.Fatalf("binary override = %+v, want %+v", inst, want)
	}
}

func testOverrideAppImageDirectory(t *testing.T) {
	t.Helper()

	root := filepath.Join(t.TempDir(), "ZCode-4.0.0-linux-x64.AppImage")
	writeBundleTree(t, root)

	inst, ok := supervisor.OverrideInstall(root)
	if !ok {
		t.Fatalf("a directory override must resolve: %+v", inst)
	}
	if inst.Layout == testAppImageLayout || inst.AppImage != "" || inst.Root != root {
		t.Fatalf("a directory named *.AppImage must stay a tree (directory branch first): %+v", inst)
	}
}

func testOverrideProbePacked(t *testing.T) {
	t.Helper()

	packed := filepath.Join(t.TempDir(), "ZCode-4.0.0-linux-x64.AppImage")
	writePackedAppImage(t, packed)

	probes := overrideProbes(t, packed)
	row, ok := probeRow(probes, "override")
	if !ok {
		t.Fatal("the override probe row must exist")
	}
	for _, want := range []string{"found: packed AppImage", packed, "(version " + fakeAppImageVer + ")"} {
		if !strings.Contains(row.Result, want) {
			t.Fatalf("the packed override row must carry %q: %+v", want, row)
		}
	}
}

func testOverrideProbeVolatile(t *testing.T) {
	t.Helper()

	const volatile = "/tmp/.mount_ZCode-a1b2c3/ZCode-4.0.0-linux-x64.AppImage"

	probes := overrideProbes(t, volatile)
	row, ok := probeRow(probes, "override")
	if !ok {
		t.Fatal("the override probe row must exist")
	}
	if !strings.Contains(row.Result, "rejected volatile mount") || !strings.Contains(row.Result, volatile) {
		t.Fatalf("a volatile AppImage override must be rejected, naming the path: %+v", row)
	}
}

// assertPackedOverride fails unless inst is the mount-first packed shape for
// wantPath and the fixture version, stored absolute (never a bare basename).
func assertPackedOverride(t *testing.T, inst supervisor.Install, wantPath string) {
	t.Helper()

	if inst.Layout != testAppImageLayout || inst.AppImage != wantPath || inst.Version != fakeAppImageVer {
		t.Fatalf("packed override = %+v, want AppImage %s version %s layout %s",
			inst, wantPath, fakeAppImageVer, testAppImageLayout)
	}
	if inst.Root != "" || inst.Binary != "" || inst.Entrypoint != "" {
		t.Fatalf("a packed override must carry no pre-mount root/binary/entrypoint: %+v", inst)
	}
	if !filepath.IsAbs(inst.AppImage) || filepath.Base(inst.AppImage) == inst.AppImage {
		t.Fatalf("the stored AppImage must be absolute and never a bare basename: %q", inst.AppImage)
	}
}

// overrideProbes runs discovery over an empty machine whose only override is
// path, returning the diagnostics rows (a rejected override still ends in
// ErrNoInstall; a packed one resolves and wins).
func overrideProbes(t *testing.T, path string) []supervisor.Probe {
	t.Helper()

	p := supervisor.Probers{
		Home:    t.TempDir(),
		OptDir:  filepath.Join(t.TempDir(), "ZCode"),
		AppDirs: func() []string { return nil },
	}
	for _, seam := range noHitSeams() {
		seam(&p)
	}
	p.Override = func() (supervisor.Install, bool) { return supervisor.OverrideInstall(path) }

	_, probes, _ := p.Discover(context.Background())

	return probes
}
