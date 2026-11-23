---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 02
subsystem: session-lifecycle
tags: [hostsvc, interact, acpport, snapshot, keyringprobe, d08-advisory, in01-single-read, in03-classify-order, tdd]

# Dependency graph
requires:
  - phase: 06-oauth-coding-plan-path-model-picker
    provides: the AUTH-02 account snapshot (BuildAccount + the fixture vectors), the AUTH-04 keyring probe (classify + the four ErrKeyring* sentinels), the SESS-01 resume delegation (acpport adapter), and the routed IN-01/IN-02/IN-03 findings (06-REVIEW-DISPOSITION.md)
provides:
  - IN-01 decodeBuiltinTable: BuildAccount derives the builtin revision AND the provider rules from ONE read+decode of install.BuiltinFile — a table swapped between two reads can never produce a stale revision/rules pairing (T-07-05 mitigated); the absorbed rules reader's coded classes are byte-unchanged
  - IN-03 classify reorder: parent-cancel -> success -> deadline -> marker -> default — a subprocess succeeding exactly at the deadline boundary classifies ok, never a false timeout (T-07-06 mitigated); the four ErrKeyring* sentinel strings are byte-untouched (T-07-07 guard held, QA-02's VM-script assertion targets intact)
  - IN-02 ResumeSession adapter comment stating the real contract (delegate-and-discard; the front owns validation via createSessionID) — comment-only, zero code motion
  - the in-package classify-order test tier (keyringprobe_classify_internal_test.go) asserting the unexported switch order directly, with the persisted RED evidence record
affects: [07-03, 07-04, 07-05 (SESS-03 session-compat arms ride the hardened snapshot/probe surfaces), 07-07 (the VM script asserts the byte-frozen sentinels), phase-7 review]

# Actuals (#2632) — pairs with the plan's `estimate` (38000 tokens) to calibrate future estimates.
actuals:
  tokens: 4623    # chars/4 over the realized diff (18492 chars, 5 files, +225/-35)
  tasks: 3
  commits: 5
  plan_head_before: 1be8fe3c0673fec652f795d9762e858ba5e2320c
  plan_head_after: 611eb313de214e8f9a815a6afaf541b856a72db6

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "single-read discipline for paired derived values: one read+decode helper (decodeBuiltinTable) feeds every value derived from the same live file, so an external swap between reads cannot pair fresh-with-stale"
    - "classification arm order as a tested contract: the coarsest-signal-first ordering (parent-cancel -> success -> deadline -> marker -> default) is asserted directly on the unexported switch by in-package tests"

key-files:
  created:
    - internal/interact/keyringprobe_classify_internal_test.go
    - .planning/tdd/07-02-red-evidence.json
  modified:
    - internal/hostsvc/snapshot.go
    - internal/interact/keyringprobe.go
    - internal/acpport/agent.go

key-decisions:
  - "readBuiltinRevision stays a standalone revision-only reader serving Build (the empty-overlay builder, untouched): a shared full-table decode would make the revision-only path reject the lenient revision-only fixtures the builtinfile vectors lock (e.g. {\"revision\":44,\"note\":\"recorded\"} has no providerRules)"
  - "readBuiltinProviderRules is absorbed into decodeBuiltinTable (its only caller was BuildAccount); every coded error class and free-text format the vectors observe is byte-unchanged — only the never-asserted read-failure wrap reworded to 'read builtin table %s'"
  - "the IN-03 regression arms landed in a new in-package file rather than keyringprobe_test.go: that file is package interact_test and cannot read the unexported classify/probeClass* the plan's behavior block names; the 07-01 *_internal_test.go precedent applies"
  - "task 3's verify pattern -run 'TestAdapter' matches zero tests (no such names in acpport); the full package under -race was run instead — a strictly stronger fulfillment of the stated intent"

patterns-established:
  - "one read+decode per build: BuildAccount's revision and rules are one decodeBuiltinTable call — the doc contract that already claimed the SAME file is now literally the same read"
  - "classify order = parent-cancel, success, deadline, marker, default: a boundary-timed success is never a false timeout; cancelled-parent remains authoritative over a success-shaped verdict"
  - "in-package *_internal_test.go files for locks that read unexported switch arms (continuing the 07-01 tier)"

requirements-completed: []

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "IN-01: BuildAccount derives the builtin revision AND the provider rules from ONE read+decode of install.BuiltinFile (decodeBuiltinTable) — no stale revision/rules pairing possible; the empty-overlay Build builder, the coded no-rules error, and the gate-may-only-omit rule are untouched; every existing account/builtinfile vector passes with no fixture edits (the second-read grep gate prints 0)"
    requirement: SESS-03
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotVerbatimMirror (+ GateOmission/NoRules/MalformedTable/PushEnvelopeCompat)"
        status: pass
      - kind: unit
        ref: "internal/hostsvc/snapshot_builtinfile_test.go#TestAccountBuildUsesRecordedBuiltinFile"
        status: pass
      - kind: other
        ref: "grep -c 'readBuiltinProviderRules(install.BuiltinFile)' internal/hostsvc/snapshot.go -> 0 (EXPECT==0 gate)"
        status: pass
    human_judgment: false
  - id: D2
    description: "IN-03: a keyring lookup succeeding exactly at the deadline boundary (execErr nil, non-blank stdout, the probe's bounded ctx expired) classifies ok with nil error; parent-cancel still wins first, timeout-on-empty still classifies timeout, marker and default arms unchanged; the four ErrKeyring* sentinel declarations are byte-identical across the plan's commits (T-07-07 guard)"
    requirement: SESS-03
    verification:
      - kind: unit
        ref: "internal/interact/keyringprobe_classify_internal_test.go#TestKeyringProbeClassifyDeadlineBoundarySuccess (+ ParentCancelWinsFirst/TimeoutOnEmptyStdout/MarkerAndDefaultArms)"
        status: pass
      - kind: unit
        ref: "internal/interact/keyringprobe_test.go#TestKeyringProbe (the six public-seam arm contracts, unchanged)"
        status: pass
      - kind: other
        ref: "grep -c 'keyring probe: secret-tool not found' internal/interact/keyringprobe.go -> 1 (>=1 sentinel-intact gate); git diff 1be8fe3..HEAD shows no sentinel-declaration line"
        status: pass
    human_judgment: false
  - id: D3
    description: "IN-02: the ResumeSession adapter comment states the real contract — the adapter delegates and discards the Handlers-returned id (the SDK resume response carries no session id; the reattached id IS the request's own), the front owns input validation and derives the id via createSessionID — the stale 'validated non-empty' claim is gone and the diff is comment lines exclusively"
    requirement: SESS-03
    verification:
      - kind: other
        ref: "grep -c 'validated non-empty' internal/acpport/agent.go -> 0 (EXPECT==0 gate); git diff -U0 shows only comment/blank lines"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/acpport/ (the full adapter delegation suite)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Plan-level verification: hostsvc + interact + acpport all green under -race, both grep gates print 0, the sentinel block diffs empty, and the whole-tree lint gate is green at wave close"
    verification:
      - kind: other
        ref: "go test -race -count=1 ./internal/hostsvc/... ./internal/interact/... ./internal/acpport/... (exit 0) + mise run lint (0 issues)"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-11-22
status: complete
---

# Phase 7 Plan 02: D-08 Advisory Batch (IN-01/IN-03/IN-02) Summary

**BuildAccount now derives revision and rules from ONE read+decode of the live builtin table (no stale pairing), the keyring probe classifies a boundary-timed success as ok instead of a false timeout, and the ResumeSession adapter comment states the real delegate-and-discard contract — with the four keyring sentinels byte-frozen and -race plus whole-tree lint green.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-11-22T11:38:47Z
- **Completed:** 2026-11-22T11:50:33Z
- **Tasks:** 3 (1 tracer + 1 full RED→GREEN TDD + 1 comment-only)
- **Files modified:** 5 (+225/-35; 2 production files, 1 comment-only file, 1 test file created, 1 evidence record)

## Accomplishments

- **IN-01 (Task 1, tracer):** `decodeBuiltinTable(path)` reads `install.BuiltinFile` ONCE and unmarshals ONCE into both the revision number and the account-provider rules; `BuildAccount` derives rev+rules from that single decode, so an auto-update swapping the table mid-build can never pair a fresh `basedOnZCodeBuiltinRevision` with stale rules (T-07-05). `readBuiltinProviderRules` is absorbed; its coded classes (`errMissingProviderRules`, the per-rule dig errors) and the coded no-rules error + gate-may-only-omit rule are byte-unchanged. `Build` and `readBuiltinRevision` are untouched (the revision-only path must stay lenient for the builtinfile vectors). The rules-reader doc comment that already claimed the SAME file is now literally the same read. The tracer feedback gate re-ran `<verify>` end-to-end in auto mode: `go test -race ./internal/hostsvc/` green, second-read grep gate 0.
- **IN-03 (Task 2, RED→GREEN):** RED `TestKeyringProbeClassifyDeadlineBoundarySuccess` drove the unexported `classify` with execErr nil, non-blank stdout, and the bounded ctx already past its deadline — the current switch answered `timeout` with the `ErrKeyringLookupTimeout` wrap (intentional failure on the planned assertion; classifier `RED_EVIDENCE_OK target_test_failed`). GREEN reordered the switch to parent-cancel → success → deadline → marker → default, so a subprocess succeeding exactly at the boundary classifies `ok` (T-07-06) while a cancelled PARENT stays authoritative and the genuine never-hangs timeout arm keeps empty-answer lookups. The four `ErrKeyring*` sentinel declarations are byte-identical across the plan's commits (T-07-07 — QA-02's VM-script assertion targets) and the probe timeout value is untouched.
- **IN-02 (Task 3):** the `ResumeSession` doc comment now states the real contract — the adapter delegates and discards the Handlers-returned id because the SDK resume response carries no session id (the reattached id IS the request's own), the front validates the inputs (empty sessionId / non-absolute cwd) and derives the id via `createSessionID`, refusals coded via `wireError`. The claimed "validated non-empty" check is gone from the file; `git diff -U0` confirms comment lines exclusively — zero code motion.
- **Plan-level verification:** `go test -race -count=1 ./internal/hostsvc/... ./internal/interact/... ./internal/acpport/...` all green; both grep gates print 0; the sentinel block diffs empty; `mise run lint` 0 issues at wave close.

## Task Commits

1. **Task 1 (tracer): IN-01 one builtin read feeds both revision and rules** — `e888605` (refactor)
2. **Task 2: IN-03 deadline-boundary success classifies ok** — RED `0f423e3` (test), GREEN `98a6f84` (feat)
3. **Task 3: IN-02 ResumeSession comment matches the code** — `b5652ce` (docs)
4. **Lint fix** — `611eb31` (style: wrap the classify lock's over-length Fatalf line)

**Plan metadata:** this SUMMARY + STATE/ROADMAP/REQUIREMENTS ride the final docs commit.

_Note: Task 1 is a tracer (behavior-invisible refactor locked by the existing vectors — no RED required); Task 2 is the plan's only tdd task._

## TDD Gate Compliance

| Task | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| 1 (tracer) | n/a (refactor locked by existing vectors) | `e888605` | — | n/a |
| 2 | `0f423e3` (`RED_EVIDENCE_OK` target_test_failed) | `98a6f84` | `611eb31` (style only, tests stayed green) | Pass |
| 3 | n/a (comment-only) | `b5652ce` | — | n/a |

- Task 2's RED is genuine: the target test executed and failed on the planned IN-03 assertion (`classified "timeout" — want "ok" with nil error`) for the intended reason (the deadline arm won over a boundary-timed success); evidence persisted in `.planning/tdd/07-02-red-evidence.json` with the semantic assessment. Gate sequence in git log: `feat(7-02)` descends from `test(7-02)`.
- `mise run lint` flagged one over-length line in the new RED test after the GREEN commit; fixed in `611eb31` with the test staying green (repo precedent: 07-01's style commit).

## Files Created/Modified

- `internal/hostsvc/snapshot.go` — `decodeBuiltinTable` (one read+decode feeding revision+rules), `BuildAccount` rewired to the single decode, `readBuiltinProviderRules` absorbed with its coded classes preserved
- `internal/interact/keyringprobe.go` — the classify switch reordered (parent-cancel → success → deadline → marker → default) with the IN-03 rationale documented at the switch; sentinels and timeout byte-untouched
- `internal/acpport/agent.go` — ResumeSession doc comment corrected (comment-only; the stale non-empty-validation claim removed)
- `internal/interact/keyringprobe_classify_internal_test.go` (NEW) — the four classify-order locks: boundary success (RED arm), parent-cancel-first, timeout-on-empty, marker + default arms
- `.planning/tdd/07-02-red-evidence.json` (NEW) — the persisted RED evidence record (command, exit code, output, target, expected/actual, semanticAssessment)

## Decisions Made

- **`readBuiltinRevision` stays standalone rather than becoming a wrapper.** A shared full-table decode would make the revision-only `Build` path reject the lenient revision-only fixtures (`{"revision":44,"note":"recorded"}` has no providerRules) that `snapshot_builtinfile_test.go` locks — the vectors-must-pass-unchanged constraint decides. The plan explicitly allows "absorbed" for the rules reader (its only caller was BuildAccount).
- **The absorbed reader's read-failure wrap reworded to `read builtin table %s`.** No test asserts that free text (verified by grep over both fixture files); every coded class (`errMissingProviderRules`, dig errors) and every assertable surface is byte-unchanged.
- **The IN-03 tests landed in a new in-package file.** The plan's behavior block names the unexported `classify`/`probeClass*` directly, but the targeted `keyringprobe_test.go` is `package interact_test` and cannot read them; `keyringprobe_classify_internal_test.go` (`package interact`) follows the 07-01 `*_internal_test.go` precedent. The external file is untouched.
- **Task 3 ran the full acpport package** instead of the plan's `-run 'TestAdapter'` pattern, which matches zero test names in the package — the stated intent ("adapter tests pass") is fulfilled strictly more strongly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The plan-targeted classify tests cannot live in keyringprobe_test.go (external package)**
- **Found during:** Task 2 RED (before writing tests)
- **Issue:** the plan's behavior block and action mandate calling the unexported `classify` and asserting `probeClass*` values, but `keyringprobe_test.go` is `package interact_test` — in-package access is impossible there
- **Fix:** the four classify-order tests landed in a new in-package `keyringprobe_classify_internal_test.go` (the 07-01 naming precedent), leaving `keyringprobe_test.go` byte-untouched
- **Files modified:** internal/interact/keyringprobe_classify_internal_test.go (NEW)
- **Verification:** `go vet` clean; the tests run in the package's normal `go test` invocation; lint green after 611eb31
- **Committed in:** 0f423e3 (the RED commit)

**2. [Rule 3 - Blocking] Task 3's verify pattern matches zero tests**
- **Found during:** Task 3 verification
- **Issue:** `-run 'TestAdapter'` selects nothing — acpport has no `TestAdapter*` names (the adapter suite is the Delegation family et al.)
- **Fix:** ran the full package `go test -race -count=1 ./internal/acpport/` (exit 0) — the stated intent, strictly stronger
- **Files modified:** none (verification substitution only)
- **Verification:** full acpport package green under -race
- **Committed in:** b5652ce (the docs commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 3 blocking, both verification/file-placement mechanics) + 1 style follow-up (`611eb31`, lll wrap on the new test — lint debt, not a plan deviation).
**Impact on plan:** both deviations preserve every planned assertion and gate; no production code moved beyond the plan's prescriptions. No scope creep.

## Issues Encountered

- `mise run lint` reported one lll (120) violation in the new test file after the GREEN commit; resolved in `611eb31` by wrapping the Fatalf message — tests stayed green, tree back to 0 issues.
- The plan's estimate (38000 tokens) overshot the realized diff (~4600 tokens at chars/4): the batch is three small mechanical fixes; recorded honestly in `actuals`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The D-08 advisory dispositions IN-01/IN-02/IN-03 in `06-REVIEW-DISPOSITION.md` are flip-ready: landing commits `e888605` (IN-01), `98a6f84` (IN-03), `b5652ce` (IN-02).
- The four keyring sentinel strings are byte-frozen at this commit — 07-07's VM script can assert them verbatim.
- The snapshot/probe/adapter surfaces this batch hardened are exactly the surfaces 07-03/07-04/07-05 (SESS-02/SESS-03 session-compat arms) compose on top of.

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*

## Self-Check: PASSED

- All six created/modified files exist on disk (SUMMARY, classify-order test file, RED evidence record, snapshot.go, keyringprobe.go, agent.go).
- All six commits are ancestors of HEAD: `e888605` (IN-01 refactor), `0f423e3` (IN-03 RED), `98a6f84` (IN-03 GREEN), `b5652ce` (IN-02 docs), `611eb31` (style), `86d4a00` (plan metadata).
- Measured commit count from the plan ledger `1be8fe3..HEAD` matches frontmatter `commits: 5` (code commits; 86d4a00 is the metadata commit).
- Both grep gates re-verified 0; the sentinel block diffs empty across the plan's commits; `mise run lint` 0 issues.
