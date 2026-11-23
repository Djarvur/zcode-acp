---
phase: 02-scaffold-ci-bootstrap
plan: 02
subsystem: testing
tags: [node-test, tdd, scrubber, wr-03, in-04, boundary-regex, capture-header, secret-gate]

# Dependency graph
requires:
  - phase: 01-research-protocol
    provides: experiments/scrub-capture.mjs + its 12-test node:test suite (the scrub-first capture pipeline)
  - phase: 02-scaffold-ci-bootstrap (plan 02-01)
    provides: strict lint gate + canonical gate order (the suite this plan hardens is test-experiments in that order)
provides:
  - "Boundary-aware home rewrite in scrub-capture.mjs (escapeRegExp + negative lookahead (?!/?[A-Za-z0-9._-])) — sibling-of-home paths survive verbatim; home-anchored and bare-home tilde rewrites preserved; idempotent"
  - "Capture-header guard (IN-04): line 1 must be a JSON object with a capture key, else clear stderr + exit 1; --no-header escape hatch scrubs line 1 as a data line; empty-input contract unchanged"
  - "6 new node:test cases (WR-03 siblings + idempotency, IN-04 fail/pass/flag) — suite 12 -> 18, RED->GREEN proven"
  - "RED evidence records (5x RED_EVIDENCE_OK) persisted in the phase evidence/ dir"
affects: [02-03 (rewrites scan-secrets.sh that these verifies execute), 02-04 (promotes the gate into CI — scrubber contract now test-locked per D-05)]

# Actuals (#2632) — pairs with the plan's estimate (35000) to calibrate future estimates.
actuals:
  tokens: 2270        # chars/4 over the realized diff (9091 diff bytes, 2 files, 123+/4-)
  tasks: 2
  commits: 2          # MEASURED: git rev-list --count gsd-plan-head-before-02-02..HEAD
plan_head_before: 7f16c7a1bced955955e36dfeb56e489ad81dcd31
plan_head_after: 52f660afbf045c7d84e8fa888414fefb7b96b22c

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Boundary-aware rewrite discipline: escape a literal path (escapeRegExp) and guard with a negative lookahead for optional-slash-then-name-character — never split/join a bare prefix"
    - "Fail-loudly input contracts on gate-adjacent tools: validate the capture header, offer an explicit flag escape hatch (--no-header) rather than guessing"
    - "TDD RED evidence: per-target records {command, exitCode, output(TAP), targetTest, expected, actual} checked via gsd-tools check tdd-red-evidence"

key-files:
  created: []
  modified:
    - experiments/scrub-capture.mjs
    - experiments/scrub-capture.test.mjs

key-decisions:
  - "WR-03 uses the execution-verified 02-RESEARCH Pattern 7 form — escapeRegExp(home) + '(?!/?[A-Za-z0-9._-])' — deliberately NOT the 01-REVIEW hint (?!/[A-Za-z0-9._-]), which was proven by execution this session to still falsify the direct-suffix sibling (home+'x' -> ~x)"
  - "IN-04 header contract: line 1 must parse as a JSON object with an own 'capture' key; failure = one stderr line naming the file + contract + the --no-header escape hatch, exit 1; --no-header routes line 1 through scrubLine like any data line"
  - "git.allow_default_branch_commits:true encoded in .planning/config.json (Rule 3) — encodes 02-01's recommendation for the pre-protection window (plans 02-02..02-04; D-01 sequences protection LAST in 02-05)"

patterns-established:
  - "Sibling-path boundary test shape: build sibling = os.homedir() + 'x' + '/seg' dynamically, assert verbatim survival AND absence of the falsified '~x' rewrite"
  - "Header-guard test shape: assert non-zero exit + stderr naming the contract + no output written (null or empty)"

requirements-completed: [SCAFF-02]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "WR-03 boundary-aware home rewrite: sibling-of-home paths (home + name char, with or without trailing segment) pass through scrubbing verbatim; home/'rest' -> ~/rest and bare home -> tilde still work; scrubbing is idempotent"
    requirement: SCAFF-02
    verification:
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#WR-03: a sibling path sharing only the home prefix survives verbatim (home + name + /segment)"
        status: pass
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#WR-03: a bare sibling of the home directory survives verbatim (home + name)"
        status: pass
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#WR-03: the boundary rewrite is idempotent — scrubbing scrubbed sibling output is a fixed point"
        status: pass
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#absolute paths under the user home directory become tilde-relative"
        status: pass
    human_judgment: false
  - id: D2
    description: "IN-04 capture-header guard: non-JSON or capture-key-less line 1 exits non-zero with a clear stderr message; --no-header exits 0 and scrubs line 1 like a data line; empty input still yields empty output with exit 0"
    requirement: SCAFF-02
    verification:
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#IN-04: a first line that is not JSON fails loudly (non-zero exit, message names the problem)"
        status: pass
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#IN-04: a JSON first line without a capture key fails loudly"
        status: pass
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#IN-04: --no-header scrubs line 1 like a data line and exits 0"
        status: pass
      - kind: unit
        ref: "experiments/scrub-capture.test.mjs#empty input produces empty output and exit 0"
        status: pass
    human_judgment: false

# Metrics
duration: 8min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 02: Scrubber Input-Contract Hardening Summary

**Boundary-aware home rewrite (WR-03) + capture-header guard with --no-header (IN-04), proven RED->GREEN by 6 new node:test cases (suite 12 -> 18, all green)**

## Performance

