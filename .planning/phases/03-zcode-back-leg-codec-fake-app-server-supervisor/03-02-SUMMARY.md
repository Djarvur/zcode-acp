---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 02
subsystem: codec
tags: [jsonl, codec, demux, wr-02, id-type-safety, single-writer, bypass-lane, stop, router, d-05, tracked-no-answer, tdd]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-01)
    provides: the tracer codec (Conn demux + two-lane writer + pending table), the in-process fake, the happy-turn golden — the working slice this plan hardens
  - phase: 02-scaffold-ci-bootstrap
    provides: WR-02 review finding (02-REVIEW) routed here by disposition; the strict lint/test gates
  - phase: 01-research-protocol-documentation
    provides: docs/protocol/app-server-dialect.md wire values (id echo verbatim §1, bypass set §1, credential-cipher §3, -32601 default §6, code-not-message matching rule §6)
provides:
  - Contract-lock tests: internal/zcode/conn_test.go (WR-02 collision, id type-safety, three-way classification, 128 KiB line buffer, malformed-line skip), writer_test.go (single-writer ordering, bypass precedence), call_test.go (verbatim-id correlation, ctx-cancellation cleanup)
  - Conn.Stop(sessionID): session/stop through the bypass lane, request-shaped — the D-02 cancel primitive the turn machine (03-05/03-08) will call
  - D-05 router policy complete: per-method injectable deadlines (default 15 s), runtime-preferences answered under its recorded deadline, credential-cipher tracked-no-answer (zero reply bytes, never a fabricated -32601) with honestly recorded expiry (OnRouterEvent hook + slog), unknown methods -32601 naming the method
  - Conn.RouterDeadline / Conn.OnRouterEvent injection surface (the Phase 5 policy-table seam)
affects: [03-03 golden corpus, 03-05 turn watcher + machine (cancel lane = conn.Stop), 03-08 turn completion proofs, 03-09 E2E gate, phase-5 interaction responder]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 11500   # ~46000 chars over this plan's own diff (internal/zcode + evidence records)
  tasks: 3
  commits: 16     # MEASURED from the plan ledger (560fab1..HEAD); 4 are this plan's task commits — the other 12 are the parallel phase-01 review-fix stream sharing the sequential branch between ledger snapshot and close-out

commits: 16
plan_head_before: 560fab1bc4903e7e633a6c5de5a4d223bc50eeec
plan_head_after: 6c3bcbe1a268066121c69ecd1a05407294094b2d

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: tidy-diff green, go.mod untouched)
  patterns:
    - "Contract-lock tests drive the public Conn surface over an io.Pipe peer: io.Pipe's synchronous writes make injections return only once the read loop consumed the line, and hook channels make deliveries deterministic — negative assertions use a bounded quiet window, never instant checks"
    - "WR-02 locked behaviorally: a method-bearing frame with a numeric id colliding with 7 in-flight call ids routes to OnRequest and never resolves the pending table"
    - "Bypass precedence locked deterministically: submit stop while io.Pipe holds the writer blocked on the in-flight frame — the stop lands first or second on the wire, never third"
    - "Tracked-no-answer: the tracker goroutine IS the registry entry — it derives from the conn's base context (closure field, like serveRequest) so the deadline can actually elapse after the per-request handler returns"
    - "D-05 record-honesty made observable: RouterEvent{Kind: answered|deadline-expired, Method, IDKey, Deadline, Err} — answers record their governing deadline, expiries record exactly once and are never swallowed"

key-files:
  created:
    - internal/zcode/conn_test.go
    - internal/zcode/writer_test.go
    - internal/zcode/call_test.go
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-02-red-router-credential-cipher.json
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-02-red-router-runtime-preferences.json
  modified:
    - internal/zcode/conn.go
    - internal/zcode/call.go
    - internal/zcode/envelope.go

