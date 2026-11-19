# Phase 4: ACP Front — Minimal Chat Loop - Context

**Gathered:** 2026-11-20
**Status:** Ready for planning
**Mode:** `--auto` — decisions auto-selected (recommended defaults); see `04-DISCUSSION-LOG.md`

<domain>
## Phase Boundary

Phase 4 delivers the **ACP v1 front leg with the minimal chat loop**: `initialize`
(protocolVersion 1, honest capabilities, agentInfo, both authMethods advertised),
`session/new{cwd}` → backend `session/create` + follow-up `session/setModel` (the
reasoningLevel-loss workaround, visible on the wire), `mcpServers` accepted without
error, `session/prompt` streaming `agent_message_chunk` deltas live with `stopReason`
answered only after all updates flush, `agent_thought_chunk` when (and only when) the
backend emits reasoning, and `session/cancel` → bypass stop + `stopReason: "cancelled"`
exactly once. Proven against the fake app-server in CI (CORE-01..05). No auth, no
permission bridging, no tools streaming yet (Phases 5–6).

Out of scope: auth (Phase 5–6), interaction responder beyond Phase 3's
requestRuntimePreferences (Phase 5), tool_call streaming (Phase 5), session/resume/close
(Phase 5), fs/terminal surfaces (never — out of scope project-wide), model configOptions
picker (Phase 6).

</domain>

<decisions>
## Implementation Decisions

### ACP Leg Implementation Base
- **D-01:** **Decided at planning by the Package Legitimacy Audit** between:
  (a) `coder/acp-go-sdk` v0.20.0 — STACK.md's HIGH-confidence recommendation (schema 1.25.0
  typed surface, stdio framing included; hides behind `internal/acpport` so v0.x breaks
  stay a one-package migration); (b) hand-rolling the ACP v1 JSON-RPC 2.0 framing on the
  proven Phase-3 codec patterns (zero deps, but reimplements the spec surface — ACP v1 is
  a large typed surface). **Default: (a) if the audit passes** (official coder org,
  license compatible, no suspicious deps, pinned version); **(b) is the pre-approved
  fallback** if the audit flags legitimacy concerns — the minimal loop needs only a
  handful of surface points, and Phase 3 proved the framing discipline. Either way the
  seam is `internal/acpport` (ARCHITECTURE) — the front leg never imports the SDK (or
  its hand-rolled twin) outside that package. Decision + audit evidence recorded in
  RESEARCH.md and COVERAGE.md-style notes before plans land.

