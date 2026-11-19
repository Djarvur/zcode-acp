---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 08
subsystem: testing
tags: [turn-machine, cancel, bypass-lane, exactly-once, concurrency-guard, -32010, corpus-replay, credential-cipher, d-05-honesty, degraded-completion, tdd, bridge-04]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-05)
    provides: the Machine skeleton (states idle/running/cancelling/done declared, guardTurnLocked/resolveLocked seams, ErrTurnActive wrapper guard, one ObserveEnvelope path, PollWatcher with quiet-window gates) this plan completes
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-02)
    provides: conn.Stop (request-shaped session/stop on the bypass lane — the D-02 cancel primitive), CodeConcurrency constant, the tracked-no-answer router with the injectable RouterDeadline and OnRouterEvent expiry events
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-03)
    provides: the JSON corpus fixtures this plan replays at the turn tier — cancel-mid-turn (script shape), completion-idle-patch, completion-quiet-window, credential-cipher-scripted
provides:
  - Machine.Cancel: the bypass-lane cancel (running -> cancelling transition, session/stop via conn.Stop sent exactly once per turn, idempotent repeats, no-op after completion, no-op with no turn running)
  - Signal-driven exactly-once cancelled resolution through the existing resolveLocked path: a patch leaving running or the quiet settle WHILE cancelling resolves StopCancelled/SignalCancel; a cancel racing a terminal event resolves whichever signal lands first, still exactly once
  - The server-side -32010 mapping: Run maps a CodeConcurrency CallError to ErrTurnActive by the numeric code only (message text never compared), surface-identical to the wrapper-side guard
  - Degraded-completion recording (D-05 honesty at the turn tier): conn.OnRouterEvent wired in NewMachine; a tracked-no-answer deadline expiry mid-turn resolves StopError/SignalDegraded exactly once with the recorded expiry on Resolution.Err — never completed/failed
  - The turn-tier corpus proofs: completion-idle-patch and completion-quiet-window replay through a real zcode.Conn + machine over io.Pipe, each resolving exactly once — ROADMAP Phase 3 success criterion 4 proven end-to-end against the fake app-server
  - RED evidence record 03-08-red-cancel.json, classified RED_EVIDENCE_OK before the feat commit
