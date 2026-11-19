---
phase: "4"
slug: "acp-front-minimal-chat-loop"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: draft
nyquist_compliant: true
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
| TestObserverFanOut | 04-01 | 1 | infra (enables CORE-01..05) | — | ordered fan-out, no hook theft, remove | unit (TDD) | `go test -race -count=1 -run TestObserverFanOut ./internal/zcode/` | ❌ W0 (04-01) | ⬜ |
| TestMinimalChatLoop | 04-02 | 2 | CORE-01..05 (tracer union) | T-4-SC, T-4-02 | honest loop end-to-end | integration (tracer, TDD) | `go test -race -count=1 -run TestMinimalChatLoop ./internal/acpfront/` | ❌ W0 (04-02) | ⬜ |
| TestSDKImportBoundary | 04-02 | 2 | infra (T-4-SC) | T-4-SC | SDK behind the seam only | boundary test | `go test -race -count=1 -run TestSDKImportBoundary ./internal/acpport/` | ❌ W0 (04-02) | ⬜ |
| TestInitializeHonestCapabilities + TestAuthenticateHonestError + TestUnadvertisedStableMethodsAnswerNotFound | 04-03 | 3 | CORE-01 | T-4-04 | honest caps (zero fs/terminal/loadSession fields); honest auth failure; -32601 defaults | integration + transcript | `go test -race -count=1 -run 'TestInitializeHonestCapabilities\|TestAuthenticateHonestError\|TestUnadvertisedStableMethodsAnswerNotFound' ./internal/acpfront/ ./internal/acpport/` | ❌ W0 (04-03) | ⬜ |
| TestNewSessionCreatesAndSetsModel + TestNewSessionRejectsRelativeCwd + TestNewSessionAcceptsMcpServers (+ TestGolden corpus) | 04-02 (corpus) / 04-03 (locks) | 2/3 | CORE-02 | T-4-01 | setModel follow-up on wire; relative cwd rejected pre-wire; mcpServers accepted | integration + golden | `go test -race -count=1 -run 'TestNewSession\|TestGolden' ./internal/acpfront/ ./internal/zcode/fake/` | ❌ W0 (04-02/03) | ⬜ |
| TestThoughtChunks | 04-04 | 3 | CORE-04 | — | thought only on reasoning events (negative locked) | integration + golden | `go test -race -count=1 -run TestThoughtChunks ./internal/acpfront/` | ❌ W0 (04-04) | ⬜ |
| TestStreamBeforeStopReason | 04-04 | 3 | CORE-03 | — | deltas precede response; stopReason after flush; both completion variants; zero-delta honesty | integration (proof) | `go test -race -count=1 -run TestStreamBeforeStopReason ./internal/acpfront/` | ❌ W0 (04-04) | ⬜ |
| TestCancelMidTurn + TestSecondPromptRejected + TestCancelAfterCompleteNoOp | 04-05 | 4 | CORE-05 | T-4-05, T-4-02 | cancelled exactly once, never -32800; clean rejection; silent no-op | integration | `go test -race -count=1 -run 'TestCancelMidTurn\|TestSecondPromptRejected\|TestCancelAfterCompleteNoOp' ./internal/acpfront/` | ❌ W0 (04-05) | ⬜ |
| TestEndToEndExecChain (Phase 3, regression) + TestAgentBinaryEndToEnd | 04-05 | 4 | CORE-05 (exec leg) | — | binary-level loop over exec pipes, fake backend | exec-chain | `go test -race -count=1 -run 'TestEndToEndExecChain\|TestAgentBinaryEndToEnd' ./internal/supervisor/ ./cmd/zcode-acp/` | ❌ W0 (04-05) | ⬜ |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Package skeletons: `internal/acpport`, `internal/acpfront`, `internal/core` (+ SDK dep added to go.mod per D-01 audit) — plan 04-01 (dep) + 04-02 (skeletons)
- [x] Fake corpus additions: `setmodel-followup`, `cancel-mid-turn` front preamble (04-02); `part-delta-streaming`, `no-reasoning` (04-04) (+ goldens + drivers; the `mcp-servers-ignored` behavior is locked at the FRONT tier over the setmodel-followup scenario — the back leg has no mcp wire surface to script; see plan 04-03)
- [x] Test ACP client (in-process over pipes) for the front-tier integration tests — plan 04-02 Task 2 (harness_test.go)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real ACP client spot check (spawn cmd/zcode-acp under a real client, fake wired via supervisor Override) | phase-level | Requires a real client binary; precedent = Phase 3 realdesktop pattern | once at phase end, non-gating (enabled by plan 04-05's --app-server/override seam; see its human-check element) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (every task in plans 04-01..04-05 carries automated + fails_when)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (mapped to plans 04-01/04-02/04-04 above)
- [x] No watch-mode flags
- [x] Feedback latency < 90s (quick run command is the four-package suite; quiet-window rides add ~4s each)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planned (2026-11-20) — per-task map finalized during plan-phase; executor updates Status column per task
