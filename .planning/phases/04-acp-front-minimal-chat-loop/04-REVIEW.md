---
phase: 04-acp-front-minimal-chat-loop
reviewed: 2026-11-20T06:20:00Z
depth: deep
files_reviewed: 20
files_reviewed_list:
  - internal/acpport/types.go
  - internal/acpport/agent.go
  - internal/acpport/sink.go
  - internal/acpport/doc.go
  - internal/acpport/pin_test.go
  - internal/acpport/agent_test.go
  - internal/acpfront/front.go
  - internal/acpfront/stream.go
  - internal/acpfront/turnctx.go
  - internal/acpfront/doc.go
  - internal/acpfront/harness_test.go
  - internal/acpfront/loop_test.go
  - internal/acpfront/cancel_test.go
  - internal/acpfront/stream_order_test.go
  - internal/core/registry.go
  - internal/core/registry_test.go
  - internal/zcode/conn.go
  - cmd/zcode-acp/main.go
  - cmd/zcode-acp/main_test.go
  - internal/zcode/fake/testdata/ (scenarios + goldens: part-delta-streaming, complete-then-cancel, completion-quiet-window, setmodel-followup, cancel-mid-turn, send-refused-32010, no-reasoning)
findings:
  critical: 0
  warning: 7
  info: 5
  total: 12
  necessity: 1
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-11-20T06:20:00Z
**Depth:** deep (cross-file: acpport ↔ acpfront ↔ core ↔ turn ↔ zcode ↔ cmd, plus the SDK v0.20.0 source read directly from the module cache as wire ground truth)
**Files Reviewed:** 20
**Status:** issues_found

## Summary

The Phase 4 composition is substantially sound. The contracts the phase was charged with hold and are wire-locked by tests I re-ran myself (`go test -race` green across acpport/acpfront/core/zcode/turn; gofmt and go vet clean):

- **Cancelled never -32800** is structurally correct, not incidental: the turn context is front-owned (turnctx.go derives from the front base ctx, never the SDK request ctx), cancellation reaches the machine only through the idempotent bypass `machine.Cancel`, and the SDK's dual path (registerPrompt ctx-cancel + the `Cancel` handler) both fan into that same idempotent `Cancel`. Locked on the raw tee (loop_test, cancel_test).
- **Honest initialize** is byte-locked (`honestInitializeFrame`), and I verified against the SDK that `AgentCapabilities` zero value marshals explicit-false capabilities and has no fs/terminal fields at all — honesty by construction is real.
- **Flush-before-stopReason** holds structurally: the tap registers on the conn fan-out before any machine, delivery is in registration order, and the SDK serializes all outbound frames under one write mutex.
- **wireError's "must stay unwrapped" claim is accurate**: I read the SDK's `toReqErr` — it type-asserts `*RequestError` with no `errors.As`; the adapter's discipline is required, and correct.
- The D-03 wire sequence (bare create → desktop-continuous subscribe → setModel follow-up with `options.reasoningLevel` + explicit `persistAsWorkspaceLastUsed:false`) matches `docs/protocol/app-server-dialect.md` §2/§4 verbatim and is asserted on fake transcripts at both test tiers.

The concerns are in the composition seams rather than the contracts: a **single-slot router-event hook that its own "one active turn per front" justification does not actually protect** across multiple sessions (WR-01), an **unsynchronized sink publication** whose "pipes are empty at construction" premise is false for production os.Stdin (WR-02), a **cross-pump liveness coupling** where the backend read loop parks inside an ACP write (WR-03), and error-path/lifecycle leaks that are cheap to fix now and expensive to debug later (WR-04, WR-07).

No structural findings were provided with this review; the `## Structural Findings (fallow)` section is therefore omitted per schema. All findings below are from direct adversarial review; each was verified against current source (and, where the SDK is cited, against the v0.20.0 module source).

## Warnings

### WR-01: OnRouterEvent single slot is stolen by every session/new — machines other than the last-registered lose the degraded-resolution path

