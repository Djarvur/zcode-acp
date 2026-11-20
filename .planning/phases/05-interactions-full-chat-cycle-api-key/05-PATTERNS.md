# Phase 5: Interactions & Full Chat Cycle (api-key) - Pattern Map

**Mapped:** 2026-11-20
**Files analyzed:** 21 (new + modified groups)
**Analogs found:** 19 / 21 (2 no-analog: cipher-decrypt subprocess, permission backend bridge — research-supplied shapes)
**Wire-shape source of truth:** 05-RESEARCH.md Patterns 1–7 (file:line-verified this session). Extract FROM THERE, never re-derive; this map points at the in-repo code each new file copies structurally.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match |
|---|---|---|---|---|
| `internal/interact/responder.go` (NEW) | middleware (reverse-request policy) | request-response | `internal/zcode/conn.go` (OnRequest seam + routeRequest + answerRuntimePreferences) | exact |
| `internal/interact/dedupe.go` (NEW) | utility (bounded correlation table) | request-response | `internal/zcode/call.go` pendingTable | role-match |
| `internal/interact/permission.go` (NEW bridge) | service (backend→front forwarder) | event-driven + request-response | none in-repo (first client round-trip) — 05-RESEARCH.md Pattern 3 code example | none (research-supplied) |
| `internal/interact/cipherdecrypt.go` (NEW, D-06 seam) | service (subprocess + crypto) | file-I/O | none in-repo — 05-RESEARCH.md Pattern 6 field-verified shim shapes | none (research-supplied) |
| `internal/hostsvc/snapshot.go` (NEW api-key snapshot builder) | service (host→server push) | request-response | `internal/acpfront/front.go` NewSession backend-call sequence | role-match |
| `internal/hostsvc/readback.go` (NEW awaitRegistryContains) | utility (poll-until-observed) | event-driven | `internal/turn/machine.go` classifyIdlePatchLocked + watcher pattern | partial |
| `internal/acpfront/auth.go` (NEW Authenticate) | controller | request-response | `internal/acpfront/front.go` (NewSession call sequence; error surfacing) | exact |
| `internal/acpfront/lifecycle.go` (NEW Resume/Close + teardown) | controller | CRUD | `internal/acpfront/front.go` NewSession + `internal/core/registry.go` Remove | exact |
| `internal/acpfront/stream.go` (EXTEND tool.updated) | observer/transform | streaming | itself (part.delta tap) | exact |
| `internal/acpfront/permission.go` (NEW bridge impl) | controller (pending table + cancel fan-in) | event-driven | `internal/acpfront/turnctx.go` acquireTurn + front.go cancel fan-in | role-match |
| `internal/acpport/types.go` (EXTEND ToolNote) | types | transform vocabulary | itself (SessionNote) | exact |
| `internal/acpport/sink.go` (EXTEND tool-note mapping) | transform | streaming | itself (noteToNotification) | exact |
| `internal/acpport/agent.go` (EXTEND: Authenticate real, Resume/Close, WR-02 atomic sink) | adapter | request-response | itself | exact |
| `internal/turn/machine.go` (EXTEND WR-07 observer remove) | state machine | event-driven | itself (NewMachine wiring :182-197) | exact |
| `internal/zcode/conn.go` (EXTEND WR-01 AddRouterObserver) | middleware | event-driven fan-out | itself (AddObserver :199-218) | exact |
| `internal/acpfront/front.go` (EXTEND WR-05 wording, IN-05 turn-id guard site) | controller | request-response | itself (turnActiveMessage :213) | exact |
| `internal/zcode/fake/scenario.go` + `fake.go` (EXTEND re-announce) | test fixture DSL | scripted reverse | itself (Reverse/runReverse) | exact |
| `internal/zcode/fake/testdata/scenarios/*.json` + `golden/` (NEW ~7) | test corpus | scripted | `runtime-preferences.json`, `credential-cipher-scripted.json` | exact |
| `internal/acpfront/harness_test.go` (EXTEND RecordingClient.RequestPermission + tool capture) | test harness | request-response | itself | exact |
| `test/realdesktop/` (NEW guarded tier) | test (manual/guarded) | request-response real | `internal/supervisor/realdesktop_test.go` | exact |
| `cmd/zcode-acp/main.go` (EXTEND responder wiring + ZCODE_ACP_API_KEY) | config/composition | — | itself (parseConfig/run) | exact |

## Pattern Assignments

### `internal/interact/responder.go` (NEW — the D-01 policy object)

