# Requirements: zcode-acp

**Defined:** 2026-11-19
**Core Value:** An ACP client can run a full turn end-to-end with ZCode — from `initialize` to streamed reply with tool-calls and permission requests — on a real z.ai account (api-key or OAuth coding plan).

## v1 Requirements

Requirements for the initial release. Each maps to roadmap phases.

### Discovery & Documentation (user-mandated first phase)

- [x] **DISC-01**: ZCode app-server 3.18.2 event surface documented from source (incl. `session/subscribe`/event-stream availability and turn-completion terminal signals), with V-SRC / V-EXP provenance labels per claim
- [x] **DISC-02**: Orca ACP dialect documented: the load-bearing method set at card startup and per turn, the full range of ACP methods Orca's client actually uses, and the preset-agent identity — **resolved: omp** (alternatives ruled out; compiled contract table in REFERENCE-ADAPTERS.md: floor 17.0.5, `--version` prints 18.1.9+, `agentCmdOverrides` wiring, Hb string contracts). Spot-check the table at first integration and answer the two remaining items: the `--session-dir` session-file format (can our Go agent write omp-compatible history files?) and omp modelDiscovery kind `session` behavior
- [x] **DISC-03**: ZCode desktop discovery matrix documented (AppImage / deb / unpacked layouts, auto-update effects on the builtin revision string)
- [x] **DISC-04**: `workspaceKey` derivation semantics documented (source + experiment)
- [x] **DISC-05**: Reference implementation studied and compared: github.com/william0wang/zcode-acp (TypeScript, Apache-2.0 → Go porting allowed with attribution, ~0.88.x, active) — it already implements and tests our hardest chain (keyring `enc:v2:os:` decryption via Secret Service, `requestCredentialCipher` in both paths, exact-revision snapshot, PAT flow): key files `src/backend/credential-store.ts`, `src/config/account-provider.ts`, `src/config/project-token.ts`, `src/handlers/server-requests.ts`, `src/backend/client.ts` + their tests as golden references. Also compare the weaker Rust/Python adapters (REFERENCE-ADAPTERS.md); findings written into `docs/research/`

### Scaffold (goswitch-style)

- [x] **SCAFF-01**: Go module scaffolded with strict pinned golangci-lint config (goswitch-style `default: all` v2 config, exact version pin)
- [ ] **SCAFF-02**: GitHub Actions CI running lint + test + build (+ govulncheck), mise-driven toolchain versions per goswitch pattern
- [ ] **SCAFF-03**: `master` branch protection configured and documented (required checks, PR-only workflow)

### ZCode Back Leg (bridge)

- [ ] **BRIDGE-01**: JSONL codec for the app-server dialect (no `jsonrpc` key, its error codes, ordered single-writer output) with unit tests
- [ ] **BRIDGE-02**: Fake app-server (in-process over pipes + standalone binary) and golden JSONL transcript tests running in CI
- [ ] **BRIDGE-03**: Desktop discovery + process supervisor: locate the ZCode desktop install, spawn `ELECTRON_RUN_AS_NODE=1 …/zcode.cjs app-server --stdio`, kill the whole process group on EOF/exit, fail loudly with actionable diagnostics when not found
- [ ] **BRIDGE-04**: Turn state machine lifting fire-and-forget `session/send` into a synchronous prompt lifecycle, with multi-signal turn-completion detection (per ARCHITECTURE.md) and cancel handling

### ACP Front (core loop)

- [ ] **CORE-01**: `initialize` replies `protocolVersion: 1`, honest capability set, real `agentInfo`, and two `authMethods` (`zai-api-key`, `zai-individual-coding-plan`)
- [ ] **CORE-02**: `session/new{cwd}` → `session/create` + follow-up `session/setModel` (reasoningLevel-loss workaround); `mcpServers` accepted and ignored
- [ ] **CORE-03**: `session/prompt` streams `agent_message_chunk` deltas live and replies `stopReason` only after all pending updates flush
- [ ] **CORE-04**: `agent_thought_chunk` streamed when the backend emits reasoning events (never synthesized)
- [ ] **CORE-05**: `session/cancel` stops the turn, resolves pending permission requests as `cancelled`, replies `stopReason: "cancelled"`
- [ ] **CORE-06**: Host-managed interaction responder always on: every `interaction/*` answered promptly (dedupe re-announcements, per-method deadlines, error replies to unknown methods instead of crash)

