---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 03
subsystem: testing
tags: [fake-app-server, golden-transcripts, scenario-corpus, fail-loud, tdd, d-05, tracked-no-answer, eof-signature, determinism]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-01)
    provides: the fake scripting core (Serve, Scenario schema, golden CompareGolden) and the happy-turn golden — the tracer slice this plan hardens and expands
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-02)
    provides: the D-05 router state the corpus encodes — credential-cipher tracked-no-answer (zero reply bytes), runtime-preferences answered with a recorded deadline, unknown host methods -32601 naming the method
  - phase: 01-research-protocol-documentation
    provides: docs/protocol/app-server-dialect.md wire values (§3 server→host shapes, §5 signal table, §6 error codes) and the EXP capture anchors
provides:
  - internal/zcode/fake/scenario.go hardened loader: strict decode (unknown schema keys are field-naming rejections), exactly-one emit/reverse, dialect §6 code required on error replies, termination contract (eof_drain_ms required when on_request is empty), times >= 0 validation
  - RequestRule.Times match budget — a spent rule falls through to the next same-method rule (the declarative -32010 double-send shape)
  - Closed-stream write honesty in fake.Serve: a frame whose write fails at the host stream close is recorded as an unanswered note on a clean run, never a run error nor a silent drop (dialect §1 EOF facts)
  - The 10-scenario golden corpus under testdata/scenarios/ + testdata/golden/ (rows 2-10 authored here) — every scenario replays byte-deterministically through a real zcode.Conn over io.Pipe under -race
  - golden_test.go: table-driven TestGolden (driver-less corpus entries fail loud) + TestEOFSignature (zero recorded frames is a reported FAILURE, never an empty success)
affects: [03-06 cmd/fake-app-server, 03-07 supervisor spawn, 03-08 turn completion proofs, 03-09 E2E + phase gate, QA-01 conformance suite (phase 7)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 15000   # ~60000 chars over this plan's diff (26 files, 1227 insertions / 93 deletions)
  tasks: 2
  commits: 4     # MEASURED from the plan ledger (d712cf0..17bf12e): 3 are this plan's own commits (1 RED + 2 feat); the fourth is the parallel phase-02 close-out stream sharing the sequential branch

commits: 4
plan_head_before: d712cf09a3b78bf4f177d1a669d06c69ea51e1ed
plan_head_after: 17bf12ef9abe8cad95b7a6b6653dbeca781157fc

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: tidy-diff green, go.mod untouched)
  patterns:
    - "Fail-loud loader validation: strict decode makes a typo'd schema key a field-naming rejection; invalid fixtures stay in test temp dirs, never under testdata/"
    - "Budget-matched rules (times): first-match-only rule tables gain an occurrence budget so a later same-method rule takes over — declarative error sequencing"
    - "Record-honesty at the write boundary: frames lost to the drain-window shutdown are recorded as unanswered notes on a clean run (dialect §1: a request issued just before stdin EOF is never answered)"
    - "Bounded-quiet-window sync for fire-and-forget bypass frames (stopSettle 100 ms): wire-level notification signals plus a settle window, never wall-clock races"
    - "Golden corpus discipline: every scenario file must have a registered replay driver (fail loud), fixed synthetic ids, and a golden recorded via -update that must reproduce byte-identically on clean runs"

key-files:
  created:
    - internal/zcode/fake/scenario_test.go
    - internal/zcode/fake/golden_test.go
    - internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json
    - internal/zcode/fake/testdata/scenarios/error-32010.json
    - internal/zcode/fake/testdata/scenarios/unknown-host-method.json
    - internal/zcode/fake/testdata/scenarios/runtime-preferences.json
    - internal/zcode/fake/testdata/scenarios/completion-idle-patch.json
    - internal/zcode/fake/testdata/scenarios/completion-quiet-window.json
    - internal/zcode/fake/testdata/scenarios/eof-drain.json
    - internal/zcode/fake/testdata/scenarios/envelope-negatives.json
    - internal/zcode/fake/testdata/scenarios/credential-cipher-scripted.json
    - internal/zcode/fake/testdata/golden/ (9 new goldens; happy-turn re-recorded byte-identical)
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-03-red-scenario-loader-validation.json
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-03-red-times-budget.json
  modified:
    - internal/zcode/fake/scenario.go
    - internal/zcode/fake/fake.go

