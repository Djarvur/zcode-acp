---
phase: 11-darwin-build-test-release-gates
plan: 02
subsystem: testing
tags: [go, testing, darwin, process-management, tdd]
requires: []
provides:
  - Static portability gate for both detached-process fixtures
  - Go re-exec mount cleaner with bounded delay and direct PID cleanup
  - Nested Go readiness helpers preserving inherited wrapper stderr
affects: [11-04, 11-05, MTEST-01, MBLD-02]
actuals:
  tokens: 5230
  tasks: 3
  commits: 3
plan_head_before: 3279f44056e47c175e72ae391bc4eaa560f673cf
plan_head_after: c9586f68d1c6c9a4184babc07b7aae9fbe263614
tech-stack:
  added: []
  patterns:
    - Re-exec the current Go test binary with anchored selectors and dedicated environment markers
    - Use Setsid only for detached fixture descendants; retain Setpgid for ordinary wrappers
    - Canonicalize expected test paths with Abs/EvalSymlinks before comparing production-resolved paths
key-files:
  created:
    - experiments/portability.test.mjs
    - .planning/tdd/11-02-task1-red.json
  modified:
    - internal/supervisor/mount_test.go
    - internal/supervisor/discover_test.go
    - cmd/zcode-acp/main_test.go
    - internal/zcode/fake/golden_test.go
key-decisions:
  - Use os.Executable for the fake runtime's absolute test-binary path
  - Keep the readiness launcher in the wrapper group and detach only the sleeping inner helper
  - Reap the mount cleaner by recorded PID because its new session escapes process-group cleanup
  - Canonicalize only expected test paths; production path resolution remains unchanged
patterns-established:
  - Detached test fixture: marker-gated helper plus bounded lifetime plus unconditional PID cleanup
  - Static source contract: reject executable host setsid and require anchored Go helpers with Setsid
requirements-completed: [MTEST-01, MBLD-02]
coverage:
  - id: D1
    description: Static portability gate rejects host session-utility fixtures and requires Go re-exec helpers
    requirement: MBLD-02
    verification:
      - kind: unit
        ref: node --test --test-reporter=tap experiments/portability.test.mjs
        status: pass
    human_judgment: false
  - id: D2
    description: Stubborn mount cleaner runs as a bounded detached Go test-binary helper with PID cleanup
    requirement: MTEST-01
    verification:
      - kind: integration
        ref: go test -race -count=1 ./internal/supervisor -run '^TestMountStop$'
        status: pass
      - kind: unit
        ref: experiments/portability.test.mjs#mount fixture creates its detached cleaner through Go
        status: pass
    human_judgment: false
  - id: D3
    description: Readiness teardown leak uses nested Go helpers while preserving inherited stderr and signal precedence
    requirement: MTEST-01
    verification:
      - kind: integration
        ref: go test -race -count=1 ./cmd/zcode-acp -run '^TestAppImageMountSignal$'
        status: pass
      - kind: unit
        ref: experiments/portability.test.mjs#readiness fixture creates its detached leak through Go
        status: pass
    human_judgment: false
  - id: D4
    description: Discovery, AppImage, and golden expectations canonicalize macOS temporary-path aliases without production changes
    requirement: MTEST-01
    verification:
      - kind: integration
        ref: go test -race -count=1 ./...
        status: pass
      - kind: unit
        ref: node --test experiments/
        status: pass
    human_judgment: false
duration: 39min
completed: 2026-11-25
status: complete
---

# Phase 11 Plan 02: Portable Detached Process Fixtures Summary

**Go test-binary re-exec helpers replace both host `setsid` fixtures while preserving detached mount cleanup and inherited-stderr teardown semantics.**

## Performance

- **Duration:** 39 min
- **Started:** 2026-11-25T12:45:13Z
- **Completed:** 2026-11-25T13:23:53Z
- **Tasks:** 3/3
- **Files modified:** 7

## Accomplishments

