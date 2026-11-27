---
phase: 261127-v5d
plan: "01"
subsystem: acp-bridge
tags: [acp, elicitation, plan-approval, user-input, golang]

# Dependency graph
requires:
  - phase: 05
    provides: the ask-the-user bridge (DecodeUserInputAsk / BuildElicitationRequest / MapElicitationOutcome) and its interact responder carry
  - phase: 261124-tj2
    provides: the re-announce dedupe discipline the ask bridge rides
provides:
  - UserInputOption.Value and UserInputAsk.Interaction on the neutral ask (wire value + schema.interaction discriminator)
  - plan-approval enum build with Const = wire option value and Title = display label
  - accepted-value normalization (enum Const or display Title) back to the wire value under the question text
  - a composed NewUserInputAsker round-trip lock for both plan-approval answer styles and the ordinary label contract
affects: [interact responder, zcode backend plan-approval decisions, any future ExitPlanMode flow]

actuals:
  tokens: 6041     # chars/4 over the realized diff (bfd9bfe..db57143)
  tasks: 2
  commits: 3       # measured: git rev-list --count bfd9bfe..HEAD
plan_head_before: bfd9bfed880b008ba8f188229f291576e19e5c03
plan_head_after: db571436895066d3d81c83ef936e061d7c1e6bc8

tech-stack:
  added: []
  patterns:
    - "Two answer identities, one discriminator: schema.interaction == plan_approval switches the bridge to wire-value identity; everything else stays label identity"
    - "Compile-first RED skeleton: the inert DTO vocabulary lands with the failing tests so RED fails on assertions, never on a build error (#3770)"

key-files:
  created: []
  modified:
    - internal/acpport/userinput.go
    - internal/acpport/userinput_internal_test.go

key-decisions:
  - "schema.interaction is the ONLY discriminator (never toolName sniffing), so an ordinary ask under a tool named ExitPlanMode stays on the label contract"
  - "An empty wire option value falls back to the label as the option identity, so the answer is the only string the client could have seen"
  - "Normalization matches only exact option value/label strings; every other answer passes through verbatim so the backend's allow-only-\"approve\" check denies it"
  - "No responder change: acceptAnswer already nests the map under content.answers, the first key normalizePlanApprovalAnswer reads"

patterns-established:
  - "Plan-approval value identity: Const = wire value, Title = display label, answer = wire value under the question text"
  - "Ordinary AskUserQuestion label identity unchanged: const = title = label on the wire and the selected label in the answer map"

requirements-completed: []

coverage:
  - id: D1
    description: "Plan-approval decode carries the params schema.interaction discriminator and each option's wire value; ordinary decode stays schema-less"
    verification:
      - kind: unit
        ref: "internal/acpport/userinput_internal_test.go#TestDecodeUserInputAsk"
        status: pass
    human_judgment: false
  - id: D2
    description: "Plan-approval build emits Const = wire value / Title = display label; ordinary asks keep const = title = label"
    verification:
      - kind: unit
        ref: "internal/acpport/userinput_internal_test.go#TestBuildElicitationRequestPlanApproval"
        status: pass
      - kind: unit
        ref: "internal/acpport/userinput_internal_test.go#TestBuildElicitationRequest"
        status: pass
    human_judgment: false
  - id: D3
    description: "Composed decode -> build -> client accept -> map path: both \"Approve\" and \"approve\" land \"approve\" under the question text; an unmatched answer passes through"
    verification:
      - kind: unit
        ref: "internal/acpport/userinput_internal_test.go#TestUserInputAskerPlanApprovalRoundTrip"
        status: pass
      - kind: unit
        ref: "internal/acpport/userinput_internal_test.go#TestMapElicitationOutcomePlanApproval"
        status: pass
    human_judgment: false

# Metrics
duration: 4min
completed: 2026-11-27
status: complete
---

# Quick Task 261127-v5d: Carry the plan-approval decision literal through the ACP ask bridge

**ExitPlanMode plan approval now renders `Const: "approve"` / `Title: "Approve"` and normalizes either accepted string back to `"approve"` under the question text, so the backend's `normalizePlanApprovalAnswer` grants `allow`; ordinary AskUserQuestion asks keep label identity untouched.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-11-27T19:33:25Z
- **Completed:** 2026-11-27T19:37:40Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- Decode no longer drops the wire option `value` or the params `schema.interaction` discriminator; both ride the neutral ask.
- Plan-approval elicitations build `OneOf[0]` as `Const` = option value (`"approve"`), `Title` = display label (`"Approve"`), description intact; an empty wire value honestly falls back to the label.
- `MapElicitationOutcome` normalizes either returned string (enum Const or display Title) to the option value keyed by question text; unrecognized answers pass through verbatim, so the backend can only allow on the intended literal.
- Multi-select shares the same per-element normalization; the free-text, decline, and cancel arms are byte-identical.
- A composed `NewUserInputAsker` round-trip locks the whole decode → build → accept → map path for both plan-approval answer styles and for the unchanged ordinary label contract.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: failing plan-approval carry tests** - `d382b1e` (test)
2. **Task 1 GREEN: carry the option value through the bridge** - `8fd323c` (feat)
3. **Task 2: composed plan-approval round-trip lock** - `db57143` (test)

_Plan metadata is committed by the orchestrator (docs artifacts are out of executor scope)._

## Files Created/Modified

