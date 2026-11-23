---
phase: 10-mount-first-appimage-backend
plan: 02
subsystem: infra
tags: [appimage, fuse, mount, failure-classification, signals, process-supervision, acp, go, tdd]

# Dependency graph
requires:
  - phase: 10-mount-first-appimage-backend
    provides: 10-01 mount owner (PrepareMount/Mount, private TMPDIR, readiness line validation, staged Stop), StdoutTap/StderrTap, Lost/Guard skeleton
  - phase: 03-zcode-back-leg
    provides: SpawnChild process-group/reap primitives, the staged app-server shutdown, the fake app-server binary
provides:
  - Four-part mount failure errors (AppImage, step, capped %q observed output, remediation) with classifyMountFailure for FUSE/libfuse2, invalid-image, and permission classes
  - Statfs presence seam and FUSE-magic fallback; live mounts are never traversed by RemoveAll and the private base is retained on an unprovable teardown
  - Mid-session loss handling (Lost closes once, Guard forbids respawn, the run ends with the mount cause)
  - SIGINT/SIGTERM/SIGHUP staged shutdown with a non-signal spawn context and deterministic exit-cause precedence (teardown error always wins)
affects: [10-03 discovery ranking, 10-04 override completeness, 10-05 real tier, 10-06 docs]

# Actuals (#2632) — same chars/4 scale as the plan's estimate.
actuals:
  tokens: 23082
  tasks: 2
  commits: 6

tech-stack:
  added: []
  patterns:
    - "Failure classification maps capped helper output to a remediation family, but always carries the raw observed text and never keys solely on libfuse2"
    - "Mount presence is zcode-first with the FUSE superblock magic as the fallback, behind an injectable Statfs seam; RemoveAll runs only after the proof"
    - "Two contexts: signalCtx cancels supervision, while spawnCtx derives from context.WithoutCancel so a signal cannot SIGKILL the app-server before its stdin-EOF drain"
    - "Exit-cause precedence is a pure classifier: teardown error > supervisor failure > proven-gone signal/client > mount cause > errChildLost"
    - "The Lost watcher checks a stopping flag before every loss verdict, so a teardown never masquerades as a loss"

key-files:
  created: []
  modified:
    - internal/supervisor/mount.go
    - internal/supervisor/mount_test.go
    - cmd/zcode-acp/main.go
    - cmd/zcode-acp/main_test.go

key-decisions:
  - "classifyMountFailure classifies the known legacy strings but the error always includes the raw %q-capped output and a remediation; the FUSE family carries both /dev/fuse and libfuse2t64/libfuse2 guidance because newer static runtimes do not need libfuse2"
  - "zcode presence alone is not a mount proof for a tree that failed validation; MountOpts.Statfs (default syscall.Statfs, FUSE magic 0x65735546) is the fallback, so a live mount is never traversed"
  - "A teardown that cannot prove the mount gone leaves the 0700 base in place and returns the mount plus manual-unmount remediation; a signal never masks it"
  - "The spawn context is deliberately not derived from the signal context; the transcript written only at the app-server's clean EOF-drain exit is the proof it was drained, not SIGKILLed"
  - "A signal during discovery or mount acquisition is a clean exit 0; the helper is still stopped and reaped by PrepareMount's failure path"

patterns-established:
  - "Fake-runtime lifecycle variants: die-after/vanish wait for an app-server PID release before removing the tree, so loss is exercised on an already-validated mount"
  - "PID-record fixtures: every helper writes $$ to an absolute file baked in at fixture creation, letting the external test package assert ESRCH without a Child handle"
  - "Statfs-seam teardown test: FUSE magic + absent zcode proves the fallback keeps Stop from traversing a live mount and retains the base"

requirements-completed: [MNT-03, MNT-04, MNT-05, MNT-06, VER-01]

