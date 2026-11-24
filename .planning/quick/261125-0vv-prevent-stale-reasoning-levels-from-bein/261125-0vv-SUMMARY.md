---
phase: 261125-0vv
plan: 01
subsystem: acpfront / model-selection reasoning
tags: [acp, setModel, set_config_option, model-picker, slash-command, reasoning-level, orca, golang, tdd]
requires: []
provides:
  - Exact-target reasoning resolution for picker and /model mutations
  - Pre-validation Orca enabled/disabled thought alias normalization
  - Fail-closed missing-target and unrepresentable disabled handling
  - Composed preserved-session retry and ACP wire-level regression coverage
affects: [internal/acpfront, session/setModel, session/set_config_option, /model, internal/zcode/fake]
actuals:
  tokens: 259
  tasks: 3
  commits: 6
tech-stack:
  added: []
  patterns:
    - "Resolve reasoning from the exact target ModelEntry: advertised current, advertised default, then first listed level"
    - "Normalize a closed alias vocabulary before option lookup and membership validation without enriching advertised options"
    - "Fail closed on missing model metadata before backend mutation"
key-files:
  created: []
  modified:
    - internal/acpfront/configoptions.go
    - internal/acpfront/front.go
    - internal/acpfront/pickerrefusal_internal_test.go
    - internal/acpfront/setconfigoption_test.go
    - internal/acpfront/configoptions_test.go
    - internal/zcode/fake/golden_test.go
    - internal/zcode/fake/testdata/golden/config-option-set-model.golden
key-decisions:
  - "Target metadata is authoritative; source/global thought values never cross into a different model"
  - "Orca aliases are accepted only as exact thought_level enabled/disabled inputs and never advertised"
  - "Disabled maps only to advertised off; unsupported disabled requests retain the existing coded allowed-values refusal"
  - "The historical recoverable-preservation lifecycle remains owned by its existing test; the new composed test starts from that preserved snapshot"
patterns-established:
  - "Exact-target preserve/default/first resolution with level-less omission and missing-target refusal"
  - "Closed-vocabulary alias normalization before rendered-option validation"
requirements-completed: []
coverage:
  - id: D1
    description: Exact-target reasoning resolution for picker and /model, including preserve/default/first and level-less omission
    verification:
      - kind: integration
        ref: "internal/acpfront/setconfigoption_test.go#TestSetConfigOptionTargetReasoningResolution"
        status: pass
      - kind: integration
        ref: "internal/acpfront/setconfigoption_test.go#TestModelSlashTargetReasoningResolution"
        status: pass
    human_judgment: false
  - id: D2
    description: One post-preservation DeepSeek/max retry sends target-advertised high and preserves the live session
    verification:
      - kind: integration
        ref: "internal/acpfront/pickerrefusal_internal_test.go#TestPickerCrossModelRetryResolvesTargetReasoning"
        status: pass
    human_judgment: false
  - id: D3
    description: Orca thought aliases normalize before lookup/validation while advertisement remains a verbatim select
    verification:
      - kind: integration
        ref: "internal/acpfront/setconfigoption_test.go#TestSetConfigOptionOrcaThoughtAliases"
        status: pass
      - kind: integration
        ref: "internal/acpfront/setconfigoption_test.go#TestThoughtLevelAdvertisementRemainsVerbatimSelect"
        status: pass
      - kind: unit
        ref: "internal/acpfront/configoptions_test.go#TestConfigOptionsTranslator"
        status: pass
    human_judgment: false
metrics:
  duration: 27min
  completed: 2026-11-25
  status: complete
  commits: 6
  plan_head_before: 7b59c64eda56c2e3b05622c28c4798056431b4c7
  plan_head_after: eff9118535511e423539a2dfe878cf7f422fac47
---

# Quick Task 261125-0vv: Target-Aware Reasoning and Orca Alias Normalization Summary

