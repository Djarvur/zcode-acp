# Phase 11: Darwin Build, Test & Release Gates - Research

**Researched:** 2026-11-25
**Domain:** Go cross-compilation, portable process tests, CI matrices, macOS universal binary releases
**Confidence:** MEDIUM

<user_constraints>
## User Constraints (from CONTEXT.md)

No `11-CONTEXT.md` exists in the phase directory. The following binding constraints come from the task, `REQUIREMENTS.md`, `ROADMAP.md`, and the confirmed audit facts.

### Locked Decisions

- Phase 11 goal, verbatim: “Developers can build, test, and package the existing bridge for both macOS architectures without Linux-only tool assumptions.” [VERIFIED: .planning/ROADMAP.md:100-108]
- Phase 11 requirements, verbatim: “MBLD-01, MBLD-02, MBLD-03, MTEST-01”. [VERIFIED: .planning/ROADMAP.md:103]
- MBLD-01, verbatim: “The project cross-compiles with `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` and `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...`.” [VERIFIED: .planning/REQUIREMENTS.md:12]
- MBLD-02, verbatim: “Standard macOS build/test gates do not require `setsid`, GNU-only utilities, or the system Bash 3.2; required tool versions are explicit.” [VERIFIED: .planning/REQUIREMENTS.md:14]
- MBLD-03, verbatim: “CI builds and tests Linux plus macOS arm64/amd64, and a release snapshot produces checksummed universal macOS binary artifacts.” [VERIFIED: .planning/REQUIREMENTS.md:16]
- MTEST-01, verbatim: “The full default test suite passes on macOS arm64 without an external `setsid` dependency.” [VERIFIED: .planning/REQUIREMENTS.md:32]
- “No CGO/native Go dependencies; use CGO_ENABLED=0 for Darwin cross-builds.” [VERIFIED: task-provided confirmed audit fact; go.mod:3-5 declares only `go 1.27` and `github.com/coder/acp-go-sdk v0.20.0`]
- “Do not execute untrusted mise config or run network-dependent commands. Static research only.” [VERIFIED: task-provided constraint]
- Automated phases remain zero-spend: “no real desktop, credentials, or Keychain access in CI”. [VERIFIED: .planning/ROADMAP.md:158]

### Claude's Discretion

None recorded.

### Deferred Ideas (OUT OF SCOPE)

- “**FPLAT-02**: macOS `.app` packaging, code signing, and notarization.” [VERIFIED: .planning/REQUIREMENTS.md:48]
- “**FPLAT-03**: Start Plan support on macOS.” [VERIFIED: .planning/REQUIREMENTS.md:49]
- Runtime `.app` discovery, desktop launch, Keychain work, fake-backend Orca acceptance, README work, and live Orca acceptance belong to Phases 12-16, not Phase 11. [VERIFIED: .planning/REQUIREMENTS.md:65-82]
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MBLD-01 | Both exact CGO-free Darwin cross-builds complete. | Cast `Statfs_t.Type` at the existing `int64` seam; gate both exact commands in CI; GoReleaser builds the same two targets. |
| MBLD-02 | Standard macOS gates avoid external `setsid`, GNU utilities, and stock Bash 3.2; tool versions are explicit. | Replace two shell `setsid` fixtures with Go test-binary re-exec helpers using `SysProcAttr.Setsid`; invoke shell gates through a mise-managed Bash; keep the Linux-only VM script outside the macOS default gate. |
| MBLD-03 | CI builds/tests Linux and both macOS architectures; snapshot emits checksummed universal macOS artifacts. | Preserve the existing Linux required-check name, add native macOS jobs, add a macOS GoReleaser snapshot job, and verify SHA-256 plus `lipo -archs`. |
| MTEST-01 | Full default suite passes on macOS arm64 without external `setsid`. | Native `macos-latest` arm64 race-test job plus re-exec helper tests; do not set `CGO_ENABLED=0` for `-race`. |
</phase_requirements>

## Summary

Phase 11 is a gate-and-portability phase, not a runtime-feature phase. The production bridge already uses Go process groups (`Setpgid`) and the only confirmed compile defect is the Darwin width of `syscall.Statfs_t.Type`: the mount seam returns `int64`, Darwin defines the field as `uint32`, and Linux defines it as `int64`. The minimal fix is an explicit `int64(st.Type)` conversion at the existing seam, preserving the injected `Statfs` function and all mount tests. [VERIFIED: internal/supervisor/mount.go:171-175,190-202; Go 1.27.4 `src/syscall/ztypes_darwin_arm64.go:85-102` quotes `Type uint32`; Go 1.27.4 `src/syscall/ztypes_linux_amd64.go:117-130` quotes `Type int64`]

The test portability work is confined to two scripted fixtures that call external `setsid`: the stubborn mount cleaner and the ignore-TERM leaked descendant. Replace both with re-exec of the current Go test binary, selected by a dedicated test name and an environment marker; set `SysProcAttr.Setsid` in the helper. Darwin supports that field natively, so this preserves the detached-session behavior without a host utility. Keep ordinary wrapper children on `Setpgid`. [VERIFIED: internal/supervisor/mount_test.go:195-206 quotes `setsid sh -c 'sleep 0.5; rm -rf "$1"' _ "$dir"`; cmd/zcode-acp/main_test.go:704-717 quotes `setsid sh -c 'echo $$ > "@@LEAK_PID@@"; exec sleep 30' &`; Go 1.27.4 `src/syscall/exec_libc2.go:5-22` is tagged `darwin || openbsd` and quotes `Setsid bool // Create session.` and `Setpgid bool`]