coverage:
  - id: D1
    description: "Missing FUSE, early helper death, readiness timeout, bogus path lines, invalid trees, and late text all fail loud with the AppImage, step, observed output, and remediation; the helper is reaped, the base removed, and no extraction copy appears"
    requirement: "MNT-04, VER-01"
    verification:
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountFailures"
        status: pass
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountFailures"
        status: pass
    human_judgment: false
  - id: D2
    description: "A mount that cannot be proven gone fails Stop at the deadline naming the mount and manual-unmount remediation; the private base is retained, never traversed by RemoveAll, and the helper is still reaped"
    requirement: "MNT-04, MNT-03"
    verification:
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountTeardownFailure"
        status: pass
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountSignal/teardown_failure_outranks_the_signal"
        status: pass
    human_judgment: false
  - id: D3
    description: "Helper death or tree loss after a validated mount closes Lost exactly once, Accessible is false, Guard wraps ErrMountLost, the guarded spawn seam runs once, and the binary exits 1 with the mount cause"
    requirement: "MNT-05"
    verification:
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountLost"
        status: pass
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountGuard"
        status: pass
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountLost"
        status: pass
    human_judgment: false
  - id: D4
    description: "SIGINT, SIGTERM, and SIGHUP run the staged teardown (app-server first, helper second), exit 0, leave zero orphans/mounts, and the app-server transcript proves a clean stdin-EOF drain; a signal during readiness exits 0 with no helper"
    requirement: "MNT-03"
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountSignal"
        status: pass
    human_judgment: false
  - id: D5
    description: "No failure, loss, signal, or teardown path falls back to discovery/extraction or leaves a squashfs-root/zcode-<ver> copy; the AppImage is untouched"
    requirement: "MNT-06"
    verification:
      - kind: unit
        ref: "internal/supervisor/mount_test.go#TestMountFailures (assertNoExtractionCopy on every arm)"
        status: pass
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageMountEndToEnd (hash/no-extraction, still green)"
        status: pass
    human_judgment: false

duration: 20 min
completed: 2026-11-24
plan_head_before: 73bbf585d2f8cd12881073e131d2e0185c446634
plan_head_after: 28ff2ad6c2c4b69138fa0512346befd11f1b4483
status: complete
---

# Phase 10 Plan 02: Mount Failure, Loss, and Signal Teardown Summary

**Every mount failure now exits 1 with the AppImage, step, observed output, and a remediation; a lost mount ends the session with an explicit cause and no respawn; SIGINT/SIGTERM/SIGHUP tear down cleanly with zero orphans — and a teardown that cannot prove the mount gone still exits 1, even under a signal.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-11-23T23:07:38Z
- **Completed:** 2026-11-23T23:27:16Z
- **Tasks:** 2
- **Files modified:** 4 (+2 TDD evidence records)

## Accomplishments

- `internal/supervisor/mount.go` now builds every readiness/spawn/validation/teardown error through `Mount.failure`: the AppImage path, the failing step (`spawn the helper`, `read the mount line`, `validate the mount tree`, `teardown`), the `%q`-quoted capped observed output, and a remediation. `classifyMountFailure` covers the legacy FUSE/libfuse2 strings, invalid-image strings, and permission/TMPDIR strings; the FUSE family names both the `/dev/fuse`/unprivileged-mount checks and `libfuse2t64`/`libfuse2` (T-10-08).
- Teardown is presence-gated: `<root>/zcode` first, then `MountOpts.Statfs` with the Linux FUSE superblock magic (`0x65735546`) for a tree that failed validation. A mount whose line never arrived is found by scanning the private base's `.mount_*` children with the same predicate. `RemoveAll` runs only after the proof; a live mount retains the 0700 base and returns the mount plus `fusermount -u` remediation (never a warning-only outcome).
- Mid-session loss is complete: `Lost` closes exactly once on helper death or a lost tree, `Accessible`/`Guard` report the vanished root, and a stopping flag set before `Stop` signals the helper prevents an intentional teardown from masquerading as a loss. The production `SpawnFunc` calls `mount.Guard()` before every spawn, so the supervisor cannot restart against a vanished root (T-10-08).
- `cmd/zcode-acp/main.go` installs `signal.NotifyContext(context.Background(), os.Interrupt, SIGTERM, SIGHUP)`. `signalCtx` drives discovery, mount acquisition, and the supervisor; `spawnCtx` derives from `context.WithoutCancel(signalCtx)` and is canceled only after `startDone` returns. The run selects on `mount.Lost` and the signal, and the exit classifier enforces: teardown error (1) > supervisor failure (1) > proven-gone signal/client (0) > mount cause (1) > `errChildLost` (1). A signal during discovery or acquisition exits 0 after PrepareMount stops and reaps the helper.
- TDD evidence: both tasks produced machine-verified RED records (`.planning/tdd/10-02-task{1,2}-red.json`, `RED_EVIDENCE_OK`), and the GREEN commits make the target tests pass.