affects: [03-09 E2E + phase gate (BRIDGE-04's third declaring plan), phase-4 ACP front (session/cancel maps to Machine.Cancel per prompt)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 10700   # ~42840 chars over this plan's diff (39120 code diff + 3716 evidence record; 4 files, 855 insertions / 17 deletions)
  tasks: 3
  commits: 3     # MEASURED from the plan ledger (1bfee84..HEAD): 1 RED + 1 feat + 1 corpus/honesty

commits: 3
plan_head_before: 1bfee8415bc251969ef897813b476be0ecb0175f
plan_head_after: 421c9d2485583d7f111e88b82786f4d3600ce11b

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: tidy-diff green, go.mod untouched)
  patterns:
    - "Signal-driven cancel: Cancel() only transitions + sends the bypass stop; the resolution stays on the one observe path — the machine state (cancelling) rewrites the idle-patch/quiet outcomes to StopCancelled while terminal events keep their own reason, so first-signal-wins falls out of resolveLocked for free"
    - "Exchange-tier bypass proof via a test-controlled read gate: pausing the fake's reader parks the conn writer in the in-flight Write and identifiable marker calls in the ordered queue — the writer-tier TestStopJumpsQueue mechanism reproduced through the real codec + fake exchange"
    - "Server-side guard proof through a second Machine on the same session (its own per-machine guard is idle): the send reaches the wire, the scripted -32010 refusal (non-bundle message text) comes back, and the mapping must fire on the numeric code alone"
    - "Degraded resolution as a first-class Resolution variant: SignalDegraded + Resolution.Err carrying the wrapped static errObligationExpired sentinel with method + deadline — the D-05 expiry is observable API surface, never a swallowed log line"

key-files:
  created:
    - internal/turn/cancel_test.go
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-08-red-cancel.json
  modified:
    - internal/turn/machine.go
    - internal/turn/machine_test.go

key-decisions:
  - "Cancel is fire-and-forget on the wire (conn.Stop, the 03-02 D-02 primitive) and signal-driven on the resolution: the machine never waits for the stop ack and never resolves optimistically at Cancel() time — the cancelled state patch (corpus cancel-mid-turn shape) or the quiet settle while cancelling carries the honest cancelled verdict"
  - "Idempotence is guarded by the resolved flag + machine state under the machine mutex: the first caller on a running turn transitions to cancelling and sends the stop; every other call path (repeat, post-completion, no-turn) writes nothing"
  - "Any tracked-no-answer deadline expiry mid-turn resolves the degraded state, not just credential-cipher's: the honesty rule (an obligation expiry is evidence the turn cannot complete, never a completion signal) is method-independent; credential-cipher is the scripted proof, not a special case"
  - "The -32010 mapping lives in Run's send-error path keyed on zcode.CodeConcurrency (numeric constant); ErrTurnActive doubles as the wrapper-side guard and the server-rejection surface so callers match one sentinel"
  - "Test-level: the corpus tier loads the SAME JSON fixtures the golden corpus records via LoadScenario (the corpus is the shared protocol fixture, not a copy), with the machine's millisecond quiet knobs injected so no test waits on wall-clock defaults"

patterns-established:
  - "readGate harness: a pausable io.Reader over the fake's pipe end (single-reader wake channel) for deterministic wire-order assertions at the exchange tier — pause, park the writer in-flight, queue markers, cancel, resume, read the order off the transcript"
  - "Marker-tagged ordinary calls (params.marker values) let transcript scans identify which frames a bypass frame must precede — no fake-side instrumentation needed"
  - "waitAck (non-zero AckAt) as the deterministic 'turn is running' sync point: the fake's side effects precede the ack, so turn id and running latch are already recorded when the ack lands"

requirements-completed: [BRIDGE-04]  # COPIED VERBATIM from 03-08-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-04: 03-05/03-08/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Cancel exactly-once over the bypass lane: session/stop on the wire ahead of ordinary traffic parked behind the in-flight write, exactly one StopCancelled/SignalCancel resolution, idempotent second Cancel (no second stop frame), cancel-after-complete no-op, fresh Run after done is a new turn; a cancel racing a terminal resolves whichever signal lands first, still exactly once"
    requirement: BRIDGE-04
    verification:
      - kind: unit
        ref: "internal/turn/cancel_test.go#TestCancelMidTurnAnswersOnce + #TestCancelRacingTerminalResolvesOnce"
        status: pass
    human_judgment: false
  - id: D2
    description: "The -32010 double-send guard both ways: wrapper-side second Run rejected ErrTurnActive with zero session/send on the wire; server-side refusal (a second machine on the same session, fixture message deliberately non-bundle) mapped to the same sentinel by the numeric CodeConcurrency code, message text never compared"
    requirement: BRIDGE-04
    verification:
      - kind: unit
        ref: "internal/turn/cancel_test.go#TestDoubleSendRejectedWrapperSide + #TestDoubleSendRejectedServerSide"
        status: pass
    human_judgment: false
  - id: D3
    description: "Corpus-level completion (ROADMAP Phase 3 success criterion 4): completion-idle-patch and completion-quiet-window replay through LoadScenario -> fake.Serve over io.Pipe -> zcode.Conn -> Machine.Run and each resolves exactly once with the expected StopReason and resolving-signal record (idle patch; quiet window strictly after the send-latency floor post-ack)"
    requirement: BRIDGE-04
    verification:
      - kind: integration
        ref: "internal/turn/cancel_test.go#TestCompletionCorpus (idle-patch, quiet-window subtests)"
        status: pass
    human_judgment: false
  - id: D4
    description: "D-05 honesty at the turn tier (T-3-12 mitigation proven): the scripted mid-turn credential-cipher exchange (corpus scenario 10) through the real codec path ends with the tracked-no-answer expiry RECORDED — surfaced via conn.OnRouterEvent into a StopError/SignalDegraded resolution carrying the expiry on Resolution.Err, exactly once, never completed/failed; zero host reply bytes for the fake's reverse request and an honest unanswered note on the fake side"
    requirement: BRIDGE-04
    verification:
      - kind: integration
        ref: "internal/turn/cancel_test.go#TestCredentialCipherExpiryRecorded"
        status: pass
    human_judgment: false

# Metrics
duration: 25min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 08: Turn Machine Part 2 — Cancel, Double-Send Guard, Corpus Completion Summary

**Bypass-lane cancel with exactly-once cancelled resolution, the numeric-code -32010 double-send guard, the JSON-corpus completion variants, and the D-05 credential-cipher honesty proof — ROADMAP Phase 3 success criterion 4 (fire-and-forget send lifted into an exactly-once synchronous turn that honors cancel mid-turn) now proven against the fake app-server**

## Performance

- **Duration:** 25 min
- **Started:** 2026-11-19T20:34:56Z
- **Completed:** 2026-11-19T21:00:18Z
- **Tasks:** 3
- **Files:** 4 (2 Go source, 1 test file, 1 RED evidence record)

## Accomplishments

- Task 1 (TDD RED): all three cancel/guard targets failed on their planned assertions against the deliberate no-op `Machine.Cancel` stub and the unmapped wire-error passthrough — cancel resolved over the quiet window because nothing sent session/stop, and the server-side -32010 surfaced the raw `zcode.CallError` (RED_EVIDENCE_OK, go-test adapter, evidence `03-08-red-cancel.json`)
- Task 2 (GREEN): `Cancel` implemented — running -> cancelling transition, `conn.Stop` on the bypass lane sent exactly once per turn, idempotent repeats and no-op after completion (PITFALL 4); the resolution stays on the 03-05 one observe path: a patch leaving running or the quiet settle while cancelling resolves StopCancelled/SignalCancel, and a cancel racing a terminal event resolves whichever signal lands first (still exactly once via `resolveLocked`); Run maps a server `CodeConcurrency` (-32010) refusal to `ErrTurnActive` by the numeric code — the fixture's refusal message is deliberately non-bundle text, proving code-only matching
- Task 3: the corpus tier — completion-idle-patch and completion-quiet-window replay through `LoadScenario` -> `fake.Serve` over io.Pipe -> real `zcode.Conn` -> `Machine.Run`, each resolving exactly once with the expected signal record (the quiet variant gated strictly after the send-latency floor post-ack); `TestCredentialCipherExpiryRecorded` proves the D-05 honesty contract at the turn tier — the tracked-no-answer expiry is surfaced via `conn.OnRouterEvent` into a StopError/SignalDegraded resolution carrying the recorded expiry on `Resolution.Err`, resolved exactly once, never completed/failed, with zero host reply bytes for the fake's reverse request and the honest unanswered note on the fake side
- Exchange-level bypass proof with teeth: a test-controlled read gate parks the conn writer in the in-flight write while two identifiable marker calls sit in the ordered queue — the transcript must show the stop ahead of both markers (the writer-tier TestStopJumpsQueue mechanism reproduced through the real codec + fake exchange)
- Full gates green: `go test -race -count=1 ./...` all packages; `mise run ci` (build, vet, lint 0 issues, test, test-experiments fail 0); `mise run tidy-diff` clean (go.mod untouched); new tests stable at `-count=3` under `-race`

## TDD Record (Task 1; Tasks 2-3 as planned)

- **Task 1 RED:** `go test -race -count=1 -run 'TestCancel|TestDoubleSend' ./internal/turn/` — TestCancelMidTurnAnswersOnce (reason "completed", want "cancelled"), TestCancelRacingTerminalResolvesOnce (signal "quiet-window", want "terminal-event"), and TestDoubleSendRejectedServerSide (raw CallError, want ErrTurnActive) failed on their planned assertions; TestDoubleSendRejectedWrapperSide passed as the expected lock of the 03-05 guard. Record classified **RED_EVIDENCE_OK** (target_test_failed) before the GREEN commit
- **Task 2 GREEN:** all cancel + guard targets green under `-race` alongside the 03-05 suite (TestHappyTurn, TestCompletionVariants, carrier tolerance) — no completion-path regression
- **Task 3 (type auto, non-TDD):** corpus + honesty tests written and green in one commit; the degraded-completion machinery they exercise (SignalDegraded, Resolution.Err, observeRouterEvent) landed in the same commit

## Task Commits

Each task was committed atomically (Task 1 carries the RED commit per TDD):

1. **Task 1 RED: failing cancel + double-send guard tests (evidence RED_EVIDENCE_OK)** - `584b582` (test)
2. **Task 2 GREEN: bypass-lane cancel + double-send guard** - `efd3178` (feat)
3. **Task 3: corpus completion variants + credential-cipher honesty** - `421c9d2` (test)

## Files Created/Modified

- `internal/turn/cancel_test.go` - the cancel exactly-once + racing-terminal + double-send (both tiers) + corpus completion + credential-cipher honesty contract tests; readGate/gatedFakeEnd harness, marker-tagged queue occupancy, transcript wire-order assertions
- `internal/turn/machine.go` - Cancel (bypass stop, cancelling transition, idempotent), cancelling-state resolution branches in the one observe path, server-side CodeConcurrency mapping in Run, SignalDegraded + Resolution.Err + observeRouterEvent (wired in NewMachine), static errObligationExpired sentinel
- `internal/turn/machine_test.go` - startFake generalized to io.ReadWriteCloser (harness wrapper support), shared unknownMethodReply const (goconst)
- `.planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-08-red-cancel.json` - persisted RED evidence, RED_EVIDENCE_OK

## Decisions Made

- Cancel is fire-and-forget on the wire and signal-driven on the resolution: it never waits for the stop ack and never resolves optimistically — the cancelled state patch (the corpus cancel-mid-turn shape) or the quiet settle while cancelling carries the verdict, so a cancel racing a terminal resolves whichever lands first through the existing exactly-once path
- Quiet-settle-during-cancelling resolves cancelled (not completed): after a stop the wind-down must not claim a completed turn — the same settle is only "completed" evidence when no stop is outstanding
- Any tracked-no-answer deadline expiry mid-turn resolves the degraded state (method-independent honesty rule); credential-cipher is the scripted proof, and the expiry record (method + deadline on Resolution.Err) is the observable API surface, not just a log line
- The server-side -32010 proof uses a second Machine on the same session (per-machine guards, so the send genuinely reaches the wire) — the realistic recreated-machine scenario; one sentinel (ErrTurnActive) serves both guard paths
- The exchange-level bypass assertion uses the writer-tier occupancy trick (TestStopJumpsQueue): a read gate over the fake's pipe end makes the occupancy deterministic instead of racing wall clocks (the repo's bounded-settle pattern applied structurally)

