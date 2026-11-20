# Roadmap: zcode-acp

## Overview

zcode-acp turns a Go stdio binary into an honest ACP v1 server in front of the ZCode
Desktop app-server, so any ACP client can run full turns — streamed text and thoughts,
tool calls, permission decisions, cancellation — on a real z.ai account via api-key or
the official OAuth coding-plan path. The road is: establish verified wire truth from
source and experiments first (Phase 1), put the repo under strict goswitch-style gates
before any code (Phase 2), build the ZCode back leg with a scriptable fake app-server so
everything after is CI-testable without the desktop bundle (Phase 3), stand up a minimal
ACP chat loop (Phase 4), complete the chat cycle with interactions, tools, permissions and
the api-key auth path (Phase 5), add the OAuth coding-plan chain and the registry-driven
model/thought-level picker (Phase 6), then close the milestone by meeting the done-v1 bar:
CI protocol conformance, a reproducible VM scenario, manual E2E in a real chat-UI client,
and English documentation including a launch recipe for ACP hosts that only accept preset
agents (Phase 7).

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

- [x] **Phase 1: Research & Protocol Documentation** - Document the ZCode app-server dialect, the client ACP dialect, install discovery, and workspaceKey semantics from source, with provenance labels (completed 2026-11-19)
- [x] **Phase 2: Scaffold & CI Bootstrap** - goswitch-style repo scaffold: strict pinned linter, GitHub Actions, protected master (completed 2026-11-19)
- [x] **Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor** - JSONL codec, scriptable fake app-server with golden transcripts in CI, desktop discovery + process supervisor, turn state machine (completed 2026-11-20)
- [x] **Phase 4: ACP Front — Minimal Chat Loop** - initialize, session/new (+ setModel workaround), streamed prompt turns with thoughts and cancel (completed 2026-11-20)
- [ ] **Phase 5: Interactions & Full Chat Cycle (api-key)** - Host-managed interaction responder, tool-call streaming, permission bridging, api-key auth, resume/close
- [ ] **Phase 6: OAuth Coding-Plan Path & Model Picker** - Official OAuth chain end-to-end (snapshot, cipher, PAT), both auth methods working, registry-driven configOptions
- [ ] **Phase 7: Hardening, VM & Manual E2E, Documentation** - CI protocol conformance, dialect compatibility no-ops, VM scenario, manual E2E, English docs + preset-agent launch recipe

## Phase Details

### Phase 1: Research & Protocol Documentation
**Goal**: As a later-phase implementer on this repo, I want to the app-server dialect, the chat-UI client's ACP behavior, desktop install layouts, and workspaceKey semantics documented from source and experiments, so that every later phase builds on verified wire truth instead of assumptions.
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: DISC-01, DISC-02, DISC-03, DISC-04, DISC-05
**Success Criteria** (what must be TRUE):
  1. `docs/` protocol documentation describes the app-server JSONL event surface — including `session/subscribe`/event-stream availability and the turn-completion terminal signals — with every claim labeled V-SRC or V-EXP against the shipped 3.18.2 binary
  2. A dialect document names the chosen preset-agent identity (omp vs opencode vs opencode2) with its version floor and env quirks, and enumerates both the load-bearing method set at card startup/per turn and the full range of ACP methods that client sends
  3. A discovery matrix documents how to locate a ZCode desktop install across AppImage / deb / unpacked layouts, including what auto-update does to the builtin revision string
  4. `workspaceKey` derivation semantics are documented with both a source citation and a recorded experiment
  5. A reference-implementation comparison under `docs/research/` records what the TypeScript zcode-acp project covers (elicitation, slash commands, resume/fork, attachments), what to adopt, and what to avoid
**Plans**: 6/6 plans complete

