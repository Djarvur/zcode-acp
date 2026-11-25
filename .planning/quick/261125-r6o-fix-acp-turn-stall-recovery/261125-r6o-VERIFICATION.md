---
phase: 261125-r6o-fix-acp-turn-stall-recovery
verified: 2026-11-25T18:34:18Z
status: passed
score: 9/9 must-haves verified
covered_files:
  - .planning/quick/261125-r6o-fix-acp-turn-stall-recovery/261125-r6o-PLAN.md
  - .planning/quick/261125-r6o-fix-acp-turn-stall-recovery/261125-r6o-SUMMARY.md
  - internal/acpfront/cancel_test.go
  - internal/acpfront/front.go
  - internal/acpfront/lifecycle.go
  - internal/acpfront/lifecycle_internal_test.go
  - internal/acpfront/orphan_test.go
  - internal/acpfront/stall_test.go
  - internal/turn/cancel_test.go
  - internal/turn/machine.go
  - internal/turn/machine_test.go
  - internal/turn/orphan_test.go
  - internal/turn/stall_test.go
  - internal/turn/turn_binding_test.go
covered_digest: "v4:sha256:948192e776eeb02e0e01d9e4d15d1b48b8768f31eefb2ca6e803e64c64d4e967"
behavior_unverified: 0
overrides_applied: 0
---

# Quick Task 261125-r6o: Fix ACP Turn Stall Recovery — Verification Report

**Goal:** fix ACP turn stall recovery — bind completion to the live turn, add a no-progress watchdog, recover orphaned backend turns.
**Verified:** 2026-11-25T18:34:18Z
**Status:** passed
**HEAD:** `a5d3f34b72d1162afc561d2af4d0d80b77cd878e` on `fix/acp-turn-stall-recovery`
**Re-verification:** No — initial verification (no prior VERIFICATION.md in the task directory)

## Goal Achievement

### Observable Truths

All nine must-have truths from `261125-r6o-PLAN.md` frontmatter (`must_haves.truths`) were verified against the code, not the SUMMARY narrative.

