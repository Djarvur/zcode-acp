---
phase: 06-oauth-coding-plan-path-model-picker
plan: 06
subsystem: api
tags: [acp, config-options, session-config, setmodel, reasoning-level, tdd, golden-corpus]

# Dependency graph
requires:
  - phase: 06-04
    provides: "the settings-snapshot translator (ParseSettings/BuildConfigOptions), the per-session confirmed-settings store, the sessionCapabilities.options advertisement, and the neutral ConfigOptionDef vocabulary"
  - phase: 05-05
    provides: "the SESS-01 teardown (dropSettings beside dropCwd) and the closeOrphan WR-04 discipline the refused-mutation arm reuses"
  - phase: 04-02
    provides: "the ComposeFront harness and the acpport adapter contract tests (the scriptedHandlers fake the delegation test extends)"
provides:
  - "acpport.Handlers.SetSessionConfigOption — the neutral vocabulary method (validate-first contract documented)"
  - "acpport agentAdapter.SetSessionConfigOption — the -32601 default replaced by the Authenticate-shape delegation (wire triple verbatim, select-variant assembly, wireError coding)"
  - "acpfront Front.SetSessionConfigOption — validate-first (-32602 naming configId + allowed set/values), the locked-OQ1 setModel mapping with the level-carry rule, closeOrphan on refusal, the reply-built full-list re-emit"
  - "corpus rows config-option-set-model + config-option-thought (+ goldens + drivers) byte-locking the back wire incl. options.reasoningLevel and persistAsWorkspaceLastUsed:false"
affects: [06-07 responder accountScope widening, 06-08 real-run, phase-7 sess-03 model contract]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 25127    # chars/4 over the realized diff (100508 chars, 16 files)
  tasks: 3
  commits: 5       # MEASURED: git rev-list --count eb49f84..HEAD

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "validate-first handler posture: the confirmed-settings store is the gate — no backend call ever carries an unvalidated value (T-06-17)"
    - "level-carry rule on the locked OQ1 setModel path: options.reasoningLevel rides with the CURRENT confirmed level unless the ask changes it (Pitfall 2)"
    - "reply-built re-emit: store overwrites from the mutation reply BEFORE the rebuild — a fabricated currentValue is impossible by construction (D-10/T-06-19)"

key-files:
  created:
    - internal/acpfront/setconfigoption_test.go
    - internal/zcode/fake/testdata/scenarios/config-option-set-model.json
    - internal/zcode/fake/testdata/scenarios/config-option-thought.json
    - internal/zcode/fake/testdata/golden/config-option-set-model.golden
    - internal/zcode/fake/testdata/golden/config-option-thought.golden
    - .planning/phases/06-oauth-coding-plan-path-model-picker/evidence/06-06-red-*.json (5 records)
  modified:
    - internal/acpport/types.go
    - internal/acpport/agent.go
    - internal/acpport/agent_test.go
    - internal/acpfront/front.go
    - internal/acpfront/configoptions.go
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "Closed-session posture for set_config_option: the isClosing gate (T-5-12 parity) plus the settings-missing -32602 arm covers never-opened AND torn-down sessions (dropSettings rode the teardown) — no registry.Get gate, honoring the plan's arm-(c) -32602 for unknown session ids"
  - "A refused picker setModel tears the backend session down best-effort via closeOrphan (the plan's WR-04 prescription, twice: action + fails_when) and surfaces the coded internal error"
  - "Boolean-variant set_config_option asks translate to the empty value id and reject through the front's membership check — one validation path, coded -32602 naming the option's own listed set"

patterns-established:
  - "setConfigMutation: the single mapping table from validated ask to the locked OQ1 wire params (model switch = strings.Cut on the FIRST slash + carried level; thought switch = confirmed model unchanged + requested level)"
  - "wireKey const family extension: wireKeyProviderID/wireKeyModelID/wireKeyReasoningLevel/wireKeyPersistLastUsed join wireKeySessionID (the goconst extraction discipline)"

