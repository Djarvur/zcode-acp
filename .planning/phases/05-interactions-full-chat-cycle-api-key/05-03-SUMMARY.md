---
phase: 05-interactions-full-chat-cycle-api-key
plan: 03
subsystem: turn-lifecycle
tags: [wr-01, wr-07, in-01, in-02, fan-out, teardown, registry]
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: conn observer fan-out (AddObserver), machine wiring, review findings WR-01/07/IN-01/02
provides:
  - ordered router-event fan-out (Conn.AddRouterObserver, SetOnRouterEvent removed)
  - Machine.CloseObservers idempotent teardown (WR-07)
  - slimmed core.Session (CancelTurn/Subscribed deleted, IN-02)
  - IN-01 delivery-ordering doc contract on both fan-outs
affects: [05-05 session/close teardown, Phase-5 multi-session]
actuals:
  tokens: 41000
  tasks: 2
  commits: 6
tech-stack:
  added: []
  patterns: [router-observer-entry fan-out, CloseObservers idempotent teardown]

key-files:
  created: []
  modified:
    - internal/zcode/conn.go
    - internal/zcode/conn_test.go
    - internal/turn/machine.go
    - internal/turn/machine_test.go
    - internal/core/registry.go
    - internal/core/registry_test.go
    - internal/acpfront/front.go

key-decisions:
  - "Router fan-out via routerObserverEntry pointer identity (mirrors AddObserver); single slot + routerMu folded — one mechanism (WR-01)"
  - "NewMachine exposes teardown via Machine.CloseObservers() (idempotent, both fan-outs) — the shape 05-05's close calls (WR-07)"
  - "CancelTurn deleted: it was never set by any Register call site (dead since Phase 3); Subscribed deleted per IN-02 (front.go owns the subscribe gate in the D-03 sequence)"

patterns-established:
  - "fan-out delivery ordering doc contract (IN-01): mid-traffic registration memory-safe, contract = ordering (new observers see subsequent events only)"

requirements-completed: [SESS-01]

coverage:
  - id: D1
    description: "WR-01 router fan-out: ordered delivery to all observers, removal semantics, machine-tier degraded path survives a second machine"
    requirement: SESS-01
    verification:
      - kind: unit
        ref: "go test -race -count=3 -run TestRouterFanout ./internal/zcode/"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 -run TestTwoMachinesDegradedSignalNotStolen ./internal/turn/"
        status: pass
    human_judgment: false
  - id: D2
    description: "WR-07 CloseObservers returns conn observer counts to baseline, idempotent"
    verification:
      - kind: unit
        ref: "go test -race -count=1 -run TestMachineCloseObserversReturnsToBaseline ./internal/turn/"
        status: pass
    human_judgment: false

# Metrics
duration: 41min
completed: 2026-11-20
status: complete
---

# Phase 5 Plan 03: Multi-session enablement — WR-01 fan-out + WR-07 teardown Summary

**Router-event fan-out replaces the stolen single slot; Machine.CloseObservers gives teardown its first release path; dead registry fields deleted (IN-02)**

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3] go.mod dependency discipline blocked the honest RED — the wire shapes required a compiling stub**
- **Found during:** Task 1 RED authoring
- **Issue:** strict RED for TestRouterFanout needs AddRouterObserver to exist (compile-first stub); the plan's RED strategy assumed it.
- **Fix:** compiling stub delegating to the single slot (last-wins — reproduces the defect); GREEN replaced it with the fan-out.
- **Committed in:** 3b54156 (RED) / 04ef5c7 (GREEN)

**2. [Process slip, documented] RED evidence classified AFTER the RED commit**
- **Found during:** evidence capture
- **Issue:** the RED commit (3b54156) landed before the evidence records were finalized (a node script dropped shell vars → records unwritten → first classifier run said unreadable_record).
- **Fix:** tests re-run at the committed RED state (unchanged tree), evidence written in the working camelCase schema, classified RED_EVIDENCE_OK (both targets failed on planned assertions); classification slip recorded here rather than silently absorbed.
- **Committed in:** a21ba05 (evidence records)

**Total deviations:** 2 documented (1 stub strategy, 1 process slip). **Impact on plan:** none on behavior — both RED targets failed on their planned assertions before any GREEN code.

## Task Commits

1. **Task 1: WR-01 router-event fan-out** — RED `3b54156`, GREEN `04ef5c7`
2. **Task 2: CloseObservers + registry slim** — `0197a6e`

**Plan metadata:** `6b83303` (feat commit) + SUMMARY/tracking commits

## Verification

- `go test -race -count=3 -run TestRouterFanout ./internal/zcode/` — green ×3
- `go test -race -count=1 -run TestTwoMachinesDegradedSignalNotStolen ./internal/turn/` — green (degraded path survives)
- `go test -race -count=1 -run TestMachineCloseObserversReturnsToBaseline ./internal/turn/` — green
- `mise run ci` — exit 0 (lint 0 issues after funlen/lll/gofmt fixes)
- Dead-field gate: `grep CancelTurn|Subscribed internal/core/ internal/acpfront/front.go (non-test)` = 0

## Next Phase Readiness

- removeRouterObserver/CloseObservers are the teardown surface plan 05-05 calls
- fan-out ordering is the multi-session delivery contract Phase 5's responder relies on

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-20*
