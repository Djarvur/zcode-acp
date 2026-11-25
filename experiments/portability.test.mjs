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

function workflowJob(jobId) {
  const lines = source(".github/workflows/pr-sanity.yml").split(/\r?\n/);
  const start = lines.indexOf(`  ${jobId}:`);
  assert.notEqual(start, -1, `pr-sanity.yml must define the ${jobId} job`);
  const nextJob = lines.findIndex(
    (line, index) => index > start && /^  [a-z0-9-]+:\s*$/.test(line),
  );
  return lines.slice(start + 1, nextJob === -1 ? lines.length : nextJob).join("\n");
}

function workflowStep(job, stepName) {
  const lines = job.split(/\r?\n/);
  const start = lines.indexOf(`      - name: ${stepName}`);
  assert.notEqual(start, -1, `workflow job must define the ${stepName} step`);
  const nextStep = lines.findIndex(
    (line, index) => index > start && /^      - /.test(line),
  );
  return lines.slice(start, nextStep === -1 ? lines.length : nextStep).join("\n");
}

function assertWorkflowOrder(document, orderedNeedles) {
  let previousIndex = -1;
  for (const needle of orderedNeedles) {
    const index = document.indexOf(needle);
    assert.ok(index > previousIndex, `workflow step is missing or out of order: ${needle}`);
    previousIndex = index;
  }
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

test("tool-version gate reads all five mise pins and probes only through mise", () => {
  const gateSource = source("scripts/check-mise-tool-versions.mjs");
  const expectedTools = ["go", "golangci-lint", "goreleaser", "node", "bash"];
  const requiredBlock = /const REQUIRED_TOOLS = \[([\s\S]*?)\];/.exec(gateSource)?.[1];
  assert.ok(requiredBlock, "the gate must declare its five required tools");
  const gateTools = [...requiredBlock.matchAll(/"([^"]+)"/g)].map((match) => match[1]);
  assert.deepEqual(gateTools, expectedTools);

  const toolsTable = tomlSection(source("mise.toml"), "tools");
  const pinnedTools = [...toolsTable.matchAll(/^([a-z-]+)\s*=\s*"([^"]+)"\s*$/gm)].map(
    (match) => match[1],
  );
  assert.deepEqual(pinnedTools, expectedTools, "mise.toml must remain the five-pin source");

  assert.ok(
    gateSource.includes('readFileSync(path.join(REPO_ROOT, "mise.toml"), "utf8")'),
    "the gate must read requirements from mise.toml",
  );
  assert.ok(
    gateSource.includes('spawnSync("mise", ["exec", "--", tool, ...VERSION_ARGS[tool]]'),
    "every version probe must run through mise exec",
  );
  assert.doesNotMatch(
    gateSource,
    /(?:spawnSync|execFileSync|execSync)\(\s*tool\b/,
    "the gate must never fall back to a bare PATH command",
  );
  assert.match(gateSource, /timeout:\s*PROBE_TIMEOUT_MS/, "each probe must be bounded");
});

test("tool-version gate exports comparators and fails non-zero on drift", () => {
  const gateSource = source("scripts/check-mise-tool-versions.mjs");

  assert.match(gateSource, /export function resolvedVersionFromOutput\(/);
  assert.match(gateSource, /export function requiredVersionMatches\(/);
  assert.ok(
    gateSource.includes("if (!passed) process.exitCode = 1;"),
    "probe failures or version mismatches must set a non-zero exit",
  );
  assert.ok(
    gateSource.includes("if (process.argv[1] && path.resolve(process.argv[1]) === SCRIPT_PATH) main();"),
    "importing the comparators must not execute mise",
  );
});

test("Linux sanity preserves its protected required-check name and runner", () => {
  const sanity = workflowJob("sanity");

  assert.match(sanity, /^    name: build \/ vet \/ lint \/ test \/ tidy$/m);
  assert.match(sanity, /^    runs-on: ubuntu-latest$/m);
});

test("macOS arm64 runs architecture and version preflights before canonical gates", () => {
  const arm64 = workflowJob("macos-arm64");

  assert.match(arm64, /^    name: build \/ vet \/ lint \/ test \/ tidy \(macOS arm64\)$/m);
  assert.match(arm64, /^    runs-on: macos-latest$/m);
  assert.match(arm64, /^    timeout-minutes: 30$/m);
  assert.ok(arm64.includes("      - uses: actions/checkout@v4"));
  assert.ok(arm64.includes("        run: curl https://mise.run | sh"));
  assert.ok(arm64.includes('        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"'));
  assert.ok(arm64.includes("        run: mise install"));

  const architecture = workflowStep(arm64, "Verify native architecture");
  assert.match(architecture, /^          EXPECTED_GOARCH: arm64$/m);
  assert.match(architecture, /^          EXPECTED_UNAME_M: arm64$/m);
  assert.ok(architecture.includes("mise exec -- go env GOARCH"));
  assert.ok(architecture.includes("uname -m"));
  assert.ok(architecture.includes('test "$(mise exec -- go env GOARCH)" = "$EXPECTED_GOARCH"'));
  assert.ok(architecture.includes('test "$(uname -m)" = "$EXPECTED_UNAME_M"'));

  const versions = workflowStep(arm64, "Verify mise tool versions");
  assert.match(
    versions,
    /^        run: mise exec -- node scripts\/check-mise-tool-versions\.mjs$/m,
  );
  assert.ok(arm64.includes("        run: mise run ci"));
  assert.ok(arm64.includes("        run: mise run tidy-diff"));
  assertWorkflowOrder(arm64, [
    "      - name: Install tools from mise.toml",
    "      - name: Verify native architecture",
    "      - name: Verify mise tool versions",
    "        run: mise run ci",
    "        run: mise run tidy-diff",
  ]);
  assert.doesNotMatch(arm64, /CGO_ENABLED\s*=\s*0/, "native race tests must keep CGO enabled");
});

test("macOS amd64 mirrors the native preflights and canonical gates", () => {
  const amd64 = workflowJob("macos-amd64");

  assert.match(amd64, /^    name: build \/ vet \/ lint \/ test \/ tidy \(macOS amd64\)$/m);
  assert.match(amd64, /^    runs-on: macos-15-intel$/m);
  assert.match(amd64, /^    timeout-minutes: 30$/m);
  assert.ok(amd64.includes("      - uses: actions/checkout@v4"));
  assert.ok(amd64.includes("        run: curl https://mise.run | sh"));
  assert.ok(amd64.includes('        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"'));
  assert.ok(amd64.includes("        run: mise install"));

  const architecture = workflowStep(amd64, "Verify native architecture");
  assert.match(architecture, /^          EXPECTED_GOARCH: amd64$/m);
  assert.match(architecture, /^          EXPECTED_UNAME_M: x86_64$/m);
  assert.ok(architecture.includes("mise exec -- go env GOARCH"));
  assert.ok(architecture.includes("uname -m"));
  assert.ok(architecture.includes('test "$(mise exec -- go env GOARCH)" = "$EXPECTED_GOARCH"'));
  assert.ok(architecture.includes('test "$(uname -m)" = "$EXPECTED_UNAME_M"'));

  const versions = workflowStep(amd64, "Verify mise tool versions");
  assert.match(
    versions,
    /^        run: mise exec -- node scripts\/check-mise-tool-versions\.mjs$/m,
  );
  assert.ok(amd64.includes("        run: mise run ci"));
  assert.ok(amd64.includes("        run: mise run tidy-diff"));
  assertWorkflowOrder(amd64, [
    "      - name: Install tools from mise.toml",
    "      - name: Verify native architecture",
    "      - name: Verify mise tool versions",
    "        run: mise run ci",
    "        run: mise run tidy-diff",
  ]);
  assert.doesNotMatch(amd64, /CGO_ENABLED\s*=\s*0/, "native race tests must keep CGO enabled");
});

test("Darwin cross-build checks tool pins before both literal CGO-free targets", () => {
  const crossBuild = workflowJob("darwin-cross-build");
  const arm64Command =
    "mise exec -- env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...";
  const amd64Command =
    "mise exec -- env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...";

  assert.match(crossBuild, /^    runs-on: ubuntu-latest$/m);
  assert.ok(crossBuild.includes("      - uses: actions/checkout@v4"));
  assert.ok(crossBuild.includes("        run: curl https://mise.run | sh"));
  assert.ok(crossBuild.includes('        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"'));
  assert.ok(crossBuild.includes("        run: mise install"));
  assert.ok(
    workflowStep(crossBuild, "Verify mise tool versions").includes(
      "        run: mise exec -- node scripts/check-mise-tool-versions.mjs",
    ),
  );
  assert.ok(workflowStep(crossBuild, "Cross-build darwin/arm64").includes(arm64Command));
  assert.ok(workflowStep(crossBuild, "Cross-build darwin/amd64").includes(amd64Command));
  assertWorkflowOrder(crossBuild, [
    "      - name: Install tools from mise.toml",
    "      - name: Verify mise tool versions",
    "      - name: Cross-build darwin/arm64",
    "      - name: Cross-build darwin/amd64",
  ]);
});

test("secret-scan runs both shell gates through mise-managed Bash tasks", () => {
  const secretScan = workflowJob("secret-scan");

  assert.match(secretScan, /^    name: secret-scan$/m);
  assert.match(secretScan, /^    runs-on: ubuntu-latest$/m);
  assert.ok(secretScan.includes("      - uses: actions/checkout@v4"));
  assert.ok(secretScan.includes("        run: curl https://mise.run | sh"));
  assert.ok(secretScan.includes('        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"'));
  assert.ok(secretScan.includes("        run: mise install"));
  assert.ok(workflowStep(secretScan, "secret scan").includes("        run: mise run secret-scan"));
  assert.ok(
    workflowStep(secretScan, "docs conformance gate").includes("        run: mise run check-docs"),
  );
  assert.doesNotMatch(secretScan, /run: bash experiments\//, "shell gates must use managed mise tasks");
  assertWorkflowOrder(secretScan, [
    "      - name: Install tools from mise.toml",
    "      - name: secret scan",
    "      - name: docs conformance gate",
  ]);
});

test("PR sanity preserves required security contexts and read-only concurrency", () => {
  const workflow = source(".github/workflows/pr-sanity.yml");
  const govulncheck = workflowJob("govulncheck");

  assert.match(workflow, /^permissions:\n  contents: read$/m);
  assert.match(workflow, /^  group: pr-sanity-\$\{\{ github\.event_name \}\}-\$\{\{ github\.ref \}\}$/m);
  assert.match(workflow, /^  cancel-in-progress: true$/m);
  assert.match(govulncheck, /^    name: govulncheck$/m);
  assert.match(govulncheck, /^    runs-on: ubuntu-latest$/m);
  assert.ok(govulncheck.includes("      - uses: actions/checkout@v4"));
  assert.ok(
    govulncheck.includes(
      "        run: mise exec -- go run golang.org/x/vuln/cmd/govulncheck@latest ./...",
    ),
  );
});
