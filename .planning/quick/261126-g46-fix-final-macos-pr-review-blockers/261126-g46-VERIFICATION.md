---
phase: 261126-g46
verified: 2026-11-26T09:17:20Z
status: passed
score: "6/6 must-haves verified"
covered_files:
  - .planning/quick/261126-g46-fix-final-macos-pr-review-blockers/261126-g46-PLAN.md
  - .planning/quick/261126-g46-fix-final-macos-pr-review-blockers/261126-g46-SUMMARY.md
  - cmd/zcode-acp/macapp_test.go
  - cmd/zcode-acp/main.go
  - cmd/zcode-acp/text_test.go
  - internal/supervisor/bundle.go
  - internal/supervisor/bundle_internal_test.go
  - internal/supervisor/discover.go
  - internal/supervisor/discover_test.go
  - internal/supervisor/mount.go
  - internal/supervisor/mount_test.go
  - internal/supervisor/path.go
  - internal/supervisor/path_internal_test.go
covered_digest: "v3:sha256:1c6d142a97227448758cf48e3138353ecc3377f8f52b64b4252d14be1c76e274"
behavior_unverified: 0
overrides_applied: 0
---

# Quick Task 261126-g46: Final macOS PR Review Blockers Verification

**Task Goal:** Fix final macOS PR review blockers

**Verified:** 2026-11-26T09:17:20Z

**Status:** passed

**Committed Range:** `bad352f95f9a68f3d3a36638aa4799b7d8fa4b9c..6b811935e7256bb3896863df6607084a7103828f`

**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Only a mac-app winner prefers the provider cache matching its app version; every Linux winner still selects the numerically newest usable cache. | VERIFIED | `internal/supervisor/discover.go:455-458` applies `preferVersion` only when the winner is `layoutMacApp`. `TestLinuxInstallKeepsNewestProviderCache` passed as a real Linux test binary in a network-disabled Debian container; `TestMacAppProviderCache` passed on Darwin. |
| 2 | A macOS override propagates its caller context into binary-plist conversion, and conversion has its own finite timeout. | VERIFIED | `cmd/zcode-acp/main.go:1373` captures the discovery context in `OverrideInstallContext`. `internal/supervisor/discover.go:323-339` keeps a bounded compatibility wrapper. `internal/supervisor/bundle.go:263-278` wraps conversion in `context.WithTimeout`, uses absolute `/usr/bin/plutil`, and joins conversion/cancellation errors. `TestReadBundleInfoBinaryContext` passed and asserted `errors.Is(err, context.Canceled)` with a fast return. |
| 3 | The login-shell PATH probe returns within a bounded interval even when a same-group descendant inherits stdout, and the probe group is cleaned up. | VERIFIED | `internal/supervisor/path.go:168-250` creates a private process tree, installs group-kill cancellation, sets `WaitDelay`, and resumes Windows jobs after assignment. `internal/supervisor/process_unix.go:27-29` sets a private group. `TestLoginShellRunnerDescendantStdout` passed on Darwin and Linux and asserted ESRCH for both the helper group and descendant. |
| 4 | Mount teardown kills and observes the helper process group even when the direct helper exits successfully on SIGTERM. | VERIFIED | `internal/supervisor/mount.go:586-658` removes successful early returns, always calls `terminateTree`, polls the group until ESRCH, and only then observes direct-helper reap. `TestMountStopReapsDescendantsAfterHelperExit` passed in the Linux container and asserted helper/descendant ESRCH plus root/base cleanup. |
| 5 | The fake macOS text E2E uses a hermetic login-shell shim so host/profile PATH cannot displace the temporary fake node; a separate subtest proves the explicit Electron fallback. | VERIFIED | `cmd/zcode-acp/text_test.go:288-300,439-457` supports explicit fixture-only PATH, Node-disable, and extra environment controls without unconditionally disabling Node. `cmd/zcode-acp/macapp_test.go:98-216` proves the shell shim ran on Darwin, checks exact Node argv, requires Electron absence in the default subtest, and independently proves Electron selection with Node absent in the fallback subtest. Both subtests passed with exact stdout, one `workspace/generateText`, and zero ACP frames. |
| 6 | Roadmap, requirements, state, and Phase 16 evidence report Phase 16 blocked pending live z.ai tool-call and permission proof; MORCA-02 remains incomplete. | VERIFIED | `.planning/REQUIREMENTS.md:35,81` leaves MORCA-02 unchecked and Blocked. `.planning/ROADMAP.md:81,189-203` reports Phase 16 unchecked, blocked, and 0/1. `.planning/STATE.md` reports blocked status, 5/6 phases, 9/10 plans, and 90%. `.planning/phases/16-live-orca-acceptance/16-UAT.md` reports zero passes and one blocked test. `.planning/phases/16-live-orca-acceptance/16-LIVE-EVIDENCE.md` explicitly says the custom DeepSeek result does not prove tool-calls or permission requests. |

