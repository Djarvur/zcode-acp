---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 05
subsystem: api
tags: [acp, json-rpc, omp-dialect, model-picker, slash-prompt, error-seam]

# Dependency graph
requires:
  - phase: "07-03 (sesscompat)"
    provides: "the composed-tier discipline, the validate-first posture, the byte-lock test idioms this plan's slash layer rides"
  - phase: "06-04..06-06 (configOptions/set_config_option)"
    provides: "the picker machinery the intercept reuses: latestSettings -> BuildConfigOptions, setConfigMutation model arm, confirmedLevel, persist-false payloads"
  - phase: "07-01 (WR-01 teardown)"
    provides: "teardownAfterPickerRefusal — the refused-mutation teardown the slash layer mirrors"
provides:
  - "the /model slash-prompt intercept in Front.Prompt (validate-first, before acquireTurn — refusals never hold the front-wide turn slot)"
  - "the opt-in detail seam: acpport.Error.Details + NewInternalWithData, wired through wireError as data:{details:...} (generic frames keep data null)"
  - "the pinned Hb refusal contracts byte-locked: 'Unknown ACP model: <name>' and 'No model selected.' at -32603 + data.details, each string in exactly one production site"
  - "the edge-arm locks: slot-free refusals, byte-exact case-sensitive selector matching, confirmed-level carry with the IN-04 omit-shape, literal pass-through for non-commands"
affects: ["07-09 (QA-03 manual E2E — the /model switch checklist row now has wrapper-side support)", "verify-work (SESS-03 acceptance)"]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 15653
  tasks: 3
  commits: 6

# Tech tracking
tech-stack:
  added: [] # stdlib-only seam (strings.CutPrefix; the SDK's existing NewRequestError data parameter)
  patterns:
    - "detail-carrying seam errors: Message == Details (one source of truth, two read paths — humans read the message, matchers read data.details)"
    - "prompt-text command intercept: detected before slot acquisition, resolved against confirmed state only"

key-files:
  created:
    - internal/acpfront/slashmodel_test.go
  modified:
    - internal/acpfront/front.go
    - internal/acpport/types.go
    - internal/acpport/agent.go

key-decisions:
  - "The intercept reuses setConfigMutation's model arm verbatim (never a second setModel path) — level carry, persist-false, and the omit-shape come from the picker's own code"
  - "A backend-REFUSED slash setModel runs teardownAfterPickerRefusal (WR-01 parity) — the plan pinned the happy path 'exactly as SetSessionConfigOption does'; its refusal side mirrors the same one-teardown-contract discipline instead of the launch binding's wire-only closeOrphan"
  - "Pinned refusal strings live as one constant each (unknownACPModelFormat / noModelSelectedDetail) referenced by all arms — 'exactly one production site' honored under three no-model arms"
  - "Task 3 landed CONFIRM-GREEN at RED per the plan's anticipated branch (05-08/07-04 precedent) — the task-1/2 implementation already carried the edge contracts; evidence persisted verbatim, no GREEN production commit"

patterns-established:
  - "Tee-frame byte-lock for wire-classified error carriages: decode the raw frame's error.code + data.details, assert exact strings (the client matches startsWith)"
  - "Opt-in error-data widening: the seam carries Details only where a client matcher demands it; unrelated frames stay data null (guard-tested)"

requirements-completed: [SESS-03]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "The /model happy switch: session/prompt '/model <listed-value>' applies through the picker's setModel machinery (confirmed level, persist-false), records the reply as the confirmed store, emits one confirmation chunk, and resolves end_turn — the literal text never reaches the backend"
    requirement: SESS-03
    verification:
      - kind: integration
        ref: "internal/acpfront/slashmodel_test.go#TestModelSlashSwitchAppliesViaSetModel"
        status: pass
    human_judgment: false
  - id: D2
    description: "The detail seam + byte-locked refusals: unknown selectors answer -32603 with data.details EXACTLY 'Unknown ACP model: <name>'; both no-model arms answer 'No model selected.'; unrelated frames keep data null"
    requirement: SESS-03
    verification:
      - kind: integration
        ref: "internal/acpfront/slashmodel_test.go#TestModelSlashUnknownRefusalByteLocked"
        status: pass
      - kind: integration
        ref: "internal/acpfront/slashmodel_test.go#TestModelSlashNoModelRefusal"
        status: pass
    human_judgment: false
  - id: D3
    description: "The edge arms: refused switches hold no turn slot; selector matching is byte-exact (case-sensitive, single separating space); non-commands pass through verbatim; the level carry matches the picker (confirmed level rides, no level = IN-04 omit-shape)"
    requirement: SESS-03
    verification:
      - kind: integration
        ref: "internal/acpfront/slashmodel_test.go#TestModelSlashRefusalHoldsNoTurnSlot"
        status: pass
      - kind: integration
        ref: "internal/acpfront/slashmodel_test.go#TestModelSlashSelectorByteEquality"
        status: pass
      - kind: integration
        ref: "internal/acpfront/slashmodel_test.go#TestModelSlashCarriesConfirmedLevel"
        status: pass
      - kind: other
        ref: "go test -race -count=3 -run 'TestModelSlash' ./internal/acpfront/ (determinism under race, exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 19min
completed: 2026-11-22
status: complete
plan_head_before: 9253a0186dd5e52426327122397256bd6201a1dc
plan_head_after: 889408745a702124987b7a74646355ac1d273700
commits: 6
---