### Capability Honesty
- **D-02:** `initialize` advertises EXACTLY what exists: `protocolVersion: 1`, both
  `authMethods` (`zai-api-key`, `zai-individual-coding-plan` — advertised now, functional
  in Phases 5–6 per REQUIREMENTS CORE-01), real `agentInfo` (name `zcode-acp`, version
  from the binary's version var), NO fs/terminal capabilities, NO slash commands, NO
  readTextFile/writeTextFile — omission is spec-correct (PROJECT.md anti-features).
  Anything not implemented is not advertised.

### session/new Contract
- **D-03:** `session/new{cwd, mcpServers, ...}` → backend `session/create` (workspaceKey =
  cwd path per docs/protocol/workspace-key.md) → follow-up `session/setModel` carrying
  model + reasoning level (the create-drops-reasoningLevel workaround — V-EXP proven);
  `mcpServers` accepted-and-ignored; the follow-up setModel visible on the wire is
  asserted by a fake transcript (CORE-02).

### Streaming Contract
- **D-04:** `session/prompt` → backend `session/send` fire-and-forget lifted by the
  Phase-3 machine; `state.updated` deltas stream to the client as `session/update`
  (`agent_message_chunk` / `agent_thought_chunk`) LIVE while the turn runs — never
  buffered-then-flushed; `stopReason` returns only after all pending updates flushed
  (CORE-03); `agent_thought_chunk` emitted ONLY on backend reasoning events, never
  synthesized (CORE-04). Update ordering = receive order (single demux path).

### Cancel Contract
- **D-05:** `session/cancel` → Phase-3 bypass `conn.Stop` + machine cancel (exactly-once);
  the prompt call resolves `stopReason: "cancelled"` exactly once; cancel-after-complete
  is a no-op; a second concurrent prompt while a turn runs is rejected (wrapper guard +
  backend -32010 mapping already in place) (CORE-05).

### Claude's Discretion
- acpport seam API shape; test structuring (fake-driven round-trips extend Phase 3 patterns)
- agentInfo field details; error message wording on the ACP surface
- Wave/task decomposition within the locked plan structure

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning & Research (in repo)
- `.planning/REQUIREMENTS.md` — CORE-01..05
- `.planning/ROADMAP.md` — Phase 4 success criteria (5 items)
- `.planning/research/STACK.md` — acp-go-sdk recommendation + acpport seam rationale
- `.planning/research/ARCHITECTURE.md` — Pattern 1 (acpfront), Pattern 5 (turn), Major
  components (acpport seam)
- `docs/protocol/app-server-dialect.md` — the back leg the front maps onto
- `docs/protocol/workspace-key.md` — workspaceKey = cwd path
- agentclientprotocol.com v1 spec (initialization, session-setup, prompt-turn,
  cancellation pages) — the front-leg spec
- `.planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-VERIFICATION.md`
  — the machinery this phase composes
- `.planning/research/REFERENCE-ADAPTERS.md` — client-side expectations (what Orca's
  client sends: capabilities shape, elicitation offer — later phases)

### External Ground Truth (outside the repo)
- github.com/coder/acp-go-sdk v0.20.0 (+ pkg.go.dev) — candidate (a); legitimacy audit target
- github.com/agentclientprotocol org SDKs (TS/Rust/Python/Kotlin/Java) — spec cross-reference
- agentclientprotocol/codex-acp — one-shared-app-server precedent

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/zcode` (Conn, machine, watcher) — the front composes it per ARCHITECTURE
  Pattern 1: acpfront ↔ acpport ↔ core ↔ zcode
- `internal/zcode/fake` + corpus — every Phase 4 test composes the fake; new scenarios
  (setModel follow-up, thought chunks, cancel-mid-turn already present) extend it
- `internal/supervisor` — tests spawn the fake binary directly (no discovery needed)
- Phase 2 gates — `mise run ci` must stay green with the new packages

### Established Patterns
- TDD RED→GREEN with machine evidence (go-test adapter)
- Zero new deps discipline (Legitimacy Audit gates any addition — D-01)
- stdout-is-protocol; logs → stderr slog; neutral phrasing; no machine paths

### Integration Points
- `internal/acpfront` (new) composes `internal/acpport` (new) + `internal/zcode` (existing)
- `cmd/zcode-acp` gains its first real wiring (initialize path) behind the agent loop
- Fake scenarios extend for ACP-leg behaviors (setModel wire assertion, thought-chunk)

</code_context>

<specifics>
## Specific Ideas

- The minimal loop is the FIRST vertical integration: a real ACP client (test client)
  → acpfront → machine → fake → wire-asserted transcript. That E2E is the tracer.
- Honest capability advertisement is a project anti-feature guarantee — test-locked.
- setModel follow-up payload asserted on the fake transcript (the workaround's regression
  lock for Phases 5–6).

</specifics>

<deferred>
## Deferred Ideas

- `session/request_permission` bridging — Phase 5 (TOOL-02)
- elicitation/create preference — Phase 5+ (client capability-gated)
- model/thought configOptions — Phase 6 (MODEL-01/02)
- subscribe-based TurnWatcher — reserved slot (Phase 3 D-02)
- acp-tck external conformance harness wiring — Phase 7 (QA-01; adjudicated at Phase-4 plan
  revision iteration 1, RESEARCH Open Questions 1 RESOLVED; no Phase-4 plan wires it — the
  optional manual smoke stays non-gating per 04-VALIDATION.md)

</deferred>

---

*Phase: 4-ACP Front — Minimal Chat Loop*
*Context gathered: 2026-11-20*