**Score:** 6/6 truths verified; 0 present-but-behavior-unverified.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/supervisor/discover.go` | Layout-scoped exact-version preference and context-aware override entry point | VERIFIED | Substantive implementation; called by production discovery and covered by Linux/macOS cache tests. |
| `internal/supervisor/bundle.go` | Bounded plutil conversion with propagated cancellation | VERIFIED | Named five-second timeout, caller context, absolute converter path, joined errors; cancellation regression passes. |
| `internal/supervisor/path.go` | Process-group-owned, WaitDelay-bounded login-shell probe | VERIFIED | Private process tree and cleanup wired through platform helpers; descendant fixture passes on Darwin/Linux. |
| `internal/supervisor/mount.go` | Unconditional helper-group sweep and bounded group-gone wait | VERIFIED | No early success return; group ESRCH precedes direct-helper reap observation; Linux fixture passes. |
| `internal/supervisor/discover_test.go` | Linux newest-cache regression test | VERIFIED | Selects 4.0.0 cache over older 3.18.2 install; passed in Linux container. |
| `internal/supervisor/path_internal_test.go` | Descendant-holding-stdout probe fixture | VERIFIED | Re-exec fixture records PIDs, registers unconditional cleanup, and asserts bounded return plus ESRCH. |
| `internal/supervisor/mount_test.go` | Linux descendant-after-helper-exit fixture | VERIFIED | Redirected-stdio descendant survives direct helper exit until group sweep; passed in Linux container. |
| `cmd/zcode-acp/macapp_test.go` | Hermetic Node-default and explicit Electron-fallback E2E subtests | VERIFIED | Both launcher paths pass with independent marker, stdout, frame-count, and no-ACP assertions. |
| `.planning/phases/16-live-orca-acceptance/16-UAT.md` | Blocked live z.ai acceptance gate and concrete follow-up | VERIFIED | `status: blocked`, `passed: 0`, `blocked: 1`, and a specific intended-account human flow. |

The automated artifact query reported 9/9 present and substantive. Manual review confirmed implementation and wiring rather than accepting file existence alone.

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/supervisor/discover.go` | `internal/supervisor/bundle.go` | `OverrideInstallContext` passes its context to `macAppInstall` and `readBundleInfo` | WIRED | Context flows through the override mapper into plist conversion. |
| `cmd/zcode-acp/main.go` | `internal/supervisor/discover.go` | `discoverInstall` captures its context in the override closure | WIRED | Production closure calls `OverrideInstallContext(ctx, appServer)`. |
| `internal/supervisor/path.go` | `internal/supervisor/process_unix.go` / `process_windows.go` | `configureCommand`, process-tree assignment, cancellation, and resume | WIRED | Unix receives `Setpgid`; Windows receives job assignment and post-assignment resume. |
| `internal/supervisor/mount.go` | `internal/supervisor/spawn.go` | `stopHelper` always calls `terminateTree` and waits for group ESRCH | WIRED | The production `Child` process tree is swept before Stop can return. |
| `cmd/zcode-acp/macapp_test.go` | `cmd/zcode-acp/text_test.go` | `textRide` supplies hermetic PATH and launcher controls | WIRED | Built-binary runner receives fixture-only PATH, optional Node disable, and hermetic SHELL. |

The automated key-link query reported 5/5 verified.

### Data-Flow Trace (Level 4)

Not applicable. This task changes supervisor utilities, process lifecycle code, tests, and status documents; it does not render dynamic data from a database, API, or store.

### Behavioral Spot-Checks

