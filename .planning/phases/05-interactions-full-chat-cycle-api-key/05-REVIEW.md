---
phase: 05-interactions-full-chat-cycle-api-key
reviewed: 2026-11-21T21:40:00Z
depth: deep
files_reviewed: 23
files_reviewed_list:
  - internal/acpfront/lifecycle.go
  - internal/acpfront/front.go
  - internal/acpfront/auth.go
  - internal/acpport/agent.go
  - internal/acpport/types.go
  - internal/zcode/conn.go
  - internal/interact/responder.go
  - internal/hostsvc/readback.go
  - internal/zcode/fake/fake.go
  - internal/zcode/fake/scenario.go
  - internal/zcode/fake/golden_test.go
  - internal/zcode/fake/scenario_test.go
  - internal/zcode/fake/testdata/scenarios/lingering-latch-32010.json
  - internal/zcode/fake/testdata/golden/lingering-latch-32010.golden
  - internal/acpfront/auth_test.go
  - internal/acpfront/cancel_test.go
  - internal/acpfront/lifecycle_test.go
  - internal/acpport/agent_test.go
  - internal/zcode/conn_test.go
  - internal/hostsvc/hostsvc_test.go
  - test/realdesktop/apicycle_test.go
  - test/realdesktop/readbackdiag_test.go
  - experiments/README.md
findings:
  critical: 0
  warning: 3
  info: 4
  total: 7
  necessity: 0
status: issues_found
---

# Phase 5: Code Review Report (delta re-review after gap closure)

**Reviewed:** 2026-11-21T21:40:00Z
**Depth:** deep (delta since af5a1ed — the prior 05-REVIEW.md commit; cross-reference ground truth re-read: turnctx.go acquireTurn/release slot discipline, permission.go pending table, turn/machine.go -32010→ErrTurnActive mapping, zcode/call.go Reply framing, conn.go readLoop/demux observer dispatch, responder.go policy/expiry table)
**Files Reviewed:** 23
**Status:** issues_found

## Summary

Delta re-review of the 05-10 gap-closure fixes (CR-01, WR-01..04), the 05-11 probe-create read-back arm, and the 05-13 turn-latch drain (fake engine + guarded real-tier drain). First-hand verification: `go build ./...` and `go vet ./...` clean; `go test -race -count=1` passes for acpfront, acpport, zcode, zcode/fake, interact, hostsvc, and turn; the golden corpus (including the new lingering-latch row) passes 3 consecutive runs; `experiments/scan-secrets.sh` exits 0 over the tree including the three new captures.

**Focus 1 — do the fixes close CR-01/WR-01..04?** CR-01, WR-01, WR-02, and WR-04 are genuinely closed, each with a wire-artifact or structural-discriminator lock (details below). WR-03 is narrowed, not closed: the entry gate plus the pre-Register re-check leave two reachable interleavings that still leak the new machine's observers / re-register a zombie after close — raised as WR-05, and the fix's own doc comment ("no machine observer leak, no zombie registration", lifecycle.go:61-62) overclaims. The WR-04 fix itself is sound — the pump-pointer identity re-check cannot be defeated by allocator address reuse because the fan-in goroutine holds the pump reference alive, and release() nils the slot under the same lock — but the same plan introduced a new observe-then-sweep shape in `Front.Cancel` with a residual stale-sweep race (WR-07), and the delta's new tests enshrine a pre-existing dangling-bridge gap on the displaced-prompt arm as the expected artifact (WR-06). No regression to the prior findings' own machinery; no new deadlock (the only nested locking is turnMu→cwdMu and turnMu→closingMu in acquireTurn, both leaf acquisitions; close marks closingMu *before* taking turnMu, so the orders never invert).

