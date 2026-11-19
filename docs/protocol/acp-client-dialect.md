# ACP Client Dialect — the `omp` preset-agent contract

**Domain:** the chat-UI client's ACP v1 dialect as consumed by the `omp` preset-agent card (DISC-02)
**Verified against:** fresh `--appimage-extract` of the installed Orca.AppImage (squashfs-root),
read 2026-11-19 in chunks `agent-session-attachment-store-registry-CR_MCAgP.js`,
`structured-agent-session-host-assembly-BbqHQ74m.js`, `session-scanner-omp-subagent-transcripts-B3v1LJlD.js`,
`session-json-object-CljYgSvh.js`, `commit-message-agent-output-lRxbgzAd.js`, `omp-profile-DU6kfV0l.js`
**Confidence:** HIGH — every load-bearing claim below was re-read from that fresh extract of the
installed build (upgrade from the parallel-session V-SRC(parallel) table); the two formerly open
items are answered statically and cited to `[V-EXP EXP-SESSIONDIR-01]`; the two remaining
known-unknowns stay V-RPT-labeled.
**Build drift warning:** chunk hash suffixes differ per client build. This document describes the
build named in the header above; re-run the freshness greps (capture `EXP-SESSIONDIR-01`) after any
client update before relying on it further.

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
  launch command cannot be replaced [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#tD/ore].
- **copilot** — `modelDiscovery:{kind:\`unavailable\`,reason:\`Copilot lists the models a plan
  allows only inside a session\`}` — a dead pre-session model picker [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#copilot-card].
- **grok** — heavy private line dialect: the card's dialect carries the `x.ai/*` method family
  (`x.ai/ask`, `x.ai/session`, `x.ai/task`, `x.ai/mcp`, …) instead of plain ACP semantics
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#grok-dialect-Eb].
- **opencode / opencode2** — occupied by the real upstream tools plus vendored child-session
  extensions (`opencode/child-session-updates` update channel, opencode2 pinned `2.0.14`,
  opencode model discovery switching by version below `2.0.0`) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#opencode-cards].
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
managed-agent-hook-controls-Nvpuw5Qw.js#zcode-hooks + session-json-object-CljYgSvh.js]. The
`omp` card therefore remains the only viable preset-agent identity for an ACP server that must
present itself as a preset agent; the identity decision is unaffected.

## Launch contract

The client launches the card as a stdio subprocess (no PTY, no TUI — the client renders the
whole UI). Effective argv for the `omp` card under the default registration:

    <program> acp [--plugin-dir <dir>] [--model <id>] --session-dir <dataDir>/acp-command-sessions/omp

- `<program>` is `omp` by default, or the **per-agent command override** (below) resolved on PATH.
- `--plugin-dir` is appended only when a visuals skill is configured; `--model <id>` only when a
  model was picked in the client UI (`launchesWithModel:!0` — the picked model is applied at
  session start) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#omp-card + structured-agent-session-host-assembly-BbqHQ74m.js#argv-assembly].
- `--session-dir` is always appended last; the value is
  `<client dataDirectory>/acp-command-sessions/omp`, pre-created by the client before launch
  [V-SRC structured-agent-session-host-assembly-BbqHQ74m.js#acp-command-sessions].
- `fullAccess` is always `false` at spawn — permissions flow through
  `session/request_permission` instead [V-SRC structured-agent-session-host-assembly-BbqHQ74m.js#fullAccess-false].
- Environment: the client sets **`PI_NO_PTY=1`** for the omp card [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#env-PI_NO_PTY]. Ignore it — ACP is a pipe
  protocol; never allocate a PTY.
- Agent home: the card's account home env is `PI_CODING_AGENT_DIR`; the client derives
  `<PI_CODING_AGENT_DIR>/.omp/agent` as the agent home (sessions live at
  `<agent-home>/sessions`; default `~/.omp/agent/sessions`) [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#account-home + session-scanner-omp-subagent-transcripts-B3v1LJlD.js#sessions-map].

**Per-agent command override semantics** (the official wiring used by this project:
`{"omp": "<wrapper> omp"}`):

- The override is `record<agentId, string>` in the client's settings store, siblings
  `agentDefaultArgs` / `agentDefaultEnv` / `agentPermissionOverrides` [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#settings-keys].
- **Prefix arguments are allowed**: the string is shell-quoted-tokenized, the first token is the
  program, the rest are prefix args prepended before the card's own arguments [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#override-parse].
- The parse rejects: empty strings, embedded NUL/CR/LF, `$`, backtick, any shell metacharacter
  (`| & ; < > ( ) { } [ ] * ? !`) in the program or any prefix arg, and env-assignment prefixes
  (`VAR=val cmd` is invalid) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#metachar-rejection].
- `~/`-prefixed args are expanded against the user home at resolution time [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#tilde-expansion].
- A non-empty override **satisfies the installed-detection** check — no binary named `omp` is
  needed in PATH [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#mD-installed-detection].

## Version floor

- Card floor: **`17.0.5`** (`minimumVersions:["17.0.5"]` on the omp card) [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#minimumVersions].
- Probe mechanics: the client runs `<program> <prefixArgs...> --version` with a 5 s timeout and a
  4096-byte output cap; parse failures yield `unknown`, non-start yields `missing`, a version
  below the applicable floor yields `tooOld`; `ready` and `unknown` are both treated as usable
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#probe-XO-qO-JO].
- Reported-version policy: the Hb dialect declares **`resolvesSelectorsFrom:"18.1.9"`** — model
  selectors (named model arguments in the `/model` flow) resolve only from that version
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#Hb-modelCommand]. The wrapper
  therefore prints **18.1.9+** on `--version`, not the bare floor (locked integration decision
  [V-RPT PROJECT.md key decisions, 2026-11-19]).
- Probe shape under prefixed overrides — the static question from research assumption A2 is now
  answered by the chunk read above: the probe is `prefixArgs + --version`, **without the card's
  own args**, so an override `<wrapper> omp` is probed as `<wrapper> omp --version`. Our binary
  must answer `--version` under the `omp` subcommand (the cobra root+subcommand decision covers
  this). Recorded here as an upgrade of the plan's V-RPT expectation — mechanics verified
  statically [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#probe-XO]; the
  end-to-end live probe against the real wrapper remains a Phase 7 item [V-RPT Phase 7 plan].

## Load-bearing method set

Observed call sequence, split by phase (methods sent both at startup and later are listed in
both sets with a cross-reference).

At card startup (per launched card process), in order:

1. `initialize` — the client sends `protocolVersion: 1` and its capabilities; a non-1 reply is
   rejected with -32602 [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#initialize-call].
2. `session/new` — with `cwd` and `mcpServers:[]` defaults; the sessionId returned by the card
   becomes the chat's handle [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#session-new].
   (Overlap note: `session/new` also recurs whenever the user opens a *new* chat on an already
   running card — same method, later in time; see the per-turn set.)
3. `authenticate` — only when the card advertises auth methods at initialize; the omp flow for
   this project does not use it (auth lives in the desktop credential store, not the card)
   [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#authenticate].
4. `session/load` / `session/resume` — **only** when the card advertised `loadSession` or a
   resume capability at initialize; otherwise the client throws
   `-32601 "ACP agent cannot load or resume this session"` (see Open items) [V-SRC
   agent-session-attachment-store-registry-CR_MCAgP.js#session-establishment].

Per turn / per user action, on the same process:

- `session/prompt` — one call per user turn, run through the client's busy-retry wrapper
  (`retryBusyPrompts` is enabled specifically for `omp`) [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#session-prompt + structured-agent-session-host-assembly-BbqHQ74m.js#retryBusyPrompts-omp].
- `session/cancel` — user interrupt [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#cancel].
- `session/set_model` — the client's model-change call; for `omp` the picker itself is
  in-session (see Open items) [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#set-model].
- `session/set_mode`, `session/set_config_option` — mode/config changes on user action [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#set-mode-config].
- `session/new` — recurs on new chats (overlap with the startup set, listed in both).
- Session management ops when triggered: `session/list`, `session/fork`, `session/delete`
  [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#session-management].

Ordering honesty: the startup order initialize → session/new is the observed code order (the
connection initializes before any session call can reference it) and is load-bearing; the order
among per-turn methods is user-action driven, not fixed.

## Full ACP method range the client sends

Complete call-site inventory of this build's ACP client stack [V-SRC
agent-session-attachment-store-registry-CR_MCAgP.js#method-literal-counts]:

| Direction | Method | Notes |
|-----------|--------|-------|
| client → agent | `initialize` | protocolVersion 1, handshake |
| client → agent | `session/new` | fresh session |
| client → agent | `session/load` | only if `loadSession` capability advertised |
| client → agent | `session/resume` | only if resume capability advertised |
| client → agent | `session/prompt` | one per turn |
| client → agent | `session/cancel` | interrupt |
| client → agent | `session/set_mode` | mode change |
| client → agent | `session/set_model` | model change |
| client → agent | `session/set_config_option` | config options |
| client → agent | `session/list` | session enumeration |
| client → agent | `session/fork` | fork a session |
| client → agent | `session/delete` | delete a session |
| client → agent | `authenticate` | only if auth methods advertised |
| client → agent | `_extension` methods | require a leading underscore, 30 s default timeout |
| agent → client | `session/update` | streaming updates (notifications) |
| agent → client | `session/request_permission` | permission prompts |
| agent → client | `elicitation/create` | form-mode questions (`{mode:"form",message}`) |

The client advertises `fs.readTextFile/writeTextFile:false` and `terminal:false`, so it never
accepts `fs/*` or `terminal/*` requests — a card must not send them (see Client capabilities).

## String contracts

Exact byte-for-byte strings (equality means exact string match after the prefix-match form
shown; no normalization) of the Hb dialect [V-SRC
agent-session-attachment-store-registry-CR_MCAgP.js#Hb-block]:

- Model switch: the client sends the slash **prompt** `/model <name>` as a `session/prompt`
  turn (`text:e=>\`/model ${e}\``) — model switching is a prompt convention, not a method.
- Model refused: the card's error detail starts with `Unknown ACP model: ` — matched by
  `startsWith` (`refusedByOption`).
- No model: the error detail starts with `No model selected.` (`noModel`).
- Session lost: the error detail equals `ACP cancel cleanup timed out` **or** equals
  `Unsupported ACP session: <sessionId>` (`sessionLost` matcher; the second form is built by
  interpolating the session id).
- Authentication-required: the error detail starts with `No API key found for ` — that is the
  *card's own* credential model; not applicable to this project (auth lives in the desktop
  credential store; the wrapper never emits this string).
- Model-selector resolution floor: `resolvesSelectorsFrom:"18.1.9"` (see Version floor).

## Client capabilities

What the client advertises at `initialize` and how the card must behave [V-SRC
agent-session-attachment-store-registry-CR_MCAgP.js#initialize-call + Hb clientCapabilities]:

- `fs: {readTextFile:false, writeTextFile:false}` and `terminal:false` — **never call client
  fs/terminal**; the card reads and writes only its own side of the wire and its own process
  resources.
- `elicitation: {form:{}}` — the card may ask form-mode questions via `elicitation/create`
  (`{mode:"form", message}`); the client renders them.
- Compaction: the omp card sets `compaction:!0, compactionInstructions:!0`; the client offers
  compaction (with instructions) to the chat, and the Hb dialect carries a
  `compactionReply` handler [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#omp-card-compaction].
- `retryBusyPrompts` is enabled for `omp` — the client retries `session/prompt` while the card
  reports busy [V-SRC structured-agent-session-host-assembly-BbqHQ74m.js#retryBusyPrompts-omp].
- Card-side capability set the client derives for `omp`: `rewind:false`, `fork:"tip"` (transcript
  import at the fork tip), `compact:true`, `compactInstructions:true`, `threadGoal:false`,
  `contextUsage:true`, `imagePrompts:true`, `steering:"queue"`,
  `approvalEnforcement:"orca"` [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#fD-capabilities].
- `clientInfo` the client sends: `{name:"orca", version:"1"}` [V-SRC
  agent-session-attachment-store-registry-CR_MCAgP.js#QS-clientInfo].
- No PTY anywhere: the client spawns the card with stdio pipes and sets `PI_NO_PTY=1`; the TUI
  layer of the underlying agent is never used.

## Open items

Two items were formally open after the parallel-session pass; both are now answered statically
and captured in `experiments/captures/EXP-SESSIONDIR-01.jsonl` [V-EXP EXP-SESSIONDIR-01]:

1. **`--session-dir` session-file format.** The client pre-creates
   `<dataDir>/acp-command-sessions/omp` and passes it via `--session-dir`; the scanner side of
   the client reads omp history from the agent-home sessions dir (`~/.omp/agent/sessions`,
   env-overridable via `OMP_CODING_AGENT_DIR`/`PI_CODING_AGENT_DIR`, profile and
   `XDG_DATA_HOME` variants). Format, fully specified [V-EXP EXP-SESSIONDIR-01]:
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
     Subagent transcripts are sibling `.jsonl` files in the same slug dir.
   - **Verdict: Yes** — a Go agent can write omp-compatible history files. Scope honesty:
     our v1 does **not** advertise `loadSession`, so the client cannot load these sessions
     interactively (it throws `-32601 "ACP agent cannot load or resume this session"` on the
     resume path [V-SRC agent-session-attachment-store-registry-CR_MCAgP.js#session-establishment]);
     compatible files are a Phase 7 nicety (history listing / fork import), not a v1 gate.
2. **`modelDiscovery` kind `session`.** The omp card block is
   `modelDiscovery:{kind:"session", listingNamesConfiguredModel:false, sessionFolderArgs:["--session-dir", …]}`:
   models are **not** listed pre-session (no startup model list is rendered — the empty-list
   surface is by design); models are listed inside the session via the `/model` mechanics, and
   model-selector arguments resolve only from version 18.1.9 [V-EXP EXP-SESSIONDIR-01].

Still unresolved, recorded as known-unknowns (never presented as verified):

- **Settings-store location for `agentCmdOverrides`** — not found in `~/.config/orca/*.json`;
  suspected Electron storage/orcad. Configuration happens through the Settings UI. [V-RPT
  parallel-session check + research assumption A4] — Phase 7 live concern.
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
  - `managed-agent-hook-controls-Nvpuw5Qw.js` — operations-only agent ids, `zcode` hooks target.
- `[V-EXP EXP-SESSIONDIR-01]` — `experiments/captures/EXP-SESSIONDIR-01.jsonl` (E5 static
  answers + re-verification evidence; produced by the scrub-first pipeline, raw capture outside
  the repo).
- `[V-RPT …]` — parallel-session compiled table (`.planning/research/REFERENCE-ADAPTERS.md`),
  PROJECT.md key decisions, and Phase 7 plan items; not verified in this phase.
- The packed `app.asar` was never read; all reads went through the unpacked chunk files
  (anti-pattern guard from the phase research).

---
*Phase: 1 — Research & Protocol Documentation (DISC-02)*
*Verified against the installed client build on 2026-11-19*
