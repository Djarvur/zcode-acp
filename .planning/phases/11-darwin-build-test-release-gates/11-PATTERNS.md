# Phase 11: Darwin Build, Test & Release Gates - Pattern Map

**Mapped:** 2026-11-25
**Files analyzed:** 15 new/modified files
**Analogs found:** 12 / 15 (`.goreleaser.yaml`, `scripts/check-mise-tool-versions.mjs`, and `scripts/verify-darwin-release.sh` have explicit no-analog dispositions)

All analog paths below were verified with `git ls-files` in the target repository. No mise configuration was executed and no network-dependent command was run.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/supervisor/mount.go` | utility | file-I/O | Itself: `MountOpts.Statfs` / `defaultStatfs` | exact |
| `internal/supervisor/mount_test.go` | test | event-driven + file-I/O | Itself: `TestMountStop`; `internal/supervisor/spawn_test.go` for re-exec | exact / role-match |
| `cmd/zcode-acp/main_test.go` | test | event-driven + file-I/O | Itself: ignore-TERM fixture and PID cleanup; `internal/supervisor/spawn_test.go` for re-exec | exact / role-match |
| `internal/supervisor/mount_internal_test.go` | test | file-I/O | Itself: in-package unexported-function tests | exact; implied by the direct-converter test gap |
| `experiments/portability.test.mjs` | test | file-I/O (static source inspection) | `experiments/vm-e2e-script.test.mjs` | exact |
| `mise.toml` | config | batch | Itself: `[tools]` and `[tasks.*]` | exact |
| `.github/workflows/pr-sanity.yml` | config | batch | Itself: `sanity` job; `.github/workflows/e2e-matrix.yml` for dispatch/concurrency only | exact / partial |
| `.github/workflows/release.yml` | config | batch + file-I/O | Itself: checkout/mise/GoReleaser job; `.github/workflows/e2e-matrix.yml` for serial concurrency | exact / role-match |
| `.goreleaser.yaml` | config | batch + file-I/O | None in the tracked tree or git history | no analog |

`internal/supervisor/mount_internal_test.go` is implied by RESEARCH.md:526,542: existing mount tests inject `MountOpts.Statfs`, so they do not exercise the production `defaultStatfs` converter. Add the direct test there only if the planner keeps that explicit Wave 0 gap.

## Pattern Assignments

### `internal/supervisor/mount.go` (utility, file-I/O)

**Analog:** `internal/supervisor/mount.go:158-202` (the existing injected seam and production converter).

**Imports pattern** (lines 27-40): the file already imports `fmt` and `syscall`; the fix needs no new import.

**Core seam to preserve** (lines 158-187):

```go
type MountOpts struct {
	// ...
	// Statfs reports path's filesystem type; nil takes syscall.Statfs.
	Statfs func(path string) (fsType int64, err error)
}

func DefaultMountOpts() MountOpts {
	return MountOpts{
		// ...
		Statfs: defaultStatfs,
	}
}
```

**Exact converter change** (lines 190-202): replace only the final return. Keep the `%w` error wrap and the `int64` function signature.

```go
func defaultStatfs(path string) (int64, error) {
	var st syscall.Statfs_t

	if err := syscall.Statfs(path, &st); err != nil {
		return 0, fmt.Errorf("zcode: statfs %s: %w", path, err)
	}

	return int64(st.Type), nil
}
```

**Do not copy/churn:**

- Do not cast through `int`.
- Do not add OS-specific build-tagged mount implementations.
- Do not change `MountOpts.Statfs` or its injected-test contract.
- Update the stale lines 190-193 comment that says no portable variant is added; the single implementation is now deliberately portable across Linux and Darwin field widths.

---

### `internal/supervisor/mount_test.go` (test, event-driven + file-I/O)

**Analogs:** the existing `TestMountStop` in this file; `internal/supervisor/spawn_test.go:44-45,143-183` for test-binary re-exec.

**Current fixture to replace** (lines 195-207): the executable `setsid` call is inside the raw shell fixture, not merely in a comment.

```go
// fakeScriptStubborn ignores SIGTERM and keeps the tree alive; a detached
// (setsid) short-lived process removes the tree after the helper is
// killed — the fake analogue of the real keepalive-pipe unmount.
fakeScriptStubborn = fakeMountPreamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
trap '' TERM
setsid sh -c 'sleep 0.5; rm -rf "$1"' _ "$dir" >/dev/null 2>&1 &
printf '%s\n' "$dir"
while :; do sleep 0.05; done
`
```

