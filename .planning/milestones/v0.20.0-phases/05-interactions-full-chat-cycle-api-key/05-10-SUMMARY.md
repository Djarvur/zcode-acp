---
phase: 05-interactions-full-chat-cycle-api-key
plan: 10
subsystem: acpfront
tags: [session-lifecycle, permission-bridge, jsonrpc, cancel-fan-in, tdd, gap-closure]
requires:
  - phase: 05-interactions-full-chat-cycle-api-key (plans 05-02..05-09)
    provides: the permission bridge, SESS-01 close/resume handlers, the D-05 cancel fan-in, the turn slot, the composed harness, the golden corpus
provides:
  - drainActiveTurn sawTurn contract — the close path fans permission pendings in exactly when a turn of the session was cancelled (CR-01 closed; the 05-02 prohibition restored on the close path)
  - Handlers.ResumeSession(ctx, sessionID, cwd) seam — resume carries and records the client cwd; absolute-cwd validation before any backend call (WR-02 closed)
  - resume gated on the closing set at entry AND before Register — no machine observer leak (WR-03 closed)
  - replyRequest structural nil-nil silence — the parked dedupe-join arm is silent by construction at conn close (WR-01 closed)
  - prompt fan-in pump-identity re-check + Front.Cancel slot-gated fan-in (WR-04 closed; the CR-01 asymmetry gone from both cancel paths)
  - 7 new lock tests; close-with-pending-permission coverage hole closed
affects: [05-11 (same-wave sibling compiles the shared realdesktop package union), 05-12 (REAL RUN UAT retry — the CR-01 gate this plan discharges), CORE-06 verification]
actuals:
  tokens: 12933    # chars/4 over the realized diff (51733 diff bytes, 963 changed lines)
  tasks: 3
  commits: 7
tech-stack:
  added: []
  patterns:
    - "sawTurn report pattern: a drain helper reports whether work was OBSERVED (not whether the wait timed out) so the caller's fan-in gate fires on the right path"
    - "structural wire silence: protocol-shape validation at the reply funnel (a JSON-RPC response carries exactly one of result/error) replaces race-dependent invariants"
    - "per-turn identity capture + re-check under the slot lock for async cancel arms"
    - "slog-capture discriminator for fan-in asserts where wire-absence asserts race the teardown window"
key-files:
  created: []
  modified:
    - internal/acpfront/lifecycle.go (drainActiveTurn sawTurn; ResumeSession cwd + closing gates)
    - internal/acpfront/front.go (WithCloseTurnWait seam; fan-in identity re-check; Cancel slot gate; turnPumpFor)
    - internal/acpfront/lifecycle_test.go (close-with-pending, deadline-arm, resume cwd/closing tests)
    - internal/acpfront/cancel_test.go (slot-gate fan-in, stale-arm containment)
    - internal/acpport/types.go (ResumeSession seam widened)
    - internal/acpport/agent.go (cwd forwarded; unconvert fix)
    - internal/acpport/agent_test.go (cwd forwarding locked; scripted double widened)
    - internal/zcode/conn.go (replyRequest structural nil-nil silence)
    - internal/zcode/conn_test.go (TestReplyStructuralSilence + silenceStage)
    - internal/interact/responder.go (join-arm comment names the structural guarantee)
    - test/realdesktop/apicycle_test.go (resumeLeg passes the workspace cwd)
key-decisions:
  - "drainActiveTurn returns sawTurn (observed-and-cancelled) per the review's fix shape verbatim; the deadline arm returns true (a turn WAS observed — fan-in best-effort)"
  - "WithCloseTurnWait Option added as the deadline-arm test seam (precedent: WithStreamCeiling / interact.WithMethodDeadline) — a real 10s test or a new corpus scenario were the alternatives"
  - "WR-01 fixed at the reply funnel (replyRequest), not a sentinel error type — the smallest structural skip; externally testable through the OnRequest seam"
  - "WR-04 uses the per-turn pump-pointer capture + re-check under turnMu (the plan's alternative shape) — acquireTurn's signature stays untouched (turnctx.go is outside files_modified)"
  - "TestCancelFanInOnlyWhenSlotHeld discriminates on the fan-in's structural log: wire-absence asserts race the WR-01 teardown window (the -32021 attempt at close loses to the writer stop almost always but not certainly)"
  - "resume rejection message family mirrors the prompt gate: 'the session is closing or closed' (invalid-request class)"
