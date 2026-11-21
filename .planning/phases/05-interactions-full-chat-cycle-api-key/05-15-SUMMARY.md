---
phase: 05-interactions-full-chat-cycle-api-key
plan: 15
subsystem: acpfront
tags: [permission-bridge, cancel-fan-in, session-scope, concurrency, tdd, zero-spend]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key (plans 05-02, 05-10)
    provides: the TOOL-02 permission bridge (pending table, -32021 cancelled arm), the CR-01 close-path fan-in, and the WR-04 pump-identity re-check this plan mirrors on the third arm
provides:
  - WR-06 closed red-first: the displaced-prompt arm fans the turn's pendings in at the displacement boundary — exactly one -32021 reply to the displaced bridge, no dangling unanswered note, exactly-once cancelled preserved (the 05-02 prohibition restored on the third and last cancel arm)
  - WR-07 closed deterministically: session-scoped pending table (map[sessionID]map[requestID]cancel) with atomic takes on all three arms — mine arm (take under the pump-identity re-check section), Front.Cancel (observe+take in ONE turnMu section), CloseSession (cancelPendingForSession; closing-gate safe)
  - cancelPendingForTurn deleted (promote-not-add-alongside); takePendingCancels/fireTakenPendings/cancelPendingForSession are the promoted surface; the discriminator log line preserved verbatim with its non-empty gate
  - The enshrining tests rewritten to demand the contract (assertions inverted, nothing weakened); the slot-free skip lock re-homed with full discriminating power (TestCancelFanInSkipAndScope/slot free)
  - Scope precision proven: the production-scoped diff since the 05-14-SUMMARY baseline names exactly the four declared files (scope-clean)