| # | Truth | Status | Evidence |
| --- | ----- | ------ | -------- |
| 1 | A prompt accepted after a prior turn resolved cannot be resolved by that prior turn's late terminal, even while unbound; the retired turnId persists until a different one is seen. Empty-turnId terminals/turn-starts never bind. | ✓ VERIFIED | `internal/turn/machine.go:280-285` (`retiredTurnID` field, persists by design), `:801-813` (`resetTurnSignalsLocked` deliberately preserves it), `:1529-1531` (`resolveLocked` retires the bound id), `:1455-1479` (`bindTerminalTurnLocked`: empty id → false, retired id → false while unbound, record-on-first-sight otherwise), `:1490-1514` (`bindTurnStartLocked`: empty id → false). Tests: `internal/turn/turn_binding_test.go:73` `TestLatePriorTurnSignalsDoNotResolveNewTurn` (late `turn.failed(fixTurnID)` + idle rev 2 vs running rev 3 must not resolve; second turn resolves completed/quiet-window, exactly 2 resolutions, not before ack+80 ms) and `:129` `TestLatePriorTurnStartBeforeRunningPatchIsIgnored`. Both pass under `-race`. |
| 2 | A session-scoped state.updated idle patch completes a running turn only when its revision is strictly greater than the running patch's revision; missing/equal/lower falls through. | ✓ VERIFIED | `internal/turn/machine.go:1242-1267` (`classifyIdlePatchLocked`), `:1313-1324` (`armRunningPatchLocked` keeps the MAX revision), `:1330-1340` (`leavingRunningCompletesLocked` requires `hasRevision && runningRevisionSeen && obs.revision > runningRevision`), `:1282-1311` (`decodeStatePatch` reads status/reason/revision), `:1546-1569` (`paramsInt64`). Test: `internal/turn/turn_binding_test.go:201` `TestIdlePatchBindingRequiresLaterRevision` — own later revision → `SignalIdlePatch`; equal revision, absent revision, and `cancelled`/`session_stopped` while running → `SignalQuietWindow`. Passes. |
| 3 | An idle patch carrying `cancelled` or `session_stopped` while running is ignored; while cancelling the leaving-running patch remains the stop's observed outcome. | ✓ VERIFIED | `internal/turn/machine.go:1330-1332` (cancelling → true unconditionally), `:1335-1337` (running → cancelled/session_stopped → false), `ObserveEnvelope::551-563` (cancelling branch resolves `StopCancelled/SignalCancel`). Locked by `turn_binding_test.go:244-254` (ignored while running) and the green pre-existing cancel corpus locks (`TestCancelMidTurnAnswersOnce`, `TestCancelRacingTerminalResolvesOnce`, `TestCancelAfterCompleteNoOp`) in the full-package run. |
| 4 | A silent RUNNING turn past the threshold is stopped over the bypass lane, resolves `StopError/SignalStall` with a recorded error, surfaces a coded -32603 naming the stall, releases the slot, joins the watchdog on every return path; a cancelling turn never fires and never sends a second stop. | ✓ VERIFIED | `internal/turn/machine.go:109-110` (15 min / 30 min defaults), `:252-261` (public fields), `:409-416` (watchdog deferred join; watcher-error path returns before the watchdog starts), `:571-585` (`startStallWatchdog` join), `:1035-1074` (`stallLoop` exits on resolved/non-running), `:1087-1125` (budget resolution incl. 2x base tool scaling), `:1127-1174` (`fireStallIfExpired`: `m.conn.Stop` at `:1152` BEFORE `resolveLocked` at `:1168`; state/budget re-checked on both sides). Front mapping: `internal/acpfront/front.go:557-567` (`Resolution().Err` → `acpport.NewInternal`, i.e. -32603) and `:246-250` (`WithTurnStallTimeout`). Tests: `internal/turn/stall_test.go:213` `TestStallWatchdogTripsOnSilentTurn` (single stop, join proof after 2x budget), `:461` `TestStallWatchdogDoesNotFireWhileCancelling` (60 ms budget, 300 ms delayed stop, exactly one stop, `StopCancelled/SignalCancel`, no `SignalStall`), `internal/acpfront/stall_test.go:110` `TestStalledTurnSurfacesCodedErrorAndReleasesSlot` (CodeInternal -32603, no session_busy, second prompt end_turn), `internal/acpfront/lifecycle_internal_test.go:390` `TestResumedSessionStallSeam`. All pass. |
| 5 | Progress accounting counts part.delta text/reasoning, model.streaming text_delta/reasoning_delta, tool scheduled/started/progress/result/error, real session/messages content changes, turn starts/terminals; ignores poll echoes and model-request/retry telemetry; an active tool gets the larger allowance. | ✓ VERIFIED | `internal/turn/machine.go:883-1015` (`noteProgressLocked`, `sessionEventProgressLocked`, `isTurnSignalKind`, `nestedPayloadKind`, `messagesProgressLocked`), `:1099-1125` (tool allowance). Tests (all pass): `stall_test.go:246` model.streaming text_delta reset, `:272` part.delta text reset, `:299` turn-start reset (asserts fire `>= ack+150 ms`), `:329` active-tool 180 ms allowance, `:362` tool result restores the 60 ms model budget (timing + message discriminate), `:403` retry telemetry ignored, `:430` messages content change resets while the byte-identical echo does not. |
| 6 | A numeric -32010 rejection with no local turn performs exactly one stop, a bounded session-scoped clear wait, one retry; foreign/absent evidence never clears; a terminal during the wait retires its turnId so a late duplicate cannot resolve; second rejection is `ErrOrphanedBackend`, non-retryable -32603, never -32003. | ✓ VERIFIED | `internal/turn/machine.go:210-221` (`ErrOrphanedBackend`, `DefaultOrphanClearTimeout` = 5 s), `:263-274` (opt-in field + timeout), `:598-656` (`recoverOrphanedSend`: one stop, bounded wait, `resetTurnSignalsLocked`, one retry, second -32010 → `ErrOrphanedBackend`), `:662-673` (`mapSendError` preserves the orphan verdict; default-off -32010 → `ErrTurnActive`), `:675-747` (`observeOrphanEvidenceLocked`: requires `sessionId == m.sessID`; retires a non-empty terminal turnId before signaling). Front: `front.go:578-592` (`ErrOrphanedBackend` checked before `ErrTurnActive`, stable orphan phrase at `:459`). Tests (all pass): `internal/turn/orphan_test.go:145` self-heal wire order send/stop/send, `:192` second rejection is ErrOrphanedBackend and NOT ErrTurnActive, `:223` timeout still retries once, `:271` terminal evidence retires the orphan (late duplicate cannot resolve failed), `:324` foreign-session evidence does not clear early; `internal/acpfront/orphan_test.go:168` front mapping (both arms, 2 sends/1 stop); `internal/acpfront/lifecycle_internal_test.go:546` resumed arms. |
| 7 | Every machine the front registers for prompts — session/new and session/resume — is built through one shared front construction helper carrying the orphan flag and the stall/orphan seam. | ✓ VERIFIED | `internal/acpfront/front.go:891-915` (`newSessionMachine`: stall fields `:902-903`, `OrphanRecovery = true` `:911`, clear timeout `:912`), `NewSession` at `front.go:411`, `resumeSessionLocked` at `internal/acpfront/lifecycle.go:156`. No inline production `turn.NewMachine` remains (grep: only inside `newSessionMachine`). Proved on a resumed machine by `TestResumedSessionStallSeam` and `TestResumedSessionOrphanRecovery` (elapsed 35 ms..2 s proves the 40 ms override reached it; -32603 not -32003 proves the flag reached it). |
| 8 | A genuinely local concurrent turn still maps to the -32003 session_busy class (front slot and default-off machine guard unchanged). | ✓ VERIFIED | `front.go:493-500` (front-slot rejection before the machine) and `:587-589` (`ErrTurnActive` → `NewSessionBusy`); default-off machine mapping `machine.go:662-673`. Locks: `internal/acpfront/cancel_test.go:187-189` `TestSecondPromptRejected` keeps the front-slot arm only; `internal/turn/cancel_test.go` `TestDoubleSendRejectedServerSide`/`TestDoubleSendRejectedWrapperSide` (default `OrphanRecovery=false`) pass. |
| 9 | Structural debug logs record state.updated status/reason/revision and the resolution reason/signal/turnId without prompt text, deltas, or tool input/output. | ✓ VERIFIED | `internal/turn/machine.go:1252-1254` (status/reason/revision/armed_revision/state), `:1533-1534` (reason/signal/turnId/err), plus `:475`, `:616-617`, `:628`, `:720-721`, `:733`, `:743-744`, `:1152-1153`, `:1170-1171`. Field audit: session, status, reason, revision, armed_revision, state, kind, turnId, err, tool_active, budget, evidence_session only. The only grep hit for content-like words is the message text "active prompt" in the structural orphan warning. No prompt text, deltas, or tool payloads. |