- Added an offline static contract that rejects executable `setsid` lines and requires anchored Go helper selectors with `SysProcAttr{Setsid: true}`.
- Replaced the stubborn mount fixture's shell cleaner with a marker-gated, 500 ms Go helper that removes only the validated mount root; the parent kills and reaps its recorded PID unconditionally.
- Replaced the readiness fixture's shell descendant with a group-owned launcher and a detached 30-second sleeping helper that records its PID and inherits the wrapper stderr pipe.
- Preserved every existing behavioral assertion: mount grace escalation, helper reap, mount/base removal, idempotent Stop, leaked stderr pipe, and teardown-error precedence over the signal.
- Canonicalized expected discovery/AppImage paths and coding-plan golden probe paths for macOS `/var/folders` ↔ `/private/var/folders`, leaving production behavior unchanged.

## Task Commits

1. **Task 1: RED — lock both detached fixtures to Go re-exec helpers** — `6374c14` (`test`)
2. **Task 2: GREEN — move the stubborn mount cleaner into a Go helper** — `078bc3b` (`feat`)
3. **Task 3: GREEN — preserve the inherited-stderr leak with nested Go helpers** — `c9586f6` (`feat`)

**Post-wave deviation:** committed separately as `fix(11-02): canonicalize macOS temp paths in tests`.

## TDD Gate Compliance

| Gate | Status | Evidence |
|------|--------|----------|
| RED | PASS | `.planning/tdd/11-02-task1-red.json`; complete TAP report, exit 1, 0 pass / 2 fail; classifier `RED_EVIDENCE_OK` |
| GREEN | PASS | Mount source contract and `TestMountStop` passed after Task 2; both source contracts and `TestAppImageMountSignal` passed after Task 3 |
| REFACTOR | NOT NEEDED | No behavior-preserving cleanup commit was required |

**RED semantic assessment:** Both named tests executed and failed on the planned source assertion (`strictEqual(false, true)`) because each current shell fixture still contained an executable `setsid` line. The report contained no discovery, load, syntax, or fixture errors. The machine classifier matched the mount target and also reported the readiness target as failing.

## Files Created/Modified

- `experiments/portability.test.mjs` — static, offline portability contract for both detached fixtures.
- `.planning/tdd/11-02-task1-red.json` — complete unmodified TAP output, exit status, expected/actual result, and semantic RED assessment.
- `internal/supervisor/mount_test.go` — Go re-exec mount cleaner, bounded delay, `Setsid`, direct PID kill/reap cleanup, and retained Stop assertions.
- `cmd/zcode-acp/main_test.go` — absolute test-binary token substitution, nested leak launcher/helper, inherited stderr, bounded lifetime, retained unconditional recorded-PID cleanup, and canonical expected AppImage diagnostics.
- `internal/supervisor/discover_test.go` — canonical expected paths for discovery/AppImage assertions via `Abs`/`EvalSymlinks`.
- `internal/zcode/fake/golden_test.go` — portable coding-plan probe-path normalization across raw and canonical macOS temp roots.

## Verification

| Command | Result |
|---------|--------|
| `node --test --test-reporter=tap experiments/portability.test.mjs` | PASS — 2/2 |
| `node --test experiments/` | PASS — 26 pass, 1 guarded skip, 0 fail |
| `go test -race -count=1 ./internal/supervisor -run '^TestMountStop$'` | PASS |
| `go test -race -count=1 ./cmd/zcode-acp -run '^TestAppImageMountSignal$'` | PASS |
| `go test -race -count=1 ./internal/supervisor -run '^(TestDiscoverFixtureTree|TestDiscoverRankPackedOverStaleUnpacked|TestDiscoverPerCandidateMisses|TestDiscoverUnpackedOnly|TestDiscoverTieBreak|TestDiscoverPureRead|TestDiscoverOrderFirstMatchWins|TestDiscoverDiagnostics|TestDiscoverXDGAppImageSeed|TestDiscoverFindsNewestBuiltin|TestDiscoverBundledBuiltinFallback|TestOverrideInstall)$'` | PASS |
| `go test -race -count=1 ./cmd/zcode-acp -run '^TestAppImageOverride$'` | PASS |
| `go test -race -count=1 ./internal/zcode/fake -run '^TestGolden$/(coding-plan-probe-setmodel|coding-plan-probe-setmodel-refused)$'` | PASS |
| `go test -race -count=1 ./...` | PASS — 11 packages, 0 failed tests |
| `go vet ./internal/supervisor ./cmd/zcode-acp` | PASS |
| `gsd_run check tdd-red-evidence .planning/tdd/11-02-task1-red.json --raw` | PASS — `RED_EVIDENCE_OK` |

