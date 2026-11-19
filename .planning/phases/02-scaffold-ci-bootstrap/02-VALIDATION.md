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
| 02-01 T1 (tracer) | 02-01 | 1 | SCAFF-01 | T-2-SC | strict gate enforced on skeleton | gate (machine check) | `mise run lint` (exit 0 on skeleton; planted violation → ≠0, 02-01 T2) | ❌ W0 | ⬜ pending |
| 02-01 T2 (red-proof) | 02-01 | 1 | SCAFF-01 | — | exact pin reproduces; violations fail | version check + negative gate | `mise exec -- golangci-lint --version` → `2.15.0`; planted 130-char line → lint ≠0 | ❌ W0 | ⬜ pending |
| 02-02 T1+T2 (RED→GREEN) | 02-02 | 1 | D-05 (CONTEXT) | T-2-01 | scrubber boundary fix + header guard behave | unit (node:test RED→GREEN) | `node --test experiments/` | ❌ W0 | ⬜ pending |
| 02-03 T1 (gate scripts) | 02-03 | 1 | D-05 (CONTEXT) | T-2-01 | widened secret patterns stay green; toplevel pin | gate | `bash experiments/scan-secrets.sh` (also from a subdirectory) | ✅ | ⬜ pending |
| 02-03 T2 (doc recipes + probe) | 02-03 | 1 | D-05 (CONTEXT) | — | leading-dash-safe recipe, digit-inclusive class, singletons (9), probe --desktop-version | grep gates + `node --check` | greps per 02-03 T2 verify; `node --check experiments/app-server-probe.mjs` | ✅ | ⬜ pending |
| 02-04 T1-T3 (workflows) | 02-04 | 2 | SCAFF-02 | T-2-04, T-2-05, T-2-SC | workflows trigger + three named jobs green | CI observation | `gh run list --workflow=pr-sanity.yml`; `gh api repos/Djarvur/zcode-acp/commits/master/check-runs` | ❌ W0 | ⬜ pending |
| 02-05 T2 (protection) | 02-05 | 3 | SCAFF-03 | T-2-02, T-2-03 | protection configured; direct pushes rejected | API check + negative test | `gh api .../branches/master/protection`; push rejection (GH006) | ❌ W0 | ⬜ pending |

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