key-decisions:
  - "Strict decode (DisallowUnknownFields + single-document check) over selective checks: ANY unknown schema key anywhere in a scenario is a field-naming rejection — a typo'd key silently ignored would change the scripted wire behavior the corpus exists to pin"
  - "Termination contract reads the plan literally: an EMPTY on_request combined with a MISSING eof_drain_ms is the rejection (a scenario must declare how it terminates); an empty table WITH a declared drain is valid (answers everything unknown_method and ends at EOF)"
  - "RequestRule.Times budget instead of reply sequencing: the fake's table matches by method; the -32010 row needs the SECOND send answered differently, so a spent rule falls through to the next same-method rule (matched counts under ruleMu — handler goroutines are concurrent)"
  - "eof-drain drives conn.Stop as the just-before-EOF request: the bypass lane is the only conn API that writes a request-shaped frame without waiting for its reply; the rule's delay_ms (150) outlasts the drain (100) so the write-vs-close race is eliminated and the honest unanswered note is deterministic"
  - "credential-cipher-scripted keeps the capture-verbatim params key order (requestId, operation, value) in the RawMessage and declares NO expect_answer — the timeout path is the fixture's point; the ack follows the 300 ms expiry because the fake's documented rule ordering is side effects before reply (the capture's 10 s send-to-cipher gap is deliberately condensed)"
  - "cancel-mid-turn's stop rule emits a scenario-declared cancelled patch (stop was never exercised live — by-construction valid like turn.completed); it doubles as the driver's wire-level sync signal proving the fake processed the bypass stop frame"

patterns-established:
  - "Corpus replay drivers live in a driver-func table keyed by scenario file name; TestGolden fails loud on a driver-less entry — adding a scenario means adding its replay driver"
  - "exchangeDriver carries t inside the harness struct (happyTurnHarness pattern) so driver funcs keep revive context-as-argument (ctx first) without tripping thelper's t-first rule"
  - "The zero-frame failure signature is a sentinel error (errZeroFrameExchange) checked by the shared harness helper — TestEOFSignature proves it at the helper level AND against a real no-traffic exchange"

requirements-completed: [BRIDGE-02]  # COPIED VERBATIM from 03-03-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-02: 03-01/03-03/03-06/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Fail-loud scenario loader validation: unknown schema keys, both-kind side effects, code-less error replies, and the empty-table-without-termination combination are rejected with errors naming the offending field; the seven existing tracer validations locked field-naming behavior"
    requirement: BRIDGE-02
    verification:
      - kind: unit
        ref: "internal/zcode/fake/scenario_test.go#TestLoadScenarioRejectsInvalidSideEffectUnknownKey (+ 10 sibling TestLoadScenarioRejectsInvalid* tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The 10-scenario golden corpus: nine authored scenarios (rows 2-10) with capture-verbatim event shapes re-serialized under fixed synthetic ids, each replayed through a real zcode.Conn over io.Pipe and byte-compared to its recorded golden under -race; happy-turn re-recorded byte-identical"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden (10 subtests, green under -race -count=3)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Exit-0-with-zero-lines failure signature honored in-process: an exchange ending with zero recorded frames reports FAILURE (errZeroFrameExchange), proven at the helper level and against a real no-traffic exchange"
    requirement: BRIDGE-02
    verification:
      - kind: unit
        ref: "internal/zcode/fake/golden_test.go#TestEOFSignature"
        status: pass
    human_judgment: false
  - id: D4
    description: "Fake scripting-core extensions the corpus required: RequestRule.Times match budget (spent rule falls through — error-32010's second-send refusal) and closed-stream write honesty (eof-drain's in-flight reply recorded as an unanswered note on a clean run, never a run error)"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/error-32010 + TestGolden/eof-drain"
        status: pass
    human_judgment: false

# Metrics
duration: 17min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 03: Golden Corpus Scenarios 2-10 + Loader Hardening Summary

**The fake app-server is now the executable protocol spec: a fail-loud scenario loader (strict decode naming offending fields) plus a 10-scenario golden corpus — happy turn, cancel, -32010 double-send, -32601 host answers, both runtime-preferences scopes, idle-patch and quiet-window completions, EOF drain, envelope negatives, and the tracked-no-answer credential cipher — every scenario replaying byte-deterministically through a real zcode.Conn under -race with zero new dependencies**

## Performance

- **Duration:** 17 min
- **Started:** 2026-11-19T18:42:18Z
- **Completed:** 2026-11-19T18:59:30Z
- **Tasks:** 2
- **Files:** 25 (4 Go source/test files, 9 scenario JSONs, 10 goldens, 2 RED evidence records)

