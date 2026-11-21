---
phase: 05-interactions-full-chat-cycle-api-key
reviewed: 2026-11-21T09:30:00Z
depth: deep
files_reviewed: 42
files_reviewed_list:
  - cmd/zcode-acp/main.go
  - cmd/zcode-acp/main_test.go
  - cmd/zcode-acp/testdata/responder-e2e.json
  - experiments/README.md
  - experiments/scan-secrets.sh
  - internal/acpfront/auth.go
  - internal/acpfront/auth_test.go
  - internal/acpfront/cancel_test.go
  - internal/acpfront/doc.go
  - internal/acpfront/front.go
  - internal/acpfront/fullcycle_test.go
  - internal/acpfront/lifecycle.go
  - internal/acpfront/lifecycle_test.go
  - internal/acpfront/newsession_test.go
  - internal/acpfront/turnctx.go
  - internal/acpport/agent.go
  - internal/acpport/agent_test.go
  - internal/acpport/types.go
  - internal/interact/cipherdecrypt.go
  - internal/interact/cipherdecrypt_test.go
  - internal/interact/dedupe.go
  - internal/interact/responder.go
  - internal/interact/responder_test.go
  - internal/turn/machine.go
  - internal/turn/machine_test.go
  - internal/zcode/conn.go
  - internal/zcode/conn_test.go
  - internal/zcode/envelope.go
  - internal/zcode/fake/fake.go
  - internal/zcode/fake/golden_test.go
  - internal/zcode/fake/scenario.go
  - internal/zcode/fake/scenario_test.go
  - internal/zcode/fake/testdata/golden/full-cycle.golden
  - internal/zcode/fake/testdata/golden/newsession-orphan-close.golden
  - internal/zcode/fake/testdata/golden/reannounce-dedupe.golden
  - internal/zcode/fake/testdata/golden/session-lifecycle.golden
  - internal/zcode/fake/testdata/scenarios/full-cycle.json
  - internal/zcode/fake/testdata/scenarios/newsession-orphan-close.json
  - internal/zcode/fake/testdata/scenarios/reannounce-dedupe.json
  - internal/zcode/fake/testdata/scenarios/session-lifecycle.json
  - test/realdesktop/apicycle_test.go
  - test/realdesktop/doc.go
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
  necessity: 0
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-11-21T09:30:00Z
**Depth:** deep (cross-file: cmd ↔ acpport ↔ acpfront ↔ interact ↔ turn ↔ hostsvc ↔ zcode/fake; the seam-adjacent non-scoped files stream.go, permission.go (both packages), call.go, writer.go, readback.go, snapshot.go, and the acp-go-sdk v0.20.0 source were read as cross-reference ground truth)
**Files Reviewed:** 42
**Status:** issues_found

## Summary

The phase stack is built to an unusually high discipline standard: wire shapes are byte-locked by goldens, credential hygiene is test-locked (no key/plaintext in any log, argv, or env), every reverse request is answered exactly once under per-method deadlines, and the composed/full-cycle/E2E tiers all assert on recorded wire artifacts rather than internals. Build, vet, and the `-race` suites of all reviewed packages pass. That baseline makes the one Critical finding stand out more: the SESS-01 close teardown's `drainActiveTurn` reports the opposite of what its contract says on the successful-drain path, so the permission-bridge cancel fan-in — the whole point of that call — is skipped precisely when a turn was actually cancelled. The remaining findings are a shutdown-boundary wire invariant violation in the dedupe join posture, a dropped cwd on the resume path (broken tool-card paths on resumed sessions), an ungated close-vs-resume race that leaks machine observers, and a latent wrong-turn cancel race in Prompt's fan-in goroutine.

No necessity findings: the candidate "guards against nothing" shapes examined (the conn's tracked-no-answer router leg, the machine's SignalDegraded path, the duplicated runtime-preferences answer) are the documented unwired-responder fallback layer with real in-repo instantiations, not phantom requirements.

## Critical Issues

### CR-01: drainActiveTurn returns false after a successful drain — close skips the permission-bridge cancel fan-in on exactly the path it exists for

