---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 01
subsystem: testing
tags: [jsonl, codec, json-rpc, dialect, fake-app-server, golden-transcripts, tdd, tracer]

# Dependency graph
requires:
  - phase: 01-research
    provides: app-server-dialect.md (verbatim 3.18.2 envelope grammar, method map, error codes) + EXP-WORKSPACEKEY-01/EXP-TURN-01 capture shapes
  - phase: 02-scaffold
    provides: strict golangci gate (mise run lint/ci/tidy-diff) + repo Go module skeleton
provides:
  - internal/zcode codec core: ID (string-or-int, i:/s: canonical key), Envelope (no framing key by construction), WireError, method + §6 error-code constants
  - Conn: read loop with raised Scanner buffer, three-way demux (reply / reverse-request / notification, WR-02 structural), single-writer two-lane output stage (ordered queue + bypass), pending-call table, router minimum (runtime-preferences static answer + -32601 default)
  - internal/zcode/fake scripting core: Serve over any io.ReadWriteCloser, declarative Scenario schema, blocking reverse requests with expect_answer validation, honest unanswered-reverse transcript notes, EOF drain
  - Golden replay: CompareGolden (-update records, missing fixture FAILS) + happy-turn.golden (11 frames, byte-deterministic)
  - TestHappyTurnRoundTrip: the tracer proof — full dialect round-trip over io.Pipe under -race
affects: [03-02 hardening, 03-03 golden corpus, 03-06 cmd/fake-app-server, 03-07 supervisor spawn, 03-08 turn proofs, 03-09 E2E gate]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 17000
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: go.mod still declares only the module line)
  patterns:
    - "Two-lane single-writer output stage (ordered queue + bypass) per dialect §1 scheduling"
    - "Three-way demux with the WR-02 structural rule: method-bearing frames never enter the pending table"
    - "Type-tagged canonical id correlation (i:/s: keys — string and integer ids never collide)"
    - "Fake rule ordering: side effects run before the rule's reply (wire truth; makes delivery race-free)"
    - "Golden transcripts: direction-tagged wire lines, fixed ids, fail-loud missing fixture, -update re-record"

key-files:
  created:
    - internal/zcode/fake/testdata/golden/happy-turn.golden
  modified:
    - internal/zcode/conn.go
    - internal/zcode/call.go
    - internal/zcode/envelope.go
    - internal/zcode/writer.go
    - internal/zcode/fake/fake.go
    - internal/zcode/fake/roundtrip_test.go

key-decisions:
  - "Fake rule ordering: side effects run in order BEFORE the reply — the behavior block mandates it for create (the runtime-preferences exchange precedes the create reply) and EXP-TURN-01 records the same order for send (state patch before the ack); it also makes notification delivery race-free (notifications precede their request's reply on the same read loop, so the test's post-ack snapshot is deterministic)"
  - "Per-method reverse-request deadline VALUE + expiry recording deferred to plan 03-02 exactly as the plan's Task 3 text assigns; Task 3 ships the request-context plumbing (base context closure-captured in NewConn, cancelled on Close)"
  - "expect_answer validated semantically — decoded-JSON deep compare so key order never matters (the codec marshals maps sorted; captures author payloads in wire order); a mismatch fails the run (the scenario assertion is the fixture's point)"
  - "containedctx compliance: the conn's base request context is closure-captured in NewConn and never stored as a struct field"
  - "Transcript Line.JSON records the wire line WITHOUT its newline terminator for both directions (identical treatment; the golden format is <dir>\\t<line>\\n)"

patterns-established:
  - "Demux classification predicate: isReply = id present AND no method AND (result or error) — the RESEARCH Pattern 1 rule that fixes the ARCHITECTURE sketch's looser first case (PATTERNS.md correction honored)"
  - "Reverse-request handling: per-request goroutine + bypass-lane reply, never inline in the read loop (ARCHITECTURE Anti-Pattern 3)"
  - "Reverse answers resolved through a pending table keyed by canonical id; timeouts and stream-close orphans recorded as DirNote transcript lines (record-honesty: never normalized away)"
  - "Library-layer diagnostics: slog to stderr, structural fields only (method names, id keys, byte sizes) — prompt content never logged (plan prohibition)"