## Deviations from Plan

### Plan-Text Adaptations (not defects)

**1. Task 1's `files_modified` names only cancel_test.go, but compiling tests require the Machine.Cancel declaration**
- The RED contract tests cannot compile against a method that exists nowhere; the deliberate no-op stub shipped with Task 1's RED commit `584b582` (machine.go), mirroring 03-05's documented stub precedent. The stub is in the plan frontmatter's own artifact list ("machine.go ... contains Cancel").
- **Committed in:** 584b582

**2. Task 1's action text names TestDoubleSendRejected; the shipped tests are TestDoubleSendRejectedWrapperSide + TestDoubleSendRejectedServerSide**
- funlen (60-line cap) rejects the combined table as one function; the split keeps both tiers under the gate and each tier's contract clearer. The plan's verify regex ('TestCancel|TestDoubleSend') matches both names unchanged.
- **Committed in:** 584b582

**3. Task 3's `files_modified` names only cancel_test.go, but the D-05 honesty surfacing lives in machine.go**
- The plan's own artifact table assigns "degraded-completion recording" to machine.go; the test-only file list was internally inconsistent. observeRouterEvent + SignalDegraded + Resolution.Err landed with Task 3's commit per the mandated commit message.
- **Committed in:** 421c9d2

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Harness generalizations required by the cancel tier**
- **Found during:** Task 1 (first vet run)
- **Issue:** startFake was typed `*pipeEnd`, so the gated fake end (the bypass-occupancy harness) could not be passed; goconst flagged the 4th "MethodNotFound" literal across the package's programmatic scenarios; revive context-as-argument rejected the ctx-not-first helper signatures
- **Fix:** startFake takes io.ReadWriteCloser; unknownMethodReply const shared in machine_test.go; runAsync/queueQueuedCall reordered ctx-first; funcorder/lll/err113/funlen findings fixed in the same tasks (static errObligationExpired sentinel wrapping; read-gate method ordering)
- **Files modified:** internal/turn/machine_test.go, internal/turn/cancel_test.go, internal/turn/machine.go
- **Verification:** mise run lint 0 issues; full suite green
- **Committed in:** 584b582, 421c9d2

