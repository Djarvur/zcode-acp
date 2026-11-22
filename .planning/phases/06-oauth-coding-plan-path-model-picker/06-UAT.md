---
status: testing
phase: 06-OAuth Coding-Plan Path & Model Picker
source: [06-VERIFICATION.md]
started: 2026-11-22T05:52:00Z
updated: 2026-11-22T05:52:00Z
---

## Current Test

number: 1
name: THE REAL RUN — official coding-plan chain end-to-end on the real account (D-12 phase bar)
expected: |
  With the ZCode desktop logged into the individual coding plan and the keyring unlocked:
  run the guarded real-tier test (harness is race-clean; ledger row EXP-CODINGPLAN-CYCLE-01 is PENDING):

    export ZCODE_ACP_RAW_CAPTURE=<scratch path outside this repo>
    ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/

  Expected: registry read-back proves account models appear; real PAT minted via the official
  api.z.ai chain; configOptions served from the live registry; one tool-less turn answered
  requestAuth {token, id, accountScope}; zero-orphan shutdown. Then the evidence contract:
  scrub capture into experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl, fill ledger actuals +
  V-EXP id, scan-secrets on tree AND staged diff, record A4/A6/A7 confirm-or-adjust notes,
  confirm the evidence commit. A failed leg records the honest verdict and the phase stays open.
awaiting: user response

## Tests

### 1. THE REAL RUN — official coding-plan chain end-to-end on the real account (D-12 phase bar)
expected: |
  With the ZCode desktop logged into the individual coding plan and the keyring unlocked:
  run the guarded real-tier test (harness is race-clean; ledger row EXP-CODINGPLAN-CYCLE-01 is PENDING):

    export ZCODE_ACP_RAW_CAPTURE=<scratch path outside this repo>
    ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/

  Expected: registry read-back proves account models appear; real PAT minted via the official
  api.z.ai chain; configOptions served from the live registry; one tool-less turn answered
  requestAuth {token, id, accountScope}; zero-orphan shutdown. Then the evidence contract:
  scrub capture into experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl, fill ledger actuals +
  V-EXP id, scan-secrets on tree AND staged diff, record A4/A6/A7 confirm-or-adjust notes,
  confirm the evidence commit. A failed leg records the honest verdict and the phase stays open.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
