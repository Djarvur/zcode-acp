---
phase: 05-interactions-full-chat-cycle-api-key
plan: 04
subsystem: tool-streaming
tags: [tool-01, wr-03, in-05, wr-06, tool-call, tool-updated, diff, coalescing, backpressure, pump, turn-slot, fake, golden-corpus]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key
    provides: acpport neutral Handlers seam + connection-bound sink/sender bindings, front harness composition, corpus preamble rows, 05-01..05-03
  - phase: 04-acp-front-minimal-chat-loop
    provides: the part.delta tap (stream.go), the front-wide turn slot (turnctx.go acquireTurn), flush-before-stopReason locks, Prompt/release ordering
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: scripted fake + golden corpus machinery, turn machine terminal/idle signals, record-on-first-sight turnId discipline
provides:
  - acpport neutral ToolNote vocabulary (NoteTool + ToolNote{ToolCallID, IsStart, Title, Name, Kind, Status, Diff{Path,OldText,NewText}, Locations, RawInput, RawOutput} + ToolKind*/ToolStatus* wire-string constants) with the sink tool arms (StartToolCall/UpdateToolCall/ToolDiffContent translation, patch semantics, oldText omission for new files, rawInput verbatim)
  - acpfront stream.go tool.updated classifier (scheduled/started/progress/result/error → notes; batch/raw + subagent/background mirrors log-only; unknown kinds warn structurally; the locked toolName→ToolKind case-insensitive table w/ other default; Edit/Write diff recipes; session-cwd-absolute path resolution)
  - IN-05 turn-id guard: the turn slot records the backend turnId record-on-first-sight; foreign-turnId part.delta AND tool.updated envelopes never reach the client
  - WR-06: the prompt-content drop site (acpport joinContentBlocks) warns structurally with kinds and counts only
  - WR-03 per-turn bounded pump (turnctx.go): non-blocking enqueue, byte-budgeted queue drained at wire pace by a per-turn goroutine, keep-latest coalescing for same-toolCallId patches only, fail-loud ceiling (injectable via acpfront.WithStreamCeiling, default 64 MiB) surfaced as the coded -32603 prompt error, bounded release (2s) closing the post-return write window with the flush-before-stopReason property inside release
  - corpus: tool-lifecycle.json + golden (front preamble + two send rules: the Edit lifecycle with batch/subagent-mirror/foreign-delta fixtures, the Write lifecycle); tests: TestToolNote, TestToolLifecycle, TestStreamBackpressure, driveToolLifecycle, composeBackLeg, TestUnsupportedPromptBlocksWarnStructural
affects: [05-05 session lifecycle teardown (turn slot/pump release joins), 05-07 cmd responder wiring, 05-09 real tier, Phase-6 configOptions surface]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 25961    # chars/4 over the realized diff (103845 chars; plan estimated 54000)
  tasks: 3
  commits: 6       # MEASURED: git rev-list --count gsd-plan-head-before-05-04..HEAD
commits: 6
plan_head_before: f74a92d5aa06b889cc4e293a37c7f9947cbcec94
plan_head_after: ba9d3629a637e1322f05ff7887c3f18bdd91ad9f

# Tech tracking
tech-stack:
  added: []        # zero new dependencies (locked discipline; go list -m all unchanged)
  patterns:
    - per-turn pump goroutine over a byte-budgeted queue: the read goroutine classifies and enqueues non-blocking, the pump drains to the ACP wire at wire pace (the WR-03 cross-pump liveness coupling is designed away)
    - keep-latest coalescing scoped to tool_call_update patches of one toolCallId, replacing in place so every other frame's relative order is untouched
    - fail-loud backpressure: ceiling breach records a coded error, cancels the turn ctx, and Prompt gates on the breach BEFORE mapping the machine outcome (a turn that dropped streamed output never answers a success stopReason)
    - bounded release: slot clear → turn ctx cancel → bounded pump wait — drains-or-discards, never hangs
    - record-on-first-sight turnId in the turn slot (machine.go discipline mirrored front-side) with foreign-envelope suppression
    - option-order SDK helper composition: locations before rawInput so WithStartLocations' rawInput.path backfill is wholly overwritten (raw input rides verbatim)

