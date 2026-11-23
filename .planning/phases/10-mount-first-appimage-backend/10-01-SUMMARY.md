---
phase: 10-mount-first-appimage-backend
plan: 01
subsystem: infra
tags: [appimage, fuse, mount, process-supervision, acp, go, tdd]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg
    provides: SpawnChild process-group/reap primitives, the spawn argv/env contract, install discovery and diagnostics
provides:
  - PrepareMount/Mount owner for a supervised --appimage-mount helper (private 0700 TMPDIR, readiness, tree validation, Lost watcher, staged Stop)
  - Install.AppImage/Version plus RequiresMount, and supervisor.OverrideInstall as the single override mapper (directory, binary, or packed .AppImage)
  - SpawnOpts.StdoutTap/StderrTap (helper-only leased streams) with capped diagnostics
  - mount-aware run: acquisition after discovery, ordered teardown after the supervisor's staged shutdown
affects: [10-02 failure handling, 10-03 discovery ranking, 10-04 override completeness, 10-05 real tier, 10-06 docs]

# Actuals (#2632) — same chars/4 scale as the plan's estimate.
actuals:
  tokens: 34650
  tasks: 2
  commits: 6

tech-stack:
  added: []
  patterns:
    - "Mount owner above the restart loop: the helper is a SpawnChild child but never owned by Supervisor.Start (no remount on app-server crashes)"
    - "Helper env by allowlist construction (PATH/HOME/TMPDIR) — APPIMAGE_EXTRACT_AND_RUN and the loader vars can never cross"
    - "Line-scoped mount-line trust: the whole absolute line must be a non-symlink .mount_* directory one component under the resolved private base"
    - "Leased stdout: SpawnOpts.StdoutTap hands the stream to exec's copier so Wait drains it to EOF (WaitDelay is only the close backstop)"
    - "Staged helper stop: SIGTERM, grace, group SIGKILL, reap, mount-gone poll, then remove only the private base"

key-files:
  created:
    - internal/supervisor/mount.go
    - internal/supervisor/mount_test.go
    - internal/supervisor/mount_internal_test.go
  modified:
    - internal/supervisor/discover.go
    - internal/supervisor/spawn.go
    - cmd/zcode-acp/main.go
    - cmd/zcode-acp/main_test.go

key-decisions:
  - "The helper env is an allowlist by construction; a deny list would be unsound because the runtime's APPIMAGE_EXTRACT_AND_RUN branch precedes the mount branch"
  - "The mount line is never field-split; the whole line is the path, and trust requires absolute + .mount_* + one component under the resolved private base + non-symlink directory"
  - "Teardown order is load-bearing: app-server staged shutdown first, helper second, mount-gone proven before the private base is removed"
  - "OverrideInstall always absolutizes a packed AppImage so exec can never PATH-resolve it; a dangling relative override is still stored absolute for the fail-loud message"
  - "A packed discovery winner defers the bundled-builtin probe; PrepareMount fills BuiltinFile from inside the mount only when discovery left it empty"

patterns-established:
  - "Fake AppImage runtime: a shell script answering only --appimage-mount creates a .mount_* tree under the wrapper TMPDIR, so the production exec/readiness/teardown path runs in CI without FUSE"
  - "Ordered-teardown proof: the fake helper records whether the app-server was alive when TERM arrived"
  - "Rule 1 fix: short stderr reads now forward buf[:n], not the whole buffer"

requirements-completed: [MNT-01, MNT-02, MNT-03, MNT-06, VER-01]

