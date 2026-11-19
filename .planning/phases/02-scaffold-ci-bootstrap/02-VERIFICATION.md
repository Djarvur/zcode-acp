---
phase: 02-scaffold-ci-bootstrap
verified: 2026-11-19T15:15:48Z
status: passed
score: 30/30 must-haves verified
covered_files: [".github/workflows/e2e-matrix.yml", ".github/workflows/pr-sanity.yml", ".github/workflows/release.yml", ".github/workflows/security-scheduled.yml", ".gitignore", ".golangci.yml", ".planning/config.json", ".planning/phases/02-scaffold-ci-bootstrap/02-01-PLAN.md", ".planning/phases/02-scaffold-ci-bootstrap/02-01-SUMMARY.md", ".planning/phases/02-scaffold-ci-bootstrap/02-02-PLAN.md", ".planning/phases/02-scaffold-ci-bootstrap/02-02-SUMMARY.md", ".planning/phases/02-scaffold-ci-bootstrap/02-03-PLAN.md", ".planning/phases/02-scaffold-ci-bootstrap/02-03-SUMMARY.md", ".planning/phases/02-scaffold-ci-bootstrap/02-04-PLAN.md", ".planning/phases/02-scaffold-ci-bootstrap/02-04-SUMMARY.md", ".planning/phases/02-scaffold-ci-bootstrap/02-05-PLAN.md", ".planning/phases/02-scaffold-ci-bootstrap/02-05-SUMMARY.md", ".planning/phases/02-scaffold-ci-bootstrap/protection-body.json", "README.md", "cmd/zcode-acp/main.go", "docs/protocol/app-server-dialect.md", "experiments/app-server-probe.mjs", "experiments/check-docs.sh", "experiments/scan-secrets.sh", "experiments/scrub-capture.mjs", "experiments/scrub-capture.test.mjs", "go.mod", "mise.toml"]
covered_digest: "v4:sha256:b3f25f922f31b8d1ac39db1a74b6e87e2f38861ab6c4e6f3de5ebc9d9e2c8d01"
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "Post-protection local-only commits (branching_strategy flip e777ba6, CR-01 fix 3969eae/4ed831a, WR-01 timeouts 3c13b6b, planning docs) reach origin master"
    addressed_in: "Phase 3 (first gsd/phase-* PR)"
    evidence: "Recorded landing path: 02-05-SUMMARY hand-off ('the flip commit e777ba6 rides the first Phase-3 gsd/phase-* PR; Phase 3 forks from updated local master') and STATE.md:120. Direct push is rejected by the protection just proven, so PR merge is the only landing path; Phase 3 forks from the updated LOCAL master, which carries all of these commits."
prohibition_verdicts: # judgment-tier (ADR-550 D4) — autonomous LLM-judge verdicts, NON-AUTHORITATIVE; recorded with deterministic enforcement evidence so nothing is silent
  - statement: "CI workflows never call api.z.ai, mint API keys or PATs, or spawn the real ZCode desktop bundle — CI exercises only the three offline test tiers"
    verdict: pass
    evidence: "Repo-wide grep over .github/workflows/: zero references to api.z.ai, key/PAT minting, or zcode.cjs execution; the only api.z.ai string is pr-sanity.yml:27's own prohibition comment; e2e-matrix.yml 'desktop' hits are discipline comments + an echo placeholder step; pr-sanity/release/security run only mise gates, govulncheck, and the two bash gates."
    human_review_recommended: true
  - statement: "Protection apply never reports success it did not verify — any API-refused rule is recorded; protection never applied before the phase's work was on master"
    verdict: pass
    evidence: "02-05-SUMMARY 'Applied-vs-Refused Record': NOTHING refused, full body applied; independently re-verified — live GET matches protection-body.json 9/9 comparison booleans (checks_match, strict, admins, pr_reviews, restrictions null, force_push, deletions, linear, conv_res), and pr-sanity run 44544600222 was completed/success on ed9a0c1 before the PUT."
    human_review_recommended: true
