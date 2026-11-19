---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
reviewed: 2026-11-20T00:00:00Z
depth: deep
files_reviewed: 15
files_reviewed_list:
  - internal/zcode/envelope.go
  - internal/zcode/writer.go
  - internal/zcode/call.go
  - internal/zcode/conn.go
  - internal/zcode/fake/fake.go
  - internal/zcode/fake/scenario.go
  - internal/zcode/fake/golden.go
  - internal/zcode/fake/stdio.go
  - internal/turn/machine.go
  - internal/turn/watcher.go
  - internal/supervisor/spawn.go
  - internal/supervisor/supervise.go
  - internal/supervisor/discover.go
  - cmd/fake-app-server/main.go
  - cmd/zcode-acp/main.go
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
  necessity: 0
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-11-20
**Depth:** deep
**Files Reviewed:** 15 (all phase production files; plus protocol dialect, E2E/real-desktop tests, corpus scenarios + goldens, and the live capture read as verification substrate)
**Status:** issues_found

## Summary

Deep review of the Phase 3 back leg: codec (two-lane writer, pending-call table, demux, bypass lane, capped line buffer, reverse-request router), scriptable fake (Serve, LoadScenario, goldens), turn machine (exactly-once resolution, PollWatcher quiet window, bypass cancel, -32010 mapping), and supervisor (discovery matrix, env allowlist, Setpgid group kill, crash window). All 15 production files read in full; call chains traced across module boundaries (Conn ↔ Machine ↔ PollWatcher ↔ Supervisor ↔ fake Serve).

**Wire-value verification (8 verbatim spot-checks against docs/protocol/app-server-dialect.md — exceeded the required 5):**
1. `session/send` params `{sessionId, content:<string>}` — machine sends exactly the shape the probe sent live (`experiments/app-server-probe.mjs:357`, verdict "accepted" in EXP-TURN-01).
2. Send ack `{accepted, sessionId, stateRevision}` — dialect §5 table verbatim in machine doc-comment, fake reply rule, and happy-turn.golden.
3. `-32010` + "A prompt is already running for this session" — dialect §6 verbatim (`CodeConcurrency` const, error-32010.json).
4. Runtime-preferences answer `{nativeSearchEnhancementsEnabled:false, memoryEnabled:false}` — dialect §3 "minimal accepted" shape, byte-identical in conn.go router and scenario expect_answer.
5. `interaction/requestCredentialCipher` params `{requestId, operation, value}` — dialect §3 verbatim in credential-cipher-scripted.json.
6. All 12 error-code constants (-32700 … -32042) match the dialect §6 table verbatim, including the 3.18.2-only -32002/-32042 drift pair.
7. String-or-int ids with type-tagged canonical keys — dialect §1 ("5" ≠ 5), correctly enforced in demux/pending table.
8. `session/stop` on the bypass lane; reverse-request replies bypass the queue — dialect §1 scheduling facts, correctly placed.

**Error-handling honesty:** verified clean. The tracked-no-answer credential-cipher path (`trackUnanswered`, conn.go:308-322) has no reply path whatsoever — it never fabricates a result and never sends -32601; its deadline expiry is recorded honestly and the machine maps it to StopError/SignalDegraded, never completed/failed (machine.go:431-449). The quiet-window fallback resolves with its honest signal label. Machine never synthesizes a StopReason on absence of evidence.

**Concurrency topology:** verified clean where it matters — one reader per pipe (conn readLoop; fake Serve scanner), one writer per sink (conn writer goroutine; fake trMu-serialized write+record), cap-1 reply buffers keep the read loops non-blocking, the fake's `wg.Add` happens-before `wg.Wait` on the same goroutine, and the orphan-answer race at stream close is explicitly handled (`drainOrphan` checks the buffer first).

**The one blocker** is in the supervisor's crash-loop exit, which skips the staged shutdown it explicitly promises — a process-group escape on exactly the path the supervisor exists for. The warnings are two Conn lifetime/liveness gaps that Phase 4 composition will inherit, one cross-session classification inconsistency in the turn machine, and a fail-loud validation gap in the scenario loader.