**Model switches now carry only reasoning levels advertised by the exact target model, and Orca's `enabled`/`disabled` thought writes normalize to supported target levels or fail before the backend wire without changing ACP option advertisement.**

## Performance

- **Duration:** 27min
- **Started:** 2026-11-24T22:07:04Z
- **Completed:** 2026-11-24T22:34:35Z (2026-11-25 local)
- **Tasks:** 3/3 completed
- **Files modified:** 7
- **Measured commits:** 6 from `7b59c64` through `eff9118`

## Accomplishments

- Added one pure exact-target resolver: preserve an advertised current level, else an advertised target default, else the target's first listed level; level-less targets omit options and missing targets fail closed.
- Routed both picker and `/model` model mutations through the resolver, so a preserved DeepSeek/max snapshot retries OpenCodeGo/hungerg-free once with advertised `high` and no teardown.
- Added exact Orca `enabled`/`disabled` normalization before rendered-option lookup and listed-value validation, including sessions whose thought select is omitted.
- Restricted `disabled` to advertised `off`; unsupported requests retain the existing coded `-32602` allowed-values shape with zero backend mutation or close frames.
- Kept thought advertisement a verbatim backend-valued ACP select, with no aliases, boolean variant, or changes to disabled/current-empty omission.
- Preserved same-model carry, recoverable/fatal lifecycle behavior, ACP error mapping, settings merge, and `CloseSession`.

## Task Commits

TDD tasks retain separate RED and GREEN commits; every commit is task-scoped and contains code/tests only.

1. **Task 1 RED: preserved cross-model retry regressions** — `7b3b3d6` (test)
2. **Task 1 GREEN: exact-target reasoning resolver** — `e6cb2c3` (feat)
3. **Task 2: ACP wire fallback matrix and /model parity** — `a3665e9` (test)
4. **Task 3 RED: alias, refusal, omission, and advertisement regressions** — `b151bf3` (test)
5. **Task 3 GREEN: pre-validation Orca alias normalization** — `988cef7` (feat)
6. **Rule 1 follow-up: level-less fake golden correction** — `eff9118` (fix)

Plan metadata is intentionally uncommitted for orchestrator handling. `STATE.md`, `ROADMAP.md`, and prior quick-task artifacts were not modified.

## RED to GREEN Evidence

### Task 1

Command:

```text
go test -race -count=1 -run '^(TestPickerCrossModelRetryResolvesTargetReasoning|TestSetConfigMutationMissingTargetFailsClosed)$' ./internal/acpfront
```

The composed target executed and failed on the decoded sole `session/setModel` payload: the unmodified shared mutation path resent source `max` instead of target-advertised `high`. The missing-target regression also returned success instead of coded invalid params. The unchanged report is `.planning/tdd/261125-0vv-task1-red.txt`; the JSON record returned `RED_EVIDENCE_OK` with `target_test_failed`.

### Task 3

Command:

```text
go test -race -count=1 -run '^(TestSetConfigOptionOrcaThoughtAliases|TestThoughtLevelAdvertisementRemainsVerbatimSelect|TestConfigOptionsTranslator)$' ./internal/acpfront
```

Rendered aliases failed listed-value validation as unlisted `enabled`/`disabled`; both aliases under thought-disabled and current-empty snapshots failed unknown-configId validation because normalization did not yet run. Missing/level-less metadata cases also exposed the global fallback rather than the exact bound target. The unchanged report is `.planning/tdd/261125-0vv-task3-red.txt`; the JSON record returned `RED_EVIDENCE_OK` with `target_test_failed`. Advertisement and translator omission controls passed.

## TDD Gate Compliance

| Task | RED | GREEN | Status |
|------|-----|-------|--------|
| 1 | `7b3b3d6` | `e6cb2c3` | Pass |
| 2 | — test-only task | `a3665e9` | Pass |
| 3 | `b151bf3` | `988cef7` | Pass |

