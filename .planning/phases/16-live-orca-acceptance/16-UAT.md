---
status: passed
phase: 16-live-orca-acceptance
source: [16-LIVE-EVIDENCE.md]
started: 2026-11-26T00:35:00+03:00
updated: 2026-11-26T03:17:00+03:00
---

## Current Test

number: 1
name: Live Orca OMP chat returns ZCODE_ORCA_OK
expected: |
  An Orca OMP structured chat authenticates and returns the literal text
  ZCODE_ORCA_OK in `orca agent read`.
awaiting: none

## Tests

### 1. Live Orca OMP chat returns ZCODE_ORCA_OK
result: pass — Orca session `2db66245-c8cf-4ff6-9dd0-77343fac7c59` returned
`ZCODE_ORCA_OK`; root cause was the custom model's missing maxOutputTokens cap
(backend asked 247486 completion tokens against a 262144-token context), fixed
by adding `optionSpecs.maxOutputTokens.max = 32768` to the local model rule.

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

- Orca's conversation-name text task still uses the desktop default
  `zai-api/GLM-5.3`, whose coding-plan quota is exhausted until 2026-11-27
  18:05:02. Chat turns using the selected custom model work.
