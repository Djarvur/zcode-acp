---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 09
subsystem: docs
tags: [uat, manual-e2e, vm-scenario, qa-gates, experiments-ledger, handoff]

# Dependency graph
requires:
  - phase: 07-hardening (07-07)
    provides: scripts/vm-e2e.sh — the QA-02 deliverable the runbook stages (run stage re-proven 5 PASS / 0 skipped this plan)
  - phase: 07-hardening (07-08)
    provides: the README launch recipe (ONE override + ONE env mechanism per auth path) the QA-03 checklists ride
  - phase: 07-hardening (07-04)
    provides: the 18.1.9+zacp.0 version stamp the pre-verification re-proved at both CLI surfaces
  - phase: 06-oauth-coding-plan-path-model-picker
    provides: the 06-UAT.md recording shape mirrored, the keyring sentinels, the PAT chain facts the QA-03 checklist cites
provides:
  - 07-UAT.md — the end-of-phase UAT: QA-02 (VM field run) + QA-03 (Orca card E2E, both auth paths) seeded awaiting-user with copy-pasteable runbooks, per-path checklists verbatim from RESEARCH Pattern 8, the two observation items (A1 --model id shape, A3 picker behavior), and honest-recording rules (human confirmation precedes any evidence commit)
  - experiments/README.md Quota ledger — QA-02 (zero API spend) and QA-03 (the D-05 minimum per path) rows seeded PENDING with capture discipline
  - the recorded pre-verification: phase gate green (mise run ci exit 0, go vet -tags realdesktop clean, --version 18.1.9+zacp.0 at root and under omp, vm-e2e run stage 5 PASS / 0 skipped)
  - WINDOWS.md unrun-verify entries for both human gates (the 06 precedent) so the ship gate sees the open items
