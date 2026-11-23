---
phase: 04-acp-front-minimal-chat-loop
plan: 04
subsystem: acp-front
tags: [acp, contract-lock, streaming, thought-chunks, reasoning-deltas, flush-before-stopreason, ordering, fake-corpus, tdd, red-green]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: "plan 04-06: the tracer Front + front-tier harness (newTracerFront, tee helpers, tee direction convention); plan 04-03: the contract-lock tier this plan extends (named RED targets over the proven slice); plan 04-01: conn.AddObserver ordered fan-out (tap-before-machine registration order)"
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: "fake.Serve + CompareGolden fail-loud corpus (the back-tier replay surface), the turn machine's quiet-window settle gates (watcher.go dialect defaults) the ordering lock rides"
provides:
  - "The CORE-04 thought-chunk contract: the tap's field switch maps backend part.delta field reasoning → acpport.NoteThought (rendered as acp.UpdateAgentThoughtText by the acpport sink) — every field-reasoning delta becomes exactly ONE agent_thought_chunk in arrival order, interleaved with the text message chunks (TestThoughtChunks/positive over the part-delta-streaming corpus)"
  - "The never-synthesized negative lock (TestThoughtChunks/negative over the no-reasoning corpus): a turn with zero reasoning deltas produces ZERO thought chunks while message chunks stay intact and the prompt answers end_turn — thought chunks are backend-sourced ONLY"
  - "The CORE-03 flush-before-stopReason ordering lock (TestStreamBeforeStopReason): on RAW ACP tee lines, every chunk frame (message AND thought) strictly precedes the prompt response frame and zero session/update frames follow — proven across BOTH completion variants (terminal-event via part-delta-streaming; quiet-window via completion-quiet-window, where the response frame is the last protocol frame of the exchange)"
  - "The zero-delta honesty probe (TestStreamBeforeStopReason/zero-delta): a turn emitting no part.delta rows yields zero chunks yet answers stopReason end_turn — the completion signal alone suffices"
  - "Two new corpus rows replayable at both tiers: part-delta-streaming (text/reasoning interleaved: Answer: / Thinking about it / done) and no-reasoning (text only), each with the front preamble, drivers, and byte-deterministic goldens (corpus grows 11 → 13)"
  - "driveFrontPreamble — the ONE source of truth for the identical front-tier preamble (create → subscribe → setModel, corpus-locked synthetic selection) that cancel-mid-turn, setmodel-followup, part-delta-streaming, no-reasoning, and completion-quiet-window all drive"
affects: [04-05, phase-5-chat-cycle, phase-6-auth, phase-7-qa]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 11007 # chars/4 over the realized diff (44029 diff chars, 760 insertions / 52 deletions across 11 files)
  tasks: 2
  commits: 5 # MEASURED: git rev-list --count 45a78f8..HEAD

commits: 5
plan_head_before: 45a78f8648aef949cd00fd7b53f5c1e46e2e0eb1
plan_head_after: 1395dd1d3e44c68b302d443f38c13897301e2030

# Tech tracking
tech-stack:
  added: [] # zero new dependencies — acp-go-sdk v0.20.0 stays the module's sole require line
  patterns:
    - "Contract-lock plans carry their own corpus: the RED target needed a scenario the corpus didn't have, so the rows (part-delta-streaming, no-reasoning) landed BEFORE the RED — TestGolden green for them at the back tier first, then the front-tier RED ran over them"
    - "Quiet-window wall-clock budgeting at the front tier: the Phase-3 settle gates (floor 3x2.5s from the send ack + 2.5s quiet) put resolution at ~10s, so the front-tier quiet-window rides use a bounded 20s exchange instead of the 10s harness default — bounded but above the wall-clock floor"
    - "Preamble deduplication by corpus convention: five corpus rows now drive the identical front preamble through ONE shared driver helper (driveFrontPreamble) — goldens stayed byte-identical through the refactor, proving the byte-lock convention holds"

