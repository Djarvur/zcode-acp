---
phase: 05-interactions-full-chat-cycle-api-key
plan: 08
subsystem: full-cycle-e2e
tags: [core-06, tool-01, tool-02, sess-01, full-cycle, composed-e2e, fake-corpus, golden, permission-bridge, tool-stream, resume-close, authenticate]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key
    provides: the leg contracts this composes — 05-02 permission bridge (+ corpus reverse shapes), 05-04 tool stream + pump (+ tool.updated framing), 05-05 resume/close (+ lifecycle rules), 05-06 auth push/read-back machinery, 05-07 Front.Authenticate + responder wiring
  - phase: 04-acp-front-minimal-chat-loop
    provides: ComposeFront harness (tee + RecordingClient), the corpus-locked preamble, flush-before-stopReason locks
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: scripted fake + golden corpus machinery (fail-loud loader, -update recording, driver registry)
provides:
  - corpus: full-cycle.json + golden — the ONE-scenario full cycle whose golden byte-locks the interleaved wire order (auth chain → preamble → send with tool scheduled/started → the permission reverse and its reply BETWEEN the tool frames → progress/result/completion → resume → re-subscribe → close)
  - test: driveFullCycle (the per-leg drivers' call sequences composed: driveAuthApikey's push→refresh→read-back verdict, driveFrontPreamble, the drivePermissionAllow bridge chain, driveSessionLifecycle's resume/subscribe/close)
  - test: TestFullCycle — the composed E2E every success-criterion-5 leg rides at the composed tier (the phase's composed-tier gate)
affects: [05-09 guarded real tier (replays this cycle on the real account), Phase-7 QA-01 full-scenario acceptance, .planning/intel/API-SURFACE.md harvest]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 8110    # chars/4 over the realized diff (32442 chars, 4 files, +714/-0); plan estimated 32000
  tasks: 2
  commits: 2      # MEASURED: git rev-list --count gsd-plan-head-before-05-08..HEAD
commits: 2
plan_head_before: ed497c5878693e11de3fd34bc02fbe4fd5bf1fdb
plan_head_after: 6a972938dc13ba02a82e4341b27a5cc2b7a5d381

# Tech tracking
tech-stack:
  added: []        # zero new dependencies (test-only plan; module graph unchanged — T-5-SC)
  patterns:
    - composed-tier placement asserts ride causally airtight edges: the completed tool patch strictly postdates the permission ask (causal through the fake's expect_answer), and the backend byte order stays golden-locked at the fake tier (the 05-04 coalescing-aware composed-tier discipline carried forward)
    - confirm-green-at-RED for pure-composition TDD tasks: when every leg is already individually locked and the composed test needs no new glue, the green run IS the result — no RED evidence is fabricated
    - transcript-level locks run once at the story's end (the harness's Transcript accessor closes the back leg), tee/collector locks run mid-flow

key-files:
  created:
    - internal/zcode/fake/testdata/scenarios/full-cycle.json
    - internal/zcode/fake/testdata/golden/full-cycle.golden
    - internal/acpfront/fullcycle_test.go
  modified:
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "Rule shapes copied verbatim from the four leg rows (no re-derivation): the auth rules from auth-apikey.json (ack + refresh with the registry state.updated emit), the preamble from the shared row shape, the send rule composing tool-lifecycle.json's Edit framing with permission-allow.json's reverse (allow_once/deny embedded responses, reannounce 2x40ms, expect allow) fired BETWEEN the started and progress emits, resume/close from session-lifecycle.json"
  - "The fake-tier authenticate leg drives the hostsvc step functions directly (the driveAuthApikey sequence, injected fixed revision so the static ack echoes verbatim) while TestFullCycle drives the REAL Front.Authenticate — the corpus row byte-locks the wire, the composed tier locks the front contract; both read-back verdicts are registry-driven (PITFALL 1)"
  - "The composed permission-placement assert pins the deterministic edges — ask after the start card and started patch, progress patch (in_progress) and completed patch strictly after the ask — verified stable across -count=5 and shuffled -count=3 under -race; the fully ordered backend bytes stay the golden's job"
  - "TestFullCycle is ONE harness and ONE continuous client story (no subtests): each leg's asserts reuse the owning plan's helpers (assertEditStartCard/assertEditPatchSequence, assertCapturedPermissionAsk, assertResumeLegOrder, assertBackLegNewSessionSequence, assertCodedInvalidRequest), proving composition rather than re-locking legs"

patterns-established:
  - "Full-cycle composition audit: tee/collector asserts mid-flow, transcript asserts once at the end — the harness lifecycle dictates the assert surface per leg"

requirements-completed: [CORE-06, TOOL-01, TOOL-02, SESS-01]  # verbatim copy of the plan frontmatter; the shared-ID gate keeps REQUIREMENTS.md open until sibling 05-09 lands its SUMMARY

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The full-cycle corpus row + golden: the one-scenario cycle replays deterministically with the golden byte-locking the interleaved wire order — auth chain (push ack echoing the injected revision, refresh, registry surface), preamble (create framing + subscribe + setModel), send with tool scheduled/started, the permission reverse (fake-2) and its embedded-response reply landing between the started (seq 3) and progress (seq 4) tool frames, progress/result/text/completed framing, resume (create-shaped snapshot) → re-issued subscribe → close"
    requirement: TOOL-01
    verification:
      - kind: e2e
        ref: "internal/zcode/fake/golden_test.go#TestGolden/full-cycle (go test -race -count=1 -run 'TestGolden/full-cycle' ./internal/zcode/fake/; byte-stable across -count=3)"
        status: pass
    human_judgment: false
  - id: D2
    description: "driveFullCycle composes the per-leg drivers' call sequences (authenticate → new → prompt → resume → close) with the full bridge chain at the fake tier — the scripted client's allow_once decision forwarded byte-verbatim through the expect_answer (the run is clean; exactly one reply per reverse)"
    requirement: CORE-06
    verification:
      - kind: e2e
        ref: "internal/zcode/fake/golden_test.go#TestGolden/full-cycle (the driver is the exercised surface; the fake's expect_answer validation gates the run)"
        status: pass
    human_judgment: false
  - id: D3
    description: "TestFullCycle — the composed E2E: authenticate (empty {} on the tee + transcript push→refresh→registry-surface evidence, both before session/new) → session/new → prompt with ordered tool frames (start card, diff at session-cwd-absolute path, patches, rawOutput verbatim, diff re-emitted), the permission ask with mapped options landing between the started and progress patches, end_turn AFTER every tool frame, zero update frames after the response → resume with ZERO replay → close releasing exactly the machine's observers, post-close prompt rejected coded-clean, no stop leg on the idle teardown — one continuous composition under -race; every per-leg suite stays green (no loosened locks)"
    requirement: SESS-01
    verification:
      - kind: integration
        ref: "internal/acpfront/fullcycle_test.go#TestFullCycle (go test -race -count=1 -run 'TestFullCycle' ./internal/acpfront/; stable -count=5 and -shuffle -count=3)"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/... (9 packages green — per-leg locks unloosened)"
        status: pass
      - kind: e2e
        ref: "mise run ci (build → vet → lint 0 issues → test -race → test-experiments) exit 0"
        status: pass
    human_judgment: false

# Metrics
duration: 17min
completed: 2026-11-20
status: complete

---

# Phase 5 Plan 8: Full-Cycle Fake E2E Summary

**The phase's composed-tier gate is green: one scenario, one harness, one continuous client story — authenticate (read-back proven) → session/new → prompt with the Edit tool stream and a permission ask bridged to the client's allow decision mid-tool → end_turn → resume without replay → close with full teardown — byte-locked by the full-cycle golden and asserted leg-by-leg on recorded wire artifacts by TestFullCycle, under -race with `mise run ci` green.**

## Performance

- **Duration:** 17 min
- **Started / Completed:** 2026-11-20T15:18:26Z / 2026-11-20T15:35:28Z
- **Tasks:** 2 · **Files:** 3 created / 1 modified (+714/-0; fixture-weighted — the corpus row + golden ARE the composed cycle's test surface)

## Accomplishments

- The full-cycle corpus row composes the four leg rows' rule shapes verbatim into ONE continuous `on_request` flow: the auth chain (ack + refreshRegistry with the registry state.updated emit) placed so authenticate runs before session/new, the standard preamble, ONE send rule emitting the ordered tool sequence (scheduled Edit-input → started) with the `interaction/requestPermission` reverse fired BETWEEN started and progress (allow_once/deny embedded responses, reannounce 2x40 ms, expect_answer the allow response), then progress → result → text delta → completion, and the resume/close rules after the turn.
- The golden byte-locks the whole interleaved wire order; the recorded run shows the permission reply (`fake-2` → `{decision:allow, reason:Approved once}`) landing exactly between the started (seq 3) and progress (seq 4) tool frames, exactly one reply per reverse, no re-announce envelopes (the bridge answers in milliseconds), deterministic across `-count=3` under `-race`.
- `driveFullCycle` composes the per-leg drivers' call sequences — the driveAuthApikey push→refresh→read-back verdict (injected fixed revision, ack echo paired, registry-driven verdict), driveFrontPreamble, the drivePermissionAllow full bridge chain over the scripted SDK client, and the driveSessionLifecycle resume/subscribe/close sequence.
- `TestFullCycle` runs the composed E2E on ONE harness: authenticate answers the empty `{}` on the tee (transcript evidence: one push + one refresh in order, both before the cycle's create, the registry surface recorded); the prompt leg asserts the ordered tool frames (start card with the Edit diff at the session-cwd-absolute path, in_progress patches, terminal completed patch with rawOutput verbatim and the re-emitted diff), the captured ask with mapped options, the tee placement between the started and progress patches, end_turn strictly after every tool frame, and ZERO update frames after the response; the resume leg collects zero new updates (no replay); the close leg releases exactly the machine's observer entries, keeps the quiet through teardown, and rejects a post-close prompt with the coded ErrNoSession-class error — and the transcript audit closes the cycle (D-03 preamble shapes, resume re-subscribe without setModel, teardown wire with no stop leg, the permission reply's embedded bytes).
- The phase's composed-tier gate: `go test -race -count=1 ./internal/...` green (9 packages — every per-leg lock unloosened, T-5-20 honored), `mise run ci` exit 0, zero new dependencies.

## Task Commits

1. **Task 1: full-cycle corpus row + golden + driver** - `8c78a0c` (feat)
2. **Task 2: TestFullCycle — the composed E2E** - `6a97293` (test)

**Plan metadata:** (this SUMMARY via the gsd-tools commit verb; STATE/ROADMAP/REQUIREMENTS ride the same metadata commit per the sequential dispatch)

## TDD Gate Compliance

- **Task 2 is `tdd="true"` and landed as CONFIRM-GREEN — the plan's own anticipated branch ("RED→GREEN if composition gaps surface, else confirm-green with the staged asserts").** The RED run was performed honestly: the test was written and executed against the committed per-leg production before any adjustment — it passed on the first run, and the repeated runs (`-count=5`, `-shuffle -count=3`, plus the full suite and CI) confirmed it. No composition gap surfaced, so no new composition glue existed to drive a genuine RED, and no failing assertion was manufactured to stage one (the RED-evidence rules forbid fabricated RED; a nonzero-exit theater would be exactly the dishonesty the gate exists to prevent). The TDD cycle applies to new composition glue only — there was none.
- **Gate validation:** `test(05-08)` exists (`6a97293`); `feat(05-08)` exists (`8c78a0c`, Task 1's corpus+driver — not a tdd task, hence no RED). The test commit postdates the feat commit because the plan orders the corpus row (Task 1) before the composed test (Task 2); there is no GREEN-after-RED violation because no RED stage ran. No `refactor(05-08)` was needed — Task 1's funlen-driven `fullCycleAuthLeg` extraction rode its own commit.
- **No RED evidence record was written** — none exists to write for a confirm-green; this section is the record.

## Files Created/Modified

- `internal/zcode/fake/testdata/scenarios/full-cycle.json` (NEW) — the one-scenario full cycle, rule shapes verbatim from the four leg rows
- `internal/zcode/fake/testdata/golden/full-cycle.golden` (NEW) — the byte-locked interleaved wire order (recorded with the canonical `-update`, verified byte-stable)
- `internal/zcode/fake/golden_test.go` — `driveFullCycle` + `fullCycleAuthLeg` in the driver registry
- `internal/acpfront/fullcycle_test.go` (NEW) — TestFullCycle + the staged leg helpers over the composed harness

## Decisions Made

See key-decisions in frontmatter: verbatim rule-shape composition; the two-tier auth split (hostsvc step functions at the fake tier, the real Front.Authenticate at the composed tier); the causally-airtight composed placement asserts with the byte order staying the golden's job; ONE harness / ONE story with the owning plans' assert helpers reused.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The strict linter flagged `driveFullCycle` at 44 statements (funlen 40); the auth leg extracted into `fullCycleAuthLeg` (the driveAuthApikey sequence it composes), riding Task 1's commit. Lint 0 issues at both task gates.
- `go test -update` must place the `-update` flag AFTER the package list (an unknown-to-go-test flag before the packages fails setup); the canonical recording command used: `go test -race -count=1 -run 'TestGolden/full-cycle' ./internal/zcode/fake/ -update`.

## User Setup Required

None - no external service configuration required (fixtures carry synthetic material only; scan-secrets green on both commits).

## Next Phase Readiness

Ready for 05-09 — the guarded real tier. This plan's composition IS the exact cycle 05-09 replays on the real api-key account at end-of-phase (success criterion 5's real-half): `TestFullCycle`'s client story (authenticate → new → prompt with tools + permission → resume → close) rides the same Front contracts the real run will exercise; the golden's wire order is the expected real-tier shape. The only unproven arms remain the live-keyring cipher decrypt and the real registry behavior — both 05-09's guarded deliverables (D5 of 05-07's coverage, the username-derivation assumption discharge). REQUIREMENTS.md's SESS-01 checkbox stays open by the shared-ID gate (sibling 05-09 lacks a SUMMARY); CORE-06/TOOL-01/TOOL-02 ride the same gate.

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-20*

## Self-Check: PASSED

All 4 task files + this SUMMARY verified present on disk; both task commits (8c78a0c, 6a97293) verified ancestors of HEAD. Commit count measured from the persisted plan ledger (gsd-plan-head-before-05-08 = ed497c5): 2 production commits, matching the frontmatter actuals. No unexpected file deletions in either task commit.
