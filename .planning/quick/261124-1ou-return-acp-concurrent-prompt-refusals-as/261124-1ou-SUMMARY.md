---
phase: 261124-1ou
plan: 01
subsystem: acp-wire-protocol
tags: [acp, json-rpc, error-codes, session-busy, orca, omp, golang]
requires:
  - phase: 04-07
    provides: ACP front turn slot, machine ErrTurnActive mapping, /model slash busy gate
provides:
  - acpport.CodeSessionBusy (-32003), ReasonSessionBusy ("session_busy"), NewSessionBusy
  - wireError data.reason carriage alongside the existing data.details carriage
  - all three concurrent-prompt rejection sites emit -32003 + data.reason session_busy
  - dialect-doc Agent-side busy contract subsection
affects: [orca-live-testing, docs/protocol/acp-client-dialect.md, future prompt-queueing work]
actuals:
  tokens: 8980
  tasks: 3
  commits: 3
tech-stack:
  added: []
  patterns:
  - "Opt-in wire data payload per seam error: Details -> data.details, Reason -> data.reason, neither -> data null"
  - "Wire-level test lock checks the SDK-decoded RequestError and the raw JSON-RPC tee frame"
key-files:
  created: []
  modified:
  - internal/acpport/types.go
  - internal/acpport/agent.go
  - internal/acpport/agent_test.go
  - internal/acpfront/front.go
  - internal/acpfront/turnctx.go
  - internal/acpfront/cancel_test.go
  - internal/acpfront/slashbusy_internal_test.go
  - internal/acpfront/slashmodel_test.go
  - docs/protocol/acp-client-dialect.md
key-decisions:
  - "Concurrent-prompt refusals use JSON-RPC -32003 with data exactly {reason: session_busy}; the message text stays byte-identical"
  - "CodeSessionBusy is a standalone const so the plan's literal structural grep (single-space declaration) passes"
  - "The busy frame is located on the tee by its -32003 code rather than first-error-frame, because fan-in flows also carry a -32021 cancel frame"
metrics:
  duration: 11min
  completed: 2026-11-24
status: complete
commits: 3
plan_head_before: 2472f3e47a83386906efcb683e2217ac89c0325b
plan_head_after: c1887e446c8e2184d48832d0a0a15beac1735e80
---

# Quick Task 261124-1ou: Return ACP concurrent-prompt refusals as -32003 session_busy

**Concurrent session/prompt refusals now answer JSON-RPC -32003 with data exactly {"reason":"session_busy"} from all three guard layers, so Orca's retryBusyPrompts queue engages for the omp card while the busy message, every other error code and cancel/stop semantics stay untouched.**

## Performance

- **Duration:** 11 min
- **Started:** 2026-11-23T22:24:13Z
- **Completed:** 2026-11-23T22:35:17Z
- **Tasks:** 3
- **Files modified:** 9

## Accomplishments

- Added the seam vocabulary: `CodeSessionBusy` (-32003), `ReasonSessionBusy` ("session_busy"), an opt-in `Error.Reason` field and `NewSessionBusy`, with the stale invalid-request comments corrected.
- Extended `wireError` to compose `data.details` and `data.reason` without changing the unwrapped `*acp.RequestError` contract; `NewInternalWithData` and `NewInvalidRequest` carriages are pinned unchanged.
- Routed all three concurrent-prompt rejection sites (front-wide slot, machine `ErrTurnActive`/backend -32010, `/model` mid-turn busy gate) through `NewSessionBusy`; exactly three call sites, zero `NewInvalidRequest(turnActiveMessage)` left.
- Locked the shape end-to-end at the seam, the SDK-decoded `*acp.RequestError` and the raw JSON-RPC tee frame for the slot, backend-refusal, `/model` and both fan-in busy paths.
- Documented the Agent-side busy contract in the dialect doc, including the one exception to the -32603/data.details refusal rule and the Orca `retryBusyPrompts` rationale.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add the -32003 session-busy seam and its wire carriage** - `276ca96` (feat)
2. **Task 2: Route every concurrent-prompt rejection through the session-busy error** - `2193a53` (fix)
3. **Task 3: Document the agent-side session-busy contract and run the full gate** - `c1887e4` (docs)

