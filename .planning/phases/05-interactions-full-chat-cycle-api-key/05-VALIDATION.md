---
phase: "5"
slug: "interactions-full-chat-cycle-api-key"
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-20"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract. Seeded from 05-RESEARCH.md §Validation Architecture (planner finalizes per-task rows).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib testing + -race; golden corpus (fake scenarios + .golden transcripts); table-driven TestGolden; TestEOFSignature fail-loud |
| **Config file** | none — canonical gate order build→vet→lint→test→test-experiments via mise (Phase-2 lock) |
| **Quick run command** | `go test -race ./internal/interact/... ./internal/acpfront/... ./internal/acpport/... ./internal/zcode/...` |
| **Full suite command** | `mise run ci` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** quick run command (touched packages)
- **After every plan wave:** `mise run ci`
- **Phase gate:** full suite green + full-cycle fake E2E green + guarded real-tier run ledgered before verify-work
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner finalizes; the RESEARCH map carries the 17-row contract: CORE-06 unknown/deadline/re-announce/migration, TOOL-01 lifecycle+diffs, TOOL-02 map/cancel/expiry, AUTH-01 snapshot+read-back+headers golden, SESS-01 resume/close teardown, WR-01/02/05 fixes, full-cycle fake E2E, guarded real tier) | | | CORE-06 / AUTH-01 / TOOL-01 / TOOL-02 / SESS-01 | T-5-NN | | | | ❌ W0 | ⬜ |

---

## Wave 0 Requirements

- [ ] internal/interact package skeleton (responder policy object)
- [ ] internal/hostsvc skeleton (api-key snapshot + registry read-back helper)
- [ ] fake re-announce reverse-request extension
- [ ] CredentialDecrypt seam interface (secret-tool transport; godbus stays Phase 6)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real api-key full cycle (new → prompt with tool + permission → resume → close) | AUTH-01/TOOL/SESS criteria | Requires real account + real desktop install; double guard (tag + env), never CI | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop ./test/realdesktop/ -run TestAPIKeyFullCycle` — quota ledgered in experiments/README |

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 deps
- [ ] Sampling continuity holds
- [ ] Wave 0 covers MISSING refs
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] nyquist_compliant: true in frontmatter

**Approval:** pending