coverage:
  - id: D1
    description: "A packed .AppImage supplied through --app-server is mounted and the app-server serves initialize, two session/new calls, and both prompt loops from the mount"
    requirement: "MNT-01"
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountEndToEnd"
        status: pass
    human_judgment: false
  - id: D2
    description: "Ordered teardown with zero orphans: app-server gone before the helper TERM, both PIDs ESRCH, mount and private base gone, AppImage hash unchanged, no extraction copy"
    requirement: "MNT-03, MNT-06"
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountEndToEnd (teardown/hash/no-extraction asserts)"
        status: pass
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountStop"
        status: pass
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountNoExtraction"
        status: pass
    human_judgment: false
  - id: D3
    description: "Mount contract: helper env allowlist under a hostile parent, spaced TMPDIR accepted whole, newline TMPDIR rejected before spawn, early-exit stdout drained, stderr tap in diagnostics"
    requirement: "VER-01"
    verification:
      - kind: unit
        ref: "internal/supervisor/mount_internal_test.go#TestMountEnv"
        status: pass
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountSpacedTmpdir"
        status: pass
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountRejectBadTmpdir"
        status: pass
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountEarlyExitOutput"
        status: pass
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountStderrTap"
        status: pass
    human_judgment: false
  - id: D4
    description: "Install gained AppImage/Version/RequiresMount and OverrideInstall maps directory/binary/packed .AppImage with an absolutized AppImage path"
    requirement: "MNT-02"
    verification:
      - kind: integration
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountEndToEnd (absolute packed override route)"
        status: pass
    human_judgment: true
    rationale: "Direct unit coverage of the relative/dangling override arms lands in plan 10-04 (TestOverrideInstall); this plan proves only the absolute-path e2e route."

duration: 38 min
completed: 2026-11-24
plan_head_before: f34b7b48a1758546de1f75f912c1b81d2da0d81e
plan_head_after: 8669ba0
status: complete
---

# Phase 10 Plan 01: Mount-First AppImage Backend Summary

**A packed `.AppImage` is mounted by a supervised `--appimage-mount` helper and the app-server serves the full ACP ride from the mount — no extraction, ordered teardown, unchanged spawn line/env pair**

## Performance

- **Duration:** 38 min
- **Started:** 2026-11-24T01:38Z (approx.)
- **Completed:** 2026-11-24T02:16Z (approx.)
- **Tasks:** 2
- **Files modified:** 9 (4 production, 3 test, 2 TDD evidence records)

## Accomplishments

- `internal/supervisor/mount.go` owns the whole mount lifecycle: private 0700 TMPDIR, allowlisted helper env, readiness line validation, invariant-tree validation, bundled-builtin fill, `Lost` watcher, and the staged `Stop` (TERM → grace → group SIGKILL → reap → mount-gone poll → base removal).
- `Install` gained `AppImage`/`Version` and `RequiresMount`; `supervisor.OverrideInstall` is now the single override mapper and routes a packed `.AppImage` (literal or resolved) through the mount, storing it absolute so `exec` can never PATH-resolve it. `cmd`'s private `overrideInstall` was deleted.
- `run` mounts after discovery and before supervision, passes the mounted install to the front/machinery/spawn seam, and stops the mount only after the supervisor's staged shutdown (app-server first, helper second). The spawn line, env pair, and JSONL dialect are untouched (D-11).
- A FUSE-free end-to-end test drives the built wrapper through a scripted fake AppImage: initialize, two session/new calls, both prompt loops, ordered teardown (the helper records that the app-server was already gone on TERM), zero surviving PIDs, an unchanged AppImage hash, and no extraction copy.
- Five mount contract tests plus the early-exit drain and stderr-tap arms lock the env allowlist, spaced/bad TMPDIR behavior, stop escalation/idempotency, no-extraction, and diagnostic completeness.

## Task Commits

Each task was committed atomically (TDD: RED → GREEN → REFACTOR):

1. **Task 1 RED — packed-AppImage mount e2e** — `ab0bb7d` (test)
2. **Task 1 GREEN — mount owner + override routing + run wiring** — `df2f2c6` (feat)
3. **Task 2 RED — mount contract tests** — `59667e9` (test)
4. **Task 2 GREEN — PATH fallback + stderr tap** — `712c198` (feat)
5. **REFACTOR — lint-driven splits (mount owner, run/serveACP/finishRun)** — `7d370aa` (refactor)
6. **Early-exit stdout-drain contract test** — `8669ba0` (test)

**Plan metadata:** the `docs(10-01)` commit carrying this SUMMARY, STATE.md, and ROADMAP.md.

## Files Created/Modified

