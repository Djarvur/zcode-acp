# Architecture Research

**Domain:** Go stdio protocol adapter — ACP v1 server front (JSON-RPC 2.0) wrapping ZCode Desktop app-server back (JSONL, non-JSON-RPC)
**Researched:** 2026-11-19
**Confidence:** HIGH for ZCode app-server wire facts (field-verified on 3.18.2 + 3.17.x source tree), HIGH for ACP v1 semantics (official spec), MEDIUM for adapter precedents (codex-acp read via READMEs), LOW items labeled inline.

## The Headline Answers (quality gates)

**One app-server process for many ACP sessions — not one per session.** The wrapper spawns exactly one
`ELECTRON_RUN_AS_NODE=1 … zcode.cjs app-server --stdio` child per wrapper process lifetime and multiplexes
all ACP sessions over it. Evidence: the app-server itself keeps `sessions: Map<string, ZCodeProtocolSessionRecord>`
(server-types.ts) and is designed as a multi-session host; the real desktop host runs one app-server for the whole
app; the provider registry is **process-global** (`provider/updateAccountConfig` is not per-session), so per-session
children would re-push the snapshot and re-mint PATs N times; and the Electron-as-node child is heavy
(hundreds of MB). Precedent: `agentclientprotocol/codex-acp` (the ACP org's own adapter over the Codex App
Server) runs one shared app-server process and multiplexes ACP sessions onto it, with a crash-restart/backoff
module (MEDIUM confidence, README-derived). ACP sessionIds map 1:1 to app-server sessionIds — the app-server
generates the id at `session/create`; the wrapper reuses it verbatim as the ACP `sessionId` (opaque to clients,
debuggable in logs, no alias table to persist).

**The wrapper is a host, not a proxy.** The app-server runs in host-managed mode and *blocks mid-turn* waiting
for `interaction/*` answers (`requestCredentialCipher`, `requestProviderRuntimeHeaders`,
`requestOfficialMcpAuthHeaders`, `requestPermission`, plus `session/requestRuntimePreferences`). An unanswered
reverse request stalls the turn until its timeout (180 s for provider-runtime-headers, source-confirmed).
So the architecture must guarantee: the JSONL read loop never blocks on a handler; every interaction method
has a registered responder (or a policy-driven auto-answer); and slow host work (keyring, PAT minting over
api.z.ai) happens in handler goroutines with their own deadlines.

## Standard Architecture

### System Overview