requirements-completed: [MODEL-02]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "set_config_option validates FIRST: unknown configId, out-of-set value, unknown session, and closed session all answer coded -32602 naming the configId and the allowed set/values, with NO backend frame on any invalid-arm transcript"
    requirement: MODEL-02
    verification:
      - kind: unit
        ref: "internal/acpfront/setconfigoption_test.go#TestSetConfigOptionInvalid"
        status: pass
    human_judgment: false
  - id: D2
    description: "The adapter delegates session/set_config_option to Handlers with the wire triple verbatim, assembles the Validate-clean select response, and passes a seam -32602 through with its code intact (never degraded to -32603); the -32601 default is gone from both unadvertised-methods locks"
    requirement: MODEL-02
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestSetConfigOptionDelegation"
        status: pass
    human_judgment: false
  - id: D3
    description: "A model switch maps to exactly one session/setModel parsed by the first-slash cut, carrying options.reasoningLevel = the CURRENT confirmed level (level-carry vector) or omitting options when none is confirmed (IN-04), persistAsWorkspaceLastUsed false; a refusal tears the backend session down best-effort (WR-04)"
    requirement: MODEL-02
    verification:
      - kind: unit
        ref: "internal/acpfront/setconfigoption_test.go#TestSetConfigOptionModelSwitch"
        status: pass
    human_judgment: false
  - id: D4
    description: "A thought-level switch travels the SAME setModel path (locked OQ1): confirmed model unchanged, requested level in options.reasoningLevel, setThoughtLevel never on the wire"
    requirement: MODEL-02
    verification:
      - kind: unit
        ref: "internal/acpfront/setconfigoption_test.go#TestSetConfigOptionThoughtSwitch"
        status: pass
    human_judgment: false
  - id: D5
    description: "The response re-emits the FULL option list rebuilt FROM THE MUTATION REPLY's settings snapshot (deep-compared against BuildConfigOptions(ParseSettings(reply))), with the confirmed store updated from the reply before the rebuild (a follow-up ask validates against the reply-confirmed set)"
    requirement: MODEL-02
    verification:
      - kind: unit
        ref: "internal/acpfront/setconfigoption_test.go#TestSetConfigOptionReEmit"
        status: pass
    human_judgment: false
  - id: D6
    description: "Two corpus goldens byte-lock the back wire: the set-model row shows the level carry on the picker's setModel; the thought row shows the setModel path as the only mutation frame; both drivers complete under the exchange timeout and the goldens are stable across two runs"
    requirement: MODEL-02
    verification:
      - kind: unit
        ref: "internal/zcode/fake/golden_test.go#TestGolden/config-option-set-model"
        status: pass
      - kind: unit
        ref: "internal/zcode/fake/golden_test.go#TestGolden/config-option-thought"
        status: pass
    human_judgment: false

# Metrics
duration: 33min
completed: 2026-11-22
status: complete
---

# Phase 6 Plan 06: Session set_config_option Summary

**Validate-first `session/set_config_option` mapped onto the locked OQ1 setModel path with the level-carry rule and reply-built full-list re-emit, byte-locked by two corpus goldens**

## Performance

- **Duration:** 33 min
- **Started:** 2026-11-22T03:09:30Z
- **Completed:** 2026-11-22T03:42:32Z
- **Tasks:** 3
- **Files modified:** 16 (6 source, 5 test/fixture, 5 evidence records)

## Accomplishments

- `session/set_config_option` validates FIRST against the CURRENT confirmed option set: unknown configId, out-of-set value, unknown session, and closed session all answer coded -32602 naming the configId and the allowed set/values — no backend frame ever leaves the wrapper on an invalid ask (T-06-17)
- Both switch kinds ride the field-proven `session/setModel` path (locked OQ1 — no setThoughtLevel constant anywhere): a model switch parses with `strings.Cut` on the FIRST slash and carries the CURRENT confirmed level (Pitfall 2), a thought-level switch carries the confirmed model unchanged with the requested level; `persistAsWorkspaceLastUsed:false` on both
- The response re-emits the FULL option list rebuilt from the mutation REPLY's settings snapshot (D-10: store overwrites from the reply before the rebuild — a fabricated currentValue is impossible by construction)
- The adapter delegation replaces the -32601 default (the Authenticate shape): wire triple verbatim, Validate-clean select assembly, seam codes intact
- Two corpus rows + goldens byte-lock the back wire, including `options.reasoningLevel` and the omit-shape discipline

## Task Commits

Each task was committed atomically (TDD RED→GREEN for tasks 1-2):

1. **Task 1: validate-first + delegation (RED)** - `62555bc` (test) — RED_EVIDENCE_OK ×2
2. **Task 1: validate-first + delegation (GREEN)** - `8464f51` (feat)
3. **Task 2: mutations + re-emit (RED)** - `d58f57a` (test) — RED_EVIDENCE_OK ×3
4. **Task 2: mutations + re-emit (GREEN)** - `032cf90` (feat)
5. **Task 3: corpus rows + goldens** - `0cf456a` (feat)

_Note: TDD tasks carry test→feat commit pairs; no REFACTOR commits were needed (the GREEN implementations landed clean)._

## Files Created/Modified