No detached fixture helper remained after verification. The full `go test -race -count=1 ./...` suite passes on this macOS arm64 host; the Linux/macOS CI matrix remains owned by Plan 04, as specified in this plan.

## Decisions Made

- Re-exec `os.Executable()` so the fake runtime receives an absolute path to the already-built test binary.
- Keep the readiness launcher in the fake helper's process group; only the inner sleeping helper calls `Setsid`.
- Use direct recorded-PID cleanup for detached sessions, since process-group termination cannot reach them.
- Keep helper bodies inert unless their dedicated environment marker equals `1`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Canonicalized expected discovery and AppImage paths on macOS**
- **Found during:** Post-wave full `go test -race ./...` run
- **Issue:** Production correctly returned `EvalSymlinks`-resolved `/private/var/folders/...`, while discovery/AppImage assertions compared against raw `t.TempDir()` `/var/folders/...`.
- **Fix:** Added test-only `Abs`/`EvalSymlinks` expectation helpers and applied them only where assertions compare production-resolved paths; display-path and missing-path expectations remain unchanged.
- **Files modified:** `internal/supervisor/discover_test.go`, `cmd/zcode-acp/main_test.go`
- **Verification:** Focused discovery/AppImage tests and the full race suite pass.

**2. [Rule 1 - Bug] Normalized raw and canonical macOS temp roots in coding-plan goldens**
- **Found during:** Full-suite follow-up after the discovery fix
- **Issue:** The golden normalizer replaced only the `zcode-acp-auth-probe-<suffix>` basename, leaving the platform temp root (`/var/folders/...` versus the recorded `/tmp`) in two coding-plan probe goldens.
- **Fix:** Match the complete probe path under both `os.TempDir()` and its `EvalSymlinks`-resolved root, then emit the existing deterministic `/tmp/zcode-acp-auth-probe-<probe-dir>` token.
- **Files modified:** `internal/zcode/fake/golden_test.go`
- **Verification:** Both focused probe goldens, the complete fake package, and the final captured full race suite pass.

**Total deviations:** 2 auto-fixed Rule 1 bugs.
**Impact on plan:** Both fixes are test-side portability corrections required by MTEST-01. No production behavior changed; no new dependencies, ACP methods, tools, or client features were added.

## Issues Encountered

The first full-suite run exposed the additional golden temp-root mismatch documented above. One intermediate full run failed without retained per-test output; isolated `internal/acpfront` and `internal/zcode/fake` runs passed, and the final JSON-captured full run completed with exit 0 and zero failed tests.

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Both default-suite host `setsid` dependencies are removed and statically guarded. The process-behavior oracles and the full `go test -race -count=1 ./...` suite pass on this macOS arm64 host. Native Linux and both macOS CI architecture gates remain ready for Plan 04; shared `STATE.md` and `ROADMAP.md` updates are intentionally left to the orchestrator.

## Self-Check: PASSED

- All seven implementation/evidence files and this summary exist on disk.
- All three original task commits are ancestors of the current HEAD.
- The RED classifier still returns `RED_EVIDENCE_OK`.
- The final captured `go test -race -count=1 ./...` run passed all 11 packages with zero failed tests.
- `STATE.md` and `ROADMAP.md` are unchanged in both the plan commit range and the working tree.