### Tools & Permissions

- [ ] **TOOL-01**: `tool_call` / `tool_call_update` streamed from app-server tool lifecycle events with title, kind, status transitions, and file diffs for edit tools
- [ ] **TOOL-02**: `session/request_permission` bridged to app-server `interaction/requestPermission` with `allow/reject` options and `cancelled` outcome on ACP-side cancel

### Auth (both paths end-to-end)

- [ ] **AUTH-01**: api-key path: `provider/updateAccountConfig` + `interaction/requestProviderRuntimeHeaders` answered correctly (registry state verified, not the ack)
- [ ] **AUTH-02**: OAuth individual coding plan: registry snapshot with exact `zcode-builtin:<rev>:<sha256>` revision string, `interaction/requestCredentialCipher` (Secret Service key decryption on Ubuntu GNOME), PAT mint via api.z.ai chain, `requestAuth` response — only official paths
- [ ] **AUTH-03**: `authenticate{methodId}` works for both advertised methods; account models appear after auth
- [ ] **AUTH-04**: Missing keyring (headless/CI) fails with a clear error and does not hang the turn

### Session Lifecycle (Orca-wide dialect)

- [ ] **SESS-01**: `session/resume` (no replay) and `session/close` (with child-side cleanup) supported
- [ ] **SESS-02**: `session/delete` answered as a graceful no-op and `session/fork` handled gracefully (never a crash path) — clients that call them keep working
- [ ] **SESS-03**: Full Orca ACP dialect coverage: every method Orca's client calls at card startup and during a turn (per DISC-02) is implemented or gracefully handled, so a card works end-to-end — including the persona CLI contract: `--model <id>` honored at launch (`launchesWithModel:true`) and the `/model <name>` slash prompt with the exact Hb string contracts (`Unknown ACP model: …`, `No model selected.`)

### Model & Thought-Level Picker

- [ ] **MODEL-01**: `session/new` response returns `configOptions` (model + thought level) built from the live provider registry after auth
- [ ] **MODEL-02**: `session/set_config_option` maps to `session/setModel` / `session/setThoughtLevel`, re-emits full `configOptions` on change, keeps `currentValue` honest

### Verification (done-v1 criteria)

- [ ] **QA-01**: CI protocol conformance suite green: fake app-server exercises initialize → session/new → prompt streaming → tool calls → permissions → cancel
- [ ] **QA-02**: VM test scenario scripted for a clean Ubuntu GNOME VM (install, configure, run) reproducible from the repo
- [ ] **QA-03**: Manual end-to-end confirmed: a working agent card in a real chat-UI client (Orca) on Ubuntu GNOME, on a real account, both auth paths

### Documentation (English)

- [ ] **DOCS-01**: README: what it is (honest ACP server for any ACP client), install, configuration, both auth paths, troubleshooting
- [ ] **DOCS-02**: README launch recipe for ACP hosts that only accept preset agents (binary naming, version floor, env, PATH shim) — the documented method of connecting to Orca's chat UI
- [ ] **DOCS-03**: `docs/`: protocol notes from DISC-01…04, provenance statement (no patching, no attestation forgery, PAT signed by the official binary; Apache-2.0 attribution for ported reference-adapter code), known limitations (no quota counter over OAuth; Start Plan does not work headless — browser/captcha required)

## v2 Requirements

Deferred to future releases. Tracked but not in current roadmap.

### Session History