key-files:
  created:
    - internal/acpfront/tool_stream_test.go
    - internal/zcode/fake/testdata/scenarios/tool-lifecycle.json
    - internal/zcode/fake/testdata/golden/tool-lifecycle.golden
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-04-red-tool-note.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-04-red-tool-lifecycle.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-04-red-backpressure.json
  modified:
    - internal/acpport/types.go
    - internal/acpport/sink.go
    - internal/acpport/agent.go
    - internal/acpport/agent_test.go
    - internal/acpfront/stream.go
    - internal/acpfront/turnctx.go
    - internal/acpfront/front.go
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "WR-06's warn lands at the PHYSICAL drop site (acpport/agent.go joinContentBlocks), not stream.go as the must_haves artifact line loosely suggested — the drop site is where image/audio/resource blocks leave the joined turn text; the warn carries kinds and counts only (Rule 3, documented deviation)"
  - "Coalescing is unconditional at enqueue (any queued same-id tool patch is replaced keep-latest), so TestToolLifecycle leg 1 locks ORDER + terminal-content invariants (start card, terminal completed patch with rawOutput + re-emitted diff, lossless text delta, foreign absence) instead of an exact intermediate-frame count — the byte-level wire ORDER stays golden-locked at the fake tier (Rule 1 reconciliation of the plan's two truths)"
  - "The ceiling breach fails the turn even when the machine already resolved completed — an incomplete stream must never ride a success stopReason"
  - "SDK discovery: the ToolCallContent union marshals the diff variant INLINE (no \"diff\" wrapper key) — the Task-1 RED expected frames were corrected to the true SDK helper wire shape on the first GREEN run (recorded-tee byte-lock discipline)"
  - "RawInput rides VERBATIM: the sink appends the locations option BEFORE the raw input so the SDK helper's rawInput.path backfill is wholly overwritten (the sink transports, never augments)"

patterns-established:
  - "Backpressure contract: read-goroutine-never-parks / tool-patches-coalesce / text-thought-lossless / ceiling-fails-loud — four named legs, each test-locked (TestStreamBackpressure a/b/c + the Phase-4 streaming locks re-run)"
  - "Back-leg-only harness (composeBackLeg): fake ⇄ conn ⇄ Front with the adversarial sink injected at the Front.Prompt boundary — no ACP leg needed to prove backend-leg liveness"
  - "Golden-locked corpus + composed-tier semantic locks: the fake golden byte-locks the backend emit order; the composed harness locks the client-visible invariants"

requirements-completed: [TOOL-01]

