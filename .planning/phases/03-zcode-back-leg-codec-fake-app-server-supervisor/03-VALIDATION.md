---
phase: "3"
slug: "zcode-back-leg-codec-fake-app-server-supervisor"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-19"
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` under `go test -race`; helper-process reexec for the standalone-binary tier |
| **Config file** | none new — existing `mise.toml` (`[tasks.test] run = "go test -race -count=1 ./..."`) |
| **Quick run command** | `go test -race ./internal/zcode/... ./internal/supervisor/... ./internal/turn/...` |
| **Full suite command** | `mise run ci` (build → vet → lint → test → test-experiments) |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** `go test -race ./internal/...` (packages touched)
- **After every plan wave:** `mise run ci`
- **Before `/gsd:verify-work`:** full suite green + golden corpus replayed + fake-binary reexec test green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (seeded by planner) | 03-xx | — | BRIDGE-01 | — | jsonrpc never reaches child (byte-level pin) | unit (-race) | `go test -race ./internal/zcode -run TestEnvelope` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 03-xx | — | BRIDGE-01 | — | single-writer ordering + bypass lane | unit (-race) | `go test -race ./internal/zcode -run 'TestWriterOrdering\|TestBypass'` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 03-xx | — | BRIDGE-02 | — | golden corpus replays; exit-0-zero-lines is failure | golden/harness | `go test -race ./internal/zcode/fake -run 'TestGolden\|TestFakeBinary\|TestEOFSignature'` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 03-xx | — | BRIDGE-03 | — | discovery positive/negative + spawn env scrub + Setpgid | unit/integration | `go test -race ./internal/supervisor -run 'TestDiscover\|TestSpawnEnv'` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 03-xx | — | BRIDGE-03 | — | group-kill leaves no survivors; restart window 3/60s | integration | `go test -race ./internal/supervisor -run 'TestGroupKill\|TestRestartWindow'` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 03-xx | — | BRIDGE-04 | — | exactly-once turn resolution; cancel; -32010 | golden/unit | `go test -race ./internal/turn -run 'TestHappyTurn\|TestCancelMidTurn\|TestDoubleSend\|TestCompletionVariants'` | ❌ W0 | ⬜ pending |
| (seeded by planner) | 03-xx | — | BRIDGE-03/04 | — | real-desktop positive path | manual-only (env-guarded) | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop ./internal/supervisor -run TestRealDesktop` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Go package skeletons: `internal/zcode`, `internal/zcode/fake`, `internal/supervisor`, `internal/turn` (+ `-tags realdesktop` guard)
- [ ] Golden-corpus scenario files under `testdata/` (10 scenarios per 03-RESEARCH corpus table)
- [ ] No framework installs: stdlib `testing` only; goldie/v2 explicitly excluded (hand-rolled diff per STACK.md)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real-desktop positive path (locate → spawn → create → close, no orphans) | BRIDGE-03 | Requires the real desktop install; env-guarded, never CI | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop ./internal/supervisor -run TestRealDesktop` on a machine with ZCode installed (this machine qualifies) |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
