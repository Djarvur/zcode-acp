---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 01
subsystem: session-lifecycle
tags: [acpfront, lifecycle, teardown, race-detector, closing-gate, wr05-toctou, zombie-session, tdd]

# Dependency graph
requires:
  - phase: 06-oauth-coding-plan-path-model-picker
    provides: the SESS-01 teardown contract (Front.CloseSession ordering, lifecycle.go), the confirmed-settings store (recordSettings/dropSettings), the picker setConfigMutation model arm, and the routed WR-01/WR-04/WR-05/IN-04 findings (06-REVIEW-DISPOSITION.md)
provides:
  - WR-05 registerIfOpen: the resume's closing re-check + reattach release + registry Register under ONE closingMu critical section, with the one-way closingMu -> registry/conn-fan-out lock order documented at the method
  - WR-01 teardownAfterPickerRefusal: a refused picker setModel on an established session runs the full SESS-01 teardown — no zombie front session against a dead mutation target; the client-facing refusal reply is byte-unchanged
  - WR-04 failure-path bounding: closeOrphan drops cwd+settings (both create-path anchors in one place); the resume subscribe-refusal return drops its cwd — zero map leakage on all three post-create failure paths (the WR-07 invariant holds on failure paths)
  - IN-04: the dead settings==nil re-init removed from recordSettings (the store is constructor-initialized only; the load-bearing closing lazy-init untouched)
  - the in-package test tier (lifecycle_internal_test.go + siblings) asserting the front's unexported close-gate/registry/store state, with an idempotent transcript-returning composed-leg harness
affects: [07-02, 07-03 (session-compat arms build on this teardown contract), phase-7 review]

# Actuals (#2632) — pairs with the plan's `estimate` (70000 tokens) to calibrate future estimates.
actuals:
  tokens: 10139    # chars/4 over the realized diff (40555 chars, 5 files, +873/-28)
  tasks: 3
  commits: 7
  plan_head_before: 4b6d98a1f5ea4be20ee4f40871013b4f5e1064b7
  plan_head_after: 649b23c33790716f8aba0ec5376890072ccc6315

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "atomic register-if-open: the closing re-check, the previous-machine release, and Register under one closingMu section (one-way nesting closingMu -> registry lock / conn fan-out mutexes)"
    - "failure-path closer owns bounding: closeOrphan drops the session's cwd/settings entries BEFORE the best-effort wire close, so a failed close cannot skip them"
    - "in-package test files (*_internal_test.go) for locks that read the front's unexported closingMu/closing/registry/cwds/settings state; the external acpfront_test suite keeps locking the wire surfaces"

key-files:
  created:
    - internal/acpfront/lifecycle_internal_test.go
    - internal/acpfront/failurepaths_internal_test.go
    - internal/acpfront/pickerrefusal_internal_test.go
    - .planning/tdd/07-01-task1-red.json
    - .planning/tdd/07-01-task2-red.json
    - .planning/tdd/07-01-task3-red.json
  modified:
    - internal/acpfront/lifecycle.go
    - internal/acpfront/front.go

key-decisions:
  - "registerIfOpen folds the in-process reattach release (prev.Machine.CloseObservers) into the closingMu section — the WR-05 stress probe caught the unguarded pre-check placement racing CloseSession's teardown CloseObservers under -race; the section also stops a closing-rejected resume from closing the live machine's observers"
  - "closeOrphan is the WR-04 failure-path closer and drops cwd+settings before the wire attempt — one change covers the subscribe-refused and launch-model-refused create anchors; the resume subscribe-refusal return drops its cwd explicitly (that path has no orphan close)"
  - "WR-01's teardown extracted into teardownAfterPickerRefusal (front.go) — mirrors closeOrphan's structural logging discipline (session/stage/cause/err) and keeps SetSessionConfigOption inside funlen"
  - "in-package test files carry the unexported-state assertions (registry/closing/cwds/settings) — the plan targeted the external-package files, which cannot read those fields"

