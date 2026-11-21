---
phase: "5"
slug: "interactions-full-chat-cycle-api-key"
review: 05-REVIEW.md
status: issues_found
critical: 1
warning: 4
info: 3
total: 8
updated: "2026-11-21"
---

# Phase 5 — Review Disposition

> Per-finding disposition ledger for `05-REVIEW.md` (advisory, never blocks).

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | open | `drainActiveTurn` returns false after a successful drain — `CloseSession` skips the permission-bridge cancel fan-in exactly on the close-with-active-turn path; cancelled turn's pending permission dangles to the 300s responder deadline then frames a deny at a closed backend session. In-scope for the phase goal (permission decisions answered from the client); must be fixed and test-covered (close-with-pending-permission) before REAL RUN UAT item on permissions can be trusted |
| WR-01 | warning | open | Parked dedupe-join arm frames a bare id-echo reply at conn close, contradicting Serve's never-send invariant (`responder.go` + `conn.go` bypass lane window) |
| WR-02 | warning | open | `ResumeSession` drops the client cwd (SDK `ResumeSessionRequest.Cwd` exists, v0.20.0) — resumed sessions get no `cwds` entry, relative diff/location paths (Pitfall-5 on the resume path) |
| WR-03 | warning | open | `session/resume` not gated on the closing set — close-vs-resume race leaks a machine's conn observers (WR-07-class from 04-REVIEW) |
| WR-04 | warning | open | Prompt's cancel fan-in goroutine can wake on both ctx arms and cancel a later, unrelated turn on the same machine — needs arm ordering synchronized with `release()` |
| IN-01 | info | open | realdesktop `resolveModelSelection` silently ignores malformed `ZCODE_ACP_MODEL` (production exits 2 on the same shape) — real spend on an unintended model |
| IN-02 | info | open | raw-capture path not enforced outside the repo — the tee holds the live api-key; "born OUTSIDE" rule lives only in comments |
| IN-03 | info | open | main's shutdown select can mask a child crash with exit 0 when it races client disconnect |

**Open:** 8 of 8 (CR-01 gates the REAL RUN UAT retry; WR-01..04 batch with the CR-01 close-path fix; IN-01..03 hardening backlog) · **Fixed:** 0 · **Accepted:** 0

**Rationale:** Fresh deep review of the 42-file phase-5 scope found one critical correctness inversion on the close/permission path the phase goal explicitly owns. The phase is already stopped on human UAT (REAL RUN attempt 1: auth read-back gap); CR-01 joins the pre-UAT fix list. Ledger stays open so the gap-closure/UAT fix plan MUST cite it.