- `internal/supervisor/mount.go` — the mount owner: PrepareMount/newMountBase/start, mountTap + cappedBuffer, helperEnv, line/tree validation, Lost watcher, Staged Stop, HelperPID/Lost/Guard/Accessible/Install accessors.
- `internal/supervisor/discover.go` — `Install.AppImage`/`Version`, `RequiresMount`, `OverrideInstall`, `appImageVersion`, packed override diagnostics row, deferred bundled-builtin row for a packed winner.
- `internal/supervisor/spawn.go` — `SpawnOpts.StdoutTap`/`StderrTap`, nil-tolerant `spawnPipes.closeAll`, empty-`argv[0]` rejection, and the `buf[:n]` stderr-forward fix.
- `cmd/zcode-acp/main.go` — deletes the private override mapper, wires `supervisor.OverrideInstall`, adds `acquireMount`/`stopMount`/`stopMountBackstop`, and extracts `serveACP`/`finishRun`.
- `cmd/zcode-acp/main_test.go` — `TestAppImageMountEndToEnd`, the scripted fake AppImage fixture, ordered-teardown/PID/hash/no-extraction asserts, the failed-run orphan cleanup, and the shared `e2eRideTwoLoops` driver.
- `internal/supervisor/mount_test.go` — fake-runtime variants (ok/bad-tree/stubborn/early-exit) and the mount lifecycle contract tests.
- `internal/supervisor/mount_internal_test.go` — the in-package `helperEnv` allowlist/fallback test.
- `.planning/tdd/10-01-task{1,2}-red.json` — the machine-verified RED evidence (`RED_EVIDENCE_OK`).

## Decisions Made

- **Helper env allowlist by construction** (PATH/HOME/TMPDIR only). `APPIMAGE_EXTRACT_AND_RUN` is checked before the runtime's mount branch, so a deny list would be unsound; TMPDIR is the private 0700 base so every `.mount_*` under it is ours.
- **Line-scoped trust, never field splitting.** The whole absolute line is the path (spaces legal); acceptance requires a non-symlink directory named `.mount_*` one component under the resolved base, then the two expected files are stat-validated.
- **Leased stdout instead of `StdoutPipe`.** `SpawnOpts.StdoutTap` lets exec's copier own the drain and `Wait` join it, so a line written immediately before exit is in the failure error; `WaitDelay` is only the close backstop.
- **Teardown order.** App-server staged shutdown first (the supervisor reaps it), helper second (TERM/grace/group SIGKILL/reap), then a mount-gone poll of `<root>/zcode`; only then is the private base removed. Helper stdin is never a stop mechanism (the runtime pauses), and `fusermount` is never required.
- **Absolutized override.** `OverrideInstall` stores the packed AppImage absolute (Abs, then EvalSymlinks when resolvable) so `exec` cannot PATH-resolve a bare basename; a dangling relative override is still stored absolute for the fail-loud message.
- **Deferred bundled builtin.** A packed winner cannot be read before the mount, so the bundled probe records "deferred" and `PrepareMount` fills `BuiltinFile` from `<mount>/resources/config/provider/zcode-builtin.json` only when discovery left it empty.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `drainChildStderr` forwarded the whole 4 KiB buffer instead of the bytes read**
- **Found during:** Task 1 GREEN (helper stderr showed NUL padding after every short read)
- **Issue:** `out := buf` made every forward write the full `stderrChunkSize` buffer; a short read leaked stale/NUL padding to the wrapper's stderr.
- **Fix:** `out := buf[:n]` (the capped branch still uses `buf[:room]`).
- **Files modified:** `internal/supervisor/spawn.go`
- **Verification:** `TestMountHappyPath` output lost the NUL padding; the full suite and lint stay green.
- **Committed in:** `df2f2c6` (Task 1 GREEN)

### Ordering deviations (recorded, no behavior impact)

- **`cappedBuffer` landed with Task 1** because Task 1's behavior already requires a bounded observed-output sink; Task 2 reuses it for `StderrTap` instead of adding a second type.
- **`TestMountStderrTap` and `TestMountEarlyExitOutput` were added after the GREEN commits** (test-only, CONFIRM-GREEN): they lock must-have behavior that the production code already satisfied.

---