**Analog:** `internal/zcode/conn.go` — the responder takes over what routeRequest does, via the seam conn already exposes.

**Seam to wire through** (conn.go:91-95, 325-336):
```go
// OnRequest, when non-nil, intercepts a server→host request before the
// built-in router; returning handled=false falls through to the
// router's defaults...
OnRequest func(ctx context.Context, env Envelope) (result any, werr *WireError, handled bool)

func (c *Conn) handleRequest(ctx context.Context, env Envelope) {
    if c.OnRequest != nil {
        result, werr, handled := c.OnRequest(ctx, env)
        if handled { c.replyRequest(env.ID, result, werr); return }
    }
    c.routeRequest(ctx, env)
}
```
The responder's `Serve` always returns `handled=true` (CORE-06: it answers everything, unknown included).

**Copy the deadline mechanics** from `answerRuntimePreferences` (conn.go:365-395) and `deadlineFor` (:422-430): `context.WithTimeout(ctx, policy.Deadline)`, `errors.Is(ctx.Err(), context.DeadlineExceeded)` distinguishes genuine expiry from conn-shutdown, expiry recorded via a RouterEvent + structural slog (`recordDeadlineExpired` :436-450 — method/id-key/duration only, never params).

**Copy the -32601 default verbatim** (conn.go:352-357):
```go
c.replyRequest(env.ID, nil, &WireError{
    Code:    CodeMethodNotFound,            // -32601 (envelope.go:160)
    Message: "unsupported host request: " + env.Method,
})
```

**Copy the static runtime-prefs answer bytes** (conn.go:384-387) — D-01 migrates it into the policy table byte-identical; the existing golden scenario `runtime-preferences.json` re-points at the responder.

**Replies ride the bypass lane** via `conn.Reply`/`conn.ReplyError` (call.go:219-249) — never the queue lane (dialect §1; anti-pattern list).

**Policy table shape** (fields verbatim from 05-RESEARCH.md Pattern 1 — Answer/Request/MethodPolicy structs + per-method deadline/dedupe table). Request normalization digs `params.requestId`/`params.sessionId` **the way machine.go does** (`paramsString`, machine.go:587-600 — camelCase wire keys, tagliatelle-safe).

### `internal/interact/dedupe.go` (NEW — re-announce dedupe)

**Analog:** `internal/zcode/call.go` pendingTable (:53-117) — the in-repo model for a mutex+map correlation table with **delete-under-lock exactly-once**: `resolve` deletes under the same lock `drainAll` holds, so an entry can never be answered twice. Copy that discipline; change the key to `(method, params.requestId)` and add entry eviction on answer / conn close / TTL (deadline + grace).

**Wire mechanics to implement** (05-RESEARCH.md Pattern 2 — cite, do not re-derive): re-announce = NEW wire id (`server-<n>`), same `params.requestId`, 1 s doubling to 10 s cap; server registers ALL wire ids until the FIRST reply; zod-validates the result. Consequence: claim once per `(method, requestId)`, reply once to the registering wire id, later re-announce envelopes are logged (id keys only) and dropped, no reply.

### `internal/interact/permission.go` (NEW — PermissionBridge backend half)

**Analog:** none in-repo. **Take the shape from 05-RESEARCH.md Pattern 3** (`PermissionBridge.handle` code example — params struct dig-out, option mapping, `RequestPermission` call, cancelled→-32021 error reply, unknown-optionId→deny, `opt.Response` verbatim) and the mapping table in the same section (allow_once→allow_once, allow_project→allow_always, deny→reject_once; unknown optionId → `{decision:"deny"}`; ACP cancelled or ctx-cancel → error reply code -32021; deadline expiry → deny reply).

**External cite-and-point (never extract):** zcode-src 3.17.1 `packages/shared/src/zcode-protocol/index.ts:2740-2753` (params), `:732-739` (option with embedded response), `interaction-broker.ts:241-243` (unknown→deny comment), `server-types.ts:183-189` (-32020/-32021/-32022).

**Error-code constants:** add -32020/-32021/-32022 to `internal/zcode/envelope.go` (:157-170 pattern — numeric values verbatim, "match the code, never the message").

### `internal/interact/cipherdecrypt.go` (NEW — CredentialDecrypt seam, real tier only, D-06)