`necessity:` is 0; the necessity-review rubric was read at this depth and no finding qualified (every guard checked defends a reachable state or a trust boundary; no impossible-state guards, cargo-culted patterns, or single-instantiation abstractions found — the poll/subscribe strategy seam is a documented two-slot design with one shipped implementation, which is a test seam, not generality without instantiation).

## Critical Issues

### CR-01: Crash-loop exit skips the staged shutdown — process-group escape on the fail-loud path

**File:** `internal/supervisor/supervise.go:133-136`
**Issue:** In `Start`, when the crash limit trips, the code returns the crash-loop error directly from inside the `select` — **before** `s.shutdown(child, policy)` runs. This contradicts the invariant stated ten lines below (supervise.go:144-146): "Staged shutdown ALWAYS — even a crashed child's group may hold orphaned grandchildren … (PITFALL 6: never trust stdin-EOF on Electron)". Consequence: when the supervisor gives up on a crash loop, the dead child's process group is never SIGKILLed and stdin is never closed. Electron's helper processes (GPU, renderer, utility grandchildren — the entire reason for `Setpgid` + group kill) survive as orphans. The leak accumulates precisely in the scenario the supervisor exists to contain: a crashing app-server. The test suite does not cover it — `TestRestartWindow/trips loud past the limit` (supervise_test.go:187-218) asserts only the error text and spawn count, never zero-survivors on this path (the E2E proves ESRCH only on the intentional-shutdown path).
**Fix:**
```go
case info := <-child.Exited:
    ...
    crashes++
    if crashes > policy.CrashLimit {
        s.shutdown(child, policy) // group kill backstop BEFORE failing loud
        return fmt.Errorf("zcode: supervisor: app-server crash loop: %d crashes within %v: %w",
            crashes, policy.CrashWindow, info.Err)
    }
```
and add a zero-survivors assertion (kill(-pgid, 0) → ESRCH) to `testRestartWindowTrip`.

## Warnings

### WR-01: Stream death and Close() never unblock in-flight Calls — pending calls hang until the caller's own ctx expires

**File:** `internal/zcode/call.go:115-126`, `internal/zcode/writer.go:100-107`, `internal/zcode/conn.go:144-159`
**Issue:** `Conn.Call` selects only on the caller's `ctx.Done()` and the reply channel. Neither EOF on the read loop, nor `Close()`, nor a write failure unblocks it: (a) `writer.recordErr` stores `firstErr` but **nothing ever reads it** (write-only state — fake's `firstErr` is returned via `snapshot`; the writer's is dead); (b) `Close()` cancels the *base* context, which only reaches request-handler goroutines — pending calls never observe it; (c) read-loop exit (`close(readDone)`) does not fail the pending table. Failure chain for the stack's core scenario: the child crashes mid-turn → every poll Call in `PollWatcher.loop` (watcher.go:223-233) blocks until the Run caller's ctx expires → `Machine.Run` wedges with no resolution while the supervisor happily restarts a new child. With a bounded test ctx this is invisible; with a session-lifetime ctx (the natural Phase 4 shape) it is a permanent hang plus a leaked pending-table entry (`register`ed, never removed). The contract is documented (writer.go:97-99 "in-flight calls surface failure via their own ctx timeout"), so this is robustness, not incorrect behavior — but production-bound protocol code should fail fast on a dead stream. Related honesty wart: a frame enqueued successfully just before `writer.stop()` can be silently dropped (send returns nil, writer exits).
**Fix:** Close a conn-lifetime channel in `Close()` and on read-loop exit; select on it in `Call` alongside `ctx.Done()` (failing with a sentinel like `ErrConnClosed`); on read-loop exit, fail and drain all pending entries; expose `firstErr` (or drop it). 

### WR-02: Hook fields assigned after NewConn are an unsynchronized data-race window — goroutines that read them start inside NewConn

