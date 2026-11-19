# Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor - Research

**Researched:** 2026-11-19
**Domain:** Go JSONL codec for a non-JSON-RPC dialect, scriptable protocol fake, process supervisor, async-to-sync turn state machine
**Confidence:** HIGH — the wire spec is Phase-1-verified (`docs/protocol/app-server-dialect.md`), every discrete value below is quoted from files Read this session, and every Go API claim is verified against the pinned go1.27.3 toolchain's own docs/source.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01 — Fake App-Server Architecture:** In-process fake over `io.Pipe`/`net.Pipe` is the primary CI instrument; a standalone binary mode (`fake-app-server`, same scripting core) exists for exec-plumbing tests (the supervisor's spawn path). Scripting = JSON scenario files (declarative: requests to answer, events to emit with timing, terminal signals) + programmatic option for tests needing dynamic behavior. Golden JSONL transcripts (request/response pairs captured per scenario) live under `testdata/` and are the protocol-spec-made-executable; CI replays them. Reversibility: reversible.

**D-02 — Turn Completion Detection:** Multi-signal, ordered: (1) terminal `state.updated` event (`turn.completed` / `turn.failed` per docs/protocol/app-server-dialect.md — V-SRC×7/×8 taxonomy) is the primary completion signal; (2) idle quiet-window fallback covers its absence (polling `session/messages` per ARCHITECTURE — the only path proven end-to-end on 3.18.2); (3) `session/cancel` travels the stop bypass lane immediately (never queued behind the single writer). The turn answers `session/prompt` exactly once; a second concurrent `session/send` is rejected -32010 (dialect doc). The TurnWatcher is an interface: polling implementation first, subscribe-based later (Phase 1 verified `session/subscribe` ACCEPTED on 3.18.2 — strategy slot reserved).

**D-03 — Supervisor Policy:** Discovery implements the Phase 1 matrix (docs/protocol/install-discovery.md: AppImage / deb / unpacked layouts, revision formula `zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>`). Spawn env: `ELECTRON_RUN_AS_NODE=1 <root> <bundle> app-server --stdio` with scrubbed proxy vars (PITFALL 9 — Go caches ProxyFromEnvironment once; validate at first use, scrub from child env). Lifecycle: process-group spawn (`Setpgid`) + group-kill + `Wait()` (PITFALL 6 — never trust stdin-EOF on Electron); crash-window restart: 3 attempts within 60 s, then fail loud (codex-acp precedent); no-install failure lists every searched location (actionable diagnostics). Health watch = stdin liveness + child exit; no protocol liveness frame exists (Phase 1 gap note).

**D-04 — Session Multiplexing:** One shared app-server child per wrapper process; ACP sessionId = app-server sessionId 1:1 (desktop host model; codex-acp precedent). Locked by ARCHITECTURE.md — re-affirmed, not re-derived.

**D-05 — Session-Establishment Host Obligations:** The back leg's interaction router answers `session/requestRuntimePreferences` (15 s deadline) with static runtime-preference defaults during session/create (Phase 1 discovery: unanswered creates fail -32603; minimal answer proven live in EXP-WORKSPACEKEY-01's harness). Per-method deadline timers land now for this one method; the full interaction-responder policy is Phase 5's AUTH/CORE-06 scope. `interaction/requestCredentialCipher` mid-turn is NOT answered in Phase 3 (needs Phase 6 keyring vault); the fake scripts it; the real-server path records the deadline expiry honestly.

**D-06 — Module Layout:** Per ARCHITECTURE.md §Major components: `internal/zcode` (codec: envelope types, method constants, pending-call table keyed by string-or-int id, reverse-request router, single writer goroutine, raised line buffer, stop bypass send) + `internal/zcode/fake` (the scriptable fake) + `internal/supervisor` (discovery, spawn, health, restart, staged shutdown). Public wire types mirror `packages/zcode-protocol` names from the dialect doc. `internal/turn` (or inside zcode) holds the turn state machine. No front-leg code in this phase.

### Claude's Discretion
- Scenario-file schema details; golden-transcript fixture granularity
- Queue/backpressure implementation inside the single-writer contract
- Test structuring within the gates (unit vs golden vs harness round-trip)

### Deferred Ideas (OUT OF SCOPE)
- Subscribe-based TurnWatcher implementation — Phase 4+ (strategy slot reserved now)
- Full interaction-responder policy (dedupe re-announcements, per-method policy table) — Phase 5
- session/list + load replay — v2 (SESSH-01)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BRIDGE-01 | JSONL codec for the app-server dialect (no `jsonrpc` key, its error codes, ordered single-writer output) with unit tests | Envelope grammar + envelope struct (Pattern 1); method/error-code constants quoted verbatim from dialect doc; single-writer + bypass-lane design (Pattern 2); demux incl. WR-02 route (Pitfall section); unit test list (Validation Architecture) |
| BRIDGE-02 | Fake app-server (in-process over pipes + standalone binary) and golden JSONL transcript tests running in CI | Scenario schema + two hosting modes + golden format (Pattern 4); corpus of 10 scenarios seeded from real captures (Golden Transcript Corpus); helper-process reexec verification (goroot-verified idiom) |
| BRIDGE-03 | Desktop discovery + process supervisor: locate install, spawn `ELECTRON_RUN_AS_NODE=1 …/zcode.cjs app-server --stdio`, kill whole process group on EOF/exit, fail loudly with actionable diagnostics when not found | Discovery interface mirroring the install-discovery order (Pattern 5); verbatim spawn line + env; group-kill via `Setpgid`+`syscall.Kill(-pid)` (go-doc verified); no-install diagnostics = every probe attempt; this machine HAS the install (Environment Availability) so the positive path is manually checkable, CI uses stubs + fake binary |
| BRIDGE-04 | Turn state machine lifting fire-and-forget `session/send` into a synchronous prompt lifecycle, with multi-signal turn-completion detection and cancel handling | State machine design (Pattern 6): exactly-once completion, ordered signal union per D-02, bypass-lane cancel, -32010 guard; scripted completion variants for tests |
</phase_requirements>

## Summary

The wire truth is settled: `docs/protocol/app-server-dialect.md` is the codec spec (transport framing, envelope grammar, method map, error codes, server→host surface, terminal signals, drift baseline), and `docs/protocol/install-discovery.md` is the discovery spec. Phase 3's research job is therefore not *what* the protocol says but *exactly what to build*: concrete Go types and function signatures for `internal/zcode` (envelope with string-or-int id, pending-call table, single writer with a stop-bypass lane, reverse-request router with the one D-05 deadline), `internal/zcode/fake` (scenario schema, two hosting modes, golden transcripts), `internal/supervisor` (prober-seamed discovery, spawn env assembly, process-group lifecycle, crash-window restart, empty-result diagnostics), and the turn machine. Everything lands under the existing `mise run ci` gate (`go test -race ./...`) with **zero new module dependencies** — stdlib covers every need, verified API-by-API against the local go1.27.3 toolchain.

The golden transcript corpus is the phase's most load-bearing artifact: the fake IS the protocol spec made executable, and its scenarios seed directly from the real scrubbed captures (`EXP-WORKSPACEKEY-01`, `EXP-TURN-01`, `EXP-SUBSCRIBE-01` — all Read in full this session, verbatim shapes quoted below). Ten scenarios cover happy turn, cancel mid-turn, the -32010 double-send rejection, unknown server→host method answered -32601, the `requestRuntimePreferences` answer shape, three completion-signal variants, EOF drain, envelope-validation negatives, oversized lines, and the scripted mid-turn `requestCredentialCipher`. Each scenario is small and deterministic, with fixed ids so goldens diff cleanly.

One review item routes here by disposition: **WR-02** (02-REVIEW-DISPOSITION) — the probe's reply-classification collision with server→host request ids. The real demux must fix this structurally: a frame with `method` never enters the reply-correlation table, whatever its id type. The 3.18.2 server observed in captures uses string ids `"server-1"…` for reverse requests, but the codec must tolerate numeric ones (the WR-02 lesson) without conflating id *types*.