**Score:** 9/9 truths verified (0 present-but-behavior-unverified).

### Parent Checklist (P0 / P1 / P2 / R1-R4)

| Item | Result | Evidence |
| ---- | ------ | -------- |
| P0 retiredTurnID persists; successor not resolved by predecessor's late terminal | PASS | machine.go:280-285, :801-813, :1529-1531, :1455-1479; turn_binding_test.go:73 |
| P0 empty-turnId terminals/turn-starts never bind/resolve | PASS | machine.go:1456-1458, :1491-1493 (`if turnID == "" { return false }` in both binders); no dedicated test exists (see residual R-RISK-3) |
| P0 idle patch strictly-greater revision; missing/equal/lower fall through; cancelled/session_stopped ignored while running; cancelling preserved | PASS | machine.go:1242-1340; turn_binding_test.go:201-262 |
| P0 structural logs only | PASS | machine.go:1252-1254, :1533-1534; field audit above |
| P1 defaults 15 min / 30 min active tool; Front/turn option seam | PASS | machine.go:109-110, :252-261; front.go:246-250, :902-903 |
| P1 running-only, cancelling carve-out | PASS | machine.go:1035-1048, :1133-1140; stall_test.go:461 |
| P1 bypass stop BEFORE resolving StopError/SignalStall | PASS | machine.go:1152 (Stop) precedes :1168 (resolve) |
| P1 front maps recorded error to coded -32603 | PASS | front.go:557-567; stall_test.go:110 |
| P1 progress scope (real vs telemetry/poll echoes) | PASS | machine.go:883-1015; stall_test.go:246/272/299/329/362/403/430 |
| P1 goroutine joined/parked on every return path; no busy re-fire while cancelling | PASS | machine.go:409-416, :571-585, :1035-1074, :1127-1174; stall_test.go:213 (join proof), :461 |
| P2 shared `newSessionMachine` used by session/new AND session/resume; both carry OrphanRecovery + seam | PASS | front.go:891-915; front.go:411; lifecycle.go:156; resumed tests pass |
| P2 one stop + bounded session-scoped clear wait + one retry on -32010 | PASS | machine.go:598-656, :675-747; orphan_test.go:145/223/271/324; front orphan_test.go:168 |
| P2 second rejection → ErrOrphanedBackend → -32603, never -32003 | PASS | machine.go:650-653; front.go:583-585; orphan_test.go:192; front orphan_test.go:203-240; resumed :547-581 |
| P2 default-off machine path and local-busy front-slot path unchanged | PASS | machine.go:612-614, :662-673; front.go:493-500; default-off locks green |
| R1 part.delta text and a turn-start alone reset the clock (outcome-discriminating) | PASS | stall_test.go:272 (`part.delta` at ~60 ms; without reset the 120 ms budget wins, terminal at ~150 ms only wins with the reset), :299 (turn-start at ~50 ms moves the 120 ms deadline; asserts fire `>= ack+150 ms`) |
| R2 cancelling under a short StallTimeout: exactly one stop, no SignalStall | PASS | stall_test.go:461-520 (60 ms budget, stop reply delayed 300 ms, `StopCancelled/SignalCancel`, stop-count == 1) |
| R3 stall loop parks/exits when the turn leaves running; join proof | PASS | machine.go:1044-1048, :1133-1140; Run's deferred join :416; stall_test.go:234-240 sleeps 2x budget then asserts one stop |
| R4 front-tier RED recorded as actually observed (successful end_turn ~7.5 s, not a deadline) | PASS (as recorded) | SUMMARY.md:198-201 and test comments `internal/acpfront/stall_test.go:104-107`, `lifecycle_internal_test.go:385-388` describe the end_turn shape. Production floor is `DefaultQuietMs 2500 x FloorMultiple 3 = 7.5 s` (watcher.go:27-28) inside `ComposeTimeout 10 s` (harness_test.go:40-42), so the shape is internally consistent. Not independently re-run (would require reverting HEAD); documentation consistency verified. |

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/turn/machine.go` | binder, idle gate, watchdog, orphan recovery, logs | ✓ VERIFIED | 1586 lines; substantive implementations at the cited lines; imported/used by `acpfront` and tests |
| `internal/turn/turn_binding_test.go` | late-terminal / idle-revision locks | ✓ VERIFIED | 262 lines; 3 tests; passes |
| `internal/turn/stall_test.go` | watchdog behavior locks | ✓ VERIFIED | 520 lines; 9 tests; passes |
| `internal/turn/orphan_test.go` | orphan-recovery locks | ✓ VERIFIED | 372 lines; 5 tests; passes |
| `internal/turn/machine_test.go` | `statePatchRev` scaffolding | ✓ VERIFIED | `statePatch` delegates with rev 1 (`:137-162`); used by all tiers |
| `internal/turn/cancel_test.go` | cancel scenario revisions | ✓ VERIFIED | cancelled patch revision 2; full package green |
| `internal/acpfront/front.go` | options, shared constructor, mapping | ✓ VERIFIED | `:240-260`, `:891-915`, `:411`, `:557-592` |
| `internal/acpfront/lifecycle.go` | resume through shared constructor | ✓ VERIFIED | `:149-156` |
| `internal/acpfront/stall_test.go` | coded stall + slot release | ✓ VERIFIED | 179 lines; passes |
| `internal/acpfront/orphan_test.go` | front orphan mapping | ✓ VERIFIED | 249 lines; passes |
| `internal/acpfront/lifecycle_internal_test.go` | resumed stall + orphan arms | ✓ VERIFIED | `:390`, `:546`; passes |
| `internal/acpfront/cancel_test.go` | supersede obsolete front lock, keep slot lock | ✓ VERIFIED | `testBackendRefusalRejection` removed (`:187-189`); `assertRejectedSessionBusy` intact (`:191-...`) |

No stub artifacts: no TODO/TBD/FIXME/placeholder markers in any changed file; no empty implementation returns in the added production code; all new tests exercise the new behavior.

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `internal/turn/machine.go` | `internal/zcode/envelope.go` | idle patch status/reason/revision decode and bind | ✓ WIRED | `decodeStatePatch` + `paramsInt64`/`paramsString` (`machine.go:1282-1311`, `:1546-1586`); `CodeConcurrency` consumed at `:608`, `:651`, `:668` |
| `internal/turn/machine.go` | `internal/zcode/call.go` | `Conn.Stop` bypass lane from watchdog and orphan recovery | ✓ WIRED | `fireStallIfExpired` `machine.go:1152`; `recoverOrphanedSend` `:627`; `Conn.Stop` enqueues on the bypass lane (`internal/zcode/call.go:196-213`) |
| `internal/acpfront/front.go` | `internal/turn/machine.go` | shared constructor injects config; `Prompt` reads `Resolution` and maps both failures | ✓ WIRED | `front.go:897-915`, `:563-564`, `:583-585` |
| `internal/acpfront/lifecycle.go` | `internal/acpfront/front.go` | `resumeSessionLocked` uses the shared constructor | ✓ WIRED | `lifecycle.go:156` |
| `internal/acpfront/stall_test.go` | `internal/acpfront/front.go` | `WithTurnStallTimeout` + composed fake leg prove -32603 and slot release | ✓ WIRED | `stall_test.go:111`; resumed arm `lifecycle_internal_test.go:391-392` |
| `internal/acpfront/orphan_test.go` | `internal/acpfront/front.go` | `WithOrphanClearTimeout` + composed fake leg prove one stop/one retry and the non-retryable mapping | ✓ WIRED | `orphan_test.go:209`; resumed arm `lifecycle_internal_test.go:549` |

### Data-Flow Trace (Level 4)

| Flow | Source | Path | Produces Real Data | Status |
| ---- | ------ | ---- | ------------------ | ------ |
| Watchdog budget | `Front.turnStallTimeout` set by `WithTurnStallTimeout` | `newSessionMachine` → `machine.StallTimeout` → `stallBudgetLocked` → `stallLoop` timer | Yes — exercised by `TestStalledTurnSurfacesCodedErrorAndReleasesSlot` and `TestResumedSessionStallSeam` (the 60 ms budget fires; default 15 min does not) | ✓ FLOWING |
| Stall cause | `fireStallIfExpired` → `Resolution.Err` | `Machine.Resolution()` → `Prompt` StopFailed/StopError arm → `acpport.NewInternal` | Yes — client test reads a `-32603` whose message names the stall | ✓ FLOWING |
| Orphan clear seam | `Front.turnOrphanClearTimeout` set by `WithOrphanClearTimeout` | `newSessionMachine` → `machine.OrphanClearTimeout` → `recoverOrphanedSend` wait | Yes — resumed test's elapsed 35 ms..2 s proves the override reached the resumed machine | ✓ FLOWING |
| Orphan flag | `Front.newSessionMachine` (front-slot proof) | `machine.OrphanRecovery` → `recoverOrphanedSend` | Yes — front tests produce 2 sends/1 stop and the orphan phrase instead of -32003 | ✓ FLOWING |

### Behavioral Spot-Checks / Commands Run

All commands run from the checkout root `/home/nil/DiskD/W/Djarvur/zcode-acp-worktrees/261125-turn-stall-recovery`.

| Command | Result | Status |
| ------- | ------ | ------ |
| `go test -race -count=1 -run '^(TestLatePriorTurnSignalsDoNotResolveNewTurn\|TestLatePriorTurnStartBeforeRunningPatchIsIgnored\|TestIdlePatchBindingRequiresLaterRevision\|TestStallWatchdogTripsOnSilentTurn\|TestStallWatchdogProgressResetsOnDelta\|TestStallWatchdogProgressResetsOnPartDelta\|TestStallWatchdogTurnStartIsProgress\|TestStallWatchdogActiveToolAllowance\|TestStallWatchdogToolCompletionRestoresBaseBudget\|TestStallWatchdogIgnoresRetryTelemetry\|TestStallWatchdogMessagesContentChangeIsProgress\|TestStallWatchdogDoesNotFireWhileCancelling\|TestOrphanedBackendSendSelfHeals\|TestOrphanRecoverySecondRejectionIsNotRetryable\|TestOrphanClearWaitTimeoutStillRetriesOnce\|TestOrphanClearWaitTerminalEvidenceRetiresOrphan\|TestOrphanClearWaitIgnoresForeignSessionEvidence\|TestDoubleSendRejectedServerSide\|TestDoubleSendRejectedWrapperSide)$' ./internal/turn` | `ok ... internal/turn 4.731s` | ✓ PASS |
| `go test -race -count=1 -run '^(TestStalledTurnSurfacesCodedErrorAndReleasesSlot\|TestResumedSessionStallSeam\|TestOrphanRecoveryFrontMapping\|TestResumedSessionOrphanRecovery\|TestSecondPromptRejected)$' ./internal/acpfront` | `ok ... internal/acpfront 1.522s` | ✓ PASS |
| `go test -race -count=1 ./internal/turn ./internal/acpfront` | `ok internal/turn 7.487s`, `ok internal/acpfront 111.524s` | ✓ PASS |
| `go build ./...` | OK | ✓ PASS |
| `go vet ./internal/turn/... ./internal/acpfront/...` | OK | ✓ PASS |
| `golangci-lint run ./internal/turn/... ./internal/acpfront/...` | `0 issues.` (one tool deprecation warning about `exhaustruct`) | ✓ PASS |
| `git show --stat 4e9a427 84a03a6 a5d3f34` + `git show --name-only \| grep '^\.planning/'` | 3 commits present; no `.planning/` path in any of them | ✓ PASS |
| `git merge-base --is-ancestor <commit> HEAD` for all three | exits 0 | ✓ PASS |
| `git diff --name-only 68b360c..HEAD` | exactly the 12 plan files; no `go.mod`/`go.sum`, corpus/golden/testdata/JSON, `internal/zcode`, `internal/acpport`, or `docs/protocol` changes | ✓ PASS |
| `git status --short` | only untracked `.planning/quick/261125-r6o-fix-acp-turn-stall-recovery/` (this report + task docs) | ✓ PASS |
| debt-marker scan (`TBD\|FIXME\|XXX`, `TODO\|HACK\|PLACEHOLDER...`) over all changed files | no hits | ✓ PASS |
| `gsd-tools query verification.fingerprint ...` | `covered_digest = v4:sha256:948192e7...` (copied into frontmatter) | ✓ PASS |

No whole-repo `go test ./...` was run (per instruction); the two changed packages plus build/vet/lint cover the delivered surface.

### Probe Execution

Not applicable — the PLAN/SUMMARY declare no probes and the repository has no `scripts/*/tests/probe-*.sh` convention involved in this task.

### Requirements Coverage

Not applicable — this is a quick task with no ROADMAP requirement IDs (`source_coverage` in the PLAN: "REQ: none"). No orphaned requirement IDs.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | No debt markers, placeholders, empty implementations, or unwired artifacts | ℹ️ Info | None |

The `return "", false` / `return obs, false` returns in `classify*` and `decodeStatePatch` are legitimate classification failures, not stubs. No hardcoded empty data reaches a user-visible surface.

### Deviation Assessment (SUMMARY claims vs code)

| SUMMARY deviation | Honest? | Assessment |
| ----------------- | ------- | ---------- |
| 1. Retired-id turn-start carve-out (a start binds only after the turn's own running patch) | Yes | Code `machine.go:1481-1514` and test `turn_binding_test.go:122-128` match the description. The must-have truth (late **terminal** of an unbound successor) is preserved and tested. The carve-out exists because the strict rule broke legacy fixtures whose turn ids are static and whose corpus files are outside the writable set. Residual risk below. |
| 2. Inert symbol scaffolding before RED | Yes | Evidence sequencing only; no behavior shipped in the scaffold step. |
| 3. Lint constant staged with Task 3 | Yes | Cosmetic; `machine_test.go:150` uses `wireKeyStatus`. |
| 4. Extra R1 turn-start test | Yes | Additive test; `stall_test.go:299`. |
| Supersession of `testBackendRefusalRejection` | Yes | `cancel_test.go:187-189` keeps the front-slot arm; `assertRejectedSessionBusy` untouched; the deleted subtest is the exact slot-free shape the orphan contract now owns. The `send-refused-32010` corpus row, golden, and fake driver are byte-untouched (`internal/zcode/fake/testdata/scenarios/send-refused-32010.json`, `internal/zcode/fake/golden_test.go:280`). |

### Residual Risks (advisory, non-blocking)

These do not fail a must-have; recording them for the ship decision.

1. **Retired-id turn-start after the successor's running patch** (`machine.go:1504`). If a backend re-announces a prior turn's `turn.started` *after* the successor's running patch (or recycles turn ids), the successor binds the retired id; a later duplicate prior terminal would then resolve it. The realistic stale shape (start before the send/running patch) is locked by `TestLatePriorTurnStartBeforeRunningPatchIsIgnored`; the field-incident shape (late prior terminal while unbound) is blocked. A follow-up could distinguish a start observed after the successor's ack by carrier/sequence.
2. **Cancel racing the stall expiry**: a Cancel landing between `fireStallIfExpired`'s pre-check and its bypass write can produce two `session/stop` frames (the cancel's and the watchdog's). The post-write state re-check prevents a wrong resolution, the extra stop is idempotent, and the must-have "no second stop after resolution" holds. The R2 test covers the ordinary (pre-budget) cancel.
3. **No dedicated empty-turnId test**: the rule is a two-line guard in both binders (`machine.go:1456-1458`, `:1491-1493`) and no corpus row drives it; a small future test would lock it explicitly.

### Human Verification Required

None. All must-have truths and residual requirements are covered by passing behavioral tests or directly inspectable structural code; no visual, real-time, external-service, or UX behavior is involved.

### Gaps Summary

None. All three scoped outcomes are implemented and test-locked: (P0) completion is bound to the live turn by a surface-level binder that retires resolved turn ids and a strictly-ordered idle gate; (P1) silent running turns are bounded by a configurable 15/30-minute no-progress watchdog that stops over the bypass lane before resolving and surfaces a coded -32603, with the goroutine joined on every return path; (P2) a slot-free -32010 heals through one stop, a bounded session-scoped clear wait, and one retry, with a second refusal mapped to a non-retryable -32603 on both session/new and session/resume machines. Build, vet, the focused tests, both full packages under `-race`, and lint are green; dependencies, corpus/goldens, and the wire vocabulary are untouched; the three commits contain no planning docs.

---

_Verified: 2026-11-25T18:34:18Z_
_Verifier: Claude (gsd-verifier)_