## Accomplishments

- Task 1 (TDD): the scenario loader is fail-loud with field-naming validation — strict decoding rejects any unknown schema key naming the key (`json: unknown field "emits"`), side effects must carry exactly one of emit/reverse, error replies must carry their dialect §6 numeric code, and the termination contract (eof_drain_ms required when on_request is empty) rejects the ambiguous combination. RED evidence RED_EVIDENCE_OK (go-test adapter, target_test_failed — 4 genuinely-new targets failed fail-open or on the wrong message; 7 existing validations locked green) before the GREEN commit
- Task 2: nine corpus scenarios authored (rows 2-10) with capture-verbatim event shapes re-serialized under fixed synthetic ids — cancel-mid-turn (bypass stop + cancelled patch + ack), error-32010 (budget-spent send rule answers the second send with the bundle-literal concurrency refusal), unknown-host-method (fake reverse answered -32601 by the 03-02 router, flow continues), runtime-preferences (runtime-materialization at create AND user-execution at send, both minimal defaults), completion-idle-patch (patch.status leaves running, no terminal event), completion-quiet-window (stable session/messages polls for the 03-05/03-08 watcher), eof-drain (just-before-EOF request honestly unanswered on a clean run), envelope-negatives (20-byte malformed + 70000-byte oversized lines skipped as notes, valid frames flow), credential-cipher-scripted (capture-verbatim params, tracked-no-answer gives ZERO reply bytes, 300 ms expiry recorded)
- Table-driven TestGolden replays EVERY scenario file through a real zcode.Conn over io.Pipe and byte-compares each direction-tagged transcript; driver-less corpus entries fail loud; happy-turn re-recorded via -update came out byte-identical to the 03-01 fixture (determinism invariant)
- TestEOFSignature pins the dialect §1 failure signature in-process: a zero-frame exchange reports FAILURE via the errZeroFrameExchange sentinel — never an empty-but-successful result
- Full gates green: go test -race -count=1 ./... all packages; mise run ci (build → vet → lint 0 issues → test → test-experiments 22/22); tidy-diff clean (go.mod untouched); experiments/check-docs.sh green

## TDD Record (Task 1; times extension in Task 2)

- **RED:** `go test -race -count=1 -run TestLoadScenario ./internal/zcode/fake/` — TestLoadScenarioRejectsInvalidSideEffectUnknownKey, TestLoadScenarioRejectsInvalidSideEffectBothKinds, TestLoadScenarioRejectsInvalidUnknownMethodWithoutCode failed fail-open (tracer loader silently accepted: unknown keys ignored, both kinds defaulted to emit, zero-value code never checked), TestLoadScenarioRejectsInvalidEmptyRequestTableWithoutTermination failed on the wrong message; the 7 existing-behavior locks passed. Record classified **RED_EVIDENCE_OK** (target_test_failed, go-test adapter) before GREEN; semantic assessment in the evidence record
- **GREEN:** strict decode + exactly-one check + code check + termination rule made all 11 family tests pass; happy-turn still loads and the tracer round-trip/golden stayed green
- **Task 2 times extension:** TestLoadScenarioAcceptsRuleTimes failed RED (strict decoder rejected the unknown field `times` — exactly the schema gap) → **RED_EVIDENCE_OK** → RequestRule.Times + >= 0 validation GREEN

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: failing scenario loader validation tests (evidence RED_EVIDENCE_OK)** - `eb41500` (test)
2. **Task 1 GREEN: fail-loud scenario validation naming the offending field** - `2ddade2` (feat)
3. **Task 2: golden corpus scenarios 2-10 + deterministic replay** - `17bf12e` (feat)

## Files Created/Modified