| Behavior | Command / Execution | Result | Status |
|----------|---------------------|--------|--------|
| Linux newest-cache and macOS exact-version selection | Focused `go test ./internal/supervisor -run ...` on Darwin; Linux test binary in Debian container | Both platform-specific tests passed | PASS |
| Canceled binary-plist context | `go test ./internal/supervisor -run '^TestReadBundleInfoBinaryContext$' -count=1 -v` | Passed in 0.01s with cancellation assertion | PASS |
| Descendant-holding-stdout PATH probe | Named test on Darwin and Linux | Passed; group and descendant observed gone | PASS |
| Mount descendant cleanup after helper exit | Linux test binary: `TestMountStopReapsDescendantsAfterHelperExit` | Passed; helper/descendant ESRCH and root/base cleanup | PASS |
| Node-default and Electron-fallback E2E | `go test ./cmd/zcode-acp -run '^TestOmpTextMacAppEndToEnd$' -count=1 -v` | Both subtests passed; exact stdout and frame assertions | PASS |
| Canonical CI | `mise run ci` | Passed; race-tested all Go packages, lint 0 issues, 47 experiments passed, one guarded skip | PASS |
| Module tidy drift | `mise run tidy-diff` | Passed; no `go.mod`/`go.sum` drift | PASS |
| macOS cross-builds | CGO-free Darwin arm64 and amd64 `go build ./...` | Both passed | PASS |
| Windows cross-platform compilation | CGO-free Windows amd64 and arm64 `go build ./...`; both architectures `go vet ./...` | All passed | PASS |
| Documentation status | Targeted grep assertions and `mise run check-docs` | Passed | PASS |
| Whitespace/range integrity | `git diff --check bad352f95f9a68f3d3a36638aa4799b7d8fa4b9c..HEAD` | Passed | PASS |

### Probe Execution

Not applicable. The task declares no probe scripts or probe-based acceptance mechanism.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|----------|-------------|-------------|--------|----------|
| MDISC-02 | Task 1 | macOS override forms and existing Linux layouts remain correct | SATISFIED | Context-aware override wiring and existing override tests remain green. |
| MDISC-04 | Task 1 | Correct runtime-provider cache selection | SATISFIED | Linux newest-cache and macOS exact-version tests pass. |
| MDISC-05 | Task 2 | Bounded, non-interactive login-shell PATH union | SATISFIED | Descendant/stdout fixture proves bounded return and group cleanup; existing union/scrub tests pass. |
| MTEST-02 | Task 3 | Zero-spend automated macOS fixtures | SATISFIED | Hermetic built-binary E2E uses only local fixtures and temp HOME/PATH. |
| MORCA-01 | Task 3 | Fake macOS text backend completes with correct stdout/exit | SATISFIED | Both launcher subtests pass exact stdout, exit 0, one generateText, zero ACP frames. |
| MORCA-02 | Task 3 | Live intended-account Orca acceptance | BLOCKED — intentional external gate | Requirement remains unchecked; UAT/evidence/state/roadmap consistently report blocked pending quota and human tool-call/permission proof. This is not a failed automated fix. |

No requirement IDs mapped to this task were orphaned.

### Test Quality Audit

| Test Area | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|--------|---------|----------|-----------------|---------|
| Cache selection regressions | Yes | 0 | No | Behavioral/value | PASS |
| Plist cancellation regression | Yes | 0 | No | Behavioral/error identity | PASS |
| PATH descendant fixture | Yes | 0 | No | Behavioral/lifecycle | PASS |
| Mount descendant fixture | Yes | 0 | No | Behavioral/lifecycle | PASS |
| Launcher E2E subtests | Yes | 0 | No | Behavioral/marker/value | PASS |
| Documentation assertions | Yes | 0 | No | Value/status | PASS |

No disabled requirement-linked tests or circular expected-value generation were found.

### Decision Coverage

Not evaluated: no task `CONTEXT.md` with a decisions block exists.

### Anti-Patterns Found

None. No unreferenced `TBD`, `FIXME`, or `XXX` debt markers, placeholder implementations, disabled requirement tests, or false-green launcher assertions were found in the changed files.

### Human Verification Required

N/A for this quick task. All acceptance criteria are programmatically verifiable and passed.

The separate Phase 16 live acceptance gate remains intentionally blocked: after z.ai quota is available, a human must run the current Mac Orca card against the intended account through initialize, session/new, a tool-call, an explicit permission decision, and a streamed final reply, then record scrubbed evidence. MORCA-02 must remain unchecked until that passes.

### Gaps Summary

No gaps blocking this quick task's goal. All six automated/documentation truths are verified. MORCA-02 remains a deliberately tracked external acceptance blocker, not an incomplete automated fix.

---

_Verified: 2026-11-26T09:17:20Z_

_Verifier: gsd-verifier_