key-files:
  created:
    - internal/acpfront/stream_test.go
    - internal/acpfront/stream_order_test.go
    - internal/zcode/fake/testdata/scenarios/part-delta-streaming.json
    - internal/zcode/fake/testdata/scenarios/no-reasoning.json
    - internal/zcode/fake/testdata/golden/part-delta-streaming.golden
    - internal/zcode/fake/testdata/golden/no-reasoning.golden
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-04-red-thought-chunks.json
  modified:
    - internal/acpfront/stream.go
    - internal/zcode/fake/testdata/scenarios/completion-quiet-window.json
    - internal/zcode/fake/testdata/golden/completion-quiet-window.golden
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "The reasoning mapping rides the EXISTING emitter plumbing (field reasoning → acpport.NoteThought; the acpport sink renders UpdateAgentThoughtText) — zero ordering logic changed: the tap already runs first in the fan-out and emits per-envelope in arrival order, so the flush-before-stopReason property is untouched by the GREEN"
  - "The positive RED asserts the 1:1 envelope mirror (3 updates for 3 envelopes) BEFORE the kind check, so the RED failure names the dropped reasoning chunk directly — the tracer tap's skip meant only 2 of 3 envelopes surfaced"
  - "The completion-quiet-window corpus row gained the front preamble (Rule 3): the front-tier ordering lock rides this row and the front cannot open a session against a preamble-less scenario (create hit unknown_method -32601) — the same extension pattern plan 04-06 applied to cancel-mid-turn"
  - "Task 2 is an honest LOCK suite, not a manufactured RED: all three subtests green on first run is the expected contract-lock semantics (registration order + SDK FIFO writes make the ordering structural); no assertion was weakened to chase a RED"
  - "The quiet-window rides keep the Phase-3 dialect defaults (no timing knobs injected): ~10s wall clock per ride is the honest cost of proving completion WITHOUT a terminal event at the front tier"

requirements-completed: [CORE-03, CORE-04] # copied verbatim from plan frontmatter; delivery is shared across plans 04-04/04-05/04-06 — requirements.ready-ids decides what marks complete (see Next Phase Readiness)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "CORE-04 positive: every field-reasoning part.delta becomes exactly ONE agent_thought_chunk in arrival order, interleaved between the text message chunks (text 'Answer:' → thought 'Thinking about it' → text ' done'), asserted on both the client's collected updates (kind + text) and the raw tee frames (one thought frame carrying content.type text, strictly between the two message frames) — chunks mirror envelopes 1:1, never merged or split"
    requirement: CORE-04
    verification:
      - kind: integration
        ref: "internal/acpfront/stream_test.go#TestThoughtChunks/positive (composed loop over part-delta-streaming, -race; go test -race -count=1 -run TestThoughtChunks ./internal/acpfront/)"
        status: pass
    human_judgment: false
  - id: D2
    description: "CORE-04 negative — never synthesized: the no-reasoning turn yields ZERO thought chunks (collector AND raw tee) while both message chunks stay intact ('Plain', ' answer') and the prompt answers end_turn — thought chunks are backend-sourced ONLY (the D-04 honesty prohibition's enforcement test)"
    requirement: CORE-04
    verification:
      - kind: integration
        ref: "internal/acpfront/stream_test.go#TestThoughtChunks/negative (composed loop over no-reasoning, -race; go test -race -count=1 -run TestThoughtChunks ./internal/acpfront/)"
        status: pass
    human_judgment: false
  - id: D3
    description: "CORE-03 flush-before-stopReason across BOTH completion variants on raw tee lines: terminal-event (part-delta-streaming) — every chunk frame (message AND thought) strictly precedes the prompt response frame and zero session/update frames follow; quiet-window (completion-quiet-window) — completion without any terminal event still answers only after the (zero) pending updates, the response frame being the last protocol frame of the exchange"
    requirement: CORE-03
    verification:
      - kind: integration
        ref: "internal/acpfront/stream_order_test.go#TestStreamBeforeStopReason/terminal-event + /quiet-window (raw tee line indexes, -race; go test -race -count=1 -run TestStreamBeforeStopReason ./internal/acpfront/)"
        status: pass
    human_judgment: false
  - id: D4
    description: "CORE-03 zero-delta honesty probe: a turn whose send emits NO part.delta rows yields zero message chunks (collector and tee) yet still answers stopReason end_turn — the completion signal alone suffices, nothing synthesized"
    requirement: CORE-03
    verification:
      - kind: integration
        ref: "internal/acpfront/stream_order_test.go#TestStreamBeforeStopReason/zero-delta (composed loop over completion-quiet-window, -race)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Corpus grows 11 → 13 rows, all replayable at both tiers: part-delta-streaming (text/reasoning interleaved emit order byte-locked) and no-reasoning (text only) with drivers and goldens recorded; completion-quiet-window re-recorded with the front preamble; TestGolden green across the whole corpus through the real conn"
    requirement: CORE-03
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden (13 rows, fail-loud loader + driver map, -race; go test -race -count=1 -run TestGolden ./internal/zcode/fake/)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Zero regressions: full gate green — build, vet, lint 0 issues (after lint-cleaning the new files), full -race internal suite, experiments suite 22/22"
    verification:
      - kind: integration
        ref: "mise run ci (build + vet + lint + go test -race -count=1 ./... + node --test experiments/)"
        status: pass
    human_judgment: false

