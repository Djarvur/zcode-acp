# Phase 4: ACP Front — Minimal Chat Loop - Pattern Map

**Mapped:** 2026-11-20
**Files analyzed:** 12 (8 new + 4 modified)
**Analogs found:** 12 / 12 (all in-repo or module-cache verified this session; every path below is git-tracked, confirmed via `git ls-files`)
**Primary source for seam files:** RESEARCH.md "Architecture Patterns" — Pattern 1 (seam), Pattern 2 (delta tap), Pattern 3 (flush-before-stopReason), Pattern 4 (observer fan-out), Pattern 5 (turn-ctx ownership), Pattern 6 (fake-driven test client). The acpport/acpfront shapes below cite those Pattern numbers plus concrete anchors.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Take (copy) | Change |
|---|---|---|---|---|---|
| `internal/acpport/types.go` (new) | types (neutral DTOs) | transform | RESEARCH Pattern 1 sketch + SDK `types_gen.go` request/response shapes | Handlers interface vocabulary (`InitReport`, `NewSessionParams`, `PromptParams`, `StopReason` consts, `UpdateSink`) | None in-repo — new package; DTO field names mirror SDK types so translation is 1:1 |
| `internal/acpport/agent.go` (new) | adapter (implements `acp.Agent`) | request-response | SDK `types_gen.go:15349-15405` (Agent iface) + `errors.go:58-60` (NewMethodNotFound) + `machine.go:178-194` (hook-wiring style) | 12-method Agent impl; unadvertised methods → `acp.NewMethodNotFound`; `agentAdapter` translates DTO ⇄ SDK | Authenticate = deliberate -32601/error exception (A2); Initialize assembles `acp.InitializeResponse` from `InitReport` |
| `internal/acpport/sink.go` (new) | adapter (outbound update send) | streaming | SDK `agent_gen.go:406-408` (`SessionUpdate`) + `helpers.go:98-111` (UpdateAgentMessageText/ThoughtText) | `sink(ctx, acp.SessionNotification{SessionId, Update})` passthrough; chunk construction via SDK helpers | Stub `RequestPermission` for Phase 5 (outbound RPC exists at `agent_gen.go:402`) |
| `internal/acpfront/front.go` (new) | service/controller (agent policy + session registry) | request-response + session CRUD | `internal/turn/machine.go` (guard/registry structure) + `internal/zcode/call.go:127` (`Call`) + `envelope.go:139-149` (method consts) | Handler-per-method structure; `ErrTurnActive`-style wrapper guard (machine.go:137, 356-367); `conn.Call` params maps (RESEARCH "Backend-leg calls") | session/new = create → subscribe → setModel sequence (D-03); registry = mutex map (home: `internal/core` or acpfront — Open Question 5) |
| `internal/acpfront/stream.go` (new) | middleware/tap (delta translation) | event-driven streaming | `internal/zcode/conn.go:243-249` (`deliverNotification`) + RESEARCH Pattern 2 | Tap `part.delta` field text/reasoning → `UpdateAgentMessageText`/`UpdateAgentThoughtText` in receive order; ignore `input`/`output` | Runs BEFORE machine classification (fan-out order, Pattern 3); never emits `part.upserted` (Pitfall 5) |
| `internal/acpfront/turnctx.go` (new) | utility (per-turn context ownership) | event-driven | `internal/turn/machine.go:290-308` (`Cancel`) + SDK `agent.go:61-119` (cancelPrompt/registerPrompt) + `errors.go:79-90` (`toReqErr`) | Turn ctx derived from front base ctx, NOT sdkCtx; cancel fan-in goroutine selecting {sdkCtx, turnCtx}; machine.Cancel idempotency | StopFailed/StopError → -32603 error from Prompt (A5); never return ctx.Canceled (Pitfall 2) |
| `internal/zcode/conn.go` (modify) | middleware (demux) | event-driven | self — `conn.go:83-88` (OnNotification slot) + `:243-249` (`deliverNotification`) | Replace single slot with ordered observer list (`AddObserver(fn) (remove func())`), synchronous delivery in registration order on the read goroutine (RESEARCH Pattern 4) | Keep the hook contract doc ("must not block", assign-before-traffic); `NewMachine` call site adapts |
| `internal/turn/machine.go` (modify) | state machine | event-driven | self — `machine.go:187-190` (hook assignment) | Register via `conn.AddObserver` instead of assigning `OnNotification` | No logic change; construction only |
| `internal/zcode/envelope.go` (modify) | config/constants | transform | self — `envelope.go:136-149` (method table, D-06 naming) | Append `MethodSessionSubscribe = "session/subscribe"`, `MethodSessionSetModel = "session/setModel"` | None — table extension only |
| `internal/zcode/fake/testdata/scenarios/{setmodel-followup,part-delta-streaming,no-reasoning,mcp-servers-ignored}.json` (new) | test fixture | event-driven | `testdata/scenarios/happy-turn.json` (whole-file shape) + `cancel-mid-turn.json` (stop rule) | `on_request` rules with `reply` + ordered `side_effects` (`emit` rows for `session/event`); `unknown_method` -32601 row; `eof_drain_ms` | Each adds `session/subscribe` + (`session/setModel`) on_request rows — the fail-loud loader rejects unscripted methods via unknown_method. NOTE: `mcp-servers-ignored` is SUPERSEDED — per 04-VALIDATION.md Wave-0 note the mcp accept-and-ignored behavior is locked at the FRONT tier over the setmodel-followup scenario (the back leg has no mcp wire surface to script); no Phase-4 plan authors it |
| `internal/zcode/fake/testdata/golden/*.golden` (new) | test fixture | — | `testdata/golden/happy-turn.golden` | `in\t{...}` / `out\t{...}` tab-separated compact-JSON lines, byte-deterministic (compactRaw at load) | Recorded with `go test -update` (CompareGolden); missing fixture FAILS |
| `internal/zcode/fake/golden_test.go` (modify: +drivers) | test | event-driven | `golden_test.go:158-200` (`goldenDrivers` map + TestGolden) | One `driverFunc` per new scenario in the map — driver-less scenario fails loud (line 193) | Drivers must register the new scenario names |
| acpfront test files incl. test ACP client (new) | test | request-response + streaming | SDK `client.go:16` (`NewClientSideConnection`) + `types_gen.go:15594` (Client iface) + `golden_test.go:204-215` (replay harness) + RESEARCH Pattern 6 | In-process composition: `NewClientSideConnection` ⇄ recording-pipe tee ⇄ `NewAgentSideConnection`; ~40 LOC tee captures raw frames for wire-order asserts; back leg rides `fake.Serve` (`fake.go:102`, returns `Transcript` `fake.go:62`) | Client impl collects SessionUpdates; transcript asserts chunk-frames-before-response (CORE-03) |
| `cmd/zcode-acp/main.go` (modify) | entrypoint | streaming (stdio) | self (26-line skeleton) + SDK `agent.go:32-40` (`NewAgentSideConnection(agent, peerInput io.Writer, peerOutput io.Reader)`) | Wire `acpport.NewAgent(os.Stdout, os.Stdin)`-shaped composition; `version` var (main.go:15) is the D-02 agentInfo version source | Keep stdout protocol-exclusive; `--version` path unchanged |

