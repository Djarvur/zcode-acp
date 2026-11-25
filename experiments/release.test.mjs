// Release contracts stay static and offline by default. The adversarial
// archive arm runs only with ZCODE_ACP_RELEASE_FIXTURES=1 and never executes
// an archive member.

import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const configPath = path.join(repoRoot, ".goreleaser.yaml");
const verifierPath = path.join(repoRoot, "scripts", "verify-darwin-release.sh");
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

test("GoReleaser config declares binary-only thin and universal Darwin artifacts", () => {
  const source = readSource(configPath);
  assert.match(source, /^version: 2$/m);
  assert.match(source, /^project_name: zcode-acp$/m);

  const builds = listEntries(topLevelSection(source, "builds"));
  assert.equal(builds.length, 2, "exactly two thin Darwin builds are required");
  for (const [id, arch] of [
    ["darwin-arm64", "arm64"],
    ["darwin-amd64", "amd64"],
  ]) {
    const build = builds.find((entry) => entry.includes(`id: ${id}`));
    assert.ok(build, `missing build ${id}`);
    assert.match(build, /main: \.\/cmd\/zcode-acp/);
    assert.match(build, /binary: zcode-acp/);
    assert.match(build, /env: \[CGO_ENABLED=0\]/);
    assert.match(build, /goos: \[darwin\]/);
    assert.match(build, new RegExp(`goarch: \\[${arch}\\]`));
    assert.match(build, /flags: \[-trimpath\]/);
    assert.match(build, /ldflags: \[-s, -w, -X, "main\.version=\{\{ \.Version \}\}"\]/);
  }

  const universal = listEntries(topLevelSection(source, "universal_binaries"));
  assert.equal(universal.length, 1);
  assert.match(universal[0], /id: darwin-universal/);
  assert.match(universal[0], /ids: \[darwin-arm64, darwin-amd64\]/);
  assert.match(universal[0], /name_template: zcode-acp/);
  assert.match(universal[0], /replace: false/);

  const archives = listEntries(topLevelSection(source, "archives"));
  assert.equal(archives.length, 2, "thin and universal archive definitions are required");
  const thin = archives.find((entry) => entry.includes("id: darwin-thin"));
  const fat = archives.find((entry) => entry.includes("id: darwin-universal"));
  assert.ok(thin, "missing thin archive");
  assert.ok(fat, "missing universal archive");
  assert.match(thin, /ids: \[darwin-arm64, darwin-amd64\]/);
  assert.match(thin, /name_template: "zcode-acp_\{\{ \.Version \}\}_darwin_\{\{ \.Arch \}\}"/);
  assert.match(fat, /ids: \[darwin-universal\]/);
  assert.match(fat, /name_template: "zcode-acp_\{\{ \.Version \}\}_darwin_universal"/);
  for (const archive of archives) {
    assert.match(archive, /formats: \[tar\.gz\]/);
    assert.match(archive, /files: \["\.zcode-acp-no-extra-files-\*"\]/);
  }

  const checksum = topLevelSection(source, "checksum");
  assert.match(checksum, /name_template: checksums\.txt/);
  assert.match(checksum, /algorithm: sha256/);
  assert.match(readSource(path.join(repoRoot, ".gitignore")), /^\/dist\/$/m);
});