# Metrics
duration: 17min
completed: 2026-11-20
status: complete
---

# Phase 4 Plan 04: Streaming Contract Locks — Thought Chunks + Flush-Before-StopReason Summary

**CORE-04's honesty core lands as named tests: backend reasoning deltas now stream as agent_thought_chunk in arrival order (RED→GREEN through the tap's named skip), a zero-reasoning turn provably produces zero thought chunks, and CORE-03's flush-before-stopReason is locked on raw wire lines across BOTH completion variants plus the zero-delta probe — corpus grows to 13 rows replayable at both tiers**

## Performance

- **Duration:** 17 min
- **Started:** 2026-11-20T01:19:42Z
- **Completed:** 2026-11-20T01:37Z
- **Tasks:** 2 (Task 1 via RED→GREEN TDD with classified evidence; Task 2 an honest lock suite — green on first run by design)
- **Files:** 7 created / 4 modified

## Accomplishments
- The reasoning→thought mapping is the only production change (stream.go): the tap's field switch now maps `field:"reasoning"` → `acpport.NoteThought` — the acpport sink renders it via `acp.UpdateAgentThoughtText` (the executed-probe wire shape) — with zero ordering logic touched: the tap already runs first in the fan-out and emits exactly one chunk per envelope in arrival order
- `TestThoughtChunks/positive` locks the CORE-04 contract over the new part-delta-streaming corpus: three envelopes surface as exactly three chunks (message "Answer:" → thought "Thinking about it" → message " done"), the raw tee shows the one thought frame (content.type text, right session) strictly interleaved between the two message frames — chunks mirror envelopes 1:1, never merged, never split
- `TestThoughtChunks/negative` locks the never-synthesized prohibition over the new no-reasoning corpus: zero thought chunks on the collector AND the raw tee while both message chunks stay intact and the prompt answers end_turn
- `TestStreamBeforeStopReason` locks CORE-03/D-04 flush-before-stopReason on RAW tee lines across both completion variants: terminal-event (part-delta-streaming — every chunk frame strictly precedes the response frame, zero session/update frames after) and quiet-window (completion-quiet-window — the response frame is the LAST protocol frame of the exchange, completion without any terminal event)
- `TestStreamBeforeStopReason/zero-delta` pins the empty probe: a turn whose send emits no part.delta rows yields zero chunks yet answers stopReason end_turn — the completion signal alone suffices
- Corpus grows 11 → 13 rows: part-delta-streaming (text/reasoning interleaved emit order byte-locked) and no-reasoning (text only), each with the front preamble, registered drivers, and recorded goldens; completion-quiet-window re-recorded with the front preamble; all 13 replay green through the real conn
- The front preamble collapsed into ONE shared driver helper (`driveFrontPreamble`) — five corpus rows drive the identical create → subscribe → setModel preamble and the goldens stayed byte-identical through the refactor
- `mise run ci` fully green: build, vet, lint 0 issues, full -race internal suite (7 packages), experiments 22/22

## Task Commits

Each task committed atomically (TDD: RED commit precedes GREEN per feature):

1. **Task 1 (RED): reasoning-delta thought-chunk lock corpus + failing TestThoughtChunks** - `0b348b7` (test)
2. **Task 1 (GREEN): stream reasoning deltas as agent_thought_chunk** - `c7781ee` (feat)
3. **Task 2 (Rule 3 corpus fix): extend completion-quiet-window with the front preamble** - `e2af3ff` (test)
4. **Task 2: lock flush-before-stopReason ordering across both completion variants** - `24d78d8` (test)
5. **Lint polish: preamble dedup + lll/funlen/rangeValCopy cleanups** - `1395dd1` (style)

**Plan metadata:** committed in the close-out docs commit (this file + STATE/ROADMAP/REQUIREMENTS).

## TDD Record

One RED→GREEN cycle with a classifier-verified evidence record (`evidence/04-04-red-thought-chunks.json`, `RED_EVIDENCE_OK`, go-test adapter):

