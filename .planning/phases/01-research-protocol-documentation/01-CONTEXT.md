# Phase 1: Research & Protocol Documentation - Context

**Gathered:** 2026-11-19
**Status:** Ready for planning
**Mode:** `--auto` — all decisions auto-selected (recommended defaults), logged for audit; see `01-DISCUSSION-LOG.md`

<domain>
## Phase Boundary

Phase 1 delivers **verified wire-truth documentation, no production code**: the ZCode
app-server JSONL event surface (DISC-01), the ACP client / preset-agent dialect
(DISC-02, identity resolved: **omp**), the desktop install discovery matrix (DISC-03),
`workspaceKey` derivation semantics (DISC-04), and a reference-implementation
comparison under `docs/research/` (DISC-05). Every claim is labeled V-SRC (source),
V-EXP (experiment), or V-RPT (reported, unverified). Deliverables feed Phases 3–7;
their success criteria quote these documents.

Out of scope: any wrapper code (Phase 2+), live chat-UI card integration (Phase 7),
CI scaffolding (Phase 2), auth implementation (Phases 5–6).

</domain>

<decisions>
## Implementation Decisions

### Ground-Truth Source Hierarchy
- **D-01:** The **installed unpacked 3.18.2 desktop bundle** (`resources/glm/zcode.cjs`
  and its chunk files) is the V-SRC ground truth for shipped-binary behavior. The
  public GitHub tree (3.17.x local clone) is **corroborating** evidence, always
  cited with its version; never present 3.17.x-source behavior as 3.18.2 truth.
  If the GitHub tree carries a tag/commit matching 3.18.2, cite it additionally;
  otherwise document the 3.17↔3.18.2 drift baseline explicitly in
  `docs/protocol/app-server-dialect.md`. Load-bearing claims are additionally
  V-EXP against the running binary.

### Experiment Artifacts
- **D-02:** Commit **small harness scripts under `experiments/`** and **scrubbed
  JSONL captures under `experiments/captures/`** (strip JWTs, `enc:v2:` blobs, PAT
  shapes, account identifiers). Raw unscrubbed captures stay **outside the repo**.
  Scrubbed transcripts are the seed for Phase 3 golden fixtures — this is why they
  are committed rather than summarized. — **Reversibility:** costly — once captures
  are committed, scrubbing mistakes require history rewrite; hence scrub-first
  discipline and a CI secret-scan shape noted for Phase 2 (JWT / `enc:v2:` / PAT
  regexes).

### Docs Layout
- **D-03:** Split per success criterion:
  - `docs/protocol/app-server-dialect.md` — DISC-01: event surface, `session/subscribe`
    availability, turn-completion terminal signals, error codes, transport facts
  - `docs/protocol/acp-client-dialect.md` — DISC-02: preset-agent identity (omp card),
    version floor, env quirks, load-bearing method set at card startup / per turn,
    full ACP method range the client sends (written neutrally — no hidden-goal wording)
  - `docs/protocol/install-discovery.md` — DISC-03: AppImage / deb / unpacked layouts,
    auto-update effects on the builtin revision string
  - `docs/protocol/workspace-key.md` — DISC-04: derivation semantics, source citation
    + recorded experiment
  - `docs/research/reference-adapters.md` — DISC-05: what the TypeScript reference
    covers, adopt/avoid verdicts, attribution obligations (Apache-2.0)
  - `docs/protocol/README.md` — index + the shared provenance legend

### Provenance Format
- **D-04:** Inline tags per claim — `[V-SRC <path>#<anchor>]`, `[V-EXP <experiment-id>]`,
  `[V-RPT <source>]` — plus a **Sources & provenance** section at the end of each doc;
  the legend (what each label means, how ids resolve) lives once in
  `docs/protocol/README.md`; `[V-EXP <id>]` resolves to an `experiments/captures/`
  filename.

### Experiment Policy
- **D-05:** **Targeted experiments only, for the open items**; existing FINDINGS
  V-EXP evidence is reused where already load-bearing (FINDINGS is private; its
  facts must be re-recorded with reproducible experiment ids in-repo):
  1. `session/subscribe` / event-stream surface probe on the shipped binary
  2. Turn-completion terminal signals — one **tiny** real turn **only if local
     account credentials exist**; otherwise source-only + record the limitation
  3. `workspaceKey` derivation — source read + one experiment
  4. Install discovery — enumerate layouts (AppImage unpacked exists locally; check
     deb / other layouts from packaging docs and release artifacts)
  5. `--session-dir` session-file format + omp `modelDiscovery` kind `session`
     behavior — from Orca source (`session-scanner-omp-subagent-transcripts`,
     `transcriptImport` in `structured-agent-session-host-assembly`) and any local
     omp session directory
  Quota discipline: minimal turns, manual tier only, **never from CI** (PROJECT.md
  constraint). If credentials are unavailable on the machine, degrade to source-only
  and mark the affected claims V-SRC-only rather than blocking the phase.

### Orca Scope
- **D-06:** **No live chat-UI integration in Phase 1.** The omp contract table in
  REFERENCE-ADAPTERS.md (V-SRC(parallel)) is **spot-checked by re-reading** the
  unpacked Orca AppImage chunks in this phase's own session — re-verified claims
  upgrade to V-SRC with in-repo citation; the two open items (`--session-dir`
  format, `modelDiscovery` kind `session`) are answered statically here; a live
  card run belongs to Phase 7 manual E2E.