**Lifecycle assertions to retain** (lines 448-493): `PrepareMount`, nonzero `HelperPID`, `StopGrace` elapsed, helper gone, mount root gone, private base gone, idempotent second `Stop`.

```go
mount, err := supervisor.PrepareMount(context.Background(),
	supervisor.Install{AppImage: appImage, Version: fakeAppImageVer}, testMountOpts())
// ...
start := time.Now()
stopErr := mount.Stop()
elapsed := time.Since(start)
// ...
assertProcessGone(t, pid, "the stubborn mount helper")
assertMountPathGone(t, mount.Root(), "the mount root")
assertMountPathGone(t, filepath.Dir(mount.Root()), "the private base")
```

**Re-exec pattern to copy** (`internal/supervisor/spawn_test.go:44-45,143-183`):

```go
const helperRunFlag = "-test.run=^TestSpawnEnvHelper$"

func TestSpawnEnvHelper(t *testing.T) {
	if os.Getenv(childRunAsNodeEnv) == "1" {
		runSpawnEnvHelper()
	}
}

// Parent:
child, err := supervisor.SpawnChild(ctx, []string{os.Args[0], helperRunFlag}, childEnv, supervisor.SpawnOpts{})
if err != nil {
	t.Fatalf("spawn the helper child through the exported SpawnChild path: %v", err)
}
t.Cleanup(func() { terminateSpawnChild(t, child) })
```

**Phase 11 adaptation (proposed, not existing code):**

1. Remove the `setsid` line from `fakeScriptStubborn`; keep the TERM-ignoring helper alive.
2. In `TestMountStop`, after `PrepareMount` reveals `mount.Root()`, start the current test binary with a unique selector and environment marker.
3. Set `SysProcAttr{Setsid: true}` on that command. Do not also set `Setpgid`.
4. The selected helper records its PID, sleeps for the existing bounded 500 ms fixture delay, removes only the supplied mount root, and exits.
5. Register unconditional parent cleanup for the recorded PID. A new session escapes the mount helper's process-group kill, so group cleanup is insufficient.

```go
// Proposed shape; finalize helper and marker names in the plan.
cmd := exec.Command(os.Args[0], "-test.run=^TestDetachedMountCleanerHelper$")
cmd.Env = append(os.Environ(),
	"ZCODE_ACP_TEST_DETACHED_CLEANER=1",
	"FIXTURE_MOUNT_ROOT="+mount.Root(),
)
cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
```

**Error/cleanup pattern:** fail the parent immediately on `cmd.Start()` failure; use a PID file or equivalent recorded PID and unconditional `t.Cleanup`; keep the existing `testMountReapDeadline` as the bounded disappearance budget.

---

### `cmd/zcode-acp/main_test.go` (test, event-driven + file-I/O)

**Analogs:** this file's ignore-TERM fixture, wrapper start, and recorded-PID cleanup; `internal/supervisor/spawn_test.go:143-183` for helper selection and early environment return.

**Current executable fixture to replace** (lines 704-718):

```go
case e2eFailureIgnoreTerm:
	// The descendant keeps the inherited stderr pipe open, so the
	// helper's process tree can never be reaped within ReapDeadline.
	return preamble + `dir=$(mktemp -d "$base/.mount_FakeXXXXXX")
