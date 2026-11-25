---
phase: 261125-r6o
plan: "01"
subsystem: turn/acpfront
tags: [acp, turn-machine, stall-watchdog, orphan-recovery, tdd, no-progress]

requires:
  - phase: "03"
    provides: the turn machine's one-observe-path completion, bypass-lane stop, and quiet-window watcher
  - phase: "04"
    provides: the front-wide turn slot, coded prompt errors, and the composed fake back leg
  - phase: "05"
    provides: the session/lifecycle teardown and observer release used by the resumed path

provides:
  - turn-scoped terminal/idle binding with the retired-turn memory and revision-ordered idle gate
  - a configurable no-progress watchdog (bypass-lane stop before the coded -32603 resolution)
  - a bounded one-stop/one-retry orphan recovery reachable from session/new and session/resume

affects: [prompt-completion, front-error-mapping, session-resume, turn-watchdog]

actuals:
  tokens: 32078     # chars/4 over the realized diff (git diff 68b360c..HEAD, 128310 chars)
  tasks: 3
  commits: 3
  plan_head_before: 68b360c5df9b49a41e99bd1d7030a6511488ca8c
  plan_head_after: a5d3f34b72d1162afc561d2af4d0d80b77cd878e

tech-stack:
  added: []
  patterns:
    - "Retired-turn binding: a resolved turnId persists across turns; a successor cannot be bound or resolved by the predecessor's late terminal"
    - "Revision-ordered idle gate: a state.updated idle patch completes a running turn only when both revisions are present and the idle revision is strictly greater"
    - "No-progress watchdog: a narrow progress scope resets the clock, the bypass stop fires before the resolution, and the goroutine is joined on every return path"
    - "Front-slot orphan recovery: a slot-free -32010 can only name an orphan; one stop, a bounded session-scoped clear wait, one retry, then a non-retryable coded error"

key-files:
  created:
    - internal/turn/turn_binding_test.go
    - internal/turn/stall_test.go
    - internal/turn/orphan_test.go
    - internal/acpfront/stall_test.go
    - internal/acpfront/orphan_test.go
  modified:
    - internal/turn/machine.go
    - internal/turn/machine_test.go
    - internal/turn/cancel_test.go
    - internal/acpfront/front.go
    - internal/acpfront/lifecycle.go
    - internal/acpfront/lifecycle_internal_test.go
    - internal/acpfront/cancel_test.go

key-decisions:
  - "The retired turnId persists across turns and is only cleared implicitly by binding a different id; an unbound successor ignores the predecessor's late terminal."
  - "A retired-id turn-start observed AFTER the successor's own running patch binds (backend turn-id reuse), while a stale start before it stays ignored; the late-terminal defense is unchanged."
  - "The watchdog is a running-turn guard only: a cancelling turn parks/exits the loop and keeps the quiet/cancel contract; no second stop is ever sent."
  - "Orphan recovery is an opt-in machine flag enabled unconditionally on every front-registered prompt machine through one shared constructor; direct machine embeddings keep the -32003 mapping."
  - "Clear evidence must be explicitly session-scoped; a terminal observed during the wait retires its non-empty turnId so a late duplicate cannot resolve the retried turn."
  - "The obsolete slot-free backend-refusal front lock was superseded (enshrining-test inversion) by the orphan contract; the send-refused-32010 corpus row/golden stay byte-untouched."

patterns-established:
  - "Red evidence is recorded per task before GREEN; CONFIRM-GREEN arms are labelled honestly instead of fabricating RED."
  - "Front prompt machines are built through exactly one constructor (NewSession and resumeSessionLocked), so per-machine config cannot diverge between fresh and resumed sessions."

requirements-completed: []

