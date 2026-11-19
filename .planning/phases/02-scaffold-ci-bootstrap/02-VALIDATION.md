---
phase: "2"
slug: "scaffold-ci-bootstrap"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-19"
validated: "2026-11-19"
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | node:test (built-in runner) for the experiments suite; golangci-lint 2.15.0 gate; bash gates (scan-secrets.sh, check-docs.sh); go build/vet/test via mise tasks |
| **Config file** | `.golangci.yml` + `mise.toml` (created this phase) |
| **Quick run command** | `bash experiments/scan-secrets.sh && bash experiments/check-docs.sh && node --test experiments/` |
| **Full suite command** | `mise run ci` (build + vet + lint + test + test-experiments) |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Quick gates; `mise run lint` once the config lands
- **After every plan wave:** `mise run ci`
- **Before `/gsd:verify-work`:** Full suite green + workflows observed running + protection GET matches the documented body
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

All 11 tasks from the 5 plans, statuses from audit re-execution 2026-11-19 (every command below re-run by the auditor, not trusted from summaries).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01 T1 (tracer) | 02-01 | 1 | SCAFF-01 | T-2-SC | strict gate enforced on skeleton | gate (machine check) | `mise run build && mise run vet && mise run lint`; `go run ./cmd/zcode-acp --version`; `mise run test && mise run test-experiments && mise run tidy-diff` | ✅ | ✅ green (lint 0 issues; version string exactly `zcode-acp 0.0.0-dev`; vet/tidy-diff/test/test-experiments all rc=0) |
| 02-01 T2 (red-proof) | 02-01 | 1 | SCAFF-01 | — | exact pin reproduces; violations fail | version check + negative gate | plant 130-char line in `red-probe/` → `mise exec -- golangci-lint run red-probe/...` must exit ≠0, then remove, `mise run lint`; `mise exec -- golangci-lint --version` | ✅ | ✅ green (re-executed: planted violation exit 1 with named issue `unused`; probe dir removed; clean tree 0 issues; pin exactly 2.15.0) |
| 02-02 T1 (RED tests) | 02-02 | 1 | D-05 (CONTEXT) | T-2-01 | failing WR-03/IN-04 tests encode the contract | unit (node:test RED) | `node --test experiments/` (target tests fail on assertions; pre-existing stay green) | ✅ | ✅ green (5 RED evidence records in phase `evidence/` — targetTest/exitCode/expected/actual verified, e.g. `~x/otherproj/file.txt` falsification; RED state machine-checked at execution) |
| 02-02 T2 (GREEN fix) | 02-02 | 1 | D-05 (CONTEXT) | T-2-01 | scrubber boundary fix + header guard behave | unit (node:test) | `node --test experiments/ && bash experiments/scan-secrets.sh && bash experiments/check-docs.sh` | ✅ | ✅ green (18/18 pass / 0 fail re-executed; both gates rc=0) |
| 02-03 T1 (gate scripts) | 02-03 | 1 | D-05 (CONTEXT) | T-2-01 | widened secret patterns stay green; toplevel pin | gate | `bash experiments/scan-secrets.sh && bash experiments/check-docs.sh` from repo root AND from a subdirectory (`cd docs && bash ../experiments/...`) | ✅ | ✅ green (rc=0 from root and from `docs/`; CRED_RX class `[A-Za-z0-9+/=_:.-]{20,}` present ×1; toplevel pin ×1 per script) |
| 02-03 T2 (doc recipes + probe) | 02-03 | 1 | D-05 (CONTEXT) | — | leading-dash-safe recipe, digit-inclusive class, singletons (9), probe --desktop-version | grep gates + `node --check` | `node --check experiments/app-server-probe.mjs`; the four doc-recipe greps per 02-03 T2 verify; `bash experiments/check-docs.sh --phase-gate` | ✅ | ✅ green (node --check rc=0; `--desktop-version` ×2, hardcoded `'3.18.2'` absent; all 4 recipe greps =1 incl. `singletons (9)`; phase-gate rc=0) |
| 02-04 T1 (pr-sanity.yml) | 02-04 | 2 | SCAFF-02 | T-2-04, T-2-05 | three named jobs; mise-only tool source | grep gates + CI observation | job-name greps per 02-04 T1 verify; `gh run list --workflow=pr-sanity.yml` | ✅ | ✅ green (3 exact job names ×1 each; 8 `mise run` steps; `actions/checkout@v4` ×3; runs observed live) |
| 02-04 T2 (inert workflow shapes) | 02-04 | 2 | SCAFF-02 | T-2-05, T-2-SC | release v*-tags + sole contents:write; e2e dispatch-only self-hosted; weekly cron | grep gates + inert observation | greps per 02-04 T2 verify (`tags: ['v*']`, `contents: write` ×1, `contents: read` ×3, `workflow_dispatch`, `runs-on: [self-hosted, gnome]`, `0 6 * * 1`); `gh run list` per workflow | ✅ | ✅ green (all greps hit; release/e2e-matrix/security-scheduled have ZERO runs — inert by construction) |
| 02-04 T3 (README + observe CI both legs) | 02-04 | 2 | SCAFF-02 | T-2-04, T-2-SC | workflows trigger on push AND PR; three check names green on both shas | CI observation | `gh run list --workflow=pr-sanity.yml --limit 3` (push + pull_request legs); `gh api repos/Djarvur/zcode-acp/commits/master/check-runs`; `gh pr list --head ci/pr-event-probe`; README greps | ✅ | ✅ green (3 runs completed success: 2 push on master + 1 pull_request on ci/pr-event-probe; master head check-runs = exactly `build / vet / lint / test / tidy`, `govulncheck`, `secret-scan` all success; PR #1 CLOSED with mergeCommit null; README greps hit) |
| 02-05 T1 (protection body) | 02-05 | 3 | SCAFF-03 | T-2-SC | body contexts = pr-sanity job names; valid JSON; pushed pre-apply | jq + cross-grep + push gate | `jq -e '.required_status_checks.checks \|.map(.context)' .planning/phases/02-scaffold-ci-bootstrap/protection-body.json`; name/context loop per 02-05 T1 verify; `git fetch origin master && git rev-list --count origin/master..HEAD` | ✅ | ✅ green (jq prints exactly the 3 contexts in workflow job order; cross-check loop rc=0; body commit `ed9a0c1` verified ancestor of origin/master — on origin before apply) |
| 02-05 T2 (protection apply) | 02-05 | 3 | SCAFF-03 | T-2-02, T-2-03 | protection configured; direct pushes rejected; flip after verify | API check + negative test | `gh api repos/Djarvur/zcode-acp/branches/master/protection` (GET) vs body field-by-field; GH006 push compound; `grep -c '"branching_strategy": "phase"' .planning/config.json` | ✅ | ✅ green (GET re-executed: 9/9 field match — 3 exact contexts, strict false, enforce_admins true, PR required 0 approvals, restrictions null, force-push/deletions false, linear true, conv-res false; GH006 behavioral probe passed at execution with push_rc=1 + probe cleaned (plan record, verbatim remote output); flip grep=1, commit `e777ba6` local-only by design, rides first Phase-3 `gsd/phase-*` PR) |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Audit note (2026-11-19):** the GH006 negative-push probe was fired and observed at execution time (plan 02-05 T2 record: `GH006: Protected branch update failed`, "Changes must be made through a pull request", push_rc=1, reset=cleaned). The audit re-verified the *state* it produced (live GET 9/9 match) rather than re-firing a live push against the now-protected branch — re-firing would pollute master history if enforcement were ever broken, and the recorded behavioral evidence plus the current enforced state together cover the requirement.

---

## Wave 0 Requirements

- [x] WR-03 + IN-04 unit tests in `experiments/scrub-capture.test.mjs` (RED→GREEN, per TDD mode) — 18 tests, all green; 5 RED evidence records in phase `evidence/`
- [x] `mise.toml`, `.golangci.yml`, `go.mod`, `cmd/zcode-acp/main.go`, four workflow files — all on disk and verified (probe contents were the starting shapes)
- [x] No framework installs: node:test built-in; linter via mise — confirmed (`mise install` only, zero registry packages)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| (none — all SCAFF checks are API/gate-observable) | — | — | — |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — all 11 tasks (02-01 ×2, 02-02 ×2, 02-03 ×2, 02-04 ×3, 02-05 ×2) carry `<verify><automated>` blocks
- [x] Sampling continuity: no 3 consecutive tasks without automated verify — every task has automated verify; zero consecutive gaps
- [x] Wave 0 covers all MISSING references — all Wave-0 artifacts exist and are verified above
- [x] No watch-mode flags
- [x] Feedback latency < 60s (quick gates ~3s lint / ~3s node:test / sub-second bash gates; full `mise run ci` re-executed rc=0)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-11-19 (audit re-execution: quick gates, red-proof, mise run ci, CI run list both legs, check-runs, protection GET 9/9 field match)