**Primary recommendation:** Build in this order — (1) envelope + conn with bypass-lane writer, (2) in-process fake + golden harness, (3) supervisor with seamed discovery against the standalone fake binary, (4) turn machine — each step fully tested under `-race` before the next imports it; keep go.mod dependency-free; gate every real-desktop contact behind a manual-only env check (never CI).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| JSONL codec (envelope, correlation, ordering) | Go wrapper process (`internal/zcode`) | — | Protocol is a proprietary non-JSON-RPC dialect; no library applies (STACK.md "What NOT to Use") |
| Stop/cancel bypass lane | Go wrapper process (`internal/zcode` writer) | app-server child (own bypass queue) | Wrapper must not queue `session/stop` behind prompts; the child has the matching server-side lane (dialect §1) |
| Reverse-request answering (D-05 minimum) | Go wrapper process (`internal/zcode` router + `internal/turn`/interact seam) | — | Server blocks work on answers; read loop must never block on a handler (ARCHITECTURE Anti-Pattern 3) |
| Scriptable fake app-server | Test harness (`internal/zcode/fake`) | standalone `cmd/fake-app-server` binary | CI instrument per D-01; binary mode exists only for exec-plumbing (supervisor spawn path) |
| Desktop discovery + spawn/kill/restart | Go wrapper process (`internal/supervisor`) | OS (process groups, desktop entries, dpkg) | Process lifecycle is OS-tier work; discovery reads OS/package state per the Phase 1 matrix |
| Turn completion detection (multi-signal) | Go wrapper process (`internal/turn`) | app-server events (signal source) | Async-to-sync lift is adapter work (PITFALLS #4); child only emits signals |
| Golden protocol gate | CI (`go test -race` under `mise run ci`) | — | PROJECT done-v1 starts here; fake-only in CI, real desktop is manual-only (STATE.md blocker) |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go toolchain | 1.27 (mise pin; local 1.27.3) | Language/runtime | Already pinned in mise.toml `[tools] go = "1.27"`; go.mod declares `go 1.27` [VERIFIED: mise.toml:6, go.mod:3 — both Read this session] |
| stdlib `bufio` | go1.27.3 | JSONL line reading with raised buffer | `Scanner.Buffer(buf []byte, max int)` exists; `ErrTooLong = errors.New("bufio.Scanner: token too long")` — default 64 KiB limit must be raised [VERIFIED: `go doc bufio.Scanner` / `go doc bufio.ErrTooLong`, local go1.27.3] |
| stdlib `encoding/json` | go1.27.3 | Envelope marshal/unmarshal with `json.RawMessage` params | Typed struct fields make a stray `jsonrpc` key unrepresentable in output — the structural fix for the dialect's strict schema [VERIFIED: go doc encoding/json — stable API] |
| stdlib `os/exec` | go1.27.3 | Child spawn with `Cancel`/`WaitDelay` lifecycle fields | `exec.Cmd.Cancel` (CommandContext default = Kill) and `WaitDelay` verified present with documented semantics [VERIFIED: `go doc os/exec.Cmd`, local go1.27.3] |
| stdlib `syscall` (unix) | go1.27.3 | Process-group spawn/kill | `syscall.SysProcAttr{Setpgid bool, Pgid int}` and `Pdeathsig Signal` verified; group kill = `syscall.Kill(-pgid, SIGKILL)` [VERIFIED: `go doc syscall.SysProcAttr`, local go1.27.3] |
| stdlib `io` / `net` pipes | go1.27.3 | In-process fake transport (`io.Pipe`) | D-01 names `io.Pipe`/`net.Pipe`; `io.Pipe` gives synchronous unbuffered duplex — sufficient for the fake [ASSUMED: io.Pipe semantics from training; trivially swappable to net.Pipe] |
| stdlib `log/slog` | go1.27.3 | Diagnostics → stderr only | stdout is protocol-exclusive (CONVENTIONS; ARCHITECTURE Anti-Pattern 2) |
| stdlib `testing` + helper-process reexec | go1.27.3 | Standalone-binary exec-plumbing tests | Idiom verified in local GOROOT: `if os.Getenv("GO_WANT_HELPER_PROCESS") == "1"` + `cmd.Env = append(cmd.Env, "GO_WANT_HELPER_PROCESS=1")` at `net/http/httptest/server_test.go:426,435`; also `os/os_test.go`, `runtime/vdso_test.go`; modern `os/exec/exec_test.go` uses `testenv.Executable(t)` + `helperCommand` map. Not a documented API — a stdlib-test convention [VERIFIED: GOROOT grep, go1.27.3 source tree] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| (none — no new module dependencies) | — | — | This phase adds zero `require` lines to go.mod; `mise run tidy-diff` enforces it |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled golden compare + `-update` flag (~30 LOC) | `github.com/sebdah/goldie/v2` v2.8.0 | goldie exists on the Go module proxy (`go list -m -versions` → v2.8.0 latest) [VERIFIED: `go list -m -versions github.com/sebdah/goldie/v2`] but STACK.md explicitly blesses the hand-rolled helper as the goswitch-style stdlib alternative; Phase 3 keeps go.mod empty — adopt goldie only if golden count grows unwieldy (planner discretion) |
| `bufio.Reader.ReadBytes('\n')` | `bufio.Scanner` + raised `Buffer` | Scanner is fine once the max is raised; Reader avoids the limit entirely but you re-implement line splitting. Either satisfies PITFALL 7 — pick one and raise/drop the limit |
| `net.Pipe` for in-process fake | `io.Pipe` | `net.Pipe` yields `net.Conn` (deadlines) — only needed if the fake must model read deadlines; D-01 allows either |

**Installation:**
```bash
# No installation — zero new dependencies. Verify go.mod stays clean:
mise run tidy-diff   # fails on any go.mod/go.sum drift
```

## Package Legitimacy Audit

> Required gate run. Ecosystem for this phase is **Go modules** (not npm/PyPI) — the phase installs zero external packages.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| (none — stdlib only) | Go module proxy | — | — | — | — | No packages installed this phase |
| github.com/sebdah/goldie/v2 | Go module proxy | — | — | github.com/sebdah/goldie | evaluated, not installed | EXCLUDED — hand-rolled `-update` helper per STACK.md; module verified present (latest v2.8.0) via `go list -m -versions` if later needed |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

Note on the seam run: `gsd_run query package-legitimacy check --ecosystem npm github.com/sebdah/goldie/v2` returned `SUS` — that verdict is an artifact of querying the **wrong registry** (goldie is a Go module, not an npm package; the npm lookup returned null signals). The correct registry is the Go module proxy, where the module resolves (v2.0.0…v2.8.0) [VERIFIED: `go list -m -versions`, local go1.27.3]. Since the phase installs nothing, the audit is trivially clean; `mise run tidy-diff` is the standing guard that go.mod never grows.

## Architecture Patterns

### System Architecture Diagram

```
                       CI (mise run ci — no desktop bundle present)
   ┌──────────────────────────────────────────────────────────────────────┐
   │  Go tests (go test -race ./...)                                       │
   │                                                                       │
   │  scenario.json ──► internal/zcode/fake core ◄── programmatic Script   │
   │        (declarative)     │          │                                 │
   │              mode A: in-process Serve(io.ReadWriteCloser)             │
   │              mode B: standalone binary (cmd/fake-app-server)          │
   │                          │                                            │
   │  internal/zcode.Conn ◄───┴── io.Pipe (mode A) / exec pipes (mode B)   │
   │        │  read loop (raised buffer) ── demux:                         │
   │        │    reply (id+result/error) → pending-call table              │
   │        │    request (id+method) → reverse router (goroutine + deadline)│
   │        │    notification (method, no id) → OnNotification             │
   │        │  writer: ONE goroutine, two lanes (normal | bypass)          │
   │        │                                                              │
   │  internal/turn.Machine ── lift session/send → synchronous turn        │
   │        signals: terminal event › idle patch › quiet-window › cancel   │
   │                                                                       │
   │  internal/supervisor ── Discover (probe seams) → spawn (Setpgid,      │
   │        scrubbed env) → health watch → restart window → staged kill    │
   └──────────────────────────────────────────────────────────────────────┘
                │ mode B + real-desktop path (manual only on dev machine)
   ┌──────────▼──────────────────────────────────────────────────────────┐
   │ child process: <root>/zcode <root>/resources/glm/zcode.cjs           │
   │                 app-server --stdio                                    │
   │   env: ELECTRON_RUN_AS_NODE=1 + ZCODE_BUILTIN_PROVIDER_CONFIG_FILE    │
   │   (fake binary in tests; real Electron child only outside CI)         │
   └──────────────────────────────────────────────────────────────────────┘
```

Entry point: a scenario file (or programmatic Script) into the fake, or a Discover result into the supervisor. Processing: codec frames flow one direction through the demux; writes serialize through one goroutine with a bypass lane. Decision points: demux classifies each inbound frame by its field set (id+result/error / id+method / method-only). External dependency: only mode-B/real-desktop spawns a child process.

### Recommended Project Structure
```
cmd/
├── zcode-acp/             # existing skeleton (Phase 2) — untouched this phase
└── fake-app-server/       # standalone fake binary (D-01 mode B): thin main over internal/zcode/fake
internal/
├── zcode/                 # BRIDGE-01 codec
│   ├── envelope.go        # Envelope, ID, WireError, method constants, error-code constants
│   ├── conn.go            # Conn: read loop, demux, pending-call table, router hooks
│   ├── writer.go          # single writer goroutine + bypass lane
│   ├── call.go            # Call/Notify/Stop/Reply helpers, pending-call entry
│   ├── fake/              # BRIDGE-02
│   │   ├── fake.go        # scriptable core: Serve(rw io.ReadWriteCloser, script) 
│   │   ├── scenario.go    # JSON scenario schema load + validate
│   │   ├── golden.go      # transcript record/normalize/compare + -update flag helper
│   │   └── testdata/
│   │       ├── scenarios/*.json   # declarative scripts (D-01)
│   │       └── golden/*.golden    # replayed transcripts, regenerated via -update
│   └── turn/              # BRIDGE-04 (D-06 allows internal/turn or inside zcode — prefer sibling
│   │   └── machine.go     #   package internal/turn per D-06's first naming)
├── supervisor/            # BRIDGE-03
│   ├── discover.go        # matrix probers + probe list + diagnostics error
│   ├── spawn.go           # env assembly, Setpgid, pipe wiring, scrubbed env
│   └── supervise.go       # health watch, crash-window restart, staged shutdown
└── ...
testdata/                  # (per-dackage testdata preferred so `go test` resolves it)
```

### Pattern 1: Envelope types + strict demux (the codec core)

**What:** One wire line = one envelope. The grammar is strict (dialect §1, quoted verbatim): client→server request `{"id":<id>,"method":"<method>","params":{…}}`; server reply `{"id":<id>,"result":{…}}` or `{"id":<id>,"error":{"code":<int>,"message":"<string>","data":{…}}}`; notifications/events carry `method`+`params` and **no** `id` [VERIFIED: docs/protocol/app-server-dialect.md:24 — Read this session]. Ids are string-or-int; replies echo the request id verbatim [VERIFIED: docs/protocol/app-server-dialect.md:27].

**The `jsonrpc` defense is structural:** the Envelope struct has no `jsonrpc` field and `Conn` only accepts typed structs — a JSON-RPC-shaped frame is unrepresentable in outbound bytes. Unit test asserts byte-level absence of `"jsonrpc"` in every encoded frame.

**Demux rule (fixes WR-02 structurally):** a frame with `method` never enters the reply-correlation table, regardless of id. Reply correlation only for frames with `id` AND (`result` OR `error`) AND no `method`.

**When to use:** Always; this is the whole back-leg grammar.

**Example:**
```go
// Source: docs/protocol/app-server-dialect.md §1 grammar; captures read this session
// ID is string-or-int per the dialect; correlate by exact value AND type
// ("5" the string and 5 the int are different wire values).
type ID struct {
    Int   int64
    Str   string
    IsInt bool
}

func (id ID) key() string { // canonical pending-table key
    if id.IsInt {
        return "i:" + strconv.FormatInt(id.Int, 10)
    }
    return "s:" + id.Str
}

type Envelope struct {
    ID     *ID             `json:"id,omitempty"`
    Method string          `json:"method,omitempty"`
    Params json.RawMessage `json:"params,omitempty"`
    Result json.RawMessage `json:"result,omitempty"`
    Error  *WireError      `json:"error,omitempty"`
}

type WireError struct {
    Code    int             `json:"code"`
    Message string          `json:"message"`
    Data    json.RawMessage `json:"data,omitempty"` // observed live: {name:"ZodError"...} / ModelProtocolError shape
}

// Classification (demux):
//   reply:          e.ID != nil && e.Method == "" && (e.Result != nil || e.Error != nil)
//   reverse request: e.ID != nil && e.Method != ""
//   notification:    e.ID == nil && e.Method != ""
```

Live wire evidence (all Read this session): reverse requests carry string ids — `{"id":"server-1","method":"session/requestRuntimePreferences","params":{"sessionId":"sess_<uuid-2>","scope":"runtime-materialization"}}` [VERIFIED: experiments/captures/EXP-WORKSPACEKEY-01.jsonl:8]; error reply with numeric id echoes — `{"error":{"code":-32603,"data":{…,"code":"invalid_model_request"},"message":"Reasoning level is required for zai-api/GLM-5.3-Flash"},"id":4}` [VERIFIED: experiments/captures/EXP-TURN-01.jsonl:32]; send reply — `{"id":5,"result":{"accepted":true,"sessionId":"sess_<uuid-2>","stateRevision":1}}` [VERIFIED: experiments/captures/EXP-TURN-01.jsonl:36].

### Pattern 2: Single writer goroutine with a stop-bypass lane

**What:** All outbound frames serialize through ONE writer goroutine (PITFALL 7 — never interleave writes on the child's stdin). Two lanes feed it: the normal ordered queue and a bypass lane that jumps the queue for `session/stop` (and reverse-request replies). The server's own transport justifies both: ordinary requests execute serially; "the bypass set is exactly `session/stop`, `workspace/cancelGenerateText`, and the MCP-UI concurrent set"; and "reverse-request responses bypass the queue (answering them from the queue would deadlock the pairing)" [VERIFIED: docs/protocol/app-server-dialect.md:29 — Read this session].

**Writer priority semantics:** check the bypass channel non-blockingly before each normal write; a normal write already in progress finishes first (mirrors the server: bypass runs as soon as the preceding request has *entered* its handler, not mid-write). Never starve: after one bypass frame, re-check the normal channel.

**When to use:** Always; every `Call`/`Notify` goes through the queue, every `Stop`/`Reply`/`ReplyError` through the bypass.

**Example:**
```go
// Source: ARCHITECTURE.md Pattern 2 + dialect §1 scheduling facts
for {
    select {
    case f := <-c.bypass:   // session/stop, reverse-request replies — jump the queue
        writeFrame(f)
        continue
    case <-c.done:
        return
    default: // no bypass pending
    }
    select {
    case f := <-c.bypass:
        writeFrame(f)
    case f := <-c.queue:    // ordered Calls/Notifies
        writeFrame(f)
    case <-c.done:
        return
    }
}
```

### Pattern 3: Reverse-request router with per-method deadlines (D-05 minimum)

**What:** Server→host requests (id+method) are dispatched to handler goroutines — never inline in the read loop (ARCHITECTURE Anti-Pattern 3: the app-server blocks mid-turn on answers and re-announces; a blocked read loop deadlocks everything). Phase 3 answers exactly one method: `session/requestRuntimePreferences` within a **15 s deadline** with the static defaults; unknown methods answer `-32601` (probe precedent: answering -32601 did not break create flows — [VERIFIED: docs/protocol/app-server-dialect.md:85]; recorded in captures as hostSent lines). The router owns per-method deadline timers (the seam Phase 5's policy table will extend).

**When to use:** Always; `requestCredentialCipher` mid-turn is scripted in the fake but NOT answered on the real path in Phase 3 (D-05) — the real-server path records the deadline expiry honestly.

**Example:**
```go
// Source: D-05 (CONTEXT.md) + EXP-WORKSPACEKEY-01.jsonl:9 answer shape
const rpDeadline = 15 * time.Second

case "session/requestRuntimePreferences":
    ctx, cancel := context.WithTimeout(ctx, rpDeadline)
    defer cancel()
    _ = c.Reply(env.ID, map[string]any{
        "nativeSearchEnhancementsEnabled": false,
        "memoryEnabled": false,
    })
default:
    _ = c.ReplyError(env.ID, &zcode.WireError{
        Code: -32601, Message: "unsupported host request: " + env.Method,
    })
```
Verbatim answer shape proven live: host replies `{"id":"server-1","result":{"nativeSearchEnhancementsEnabled":false,"memoryEnabled":false}}` [VERIFIED: experiments/captures/EXP-WORKSPACEKEY-01.jsonl:9, EXP-TURN-01.jsonl:34]. Note the request fires at create with `scope:"runtime-materialization"` [VERIFIED: EXP-WORKSPACEKEY-01.jsonl:8] and AGAIN at send with `scope:"user-execution"` [VERIFIED: EXP-TURN-01.jsonl:33].

### Pattern 4: Scriptable fake app-server (D-01) + golden transcripts

**What:** One scripting core, two hosting modes. The core is a `Serve(rw io.ReadWriteCloser, script Script)` loop that: reads envelopes, matches requests against the scenario table, replies (with configured delay), emits scripted events, issues its own reverse requests (blocking the turn until answered or timed out), and honors the terminal signal (EOF drain ~100 ms then exit, or error exit). Scenario files are declarative JSON; a programmatic `Script` interface covers dynamic cases (echo sessionId from create into send events).

**Golden transcripts:** per scenario, record every frame in wire order with a direction tag; normalize nothing at runtime (scenarios use fixed ids so replays are byte-deterministic); `-update` flag regenerates; diff fails the test. Location per D-01: `internal/zcode/fake/testdata/golden/` (package testdata so `go test` resolves it).

**When to use:** Every protocol test; CI replays all scenarios under `-race`.

**Example scenario schema (Claude's discretion area — concrete starting shape):**
```json
{
  "name": "happy-turn",
  "onRequest": [
    {"method": "session/create",
     "reply": {"result": {"session": {"sessionId": "sess_fix1"}, "protocol": {"name": "ZCode Protocol", "version": 1}}},
     "sideEffects": [
       {"emit": {"method": "session/lifecycle/changed", "params": {"sessionId": "sess_fix1", "state": "starting", "generation": 1, "revision": 1, "lastError": null}}},
       {"reverse": {"method": "session/requestRuntimePreferences",
                    "params": {"sessionId": "sess_fix1", "scope": "runtime-materialization"},
                    "timeoutMs": 15000,
                    "expectAnswer": {"result": {"nativeSearchEnhancementsEnabled": false, "memoryEnabled": false}}}},
       {"emit": {"method": "session/lifecycle/changed", "params": {"sessionId": "sess_fix1", "state": "ready", "generation": 1, "revision": 2, "lastError": null}}}
     ]},
    {"method": "session/send",
     "reply": {"result": {"accepted": true, "sessionId": "sess_fix1", "stateRevision": 1}},
     "sideEffects": [
       {"emit": {"method": "state.updated", "params": {"patch": {"status": "running"}, "reason": "prompt_started", "revision": 1, "scope": "session", "sessionId": "sess_fix1", "type": "state.updated"}}},
       {"emit": {"method": "session/event", "params": {"type": "turn.started", "seq": 4, "turnId": "turn_fix1", "deliveryKind": "desktop-continuous"}}},
       {"emit": {"method": "session/event", "params": {"type": "turn.completed", "seq": 9, "turnId": "turn_fix1", "deliveryKind": "desktop-continuous"}}}
     ]},
    {"method": "session/stop", "reply": {"result": {}}}
  ],
  "unknownMethod": {"reply": {"error": {"code": -32601, "message": "MethodNotFound"}}},
  "eofDrainMs": 100
}
```
Event shapes above quote the live capture verbatim where load-bearing: lifecycle `starting→ready` with `generation`/`revision`/`lastError` [VERIFIED: EXP-WORKSPACEKEY-01.jsonl:7,14]; state patch `{patch:{status:"running"},reason:"prompt_started",revision,scope:"session",type:"state.updated"}` [VERIFIED: EXP-TURN-01.jsonl:35]; send ack result keys `accepted`/`sessionId`/`stateRevision` [VERIFIED: EXP-TURN-01.jsonl:36]; subscribe result `{eventSeq:0,events:[],sessionId}` [VERIFIED: EXP-TURN-01.jsonl:31]. Terminal `turn.completed` literal is V-SRC-only on the real wire (bundle counts ×7 quoted) [VERIFIED: EXP-TURN-01.jsonl:56 — `"turn.completed (quoted literal)":7`] — in the fake it is by-construction valid: the scenario declares it.

**Standalone binary (mode B):** `cmd/fake-app-server` main reads `--scenario <path>` and serves over its own stdio, so the supervisor's spawn path (exec pipes, Setpgid, group kill, EOF behavior) is exercised against a real child process in tests without the desktop bundle.

### Pattern 5: Supervisor — seamed discovery, spawn, group-kill, crash window

**What:** Discovery implements the Phase 1 recommended order exactly [VERIFIED: docs/protocol/install-discovery.md:93-97 — Read this session]: (1) env/flag override first; (2) desktop entry (`xdg-mime query default x-scheme-handler/zcode` → resolve `Exec`); (3) AppImage paths (`~/Applications/ZCode*.AppImage` file + symlink, then `~/Applications/zcode-*/squashfs-root/`); (4) package paths (`/opt/ZCode/resources/glm/zcode.cjs`, `dpkg-query -W zcode` + `dpkg -L zcode` with the empty-list rc-remnant case, `rpm` where applicable); (5) never volatile `/tmp/.mount_*`, never localhost listeners. **Empty result is a first-class outcome**: the error names every probe attempted with its result [VERIFIED: docs/protocol/install-discovery.md:34].

Testability comes from function seams — each probe is an injectable func, so the no-install test stubs everything empty and asserts the diagnostics list; the positive test seeds a fixture tree.

**Spawn contract** (verbatim canonical line [VERIFIED: docs/protocol/install-discovery.md:48-52]):
```
ELECTRON_RUN_AS_NODE=1 \
ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=<newest zcode-builtin.json under ~/.zcode/v2/runtime/provider/linux-x86_64/> \
<root>/zcode <root>/resources/glm/zcode.cjs app-server --stdio
```
Env assembly: explicit allowlist (PITFALLS Security Mistakes: copy `PATH`, `HOME`; set `ELECTRON_RUN_AS_NODE=1`, `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE`; scrub `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY` or set `NO_PROXY` — PITFALL 9, D-03). Process group: `SysProcAttr{Setpgid: true}` [VERIFIED: go doc syscall.SysProcAttr]. Shutdown order: close child stdin → wait (child drains ~100 ms by design) → grace → `syscall.Kill(-pgid, SIGKILL)` backstop → `cmd.Wait()` reaps (never `Wait()` before stderr/stdout pumps finish — PITFALL 7). Health watch = stdin liveness + child exit; no protocol liveness frame exists (D-03). Restart: 3 attempts within a 60 s window, exponential backoff, then fail loud (D-03; codex-acp `CODEX_ACP_APP_SERVER_CRASH_LIMIT`/`CRASH_WINDOW_MS` precedent — PITFALLS/ARCHITECTURE, MEDIUM confidence README-derived).

**When to use:** Always for the child lifecycle; `Conn` stays spawn-agnostic (takes any `io.ReadWriteCloser`) which is what lets the fake replace the real child in tests (ARCHITECTURE internal boundary: "supervisor ↔ zcode: owns pipes, hands Conn a ReadWriteCloser").

**Example:**
```go
// Source: install-discovery.md §Recommended discovery order + dialect §1 EOF facts;
// syscall semantics verified via go doc (go1.27.3)
type Install struct {
    Root        string // dir containing the Electron binary
    Binary      string // <root>/zcode
    Entrypoint  string // <root>/resources/glm/zcode.cjs (invariant relative path)
    BuiltinFile string // ZCODE_BUILTIN_PROVIDER_CONFIG_FILE value
    Layout      string // "unpacked-appimage" | "appimage" | "deb" | "rpm" | "explicit"
}

type Probe struct{ Name, Query, Result string } // one row of the diagnostics list

type Probers struct { // function seams — stubbed in tests
    XDGQuery  func(ctx context.Context) (string, error) // xdg-mime query default x-scheme-handler/zcode
    AppDirs   func() []string                           // ~/Applications globs
    DpkgQuery func(ctx context.Context) (string, error)
    RpmQuery  func(ctx context.Context) (string, error)
    Home      string
    Override  func() (Install, bool) // env/flag override wins (order step 1)
}

// Discover returns every probe attempt even on failure — the diagnostics ARE the product.
func (p Probers) Discover(ctx context.Context) (Install, []Probe, error)

func (s *Supervisor) spawn(ctx context.Context, inst Install) (*exec.Cmd, error) {
    cmd := exec.CommandContext(ctx, inst.Binary, inst.Entrypoint, "app-server", "--stdio")
    cmd.Env = childEnv(inst) // allowlist + scrub proxy vars (D-03/PITFALL 9)
    cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
    return cmd, cmd.Start()
}

// shutdown: stdin end → wait for exit (child drains ~100 ms) → grace → group SIGKILL → Wait()
func (s *Supervisor) killGroup(cmd *exec.Cmd) {
    _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) // negative pid = whole group
}
```

### Pattern 6: Turn machine — async lift with multi-signal completion (D-02)

**What:** Per turn: send `session/send` (fire-and-forget — the ack `{accepted:true,…}` arrives immediately [VERIFIED: EXP-TURN-01.jsonl:36]), then watch completion signals in priority order and answer the (future) `session/prompt` exactly once. States: `idle → running → cancelling → done`. The machine owns the response write (PITFALL 8: "Always answer `session/prompt` exactly once — the response write must be owned by the state machine").

**Signal union (D-02 ordered):**
1. **Terminal event** `turn.completed` / `turn.failed` — the v4 taxonomy literals; bundle counts `"turn.completed"` quoted ×7 / `"turn.failed"` quoted ×8; kind literals `turn-completed`/`turn-failed`/`turn-started` each carrying `turnId` [VERIFIED: EXP-TURN-01.jsonl:56 source-enumeration; docs/protocol/app-server-dialect.md:132-134]. On the REAL wire these are V-SRC-only (never observed live — the D-05 degradation record [VERIFIED: EXP-TURN-01.jsonl:55]); in the fake they are scenario-declared.
2. **Idle patch** — `state.updated` whose `patch.status` leaves `"running"` (only `"running"` was ever observed live [VERIFIED: EXP-TURN-01.jsonl:54 — `turnStatusSeen:["running"]`]); the dialect warns "do not rely on it alone" [VERIFIED: docs/protocol/app-server-dialect.md:140].
3. **Quiet-window fallback** — the only path proven end-to-end: poll `session/messages`, hardened form = send-latency floor (3× quiet-ms after the send reply) + settle gate, quiet = 2.5 s [VERIFIED: docs/protocol/app-server-dialect.md:141; probe implements exactly this at experiments/app-server-probe.mjs:570-597 — `QUIET_SEND_FLOOR_MULTIPLE = 3`].
4. **Cancel** — `session/stop` via the bypass lane immediately; prompt resolves `cancelled` exactly once; idempotent (cancel-after-complete = no-op).

**Concurrency guard:** a second concurrent `session/send` on the same session is rejected -32010 `"A prompt is already running for this session"` (literal ×1 in bundle) [VERIFIED: docs/protocol/app-server-dialect.md:156]. The machine enforces this wrapper-side too — surface a clean error, never silently queue.

**TurnWatcher interface** (D-02; polling first, subscribe later):
```go
// Source: D-02 + ARCHITECTURE Pattern 5
type TurnWatcher interface {
    // Watch delivers envelopes for sessionID until stop; implementations:
    // pollWatcher{} (session/messages + quiet window) now; subscribeWatcher{} later.
    Watch(ctx context.Context, sessionID string, onEvent func(zcode.Envelope)) (stop func(), err error)
}
```

**When to use:** Always; this is BRIDGE-04. Phase 4's ACP front will call `Machine.Run`/`Wait` per `session/prompt`.

**Example:**
```go
// Source: D-02 decisions; PITFALL 8 exactly-once rule
type StopReason string // "completed" | "failed" | "cancelled" | "error"

type Machine struct {
    conn    *zcode.Conn
    sessID  string
    watcher TurnWatcher
    // ...state, cancel hooks, completion sync (chan struct{} + sync.Once)
}

// Run lifts one fire-and-forget send into a synchronous turn.
func (m *Machine) Run(ctx context.Context, content string) (StopReason, error) {
    // 1. guard: reject concurrent send (-32010 wrapper-side)
    // 2. conn.Call("session/send", {sessionId, content}) — ack only
    // 3. watch signals; classify via one observe(Envelope) path
    // 4. terminal signal or cancel → resolve StopReason exactly once
}
func (m *Machine) Cancel() { _ = m.conn.Stop(m.sessID) /* bypass lane */ }
```

### Anti-Patterns to Avoid
- **JSON-RPC framing on this leg:** any `jsonrpc` key is rejected by the child with `-32600 "Invalid ZCode Protocol message"` (zod `unrecognized_keys`) [VERIFIED: docs/protocol/app-server-dialect.md:25]. Never reuse the ACP codec or sourcegraph/jsonrpc2 here (STACK.md "What NOT to Use").
- **Blocking the read loop on a handler:** reverse requests must be served from handler goroutines; the server blocks mid-turn and re-announces — inline handling deadlocks (ARCHITECTURE Anti-Pattern 3).
- **Reply correlation eating reverse requests:** storing any id-bearing frame in the pending table (the probe's WR-02 bug) turns a no-reply verdict into a false accept; demux on the presence of `method` (02-REVIEW WR-02, routed here by disposition).
- **Goroutine-per-message writes:** reorders frames and interleaves stdin writes (PITFALL 7) — single writer only.
- **Trusting stdin-EOF to stop the child:** explicitly kill the process group after a short grace; the child drains ~100 ms on EOF but Electron helpers may survive the parent otherwise (PITFALL 6, D-03).
- **`bufio.Scanner` defaults:** 64 KiB `ErrTooLong` truncation on big payloads (PITFALL 7) — raise `Scanner.Buffer` explicitly.
- **Early empty-success read:** the child exiting 0 with zero captured lines is the EOF-pitfall **failure signature**, never success [VERIFIED: docs/protocol/app-server-dialect.md:30; probe mirrors this guard at app-server-probe.mjs:661-673].

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSONL line reading w/ big lines | custom chunked reader | `bufio.Scanner` + `Scanner.Buffer(make([]byte,0,64*1024), 16<<20)` | stdlib handles split/flush edge cases; limit raise is one call [VERIFIED: go doc bufio.Scanner] |
| Child process lifecycle | custom pid tracking / polling /proc | `os/exec` `CommandContext` + `Cancel` + `WaitDelay` + `SysProcAttr{Setpgid}` | stdlib owns reaping (`Wait`), context kill, and I/O-goroutine bounds (`WaitDelay`) [VERIFIED: go doc os/exec.Cmd] |
| Process-group kill | killing each pid found via ps | `syscall.Kill(-pgid, SIGKILL)` | one atomic group operation; per-pid walks race with helpers spawning [VERIFIED: go doc syscall.SysProcAttr + PITFALLS citations golang/go#27505, felixge article] |
| Standalone-binary tests | building a test binary via `go build` in TestMain | helper-process reexec idiom (`GO_WANT_HELPER_PROCESS=1`, reexec `os.Args[0]` / `testenv.Executable(t)`) | the stdlib's own suite does exactly this — no build orchestration, works under `-race` [VERIFIED: GOROOT net/http/httptest/server_test.go:426,435; os/exec/exec_test.go helperCommand] |
| JSON envelope encode/decode | reflection-based custom codec | `encoding/json` struct tags + `json.RawMessage` | typed fields make `jsonrpc` unrepresentable; RawMessage preserves params byte-exactly for round-trip tests |
| Test doubles for time | ad-hoc tick channels | `time.Timer`/`time.After` + short test durations; (option) `testing/synctest` if flake appears | stdlib; keep fake timings deterministic via scenario `delayMs`, not wall-clock racing |

**Key insight:** every "hard" part of this phase (framing, correlation, process groups, reexec) is a solved stdlib problem — the only genuinely custom code is the dialect itself (~200 LOC envelope/conn per ARCHITECTURE) and the scenario interpreter.

## Common Pitfalls

### Pitfall 1: `bufio.Scanner` 64 KiB truncation
**What goes wrong:** `session/event` payloads with tool output exceed 64 KiB → `ErrTooLong`, stream silently stops mid-turn.
**Why:** `bufio.MaxScanTokenSize` default. **How to avoid:** `sc.Buffer(make([]byte, 0, 64*1024), 16<<20)` before the scan loop [VERIFIED: go doc bufio.ErrTooLong/Scanner.Buffer].
**Warning signs:** "token too long" in logs; events stop after large tool results.
**(PITFALLS.md #7; raises are D-06 "raised line buffer".)**

### Pitfall 2: Undrained child stderr
**What goes wrong:** Electron prints GTK/Wayland noise; the 64 KB pipe buffer fills; the child blocks mid-turn — mysterious hang.
**How to avoid:** one drain goroutine per pipe; forward to wrapper stderr (capped, like the probe's 64 KiB cap). Never merge child stderr into stdout.
**(PITFALLS.md #7.2; probe precedent app-server-probe.mjs:529-536.)**

### Pitfall 3: Goroutine topology drift
**What goes wrong:** same-goroutine read+write deadlocks; goroutine-per-message reorders chunks; `cmd.Wait()` before pumps finish deadlocks on pipe buffers.
**How to avoid:** fixed topology — one reader per pipe (stdout, stderr), one writer per sink (child stdin), demux dispatches by id/method; `Wait()` last. Test under `-race` with interleaved traffic.
**(PITFALLS.md #7.1/#7.3; ARCHITECTURE Pattern 2.)**

### Pitfall 4: Cancel races / prompt answered twice or never
**What goes wrong:** cancel lands before send ack, after completion, or during a pending interaction; the prompt request hangs the client or is answered twice.
**How to avoid:** one state machine per turn with `cancelling` state; completion resolved exactly once (`sync.Once`-style); idempotent cancel; cancel sends `session/stop` through the bypass lane immediately (D-02).
**(PITFALLS.md #8; D-02.)**

### Pitfall 5: The -32010 double-send rejection
**What goes wrong:** a second `session/send` while a turn runs is refused `-32010 "A prompt is already running for this session"` [VERIFIED: docs/protocol/app-server-dialect.md:156].
**How to avoid:** wrapper-side guard rejects concurrent sends with a clean error; map the numeric code (never the message text — messages are not a stable contract [VERIFIED: docs/protocol/app-server-dialect.md:163]).
**(D-02; PITFALLS.md #4.)**

### Pitfall 6: EOF drain misunderstanding
**What goes wrong:** closing stdin immediately after a request loses the reply — the child drains ~100 ms then exits; an unanswered request is silently dropped; exit-0-with-zero-lines masquerades as success.
**How to avoid:** hold stdin until collection completes; treat `exit 0 + zero lines` as the failure signature; shutdown order = stdin close → wait → grace → group kill [VERIFIED: docs/protocol/app-server-dialect.md:30].
**(Dialect §1; probe guard app-server-probe.mjs:661-673.)**

### Pitfall 7: Restart-window bookkeeping
**What goes wrong:** crash counter never resets (healthy long uptime still trips the limit) or resets wrongly (crash loop never trips).
**How to avoid:** sliding 60 s window: crash increments only within the window; a stable run longer than the window resets it; on tripping, fail loud with remaining-turn failures, not a loop (D-03; ARCHITECTURE Pattern 4 example).
**(D-03.)**

### Pitfall 8: WR-02 reply-collision in the real demux
**What goes wrong:** a server→host request with a numeric id lands in the reply table; a never-answered call reports accepted.
**How to avoid:** demux rule — method-bearing frames route to the router, never the pending table; pending table keyed by type-tagged canonical id (`i:`/`s:`) so `"5"`/`5` never conflate. Unit test: reverse request with numeric id colliding with an in-flight call id must NOT resolve it.
**(02-REVIEW WR-02; disposition routes the fix here.)**

### Pitfall 9: Desktop-entry/URL-handler clash and decoys
**What goes wrong:** discovery resolves `/tmp/.mount_*` (volatile) or a localhost listener (decoy answering 404/400) as the install.
**How to avoid:** never record mounts as spawn roots; never probe listeners; the discovery order and its do-not-rely rows are quoted in Pattern 5 [VERIFIED: docs/protocol/install-discovery.md:25,44,97].
**(install-discovery.md rows 5 + §Recommended order step 5.)**

### Pitfall 10: Real-desktop leakage into CI
**What goes wrong:** a test spawns the real Electron child in CI — quota risk, flake, and on runner machines a confusing no-install failure.
**How to avoid:** every real-child test is guarded (env check like `ZCODE_ACP_REAL_E2E=1` skipped by default, or a build tag); CI runs the fake only. The fake binary is the CI child; the real child is manual-tier on dev machines only.
**(STATE.md Phase-2 blocker: "CI must never call api.z.ai or spawn the real desktop".)**

## Code Examples

### Envelope round-trip unit test shape (BRIDGE-01)
```go
// Source: dialect §1 grammar; struct per Pattern 1
func TestEncodeOmitsJSONRPC(t *testing.T) {
    frame, err := json.Marshal(zcode.Envelope{
        ID:     &zcode.ID{Int: 5, IsInt: true},
        Method: "session/create",
        Params: json.RawMessage(`{"workspace":{"workspacePath":"/w","workspaceKey":"/w"}}`),
    })
    if err != nil { t.Fatal(err) }
    if bytes.Contains(frame, []byte("jsonrpc")) {
        t.Fatalf("JSON-RPC key leaked to wire: %s", frame) // dialect: -32600 unrecognized_keys
    }
}

func TestDemuxReverseRequestNeverResolvesPendingCall(t *testing.T) { // WR-02 fix
    // pending table has id i:7 in flight; inbound frame {"id":7,"method":"interaction/requestPermission",...}
    // must route to the request router, NOT resolve the pending call.
}
```

### Writer bypass-lane ordering test
```go
// Source: Pattern 2; dialect §1 bypass semantics
func TestStopJumpsQueue(t *testing.T) {
    // queue 3 slow Calls, then Stop; assert wire order on the fake side:
    // stop precedes at most the in-flight write, never queues behind both.
}
```

### Helper-process reexec test (mode B — verified idiom)
```go
// Source: GOROOT net/http/httptest/server_test.go:426,435 (verified pattern); PITFALLS/STACK note: convention, not API
func TestFakeBinarySpawnPath(t *testing.T) {
    if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
        fake.Main(os.Getenv("FAKE_SCENARIO")) // runs as the "child"; serves scenario over stdio
        return
    }
    cmd := exec.Command(os.Args[0], "-test.run=TestFakeBinarySpawnPath")
    cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "FAKE_SCENARIO=scenarios/happy-turn.json")
    cmd.Stdin, cmd.Stdout = os.Pipe()... // exec plumbing identical to the supervisor's spawn
    // assert frames flow and group-kill leaves no survivors
}
```

### Supervisor negative test (no install — stubbed probes)
```go
// Source: install-discovery.md:34 "empty result ... naming every probe attempted"
func TestDiscoverNoInstallListsEveryProbe(t *testing.T) {
    p := supervisor.Probers{ /* all seams stubbed empty */ Home: t.TempDir() }
    _, probes, err := p.Discover(context.Background())
    if err == nil { t.Fatal("want error") }
    for _, name := range []string{"override", "xdg-entry", "appimage-glob", "unpacked-glob", "dpkg", "opt-path", "rpm"} {
        if !slices.ContainsFunc(probes, func(p supervisor.Probe) bool { return p.Name == name }) {
            t.Errorf("diagnostics missing probe %q: %+v", name, probes)
        }
    }
}
```

### Turn exactly-once test
```go
// Source: D-02; PITFALL 8
func TestCancelMidTurnAnswersOnce(t *testing.T) {
    // scenario: send ack → events → (test calls Cancel) → stop ack → machine resolves
    // exactly one StopReason("cancelled"); a second Cancel is a no-op.
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| JSON-RPC library for both legs | separate codecs; plain JSONL envelope on the back leg | Phase 1 verification (jsonrpc key rejected -32600) | no `jsonrpc2` dependency anywhere; codec is hand-rolled per STACK.md |
| Single-signal turn completion (`state.updated` idle only) | multi-signal ordered union (terminal event › idle patch › quiet window) | Phase 1 honesty record (idle never arrived live) | D-02 encodes the union; fake scripts all variants |
| Probe-script process handling (detached + backstop) | product-grade supervisor (`Setpgid` + `WaitDelay` + crash window) | this phase | the probe's proven shapes translate into `internal/supervisor` |
| Golden tests via external framework | stdlib hand-rolled `-update` helper | STACK.md blessing (goswitch style) | go.mod stays dependency-free through Phase 3 |

**Deprecated/outdated:**
- `CODEX_ACP_APP_SERVER_CRASH_*` env names: codex-acp precedent for the *concept*; adopt the policy (3/60 s), not the names — ours are config-surface decisions (planner names them per the Phase 4 config table style).
- Treating `session/subscribe` as the watcher: ACCEPTED on 3.18.2 [VERIFIED: docs/protocol/app-server-dialect.md:99] but the subscribe-based implementation is deferred (CONTEXT Deferred Ideas); the polling watcher is the Phase 3 implementation.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `io.Pipe` (synchronous, unbuffered) is sufficient for the in-process fake; no read deadlines needed | Standard Stack / Pattern 4 | Low — swap to `net.Pipe` (deadlines) is a one-line transport change behind the same interface |
| A2 | Helper-process reexec idiom behaves under `go test -race` for a helper that serves stdin/stdout | Pattern 4 mode B / Code Examples | Medium — if flaky, fall back to a tiny built binary via `go build` in TestMain (still no module deps); idiom is GOROOT-verified but not under race in all stdlib uses |
| A3 | Terminal-signal classification accepts `turn.completed`/`turn.failed` arriving as v4-family event payloads (exact envelope wrapping unverified live — V-SRC-only) | Pattern 6 | Medium — real wrapping lands in Phase 5/6 when credential-answering turns run; the fake defines the shape now, and the signal union degrades gracefully to quiet-window |
| A4 | `state.updated` idle patch exists with `patch.status` leaving `"running"` (only `"running"` observed live) | Pattern 6 | Medium — same as A3: V-SRC-only on real wire; quiet-window fallback is the proven path and covers absence |
| A5 | Windows group-kill needs a build-tagged shim (Job Objects); v1 validates Ubuntu only | Pattern 5 | Low — code stays cross-platform behind a narrow seam (PROJECT.md platform scope) |
| A6 | Fake binary as `cmd/fake-app-server` is acceptable to ship in the repo (D-01 names it `fake-app-server`) | Project structure | Low — alternative is a test-only main under `internal/zcode/fake`; planner can relocate without design impact |
| A7 | Quiet-window parameters (2.5 s quiet, 3× send-latency floor) port from the probe harness into the poll watcher unchanged | Pattern 6 | Low — values are tunable test knobs; mis-tuning shows as flaky turn tests, not silent breakage |
| A8 | `xdg-mime query default x-scheme-handler/zcode` via `exec` is the right seam shape for desktop-entry probing on Ubuntu GNOME | Pattern 5 | Low — matrix row verified in Phase 1 [VERIFIED: docs/protocol/install-discovery.md:65-69]; only the Go invocation shape is new |

## Open Questions

1. **Exact terminal-event envelope wrapping for `turn.completed`/`turn.failed`**
   - What we know: the literals exist in the v4 taxonomy; kind schemas carry `turnId` [VERIFIED: EXP-TURN-01.jsonl:56]; never observed live (V-SRC-only).
   - What's unclear: whether they arrive as `session/event` payloads (`type:"turn.completed"`), `v4/telemetry/event` kinds, or v4 conversation frames.
   - Recommendation: define the machine's classifier on `turnId` + terminal-kind matching across the known carriers (all three are already demuxed); the fake fixes the shape for tests; reconcile live in Phase 5/6.
2. **Whether `session/messages` polling needs delta cursors** (quiet-window fallback reads it repeatedly)
   - What we know: result key set observed (`messages` in the create reply's sibling surface; poll shape proven by the predecessor shim, V-RPT).
   - What's unclear: exact `session/messages` params/result schema on 3.18.2.
   - Recommendation: encode the poll watcher against a minimal `{sessionId}` request and treat messages opaquely for length/time heuristics (the probe only needs presence, not content); verify shape when the real-server smoke runs (manual tier).
3. **Golden-fixture id strategy** (planner discretion): fixed synthetic ids (`sess_fix1`) make goldens byte-deterministic; a normalization layer would allow replaying scrubbed real transcripts verbatim. Recommendation: fixed ids in scenarios; keep scrubbed captures as *seed* documentation, not goldens.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | 1.27.3 local; mise pins 1.27 [VERIFIED: mise.toml:6, `go version`] | — |
| mise | CI gate runner | ✓ | 2026.10.17 [VERIFIED: `mise --version`] | — |
| golangci-lint | lint gate | ✓ | 2.15.0 via mise [VERIFIED: `mise current`] | — |
| node 26 | experiments suite (`mise run test-experiments`) | ✓ | 26.13.1 [VERIFIED: `node --version`] | — |
| ZCode 3.18.2 desktop install | manual real-spawn check (BRIDGE-03 positive path) | ✓ | unpacked squashfs-root present at the documented path [VERIFIED: `ls` this session] | fake binary in tests; manual tier only |
| AppImage + symlink | discovery positive rows | ✓ | `ZCode-3.18.2-linux-x64.AppImage` + `ZCode.AppImage` present [VERIFIED: `ls` this session] | — |
| deb install | discovery deb row positive | ✗ (rc-remnant only: `3.17.2-9661 deinstall ok config-files`; `/opt/ZCode` absent) [VERIFIED: `dpkg-query`, `ls` this session] | — | negative-row unit tests use stubs; diagnostics cover it |
| Desktop bundle in CI | must be ABSENT | ✓ (correctly absent — hosted runners) | — | the whole phase is designed to prove everything fake-only |

**Missing dependencies with no fallback:** none — the phase is deliberately CI-complete without the desktop.

**Missing dependencies with fallback:** deb layout not installed on this machine → discovery deb row is exercised via stubbed `DpkgQuery` seams and the rc-remnant negative case; rpm likewise (no rpm tooling here — matches the discovery doc's own DEB-GENERALIZED status).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` under `go test -race` (existing `mise run test` task); helper-process reexec for the standalone-binary tier |
| Config file | none new — existing `mise.toml` (`[tasks.test] run = "go test -race -count=1 ./..."`) + `.golangci.yml` [VERIFIED: mise.toml Read this session] |
| Quick run command | `go test -race ./internal/zcode/... ./internal/supervisor/...` |
| Full suite command | `mise run ci` (build → vet → lint → test → test-experiments) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BRIDGE-01 | envelope encode omits `jsonrpc`; decode accepts string-or-int ids; demux classifies reply/request/notification; WR-02 route (method-frame never resolves pending call); error-code constants round-trip | unit (`-race`) | `go test -race ./internal/zcode -run TestEnvelope -x` (and `TestDemux`) | ❌ Wave 0 |
| BRIDGE-01 | single writer: concurrent Calls produce ordered lines on the child side | unit (`-race`) | `go test -race ./internal/zcode -run TestWriterOrdering` | ❌ Wave 0 |
| BRIDGE-01 | stop bypass lane jumps the queue; reverse replies bypass | unit (`-race`) | `go test -race ./internal/zcode -run TestBypass` | ❌ Wave 0 |
| BRIDGE-01 | raised buffer: 128 KiB line survives; malformed JSON line skipped+logged, stream continues | unit | `go test -race ./internal/zcode -run TestLineBuffer` | ❌ Wave 0 |
| BRIDGE-02 | scenario load/validate; in-process fake + Conn round-trip replays every scenario | golden/harness (`-race`) | `go test -race ./internal/zcode/fake -run TestGolden` | ❌ Wave 0 |
| BRIDGE-02 | standalone binary exec-plumbing: reexec helper child serves scenario; frames flow; EOF drain exits 0 | integration (`-race`) | `go test -race ./internal/zcode/fake -run TestFakeBinary` | ❌ Wave 0 |
| BRIDGE-02 | exit-0-with-zero-lines is a failure signature, never success | negative | `go test -race ./internal/zcode/fake -run TestEOFSignature` | ❌ Wave 0 |
| BRIDGE-03 | discovery positive: fixture tree yields Install with correct root/binary/entrypoint/builtin | unit | `go test -race ./internal/supervisor -run TestDiscoverFixture` | ❌ Wave 0 |
| BRIDGE-03 | discovery negative (no install): error lists every probe by name (override, xdg-entry, appimage-glob, unpacked-glob, dpkg, opt-path, rpm) | unit | `go test -race ./internal/supervisor -run TestDiscoverNoInstall` | ❌ Wave 0 |
| BRIDGE-03 | spawn: env allowlist + proxy scrub asserted on the fake binary child; `Setpgid` verified (child pgid == pid) | unit/integration | `go test -race ./internal/supervisor -run TestSpawnEnv` | ❌ Wave 0 |
| BRIDGE-03 | group-kill: after EOF/exit path, no process from the group survives (reap + `syscall.Kill(-pgid,0)` ESRCH check) | integration | `go test -race ./internal/supervisor -run TestGroupKill` | ❌ Wave 0 |
| BRIDGE-03 | crash-window restart: 3 attempts/60 s then fail loud; window reset on stable run | unit (fake child exits) | `go test -race ./internal/supervisor -run TestRestartWindow` | ❌ Wave 0 |
| BRIDGE-04 | happy turn: send → signals → resolved `completed` exactly once | golden (`-race`) | `go test -race ./internal/turn -run TestHappyTurn` | ❌ Wave 0 |
| BRIDGE-04 | cancel mid-turn: bypass stop, resolve `cancelled` exactly once, idempotent second cancel | unit (`-race`) | `go test -race ./internal/turn -run TestCancelMidTurn` | ❌ Wave 0 |
| BRIDGE-04 | double send rejected: wrapper guard + -32010 mapping | unit | `go test -race ./internal/turn -run TestDoubleSend` | ❌ Wave 0 |
| BRIDGE-04 | completion variants: terminal-event / idle-patch / quiet-window-fallback each resolve exactly once | golden (`-race`) | `go test -race ./internal/turn -run TestCompletionVariants` | ❌ Wave 0 |
| BRIDGE-03/04 | real-desktop positive path (locate → spawn → create → close, no orphans) — this machine HAS the install | manual-only (env-guarded, never CI) | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop ./internal/supervisor -run TestRealDesktop` (skipped without env) | ❌ Wave 0 (guard + test body) |

### Golden Transcript Corpus (scenarios to author — D-01/D-05 seeds)
| # | Scenario | Seeds / provenance |
|---|----------|--------------------|
| 1 | happy-turn: create (+ rtp reverse + lifecycle starting→ready) → send ack → events → terminal `turn.completed` | EXP-TURN-01/EXP-WORKSPACEKEY-01 verbatim shapes (Patterns 3/4) |
| 2 | cancel-mid-turn: send → events → `session/stop` via bypass → cancelled exactly once | D-02; PITFALL 8 matrix |
| 3 | error-32010: second send while turn running → `-32010` error reply surfaced | dialect §6 [VERIFIED: app-server-dialect.md:156] |
| 4 | unknown-host-method: server→host unknown method answered `-32601`, flow continues | probe precedent [VERIFIED: app-server-dialect.md:85; EXP-WORKSPACEKEY-01.jsonl:11] |
| 5 | runtime-preferences: reverse request `{sessionId, scope:"runtime-materialization"}` → minimal answer; and the `scope:"user-execution"` variant at send | [VERIFIED: EXP-WORKSPACEKEY-01.jsonl:8-9; EXP-TURN-01.jsonl:33-34] |
| 6 | completion-idle-patch: no terminal event; `state.updated` status leaves "running" resolves the turn | dialect §5 reliability column |
| 7 | completion-quiet-window: neither terminal nor idle; poll watcher settles via quiet window (floor+settle gates) | dialect §5; probe gates app-server-probe.mjs:570-597 |
| 8 | eof-drain: stdin close → fake drains ~100 ms → exit 0; in-flight request recorded unanswered | dialect §1 EOF facts |
| 9 | envelope-negatives: oversized (>64 KiB) line; malformed JSON line skipped | PITFALLS #7; dialect -32700 analog |
| 10 | credential-cipher-scripted: mid-turn `interaction/requestCredentialCipher` reverse request; real path records deadline expiry honestly (D-05) | [VERIFIED: EXP-TURN-01.jsonl:50 params shape] |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/zcode/... ./internal/supervisor/... ./internal/turn/...` + `mise run lint`
- **Per wave merge:** `mise run ci` (full gate incl. test-experiments)
- **Phase gate:** `mise run ci` green + manual real-desktop check recorded before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/zcode/{envelope,conn,writer,call}.go` + `internal/zcode/*_test.go` — covers BRIDGE-01 (no `*_test.go` exists anywhere yet [VERIFIED: repo-wide find this session])
- [ ] `internal/zcode/fake/{fake,scenario,golden}.go` + `testdata/scenarios/*.json` + `testdata/golden/*.golden` — covers BRIDGE-02
- [ ] `cmd/fake-app-server/main.go` — standalone mode B binary
- [ ] `internal/supervisor/{discover,spawn,supervise}.go` + `testdata/` fixture trees — covers BRIDGE-03
- [ ] `internal/turn/machine.go` + tests — covers BRIDGE-04
- [ ] Framework install: none — stdlib testing; no go.mod changes (tidy-diff guards)

## Security Domain

> Required: `workflow.security_enforcement` is true, `security_asvs_level: 1` [VERIFIED: .planning/config.json Read this session].

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | no auth surfaces in this phase (D-05 minimum answering only; full responder is Phase 5) |
| V3 Session Management | no | ACP session mapping is Phase 4 |
| V4 Access Control | no | single-user local bridge; no authorization decisions here |
| V5 Input Validation | yes | strict envelope struct decoding; unknown inbound method answered `-32601` not executed; malformed lines skipped+logged; scenario files validated at load |
| V6 Cryptography | no | no crypto in this phase (vault/GCM is Phase 6) |
| V7 Error handling/logging | yes | diagnostics to stderr via slog; error messages name probes/paths, never secrets; redaction list inherited (`apiKey`, `token`, `access_token`) for future trace logging |
| V14 Config | yes | child env allowlist (PATH, HOME, ELECTRON_RUN_AS_NODE, ZCODE_BUILTIN_PROVIDER_CONFIG_FILE); proxy vars scrubbed (D-03/PITFALL 9); secrets never in argv |

### Known Threat Patterns for this stack
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Proxy/env injection into child (dead HTTPS_PROXY breaks child fetches; hostile proxy vars redirect traffic) | Tampering | explicit env allowlist + scrub proxy vars at spawn (D-03; PITFALLS Security Mistakes table) |
| Secret leakage via logs/argv | Information Disclosure | env-not-argv for anything sensitive; slog→stderr only; no envelope params logged wholesale in this phase |
| stdout pollution corrupting a protocol stream | Tampering | stdout is protocol-exclusive; all diagnostics stderr (CONVENTIONS; ARCHITECTURE Anti-Pattern 2) |
| Orphaned Electron processes burning resources/quota | Denial of Service (resource) | `Setpgid` + group SIGKILL + `Wait()`; leftover-check in the group-kill test (PITFALL 6) |
| Unbounded memory via huge lines / stalled reader | DoS | raised-but-capped scanner max (16 MiB) + bounded queues with backpressure policy (planner discretion area) |
| Scenario-file injection (golden/scenario JSON treated as trusted) | Tampering | scenario loader validates schema; scenarios are repo-controlled, never user input |

## Sources

### Primary (HIGH confidence)
- `docs/protocol/app-server-dialect.md` — THE codec spec: §1 transport/envelope/EOF drain/bypass sets, §2 method map, §3 server→host surface, §5 terminal signals, §6 error codes (Read in full this session; every quoted value carries its V-SRC/V-EXP anchor)
- `docs/protocol/install-discovery.md` — discovery matrix, spawn line, discovery order, empty-result diagnostics contract (Read in full this session)
- `experiments/captures/EXP-WORKSPACEKEY-01.jsonl` — create-time obligations, rtp answer shape, lifecycle shapes, -32601 host answers (Read in full this session)
- `experiments/captures/EXP-TURN-01.jsonl` — send ack, state patch, turn-start signals, setModel refusal, credentialCipher params, V-SRC-only degradation record (Read lines 18-57 this session)
- Local go1.27.3 toolchain — `go doc bufio.Scanner`/`bufio.ErrTooLong`/`os/exec.Cmd`/`syscall.SysProcAttr`; GOROOT source greps for the helper-process idiom (all commands run this session)
- `.planning/CONTEXT.md` files: `03-CONTEXT.md` (D-01..D-06), `02-REVIEW.md` + `02-REVIEW-DISPOSITION.md` (WR-02 routing), `02-PATTERNS.md`, `02-VALIDATION.md`, `STATE.md`, `REQUIREMENTS.md`, `research/ARCHITECTURE.md`, `research/PITFALLS.md` (all Read this session)

### Secondary (MEDIUM confidence)
- `experiments/app-server-probe.mjs` — validated harness shapes translated into supervisor/fake designs (Read in full this session; the quiet-window gates at :570-597 are the poll-watcher blueprint)
- codex-acp crash-window precedent (via ARCHITECTURE/PITFALLS, README-derived)

### Tertiary (LOW confidence)
- `github.com/sebdah/goldie/v2` current version (v2.8.0 via `go list -m -versions` — verified on the proxy, but the library itself was not exercised)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib APIs verified against the pinned toolchain docs; zero external deps
- Architecture: HIGH — Patterns 2/4/6 come from ARCHITECTURE.md code shapes; all wire values quoted from Phase-1-verified docs and captures Read this session
- Pitfalls: HIGH — every pitfall carries a Phase-1 file:line anchor or toolchain-doc verification
- Turn completion signals on the REAL server: MEDIUM — terminal-event wrapping is V-SRC-only (Phase-1 honesty record); the fake defines shapes and the multi-signal union degrades gracefully

**Research date:** 2026-11-19
**Valid until:** 2026-12-19 (stable: wire spec is versioned to desktop 3.18.2; re-check on any desktop auto-update)

## RESEARCH COMPLETE
