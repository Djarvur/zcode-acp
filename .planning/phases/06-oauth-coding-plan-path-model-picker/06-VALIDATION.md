---
phase: "6"
slug: "oauth-coding-plan-path-model-picker"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-22"
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib, -race), golangci-lint strict (goswitch-style) |
| **Config file** | `.golangci.yml` (pinned), `mise.toml` toolchain |
| **Quick run command** | `go test -race -count=1 ./... 2>&1 \| tail -8` |
| **Full suite command** | `mise run ci` (lint + test + build + govulncheck) |
| **Estimated runtime** | ~50–70 seconds (full -race suite per Phase-5 actuals) |

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

- Existing infrastructure covers all phase requirements (go test + fake app-server corpus from Phases 3–5; no new framework).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real coding-plan authenticate end-to-end (keyring decrypt + PAT mint + registry accepted + one real turn) | AUTH-02 | Real account + desktop keyring on the local machine; CI never touches it (D-05/D-12/D-13) | Guarded manual tier per Phase-5 AR precedent: desktop logged in, run wrapper on real install, ledger spend in experiments/README, scrubbed capture + V-EXP id |
| Keyring-absent fail-fast on the real machine (AUTH-04 probe arms) | AUTH-04 | Requires an actual headless env / real keyring state | Observe coded actionable error with no hang; research already probed both arms live (rc=1 fast) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 70s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
