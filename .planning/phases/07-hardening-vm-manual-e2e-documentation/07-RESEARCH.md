# Phase 7: Hardening, VM & Manual E2E, Documentation - Research

**Researched:** 2026-11-22
**Domain:** ACP v1 session-compat hardening (`session/delete` / `session/fork`), full Orca omp-dialect coverage (the `/model` slash prompt + `--model` launch binding), one named CI conformance scenario, a reproducible clean-VM script, the manual Orca E2E user gate, and the English docs surface
**Confidence:** HIGH — every load-bearing wire shape was read this session from the pinned SDK (module cache), the in-repo dialect documents (V-SRC/V-EXP graded), and cross-verified against the Orca client source @v1.4.262 and the official ACP spec fetched live; the routed hardening items were re-anchored in the current code line-by-line

## Summary

Phase 7 closes the done-v1 bar. This research answers the six questions the CONTEXT decisions pinned: (1) the exact `session/delete` / `session/fork` wire shapes (SESS-02), (2) the DISC-02 method-table sweep status as a concrete gap table plus the byte-exact Hb string contracts and their wire carriage (`-32603` with `data.details`) (SESS-03), (3) the `--model` launch-binding mechanics including the reasoning-level default (D-03), (4) what the clean-VM script must cover for THIS wrapper including the no-keyring error path and the paired provider-config env fact (QA-02), (5) the verbatim omp card config + the per-auth-path manual E2E checklist (QA-03), and (6) what the fake corpus already holds vs what the named conformance scenario adds (QA-01). It also re-anchors all seven routed findings (WR-01/04/05, IN-01..04) in the current code.

The decisive findings: **`session/delete` must change behavior** — today the wrapper answers `-32601`; the dialect result shape is the empty result `{}` (SDK `DeleteSessionResponse{}`, spec-confirmed "SHOULD succeed silently" for unknown sessions). **`session/fork` is already graceful** — the SDK dispatcher answers a clean `-32601` for the unimplemented optional interface, never a crash; Phase 7 byte-locks that shape and optionally makes the arm explicit. **Neither delete nor fork is ever called by Orca on our wrapper** (both are capability-gated client-side and we advertise neither — verified in Orca source this session), so the graceful arms are robustness for spec-loose clients plus the SESS-02 acceptance bar. **The `/model` slash prompt refusals are invisible unless carried as `-32603` with `data.details`** — the Hb matchers read exactly that path (`z.looseObject({details: z.string()})` on code `-32603`), and our current `wireError` always passes `data: nil`, so the slash-prompt layer needs a new seam surface for detail-carrying refusals. **`--version` is load-bearing for the whole E2E**: the client probes `<wrapper> omp --version`, parses semver, and treats `< 17.0.5` as `tooOld` (unusable) — the current `0.0.0-dev` stamps would block the card; the locked decision is to print 18.1.9+.

**Primary recommendation:** Build Phase 7 as six waves — (1) the lifecycle-consistency pass (WR-01/04/05 + IN-01..04 batched), (2) session-compat arms (delete no-op + fork explicit decline + the DISC-02 corpus rows), (3) the `/model` slash prompt + `--model` omp launch binding with the version stamp, (4) the named conformance scenario, (5) the VM script, (6) README + launch recipe + docs/ + the UAT recording for the user gate.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Implementation Decisions

**Session-Compat Surface (SESS-02)**
- **D-01:** `session/delete` answers a graceful no-op in the dialect's result shape (child-side cleanup follows `session/close` semantics where applicable — the runtime owns the real deletion); `session/fork` is handled gracefully — never a crash path. Exact result shapes are taken from the dialect document/bundle (researcher pins; fake corpus byte-locks both).

**Full Orca Dialect Coverage (SESS-03)**
- **D-02:** the phase-1 dialect method table (the DISC-02 load-bearing set) is swept one-by-one: every method is implemented or gracefully handled (coded no-op / protocol-correct decline per the established posture) — proven by a fake-corpus conformance row per uncovered method.
- **D-03:** the persona CLI contract lands: the wrapper accepts `--model <provider/model>` at launch (the launch binding rides the same setModel follow-up machinery — the reasoning level defaults to the backend-confirmed shape) and the `/model <name>` slash prompt is handled with the exact Hb string contracts (`Unknown ACP model: …`, `No model selected.` — researcher pins the verbatim strings + expected behaviors from the omp card contract); `launchesWithModel:true` becomes claimable in the launch recipe (DOCS-02).

**CI Protocol Conformance (QA-01)**
- **D-04:** one named CI conformance scenario composes the existing proven corpus pieces into the full v1 arc — initialize → session/new → prompt streaming → tool calls → permission decision → cancel — byte-locked, named, and green in `mise run ci`. Existing suites stay; this is the single named "conformance suite" the done-v1 bar names.

**Reproducible VM Scenario (QA-02)**
- **D-05:** the clean-Ubuntu-GNOME scenario is scripted in-repo (`scripts/vm-*` or a documented equivalent): install → configure (both auth paths) → run → the no-keyring error path asserted. Running it on an actual VM is a MANUAL tier act (the script is the deliverable; one documented execution by the user is the field proof — recorded in the ledger).

**Manual E2E (QA-03 — the USER GATE)**
- **D-06:** the wrapper side completes everything automatable (recipe correctness, PATH shim, card config docs); the actual Orca card E2E through BOTH auth paths is performed by the user on the real desktop and recorded in the UAT + ledger (the phase-6 UAT precedent). End-of-phase human gate, never autonomous.

**Documentation (DOCS-01/02/03)**
- **D-07:** English README (what it is — an honest ACP server for any ACP client; install; configuration; both auth paths; troubleshooting), the preset-agent launch recipe (binary naming `zcode-acp` with the omp-compatible contract, version floor, env, PATH shim), and `docs/` protocol notes with the provenance statement (no patching, no attestation forgery, PAT signed by the official binary; Apache-2.0 attribution for ported reference-adapter code) and known limitations (no quota counter over OAuth; Start Plan not headless). Language: the repo's user-facing docs are English (PROJECT.md), planning artifacts stay as they are.

**Routed Hardening (from 06-REVIEW-DISPOSITION.md)**
- **D-08:** WR-01 (refused picker setModel leaves the front session alive vs dead backend), WR-04 (three post-create failure paths leak cwd/settings map entries), WR-05 (resume/close TOCTOU on isClosing + Register) — settled as one lifecycle-consistency pass in this phase. IN-01 (builtin-table double-read), IN-02 (adapter comment claims absent validation), IN-03 (keyringprobe deadline-boundary classification race), IN-04 (dead `settings == nil` re-init) — advisory cleanups batched here. Dispositions flip to resolved as each lands.

### Claude's Discretion
- Exact delete/fork result shapes (dialect-verified); the conformance scenario's name/location; VM script form (bash + docs vs testfile); doc file layout inside `docs/`; whether IN-03 needs a behavior change or a comment (the fail-fast semantics are already correct); task/wave decomposition; which routed advisory lands in which wave.

