---
status: blocked
phase: 16-live-orca-acceptance
source: [16-LIVE-EVIDENCE.md]
started: 2026-11-26T00:35:00+03:00
updated: 2026-11-26T00:45:00+03:00
---

## Current Test

number: 1
name: Live Orca OMP chat returns ZCODE_ORCA_OK
expected: |
  An Orca OMP structured chat authenticates and returns the literal text
  ZCODE_ORCA_OK in `orca agent read`.
awaiting: ZCode account quota reset (2026-11-27 18:05:02)

## Tests

### 1. Live Orca OMP chat returns ZCODE_ORCA_OK
result: blocked — Orca launches OMP and auth/read-back succeeds, but every model
request returns `[1310] Weekly/Monthly Limit Exhausted` until 2026-11-27 18:05:02.

## Summary

total: 1
passed: 0
issues: 0
pending: 0
skipped: 0
blocked: 1

## Gaps

- External account quota. No wrapper-side gap was observed: discovery selected
  the macOS `.app`, the Darwin provider cache was used, Keychain auth succeeded,
  and the text-task level validation passed.
