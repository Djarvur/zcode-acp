---
phase: "2"
slug: "scaffold-ci-bootstrap"
review: 02-REVIEW.md
status: issues_found
critical: 1
warning: 2
info: 6
total: 9
updated: 2026-11-19
---

# Phase 2 — Review Disposition

> Per-finding disposition ledger for `02-REVIEW.md` (advisory, never blocks).
> Recorded by the execute-phase code-review gate. Default disposition: `open`.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | fixed | scrubber USER_INFO_RX lastIndex leak (multi-object lines) — TDD RED→GREEN commits 3969eae/4ed831a, multi-object + fixed-point regressions, suite 20/20; T-2-01 re-audited and re-closed in 02-SECURITY.md |
| WR-01 | warning | fixed | timeout-minutes: 15 on all 5 jobs (3c13b6b); e2e-matrix keeps its deliberate 150 |
| WR-02 | warning | open | probe classifyReply collision risk with numeric server→host ids — route to Phase 3 (real supervisor owns request/response demux; the probe's string-id observation holds for 3.18.2) |
| IN-01 | info | open | WR-03 lookahead ASCII-only — non-ASCII homes are EXT-02 territory (PR-driven); noted for Phase 3 backlog |
| IN-02 | info | open | --redact-path '~/x' silent no-op — minor UX; Phase 3 backlog |
| IN-03 | info | open | govulncheck@latest floating pin under required context — pin at Phase 7 release hardening |
| IN-04 | info | open | scan-secrets fail-open on git grep exit 2 (subtree case) — Phase 3 backlog |
| IN-05 | info | accepted | curl mise.run \| sh trust root — documented in 02-RESEARCH Package Legitimacy Audit (official installer channel, pinned versions); accepted with that rationale |
| IN-06 | info | open | header-dialect mismatch scrubber vs check-docs (f) compact-only grep — Phase 3 backlog (both gates currently green on committed captures) |

**Open:** 7 of 9 · **Fixed:** 2 (incl. the critical) · **Accepted:** 1
