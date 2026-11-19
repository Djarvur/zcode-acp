---
phase: 01
review: 01-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "Scrubber `user_info` redaction never fires on the wrapped/escaped wire-line form the probe writes"
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "Probe exits 0 (success) on a zero-capture run whenever the child exits non-zero"
  - id: WR-02
    severity: warning
    disposition: fixed
    title: "Documented example `--window-ms 12000` expires before the last step of both 01-02 flows fires (13000 / 13500 ms)"
  - id: IN-01
    severity: info
    disposition: fixed
    title: "Header claims \"the two remaining known-unknowns\" — the Open items section lists three"
  - id: IN-02
    severity: info
    disposition: fixed
    title: "Build-match claim stated at two different strengths — \"byte-identical\" vs \"byte-count-identical\""
  - id: IN-03
    severity: info
    disposition: fixed
    title: "`EXP-ORCASRC-01` is cited in prose only — its capture header escapes the resolution and continuity gate"
  - id: IN-04
    severity: info
    disposition: fixed
    title: "`observeReply` conflates server→host requests with replies (latent verdict falsification path)"
  - id: WR-03
    severity: warning
    disposition: open
  - id: WR-04
    severity: warning
    disposition: open
open: 2
total: 9
recorded: 2026-11-19T18:25:05.809Z
---

# Phase 01: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | fixed | 01-REVIEW-FIX.md |
| WR-01 | warning | fixed | 01-REVIEW-FIX.md |
| WR-02 | warning | fixed | 01-REVIEW-FIX.md |
| IN-01 | info | fixed | 01-REVIEW-FIX.md |
| IN-02 | info | fixed | 01-REVIEW-FIX.md |
| IN-03 | info | fixed | 01-REVIEW-FIX.md |
| IN-04 | info | fixed | 01-REVIEW-FIX.md |
| WR-03 | warning | open | scrub-capture.mjs bare-home rewrite lacks boundary check — needs TDD fix + unit test; batch with Phase 2 gate hardening (not in the current review) |
| WR-04 | warning | open | probe harness hardcodes desktopVersion:'3.18.2' — read at capture time like builtinRevision; batch with Phase 3 harness work (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
