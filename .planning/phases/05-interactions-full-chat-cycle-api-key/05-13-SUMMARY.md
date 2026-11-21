---
phase: 05-interactions-full-chat-cycle-api-key
plan: 13
subsystem: testing
tags: [turn-latch, golden-corpus, fake-app-server, realdesktop-guard, protocol-docs]

requires:
  - phase: 05-interactions-full-chat-cycle-api-key (05-12)
    provides: EXP-APIKEY-CYCLE-02.jsonl committed scrubbed capture (the diagnosis source); the 05-UAT.md turn-sequencing gap; close-out baseline 0dfce4f
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (03-03/03-05)
    provides: strict-decode scenario loader + golden corpus discipline; watcher quiet-floor defaults (3x2500 ms)
provides:
  - RequestRule.LatchMs + Scenario.LatchRefusal schema fields and the Serve-path latch gate (refusal verbatim, zero rule budget consumed)
  - lingering-latch-32010 corpus row + byte-golden (defect repro + drained recovery through a real Conn)
  - turnWithLatchDrain + askRecorder tap + UTC-stamped leg logging in the guarded real-tier test
  - dialect §5 turn-latch addendum with per-fact capture-line provenance
  - mechanically proven zero production change vs baseline 0dfce4f (internal/ minus internal/zcode/fake)
affects: [05-14 (REAL RUN attempt 3), phase-7 hardening (QA-01 corpus acceptance)]

tech-stack:
  added: []
  patterns: [latch-window scripting (latch_ms + latch_refusal), ask-silence settle + bounded invalid-request-class retry, structural interaction-ask tap]

key-files:
  created:
    - internal/zcode/fake/testdata/scenarios/lingering-latch-32010.json
    - internal/zcode/fake/testdata/golden/lingering-latch-32010.golden
  modified:
    - internal/zcode/fake/scenario.go
    - internal/zcode/fake/fake.go
    - internal/zcode/fake/scenario_test.go
    - internal/zcode/fake/golden_test.go
    - test/realdesktop/apicycle_test.go
    - docs/protocol/app-server-dialect.md

key-decisions:
  - "Latch refusal consumes NO rule budget — the drained retry finds the next rule fresh (the corpus row's honesty lock)"
  - "V-EXP citation gate-compliance: the §5 addendum tags [V-EXP EXP-APIKEY-CYCLE-01] (the header-matching test-locked id) while naming EXP-APIKEY-CYCLE-02.jsonl with line pointers — a literal [V-EXP EXP-APIKEY-CYCLE-02] tag would fail check-docs (f) capture-header continuity (the 05-12 monotonicity decision)"
  - "Ask tap landed as tapInteractionAsks wrapping the already-assigned conn.OnRequest (wireCycleResponder signature untouched) — readbackdiag_test.go shares that helper and sits outside plan scope"
  - "REQUIREMENTS.md untouched per the plan contract: AUTH-01 stays [ ] until a passing real run (the orchestrator's post-verification act, never 05-13's)"

patterns-established:
  - "Latch scripting: a rule's latch_ms arms a post-reply refusal window answered from the scenario's latch_refusal; only session/send is modeled (the desktop latches prompts, not host calls)"
  - "Drain discipline: settle on interaction-ask silence, retry ONLY the acpport invalid-request class (code-matched via errors.As, never message text), budget exhaustion = hypothesis-(b) evidence surfaced loudly"

requirements-completed: []  # deliberate: AUTH-01 stays [ ] per the plan contract (the flip act is a passing real run; 05-13 is zero-spend)

