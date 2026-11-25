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
