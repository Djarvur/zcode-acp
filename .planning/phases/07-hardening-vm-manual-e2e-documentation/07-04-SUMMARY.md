---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 04
subsystem: cli
tags: [acp, omp, cli, stdlib-flag, json-rpc, version-probe, tdd]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: the stdlib-flag root CLI (flag block, parseConfig, exitInvalidConfig, version var) this plan extends
  - phase: 06-oauth-coding-plan-path-model-picker
    provides: applyConfiguredModel setModel follow-up machinery, ModelSelection, the settings snapshot store (recordSettings/latestSettings)
  - phase: 07-hardening (plans 01-02)
    provides: lifecycle-consistency + advisory batch the tree rides (mise run ci green baseline, IN-04 omit-shape)
provides:
  - omp subcommand dispatch: first-positional switch (ompRequested), permissive re-tokenizing parse (ompParse/ompFlagSet) tolerating unknown flags and the card's accepted-and-ignored --plugin-dir/--session-dir
  - --model launch binding under omp through the SAME parseConfig -> WithModel -> applyConfiguredModel machinery (ids/levelExplicit logged, shapes only)
  - version floor stamp 18.1.9+zacp.0 answering the client probe at root, under omp, in any position (-ldflags override seam kept)
  - the reasoning level's default chain in applyConfiguredModel: explicit first, else the target model's backend-confirmed defaultLevel from the create-reply snapshot (snapshotDefaultLevel), else the IN-04 omit-shape
  - regression locks: root-mode vectors unchanged, first-positional dispatch coexistence, usage-vs-runtime exit-code classes on both arms