```
              ACP client (Orca / Zed / any v1 client)
                  │  stdin/stdout — ACP v1: JSON-RPC 2.0, newline-JSON, UTF-8
┌─────────────────▼─────────────────────────────────────────────────────┐
│  zcode-acp  (single static Go binary)                                 │
│                                                                       │
│  ┌───────────────────┐        ┌──────────────────────────────┐        │
│  │ ACP Front         │        │ Session Registry             │        │
│  │ acp-go-sdk Agent  │◄──────►│ ACP sid → {appServer sid,    │        │
│  │ (internal/acpfront│        │  workspace, model, turnCtx}  │        │
│  │ + internal/acpport│        └──────────────▲───────────────┘        │
│  └──────┬────────────┘                       │ normalized TurnEvents  │
│         │ AgentServicer calls                │ (chunk/tool/perm/done) │
│  ┌──────▼────────────────────────────────────┴───────────────────┐    │
│  │ Turn Translator (internal/core)                               │    │
│  │  · app-server events/messages → ACP session/update            │    │
│  │  · stopReason synthesis (end_turn/cancelled/…)                │    │
│  │  · per-turn context: cancel fan-out, idle detection           │    │
│  └──────┬────────────────────────────────────▲───────────────────┘    │
│         │ typed Requests                     │ typed Events           │
│  ┌──────▼────────────────────────────────────┴───────────────────┐    │
│  │ ZCode Backend Conn (internal/zcode — hand-rolled JSONL codec) │    │
│  │  · Request(id, method, params) → pending-call table           │    │
│  │  · event dispatch → subscribers                               │    │
│  │  · reverse-request router → Interaction Responder             │    │
│  └──────┬────────────────────────────────────────▲───────────────┘    │
│         │ child stdin                        child stdout            │
│  ┌──────▼────────────────────────────────────────┴───────────────┐    │
│  │ Supervisor (internal/supervisor)                              │    │
│  │  discovery → spawn (ELECTRON_RUN_AS_NODE=1 … zcode.cjs        │    │
│  │  app-server --stdio) → health/exit watch → restart w/ backoff │    │
│  └───────────────────────────────────────────────────────────────┘    │
│                                                                       │
│  Host services (internal/hostsvc)        Interaction (internal/       │
│  ┌────────────────┐ ┌────────────────┐   interact)                    │
│  │ CredentialVault│ │ RegistrySnap-  │   ┌────────────────────────────┐ │
│  │ keyring→GCM    │ │ shot builder   │   │ requestCredentialCipher →  │ │
│  │ decrypt        │ │ zcode-builtin: │   │   vault decrypt            │ │
│  │ ~/.zcode/v2/…  │ │ <rev>:<sha256> │   │ requestProviderRuntime     │ │
│  └────────────────┘ └────────────────┘   │   Headers → PAT client     │ │
│  ┌────────────────┐                      │   → {headersApplied:true,  │ │
│  │ PAT client     │                      │     requestAuth:{…}}       │ │
│  │ api.z.ai REST  │                      │ interaction/request-       │ │
│  └────────────────┘                      │ Permission → ACP client    │ │
│                                          │ requestRuntimePreferences  │ │
│  Logging: slog → stderr only             │ → defaults                 │ │
│  (stdout is protocol-exclusive)          └────────────────────────────┘ │
└───────────────────────────────────────────────────────────────────────┘
                  │  child stdin/stdout — ZCode JSONL, NOT JSON-RPC
                  │  (a `jsonrpc` key is rejected with -32600)
       ┌──────────▼──────────────────────────────┐
       │ zcode app-server (Electron-as-node)     │
       │ serial request queue; stop/cancel lane; │
       │ blocks on interaction/* while turning   │
       └─────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| ACP Front (`internal/acpfront`) | Serve ACP v1 over own stdio: `initialize`, `authenticate`, `session/new`, `session/prompt`, `session/cancel`; emit `session/update`, `session/request_permission` | `coder/acp-go-sdk` `AgentSideConnection` (see STACK.md); SDK types hidden behind `internal/acpport` so SDK churn doesn't bleed |
| Session Registry (`internal/core`) | One entry per ACP session: app-server sessionId, workspace ref, model selection, active turn state, cancel context | mutex-protected map; per-turn `context.Context` with cancel |
| Turn Translator (`internal/core`) | Normalize app-server output (messages/parts/tool/permission/state) into ACP `session/update` variants; synthesize `stopReason`; enforce "updates before prompt response" ordering | state machine per active turn; id-keyed tool_call table for coalescing `tool.updated` into `tool_call`/`tool_call_update` |
| Backend Conn (`internal/zcode`) | JSONL codec over child pipes; request/response correlation by `id` (string or int); event fan-out; reverse-request routing; method constants | `bufio.Scanner` + `encoding/json`; one writer goroutine (single-flush serialization mirrors app-server expectations); pending-call table |
| Supervisor (`internal/supervisor`) | Discover ZCode install, spawn child with right env, monitor exit, restart with backoff + crash window, staged shutdown | `os/exec`, `context`, restart policy (crash limit/window like codex-acp), child stderr → stderr logger |
| CredentialVault (`internal/hostsvc`) | Read key from Secret Service (`godbus`), AES-256-GCM decrypt `enc:v2:os:<kid>:<iv>.<tag>.<ct>` values from `~/.zcode/v2/credentials.json` | godbus v5 + stdlib crypto/aes; segment order iv.tag.ct (field-verified); key id = first 24 hex of sha256(key) |
| Registry Snapshot builder (`internal/hostsvc`) | Read active `zcode-builtin.json`, compute revision `zcode-builtin:<rev>:<sha256(abs path)>`, build accepted snapshot shape, push `provider/updateAccountConfig` | sha256 of resolved file path (source-confirmed: `zcode-builtin-provider-config-source.ts:41,241`); verify applied via registry state, not ack |
| PAT client (`internal/hostsvc`) | Mint Project Access Token for OAuth coding plan (getCustomerInfo → api_keys → access_tokens), cache with TTL | stdlib net/http; explicit IPv4-first resolver and proxy-bypass handling (field-verified node gotchas translate to Go client config) |
| Interaction Responder (`internal/interact`) | Answer every server→host request; policy per auth mode; forward permission to ACP client; emit cancellation notification on turn abort | handler registry keyed by method; correlation by `requestId`; deadline timers |
| Logging (`internal/logging`) | Structured debug trace of both protocol legs | `log/slog` → stderr, secret redaction, gated by `--log-level`/env |

## Recommended Project Structure

```
zcode-acp/
├── cmd/zcode-acp/           # main.go: flag/env parsing, logging init, run ACP server
├── internal/
│   ├── config/              # Config struct; precedence flags > env > defaults; discovery
│   ├── acpport/             # narrow port over acp-go-sdk types (upgrade isolation)
│   ├── acpfront/            # AgentServicer: initialize/authenticate/new/prompt/cancel
│   ├── core/                # SessionRegistry, TurnTranslator, TurnWatcher, mapping tables
│   ├── zcode/               # JSONL codec: envelope types, method consts, event types
│   │   └── fake/            # FakeAppServer: scriptable in-process app-server (tests)
│   ├── supervisor/          # discovery, spawn, env assembly, restart policy, shutdown
│   ├── hostsvc/             # credentials, registry snapshot, PAT client, zcode discovery
│   ├── interact/            # interaction/* handlers + policy (auto-answer vs forward)
│   └── logging/             # slog setup, redaction
└── testdata/                # golden protocol transcripts (.golden + -update flag)
```

**Structure rationale**

- **`internal/zcode` with `fake/` inside:** the codec is the first thing built and the thing most tested; the fake lives beside it so it can share envelope types and drift with the codec, not with tests.
- **`internal/acpport`:** acp-go-sdk is v0.x with deliberate API breaks between minors; one package owns every SDK import so a bump is a single-diff migration.
- **`internal/hostsvc` separate from `internal/interact`:** host capabilities (decrypt, snapshot, PAT) are pure, testable services; the interaction layer is policy wiring that calls them. This split is what makes the OAuth path testable without a keyring (inject a fake vault).
- **No `pkg/`:** nothing here is imported by outsiders; the binary is the product.

## Architectural Patterns

### Pattern 1: Dual-protocol bridge with a normalized internal event bus

**What:** Two protocol legs that share nothing on the wire meet in an internal, protocol-free middle: a
`TurnEvent` model (text delta / thought delta / tool call started-updated-finished / permission requested /
turn done(reason) / error). Each leg has its own codec; the translator converts in and out.
**When to use:** Always for protocol adapters. Never let leg A's types leak into leg B's code.
**Trade-offs:** One extra mapping layer, but both legs stay independently testable and the (likely) app-server
version drift between 3.17.x source and 3.18.2 wire lands in exactly one package.
**Example:**

```go
// internal/core/events.go
type TurnEvent interface{ Seq() uint64 }