mkdir -p "$dir/resources/glm" "$dir/resources/config/provider"
printf '#!/bin/sh\nexit 0\n' > "$dir/zcode"
chmod 755 "$dir/zcode"
: > "$dir/resources/glm/zcode.cjs"
setsid sh -c 'echo $$ > "@@LEAK_PID@@"; exec sleep 30' &
trap '' TERM
while :; do sleep 0.05; done
`
```

**Load-bearing process contract** (lines 901-935): the detached descendant must exist before SIGTERM and must keep the wrapper's stderr pipe open after the helper's process group is killed.

```go
wrapper := exec.CommandContext(ctx, agentBin, ompFlagAppServer, appImage)
wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
wrapper.Env = append(os.Environ(), "TMPDIR="+tmpRoot)
stderrBuf := &e2eSyncBuffer{}
wrapper.Stderr = stderrBuf
// ...
e2eWaitPIDFile(t, filepath.Join(fixture, e2eFakeHelperPID), stderrBuf)
e2eWaitPIDFile(t, leakPID, stderrBuf)
wrapper.Process.Signal(syscall.SIGTERM)
```

**Existing cleanup to retain** (lines 913-916,956-966):

```go
leakPID := filepath.Join(fixture, e2eFakeLeakPID)
t.Cleanup(func() { e2eKillRecordedPID(leakPID) })

func e2eKillRecordedPID(path string) {
	pid := e2eReadPID(path)
	if pid <= 0 {
		return
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
```

**Phase 11 adaptation (proposed):**

- Replace the shell `setsid` command with re-exec of the current Go test binary.
- Give every helper test a unique `-test.run=^Test...Helper$` selector and an early environment-marker return, following `TestSpawnEnvHelper`.
- The helper that writes `leak.pid` must sleep for a bounded interval (the current fixture uses 30 seconds) and then exit.
- Preserve `wrapper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` for the ordinary wrapper; use `Setsid` only for the detached descendant.

**Critical inherited-stderr constraint:** do not simply launch the sleeping descendant from the test parent. The present descendant inherits the *wrapper child's* stderr pipe (`main_test.go:704-709,925-926`); the test parent's stderr is not that pipe. Launch a re-executed Go launcher from the fake-runtime context, then have that launcher start the selected sleeping helper with `SysProcAttr{Setsid: true}` and explicitly wire `cmd.Stderr = os.Stderr` (and stdout if required). The launcher may remain in the wrapper's group; the inner helper is the detached session that holds the pipe open. Keep `e2eWaitPIDFile` and unconditional recorded-PID cleanup unchanged.

---

### `internal/supervisor/mount_internal_test.go` (test, file-I/O; implied)

**Analog:** this file's in-package tests for unexported `helperEnv` and `MountFailure`.

**Imports/test style** (lines 9-17,27-32):

```go
package supervisor

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
)

func TestMountEnv(t *testing.T) {
	t.Run("allowlist by construction", testMountEnvAllowlist)
	t.Run("PATH fallback when its parent has none", testMountEnvPathFallback)
}
```

**Why this file:** `mount_test.go` is `package supervisor_test` and cannot call `defaultStatfs`. Its existing `TestMountTeardownFailure` only injects a fake through `opts.Statfs` (`mount_test.go:982-990`).

**Direct converter test shape (proposed):** add `syscall` to imports, call `defaultStatfs(t.TempDir())`, independently populate `syscall.Statfs_t`, and compare with `int64(st.Type)`. Assert the returned value is usable as the seam's `int64` and that syscall errors are wrapped. This is a local filesystem test, not a desktop/credential test.

---

### `experiments/portability.test.mjs` (test, static file-I/O)

**Analog:** `experiments/vm-e2e-script.test.mjs:14-26,44-82`.

**Imports and repository-relative path pattern:**

```js
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const scriptPath = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
  'scripts',
  'vm-e2e.sh',
);
const scriptSource = () => readFileSync(scriptPath, 'utf8');
```

**Static assertion pattern** (lines 46-66):