---

**Total deviations:** 3 plan-text adaptations + 1 bundled auto-fix (Rule 3 blocking)
**Impact on plan:** all adaptations required by the plan's own artifact table, lint gates, or the tests' ability to compile; no scope creep; no checkpoints consumed.

## Issues Encountered

- The RED cancel failure surfaced as "resolved completed over the quiet window" rather than a hang: with the no-op stub the watcher's quiet path legitimately ends the turn, so the RED assertion had to bind the resolution REASON (cancelled) rather than resolution-at-all — a stronger contract, since a post-cancel quiet resolution claiming completed is exactly the dishonesty the plan prohibits.
- The exchange-level bypass assertion initially had no teeth when the writer's queue was empty at cancel time (a queue-lane stop is indistinguishable then); the read-gate occupancy makes the jumping observable deterministically.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03-09 (E2E + phase gate): BRIDGE-04's third declaring plan — with this SUMMARY, 03-05 + 03-08 have landed and BRIDGE-04 completion stays gated on 03-09 (requirements.ready-ids correctly returns [] until then); the phase gate re-runs the full suite this plan kept green
- Phase 4 ACP front: session/cancel maps to Machine.Cancel per prompt; the exactly-once cancelled contract, the idempotence surface, and the ErrTurnActive sentinel are the interface it inherits (D-04 binding preserved: one machine per app-server sessionId on the shared Conn)
- Phase 5/6 note for live reconciliation: the degraded resolution fires on ANY tracked-no-answer expiry mid-turn — when Phase 5's interaction responder starts answering obligations, the RouterDeadlineExpired class shrinks to the genuinely-unanswered, and StopError/SignalDegraded remains the honest end state for real expirations