Plans:
**Wave 1**
- [x] 01-01-PLAN.md — Documentation-pipeline walking skeleton: validation gates, provenance legend, scrub-first capture pipeline, E3 experiment + workspace-key.md (DISC-04)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 01-02-PLAN.md — E1 session/subscribe probe + E2 turn-completion experiment, scrubbed captures (DISC-01 evidence)
- [x] 01-04-PLAN.md — Orca chunk re-verification + E5 static answers + docs/protocol/acp-client-dialect.md (DISC-02)
- [x] 01-05-PLAN.md — E4 install-layout enumeration + docs/protocol/install-discovery.md (DISC-03)
- [x] 01-06-PLAN.md — Reference-repo study + docs/research/reference-adapters.md (DISC-05)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 01-03-PLAN.md — docs/protocol/app-server-dialect.md: event surface, subscribe verdict, terminal signals, drift baseline (DISC-01)

### Phase 2: Scaffold & CI Bootstrap
**Goal**: As a later-phase implementer on this repo, I want to have strict quality gates — strict pinned lint, CI, and a protected `master` modeled on Djarvur/goswitch — enforced from the very first code commit onward, so that all feature code lands under the same gates and later phases merge only through green PRs.
**Mode:** mvp
**Depends on**: Nothing (no technical dependency on Phase 1; sequenced second by project mandate so all code lands under strict gates)
**Requirements**: SCAFF-01, SCAFF-02, SCAFF-03
**Success Criteria** (what must be TRUE):
  1. A fresh checkout passes the strict pinned golangci-lint config (goswitch-style `default: all` v2 config, exact version pinned via mise); any violation fails the lint target
  2. A PR to the repository triggers GitHub Actions running lint, test, build, and govulncheck, with toolchain versions driven by mise
  3. `master` is protected: direct pushes are rejected, merges require the configured checks, and the protection setup is documented in the repo
**Plans**: 5/5 plans complete

Plans:
**Wave 1**
- [x] 02-01-PLAN.md — Tracer: module skeleton (go.mod, cmd/zcode-acp --version) + mise pins + goswitch strict lint config; `mise run lint` green end-to-end + planted-violation red-proof (SCAFF-01, D-02, D-03, D-06)
- [x] 02-02-PLAN.md — TDD scrubber hardening: WR-03 boundary-aware home rewrite + IN-04 capture-header guard, RED→GREEN node:test (D-05)
- [x] 02-03-PLAN.md — Sibling review fixes: scan-secrets WR-01/IN-03, check-docs IN-03, probe WR-04, dialect-doc recipes WR-02/IN-01/IN-02 (D-05)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 02-04-PLAN.md — Four goswitch workflow shapes (pr-sanity / release / e2e-matrix / security-scheduled) + README CI/protection section + push + observed green pr-sanity run (SCAFF-02, D-04, D-05, D-06)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 02-05-PLAN.md — Branch protection applied LAST via gh api + GET verify + GH006 negative push test + branching_strategy flip to `phase` (SCAFF-03, D-01)

### Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor
**Goal**: As a later-phase implementer on this repo, I want to the wrapper's back leg — a JSONL codec for the app-server dialect, a scriptable fake app-server with golden transcripts in CI, a desktop discovery + process supervisor, and a turn state machine, so that every later phase is testable without the real desktop bundle.
**Mode:** mvp
**Depends on**: Phase 1 (dialect docs, discovery matrix), Phase 2 (CI gates)
**Requirements**: BRIDGE-01, BRIDGE-02, BRIDGE-03, BRIDGE-04
**Success Criteria** (what must be TRUE):
  1. The codec round-trips the app-server JSONL dialect in unit tests: frames without a `jsonrpc` key, the dialect's own error codes, ordered single-writer output — and a JSON-RPC-shaped frame never reaches the child
  2. CI runs golden JSONL transcript tests against the scriptable fake app-server (in-process over pipes plus a standalone binary), exercising the protocol with no desktop bundle present
  3. On a machine with ZCode desktop installed, the supervisor locates the install per the Phase 1 matrix, spawns the app-server child with the correct environment, and kills the whole process group on EOF/exit leaving no orphaned Electron processes; when no install is found, startup fails with actionable diagnostics naming what was searched
  4. Against the fake app-server, a fire-and-forget `session/send` is lifted into a synchronous turn that completes exactly once on multi-signal turn-completion detection and honors cancel mid-turn
