---
phase: 04-acp-front-minimal-chat-loop
plan: 03
subsystem: acp-front
tags: [acp, contract-lock, honesty, authenticate, unadvertised-methods, cwd-validation, mcp-servers, setmodel-payload, tdd, red-green]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: "plan 04-02: the acpport seam (agentAdapter + neutral vocabulary + error helpers) and the front-tier test harness; plan 04-06: the tracer Front (D-02 initialize report, D-03 session/new sequence, newTracerFront composition) whose behaviors this plan locks; plan 04-01: the SDK pin and conn fan-out underneath"
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: "fake.Serve transcript (the back-leg assertion surface), the setmodel-followup corpus row, zcode.Conn"
provides:
  - "TestInitializeHonestCapabilities — the CORE-01 honesty lock on the RAW initialize response frame (decoded nothing): the executed-probe honest shape byte-locked (protocolVersion 1, agentInfo zcode-acp, both authMethods in contract order without a type field), loadSession/readTextFile/writeTextFile/terminal/fs absence asserted (D-02/A7)"
  - "TestAuthenticateHonestError + the honest authenticate handler — the deliberate non-(-32601) exception: authenticate (advertised per D-02) answers a -32603-class RequestError whose message field carries 'authentication is not available in this build yet (z.ai auth flows land in Phases 5-6)' — never fake success, never MethodNotFound (A2/D-02)"
  - "TestUnadvertisedStableMethodsAnswerNotFound — the seven unadvertised stable methods driven through a REAL agent-side connection with raw JSON-RPC frames, each answering -32601 naming the method with Handlers untouched (Pitfall 6 lock at the wire tier)"
  - "TestNewSessionRejectsRelativeCwd + the pre-wire guard — T-4-01/V5 mitigation: session/new with a relative (or empty) cwd answers -32602 invalid-params naming the absolute requirement BEFORE any backend call; the back-leg transcript shows no session/create frame"
  - "TestNewSessionAcceptsMcpServers — D-03 accept-and-ignored lock: a non-empty mcpServers list (HTTP name+url entry) succeeds and no mcp content ever reaches the backend create frame"
  - "TestNewSessionCreatesAndSetsModel — the CORE-02 payload lock: create bare (no model key, Pitfall 4) → subscribe desktop-continuous → setModel carrying model.options.reasoningLevel + persistAsWorkspaceLastUsed:false, in wire order on the fake transcript"
  - "acpport.NewInvalidParams (-32602) — the invalid-params error helper mirroring NewInvalidRequest/NewInternal"
affects: [04-04, 04-05, phase-5-chat-cycle, phase-6-auth]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 7421 # chars/4 over the realized diff (29685 diff chars, 450 insertions / 21 deletions across 8 files)
  tasks: 2
  commits: 4 # MEASURED: git rev-list --count c108baa..HEAD

commits: 4
plan_head_before: c108baac6d4b8750ed4b476209517cfecd1db1e0
plan_head_after: 83f5b4e54f3323eb3d4d94cc9f82e66cbe108ea4

# Tech tracking
tech-stack:
  added: [] # zero new dependencies — the SDK (04-01) remains the module's sole require line
  patterns:
    - "Connection-tier contract locks: acpport.NewAgentSideConnection over io pipes driven with RAW JSON-RPC frames asserts the SDK dispatcher's actual wire answer (error code + data.method) — a tier above adapter-method tests, reusing the package's drainFrames/readFrame helpers"
    - "Raw-frame honesty locks: the initialize contract is asserted on the recorded tee line with substring containment only (decoded nothing) — the executed-probe shape as one byte-exact want covers protocolVersion, agentInfo, authMethod order/count/variant in a single lock"
    - "Transcript-first RED ordering: the no-create-frame assertion runs BEFORE the error-shape assertions so the RED lands exactly on the planned T-4-01 assertion (create IS attempted today) instead of the trivially-failing missing-error check"

key-files:
  created:
    - internal/acpfront/initialize_test.go
    - internal/acpfront/newsession_test.go
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-03-red-authenticate.json
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-03-red-cwd.json
  modified:
    - internal/acpport/agent.go
    - internal/acpport/agent_test.go
    - internal/acpfront/front.go
    - internal/acpport/types.go