### Deferred Ideas (OUT OF SCOPE)
- session/load/fork real support — v2
- elicitation/create preference — beyond v1
- Non-Ubuntu-GNOME environments — accepted PRs post-v1
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SESS-02 | `session/delete` graceful no-op; `session/fork` handled gracefully | Exact shapes pinned: delete → empty result `{}` (SDK `DeleteSessionResponse{}` + spec "SHOULD succeed silently"); fork → SDK `-32601 "Method not found" {method:"session/fork"}` via the unimplemented optional interface. Orca never calls either (capability-gated). See Architecture Patterns 1-2. |
| SESS-03 | Full Orca dialect coverage incl. `--model` at launch (`launchesWithModel:true`) and the `/model` slash prompt with exact Hb contracts | Gap table (Architecture Patterns 3): every DISC-02 method classified implemented / graceful / missing. Hb strings byte-pinned from two sources; refusals must ride `-32603` + `data.details`; `--model` maps to the setModel follow-up with the backend-confirmed reasoning level. See Patterns 3-5, Pitfalls 1-3. |
| QA-01 | CI conformance suite green: one scripted scenario covering the full arc | Corpus inventory (34 scenarios, per-name fail-loud drivers) + what the named scenario adds (the cancel leg composed into one ACP-side arc over the composed harness). See Patterns 6. |
| QA-02 | VM scenario scripted, reproducible from the repo, incl. the no-keyring error path | Script contents derived: static binary (no mise on the VM), desktop discovery, both auth paths, the four coded keyring sentinels, the paired provider-config env fact. See Patterns 7, Pitfall 6. |
| QA-03 | Manual E2E: working card in real Orca, real account, both auth paths | Verbatim card config (`agentCmdOverrides`), the PATH-shim necessity (env-assignment prefixes rejected by the client tokenizer), the version-floor probe mechanics, and the per-path checklist + UAT recording precedent. See Patterns 5 and 8, Pitfalls 2-3. |
| DOCS-01 | English README | Current README is CI-only (71 lines) — full rewrite per D-07. See Patterns 8. |
| DOCS-02 | Launch recipe for preset-agent-only hosts | Recipe inputs all pinned: override wiring, binary naming, version floor 18.1.9+, env, shim. See Pattern 8. |
| DOCS-03 | `docs/` protocol notes + provenance + limitations | Base docs exist (`docs/protocol/` 5 files, `docs/research/` 1); provenance statement + limitations content enumerated. See Pattern 8. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| `session/delete` graceful no-op | ACP front adapter (`internal/acpport/agent.go` + `internal/acpfront/`) | Fake corpus (byte-lock) | The adapter owns the wire arm; the front supplies close-semantics cleanup where a session exists |
| `session/fork` graceful decline | SDK dispatcher (`acp-go-sdk`) → adapter | Fake corpus | The SDK already answers `-32601`; making it explicit keeps the corpus honest |
| `/model` slash prompt | `internal/acpfront/` (prompt intercept layer) | `internal/acpport` (detail-carrying error surface) | Model switching is a prompt convention in the omp dialect, not a method; the front owns the session's confirmed options |
| `--model` launch binding | `cmd/zcode-acp` (CLI parse) | `internal/acpfront` (setModel follow-up) | The card argv arrives at process launch; the binding reuses `applyConfiguredModel` |
| Lifecycle consistency (WR-01/04/05) | `internal/acpfront/` (`front.go`, `lifecycle.go`) | tests | All three live in the same teardown/set-gated code |
| Conformance scenario | `internal/acpfront` composed test tier | `internal/zcode/fake` corpus | The full arc needs the SDK-client ⇄ tee ⇄ agent ⇄ fake composition |
| VM scenario script | `scripts/` (new) + docs | `internal/supervisor` behavior | The script asserts the shipped binary's discovery/probe/error behavior |
| README / launch recipe / docs/ | Repo docs surface | — | English user-facing docs per D-07 |
| Manual E2E | User on the real desktop | guarded real-tier tests (`test/realdesktop`) as pre-verification | D-06: human gate, never autonomous |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/coder/acp-go-sdk | v0.20.0 (pinned, schema 1.25.0) | ACP v1 front leg — owns dispatch, framing, the delete/fork arms' default answers | Already the module's single dependency; its dispatcher already implements the graceful fork decline [VERIFIED: module cache read this session] |
| Go stdlib `flag` | go 1.27 | CLI parse for the omp card argv contract | The shipped binary is stdlib-flag based (`cmd/zcode-acp/main.go:82-89`); go.mod has exactly ONE require line and the `tidy-diff` mise gate asserts it [VERIFIED: go.mod + mise.toml read this session] |
| Go stdlib `os/exec` (secret-tool, script plumbing) | — | Keyring probe, VM script assertions | Established zero-dep posture: the D-06 credential loader shells `secret-tool` (`main.go:52-55`) — godbus never landed |

### Supporting
No new packages are introduced by this phase. Every work area rides the existing stack. The only dependency-adjacent decision is omp subcommand parsing (see Alternatives).

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled omp subcommand dispatch over stdlib `flag` | spf13/cobra v1.10.2 (the STACK-table 2026-11-19 decision, `FParseErrWhitelist`) | Cobra gives unknown-flag tolerance for free but adds a second require line — the `tidy-diff` gate task explicitly anticipates this ("extend this expectation when the godbus dependency lands") and would need updating. The card argv surface is tiny (`acp` positional ignored; `--plugin-dir`/`--session-dir` accepted-ignored; `--model` used; `--version` any position), so a pre-tokenizing `flag.FlagSet` pass is ~50 lines. **Planner decides; both are compliant with D-03.** The cobra decision was recorded in the stack table but was never adopted — phase 4 shipped stdlib flag |

**Installation:** none — no new packages.

**Version verification:** no registry lookups required this phase (no new packages). The SDK pin is asserted by the `tidy-diff` gate.

## Package Legitimacy Audit

