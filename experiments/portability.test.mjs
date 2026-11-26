// Static portability contract for the detached-process fixtures. These checks
// are offline and run on every platform: the default Go suite must not depend
// on a host session-creation utility.

import { test } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
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

function assertNoBashDependency(document, label) {
  assert.doesNotMatch(
    document,
    /brew install bash|brew --prefix[^\n]*GITHUB_PATH|ZCODE_ACP_REQUIRED_BASH|mise exec -- bash|run: bash\b/,
    `${label} must not install or invoke Bash for a standard gate`,
  );
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

test("Go commands replace the docs and secret shell gates", () => {
  assert.equal(
    existsSync(path.join(repoRoot, "experiments/check-docs.sh")),
    false,
    "the shell docs gate must be deleted",
  );
  assert.equal(
    existsSync(path.join(repoRoot, "experiments/scan-secrets.sh")),
    false,
    "the shell secret gate must be deleted",
  );

  const docsSource = source("cmd/check-docs/main.go");
  assert.doesNotMatch(docsSource, /"os\/exec"/, "the docs gate must not shell out");
  assert.doesNotMatch(docsSource, /exec\.Command/, "the docs gate must not invoke grep or find");
  assert.ok(docsSource.includes("filepath.WalkDir"), "the docs gate must use filesystem traversal");
  assert.ok(docsSource.includes("os.ReadFile"), "the docs gate must read files portably");

  const secretSource = source("cmd/scan-secrets/main.go");
  assert.ok(
    secretSource.includes(
      'exec.CommandContext(context.Background(), "git", "ls-files", "-z")',
    ),
    "the secret gate must use git ls-files for tracked scope",
  );
  assert.ok(secretSource.includes("[REDACTED]"), "the secret gate must redact matched values");
});

test("mise exposes the Go gates and no managed Bash dependency", () => {
  const miseSource = source("mise.toml");
  const tools = tomlSection(miseSource, "tools");
  const environment = tomlSection(miseSource, "env");

  assert.doesNotMatch(tools, /^bash\s*=/m, "Bash must not be a mise tool-registry entry");
  assert.doesNotMatch(
    environment,
    /ZCODE_ACP_REQUIRED_BASH/,
    "the Bash environment requirement must be removed",
  );
  assert.match(
    tomlSection(miseSource, "tasks.secret-scan"),
    /^run\s*=\s*"go run \.\/cmd\/scan-secrets"\s*$/m,
    "secret-scan must run its Go command",
  );
  assert.match(
    tomlSection(miseSource, "tasks.check-docs"),
    /^run\s*=\s*"go run \.\/cmd\/check-docs"\s*$/m,
    "check-docs must run its Go command",
  );
});

test("CI includes Go gates while preserving native race and Linux VM separation", () => {
  const miseSource = source("mise.toml");

  assert.match(
    tomlSection(miseSource, "tasks.ci"),
    /^depends\s*=\s*\["build", "vet", "lint", "test", "test-experiments", "secret-scan", "check-docs"\]\s*$/m,
    "Go gates must follow test-experiments in the canonical CI graph",
  );
  assert.match(
    tomlSection(miseSource, "tasks.test"),
    /^run\s*=\s*"go test -race -count=1 \.\/\.\.\."\s*$/m,
    "native race testing must remain CGO-enabled and unchanged",
  );
  assert.match(
    tomlSection(miseSource, "tasks.test-experiments"),
    /^run\s*=\s*"node --test experiments\/\*\.test\.mjs"\s*$/m,
    "the experiment task must use an explicit file glob across the Node line",
  );
  assert.doesNotMatch(miseSource, /CGO_ENABLED\s*=/m, "CGO settings must not be globalized");
  assert.doesNotMatch(miseSource, /vm-e2e\.sh/, "the Linux-only VM script must stay outside CI");
});

test("tool-version gate reads only the four mise tool pins", () => {
  const gateSource = source("scripts/check-mise-tool-versions.mjs");
  const expectedTools = ["go", "golangci-lint", "goreleaser", "node"];
  const requiredBlock = /const REQUIRED_TOOLS = \[([\s\S]*?)\];/.exec(gateSource)?.[1];
  assert.ok(requiredBlock, "the gate must declare its four required tools");
  const gateTools = [...requiredBlock.matchAll(/"([^"]+)"/g)].map((match) => match[1]);
  assert.deepEqual(gateTools, expectedTools);
  assert.doesNotMatch(gateSource, /bash|BASH/, "the tool-version gate must not probe Bash");

  const miseSource = source("mise.toml");
  const toolsTable = tomlSection(miseSource, "tools");
  const pinnedTools = [...toolsTable.matchAll(/^([a-z-]+)\s*=\s*"([^"]+)"\s*$/gm)].map(
    (match) => match[1],
  );
  assert.deepEqual(pinnedTools, expectedTools, "mise.toml must contain exactly four tool pins");
  assert.doesNotMatch(tomlSection(miseSource, "env"), /BASH|bash/, "no Bash env pin may remain");

  assert.ok(
    gateSource.includes('readFileSync(path.join(REPO_ROOT, "mise.toml"), "utf8")'),
    "the gate must read requirements from mise.toml",
  );
  assert.ok(
    gateSource.includes(
      'probeVersion(tool, "mise", ["exec", "--", tool, ...VERSION_ARGS[tool]], required, "mise")',
    ),
    "the four managed tools must probe through mise exec",
  );
  assert.doesNotMatch(
    gateSource,
    /(?:spawnSync|execFileSync|execSync)\(\s*tool\b/,
    "managed tools must never fall back to a bare PATH command",
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

test("macOS arm64 runs preflights and canonical gates without Bash setup", () => {
  const arm64 = workflowJob("macos-arm64");

  assert.match(arm64, /^    name: build \/ vet \/ lint \/ test \/ tidy \(macOS arm64\)$/m);
  assert.match(arm64, /^    runs-on: macos-latest$/m);
  assert.match(arm64, /^    timeout-minutes: 30$/m);
  assert.ok(arm64.includes("      - uses: actions/checkout@v4"));
  assert.ok(arm64.includes("        run: curl https://mise.run | sh"));
  assert.ok(arm64.includes('        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"'));
  assertNoBashDependency(arm64, "macOS arm64");
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

test("macOS amd64 mirrors preflights and canonical gates without Bash setup", () => {
  const amd64 = workflowJob("macos-amd64");

  assert.match(amd64, /^    name: build \/ vet \/ lint \/ test \/ tidy \(macOS amd64\)$/m);
  assert.match(amd64, /^    runs-on: macos-15-intel$/m);
  assert.match(amd64, /^    timeout-minutes: 30$/m);
  assert.ok(amd64.includes("      - uses: actions/checkout@v4"));
  assert.ok(amd64.includes("        run: curl https://mise.run | sh"));
  assert.ok(amd64.includes('        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"'));
  assertNoBashDependency(amd64, "macOS amd64");
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

test("cross-build checks tool pins and the GoReleaser config before all five CGO-free targets", () => {
  const crossBuild = workflowJob("cross-build");
  const targets = [
    ["Cross-build linux/arm64", "linux", "arm64"],
    ["Cross-build darwin/arm64", "darwin", "arm64"],
    ["Cross-build darwin/amd64", "darwin", "amd64"],
    ["Cross-build windows/amd64", "windows", "amd64"],
  ];

  assert.match(crossBuild, /^    name: cross-build$/m);
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
  assert.ok(
    workflowStep(crossBuild, "Validate GoReleaser config").includes(
      "        run: GOPROXY=off mise exec -- goreleaser check",
    ),
  );
  for (const [step, goos, goarch] of targets) {
    assert.ok(
      workflowStep(crossBuild, step).includes(
        `        run: mise exec -- env CGO_ENABLED=0 GOOS=${goos} GOARCH=${goarch} go build ./...`,
      ),
      `${step} must be the literal CGO-free command`,
    );
  }
  assertWorkflowOrder(crossBuild, [
    "      - name: Install tools from mise.toml",
    "      - name: Verify mise tool versions",
    "      - name: Validate GoReleaser config",
    "      - name: Cross-build linux/arm64",
    "      - name: Cross-build darwin/arm64",
    "      - name: Cross-build darwin/amd64",
    "      - name: Cross-build windows/amd64",
  ]);
});

test("standard workflows install and invoke no Bash dependency", () => {
  for (const workflow of [".github/workflows/pr-sanity.yml", ".github/workflows/release.yml"]) {
    assertNoBashDependency(source(workflow), workflow);
  }
});

test("secret-scan runs both Go gates through named mise tasks", () => {
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
  assertNoBashDependency(secretScan, "secret-scan job");
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