---

# Phase 2: Scaffold & CI Bootstrap — Verification Report

**Phase Goal:** As a later-phase implementer on this repo, I want to have strict quality gates — strict pinned lint, CI, and a protected `master` modeled on Djarvur/goswitch — enforced from the very first code commit onward, so that all feature code lands under the same gates and later phases merge only through green PRs.
**Verified:** 2026-11-19T15:15:48Z
**Status:** passed
**Re-verification:** No — initial verification
**Mode:** mvp (User Story goal validated: `user-story.validate` → true)

## User Flow Coverage

User story: «As a later-phase implementer on this repo, I want to have strict quality gates — strict pinned lint, CI, and a protected `master` modeled on Djarvur/goswitch — enforced from the very first code commit onward, so that all feature code lands under the same gates and later phases merge only through green PRs.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Check out fresh; run the lint gate | Strict goswitch config (`default: all`) passes on golangci-lint exactly 2.15.0 | `mise run lint` re-run this session: exit 0, `0 issues.`; `mise exec -- golangci-lint --version` → 2.15.0; `mise.toml:13` exact pin; `.golangci.yml:13` `default: all` | ✓ |
| Plant a violation | The lint target fails with named issues | Re-fired red-proof: 130+ char line in `red-probe/` → exit 1, `unused: 1` named; second variant (long string literal) → exit 1 as well; clean tree restored to green, no residue | ✓ |
| Open a PR | CI runs lint, test, build, govulncheck — tool versions from mise only | pr-sanity.yml `on:` push(master)/pull_request/workflow_dispatch; live check-runs on PR-leg sha `e064969`: all three names success (run 44542927603); no `setup-go`/`go-version:`/second pin anywhere in `.github/workflows/` | ✓ |
| Try to push directly to `master` | Rejected | GH006 recorded at plan execution (02-05-SUMMARY verbatim: `push_rc=1`, "Changes must be made through a pull request", "3 of 3 required status checks are expected"); live GET this session confirms the exact enforcement settings that produce it (PR required + enforce_admins + 3 required checks) | ✓ |
| Merge only when configured checks pass | The three check names are required contexts, admins bound | Live GET matches `protection-body.json` 9/9 fields; contexts character-for-character the pr-sanity job `name:` fields (verified on both the workflow file and the live check-run names) | ✓ |
| Look up the setup in the repo | README documents CI + protection; the exact PUT body is committed | README.md CI/branch-protection section (four workflows, three check names, PR-only flow, honesty note, re-apply recipe); `protection-body.json` present on origin master (`git ls-tree origin/master` → blob at `ed9a0c1`) | ✓ |
| Outcome: all feature code lands under the same gates; later phases merge only through green PRs | Gates live before any feature code; PR-only flow enforced | Protection live (GET-verified), probe PR #1 observed running all three checks then closed unmerged; `branching_strategy: "phase"` live locally (`.planning/config.json:15`); Phase 3+ forks `gsd/phase-*` branches that merge only through green PRs | ✓ |

## Goal Achievement

### Observable Truths