> No external packages are installed by this phase — the gate was considered and vacuously passes.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| (none — zero new dependencies) | — | — | — | — | — | n/a |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                ┌────────────────────────── Orca (chat UI, omp card) ──────────────────────────┐
                │  version probe: "<wrapper> omp --version" (5 s, 4096 B cap, semver ≥17.0.5)  │
                │  chat argv: zcode-acp omp acp [--plugin-dir <dir>] [--model <id>]            │
                │  discovery argv: zcode-acp omp acp --session-dir <folder>  (disposable)      │
                │  /model <name> slash prompt ── as a session/prompt TURN                      │
                └──────────────┬───────────────────────────────────────────────────────────────┘
                               │ ACP v1 (JSON-RPC, stdio)
                               ▼
   ┌────────────────────────── zcode-acp wrapper process ──────────────────────────┐
   │ cmd/zcode-acp: argv dispatch (--version | omp acp ... | root acp mode)        │
   │        │                                                                      │
   │        ▼                                                                      │
   │ internal/acpport: SDK Agent adapter (12-method iface)                         │
   │   initialize/session-new/prompt/cancel/authenticate/resume/close/             │
   │   set_config_option  ── implemented                                          │
   │   session/delete     ── Phase 7: graceful {} (was -32601)                     │
   │   session/fork       ── SDK default -32601 (optional iface) / explicit arm    │
   │   logout/list/set_mode/load ── clean -32601 declines                          │
   │        │                                                                      │
   │        ▼                                                                      │
   │ internal/acpfront: session policy (D-03 create+subscribe+setModel;            │
   │   turn slot; closing set; settings/cwd stores; /model slash intercept;        │
   │   lifecycle teardown — WR-01/04/05 pass)                                      │
   │        │                                                                      │
   │        ▼                                                                      │
   │ internal/zcode: JSONL codec conn ──► supervisor-owned child                   │
   │                                        │                                      │
   │        interaction/* responder ◄───────┤ (permission, runtime-headers PAT,     │
   │        (interact/hostsvc)              │  credential-cipher keyring, MCP-auth  │
   │                                        │  decline)                             │
   └────────────────────────────────────────┼──────────────────────────────────────┘
                                             ▼
                        ZCode desktop app-server 3.18.2 (JSONL, host-managed)
                        "session/delete" ×0 on this wire — nothing to delete
```

### Pattern 1: `session/delete` — the graceful no-op arm (SESS-02, D-01)

**What:** Replace the current `-32601` default with the dialect's result shape: an empty result `{}`.

**Wire truth (pinned):**
- Request params: `{sessionId}` — SDK `DeleteSessionRequest` [VERIFIED: SDK types_gen.go `DeleteSessionRequest`, read this session].
- Result: `{}` — SDK `DeleteSessionResponse` carries only `_meta,omitempty` [VERIFIED: SDK types_gen.go:2902-2909]; the spec shows `"result": {}` [CITED: agentclientprotocol.com/protocol/v1/session-delete — fetched 2026-11-22].
- Spec semantics: deleting a nonexistent or already-deleted session "SHOULD succeed silently"; the method is gated on `sessionCapabilities.delete` and "Clients MUST NOT attempt to call session/delete" when unadvertised [CITED: same spec page].
- The back leg has nothing to delete: `"session/delete"` occurs **×0** in the shipped 3.18.2 bundle [VERIFIED: grep re-run this session on `~/Applications/zcode-3.18.2/squashfs-root/resources/glm/zcode.cjs` → 0; matches app-server-dialect.md §7].
- Current wrapper: `DeleteSession` returns `unadvertised(acp.AgentMethodSessionDelete)` → `-32601 "Method not found", data {"method":"session/delete"}` [VERIFIED: internal/acpport/agent.go:417-423 + SDK errors.go:58-59 read this session].

**Design shape (planner options):**
- For an unknown/unregistered id: answer `{}` immediately (the spec's silent-success arm; the runtime owns real deletion — there is nothing to delete).
- For a registered id: run the SESS-01 teardown semantics ("child-side cleanup follows `session/close` semantics where applicable" — D-01) and answer `{}`. Reusing `Front.CloseSession` gives cancel-drain → backend close → registry release → observer removal → map drops for free; the reply is the same empty result.
- **Keep `sessionCapabilities.delete` UNADVERTISED** — advertising it would be dishonest (no real deletion exists) and would change Orca behavior for the worse (Orca would start calling delete on its disposable listing sessions; today it uses throwaway folders precisely because omp does not advertise delete [CITED: docs/protocol/acp-client-dialect.md:275-280]).

**Byte-lock:** the fake corpus row pins the empty-result frame.

### Pattern 2: `session/fork` — already graceful, make it explicit and byte-locked (SESS-02, D-01)

**Wire truth (pinned):**
- `session/fork` is an **UNSTABLE** method in schema 1.25.0. The SDK routes it through the optional interface `AgentUnstableForkSession`; when the agent does not implement it the dispatcher answers exactly `NewMethodNotFound(method)` → `-32601 "Method not found", data {"method":"session/fork"}` [VERIFIED: SDK agent_gen.go:191-206 read this session — `impl, ok := a.agent.(AgentUnstableForkSession); if !ok { return nil, NewMethodNotFound(method) }`].
- The fork response shape (for the record, v2 scope): `{sessionId (required), configOptions?, modes?}` [VERIFIED: SDK types_gen.go:14141-14156].
- Orca gates the call client-side: `forkAdvertised` throws `-32601 'session/fork is not advertised'` unless `agentCapabilities.sessionCapabilities.fork` is set [VERIFIED: github.com/stablyai/orca v1.4.262 `src/main/acp/conversation-copy/acp-conversation-copy.ts`, fetched this session]. We advertise no fork capability → Orca never sends it.

**Design shape:** the graceful arm exists today by SDK default and never crashes. Phase 7's work is (a) a corpus/conformance row byte-locking the `-32601` frame, and (b) optionally an explicit `UnstableForkSession` method on `agentAdapter` returning `unadvertised(acp.AgentMethodSessionFork)` — same bytes, self-documenting, testable without the SDK default. Keeping the SDK default is equally compliant; the byte shape is identical.

### Pattern 3: the DISC-02 method-table sweep — gap table (SESS-03, D-02)

The full Orca client method range (docs/protocol/acp-client-dialect.md "Full ACP method range") vs the wrapper today:

| Orca client → agent method | Wrapper status today | Phase 7 action |
|---|---|---|
| `initialize` | implemented (D-02 honest report) | regression row |
| `session/new` | implemented (+ configOptions, MODEL-01) | regression row |
| `session/prompt` | implemented (D-04/D-05) | + the `/model` slash intercept (Pattern 4) |
| `session/cancel` | implemented (notification, D-05) | regression row |
| `session/set_config_option` | implemented (MODEL-02) | regression row |
| `session/set_mode` | adapter `-32601` default (`unadvertised`, agent.go:497-503) | corpus row (graceful decline). Orca sends it only "on user action"; we advertise no modes, so the client has no mode picker to drive it |
| `session/set_model` | **not in the SDK dispatch table at all** (schema 1.25.0 has no agent `session/set_model` case) → SDK fallback `-32601` (connection.go:663 `NewMethodNotFound(req.Method)`) [VERIFIED: constants_gen.go + agent_gen.go case list read this session] | corpus row. Unreachable from Orca for us: the client prefers config-option writes and only falls to `session/set_model` when model state (`models` field) was published — we never publish it [CITED: acp-client-dialect.md:232-236, 343-345] |
| `session/list` | adapter `-32601` default (agent.go:425-431) | corpus row. Orca: "Requires advertised session listing support" [VERIFIED: orca v1.4.262 acp-agent-connection.ts, fetched this session] |
| `session/fork` | SDK optional-interface `-32601` (Pattern 2) | explicit arm (optional) + corpus row |
| `session/delete` | adapter `-32601` → **change to graceful `{}`** (Pattern 1) | implement + corpus row |
| `session/close` | implemented (SESS-01 teardown) | regression row. **Orca calls it UNGATED** on child release with a 10 s timeout, `onTimeout: 'cancel'` [VERIFIED: orca v1.4.262 acp-agent-connection.ts — `closeSession = (sessionId, timeoutMs = 10_000)` with no capability check, fetched this session]. Advertisement question for the planner: we implement close but do not advertise `sessionCapabilities.close`; advertising `{}` is the honest improvement for spec-conformant clients and is a no-op for Orca (it never checks) |
| `session/load` | SDK `AgentLoader` optional → `-32601`; Orca never calls (loadSession unadvertised — the client itself throws `-32601 "ACP agent cannot load or resume this session"`) [CITED: acp-client-dialect.md:245-249] | corpus row optional |
| `session/resume` | implemented (SESS-01); Orca never calls (resume capability unadvertised) | regression row |
| `authenticate` | implemented, both advertised method ids (`zai-api-key`, `zai-individual-coding-plan` — auth.go:41-42) | regression row |
| `session/update` (agent → client) | implemented (sink) | none |
| `session/request_permission` (agent → client) | implemented (permission sender) | none |
| `elicitation/create`, `elicitation/complete` | wrapper never sends them | none (deferred beyond v1) |

Every "corpus row" arm is proven per D-02 by a fake-corpus/conformance row (the graceful arms byte-lock the exact error frames).

### Pattern 4: the `/model` slash prompt layer (SESS-03, D-03)

**What:** a prompt-intercept in the front: when `session/prompt` text starts with `/model ` (the omp dialect sends model switches as prompt turns — `text:e=>`/model ${e}``` [CITED: acp-client-dialect.md:336-338]), resolve the selector against the session's confirmed configOptions and apply it through the SAME setModel machinery `SetSessionConfigOption`'s model arm uses.

**The exact Hb string contracts (byte-for-byte, two independent sources):**
- `Unknown ACP model: ` — **startsWith** match (`refusedByOption`) [VERIFIED: docs/protocol/acp-client-dialect.md:345-346; re-verified verbatim from github.com/stablyai/orca v1.4.262 `src/main/acp/acp-dialects/omp-dialect.ts` — `startsWith('Unknown ACP model: ')`, fetched this session]
- `No model selected.` — **startsWith** match (`noModel`) [VERIFIED: dialect doc :347; same fetch — `startsWith('No model selected.')`]
- `ACP cancel cleanup timed out` — equality (`sessionLost`), and `Unsupported ACP session: ${sessionId}` — equality [VERIFIED: same fetch; dialect doc :348-351]. Never emitted by this wrapper.
- (Not applicable: `No API key found for ` — the card's own credential model; this wrapper never emits it [CITED: dialect doc :352-354].)
- Selector resolution floor: `resolvesSelectorsFrom: '18.1.9'` [VERIFIED: same fetch; dialect doc :355].

**Load-bearing wire carriage:** the matchers extract the detail ONLY from JSON-RPC errors with **code `-32603`** whose `data.details` is a string — `promptErrorDataSchema` is `z.looseObject({ details: z.string() })`; "a refusal carried in any other code or field is invisible to them" [VERIFIED: dialect doc :357-361 + the fetched omp-dialect.ts]. The wrapper's current `wireError` always passes `data: nil` (agent.go:541-552), so **the slash-prompt refusals need a new seam surface**: a coded internal-class error carrying `data: {"details": "Unknown ACP model: <name>"}`. The prompt response for a refused switch is an error envelope (the turn fails with the coded -32603); the client classifies it by the detail prefix and renders its own message.

**Expected behaviors (D-03 "researcher pins"):**
- `/model <name>` matching a listed model value (the configOptions values are the `providerId/modelId` serializations, configoptions.go:296) → apply via `session/setModel` carrying the CURRENT confirmed reasoning level (the `setConfigMutation` model arm, front.go:594-615), then reply as a normal completed turn (a short confirmation text chunk is safe — the client classifies only refusals, never success text).
- `/model <name>` not matching → coded `-32603` with `data.details = "Unknown ACP model: " + <name>` (exact prefix, verbatim).
- `/model` with no argument, or no model bound in the session → `-32603` with `data.details` starting `"No model selected."`.

### Pattern 5: the `--model` launch binding + the omp CLI contract (SESS-03, D-03)

**The card argv contract (verified):** chat launch is `<program> [prefixArgs] acp [--plugin-dir <dir>] [--model <id>]` — no `--session-dir`; disposable discovery launches are `<program> [prefixArgs] acp --session-dir <folder>` instead; `--model` is appended only when a model was picked in the client UI (`launchesWithModel: true`) [CITED: acp-client-dialect.md "Launch contract", corrected-against-source section :108-141].

**Current binary state:** stdlib `flag` root-only — no `omp` subcommand exists yet [VERIFIED: cmd/zcode-acp/main.go read this session]. Critically, stdlib `flag.Parse` stops at the first non-flag argument, so on `zcode-acp omp acp --model X` the `--model` lands in `flag.Args()` and is **silently ignored** — the subcommand dispatch is genuinely new work, not a flag rename.

**What the id looks like:** the client's picker for our card is the config-options picker; its value ids are the `providerId/modelId` serializations our `session/new` returns (configoptions.go:296). So `--model <provider/model>` — matching D-03's stated shape and the existing `parseConfig` cut-on-first-slash parse (main.go:152-155). [ASSUMED: that Orca passes the config-option value id verbatim as `--model` — highly likely by construction, confirmed only at the QA-03 live run; the wrapper should tolerate and log an unexpected shape honestly rather than crash.]

**Mapping (D-03 "the launch binding rides the same setModel follow-up machinery"):** parse `--model` with the existing `strings.Cut` shape → a per-launch `ModelSelection` → `applyConfiguredModel` (front.go:718-758) fires the setModel follow-up after bare create + subscribe. Note `applyConfiguredModel` already implements the whole contract including the WR-04 orphan close on refusal.

**The reasoning-level default (D-03 "defaults to the backend-confirmed shape"):**
- Explicit level first (`--reasoning-level` / `ZCODE_ACP_REASONING_LEVEL` — exists today).
- Otherwise the target model's own confirmed default: `settings.model.available[].reasoning.defaultLevel` is already captured as `ModelEntry.DefaultLevel` [VERIFIED: configoptions.go:137-140]. After bare create the front holds the create-reply snapshot; the binding derives the level for the `--model` target from that snapshot (never a fabricated level — the D-10 posture).
- Fallback: the IN-04 omit-shape (no `options` object) — the backend's own actionable refusal then surfaces, which is field-proven honest behavior: omitting the level on a level-strict provider answers `-32603 "Reasoning level is required for zai-api/GLM-5.3-Flash"` [CITED: app-server-dialect.md §2 quirk].

**Version floor (load-bearing for QA-03):** the client probes `<wrapper> <prefixArgs> --version` — i.e. `zcode-acp omp --version`, prefix args included, card args NOT — 5 s timeout, 4096-byte cap; parse failure → `unknown`, non-start → `missing`, below floor → `tooOld`; only `ready`/`unknown` are usable [CITED: acp-client-dialect.md "Version floor" :186-214]. Card floor 17.0.5; the locked decision prints **18.1.9+** (`resolvesSelectorsFrom`). The current `version = "0.0.0-dev"` (main.go:36) parses as semver 0.0.0 → `tooOld` → **the card is unusable in Orca**. The release stamping must make `--version` print a semver ≥ 18.1.9 (the locked "prints 18.1.9+" decision; exact stamp mechanics are planner discretion — e.g. `18.1.9` compatibility line vs build-metadata form `18.1.9+zacp.N`). Note `--version` must answer at root AND under the `omp` subcommand, any position (the probe shape).

### Pattern 6: the named conformance scenario (QA-01, D-04)

**What exists:** 34 scenario/golden pairs in `internal/zcode/fake/testdata/{scenarios,golden}` [VERIFIED: ls this session], driven by `TestGolden` with a per-name driver map that FAILS LOUD when a scenario has no replay driver (golden_test.go:365-384). The composed ACP-side tier already runs full arcs: `TestFullCycle` (fullcycle_test.go:50) drives authenticate → session/new → prompt with streamed tool cards AND a bridged permission decision → end_turn → resume → close over one harness (SDK `ClientSideConnection` ⇄ recording tee ⇄ `acpport` ⇄ front ⇄ `zcode.Conn` ⇄ `fake.Serve` — harness_test.go:15-25). Cancel has dedicated corpus rows (`cancel-mid-turn`, `complete-then-cancel`, `send-refused-32010`, `lingering-latch-32010`) and unit tiers.

**What the named scenario adds:** the full arc **including the cancel leg composed into ONE named, byte-locked suite** — the ROADMAP success criterion names "initialize → session/new → prompt streaming → tool calls → permission decision → cancel" as a single green suite. Concretely: one named test (planner names it; e.g. `TestConformanceV1Arc`) over the composed harness reusing the proven corpus pieces, byte-locked via the tee, green in `mise run ci`. The SESS-02/03 arms (delete `{}`, fork `-32601`, `/model` refusals) each get their own corpus/conformance rows per D-02.

### Pattern 7: the clean-VM scenario script (QA-02, D-05)

**Script location:** `scripts/vm-*` (the directory does not exist yet — it is new) or a documented equivalent; bash + docs vs a Go testfile is discretion. Running it on an actual VM is MANUAL tier.

**What it must cover for THIS wrapper:**
1. **Install:** obtain the ZCode desktop (the layout matrix: unpacked AppImage at `~/Applications/zcode-*/squashfs-root/`, AppImage + symlink, deb `/opt/ZCode/`; the recommended discovery order env/flag → desktop entry → AppImage paths → package paths) [CITED: docs/protocol/install-discovery.md]. The script asserts `zcode-acp` discovery succeeds (or exercises the actionable nothing-found diagnostics naming every probe).
2. **Run deps:** the wrapper is a static Go binary — NO mise/Go toolchain needed on the VM at run time (build elsewhere or fetch a release artifact). `secret-tool` (libsecret-tools) + gnome-keyring + an unlocked session keyring are required only for the coding-plan path.
3. **Configure both auth paths:** api-key (`ZCODE_ACP_API_KEY` env / `--api-key`) and coding-plan (the desktop login IS the credential — D-12; no key export).
4. **Run:** the version probe (`zcode-acp omp --version`), then a scripted stdio ACP arc (initialize → session/new → one prompt) driven against the real binary.
5. **The no-keyring error path asserted:** the probe's four coded sentinels, each an actionable client-facing message, never a hang [VERIFIED: internal/interact/keyringprobe.go:44-68 verbatim this session]:
   - `ErrKeyringBinaryMissing` = "keyring probe: secret-tool not found (install libsecret-tools on Ubuntu: sudo apt install libsecret-tools)"
   - `ErrKeyringNoDesktopSession` = "keyring probe: no desktop session available (the keyring bus cannot autolaunch headless — run inside your desktop session)" (classified from the coarse `Cannot autolaunch` stderr marker)
   - `ErrKeyringNoEntry` = "keyring probe: no ZCode desktop login found in the keyring (log in via the ZCode desktop first)"
   - `ErrKeyringLookupTimeout` = "keyring probe: keyring lookup did not answer within its bound" (5 s `KeyringProbeTimeout`)
6. **The paired provider-config env fact:** `ChildEnv` passes `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` AND `ZCODE_PERSONAL_PROVIDER_CONFIG_FILE` (`$HOME/.zcode/v2/provider_config.json`) TOGETHER — the unpaired builtin-only form runs the worker degraded (account models never publish; field-proven in the phase-6 gap cycle) [VERIFIED: internal/supervisor/spawn.go `ChildEnv` read this session; CITED: app-server-dialect.md §8]. The script asserts the coding-plan path publishes account models (read-back, never the ack).

### Pattern 8: the launch recipe + manual E2E checklist (QA-03, D-06/DOCS-02)

**The card config, verbatim mechanics:**
- Orca Settings → per-agent command override for `omp` = `<path>/zcode-acp omp` (prefix args allowed; the string is shell-quoted-tokenized; `~/` expanded; shell metacharacters and env-assignment prefixes `VAR=val cmd` are REJECTED; a non-empty override satisfies installed-detection — no binary named `omp` needed in PATH) [CITED: acp-client-dialect.md :160-184].
- Env the client sets: `PI_NO_PTY=1` (ignore); agent home `PI_CODING_AGENT_DIR` [CITED: :148-158].
- **The PATH shim is the recipe's env mechanism:** the client's tokenizer rejects `VAR=val cmd` inside the override, so `ZCODE_ACP_API_KEY` cannot ride the override string. Siblings `agentDefaultArgs`/`agentDefaultEnv`/`agentPermissionOverrides` exist in the settings store [CITED: :163-167] — the recipe documents EITHER an `agentDefaultEnv` entry OR a two-line shim script on PATH that exports the env and execs the real binary. (The api-key path needs the key; the coding-plan path needs none.)
- The settings-store persistence path for `agentCmdOverrides` remains a known-unknown (Electron storage; configured through the Settings UI) — a QA-03 live item [CITED: dialect doc Open items :479-486].

**Per-auth-path manual checklist (prepared so the user's run is ONE recipe + ONE card config per path):**
- api-key: shim/env carries `ZCODE_ACP_API_KEY` → card opens → authenticate{zai-api-key} → models render → one streamed turn with a tool call + permission prompt.
- coding-plan: desktop logged into the individual coding plan, keyring unlocked → card opens → authenticate{zai-individual-coding-plan} (read-back proven, PAT minted) → account models render → one streamed turn.
- Both: `/model <name>` switch from the picker; `--model` honored when a model is picked before launch (observe what id Orca passes — closes the Pattern 5 assumption); session/close on Stop; cancel on interrupt; no crash paths.
- Recording: the phase-6 UAT precedent — `07-UAT.md` + the `experiments/README.md` ledger row + scrubbed capture + `scan-secrets` green on tree and staged diff, human confirmation preceding the evidence commit [VERIFIED: 06-UAT.md and 06-VERIFICATION.md read this session].

### Pattern 9: the routed lifecycle-consistency pass (D-08) — anchors re-verified this session

| Item | Anchor (current code) | Fix shape (from 06-REVIEW.md, confirmed) |
|---|---|---|
| WR-01 | `front.go:549-557` — refused picker setModel calls wire-only `closeOrphan`; the front session stays registered (machine, observers, cwd, settings all persist) → zombie: later prompts send to a dead backend, later picker asks re-fail | On the mutation-refusal path run the full SESS-01 teardown (`f.CloseSession`) or mark closing + release front state — one designed teardown contract |
| WR-04 | `front.go:272` (cwd recorded before subscribe; subscribe refusal returns without dropping), `front.go:289` (settings recorded before applyConfiguredModel; refusal leaks cwd+settings), `lifecycle.go:101` (resume cwd recorded before re-issued subscribe; refusal leaks it) | Drop the entries on the failure paths — `closeOrphan` is the natural single place for the first two; resume drops its cwd on the subscribe-refusal return |
| WR-05 | `lifecycle.go:126-133` — resume's final `isClosing` re-check and `registry.Register` are not atomic with close's marking (`lifecycle.go:148-165`) → microsecond window registers machine B into a closing session; teardown closes only A's observers | Hold `closingMu` across the re-check + Register, or have close re-`Get` after marking and release whatever is registered |
| IN-01 | `snapshot.go` `BuildAccount` — `readBuiltinRevision` then `readBuiltinProviderRules` = two reads of the live builtin file → stale revision/rules pairing | Single read+decode; derive both from the same bytes |
| IN-02 | `agent.go:434-444` — the ResumeSession comment claims "the Handlers return value is validated non-empty" but the code discards the id; the real guarantee lives in the front's `createSessionID` | Correct the comment (or add the check) — batch with the doc pass |
| IN-03 | `keyringprobe.go` `classify` switch — `pctx.Err() == DeadlineExceeded` checked BEFORE the success arm → a boundary-timed success misclassifies as timeout | Reorder (parent-cancel → success → deadline → marker → default) or comment; fail-fast semantics already correct — discretion per CONTEXT |
| IN-04 | `front.go:833-836` — dead `if f.settings == nil` guard (the only constructor initializes the map; never reassigned). Keep `closing`'s lazy init — that one IS load-bearing | Delete the nil check; note the invariant |

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON-RPC framing/dispatch of the ACP front leg | Own dispatcher for delete/fork/set_mode arms | The SDK's dispatcher (already answers `-32601` for unimplemented optional interfaces; `connection.go:663` fallback covers unknown methods) | Byte-exact error frames for free; the SESS-02 fork arm is already correct |
| Coded error envelopes | Raw error strings | `acpport` seam constructors (`NewInvalidRequest`/`NewInvalidParams`/`NewInternal`, types.go:328-357) + `wireError` | The SDK type-asserts `*RequestError`; raw errors degrade to generic `-32603` |
| Teardown ordering | Ad-hoc close sequences in new arms | `Front.CloseSession` (cancel-drain → backend close → registry Remove → CloseObservers → map drops) | The SESS-01 contract is already race-gated and proven; WR-01's fix reuses it |
| Model switch application | A second setModel path for `/model` and `--model` | `setConfigMutation` + `applyConfiguredModel` (the D-03 machinery) | Level-carry, persist-false, orphan-close, and reply-snapshot capture are already locked |
| Keyring probing/failure classification | New error taxonomy for the VM script | The four `ErrKeyring*` sentinels + probe classes | They ARE the client-facing actionable text (AUTH-04) |
| Golden transcript diffing | Custom comparison | `fake.CompareGolden` + the per-name driver discipline | Fail-loud corpus hygiene is established |

**Key insight:** Phase 7 is almost entirely composition and consistency: the wire shapes already exist in the SDK/corpus, the teardown contract exists, the model machinery exists. The genuinely new surfaces are the slash-prompt intercept (with the `data.details` refusal carriage), the CLI subcommand dispatch, and the docs/scripts.

## Common Pitfalls

### Pitfall 1: Hb refusals carried in the wrong field are invisible to the client
**What goes wrong:** emitting `Unknown ACP model: …` as the error MESSAGE, or as `-32602`, or with `data` absent — the client's matchers read ONLY code `-32603` + `data.details` (a string); any other carriage silently fails the classification and the picker shows a generic failure.
**Why it happens:** our `wireError` passes `data: nil` always; the message/data distinction is invisible in our current corpus rows.
**How to avoid:** a dedicated seam error carrying the detail; corpus rows byte-lock the exact frame (`code:-32603, data.details:"Unknown ACP model: <name>"`).
**Warning signs:** Orca shows a generic error instead of "model not found" styling on a bad `/model`.

### Pitfall 2: `--version` too old blocks the card before any protocol flows
**What goes wrong:** `0.0.0-dev` stamps → the probe classifies `tooOld` (< 17.0.5) → unusable → Orca refuses the card; QA-03 fails before authenticate.
**Why it happens:** the version var was a placeholder "until the release flow stamps a real value (Phase 7)" (main.go:35).
**How to avoid:** the phase MUST stamp the binary so `zcode-acp [--omp] --version` prints semver ≥ 18.1.9 (the locked decision), at root and under the subcommand, on stdout, one line, < 5 s.
**Warning signs:** Orca's card shows "update your CLI"-class messaging at add-time.

### Pitfall 3: stdlib `flag` silently ignores the card's flags after positionals
**What goes wrong:** `zcode-acp omp acp --model X --session-dir Y` — `flag.Parse` stops at `omp`; `--model`/`--session-dir` land in `flag.Args()`; the launch binding silently never fires.
**Why it happens:** Go flag semantics: parsing stops at the first non-flag argument.
**How to avoid:** the omp dispatch must re-tokenize the post-`omp` args (and tolerate unknown flags — the client may add flags across versions). Corpus/unit rows lock `omp acp --model p/m` parsing.
**Warning signs:** the model picker selection never applies at launch; logs never show the launch binding.

### Pitfall 4: advertising delete/fork/close capabilities changes Orca behavior for the worse
**What goes wrong:** advertising `sessionCapabilities.delete` makes Orca call `session/delete` on its disposable listing sessions (today it uses throwaway folders BECAUSE omp does not advertise delete) — our no-op would leave folders accumulating client-side expectations; advertising `fork` invites conversation-copy fork RPCs we decline.
**How to avoid:** keep delete/fork/list/load/resume UNADVERTISED; the graceful arms serve spec-loose clients only. `close` advertisement is the one honest-improvement candidate (we implement it; Orca never checks it) — planner decision.
**Warning signs:** disposable `<dataDir>/acp-listing-sessions/omp` behavior changes after an advertisement change.

### Pitfall 5: the close timeout races the client's close timeout
**What goes wrong:** Orca's `session/close` allows 10 s with `onTimeout:'cancel'`; our `closeTurnWait` is exactly 10 s (lifecycle.go:50). A turn at the boundary can make both fire together.
**How to avoid:** note it in docs/troubleshooting; the client tolerates the timeout error class (`AcpRequestTimeoutError` caught) [VERIFIED: orca child-stop fetch this session]; no behavior change required — but a fix to WR-01 that widens close latency should re-check this budget.
**Warning signs:** manual E2E Stop shows a close error toast intermittently.

### Pitfall 6: the VM script must not require a dev toolchain on the VM
**What goes wrong:** assuming mise/Go on the "clean" VM.
**How to avoid:** the deliverable is a static binary + the script; the script documents build-or-fetch for the binary and asserts runtime-only deps (secret-tool, keyring, desktop install).
**Warning signs:** the script's first line installs a toolchain.

### Pitfall 7: the fake corpus is back-leg-only — ACP-side arms need the composed tier
**What goes wrong:** adding `session/delete`/fork rows as zcode.Conn↔fake scenarios (wrong leg — the fake speaks the ZCode dialect, delete/fork are ACP-front methods).
**How to avoid:** ACP-side arms ride the composed harness (SDK client ⇄ tee ⇄ adapter ⇄ front ⇄ fake) like `TestFullCycle`; the fake-corpus additions for this phase are the backend-leg shapes only where one exists (none for delete).
**Warning signs:** a new scenario JSON trying to script `session/delete` on the back leg.

### Pitfall 8: never `grep -c` on the minified bundle; `-e` for leading-dash patterns
Standing dialect-doc hygiene for any bundle re-verification this phase (app-server-dialect.md §2/§6): occurrence counts via `grep -o <pattern> | wc -l`; `grep -o -e '-32603'` (a bare leading-dash pattern is silently parsed as options, rc=1).

## Code Examples

### The exact SDK arms Phase 7 touches (all read from the pinned module cache this session)

```go
// Source: github.com/coder/acp-go-sdk@v0.20.0 agent_gen.go:191-206 (module cache)
case AgentMethodSessionFork:
    var p UnstableForkSessionRequest
    ...
    impl, ok := a.agent.(AgentUnstableForkSession)
    if !ok {
        return nil, NewMethodNotFound(method) // ← the graceful fork decline: -32601 {"method":"session/fork"}
    }