coverage:
  - id: D1
    description: "Fake-tier latch gate + lingering-latch-32010 corpus row byte-golden: refusal inside the window with zero budget consumed, acceptance after release"
    verification:
      - kind: unit
        ref: "internal/zcode/fake/golden_test.go#TestGolden/lingering-latch-32010"
        status: pass
      - kind: unit
        ref: "internal/zcode/fake/scenario_test.go#TestScenarioLatchRequiresRefusal (with the negative-latch, empty-refusal, and accepts-pair cases)"
        status: pass
      - kind: unit
        ref: "command: go test -race -count=1 ./internal/zcode/fake/ (full suite, zero corpus regressions)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Guarded real-tier test drains the latch between turns (ask-silence settle + bounded invalid-request retry + UTC-stamped evidence) with the double guard intact"
    requirement: TOOL-01
    verification:
      - kind: unit
        ref: "command: go build -tags realdesktop ./... && go vet -tags realdesktop ./..."
        status: pass
      - kind: unit
        ref: "command: go test -count=1 ./test/realdesktop/ ([no test files] — the tag guard holds)"
        status: pass
      - kind: unit
        ref: "command: go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/ (skips in ~3ms without ZCODE_ACP_REAL_E2E=1 — honest skip, no execution)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Dialect §5 turn-latch addendum with per-fact capture-line provenance (floor-expiry settle verdict, post-completion -32010 refusal, cipher asks live until 52.462, messages-only polls, UNOBSERVED terminal surface)"
    verification:
      - kind: other
        ref: "command: bash experiments/check-docs.sh (structure + provenance-continuity gate green)"
        status: pass
      - kind: other
        ref: "line citations mechanically verified against EXP-APIKEY-CYCLE-02.jsonl (:34-:52, :114-:119) by direct reads during execution"
        status: pass
    human_judgment: false
  - id: D4
    description: "Zero production change mechanically proven and a fully green tree handed to 05-14's human checkpoint"
    verification:
      - kind: other
        ref: "command: git diff --quiet 0dfce4f..HEAD -- internal/ ':(exclude)internal/zcode/fake' (prints internal-clean)"
        status: pass
      - kind: other
        ref: "command: mise run ci (build/vet/lint/test/test-experiments all green on the finished tree)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The drain's real-desktop efficacy and the (a)/(b) discrimination evidence — zero-spend by construction; produced only by 05-14's human-gated REAL RUN attempt 3"
    verification: []
    human_judgment: true
    rationale: "The plan is zero-spend by contract (D-05): the real-tier drain never executes in CI, and its live efficacy plus the latch-lag vs premature-end_turn discrimination is exactly what 05-14's human-gated attempt-3 run exists to produce."

actuals:
  tokens: 8490
  tasks: 3
  commits: 5
  plan_head_before: 8bee5934f19605a8fca60574a991528ca2aba317
  plan_head_after: 34df7a672382457b7e9d069b820282233a74331a

duration: 16min
completed: 2026-11-21
status: complete
---

# Phase 5 Plan 13: Turn-Latch Gap Closure (Cycle 3) Summary

**The desktop's lingering turn latch is now a CI-reproducible byte-golden corpus row (refusal inside the window, budget intact; acceptance after release), the guarded real-tier test drains the latch between turns, and the attempt-2 wire truth is documented with capture-line provenance — all zero-spend, zero production change.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-11-21T10:51:35Z
- **Completed:** 2026-11-21T11:07:59Z
- **Tasks:** 3
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- Fake engine turn-latch extension (RED `0590778` → GREEN `8f59510`): `latch_ms`/`latch_refusal` schema fields with fail-loud validation, and a Serve-path gate that answers a `session/send` arriving inside an armed window with the capture-verbatim -32010 refusal WITHOUT consuming rule budget
- `lingering-latch-32010` corpus row: send-1 acked with EARLY completion surfaces (running→idle patches), immediate send-2 refused -32010, drained send-3 accepted after release — the desktop's EXP-APIKEY-CYCLE-02 behavior, byte-golden-locked and replayed through a real Conn
- The guarded test's latch drain (`7c10d37`): `tapInteractionAsks` records interaction/* asks structurally (method + UTC timestamp ONLY — never params, T-5-33); `turnWithLatchDrain` settles on ask silence (3 s quiet, 30 s cap) then retries ONLY the acpport invalid-request class (2 s doubling backoff, 15 s cap, 120 s / 8 attempts) — any other error fails the leg loud (T-5-32); budget exhaustion returns the hypothesis-(b) sentinel for the attempt-3 verdict; a UTC-stamped turn-1 resolution anchor correlates with the capture tee
- Dialect §5 turn-latch addendum (`34df7a6`): the attempt-2 wire truth with per-fact line provenance — a COMPLETED real turn surfaced NO terminal envelope within the quiet-floor window; the -32010 refusal landed ~7.5 s post-ack with cipher asks live until 52.462; polls carried messages-only; the true terminal-surface shape is honestly UNOBSERVED
- The plan's own gates green on the finished tree: check-docs, the production-diff proof (`internal-clean` vs 0dfce4f with the fake tier excluded), and `mise run ci` end-to-end

