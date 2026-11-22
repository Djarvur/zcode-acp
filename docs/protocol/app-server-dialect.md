# App-server dialect — ZCode 3.18.2 JSONL event surface

**Domain:** the desktop app-server's wire dialect — transport framing, full method map, server→host surface, `session/subscribe` / event-stream availability, turn-completion terminal signals, error codes, session lifecycle, snapshot semantics, and the 3.17↔3.18.2 drift baseline (DISC-01)
**Verified against:** the shipped 3.18.2 desktop bundle — the unpacked-AppImage squashfs-root copy (`~/Applications/zcode-<ver>/squashfs-root/`, the same copy `install-discovery.md` row 1 names as ground truth; `resources/glm/zcode.cjs` byte-compared identical to its `.orig` sibling while writing this doc, consistent with the recorded pristine check in `[V-EXP EXP-DISCOVERY-01]`) plus the live experiments below — and the public 3.17.x source tree read at **3.17.1**, which is corroborating only and never 3.18.2 truth (per D-01)
**Confidence:** HIGH for everything tagged `[V-EXP <experiment-id>]` (live on the shipped binary) and for bundle-anchored `[V-SRC ...]` claims (grep anchors with occurrence counts, re-derivable via the recorded commands); HIGH for 3.17.1-cited mechanics re-verified file:line in the clone this phase; the turn-COMPLETION half of section 5 is honestly `V-SRC-only` per the recorded D-05 degradation — the turn never completed under the Phase 1 harness (limitation quoted from `[V-EXP EXP-TURN-01]`)

Provenance tags (`V-SRC` / `V-EXP` / `V-RPT`) are used per the legend in
[README.md](README.md) — defined once there (D-04). Every `[V-EXP <id>]` in this
document resolves to a scrubbed capture under `experiments/captures/`. Bundle
anchors cite `resources/glm/zcode.cjs` relative to the bundle root actually
probed; `grep -o <pattern> | wc -l` occurrence counts are recorded so every
bundle-derived claim is re-derivable by hand.

This document is the phase's DISC-01 deliverable. Phase 3 quotes sections 1, 2,
6 (codec frames, error codes); Phase 4 quotes sections 4–5 (turn machine);
Phase 5 quotes section 3 (interaction shapes); Phase 6 quotes section 8
(snapshot semantics, revision formula).

## 1. Transport facts

The only channel is the stdio spawn line (`ELECTRON_RUN_AS_NODE=1 <root>/zcode <root>/resources/glm/zcode.cjs app-server --stdio` with `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` set); localhost HTTP listeners are decoys, not protocol surfaces [V-EXP EXP-DISCOVERY-01].

