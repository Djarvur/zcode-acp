# Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor - Context

**Gathered:** 2026-11-19
**Status:** Ready for planning
**Mode:** `--auto` — all decisions auto-selected (recommended defaults), logged for audit; see `03-DISCUSSION-LOG.md`

<domain>
## Phase Boundary

Phase 3 delivers the **wrapper's back leg and its CI-proof**: the JSONL codec for the
app-server dialect (no `jsonrpc` key, dialect error codes, ordered single-writer output,
stop bypass lane), the scriptable fake app-server (in-process + standalone binary) with
golden transcript tests, desktop discovery + process supervisor (spawn env, process-group
kill, crash-window restart, actionable diagnostics), and the turn state machine lifting
fire-and-forget `session/send` into a synchronous turn with multi-signal completion and
cancel. Everything proven in CI without the real desktop bundle (BRIDGE-01..04).

Out of scope: the ACP front leg (Phase 4), auth/interactions beyond answering the
session-establishment host obligations (Phase 5), OAuth (Phase 6), VM/manual E2E (Phase 7).

</domain>

<decisions>
## Implementation Decisions

### Fake App-Server Architecture
- **D-01:** **In-process fake over `io.Pipe`/`net.Pipe` is the primary CI instrument**;
  a **standalone binary mode** (`fake-app-server`, same scripting core) exists for
  exec-plumbing tests (the supervisor's spawn path). Scripting = **JSON scenario files**
  (declarative: requests to answer, events to emit with timing, terminal signals) +
  programmatic option for tests needing dynamic behavior. **Golden JSONL transcripts**
  (request/response pairs captured per scenario) live under `testdata/` and are the
  protocol-spec-made-executable; CI replays them. — **Reversibility:** reversible.

### Turn Completion Detection
- **D-02:** **Multi-signal, ordered:** (1) terminal `state.updated` event
  (`turn.completed` / `turn.failed` per docs/protocol/app-server-dialect.md — V-SRC×7/×8
  taxonomy) is the primary completion signal; (2) **idle quiet-window fallback** covers
  its absence (polling `session/messages` per ARCHITECTURE — the only path proven
  end-to-end on 3.18.2); (3) `session/cancel` travels the **stop bypass lane**
  immediately (never queued behind the single writer). The turn answers `session/prompt`
  exactly once; a second concurrent `session/send` is rejected -32010 (dialect doc).
  The TurnWatcher is an interface: polling implementation first, subscribe-based later
  (Phase 1 verified `session/subscribe` ACCEPTED on 3.18.2 — strategy slot reserved).

### Supervisor Policy
- **D-03:** Discovery implements the **Phase 1 matrix** (docs/protocol/install-discovery.md:
  AppImage / deb / unpacked layouts, revision formula
  `zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>`). Spawn env:
  `ELECTRON_RUN_AS_NODE=1 <root> <bundle> app-server --stdio` with **scrubbed proxy vars**
  (PITFALL 9 — Go caches ProxyFromEnvironment once; validate at first use, scrub from
  child env). Lifecycle: **process-group spawn (`Setpgid`) + group-kill + `Wait()`**
  (PITFALL 6 — never trust stdin-EOF on Electron); **crash-window restart: 3 attempts
  within 60 s, then fail loud** (codex-acp precedent); no-install failure lists every
  searched location (actionable diagnostics). Health watch = stdin liveness + child exit;
  no protocol liveness frame exists (Phase 1 gap note).

### Session Multiplexing
- **D-04:** **One shared app-server child per wrapper process; ACP sessionId =
  app-server sessionId 1:1** (desktop host model; codex-acp precedent). Locked by
  ARCHITECTURE.md — re-affirmed, not re-derived.

### Session-Establishment Host Obligations
- **D-05:** The back leg's interaction router **answers `session/requestRuntimePreferences`
  (15 s deadline) with static runtime-preference defaults** during session/create
  (Phase 1 discovery: unanswered creates fail -32603; minimal answer proven live in
  EXP-WORKSPACEKEY-01's harness). Per-method deadline timers land now for this one
  method; the full interaction-responder policy is Phase 5's AUTH/CORE-06 scope —
  this is the minimum viable answering the codec needs for create to succeed.
  `interaction/requestCredentialCipher` mid-turn is NOT answered in Phase 3 (needs
  Phase 6 keyring vault); the fake scripts it; the real-server path records the
  deadline expiry honestly.

### Module Layout
- **D-06:** Per ARCHITECTURE.md §Major components: `internal/zcode` (codec: envelope
  types, method constants, pending-call table keyed by string-or-int id, reverse-request
  router, single writer goroutine, raised line buffer, stop bypass send) +
  `internal/zcode/fake` (the scriptable fake) + `internal/supervisor` (discovery, spawn,
  health, restart, staged shutdown). Public wire types mirror `packages/zcode-protocol`
  names from the dialect doc. `internal/turn` (or inside zcode) holds the turn state
  machine. No front-leg code in this phase.

### Claude's Discretion
- Scenario-file schema details; golden-transcript fixture granularity
- Queue/backpressure implementation inside the single-writer contract
- Test structuring within the gates (unit vs golden vs harness round-trip)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning & Research (in repo)
- `.planning/REQUIREMENTS.md` — BRIDGE-01..04
- `.planning/ROADMAP.md` — Phase 3 success criteria (4 items)
- `docs/protocol/app-server-dialect.md` — **THE codec spec**: transport facts (JSONL, no
  `jsonrpc` key, serial queue, stop bypass, EOF drain ~100 ms), 110 host→server + 43 v4
  method map with counts, error codes (-32010/-32042/-32002...), server→host surface
  incl. `session/requestRuntimePreferences` 15 s, terminal signals, drift baseline
- `docs/protocol/install-discovery.md` — THE discovery matrix + revision formula
- `docs/protocol/workspace-key.md` — workspaceKey semantics (D-05 harness precedent)
- `.planning/research/ARCHITECTURE.md` — Patterns 2/4/6 (codec, supervisor, turn) with
  code shapes; Major components layout; interaction broker facts (1 s re-announce,
  180 s deadlines)
- `.planning/research/PITFALLS.md` — pitfalls 1 (codec half), 5 (fail-fast handshake),
  6 (zombies), 7 (deadlocks/scanner/stderr), 8 (cancel races) with file:line citations
- `experiments/app-server-probe.mjs` + `experiments/captures/EXP-*.jsonl` — the
  validated harness pattern + real wire transcripts (fixture seeds; EXP-WORKSPACEKEY-01
  proves the requestRuntimePreferences answer shape)
- `.planning/phases/02-scaffold-ci-bootstrap/02-REVIEW.md` + DISPOSITION — probe
  WR-02 (server→host reply-collision risk) routes its fix into the real demux here
- `.planning/phases/01-research-protocol-documentation/01-CONTEXT.md` — private-path
  and phrasing discipline

### External Ground Truth (outside the repo)
- ZCode source tree 3.17.x (local clone; path in agent prompts) — corroborating wire
  behavior; the 4 drift-baseline methods are bundle-only (never cite 3.17.x as 3.18.2 truth)
- Unpacked 3.18.2 bundle (local) — re-derivable method strings
- codex-acp README (crash-window restart precedent)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `experiments/app-server-probe.mjs` — the validated spawn/answer/capture harness
  (stdin-hold, process-group backstop, hostSent recording, `--desktop-version` flag);
  its shapes translate directly into the Go supervisor + fake
- `experiments/captures/EXP-*.jsonl` — real wire transcripts (scrubbed) = golden-fixture
  seeds and the requestRuntimePreferences answer evidence
- Phase 2 gates: `mise run ci` runs `go test -race ./...` — the codec lands under it

### Established Patterns
- Go style: strict golangci (`_, _ = fmt.Fprintln` discard, nlreturn blank line — the
  two proven traps); `internal/` layout per ARCHITECTURE
- Provenance discipline: wire claims cite the dialect doc; drift baseline respected
- Logs → stderr via slog (never stdout — stdout is protocol)

### Integration Points
- `internal/zcode` is imported by everything from Phase 4 onward (ACP front)
- `internal/supervisor` consumes the Phase 1 discovery matrix
- CI (`pr-sanity`) must pass with the new packages; golden tests run under `go test -race`

</code_context>

<specifics>
## Specific Ideas

- The fake app-server IS the protocol spec made executable — golden transcripts are the
  CI protocol gate the PROJECT done-v1 definition starts here.
- Stop bypass lane: cancellation must jump the write queue — verified transport fact.
- Pending-call table keyed by string-or-int id (server uses string ids "server-1"-style
  today; numeric-tolerant per REVIEW WR-02 lesson).
- Raised line buffer (bufio.Scanner 64 KiB default truncates tool payloads — PITFALL 7).
- Child stderr must be drained or the 64 KB pipe buffer stalls the child mid-turn.

</specifics>

<deferred>
## Deferred Ideas

- Subscribe-based TurnWatcher implementation — Phase 4+ (strategy slot reserved now)
- Full interaction-responder policy (dedupe re-announcements, per-method policy table) — Phase 5
- session/list + load replay — v2 (SESSH-01)

</deferred>

---

*Phase: 3-ZCode Back Leg — Codec, Fake App-Server, Supervisor*
*Context gathered: 2026-11-19*
