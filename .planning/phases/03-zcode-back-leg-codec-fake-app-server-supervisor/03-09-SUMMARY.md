---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 09
subsystem: testing
tags: [e2e, exec-chain, composition, supervisor, fake-binary, codec, turn-machine, phase-gate, zero-dependencies, ci-isolation]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-07)
    provides: the supervisor half this plan composes — Supervisor.Start over the SpawnFunc seam, SpawnChild (Setpgid, pipes, capped stderr drain), ChildEnv (allowlist + proxy scrub), the staged shutdown, and the shared external-test-package helpers (waitSpawnedChild/assertReaped/assertGone)
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-06)
    provides: the standalone fake-app-server binary (cmd/fake-app-server, --scenario, stdio serving, EOF-drain exit) and the exec-tier spawn hygiene conventions
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plans 03-01..03-05, 03-08)
    provides: the codec Conn (demux, WR-02 lock, bypass lane, D-05 router), the golden corpus fixtures, and the turn Machine (exactly-once completion, bypass-lane cancel, -32010 guard)
provides:
  - internal/supervisor/e2e_test.go: TestEndToEndExecChain — the composed exec-chain proof: each subtest runs a FULL supervised lifecycle (discovery override → SpawnChild of the BUILT fake binary → Conn over the exec pipes → machine turn) and ends in the poll-verified zero-survivors staged shutdown; happy turn resolves completed exactly once, mid-turn cancel resolves cancelled exactly once
  - the phase-gate re-proof: mise run ci (build → vet → lint 0 issues → go test -race ./... with the golden corpus and the E2E inside → test-experiments 22/22) + tidy-diff clean — go.mod untouched
