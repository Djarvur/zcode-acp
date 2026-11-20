---
phase: 05-interactions-full-chat-cycle-api-key
plan: 05
subsystem: session-lifecycle
tags: [sess-01, resume, close, teardown, wr-04, wr-05, in-03, in-04, tdd]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key (05-02/05-03/05-04)
    provides: permission bridge + cancel fan-in; router fan-out and Machine.CloseObservers (WR-01/WR-07); the WR-03 pump and tap turn-slot state
provides:
  - Front.ResumeSession / Front.CloseSession (SESS-01 contracts, teardown internals, closing-set race gate)
  - acpport Handlers.ResumeSession/CloseSession + adapter delegates (the two former -32601 stubs now advertised surface)
  - WR-04 best-effort orphan close in NewSession's error path (closeOrphan)
  - WR-05 scope-accurate turnActiveMessage; IN-04 options omit-shape; IN-03 doc truth
  - corpus rows session-lifecycle + newsession-orphan-close (goldens, drivers) — the resume/close wire order and the orphan-close wire shape
affects: [05-06+ phase plans sharing SESS-01, Phase-7 SESS-02/03 hardening, .planning/intel/API-SURFACE.md harvest]

# Actuals (#2632)
actuals:
  tokens: 22122      # chars/4 over the realized diff (88486 chars, 20 files, +1560/-86)
  tasks: 3
  commits: 4         # MEASURED: git rev-list --count f389643..HEAD

tech-stack:
  added: []
  patterns:
    - "closing-set race gate: mid-teardown sessions reject prompts with the coded ErrNoSession class — the close-vs-prompt race is deterministic"
    - "slot done-channel drain: acquireTurn publishes a per-turn done channel closed after release's flush — close waits bounded before the release steps"
    - "compile-presence placeholder: an interface widening that outruns its implementation ships the honest not-available answer (Authenticate posture) so TDD RED stays genuine"

key-files:
  created:
    - internal/acpfront/lifecycle.go
    - internal/acpfront/lifecycle_test.go
    - internal/zcode/fake/testdata/scenarios/session-lifecycle.json
    - internal/zcode/fake/testdata/golden/session-lifecycle.golden
    - internal/zcode/fake/testdata/scenarios/newsession-orphan-close.json
    - internal/zcode/fake/testdata/golden/newsession-orphan-close.golden
  modified:
    - internal/acpport/types.go
    - internal/acpport/agent.go
    - internal/acpport/agent_test.go
    - internal/acpfront/front.go
    - internal/acpfront/turnctx.go
    - internal/acpfront/cancel_test.go
    - internal/acpfront/doc.go
    - internal/acpfront/newsession_test.go
    - internal/zcode/envelope.go
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "CloseSession keeps the plan's teardown order and adds the closing-set gate: a prompt for a mid-teardown session lands ONE coded rejection instead of racing the release steps (T-5-12 determinism); the drain re-cancels sneaked turns"
  - "Second close and post-close prompt answer the coded ErrNoSession-class invalid-request with zero wire traffic (the Get gate) — idempotent-by-gate, consistent with Prompt/Cancel (T-5-13)"
  - "Resume carries NO registry gate (reattach semantics — the backend is the source of truth); empty id fails invalid-params before the wire; an in-process reattach releases the previous machine's observers before Register replaces it (WR-07)"
  - "closeTurnWait = 10s (the spawnWait scale, defined locally per plan); the cancelled turn's pending permission bridges answer -32021 (Pitfall 2)"
  - "TDD RED ran against the Task-1 honest not-available placeholders (the 05-03 compiling-stub precedent); all four targets failed on their planned assertions, RED_EVIDENCE_OK x4"

patterns-established:
  - "compile-presence placeholder discipline: interface widening + honest not-available body + same-plan TDD GREEN replacement, documented in the evidence records"

