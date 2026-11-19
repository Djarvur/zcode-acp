---
phase: 02-scaffold-ci-bootstrap
plan: 04
subsystem: ci
tags: [github-actions, pr-sanity, release, e2e-matrix, security-scheduled, mise, secret-scan, branch-protection-docs, ci-observation]

# Dependency graph
requires:
  - phase: 02-scaffold-ci-bootstrap (plan 02-01)
    provides: mise.toml pins + gate tasks (build/vet/lint/test/test-experiments/tidy-diff) and the go.mod skeleton the CI jobs exercise
  - phase: 02-scaffold-ci-bootstrap (plan 02-02)
    provides: test-locked scrubber contract (18 green tests) — the experiments suite the sanity job runs
  - phase: 02-scaffold-ci-bootstrap (plan 02-03)
    provides: corrected gate scripts (scan-secrets.sh CRED_RX-aligned + toplevel-pinned, check-docs.sh) the secret-scan job promotes
provides:
  - ".github/workflows/pr-sanity.yml — three jobs whose check names are the protection contexts: build / vet / lint / test / tidy, govulncheck, secret-scan (docs gate shares the secret-scan job)"
  - ".github/workflows/release.yml — goreleaser-on-v*-tags shape, the only contents: write, inert by construction"
  - ".github/workflows/e2e-matrix.yml — dispatch-only self-hosted GNOME skeleton (goswitch discipline, no fixture plumbing), inert without a registered runner"
  - ".github/workflows/security-scheduled.yml — weekly govulncheck (cron 0 6 * * 1) + dispatch"
  - "OBSERVED SCAFF-02 on both trigger legs: push run and PR-triggered run each green with all three check names; probe ledger's SCAFF-02 assumption discharged by observation"
  - "README CI + branch-protection documentation (required-check names, PR-only flow, gsd/phase-{phase}-{slug} branches, fresh-protection honesty note + re-apply recipe)"
affects: [02-05 (protection contexts = the observed check names; protection-body.json apply), phase-3+ (all feature code lands under these gates)]

# Actuals (#2632) — pairs with the plan's estimate (50000) to calibrate future estimates.
actuals:
  tokens: 3931        # 15725 diff bytes / 4 over the 5 changed files (4 workflows + README)
  tasks: 3
  commits: 5          # MEASURED: git rev-list --count a266eef..57d48a2 (4 authored + 1 adopted remote scaffold root via the reconciliation merge)
plan_head_before: a266eef59f0115f9846d656a8a725c194a9d4266
plan_head_after: 57d48a20044f0da4262c1be74eb3d6ddabebb680

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CI mirrors the local loop via mise tasks only — no setup-go, no golangci-lint-action, no second version pin anywhere (D-02)"
    - "Job name = required-check discipline: pr-sanity job name: fields are character-for-character the protection contexts plan 02-05 requires"
    - "Write-scope confinement: contents: read on every workflow except release.yml's single contents: write (cross-file grep gate)"
    - "Dispatch-only + serial concurrency + read-only token for anything that would ever drive the self-hosted desktop runner"

key-files:
  created:
    - .github/workflows/pr-sanity.yml
    - .github/workflows/release.yml
    - .github/workflows/e2e-matrix.yml
    - .github/workflows/security-scheduled.yml
  modified:
    - README.md

key-decisions:
  - "docs-gate shares the secret-scan job (check-docs.sh as its second step) per RESEARCH Open Question 1 resolution — no fourth required check name"
  - "e2e-matrix.yml carries goswitch's discipline only (dispatch-only, e2e-gnome serial group, self-hosted+gnome labels, 150-min watchdog, env:-indirection comment); the 15 KB AT-SPI fixture body is deliberately not copied (Phase 7 authors the real scenario)"
  - "Remote-master reconciliation via one unrelated-histories merge (not rebase, not force-push): preserves all 81 local commit hashes referenced by sibling SUMMARYs and the remote's GitHub scaffold root"
  - "SCAFF-02 probe-row discharge by observation: a real PR-triggered run was watched green on its head sha before closing the scratch PR — the PR leg is no longer assumed"

patterns-established:
  - "Local gates before push: scan-secrets.sh + check-docs.sh run green locally before every CI-bearing push"
  - "CI observation bar: a plan is done only with a GREEN observed run; both trigger legs watched with gh before close-out"

