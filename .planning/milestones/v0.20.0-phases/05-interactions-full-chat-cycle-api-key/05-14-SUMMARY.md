---
phase: 05-interactions-full-chat-cycle-api-key
plan: 14
subsystem: testing
tags: [acp, zcode, realdesktop, real-run, api-key, uat, evidence, quota-ledger, capture-hygiene, turn-latch]

# Dependency graph
requires:
  - phase: 05 (05-13)
    provides: the latch-drained guarded test (ask-tap settle + bounded invalid-request retry, UTC-stamped correlation anchors) and the §5 turn-latch addendum this run's verdict validates
  - phase: 05 (05-12)
    provides: the checkpoint round-trip pattern (executor mechanical support, human verdict + "evidence recorded" confirmation) and the evidence-monotonicity discipline
provides:
  - The attempt-3 REAL RUN verdict: PASS 49.43s — the FULL cycle is field-proven on the real z.ai api-key account through the drained test (authenticate read-back → tool-less turn → file-read tool turn → resume → close → zero orphans); hypothesis (a) validated
  - The (b)-flavored observation recorded explicitly: the desktop's true turn-1 completion surfaces arrived ~16.4 s AFTER our quiet-floor end_turn; ask-silence does not correlate with latch release — the §5 addendum's open question now has answer data
  - The complete evidence trail: scrubbed capture EXP-APIKEY-CYCLE-03.jsonl (NEW monotonic file, scan green twice) + the ledger row flipped PENDING → actuals on the first passing run + UAT attempt-3 records with the turn-sequencing gap CLOSED
affects: [/gsd:verify-work, orchestrator post-verification acts (AUTH-01 restore + 05-VERIFICATION.md refresh), any next gap cycle / Phase-7 hardening]

# Actuals (#2632) — same estimateTokens scale (chars/4 over the realized diff).
actuals:
  tokens: 379053    # measured at write: 1497600 git-diff chars (b45f086..4bc2abe) + 18613 SUMMARY bytes, / 4 — the scrubbed capture (1.48 MB) dominates; the 18000-token estimate priced the run bookkeeping, not the capture bytes
  tasks: 3
  commits: 1       # MEASURED: git rev-list --count b45f086..HEAD at SUMMARY write (the docs close-out commit follows, precedent: 05-12)
plan_head_before: b45f086f05c7c5102d8e5bbe8c4ef040c567c53b
plan_head_after: 4bc2abe3e6988ba2ec929de9d36e46960a80031b

# Tech tracking
tech-stack:
  added: []        # zero new dependencies — the plan ran the existing guarded test and the existing scrub pipeline only (T-5-SC held)
  patterns:
    - "human-gated spending runs: the executor never runs the spending command; the verdict arrives as the user's in-chat message; evidence lands through the scrub-first pipeline with the human confirming the scrubbed content before any commit"

key-files:
  created:
    - experiments/captures/EXP-APIKEY-CYCLE-03.jsonl
    - .planning/phases/05-interactions-full-chat-cycle-api-key/05-14-SUMMARY.md
  modified:
    - experiments/README.md
    - .planning/phases/05-interactions-full-chat-cycle-api-key/05-UAT.md
    - .planning/STATE.md
    - .planning/ROADMAP.md

key-decisions:
  - "Branch decision applied explicitly, no silent divergence: hypothesis (a) VALIDATED (the drain sufficed — no -32010 ever surfaced), with the (b)-flavored observation recorded alongside it: the desktop's true turn-1 completion surfaces (operation-event turn-completed 19:29:48.740Z + state.updated rev 3 prompt_completed 19:29:49.093Z) arrived ~16.4 s AFTER our quiet-floor end_turn, and ask-silence did not correlate with latch release — the PASS stands and the observation feeds the §5 addendum's open question"
  - "Ledger row flipped PENDING → actuals on this first passing run per the row's own discipline; the attempt-3 line appends beneath attempts 1-2 + DIAGNOSTIC with spend actuals (one registry read-back + two tiny completed turns, the D-05 minimum), never credentials"
  - "Capture monotonicity: the attempt-3 capture scrubbed to a NEW file EXP-APIKEY-CYCLE-03.jsonl — EXP-APIKEY-CYCLE-01/-02 stay byte-intact so every earlier ledger pointer holds; the header's test-locked V-EXP id EXP-APIKEY-CYCLE-01 is recorded with the -03 filename named explicitly"
  - "Hygiene held end-to-end: the raw capture lived at /tmp/zcode-acp-uat/apikey-cycle-03.raw.jsonl (mode 600, outside the repo) and was read ONLY by experiments/scrub-capture.mjs; the key was exported into the manual-run shell only via the tools/keypeek-tmp helper; exactly ONE spending run per the checkpoint resolution"

patterns-established:
  - "Settle-budget exhaustion is honest: the drain proceeds to its bounded invalid-request retry rather than masking a live ask loop — the retry phase carried this run (send accepted first-try at settle expiry + 124 µs)"
  - "Branch-decision recording: a PASS verdict and its (b)-flavored evidence are recorded together — the implemented fix and the evidence never silently diverge"

requirements-completed: []  # deliberate: the plan's prohibition is always-binding — REQUIREMENTS.md is untouched; AUTH-01's restore to Complete is the orchestrator's post-verification act ONLY after this passing run (no mark-complete called)

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Pre-flight automated gate: the drained tree green with both double-guard directions proven before any human run"
    requirement: AUTH-01
    verification:
      - kind: integration
        ref: "mise run ci (27.51s green: build/vet/lint/test/test-experiments, lingering-latch corpus row included) + tagged run without ZCODE_ACP_REAL_E2E=1 skips honestly (--- SKIP, 0.005s) + no-tag build reports [no test files] + scan-secrets exit 0 + production-scoped diff vs 0dfce4f prints internal-clean (Task 1)"
        status: pass
    human_judgment: false
  - id: D2
    description: "REAL RUN attempt 3 executed by the human on the real z.ai api-key account through the drained test — verdict PASS 49.43s, hypothesis (a) validated with the (b)-flavored observation recorded"
    requirement: AUTH-01
    verification: []
    human_judgment: true
    rationale: "The run spends real API budget and is human-executed per the blocking-human checkpoint gates; the verdict arrived as the user's in-chat message with the go test -v UTC-stamped anchors (settle exhaustion 19:30:02.473499283Z, acceptance 19:30:10.670248909Z, tool frames, zero orphans); no automated tier may re-run it"
  - id: D3
    description: "Evidence trail: scrubbed capture EXP-APIKEY-CYCLE-03.jsonl + clean secret scan (twice) + ledger row flip to actuals + UAT attempt-3 records with the gap CLOSED"
    requirement: AUTH-01
    verification:
      - kind: other
        ref: "bash experiments/scan-secrets.sh — exit 0 on the tree AND on the staged diff immediately before the evidence commit (449/449 lines scrubbed, zero secret shapes, zero /tmp//home residue, idempotent fixed point; -01/-02 byte-intact)"
        status: pass
    human_judgment: true
    rationale: "Credential hygiene and quota accounting are human judgment acts on machine-local material (UAT item 2 why_human) — the user reviewed the scrubbed content and confirmed 'evidence recorded' on 2026-11-21 before the commit"

# Metrics
duration: 8h 47min (wall, includes the human round trip)
completed: 2026-11-21
status: complete
---

# Phase 5 Plan 14: REAL RUN Attempt 3 (Drained Test) & Evidence Recording Summary

**REAL RUN attempt 3 PASSED (49.43s) on the real z.ai api-key account through the drained guarded test — hypothesis (a) validated: the full cycle (authenticate read-back → tool-less turn → file-read tool turn → resume → close → zero orphans) is field-proven, the evidence trail is committed, and the Phase-5 ledger row flipped to actuals.**

