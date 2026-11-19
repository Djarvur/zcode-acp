# Phase 4: ACP Front — Minimal Chat Loop - Research

**Researched:** 2026-11-20
**Domain:** ACP v1 front leg in Go — `coder/acp-go-sdk` seam (`internal/acpport`), agent loop (`internal/acpfront`), live streaming over the Phase-3 turn machine, wired against the fake app-server
**Confidence:** HIGH for the SDK API surface and wire shapes (read from the actual v0.20.0 module source + executed marshal probe), HIGH for the back-leg delta carriers (read from the local zcode-src 3.17.x tree + shipped-bundle taxonomy already verified in Phase 1), MEDIUM for ACP spec semantics (official pages via fetch, quoted), MEDIUM for live app-server streaming behavior (no completed real turn has ever been captured — see A1).

## Summary

This phase composes the first vertical integration: a real ACP client → `internal/acpport` → `internal/acpfront` → `internal/turn` machine → `internal/zcode` conn → fake app-server, wire-asserted at both legs. The research question D-01 posed — SDK vs hand-roll — is answered: **the SDK audit PASSES on every axis**, so `coder/acp-go-sdk` v0.20.0 is the implementation base, hidden behind `internal/acpport` exactly as ARCHITECTURE.md prescribes. The audit found one factual correction to STACK.md: v0.20.0's generated types pin **schema 1.32.0** (`acp.SchemaVersion`, read from `constants_gen.go:9` and confirmed by the v0.20.0 release notes "ACP schema stays at 1.32.0"), not 1.25.0 — the README's "schema 1.25.0" figure is the *stable-interface baseline* (which methods sit on the required `Agent` interface). Wire `protocolVersion` is 1 either way, so Orca compatibility is unaffected.

The decisive architectural discovery is **where content deltas come from**. Neither the Phase-3 conn nor the turn machine interprets content (the poll source deliberately wraps `session/messages` opaquely). The delta carrier is the `session/event` notification stream: `type:"part.delta"` with `payload{messageId, partId, field, delta}` where `field` discriminates `"text"` (→ `agent_message_chunk`) from `"reasoning"` (→ `agent_thought_chunk`) — schema read verbatim from the zcode-src protocol types. Critically, the app-server **only pushes `session/event` notifications once the session has an active `session/subscribe`** (`record.deliveryKind` guard, source-verified), so the front must subscribe after `session/create`. Deltas are wire-ordered before the prompt response automatically once the front taps notifications on the same demux path the machine uses — the flush-before-stopReason contract (CORE-03, D-04) falls out of single-path ordering plus the SDK's FIFO write mutex.

Two Phase-3 realities shape the plan: (1) `zcode.Conn.OnNotification` is a **single slot** that `NewMachine` overwrites per machine — multi-session support requires an ordered observer fan-out in `internal/zcode` (small, test-first change); (2) the SDK maps a handler's `context.Canceled` to a `-32800` ERROR response, but ACP requires the prompt request answered with `stopReason:"cancelled"` — the Prompt handler must own its turn context and convert cancellation into a success response, never return the ctx error.