type TextDelta struct{ seq uint64; Text string; Thought bool }
type ToolUpdated struct {
    seq     uint64
    CallID  string
    Title   string
    Status  ToolStatus // Pending, InProgress, Completed, Failed
    RawIn, RawOut json.RawMessage
}
type TurnFinished struct{ seq uint64; Reason FinishReason } // idle | failed | cancelled
```

### Pattern 2: Multiplexed JSONL connection with a pending-call table and a reverse-request router

**What:** The back leg is a single duplex JSONL stream carrying three message shapes: client→server requests
(`{id, method, params}`), server→client responses (`{id, result}` or `{id, error{code,message,data}}`), and
notifications/events (`{method, params}`, no id). Correlate requests by `id` (string *or* int on this protocol);
route events to per-session subscribers; route server→host requests to the interaction responder.
**When to use:** This is the entire back-leg design; nothing else fits a protocol that rejects the `jsonrpc` key.
**Trade-offs:** Hand-rolled (~200 lines) but mirrors `packages/zcode-protocol` shapes exactly; no library can do
this leg (STACK.md, HIGH confidence).
**Example:**

```go
// internal/zcode/conn.go — read loop must never block on handlers
func (c *Conn) readLoop(ctx context.Context) {
    sc := bufio.NewScanner(c.childStdout)
    sc.Buffer(make([]byte, 0, 1<<20), 16<<20) // tool outputs can be large; don't die on long lines
    for sc.Scan() {
        var m Envelope
        if err := json.Unmarshal(sc.Bytes(), &m); err != nil { c.logWarn(...); continue }
        switch {
        case m.ID != nil && m.Result != nil || m.Error != nil:
            c.pending.Resolve(*m.ID, m)          // complete in-flight call
        case m.ID != nil && m.Method != "":
            go c.interact.Serve(ctx, m)          // interaction/*: handler goroutine, never inline
        default:
            c.events.Dispatch(m)                 // state.updated, v4/conversation/*, …
        }
    }
}
```

Two load-bearing details copied from the app-server's own transport (source: `transport.ts`):
the app-server **serializes incoming requests** through one queue but has a **bypass lane for
`session/stop`** and lifecycle methods — so when the wrapper cancels, it must send `session/stop` (or the v4
cancel) *without waiting behind anything*, and expect in-flight stream output to continue briefly. And on
client EOF the app-server drains ~100 ms then exits — the wrapper's shutdown should close child stdin first
and wait, before SIGKILL.

### Pattern 3: Interaction responder as a policy object

**What:** Every server→host request gets a handler; the handler set is chosen by auth mode and flags.
`requestCredentialCipher` → CredentialVault decrypt; `requestProviderRuntimeHeaders` → PAT client (OAuth mode)
or header assembly (api-key mode), answering `{headersApplied:true, requestAuth:{apiKey, apiKeyId}}` —
`headersApplied:false` is never acceptable for account providers (field-verified: turn dies with
"Provider runtime headers were not applied"); `session/requestRuntimePreferences` → static
`{nativeSearchEnhancementsEnabled:false, memoryEnabled:false}`; `interaction/requestPermission` → forwarded
to the ACP client as `session/request_permission`, mapping decision `allow|deny|escalate|modify` and option
kinds; `interaction/requestOfficialMcpAuthHeaders` → `{headers:{}}` stub in v1.
**When to use:** Always; the policy object is also where "headless auto-allow" vs "always ask the ACP client"
is decided.
**Trade-offs:** Foregrounding policy keeps the risky shortcut (auto-allow everything) contained and auditable.
**Example:**

```go
// internal/interact/responder.go
func (r *Responder) Serve(ctx context.Context, req zcode.Envelope) {
    switch req.Method {
    case zcode.MethodRequestRuntimePreferences:
        r.conn.Reply(req.ID, zcode.RuntimePreferencesResult{Memory:false, NativeSearch:false})
    case zcode.MethodRequestProviderRuntimeHeaders:
        ctx, cancel := context.WithTimeout(ctx, 180*time.Second) // source-confirmed server timeout
        defer cancel()
        auth, err := r.pat.Ensure(ctx, req.Params.SessionID)
        if err != nil { r.conn.ReplyError(req.ID, err); return }
        r.conn.Reply(req.ID, zcode.ProviderRuntimeHeadersResult{
            HeadersApplied: true,
            RequestAuth:    zcode.RequestAuth{APIKey: auth.Token, APIKeyID: auth.KeyID},
        })
    case zcode.MethodRequestPermission:
        r.permission.ForwardToACP(ctx, req) // correlates app-server requestId ↔ ACP request
    }
}
```

Permission cancellation has two directions and both must be wired: ACP client cancels the turn → the wrapper
must answer the pending app-server `interaction/requestPermission` (field doc: `{decision:"allow"}` shape,
or deny) *and* send `session/stop`; the app-server, on its own abort/timeout, emits the
`interaction/providerRuntimeHeadersCancelled` notification (source-confirmed) which the wrapper uses to drop
its in-flight PAT mint.

### Pattern 4: Supervisor with health watch, crash window, and staged shutdown

**What:** Spawn the child once per wrapper lifetime; watch stderr (log) and exit; on unexpected exit, restart
with capped backoff inside a crash window, then fail remaining ACP turns with a clear error instead of
looping. On wrapper EOF/INT: stop accepting ACP requests → notify sessions → close child stdin → wait
(EOF drain is ~100 ms by design) → SIGKILL after grace.
**When to use:** Always; the child is a long-lived stateful process and ACP clients treat the wrapper as the agent.
**Trade-offs:** Restarting loses *all* live app-server sessions (process-global state) — codex-acp accepts the
same blast radius (MEDIUM confidence, its README documents global replacement with crash counting
`CODEX_ACP_APP_SERVER_CRASH_LIMIT`/`CRASH_WINDOW_MS`). Sessions with persisted ZCode history can be
reattached later via `session/resume`; v1 does not advertise ACP `loadSession`, so restart mid-session is
surfaced as a failed turn, not silent replay.
**Example:**

```go
// internal/supervisor/supervisor.go
type Policy struct{ CrashLimit int; CrashWindow time.Duration; BaseBackoff time.Duration }