requirements-completed: [SESS-01]  # copy of PLAN.md frontmatter; the shared-ID gate blocks marking REQUIREMENTS.md while sibling plans 05-08/05-09 also declare SESS-01 without SUMMARYs

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "session-lifecycle corpus row: golden byte-locks create→subscribe→setModel→resume→subscribe→close; resume reply decodes through the createSessionID snapshot dig-out"
    requirement: SESS-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 -run 'TestGolden/session-lifecycle' ./internal/zcode/fake/"
        status: pass
    human_judgment: false
  - id: D2
    description: "Adapter delegates: Handlers widened with ResumeSession/CloseSession; both former -32601 stubs translate-call-code; unadvertised table shrank to five; wire-level probes answer results"
    requirement: SESS-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/acpport/ (TestLifecycleDelegation, TestLifecycleDelegationCoded, TestLifecycleDelegationWire, TestUnadvertised*)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Front.ResumeSession: same id via the create-shaped snapshot, re-issued subscribe (desktop-continuous), NO setModel on the leg, post-resume prompt resolves, zero replayed updates"
    requirement: SESS-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 -run 'TestResume$' ./internal/acpfront/"
        status: pass
    human_judgment: false
  - id: D4
    description: "Front.CloseSession ordered teardown: stop precedes close on the wire, registry release (post-close prompt rejected, no send), observer counts drop by exactly the machine's (1,1), second close clean"
    requirement: SESS-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 -run 'TestCloseTeardown$' ./internal/acpfront/"
        status: pass
    human_judgment: false
  - id: D5
    description: "Close-vs-prompt race probe: cancel fires first, bounded drain, the racing prompt lands exactly one clean outcome, 3 prompts / 3 outcome frames, no resolve-after-removal"
    requirement: SESS-01
    verification:
      - kind: integration
        ref: "go test -race -count=3 -run 'TestCloseWhileTurnActiveRace$' ./internal/acpfront/"
        status: pass
    human_judgment: false
  - id: D6
    description: "WR-04: a subscribe refusal after create succeeds closes the backend session best-effort (exactly one session/close after the refused subscribe) and answers the coded error"
    requirement: SESS-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 -run 'TestNewSessionOrphanClosed$' ./internal/acpfront/"
        status: pass
    human_judgment: false
  - id: D7
    description: "WR-05: the rejection message names the front-wide scope; asserted verbatim from both guard layers (front slot + machine ErrTurnActive mapping)"
    requirement: SESS-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 -run 'TestSecondPromptRejected' ./internal/acpfront/"
        status: pass
    human_judgment: false
  - id: D8
    description: "IN-04: level-less setModel omits the options object entirely (named test); the configured-with-level preamble stays byte-locked in the goldens"
    requirement: SESS-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 -run 'TestSetModelOmitsOptionsWithoutLevel|TestGolden/setmodel' ./internal/acpfront/ ./internal/zcode/fake/"
        status: pass
    human_judgment: false
  - id: D9
    description: "IN-03: front.go New base-context comment and doc.go paragraph state shipped reality (no behavior change — comment-only deliverable)"
    verification: []
    human_judgment: true
    rationale: "Doc-only deliverable: the refreshed comments were reviewed against the shipped code in this plan; no executable assertion exists for prose. Verifier should eyeball front.go New + doc.go."

# Metrics
duration: 49min
completed: 2026-11-20
status: complete
---

# Phase 5 Plan 05: Session Lifecycle — Resume & Close (SESS-01) Summary

**`session/resume` reattaches without replay (re-issued subscribe, no setModel) and `session/close` runs the ordered cancel→close→Remove→observer-removes teardown behind a deterministic close-vs-prompt race gate, with WR-04 orphan close, WR-05 scope-accurate wording, IN-04 options omit-shape, IN-03 doc truth, the Resume/Close adapter delegates, and the session-lifecycle + orphan-close corpus rows — `mise run ci` and the full `-race` suite green.**

## Performance
- **Duration:** 49min
- **Started:** 2026-11-20T12:50:56Z
- **Completed:** 2026-11-20T13:40:12Z
- **Tasks:** 3 (Task 2 as RED→GREEN)
- **Files modified:** 20 (6 created, 14 modified)

## Accomplishments
- SESS-01 is test-locked at both tiers: the corpus golden byte-locks the resume/close wire order, and the composed-tier tests (TestResume, TestCloseTeardown, TestCloseWhileTurnActiveRace) lock the front contracts — no replay, re-issued subscribe, no setModel on the resume leg; ordered idempotent baseline-restoring teardown.
- The adapter advertises Resume/Close through clean delegates (translate → call → wireError, RequestError unwrapped); the unadvertised -32601 table shrank from seven methods to five.
- WR-04 closes error-path orphans: every post-create failure in NewSession best-effort closes the backend session with structural logging — a client retry loop can no longer leak one backend session per attempt.
- The close-vs-prompt race is deterministic (the closing-set gate + bounded slot drain), passing under `-race` ×3; unregistered ids never touch the backend wire (T-5-13).

## Task Commits

Each task was committed atomically:

