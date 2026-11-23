---
phase: 04-acp-front-minimal-chat-loop
plan: 02
subsystem: acp-front
tags: [acp, sdk-seam, adapter, session-registry, fake-corpus, golden-transcript, test-harness]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: "plan 04-01: acp-go-sdk v0.20.0 pinned behind the acpport seam skeleton + tidy-diff gate, zcode.Conn.AddObserver ordered fan-out, MethodSessionSubscribe/SetModel constants, fake golden corpus machinery"
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: "zcode.Conn codec, turn.Machine (Run/Wait/Cancel, NewMachine nil-conn tolerant), fake.Serve + CompareGolden fail-loud corpus"
provides:
  - "internal/acpport — the complete D-01 seam surface: neutral Handlers vocabulary (InitReport/NewSessionParams/PromptParams/SessionNote/UpdateSink/StopReason, NewInvalidRequest/NewInternal), the 12-method acp.Agent adapter (honest D-02 initialize assembly, NewSession/Prompt/Cancel delegation, -32601 defaults for the 8 unadvertised stable methods), and the connection-bound UpdateSink emitting agent_message_chunk/agent_thought_chunk wire shapes"
  - "TestSDKImportBoundary — the structural T-4-SC seam lock: zero non-test coder/acp-go-sdk imports outside internal/acpport"
  - "internal/core SessionRegistry — mutex map ACP session id → Session{Machine, CancelTurn, Subscribed}, loud ErrNoSession lookup miss, last-wins Register"
  - "fake corpus row setmodel-followup — scenario + replay driver + recorded golden byte-locking the front-tier contract on the back-leg wire: bare create → subscribe {deliveryKind:desktop-continuous} idle result → setModel carrying model.options.reasoningLevel + persistAsWorkspaceLastUsed:false → send with two part.delta text rows → turn.completed, in wire order"
  - "internal/acpfront test harness (package acpfront_test) — TeeRW recording tee, RecordingClient SDK test ACP client, ComposeFront parameterized on any caller-supplied acp.Agent impl over the fake-backed zcode.Conn (compile-only at this plan's boundary; plan 04-06's tracer is the first full-loop consumer)"
affects: [04-03, 04-04, 04-05, 04-06, phase-5-chat-cycle, phase-6-auth]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 20973 # chars/4 over the realized diff (83891 diff chars, 2056 lines)
  tasks: 3
  commits: 8 # MEASURED: git rev-list --count 31cea50..HEAD

commits: 8
plan_head_before: 31cea50d25231308d9a424b88386127ea48d21a9
plan_head_after: 6abdc1e140713b97979bc5be16729285111b72eb

# Tech tracking
tech-stack:
  added: [] # zero new dependencies — the SDK (04-01) remains the module's sole require line
  patterns:
    - "Seam adapter construction: acpport.NewAgent(Handlers) → acp.Agent + acpport.NewAgentSideConnection(any acp.Agent, in, out) → (*AgentSideConnection, UpdateSink) with the sink bound into our adapter pre-traffic via the unexported sinkBinder hook"
    - "Coded-error passthrough: front errors map onto the SDK's *RequestError UNWRAPPED because the SDK's toReqErr type-asserts the concrete type (no errors.As) — %w wrapping would silently downgrade every coded seam error to a generic -32603"
    - "Compile-only test infrastructure ahead of its consumer, exported from a _test package so the strict unused linter stays green until the phase's later plans exercise it"

key-files:
  created:
    - internal/acpport/types.go
    - internal/acpport/agent.go
    - internal/acpport/sink.go
    - internal/acpport/agent_test.go
    - internal/core/registry.go
    - internal/core/registry_test.go
    - internal/acpfront/doc.go
    - internal/acpfront/harness_test.go
    - internal/zcode/fake/testdata/scenarios/setmodel-followup.json
    - internal/zcode/fake/testdata/golden/setmodel-followup.golden
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-02-red-setmodel-corpus.json
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-02-red-acpport-adapter.json
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-02-red-core-registry.json
  modified:
    - internal/zcode/fake/golden_test.go
    - internal/acpport/doc.go
    - .golangci.yml