// Source: errors.go:58-59
func NewMethodNotFound(method string) *RequestError {
    return NewRequestError(-32601, "Method not found", map[string]string{"method": method})
}

// Source: types_gen.go:2902-2909 — DeleteSessionResponse: the empty result
type DeleteSessionResponse struct {
    Meta Meta `json:"_meta,omitempty"`
}
```

### The detail-carrying refusal surface the `/model` layer needs

```go
// The Hb matchers read ONLY: code -32603 with data.details (string).
// Source: orca v1.4.262 omp-dialect.ts — promptErrorDataSchema = z.looseObject({ details: z.string() })
// Our seam currently cannot express data payloads (wireError passes nil) — the phase adds e.g.:
return nil, acpport.NewInternalWithData(
    "Unknown ACP model: %s", name) // → wire: {"code":-32603,"message":...,"data":{"details":"Unknown ACP model: <name>"}}
```

### The current delete arm being replaced (verbatim)

```go
// Source: internal/acpport/agent.go:417-423 (read this session)
// DeleteSession is unadvertised in Phase 4 — clean -32601 default.
func (a *agentAdapter) DeleteSession(
    ctx context.Context,
    params acp.DeleteSessionRequest,
) (acp.DeleteSessionResponse, error) {
    return acp.DeleteSessionResponse{}, unadvertised(acp.AgentMethodSessionDelete)
}
// Phase 7: delegate to the front (CloseSession semantics where registered; immediate {} otherwise).
```

### The launch argv the wrapper must parse

```
# Source: acp-client-dialect.md Launch contract (source-corrected section)
zcode-acp omp acp [--plugin-dir <dir>] [--model <provider/model>]     # chat launch
zcode-acp omp acp --session-dir <folder>                              # disposable discovery launch
zcode-acp omp --version                                               # the probe: prefixArgs + --version, no card args
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `session/delete` answered `-32601` | Graceful `{}` no-op | This phase (SESS-02) | Spec-loose clients keep working; Orca unchanged (capability-gated) |
| Root-only stdlib-flag CLI | omp subcommand dispatch | This phase (D-03) | The card argv contract becomes claimable (`launchesWithModel:true`) |
| Version `0.0.0-dev` | Release stamp ≥ 18.1.9 | This phase | The version probe passes; the card is installable in Orca |