```js
test('no-keyring stage asserts the four coded keyring sentinels verbatim', () => {
  const src = scriptSource();
  for (const [arm, sentence] of Object.entries(SENTINELS)) {
    assert.ok(
      src.includes(sentence),
      `the ${arm} sentinel sentence is missing: ${sentence}`,
    );
  }
});
```

**Phase 11 assertions to implement:**

- Read both modified Go test files and reject an executable line matching `setsid` (for example `(?m)^\s*setsid\s+`); comments may still explain the former fixture.
- Read `mise.toml` and require the managed Bash pin.
- Read `.github/workflows/pr-sanity.yml` and reject bare `run: bash experiments/check-docs.sh` / `scan-secrets.sh`; require a mise task or `mise exec -- bash ...`.
- Keep this test static and offline. Do not invoke mise, GoReleaser, a desktop, Keychain, or a network service.

The guarded live-shell pattern at `vm-e2e-script.test.mjs:84-105` is not the model for this phase: the new portability checks must run in the default suite on every platform.

---

### `mise.toml` (config, batch)

**Analog:** `mise.toml:1-55` itself.

**Single-source tool pin pattern** (lines 1-19):

```toml
[tools]
go = "1.27"
golangci-lint = "2.15.0"
goreleaser = "2.20.2"
node = "26"
```

Add the researched Bash `5.3` pin here, not in workflow inputs. The exact patch/availability remains an execution checkpoint (RESEARCH assumption A1).

**Task pattern** (lines 25-55):

```toml
[tasks.build]
description = "Build all packages"
run = "go build ./..."

[tasks.test]
description = "Run all Go tests with the race detector"
run = "go test -race -count=1 ./..."

[tasks.test-experiments]
description = "Run the Phase-1 experiments suite (node --test)"
run = "node --test experiments/"

[tasks.ci]
description = "Full CI gate: build + vet + lint + test + test-experiments (D-02)"
depends = ["build", "vet", "lint", "test", "test-experiments"]
```

**Phase 11 adaptation:**

- Keep native `test` as `go test -race -count=1 ./...`; do not put `CGO_ENABLED=0` in `[env]`.
- Put the two CGO-free Darwin commands in a separate cross-build task/job, not in the native race task.
- Prefer named mise tasks for the shell gates so their `bash` resolves to the managed tool and local/CI commands stay identical. If no tasks are added, workflow steps must use `mise exec -- bash ...`.
- Do not add `scripts/vm-e2e.sh` to `[tasks.ci]`. It is Linux-only: `scripts/vm-e2e.sh:95` uses `readlink -f`, and line 604 uses GNU `timeout`.

---

### `.github/workflows/pr-sanity.yml` (config, batch)

**Analog:** `.github/workflows/pr-sanity.yml:29-91` itself. `.github/workflows/e2e-matrix.yml` supplies dispatch/concurrency discipline, but the repository currently has no `strategy.matrix` precedent.

**Required check and mise bootstrap to preserve** (lines 47-91):

```yaml
jobs:
  sanity:
    # This string is a branch-protection required check name.
    name: build / vet / lint / test / tidy
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4
      - name: Install mise
        run: curl https://mise.run | sh
      - name: Add mise to PATH
        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"
      - name: Install tools from mise.toml
        run: mise install
      - name: build
        run: mise run build
      - name: vet
        run: mise run vet
      - name: golangci-lint
        run: mise run lint
      - name: test (go)
        run: mise run test
      - name: test (experiments suite)
        run: mise run test-experiments
      - name: modules are tidy
        run: mise run tidy-diff
```

**Phase 11 job shape (proposed):**

- Leave the Linux `sanity` job and its exact name unchanged.
- Add a native macOS matrix with `fail-fast: false` and explicit `include` entries for runner and expected architecture. Do not map both entries to `macos-latest`.
- Add an architecture preflight before tests, comparing `go env GOARCH` with a matrix value passed through `env`.
- Run the same mise build/vet/lint/test/experiment/tidy tasks natively. Do not set `CGO_ENABLED=0` for `-race`.
- Add a separate Darwin cross-build job with both literal required commands.