CI and release need structural changes. The current sanity workflow is Ubuntu-only, and the release workflow runs GoReleaser on Ubuntu even though no GoReleaser config exists. Build/test Linux and both macOS architectures on native runners, cross-build both Darwin targets with CGO disabled, and run the release snapshot on macOS so GoReleaser can use the platform's universal-binary tooling. Publish one universal archive containing a fat `zcode-acp`, with SHA-256 checksums, then verify both checksum and architecture slices. [VERIFIED: .github/workflows/pr-sanity.yml:47-54; .github/workflows/release.yml:39-67; release.yml:15-19 states no `.goreleaser.yaml` exists]

**Primary recommendation:** Fix the `Statfs` conversion, replace both `setsid` fixture calls with Go re-exec helpers, add a mise-managed Bash for shell gates, extend CI with native macOS arm64/amd64 jobs plus exact Darwin cross-builds, and add a macOS GoReleaser snapshot that produces and verifies a universal archive plus SHA-256 checksums.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Darwin compile compatibility | API / Backend (Go packages) | Build tooling | The type mismatch is in the supervisor's filesystem predicate; no ACP or desktop behavior changes. |
| Portable detached-process fixtures | Test tier | API / Backend | Process setup belongs in test helpers, using Go's process API rather than a host command. |
| Cross-build and native test gates | CI / Build pipeline | Developer tooling | The same pinned mise tasks run locally and in CI; architecture-specific execution requires native runners. |
| Universal binary creation and checksums | Release pipeline | CDN / Static artifacts | GoReleaser owns multi-arch build orchestration and checksum generation; the snapshot verifies the resulting artifact. |
| Runtime macOS discovery/launch/auth | Later phases | — | Explicitly outside Phase 11. |

## Project Constraints (from CLAUDE.md)

The target repository's configured project instructions are `.claude/CLAUDE.md`. Actionable directives:

- **Tech stack:** Go; the only backend is ZCode Desktop. [VERIFIED: .claude/CLAUDE.md:18-21]
- **Research policy:** investigate ZCode from source first, then experiments; investigate Orca from source first. [VERIFIED: .claude/CLAUDE.md:22-27]
- **Docs language:** English for README, recipes, and `docs/`. [VERIFIED: .claude/CLAUDE.md:27]
- **Repository shape:** strict linter, GitHub Actions, protected branch. [VERIFIED: .claude/CLAUDE.md:28-29]
- **Git workflow:** feature branches from fresh `master`; no direct main-branch work. [VERIFIED: .claude/CLAUDE.md:30-31]
- **GSD enforcement:** work through a GSD workflow; do not make direct repository edits outside it unless explicitly bypassed. [VERIFIED: .claude/CLAUDE.md:178-190]
- No project skills were found by the configured project-skills locations. [VERIFIED: .claude/CLAUDE.md:171-175]

## Standard Stack

### Core

| Library / Tool | Version | Purpose | Why Standard |
|---------------|---------|---------|--------------|
| Go | `1.27` line; local direct toolchain `go1.27.4 darwin/arm64` | Compiler, test runner, cross-builds | Already required by `go.mod` and pinned as the mise tool line. [VERIFIED: go.mod:3; mise.toml:6-9; local `go version` output] |
| `github.com/coder/acp-go-sdk` | `v0.20.0` | Existing ACP dependency | Keep the frozen dependency; Phase 11 adds no module. [VERIFIED: go.mod:5; go.sum:1-2] |
| mise | Host `2026.11.13`; repository config currently untrusted | Single source of tool versions and tasks | Existing D-02 convention; do not duplicate pins in workflow inputs. [VERIFIED: mise.toml:1-17; local mise error output] |
| GoReleaser | Repository pin `2.20.2`; only `2.17.1` installed locally | Multi-arch builds, universal binary, archives, checksums | Already pinned for release; add the missing config rather than hand-rolling packaging. [VERIFIED: mise.toml:14-17; local `goreleaser --version` output] |
| Bash | Recommended mise pin `5.3` [ASSUMED] | Run shell gates on macOS without stock 3.2 | `check-docs.sh` uses Bash 4 associative arrays; a managed Bash preserves the script. [VERIFIED: experiments/check-docs.sh:58 quotes `declare -A REQUIRED_SECTIONS`; local `/bin/bash --version` reports `3.2.57(1)-release`] |

### Supporting

