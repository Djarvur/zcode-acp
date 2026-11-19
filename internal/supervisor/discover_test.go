package supervisor_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Djarvur/zcode-acp/internal/supervisor"
)

// Contract tests for the seamed install discovery (BRIDGE-03 part 1). Source
// of truth: docs/protocol/install-discovery.md — §Layout matrix (:15-35), the
// entrypoint invariant (:38), the desktop-entry procedure (:65-69), the
// revision formula (:82), §Recommended discovery order (:89-97). Every
// fixture is a t.TempDir() tree and every OS contact is a stubbed seam — no
// test reads the real machine layout.

// fixtureEndpointDir mirrors the documented runtime-provider fact that every
// version dir shares ONE endpoint dir name (install-discovery.md :77).
const fixtureEndpointDir = "endpoint-78d7c3bef4024722642626fe3669a799"

// documentedProbeNames returns the fixed probe set in the documented order
// (install-discovery.md :93-97) extended with the runtime-provider-tree row
// the spawn line needs; diagnostics preserve this order.
func documentedProbeNames() []string {
	return []string{
		"override", "xdg-entry", "appimage-glob", "unpacked-glob",
		"dpkg", "opt-path", "rpm", "builtin-runtime",
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

// probeRow finds the diagnostics row for one probe name.
func probeRow(probes []supervisor.Probe, name string) (supervisor.Probe, bool) {
	for _, row := range probes {
		if row.Name == name {
			return row, true
		}
	}

	return supervisor.Probe{}, false
}

// assertProbeOrder fails unless the diagnostics rows carry exactly the
// documented probe set, in the documented order.
func assertProbeOrder(t *testing.T, probes []supervisor.Probe) {
	t.Helper()
	names := documentedProbeNames()
	if len(probes) != len(names) {
		t.Fatalf("want %d probe rows, got %d: %+v", len(names), len(probes), probes)
	}
	for i, name := range names {
		if probes[i].Name != name {
			t.Fatalf("diagnostics row %d = %q, want %q (documented order is stable)", i, probes[i].Name, name)
		}
	}
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

// discoverUnpackedFixture seeds the unpacked-AppImage layout (matrix row 1:
// ~/Applications/zcode-*/squashfs-root) and asserts it resolves.
func discoverUnpackedFixture(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	root := filepath.Join(home, "Applications", "zcode-3.18.2", "squashfs-root")
	writeBundleTree(t, root)

	p := supervisor.Probers{
		Home:    home,
		AppDirs: func() []string { return []string{root} },
	}
	install, probes, err := p.Discover(context.Background())
	if err != nil {
		t.Fatalf("unpacked squashfs-root tree must resolve: %v", err)
	}
	want := supervisor.Install{
		Root:       root,
		Binary:     filepath.Join(root, "zcode"),
		Entrypoint: filepath.Join(root, "resources", "glm", "zcode.cjs"),
		Layout:     "unpacked-appimage",
	}
	if install != want {
		t.Fatalf("install mismatch:\n got %+v\nwant %+v", install, want)
	}
	row, ok := probeRow(probes, "unpacked-glob")
	if !ok || !strings.Contains(row.Result, root) {
		t.Errorf("unpacked-glob row must name the resolved root: %+v", row)
	}
}

// discoverAppImageFixture seeds the packed-AppImage layout (matrix row 2:
// versioned file + unversioned symlink, extraction stubbed) and asserts it
// resolves through EvalSymlinks.
func discoverAppImageFixture(t *testing.T) {
	t.Helper()
	apps := filepath.Join(t.TempDir(), "Applications")
	versioned := filepath.Join(apps, "ZCode-3.18.2-linux-x64.AppImage")
	if err := os.MkdirAll(apps, 0o750); err != nil {
		t.Fatalf("mkdir applications dir: %v", err)
	}
	if err := os.WriteFile(versioned, []byte("appimage-payload"), 0o600); err != nil {
		t.Fatalf("write AppImage file: %v", err)
	}
	// Relative target, per matrix row 2: the unversioned symlink is the
	// user-facing entry resolving to the versioned file.
	symlink := filepath.Join(apps, "ZCode.AppImage")
	if err := os.Symlink(filepath.Base(versioned), symlink); err != nil {
		t.Fatalf("symlink AppImage: %v", err)
	}
	extractRoot := filepath.Join(t.TempDir(), "squashfs-root")
	writeBundleTree(t, extractRoot)

	var gotAppImagePath string
	p := supervisor.Probers{
		Home:    filepath.Dir(apps),
		AppDirs: func() []string { return []string{versioned, symlink} },
		ExtractAppImage: func(_ context.Context, appImagePath, _ string) (string, error) {
			gotAppImagePath = appImagePath

			return extractRoot, nil
		},
	}
	install, _, err := p.Discover(context.Background())
	if err != nil {
		t.Fatalf("packed AppImage + extraction seam must resolve: %v", err)
	}
	if install.Root != extractRoot || install.Layout != "appimage" {
		t.Fatalf("extraction root must win with layout appimage: got %+v, want Root %q",
			install, extractRoot)
	}
	resolved, err := filepath.EvalSymlinks(symlink)
	if err != nil {
		t.Fatalf("eval symlink: %v", err)
	}
	if gotAppImagePath != resolved {
		t.Fatalf("extraction must receive the EvalSymlinks-resolved AppImage path: got %q, want %q",
			gotAppImagePath, resolved)
	}
	if install.Binary != filepath.Join(extractRoot, "zcode") {
		t.Errorf("extracted tree binary: got %q", install.Binary)
	}
	if install.Entrypoint != filepath.Join(extractRoot, "resources", "glm", "zcode.cjs") {
		t.Errorf("extracted tree entrypoint: got %q", install.Entrypoint)
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
		Layout:     "deb",
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
	t.Run("unpacked-appimage", discoverUnpackedFixture)
	t.Run("appimage", discoverAppImageFixture)
	t.Run("deb", discoverDebFixture)
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
	if install.Binary != wantBinary || install.Entrypoint != wantEntrypoint || install.Layout != "deb" {
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
	row, ok := probeRow(probes, "appimage-glob")
	if !ok || !strings.Contains(row.Result, mountDecoy) || !strings.Contains(row.Result, "reject") {
		t.Fatalf("appimage-glob row must record the mount as rejected: %+v", row)
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
	writeRuntimeTree(t, home, []string{"0.0.0-dev", "3.17.1", "3.18.2"}, []int{1, 43, 44})

	p := supervisor.Probers{
		Home:    home,
		AppDirs: func() []string { return []string{root} },
	}
	install, probes, err := p.Discover(context.Background())
	if err != nil {
		t.Fatalf("root install plus runtime tree must resolve: %v", err)
	}
	wantBuiltin := filepath.Join(home, ".zcode", "v2", "runtime", "provider",
		"linux-x86_64", "3.18.2", fixtureEndpointDir, "zcode-builtin.json")
	if install.BuiltinFile != wantBuiltin {
		t.Fatalf("builtin file under the highest version dir must win:\n got %q\nwant %q",
			install.BuiltinFile, wantBuiltin)
	}
	if install.Root != root || install.Layout != "unpacked-appimage" {
		t.Fatalf("root install must still resolve alongside the runtime tree: %+v", install)
	}
	row, ok := probeRow(probes, "builtin-runtime")
	if !ok || !strings.Contains(row.Result, "3.18.2") {
		t.Fatalf("builtin-runtime row must name the picked version: %+v", row)
	}
}