- **Duration:** 8 min
- **Started:** 2026-11-19T13:49:27Z
- **Completed:** 2026-11-19T13:57:32Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- WR-03 closed: the scrubber's bare-home rewrite is boundary-aware — a path that merely shares the home prefix (sibling directory) passes through verbatim; the falsified `~x` rewrite is gone; home-anchored tilde rewrites and the empty-input contract are untouched
- IN-04 closed: malformed captures fail loudly (clear stderr naming the file + the capture-header contract, exit 1) instead of passing an unvalidated first line through; `--no-header` legitimately scrubs header-less raw captures
- RED->GREEN proven under TDD mode: 5 target tests captured in TAP evidence records, all classified RED_EVIDENCE_OK before the GREEN commit
- Suite grew 12 -> 18, fully green; `scan-secrets.sh` and `check-docs.sh` stay green — the scrubber contract is test-locked before plan 02-04 promotes the gate into CI (D-05 ordering)

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — failing tests for WR-03 sibling cases + IN-04 header guard** - `b886a32` (test)
2. **Task 2: GREEN — boundary-aware rewrite + header validation + --no-header** - `52f660a` (feat)

## Files Created/Modified
- `experiments/scrub-capture.mjs` - escapeRegExp helper + boundary-aware bare-home rewrite; capture-header validation on line 1; --no-header flag (argv loop + usage docs)
- `experiments/scrub-capture.test.mjs` - 6 new tests: 2 WR-03 sibling verbatim-survival, 1 WR-03 idempotency companion, 3 IN-04 (non-JSON fails, no-capture-key fails, --no-header passes + scrubs line 1)
- `.planning/phases/02-scaffold-ci-bootstrap/evidence/02-02-red-*.json` - 5 RED evidence records (RED_EVIDENCE_OK), persisted in-project

## TDD Gate Compliance

- **RED (commit `b886a32`):** `node --test --test-reporter tap experiments/` exit 1 — 18 tests discovered, 13 pass, 5 fail; every failure an AssertionError (testCodeFailure) on its planned assertion: sibling segment (`'~x/otherproj/file.txt'` vs verbatim `/home/nilx/otherproj/file.txt`), bare sibling (`'~x'` vs `/home/nilx`), IN-04 non-JSON (exit 0 !== non-zero), IN-04 no-capture-key (exit 0), IN-04 --no-header (exit 1 usage — flag parsed as positional). All 12 pre-existing tests green in the same run; the WR-03 idempotency companion passed (preservation lock, expected green in both phases).
- **Evidence:** 5 records at `.planning/phases/02-scaffold-ci-bootstrap/evidence/02-02-red-*.json`; `gsd-tools check tdd-red-evidence` returned `RED_EVIDENCE_OK` (reason `target_test_failed`) for each before any production edit.
- **semanticAssessment:** No fixture/load/usage crashes among failures — all five failed on the intended contract assertions (verified from the TAP failure blocks: `failureType: 'testCodeFailure'`, `code: 'ERR_ASSERTION'`).
- **GREEN (commit `52f660a`):** 18/18 pass; `bash experiments/scan-secrets.sh` and `bash experiments/check-docs.sh` exit 0.

## Decisions Made
- Used the 02-RESEARCH Pattern 7 corrected lookahead `(?!/?[A-Za-z0-9._-])` with an escapeRegExp helper, replacing via `.replace(..., 'g')` — not the 01-REVIEW hint form (execution-proven broken for direct-suffix siblings)
- IN-04 message names the file, the expected contract (JSON object with a `capture` key), and the `--no-header` escape hatch in one stderr line
- Encoded `git.allow_default_branch_commits: true` in `.planning/config.json` (see Deviations) to carry 02-01's recommendation forward

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] IN-04 invalid-input assertions expected an empty output string; the scrubber correctly writes no output file at all**
- **Found during:** Task 2 (GREEN) — first run after implementing the header guard
- **Issue:** `runScrub` returns `out === null` when the output file is never written; the two invalid-header tests asserted `out === ''` (the assertion the RED run never reached, since the exit-code assertion failed first)
- **Fix:** Assertions changed to `out === null || out === ''` ("no scrubbed output is written")
- **Files modified:** experiments/scrub-capture.test.mjs
- **Verification:** Suite 18/18 green
- **Committed in:** 52f660a (Task 2 commit)

**2. [Rule 3 - Blocking] Protected-branch commit guard would refuse the plan's commits on master**
- **Found during:** Task 1 commit (pre-commit HEAD safety assertion)
- **Issue:** `git.base-branch --is-protected master` = true and `git.allow_default_branch_commits` was unset, while the orchestrator dispatched this plan as a sequential executor committing directly to the main working tree (pre-protection window; plan 02-01 committed here identically)
- **Fix:** Set `git.allow_default_branch_commits: true` in `.planning/config.json` — the guard's documented override, and exactly what 02-01's SUMMARY recommended for plans 02-02..02-04 (D-01 sequences protection LAST in 02-05)
- **Files modified:** .planning/config.json
- **Verification:** Subsequent commits proceed; guard query returns the override
- **Committed in:** this plan's tracking/docs commit (committed with the SUMMARY)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes necessary for correctness of the committed contract and the mandated commit flow. No scope creep.

## Issues Encountered
None beyond the two deviations above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Scrubber contract is test-locked (18 green tests) — plan 02-03 can now align `scan-secrets.sh` pattern classes (WR-01) and re-run these verifies against the rewritten gate script
- Plan 02-04 promotes `scan-secrets.sh` into CI with the D-05 precondition (review fixes landed) satisfied for WR-03/IN-04
- RED evidence records remain in the phase `evidence/` dir for the verifier

---
*Phase: 02-scaffold-ci-bootstrap*
*Completed: 2026-11-19*