```yaml
# Proposed; verify the Intel runner label at execution.
strategy:
  fail-fast: false
  matrix:
    include:
      - arch: arm64
        runner: macos-latest
      - arch: amd64
        runner: macos-15-intel
runs-on: ${{ matrix.runner }}
# ...
- name: Verify native architecture
  env:
    EXPECTED_GOARCH: ${{ matrix.arch }}
  run: test "$(go env GOARCH)" = "$EXPECTED_GOARCH"
```

```yaml
- name: Cross-build darwin/arm64
  run: CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...
- name: Cross-build darwin/amd64
  run: CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...
```

**Shell-gate seam to change** (current lines 121-137): the `secret-scan` job currently invokes system `bash` directly and does not install mise.

```yaml
- name: secret scan
  run: bash experiments/scan-secrets.sh
- name: docs conformance gate
  run: bash experiments/check-docs.sh
```

Route these through managed Bash; if using mise tasks, add the same checkout/mise install/bootstrap steps used by `sanity`. `check-docs.sh:58` uses `declare -A`, and line 127 uses `read -ra`, so stock macOS Bash 3.2 is not an acceptable interpreter.

---

### `.github/workflows/release.yml` (config, batch + file-I/O)

**Analog:** `.github/workflows/release.yml:23-69` itself; `.github/workflows/e2e-matrix.yml:31-37` for non-cancelling serial concurrency.

**Existing release skeleton to preserve** (lines 23-69):

```yaml
on:
  push:
    tags: ['v*']

permissions:
  contents: write

concurrency:
  group: release
  cancel-in-progress: false

jobs:
  release:
    name: goreleaser release
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Install mise
        run: curl https://mise.run | sh
      - name: Add mise to PATH
        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"
      - name: Install tools from mise.toml
        run: mise install
      - name: goreleaser release
        run: mise exec -- goreleaser release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Phase 11 adaptation:**

- Add `workflow_dispatch` for snapshots.
- Move GoReleaser jobs to macOS so `lipo` is available.
- Give the snapshot job `permissions: contents: read`; do not pass `GITHUB_TOKEN` to it. Keep `contents: write` scoped to the tag-publish job.
- Keep `fetch-depth: 0`, mise as the only version source, and `cancel-in-progress: false`.
- In both jobs, run the mise version gate, then a five-minute online `mise exec -- go mod download` with `GOPROXY=https://proxy.golang.org`, reject any go.mod/go.sum drift, and run `GOPROXY=off mise exec -- go mod verify` plus `GOPROXY=off mise run tidy-diff` before any offline GoReleaser command. `mise install` hydrates tools, not project modules; no pre-populated module cache is assumed.
- Validate the config with the pinned tool before producing artifacts.

```yaml
# Proposed dependency and snapshot command sequence for each macOS job.
- name: Validate tool versions
  run: mise exec -- node scripts/check-mise-tool-versions.mjs go goreleaser bash
- name: Download frozen Go modules
  timeout-minutes: 5
  env:
    GOPROXY: https://proxy.golang.org
  run: |
    mise exec -- go mod download
    git diff --exit-code HEAD -- go.mod go.sum
- name: Verify frozen modules offline
  run: |
    GOPROXY=off mise exec -- go mod verify
    GOPROXY=off mise run tidy-diff
- name: Validate GoReleaser config
  run: GOPROXY=off mise exec -- goreleaser check
- name: Build release snapshot
  run: GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish
- name: Verify checksums
  working-directory: dist
  run: shasum -a 256 -c checksums.txt
```

**Artifact verification pattern:** locate exactly one arm64, amd64, and universal archive, verify `checksums.txt` from the same `dist/`, require each archive's complete member/type list to be one regular `zcode-acp`, then extract into a fresh `mktemp -d` directory and assert exact thin/universal slices. Do not execute the extracted binary.

```sh
archs=$(lipo -archs "$verify_dir/zcode-acp")
printf '%s\n' "$archs" | grep -qw arm64
printf '%s\n' "$archs" | grep -qw x86_64
```

