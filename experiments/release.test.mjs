// Static release contracts stay offline by default. Behavior of the Go
// verifier, including adversarial archives, is covered by its Go unit tests.

import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const configPath = path.join(repoRoot, ".goreleaser.yaml");
const verifierPath = path.join(repoRoot, "cmd", "verify-release", "main.go");
const readSource = (filePath) => readFileSync(filePath, "utf8");

function topLevelSection(source, name) {
  const lines = source.split("\n");
  const start = lines.indexOf(`${name}:`);
  assert.notEqual(start, -1, `missing top-level ${name}: section`);
  let end = lines.length;
  for (let index = start + 1; index < lines.length; index += 1) {
    if (/^[A-Za-z][A-Za-z0-9_]*:/.test(lines[index])) {
      end = index;
      break;
    }
  }
  return lines.slice(start + 1, end).join("\n");
}

function listEntries(section) {
  return section
    .split(/^  - /m)
    .slice(1)
    .map((entry) => entry.replace(/\n$/, ""));
}

function assertOrdered(source, ordered) {
  let previous = -1;
  for (const token of ordered) {
    const index = source.indexOf(token);
    assert.ok(index > previous, `${token} must appear after the preceding contract item`);
    previous = index;
  }
}

function assertNoBashDependency(document, label) {
  assert.doesNotMatch(
    document,
    /brew install bash|brew --prefix[^\n]*GITHUB_PATH|ZCODE_ACP_REQUIRED_BASH|mise exec -- bash|run: bash\b|verify-(?:darwin-)?release\.sh/,
    `${label} must not install or invoke Bash for release gates`,
  );
}

test("GoReleaser config declares five thin builds plus the Darwin universal binary", () => {
  const source = readSource(configPath);
  assert.match(source, /^version: 2$/m);
  assert.match(source, /^project_name: zcode-acp$/m);

  const builds = listEntries(topLevelSection(source, "builds"));
  assert.equal(builds.length, 5, "five thin platform builds are required");
  const targets = [
    ["linux-amd64", "linux", "amd64"],
    ["linux-arm64", "linux", "arm64"],
    ["darwin-arm64", "darwin", "arm64"],
    ["darwin-amd64", "darwin", "amd64"],
    ["windows-amd64", "windows", "amd64"],
  ];
  for (const [id, goos, goarch] of targets) {
    const build = builds.find((entry) => entry.includes(`id: ${id}`));
    assert.ok(build, `missing build ${id}`);
    assert.match(build, /main: \.\/cmd\/zcode-acp/);
    assert.match(build, /binary: zcode-acp/);
    assert.match(build, /env: \[CGO_ENABLED=0\]/);
    assert.match(build, new RegExp(`goos: \\[${goos}\\]`));
    assert.match(build, new RegExp(`goarch: \\[${goarch}\\]`));
    assert.match(build, /flags: \[-trimpath\]/);
    assert.match(
      build,
      /ldflags: \[-s, -w, -X, "main\.version=18\.1\.9\+zacp\.\{\{ \.Version \}\}"\]/,
    );
  }

  const universal = listEntries(topLevelSection(source, "universal_binaries"));
  assert.equal(universal.length, 1);
  assert.match(universal[0], /id: darwin-universal/);
  assert.match(universal[0], /ids: \[darwin-arm64, darwin-amd64\]/);
  assert.match(universal[0], /name_template: zcode-acp/);
  assert.match(universal[0], /replace: false/);

  const archives = listEntries(topLevelSection(source, "archives"));
  assert.equal(archives.length, 2, "thin and universal archive definitions are required");
  const thin = archives.find((entry) => entry.includes("id: thin"));
  const fat = archives.find((entry) => entry.includes("id: darwin-universal"));
  assert.ok(thin, "missing thin archive");
  assert.ok(fat, "missing universal archive");
  assert.match(thin, /ids: \[linux-amd64, linux-arm64, darwin-arm64, darwin-amd64, windows-amd64\]/);
  assert.match(thin, /formats: \[tar\.gz\]/);
  assert.match(thin, /format_overrides:\n      - goos: windows\n        formats: \[zip\]/);
  assert.match(thin, /name_template: "zcode-acp_\{\{ \.Version \}\}_\{\{ \.Os \}\}_\{\{ \.Arch \}\}"/);
  assert.match(fat, /ids: \[darwin-universal\]/);
  assert.match(fat, /formats: \[tar\.gz\]/);
  assert.match(fat, /name_template: "zcode-acp_\{\{ \.Version \}\}_darwin_universal"/);
  for (const archive of archives) {
    assert.match(archive, /files: \["\.zcode-acp-no-extra-files-\*"\]/);
  }

  const checksum = topLevelSection(source, "checksum");
  assert.match(checksum, /name_template: checksums\.txt/);
  assert.match(checksum, /algorithm: sha256/);
  assert.match(readSource(path.join(repoRoot, ".gitignore")), /^\/dist\/$/m);
});

