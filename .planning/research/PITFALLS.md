# Pitfalls Research

**Domain:** Go ACP v1 server wrapping ZCode Desktop's proprietary app-server stdio protocol
**Researched:** 2026-11-19
**Confidence:** HIGH overall — every ZCode-side claim is either experimentally verified (FINDINGS.md, real 3.18.2 binary on this machine) or confirmed in the public source tree (`ZCode source tree (github.com/zai-org/ZCode; local clone)`); ACP claims come from the official v1 spec; Go claims cite Go source/docs/issue tracker. The few unverifiable items are explicitly labeled.

Legend: **[V-EXP]** verified by experiment on the real machine (FINDINGS.md) · **[V-SRC]** verified by reading the source tree · **[V-SPEC]** verified against the ACP v1 spec · **[V-GO]** verified against Go source/docs/issue tracker · **[M]** medium confidence · **[L]** low confidence / unverified.

---

## Critical Pitfalls

### Pitfall 1: Treating `provider/updateAccountConfig` as idempotent-and-trusted ("ack ≠ acceptance")

**What goes wrong:**
Two combined traps, both experimentally verified:
1. The app-server transport is **plain JSONL, not JSON-RPC 2.0**. A request carrying the `jsonrpc: "2.0"` key is rejected outright. [V-EXP]
2. The snapshot you push via `provider/updateAccountConfig` is **silently discarded** when the revision string does not match `zcode-builtin:<rev>:<sha256(resolve(activeBuiltinFilePath))>` — and the method still answers `status: "received"` (or `"unchanged"`). [V-EXP + V-SRC: `packages/provider/src/registry-service.ts:205-212` — on `basedOnZCodeBuiltinRevision !== config.zcodeBuiltinRevision` the registry keeps serving the *previous* snapshot and just `continue`s the refresh loop; `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/account-provider-config.ts:28-33` — the ack reflects envelope parsing only]

**Why it happens:**
The registry service is designed for the Desktop host, which already knows the exact builtin file path and revision; consistency is restored by a later `provider/refreshRegistry` + `state.updated`. The protocol therefore has no error channel for "your snapshot was stale" — a new wrapper author assumes request/response success means state mutation. And JSON-RPC 2.0 is the default mental model for anyone who has touched LSP/MCP/ACP, so the `jsonrpc` key leaks in from muscle memory.

**How to avoid:**
- Separate serializers per hop: a strict JSON-RPC 2.0 codec for the ACP side (client ↔ wrapper), a plain `{id, method, params}` / `{id, result}` JSONL codec for the wrapper ↔ app-server hop. Never share a message type between the two.
- **Never trust the ack.** After `updateAccountConfig`, confirm acceptance by reading state back: call `provider/refreshRegistry` then await the `state.updated` notification and check the registry actually contains `account:zai-individual-coding-plan` with `availability: "available"`. Build a `awaitRegistryContains(providerId, timeout)` helper in Phase 2 and make every auth-path test use it.
- Construct the revision string at runtime: read the resolved active builtin file path (e.g. `~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/endpoint-*/zcode-builtin.json`), compute `zcode-builtin:<rev>:<sha256-of-resolved-path>` in Go (`filepath`-resolve, then `sha256sum` of the *resolved* path string — not the file contents). It is machine- and version-specific; hard-coding the string from one machine (`zcode-builtin:44:e9c0…`) breaks everywhere else.
- The snapshot schema is zod `.strict()` [V-SRC: `packages/shared/src/zcode-protocol/index.ts` — pervasive `.strict()`]: extra fields (`group`, `api`, `accountType`, `mode`, `config`) cause a parse *error*, not a silent strip. Keep the payload to exactly: `revision`, `basedOnZCodeBuiltinRevision`, `providers{builtinModelIds, access}`, `states{availability, entitled, current}`.

**Warning signs:**
- `session/create` then first turn fails with `Provider Registry 中不存在 Provider` / provider-not-found; or `account:*` models never appear in the model list.
- Requests fail only on machines other than the one where the revision string was generated.
- The shim v2 logs show `status: "received"` while the provider log shows no `account:*` model request.

**Phase to address:** Phase 2 (protocol core) for the codec split; Phase 3 (auth) for the registry-verification helper. Encode "verify registry, not ack" as a test in the fake app-server.

---

### Pitfall 2: Binding a thinking model through `session/create` (`options.reasoningLevel` is dropped)

**What goes wrong:**
`session/create` with a model selection for a thinking model (GLM-5.3, etc.) loses `options.reasoningLevel`, and the turn then fails with `Reasoning level is required for <provider>/<model>`. [V-EXP + V-SRC: `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/model-mapper.ts:4-6` — `formatProtocolModelSelection` serializes to the bare string `providerId/modelId`; `apps/zcode-cli/packages/bootstrap/src/app/provider-registry-selection.ts:133-142` — `reasoning-level-missing` maps to `InvalidModelRequest`]

**Why it happens:**
The protocol model-selection field is a string (`provider/model`); reasoning level simply has no channel in `session/create`. A second, sibling error exists: `reasoning effort "…" is not supported by …` for wrong levels — so level validation is strict in both directions.