key-decisions:
  - "Authenticate answers acp.NewRequestError(-32603, readable-message, nil) rather than NewInternalError: the SDK's NewInternalError puts the text in error.data behind a generic 'Internal error' message, while A2's user-visible wording must ride error.message — the honest message is what a client renders"
  - "The relative-cwd guard REPLACES the tracer's empty-cwd check (NewInvalidRequest 'cwd required'): filepath.IsAbs(\"\") is false so one check covers both, and invalid-params (-32602) is the accurate class for a malformed param — the pre-existing empty-cwd -32600 answer had no test pinning it"
  - "The unadvertised-method lock drives the REAL connection (raw frames through the SDK dispatcher), not just adapter methods — the wire answer (error object with code -32601 and data.method) is the truth a client actually sees; session/set_config_option's union decoder forced real params (the value-id variant keys) even for a method that is refused anyway"
  - "Authenticate is REMOVED from the pre-existing TestUnadvertisedMethodsAnswerMethodNotFound table (it is advertised per D-02, so it never belonged among the unadvertised defaults) — its honest contract now lives in TestAuthenticateHonestError at the connection tier"
  - "The probe-ledger's flagged assumption stands: CORE-02's edge space is fully covered by explicit tests (relative cwd, empty-vs-non-empty mcpServers, payload verbatim) but the probe classifier offered no category — carried as a flagged assumption, not auto-resolved"

requirements-completed: [CORE-01, CORE-02] # copied verbatim from plan frontmatter; ready-ids reported CORE-01+CORE-02 after this SUMMARY (04-01/04-02/04-06 + 04-03 all carry SUMMARYs)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "CORE-01 honesty lock on the RAW initialize response frame (decoded nothing): the executed-probe honest shape byte-locked via one substring lock on the tee line — protocolVersion 1, agentInfo zcode-acp/zcode-acp agent/0.0.0-dev, both authMethods in contract order (zai-api-key then zai-individual-coding-plan) without a type field — plus absence assertions for loadSession/readTextFile/writeTextFile/terminal/fs (D-02/A7)"
    requirement: CORE-01
    verification:
      - kind: integration
        ref: "internal/acpfront/initialize_test.go#TestInitializeHonestCapabilities (SDK client ⇄ tee ⇄ Front over the setmodel-followup corpus, -race; go test -race -count=1 -run TestInitializeHonestCapabilities ./internal/acpfront/)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Authenticate answers the honest not-available error — the deliberate non-(-32601) exception: a -32603-class RequestError whose message field carries the stable readable 'authentication is not available in this build yet (z.ai auth flows land in Phases 5-6)' (A2), driven through a real agent-side connection with {\"methodId\":\"zai-api-key\"}; never fake success, never MethodNotFound, Handlers untouched"
    requirement: CORE-01
    verification:
      - kind: integration
        ref: "internal/acpport/agent_test.go#TestAuthenticateHonestError (raw JSON-RPC frame through acpport.NewAgentSideConnection, -race; go test -race -count=1 -run TestAuthenticateHonestError ./internal/acpport/)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The advertise⇄implement boundary is exact: the seven unadvertised stable methods (Logout, CloseSession, DeleteSession, ListSessions, ResumeSession, SetSessionConfigOption, SetSessionMode) each answer -32601 MethodNotFound naming the method on the actual wire (raw frames through the SDK dispatcher), with zero Handlers invocations — authenticate excluded as the deliberate exception"
    requirement: CORE-01
    verification:
      - kind: integration
        ref: "internal/acpport/agent_test.go#TestUnadvertisedStableMethodsAnswerNotFound (+ TestUnadvertisedMethodsAnswerMethodNotFound at the adapter tier, -race; go test -race -count=1 -run TestUnadvertisedStable ./internal/acpport/)"
        status: pass
    human_judgment: false
  - id: D4
    description: "T-4-01/V5 mitigation: session/new with a RELATIVE cwd answers -32602 invalid-params naming the absolute requirement BEFORE any backend call — the back-leg fake transcript shows NO session/create frame; the guard folds the empty-cwd case (IsAbs(\"\") is false)"
    requirement: CORE-02
    verification:
      - kind: integration
        ref: "internal/acpfront/newsession_test.go#TestNewSessionRejectsRelativeCwd (transcript-first assertion over the setmodel-followup composition, -race; go test -race -count=1 -run TestNewSession ./internal/acpfront/)"
        status: pass
    human_judgment: false
  - id: D5
    description: "D-03 accept-and-ignored lock: session/new with a non-empty mcpServers list (one HTTP entry, name+url shape) succeeds, returns a sessionId, and the backend create IN frame carries no mcp content — the list never reaches the backend wire"
    requirement: CORE-02
    verification:
      - kind: integration
        ref: "internal/acpfront/newsession_test.go#TestNewSessionAcceptsMcpServers (fake-transcript mcp-absence assert, -race)"
        status: pass
    human_judgment: false
  - id: D6
    description: "CORE-02 payload lock over the tracer's D-03 sequence: back-leg transcript order create → subscribe → setModel; the setModel IN frame carries model.options.reasoningLevel (zai-api/GLM-5.3-Flash/high, corpus-locked) and explicit persistAsWorkspaceLastUsed:false; the create IN frame stays bare — no model key (Pitfall 4 regression lock)"
    requirement: CORE-02
    verification:
      - kind: integration
        ref: "internal/acpfront/newsession_test.go#TestNewSessionCreatesAndSetsModel (assertCreateFrame/assertSubscribeFrame/assertSetModelFrame over the fake transcript, -race)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Zero regressions: full gate green — build, vet, lint 0 issues, full -race internal suite, experiments suite"
    verification:
      - kind: integration
        ref: "mise run ci (build + vet + lint + go test -race -count=1 ./... + node --test experiments/)"
        status: pass
    human_judgment: false