func (s *Supervisor) run(ctx context.Context) error {
    crashes := 0
    for {
        cmd := s.spawnCommand()          // ELECTRON_RUN_AS_NODE=1, ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=…
        if err := s.startAndBootstrap(ctx, cmd); err != nil { /* log */ }
        err := s.conn.Wait()
        if ctx.Err() != nil { return ctx.Err() } // intentional shutdown
        crashes++
        if crashes > s.policy.CrashLimit { return fmt.Errorf("app-server crash loop: %w", err) }
        select { case <-time.After(s.policy.BaseBackoff << (crashes - 1)): case <-ctx.Done(): }
        s.registry.FailAllTurns(errors.New("app-server exited; session state lost"))
    }
}
```

### Pattern 5: Subscribe-first turn watching with a polling fallback

**What:** Per active turn, obtain app-server output in this priority order: (1) `session/subscribe` with
`deliveryKind:"desktop-continuous"` — 3.17.x source shows event envelopes `part.delta`, `part.upserted`,
`tool.updated`, `turn.completed/failed` with sequence numbers and `includeSnapshot`; (2) `v4/conversation/subscribe`
→ `v4/conversation/frame` notifications (method family source-confirmed; presence on the 3.18.2 wire observed
in FINDINGS but unexercised by the shim — MEDIUM); (3) fallback: poll `session/messages` with delta tracking
plus idle detection (field-verified shim used "text present + 2.5 s without new parts" because
`state.updated.patch.status==='idle'` did not always arrive).
**When to use:** Implement (3) first — it is the only path proven end-to-end on 3.18.2 — then layer (1)/(2) in a
later phase behind the same `TurnWatcher` interface when re-verified live.
**Trade-offs:** Polling adds latency and requires idle heuristics; event streams give exact sequences and
`turn.completed` but need re-verification against the shipped 3.18.x bundle. The interface keeps that swap local.
**Example:**

```go
// internal/core/turnwatcher.go — one implementation, swappable strategy
type TurnWatcher interface {
    Stream(ctx context.Context, sessionID string, sink func(TurnEvent)) error
    // implementations: subscribeWatcher{}, v4FrameWatcher{}, pollWatcher{Idle: 2500*time.Millisecond}
}
```

### Pattern 6: Fake app-server for protocol tests (testability contract)

**What:** `internal/zcode/fake` is a scriptable app-server implementing the same envelope grammar: it accepts
requests from a `Conn` wired over `io.Pipe`/`net.Pipe`, replies from a scripted table, emits configured events,
and — critically for this domain — *issues reverse `interaction/*` requests and blocks the turn until answered*,
with configurable timeouts. Golden transcripts (`testdata/*.golden`, `-update` flag) freeze whole turns:
`session/prompt` in, chunk/tool/permission messages out, in both directions.
**When to use:** CI protocol gate (PROJECT.md: done v1 = "CI-тесты протокола (мок app-server)"). The fake is
the mock. For the few tests that must exercise real `os/exec` pipes and process lifecycle, use the stdlib
helper-process idiom (`TestMain` re-exec with an env flag) — no extra framework (STACK.md, HIGH confidence in
pattern).
**Trade-offs:** The fake can drift from the real 3.18.2 child; mitigate by replaying captured real transcripts
(generated during Phase 1 field research) as fake scripts, and by the VM-level E2E tier that runs the real binary.

## Data Flow

### Startup and bootstrap

```
ACP client spawns zcode-acp
    ↓
config resolve (flags > env > discovery) — locate zcode.cjs + builtin file
    ↓
Supervisor: spawn app-server (ELECTRON_RUN_AS_NODE=1, ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=<file>)
    ↓ (child up — app-server waits for first request; no handshake frame is required)
[OAuth mode] Hostsvc: push provider/updateAccountConfig (snapshot, revision zcode-builtin:<rev>:<sha256>)
    → verify applied via registry state / state.updated patch.model.available — ack "received" is NOT proof
[OAuth mode] PAT client: prewarm token (getCustomerInfo → api_keys → access_tokens)
    ↓
ACP Front answers initialize {protocolVersion:1, agentCapabilities, authMethods, agentInfo}
    ↓ (optional) authenticate {authMethodId: "zai-api-key" | "zai-oauth-coding-plan"}
ready for session/new
```

### session/new

```
ACP session/new {cwd, mcpServers}
    ↓ Translator: cwd → workspace {workspacePath, workspaceKey}; never pass `model` in create
zcode session/create {workspace, mode, persistence, titleGenerationEnabled}
    ↓ result.session.sessionId  (= ACP sessionId, 1:1)
[thinking model chosen] zcode session/setModel {providerId, modelId, options:{reasoningLevel}}
    ↓ (session/create drops options.reasoningLevel — field-verified trap)
ACP session/new → {sessionId}
```

### A turn (happy path)

```
ACP session/prompt {sessionId, prompt:[blocks]}           ACP client
    ↓ Translator: blocks → content string (text join; images only if advertised)
zcode session/send {sessionId, content}                     │
    ↓ {sessionId, accepted:true, stateRevision}  (fast ack — turn continues async)
events/polls: message parts, tool updates …                │
    ↓ Translator maps:                                     │
      text part delta → session/update agent_message_chunk │ (streamed during turn)
      reasoning part delta → session/update agent_thought_chunk
      tool.updated → session/update tool_call (first) / tool_call_update (subsequent)
    ↓ finish signal (turn.completed | idle | quiet-window)
ACP session/prompt → {stopReason:"end_turn"}               ▼
```

### Permission round-trip

```
app-server: interaction/requestPermission {requestId, sessionId, toolCallId, toolName,
             riskLevel, options:[{optionId,name,kind}]}
    ↓ Interaction responder correlates (requestId ↔ in-flight ACP request)
ACP session/request_permission {sessionId, toolCall:{toolCallId}, options:[{optionId,name,kind}]}
    ↓ client answers {outcome:{outcome:"selected", optionId}} | {outcome:"cancelled"}
map ACP optionId → app-server response {decision:"allow"|"deny"|…} → reply to app-server
    ↓ (on ACP turn cancel meanwhile: answer the pending request + session/stop)
```

### Cancellation

```
ACP session/cancel (notification, no response)
    ↓ Translator: cancel turn context → stop watcher, fail pending prompt with stopReason:"cancelled"
zcode session/stop {sessionId}   ← bypass-lane method on the app-server; send immediately, don't queue
    ↓ drain in-flight session/update notifications (allowed until prompt response — spec rule)
ACP session/prompt → {stopReason:"cancelled"}
    ↓ answer any pending interaction/requestPermission for this turn (deny/cancel outcome)
```

### Failure flows

- **Child exits mid-turn:** Supervisor restarts per Policy; all registered turns fail with a protocol error;
  ACP prompt gets `stopReason` reflecting failure or a JSON-RPC error if no turn was active.
- **Interaction unanswered:** app-server holds the turn; wrapper enforces per-method deadlines
  (180 s provider-runtime-headers source-confirmed) and returns a readable ACP error rather than hanging.
- **PAT mint fails / keyring missing:** OAuth-mode turns fail fast with `headersApplied:false`-avoiding
  early error at bootstrap, not mid-turn; api-key mode unaffected.

## One Process vs Per Session — the explicit recommendation

| Criterion | One shared app-server (recommended) | One app-server per ACP session |
|-----------|-------------------------------------|--------------------------------|
| Memory (Electron-as-node child) | One child, amortized | N × heavy child |
| Provider registry / OAuth snapshot | Pushed once per process | Pushed N times; snapshot race-prone |
| PAT minting | Cached once, shared | Per child; api.z.ai churn |
| Crash blast radius | All sessions (mitigated by restart + resume) | One session |
| Matches ZCode desktop host | Yes (single app-server, many sessions) | No |
| Matches ACP client expectations | Yes (one agent process, many sessions — Orca/Zed spawn one) | Divergent |
| Cancel/stop semantics | Per-session on a shared serial queue; app-server already has a stop bypass lane | Simpler but unneeded |
| Precedent | codex-acp (ACP org's own app-server adapter) — MEDIUM confidence | No known ACP adapter does this |

**Recommendation: one shared child per wrapper process; ACP sessionId = app-server sessionId, 1:1, no alias
persistence.** ACP v1 `session/load` is optional and not advertised in v1, so no session persistence across
wrapper restarts is owed to clients; ZCode-side session history can still be reached later via
`session/resume` in a future milestone.

## Configuration Surface

Precedence: **flags > environment > discovered defaults**. No config file in v1 (three inputs don't justify one;
STACK.md agrees). Secrets only via env, never argv (codex-acp precedent: headers/env "so that they are not in
the process arguments" — MEDIUM confidence).

| Setting | Flag | Env | Default / discovery |
|---------|------|-----|---------------------|
| ZCode install root | `--zcode-home` | `ZCODE_ACP_HOME` | Search: unpacked AppImage dirs (`~/Applications/zcode-*/squashfs-root`), `/tmp/.mount_ZCode-*` (avoid — volatile), deb path; verify `<root>/resources/glm/zcode.cjs` exists |
| Explicit builtin config file | `--zcode-builtin-file` | `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` | Newest `zcode-builtin.json` under `~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/endpoint-*/`; the child **needs** this env (field-verified launch line) |
| Default model | `--model` | `ZCODE_ACP_MODEL` | `provider/model` string, optional `@level` (e.g. `account:zai-individual-coding-plan/GLM-5.3@high`); empty = app-server personal-config default |
| Auth mode | `--auth-mode` | `ZCODE_ACP_AUTH_MODE` | `auto` (api-key if credentials.json has `api-key`; else OAuth if `oauth:zai:access_token`), `api-key`, `oauth` |
| OAuth token endpoint origin | (none) | `ZCODE_ACP_ZAI_ORIGIN` | `https://api.z.ai` |
| Log level / trace | `--log-level` | `ZCODE_ACP_LOG_LEVEL`, `ZCODE_ACP_DEBUG=1` | `info` to stderr; `debug` traces both protocol legs (redacted) |
| Orca mimicry surface | (subcommand) | — | `acp` subcommand = ACP server (default when argv empty? — Phase 1 decides against `omp acp` exact shape); `--version` prints mimic-compatible version string (Phase 1 research item) |