**Plan metadata:** not committed by the executor (orchestrator Step 8 commits SUMMARY/STATE/PLAN).

_Note: TDD tasks had one commit each; the RED evidence was observed before each GREEN (busy data carriage failing on null data in Task 1; five busy assertions failing on -32600 in Task 2)._

## Files Created/Modified

- `internal/acpport/types.go` - CodeSessionBusy, ReasonSessionBusy, Error.Reason, NewSessionBusy, rewritten Error/NewInvalidRequest docs
- `internal/acpport/agent.go` - wireError composes the `details`/`reason` data payload, nil when neither is set
- `internal/acpport/agent_test.go` - TestSessionBusyWireCarriage plus the coding-table -32003 row and details/invalid-request regressions
- `internal/acpfront/front.go` - three busy sites use NewSessionBusy; stale invalid-request comments corrected
- `internal/acpfront/turnctx.go` - acquireTurn comment names the -32003 session-busy refusal
- `internal/acpfront/cancel_test.go` - shared assertRejectedSessionBusy (SDK error + raw tee frame), busy-frame locator, updated fan-in call sites
- `internal/acpfront/slashbusy_internal_test.go` - seam assertion on CodeSessionBusy/ReasonSessionBusy
- `internal/acpfront/slashmodel_test.go` - mid-turn /model switch reuses the shared wire-level busy helper
- `docs/protocol/acp-client-dialect.md` - Agent-side busy contract subsection, scoped -32603/data.details wording, retryBusyPrompts cross-reference

## Decisions Made

- The refusal message text is unchanged (byte-identical `turnActiveMessage`); only the code and data carriage changed.
- `CodeSessionBusy` lives as its own const declaration because the plan's structural verify greps the single-space form `CodeSessionBusy = -32003`; the other codes stay in the aligned group.
- The raw-frame busy assertion locates the frame by `"code":-32003` instead of the first agent-to-client error frame, because the cancel fan-in flows legitimately carry an earlier or racing `-32021` frame and first-match would be nondeterministic.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed the map capacity literal flagged by `mnd`**
- **Found during:** Task 1 (seam/wire commit, first lint run)
- **Issue:** `make(map[string]string, 2)` tripped golangci-lint `mnd` (magic number 2), which would have failed the task's lint gate.
- **Fix:** Used `make(map[string]string)` without a capacity hint (two-key map; the hint buys nothing).
- **Files modified:** `internal/acpport/agent.go`
- **Verification:** `golangci-lint run ./internal/acpport/...` -> 0 issues.
- **Committed in:** `276ca96` (Task 1 commit)

### Plan-text adjustments

- The plan named two busy call sites in `cancel_test.go`; the renamed helper had four call sites (two additional fan-in tests also assert the second-prompt refusal). All four were updated, and the raw-frame check uses the code-specific locator described above to stay deterministic. The extra two sites strengthen the wire lock rather than change scope.

---

**Total deviations:** 1 auto-fixed (blocking lint), 1 plan-text adjustment (helper call sites/locator).
**Impact on plan:** No scope creep; the delivered behavior is exactly the plan's must-haves, and the extra fan-in assertions cover the same refusal shape.

## Issues Encountered

- The first focused test run confirmed a genuine RED for Task 1: the constructor subtest and coding table passed while the busy mapping failed on `decode busy data: unexpected end of JSON input (null)`.
- Task 2's RED failed on code (`-32600` vs `-32003`) at all five busy assertions: front slot, backend -32010, both cancel fan-in tests and the mid-turn `/model` switch.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The -32003 / session_busy contract is ready for the Orca live check: the omp card should now engage `retryBusyPrompts` on a concurrent prompt instead of surfacing the refusal.
- No prompt queueing was added and cancel/stop semantics are untouched; the backend -32010 mapping and the busy message text are unchanged.

## Self-Check: PASSED

- All 9 modified files and this SUMMARY exist in the worktree.
- All 3 task commits (`276ca96`, `2193a53`, `c1887e4`) are ancestors of HEAD (`c1887e4`).
- `mise run ci` exited 0 (build, vet, lint 0 issues, race tests, experiments 24 pass / 1 pre-existing guarded skip).
- `bash experiments/check-docs.sh` exited 0.