# Metrics
duration: 16min
completed: 2026-11-20
status: complete
---

# Phase 4 Plan 03: Contract Locks — Initialize Honesty + session/new Contract Summary

**The CORE-01/CORE-02 honesty and session-contract locks land as six named tests with two RED→GREEN deltas: authenticate flips from the misleading -32601 default to the honest -32603-class not-available error, and a relative cwd is rejected with -32602 before a single backend frame — plus the raw-frame initialize honesty lock, the connection-tier -32601 defaults lock, mcpServers accept-and-ignored, and the setModel payload (reasoningLevel + persist:false) verbatim**

## Performance

- **Duration:** 16 min
- **Started:** 2026-11-20T00:59:05Z
- **Completed:** 2026-11-20T01:15Z
- **Tasks:** 2 (both via RED→GREEN TDD with classified evidence)
- **Files:** 4 created / 4 modified

## Accomplishments
- `TestInitializeHonestCapabilities` locks CORE-01 on the RAW tee frame, decoded nothing: one byte-exact want carries the executed-probe honest shape (protocolVersion 1, agentInfo zcode-acp, both authMethods in contract order without a type field), and the absence loop asserts loadSession/readTextFile/writeTextFile/terminal/fs never appear (D-02/A7 — absent features, not empty-object cosmetics)
- `TestAuthenticateHonestError` flips the tracer's deliberate -32601 placeholder into the honest contract: authenticate — advertised per D-02, so a client may legitimately call it — answers a -32603-class RequestError whose message field carries "authentication is not available in this build yet (z.ai auth flows land in Phases 5-6)"; never fake success, never the misleading MethodNotFound, Handlers untouched; the agent.go comment names A2/D-02 as the deliberate exception
- `TestUnadvertisedStableMethodsAnswerNotFound` locks the advertise⇄implement boundary at the WIRE tier: all seven unadvertised stable methods driven through a real agent-side connection with raw JSON-RPC frames each answer -32601 naming the method (Pitfall 6), with zero Handlers invocations — authenticate left the pre-existing adapter-tier table (it is advertised, not unadvertised)
- `TestNewSessionRejectsRelativeCwd` (T-4-01/V5 mitigation) fails honestly at RED — the relative cwd flowed through and a create WAS attempted — then lands the pre-wire guard: `filepath.IsAbs` at the top of `Front.NewSession` answers -32602 invalid-params naming the absolute requirement, and the back-leg transcript shows no session/create frame; the empty-cwd case folds into the same check
- `TestNewSessionAcceptsMcpServers` makes D-03 accept-and-ignored permanent: a non-empty mcpServers list (HTTP name+url entry) succeeds and no mcp content ever reaches the backend create frame
- `TestNewSessionCreatesAndSetsModel` locks the CORE-02 payload sequence on the fake transcript: create bare (no model key — Pitfall 4 regression lock) → subscribe desktop-continuous → setModel carrying model.options.reasoningLevel + persistAsWorkspaceLastUsed:false
- `mise run ci` fully green: build, vet, lint 0 issues, full -race internal suite, experiments

