---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 05
subsystem: testing
tags: [turn-machine, state-machine, poll-watcher, quiet-window, exactly-once, completion-signals, carrier-tolerance, tdd, d-02, bridge-04]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-01)
    provides: the zcode.Conn (Call/Stop/notification hook), the fake scripting core (Serve + programmatic Scenario), and the happy-turn contract this plan's machine consumes
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-03)
    provides: the corpus completion-variant shapes (completion-idle-patch, completion-quiet-window) and the stable session/messages settling surface the PollWatcher polls
  - phase: 01-research-protocol-documentation
    provides: docs/protocol/app-server-dialect.md §5 terminal signals (turn-start carrier table, quiet-window gates) and the probe harness blueprint (app-server-probe.mjs:555-601)
provides:
  - internal/turn package: StopReason constants, TurnWatcher interface (subscribe-based watcher a reserved, unimplemented strategy slot), PollWatcher with the probe-ported quiet-window gates (QuietMs 2500 / FloorMultiple 3 / PollIntervalMs 500 defaults, injectable PollSource seam, synchronous stop)
  - Machine: NewMachine bound to one Conn + one sessionId (D-04), Run lifting fire-and-forget session/send into a synchronous turn, ErrTurnActive (-32010 wrapper-side guard), Wait idempotent second wait, AckAt floor-clock origin, OnResolve exactly-once hook
  - The D-02 ordered signal union resolved exactly once per turn: turnId-matched terminal events across the three known carriers, idle patch (status leaving running; running patch never resolves), quiet-window settlement strictly after the send-latency floor — with the honest resolving-signal record (Resolution{Reason, Signal, At}) and no synthesized completion on absence of evidence
  - RED evidence records for both TDD stages (03-05-red-turn-machine.json, 03-05-red-completion-variants.json), both RED_EVIDENCE_OK before their feat commits
affects: [03-08 turn cancel + concurrency guard + corpus completion proofs, 03-09 E2E + phase gate, phase-4 ACP front (session/prompt calls Machine.Run per prompt)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 13100   # 52250 chars over this plan's diff (5 files, 1467 insertions / 0 deletions)
  tasks: 3
  commits: 4     # MEASURED from the plan ledger (3e817f0..HEAD): 2 RED + 2 feat

commits: 4
plan_head_before: 3e817f010d6feef6156b8dc5acad63343d5d8e3e
plan_head_after: 1e059d2755c8d097804f0242887d1a935f6cbec4

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: tidy-diff green, go.mod untouched)
  patterns:
    - "Exactly-once via a per-turn resolved flag under the machine mutex (sync.Once semantics — sync.Once cannot be reset between turns); the lifetime resolution counter plus the OnResolve hook make it observable"
    - "One observe path: ObserveEnvelope is conn.OnNotification's receiver; watcher deliveries funnel into the same path — terminal/idle/quiet all classify or resolve in one place (T-3-06 mitigation)"
    - "Opaque message polling: the session/messages result wraps as ONE session-scoped observation envelope per poll; activity = raw payload change between polls (presence/time heuristics only, no delta cursors — OQ2 resolution)"
    - "Record-on-first-sight turnId: under the one-turn-at-a-time -32010 guard the first turn-scoped signal defines the turn id; later terminals must match it — foreign-turnId envelopes never resolve"
    - "No synthesized completion: ctx cancellation and transport failures return errors, never StopReasons"

key-files:
  created:
    - internal/turn/machine.go
    - internal/turn/watcher.go
    - internal/turn/machine_test.go
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-05-red-turn-machine.json
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-05-red-completion-variants.json
  modified: []

key-decisions:
  - "sync.Once semantics implemented as a per-turn resolved flag under the machine mutex — sync.Once cannot be reset for the next turn; the lifetime resolution counter is the exactly-once proof the tests observe through the OnResolve hook"
  - "PollWatcher polls session/messages opaquely (OQ2 resolution): one session-scoped observation envelope per poll, quiet activity = raw payload change between polls — presence/time heuristics only, content never interpreted, no delta cursors"
  - "The quiet-window floor clock starts at Watch (the machine calls Watch immediately after the send ack returns), so the settle gate is strictly gated past ack + FloorMultiple x QuietMs; the test asserts Resolution.At >= AckAt + floor"
  - "Task 2's GREEN implements the terminal-only classification path so Task 3's mandated variant RED fails genuinely — the idle patch, the quiet-window consumption, and their exactly-once proofs land with Task 3's GREEN (the plan's own Task 3 RED requirement forces this sequencing of the shared action text)"
  - "Carrier classification maps session/event to params.type and v4/telemetry + computer-use to params.kind (dialect §5 signal tables); carrier method constants stay local to the turn package instead of growing the codec surface"
  - "The idle-patch latch requires a running patch first (leaving running implies running was seen) and ignores foreign-session patches; the running patch itself never resolves"