- **Framing:** newline-delimited JSON text over stdio, one envelope per line, UTF-8 — observed live as one wire line per message in both captures [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01]. Not LSP-framed, not JSON-RPC 2.0.
- **Envelope grammar:** client→server request `{"id":<id>,"method":"<method>","params":{…}}`; server reply `{"id":<id>,"result":{…}}` or `{"id":<id>,"error":{"code":<int>,"message":"<string>","data":{…}}}`; notifications/events carry `method`+`params` and **no** `id` [V-EXP EXP-SUBSCRIBE-01].
- **No `jsonrpc` key:** the envelope schema is strict — a key outside the grammar (e.g. a JSON-RPC `jsonrpc` key) fails the envelope parse, and the server answers `-32600 "Invalid ZCode Protocol message"` carrying the zod `issues` array (which reports `unrecognized_keys` for the extra key) [V-SRC bundle resources/glm/zcode.cjs#`sendError("invalid-message",-32600,"Invalid ZCode Protocol message",{issues:…})`] [V-SRC 3.17.1 transport.ts:283-289 — strict `zcodeProtocolMessageSchema.safeParse` → `sendError(...,-32600,...)`]. The classic full request "send `jsonrpc:"2.0"` → `-32600 unrecognized_keys`" observation comes from predecessor private research and was not re-run in this phase [V-RPT private field notes].
- **Undecodable JSON** (not valid JSON at all) answers `-32700 "Parse error"` [V-SRC bundle resources/glm/zcode.cjs#`sendError("parse-error",-32700,"Parse error")`; literal occurs ×2] [V-SRC 3.17.1 transport.ts:292-296].
- **Ids are string-or-int:** replies echo the request's `id` verbatim — our probes addressed the server with integer ids (1…5) and every reply carried the same integer; the server's own reverse requests arrive with string ids (`"server-1"`, `"server-2"`, …) which our host replies echoed [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01]. Per the plan 01-02 encoding-assumption mapping: the grammar is UTF-8 text with string-or-int ids and the captures record which id type each reply echoed — no byte-vs-codepoint ambiguity exists on this wire; consumers correlate by exact id value, not by id type.
- **Error payload shape:** `data` is optional and carries structured detail — observed live: a `-32603` reply whose `data` held `{name:"ZodError", stack, code?}` plus the full issue array in `message` [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01].
- **Request scheduling (what a codec must not break):** ordinary requests execute **serially** on one queue; a bypass lane lets stop/cancel control requests run as soon as the preceding request has *entered* its handler and established its abort controller (not queued behind its async execution); a set of read/lifecycle methods runs fully concurrently outside the queue; reverse-request responses bypass the queue (answering them from the queue would deadlock the pairing) [V-SRC 3.17.1 transport.ts:212-281 — `dispatchLine` serial chain on `processing`, `shouldBypassProcessingQueue`, fire-and-forget lanes]. The shipped bundle contains the same machinery (`shouldBypassProcessingQueue` ×2, `lastQueuedMessageStarted` ×3 occurrences) [V-SRC bundle resources/glm/zcode.cjs#transport dispatch identifiers]. The bypass set is exactly `session/stop`, `workspace/cancelGenerateText`, and the MCP-UI concurrent set; the fully-concurrent set is `v4/command`, `v4/conversation/subscribe`, `v4/commands/query`, `session/lifecycle/getState`, `session/lifecycle/ensureReady`, `session/lifecycle/stop`, `session/lifecycle/restart` (+ plugin reads `plugins/describe|list|overview`) [V-SRC 3.17.1 transport.ts:31-58,305-314]. Corollary for Phase 3: cancel via `session/stop` must jump the wrapper's own write queue, and reverse-request replies must never queue behind prompts.
- **Client EOF:** on stdin EOF the server drains for ~100 ms and then exits — a request issued just before EOF is never answered (the drain window beats request processing; every experiment harness must hold stdin open until collection ends) [V-SRC 3.17.1 transport.ts:28 — `PROTOCOL_EOF_DRAIN_MS = 100`, drain timer at :163] [V-RPT research-session probe: clean exit 0 on EOF, empty capture when stdin closed early]. An empty capture (exit 0, zero lines) is therefore the recorded EOF-pitfall **failure signature** — it must never be read as an empty-but-successful result [V-EXP EXP-SUBSCRIBE-01].
- **Notifications need no reply:** the server proceeded normally with `startup/storageState` notifications never answered [V-EXP EXP-SUBSCRIBE-01].

## 2. Method map

Extracted from the shipped bundle; the method-name table survives minification because names are string literals (`…,interactionBrowserExecute:"interaction/browserExecute"},…`). Extraction commands (pattern form — the bundle root is machine-local and is supplied at run time, never committed):

```bash
# (a) anchored context window (RESEARCH Pattern 4), prettified:
grep -o '.\{600\}sessionSubscribe:"session/subscribe".\{2500\}' \
  <bundle>/resources/glm/zcode.cjs | head -1 | tr ',' '\n' \
  | grep -o '"[a-zA-Z0-9/]*"$' | sort -u
# (b) byte-offset slices around the anchor for the map edges (windows shifted
#     until a wider slice yields no NEW method strings — stability criterion):
grep -bo 'sessionSubscribe:"session/subscribe"' <bundle>/resources/glm/zcode.cjs
dd if=<bundle>/resources/glm/zcode.cjs bs=1 skip=<off-3000> count=12000
# (c) occurrence counts (grep -c counts LINES — useless on a minified bundle):
grep -o '"session/create"' <bundle>/resources/glm/zcode.cjs | wc -l
```

Observed while writing this section: the anchored window (a) plus two shifted slices (b) reached stability — the 12000-char slice added only non-method enum literals, no new method strings. Every method string below occurs **exactly ×1** as a quoted map key in the bundle (`grep -o '"<method>"' | wc -l` = 1), except the v4 family strings also counted below. The same method-name table exists in the 3.17.1 source (`sessionSubscribe: "session/subscribe"` at `index.ts:4265`), corroborating the extraction [V-SRC 3.17.1 index.ts:4381-4391,4265] [V-SRC bundle resources/glm/zcode.cjs#method-name map].

**Host→server requests (110 method strings in the map):**

| Namespace | Methods |
|-----------|---------|
| `session/*` (27) | `create`, `resume`, `send`, `messages`, `read`, `events`, `subscribe`, `list`, `fork`, `compact`, `goal`, `stop`, `cancelBackgroundTask`, `close`, `setModel`, `setThoughtLevel`, `setMode`, `debug`, `subagents`, `releaseUnderDirectory`, `usage`, `requestRuntimePreferences`†, `lifecycle/getState`, `lifecycle/ensureReady`, `lifecycle/stop`, `lifecycle/restart`, `lifecycle/changed`‡ |
| `provider/*` (3) | `refreshRegistry`, `updateAccountConfig`, `testModelConnectivity` |
| `interaction/*` (7) | `requestPermission`, `requestUserInput`, `requestProviderRuntimeHeaders`, `requestOfficialMcpAuthHeaders`, `requestCredentialCipher`, `browserList`, `browserExecute` — all **server→host** (see section 3) |
| `process/*` (2) | `childProcesses`, `networkCapture` (the latter is also a client→server *notification* — it bypasses the serial queue, section 1) |
| `workspace/*` (9) | `generateText`, `cancelGenerateText`, `hooks/trustGrant`, `mcp/trust`, `readPresentation`, `updateDynamicWorkflowPolicy`, `updateInteractionPreferences`, `updateModelIoPreferences`, `updateOffPeakToolPolicy` |
| `automation/*` (6) | `create`, `list`, `update`, `delete`, `pause`, `checkTaskBinding` |
| `plugins/*` (21) | `list`, `listUiSurfaces`, `referenceCatalog`, `referenceCatalogWithCategory`, `resolveSuggestedReference`, `cancelOperation`, `configure`, `describe`, `install`, `marketplace/add`, `marketplace/remove`, `marketplace/update`, `mentionItems`, `overview`, `readUserOptions`, `resetConfig`, `restoreBuiltin`, `setEnabled`, `uninstall`, `update`, `validate` |
| `mcp/*` (18) | `list`, `readResource`, `uiCallTool`, `uiCancelCall`, `uiCancelSampling`, `uiClaimAppToolCall`, `uiCloseInstance`, `uiListResources`, `uiListResourceTemplates`, `uiOpenInstance`, `uiReadResource`, `uiRegisterAppTools`, `uiResolveAppToolCall`, `uiSampling`, `uiSubscribeResource`, `uiUnregisterAppTools`, `uiUnsubscribeResource`, `uiValidateInstance` |
| `workflows/*` (8) | `delete`, `forRun`, `get`, `list`, `move`, `runs`, `save`, `updateMeta` |
| singletons (9) | `bot/channel/reply`, `offPeak/create`, `offPeak/list`, `runtime/capabilities`, `skills/referenceCatalog`, `subagents/readRuntimeConfig`, `topic/resource/read`, `topic/resource/cancel`, `usage/stats` |

† `session/requestRuntimePreferences` is registered in the `session/*` namespace but travels **server→host** (see section 3). ‡ `session/lifecycle/changed` is the lifecycle **notification**.

**v4 family (43 method strings, ×1 each unless noted)** [V-SRC bundle resources/glm/zcode.cjs#v4 method strings]: `v4/command`, `v4/commands/query`, `v4/connection/flow`, `v4/controller/{subscribe,unsubscribe,resync}`, `v4/attachment/{begin,chunk,commit,abort,read,previewSource}`, `v4/conversation/subscribe`, `v4/conversation/unsubscribe`, `v4/conversation/frame` (×2 occurrences), `v4/conversation/contextComposition`, `v4/conversation/contextCompositionChanged`, `v4/conversation/{attachmentRead,attachmentStat,backgroundBashOutput,fileChanges,fileRewindPreview,inputPrediction,offPeakControl,plans,resync,rowsRange,usage,workflowRunArtifactData,workflowRunArtifactRead,workflowRunArtifacts,workflowRunEvents,workflowRunNodeResult,workflowRuns,workflowRunWorkspace}`, `v4/offPeak/lifecycle`, `v4/sessionsIndex/reconcile`, `v4/storage/sessionData/{measure,deletionPreview,deletionExecute,purge}`, `v4/telemetry/event`, `v4/usage/stats`. One-line re-derivation: `grep -o '"v4/[^"]*"' <bundle>/resources/glm/zcode.cjs | sort -u` — all 43 v4 strings carry digits, so the extraction class above must include `0-9` (the narrower `[a-zA-Z/]*` form silently dropped every v4 string it matched). The v4 conversation channel is where turn lifecycle events live (section 5); its subscribe params differ from `session/subscribe` (section 4).

**Load-bearing quirk — `session/create` drops `options.reasoningLevel` (Phase 4/6 must route around it):** the create params schema *accepts* a full model selection object (`model: modelSelectionSchema`, where `modelSelectionSchema = {providerId, modelId, options?: {reasoningLevel?: string}}`) [V-SRC 3.17.1 index.ts:1979-1999 — `zcodeSessionCreateParamsSchema`] [V-SRC 3.17.1 model-selection.ts:3-17], but the create path serializes the selection to the bare `"providerId/modelId"` string, which has no channel for the level, so the create-time level is silently lost [V-SRC 3.17.1 model-mapper.ts:4-6 — `formatProtocolModelSelection` returns `${ref.providerId}/${ref.modelId}`]. Live consequence on 3.18.2: binding the level afterwards with `session/setModel` **without** `options.reasoningLevel` is refused `-32603 "Reasoning level is required for zai-api/GLM-5.3-Flash"` [V-EXP EXP-TURN-01]. The workaround is the documented contract: create bare, then `session/setModel` whose params schema carries the full selection (`{sessionId, model: modelSelectionSchema, expectedRevision?, persistAsWorkspaceLastUsed (default true)}`) [V-SRC 3.17.1 index.ts:2396-2403] [V-SRC 3.17.1 server.ts:757 — dispatch entry]. Level validation is strict in both directions (`reasoning-level-missing` / `reasoning-level-not-supported` selection issues) [V-SRC 3.17.1 model-selection.ts:22-31]; the exact refusal string literal occurs ×1 in the bundle [V-SRC bundle resources/glm/zcode.cjs#`Reasoning level is required` ×1]. The predecessor shim proved the two-step flow end-to-end on 3.18.2 [V-RPT private field notes]. Our probe sent `persistAsWorkspaceLastUsed:false` so the experiment could not mutate account config state [V-EXP EXP-TURN-01].

## 3. Server-to-host surface

The app-server runs in **host-managed mode**: it sends requests *to* the host and blocks the work that depends on them until answered. Two request families and a notification stream travel this direction.

**Requests (a reply is required; correlation by the request's string id):**

| Method | Purpose / shape | Provenance |
|--------|-----------------|------------|
| `session/requestRuntimePreferences` | Fires at `session/create` (params `{sessionId, scope:"runtime-materialization"}`) and again at `session/send` (scope `"user-execution"`); a minimal `{nativeSearchEnhancementsEnabled:false, memoryEnabled:false}` result is accepted; unanswered requests fail the surrounding operation (creates failed `-32603` before the harness answered) [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01] [V-EXP EXP-WORKSPACEKEY-01] | |
| `interaction/requestPermission` | Tool-permission round-trip; broker re-announces pending requests at a 1 s interval; answers map to permission decisions [V-SRC 3.17.1 interaction-broker.ts:60 (`INTERACTION_REQUEST_REANNOUNCE_INTERVAL_MS = 1_000`)] [V-SRC 3.17.1 index.ts — permission request/response schemas] | |
| `interaction/requestUserInput` | The ask-the-user channel (used e.g. by an AskUserQuestion-style tool): params `{requestId, sessionId, turnId?, toolCallId?, toolName?, prompt?, questions?: [{question, header, options (≥1), multiSelect?}], input?, origin?, schema?}` — one or more questions with fixed options; response `{action: "accept"\|"decline"\|"cancel", content?, reason?}` where `content` carries the per-question answers/annotations. The broker runs it over two equivalent channels — this legacy RPC and the v4 pending-interaction flow — with the answer passed through verbatim [V-SRC 3.17.1 index.ts:2808-2841 — question/request/response schemas] [V-SRC 3.17.1 interaction-broker.ts:247,342-400 — dual-channel comment + `v4AnswerToUserInputResponse`] [V-SRC bundle resources/glm/zcode.cjs#`interactionRequestUserInput` map key ×1]. (Resolves the phase's open question 4: a plain no-tool turn does not trigger it — none fired in the E2 turn [V-EXP EXP-TURN-01].) | |
| `interaction/requestProviderRuntimeHeaders` | Credential/headers delegation for account providers; answer `{headersApplied:true, requestAuth:{apiKey, apiKeyId}}`; the server holds a credentials queue up to **180 s** for the answer (`180000`, minified `18e4`, ×5 occurrences in the bundle) and enforces `-32031` when headers were not applied; re-asks with `rejectedProjectTokenFingerprint` after a rejected token; emits `interaction/providerRuntimeHeadersCancelled` on abort/timeout [V-SRC 3.17.1 provider-runtime-headers.ts:17,48-100] [V-SRC bundle resources/glm/zcode.cjs#`18e4` ×5, `Provider runtime headers were not applied` ×2] [V-RPT private field notes — live `headersApplied:false` turn failure] | |
| `interaction/requestOfficialMcpAuthHeaders` | Official-plugin MCP auth headers (full shape in the section below); params `{mcpKey, pluginId, requestId, targetOrigin, workspace{workspaceKey, workspacePath}, consumer?}`; fired 6–7 times during every `session/create` for the user's installed official plugins; the host replies a RESULT (never an error envelope) — `{ok:true, headers}` / `{ok:false, reason:"official_mcp_origin_untrusted"}` / `{ok:false, reason:"official_auth_unavailable"}` — and the plugin proceeds header-less on `ok:false`. Answering `-32601` was harmless in the E1 idle-create context but **stalls the post-auth plugin runtime materialization** (attempt-1 field proof, section below) — never answer it -32601 [V-SRC bundle resources/app.asar#interactionRequestOfficialMcpAuthHeaders] [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-CODINGPLAN-CYCLE-01] | |
| `interaction/requestCredentialCipher` | Keyring-material delegation: params `{requestId, operation:"encrypt"\|"decrypt", value}`; **fires mid-turn** in host-managed mode (observed `operation:"decrypt"` ~10 s after send) — a full turn requires the host to answer it, and an unanswered one stalls the turn (the Phase 1 degradation cause) [V-EXP EXP-TURN-01]. Not present anywhere in the 3.17.1 tree — a 3.18-only surface (section 9) [V-SRC bundle resources/glm/zcode.cjs#`interactionRequestCredentialCipher` map key ×1] [V-SRC 3.17.1 clone — `grep -r requestCredentialCipher` → 0 files] | |
| `interaction/browserList`, `interaction/browserExecute` | Browser-tool delegation pair, registered in the interaction map [V-SRC bundle resources/glm/zcode.cjs#`interactionBrowserList`/`interactionBrowserExecute` map keys ×1 each]; not exercised live in Phase 1 (wire shapes remain source-level until a browser-enabled turn is run) [V-RPT research notes] | |

**Notifications (no reply expected):** `state.updated` (×72 bundle occurrences; patch/reason/revision/scope/sessionId/workspace — observed `{patch:{status:"running"}, reason:"prompt_started", revision, scope:"session"}` at send [V-EXP EXP-TURN-01]), `startup/storageState` (×3 bundle occurrences; observed ×5 per boot with monotone `sequence` 1→5 through phases `checking`→`committing`→`ready`, pinning `databaseId`, `databaseKind`, migration info [V-EXP EXP-SUBSCRIBE-01]), `session/lifecycle/changed` (observed `{sessionId, state: starting→ready, generation, revision, lastError}` [V-EXP EXP-SUBSCRIBE-01]), `process/mcpTelemetry` (observed ×6–7 per create: `process_start` per plugin, `session_startup` summary [V-EXP EXP-SUBSCRIBE-01]), `session/event` (per-event envelopes with `seq` numbering and `deliveryKind` — section 5 [V-EXP EXP-TURN-01]), `computer-use/operation-event` (observed `kind:"turn-started"` + `turnId` + `sequenceNumber` [V-EXP EXP-TURN-01]), `v4/telemetry/event` (observed `kind:"turn.started"` and `kind:"model.request.status"` with `eventSeq` [V-EXP EXP-TURN-01]), `v4/conversation/contextCompositionChanged` (observed `{sessionId, revision}` [V-EXP EXP-TURN-01]), `v4/conversation/frame` (×2 bundle occurrences; the v4 channel's data frames) [V-SRC bundle resources/glm/zcode.cjs#notification strings with counts]. Sequence keys are per-stream: `sequence` on storageState, `revision` on lifecycle/contextComposition/state patches, `seq` on session events, `eventSeq` on v4 telemetry and on the subscribe snapshot cursor [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01].

**Envelope surface observed per create** (harness surface-summary verdict): `startup/storageState` ×5, `session/lifecycle/changed` ×2, `process/mcpTelemetry` ×6–7, `interaction/requestOfficialMcpAuthHeaders` ×6–7, `session/requestRuntimePreferences` ×1 [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01].

### `interaction/requestOfficialMcpAuthHeaders` — the official-plugin MCP auth ask, 3.18.2 [V-SRC bundle resources/app.asar#interactionRequestOfficialMcpAuthHeaders] [V-EXP EXP-CODINGPLAN-CYCLE-01]

The desktop's host-side handler for the official plugins' MCP auth-headers ask (installed 3.18.2
bundle `resources/app.asar`, `interactionRequestOfficialMcpAuthHeaders` host handler; field-proven
by the EXP-CODINGPLAN-CYCLE-01 attempt-1 raw capture, 2026-11-22). Fired once per official plugin
per `session/create` during runtime materialization — observed `mcpKey:"sites"`,
`pluginId:"zcode-site@zcode-plugins-official"`, `targetOrigin:"https://zcode.z.ai"`,
`requestId:"official-mcp-auth:1"`.

**Params:** `{mcpKey: string, pluginId: string, requestId: string, targetOrigin: string, workspace: {workspaceKey, workspacePath}, consumer?: string}`.

**The host replies a RESULT (never an error envelope) — three shapes:**

| Reply | Meaning |
|-------|---------|
| `{ok:true, headers:{...}}` | the host's resolver resolved auth headers for `targetOrigin` |
| `{ok:false, reason:"official_mcp_origin_untrusted"}` | `targetOrigin` failed the host-side trust check |
| `{ok:false, reason:"official_auth_unavailable"}` | no auth-headers resolver available — the desktop's own honest "cannot resolve" decline |

**The plugin proceeds header-less on `ok:false`** — the decline is the designed survivable path.

**Field proof that `-32601` breaks materialization (attempt-1):** the probe session's create fired
the ask and the shim answered the -32601 error envelope (capture `:15-16`). The create reply's
`settings` block then carried only the builtin registry (zai-api builtin pair + a third-party) —
the pushed account-provider models never surfaced, so the registry read-back verdict timed out and
the authenticate attempt aborted (the probe session closed at the create boundary). The honest
structured decline (`official_auth_unavailable`) keeps the plugin's designed header-less path open
so materialization can complete.

**Wrapper policy:** the wrapper answers the honest `official_auth_unavailable` decline — its own
native "no resolver" shape. Never `{ok:true}` (there is no official-MCP auth resolver behind the
wrapper; fabricated headers would poison the plugin's auth state), never `-32601` (it stalls
runtime materialization, above), never silence.

## 4. session/subscribe and event-stream availability

Verdict of experiment E1 against the shipped binary (idle session, no send), from the per-method verdict lines recorded in the capture [V-EXP EXP-SUBSCRIBE-01]:

| Method | Verdict | Detail |
|--------|---------|--------|
| `session/subscribe` | **ACCEPTED** | result `{sessionId, eventSeq, events}` — idle session: `eventSeq: 0`, `events: []` (an empty list is a valid result, not an error) |
| `session/events` | **ACCEPTED** | poll-style alternative — result `{events}` (empty list on an idle session) |
| `v4/conversation/subscribe` | **REFUSED** `-32603` | zod error enumerates the *real* params surface: `{topic: string, connectionId: string, clientMode: "desktop-continuous"\|"web-remote-replayable"}`; `sessionId`/`deliveryKind` reported as `unrecognized_keys`. A refusal is wire truth: the v4 conversation channel is a separate subscription with its own params, not a session-scoped one |

- Params/result schemas corroborate the verdict: `session/subscribe` params `{sessionId, deliveryKind, afterSeq?, includeSnapshot (default false)}` (strict); result `{sessionId, eventSeq, events[], snapshot?}` — the observed result is exactly this minus the omitted optional `snapshot` (consistent with the default `includeSnapshot:false`) [V-SRC 3.17.1 index.ts:1916-1931 — subscribe params/result schemas] [V-EXP EXP-SUBSCRIBE-01].
- `deliveryKind` enum: `"desktop-continuous" | "web-remote-replayable"` [V-SRC 3.17.1 zcode-protocol-legacy-types.ts:14] — the same pair the v4 refusal named for `clientMode`. Bundle counts: `deliveryKind` ×61, `desktop-continuous` ×15, `web-remote-replayable` ×11 [V-SRC bundle resources/glm/zcode.cjs#occurrence counts]. Observed on the wire: every `session/event` envelope carried `deliveryKind:"desktop-continuous"` [V-EXP EXP-TURN-01].
- **Empty-input behavior** (per the plan 01-02 assumption mapping): an idle subscribe returns the empty list shown above; an **unknown/unrecognized method key** lands in the error family `-32600` (`InvalidRequest` — invalid envelope/keys) / `-32601` (`MethodNotFound`) — both literals are bundle error-enum entries (`e[e.MethodNotFound=-32601]="MethodNotFound"`, `e[e.InvalidRequest=-32600]="InvalidRequest"`) [V-SRC bundle resources/glm/zcode.cjs#error-code enum]; the live observation of the family's zod edge is the v4 refusal above (`unrecognized_keys` listing valid keys) [V-EXP EXP-SUBSCRIBE-01]. The genuinely-unknown-method path was not itself probed with a bogus method name — the enum literals plus the schema-strictness mechanism are the evidence; marked here so the reader knows which half is live [V-EXP EXP-SUBSCRIBE-01] [V-RPT research-session probe].
- **Envelope names + sequence numbers on an idle boot** (15 s window): `startup/storageState` ×5 with `sequence` 1→5; `session/lifecycle/changed` ×2 with `revision` 1→2; `process/mcpTelemetry` ×6; **no `state.updated`** in the window after `provider/refreshRegistry` — recorded honestly as *not observed in that window* (the E1 design's assumption A3); `state.updated` *was* observed live in E2 at send time, so the honest statement is "not triggered by refreshRegistry alone within 15 s", not "does not exist" [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01].
- `session/subscribe` result on an **active** turn: the E2 run subscribed before sending and the turn's events arrived as `session/event` envelopes with `seq` numbering (section 5) — i.e. the subscribe-then-receive shape works on 3.18.2 for turn-start traffic [V-EXP EXP-TURN-01].

## 5. Turn-completion terminal signals

**Honesty statement (D-05 degradation, applies to the completion half of this section):** across three live runs the model stream was cancelled by harness shutdown before any completion envelope arrived, and the session status patch stayed `"running"` until teardown; answering the mid-turn `interaction/requestCredentialCipher` (operation `decrypt`) would require decrypted keyring credential material, which Phase 1 deliberately never reads (threat register T-1-05). Turn-completion terminal envelopes were therefore **never observed on the wire in this phase**; every completion claim below is labelled `V-SRC-only` (bundle source enumeration) or `V-RPT`, with this limitation attached [V-EXP EXP-TURN-01]. Quota honesty: 3 tiny `session/send` calls issued, none completed a model response, no keys minted [V-EXP EXP-TURN-01].

**Observed live — turn-start signal table (what a turn machine receives when a turn begins)** [V-EXP EXP-TURN-01]:

| Signal | Envelope | Payload shape (as observed) |
|--------|----------|------------------------------|
| Send accepted | reply to `session/send` | `{accepted:true, sessionId, stateRevision}` — fast ack, turn continues async |
| Session status patch | `state.updated` | `{patch:{status:"running"}, reason:"prompt_started", revision, scope:"session", sessionId, workspace}` |
| Turn-started (computer-use stream) | `computer-use/operation-event` | `{kind:"turn-started", turnId, sequenceNumber, sessionId, eventId, timestamp}` |
| Turn-started (v4 telemetry) | `v4/telemetry/event` | `{kind:"turn.started", turnId, eventSeq, sessionId, modelName, modelProvider, version:1, …}` |
| Turn-started (session event stream) | `session/event` | `type:"turn.started"` with `{seq, turnId, traceId, payload{messageId, queryId, turnNumber, input, …}}` |
| Title set | `session/event` | `type:"session.titleUpdated"` `{seq, payload{title, previousTitle, source:"first_input"}}` |
| Model request started | `session/event` + `v4/telemetry/event` | `type:"session.updated"` payload `model_request_started` (baseURL, requestHeaders, transport `sse`, attempt 1 of maxAttempts 11) paired with `v4/telemetry/event` `kind:"model.request.status" status:"model_request_started"` |
| Context recomposition | `v4/conversation/contextCompositionChanged` | `{sessionId, revision}` (observed revision 1→3 during one turn) |

All `session/event` envelopes above carried `deliveryKind:"desktop-continuous"`, monotone `seq`, `turnId` and `traceId` [V-EXP EXP-TURN-01].

### The model-response delta surface — `session/event` `model.streaming`, 3.18.2 [V-EXP EXP-APIKEY-CYCLE-01]

The completed guarded runs (capture file `experiments/captures/EXP-APIKEY-CYCLE-03.jsonl`, the `-03`
re-run under the test-locked header id per the capture-monotonicity rule — see its Sources row)
recorded the first live STREAMING turns, and the carrier fact is load-bearing for any client-facing
wrapper: **the model's reply text rides `session/event` envelopes with `type:"model.streaming"`
(the desktop-continuous delivery kind as every `session/event` envelope), with the delta inside
`params.payload` — NOT the `part.delta` type the 3.17.1-source v4 taxonomy names.** `part.delta`
stayed ×0 on the wire across every committed capture: the taxonomy block is registration names,
not observed traffic. Frame shape, capture-verbatim (scrubbed, `:177` — the turn's first delta):

```json
{"method":"session/event","params":{"deliveryKind":"desktop-continuous","eventId":"<uuid-69>","payload":{"assistantMessageId":"msg_mwqsxze9_<uuid-70>","delta":"The","done":false,"kind":"reasoning_delta"},"seq":7,"sessionId":"sess_<uuid-4>","timestamp":1795289387694,"traceId":"<uuid-5>","turnId":"turn_<uuid-9>","type":"model.streaming"}}
```

**The `payload.kind` vocabulary is a closed bundle enum.** The `model.streaming` payload schema
(`assistantMessageId`/`delta`/`done`/`input` members; `kind` enum) reads verbatim in the bundle:
`"start","finish","error","text_start","text_delta","text_end","reasoning_start","reasoning_delta",
"reasoning_end","tool_input_start","tool_input_delta","tool_input_end","tool_call"`, registered as
the `model.streaming` event schema alongside `part.delta` and `tool.updated` [V-SRC bundle
resources/glm/zcode.cjs#model.streaming payload schema + registration; occurrence counts:
`text_delta` ×25, `reasoning_delta` ×18, `tool_input_delta` ×19, `tool_input_start` ×13,
`model.streaming` ×10]. Observed live on the `-03` capture: `reasoning_delta` (thinking text —
first at `:177`, seven across turn 1), `text_delta` (reply text — `:218` `"ok"`, `:425`
`"READ-OK"`), and the tool-args trio `tool_input_start`/`tool_input_delta`/`tool_input_end`
(`:364`–`:366`; the delta carries the tool-call JSON `{"file_path":…}`).

**Wrapper translation (the QA-03 contract):** per delta envelope, exactly ONE ACP client frame in
arrival order — `kind:"text_delta"` → `session/update` `agent_message_chunk` with content
`{type:"text", text:<delta>}`; `kind:"reasoning_delta"` → `agent_thought_chunk`; every other kind —
the tool-args stream and the start/finish/error markers — is a desktop-UI fact, held from the
client surface (tool cards ride the `tool.updated` stream, whose `scheduled` frame carries the
full input). `payload.done` is not a completion signal in practice: every observed delta carried
`done:false` (`"done":true` ×0 in the capture) — the turn-end verdict rides `turn.completed` /
the quiet-window settle, and the wrapper emits no extra client frame on done. Stream shape:
monotone per-session `seq`, `turnId` scoped per turn, deltas interleaved across
`assistantMessageId`s; the client-visible chunk order is the arrival order (single demux path).
Field proof of the translation's necessity: the 2026-11-22 Orca card reproduction completed
`end_turn` (~9.3 s, real model reply) while the ACP client received ZERO `session/update` frames —
the wrapper translated only `part.delta`, which the desktop never sends [V-RPT private field
notes — the QA-03 field report].

**The completion-side reconcile (the no-delta reply path, QA-03 second half):** the delta stream
itself is not guaranteed per turn — the same prompt (`Reply with the single word: ok`) produced
streaming `model.streaming` turns on some desktop runs and turns that completed `end_turn` with
ZERO deltas on others (the clean-build field turn emitted zero client frames; the reply text
existed only in the backend's message store) [V-RPT private field notes — the QA-03 field
report]. A streaming-only translation therefore leaves such turns with an empty client reply.
The wrapper's reconcile closes that path: when a turn resolves completed and ZERO
`agent_message_chunk` were emitted during the turn, the wrapper polls `session/messages` ONCE
(bounded read; the idle-window poll shape — params `{sessionId}`, result carries the messages
projection), extracts the LAST `role:"assistant"` entry's `type:"text"` parts (concatenated in
part order; the `reasoning`/`step-start`/`step-finish`/`timeline` parts are never message text),
and emits it as ONE `agent_message_chunk` BEFORE the stopReason reply leaves — the chunk rides
the same flush as any streamed delta. The projection wire truth (capture-verbatim scrubbed):
the completed turn's poll carried the reply as the last `messages[]` entry with
`info.role:"assistant"`, `info.finish:"stop"`, and parts `step-start` → `reasoning` (thinking
text) → `text` (`"ok"` — the reply) → `step-finish` [V-EXP EXP-CODINGPLAN-CYCLE-01]. Honesty
arms: deltas DID stream → the reconcile never fires (no duplicate); the poll fails, or the
projection carries no assistant reply, or no `type:"text"` parts → NO fabricated chunk and the
stopReason still leaves (the client sees the honest empty reply + `end_turn`, never a hang).
The machine's own idle-window polls cannot source this text: their poll results are consumed as
presence/time evidence only — content never interpreted, nothing stored — and the poll loop is
joined before the turn resolves, so the reconcile reads the surface once at completion.

**Completion half — `V-SRC-only` (bundle source enumeration; limitation above applies):**

| Signal | Where it lives | Evidence |
|--------|----------------|----------|
| `turn.completed` / `turn.failed` | **v4 conversation channel events** (registration names in the v4 taxonomy), not `session/subscribe` payloads | Quoted literals `"turn.completed"` ×7 and `"turn.failed"` ×8 in the bundle (bare unquoted forms ×4 / ×5) — re-derived while writing this doc and matching the capture's source-enumeration block exactly [V-SRC bundle resources/glm/zcode.cjs#occurrence counts] [V-EXP EXP-TURN-01] |
| Kind literals `turn-completed` / `turn-failed` / `turn-started` | typed event kinds, each schema carrying `turnId` — the completion events identify their turn [V-SRC bundle resources/glm/zcode.cjs#kind literals + `turnId` co-occurrence] [V-EXP EXP-TURN-01] | |
| Full v4 event taxonomy (25 registration names) | `turn.started, turn.steerQueued, turn.steerDrained, turn.completed, turn.failed, message.upserted, message.removed, part.started, part.delta, part.upserted, part.removed, model.streaming, tool.updated, permission.requested, permission.resolved, userInput.requested, userInput.resolved, checkpoint.created, rewind.triggered, session.created, session.resumed, session.updated, session.titleUpdated, session.closed, streamRecovery.updated` — the taxonomy block is verbatim present in the bundle [V-SRC bundle resources/glm/zcode.cjs#v4 registration list] [V-EXP EXP-TURN-01] | |
| How a host receives them | subscribe to the v4 conversation channel — whose real params E1 recorded: `{topic, connectionId, clientMode}` (section 4 refusal) — or consume `v4/conversation/frame` notifications [V-EXP EXP-SUBSCRIBE-01] [V-SRC bundle resources/glm/zcode.cjs#`v4/conversation/frame` ×2] | |

**Correction (2026-11-22, the `-03` run): the terminal surface IS observed live now.** The first
completed turn of the `-03` capture received `session/event` `type:"turn.completed"` on the
desktop-continuous stream (`:227`, `seq` 14, ~1 s after the last delta, payload carrying
`response`/`tokenCount`/`usage`/`toolCallCount`/`resultType:"success"`) — well inside the
quiet-floor window, i.e. the terminal event resolves such turns before the quiet fallback fires.
The `-02` run's no-terminal-in-window fact stands as recorded above; both facts together read:
the terminal envelope may arrive and still be absent on some completed turns — a consumer keeps
both signals armed.

**Reliability column (planning input for Phase 4):**

- Explicit finish events (`turn.completed`/`turn.failed`): source-level only until a credential-answering host exists (Phase 6) — `V-SRC-only`, limitation as stated [V-EXP EXP-TURN-01].
- `state.updated` idle patch (`patch.status` leaving `"running"`): the settle gate of the harness waits for it; in the three runs it never arrived before teardown — do not rely on it alone [V-EXP EXP-TURN-01] [V-RPT private field notes].
- **Quiet-window fallback (2.5 s without new parts): the proven secondary** — field-verified end-to-end by the predecessor shim, and the harness implements its hardened form (send-latency floor = 3× quiet-ms after the send reply, plus the settle gate) after live runs showed a bare quiet window mis-arms during first-token latency [V-RPT private field notes] [V-EXP EXP-TURN-01].
- Predecessor-shim provider logs carried `model.request.completed` / `turn.completed` markers at turn end — client-side log events, not captured wire envelopes [V-RPT private field notes; quoted in the capture's v-rpt block] [V-EXP EXP-TURN-01].

### The one-turn-at-a-time latch — post-completion window, 3.18.2 [V-EXP EXP-APIKEY-CYCLE-01]

Attempt 2 of the guarded api-key full cycle recorded the first COMPLETED real turn and its refused
successor (capture file `experiments/captures/EXP-APIKEY-CYCLE-02.jsonl`, committed under the
test-locked V-EXP id per the capture-monotonicity rule — its header line 1 carries the id above;
line references below point into that file). Every fact in this subsection is capture-verified.

- **A genuinely COMPLETED turn may surface NO terminal envelope within the quiet-floor window.**
  The first real turn (end_turn, zero tool frames) emitted the send (`:36`), the `state.updated`
  running patch (`reason:"prompt_started"`, revision 2, t=45.010, `:38`) and the turn-start
  signals (context recomposition, session events, computer-use `turn-started`, v4/telemetry
  `turn.started`, t=45.130–45.239, `:41`–`:47`) — and then NEVER a terminal envelope: no
  `turn.completed` carrier on any of the three channels (session/event / v4/telemetry /
  computer-use), and no `state.updated` idle patch after the prompt_started patch anywhere in the
  window (`:38`–`:117`). The completion verdict was the quiet-window settle at floor expiry — send
  ack 45.0105 + 3×2500 ms = 52.5105, landing on the aligned 500 ms poll tick (`:116`) —
  strengthening the reliability column above: even a genuinely completed turn may present no
  terminal surface at all.
- **The post-completion latch is real and lingers ~7.5 s past the ack.** A second `session/send`
  issued ~2 ms after the watcher's resolution (52.5133, `:118`) was refused `-32010 "A prompt is
  already running for this session"` at 52.5149 (`:119`) — the same concurrency guard section 6
  records for mid-turn sends, now observed POST-completion: the desktop holds the one-turn latch
  after its turn finished.
- **The interaction loop is the latch's live signal.** `interaction/requestCredentialCipher`
  decrypt asks (fresh wire ids `server-7`…`server-21` with a fresh `requestId` each, ~500 ms
  cadence) continued until 52.462 — ~50 ms before the refused send (`:50`–`:115`, last
  ask/answer `:114`–`:115`).
- **The poll surface cannot read the latch.** `session/messages` poll replies (ids 9–23, 500 ms
  cadence, `:48`–`:117`) carried ONLY `messages` — no projection, no session status — so a
  poll-based watcher cannot observe the latch state; the quiet-window gates remain the only
  completion signal this surface offers.
- **Open question, stated honestly: the TRUE terminal-surface shape is UNOBSERVED.** Whether a
  late `state.updated` idle patch or a completion `session/event` arrives after the latch releases
  is not in this capture; the next guarded run records it and arbitrates hypothesis (a) desktop
  latch-lag (the wrapper's completion resolved before the desktop released the latch) vs
  hypothesis (b) premature `end_turn` (our verdict fired before the desktop's true completion).
  The wrapper's countermeasure — a latch drain between sends, rehearsed as the fake-tier corpus
  row `lingering-latch-32010` — is correct under both.

## 6. Error-code table

Literals extracted from the shipped bundle; occurrence counts are re-derivable, e.g. `grep -o -e '-32603' <bundle>/resources/glm/zcode.cjs | wc -l`. The `-e` is load-bearing: a bare leading-dash pattern (`grep -o '-32603' …`) is silently parsed as numeric context options by GNU grep — rc=1, no output, no error (verified with GNU grep 3.11) — so the plausible-looking form quietly "counts" nothing. The bundle carries a symbolic enum for the standard JSON-RPC codes plus protocol-specific codes (`e[e.MethodNotFound=-32601]="MethodNotFound"`, …) [V-SRC bundle resources/glm/zcode.cjs#error-code enum]:

| Code | Enum name (where present) | Meaning / trigger | Bundle occurrences | Live evidence |
|------|---------------------------|-------------------|--------------------|---------------|
| `-32700` | ParseError | Line is not valid JSON — `"Parse error"` | ×2 | — [V-SRC bundle] |
| `-32600` | InvalidRequest | Envelope fails the strict message schema — `"Invalid ZCode Protocol message"` + zod `issues` (an extra key such as `jsonrpc` lands here as `unrecognized_keys`) | ×3 | — [V-SRC bundle] [V-RPT private field notes] |
| `-32601` | MethodNotFound | Unknown method key | ×5 | Harness answered server→host requests with -32601 without breaking flows [V-EXP EXP-SUBSCRIBE-01] |
| `-32602` | InvalidParams | Params-schema failure | ×14 | — [V-SRC bundle] |
| `-32603` | InternalError | Server-side refusal/inner error; the observed ZodError and ModelProtocolError refusals travel here | ×4 | v4/subscribe refusal; `session/setModel` "Reasoning level is required" refusal [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01] |
| `-32002` | ResourceNotFound | Named-resource miss | ×1 (enum) | — [V-SRC bundle resources/glm/zcode.cjs#error-code enum] |
| `-32010` | — | Concurrency guard: `"A prompt is already running for this session"` (literal ×1) — a second `session/send` while a turn runs; also used for the subagent read-only guard | ×4 | — [V-SRC bundle] [V-SRC 3.17.1 server-operations.ts:2036-2045] |
| `-32020` | — | Header mismatch on provider runtime headers (`-32020 (HeaderMismatch)` literal) | ×8 | — [V-SRC bundle] [V-SRC 3.17.1 provider-runtime-headers.ts] |
| `-32021` | MissingRequiredClientCapability | Client capability missing | ×3 (enum) | — [V-SRC bundle] |
| `-32022` | UnsupportedProtocolVersion | Protocol version unsupported | ×5 (enum) | — [V-SRC bundle] |
| `-32031` | — | `"Provider runtime headers were not applied before model request attempt."` — the turn-level enforcement that a `headersApplied:false` answer is fatal for account providers | ×5 | — [V-SRC bundle] [V-SRC 3.17.1 provider-runtime-headers.ts:84-100] [V-RPT private field notes — live turn failure] |
| `-32042` | UrlElicitationRequired | URL elicitation required | ×1 (enum) | — [V-SRC bundle resources/glm/zcode.cjs#error-code enum] |

Matching rule for consumers: match the numeric **code**, never the message text — messages are not a stable contract (some are locale-mixed in source, e.g. the registry's `"ZCode Built-in 相同 revision …"` error observed in the bundle ack neighborhood) [V-SRC bundle resources/glm/zcode.cjs#registry comparison neighborhood] [V-RPT private field notes]. Drift note: `-32042` and `-32002` exist in the 3.18.2 bundle but nowhere in the 3.17.1 tree; `-32010/-32020/-32021/-32022/-32031/-32700` exist in both (section 9) [V-SRC 3.17.1 clone — per-code grep, 2026-11-19].

## 7. Session lifecycle methods

- **Method set:** `session/create`, `session/resume`, `session/close`, `session/stop`, `session/cancelBackgroundTask` — all present ×1 in the bundle map; dispatch entries corroborated in the 3.17.1 server (`server.ts:746 stopSession`, `:754 closeSession`, plus `forkSession`, `compactSession`, `goalSession`, `setModel`, `setThoughtLevel`, `setMode`) [V-SRC bundle resources/glm/zcode.cjs#method-name map ×1 each] [V-SRC 3.17.1 server.ts:735-757] [V-EXP EXP-TURN-01] [V-RPT research notes]. Live in Phase 1: create and send accepted; stop/close not exercised live [V-EXP EXP-TURN-01].
- **`session/delete` is verifiably ABSENT:** `grep -o '"session/delete"' <bundle> | wc -l` → **0** occurrences in the shipped 3.18.2 bundle — there is no delete method on this wire, nothing to delete. This is the in-repo evidence for the project stance that a host answers a client `session/delete` request as a graceful no-op (REQUIREMENTS.md: "Real `session/delete` server-side — no wire method exists in app-server; nothing to delete"; SESS-02) [V-SRC bundle resources/glm/zcode.cjs#`"session/delete"` ×0].
- **Stop vs lifecycle-stop:** cancellation has two lanes with different scheduling — `session/stop` uses the bypass lane (runs as soon as the in-flight request has entered its handler and built its abort controller), while `session/lifecycle/stop` is in the fully-concurrent set and never queues at all (section 1) [V-SRC 3.17.1 transport.ts:31-43,247-262,305-314].
- **Lifecycle notifications:** `session/lifecycle/changed` carries `{sessionId, state, generation, revision, lastError}` — observed `starting`→`ready` with `generation:1`, `revision` 1→2 on every boot-to-create [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01]. A separate `session/lifecycle/{getState,ensureReady,stop,restart}` request family exists in the map and is fully concurrent [V-SRC bundle resources/glm/zcode.cjs#method-name map] [V-SRC 3.17.1 transport.ts:36-38].
- **create carries host obligations:** every observed `session/create` triggers `session/requestRuntimePreferences` and several `interaction/requestOfficialMcpAuthHeaders` requests before the session reaches `ready`; the create reply is the full session snapshot (`result.session.sessionId`, `settings.model.available` with per-model `reasoning.levels`, `slashCommands`, …) [V-EXP EXP-SUBSCRIBE-01] [V-EXP EXP-TURN-01] [V-EXP EXP-WORKSPACEKEY-01]. `workspaceKey` semantics of the create params are documented separately in [workspace-key.md](workspace-key.md) (DISC-04) — wire schema `nonEmptyString`, host-opaque [V-EXP EXP-WORKSPACEKEY-01] [V-SRC 3.17.1 index.ts:2762,2784].
- **resume:** `session/resume` exists in the map (×1) and in the 3.17.1 source; not exercised live in Phase 1 [V-SRC bundle resources/glm/zcode.cjs#method-name map] [V-SRC 3.17.1 index.ts] [V-RPT research notes].

## 8. Snapshot semantics

The registry snapshot flows host→server via `provider/updateAccountConfig`; the protocol gives it **no acceptance channel** — this asymmetry is the section's warning-level fact.

- **Ack shape:** the reply result schema is `{receivedRevision, providerCount, status: enum(["received","unchanged"])}` — bundle literal `status:m.enum(["received","unchanged"])`; the 3.17.1 handler returns `status: changed ? "received" : "unchanged"` — i.e. the ack reflects the config-sync step only [V-SRC bundle resources/glm/zcode.cjs#updateAccountConfig result schema] [V-SRC 3.17.1 account-provider-config.ts:28-33].
- **An ack `"received"` is NOT acceptance — silent snapshot discard (warning-level fact):** when the snapshot's `basedOnZCodeBuiltinRevision` does not equal the server's active `zcodeBuiltinRevision`, the registry keeps serving the *previous* complete registry and simply continues its refresh loop — no error is emitted to the host; the ack has already said `"received"`. The bundle's registry-comparison neighborhood shows the same logic (`"stale"` literal, revision-equality + content comparison) [V-SRC 3.17.1 registry-service.ts:205-212 — "Built-in 已变化但 Account 仍基于旧事实时，继续服务上一份完整 Registry"] [V-SRC bundle resources/glm/zcode.cjs#registry comparison neighborhood] [V-RPT private field notes — live silent drop: models never materialize, later turns fail provider-not-found].
- **Verification is registry read-back** (what Phase 6 must implement, never ack-trusting): call `provider/refreshRegistry` (observed reply `{"refreshed":true}`) and check the applied state via `state.updated` patches / the model list a fresh session reports — the create reply's `settings.model.available` is the read-back surface [V-EXP EXP-WORKSPACEKEY-01] [V-EXP EXP-SUBSCRIBE-01] [V-SRC 3.17.1 registry-service.ts — refresh + generation design].
- **Strict schema:** the snapshot must parse under pervasive zod `.strict()` — `.strict()` occurs **×1409** in the bundle; extra fields are parse errors, not silent strips; keep the payload to exactly the accepted shape [V-SRC bundle resources/glm/zcode.cjs#`.strict()` ×1409] [V-SRC 3.17.1 index.ts — pervasive `.strict()`].
- **Pushed `zhipu-account` access is the SPARSE `{type, entitled}` pair — nothing else (coding-plan push, warning-level fact, corrected 2026-11-22):** the updateAccountConfig wire schema parses each `providers.<id>.access` under the sparse zod union, and **extra keys are hard `-32600` rejections** (`Unrecognized keys: "accountType", "mode"` — field-proven on the live desktop, EXP-CODINGPLAN-CYCLE-01 attempt 6). An earlier draft of this doc required the complete access shape in the push; that requirement is **superseded**: registry-side completeness comes from the BUILTIN-RULE overlay (`zcodeBuiltinProviders.overlay(accountProviders)` field-merges the pushed `{type, entitled}` over the table rule's `{type, accountType, mode, api, …}`, so the merged provider satisfies the complete schema without the push carrying it) [V-SRC 3.17.1 resolver.ts:184-190 — `withoutGroup()` overlay; provider-config.ts:495 `overlayProviderAccess`] [V-EXP EXP-CODINGPLAN-CYCLE-01] (attempt 6: live `-32600` rejection of the extra keys). The registry-level completeness requirement (`createRegistryProviderConfig` → `providerIssues` gating `providerExecutable`) still holds for the MERGED config [V-SRC 3.17.1 resolver.ts:244,275].
- **The runtime needs BOTH provider-config env paths paired (worker startup, warning-level fact, field-proven 2026-11-22):** `resolveNodeProviderRuntimePaths` throws unless `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` and `ZCODE_PERSONAL_PROVIDER_CONFIG_FILE` are provided TOGETHER; a worker started with only the builtin path runs its protocol surface degraded — `provider/updateAccountConfig` still acks `received`, but account models never publish to `settings.model.available` and `session/setModel` on an account model answers `Provider Registry 中不存在 Model`. Passing the personal path (`<dataBaseDir>/.zcode/v2/provider_config.json`, the desktop's own value) alongside the builtin path makes the same push publish every entitled account model into the create reply's `settings.model.available` [V-SRC 3.17.1 runtime-paths.ts:31-40] [V-EXP EXP-CODINGPLAN-CYCLE-01] (attempts 1–6: five degraded runs, then one paired-env run with the account models live in the create reply).
- **Revision formula:** `zcode-builtin:<rev>:<sha256(path.resolve(activeBuiltinFilePath))>` — the hash input is the **resolved absolute path string** of the active `zcode-builtin.json`, not the file contents. Source lines: `createHash("sha256").update(resolve(this.#activeFilePath)).digest("hex")` and `` revision: `zcode-builtin:${release.revision}:${sourceKey}` `` — the same literals were located in the shipped bundle (quoted with the computed revision-44 example in [install-discovery.md](install-discovery.md)) [V-SRC 3.17.1 zcode-builtin-provider-config-source.ts:41,241] [V-SRC bundle resources/glm/zcode.cjs#release-store literals] [V-EXP EXP-DISCOVERY-01]. Never hard-code the concrete string — it is machine- and version-specific; compute at runtime [V-RPT private field notes].

## 9. 3.17 vs 3.18.2 drift baseline

First-class section per D-01: everything above documents the **shipped 3.18.2** surface; this section states where the public 3.17.x tree (read at **3.17.1**, `package.json` version) disagrees, so no corroborating citation can be mistaken for shipped-binary truth.

- **Which bundle copy was probed:** the unpacked-AppImage squashfs-root copy (`~/Applications/zcode-<ver>/squashfs-root/`) — the same copy `install-discovery.md` row 1 names as ground truth. Honesty posture re-verified while writing this document: `cmp resources/glm/zcode.cjs resources/glm/zcode.cjs.orig` → byte-identical (pristine, no patching), consistent with the recorded check in `[V-EXP EXP-DISCOVERY-01]`. All bundle anchors and counts in this doc are re-derivable against that copy with the commands recorded in sections 2 and 6 [V-SRC bundle resources/glm/zcode.cjs#pristine cmp identical, 2026-11-19] [V-EXP EXP-DISCOVERY-01].
- **Server→host surface drift — `interaction/requestCredentialCipher`:** present in the 3.18.2 bundle (interaction map key ×1) and **absent from the entire 3.17.1 clone** (`grep -r requestCredentialCipher` across all `.ts` → 0 files, src and dist); it fires mid-turn on 3.18.2 (section 3) [V-SRC bundle resources/glm/zcode.cjs#`interactionRequestCredentialCipher` ×1] [V-SRC 3.17.1 clone — 0 files] [V-EXP EXP-TURN-01]. This is the concrete "never present 3.17.x-source behavior as 3.18.2 truth" example: a wrapper written against the 3.17.1 method table would not know this obligation exists.
- **Method-map diff (bundle map vs the 3.17.1 `packages/shared/src/zcode-protocol/index.ts` table):** exactly **4** of the 110 host→server method strings are absent from the 3.17.1 tree — `interaction/requestCredentialCipher`, `plugins/mentionItems`, `plugins/readUserOptions`, `session/releaseUnderDirectory`; the other 106 (sampled explicitly: `interaction/browserList`, `interaction/browserExecute`, `session/usage`, `session/lifecycle/*`, `offPeak/*`, `automation/*`, `bot/channel/reply`, `topic/resource/*`, `usage/stats`, `runtime/capabilities`) are present in 3.17.1 [V-SRC bundle resources/glm/zcode.cjs#method-name map] [V-SRC 3.17.1 clone — per-method grep, 2026-11-19]. Note the diff direction: the bundle map is a *superset here*; presence in 3.17.1 does not prove identical 3.18.2 params schemas — shapes in this doc are bundle-anchored or experiment-backed wherever load-bearing.
- **Error-code drift:** `-32042` (UrlElicitationRequired) and `-32002` (ResourceNotFound) exist in the bundle and nowhere in the 3.17.1 tree; `-32010/-32020/-32021/-32022/-32031/-32700` exist in both [V-SRC bundle resources/glm/zcode.cjs#error-code enum] [V-SRC 3.17.1 clone — per-code grep, 2026-11-19].
- **`startup/storageState` — honest correction of the plan's draft claim:** present in **both** — bundle ×3, and 3.17.1 `index.ts:476` (`storageStartup: "startup/storageState"`) with the notification schema at `:4509`. It is "new" only relative to the *prior private research docs* (the research-session probe discovered it on the wire); it is **not** a 3.17→3.18.2 drift item, and this section records that correction instead of repeating the draft claim [V-SRC bundle resources/glm/zcode.cjs#`"startup/storageState"` ×3] [V-SRC 3.17.1 index.ts:476,4509] [V-EXP EXP-SUBSCRIBE-01].
- **v4 family:** the v4 conversation method strings and the 25-name registration taxonomy confirmed in the bundle (section 5) have counterparts in the 3.17.1 tree (`packages/shared/src/zcode-protocol-v4/`, `model-selection.ts`, `validation.ts` carry the clientMode enum and v4 session-config shapes) — the v4 channel exists in both versions; the 3.18.2 taxonomy enumeration above is the bundle-side record [V-SRC 3.17.1 zcode-protocol-v4/transport.ts] [V-SRC 3.17.1 validation.ts:803] [V-SRC bundle resources/glm/zcode.cjs#v4 strings].
- **Builtin revision drift:** the revision value moves per desktop version — recorded per-version in [install-discovery.md](install-discovery.md) (28/30/31/43/44 across the runtime tree; 3.18.2 = 44) [V-EXP EXP-DISCOVERY-01]. A snapshot computed against revision 44 is silently discarded on any other version (section 8) — the practical consequence of this drift.
- **Not drift (verified same in both):** transport mechanics (serial queue, bypass lane, concurrent sets, 100 ms EOF drain, `-32600`/-`-32700` envelope errors — the shipped bundle contains the same `shouldBypassProcessingQueue`/`lastQueuedMessageStarted` machinery and the same `-32600 "Invalid ZCode Protocol message"` sendError), the deliveryKind/clientMode enum pair, `requestUserInput`, `workspaceKey` schema lines (`index.ts:2762` required / `:2784` optional) [V-SRC 3.17.1 transport.ts:28-58,212-314] [V-SRC bundle resources/glm/zcode.cjs#transport identifiers] [V-SRC 3.17.1 zcode-protocol-legacy-types.ts:14] [V-SRC 3.17.1 index.ts:4381,2762,2784].

## Sources & provenance

| Tag / id | Origin |
|----------|--------|
| `[V-EXP EXP-SUBSCRIBE-01]` | `experiments/captures/EXP-SUBSCRIBE-01.jsonl` (E1, 2026-11-19, no quota): per-method subscribe verdicts (session/subscribe accepted; session/events accepted; v4/conversation/subscribe refused -32603 with the real params schema), idle-boot envelope surface (`startup/storageState` ×5 with `sequence`, `session/lifecycle/changed` ×2 with `revision`, `process/mcpTelemetry` ×6), host-obligation traffic at create (`session/requestRuntimePreferences` answered minimal; `interaction/requestOfficialMcpAuthHeaders` ×6 answered -32601), integer-client-id/string-server-id echoes. |
| `[V-EXP EXP-TURN-01]` | `experiments/captures/EXP-TURN-01.jsonl` (E2, 2026-11-19, 3 tiny sends / 0 completed): live turn-start signal table (turn-started kinds with `turnId`, `state.updated` running-patch, `session/event` seq envelopes, v4 telemetry + contextComposition), create/setModel/send verdicts (`setModel` refused -32603 "Reasoning level is required"), mid-turn `interaction/requestCredentialCipher` decrypt obligation, the D-05 degradation record (limitation + quota honesty, quoted in section 5), and the source-enumeration block (turn.completed/failed counts, kind literals, 25-name v4 taxonomy). |
| `[V-EXP EXP-WORKSPACEKEY-01]` | `experiments/captures/EXP-WORKSPACEKEY-01.jsonl` (E3, plan 01-01): create-time host obligations (`session/requestRuntimePreferences` — unanswered ⇒ creates fail -32603), `workspaceKey` host-opaque acceptance; read-back surface (`provider/refreshRegistry` → `{"refreshed":true}`). |
| `[V-EXP EXP-CODINGPLAN-CYCLE-01]` | `experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl` (the Phase-6 guarded coding-plan full-cycle run, plan 06-08 — attempt 1 raw capture, 2026-11-22, scrubbed via the canonical scrubber with the scratch prefixes redacted): the failed attempt whose create fired `interaction/requestOfficialMcpAuthHeaders`, got the shim's -32601 error envelope back (`:15-16`), and whose create reply's `settings` then carried only the builtin registry — the account models never surfaced, the registry read-back verdict timed out, and the authenticate attempt aborted. The wire-truth source for the section 3 subsection's reply shapes and the never--32601 policy; ALSO the section 5 completion-side reconcile's projection wire truth (the completed turn's idle-window poll reply carried the reply text as the LAST `role:"assistant"` entry's `type:"text"` parts — poll id 24, the reconcile paragraph's extraction contract); re-runs persist as monotonic `-02`/`-03` files under the same test-locked header id (the EXP-APIKEY-CYCLE precedent). |
| `[V-EXP EXP-APIKEY-CYCLE-01]` | `experiments/captures/EXP-APIKEY-CYCLE-01.jsonl` plus its monotonic re-run files `-02`/`-03` under the same test-locked header id (`"capture":"EXP-APIKEY-CYCLE-01"` on each first line): the guarded api-key full-cycle run family. `-01` (2026-11-20) the first guarded cycle; `-02` (2026-11-21 morning) the completed turn with NO terminal envelope in-window and the post-completion `-32010` latch (section 5 latch subsection); `-03` (2026-11-21 evening, scrubbed) two COMPLETED streaming turns — the section 5 delta-surface subsection's wire truth (model.streaming `payload.kind` deltas `:177`/`:218`/`:425`, the tool-args trio `:364`–`:366`), the terminal surface observed live (`turn.completed` `:227`), and the full desktop-continuous envelope surface (`deliveryKind`/`eventId`/`seq`/`sessionId`/`timestamp`/`traceId`/`turnId`/`type` on every `session/event`). |
| `[V-EXP EXP-DISCOVERY-01]` | `experiments/captures/EXP-DISCOVERY-01.jsonl` (plan 01-05): pristine byte-compare of the probed bundle copy, canonical spawn line, localhost-listener decoys, per-version builtin revisions + revision-formula bundle literals and computed example. |
| `[V-SRC bundle resources/app.asar#…]` | The installed desktop 3.18.2 app bundle (`resources/app.asar`), host-side side: the `interactionRequestOfficialMcpAuthHeaders` host handler anchors the `interaction/requestOfficialMcpAuthHeaders` reply shapes (the three RESULT forms, the origin trust check, `official_auth_unavailable` as the no-resolver decline) — re-derivable by unpacking the installed asar and grepping the handler name (section 3 subsection). |
| `[V-SRC bundle resources/glm/zcode.cjs#…]` | The shipped 3.18.2 entrypoint bundle (V-SRC ground truth per D-01), probed at the unpacked-AppImage squashfs-root copy named in section 9 (pristine `cmp` vs `.orig`: identical, re-run 2026-11-19). Every anchor is re-derivable: method-name map via the section 2 extraction commands; occurrence counts via `grep -o <pattern> \| wc -l` (Pitfall 8: never `grep -c` on a minified bundle). Key anchors and observed counts: method map ×1 per string; v4 strings ×1 (`v4/conversation/frame` ×2); `state.updated` ×72; `startup/storageState` ×3; `session/lifecycle/changed` ×1; `process/mcpTelemetry` ×1; `session/event` ×3; `computer-use/operation-event` ×1; `v4/telemetry/event` ×1; `v4/conversation/contextCompositionChanged` ×1; `deliveryKind` ×61 / `desktop-continuous` ×15 / `web-remote-replayable` ×11; `workspaceKey` ×141; `reasoningLevel` ×184; error codes `-32700` ×2, `-32600` ×3, `-32601` ×5, `-32602` ×14, `-32603` ×4, `-32010` ×4, `-32020` ×8, `-32021` ×3, `-32022` ×5, `-32031` ×5; `"A prompt is already running"` ×1; `"Provider runtime headers were not applied"` ×2; `"Reasoning level is required"` ×1; `unrecognized_keys` ×81; `.strict()` ×1409; `18e4` ×5; `"session/delete"` ×0; `"turn.completed"` ×7 quoted / ×4 bare, `"turn.failed"` ×8 quoted / ×5 bare; error-code enum entries; ack `status:m.enum(["received","unchanged"])`; `sendError("invalid-message",-32600,…)` / `("parse-error",-32700,…)`. |
| `[V-SRC 3.17.1 <path>:<line>]` | Public 3.17.x source tree read at version **3.17.1** (`package.json`), local clone — corroborating only, never 3.18.2 truth (D-01). Cited anchors (all re-verified file:line 2026-11-19): `transport.ts:28` (`PROTOCOL_EOF_DRAIN_MS = 100`), `:31-58` (concurrent/immediate sets), `:163` (drain timer), `:212-281` (`dispatchLine` serial chain + bypass), `:283-296` (-32600/-32700 senders), `:305-314` (`shouldBypassProcessingQueue` = `session/stop`, `workspace/cancelGenerateText`, MCP-UI set); `server.ts:735-757` (dispatch incl. `sessionStop`, `closeSession`, `setModel`); `model-mapper.ts:4-6` (`formatProtocolModelSelection` drops `options`); `model-selection.ts:3-17,22-31` (selection schema + strict level issues); `index.ts:1979-1999` (create params), `:1916-1931` (subscribe params/result), `:2396-2403` (setModel params), `:2762,2784` (workspaceKey nonEmptyString), `:2808-2841` (requestUserInput schemas), `:4265,4381` (method constants), `:4509` (storageState notification schema), `:476` (`storageStartup` constant); `account-provider-config.ts:28-33` (ack received/unchanged); `registry-service.ts:205-212` (silent previous-snapshot retention); `provider-runtime-headers.ts:17,48-100` (180 s queue, -32031, fingerprint re-ask); `interaction-broker.ts:60,247,342-400` (1 s re-announce, dual-channel requestUserInput); `zcode-protocol-legacy-types.ts:14` (deliveryKind enum); `validation.ts:803` (clientMode enum); `zcode-builtin-provider-config-source.ts:41,241` (revision formula); `provider-data-schema.ts:43-50` (`completeZhipuAccountAccessDataSchema` — the complete zhipu-account access schema, `.strict()`), `:51-56` (the sparse probe schema `zhipuAccountAccessDataSchema`); `resolver.ts:108-115` (`createRegistryProviderConfig` — validateComplete issues → `{ok:false}`), `:244,275` (the registry loop's `providerIssues` gating `providerExecutable` — a provider with validation issues never publishes; re-verified file:line 2026-11-22). |
| `[V-RPT private field notes]` | Predecessor shim + FINDINGS private research notes (outside this repo; real locations supplied in agent prompts, never committed): jsonrpc-key rejection (-32600 unrecognized_keys), silent snapshot drop + provider-not-found turn failures, `headersApplied:false` fatal turn failure, quiet-window fallback proven end-to-end, two-step setModel flow end-to-end, EOF exit-0 behavior. Inherited facts re-recorded here as V-RPT per D-05; each is paired above with the bundle/source/live evidence that covers it in-repo. |

Assumption-mapping carryovers from plan 01-02 (both landed): the **encoding** row is stated in section 1 (string-or-int ids, UTF-8, echoes recorded in captures); the **empty/unknown-method** row is stated in section 4 (empty subscribe result is a valid result; unknown-method family -32600/-32601; empty capture = EOF-pitfall failure signature).

Path note: this document records glob/tilde/`<placeholder>` patterns only — no user-specific absolute paths (enforced by `experiments/check-docs.sh`).