**Deprecated/outdated:** the STACK-table cobra decision (2026-11-19) was superseded by the shipped stdlib-flag CLI — Phase 7 either adopts cobra consciously (updating the `tidy-diff` gate expectation) or hand-rolls the tiny dispatch; both documented above.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Orca passes the config-option value id (`providerId/modelId`) verbatim as `--model` | Pattern 5 | The launch binding mis-parses an unexpected id shape → honest parse error logged, model not bound; E2E observes and corrects (the checklist includes this observation) |
| A2 | A successful `/model` switch may reply with a short confirmation text chunk (client classifies only refusals) | Pattern 4 | Cosmetic only; any end_turn reply is compliant |
| A3 | The wrapper advertising no `models` field keeps Orca's picker on the config-options path (never the `models` path) | Patterns 3-5 | If Orca's UI expects the `models` extension for its picker, model switching falls to `/model` prompts exclusively — which D-03 makes us implement anyway; QA-03 observes the real picker behavior |
| A4 | The exact release-stamp form for ≥18.1.9 (bare `18.1.9` vs `18.1.9+build`) | Pattern 5 / Pitfall 2 | Any semver ≥ 18.1.9 satisfies the probe; the form is cosmetic |
| A5 | `session/close` advertisement (`sessionCapabilities.close: {}`) is safe for spec-conformant clients | Pattern 3 / Pitfall 4 | It is an optional advertisement of implemented functionality; worst case a client calls close more often — implemented path |