**Analog:** none in-repo (first os/exec + crypto/aes use). **Take the shapes from 05-RESEARCH.md Pattern 6**: `secret-tool lookup service "ZCode Credential Encryption" username <sha256(realpath credentials.json)>` → 43-char base64url key → AES-256-GCM over `enc:v2:os:<kid>:<iv>.<tag>.<ct>` (segments iv.tag.ct, base64url); reply `{requestId, result:{ok:true, value:<decrypted>}}`; encrypt ops decline `{ok:false,...}`. Keep the policy injectable so OQ-1's decision is one wiring change (Pitfall 7). Keyring-locked/headless → tracked-no-answer + honest turn failure (documented). CI tiers never touch the keyring.

### `internal/hostsvc/snapshot.go` + `readback.go` (NEW — AUTH-01 execution)

**Analog:** `internal/acpfront/front.go` NewSession (:162-193) — the in-repo shape for a sequence of `conn.Call` backend requests with per-step `acpport.NewInternal`-style error wrapping. The snapshot push (`provider/updateAccountConfig`) and `provider/refreshRegistry` copy this Call + wrap-on-error pattern.

**Envelope fields** — from 05-RESEARCH.md Pattern 5 (strict envelope, `{}` providers/states valid, revision any non-empty string, `basedOnZCodeBuiltinRevision` = **reuse** the existing `supervisor.BuiltinRevision` formula — realdesktop_test.go:144 already calls it; never re-derive).

**awaitRegistryContains** — model on `machine.go` classifyIdlePatchLocked (:461-497): observe `state.updated`/create-reply payloads, dig camelCase keys via a params-style helper, poll-with-deadline until the model list contains the expected provider models; never trust the `received` ack (PITFALL 1).

### `internal/acpfront/auth.go` (NEW — Authenticate{zai-api-key})

**Analog:** replace the stub `agentAdapter.Authenticate` (agent.go:207-216 — the deliberate not-available error). Flow copies `front.go` NewSession's call sequence + honest error surfacing; **honesty gate** per PITFALL 3: record key → push snapshot → refreshRegistry → awaitRegistryContains → only then `{}` success; no configured key → honest coded error naming `ZCODE_ACP_API_KEY` (OQ4). Key enters via env/flag config (main.go:37-57 const pattern, :113-139 parseConfig pattern — flag wins over env).

### `internal/acpfront/lifecycle.go` (NEW — Resume/Close + teardown, WR-04/07)

**Analog:** `front.go` NewSession (:151-205) for resume (backend `session/resume` → same snapshot shape → `createSessionID` :301-321 digs `result.session.sessionId` unchanged → **re-issue subscribe** `{deliveryKind:"desktop-continuous"}` → register machine; NO setModel unless configured, NO replay). Teardown order per 05-RESEARCH.md Pattern 7: cancel active turn (machine.Cancel; bounded wait) → best-effort backend `session/close` → `registry.Remove` (core/registry.go:80-85 — exists, unused until now) → observer removes; lock with an observer-count-baseline test (WR-07 fix text).

### `internal/turn/machine.go` (EXTEND — WR-07 observer remove; WR-01 receiver)

**Analog:** itself. `NewMachine` (:182-197) today discards the remove funcs:
```go
if conn != nil {
    conn.AddObserver(m.ObserveEnvelope)
    conn.SetOnRouterEvent(m.observeRouterEvent)
}
```
Extend to return/expose both remove funcs (WR-07). `observeRouterEvent` (:435-453) becomes one receiver of the WR-01 fan-out (machines with state != running/cancelling early-return :443-445, so delivery to ALL machines is safe).

### `internal/zcode/conn.go` (EXTEND — WR-01 router-event fan-out)

**Analog:** itself — mirror `AddObserver` (:199-218) exactly: entry-pointer identity (`observerEntry` :223-227 — function values are not comparable), ordered delivery under one mutex, remove func with `slices.Delete`, "empty list = silent drop" doc contract. Deliver router events the way `deliverNotification` (:311-318) iterates; fold `SetOnRouterEvent` single slot into the fan-out (keep `routerMu` read discipline :456-464 or the fan-out mutex — one mechanism, preferred). Do NOT touch routeRequest/demux/pending (03-02 test-locked).

### `internal/acpport/agent.go` (EXTEND — WR-02 atomic sink; Resume/Close delegates)

**Analog:** itself. WR-02: `sink UpdateSink` (:28-30) becomes `atomic.Pointer[UpdateSink]`; `bindSink` (:273-275) stores; `sinkForPrompt` (:280-288) loads — nil-fallback loud error preserved. Resume/Close: replace `unadvertised(...)` stubs (:224-229, :248-253) with delegations following the `Prompt`/`NewSession` delegate shape (:110-117, :154-173) + `wireError` coding (:295-306 — RequestError stays unwrapped).