## Task Commits

Each task committed atomically (TDD: RED commit precedes GREEN per feature):

1. **Task 1 (RED): initialize honesty lock + connection-level unadvertised lock + failing authenticate honest-error** - `a8a0bb9` (test)
2. **Task 1 (GREEN): authenticate answers the honest not-available error** - `e503df0` (feat)
3. **Task 2 (RED): session/new contract locks with failing relative-cwd rejection** - `d08ff10` (test)
4. **Task 2 (GREEN): reject relative cwd before any backend call in session/new** - `83f5b4e` (feat)

**Plan metadata:** committed in the close-out docs commit (this file + STATE/ROADMAP/REQUIREMENTS).

## TDD Record

Two RED→GREEN cycles, each with a classifier-verified evidence record (`evidence/04-03-red-*.json`, both `RED_EVIDENCE_OK`, go-test adapter):

- **RED 1 — authenticate honest error** (`evidence/04-03-red-authenticate.json`): TestAuthenticateHonestError failed on its first planned assertion — the raw wire answer was `{"code":-32601,"message":"Method not found","data":{"method":"authenticate"}}`, the misleading default the RED target exists to flip. The other two Task 1 tests PASSED as locks (the plan text sanctions exactly this: the initialize honesty shape and the -32601 defaults already exist from 04-02/04-06; a passing lock is the expected contract-lock outcome, never a manufactured RED). **Semantic assessment:** genuine RED on the planned assertion. Two test-authoring fixes happened BEFORE evidence capture so the RED lands only on planned assertions: an unused `context` import (build), and the connection-level probe initially sent `{}` to `session/set_config_option`, whose union decoder rejects the empty object with -32602 — a test-params defect, fixed by sending the value-id variant keys.
- **RED 2 — relative-cwd rejection** (`evidence/04-03-red-cwd.json`): TestNewSessionRejectsRelativeCwd failed on its FIRST planned assertion (the no-create-frame check, deliberately ordered before the error-shape asserts): "relative-cwd session/new must be rejected BEFORE any backend call (T-4-01) — back leg records 1 session/create frame(s)". The other two Task 2 tests PASSED as locks (mcpServers accept-and-ignored and the setModel payload sequence already hold from the tracer). **Semantic assessment:** genuine RED exactly as the plan prescribes; one test-authoring fix before capture — the mcp probe constructed `acp.McpServerHttp` where the SDK union field is `acp.McpServerHttpInline` (compile error, a test-params defect against the SDK surface).
- GREEN-side nuance (Task 1): the first GREEN attempt used `acp.NewInternalError`, which puts the text in `error.data` behind a generic "Internal error" message — switched to `acp.NewRequestError` so A2's user-visible wording rides `error.message`.