The app-server child additionally inherits a scrubbed environment (notably: strip dead `HTTPS_PROXY` or
exempt `api.z.ai` — field-verified fetch breaker; the Go PAT client sets its own transport anyway, but the
*child* Node fetch needs the env fix).

## Scaling Considerations

"Scale" here = concurrent ACP sessions and long-lived child health, not users.

| Concern | 1–3 sessions | ~10 sessions | More / long-running |
|---------|--------------|--------------|---------------------|
| Child CPU/memory | Fine — one Electron-as-node process | Monitor via child stderr/`state.updated`; still one process | Consider per-workspace children (same architecture, keyed by workspaceKey) — only if profiled need arises |
| JSONL throughput | Scanner with raised buffer cap (tool outputs are large lines) | Same; single writer goroutine is not a bottleneck at chat rates | If backpressure appears (the app-server itself drops `process/networkRequests` beyond a 256 KB queue — source-confirmed precedent), apply the same drop-oldest policy to non-essential notifications |
| Concurrent interactions | Trivially concurrent handlers | Same | PAT cache with singleflight mint (avoid stampede at 180 s timeout edges) |
| Turn watcher | One poll timer per active turn | Cheap | Event-driven subscribe path removes poll timers entirely |
| Restart blast radius | Acceptable (fail turns loudly) | Worse — restart policy must cap crash loop early | Add optional per-session child isolation behind a flag (architecture already supports: registry keyed sessions → spawn key) |

