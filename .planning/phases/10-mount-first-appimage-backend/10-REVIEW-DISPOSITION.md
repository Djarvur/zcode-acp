---
phase: 10
review: 10-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: fixed
    title: "Signal during mount acquisition discards a failed mount teardown (exit 0, stale mount / orphan helper)"
  - id: WR-01
    severity: warning
    disposition: fixed
    title: "dpkg probe can never detect an installed package (status match is off by a leading space)"
  - id: IN-01
    severity: info
    disposition: open
    title: "Teardown errors bypass the four-part mount diagnostics; stepTeardown is unused"
  - id: IN-02
    severity: info
    disposition: open
    title: "vm-e2e FUSE/libfuse2 verdicts key on packed-file presence, not the ranked winner"
  - id: IN-03
    severity: info
    disposition: open
    title: "Signal-vs-child-exit select race can report errChildLost (exit 1) for a signal-only shutdown"
  - id: IN-04
    severity: info
    disposition: open
    title: "A second SIGINT/SIGTERM during the bounded teardown is swallowed (no force-quit)"
  - id: IN-05
    severity: info
    disposition: open
    title: "Arch is not part of candidate evaluation; the documented tie-break can select a non-runnable AppImage"
  - id: IN-06
    severity: info
    disposition: open
    title: "Every clean run logs \"appimage mount stopped\" twice"
  - id: IN-07
    severity: info
    disposition: open
    title: "The \"never signal an already-reaped helper\" claim has a PID-reuse window"
open: 7
total: 9
recorded: 2026-11-24T04:10:00.000Z
---

# Phase 10: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | fixed | 10-REVIEW-FIX.md |
| WR-01 | warning | fixed | 10-REVIEW-FIX.md |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |
| IN-06 | info | open | - |
| IN-07 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