key-decisions:
  - "Id-type conflation test locked on an integer-keyed pending call (foreign-type string reply no-op, own-type echo resolves): the plan sketch's string-keyed pending call is unreachable through the public Conn surface — Call only assigns integer ids — and the type-tagged i:/s: keys make the two formulations behaviorally equivalent"
  - "Runtime-preferences deadline observability: the static answer is immediate, so the governing deadline travels on the RouterAnswered event (default 15 s asserted via RouterDefaultDeadline, injectability asserted via a 40 ms override) — the deadline is policy metadata until Phase 5 replaces the static answer with real computation under the same bound"
  - "The tracked-no-answer entry is a tracker goroutine bound to the conn's base context (closure field trackRequest): the per-request context is cancelled when the handler goroutine returns, so deriving from it would cut every tracked deadline short; base-context derivation keeps 'dies with the conn without recording' semantics honest"
  - "conn.Stop is fire-and-forget with a conn-assigned id (request-shaped per dialect §2) and the ack reply deliberately unobserved — cancellation must not block on wire I/O and the turn machine watches completion signals, not the stop ack (D-02)"
  - "Test params decode via maps rather than camelCase struct tags where wire keys are multi-word (sessionId) — tagliatelle enforces snake_case tags and the wire dictates the key spelling"

patterns-established:
  - "Contract-lock test naming after the finding (WR-02 in the doc comment; grep-count 1 for the pinned test name)"
  - "Slow-peer writer tests exploit io.Pipe synchronous blocking instead of wall-clock racing (RESEARCH State-of-the-Art row)"
  - "TDD RED with declaration surface: the failing tests compile against new exported fields/types/constants only, so the RED failure is behavioral (go-test adapter classified RED_EVIDENCE_OK twice)"

requirements-completed: [BRIDGE-01]  # COPIED VERBATIM from 03-02-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-01: 03-01/03-02/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "WR-02 structural demux guarantee test-locked: method-bearing frames with colliding numeric ids route to the request router and never resolve the pending table"
    requirement: BRIDGE-01
    verification:
      - kind: integration
        ref: "internal/zcode/conn_test.go#TestDemuxReverseRequestNeverResolvesPendingCall"
        status: pass
    human_judgment: false
  - id: D2
    description: "Id correlation by exact value AND type: cross-type replies are no-ops, own-type echoes resolve; concurrent calls each receive exactly their own reply; ctx cancellation deregisters cleanly (late reply dropped)"
    requirement: BRIDGE-01
    verification:
      - kind: unit
        ref: "internal/zcode/conn_test.go#TestDemuxIdTypeNonConflation + internal/zcode/call_test.go#TestCallReplyEchoesIDVerbatim,#TestCallCtxCancellationCleanup"
        status: pass
    human_judgment: false
  - id: D3
    description: "Three-way demux classification (reply / reverse request / notification land on exactly one path) and raised-line behavior (128 KiB line intact; malformed line skipped with a logged warning, stream continues in wire order)"
    requirement: BRIDGE-01
    verification:
      - kind: unit
        ref: "internal/zcode/conn_test.go#TestDemuxClassification,#TestLineBufferLargeLineIntact,#TestLineBufferMalformedLineBetweenValidLines"
        status: pass
    human_judgment: false
  - id: D4
    description: "Single-writer ordering (32 concurrent Calls, zero interleaved frames, count exact) and bypass precedence (stop on the wire within the first two frames while two calls are queued) plus the conn.Stop bypass helper"
    requirement: BRIDGE-01
    verification:
      - kind: unit
        ref: "internal/zcode/writer_test.go#TestWriterOrdering,#TestStopJumpsQueue"
        status: pass
    human_judgment: false
  - id: D5
    description: "D-05 router policy complete and honest: runtime-preferences answered under its injectable deadline (default 15 s) with the deadline recorded; credential-cipher tracked-no-answer producing zero reply bytes and exactly one recorded expiry per deadline; unknown methods -32601 naming the method"
    requirement: BRIDGE-01
    verification:
      - kind: unit
        ref: "internal/zcode/conn_test.go#TestRouterCredentialCipherNeverAnsweredExpiryRecorded,#TestRouterRuntimePreferencesDeadline,#TestRouterUnknownMethodAnswered32601"
        status: pass
    human_judgment: false

