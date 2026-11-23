# Phase 5: Interactions & Full Chat Cycle (api-key) - Discussion Log

> **Audit trail only.** Decisions captured in CONTEXT.md.

**Date:** 2026-11-20
**Phase:** 5-Interactions & Full Chat Cycle (api-key)
**Mode:** `--auto` (standing directive; real-account ops per Phase-1 AR precedent)

---

## Responder architecture / api-key path / tools & permissions / lifecycle / real-account discipline

| Decision | Choice |
|----------|--------|
| D-01 Responder | Policy object per ARCHITECTURE Pattern 3: every interaction/* handled, per-method deadlines (180s headers, 15s rtp migrated), dedupe 1s re-announcements, unknown → protocol error never crash |
| D-02 api-key auth | updateAccountConfig (exact-revision snapshot) + requestProviderRuntimeHeaders answered {headersApplied:true, requestAuth}; VERIFY via registry read-back, never the ack; one real round-trip to prove (AR precedent) |
| D-03 Tools & permissions | tool_call/update with title/kind/status + edit diffs; requestPermission → session/request_permission; decision forwarded; ACP cancel resolves cancelled; real-tool verification = one tiny file-read turn |
| D-04 Lifecycle | SESS-01 lands WITH Phase-4-review teardown (WR-04/07 + WR-01/02/05/06 by scope); resume without replay; close releases everything |
| D-05 Real-account discipline | Minimal turns, ledgered, never CI, credentials never committed, scrubbed captures + V-EXP ids |

## Claude's Discretion
- Policy API shape, tool-event mapping details, fake scenario design, smoke-tier composition

## Deferred Ideas
- OAuth + keyring (P6); configOptions (P6); elicitation (P7 integration); session/load/fork (v2)