## Task Commits

Each task was committed atomically (TDD: RED → GREEN → REFACTOR):

1. **Task 1 RED — failure classification and loud exit** — `7ad0343` (test)
2. **Task 1 GREEN — classify failures, Statfs-gated teardown** — `943d8b9` (feat)
3. **Task 1 teardown-failure contract (CONFIRM-GREEN)** — `327cd4c` (test)
4. **Task 2 RED — loss, spawn guard, OS signals** — `94a5a2c` (test)
5. **Task 2 GREEN — signal shutdown, loss, spawn guard** — `efb5cba` (feat)
6. **REFACTOR — lint-driven test-matrix splits** — `28ff2ad` (test)

**Plan metadata:** the `docs(10-02)` commit carrying this SUMMARY, STATE.md, ROADMAP.md, and REQUIREMENTS.md.

## Files Created/Modified

- `internal/supervisor/mount.go` — four-part failure builder + `classifyMountFailure`; remediation constants; `MountOpts.Statfs` + `defaultStatfs`; FUSE-magic presence predicate; base-wide `waitBaseMountsGone`; `RemoveAll` gated on the gone proof; `stopping` flag in `Stop`/`watch`.
- `internal/supervisor/mount_test.go` — `TestMountFailures` matrix, `TestMountTeardownFailure`, `TestMountLost` (die-after, vanish, stop-is-not-loss), `TestMountGuard`; PID-recording fake-runtime variants (no-line, bogus-line, missing-tree, no-fuse, late-text, leaky, die-after, vanish); retention/reap assertions.
- `cmd/zcode-acp/main.go` — `signal.NotifyContext`; non-signal `spawnCtx`; `runOutcome` and the `Lost`/signal select arms; guarded `SpawnFunc`; `mountLostCause`, `awaitSupervisorShutdown`, and `classifyRunExit` precedence.
- `cmd/zcode-acp/main_test.go` — `TestAppImageMountFailures`; `TestAppImageMountSignal` (SIGTERM/SIGINT/SIGHUP + startup signal + teardown-failure); `TestAppImageMountLost`; lifecycle fake-runtime variants and exit/PID wait helpers.
- `.planning/tdd/10-02-task1-red.json`, `.planning/tdd/10-02-task2-red.json` — machine-verified RED evidence.

## Decisions Made