## Performance

- **Duration:** 8h 47min wall (includes the human-executed run + the two checkpoint round-trips)
- **Started:** 2026-11-21T11:12:23Z (plan-head ledger birth, post-05-13)
- **Completed:** 2026-11-21T19:59:49Z
- **Tasks:** 3 (Task 1 gate run by the executor; Task 2 executed BY THE USER per the blocking-human gate; Task 3 evidence recorded on the user's "evidence recorded" confirmation)
- **Files modified:** 6

## Accomplishments

- **The full cycle is FIELD-PROVEN on the real account.** Attempt 3 passed every leg: authenticate{zai-api-key} proven by the registry read-back verdict (never the ack); session/new + D-03 subscribe gate + setModel zai-api/GLM-5.3-Flash; the tiny tool-less turn completed end_turn with zero tool frames; the file-read tool turn was ACCEPTED FIRST TRY through the 05-13 latch drain — tool_call frames `call_369286416d214a268f952583` pending→in_progress→completed (TOOL-01 live); session/resume reattached the same session with no replay; session/close tore down clean with the post-close prompt rejected coded-clean; staged shutdown left zero orphans (kill(-pgid,0) ESRCH). Hypothesis (a) VALIDATED: the desktop's end_turn was genuine and the latch needed draining — the 05-13 fix worked.
- **Real spend happened and is ledgered:** one registry read-back + two tiny completed turns — the D-05 minimum. No -32010 ever surfaced, so zero refused sends. The Phase-5 quota-ledger row FLIPPED from PENDING to actuals (its first passing run after attempts 1-2 + DIAGNOSTIC).
- **The (b)-flavored observation is recorded explicitly (the branch rules' PASS clause):** the desktop's TRUE turn-1 completion surfaces arrived ~16.4 s AFTER our quiet-floor end_turn (computer-use/operation-event kind=turn-completed at 19:29:48.740Z + state.updated rev 3 reason=prompt_completed at 19:29:49.093Z — attempt 2 observed NO terminal surface; both carriers exist on this run), and turn-2's send found the latch already released 13.4 s after that patch. Ask-silence does not correlate with latch release (the ask loop outlived the completion patch ~13 s; into turn 2 until 19:30:10.907Z). The PASS stands; this answers the §5 addendum's open question — terminal surfaces EXIST on 3.18.2, arriving ~16–17 s after the quiet floor — and the data is handed to the verifier/next gap cycle.
- **Evidence trail committed with the twice-scan discipline:** scrubbed capture `experiments/captures/EXP-APIKEY-CYCLE-03.jsonl` (NEW monotonic file; 449/449 lines; 244 distinct `<uuid-N>` placeholders, 79 `[REDACTED-CREDENTIAL]`, 272 `<path>`; zero `/tmp/`/`/home/` residue; idempotent fixed point; header byte-identical; EXP-APIKEY-CYCLE-01/-02 byte-intact) + scan-secrets exit 0 on the tree AND the staged diff + UAT attempt-3 records with the TOOL-01/turn-sequencing gap CLOSED.

## The attempt-3 verdict (human-executed 2026-11-21, 49.43s)

```text
--- PASS: TestAPIKeyFullCycle (49.43s)
Session sess_58aaed32-9f68-430a-b903-33b67c5cbcc7 (scrubbed to <uuid-N> in the capture)
Same desktop install: 3.18.2, rev-44 builtin (zcode-builtin:44:e9c0674a…)
1. authenticate read-back proven (never the ack)
2. session/new + subscribe gate + setModel zai-api/GLM-5.3-Flash
3. turn 1 resolved at 2026-11-21T19:29:32.410159373Z stop=end_turn (quiet-floor per watcher.go floor math)
4. latch drain: settle budget (30s) exhausted at 19:30:02.473499283Z (last ask 19:30:02.230578528Z)
   → retry phase; send hit the wire 19:30:02.473622947Z (attempt 1) — ACCEPTED, no -32010 this run
5. file-read tool turn ok: call_369286416d214a268f952583 pending→in_progress→completed (TOOL-01 live)
6. leg resolved 19:30:10.670248909Z ("prompt accepted (attempt 1)")
7. session/resume ok (same session, re-issued subscribe accepted, no replay)
8. session/close ok (teardown clean, post-close prompt rejected coded-clean)
9. no orphaned group members (kill(-pgid,0) ESRCH)
Branch decision: hypothesis (a) VALIDATED — the drain sufficed; (b)-flavored observation recorded (see Accomplishments).
```

## Task Commits

1. **Task 1: pre-flight automated gate** — verification-only, no commit (evidence in the first checkpoint return: mise ci 27.51s green, double guard proven both ways, scan-secrets clean, `internal-clean` vs 0dfce4f)
2. **Task 2: REAL RUN attempt 3** — executed BY THE USER at the blocking-human checkpoint; no executor commit (the spending command never ran under executor control, exactly one run)
3. **Task 3: evidence recording** — `4bc2abe` (docs: scrubbed capture + ledger row flip to actuals + UAT attempt-3 records; scan-secrets green on the tree AND the staged diff; committed after the user's "evidence recorded" confirmation)

**Plan metadata:** committed with the docs close-out commit (SUMMARY + STATE + ROADMAP).

## Files Created/Modified

- `experiments/captures/EXP-APIKEY-CYCLE-03.jsonl` - the attempt-3 scrubbed capture (NEW monotonic file; header V-EXP id EXP-APIKEY-CYCLE-01 is the test-locked captureID; -01/-02 byte-intact)
- `experiments/README.md` - the Phase-5 row flipped PENDING → actuals + the attempt-3 line beneath attempts 1-2 + DIAGNOSTIC (spend actuals + verdict + the (b)-flavored evidence, never credentials)
- `.planning/phases/05-interactions-full-chat-cycle-api-key/05-UAT.md` - attempt-3 result lines (item 1 pass, item 2 pass; merge semantics — nothing recorded was reset), Summary recomputed (passed 2, issues 0), the turn-sequencing gap CLOSED with the residual-evidence note
- `.planning/STATE.md` - position advance, decisions, the turn-latch blocker marked resolved, metrics, session
- `.planning/ROADMAP.md` - 05-14 row + Phase-5 progress updated
- `.planning/phases/05-interactions-full-chat-cycle-api-key/05-14-SUMMARY.md` - this file

## Decisions Made

- **Branch decision applied explicitly.** Hypothesis (a) validated — the drain sufficed; the (b)-flavored observation (true completion surfaces ~16.4 s late; ask-silence ≠ latch release) is recorded alongside the PASS, not instead of it. No silent divergence between the implemented fix and the evidence; the drain stays in the test (correct under both hypotheses).
- **Row flip on the first passing run.** The ledger's own discipline (restated by attempts 1-2): the row flips to actuals only on a passing run — attempt 3 is that run.
- **Capture monotonicity.** The attempt-3 capture is a NEW file (-03); the committed -01/-02 are byte-intact so the earlier ledger lines' evidence pointers never repoint (the 05-12 T-5-31 lesson held).
- **Scan before commit, twice.** scan-secrets ran on the working tree during scrub verification AND on the staged diff immediately before the evidence commit (the staged capture entered `git grep`'s tracked set — the gate genuinely covered the new bytes).

## Deviations from Plan

None - plan executed exactly as written. (The plan already named EXP-APIKEY-CYCLE-03.jsonl explicitly — the 05-12 naming deviation is not repeated; both checkpoint gates resolved by the user exactly as designed.)

## Issues Encountered

- **The settle budget (30 s) exhausted honestly** with turn-1's cipher-ask loop still live (last ask 19:30:02.230Z, ~37 s total — longer than attempt-2's ~7.5 s latch-hold). By design the drain proceeded to its bounded retry; the send was accepted first-try. Recorded as evidence, not a defect: the desktop's post-completion behavior varies run-to-run.
- **Wire-vs-log timing clarified for the record:** the drain's "prompt accepted (attempt 1)" log instant (19:30:10.670248909Z) is front.Prompt's return (the leg's resolution, matching the turn-completed operation event 19:30:10.670005953Z); the wire send itself went at 19:30:02.473622947Z. The ledger/UAT records use the wire instants so the evidence is tee-correlatable.
- `gsd_run` is not on PATH in this executor's shells — invoked via its absolute path (`$HOME/.zcode/gsd-core/bin/gsd_run`) for the state handlers. No functional impact (same as 05-11/05-12).

## User Setup Required

None new. The existing [05-USER-SETUP.md](./05-USER-SETUP.md) recipe gated this run unchanged (keyring path-string-hash check; key exported via `tools/keypeek-tmp` into the manual-run shell only; raw capture born outside the repo). The raw capture at `/tmp/zcode-acp-uat/apikey-cycle-03.raw.jsonl` remains the user's to keep or delete — the committed evidence is the scrubbed derivative.

## Next Phase Readiness

- **The phase's real half closes here.** The guarded full cycle is field-proven on the real z.ai api-key account: the run verdict (PASS) + the evidence trail (scrubbed -03 capture + V-EXP id + ledger actuals) + the branch decision stated are exactly the phase gate the plan's verification section defines for the PASS path.
- **Phase closure path (PASS branch) rides the orchestrator's post-verification acts:** the 05-VERIFICATION.md refresh and the AUTH-01 restore in REQUIREMENTS.md. This plan did NOT touch REQUIREMENTS.md and did NOT call mark-complete (the shared-ID gate untouched; the prohibition held).
- **The §5 addendum's open question now has answer data** (terminal surfaces ~16-17 s post-quiet-floor; ask-silence ≠ latch release) — for the verifier, and as candidate input for a Phase-7 hardening pass (a completion-wait behind the drain would tighten the early-end_turn skew; the drain itself stays, correct under both hypotheses).

## Known Stubs

None — no placeholder data paths; the evidence trail is complete for what it claims (a passing run with its full wire evidence).

## Threat Flags

None — no security-relevant surface beyond the plan's threat model. T-5-34 held (raw capture born outside the repo and read only by the scrubber; zero credential bytes in any log, context, or committed file; the keypeek key export stayed in the manual-run shell). T-5-35 held (the human was the only spender; exactly ONE guarded run per checkpoint resolution; actual spend = one read-back + two tiny turns). T-5-36 held (the green verdict is recorded WITH its evidence trail — scrubbed monotonic capture + twice-scan + ledger actuals; no evidence pointer repoints).

## Self-Check: PASSED

- `experiments/captures/EXP-APIKEY-CYCLE-03.jsonl` exists on disk and in git (committed in `4bc2abe`, mode 100644 in the index / 600 on disk)
- `05-14-SUMMARY.md` exists; the evidence commit `4bc2abe` and the docs close-out commit are ancestors of HEAD at final verification
- `bash experiments/scan-secrets.sh` exit 0 on the final tree (and on the staged diff immediately before the evidence commit — the twice-scan discipline held through every staged diff)
- `EXP-APIKEY-CYCLE-01.jsonl` and `-02.jsonl` are byte-intact across this plan's range (`git diff b45f086..4bc2abe` over both paths is empty) — evidence monotonicity held
- `REQUIREMENTS.md` untouched — AUTH-01 stays `[ ]`, no mark-complete called, the restore rides the orchestrator's post-verification act
- The working tree carries no unstaged tracked changes at close-out; the raw capture was never read by anything but the scrubber

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-21*