- `internal/zcode/fake/scenario_test.go` - 13 fail-loud loader tests (11 rejects + times acceptance), each its own test with an inline temp-dir fixture (invalid fixtures never under testdata/)
- `internal/zcode/fake/scenario.go` - strict decode (DisallowUnknownFields + single-document), termination contract, error-code requirement, exactly-one side effect, RequestRule.Times + validation
- `internal/zcode/fake/golden_test.go` - TestGolden (table-driven corpus replay, driver table, fail-loud driver-less entries), TestEOFSignature, errZeroFrameExchange sentinel, exchangeDriver harness (notification channel, state-patch wait, raw-line writer), nine per-scenario drivers
- `internal/zcode/fake/fake.go` - ruleFor match budgets under ruleMu; closed-stream write honesty (unanswered notes on clean runs, other write errors still fail the run); appendLine/record split (lock-held append)
- `internal/zcode/fake/testdata/scenarios/*.json` - nine new corpus rows (fixed ids sess_fix1/turn_fix1; capture shapes re-serialized, never copied)
- `internal/zcode/fake/testdata/golden/*.golden` - nine new goldens recorded via -update; happy-turn.golden re-recorded byte-identical
- `.planning/.../evidence/03-03-red-scenario-loader-validation.json`, `03-03-red-times-budget.json` - persisted RED evidence, both RED_EVIDENCE_OK

## Decisions Made

- Strict decode over selective checks: one mechanism rejects every typo'd schema key with a field-naming message, instead of enumerating unknown-key cases per struct
- Termination contract reads the plan literally: empty on_request + missing eof_drain_ms is the rejected combination; an empty table with a declared drain is valid
- RequestRule.Times budget (not reply sequencing) expresses per-invocation scripting declaratively; matched counts under a dedicated ruleMu since handler goroutines are concurrent
- eof-drain uses conn.Stop (bypass, fire-and-forget, request-shaped) as the just-before-EOF request with delay_ms 150 > drain 100, eliminating the write-vs-close race so the honest note is deterministic
- credential-cipher-scripted preserves the capture's params key order in the RawMessage and declares no expect_answer; the ack follows the expiry because the fake's documented ordering is side effects before reply (the capture's 10 s send-to-cipher gap is condensed)
- cancel-mid-turn's stop rule emits a scenario-declared cancelled patch — by-construction valid (stop never exercised live), and the driver's wire-level sync signal
- Fire-and-forget bypass frames sync via a notification signal plus a bounded 100 ms quiet window (stopSettle) — the 03-02 bounded-quiet-window pattern applied positively, never wall-clock racing

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] RequestRule.Times match budget (fake.go ruleFor + scenario.go field/validation)**
- **Found during:** Task 2 (authoring error-32010)
- **Issue:** the tracer's ruleFor matches the FIRST rule per method, so a second session/send gets the same ack — the plan's error-32010 scenario (second send answered -32010) is inexpressible in declarative JSON
- **Fix:** Times field (json times, omitempty) caps a rule's matches; a spent rule falls through to the next same-method entry; matched counter guarded by ruleMu; fail-loud negative-times validation added TDD-style with its own RED_EVIDENCE_OK record
- **Files modified:** internal/zcode/fake/fake.go, internal/zcode/fake/scenario.go, internal/zcode/fake/scenario_test.go
- **Verification:** TestGolden/error-32010 green (second send refused -32010); TestLoadScenarioRejectsInvalidNegativeTimes green
- **Committed in:** 17bf12e

**2. [Rule 3 - Blocking] Closed-stream write honesty in fake.Serve (write() note path)**
- **Found during:** Task 2 (authoring eof-drain)
- **Issue:** the tracer treats ANY failed write as a run error, so the planned eof-drain row ("stdin close → drains ~100 ms → exit 0; in-flight request recorded unanswered", RESEARCH corpus row 8 / dialect §1) was unachievable — the lost reply failed the run instead of being recorded
- **Fix:** a write failing with a closed-stream error (io.ErrClosedPipe/os.ErrClosed, the isCleanEOF class) records an unanswered note labeling the frame and the run stays clean; all other write errors still fail the run (a mid-run broken pipe is a genuine failure)
- **Files modified:** internal/zcode/fake/fake.go
- **Verification:** TestGolden/eof-drain green — transcript is the request line plus `note reply i:1 unanswered (host stream closed)`, Serve returns nil error
- **Committed in:** 17bf12e

**3. [Rule 1 - Bug] trMu re-entrant deadlock in the new write() note path**
- **Found during:** Task 2 (first eof-drain replay run)
- **Issue:** the first draft called s.record() (locks trMu) inside write()'s already-locked critical section — the handler goroutine deadlocked on the non-reentrant mutex and Serve never returned (test timed out at 5 s)
- **Fix:** extracted appendLine (assumes trMu held); record() becomes the locking wrapper; write() appends directly in its critical section
- **Files modified:** internal/zcode/fake/fake.go
- **Verification:** eof-drain replay completes in ~200 ms; full corpus green under -race -count=3
- **Committed in:** 17bf12e

