---
phase: "1"
slug: "research-protocol-documentation"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: true
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
| 01-01-T1 | 01 | 1 | D-02/D-04 (CONTEXT) | T-1-01, T-1-02 | no secret shapes; legend defined once; path-shape grep | structure+gate check | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | created in-task | ⬜ pending |
| 01-01-T2 | 01 | 1 | D-02 (CONTEXT) | T-1-01 | scrubber rules tested (RED before GREEN) | unit tests (node --test) | `node --test experiments/` | ❌ W0 | ⬜ pending |
| 01-01-T3 | 01 | 1 | DISC-04 | T-1-01, T-1-07 | V-EXP id resolves; capture header records revision | tag+gate check | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | ❌ W0 | ⬜ pending |
| 01-02-T1 | 02 | 2 | DISC-01 | T-1-01, T-1-03 | capture verdict fields; no key-minting endpoint | capture header+gate | `bash experiments/scan-secrets.sh` | ❌ W1 | ⬜ pending |
| 01-02-T2 | 02 | 2 | DISC-01 | T-1-03, T-1-04 | one tiny turn max; honest degradation; quota ledger | capture header+gate | `bash experiments/scan-secrets.sh` | ❌ W1 | ⬜ pending |
| 01-03-T1 | 03 | 3 | DISC-01 | T-1-08 | tag grammar; capture ids cited and resolving | structure+tag check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| 01-03-T2 | 03 | 3 | DISC-01 | T-1-08 | drift baseline first-class; 3.17.x version-pinned | structure+tag check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| 01-04-T1 | 04 | 2 | DISC-02 | T-1-09 | chunk citations; build named in capture | capture header+gate | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | ❌ W1 | ⬜ pending |
| 01-04-T2 | 04 | 2 | DISC-02 | T-1-09 | neutral phrasing; known-unknowns stay V-RPT | structure+tag check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| 01-05-T1 | 05 | 2 | DISC-03 | T-1-06, T-1-11 | no package installs; pristine byte-compare recorded | capture header+gate | `bash experiments/check-docs.sh && bash experiments/scan-secrets.sh` | ❌ W1 | ⬜ pending |
| 01-05-T2 | 05 | 2 | DISC-03 | T-1-06 | matrix rows cite recorded commands | structure check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |
| 01-06-T1 | 06 | 2 | DISC-05 | T-1-10 | license live-verified; nothing upstream committed | CLI check | `gh api repos/william0wang/zcode-acp --jq '.license.spdx_id'` | n/a (network) | ⬜ pending |
| 01-06-T2 | 06 | 2 | DISC-05 | T-1-10 | Apache-2.0 attribution section present | structure check | `bash experiments/check-docs.sh` | ❌ W0 | ⬜ pending |

*File Exists: ❌ W0 = created by plan 01-01 (Wave 0); ❌ W1 = created by the capture task in the same row; "created in-task" = the task itself authors the files the command runs.*

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
