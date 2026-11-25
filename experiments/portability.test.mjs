// Static portability contract for the detached-process fixtures. These checks
// are offline and run on every platform: the default Go suite must not depend
// on a host session-creation utility.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const source = (relativePath) => readFileSync(path.join(repoRoot, relativePath), "utf8");
const goSource = (relativePath) => source(relativePath);
const legacyDetachedCommand = /^[\t ]*setsid[\t ]+/m;
const detachedSession = /&syscall\.SysProcAttr\s*\{\s*Setsid:\s*true,?\s*\}/;

function tomlSection(document, name) {
  const lines = document.split(/\r?\n/);
  const start = lines.indexOf(`[${name}]`);
  assert.notEqual(start, -1, `mise.toml must define [${name}]`);
  const body = [];
  for (const line of lines.slice(start + 1)) {
    if (/^\[[^\]]+\]\s*$/.test(line)) break;
    body.push(line);
  }
  return body.join("\n");
}

function assertPortableDetachedFixture(source, fixture, selectors) {
  assert.equal(
    legacyDetachedCommand.test(source),
    false,
    `${fixture} must not execute the host setsid utility`,
  );

  for (const selector of selectors) {
    assert.ok(
      source.includes(`-test.run=^${selector}$`),
      `${fixture} must select ${selector} with an anchored test-binary re-exec`,
    );
  }

  assert.match(
    source,
    detachedSession,
    `${fixture} must create its detached session with SysProcAttr{Setsid: true}`,
  );
}

test("mount fixture creates its detached cleaner through Go", () => {
  const source = goSource("internal/supervisor/mount_test.go");

  assertPortableDetachedFixture(source, "the mount fixture", [
    "TestDetachedMountCleanerHelper",
  ]);
});

test("readiness fixture creates its detached leak through Go", () => {
  const source = goSource("cmd/zcode-acp/main_test.go");

  assertPortableDetachedFixture(source, "the readiness fixture", [
    "TestE2EDetachedLeakLauncherHelper",
    "TestE2EDetachedLeakHelper",
  ]);
});

test("docs gate uses four portable find-plus-grep scans", () => {
  const docsSource = source("experiments/check-docs.sh");
  const mintPattern = ["api", "keys"].join("_");
  const portableScans = [
    String.raw`find docs -type f -name '*.md' -exec grep -cE '^## Provenance legend' {} +`,
    String.raw`find docs -type f -name '*.md' -exec grep -hoE '\[V-EXP [^]]+\]' {} +`,
    String.raw`find docs -type f -name '*.md' -exec grep -hoE '\[V-[A-Za-z]+[^]]*\]' {} +`,
    String.raw`find experiments -type f \( -name '*.mjs' -o -name '*.sh' \) ! -name 'check-docs.sh' -exec grep -nHiE '${mintPattern}' {} +`,
  ];

  for (const scan of portableScans) {
    assert.ok(docsSource.includes(scan), `portable docs scan is missing: ${scan}`);
  }
  assert.doesNotMatch(
    docsSource,
    /--(?:include|exclude)(?:=|\b)/,
    "the docs gate must not use GNU-only grep include/exclude filters",
  );
  assert.ok(
    docsSource.includes("path_hits=$(grep -rnE \"$path_rx\" docs/ experiments/ 2>/dev/null || true)"),
    "the existing portable recursive path scan must remain unchanged",
  );
});

test("mise pins managed Bash and exposes both shell gates", () => {
  const miseSource = source("mise.toml");
  const tools = tomlSection(miseSource, "tools");

  assert.match(tools, /^bash\s*=\s*"5\.3"\s*$/m, "managed Bash must be pinned to 5.3");
  assert.match(
    tomlSection(miseSource, "tasks.secret-scan"),
    /^run\s*=\s*"bash experiments\/scan-secrets\.sh"\s*$/m,
    "secret-scan must invoke the script through managed bash",
  );
  assert.match(
    tomlSection(miseSource, "tasks.check-docs"),
    /^run\s*=\s*"bash experiments\/check-docs\.sh"\s*$/m,
    "check-docs must invoke the script through managed bash",
  );
});

test("CI appends shell gates while preserving native race and Linux VM separation", () => {
  const miseSource = source("mise.toml");

  assert.match(
    tomlSection(miseSource, "tasks.ci"),
    /^depends\s*=\s*\["build", "vet", "lint", "test", "test-experiments", "secret-scan", "check-docs"\]\s*$/m,
    "shell gates must follow test-experiments in the canonical CI graph",
  );
  assert.match(
    tomlSection(miseSource, "tasks.test"),
    /^run\s*=\s*"go test -race -count=1 \.\/\.\.\."\s*$/m,
    "native race testing must remain CGO-enabled and unchanged",
  );
  assert.doesNotMatch(miseSource, /CGO_ENABLED\s*=/m, "CGO settings must not be globalized");
  assert.doesNotMatch(miseSource, /vm-e2e\.sh/, "the Linux-only VM script must stay outside CI");
});
