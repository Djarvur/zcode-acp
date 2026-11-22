---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 06
subsystem: testing
tags: [acp, conformance, fake-app-server, composed-harness, cancel, permission-bridge, race-detector]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: the 34-scenario fake corpus + TestGolden fail-loud drivers (the additive-property anchor)
  - phase: 04-acp-front-minimal-chat-loop
    provides: the FrontHarness composed harness (ComposeFront, TeeRW, RecordingClient) and the CORE-05 cancel contract shapes
  - phase: 05-interactions-full-chat-cycle-api-key
    provides: TestFullCycle's arc legs + leg helpers (mustInitialize family, fullCycleAuthenticate/OpenSession/Prompt, tee/transcript assert helpers), the permission bridge locks, the tool-lifecycle mapping locks
  - phase: 06-oauth-coding-plan-path-model-picker
    provides: the initialize options capability (06-04) the arc's leg-1 probe names
provides:
  - TestConformanceV1Arc — the ONE named, byte-locked v1 conformance suite the done-v1 bar names: initialize → authenticate → session/new → prompt streaming (tool calls + permission decision) → cancel, one harness, one continuous client story, green inside `mise run ci`
  - the in-file conformance-composition pattern: load a proven corpus row and append rules programmatically (the fake's later-rule takeover routes post-budget sends) — additive arcs with zero corpus/golden/shared-file edits
affects: [07-07 VM scenario (the scripted stdio arc mirrors this arc's leg order), 07-09 manual E2E (the named suite is the automated-tier evidence the user gate rides), ROADMAP done-v1 criterion 1 (maps to TestConformanceV1Arc by name)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 3279      # chars/4 over the realized diff (13114 diff chars)
  tasks: 2
  commits: 1         # MEASURED: git rev-list --count c193749..HEAD (task 2 was verification-only: no code changes to commit)
plan_head_before: c193749ec2f3abf854144d3bd72a6e6fc2704be0
plan_head_after: 89c9489a240a210eaa2c387142efbdeca8181550

# Tech tracking
tech-stack:
  added: []          # test-only file over the existing harness — no dependency change (T-07-SC held)
  patterns:
    - "conformance composition: mustLoadScenario + append(cancel-leg rules) inside the test file — the fake's rule-takeover semantics (a later on_request entry for the same method takes over once a times-capped rule's budget is spent) route the arc's second send"
    - "leg-level byte-locking: tee-frame probe substring constants per leg + proven leg helpers ridden whole (fullCyclePrompt family) + one transcript audit at arc end"

key-files:
  created:
    - internal/acpfront/conformance_test.go (TestConformanceV1Arc + the appended cancel-leg rules + per-leg asserts)
  modified: []

key-decisions:
  - "The arc rides the full-cycle corpus row for legs 1-4 and composes the cancel leg by APPENDING the proven cancel-mid-turn shapes (parkable second send + session/stop cancelled patch) programmatically inside conformance_test.go — the only way to satisfy ONE-harness + self-contained-file + untouched-corpus simultaneously, since no existing row carries both a featured completed turn and a cancellable second turn"
  - "TestConformanceV1Arc landed CONFIRM-GREEN at first run (the 05-08 precedent): recorded honestly with per-leg evidence — no synthetic RED, no assertion weakened to force one"
  - "Transcript-level locks (auth chain evidence, D-03 gate, permission reply, arc wire) run ONCE at arc end — the harness's Transcript accessor closes the back leg, so mid-arc transcript asserts would kill the story"

patterns-established:
  - "in-file scenario extension for composed arcs: append fake.RequestRule values to a loaded corpus row — additive-only arcs without touching testdata or golden drivers"
  - "arc cancel-leg assert shape: exactly one end_turn + exactly one cancelled response, ordered, exactly two stopReason frames total, no update after the cancelled response, blanket no--32800 scan (no pending ask at cancel time)"

requirements-completed: [QA-01]

coverage:
  - id: D1
    description: "The named conformance suite TestConformanceV1Arc: the full v1 arc (initialize → authenticate → session/new → prompt streaming with tool calls and a bridged ALLOW permission decision → cancel mid-turn) byte-locked on the tee and fake transcript, green under -race -count=3 and inside the full `mise run ci` gate with every existing suite untouched (TestGolden's 34 scenarios, TestFullCycle, the cancel rows)"
    requirement: QA-01
    verification:
      - kind: e2e
        ref: "internal/acpfront/conformance_test.go#TestConformanceV1Arc"
        status: pass
      - kind: other
        ref: "go test -race -count=1 ./... (whole module, exit 0)"
        status: pass
      - kind: other
        ref: "mise run ci (build, vet, lint, test, test-experiments — exit 0)"
        status: pass
      - kind: other
        ref: "go test -race -count=1 -run 'TestGolden' ./internal/zcode/fake/ (the D-04 additive-property anchor, exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 16min
completed: 2026-11-22
status: complete
---

# Phase 7 Plan 06: Conformance Suite Summary

**TestConformanceV1Arc — the named byte-locked v1 arc (initialize → session/new → prompt streaming → tool calls → permission decision → cancel) over the composed harness, green under -race -count=3 and inside `mise run ci` with every existing suite untouched**

## Performance

- **Duration:** 16 min
- **Started:** 2026-11-22T13:06:38Z
- **Completed:** 2026-11-22T13:22:30Z
- **Tasks:** 2
- **Files modified:** 1 (created)

## Accomplishments

- One named CI conformance suite, `TestConformanceV1Arc`, walks the whole v1 arc continuously over ONE harness in the real client's order — the single green checkmark the ROADMAP done-v1 success criterion 1 names (D-04 complete)
- Every arc leg asserts its load-bearing wire frames: initialize's honest capability reply (protocolVersion 1, the sessionCapabilities options opt-in, both authMethod ids), authenticate's `{}` success, session/new's D-03 create → subscribe → setModel transcript gate, the featured prompt turn (streamed delta, tc1 Edit tool_call/tool_call_update pair with title/kind/status, the bridged session/request_permission answered ALLOW, flush-before-end_turn), and the cancel leg (exactly-once cancelled resolution, no -32800 anywhere, updates precede the response)
- Additive property held (the CORE-05 cancel lock and the T-07-17 anti-greenwashing mitigations): zero skips, zero shortened legs, zero edits to any existing suite — TestGolden's 34 scenarios, TestFullCycle, the cancel rows and the phase-6 locks all green untouched
- CONFIRM-GREEN branch (the 05-08 precedent): the suite passed at first run and under -race -count=3; recorded honestly — no fabricated RED, no weakened assertion

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — TestConformanceV1Arc, the full arc composed and byte-locked** - `89c9489` (test)
2. **Task 2: Conformance suite integrated into the phase gate** — verification-only (no code changes expected and none produced): `go test -race -count=1 ./...` exit 0, `mise run ci` exit 0, TestGolden green untouched, `git status` shows only conformance_test.go from this plan. Nothing to commit.

## Files Created/Modified

- `internal/acpfront/conformance_test.go` (new, 304 lines) — TestConformanceV1Arc: the arc test, the in-file cancel-leg rule composition, the per-leg tee asserts, and the arc-end transcript audit

## Decisions Made

- **The cancel leg is composed programmatically onto the full-cycle row.** No existing corpus row carries both a featured completed turn (tools + permission + streaming + end_turn — full-cycle's `session/send` with `times: 1`) and a parkable, cancellable second turn with a session/stop handler. The suite loads the proven full-cycle row (`mustLoadScenario`) and appends the proven cancel-mid-turn shapes — a second `session/send` rule (running + `turn.started` on `turn_fix2`, no terminal event) and the `session/stop` rule (`{}` + the cancelled state patch) — entirely inside conformance_test.go. The fake's documented rule-takeover semantics ("a later on_request entry for the same method takes over once the budget is spent") route the arc's second send to the appended rule. This keeps all three plan constraints simultaneously satisfiable: ONE harness, self-contained file (edits no shared harness file), and the D-04 additive property (corpus, goldens, and golden drivers untouched — adding a scenario JSON would have failed TestGolden's fail-loud no-driver check).
- **Transcript-level locks run once, at the arc's end.** The harness's `Transcript(t)` accessor closes the back leg, so the auth-chain evidence, the D-03 session/new sequence, the permission reply's embedded response, and the arc wire (exactly two sends, one stop after the second) are asserted after the whole story — the fullcycle discipline.
- **Blanket no--32800 scan at the cancel leg.** The arc's scripted client answers ALLOW before turn 2, so no permission round-trip is pending at cancel time — the blanket both-direction scan (`assertNoRequestCancelled`) is the honest posture per its own doc.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] The arc's cancel-leg rules are composed programmatically — no single existing corpus row hosts the one-harness arc**
- **Found during:** Task 1 (corpus survey before implementation)
- **Issue:** The plan's task action assumes one scenario can carry legs 1-4 (full featured turn: streaming + tools + permission + end_turn) AND leg 5 (a second prompt parked mid-turn with a session/stop handler). Surveying all 34 rows: full-cycle is the only featured-turn row and its send is `times: 1` with no session/stop rule (a second send would get the fake's unknown-method `-32601`, and cancel would have no scripted outcome); the cancel-capable rows (cancel-mid-turn, complete-then-cancel, session-lifecycle) carry no tool/permission legs. Adding a new scenario JSON was not an option: it would fail TestGolden's fail-loud no-driver check or require editing `internal/zcode/fake/golden_test.go` — a shared harness file outside the plan's pinned file scope and a D-04 violation.
- **Fix:** Load the proven full-cycle row and append the two cancel-leg rules (copied from the proven cancel-mid-turn/complete-then-cancel shapes, capture-verbatim compact payloads) in `newConformanceFront`/`conformanceCancelLegRules` inside conformance_test.go — within the pinned file scope, zero corpus/shared-file edits, all legs byte-locked as the plan requires.
- **Files modified:** internal/acpfront/conformance_test.go only
- **Verification:** TestConformanceV1Arc green under -race -count=3; every existing suite green untouched (`go test -race -count=1 ./...`, `mise run ci`, TestGolden)
- **Committed in:** 89c9489

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** The fix is the minimal in-scope composition that makes the plan's arc executable at all; it changes no assertion, no corpus byte, and no shared file. No scope creep.

## Issues Encountered

- None beyond the deviation above. One compile-time type fix during authoring (`fake.DirOut` is a typed constant — `string(fake.DirOut)` for the `transcriptIndexes` call, matching the existing fullcycle usage) never reached a commit boundary.

## Authentication Gates

None — the plan touches no real credentials or external services (the auth leg rides the synthetic fixture key over the in-process fake).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- D-04 complete: the ROADMAP Phase-7 success criterion 1 maps to `TestConformanceV1Arc` by name; `mise run ci` carries it inside the `[tasks.test]` step (`go test -race -count=1 ./...`)
- The in-file rule-composition pattern is available to 07-07's scripted VM arc (which mirrors this arc's leg order against the real binary) and to any future arc needing shapes no corpus row carries
- Known flake surface: none observed — the composition is the same in-process harness 05-08 proved stable (T-07-18), and the cancel leg reuses the cancel-mid-turn row's deterministic sequencing (turn.started sync before cancel)

## Self-Check: PASSED

- FOUND: internal/acpfront/conformance_test.go (304 lines, on disk)
- FOUND: 89c9489 — `git merge-base --is-ancestor` confirms it is HEAD's ancestor
- Measured: `git rev-list --count c193749..HEAD` = 1 (task 2 is verification-only by design — no uncommitted changes: `git status` clean for internal/)