- `internal/acpport/types.go` — Handlers.SetSessionConfigOption with the validate-first doc contract
- `internal/acpport/agent.go` — the delegation replacing the -32601 default; the unadvertised surface shrank to four
- `internal/acpport/agent_test.go` — TestSetConfigOptionDelegation + the scriptedHandlers recording fake; the unadvertised tables updated
- `internal/acpfront/front.go` — Front.SetSessionConfigOption (validate-first, setConfigMutation, reply-built re-emit, closeOrphan on refusal) + the wire-key const family
- `internal/acpfront/configoptions.go` — the translator's option ids now read the shared consts (goconst hygiene)
- `internal/acpfront/setconfigoption_test.go` — the six front-tier contract tests + fixtures
- `internal/zcode/fake/golden_test.go` — the two config-option drivers (full front-stack composition)
- `internal/zcode/fake/testdata/scenarios/config-option-{set-model,thought}.json` + `testdata/golden/*.golden` — the back-wire locks

## Decisions Made

- **Closed-session posture:** the isClosing gate (T-5-12 parity with prompt/resume) plus the settings-missing -32602 arm covers both never-opened and torn-down sessions — no registry.Get gate, because the plan's arm (c) pins -32602 (not -32600) for unknown session ids, and the probe ledger names "the -32602 arm and the tear-down path" as the closed-session coverage
- **Refused-mutation teardown:** closeOrphan on a refused picker setModel exactly as the plan prescribes (action + fails_when); the registry entry deliberately stays for the client's explicit close — the WR-04 discipline applied verbatim
- **Boolean-variant asks:** translated to the empty value id and rejected through the front's membership check (one validation path; the empty string is never a listed value)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] configoptions.go option-id const reuse (goconst)**
- **Found during:** Task 2 GREEN (lint gate)
- **Issue:** introducing the optionIDModel/optionIDThought consts in front.go pushed configoptions.go's `"model"` literal to 3+ occurrences — the strict goconst flagged the pre-existing line, and the const now exists to serve it
- **Fix:** buildModelOption/buildThoughtLevelOption IDs now read the shared consts (mechanical, zero behavior change)
- **Files modified:** internal/acpfront/configoptions.go
- **Verification:** `mise run lint` 0 issues; the full acpfront suite green
- **Committed in:** 032cf90 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking-lint scope, mechanical)
**Impact on plan:** The deviation is the repo's established const-extraction lint hygiene touching a file one line outside the plan's <files> list; no scope creep.

## TDD Gate Compliance

- Task 1: RED `62555bc` (test(06-06)) precedes GREEN `8464f51` (feat(06-06)) — machine evidence RED_EVIDENCE_OK for TestSetConfigOptionInvalid and TestSetConfigOptionDelegation (evidence/06-06-red-validation.json, 06-06-red-delegation.json), semantic assessment recorded in each record
- Task 2: RED `d58f57a` precedes GREEN `032cf90` — RED_EVIDENCE_OK for TestSetConfigOptionModelSwitch, TestSetConfigOptionThoughtSwitch, TestSetConfigOptionReEmit (evidence/06-06-red-modelswitch.json, -thoughtswitch.json, -reemit.json)
- Both RED runs failed on the planned assertions (the -32601 default still in place / zero mutation frames / pre-ask rebuild), never on build or fixture faults
- No violations; the git-log gate sequence is intact

## Issues Encountered

- The plan's arm (c) (-32602 for unknown session ids) and arm (d) ("the existing isClosing/registry handling") pull toward different error classes for overlapping cases; resolved with the gate order isClosing → settings-missing -32602, documented in Decisions Made
- The SDK's `acp.NewInternalError` carries a generic message with detail in data — the raw-error delegation arm asserts the -32603 class (matching the authenticate delegation's established plain-error lock) rather than the message text

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- MODEL-02's handler surface is complete and corpus-locked; 06-07 (the responder accountScope widening) can proceed against the locked 05-06/05-07 seams with the red-first lock updates it plans
- The method-constants block is unchanged (envelope.go diff empty) — locked OQ1 honored; the dedicated setThoughtLevel method remains deliberately off the v1 wire
- The real-account proof of the level-carry rule (Pitfall 2's field check) remains the phase's real-run criterion (06-08), not this plan's scope

---
*Phase: 06-oauth-coding-plan-path-model-picker*
*Completed: 2026-11-22*

## Self-Check: PASSED

- All created files exist on disk (setconfigoption_test.go, 2 scenarios, 2 goldens, 5 evidence records, this SUMMARY — plus the modified types.go/agent.go/agent_test.go/front.go/configoptions.go/golden_test.go present)
- All 5 plan commits are ancestors of HEAD: 62555bc, 8464f51, d58f57a, 032cf90, 0cf456a
- TDD gate sequence: two test(06-06) → feat(06-06) pairs in ancestry order (each GREEN descends from its RED), machine evidence RED_EVIDENCE_OK on all five targets
- envelope.go diff vs eb49f84 is empty (the method-constants block unchanged — locked OQ1); zero non-test setThoughtLevel wire references
- Commits measured from the ledger: eb49f84..HEAD = 5 task commits + the docs commit (plan_head_before: eb49f846fe45861531391f6fb0fab654e350e2f8)