1. **Task 1: session-lifecycle corpus row + adapter delegates** — `b93a23d` (feat)
2. **Task 2: Front.ResumeSession / Front.CloseSession with teardown (RED→GREEN)** — `0d9410f` (test RED) → `541a5ab` (feat GREEN)
3. **Task 3: WR-05 wording + IN-04 omit-shape + IN-03 doc refresh** — `23925e7` (feat)

**Plan metadata:** SUMMARY commit (docs)

_Note: Task 2 is TDD — RED evidence records for all four targets (TestResume, TestCloseTeardown, TestCloseWhileTurnActiveRace, TestNewSessionOrphanClosed) are committed with the RED state under `.planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-05-red-*.json`, each classified RED_EVIDENCE_OK (target failed on its planned assertion)._

## Files Created/Modified
- `internal/acpfront/lifecycle.go` (NEW) — ResumeSession/CloseSession, the closing-set gate, drainActiveTurn, closeTurnWait, closeOrphan
- `internal/acpfront/lifecycle_test.go` (NEW) — the four SESS-01/WR-04 locks over the composed harness
- `internal/acpfront/front.go` — slot session/done fields, closing set, Prompt closing gate, WR-04 hooks, IN-04 omit-shape, IN-03 comment, createSessionID method label, wireKeySessionID const, dropCwd/isClosing
- `internal/acpfront/turnctx.go` — acquireTurn closing gate + (ctx, release, err) signature; per-turn done channel closed after release's flush
- `internal/acpfront/cancel_test.go` — the WR-05 locked constant (same commit as the constant change)
- `internal/acpfront/doc.go` — IN-03 present-tense package doc
- `internal/acpfront/newsession_test.go` — TestSetModelOmitsOptionsWithoutLevel (IN-04 named test)
- `internal/acpport/types.go` — Handlers widened with ResumeSession/CloseSession
- `internal/acpport/agent.go` — the two delegates; adapter doc counts updated
- `internal/acpport/agent_test.go` — scriptedHandlers lifecycle methods; unadvertised tables at five; delegation + wire tests
- `internal/zcode/envelope.go` — session/resume + session/close method constants
- `internal/zcode/fake/golden_test.go` — driveSessionLifecycle, driveNewSessionOrphanClose, snapshot/close asserts, delivery-kind consts
- `internal/zcode/fake/testdata/scenarios/session-lifecycle.json` + golden — the SESS-01 corpus row
- `internal/zcode/fake/testdata/scenarios/newsession-orphan-close.json` + golden — the WR-04 corpus row

