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
  tokens: 2641
  tasks: 3
  commits: 3
plan_head_before: 3279f44056e47c175e72ae391bc4eaa560f673cf
plan_head_after: c9586f68d1c6c9a4184babc07b7aae9fbe263614
tech-stack:
  added: []
  patterns:
    - Re-exec the current Go test binary with anchored selectors and dedicated environment markers
    - Use Setsid only for detached fixture descendants; retain Setpgid for ordinary wrappers
key-files:
  created:
    - experiments/portability.test.mjs
    - .planning/tdd/11-02-task1-red.json
  modified:
    - internal/supervisor/mount_test.go
    - cmd/zcode-acp/main_test.go
key-decisions:
  - Use os.Executable for the fake runtime's absolute test-binary path
  - Keep the readiness launcher in the wrapper group and detach only the sleeping inner helper
  - Reap the mount cleaner by recorded PID because its new session escapes process-group cleanup
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
duration: 8min
completed: 2026-11-25
status: complete
---

# Phase 11 Plan 02: Portable Detached Process Fixtures Summary

**Go test-binary re-exec helpers replace both host `setsid` fixtures while preserving detached mount cleanup and inherited-stderr teardown semantics.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-11-25T12:45:13Z
- **Completed:** 2026-11-25T12:52:59Z
- **Tasks:** 3/3
- **Files modified:** 4

## Accomplishments

- Added an offline static contract that rejects executable `setsid` lines and requires anchored Go helper selectors with `SysProcAttr{Setsid: true}`.
- Replaced the stubborn mount fixture's shell cleaner with a marker-gated, 500 ms Go helper that removes only the validated mount root; the parent kills and reaps its recorded PID unconditionally.
- Replaced the readiness fixture's shell descendant with a group-owned launcher and a detached 30-second sleeping helper that records its PID and inherits the wrapper stderr pipe.
- Preserved every existing behavioral assertion: mount grace escalation, helper reap, mount/base removal, idempotent Stop, leaked stderr pipe, and teardown-error precedence over the signal.

## Task Commits

1. **Task 1: RED — lock both detached fixtures to Go re-exec helpers** — `6374c14` (`test`)
2. **Task 2: GREEN — move the stubborn mount cleaner into a Go helper** — `078bc3b` (`feat`)
3. **Task 3: GREEN — preserve the inherited-stderr leak with nested Go helpers** — `c9586f6` (`feat`)

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
- `cmd/zcode-acp/main_test.go` — absolute test-binary token substitution, nested leak launcher/helper, inherited stderr, bounded lifetime, and retained unconditional recorded-PID cleanup.

## Verification

| Command | Result |
|---------|--------|
| `node --test --test-reporter=tap experiments/portability.test.mjs` | PASS — 2/2 |
| `node --test experiments/` | PASS — 26 pass, 1 guarded skip, 0 fail |
| `go test -race -count=1 ./internal/supervisor -run '^TestMountStop$'` | PASS |
| `go test -race -count=1 ./cmd/zcode-acp -run '^TestAppImageMountSignal$'` | PASS |
| `go vet ./internal/supervisor ./cmd/zcode-acp` | PASS |
| `gsd_run check tdd-red-evidence .planning/tdd/11-02-task1-red.json --raw` | PASS — `RED_EVIDENCE_OK` |

No detached fixture helper remained after verification. The full native Linux/macOS matrix remains owned by Plan 04, as specified in this plan.

## Decisions Made

- Re-exec `os.Executable()` so the fake runtime receives an absolute path to the already-built test binary.
- Keep the readiness launcher in the fake helper's process group; only the inner sleeping helper calls `Setsid`.
- Use direct recorded-PID cleanup for detached sessions, since process-group termination cannot reach them.
- Keep helper bodies inert unless their dedicated environment marker equals `1`.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Authentication Gates

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Both default-suite host `setsid` dependencies are removed and statically guarded. The two process-behavior oracles pass under `-race` on this macOS arm64 host. Native Linux and both macOS CI architecture gates remain ready for Plan 04; shared `STATE.md` and `ROADMAP.md` updates are intentionally left to the orchestrator.

## Self-Check: PASSED

- All four implementation/evidence files and this summary exist on disk.
- All three task commits are ancestors of the current HEAD.
- The RED classifier still returns `RED_EVIDENCE_OK`.
- `STATE.md` and `ROADMAP.md` are unchanged in both the plan commit range and the working tree.
