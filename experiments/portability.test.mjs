// Static portability contract for the detached-process fixtures. These checks
// are offline and run on every platform: the default Go suite must not depend
// on a host session-creation utility.

import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const goSource = (relativePath) => readFileSync(path.join(repoRoot, relativePath), "utf8");
const legacyDetachedCommand = /^[\t ]*setsid[\t ]+/m;
const detachedSession = /&syscall\.SysProcAttr\s*\{\s*Setsid:\s*true,?\s*\}/;

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