affects: [07-05 /model slash prompt (the SESS-03 prompt half), 07-08 launch recipe (launchesWithModel:true becomes claimable), 07-09 manual E2E (live probe + A1 id-shape observation)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 13299      # chars/4 over the realized diff (53198 diff chars)
  tasks: 3
  commits: 5         # MEASURED: git rev-list --count 6f6281d..HEAD
plan_head_before: 6f6281d4ba633666ebfba44dbe05897984a56462
plan_head_after: 8f174318caee38841cd00cdaf67c95ee648fd274

# Tech tracking
tech-stack:
  added: []          # the prohibition held: no second require line — stdlib flag.FlagSet pre-tokenizing dispatch
  patterns:
    - "first-positional subcommand dispatch over stdlib flag (pre-scan before flag.Parse — Pitfall 3)"
    - "permissive chunk-parse loop: positionals skipped, unknown flags tolerated-and-counted, known-flag syntax errors honest"
    - "snapshot-derived level default chain (explicit -> snapshot defaultLevel -> IN-04 omit-shape)"

key-files:
  created:
    - cmd/zcode-acp/testdata/omp-binding-e2e.json (the binding ride's fake scenario: create reply carries the account snapshot with defaultLevel high)
    - .planning/tdd/07-04-task1-red.json, 07-04-task1-root-red.json, 07-04-task2-red.json, 07-04-task2-flagset-red.json (persisted RED evidence)
  modified:
    - cmd/zcode-acp/main.go (version stamp, omp dispatch, runOmp contract)
    - cmd/zcode-acp/main_test.go (version-floor, binding E2E, flag matrix, sweep tests)
    - internal/acpfront/front.go (applyConfiguredModel level chain + snapshotDefaultLevel)

key-decisions:
  - "The reasoning level's default chain lives in applyConfiguredModel (front.go), not the cmd layer — the create-reply snapshot only exists inside the front; explicit -> snapshotDefaultLevel -> omit-shape, never a fabricated level"
  - "Task 1 shipped the full permissive parse skeleton (the plan's 'delegation stub' bound no card flags), keeping task 2's binding RED genuine"
  - "The binding ride passes the fake install root via BOTH --app-server flag and env override: flag wins post-dispatch, env keeps any pre-dispatch fall-through hermetic (never a real desktop spawn — the CI blocker rule)"
  - "TestOmpFlagSet landed CONFIRM-GREEN at RED and was recorded honestly (classifier INVALID_RED unexpected_green, persisted verbatim) — the plan's action text anticipated the branch; the genuine RED for task 2's production edits is TestOmpLaunchBinding"

patterns-established:
  - "omp dispatch skeleton: ompRequested pre-scan + ompParse permissive chunk loop + ompFlagSet — future card personas can reuse the dispatch shape per the EXT-04 pluggable-profiles intent"
  - "snapshotDefaultLevel: derive backend-confirmed defaults from the recorded create-reply snapshot, never fabricate"

requirements-completed: [SESS-03]

coverage:
  - id: D1
    description: "Version floor: `zcode-acp --version`, `zcode-acp omp --version`, and `zcode-acp omp acp --version` each print one stdout line `zcode-acp <semver>` with a semver >= 18.1.9 (the stamp 18.1.9+zacp.0)"
    requirement: SESS-03
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestOmpVersionProbeFloor"
        status: pass
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestRootVersionFloor"
        status: pass
    human_judgment: false
  - id: D2
    description: "omp dispatch re-tokenization: unknown flags tolerated-and-counted, --plugin-dir/--session-dir accepted-and-ignored, the bare `acp` positional ignored, version answered in any position; first-positional dispatch is the only switch (value tokens never hijack the arm)"
    requirement: SESS-03
    verification:
      - kind: unit
        ref: "cmd/zcode-acp/main_test.go#TestOmpFlagSet"
        status: pass
      - kind: unit
        ref: "cmd/zcode-acp/main_test.go#TestOmpRootCoexistence (first-positional dispatch unit scan)"
        status: pass
    human_judgment: false
  - id: D3
    description: "--model launch binding: `omp acp --model <providerId/modelId>` fires the setModel follow-up after create+subscribe with the parsed ids; the reasoning level defaults to the snapshot's backend-confirmed defaultLevel (never fabricated), explicit --reasoning-level wins"
    requirement: SESS-03
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestOmpLaunchBinding"
        status: pass
      - kind: unit
        ref: "internal/acpfront (applyConfiguredModel level chain — full acpfront suite green, corpus preamble locks unchanged)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Honest parse errors and exit-code classes: malformed --model shape and level-without-model answer the existing sentinels with the root path's exitInvalidConfig usage class (2); runtime failures stay the distinct exit-1 class on both arms"
    requirement: SESS-03
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestOmpExitCodes"
        status: pass
    human_judgment: false
  - id: D5
    description: "Root acp mode byte-identical with omp present: root --version, root --model acp-mode binding, bad-config usage class, and the wiring seams vectors all pass unchanged"
    requirement: SESS-03
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestRootModeUnchanged"
        status: pass
      - kind: integration
        ref: "mise run ci (build + vet + lint + full -race suite + experiments) exit 0"
        status: pass
    human_judgment: false
  - id: D6
    description: "Dependency and secret gates hold: go.mod keeps exactly one require line pinned to acp-go-sdk v0.20.0 (the tidy-diff gate); no credential shapes introduced (scan-secrets green)"
    requirement: SESS-03
    verification:
      - kind: other
        ref: "mise run tidy-diff exit 0; bash experiments/scan-secrets.sh exit 0"
        status: pass
    human_judgment: false

# Metrics
duration: 35min
completed: 2026-11-22
status: complete
---

# Phase 7 Plan 04: D-03 Persona CLI Contract Summary

**omp subcommand dispatch with re-tokenized card argv, `--model` launch binding through the setModel follow-up with a snapshot-derived reasoning-level default, and the 18.1.9+zacp.0 version floor — root acp mode byte-identical, one require line intact**

## Performance

- **Duration:** 35 min
- **Started:** 2026-11-22T11:57:20Z
- **Completed:** 2026-11-22T12:33:35Z
- **Tasks:** 3
- **Files modified:** 8 (2 production, 2 test/fixture, 4 evidence records)

## Accomplishments
- The version probe now passes the client's floor in all three argv shapes (root, `omp --version`, `omp acp --version`) — the card is installable in Orca once wired (Pitfall 2 closed); the stamp carries build metadata with the `-ldflags` release seam kept open
- The omp arm re-tokenizes post-omp argv (Pitfall 3 closed): unknown flags tolerated-and-counted, `--plugin-dir`/`--session-dir` accepted-and-ignored with contract-naming debug lines, the bare `acp` marker skipped, honest errors for known-flag syntax failures
- `--model` binds at launch through the SAME machinery as the picker (parseConfig -> WithModel -> applyConfiguredModel) with the level default chain: explicit first, else the create-reply snapshot's backend-confirmed defaultLevel, else the IN-04 omit-shape — never a fabricated level
- Regression locks both ways: root-mode vectors unchanged, first-positional dispatch is the only switch (value tokens and the `--` terminator handled), usage-vs-runtime exit classes preserved; `launchesWithModel:true` becomes claimable for the launch recipe (07-08)

## Task Commits

Each task was committed atomically (TDD tasks as test->feat pairs):

1. **Task 1 (tracer): version probe floor + omp dispatch** — `0238440` (test, RED: RED_EVIDENCE_OK x2) -> `b3c2660` (feat)
2. **Task 2: card argv contract + launch binding** — `5915e95` (test, RED: RED_EVIDENCE_OK; flag matrix CONFIRM-GREEN recorded honestly) -> `30e8b08` (feat)
3. **Task 3: regression sweep + phase gates** — `8f17431` (test, incl. the behavior-identical lint refactor)

**Plan metadata:** the docs commit (this SUMMARY + STATE/ROADMAP) follows.

_Note: TDD tasks carry RED->GREEN pairs; no refactor commit was needed._

## Files Created/Modified
- `cmd/zcode-acp/main.go` — version stamp 18.1.9+zacp.0; ompSubcommand dispatch (ompRequested pre-scan, ompRootValueConsumes, ompFlagSet/ompParse permissive loop, runOmp contract with binding notice and contract-naming debug lines)
- `cmd/zcode-acp/main_test.go` — TestOmpVersionProbeFloor, TestRootVersionFloor, TestOmpLaunchBinding, TestOmpFlagSet (+8 named subtests), TestRootModeUnchanged, TestOmpRootCoexistence, TestOmpExitCodes; ride/scan helpers
- `cmd/zcode-acp/testdata/omp-binding-e2e.json` — the binding ride's fake scenario (create reply carries settings.model.available with GLM-5.3-Flash defaultLevel high)
- `internal/acpfront/front.go` — applyConfiguredModel level default chain + snapshotDefaultLevel (doc comments updated; IN-04 omit-shape preserved)
- `.planning/tdd/07-04-task{1,1-root,2,2-flagset}-red.json` — persisted RED evidence records

## Decisions Made
- **The level-default chain lives in applyConfiguredModel (front.go), not cmd** — the create-reply snapshot only exists inside the front (recorded just above the follow-up), so the derivation rides the same setModel machinery the picker uses; an unlisted target answers "" (the omit-shape), never a fabricated level
- **Task 1's "delegation stub" bound no card flags** — the stub delegates with an empty config so task 2's binding RED stayed genuine (the plan's own sequencing)
- **Hermetic binding rides** — the fake install root rides both `--app-server` (wins post-dispatch) and the env override (keeps any pre-dispatch fall-through on the fake, never a real desktop spawn)
- **stdlib FlagSet dispatch, not cobra** — per the plan's prohibition; go.mod keeps exactly one require line and the tidy-diff gate stays green

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] The reasoning level's snapshot derivation had to land in front.go**
- **Found during:** Task 2 (launch binding)
- **Issue:** The plan's files_modified lists only cmd files and its read_first says "NO changes to applyConfiguredModel" — but the must-have truth and Test 2 require the level to default from the create-reply snapshot's defaultLevel, which the cmd layer cannot reach (the snapshot is recorded inside the front at session/new)
- **Fix:** `snapshotDefaultLevel` + the level chain inside `applyConfiguredModel` (explicit -> snapshot defaultLevel -> omit-shape); no second setModel path — the SAME machinery the picker rides
- **Files modified:** internal/acpfront/front.go
- **Verification:** TestOmpLaunchBinding's derived-default arm passes; the full acpfront/acpport suites stay green (corpus preamble locks unaffected — no existing scenario combines a settings snapshot with an empty level)
- **Committed in:** 30e8b08

**2. [Rule 3 - Blocking] Added the binding E2E's fake scenario fixture**
- **Found during:** Task 2 RED authoring
- **Issue:** No existing fake scenario's create reply carries a settings snapshot with a defaultLevel, and the golden corpus is byte-locked — the derived-level arm had no fixture
- **Fix:** `cmd/zcode-acp/testdata/omp-binding-e2e.json` following the committed responder-e2e.json precedent (cmd-local testdata is outside the fake corpus's scan path)
- **Files modified:** cmd/zcode-acp/testdata/omp-binding-e2e.json
- **Verification:** The fixture loads and rides clean (frames=6); TestOmpLaunchBinding fails on the binding pre-GREEN, never on fixture errors
- **Committed in:** 5915e95

**3. [Rule 3 - Blocking] Task 3 needed a small production refactor for the strict lint gate**
- **Found during:** Task 3 (phase gates)
- **Issue:** The plan says "No production changes in this task", but `mise run lint` (the wave-close proven gate) rejects the package-level `ompRootValueFlags` map literal (gochecknoglobals) along with test-file globals, repeated literals, gocognit on the matrix test, and four over-length lines
- **Fix:** Behavior-identical refactor: the map became `ompRootValueConsumes(name)`; the test floor global became comparator constants; the matrix test split into eight named subtest funcs; string constants for the repeated argv tokens; line wraps — the tree stayed lint-green
- **Files modified:** cmd/zcode-acp/main.go, cmd/zcode-acp/main_test.go
- **Verification:** `mise run lint` 0 issues; full omp test set green after the refactor
- **Committed in:** 8f17431

---

**Total deviations:** 3 auto-fixed (1 missing critical, 2 blocking) + 1 TDD-gate branch recorded honestly (below)
**Impact on plan:** All three were required by the plan's own contracts (must-have truths, the E2E design, the lint gate). No scope creep; the one-require-line prohibition held throughout.

## TDD Gate Compliance

- Gate sequence in git log: `test(7-04)` 0238440 -> `feat(7-04)` b3c2660; `test(7-04)` 5915e95 -> `feat(7-04)` 30e8b08. Task 3 is non-TDD (test-only commit).
- Task 1: both records `RED_EVIDENCE_OK (target_test_failed)` — the omp shapes never answered the probe (Pitfall 3 fall-through) and the root answer parsed to 0.0.0 (Pitfall 2's tooOld shape).
- Task 2: TestOmpLaunchBinding `RED_EVIDENCE_OK (target_test_failed)` — all three arms failed on the missing setModel (the task-1 stub's designed silence). **TestOmpFlagSet landed CONFIRM-GREEN at RED** — the classifier verdict `INVALID_RED (unexpected_green)` is persisted verbatim in `.planning/tdd/07-04-task2-flagset-red.json` with the semanticAssessment: the plan's action text explicitly anticipates the branch ("the parse skeleton exists from task 1's minimal dispatch; ... so assertions fail on values, never on build errors"), no failing assertion was manufactured (07-01 task-1 precedent), and the genuine RED authorizing task 2's production edits is TestOmpLaunchBinding. The matrix stays as the contract lock.

## Issues Encountered
- The nonexistent-install `--app-server` override fails at SPAWN (fork/exec), not at discovery — discovery's step-1 override accepts the shape unconditionally by design; the exit-1 arms assert the honest "zcode-acp: exiting after failure" framing instead of a discovery diagnostic
- slog's TextHandler escapes quotes inside err values, so the got-value assertions match on stable fragments (sentinel text + the raw value) rather than the exact quoted wrap

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 07-05 (the `/model` slash prompt, SESS-03's prompt half) rides the same applyConfiguredModel/setConfigMutation machinery this plan's level chain extended; the detail-carrying refusal surface (`-32603` + `data.details`) is still to land there
- 07-08 can document `launchesWithModel:true` (the binding is live and test-proven); RESEARCH Pattern 8's recipe inputs (version floor 18.1.9+, override wiring, PATH shim) are all real now
- 07-09's manual E2E observes the live probe and the A1 id shape (what Orca actually passes as `--model`) — the wrapper logs the binding at info (`zcode-acp omp: model selection bound at launch`) for exactly that observation; the strictness of the snapshot match (case-sensitive exact id equality) is the documented posture if the id differs
- Known Stubs: none. Threat mitigations landed as designed (T-07-11 re-tokenization + binding wire proof; T-07-12 ids/shapes-only logging, scan-secrets green; T-07-SC stdlib choice, one require line).

## Self-Check: PASSED

All nine created/modified files verified on disk; all five task commits verified as ancestors of HEAD (0238440, b3c2660, 5915e95, 30e8b08, 8f17431); commits measured from the persisted plan ledger (6f6281d..HEAD = 5).

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*