coverage:
  - id: D1
    description: "A new prompt cannot be completed by a prior turn's late terminal or by an idle patch not strictly newer than its own running patch."
    verification:
      - kind: unit
        ref: "internal/turn/turn_binding_test.go#TestLatePriorTurnSignalsDoNotResolveNewTurn"
        status: pass
      - kind: unit
        ref: "internal/turn/turn_binding_test.go#TestIdlePatchBindingRequiresLaterRevision"
        status: pass
      - kind: unit
        ref: "internal/turn/turn_binding_test.go#TestLatePriorTurnStartBeforeRunningPatchIsIgnored"
        status: pass
    human_judgment: false
  - id: D2
    description: "A silent running turn is stopped over the bypass lane, resolved StopError/SignalStall, and surfaced as a coded -32603; real progress resets the clock while telemetry does not."
    verification:
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogTripsOnSilentTurn"
        status: pass
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogProgressResetsOnDelta"
        status: pass
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogProgressResetsOnPartDelta"
        status: pass
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogTurnStartIsProgress"
        status: pass
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogActiveToolAllowance"
        status: pass
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogToolCompletionRestoresBaseBudget"
        status: pass
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogIgnoresRetryTelemetry"
        status: pass
      - kind: unit
        ref: "internal/turn/stall_test.go#TestStallWatchdogMessagesContentChangeIsProgress"
        status: pass
      - kind: integration
        ref: "internal/acpfront/stall_test.go#TestStalledTurnSurfacesCodedErrorAndReleasesSlot"
        status: pass
      - kind: integration
        ref: "internal/acpfront/lifecycle_internal_test.go#TestResumedSessionStallSeam"
        status: pass
    human_judgment: false
  - id: D3
    description: "An orphaned backend turn takes exactly one stop and one retried send; a second refusal is a stable non-retryable -32603; a resumed session carries the same flag and seam."
    verification:
      - kind: unit
        ref: "internal/turn/orphan_test.go#TestOrphanedBackendSendSelfHeals"
        status: pass
      - kind: unit
        ref: "internal/turn/orphan_test.go#TestOrphanRecoverySecondRejectionIsNotRetryable"
        status: pass
      - kind: unit
        ref: "internal/turn/orphan_test.go#TestOrphanClearWaitTimeoutStillRetriesOnce"
        status: pass
      - kind: unit
        ref: "internal/turn/orphan_test.go#TestOrphanClearWaitTerminalEvidenceRetiresOrphan"
        status: pass
      - kind: unit
        ref: "internal/turn/orphan_test.go#TestOrphanClearWaitIgnoresForeignSessionEvidence"
        status: pass
      - kind: integration
        ref: "internal/acpfront/orphan_test.go#TestOrphanRecoveryFrontMapping"
        status: pass
      - kind: integration
        ref: "internal/acpfront/lifecycle_internal_test.go#TestResumedSessionOrphanRecovery"
        status: pass
      - kind: unit
        ref: "internal/acpfront/cancel_test.go#TestSecondPromptRejected (front slot arm)"
        status: pass
      - kind: unit
        ref: "internal/turn/cancel_test.go#TestDoubleSendRejectedServerSide (default-off machine)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Final full gate: build, vet, full race suite, and strict lint are green; no new dependencies; logs stay structural."
    verification:
      - kind: other
        ref: "go build ./... && go vet ./... && go test -race -count=1 ./... && golangci-lint run"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-11-25
status: complete
---

# Quick Task 261125-r6o: Fix ACP Turn Stall Recovery — Summary

**Turn completion is now bound to the live backend turn, silent turns are bounded and coded, and a slot-free -32010 heals through one stop plus one retry instead of feeding an unbounded client retry loop.**

## Performance

- **Duration:** ~40 min (20:44–21:23 local commit span + summary/review)
- **Tasks:** 3 / 3 complete
- **Commits:** 3 code commits (docs committed separately by the orchestrator)
- **Files:** 12 changed (5 created, 7 modified)
- **Diff:** 2940 insertions / 133 deletions (chars/4 ≈ 32,078 tokens)
- **Final gate:** `go build ./...` OK, `go vet ./...` OK, `go test -race -count=1 ./...` all packages OK, `golangci-lint run` 0 issues

## Task Commits

| Task | Commit | Scope |
|------|--------|-------|
| 1 — Bind terminal/idle completion to the live turn | `4e9a427` | turn machine retired-turn binder, revision-ordered idle gate, structural logs, binding tests |
| 2 — No-progress stall watchdog + coded front error | `84a03a6` | watchdog/backend progress scope/join, Front stall seam + shared machine constructor, machine/front/resumed tests |
| 3 — Orphan recovery with one stop and one retry | `a5d3f34` | ErrOrphanedBackend path + clear wait + retirement, Front orphan flag/mapping, superseded obsolete lock, machine/front/resumed tests |