## Decisions Made
- **Closing-set race gate (T-5-12):** close marks the session for the whole teardown span; Prompt/acquireTurn reject with the coded ErrNoSession class; drainActiveTurn re-cancels sneaked turns — the racing prompt lands exactly one clean outcome (the coded rejection or the cancelled resolution), never a hang or a completed-turn leak.
- **Idempotent close by the Get gate:** a second close answers the clean coded invalid-request with zero wire traffic (consistent with Prompt/Cancel and T-5-13's "rejected lifecycle calls never touch the wire"), and the teardown primitives (Remove, CloseObservers) stay idempotent beneath it.
- **Resume has no registry gate:** reattach semantics — the backend owns session truth; only the empty id is rejected pre-wire. In-process reattach releases the previous machine's observers before Register replaces it (no WR-07 leak).
- **Bounded waits:** closeTurnWait = 10s (spawnWait scale, local constant); the turn's done channel closes only after release's flush, so close cannot outrun the turn's last ACP write.
- **TDD RED against compile-presence placeholders:** Task 1's interface widening forced compile-level Front methods; the honest not-available answer (the Authenticate posture) kept the RED genuine — each target failed on its own planned assertion (classifier-verified ×4).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3] Widened Handlers forced compile-level Front methods (Task 1)**
- **Found during:** Task 1
- **Issue:** widening `acpport.Handlers` with ResumeSession/CloseSession makes `acpfront.Front` a compile-time implementor (`cmd/main.go` passes `*Front` to `acpport.NewAgent`), but the plan's Task 1 file list carried no acpfront changes — the build would break.
- **Fix:** lifecycle.go shipped the honest not-available placeholders (Authenticate posture, no backend wire traffic) in Task 1; Task 2's RED failed against them on the planned assertions and GREEN replaced the bodies.
- **Files modified:** internal/acpfront/lifecycle.go
- **Verification:** RED evidence records 05-05-red-resume/close-teardown/close-race (RED_EVIDENCE_OK); Task 2 verify green
- **Committed in:** b93a23d (placeholders) / 541a5ab (real contracts)

**2. [Rule 3] Missing method constants blocked the corpus row (Task 1)**
- **Found during:** Task 1
- **Issue:** `zcode` had no `session/resume` / `session/close` method constants — the driver and close paths need named constants per the D-06 discipline.
- **Fix:** added MethodSessionResume/MethodSessionClose to envelope.go's dialect method map.
- **Files modified:** internal/zcode/envelope.go
- **Verification:** corpus row replays; golden recorded
- **Committed in:** b93a23d

**3. [Rule 2] WR-04 needed a named test vehicle (Task 2)**
- **Found during:** Task 2
- **Issue:** the plan's success criteria require WR-04 to land "with named tests", but no planned file carried one (no corpus row exercised a post-create failure).
- **Fix:** added the newsession-orphan-close corpus row + golden + back-tier driver, and TestNewSessionOrphanClosed over the composed harness (RED evidence 05-05-red-orphan-close).
- **Files modified:** internal/zcode/fake/testdata/scenarios/newsession-orphan-close.json, internal/zcode/fake/testdata/golden/newsession-orphan-close.golden, internal/zcode/fake/golden_test.go, internal/acpfront/lifecycle_test.go
- **Verification:** TestGolden/newsession-orphan-close + TestNewSessionOrphanClosed green
- **Committed in:** 0d9410f / 541a5ab

**4. [Rule 2] Teardown correctness additions beyond the named order (Task 2)**
- **Found during:** Task 2 GREEN
- **Issue:** the named teardown order alone leaves (a) a prompt racing close able to acquire the slot mid-teardown, (b) the cancelled turn's pending permission bridges dangling (Pitfall 2), (c) the session-cwd entry leaking, (d) an in-process reattach leaking the previous machine's observers (Register is last-wins).
- **Fix:** closing-set gate (Prompt/acquireTurn), perm.cancelPendingForTurn on an actually-cancelled turn, dropCwd, and prev.Machine.CloseObservers before re-Register.
- **Files modified:** internal/acpfront/lifecycle.go, front.go, turnctx.go
- **Verification:** TestCloseTeardown / TestCloseWhileTurnActiveRace (-race ×3) green
- **Committed in:** 541a5ab

**5. [Rule 1] createSessionID's error surface named the wrong method for resume (Task 2)**
- **Found during:** Task 2 GREEN
- **Issue:** the shared snapshot dig-out hardcoded `"session/create:"` prefixes — a resume decode failure would misname the wire method.
- **Fix:** createSessionID takes the calling method label; both call sites pass their own.
- **Files modified:** internal/acpfront/front.go
- **Verification:** suite green (no test asserted the old literal)
- **Committed in:** 541a5ab

---

**Total deviations:** 5 auto-fixed (2 × Rule 3 blockers, 2 × Rule 2 missing-critical, 1 × Rule 1 bug). **Impact on plan:** none on the contracts — every must_haves truth landed test-locked; the additions close correctness gaps the review/pitfall record names explicitly.

## Issues Encountered
- `TestGolden` fail-loud correctly rejected the WR-04 corpus row until its replay driver landed — the corpus discipline working as designed; driver added in the same task.
- The close tests' observer-baseline asserts needed the stage's own turn.started observer accounted for; final form asserts exactly the machine's (1,1) removed from the pre-close snapshot — a tighter WR-07 lock than baseline-equality.
- goconst/funlen/lll gates caught three stylistic violations mid-task (wire-key literals, acquireTurn at 61 lines, two long test strings) — all fixed; `mise run ci` green.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- SESS-01's front contracts are locked; 05-08/05-09 (which share the SESS-01 requirement id) can build on Front.ResumeSession/CloseSession and the closing-set gate.
- `Machine.CloseObservers`, `Registry.Remove`, and `closeOrphan` are the complete release surface — Phase 7's SESS-02/03 hardening extends rather than reinvents it.
- REQUIREMENTS.md's SESS-01 checkbox is intentionally untouched: the shared-ID gate reports it blocked by sibling plans 05-08/05-09 (no SUMMARYs yet); STATE.md/ROADMAP.md were not written per the dispatch instruction (orchestrator-owned).
- The plan's `estimate.tokens` (52000) vs actuals (≈22100 by chars/4): the plan over-estimated — the composed-harness reuse kept the diff compact.

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-20*