| Library / Tool | Version | Purpose | When to Use |
|---------------|---------|---------|-------------|
| `golangci-lint` | `2.15.0` | Strict lint gate | Existing pin; local installed `2.12.2` is not the required version. [VERIFIED: mise.toml:10-13; local install inventory] |
| Node.js | `26` line; local `v26.2.0` | `node --test experiments/` | Existing experiment/static-gate suite. [VERIFIED: mise.toml:18-19,41-43; local `node --version` output] |
| Go `os/exec` + `syscall.SysProcAttr` | Go 1.27 | Re-exec detached test helpers | Replaces external `setsid`; `Setsid` is available in Darwin's syscall implementation. [VERIFIED: Go 1.27.4 `src/syscall/exec_libc2.go:5-22`] |
| Apple `lipo` | macOS platform tool | Verify universal architecture slices | Release runs on macOS; verify the fat binary instead of parsing Mach-O manually. [ASSUMED: exact runner tool availability must be checked at execution] |
| `shasum -a 256` | macOS platform tool | Verify GoReleaser checksum manifest | Portable release verification on the macOS snapshot runner. [ASSUMED: exact command availability must be checked at execution] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Managed Bash 5.x | Rewrite `check-docs.sh` to POSIX/Bash 3.2 | Removes a tool dependency but churns a working provenance gate; managed Bash is smaller and matches D-02. |
| Go re-exec helpers | Install GNU `setsid` via Homebrew/mise | Adds a host dependency and violates MTEST-01's intent. |
| GoReleaser universal binary | Manual `lipo -create` script | GoReleaser already coordinates builds, archives, checksums, and release metadata. |
| Native macOS test jobs | Cross-compile tests only | Cross-compiled Darwin tests cannot execute on Linux and would not satisfy MTEST-01. |
| Universal-only archive | Separate thin arm64/amd64 downloads | Universal-only is simpler; retain thin binaries as verified intermediate inputs. Confirm if separate downloads are required. |

**Installation:**

No new Go module dependency is needed. Phase 11 should add only the managed Bash tool and the missing GoReleaser configuration. Exact tool installation must use the repository's mise pins after the user trusts the target config; do not run `mise trust` automatically. [VERIFIED: mise.toml:1-17; local mise refusal output]

**Version verification:** Repository pins were read directly from `mise.toml` and `go.mod`. Network registry checks were intentionally not run under the static-only constraint. The target GoReleaser `2.20.2` is not installed locally, so its schema compatibility is an execution checkpoint. [VERIFIED: mise.toml:9,13,17; local install inventory]

## Package Legitimacy Audit

Phase 11 installs no new language package. `acp-go-sdk v0.20.0` is an existing frozen module, and Bash/GoReleaser are mise-managed tools rather than npm/PyPI/crates dependencies. The package-legitimacy seam supports only `npm`, `pypi`, and `crates`, so it is not applicable to this Go/tool-only phase. [VERIFIED: go.mod:3-5; mise.toml:6-19]

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/coder/acp-go-sdk` | Go module proxy | Not checked | Not checked | Not checked this session | N/A | Existing pin; no change |
| GoReleaser | mise tool | Not checked | Not checked | Not checked this session | N/A | Existing pin; install exact `2.20.2` before release validation |
| Bash | mise tool | Not checked | Not checked | Not checked this session | N/A | Proposed tool pin `5.3` [ASSUMED]; human-verify before adding |

**Packages removed due to SLOP verdict:** none
**Packages flagged as suspicious (SUS):** none

## Architecture Patterns

### System Architecture Diagram

```text
Developer / CI trigger
        |
        v
mise-pinned toolchain + tasks
        |
        +--> build / vet / lint / tidy-diff
        |
        +--> native go test -race
        |        |
        |        +--> re-exec current test binary
        |        |        |
        |        |        +--> SysProcAttr.Setsid helper
        |        |        +--> bounded sleep / cleanup / exit
        |        |
        |        +--> mount + ACP + fake-backend suites
        |
        +--> CGO_ENABLED=0 Darwin cross-builds (arm64, amd64)
        |
        +--> shell gates via managed Bash
        |
        v
macOS GoReleaser snapshot
        |
        +--> thin darwin/arm64 binary
        +--> thin darwin/amd64 binary
        |
        v
universal_binaries (lipo)
        |
        v
darwin_universal archive + checksums.txt
        |
        v
