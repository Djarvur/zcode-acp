# Project Research Summary

**Project:** zcode-acp
**Domain:** Go stdio protocol adapter — ACP v1 server wrapping ZCode Desktop app-server (JSONL, non-JSON-RPC)
**Researched:** 2026-11-19
**Confidence:** HIGH — ZCode-side facts experimentally verified on the real 3.18.2 binary (FINDINGS.md) and cross-checked against the public 3.17.x source tree; ACP semantics from the official v1 spec read the same day; stack choices triple-sourced

## Executive Summary

zcode-acp is a protocol-bridge agent adapter: a single static Go binary that speaks ACP v1 (JSON-RPC 2.0, newline-delimited JSON, stdout-exclusive) to any client — Orca chat cards, Zed — on the front leg, and ZCode Desktop's proprietary app-server dialect (plain JSONL that **rejects the `jsonrpc` key** — experimentally verified) on the back leg. The ecosystem's dominant shape for this exact problem is a thin ACP skin over a proprietary backend (claude-agent-acp, the ACP org's own codex-acp), so the pattern is proven. Two structural facts dominate the design. First, the wrapper is a **host, not a proxy**: the app-server runs host-managed and *blocks mid-turn* until every `interaction/*` reverse request is answered (`requestCredentialCipher`, `requestProviderRuntimeHeaders`, `requestPermission`, …), and answering `headersApplied:false` for an account provider is a guaranteed turn failure. Second, the app-server's `session/send` is **fire-and-forget** — the adapter's real work is lifting an async event stream into ACP's synchronous `session/prompt` → `stopReason` contract, with streaming updates flushed before the response.

Recommended approach: use **`coder/acp-go-sdk` v0.20.0** (the de facto Go ACP SDK — stdlib-only, zero transitive deps, schema 1.25.0, wire `protocolVersion: 1`, matches Orca's client) behind a narrow `internal/acpport` port so v0.x API breaks stay a one-package migration; **hand-roll the ~200-line JSONL codec** for the ZCode leg (no library can do it — any JSON-RPC wrapper corrupts framing on the first request); spawn **one shared app-server child per wrapper process** and multiplex ACP sessions onto it with ACP sessionId = app-server sessionId 1:1 (matches the desktop host model and the codex-acp precedent; per-session children would re-push registry snapshots and re-mint PATs N times); scaffold tooling from Djarvur/goswitch (mise single-source versions, exact-pinned strict golangci-lint, 4-workflow GitHub Actions shape, protected branch). Testing follows the PROJECT.md done-v1 definition in three tiers: a scriptable **in-process fake app-server** with golden JSONL transcripts as the CI protocol gate (~90% of tests, no processes, race-detector-friendly), VM tests with the real desktop for what the fake cannot prove (Electron spawn, keyring, revision resolution), and a manual Orca E2E checklist.

Key risks are unusually well characterized — nearly every pitfall was experimentally verified by the predecessor shim and has a specific, testable prevention: silent registry-snapshot discard (compute the revision string `zcode-builtin:<rev>:<sha256(resolved builtin path)>` at runtime and **verify by reading registry state back — never trust the `received` ack**), `session/create` dropping `options.reasoningLevel` for thinking models (always follow with `session/setModel`), premature prompt completion (per-session turn state machine answering `session/prompt` exactly once), protocol drift between desktop releases (fail-fast startup handshake, version-pinned range, unknown-method tolerance, versioned fixtures), orphaned Electron processes (kill the process group, reap with `Wait()`), and stdio deadlocks (fixed goroutine topology, single writer per sink, no 64 KiB scanner default, drained child stderr). The genuinely open items are confined and small: the shipped 3.18.2 event-stream surface (subscribe vs poll), Orca's persona decision and load-bearing startup call set, ZCode install discovery, and `workspaceKey` semantics — all of them Phase 1 research deliverables.

## Key Findings

### Recommended Stack

The stack is deliberately small: one runtime dependency for the ACP leg, one for keyring access, everything else stdlib. The ZCode leg *cannot* use a library (protocol is JSONL but not JSON-RPC); the ACP leg *should not* be hand-rolled (large, evolving typed surface).

**Core technologies:**
- **Go 1.27.x** (stable 1.27.4) — single static binary, first-class `os/exec` hygiene; pin in mise.toml and go.mod
- **github.com/coder/acp-go-sdk v0.20.0** — ACP v1 front leg: `acp.Agent` + `AgentSideConnection`, types generated from official schema 1.25.0, stdio framing included, `-32601` auto-replies for unsupported optional methods; hide every import behind `internal/acpport` (v0.20.0 is a deliberate "all planned breaks at once" stability point)
- **Hand-rolled JSONL codec (~200 LOC, bufio + encoding/json)** — ZCode app-server back leg; own method constants and event types mirroring `packages/zcode-protocol`; single writer goroutine owns child stdin; `interaction/*` are first-class inbound requests
- **github.com/godbus/dbus/v5 v5.2.2** — Secret Service (gnome-keyring) lookup of the credential-encryption key (pure Go, preserves static binary; same library goswitch uses)
- **stdlib crypto/aes + cipher.NewGCM** — AES-256-GCM decryption of `enc:v2:os:<kid>:<iv>.<tag>.<ct>` blobs (segment order iv.tag.ct — field-verified)
- **stdlib net/http** — z.ai OAuth/PAT minting (getCustomerInfo → api_keys → access_tokens); exact header shapes documented; configure transport explicitly (proxy-env traps)
- **goswitch scaffold**: mise.toml (go 1.27, golangci-lint **2.15.0 pinned exactly**, goreleaser 2.20.2), strict `.golangci.yml` (copy goswitch, change only `goimports.local-prefixes`), pr-sanity/release/e2e-matrix/security-scheduled workflows, govulncheck in its own job
- **Testing**: in-process fake app-server over `io.Pipe`/`net.Pipe` (primary); `TestHelperProcess` reexec idiom for exec plumbing only; golden JSONL transcripts (goldie/v2 or a ~30-line hand-rolled `-update` helper); `acp-tck` pinned tag as a non-blocking CI conformance signal

**What NOT to use (hard constraints, experimentally grounded):** any JSON-RPC library on the back leg (`jsonrpc` key is rejected); `net/rpc` anywhere (JSON-RPC 1.0 semantics); LSP Content-Length framing (neither leg uses it); `bufio.Scanner` defaults on the child leg (64 KiB line limit truncates tool payloads); anything but ACP frames on stdout (logs → stderr via slog); PTY libraries (ACP is a pipe protocol; Orca sets `PI_NO_PTY=1`); cobra/DI/config frameworks; waiting for an official Go ACP SDK (none exists; TS/Rust/Python/Kotlin/Java only).

### Expected Features

**Must have (table stakes — clients break without these):**
- `initialize` with `protocolVersion: 1`, honest minimal capabilities, real `agentInfo`; `authMethods[]` (`zai-api-key`, `zai-individual-coding-plan`) + `authenticate`
- ZCode desktop discovery + app-server spawn (`ELECTRON_RUN_AS_NODE=1 <app>/resources/glm/zcode.cjs app-server --stdio`), loud actionable failure when not found
- JSONL ↔ JSON-RPC 2.0 bidirectional transport translation (the whole adapter is this dialect bridge)
- `session/new` → `session/create` **+ follow-up `session/setModel`** (create drops `options.reasoningLevel` for thinking models)
- `session/prompt` with `stopReason` answered only after all updates flush; `agent_message_chunk` + `agent_thought_chunk` streaming from `state.updated` deltas
- `tool_call`/`tool_call_update` lifecycle with diffs (drives Zed edit-review UI; PROJECT.md: a card without tools is meaningless)
- `session/request_permission` bridging (app-server `interaction/requestPermission` — host MUST answer)
- `session/cancel` → stop path + `stopReason:"cancelled"`; pending permissions resolved as cancelled
- Host-managed `interaction/*` responder always-on (correctness precondition, not a feature)
- Child lifecycle hygiene (stdin EOF → kill process group); protocol test suite with mock app-server in CI

**Should have (differentiators):**
- **OAuth individual coding plan end-to-end** — the project's moat; no public ACP adapter drives z.ai's coding-plan OAuth chain (registry snapshot with exact revision string → keyring cipher → PAT mint → signed turns by the official binary)
- Config-options model + reasoning-level picker (`category:"model"` / `"thought_level"` via `session/setModel` + `session/setThoughtLevel`; model change re-emits full configOptions)
- Full thought-stream fidelity (only when backend emits; never synthesize)
- `session/resume` + `session/close` (cheap, native mappings exist); `session/list` + `session/load` replay (add when fidelity proven)
- Orca launch recipe + compatibility no-ops (`session/delete` is a no-op — the wire method does not exist server-side; `session/fork` graceful error)
- Protocol-conformance test suite with mock app-server (also a PROJECT.md done criterion)

**Defer (v2+):** slash commands (no native ZCode equivalent), ACP v2 draft surfaces, non-Ubuntu keyring paths (PR-driven), zcode CLI backend (explicitly out of scope), `logout`/`sessionInfo_update` if demanded.

**Anti-features (deliberately never build):** `fs/*` and `terminal/*` support (ZCode touches disk/shell itself; clients advertise fs=false/terminal=false); re-implementing the agent loop; patching `zcode.cjs`; attestation/quota forgery (ToS, explicitly out of scope); fake quota reporting (the counter is not retrievable — report nothing rather than wrong numbers); 1:1 proxying of proprietary methods onto ACP.

### Architecture Approach

Dual-protocol bridge with a **normalized internal event bus**: each leg has its own codec; leg types never leak across; a protocol-free `TurnEvent` middle (text/thought delta, tool lifecycle, permission, turn done(reason)) is what both sides convert in and out of. One shared app-server child per wrapper process; sessions multiplexed 1:1. The interaction responder is a **policy object** — every server→host request gets a handler, chosen by auth mode, with per-method deadlines (180 s for runtime-headers) and both cancel directions wired. The supervisor owns discovery → spawn → health watch → crash-window restart → staged shutdown. Turn watching uses a `TurnWatcher` interface: **implement polling first** (`session/messages` + idle quiet-window — the only path proven end-to-end on 3.18.2), layer subscribe-based strategies later behind the same interface.

**Major components:**
1. `internal/acpfront` + `internal/acpport` — ACP v1 server via coder SDK; SDK types isolated behind a narrow port
2. `internal/core` — session registry (ACP sid ↔ app-server sid), turn translator, stopReason synthesis, per-turn cancel contexts
3. `internal/zcode` (+ `internal/zcode/fake`) — JSONL codec: envelope types, method constants, pending-call table, reverse-request router; the scriptable fake app-server lives beside it sharing types
4. `internal/supervisor` — discovery per Phase-1 matrix, spawn env assembly, process-group kill, crash window/backoff, staged shutdown
5. `internal/hostsvc` — CredentialVault (keyring + GCM), registry snapshot builder (runtime revision string), PAT client (cache + singleflight)
6. `internal/interact` — interaction/* responder policy; `internal/logging` — slog → stderr with secret redaction

Load-bearing wire details (all source/experiment-confirmed): the app-server serializes requests but has a **bypass lane for stop/cancel** (cancellation must jump the write queue); on client EOF it drains ~100 ms then exits; unknown server→host methods must be answered with a protocol error, never crash; `provider/updateAccountConfig` acks `received` even when the snapshot is silently discarded — verify via registry state.

### Critical Pitfalls

1. **Ack ≠ acceptance, and the back leg is not JSON-RPC** — snapshots with wrong revision strings are silently discarded while the method still answers `received`; separate serializers per hop, compute the revision at runtime from the resolved builtin path, build an `awaitRegistryContains()` helper and make every auth test use it
2. **`session/create` drops `options.reasoningLevel`** — two-step binding is the *only* model-binding path: create bare, then `session/setModel` with the level; fake-server test asserts the payload
3. **`headersApplied:false` / authenticating with the OAuth JWT instead of a minted PAT** — V4 handshake fails with raw tokens (4001/4011); answer `{headersApplied:true, requestAuth:{apiKey:<PAT>, apiKeyId}}` or an explicit error, never `false`; re-mint when the request carries `rejectedProjectTokenFingerprint`
4. **Premature prompt completion** — `session/send` returns before the turn ends (`void runPromptTurnInBackground`); a per-session turn state machine owns the single `session/prompt` response; a second concurrent `session/send` is rejected -32010
5. **Protocol drift between desktop releases** — `interaction/requestCredentialCipher` exists only in the 3.18.2 binary, not the 3.17.x source; fail fast on a startup revision handshake, tolerate unknown methods, pin a supported version range, version the fixtures
6. **Zombie/orphan app-server processes** — `Process.Kill()` kills only the direct child; `Setpgid` + kill the negative process group, always `Wait()`, don't trust stdin-EOF to stop Electron; `Pdeathsig` is unreliable in Go (parent-*thread* semantics)
7. **Pipe deadlocks** — fixed goroutine topology (one reader per pipe, one writer per sink), never handle `interaction/*` inline in the read loop, drain child stderr or the 64 KB pipe buffer stalls the child mid-turn, no goroutine-per-update (reorders chunks, corrupts NDJSON)
8. **Cancel races** — `session/cancel` is a notification but `session/prompt` MUST still be answered exactly once with `stopReason:"cancelled"`; an unanswered `interaction/*` holds a 180 s credentials queue that blocks *subsequent* turns; dedupe the 1 s re-announced requests by `requestId`

Moderate but roadmap-shaping: Go caches `ProxyFromEnvironment` once per process (validate proxy at first use, scrub proxy vars from the child env); keyring-less environments must produce an actionable error, never a hang; the Orca masquerade (persona binary name + parseable single-line `--version` + liberal flag handling + fast `initialize` — PAT mint must NOT happen before initialize returns); capability honesty (advertise nothing fs/terminal-related); never let CI call api.z.ai or spawn the real desktop (quota burn, PAT minting *creates* API keys on the user's org) — three test tiers only.

## Implications for Roadmap

Suggested structure: **7 phases**, reconciling the researchers' converged 5-step spine with the PROJECT.md constraints (research/doc phase FIRST, goswitch scaffold early, code after research, done = CI protocol tests + VM tests + manual Orca E2E). Mapping to the converged spine: Phase 2 (scaffold) is positioned early per constraint; Phases 3–4 split spine-step 2 (codec/supervisor vs ACP loop) so each stays focused; Phases 5, 6, 7 = spine-steps 3, 4, 5.

### Phase 1: Research & Protocol Documentation
**Rationale:** User-mandated first, and independently correct: every downstream phase consumes its outputs, and this is the cheapest place to retire the highest-risk unknowns (wire truth of the shipped 3.18.2 binary, Orca persona).
**Delivers:** `docs/` protocol documentation of the ZCode app-server JSONL dialect with a "verified against: 3.18.2 binary / 3.17.x source" drift baseline (every claim labeled V-EXP vs V-SRC); ACP↔ZCode capability/mapping doc; Orca persona decision (`omp` vs `opencode`/`opencode2`) + the load-bearing card-start call set read from `app.asar.unpacked/out/main/chunks/` (never the packed asar); turn-completion terminal-event enumeration; ZCode install discovery matrix (AppImage/deb/unpacked); `workspaceKey` semantics; captured real JSONL transcripts for later fixture replay; unknown-method tolerance policy.
**Addresses:** all four PROJECT.md open questions; Pitfall 5 (drift baseline); FEATURES capability-map LOW-confidence rows.
**Avoids:** documenting 3.17.x-source behavior as truth for the 3.18.2 binary; the packed-asar research trap.

### Phase 2: Scaffold & CI Bootstrap (goswitch-style)
**Rationale:** PROJECT.md mandates the goswitch scaffold early, before feature code; it is purely mechanical with zero domain research needs, and every later phase then lands into strict gates from day one.
**Delivers:** `go.mod` (go 1.27), `cmd/zcode-acp` skeleton, mise.toml with exact pins, strict `.golangci.yml` copied from goswitch, pr-sanity/release/e2e-matrix/security-scheduled workflows, branch protection, govulncheck job, golden-file test helper (`-update` flag), slog-to-stderr logging package with secret redaction, `internal/acpport` stub.
**Uses:** STACK.md goswitch scaffold (HIGH confidence, local).
**Avoids:** lint-config churn (exact linter pin), stdout-logging habit from day one.

### Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor
**Rationale:** Everything imports `internal/zcode`; building the fake app-server alongside the codec makes every later phase CI-testable without the real bundle — the fake *is* the protocol spec made executable and the PROJECT.md "CI protocol tests" gate starts here.
**Delivers:** JSONL codec (envelope types, method constants, pending-call table keyed by string-or-int id, reverse-request router, single writer goroutine, raised line buffer, stop bypass-lane send), scriptable fake app-server (scripted replies, configured events, `interaction/*` blocking with timeouts), golden transcript tests, supervisor (discovery per Phase-1 matrix, `ELECTRON_RUN_AS_NODE=1` spawn env with scrubbed proxy, process-group kill + reaping, crash window/backoff, staged shutdown), real 3.18.2 child smoke test capturing transcripts to replay into the fake.
**Implements:** ARCHITECTURE Patterns 2, 4, 6.
**Avoids:** Pitfalls 1 (codec half), 5 (fail-fast startup handshake), 6 (zombies), 7 (deadlocks/scanner/stderr).

### Phase 4: ACP Front — Minimal Chat Loop
**Rationale:** Interop risk is retired by getting a real streamed turn in a real ACP client before auth and interaction complexity accrete; api-key-only, no interactions yet.
**Delivers:** `acpfront` Agent implementation via SDK behind `acpport`; `initialize` (protocolVersion 1, honest minimal capabilities, agentInfo); `session/new` + `session/setModel` follow-up; `session/prompt` → `session/send` with the polling `TurnWatcher` (idle quiet-window fallback); `agent_message_chunk` + `agent_thought_chunk` streaming in receive order; `session/cancel` with correct `stopReason` lifecycle and updates-flushed-before-response ordering.
**Implements:** ARCHITECTURE Patterns 1, 5.
**Avoids:** Pitfalls 2 (reasoningLevel), 4 (premature completion — turn state machine), 12 (capability honesty).

### Phase 5: Interactions + Full Chat Cycle (api-key E2E)
**Rationale:** Tool streaming and permission bridging must land together (spec pairing — tools without permissions auto-run or hang); the host-managed responder is a precondition for *both* auth paths, so it lands before OAuth; completes the PROJECT.md "full chat cycle" on the simpler auth path first.
**Delivers:** interaction responder (runtime-preferences defaults; `requestPermission` forwarded to ACP `session/request_permission` with option-kind mapping; both cancel directions wired; per-method deadlines), `tool_call`/`tool_call_update` coalescing with diffs from app-server tool lifecycle events, api-key auth path (`provider/updateAccountConfig` + registry read-back verification), cancel-matrix tests (before-prompt / mid-stream / during-permission / double), scripted fake-server sequences.
**Addresses:** FEATURES P1 tool/permission rows, api-key auth; PROJECT.md full chat cycle.
**Avoids:** Pitfalls 1 (registry verification), 8 (cancel races); "looks done but isn't" chat-loop items.

### Phase 6: OAuth Coding-Plan Path
**Rationale:** The moat feature, deliberately sequenced after the api-key cycle proves the plumbing (matches ARCHITECTURE build order); it needs the Phase-5 interaction responders and the Phase-3 supervisor/env groundwork.
**Delivers:** CredentialVault (godbus Secret Service lookup with exact `sha256(realpath)` semantics, AES-256-GCM iv.tag.ct decryption, in-memory only), registry snapshot builder (runtime revision string, exact `.strict()` payload) + push + verify-applied, PAT client (exact header styles per call; cache + singleflight; `rejectedProjectTokenFingerprint` re-mint; short timeout answering the interaction even on failure), `requestCredentialCipher` handler, `{headersApplied:true, requestAuth}` answers, OAuth bootstrap failures surfaced at startup not mid-turn, proxy-env validation, api-key regression maintained (FINDINGS `APIKEY_PATH_OK` discipline).
**Addresses:** FEATURES P1 OAuth row; PROJECT.md both-auth-paths requirement.
**Avoids:** Pitfalls 3 (headersApplied/PAT), 9 (proxy env once-caching), 10 (keyring/headless).

### Phase 7: Hardening, Orca Recipe, VM E2E + Documentation
**Rationale:** PROJECT.md done-v1 = CI protocol tests + VM tests + manual Orca E2E — this phase *is* the done criterion; fidelity hardening and docs close the milestone.
**Delivers:** subscribe-based `TurnWatcher` if Phase 1 verified it on 3.18.2; restart-mid-turn behavior; version-gate + unknown-method tolerance tests; versioned, scrubbed fixtures (CI secret-scan for JWT/`enc:v2:`/PAT shapes); config surface (flags > env > discovery per ARCHITECTURE table); Orca mimicry CLI (persona per Phase 1, single-line semver `--version`, liberal flag/env handling, fast `initialize`); README + Orca recipe reproducible on a clean VM (the acceptance test); VM E2E on the self-hosted runner (goswitch e2e-matrix pattern); `acp-tck` non-blocking CI job; manual Orca E2E checklist; no-keyring actionable-error VM test.
**Addresses:** PROJECT.md done-v1 tiers 2–3, English docs requirement; FEATURES P2 Orca recipe + compat no-ops.
**Avoids:** Pitfalls 11 (masquerade), 13 (real-desktop testing discipline); the full "looks done but isn't" checklist as exit criteria.

### Phase Ordering Rationale

- **Research before code** is user-mandated *and* de-risks: Phase 1 outputs feed Phase 3 (discovery matrix, spawn), Phase 4 (event surface, terminal events), Phase 6 (snapshot/revision verification), Phase 7 (persona).
- **Scaffold before code** keeps every phase under strict lint/CI/branch-protection gates; it has no research dependencies so it costs nothing to place early.
- **Back leg before front leg:** the codec is the package everything imports, and the fake app-server converts the PROJECT.md CI-gate requirement into the development loop itself — from Phase 3 onward, every feature is testable without the real desktop.
- **Tools + permissions together** in Phase 5 (FEATURES dependency note: shipping tool streaming without permission bridging produces cards that auto-run tools or hang).
- **OAuth last among feature phases:** it is the longest chain (keyring → snapshot → PAT → headers), it depends on the interaction layer, and the api-key path must remain a passing regression throughout.
- **Orca recipe in the final phase** because it is docs + CLI fidelity gated on the Phase-1 persona decision, and its acceptance test (clean-VM reproduction) is inherently a closing activity.

### Research Flags

Phases likely needing deeper research during planning (`/gsd:plan-phase --research-phase`):
- **Phase 1:** *is* the research phase — planning should enumerate its deliverables (persona, call set, event surface, discovery, workspaceKey) rather than re-research them; primary inputs are FINDINGS.md, `ZCode source tree (github.com/zai-org/ZCode; local clone)`, and Orca `app.asar.unpacked` chunks.
- **Phase 4:** light — confirm the exact `acp-go-sdk` v0.20.0 `Agent` interface surface (pkg.go.dev) at planning time; the event→update mapping consumes Phase 1 docs.
- **Phase 5:** light-medium — exact tool-lifecycle event shapes on the 3.18.2 wire (progress kinds are documented from 3.17.x source; wire truth comes from Phase 1 captures).
- **Phase 6:** medium — PAT mint shapes are documented and were experimentally verified once, but plan for live re-verification plus keyring D-Bus specifics on the target VM.

Phases with standard patterns (skip research-phase):
- **Phase 2:** goswitch scaffold is a local copy-paste template (HIGH confidence).
- **Phase 3:** codec/supervisor/fake patterns are fully specified in ARCHITECTURE.md (Patterns 2, 4, 6 with code) with Go process/pipe hygiene sources cited; only the discovery matrix arrives from Phase 1.
- **Phase 7:** VM E2E and release mechanics follow the goswitch e2e-matrix/release pattern; only recipe content depends on Phase 1.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | SDK facts triple-sourced (repo README raw + pkg.go.dev + ACP community page); goswitch scaffold read locally; only secondary ecosystem facts (competitor SDKs) are MEDIUM |
| Features | HIGH | ACP v1 spec pages read the same day; ZCode surface from source tree + experimentally verified FINDINGS; client-behavior claims beyond official docs are labeled MEDIUM |
| Architecture | HIGH | Wire facts field-verified on 3.18.2 + 3.17.x source with file:line citations; ACP semantics from official spec; only the codex-acp precedent is README-derived (MEDIUM) |
| Pitfalls | HIGH | Nearly all claims experimentally verified or source-cited (file:line); a handful of items explicitly labeled M/LOW and assigned to phases |

**Overall confidence:** HIGH — unusually strong for greenfield work, because a predecessor shim already exercised the hardest paths end-to-end on the real binary, and the backend's source tree is readable.

### Gaps to Address

- **3.18.2 event-stream surface** — does the shipped binary accept `session/subscribe` / `v4/conversation/subscribe` and emit 3.17.x-style envelopes? Decides whether the subscribe-based TurnWatcher ever replaces polling. → Phase 1 deliverable.
- **Turn-completion terminal events** — exactly which `state.updated` payloads end a turn (FINDINGS §3.6 ground truth). → Phase 1; unblocks the Phase 4 state machine.
- **Orca persona + load-bearing startup call set** — which hardcoded agent to masquerade as, and which ACP methods card start actually invokes. → Phase 1, from `agent-session-attachment-store-registry-*.js` in `app.asar.unpacked`.
- **ZCode install discovery matrix** — AppImage/deb/unpacked layouts and auto-update side effects on the builtin revision. → Phase 1; blocks the Phase 3 supervisor.
- **`workspaceKey` derivation semantics** — wire-required, shim passed a value but rules unconfirmed (LOW). → Phase 1.
- **Child stdin-EOF exit behavior** — whether Electron-as-node app-server exits when stdin closes is unverified; the supervisor must explicitly kill the process group regardless. → verify in Phase 3.
- **Child idle/exit behavior under long idle** — no protocol liveness frame exists; affects supervisor health strategy. → Phase 3/7.
- **Go `NO_PROXY` dialect vs curl edge cases** (MEDIUM) → validate in Phase 6/7 VM tests with a proxy env set.
- **Orca startup probe timeout** (LOW) — unverified; the "fast initialize, defer PAT mint" rule is mandated regardless. → Phase 7.
- **Slash commands and plan/todo ACP updates** — no verified ZCode mapping; legitimately deferred to v2+ (omission in initialize is spec-correct, not a gap).

## Sources

### Primary (HIGH confidence)
- `FINDINGS.md (private research notes, outside this repo)` — experimentally verified on real ZCode 3.18.2 (JSONL dialect, OAuth/PAT chain, Orca client facts, protocol bugs, revision table)
- `ZCode source tree (github.com/zai-org/ZCode; local clone)` (public 3.17.x tree) — wire-method tables, transport (serial queue, stop bypass, EOF drain), registry service (silent snapshot retention), interaction broker (1 s re-announce), provider-runtime-headers (180 s timeout, cancellation)
- `local field-notes references/` — app-server-stdio-protocol.md, oauth-coding-plan-shim.md (field-verified interaction reply shapes, snapshot/PAT details)
- agentclientprotocol.com v1 spec — overview, transports, initialization, session-setup, prompt-turn, authentication, session-config-options, tool-calls, cancellation, file-system
- github.com/coder/acp-go-sdk v0.20.0 + pkg.go.dev — schema pin 1.25.0, protocolVersion 1, both connection sides, zero deps, 18 importers
- Djarvur/goswitch v1.2.3 (local) — scaffold model mandated by PROJECT.md: mise, .golangci.yml, workflows, stdlib-only test style
- github.com/agentclientprotocol org — all 15 repos enumerated: official SDKs TS/Rust/Python/Kotlin/Java, **no Go**; acp-tck harness

### Secondary (MEDIUM confidence)
- agentclientprotocol/codex-acp — one-shared-app-server precedent, crash-window restart policy (README-derived)
- github.com/agentclientprotocol/acp-tck — tiers, verdicts, experimental upstream status
- Zed external-agents docs, google-gemini/gemini-cli ACP docs (+ issue #13913 regression), zed-industries/claude-code-acp README — competitor feature analysis
- ironpark/acp-go, BrokkAi/acp-go — alternative Go SDK survey
- Go-specific: net/http `envProxyOnce`, golang/go#27505 (Pdeathsig), Setpgid/group-kill references, bufio.ErrTooLong

### Tertiary (LOW confidence — assigned to phases)
- workspaceKey semantics, Orca startup-probe timeout, child stdin-EOF exit behavior, exact 1 s re-announce wire behavior, NO_PROXY dialect edge cases — each labeled in the detailed research files and mapped to a Phase 1/3/6/7 deliverable above

---
*Research completed: 2026-11-19*
*Ready for roadmap: yes*