### Claude's Discretion
- Exact experiment script implementations and capture formats (within D-02 rules)
- Doc prose, depth, and section structure (within D-03/D-04 rules)
- Fallback behavior when local credentials or a deb layout are unavailable
  (record limitations, do not block)
- Choice of which V-SRC(parallel) claims to re-verify first (prioritize
  load-bearing: launch line, `--version` shape, `agentCmdOverrides` semantics)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning & Research (in repo)
- `.planning/REQUIREMENTS.md` — DISC-01…05 definitions (the phase deliverables)
- `.planning/PROJECT.md` — constraints, key decisions, open questions; hidden-goal
  wording rule (neutral ACP-server phrasing in all committed docs)
- `.planning/ROADMAP.md` — Phase 1 success criteria (quoted verbatim per deliverable)
- `.planning/research/REFERENCE-ADAPTERS.md` — omp contract table (V-SRC(parallel)),
  reference-impl key files (`credential-store.ts`, `account-provider.ts`,
  `project-token.ts`, `server-requests.ts`, `client.ts`), adopt/avoid leads
- `.planning/research/SUMMARY.md` — §Gaps to Address (the exact open items this
  phase retires) and per-phase deliverable mapping
- `.planning/research/ARCHITECTURE.md` — wire details (serial queue, stop bypass,
  EOF drain, interaction broker) that D-01 evidence must re-verify
- `.planning/research/PITFALLS.md` — pitfalls with file:line citations to re-verify
  against the 3.18.2 bundle
- `.planning/research/FEATURES.md` — capability map rows with LOW confidence to retire
- `.planning/research/STACK.md` — stack decisions informing the "implications" notes

### External Ground Truth (outside the repo — real locations are supplied in agent
prompts, never written into committed files)
- `FINDINGS.md` (private research notes, outside this repo) — prior V-EXP evidence:
  JSONL dialect facts, OAuth/PAT chain, revision table, protocol bugs
- ZCode source tree (github.com/zai-org/ZCode; local clone, 3.17.x) — corroborating
  source with file:line citations
- Unpacked ZCode 3.18.2 desktop bundle (local AppImage squashfs-root) — V-SRC ground
  truth per D-01 + experiment target
- local field-notes `references/` — `app-server-stdio-protocol.md`,
  `oauth-coding-plan-shim.md`, `client-request-signing-v4.md` (field-verified reply
  shapes, snapshot/PAT details)
- github.com/stablyai/orca — client source; read `app.asar.unpacked/out/main/chunks/`
  (`agent-session-attachment-store-registry-*`,
  `structured-agent-session-host-assembly`), never the packed asar
- github.com/william0wang/zcode-acp — reference implementation (Apache-2.0; port
  allowed with attribution)
- agentclientprotocol.com — ACP v1 spec (initialization, session-setup, prompt-turn,
  authentication, session-config-options, tool-calls, cancellation)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — the repository is greenfield (only `.planning/` exists). The planning
  research documents ARE this phase's "codebase": `REFERENCE-ADAPTERS.md` carries a
  compiled omp contract table to spot-check, not re-derive; FINDINGS-derived facts
  in `PROJECT.md` §Context are leads to re-verify with in-repo provenance.

### Established Patterns
- Provenance labeling convention already used in `.planning/research/*`
  (V-SRC / V-EXP / V-RPT) — D-04 formalizes it for `docs/`.
- Private-vs-committed discipline: real machine paths and the hidden goal never
  enter committed files (PROJECT.md neutral phrasing is intentional).

### Integration Points
- `docs/` tree created here is consumed by Phase 3 (discovery matrix, spawn env,
  dialect), Phase 4 (event surface, terminal events), Phase 5 (interaction shapes),
  Phase 6 (snapshot/revision verification), Phase 7 (persona recipe, conformance
  scenario).
- `experiments/captures/` transcripts are the seed for Phase 3 golden fixtures.

</code_context>

<specifics>
## Specific Ideas

- The omp contract table must be **spot-checked, not re-derived** (floor 17.0.5;
  `--version` prints ≥ 18.1.9; `agentCmdOverrides` wiring; Hb string contracts —
  `Unknown ACP model: …`, `No model selected.`, sessionLost strings).
- `session/create` dropping `options.reasoningLevel` (→ `session/setModel`
  workaround) must appear in the dialect doc as a load-bearing quirk with V-EXP id.
- `provider/updateAccountConfig` ack (`received`) ≠ acceptance — silent snapshot
  discard must be documented as a warning-level fact (registry read-back is the
  verification; Phase 6 consumes this).
- The 3.17↔3.18.2 drift baseline is a first-class deliverable section, not an
  afterthought (Pitfall 5).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (v2 items EXT-01 slash commands,
SESSH-01 session history, EXT-04 additional persona cards remain tracked in
REQUIREMENTS.md v2 section.)

</deferred>

---

*Phase: 1-Research & Protocol Documentation*
*Context gathered: 2026-11-19*
