# Phase 7 — Review Disposition Ledger

> One row per finding from [07-REVIEW.md](./07-REVIEW.md) (0 Critical / 3 Warning / 8 Info).
> Default disposition: `open`. Advisory artifact — never blocks.
> Written by the execute-phase code_review_gate, 2026-11-22.

| Finding | Severity | Disposition | Rationale / Route |
|---------|----------|-------------|-------------------|
| WR-01 resume discards the backend reply's settings snapshot | warning | **resolved** `6ffad04`/`5236e96` | `recordSettings(sid, res)` beside `recordCwd` on resume; `dropSettings` in the subscribe-failure path; TestResumeRecordsConfirmedSettings locks picker + slash arms on resumed sessions |
| WR-02 /model intercept bypasses CORE-05 mid-turn; refusal teardown kills the running turn | warning | **resolved** `0ccd660`/`5e735f9` | session-scoped `slashTurnBusy` gate under turnMu rejects with turnActiveMessage before any backend call; TestModelSlashMidTurnRejectedWhenTurnActive + TestPromptModelSlashBusyScope lock same/cross-session scoping |
| WR-03 --api-key rides argv world-readable | warning | **resolved** `7c15e09` | README documents the ps//proc exposure and steers to ZCODE_ACP_API_KEY; main() emits a stderr warning when the flag form is used (flag kept for compat) |
| IN-a vm-e2e.sh dead `ambiguous)` arm misreports multi-match | info | **open** → post-v1 | cosmetic classification nuance; the four primary sentinels are byte-exact |
| IN-b `--stage nokeyring-arm` vacuous PASS(0 checks) | info | **open** → post-v1 | publicly-accepted diagnostic arm; add a check when a real keyring-less VM run exercises it |
| IN-c per-read `read -t` bound not a total leg bound | info | **open** → post-v1 | timeout hardening nuance |
| IN-d --help truncates the output-contract section | info | **open** → post-v1 | doc polish |
| IN-e shutdown `select` masks coincident child crash as exit 0 | info | **open** → post-v1 | exit-code nuance; the child crash itself is loud in logs |
| IN-f unknown-flag tolerance coupled to the stdlib flag error string | info | **open** → post-v1 | stdlib coupling accepted (zero-dep discipline) |
| IN-g DeleteSession silent-success race window | info | **open** → post-v1 | narrow window; the 07-01 teardown semantics bound the damage |
| IN-h resume reply session id trusted without equality check | info | **open** → post-v1 | adapter discards the returned id; equality check is hardening polish |

**Security pass:** no Critical; the omp dispatch, SESS-01 teardown ordering, closingMu discipline, wireError tables, keyring classification, and all doc claims verified clean.

**Summary:** 3 resolved in-phase (WR-01/02/03), 8 open → post-v1 advisories, 0 critical.