**File:** `internal/zcode/conn.go:88-106,112-137`, `internal/turn/machine.go:178-193`
**Issue:** `NewConn` starts the read loop (and the writer) before returning, but `OnNotification`, `OnRequest`, `RouterDeadline`, and `OnRouterEvent` are assigned by the caller afterwards, with no happens-before edge to the goroutines that read them. The doc-comments acknowledge the contract ("assigning while traffic is in flight is a data race"), and in practice child boot latency hides the window — but per the Go memory model this is a genuine data race whenever a frame arrives in the NewConn→assignment gap (the fake over `io.Pipe` can write instantly; a race-detector flag is a matter of timing). `NewMachine` takes over the conn's hooks the same way. An API whose correct use depends on winning a startup race is a structural footgun in the exact package whose job is goroutine discipline.
**Fix:** take the hooks in the constructor (a `ConnConfig`/options struct, or make `NewConn` not start loops and add an explicit `Start()` called after hook assignment). `NewMachine` then cannot race by construction.

### WR-03: Terminal classifier ignores sessionId — a foreign turn's terminal can resolve this turn while turnID is still unset

**File:** `internal/turn/machine.go:502-533` (vs `classifyIdlePatchLocked` at :457-467)
**Issue:** `classifyIdlePatchLocked` correctly filters `state.updated` patches by `sessionId`, but `classifyTerminalLocked` classifies on carrier method + kind + turnId only — it never checks the `sessionId` that `v4/telemetry/event` and `computer-use/operation-event` both carry (dialect §5 signal tables). `turnIDMatchesLocked` records-on-first-sight: while `m.turnID` is still `""` (send acked, our `turn.started` not yet observed), a *foreign* turn's `turn.completed` claims the id and resolves this turn StopCompleted — a fabricated completion. Today this cannot fire because D-04 maps one ACP session 1:1 per app-server child, but the machine is written as if sessions share the conn (the idle-patch sessionId check proves the concern is in scope), and Phase 4+ composition that hosts more than one session per child turns this inconsistency into a wrong-turn resolution. The fix is cheap and the asymmetry is otherwise invisible.
**Fix:** in `classifyTerminalLocked`, when the carrier's params carry a non-empty `sessionId` that differs from `m.sessID`, return not-terminal (same guard shape as `classifyIdlePatchLocked`).

### WR-04: Scenario loader accepts a reply with both result and error — the result is then silently dropped on the wire

**File:** `internal/zcode/fake/scenario.go:181-197`, `internal/zcode/fake/fake.go:370-384,354-365`
**Issue:** `Reply.validate` rejects a reply with *neither* payload but accepts one with *both* — while the struct contract says "exactly one of result or error" (scenario.go:47-48) and `writeReply`'s comment repeats "Exactly one of result or error was validated at load" (fake.go:369-370). At serve time `writeReply` prefers `Error` and silently drops the declared `Result` — a fixture typo changes the scripted wire behavior without any load-time or run-time signal, contradicting the phase's fail-loud headline ("unknown schema keys are rejections, never silent defaults", scenario.go:78-80). Same family: `unknown_method` declared with a result-only Reply silently falls back to the default -32601 (fake.go:360-362). The loader's other validations (unknown keys, empty termination, side-effect exclusivity — all covered by scenario_test.go) set a standard this rule doesn't meet.
**Fix:** in `Reply.validate`, reject `Result != nil && Error != nil` ("carries both"); in the unknown-method branch, reject a result-only `UnknownMethod` at load (`unknown_method must be an error reply`).

## Info

### IN-01: JSON `null` id silently decodes as integer id 0

**File:** `internal/zcode/envelope.go:67-83`
**Issue:** `ID.UnmarshalJSON` tries number, then string; `json.Unmarshal("null", &n)` succeeds with n=0, so a wire line with `"id":null` (outside the dialect §1 grammar: ids are string-or-int) is accepted as `ID{Int:0, IsInt:true}` and a reply to it would echo `{"id":0}` — a silently rewritten wire value in a codec whose envelope grammar is otherwise strict by construction. Harmless today (ids from the counter start at 1, so `i:0` never correlates; a null-id *request* would get a -32601 reply keyed to 0), but the honest decode is to reject the third case.
**Fix:** check `json.Unmarshal(b, &n)` against a `null` literal first (or verify `!bytes.Equal(b, []byte("null"))`) and return "zcode: id is neither number nor string".

### IN-02: Wait() blocks forever on a turn that ends without resolving