Both behavior-changing tasks have a `test(...)` commit followed by a descendant `feat(...)` commit. The Task 1 tracer feedback gate re-ran its full focused race/lint verification in auto mode before Task 2 expansion and passed.

## Files Created/Modified

- `internal/acpfront/configoptions.go` — adds the pure exact-target reasoning resolver.
- `internal/acpfront/front.go` — integrates target resolution into model mutation and adds narrow pre-validation Orca alias normalization.
- `internal/acpfront/pickerrefusal_internal_test.go` — adds the composed already-preserved retry and direct missing-target fail-closed regression.
- `internal/acpfront/setconfigoption_test.go` — adds the reusable snapshot builder, ACP target matrix, `/model` parity, alias/refusal matrices, and verbatim advertisement lock.
- `internal/acpfront/configoptions_test.go` — adds alias-era disabled/current-empty translator omission locks.
- `internal/zcode/fake/golden_test.go` — aligns the config-option corpus driver comment with target-aware resolution.
- `internal/zcode/fake/testdata/golden/config-option-set-model.golden` — expects options omission on the level-less return target.

## Decisions Made

- Exact target metadata is authoritative; `settings.thoughtLevel.available` is never consulted for cross-model mutation or alias normalization.
- Only exact `enabled` and `disabled` strings on `thought_level` are aliases; unknown IDs and non-alias values keep the existing validation path.
- A missing binding, missing catalog entry, or level-less enabled target fails before mutation rather than synthesizing a level.
- Unsupported `disabled` fails rather than mapping to `low` or omitting a required level.
- The new composed regression starts from the preserved snapshot; the prior recoverable-preservation test remains the refusal-lifecycle owner.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated the directly affected level-less fake golden**
- **Found during:** Full repository race verification after Task 3
- **Issue:** `internal/zcode/fake` still expected stale `high` when switching back to level-less `zai-api/GLM-5.3`, contradicting the required omit-shape.
- **Fix:** Removed the stale options object from that expected frame and updated the corpus driver comment.
- **Files modified:** `internal/zcode/fake/golden_test.go`, `internal/zcode/fake/testdata/golden/config-option-set-model.golden`
- **Verification:** `go test -race -count=1 ./internal/zcode/fake` passed, followed by a passing full repository race suite.
- **Committed in:** `eff9118`

**Total deviations:** 1 auto-fix (Rule 1)
**Impact on plan:** The extra fixture update was directly caused by the planned resolver behavior and was required for the mandated full-repository race gate. No architectural or protected lifecycle changes were made.

## Issues Encountered

- Task 2's first test build used the wrong package for `LevelEntry`; corrected to `acpfront.LevelEntry`. Its same-model fixture initially referenced the cross-model target; corrected the fixture identity. Final matrix passed.
- Strict lint required test-helper decomposition, shared literals, line wrapping, and a small production normalizer helper split. Final full-repository lint reports zero issues.
- `golangci-lint` emits the existing `exhaustruct` deprecation warning; it is not a lint failure.

## User Setup Required

None — no external service configuration, dependency installation, or live-backend test was added.

## Next Phase Readiness

The target-aware resolver and Orca alias boundary are complete and fully verified. Orchestrator documentation handling can proceed; no code blockers remain.

## Known Stubs

None.

## Threat Flags

None. No new endpoint, authentication path, file-access pattern, schema, or dependency was introduced. The changes harden the existing ACP-input and front-to-backend trust boundaries described in the plan.

## Self-Check: PASSED

- All seven planned/directly affected code and fixture files exist.
- All six code commits are ancestors of `eff9118`; the measured plan-base commit count is exactly six.
- Both RED records re-validated as `RED_EVIDENCE_OK` and semantically match the planned failures.
- Full repository build, vet, lint, and race tests pass.
- No tracked files were deleted; planning and TDD evidence artifacts remain uncommitted for orchestrator handling.
