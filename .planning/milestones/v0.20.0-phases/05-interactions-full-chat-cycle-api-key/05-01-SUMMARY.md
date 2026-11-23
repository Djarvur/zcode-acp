---
phase: 05-interactions-full-chat-cycle-api-key
plan: 01
subsystem: interaction-responder
tags: [core-06, interaction, responder, dedupe, reannounce, deadlines, fake, golden-corpus]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: zcode.Conn OnRequest intercept seam + bypass-lane Reply/ReplyError, RouterEvent record-honesty mechanics, pendingTable delete-under-lock discipline, scripted fake with reverse/expect_answer, golden corpus machinery
  - phase: 04-acp-front-minimal-chat-loop
    provides: corpus preamble discipline (front-tier framing rows), TDD RED-evidence machine
provides:
  - internal/interact Responder policy object (per-method handler table, per-method deadlines, unknown-method -32601 floor) wired through conn.OnRequest in tests
  - interact dedupe table — bounded (method, params.requestId) claim/join with resolution tombstone + TTL eviction, exactly-once
  - zcode envelope client-request-error constants CodeHostNoClientAttached/CodeHostRequestCancelled/CodeHostRequestTimedOut (-32020/-32021/-32022)
  - fake Reverse reannounce scripting (count/interval_ms, fresh wire id per re-announce, expect_answer stays on the first)
  - reannounce-dedupe corpus row + golden (first-reply-wins locked on the wire) + driveReannounceDedupe driver
  - seam placeholders for later plans: PermissionForwarder (05-02), RuntimeHeadersKeySource (05-06), credential-cipher fall-through replaced by the D-06 decrypt seam (05-07)
affects: [05-02 permission bridge, 05-06 runtime-headers key source, 05-07 responder wiring + cipher seam, 05-09 real tier]

# Actuals (#2632)
actuals:
  tokens: 19000    # chars/4 over the realized diff (75998 chars); plan estimated 56000
  tasks: 2
  commits: 3       # MEASURED: git rev-list --count gsd-plan-head-before-05-01..HEAD
commits: 3
plan_head_before: 6f7796d09fd7fa48fab83ca529b143a777e3c517
plan_head_after: 0f3f12de7154b3f27f40c91d02cc9edd35669a52

# Tech tracking
tech-stack:
  added: []        # zero new dependencies (locked discipline; go list -m all unchanged)
  patterns:
    - policy object over the OnRequest seam (Serve always handled=true except the D-05 cipher fall-through)
    - bounded correlation table with delete-under-lock claim + resolution tombstone + TTL sweep
    - park-without-framing join posture (silence is the only safe re-announce join)
    - deadline gate mirroring conn.go:365-450 (WithTimeout + DeadlineExceeded discrimination + RouterEvent honesty)

key-files:
  created:
    - internal/interact/responder.go
    - internal/interact/dedupe.go
    - internal/interact/responder_test.go
    - internal/zcode/fake/testdata/scenarios/reannounce-dedupe.json
    - internal/zcode/fake/testdata/golden/reannounce-dedupe.golden
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-01-red-responder-contract.json
  modified:
    - internal/zcode/fake/scenario.go
    - internal/zcode/fake/fake.go
    - internal/zcode/fake/scenario_test.go
    - internal/zcode/fake/golden_test.go
    - internal/zcode/envelope.go

key-decisions:
  - "Re-announce joins park without framing ANY reply until the conn closes — a frame of any shape to a still-registered wire id would reject the pending ask server-side (resultSchema.parse), and no post-release frame can be ordered after the primary's reply through the OnRequest seam"
  - "Dedupe release tombstones the entry for a 2 s resolution grace instead of deleting, so an in-flight re-announce joins (stays silent) instead of re-running the handler — first-reply-wins holds under immediate answering"
  - "interaction/requestCredentialCipher falls through (handled=false) to the conn's built-in tracked-no-answer tracker — the Phase-3 locked D-05 posture; the D-06 decrypt seam (plan 05-07) replaces the delegation"
  - "Unknown methods answer the verbatim -32601 unsupported-host-request floor with a structural log; officialMcpAuthHeaders and browser* are explicit table entries with the same floor (field-verified harmless, EXP-SUBSCRIBE-01)"
  - "The runtime-preferences golden row is re-pointed at the responder — the golden staying byte-green IS the migration's byte-identity proof at the corpus tier"

patterns-established:
  - "Responder.Serve as the conn.OnRequest assignment (assign-before-traffic hook contract), always answering with (result, werr, handled=true)"
  - "Per-method policy: {Deadline, Handler} map with Option-based deadline overrides and injected seams (nil = coded not-wired/not-configured error, never fabricated success)"
  - "Fake re-announce scripting: same params under NEW fake-N wire ids, expect_answer stays on the first envelope, honest timeout notes for unanswered joins"