patterns-established:
  - "atomic register-if-open section (WR-05): a surviving resume either answers the coded closing rejection or registers before any later close can mark"
  - "WR-07 bounding on failure paths: every post-create failure path leaves zero cwd/settings entries"
  - "in-package interleave harness: fake.Serve <-> zcode.Conn <-> Front with no SDK client leg; direct machine registration mirrors NewSession's tail"

requirements-completed: [SESS-02, SESS-03]

coverage:
  - id: D1
    description: "WR-05: no resume can register a machine into a closing session — the closing re-check, reattach release, and Register are one closingMu-guarded section, deterministic under -race across the 2000-iteration stress and three count=3 repeats"
    requirement: SESS-02
    verification:
      - kind: unit
        ref: "internal/acpfront/lifecycle_internal_test.go#TestResumeCloseInterleave"
        status: pass
      - kind: unit
        ref: "internal/acpfront/lifecycle_internal_test.go#TestRegisterIfOpen"
        status: pass
      - kind: other
        ref: "grep -c 'closingMu.Lock()' internal/acpfront/lifecycle.go -> 3 (>= 2 gate)"
        status: pass
    human_judgment: false
  - id: D2
    description: "WR-04 + IN-04: all three post-create failure paths leave zero cwd/settings entries; closeOrphan drops both stores; the dead settings==nil guard is gone (grep gate prints 0); the happy path still records and teardown still drops"
    requirement: SESS-03
    verification:
      - kind: unit
        ref: "internal/acpfront/failurepaths_internal_test.go#TestCreateFailurePathsLeaveNoMapEntries"
        status: pass
      - kind: other
        ref: "grep -c 'if f.settings == nil' internal/acpfront/front.go -> 0 (EXPECT==0 gate)"
        status: pass
    human_judgment: false
  - id: D3
    description: "WR-01: a refused picker setModel fully tears the front session down (registry released, stores dropped, no post-refusal backend send), the coded refusal reply is preserved, successful switches keep the session alive, and the create-path orphan-close call sites are unchanged (grep gate prints 0)"
    requirement: SESS-03
    verification:
      - kind: unit
        ref: "internal/acpfront/pickerrefusal_internal_test.go#TestPickerSetModelRefusalRunsFullTeardown"
        status: pass
      - kind: other
        ref: "grep -c 'closeOrphan(ctx, sessionID, \"set_config_option setModel\"' internal/acpfront/front.go -> 0 (EXPECT==0 gate)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Plan-level verification: the full internal suite under -race and the whole-tree mise run ci gate (build -> vet -> lint -> test -> test-experiments) are green at wave close"
    verification:
      - kind: other
        ref: "mise run ci (exit 0, lint 0 issues, all packages ok under -race)"
        status: pass
    human_judgment: false

# Metrics
duration: 39min
completed: 2026-11-22
status: complete
---

# Phase 7 Plan 01: Lifecycle-Consistency Pass (WR-01/04/05 + IN-04) Summary

**WR-05's resume/close TOCTOU closed under one closingMu-guarded registerIfOpen section (with the reattach-release race the stress probe caught folded in), refused picker setModel now runs the full SESS-01 teardown (no zombie), and all three post-create failure paths drop their cwd/settings entries with the IN-04 dead nil-guard gone — -race and mise run ci fully green.**

## Performance

- **Duration:** 39 min
- **Started:** 2026-11-22T10:53:29Z
- **Completed:** 2026-11-22T11:32:29Z
- **Tasks:** 3 (all RED→GREEN)
- **Files modified:** 5 (+873/-28; 3 test files created, 2 production files hardened)

## Accomplishments