**Plans**: 9/9 plans complete

Plans:
**Wave 1**
- [x] 03-01-PLAN.md — Tracer (TDD): dialect round-trip end-to-end — envelope + two-lane writer + Conn demux + in-process fake + happy-turn golden replay over io.Pipe (BRIDGE-01, BRIDGE-02, D-01, D-05)
- [x] 03-04-PLAN.md — Discovery (TDD): Phase-1 matrix probes with seams, first-match order, every-probe diagnostics, mount rejection, builtin revision (BRIDGE-03, D-03)

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 03-02-PLAN.md — Codec hardening: WR-02 demux lock, id type-safety, single-writer ordering, bypass precedence, line buffer, router deadlines + credential-cipher tracked-no-answer (BRIDGE-01, WR-02, D-05)
- [x] 03-03-PLAN.md — Golden corpus: fail-loud scenario loader + scenarios 2-10 + byte-deterministic goldens + EOF failure signature (BRIDGE-02, D-01)
- [x] 03-05-PLAN.md — Turn machine part 1 (TDD): TurnWatcher poll + quiet-window gates + Machine.Run exactly-once completion, carrier-tolerant terminal classification (BRIDGE-04, D-02, D-04)
- [x] 03-06-PLAN.md — Standalone fake-app-server binary + helper-reexec exec-plumbing tier + EOF drain contract (BRIDGE-02, D-01)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 03-07-PLAN.md — Supervisor lifecycle: canonical spawn line + allowlisted/scrubbed env + Setpgid, staged shutdown + group kill + crash-window restart, env-guarded real-desktop manual path (BRIDGE-03, D-03)
- [x] 03-08-PLAN.md — Turn machine part 2 (TDD): bypass-lane cancel exactly-once, -32010 double-send guard, corpus completion variants, credential-cipher honesty (BRIDGE-04, D-02, D-05)

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 03-09-PLAN.md — E2E exec chain (supervisor + fake binary + Conn + turn machine) + full phase gate `mise run ci` (BRIDGE-01..04)

### Phase 4: ACP Front — Minimal Chat Loop
**Goal**: As a v1 chat UI client, I want to open a session against ZCode and run a streamed prompt turn end-to-end, with live text and thought chunks and correct cancellation, against the app-server, so that the minimal chat loop is proven on the real ACP boundary before auth and interaction complexity accrete.
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: CORE-01, CORE-02, CORE-03, CORE-04, CORE-05
**Success Criteria** (what must be TRUE):
  1. `initialize` replies `protocolVersion: 1` with an honest capability set (nothing fs/terminal-related), real `agentInfo`, and both advertised authMethods (`zai-api-key`, `zai-individual-coding-plan`)
  2. `session/new{cwd}` creates a backend session, a follow-up setModel call carrying model + reasoning level is visible on the wire (fake transcript asserts the payload), and `mcpServers` is accepted without error
  3. `session/prompt` streams `agent_message_chunk` deltas live while the turn runs and returns `stopReason` only after all pending updates have flushed
  4. When the backend emits reasoning events the client receives `agent_thought_chunk` updates; none are synthesized when it does not
  5. `session/cancel` during a turn stops it and the prompt call is answered exactly once with `stopReason: "cancelled"`
**Plans**: 6/6 plans complete

Plans:
**Wave 1**
- [x] 04-01-PLAN.md — SDK dep lands (D-01 audit PASS, exact v0.20.0 pin + tidy-diff gate expectation) + subscribe/setModel method constants + conn notification observer fan-out (TDD)

**Wave 2** *(blocked on Wave 1)*
- [x] 04-02-PLAN.md — Enablement (TDD): acpport seam package (SDK imports confined, D-02-honest initialize assembly, -32601 unadvertised defaults) + internal/core session registry + setmodel-followup corpus row + front-tier test harness + seam boundary test