Merged from ROADMAP success criteria (contract) + all five PLAN must_haves frontmatters. Score **30/30**; 0 present-but-behavior-unverified.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Skeleton compiles; `--version` prints exactly `zcode-acp 0.0.0-dev` | ✓ VERIFIED | Re-run this session: stdout exact, rc=0 |
| 2 | `mise run lint` exits 0 on the skeleton under the goswitch strict config | ✓ VERIFIED | Re-run: rc=0, `0 issues.` |
| 3 | A planted violation fails the lint target | ✓ VERIFIED | Re-fired red-proof (2 shapes): rc=1 with named issues; tree restored, `git status` clean |
| 4 | mise.toml is the single source of tool versions (go 1.27 line, golangci-lint EXACTLY 2.15.0, goreleaser 2.20.2, node 26) | ✓ VERIFIED | mise.toml:9/13/17/19; `golangci-lint --version` → 2.15.0; `go version` → go1.27.3 |
| 5 | go.mod adjacency: module github.com/Djarvur/zcode-acp + `go 1.27` agreeing with the mise pin | ✓ VERIFIED | go.mod read; key-link verify: pattern found |
| 6 | mise tasks cover build/vet/lint/test/test-experiments/tidy-diff + `ci` aggregate in gate order | ✓ VERIFIED | mise.toml `[tasks.*]` inventory; `[tasks.ci]` depends = build→vet→lint→test→test-experiments (tidy-diff deliberately in CI's sanity job per the plan's ordering probe truth); full `mise run ci` rc=0 |
| 7 | Scrubber: sibling-prefix paths pass through verbatim (WR-03) | ✓ VERIFIED | node:test `WR-03: sibling path sharing only the home prefix survives verbatim` pass in live suite run (20/20) |
| 8 | Scrubber: home-anchored + bare-home tilde rewrites preserved (WR-03 truth-preserving) | ✓ VERIFIED | `absolute paths under the user home directory become tilde-relative` pass; boundary regex at scrub-capture.mjs with `(?!/?[A-Za-z0-9._-])` |
| 9 | Scrubbing is idempotent | ✓ VERIFIED | `scrubbing is idempotent` + `fixed point` tests pass live |
| 10 | Capture-header guard: malformed line 1 fails loudly; `--no-header` scrubs line 1 (IN-04) | ✓ VERIFIED | Three IN-04 tests pass live (non-JSON fails, no-capture-key fails, --no-header exits 0 + scrubs) |
| 11 | Empty input → empty output, exit 0 | ✓ VERIFIED | Test passes live |
| 12 | `node --test experiments/` fully green and `scan-secrets.sh` green | ✓ VERIFIED | Re-run: 20/20 pass, 0 fail/skipped; scan rc=0 |
| 13 | Secret-gate credential class byte-identical to scrubber CRED_RX (`[A-Za-z0-9+/=_:.-]{20,}`) | ✓ VERIFIED | scan-secrets.sh:28 inside the `enc:v[12]:(os:)?…` frame; key-link verify: pattern found in both files |
| 14 | Keyed-token pattern requires no closing quote | ✓ VERIFIED | scan-secrets.sh:29 — class `{20,}` with no trailing quote |
| 15 | Both gate scripts toplevel-pinned; green from any CWD (IN-03) | ✓ VERIFIED | `git rev-parse --show-toplevel` pins; re-run green from repo root AND from `docs/` |
| 16 | Dialect doc error-code recipe leading-dash-safe (WR-02) | ✓ VERIFIED | dialect doc:146 `grep -o -e '-32603'` + GNU-grep caveat |
| 17 | Method-extraction digit-inclusive + `singletons (9)` label (IN-01/IN-02) | ✓ VERIFIED | dialect doc:41 class `'"[a-zA-Z0-9/]*"$'`; line 65 label lists 9 methods |
| 18 | Probe records desktopVersion from `--desktop-version`, default `unknown` (WR-04) | ✓ VERIFIED | 2 occurrences (parse + use); hardcoded `'3.18.2'` literal absent; `node --check` green (per 02-03 records), suite green |
| 19 | Push AND PR both trigger pr-sanity with the three exact check names; both legs green (SC-2) | ✓ VERIFIED | Live API: push runs 44542652193/44543958415/44544600222 success; pull_request run 44542927603 success; check-runs on BOTH shas (ed9a0c1 push, e064969 PR head) list all three names = success |
| 20 | CI reads tool versions only from mise — no second pin anywhere | ✓ VERIFIED | Workflow inspection: mise install + `mise run` steps only (8 `mise run` in pr-sanity); no setup-go/golangci-lint-action/version inputs (hits are the rejection-rationale comments) |
| 21 | secret-scan job runs the fixed scan-secrets.sh + check-docs.sh | ✓ VERIFIED | pr-sanity.yml:123-140 — both bash gates as job steps; docs-gate shares the job (no fourth required name) |
| 22 | release.yml: `v*` tags only, sole `contents: write`, fetch-depth 0, inert | ✓ VERIFIED | release.yml:27/31/49; `gh run list --workflow=release.yml` → 0 runs (no tag exists) |
| 23 | e2e-matrix.yml: dispatch-only skeleton, contents:read, serial e2e-gnome, self-hosted+gnome, placeholder step — inert (D-04) | ✓ VERIFIED | All greps hit; 0 runs (no registered runner); placeholder is the plan's own prescribed shape (02-04 Known Stubs) |
| 24 | security-scheduled.yml: weekly cron `0 6 * * 1` + dispatch, contents:read | ✓ VERIFIED | cron at line 13; 0 runs in-window (inert by schedule) |
| 25 | README documents CI + branch protection (SCAFF-03 documentation half) | ✓ VERIFIED | CI table, mise single-source note, three check names, `gsd/phase-{phase}-{slug}` flow, fresh-protection honesty note + re-apply recipe |
| 26 | Live protection GET returns the applied state, matching the body field-by-field (SC-3) | ✓ VERIFIED | Live `gh api …/branches/master/protection`: jq comparison 9/9 true (checks, strict, admins, PR reviews 0-approval, restrictions null, no force-push/deletions, linear, conv-res) |
| 27 | Direct push to master rejected with GH006 (SC-3) | ✓ VERIFIED | Behavioral run recorded at plan execution (02-05-SUMMARY verbatim: GH006, push_rc=1, probe reset cleaned); standing evidence re-confirmed live this session: the exact enforcement configuration (PR required, enforce_admins, 3 required checks) is applied. Re-fire deliberately not repeated — if protection were silently off, the probe push would land on origin master (pollution risk); the live GET of the enforced state is the standing evidence per the honest-evidence contract |
| 28 | Config flip to `phase` happened ONLY after GET verification (D-01 sequencing) | ✓ VERIFIED | Recorded order apply→GET→GH006→flip with plan verify-gates enforcing it; flip live in `.planning/config.json:15`; flip commit e777ba6 exists locally, on no remote branch |
| 29 | protection-body.json equals the GET AND is on origin master before apply | ✓ VERIFIED | 9/9 field match; `git ls-tree origin/master` shows the blob at `ed9a0c1`; pr-sanity 44544600222 green on that sha before the PUT |
| 30 | If the API refused any rule it is recorded honestly (D-01 fallback) | ✓ VERIFIED | "Applied-vs-Refused: NOTHING was refused" record; independently consistent — my GET matches the full intended body, so nothing can have been silently degraded |