requirements-completed: [SCAFF-02, SCAFF-03]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "pr-sanity.yml with three jobs named exactly build / vet / lint / test / tidy, govulncheck, secret-scan; six mise-run gate steps in sanity (incl. test-experiments); govulncheck via mise exec; docs gate inside secret-scan; permissions contents: read; concurrency with event_name+ref; checkout@v4 x3; push branches [master]"
    requirement: SCAFF-02
    verification:
      - kind: other
        ref: "grep -cF 'name: build / vet / lint / test / tidy' (=1); 'name: govulncheck' (=1); 'name: secret-scan' (=1)"
      - kind: other
        ref: "grep -c 'mise run' (>=6, actual 8); -cF 'actions/checkout@v4' (=3); uses: lines all checkout@v4; no version: YAML keys"
      - kind: integration
        ref: "push run 44542652193 completed success, sanity 13/13 steps, govulncheck 8/8, secret-scan 6/6"
    human_judgment: false
  - id: D2
    description: "release.yml inert-by-construction: v* tags only (tags: ['v*'] on one line), sole contents: write, fetch-depth 0, serial release group, mise-installed goreleaser release --clean with GITHUB_TOKEN, registration note, English provenance header"
    requirement: SCAFF-02
    verification:
      - kind: other
        ref: "grep -cF \"tags: ['v*']\" (=1); -cF 'contents: write' (=1); fetch-depth: 0 (=1); cancel-in-progress: false (=1)"
      - kind: other
        ref: "gh run list: zero release workflow runs (no v* tag exists — inert)"
    human_judgment: false
  - id: D3
    description: "e2e-matrix.yml dispatch-only skeleton: workflow_dispatch only, contents: read, serial e2e-gnome group, runs-on: [self-hosted, gnome], 150-min watchdog, placeholder step, fork-event + env:-indirection discipline comments; no goswitch fixture plumbing"
    requirement: SCAFF-02
    verification:
      - kind: other
        ref: "grep -cF 'workflow_dispatch' (>=1); -cF 'runs-on: [self-hosted, gnome]' (=1); permissions block present"
      - kind: other
        ref: "gh run list: zero e2e-matrix runs (dispatch-only, none issued — inert)"
    human_judgment: false
  - id: D4
    description: "security-scheduled.yml weekly govulncheck: cron '0 6 * * 1' + workflow_dispatch, contents: read, mise exec govulncheck @latest, English provenance header"
    requirement: SCAFF-02
    verification:
      - kind: other
        ref: "grep -cF '0 6 * * 1' (=1); contents: read (=1); mise exec govulncheck line (=1)"
      - kind: other
        ref: "gh run list: zero security-scheduled runs (cron/dispatch never fired in-window — inert by schedule)"
    human_judgment: false
  - id: D5
    description: "README CI/branch-protection section: four workflows table, mise.toml single-version-source note, the three exact required-check names, PR-only flow + gsd/phase-{phase}-{slug} branches, fresh-protection honesty note with gh api re-apply recipe; docs/protocol footer style; no machine paths"
    requirement: SCAFF-03
    verification:
      - kind: other
        ref: "grep -cF 'secret-scan' README.md (>=1, actual 2); -cF 'gsd/phase-' (>=1); 'build / vet / lint / test / tidy' present; machine-path grep (=0)"
      - kind: other
        ref: "bash experiments/check-docs.sh green after the rewrite (path-shape scan extension unaffected)"
    human_judgment: false
  - id: D6
    description: "SCAFF-02 OBSERVED on both trigger legs: push run on master head sha AND PR-triggered run on scratch-branch head sha, each with build / vet / lint / test / tidy + govulncheck + secret-scan all success; probe PR closed unmerged, branch deleted; release/e2e/security never triggered"
    requirement: SCAFF-02
    verification:
      - kind: integration
        ref: "gh api repos/Djarvur/zcode-acp/commits/master/check-runs — three names all success (push leg, run 44542652193)"
      - kind: integration
        ref: "gh api .../commits/e064969.../check-runs — three names all success (PR leg, run 44542927603); gh pr list --head ci/pr-event-probe state=CLOSED mergeCommit=none"
    human_judgment: false

# Metrics
duration: 13min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 4: CI Surface Wiring (pr-sanity + Inert Shapes + README) Summary