requirements-completed: [BRIDGE-01, BRIDGE-02]  # COPIED VERBATIM from 03-01-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-01: 03-01/03-02/03-09; BRIDGE-02: 03-01/03-03/03-06/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Conn read loop: raised Scanner buffer (1 MiB initial / 16 MiB cap), per-line decode, malformed lines skipped with structural slog warning"
    requirement: BRIDGE-01
    verification:
      - kind: integration
        ref: "internal/zcode/fake/roundtrip_test.go#TestHappyTurnRoundTrip"
        status: pass
    human_judgment: false
  - id: D2
    description: "Three-way demux: reply frames resolve the pending table, method-bearing frames dispatch to per-request goroutines (WR-02 structural), notifications deliver on the hook in wire order"
    requirement: BRIDGE-01
    verification:
      - kind: integration
        ref: "internal/zcode/fake/roundtrip_test.go#TestHappyTurnRoundTrip (notification wire-order + reverse-request assertions)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Router minimum (D-05): session/requestRuntimePreferences answered with the static defaults {nativeSearchEnhancementsEnabled:false, memoryEnabled:false}; unknown methods -32601"
    requirement: BRIDGE-01
    verification:
      - kind: integration
        ref: "internal/zcode/fake/roundtrip_test.go#TestHappyTurnRoundTrip (create answered via the runtime-preferences exchange)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Pending-call resolution by type-tagged canonical id (resolve: one reply per id, deregister-before-deliver, unknown ids dropped)"
    requirement: BRIDGE-01
    verification:
      - kind: unit
        ref: "internal/zcode/roundtrip_test.go#TestIDKeyTypeTagging"
        status: pass
    human_judgment: false
  - id: D5
    description: "fake.Serve scripting core: scenario matching, ordered side effects (emit + blocking reverse with expect_answer validation), unknown-method reply, EOF drain, honest unanswered-reverse notes"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/roundtrip_test.go#TestHappyTurnRoundTrip"
        status: pass
    human_judgment: false
  - id: D6
    description: "Golden replay: happy-turn.golden fixture (11 wire frames) byte-deterministic on every run; drift or missing frames FAIL; -update re-records; missing fixture fails loud"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/roundtrip_test.go#TestGoldenHappyTurn"
        status: pass
    human_judgment: false

# Metrics
duration: 30min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 01: Happy-Turn Dialect Round-Trip (Tracer) Summary

**Full app-server dialect round-trip proven end-to-end: codec Conn (two-lane writer, WR-02 demux, runtime-preferences router) vs scripted fake app-server over io.Pipe, with an 11-frame byte-deterministic golden replay — all under -race with zero new dependencies**

## Performance

- **Duration:** 30 min (spanning three executor sessions — see Issues Encountered)
- **Started:** 2026-11-19T16:53:00Z (Task 1 RED commit)
- **Completed:** 2026-11-19T17:23:00Z
- **Tasks:** 3
- **Files modified:** 12 (10 code/test + 2 fixtures incl. RED evidence record)