patterns-established:
  - "Turn machine skeleton for 03-08: states idle -> running -> cancelling -> done declared now; cancel transitions and the idempotent Cancel build on guardTurnLocked/resolveLocked without touching the observe path"
  - "resolutionSpy pattern: a synchronized OnResolve hook recorder — the exactly-once counter that survives concurrent observing goroutines (the drained-channel counter it replaced was a harness bug)"
  - "Fake-driven machine tests: programmatic Scenario scripts (D-01) with fixed synthetic ids and injectable millisecond knobs (QuietMs 40 / FloorMultiple 2 / PollIntervalMs 10) — no test waits on wall-clock defaults"

requirements-completed: [BRIDGE-04]  # COPIED VERBATIM from 03-05-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-04: 03-05/03-08/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "TurnWatcher seam + PollWatcher: session/messages polling through the Conn with the probe-ported quiet-window gates (send-latency floor = FloorMultiple x QuietMs from the send ack, then QuietMs settle), injectable PollSource, synchronous stop with no delivery after it returns; subscribe-based watcher reserved, not implemented"
    requirement: BRIDGE-04
    verification:
      - kind: unit
        ref: "internal/turn/machine_test.go#TestTurnWatcherDeliversUntilStopped"
        status: pass
    human_judgment: false
  - id: D2
    description: "Machine exactly-once completion: Run lifts fire-and-forget session/send into a synchronous turn — one StopCompleted after the backend's terminal signal, honest terminal-event resolving-signal record, idempotent second Wait, resolution hook fired exactly once; -32010 wrapper-side guard (ErrTurnActive)"
    requirement: BRIDGE-04
    verification:
      - kind: unit
        ref: "internal/turn/machine_test.go#TestHappyTurnCompletedExactlyOnce"
        status: pass
    human_judgment: false
  - id: D3
    description: "D-02 ordered completion variants each resolve exactly once with the correct StopReason + resolving-signal record: terminal event, idle patch (running patch itself never resolves), quiet window (strictly after the send-latency floor post-ack) — no scenario resolves on absence of evidence"
    requirement: BRIDGE-04
    verification:
      - kind: unit
        ref: "internal/turn/machine_test.go#TestCompletionVariants (terminal-event, idle-patch, quiet-window subtests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Carrier-tolerant terminal classification (T-3-06 mitigation, RESEARCH Open Question 1 resolution): the same turnId-matched terminal classification fires across session/event (type), v4/telemetry/event (kind), and computer-use/operation-event (kind) carriers; a foreign-turnId terminal never resolves the turn"
    requirement: BRIDGE-04
    verification:
      - kind: unit
        ref: "internal/turn/machine_test.go#TestTerminalCarrierTolerance (session-event, v4-telemetry, computer-use subtests)"
        status: pass
    human_judgment: false

# Metrics
duration: 26min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 05: Turn Machine (TurnWatcher + Machine Exactly-Once Completion) Summary

**The turn state machine is live: Machine.Run lifts one fire-and-forget session/send into a synchronous turn resolved exactly once by the D-02 ordered signal union — turnId-matched terminal events across three carriers, the idle patch, and the probe-ported quiet-window gates — with the honest resolving-signal record and zero new dependencies**

## Performance

- **Duration:** 26 min
- **Started:** 2026-11-19T19:05:19Z
- **Completed:** 2026-11-19T19:31:28Z
- **Tasks:** 3
- **Files:** 5 (3 Go source/test files, 2 RED evidence records)

## Accomplishments