Assert exact architecture-token sets, including both universal slices. Upload all three binary-only archives and `checksums.txt` as snapshot artifacts.

---

### `.goreleaser.yaml` (config, batch + file-I/O)

**Analog:** none. `git ls-files` and `git log --all -- .goreleaser.yaml .goreleaser.yml` found no repository configuration. Use RESEARCH.md:240-286 as the schema-derived starting point, then validate with the pinned GoReleaser `2.20.2`; the local `2.17.1` schema observation is not sufficient certification.

**Version stamp analog:** `cmd/zcode-acp/main.go:37-45` already exposes the linker seam.

```go
var version = "18.1.9+zacp.0"
```

**Configuration skeleton from RESEARCH.md:246-284:**

```yaml
version: 2

project_name: zcode-acp

builds:
  - id: darwin-arm64
    main: ./cmd/zcode-acp
    binary: zcode-acp
    env: [CGO_ENABLED=0]
    goos: [darwin]
    goarch: [arm64]
    flags: [-trimpath]
    ldflags: [-s, -w, -X, "main.version={{ .Version }}"]
  - id: darwin-amd64
    main: ./cmd/zcode-acp
    binary: zcode-acp
    env: [CGO_ENABLED=0]
    goos: [darwin]
    goarch: [amd64]
    flags: [-trimpath]
    ldflags: [-s, -w, -X, "main.version={{ .Version }}"]

universal_binaries:
  - id: darwin-universal
    ids: [darwin-arm64, darwin-amd64]
    name_template: zcode-acp
    replace: false

archives:
  - id: darwin-thin
    builds: [darwin-arm64, darwin-amd64]
    formats: [tar.gz]
    files: [zcode-acp]
    name_template: "zcode-acp_{{ .Version }}_darwin_{{ .Arch }}"
  - id: darwin-universal
    builds: [darwin-universal]
    formats: [tar.gz]
    files: [zcode-acp]
    name_template: "zcode-acp_{{ .Version }}_darwin_universal"

checksum:
  name_template: checksums.txt
  algorithm: sha256
```

**Load-binding details:**

- Both thin builds use `CGO_ENABLED=0` and the existing `./cmd/zcode-acp` main package.
- `universal_binaries.ids` must name both thin build IDs.
- `replace: false` retains both thin build outputs while adding the universal binary.
- The thin archive references both thin IDs; the universal archive references `darwin-universal`.
- Both archive definitions set `files: [zcode-acp]`, excluding default README/LICENSE members.
- SHA-256 checksums must cover all three archives in the same snapshot `dist/`.
- Run `goreleaser check` with `2.20.2` before trusting field compatibility.

## Shared Patterns

### Test-binary re-exec

**Sources:** `internal/supervisor/spawn_test.go:44-45,143-183`; `internal/supervisor/supervise_test.go:51,108,392` (located, same idiom).
**Apply to:** both detached-process fixtures.

- Re-execute `os.Args[0]`.
- Select one helper with an anchored `-test.run=^Test...Helper$`.
- Gate the helper body on a dedicated environment marker and return immediately otherwise.
- Keep the helper's effect bounded and its PID observable.
- Register cleanup in the parent, not only on failure.

### Process-group versus session semantics

**Sources:** `internal/supervisor/spawn.go:259`; `cmd/zcode-acp/main_test.go:193,921-923`; RESEARCH.md:205-218,290-296.
**Apply to:** all process fixtures.

```go
// Ordinary wrapper/helper child: killable as a group.
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

// Detached fixture only: new session; never combine with Setpgid.
cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
```

A detached helper needs direct PID cleanup because negative-PGID group kills do not reach its new session. For the ignore-TERM fixture, inherited wrapper stderr is part of the behavior under test and must survive the helper kill.

### Cross-build versus native race execution

**Sources:** `mise.toml:37-39`; RESEARCH.md:220-232,352-357.
**Apply to:** mise tasks and all CI jobs.