**Score:** 30/30 truths verified (0 present, behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Post-protection local-only commits reach origin master: flip commit `e777ba6`, CR-01 fix (`3969eae`/`4ed831a`), WR-01 job timeouts (`3c13b6b`), planning docs | Phase 3 (first `gsd/phase-*` PR) | Recorded landing path in 02-05-SUMMARY hand-off + STATE.md:120; direct push is rejected by the protection just proven, and Phase 3 forks from the updated LOCAL master carrying all of them. Documented limitation, not a gap — origin/master (`ed9a0c1`) itself is fully green under all three SCs (its CI runs observed success) |

### Advisory (New Scope, Unevidenced)

Not applicable — initial verification (no prior contract; the re-verification-only advisory gate did not run). For the record: the probe red-proof re-fired this session caught `unused: 1` where 02-RESEARCH/02-01 recorded 3 issues (lll/nlreturn/unused) for their own probe shape — a shape difference in the negative-test recipe, not a gate weakness: both shapes exit non-zero with named issues, which is the load-bearing claim.

### Required Artifacts

17/17 artifacts across the five plans: exists + substantive (gsd-tools `verify.artifacts` all_passed on all five plans, exit 0 each) + wired (key links below).

| Artifact | Expected | Status |
| -------- | -------- | ------ |
| `mise.toml` / `.golangci.yml` / `go.mod` / `cmd/zcode-acp/main.go` | Toolchain pins + strict config + module + skeleton | ✓ VERIFIED |
| `experiments/scrub-capture.mjs` / `scrub-capture.test.mjs` | WR-03/IN-04 hardened scrubber + tests | ✓ VERIFIED |
| `experiments/scan-secrets.sh` / `check-docs.sh` / `app-server-probe.mjs` / `docs/protocol/app-server-dialect.md` | Review-fix hardening | ✓ VERIFIED |
| `.github/workflows/pr-sanity.yml` / `release.yml` / `e2e-matrix.yml` / `security-scheduled.yml` | Four goswitch-shaped workflows | ✓ VERIFIED |
| `README.md` | CI + protection documentation | ✓ VERIFIED |
| `.planning/phases/02-scaffold-ci-bootstrap/protection-body.json` / `.planning/config.json` | PUT payload + strategy flip | ✓ VERIFIED |

### Key Link Verification

12/12 links WIRED (`verify.key-links` all_verified=true on all five plans, exit 0 each): mise↔go.mod adjacency, mise lint→.golangci.yml, main.go→go.mod; test→scrubber harness + string-concat fixtures; scan-secrets→scrubber CRED_RX; probe→dialect provenance; pr-sanity→mise tasks; pr-sanity→scan-secrets; README→workflow check names; protection-body→pr-sanity contexts; config-flip→README flow.

### Data-Flow Trace (Level 4)

No rendered dynamic UI in this phase (infrastructure). The dynamic values that matter are config-driven: CI job versions flow mise.toml → `mise install` → runners (verified: no second pin); required contexts flow pr-sanity job `name:` fields → protection-body.json → live GET → check-run names (all four points verified character-for-character). Status: FLOWING.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Clean lint gate | `mise run lint` | rc=0, `0 issues.` | ✓ PASS |
| Planted violation fails gate | `golangci-lint run red-probe/...` (planted 130+ char line) | rc=1, `unused: 1` named; variant 2 (long literal) rc=1 | ✓ PASS |
| Full local gate | `mise run ci` | rc=0; experiments suite 20 pass / 0 fail / 0 skipped | ✓ PASS |
| Version string | `go run ./cmd/zcode-acp --version` | `zcode-acp 0.0.0-dev`, rc=0 | ✓ PASS |
| Gates CWD-independent | `cd docs && bash ../experiments/{scan-secrets,check-docs}.sh` | rc=0 | ✓ PASS |
| Phase docs gate | `bash experiments/check-docs.sh --phase-gate` | rc=0 | ✓ PASS |
| CI push leg | `gh api .../commits/ed9a0c1/check-runs` | 3 names all success | ✓ PASS |
| CI PR leg | `gh api .../commits/e064969/check-runs` | 3 names all success | ✓ PASS |
| Inert workflows never ran | `gh run list` release/e2e-matrix/security-scheduled | 0 runs each | ✓ PASS |
| Probe PR hygiene | `gh pr list --head ci/pr-event-probe` | CLOSED, mergedAt=null; branch absent | ✓ PASS |
| GH006 push rejection | (recorded at 02-05 execution; re-fire intentionally not repeated — pollution risk if enforcement were silently off) | 02-05-SUMMARY verbatim: push_rc=1, GH006, reset=cleaned; enforcement settings live-verified by GET | ✓ PASS (recorded behavioral + live state) |

### Probe Execution

No conventional `scripts/**/tests/probe-*.sh` probes exist and none were declared by the plans (discovery returned empty). The phase's declared red-proof recipe is covered by the Behavioral Spot-Checks above — re-fired first-hand.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| SCAFF-01 | 02-01 | Strict pinned golangci-lint scaffold; violation fails | ✓ SATISFIED | Truths 1-6, spot-checks 1-4 |
| SCAFF-02 | 02-02, 02-03, 02-04 | CI running lint+test+build+govulncheck, mise-driven; review-fix hardening feeding CI | ✓ SATISFIED | Truths 7-21, spot-checks 5-10 |
| SCAFF-03 | 02-04 (docs half), 02-05 | master protection configured and documented | ✓ SATISFIED | Truths 22-30 (25-30), spot-checks 7-11 |

Orphaned requirements: none — REQUIREMENTS.md maps exactly SCAFF-01/02/03 to Phase 2, all three claimed across plans, all marked Complete in the traceability table.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts: 6 honored / 6 total, 0 not honored (`check.decision-coverage-verify`, exit 0).

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `experiments/scrub-capture.test.mjs` | SCAFF-02 contract (WR-03/IN-04/CR-01) | 20 | 0 | No (line 38 writes the INPUT fixture only; expected values are inline literals) | Behavioral/value (verbatim survival, fixed points, exit codes + stderr) | STRONG |

Disabled-test scan: no `.skip`/`.todo`/`.only` (the `xit(` regex hits were `process.exit(` false positives — fail-loudly exits, not disabled tests). RED evidence records persisted: 7 files in the phase `evidence/` dir (5× 02-02 RED, 2× CR-01 review RED).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | No TBD/FIXME/XXX debt markers in any phase-modified file | — | — |
| `.github/workflows/e2e-matrix.yml` | 40-51 | `placeholder` step | ℹ️ Info | Intentional per plan D-04 — the skeleton IS the deliverable; inert without a registered runner; documented in 02-04 Known Stubs and README |
| `cmd/zcode-acp/main.go` | 15 | `var version = "0.0.0-dev"` dev fallback | ℹ️ Info | Intentional stamping contract (goreleaser stamps real value at Phase 7 release); documented in 02-01 Known Stubs |

### Human Verification Required

N/A — Infrastructure/foundation phase with no user-facing elements. All acceptance criteria were verifiable programmatically, and every behavior-dependent claim had first-hand or live behavioral evidence (lint red/green re-fired, CI runs fetched live from the GitHub API, experiments suite re-run, protection GET re-run). No ⚠️ PRESENT_BEHAVIOR_UNVERIFIED truths remain (`behavior_unverified: 0`).

### Prohibition Verification (judgment-tier, recorded per ADR-550 D4)

Both plan-authored prohibitions were checked with deterministic evidence and PASS (verdicts in frontmatter `prohibition_verdicts`, NON-AUTHORITATIVE — human review recommended as a formality): (1) CI stays offline — no api.z.ai calls, no key/PAT minting, no desktop-bundle spawn in any workflow; (2) protection apply reported no unverified success — the "nothing refused" record is independently consistent with the live GET matching the full intended body 9/9.

### Gaps Summary

None. The phase goal is achieved and observable in the codebase and on the live repository: the strict pinned lint gate passes green and demonstrably bites red; a push and a real PR each trigger mise-driven CI with the three check names green (live API evidence on both legs); master protection is applied, matches its documented payload field-by-field, rejects direct pushes (recorded GH006 + live enforcement state), and is documented in-repo (README + protection-body.json on origin master). The review round's critical finding (CR-01) is fixed under TDD with both regression tests passing in the live suite; its remaining info/warning items are explicitly routed to later-phase backlogs in the disposition ledger and block nothing here. The only caveat is the documented post-protection local-only commit set (flip, CR-01 fix, WR-01 timeouts, planning docs) whose recorded landing path is the first Phase-3 PR — a documented limitation of the protection sequencing, not a gap.

---

_Verified: 2026-11-19T15:15:48Z_
_Verifier: Claude (gsd-verifier)_
