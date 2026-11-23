# ACP Client Dialect — the `omp` preset-agent contract

**Domain:** the chat-UI client's ACP v1 dialect as consumed by the `omp` preset-agent card (DISC-02)
**Verified against:** fresh `--appimage-extract` of the installed Orca.AppImage (squashfs-root),
read 2026-11-19 in chunks `agent-session-attachment-store-registry-CR_MCAgP.js`,
`structured-agent-session-host-assembly-BbqHQ74m.js`, `session-scanner-omp-subagent-transcripts-B3v1LJlD.js`,
`session-json-object-CljYgSvh.js`, `commit-message-agent-output-lRxbgzAd.js`, `omp-profile-DU6kfV0l.js`
**Confidence:** HIGH — every load-bearing claim below was re-read from that fresh extract of the
installed build (upgrade from the parallel-session V-SRC(parallel) table); the two formerly open
items are answered statically and cited to `[V-EXP EXP-SESSIONDIR-01]`; the three remaining
known-unknowns (settings-store persistence path, live end-to-end `--version`, interruption
semantics) stay V-RPT-labeled (see Open items).
**Build drift warning:** chunk hash suffixes differ per client build. This document describes the
build named in the header above; re-run the freshness greps (capture `EXP-SESSIONDIR-01`) after any
client update before relying on it further. The client source is public
(github.com/stablyai/orca, MIT): this document was cross-verified against source tag
`v1.4.262` (@`13619bc1`, 2026-11-17) — the newest release whose card registry is byte-count-identical
to the installed build's 8-card chunk registry (a 9th card, `hermes`, was merged after it and
shipped in v1.4.263; see Sources & provenance for the exact claim strength — byte-COUNT identity
is what the cross-check recorded, not content identity). The installed-build anchors remain the ground truth for what the shipped
app does; `[V-SRC repo … @v1.4.262]` tags are the corroborating source half of each claim.

Provenance tags (V-SRC / V-EXP / V-RPT) are used per the legend in
`docs/protocol/README.md`. Dialect facts about the desktop app-server side live in
`docs/protocol/app-server-dialect.md`; this file is only the client-facing half of the wire.

## Preset-agent identity

The wrapper's preset-agent identity is the **`omp` card** (the ACP-exposed card of the `pi`
agent family). The identity decision is locked (2026-11-19) and is not re-litigated here; this
build's fresh extract re-confirms the card exists and carries the contract documented below.

Why `omp` was chosen over the other ACP cards [V-RPT parallel-session decision table, locked]:

- The `omp` dialect ("Hb") is built on **string contracts** — model switching is a slash prompt,
  errors are matched by exact string prefixes, session loss by two fixed strings. String
  contracts are trivial to implement and test in Go.
- The card's launch command is **replaceable** via the official per-agent command override
  (see Launch contract) — no binary named `omp` is needed in PATH.
- The `--version` probe is simple: one line of semver on stdout within a 5 s window.

Alternatives ruled out, with the reasons verified against this build:

