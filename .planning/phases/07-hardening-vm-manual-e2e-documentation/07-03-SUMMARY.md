---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 03
subsystem: acp-session-compat
tags: [acp, json-rpc, session-delete, session-fork, decline-surface, capability-advertisement, composed-harness, tdd]

# Dependency graph
requires:
  - phase: 07-hardening (plan 01)
    provides: the 07-01 teardown contract (CloseSession cancel-drain -> backend close -> registry release -> observer close -> map drops; teardownAfterPickerRefusal, registerIfOpen) the delete arm reuses
  - phase: 07-hardening (plan 02)
    provides: the advisory batch baseline (single builtin read, probe success-arm-before-deadline) the tree rides
  - phase: 05-interactions
    provides: the composed harness (ComposeFront/TeeRW/fullcycle helpers) every sess-compat row rides; the SESS-01 lifecycle pair whose delegation shape the delete arm copies
provides:
  - session/delete as a Handlers delegate: registered id -> the full SESS-01 teardown answering the dialect's empty result {}; unknown/already-closed/empty id -> the spec's silent-success {} with ZERO backend frames ("session/delete" x0 in the bundle — nothing to delete)
  - the explicit UnstableForkSession decline arm on agentAdapter — byte-identical to the SDK dispatcher's type-assert default (-32601 "Method not found" + {"method":"session/fork"}), drift-proof, testable without the default path
  - the D-02 decline surface byte-locked by composed-tier rows: set_mode/list/load typed declines + the raw-frame session/set_model row (SDK connection fallback; no dispatch case in schema 1.25.0)
  - the T-07-08 advertisement honesty guard: sessionCapabilities stays options-only (D-11); delete/fork/list/load/resume/close never advertised; mutation-verified (a temporary Delete advertisement flips it red)
  - the delete delegation contract at the adapter tier (TestDeleteSessionDelegation: id verbatim, {} result, coded-error passthrough via wireError)