**First bottleneck in practice:** not throughput — it's *correctness under concurrency*: interleaved turns on
two sessions sharing one serial app-server queue (fine, `session/send` acks immediately) while interaction
handlers race (PAT refresh vs concurrent turn starts). Design singleflight + cache before optimizing anything else.

## Anti-Patterns

### Anti-Pattern 1: JSON-RPC framing on the back leg

**What people do:** Reuse the ACP JSON-RPC codec (or sourcegraph/jsonrpc2) for the app-server leg.
**Why it's wrong:** The app-server rejects the `jsonrpc` key (`-32600 unrecognized_keys` — field-verified);
every request would fail or the codec would mis-handle the response/event union (events have no id; responses
use string *or* int ids).
**Do this instead:** Own JSONL codec in `internal/zcode`, mirroring `packages/zcode-protocol` shapes.

### Anti-Pattern 2: Writing anything but protocol frames to ACP stdout

**What people do:** `fmt.Println` diagnostics, child stdout passthrough, panic dumps.
**Why it's wrong:** ACP framing is newline-delimited JSON on stdout; one foreign line desyncs the client
(spec: nothing non-ACP may be written to stdout).
**Do this instead:** All logs → stderr (`log/slog`); child stderr → stderr; child stdout is protocol-only.

### Anti-Pattern 3: Blocking the JSONL read loop (deadlock by interaction)

**What people do:** Handle `interaction/*` inline in the read goroutine, or wait for the ACP client's
permission answer synchronously before reading the next child line.
**Why it's wrong:** The app-server *blocks mid-turn* until the wrapper answers; meanwhile it emits events and
other reverse requests (including the 1 s re-announce pattern — source-confirmed). A blocked read loop
deadlocks the whole process (this is precisely why the app-server's own transport has a bypass lane for
`session/stop`).
**Do this instead:** Handler goroutines per reverse request; read loop only dispatches; permission answers
arrive asynchronously and are correlated by `requestId`.

### Anti-Pattern 4: Trusting `provider/updateAccountConfig` ack

**What people do:** Treat `{"status":"received"}` as "snapshot applied".
**Why it's wrong:** Field-verified: the ack arrives even when the snapshot is silently discarded (revision
mismatch or zod rejection); models then never materialize and turns fail with "Provider Registry 中不存在 Model".
**Do this instead:** Verify by reading back registry state (`state.updated` patch / `provider/refreshRegistry`
result) and by checking the target provider appears before `session/setModel` to it.

### Anti-Pattern 5: Sending the model in `session/create` for thinking models

**What people do:** `session/create {model:{providerId, modelId, options:{reasoningLevel}}}`.
**Why it's wrong:** `formatProtocolModelSelection()` drops `options` — create fails with "Reasoning level is
required" for every thinking model (field-verified).
**Do this instead:** Create bare, then `session/setModel {providerId, modelId, options:{reasoningLevel}}`.

### Anti-Pattern 6: One child per ACP session

