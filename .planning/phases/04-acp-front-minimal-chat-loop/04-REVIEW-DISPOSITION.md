---
phase: "4"
slug: "acp-front-minimal-chat-loop"
review: 04-REVIEW.md
status: issues_found
critical: 0
warning: 7
info: 5
total: 12
updated: "2026-11-20"
---

# Phase 4 — Review Disposition

> Per-finding disposition ledger for `04-REVIEW.md` (advisory, never blocks).

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | OnRouterEvent slot stolen by every session/new (non-latest machine loses degraded-resolution path at 2+ sessions) — **Phase 5 owns this**: SESS-01/registry scope delivers per-session machine wiring; the fan-out extension lands with multi-session (supersede single-slot with ordered observers, extending 04-01's AddObserver pattern) |
| WR-02 | warning | open | bindSink publishes a.sink unsynchronized (formal memory-model race, unobservable today) — one-line atomic publish; batch with Phase 5 seam touch |
| WR-03 | warning | open | tap's inline ACP write on backend read goroutine — stalled client wedges both legs (violates the hook must-not-block contract) — Phase 5 design decision: buffered streaming queue or dedicated writer goroutine; needs a deliberate design, not a drive-by |
| WR-04/WR-07 | warning | open | error-path orphan backend sessions; no teardown (sessions/observers/machines never released) — **Phase 5 owns teardown** (SESS-01 session/close + registry release; NewMachine must expose observer remove) |
| WR-05/WR-06 | warning | open | front-wide rejection message wording ("for this session" vs front-wide); image/audio/resource blocks dropped silently — wording fix + a log line; batch with Phase 5 |
| IN-01..IN-05 | info | open | (see 04-REVIEW.md) minor hardening notes — batched with the Phase 5 multi-session pass |

**Open:** 12 of 12 (all routed to Phase 5, which owns multi-session/teardown/permission scope where these belong) · **Fixed:** 0 · **Accepted:** 0

**Rationale:** Phase 4's scope is the minimal single-turn loop; every warning lives in the multi-session/teardown/permission space that Phase 5 explicitly owns (SESS-01, CORE-06). No critical findings; nothing blocks the phase goal. The ledger stays open so Phase 5 planning MUST cite it.