affects: [verify-work (QA-02/QA-03 end-of-phase harvest), the v1 milestone close, the orchestrator's post-verification REQUIREMENTS.md flips]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 3959      # 15835 diff chars / 4 over 07-UAT.md (new) + experiments/README.md (mod)
  tasks: 3
  commits: 3        # MEASURED: git rev-list --count 6b3f1825..HEAD

# Tech tracking
tech-stack:
  added: []         # planning-tier recording only — no new dependencies (T-07-SC: no package installs)
  patterns:
    - "honest-awaiting recording: manual-gate entries are seeded result: awaiting with the full runbook staged BEFORE handoff; verdicts only from the user's actual runs, human confirmation precedes any evidence commit (T-07-25)"
    - "handoff staging proof: every runnable fact the runbook names was re-proven on the dev machine at staging time (bash -n, per-stage live runs, header build-command match)"

key-files:
  created:
    - .planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md
  modified:
    - experiments/README.md

key-decisions:
  - "Requirements stay untouched: requirements.ready-ids reports QA-02/QA-03 READY (blocked: none) but mark-complete was NOT run — the checkbox flips are the orchestrator's post-verification act because the manual runs have not happened (T-07-25, the 05-12 precedent)"
  - "WINDOWS.md unrun-verify entries recorded for both human gates at handoff time — the ship gate must see the open items even after the SUMMARY scrolls out of context"
  - "Task 2/3 preconditions (clean VM, Orca + real account) are user_setup facts gated on the RUN, not the preparation — the executor staged the runbooks and returned awaiting per the plan's own done criteria, never touching the manual runs (D-05/D-06)"

patterns-established:
  - "Manual-gate handoff shape: seed UAT entry + ledger row PENDING -> stage the exact commands -> prove every runnable named fact -> return awaiting-user for end-of-phase harvest"

requirements-completed: []  # QA-02/QA-03 stay OPEN by design: ready-ids reports both READY; the flips happen post-verification (orchestrator act)

# Coverage metadata (#1602) — per-deliverable traceability for verify-work routing.
coverage:
  - id: D1
    description: "Phase gate pre-verification green before handoff: mise run ci exit 0, go vet -tags realdesktop clean, zcode-acp [--omp] --version prints 18.1.9+zacp.0, vm-e2e run stage 5 PASS / 0 skipped"
    verification:
      - kind: other
        ref: "mise run ci (exit 0, 2026-11-22 this plan)"
        status: pass
      - kind: other
        ref: "go vet -tags realdesktop ./test/realdesktop/ (exit 0, no diagnostics)"
        status: pass
      - kind: other
        ref: "zcode-acp --version && zcode-acp omp --version (both print zcode-acp 18.1.9+zacp.0)"
        status: pass
      - kind: other
        ref: "bash scripts/vm-e2e.sh --stage run --binary <built> (5 PASS, 0 skipped, exit 0)"
        status: pass
    human_judgment: false
  - id: D2
    description: "07-UAT.md QA-02 entry: copy-pasteable VM runbook staged before handoff (build off-VM, transfer, full run + per-stage + --require-auth variants), result awaiting-user; ledger row PENDING"
    requirement: QA-02
    verification:
      - kind: other
        ref: "bash -n scripts/vm-e2e.sh (clean) + --stage discovery PASS live + header build-or-fetch instructions match the actual build command"
        status: pass
      - kind: other
        ref: "grep -c 'vm-e2e' 07-UAT.md (14 >= 1)"
        status: pass
    human_judgment: true
    rationale: "QA-02's field proof is ONE documented user run of the script on an actual clean Ubuntu GNOME VM (D-05) — the executor can never provision or run the VM; the run executes under user control at end-of-phase harvest (07-UAT.md Test 1), verdict recorded honestly from the script's summary output"
  - id: D3
    description: "07-UAT.md QA-03 entry: both per-path checklists verbatim from RESEARCH Pattern 8 (api-key + coding-plan), both-paths steps (/model switch, --model launch binding, clean Stop/interrupt), the A1/A3 observation items, recording+spend+scrub discipline staged; result awaiting-user; ledger row PENDING"
    requirement: QA-03
    verification:
      - kind: other
        ref: "grep -c 'zai-api-key\\|zai-individual-coding-plan' 07-UAT.md (2 >= 2)"
        status: pass
      - kind: other
        ref: "read-only precondition facts: Orca.AppImage present on this machine, README launch-recipe section present, zcode-acp omp --version prints 18.1.9+zacp.0"
        status: pass
    human_judgment: true
    rationale: "QA-03 is THE user gate (D-06): a working agent card in real Orca on a real z.ai account through BOTH auth paths — never autonomous; the run executes under user control at end-of-phase harvest (07-UAT.md Test 2) with verdicts per path plus the A1/A3 observations recorded from the actual runs"
  - id: D4
    description: "experiments/README.md Quota ledger: QA-02 (zero API spend) and QA-03 (D-05 minimum per path) rows seeded PENDING referencing the UAT entries, with capture discipline (raw outside repo, scrubber-only entry, scan-secrets on tree AND staged diff)"
    verification:
      - kind: other
        ref: "grep -c 'QA-02\\|QA-03' experiments/README.md (>= 2 new PENDING rows present)"
        status: pass
      - kind: other
        ref: "bash experiments/scan-secrets.sh && bash experiments/check-docs.sh (both exit 0 at every task commit)"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-11-22
status: complete
plan_head_before: 6b3f18251133648535a3dd07e507ccf22ea2ff27
plan_head_after: 9a6a35d2c2149f252dc29c4faa952f1d4918418b
---

# Phase 7 Plan 09: VM & Manual E2E, Documentation Summary

**The done-v1 handoff ships: phase gate pre-verified green end-to-end, and the two manual gates (QA-02 VM field run, QA-03 Orca card E2E through both auth paths) are staged as copy-pasteable runbooks in 07-UAT.md + PENDING ledger rows — awaiting the user's runs at end-of-phase, never executed by the executor (D-05/D-06).**

## Performance

- **Duration:** 12 min
- **Started:** 2026-11-22T14:35:36Z
- **Completed:** 2026-11-22T14:47:52Z
- **Tasks:** 3
- **Files modified:** 2 (07-UAT.md new; experiments/README.md ledger rows)

## Accomplishments
- Everything automatable proven before handoff: `mise run ci` exit 0 (build + vet + lint 0 issues + race suite + experiments suite), `go vet -tags realdesktop ./test/realdesktop/` clean (the guarded tier compiles), `zcode-acp --version` and `zcode-acp omp --version` both print `18.1.9+zacp.0` (the card floor), and the vm-e2e run stage passed 5/5 checks live on this machine (one minimal real turn completed — the plan's sanctioned pre-verification)
- 07-UAT.md seeded mirroring the 06-UAT.md shape (front matter status: awaiting-user, Current Test, numbered Tests with expected/awaiting/result, Summary counts, Gaps): Test 1 = QA-02 with the copy-pasteable VM sequence (build off-VM, transfer, full run + five per-stage invocations + `--require-auth` hardening), Test 2 = QA-03 with both per-path checklists verbatim from RESEARCH Pattern 8, the both-paths steps, and the two OBSERVATION items (A1: the `--model` id shape Orca passes, read from the wrapper's launch log; A3: picker behavior with configOptions but no models field) — recorded, never assumed
- experiments/README.md Quota ledger carries the two PENDING rows referencing the UAT entries: QA-02 zero API spend, QA-03 the D-05 minimum per path (one registry read-back + two tiny completed turns per path), with the full capture discipline (raw captures outside the repo, scrubber-only entry, scan-secrets on tree AND staged diff, human confirmation precedes any evidence commit)
- Both human gates recorded in the cross-phase broken-windows ledger (.planning/WINDOWS.md, unrun-verify) per the phase-6 precedent, so the ship gate sees the open items
- REQUIREMENTS.md untouched: `requirements.ready-ids` reports QA-02 and QA-03 READY (blocked: none) — the checkbox flips remain the orchestrator's post-verification act

## Task Commits

Each task was committed atomically:

1. **Task 1: Pre-verification + UAT scaffold** - `a6d95ed` (docs) — gates green + 07-UAT.md seeded + two ledger rows
2. **Task 2: QA-02 manual gate staged** - `ac39e1a` (docs) — the copy-pasteable VM runbook into the UAT entry
3. **Task 3: QA-03 manual gate staged** - `9a6a35d` (docs) — per-path checklists + observation items into the UAT entry

## Files Created/Modified
- `.planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md` — the end-of-phase UAT: both manual gates awaiting-user with full runbooks and honest-recording rules
- `experiments/README.md` — the QA-02/QA-03 Quota-ledger rows seeded PENDING with spend + capture discipline
- `.planning/WINDOWS.md` — two unrun-verify entries (the human-gated runs, open until the end-of-phase harvest)

## Decisions Made
- Requirements not flipped: ready-ids reports QA-02/QA-03 READY, but the manual runs have not happened — flipping them would fabricate the done-v1 bar (T-07-25); mark-complete was deliberately not run and the ready-ids result is recorded here for the orchestrator's post-verification act
- Tasks 2/3 preconditions treated as run-gated, not preparation-gated: the clean VM and the Orca/real-account facts are user_setup for the RUN itself (the plan's done criteria accept "honestly recorded as awaiting"); the executor verified the checkable facts read-only (Orca.AppImage present on this machine, README recipe present, version floor green) and staged the runbooks
- No mid-flight checkpoint: human_verify_mode=end-of-phase — the plan tasks carry `<human-check>` blocks harvested by the verifier; the runbooks staged in 07-UAT.md make the harvest self-sufficient

## Deviations from Plan

None - plan executed exactly as written. (The manual runs not being executed by this plan is the plan's designed outcome, recorded as awaiting — not a deviation.)

## Issues Encountered
None material — every pre-verification gate was already green from plans 07-01..07-08; this plan re-proved them and staged the handoff.

## User Setup Required

The two manual gates ARE the user setup (plan frontmatter `user_setup`), harvested at end-of-phase:

- **QA-02 — one documented VM field run** (`.planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md` Test 1): provision a clean Ubuntu GNOME VM, build the binary off-VM (`go build -o zcode-acp ./cmd/zcode-acp`) or fetch a release artifact, transfer script + binary, run `bash ~/vm-e2e.sh --binary ~/zcode-acp`; paste the summary output into the entry and confirm "evidence recorded" before any evidence commit.
- **QA-03 — the Orca card E2E, BOTH auth paths** (07-UAT.md Test 2): Orca.AppImage (present on this machine) + the real z.ai account (api-key value for path 1; desktop logged into the individual coding plan with unlocked keyring for path 2); follow the README launch recipe; record verdicts per path + the A1/A3 observations.

No USER-SETUP.md generated — the runbooks live in 07-UAT.md and the `<human-check>` blocks in 07-09-PLAN.md tasks 2-3 are the harvest items.

## Handoff for End-of-Phase Harvest

- The two `<human-check>` blocks in this plan (tasks 2 and 3) are the verifier's harvest items; 07-UAT.md carries the same runbooks user-side
- After each run: flip the UAT result field honestly from the actual output, update the ledger row actuals, run `bash experiments/scan-secrets.sh` on the tree AND the staged diff, record the human's confirmation — only then any evidence commit
- A failed leg records the honest verdict (machine findings to the verifier; a fix would be a new gap plan) — the phase stays open per the 05-14 discipline
- REQUIREMENTS.md flips (QA-02/QA-03 checkboxes + traceability rows) are the orchestrator's post-verification act; ready-ids already reports both READY

## Next Phase Readiness
- Phase 7's automated surface is fully green (SESS/DOCS/QA-01/QA-02-script); the manual gates are the only open items, properly recorded in the UAT, the ledger, and WINDOWS.md
- This was the final plan of phase 7 (9 of 9); the phase closes through end-of-phase verification (verify-work) harvesting the two human gates
- No stubs, no skipped automated verifications, no deferred items introduced

## Self-Check: PASSED

All created files exist (07-UAT.md, 07-09-SUMMARY.md, experiments/README.md, WINDOWS.md); all three task commits are ancestors of HEAD (a6d95ed, ac39e1a, 9a6a35d); commits measured from the plan ledger: 3 (6b3f1825..9a6a35d).

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*
