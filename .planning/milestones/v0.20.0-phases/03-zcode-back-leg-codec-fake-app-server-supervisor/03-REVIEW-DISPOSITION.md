---
phase: "3"
slug: "zcode-back-leg-codec-fake-app-server-supervisor"
review: 03-REVIEW.md
status: issues_found
critical: 1
warning: 4
info: 3
total: 8
updated: "2026-11-20"
---

# Phase 3 — Review Disposition

> Per-finding disposition ledger for `03-REVIEW.md` (advisory, never blocks).

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | fixed | crash-loop exit skipped staged shutdown → orphans; TDD RED (real grandchild survivor asserted ESRCH) → GREEN shutdown-on-every-exit-path; commits 6794626/5bbb38f |
| WR-01 | warning | fixed | stream death / Close never unblocked in-flight Calls; TDD RED (bounded-resolution tests wedged at 2s) → GREEN drain-all + ErrConnClosed + readable firstErr; commits 2e2b299/ff96773 |
| WR-02 | warning | open | hooks assigned after NewConn race the read loop (documented contract, unsynchronized per Go memory model) — route to Phase 4 when acpport composes Conn with hooks (constructor-injection refactor belongs at the integration seam) |
| WR-03 | warning | open | terminal classifier ignores sessionId (safe under D-04 one-session-per-child) — Phase 5 multi-session hardening: add sessionId check when sessions multiplex beyond 1:1 |
| WR-04 | warning | open | fake Reply.validate accepts result+error together; result-only unknown_method silently defaults -32601 — Phase 3 scope ships fake as CI instrument; fail-loud tightening batched with Phase 5 fake-policy work |
| IN-01 | info | open | JSON null id decodes as integer 0 — envelope id-type discrimination note; batch with WR-02 seam work |
| IN-02 | info | open | Wait() blocks forever on error-path turns (resolvedCh unclosed) — no current caller uses Wait() on error paths; batch with Phase 4 integration |
| IN-03 | info | open | cancel-before-send bypass race (stop precedes send on the wire) — probe-verified 3.18.2 tolerates; revisit if a real client exhibits it (Phase 7 hardening) |

**Open:** 6 of 8 (all routed to later phases with named seams) · **Fixed:** 2 (the blocker + the highest-impact warning)