patterns-established:
  - "Close-path and cancel-path permission fan-ins share ONE gate semantics: a turn of THIS session holds the slot (f.turnActive && f.turnSession == sessionID)"
  - "RED tests written against the stable client surface when the fix widens an interface; the seam lands in GREEN with its forwarding lock"
requirements-completed: [CORE-06, SESS-01, TOOL-02]
coverage:
  - id: D1
    description: "Closing a session whose active turn holds a pending permission ask cancels the pendings — the backend observes the -32021 cancelled arm (CR-01 closed)"
    requirement: CORE-06
    verification:
      - kind: integration
        ref: "internal/acpfront/lifecycle_test.go#TestCloseWithPendingPermissionCancelled"
        status: pass
      - kind: integration
        ref: "internal/acpfront/lifecycle_test.go#TestCloseTeardown"
        status: pass
    human_judgment: false
  - id: D2
    description: "drainActiveTurn's deadline arm reports sawTurn=true and the fan-in fires best-effort past the wait"
    requirement: CORE-06
    verification:
      - kind: integration
        ref: "internal/acpfront/lifecycle_test.go#TestCloseDeadlineArmFansPendingIn"
        status: pass
    human_judgment: false
  - id: D3
    description: "Resume carries the client cwd end-to-end; a resumed session's first turn resolves absolute tool diff paths; relative/empty cwd rejected -32602 pre-wire (WR-02 closed)"
    requirement: SESS-01
    verification:
      - kind: integration
        ref: "internal/acpfront/lifecycle_test.go#TestResumeRecordsCwd"
        status: pass
      - kind: integration
        ref: "internal/acpfront/lifecycle_test.go#TestResumeRejectsRelativeCwd"
        status: pass
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestLifecycleDelegation/resume_delegates_the_session_id_and_cwd"
        status: pass
    human_judgment: false
  - id: D4
    description: "Resume racing a close lands exactly one coded closing/closed rejection with no machine observer leak (WR-03 closed)"
    requirement: SESS-01
    verification:
      - kind: integration
        ref: "internal/acpfront/lifecycle_test.go#TestResumeRejectedWhileClosing"
        status: pass
    human_judgment: false
  - id: D5
    description: "replyRequest nil-nil writes no frame; both genuine arms deliver; the parked join arm still parks on a live conn (WR-01 closed; reannounce-dedupe golden replays green)"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/zcode/conn_test.go#TestReplyStructuralSilence"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake#TestGolden/reannounce-dedupe"
        status: pass
      - kind: other
        ref: "grep -n 'return nil, nil, true' internal/interact/responder.go → exactly one hit (line 246, the parked join)"
        status: pass
    human_judgment: false
  - id: D6
    description: "The prompt fan-in skips past the turn boundary (pump identity re-check) and Front.Cancel fans pendings in only under the this-session slot gate (WR-04 closed)"
    requirement: TOOL-02
    verification:
      - kind: integration
        ref: "internal/acpfront/cancel_test.go#TestCancelFanInOnlyWhenSlotHeld"
        status: pass
      - kind: integration
        ref: "internal/acpfront/cancel_test.go#TestCancelFanInStaleArmSkips"
        status: pass
      - kind: integration
        ref: "internal/acpfront/cancel_test.go#testFrontSlotRejection (existing genuine displaced arm)"
        status: pass
    human_judgment: false
  - id: D7
    description: "EDGE-PROBE backstops (5 rows carried from must_haves — flagged, never auto-dismissed): CORE-06/adjacency (equal-key dedupe merge untouched — unverified by new tests), CORE-06/empty (no covering empty-input probe at the conn tier beyond the two-arm lock), CORE-06/ordering (fan-out registration-order contract untouched), TOOL-02/unclassified (the exactly-once outcome union beyond the -32021 close arm), SESS-01/unclassified (resume-recheck window locked only by the composed test, no dedicated interleaving probe)"
    verification: []
    human_judgment: true
    rationale: "The plan's must_haves flag these five probe rows as UNRESOLVED backstop truths for the verifier — deliberately unverified by this plan's tests; the verifier must classify them."
