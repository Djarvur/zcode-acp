---
phase: "4"
slug: "acp-front-minimal-chat-loop"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-20"
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (house style), `-race` mandatory |
| **Config file** | none — `go test` + mise tasks (existing) |
| **Quick run command** | `go test -race -count=1 ./internal/acpport/... ./internal/acpfront/... ./internal/zcode/... ./internal/turn/...` |
| **Full suite command** | `mise run ci` (build → vet → lint → test -race → test-experiments) |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** quick run command (four packages)
- **After every plan wave:** `mise run ci`
- **Phase gate:** full suite green + exec-chain E2E green (happy + cancel in default suite)
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (planner) | 04-xx | — | CORE-01 | — | honest caps (zero fs/terminal/loadSession fields) | integration + transcript | `go test -race -run TestInitializeHonestCapabilities ./internal/acpfront/` | ❌ W0 | ⬜ |
| (planner) | 04-xx | — | CORE-02 | — | setModel follow-up on wire; mcpServers accepted | integration + golden | `go test -race -run 'TestGolden\|TestNewSessionCreatesAndSetsModel' ./...` | ❌ W0 | ⬜ |
| (planner) | 04-xx | — | CORE-03 | — | deltas precede response; stopReason after flush | integration (proof) | `go test -race -run TestStreamBeforeStopReason ./internal/acpfront/` | ❌ W0 | ⬜ |
| (planner) | 04-xx | — | CORE-04 | — | thought only on reasoning events | integration + golden | `go test -race -run TestThoughtChunks ./internal/acpfront/` | ❌ W0 | ⬜ |
| (planner) | 04-xx | — | CORE-05 | — | cancelled exactly once, never -32800 | integration + exec-chain | `go test -race -run 'TestCancelMidTurn\|TestEndToEndExecChain' ./...` | ❌ W0 | ⬜ |
| (planner) | 04-xx | — | infra | — | observer fan-out (multi-session) | unit | `go test -race -run TestObserverFanOut ./internal/zcode/` | ❌ W0 | ⬜ |
| (planner) | 04-xx | — | infra (T-4-SC) | — | SDK behind the seam only | boundary test | `go test -run TestSDKImportBoundary ./internal/acpport/` | ❌ W0 | ⬜ |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Package skeletons: `internal/acpport`, `internal/acpfront` (+ SDK dep added to go.mod per D-01 audit)
- [ ] Fake corpus additions: `setmodel-followup`, `part-delta-streaming`, `no-reasoning` (+ goldens)
- [ ] Test ACP client (in-process over pipes) for the front-tier integration tests

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real ACP client spot check (spawn cmd/zcode-acp under a real client, fake wired via supervisor Override) | phase-level | Requires a real client binary; precedent = Phase 3 realdesktop pattern | once at phase end, non-gating |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