## Threat Surface Scan

No new security-relevant surface beyond the plan's threat model: T-3-11's mitigation IS the shipped cancel design (exactly-once resolution test-locked, stop on the bypass lane so cancel cannot deadlock behind prompts, idempotent repeats proven at the exchange level); T-3-12's mitigation is proven at the turn tier (zero reply bytes for the cipher request asserted on the transcript; the degraded resolution records the expiry without fabricating any answer); T-3-SC clean — zero new dependencies, `mise run tidy-diff` green, go.mod untouched. Logs carry method names, session ids and durations only; the read-gate markers and resolution records never carry params content.

## Self-Check: PASSED

- 03-08-SUMMARY.md exists on disk
- All claimed files exist: internal/turn/cancel_test.go, internal/turn/machine.go, internal/turn/machine_test.go, evidence/03-08-red-cancel.json (FOUND x4)
- All claimed commits are ancestors of HEAD: 584b582 (Task 1 RED), efd3178 (Task 2 GREEN), 421c9d2 (Task 3)
- Acceptance grep: `grep -c 'TestCancelMidTurnAnswersOnce' internal/turn/cancel_test.go` == 1
- RED evidence classified RED_EVIDENCE_OK (go-test adapter) before the GREEN commit
- Final gate run: mise run ci fully green (build, vet, lint 0 issues, go test -race all packages, test-experiments fail 0); tidy-diff clean; new tests stable at -count=3 under -race
- Ledger-measured plan commit count at SUMMARY write: 3 (1 RED + 1 feat + 1 corpus/honesty, all this plan's own)

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