affects: [acpfront cancel/close paths, phase-05 verification (truth 7 re-pass), interact responder wiring, phase-7 hardening cycle (WR-05 remains open there)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 9089     # chars/4 over the realized internal/ diff vs the 05-14-SUMMARY baseline
  tasks: 2
  commits: 4       # MEASURED: git rev-list --count gsd-plan-head-before-05-15..HEAD

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "session-scoped pending table with atomic takes: the check (slot observation / pump-identity re-check) and the take run under ONE turnMu section; cancels fire outside any lock; one-way ordering turnMu → bridge.mu (T-5-38)"
    - "enshrining-test inversion: a test that locks a contract violation as expected is rewritten by inverting its assertions (dangle → fan-in), never by weakening survivors; the re-homed lock lands as a dedicated discriminating test in the same plan"

key-files:
  created:
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-15-red-t1-TestCancelFanInOnlyWhenSlotHeld.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-15-red-t2-TestCancelFanInSkipAndScope.json
  modified:
    - internal/acpfront/front.go
    - internal/acpfront/permission.go
    - internal/acpfront/lifecycle.go
    - internal/acpfront/cancel_test.go

key-decisions:
  - "WR-07 batched with WR-06 (per the disposition-ledger pattern): the WR-06 fix lands a new fan-in call site in exactly the files WR-07 touches (double churn otherwise); the sweep race is the same stale-sweep family the new mine arm would inherit at fire time — the atomic take-under-identity-lock closes both; 2 tasks, 4 files, inside one plan"
  - "Fake-tier displacement rehearsal SKIPPED, justified: the displacement is an ACP-client-side behavior (the SDK's second-prompt registration cancels prompt 1's ctx) existing ABOVE the back-leg wire the fake tier models — no corpus row can host the shape without force-fitting a fake-engine extension for client behavior the fake never observes; the front tier drives the shape deterministically through the REAL SDK client (rewritten + new locks), and the golden corpus replay in the gate proves zero drift"
  - "Promote-not-add-alongside (assumption_delta_decision): the front-wide cancelPendingForTurn is deleted, not kept beside the session-scoped sweep — keeping both would leave an unsound path callable"
  - "The RED is an ENSHRINING-TEST INVERSION (stated in both the test doc comment and the RED commit): the prior TestCancelFanInOnlyWhenSlotHeld locked the dangling bridge as the expected artifact; both WR-06 and verifier gap 3 classified it a contract violation — the rewrite inverts dangle assertions into fan-in assertions and keeps every surviving lock (exactly-once cancelled success, single send, no agent-sent -32800); assertDanglingAskNote deleted; the dropped tail's slot-free lock re-homed with full discriminating power in TestCancelFanInSkipAndScope"
  - "REQUIREMENTS.md untouched (plan prohibition): TOOL-02/CORE-06 flips and the truth-7 flip remain the orchestrator's post-verification acts after the verifier's next pass"

patterns-established:
  - "Atomic check-and-take fan-in: the pendings take rides the same turnMu section as the identity/slot observation; fireTakenPendings runs strictly after unlock (outside any lock) — the pattern all three cancel arms now share"
  - "Bounded settle before a single transcript snapshot: the fake materializes its transcript only at Serve return, so the settle is keyed to the corpus row's own reverse deadline (backLegSettle 700ms > 500ms) making the snapshot final either way"

requirements-completed: [TOOL-02, CORE-06]  # copied from the plan frontmatter — REQUIREMENTS.md itself stays untouched per the plan prohibition (orchestrator's post-verification act)

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The displaced-prompt arm fans the turn's pendings in: exactly one -32021 reply to the displaced bridge, no dangling unanswered note, exactly-once cancelled preserved (WR-06 closed)"
    requirement: TOOL-02
    verification:
      - kind: unit
        ref: "internal/acpfront/cancel_test.go#TestCancelFanInOnlyWhenSlotHeld"
        status: pass
      - kind: unit
        ref: "gsd_run check tdd-red-evidence → RED_EVIDENCE_OK (target_test_failed: 0 reply envelopes + unanswered note present at HEAD)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The cancel fan-in is session-scoped with atomic takes on all three arms; a foreign session's in-flight ask survives a cancel of another session (WR-07 closed deterministically)"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/acpfront/cancel_test.go#TestCancelFanInSkipAndScope/session_scope_spares_foreign_sessions"
        status: pass
      - kind: unit
        ref: "gsd_run check tdd-red-evidence → RED_EVIDENCE_OK (target_test_failed: foreign round-trip cancelled by the front-wide sweep at Task-1-GREEN HEAD)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A round-trip parked with NO active turn survives front.Cancel (the WR-04/CR-01 slot-gate preserved with full discriminating power — re-homed lock)"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/acpfront/cancel_test.go#TestCancelFanInSkipAndScope/slot_free_spares_parked_pendings"
        status: pass
    human_judgment: false
  - id: D4
    description: "Close-path and ACP-cancel locks stay green untouched (TestCloseWithPendingPermissionCancelled, TestCloseDeadlineArmFansPendingIn, TestPermissionCancel, TestPermissionExpiry, TestCancelFanInStaleArmSkips, TestFullCycle)"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "command: go test -race -count=1 ./internal/acpfront/"
        status: pass
    human_judgment: false
  - id: D5
    description: "Full gates green: acpfront -race, fake golden corpus replay, vet/build, mise run lint 0 issues; production-scoped scope gate prints scope-clean (exactly the four declared files under internal/ since the 05-14-SUMMARY baseline)"
    requirement: CORE-06
    verification:
      - kind: other
        ref: "command: go test -race -count=1 ./internal/acpfront/ && go test -run 'TestGolden|TestEOFSignature' ./internal/zcode/fake/ && go vet ./... && go build ./... && mise run lint && scope-gate (git diff BASE..HEAD -- internal/)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Zero real-account spend: verification is fake-tier + unit rehearsal only — no realdesktop execution, no new real-run checkpoint (attempt 3's PASS covers the field; the verifier re-runs after execution)"
    requirement: CORE-06
    verification:
      - kind: other
        ref: "command: scope gate (no test/realdesktop path in the plan's diff) + the fake-tier/unit gate chain above"
        status: pass
    human_judgment: false

# Metrics
duration: 26min
completed: 2026-11-22
status: complete
---

# Phase 05 Plan 15: WR-06 Displaced-Prompt Fan-In + WR-07 Session-Scoped Pendings Summary

**The displaced turn's permission bridge now fans in at the displacement boundary — exactly one -32021 reply, no dangling ask — and the fan-in is session-scoped with atomic takes on all three cancel arms (WR-06 + WR-07 closed red-first, zero spend).**

## Performance

- **Duration:** 26 min
- **Started:** 2026-11-21T21:03:28Z
- **Completed:** 2026-11-21T21:29:19Z
- **Tasks:** 2 (both TDD: RED commit → GREEN commit each)
- **Files modified:** 4 production/test files (+ 2 RED evidence records)

## Accomplishments
- WR-06 closed: the fan-in goroutine's mine arm takes the displaced turn's session pendings under the SAME turnMu section as the WR-04 pump-identity re-check (check-and-take atomic), fires machine.Cancel then the cancels outside any lock — the back leg observes exactly one -32021 reply envelope for the displaced ask and no unanswered-at note; the displaced prompt still resolves exactly once as the SUCCESS cancelled stopReason.
- WR-07 closed: the pending table is session-scoped (map[sessionID]map[requestID]cancel); all three fan-in arms take atomically — mine arm (identity re-check section), Front.Cancel (slotHeld observation + take in ONE turnMu section, closing the review's deschedule window by construction), CloseSession (cancelPendingForSession — the closing gate makes an atomic take unnecessary there); cancelPendingForTurn is deleted; the discriminator log line survives verbatim with its non-empty gate.
- The enshrining tests now demand the contract: TestCancelFanInOnlyWhenSlotHeld's dangle assertions inverted into fan-in assertions (assertDanglingAskNote deleted); the slot-free skip lock re-homed as TestCancelFanInSkipAndScope/slot free with a pending genuinely parked while the slot is free; the foreign-session scope lock added as TestCancelFanInSkipAndScope/session scope.
- Full gates green and scope-clean: the production-scoped diff since the 05-14-SUMMARY baseline names exactly internal/acpfront/{front,permission,lifecycle,cancel_test}.go.

## Task Commits

Each task committed atomically (RED → GREEN per task):

1. **Task 1: WR-06 red-first — enshrining rewrite (RED) → mine-arm fan-in (GREEN)**
   - RED `a7f67f9` (test(05-15): test file only — inverted assertions, evidence validated `RED_EVIDENCE_OK`)
   - GREEN `7f18358` (feat(05-15): front.go + permission.go — transient front-wide semantics recorded in the commit)
2. **Task 2: WR-07 batch — session-scoped table + atomic takes (RED → GREEN)**
   - RED `b358cfa` (test(05-15): TestCancelFanInSkipAndScope — evidence validated `RED_EVIDENCE_OK`)
   - GREEN `8add9e6` (feat(05-15): permission.go + front.go + lifecycle.go + cancel_test.go lint discipline)

**Plan metadata:** (this commit) docs(05-15).

## RED Evidence (TDD gate sequence)

- **Task 1 RED at HEAD** (validated `gsd_run check tdd-red-evidence` → `RED_EVIDENCE_OK`, reason `target_test_failed`): `--- FAIL: TestCancelFanInOnlyWhenSlotHeld — the displaced ask earned 0 reply envelopes within the bound, want exactly 1 (-32021 cancelled) — the mine arm omitted the pendings fan-in (WR-06), so the pending bridge dangled; unanswered-at note present on the transcript: true`. The genuine missing-fan-in signal; re-verified with the committed (amended) test against the pre-fix production files checked out from the RED commit's parent — identical failure line.
- **Task 2 RED at Task-1-GREEN HEAD** (validated `RED_EVIDENCE_OK`): `--- FAIL: TestCancelFanInSkipAndScope/session_scope_spares_foreign_sessions — the FOREIGN session's pending round-trip read cancelled — the fan-in swept past its session (the front-wide table, WR-07)`. The sibling slot-free subtest passed at HEAD as planned (it is the re-homed skip lock, not a RED target).
- Records: `.planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-15-red-t1-TestCancelFanInOnlyWhenSlotHeld.json`, `.../05-15-red-t2-TestCancelFanInSkipAndScope.json`.

## TDD Gate Compliance

| Plan | RED | GREEN | REFACTOR | Status |
|------|-----|-------|----------|--------|
| 05-15 Task 1 | ✓ a7f67f9 | ✓ 7f18358 | — | Pass |
| 05-15 Task 2 | ✓ b358cfa | ✓ 8add9e6 | — | Pass |

Both RED commits precede their GREEN commits and carry validated `RED_EVIDENCE_OK` evidence; both GREEN commits descend from their RED commits. No violations.

## Files Created/Modified
- `internal/acpfront/permission.go` — session-scoped pending table; takePendingCancels(sessionID), fireTakenPendings (verbatim log, non-empty gate), cancelPendingForSession; cancelPendingForTurn deleted; one-way turnMu → bridge.mu ordering documented (T-5-38)
- `internal/acpfront/front.go` — mine-arm fan-in (take under the identity re-check section, fires outside any lock) extracted into watchCancelFanIn; Front.Cancel's atomic observe+take section
- `internal/acpfront/lifecycle.go` — CloseSession's sawTurn-gated fan-in → cancelPendingForSession(sessionID); drain/gate order unchanged
- `internal/acpfront/cancel_test.go` — TestCancelFanInOnlyWhenSlotHeld rewritten (contract assertions); assertDanglingAskNote deleted; TestCancelFanInSkipAndScope added (session-scope + slot-free subtests); safeLogBuffer.count; scoped -32800 lock

## Decisions Made

See key-decisions: WR-07 batching, fake-tier rehearsal skip (with justification), promote-not-add-alongside, enshrining-inversion legitimacy, REQUIREMENTS.md left untouched.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Test-shape bug] The blanket no--32800 scan contradicted the honest cancellation artifact**
- **Found during:** Task 1 GREEN verification (the fix worked — the -32021 assertions passed — but the scan failed)
- **Issue:** The plan's behavior bullet "no -32800 anywhere on the ACP tee" taken as a blanket scan flags a legitimate frame: when the fan-in cancels the displaced turn's pending bridge, the session/request_permission round-trip dies mid-flight and the client-side SDK answers the cancelled request with its OWN -32800 — the honest client-side mirror of the -32021 arm, present exactly when the fan-in fires. The D-05/Pitfall-2 contract governs the AGENT's answers (the prompt must never surface cancellation as a RequestCancelled ERROR), not the counterparty's reply to its own cancelled request.
- **Fix:** Scoped the lock to the agent→client direction (new `assertNoAgentRequestCancelled` helper, doc comment stating the exemption); the blanket `assertNoRequestCancelled` stays for flows with no pending ask. The prompt-answer shape remains fully locked (exactly-one success response + the scoped scan).
- **Files modified:** internal/acpfront/cancel_test.go
- **Verification:** TestCancelFanInOnlyWhenSlotHeld green under -race; the RED property re-verified against the pre-fix production state (identical failure line)
- **Committed in:** a7f67f9 (amended RED commit — the assertion scope was test-design, so it belongs to the RED commit; local-only commit, no rewrite of pushed history)

**2. [Rule 3 - Blocking] Strict-lint gate failures on the new code (cyclop/funlen/gofmt/unused/revive/funcorder/gocritic)**
- **Found during:** Task 2 full gate (`mise run lint`)
- **Issue:** The GREEN changes tripped the strict linter: Prompt's cyclomatic complexity 16 > 15 (the added take branch), testCancelFanInSessionScope 80 > 60 lines (funlen), gofmt formatting, unused safeLogBuffer.contains (its last caller was the dropped enshrining tail), context-as-argument order, funcorder method placement, gocritic unnamedResult.
- **Fix:** Behavior-identical refactors only: extracted the fan-in goroutine body into `watchCancelFanIn` (Prompt's complexity back under the cap; doc comment carries the WR-04/WR-06/T-5-38 contract text); extracted `parkForeignPermissionRoundTrip` + `assertDisplacedFanInBackLeg` helpers; deleted the unused `contains` (count is the surviving discriminator reader); gofmt; ctx-first param; named results; method moved after Cancel. No test semantics changed — the suite re-ran green after each step.
- **Files modified:** internal/acpfront/front.go, internal/acpfront/cancel_test.go
- **Verification:** mise run lint → 0 issues; full acpfront -race suite green
- **Committed in:** 8add9e6 (Task 2 GREEN)

---

**Total deviations:** 2 auto-fixed (1 test-shape bug Rule 1, 1 blocking lint Rule 3). **Impact on plan:** Both fixes necessary (contract-faithful assertion scoping; the plan's own lint gate). No scope creep — the production shape is exactly the plan's.

## Issues Encountered
- An execution incident, resolved with zero code impact: the post-hoc RED re-verification step checked out production files from the RED commit's parent to prove the amended test still fails genuinely — the subsequent restore (`git checkout HEAD --`) was issued while the GREEN production edits were still uncommitted, briefly wiping them from the working tree. Both edits were re-applied identically from the verified state and the full suite re-ran green before the GREEN commit; the committed content is the verified content.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The 05-02 no-dangling-bridge invariant now holds on all three cancel arms (close, ACP cancel, SDK displacement); truth 7's failure cause is fixed and unit-locked — ready for the verifier's 13/13 re-pass (zero new spend required; attempt 3's field PASS stands).
- WR-05 (resume TOCTOU) intentionally remains open in the disposition ledger — advisory, separate arm, feeds the hardening cycle.
- REQUIREMENTS.md (AUTH-01) and the truth-7 flip remain the orchestrator's post-verification acts.

## Self-Check: PASSED

All seven claimed files exist on disk; all five commit hashes (a7f67f9, 7f18358, b358cfa, 8add9e6, 8c0cbd5) verified as ancestors of HEAD. Both RED evidence records re-validated through `gsd_run check tdd-red-evidence` (RED_EVIDENCE_OK × 2). Full gate re-ran green on the committed tree: acpfront -race ok, fake corpus ok, vet/build ok, lint 0 issues, scope gate scope-clean.

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-22*