## TDD Evidence

### Task 1 (machine tier)

Genuine RED recorded at HEAD before GREEN:

- `TestLatePriorTurnSignalsDoNotResolveNewTurn` — RED: `second Run reason = "failed", want "completed"` (the late prior `turn.failed` bound the new turn on record-on-first-sight).
- `TestIdlePatchBindingRequiresLaterRevision` — RED for equal revision, absent revision, and cancelled/session_stopped while running: all resolved `idle-patch` at HEAD; expected `quiet-window`.
- `TestIdlePatchBindingRequiresLaterRevision/own_later_revision_resolves` — PASS at RED: CONFIRM-GREEN (the positive path already worked).
- `TestLatePriorTurnStartBeforeRunningPatchIsIgnored` — added during GREEN as the carve-out boundary lock; it was not part of the plan's RED set and is not claimed as RED evidence.

### Task 2 (machine + front tiers)

Machine RED recorded before the watchdog existed (fields/constants/accessor added as inert scaffolding only so the tests compile):

- Genuine RED (`Run reason = "completed", want "error"`, i.e. the quiet fallback resolved instead of a stall): `TestStallWatchdogTripsOnSilentTurn`, `TestStallWatchdogTurnStartIsProgress`, `TestStallWatchdogActiveToolAllowance`, `TestStallWatchdogToolCompletionRestoresBaseBudget`, `TestStallWatchdogIgnoresRetryTelemetry`.
- CONFIRM-GREEN at RED (no watchdog exists yet, so the terminal already won): `TestStallWatchdogProgressResetsOnDelta`, `TestStallWatchdogProgressResetsOnPartDelta`, `TestStallWatchdogMessagesContentChangeIsProgress`, `TestStallWatchdogDoesNotFireWhileCancelling`.

Front RED (recorded WITHOUT `WithTurnStallTimeout`; the machine default is 15 min, so the watchdog cannot fire before the production quiet floor):

- `TestStalledTurnSurfacesCodedErrorAndReleasesSlot` — RED at 7.51 s: `a stalled turn must surface a coded error, got a success response`.
- `TestResumedSessionStallSeam` — RED at 7.50 s: `a stalled resumed turn must surface a coded error, got a success answer`.

### Task 3 (machine + front + resumed tiers)

Genuine RED recorded at HEAD: all five machine tests failed with the direct numeric -32010 → `ErrTurnActive` mapping and no stop/retry on the wire:

- `TestOrphanedBackendSendSelfHeals`, `TestOrphanRecoverySecondRejectionIsNotRetryable`, `TestOrphanClearWaitTimeoutStillRetriesOnce`, `TestOrphanClearWaitTerminalEvidenceRetiresOrphan`, `TestOrphanClearWaitIgnoresForeignSessionEvidence` — all RED with `session/send: zcode: wire error -32010 ...: turn: a prompt is already running`.

Front-tier orphan tests were written after the machine GREEN and exercised the new contract directly (no intermediate RED was recorded for them; the machine-tier RED is the authorizing evidence).

## Residual Requirement Coverage (R1–R4)

- **R1 — outcome-discriminating progress sources:**
  - `session/event type:"part.delta"` with `field:"text"` — `TestStallWatchdogProgressResetsOnPartDelta` (ignore it → the base budget fires before the terminal; the terminal only wins with the reset).
  - A turn-start alone keeps a silent pre-budget turn alive past the base budget — `TestStallWatchdogTurnStartIsProgress` (turn-start at ~50 ms moves the 120 ms deadline to ~170 ms; asserts `At >= ack + 150 ms`). The model.streaming delta arm remains locked by `TestStallWatchdogProgressResetsOnDelta`.