**4. [Rule 3 - Blocking] Strict-gate lint findings on the new test/harness code (7 findings)**
- **Found during:** Task 2 (mise run lint gate)
- **Issue:** lll ×4 (long error string, long fixture JSON lines), thelper ×2 (t-not-first driver signatures conflicting with revive context-as-argument), unused mustCall, revive var-naming driveEofDrain, goconst sessionId ×3, gochecknoglobals goldenDrivers map
- **Fix:** split long lines; driverFunc carries t inside exchangeDriver (happyTurnHarness pattern — no *testing.T param, ctx stays first); removed unused helper; driveEOFDrain; wireKeySessionID constant; goldenDrivers converted to a function
- **Files modified:** internal/zcode/fake/golden_test.go, internal/zcode/fake/scenario_test.go
- **Verification:** mise run lint — 0 issues
- **Committed in:** 17bf12e

---

**Total deviations:** 4 auto-fixed (1 Rule 1 bug, 3 Rule 3 blocking) + 0 plan-text adaptations
**Impact on plan:** all fixes required to satisfy the tasks' own acceptance criteria and the corpus rows as specified; no scope creep; no checkpoints consumed

## Issues Encountered

- The eof-drain row exposed the tracer's fail-on-any-write-error semantics AND a self-inflicted mutex deadlock in the fix's first draft — both caught by the scenario's own replay test inside the task, before commit
- The parallel phase-02 close-out stream landed one commit (c14ac96 docs(phase-02)) on the shared sequential branch inside this plan's ledger window; no overlap with this plan's files; the measured count (4) therefore exceeds this plan's own 3 commits — both numbers recorded in frontmatter actuals
- Scenario 10's golden was recorded against the CURRENT tree state as the dispatch contract required: the conn router's tracked-no-answer (03-02, commit 6c3bcbe) produces zero reply bytes, so the fake's note is a timeout expiry, never a -32601 answer; scenarios 4 and 5 replayed byte-identically through the 03-02 router as predicted

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03-06 (cmd/fake-app-server): Serve's clean-EOF classification now covers the closed-stream write class, so the standalone binary can exit 0 through the drain window with honest notes
- Plans 03-05/03-08 (turn watcher + machine): completion-idle-patch and completion-quiet-window scenarios are the scripted completion variants the machine's signal union consumes; the quiet-window row's session/messages poll answers are the settling surface; cancel-mid-turn's cancelled patch is the cancel-path fixture
- Plan 03-09 (E2E + phase gate): TestGolden is the replay-all entry point the phase gate runs; the driver-table pattern extends to any new corpus row
- BRIDGE-02 completion remains intentionally pending: declared by 03-01/03-03/03-06/03-09, so requirements.ready-ids returns [] until 03-06/03-09 SUMMARYs land

## Threat Surface Scan

No new security-relevant surface beyond the plan's threat model: T-3-04's mitigation IS Task 1 (the loader validates schema and fails loud naming the field; scenarios stay repo-controlled data, never user input) and is now test-locked; the prohibition holds — fixtures re-serialize scrubbed capture shapes under fixed synthetic ids (credential value is an explicit `fix1-synthetic-not-a-real-credential` marker), verified by experiments/check-docs.sh green. T-3-SC clean: zero dependencies, tidy-diff green.

## Self-Check: PASSED

- 03-03-SUMMARY.md exists on disk
- All claimed files exist: scenario_test.go, golden_test.go, scenario.go, fake.go, 9 scenario JSONs, 10 goldens, 2 evidence records (FOUND ×25)
- All claimed commits are ancestors of HEAD: eb41500 (Task 1 RED), 2ddade2 (Task 1 GREEN), 17bf12e (Task 2)
- Acceptance greps: `grep -c TestLoadScenarioRejectsInvalid scenario_test.go` = 22 (>= 1); scenario files = 10; goldens = 10; completion-quiet-window.json present (exit 0); expect_answer absent from credential-cipher-scripted.json; happy-turn.golden re-record byte-identical (empty git diff)
- RED evidence classified RED_EVIDENCE_OK (×2 records) before their GREEN stages
- Final gate run: mise run ci fully green (build, vet, lint 0 issues, go test -race all packages, test-experiments 22/22); tidy-diff clean; check-docs.sh green
- Ledger-measured plan commit count at SUMMARY write: 4 (3 this plan's own; 1 parallel phase-02 docs stream)

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