shasum verification + lipo -archs verification
```

### Recommended Project Structure

```text
.
├── mise.toml                         # add managed Bash; keep single-source pins
├── .goreleaser.yaml                  # new: two Darwin builds + universal archive
├── .github/workflows/pr-sanity.yml   # native OS/arch jobs + cross-build job
├── .github/workflows/release.yml     # macOS snapshot + tag publish
├── internal/supervisor/mount.go      # int64(st.Type) conversion
├── internal/supervisor/mount_test.go # Go re-exec detached cleaner
├── cmd/zcode-acp/main_test.go        # Go re-exec leaked descendant
└── experiments/portability.test.mjs  # static guards: no fixture setsid, managed Bash
```

### Pattern 1: Normalize the syscall width at the existing seam

**What:** Keep `MountOpts.Statfs func(path string) (int64, error)` unchanged and convert the platform field explicitly.
**When to use:** Always in `defaultStatfs`; no build tags or test-seam churn are needed.
**Example:**

```go
// Source: internal/supervisor/mount.go:190-202
func defaultStatfs(path string) (int64, error) {
	var st syscall.Statfs_t

	if err := syscall.Statfs(path, &st); err != nil {
		return 0, fmt.Errorf("zcode: statfs %s: %w", path, err)
	}

	return int64(st.Type), nil
}
```

The quoted source values are `func defaultStatfs(path string) (int64, error)`, `var st syscall.Statfs_t`, and `return st.Type, nil`; Darwin's field is verbatim `Type uint32`, while Linux's is verbatim `Type int64`. [VERIFIED: internal/supervisor/mount.go:190-202; Go 1.27.4 `src/syscall/ztypes_darwin_arm64.go:85-102`; Go 1.27.4 `src/syscall/ztypes_linux_amd64.go:117-130`]

### Pattern 2: Re-exec the test binary for detached fixture processes

**What:** A parent test starts `os.Args[0]` with a unique `-test.run` selector and an environment marker. The selected helper test creates a new session with `SysProcAttr.Setsid`, performs the delayed cleanup or records its PID and sleeps, then exits.
**When to use:** For both former `setsid` fixtures. Use `Setpgid` only for ordinary children that should remain killable as a group.
**Example skeleton [ASSUMED: helper names and marker are proposed, not existing values]:**

```go
// Proposed helper contract; names must be finalized in the plan.
cmd := exec.Command(os.Args[0], "-test.run=^TestDetachedFixtureHelper$")
cmd.Env = append(os.Environ(), "ZCODE_ACP_TEST_DETACHED_HELPER=1", "FIXTURE_DIR="+dir)
cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
```

The helper must exit only after its intended effect, and the parent must retain a bounded cleanup path for the recorded PID. The repository already has a re-exec precedent in `TestSpawnEnvHelper`, selected by `helperRunFlag`. [VERIFIED: internal/supervisor/spawn_test.go:143-168]

### Pattern 3: Separate compile portability from race-test execution

**What:** Use `CGO_ENABLED=0` for Darwin cross-builds and release binaries. Run `go test -race` natively with CGO enabled on Linux, macOS arm64, and macOS amd64 runners.
**When to use:** Every build/test gate. Never try to execute a cross-compiled Darwin test on Linux.
**Example:**

```bash
# Source: .planning/REQUIREMENTS.md:12
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...
```

The existing default test task is verbatim `go test -race -count=1 ./...`. [VERIFIED: mise.toml:37-39]

### Pattern 4: Preserve the existing required check while adding platform jobs

**What:** Keep the Linux `sanity` job name exactly `build / vet / lint / test / tidy`. Add separate `macos-arm64` and `macos-amd64` jobs plus a `darwin-cross-build` job.
**When to use:** When extending `pr-sanity.yml`. Do not turn the existing job into a matrix with a suffixed name without updating branch protection in the same change.
**Example:** Existing name and runner are verbatim `name: build / vet / lint / test / tidy` and `runs-on: ubuntu-latest`. [VERIFIED: .github/workflows/pr-sanity.yml:48-54]

### Pattern 5: Build the universal artifact on macOS

**What:** Configure two CGO-free Darwin builds, combine their IDs through GoReleaser's top-level `universal_binaries`, archive only the universal result, and emit SHA-256 checksums.
**When to use:** Both tag releases and dispatch-only snapshots.
**Example skeleton [ASSUMED: target GoReleaser 2.20.2 compatibility and chosen artifact IDs require execution verification]:**

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
    replace: true

archives:
  - id: darwin-universal
    builds: [darwin-universal]
    formats: [tar.gz]
    name_template: "zcode-acp_{{ .Version }}_darwin_universal"

checksum:
  name_template: checksums.txt
  algorithm: sha256
```

The installed GoReleaser `2.17.1` schema confirms top-level `universal_binaries`, universal fields `id`, `ids`, `name_template`, and `replace`, archive fields `formats`, `name_template`, and `builds`, and checksum fields `name_template` and `algorithm`. [VERIFIED: local GoReleaser 2.17.1 `jsonschema` output] The repository target is `2.20.2`, so validate with `goreleaser check` after installing that exact version. [VERIFIED: mise.toml:17]

The binary path `./cmd/zcode-acp` corresponds to the package already built by tests as `github.com/Djarvur/zcode-acp/cmd/zcode-acp`; the version seam is verbatim `var version = "18.1.9+zacp.0"`. [VERIFIED: cmd/zcode-acp/main_test.go:123-126; cmd/zcode-acp/main.go:37-45]

### Anti-Patterns to Avoid

- **Casting through `int`:** Use `int64(st.Type)` directly; an intermediate platform-sized `int` adds no value and obscures the seam.
- **Build-tagging the whole mount implementation:** Only the field width differs; keep one mount lifecycle and one injected `Statfs` seam.
- **Installing `setsid`:** It preserves the test but violates the requirement; use the Go re-exec helper.
- **Setting `Setsid` and `Setpgid` together:** Choose session creation for detached helpers and process-group creation for ordinary children.
- **Running `-race` with `CGO_ENABLED=0`:** Cross-build settings and native race-test settings are separate gates. [ASSUMED: Go race-detector platform requirements must be rechecked against the pinned toolchain at execution]
- **Using `macos-latest` for both architectures:** The label alone does not prove amd64 execution; use an explicit Intel runner or an architecture preflight. [ASSUMED: current GitHub runner label mapping]
- **Building a universal binary on Ubuntu:** The release job must run where the required Apple tooling exists. [ASSUMED: GoReleaser universal-binary host requirement]
- **Renaming the Linux required check:** Branch protection currently depends on the exact existing context. [VERIFIED: .github/workflows/pr-sanity.yml:49-52]
- **Asserting the Orca version floor on a snapshot:** Snapshot versions may be below the stamped compatibility floor; verify artifact structure and checksums instead. [ASSUMED: GoReleaser snapshot version format]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detached test process | Shell `setsid` wrapper or Homebrew dependency | Go test-binary re-exec + `syscall.SysProcAttr.Setsid` | Portable across Linux/macOS and keeps process setup in the test. |
| Filesystem-type normalization | Per-OS mount implementations | Existing `MountOpts.Statfs` seam + explicit conversion | One lifecycle, one test seam, minimal change. |
| Universal binary | Custom `lipo -create` orchestration | GoReleaser `universal_binaries` | Coordinates build IDs, replacement, archives, and checksums. |
| Checksum generation | Custom SHA-256 shell loop | GoReleaser `checksum` + `shasum -a 256 -c` | Standard manifest format and independent verification. |
| Tool pinning | Version inputs duplicated in workflows | `mise.toml` + `mise install` + `mise run` | Existing D-02 single-source convention. |
| Mach-O architecture parsing | Custom binary parser | `lipo -archs` / `file` | Platform tools handle universal Mach-O correctly. |