### `internal/acpport/types.go` + `sink.go` (EXTEND — tool notes, TOOL-01)

**Analog:** themselves. Extend `SessionNoteKind` (:87-93) with a tool note carrying `{ToolCallID, IsStart, Title, Name, Kind, Status, Diff{Path,OldText,NewText}, Locations, RawInput, RawOutput}` — same neutral-vocabulary discipline (SDK types never cross the seam). `sink.go` `noteToNotification` (:29-44) gains the tool arms mapping onto `acp.StartToolCall`/`acp.UpdateToolCall`/`acp.ToolDiffContent` — same "unknown kind fails loud" default (:42). Field values from 05-RESEARCH.md Pattern 4 mapping tables (kind/status taxonomy, toolName→ToolKind, diff from edit/write inputs, absolute-path resolution).

### `internal/acpfront/stream.go` (EXTEND — tool.updated classification; WR-06)

**Analog:** itself. The tap (:40-89) currently switches on part.delta fields; extend the session/event parse to `type:"tool.updated"` discriminated-union kinds (scheduled/started/progress/result/error — batch/raw and subagent mirrors log-only). Same rules: read-goroutine-must-not-block, turn-slot emitter lookup (:70-72), structural warn on failed write. WR-06: the dropped-block warn for unsupported content lands at the drop site (kinds+counts only). IN-05: record the turn's id in the slot and skip foreign-turnId deltas here.

### `internal/acpfront/permission.go` (NEW — bridge impl) + `turnctx.go` reuse

**Analog:** `turnctx.go` acquireTurn (:31-63) — the emitter-closure pattern: the bridge's pending map (keyed by backend requestId, holding the in-flight `conn.RequestPermission` ctx+cancel) publishes under a mutex so the tap/cancel see complete-or-none state. Cancel fan-in copies front.go Prompt (:238-244): ACP `session/cancel` additionally cancels ALL pending bridges of the turn (→ -32021 replies).

### Fake corpus (EXTEND + NEW scenarios)

**Analog:** `internal/zcode/fake/fake.go` `runReverse` (:241-268) — the re-announce extension point is the fresh-id allocation:
```go
id := &zcode.ID{Str: fmt.Sprintf("fake-%d", s.nextRev.Add(1))}
```
Extend `Reverse` (scenario.go:70-75) with a re-announce field (count/interval) that re-issues the SAME params under NEW wire ids; `expect_answer` stays on the first (research: "A re-announce variant repeats the same reverse with identical params.requestId... expect_answer stays on the first"). Semantic compare (`matchExpectAnswer` :327-352) is the assertion; golden recording via `golden.go` CompareGolden (:29-58) + `TestGolden` (golden_test.go:182).

**New scenarios** copy `runtime-preferences.json` (reverse with expect_answer on create) and `credential-cipher-scripted.json` (mid-turn reverse on session/send): permission-allow / permission-deny / permission-cancel / reannounce-dedupe / runtime-headers-answer / user-input-decline / tool-lifecycle (+ goldens). The permission-allow skeleton is in 05-RESEARCH.md (fake scenario code example — tool.updated emit + interaction/requestPermission reverse with embedded-response options).

### `internal/acpfront/harness_test.go` (EXTEND)

**Analog:** itself. `RecordingClient.RequestPermission` (:198-204) is currently a no-op — extend to a scripted outcome (configurable selected/cancelled + delay) with capture of received requests; tool-note capture rides `Updates()` (:146-167, already collects `acp.SessionUpdate`). `ComposeFront` (:330-380) needs no structural change — new tests compose via `newAgent` callbacks like `newTracerFront` (loop_test.go:63-87) and assert wire order via `teeLineIndexes` (loop_test.go:97-124).

### `test/realdesktop/` (NEW — guarded api-key full cycle)

**Analog:** `internal/supervisor/realdesktop_test.go` — copy the whole guard discipline: `//go:build realdesktop` (:1), env double-guard skip (:80-83, `ZCODE_ACP_REAL_E2E=1`), canonical command in the file header (:10-13), discover→spawn→exchange→staged-shutdown composition (:87-101), spawn through production `SpawnChild` (:174-182), and `supervisor.BuiltinRevision` usage (:144). The full cycle adds: api-key config before turn (env-sourced, never committed), responder wired into the conn, one tool-less turn + one file-read turn, scrubbed capture + V-EXP id + quota ledger (D-05).

### `cmd/zcode-acp/main.go` (EXTEND)