**How to avoid:**
- Two-step binding as the only model-binding path: `session/create` without a model (or with the workspace default), then `session/setModel` with `{providerId, modelId, options:{reasoningLevel}}`; where available use `session/setThoughtLevel` for level-only changes [V-SRC: both methods exist in the server dispatch table, `server.ts:755-757`].
- Map ACP client model selection (e.g. Orca's model picker / `session/options`) to a `(providerId, modelId, reasoningLevel)` triple, with a documented default level (e.g. `medium`) for thinking models and `undefined` for non-thinking ones. This mapping is an open question flagged in PROJECT.md — resolve it in Phase 1 by reading Orca's `agent-session-attachment-store-registry-*.js`.

**Warning signs:**
- Every turn on a thinking model fails with "Reasoning level is required" while non-thinking models work.
- Works with `zai-api/GLM-5.3` (api-key path) but fails on `account:*` models, or vice versa — the failure follows the model class, not the auth path.

**Phase to address:** Phase 2 (session lifecycle). Add a fake-app-server test asserting the `setModel` payload carries `options.reasoningLevel`.

---

### Pitfall 3: Answering `interaction/requestProviderRuntimeHeaders` with `headersApplied:false`, or trying to authenticate with the OAuth access token instead of a minted PAT

**What goes wrong:**
Two related fatal failures on the OAuth individual-coding-plan path:
1. For account providers, a `headersApplied:false` response to the runtime-headers request is **not accepted**: the turn fails with `Provider runtime headers were not applied before model request attempt.` (error code -32031). [V-EXP + V-SRC: `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/provider-runtime-headers.ts:84-100`]
2. The V4 handshake (`POST /api/paas/c1f3a7e2/v2/client`) fails with the raw OAuth access token (`4001 HANDSHAKE_INVALID_REQUEST`, JWT claim `api_key: null`) or api-key (`4011 HANDSHAKE_AUTH_FAILED`). Only a **Project Access Token** (minted via getCustomerInfo → api_keys → access_tokens on api.z.ai) carries a usable `api_key` claim; the app-server binary itself does the per-request V4 signing after receiving `{headersApplied:true, requestAuth:{apiKey, apiKeyId}}`. [V-EXP]

**Why it happens:**
The host is the credential authority in host-managed mode: the app-server outsources "give me the auth material" to the host and assumes a truthful `headersApplied` answer. A wrapper that replies `false` (to be "honest" about not knowing) or that forwards the OAuth JWT (because it decrypted it from the keyring and it *looks* like a bearer token) produces a guaranteed turn failure — the protocol offers no downgrade path.

**How to avoid:**
- Implement the PAT minting pipeline exactly as verified (raw OAuth token **without** `Bearer` prefix on `getCustomerInfo`; `Bearer` on the api_keys/access_tokens calls; choose org/project by `默认机构`/`默认项目` falling back to first; `usageScene: 1`).
- Respond to `interaction/requestProviderRuntimeHeaders` with `{headersApplied:true, requestAuth:{apiKey:<PAT>, apiKeyId:<keyId>}}` or, on minting failure, with an explicit error response — never `headersApplied:false`.
- Note the request can recur: the params include `rejectedProjectTokenFingerprint` [V-SRC: `provider-runtime-headers.ts:48-50,92-95`] — when the server rejects a previously supplied token it re-asks. Cache the PAT but re-mint when the request carries a rejected fingerprint; do not build a one-shot assumption.
- Keep the wrapper out of the signing business entirely: no X-Client-Sig/PoW code in the wrapper (PROJECT.md explicitly excludes attestation forgery; the official binary signs after the PAT handshake).

**Warning signs:**
- Handshake responses 4001/4011 in the z.ai provider log while credentials decrypt fine.
- Turn fails instantly with "runtime headers were not applied" on every attempt for `account:*` but api-key turns work.
- Token works for a while then turns fail mid-session until re-mint (refresh signal ignored).

**Phase to address:** Phase 3 (OAuth path). Requires the keyring reader (see Pitfall 11) as a prerequisite — order phases accordingly.

---

### Pitfall 4: Mapping `session/prompt` completion incorrectly — `session/send` returns before the turn ends

**What goes wrong:**
ACP requires that `session/update` notifications stream during the turn and that the `session/prompt` **request** is answered only when the turn is over, with a `stopReason` (`end_turn`, `cancelled`, …). But app-server `session/send` is fire-and-forget: it spawns `runPromptTurnInBackground` with `void` and returns immediately; progress arrives as `state.updated` / session events, not as the RPC result. A naive bridge answers `session/prompt` as soon as `session/send` returns — clients see an empty, instantly-completed turn, then orphaned updates. [V-SRC: `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/server-operations.ts:2025-2100` (`void runPromptTurnInBackground(...)`); ACP semantics V-SPEC: prompt-turn + cancellation pages]

**Why it happens:**
The app-server protocol is a UI protocol (async event stream, poll-or-subscribe), while ACP wraps a turn into a single request/response. The async-to-sync lift is the actual adapter work and is easy to skip.

**How to avoid:**
- Per session, keep a turn state machine: `session/send` accepted → subscribe/collect events → stream ACP `session/update` (message chunks, thought chunks, tool_call lifecycle) → detect terminal event → reply to `session/prompt` with the correct `stopReason`.
- Respect the built-in concurrency guard: a second `session/send` on the same session while a turn runs is rejected with -32010 "A prompt is already running for this session" [V-SRC: same file, lines 2036-2045]. Either serialize prompts per session in the wrapper or surface a clean ACP error; never queue silently (the user believes the prompt is pending).
- `session/send` also validates `expectedRevision` / `expectedProviderRevision` [V-SRC: line 2034-2035] — stale revision state after a dropped snapshot (Pitfall 1) surfaces here; pass no expected-revision fields unless you track them.
- Terminal-event detection needs Phase 1 research: enumerate exactly which `state.updated` payloads end a turn (turn.completed in the provider log is the ground truth per FINDINGS.md §3.6) before writing the state machine.

**Warning signs:**
- Orca card shows the answer after the turn "finished" with no streaming; or chunks arrive after the prompt response (clients may drop them).
- Second prompt in the same session errors with -32010 in wrapper logs while the client UI shows nothing.

**Phase to address:** Phase 2 (chat loop) — this is the heart of the adapter. Verify with the fake app-server emitting scripted event sequences.

---

### Pitfall 5: Protocol drift between desktop releases — hard-coding the method set or the revision

**What goes wrong:**
The app-server protocol changes between desktop versions. Direct evidence: the public 3.17.x source tree contains **no** `interaction/requestCredentialCipher`, but the installed 3.18.2 binary sends it [V-EXP FINDINGS §2 + V-SRC absence in `packages/shared/src/zcode-protocol/index.ts`]. Builtin revision numbers drift too: 3.12.3→28, 3.15.x→30/31, 3.16.3/3.17.x→43, 3.18.2→44 [V-EXP]. A wrapper compiled against one snapshot of the protocol breaks silently on the next desktop update — often by *crashing on an unknown server→host method* or by pushing snapshots that are always discarded (Pitfall 1).

**Why it happens:**
ZCode Desktop is a fast-moving proprietary target; there is no versioned protocol contract to code against, and the only stable-ish artifact is the source tree of one release.

**How to avoid:**
- **Fail fast on startup mismatch:** at spawn, run a capability handshake — send `provider/refreshRegistry`, wait for the first `state.updated`, and validate the observed revision matches the builtin file you resolved; if the exchange contains unexpected shapes or times out, exit with a clear error naming the detected desktop version and the supported range. Do not limp along.
- **Tolerate unknown server→host methods:** unknown `interaction/*` or notification methods must be answered with a JSONL error response (or ignored if one-way) — never an unhandled exception that kills the wrapper mid-turn.
- Pin a supported-version range in config; print it in `--version` and startup logs; gate known-fragile features (e.g. credential-cipher flow) on detected revision.
- Record fixtures per desktop version (see Testing pitfalls) so drift shows up as CI diffs, not user bug reports.
- Phase 1 must produce a protocol doc with a "verified against: 3.18.2 / source 3.17.x" header so reviewers know the drift baseline.

**Warning signs:**
- Wrapper logs show unknown-method errors after a desktop auto-update.
- The revision string computed from the builtin file changes shape (new fields in `state.updated` payloads).
- FINDINGS-style tricks (exact revision `44`) stop working on other machines.

**Phase to address:** Phase 1 (document drift baseline + unknown-method policy), Phase 2 (fail-fast startup handshake), Phase 5 (versioned fixtures).

---

### Pitfall 6: Zombie/orphan app-server processes after wrapper crash or client disconnect

**What goes wrong:**
`cmd.Process.Kill()` in Go kills only the direct child. The app-server runs as `ELECTRON_RUN_AS_NODE=1 zcode.cjs app-server --stdio` and Electron/Node spawns helper processes; on wrapper exit these are reparented to init and keep running (holding the workspace, burning quota via open sessions). Additionally: killed-but-unwaited children linger as zombies unless `cmd.Wait()` reaps them, and Linux `Pdeathsig` is delivered when the parent *thread* dies, not the parent process — Go's runtime can migrate/exit threads, so it is not a reliable safety net. [V-GO: golang/go#27505; felixge process-group article; SO "Why won't Go kill a child process correctly"]

**Why it happens:**
Standard `exec.Cmd` usage covers the 90% case; the orphan case only manifests on crash/timeout paths that are rarely exercised in dev.

**How to avoid:**
- `SysProcAttr{Setpgid: true}` on the app-server child; on shutdown/EOF/crash path kill the whole group: `syscall.Kill(-pgid, syscall.SIGKILL)` (Windows: Job Objects — code stays cross-platform behind a build-tagged shim, v1 only validates Ubuntu).
- Install SIGTERM/SIGINT/SIGHUP handlers that run the same shutdown path; `defer`-guard the kill in the goroutine owning the child.
- Do **not** assume the child exits when stdin closes: ACP clients terminate agents by closing stdin [V-SPEC], but whether the Electron-based app-server exits on stdin EOF is unverified [M]. Explicitly kill the process group after a short grace period on EOF.
- Always `cmd.Wait()` (in the same goroutine that pumps stdout) so the child is reaped.

**Warning signs:**
- `pgrep -f "app-server --stdio"` shows stale processes after wrapper exits or panics.
- Zombie entries in `ps` with the wrapper as parent.
- Quota usage appears with no active client.

**Phase to address:** Phase 2 (process supervisor module). Add a CI test that starts/stops the fake wrapper against a real `sleep`-style child and asserts no leftovers.

---

### Pitfall 7: Deadlocking or hanging the single stdio pipe — bidirectional RPC, backpressure, and stderr

**What goes wrong:**
Three deadlock/hang variants around the two pipes:
1. **Same-goroutine read+write:** issuing an app-server request and *then* reading for its response on one goroutine deadlocks when the response arrives interleaved with unrelated notifications; conversely, writing to the child's stdin blocks if the child isn't reading while you also pump its stdout.
2. **Undrained stderr:** the Electron runtime prints GTK/Wayland warnings to stderr; if the wrapper never drains the child's stderr, the 64 KB pipe buffer fills and the child blocks on its next log write — appearing as a mysterious mid-turn hang.
3. **Unbounded client fan-out:** spawning a goroutine per outgoing `session/update` to "go faster" reorders chunks (ACP clients render receive-order) and can interleave writes on stdout, producing corrupt NDJSON. [V-SPEC: updates apply in receive order; NDJSON must not contain embedded newlines; stdout purity]

**Why it happens:**
One pipe, two roles (RPC responses and event stream), plus the classic Go trap that `cmd.Wait()` must not be called before stdout/stderr pumps finish draining (or `Wait` deadlocks on the pipe buffers).

**How to avoid:**
- Fixed goroutine topology: one reader goroutine per child pipe (stdout, stderr), one writer goroutine per sink (child stdin, wrapper stdout), connected by channels; a single demultiplexing loop dispatches by method/id. Never `Wait()` before the pumps return.
- Ordered per-session event queue feeding the single stdout writer; ordering lives in the queue, not in goroutine scheduling.
- Reader uses `bufio.Reader.ReadBytes('\n')` or `json.Decoder` — **not** `bufio.Scanner` with defaults: the scanner fails with `token too long` on lines >64 KB (`bufio.MaxScanTokenSize`), and tool-call payloads easily exceed that. [V-GO: SO/Go forum on ErrTooLong]
- Backpressure policy: writing to wrapper stdout blocks when the ACP client stops reading — that is *correct* (ordered, lossless); do not swap it for unbounded buffering that grows RAM on a stuck client. Cap an internal per-session event buffer and, if exceeded, log to stderr and apply backpressure.
- Child stderr: drain and forward to wrapper stderr (or a capped log file). Never merge child stderr into wrapper stdout.

**Warning signs:**
- Hangs that clear when stderr is redirected to a file (classic undrained-stderr signature).
- `bufio.Scanner: token too long` in logs.
- Clients report garbled/interleaved JSON or out-of-order chunks.

**Phase to address:** Phase 2 (transport module + goroutine topology test under `-race`).

---

### Pitfall 8: Mapping ACP `session/cancel` races onto app-server `session/stop`

**What goes wrong:**
ACP cancellation has sharp edges the wrapper must honor precisely:
- `session/cancel` is a notification with **no response**; but the original `session/prompt` request **MUST** still be answered — either a normal response with `stopReason: "cancelled"` or error -32800. Leaving the request unanswered hangs the client UI. [V-SPEC]
- On cancel, the *client* answers pending `session/request_permission` prompts with `cancelled`; the wrapper must accept that outcome for a permission request it already forwarded, propagate stop to the turn, and still return `cancelled` (not an error) for the prompt. The spec explicitly warns agents must map aborted operations to the cancelled stop reason so clients don't render cancellation as a failure. [V-SPEC]
- Races: cancel arriving before the prompt is forwarded, after the turn already ended, or while a PAT-mint / runtime-headers round-trip is in flight. The app-server itself holds a **credentials-resolution queue** for up to 180 s on unanswered `interaction/requestProviderRuntimeHeaders` and only sends a `interaction/providerRuntimeHeadersCancelled` notification on abort/timeout of that port [V-SRC: `provider-runtime-headers.ts:17, 60-72`; 180 000 ms constant].

**Why it happens:**
Cancellation state lives in three places (client turn, wrapper turn state machine, app-server session) and transitions are asynchronous in all three.

**How to avoid:**
- One turn state machine per session with an explicit `cancelling` state; idempotent cancel (cancel-after-complete is a no-op, not an error).
- Always answer `session/prompt` exactly once — the response write must be owned by the state machine, not by the send path.
- Forward cancel as `session/stop` [V-SRC: dispatch entry exists, `server.ts:746`]; also cancel in-flight host-side work: answer any pending `interaction/*` with the protocol's cancel/error outcome and send `interaction/providerRuntimeHeadersCancelled`-style notifications where the port expects them (mirror what `provider-runtime-headers.ts:66-72` does).
- Never let an unanswered `interaction/*` hold the turn: the 180 s queue-hold blocks *subsequent* turns' credential resolution too; answer every interaction request promptly, even if the answer is an error.
- Deduplicate `interaction/*` requests: the broker re-announces at a 1 s interval [V-SRC: `interaction-broker.ts:60`, `INTERACTION_REQUEST_REANNOUNCE_INTERVAL_MS = 1_000`] [M on exact wire behavior] — key pending interactions by `requestId`.

**Warning signs:**
- Orca card spins forever after pressing stop (unanswered prompt request).
- Cancellation surfaces as a red error in the client (stop reason not mapped).
- A turn after a cancelled one hangs for ~3 minutes, then times out (queue held by an unanswered interaction).

**Phase to address:** Phase 2 (cancel semantics tests: cancel-before-prompt, cancel-mid-stream, cancel-during-permission, double-cancel), Phase 3 (interaction answering).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hard-coding the builtin revision string (`zcode-builtin:44:e9c0…`) from one machine | OAuth path works day 1 | Breaks on every other machine/version; debugging cost high (silent drop) | Never — compute at runtime (Pitfall 1) |
| One shared message struct for ACP and app-server JSONL | Less code | `jsonrpc` key leaks into app-server requests (rejected); drift risk (Pitfall 5) | Never |
| Trusting method acks (`received`) instead of reading registry state | Fast OAuth path | Silent stale registry; user-facing model-list bugs (Pitfall 1) | Never |
| Polling `session/messages` with sleep loops instead of event subscription | Simpler first draft | Latency, ordering races, missed terminal events; rewrite of the turn state machine | Only as a throwaway probe script, not in the product |
| Matching error *strings* for control flow | Quick triage | Error text is not a stable contract across versions (some messages are even locale-mixed in source); match error codes (-32010/-32031/-32800/…) | Never |
| Single mutex over both protocol id spaces (ACP ids and app-server ids) | Simple correlation map | Deadlock under notification storms; safer: separate id allocators + separate pending maps per hop | Never |
| Ignoring unknown `interaction/*` methods silently | Forward-compat today | Hides drift; a required credential flow silently never completes — better: log to stderr *and* answer with a protocol error | Log-always, answer-with-error is the floor |
| Emitting wrapper diagnostics to stdout "temporarily" | Easy debugging | Corrupts ACP NDJSON stream; breaks Orca card with cryptic parse errors | Never — stdout is protocol-only; stderr is free |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| app-server spawn | Running `zcode` (the desktop launcher) or the CLI binary instead of `ELECTRON_RUN_AS_NODE=1 <path>/resources/glm/zcode.cjs app-server --stdio`; or forgetting `ELECTRON_RUN_AS_NODE=1` (launches the GUI) | Resolve the installed desktop's `zcode.cjs` (AppImage/deb/unpacked differ — open PROJECT.md question), set `ELECTRON_RUN_AS_NODE=1`, pass `app-server --stdio` |
| app-server transport | Sending JSON-RPC 2.0 (with `jsonrpc` key) or expecting `Content-Length` framing (LSP-style) | NDJSON `{id, method, params}` lines; no `jsonrpc` key; `\n`-delimited, UTF-8 only |
| Orca (chat UI) | Registering a custom ACP agent via `customAgentCommand` (only for operations, not ACP); grepping packed `app.asar/out/main/index.js` for ACP strings and concluding "Orca has no ACP" (FINDINGS §4 — the client lives in `app.asar.unpacked/out/main/chunks/`) | Follow the README recipe path: wrapper runs as the chosen pre-approved agent (**omp** — resolved; wiring via `agentCmdOverrides`, see REFERENCE-ADAPTERS.md); research in `app.asar.unpacked` chunks |
| Orca agent detection | Naming the binary differently ("zcode-acp" is not in Orca's registry) or printing multi-line/verbose `--version` output | Install as a PATH name Orca hard-codes (e.g. `omp`-compatible wrapper honoring `omp acp`-style invocation), `--version` prints exactly one parseable semver line ≥ the minimum |
| Orca startup probes | Exiting on unknown flags (Orca passes `--session-dir`, sets `PI_NO_PTY=1`, possibly more) | Accept-and-honor (or accept-and-ignore-with-stderr-note) unknown flags and env; never require a TTY |
| Secret Service | `secret-tool lookup` with a different path resolution than the desktop used (the key is `sha256(realpath ~/.zcode/v2/credentials.json)`) | Resolve with `filepath.EvalSymlinks`+`Abs` exactly like `realpath` before hashing; handle lookup-miss with an actionable error |
| api.z.ai PAT endpoints | Sending the OAuth token *with* `Bearer` on `getCustomerInfo` (it must be raw), or with *no* `Bearer` on api_keys/access_tokens (it must have it) | Copy the exact header pattern verified in FINDINGS §3.3; keep the two styles in one typed client |
| Go http client | Assuming Node's `ipv4first` fix applies (Go has Happy Eyeballs by default) but inheriting `HTTPS_PROXY` uncritically | See Pitfall 9; set `NO_PROXY` for api.z.ai in the child env if a proxy is configured |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Goroutine-per-outgoing-message (ordering loss, stdout interleaving) | Garbled NDJSON, reordered chunks in client | Single writer goroutine + ordered per-session queues (Pitfall 7) | Immediately under concurrent tool-calls; intermittent = worst kind |
| Unbounded event buffering toward a slow client | Wrapper RAM grows during long tool-heavy turns | Bounded queue + backpressure on stdout write; cap + stderr log | Long sessions with big file/tool outputs |
| `bufio.Scanner` on child stdout | `token too long`, stream stops mid-turn | `bufio.Reader`/`json.Decoder` or `Scanner.Buffer` with a raised max (e.g. 10 MB) | First tool result >64 KB |
| Synchronous PAT mint inside the interaction answer path with no timeout | Turn hangs up to the server's 180 s window; queue blocks later turns | Mint with its own (shorter) timeout + retry; answer the interaction even on failure | Weak network / z.ai incidents |
| Spawning a new app-server per prompt instead of one persistent process | Multi-second startup per turn; state (session, registry snapshot) lost | One long-lived child per wrapper; sessions multiplexed over it; restart = full re-init (registry + auth) | Every turn, painfully |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Logging protocol traces (requests/responses) unredacted | PAT, OAuth JWT, api-key land in wrapper logs / CI output | Redaction middleware on the trace logger (`apiKey`, `token`, `access_token`, `authorization` fields); document it in README |
| Recording app-server JSONL fixtures from live sessions without scrubbing | Credentials committed to the repo | Fixture scrubber in CI (grep for JWT shapes `eyJ…`, `enc:v2:os:…`, 32-hex PAT ids); fail CI on match |
| Widening credentials.json/keyring access beyond what decryption needs | Keyring entry readable by other processes/user sessions | Read-only access; never persist decrypted material to disk; in-memory only; chmod 600 on anything transient |
| Patching `zcode.cjs` or the desktop bundle to make integration easier | Explicitly out of scope (PROJECT.md); ToS/quota-attestation territory; breaks on update | Wrapper is read-only toward the desktop installation; resist any bundle edit |
| Trusting `customAgentCommand`-style escape hatches or env passthrough of user-controlled vars into the child | Proxy/credential env can redirect api.z.ai traffic | Explicit child env allowlist: copy PATH, HOME, ELECTRON_RUN_AS_NODE; scrub HTTP(S)_PROXY or set NO_PROXY for api.z.ai |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Cancellation rendered as an error (stop reason not mapped) | User thinks the app broke when pressing stop | Map aborts to `stopReason:"cancelled"` — the ACP spec calls this out explicitly [V-SPEC] |
| Silence during OAuth bootstrap (decrypt → PAT mint = several HTTP round trips) | Card looks dead for seconds; user retries and double-sends | Emit `session/update` status/notice or an early `tool_call`-style progress while minting |
| Stale model list after a silently dropped registry snapshot | User picks `account:*` model, every turn fails | Registry verification after `updateAccountConfig` (Pitfall 1) + rebuild model list from verified state |
| Permission requests that never surface (wrapper swallowed `interaction/requestPermission`) | Turn hangs; user cannot approve tools | Forward every permission request to ACP `session/request_permission`; answer-with-deny on wrapper-side timeout rather than hanging |
| Version-mismatch wrapper failing with a stack trace | User cannot act on it | Fail fast with a human message: detected desktop version, supported range, where to look |

## "Looks Done But Isn't" Checklist

- [ ] **Chat loop:** `session/prompt` answered exactly once with a real `stopReason` — verify no path answers early on `session/send` ack (Pitfall 4)
- [ ] **Streaming:** thought chunks (`agent_thought_chunk`) wired, not just message chunks — thinking models produce them and Orca renders them separately
- [ ] **OAuth path end-to-end:** registry verified (not acked), PAT minted, `requestAuth` answered, turn actually completes on `account:*` (FINDINGS §3.6 parity for the ACP surface)
- [ ] **api-key path regression:** after OAuth work, `zai-api/*` turns still pass (FINDINGS `APIKEY_PATH_OK` discipline)
- [ ] **Cancellation:** cancel mid-stream, cancel during permission, cancel during PAT mint, double-cancel — prompt always answered once, no orphans
- [ ] **Shutdown:** client closes stdin → child process group actually gone (check `pgrep` in a test, Pitfall 6)
- [ ] **Headless:** wrapper works with no D-Bus/Secret Service and produces an actionable error, not a hang (matters for CI and ssh)
- [ ] **Version gate:** wrapper refuses (clearly) a desktop whose revision handshake fails, instead of half-working (Pitfall 5)
- [ ] **Orca recipe:** fresh machine, `omp`-named binary ≥17.0.5 in PATH, card starts, permissions surface, stop works — the recipe is only done when reproduced on a clean checkout
- [ ] **stdout purity:** no wrapper log line ever reaches stdout under panic/recovery paths (panic messages default to stderr, but custom handlers often print to stdout)

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Snapshot silently dropped (1) | LOW | Recompute revision from resolved builtin path, re-push, verify registry via `refreshRegistry`+`state.updated` |
| reasoningLevel lost (2) | LOW | Re-bind with `session/setModel` after create; make it the only binding path |
| headersApplied:false / wrong token (3) | MEDIUM | Mint PAT (needs working keyring read + network), re-answer interaction on next request; user-visible turn is lost |
| Premature prompt response (4) | MEDIUM | State-machine fix + scripted fake-server tests; clients may need reconnect to reset UI state |
| Unknown-method crash after desktop update (5) | MEDIUM | Add unknown-method tolerance + version gate; release wrapper update pinning new supported range |
| Orphaned app-server processes (6) | LOW | `pkill -g <pgid>`; add supervisor + tests to prevent recurrence |
| Pipe deadlock (7) | LOW | Fix goroutine topology; add `-race` CI test with interleaved traffic |
| Cancel races (8) | LOW→MEDIUM | Idempotent state machine; if the queue is already held (180 s), the turn is lost — answer next interactions promptly |

## Pitfall-to-Phase Mapping

Phases per PROJECT.md shape: Phase 1 Research & protocol docs → Phase 2 ACP core (transport, chat loop, process supervision) → Phase 3 Auth (api-key + OAuth/PAT + keyring) → Phase 4 Orca integration recipe & E2E → Phase 5 CI/VM hardening & docs.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1 JSONL codec / ack≠acceptance | Phase 2 (codec), Phase 3 (registry verify) | Fake-server test: stale revision discarded → wrapper detects via registry read and recovers |
| 2 reasoningLevel drop | Phase 2 | Fake-server asserts `session/setModel` carries `options.reasoningLevel`; E2E on thinking model |
| 3 headersApplied / PAT | Phase 3 | Fake-server interaction flow; E2E OAuth turn (FINDINGS §3.6 parity); PAT-mint timeout path answers interaction with error |
| 4 Turn completion mapping | Phase 2 | Scripted event streams incl. delayed terminal event; prompt answered exactly once |
| 5 Protocol drift / fail-fast | Phase 1 (baseline doc), Phase 2 (handshake), Phase 5 (versioned fixtures) | Unknown-method tolerance test; startup fails cleanly against fake old-revision server |
| 6 Zombie processes | Phase 2 | CI test asserts no child/group survivors after kill/EOF/panic paths |
| 7 Pipe deadlocks / scanner / stderr | Phase 2 | `-race` test with >64 KB lines and chatty stderr; ordered-chunk assertion |
| 8 Cancel races | Phase 2 (state machine), Phase 3 (interaction answering) | Cancel matrix tests (before/mid/during-permission/double) |
| 9 Go proxy env (below) | Phase 3 | Unit test with fake proxy env; NO_PROXY documented in README |
| 10 Keyring/headless (below) | Phase 3 | VM test without gnome-keyring → actionable error, no hang |
| 11 Orca recipe (below) | Phase 4 | Clean-VM reproduction from README alone |

## Moderate Pitfalls (beyond the critical eight)

### Pitfall 9: Go translation of the Node network traps — proxy env is read once, and IPv6 is a different story

**What goes wrong:** FINDINGS §3.5 (Node: `fetch` picks IPv6 first and hangs; dead `HTTPS_PROXY` breaks calls while curl works) translates to Go as follows:
- **Dead proxy:** `http.DefaultTransport` uses `ProxyFromEnvironment`, which caches the parsed env in a package-level `sync.Once` [V-GO: `net/http/transport.go`, `envProxyOnce` in go1.22 source; env-var rules + localhost exemption per pkg.go.dev]. A dead `HTTPS_PROXY` in the wrapper's environment therefore breaks all api.z.ai calls for the *whole process lifetime*; unsetting env later does nothing. Also `NO_PROXY` dialect differs from curl's (Go supports CIDR; curl uses suffix matching) — a rule that "works with curl" may not exempt api.z.ai in Go [M on exact dialect edge cases].
- **IPv6:** Go's default dialer implements Happy Eyeballs (RFC 8305) with a 300 ms fallback, so the Node ipv6first hang does **not** apply to default transports [M: standard-library behavior, widely documented]. It *can* be reintroduced by a custom `Dialer` with `FallbackDelay < 0` or exotic `Control` functions — review any custom transport.

**Prevention:** Explicit transport config at startup: log the effective proxy once; if `HTTPS_PROXY` is set, validate connectivity to api.z.ai through it at first use and fail with a targeted message ("proxy X unreachable; unset HTTPS_PROXY or add NO_PROXY"); construct the app-server child env deliberately (scrub proxy vars or set NO_PROXY). Keep default dialer semantics (no custom ipv6-only dialer).

**Phase to address:** Phase 3. Verify in the VM E2E with a proxy env set.

### Pitfall 10: Keyring and credential-store availability assumptions

**What goes wrong:** The OAuth path starts with decrypting `~/.zcode/v2/credentials.json` (`enc:v2:os:<kid>:<iv>.<tag>.<ct>`, key in Secret Service under `ZCode Credential Encryption` with username = `sha256(realpath credentials.json)`). In headless contexts (CI, ssh, distro without gnome-keyring) there is no session bus or the login keyring is locked → lookup returns nothing or blocks on a prompt that nobody can answer. Separately, the desktop can rotate/rewrite `credentials.json` (new kid) while the wrapper holds a cached key or token; FINDINGS also proves the OAuth access token is a JWT with expiry. The `rejectedProjectTokenFingerprint` re-request path [V-SRC] is the server's mid-session "your token was rejected" signal.

**Prevention:** Path-hash computed with exactly `realpath` semantics; keyring miss → immediate actionable error (name the exact secret-tool query to reproduce), never a retry loop or hang; never block startup on keyring availability when only the api-key path is configured; handle token-refresh as *re-run the PAT mint on demand* (stateless cache keyed by kid/fingerprint); guard against reading `credentials.json` mid-rewrite (parse failure → one clean retry with backoff).

**Phase to address:** Phase 3; VM CI test for the no-keyring environment (PROJECT.md: v1 is Ubuntu GNOME — the error path still must be clean).

### Pitfall 11: Orca integration recipe — running as a pre-approved agent

**What goes wrong:** Orca hard-codes its ACP agents (`omp` ≥17.0.5 with `PI_NO_PTY=1`, `opencode` ≥2.0.0, `opencode2` ≥2.0.14), detects by **binary name in PATH + version**, and has no UI to add custom ACP agents. Failure modes: wrapper named something Orca doesn't know → agent never offered (mitigated: the `agentCmdOverrides` override removes the PATH-name requirement); `--version` output unparseable or below minimum → silently rejected; wrapper doesn't answer the startup `initialize` in time → card dies without a visible error; flag mismatches (`--session-dir` expected, wrapper is strict) → exit before ACP starts. Also the identity trap FINDINGS §6.8: confusing `pi` (1.5.2, no ACP) with `omp` (the fork that has ACP). [V-EXP FINDINGS §4]

**Prevention:** Persona decided (2026-11-19): **omp** — the Hb dialect is string-contract driven, trivially portable to Go (rationale and rejected alternatives in REFERENCE-ADAPTERS.md); `--version` prints exactly `x.y.z` and exits 0; accept unknown flags/env liberally, honor `--session-dir` if trivial; answer `initialize` as fast as possible (defer everything non-essential — the PAT mint must *not* happen before `initialize` returns); all diagnostics to stderr with an `zcode-acp:` prefix so users can find them in Orca's captured logs. README recipe must be reproducible on a clean VM — that is the acceptance test.

**Phase to address:** Phase 4. Load-bearing method set ("which calls Orca makes at card start") is a Phase 1 research deliverable read from `agent-session-attachment-store-registry-*.js` in `app.asar.unpacked` — **not** the packed asar (FINDINGS §4, pitfall h).

### Pitfall 12: ACP version negotiation and capability honesty

**What goes wrong:** ACP v1 negotiates a single integer `protocolVersion`: client sends its latest; the agent echoes it if supported, otherwise replies with its *newest* supported version; the client may then close the connection (SHOULD, with user notice). Lying (echoing v1 while implementing v2 shapes, or inventing a version) yields undefined client behavior. Equally: advertising capabilities the wrapper can't honor (e.g. `terminal`, `fs.readTextFile`) leads clients to send `terminal/*`/`fs/*` requests the wrapper must then fail mid-turn. Orca itself advertises `fs:false, terminal:false` [V-EXP] — a good floor to test against. [V-SPEC: initialization page]

**Prevention:** Implement v1 exactly; echo `1`; declare an honest, minimal capability set (no fs, no terminal in v1 — matches PROJECT.md scope); include `agentInfo{name, version}` (required fields per spec); on initialize from an unknown future version (e.g. `2`), reply with the newest *supported* version and let the client decide — never guess-support a newer major.

**Phase to address:** Phase 2. Conformance-test the initialize exchange against the spec examples.

### Pitfall 13: Testing against the real desktop — quota burn, flakes, and the fake app-server

**What goes wrong:** Tests that spawn the real app-server + hit api.z.ai are slow (seconds per turn), network-flaky, depend on Secret Service state, mutate account state (PAT minting *creates* API keys on the user's z.ai org), burn quota credits (`credits = (in×6.9 + cached×1.7 + out×24)/10000` per FINDINGS), and break on every desktop update (Pitfall 5). CI becomes red-for-unrelated-reasons and the team stops trusting it. Recorded fixtures carry live credentials (see Security Mistakes).

**Prevention:** Three tiers, matching the PROJECT.md decision: (1) protocol CI against a **fake app-server** — a Go test double speaking the verified JSONL dialect, scripted per scenario (happy turn, thinking-model setModel, snapshot-drop, interaction flows, unknown method, oversized line, stderr noise); (2) VM E2E with the real desktop for the small set of things the fake can't prove (Electron spawn, keyring, revision resolution); (3) a minimal manual Orca E2E checklist. Fixtures recorded from real sessions are scrubbed and version-tagged by desktop release. Never let CI call api.z.ai.

**Phase to address:** Fake server is a Phase 2 deliverable (it *is* the protocol spec made executable); VM E2E Phase 5.

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Phase 1 protocol research | Documenting 3.17.x-source behavior as truth for the 3.18.2 binary (`requestCredentialCipher` exists only in the binary) | Mark every protocol claim V-SRC vs V-EXP; drive the binary once per documented flow |
| Phase 1 Orca research | Grepping packed `app.asar` instead of `app.asar.unpacked`; concluding "no ACP" | Research only in `resources/app.asar.unpacked/out/main/chunks/` (FINDINGS §4) |
| Phase 2 transport | `jsonrpc` key, LSP framing, scanner 64 KB, same-goroutine request/response | Dedicated JSONL codec + goroutine topology from Pitfall 7, all under `-race` |
| Phase 2 session lifecycle | Premature prompt response; -32010 on overlapping prompts | Turn state machine + scripted fake-server sequences |
| Phase 3 auth | Trusting ack; headersApplied:false; PAT mint without timeout; keyring hang in CI | Registry-verify helper; interaction answering always-promptly; no-keyring error path test |
| Phase 4 Orca recipe | Name/version detection failure; strict flag parsing; slow `initialize` | `omp` persona, single-line `--version`, liberal flag handling, fast initialize |
| Phase 5 CI/VM | Flaky real-desktop tests; fixture secret leakage; orphaned processes in CI runners | Three-tier testing (Pitfall 13); fixture scrubber; process-leftover assertion |

## Sources

**Experimentally verified (HIGH) — FINDINGS.md** (`FINDINGS.md (private research notes, outside this repo)`, real 3.18.2 machine):
- §2 (JSONL transport, jsonrpc-key rejection, revision table 28→44, credentials layout), §3.1 (silent snapshot drop, reasoningLevel loss, headersApplied enforcement), §3.3 (PAT minting endpoint sequence), §3.4 (handshake 4001/4011 with wrong token), §3.5 (Node IPv6/proxy traps), §4 (Orca ACP client in `app.asar.unpacked`, hardcoded agents, env/flags), §6 (pitfall summary list 1–8)

**Source-tree verified (HIGH)** — `ZCode source tree (github.com/zai-org/ZCode; local clone)` (3.17.x public tree):
- `packages/provider/src/registry-service.ts:205-212` — silent snapshot retention on revision mismatch
- `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/account-provider-config.ts:28-33` — `received`/`unchanged` ack semantics
- `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/model-mapper.ts:4-6` + `app/provider-registry-selection.ts:133-142` — model-selection serialization and reasoning-level errors
- `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/provider-runtime-headers.ts` — headersApplied throw (-32031), 180 s timeout, cancel notification, `rejectedProjectTokenFingerprint`
- `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/server-operations.ts:2025-2100` — fire-and-forget `session/send`, -32010 concurrency guard, expected-revision asserts
- `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/interaction-broker.ts:60,70-103` — permission broker, cancel/timedOut/noClientAttached error classes, 1 s re-announce
- `apps/zcode-cli/packages/bootstrap/src/zcode-protocol/server.ts:740-757` — method dispatch incl. `session/stop`, `session/setThoughtLevel`
- `packages/shared/src/zcode-protocol/index.ts` — `.strict()` zod schemas; method list **without** `interaction/requestCredentialCipher` (drift evidence)

**ACP v1 spec (HIGH for protocol semantics)** — agentclientprotocol.com: `/protocol/v1/transports` (NDJSON, stdout purity, UTF-8, stdin-close lifecycle), `/protocol/v1/initialization` (version negotiation, capabilities, agentInfo), `/protocol/v1/cancellation` (notification; original request MUST be answered; pending permissions answered `cancelled`; -32800), `/protocol/v1/prompt-turn` (stopReason set, update ordering, cancel-then-update-before-response rule, permission flow)

**Go-specific (HIGH unless noted):**
- `net/http/transport.go` (go1.22 source): `envProxyOnce sync.Once` caching of `httpproxy.FromEnvironment()`; pkg.go.dev `ProxyFromEnvironment` (env vars, localhost exemption) — [V-GO]
- golang/go issue #27505: `Pdeathsig` delivered on parent-*thread* death — [V-GO]
- Felix Geisendörfer, "Killing a child process and all of its children in Go"; Stack Overflow 22470193 — `Setpgid` + negative-PID group kill, zombie reaping via `Wait()` — [V-GO]
- Stack Overflow 21124327 / Go forum on `bufio.ErrTooLong` — 64 KB `bufio.MaxScanTokenSize` default — [V-GO]
- Go Happy Eyeballs default dialer behavior — [M: standard-library documentation, not independently re-fetched this session]

**Unverified / open (labeled LOW-MEDIUM in text):** whether the Electron app-server exits on stdin EOF [M]; Orca's exact startup-probe timeout and full startup method set [L — Phase 1 deliverable]; exact wire behavior of the 1 s interaction re-announce [M]; Go `NO_PROXY` vs curl dialect edge cases [M].

---
*Pitfalls research for: zcode-acp — Go ACP v1 server wrapping ZCode Desktop app-server stdio protocol*
*Researched: 2026-11-19*