- **R2 — cancelling carve-out:** `TestStallWatchdogDoesNotFireWhileCancelling` uses a 60 ms `StallTimeout`, cancels immediately after the ack, and holds the turn cancelling past the budget (the stop reply is delayed 300 ms; the quiet floor is ~400 ms). Assertions: `StopCancelled`/`SignalCancel`, resolution at/after ack+90 ms, exactly ONE `session/stop` on the wire, no `SignalStall`. The production-default cancel tests remain green unchanged.
- **R3 — watchdog parking/exit:** `stallLoop` recomputes the remaining budget on every wake/timer, and exits immediately when the turn is resolved or leaves `stateRunning` (cancelling included) instead of re-firing a non-positive remainder; `fireStallIfExpired` re-checks state and the remaining budget on both sides of the bypass write. `TestStallWatchdogTripsOnSilentTurn` additionally sleeps 2x the budget after `Run` returns and asserts the stop count is still one (the join proof).
- **R4 — front-tier RED shape:** the observed RED was the **successful end_turn** shape, not a context deadline: both front arms returned a success answer at ~7.50 s over the production quiet window with zero `session/stop` frames and no coded error (the stable messages projection carried the stored reply, so `reconcileNoDeltaReply` returned promptly inside the 10 s exchange bound). This is recorded verbatim above and not overclaimed.

## Deviations from Plan

### 1. [Rule 3 — blocking] Retired-id turn-start carve-out

- **Found during:** Task 1 GREEN, running the full front package.
- **Issue:** The plan's strict rule ("a turn-start equal to the retired turnId while unbound is ignored") made backends that REUSE a turnId across turns unresolvable by terminal. Legacy fixtures (`slashmodel_test.go`, `tool-lifecycle.json`, `test/tool-lifecycle` corpus) reuse static ids, so their second prompts fell to the 7.5 s quiet window and blew the 10 s composed exchange bound (`TestModelSlashSelectorByteEquality` and `TestToolLifecycle` failed with context deadline). Those fixtures/corpus files are outside the plan's writable set and corpus edits are forbidden.
- **Fix:** A retired-id turn-start binds only when this turn's own running patch has already armed the turn (proof the successor's send was accepted). The realistic stale shape — a prior turn's start reaching the wire before the new send — is still ignored, locked by `TestLatePriorTurnStartBeforeRunningPatchIsIgnored`; the late-TERMINAL defense (the must-have truth) is unchanged.
- **Files:** `internal/turn/machine.go`, `internal/turn/turn_binding_test.go`.

### 2. [Sequencing] Inert symbol scaffolding before RED

- **Found during:** Tasks 2 and 3 RED.
- **Issue:** The new tests reference `SignalStall`, `StallTimeout`, `StallToolTimeout`/`Resolution()`, and `ErrOrphanedBackend`/`OrphanRecovery`/`OrphanClearTimeout`, which the plan adds in the GREEN steps; without them the RED run is a compile failure, not behavioral evidence.
- **Fix:** Declared the constants/fields/accessor as inert scaffolding first, ran the behavioral RED, then implemented the behavior. No behavior changed in the scaffold step.

### 3. [Lint] `machine_test.go` constant refactor staged with Task 3

- **Found during:** Task 3 lint.
- **Issue:** The new orphan test's third `"status"` literal crossed the package-wide goconst threshold; the fix is a one-line `wireKeyStatus` constant use in `statePatchRev`.
- **Fix:** Included with the Task 3 commit (the file is in the plan's overall `files_modified`).

### 4. [Residual] R1 turn-start test added beyond the plan's test list

- Added `TestStallWatchdogTurnStartIsProgress` in `internal/turn/stall_test.go` per the orchestrator's residual requirement; it is a progress-scope discrimination test, not a behavior change.

## Auth Gates

None — no authentication step was required for this task.

## Known Stubs

None. No hardcoded empty values, placeholder text, or unwired data sources were introduced; all new behavior is exercised by the tests listed above.

## Threat Flags

None beyond the plan's threat register: the new logs carry only session id, turnId, status/reason/revision, signal, budget, and structural errors — no prompt text, deltas, or tool input/output.

## Self-Check: PASSED

- Created files exist: `internal/turn/turn_binding_test.go`, `internal/turn/stall_test.go`, `internal/turn/orphan_test.go`, `internal/acpfront/stall_test.go`, `internal/acpfront/orphan_test.go`.
- Commits exist on `fix/acp-turn-stall-recovery`: `4e9a427`, `84a03a6`, `a5d3f34`.
- Task gates and the final full gate (`build`, `vet`, `go test -race -count=1 ./...`, `golangci-lint run`) all pass; `go.mod`/`go.sum`, corpus JSON/goldens, and the ACP wire vocabulary are untouched.