- **RED — thought chunks** (`evidence/04-04-red-thought-chunks.json`): TestThoughtChunks/positive failed on its planned zero-thought-chunks assertion — "client collected 2 updates, want exactly 3 (one chunk per part.delta envelope, never merged or split)": the tracer tap's named reasoning skip dropped the field-reasoning envelope, so only the two text deltas surfaced. The negative subtest passed simultaneously — the plan's stated correct outcome (nothing synthesizes today). **Semantic assessment:** genuine RED on the planned assertion; the failure is the missing reasoning chunk (the exact code GREEN flips), not build/fixture/discovery. One test-authoring fix happened BEFORE evidence capture (an unused `context` import — build error, removed) so the recorded RED lands only on planned assertions.
- **Corpus-first ordering:** the two scenarios + drivers + goldens landed in the same RED commit but were authored and verified green at the back tier FIRST (TestGolden for part-delta-streaming and no-reasoning replay green before the front-tier RED ran over them) — the plan's prescribed order.
- **Task 2 lock semantics (honest):** all three TestStreamBeforeStopReason subtests green on FIRST run — the expected contract-lock outcome for tracer-built behavior. No RED was manufactured and no assertion weakened; the suite's value is pinning the ordering so plans 04-05+ refactors cannot silently reorder it. 16.1s wall clock: two quiet-window settles (~7s each after the floor) plus the fast terminal-event ride.

## Files Created/Modified
- `internal/acpfront/stream.go` — the field switch: reasoning → `acpport.NoteThought` (CORE-04); input/output keep the Phase-5 skip; doc comments updated from "plan 04-04's RED target" to the landed mapping
- `internal/acpfront/stream_test.go` — TestThoughtChunks positive + negative, the kind-filter helpers, and the raw-tee interleaving lock
- `internal/acpfront/stream_order_test.go` — TestStreamBeforeStopReason (terminal-event / quiet-window / zero-delta) + assertChunksBeforeResponse; the 20s bounded exchange for the quiet-window rides
- `internal/zcode/fake/testdata/scenarios/part-delta-streaming.json` + `testdata/golden/part-delta-streaming.golden` — the CORE-04 positive row (interleaved deltas byte-locked)
- `internal/zcode/fake/testdata/scenarios/no-reasoning.json` + `testdata/golden/no-reasoning.golden` — the CORE-04 negative row
- `internal/zcode/fake/testdata/scenarios/completion-quiet-window.json` + `testdata/golden/completion-quiet-window.golden` — front preamble added (Rule 3), golden re-recorded
- `internal/zcode/fake/golden_test.go` — drivePartDeltaStreaming, driveNoReasoning, the shared driveFrontPreamble (now also driven by driveCancelMidTurn + driveSetModelFollowup), driveCompletionQuietWindow extended with the preamble