- **WR-05 (Task 1, tracer):** `registerIfOpen` holds `closingMu` across the closing re-check (read inline — no nested isClosing), the in-process reattach's previous-machine release, and `registry.Register`. A resume that survived the entry gate either answers the coded closing rejection or registers before any later close can mark; the lock-order discipline (one-way `closingMu -> registry lock / conn fan-out mutexes`; CloseSession keeps releasing closingMu before its Get/Remove) is documented at the method. The plan's 2000-iteration stress probe (`TestResumeCloseInterleave`) runs deterministic under `-race -count=3`, and `TestRegisterIfOpen` locks the serialization contract (held mutex + mark blocks registration past the subscribe; the unmarked control registers).
- **WR-04 + IN-04 (Task 2):** `closeOrphan` drops the session's cwd and settings entries before the best-effort wire close (subscribe-refused and launch-model-refused creates in one place); the resume's re-issued subscribe refusal drops its recorded cwd explicitly. `recordSettings` lost the dead `settings == nil` re-init (the only constructor initializes the store; never reassigned); the load-bearing `closing` lazy-init in CloseSession is byte-unchanged. `TestCreateFailurePathsLeaveNoMapEntries` locks all three refusal arms (genuine RED pre-fix) plus the IN-04 liveness arm.
- **WR-01 (Task 3):** the refused picker setModel path replaces the wire-only `closeOrphan` with the full `Front.CloseSession` teardown via `teardownAfterPickerRefusal` (structural slog.Warn on a teardown failure — never a silent skip; the client-facing coded refusal unchanged). `TestPickerSetModelRefusalRunsFullTeardown` locks: registry released + stores dropped (RED pre-fix: the zombie persisted), refusal reply preserved, a follow-up prompt lands the coded invalid-request closing/closed class with ZERO backend sends (RED pre-fix: the prompt actually reached session/send on the dead backend), and a successful switch keeps the session fully alive with no close frame on the wire.
- **Gate greps:** `closingMu.Lock()` count 3 (>= 2); `if f.settings == nil` count 0; the picker-refusal `closeOrphan(ctx, sessionID, "set_config_option setModel"` call site count 0 — the two create-path orphan closes remain untouched.
- **Plan-level verification:** `go test -race -count=1 ./internal/...` all green; `mise run ci` exit 0 (lint 0 issues) at wave close; the three named tests green under `-race -count=3`.

## Task Commits

Each task was committed atomically (RED → GREEN per TDD):

1. **Task 1 (tracer): WR-05 registerIfOpen** — RED `f4338cd` (test), GREEN `8fce234` (feat)
2. **Task 2: WR-04 + IN-04** — RED `4ea7238` (test), GREEN `c9262e8` (feat)
3. **Task 3: WR-01 full teardown** — RED `c5c340f` (test), GREEN `ca973e4` (feat)
4. **Lint debt** — `649b23c` (style: teardownAfterPickerRefusal extraction for funlen, ctx threading, intrange, lll rewraps)

**Plan metadata:** this SUMMARY + STATE/ROADMAP/REQUIREMENTS ride the final docs commit.

## TDD Gate Compliance

| Task | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| 1 (tracer) | `f4338cd` (CONFIRM-UNREPRODUCED — see below) | `8fce234` | — | Pass-with-note |
| 2 | `4ea7238` (`RED_EVIDENCE_OK` target_test_failed) | `c9262e8` | — | Pass |
| 3 | `c5c340f` (`RED_EVIDENCE_OK` target_test_failed) | `ca973e4` | `649b23c` (style only, tests stayed green) | Pass |