**Focus 2 — probe-create honesty.** The honesty contract holds. `Applied()` gates only the probe's *necessity* (skip when a patch already latched); the verdict remains `watcher.Await`. The probe feeds `watcher.OfferCreateReply` with the backend's own create-reply bytes, which pass `considerSurface`'s want-models check (`containsAllModels`) — a registry that silently discarded the push yields a reply without the want models, so nothing latches and Await surfaces the coded negative. The latch is write-once (`considerSurface` returns early when `applied` is set, readback.go:146-148), so a probe racing a late patch cannot flap or overwrite the verdict. Every probe failure (temp dir, refused create, no session id, failed close) logs structurally and skips its step; a cancelled ctx surfaces through Await's coded error with no success recorded. The gate cannot fabricate success by construction.

**Focus 3 — latch drain honesty.** The real-tier drain keeps the honest ACP surface: retries only the invalid-request class matched via `errors.As` on the code (never message text, T-5-32); -32603 and every other class fails the leg immediately; the retry loop is bounded (8 attempts, 120 s budget, capped backoff) and exhaustion returns the loud hypothesis-(b) sentinel instead of a pass; the ask-tap logs method name + UTC timestamp only, never params (T-5-33 held — the cipher asks carry encrypted credential bytes). The fake engine models the refusal without consuming rule budget (scenario validation fail-loud on latch-without-refusal and negative latch_ms), and the golden locks the full refuse→drain→accept transcript. Attempt-3 evidence (experiments/README.md, EXP-APIKEY-CYCLE-03) records the drain working on the real desktop with the settle budget exhausting honestly — no fabricated pass anywhere in the chain.

**Focus 4 — new findings.** Three warnings and four infos, below. No necessity findings: the candidate "guards against nothing" shapes in the delta were checked and are all reachable — the second `drainActiveTurn` call handles the deadline-arm leftover turn (the turn can genuinely still be active there), the Cancel slot gate has a proven reachable skip state (TestCancelFanInOnlyWhenSlotHeld constructs it), the `Applied()` gate has both arms proven, and `WithCloseTurnWait` is a documented test seam (negative space, not necessity). machine.go was not touched in this delta, consistent with the scoped-production-diff gate.

Closure verification of the prior findings:

- **CR-01 — closed.** `drainActiveTurn` now reports `sawTurn` (true on both the successful-drain and deadline-elapsed arms, false only when no turn of the session was ever observed — lifecycle.go:221-260), and `CloseSession` fans the permission pendings in on exactly that signal (lifecycle.go:170-175). `TestCloseWithPendingPermissionCancelled` locks the -32021 cancelled reply with exactly-one-reply and the stop-before-close teardown order; `TestCloseDeadlineArmFansPendingIn` locks the deadline arm via the 1 ns seam. Idle close still returns false (the slot-free observation is the first loop iteration).
- **WR-01 — closed.** The bare id-echo is now structurally impossible: `replyRequest` drops a neither-result-nor-error answer before any frame can be built (conn.go:501-510), independent of the cancelBase→rw.Close window. `TestReplyStructuralSilence` locks the drop plus both genuine arms through the real handleRequest→replyRequest path.
- **WR-02 — closed.** The seam is widened end-to-end: `Handlers.ResumeSession(ctx, sessionID, cwd)` (types.go), verbatim `params.Cwd` pass-through (agent.go), absolute-cwd validation before any backend call and `recordCwd` on confirmation (lifecycle.go:72-75, :101). `TestResumeRecordsCwd` locks the resumed-cwd diff path, `TestResumeRejectsRelativeCwd` locks the coded rejection plus the zero-wire-frame parity arm, and the adapter tests lock the pass-through.
- **WR-03 — partially closed; see WR-05.**
- **WR-04 — closed for the Prompt fan-in** (pump identity re-check, front.go:355-371; `TestCancelFanInStaleArmSkips` locks later-turn containment end-to-end over the exactly-one-session/stop transcript artifact). The Cancel-side gate it introduced carries WR-07 below, and the delta's test tier surfaces WR-06.

## Warnings

### WR-05: Resume's closing gate is TOCTOU — two reachable interleavings still leak machine observers / register a zombie after close