**What people do:** "Isolation" via spawning an app-server per session.
**Why it's wrong:** Process-global registry duplicated N times; N× Electron memory; PAT minted N times;
contradicts both the desktop host model and the codex-acp precedent.
**Do this instead:** Shared child + Session Registry (see the recommendation table above).

### Anti-Pattern 7: Single-signal turn completion

**What people do:** Finish the ACP turn only on `state.updated.patch.status==='idle'` (or only on
`turn.completed`).
**Why it's wrong:** Field-verified: `idle` did not always arrive on 3.18.2; conversely event streams may not
exist on every bundle version.
**Do this instead:** Completion = union of signals: explicit finish event → idle patch → quiet-window fallback;
each behind the `TurnWatcher` interface.

### Anti-Pattern 8: Queueing cancellation behind work

**What people do:** Send `session/stop` through the same serialized outbound queue as prompts.
**Why it's wrong:** The app-server handles `session/stop` in a bypass lane precisely so cancel can interrupt a
running model request (source-confirmed); queueing it in the wrapper delays abort by one whole turn in the
worst case and violates ACP's "halt promptly" expectation.
**Do this instead:** Stop/cancel messages jump the write queue; the turn context also aborts the watcher and
answers pending permissions immediately.

### Anti-Pattern 9: Secrets in argv or logs

**What people do:** Pass the PAT/decrypted tokens as CLI args to the child, or debug-log envelope params
wholesale.
**Why it's wrong:** argv is visible in `ps`; FINDINGS-level secrets (OAuth access token, PAT) in logs defeat
the whole "official paths only" security posture of the project.
**Do this instead:** Child env assembly in-process (not via shell), redaction list (`apiKey`, `token`,
`access_token`, cipher blobs) in the logging package.

## Integration Points

### External services and surfaces

| Surface | Integration pattern | Notes / gotchas |
|---------|--------------------|-----------------|
| Secret Service (gnome-keyring) | godbus v5, `secret-tool`-equivalent lookup: service `"ZCode Credential Encryption"`, username `sha256(realpath(credentials.json))` | v1 Ubuntu GNOME only (PROJECT.md); fallback `credential-secret` file exists in app code but is absent on target machines; all D-Bus work behind `CredentialVault` interface for fakes |
| `~/.zcode/v2/credentials.json` | Read + AES-256-GCM decrypt `enc:v2:os:<kid>:<iv>.<tag>.<ct>` | Segment order iv.tag.ct (field-verified — `iv.ct.tag` fails); kid = first 24 hex of sha256(key); keys of interest: `oauth:zai:access_token`, `api-key`, `oauth:active_provider`. 3.17.x source shows an older `enc:v1:` scheme with `ZCODE_CREDENTIAL_SECRET` env fallback — version drift; trust the v2 field facts (HIGH) |
| api.z.ai REST | Three-call PAT mint (getCustomerInfo → api_keys GET/POST → access_tokens POST); raw OAuth token without Bearer on call 1, Bearer on 2/3 | Field-verified shapes in FINDINGS 3.3; pick org/project by "默认机构"/"默认项目" contains-else-first; Go client: IPv4-first dialer, proxy handling independent of env |
| ZCode app-server child | JSONL over stdio, spawn `ELECTRON_RUN_AS_NODE=1 <electron-or-appimage> zcode.cjs app-server --stdio` with `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` | No handshake frame; first request may come immediately; EOF drain ~100 ms; rejects `jsonrpc` key |
| ACP client (Orca/Zed/…) | acp-go-sdk AgentSideConnection over own stdio; protocolVersion 1; advertise minimal capabilities (no loadSession, no fs/terminal in v1) | Orca hardcodes agents by name+version — mimicry surface (name, `--version`, `acp` subcommand) is CLI-layer, not architecture (Phase 1 research item) |
| `zcode-builtin.json` runtime file | Read for revision + provider list; path feeds revision sha256 | Located under `~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/endpoint-*/`; also may be passed to the child via env |

### Internal boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| acpfront ↔ core | typed method calls + TurnEvent channel | acpfront never sees app-server types |
| core ↔ zcode | typed Request/Event API on `Conn` | core never sees raw envelopes |
| interact ↔ hostsvc | service calls (vault.Decrypt, pat.Ensure, snapshot.Build) | policy lives in interact, capability in hostsvc |
| supervisor ↔ zcode | owns pipes, hands `Conn` a ReadWriteCloser | `Conn` is spawn-agnostic — takes any pipes, enabling the fake |

## Build Order Implications (for roadmap)

Dependency-driven; each step is testable without the next:

1. **JSONL codec + fake app-server + golden tests** — everything else imports `internal/zcode`; the fake makes
   every later step testable in CI without the real bundle. (Patterns 1, 2, 6)
2. **Supervisor + real child smoke** — discovery, env assembly, spawn real 3.18.2 child, `session/create` +
   `session/setModel` against it; capture real transcripts to replay into the fake. (Pattern 4)
3. **ACP front minimal loop** — initialize/session/new/session/prompt with message + thought chunks over the
   polling TurnWatcher; api-key path only, no interactions yet. Ends: first real streamed turn in any ACP client.
   (Patterns 1, 5)
4. **Interaction responders, api-key E2E** — requestRuntimePreferences, requestCredentialCipher,
   requestProviderRuntimeHeaders (`{headersApplied:true…}`), requestPermission forwarded to ACP; cancel +
   stop bypass. Ends: full PROJECT.md chat cycle on `zai-api`. (Pattern 3)