## Open Questions (RESOLVED)

1. **The settings-store persistence path for `agentCmdOverrides`** (known-unknown inherited from the dialect doc)
   - What we know: the settings key and shape are source-verified; the persistence location (Electron storage/orcad) is not; configuration happens via the Settings UI.
   - What's unclear: whether the recipe can name a file for headless provisioning.
   - Resolution (implemented): the recipe documents the Settings-UI path and carries the known-unknown label — 07-08 (DOCS-02); live confirmation at the QA-03 gate (07-09 T3 human-check).
2. **What id shape Orca actually passes as `--model`** (A1)
   - Resolution (implemented): the wrapper binds `--model` by parsing provider/model and fails honestly on other shapes — 07-04 (launch binding reuse); observing the wrapper's launch log is a 07-09 T3 checklist observation item (COVERAGE flagged row 8, verification: backstop).
3. **Orca picker behavior with configOptions but no `models` field** (A3)
   - Resolution (implemented): observed at the QA-03 gate, never assumed — 07-09 T3 observation item (COVERAGE flagged row 8); the `/model` prompt path is implemented regardless — 07-05 (belt and braces per D-03).
4. **Whether the release flow (goreleaser config) lands in this phase** — mise.toml and README both defer it to "the Phase 7 release flow", but CONTEXT's boundary names only the version floor and recipe
   - Resolution (implemented): planner scoped the minimum — the static floor stamp `18.1.9+zacp.0` with the `-ldflags` override seam for a future release flow — 07-04 (version stamp task); full goreleaser wiring is NOT planned in this phase (CONTEXT's boundary names only the version floor and recipe).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build/test | ✓ | go1.27.3 (mise pins 1.27) | — |
| mise | CI gate parity | ✓ | 2026.10.17 | — |
| golangci-lint | `mise run ci` | ✓ (mise-managed, pinned 2.15.0) | 2.15.0 | — |
| node | experiments `node --test` | ✓ | v26.13.1 | — |
| goreleaser | release flow (deferred) | ✓ (mise-managed, pinned 2.20.2; no `.goreleaser.yaml` yet) | 2.20.2 | ldflags version stamp only |
| ZCode desktop 3.18.2 | VM scenario, real E2E | ✓ (`~/Applications/zcode-3.18.2/squashfs-root/`, pristine; `"session/delete"` ×0 re-verified this session) | 3.18.2 | — |
| Orca.AppImage | QA-03 manual E2E | ✓ (`~/Applications/Orca.AppImage`) | v1.4.26x build window | — |
| secret-tool | coding-plan path, VM scenario | ✓ | /usr/bin/secret-tool | — |
| Real keyring + desktop login | coding-plan authenticate | ✓ (phase-6 UAT PASSED on this machine) | — | api-key path needs no keyring |
| uvx / acp-tck | (not in phase scope) | ✗ | — | Not needed: D-04 defines the in-repo conformance suite as the bar |
| Clean Ubuntu GNOME VM | QA-02 field run | ✗ ( MANUAL tier — the user provisions it) | — | The script is the deliverable (D-05); one documented user run is the field proof |
| Self-hosted GNOME runner | e2e-matrix workflow | ✗ (not registered; workflow is an inert placeholder) | — | Stays inert; D-05 keeps the VM run manual |

**Missing dependencies with no fallback:** none blocking — the VM and the runner are manual-tier/inert by design (D-05/D-06).
**Missing dependencies with fallback:** uvx/acp-tck — intentionally unused (the named in-repo conformance suite is the done-bar per D-04).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (`go test -race`) + goldie-free golden corpus (`fake.CompareGolden`) + `node --test` (experiments) |
| Config file | none (convention-driven); corpus at `internal/zcode/fake/testdata/` |
| Quick run command | `go test -race -count=1 -run 'TestConformance|TestFullCycle' ./internal/acpfront/` |
| Full suite command | `mise run ci` (build + vet + lint + `go test -race -count=1 ./...` + `node --test experiments/`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SESS-02a | delete → `{}` no-op | unit + composed | `go test -race -count=1 -run DeleteSession ./internal/...` | ❌ Wave 0 (new rows) |
| SESS-02b | fork → `-32601 {"method":"session/fork"}` | composed (byte-lock) | same pattern | ❌ Wave 0 |
| SESS-03a | every DISC-02 method: implemented-or-graceful row | composed corpus rows | `go test -race -count=1 ./internal/acpfront/ ./internal/acpport/` | ❌ Wave 0 |
| SESS-03b | `/model` refusals byte-exact (`-32603` + `data.details`) | unit + composed | same | ❌ Wave 0 |
| SESS-03c | `--model` launch binding | unit (CLI parse) + composed | `go test -race -count=1 -run Omp ./cmd/zcode-acp/` | ❌ Wave 0 |
| QA-01 | the named conformance arc green in CI | composed E2E | `mise run ci` (the named suite inside) | ❌ Wave 0 |
| QA-02 | VM script asserts discovery/config/run/no-keyring path | MANUAL tier (script) | `scripts/vm-*` run by the user | ❌ new file |
| QA-03 | real Orca card, both auth paths | MANUAL (user gate) | — (UAT + ledger recording) | n/a |
| WR-01/04/05 | lifecycle consistency | unit (-race, interleavings) | `go test -race -count=1 -run 'Lifecycle|Close|Resume' ./internal/acpfront/` | existing files extended |
| IN-01..04 | advisory cleanups | unit | `go test -race -count=1 ./internal/hostsvc/ ./internal/interact/ ./internal/acpport/` | existing files extended |

### Sampling Rate
- **Per task commit:** the package's focused `-race` run
- **Per wave merge:** `mise run ci`
- **Phase gate:** full suite green + the guarded real-tier compile/skip check (`go vet -tags realdesktop ./test/realdesktop/`) + QA-03 recorded PASS

### Wave 0 Gaps
- [ ] New corpus/conformance rows (SESS-02/03 arms) + their goldens
- [ ] The named conformance suite test file
- [ ] omp CLI parse unit tests (`cmd/zcode-acp`)
- [ ] No framework install needed (infrastructure exists)

## Security Domain

> `security_enforcement: true`, ASVS level 1 (config.json). Phase 7's security-relevant surface: the CLI parses untrusted argv (the card contract), docs publish operational guidance, and the VM script drives real processes.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes (both auth paths exercised in E2E/docs) | Existing: authenticate chains (AUTH-01/02), keyring sentinels; docs must never instruct exporting key material |
| V3 Session Management | yes (delete/fork/close arms) | Existing teardown contract; graceful arms never widen the session surface |
| V4 Access Control | no | No new authorization surface |
| V5 Input Validation | yes | CLI argv validated (`providerId/modelId` cut, unknown flags tolerated-but-ignored, honest errors); `/model` selector validated against the confirmed option list only — never forwarded unvalidated to the backend |
| V6 Cryptography | no | No new crypto; keyring decrypt path unchanged |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Prompt-text injection via `/model <name>` (untrusted string reaches the backend) | Tampering | Validate against the confirmed listed values FIRST (the T-06-17 posture: no backend call carries an unvalidated value); refuse with the exact Hb prefix otherwise |
| Secret leakage through docs/captures (README examples, VM script logs, E2E captures) | Information disclosure | The scrub-first capture pipeline + `scan-secrets` gate on tree AND staged diff (the 06 precedent); docs use env-var names, never values |
| Log/argv echo of credential material in the new CLI paths | Information disclosure | The established rule: keys in-memory only, never logged; slog carries shapes/counts only |
| VM script running untrusted downloads | Elevation | Script fetches only the checksum-verified official artifacts (the CDN sha512 verification precedent from install-discovery.md) or uses an existing install |

## Sources

### Primary (HIGH confidence)
- Module cache read this session: `github.com/coder/acp-go-sdk@v0.20.0` — `agent_gen.go` dispatch (fork arm, optional interfaces), `types_gen.go` (DeleteSessionResponse, UnstableForkSessionRequest/Response, SessionCapabilities, Agent interface), `constants_gen.go` (method map — no `session/set_model`), `errors.go` (`NewMethodNotFound`), `connection.go:663` (unknown-method fallback)
- In-repo graded documents: `docs/protocol/acp-client-dialect.md` (Hb strings, launch contract, version floor, method range, capability gating), `docs/protocol/app-server-dialect.md` (§2 setModel quirk, §3 interaction surface, §7 `"session/delete"` ×0, §8 snapshot/paired-env facts), `docs/protocol/install-discovery.md` (layout matrix, spawn line, revision formula)
- Code read this session (all `[VERIFIED: <path>:<lines>]` in place): `internal/acpfront/{front.go,lifecycle.go,configoptions.go}`, `internal/acpport/{agent.go,types.go,pin_test.go}`, `internal/interact/keyringprobe.go`, `internal/hostsvc/snapshot.go`, `internal/supervisor/spawn.go`, `cmd/zcode-acp/main.go`, `cmd/fake-app-server/main.go`, `internal/zcode/fake/{golden_test.go,scenario.go}` + `testdata/` listings, `internal/acpfront/{harness_test.go,fullcycle_test.go}`, `go.mod`, `mise.toml`, `.github/workflows/e2e-matrix.yml`
- Fetched this session: github.com/stablyai/orca @v1.4.262 — `src/main/acp/acp-dialects/omp-dialect.ts` (verbatim Hb strings + `promptErrorDataSchema`), `src/main/acp/conversation-copy/acp-conversation-copy.ts` (`forkAdvertised` gate), `src/main/acp/turns/acp-structured-child-stop.ts` (child-stop close gating), `src/main/acp/process/acp-agent-connection.ts` (`closeSession` 10 s ungated, `listSessions` advertisement comment)
- Fetched this session: agentclientprotocol.com — `/protocol/v1/session-delete` (shape `{}` + gating + silent-success semantics), `/protocol/overview` (method tiering)

### Secondary (MEDIUM confidence)
- WebSearch corroboration: session-delete stabilization (June 2026) and capability-gating summaries (agentclientprotocol.com schema page)
- `.planning/phases/06-*/` artifacts (REVIEW, REVIEW-DISPOSITION, VERIFICATION, UAT) — the routed findings' fix sketches and the UAT recording precedent (planning-tier, code anchors re-verified in current source)

### Tertiary (LOW confidence)
- None material; A1/A3 assumptions are E2E-observed items by design (see Assumptions Log)

## Metadata

**Confidence breakdown:**
- SESS-02 shapes: HIGH — SDK module cache + official spec + Orca source, three independent confirmations; bundle absence re-run
- SESS-03 gap table + Hb contracts: HIGH — in-repo V-SRC-graded dialect doc + fresh Orca source fetch agreeing verbatim; adapter status read line-by-line
- CLI/launch mechanics: HIGH for the contract (dialect doc source-corrected sections), MEDIUM for the runtime id shape (A1 — E2E item)
- QA-01 corpus inventory: HIGH — direct reads
- QA-02 script contents: HIGH (sentinels + env facts read in source; the VM run itself is manual by decision)
- Routed items: HIGH — every anchor re-read in current code this session

**Research date:** 2026-11-22
**Valid until:** 2026-12-22 (stable — pinned SDK + graded in-repo dialect docs; re-check the Orca source tags only if the client updates past the v1.4.26x build window)

---
*Phase: 7 — Hardening, VM & Manual E2E, Documentation*
*Researched: 2026-11-22*