**Key insight:** The hard parts here are process semantics and release orchestration, not new product behavior. Reusing Go's process API, the existing mount seam, mise, and GoReleaser avoids a second platform-specific implementation.

## Common Pitfalls

### Pitfall 1: Darwin `Statfs_t.Type` does not match the seam

**What goes wrong:** Both Darwin cross-builds fail at `return st.Type, nil`.
**Why it happens:** The field is `uint32` on Darwin and `int64` on Linux, while the seam returns `int64`.
**How to avoid:** Convert explicitly at the return site and keep the seam unchanged.
**Warning signs:** A Darwin compile error naming `st.Type`, `int64`, and `defaultStatfs`. [VERIFIED: internal/supervisor/mount.go:175,194-201; Go 1.27.4 Darwin and Linux type files]

### Pitfall 2: A Go re-exec helper accidentally runs the whole suite

**What goes wrong:** The child hangs or recursively launches tests.
**Why it happens:** The helper selector or environment gate is missing or mismatched.
**How to avoid:** Use a unique `-test.run=^Test...Helper$` selector and an early environment-marker return in the helper test.
**Warning signs:** Child output contains normal suite tests, repeated helper spawns, or timeout. [ASSUMED: proposed helper design]

### Pitfall 3: Detached helper cleanup is not bounded

**What goes wrong:** A failed test leaves a sleeping helper behind.
**Why it happens:** New sessions escape the wrapper's process-group kill.
**How to avoid:** Record the helper PID, use short fixture delays, and register unconditional PID cleanup in the parent.
**Warning signs:** `kill(pid, 0)` still succeeds after the test, or CI workers accumulate sleep processes. [VERIFIED: cmd/zcode-acp/main_test.go:956-966 already has recorded-PID cleanup for the old detached descendant]

### Pitfall 4: Stock Bash cannot parse the docs gate

**What goes wrong:** `declare -A` fails on macOS Bash 3.2.
**Why it happens:** The script uses associative arrays, while `/bin/bash` is 3.2.57 on this host.
**How to avoid:** Add a managed Bash pin and invoke shell gates through `mise exec -- bash ...` or a mise task whose PATH contains that Bash.
**Warning signs:** `declare: -A: invalid option` or an unsupported `read -ra` path. [VERIFIED: experiments/check-docs.sh:58,127; local `/bin/bash --version`]

### Pitfall 5: The Linux VM script is mistaken for a macOS default gate

**What goes wrong:** macOS validation fails on GNU `readlink -f` or `timeout`.
**Why it happens:** `scripts/vm-e2e.sh` is a Linux VM acceptance script, not part of `mise run ci`.
**How to avoid:** Keep it explicitly Linux-only and out of Phase 11's macOS default gate; do not add it to the cross-platform task graph.
**Warning signs:** A plan adds `vm-e2e.sh` to `tasks.ci` or the macOS matrix. [VERIFIED: scripts/vm-e2e.sh:91-95 quotes `set -euo pipefail` and `SELF=$(readlink -f "$0")`; scripts/vm-e2e.sh:604 uses `timeout "$TIMEOUT_ARM"`; mise.toml:53-55 does not include it]

### Pitfall 6: Race tests and static release builds share the wrong CGO setting

**What goes wrong:** Either the race build fails or the release binary acquires a C runtime dependency.
**Why it happens:** `CGO_ENABLED=0` is correct for the required cross-builds but is also copied into native `-race` jobs.
**How to avoid:** Keep separate commands: CGO-free cross-build/release, CGO-enabled native race tests.
**Warning signs:** Race jobs fail at startup, or `go version -m`/link inspection shows an unexpected dynamic dependency. [ASSUMED: race-detector requirement]

### Pitfall 7: GitHub macOS runner labels drift

**What goes wrong:** The amd64 job actually executes arm64, or the label does not exist.
**Why it happens:** Runner catalogs change and `macos-latest` can move architectures.
**How to avoid:** Add an architecture preflight that compares `go env GOARCH` with the matrix's expected value; verify the Intel label before locking the workflow.
**Warning signs:** Both macOS jobs report `darwin/arm64`. [ASSUMED: runner label mapping]

### Pitfall 8: GoReleaser 2.20.2 is assumed from a 2.17.1 schema

**What goes wrong:** `goreleaser check` rejects a field or changed default.
**Why it happens:** Only 2.17.1 is installed locally; the repository pins 2.20.2.
**How to avoid:** Install exact 2.20.2 through mise after trust, run `goreleaser check`, then run the snapshot.
**Warning signs:** Schema validation errors or unexpected artifact names. [VERIFIED: mise.toml:17; local `goreleaser --version` reports `2.17.1`]