coverage:
  - id: D1
    description: "Neutral ToolNotes translate onto the SDK tool vocabulary with exact wire shapes: the start frame carries toolCallId+title+name+kind+status pending, the diff content, and the location, with rawInput verbatim; update frames carry only populated fields (patch semantics); the empty-kind and payload-less arms fail loud"
    requirement: TOOL-01
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestToolNote (4-row byte-exact wire table + loud defaults)"
        status: pass
      - kind: unit
        ref: "internal/acpport/ full suite -race (existing note suites unchanged-green)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Backend tool.updated events stream to the client as ACP tool_call/tool_call_update with the locked mapping (title=description else toolName, kind per the case-insensitive table w/ other default, status transitions pending→in_progress→completed, rawOutput verbatim, Edit diff at the session-cwd-ABSOLUTE path, locations carried), every emitted frame traceable to one received envelope — batch and subagent-mirror envelopes produce no client cards"
    requirement: TOOL-01
    verification:
      - kind: integration
        ref: "internal/acpfront/tool_stream_test.go#TestToolLifecycle leg 1 (ordered sequence, mapping, absolute path, log-only mirrors)"
        status: pass
      - kind: e2e
        ref: "internal/zcode/fake/golden_test.go#TestGolden/tool-lifecycle (backend emit order byte-locked)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Write-tool inputs render the new-file diff shape: newText carried, oldText ABSENT on the raw wire (decoded OldText nil), path resolved absolute against the session cwd"
    requirement: TOOL-01
    verification:
      - kind: integration
        ref: "internal/acpfront/tool_stream_test.go#TestToolLifecycle leg 2 (assertWriteStartCard + assertWriteStartWire)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Foreign-turnId deltas never reach the client (IN-05): the turn slot records the backend turnId record-on-first-sight and the tap skips both part.delta and tool.updated envelopes of other turns"
    requirement: TOOL-01
    verification:
      - kind: integration
        ref: "internal/acpfront/tool_stream_test.go#TestToolLifecycle leg 1 assertForeignSuppression (FOREIGN absent from tee and updates; same-turn text lossless)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Unsupported prompt content blocks (image/audio/embedded resource) warn structurally at the drop site with kinds and counts only — the block content never appears in the log record; the text baseline joins intact"
    requirement: TOOL-01
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestUnsupportedPromptBlocksWarnStructural (mutex-guarded log capture, no-leak assert)"
        status: pass
    human_judgment: false
  - id: D6
    description: "WR-03 backend-leg liveness: with the sink parked on a gate, the machine still consumes the terminal event and Prompt resolves end_turn WHILE the gate is closed — the read goroutine never parks on the ACP write"
    requirement: TOOL-01
    verification:
      - kind: integration
        ref: "internal/acpfront/tool_stream_test.go#TestStreamBackpressure leg a (stalled-sink liveness)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Same-toolCallId tool updates coalesce keep-latest behind a parked drain: of tc1's in_progress/in_progress/completed patches only the LAST (completed) reaches the client; text/thought deltas stay lossless and the Phase-4 chunk locks stay green"
    requirement: TOOL-01
    verification:
      - kind: integration
        ref: "internal/acpfront/tool_stream_test.go#TestStreamBackpressure leg b + TestStreamBeforeStopReason + TestThoughtChunks"
        status: pass
    human_judgment: false
  - id: D8
    description: "The byte ceiling (injectable via WithStreamCeiling, default 64 MiB) fails the turn LOUD: Prompt returns the coded -32603 error naming the ceiling cause, zero notes delivered past the breach, never a hang"
    requirement: TOOL-01
    verification:
      - kind: integration
        ref: "internal/acpfront/tool_stream_test.go#TestStreamBackpressure leg c (coded error, zero delivered)"
        status: pass
    human_judgment: false
  - id: D9
    description: "The tool-lifecycle corpus row replays deterministically through the composed harness and the fake golden: scheduled→started→progress→result emit order byte-locked; Prompt's response frame lands after every streamed frame"
    requirement: TOOL-01
    verification:
      - kind: e2e
        ref: "internal/zcode/fake/golden_test.go#TestGolden/tool-lifecycle + assertToolFramesBeforeResponse"
        status: pass
    human_judgment: false
  - id: D10
    description: "The pump preserves arrival order for text/thought and closes Prompt's post-return write window (bounded release wait) — the Phase-4 streaming locks (chunk-before-response, exactly-one chunk per delta) stay green"
    requirement: TOOL-01
    verification:
      - kind: integration
        ref: "internal/acpfront/stream_order_test.go#TestStreamBeforeStopReason (all three subtests) under -race after the pump landed"
        status: pass
    human_judgment: false

# Metrics
duration: 48min
completed: 2026-11-20
status: complete
---

# Phase 5 Plan 4: Tool-Call Streaming (TOOL-01) Summary

**TOOL-01 end-to-end: the tap extends to the tool.updated discriminated union feeding a neutral acpport ToolNote vocabulary whose sink arms render exact ACP tool_call/tool_call_update frames — titles, kinds, status transitions, Edit/Write diffs at session-cwd-absolute paths — with IN-05's turn-id guard, WR-06's structural drop-site warn, and WR-03's designed per-turn bounded pump (coalescing, fail-loud ceiling, bounded release) replacing the inline-write coupling; locked by TestToolNote, TestToolLifecycle, TestStreamBackpressure and the tool-lifecycle corpus row + golden, all -race green over the composed harness.**

