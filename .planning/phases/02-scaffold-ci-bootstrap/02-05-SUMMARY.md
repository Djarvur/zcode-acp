---
phase: 02-scaffold-ci-bootstrap
plan: 05
subsystem: infra
tags: [branch-protection, github-api, gh006, gsd-config, pr-only-flow, required-checks, enforce-admins, linear-history]

# Dependency graph
requires:
  - phase: 02-scaffold-ci-bootstrap (plan 02-04)
    provides: pr-sanity.yml with the three observed check names (build / vet / lint / test / tidy, govulncheck, secret-scan) and the README protection documentation section
  - phase: 02-scaffold-ci-bootstrap (plan 02-01)
    provides: the pre-protection direct-push window (branching_strategy=none) this plan closes
provides:
  - ".planning/phases/02-scaffold-ci-bootstrap/protection-body.json — the exact, re-applicable PUT body (RESEARCH Pattern 6, live-422-verified key set), contexts character-for-character the pr-sanity job names, committed on origin master"
  - "Branch protection LIVE on Djarvur/zcode-acp master: required checks = the three pr-sanity names, enforce_admins, PR required with 0 approving reviews, force-push/deletion disallowed, linear history required — verified field-by-field via GET"
  - "GH006 negative proof: a direct push to master is rejected by the remote ('Changes must be made through a pull request' + '3 of 3 required status checks are expected') — RESEARCH Assumption A2 discharged empirically"
  - ".planning/config.json branching_strategy flip none -> phase — Phases 3+ work on gsd/phase-{phase}-{slug} branches merged via PRs (flip commit is LOCAL, rides the first Phase-3 gsd/phase-* PR)"
  - "Applied-vs-refused record: NOTHING was refused — the full body applied and verified; D-01 fallback not needed"
affects: [phase-3 (forks gsd/phase-3-* from the updated local master carrying the flip commit; merges only through green PRs), phase-4..7 (all feature code lands under the protection gate), verify-work (protection is the phase gate SCAFF-03)]

# Actuals (#2632) — pairs with the plan's estimate (20000) to calibrate future estimates.
actuals:
  tokens: 321        # 1285 diff bytes / 4 over the 2 changed files (protection-body.json + config.json)
  tasks: 2
  commits: 2         # MEASURED: git rev-list --count 001a236..e777ba6
plan_head_before: 001a2367159f23ff70c4194914703de38623c72f
plan_head_after: e777ba61c7bdad4c7869224443d385cb0a196b34

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Protection-as-payload discipline: the applied body lives in-repo (protection-body.json) and is applied/removed via one gh api command — settings documented where the README recipe can read them"
    - "Apply -> GET-verify -> negative-push-test -> config-flip ordering: a settings change is never trusted until read back AND behaviorally probed"
    - "GH006 probe: one empty commit proves enforcement empirically, then is reset --hard HEAD~1 away — no residue on master or origin"

key-files:
  created:
    - .planning/phases/02-scaffold-ci-bootstrap/protection-body.json
  modified:
    - .planning/config.json

key-decisions:
  - "contexts[] deprecated form avoided: body carries checks[] {context} objects (RESEARCH State of the Art); GET echoes both forms, comparison used checks[]"
  - "Body pushed to origin master BEFORE the APPLY step (commit ed9a0c1, run 44544600222 observed green on that sha) — the README re-apply recipe references a file that exists on origin master before protection ever goes live"
  - "Flip commit deliberately local-only (e777ba6): direct pushes to master are now rejected, so its landing path is the FIRST Phase-3 gsd/phase-* PR — Phase 3 forks from updated local master and the PR merge carries the flip to origin master"
  - "GH006 message doubles as A2 discharge: 'Changes must be made through a pull request' proves required_pull_request_reviews with 0 approvals DOES enforce PR-only merges for this solo repo"

patterns-established:
  - "Never apply protection over a red master: CI observed success on the pushed sha before the PUT was issued"
  - "Settings honesty: applied state is verified by GET field-by-field against the in-repo body; refusals would be recorded, never faked (D-01) — this plan had zero refusals"