## Accomplishments
- The tracer slice works end-to-end: TestHappyTurnRoundTrip drives a real zcode.Conn against fake.Serve over io.Pipe — session/create answered after the scripted reverse requestRuntimePreferences exchange (answer validated against the scenario's expect_answer), session/send acked accepted:true, five scripted notifications delivered on the hook in exact wire order — green under -race (x3)
- Conn demux implements the WR-02 structural rule: only result/error frames resolve the pending table (type-tagged canonical id keys); method-bearing frames dispatch to per-request goroutines and are answered on the bypass lane; the router answers runtime-preferences with the verbatim EXP-WORKSPACEKEY-01 defaults and everything else with -32601
- fake.Serve: the full scripting core — scenario table matching, ordered side effects (emit notifications; blocking reverse requests with semantic expect_answer validation and honest unanswered notes), verbatim id echo, single-writer sink with boundary-crossing transcript order, EOF drain window with clean-EOF classification
- Golden replay live: testdata/golden/happy-turn.golden (11 frames, direction-tagged) reproduces byte-deterministically; TestGoldenHappyTurn fails loud on drift or a missing fixture
- Full gates green: go test -race ./internal/zcode/... all green; mise run ci green (lint 0 issues); tidy-diff green — go.mod still declares only the module line (zero new dependencies, T-3-SC clean)

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — failing end-to-end happy-turn round-trip test + compiling codec/fake skeletons + scenario 1** - `394dcbb` (test)
2. **Task 2: GREEN — envelope, two-lane writer, pending-call table (codec leaf layer)** - `acfa17e` (feat)
3. **Task 3: GREEN — conn read loop + demux + router minimum + fake Serve + golden replay** - `4bcae69` (feat)

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

**Plan metadata:** pending (docs commit lands with this SUMMARY)

## Files Created/Modified
- `internal/zcode/conn.go` - Read loop (Scanner 1 MiB/16 MiB), three-way demux, router minimum, base-context plumbing, Close joins the read loop
- `internal/zcode/call.go` - pendingTable.resolve (one reply per canonical id, deregister-before-deliver)
- `internal/zcode/envelope.go` - (Task 1/2) ID/Envelope/WireError, Encode/Decode, method + error-code constants
- `internal/zcode/writer.go` - (Task 1/2) single-writer two-lane loop per dialect §1
- `internal/zcode/roundtrip_test.go` - (Task 1/2) leaf tests: id key tagging, framing-key omission, id-type echo, byte-exact params
- `internal/zcode/fake/fake.go` - Serve scripting core: scan loop, handler goroutines, side effects, reverse blocking, transcript recording
- `internal/zcode/fake/scenario.go` - (Task 1) declarative Scenario schema + fail-loud loader
- `internal/zcode/fake/golden.go` - (Task 1) CompareGolden: -update records, missing fixture FAILS
- `internal/zcode/fake/roundtrip_test.go` - the tracer RED target (Task 1) + transcript assertion fix + TestGoldenHappyTurn (Task 3)
- `internal/zcode/fake/testdata/scenarios/happy-turn.json` - (Task 1) scenario 1 fixture, fixed ids sess_fix1/turn_fix1
- `internal/zcode/fake/testdata/golden/happy-turn.golden` - recorded happy-turn transcript, 11 frames
- `.planning/phases/03-.../evidence/03-01-red-happy-turn.json` - (Task 1) persisted RED evidence, RED_EVIDENCE_OK

## Decisions Made
- Fake rule ordering — side effects run in order BEFORE the rule's reply: the behavior block mandates it for create (the reverse exchange precedes the create reply) and EXP-TURN-01 records the same order for send (state patch at :35 before the ack at :36). Side benefit: notification delivery becomes race-free (notifications precede their request's reply on the same read loop, so the post-ack snapshot in the test is deterministic)
- Per-method deadline VALUE + expiry recording deferred to plan 03-02 exactly as the plan's Task 3 text assigns ("the deadline VALUE and its expiry recording land in plan 03-02 — here ship the answer path with a context deadline plumbed from the Conn's request context"); Task 3 ships the plumbing (base context, per-request derivation, cancelled on Close)
- expect_answer validated semantically (decoded-JSON deep compare): the codec marshals maps sorted while scenario payloads preserve authored order, so byte comparison would false-fail on key order; a real payload mismatch still fails the run
- The conn's base request context is closure-captured in NewConn (not a struct field) — containedctx-clean; per-request contexts derive from it and die with Close
- Transcript lines record the wire line without its newline terminator in both directions, making the golden format (<dir>\t<line>\n) byte-clean

## Deviations from Plan

### Execution note — interruption + continuation
The prior executor completed Tasks 1-2 (commits 394dcbb, acfa17e) and was interrupted mid-Task-3. This plan was continued by a fresh executor (itself resumed once after a harness concurrency limit — no work lost, no duplicate commits). The dispatch briefing's "uncommitted Task 3 residue" was stale: git status showed all Task 1-2 work already committed; Task 3 started from the committed conn/fake skeletons (read loop absent, Serve returning errSkeleton). The failing deadline-exceeded test state named in the briefing was reproduced exactly and fixed under TDD (the RED for this task's chain was Task 1's committed evidence; no new RED commit per the dispatch contract).

### Auto-fixed Issues