## Performance

- **Duration:** 48 min
- **Started:** 2026-11-20T11:45:58Z
- **Completed:** 2026-11-20T12:33:42Z
- **Tasks:** 3 (all TDD: RED evidence records captured per task, RED_EVIDENCE_OK each)
- **Files:** 6 created / 8 modified (fixture-weighted: corpus row + golden ARE the TOOL-01 test surface)

## Accomplishments

- acpport neutral ToolNote vocabulary (NoteTool + the prescription field set + ToolKind*/ToolStatus* wire constants) with the sink tool arms: StartToolCall/UpdateToolCall/ToolDiffContent translation, patch semantics carrying only populated fields, oldText omitted for new files, rawInput riding verbatim (locations-before-rawInput option order defeats the SDK helper's path backfill)
- The tap classifies the tool.updated discriminated union: scheduled → the start note (title=description else toolName, the locked case-insensitive kind table with the other default, Edit/Write diff recipes, cwd-absolute locations); started/progress → in_progress patches (tails held structurally, v1-minimal); result → completed + rawOutput verbatim + the scheduled diff re-emitted; error → failed; batch/raw + subagent/background mirrors stay log-only; unknown kinds warn and never synthesize
- IN-05 landed in the same files: the turn slot records the backend turnId record-on-first-sight (machine.go discipline mirrored front-side, plus the session-cwd and per-turn diff registry); foreign-turnId part.delta AND tool.updated envelopes never reach the client
- WR-06: the prompt-content drop site warns with kinds and counts only, content never logged
- WR-03's deliberate design (the OQ3 disposition): a per-turn pump goroutine drains a byte-budgeted queue at wire pace — the backend read loop never parks (leg a proves it: the machine resolves end_turn while the sink gate is closed); same-toolCallId patches coalesce keep-latest (leg b: exactly one surviving tc1 patch); the injectable ceiling (WithStreamCeiling, default 64 MiB) fails the turn loud with a coded -32603 error naming the cause (leg c); release closes Prompt's post-return write window with a bounded wait and the flush-before-stopReason property inside it
- The tool-lifecycle corpus row + golden byte-lock the backend emit order; TestToolLifecycle locks the client-visible composed-tier invariants; all three RED evidence records captured with RED_EVIDENCE_OK
- Wave gate `mise run ci` green (build → vet → lint 0 issues → test -race → test-experiments); full `go test -race -count=1 ./internal/...` green; zero new dependencies

## Task Commits

Each task committed atomically (RED test commit precedes the GREEN feat commit per task):

1. **Task 1 RED: acpport ToolNote vocabulary skeleton + TestToolNote** - `302c0df` (test)
2. **Task 1 GREEN: sink tool arms with exact wire shapes** - `e71b2ae` (feat)
3. **Task 2 RED: TestToolLifecycle + tool-lifecycle corpus row (no golden yet)** - `b8eaf9e` (test)
4. **Task 2 GREEN: tap classification + IN-05 guard + WR-06 warn + golden** - `e3f748b` (feat)
5. **Task 3 RED: TestStreamBackpressure (stalled-sink/coalescing/ceiling) + WithStreamCeiling skeleton** - `fb82883` (test)
6. **Task 3 GREEN: WR-03 per-turn bounded pump** - `ba9d362` (feat)

**Plan metadata:** (this SUMMARY via the gsd-tools commit verb; STATE/ROADMAP/REQUIREMENTS are the orchestrator's writes per this dispatch)

_Note: TDD tasks produced exactly 2 commits each (RED → GREEN); no REFACTOR-only commits were needed — the lint-driven extractions rode their task's GREEN commit._

## TDD Gate Compliance

- **Task 1 RED:** TestToolNote failed on the first staged assert (the sink's not-wired skeleton returned the coded error before any frame) — `gsd_run check tdd-red-evidence` → **RED_EVIDENCE_OK** (target_test_failed). GREEN passes all four wire rows byte-exactly plus the loud defaults. RED commit `302c0df` precedes GREEN `e71b2ae`.
- **Task 2 RED:** TestToolLifecycle failed both legs (leg 1 got [message message] — zero tool frames AND the foreign delta leaking, the IN-05 defect itself; leg 2 zero starts); TestGolden/tool-lifecycle failed on the missing fixture. → **RED_EVIDENCE_OK**. GREEN passes every staged assert and records the deterministic golden. RED `b8eaf9e` precedes GREEN `e3f748b`.
- **Task 3 RED:** TestStreamBackpressure failed all three legs (stalled sink: prompt never resolved — the read loop parked, the reviewed WR-03 defect reproduced; coalescing: [in_progress in_progress completed] delivered verbatim; ceiling: end_turn with nil error from the option skeleton). → **RED_EVIDENCE_OK**. GREEN passes all three legs deterministically (5× repeated runs). RED `fb82883` precedes GREEN `ba9d362`.
- Gate validation: all three `test(05-04)` commits precede their `feat(05-04)` descendants (sequential history verified via git log).

## Files Created/Modified

- `internal/acpport/types.go` — NoteTool + ToolNote/ToolDiff/ToolLocation vocabulary, ToolKind*/ToolStatus* constants
- `internal/acpport/sink.go` — the tool arms (startOpts/update translation, verbatim rawInput, patch semantics, oldText omission)
- `internal/acpport/agent.go` — WR-06 structural drop-site warn in joinContentBlocks (kinds and counts only)
- `internal/acpfront/stream.go` — the tool.updated classifier, IN-05 turn-id guard site, camelCase dig-outs, kind table, diff recipes, absolute-path resolution
- `internal/acpfront/turnctx.go` — the WR-03 per-turn bounded pump (enqueue/drain/coalesce/breach/wait) + slot setup/reset
- `internal/acpfront/front.go` — turn slot fields (turnID/turnCwd/turnDiffs/turnPump), session-cwd registry, Prompt's breach gate, WithStreamCeiling
- `internal/acpfront/tool_stream_test.go` (NEW) — TestToolLifecycle, TestStreamBackpressure, composeBackLeg, gatedPromptResult/driveGatedPrompt, assert helpers
- `internal/acpport/agent_test.go` — TestToolNote (byte-exact wire table), TestUnsupportedPromptBlocksWarnStructural, mutex-guarded log capture
- `internal/zcode/fake/golden_test.go` — driveToolLifecycle driver
- corpus: `tool-lifecycle.json` + `tool-lifecycle.golden` (the TOOL-01 test surface)

## Decisions Made

- **WR-06 at the physical drop site.** The must_haves artifact line put the drop-site warn in stream.go; the physical drop of unsupported prompt blocks happens in acpport's joinContentBlocks — the warn landed there (structural: kinds and counts only, no content). Deviation documented below.
- **Coalescing-aware composed-tier lock.** The plan's two truths ("exactly the ordered sequence" for the corpus replay vs "same-toolCallId updates coalesce keep-latest") meet at the composed tier: the pump coalesces at enqueue whenever a same-id patch is still queued, so the intermediate in_progress count is pressure-dependent. Leg 1 therefore locks the deterministic invariants (order, exactly one start card, terminal completed patch with rawOutput + re-emitted diff, lossless text, foreign absence) and the byte-level backend emit order stays golden-locked at the fake tier.
- **Breach overrides the machine outcome.** A ceiling breach fails Prompt with the coded error even when the machine resolved completed — an incomplete stream must never ride a success stopReason.
- **Verbatim rawInput.** The sink composes StartToolCall with the locations option before the raw input so the SDK helper's rawInput.path backfill is wholly overwritten — the sink transports, never augments (the TOOL-01 honesty carry-over).
- **SDK union discovery.** ToolCallContent marshals the diff variant inline (no "diff" wrapper key) — the RED-stage expected frames were corrected to the recorded-tee truth on the first GREEN run.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] WR-06's drop site physically lives in acpport/agent.go, not stream.go**
- **Found during:** Task 2 GREEN
- **Issue:** The must_haves artifact list assigns the WR-06 drop-site warn to internal/acpfront/stream.go, but prompt content blocks are dropped in acpport's joinContentBlocks (the front only sees the joined text) — implementing the warn in stream.go was structurally impossible
- **Fix:** The structural slog.Warn (kinds and counts only) landed at the real drop site (joinContentBlocks), test-locked by TestUnsupportedPromptBlocksWarnStructural in acpport's agent_test.go (that file was already in Task 1's set)
- **Files modified:** internal/acpport/agent.go, internal/acpport/agent_test.go
- **Verification:** the warn's kinds+counts are asserted and the no-leak assert proves no block content is logged
- **Committed in:** e3f748b

**2. [Rule 1 - Bug] TestToolLifecycle leg 1 pinned an exact 5-frame count that the pump's designed coalescing makes nondeterministic**
- **Found during:** Task 3 GREEN (leg 1 collected 4 updates — the completed patch had coalesced a queued in_progress under drain pressure)
- **Issue:** Task 2's staged assert locked the intermediate-frame count; Task 3's unconditional keep-latest coalescing legitimately collapses same-id patches whenever the drain lags — the two staged contracts could not both hold deterministically
- **Fix:** reworked leg 1 to the coalescing-aware invariants (order: start → patches → text; exactly one start card; terminal completed patch with verbatim rawOutput and re-emitted diff; foreign absence; lossless text) — the backend emit ORDER remains byte-locked by the fake golden; 5× repeated -race runs stable
- **Files modified:** internal/acpfront/tool_stream_test.go
- **Verification:** TestToolLifecycle + TestStreamBackpressure green across repeated -shuffle -race runs; mise run ci green
- **Committed in:** ba9d362

**3. [Rule 1 - Bug] The WR-06 log-capture buffer raced the SDK's background connection logs**
- **Found during:** Task 3 GREEN (shuffle-enabled CI run caught a DATA RACE: the swapped-in default slog handler receives other tests' background goroutine logs concurrently with the test's buffer read)
- **Issue:** plain bytes.Buffer under slog.SetDefault is written by foreign goroutines while the test reads it
- **Fix:** mutex-guarded syncBuffer for the capture (write side = the default handler, read side = the assertion snapshot); 10 shuffled -race runs clean
- **Files modified:** internal/acpport/agent_test.go
- **Verification:** shuffled -race runs green; mise run ci green
- **Committed in:** ba9d362

---

**Total deviations:** 3 auto-fixed (2 × Rule 1, 1 × Rule 3)
**Impact on plan:** All deviations are correctness-critical within the plan's own contracts (the drop site's physical location, the coalescing-vs-order reconciliation, the race-free log capture). The 3-task scope held — no task split.

## Issues Encountered

- golangci-lint (strict profile) required the usual staged-test decomposition (funlen/gocognit/cyclop splits, goconst constants, lll reflows, rangeValCopy indexing); final gate 0 issues.
- The pre-commit lint loop surfaced the log-capture race only under `-shuffle` — the fix (syncBuffer) is itself the kind of lock this plan exists to teach.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TOOL-01's contract is test-locked end-to-end; 05-05 (session lifecycle) joins its teardown to the turn slot's pump release ordering (release is the single close-out point the close path should call).
- 05-07 (cmd responder wiring) inherits WithStreamCeiling as the production knob if a non-default ceiling is ever wanted (the 64 MiB default needs no wiring).
- The tool-lifecycle row joins the corpus the Phase-7 QA-01 full-scenario acceptance builds on; no blockers or concerns.

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-20*

## Self-Check: PASSED

All 15 created/modified files verified on disk; commits 302c0df, e71b2ae, b8eaf9e, e3f748b, fb82883, ba9d362, 0435734 verified as ancestors of HEAD (git rev-list over the plan ledger gsd-plan-head-before-05-04 = f74a92d → 6 measured plan commits + the metadata commit). No unexpected file deletions in any task commit.