requirements-completed: [SCAFF-03]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "protection-body.json — the exact re-applicable PUT body: four required members (required_status_checks with checks[], enforce_admins, required_pull_request_reviews, restrictions null) + allow_force_pushes/allow_deletions false, required_linear_history true, required_conversation_resolution false; the three contexts equal the pr-sanity job name: fields character-for-character"
    requirement: SCAFF-03
    verification:
      - kind: other
        ref: "jq -e '.required_status_checks.checks | map(.context)' — prints exactly [build / vet / lint / test / tidy, govulncheck, secret-scan] in workflow job order"
      - kind: other
        ref: "grep -qF cross-check loop: each of the 3 names hits both 'name: <n>' in pr-sanity.yml and '\"context\": \"<n>\"' in the body (rc=0)"
      - kind: other
        ref: "jq field audit: strict=false, enforce_admins=true, PR-review triple (dismiss/code_owner false, count 0), restrictions null, force-push/deletions false, linear true, conversation false — ALL_PATTERN6_FIELDS_OK"
      - kind: other
        ref: "git rev-list origin/master..HEAD empty after fetch — body commit ed9a0c1 pushed BEFORE the APPLY step"
    human_judgment: false
  - id: D2
    description: "Branch protection LIVE on master and GET-verified: three exact contexts, enforce_admins enabled, PR required with required_approving_review_count 0, force-push disabled, deletion disabled, linear history enabled — field-by-field match against protection-body.json (9/9 comparison booleans true)"
    requirement: SCAFF-03
    verification:
      - kind: integration
        ref: "gh api repos/Djarvur/zcode-acp/branches/master/protection --jq '{checks: [...].sort, admins, pr, approvals, force_push, deletions, linear}' -> {admins:true, approvals:0, checks:[3 names], deletions:false, force_push:false, linear:true, pr:true}"
      - kind: integration
        ref: "jq field-by-field comparison of GET response vs --slurpfile protection-body.json: checks_match, strict_match, admins_match, pr_reviews_match, restrictions_null, force_push_match, deletions_match, linear_match, conv_res_match — all true"
    human_judgment: false
  - id: D3
    description: "Direct push to master empirically REJECTED: empty probe commit pushed -> remote error GH006 'Protected branch update failed' citing PR-only + 3 expected status checks; probe commit dropped locally (reset --hard HEAD~1, verified cleaned)"
    requirement: SCAFF-03
    verification:
      - kind: integration
        ref: "plan's exact negative-test compound: push_rc=1, 'GH006' in remote output, r2=cleaned -> NEGATIVE_TEST_PASS"
    human_judgment: false
  - id: D4
    description: ".planning/config.json git.branching_strategy flipped none -> phase AFTER verification (phase_branch_template gsd/phase-{phase}-{slug} pre-existing); flip committed locally, NOT pushed (rev-list origin/master..HEAD = exactly the flip commit) — landing path: first Phase-3 gsd/phase-* PR"
    requirement: SCAFF-03
    verification:
      - kind: other
        ref: "grep -c '\"branching_strategy\": \"phase\"' .planning/config.json = 1; jq confirms branching_strategy==phase + template intact"
      - kind: other
        ref: "git rev-list origin/master..HEAD lists only e777ba6 (flip commit is local-only)"
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 5: Branch Protection Applied + Verified + Config Flip Summary

**master is protected and proven: the documented protection-body.json pushed first, the live PUT applied and GET-verified field-by-field, a real push rejected with GH006 ("Changes must be made through a pull request"), and only then branching_strategy flipped to phase — the phase's closing gate closed honestly, nothing refused**

## Performance

- **Duration:** 5 min
- **Started:** 2026-11-19T14:29:28Z
- **Completed:** 2026-11-19T14:34:30Z (production commits)
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments

- `protection-body.json` landed as the documented, re-applicable PUT payload: the four required members per RESEARCH Pattern 6 (the live-422-verified key set), modern `checks[]` form (deprecated `contexts[]` avoided), with the three contexts copied character-for-character from the pr-sanity job `name:` fields — proven in sync by the plan's grep cross-check loop (rc=0)
- Body commit `ed9a0c1` pushed to origin master BEFORE the apply (pre-protection window), with pr-sanity run 44544600222 observed success on that exact sha — protection was never applied over a red master
- Protection applied via one `gh api ... -X PUT --input protection-body.json` call and verified by GET with the plan's jq shape AND a 9-field comparison against the in-repo body — all match: contexts `build / vet / lint / test / tidy`, `govulncheck`, `secret-scan`; enforce_admins enabled; PR required with 0 approving reviews; force-push disabled; deletion disabled; linear history enabled
- **GH006 negative test passed**: an empty probe commit was rejected by the remote — `GH006: Protected branch update failed for refs/heads/master` with the reasons "Changes must be made through a pull request" and "3 of 3 required status checks are expected"; probe commit dropped locally, tree clean
- `branching_strategy` flipped `none` → `phase` in `.planning/config.json` strictly AFTER verification, committed locally as `e777ba6` — deliberately NOT pushed (the protection just applied rejects direct pushes); its landing path is the first Phase-3 `gsd/phase-*` PR
- Phase-level sanity: `bash experiments/check-docs.sh --phase-gate` exit 0 after all changes

## Applied-vs-Refused Record (D-01 fallback honesty)

**Everything applied; NOTHING refused.** The GitHub API accepted the complete PUT body in one call (rc=0) and the GET read-back matches `protection-body.json` on every field (9/9 comparison booleans true, `restrictions: null` echoed as null). No org policy, plan limit, or API error degraded any rule. The D-01 fallback (recording refusals) was not needed; the README's documented setup is exactly the live state. No success is faked anywhere in this record — the applied state is the verified state.