### Pitfall 9: Checksums exist but cover the wrong artifact

**What goes wrong:** The manifest verifies thin binaries but not the universal archive, or vice versa.
**Why it happens:** Archive `builds` does not reference the universal ID, or `replace` semantics are misunderstood.
**How to avoid:** Archive only `darwin-universal`, inspect the manifest, extract the archive, and verify the contained binary's slices.
**Warning signs:** `checksums.txt` lacks the universal archive or `lipo -archs` shows only one architecture. [ASSUMED: chosen GoReleaser configuration]

### Pitfall 10: First build needs an uncached module

**What goes wrong:** Build/test stops before compiling project code.
**Why it happens:** `acp-go-sdk v0.20.0` is pinned and present in `go.sum`, but absent from the local module cache.
**How to avoid:** Treat module acquisition as an explicit execution checkpoint: approved network fetch or an approved vendoring path. Never invent a replacement dependency.
**Warning signs:** `module lookup disabled by GOPROXY=off` or a download error for `github.com/coder/acp-go-sdk`. [VERIFIED: go.sum:1-2; local module-cache inspection found no `github.com/coder/acp-go-sdk@v0.20.0` directory]

### Pitfall 11: Branch protection rejects the new workflow shape

**What goes wrong:** The Linux required context disappears after converting the job to a matrix.
**Why it happens:** The exact string `build / vet / lint / test / tidy` is a protected check name.
**How to avoid:** Preserve that job name for Linux and add new macOS contexts; update protection only after observing the new check names.
**Warning signs:** A green PR is blocked by a missing required status. [VERIFIED: .github/workflows/pr-sanity.yml:49-52]

## Code Examples

### Exact Darwin build gates

```bash
# Source: .planning/REQUIREMENTS.md:12
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...
```

### Native test gates

```bash
# Existing task: mise.toml:37-39
go test -race -count=1 ./...

# Existing task: mise.toml:41-43
node --test experiments/
```

### Snapshot and artifact verification

```bash
# Proposed release-snapshot commands [ASSUMED: exact GoReleaser 2.20.2 CLI behavior]
mise exec -- goreleaser release --snapshot --clean --skip=publish
shasum -a 256 -c checksums.txt

# Extract into a fresh empty directory first; then verify the contained binary.
lipo -archs <extracted>/zcode-acp
file <extracted>/zcode-acp
```

The architecture assertion must contain both `arm64` and `x86_64`; the checksum command must run against the manifest produced by the same snapshot. [ASSUMED: `lipo` architecture spelling and archive layout]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Linux-only sanity job | Linux + native macOS arm64/amd64 jobs | Phase 11 target | Real execution coverage for MTEST-01 and MBLD-03. |
| Platform-width return from `Statfs_t.Type` | Explicit `int64` normalization | Phase 11 target | Both Darwin cross-builds compile without OS-specific mount lifecycles. |
| External `setsid` in fixtures | Go test-binary re-exec with `Setsid` | Phase 11 target | Default macOS suite needs no host utility. |
| Ubuntu GoReleaser job without config | macOS GoReleaser config + snapshot verification | Phase 11 target | Universal artifact and checksums become reproducible gates. |
| Stock Bash for shell scripts | mise-managed Bash | Phase 11 target | macOS shell gates no longer depend on `/bin/bash` 3.2. |

**Deprecated/outdated:**

- External `setsid` in the two fixture scripts: replace, do not preserve as an optional fallback.
- Untagged `return st.Type`: replace with the explicit conversion.
- Ubuntu-only release assumption: universal binary creation belongs on macOS.
- Treating the pinned GoReleaser version as installed: local `2.17.1` does not satisfy `2.20.2`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | mise can provide Bash `5.3` on macOS and Linux; exact patch resolves consistently. | Standard Stack; Pitfall 4 | Shell-gate install fails or results differ; fallback is rewriting `check-docs.sh` to Bash 3.2-compatible syntax. |
| A2 | `macos-latest` is arm64 and `macos-15-intel` is amd64 at execution time. | Standard Stack; Pitfall 7 | One architecture is not truly tested; add a `go env GOARCH` preflight and adjust labels. |
| A3 | GoReleaser `2.20.2` retains the `universal_binaries`, archive, and checksum schema observed in local `2.17.1`. | Pattern 5; Pitfall 8 | Config fails validation; install exact version and adapt before snapshot. |
| A4 | GoReleaser's universal-binary creation must run on macOS and can use `lipo` there. | Pattern 5; Pitfall 8 | Snapshot cannot create a fat binary on Ubuntu; move the job or install equivalent tooling. |
| A5 | `-X main.version={{ .Version }}` is the desired release stamp and snapshot versions need not meet Orca's floor. | Pattern 5; Pitfall 9 | Release advertises the wrong version; add a tag-release version assertion. |
| A6 | Proposed helper test name `TestDetachedFixtureHelper` and marker `ZCODE_ACP_TEST_DETACHED_HELPER` are acceptable. | Pattern 2 | Names collide or leak into other tests; finalize unique names in the plan. |
| A7 | One universal tar.gz plus `checksums.txt` satisfies “universal macOS binary artifacts”. | Alternatives; Pattern 5 | Stakeholder expects separate thin downloads; add them without removing the universal artifact. |
| A8 | Native `go test -race` requires CGO enabled on the selected platforms. | Pattern 3; Pitfall 6 | Race jobs fail or skip; verify with the pinned Go toolchain. |
| A9 | `shasum -a 256` and `lipo -archs` are available on the chosen macOS runner. | Code Examples | Verification step fails; use runner-provided alternatives after checking tool availability. |