**Primary recommendation:** SDK-first per D-01 audit PASS; pin `github.com/coder/acp-go-sdk@v0.20.0` exactly; `internal/acpport` owns every SDK import and translates to neutral types consumed by `internal/acpfront`; add an ordered notification fan-out to `internal/zcode.Conn`; subscribe (`deliveryKind:"desktop-continuous"`) after each `session/create`; tap `part.delta` (`field:"text"/"reasoning"`) into `conn.SessionUpdate` in receive order; own the Prompt turn-context so cancel resolves `stopReason:"cancelled"` exactly once.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01 (ACP Leg Implementation Base):** Decided at planning by the Package Legitimacy Audit between (a) `coder/acp-go-sdk` v0.20.0 and (b) hand-rolling ACP v1 JSON-RPC 2.0 framing on Phase-3 codec patterns. **Default (a) if the audit passes; (b) is the pre-approved fallback.** Either way the seam is `internal/acpport` — the front leg never imports the SDK (or its hand-rolled twin) outside that package. Decision + audit evidence recorded in RESEARCH.md before plans land. **→ Audit PASSED; verdict recorded in this file (section below).**
- **D-02 (Capability Honesty):** `initialize` advertises EXACTLY what exists: `protocolVersion: 1`, both `authMethods` (`zai-api-key`, `zai-individual-coding-plan` — advertised now, functional in Phases 5–6), real `agentInfo` (name `zcode-acp`, version from the binary's version var), NO fs/terminal capabilities, NO slash commands, NO readTextFile/writeTextFile — omission is spec-correct. Anything not implemented is not advertised.
- **D-03 (session/new Contract):** `session/new{cwd, mcpServers, ...}` → backend `session/create` (workspaceKey = cwd path) → follow-up `session/setModel` carrying model + reasoning level (the create-drops-reasoningLevel workaround); `mcpServers` accepted-and-ignored; the follow-up setModel visible on the wire is asserted by a fake transcript (CORE-02).
- **D-04 (Streaming Contract):** `session/prompt` → backend `session/send` fire-and-forget lifted by the Phase-3 machine; deltas stream to the client as `session/update` (`agent_message_chunk` / `agent_thought_chunk`) LIVE while the turn runs — never buffered-then-flushed; `stopReason` returns only after all pending updates flushed (CORE-03); `agent_thought_chunk` emitted ONLY on backend reasoning events, never synthesized (CORE-04). Update ordering = receive order (single demux path).
- **D-05 (Cancel Contract):** `session/cancel` → Phase-3 bypass `conn.Stop` + machine cancel (exactly-once); the prompt call resolves `stopReason: "cancelled"` exactly once; cancel-after-complete is a no-op; a second concurrent prompt while a turn runs is rejected (wrapper guard + backend -32010 mapping already in place) (CORE-05).

### Claude's Discretion
- acpport seam API shape; test structuring (fake-driven round-trips extend Phase 3 patterns)
- agentInfo field details; error message wording on the ACP surface
- Wave/task decomposition within the locked plan structure

### Deferred Ideas (OUT OF SCOPE)
- `session/request_permission` bridging — Phase 5 (TOOL-02)
- elicitation/create preference — Phase 5+ (client capability-gated)
- model/thought configOptions — Phase 6 (MODEL-01/02)
- subscribe-based TurnWatcher — reserved slot (Phase 3 D-02)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CORE-01 | `initialize` replies `protocolVersion: 1`, honest capability set, real `agentInfo`, two `authMethods` | SDK `InitializeResponse{ProtocolVersion, AgentCapabilities, AgentInfo *Implementation, AuthMethods}` verified in module source; executed marshal probe shows the exact honest JSON (no fs/terminal/loadSession/prompt fields; `authMethods:[{id,name},…]`; `protocolVersion:1`) — section "initialize response wire shape". Spec basis: omitted capabilities = MUST-UNSUPPORTED |
| CORE-02 | `session/new{cwd}` → `session/create` + follow-up `session/setModel`; `mcpServers` accepted-and-ignored | SDK `NewSessionRequest{Cwd, McpServers}` (nil → `[]` on marshal) verified; setModel workaround + refusal trap documented (dialect §2, V-EXP EXP-TURN-01); fake scenario pattern for the transcript assertion designed (Validation Architecture) |
| CORE-03 | `session/prompt` streams `agent_message_chunk` deltas live, `stopReason` only after flush | Delta carrier = `session/event` `part.delta{field:"text"}` (zcode-src schema verbatim); requires `session/subscribe` (deliveryKind guard); SDK FIFO write mutex gives wire ordering; spec cancellation-ordering rule quoted; proof-test design in Validation Architecture |
| CORE-04 | `agent_thought_chunk` streamed when (and only when) backend emits reasoning | `field:"reasoning"` delta → `UpdateAgentThoughtText(delta)` — never synthesized; negative test = scenario with zero reasoning deltas must produce zero thought chunks (probe output: chunk wire shapes verbatim) |
| CORE-05 | `session/cancel` stops the turn, replies `stopReason:"cancelled"` exactly once | SDK `registerPrompt` cancels the Prompt ctx + calls `Agent.Cancel`; machine `Cancel()` = bypass `session/stop` + exactly-once resolution (Phase 3 proven); pitfall: never return ctx.Canceled (SDK would emit -32800 error); turn-context ownership pattern in Architecture Patterns |
</phase_requirements>

## D-01 Verdict: Package Legitimacy Audit — SDK-first (PASS)

**Verdict: (a) `coder/acp-go-sdk` v0.20.0 — AUDIT PASSED on every axis. Hand-roll fallback (b) not exercised.**

Audit method: the Go ecosystem is not covered by the `gsd-tools package-legitimacy check` seam (npm/pypi/crates only — verified: the seam rejects `--ecosystem go`), so the audit was performed manually per the phase's research priorities: module fetched via `go get github.com/coder/acp-go-sdk@v0.20.0` (succeeded) and the actual source tree read at `/home/nil/go/pkg/mod/github.com/coder/acp-go-sdk@v0.20.0/`, plus the GitHub API for org/maintenance signals. Go modules have no install scripts (no postinstall risk surface) and no download-count telemetry; registry existence was proven by the successful module fetch.

| Axis | Finding | Verdict |
|------|---------|---------|
| LICENSE | Apache-2.0, full text in module root [VERIFIED: module `LICENSE` lines 1–5] | PASS — permissive; module-dependency use (not code porting), no attribution obligation beyond license retention |
| Org legitimacy | `coder` — GitHub Organization account (id 95932066), the company behind Coder (well-known remote-dev company); repo not a fork, not archived, not disabled [VERIFIED: api.github.com/repos/coder/acp-go-sdk] | PASS |
| go.mod dependencies | Root module go.mod = module line + `go 1.21` only — **zero require directives, zero transitive deps** (the generator's `cmd/generate` is a separate module; its jennifer dep never reaches consumers) [VERIFIED: module `go.mod`] | PASS — T-4-SC supply-chain exposure is exactly one new dep line |
| Maintenance | created 2025-09-26; pushed 2026-11-19 (daily-active); v0.20.0 released 2026-11-18; 257 stars / 47 forks / 35 open issues; v0.20.0 commit: "v0.20.0 makes every planned API break at once (#117 to #122) … The ACP schema stays at 1.32.0" [VERIFIED: api.github.com commits + repo metadata] | PASS — deliberate stability signal despite v0.x semver |
| API surface for minimal needs | `NewAgentSideConnection(agent Agent, peerInput io.Writer, peerOutput io.Reader)`, 12-method `Agent` interface, `InitializeRequest/Response`, `NewSessionRequest/Response`, `PromptRequest/Response`, `CancelNotification`, `SessionUpdate` + chunk helpers, `AuthMethod{Agent: &AuthMethodAgent{Id, Name, Description}}` — all present and read [VERIFIED: module `agent_gen.go`, `agent.go`, `types_gen.go`, `helpers.go`] | PASS |
| fs/terminal fields we must NOT advertise | **`AgentCapabilities` has no fs or terminal fields at all** — fs/terminal are CLIENT-side capabilities (`ClientCapabilities`); the honest agent surface is structural, not a suppression task [VERIFIED: `types_gen.go:62-98`] | PASS — D-02 satisfied by construction |
| `request_permission` (Phase 5 stub check) | `AgentSideConnection.RequestPermission(ctx, RequestPermissionRequest)` exists for the outbound client RPC — Phase 5 surface present, unused in Phase 4 [VERIFIED: `agent_gen.go:406+` outbound method list] | PASS — no blocker for later phases |
| Schema pin | `const SchemaVersion = "1.32.0"`, `const ProtocolVersionNumber = 1` [VERIFIED: `constants_gen.go:5-9`, quoted verbatim]; **corrects STACK.md's "schema pin 1.25.0"** — README:194's "stable methods of ACP schema 1.25.0" is the stable-interface baseline (a schema release never adds methods to `Agent`/`Client`), while generated types track 1.32.0. Wire `protocolVersion: 1` unchanged — no Orca impact | Recorded (see State of the Art) |
| Go version requirement | SDK requires go ≥ 1.21; repo pins go 1.27 [VERIFIED: module `go.mod`; repo go.mod] | PASS — no conflict |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| ACP v1 JSON-RPC framing over stdio | API/Backend (`internal/acpport` over SDK) | — | SDK owns framing + dispatch; per ARCHITECTURE the SDK never leaks past acpport |
| Capability/authMethods advertisement (CORE-01) | API/Backend (`internal/acpfront`) | acpport translation | Honest capability report is agent policy, not protocol plumbing |
| Session registry (ACP sid ↔ machine) | API/Backend (`internal/core` or acpfront — planner's call) | — | Mutex-protected map per ARCHITECTURE Component Responsibilities; Phase 5 needs it regardless |
| `session/create` + setModel follow-up (CORE-02) | API/Backend (`internal/acpfront` → `internal/zcode`) | — | Direct typed `Conn.Call` on the existing back leg |
| Live delta streaming (CORE-03/04) | API/Backend (tap on `internal/zcode` demux → acpport `SessionUpdate`) | `internal/turn` (completion) | Deltas and completion share the notification demux path — ordering falls out |
| Turn lifecycle + cancel exactly-once (D-05) | API/Backend (`internal/turn` machine — existing) | acpfront turn-context ownership | Phase-3 machinery is the locked cancel primitive |
| Model/thought default selection source | Config (flags/env per ARCHITECTURE config surface) | — | SetModel payload needs a configured selection; production default deferred to later config work |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/coder/acp-go-sdk | v0.20.0 (pin exact) | ACP v1 front leg: JSON-RPC 2.0 framing, typed schema surface (schema 1.32.0, protocolVersion 1), Agent dispatch, cancel-ctx machinery | D-01 audit PASS (above); stdlib-only, zero transitive deps; de facto Go ACP SDK (257★, 18 importers per pkg.go.dev) [VERIFIED: module source + pkg.go.dev + GitHub API] |
| Go toolchain | 1.27.x (local: go1.27.3) | Language/runtime | Already pinned via mise; SDK needs ≥1.21 [VERIFIED: `go version`, module go.mod] |
| internal/zcode, internal/turn, internal/zcode/fake | Phase-3 shipped | Back leg conn, turn machine, scriptable fake + golden corpus | The machinery this phase composes [VERIFIED: repo tree, 03-VERIFICATION.md 43/43] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| acp.NewClientSideConnection (from the same SDK) | v0.20.0 | The test ACP client: full ACP-v1 client speaking to acpfront over io.Pipe in-process | Every CORE-01..05 test; the SDK's client side is the honest counterparty (no hand-rolled client to drift) |
| Hand-rolled ACP-leg transcript recorder | ~40 LOC in tests | Record raw JSON-RPC frames both directions on the ACP pipes for wire-order assertions (chunk-before-stopReason, exactly-once answers) | Extends the Phase-3 golden pattern to the front leg; stdlib-only per house style |
| agentclientprotocol/acp-tck | pinned tag, non-blocking | External conformance harness | Optional; STACK assigns it a separate non-blocking CI job. Deferring wiring to Phase 7 (QA-01) is acceptable — planner decides (Open Question 1) |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| coder/acp-go-sdk | Hand-rolled ACP v1 subset on Phase-3 codec patterns | Pre-approved D-01 fallback — NOT needed; audit passed. Would re-implement a large typed surface (~8 surface points × union types) with drift risk |
| SDK's own `Done()`-only lifecycle | Extra liveness pings | Unnecessary — SDK closes Done() on peer disconnect (connection.go `shutdownReceive`); supervisor stdin-liveness already covers backend health |

**Installation:**
```bash
go get github.com/coder/acp-go-sdk@v0.20.0
# go.mod gains exactly one require line; go.sum one module hash (zero transitive deps — VERIFIED: module go.mod)
```

**Version verification:** performed this session — `go get github.com/coder/acp-go-sdk@v0.20.0` succeeded; module source inspected at `/home/nil/go/pkg/mod/github.com/coder/acp-go-sdk@v0.20.0/`; GitHub metadata confirms v0.20.0 is the latest tag (released 2026-11-18).

## Package Legitimacy Audit

> Required: this phase installs one external package. Full audit in "D-01 Verdict" section above (seam does not support the go ecosystem — manual audit performed, per the phase's research priorities).

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| github.com/coder/acp-go-sdk | Go module proxy | ~14 mo (created 2025-09-26; v0.20.0 2026-11-18) | n/a (Go module proxy has no public download stats) | github.com/coder/acp-go-sdk (org: Coder, Organization) | OK | Approved — SDK-first per D-01 |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
 ACP test client (SDK ClientSideConnection, in-process)          ── CI path
 ACP client (Orca/Zed/any v1 client) over stdio                  ── production path
        │  newline-JSON JSON-RPC 2.0 (SDK framing)
        ▼
┌──────────────────────────────────────────────────────────────┐
│ cmd/zcode-acp                                                │
│   acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)     │
│        │ implements acp.Agent (12 methods)                   │
│   ┌────▼──────────────────────────────┐                      │
│   │ internal/acpport  (ONLY SDK importer)                    │
│   │  agentAdapter: acp.Agent → front.Handlers (neutral types)│
│   │  updateSink:   front → conn.SessionUpdate                │
│   └────┬──────────────────────────────┘                      │
│   ┌────▼──────────────────────────────┐                      │
│   │ internal/acpfront                                       │
│   │  Initialize → honest capability report (D-02)           │
│   │  NewSession → session/create + session/subscribe        │
│   │              + follow-up session/setModel (D-03)        │
│   │  Prompt → content join → machine.Run on OWN ctx         │
│   │  Cancel  → machine.Cancel (exactly-once, D-05)          │
│   │  deltaTap: session/event part.delta → SessionUpdate     │
│   └────┬──────────────────────────────┘                      │
│   ┌────▼───────────────────────────────────────────┐         │
│   │ internal/turn Machine (Phase 3, unchanged)     │         │
│   │  Run/Wait/Cancel exactly-once; -32010 guard    │         │
│   └────┬───────────────────────────────────────────┘         │
│   ┌────▼───────────────────────────────────────────┐         │
│   │ internal/zcode.Conn + NEW ordered observer     │         │
│   │ fan-out (multi-session fix, see Pattern 4)     │         │
│   └────┬───────────────────────────────────────────┘         │
│   internal/supervisor (unchanged)                              │
│   logs: slog → stderr ONLY (stdout is ACP protocol)            │
└──────────┬────────────────────────────────────────────────────┘
           │ JSONL (no jsonrpc key) — Phase 3 codec
           ▼
   fake app-server (CI) / real app-server child
     session/event pushes REQUIRE session/subscribe
     (record.deliveryKind guard — source-verified)
```

Primary use case trace: client sends `initialize` → honest response; `session/new{cwd}` → create+subscribe+setModel on the back leg → `{sessionId}`; `session/prompt` → `session/send` ack → `part.delta` notifications fan out → tap → `session/update` chunks on the ACP wire → completion signal → machine resolves → `PromptResponse{stopReason}` written after all chunk frames (FIFO).

### Recommended Project Structure
```
internal/
├── acpport/          # NEW — every SDK import lives here (D-01 seam lock)
│   ├── agent.go      # acp.Agent adapter → front.Handlers; unadvertised stable methods → -32601 defaults
│   ├── sink.go       # SessionUpdate/RequestPermission send surface (RequestPermission stubbed for Phase 5)
│   └── types.go      # neutral DTOs: InitReport, NewSessionParams/Result, PromptParams, StopReason mapping
├── acpfront/         # NEW — agent policy: initialize report, session/new contract, prompt/cancel flows
│   ├── front.go      # Handlers impl + session registry (or internal/core if planner prefers ARCHITECTURE verbatim)
│   ├── stream.go     # delta tap: part.delta text/reasoning → sink, receive order
│   └── turnctx.go    # per-turn context ownership (cancel → stopReason:cancelled, never ctx error)
├── core/             # optional home for SessionRegistry + TurnTranslator per ARCHITECTURE (planner's call)
├── zcode/            # Phase 3 codec + NEW: ordered notification observer fan-out (Pattern 4)
└── turn/             # Phase 3 machine (unchanged logic; construction adapts to the fan-out)
```

### Pattern 1: The acpport seam (D-01 lock — SDK types never leave one package)
**What:** `acpport` implements `acp.Agent` and exposes a narrow interface in OUR vocabulary; `acpfront` implements that interface and never imports the SDK.
**When to use:** always — this is the locked D-01/ARCHITECTURE seam; v0.x breaks stay a one-package migration.
**Example:**
```go
// internal/acpport/types.go — neutral surface (sketch; exact shape is Claude's discretion per CONTEXT)
type Handlers interface {
    Initialize(ctx context.Context) (InitReport, error)          // InitReport: Name, Title, Version, AuthMethods
    NewSession(ctx context.Context, p NewSessionParams) (string, error) // returns sessionId
    Prompt(ctx context.Context, p PromptParams, sink UpdateSink) (StopReason, error)
    Cancel(ctx context.Context, sessionID string) error
}
// agentAdapter.Initialize builds acp.InitializeResponse{ProtocolVersion: acp.ProtocolVersionNumber, ...}
// from InitReport — the only place the SDK capability struct is assembled.
```

### Pattern 2: The delta tap — where CORE-03/04 content comes from
**What:** Backend content deltas travel the `session/event` notification stream, variant `part.delta`, payload `{messageId, partId, field, delta}` with `field ∈ {"text","reasoning","input","output"}` [VERIFIED: zcode-src `packages/shared/src/zcode-protocol/index.ts:1702-1709`, quoted verbatim]:
```ts
export const zcodeMessagePartDeltaEventPayloadSchema = z
  .object({
    messageId: nonEmptyString,
    partId: nonEmptyString,
    field: z.enum(["text", "reasoning", "input", "output"]).optional(),
    delta: z.string(),
  })
  .strict();
```
Part kinds corroborate: `zcodeMessagePartSchema` discriminates `type:"text"` / `type:"reasoning"` parts, both carrying `text` [VERIFIED: zcode-src `zcode-protocol-legacy-types.ts:373-395`].
**Critical wire precondition:** the app-server pushes `session/event` notifications ONLY when the session has an active subscription — `if (!record.deliveryKind) return;` guards the push path, and `session/subscribe` is what sets it (`record.deliveryKind = params.deliveryKind;`) [VERIFIED: zcode-src `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/server-operations.ts:3183` and `:2003`; subscribe ACCEPTED on the shipped 3.18.2 binary per dialect §4, V-EXP EXP-SUBSCRIBE-01]. The front MUST call `session/subscribe {sessionId, deliveryKind:"desktop-continuous"}` after `session/create`.
**Mapping:** `field:"text"` → `conn.SessionUpdate(ctx, SessionNotification{SessionId, Update: acp.UpdateAgentMessageText(delta)})`; `field:"reasoning"` → `UpdateAgentThoughtText(delta)` (executed-probe wire shapes: `{"sessionId":"s1","update":{"content":{"text":"hello","type":"text"},"sessionUpdate":"agent_message_chunk"}}`). `field:"input"/"output"` are tool-call surfaces — Phase 5; ignore silently in Phase 4.
**Server-side batching note:** the app-server merges consecutive streaming deltas into batches (maxChars budget + interval flush; first delta flushes immediately) — the front therefore receives *batched* deltas, not per-token ones; emit each `part.delta` envelope as ONE chunk, in arrival order [VERIFIED: zcode-src server-operations.ts delta-batch machinery ~:3150-3230].
**When to use:** always for this phase; never synthesize a chunk that no backend event produced (D-02/D-04 honesty).

### Pattern 3: Flush-before-stopReason — ordering falls out of single-path demux + FIFO writes
**What:** CORE-03/D-04 require `stopReason` answered only after all pending updates flushed. Three facts make it structural rather than best-effort:
1. `zcode.Conn` delivers notifications synchronously on its read goroutine in wire order (`deliverNotification` inline in `demux` — Phase 3 verified), so the tap sends chunks in receive order.
2. The SDK serializes every outbound frame under one mutex and writes synchronously: `sendMessage` takes `writeMu` and `c.w.Write(b)` before returning [VERIFIED: SDK `connection.go:713-725`]. Wire order = SessionUpdate call order, and the response frame is written only after `Prompt` returns.
3. Completion arrives on the same ordered path: `turn.completed`/`turn.failed` are `session/event` envelopes delivered after the deltas; the quiet-window path fires only after ≥2.5 s of poll-observed inactivity (poll cadence bounds any in-flight delta; a delta would reset poll activity).
**Residual rule for the implementer:** the tap must process an envelope BEFORE the machine classifies it (fan-out order: tap first, then `machine.ObserveEnvelope`) so a terminal event can never resolve the machine ahead of earlier delta envelopes; and `Prompt` must not return until `machine.Wait` has returned (join point already in `Machine.Run`).
**Spec anchor:** "The Agent MAY send `session/update` notifications with content or tool call updates after receiving the `session/cancel` notification, but it MUST ensure that it does so before responding to the `session/prompt` request." [CITED: agentclientprotocol.com/protocol/v1/prompt-turn — cancellation section]

### Pattern 4: Ordered notification fan-out in `internal/zcode` (the one Phase-3 change this phase needs)
**What:** `zcode.Conn.OnNotification` is a single slot; `NewMachine` overwrites it per machine (`conn.OnNotification = m.ObserveEnvelope` — [VERIFIED: `internal/turn/machine.go:192-194`]). A second ACP session would silently steal the first session's completion signals. Replace with an ordered observer list (e.g. `conn.AddObserver(fn) (remove func())`), delivering synchronously in registration order on the existing read goroutine; `NewMachine` registers instead of assigning. Machine safety under fan-out is already proven: `ObserveEnvelope` drops envelopes when not running/cancelling, and terminal classification matches turnId/session ("foreign-turnId never resolves" — Phase 3) [VERIFIED: `internal/turn/machine.go:326-356`].
**Why not a front-level demux:** routing `OnRouterEvent`/deadline events in the front would duplicate machine-internal knowledge; the conn-level list keeps one demux path (D-04 wording) and is a small, test-first change.
**Backpressure note:** the tap's `SessionUpdate` write happens on the read goroutine; a stalled ACP client propagates backpressure end-to-end (backend stdout fills). This is correct drop-free behavior for protocol updates; do not "fix" with buffering (D-04 forbids buffered-then-flushed).

### Pattern 5: Prompt turn-context ownership (CORE-05 — the SDK cancel contract done right)
**What:** On `session/cancel` the SDK (1) cancels the registered prompt's context (`cancelPrompt` → `pc.cancel()`) and (2) invokes `Agent.Cancel` [VERIFIED: SDK `agent.go:61-73`, `agent_gen.go:152-164`, `:268-285` — Prompt runs under `context.WithCancel` + `registerPrompt`]. But `toReqErr` maps `context.Canceled` from a handler to a `-32800 RequestCancelled` ERROR [VERIFIED: SDK `errors.go:79-90`], while ACP requires the prompt answered with `stopReason:"cancelled"` as a SUCCESS result [CITED: prompt-turn cancellation: "MUST respond … with the `cancelled` stop reason"; spec explicitly: catch abort exceptions, return cancelled, never an error].
**Therefore:** run `Machine.Run` on a front-owned turn `context.Context` (cancelled by `Agent.Cancel`, shutdown, or session teardown — NOT by the SDK request ctx). `Prompt` selects on {machine resolution, SDK ctx.Done}: on SDK ctx.Done call `machine.Cancel()` (idempotent — repeats no-op per Phase 3), keep Waiting on the machine-owned ctx for the exactly-once `StopCancelled` resolution, then return `PromptResponse{StopReason: StopReasonCancelled}, nil`. Post-cancel drain updates still send because the tap uses the turn ctx.
**Machine stopReason → ACP mapping:** `StopCompleted → "end_turn"`; `StopCancelled → "cancelled"`; `StopFailed`/`StopError →` JSON-RPC error from `Prompt` (ACP v1 has no "failed" stopReason; "refusal" means agent-policy refusal, not backend failure) — recommended `-32603` with a readable message [ASSUMED A5: mapping choice is planner/user-confirmable].

### Pattern 6: Fake-driven test client over SDK both sides
**What:** in-process composition — `acp.NewClientSideConnection(clientImpl, agentPipeW, agentPipeR)` ↔ recording pipes ↔ `acp.NewAgentSideConnection(agentAdapter, clientPipeW, clientPipeR)`; the back leg rides `fake.Serve(rw, scenario)` exactly like Phase 3 (`fake.Serve(rw io.ReadWriteCloser, sc *Scenario) (Transcript, error)` [VERIFIED: `internal/zcode/fake/fake.go:102`]). A ~40-line recording `ReadWriter` tee captures raw ACP frames both directions for wire-order assertions. Extends the Phase-3 pattern set; zero processes in the default suite.

### Anti-Patterns to Avoid
- **Returning ctx.Canceled from Prompt:** the SDK converts it to a `-32800` error response — violates the spec's cancelled-stopReason rule. Always convert to `PromptResponse{StopReasonCancelled}, nil` (Pattern 5).
- **Assuming events flow without subscribing:** `session/event` pushes are gated on `record.deliveryKind` — no `session/subscribe` means silent dead streaming with a still-working prompt answer. Subscribe at `session/new`, before the first prompt.
- **Letting the SDK request ctx drive `Machine.Run`:** SDK cancel would cancel the watcher mid-turn and `Wait` would surface a ctx error instead of the honest `StopCancelled` resolution. Own the turn ctx (Pattern 5).
- **Overwriting `conn.OnNotification` per machine:** second session steals the slot; first session's turns hang. Fan-out first (Pattern 4).
- **Emitting `part.upserted` as a chunk:** upserts carry cumulative part text; emitting them alongside `part.delta` duplicates content. Phase 4 maps `part.delta` only; document upsert policy for Phase 5 (snapshot reconciliation).
- **Writing anything but ACP frames to stdout:** unchanged project rule (ARCHITECTURE Anti-Pattern 2); the SDK owns our stdout writes — keep slog on stderr.
- **Advertising what doesn't exist:** SDK struct fields make under-advertising automatic (bool omitempty); do not populate `SessionCapabilities`, `PromptCapabilities`, `McpCapabilities` modes, or `LoadSession` in Phase 4 (the executed probe shows the zero-capability marshaling — see Code Examples).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ACP JSON-RPC framing, dispatch, -32601 defaults | Own JSON-RPC front codec | `acp.NewAgentSideConnection` | Framing + dispatcher + cancel-arrival bookkeeping are subtle (arrival counters, notification barriers); SDK tested (see its `connection_notification_barrier_test.go` names) [VERIFIED: module tests] |
| Typed ACP schema surface | Hand-written ACP structs | SDK `types_gen.go` (schema 1.32.0) | Generated from the official schema; hand types drift |
| Test ACP client | Hand-rolled fake ACP client frames | `acp.NewClientSideConnection` + implement `acp.Client` (collect `SessionUpdate`s) | The client side is where spec-compliance bugs hide; SDK client keeps the counterparty honest |
| Cancel/prompt race handling | Own arrival-order bookkeeping | SDK `registerPrompt`/`cancelPrompt` + Phase-3 machine exactly-once | Both layers already solve it; compose, don't duplicate |
| JSONL back leg, turn machine, fake | (unchanged) | Phase-3 packages | Verified 43/43 |

**Key insight:** the only protocol code this phase writes by hand is the *translation* (acpport DTOs) and the *policy* (acpfront handlers) — never framing, never schema types, never the cancel state machines.

## Common Pitfalls

### Pitfall 1: Silent dead streaming (no subscribe)
**What goes wrong:** prompt works, chunks never arrive.
**Why:** `session/event` pushes require `record.deliveryKind` (set only by `session/subscribe`) [VERIFIED: zcode-src server-operations.ts:3183, :2003].
**Avoid:** subscribe with `deliveryKind:"desktop-continuous"` right after `session/create`; test asserts at least one `session/event` reaches the conn during a scripted turn.
**Warning signs:** machine completion still works (poll watcher) while the ACP wire shows zero updates.

### Pitfall 2: -32800 instead of cancelled
**What goes wrong:** client sees a JSON-RPC error on cancel; spec violation.
**Why:** SDK `toReqErr` maps `context.Canceled` → `NewRequestCancelled` (-32800) [VERIFIED: errors.go:86-88].
**Avoid:** Pattern 5 turn-ctx ownership; the Prompt handler returns `StopReasonCancelled` as a normal result.
**Warning signs:** cancel test sees `{"error":{"code":-32800,...}}` instead of `{"stopReason":"cancelled"}`.

### Pitfall 3: OnNotification slot theft across sessions
**What goes wrong:** second session's turns never complete; first session randomly works.
**Why:** `NewMachine` assigns the single hook [VERIFIED: machine.go:192-194].
**Avoid:** conn-level ordered observer fan-out (Pattern 4) with a registration-order test and a two-session interleaved test.
**Warning signs:** tests passing with one session, hanging with two.

### Pitfall 4: setModel refusal — "Reasoning level is required"
**What goes wrong:** follow-up setModel fails `-32603` for thinking models.
**Why:** `session/create` serializes the selection to `providerId/modelId` (drops `options.reasoningLevel`); setModel without the level is refused [VERIFIED: dialect §2 quirk, V-EXP EXP-TURN-01 refusal observed live].
**Avoid:** D-03's two-step contract; the fake transcript asserts the setModel payload carries `model.options.reasoningLevel`.
**Warning signs:** golden transcript shows create carrying a model object.

### Pitfall 5: Duplicated content from upserts + deltas
**What goes wrong:** client sees doubled text.
**Why:** `part.upserted` carries cumulative part text; deltas are incremental.
**Avoid:** map `part.delta` only in Phase 4; never emit upserts as chunks.
**Warning signs:** scenario with both upsert+delta rows shows repeated text at the client.

### Pitfall 6: Unadvertised stable methods still dispatch
**What goes wrong:** a client calling e.g. `session/set_mode` gets a confusing result or panic instead of a clean error.
**Why:** the SDK's `Agent` interface has 12 REQUIRED methods ( Authenticate, Initialize, Logout, Cancel, CloseSession, DeleteSession, ListSessions, NewSession, Prompt, ResumeSession, SetSessionConfigOption, SetSessionMode) [VERIFIED: types_gen.go:15349+]; capability omission stops legitimate clients but the dispatcher still routes to our impls.
**Avoid:** acpport defaults every unadvertised method to `acp.NewMethodNotFound` (-32601) — matching the SDK's own optional-interface behavior; Authenticate is the deliberate exception (Open Question 2).

### Pitfall 7: Frame-size mismatch between legs
**What goes wrong:** large update payloads truncate or fail on one leg only.
**Why:** the SDK's inbound scanner caps at 10 MiB (1 MiB initial) [VERIFIED: connection.go:422-430]; the backend codec uses its own raised buffer (Phase 3). Our outbound ACP frames are small in Phase 4 (text chunks), so no action — but keep the size test honest at both legs.
**Avoid:** assert the Phase-3 128 KiB raised-line behavior still holds; don't assume shared limits.

## Code Examples

### initialize response — exact honest wire shape (executed probe against v0.20.0)
```go
// Executed probe output (go run against acp-go-sdk v0.20.0, 2026-11-20):
// {"agentCapabilities":{"auth":{},"mcpCapabilities":{},"promptCapabilities":{},
//  "sessionCapabilities":{}},
//  "agentInfo":{"name":"zcode-acp","title":"zcode-acp agent","version":"0.0.0-dev"},
//  "authMethods":[{"id":"zai-api-key","name":"z.ai API key"},
//                 {"id":"zai-individual-coding-plan","name":"z.ai Individual Coding Plan"}],
//  "protocolVersion":1}
resp := acp.InitializeResponse{
    ProtocolVersion: acp.ProtocolVersionNumber, // = 1 [VERIFIED: constants_gen.go:6]
    AgentInfo:       &acp.Implementation{Name: "zcode-acp", Title: &title, Version: version},
    AuthMethods: []acp.AuthMethod{
        {Agent: &acp.AuthMethodAgent{Id: "zai-api-key", Name: "z.ai API key"}},
        {Agent: &acp.AuthMethodAgent{Id: "zai-individual-coding-plan", Name: "z.ai Individual Coding Plan"}},
    },
    // AgentCapabilities zero value — empty objects advertise nothing (spec: omitted/empty = unsupported)
}
```
Note the agent-variant authMethod marshals WITHOUT a `type` field (agent is the default variant); no fs/terminal/loadSession/prompt fields appear anywhere [VERIFIED: executed marshal probe].

### Chunk updates — exact wire shapes (executed probe)
```go
// {"sessionId":"s1","update":{"content":{"text":"hello","type":"text"},"sessionUpdate":"agent_message_chunk"}}
sink(ctx, acp.SessionNotification{SessionId: sid, Update: acp.UpdateAgentMessageText(delta)})
// {"sessionId":"s1","update":{"content":{"text":"thinking","type":"text"},"sessionUpdate":"agent_thought_chunk"}}
sink(ctx, acp.SessionNotification{SessionId: sid, Update: acp.UpdateAgentThoughtText(delta)})
// {"stopReason":"cancelled"}
return acp.PromptResponse{StopReason: acp.StopReasonCancelled}, nil
```
StopReason constants verbatim [VERIFIED: types_gen.go:12442-12458]: `StopReasonEndTurn = "end_turn"`, `StopReasonMaxTokens = "max_tokens"`, `StopReasonMaxTurnRequests = "max_turn_requests"`, `StopReasonRefusal = "refusal"`, `StopReasonCancelled = "cancelled"`.

### Prompt handler skeleton (turn-ctx ownership)
```go
func (f *Front) Prompt(sdkCtx context.Context, p acpport.PromptParams, sink acpport.UpdateSink) (acpport.StopReason, error) {
    st := f.registry.mustGet(p.SessionId)
    turnCtx, turnCancel := context.WithCancel(f.baseCtx) // front-owned, NOT sdkCtx
    st.setTurn(turnCtx, turnCancel)
    defer turnCancel()

    go func() { // cancel fan-in: either source triggers exactly one machine.Cancel
        select {
        case <-sdkCtx.Done():
            st.machine.Cancel() // idempotent (Phase 3: repeats no-op)
        case <-turnCtx.Done():
        }
    }()

    reason, err := st.machine.Run(turnCtx, joinTextBlocks(p.Prompt)) // session/send {sessionId, content}
    if err != nil {
        if errors.Is(err, turn.ErrTurnActive) {
            return "", acpport.InvalidRequest("a prompt is already running for this session") // D-05 guard surface
        }
        return "", acpport.Internal(err) // backend failure → -32603 (A5)
    }
    switch reason {
    case turn.StopCompleted:
        return acpport.StopEndTurn, nil
    case turn.StopCancelled:
        return acpport.StopCancelled, nil
    default:
        return "", acpport.Internal(fmt.Errorf("turn ended: %s", reason))
    }
}
```

### Backend-leg calls (D-03 contract, all Phase-3 typed surfaces)
```go
// session/create — workspaceKey = cwd (docs/protocol/workspace-key.md); bare (no model — Pitfall 4)
conn.Call(ctx, zcode.MethodSessionCreate, map[string]any{
    "workspace": map[string]any{"workspacePath": cwd, "workspaceKey": cwd},
})
// session/subscribe — REQUIRED for session/event pushes (Pattern 2)
conn.Call(ctx, "session/subscribe", map[string]any{
    "sessionId": sid, "deliveryKind": "desktop-continuous",
})
// session/setModel follow-up — the CORE-02 wire assertion target
conn.Call(ctx, "session/setModel", map[string]any{
    "sessionId": sid,
    "model":     map[string]any{"providerId": prov, "modelId": model,
                                 "options": map[string]any{"reasoningLevel": level}},
})
// session/stop — machine.Cancel already fires this on the bypass lane (Phase 3, unchanged)
```
Note: `zcode.MethodSessionCreate/Send/Stop` constants exist [VERIFIED: `internal/zcode/envelope.go:140-142`]; `session/subscribe` and `session/setModel` have NO constants yet — this phase adds them (`MethodSessionSubscribe`, `MethodSessionSetModel`).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| STACK.md claimed acp-go-sdk pins schema 1.25.0 | v0.20.0 pins schema **1.32.0** (`acp.SchemaVersion`); 1.25.0 is the stable-interface baseline (README:194) | v0.20.0 release (2026-11-18) | No wire impact (protocolVersion 1); planner should correct STACK.md wording when convenient |
| No Go SDK in the ACP org (STACK verified 15 repos) | coder/acp-go-sdk is the community-standard Go SDK (first of 7 Go libs on agentclientprotocol.com/libraries/community) | ongoing | Confirms D-01 default |
| v0.19-style SDK API | v0.20.0 = "all planned API breaks at once" + MIGRATION_v0.20.md | 2026-11-18 | Pin exact version; future bumps are guided migrations |
| Poll-only turn watching (Phase 3) | Phase 4 adds the notification delta tap; polls stay for completion fallback | this phase | Subscribe watcher stays a reserved slot (CONTEXT Deferred) — the tap is NOT the Phase-3 reserved subscribe *watcher*; it rides existing notifications |

**Deprecated/outdated:**
- STACK.md "schema pin 1.25.0" — superseded by the source-verified 1.32.0 (see above).
- Hand-rolled-ACP-fallback framing design — retained only as D-01's documented fallback; not exercised.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The real app-server emits `session/event` `part.delta` (field text/reasoning) mid-turn on 3.18.2 as it does per source schema | Pattern 2 | Phase 4 truth is fake-defined (CI); real-streaming behavior first verified via realdesktop/manual runs — a drift would surface in Phase 5+ live checks, fixable in the tap mapping only |
| A2 | Phase 4 `Authenticate` returns a clear error (auth lands Phases 5–6) while authMethods are advertised per D-02 | Open Question 2 | A real client invoking authenticate shows an auth failure card; user-visible wording is planner/user-confirmable |
| A3 | Minimal `resource_link` mapping = include URI text in the joined content string (spec baseline REQUIRES resource_link support) | Pattern/Prompt mapping | A client sending resource_link-only prompts gets slightly different content than a richer mapping would produce; purely additive later |
| A4 | A `session/subscribe` binding persists for the session's lifetime (no TTL in source; `record.deliveryKind` set at subscribe, read at every event push) | Pattern 2 | If the server ever drops the binding, streaming would silently stop — the Phase 5+ live checks would catch it; re-subscribe-on-silence is a cheap future hardening |
| A5 | Backend `StopFailed`/`StopError` map to a JSON-RPC error response from Prompt (-32603), since ACP v1 has no failure stopReason | Pattern 5 | Client UX shows a turn error instead of a synthetic stopReason — this is the honest mapping; confirm in discuss if user-visible wording matters |
| A6 | `session/subscribe` before the first prompt guarantees complete delta capture from turn start (no snapshot reconciliation needed in Phase 4) | Pattern 2 | If a client could prompt before our subscribe landed, early deltas could be missed — sequence subscribe-then-ready inside `session/new` handling to close the window |
| A7 | Empty sub-capability objects (`"mcpCapabilities":{}` etc.) in the marshaled response advertise nothing (spec: all-null/empty = no modes) | Code Examples (probe output) | Spec text supports it for elicitation explicitly; for mcp/prompt caps, `{}` = all flags false = baseline-only. Cosmetic either way; the honesty test asserts on the ABSENT features (no loadSession/fs/terminal), not on empty-object cosmetics |

## Open Questions

1. **Wire acp-tck in Phase 4 or defer to Phase 7 (QA-01)?**
   - What we know: STACK assigns acp-tck a separate non-blocking CI job; CONTEXT defers nothing about it explicitly; QA-01 (Phase 7) owns the full conformance suite.
   - What's unclear: whether the marginal value of an external harness justifies CI wiring now.
   - Recommendation: defer wiring to Phase 7; optionally run it manually once at end of Phase 4 as a smoke signal (it spawns the agent over stdio — cheap).
2. **Phase 4 behavior of `authenticate` (advertised but not functional).**
   - What we know: D-02 locks ADVERTISING both authMethods now; auth flows land Phase 5–6; spec: clients call `authenticate{methodId}` after the user picks a method.
   - What's unclear: the exact error shape a real client sees in Phase 4 (recommended: `-32603` "authentication is not available in this build yet" — honest, non-crashing).
   - Recommendation: confirm the wording in discuss-phase; no spec constraint violated either way.
3. **`session/setModel` `persistAsWorkspaceLastUsed` for product sessions.**
   - What we know: server schema default is `true` (persists to workspace last-used); experiments sent `false` to avoid mutating account config [VERIFIED: dialect §2]; success criterion 2 only requires the setModel payload on the wire.
   - What's unclear: whether honoring a configured model should mutate the user's workspace default (desktop-parity says the desktop host persists user picks; a wrapper honoring `--model` arguably shouldn't).
   - Recommendation: send `persistAsWorkspaceLastUsed:false` explicitly in Phase 4 (predictable, experiment-proven shape); revisit in Phase 6 with the model picker.
4. **SetModel only-when-configured branch.**
   - What we know: success criterion 2 requires the setModel visible on the wire; production default model arrives via config (ARCHITECTURE `--model`/env).
   - What's unclear: behavior when no model is configured (skip setModel and use the app-server personal-config default — but then the criterion's fake test must configure one, which it can).
   - Recommendation: implement "setModel iff a model selection is configured"; the fake test always configures one so the criterion holds.
5. **Session registry home: `internal/core` (ARCHITECTURE verbatim) vs inside `acpfront` for Phase 4 minimalism.**
   - What we know: ARCHITECTURE places SessionRegistry + TurnTranslator in `internal/core`; Phase 5 will need both regardless.
   - Recommendation: create `internal/core` now with the registry only (translator logic can start in acpfront and move when it grows) — planner's wave decomposition call (explicit discretion per CONTEXT).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | all code | ✓ | go1.27.3 (SDK needs ≥1.21) | — |
| mise-managed golangci-lint | CI gate | ✓ (Phase 2 wired, exact pin 2.15.0) | 2.15.0 | — |
| Module proxy access | `go get acp-go-sdk@v0.20.0` | ✓ (fetch succeeded this session) | — | GOFLAGS=-mod=mod verified working |
| Fake app-server binary | CORE-01..05 tests | ✓ (Phase 3: cmd/fake-app-server + in-process fake.Serve) | — | — |
| Real ZCode desktop | realdesktop smoke (optional, manual) | ✓ on this machine (Phase 3 TestRealDesktop passed) | 3.18.2 | not required for the phase gate |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (house style), `-race` mandatory |
| Config file | none — `go test` + mise tasks (existing) |
| Quick run command | `go test -race -count=1 ./internal/acpport/... ./internal/acpfront/... ./internal/zcode/... ./internal/turn/...` |
| Full suite command | `mise run ci` (build → vet → lint → test -race → test-experiments, Phase-2 canonical order) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CORE-01 | initialize returns protocolVersion 1 + honest caps + agentInfo + both authMethods; ACP-leg transcript asserts exact response frame (probe JSON as golden reference); zero fs/terminal/loadSession fields | integration (SDK client ↔ acpfront in-process over pipes, recorded transcript) | `go test -race -run TestInitializeHonestCapabilities ./internal/acpfront/` | ❌ Wave 0 |
| CORE-02 | session/new → create + subscribe + setModel follow-up with `model.options.reasoningLevel` visible on the BACK-leg transcript; mcpServers accepted (non-empty list in request, no error, ignored) | integration + golden (new fake scenarios: `setmodel-followup`, `mcp-servers-ignored`) | `go test -race -run TestGolden ./internal/zcode/fake/ && go test -race -run TestNewSessionCreatesAndSetsModel ./internal/acpfront/` | ❌ Wave 0 (new scenarios + goldens) |
| CORE-03 | deltas streamed live (client receives chunks WHILE turn runs — assert chunk frames precede the prompt response frame in the recorded ACP transcript) and stopReason only after flush (zero updates after the response frame; ordering proof across terminal-event AND quiet-window completion variants) | integration (proof test) | `go test -race -run TestStreamBeforeStopReason ./internal/acpfront/` | ❌ Wave 0 |
| CORE-04 | thought chunks ONLY from reasoning deltas: scenario A (reasoning deltas present → agent_thought_chunk frames in order); scenario B negative (zero reasoning deltas → zero thought chunks, message chunks intact) | integration + golden | `go test -race -run TestThoughtChunks ./internal/acpfront/` | ❌ Wave 0 |
| CORE-05 | cancel mid-turn: exactly one prompt answer, `{"stopReason":"cancelled"}` (never -32800); cancel-after-complete no-op; second concurrent prompt rejected (machine ErrTurnActive → clean JSON-RPC error); composed through supervisor+fake binary exec chain for the cancel leg | integration + exec-chain | `go test -race -run TestCancelMidTurn ./internal/acpfront/ && go test -race -run TestEndToEndExecChain ./internal/supervisor/` | ❌ Wave 0 (front tier; exec-chain exists) |
| (infra) | conn observer fan-out: two sessions interleaved, both machines resolve; registration-order delivery (tap before machine) | unit (internal/zcode) | `go test -race -run TestObserverFanOut ./internal/zcode/` | ❌ Wave 0 |
| (infra) | SDK stays behind the seam: no `coder/acp-go-sdk` import outside internal/acpport | lint-level (grep/test) | `go test -run TestSDKImportBoundary ./internal/acpport/` (or add to CI script) | ❌ Wave 0 |

**Fake corpus additions** (extend Phase-3 scenario schema, fail-loud loader unchanged): `setmodel-followup` (create + reverse runtime-preferences + setModel rule with reply; golden asserts the setModel request payload verbatim), `part-delta-streaming` (session/event `part.delta` rows: text, reasoning, interleaved, then `turn.completed`), `no-reasoning` (text deltas only), plus reuse of `cancel-mid-turn` through the front tier. Back-leg goldens stay byte-deterministic with the fixed-id corpus convention.

**Real-client spot check (manual, non-gating):** spawn `cmd/zcode-acp` (with the fake binary wired via the supervisor Override seam) under a real ACP client once at phase end — precedent: Phase 3's realdesktop human-check pattern. Not part of `mise run ci`.

### Sampling Rate
- **Per task commit:** quick run command (above) — the four packages touched by this phase.
- **Per wave merge:** `mise run ci` full gate (Phase-2 canonical order; zero new lint debt).
- **Phase gate:** full suite green before `/gsd:verify-work`; exec-chain E2E green (happy + cancel) included by default suite.

### Wave 0 Gaps
- [ ] `internal/zcode` observer fan-out (conn change) + its ordering/interleave tests — everything else composes on it
- [ ] `internal/acpport/` package skeleton + SDK import boundary test
- [ ] `internal/acpfront/` package skeleton + test scaffolding (recording pipe helper)
- [ ] New fake scenarios + goldens: `setmodel-followup`, `part-delta-streaming`, `no-reasoning`, `mcp-servers-ignored`
- [ ] `zcode.MethodSessionSubscribe` / `MethodSessionSetModel` constants (envelope.go method table)
- Framework install: none needed (stdlib)

## Security Domain

> security_enforcement enabled (ASVS Level 1 per .planning/config.json).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | partial (advertisement only) | authMethods advertised per D-02; no credential handling in Phase 4 (Phases 5–6); `authenticate` answered with an error, never a fake success (honesty rule) |
| V3 Session Management | yes | SDK arrival-order cancel/prompt bookkeeping + machine exactly-once; session ids are app-server-generated opaque strings (1:1, no alias table) |
| V4 Access Control | no (N/A this phase) | no authorization surface exists yet |
| V5 Input Validation | yes | SDK typed decode + `p.Validate()` on all requests; `cwd` MUST-be-absolute checked before backend create (spec requirement); unknown content-block types rejected as invalid params (promptCapabilities advertise text baseline only) |
| V6 Cryptography | no (N/A this phase) | no secrets handled until Phase 5/6 (CredentialVault/PAT) |
| V7 Error handling/logging | yes | logs → stderr slog only; stdout protocol-exclusive; error responses carry readable messages, no internal paths (machine-path rule from project conventions) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Malformed ACP frame desyncing the client | Tampering | SDK skips undecodable lines with a log and continues [VERIFIED: connection.go:439-442]; never echo raw input to stdout |
| Notification flood exhausting memory | DoS | SDK inbound notification queue bounded (1024); overflow shuts the connection down loudly [VERIFIED: connection.go:19, 164, 516-533] — do not raise silently |
| Secrets leaking into ACP frames/logs | Info disclosure | Phase 4 surface carries no secrets; agentInfo = name/title/version only; keep the Phase-2 redaction/secret-scan gates green |
| Unvalidated cwd reaching the backend | Tampering | validate absolute path (spec MUST) before `session/create`; fake scenarios include a relative-cwd negative row |
| Oversized prompt content | DoS | SDK scanner 10 MiB cap [VERIFIED: connection.go:423-430]; machine sends `content` string — Phase 4 documents the cap rather than lifting it |

## Sources

### Primary (HIGH confidence)
- Module source read: `/home/nil/go/pkg/mod/github.com/coder/acp-go-sdk@v0.20.0/` — `constants_gen.go:5-9` (ProtocolVersionNumber=1, SchemaVersion="1.32.0", verbatim), `types_gen.go` (InitializeRequest/Response :5272-5389 incl. AgentInfo :5341 + AuthMethods :5345; AgentCapabilities :62-98 no fs/terminal fields; AuthMethod/AuthMethodAgent :434-536; NewSessionRequest/Response :6600-6725; PromptRequest/Response :7432-7526; CancelNotification :920-947; Implementation :5074-5092; Agent interface :15349+; StopReason consts :12442-12458), `agent.go` (registerPrompt/cancelPrompt :48-119), `agent_gen.go` (dispatch :152-164 cancel, :268-285 prompt, SessionUpdate outbound :406-408), `connection.go` (receive loop :422-545, sendMessage :713-725, scanner caps :422-430, queue cap :19/:164), `errors.go` (toReqErr :79-90), `helpers.go` (TextBlock :6-11, Update helpers :88-116), go.mod (zero requires), LICENSE (Apache-2.0) — all read with the Read tool this session
- Executed marshal probe (`go run`, 2026-11-20): exact JSON for InitializeResponse (zero caps + agentInfo + both authMethods), both chunk updates, PromptResponse cancelled, authMethods array — quoted verbatim in Code Examples
- Local ZCode source tree (github.com/zai-org/ZCode; local clone at zcode-src): `packages/shared/src/zcode-protocol/index.ts:1702-1709` (part.delta payload verbatim), `:1863-1887` (session/event discriminated union incl. part.started/delta/upserted), `zcode-protocol-legacy-types.ts:373-395` (text/reasoning part types), `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/server-operations.ts` (:3183 deliveryKind push guard, :2003/:1932 subscribe sets deliveryKind, :3150-3230 live delta batching, `sendProtocolSessionEvent` :3245-3257), `:1898` session/messages result schema — read this session
- In-repo docs (HIGH, provenance-tagged): `docs/protocol/app-server-dialect.md` §1-§7 (transport, method map, subscribe verdicts V-EXP, setModel trap, error codes); `internal/turn/machine.go` + `watcher.go` + `internal/zcode/{envelope,conn,call}.go` + `internal/zcode/fake/{fake,golden}.go` read this session; `03-VERIFICATION.md` (43/43)
- GitHub API: api.github.com/repos/coder/acp-go-sdk (org, dates, stars, license spdx_id, archived=false) + /commits (v0.20.0 release notes verbatim)

### Secondary (MEDIUM confidence)
- agentclientprotocol.com v1 spec pages (official docs, fetched 2026-11-20): `/protocol/v1/initialization` (negotiation, omitted-caps MUST-UNSUPPORTED, baseline session methods, Implementation fields), `/protocol/v1/session-setup.md` (cwd MUST absolute, auth_required), `/protocol/v1/prompt-turn.md` (stopReason MUST + enum semantics, sessionUpdate discriminator, cancellation ordering rule verbatim), `/protocol/v1/cancellation.md` ($/cancel_request machinery, -32800, permission cancelled outcome)

### Tertiary (LOW confidence)
- None — every load-bearing claim is grounded in Primary sources; A1's live-wire half is honestly unverified (no completed real turn has ever been captured — dialect §5 D-05 degradation record) and is carried as Assumption A1, not as a finding.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — module source + executed probe + GitHub API, three independent looks at v0.20.0
- Architecture: HIGH for the seam/streaming/cancel patterns (SDK + zcode-src + in-repo reads agree); MEDIUM only where the real wire is unexercised (A1)
- Pitfalls: HIGH — each pitfall cites a verified mechanism (source line or executed behavior)

**Research date:** 2026-11-20
**Valid until:** 2026-12-20 (SDK is mid-v0.x but pinned; schema line moves additively — re-verify `SchemaVersion` on any SDK bump)

## RESEARCH COMPLETE