affects: [07-05 /model slash prompt (rides the same front decline/error surface), 07-07 conformance suite (composes the sess-compat arms), 07-09 manual E2E (delete/fork never fire from Orca — unadvertised)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 11325      # chars/4 over the realized diff (45299 diff chars)
  tasks: 3
  commits: 5         # MEASURED: git rev-list --count e7eb577..HEAD
plan_head_before: e7eb577ad3742ba4b24a64a8d803b2eae23c4c6b
plan_head_after: 3c2a10ab8f9720d467effb3510833f96e4bd812f

# Tech tracking
tech-stack:
  added: []          # no new dependencies — the SDK pin untouched, tidy-diff gate holds
  patterns:
    - "silent-success arms: a registry.Get miss IS the arm's trigger, never an error to surface (nilerr-nolinted with contract rationale)"
    - "explicit optional-interface arms: implementing the SDK optional interface with unadvertised(method) keeps the dispatcher default's bytes while making the posture self-documenting"
    - "decline-frame byte-locking: sessTeeReply (id-matched, method-key-tolerant) + substring constants for code/message/data.method — never paraphrased"
    - "guard self-verification: mutation-test the guard (temporary capability advertisement -> red) then restore"

key-files:
  created:
    - internal/acpfront/sesscompat_test.go
    - .planning/tdd/07-03-task1-red.json
    - .planning/tdd/07-03-task2-red.json
    - .planning/tdd/07-03-task2-confirm-green.json
  modified:
    - internal/acpport/agent.go
    - internal/acpport/types.go
    - internal/acpport/agent_test.go
    - internal/acpfront/front.go

key-decisions:
  - "delete-unknown RED landed CONFIRM-GREEN honestly: Task 1's action text implemented BOTH front arms, so Task 2's delete-unknown row passed at entry — the classifier verdict (INVALID_RED unexpected_green) persisted verbatim in 07-03-task2-confirm-green.json"
  - "the fork GREEN's genuine RED is the type-assert row TestForkArmIsExplicit (adapter must implement AgentUnstableForkSession explicitly), not a wire-shape change — the decline bytes stay the SDK default's by construction"
  - "close-advertisement DECIDED AGAINST for v1: close is implemented since SESS-01 but Orca never checks the field; advertising would churn every initialize byte-lock for zero v1 benefit — revisit on demand"
  - "sessTeeReply replaces authTeeResponse for decline rows: the decline frame's data.method key violates authTeeResponse's method-free-reply filter"

patterns-established:
  - "Pattern: silent-success delete arm — registry miss answers {} immediately with zero backend frames; registered id delegates to the SESS-01 teardown"
  - "Pattern: explicit decline arms for unadvertised optional interfaces (UnstableForkSession) — byte-identical to SDK defaults, grep-locatable"
  - "Pattern: composed-tier decline rows with computed data.method substring locks (assertSessDeclineFrame)"

requirements-completed: [SESS-02, SESS-03]

coverage:
  - id: D1
    description: "session/delete on a REGISTERED id answers the dialect's empty result {} and runs the full SESS-01 teardown (registry release, settings drop, observer removal, backend close, zero send traffic)"
    requirement: SESS-02
    verification:
      - kind: integration
        ref: "internal/acpfront/sesscompat_test.go#TestSessionDeleteRegisteredRunsTeardown"
        status: pass
    human_judgment: false
  - id: D2
    description: "session/delete on an UNKNOWN, already-closed, or empty id answers {} immediately with zero backend frames (the spec's silent-success arm)"
    requirement: SESS-02
    verification:
      - kind: integration
        ref: "internal/acpfront/sesscompat_test.go#TestSessionDeleteUnknownSilentSuccess"
        status: pass
    human_judgment: false
  - id: D3
    description: "session/fork declines via the EXPLICIT UnstableForkSession arm with the SDK's exact -32601 \"Method not found\" + {\"method\":\"session/fork\"} frame, byte-locked on the tee"
    requirement: SESS-02
    verification:
      - kind: integration
        ref: "internal/acpfront/sesscompat_test.go#TestSessionForkDeclineByteLocked"
        status: pass
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestForkArmIsExplicit"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every uncovered DISC-02 method byte-locked graceful: set_mode/list/load typed declines + the raw-frame session/set_model row (SDK fallback), each -32601 naming the method"
    requirement: SESS-03
    verification:
      - kind: integration
        ref: "internal/acpfront/sesscompat_test.go#TestUncoveredMethodDeclines"
        status: pass
    human_judgment: false
  - id: D5
    description: "The initialize advertisement stays honest: sessionCapabilities options-only (D-11), delete/fork/list/load/resume/close unadvertised, authMethods exactly the two ids — frozen by a mutation-verified guard"
    requirement: SESS-03
    verification:
      - kind: integration
        ref: "internal/acpfront/sesscompat_test.go#TestSessionCapabilitiesAdvertisementStayedHonest"
        status: pass
    human_judgment: false

# Metrics
duration: 23min
completed: 2026-11-22
status: complete
---

# Phase 7 Plan 03: Session-Compat Surface (SESS-02) + the D-02 Decline Sweep Summary

**session/delete now tears down registered ids and answers the dialect's `{}` everywhere (silent for unknown), fork declines via an explicit byte-identical arm, every uncovered DISC-02 method has a composed-tier byte-lock row, and the options-only advertisement is frozen by a mutation-verified guard.**

## Performance

- **Duration:** 23 min
- **Started:** 2026-11-22T12:39:17Z
- **Completed:** 2026-11-22T13:02:59Z
- **Tasks:** 3
- **Files modified:** 8 (4 production/test sources, 1 new test file, 3 TDD evidence records)

## Accomplishments

- SESS-02 delete arm: `Handlers.DeleteSession` added to the interface; `Front.DeleteSession` reuses the 07-01 teardown for registered ids and answers the spec's silent-success `{}` for unknown/already-closed/empty ids with zero backend frames; the adapter delegates (`a.h.DeleteSession`) coding front errors via wireError — the `-32601` default is gone
- SESS-02 fork arm: `agentAdapter.UnstableForkSession` implements the SDK's optional interface explicitly, returning `unadvertised(AgentMethodSessionFork)` — byte-identical to the dispatcher's type-assert default, proven by the composed-tier tee byte-lock
- D-02 sweep rows: set_mode/list/load typed declines + the raw-frame `session/set_model` row (the SDK connection fallback shape Orca would see) each byte-locked to `-32601 "Method not found"` with their exact `data.method` string
- T-07-08 guard: `TestSessionCapabilitiesAdvertisementStayedHonest` freezes the initialize advertisement at options-only (D-11) and locks authMethods to exactly the two advertised ids — mutation-verified (temporary `Delete` advertisement flips it red, then restored)
- Full plan verification green: named rows under `-race -count=3`, both packages under `-race -count=1`, and `mise run ci` green at wave close

## Task Commits

Each task was committed atomically (TDD tasks carry RED->GREEN pairs):

1. **Task 1 (tracer): session/delete registered teardown** — `e7476f3` (test, RED: RED_EVIDENCE_OK) -> `76c04d0` (feat)
2. **Task 2: unknown-id silent success + fork decline + D-02 rows** — `9e9ba7d` (test, RED: RED_EVIDENCE_OK on TestForkArmIsExplicit; the wire-shape rows landed CONFIRM-GREEN-DEFAULT, classifier verdict persisted verbatim) -> `88df6c7` (feat)
3. **Task 3: advertisement honesty lock** — `3c2a10a` (test, GUARD passing at HEAD, mutation-verified)

## Files Created/Modified

- `internal/acpfront/sesscompat_test.go` (new) — the SESS-02/D-02 composed-tier rows: delete registered/unknown/empty/already-closed, fork byte-lock, the set_mode/list/load/set_model decline table, and the advertisement guard
- `internal/acpport/types.go` — `Handlers.DeleteSession(ctx, sessionID) error` (the SESS-02 row beside CloseSession)
- `internal/acpfront/front.go` — `Front.DeleteSession` (registry-miss silent success; registered id delegates to CloseSession) with the nilerr contract rationale
- `internal/acpport/agent.go` — the DeleteSession delegation arm (replacing the `-32601` default) + the explicit `UnstableForkSession` decline arm
- `internal/acpport/agent_test.go` — scriptedHandlers gains DeleteSession; the two plan-04-03 unadvertised tables drop the session/delete row; `TestDeleteSessionDelegation` + `TestForkArmIsExplicit`
- `.planning/tdd/07-03-task{1-red,2-red,2-confirm-green}.json` — persisted RED evidence (RED_EVIDENCE_OK x2) and the honest confirm-green record (INVALID_RED unexpected_green)

## Decisions Made

- **Delete-unknown RED accounting:** the plan's Task-2 text expected the delete-unknown row to RED-fail "today" (against the `-32601` default), but Task 1's action — followed verbatim — implemented BOTH front arms in `Front.DeleteSession`, consuming that RED in Task 1's GREEN (`76c04d0`). The row landed CONFIRM-GREEN at Task-2 entry and is recorded honestly (the 05-08/07-04 precedent: classifier verdict persisted verbatim, never used as RED authorization).
- **Genuine fork RED:** the fork wire shape passes pre-implementation via the SDK default (the plan's own CONFIRM-GREEN-DEFAULT branch), so Task 2's genuine RED is the type-assert row `TestForkArmIsExplicit` — encoding the GREEN's actual production change (the explicit arm) — RED_EVIDENCE_OK before GREEN.
- **sessTeeReply helper:** decline reply frames carry `data.method`, which `authTeeResponse`'s method-free-reply filter excludes — the decline rows lock through a new id-matched helper instead of weakening the auth helper.
- **Close-advertisement disposition (recorded per the plan's Task-3 action):** NOT advertised in v1. Close is implemented (SESS-01) and Orca calls it ungated regardless; no v1 client checks the field, and advertising would churn every initialize byte-lock for zero v1 benefit. Revisit on demand.
- **Advertisement guard mutation-verified:** a temporary `SessionDeleteCapabilities` advertisement flipped the guard red with the Pitfall-4 message; HEAD restored to green — the guard guards.

## Deviations from Plan

None - plan executed exactly as written. (The delete-unknown RED accounting above is the plan's own CONFIRM-GREEN-DEFAULT branch applied to a row whose RED Task 1 consumed; `front.go` gaining nothing in Task 2 held — the helper and guard additions are test-only.)

## Issues Encountered

- `TestUnadvertisedStableMethodsAnswerNotFound` (the plan-04-03 raw-wire lock at agent_test.go:750) also drove session/delete and failed after the interface growth — the plan's read_first note ("the :418-423 default-arm test flips") mapped to both tables; both dropped the row, with the delegation + wire shape now locked by `TestDeleteSessionDelegation` and the composed-tier row.
- golangci strict-gate friction on the new code, resolved in-task: funlen splits (`assertSessDeleteTeardownWire`, `sessRawSetModelDecline`, `sessDeclineTable`), lll wraps, funcorder placement (exported `DeleteSession` before unexported `teardownAfterPickerRefusal`), and a nilerr nolint carrying the silent-success contract rationale.

## TDD Gate Compliance

- Task 1: `test(7-03)` `e7476f3` (RED, RED_EVIDENCE_OK) precedes `feat(7-03)` `76c04d0` — valid sequence.
- Task 2: `test(7-03)` `9e9ba7d` (RED on TestForkArmIsExplicit, RED_EVIDENCE_OK) precedes `feat(7-03)` `88df6c7` — valid sequence. The wire-shape rows in the same test commit passed at entry (CONFIRM-GREEN-DEFAULT); the classifier's `INVALID_RED unexpected_green` verdict on `07-03-task2-confirm-green.json` is persisted verbatim and was never used as RED authorization.
- No refactor commits were needed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The dialect table's uncovered rows are all implemented-or-locked; 07-05 (`/model` slash prompt) rides the same front surface and the `-32603`/`data.details` seam remains its own work (this plan touched neither `wireError`'s data path nor the prompt layer)
- 07-07's named conformance suite can compose the sess-compat arms (delete `{}` / fork decline rows) directly from `sesscompat_test.go`'s helpers
- The unadvertised delete/fork means Orca's behavior is unchanged by this plan — nothing new for 07-09's manual E2E beyond the documented posture

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*