**Wave 3** *(blocked on Wave 2)*
- [x] 04-06-PLAN.md — Tracer (TDD): minimal chat loop end-to-end — SDK test client ↔ acpport ↔ acpfront ↔ machine ↔ fake over pipes (honest initialize, create→subscribe→setModel on wire, live text streaming, cancelled exactly-once) + cancel-mid-turn front-preamble corpus

**Wave 4** *(blocked on Wave 3; 04-03 and 04-04 run in parallel)*
- [x] 04-03-PLAN.md — CORE-01/CORE-02 contract locks: initialize honesty exact-shape lock, authenticate honest error (RED), unadvertised -32601 defaults, relative-cwd validation (RED), mcpServers accept-and-ignored, setModel payload verbatim
- [x] 04-04-PLAN.md — CORE-03/CORE-04 streaming locks: reasoning→thought chunks (RED) + never-synthesized negative, flush-before-stopReason ordering proof across both completion variants, zero-delta turn honesty

**Wave 5** *(blocked on Wave 4 — shares front.go with 04-03)*
- [x] 04-05-PLAN.md — CORE-05 cancel locks: cancelled exactly-once never -32800, second-prompt rejection (RED), cancel-after-complete no-op + cmd/zcode-acp first real wiring (supervisor override seam, model flags) + binary-level E2E

### Phase 5: Interactions & Full Chat Cycle (api-key)
**Goal**: On a real z.ai account via api-key, a client gets a complete chat cycle — streamed tool calls, permission decisions answered from the client, working session lifecycle — with the host-managed interaction responder always on.
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: AUTH-01, CORE-06, SESS-01, TOOL-01, TOOL-02
**Success Criteria** (what must be TRUE):
  1. Every server→host `interaction/*` request is answered promptly: re-announced requests dedupe, per-method deadlines fire, and unknown methods receive protocol-error replies instead of crashing the wrapper
  2. Tool use streams to the client as `tool_call` / `tool_call_update` with title, kind, status transitions, and file diffs for edit tools
  3. A backend permission request reaches the client as `session/request_permission` with allow/reject options; the chosen outcome is forwarded to the backend, and an ACP-side cancel resolves pending requests as `cancelled`
  4. `authenticate{zai-api-key}` on a real account enables working turns, with success verified by reading registry state back — not by trusting the `provider/updateAccountConfig` ack
  5. `session/resume` reattaches a session without replay and `session/close` cleans up the child-side session; a full cycle — new → prompt with tool calls and a permission decision → resume → close — runs end-to-end on api-key
**Plans**: 8/9 plans executed
Plans:
**Wave 1**
- [x] 05-01-PLAN.md — CORE-06 interaction responder policy object (deadlines, requestId dedupe, unknown -32601, runtime-prefs migration) + fake re-announce extension

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 05-02-PLAN.md — TOOL-02 permission bridge end-to-end (option mapping, outcome forwarding verbatim, cancel → -32021, expiry → deny) + WR-02 atomic sink
- [x] 05-03-PLAN.md — WR-01 router-event fan-out + WR-07 exposed observer removes + IN-01/IN-02 (multi-session enablement for teardown)

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 05-04-PLAN.md — TOOL-01 tool streaming (tool.updated → tool_call/update, edit/write diffs) + WR-03 bounded pump + WR-06/IN-05

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 05-05-PLAN.md — SESS-01 resume/close with teardown (WR-04/WR-05, IN-03/IN-04) + adapter delegates

**Wave 5** *(blocked on Wave 4 completion)*
- [x] 05-06-PLAN.md — AUTH-01 backend: hostsvc snapshot push (exact-revision builtin), awaitRegistryContains read-back, runtime-headers answer

**Wave 6** *(blocked on Wave 5 completion)*
- [x] 05-07-PLAN.md — AUTH-01 front: authenticate (read-back-proven, no-key honest error) + D-06 CredentialDecrypt seam + cmd wiring (ZCODE_ACP_API_KEY, responder OnRequest)

