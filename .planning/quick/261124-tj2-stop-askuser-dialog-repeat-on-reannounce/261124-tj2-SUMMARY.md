---
phase: 261124-tj2
plan: 01
subsystem: interaction-broker / ask-the-user bridge
tags: [interaction, requestUserInput, re-announce, dedupe, ttl, elicitation, acp, orca, golang]
requires:
  - phase: 05
    provides: re-announce dedupe table keyed on (method, params.requestId)
  - phase: 09
    provides: requestUserInput → one ACP form elicitation bridge with Deadline 0
provides:
  - A joined re-announce refreshes the pending dedupe entry's TTL, so a slow human answer keeps ONE dialog
  - Fake-clock regression tests for the refresh, the backstop expiry and the un-refreshed resolved tombstone
  - Dialect-doc note stating the one-dialog guarantee and its implementation
affects: [ask-the-user dialog UX in Orca, internal/interact, docs/protocol/app-server-dialect.md]
actuals:
  tasks: 2
  commits: 2
tech-stack:
  added: []
  patterns:
    - "TTL backstop + touch-on-join: the entry measures server silence, not human answer latency"
    - "Dedupe contract tested through the injected now clock, never by sleeping"
key-files:
  created:
    - internal/interact/dedupe_internal_test.go
  modified:
    - internal/interact/dedupe.go
    - internal/interact/responder.go
    - docs/protocol/app-server-dialect.md
key-decisions:
  - "Refresh-on-join instead of an infinite TTL for Deadline 0: the entry still evicts ttl after the last re-announce, preserving the wedged-primary backstop"
  - "Resolved tombstones are never refreshed — extending them would delay the 2 s resolution-grace sweep indefinitely"
  - "The regression test asserts the field-observed 20 s cadence (dedupeTTL(0) = 15 s router default + 5 s grace) and then drives claims across that boundary on a fake clock"
metrics:
  duration: 25min
  completed: 2026-11-24
status: complete
commits: 2
plan_head_before: 0b03caca5b93a1d394cae7513ea4cc158d6cc436
plan_head_after: c9b27e043946884695e8b95fa75350c6593d9a7b
---

# Quick Task 261124-tj2 Summary

**One `interaction/requestUserInput` ask now renders exactly ONE ACP form elicitation for an arbitrarily slow human answer: every joined re-announce refreshes the dedupe entry, so the entry expires only after the server stops re-announcing.**

## Root cause

The app-server re-announces an unanswered ask with the same `params.requestId`
(dual-channel broker; `docs/protocol/app-server-dialect.md:83`). `Responder.Serve`
dedupes those by `(method, params.requestId)` in `internal/interact/dedupe.go`,
but a join only returned early — it never touched the entry's `expires`.

The `interaction/requestUserInput` policy row deliberately has `Deadline: 0`
(“a human answering a dialog takes minutes”, `internal/interact/responder.go`),
yet `dedupeTTL(0)` fell back to `zcode.RouterDefaultDeadline` (15 s) plus
`dedupeGrace` (5 s) = **20 s**. After 20 s the next re-announce was treated as a
fresh primary and ran `handleUserInput` again → another `elicitation/create` →
another dialog, every 20 s until the human answered.

Field evidence (Orca journal `agent-session-domain.db`, `journal_items`,
session `19467492-c156-4cd1-b922-17cef726fd11`, 2026-11-24): the same
“Complete the milestone?” ask was recorded as 15 separate `elicitation/create`
requests with contiguous ids `…:688`–`…:702`, created at 04:27:30, 04:27:55,
then exactly every 20 s to 04:32:15; a second ask repeated the pattern with
ids `…:29`–`…:43` (01:18:55 → 01:23:40). The intervals match the TTL, not the
server’s re-announce schedule — conclusive.

## Task Commits

1. **Plan** — `c62636f` (docs): `docs(quick-261124-tj2): stop the ask-user dialog repeating on re-announce`
2. **Task 1: refresh a pending dedupe entry on join** — `81527f0` (fix): `fix(261124-tj2): refresh the re-announce dedupe entry while the ask is pending`
3. **Task 2: document the guarantee** — `c9b27e0` (docs): `docs(261124-tj2): document the one-dialog ask guarantee`

## RED → GREEN Evidence

RED on the unfixed code (`go test -race -count=1 -run TestDedupe ./internal/interact`):

```
--- FAIL: TestDedupeJoinRefreshesPendingTTL (0.00s)
    dedupe_internal_test.go:49: claim 2 at 04:28:08: joined = false, want true
```

GREEN after the fix:

```
go test -race -count=1 ./internal/interact        -> ok 2.394s
golangci-lint run ./internal/interact/...         -> 0 issues.
bash experiments/check-docs.sh                    -> exit 0
mise run ci                                       -> build+vet+lint+test+experiments all green
                                                     (interact 2.403s, acpfront 101.943s, all packages ok)
```

## Files Created/Modified

- `internal/interact/dedupe.go` — `claim` now refreshes `expires` for a joined
  **pending** entry and leaves a resolved tombstone untouched; `dedupeEntry`
  and `dedupeTTL` comments explain that the TTL measures server silence and the
  refresh is what keeps an unbounded human dialog alive.
- `internal/interact/dedupe_internal_test.go` (new) — two fake-clock tests:
  join-refreshes-pending-TTL (including the post-TTL join that regressed and the
  eventual backstop expiry) and tombstone-is-not-refreshed (join inside the
  resolution grace vs a fresh primary past it). No sleeps.
- `internal/interact/responder.go` — comment-only: the Serve join arm names the
  refresh, and the requestUserInput policy row ties `Deadline: 0` to it.
- `docs/protocol/app-server-dialect.md` — requestUserInput row gains the
  re-announce-cadence note: joined re-announces refresh the entry, so a slow
  human answer still renders ONE form elicitation; the entry still expires once
  re-announces stop.

## Deviations from Plan

- The regression test file was named `dedupe_internal_test.go` (not
  `dedupe_test.go`): the repo’s `testpackage` linter rejects an in-package test
  file without the `_internal_test.go` suffix. The plan file was updated before
  its commit; behavior and assertions are unchanged.
- Task 2’s plan action named only the doc row and the policy comment; the Serve
  join-arm comment was updated alongside them because it describes the same
  contract. Same commit, comment-only.

## Verification

- One ask → one elicitation: the dedupe entry survives any number of joins while
  the server keeps re-announcing.
- The backstop still fires: 20 s after the last join a new claim is a fresh
  primary; a resolved tombstone still sweeps after `resolutionGrace`.
- No wire shape, policy deadline, parking semantics or other method row changed.
- Full `mise run ci` green; `check-docs` exit 0.
