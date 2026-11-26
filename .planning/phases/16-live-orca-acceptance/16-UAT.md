---
status: blocked
phase: 16-live-orca-acceptance
source: [16-LIVE-EVIDENCE.md]
started: 2026-11-26T00:35:00+03:00
updated: 2026-11-26T11:55:37+03:00
---

## Current Test

number: 1
name: Live Orca ACP flow on the intended z.ai account
expected: |
  A human runs the current Mac Orca card against the intended z.ai account
  through initialize, session/new, a prompt that emits a tool-call, a
  permission request with an explicit decision, and a streamed final reply.
awaiting: z.ai quota reset and a human-run acceptance session

## Tests

### 1. Live Orca ACP flow on the intended z.ai account
result: blocked — the intended z.ai coding-plan quota is exhausted (`[1310]`,
reset 2026-11-27 18:05:02). The recorded custom DeepSeek text exchange is
partial diagnostic evidence only; it is not the intended account and does not
prove tool-calls or permission requests. No live test passed in this phase.

## Summary

total: 1
passed: 0
issues: 0
pending: 0
skipped: 0
blocked: 1

## Gaps

- MORCA-02 remains incomplete until the intended-account flow above passes.
- After quota is available, a human must run the current Mac Orca card against
  the intended z.ai account through `initialize`, `session/new`, a prompt that
  emits a tool-call, a permission request with an explicit decision, and a
  streamed final reply.
- Record scrubbed environment, ZCode version, launch command, authentication
  path, flow, and outcome evidence. Do not promote the custom-provider text
  result into live tool-call or permission acceptance.
