---
phase: 261125-s7k
plan: 01
subsystem: acp
tags: [acp, cancel, steering, orca, turn-machine, tdd]
requires: []
provides:
  - Deterministic session/cancel answer (cancelled stopReason, bounded)
  - Failed-terminal-while-cancelling resolves cancelled, not failed
  - Cancel grace seam on turn machines and the front (session/new and session/resume)
  - Composed steer regression over the ACP wire
affects: [turn-machine, cancel-path, acp-front, session-new, session-resume, protocol-docs]
tech-stack:
  added: []
  patterns:
    - Signal-first resolution with a bounded backstop for a client-demanded outcome
    - One shared front machine constructor for new and resumed sessions
key-files:
  created:
    - internal/turn/cancel_outcome_test.go
    - internal/acpfront/steer_test.go
    - internal/acpfront/testdata/steer/cancel-turn-failed.json
    - internal/acpfront/testdata/steer/resume-silent-stop.json
    - .planning/tdd/261125-s7k-task1-red.txt
    - .planning/tdd/261125-s7k-task2-red.txt
    - .planning/tdd/261125-s7k-task1-green.txt
    - .planning/tdd/261125-s7k-task2-green.txt
    - .planning/tdd/261125-s7k-gates.txt
  modified:
    - internal/turn/machine.go
    - internal/acpfront/front.go
    - internal/acpfront/lifecycle.go
    - docs/protocol/acp-client-dialect.md
key-decisions:
  - A failed terminal while the turn is cancelling is the stop's observed outcome - resolve cancelled, never failed.
  - A completed terminal keeps completed even when a cancel raced it (no false cancellation).
  - The grace is a bounded backstop only; terminal, cancelled patch and quiet settle stay primary.
  - Default grace 2 s (6x the live ~300 ms stop-to-terminal latency); exposed only as internal seams.
  - Out of scope by design: orphaned-backend -32010 recovery, the no-progress watchdog and terminal/idle turn binding (planned separately as 261125-r6o).
requirements-completed: []
coverage:
  - id: D1
    description: A cancelled turn answers cancelled when the backend reports the stop as turn.failed (both carriers).
    verification:
      - kind: unit
        ref: internal/turn/cancel_outcome_test.go#TestCancelResolvesCancelledOnFailedTerminal
        status: pass
    human_judgment: false
  - id: D2
    description: A completed terminal racing the cancel keeps end_turn; an idle turn is never resolved by a stale grace.
    verification:
      - kind: unit
        ref: internal/turn/cancel_outcome_test.go#TestCancelRacingCompletedStaysCompleted
        status: pass
      - kind: unit
        ref: internal/turn/cancel_outcome_test.go#TestCancelGraceDoesNotResolveIdleTurn
        status: pass
    human_judgment: false
  - id: D3
    description: A cancelling turn with no backend signal answers inside the configured grace, exactly once.
    verification:
      - kind: unit
        ref: internal/turn/cancel_outcome_test.go#TestCancelGraceResolvesWithoutSignal
        status: pass
      - kind: unit
        ref: internal/turn/cancel_outcome_test.go#TestCancelGraceNeverDoubleResolves
        status: pass
    human_judgment: false
  - id: D4
    description: The steer sequence works end-to-end on session/new with the real desktop's turn.failed cancel outcome.
    verification:
      - kind: integration
        ref: internal/acpfront/steer_test.go#TestSteerAfterCancelStartsNewTurn
        status: pass
    human_judgment: false
  - id: D5
    description: The grace reaches session/resume machines through the shared constructor.
    verification:
      - kind: integration
        ref: internal/acpfront/steer_test.go#TestSteerAfterCancelOnResumedSession
        status: pass
    human_judgment: false
status: complete
---

# Quick Task 261125-s7k: Answer session/cancel so a steer can start

## What the client does

Orca's steer path (1.4.274 `src/main/acp/turns/acp-structured-turns.ts`) is not a
mid-turn prompt. `dispatch` pushes the new send onto `steers`, `cancelForSteer`
sends the `session/cancel` notification, and the queued send starts only from
`finish()` — which runs when the ACTIVE `session/prompt` request settles. The
source comment states it: "Steers wait for the active prompt to answer
cancellation." A wrapper that never answers the cancelled prompt leaves exactly
the field symptom: the current answer is interrupted, the steer never starts.

Field evidence: in Orca's journal (`agent-session-domain.db`, session
`8c4ebde4-a310-4627-8da2-a791dc87a939`) the last user send is recorded at
17:36:06Z; no later submission row exists for the steer — it never reached
dispatch, because the active prompt never answered.

## Root cause in the wrapper

- `Machine.ObserveEnvelope` classified `turn.failed` as `StopFailed`
  unconditionally. The live desktop reports the stop with `turn.failed`
  (2026-11-25 field log 17:20:28.882 stop → 17:20:29.181 turn.failed), so the
  front answered the cancelled prompt with `-32603`, not `stopReason:
  "cancelled"`.