## Open Questions

1. **Which exact macOS Intel runner label should the amd64 job use?**
   - What we know: GitHub's macOS labels have changed architecture defaults. [ASSUMED]
   - What's unclear: The current public label mapping for this repository's GitHub settings.
   - Recommendation: Use an explicit Intel label, then assert `go env GOARCH == amd64` before tests.

2. **Does MBLD-03 require separate thin downloads as well as one universal archive?**
   - What we know: The requirement says “universal macOS binary artifacts”; the success criteria mention arm64 and amd64 artifacts. [VERIFIED: .planning/REQUIREMENTS.md:16; .planning/ROADMAP.md:108]
   - What's unclear: Whether one fat archive is sufficient.
   - Recommendation: Ship the universal archive as the primary artifact; keep thin binaries as build intermediates unless separate downloads are explicitly requested.

3. **How should the uncached SDK be acquired?**
   - What we know: `v0.20.0` is pinned and missing locally. [VERIFIED: go.mod:5; go.sum:1-2; local cache inspection]
   - What's unclear: Whether network fetch or vendoring is approved.
   - Recommendation: Make module acquisition a human-approved execution checkpoint; do not change the dependency.

4. **Should shell portability be solved by managed Bash or by rewriting the docs gate?**
   - What we know: `check-docs.sh` uses `declare -A`; stock macOS Bash is 3.2. [VERIFIED: experiments/check-docs.sh:58; local Bash version]
   - What's unclear: Whether adding another managed tool is preferred.
   - Recommendation: Use mise-managed Bash for Phase 11 to minimize gate churn; revisit POSIX conversion separately.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|-------------|-----------|---------|----------|
| Direct Go 1.27 toolchain | Build/test/cross-build | Yes | `go1.27.4 darwin/arm64` | Use absolute toolchain path; do not use host default 1.26.5. |
| Host default Go | None | Yes, wrong version | `1.26.5` per confirmed audit | `GOTOOLCHAIN=local` plus direct Go 1.27.4 avoids auto-download. |
| `acp-go-sdk` module cache | First compile/test | No | `v0.20.0` absent | Approved network fetch or approved vendoring only. |
| mise binary | Tool/task orchestration | Yes | `2026.11.13 macos-arm64` | Direct commands; target `mise.toml` is untrusted and was not executed. |
| Managed Bash 5.x | Shell gates on macOS | No evidence | Stock `/bin/bash` `3.2.57` | Add exact mise pin or rewrite `check-docs.sh`. |
| GoReleaser 2.20.2 | Snapshot/release | No | Only `2.17.1` installed | Install exact pin after trust; do not validate release with 2.17.1 alone. |
| golangci-lint 2.15.0 | Lint gate | No evidence | Installed inventory shows `2.12.2` | Install exact pin after trust. |
| Node 26 | Experiment tests | Yes | `v26.2.0` | Direct Node binary path. |
| Apple `lipo` / `shasum` | Artifact verification | Not probed | — | Check on selected macOS runner before snapshot. |

**Missing dependencies with no fallback:**

- `acp-go-sdk v0.20.0` module contents: compile/test cannot run offline until fetched or vendored with approval.
- Exact GoReleaser `2.20.2`: the release snapshot cannot be certified against the pinned version until installed.
- Exact golangci-lint `2.15.0`: the pinned lint gate cannot be reproduced with the older local install.

**Missing dependencies with fallbacks:**

- Trusted mise activation: use direct Go/Node paths for static checks; restore mise tasks after user approval.
- Managed Bash: either add the pin or remove the Bash 4 dependency from `check-docs.sh`.
- Intel macOS runner: verify the label and add an architecture preflight.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` plus Node `node:test` |
| Config file | None; commands live in `mise.toml` and GitHub workflows |
| Quick run command | `go test ./internal/supervisor -run 'TestMountStop|TestMountHappyPath' -count=1` |
| Full suite command | `go test -race -count=1 ./... && node --test experiments/` |
| Cross-build command | The two exact commands from MBLD-01 |
| Release verification | `goreleaser release --snapshot --clean --skip=publish`, then checksum and `lipo -archs` checks |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| MBLD-01 | Darwin arm64 and amd64 compile with CGO disabled | build gate | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` and the amd64 equivalent | ❌ Wave 0: CI cross-build job |
| MBLD-02 | No fixture invokes external `setsid`; shell gates use managed Bash | static unit | `node --test experiments/portability.test.mjs` | ❌ Wave 0 |
| MBLD-02 | `defaultStatfs` preserves the `int64` seam on both OS widths | unit | `go test ./internal/supervisor -run 'TestMount|TestStatfs' -count=1` | ✅ Existing mount suite; add a direct seam test if absent |
| MBLD-03 | Linux and both macOS architectures build/test | CI matrix | `mise run build`, `mise run vet`, `mise run lint`, `mise run test`, `mise run test-experiments`, `mise run tidy-diff` | ❌ Wave 0: macOS jobs |
| MBLD-03 | Snapshot emits a universal archive and valid SHA-256 manifest | release smoke | `goreleaser release --snapshot --clean --skip=publish` + `shasum -a 256 -c checksums.txt` + `lipo -archs` | ❌ Wave 0: config and snapshot job |
| MTEST-01 | Full default suite passes on macOS arm64 without `setsid` | native integration | `go test -race -count=1 ./...` on arm64 macOS | ✅ Suite exists; ❌ macOS runner gate |

