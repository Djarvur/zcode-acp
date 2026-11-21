---
phase: 05-interactions-full-chat-cycle-api-key
plan: 12
subsystem: testing
tags: [acp, zcode, realdesktop, real-run, api-key, uat, evidence, quota-ledger, capture-hygiene]

# Dependency graph
requires:
  - phase: 05 (05-09)
    provides: the guarded real tier (TestAPIKeyFullCycle, double guard, tee capture, staged shutdown) this plan ran
  - phase: 05 (05-10)
    provides: CR-01 drainActiveTurn sawTurn closure — the disposition-ledger gate this retry ran behind
  - phase: 05 (05-11)
    provides: the field-verified branch-(c) authenticate chain (probe-create read-back) the run proved live
provides:
  - The attempt-2 REAL RUN verdict, recorded honestly: auth read-back leg FIELD-PROVEN on the live 3.18.2 desktop (the 05-11 fix works — authenticate confirmed via the corrected registry read-back verdict, never the ack); one tiny tool-less turn completed end_turn (the run's only real spend); the file-read tool turn FAILED on the desktop's one-turn-at-a-time latch — the full real-tier cycle is still unproven
  - The attempt-2 evidence trail: scrubbed capture experiments/captures/EXP-APIKEY-CYCLE-02.jsonl (scan-secrets green, human-confirmed) + the attempt-2 ledger line (row stays PENDING for a passing run) + UAT attempt-2 records with the new turn-latch gap
  - The next gap plan's charter: the guarded test needs a completion-wait/latch-drain between sends (await the first turn's terminal surface before the second session/send)
affects: [next Phase-5 gap plan (turn-latch drain + REAL RUN attempt 3), Phase 6 auth/registry, /gsd:verify-work]

# Actuals (#2632) — same estimateTokens scale (chars/4 over the realized diff).
actuals:
  tokens: 41508    # measured at write: 148988 git-diff chars (64c8e1f..3c80488) + 17045 SUMMARY bytes, / 4
  tasks: 3
  commits: 1       # MEASURED: git rev-list --count 64c8e1f..HEAD at SUMMARY write (the docs close-out commit follows, precedent: 05-11)
plan_head_before: 64c8e1f66a43fbad226ad11f5608eccb1aa9b086
plan_head_after: 3c8048884de6b0572118dc92a70d65c5eb20eed5

# Tech tracking
tech-stack:
  added: []        # zero new dependencies — the plan ran existing guarded tests and the existing scrub pipeline only (T-5-SC held)
  patterns:
    - "human-gated spending runs: the executor never runs the spending command; the verdict arrives as the user's in-chat message; evidence lands through the scrub-first pipeline with the human confirming the scrubbed content before any commit"
    - "evidence monotonicity: each attempt's capture is a new file — a committed capture is never overwritten, so ledger evidence pointers never silently repoint"

key-files:
  created:
    - experiments/captures/EXP-APIKEY-CYCLE-02.jsonl
    - .planning/phases/05-interactions-full-chat-cycle-api-key/05-12-SUMMARY.md
  modified:
    - experiments/README.md
    - .planning/phases/05-interactions-full-chat-cycle-api-key/05-UAT.md
    - .planning/STATE.md
    - .planning/ROADMAP.md

key-decisions:
  - "Honest split verdict recorded, no green-washing: the auth read-back leg is FIELD-PROVEN (the 05-11 branch-(c) probe-create fix works on the live desktop — apicycle_test.go:205), but the full cycle FAILS at the file-read tool turn (desktop turn-latch refusal, apicycle_test.go:208); the phase stays gaps_found and AUTH-01 stays [ ] until a passing run"
  - "Attempt-2 capture scrubbed to EXP-APIKEY-CYCLE-02.jsonl (not overwriting EXP-APIKEY-CYCLE-01.jsonl): attempt-1's committed capture (d15a745) stays byte-intact so the attempt-1 ledger line's evidence pointer never repoints; the header's V-EXP id EXP-APIKEY-CYCLE-01 is the test-locked captureID and is recorded as such in the ledger line"
  - "The Phase-5 ledger row header stays PENDING per its discipline — it flips to actuals only on a passing run; the attempt-2 failure line appends beneath attempt-1 + DIAGNOSTIC with spend actuals (one tiny completed turn) and the failing leg, never credentials"
  - "Hygiene held end-to-end: the raw capture lived at /tmp/zcode-acp-uat/apikey-cycle-02.raw.jsonl (outside the repo, mode 600) and was read ONLY by experiments/scrub-capture.mjs; the user retains the raw file for disposal"