- Task 1 (TDD RED): both contract targets failed on their planned assertions against compiling stubs — the happy-turn exactly-once Run contract and the PollWatcher delivery/stop contract (RED_EVIDENCE_OK, go-test adapter, evidence `03-05-red-turn-machine.json`)
- Task 2 (GREEN): PollWatcher implemented — session/messages polling through the Conn with the result wrapped opaquely as one session-scoped observation envelope (presence/time evidence only, OQ2 resolution), the probe-harness quiet-window gates ported verbatim (send-latency floor = FloorMultiple x QuietMs measured from the send ack, then the QuietMs settle gate; defaults 2500/3/500, injectable), synchronous stop that joins the loop (no delivery after stop returns); Machine implemented — the -32010 wrapper-side guard (ErrTurnActive), session/send via Conn.Call with the ack starting the floor clock, watcher lifecycle joined by Run, ONE observe path (ObserveEnvelope) with turnId-matched terminal classification, exactly-once resolution (per-turn resolved flag + closed wait channel + OnResolve hook), no synthesized StopReason on ctx/transport failure
- Task 3 (TDD RED then GREEN): the idle-patch and quiet-window variants failed the planned resolve-within-window gate against the terminal-only Task 2 machine (RED_EVIDENCE_OK, evidence `03-05-red-completion-variants.json`); GREEN extended the single observe path with the idle-patch latch (status leaving running; the running patch itself only latches and never resolves; foreign-session patches ignored) and the quiet-window consumption (strictly after the send-latency floor — asserted `Resolution.At >= AckAt + floor`) — all three variants resolve exactly once with the correct reason + signal record
- Carrier-tolerance locked (RESEARCH Open Question 1 resolution): the same terminal classification fires across session/event (type turn.completed), v4/telemetry/event (kind turn.completed), and computer-use/operation-event (kind turn-completed), each with a foreign-turnId decoy that must not resolve the turn
- Full gates green: `go test -race -count=1 ./...` all packages; `mise run ci` (build, vet, lint 0 issues, test, test-experiments); `mise run tidy-diff` clean (go.mod untouched); turn package re-run at `-count=3` under `-race` without flake

## TDD Record

- **Task 1 RED:** `go test -race -count=1 -run 'TestHappyTurn|TestTurnWatcher' ./internal/turn/` — both targets failed at their first contract gate on the declared errNotImplemented stubs; record classified **RED_EVIDENCE_OK** (target_test_failed) before the GREEN commit
- **Task 2 GREEN:** both Task 1 targets green under `-race`; the terminal-only observe path resolves only turnId-matched terminal kinds (no resolution on absence of evidence)
- **Task 3 RED:** `go test -race -count=1 -run 'TestCompletionVariants|TestTerminalCarrierTolerance' ./internal/turn/` — idle-patch and quiet-window failed the planned resolve-within-window gate (the Task 2 machine consumes only terminal signals); terminal-event and the three carrier subtests passed as locks; record classified **RED_EVIDENCE_OK** before the GREEN commit
- **Task 3 GREEN:** all variants green with the exactly-once spy proving single resolution per variant; full suite and `mise run ci` green

## Task Commits

Each task was committed atomically (TDD tasks carry RED + GREEN commits):

1. **Task 1 RED: failing turn machine + watcher contract tests (evidence RED_EVIDENCE_OK)** - `bfc4697` (test)
2. **Task 2 GREEN: poll watcher + exactly-once turn completion** - `97f7d47` (feat)
3. **Task 3 RED: failing completion-variant tests (evidence RED_EVIDENCE_OK)** - `d517805` (test)
4. **Task 3 GREEN: ordered completion variants + carrier-tolerant terminal classification** - `1e059d2` (feat)

## Files Created/Modified

- `internal/turn/machine.go` - StopReason/SignalKind/Resolution surface, carrier kind literals, Machine (NewMachine, Run, Wait, AckAt, ObserveEnvelope, ErrTurnActive guard), terminal + idle-patch classification, exactly-once resolveLocked
- `internal/turn/watcher.go` - Observation, TurnWatcher interface, PollSource seam, PollConfig/PollWatcher with the quiet-window gates and the conn-backed opaque poll source
- `internal/turn/machine_test.go` - happy-turn exactly-once, watcher delivery/stop, three completion variants, carrier tolerance; programmatic fake scenarios with fixed synthetic ids and millisecond knobs
- `.planning/.../evidence/03-05-red-turn-machine.json`, `03-05-red-completion-variants.json` - persisted RED evidence, both RED_EVIDENCE_OK

## Decisions Made

- Exactly-once via a per-turn resolved flag under the machine mutex (sync.Once semantics; sync.Once itself cannot reset between turns); the lifetime resolution counter is the proof the OnResolve hook observes
- PollWatcher treats session/messages opaquely — one observation envelope per poll, activity = raw payload change (presence/time heuristics only, no delta cursors; OQ2 resolution adopted by Task 2)
- The quiet-window floor is measured from Watch start, and the machine calls Watch immediately after the send ack, so the settle gate is strictly past ack + floor; the test asserts the resolution timestamp against AckAt + floor
- Task 2's GREEN ships the terminal-only classification path so Task 3's mandated variant RED fails genuinely; idle/quiet/carrier-resolution land in Task 3's GREEN (the plan's own Task 3 RED requirement forces this sequencing of the shared action text)
- Carrier kind mapping lives in the turn package (session/event -> params.type; v4/telemetry + computer-use -> params.kind) — codec surface untouched
- The subscribe-based watcher remains a reserved strategy slot, NOT implemented (CONTEXT Deferred Ideas)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Resolution counter counted a drained channel**
- **Found during:** Task 2 GREEN (first full turn-test run)
- **Issue:** the harness delivered the single resolution into a buffered channel and then asserted on `len(channel)` — the signal-check receive had already drained it, so the exactly-once count read 0 with the machine behaving correctly
- **Fix:** replaced the channel-length counter with a synchronized resolutionSpy (mutex-guarded record of every OnResolve delivery); the assertion got stronger — the count is checked before the first Wait and again after the idempotent second Wait
- **Files modified:** internal/turn/machine_test.go
- **Verification:** TestHappyTurnCompletedExactlyOnce green under -race; the spy pattern re-used by the variant and carrier tests
- **Committed in:** 97f7d47 (Task 2 commit)