**Analog:** itself. New env const `ZCODE_ACP_API_KEY` beside :37-40; parseConfig-style validation (:113-139); in `run` (:155-233) wire the responder: `conn.OnRequest = responder.Serve` before any frame flows (the Conn hook contract, conn.go:97-101), build the policy table with the CredentialDecrypt seam injected (OQ-1 decision = one wiring change here).

## Shared Patterns

### Reverse-request replies ride the bypass lane
**Source:** `internal/zcode/call.go:219-249` (Reply/ReplyError → `out.enqueueBypass`; writer.go:127-130)
**Apply to:** every responder handler and permission-bridge reply. Never the queue lane.

### Numeric-code matching, message text never matched
**Source:** `internal/zcode/envelope.go:157-170` (codes) + machine.go:130-137 (the -32010 mapping)
**Apply to:** -32601 default, -32020/-32021/-32022 bridge codes, -32018 handling. Add new constants in envelope.go's code block.

### camelCase wire-key dig-out (tagliatelle-safe)
**Source:** `internal/turn/machine.go:587-600` (paramsString), `internal/acpfront/front.go:301-321` (createSessionID map-decode)
**Apply to:** Request normalization (params.requestId/sessionId), permission params, tool.updated payloads, runtime-headers params, snapshot/ack decode.

### Bounded map + delete-under-lock exactly-once
**Source:** `internal/zcode/call.go:53-117` (pendingTable resolve/drainAll)
**Apply to:** dedupe table, permission pending map, registry teardown.

### Structural logging only (id keys, durations, methods — never params)
**Source:** `internal/zcode/conn.go:436-450`, `internal/acpfront/stream.go:87`
**Apply to:** dedupe joins, deadline expiries, dropped blocks, tool notes, redaction list per 05-RESEARCH.md Security V7 (apiKey/token/enc:v2:*).

### Coded seam errors, never raw transport noise
**Source:** `internal/acpport/types.go:140-155` (New* helpers) → `agent.go:295-306` (wireError, RequestError unwrapped)
**Apply to:** authenticate, resume, close, permission failures.

### Assign/register-before-traffic hook contract
**Source:** `internal/zcode/conn.go:84-120` (hook docs), `front.go:96-120` (tap before machines)
**Apply to:** responder OnRequest wiring, WR-01 fan-out registrations, harness composition.

### Real-tier guard: build tag + env double lock
**Source:** `internal/supervisor/realdesktop_test.go:1-19, 79-83`
**Apply to:** `test/realdesktop/` — CI never contacts the desktop or the account (D-05).

## No Analog Found

| File | Role | Data Flow | Reason / Substitute |
|---|---|---|---|
| `internal/interact/cipherdecrypt.go` | service | file-I/O (subprocess+GCM) | No in-repo os/exec+crypto precedent; method is 3.18-only (absent from the 3.17.1 clone). Use 05-RESEARCH.md Pattern 6 field-verified shapes verbatim; stdlib only. |
| `internal/interact/permission.go` (backend bridge) | service | event-driven | First client→agent round-trip in the repo. Use 05-RESEARCH.md Pattern 3 code example + mapping table; SDK `RequestPermission` owns framing (Don't-Hand-Roll row). |

## Metadata

**Analog search scope:** internal/{zcode,zcode/fake,acpfront,acpport,turn,core,supervisor}, cmd/zcode-acp, test/
**Files read in full:** conn.go, call.go, envelope.go, front.go, stream.go, turnctx.go, machine.go, registry.go, types.go, sink.go, agent.go, scenario.go, fake.go, golden.go, harness_test.go, realdesktop_test.go, main.go (+ corpus JSONs, loop_test.go excerpts)
**External anchors (cite-and-point, never extract):** `/home/nil/DiskD/W/Djarvur/zcode-src` 3.17.1 clone — `interaction-broker.ts:60,241-243`, `server.ts:1053-1152,162`, `server-types.ts:183-189`, `permission-options.ts:66-106,108-125`, `packages/shared/src/zcode-protocol/index.ts:732-739,1528,1738-1812,2007-2029,2427-2435,2590-2621,2740-2753,2846-2915`, `adapters/src/model/runner.ts:186-221`, `runner-runtime-headers.ts:24-28`, tool handlers `read.ts:569 glob.ts:87 grep.ts:143 bash.ts:488 edit.ts:86,258 write.ts:216,229 todo.ts:78`
**All analog paths git-tracked:** verified via `git ls-files` this session
**Pattern extraction date:** 2026-11-20

## PATTERN MAPPING COMPLETE