**File:** `internal/zcode/conn.go:107-119`, `internal/turn/machine.go:182-197`, `internal/acpfront/front.go:199`
**Issue:** `NewMachine` calls `conn.SetOnRouterEvent(m.observeRouterEvent)` — a single, last-wins slot — once per `session/new`. The slot's design comment justifies the limitation with "one active turn per front", but that invariant gates *prompts*, not *machine construction*: nothing serializes `NewSession` against the turn slot, and any number of machines may be registered. With two sessions A then B, the slot points at machine B forever. If a tracked-no-answer reverse request (the credential-cipher obligation the real backend fires mid-turn, dialect §3) expires its deadline while **A's** turn is running, `recordRouterEvent` hands the expiry to B — which is idle and returns early — so machine A never resolves `SignalDegraded`/`StopError`. The turn then resolves via terminal event or quiet-window, i.e. it can answer `end_turn` for a turn the Phase-3 semantics classify as unable to complete, and the D-05 "recorded, never swallowed" contract silently degrades at the machine level (the conn's slog warning still fires, so the observation survives in logs only). The mid-traffic *memory race* was correctly fixed by `routerMu` (commit b312fe9), but the *semantic* clobber is unguarded and unreachable by the current tests: the E2E creates A, prompts A, then creates B — ordering that masks the steal. Multi-session is normal ACP-client usage (one session per conversation), so this is reachable in the first real client. Borderline-BLOCKER; must be fixed before multi-session support (Phase 5 CORE-06) builds on this seam.
**Fix:** Make router-event delivery a fan-out keyed by session, mirroring the notification observer list (`AddRouterObserver` returning a remove func), or re-bind the slot to the active machine inside `acquireTurn` and restore on release as an interim guard. Add a test: create A, create B, prompt A, force a `RouterDeadlineExpired`, assert A resolves `StopError`/`SignalDegraded`.

### WR-02: agentAdapter.sink is published without synchronization — a memory-model data race whose "pipes are empty at construction" premise is false in production

**File:** `internal/acpport/agent.go:27-30, 50-63, 271-288`; wiring at `cmd/zcode-acp/main.go:196`
**Issue:** `bindSink` writes the plain field `a.sink` *after* `acp.NewAgentSideConnection` has already spawned the SDK's receive/dispatch goroutines; `sinkForPrompt` reads it with no synchronization. The safety argument in the comment — "the pipes are empty at construction" — holds for the test harness's io.Pipes (the harness writes only after construction, which creates the happens-before edge), but not for production: `os.Stdin` is an OS pipe the ACP client may have written into long before `run()` finishes discovery and child spawn. The SDK's `receive()` drains buffered bytes immediately, so handler goroutines genuinely run concurrently with `bindSink`. `Initialize` does not touch the sink, and a Prompt cannot be dispatched before the initialize response reaches the client, so on current hardware the stale-nil read is unobservable — but there is no Go-memory-model happens-before edge between the write and a Prompt goroutine's read, and the race detector could never catch it here because the E2E builds a non-instrumented binary. The failure mode if it ever does misfire is bounded but protocol-visible (every delta fails via the fallback error sink, chunks silently lost to warnings).
**Fix:** Publish atomically: store `sink atomic.Pointer[UpdateSink]` (set in `bindSink`, loaded in `sinkForPrompt`), or bind the sink via a channel handed to the adapter at `NewAgent` time and have `bindSink` close it. One-line change; makes the construction discipline structural instead of rhetorical.

### WR-03: The delta tap's inline ACP write couples the backend read loop to ACP-client progress — the conn hook contract ("must not block") is violated by the streaming design itself

**File:** `internal/acpfront/stream.go:78-88`; delivery context at `internal/zcode/conn.go:311-318`; sink write at `internal/acpport/sink.go:16-24` → SDK `connection.go sendMessage` (blocking `c.w.Write` under `writeMu`)
**Issue:** `deliverNotification` iterates observers synchronously under `obsMu` on the backend read goroutine. The tap's `emit` → `conn.SessionUpdate` → `SendNotification` → blocking write to the ACP wire runs inside that iteration. If the ACP client applies backpressure (slow renderer, stalled reader — the 64 KiB OS pipe buffer fills), the zcode read loop parks inside the tap's write while holding `obsMu`: backend events stop being demuxed (terminal events, stop-outcome patches, poll replies all queue behind it), `AddObserver`/`Remove` block, and the prompt's own response — written through the same SDK `writeMu` — cannot go out either. The turn eventually unwinds via the wall-clock quiet-window gates, but the response stays undeliverable until the client resumes reading: a stall of one leg wedges both legs with no bound and no drop policy. This is not the out-of-scope "performance" category — it is a liveness property of the streaming path, and the conn's own hook contract ("an observer must not block") is formally violated by the one observer that streams.
**Fix:** Either (a) decouple: per-turn bounded queue + writer goroutine with an explicit policy (drop-oldest with a structural warn, or coalesce), so the read loop never parks on the ACP wire; or (b) keep the synchronous pump but document the coupling as the deliberate backpressure design in stream.go and conn.go, and bound it — today the tradeoff (no unbounded buffering, in exchange for whole-front stall propagation) exists only implicitly.

### WR-04: session/new error path orphans a live backend session

**File:** `internal/acpfront/front.go:162-193`
**Issue:** If `session/create` succeeds but `session/subscribe` or `session/setModel` fails (backend refusal, wire hiccup, `-32603` level refusal), the front returns a coded error and leaves the just-created backend session fully live: it holds host obligations, is subscribed-or-not depending on where it failed, has no registry entry, and nothing will ever close it — `CloseSession` is answered `-32601` in this build, and the process-lifetime conn keeps it until exit. A client retry loop against a misconfigured backend (e.g. the missing-reasoning-level refusal of IN-04) leaks one backend session per attempt.
**Fix:** On any post-create failure, best-effort tear the backend session down (`conn.Call(ctx, "session/close", ...)` on the dialect's method map — the request is host-side, so the ACP `-32601` advertisement is irrelevant) and log structurally; or at minimum record the orphan in a slog warning naming the sessionId so operators can see accumulation.

### WR-05: The front-wide turn slot rejects cross-session prompts with a per-session message — the client-visible text contradicts the actual scope

**File:** `internal/acpfront/front.go:207-213`, `internal/acpfront/turnctx.go:31-37`
**Issue:** The slot is front-wide by design (D-04's delta-routing precondition), but the one stable rejection message says "a prompt is already running **for this session**". When a client prompts session B while session A's turn runs, the guard fires and the client is told the conflict is in session B — which is false, and will send a real user debugging in circles (they will look for a stuck turn in the wrong conversation). The message text is also the exact string locked by cancel_test.go, so fixing it requires touching the lock — which is the point of the lock.
**Fix:** Change the constant to scope-accurate text, e.g. `"another prompt is already running (this agent runs one turn at a time)"`, update the two test constants (`front.go:213`, `cancel_test.go:45`), and keep the numeric class `-32600` unchanged. Optionally include both session ids structurally — the ids are client-supplied, so echoing them rides no internal state.

### WR-06: Image/audio/resource content blocks are dropped silently — no error, no log, no client signal

**File:** `internal/acpport/agent.go:178-190` (`joinContentBlocks`)
**Issue:** The minimal loop keeps only text and resource-link-URI blocks; image, audio, and embedded-resource blocks vanish without any observable trace — not even a debug log. The defense "promptCapabilities.image/audio are advertised false" is real (I verified the byte-locked initialize frame), and per the ACP v1 schema those blocks "require the 'image'/'audio'/'embeddedContext' prompt capability when included in prompts" — so a conforming client will not send them. But the advertise=implemented guarantee this project runs on cuts both ways: when a non-conforming or buggy client *does* send an image, the user's content is silently truncated and the model answers as if it never arrived, with zero evidence anywhere. For production-bound protocol code, silent data loss with no signal is the wrong default even for out-of-contract input.
**Fix:** Answer unsupported block kinds loudly: `-32602` invalid-params naming the block type ("session/prompt: image content requires a capability this build does not advertise"), or minimally emit a structural slog.Warn (kinds and counts only, never content) at the drop site so the behavior is at least diagnosable.

### WR-07: Sessions accumulate without bound — registry entries, fan-out observers, and machines are never released

**File:** `internal/acpfront/front.go:199-200`; `internal/core/registry.go:52-85`; `internal/turn/machine.go:182-197`
**Issue:** Nothing ever calls `registry.Remove` or the `AddObserver` remove func in production: every `session/new` permanently adds a registry entry, a conn observer (each one re-scans every subsequent notification for the life of the process), and a machine. Phase 5's session/close is the designed owner of teardown, but until then a long-lived process under a client that creates sessions liberally (the normal pattern) grows all three without bound, and every notification costs O(sessions) observer iterations. The review scope explicitly names session leaks; this is recorded so the Phase-5 plan owns the cleanup contract — including that `Machine` must expose an unregister path (today `NewMachine` hides the `AddObserver` remove func internally, so teardown cannot remove the observer even once close exists).
**Fix:** Now: have `NewMachine` return (or the front capture) the observer remove func and register it on the `core.Session` so Phase 5's close can release both. Phase 5: wire `session/close` → `Remove` + observer removal + machine teardown, and lock it with a test asserting the conn's observer count returns to baseline.

## Info

### IN-01: AddObserver's doc claims mid-traffic registration "is a data race" — the mutex makes it safe; the real contract is delivery ordering, and the text misstates it

**File:** `internal/zcode/conn.go:188-198`
**Issue:** The fan-out doc carries over the single-slot era's warning verbatim ("registering (or removing) while traffic is in flight is a data race"). Since deliveries iterate under `obsMu`, registration mid-traffic is memory-safe — it merely means the new observer sees only subsequent envelopes. WR-01 is exactly the kind of bug a maintainer might mis-diagnose using this sentence (looking for a race where the defect is semantic). Commit 9d60672 replaced the slot with a locked fan-out but did not update the contract text.
**Fix:** Rewrite the sentence: registration/removal is lock-safe at any time; the guarantee an observer gets is "every envelope from its registration onward, in wire order"; register-before-traffic remains required only where *first* envelope delivery matters (the tap-before-machine ordering).

### IN-02: Generality without instantiation — `core.Session.CancelTurn` is written by no one and read by no one; `Subscribed` is written but never read

**File:** `internal/core/registry.go:30-35`
**Category:** necessity
**Issue:** `CancelTurn context.CancelFunc` is documented as "the per-turn cancel func the front owns (Pattern 5 turn-context ownership)", but the Pattern-5 mechanism that actually shipped lives in `acpfront`'s `acquireTurn` closures (turnctx.go) — the field has zero production references (verified by grep across `internal/` and `cmd/`). `Subscribed` is set `true` at front.go:200 and never consulted — the subscribe gating it documents is enforced by call ordering, not by reading the flag. The state cannot occur that this field defends against: there is no code path where a registry consumer needs either field today.
**Fix:** Delete `CancelTurn` (and fold `Subscribed` into a comment on the D-03 sequence) unless the Phase-5 plan names a consumer; if they are genuinely reserved slots for CORE-06/session-close, mark them as such in the doc comment citing the plan, so the next reviewer does not re-derive their absence.

### IN-03: Stale doc comments — the front's base-context claim and the package doc's plan framing no longer match the code

**File:** `internal/acpfront/front.go:97-100`; `internal/acpfront/doc.go:9-12`
**Issue:** front.go's New comment says "the cmd wiring derives a cancellable one when the supervisor lifecycle joins (plan 04-05)" — plan 04-05 landed and `New` still hardcodes `context.Background()` with no override seam; the comment promises a mechanism that does not exist. doc.go still says "the Front itself arrives with plan 04-06, whose tracer is this harness's first full-loop consumer" — the Front and the tracer both shipped; the paragraph reads as if the package were still a harness-only shell.
**Fix:** Update both comments to state what is true: the base context is deliberately process-lifetime (turn contexts are cancelled by `release`; SDK disconnect reaches the turn via the fan-in → `machine.Cancel`), and the harness now drives the Front (04-06).

### IN-04: `--model` without `--reasoning-level` is accepted at config time and sends `options.reasoningLevel:""` — a guaranteed backend refusal on level-strict providers, discovered only at first session/new

**File:** `cmd/zcode-acp/main.go:113-139`; consequence at `internal/acpfront/front.go:181-193`
**Issue:** The dialect records that level validation is strict in both directions and that the known refusal for a level-less z.ai binding is `-32603 "Reasoning level is required"` (§2). `parseConfig` only rejects the inverse combination (level without model); a bare `--model zai-api/GLM-5.3-Flash` passes and the follow-up serializes `"options":{"reasoningLevel":""}` — neither the documented omit-shape nor a valid level — so the failure surfaces at session/new time with backend noise, compounded by WR-04's session orphan per attempt.
**Fix:** Either warn at startup when a model is configured without a level (naming the providers that require one), or omit the `options` object when `ReasoningLevel` is empty so the wire carries the documented shape and the backend's own actionable refusal text.

### IN-05: The tap attributes every part.delta to the active turn without checking turnId — correct only under the documented single-turn precondition, and one field away from being structural

**File:** `internal/acpfront/stream.go:40-89`
**Issue:** `observeEnvelope` routes any `part.delta` envelope to the active turn's emitter, ignoring the `turnId` every session/event envelope carries. Under the enforced invariants (front-wide slot + backend `-32010`) a foreign-turn delta cannot exist, so this is not a bug today — but the correctness rests on an invariant documented in a different file (front.go type doc), and nothing in the tap would catch its violation: when Phase 5 adds steering (the dialect's `turn.steerQueued` exists) or per-turn routing, the first symptom will be misattributed chunks. The wire already provides the discriminator.
**Fix:** Record the turn's id in the turn slot (learn it from the `turn.started` envelope the tap already sees, or expose the machine's learned id) and skip deltas whose turnId does not match — a three-line guard that turns the precondition into a structural check.

---

_Reviewed: 2026-11-20T06:20:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_

## REVIEW COMPLETE

- **Critical (BLOCKER): 0**
- **Warning: 7** (WR-01 router-slot steal, WR-02 sink publication race, WR-03 cross-pump stall coupling, WR-04 session/new orphan leak, WR-05 mis-scoped rejection message, WR-06 silent content-block drop, WR-07 unbounded session accumulation)
- **Info: 5** (IN-01 stale fan-out contract doc, IN-02 necessity: dead registry fields, IN-03 stale package docs, IN-04 reasoning-level config gap, IN-05 turnId guard absent)
- **Total: 12** (necessity findings: 1 — counted in Info, not in total)