## Task Commits

Each task was committed atomically (TDD sequence for Task 1):

1. **Task 1 RED: failing lingering-latch corpus row (latch schema RED)** - `0590778` (test)
2. **Task 1 GREEN: fake engine turn latch + corpus golden** - `8f59510` (feat)
3. **Task 2: guarded test latch drain — ask-tap settle + bounded invalid-request retry** - `7c10d37` (feat)
4. **Task 3 lint fix: fake.go dupword + gofmt** - `e0dc0a6` (style)
5. **Task 3: dialect §5 turn-latch addendum** - `34df7a6` (docs)

_No REFACTOR commit: the GREEN implementation was already minimal (no cleanup needed)._

## Files Created/Modified

- `internal/zcode/fake/testdata/scenarios/lingering-latch-32010.json` - the lingering-latch corpus row (preamble copied verbatim from send-refused-32010; early-completion emits from completion-idle-patch; latch_ms 2000 + capture-verbatim latch_refusal)
- `internal/zcode/fake/testdata/golden/lingering-latch-32010.golden` - the byte-golden transcript (20 frames, one -32010)
- `internal/zcode/fake/scenario.go` - RequestRule.LatchMs + Scenario.LatchRefusal fields; validate() fail-loud (missing refusal pairing, negative latch_ms, empty refusal payload)
- `internal/zcode/fake/fake.go` - the latchMu/latchUntil gate in handleRequest (send-only, budget intact) + armLatch after the completion rule's reply write
- `internal/zcode/fake/scenario_test.go` - four TestScenario* latch loader-validation cases
- `internal/zcode/fake/golden_test.go` - driveLingeringLatch32010 registered (driver-less entries fail loud) + lingeringLatchSettle knob
- `test/realdesktop/apicycle_test.go` - ask tap, drain helper, toolTurnLeg wiring, turn-1 UTC correlation anchor, named drain constants
- `docs/protocol/app-server-dialect.md` - §5 subsection "The one-turn-at-a-time latch — post-completion window, 3.18.2"

## Decisions Made

