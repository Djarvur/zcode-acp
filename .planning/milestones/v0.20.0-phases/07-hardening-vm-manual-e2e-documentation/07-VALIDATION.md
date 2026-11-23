---
phase: "7"
slug: "hardening-vm-manual-e2e-documentation"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-22"
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib, -race), golangci-lint strict (goswitch-style) |
| **Config file** | `.golangci.yml` (pinned), `mise.toml` toolchain |
| **Quick run command** | `go test -race -count=1 ./... 2>&1 \| tail -8` |
| **Full suite command** | `mise run ci` (lint + test + build + govulncheck) |
| **Estimated runtime** | ~50–70 seconds (full -race suite per phase-5/6 actuals) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race -count=1 ./internal/... 2>&1 | tail -8`
- **After every plan wave:** Run `mise run ci`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 70 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (seeded — planner fills from PLAN frontmatter) | | | | | | | | | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements (go test + fake corpus + golden locks from phases 3–6; no new framework). Docs/VM-script tasks verify by content checks (`mise run ci` + targeted greps), not a new harness.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| QA-03: working agent card in Orca (real chat-UI client), real account, BOTH auth paths | QA-03 | Requires the user's real Orca install + desktop + real account; never autonomous (D-05/D-12 posture) | End-of-phase UAT per the launch recipe (DOCS-02): configure the omp card for `zcode-acp`, run a card turn per auth path, record in UAT + ledger |
| QA-02 execution on an actual clean VM | QA-02 | The script is the repo deliverable; running a VM is the user's manual act | Run `scripts/` VM scenario on a clean Ubuntu GNOME VM; the no-keyring path asserts the coded error |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 70s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