key-decisions:
  - "acpport exposes TWO construction seams: NewAgent(h Handlers) builds the adapter (the D-01 translation point) and NewAgentSideConnection(agent acp.Agent, in, out) wraps ANY acp.Agent impl while binding the update sink — the harness compose contract ('parameterized on a caller-supplied acp.Agent impl wrapped by acpport.NewAgentSideConnection') holds verbatim; plan 04-06 passes acpport.NewAgent(front)"
  - "ComposeFront takes a conn-carrying constructor callback (func(conn *zcode.Conn) acp.Agent) so caller agents that own the back-leg conn (04-06's Front) are constructed before any frame flows — the Phase-3 construction discipline, parameterized"
  - "Front errors are wireError-coded on ALL four advertised methods, not just Prompt: the SDK's toReqErr type-asserts *RequestError without errors.As, so an untranslated acpport.Error would degrade to a generic -32603 carrying only the message string (Rule 1 fix, test-pinned)"
  - "The .golangci.yml wrapcheck exclusion for internal/acpport documents the same SDK semantic: the seam returns the SDK's coded RequestErrors unwrapped BY CONTRACT"
  - "SessionRegistry.Register is last-wins (the front is the only writer; ids are app-server-generated opaque strings); Session fields CancelTurn/Subscribed are exported because the strict testpackage linter forces external test packages, which cannot populate unexported fields"
  - "The setmodel-followup driver locks synthetic selection values (zai-api/GLM-5.3-Flash, reasoningLevel high, persistAsWorkspaceLastUsed:false) — the golden byte-asserts the payload verbatim, so the front-tier tracer's setModel must match or the corpus row fails"

patterns-established:
  - "acpport adapter contract testing: scripted Handlers fake + real AgentSideConnection over io.Pipe captures the actual outbound wire frames (full JSON-RPC envelopes, byte-compared) — the probe shapes in RESEARCH are the params payload inside the envelope"
  - "TeeRW recording tee: io.ReadWriter over one connection side's legs with newline reassembly and direction-tagged frames in shared recorded order — the wire-order assertion surface for CORE-03 (chunks before stopReason)"
  - "RED→GREEN evidence discipline reused across all three tasks: compiling stubs fail planned assertions (never build failures), records classified RED_EVIDENCE_OK before GREEN commits"

requirements-completed: [CORE-01, CORE-02, CORE-03, CORE-04, CORE-05] # copied verbatim from plan frontmatter; delivery is shared across plans 04-02..04-06 — requirements.ready-ids returned [], so NOTHING is marked complete by this plan (correct: 04-03..04-06 have no SUMMARYs yet)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "acpport seam package: neutral Handlers vocabulary + 12-method acp.Agent adapter — honest D-02 initialize assembly (byte-locked to the executed probe frame, no fs/terminal/loadSession/type surface), NewSession/Prompt/Cancel delegation with neutral DTO translation, coded front errors, -32601 defaults for the 8 unadvertised stable methods, and the connection-bound sink emitting agent_message_chunk/agent_thought_chunk shapes"
    requirement: CORE-01
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestInitializeAssemblesHonestReport|TestUnadvertisedMethodsAnswerMethodNotFound|TestPromptMapsStopReasonsAndCodesErrors|TestNewSessionAndCancelDelegate|TestFrontErrorsCodedOnAllAdvertisedMethods|TestUpdateSinkEmitsChunkShapes (go test -race -count=1 ./internal/acpport/)"
        status: pass
    human_judgment: false
  - id: D2
    description: "SDK import boundary lock (T-4-SC): no non-test file outside internal/acpport imports coder/acp-go-sdk; _test.go files exempt per RESEARCH Pattern 6"
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestSDKImportBoundary (repo walk, go/parser ImportsOnly)"
        status: pass
    human_judgment: false
  - id: D3
    description: "core.SessionRegistry: register/get/remove under a mutex, same-machine-pointer resolution, last-wins re-registration, loud ErrNoSession lookup miss, remove-no-op safety, -race concurrency smoke"
    verification:
      - kind: unit
        ref: "internal/core/registry_test.go#TestRegistryRegisterGetRemove|TestRegistryConcurrentAccess (go test -race -count=1 ./internal/core/)"
        status: pass
    human_judgment: false
  - id: D4
    description: "setmodel-followup corpus row: scenario + driveSetModelFollowup driver + recorded golden byte-asserting the full exchange in wire order — bare create (no model object), subscribe idle result {sessionId,eventSeq:0,events:[]}, setModel carrying model.options.reasoningLevel + persistAsWorkspaceLastUsed:false, send with two part.delta text rows and turn.completed"
    requirement: CORE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/setmodel-followup (real zcode.Conn over io.Pipe, -race)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Front-tier test harness compiled and ready: TeeRW recording tee (both directions, wire order), RecordingClient SDK test ACP client (update collector), ComposeFront over any caller-supplied acp.Agent impl — no acpfront dependency; plan 04-06's tracer is the first full-loop consumer"
    verification:
      - kind: other
        ref: "go vet ./internal/acpfront/ && go test -race -count=1 -run TestNothingPlaceholder ./internal/acpfront/ (compiles, no tests to run — compile-only boundary per plan)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Zero regressions: full gate green — build, vet, lint 0 issues, full -race internal suite, experiments suite"
    verification:
      - kind: integration
        ref: "mise run ci (build + vet + lint + go test -race -count=1 ./... + node --test experiments/)"
        status: pass
    human_judgment: false

# Metrics
duration: 32min
completed: 2026-11-20
status: complete
---

# Phase 4 Plan 02: ACP Front Enablement Summary

**internal/acpport lands as the complete SDK seam (adapter + sink + structural import-boundary test), internal/core gets the session registry, the setmodel-followup corpus row byte-locks the front-tier contract on the back-leg wire, and the front-tier test harness compiles ready for the 04-06 tracer**

## Performance

- **Duration:** 32 min
- **Started:** 2026-11-19T23:49:58Z
- **Completed:** 2026-11-20T00:22Z
- **Tasks:** 3 (all via RED→GREEN TDD with classified evidence)
- **Files:** 13 created / 3 modified

## Accomplishments
- `internal/acpport` is the complete D-01 seam: neutral Handlers vocabulary (types.go), the 12-method `acp.Agent` adapter (agent.go — honest D-02 initialize byte-locked to the executed probe frame, NewSession/Prompt/Cancel delegation over neutral DTOs, coded front errors, -32601 defaults for the 8 unadvertised stable methods naming each method), and the connection-bound UpdateSink (sink.go — message/thought notes map onto the SDK chunk helpers, unknown kinds fail loud). Contract-tested over a scripted Handlers fake with a real AgentSideConnection capturing full JSON-RPC wire frames
- `TestSDKImportBoundary` walks every non-test .go file in the repo and fails on any SDK import outside internal/acpport — the T-4-SC supply-chain containment is now structural, not review-enforced
- `internal/core.SessionRegistry` registers/gets/removes under a mutex, resolves the SAME machine pointer, fails loud with ErrNoSession on lookup miss, is last-wins on re-registration — unit-tested including a parallel -race smoke (Open Question 5 resolution: the registry lives in core now, TurnTranslator deferred)
- The `setmodel-followup` corpus row replays through a real zcode.Conn under -race and its golden byte-asserts the front-tier contract in wire order: bare create → subscribe {deliveryKind:desktop-continuous} with the idle result {sessionId, eventSeq:0, events:[]} → setModel carrying `model.options.reasoningLevel` AND `persistAsWorkspaceLastUsed:false` → send with two part.delta text rows ("Hello", " world") and turn.completed — the tracer's setModel payload is now corpus-locked
- The front-tier test harness compiles under -race: TeeRW recording tee (direction-tagged newline reassembly, wire-order surface), RecordingClient (SDK Client interface, SessionUpdate collector, Phase-5+ callbacks as no-ops), ComposeFront wiring fake.Serve ⇄ zcode.Conn ⇄ acpport connection ⇄ SDK test client — parameterized on any caller-supplied acp.Agent impl, importing NO acpfront code
- `mise run ci` fully green: build, vet, golangci-lint 0 issues, full -race internal suite, experiments

## Task Commits

Each task committed atomically (TDD: RED commit precedes GREEN per feature):

1. **Task 1 (RED): setmodel-followup scenario + replay driver** - `79c7f36` (test)
2. **Task 1 (GREEN): recorded golden — setModel payload locked on the wire** - `c964c10` (feat)
3. **Task 2 (RED): acpport contract tests + neutral vocabulary + stub adapter** - `fd57b1f` (test)
4. **Task 2 (GREEN): adapter translations + sink** - `5c8be6f` (feat)
5. **Task 2/1 (fix): front errors coded on ALL advertised methods + lint hardening** - `f0627fb` (fix)
6. **Task 3 (RED): SessionRegistry unit tests** - `68b5df3` (test)
7. **Task 3 (GREEN): registry mutex map** - `30ebae8` (feat)
8. **Task 3 (feat): front-tier test harness** - `6abdc1e` (feat)

**Plan metadata:** committed in the close-out docs commit (this file + STATE/ROADMAP/REQUIREMENTS/WINDOWS).

## TDD Record

Three RED→GREEN cycles, each with a classifier-verified evidence record (`evidence/04-02-red-*.json`, all `RED_EVIDENCE_OK`, go-test adapter):

- **RED 1 — corpus row** (`evidence/04-02-red-setmodel-corpus.json`): TestGolden/setmodel-followup failed on "golden fixture missing" (the fail-loud record-honesty gate); all ten pre-existing goldens stayed green; the driver resolved and the exchange replayed clean — only the fixture was absent. **Semantic assessment:** genuine RED on the planned assertion (the GREEN step records the fixture with -update).
- **RED 2 — acpport adapter** (`evidence/04-02-red-acpport-adapter.json`): the compiling stub adapter failed all 5 contract tests (17 subtests) on planned assertions — protocolVersion 0 want 1, nil errors want coded -32601, zero Prompt responses, empty session id, zero sink frames. **Semantic assessment:** genuine RED; recorded nuance — TestSDKImportBoundary is green at RED by design (an absence-guard, satisfied before the implementation lands).
- **RED 3 — registry** (`evidence/04-02-red-core-registry.json`): the stub registry answered everything with a sentinel session, failing pointer-identity resolution, the ErrNoSession miss probe, and remove-then-miss. **Semantic assessment:** genuine RED on identity + sentinel assertions the stub cannot satisfy.

## Files Created/Modified
- `internal/acpport/types.go` — Handlers interface, InitReport/AuthMethodDef, NewSessionParams/McpServerDef, PromptParams, SessionNote/NoteMessage/NoteThought, UpdateSink, StopEndTurn/StopCancelled, Error + NewInvalidRequest/NewInternal
- `internal/acpport/agent.go` — agentAdapter (12 methods), NewAgent, NewAgentSideConnection (sink binding via sinkBinder), initializeResponse, newSessionParams/mcpServerDef, joinContentBlocks (text verbatim, resource_link as URI per A3), sinkForPrompt, wireError, unadvertised
- `internal/acpport/sink.go` — NewUpdateSink + noteToNotification (message → agent_message_chunk, thought → agent_thought_chunk)
- `internal/acpport/agent_test.go` — TestSDKImportBoundary + 6 contract tests + scriptedHandlers fake
- `internal/acpport/doc.go` — skeleton note updated to "landed with plan 04-02"
- `internal/core/registry.go` / `registry_test.go` — SessionRegistry + Session + ErrNoSession, unit tests
- `internal/acpfront/doc.go` / `harness_test.go` — package contract + the compile-only harness (TeeRW, RecordingClient, FrontHarness, ComposeFront)
- `internal/zcode/fake/testdata/scenarios/setmodel-followup.json` + `testdata/golden/setmodel-followup.golden` — the new corpus row
- `internal/zcode/fake/golden_test.go` — driveSetModelFollowup registered in goldenDrivers (corpus stays fail-loud)
- `.golangci.yml` — wrapcheck exclusion for internal/acpport (documented: the seam returns the SDK's coded RequestErrors unwrapped by contract)

## Decisions Made
- Two construction seams in acpport: `NewAgent(h Handlers)` (the adapter) and `NewAgentSideConnection(agent acp.Agent, in, out)` (wraps any acp.Agent impl + binds the sink) — the plan's compose contract holds verbatim while Pattern 1's Handlers route stays the production path
- ComposeFront parameterizes on a conn-carrying constructor callback so Front-owns-conn composition (04-06) constructs before traffic
- Front errors are wireError-coded on ALL advertised methods — the SDK's toReqErr type-asserts `*RequestError` (no errors.As), so raw passthrough would degrade coded seam errors to generic -32603
- Harness surface exported from `package acpfront_test` (this phase's shared test infra for 04-03..04-06) — keeps the strict `unused` linter green at the compile-only boundary
- setModel payload values locked synthetic (zai-api/GLM-5.3-Flash, reasoningLevel "high", persistAsWorkspaceLastUsed:false) in the driver; the golden enforces them byte-for-byte
- Sink wire-shape assertions strengthened to the FULL JSON-RPC envelope — the executed-probe shapes in RESEARCH are the params payload inside it (`{"jsonrpc":"2.0","method":"session/update","params":{...}}`)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] internal/acpfront/doc.go added (not in the plan's files_modified)**
- **Found during:** Task 3
- **Issue:** harness_test.go declares `package acpfront_test` — an external test package requires the base package to exist; with only the test file present, `go vet ./internal/acpfront/` fails ("no non-test Go files").
- **Fix:** added the package contract doc (machine.go decision-ID comment style) naming the Front's contracts and plan 04-06 as the first consumer.
- **Files modified:** internal/acpfront/doc.go
- **Committed in:** 6abdc1e

**2. [Rule 1 - Bug] Front errors were coded only on Prompt; Initialize/NewSession/Cancel passed them raw**
- **Found during:** Task 2 GREEN (lint's wrapcheck investigation surfaced the SDK semantics)
- **Issue:** the SDK's `toReqErr` type-asserts `*RequestError` without errors.As — a raw `*acpport.Error` from those methods would degrade to a generic -32603 carrying only the message, breaking the seam's coded-error contract.
- **Fix:** all four advertised methods route front errors through wireError (now nil-safe); TestFrontErrorsCodedOnAllAdvertisedMethods pins the coding on Initialize/NewSession/Cancel; the same semantics justified the documented .golangci.yml wrapcheck exclusion for internal/acpport.
- **Files modified:** internal/acpport/agent.go, internal/acpport/agent_test.go, .golangci.yml
- **Committed in:** f0627fb

**3. [Rule 3 - Blocking] Session.CancelTurn/Subscribed exported (plan sketch had them lowercase)**
- **Found during:** Task 3 RED authoring
- **Issue:** the strict testpackage linter forces external test packages (`core_test`), which cannot set unexported struct fields — the unit tests could not construct complete Session values with the sketch's lowercase fields.
- **Fix:** exported CancelTurn and Subscribed (ID/Machine were already exported in the sketch); Register's last-wins semantics documented at the type.
- **Files modified:** internal/core/registry.go
- **Committed in:** 68b5df3 / 30ebae8

---

**Total deviations:** 3 auto-fixed (1 bug, 2 blocking) — none architectural, no Rule 4 decisions required.
**Impact on plan:** all three fixes serve the plan's own gates (compiling verify commands, lint-clean CI, the coded-error contract). Prohibitions hold: no synthesized chunks (the sink is a passthrough with loud unknown-kind failures), stdout stays protocol-exclusive (no stdout writes in any new file; no logging added).

## Issues Encountered
- `go test -update` is a test-binary flag: it must come AFTER the package list (`go test -run <pat> ./pkg -update`), otherwise the go command stops flag parsing at the unknown flag and the package argument vanishes ("no Go files in cwd").
- golangci-lint (strict v2.15.0, default:all) flagged 76 issues across the new files after GREEN; all resolved in the fix/harness commits — funcorder ordering, inamedparam, ireturn (NewAgent now returns the concrete adapter), goconst synthetic-id constants, gocognit (subtest bodies extracted to top-level funcs), nlreturn blank-line-before-return, err113 package-level sentinels, gochecknoglobals (honestReport became a builder func), lll multi-line signatures/frames, and the 34 `unused` flags on the compile-only harness (resolved by exporting the shared test-infra surface).

## Known Stubs

| Stub | File | Reason | Resolved by |
|------|------|--------|-------------|
| ComposeFront is never invoked (harness is compile-only; verify filters to TestNothingPlaceholder) | internal/acpfront/harness_test.go | intentional per plan: "Compile-only at this task's boundary — the tracer's RED test is the first consumer of the full composition" | plan 04-06 (tracer RED) |

Not recorded in the windows ledger: this is the plan's own scoped sequencing (its verify command defines the boundary), not a defect — the harness helpers are fully implemented, only unexercised. The 04-01 acpport-skeleton stub (windows entry 8) IS resolved by this plan and marked fixed in `.planning/WINDOWS.md` (open_count 8 → 7).

## User Setup Required

None - no external service configuration required (SDK read from the module cache; no quota cost).

## Next Phase Readiness
- Plans 04-03..04-05 implement Handlers consumers: the vocabulary (InitReport, PromptParams, SessionNote, StopReason, NewInvalidRequest/NewInternal), the registry, the replaying corpus row, and the ready harness are all compiled and green
- The Authenticate -32601 default is the deliberate exception (Open Question 2) — plan 04-03's RED target (TestAuthenticateHonestError) flips it to the honest not-available error
- The tracer (04-06) composes: acpport.NewAgent(front) → ComposeFront — the full-loop RED rides the setmodel-followup golden and the tee's wire-order surface
- No blockers

---
*Phase: 04-acp-front-minimal-chat-loop*
*Completed: 2026-11-20*

## Self-Check: PASSED

All created files exist on disk; all 8 task commits (79c7f36, c964c10, fd57b1f, 5c8be6f, f0627fb, 68b5df3, 30ebae8, 6abdc1e) are ancestors of HEAD (6abdc1e). Commit count measured from the persisted plan ledger: 8 (31cea50..HEAD). `mise run ci` green at HEAD.
