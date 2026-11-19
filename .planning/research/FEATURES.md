# Feature Research

**Domain:** ACP v1 agent-server adapter (Go, stdio) wrapping ZCode Desktop's proprietary app-server stdio protocol
**Researched:** 2026-11-19
**Confidence:** HIGH — protocol surface read from the official spec (agentclientprotocol.com v1 docs) the same day; ZCode-side facts from the public source tree (`ZCode source tree (github.com/zai-org/ZCode; local clone)`, wire-method table in `packages/shared/src/zcode-protocol/index.ts`) plus experimentally verified FINDINGS.md. Client-behavior claims beyond official docs are labelled MEDIUM.

## Feature Landscape

### Table Stakes (Clients Break Without These)

The ACP v1 spec makes a small core mandatory: every agent MUST support `initialize`, `session/new`, `session/prompt`, `session/cancel`, and `session/update` (spec, HIGH). Everything else is capability-gated. For a chat-UI client like Orca (local FINDINGS, HIGH: fs=false, terminal=false in `clientCapabilities`) and for Zed, this core plus tool/permission streaming is what makes a card usable.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| `initialize` + protocolVersion negotiation + `agentInfo{name,title,version}` | First call every client makes; wrong `protocolVersion` reply = client closes connection. Zed/Orca both gate everything on it | LOW | Always reply `protocolVersion: 1`. Echo back only capabilities actually implemented — clients treat omitted = unsupported. Include real `agentInfo` (name `zcode-acp`, version from build) |
| `authMethods[]` in initialize response + `authenticate{methodId}` | ZCode always needs a credential; Zed renders auth methods as a picker, Orca chat cards expect `auth_required` handling. Advertising zero methods = unusable for new users | HIGH | Two methods: `zai-api-key` (api-key) and `zai-individual-coding-plan` (OAuth). OAuth path = registry snapshot via `provider/updateAccountConfig` (revision string `zcode-builtin:<rev>:<sha256 of builtin path>` — FINDINGS), answer `interaction/requestCredentialCipher`, mint PAT via api.z.ai, answer `interaction/requestProviderRuntimeHeaders` with `{headersApplied:true, requestAuth:{apiKey,apiKeyId}}` |
| ZCode desktop discovery + app-server spawn (`ELECTRON_RUN_AS_NODE=1 <app>/resources/glm/zcode.cjs app-server --stdio`) | Without the backend process nothing works; this is the adapter's reason to exist | MEDIUM | Detection mechanism (AppImage/deb/unpacked) is an open question in PROJECT.md. Fail loudly with an actionable error when not found |
| JSONL ↔ JSON-RPC 2.0 transport translation | app-server speaks JSONL-with-`method` fields, NOT JSON-RPC (key `jsonrpc` is rejected — FINDINGS). The whole adapter is this dialect bridge | MEDIUM (foundational) | Bidirectional: client requests → app-server calls; `state.updated` events + `interaction/*` requests ← app-server. Multiplex per ACP sessionId ↔ app-server session |
| `session/new{cwd, mcpServers}` → `sessionId` | Spec baseline. Zed passes project cwd; Orca passes the card workspace | MEDIUM | Map to `session/create`. CRITICAL: `session/create` drops `options.reasoningLevel` for thinking models — must follow up with `session/setModel{providerId, modelId, options:{reasoningLevel}}` (FINDINGS, HIGH). Accept and ignore `mcpServers` unless implementing MCP pass-through |
| `session/prompt{sessionId, prompt[]}` with `stopReason` reply | Spec baseline; the request/response every chat turn hangs on. Must reply `end_turn`/`cancelled`/etc. only after all pending updates are flushed | MEDIUM | Map to `session/send`. Support `ContentBlock::Text` (MUST) and `ResourceLink` (MUST per promptCapabilities baseline); do NOT advertise image/audio support in v1 |
| Stream `session/update` `agent_message_chunk` | Streaming text is the entire perceived quality of a chat card; batched text feels broken | MEDIUM | Source: `state.updated` model_stream events. Chunk on deltas as they arrive; do not buffer whole messages |
| Stream `agent_thought_chunk` | Thinking models (GLM-5.x) emit reasoning; Zed and Orca render it in a collapsible section. Most adapters drop it — clients at minimum tolerate it | LOW-MEDIUM | Data already present in app-server reasoning events. Emit only when backend provides it |
| Stream `tool_call` / `tool_call_update` (title, kind, status pending→in_progress→completed/failed, diff content) | PROJECT.md core value: "карточка в chat UI без инструментов не имеет смысла". Zed renders diffs from `content[{type:"diff",path,oldText,newText}]` | HIGH | Map app-server tool lifecycle events (`tool-scheduled`, `tool-started`, progress kinds `scheduled|started|progress|result|error`, local source, HIGH) onto `tool_call` then `tool_call_update` deltas. Set `kind` from tool name (read/edit/execute/search/think/...). Emit file diffs for edit tools — this drives Zed's edit-review UI |
| `session/request_permission` bridging (options `allow_once/allow_always/reject_once/reject_always`, outcome `selected{optionId}` / `cancelled`) | Tool approval is mandatory UX for coding agents; blocking the turn with no dialog = hang. Orca and Zed both implement the client side | MEDIUM | Map app-server `interaction/requestPermission` (host MUST answer in host-managed mode — FINDINGS). Timeout policy needed: app-server interactions may have their own deadlines; respond `cancelled` on ACP-side cancel |
| `session/cancel` notification → stop turn, reply `stopReason:"cancelled"` to the in-flight `session/prompt` | Spec baseline. Users hit Stop constantly; leaking the turn = orphaned backend work + confused client | MEDIUM | Map to the app-server stop path (`session/stop` deprecated → v4 stop command / `session/cancelBackgroundTask`, local source, HIGH). Pending permission requests must resolve as `cancelled`. Flush remaining updates BEFORE replying to `session/prompt` (spec) |
| Host-managed interaction/* responder (always-on duty) | app-server runs host-managed; unanswered `interaction/requestProviderRuntimeHeaders` → `headersApplied:false` = guaranteed turn failure (FINDINGS) | MEDIUM (cross-cutting) | Not a client-visible feature but a correctness precondition for every table-stakes flow. `provider/updateAccountConfig` acks `received` even when the snapshot is rejected — verify via registry/`state.updated`, not the ack |
| `session/close` graceful shutdown of the app-server child on process exit | Zed kills agents between sessions; a leaked Electron-as-Node process per session is a real bug users will file | LOW | Map to app-server `session/close` where used; kill process group on ACP stdio EOF |

Spec-optional (capability-gated) methods a minimal client never calls: `session/load`, `session/resume`, `session/list`, `session/delete`, `session/fork`, modes, config options, slash commands, `fs/*`, `terminal/*`, `plan` updates, MCP http/sse. All are legitimately omittable in the initialize response — omission is the correct v1 behavior, not a gap.

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| OAuth individual coding plan end-to-end | No public ACP adapter drives z.ai's coding-plan OAuth (registry snapshot + PAT mint + signed requests by the official binary). This is the project's moat — the shim already proved both paths (FINDINGS: OAUTH_SHIM_OK / OAUTH_TOOL_OK) | HIGH | Full chain: snapshot with exact revision string, `interaction/requestCredentialCipher` (Secret Service key decryption on Ubuntu GNOME), getCustomerInfo → api_keys → access_tokens PAT mint, runtime-headers response. Only official paths |
| Model + reasoning-level picker via ACP config options (`category:"model"`, `category:"thought_level"`) | Zed shows native model dropdowns; chat UIs get a selector instead of env-var config. Directly resolves a PROJECT.md open question (map ACP model choice → `session/setModel` + reasoningLevel) | MEDIUM | Stabilized v1 surface: return `configOptions[]` in `session/new` response (built from the app-server registry after auth), handle `session/set_config_option{configId,value}` → `session/setModel`; changing model MUST re-emit full `configOptions` (reasoning options cascade). Keep `currentValue` honest — graceful degradation is spec-mandated |
| Full thought-stream fidelity | Differentiates from claude-code-acp-style adapters that fold reasoning into message text | LOW-MEDIUM | Only emit when the backend emits; never synthesize |
| `session/load` (replay) + `session/list` → thread history | Zed imports existing threads; users resume yesterday's card after restart. app-server has native `session/list` + `session/messages` (local source, HIGH) — replay is mostly re-emission | MEDIUM | Advertise `loadSession:true` only when replay is faithful (user chunks, agent chunks, tool calls in order); respond to `session/load` only after replay completes (spec) |
| `session/resume` (no replay) | Cheaper reconnect for chat UIs; native `session/resume` exists app-server-side | LOW | Gate behind `sessionCapabilities.resume:{}` |
| Protocol-conformance test suite with a mock app-server | PROJECT.md decision: CI protocol tests = done-v1 criterion. Also differentiates from every wrapper shipped with zero protocol tests | MEDIUM | Golden-file JSONL fixtures + an in-process fake app-server asserting ACP framing; run adapters-under-test from Zed's and Orca's client harnesses if feasible |
| Orca launch recipe (README, not architecture) | PROJECT.md decision: recipe, not architectural coupling. Still a differentiator for the primary audience | LOW | Document binary naming/version floor (`omp` ≥17.0.5 detect, `PI_NO_PTY=1` env — FINDINGS) as a user-side PATH shim recipe |
| Honest-provenance documentation (no patching, no attestation forgery, PAT signed by official binary) | Trust story no quota-forging wrapper can claim; PROJECT.md Out-of-Scope explicitly protects this | LOW | README section; also drives anti-features below |

### Anti-Features (Deliberately NOT Building)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| `fs/read_text_file` / `fs/write_text_file` agent-side support ("full ACP coverage") | Looks like completeness | ZCode's agent reads/writes the real filesystem itself; chat-UI clients advertise fs=false, and calling unadvertised fs methods violates the spec (agent MUST NOT). Editor-buffer access is a Zed-specific concern ZCode cannot honor anyway | Advertise nothing fs-related; never call `fs/*`. Costs zero |
| `terminal/*` methods (create/output/release) | "Zed has terminal support for agents" | Same class: ZCode executes its own bash tools; routing through a client terminal duplicates execution and breaks tool results. Chat-UI clients advertise terminal=false | Never advertise, never call |
| Re-implementing the agent (own tool loop, own model HTTP calls, own prompt templates) | Tempting because the shim (zcode-claude-shim) already talks to api.z.ai directly | Two agent implementations drift forever; loses desktop-app parity (provider registry, MCP, skills, updates); kills the OAuth-entitlement path that depends on the official runtime | The adapter is a pure protocol bridge: every model/tool decision stays inside app-server |
| Patching zcode.cjs / the ZCode bundle (e.g. to fix `reasoningLevel` loss at the source) | One-line fix would remove the setModel workaround | Breaks update-proofness (every ZCode release needs a new patch), violates PROJECT.md Out-of-Scope, kills the honesty claim | `session/create` then `session/setModel` — already proven (FINDINGS) |
| Attestation / V4-signature forgery to inflate quota (the "150% quota" temptation from FINDINGS) | More quota for users | Explicitly out of scope in PROJECT.md; ToS violation; fragile (server-side PoW/sig checks); legal exposure | Only official PAT path: z.ai mints the PAT, the official binary signs |
| Fake quota/usage reporting (`sessionInfo_update` with invented numbers) | Users ask "how much quota left?" | The OAuth quota counter is not retrievable (`/api/monitor/usage/quota/limit` 401s on OAuth token — FINDINGS); inventing numbers is a lie with a UI | Report nothing rather than wrong numbers; document the limitation |
| Proxying proprietary app-server methods 1:1 onto the ACP surface (`session/compact`, `session/goal`, `runtime/capabilities`, …) via `_meta` | "Expose everything ZCode can do" | Couples clients to a private dialect that changes between ZCode releases; leaks backend details; maintenance sink | Map only what ACP names; keep any passthrough behind a small, documented, versioned `_meta` extension if ever needed |
| Real `session/delete` implementation | Orca calls `session/delete` on card close (FINDINGS, HIGH) | app-server has NO `session/delete` wire method (verified in source, HIGH) — there is nothing to delete server-side | Advertise nothing (or accept + no-op returning empty result for Orca friendliness); local manifest cleanup only |
| ACP v2 draft support now | claude-agent-acp ships it | Draft is explicitly unstable ("can still change"); v2 methods are RFD-gated; splits test matrix before v1 works | v1 only; revisit when v2 stabilizes |
| Supporting the zcode CLI as a second backend | Smaller dependency for headless servers | PROJECT.md Out-of-Scope for v1; CLI and desktop differ in provider runtime (standalone vs host-managed), doubling the auth matrix | Desktop app-server only; CLI via PRs later |

## ACP ↔ ZCode app-server Capability Map

The load-bearing mapping (spec confidence HIGH; local-source confidence HIGH; flagged otherwise):

| ACP surface | ACP status | ZCode app-server mapping | Conf. |
|-------------|-----------|--------------------------|-------|
| `initialize` | REQUIRED | Spawn child; optionally `runtime/capabilities`; `provider/refreshRegistry` to warm models | HIGH |
| `authenticate{methodId}` | Required-if `authMethods` advertised | api-key: `provider/updateAccountConfig`; OAuth: snapshot + `interaction/requestCredentialCipher` + PAT mint (api.z.ai) + `interaction/requestProviderRuntimeHeaders` → `requestAuth` | HIGH (FINDINGS, experimentally verified) |
| `session/new{cwd}` | REQUIRED | `session/create` + follow-up `session/setModel` (reasoningLevel loss bug) | HIGH (FINDINGS) |
| `session/prompt` | REQUIRED | `session/send` (v4 sendText main path) | HIGH |
| `session/update` message/thought chunks | REQUIRED | `state.updated` model_stream deltas | HIGH |
| `tool_call` / `tool_call_update` | REQUIRED for a coding agent | tool lifecycle events: `tool-scheduled`, `tool-started`, `progress{result|error|batch|raw}` | HIGH (source) |
| `session/request_permission` | REQUIRED for tool UX | `interaction/requestPermission` (host MUST answer) | HIGH (FINDINGS) |
| `session/cancel` | REQUIRED (baseline) | stop path (`session/stop` deprecated → v4 stop / `cancelBackgroundWork`) | HIGH (source) |
| `session/close` | Optional (`sessionCapabilities.close`) | native `session/close` exists | HIGH (source) |
| `session/load` (replay) | Optional (`loadSession`) | `session/messages` → re-emit as `session/update` (incl. `user_message_chunk`) | HIGH (source) |
| `session/resume` (no replay) | Optional (`sessionCapabilities.resume`) | native `session/resume` | HIGH (source) |
| `session/list` | Optional (`sessionCapabilities`/registry) | native `session/list` | HIGH (source) |
| `session/delete` | Optional | **No wire method exists** — no-op or omit | HIGH (verified absence, source) |
| `session/fork` | UNSTABLE in v1 | `session/fork` wire name preserved for v4 op (source); Orca calls it (FINDINGS) | MEDIUM |
| Config options (model/thought_level) | Optional (stabilized surface) | `session/setModel` + `session/setThoughtLevel`; options list from provider registry | HIGH (FINDINGS for setModel) |
| Modes (`session/set_mode`) | Optional, deprecated surface | native `session/setMode` exists (v4 value range excludes `auto`) | HIGH (source) |
| Slash commands (`available_commands_update`) | Optional | No native equivalent — synthesize or omit | LOW (no source study yet) |
| `plan` update | Optional | ZCode has todo/plan notions in events (unmapped — needs phase research) | LOW |
| `fs/read_text_file`, `fs/write_text_file` | Client capability | NEVER call — ZCode touches disk directly | HIGH |
| `terminal/*` | Client capability | NEVER call — ZCode runs its own shell tools | HIGH |
| MCP servers param / `mcpCapabilities.http` | Optional | app-server manages its own MCP (`interaction/requestOfficialMcpAuthHeaders`); ACP `mcpServers`: accept and ignore in v1 | MEDIUM |
| `logout` | Optional (`agentCapabilities.auth.logout{}`) | Feasible (clear provider config + keyring), but only advertise if implemented | MEDIUM |

## Feature Dependencies

```
[ZCode discovery + spawn]
    └──requires──> [initialize] ──requires──> [everything else]

[authenticate / authMethods]
    └──requires──> [host-managed interaction/* responder]
                        └──requires──> [Secret Service decryption (Ubuntu GNOME)]

[session/new]
    └──requires──> [authenticate]          (account models absent without registry snapshot)
    └──enhanced-by──> [session/setModel follow-up]   (thinking models unusable without it)

[session/prompt streaming]
    └──requires──> [JSONL↔JSON-RPC bridge]
    └──requires──> [session/cancel + stopReason:"cancelled"]   (spec pairing)

[tool_call/tool_call_update]
    └──requires──> [session/request_permission bridging]   (approval flow shares the tool lifecycle)

[config options model picker]
    └──requires──> [authenticate] + [session/setModel fix]

[session/load replay]
    └──requires──> [session/list] + faithful event→update mapping
    └──conflicts-with──> [v1 scope pressure]   (replay fidelity is the long pole; defer if thin)

[Orca recipe] ──enhances──> [discovery]   (docs only; no code dependency)
[fs/terminal capabilities] ──conflicts-with──> [honest-bridge architecture]   (must NOT be built)
```

### Dependency Notes

- **Streaming requires the transport bridge:** every feature funnels through the JSONL↔JSON-RPC translator; it must land first and be test-fixture-grade.
- **Auth requires the interaction responder:** both auth paths answer `interaction/*` requests; the responder is a precondition, not a feature.
- **Tool calls require permission bridging:** shipping tool_call streaming without `session/request_permission` produces cards that auto-run tools or hang — they must land together.
- **Config options require the setModel fix:** without the `session/setModel` follow-up the picker is cosmetic for thinking models.
- **`session/delete` conflicts with the backend:** verified absent server-side; implement as no-op/omit, never as an error path clients can crash on.

## MVP Definition

### Launch With (v1 core, PROJECT.md Active list)

- [ ] Discovery + spawn of desktop app-server with loud failure diagnostics
- [ ] `initialize` (v1, minimal capabilities), `session/new` (+ setModel fix), `session/prompt` with `agent_message_chunk` + `agent_thought_chunk` streaming, `session/cancel`, correct `stopReason` lifecycle
- [ ] `tool_call`/`tool_call_update` with diffs + `session/request_permission` bridging
- [ ] Both auth paths end-to-end: api-key and OAuth coding plan (snapshot → PAT → signed turns)
- [ ] Graceful child lifecycle (stdin EOF → kill process group; one app-server per ACP session or a documented sharing model)
- [ ] Protocol test suite with mock app-server in CI

### Add After Validation (v1.x)

- [ ] Config-options model/reasoning picker (depends on auth + setModel fix being stable)
- [ ] `session/resume` + `session/close` (cheap, native mappings exist)
- [ ] `session/list` + `session/load` replay for Zed thread import (add when replay fidelity proven)
- [ ] Orca compatibility no-ops (`session/delete`, `session/fork` graceful errors) — add the moment the Orca recipe is published
- [ ] `logout` + `sessionInfo_update` basics if demanded by users

### Future Consideration (v2+)

- [ ] Slash commands (`available_commands_update`) — needs a decision on what ZCode commands even surface
- [ ] ACP v2 draft surfaces — blocked on protocol stability
- [ ] Non-Ubuntu platforms (KDE/macOS/Windows keyring paths) — PR-driven per PROJECT.md
- [ ] zcode CLI backend — explicitly out of v1 scope

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| initialize + session/new/prompt/cancel + message streaming | HIGH | MEDIUM | P1 |
| tool_call/update + permission bridging | HIGH | HIGH | P1 |
| Auth: api-key path | HIGH | MEDIUM | P1 |
| Auth: OAuth coding plan | HIGH | HIGH | P1 |
| setModel reasoningLevel fix | HIGH | LOW | P1 |
| Mock app-server conformance tests | MEDIUM (project health) | MEDIUM | P1 |
| Config-options model picker | MEDIUM-HIGH | MEDIUM | P2 |
| session/resume + session/close | MEDIUM | LOW | P2 |
| session/list + session/load replay | MEDIUM | HIGH | P2 |
| Orca recipe + compat no-ops | HIGH (for primary audience) | LOW | P2 |
| Slash commands | LOW-MEDIUM | MEDIUM | P3 |
| session/fork (unstable) | LOW | MEDIUM | P3 |

**Priority key:** P1 = must have for launch; P2 = should have, add when possible; P3 = nice to have.

## Competitor Feature Analysis

| Feature | claude-agent-acp (Zed/Anthropic SDK adapter) | Gemini CLI `--acp` | OpenCode / omp | zcode-acp (ours) |
|---------|---------------------------------------------|--------------------|----------------|------------------|
| Architecture | SDK wrapper around Claude Agent SDK (proprietary backend, like ours) | Native ACP mode inside the CLI | Native ACP in agent | Wrapper around proprietary app-server — same pattern as claude-agent-acp |
| Auth methods | SDK's own auth | `authenticate` + terminal-type interactive login | provider config | Two explicit `authMethods`: api-key + OAuth coding plan |
| Streaming | messages + thoughts + tool calls + diffs | messages, tool calls, permission meta | messages, tool calls | messages + thoughts + tool calls with diffs (target parity) |
| Model selection | session config option (Ultracode etc.) | `unstable_setSessionModel` | modelDiscovery | config options `model` + `thought_level` via setModel+setThoughtLevel |
| Session lifecycle | v1: new/list/resume/close; draft v2: delete/fork | new/load/setSessionMode | new/prompt/cancel (+fork per FINDINGS) | new (+resume/close/list/load planned; delete = no-op; fork = graceful error) |
| Distinctive | v2 draft, terminals, MCP-over-ACP | IDE integration, MCP `_meta` in permissions | hardcoded-client compatibility (Orca) | OAuth coding-plan e2e; honest no-patch provenance; conformance test suite |

Reading: the dominant shape in the ecosystem is exactly ours — a thin ACP skin over a proprietary backend (claude-agent-acp, codex-acp, acp-llm-adapter). None of the documented wrappers implement the OAuth-entitlement chain this project must; that is both the hardest feature and the differentiator. Ecosystem-list confidence HIGH (official registry page); per-implementation behavior MEDIUM except where noted.

## Sources

- ACP v1 spec pages (agentclientprotocol.com, read 2026-11-19, HIGH): initialization, session-setup, prompt-turn, authentication, session-modes, session-config-options, slash-commands, tool-calls, file-system; llms.txt doc index; RFD session-fork
- ACP agents registry: agentclientprotocol.com/get-started/agents (HIGH)
- Zed external-agents docs: zed.dev/docs/ai/external-agents (HIGH for configuration surface)
- gemini-cli ACP docs: github.com/google-gemini/gemini-cli `docs/cli/acp-mode.md` + `packages/cli/src/acp/` (HIGH); gemini-cli issue #13913 (ACP subprocess regression in 0.18+, MEDIUM — reminder that even first-party ACP modes regress)
- claude-agent-acp README: github.com/zed-industries/claude-code-acp (HIGH)
- FINDINGS.md (`FINDINGS.md (private research notes, outside this repo)`): Orca ACP client surface, app-server dialect, OAuth/PAT chain, known protocol bugs (HIGH — experimentally verified 2026-11-18/19)
- ZCode public source `ZCode source tree (github.com/zai-org/ZCode; local clone)` 3.17.x: `packages/shared/src/zcode-protocol/index.ts` wire-method table + server→host event kinds (HIGH, read 2026-11-19)

---
*Feature research for: zcode-acp — ACP v1 agent server wrapping ZCode Desktop app-server*
*Researched: 2026-11-19*