5. **OAuth coding-plan path** — CredentialVault (keyring+GCM), snapshot builder + push + verify, PAT client
   with cache/singleflight; `headersApplied` failures surfaced at bootstrap. Ends: `account:*` models work.
6. **Turn fidelity hardening** — tool_call/tool_call_update coalescing, permission option-kind mapping,
   cancellation edge cases, restart-mid-turn behavior, subscribe-based TurnWatcher if re-verified on 3.18.2.
7. **Config surface + Orca mimicry + docs** — flags/env per table above, `acp` subcommand + `--version` fidelity
   (needs Phase 1 Orca findings), README recipes, VM E2E.

Phases 1–3 unblock the highest-risk unknowns earliest (wire truth of 3.18.2 events, ACP client interop);
the OAuth path is deferred until the api-key cycle proves the plumbing, matching PROJECT.md's
"research-and-document first" decision.

## Open Questions (phase-specific research needed)

- **Exact 3.18.2 event stream surface:** does the shipped bundle accept `session/subscribe` /
  `v4/conversation/subscribe` and deliver the 3.17.x-style event envelopes? The shim never exercised them
  (it polled). Determines whether the subscribe-first TurnWatcher replaces polling. (MEDIUM confidence today)
- **`workspaceKey` semantics:** the wire schema requires it; the shim passed a value successfully, but its
  derivation rules (hash of path? host-assigned?) are unconfirmed. Verify in Phase 1. (LOW confidence)
- **Orca load-bearing call set at card start** (which ACP methods it actually invokes) — decides mimicry
  flags and capability advertisement; local Orca chunks to read. (Local source, HIGH once read)
- **ZCode install discovery matrix** (AppImage vs deb layouts, version pinning, auto-update side effects on
  `zcode-builtin.json` revision) — Phase 1 field work. (LOW until verified)
- **App-server idle/exit behavior under long idle:** does the child self-exit or stay resident? Affects
  Supervisor health strategy (no liveness frame exists in the protocol — only request/response). (LOW)

## Sources

- **Local, HIGH (field-verified on ZCode 3.18.2):** `FINDINGS.md (private research notes, outside this repo)` — app-server JSONL
  protocol, host-managed mode, snapshot revision format, PAT flow, `session/create` reasoningLevel trap,
  Orca ACP v1 client facts.
- **Local, HIGH (field-verified):** `local field-notes references/app-server-stdio-protocol.md`
  — JSONL envelope grammar, interaction reply shapes (`requestCredentialCipher` reply union,
  `{decision:"allow"}`, runtime-preferences defaults), credential v2 blob format + Secret Service lookup,
  turn-completion quiet-window finding, `provider/updateAccountConfig` ack-vs-applied trap.
- **Local, HIGH (field-verified):** `local field-notes references/oauth-coding-plan-shim.md`
  — accepted snapshot shape, PAT mint call sequence, `headersApplied:false` rejection, IPv6/proxy gotchas.
- **Local, HIGH (source tree 3.17.x):** `ZCode source tree (github.com/zai-org/ZCode; local clone)` —
  `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/transport.ts` (JSONL framing, serial queue, stop
  bypass, EOF drain), `server-types.ts` (sessions map, error codes −32020/21/22, requestClient), `interaction-broker.ts`
  (1 s re-announce, guarded errors), `provider-runtime-headers.ts` (180 s timeout, cancellation notification),
  `packages/shared/src/zcode-protocol/index.ts` (method/notification/event schemas, `state.updated` shape,
  deliveryKind enum, workspaceRef schema), `zcode-protocol-legacy-types.ts` (permission decisions), `packages/provider-node/src/zcode-builtin-provider-config-source.ts`
  (revision string), `packages/services/src/credential/providers/credentialCipherProvider.ts` (v1 scheme —
  version drift noted), `packages/shared/src/zcode-protocol-v4/transport.ts` (v4/conversation methods).
- **Web, HIGH (official spec):** agentclientprotocol.com v1 pages —
  [overview](https://agentclientprotocol.com/protocol/v1/overview.md),
  [transports](https://agentclientprotocol.com/protocol/v1/transports.md) (newline-JSON, UTF-8, stdout exclusivity),
  [initialization](https://agentclientprotocol.com/protocol/v1/initialization.md),
  [session-setup](https://agentclientprotocol.com/protocol/v1/session-setup.md) (cwd rules, auth_required),
  [prompt-turn](https://agentclientprotocol.com/protocol/v1/prompt-turn.md) (update variants, stopReasons,
  "updates before response" cancellation rule),
  [tool-calls](https://agentclientprotocol.com/protocol/v1/tool-calls.md) (kinds, statuses, permission options
  allow/reject × once/always, outcome selected/cancelled).
- **Web, MEDIUM (adapter precedent):** [agentclientprotocol/codex-acp](https://github.com/agentclientprotocol/codex-acp)
  + its `readme-dev.md` — one shared app-server process, crash-window restart policy, provider-routed lifetime,
  env-not-argv for headers; [zed-industries/codex-acp](https://github.com/zed-industries/codex-acp) (archived,
  superseded); [Zed ACP blog](https://zed.dev/blog/bring-your-own-agent-to-zed).

---
*Architecture research for: zcode-acp (Go ACP v1 server wrapping ZCode Desktop app-server)*
*Researched: 2026-11-19*