**Four goswitch-shaped workflows landed mise-driven; pr-sanity OBSERVED green with the three protection check names on BOTH trigger legs (push + real PR, scratch probe closed unmerged); README documents CI + branch protection before the gate lands**

## Performance

- **Duration:** 13 min
- **Started:** 2026-11-19T14:11:25Z
- **Completed:** 2026-11-19T14:24:38Z (production commits; observation legs included)
- **Tasks:** 3
- **Files modified:** 5 (4 created, 1 modified)

## Accomplishments

- `pr-sanity.yml` landed with the three check names plan 02-05 will require — `build / vet / lint / test / tidy`, `govulncheck`, `secret-scan` — sanity job running the six mise gates (build, vet, lint, test, test-experiments, tidy-diff), govulncheck via `mise exec`, and the docs conformance gate sharing the secret-scan job (no fourth required name)
- `release.yml` (v* tags only, sole `contents: write`, fetch-depth 0, serial release group, mise-installed goreleaser), `e2e-matrix.yml` (dispatch-only self-hosted GNOME skeleton, goswitch discipline without its desktop-fixture body), and `security-scheduled.yml` (weekly cron `0 6 * * 1`) landed inert by construction — none triggered during the plan
- README rewritten: honest English one-liner + CI table (four workflows), mise.toml single-version-source note, the three exact required-check names, PR-only branch flow with `gsd/phase-{phase}-{slug}`, and the fresh-protection honesty note with the `gh api` re-apply recipe
- **SCAFF-02 proven live on both trigger legs**: push run 44542652193 green on the master head sha AND a real PR-triggered run 44542927603 green on the scratch branch head sha — all three check names `success` on both shas; probe PR #1 closed unmerged, `ci/pr-event-probe` branch deleted locally and remotely
- All plan commits pushed to origin master (fast-forward after reconciliation); local gates (scan-secrets.sh, check-docs.sh) verified green before the CI-bearing push

## Task Commits

Each task was committed atomically:

1. **Task 1: pr-sanity.yml — sanity + govulncheck + secret-scan jobs** - `34b0196` (feat)
2. **Task 2: release.yml + e2e-matrix.yml + security-scheduled.yml — inert shapes** - `cd88698` (feat)
3. **Task 3: README CI/protection section + push + observe both legs** - `3ca6e96` (docs) + `57d48a2` (chore — remote-root reconciliation merge, see Deviations)

## Files Created/Modified