**File:** `internal/acpfront/lifecycle.go:118-133` (re-check + Register vs CloseSession's `internal/acpfront/lifecycle.go:143-165` Get→mark window and `:197-203` teardown)
**Issue:** The WR-03 fix gates resume at entry (:81-83) and re-checks just before Register (:126-128), which deterministically rejects a resume that arrives while the closing set is marked. But both checks are unsynchronized snapshots against close's own ordering, and close's teardown spans network round trips. Two interleavings survive:

1. **Close completes entirely between resume's entry gate and its re-check.** Resume enters (not closing), issues its two backend round trips (session/resume + subscribe — seconds against a real desktop); close for the same session starts, marks, drains, closes backend, `registry.Remove`, `CloseObservers`, and *deletes the closing entry* — all inside resume's round-trip window. Resume's re-check then reads `closing == false` (already deleted) and Registers a machine whose session is closed backend-side: zombie registration plus notification/router observer entries nothing will remove (the registry entry is gone, so no later close can find them; the leaked machine's fan-out entries live until process end).
2. **Register lands inside close's Get→mark gap** (lifecycle.go:143-148, a few instructions): close snapshots the *old* machine at Get; resume closes the old machine's observers and Registers the new one; close then removes the registry entry and closes the *old* machine's observers again — the new machine's conn observers leak exactly as pre-fix.

The doc comment's claim — "no machine observer leak, no zombie registration" (lifecycle.go:61-62) — is therefore not true in general; the fix shrank the windows but did not make the ordering deterministic the way the prompt gate is (acquireTurn re-checks `isClosing` *under turnMu*, so close's mark-then-drain strictly orders against it; resume's Register has no such shared lock). Both interleavings require a client racing close against resume of the same session — the exact race class T-5-12/WR-03 declared owned — and the second is fully local to this file's instruction ordering.

**Fix:** Make the ordering deterministic with the locks that already exist:

- In `ResumeSession`, hold `closingMu` across the pre-Register re-check **and** the `registry.Register` call, and inside that critical section also require the registry entry to still be present (`f.registry.Get(sessionID)` succeeds) — close's `Remove` ran otherwise, so the reattach target is gone and the resume must take the coded rejection.
- In `CloseSession`, mark the closing set *before* `registry.Get` (unmark and return the Get failure if absent). Then any Register that precedes the mark is visible to close's Get — close tears down the machine resume actually registered — and any Register after the mark is excluded by the re-check, whose critical section the mark cannot enter.

Soften the lifecycle.go:60-62 doc to match ("a resume racing a close is ordered by the closing set; the Register–mark mutual exclusion makes the ordering deterministic") rather than claiming leak-impossibility the pre-fix comment shape can't support.

### WR-06: Displaced-prompt cancel arm resolves the turn but never fans its pending permission bridge in — the bridge dangles to the 300 s deadline and answers deny for a cancelled turn