test("Go verifier covers all six platform archives without executing them", () => {
  assert.equal(
    existsSync(path.join(repoRoot, "scripts", "verify-darwin-release.sh")),
    false,
    "the shell verifier must be deleted",
  );
  assert.equal(
    existsSync(path.join(repoRoot, "scripts", "verify-release.sh")),
    false,
    "the shell verifier must stay deleted",
  );

  const source = readSource(verifierPath);
  for (const required of [
    '"archive/tar"',
    '"archive/zip"',
    '"compress/gzip"',
    '"crypto/sha256"',
    '"debug/elf"',
    '"debug/pe"',
    '"os/exec"',
    'exec.CommandContext(context.Background(), "lipo", "-archs", binary)',
    "os.MkdirTemp",
    "extractTarGz",
    "extractZip",
    "elfArch",
    "peArch",
    "architectureMatches",
    "verifyVersionStamp",
  ]) {
    assert.ok(source.includes(required), `Go verifier is missing ${required}`);
  }
  assert.match(source, /expectedBinary\s+= "zcode-acp"/, "Go verifier binary name drifted");
  assert.match(source, /windowsBinary\s+= "zcode-acp\.exe"/, "Go verifier Windows binary name drifted");
  assert.match(source, /versionStampPrefix\s+= "18\.1\.9\+zacp\."/, "Go verifier stamp drifted");
  for (const label of [
    "linux_amd64",
    "linux_arm64",
    "darwin_amd64",
    "darwin_arm64",
    "darwin_universal",
    "windows_amd64",
  ]) {
    assert.match(source, new RegExp(`label:\\s+"${label}"`), `Go verifier is missing ${label}`);
  }
  assert.doesNotMatch(
    source,
    /exec\.Command\("(?:tar|gzip|shasum)"/,
    "the verifier must not shell out to tar, gzip, or shasum",
  );
  assert.doesNotMatch(
    source,
    /exec\.Command\(expectedBinary/,
    "the verifier must never execute a release binary",
  );

  const mise = readSource(path.join(repoRoot, "mise.toml"));
  assert.match(mise, /^run = "go run \.\/cmd\/verify-release dist"$/m);
});

function jobSection(source, name) {
  const lines = source.split("\n");
  const start = lines.indexOf(`  ${name}:`);
  assert.notEqual(start, -1, `missing job ${name}`);
  let end = lines.length;
  for (let index = start + 1; index < lines.length; index += 1) {
    if (/^  [A-Za-z][A-Za-z0-9_-]*:/.test(lines[index])) {
      end = index;
      break;
    }
  }
  return lines.slice(start + 1, end).join("\n");
}

function namedStep(job, name) {
  const marker = `      - name: ${name}\n`;
  const start = job.indexOf(marker);
  assert.notEqual(start, -1, `missing step ${name}`);
  const rest = job.slice(start + marker.length);
  const next = rest.search(/^      - (?:name|uses):/m);
  return next === -1 ? rest : rest.slice(0, next);
}

function assertDependencyGate(job) {
  const version = namedStep(job, "Validate tool versions");
  const download = namedStep(job, "Download frozen Go modules");
  const offline = namedStep(job, "Verify frozen modules offline");
  const apple = namedStep(job, "Require Apple lipo");
  const config = namedStep(job, "Validate GoReleaser config");

  assert.match(version, /GOPROXY=off mise exec -- node scripts\/check-mise-tool-versions\.mjs go goreleaser\n/);
  assert.doesNotMatch(version, /bash/, "release version validation must not probe Bash");
  assert.match(download, /timeout-minutes: 5/);
  assert.match(download, /GOPROXY: https:\/\/proxy\.golang\.org/);
  assert.match(download, /mise exec -- go mod download/);
  assert.match(download, /git diff --exit-code HEAD -- go\.mod go\.sum/);
  assert.doesNotMatch(download, /GOPROXY:\s*off/);
  assert.doesNotMatch(download, /^\s+if:/m);
  assert.match(offline, /GOPROXY=off mise exec -- go mod verify/);
  assert.match(offline, /GOPROXY=off mise run tidy-diff/);
  assert.doesNotMatch(offline, /^\s+if:/m);
  assert.match(apple, /command -v lipo/);
  assert.doesNotMatch(apple, /shasum/);
  assert.match(config, /GOPROXY=off mise exec -- goreleaser check/);

  assertOrdered(job, [
    "name: Validate tool versions",
    "name: Download frozen Go modules",
    "mise exec -- go mod download",
    "git diff --exit-code HEAD -- go.mod go.sum",
    "name: Verify frozen modules offline",
    "GOPROXY=off mise exec -- go mod verify",
    "GOPROXY=off mise run tidy-diff",
    "name: Require Apple lipo",
    "name: Validate GoReleaser config",
    "GOPROXY=off mise exec -- goreleaser check",
  ]);

  const offlineIndex = job.indexOf("name: Verify frozen modules offline");
  for (const match of job.matchAll(/GOPROXY=off mise exec -- goreleaser [^\n]+/g)) {
    assert.ok(match.index > offlineIndex, "offline GoReleaser command precedes the dependency gate");
  }
}

test("release workflow runs the proven snapshot with read-only rights", () => {
  const source = readSource(path.join(repoRoot, ".github", "workflows", "release.yml"));
  const triggers = topLevelSection(source, "on");
  assert.match(triggers, /push:\s*\n\s+tags: \['v\*'\]/);
  assert.match(triggers, /^  workflow_dispatch:/m);
  assert.match(topLevelSection(source, "permissions"), /contents: read/);
  assert.doesNotMatch(source, /GOPROXY=off[^\n]*go mod download/);

  const snapshot = jobSection(source, "snapshot");
  assert.match(snapshot, /name: goreleaser snapshot \(multi-platform\)/);
  assert.match(snapshot, /runs-on: macos-latest/);
  assert.match(snapshot, /uses: actions\/checkout@v4/);
  assert.match(snapshot, /fetch-depth: 0/);
  assert.match(snapshot, /run: curl https:\/\/mise\.run \| sh/);
  assert.match(snapshot, /run: mise install/);
  assertNoBashDependency(snapshot, "snapshot job");
  assert.doesNotMatch(snapshot, /GITHUB_TOKEN|secrets\.|contents:\s*write/);
  assert.doesNotMatch(snapshot, /^\s+if:/m);
  assertDependencyGate(snapshot);

  const build = namedStep(snapshot, "Build release snapshot");
  const verify = namedStep(snapshot, "Verify release snapshot artifacts");
  const upload = namedStep(snapshot, "Upload release snapshot artifacts");
  assert.match(build, /GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish/);
  assert.match(verify, /mise run verify-release/);
  assert.match(upload, /uses: actions\/upload-artifact@v4/);
  assert.match(upload, /if-no-files-found: error/);
  assert.match(upload, /dist\/zcode-acp_\*_linux_amd64\.tar\.gz/);
  assert.match(upload, /dist\/zcode-acp_\*_linux_arm64\.tar\.gz/);
  assert.match(upload, /dist\/zcode-acp_\*_darwin_arm64\.tar\.gz/);
  assert.match(upload, /dist\/zcode-acp_\*_darwin_amd64\.tar\.gz/);
  assert.match(upload, /dist\/zcode-acp_\*_darwin_universal\.tar\.gz/);
  assert.match(upload, /dist\/zcode-acp_\*_windows_amd64\.zip/);
  assert.match(upload, /dist\/checksums\.txt/);
  assertOrdered(snapshot, [
    "GOPROXY=off mise exec -- goreleaser check",
    "name: Build release snapshot",
    "GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish",
    "name: Verify release snapshot artifacts",
    "name: Upload release snapshot artifacts",
  ]);
});

test("release workflow scopes snapshot and tag publication", () => {
  const source = readSource(path.join(repoRoot, ".github", "workflows", "release.yml"));
  const snapshot = jobSection(source, "snapshot");
  const release = jobSection(source, "release");
  assert.match(topLevelSection(source, "permissions"), /contents: read/);
  assert.equal((source.match(/contents:\s*write/g) ?? []).length, 1);
  assert.equal((source.match(/GITHUB_TOKEN:/g) ?? []).length, 1);
  assert.doesNotMatch(snapshot, /GITHUB_TOKEN|contents:\s*write/);
  assert.doesNotMatch(source, /GOPROXY=off[^\n]*go mod download/);

  assert.match(release, /^    needs: snapshot$/m);
  assert.match(release, /^    if: github\.event_name == 'push' && startsWith\(github\.ref, 'refs\/tags\/v'\)$/m);
  assert.match(release, /runs-on: macos-latest/);
  assert.match(release, /permissions:\s*\n\s+contents: write/);
  assert.match(release, /uses: actions\/checkout@v4/);
  assert.match(release, /fetch-depth: 0/);
  assert.match(release, /run: curl https:\/\/mise\.run \| sh/);
  assert.match(release, /run: mise install/);
  assertNoBashDependency(release, "publish job");
  assertDependencyGate(release);

  const publish = namedStep(release, "goreleaser release");
  assert.match(publish, /GOPROXY=off mise exec -- goreleaser release --clean/);
  assert.match(publish, /GITHUB_TOKEN: \$\{\{ secrets\.GITHUB_TOKEN \}\}/);
  assert.doesNotMatch(publish, /--snapshot|--skip=publish/);
  assertOrdered(release, [
    "name: Validate tool versions",
    "name: Download frozen Go modules",
    "name: Verify frozen modules offline",
    "name: Require Apple lipo",
    "name: Validate GoReleaser config",
    "GOPROXY=off mise exec -- goreleaser check",
    "      - name: goreleaser release",
    "GOPROXY=off mise exec -- goreleaser release --clean",
  ]);

  assert.match(snapshot, /name: goreleaser snapshot \(multi-platform\)/);
  assert.match(snapshot, /GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish/);
  assert.match(snapshot, /mise run verify-release/);
  assert.match(snapshot, /uses: actions\/upload-artifact@v4/);
});