- **antigravity** — the only card registered with `ignoreAgentCommandSetting:!0`; the override
  setting is ignored for it (`e.ignoreAgentCommandSetting?{}:n.commandSettings??{}`), so the
  launch command cannot be replaced [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#tD/ore;
  repo src/main/antigravity/antigravity-acp-launch-spec.ts#60 + src/main/acp/launch/acp-launch-command.ts#35 @v1.4.262].
- **copilot** — `modelDiscovery:{kind:\`unavailable\`,reason:\`Copilot lists the models a plan
  allows only inside a session\`}` — a dead pre-session model picker [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#copilot-card;
  repo src/main/copilot/copilot-acp-launch-spec.ts#45-46 @v1.4.262].
- **grok** — heavy private line dialect: the card's dialect carries the `x.ai/*` method family
  (`x.ai/ask`, `x.ai/session`, `x.ai/task`, `x.ai/mcp`, …) instead of plain ACP semantics
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#grok-dialect-Eb;
  repo src/main/acp/acp-dialects/grok-dialect.ts#45,132-162 @v1.4.262].
- **opencode / opencode2** — occupied by the real upstream tools plus vendored child-session
  extensions (`opencode/child-session-updates` update channel, opencode2 pinned `2.0.14`,
  opencode model discovery switching by version below `2.0.0`) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#opencode-cards;
  repo src/main/acp/acp-dialects/opencode-child-sessions.ts#8 + src/main/acp/launch/acp-launch-registry.ts#70-71,107,116-118 @v1.4.262].
- **kimi** — its own session-discovery path and its own dialect; possible conflict surface,
  no advantage over `omp` for this project [V-RPT parallel-session rationale].

## Card registry of this build

The build described in the header carries:

- **ACP preset cards (the chat-UI card registry array, 8 entries)**: `grok`, `opencode`,
  `opencode2`, `omp`, the antigravity host-backend variant, `copilot`, `kimi`,
  `cursor` (command `cursor-agent`) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#var-dD].
- **Native-chat handles (not ACP cards)**: `pi` (in-host, transport `jsonl-rpc`, account home
  env `PI_CODING_AGENT_DIR`), `claude` and `codex` (host transports, account homes
  `CLAUDE_CONFIG_DIR` / `CODEX_HOME`) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#GO-LO-RD].
- **Operations-only agent ids** (managed-hooks install/status enumeration against remote agent
  configs — never chat cards): `amp`, `codebuddy`, `devin`, `droid`, `dsh`, `gemini`, `jcode`,
  `kiro`, `muse`, `qoder`, `openclaude`, `hermes`, `zcode`, … [V-SRC managed-agent-hook-controls-Nvpuw5Qw.js#agent-status].
- **Session-history scan canon (25 ids)**: `claude, codebuddy, qoder, codex, hermes, pi, omp,
  prime-agent, cursor, gemini, antigravity, rovo, copilot, opencode, opencode2, zcode, grok,
  openclaw, devin, droid, cline, kimi, muse, jcode, kiro` — with display names
  (`omp` → `OMP`, `zcode` → `ZCode`) [V-SRC session-json-object-CljYgSvh.js#canon-list].

**Delta vs the earlier 8-card list** (compiled from a different build, see
`.planning/research/REFERENCE-ADAPTERS.md`): the ACP card registry itself is **unchanged at 8
cards**; the build delta the earlier research spotted is in the *wider* id surface — the
25-id scan canon and the operations-only id families — not in the chat-card registry.

**Is there a native `zcode` preset card? No.** The id `zcode` appears only as an
operations-side hooks-installation target and in the scan canon; no card launches a `zcode`
command, and `command:\`zcode\`` occurs nowhere in the build [V-SRC
managed-agent-hook-controls-Nvpuw5Qw.js#zcode-hooks + session-json-object-CljYgSvh.js; repo
src/main/agent-hooks/installation/managed-agent-hook-registry.ts#64,89,115,145 +
src/shared/ai-vault/contracts/ai-vault-types.ts#9-33 @v1.4.262 — the id exists only in the
hooks registry and the history-scan canon (plus the TUI/native-chat catalog, not ACP)]. The
`omp` card therefore remains the only viable preset-agent identity for an ACP server that must
present itself as a preset agent; the identity decision is unaffected.

**Source cross-check (2026-11-19).** The chunk registry maps to
`ACP_LAUNCH_SPECS` (repo `src/main/acp/launch/acp-launch-registry.ts#155-164 @v1.4.262`):
same 8 entries, same ids, same omp block (env `PI_NO_PTY:'1'` #133, `launchesWithModel` #130,
`minimumVersions:['17.0.5']` #141, `modelDiscovery` kind `session` with
`sessionFolderArgs:['--session-dir',folder]` #143-147, `compaction`/`compactionInstructions`
#149-150). Build drift to watch: source **v1.4.263 adds a 9th ACP card, `hermes`**
(`HERMES_LAUNCH_SPEC`, merge #42895 on 2026-11-17T21:13Z); the installed build's chunk registry
still has 8, so any client update past v1.4.262 also ships it. The 25-id scan canon is
byte-identical to `AI_VAULT_AGENTS` (repo `src/shared/ai-vault/contracts/ai-vault-types.ts#9-33`,
labels #66-88 — `omp:'OMP'`, `zcode:'ZCode'`).

## Launch contract

> **CORRECTED against source (2026-11-19, supersedes the chunk-only reading; feeds Phases 4–7):**
> the chunk assembly that passes `--session-dir` belongs to the client's **disposable
> command/model-discovery sessions**, not to the chat launch. Source
> `src/main/acp/launch/acp-structured-launch-resolution.ts#200-207 @v1.4.262` assembles the chat
> argv as `[...prefixArgs, ...spec.args({fullAccess, pluginDir, model})]` — no session-folder
> args, and this holds across the whole installed-build window (v1.4.259–v1.4.262 have zero
> `sessionFolderArgs` hits in that file). The chunk literal the original reading quoted even
> carries the discovery path's tell: `fullAccess:!1` is hardcoded there
> (`src/main/acp/discovery/acp-at-rest-commands.ts#186 @v1.4.262`), whereas the chat path's
> `fullAccess` is the user's permission-mode choice (`#193`). Wrapper consequences: (a) the
> chat argv is `<program> [prefixArgs] acp [--plugin-dir <dir>] [--model <id>]`; (b) the
> wrapper must still **accept and ignore `--session-dir`** because the client launches a second,
> disposable card process for command/model discovery with it appended; (c) never depend on a
> session dir existing.

The client launches the card as a stdio subprocess (no PTY, no TUI — the client renders the
whole UI). Effective argv for the `omp` card under the default registration:

- **Chat launch** (the card process the user talks to):
  `<program> acp [--plugin-dir <dir>] [--model <id>]`
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#omp-card + structured-agent-session-host-assembly-BbqHQ74m.js#argv-assembly;
  repo src/main/acp/launch/acp-structured-launch-resolution.ts#200-207 + src/main/acp/session/acp-structured-acquire.ts#83-89 @v1.4.262].
- **Disposable discovery launches** (separate card processes, same program/env/prefixArgs):
  `<program> acp [--plugin-dir <dir>] --session-dir <folder>` with `fullAccess` hardcoded
  `false`; the folder is `<client dataDirectory>/acp-command-sessions/omp` for the at-rest
  command catalog (pre-created via `acpRunFolder`, removed after the scan) and
  `<client dataDirectory>/acp-listing-sessions/omp` for model-catalog probes
  [V-SRC structured-agent-session-host-assembly-BbqHQ74m.js#acp-command-sessions;
  repo src/main/acp/discovery/acp-at-rest-commands.ts#172-188 + src/main/acp/discovery/acp-model-catalog-probe.ts#125-131,156-168 @v1.4.262].
- `<program>` is `omp` by default, or the **per-agent command override** (below) resolved on PATH.
- `--plugin-dir` is appended only when a visuals skill is configured; `--model <id>` only when a
  model was picked in the client UI (`launchesWithModel:!0` — the picked model is applied at
  session start) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#omp-card + structured-agent-session-host-assembly-BbqHQ74m.js#argv-assembly;
  repo src/main/acp/launch/acp-launch-registry.ts#125-131 + acp-structured-launch-resolution.ts#199 @v1.4.262].
- `fullAccess`: discovery sessions always `false` (above); **chat sessions follow the user's
  permission mode** — default Ask keeps `false` and permissions flow through
  `session/request_permission`; a Full-access chat spawns with `true` and the client then
  auto-approves `allow_once` options instead of prompting. For `omp` the argv is unaffected
  either way (its `args()` ignores `fullAccess`) [V-SRC structured-agent-session-host-assembly-BbqHQ74m.js#fullAccess-false;
  repo src/main/acp/launch/acp-structured-launch-resolution.ts#193 + src/main/acp/session/acp-structured-acquire.ts#115-124,159-161 @v1.4.262].
- Environment: the client sets **`PI_NO_PTY=1`** for the omp card [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#env-PI_NO_PTY; repo
  src/main/acp/launch/acp-launch-registry.ts#133 @v1.4.262 — the source comment explains why:
  answering forms marks the session as having a UI, so `pty:true` bash would open an
  undrawable TUI overlay]. Ignore it — ACP is a pipe protocol; never allocate a PTY.
- Agent home: the card's account home env is `PI_CODING_AGENT_DIR`; the client derives
  `<PI_CODING_AGENT_DIR>/.omp/agent` as the agent home (sessions live at
  `<agent-home>/sessions`; default `~/.omp/agent/sessions`) [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#account-home + session-scanner-omp-subagent-transcripts-B3v1LJlD.js#sessions-map;
  repo src/main/acp/launch/acp-launch-registry.ts#136-139 + src/main/ai-vault/discovery/session-scanner-roots.ts#17 +
  src/main/ai-vault/scanners/omp/omp-session-root.ts#20-69 @v1.4.262].

**Per-agent command override semantics** (the official wiring used by this project:
`{"omp": "<wrapper> omp"}`):

- The override is `record<agentId, string>` in the client's settings store, siblings
  `agentDefaultArgs` / `agentDefaultEnv` / `agentPermissionOverrides` [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#settings-keys; repo
  src/main/native-chat/agent-launch/structured-agent-command-resolution.ts#42-56 +
  src/shared/agent-launch/choices/agent-launch-settings.ts#65 @v1.4.262].
- **Prefix arguments are allowed**: the string is shell-quoted-tokenized, the first token is the
  program, the rest are prefix args prepended before the card's own arguments [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#override-parse; repo
  src/shared/agent-launch/startup/tui-agent-launch-command-override.ts#6-26 +
  src/shared/agent-launch/startup/agent-command-plan.ts#7-41 @v1.4.262].
- The parse rejects: empty strings, embedded NUL/CR/LF, `$`, backtick, any shell metacharacter
  (`| & ; < > ( ) { } [ ] * ? !`) in the program or any prefix arg, and env-assignment prefixes
  (`VAR=val cmd` is invalid) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#metachar-rejection;
  repo tui-agent-launch-command-override.ts#8,17-23 @v1.4.262 — the env-assignment form is
  extracted by the tokenizer and a non-empty extraction invalidates the plan (`parsed.env`)].
- `~/`-prefixed args are expanded against the user home at resolution time [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#tilde-expansion; repo
  src/main/native-chat/agent-launch/structured-agent-command-resolution.ts#67-71 @v1.4.262].
- A non-empty override **satisfies the installed-detection** check — no binary named `omp` is
  needed in PATH [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#mD-installed-detection;
  repo tui-agent-launch-command-override.ts#29-34 (`Boolean(overrides[agent]?.trim())`) +
  structured-agent-command-resolution.ts#90-108 @v1.4.262].

## Version floor

- Card floor: **`17.0.5`** (`minimumVersions:["17.0.5"]` on the omp card) [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#minimumVersions; repo
  src/main/acp/launch/acp-launch-registry.ts#141 @v1.4.262 — comment: "First release verified to
  serve ACP"].
- Probe mechanics: the client runs `<program> <prefixArgs...> --version` with a 5 s timeout and a
  4096-byte output cap; parse failures yield `unknown`, non-start yields `missing`, a version
  below the applicable floor yields `tooOld`; `ready` and `unknown` are both treated as usable
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#probe-XO-qO-JO; repo
  src/main/agent-launch/preflight/agent-cli-version-probe.ts#39-48,72,78,91-92 +
  src/main/acp/launch/acp-launch-version.ts#11-36 @v1.4.262]. Source adds two precision points:
  non-zero exit / output-cap overrun / timeout also degrade to `unknown` (probe #101-109), and
  the applicable floor is the last one whose major is `<=` the probed major, else the first
  (`findLast`, #60-65 — matching the chunk).
- Reported-version policy: the Hb dialect declares **`resolvesSelectorsFrom:"18.1.9"`** — model
  selectors (named model arguments in the `/model` flow) resolve only from that version
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#Hb-modelCommand; repo
  src/main/acp/acp-dialects/omp-dialect.ts#145 @v1.4.262 — comment: 18.1.8's `/model` matches
  ids, 18.1.9's resolves roles]. The wrapper therefore prints **18.1.9+** on `--version`, not
  the bare floor (locked integration decision [V-RPT PROJECT.md key decisions, 2026-11-19]).
- Probe shape under prefixed overrides — the static question from research assumption A2 is now
  answered by the chunk read above: the probe is `prefixArgs + --version`, **without the card's
  own args**, so an override `<wrapper> omp` is probed as `<wrapper> omp --version`. Our binary
  must answer `--version` under the `omp` subcommand (the cobra root+subcommand decision covers
  this). Recorded here as an upgrade of the plan's V-RPT expectation — mechanics verified
  statically [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#probe-XO; repo
  src/main/agent-launch/preflight/agent-cli-version-probe.ts#78 @v1.4.262]; the
  end-to-end live probe against the real wrapper remains a Phase 7 item [V-RPT Phase 7 plan].

## Load-bearing method set

Observed call sequence, split by phase (methods sent both at startup and later are listed in
both sets with a cross-reference).

At card startup (per launched card process), in order:

1. `initialize` — the client sends `protocolVersion: 1` and its capabilities; a non-1 reply is
   rejected with -32602 (`Unsupported ACP protocol version: <v>`) and the connection closed
   [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#initialize-call; repo
   src/main/acp/protocol/acp-session-runtime.ts#89-119 + generated/acp-protocol.generated.ts#200
   (`ACP_PROTOCOL_VERSION = 1`) @v1.4.262].
2. `session/new` — with `cwd` and `mcpServers:[]` defaults; the sessionId returned by the card
   becomes the chat's handle [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#session-new;
   repo src/main/acp/session/start/acp-structured-handshake.ts#163 +
   src/main/acp/protocol/acp-session-setup.ts#58-62 @v1.4.262]. Source precision the wrapper
   must know: the client **extends** the card's `session/new` response with an optional
   `models` field (SessionModelState: `availableModels:[{modelId,name,description?}]` +
   `currentModelId`) — publishing it drives the model picker and switches via
   `session/set_model`; omitting it routes model changes through the `/model` slash prompt
   instead (see String contracts and the modelDiscovery correction in Open items).
   (Overlap note: `session/new` also recurs whenever the user opens a *new* chat on an already
   running card — same method, later in time; see the per-turn set.)
3. `authenticate` — only when the card advertises auth methods at initialize, and **only
   error-driven**: sent after a session-establishment call is refused as auth-required, with the
   matching advertised `methodId`, then the establishment call retries; the omp flow for
   this project does not use it (auth lives in the desktop credential store, not the card)
   [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#authenticate; repo
   src/main/acp/protocol/acp-session-setup.ts#93-119 @v1.4.262].
4. `session/load` / `session/resume` — **only** when the card advertised `loadSession` or a
   resume capability at initialize; otherwise the client throws
   `-32601 "ACP agent cannot load or resume this session"` (see Open items) [V-SRC
   agent-session-attachment-store-registry-CR_MCAgP.js#session-establishment; repo
   src/main/acp/protocol/acp-session-setup.ts#64-91 @v1.4.262].

Per turn / per user action, on the same process:

- `session/prompt` — one call per user turn, run through the client's busy-retry wrapper
  (`retryBusyPrompts` is enabled specifically for `omp`). Source precision (load-bearing for
  the wrapper): a retry happens **only** when the card's prompt error is code `-32003` with
  `data.reason === 'session_busy'`; the retry delay starts at 1 s and doubles to a 5 s cap; a
  client cancel during the wait resolves the turn as `stopReason:'cancelled'` — no other error
  shape triggers a re-send [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#session-prompt + structured-agent-session-host-assembly-BbqHQ74m.js#retryBusyPrompts-omp;
  repo src/main/acp/session/acp-structured-acquire.ts#108 + src/main/acp/protocol/acp-prompt-request.ts#6-16,28-58 @v1.4.262].
- `session/cancel` — user interrupt, sent as a JSON-RPC **notification** with the sessionId
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#cancel; repo
  src/main/acp/protocol/acp-session-runtime.ts#180-190 @v1.4.262].
- `session/set_model` — the client's model-change call; for `omp` the picker itself is
  in-session (see Open items; the write priority is config option → `session/set_model` →
  `/model` slash prompt) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#set-model; repo
  src/main/acp/protocol/acp-session-runtime.ts#205-222 + src/main/acp/options/acp-structured-options.ts#230-238 +
  src/main/acp/turns/acp-message-execution-options.ts#42-58 @v1.4.262].
- `session/set_mode`, `session/set_config_option` — mode/config changes on user action [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#set-mode-config; repo
  src/main/acp/protocol/acp-session-runtime.ts#192-195,223-242 @v1.4.262].
- `session/new` — recurs on new chats (overlap with the startup set, listed in both).
- Session management ops when triggered: `session/list` (session listing support advertised;
  used on reopen-failure re-listing), `session/fork` (fork support advertised; conversation
  copy), `session/delete` (deletion support advertised; also the client's own cleanup of its
  disposable listing sessions — omp does not advertise delete, so the client uses throwaway
  folders instead) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#session-management;
  repo src/main/acp/process/acp-agent-connection.ts#257-260 + src/main/acp/protocol/acp-session-runtime.ts#139-146 +
  src/main/acp/session/start/acp-session-reopen-failure.ts#75 + src/main/acp/conversation-copy/acp-conversation-copy.ts#174 +
  src/main/acp/discovery/acp-listing-session-models.ts#74-87 @v1.4.262].

Ordering honesty: the startup order initialize → session/new is the observed code order (the
connection initializes before any session call can reference it) and is load-bearing; the order
among per-turn methods is user-action driven, not fixed.

## Full ACP method range the client sends

Complete call-site inventory of this build's ACP client stack [V-SRC
agent-session-attachment-store-registry-CR_MCAgP.js#method-literal-counts; re-derived from
source v1.4.262 (repo sweep over src/main/acp/** call sites) — **two methods the chunk-only
table missed are marked NEW**]:

| Direction | Method | Notes |
|-----------|--------|-------|
| client → agent | `initialize` | protocolVersion 1, handshake |
| client → agent | `session/new` | fresh session; response may carry the `models` extension |
| client → agent | `session/load` | only if `loadSession` capability advertised |
| client → agent | `session/resume` | only if resume capability advertised |
| client → agent | `session/prompt` | one per turn; busy-retry for omp (-32003 `session_busy`) |
| client → agent | `session/cancel` | interrupt (notification) |
| client → agent | `session/set_mode` | mode change |
| client → agent | `session/set_model` | model change |
| client → agent | `session/set_config_option` | config options |
| client → agent | `session/list` | session enumeration (advertised support; reopen re-listing) |
| client → agent | `session/fork` | fork a session (advertised support; conversation copy) |
| client → agent | `session/delete` | delete a session (advertised support; listing-session cleanup) |
| client → agent | `session/close` | **NEW** — session teardown on Stop/child release, 10 s timeout, timeout cancels the request (repo src/main/acp/process/acp-agent-connection.ts#75-76 + src/main/acp/turns/acp-structured-child-stop.ts#97,116 @v1.4.262) |
| client → agent | `authenticate` | only if auth methods advertised; error-driven (see method set above) |
| client → agent | `_extension` methods | require a leading underscore, 30 s default timeout; a session-free discovery variant sends non-`session/` extensions with no timeout |
| agent → client | `session/update` | streaming updates (notifications) |
| agent → client | `session/request_permission` | permission prompts; auto-answered `allow_once` under chat full-access |
| agent → client | `elicitation/create` | form-mode questions (`{mode:"form",message,requestedSchema}`); chats also accept `{mode:"url",message,elicitationId,url}` sign-in requests |
| agent → client | `elicitation/complete` | **NEW** — notification that finishes an open URL elicitation by `elicitationId` (repo src/main/acp/requests/acp-url-elicitation.ts#117-132 + src/main/acp/timeline/acp-timeline-translator.ts#170 @v1.4.262) |

`session/close` and the url-elicitation pair exist in the source call graph of the matching
release window; they were not visible as call-site literals in the single chunk the original
pass counted. Any wrapper implementation should tolerate all of the above. The chunk counts
(occurrences of each literal in the registry chunk) remain as recorded in
`experiments/captures/EXP-SESSIONDIR-01.jsonl`.

The client advertises `fs.readTextFile/writeTextFile:false` and `terminal:false`, so it never
accepts `fs/*` or `terminal/*` requests — a card must not send them; at the session layer an
unhandled client-request method is answered with -32601 (`Unsupported ACP client method:
<method>`) and unknown session ids on requests are answered with -32602 / a `cancelled` outcome
(see Client capabilities; repo src/main/acp/protocol/acp-session-runtime.ts#278-295 +
src/main/acp/requests/acp-structured-prompts.ts @v1.4.262).

## String contracts

Exact byte-for-byte strings (equality means exact string match after the prefix-match form
shown; no normalization) of the Hb dialect [V-SRC
agent-session-attachment-store-registry-CR_MCAgP.js#Hb-block; all confirmed in repo
src/main/acp/acp-dialects/omp-dialect.ts @v1.4.262 — prefixes #11,141,143, equality strings
#12,145-146,160, matcher wiring #139-162]:

- Model switch: the client sends the slash **prompt** `/model <name>` as a `session/prompt`
  turn (`text:e=>\`/model ${e}\``) — model switching is a prompt convention, not a method.
  Source precision: the prompt is the **fallback path** — if the card published model state in
  its `session/new` response, the client switches via `session/set_config_option` /
  `session/set_model` first, and also retries via the slash prompt when a native write is
  refused with the `Unknown ACP model: ` prefix (repo
  src/main/acp/options/acp-structured-options.ts#230-238 +
  src/main/acp/turns/acp-message-execution-options.ts#42-58 @v1.4.262). A wrapper that omits
  the `models` field keeps this contract on the prompt path exclusively.
- Model refused: the card's error detail starts with `Unknown ACP model: ` — matched by
  `startsWith` (`refusedByOption`).
- No model: the error detail starts with `No model selected.` (`noModel`).
- Session lost: the error detail equals `ACP cancel cleanup timed out` **or** equals
  `Unsupported ACP session: <sessionId>` (`sessionLost` matcher; the second form is built by
  interpolating the session id; source comment: a timed-out cancel cleanup closes the session,
  later prompts get the second string).
- Authentication-required: the error detail starts with `No API key found for ` — that is the
  *card's own* credential model; not applicable to this project (auth lives in the desktop
  credential store; the wrapper never emits this string).
- Model-selector resolution floor: `resolvesSelectorsFrom:"18.1.9"` (see Version floor).

Wire detail all of the above depend on: these matchers read the error **detail** extracted only
from JSON-RPC errors with **code -32603** whose `data.details` is a string — a refusal carried
in any other code or field is invisible to them (repo
src/main/acp/acp-dialects/omp-dialect.ts#126-130 @v1.4.262). The wrapper must emit its refusals
as -32603 with `data.details` set.

## Client capabilities

What the client advertises at `initialize` and how the card must behave [V-SRC
agent-session-attachment-store-registry-CR_MCAgP.js#initialize-call + Hb clientCapabilities;
repo src/main/acp/protocol/acp-session-runtime.ts#94-98 + src/main/acp/acp-dialects/omp-dialect.ts#136 +
src/main/acp/session/acp-structured-acquire.ts#112 @v1.4.262]:

- `fs: {readTextFile:false, writeTextFile:false}` and `terminal:false` — **never call client
  fs/terminal**; the card reads and writes only its own side of the wire and its own process
  resources (unhandled requests get -32601; see the method-range section).
- `elicitation: {form:{}, url:{}}` — the dialect declares `form`; the chat layer **adds `url`**
  for chat connections (discovery connections omit it), so a card may ask form-mode questions
  (`elicitation/create` with `{mode:"form", message, requestedSchema}`; malformed forms get
  -32602 `Unsupported ACP elicitation form`) and URL sign-in requests
  (`{mode:"url", message, elicitationId, url}`, answered accept/decline/cancel), finishing them
  with an `elicitation/complete` notification (repo
  src/main/acp/acp-dialects/omp-dialect.ts#136 + src/main/acp/session/acp-structured-acquire.ts#112 +
  src/main/acp/requests/acp-url-elicitation.ts#36-38,44-84 @v1.4.262).
- Compaction: the omp card sets `compaction:!0, compactionInstructions:!0`; the client offers
  compaction (with instructions) to the chat — `/compact` is enabled **as a prompt command**
  and `compactionInstructions` means the card's `/compact` honours text after the command — and
  the Hb dialect carries a `compactionReply` handler that classifies the card's reply text
  (`compaction failed …` → failed/skipped outcomes) (repo
  src/main/acp/launch/acp-launch-contract.ts#61-64 + src/main/acp/acp-dialects/omp-dialect.ts#96-108 @v1.4.262).
- `retryBusyPrompts` is enabled for `omp` — the client retries `session/prompt` **only** on the
  -32003 `session_busy` refusal, 1 s doubling to a 5 s cap (see Load-bearing method set).
- Card-side capability set the client derives for `omp`: `rewind:false`, `fork:"tip"` (transcript
  import at the fork tip), `compact:true`, `compactInstructions:true`, `threadGoal:false`,
  `contextUsage:true`, `imagePrompts:true`, `steering:"queue"`,
  `approvalEnforcement:"orca"` [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#fD-capabilities;
  repo src/main/acp/session/acp-structured-agent-definitions.ts#18-51 @v1.4.262 — byte-equal
  derivation, incl. the omp/opencode/opencode2 fork-tip condition].
- `clientInfo` the client sends: `{name:"orca", version:"1"}` [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#QS-clientInfo; repo
  src/main/acp/launch/acp-client-info.ts#4 @v1.4.262 — the default; a spec may override it by
  launched version, omp does not].
- No PTY anywhere: the client spawns the card with stdio pipes and sets `PI_NO_PTY=1`; the TUI
  layer of the underlying agent is never used.

## Open items

Two items were formally open after the parallel-session pass; both are now answered statically
and captured in `experiments/captures/EXP-SESSIONDIR-01.jsonl` [V-EXP EXP-SESSIONDIR-01], and
both were re-checked against the public client source on 2026-11-19 (item 2's interpretation
corrected — see below):

1. **`--session-dir` session-file format.** The client pre-creates
   `<dataDir>/acp-command-sessions/omp` and passes it via `--session-dir` (on the disposable
   discovery launch; chat launches carry no `--session-dir` — see Launch contract); the scanner
   side of the client reads omp history from the agent-home sessions dir
   (`~/.omp/agent/sessions`, env-overridable via `OMP_CODING_AGENT_DIR`/`PI_CODING_AGENT_DIR`,
   profile and `XDG_DATA_HOME` variants). Format, fully specified [V-EXP EXP-SESSIONDIR-01]:
   - Layout: `<sessions-dir>/<path-slug>/<timestamp>_<uuid>.jsonl`
   - Path slug (writer algorithm, verified byte-for-byte against a real local agent home):
     `--` + path with one leading `/` stripped, trailing slashes stripped (when length > 1),
     every `/`, `\`, `:` replaced by `-`, then `--` (NFC-variant added when it differs). Note
     only `/`, `\`, `:` are replaced — other non-alphanumerics survive.
   - File name: `YYYY-MM-DDTHH-MM-SS-mmmZ` (dashes for colons) + `_` + hex uuid
     (8-4-4-4-12) + `.jsonl`, matched case-insensitively.
   - Records: Claude-Code-style JSONL — `type`/`version`/`id`/`timestamp`/`cwd` header fields;
     tool use records (`type:"tool_use"` or `{id,name,input}`), tool results
     (`type:"tool_result"` or `{tool_use_id}`), `is_error`/`*_error` content, string or
     `{text}`/`{content}` content items, `user`/`assistant` roles; ISO or epoch timestamps.
     Subagent transcripts are sibling `.jsonl` files in the same slug dir. Source precision
     (scanner side, v1.4.262): each transcript may own an **artifact directory named after its
     stem** (`<sessions-dir>/<slug>/<stem>/`) holding the subagent `.jsonl` files as direct
     children (nested grandchildren excluded), and the pi/omp record consumer reads
     `type:"session"` (id + cwd), `type:"model_change"` (`modelId`/`model`) and
     `type:"message"` (`message.role`/`content`/`model`/`usage`) records, with
     OMP-written detection via `type:"title"` v1 / a `runtime` field / `model_change` with
     `model` but no `modelId` (repo
     src/main/ai-vault/scanners/omp/session-scanner-omp-subagent-transcripts.ts#12-14,28-31 +
     src/main/ai-vault/scanners/session-scanner-graph-parsers.ts#209-259 +
     src/main/ai-vault/scanners/omp/session-scanner-omp-written-record.ts @v1.4.262).
   - Source cross-check on the slug: the doc's writer algorithm is byte-equal to
     `PI_CWD_BUCKET_LAYOUT` (`encodePiSessionDirPrefix`: `--` + strip trailing slashes when
     length > 1, strip one leading `/`/`\`, replace `/ \ :` with `-`; NFC variant added when it
     differs) with the caller appending the closing `--`; the id matcher is
     `<stem> === id || stem.endsWith('_' + id)` on case-insensitive `.jsonl` entries (repo
     src/main/ai-vault/scope/session-cwd-bucket-layouts.ts#22-50 +
     src/main/native-chat/transcript-discovery/session-file-id-layouts.ts#35-42 @v1.4.262).
     The client's find-by-cwd additionally tolerates three more bucket families for newer OMP
     layouts — home-relative `-<path>`, temp `-tmp<…>`, and the OMP 17.2.5–17.2.8 digest form
     `<scope>-<readable>-<sha256(canonical cwd)>` — so a writer emitting the documented legacy
     `--<slug>--` form stays resolvable (repo
     src/main/omp/omp-session-cwd-lookup.ts#11-48 @v1.4.262).
   - **Verdict: Yes** — a Go agent can write omp-compatible history files. Scope honesty:
     our v1 does **not** advertise `loadSession`, so the client cannot load these sessions
     interactively (it throws `-32601 "ACP agent cannot load or resume this session"` on the
     resume path [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#session-establishment]);
     compatible files are a Phase 7 nicety (history listing / fork import), not a v1 gate.
2. **`modelDiscovery` kind `session`.** The omp card block is
   `modelDiscovery:{kind:"session",listingNamesConfiguredModel:false,sessionFolderArgs:["--session-dir", …]}`.
   > **CORRECTED against source (2026-11-19; supersedes the chunk-only interpretation):**
   > `kind:"session"` does **not** mean "nothing is listed until the user's session" — it means
   > the model list is read **inside an ACP session the client launches for that purpose**.
   > Source: the model-catalog probe spawns a disposable card process
   > (`<program> acp --session-dir <dataDir>/acp-listing-sessions/omp`, `fullAccess:false`),
   > initializes, `session/new`s, and reads the **`models` state in the `session/new`
   > response**; when the card advertises session deletion the client deletes that listing
   > session, otherwise (omp) it relies on the throwaway folder; folderless workspaces are
   > refused (`<agent> lists in a chat's folder`) [V-EXP EXP-SESSIONDIR-01; repo
   > src/main/acp/discovery/acp-model-catalog-probe.ts#152-168,237-238 +
   > src/main/acp/discovery/acp-listing-session-models.ts#53-96 +
   > src/main/acp/discovery/acp-model-discovery-policy.ts#29-40 @v1.4.262].
   > `listingNamesConfiguredModel:false` is a picker-semantics flag (the listing does not name
   > the configured model as the CLI default), not a "no list is rendered" statement. In the
   > user's chat the same mechanics fill the picker from the chat's own `session/new` response.
   > The `/model` slash prompt remains the in-chat **switch** mechanism (with
   > `session/set_model`/config preferred when model state was published; selectors resolve
   > only from version 18.1.9) — see String contracts. Wrapper decision input for Phases 4–7:
   > publishing no `models` field keeps the picker catalog-driven via these probes while
   > switches go through `/model` prompts exclusively.

Still unresolved, recorded as known-unknowns (never presented as verified):

- **Settings-store location for `agentCmdOverrides`** — not found in `~/.config/orca/*.json`;
  suspected Electron storage/orcad. Configuration happens through the Settings UI. The source
  cross-check confirms the settings **key** and shape (`agentCmdOverrides: record<agentId,
  string>` on GlobalSettings, read live per lookup) but not the persistence path, so the
  storage question stays open [V-RPT parallel-session check + research assumption A4; repo
  src/shared/agent-launch/choices/agent-launch-settings.ts#65,130,240-246 +
  src/main/native-chat/agent-launch/structured-agent-command-resolution.ts#83 @v1.4.262] —
  Phase 7 live concern.
- **End-to-end `--version` probe against the real wrapper** — the static mechanics are now
  source-verified (see Version floor); the live half (parse tolerance, real output capture)
  stays [V-RPT Phase 7 plan].
- **Interruption semantics beyond the string contracts** — cancel/interrupt behavior beyond the
  `sessionLost` strings and `retryBusyPrompts` is a Phase 7 live-test concern, not assumed from
  source [V-RPT Phase 7 plan].

## Sources & provenance

- Legend and tag semantics: `docs/protocol/README.md` (single definition site).
- `[V-SRC <chunk>#<anchor>]` — read from the fresh extract of the installed client build named
  in the header (squashfs-root `resources/app.asar.unpacked/out/main/chunks/`); anchors are
  minified identifiers or nearby string literals. Chunks cited:
  - `agent-session-attachment-store-registry-CR_MCAgP.js` — card registry, omp card block, Hb
    dialect, override parse/probe, session-establishment, initialize/call sites, clientInfo.
  - `structured-agent-session-host-assembly-BbqHQ74m.js` — argv assembly,
    `acp-command-sessions` pre-creation, `retryBusyPrompts` wiring for omp.
  - `session-scanner-omp-subagent-transcripts-B3v1LJlD.js` — sessions-dir map and file-name
    pattern.
  - `session-json-object-CljYgSvh.js` — scan canon list, display names, transcript record shapes.
  - `commit-message-agent-output-lRxbgzAd.js` — per-agent scope-dir (slug) encoders (pi/omp).
    Source note: at v1.4.262 the pi/omp encoder lives in
    `src/main/ai-vault/scope/session-cwd-bucket-layouts.ts` (`PI_CWD_BUCKET_LAYOUT`);
    `src/shared/source-control/ai/generation/commit-message-agent-output.ts` is the jcode
    envelope parser — the chunk name bundles several modules, cite the bucket-layout file for
    the slug algorithm.
  - `managed-agent-hook-controls-Nvpuw5Qw.js` — operations-only agent ids, `zcode` hooks target.
- `[V-EXP EXP-SESSIONDIR-01]` — `experiments/captures/EXP-SESSIONDIR-01.jsonl` (E5 static
  answers + re-verification evidence; produced by the scrub-first pipeline, raw capture outside
  the repo).
- `[V-SRC repo src/<path>#<line> @v1.4.262]` — the public client source
  (github.com/stablyai/orca, MIT), read at tag `v1.4.262` (commit `13619bc1`,
  2026-11-17T22:43:33Z). Capture evidence: `[V-EXP EXP-ORCASRC-01]` —
  `experiments/captures/EXP-ORCASRC-01.jsonl` (IN-03: tagged in V-EXP form so the
  check-docs resolution + capture-header continuity gate covers this capture like
  the other five). Build match: the installed AppImage
  (mtime 2026-11-18) has no exactly provable tag without rebuilding, but its chunk registry
  (8 cards, kimi and cursor present, no hermes) is byte-count-identical to v1.4.261 and
  v1.4.262, and brackets the build between the cursor merge (2026-11-15) and the hermes merge
  (#42895, 2026-11-17T21:13Z, first shipped in v1.4.263) — so v1.4.262 (the newest
  byte-count-identical release) is used as the citation state; every repo file cited here except
  `src/main/acp/launch/acp-launch-registry.ts` is byte-identical between v1.4.262 and
  `main` @`e1d344b` (2026-11-19), and the registry differs only by the added `hermes` card.
  Per the legend these repo tags corroborate the installed-build V-SRC anchors; where source
  contradicted a chunk-derived interpretation (launch argv `--session-dir`, `fullAccess`,
  `modelDiscovery` semantics) the repo evidence is decisive and the correction is flagged in
  place.
- `[V-RPT …]` — parallel-session compiled table (`.planning/research/REFERENCE-ADAPTERS.md`),
  PROJECT.md key decisions, and Phase 7 plan items; not verified in this phase.
- The packed `app.asar` was never read; all reads went through the unpacked chunk files
  (anti-pattern guard from the phase research).

---
*Phase: 1 — Research & Protocol Documentation (DISC-02)*
*Verified against the installed client build on 2026-11-19*

---

## The slash-command catalog surface (2026-11-23, V-SRC + wire-truth read of the installed build)

How the card's composer autocomplete gets its commands ("No matching commands" = the
agent never sent the catalog update):

- **The discovery session**: a disposable card process
  `<program> acp [--plugin-dir …] --session-dir <dataDirectory>/acp-command-sessions/<agent>`
  (fullAccess hardcoded false) — `acp-at-rest-commands.ts` drives it: `initialize` →
  `session/new` (`start({cwd, mcpServers: []})`) → wait ≤9 s (`scanTimeoutMs`) for ONE
  `session/update` notification whose `update.sessionUpdate === "available_commands_update"`
  carrying `update.availableCommands` → teardown. If `agentCapabilities.sessionCapabilities.delete`
  is truthy the client calls `session/delete`; otherwise (disposable `--session-dir`
  storage present) it just kills the process. [V-SRC structured-agent-session-host-assembly-Do87TcHO.js
  (`Jp`/`Yp`) + claude-stream-json-connection-BbSXMIDZ.js (`IV.adoptCommands`), installed build 2026-11-23.]
- **Item shape** (per item of `availableCommands`): `{name, description?, input?: {hint?}}`.
  The per-dialect mapper (`PV`) renders `{name, kind, description?, argumentHint?, builtin?}`
  where `kind` is `"skill"` only when the dialect's `skillDescription(item)` returns a
  description — **the omp dialect defines no skillDescription, so every omp item renders as
  `command`**; skills rows for other agents ride the same list through that hook
  (copilot: `input.hint === "instructions for the skill"`; kimi: `name.startsWith("skill:")`).
- **The omp chat composer gates the menu on `commandsEnabled`/`skillsEnabled`; with neither
  fed the composer shows "No matching commands".**
- **The omp dialect's `/model` and its refusals are already catalog-aware**:
  `modelCommand.resolvesSelectorsFrom: "18.1.9"` — a wrapper on/above 18.1.9 is expected to
  serve the catalog (the wrapper's version stamp is `18.1.9+zacp.0`).
- **skillRoots (visuals)**: the omp spec's `skillRoots` hook passes the configured visuals
  skill roots to the card (`--plugin-dir`); version window `18.1.5` ≤ v < `18.1.9` disables it —
  the wrapper's stamp (`18.1.9+zacp.0`) is on the enabled side.

**Wrapper consequence (the v0.21.0 feature)**: on `session/new` the wrapper emits ONE
`session/update` with `sessionUpdate: "available_commands_update"` whose items are the
ZCode skills (scanned from `~/.zcode/skills/**/SKILL.md` + plugin caches, name/description
from frontmatter) plus the wrapper's own `/model` — the same list for chat and discovery
sessions. No delete-capability advertisement is required (the client's disposable-storage
arm suffices; the wrapper answers `session/delete` regardless — SESS-02).