**Wave 7** *(blocked on Wave 6 completion)*
- [x] 05-08-PLAN.md — Full-cycle fake E2E: authenticate → new → prompt(tool+permission) → resume → close

**Wave 8** *(blocked on Wave 7 completion)*
- [ ] 05-09-PLAN.md — Guarded real tier (TestAPIKeyFullCycle, double guard) + quota ledger; real run = end-of-phase verification

### Phase 6: OAuth Coding-Plan Path & Model Picker
**Goal**: Users on an individual coding plan can authenticate through the official OAuth chain end-to-end, and any client can see and switch model and thought level from the live provider registry.
**Mode:** mvp
**Depends on**: Phase 5 (interaction responders; supervisor and setModel mechanics arrive transitively from Phases 3–4)
**Requirements**: AUTH-02, AUTH-03, AUTH-04, MODEL-01, MODEL-02
**Success Criteria** (what must be TRUE):
  1. `authenticate{zai-individual-coding-plan}` completes the official chain on a real account — registry snapshot accepted with the exact `zcode-builtin:<rev>:<sha256>` revision (verified via registry state, not the ack), credential cipher answered by decrypting the Secret Service key, PAT minted via the api.z.ai chain, `requestAuth` returned — and subsequent turns run
  2. Both advertised authMethods authenticate successfully, and account models appear after auth
  3. On a machine without a keyring (headless/CI), auth fails fast with a clear actionable error and never hangs a turn
  4. After auth, `session/new` returns `configOptions` covering model and thought level, built from the live provider registry
  5. A client-side config-option change maps to `session/setModel` / `session/setThoughtLevel` on the wire, re-emits the full `configOptions` list, and `currentValue` always reflects the real session state
**Plans**: TBD

### Phase 7: Hardening, VM & Manual E2E, Documentation
**Goal**: The done-v1 bar is met: full CI protocol conformance, a reproducible clean-VM scenario, manual E2E confirmed in a real chat-UI client on a real account through both auth paths, and English documentation including the launch recipe for ACP hosts that only accept preset agents.
**Mode:** mvp
**Depends on**: Phase 6 (all functional phases complete)
**Requirements**: DOCS-01, DOCS-02, DOCS-03, QA-01, QA-02, QA-03, SESS-02, SESS-03
**Success Criteria** (what must be TRUE):
  1. The CI protocol conformance suite is green: the fake app-server drives one scripted scenario covering initialize → session/new → prompt streaming → tool calls → permissions → cancel
  2. Clients that send `session/delete` keep working (graceful no-op) and `session/fork` is handled gracefully — never a crash path
  3. Every ACP method Orca's client calls at card startup and during a turn (per the Phase 1 dialect document) is implemented or gracefully handled, so a card works end-to-end
  4. A clean Ubuntu GNOME VM scenario — install, configure, run — is scripted in the repo and reproducible, including the no-keyring error path
  5. Manual E2E confirmed and documentation published: a working agent card in a real chat-UI client (Orca) on Ubuntu GNOME on a real account through both auth paths, backed by an English README (what it is, install, configuration, both auth paths, troubleshooting), a launch recipe for ACP hosts that only accept preset agents (binary naming, version floor, env, PATH shim), and `docs/` protocol notes with the provenance statement and known limitations
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Research & Protocol Documentation | 6/6 | Complete    | 2026-11-19 |
| 2. Scaffold & CI Bootstrap | 5/5 | Complete    | 2026-11-19 |
| 3. ZCode Back Leg — Codec, Fake App-Server, Supervisor | 9/9 | Complete    | 2026-11-20 |
| 4. ACP Front — Minimal Chat Loop | 6/6 | Complete    | 2026-11-20 |
| 5. Interactions & Full Chat Cycle (api-key) | 8/9 | In Progress | - |
| 6. OAuth Coding-Plan Path & Model Picker | 0/TBD | Not started | - |
| 7. Hardening, VM & Manual E2E, Documentation | 0/TBD | Not started | - |

---
*Roadmap created: 2026-11-19*