# Phase 7 Plan 05: The /model Slash-Prompt Layer Summary

**The D-03 /model slash layer: a validate-first Prompt intercept that applies listed selectors through the picker's own setModel machinery and refuses byte-exactly at -32603 + data.details — the omp Hb contracts (`Unknown ACP model: `, `No model selected.`) now live on the wire, making `resolvesSelectorsFrom: 18.1.9` true of the wrapper**

## Performance

- **Duration:** 19 min
- **Started:** 2026-11-22T14:01:18Z
- **Completed:** 2026-11-22T14:19:49Z
- **Tasks:** 3
- **Files modified:** 4 (plus 6 committed RED-evidence records)

## Accomplishments
- The /model intercept sits in Front.Prompt between the isClosing gate and acquireTurn: a refused switch never holds the front-wide turn slot (edge-locked, not inferred)
- The selector resolves byte-exactly against the confirmed configOptions listed values via the SAME latestSettings -> BuildConfigOptions read the picker uses — validate-first, T-06-17 posture: no unvalidated string ever reaches a backend setModel
- The opt-in detail seam: acpport.Error.Details + NewInternalWithData; wireError widens ONLY detail-carrying frames with data:{details:...} — generic coded frames keep data null (guard-tested)
- The pinned Hb strings exist in exactly one production site each, byte-locked on the tee frames from two independent sources (dialect doc + orca omp-dialect.ts)
- Level carry is picker-verbatim: confirmedLevel rides; no confirmed level emits the IN-04 omit-shape so the backend's own actionable refusal surfaces

## TDD Evidence

- Task 1 (tracer): genuine RED — all three subtests failed with the literal slash text riding session/send into the fake's unknown-method refusal; RED_EVIDENCE_OK (target_test_failed); GREEN ab112b8
- Task 2: genuine RED x2 — every refusal frame carried data null (the plan's stated RED state); RED_EVIDENCE_OK on both targets; GREEN 7c0a7a3
- Task 3: CONFIRM-GREEN at RED per the plan's anticipated branch ("if task 1 landed correctly some may pass; record honestly") — all six edge arms passed at RED time; INVALID_RED (unexpected_green) records persisted verbatim in .planning/tdd/07-05-task3-red-*.json; no GREEN production commit needed (nothing failed)

## Task Commits

Each task was committed atomically (TDD: RED -> GREEN per task):

1. **Task 1: /model happy switch tracer** — `b3acf06` (test RED), `ab112b8` (feat GREEN)
2. **Task 2: the detail seam + byte-locked refusal arms** — `16ab4db` (test RED), `7c0a7a3` (feat GREEN)
3. **Task 3: the edge arms** — `e210c6d` (test, CONFIRM-GREEN at RED)
4. **RED evidence records** — `8894087` (test)

**Plan metadata:** the docs commit (this SUMMARY + STATE + ROADMAP + REQUIREMENTS)

## Files Created/Modified
- `internal/acpfront/front.go` — the promptModelSlash intercept + modelSlashSelector + the pinned-string constants; Prompt call site before acquireTurn
- `internal/acpport/types.go` — Error gains the opt-in Details field; NewInternalWithData constructor
- `internal/acpport/agent.go` — wireError carries Details as data:{details:...}; the empty path unchanged; the *RequestError stays unwrapped
- `internal/acpfront/slashmodel_test.go` — the byte-locked happy/refusal/edge rows (5 top-level tests, 12 subtest arms)

## Decisions Made
- The intercept reuses setConfigMutation's model arm verbatim — never a second setModel path (the D-03 machinery instruction read literally)
- A backend-refused slash setModel runs teardownAfterPickerRefusal (WR-01 parity): the plan pinned the happy path's "exactly as SetSessionConfigOption does"; the refusal side mirrors the picker's one-teardown-contract discipline rather than the launch binding's wire-only closeOrphan, so a dead mutation target can never leave a zombie front session reachable by later prompts
- Pinned strings ride one constant each (unknownACPModelFormat / noModelSelectedDetail) — "exactly one production site" holds even though three arms answer the no-model sentence
- The undecodable-snapshot arm stays a plain NewInternal (data null): a wrapper-bug class, deliberately NOT dressed as a client-facing no-model refusal

## Deviations from Plan

None - plan executed exactly as written. (The WR-01 teardown parity and the CONFIRM-GREEN branch are both plan-sanctioned readings, documented under Decisions Made and TDD Evidence.)

## Issues Encountered
- The RED-evidence classifier rejected the first Task 2 record because its targetTest named the run pattern (`A|B`) instead of one exact test — resolved by splitting into per-target records from the same run (the 07-04 multi-record precedent); the superseded record was removed
- gofmt's comment reflowing mangled a double-backtick sequence in a test doc comment — reworded; two lint findings (lll 121-chars, an unused nolint directive) fixed before the GREEN commits

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- SESS-03's slash half is complete end-to-end; both declaring siblings (07-03/07-04) landed earlier, so the requirement is ready to mark
- 07-09's QA-03 checklist row "/model <name> switch from the picker" now exercises a fully-locked wrapper surface; a typo'd switch renders the card's own Unknown-model styling instead of a generic failure
- `mise run ci` green at wave close (exit 0, all 10 packages under -race); `bash experiments/scan-secrets.sh` exit 0

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*