## Pattern Assignments (anchors)

### `internal/acpport/*` (seam — RESEARCH Patterns 1, 2, 5 are the design source)
**External analog:** `/home/nil/go/pkg/mod/github.com/coder/acp-go-sdk@v0.20.0/` (git-tracked origin: github.com/coder/acp-go-sdk v0.20.0; planner reads from module cache after `go get`)
- Agent interface to implement (12 required methods, verbatim): `types_gen.go:15349-15405` — Authenticate, Initialize, Logout, Cancel, CloseSession, DeleteSession, ListSessions, NewSession, Prompt, ResumeSession, SetSessionConfigOption, SetSessionMode
- Connection constructor: `agent.go:32-40` `NewAgentSideConnection(agent Agent, peerInput io.Writer, peerOutput io.Reader)`
- Outbound updates: `agent_gen.go:406-408` `func (c *AgentSideConnection) SessionUpdate(ctx, params SessionNotification) error`
- Chunk construction: `helpers.go:98-101` `UpdateAgentMessageText(text)`; `:108-111` `UpdateAgentThoughtText(text)`; `:5-11` `TextBlock`
- Unadvertised-method default: `errors.go:58-60` `NewMethodNotFound(method)` → -32601 (matches SDK's own optional-interface behavior, Pitfall 6)
- Cancel machinery to compose with: `agent.go:61-73` `cancelPrompt` cancels the registered prompt ctx; `:90-119` `registerPrompt` — the Prompt handler runs under SDK `context.WithCancel`
- Constants: `constants_gen.go` — `ProtocolVersionNumber = 1` (wire), `SchemaVersion = "1.32.0"`
- Honest-initialize shape (executed probe, RESEARCH Code Examples): zero-value `AgentCapabilities` + `AgentInfo{name,title,version}` + `AuthMethod{Agent: &AuthMethodAgent{Id, Name}}` both entries; agent-variant marshals without a `type` field

### `internal/acpfront/front.go` (policy + registry)
**Analog:** `internal/turn/machine.go`
- Concurrency-guard structure: `machine.go:130-137` (`ErrTurnActive` sentinel) and `:356-367` (`guardTurnLocked`) — the D-05 second-prompt guard surfaces the same sentinel
- Backend calls (RESEARCH "Backend-leg calls", all verified Phase-3 surfaces): `zcode/call.go:127` `Call(ctx, method, params)`; `envelope.go:140-142` `MethodSessionCreate/Send/Stop` exist; subscribe/setModel constants added this phase. create is bare (no model — Pitfall 4); setModel payload carries `model.options.reasoningLevel` (+ `persistAsWorkspaceLastUsed:false`, Open Question 3); `cwd` MUST be absolute (V5)
- Package doc style: `machine.go:1-13` — decision-ID-annotated package comment convention to copy

### `internal/acpfront/stream.go` (delta tap)
**Analog:** `internal/zcode/conn.go` + RESEARCH Pattern 2
- Delivery contract: `conn.go:83-88` — hooks run on the read goroutine, must not block; `:243-249` `deliverNotification` is wire-order preserving. The tap joins this path via the new observer list
- Delta carrier (zcode-src schema, RESEARCH Pattern 2 verbatim): `part.delta` payload `{messageId, partId, field, delta}`, `field ∈ {"text","reasoning","input","output"}`; text→message chunk, reasoning→thought chunk, input/output ignored in Phase 4
- Ordering rule (RESEARCH Pattern 3): tap processes an envelope BEFORE `machine.ObserveEnvelope` classifies it; per-envelope one chunk, arrival order; server-side batching means batched (not per-token) deltas

### `internal/acpfront/turnctx.go` (cancel contract)
**Analog:** `internal/turn/machine.go` + SDK `errors.go`/`agent.go`
- Exactly-once cancel: `machine.go:290-308` `Cancel()` — idempotent, bypass `conn.Stop` (`zcode/call.go:196`), first call wins
- The -32800 trap: SDK `errors.go:86-88` `toReqErr` maps `context.Canceled` → `NewRequestCancelled` — Prompt must NEVER return ctx.Canceled; convert to `StopReasonCancelled` success (RESEARCH Pattern 5 skeleton, quoted there in full)
- StopReason mapping: machine `StopCompleted/Cancelled/Failed/Error` (`machine.go:35-40`) → `end_turn`/`cancelled`/JSON-RPC -32603 (A5); SDK stopReason consts at `types_gen.go:12442-12458`

### `internal/zcode` observer fan-out (RESEARCH Pattern 4)
**Analog:** self, two sites
- The slot being replaced: `conn.go:83-88` (`OnNotification func(env Envelope)`) — single field, assigned post-`NewConn` before traffic
- The overwrite site: `machine.go:187-190` — `conn.OnNotification = m.ObserveEnvelope` per machine (second session steals the slot, Pitfall 3)
- New API per RESEARCH: `AddObserver(fn) (remove func())`; ordered synchronous delivery inside `deliverNotification` (`conn.go:243-249`); safety precedent for late observers: `machine.go:330` (ObserveEnvelope drops envelopes when not running)

### Fake corpus additions
**Analog:** `internal/zcode/fake/` + `testdata/`
- Scenario schema: `scenario.go:23-75` (`Scenario`/`RequestRule`/`Reply`/`SideEffect`/`Emit`/`Reverse`; snake_case schema keys, camelCase wire payloads verbatim); strict loader `:82-106` (DisallowUnknownFields, fail-loud)
- Row formats to copy: `testdata/scenarios/happy-turn.json` — `session/create` rule with lifecycle emits + reverse `requestRuntimePreferences`; `session/send` rule with `state.updated` + `session/event` turn.started/completed emits; `session/stop` bare rule (from `cancel-mid-turn.json`)
- New emit rows: `session/event` `part.delta` params `{"type":"part.delta","seq":N,"turnId":"turn_fix1","messageId":"msg_1","partId":"part_1","field":"text","delta":"..."}` (shape per RESEARCH Pattern 2 zcode-src schema; envelope type/seq/turnId per happy-turn.json:47-54)
- Golden format: `testdata/golden/happy-turn.golden` — `in\t{...}`/`out\t{...}` one compact envelope per line; recording via `fake.CompareGolden` (`golden.go:29-58`, `-update` flag; missing fixture fails)
- Driver registration: `golden_test.go:158-171` `goldenDrivers()` map — every new scenario needs its `driverFunc` or TestGolden fails loud (`:193`)
- Serve surface: `fake.go:102` `Serve(rw io.ReadWriteCloser, sc *Scenario) (Transcript, error)`

## Shared Patterns

### Hook/liveness contract
**Source:** `conn.go:83-88`, `machine.go:319-325`
**Apply to:** fan-out observers, the tap, all front handlers on the demux path
Runs on the read goroutine; must not block; assign/register before any frame flows.

### Exactly-once under mutex
**Source:** `machine.go:567-579` (`resolveLocked`), `:290-308` (`Cancel`)
**Apply to:** turnctx cancel fan-in, observer remove funcs, prompt-answer resolution
First signal wins; repeats are no-ops; the hook/counter is the observable proof.

### Fail-loud fixtures
**Source:** `golden.go:44-49` (missing golden fails), `golden_test.go:191-194` (driver-less scenario fails), `scenario.go:91` (strict decode)
**Apply to:** the new scenarios + goldens + drivers (setmodel-followup, part-delta-streaming, no-reasoning, plus cancel-mid-turn's preamble extension; `mcp-servers-ignored` superseded — see the File Classification note).

### Method-constant naming (D-06)
**Source:** `envelope.go:136-149`
**Apply to:** `MethodSessionSubscribe`, `MethodSessionSetModel` — PascalCase name from the verbatim dialect method string.

### Never return ctx.Canceled from Prompt
**Source:** SDK `errors.go:79-90`; spec rule quoted in RESEARCH Pattern 3
**Apply to:** acpfront Prompt/Cancel — cancelled is a success `stopReason`, exactly once.

### Honest advertisement by construction
**Source:** SDK `types_gen.go:62-98` (AgentCapabilities has no fs/terminal fields at all); RESEARCH A7 + probe output
**Apply to:** acpport Initialize assembly, CORE-01 honesty test — assert on ABSENT fields (no loadSession/fs/terminal), not on empty-object cosmetics.

### stdout-is-protocol
**Source:** `envelope.go:1-9` (package doc), RESEARCH Anti-Pattern "Writing anything but ACP frames to stdout"
**Apply to:** cmd/zcode-acp wiring, all new packages — slog to stderr only.

## No Analog Found

None. Every file maps to an in-repo or module-cache analog. Note: the acpport *neutral DTO vocabulary* itself has no in-repo precedent — its shape is RESEARCH Pattern 1's sketch and is planner discretion per CONTEXT ("acpport seam API shape").

## Metadata

**Analog search scope:** `internal/{turn,zcode,zcode/fake,supervisor}`, `cmd/`, `experiments/`, SDK module cache `/home/nil/go/pkg/mod/github.com/coder/acp-go-sdk@v0.20.0/`
**Files scanned:** 16 in-repo sources + 6 SDK files (targeted reads; types_gen.go read via grep+window only — 16,399 lines)
**Tracked-source gate:** all named analogs verified `git ls-files`-tracked (SDK analog cited at its tracked origin repo, read from the module cache)
**Pattern extraction date:** 2026-11-20

## PATTERN MAPPING COMPLETE