test("artifact verifier enforces safe checksummed architecture checks", () => {
  const source = readSource(verifierPath);
  assert.match(source, /^#!\/usr\/bin\/env bash$/m);
  assert.match(source, /set -euo pipefail/);
  assert.match(source, /\[\[ "\$#" -ne 1 \]\]/);
  assert.match(source, /shasum -a 256 -c checksums\.txt/);
  assert.match(source, /tar -tzf/);
  assert.match(source, /tar -tvzf/);
  assert.match(source, /\[\[ "\$members" == "\$EXPECTED_BINARY" \]\]/);
  assert.match(source, /\[\[ "\$\{#listing_lines\[@\]\}" -eq 1 \]\]/);
  assert.match(source, /\[\[ "\$\{listing_lines\[0\]\}" == -\* \]\]/);
  assert.match(source, /mktemp -d/);
  assert.match(source, /trap cleanup EXIT/);
  assert.match(source, /tar -xzf/);
  assert.match(source, /\[\[ -f "\$binary" && ! -L "\$binary" \]\]/);
  assert.match(source, /lipo -archs "\$binary"/);
  assert.match(source, /"\$\{actual_archs\[0\]\}" == arm64/);
  assert.match(source, /"\$\{actual_archs\[0\]\}" == x86_64/);
  assert.match(source, /"\$\{#actual_archs\[@\]\}" -eq 2/);
  assert.match(source, /" \$\{actual_archs\[\*\]\} " == \*" arm64 "\*/);
  assert.match(source, /" \$\{actual_archs\[\*\]\} " == \*" x86_64 "\*/);
  assert.doesNotMatch(source, /^\s*["']?\$(?:binary|extracted_binary)["']?(?:\s|$)/m);
  assertOrdered(source, [
    "shasum -a 256 -c checksums.txt",
    '[[ "$members" == "$EXPECTED_BINARY" ]]',
    '[[ "${listing_lines[0]}" == -* ]]',
    "tar -xzf",
    "lipo -archs",
  ]);
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

function assertHomebrewBashBootstrap(job) {
  const install = namedStep(job, "Install Bash 5 via Homebrew");
  const prepend = namedStep(job, "Prepend Homebrew Bash to PATH");

  assert.match(install, /^        run: brew install bash$/m);
  assert.match(prepend, /^        run: echo "\$\(brew --prefix\)\/bin" >> "\$GITHUB_PATH"$/m);
  assertOrdered(job, [
    "name: Install Bash 5 via Homebrew",
    "name: Prepend Homebrew Bash to PATH",
    "name: Install tools from mise.toml",
  ]);
}

function assertDependencyGate(job) {
  const version = namedStep(job, "Validate tool versions");
  const download = namedStep(job, "Download frozen Go modules");
  const offline = namedStep(job, "Verify frozen modules offline");
  const apple = namedStep(job, "Require Apple release tools");
  const config = namedStep(job, "Validate GoReleaser config");

  assert.match(version, /GOPROXY=off mise exec -- node scripts\/check-mise-tool-versions\.mjs go goreleaser bash/);
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
  assert.match(apple, /command -v shasum/);
  assert.match(config, /GOPROXY=off mise exec -- goreleaser check/);

  assertOrdered(job, [
    "name: Validate tool versions",
    "name: Download frozen Go modules",
    "mise exec -- go mod download",
    "git diff --exit-code HEAD -- go.mod go.sum",
    "name: Verify frozen modules offline",
    "GOPROXY=off mise exec -- go mod verify",
    "GOPROXY=off mise run tidy-diff",
    "name: Require Apple release tools",
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
  assert.match(snapshot, /name: goreleaser snapshot \(macOS universal\)/);
  assert.match(snapshot, /runs-on: macos-latest/);
  assert.match(snapshot, /uses: actions\/checkout@v4/);
  assert.match(snapshot, /fetch-depth: 0/);
  assert.match(snapshot, /run: curl https:\/\/mise\.run \| sh/);
  assert.match(snapshot, /run: mise install/);
  assertHomebrewBashBootstrap(snapshot);
  assert.doesNotMatch(snapshot, /GITHUB_TOKEN|secrets\.|contents:\s*write/);
  assert.doesNotMatch(snapshot, /^\s+if:/m);
  assertDependencyGate(snapshot);

  const build = namedStep(snapshot, "Build release snapshot");
  const verify = namedStep(snapshot, "Verify Darwin snapshot artifacts");
  const upload = namedStep(snapshot, "Upload Darwin snapshot artifacts");
  assert.match(build, /GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish/);
  assert.match(verify, /mise exec -- bash scripts\/verify-darwin-release\.sh dist/);
  assert.match(upload, /uses: actions\/upload-artifact@v4/);
  assert.match(upload, /if-no-files-found: error/);
  assert.match(upload, /dist\/zcode-acp_\*_darwin_arm64\.tar\.gz/);
  assert.match(upload, /dist\/zcode-acp_\*_darwin_amd64\.tar\.gz/);
  assert.match(upload, /dist\/zcode-acp_\*_darwin_universal\.tar\.gz/);
  assert.match(upload, /dist\/checksums\.txt/);
  assertOrdered(snapshot, [
    "GOPROXY=off mise exec -- goreleaser check",
    "name: Build release snapshot",
    "GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish",
    "name: Verify Darwin snapshot artifacts",
    "name: Upload Darwin snapshot artifacts",
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
  assertHomebrewBashBootstrap(release);
  assertDependencyGate(release);

  const publish = namedStep(release, "goreleaser release");
  assert.match(publish, /GOPROXY=off mise exec -- goreleaser release --clean/);
  assert.match(publish, /GITHUB_TOKEN: \$\{\{ secrets\.GITHUB_TOKEN \}\}/);
  assert.doesNotMatch(publish, /--snapshot|--skip=publish/);
  assertOrdered(release, [
    "name: Validate tool versions",
    "name: Download frozen Go modules",
    "name: Verify frozen modules offline",
    "name: Require Apple release tools",
    "name: Validate GoReleaser config",
    "GOPROXY=off mise exec -- goreleaser check",
    "      - name: goreleaser release",
    "GOPROXY=off mise exec -- goreleaser release --clean",
  ]);

  // Task 2's verified read-only snapshot remains present and unchanged in contract.
  assert.match(snapshot, /name: goreleaser snapshot \(macOS universal\)/);
  assert.match(snapshot, /GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish/);
  assert.match(snapshot, /mise exec -- bash scripts\/verify-darwin-release\.sh dist/);
  assert.match(snapshot, /uses: actions\/upload-artifact@v4/);
});

const archiveNames = {
  arm64: "zcode-acp_0.0.0-SNAPSHOT-test_darwin_arm64.tar.gz",
  amd64: "zcode-acp_0.0.0-SNAPSHOT-test_darwin_amd64.tar.gz",
  universal: "zcode-acp_0.0.0-SNAPSHOT-test_darwin_universal.tar.gz",
};

function runTool(command, args, options = {}) {
  const result = spawnSync(command, args, { encoding: "utf8", ...options });
  assert.ifError(result.error);
  assert.equal(
    result.status,
    0,
    `${command} failed: ${result.stdout ?? ""}${result.stderr ?? ""}`,
  );
  return result;
}

function createArchive(stage, archive, entries = ["zcode-acp"], preservePaths = false) {
  const args = ["-czf", archive];
  if (preservePaths) args.push("-P");
  args.push(...entries);
  runTool("/usr/bin/tar", args, { cwd: stage });
}

function createFixture(scenario) {
  const root = mkdtempSync(path.join(tmpdir(), "zcode-acp-release-fixture-"));
  const dist = path.join(root, "dist");
  const bin = path.join(root, "bin");
  const tarLog = path.join(root, "tar.log");
  const lipoLog = path.join(root, "lipo.log");
  const binaryMarker = path.join(root, "binary-executed");
  mkdirSync(dist);
  mkdirSync(bin);
  writeFileSync(tarLog, "");
  writeFileSync(lipoLog, "");

  writeFileSync(
    path.join(bin, "tar"),
    '#!/bin/sh\nprintf "%s\\n" "$*" >> "$FIXTURE_TAR_LOG"\nexec /usr/bin/tar "$@"\n',
    { mode: 0o755 },
  );
  writeFileSync(
    path.join(bin, "lipo"),
    '#!/bin/sh\nprintf "%s\\n" "$*" >> "$FIXTURE_LIPO_LOG"\nexit 99\n',
    { mode: 0o755 },
  );

  for (const kind of ["arm64", "amd64", "universal"]) {
    const stage = path.join(root, `stage-${kind}`);
    mkdirSync(stage);
    const archive = path.join(dist, archiveNames[kind]);
    if (kind === "arm64" && scenario === "symlink") {
      symlinkSync("/etc/passwd", path.join(stage, "zcode-acp"));
      createArchive(stage, archive);
      continue;
    }
    if (kind === "arm64" && scenario === "traversal") {
      writeFileSync(path.join(root, "outside"), "unsafe");
      createArchive(stage, archive, ["../outside"], true);
      continue;
    }
    if (kind === "arm64" && scenario === "absolute") {
      const outside = path.join(root, "absolute-outside");
      writeFileSync(outside, "unsafe");
      createArchive(stage, archive, [outside], true);
      continue;
    }

    writeFileSync(
      path.join(stage, "zcode-acp"),
      '#!/bin/sh\nprintf executed > "$FIXTURE_BINARY_MARKER"\n',
      { mode: 0o755 },
    );
    if (kind === "arm64" && scenario === "extra") {
      writeFileSync(path.join(stage, "README.md"), "unexpected");
      createArchive(stage, archive, ["zcode-acp", "README.md"]);
      continue;
    }
    if (kind === "arm64" && scenario === "missing") {
      rmSync(path.join(stage, "zcode-acp"));
      writeFileSync(path.join(stage, "README.md"), "no binary");
      createArchive(stage, archive, ["README.md"]);
      continue;
    }
    createArchive(stage, archive);
  }

  const checksums = Object.values(archiveNames)
    .map((name) => {
      const digest = createHash("sha256").update(readFileSync(path.join(dist, name))).digest("hex");
      return `${digest}  ${name}\n`;
    })
    .sort()
    .join("");
  writeFileSync(path.join(dist, "checksums.txt"), checksums);

  return { root, dist, tarLog, lipoLog, binaryMarker };
}

test("artifact verifier rejects adversarial archives before extraction", {
  skip: process.env.ZCODE_ACP_RELEASE_FIXTURES !== "1",
}, async (t) => {
  const bashOverride = process.env.ZCODE_ACP_RELEASE_BASH?.trim();
  if (bashOverride) assert.ok(path.isAbsolute(bashOverride), "release Bash override must be absolute");
  const command = bashOverride || "mise";
  const args = bashOverride
    ? [verifierPath]
    : ["exec", "--", "bash", verifierPath];

  for (const scenario of ["traversal", "absolute", "symlink", "extra", "missing"]) {
    await t.test(scenario, () => {
      const fixture = createFixture(scenario);
      try {
        const result = spawnSync(command, [...args, fixture.dist], {
          encoding: "utf8",
          timeout: 30_000,
          env: {
            ...process.env,
            PATH: `${path.join(fixture.root, "bin")}:${process.env.PATH ?? ""}`,
            FIXTURE_TAR_LOG: fixture.tarLog,
            FIXTURE_LIPO_LOG: fixture.lipoLog,
            FIXTURE_BINARY_MARKER: fixture.binaryMarker,
          },
        });
        const output = `${result.stdout ?? ""}${result.stderr ?? ""}`;
        assert.ifError(result.error);
        assert.notEqual(result.status, 0, `${scenario} fixture was accepted:\n${output}`);
        assert.match(
          output,
          scenario === "symlink" ? /not a regular file/ : /archive member list is not exactly zcode-acp/,
          `${scenario} fixture failed at the wrong validation stage:\n${output}`,
        );
        assert.equal(readFileSync(fixture.lipoLog, "utf8"), "", "lipo ran before rejection");
        assert.doesNotMatch(readFileSync(fixture.tarLog, "utf8"), /(^|\s)-x/, "tar extracted before rejection");
        assert.equal(
          requireFileExists(fixture.binaryMarker),
          false,
          "an archive member was executed",
        );
      } finally {
        rmSync(fixture.root, { recursive: true, force: true });
      }
    });
  }
});

function requireFileExists(filePath) {
  try {
    readFileSync(filePath, "utf8");
    return true;
  } catch (error) {
    if (error.code === "ENOENT") return false;
    throw error;
  }
}