**File:** `internal/acpfront/front.go:355-371` (the fan-in goroutine's `mine` arm), contract at `internal/acpfront/front.go:420-427`; enshrined as expected behavior by `internal/acpfront/cancel_test.go:398-465` (`TestCancelFanInOnlyWhenSlotHeld` + `assertDanglingAskNote`)
**Issue:** When the SDK displaces an active prompt (the client's second prompt registration cancels the first request's ctx — the case cancel_test.go documents deliberately), the fan-in goroutine's `sdkCtx.Done` arm re-checks the slot, sees this turn still holds it, and fires `st.Machine.Cancel()` only. No `cancelPendingForTurn` runs on this arm, and `release()` does not fan pendings in either (turnctx.go:427-461; the only fan-in call sites are Front.Cancel front.go:446 and CloseSession lifecycle.go:174). The cancelled turn's in-flight `interaction/requestPermission` round-trip therefore rides the responder's per-method deadline — `DefaultPermissionDeadline` = 300 s (responder.go:41-43), not the fake's 500 ms — with the backend holding the ask and re-announcing (the T-5-05 harm the TOOL-02 contract at front.go:420-427 explicitly forbids: "a cancelled turn leaves no pending bridge silently dangling while the server re-announces forever"), and at expiry the responder answers **deny** (responder.go:361-362) for a turn that was cancelled. The Cancel and close paths both fan pendings in; the displacement path is the one remaining arm that violates the contract. `TestCancelFanInOnlyWhenSlotHeld` locks the dangling shape as the *expected* wire artifact ("the ask was never cancelled by anything in this flow" + `assertDanglingAskNote`), which converts the gap into enshrined behavior the fixer would otherwise preserve. The dangling state was pre-existing, but this delta reshaped the arm (WR-04) and added the test that pins it.

**Fix:** Fan the pendings in on the same arm, mirroring Front.Cancel — the slot re-check already guarantees the table belongs to this turn when `mine` is true:

```go
if mine {
    st.Machine.Cancel() // idempotent (Phase 3: repeats no-op)
    f.perm.cancelPendingForTurn() // TOOL-02: the cancelled turn's bridge fans in (→ -32021)
}
```

Safe for the normal-completion race (post-release wake finds the slot nil/next-turn and skips both; a wake before release hits an already-resolved machine — idempotent — and sweeps only this turn's already-removed or legitimately-orphaned entries). Then update `TestCancelFanInOnlyWhenSlotHeld`'s expectation: the dangling-ask note asserts the *slot-free* cancel skips the fan-in, which stays correct; add a displacement-with-pending case asserting the -32021 reply, replacing the dangling-note artifact for the displaced turn.

### WR-07: Cancel's slot-gate snapshot can sweep a *later* turn's pendings after a deschedule — the observe-before-cancel shape fixed the miss but left the sweep unguarded

**File:** `internal/acpfront/front.go:436-449`
**Issue:** The gate observes `slotHeld` before `st.Machine.Cancel()` so a cancel that itself resolves the turn still fans its pendings in (the comment at :436-439 says exactly this). But the fan-in at :445-447 runs unconditionally on the stale snapshot. Reachable interleave: `slotHeld = true` (session A's turn active) → `machine.Cancel()` fires the bypass cancel → the handler goroutine is descheduled → A's stop arrives, the turn resolves, `release()` frees the slot, a **session B** prompt acquires it and registers a permission pending → the handler resumes and calls `cancelPendingForTurn()`, which sweeps the front-wide table — cancelling B's fresh ask with -32021 while B's turn continues. The backend sees its brand-new permission ask rejected as cancelled; the wrong outcome is silent and wire-indistinguishable from a real client decision. This is the same narrow-deschedule shape WR-04 fixed for the Prompt fan-in (with an identity re-check); the Cancel path got a pre-observation instead of a guard at the sweep, so the class is only half-closed. The window needs a deschedule spanning a full turn unwind plus a next-turn ask registration — narrow, but the failure is silent and the phase's own WR-04 rationale ("the dark side deserves a guard rather than a lucky scheduler") applies verbatim.

**Fix:** Scope the pending table by session instead of sweeping the front-wide table on a stale snapshot: stamp each `addPending` entry with the ask's session id (`PermissionAsk` carries it) and change the fan-in to `cancelPendingForSession(sessionID)`, cancelling only entries of the cancelled turn's session. That kills the race deterministically on all three call sites (Cancel, CloseSession, and the WR-06 arm) — `drainActiveTurn`'s sawTurn semantics already assume "the pending table belongs to the slot-holder's turn", and session scoping is what makes that assumption true at fire time rather than at observe time.

## Info

### IN-04: Probe-create read-back is single-shot before Await — a late-applying push on a patchless desktop can never latch

**File:** `internal/acpfront/auth.go:160-166` (gate), `:193-230` (probe)
**Issue:** `probeCreateReadback` runs once, before `Await`. On the field-verified 3.18.2 shape — no `state.updated` patch ever fires for push+refresh — the only latch surface is a create reply, and only the probe's. If the desktop applies the pushed registry *after* the probe's create reply was rendered (async application slower than the push→refresh→probe round trips), the probe reply lacks the want models, nothing latches, and Await burns its full 30 s window to the honest coded failure even though the application succeeded a moment later. The client can retry authenticate, and attempt-3 field evidence shows the probe surfacing the models in practice, so this is a latency-shape robustness gap, not a correctness one — the failure mode is honest, never a fabricated pass.
**Fix:** Re-offer the probe inside the wait — e.g. run the probe at the Await poll cadence (a bounded number of probe-creates across the read-back window), or delay the first probe by a short grace period so a slow async application lands inside the reply. Even two probe attempts (t≈0 and t≈half-window) would close the observed-latency band without changing the verdict surface.

### IN-05: Fake latch arms *after* the reply write — a dispatch race can let the refused send slip ahead of the arm and consume the next rule (loud golden flake)

**File:** `internal/zcode/fake/fake.go:197-212`
**Issue:** `handleRequest` arms the latch only after `writeReply` returns (`:209-212`), while request handlers are one goroutine per dispatch (`go s.handleRequest(env)`). The golden driver's send-2 can therefore, in principle, complete its whole round trip (reply wakes the driver, `waitStatePatch("idle")` returns immediately — the patches were emitted before the reply — and send-2 is written and dispatched) before send-1's handler executes `armLatch`. In that interleaving `latchArmed()` reads false, send-2 consumes the second send rule, and the golden mismatches. The failure is loud (byte-diff), not a false pass, and the window is two instructions versus two goroutine handoffs plus a pipe write — three consecutive corpus runs pass — but the test's own comment claims byte-determinism, and a loaded CI runner could flake it.
**Fix:** Arm before writing the completion reply (`armLatch` then `writeReply`) — the window semantics shift by microseconds (the desktop arms at completion; the reply *is* the completion), the golden's 2500 ms settle dwarfs it, and the ordering becomes race-free by construction.

### IN-06: realdesktop ask tap logs from conn goroutines with `t.Logf` — a mid-traffic failure can panic after the test completes

**File:** `test/realdesktop/apicycle_test.go:731-746` (`tapInteractionAsks`'s closure)
**Issue:** The tap calls `t.Logf` on the conn's request-handler goroutines. `testing.T` panics ("Log in goroutine after test has finished") when called after the test completes, and a panic in a non-test goroutine kills the whole test binary. On a `t.Fatalf` mid-turn (e.g. `toolTurnLeg` failing while an interaction ask is still being served — exactly the latch-failure scenario this drain was built for), the defers close the conn but an already-dispatched handler can still be inside the tap when the test ends. The original failure message is already printed, so the harm is crash noise over the failure evidence, on a manual-only tier — but this is the diagnostic path whose failure output matters most. (`readbackdiag_test.go`'s `diagObserver` has the same shape but removes the observer LIFO before return, so it is bounded by the remove; the OnRequest tap has no removal seam.)
**Fix:** Replace `t.Logf` in the tap with an append to a mutex-guarded slice of `(method, time)` records and log from the test goroutine (the recorder already exists — extend `askRecorder`), or guard with `sync/atomic` + a done flag checked before logging.

### IN-07: readbackdiag's capture opener repeats the IN-02 class — raw-credential path still not enforced outside the repo

**File:** `test/realdesktop/readbackdiag_test.go:323-356` (`openDiagCapture`, `os.OpenFile` at `:333`)
**Issue:** The diagnostic's raw tee holds the same live credential material as apicycle's (`O_TRUNC` at any env-given path, 0o600), and the "born OUTSIDE the repo" ground rule is again comment-only: nothing rejects a repo-internal `ZCODE_ACP_RAW_CAPTURE`. IN-02 (open, disposition ledger) covers apicycle's `openCapture`; this delta adds a second instance of the class, so the eventual IN-02 fix must cover both call sites or the gap survives.
**Fix:** Share one guarded opener for both instruments: resolve the path and reject anything inside the git toplevel (or require a parent-dir marker) with the ground-rule text in the failure, before the first byte is written.

---

_Reviewed: 2026-11-21T21:40:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep (delta re-review; prior findings CR-01, WR-01..04, IN-01..03 — CR/WR closure verified above, IN-01..03 remain open in the disposition ledger outside this delta's scope)_