patterns-established:
  - "Checkpoint round-trip for judgment acts: mechanical support (scrub, scan, counts-only verification) by the executor; the ledger flip + commit only after the human's 'evidence recorded' confirmation"
  - "Counts-only verification posture: capture verification emits counts and placeholder tallies, never content — no secret-shaped byte ever enters a log or context"

requirements-completed: []  # deliberate: AUTH-01 stays [ ] — the restore to Complete is the orchestrator's post-verification act ONLY after a passing run (05-UAT item 1 still fails); no mark-complete called

coverage:
  - id: D1
    description: "Pre-flight automated gate: the gap-fixed tree green with both double-guard directions proven before any human run"
    requirement: AUTH-01
    verification:
      - kind: integration
        ref: "mise run ci (27.55s green: build/vet/lint/test/test-experiments) + tagged run without ZCODE_ACP_REAL_E2E=1 skips in ~2ms + no-tag build excludes the guarded package + scan-secrets exit 0 (Task 1, previous checkpoint return)"
        status: pass
    human_judgment: false
  - id: D2
    description: "REAL RUN attempt 2 executed by the human on the real z.ai api-key account, verdict recorded (FAIL at the tool-turn leg; auth leg field-proven passing)"
    requirement: AUTH-01
    verification: []
    human_judgment: true
    rationale: "The run spends real API budget and is human-executed per the blocking checkpoint gates (T-5-30); the verdict arrived as the user's in-chat message with the go test -v sequence; the full cycle remains unproven — UAT item 1 records the honest fail"
  - id: D3
    description: "Evidence trail: scrubbed capture EXP-APIKEY-CYCLE-02.jsonl + clean secret scan + attempt-2 ledger line + UAT attempt-2 records"
    requirement: AUTH-01
    verification:
      - kind: other
        ref: "bash experiments/scan-secrets.sh — exit 0 on the tree AND on the staged diff (capture staged: 119/119 lines, zero JWT/credential/token shapes, zero absolute-path residue, scrubber idempotent fixed point)"
        status: pass
    human_judgment: true
    rationale: "Credential hygiene and quota accounting are human judgment acts on machine-local material (UAT item 2 why_human) — the user reviewed and confirmed 'evidence recorded' on 2026-11-21 before the commit"

# Metrics
duration: 77min
completed: 2026-11-21
status: complete
---

# Phase 5 Plan 12: REAL RUN UAT Retry (Attempt 2) & Evidence Recording Summary

**REAL RUN attempt 2 recorded honestly: the 05-11 read-back fix is field-proven on the live 3.18.2 desktop (auth leg PASS via the corrected registry read-back verdict, one real tiny turn spent), but the file-read tool turn hit the desktop's one-turn-at-a-time latch — the phase stays gaps_found with the evidence trail committed.**

## Performance