**File:** `internal/acpfront/lifecycle.go:178-213` (caller at `internal/acpfront/lifecycle.go:131-136`)
**Issue:** The function's contract says "It reports whether a turn of THIS session was actually cancelled, so the caller only cancels permission bridges that belong to it." The implementation returns the opposite on the normal path. Trace: iteration 1 finds the active turn (`done = f.turnDone`), cancels, and waits on `done`. `done` closes only inside `release()` after `f.turnActive` was already set false under `turnMu` (turnctx.go:427-440, `close(done)` at :460). The loop then re-enters, finds `done == nil`, and returns **false**. The only `return true` is the deadline-elapsed failure path (turn still active past `closeTurnWait`). Consequently in `CloseSession`:

```go
if f.drainActiveTurn(sessionID, st) {
    f.perm.cancelPendingForTurn()
}
```

the pending permission bridges are cancelled ONLY when the drain timed out. In the normal case — a turn actively holding a pending `interaction/requestPermission` bridge when the client calls `session/close` — the turn is cancelled and drained, the backend session is closed, and the bridge's pending entry dangles: the round-trip keeps waiting on the ACP client for up to the responder's 300 s permission deadline (`interact.DefaultPermissionDeadline`), after which a deny is framed at an already-closed backend session. This is precisely the Pitfall-2 / T-5-05 dangling-pending shape the fan-in exists to prevent (the doc comment on this very call site says so), and the `Front.Cancel` notification path performs the same fan-in unconditionally, so the asymmetry is not deliberate. No test covers close-with-pending-permission (`lifecycle_test.go`'s close tests run turns with no permission reverse), so nothing catches it.

**Fix:** Return whether an active turn of this session was observed and cancelled, not whether the drain timed out:

```go
func (f *Front) drainActiveTurn(sessionID string, st core.Session) bool {
	deadline := time.Now().Add(closeTurnWait)
	sawTurn := false

	for {
		f.turnMu.Lock()

		var done chan struct{}
		if f.turnActive && f.turnSession == sessionID {
			done = f.turnDone
		}

		f.turnMu.Unlock()

		if done == nil {
			return sawTurn // false = nothing ever ran; true = cancelled and drained
		}

		sawTurn = true
		st.Machine.Cancel()

		remain := time.Until(deadline)
		if remain <= 0 {
			slog.Warn("acpfront: close turn wait elapsed; proceeding best-effort", "session", sessionID)
			return true
		}

		select {
		case <-done:
		case <-time.After(remain):
		}
	}
}
```

Add a lock: close a session whose active turn holds a pending permission ask (the corpus tool rows already script permission reverses mid-turn) and assert `front` cancelled the pendings (e.g. via the bridge's pending count or a -32021 frame on the fake transcript).

## Warnings

### WR-01: Parked dedupe-join arm frames a bare id-echo reply at conn close — the "we simply never send them" invariant is not upheld

**File:** `internal/interact/responder.go:225-249` (with `internal/zcode/conn.go:348-359`, `internal/zcode/call.go:219-236`, `internal/zcode/conn.go:180-196`)
**Issue:** When a re-announce joins a pending dedupe entry, `Serve` parks on `<-ctx.Done()` and then returns `(nil, nil, true)` — handled. The per-request ctx only cancels at `Conn.Close` (`cancelBase` runs before `rw.Close` and `out.stop`). `handleRequest` then takes the `handled` branch: `c.replyRequest(env.ID, nil, nil)` → `Reply(id, nil)` — and `Reply` with a nil result marshals `{"id":X}` with no payload (call.go:220-231: `result != nil` gates `env.Result`). In the window between `cancelBase()` and `rw.Close()` the writer goroutine is still alive, so the bare frame can be written to the still-open stream — exactly the payload-less frame the Serve comment says must never be framed ("a frame of ANY shape to a still-registered id would reject the pending ask — the server result-schema-parses reply payloads … Silence is the only safe join posture: we simply never send them"). The golden corpus never records the frame because the race is usually lost to `rw.Close`, which makes it worse, not better: it is a nondeterministic wire-garbage window at every conn close that has pending joins. Mostly benign at shutdown, but the stated invariant is false, and the wrapper's staged shutdown routinely closes the conn while reverse sets may still be registered.

**Fix:** Make the join arm's silence structural instead of relying on shutdown racing: return a distinguishable outcome for joins (e.g. a sentinel wire error that `replyRequest` drops, or have `Serve` signal "handled, do not reply" via a package-level `noReply` marker the conn skips), or in `replyRequest` skip the reply when both result and werr are nil for an intercepted request. Then the park can also shrink: park on a done channel closed when the primary's entry tombstones rather than on conn close.

### WR-02: ResumeSession drops the client cwd — tool diff/location paths stay relative on resumed sessions (Pitfall 5 on the resume path)

**File:** `internal/acpport/agent.go:375-384`, `internal/acpfront/lifecycle.go:55-99` (effect at `internal/acpfront/stream.go:361-376`)
**Issue:** The SDK's `ResumeSessionRequest` carries `Cwd` ("The working directory for this session. Must be an absolute path." — acp-go-sdk v0.20.0 types_gen.go:8346), and the harness tests even pass it (`lifecycle_test.go:121-124`, `fullcycle_test.go:266-269`). The adapter forwards only the session id (`Handlers.ResumeSession(ctx, sessionID)`), so `Front.ResumeSession` never records the cwd: no `f.cwds[sid]` entry is made (only `NewSession` calls `recordCwd`). On the resumed session's first turn, `acquireTurn`'s `sessionCwdLocked` returns "" and the tap's `resolveAbs` passes relative backend paths through unchanged — an Edit `tool.updated` with `input.file_path: "src/main.go"` (the full-cycle corpus shape, scenarios/full-cycle.json:75) renders a client card with a relative diff path and location, the exact broken-card failure Pitfall 5 documents. The stream.go doc dismisses the unknown-cwd path as non-production ("the front's session/new validates an absolute cwd, so the known-cwd path is the production path") — but resume has been a production handler since plan 05-05, so the assumption lapsed. The new-session cwd validation (T-4-01) is also skipped on resume: the SDK contract makes resume `cwd` MUST-be-absolute too.

**Fix:** Widen the seam: `Handlers.ResumeSession(ctx, sessionID, cwd string)` (or a params struct), validate `filepath.IsAbs(cwd)` before the backend call exactly like `NewSession`, and `f.recordCwd(sid, cwd)` once `createSessionID` confirms the reattached id. Update `agent.go` translation to pass `req.Cwd` through.

### WR-03: session/resume is not gated on the closing set — close-vs-resume race leaks a machine's conn observers

**File:** `internal/acpfront/lifecycle.go:55-99` (ungated resume) vs `internal/acpfront/lifecycle.go:103-168` (close), gate precedent at `internal/acpfront/front.go:318-323` and `:384-387`
**Issue:** The SESS-01 race gate (`f.closing`) deterministically orders close-vs-**prompt**, but `ResumeSession` never checks `isClosing`. The SDK dispatcher serves requests concurrently, so `session/resume` can interleave with `session/close`: if resume passes its `registry.Get` reattach check (old machine found) and registers its NEW machine after close already snapshotted `st` and ran `f.registry.Remove(sessionID)`, then close's `st.Machine.CloseObservers()` removes only the OLD machine's two fan-out entries — the new machine's notification + router observer entries are never removed (the WR-07 discipline this phase introduced), and the registry is left holding a machine for a session that was torn down (a zombie until the next close's Get gate rejects... except the closing flag was already deleted, so the zombie is silently closable again — after leaking its predecessor's registration). Ordering the other way, resume can also re-register mid-teardown between close's step 2 and step 3 with the same leak.

**Fix:** Gate resume like prompt: at `ResumeSession` entry (and ideally again just before `Register`), `if f.isClosing(sessionID) { return "", acpport.NewInvalidRequest("session/resume: %s: the session is closing or closed", sessionID) }`. Optionally re-check `f.registry.Get` under the closing set's protection so the reattach path and the teardown share one ordering.

### WR-04: Prompt's cancel fan-in goroutine can cancel a later, unrelated turn on the same machine

**File:** `internal/acpfront/front.go:333-340`
**Issue:** The fan-in goroutine selects `{sdkCtx.Done → st.Machine.Cancel; turnCtx.Done → return}` and is only reaped when the turn context cancels. When the client abandons the prompt request (SDK cancels `sdkCtx` — the displaced-prompt case `cancel_test.go:203-206` documents deliberately), both channels can be ready at the goroutine's wake: `select` picks randomly, and the goroutine may fire `st.Machine.Cancel()` unsynchronized with `release()`. If the goroutine is descheduled past the turn boundary — turn 1 resolved, `release()` ran, the front slot was reacquired, and the machine's `beginTurnLocked` reset `resolved=false` for turn 2 — the stale `Cancel()` lands on turn 2 and resolves a prompt the client never cancelled as `cancelled`. The exactly-once machinery makes the wrong resolution indistinguishable from a real one on the wire. The window is narrow (goroutine descheduled across release + next acquisition) but the failure is silent and the mechanism is already load-bearing for the displaced-prompt behavior, so the dark side deserves a guard rather than a lucky scheduler.

**Fix:** Bind the cancel to the turn it belongs to: capture the slot identity for this turn (e.g. the pump pointer or a per-turn token) and have the `sdkCtx.Done` arm re-check `f.turnPump == pump` under `turnMu` before calling `st.Machine.Cancel()` (skip silently when release already ran), or select `machine.Cancel` through the same once-channel release closes.

## Info

### IN-01: realdesktop override silently ignores a malformed ZCODE_ACP_MODEL — production fails loud on the same shape

**File:** `test/realdesktop/apicycle_test.go:239-249`
**Issue:** `resolveModelSelection` drops a `ZCODE_ACP_MODEL` value that does not split into non-empty `providerId/modelId` and keeps the corpus default, while production `parseConfig` (cmd/zcode-acp/main.go:151-154) exits 2 on exactly that shape. On the guarded real tier this means a typo'd override silently runs the ledgered, quota-spending cycle against the default model instead of failing the run — the spend happens, on an unintended model, with the ledger row attributing it to the wrong selection.
**Fix:** Fail the test (`t.Fatalf`) on a non-empty override that does not parse, mirroring `errModelShape`.

### IN-02: raw-capture path is not enforced to be outside the repo — the "born OUTSIDE" ground rule lives only in comments

**File:** `test/realdesktop/apicycle_test.go:508-541`
**Issue:** `openCapture` accepts any `ZCODE_ACP_RAW_CAPTURE` path and writes the unscrubbed wire tee there at 0o600. The tee records every back-leg byte, including the api key inside the runtime-headers reply and any credential-blob traffic — the file is live credential material by construction. The Phase-1 ground rule ("raw captures are born OUTSIDE the repo", restated in the file header and experiments/README.md) is enforced nowhere in code; a repo-internal path would plant unscrubbed credentials in the working tree with only `scan-secrets.sh` as a downstream commit-time guard.
**Fix:** Before opening, resolve the path and reject one inside the git toplevel (or require a parent-directory marker), failing loud with the ground-rule text.

### IN-03: shutdown select can mask a child crash that races the client disconnect

**File:** `cmd/zcode-acp/main.go:307-314`
**Issue:** `select { case <-agentConn.Done(): ...; case <-child.Done: childLost = true }` picks randomly when both channels are ready. A child that dies at the same moment the ACP client disconnects can lose the race, `childLost` stays false, and `run` returns nil — clean exit 0 — even though the supervised child crashed instead of being torn down. The staged shutdown still runs correctly either way; only the exit verdict is nondeterministic.
**Fix:** After the select, re-check the child with a short bounded drain (e.g. `select { case <-child.Done: childLost = true; default: }` after the staged teardown) so the crash verdict does not depend on which channel won.

---

_Reviewed: 2026-11-21T09:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
