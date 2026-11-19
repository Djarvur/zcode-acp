---
status: testing
phase: 01-Research & Protocol Documentation
source: [01-VERIFICATION.md]
started: 2026-11-19T09:10:00Z
updated: 2026-11-19T11:40:00Z
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
expected: ROADMAP Phase 1 goal validates as a canonical User Story.
result: pass (2026-11-19, agent) — goal rewritten to the validated user story from plan 01-01 (user-story.validate: valid:true, all slots), committed 860e98d; approved by user

### 2. Provenance spot-check (COMPLETED — agent-run with machine paths, user-approved approach)
expected: ~5 load-bearing V-SRC citations re-derive against machine-local sources against the machine-local sources (3.18.2 bundle, 3.17.x clone, Orca AppImage extract, upstream v0.89.0). Suggested set: the drift-baseline 4 bundle-only method strings, the omp launch argv literal (EXP-SESSIONDIR-01), the workspaceKey convention (workspaceIdentity?.trim() || workspacePath), the revision formula sha256(path.resolve(...)), the v0.89.0 no-headless-mint claim. The repo cannot self-verify these by design — machine paths required.
result: pass (2026-11-19, agent) — 5/5 PASS with independent re-derivation: drift baseline re-computed via comm -13 diff of the two 110-method maps (exactly the 4 named strings); omp argv literal read at byte offsets from the current-build chunk; workspaceKey at zcode-protocol-entrypoint.ts:227-236 + env const; revision digest e9c0674a… reproduced exactly (path-string hash, not contents); upstream v0.89.0 has exactly 3 HTTP requests, no POST to api_keys, project_token_key_missing present

### 3. Quota accounting framing
expected: Honest ledger of real-account spend accepted: 3 tiny sends issued, 0 completed (2 mis-armed quiet-window retries documented as Rule-1 deviations; mid-turn requestCredentialCipher forced the D-05 V-SRC-only degradation). Ledger at experiments/README.md:183. Judgment: accept the framing, or require a tighter experiment protocol in Phase 6.
result: pass (2026-11-19, user) — quota framing accepted as-is; spend practically zero (3 truncated requests, no model replies)

### 4. Citation-honesty + source cross-verification (DISC-02)
expected: every client-build claim carries a chunk citation or is honestly V-RPT; build-drift warning present.
result: pass (2026-11-19) — user challenged the closed-source premise; stablyai/orca re-verified PUBLIC/MIT (the early 404 was transient VPN failure; research corrected, commit 9b4534b). Targeted supplement re-verified the whole doc against public source @v1.4.262: ~20 claim groups dual-cited, 3 CORRECTED (--session-dir belongs to discovery processes, not chat launch; fullAccess=false only for discovery sessions; modelDiscovery models listed pre-session via probe session), +2 methods (session/close, elicitation/complete), 3 items stay honestly V-RPT. Commit ed897a8, capture EXP-ORCASRC-01, gates green.

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps
