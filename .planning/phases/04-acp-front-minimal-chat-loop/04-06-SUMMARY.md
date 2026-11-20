---
phase: 04-acp-front-minimal-chat-loop
plan: 06
subsystem: acp-front
tags: [acp, tracer, minimal-chat-loop, streaming, cancel, delta-tap, fake-corpus, golden-transcript, turn-ctx]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: "plan 04-02: the acpport seam (Handlers vocabulary, adapter, connection-bound sink), core.SessionRegistry, the setmodel-followup corpus row + golden, and the front-tier test harness (TeeRW + RecordingClient + ComposeFront); plan 04-01: conn.AddObserver ordered fan-out and the MethodSessionSubscribe/SetModel constants"
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: "turn.Machine (Run/Wait/Cancel exactly-once, ErrTurnActive), zcode.Conn codec, fake.Serve + CompareGolden fail-loud corpus"
provides:
  - "internal/acpfront.Front — the full minimal chat loop behind acpport.Handlers: honest D-02 initialize (probe-verbatim agentInfo + both authMethods, version injectable, default 0.0.0-dev), the D-03 session/new contract (bare create workspaceKey=cwd → subscribe desktop-continuous BEFORE any turn → setModel iff configured with persistAsWorkspaceLastUsed:false), the D-04 prompt turn over the Phase-3 machine, and the D-05 exactly-once cancel"
  - "the acpfront delta tap (stream.go) — registered via conn.AddObserver BEFORE any machine (flush-before-stopReason registration order): session/event part.delta with field text becomes exactly ONE agent_message_chunk in arrival order; reasoning/input/output silently skipped (04-04/Phase-5 named comments); never an upsert"
  - "the front-wide turn slot + turn-context ownership (turnctx.go): single-active-turn per front (the delta-routing precondition), front-owned turn contexts (never the SDK request ctx — cancelled resolves as a SUCCESS stopReason, never -32800), cancel fan-in goroutine, tap-facing emitter published under one lock"
  - "TestMinimalChatLoop — the tracer proof: initialize honesty on the raw tee frame, D-03 back-leg wire order, chunks-before-stopReason with zero post-response updates, cancelled exactly once with no -32800 anywhere, clean second-prompt rejection — green under -race on both corpus rows"
  - "cancel-mid-turn corpus row extended with the front preamble (create/subscribe/setModel verbatim from setmodel-followup) + re-recorded golden asserting create→subscribe→setModel→send→stop in wire order — the corpus row the cancel leg rides"
  - "zcode.Conn.SetOnRouterEvent — the router-event single slot now registers under a lock, so mid-traffic machine construction (one machine per session/new) cannot race the routing goroutine"
affects: [04-03, 04-04, 04-05, phase-5-chat-cycle, phase-6-auth]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 15124 # chars/4 over the realized diff (60497 diff chars, 1287 lines)
  tasks: 2
  commits: 5 # MEASURED: git rev-list --count 93983ae..HEAD

commits: 5
plan_head_before: 93983ae59e752a1672ce5b72a9eb0d7d50e536da
plan_head_after: 1b068187267ef5bdeb6163f193b99224f3f16bed

# Tech tracking
tech-stack:
  added: [] # zero new dependencies — acp-go-sdk v0.20.0 stays the module's sole require line
  patterns:
    - "Tracer discipline proven: the vertical slice (SDK client ⇄ tee ⇄ acpport ⇄ Front ⇄ conn ⇄ fake) exposes integration facts no unit test can — the -race data race on mid-traffic machine construction surfaced on the first GREEN run and was fixed at the conn layer"
    - "Closure-captured context ownership: no context.Context struct fields anywhere in the Front (containedctx house rule) — newTurnCtx func field derives front-owned turn contexts; the tap-facing turn state is ONE bound emitter closure (sink + session id + ctx) published under the slot lock"
    - "Back-leg observer as a test synchronization primitive: the stage's own conn.AddObserver fires AFTER the front's tap and machine (registration order), so turn.started on that channel proves the machine is classifying a running turn and the front-wide slot is held — the deterministic second-prompt-while-active trigger"

key-files:
  created:
    - internal/acpfront/front.go
    - internal/acpfront/stream.go
    - internal/acpfront/turnctx.go
    - internal/acpfront/loop_test.go
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-06-red-cancel-corpus.json
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-06-red-tracer.json
  modified:
    - internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json
    - internal/zcode/fake/testdata/golden/cancel-mid-turn.golden
    - internal/zcode/fake/golden_test.go
    - internal/zcode/conn.go
    - internal/turn/machine.go
    - .golangci.yml

key-decisions:
  - "The turn slot is FRONT-WIDE, not per-session: the app-server event stream routes by turnId only (dialect §5), so one active turn across the whole front is the delta-routing precondition — the second prompt anywhere fails cleanly with the invalid-request guard (exact-class sharpening is plan 04-05's RED target)"
  - "Machine errors map to NewInternal (-32603) this plan per the plan text — the ErrTurnActive → invalid-request sharpening is deliberately left for plan 04-05's RED to flip"
  - "The Front's turn state is one emitter closure, not a context/sink/id field triple: contexts never live in struct fields (the conn.go containedctx precedent), and the emitter binds session id + turn ctx + sink so the tap makes one call per delta"
  - "The cancel corpus preamble copies the setmodel-followup rules verbatim (create + subscribe + setModel, same locked synthetic selection) so both corpus rows byte-lock the identical front-tier preamble — the tracer's Front config must match zai-api/GLM-5.3-Flash/level high or the golden fails"
  - "OnRouterEvent stays a single slot (plan 04-01 design) but its reads are now RWMutex-guarded with a SetOnRouterEvent setter — the minimal fix that lets per-session machines register mid-exchange without changing the CORE-06 revisit path"

patterns-established:
  - "Front-tier wire asserts read agent→client frames at tee dir \"in\" — the harness tee rides the CLIENT side (out = client requests, in = agent responses/updates); the fake transcript keeps the host's directions (in = host→fake requests). Documented in loop_test.go's const block for plans 04-03..04-05"
  - "Test decode structs carry the dialect's camelCase json tags verbatim (tagliatelle relaxed for test files by a documented exclusion) while production decode structs stay snake-safe via map digs (createSessionID follows machine.go's paramsString pattern)"

requirements-completed: [CORE-01, CORE-02, CORE-03, CORE-04, CORE-05] # copied verbatim from plan frontmatter; delivery is shared across plans 04-02..04-06 — requirements.ready-ids returned [], so NOTHING is marked complete by this plan (correct: 04-03/04-04/04-05 have no SUMMARYs yet)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "cancel-mid-turn corpus row extended with the front preamble (create/subscribe/setModel rules verbatim from setmodel-followup) and driveCancelMidTurn driving them; re-recorded golden byte-asserts create→subscribe→setModel→send→stop in wire order with every prior send/stop frame preserved; corpus stays fail-loud (strict loader, driver registered)"
    requirement: CORE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/cancel-mid-turn (real zcode.Conn over io.Pipe, -race; go test -race -count=1 -run TestGolden ./internal/zcode/fake/)"
        status: pass
    human_judgment: false
  - id: D2
    description: "acpfront.Front — the D-02 honest initialize report (protocolVersion 1 via the adapter, agentInfo zcode-acp, both authMethods agent-variant without a type field, no loadSession/fs/terminal anywhere on the raw response frame, promptCapabilities empty), asserted on the recorded tee frame"
    requirement: CORE-01
    verification:
      - kind: integration
        ref: "internal/acpfront/loop_test.go#TestMinimalChatLoop/happy_turn (assertInitializeReport on the raw tee line, -race)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The D-03 session/new contract on the back leg: bare create (workspaceKey = cwd, no model object), subscribe desktop-continuous, setModel follow-up carrying model.options.reasoningLevel + persistAsWorkspaceLastUsed:false — wire order asserted on the fake transcript; mcpServers accepted with a non-empty list and ignored"
    requirement: CORE-02
    verification:
      - kind: integration
        ref: "internal/acpfront/loop_test.go#TestMinimalChatLoop/happy_turn (assertBackLegNewSessionSequence on the fake transcript, -race)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The D-04 streaming contract: prompt(\"hi\") streams exactly two agent_message_chunk frames (\"Hello\", \" world\") BEFORE the prompt response frame on the ACP tee, stopReason end_turn, zero update frames after the response, zero thought chunks (nothing synthesized from a text-only corpus) — the delta tap maps part.delta field:text only"
    requirement: CORE-03
    verification:
      - kind: integration
        ref: "internal/acpfront/loop_test.go#TestMinimalChatLoop/happy_turn (assertHappyPromptStream: updates collector + tee wire order, -race)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The D-05 cancel contract over the extended corpus: cancel mid-turn resolves exactly ONE prompt response {stopReason:cancelled} as a SUCCESS (no -32800 anywhere on the tee), a second prompt while the turn is active fails cleanly (an error frame, and exactly ONE session/send reaches the back leg)"
    requirement: CORE-05
    verification:
      - kind: integration
        ref: "internal/acpfront/loop_test.go#TestMinimalChatLoop/cancel_mid-turn (assertCancelOutcome + assertCancelBackLeg, -race)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The D-01 seam lock holds through the front: acpport owns every SDK import; acpfront/core never see the SDK in production code (TestSDKImportBoundary stays green); both corpus rows replay byte-deterministically through the real conn (TestGolden green across the corpus)"
    requirement: CORE-01
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestSDKImportBoundary + internal/zcode/fake/golden_test.go#TestGolden (repo walk + corpus replay, -race)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Zero regressions: full gate green — build, vet, lint 0 issues (after the two documented exclusions), full -race internal suite, experiments suite; tracer re-verified end-to-end (the tracer feedback gate) with -count=2"
    verification:
      - kind: integration
        ref: "mise run ci (build + vet + lint + go test -race -count=1 ./... + node --test experiments/)"
        status: pass
    human_judgment: false

# Metrics
duration: 28min
completed: 2026-11-20
status: complete
---

# Phase 4 Plan 06: Minimal Chat Loop Tracer Summary

**The first vertical integration proven end-to-end under -race: a real SDK ACP client runs initialize → session/new (create + subscribe + setModel on the back wire) → streamed prompt (live text chunks before the stopReason) → mid-turn cancel (cancelled exactly once, never -32800) through acpport → the new acpfront Front → the Phase-3 machine → the fake app-server**

## Performance

- **Duration:** 28 min
- **Started:** 2026-11-20T00:26:32Z
- **Completed:** 2026-11-20T00:55Z
- **Tasks:** 2 (Task 2 via RED→GREEN TDD with classified evidence)
- **Files:** 6 created / 6 modified

## Accomplishments
- `internal/acpfront.Front` lands as the `acpport.Handlers` implementation carrying the full minimal loop: the honest D-02 initialize report (probe-verbatim auth methods, version injectable via `WithVersion`, default `0.0.0-dev`), the D-03 session/new contract (bare create with `workspaceKey = cwd`, `session/subscribe` desktop-continuous BEFORE any turn, setModel follow-up only when configured with explicit `persistAsWorkspaceLastUsed:false`), the D-04 streamed prompt over the Phase-3 machine, and the D-05 exactly-once cancel
- The delta tap (stream.go) registers on the conn's ordered fan-out BEFORE any machine exists — flush-before-stopReason by registration order; one `part.delta` (field text) envelope becomes exactly ONE message chunk in arrival order; reasoning/input/output are silently skipped with named plan-04-04/Phase-5 comments; upserts are never emitted
- turnctx.go owns the turn context the way RESEARCH Pattern 5 prescribes: front-owned (never the SDK request ctx), cancelled resolves as a SUCCESS stopReason, cancel fan-in goroutine triggers exactly one idempotent `machine.Cancel`, and the front-wide turn slot (single-active-turn invariant) rejects a second prompt cleanly
- `TestMinimalChatLoop` is green under -race on both corpus rows, asserting on RECORDED transcripts at both legs: initialize honesty on the raw tee frame, the D-03 back-leg wire order, chunks-before-response with zero post-response updates, cancelled exactly once with no -32800 anywhere, and the second-prompt rejection leaving exactly one `session/send` on the wire
- The cancel-mid-turn corpus row now carries the front preamble (create/subscribe/setModel verbatim from setmodel-followup) and its re-recorded golden byte-asserts create→subscribe→setModel→send→stop in wire order — every prior send/stop frame preserved, all ten pre-existing goldens untouched
- The tracer earned its keep: the first GREEN run tripped -race on a real integration bug (mid-traffic machine construction racing the conn's router-event slot) — fixed at the conn layer with a lock-guarded slot and setter, semantics unchanged
- `mise run ci` fully green: build, vet, lint 0 issues, full -race internal suite, experiments

## Task Commits

Each task committed atomically (TDD: RED commit precedes GREEN per feature):

1. **Task 1 (RED): extend cancel-mid-turn corpus with the front preamble** - `350a86a` (test)
2. **Task 1 (GREEN): record cancel-mid-turn golden with the front preamble** - `c63399a` (feat)
3. **Task 2 (RED): failing TestMinimalChatLoop against compiling acpfront skeletons** - `dc0f754` (test)
4. **Task 2 (fix): guard the conn router-event slot for mid-traffic machine registration** - `b312fe9` (fix)
5. **Task 2 (GREEN): implement acpfront Front — the minimal chat loop end-to-end** - `1b06818` (feat)

**Plan metadata:** committed in the close-out docs commit (this file + STATE/ROADMAP/REQUIREMENTS).

## TDD Record

Two RED→GREEN cycles, each with a classifier-verified evidence record (`evidence/04-06-red-*.json`, both `RED_EVIDENCE_OK`, go-test adapter):

- **RED 1 — cancel corpus preamble** (`evidence/04-06-red-cancel-corpus.json`): TestGolden/cancel-mid-turn failed on "golden drift" (the byte-compare against the now-stale send/stop-only fixture) after the scenario + driver gained the preamble; the exchange replayed clean and all ten pre-existing goldens stayed green. **Semantic assessment:** genuine RED on the planned assertion — the GREEN step re-records the fixture with -update.
- **RED 2 — the tracer** (`evidence/04-06-red-tracer.json`): the compiling skeletons failed both stages on planned assertions — the happy stage walked the earlier honesty asserts green (absent loadSession/fs/terminal, empty promptCapabilities, protocolVersion 1 — the adapter hardcodes it) then failed on `agentInfo.name = ""`; the cancel stage failed on the stub's empty session id. Zero hangs. **Semantic assessment:** genuine RED per the plan's compile-first stub strategy; one test-authoring fix (the tee's direction convention — this harness's tee rides the client side, so agent→client frames are dir "in") happened BEFORE the evidence was captured, so the recorded RED lands only on planned assertions.

## Files Created/Modified
- `internal/acpfront/front.go` — Front (Handlers impl): Initialize honest report, NewSession D-03 sequence + registry registration, Prompt turn slot + front-owned ctx + machine.Run + stop-reason mapping, Cancel via machine, createSessionID (map-dig, snake-safe)
- `internal/acpfront/stream.go` — the delta tap: part.delta field literals, text→message-chunk emission through the slot's emitter, reasoning/input/output named skips
- `internal/acpfront/turnctx.go` — the ownership contract doc + acquireTurn (slot guard, emitter closure, release)
- `internal/acpfront/loop_test.go` — TestMinimalChatLoop (two stages) + the assert helpers (initialize frame, auth methods, session open, prompt stream, tee order, back-leg D-03 sequence, cancel outcome/back-leg)
- `internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json` + `testdata/golden/cancel-mid-turn.golden` — the extended corpus row
- `internal/zcode/fake/golden_test.go` — driveCancelMidTurn drives the preamble (shape-mirrors driveSetModelFollowup)
- `internal/zcode/conn.go` — routerMu-guarded OnRouterEvent reads + SetOnRouterEvent setter (single slot unchanged)
- `internal/turn/machine.go` — NewMachine registers via conn.SetOnRouterEvent
- `.golangci.yml` — two documented exclusions: acpfront returns acpport's coded errors unwrapped (the seam's toReqErr semantics), wire-shape tests decode dialect camelCase verbatim

## Decisions Made
- The turn slot is FRONT-WIDE, not per-session — the app-server event stream routes by turnId only (dialect §5), so single-active-turn-per-front is the delta-routing precondition the truths name; the guard answers invalid-request (exact class sharpened by 04-05)
- Machine errors map to NewInternal (-32603) this plan, exactly per the plan text — the ErrTurnActive → invalid-request sharpening stays plan 04-05's RED target
- Contexts never live in struct fields (the conn.go containedctx precedent): `newTurnCtx` is a func field over a closure-captured base context, and the tap-facing turn state is ONE emitter closure binding sink + session id + turn ctx
- The cancel-corpus preamble copies the setmodel-followup rules verbatim with the same locked synthetic selection — both corpus rows byte-lock the identical preamble, so the tracer's Front config must be zai-api/GLM-5.3-Flash/level high
- OnRouterEvent stays single-slot (plan 04-01 design) but now registers under a lock — the minimal mid-traffic-safe fix that leaves the CORE-06 revisit path intact

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Data race: conn.OnRouterEvent assigned mid-traffic by per-session machine construction**
- **Found during:** Task 2 GREEN (the tracer's first full-loop run tripped -race)
- **Issue:** the front constructs one turn machine per `session/new` (the plan's own design), and `NewMachine` wrote the conn's single-slot `OnRouterEvent` while the create rule's runtime-preferences reverse exchange was concurrently reading it — the Phase-3 assign-before-traffic contract cannot hold once machines arrive mid-exchange
- **Fix:** reads of the slot now go through an RWMutex-guarded getter in `recordRouterEvent`; `NewMachine` registers via the new `conn.SetOnRouterEvent` setter; single-slot semantics unchanged, direct pre-traffic field assignment still supported for existing tests
- **Files modified:** internal/zcode/conn.go, internal/turn/machine.go
- **Verification:** full -race internal suite green (the race reproducible before, absent after)
- **Committed in:** b312fe9

**2. [Rule 3 - Blocking] Two lint-gate exclusions required for the new code's documented contracts**
- **Found during:** Task 2 GREEN (mise run ci lint stage: 43 issues in the new files)
- **Issue:** wrapcheck flags the front returning `acpport.NewInternal/NewInvalidRequest` unwrapped — but that IS the coded-error contract (the adapter maps `acpport.Error` onto the SDK's coded RequestError with the same toReqErr type-assert semantics as the existing acpport exclusion); tagliatelle flags the test decode structs carrying the dialect's camelCase wire keys — snake_case tags would never match the wire
- **Fix:** two documented exclusion rules in .golangci.yml (mirroring the plan-04-02 acpport exclusion pattern); all other lint findings (complexity splits, context-as-argument ordering, lll wrapping, exhaustive switch cases, containedctx closure redesign, funcorder placement) fixed in code
- **Files modified:** .golangci.yml, internal/acpfront/front.go, internal/acpfront/loop_test.go, internal/acpfront/turnctx.go, internal/zcode/conn.go
- **Verification:** mise run ci lint stage 0 issues
- **Committed in:** 1b06818

---

**Total deviations:** 2 auto-fixed (2 blocking) — none architectural, no Rule 4 decisions required. The race fix is the tracer doing its job: an integration fact invisible to every unit test, surfaced and fixed before plans 04-03..04-05 build on the slice.
**Impact on plan:** both fixes serve the plan's own gates (-race green, lint-clean CI). Prohibitions hold: every emitted chunk is traceable to one received part.delta envelope (the sink is a passthrough; the tap never synthesizes), and stdout stays protocol-exclusive (all logging is stderr slog with structural fields only — session ids and error strings, never params/content).

## Issues Encountered
- The harness tee's direction convention tripped the first RED run: the tee rides the CLIENT side of the composition, so agent→client frames (responses, session/update) record as dir "in" and client→agent requests as dir "out" — the first run's initialize assert matched the client's REQUEST frame (which carries `"fs":{}` inside clientCapabilities). Fixed before RED evidence was captured; the convention is now documented in loop_test.go's const block for plans 04-03..04-05.
- golangci-lint (strict v2.15.0, default:all) flagged 43 issues across the new files; all resolved — two via documented .golangci.yml exclusions (the coded-error contract and the wire-key test decodes), the rest in code (test-function complexity splits under gocognit/gocyclo, context-first helper signatures under revive, lll wrapping, an exhaustive stop-reason switch, the containedctx closure redesign, funcorder constructor placement).

## User Setup Required

None - no external service configuration required (fake-backed composition, zero processes, no quota cost).

## Next Phase Readiness
- Plans 04-03..04-05 sharpen the proven slice into named RED targets: 04-03 owns the cwd-absoluteness validation + Authenticate honest error + setModel payload lock, 04-04 the reasoning→thought mapping (the tap's named skip is its RED target), 04-05 the -32010 concurrency class + cmd wiring (the version var and supervisor lifecycle)
- The recorded ACP tee transcripts from TestMinimalChatLoop are the CORE-03/CORE-05 wire-order evidence those plans sharpen into named tests
- The tracer feedback gate passed (auto mode): the full verify battery re-run green end-to-end including `-count=2` flakiness, so expansion proceeds on a proven slice
- No blockers

---
*Phase: 04-acp-front-minimal-chat-loop*
*Completed: 2026-11-20*

## Self-Check: PASSED

All created files exist on disk; all 5 task commits (350a86a, c63399a, dc0f754, b312fe9, 1b06818) are ancestors of HEAD (1b06818). Commit count measured from the persisted plan ledger: 5 (93983ae..HEAD). `mise run ci` green at HEAD; the tracer feedback gate re-verified TestMinimalChatLoop + TestSDKImportBoundary + the full -race internal suite end-to-end.