## Decisions Made
- The mapping rides the existing emitter plumbing (NoteThought through the slot's emitter closure) — no ordering change, no new synchronization: the flush-before-stopReason property was already structural (tap-before-machine registration order + SDK FIFO writes)
- The positive RED's first assertion is the 1:1 envelope count (3 updates for 3 envelopes) so the RED names the dropped reasoning chunk directly
- completion-quiet-window gained the front preamble rather than the lock suite hand-rolling a session: corpus rows stay the single scripting surface and the preamble stays byte-locked identical across all five front-tier rows
- Task 2 recorded as an honest lock (green first run), per the plan's own instruction — never manufacture RED
- Quiet-window rides keep the watcher's dialect defaults (~10s) instead of injecting faster timing knobs: the honest cost of proving completion-without-terminal-event at the front tier, bounded at 20s

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] completion-quiet-window corpus row extended with the front preamble (scenario + golden + driver not in Task 2's declared files)**
- **Found during:** Task 2 (before authoring the lock suite — the composition cannot open a session against the row)
- **Issue:** the plan's quiet-window variant rides completion-quiet-window at the front tier, but the Phase-3 row scripted only `session/send` + `session/messages` — the front's session/new contract (create → subscribe, setModel iff configured) hit `unknown_method` -32601 on session/create, so session/new could never succeed and the ride was impossible as scripted
- **Fix:** copied the front preamble rules verbatim from setmodel-followup into the scenario (same locked synthetic selection), extended driveCompletionQuietWindow to drive them through the new shared driveFrontPreamble helper, re-recorded the golden (-update) — the exact extension pattern plan 04-06 applied to cancel-mid-turn; full corpus replay stayed green and all pre-existing goldens byte-stable
- **Files modified:** internal/zcode/fake/testdata/scenarios/completion-quiet-window.json, internal/zcode/fake/testdata/golden/completion-quiet-window.golden, internal/zcode/fake/golden_test.go
- **Verification:** TestGolden green (13 rows); TestStreamBeforeStopReason/quiet-window + /zero-delta green over the extended row
- **Committed in:** e2af3ff

**2. [Rule 3 - Blocking] goconst/lll/funlen/rangeValCopy lint debt in the new files (mise run ci lint stage: 12 issues)**
- **Found during:** the wave gate after Task 2
- **Issue:** the shared preamble block appeared three times across drivers (goconst: 6 duplicate literals), and the new front-tier tests carried 5 over-length Fatalf lines, one rangeValCopy, one funlen
- **Fix:** the preamble deduplicated into driveFrontPreamble (driveCancelMidTurn and driveSetModelFollowup now drive it too — transcripts byte-identical, goldens unchanged), Fatalf lines wrapped, kindUpdates switched to indexing, testThoughtChunksPositive split (assertTextChunksIntact helper); lint 0 issues
- **Files modified:** internal/zcode/fake/golden_test.go, internal/acpfront/stream_test.go
- **Verification:** mise run ci lint stage 0 issues; TestGolden + TestThoughtChunks green after the refactor
- **Committed in:** 1395dd1

---

**Total deviations:** 2 auto-fixed (2 blocking) — none architectural, no Rule 4 decisions required.
**Impact on plan:** both fixes serve the plan's own gates (the ride-ability of the quiet-window variant; the lint-clean CI gate). Prohibitions hold: every emitted chunk is traceable to one received part.delta envelope (the negative test is its enforcement evidence), and nothing was synthesized for input/output fields or the zero-delta turn.

## Probe Ledger (from the plan)

- CORE-03 adjacency → resolved explicit: each part.delta envelope becomes exactly ONE chunk — chunks mirror envelopes 1:1 (TestThoughtChunks/positive + the 3-updates-for-3-envelopes assertion)
- CORE-03 empty → resolved explicit: zero-delta turn yields zero chunks yet answers end_turn (TestStreamBeforeStopReason/zero-delta)
- CORE-03 ordering → resolved explicit: chunk frames precede the response frame; zero update frames after — both completion variants (TestStreamBeforeStopReason/terminal-event + /quiet-window)
- CORE-04 unclassified → stays UNRESOLVED, carried as the plan's flagged assumption: "CORE-04's surface (positive mapping + negative never-synthesized) is covered by explicit tests, but the probe classifier offered no category — carried as flagged, not auto-resolved." Its enforcement evidence: TestThoughtChunks/negative.

Ledger equality holds: this plan authors 3 resolved probe truths + 1 flagged assumption; phase totals (COVERAGE.md): 10 = 7 resolved (4 in 04-03 + 3 here) + 3 flagged (CORE-02 in 04-03, CORE-04 here, CORE-05 in 04-05).

## Issues Encountered
- `go test -update <flag-before-package>` fails with "no Go files in <root>" (go test's flag-query pass defaults the package list to `.` when a custom test flag precedes the packages) — the working form is `go test -run ... ./internal/zcode/fake/ -update` (flag after the package path). Worth remembering for golden re-records.
- The quiet-window rides cost ~7s each at the front tier (floor 3x2.5s from the send ack + 2.5s settle — the watcher's dialect defaults): TestStreamBeforeStopReason runs in 16.1s under -race, safely inside its 20s bounded exchanges.

## User Setup Required

None - no external service configuration required (fake-backed composition, zero processes, no quota cost).

## Next Phase Readiness
- Plan 04-05 (the last plan of the phase) sharpens the -32010 concurrency class and wires cmd/zcode-acp (version var, supervisor lifecycle) — the streaming locks it builds on are now named tests, not tracer side effects
- Requirements CORE-03/CORE-04 are delivered by this plan (declared by 04-04 + 04-06, with 04-06 already carrying a SUMMARY): after this SUMMARY lands, `requirements.ready-ids` governs the mark-complete step per the shared-ID gate (CORE-03/04 also declared by 04-05, which has no SUMMARY yet — mark-complete ONLY for what ready-ids reports ready)
- The corpus (13 rows) is the regression perimeter for Phase 5's tool-surface work: input/output delta fields stay deliberately unimplemented with the Phase-5 skip comment in stream.go
- No blockers

---
*Phase: 04-acp-front-minimal-chat-loop*
*Completed: 2026-11-20*

## Self-Check: PASSED

All created/modified files exist on disk (9/9 checked, including this SUMMARY); all 5 task commits (0b348b7, c7781ee, e2af3ff, 24d78d8, 1395dd1) are ancestors of HEAD (1395dd1). Commit count measured from the persisted plan ledger: 5 (45a78f8..HEAD). The RED evidence record classified RED_EVIDENCE_OK by the gsd-tools gate; `mise run ci` green at HEAD (build, vet, lint 0 issues, full -race internal suite, experiments 22/22).