- Native Linux/macOS jobs: `go test -race -count=1 ./...` with CGO enabled.
- Darwin compile/release gates: `CGO_ENABLED=0 GOOS=darwin GOARCH=<arch> go build ./...`.
- Never execute a cross-compiled Darwin test on Linux.

### Tool-version and workflow discipline

**Sources:** `mise.toml:1-19`; `.github/workflows/pr-sanity.yml:20-24,58-69`; `.github/workflows/release.yml:45-67`.
**Apply to:** all build/release files.

- `mise.toml` is the only tool-version source.
- CI installs through `mise install` and runs the same tasks as local development.
- Preserve `actions/checkout@v4` and `fetch-depth: 0` for release history unless consciously upgraded.
- Preserve the exact Linux required-check name `build / vet / lint / test / tidy`.
- PR/snapshot jobs stay `contents: read`; only tag publication gets `contents: write`.

### Error handling and cleanup

**Sources:** `internal/supervisor/mount.go:194-202`; `cmd/zcode-acp/main_test.go:956-966`.
**Apply to:** converter and test helpers.

- Wrap syscall errors with `%w` and path context.
- Fail loudly on helper start/write failures.
- Always clean recorded helper PIDs, even when the assertion path succeeds.
- Do not let a failed fixture leave a sleeping process or mount tree behind.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `.goreleaser.yaml` | config | batch + file-I/O | No tracked or historical GoReleaser config exists. Use RESEARCH.md's schema-derived skeleton and validate with pinned `2.20.2`. |
| `scripts/check-mise-tool-versions.mjs` | executable gate | batch + process-I/O | No existing mise-resolved version comparator; use ESM CLI conventions only. |
| `scripts/verify-darwin-release.sh` | executable verifier | file-I/O + process-I/O | No existing release verifier; use shell safety discipline, not VM behavior. |

**Additional pattern gap:** no existing tracked workflow uses `strategy.matrix`. The native macOS matrix is new; preserve the existing `sanity` job as the structural analog and use explicit `include` entries plus an architecture preflight. The Intel runner label remains an execution-time assumption.

## Additional File Coverage

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `scripts/check-mise-tool-versions.mjs` | executable gate | batch + process-I/O | No existing tool-version gate; `experiments/scrub-capture.mjs` for ESM CLI structure | no analog / role-match |
| `experiments/tool-versions.test.mjs` | test | pure function-I/O | `experiments/scrub-capture.test.mjs` | role-match |
| `scripts/verify-darwin-release.sh` | executable verifier | file-I/O + process-I/O | No existing release verifier; `scripts/vm-e2e.sh` for bounded shell/check discipline only | no analog / role-match |
| `experiments/release.test.mjs` | test | static file-I/O + guarded fixture-I/O | `experiments/vm-e2e-script.test.mjs` and `experiments/scrub-capture.test.mjs` | role-match |
| `.gitignore` | config | batch | Itself: root binary and experiment-output exclusions | exact |
| `experiments/check-docs.sh` | executable gate | file-I/O | Itself: docs/provenance checks; `experiments/scan-secrets.sh` for repo-root pinning | exact |

### `scripts/check-mise-tool-versions.mjs` (executable gate, batch + process-I/O)

**Analog disposition:** no analog — no tracked script currently compares mise-resolved tool output with repository pins. Use the ESM/direct-invocation shape of `experiments/scrub-capture.mjs`, but do not copy its file-transform behavior.

**Required pattern:** built-in Node imports only; read `mise.toml` as data; parse the five `[tools]` entries (`go`, `golangci-lint`, `goreleaser`, `node`, `bash`); export `resolvedVersionFromOutput` and `requiredVersionMatches`; guard main execution with the ESM direct-invocation check. Probe only through `mise exec -- <tool> <version-args>`, bound each probe, and set a non-zero exit on unknown tool, malformed/missing pin, failed probe, missing version, or mismatch. Never install/trust tools, fall back to bare PATH, or print credential material.