affects: [phase 4 ACP front (inherits the proven spawn → Conn → machine composition and the e2eStartCycle harness shape), /gsd:verify-work (the phase's execution-complete evidence)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 4060    # 16242 chars over this plan's diff (1 file, 456 insertions) / 4
  tasks: 2
  commits: 1     # MEASURED: git rev-list --count 6225bd6..68c2747 (Task 1; Task 2 needed no fixes — gate-run only, the plan's empty-files annotation)

commits: 1
plan_head_before: 6225bd6c8ed79ed8fa1f5b6ae5ee310f175e5282
plan_head_after: 68c27474cb17a3a61b526acb2de7c84a5a55f08d

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: tidy-diff clean; go.mod still declares only the module + go 1.27 lines)
  patterns:
    - "E2E composition via the shipped seams only: the discovery Override resolves the synthetic Install, the SpawnFunc seam hands SpawnChild the fake's argv ([fake, --scenario, <corpus row>]) with the REAL ChildEnv output, and Start's health watch / crash window / staged shutdown run production-unchanged — the test composes, never mocks, the phase's layers"
    - "Hooks-before-frames discipline at the E2E tier: the Machine is constructed BEFORE any frame flows — the Conn's OnNotification/OnRouterEvent hooks are read unsynchronized by the demux and router goroutines, so assigning them mid-exchange is a data race (found and fixed here under -race)"
    - "Per-test-run binary build: a process-global sync.Once cache points at a TempDir that is DELETED when the first test run ends — -count repeats must build into the current run's t.TempDir() once per parent-test invocation"
    - "Corpus-row-driven exec proofs: the exec tier replays the SAME testdata scenario JSONs the in-process golden tests pin; cancel-mid-turn has no create rule, so its machine binds the fixed corpus session id directly (the 03-08 corpus-proof shape)"

key-files:
  created:
    - internal/supervisor/e2e_test.go
  modified: []  # the conditional spawn.go/supervise.go ChildSpec seam candidates were NOT needed — the shipped 03-07 SpawnFunc seam accepts a prepared spec directly (production Install path untouched)

key-decisions:
  - "The cancel turn runs as a SECOND supervised cycle serving the cancel-mid-turn corpus row instead of a second machine on the happy child's Conn: the fake's Times=0 rules are unlimited, so the happy-turn send rule re-fires turn.completed on a second send — a same-Conn cancel there would race a completed resolution and could never replay corpus-2 shapes (documented as a plan-text adaptation; both cycles still exercise the full supervisor lifecycle, and the ESRCH proof runs twice)"
  - "The child env is the REAL ChildEnv(inst, os.Environ()) output even though the plain-Go fake ignores every var in it: the E2E's subject is the whole chain, so the production env path runs unchanged — and T-3-13 stays satisfied because the child is the built FAKE binary only"
  - "PollWatcher knobs injected at milliseconds (QuietMs 40, FloorMultiple 2, PollIntervalMs 15 — the 03-05 test pattern): the terminal/cancel signals land in ~ms, deterministically beating the ~120 ms quiet-settle floor; the fake's session/messages polls answer -32601 via unknown_method and the watcher treats failed polls as recorded-not-fatal"
  - "Each corpus row gets its own full supervised lifecycle and its ESRCH check completes before the next cycle spawns, so a reused pid can never contaminate the zero-survivors proof"
  - "The machine is constructed before session/create in the happy cycle (binding the known corpus session id), and the create reply is asserted against that binding — the hook contract made this mandatory, and the assertion keeps the consistency check the decode-then-bind shape would have had"

patterns-established:
  - "e2eStartCycle harness (discovery override → supervisor → SpawnChild of a built binary → Conn over e2eDuplex): the template the Phase 4 front leg's tests can lift for full-chain proofs without the desktop bundle"
  - "OnResolve-spy exactly-once assertion across a real child boundary (the resolutionSpy pattern from 03-05, reused at the E2E tier)"

requirements-completed: [BRIDGE-01, BRIDGE-02, BRIDGE-03, BRIDGE-04]  # COPIED VERBATIM from 03-09-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (this final plan completes all four: BRIDGE-01 03-01/02, BRIDGE-02 03-01/03/06, BRIDGE-03 03-04/07, BRIDGE-04 03-05/08)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "BRIDGE-01 proven over the exec chain: the codec Conn rides the real child's pipes — strict demux, single-writer ordering, and the D-05 reverse-request router answering the create-time session/requestRuntimePreferences obligation over exec pipes (the create reply returns only after the fake's expect_answer validates the router's answer verbatim; unanswered creates would fail -32603)"
    requirement: BRIDGE-01
    verification:
      - kind: integration
        ref: "internal/supervisor/e2e_test.go#TestEndToEndExecChain/happy_turn_completes_exactly_once"
        status: pass
    human_judgment: false
  - id: D2
    description: "BRIDGE-02 at the exec tier: the child is the BUILT fake-app-server binary (go build into the run's temp dir — never an in-process Serve, never the desktop bundle per T-3-13) serving the SAME corpus scenario JSONs the in-process golden tests pin; both rows replay over real stdio with the EOF-drain exit contract owned by the staged shutdown"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/supervisor/e2e_test.go#TestEndToEndExecChain (both subtests; acceptance grep 'go build' = 4 >= 1)"
        status: pass
    human_judgment: false
  - id: D3
    description: "BRIDGE-03 composed end-to-end: the REAL Discover resolves the synthetic Install through the order-step-1 Override (all other probes run to completion against an empty fixture home — hermetic by nil seams, 8 probe rows recorded), the production Start machinery owns the child, and both subtests end in the staged shutdown: Start returns nil, monitor reaped (Done/Exited), kill(-pgid, 0) ESRCH by bounded poll — zero survivors, twice"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/e2e_test.go#TestEndToEndExecChain (e2eDiscoverOverride + e2eShutdownAndAssertGone, x2 cycles)"
        status: pass
    human_judgment: false
  - id: D4
    description: "BRIDGE-04 proven over the exec chain: Machine.Run resolves the happy turn completed exactly once (turnId-matched terminal event, OnResolve spy) and the ack-gated mid-turn Cancel resolves cancelled exactly once (SignalCancel via the fake's stop rule's cancelled patch) — the exactly-once contract holds across a real child boundary under -race -count=3"
    requirement: BRIDGE-04
    verification:
      - kind: integration
        ref: "internal/supervisor/e2e_test.go#TestEndToEndExecChain (e2eResolutionSpy.assertSingle in both subtests)"
        status: pass
    human_judgment: false

# Metrics
duration: 18min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 09: End-to-End Exec Chain + Full Phase Gate Summary

**The full phase chain composed in one -race test — the supervisor spawns the BUILT fake-app-server binary through the production lifecycle, the codec Conn rides real exec pipes with the D-05 router answering the create obligation, the turn machine resolves happy and cancelled turns exactly once over the golden-corpus fixtures with a twice-proven zero-survivors staged shutdown — and the canonical mise run ci gate lands fully green with zero new dependencies: the phase is execution-complete**

## Performance

- **Duration:** 18 min
- **Started:** 2026-11-19T21:02:38Z
- **Completed:** 2026-11-19T21:20:40Z
- **Tasks:** 2
- **Files:** 1 created (internal/supervisor/e2e_test.go, 456 lines)

## Accomplishments

- Task 1 (`68c2747`): internal/supervisor/e2e_test.go — TestEndToEndExecChain composes every phase layer in one test, on real exec pipes, green under `-race`:
  - the fake binary is BUILT once per test run (`go build` of cmd/fake-app-server into the run's temp dir — the RESEARCH A2-blessed fallback; no module deps) and spawned through the supervisor's production Start machinery via the SpawnFunc seam, with the REAL ChildEnv(Install, os.Environ()) allowlist output, Setpgid process group, and the capped stderr drain — no production files touched (the conditional spawn.go/supervise.go ChildSpec seam candidates were NOT needed: the shipped 03-07 SpawnFunc seam accepts a prepared spec directly)
  - the discovery layer participates: the REAL Probers.Discover resolves the synthetic Install through the order-step-1 Override while every other probe runs to completion against an empty fixture home (hermetic — nil seams record misses, 8 probe rows logged)
  - happy-turn corpus row: session/create over the exec pipes with the conn's router answering the create-time runtime-preferences obligation inside the fake's expect_answer validation, then Machine.Run resolves completed exactly once (turnId-matched terminal event; OnResolve spy)
  - cancel-mid-turn corpus row: machine bound to the corpus session id, ack-gated mid-turn Cancel through the bypass lane, resolved cancelled exactly once (SignalCancel via the fake's stop rule's cancelled patch)
  - both subtests end in the staged shutdown: Start returns nil on the intentional path, the monitor reaps (Done closed, Exited delivered), and kill(-pgid, 0) reports ESRCH by bounded poll — zero survivors, proven twice
- Task 2 (no commit — no fixes were needed): the full phase gate — mise run ci fully green (build → vet → lint 0 issues → go test -race -count=1 ./... with the golden corpus and the E2E inside → node --test experiments 22/22), mise run tidy-diff clean (go.mod still declares only the module + go 1.27 lines — T-3-SC), and the explicit acceptance run `go test -race -count=1 ./internal/zcode/... ./internal/supervisor/... ./internal/turn/...` green in one pass
- Stability: the E2E suite green at `-race -count=3`; the lint gate 0 issues on the strict config

## Task Commits

Each task was committed atomically:

1. **Task 1: end-to-end exec chain through supervisor + fake + codec + turn** - `68c2747` (test)
2. **Task 2: full phase gate — mise run ci + tidy-diff** - no commit (gate fully green on the first run; the plan commits only if fixes were needed, and none were)

## Files Created/Modified

- `internal/supervisor/e2e_test.go` - the composed exec-chain proof: TestEndToEndExecChain (two subtests), e2eBuildFakeBinary (go build per test run), e2eStartCycle (discovery override → supervisor → SpawnChild → Conn over e2eDuplex), e2eDiscoverOverride, e2eShutdownAndAssertGone (Start nil + reaped + ESRCH poll), e2eResolutionSpy (exactly-once), the ack gate and async-run helpers
- *(no other files — the plan's conditional seam candidates in spawn.go/supervise.go were unnecessary)*

## Decisions Made

- The cancel turn is a second supervised cycle serving cancel-mid-turn.json rather than a second machine on the happy child's Conn: the fake's Times=0 request rules are unlimited, so a second send against happy-turn.json re-fires the send rule including its turn.completed side effect — a same-Conn cancel would race a completed resolution and could never replay corpus-2 shapes. The chosen shape keeps every must-have truth and doubles the zero-survivors proof (documented as a deviation below)
- The child env is the real ChildEnv output (PATH+HOME allowlist, ELECTRON_RUN_AS_NODE=1) even though the plain-Go fake ignores it — the E2E proves the chain as shipped, and the fake binary child keeps CI hermetic (T-3-13)
- The machine is constructed BEFORE any frame flows (Conn hook contract): NewMachine assigns OnNotification/OnRouterEvent, which the demux and router goroutines read unsynchronized — constructing it mid-exchange is a data race, found by -race on the first run and fixed by construction-order
- PollWatcher knobs at milliseconds (40/2/15) so the terminal and cancel signals deterministically beat the ~120 ms quiet settle floor; failed session/messages polls (the corpus rows have no messages rule → -32601) are recorded-not-fatal, exactly the watcher's documented contract
- Each corpus row's ESRCH check completes before the next cycle spawns, so pid reuse can never contaminate the zero-survivors assertion

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Data race: machine constructed after frames flowed**
- **Found during:** Task 1 (first -race run)
- **Issue:** the first draft bound the machine after session/create returned, but NewMachine writes conn.OnRouterEvent while the router's request goroutine (spawned by the create exchange) reads it — the Conn hook contract requires hooks assigned before any frame flows
- **Fix:** construct the machine (binding the known corpus session id) before the create Call; assert the create reply's sessionId against the binding, preserving the consistency check
- **Files modified:** internal/supervisor/e2e_test.go
- **Committed in:** 68c2747

**2. [Rule 1 - Bug] -count repeats failed: process-global sync.Once cached a deleted TempDir**
- **Found during:** Task 1 (stability run at -count=3)
- **Issue:** the once-built binary path pointed into the FIRST test run's t.TempDir(), which testing removes when that run ends — repeats 2 and 3 spawned a nonexistent binary (every child spawn failed within the 5 s spawn watchdog)
- **Fix:** build per parent-test invocation into the current run's t.TempDir() (one build per run still holds — the parent test is the only caller; the sync.Once and its globals are gone)
- **Files modified:** internal/supervisor/e2e_test.go
- **Verification:** -race -count=3 green
- **Committed in:** 68c2747

**3. [Rule 3 - Blocking] Strict-gate lint findings**
- **Found during:** Task 1 (mise run lint before the commit)
- **Issue:** errname (e2eFakeErr not errXxx), gochecknoglobals (x2 — the sync.Once cache), lll (x2 — helper signatures over 120 cols), contextcheck (NewConn in a ctx-carrying helper)
- **Fix:** the globals and sentinel vanished with the per-run build; signatures wrapped; the NewConn line carries the same //nolint:contextcheck justification realdesktop_test.go established (NewConn owns its base context)
- **Files modified:** internal/supervisor/e2e_test.go
- **Verification:** mise run lint 0 issues
- **Committed in:** 68c2747

### Plan-Text Adaptations (not defects)

**1. "a second machine/turn on the same Conn" → a second supervised cycle serving the cancel corpus row**
- The plan's Task-1 action text reads the cancel turn onto the happy child's Conn. With the shipped fake, that composition cannot replay "corpus scenario 2 shapes": the happy-turn send rule is unlimited (Times=0) and its side effects include turn.completed, so the second send resolves completed within ~ms of the ack — before any cancel can land. The cancel turn therefore runs as a second full supervised cycle serving cancel-mid-turn.json (the corpus shapes verbatim: no create rule, send-without-terminal, stop rule emitting the cancelled patch), through the SAME supervisor machinery, ending in the same zero-survivors proof. Both frontmatter truths ("cancel works over the exec chain … exactly once … zero surviving group members") are satisfied; the same-Conn literal is the only text not honored, and honoring it would have made the acceptance criterion ("cancel resolved cancelled exactly once") unraceable-by-construction.
- **Committed in:** 68c2747

---

**Total deviations:** 3 auto-fixed (2 Rule 1, 1 Rule 3) + 1 plan-text adaptation
**Impact on plan:** none on scope — every fix was inside the one file the plan owns, the production files were never touched, and all three must-have truths verify green. The phase gate (Task 2) needed no fixes at all.

## Issues Encountered

- None beyond the fixes above — the composed chain passed -race on its second run and stayed green across -count=3 repeats; the full ci gate passed on its first run

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The phase is EXECUTION-COMPLETE: all nine plans landed; BRIDGE-01..04 complete; the canonical gate green with zero new dependencies
- /gsd:verify-work inherits: the composed E2E as the automated proof, plus the ONE documented end-of-phase human check from 03-07 (the real-desktop positive path: `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop ./internal/supervisor/` — manual-only, never CI)
- Phase 4 ACP front: e2eStartCycle + the childDuplex/e2eDuplex pattern are the reference for wiring its Conn over Supervisor.Start; Machine.Run per session/prompt inherits the exactly-once contract proven here across a real child boundary

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| *(none new — this plan introduces no stubs; the inherited defaultOverride no-op from 03-04 remains Phase 4's config-surface work)* | | | |

## Self-Check: PASSED

- 03-09-SUMMARY.md exists on disk
- Claimed file exists: internal/supervisor/e2e_test.go (FOUND)
- Claimed commit is an ancestor of HEAD: 68c2747 (Task 1); Task 2 intentionally commit-free (gate green, no fixes)
- Acceptance greps: 'go build' in e2e_test.go = 4 (>= 1); TestEndToEndExecChain present (x2: test + subtest refs)
- E2E green: go test -race -count=1 -run 'TestEndToEndExecChain' ./internal/supervisor/ ok (also -count=3)
- Final gate: mise run ci fully green (build → vet → lint 0 issues → test -race all packages → test-experiments 22/22); mise run tidy-diff clean; go.mod = module + go 1.27 only
- Ledger-measured plan commit count: 1 (plan_head_before 6225bd6, plan_head_after 68c2747)

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