- `.github/workflows/pr-sanity.yml` - three jobs with exact check names; mise-only tool source; header comment names all four adaptations from the goswitch template (master branch, test-experiments, secret-scan job, docs-gate step) and carries the rejection rationale for any second tool pin
- `.github/workflows/release.yml` - English provenance header (translated from goswitch's Russian, decision ids remapped), registration note preserved, inert-by-construction note
- `.github/workflows/e2e-matrix.yml` - discipline-only skeleton: fork-event prohibition, serial e2e-gnome group, env:-indirection comment preserved for Phase 7, no AT-SPI plumbing
- `.github/workflows/security-scheduled.yml` - verbatim goswitch shape, English header
- `README.md` - CI + branch-protection documentation section per D-06 (docs precede the gate 02-05 applies)

## CI Observation Record

| Leg | Run | Trigger | Sha | Conclusion | Check names |
|-----|-----|---------|-----|------------|-------------|
| push | 44542652193 | push master | 57d48a2 | success | build / vet / lint / test / tidy = success; govulncheck = success; secret-scan = success (sanity 13/13, govulncheck 8/8, secret-scan 6/6 steps) |
| PR | 44542927603 | pull_request (ci/pr-event-probe) | e064969 | success | same three names, all success |

A fourth check, `update-go_modules-graph`, is GitHub's native dependency-graph update (event `dynamic`), not one of our workflows — harmless alongside protection's three required contexts. `release.yml`, `e2e-matrix.yml`, and `security-scheduled.yml` produced zero runs (inert by construction, as intended).

## Decisions Made

- Docs-gate runs as the second step of the `secret-scan` job (RESEARCH Open Question 1, resolution A4) — protected via the job's required context without expanding the required-check set
- e2e skeleton watchdog mirrors the template's order of magnitude (150 min) though the placeholder finishes instantly — the ceiling exists for Phase 7's live-desktop scenario
- Step names adjusted (`run govulncheck` instead of `govulncheck`; `scheduled govulncheck`) so the plan's job-name grep gates stay exact — the job `name:` fields (the protection contexts) are untouched
- Comment phrasing avoids the literal gate strings (`contents: read`/`contents: write` appear exactly once per file where the permission block requires them) — keeps the plan's cross-file grep gates meaningful

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Remote master had been re-initialized with an unrelated root commit; the plan's push was rejected (fetch-first)**
- **Found during:** Task 3, step 2 (push to origin master)
- **Issue:** `git push origin master` rejected — remote master pointed at `f444edf Initial commit`, a GitHub-generated scaffold commit (.gitignore + LICENSE + 2-line README stub, committer GitHub, 2026-11-19 03:59:56 +0300) unrelated to the local root `9f7cf5c`, whose tree is byte-identical (verified: `git diff 9f7cf5c f444edf` empty). Local master carried 81 commits (all Phase 1 + Phase 2 work, never pushed).
- **Fix:** One unrelated-histories merge (`git merge f444edf --allow-unrelated-histories`), resolving the two add/add conflicts (.gitignore, README.md) with the local evolved versions. Verified the merged tree equals the pre-merge local tree exactly (remote contributed nothing new). Rejected alternatives: rebase would rewrite 80 local commit hashes referenced by sibling SUMMARYs (audit-trail corruption); force-push would destroy the remote's commit and is prohibited on a branch I did not create. The merge keeps every local hash intact and makes the push a fast-forward (`f444edf..57d48a2`).
- **Files modified:** none content-wise (merge commit only; `.gitignore`/`README.md` conflict resolutions keep local versions)
- **Verification:** `git diff 3ca6e96 57d48a2` empty; push fast-forwarded; CI green on the merged head
- **Committed in:** 57d48a2 (chore(02-04): merge re-initialized GitHub master root into project history)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The merge is the minimal non-destructive reconciliation that lets the plan's mandated push land; no content changed and no history was rewritten.

## Issues Encountered

- The push run completed suspiciously fast (~34 s); verified not a false positive — all steps of all three jobs report success (13/13, 8/8, 6/6) and the check-runs API shows the three names success on the exact pushed sha.
- The first `gh run list --event pull_request` poll found the run already in progress/complete within ~90 s of opening the PR — both legs are recorded above with run ids and shas.

## User Setup Required

None - no external service configuration required (gh admin session already authenticated; mise installs toolchain runner-side).

## Known Stubs

- `.github/workflows/e2e-matrix.yml` placeholder step (`echo "e2e scenario lands in Phase 7 — skeleton only"`) — **intentional per plan** (D-04: skeleton now, real scenario Phase 7; RESEARCH Pattern 4). No runner is registered with the `self-hosted`/`gnome` labels, so dispatches queue forever; Phase 7 authors the scenario and registers the runner.

## Next Phase Readiness

- Plan 02-05 can lock protection to the three OBSERVED check names (all three seen green on both legs); `protection-body.json` contexts must copy the pr-sanity job `name:` fields character-for-character
- README already documents the protection setup 02-05 applies (docs precede the gate, per RESEARCH ordering)
- Pre-protection direct-push window closes with 02-05: `branching_strategy` flips to `phase` there; Phases 3+ work via `gsd/phase-*` PRs under these gates

## Self-Check: PASSED

- SUMMARY.md exists on disk at `.planning/phases/02-scaffold-ci-bootstrap/02-04-SUMMARY.md`
- Task commits verified as ancestors of HEAD: `34b0196` (Task 1), `cd88698` (Task 2), `3ca6e96` + `57d48a2` (Task 3)
- Plan ledger MEASURED: `git rev-list --count a266eef..57d48a2` = 5 (4 authored + 1 adopted remote scaffold root)
- All six Task-3 verifies green: master check-runs (3/3 names success), push run completed/success, pull_request run completed/success, PR head-sha check-runs (3/3 names success), probe PR CLOSED with mergeCommit=none, README greps hit
- Workflow files on disk verified; release/e2e/security workflows never triggered; probe branch absent locally and remotely

---
*Phase: 02-scaffold-ci-bootstrap*
*Completed: 2026-11-19*