- **Task 1's RED is the plan's own anticipated CONFIRM-UNREPRODUCED branch.** Both WR-05 tests were written first and executed honestly against the current inline sequence: they passed (exit 0) — the re-check-to-Register window cannot land pre-fix (close's whole teardown cannot fit inside the ~2µs NewMachine span; the re-check reads the same closingMu-guarded mark WR-03 set). The classifier verdict on the honest record is `INVALID_RED (unexpected_green)` — persisted verbatim in `.planning/tdd/07-01-task1-red.json` with the semanticAssessment. No failing assertion was manufactured; the plan explicitly instructs: "record that honestly as CONFIRM-UNREPRODUCED in the RED evidence (the 05-08 CONFIRM-GREEN precedent; never fabricate a RED), keep Test 2 as the contract lock, and proceed." GREEN then made the invariant structural; the stress probe's real value showed immediately when it caught the CloseObservers data race (see Deviations) — the gate's protective purpose was served.
- **Tasks 2 and 3 produced genuine REDs** — `RED_EVIDENCE_OK target_test_failed` from the classifier on the actual runs; evidence persisted in `.planning/tdd/07-01-task{2,3}-red.json`.
- Gate sequence in git log: every `feat(07-01)` descends from a `test(07-01)` commit of this plan (f4338cd→8fce234, 4ea7238→c9262e8, c5c340f→ca973e4). No `refactor(07-01)` was needed (the style commit is lint-debt resolution, not behavior).

## Files Created/Modified

- `internal/acpfront/lifecycle.go` — `registerIfOpen` (the atomic close-gate section + reattach release), the resume tail rewired to it, the resume subscribe-refusal cwd drop, doc-comment updates
- `internal/acpfront/front.go` — `closeOrphan` drops cwd+settings before the wire attempt; `recordSettings` dead nil-guard removed with the invariant documented; the picker-refusal path calls `teardownAfterPickerRefusal` (new helper running the full SESS-01 teardown)
- `internal/acpfront/lifecycle_internal_test.go` (NEW) — `TestResumeCloseInterleave` (2000-iteration stress) + `TestRegisterIfOpen` (serialization contract lock + liveness control) + the shared in-package interleave harness (fake.Serve ⇄ zcode.Conn ⇄ Front, idempotent transcript-returning closer)
- `internal/acpfront/failurepaths_internal_test.go` (NEW) — `TestCreateFailurePathsLeaveNoMapEntries`: the three refusal arms + the IN-04 liveness arm
- `internal/acpfront/pickerrefusal_internal_test.go` (NEW) — `TestPickerSetModelRefusalRunsFullTeardown`: the zombie-gone, reply-preserved, no-backend-send, and success-keeps-alive arms
- `.planning/tdd/07-01-task{1,2,3}-red.json` (NEW) — the persisted RED evidence records (command, exit code, output, target, expected/actual, semanticAssessment)

## Decisions Made

- **The reattach release moved inside registerIfOpen (Rule 1).** The stress probe caught a real data race at HEAD: `ResumeSession`'s unguarded reattach `CloseObservers` raced `CloseSession`'s teardown `CloseObservers` on the same machine (`-race` report across the count=3 repeats). Folding it into the closingMu section fixes the race AND a latent inconsistency: a closing-rejected resume no longer closes the live machine's observers before rejecting. CloseSession's own code is untouched.
- **closeOrphan owns the WR-07 failure-path drops** (the plan's suggested shape): one change covers both create-path anchors; drops ride BEFORE the wire attempt so a failed best-effort close cannot skip them.
- **In-package test files for the unexported-state locks.** The plan's read_first assumed in-package access ("tests ... read f.registry/f.closing/f.cwd/f.settings directly"), but every existing test file is `package acpfront_test` (external). The assertions the plan specifies are unreachable from there, so each task's test landed in a `*_internal_test.go` file (`package acpfront`); acpport stays import-safe (it does not import acpfront — no cycle).
- **The resume subscribe-refusal cwd drop is unconditional** (the plan's letter). Nuance noted: a refused re-issue of an ALREADY-REGISTERED session now drops that live entry's anchor (at HEAD the failed resume had already overwritten it with the resume's cwd — the pre-existing overwrite is unchanged by this plan). The bounding invariant wins; the degraded-anchor edge (relative paths in tool cards via resolveAbs's no-cwd arm) is acceptable on this backend-failure path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Data race: reattach CloseObservers vs teardown CloseObservers**
- **Found during:** Task 1 GREEN verification (`-race -count=3` on the stress probe — the plan's own verify gate caught it)
- **Issue:** `ResumeSession`'s in-process reattach block called `prev.Machine.CloseObservers()` unguarded, racing `CloseSession`'s teardown `st.Machine.CloseObservers()` on the same machine (`Machine.CloseObservers` read/writes its remove-func fields with no lock) — a pre-existing race on the exact resume/close interleave this task hardens
- **Fix:** folded the reattach release into `registerIfOpen`'s closingMu section — the section-holder excludes a concurrent close from marking until the section completes, and close's CloseObservers either preceded the section's unmarked read or follows it (no overlap possible)
- **Files modified:** internal/acpfront/lifecycle.go
- **Verification:** `go test -race -count=3 -run TestResumeCloseInterleave` green ×2 consecutive runs (previously ~1 run in 2 failed with WARNING: DATA RACE); full suite + mise run ci green
- **Committed in:** 8fce234 (Task 1 GREEN commit)

**2. [Rule 3 - Blocking] Plan-targeted test files are external-package; the specified assertions are unexported-state reads**
- **Found during:** Task 1 RED (before writing tests)
- **Issue:** the plan targets `lifecycle_test.go` / `newsession_test.go` / `setconfigoption_test.go` for tests that read `f.closingMu` / `f.closing` / `f.registry` / `f.cwds` / `f.settings` directly — all unexported, and all three files are `package acpfront_test`
- **Fix:** new in-package files (`lifecycle_internal_test.go`, `failurepaths_internal_test.go`, `pickerrefusal_internal_test.go`) holding exactly the tests the plan specifies, reusing the plan's fixture shapes; the external files are untouched
- **Files modified:** the three new test files
- **Verification:** the tests run in the package's normal `go test` invocation; `go vet` clean; lint green
- **Committed in:** f4338cd, 4ea7238, c5c340f (the RED commits)

---

**Total deviations:** 2 auto-fixed (1 Rule 1 bug, 1 Rule 3 blocking) + the plan-sanctioned CONFIRM-UNREPRODUCED RED branch on Task 1 (documented above, not a deviation from the plan — it IS the plan's branch).
**Impact on plan:** the race fix was necessary for the plan's own verify gate to pass deterministically and lands squarely in the task's hardened interleave; the file-placement deviation preserves every planned assertion. No scope creep — the session/new and launch orphan-close call sites, the CloseSession lazy-init, and the load-bearing teardown ordering are untouched.

## Issues Encountered

- The Task 1 stress test initially hung: the success-arm subtest called the harness closer twice (deferred + explicit) and the second `<-serveDone` receive blocked forever. Fixed by making the closer idempotent (`sync.Once`) — the fix landed with the Task 3 RED commit since the transcript-returning closer arrived there.
- `mise run lint` initially reported 20 issues from the new code (contextcheck on the harness call sites, funlen on SetSessionConfigOption/TestRegisterIfOpen, intrange, unnamedResult, 3 lll). All resolved in `649b23c` with repo-convention suppressions (wire-shape reasons on nolint comments) and a helper extraction (teardownAfterPickerRefusal) — the tree stayed lint-green.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- **The SESS-01 teardown contract is now hardened end-to-end and race-clean** — 07-03's session-compat arms (delete `{}` riding CloseSession semantics where registered) build on exactly the contract this plan locked.
- **06-REVIEW-DISPOSITION.md dispositions are flip-ready** with the landing commits: WR-01 → resolved `ca973e4`, WR-04 → resolved `c9262e8`, WR-05 → resolved `8fce234`, IN-04 → resolved `c9262e8`. (IN-01/IN-02/IN-03 remain open for their routed plans.)
- One watch item carried forward: the unconditional resume subscribe-refusal cwd drop (Decisions Made) — if a future phase ever re-issues resumes against live registered sessions in production flows, revisit the reattach-anchor nuance.

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*
