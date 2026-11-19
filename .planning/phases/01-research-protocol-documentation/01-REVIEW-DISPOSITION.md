---
phase: "1"
slug: "research-protocol-documentation"
review: 01-REVIEW.md
status: has-findings
critical: 0
warning: 4
info: 4
total: 8
updated: 2026-11-19
---

# Phase 1 — Review Disposition

> Per-finding disposition ledger for `01-REVIEW.md` (advisory, never blocks).
> Recorded by the execute-phase code-review gate. Default disposition: `open`.
> Hand-edit the Disposition column to triage (`fixed` / `skipped` / `deferred`); keep the reason in Source.

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | scan-secrets.sh credential-body class omits `.`/`:` vs scrubber CRED_RX — align when the gate is adopted into Phase 2 CI (VALIDATION.md already schedules that reuse) |
| WR-02 | warning | open | app-server-dialect.md:146 error-code recipe needs `grep -o -e '-32603'` — doc-recipe fix, safe to batch with the next dialect-doc touch |
| WR-03 | warning | open | scrub-capture.mjs bare-home rewrite lacks boundary check — needs TDD fix + unit test; batch with Phase 2 gate hardening |
| WR-04 | warning | open | probe harness hardcodes desktopVersion:'3.18.2' — read at capture time like builtinRevision; batch with Phase 3 harness work |
| IN-01 | info | open | dialect doc extraction command (a) drops digit-bearing v4 strings — doc-recipe fix, batch with WR-02 |
| IN-02 | info | open | dialect doc singletons row label (8) vs 9 listed — trivial doc fix, batch with WR-02 |
| IN-03 | info | open | scan-secrets.sh CWD-relative git grep — add toplevel pin; batch with WR-01 (same script) |
| IN-04 | info | open | scrub-capture.mjs line-1 byte-identity on header-less captures — batch with WR-03 |

**Open:** 8 of 8