### `experiments/tool-versions.test.mjs` (test, pure function-I/O)

**Analog:** `experiments/scrub-capture.test.mjs:10-19,33-49` for `node:test`, strict assertions, and a small deterministic harness.

**Required pattern:** import only the two exported comparator functions. Table-drive matching exact pins, matching one/two-component lines, wrong patch/minor/major, malformed input, and missing output. Keep the suite offline: it must never spawn mise or inspect the host tool installation. The runnable gate, not the comparator test, owns process-exit behavior.

### `scripts/verify-darwin-release.sh` (executable verifier, file-I/O + process-I/O)

**Analog disposition:** no analog — the tracked tree has scripts for VM acceptance and docs/secret scanning, but none verifies release archives. `scripts/vm-e2e.sh:66-89` supplies only the safety discipline: bounded checks, actionable diagnostics, no credential values, and no implicit downloads.

**Required pattern:** managed Bash 5.3, one required dist argument, `set -euo pipefail`, fresh temporary directories with exit cleanup, standard `shasum`/`tar`/`lipo`, and no extracted-binary execution. Validate the complete tar member list and entry types before extraction. The configured contract is exactly one regular `zcode-acp`; reject traversal, absolute paths, symlinks/hardlinks, directories, missing binary, and extra files. Compare normalized architecture-token sets exactly. Fail non-zero with a diagnostic that names the failed check but never logs secrets.

### `experiments/release.test.mjs` (test, static file-I/O + guarded fixture-I/O)

**Analogs:** `experiments/vm-e2e-script.test.mjs:14-82` for repository-relative static reads; `experiments/scrub-capture.test.mjs:33-49` for temporary-file process fixtures.

**Required pattern:** static tests read `.goreleaser.yaml`, the verifier, and workflow as text and assert the binary-only three-archive contract, safe exact-member verification, read-only snapshot, and tag-only write scope. In each release job, assert the bounded online frozen-module download and unchanged go.mod/go.sum check precede offline `go mod verify`, `tidy-diff`, and every offline GoReleaser command; a cold runner with unavailable modules must stop at the dependency step. Guard adversarial archive execution behind `ZCODE_ACP_RELEASE_FIXTURES=1`; only that explicit mode spawns `mise exec -- bash` against locally generated traversal/symlink/extra/missing-member fixtures. Default `node --test experiments/` stays offline and never invokes mise, GoReleaser, lipo, a desktop, or a service.

### `.gitignore` (config, batch)

**Analog:** `.gitignore:36-42` itself for root-anchored generated-output exclusions.

**Required pattern:** append a commented `/dist/` entry. Keep existing raw-capture and root-binary exclusions unchanged; do not ignore source, checksums outside `dist/`, or release workflow files.

### `experiments/check-docs.sh` (executable gate, file-I/O)

**Analogs:** this file's existing diagnostics and accumulated `rc`; `experiments/scan-secrets.sh:26-28` for pinning to the repository top level.

**Required pattern:** replace only the four GNU-only include-filter scans. Use `find` with explicit `*.md` or `*.mjs`/`*.sh` predicates plus `grep -cE`, `grep -hoE`, or `grep -nHiE`; preserve `sort -u`, zero-count filtering, self-exclusion, PENDING/phase-gate behavior, first-five-hit limits, diagnostic prefixes, and final exit status. Leave the existing portable recursive path scan and Bash associative-array section checks unchanged. Managed Bash 5.3 remains the interpreter selected by mise.

## Metadata

**Analog search scope:** `internal/supervisor/`, `cmd/zcode-acp/`, `experiments/`, `scripts/`, `.github/workflows/`, root build config, and git history for GoReleaser config.
**Files scanned:** 15 tracked source/config files (large Go tests read in non-overlapping targeted ranges).
**Pattern extraction date:** 2026-11-25
**Validation performed:** static source inspection only; no build, test, mise, GoReleaser, network, desktop, credential, or Keychain execution.
