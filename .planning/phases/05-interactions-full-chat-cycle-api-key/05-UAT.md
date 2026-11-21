---
status: testing
phase: 05-Interactions & Full Chat Cycle (api-key)
source: [05-VERIFICATION.md]
started: 2026-11-20T19:45:00Z
updated: 2026-11-21T10:05:00Z
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
awaiting: none — attempt 2 verdict recorded (FAIL at the tool-turn leg; auth read-back leg field-proven passing). The phase stays gaps_found pending the next gap plan (turn-latch drain between sends).

## Tests

### 1. End-of-phase REAL RUN (EXP-APIKEY-CYCLE-01) — guarded api-key full cycle on the real account
expected: With the keyring unlocked and ZCODE_ACP_API_KEY exported (see 05-USER-SETUP.md for the keyring check via secret-tool and the scrub-first commands), `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/` passes: authenticate proven by registry read-back (not the ack), two tiny turns, resume/close clean, cipher ask answered from the keyring, zero orphans.
result: fail (2026-11-21 attempt 1, orchestrator-run) — the run reached the real account and failed at the authenticate read-back: provider/updateAccountConfig acked `{status:"received", providerCount:0, receivedRevision:"account-bridge:<epoch-nanos>"}` and provider/refreshRegistry acked `{refreshed:true}`, the desktop's own log applied runtime headers for GLM-5.3-Flash repeatedly (trace f0aff280, its refresh loop), but NO `state.updated` registry patch and no create-reply surface arrived within the 30 s window → ErrRegistryNotConfirmed; zero turns ran, zero model spend. Finding: the fake-tier read-back surface (state.updated registry patches) does not match real-desktop 3.18.2 behavior — candidate causes: (a) the desktop resolves its active builtin revision from a version dir its runtime version selects (runtime self-reports 0.17.1; version dirs are 3.x/0.0.0-dev) so our 3.18.2-path basedOn hash mismatches (dialect §8 silent discard), (b) an empty-providers snapshot is nothing-to-apply and never emits a patch, (c) the authenticate read-back surface should be the first session/create reply (settings.model.available), not a state.updated wait. Capture saved clean: experiments/captures/EXP-APIKEY-CYCLE-01.jsonl.
result: fail (2026-11-21 attempt 2, human-run per the 05-12 checkpoint) — the run reached far deeper and split honestly: authenticate{zai-api-key} PASSED proven by the corrected registry read-back verdict — the 05-11 branch-(c) fix is field-verified working (apicycle_test.go:205); session/new + D-03 subscribe gate + setModel zai-api/GLM-5.3-Flash ok; mid-turn interaction/requestCredentialCipher decrypt asks answered from the keyring (value_len=815, plaintext_len=555); the tiny tool-less turn COMPLETED end_turn with zero tool frames (apicycle_test.go:207 — the run's only real spend); the file-read tool turn then FAILED — the desktop refused the second send with "another prompt is already running (this agent runs one turn at a time)" (apicycle_test.go:208) → test FAIL at 10.34s; teardown read-loop warning benign (post-failure drain). New gap: the guarded test issues the second send without waiting for the first turn's terminal state — it needs a completion-wait/latch-drain between turns (next gap plan). Capture: experiments/captures/EXP-APIKEY-CYCLE-02.jsonl (scrubbed, scan-secrets clean, human-confirmed "evidence recorded").

### 2. Evidence recording for the real run — scrub + ledger flip
expected: The transport capture is scrubbed to EXP-APIKEY-CYCLE-01.jsonl (no credential material — `experiments/scan-secrets.sh` clean on the scrubbed file), the V-EXP id is recorded, and the PENDING Phase-5 ledger row at experiments/README.md:195 is flipped to actuals (planned spend vs actual spend + verdict line beneath it).
result: pass with caveat (2026-11-21 attempt 2, fail-path evidence recording, human-confirmed "evidence recorded") — the raw capture (born outside the repo) was scrubbed by experiments/scrub-capture.mjs into experiments/captures/EXP-APIKEY-CYCLE-02.jsonl (119/119 lines, zero secret shapes, zero absolute-path residue, idempotent fixed point; scan-secrets exit 0 on the tree and the staged diff); the V-EXP id (EXP-APIKEY-CYCLE-01 — the test-locked captureID) is recorded and the Phase-5 ledger row gained the attempt-2 actuals + verdict line beneath the attempt-1 record. Caveat: the row header stays PENDING per its discipline — it flips to actuals only on a passing run. Naming note: attempt 2 scrubs to file -02 (not -01) so attempt-1's committed capture (d15a745) stays byte-intact.

## Summary

total: 2
passed: 1
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- TOOL-01/turn sequencing (NEW, attempt 2, 2026-11-21): the desktop runs one turn at a time — the guarded test's second `session/send` (file-read turn) was refused with "another prompt is already running (this agent runs one turn at a time)". The test needs a completion-wait/latch-drain between turns (await the first turn's terminal surface before the second send). The attempt-1 read-back gap is CLOSED (05-11 branch (c) field-proven in attempt 2); the phase stays gaps_found on this new gap. AUTH-01 stays `[ ]` until a passing real run.