- `internal/acpport/userinput.go` — `UserInputOption.Value`, `UserInputAsk.Interaction`, the `interactionPlanApproval` constant, `optionIdentity` / `answerSelection`, decode/build/map behavior, refreshed comments.
- `internal/acpport/userinput_internal_test.go` — decode/build/map plan-approval locks, shared params fixtures, and the composed asker round-trip.

## TDD Gate Compliance

| Gate | Commit | Evidence |
|------|--------|----------|
| RED | `d382b1e` (`test(261127-v5d): ...`) | `.planning/tdd/261127-v5d-task1-red.json` — `RED_EVIDENCE_OK`, reason `target_test_failed`, target `TestBuildElicitationRequestPlanApproval` |
| GREEN | `8fd323c` (`feat(261127-v5d): ...`) | focused + full `-race` suites green, `golangci-lint run ./internal/acpport/...` 0 issues |
| REFACTOR | — | none needed (lint-driven test helper extraction folded into GREEN) |

**RED semantic assessment:** the target tests failed on the planned assertions — `Const` was `"Approve"` (the label) instead of `"approve"` (the wire value), the map left `"Approve"` verbatim, and decode yielded empty `Value`/`Interaction`. No compile, load, syntax, or fixture fault. The inert struct-field skeleton landed with the RED commit so the vectors compile and fail on assertions, matching the project's 06-01/06-02/06-03 compile-first RED precedent and the `build_failure` refusal in the RED evidence gate.

**Task 2 note:** test-only by plan (no production edit). The composed round-trip test landed CONFIRM-GREEN on its first run after Task 1's GREEN; no RED evidence was fabricated for it.

## Decisions Made

- `schema.interaction` is the sole discriminator; tool-name sniffing was deliberately not used, so an ordinary ask running under a tool named `ExitPlanMode` stays on the label contract.
- Empty wire value → option identity is the label (the only string the client could have seen).
- Unmatched accepted strings pass through verbatim — never synthesize `"approve"`, which keeps the backend's allow-only check as the real gate.
- No `internal/interact` change: `acceptAnswer` already nests the answer map under `content.answers`, the first key the backend's normalizer reads.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical test fixture] Extended `elicitOrdinaryParams` with the second ordinary option**
- **Found during:** Task 2 planning against the plan text
- **Issue:** The plan asked `elicitOrdinaryParams` to be hoisted verbatim (one option, "Verify 1"), but Task 2's ordinary arm must accept q1 with "Exit manager" and assert the captured `Const`/`Title` equal that label — impossible with a one-option form.
- **Fix:** The hoisted fixture now carries both ordinary options ("Verify 1" with its description and "Exit manager"), and decode asserts both wire values.
- **Files modified:** `internal/acpport/userinput_internal_test.go`
- **Verification:** `TestDecodeUserInputAsk` and `TestUserInputAskerPlanApprovalRoundTrip` pass under `-race`.
- **Committed in:** `db57143`

**2. [Rule 3 - Blocking] Strict lint gate fixes in the test file**
- **Found during:** Task 1 GREEN lint run
- **Issue:** `golangci-lint` (default: all) flagged `goconst` on the four `"accept"` literals and `gocyclo` 23 > 20 on `TestDecodeUserInputAsk` after the new assertions.
- **Fix:** Hoisted `elicitAccept` and split the decode assertions into `assertOrdinaryDecode` / `assertPlanApprovalDecode` / `assertFreeTextDecode` helpers; no assertion dropped.
- **Files modified:** `internal/acpport/userinput_internal_test.go`
- **Verification:** `golangci-lint run ./internal/acpport/...` → 0 issues; tests green.
- **Committed in:** `8fd323c` (helper split and constant)

---

**Total deviations:** 2 auto-fixed (1 missing critical fixture, 1 blocking lint)
**Impact on plan:** Test-only adjustments; production behavior matches the plan exactly, and the diff remains limited to the two declared files.

## Issues Encountered

None beyond the lint-driven test helper extraction documented above.

## Verification

- `go test -race -count=1 -run '^(TestDecodeUserInputAsk|TestBuildElicitationRequest|TestBuildElicitationRequestFreeText|TestBuildElicitationRequestPlanApproval|TestMapElicitationOutcome|TestMapElicitationOutcomePlanApproval)$' ./internal/acpport` — pass.
- `go test -race -count=1 -run '^TestUserInputAskerPlanApprovalRoundTrip$' ./internal/acpport` — all three subtests pass.
- `go test -race -count=1 ./internal/acpport ./internal/interact` — pass (source compatibility of the added fields proven).
- `golangci-lint run ./internal/acpport/...` — 0 issues.
- `go build ./...` — pass.
- `git diff --stat bfd9bfe..HEAD` — only `internal/acpport/userinput.go` and `internal/acpport/userinput_internal_test.go`.

## Known Stubs

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The ACP bridge now answers ExitPlanMode plan approval with `content.answers[<question text>] == "approve"` for both client answer styles, so the backend's plan-approval normalizer grants `allow`.
- Remaining deliberately out of scope: `docs/protocol/app-server-dialect.md` sync for the plan-approval exception to the "options value=label" note (separate doc change).

## Self-Check: PASSED

- Files verified: `internal/acpport/userinput.go`, `internal/acpport/userinput_internal_test.go`, `261127-v5d-SUMMARY.md`
- Commits verified: `d382b1e`, `8fd323c`, `db57143`