- **Duration:** 77 min (includes the human-executed run + the checkpoint review round-trip)
- **Started:** 2026-11-21T08:50:19Z (plan-head ledger birth, post-05-11)
- **Completed:** 2026-11-21T10:07:30Z
- **Tasks:** 3 (Task 2 executed BY THE USER per the blocking-human gate; Task 3 completed on the user's "evidence recorded" confirmation)
- **Files modified:** 6

## Accomplishments

- **The 05-11 authenticate fix is FIELD-VERIFIED WORKING.** Attempt 2 got past the leg that killed attempt 1: `authenticate{zai-api-key}` was proven by the corrected registry read-back verdict (the probe-create surface — never the ack; apicycle_test.go:205 "authenticate ok — the registry read-back verdict proved the key landed"). Attempt 1's read-back gap is CLOSED.
- **Real spend happened and is ledgered:** one tiny tool-less turn completed `end_turn` with zero tool frames (apicycle_test.go:207) — the run's only model spend. Session/new + D-03 subscribe gate + setModel zai-api/GLM-5.3-Flash all worked; mid-turn `interaction/requestCredentialCipher` decrypt asks were answered from the keyring live (value_len=815, plaintext_len=555) — the D-06 seam works on the real desktop.
- **The honest failure is recorded with its evidence:** the file-read tool turn was refused by the desktop — "another prompt is already running (this agent runs one turn at a time)" (apicycle_test.go:208) → test FAIL at 10.34s; teardown read-loop warning benign (post-failure drain). No green-washing anywhere: UAT item 1 fails, the ledger row stays PENDING, AUTH-01 stays `[ ]`, the phase stays gaps_found.
- **Evidence trail committed:** scrubbed capture `experiments/captures/EXP-APIKEY-CYCLE-02.jsonl` (119/119 lines, zero secret shapes, zero absolute-path residue, scrubber idempotent fixed point; scan-secrets exit 0 on the tree AND the staged diff) + the attempt-2 ledger line beneath attempt-1 + DIAGNOSTIC + UAT attempt-2 records with the new turn-latch gap.

## The attempt-2 verdict (human-executed 2026-11-21, 10.34s)

```text
1. Install observed: layout=unpacked-appimage, root=~/Applications/zcode-3.18.2/squashfs-root,
   builtinRevision=zcode-builtin:44:e9c0674a8011…cd1        (rev-44 endpoint builtin matched)
2. authenticate method=zai-api-key  outcome="authenticated (read-back proven)"   ← 05-11 fix FIELD-PROVEN
3. session/new ok: sess_<uuid> (create + D-03 subscribe gate + setModel zai-api/GLM-5.3-Flash)
4. mid-turn interaction/requestCredentialCipher decrypt asks answered from the keyring
5. apicycle_test.go:207  tiny tool-less turn ok: end_turn, zero tool frames   ← the run's only real spend
6. apicycle_test.go:208  file-read tool turn: desktop refused — "another prompt is already
   running (this agent runs one turn at a time)"              ← --- FAIL: TestAPIKeyFullCycle (10.34s)
   WARN zcode: read loop ended err="read |0: file already closed"  (benign teardown after the failure)
```

Session ids in the raw capture are scrubbed to stable `<uuid-N>` placeholders (referential integrity preserved for any later diagnosis).

## Task Commits

1. **Task 1: pre-flight automated gate** — verification-only, no commit (evidence in the first checkpoint return: mise ci 27.55s green, double guard proven both ways, scan-secrets clean)
2. **Task 2: REAL RUN attempt 2** — executed BY THE USER at the blocking-human checkpoint; no executor commit (the spending command never ran under executor control)
3. **Task 3: evidence recording** — `3c80488` (docs: scrubbed capture + ledger attempt-2 line + UAT attempt-2 records; scan-secrets green on the staged diff; committed after the user's "evidence recorded" confirmation)

**Plan metadata:** committed with the docs close-out commit (SUMMARY + STATE + ROADMAP).

## Files Created/Modified

- `experiments/captures/EXP-APIKEY-CYCLE-02.jsonl` - the attempt-2 scrubbed capture (mode 600; header V-EXP id EXP-APIKEY-CYCLE-01 is the test-locked captureID; file -02 preserves attempt-1's capture byte-intact)
- `experiments/README.md` - the attempt-2 ledger line (auth leg passed, tool-turn leg failed, one tiny turn spent; row stays PENDING for a passing run)
- `.planning/phases/05-interactions-full-chat-cycle-api-key/05-UAT.md` - attempt-2 result lines (merge semantics: nothing recorded was reset), item 2 pass-with-caveat, Summary recomputed (passed 1, issues 1, pending 0), the turn-latch gap recorded
- `.planning/STATE.md` - position, decisions, blocker ledger (attempt-1 read-back blocker marked resolved; turn-latch gap added), metrics, session
- `.planning/ROADMAP.md` - 05-12 row checked with the honest outcome; Phase-5 progress row updated

## Decisions Made

- **Honest split verdict over either simplification.** The run is neither a PASS (the cycle did not complete) nor a repeat of attempt 1 (the auth read-back leg now demonstrably works). UAT, ledger, and STATE all record the split: read-back fixed and field-proven; turn sequencing broken.
- **Capture monotonicity (file -02, sanctioned by the orchestrator's dispatch note).** Overwriting the committed attempt-1 capture would silently repoint the attempt-1 ledger line's evidence — the exact repudiation shape T-5-31 exists to prevent. The header's test-locked V-EXP id (`captureID = "EXP-APIKEY-CYCLE-01"` in apicycle_test.go:117) stays authoritative in the ledger line, which names the -02 file explicitly.
- **Row stays PENDING.** The ledger's own discipline (restated by attempt-1's precedent): the row flips to actuals only on a passing run; failing attempts append failure lines beneath it.
- **Scan before commit, twice.** scan-secrets ran on the working tree during scrub verification AND on the staged diff immediately before the evidence commit (the staged capture enters `git grep`'s tracked set — the gate genuinely covered the new bytes).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Attempt-2 capture file named EXP-APIKEY-CYCLE-02.jsonl instead of the plan's EXP-APIKEY-CYCLE-01.jsonl**
- **Found during:** Task 3 (scrub step)
- **Issue:** the plan's Task 3 `files`/artifact list names EXP-APIKEY-CYCLE-01.jsonl, but attempt-1's capture already occupies that path as committed evidence (d15a745) referenced by the attempt-1 ledger line; scrubbing attempt 2 over it would falsify that pointer.
- **Fix:** scrubbed to EXP-APIKEY-CYCLE-02.jsonl; the ledger line names the file and explains the header-id/file relationship. Sanctioned by the orchestrator's dispatch note ("EXP-APIKEY-CYCLE-02 or the plan's named id") and confirmed by the user with the evidence.
- **Files modified:** experiments/captures/EXP-APIKEY-CYCLE-02.jsonl
- **Verification:** attempt-1 capture unchanged (`git log -1 --stat d15a745` file list untouched by this plan); scan-secrets exit 0
- **Committed in:** 3c80488

---

**Total deviations:** 1 (evidence-integrity naming; zero scope creep, zero production code touched)
**Impact on plan:** none on the plan's outcomes — the evidence trail is complete and strictly additive.

## Issues Encountered

- **The REAL RUN itself failed again, at a NEW leg** (this is the plan's honest outcome, not an execution problem): the desktop latches one turn at a time, and the guarded test issues its second `session/send` without waiting for the first turn's terminal surface. Candidate fix (next gap plan): after the tool-less turn resolves, await the first turn's terminal/idle state patch (or end_turn carrier) before sending the file-read turn — the 03-05 turn machine's terminal classification already provides the polling surface.
- `gsd_run` is not on PATH in this executor's shells — invoked via its absolute path (`$HOME/.zcode/gsd-core/bin/gsd_run`) for the state handlers. No functional impact (same as 05-11).
- The scrub initially left 28 lines carrying the test's generic temp-dir paths (`/tmp/TestAPIKeyFullCycle…`, `/tmp/zcode-acp-auth-probe-…`) — re-scrubbed from the raw with those exact `--redact-path` prefixes to match the committed-capture precedent (every existing capture carries zero `/tmp/` paths); raw capture untouched.

## User Setup Required

None new. The existing [05-USER-SETUP.md](./05-USER-SETUP.md) recipe gates the next attempt unchanged. The raw capture at `/tmp/zcode-acp-uat/apikey-cycle-02.raw.jsonl` remains the user's to keep or delete — the committed evidence is the scrubbed derivative.

## Next Phase Readiness

- **The phase stays gaps_found.** SC4's real half is one leg short: the real-tier cycle now provably authenticates and completes a real turn, but the two-turn shape (tool-less + file-read) is blocked by the desktop's turn latch.
- **Next gap plan (the charter this SUMMARY hands over):** teach `TestAPIKeyFullCycle` to wait for the first turn's terminal state before the second send (latch-drain), then re-run the SAME human-gated checkpoint recipe as attempt 3. The probe/verdict machinery needs no changes — the fix is test-side sequencing.
- **AUTH-01 stays `[ ]`** — no mark-complete was called; the shared-ID gate stays blocked. REQUIREMENTS.md is untouched by this plan. The restore to Complete is the orchestrator's post-verification act ONLY after a passing run.
- Edge-probe carry (TOOL-01/concurrency backstop truth) stays flagged: the live run exercised the happy-path order only and the ceiling-breach/backpressure behavior remains fake-tier-locked (05-04) — for the verifier.

## Known Stubs

None — no placeholder data paths; the evidence trail is complete for what it claims (an honest failure record).

## Threat Flags

None — no security-relevant surface beyond the plan's threat model. T-5-29 held (raw capture born outside the repo and read only by the scrubber; zero credential bytes in any log, context, or committed file); T-5-30 held (the human was the only spender; actual spend recorded); T-5-31 held (a failing verdict recorded WITH its evidence trail — scrubbed capture + scan + ledger actuals — no green verdict without evidence, and no evidence without the scan gate).

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-21*
