---
phase: "1"
slug: "research-protocol-documentation"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-19"
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Shell check scripts (no test framework exists yet; greenfield docs phase). Node available for helpers. |
| **Config file** | none — Wave 0 installs `experiments/check-docs.sh` + `experiments/scan-secrets.sh` |
| **Quick run command** | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` |
| **Full suite command** | Same + `git grep` provenance coverage audit (every V-EXP id in docs resolves to an existing `experiments/captures/` file) |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh`
- **After every plan wave:** Run full suite + human skim of changed doc sections (provenance tags are judgment-bearing)
- **Before `/gsd:verify-work`:** Full suite must be green; all five docs pass; every V-EXP id resolves; secret-scan clean
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (seeded by planner) | 01 | 1 | DISC-01 | — | N/A | structure+tag check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 01 | 1 | DISC-02 | — | neutral phrasing (no hidden-goal wording) | structure+tag check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 01 | 1 | DISC-03 | — | N/A | structure check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 01 | 1 | DISC-04 | — | N/A | tag check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 01 | 1 | DISC-05 | — | Apache-2.0 attribution present | structure check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 01 | 1 | D-02 (CONTEXT) | — | no secret shapes in tracked files | secret-scan + id-resolution | `bash experiments/scan-secrets.sh` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 01 | 1 | D-04 (CONTEXT) | — | N/A | structure check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `experiments/check-docs.sh` — structure + provenance-tag conformance for all five docs
- [ ] `experiments/scan-secrets.sh` — secret-shape gate (JWT / `enc:v2:` / PAT regexes; Phase 2 CI reuses it)
- [ ] `experiments/scrub-capture.mjs` — canonical scrubber (must exist BEFORE the first capture is written)
- [ ] `experiments/README.md` — how to run E1–E5; scrub-first rules; "never in CI" warning
- [ ] `docs/protocol/README.md` — index + provenance legend (needed before claim-ledger docs land)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Bundle-derived claims re-derivable (spot-check method strings via grep) | DISC-01 | Bundle path is machine-private (supplied in agent prompts, never committed) | `grep -o 'sessionSubscribe:"session/subscribe"' <bundle>` — record result in the doc |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