requirements-completed: [CORE-06]

coverage:
  - id: D1
    description: "Unknown interaction/* method answers -32601 naming the method, process stays alive and answers a follow-up (never crash, never fabricated success)"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/interact/responder_test.go#TestResponderContract stages 1+8"
        status: pass
    human_judgment: false
  - id: D2
    description: "session/requestRuntimePreferences migrated static answer byte-identical to conn.go:384-387, served from the policy table; the existing runtime-preferences golden re-pointed at the responder stays byte-green"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/interact/responder_test.go#TestResponderContract stage 2"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/runtime-preferences"
        status: pass
    human_judgment: false
  - id: D3
    description: "Re-announced interaction requests dedupe on (method, params.requestId): the reannounce-dedupe corpus row replays through a real Conn wired with the Responder — expect_answer satisfied once, exactly one reply to the registering wire id, zero replies to join ids, forwarder runs exactly once; equal requestIds merge, distinct requestIds answer independently; concurrent same-key claims run the handler once (delete-under-lock, -race clean)"
    requirement: CORE-06
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/reannounce-dedupe + assertReannounceDedupeTranscript"
        status: pass
      - kind: unit
        ref: "internal/interact/responder_test.go#TestResponderContract stage 3 (corpus replay + adjacency probes)"
        status: pass
      - kind: unit
        ref: "internal/interact/responder_test.go#TestResponderDedupeRace"
        status: pass
    human_judgment: false
  - id: D4
    description: "Per-method deadlines fire: requestPermission 300 s default settable via Option with expiry answering deny; requestProviderRuntimeHeaders expiry answering the coded -32022 timeout error (never the failure-shaped success arm); every expiry recorded via RouterEvent (method/id key/duration only) + structural log"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/interact/responder_test.go#TestResponderDeadlineExpiry"
        status: pass
    human_judgment: false
  - id: D5
    description: "The scripted fake re-announces a pending reverse under NEW wire ids at a scripted interval (strict-schema validated fail-loud); the reannounce-dedupe corpus row + golden lock the wire truth (create framing, reverse line, two re-announce envelopes, exactly one reply, honest unanswered notes)"
    requirement: CORE-06
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/reannounce-dedupe"
        status: pass
      - kind: unit
        ref: "internal/zcode/fake/scenario_test.go#TestLoadScenarioRejectsInvalidReannounce*/AcceptsReannounce"
        status: pass
    human_judgment: false
  - id: D6
    description: "Client-request-error code constants -32020/-32021/-32022 join the envelope code block (numeric collision with the header space documented; consumers match the code, never the message)"
    requirement: CORE-06
    verification:
      - kind: unit
        ref: "internal/interact/responder_test.go#TestResponderDeadlineExpiry (asserts zcode.CodeHostRequestTimedOut on the wire)"
        status: pass
    human_judgment: false

# Metrics
duration: 50min
completed: 2026-11-20
status: complete
---

# Phase 5 Plan 1: Interaction Responder Policy Object Summary

**internal/interact Responder answers every server→host request exactly once — unknown -32601 floor, byte-identical runtime-prefs migration, (method, params.requestId) re-announce dedupe with first-reply-wins, per-method deadlines with honest expiry recording — proven end-to-end over a real Conn against the fake's new reannounce-dedupe corpus row, -race clean, zero new dependencies.**

## Performance

- **Duration:** 50 min
- **Started:** 2026-11-20T05:01:58Z
- **Completed:** 2026-11-20T05:52:00Z
- **Tasks:** 2 (Task 2 TDD: RED → GREEN)
- **Files modified:** 11

## Accomplishments

- `internal/interact` Responder policy object (D-01/CORE-06): per-method handler table {deadline, handler}, deadline gate mirroring conn.go:365-450 mechanics, unknown-method -32601 floor verbatim, replies returned to the conn for the bypass lane (the responder never writes envelopes itself)
- Re-announce dedupe keyed (method, params.requestId) modeled on call.go's pendingTable: claim/join under one mutex, resolution tombstone catching in-flight re-announces, per-entry TTL (deadline + grace) sweep — one entry per distinct requestId is the growth ceiling (T-5-01)
- The fake scripts re-announced reverses (new wire id per re-announce, count/interval scripted, expect_answer stays on the first); the reannounce-dedupe corpus row + golden lock first-reply-wins on the wire with honest unanswered notes
- TDD machine evidence: RED failed genuinely on the planned dedupe assertion (RED_EVIDENCE_OK, target_test_failed), GREEN makes every staged contract assertion pass under -race
- `mise run ci` green (build → vet → lint 0 issues → test -race → test-experiments); full `go test -race ./internal/...` green; conn.go untouched; Phase-3 router locks intact

## Task Commits

Each task was committed atomically:

1. **Task 1: Fake re-announce extension + reannounce-dedupe corpus row** - `3792aa6` (feat)
2. **Task 2: TRACER — internal/interact Responder (RED: contract locks + skeleton)** - `78cbd8e` (test)
3. **Task 2: TRACER — internal/interact Responder (GREEN: policy object end-to-end)** - `0f3f12d` (feat)

**Plan metadata:** (this SUMMARY + STATE/ROADMAP/REQUIREMENTS via the gsd-tools commit verb)

_Note: Task 2 is a TDD tracer — test(05-01) commit precedes the feat(05-01) GREEN commit; the RED evidence record lives at `.planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-01-red-responder-contract.json` (verdict RED_EVIDENCE_OK; semanticAssessment recorded inline)._

## TDD Gate Compliance

- **RED:** `go test -race -count=1 -run 'TestResponder' ./internal/interact/` failed on the planned assertion — TestResponderContract failed at stage 3 (the reannounce-dedupe corpus replay: the zero-value skeleton's handled=false routed the permission ask to the built-in router's -32601, so the scripted expect_answer was never satisfied), TestResponderDeadlineExpiry and TestResponderDedupeRace failed on their planned clauses (no deadline gate, no claim). Machine check: `gsd_run check tdd-red-evidence` → **RED_EVIDENCE_OK** (go-test adapter, 3 tests / 3 failures, matched target TestResponderContract). Semantic assessment recorded in the evidence: stages 1-2 and the -32601 default pass in RED because the skeleton's fall-through delegates to the conn's built-in router — the migration must reproduce those bytes, so the fall-through equivalence is contract-relevant, and those stages become load-bearing only once Serve returns handled=true.
- **GREEN:** all contract locks pass under `-race -count=3`; `mise run ci` green.

## Files Created/Modified

- `internal/interact/responder.go` (NEW) — the D-01 policy object: Request normalization (camelCase dig-out), policy table, deadline gate with expiry answers + RouterEvent honesty, seam-backed permission/runtime-headers handlers, decline posture, -32601 floor, credential-cipher fall-through
- `internal/interact/dedupe.go` (NEW) — the bounded (method, params.requestId) table: claim/join, resolution tombstone, TTL sweep
- `internal/interact/responder_test.go` (NEW) — TestResponderContract (staged), TestResponderDeadlineExpiry, TestResponderDedupeRace
- `internal/zcode/envelope.go` — client-request-error constants -32020/-32021/-32022 (numeric collision with the header space documented)
- `internal/zcode/fake/scenario.go` — Reverse reannounce field + fail-loud validation
- `internal/zcode/fake/fake.go` — runReverse re-announce schedule (fresh wire ids, expect_answer on the first, honest timeout notes)
- `internal/zcode/fake/scenario_test.go` — strict-schema reannounce rows (zero count, negative interval, unknown key rejected; well-formed parses)
- `internal/zcode/fake/golden_test.go` — driveReannounceDedupe driver (answers exactly once, holds join envelopes), reply-addressing post-assert, runtime-preferences driver re-pointed at the responder
- `internal/zcode/fake/testdata/scenarios/reannounce-dedupe.json` + `testdata/golden/reannounce-dedupe.golden` — the dedupe corpus row

## Decisions Made