duration: 50min
completed: 2026-11-21
status: complete
plan_head_before: 2c614949473b51fc6e180a17e07319ec1275c9e1
plan_head_after: 456364e
commits: 7
---

# Phase 5 Plan 10: Gap Closure — CR-01 Close-Path Fan-In + WR-01..04 Batch Summary

**Close-path permission fan-in fires on the successful-drain arm (CR-01), resume carries the client cwd and is closing-gated (WR-02/03), the dedupe-join silence is structural at conn close (WR-01), and both cancel paths fan pendings in under one slot-identity gate (WR-04) — each red-first locked.**

## Performance

- **Duration:** 50 min
- **Started:** 2026-11-21T07:27:11Z
- **Completed:** 2026-11-21T08:16:56Z
- **Tasks:** 3 (all TDD: RED commit → GREEN commit each)
- **Files modified:** 11 of the plan's 12 (`internal/interact/responder_test.go` needed no change — the existing join/dedupe tests stayed green unmodified)

## Accomplishments

- **CR-01 closed:** `drainActiveTurn` reports `sawTurn` (true on the successful-drain arm AND the deadline arm; false only when no turn of the session ever ran) — `CloseSession`'s `cancelPendingForTurn` now fires exactly when a turn was cancelled. The close-with-pending-permission lock test (zero coverage before, grep-confirmed) drives the real bridge through the composed harness and asserts the -32021 cancelled reply on the backend transcript.
- **WR-02 closed:** `Handlers.ResumeSession(ctx, sessionID, cwd)` — the adapter forwards `ResumeSessionRequest.Cwd` verbatim, the front validates `filepath.IsAbs` BEFORE any backend call (-32602, the T-4-01 parity guard) and records the cwd once `createSessionID` confirms; the resumed session's first turn resolves absolute tool diff paths.
- **WR-03 closed:** resume is gated on the closing set at entry and re-checked immediately before `Register` — the composed test proves no registration happens mid-close (observer counts return to exactly the machine's removed entries).
- **WR-01 closed:** `replyRequest` drops a neither-result-nor-error answer with a structural log (id key only) — the bare `{"id":X}` echo the parked join arm framed in the `cancelBase→rw.Close` window is unframmable; the responder's join-arm comment names the structural guarantee (grep-confirmed: exactly one `(nil, nil, true)` return — the parked join).
- **WR-04 closed:** Prompt's cancel fan-in captures the turn's pump identity and re-checks `f.turnPump` under `turnMu` before `machine.Cancel` (skips silently past the boundary); `Front.Cancel` observes the slot BEFORE cancelling and fans the pendings in only when a turn of THIS session holds it — the CR-01 asymmetry is gone from both cancel paths.
- **Gates:** `go build ./...`, `go vet ./internal/...`, all `-race` suites, `mise run lint` (0 issues), `mise run ci` — all green; `go build -tags realdesktop ./...` + tagged vet green (apicycle resumeLeg updated for the widened seam).

## Task Commits

Each task was committed atomically (RED → GREEN per task):

1. **Task 1 (CR-01):** `b2c8aaa` (test: failing close-with-pending lock) → `e0b2c74` (feat: sawTurn + deadline-arm seam)
2. **Task 2 (WR-02/WR-03):** `134cfe0` (test: failing resume cwd/closing tests) → `14b4a53` (feat: seam widening + gates)
3. **Task 3 (WR-01/WR-04):** `6b18a20` (test: failing conn silence + slot-gate tests) → `1d3b8a1` (feat: structural skip + identity re-check + Cancel gate)
4. **Lint conformance:** `456364e` (style: strict lint 0 issues across the fixes)

## CR-01 Closure Evidence (the disposition-ledger REAL RUN gate)

- Lock test: `TestCloseWithPendingPermissionCancelled` (internal/acpfront/lifecycle_test.go) — close mid-pending → the -32021 cancelled reply on the backend transcript, teardown order intact (stop precedes the one close).
- `grep -c sawTurn internal/acpfront/lifecycle.go` → **4** (baseline HEAD was 0).
- 05-REVIEW-DISPOSITION.md rows CR-01, WR-01, WR-02, WR-03, WR-04 are discharged by this plan (the ledger file itself is the reviewer's to flip).

## Files Created/Modified

- `internal/acpfront/lifecycle.go` — drainActiveTurn sawTurn contract + closeWait seam read; ResumeSession cwd validation/record + entry/pre-Register closing gates
- `internal/acpfront/front.go` — `WithCloseTurnWait` Option + `closeWaitOverride` field; fan-in pump-identity capture/re-check; `turnPumpFor`; Cancel slot gate
- `internal/acpfront/lifecycle_test.go` — TestCloseWithPendingPermissionCancelled, TestCloseDeadlineArmFansPendingIn, TestResumeRecordsCwd, TestResumeRejectsRelativeCwd, TestResumeRejectedWhileClosing + assert helpers
- `internal/acpfront/cancel_test.go` — TestCancelFanInOnlyWhenSlotHeld, TestCancelFanInStaleArmSkips + newSlotStage/safeLogBuffer helpers
- `internal/acpport/types.go` — ResumeSession seam widened (doc names the SDK Cwd MUST-be-absolute contract)
- `internal/acpport/agent.go` — cwd forwarded verbatim
- `internal/acpport/agent_test.go` — cwd forwarding locked (direct + raw-wire paths)
- `internal/zcode/conn.go` — replyRequest structural nil-nil silence
- `internal/zcode/conn_test.go` — TestReplyStructuralSilence + silenceStage
- `internal/interact/responder.go` — join-arm comment: structural guarantee
- `test/realdesktop/apicycle_test.go` — resumeLeg(t, ctx, front, sid, ws)

## Decisions Made

- The review's sawTurn fix shape applied verbatim; the teardown ORDER in CloseSession untouched (Cancel before the bounded wait; the second drain re-check unchanged).
- Deadline-arm testability via `WithCloseTurnWait` (deviation 1 below) instead of a real 10s wait or a new scenario+golden (outside files_modified).
- WR-04 via the plan's alternative shape (per-turn token field captured right after acquisition) — `turnctx.go` stays out of the diff entirely.
- The ResumeRecordsCwd test rides the full-cycle row (the only row with BOTH a resume rule and a relative-path tool card); the session/new→resume→prompt flow resolves the tool card against the RESUMED cwd.

## Deviations from Plan

### Auto-fixed / Documented Adjustments

**1. [Rule 3 - Blocking] WithCloseTurnWait test seam added (front.go)**
- **Found during:** Task 1 GREEN
- **Issue:** The plan's `<behavior>` requires the deadline arm locked ("a turn still active past closeTurnWait also reports sawTurn=true"), but `closeTurnWait` is a package const (10s) — a real-deadline test needs a >10s turn, and no corpus row produces one (the quiet settle resolves any turn at ~7.5s; a new scenario+golden is outside files_modified).
- **Fix:** `WithCloseTurnWait` Option + `closeWaitOverride` field (precedent: `WithStreamCeiling`, `interact.WithMethodDeadline`); production wiring never sets it. The deadline-arm test shrinks the wait to 1ns, making the elapsed-deadline arm deterministic.
- **Files modified:** internal/acpfront/front.go (Task 3's file — within the plan's 12-file scope), internal/acpfront/lifecycle.go, internal/acpfront/lifecycle_test.go
- **Verification:** TestCloseDeadlineArmFansPendingIn green under -race
- **Committed in:** e0b2c74

**2. [Refinement - test drive] TestCancelFanInOnlyWhenSlotHeld stages the slot-free state on ONE session**
- **Found during:** Task 3 RED
- **Issue:** The plan's drive ("a pending ask belonging to another session's turn survives") needs two registry-distinct sessions — impossible over the fake: every corpus create returns the same deterministic backend id (sess_fix1), so a second session/new overwrites the same registry entry.
- **Fix:** The identical slotHeld=false skip arm staged deterministically: the SDK's second-prompt displacement cancels turn 1 (whose fan-in arm never fans pendings), releasing the slot with the ask dangling; the slot-free `front.Cancel` (exported seam, driven directly) must skip. The discriminator is the fan-in's structural log (synchronous inside the cancel call) because a wire-absence assert races the WR-01 teardown window; the surviving ask's honest unanswered note is also locked.
- **Files modified:** internal/acpfront/cancel_test.go
- **Verification:** RED at HEAD (the unconditional fan-in logged count=1), GREEN after the gate — deterministic across repeated -race runs
- **Committed in:** 6b18a20 (RED) / 1d3b8a1 (GREEN)

**3. [Refinement - test drive] TestCancelFanInStaleArmSkips locks the reachable containment contract**
- **Found during:** Task 3 GREEN
- **Issue:** The HEAD bug's harmful interleaving (a descheduled fan-in arm waking past release + re-acquisition) is a pure scheduler race with no deterministic drive through the public surface: the arm's own turn context closes inside release, so an sdkCtx cancellation arriving after the boundary always finds the goroutine already reaped. A test asserting turn 2 uncancelled after a post-boundary sdkCtx cancel passes at HEAD too — it cannot serve as RED.
- **Fix:** The committed test drives the deterministic interleaving that IS reachable (displaced genuine arm → later turn completes untouched, exactly one stop) and documents the drive limitation in-file; the skip arm's identity semantics are locked deterministically by TestCancelFanInOnlyWhenSlotHeld (same `f.turnPump`/`f.turnSession` gate shape). Task 3's deterministic RED evidence comes from TestReplyStructuralSilence + TestCancelFanInOnlyWhenSlotHeld.
- **Files modified:** internal/acpfront/cancel_test.go
- **Verification:** green under -race; the genuine displaced-prompt cancel stays locked by the existing testFrontSlotRejection
- **Committed in:** 1d3b8a1

---

**Total deviations:** 3 (1 blocking-fix seam, 2 documented test-drive refinements). **Impact on plan:** the fix shapes match the review verbatim; the deviations are test-infrastructure only. No scope creep; 11 of 12 files touched.

## Issues Encountered

- **Task 3 RED drives needed empirical grounding:** the slot-gate test went through three constructions (two-session — impossible over the corpus's fixed ids; SDK-client cancel — the notification path swallowed the drive; direct `front.Cancel` — worked). The displaced-turn staging and the slog discriminator came out of the transcript/slog evidence gathered along the way; the final test is deterministic across repeated `-race` runs.
- The coupling note (05-11 same-wave realdesktop build) never fired: every tagged build/vet was green throughout.

## TDD Gate Compliance

| Gate | Evidence |
|------|----------|
| RED | 3 `test(05-10):` commits (b2c8aaa, 134cfe0, 6b18a20), each validated by `gsd_run check tdd-red-evidence` → `RED_EVIDENCE_OK` (5 records: 1 + 3 + 2 targets; all target tests failed on their planned assertions, never on build/setup) |
| GREEN | 3 `feat(05-10):` commits (e0b2c74, 14b4a53, 1d3b8a1), each descending from its RED commit (ancestry verified) |
| REFACTOR | none needed — `456364e` is lint conformance (style), behavior unchanged, suites green |

## Verification Log

- Task 1: `go test -race -run 'TestCloseWithPendingPermissionCancelled' ./internal/acpfront/` PASS; regression set (TestCloseTeardown|TestCloseWhileTurnActiveRace|TestPermissionCancel|TestPermissionCancelRace) PASS; sawTurn grep 4 (baseline 0, expected ≥1)
- Task 2: `go test -race -run 'TestResume' ./internal/acpfront/` PASS (TestResume + 3 new); `go test -race ./internal/acpport/` PASS; `go build -tags realdesktop ./...` + tagged vet PASS; cwd-seam grep 1 (baseline 0, expected ≥1)
- Task 3: `go test -race -run 'TestReplyStructuralSilence' ./internal/zcode/` PASS; `go test -race -run 'TestCancelFanInStaleArmSkips|TestCancelFanInOnlyWhenSlotHeld|TestCancelMidTurn|TestCancelAfterCompleteNoOp' ./internal/acpfront/` PASS; `TestGolden/reannounce-dedupe` PASS; `go test -race ./internal/interact/` PASS; Serve-path grep: exactly one `(nil, nil, true)` (responder.go:246, the parked join)
- Plan gate: `go build ./... && go vet ./internal/... && go test -race -count=1 ./internal/... ./cmd/...` — all green
- Wave gate: `mise run ci` — green (build → vet → lint 0 issues → test → tidy)

## Known Stubs

None — no placeholder values, no unwired data paths, no skipped tests.

## EDGE-PROBE Backstops Carried (for the verifier — flagged, never auto-dismissed)

1. **CORE-06/adjacency** — the dedupe-join silence fix must not change the equal-key merge semantics (equal (method, requestId) still merges into ONE entry answered once); unverified by any new test in this plan.
2. **CORE-06/empty** — replyRequest's nil-nil skip has no covering empty-input probe at the conn tier beyond the two-arm lock (TestReplyStructuralSilence).
3. **CORE-06/ordering** — the fan-out registration-order delivery contract untouched; no new ordering lock authored here.
4. **TOOL-02/unclassified** — the permission-bridge outcome exactly-once union (selected vs -32021, never both, never neither) is not re-locked by the close-path tests beyond the -32021 close arm.
5. **SESS-01/unclassified** — close-vs-resume ordering is gated (WR-03) but the resume-recheck-before-Register window is locked only by the composed TestResumeRejectedWhileClosing, not a dedicated interleaving probe.

Both prohibitions carried: bridge-never-outlives-turn and join silence — now test-locked at the composed/conn tiers; the 05-12 REAL RUN re-run remains the live-behavior proof.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The CR-01 REAL RUN UAT gate (05-REVIEW-DISPOSITION.md) is discharged: the fix is in and test-covered; 05-12 owns the REAL RUN retry.
- 05-11 (same wave) may compile this plan's test files freely — all shared-package suites are green at `456364e`.
- IN-01..03 (realdesktop model-shape fail-loud, raw-capture path guard, shutdown select re-check) remain open as the hardening backlog — out of this plan's scope by the disposition ledger.

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-21*

## Self-Check: PASSED

- All 7 task/lint commits verified as ancestors of HEAD (b2c8aaa, e0b2c74, 134cfe0, 14b4a53, 6b18a20, 1d3b8a1, 456364e) + the SUMMARY/docs commit (5d6c1d7).
- All 11 touched files exist on disk (responder_test.go intentionally unmodified — its suite stayed green unmodified).
- Plan gate + wave gate (`mise run ci`) green at authoring time; all `<verify>` commands logged above with their results.
