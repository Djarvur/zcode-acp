# Phase 5: Interactions & Full Chat Cycle (api-key) - Research

**Researched:** 2026-11-20
**Domain:** host-managed interaction responder (CORE-06), tool-call streaming (TOOL-01), permission bridging (TOOL-02), api-key auth path (AUTH-01), session resume/close (SESS-01), plus the 12 routed Phase-4 review findings
**Confidence:** HIGH — every load-bearing wire shape was re-read this session from the 3.17.1 source clone (file:line quoted), the shipped 3.18.2 bundle (grep counts re-derived, pristine `cmp` re-run), the coder/acp-go-sdk v0.20.0 module source, the in-repo Phase-4 code, the ACP v1 official spec pages, and the machine-local field ground truth (FINDINGS.md + field-notes + captures). The few genuinely unverifiable items are tagged `[ASSUMED]`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** The responder is a **policy object** per ARCHITECTURE (Pattern 3): every
  server→host method gets a handler chosen by auth mode; **per-method deadlines**
  (180 s for requestProviderRuntimeHeaders per Phase-1-verified transport facts; 15 s
  requestRuntimePreferences already landed); **dedupe 1 s re-announcements by
  requestId** (Phase-1-verified broker behavior); unknown methods answered with a
  protocol error (-32601-class) — never crash (CORE-06). Phase-3's
  requestRuntimePreferences static answer migrates into this policy object (same
  behavior, new home).
- **D-02:** AUTH-01 answers `provider/updateAccountConfig` with the api-key snapshot
  (exact-revision builtin per install-discovery.md formula) and
  `interaction/requestProviderRuntimeHeaders` with `{headersApplied:true,
  requestAuth:{apiKey, apiKeyId}}` — the PAT comes from the account config the user
  supplies via the new `authenticate{zai-api-key}` front method. **Verification = read
  `provider/refreshRegistry`/registry state back**, never the `received` ack (PITFALL 1).
  Requires ONE real account round-trip to prove end-to-end (success criterion 4) —
  quota-minimal (registry read-back + one tiny tool-less turn), the Phase-1 UAT AR
  precedent covers the spend format.
- **D-03:** Tool lifecycle events map to `tool_call` (start) / `tool_call_update`
  (progress/completion) with title/kind/status from the dialect's event taxonomy;
  **file diffs for edit tools** rendered as ACP diffs (TOOL-01). Permission bridging
  maps `interaction/requestPermission` options to ACP
  `session/request_permission` option kinds; the client's decision forwards to the
  backend; **ACP-side cancel resolves the pending request as `cancelled`** and the
  backend turn ends (TOOL-02). Real-tool verification: one tiny real turn exercising a
  file-read tool on the api-key account (quota-minimal, same AR precedent).
- **D-04:** SESS-01 lands WITH the Phase-4-review teardown work: `session/resume`
  reattaches (backend `session/resume` — no replay), `session/close` runs child-side
  cleanup + releases registry/machine/observer (04-REVIEW WR-04/WR-07: NewMachine
  exposes observer remove). `session/load`/fork stay deferred (v2/Phase 7).
  Phase-4 review items land here by scope: WR-01 (per-session OnRouterEvent via
  fan-out), WR-02 (atomic sink publish), WR-05/06 (wording + dropped-block logging).
- **D-05:** Real api-key operations follow the Phase-1 AR precedent: minimal turns,
  ledgered in experiments/README quota section, **CI never** runs them (guarded manual
  tier); credentials from local config (never committed); every real-account claim
  gets a scrubbed capture + V-EXP id (evidence discipline unchanged).

### Claude's Discretion

- Policy-object API shape; tool-event mapping details within the dialect taxonomy;
  test design (fake-driven permission/tool scenarios + the real-account smoke tier)

### Deferred Ideas (OUT OF SCOPE)

- OAuth chain + keyring vault (Phase 6); configOptions/model picker (Phase 6)
- elicitation/create preference (Phase 5+ if client capability-gated — needs a real
  client to test; defer to Phase 7 integration)
- session/load/fork (v2)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CORE-06 | Host-managed interaction responder always on: every `interaction/*` answered promptly (dedupe re-announcements, per-method deadlines, error replies to unknown methods instead of crash) | Responder policy-object API (Architecture Patterns 1–2); re-announce wire mechanics (new wire id per re-announce, same `params.requestId`, interval 1 s doubling to a 10 s cap — Architecture Pattern 2); per-method deadline table; conn.OnRequest wiring seam |
| AUTH-01 | api-key path: `provider/updateAccountConfig` + `interaction/requestProviderRuntimeHeaders` answered correctly (registry state verified, not the ack) | Snapshot schema (strict envelope, `providers`/`states` records accept `{}`), revision formula, requestAuth response schema verbatim, read-back surfaces, key sourcing constraint (ACP `authenticate` carries no credentials) |
| TOOL-01 | `tool_call` / `tool_call_update` streamed with title, kind, status transitions, and file diffs for edit tools | `tool.updated` kind taxonomy (scheduled/started/progress/result/error/batch/raw) with verbatim payload schemas; toolName→ToolKind mapping; diff rendering from edit-tool inputs (`{file_path, old_string, new_string}` / write `{file_path, content}`) |
| TOOL-02 | `session/request_permission` bridged with `allow/reject` options and `cancelled` outcome on ACP-side cancel | zcode option schema (each option carries its pre-built `response`), decision enum `allow|deny|escalate|modify`, ACP option kinds, outcome forwarding + cancel mapping (-32021), deadline-expiry deny policy |
| SESS-01 | `session/resume` (no replay) and `session/close` (child-side cleanup) | Backend resume params/result (same snapshot shape as create), close params/result `{closed?}`, teardown contract incl. NewMachine observer-remove (WR-04/WR-07) |
</phase_requirements>

## Summary

Phase 5 turns the Phase-3/4 plumbing into the first full chat cycle on a real api-key
account. The center of gravity is the **interaction responder**: the app-server runs
in host-managed mode and blocks mid-turn on server→host requests; Phase 3's inline
router (two hardcoded cases + a -32601 default in `internal/zcode/conn.go`) migrates
into a policy object with a per-method handler table, per-method deadlines, and
**dedupe keyed on `params.requestId`**. Source truth refined the locked D-01 picture in
one important way: each re-announce is a **NEW wire request with a NEW wire id**
(`server-<n>`), carrying the SAME `params.requestId`, at an interval that starts at
1 s and doubles to a 10 s cap — and the server registers every outstanding wire id of
one pending interaction until the FIRST reply resolves it and silently drops the rest.
The dedupe design follows directly: one pending entry per `(method, requestId)`, reply
once to the registered wire id, ignore superseded re-announce envelopes.

Two findings materially affect planning. **(1) The real-tier turn cannot complete
without answering `interaction/requestCredentialCipher`.** Phase 1 proved live on this
machine that a real `session/send` fires a `decrypt` cipher ask mid-turn and the turn
stalls while it is unanswered (the D-05 degradation cause); the field-verified shim
answers it via `secret-tool` + stdlib AES-256-GCM, and real turns then complete
(APIKEY_PATH_OK discipline). Phase 3's "tracked-no-answer" therefore cannot stand on
the real-account tier — the planner must choose a Phase-5 answer strategy (this
research recommends the secret-tool seam behind the CredentialVault interface shape,
keeping godbus out until Phase 6). **(2) `interaction/requestProviderRuntimeHeaders`
is an account-provider-only ask** — the model runner's gate makes the refresh callback
`undefined` for non-`zhipu-account` providers, and no api-key capture or shim run ever
saw it fire — so AUTH-01's runtime-headers half is exercised at the fake tier plus as
a defensive responder entry, while the REAL proof of AUTH-01 is the snapshot push +
registry read-back + a working turn. Both findings are developed below with sources.