**Total deviations:** 1 auto-fixed (1 Rule 1 bug) + 2 recorded ordering deviations.
**Impact on plan:** No scope creep. The bug fix is required for correct diagnostics (and removes a NUL-injection path into stderr); the ordering notes are test/type placement only.

## TDD Gate Compliance

| Phase | Commit | Evidence |
|-------|--------|----------|
| RED (Task 1) | `ab0bb7d` — `test(10-01)` | `TestAppImageMountEndToEnd` executed and failed at the mount-record wait: the AppImage was spawned as the app-server binary (`RED_EVIDENCE_OK`, `.planning/tdd/10-01-task1-red.json`) |
| GREEN (Task 1) | `df2f2c6` — `feat(10-01)` | The same e2e passes: the mount is created, the tree validated, both loops served, exit 0 |
| RED (Task 2) | `59667e9` — `test(10-01)` | `TestMountEnv`'s PATH-fallback arm failed on its assertion (`RED_EVIDENCE_OK`, `.planning/tdd/10-01-task2-red.json`); the other four arms are CONFIRM-GREEN for Task 1 behavior, recorded honestly |
| GREEN (Task 2) | `712c198` — `feat(10-01)` | All five contract tests green; helper PATH fallback and stderr tap shipped |
| REFACTOR | `7d370aa` — `refactor(10-01)` | Lint-driven splits only; full `./...` suite and `golangci-lint` stay green |

No gate violations: every `feat(10-01)` descends from a `test(10-01)` commit, and the RED target tests failed on planned assertions (not build/setup faults).

## Verification

| Command | Result |
|---------|--------|
| `go test -race -count=1 ./cmd/zcode-acp/ -run TestAppImageMountEndToEnd -v` | PASS (0.98s) |
| `go test -race -count=1 ./internal/supervisor/ -run 'TestMountHappyPath\|TestMountEnv\|TestMountStop\|TestMountNoExtraction' -v` | PASS (all four) |
| `go test -race -count=1 ./internal/supervisor/ -run 'TestMountStop\|TestMountSpacedTmpdir\|TestMountRejectBadTmpdir\|TestMountNoExtraction\|TestMountEnv\|TestMountStderrTap\|TestMountEarlyExitOutput' -v` | PASS |
| `go test -race -count=1 ./...` | PASS (full suite) |
| `mise run lint` (golangci-lint 2.15.0) | 0 issues |
| `mise run tidy-diff` | clean (one `require` line, acp-go-sdk v0.20.0) |

Acceptance criteria (all pass): `func PrepareMount` ×1, `func (i Install) RequiresMount` ≥1, `supervisor.PrepareMount` ≥1 in `cmd/zcode-acp/main.go`, `ExtractAppImage` ×0 in `mount.go`; `func helperEnv` ≥1, `StderrTap` ≥1 in `spawn.go`, `func TestMountNoExtraction` ×1.

## Issues Encountered

- The fake helper's `sleep` loop prints `Terminated` to stderr when the group is killed; the capped forward (production behavior) surfaces it in test output. Harmless.
- `requirements.ready-ids` reports MNT-01, MNT-02, MNT-03, MNT-06, and VER-01 **blocked** (sibling Phase-10 plans declare the same IDs and have no SUMMARY yet), so no checkbox was flipped. The orchestrator owns the post-phase completion flips.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 10-02 can build on `ErrMountLost`/`Guard`/`Lost` (already present and wired into the mount owner, not yet into the supervisor's spawn seam) for the failure classification, signal handling, and mount-loss session cause.
- 10-03 replaces the extraction-era `probeAppImageGlob`/`probeUnpackedGlob` with the version-ranked stage; `Install.AppImage`/`Version`/`RequiresMount` and the packed override row are in place.
- 10-04 adds the relative/dangling override unit coverage and the mount-safe PAT version seam (`desktopVersion` still scans only Binary/Root/Entrypoint in this plan).
- 10-05 can drive the guarded real tier through `PrepareMount`/`Install` without new plumbing.

## Self-Check: PASSED

All created files exist (`mount.go`, `mount_test.go`, `mount_internal_test.go`, both RED evidence records); all six commits are ancestors of HEAD; no commit in the plan deleted tracked files.