## Files Created/Modified
- `internal/acpfront/initialize_test.go` — TestInitializeHonestCapabilities (raw-frame honesty lock) + the shared setModelScenario corpus constant
- `internal/acpfront/newsession_test.go` — TestNewSessionRejectsRelativeCwd (transcript-first RED ordering), TestNewSessionAcceptsMcpServers, TestNewSessionCreatesAndSetsModel (reusing the tracer's assertCreateFrame/assertSubscribeFrame/assertSetModelFrame/transcript helpers)
- `internal/acpport/agent.go` — Authenticate: the honest -32603-class NewRequestError with the stable readable message; adapter type doc updated (seven -32601 defaults + one deliberate exception naming A2/D-02)
- `internal/acpport/agent_test.go` — newConnectionProbe (raw JSON-RPC frames through a real connection) + jsonRPCErrorFrame + TestUnadvertisedStableMethodsAnswerNotFound + TestAuthenticateHonestError; TestUnadvertisedMethodsAnswerMethodNotFound drops the authenticate row
- `internal/acpfront/front.go` — the absolute-cwd pre-wire guard (spec session-setup MUST, T-4-01) replacing the empty-cwd check
- `internal/acpport/types.go` — CodeInvalidParams (-32602) + NewInvalidParams helper

## Decisions Made
- Authenticate answers `acp.NewRequestError(-32603, message, nil)` rather than `NewInternalError` — the SDK's constructor puts the text in error.data behind a generic "Internal error" message; the honest message must be what a client renders (A2)
- The relative-cwd guard replaces the tracer's empty-cwd check: one `filepath.IsAbs` test covers both (empty is not absolute), and -32602 invalid-params is the accurate class for a malformed param (the old -32600 answer had no test pinning it)
- The unadvertised-method lock drives the REAL connection with raw frames — the wire answer (error object, code, data.method) is the truth a client sees, a tier above adapter-method tests
- Authenticate left the pre-existing unadvertised table (advertised per D-02 — it never belonged among the unadvertised defaults); its contract lives in its own named test
- The probe-ledger's flagged assumption stands: CORE-02's edge space is fully covered by explicit tests (relative cwd, empty-vs-non-empty mcpServers, payload verbatim) but the probe classifier offered no category — carried as a flagged assumption, not auto-resolved

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] internal/acpport/types.go modified (not in the plan's files_modified list)**
- **Found during:** Task 2 GREEN authoring
- **Issue:** the plan's behavior demands a "JSON-RPC ERROR in the invalid-params class" for the relative-cwd rejection, but acpport's error vocabulary only carried NewInvalidRequest (-32600) and NewInternal (-32603) — there was no invalid-params (-32602) constructor to return, and front.go cannot import the SDK
- **Fix:** added `CodeInvalidParams = -32602` and `NewInvalidParams` to types.go, mirroring the existing helpers exactly as the plan's own Task 1 action contemplates ("extend acpport's error helpers in types.go if a new class is needed")
- **Files modified:** internal/acpport/types.go
- **Verification:** TestNewSessionRejectsRelativeCwd asserts the -32602 wire code; mise run ci green
- **Committed in:** 83f5b4e

---

**Total deviations:** 1 auto-fixed (1 blocking) — none architectural, no Rule 4 decisions required.
**Impact on plan:** the extension serves the plan's own gate (the invalid-params class its contract names). Prohibitions hold: nothing unimplemented is advertised (the raw-frame lock keeps D-02 exact; authenticate fails honestly, never faking success), and the initialize frame gained no fields this phase.

## Issues Encountered
- The SDK's `NewSessionConfigOptionRequest` union decoder rejects `{}` params with -32602 before the handler is reached — the connection-level probe needed the value-id variant keys even for a method that is refused anyway; a useful reminder that -32602 from the dispatcher means "params never decoded", distinct from the -32601 defaults the probe locks.
- `acp.McpServer`'s HTTP union field is `McpServerHttpInline` (not `McpServerHttp`) — the client-side test fixture compiles against the Inline variant.
- The SDK's `NewInternalError` constructor is message-in-data, not message-in-message — `NewRequestError(code, msg, nil)` is the shape whose message a client actually renders.

## User Setup Required

None - no external service configuration required (fake-backed composition, zero processes, no quota cost).

## Next Phase Readiness
- Plans 04-04/04-05 sharpen the remaining named RED targets on the proven slice: 04-04 the reasoning→thought mapping (the tap's named skip is its RED target), 04-05 the concurrency class sharpening (-32010) + cmd wiring (version var, supervisor lifecycle)
- The six named locks (TestInitializeHonestCapabilities, TestAuthenticateHonestError, TestUnadvertisedStableMethodsAnswerNotFound, TestNewSessionRejectsRelativeCwd, TestNewSessionAcceptsMcpServers, TestNewSessionCreatesAndSetsModel) are the regression perimeter for Phases 5-6 auth/model work — the honest-error and payload shapes are now test-owned, not tracer-demonstrated
- Requirements CORE-01/CORE-02 marked complete (all four declaring plans carry SUMMARYs); CORE-03/04/05 stay pending for 04-04/04-05
- No blockers

---
*Phase: 04-acp-front-minimal-chat-loop*
*Completed: 2026-11-20*

## Self-Check: PASSED

All created/modified files exist on disk (9/9 checked, including this SUMMARY); all 4 task commits (a8a0bb9, e503df0, d08ff10, 83f5b4e) are ancestors of HEAD (83f5b4e). Commit count measured from the persisted plan ledger: 4 (c108baa..HEAD). Both RED evidence records classified RED_EVIDENCE_OK by the gsd-tools gate; `mise run ci` green at HEAD (build, vet, lint 0 issues, full -race internal suite, experiments).