**Primary recommendation:** build the responder as `internal/interact` wired through
the existing `Conn.OnRequest` seam; extend the front's tap into a small tool-call
translator feeding new neutral `acpport` note kinds; add the permission bridge with
option-verbatim forwarding and cancelled→-32021; land SESS-01 together with the
WR-04/07 teardown (NewMachine exposes its observer remove func); and settle the
credential-cipher real-tier strategy before planning the real tier.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| interaction/* answering, deadlines, dedupe (CORE-06) | Backend Conn tier (`internal/interact` policy object over `internal/zcode`) | — | The responder rides the back-leg conn's reverse-request path; it must never depend on the ACP front being healthy |
| Tool lifecycle → tool_call streaming (TOOL-01) | Front tier (`internal/acpfront` tap + `internal/acpport` sink) | `internal/zcode` (envelope decoding only) | Tool events are client presentation over `session/event` notifications; the turn machine stays completion-only |
| Permission bridging (TOOL-02) | Backend Conn tier (`internal/interact`) + Front tier (ACP `session/request_permission`) | `internal/acpport` (RequestPermission sender) | Backend side answers the reverse request; front side owns the client round-trip and cancel fan-in |
| api-key auth (AUTH-01) | Front tier (`authenticate` handler in `internal/acpfront`) + `internal/interact` (headers/`updateAccountConfig` execution) | `cmd/zcode-acp` (key sourcing via env/flag) | Auth state is process-global (registry is process-global); the snapshot push and read-back are backend-tier operations |
| Session resume/close + teardown (SESS-01, WR-04/07) | Front tier (`internal/acpfront`) + `internal/core` registry | `internal/turn` (Machine exposes observer remove) | ACP surface methods; teardown releases registry entries, conn observers, machines |
| Phase-4 WR-01/02/03/05/06 fixes | `internal/zcode` (router fan-out), `internal/acpport` (atomic sink), `internal/acpfront` (streaming pump, wording, block logging) | — | Each fix lives where the defect lives (04-REVIEW file map) |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`context`, `sync`, `encoding/json`, `crypto/aes`, `os/exec`) | go 1.27 | Responder, dedupe table, deadlines, (optional) cipher decrypt | Zero-dep discipline (CONTEXT); everything needed is stdlib [VERIFIED: in-repo go.mod — `module github.com/Djarvur/zcode-acp`, `go 1.27`, single require] |
| github.com/coder/acp-go-sdk | v0.20.0 (pinned, schema 1.25.0) | `AgentSideConnection.RequestPermission` sender, `StartToolCall`/`UpdateToolCall`/`ToolDiffContent` helpers, `RequestPermissionOutcome` union | Already the front-leg seam behind `internal/acpport`; all Phase-5 ACP surfaces verified present in the module source [VERIFIED: acp-go-sdk v0.20.0 agent_gen.go:402-404, helpers.go:62-71,119-141,194; types_gen.go:6835-6878,8041-8070,13097-13162,13191-13227,13285-13360,13492-13510,9978-10002] |
| internal/zcode (codec + conn) | in-repo | Envelope grammar, bypass-lane replies, OnRequest intercept seam, RouterEvent record | Phase-3/4 landed; the responder extends it, never bypasses it [VERIFIED: in-repo internal/zcode/conn.go:91-101,325-358; writer.go:127-130] |
| internal/zcode/fake | in-repo | Scripted app-server for every new scenario tier | Reverse-exchange scripting with `expect_answer` already proven (runtime-preferences scenario) [VERIFIED: in-repo internal/zcode/fake/scenario.go:54-75; fake/testdata/scenarios/runtime-preferences.json] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `secret-tool` (libsecret CLI) + stdlib `crypto/aes` GCM | system binary | Credential-decrypt answers on the REAL tier only (option — see Open Question 1) | Field-verified shim parity: `secret-tool lookup service "ZCode Credential Encryption" username <sha256(realpath credentials.json)>`, then AES-256-GCM over `enc:v2:os:<kid>:<iv>.<tag>.<ct>` (segments iv.tag.ct, base64url) [V-RPT field-notes app-server-stdio-protocol.md + shim source, quoted in Architecture Pattern 6] |
| godbus/dbus/v5 | v5.2.2 | NOT in Phase 5 — keyring vault is Phase 6 | Locked deferral (CONTEXT: godbus deferred to Phase 6; zero-dep discipline) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `Conn.OnRequest` seam for the responder | Replacing conn's built-in `routeRequest` switch | OnRequest keeps Phase-3 behavior as the fallback when unwired and makes the policy object a pure composition concern; replacing would churn test-locked router contracts (03-02 locks) for no gain |
| Forwarding `interaction/requestUserInput` as ACP permission | Answering `{action:"decline"}` | Decline is schema-valid, keeps the turn alive, and matches the locked deferral of elicitation to Phase 7; a permission-shaped misuse would render questions as allow/reject |
| godbus now for cipher decrypt | secret-tool subprocess (real tier) | godbus is the Phase-6 locked design; secret-tool is the field-verified shim mechanism, needs no new Go module, and Phase 6 swaps it out behind the same interface |

**Installation:** none — zero new dependencies (Package Legitimacy Audit below).

**Version verification:** `go list -m all` re-run this session prints exactly
`github.com/Djarvur/zcode-acp` + `github.com/coder/acp-go-sdk v0.20.0` — no new
modules. acp-go-sdk v0.20.0 was verified in Phase 4 (STACK.md: three independent
sources) and is present in the module cache read this session.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| coder/acp-go-sdk v0.20.0 | Go module proxy (NOT npm) | since 2026-11-18 | n/a (Go) | github.com/coder/acp-go-sdk | OK (Phase-4 vetted; seam `npm` check returns SUS on registry mismatch — the package does not live on npm; verdict is an ecosystem artifact, not a signal) | Already pinned in go.mod; no action |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none (the single seam output above is an
ecosystem-mismatch artifact; `go list -m all` confirms the module graph is exactly
the project + the SDK — zero transitive deps)

*Expected zero new dependencies confirmed: Phase 5 installs nothing. The optional
real-tier cipher strategy uses the system `secret-tool` binary (verified present on
this machine) — an external runtime dependency for the guarded manual tier only, never
a Go module and never CI.*

## Architecture Patterns

### System Architecture Diagram

```
 ACP client (Orca / Zed / test RecordingClient)
     │ ACP v1 JSON-RPC over stdio (newline JSON)
┌────▼──────────────────────────────────────────────────────────────────┐
│ zcode-acp                                                             │
│                                                                       │
│  acpport agentAdapter (SDK)                                           │
│   • authenticate{zai-api-key} → Front.Authenticate  (NEW, AUTH-01)    │
│   • ResumeSession / CloseSession → Front  (NEW, SESS-01)              │
│   • SessionUpdate: tool_call / tool_call_update (NEW, TOOL-01)        │
│   • RequestPermission(ctx, req) → outcome  (NEW client round-trip)    │
│         ▲ answers arrive async; cancel → outcome cancelled            │
│  acpfront Front                                                       │
│   • tap (observer #1): part.delta → chunks  (landed)                  │
│   • tap EXTENDED: tool.updated kinds → tool notes (NEW)               │
│   • turn slot / cancel fan-in  (landed; WR-05 wording fix)            │
│   • teardown: close → registry.Remove + observers + machine (NEW)     │
│         │ typed Requests (session/create|resume|close|send|stop…)     │
│  internal/core registry ── WR-07: entries removable; WR-02 sink atomic│
│         │                                                             │
│  zcode.Conn ── read loop demux ──┬─ replies → pending table            │
│                                  ├─ notifications → observer fan-out  │
│                                  │     (WR-01: + router-event fan-out) │
│                                  └─ server→host requests              │
│                                        │ OnRequest seam               │
│  internal/interact Responder (NEW policy object — CORE-06)            │
│   • handler table {deadline, dedupe, handler} per method              │
│   • dedupe by (method, params.requestId); bounded map                 │
│   • requestRuntimePreferences  (migrated static answer, 15 s)         │
│   • requestPermission         → PermissionBridge (forward/cancel)     │
│   • requestProviderRuntimeHeaders → api-key answer (180 s)            │
│   • requestUserInput → {action:"decline"}                             │
│   • requestCredentialCipher → policy decision (OQ-1)                  │
│   • officialMcpAuthHeaders / browser* → -32601 (probe-verified safe)  │
│   • unknown method → -32601 naming the method (never crash)           │
│   • answers write on the BYPASS lane (never queued behind prompts)    │
│         │ child stdin/stdout — ZCode JSONL (no jsonrpc key)           │
└────────▼──────────────────────────────────────────────────────────────┘
     zcode app-server (Electron-as-node): serial queue, stop bypass,
     blocks mid-turn on unanswered interaction/* (host-managed mode)
```

Primary use case trace (success criterion 5): `session/new` (create+subscribe+setModel,
host obligations answered by the responder) → `session/prompt` (send ack → events
stream: chunks via the tap, `tool.updated` → tool_call/update, permission ask →
responder → ACP `session/request_permission` → client outcome → backend reply) →
prompt returns `stopReason` → `session/resume` (reattach, no replay) → `session/close`
(teardown: backend close + registry/observer/machine release).

### Recommended Project Structure

```
internal/
├── interact/            # NEW: Responder policy object, dedupe table, per-method
│   │                    # policies; PermissionBridge interface (impl in acpfront wiring)
│   └── responder_test.go
├── acpfront/
│   ├── auth.go          # NEW: Authenticate{zai-api-key} (key from config, never wire)
│   ├── lifecycle.go     # NEW: ResumeSession / CloseSession + teardown (WR-04/07)
│   ├── stream.go        # EXTEND: tool.updated classification → tool notes
│   └── permission.go    # NEW: PermissionBridge impl over acpport sink + pending table
├── acpport/
│   ├── agent.go         # EXTEND: Authenticate real impl; Resume/Close delegate; WR-02 atomic sink
│   ├── sink.go          # EXTEND: tool-note → StartToolCall/UpdateToolCall/ToolDiffContent
│   └── types.go         # EXTEND: neutral ToolNote vocabulary + PermissionOption types
├── turn/machine.go      # EXTEND: expose observer remove func (WR-07); router fan-out receiver
├── zcode/conn.go        # EXTEND: AddRouterObserver fan-out (WR-01); nothing else moves
└── hostsvc/             # NEW-if-OQ1: CredentialDecrypt service (secret-tool seam), Phase-6 vault shape
```

### Pattern 1: Responder policy object wired through the existing OnRequest seam

**What:** `internal/interact` owns a `Responder` with a per-method handler table; the
production wiring assigns it to `Conn.OnRequest` (the intercept seam Phase 3 already
landed, returning `handled=true` for everything — CORE-06 means the responder answers
every method, including the unknown ones, via its own default).

**When to use:** always — the conn's built-in router stays as the fallback when
`OnRequest` is nil, so Phase-3 test-locked contracts are untouched.

**Example:**

```go
// Source: in-repo internal/zcode/conn.go:91-101 (seam), :325-336 (handleRequest)
// The seam exists: OnRequest intercepts before the built-in router.
// OnRequest func(ctx context.Context, env zcode.Envelope) (result any, werr *zcode.WireError, handled bool)

// internal/interact/responder.go (NEW)
type Answer struct {
    Result any              // marshaled into the reply envelope
    Err    *zcode.WireError // non-nil → error reply (Code* constants, never message matching)
}

// Request is the normalized reverse request; ReqID/SessionID are dug out of
// params once, at dedupe time (the wire keys are camelCase — same dig-out
// discipline as machine.go paramsString).
type Request struct {
    Method    string
    ID        *zcode.ID // the wire id the reply must echo
    Params    json.RawMessage
    ReqID     string    // params.requestId, "" when absent
    SessionID string    // params.sessionId, "" when absent
}

type Handler func(ctx context.Context, req Request) (Answer, error)

type MethodPolicy struct {
    Deadline time.Duration // 0 = no deadline (permission: wait for client/cancel)
    Handler  Handler
}

type Responder struct {
    methods map[string]MethodPolicy
    // unknown: -32601 naming the method (conn.go routeRequest default, preserved)
}

func (r *Responder) Serve(ctx context.Context, env zcode.Envelope) (any, *zcode.WireError, bool)
```

Per-method policy table (the CORE-06 contract, all values sourced below):

| Method | Deadline | Dedupe | Behavior |
|--------|----------|--------|----------|
| `session/requestRuntimePreferences` | 15 s (migrated) | by requestId (params carry it, scope differs — dedupe key `(method, requestId)`) | static `{nativeSearchEnhancementsEnabled:false, memoryEnabled:false}` — same bytes as today [VERIFIED: in-repo internal/zcode/conn.go:384-387; V-EXP EXP-WORKSPACEKEY-01] |
| `interaction/requestPermission` | configurable; recommended default 300 s (see OQ-2); expiry → `deny` reply | **by (method, params.requestId) — mandatory** (re-announce mechanics, Pattern 2) | forward to ACP `session/request_permission`; forward outcome; ACP cancel → error reply `-32021` |
| `interaction/requestProviderRuntimeHeaders` | 180 s (D-01; server holds the same window) | by (method, params.requestId) | `{headersApplied:true, requestAuth:{apiKey, apiKeyId}}` from the authenticate-sourced key; never `headersApplied:false` (PITFALL 3) |
| `interaction/requestUserInput` | 15 s (host policy; server default has none) | by (method, params.requestId) | `{action:"decline"}` + structural slog (honest no-UI-channel; elicitation deferred to Phase 7) |
| `interaction/requestCredentialCipher` | — | n/a | **OQ-1 decision**: tracked-no-answer (status quo, blocks the real tier) vs decrypt answer (recommended for the real tier) |
| `interaction/requestOfficialMcpAuthHeaders` | n/a | n/a | keep `-32601` — field-verified harmless (6–7 asks per create answered -32601 without breaking the flow) [V-EXP EXP-SUBSCRIBE-01] |
| `interaction/browserList` / `browserExecute` | n/a | n/a | `-32601` default (never exercised live; shapes source-level only) |
| unknown `interaction/*` or any other method | n/a | n/a | `-32601` `"unsupported host request: " + method` (verbatim preserved behavior) [VERIFIED: in-repo internal/zcode/conn.go:352-357] |

### Pattern 2: Re-announce dedupe — the source-verified wire mechanics

**What the server actually does** (all read this session from the 3.17.1 clone,
corroborating-only per D-01 but mechanism-matched by the bundle literals):

- The interaction broker re-announces pending asks with
  `INTERACTION_REQUEST_REANNOUNCE_INTERVAL_MS = 1_000` [VERIFIED: zcode-src 3.17.1
  `interaction-broker.ts:60` — verbatim `const INTERACTION_REQUEST_REANNOUNCE_INTERVAL_MS = 1_000;`]
- `requestClient` allocates a **fresh wire id per send**: `const id = \`server-${this.nextClientRequestId++}\`; pending.requestKeys.add(key);` and the re-announce timer re-invokes `sendClientRequest()` — every re-announce is a NEW wire request carrying the SAME `params` [VERIFIED: zcode-src 3.17.1 `server.ts:1053-1057` — quoted `const id = \`server-${this.nextClientRequestId++}\``; `:1094-1118` — `scheduleReannounce` doubles: `nextReannounceIntervalMs = Math.min(nextReannounceIntervalMs * 2, MAX_CLIENT_REQUEST_REANNOUNCE_INTERVAL_MS)` with `MAX_… = 10_000` at `server.ts:162`]
- ALL outstanding wire ids of one pending interaction stay registered until the FIRST reply resolves it: `resolveClientRequest` → `cleanupClientRequest` deletes every key in `pending.requestKeys`; replies to already-cleaned ids hit `if (!pending) { return; }` — silently dropped [VERIFIED: zcode-src 3.17.1 `server.ts:1120-1152`]
- The reply is validated against the method's result schema: an invalid reply **rejects** the pending request ("Invalid response") — the shim's `{headers:{}}` stub for requestProviderRuntimeHeaders would have failed this way, which is itself evidence the ask never fired on api-key runs [VERIFIED: zcode-src 3.17.1 `server.ts:1124-1134` — `pending.resolve(pending.resultSchema.parse(result))` / reject `Invalid response: ${pending.method}`]

**Host-side consequence:** one pending dedupe entry per `(method, params.requestId)`;
the FIRST arriving envelope registers its wire id; re-announces join the entry
(optionally refreshing a deadline); the handler's answer replies **once** to the
registered wire id. Do NOT reply to every re-announce individually — the first reply
resolves the whole pending set and later replies are dropped by the server (harmless
but noisy). The dedupe map is **bounded**: entries evict on answer, on conn close, and
on a per-entry TTL (deadline + grace); a structural slog line records each joined
re-announce (id keys only). This is also the DoS bound: a server that re-announces
forever must grow the map by exactly one entry per distinct requestId, not one per
wire frame.

### Pattern 3: Permission bridging (TOOL-02), both directions

**Backend shapes** (read this session; every value verbatim):

- Request params (`zcodePermissionRequestParamsSchema`) [VERIFIED: zcode-src 3.17.1
  `packages/shared/src/zcode-protocol/index.ts:2740-2753`]: `{requestId,
  sessionId, turnId?, toolCallId, toolName, reason, riskLevel:
  "low"|"medium"|"high"|"critical", input: unknown, origin?, options:
  [{optionId, kind, name, description?, response}] (min 1)}` — strict.
- Each option **embeds its own pre-built response** [VERIFIED: zcode-src 3.17.1
  `index.ts:732-739` — `zcodePermissionOptionSchema = z.object({ optionId:
  nonEmptyString, kind: nonEmptyString, name: nonEmptyString, description:
  z.string().optional(), response: zcodePermissionResponseSchema }).strict()`]
- The response schema [VERIFIED: zcode-src 3.17.1
  `packages/shared/src/zcode-protocol-legacy-types.ts:44,67-74`]: decision enum
  `z.enum(["allow", "deny", "escalate", "modify"])`; response object
  `{decision, reason?, modifiedInput?, permissionUpdates?}` strict. The field-verified
  minimal accepted answer is `{decision:"allow"}` [V-RPT field-notes app-server-stdio-protocol.md — `interaction/requestPermission` → `{decision:"allow"}`]
- Option kinds the CLI synthesizes [VERIFIED: zcode-src 3.17.1
  `apps/zcode-cli/packages/bootstrap/src/permission-options.ts:66-106`]: `allow_once`
  (optionId `allow_once`, response `{decision:"allow", reason:"Approved once"}`),
  `allow_always` (optionId `allow_project`, response carries `permissionUpdates`),
  `deny` (optionId `deny`, response `{decision:"deny", reason: PERMISSION_DENIED_BY_USER_CONTENT}`). On the legacy reverse RPC the session-allow variant is stripped (`toLegacyPermissionOptionsPolicy` maps both always-policies to `no-always-allow` → no always option) — the wire vocabulary is exactly `allow_once` / `allow_always` / `deny` [VERIFIED: same file `:108-125`; bundle literals re-derived this session: `allow_once` ×5, `allow_always` ×4, `allow_project` ×1, `allow_session` ×1]
- The backend maps a selected option by optionId and **unknown optionIds fall to
  deny**: `return { decision: "deny", reason: buildPermissionDeniedContent(answer.freeText) };`
  [VERIFIED: zcode-src 3.17.1 `interaction-broker.ts:241-243` — comment verbatim
  `// deny/rejectOnce/rejectAlways、未知 optionId、无 optionId 全部落 deny。`] — the
  wrapper mirrors this posture on its own ACP side.
- Error codes the broker understands in guarded mode: cancelled `-32021`, timedOut
  `-32022`, noClientAttached `-32020` [VERIFIED: zcode-src 3.17.1
  `server-types.ts:183-189` — `PROTOCOL_CLIENT_REQUEST_ERROR_CODES = { noClientAttached: -32020, cancelled: -32021, timedOut: -32022 }`], mapped to `ToolCancelled` / `PermissionTimeout` / `ConfigurationError` respectively [VERIFIED: `interaction-broker.ts:79-91`]

**ACP side** (SDK v0.20.0 + official spec, cross-checked):

- Agent sends `session/request_permission` via
  `conn.RequestPermission(ctx, RequestPermissionRequest{SessionId, Options: []PermissionOption{{Kind, Name, OptionId}}, ToolCall: ToolCallUpdate}) → RequestPermissionResponse{Outcome}` [VERIFIED: acp-go-sdk v0.20.0 `agent_gen.go:402-404`; `types_gen.go:8171-8198` request fields, `:8041-8070` outcome union `{cancelled} | {selected, optionId}`]
- Option kinds: `allow_once | allow_always | reject_once | reject_always` [VERIFIED: acp-go-sdk v0.20.0 `types_gen.go:6866-6878` — constants quoted `PermissionOptionKindAllowOnce PermissionOptionKind = "allow_once"` … `PermissionOptionKindRejectAlways PermissionOptionKind = "reject_always"`]; spec: kind is a UI hint only [CITED: agentclientprotocol.com/protocol/v1/tool-calls.md]
- Cancellation: when the turn is cancelled while a permission request is outstanding, the client answers `{outcome:{outcome:"cancelled"}}`; the agent must accept it [CITED: agentclientprotocol.com/protocol/v1/tool-calls.md — client MUST respond cancelled; SDK doc comment on `RequestPermissionOutcome.Cancelled`]

**Mapping table (prescriptive):**

| zcode option (kind / optionId) | ACP `PermissionOption` | client selects → backend reply |
|---|---|---|
| `allow_once` / `allow_once` | `{Kind: allow_once, Name: <verbatim>, OptionId: "allow_once"}` | option's embedded `response` verbatim (`{decision:"allow", reason:"Approved once"}`) |
| `allow_always` / `allow_project` | `{Kind: allow_always, Name: <verbatim>, OptionId: "allow_project"}` | embedded `response` verbatim (carries `permissionUpdates`) |
| `deny` / `deny` | `{Kind: reject_once, Name: <verbatim>, OptionId: "deny"}` | embedded `response` verbatim (`{decision:"deny", reason: …}`) |
| (any unknown optionId returned by the client) | — | `{decision:"deny", reason: <denied-by-user content>}` — never allow (mirror of the backend posture) |
| ACP outcome `cancelled` | — | JSONL **error reply** `{code:-32021, message:"Client request cancelled: <method>"}` to the pending backend request; the turn's `session/stop` already went out via `machine.Cancel` (landed CORE-05 path) |
| host deadline expiry | — | `{decision:"deny", reason: <denied-by-user content>}` (turn continues honestly; alternative `-32022` timedOut — OQ-2) |

Bridge mechanics: a pending map keyed by backend `requestId` holding the in-flight
`conn.RequestPermission` context + cancel func. ACP `session/cancel` (the landed
cancel fan-in) additionally cancels ALL pending permission bridges for the turn. The
`ToolCall` field of the ACP request carries `{toolCallId: <zcode toolCallId>}` (plus
title/pending status when the tap already saw the tool — clients render the ask on
the tool card).

### Pattern 4: Tool streaming (TOOL-01) — the `tool.updated` taxonomy → ACP mapping

**Backend event shapes** (`session/event` envelopes, `type:"tool.updated"`,
discriminated union on `kind`) [VERIFIED: zcode-src 3.17.1
`packages/shared/src/zcode-protocol/index.ts:1528` (event name in the session-event
type enum), `:1738-1800` (`zcodeToolUpdatedEventPayloadSchema`), `:1880` (envelope
binding `zcodeSessionEventEnvelopeFor("tool.updated", …)`)]:

| kind | payload fields (beyond base) | wire verbatim |
|------|------------------------------|---------------|
| base (all) | `toolCallId` (required), `toolName?`, `parentToolCallId?`, `source?` ("subagent"), `agentId?`, `agentType?`, `background?`, `childSessionId?`, `childToolCallId?`, `description?` | [VERIFIED: index.ts:1743-1756 `zcodeToolCallBasePayloadSchema`] |
| `scheduled` | `toolName` (required), `input?`, `inputByteLength?`, `inputOmitted?`, `inputRef?`, `dependencies?`, `parallelGroupIndex?`, `canRunParallel?`, `schedule?`, `assistantMessageId?` | [VERIFIED: index.ts:1758-1771] |
| `started` | `startedAt` | [VERIFIED: index.ts:1772-1777] |
| `progress` | `elapsedMs?`, `pid?`, `stdoutBytes?`, `stderrBytes?`, `outputBytes?`, `outputPreview?`, `stdoutTail?`, `stderrTail?` | [VERIFIED: index.ts:1778-1788] |
| `result` | `result` (loose JSON object — `jsonObjectSchema` with superRefine on `display.kind`), `duration` | [VERIFIED: index.ts:1789-1793 + `:366-385`] |
| `error` | `error` (protocol error detail) | [VERIFIED: index.ts:1794-1798] |
| `batch` / `raw` | `{toolCallIds, successCount, errorCount}` / `{payload}` | [VERIFIED: index.ts:1799-1812] |

**Mapping (prescriptive):**

| backend event | ACP surface | fields |
|---|---|---|
| `scheduled` (first sight of toolCallId) | `tool_call` start — SDK `acp.StartToolCall(id, title, opts...)` | title = `description` else `toolName`; `Name` = toolName; `Kind` per table below; `Status` = pending; `RawInput` = `input`; `Locations` = `[{Path: file_path}]` when input carries one; `Content` = diff for edit/write tools (below) |
| `started` | `tool_call_update` | `Status` = in_progress |
| `progress` | `tool_call_update` | `Status` = in_progress; optionally `stdoutTail`/`outputPreview` as text `ToolContent` (keep v1 minimal: status only, log the rest structurally) |
| `result` | `tool_call_update` | `Status` = completed; `RawOutput` = `result`; optional text content from a bounded output excerpt |
| `error` | `tool_call_update` | `Status` = failed |
| `batch` / `raw` | no ACP equivalent — structural log only | never synthesized |
| subagent mirrors (`source:"subagent"`, `background:true`) | v1: log-only (single-agent card) | avoid misattributing child tool cards to the top-level turn |

**Kind mapping** (toolName → ACP ToolKind; case-insensitive; default `other` — the
SDK decodes unknown kinds as `ToolKindOther` [VERIFIED: acp-go-sdk v0.20.0
`types_gen.go:13644` comment "UnmarshalJSON reads a string that ToolKind does not
list as ToolKindOther"]): `Read`→read, `Edit`→edit, `Write`→edit, `Glob`/`Grep`→search,
`Bash`→execute, `WebFetch`/webfetch→fetch, `TodoRead`/`TodoWrite`/think-tools→think or
other, everything else→other. Tool names verified in the clone: `"Read"`, `"Glob"`,
`"Grep"`, `"Bash"`, `"Edit"`, `"Write"`, `"TodoRead"` [VERIFIED: zcode-src 3.17.1
`packages/core/src/tool/handlers/{read.ts:569, glob.ts:87, grep.ts:143, bash.ts:488,
edit.ts:258, write.ts:216, todo.ts:78}`]; webfetch/task tool names not read this
session — the `other` default covers them [ASSUMED for unlisted names].

**Diff rendering for edit tools (D-03):** the edit/write tool INPUTS carry the diff
material [VERIFIED: zcode-src 3.17.1
`packages/core/src/tool/handlers/edit.ts:86` — `const { file_path, old_string,
new_string, replace_all } = EditInputSchema.parse(...)`; `handlers/write.ts` carries
`{file_path, content}` shape (WriteInputJsonSchema, `:229` inputSchema binding)]:

- `Edit`: `acp.ToolDiffContent(path, newText=new_string, oldText=old_string)` — the
  SDK helper sets `{type:"diff", path, newText, oldText}` [VERIFIED: acp-go-sdk
  v0.20.0 `helpers.go:58-71` — `ToolDiffContent(path string, newText string, oldText ...string)`]
- `Write` (new file): `ToolDiffContent(path, newText=content)` — oldText omitted
  (spec: omitted/null for new files [CITED: agentclientprotocol.com tool-calls])
- `path` must be the **absolute** path per spec [CITED: agentclientprotocol.com
  tool-calls — diff `path` "absolute file path"]; resolve relative `file_path` against
  the session cwd before emitting
- Emit the diff on BOTH the `scheduled` tool_call and the `result` update (clients
  render early; ACP `tool_call_update` content merges per spec)

ACP wire shapes (all verified in the SDK this session): `SessionUpdate.ToolCall` →
`sessionUpdate:"tool_call"` requires `toolCallId`+`title`, optional
`kind/status/content/locations/name/rawInput/rawOutput` [VERIFIED: types_gen.go:9978-10002];
`SessionUpdate.ToolCallUpdate` → `tool_call_update`, only `toolCallId` required, all
else optional patch fields [VERIFIED: types_gen.go:10040-10052, 13492-13510]; statuses
`pending/in_progress/completed/failed` [VERIFIED: types_gen.go:13477-13483]; kinds
`read/edit/delete/move/search/execute/think/fetch/other` [VERIFIED:
types_gen.go:13623-13641]. Spec cross-check: tool_call when the model requests the
tool; tool_call_update carries only changed fields [CITED:
agentclientprotocol.com/protocol/v1/tool-calls.md].

**Neutral vocabulary (acpport):** extend `SessionNoteKind` with a tool note (or add a
second sink method) carrying `{ToolCallID, IsStart, Title, Name, Kind, Status, Diff
{Path, OldText, NewText}, Locations, RawInput, RawOutput}` — SDK types never cross
the seam (D-01 discipline); `sink.go` maps onto `StartToolCall`/`UpdateToolCall`/
`ToolDiffContent` exactly as it already maps message/thought notes.

### Pattern 5: api-key auth path (AUTH-01)

**ACP authenticate carries NO credentials.** The v1 request is
`AuthenticateRequest{MethodId AuthMethodId}` — nothing else [VERIFIED: acp-go-sdk
v0.20.0 `types_gen.go` AuthenticateRequest struct: fields `Meta`, `MethodId` only;
CITED: agentclientprotocol.com/protocol/v1/authentication.md — "No credentials are
passed in the request … The agent manages any credential exchange itself"]. Success is
an empty result `{}`. Therefore the api key enters the wrapper through its own config
surface: env var (recommended `ZCODE_ACP_API_KEY`) and/or flag, per D-05
"credentials from local config (never committed)". `authenticate{zai-api-key}`
records the key, pushes the snapshot, and verifies by read-back; `session/new`-gated
`auth_required` behavior stays out of v1 (the wrapper never refuses sessions on auth).

**Snapshot push (`provider/updateAccountConfig`)** — host→server request:

- Envelope schema (strict) [VERIFIED: zcode-src 3.17.1
  `packages/shared/src/zcode-protocol/index.ts:2590-2613`]: `{revision:
  nonEmptyString, basedOnZCodeBuiltinRevision: nonEmptyString, providers: record,
  states: record(state-entry strict)}` — **`providers: {}` and `states: {}` are
  schema-valid** (both are `z.record`); per-entry, only `zhipu-account`-typed
  entitled providers require `states[id].current` boolean [VERIFIED:
  `apps/zcode-cli/packages/bootstrap/src/app/process-provider-registry-runtime.ts:202-233` — `parseProcessAccountProviderConfigSnapshot`]
- The protocol comment settles what the snapshot is FOR: `"该协议传 Account Overlay
  与对应状态；API Key、JWT 和动态 Header 由请求期鉴权协议处理。"` — the snapshot carries
  the account overlay + states; **API keys, JWTs and dynamic headers are handled by
  the request-time auth protocol** (i.e. the runtime-headers ask), never the snapshot
  [VERIFIED: zcode-src 3.17.1
  `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/account-provider-config.ts:16-18`]
- The wrapper's own `revision` may be any non-empty string (reference adapter
  precedent: `"account:bridge:<timestamp>"` [CITED: docs/research/reference-adapters.md
  row 3]); **`basedOnZCodeBuiltinRevision` must be the exact formula**
  `zcode-builtin:<rev>:<sha256(path.resolve(activeBuiltinFilePath))>` — hash of the
  resolved absolute PATH STRING, computed at runtime, never hard-coded [VERIFIED:
  docs/protocol/install-discovery.md:79-85; V-EXP EXP-DISCOVERY-01 computed example
  `zcode-builtin:44:e9c0…`; in-repo Phase-3 BuiltinRevision already implements it —
  reuse, do not re-derive]
- For api-key mode the account overlay is intentionally EMPTY (zai-api is the
  personal layer, untouched by account snapshots — field-verified: `"Пустой снапшот
  не удаляет zai-api — он personal-слой"` [V-RPT field-notes]). The push's purpose on
  this path is D-02 conformance + a verified no-op: the ack arrives, the read-back
  confirms the personal `zai-api` models survived intact.
- Ack: `{receivedRevision, providerCount, status:"received"|"unchanged"}` — the
  schema's own comment warns the ack is not application:
  `"收到账号结果不代表配套 Built-in 已到达；应用版本只能读取 Registry 快照。"` [VERIFIED:
  zcode-src 3.17.1 `index.ts:2614-2621`; handler `account-provider-config.ts:22-34` —
  note also the configured error: `-32018 "Account Provider Config runtime is not
  configured"` when the sync hook is missing]

**Verification (PITFALL 1 — never the ack):**

1. `provider/refreshRegistry {reason}` → `{"refreshed":true}` [V-EXP EXP-WORKSPACEKEY-01 live]
2. `awaitRegistryContains` helper (per PITFALL 1; CONTEXT specifics name it as a
   Phase-6-reused asset): await a `state.updated` patch / a fresh `session/create`
   reply and assert the model list contains the expected provider models. The
   field-verified read-back surface is the create reply's
   `settings.model.available` [CITED: docs/protocol/app-server-dialect.md §8 —
   "the create reply's `settings.model.available` is the read-back surface", V-EXP-backed]
3. For api-key mode the assertion is: `zai-api` models present and the ack
   `receivedRevision` echoes the pushed revision — proving the empty overlay neither
   errored nor disturbed the personal layer.

**Runtime-headers ask:** answer (schema verbatim, success arm)
`{headersApplied:true, requestAuth:{apiKey, apiKeyId}}` — note both `apiKey` and
`apiKeyId` are individually OPTIONAL strings in the strict requestAuth object; the
discriminant is `headersApplied` [VERIFIED: zcode-src 3.17.1
`index.ts:2888-2915` — `zcodeProviderRuntimeHeadersResponseSchema =
z.discriminatedUnion("headersApplied", […])`, requestAuth keys quoted `apiKey:
nonEmptyString.optional(), apiKeyId: z.string().trim().min(1).optional(),
accountScope: …optional(), headers: record…optional()`]. Request params carry
`requestId` (format `${sessionId}:provider-runtime-headers:${uuid}`), `sessionId`,
`turnId?`, `workspace`, `modelSelection`, `providerId`, `accountAccess?`,
`expectedAccountScope?`, `rejectedProjectTokenFingerprint?`, `reason` [VERIFIED:
index.ts:2846-2867]. The 180 s server-side window and the abort/timeout
notification `interaction/providerRuntimeHeadersCancelled` `{requestId, sessionId,
workspace}` are bundle-anchored [CITED: docs/protocol/app-server-dialect.md §3;
V-SRC provider-runtime-headers.ts:17,48-100 re-read this session].

**Honesty note (evidence-grade):** on the zai-api personal path the ask is
**expected never to fire** — the model runner gates the refresh callback to
`zhipu-account` access types only: `requestAuthRequired = options.providerConfig.access.type === "zhipu-account" && …` and for any other access type the callback is `undefined`, with the source comment stating
`"调用级 runtime header Port 只服务绑定完整 Account Access 的账号型 Model；普通 API-key
Model 若也消费该 Port，会把静态鉴权误送到 Host 刷新并在请求前失败。"` [VERIFIED:
zcode-src 3.17.1 `apps/zcode-cli/packages/adapters/src/model/runner.ts:186-221` —
quoted gate + comment; final else arm `: undefined`]. `resolveModelForAttempt` then
skips the refresh entirely when the callback is absent [VERIFIED:
`packages/adapters/src/model/runner-runtime-headers.ts:24-28`]. This is 3.17.1
source (corroborating tier) + absence in every api-key capture/shim run; treat the
"never fires on 3.18.2" claim as high-plausibility [ASSUMED with strong
corroboration] — the responder answers it correctly regardless (CORE-06), and the
fake tier locks the reply shape.

### Pattern 6: The real-tier credential-cipher constraint (P0 planning input)

**The fact (HIGH, field-verified):** a real `session/send` on this machine fires
`interaction/requestCredentialCipher` with `operation:"decrypt"` ~10 s into the turn;
an unanswered ask stalls the turn until its deadline — this was THE Phase-1
degradation cause [V-EXP EXP-TURN-01; CITED: docs/protocol/app-server-dialect.md §3
row requestCredentialCipher]. The method is 3.18-only (absent from the entire 3.17.1
tree — drift baseline) and stays ×1 in the bundle re-check this session [VERIFIED:
bundle grep `"interaction/requestCredentialCipher"` ×1; pristine `cmp` vs `.orig`
identical].

**The field-verified answer shape:** `{requestId, result:{ok:true, value:<decrypted>}}`
(or `{requestId, result:{ok:false, error:{code, message}}}`) [V-RPT field-notes
app-server-stdio-protocol.md — `credentialCipherReplySchema = {requestId:uuid,
result: discriminatedUnion("ok")}`]. The shim answers decrypt by loading the keyring
key (`secret-tool lookup service "ZCode Credential Encryption" username
sha256(realpath credentials.json)`, 43-char base64url → 32 bytes) and AES-256-GCM
decrypting `enc:v2:os:<kid>:<iv>.<tag>.<ct>` (segments iv.tag.ct, kid = first 24 hex
of sha256(key)) — real turns complete only with this answered [VERIFIED: shim source
read this session — `loadKey`/`decrypt`/`requestCredentialCipher` handler quoted;
V-RPT FINDINGS.md §3.6 APIKEY_PATH_OK / OAUTH_TOOL_OK].

**Phase-5 options** (decision needed — Open Question 1):

1. **(recommended) Answer decrypt via a `secret-tool` subprocess behind a narrow
   `CredentialDecrypt` service** (os/exec + stdlib crypto/aes; same interface shape
   Phase 6's godbus vault will implement). No new Go module; the binary is present on
   this machine; the real tier (guarded, manual, quota-ledgered) is the only consumer.
   Encrypt ops: decline with `{ok:false,…}` (never fabricate).
2. Keep tracked-no-answer → the real tier CANNOT complete a turn → success criteria
   4/5 unattainable in Phase 5. Not viable without descoping the real-tier proof.
3. Pull godbus forward → violates the locked zero-new-deps/Phase-6 decision.

### Pattern 7: Session lifecycle (SESS-01) + Phase-4 review teardown

**`session/resume`** — backend params `{sessionId (required), workspace?,
thoughtLevel?, mcpServers?, toolAllowlist?, toolDenylist?, additionalDirectories?,
offPeakToolEnabled?, dynamicWorkflowEnabled?, dynamicWorkflowMode?}` (strict;
minimal call `{sessionId}`) [VERIFIED: zcode-src 3.17.1 `index.ts:2007-2029`]. The
reply is the SAME full session snapshot shape as create (`resumeSession` returns
`snapshot(context, …)`) [VERIFIED: `server-operations.ts:1586-1590` — quoted
`return await snapshot(context, activated.record, activated.knownSession);`], so
`createSessionID` (front.go) extracts `result.session.sessionId` unchanged. ACP side:
`ResumeSessionRequest{SessionId (required), Cwd (required), McpServers?,
AdditionalDirectories?, ConfigOptions?}` [VERIFIED: acp-go-sdk v0.20.0 types_gen.go
ResumeSessionRequest]. Front contract: validate the client-supplied SessionId is a
string we can pass through; call backend `session/resume`; **re-issue
`session/subscribe {deliveryKind:"desktop-continuous"}`** (the D-03
subscribe-before-any-turn gate applies to a reattached session exactly as to a new
one); register machine + registry entry; reply `{sessionId}`. **No replay**: the
wrapper never synthesizes updates for pre-resume history — the client sees only
future turns (D-04). Do NOT setModel on resume unless the client asked (model
binding persists backend-side; the resume params carry optional thoughtLevel hints
we do not fabricate).

**`session/close`** — backend params `{sessionId, expectedPersistence?}` → result
`{closed?: boolean}` [VERIFIED: zcode-src 3.17.1 `index.ts:2427-2435`]. ACP side:
`CloseSessionRequest{SessionId}` [VERIFIED: SDK types_gen.go]. Front contract
(teardown order): if a turn is active on the session, cancel it first (machine
.Cancel; wait bounded); best-effort backend `session/close`; release
registry entry (core.Registry.Remove — exists, unused until now [VERIFIED: in-repo
internal/core/registry.go:78-85]); remove the conn notification observer AND the
router observer (WR-07 fix: `NewMachine` must return/expose its remove funcs —
today the AddObserver remove func is hidden inside NewMachine [VERIFIED: in-repo
internal/turn/machine.go:182-197 — `conn.AddObserver(m.ObserveEnvelope)` discard +
`conn.SetOnRouterEvent(m.observeRouterEvent)`]); reply `{}`. Lock with a test
asserting the conn's observer count returns to baseline (04-REVIEW WR-07 fix text).
WR-04 (error-path orphan) folds in: any post-create failure in NewSession best-effort
closes the backend session and logs structurally.

**WR-01 fix shape (router-event fan-out):** mirror the notification observer list:
`AddRouterObserver(fn) (remove func())`, ordered delivery under a mutex, machines
register at construction and remove at teardown. Machines other than the active one
early-return (state!=running), so fan-out delivery to ALL machines is safe and fixes
the last-wins steal. The existing single-slot `SetOnRouterEvent` can remain for
pre-traffic test wiring or be folded into the fan-out — prefer folding (one
mechanism), keeping the routerMu read discipline [VERIFIED: in-repo
internal/zcode/conn.go:103-119, 456-464; 04-REVIEW WR-01 fix text].

**WR-02:** `agentAdapter.sink` becomes `atomic.Pointer[UpdateSink]` (bindSink store /
sinkForPrompt load) [04-REVIEW WR-02 fix text; in-repo internal/acpport/agent.go:27-30, 271-288].

**WR-05/06:** the front-wide rejection message becomes scope-accurate
(e.g. `"another prompt is already running (this agent runs one turn at a time)"`,
updating the cancel_test.go-locked constant); unsupported content blocks (image/audio/
embedded resource) get a structural slog.Warn at the drop site (kinds and counts
only) or `-32602` — recommend the warn+drop variant for v1 (a conforming client never
sends them; non-conforming input should not hard-fail a chat) [04-REVIEW WR-05/WR-06].

**IN-01..IN-05 (info tier):** fix the fan-out doc contract text (IN-01); delete or
justify the dead `core.Session.CancelTurn`/`Subscribed` fields — Phase 5 IS the named
potential consumer: keep `Subscribed` only if the resume path reads it, else delete
with the D-03 comment (IN-02); refresh stale docs (IN-03); guard
`--model` without `--reasoning-level` at config time or omit the empty
`options` object (IN-04 — the omit-shape matches the documented setModel contract);
record the turn's id in the turn slot and skip foreign-turnId deltas in the tap
(IN-05 — three-line guard, becomes structural once tool events flow).

**WR-03 (backpressure) — the deliberate design:** the tap currently writes ACP frames
inline on the backend read goroutine while holding `obsMu`; a stalled client wedges
both legs (04-REVIEW WR-03). Phase 5 makes this worse (tool diffs are large frames),
so it needs the designed policy, not a drive-by. Options:

- **(a) recommended — per-turn pump goroutine with a bounded queue and a memory
  ceiling:** the tap enqueues non-blocking; a per-turn goroutine drains to the ACP
  wire at wire pace; tool_call_update frames for the SAME toolCallId coalesce
  (keep-latest — status/content are patch semantics per spec); text/thought deltas
  are lossless, so when the queue's byte ceiling is hit (e.g. 64 MiB/turn) the turn
  FAILS LOUD (prompt error naming the cause) instead of hanging or silently dropping
  — honest, bounded, and the read goroutine never parks on the ACP wire.
- (b) keep the synchronous pump and document the coupling (04-REVIEW option b) —
  minimal, but leaves the unbounded stall property exactly as reviewed.
- (c) conn-level per-observer mailboxes — cleanest long-term, largest refactor;
  defer past v1.

### Anti-Patterns to Avoid

- **Answering a reverse request from the ordinary write queue** — replies must ride
  the bypass lane (dialect §1; already the conn's behavior — the responder must keep
  using Reply/ReplyError) [VERIFIED: in-repo internal/zcode/writer.go:127-130].
- **Deduping by wire envelope id** — every re-announce carries a NEW wire id; the
  stable key is `params.requestId` (Pattern 2).
- **Trusting the `received` ack** (PITFALL 1) — read the registry back.
- **`headersApplied:false`** for account providers (PITFALL 3) — guaranteed -32031
  turn death; on failure reply with an ERROR envelope instead.
- **Answering an unknown optionId with allow** — backend posture falls to deny;
  mirror it (Pattern 3).
- **Replying to every re-announce envelope** — first-reply-wins on the server;
  redundant bytes and a race window; reply once via the dedupe entry.
- **Crashing/unhandled panics on unknown `interaction/*`** — CORE-06: the -32601
  default is the floor, plus a structural log naming the method.
- **Synthesizing tool/permission events** — every ACP frame traces to one received
  backend envelope (D-04 honesty carries over to TOOL-01).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ACP request_permission round-trip | Custom JSON-RPC request correlation over the front leg | `AgentSideConnection.RequestPermission` + SDK outcome union | SDK owns framing/pending; the outcome union decodes `selected`/`cancelled` correctly [VERIFIED: SDK agent_gen.go:402] |
| tool_call wire assembly | Hand-marshaled session/update maps | `acp.StartToolCall`/`UpdateToolCall`/`ToolDiffContent`/`WithStart*` helpers | The helpers set `sessionUpdate` discriminants and patch semantics correctly [VERIFIED: SDK helpers.go:119-141,194] |
| AES-256-GCM credential decrypt (if OQ-1 lands) | Any custom cipher | stdlib `crypto/aes`+`cipher.NewGCM` over the verified `enc:v2:os:` layout (iv.tag.ct base64url) | Field-verified layout; stdlib GCM is the STACK decision; Phase 6 replaces the key source with godbus behind the same interface |
| JSONL replies/deadlines/pending table | A second reverse-request framework beside the conn | Extend conn via OnRequest + RouterEvent | conn.go already owns demux, bypass lane, deadline recording, RouterEvent honesty (03-02 locked tests) |
| Golden transcript assertions | Ad-hoc line comparisons | fake.Scenario + TestGolden corpus machinery | `expect_answer` semantic compare and fail-loud loader already exist [VERIFIED: in-repo internal/zcode/fake/fake.go:309-352] |

**Key insight:** Phase 5 is composition, not transport — every primitive (codec,
bypass lane, turn machine, fake, SDK seam) exists and is test-locked; the new code is
policy (who answers what, when, with which deadline) and mapping (backend events →
ACP vocabulary).

## Common Pitfalls

### Pitfall 1: Deduping re-announcements by the wrong key or answering each one
**What goes wrong:** duplicate backend replies; the first reply already resolved the
pending set, later replies target cleaned ids (dropped server-side) — or worse, a
per-wire-id dedupe map grows one entry per second per pending ask (unbounded).
**Why it happens:** assuming the re-announce reuses the same wire id.
**How to avoid:** dedupe `(method, params.requestId)`; reply once to the registered
wire id; bounded map with eviction on answer/close/TTL.
**Warning signs:** fake transcript shows N replies for one reverse; memory growth
during long permission waits.

### Pitfall 2: Letting a permission bridge outlive its turn
**What goes wrong:** an ACP-cancelled turn leaves the pending backend request
unanswered; the server holds the tool (guarded mode) and later asks re-announce
forever.
**How to avoid:** the turn cancel path (landed fan-in) must also cancel all pending
bridges of the turn (reply -32021); teardown (session/close) cancels any session
pendings; conn close drains the dedupe map with error replies? — no: on conn close
nothing is owed on a dead stream (Phase-3 tracker precedent).
**Warning signs:** re-announce notes in the fake transcript after cancel tests.

### Pitfall 3: Faking `authenticate` success (honesty regression)
**What goes wrong:** a success ack without a real snapshot push + read-back repeats
Phase-4's problem at a higher tier; AUTH-01's whole point is the read-back proof.
**How to avoid:** authenticate = record key → push snapshot → refreshRegistry →
awaitRegistryContains → only then `{}` success; failures surface as coded errors.
**Warning signs:** tests asserting the ack instead of the read-back.

### Pitfall 4: Schema-invalid reverse answers
**What goes wrong:** the server zod-parses every reverse reply; an invalid reply
REJECTS the pending request ("Invalid response") — e.g. the shim's `{headers:{}}`
stub for runtime-headers would fail the schema.
**How to avoid:** lock every reply shape against the schema quotes in this document
with fake `expect_answer` vectors.
**Warning signs:** fake scenarios pass but the real tier resolves "Invalid response".

### Pitfall 5: Relative diff paths and non-absolute locations
**What goes wrong:** ACP diff `path` must be absolute; tool inputs may carry
workspace-relative paths.
**How to avoid:** resolve against the session cwd before emitting.
**Warning signs:** clients render broken diff cards.

### Pitfall 6: Teardown ordering races
**What goes wrong:** session/close while a turn runs leaks the backend session or
double-cancels; observer removal mid-delivery is lock-safe (IN-01) but the machine
must not resolve after removal.
**How to avoid:** close = cancel-then-wait(bounded) → backend close → registry
Remove → observer removes; assert observer-count baseline in a test (WR-07 fix).
**Warning signs:** -race reports; observer count not returning to baseline.

### Pitfall 7: The cipher-ask blindside on the real tier
**What goes wrong:** the fake tier is green but every real turn stalls at ~10 s
(unanswered decrypt ask, 15 s deadline, degraded resolution).
**How to avoid:** settle OQ-1 BEFORE building the real tier; keep the responder's
cipher policy injectable so the decision is one wiring change.
**Warning signs:** real-tier capture shows `interaction/requestCredentialCipher`
unanswered note.

## Code Examples

### Responder entry + dedupe skeleton

```go
// Source: composition of in-repo conn.go seam (handleRequest:325-336) with
// zcode-src 3.17.1 server.ts re-announce mechanics (quoted in Pattern 2).
func (r *Responder) serve(ctx context.Context, env zcode.Envelope) (any, *zcode.WireError) {
    req := normalize(env) // digs out params.requestId / params.sessionId once

    if ent, fresh := r.dedupe.claim(req.Method, req.ReqID); !fresh {
        ent.logJoin() // structural: re-announce joined, id keys only
        return nil, nilRedirectHandled() // no reply; first envelope owns the answer
    }
    defer r.dedupe.release(req.Method, req.ReqID)

    pol, ok := r.methods[req.Method]
    if !ok {
        return nil, &zcode.WireError{Code: zcode.CodeMethodNotFound,
            Message: "unsupported host request: " + req.Method}
    }
    if pol.Deadline > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, pol.Deadline)
        defer cancel()
    }
    return pol.Handler(ctx, req)
}
```

### Permission forward + outcome mapping

```go
// Source: zcode-src 3.17.1 index.ts:2740-2753 (params), index.ts:732-739 (option
// with embedded response), interaction-broker.ts:241-243 (unknown→deny);
// acp-go-sdk v0.20.0 types_gen.go:8171-8198 (request), :8041-8070 (outcome).
func (b *PermissionBridge) handle(ctx context.Context, req interact.Request) (interact.Answer, error) {
    var p struct {
        RequestID string            `json:"requestId"`
        ToolCallID string           `json:"toolCallId"`
        ToolName  string            `json:"toolName"`
        Options   []zcodePermOption `json:"options"`
    }
    _ = json.Unmarshal(req.Params, &p)

    opts := make([]acp.PermissionOption, 0, len(p.Options))
    for _, o := range p.Options {
        opts = append(opts, acp.PermissionOption{
            Kind:     zcodeKindToACP(o.Kind), // allow_once→allow_once; allow_always→allow_always; deny→reject_once
            Name:     o.Name,
            OptionId: acp.PermissionOptionId(o.OptionID), // verbatim passthrough
        })
    }

    resp, err := b.acpConn.RequestPermission(ctx, acp.RequestPermissionRequest{
        SessionId: acp.SessionId(req.SessionID),
        Options:   opts,
        ToolCall:  acp.ToolCallUpdate{ToolCallId: acp.ToolCallId(p.ToolCallID)},
    })
    if err != nil { // ctx cancelled (turn cancel) → backend cancelled code
        return interact.Answer{Err: &zcode.WireError{Code: CodeCancelled(-32021),
            Message: "Client request cancelled: interaction/requestPermission"}}, nil
    }
    if out := resp.Outcome; out.Cancelled != nil {
        return interact.Answer{Err: cancelledWireError()}, nil
    }
    opt := findOption(p.Options, string(out.Selected.OptionId))
    if opt == nil {
        return interact.Answer{Result: map[string]any{
            "decision": "deny", "reason": deniedByUserContent}}, nil // unknown optionId → deny
    }
    return interact.Answer{Result: json.RawMessage(opt.Response)}, nil // option.response verbatim
}
```

### Tool note translation (tap → sink)

```go
// Source: zcode-src 3.17.1 index.ts:1738-1812 (tool.updated kinds); SDK
// helpers.go:119-141 (StartToolCall), :58-71 (ToolDiffContent), edit.ts:86 (inputs).
case "scheduled":
    note := acpport.ToolNote{IsStart: true, ToolCallID: p.ToolCallID,
        Title: firstNonEmpty(p.Description, p.ToolName), Name: p.ToolName,
        Kind: toolKind(p.ToolName), Status: acpport.ToolStatusPending, RawInput: p.Input}
    if fp := inputString(p.Input, "file_path"); fp != "" {
        note.Locations = []acpport.Location{{Path: absJoin(f.sessionCwd, fp)}}
    }
    if d, ok := editDiff(p.Input, p.ToolName); ok { // Edit: old/new_string; Write: content
        note.Diff = &d
    }
```

### Fake scenario (permission allow, new tier)

```json
{
  "name": "permission-allow",
  "on_request": [
    {"method": "session/send",
     "reply": {"result": {"accepted": true, "sessionId": "sess_p1", "stateRevision": 1}},
     "side_effects": [
       {"emit": {"method": "state.updated",
                 "params": {"patch": {"status": "running"}, "reason": "prompt_started",
                            "revision": 1, "scope": "session", "sessionId": "sess_p1"}}},
       {"emit": {"method": "session/event",
                 "params": {"type": "tool.updated", "seq": 1, "turnId": "t1",
                            "payload": {"kind": "scheduled", "toolCallId": "tc1",
                                        "toolName": "Bash", "input": {"command": "echo hi"}}}},
       {"reverse": {"method": "interaction/requestPermission",
                    "params": {"requestId": "perm-1", "sessionId": "sess_p1",
                               "toolCallId": "tc1", "toolName": "Bash", "reason": "run echo",
                               "riskLevel": "medium", "input": {"command": "echo hi"},
                               "options": [
                                 {"optionId": "allow_once", "kind": "allow_once", "name": "Allow once",
                                  "response": {"decision": "allow", "reason": "Approved once"}},
                                 {"optionId": "deny", "kind": "deny", "name": "Deny",
                                  "response": {"decision": "deny", "reason": "denied"}}]},
                    "timeout_ms": 15000,
                    "expect_answer": {"result": {"decision": "allow", "reason": "Approved once"}}}}
     ]}
  ],
  "eof_drain_ms": 100
}
```

(A re-announce variant repeats the same `reverse` with identical
`params.requestId` — the fake needs a small extension to re-issue with a NEW wire id
per re-announce, matching the server mechanics; `expect_answer` stays on the first.)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| conn's inline 2-case router (Phase 3) | policy object over OnRequest (Phase 5, D-01) | this phase | CORE-06 surface; runtime-prefs answer migrates |
| tracked-no-answer credential-cipher | must answer on the real tier (OQ-1) | Phase-1 V-EXP fact vs Phase-3 policy | real-tier design input |
| "dedupe the 1 s re-announce" (uniform interval, same id — mental model) | new wire id per re-announce, 1 s doubling → 10 s cap, first-reply-wins | verified this session from server.ts | dedupe design now precise |
| headers `{headers:{}}` stubs (shim era) | schema-valid `{headersApplied, requestAuth}` replies | field notes + schema read | shim stub would be REJECTED by the result schema today |

**Deprecated/outdated:** none new — the dialect doc's drift baseline stands
(re-verified pristine this session: `cmp` bundle vs `.orig` identical; all quoted
method strings ×1; `allow_*` literals present; `18e4` ×5).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `interaction/requestProviderRuntimeHeaders` never fires on zai-api personal-provider turns on 3.18.2 (basis: 3.17.1 runner gate + absence in all api-key captures/shim runs) | Pattern 5 | If it does fire, the responder's api-key answer is exercised live — the design already covers it (CORE-06); low risk |
| A2 | `secret-tool` exists on target v1 machines (Ubuntu GNOME) — verified on THIS machine only | Pattern 6 / Environment | Real tier blocked on machines without libsecret-tools; fallback = document + OQ-1 alternative |
| A3 | Tool names for webfetch/task/other tools are unlisted → `other` kind default renders acceptably | Pattern 4 | Cosmetic (wrong icon/label); mapping table is data, easily corrected |
| A4 | An empty-providers snapshot push is harmless to the personal layer on 3.18.2 (field-note V-RPT + schema-valid) | Pattern 5 | Read-back step (PITFALL 1 discipline) catches any disturbance before a turn runs |
| A5 | 300 s is an acceptable requestPermission host deadline default | OQ-2 | Too short → user-visible auto-denies; too long → cancelled-by-other-means; configurable knob contains it |
| A6 | GLM-5.3-Flash keeps `high` as a valid reasoning level for the corpus-locked setModel preamble (carried from Phase-3 locks, unchanged bundle) | Validation | Real tier setModel refusal → use the documented refusal to re-lock levels |

## Open Questions

1. **requestCredentialCipher strategy on the real tier** (Pattern 6, P0)
   - What we know: the ask fires mid-turn on real runs (V-EXP), the turn cannot
     complete unanswered, the shim's `secret-tool`+GCM answer is field-verified, and
     godbus is locked to Phase 6.
   - What's unclear: whether Phase 5 accepts the secret-tool subprocess (new external
     runtime dependency, guarded-manual-tier-only) or descopes the real turn proof.
   - Recommendation: option 1 (secret-tool seam behind the CredentialDecrypt
     interface; decrypt-only, decline encrypt; real tier only). Needs planner/user
     confirmation — it revises Phase-3's tracked-no-answer posture on this tier.
2. **requestPermission host deadline default** (Pattern 3, OQ-2)
   - What we know: the server sets NO timeout for requestPermission (guarded mode
     waits); D-01/CONTEXT require a per-method deadline whose expiry answers
     deny/cancel.
   - What's unclear: the right default (user-thinking time vs never-hang).
   - Recommendation: configurable, default 300 s, expiry → deny with the
     denied-by-user reason (turn continues honestly); `-32022` timedOut is the
     alternative if the planner prefers a hard tool-cancel.
3. **WR-03 backpressure policy** (Pattern 7)
   - What we know: the review demands a deliberate design; three options sketched.
   - Recommendation: option (a) — per-turn pump, bounded queue, coalesce tool
     updates, memory-ceiling → fail loud.
4. **Authenticate failure semantics without a configured key**
   - What we know: `authenticate{zai-api-key}` carries no key on the wire; the key
     comes from env/flag config (D-05).
   - What's unclear: whether authenticate without a key answers an honest error
     (recommended) or falls back to reading the desktop's own personal config
     (read-only, but couples the wrapper to the desktop's store in Phase 5).
   - Recommendation: env/flag key required for the api-key authenticate to succeed;
     honest -32603-class error naming `ZCODE_ACP_API_KEY` otherwise.
5. **Whether `Subscribed`/`CancelTurn` registry fields survive** (IN-02)
   - Recommendation: `CancelTurn` deleted (the front's closure owns cancellation);
     `Subscribed` deleted with the D-03 comment unless the resume path reads it.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | 1.27.3 | — |
| mise | gates (build/vet/lint/test) | ✓ | (manages pinned golangci-lint 2.15.0) | — |
| Real ZCode install | guarded real tier | ✓ | 3.18.2 squashfs-root (pristine re-verified) | none — real tier is manual/guarded anyway |
| builtin provider config | real tier spawn env | ✓ | revision 44 tree | — |
| `secret-tool` | OQ-1 real-tier cipher answers | ✓ | libsecret CLI present | OQ-1 alternatives |
| Real api-key credentials | real tier | ✓ in local ZCode config (never committed) | — | none (D-05) |
| godbus/keyring | Phase 6 only | deliberately absent | — | out of scope |

**Missing dependencies with no fallback:** none for CI/fake tiers. For the real tier:
real account credentials (present) and, if OQ-1 lands, `secret-tool` (present here).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `-race`; golden corpus (fake scenarios + `.golden` transcripts); table-driven TestGolden; TestEOFSignature zero-frame fail-loud |
| Config file | none — canonical gate order build→vet→lint→test→test-experiments via mise (Phase-2 lock) |
| Quick run command | `go test -race ./internal/interact/... ./internal/acpfront/... ./internal/acpport/... ./internal/zcode/...` |
| Full suite command | `mise run ci` (or `go build ./... && go vet ./... && golangci-lint run && go test -race ./...`) |
| Guarded real tier | `//go:build realdesktop` + `ZCODE_ACP_REAL_E2E=1` double guard (03-07 precedent); manual-only; canonical command in the file header |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CORE-06 | unknown `interaction/*` → -32601, never crash | unit (conn/responder) | `go test -race ./internal/interact/ -run TestUnknown` (extends existing conn unknown-host-method lock) | ❌ Wave 0 (new package; conn-side lock exists) |
| CORE-06 | per-method deadlines answered/recorded | unit | `go test -race ./internal/interact/ -run TestDeadline` | ❌ Wave 0 |
| CORE-06 | re-announce dedupe by requestId (new wire id per re-announce) | unit + fake scenario | `go test -race ./internal/interact/ ./internal/zcode/fake/ -run TestReannounce` | ❌ Wave 0 (fake needs re-announce reverse extension) |
| CORE-06 | runtime-prefs answer migrates byte-identical | fake golden | `go test -race ./internal/zcode/fake/ -run TestGolden/runtime-preferences` | ✅ (existing scenario re-pointed at the responder) |
| TOOL-01 | tool.updated scheduled/started/progress/result → tool_call/update order + fields | fake golden + front harness | `go test -race ./internal/acpfront/ -run TestToolLifecycle` | ❌ Wave 0 |
| TOOL-01 | edit-tool diff rendering (Edit/Write inputs) | unit + golden | `go test -race ./internal/acpfront/ -run TestToolDiff` | ❌ Wave 0 |
| TOOL-02 | permission options map (allow_once/allow_always/deny→reject_once), outcome forwarded verbatim | front harness (tee byte-lock) | `go test -race ./internal/acpfront/ -run TestPermission` | ❌ Wave 0 |
| TOOL-02 | ACP cancel → pending resolved -32021; deadline expiry → deny | front harness + fake | `go test -race ./internal/acpfront/ ./internal/interact/ -run TestPermissionCancel` | ❌ Wave 0 |
| AUTH-01 | snapshot push shape + read-back (awaitRegistryContains) | unit + fake scenario | `go test -race ./internal/interact/ ./internal/hostsvc/ -run TestAuthReadback` | ❌ Wave 0 |
| AUTH-01 | runtime-headers reply schema vector (incl. headersApplied:false never emitted) | fake golden | `go test -race ./internal/zcode/fake/ -run TestGolden/runtime-headers` | ❌ Wave 0 |
| SESS-01 | resume = backend resume + re-subscribe + no replay | front harness | `go test -race ./internal/acpfront/ -run TestResume` | ❌ Wave 0 |
| SESS-01 | close = cancel→backend close→registry/observer/machine released (observer-count baseline) | front harness | `go test -race ./internal/acpfront/ -run TestCloseTeardown` | ❌ Wave 0 |
| WR-01 | router-event fan-out (A then B; expiry resolves A) | unit | `go test -race ./internal/zcode/ ./internal/turn/ -run TestRouterFanout` | ❌ Wave 0 |
| WR-02 | sink published atomically | unit (compile-time + race) | `go test -race ./internal/acpport/` | ✅ (existing suite re-run) |
| WR-05 | scope-accurate concurrency rejection message | harness (locked constant update) | `go test -race ./internal/acpfront/ -run TestConcurrent` | ✅ (update locked text) |
| Full cycle | new → prompt(tool+permission) → resume → close, CI tier (fake) | composed harness E2E | `go test -race ./internal/acpfront/ -run TestFullCycle` | ❌ Wave 0 |
| Real tier | same cycle on api-key (guarded) | manual + tagged test | `go test -tags realdesktop ./test/realdesktop/ -run TestAPIKeyFullCycle` (ZCODE_ACP_REAL_E2E=1) | ❌ (guarded tier, Phase-5 late) |

### Sampling Rate

- **Per task commit:** quick run command (touched packages, `-race`)
- **Per wave merge:** full suite (`mise run ci` gates)
- **Phase gate:** full suite green + guarded real-tier evidence (scrubbed capture,
  V-EXP id, quota-ledger entry) before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/interact/` package skeleton + responder/dedupe/deadline tests (CORE-06 core)
- [ ] fake re-announce extension (new wire id per re-announce) + scenarios:
      permission-allow / permission-deny / permission-cancel / reannounce-dedupe /
      runtime-headers-answer / user-input-decline / tool-lifecycle (+golden)
- [ ] acpport tool-note vocabulary + sink mapping tests
- [ ] front harness: RecordingClient.RequestPermission extension (scripted outcome +
      cancel), tool-note capture in Updates()
- [ ] framework install: none needed (stdlib; no new deps)

## Security Domain

### Applicable ASVS Categories (security_enforcement=true, level 1)

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | `authenticate{zai-api-key}`: key from env/flag only (never argv-visible child env beyond allowlist, never wire, never logs); honest error when absent |
| V3 Session Management | yes | session registry keyed by backend-generated ids; teardown releases all per-session state (WR-07); resume re-validates before registering |
| V4 Access Control | yes | permission bridge: deny-by-default on unknown optionId AND on deadline expiry; never auto-allow; auto-allow policy explicitly does not exist in v1 |
| V5 Input Validation | yes | validate `authenticate` methodId ∈ advertised set; resume/close session ids registered; unknown backend methods → bounded -32601; diff paths resolved absolute |
| V6 Cryptography | conditional | only if OQ-1 lands: stdlib AES-256-GCM with the field-verified `enc:v2:os:` layout (iv.tag.ct, base64url) — never hand-rolled; key never logged/persisted |
| V7 Secrets Logging | yes | redaction list (`apiKey`, `token`, `access_token`, `enc:v2:*`, `authorization`) on the trace logger; reverse-request logs carry id keys/durations/methods only (conn.go precedent) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Credential leakage via logs/captures | Information Disclosure | structural logging (no params), scrub-capture gate + scan-secrets CI (existing); real-tier captures scrubbed before commit (D-05) |
| Re-announce flood growing host state | Denial of Service | bounded dedupe map + eviction (answer/close/TTL); join logged, never stored per-frame |
| Unknown-method crash after desktop update | Denial of Service / Tampering | -32601 default + structural log (CORE-06; PITFALL 5 posture) |
| Permission bypass via malformed outcome | Elevation of Privilege | unknown optionId → deny; `escalate`/`modify` decisions never synthesized; forwarding restricted to the option's own embedded response |
| Orphaned backend session after error | Denial of Service | WR-04 best-effort close on post-create failure + WR-07 teardown |

## Sources

### Primary (HIGH confidence)

- zcode-src 3.17.1 clone (read this session, file:line quoted): interaction-broker.ts
  (1 s re-announce constant, v4 answer mapping, unknown-optionId deny),
  interaction-response-race.ts, server.ts (requestClient: new wire id per send,
  re-announce doubling to 10 s cap, first-reply-wins, result-schema reject),
  server-types.ts (-32020/-32021/-32022), provider-runtime-headers.ts (180 s,
  cancel notification, -32031), permission-options.ts (option kinds + embedded
  responses), packages/shared/src/zcode-protocol/index.ts (requestPermission params,
  option schema, response schema, runtime-headers params/response, snapshot params +
  ack, resume/close params, tool.updated taxonomy, session-event type list),
  zcode-protocol-legacy-types.ts (decision enum), adapters/src/model/runner.ts +
  runner-runtime-headers.ts (account-only refresh gate), core tool handlers (tool
  names, edit/write inputs), account-provider-config.ts (snapshot handler comment +
  -32018), process-provider-registry-runtime.ts (snapshot parse, standalone gating)
- coder/acp-go-sdk v0.20.0 module source (read this session): agent_gen.go
  (RequestPermission sender), types_gen.go (PermissionOption/kinds, outcome union,
  ToolCall/ToolCallUpdate/ToolCallContentDiff, SessionUpdate variants, kind/status
  constants, AuthenticateRequest, Resume/CloseSessionRequest), helpers.go
  (StartToolCall/UpdateToolCall/ToolDiffContent), client_gen.go (client-side
  RequestPermission dispatch)
- In-repo source (Read this session): internal/zcode/conn.go, envelope.go, writer.go
  (grep-anchored), call.go (anchored), internal/zcode/fake/{fake.go,scenario.go},
  internal/turn/machine.go, internal/acpfront/{front.go,stream.go,turnctx.go},
  internal/acpport/{types.go,agent.go,sink.go}, internal/core/registry.go,
  cmd/zcode-acp/main.go, go.mod
- docs/protocol/app-server-dialect.md (V-EXP/V-SRC provenance carried),
  docs/protocol/install-discovery.md (revision formula), docs/research/reference-adapters.md
  (v0.89.0 rows 3/4/5/12 — snapshot formula, runtime-headers reply, reannounce dedup)
- Machine-local ground truth (V-EXP/V-RPT, never committed): FINDINGS.md
  (APIKEY_PATH_OK, credentials layout, headersApplied facts),
  field-notes app-server-stdio-protocol.md + oauth-coding-plan-shim.md (host
  obligation reply shapes, snapshot recipe, personal-layer facts), shim source
  (~/.local/bin/zcode-claude-shim — cipher decrypt path), experiments/captures/*
  (EXP-TURN-01 cipher-ask fact; no permission/runtime-headers shapes recorded live)
- 3.18.2 bundle drift re-check (this session): pristine `cmp` identical; method
  strings ×1 (`requestPermission`, `requestUserInput`, `requestProviderRuntimeHeaders`,
  `requestCredentialCipher`, `session/resume`, `session/close`);
  `allow_once`×5 `allow_always`×4 `allow_project`×1 `allowSession`×1 `allow_session`×1;
  `providerRuntimeHeadersCancelled`×3; `18e4`×5

### Secondary (MEDIUM-HIGH confidence)

- agentclientprotocol.com v1 spec pages fetched this session: tool-calls (request
  permission shapes, tool_call/update semantics, diff shape, cancellation MUST),
  initialization + authentication (authenticate `{methodId}`-only, credential
  handling, auth_required gating) — cross-checked against the SDK source (two
  independent authoritative sources)

### Tertiary (LOW — flagged)

- None material; A1/A2/A3 in the Assumptions Log carry the residual uncertainty

## Metadata

**Confidence breakdown:**
- Responder/dedupe mechanics: HIGH — server.ts + interaction-broker.ts read with
  verbatim quotes this session; matches the locked D-01 decision
- Tool/permission/snapshot/auth wire shapes: HIGH — zod schemas + SDK types quoted
  verbatim; bundle literals re-derived
- Real-tier behavior (which asks fire on zai-api turns): HIGH for the cipher ask
  (V-EXP), ASSUMED-with-corroboration for runtime-headers absence (A1)
- Test design / validation map: HIGH — extends proven Phase-3/4 machinery

**Research date:** 2026-11-20
**Valid until:** 2026-12-20 (stable domain — private dialect + pinned SDK; re-check on
any desktop version bump or SDK schema bump)

## RESEARCH COMPLETE