- **SESSH-01**: `session/list` + `session/load` faithful replay (import existing threads into Zed)
- **SESSH-02**: `logout` + `sessionInfo_update` basics if users ask

### Extras

- **EXT-01**: Slash commands (`available_commands_update`) — pending a decision on which ZCode commands surface
- **EXT-02**: Non-Ubuntu environments (KDE, macOS, Windows keyring paths) — PR-driven
- **EXT-03**: zcode CLI as an alternative backend — PR-driven
- **EXT-04**: Additional persona subcommands for other preset-agent cards (opencode, opencode2, kimi, cursor, grok, copilot — see the card list in REFERENCE-ADAPTERS.md) — each card = a new cobra subcommand + a dialect-contract profile over the same bridge; the codec/supervisor/turn-machine stay untouched; PR-driven

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| `fs/*` and `terminal/*` agent-side support | ZCode touches the filesystem and shell itself; chat-UI clients advertise fs=false/terminal=false; calling unadvertised surfaces violates the spec |
| Re-implementing the agent (own tool loop / model HTTP calls) | Two implementations drift forever; kills desktop parity and the OAuth entitlement path |
| Patching the ZCode bundle (zcode.cjs) | Breaks update-proofness; setModel workaround already proven |
| Attestation / signature forgery for quota | ToS violation, fragile, explicitly against project honesty rules; only official PAT path |
| Invented quota/usage numbers | OAuth quota counter is not retrievable; report nothing rather than wrong numbers |
| 1:1 leak of proprietary app-server methods onto the ACP surface | Couples clients to a private dialect that changes between releases |
| Real `session/delete` server-side | No wire method exists in app-server; nothing to delete |
| ACP v2 draft surfaces | Unstable draft; splits the test matrix before v1 works |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DISC-01 | Phase 1 | Complete |
| DISC-02 | Phase 1 | Complete |
| DISC-03 | Phase 1 | Complete |
| DISC-04 | Phase 1 | Complete |
| DISC-05 | Phase 1 | Complete |
| SCAFF-01 | Phase 2 | Complete |
| SCAFF-02 | Phase 2 | Pending |
| SCAFF-03 | Phase 2 | Pending |
| BRIDGE-01 | Phase 3 | Pending |
| BRIDGE-02 | Phase 3 | Pending |
| BRIDGE-03 | Phase 3 | Pending |
| BRIDGE-04 | Phase 3 | Pending |
| CORE-01 | Phase 4 | Pending |
| CORE-02 | Phase 4 | Pending |
| CORE-03 | Phase 4 | Pending |
| CORE-04 | Phase 4 | Pending |
| CORE-05 | Phase 4 | Pending |
| AUTH-01 | Phase 5 | Pending |
| CORE-06 | Phase 5 | Pending |
| SESS-01 | Phase 5 | Pending |
| TOOL-01 | Phase 5 | Pending |
| TOOL-02 | Phase 5 | Pending |
| AUTH-02 | Phase 6 | Pending |
| AUTH-03 | Phase 6 | Pending |
| AUTH-04 | Phase 6 | Pending |
| MODEL-01 | Phase 6 | Pending |
| MODEL-02 | Phase 6 | Pending |
| DOCS-01 | Phase 7 | Pending |
| DOCS-02 | Phase 7 | Pending |
| DOCS-03 | Phase 7 | Pending |
| QA-01 | Phase 7 | Pending |
| QA-02 | Phase 7 | Pending |
| QA-03 | Phase 7 | Pending |
| SESS-02 | Phase 7 | Pending |
| SESS-03 | Phase 7 | Pending |

**Coverage:**
- v1 requirements: 35 total
- Mapped to phases: 35
- Unmapped: 0 ✓

> Note: the Coverage block previously stated "34 total". The actual count of
> listed v1 requirement IDs is 35; corrected during roadmap creation (2026-11-19).

---
*Requirements defined: 2026-11-19*
*Last updated: 2026-11-19 after roadmap creation (traceability populated)*