### Plan-Text Adaptations (not defects)

**1. Task 1's `files_modified` names only machine_test.go, but the action text requires compiling stubs**
- The stub declarations for the declared surface (machine.go, watcher.go) necessarily shipped with Task 1's RED commit `bfc4697` — a test cannot compile against types that exist nowhere. Both files are the plan frontmatter's own artifacts.
- **Committed in:** bfc4697

**2. Task 2's action text enumerates idle/quiet/carrier classification, Task 3 requires those variants to fail RED first**
- Implementing the full union in Task 2 would have made Task 3's mandated RED gate impossible. Task 2's GREEN ships the terminal-only observe path (satisfying every Task 2 acceptance criterion, none of which require variant-green); the variants' classification lands with Task 3's GREEN. The plan's task split is internally consistent only under this reading, which was applied.
- **Committed in:** 97f7d47 (terminal-only), 1e059d2 (variants)

---

**Total deviations:** 1 auto-fixed Rule 1 bug + 2 plan-text adaptations
**Impact on plan:** the harness fix strengthened the exactly-once assertion; both adaptations were required to honor the plan's own TDD RED gates. No scope creep; no checkpoints consumed.

## Issues Encountered

- The tdd-red-evidence adapter requires `targetTest` to be a single test name: the first Task 1 record combined both targets ("A + B") and classified INVALID_RED (no_target_test_failure) despite both failing; re-recording with the primary target (TestHappyTurnCompletedExactlyOnce) classified RED_EVIDENCE_OK with both failures still listed in the parsed evidence. No behavior change — a record-format fix.
- The fake's documented rule ordering (side effects before reply) means turn signals can hit the observe path before Run's send ack returns; the machine handles this honestly — resolution is signal-driven from the moment Run begins, and the ack only starts the floor clock. The happy-path test depends on this ordering and passes deterministically.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03-08 (cancel/guard/corpus completion): the skeleton is ready — states idle/running/cancelling/done are declared, guardTurnLocked and resolveLocked are the seams, StopCancelled/SignalCancel constants exist; cancel transitions, the idempotent Cancel, TestCancelMidTurn/TestDoubleSend, and the JSON corpus-driven completion variants build on this package without touching the observe path
- Plan 03-09 (E2E + phase gate): the turn package joins the full `go test -race ./...` gate the phase runs
- Phase 4 ACP front: session/prompt maps to Machine.Run per prompt; the exactly-once contract and the signal union ARE the interface it inherits (D-04 binding preserved: one machine per app-server sessionId on the shared Conn)

## Threat Surface Scan

No new security-relevant surface beyond the plan's threat model: T-3-06's mitigation IS the shipped design — the single ObserveEnvelope classification path with turnId-matched carrier-tolerant classification, exactly-once resolution locked by the spy-based tests, and no synthesized completion on absence of evidence (ctx expiry and transport failures return errors, never StopReasons). T-3-SC clean: zero new dependencies, `mise run tidy-diff` green, go.mod untouched. Logs carry method names and durations only, never params (prompt content stays out of stderr).

## Self-Check: PASSED

- 03-05-SUMMARY.md exists on disk
- All claimed files exist: internal/turn/{machine,watcher,machine_test}.go + 2 evidence records (FOUND x5)
- All claimed commits are ancestors of HEAD: bfc4697 (Task 1 RED), 97f7d47 (Task 2 GREEN), d517805 (Task 3 RED), 1e059d2 (Task 3 GREEN)
- Acceptance grep: `grep -c 'TestHappyTurnCompletedExactlyOnce' internal/turn/machine_test.go` == 1
- RED evidence classified RED_EVIDENCE_OK (x2 records) before their GREEN stages
- Final gate run: mise run ci fully green (build, vet, lint 0 issues, go test -race all packages, test-experiments); tidy-diff clean; turn package green at -count=3 under -race
- Ledger-measured plan commit count at SUMMARY write: 4 (2 RED + 2 feat, all this plan's own)

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