- **Observed output is always carried.** Classification chooses the remediation sentence, but the raw capped stdout+stderr rides the error regardless, so a newer static runtime's unknown failure text is still actionable and a wrong libfuse2 diagnosis cannot hide the truth.
- **Presence is a proof, not a guess.** For a tree that failed validation the zcode probe cannot work, so the FUSE superblock type is the fallback; a plain leftover directory is correctly treated as gone (the fake runtime's case), while a real FUSE mount retains the base.
- **Teardown error has absolute precedence.** Even with a signal already delivered, a mount that cannot be proven gone exits 1; signal classification only yields 0 when Stop returned nil.
- **The spawn context is not signal-derived.** The fake transcript is written only after the app-server's clean EOF-drain exit; its presence after SIGTERM/SIGINT/SIGHUP is the proof the child was drained, not SIGKILLed mid-flight.
- **Loss vs teardown.** The stopping flag is set before the helper is signalled; the watcher checks it before every loss verdict, so Stop can never close `Lost`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The fake-runtime harness's PID record polluted the no-extraction fixture comparison**
- **Found during:** Task 1 GREEN (full supervisor suite)
- **Issue:** `TestMountNoExtraction` compared the fixture directory before/after the mount cycle; the new `helper.pid` harness record appeared only after the cycle and failed the equality check.
- **Fix:** The comparison filters the harness PID record (the assertion targets extraction copies, not test bookkeeping).
- **Files modified:** `internal/supervisor/mount_test.go`
- **Verification:** Full `./internal/supervisor/` suite green.
- **Committed in:** `327cd4c` (Task 1 test commit)

### Ordering deviations (recorded, no behavior impact)

- **`TestMountTeardownFailure` landed after the GREEN commit** (test-only, CONFIRM-GREEN) because it references the new `MountOpts.Statfs` seam; a RED record would have been a build failure, not a valid RED.
- **`TestMountEarlyExitOutput` moved from the early-exit fixture to the late-text fixture.** The plan defines early-exit as a helper that prints nothing and exits 127 (covered by `TestMountFailures`); the existing drain-proof test now uses the variant that actually writes a final line.
- **Task 2's supervisor-tier die-after/vanish/guard arms were already satisfied by 10-01's `Lost`/`Guard` skeleton** (CONFIRM-GREEN); the new RED surface was the intentional-stop-is-not-loss arm and both binary-tier signal/loss tests.
- **A lint-driven test refactor followed the GREEN commit** (named variant constants, table functions, extracted assertion helpers); no production behavior changed.

---

**Total deviations:** 1 auto-fixed (Rule 1) + 4 recorded ordering deviations.
**Impact on plan:** No scope creep. The auto-fix is test bookkeeping; the ordering notes are TDD-placement and lint-shape only.

## TDD Gate Compliance

| Phase | Commit | Evidence |
|-------|--------|----------|
| RED (Task 1) | `7ad0343` — `test(10-02)` | `TestMountFailures` failed on the missing step names/remediation (`RED_EVIDENCE_OK`, `.planning/tdd/10-02-task1-red.json`); the binary failure test failed on the missing remediation |
| GREEN (Task 1) | `943d8b9` — `feat(10-02)` | All six failure arms and four binary failure arms pass |
| RED (Task 2) | `94a5a2c` — `test(10-02)` | `TestAppImageMountSignal` observed the default signal death and `TestAppImageMountLost` never exited (`RED_EVIDENCE_OK`, `.planning/tdd/10-02-task2-red.json`); the supervisor stop-masquerade arm failed |
| GREEN (Task 2) | `efb5cba` — `feat(10-02)` | Signals exit 0 with the served transcript; loss exits 1; teardown failure exits 1 under a signal |
| REFACTOR | `28ff2ad` — `test(10-02)` | Lint-driven test splits only; full suite and `golangci-lint` stay green |

No gate violations: every `feat(10-02)` descends from a `test(10-02)` commit, and each RED target failed on a planned assertion (step naming, remediation, default signal death, missing loss exit), not on build/setup faults.

## Verification

| Command | Result |
|---------|--------|
| `go test -race -count=1 ./internal/supervisor/ -run 'TestMountFailures\|TestMountLost\|TestMountGuard' -v` | PASS (all arms) |
| `go test -race -count=1 ./cmd/zcode-acp/ -run 'TestAppImageMountFailures\|TestAppImageMountSignal\|TestAppImageMountLost' -v` | PASS (all arms; teardown-failure ~10 s) |
| `go test -race -count=1 ./...` | PASS (full suite) |
| `mise run lint` (golangci-lint 2.15.0) | 0 issues |
| `go vet ./...` | clean |
| `mise run tidy-diff` | clean (one `require` line, acp-go-sdk v0.20.0) |

Acceptance criteria (all pass): `func classifyMountFailure` ×1, `libfuse2t64` ≥1, `func TestAppImageMountFailures` ×1, `signal.NotifyContext` ×1, `ErrMountLost` ≥1 in `mount.go`, `func TestAppImageMountSignal` ×1, `func TestAppImageMountLost` ×1, `mount.Guard` ≥2 in `main.go`, `func TestMountTeardownFailure` ×1.

## Issues Encountered

- The teardown-failure binary arm runs the production timings (5 s helper grace + 10 s mount-gone deadline); the subtest uses a 45 s exit watchdog and removes the retained base in `t.Cleanup`. It completes in ~10–11 s on this host.
- `requirements.mark-complete` accepted all five IDs (MNT-03, MNT-04, MNT-05, MNT-06, VER-01) this wave, unlike 10-01 where sibling plans blocked the ready check.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 10-03 can rely on the failure/loss/signal contract; discovery changes are independent of the mount lifecycle.
- 10-04's override completeness inherits the mount failure classification (packed overrides that fail to mount exit 1 with remediation).
- 10-05's guarded real tier can reuse `DefaultMountOpts` (production Statfs) without new plumbing.
- 10-06 documents the new failure/remediation strings and the signal behavior.

## Self-Check: PASSED

All created/modified files exist; all six commits are ancestors of HEAD; no commit in the plan deleted tracked files.