- **Join posture = park without framing.** The plan's shorthand "a join ... returns handled with NO reply" is implemented as: the join logs structurally and parks on its request context until the conn closes, emitting zero frames. Rationale: the server result-schema-parses reply payloads (a frame of ANY shape to a still-registered wire id rejects the pending ask — "Invalid response"), and no post-release frame can be ordered after the primary's reply through the OnRequest seam (the conn writes the reply after Serve returns; the release-before-reply ordering race is unwinnable). The server itself silently drops replies to already-cleaned ids — we simply never send them. Parked goroutines add no per-frame state the conn has not already spent (the demux spawns one goroutine per request frame regardless); the dedupe MAP stays one entry per distinct requestId (T-5-01).
- **Resolution tombstone.** release() tombstones the entry for 2 s (resolutionGrace) instead of deleting, so an in-flight re-announce (already on the wire when the first reply resolves) joins silently instead of re-running the handler and framing a second reply. Without it, first-reply-wins breaks under immediate answering.
- **Credential-cipher fall-through.** The plan's must_haves sketched "Serve returning handled=true always", but Phase-3's locked D-05 posture requires tracked-no-answer for interaction/requestCredentialCipher (a reply of any shape would claim an answer the wrapper cannot give while the server holds the turn on it) — unrepresentable through the seam with handled=true. Serve returns handled=false for exactly that one method, delegating to the conn's base-context tracker; the D-06 decrypt seam (plan 05-07) replaces the delegation with a policy entry.
- **Envelope constants in the RED commit.** The contract test asserts against zcode.CodeHostRequestTimedOut by constant (numeric-code matching rule), so the -32020/-32021/-32022 block landed in the test(05-01) commit as the test's compilation surface.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] Join replies would race the primary's reply on the real wire**
- **Found during:** Task 2 (design of Serve's dedupe path)
- **Issue:** The plan's "returns handled with NO reply" as a literal immediate return would let the conn write a bare id-echo frame for the join; against the real server a payload-less reply to a still-registered wire id fails resultSchema.parse and REJECTS the pending ask (turn death), and the join's frame cannot be ordered after the primary's reply through the seam
- **Fix:** Joins park on their request context (zero frames ever); release tombstones for a 2 s resolution grace so in-flight re-announces join instead of re-running
- **Files modified:** internal/interact/responder.go, internal/interact/dedupe.go
- **Verification:** TestResponderContract stage 3 (exactly one reply to fake-2, zero replies to fake-3/fake-4, forwarder runs once); TestGolden/reannounce-dedupe byte-golden; TestResponderDedupeRace
- **Committed in:** 0f3f12d

**2. [Rule 2 - Missing critical functionality] Credential-cipher must never be answered by the responder**
- **Found during:** Task 2 (policy table construction)
- **Issue:** With the responder wired, a credential-cipher ask would hit the unknown-default and be answered -32601 — violating the Phase-3 locked D-05 posture (tracked-no-answer; "never a reply, never a fabricated -32601") and contradicting the conn's locked behavior the plan itself preserves
- **Fix:** Serve returns handled=false for interaction/requestCredentialCipher only, delegating to the conn's built-in base-context tracker; documented in defaultPolicyTable with the 05-07 replacement path
- **Files modified:** internal/interact/responder.go
- **Verification:** The conn's own Phase-3 suite stays green (the tracker still owns the posture); no responder test answers cipher
- **Committed in:** 0f3f12d

**3. [Rule 3 - Blocking issue] Test harness pipePair crossed the io.Pipe wiring**
- **Found during:** Task 2 RED (the first RED run blocked permanently)
- **Issue:** pipePair handed each end both halves of the SAME io.Pipe, so the conn read one pipe while the test wrote another — a permanent write block, not a test assertion
- **Fix:** Crossed the pipes like roundtrip_test.go's roundTripPipes (one pipe per direction); fixed before the RED evidence capture (recorded in the evidence's semanticAssessment)
- **Files modified:** internal/interact/responder_test.go
- **Verification:** RED run completes and fails on assertions only
- **Committed in:** 78cbd8e (as part of the RED commit)

---

**Total deviations:** 3 auto-fixed (2 × Rule 2, 1 × Rule 3)
**Impact on plan:** All deviations are correctness-critical within the plan's own contracts (first-reply-wins, D-05, answered-ONCE). No scope creep; the task/verify surface is exactly as planned.

## Issues Encountered

- The first RED run hung on the harness pipe defect above; diagnosed via SIGQUIT goroutine dump (parked in io.pipe.write), fixed, and the RED evidence was captured from the clean run.
- golangci-lint (strict profile) required iterative decomposition of the staged contract test (helpers per stage, named results, ID spelling); final gate: 0 issues.

## Known Stubs

None blocking — the two nil-seam defaults are deliberate, plan-prescribed interim postures, each with a named resolving plan:

| Seam | File | Interim posture (intentional) | Resolved by |
|------|------|-------------------------------|-------------|
| PermissionForwarder | internal/interact/responder.go (handlePermission) | nil default answers the coded not-wired error; deadline expiry still answers deny | Plan 05-02 (the real bridge) |
| RuntimeHeadersKeySource | internal/interact/responder.go (handleRuntimeHeaders) | nil default answers a coded internal error naming the unconfigured key state, never headersApplied:false | Plan 05-06 (real key source) |
| requestCredentialCipher | internal/interact/responder.go (Serve) | fall-through to the conn's tracked-no-answer tracker (D-05) | Plan 05-07 (D-06 decrypt seam) |

## Verification Run

- Task 1: `go test -race -count=1 -run 'TestScenario|TestGolden' ./internal/zcode/fake/` + `go test -race -count=1 ./internal/zcode/...` — green; reannounce-dedupe golden deterministic across `-count=3`
- Task 2: `go test -race -count=1 ./internal/interact/` + `TestGolden/reannounce-dedupe` + `go test -race -count=1 ./internal/...` — green
- Wave gate: `mise run ci` — green (build, vet, lint 0 issues, test -race, test-experiments)
- Phase-3/4 suites untouched and green (conn.go not modified; fallback posture per D-01)

## Self-Check: PASSED

All 11 created/modified files exist on disk; commits 3792aa6, 78cbd8e, 0f3f12d verified as ancestors of HEAD; commit count measured from the plan ledger (gsd-plan-head-before-05-01 = 6f7796d).