### Sampling Rate

- **Per task commit:** Quick supervisor test plus the new portability static test; run the exact affected cross-build after the `Statfs` change.
- **Per wave merge:** Full native `-race` suite and Node experiment suite on each supported runner.
- **Phase gate:** Both Darwin cross-builds green, Linux/macOS arm64/macOS amd64 native gates green, shell gates green under managed Bash, and snapshot checksum/architecture verification green.

### Wave 0 Gaps

- [ ] Add `experiments/portability.test.mjs` — rejects executable `setsid` fixture calls and asserts shell gates use managed Bash.
- [ ] Add the Go re-exec helper test(s) and replace both scripted `setsid` call sites.
- [ ] Add a direct `defaultStatfs` seam test if the existing injected-Statfs tests do not exercise the production converter.
- [ ] Add native macOS arm64/amd64 CI jobs and the two-command Darwin cross-build job.
- [ ] Add managed Bash to `mise.toml` and route shell-gate workflow steps through it.
- [ ] Add `.goreleaser.yaml` and a macOS snapshot job with checksum and architecture verification.
- [ ] Install exact GoReleaser `2.20.2`, golangci-lint `2.15.0`, and approved module contents before claiming the phase gate green.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | No | No new authentication surface; CI remains credential-free. |
| V3 Session Management | No | No runtime session changes. |
| V4 Access Control | No | No new authorization surface; keep release write token scoped to the publish job. |
| V5 Input Validation | Yes, release artifacts | Treat downloaded/extracted artifacts as untrusted; extract into a fresh directory and verify before execution. |
| V6 Cryptography | Yes, integrity | Use standard SHA-256 via GoReleaser and `shasum`; never hand-roll hashing. |

### Known Threat Patterns for Build/Release

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Artifact substitution or corruption | Tampering | SHA-256 manifest generated with the artifact and independently verified. |
| Wrong-architecture binary shipped | Tampering / Information disclosure | Verify `lipo -archs` contains exactly the expected Darwin slices. |
| Supply-chain version drift | Tampering | Keep mise as the single version source; install exact release/lint pins; pin Actions versions. |
| Untrusted archive extraction | Tampering | Extract into a new empty directory; inspect paths; do not execute before verification. |
| Excess release token scope | Elevation of privilege | `contents: read` for PR/snapshot jobs; `contents: write` only for tag publish. |
| Accidental live desktop/credential use | Information disclosure | Keep Phase 11 fixtures fake and offline; no real ZCode, Keychain, or account calls. |

## Sources

### Primary (HIGH confidence)

- `.planning/REQUIREMENTS.md:10-32,65-82` — exact Phase 11 requirements and phase ownership.
- `.planning/ROADMAP.md:100-109,158` — goal, success criteria, zero-spend boundary.
- `go.mod:1-5` and `go.sum:1-2` — Go line and frozen SDK pin.
- `mise.toml:1-55` — tool pins, task definitions, canonical gate order.
- `internal/supervisor/mount.go:158-202` — `Statfs` seam and failing return.
- Go 1.27.4 `src/syscall/ztypes_darwin_arm64.go:85-102` — verbatim `Type uint32`.
- Go 1.27.4 `src/syscall/ztypes_linux_amd64.go:117-130` — verbatim `Type int64`.
- Go 1.27.4 `src/syscall/exec_libc2.go:5-22` — Darwin `Setsid` and `Setpgid` fields.
- `internal/supervisor/mount_test.go:195-206` and `cmd/zcode-acp/main_test.go:704-717` — the two external `setsid` fixtures.
- `.github/workflows/pr-sanity.yml:47-92` and `.github/workflows/release.yml:39-69` — current Ubuntu-only gates and inert release flow.
- `experiments/check-docs.sh:37-64,123-131` — Bash 4 associative-array dependency.
- `scripts/vm-e2e.sh:91-95,598-609` — Linux-only GNU utility usage.
- Local GoReleaser `2.17.1` `jsonschema` output — universal binary, archive, and checksum field names for that installed version only.

### Secondary (MEDIUM confidence)

- Direct local version probes: Go `1.27.4 darwin/arm64`, Node `v26.2.0`, Bash `3.2.57(1)-release`, GoReleaser `2.17.1`.
- GSD research-plan seam selected Context7 for Go/GoReleaser questions and WebSearch for shell portability. No fetch was executed because the task required static-only research.

### Tertiary (LOW confidence)

- Proposed Bash `5.3` pin, macOS runner labels, GoReleaser `2.20.2` schema continuity, release stamping behavior, and artifact naming are marked `[ASSUMED]` and require execution confirmation.

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH — repository pins and local tool versions were read directly; target GoReleaser installation remains a checkpoint.
- Architecture: HIGH — compile defect, process API, fixture sites, and current workflows are source-verified.
- Pitfalls: MEDIUM — repository facts are verified, but runner labels and GoReleaser 2.20.2 behavior were not executed under the static-only constraint.

**Research date:** 2026-11-25
**Valid until:** 2026-12-25 for stable repository facts; 7 days for runner labels and GoReleaser behavior
