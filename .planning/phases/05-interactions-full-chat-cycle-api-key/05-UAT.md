---
status: testing
phase: 05-Interactions & Full Chat Cycle (api-key)
source: [05-VERIFICATION.md]
started: 2026-11-20T19:45:00Z
updated: 2026-11-20T19:45:00Z
---

## Current Test

number: 1
name: End-of-phase REAL RUN (EXP-APIKEY-CYCLE-01) — guarded api-key full cycle on the real account
expected: |
  With the login keyring unlocked and ZCODE_ACP_API_KEY exported from the local
  ZCode desktop config (never committed), run the canonical guarded command:

    ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/

  Expected: authenticate proven by registry read-back (not the ack), two tiny
  turns, resume/close clean, the interaction/requestCredentialCipher ask answered
  from the keyring, zero orphan processes. Success criteria 4+5 close with this run.
awaiting: user response

## Tests

### 1. End-of-phase REAL RUN (EXP-APIKEY-CYCLE-01) — guarded api-key full cycle on the real account
expected: With the keyring unlocked and ZCODE_ACP_API_KEY exported (see 05-USER-SETUP.md for the keyring check via secret-tool and the scrub-first commands), `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/` passes: authenticate proven by registry read-back (not the ack), two tiny turns, resume/close clean, cipher ask answered from the keyring, zero orphans.
result: [pending]

### 2. Evidence recording for the real run — scrub + ledger flip
expected: The transport capture is scrubbed to EXP-APIKEY-CYCLE-01.jsonl (no credential material — `experiments/scan-secrets.sh` clean on the scrubbed file), the V-EXP id is recorded, and the PENDING Phase-5 ledger row at experiments/README.md:195 is flipped to actuals (planned spend vs actual spend + verdict line beneath it).
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