# Metrics
duration: 23min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 02: Codec Hardening — Contract Locks + D-05 Router Policy Summary

**The tracer's working round-trip is now a contract-locked codec: WR-02 demux, id type-safety, single-writer ordering and bypass precedence locked by named tests, and the D-05 router policy complete — runtime preferences answered on an injectable deadline, the mid-turn credential-cipher request tracked-but-never-answered with honestly recorded expiry — all green under -race with zero new dependencies**

## Performance

- **Duration:** 23 min
- **Started:** 2026-11-19T18:03:23Z
- **Completed:** 2026-11-19T18:26:46Z
- **Tasks:** 3
- **Files:** 8 (3 test files + 3 source files created/modified, 2 RED evidence records)

## Accomplishments

- The four transport guarantees are test-locked under -race at the public Conn surface over an io.Pipe peer: WR-02 (a method-bearing frame with a numeric id colliding with 7 in-flight call ids routes to OnRequest and never resolves the pending table), id type-safety (a cross-type reply is a no-op; only the verbatim own-type echo resolves), three-way demux classification, and the raised line buffer (128 KiB line intact; malformed line skipped with a logged warning while the stream continues in wire order)
- Writer contracts locked: TestWriterOrdering proves 32 concurrent Calls produce complete, never-interleaved frames with exact per-call correlation; TestStopJumpsQueue proves the stop frame lands first or second on the wire while two calls are queued — deterministically, via io.Pipe's synchronous blocking, no wall-clock racing
- conn.Stop(sessionID) shipped: session/stop through the bypass lane, request-shaped with a conn-assigned id, ack deliberately unobserved — the D-02 cancel primitive 03-05/03-08 will call
- D-05 router policy complete via TDD (RED evidence RED_EVIDENCE_OK on both targets before the feat commit): session/requestRuntimePreferences answered with the static defaults under its injectable per-method deadline (default 15 s) with the governing deadline recorded on the answered event; interaction/requestCredentialCipher tracked-no-answer — zero reply bytes (the RED run proved the tracer's -32601 default violated this, the GREEN run proves silence), exactly one recorded expiry per deadline (OnRouterEvent hook + structural slog warning), tracker dies silently with the conn; unknown methods still -32601 naming the method
- Full gates green: mise run ci (build → vet → lint 0 issues → test -race all packages → test-experiments 22/22), tidy-diff green (go.mod untouched — T-3-SC clean)

## Task Commits

Each task was committed atomically:

1. **Task 1: Demux contract locks (WR-02, id types, classification, line buffer) + call correlation tests** - `3eef389` (test)
2. **Task 2: Writer ordering + bypass precedence locks; conn.Stop bypass helper** - `a4c631b` (test)
3. **Task 3 RED: failing router deadline + credential-cipher honesty tests (evidence RED_EVIDENCE_OK ×2)** - `f67e21d` (test)
4. **Task 3 GREEN: router per-method deadlines + tracked-no-answer credential cipher** - `6c3bcbe` (feat)

## Files Created/Modified

- `internal/zcode/conn_test.go` - WR-02 collision test, id type non-conflation, three-way classification, 128 KiB line buffer, malformed-line skip with log assertion, router policy tests + the shared io.Pipe/hook test harness
- `internal/zcode/call_test.go` - verbatim-id correlation under concurrency, ctx-cancellation cleanup (late reply dropped, table stays clean)
- `internal/zcode/writer_test.go` - single-writer ordering (32 concurrent calls), bypass precedence (stop jumps the queue), stop-frame shape assertion
- `internal/zcode/conn.go` - router policy table (answerRuntimePreferences under ctx deadline, trackUnanswered on the base-context closure field, deadlineFor, recordDeadlineExpired, recordRouterEvent), RouterDeadline/OnRouterEvent fields, RouterEvent types, RouterDefaultDeadline = 15 s
- `internal/zcode/call.go` - Conn.Stop(sessionID) through the bypass lane
- `internal/zcode/envelope.go` - MethodRequestCredentialCipher constant (verbatim dialect §3)
- `.planning/phases/03-.../evidence/03-02-red-router-credential-cipher.json` + `03-02-red-router-runtime-preferences.json` - persisted RED evidence, both classified RED_EVIDENCE_OK by the go-test adapter

## TDD Record (Task 3)

- **RED:** `go test -race -count=1 -run TestRouter ./internal/zcode/` with the declaration surface only (new exported fields/types/constants, routing unchanged) — TestRouterCredentialCipherNeverAnsweredExpiryRecorded failed on its own planned assertion (the tracer's -32601 default drew reply bytes: a D-05 violation on the wire), TestRouterRuntimePreferencesDeadline failed on its own planned assertion (no answered event recorded); the existing-behavior lock TestRouterUnknownMethodAnswered32601 and the rest of the suite stayed green. Both records classified **RED_EVIDENCE_OK** (`target_test_failed`, go-test format adapter) before the GREEN commit; semantic assessments recorded inside the evidence JSONs
- **GREEN:** the routeRequest policy table + deadline tracking made both targets pass; the full package (including the fake golden replay) stayed green

## Decisions Made

- Id-type conflation locked on an integer-keyed pending call (foreign-type no-op, own-type resolves): the sketch's string-keyed pending call is unreachable via the public surface (Call only assigns integer ids); the i:/s: type-tagged keys make the formulations behaviorally equivalent
- Runtime-preferences deadline is observable through the RouterAnswered event's Deadline field (default asserted = 15 s; injectability asserted with a 40 ms override) — a static answer is instantaneous, so the bound is policy metadata until Phase 5
- The tracked-no-answer entry derives from the conn's BASE context (closure field `trackRequest`, mirroring `serveRequest`): the per-request context is cancelled when its handler goroutine returns, which would have cut every tracked deadline short — found by the GREEN run's own test and fixed under Rule 1
- conn.Stop is fire-and-forget (request-shaped, ack unobserved): cancellation must not block on wire I/O; the turn machine watches completion signals, not the stop ack (D-02)
- Test params with multi-word wire keys decode via maps, not camelCase struct tags (tagliatelle enforces snake_case; the wire dictates the key spelling)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Tracked-no-answer deadline never elapsed — tracker derived from the dying per-request context**
- **Found during:** Task 3 GREEN (first run of TestRouterCredentialCipherNeverAnsweredExpiryRecorded)
- **Issue:** the tracker goroutine's context derived from the per-request context, which serveRequest cancels when its handler returns — the tracked deadline was cut short (Canceled, not DeadlineExceeded) and the expiry was never recorded
- **Fix:** trackRequest closure field derives the tracker from the conn's base context (same containedctx-clean pattern as serveRequest); routeRequest dispatches credential-cipher through it
- **Files modified:** internal/zcode/conn.go
- **Verification:** TestRouterCredentialCipherNeverAnsweredExpiryRecorded green under -race (×3 package repeats)
- **Committed in:** 6c3bcbe

**2. [Rule 3 - Blocking] Strict lint-gate failures on the new test/helper code (5 findings)**
- **Found during:** Task 2 (mise run lint gate is a Task-2 acceptance criterion)
- **Issue:** err113 (dynamic error in a test assertion), gocritic sprintfQuotedString (`"%s"` inside a Sprintf'd wire line), revive context-as-argument (ctx not first param on issueCalls/concurrentCalls), tagliatelle (camelCase `sessionId` struct tag)
- **Fix:** static errCorrelationBroken sentinel wrapped via %w; `"blob":%q`; ctx promoted to first parameter; stop-params decode via map
- **Files modified:** internal/zcode/conn_test.go, internal/zcode/call_test.go, internal/zcode/writer_test.go
- **Verification:** mise run lint — 0 issues
- **Committed in:** a4c631b

**3. [Rule 3 - Blocking] err113 on direct error comparison in the GREEN router code (2 findings)**
- **Found during:** Task 3 GREEN (lint gate)
- **Issue:** `ctx.Err() == context.DeadlineExceeded` — err113 requires errors.Is for error comparison
- **Fix:** `errors.Is(ctx.Err(), context.DeadlineExceeded)` at both sites (+ errors import)
- **Files modified:** internal/zcode/conn.go
- **Verification:** mise run lint — 0 issues; package green under -race
- **Committed in:** 6c3bcbe

### Plan-text adaptations (not code defects)

- **TestDemuxIdTypeNonConflation shape:** the plan sketch specifies a string-keyed pending call receiving an integer-id reply; no public Conn API creates a string-keyed pending call. Locked the equivalent contract on an integer-keyed call (foreign-type string reply no-op, own-type echo resolves) — acceptance criterion "foreign type no-op, own type resolves" met exactly
- **call_test.go landed in Task 1's commit:** the plan's task <files> lists don't name it, but the frontmatter artifacts spec does (correlation tests), and Task 1's done criterion ("demux and correlation contracts test-locked") covers it

---

**Total deviations:** 3 auto-fixed (1 Rule 1 bug, 2 Rule 3 lint-blocking) + 2 documented plan-text adaptations
**Impact on plan:** all fixes required to satisfy the tasks' own acceptance criteria; no scope creep; no checkpoints consumed

## Issues Encountered

- A parallel executor stream landed 12 phase-01 review-fix commits on the shared sequential branch between this plan's ledger snapshot and close-out (visible as interleaved `fix(01)`/`docs(01)` commits). No overlap with this plan's files (experiments/ + phase-01 planning docs); the four 03-02 commits are self-contained. The measured ledger commit count (16) therefore exceeds this plan's own 4 task commits — both numbers recorded in frontmatter `actuals`
- The credential-cipher RED target surfaced the -32601 default as the D-05 violation on the wire, exactly as the plan's fails_when predicted — captured verbatim in the persisted evidence record

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03-03 (golden corpus) can script credential-cipher scenarios against a router that now tracks them honestly; the fake's reverse-answer path is untouched
- Plan 03-05/03-08 (turn machine): conn.Stop is the cancel primitive; the OnRouterEvent hook is the seam Phase 5's full interaction-responder policy table extends (RouterDeadline(method) already per-method)
- The runtime-preferences deadline is policy metadata on the answered event until Phase 5 replaces the static answer — the injectable bound is already wired
- BRIDGE-01 completion remains intentionally pending: declared by 03-01/03-02/03-09, so requirements.ready-ids correctly returns [] until 03-09's SUMMARY lands

## Threat Surface Scan

No new security-relevant surface beyond the plan's threat model: T-3-03 mitigations (WR-02 demux rule, type-tagged correlation, raised-but-capped line buffer) are now test-locked; T-3-SC holds (zero dependencies, tidy-diff green). Router logging stays structural (method names, canonical id keys, durations — never params).

## Self-Check: PASSED

- 03-02-SUMMARY.md exists on disk
- All claimed files exist: conn_test.go, call_test.go, writer_test.go, conn.go, call.go, envelope.go, 2 evidence records (FOUND ×8)
- All claimed commits are ancestors of HEAD: 3eef389 (Task 1), a4c631b (Task 2), f67e21d (Task 3 RED), 6c3bcbe (Task 3 GREEN)
- Acceptance greps: TestDemuxReverseRequestNeverResolvesPendingCall ×1 in conn_test.go; MethodSessionStop ×1 in call.go
- RED evidence classified RED_EVIDENCE_OK (×2 records) before the GREEN commit
- Final post-task run: mise run ci fully green (build, vet, lint 0 issues, go test -race all packages, test-experiments 22/22)

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
