---
status: testing
phase: 01-Research & Protocol Documentation
source: [01-VERIFICATION.md]
started: 2026-11-19T09:10:00Z
updated: 2026-11-19T09:10:00Z
---

## Current Test

number: 1
name: MVP goal format — ROADMAP Phase 1 goal into canonical User Story form
expected: |
  ROADMAP.md declares `mode: mvp` but the Phase 1 goal line is not in user-story
  format (`user-story.validate` → valid: false). Run /gsd:mvp-phase 1 (or manually
  reformat the goal) so the goal validates; plan 01-01's reframe (valid: true) is
  the shape to match.
awaiting: user response

## Tests

### 1. MVP goal format
expected: ROADMAP Phase 1 goal validates as a canonical User Story (run /gsd:mvp-phase 1 or approve the existing reframed form in plan 01-01 as the recorded intent).
result: [pending]

### 2. Provenance spot-check
expected: ~5 load-bearing V-SRC citations re-derive against the machine-local sources (3.18.2 bundle, 3.17.x clone, Orca AppImage extract, upstream v0.89.0). Suggested set: the drift-baseline 4 bundle-only method strings, the omp launch argv literal (EXP-SESSIONDIR-01), the workspaceKey convention (workspaceIdentity?.trim() || workspacePath), the revision formula sha256(path.resolve(...)), the v0.89.0 no-headless-mint claim. The repo cannot self-verify these by design — machine paths required.
result: [pending]

### 3. Quota accounting framing
expected: Honest ledger of real-account spend accepted: 3 tiny sends issued, 0 completed (2 mis-armed quiet-window retries documented as Rule-1 deviations; mid-turn requestCredentialCipher forced the D-05 V-SRC-only degradation). Ledger at experiments/README.md:183. Judgment: accept the framing, or require a tighter experiment protocol in Phase 6.
result: [pending]

### 4. Citation-honesty skim (DISC-02 judgment-tier prohibition)
expected: docs/protocol/acp-client-dialect.md — every client-build claim carries a chunk citation or is honestly V-RPT; build-drift warning present. Non-authoritative LLM-judge pre-verdict: PASS (43 chunk citations, 11 V-RPT). Judgment: skim and confirm or flag.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