- **Latch refusal consumes no rule budget** — the corpus row's whole point is that the drained retry finds the next rule fresh; modeled exactly.
- **V-EXP citation gate-compliance** — the §5 addendum cites `[V-EXP EXP-APIKEY-CYCLE-01]` (the id BOTH capture files' headers carry, per the 05-12 capture-monotonicity decision) while every fact names `EXP-APIKEY-CYCLE-02.jsonl` with line pointers; a literal CYCLE-02 tag fails check-docs check (f).
- **Ask tap as a separate wrapper** — `tapInteractionAsks` wraps the already-assigned `conn.OnRequest` immediately after `wireCycleResponder`, still before any traffic; `wireCycleResponder`'s signature stays untouched because `readbackdiag_test.go` (same guarded package, outside plan scope) calls it.
- **REQUIREMENTS.md untouched** — AUTH-01 stays `[ ]` per the plan contract; the flip act is the orchestrator's, only after a passing real run.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Ask tap shape adjusted to keep readbackdiag_test.go compiling**
- **Found during:** Task 2 (tagged vet failed: `not enough arguments in call to wireCycleResponder` from readbackdiag_test.go:146)
- **Issue:** The plan's "extend wireCycleResponder" (new `t` param + return value) broke a sibling file in the same guarded package that the plan's scope rule forbids touching (`git diff --stat` must name ONLY apicycle_test.go)
- **Fix:** Kept `wireCycleResponder` byte-identical; added `tapInteractionAsks(t, conn)` which wraps the ALREADY-assigned `conn.OnRequest` with the recorder immediately after wiring — still before any traffic (the Conn hook contract). Same observable behavior the plan specified: interaction/* recorded + logged structurally, responder chain fall-through untouched.
- **Files modified:** test/realdesktop/apicycle_test.go only
- **Verification:** tagged build + vet green; git diff --stat names ONLY apicycle_test.go
- **Committed in:** 7c10d37

**2. [Rule 3 - Blocking] §5 V-EXP tag form made gate-compliant**
- **Found during:** Task 3 (check-docs check (f) would fail a literal `[V-EXP EXP-APIKEY-CYCLE-02]` tag: the capture's first line carries `"capture":"EXP-APIKEY-CYCLE-01"` — the test-locked id the 05-12 monotonicity decision deliberately preserved)
- **Issue:** The plan's literal "V-EXP EXP-APIKEY-CYCLE-02 labels" contradicted the plan's own verify chain (`bash experiments/check-docs.sh` must be green)
- **Fix:** The addendum tags `[V-EXP EXP-APIKEY-CYCLE-01]` (resolves against both capture files with matching headers) and every fact cites `EXP-APIKEY-CYCLE-02.jsonl` by name with exact line pointers — the plan's provenance intent (attempt-2 capture, per-fact lines) fully preserved
- **Files modified:** docs/protocol/app-server-dialect.md only
- **Verification:** bash experiments/check-docs.sh green on the committed doc
- **Committed in:** 34df7a6

**3. [Rule 1 - Bug] Lint findings in Task 1's fake.go fixed**
- **Found during:** Task 3's `mise run ci` gate
- **Issue:** dupword ("until until" in the armLatch comment) + gofmt (server struct field alignment after adding latchMu/latchUntil)
- **Fix:** Comment reworded; gofmt -w applied
- **Files modified:** internal/zcode/fake/fake.go
- **Verification:** mise run ci green end-to-end
- **Committed in:** e0dc0a6

**4. [Plan-text precision] turnWithLatchDrain carries an `asks *askRecorder` parameter**
- **Found during:** Task 2
- **Issue:** The plan's 6-parameter prose shape has no route to the recorder's `lastAskAt` accessor
- **Fix:** The helper takes `(t, ctx, front, asks, sid, prompt, sink)` — the recorder is an explicit parameter; behavior identical to the plan's specification
- **Files modified:** test/realdesktop/apicycle_test.go
- **Committed in:** 7c10d37

---

**Total deviations:** 3 auto-fixed (2 blocking-scope, 1 lint) + 1 plan-text precision.
**Impact on plan:** All fixes necessary to satisfy the plan's own gates (scope check, check-docs, mise ci). No scope creep — every change stayed inside the plan's file list.

## Issues Encountered

- First `mise run ci` failed on the two fake.go lint findings (documented as deviation 3) — fixed and re-run green.
- The RED-phase loader-validation tests were authored so the package still compiles before the engine fields exist (the accepts-case asserts only successful loading in RED; field assertions added with GREEN) — keeps the RED evidence valid (no build_failure).

## Known Stubs

None — no placeholder/TODO/stub patterns in any file this plan created or modified.

## User Setup Required

None — no external service configuration required (zero-spend plan; no REAL RUN executed).

## Next Phase Readiness

- The tree is fully green (check-docs + internal-clean + `mise run ci`), ready for 05-14's human-gated REAL RUN attempt 3 with the drain in place
- The drain is correct under BOTH open hypotheses (a) latch-lag / (b) premature end_turn — attempt 3's UTC-stamped resolution anchor + ask tap correlate 1:1 with the capture tee for the verdict
- AUTH-01 stays `[ ]`; REQUIREMENTS.md untouched by this plan per contract
- A persistent invalid-request refusal past the drain budget (120 s / 8 attempts) surfaces loudly as hypothesis-(b) evidence — the honest 05-14 input

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-21*
