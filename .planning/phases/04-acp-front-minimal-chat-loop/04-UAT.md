---
status: complete
phase: 04-ACP Front — Minimal Chat Loop
source: [04-VERIFICATION.md]
started: 2026-11-20T03:30:00Z
updated: 2026-11-20T03:21:10.111Z
---

## Current Test

[testing complete]

## Tests

### 1. CORE-01 concurrency backstop (DISCHARGED — agent-run with machine evidence)
expected: Concurrent dispatch over one connection — id-matched responses, no crossed frames, registry resolution.
result: pass (2026-11-20, agent) — `TestConcurrentRequestDispatch` written (core01_concurrency_test.go, 3 assertion layers: typed caller identity / registry resolution via real prompt / raw wire pairing 3-request-1-response-each), `-race -count=3` PASS 3/3, commit 19b726b. Verifier abstention discharged with machine evidence.

### 2. Real-client spot check
expected: Built binary under a real ACP v1 client (`--app-server` shim) — initialize → session/new → streamed turn → mid-turn cancel, exactly-once cancelled.
result: pass-with-scope-note (2026-11-20, agent) — `TestAgentBinaryEndToEnd` drives the BUILT binary over real exec pipes under the SDK's ClientSideConnection (the reference protocol client implementation): initialize → session/new → end_turn loop → mid-turn cancel → staged shutdown, all green. A NON-SDK real client (Orca chat UI) is the Phase 7 manual-E2E done-criterion (QA-03) — recorded there.

### 3. Five judgment-tier prohibitions (advertise=implemented; no fake auth success; never synthesize chunks; stdout protocol-exclusive; no paths/content in errors)
expected: Each judged SATISFIED with cited evidence.
result: pass (2026-11-20, user) — pre-approved per standing autonomous directive; verifier's non-authoritative judgments all SATISFIED with cited tests (honesty locks, tracked-no-answer, never-synthesized negative, stdout golden, structural-fields-only grep). One recorded nuance: generic NewInternal("%v") paths pass transport-error text through un-pinned — routed to Phase 5 error-hygiene pass with WR-02.

## Summary

total: 3
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