## Task Commits

Each task was committed atomically:

1. **Task 1: protection-body.json — exact PUT body, check names verified against pr-sanity** - `ed9a0c1` (chore) + push `001a236..ed9a0c1` to origin master (pre-protection window; pr-sanity 44544600222 success on that sha)
2. **Task 2: Apply → verify → GH006 → flip** - `e777ba6` (chore, LOCAL ONLY — flip commit; landing path: first Phase-3 `gsd/phase-*` PR). The protection apply itself is a repo-settings change, not a commit.

**Plan metadata:** SUMMARY + tracking commits follow as docs commits (local; post-protection commits to origin master must all land via PRs).

## Files Created/Modified

- `.planning/phases/02-scaffold-ci-bootstrap/protection-body.json` - the exact branch-protection PUT body (RESEARCH Pattern 6): required checks (3 contexts, strict false), enforce_admins true, 0-approval PR requirement, restrictions null, no force-push/deletions, linear history true, conversation resolution false
- `.planning/config.json` - `git.branching_strategy` `none` → `phase` (one line; `phase_branch_template` `gsd/phase-{phase}-{slug}` already present)

## Protection Verification Record

| Field | Body (intended) | GET (applied) | Match |
|-------|-----------------|---------------|-------|
| checks contexts | 3 pr-sanity names | same 3, exact strings | true |
| required_status_checks.strict | false | false | true |
| enforce_admins.enabled | true | true | true |
| PR reviews (dismiss_stale / code_owner / count) | false / false / 0 | false / false / 0 | true |
| restrictions | null | null | true |
| allow_force_pushes.enabled | false | false | true |
| allow_deletions.enabled | false | false | true |
| required_linear_history.enabled | true | true | true |
| required_conversation_resolution.enabled | false | false | true |

GH006 probe output (verbatim): `remote: error: GH006: Protected branch update failed for refs/heads/master.` / `remote: - Changes must be made through a pull request.` / `remote: - 3 of 3 required status checks are expected.` — push_rc=1, probe reset (`reset=cleaned`).

## Decisions Made

- Applied the PUT only after the body commit was on origin master AND its CI run was observed green (never apply protection over a red master — plan Task 1 action)
- Used the plan's exact negative-test compound so the GH006 evidence is captured with its own cleanup and exit-code contract in one line
- Left the flip commit unpushed by design and recorded its landing path (first Phase-3 `gsd/phase-*` PR) — pushing it directly would contradict the protection just proven
- A2 flagged assumption discharged by observation: the GH006 message "Changes must be made through a pull request" proves `required_pull_request_reviews` with `required_approving_review_count: 0` DOES enforce a PR-only merge flow for this solo repo — the probe-ledger row is resolved, not assumed

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The first jq one-liner for the field-by-field comparison had a syntax error (`.all_match:` label form); re-ran with the corrected filter — all nine comparison booleans printed true. Evidence-identical, no behavior impact.

## User Setup Required

None - no external service configuration required (gh admin session already authenticated; the protection apply was the last settings change).

## Known Stubs

None - both deliverables are complete: the payload file is the real applied body, and the config flip is live in `.planning/config.json`. No placeholder values anywhere in this plan's output.

## Next Phase Readiness (hand-off)

- **The flip commit `e777ba6` rides the first Phase-3 `gsd/phase-*` PR**: Phase 3 forks its branch from the updated LOCAL master (which carries the flip); when that PR merges (all three required checks green), the flip reaches origin master. No direct pushes to origin master exist anymore — GH006 proved it.
- **Re-apply recipe**: `gh api repos/Djarvur/zcode-acp/branches/master/protection -X PUT --input .planning/phases/02-scaffold-ci-bootstrap/protection-body.json` — the file is on origin master (`ed9a0c1`), exactly where the README's protection section points.
- **If a pr-sanity job is ever renamed**: update the job `name:` AND the corresponding context in protection-body.json in the same change (Pitfall 3 — a stale required context hangs every PR on "Expected — waiting for status" forever).
- **Phase 2 is now fully executed**: strict pinned lint (02-01/02-02/02-03), four CI workflows observed green (02-04), protected master + PR-only flow (this plan). Phases 3+ land all feature code under these gates.

## Self-Check: PASSED

- `protection-body.json` exists on disk; `ed9a0c1` and `e777ba6` verified as ancestors of HEAD
- Plan ledger MEASURED: `git rev-list --count 001a236..e777ba6` = 2 (both production commits)
- GET matches protection-body.json 9/9 fields; GH006 observed (push_rc=1, probe cleaned); flip grep count=1, local-only
- `check-docs.sh --phase-gate` exit 0 after all changes

---
*Phase: 02-scaffold-ci-bootstrap*
*Completed: 2026-11-19*