**1. [Rule 1 - Bug] Transcript assertion filtered out the reverse request by direction**
- **Found during:** Task 3 (running the committed RED target past the deadline stage)
- **Issue:** assertTranscript skipped all non-DirIn lines before looking for the session/requestRuntimePreferences frame — but the fake issues that request itself, which travels fake→host = DirOut per the direction contract committed in Task 1's fake.go skeleton ("DirOut ... the fake's own reverse requests"). The assertion was unsatisfiable as written: sawReverseRequest could never become true
- **Fix:** scan ALL transcript lines for the reverse-request/scope and answer assertions; keep the JSON-validity + no-framing-key checks scoped to DirIn lines only (the plan's behavior block: "every recorded host→server line parses as JSON and contains no JSON-RPC framing key")
- **Files modified:** internal/zcode/fake/roundtrip_test.go
- **Verification:** TestHappyTurnRoundTrip green under -race (x3)
- **Committed in:** 4bcae69 (Task 3 commit)

**2. [Rule 1 - Bug] DirOut transcript lines recorded with the trailing newline, DirIn without**
- **Found during:** Task 3 (first golden recording showed a blank line after every fake→host frame)
- **Issue:** fake write() recorded zcode.Encode's full output (frame + '\n') while the DirIn path records the scanner's newline-stripped bytes — inconsistent Line.JSON content polluted the golden fixture with blank lines
- **Fix:** write() records line[:len(line)-1] (the wire line without the terminator); Line doc updated; golden re-recorded
- **Files modified:** internal/zcode/fake/fake.go, internal/zcode/fake/testdata/golden/happy-turn.golden
- **Verification:** golden replay byte-identical across -count=3 -race runs
- **Committed in:** 4bcae69 (Task 3 commit)

**3. [Rule 3 - Blocking] Strict lint gate failures on the new code (4 issues)**
- **Found during:** Task 3 (mise run lint)
- **Issue:** containedctx flagged the context.Context field on Conn; err113 flagged the two dynamic fmt.Errorf mismatch errors in matchExpectAnswer; unparam flagged write()'s always-DirOut direction parameter
- **Fix:** base context closure-captured in NewConn (handleRequest takes ctx, serveRequest closure derives it); errReverseMismatch sentinel wrapped via %w in both mismatch errors; write() drops the dir parameter and records DirOut directly (inbound/note lines go through record())
- **Files modified:** internal/zcode/conn.go, internal/zcode/fake/fake.go
- **Verification:** mise run lint — 0 issues
- **Committed in:** 4bcae69 (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bugs, 1 Rule 3 lint-blocking) + the execution note above (process, not a code deviation)
**Impact on plan:** All fixes were required to make the committed test's own assertions satisfiable and the golden byte-clean. No scope creep; the plan's acceptance criteria are all met.

## Issues Encountered
- Three executor sessions spanned this plan (prior executor interrupted mid-Task-3; this executor resumed once under a harness concurrency limit). No work was lost or duplicated — commit history 394dcbb → acfa17e → 4bcae69 is linear and each task committed exactly once
- The plan's Task 3 action text ("write the configured reply ... run sideEffects in order") read literally contradicts the feature's behavior block ("before the create reply the fake issued session/requestRuntimePreferences ... and observed the host answer") for the create rule. Resolved in favor of the behavior block + capture wire truth (side effects before reply) — see Decisions Made. The committed Task-1 test only passes under this semantics, confirming it is the intended spec
- go test custom flags (-update) must be passed AFTER the package list; passing them before silently mis-resolves the package pattern ("no Go files" / "FAIL ."). Mechanical, noted for future golden recordings

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 03-02 (demux/writer/router hardening + deadline tracking) builds directly on the seams this plan shipped: routeRequest is the deadline site; writer error recording and read-loop EOF handling are the hardening surfaces; the oversized-line stream-continuation nuance (Scanner cannot resume past a >16 MiB line) is the known hardening item
- Plan 03-03 (golden corpus) expands the fake via the Scenario schema as-is — the ten scenarios need no fake changes
- cmd/fake-app-server (03-06) wraps Serve over stdio; Serve's clean-EOF classification already covers io.EOF, io.ErrClosedPipe, and os.ErrClosed (exec pipes)
- BRIDGE-01/BRIDGE-02 completion is intentionally pending: both are declared by later plans (03-02/03-09 and 03-03/03-06/03-09 respectively) and mark-complete fires only when every declaring plan has a SUMMARY

## Self-Check: PASSED

- 03-01-SUMMARY.md exists on disk
- internal/zcode/fake/testdata/golden/happy-turn.golden exists (1 fixture, >= 1 required)
- All claimed commits are ancestors of HEAD: 394dcbb (Task 1 RED), acfa17e (Task 2 GREEN), 4bcae69 (Task 3 GREEN), 28c498d (SUMMARY), ced0797 (tracking)
- Ledger-measured plan commit count: 5 (3 task commits at SUMMARY write + 2 docs commits)
- Final post-close-out run: go test -race -count=1 ./internal/zcode/... — all green

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
