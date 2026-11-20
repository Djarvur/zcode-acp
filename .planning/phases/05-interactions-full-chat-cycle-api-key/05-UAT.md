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
result: fail (2026-11-21 attempt 1, orchestrator-run) — the run reached the real account and failed at the authenticate read-back: provider/updateAccountConfig acked `{status:"received", providerCount:0, receivedRevision:"account-bridge:<epoch-nanos>"}` and provider/refreshRegistry acked `{refreshed:true}`, the desktop's own log applied runtime headers for GLM-5.3-Flash repeatedly (trace f0aff280, its refresh loop), but NO `state.updated` registry patch and no create-reply surface arrived within the 30 s window → ErrRegistryNotConfirmed; zero turns ran, zero model spend. Finding: the fake-tier read-back surface (state.updated registry patches) does not match real-desktop 3.18.2 behavior — candidate causes: (a) the desktop resolves its active builtin revision from a version dir its runtime version selects (runtime self-reports 0.17.1; version dirs are 3.x/0.0.0-dev) so our 3.18.2-path basedOn hash mismatches (dialect §8 silent discard), (b) an empty-providers snapshot is nothing-to-apply and never emits a patch, (c) the authenticate read-back surface should be the first session/create reply (settings.model.available), not a state.updated wait. Capture saved clean: experiments/captures/EXP-APIKEY-CYCLE-01.jsonl.

### 2. Evidence recording for the real run — scrub + ledger flip
expected: The transport capture is scrubbed to EXP-APIKEY-CYCLE-01.jsonl (no credential material — `experiments/scan-secrets.sh` clean on the scrubbed file), the V-EXP id is recorded, and the PENDING Phase-5 ledger row at experiments/README.md:195 is flipped to actuals (planned spend vs actual spend + verdict line beneath it).
result: [pending]

## Summary

total: 2
passed: 0
issues: 1
pending: 1
skipped: 0
blocked: 0

## Gaps
