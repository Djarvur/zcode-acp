---
phase: "2"
slug: "scaffold-ci-bootstrap"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-19"
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

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (seeded by planner) | 02-01 | 1 | SCAFF-01 | — | strict gate enforced | gate (machine check) | `mise run lint` (exit 0 on skeleton; planted violation → ≠0) | ❌ W0 | ⬜ pending |
| (seeded by planner) | 02-01 | 1 | SCAFF-01 | — | exact pin reproduces | version check | `golangci-lint version` → `2.15.0` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 02-02 | 1 | D-05 (CONTEXT) | — | scrubber boundary fix behaves | unit (node:test RED→GREEN) | `node --test experiments/` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 02-02 | 1 | D-05 (CONTEXT) | — | widened secret patterns stay green | gate | `bash experiments/scan-secrets.sh` | ✅ | ⬜ pending |
| (seeded by planner) | 02-03 | 2 | SCAFF-02 | — | workflows trigger + govulncheck job | CI observation | `gh run list --workflow=pr-sanity.yml`; check-runs on the PR sha | ❌ W0 | ⬜ pending |
| (seeded by planner) | 02-04 | 2 | SCAFF-03 | — | protection configured; direct pushes rejected | API check + negative test | `gh api .../branches/master/protection`; push rejection (GH006) | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] WR-03 + IN-04 unit tests in `experiments/scrub-capture.test.mjs` (RED→GREEN, per TDD mode)
- [ ] `mise.toml`, `.golangci.yml`, `go.mod`, `cmd/zcode-acp/main.go`, four workflow files (probe contents in 02-RESEARCH.md are the verified starting shapes)
- [ ] No framework installs: node:test built-in; linter via mise

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| (none — all SCAFF checks are API/gate-observable) | — | — | — |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