**File:** `internal/turn/machine.go:236-257,262-276`
**Issue:** `Run`'s error paths (send refused, watcher start failure) call `finishTurn()` — state goes to `done` — without closing `resolvedCh`. A concurrent `Wait` (the documented second-waiter contract: "a second wait returns the SAME StopReason") that grabbed the channel for that turn blocks until its own ctx cancels, even though the turn is over and `Run` already returned an error. The stale channel is also replaced (not closed) by the next `beginTurnLocked`. Single-owner `Run` never hits it; the Phase 4 cancel-handler pattern (cancel + Wait) can.
**Fix:** close `resolvedCh` (or a companion error channel) in `finishTurn` and have `Wait` return the run error, or document that `Wait` must not be issued before `Run` proves the turn is running (non-zero `AckAt`).

### IN-03: Cancel racing the send can put session/stop on the wire before session/send

**File:** `internal/turn/machine.go:290-308`, `internal/zcode/call.go:158-175`
**Issue:** `Run` unlocks after `beginTurnLocked` and only then enqueues `session/send` on the ordered lane; a `Cancel` arriving in that window transitions to `cancelling` and puts `session/stop` on the **bypass** lane — which by design jumps the queue (dialect §1), so the stop can reach the child before the send. The server ignores a stop for a not-yet-running prompt, the turn then runs to completion, and the machine's quiet settle (while `cancelling`) resolves StopCancelled for a turn that actually executed — a semantic lie in a microscopic window. Exactly-once is preserved either way; only the label can be wrong.
**Fix:** document the race in `Cancel`'s contract (a cancel before the send ack is best-effort), or have `Cancel` defer the stop until `ackAt` is non-zero (a tiny waiter goroutine), accepting that a pre-ack cancel then resolves completed if the send proceeds.

---

## Verified-clean notes (evidence for the absence of findings)

- **Goroutine topology** (one reader per pipe, one writer per sink): conn readLoop and writer are singletons; fake Serve serializes all outbound bytes under `trMu` covering write+record atomically (fake.go:393-417); `wg.Add` precedes `go` on the scan goroutine, `wg.Wait` follows `close(closed)` on the same goroutine — no Add/Wait race.
- **Deadlock paths**: demux never routes method-bearing frames into the pending table (WR-02 rule enforced at conn.go:192-214); reverse-request replies and stop use the bypass lane; the fake's reverse table delivers via cap-1 channels and handles the close-race explicitly (`drainOrphan`).
- **Resource leaks**: spawn failure closes all three pipe fds (`closeAll`, spawn.go:173-176); `cmd.Wait` reaps exactly once in the monitor after the stderr pump finishes (PITFALL 3 honored); `killGroup` tolerates ESRCH; `trackUnanswered`'s tracker goroutine exits on base-ctx cancel via `defer cancel()`.
- **Env allowlist**: `ChildEnv` copies exactly PATH+HOME plus the two contract vars; proxy vars cannot cross by construction; `strings.Cut` preserves values containing `=`.
- **Credentials hygiene**: logs carry method names, id keys, byte counts, durations only — never params/results (conn.go:164-176, 336-354); the cipher value in fixtures is a synthetic non-credential.
- **Real-desktop guard**: build tag `realdesktop` + `ZCODE_ACP_REAL_E2E=1` runtime guard both required (realdesktop_test.go:1-19,80-83); CI path spawns the built fake binary only.
- **Goldens**: fail-loud on missing fixtures (`-update` is explicit); semantic expect_answer compare is key-order-insensitive while transcript bytes stay deterministic for the serialized traffic the tests drive.

---

_Reviewed: 2026-11-20T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

## REVIEW COMPLETE

- Critical (BLOCKER): 1 — CR-01 supervisor crash-loop exit skips staged shutdown (process-group escape)
- Warning: 4 — WR-01 dead-stream/Close never unblocks pending Calls; WR-02 post-construction hook assignment race window; WR-03 terminal classifier ignores sessionId; WR-04 scenario loader accepts result+error replies and silently drops the result
- Info: 3 — IN-01 JSON null id decodes as int 0; IN-02 Wait hangs on a turn that ends unresolved; IN-03 cancel-before-send bypass race can mislabel the resolution
- Total: 8 findings (1 Critical / 4 Warning / 3 Info); necessity-category findings: 0