- Resolution was otherwise signal-only: the cancelled state patch (which the
  real desktop does not send — the `state.updated` channel carried only
  `prompt_started`) or the quiet-window settle, gated behind `3 × QuietMs`
  (7.5 s) from the send ack plus `QuietMs` (2.5 s) of no watched-session
  activity. A stop arriving early in a turn, or any activity after it, held the
  prompt open past the client's steer window — indefinitely if activity never
  stopped.

## The fix

1. `internal/turn/machine.go`
   - A matched terminal classified `StopFailed` while the machine is
     `stateCancelling` resolves `StopCancelled` / `SignalCancel`. A
     `StopCompleted` terminal is untouched: a turn that really finished must
     not be relabeled as cancelled.
   - `Cancel` arms a cancel grace (`DefaultCancelGrace = 2 s`) under the same
     lock section as the running→cancelling transition; the callback resolves
     `StopCancelled` / `SignalCancel` only while `stateCancelling` and
     unresolved. `resolveLocked` stops the timer; `finishTurn` stops and clears
     it, so no timer can resolve a later turn.
   - `NewMachine` takes variadic `MachineOption`s; `WithCancelGrace` overrides
     the default. Zero keeps the production value.
2. `internal/acpfront`
   - `WithCancelGrace` option and `cancelGrace` field; a single
     `newSessionMachine` constructor now builds the machine for `NewSession`
     and `resumeSessionLocked`, so the seam cannot diverge between the two.
3. `docs/protocol/acp-client-dialect.md` — the `session/cancel` bullet states
   the deterministic answer (cancelled success, failed-terminal mapping,
   bounded grace) and why it dispatches the steer queue.

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED | `dcf69c9` (same commit; captured before the production edits) | PASS — `.planning/tdd/261125-s7k-task1-red.txt` (turn.failed answered `failed`), `.planning/tdd/261125-s7k-task2-red.txt` (resumed session never answered), grace RED in the session log (prompt open past the 3 s bound) |
| GREEN | `dcf69c9` | PASS — `.planning/tdd/261125-s7k-task1-green.txt`, `.planning/tdd/261125-s7k-task2-green.txt` |

The RED evidence files were produced before the production edits in this
worktree; the fix, tests and docs landed as one atomic commit on the branch
(`dcf69c9`), with the RED outputs preserved verbatim.

## Verification

Full gates in this worktree (`.planning/tdd/261125-s7k-gates.txt`):

- `go build ./...` — PASS
- `go vet ./...` — PASS
- `golangci-lint run` — 0 issues
- `go test -race -count=1 ./...` — all 11 packages ok
- `bash experiments/check-docs.sh` — PASS
- `node --test experiments/` — 24 pass, 1 skipped, 0 fail
- `go mod tidy` — no drift

Behavioural locks beyond the new tests: the existing `cancel-mid-turn`,
`complete-then-cancel`, `TestCancelMidTurnAnswersOnce`,
`TestCancelRacingTerminalResolvesOnce`, `TestSecondPromptRejected` (both guard
layers) and the completion corpus stayed green.

## Scope Discipline

- Wire vocabulary, ACP error shapes, the busy contract, the single-active-turn
  invariant, the front-owned turn context, the bypass-lane stop and the
  flush-before-stopReason release order are unchanged.
- No corpus scenario or golden was edited; the two new fixtures live under
  `internal/acpfront/testdata/steer/` because they are front-tier wire
  sequences, not replayable back-leg corpus rows.
- No new dependency; `go.mod`/`go.sum` unchanged.
- Deliberately NOT in this task (planned separately as quick task
  `261125-r6o`): slot-free backend `-32010` orphan recovery, the no-progress
  watchdog, and terminal/idle turn binding. A false local completion followed
  by a backend `-32010` retry storm is that task's territory, not this one's.

## Known Stubs

None.

## Issues Encountered

- The initial fixtures were placed in the fake corpus directory; the corpus
  gate (`TestGolden`) requires a replay driver and golden for every scenario
  there. They were moved to `internal/acpfront/testdata/steer/` as front-tier
  fixtures (the strict loader still validates them).
- `goconst` tipped over the 3-occurrence threshold across the turn test package
  when the new file repeated `status`/`eventSeq`/`version`; the new file now
  carries its own key constants.

## User Setup Required

None.

## Next Readiness

The cancel answer is now bounded and ACP-correct for both session/new and
session/resume, so Orca's steer queue always receives the answer that
dispatches the steered prompt. The separate stall-recovery quick task
(`.planning/quick/261125-r6o-fix-acp-turn-stall-recovery/`, worktree
`261125-turn-stall-recovery`) remains unexecuted and covers the adjacent
failure modes.

## Self-Check: PASSED

- RED and GREEN evidence files exist and show the intended before/after.
- The branch commit touches exactly the planned files plus the GSD evidence.
- Whole-tree build, vet, lint, race tests, docs and experiments gates pass.
- No protected path, corpus golden or dependency file was modified.
