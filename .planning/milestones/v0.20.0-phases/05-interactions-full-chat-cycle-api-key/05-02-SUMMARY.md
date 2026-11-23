---
phase: 05-interactions-full-chat-cycle-api-key
plan: 02
subsystem: permission-bridge
tags: [tool-02, core-06, permission, request-permission, bridge, cancel, expiry, deny-by-default, fake, golden-corpus]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key
    provides: interact Responder policy object with the PermissionForwarder seam + envelope client-request-error constants (-32020/-32021/-32022), 05-01
  - phase: 04-acp-front-minimal-chat-loop
    provides: acpport neutral Handlers seam + sinkBinder construction discipline, front harness composition (ComposeFront/TeeRW/RecordingClient), corpus preamble rows, 04-02..04-06
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: conn OnRequest intercept + bypass-lane replies, turn machine cancel (stop bypass), scripted fake with reverse/expect_answer/reannounce, golden corpus machinery
provides:
  - acpport neutral permission vocabulary (PermissionOptionDef/PermissionAsk/PermissionOutcome/PermissionSender) + ACP kind constants (PermKindAllowOnce/AllowAlways/RejectOnce), verbatim-passthrough discipline
  - acpport construction-bound PermissionSender over AgentSideConnection.RequestPermission (3-value NewAgentSideConnection, outcome union decoded both arms, neither-arm fails loud)
  - acpport.PermissionBinder — the exported Handlers-side opt-in hook (adapter-side unexported permissionBinder for the construction hop; Go qualifies unexported method names by declaring package)
  - WR-02: agentAdapter.sink as atomic.Pointer[UpdateSink] (bindSink stores, sinkForPrompt loads, loud nil-fallback preserved)
  - interact backend permission handler (the Responder's PermissionForwarder impl): camelCase params dig-out, locked kind mapping (allow_once→allow_once, allow_project/allow_always→allow_always, deny→reject_once; names/optionIds verbatim), selected option's embedded response forwarded byte-verbatim (RawMessage passthrough), deny-by-default on unknown/missing optionId + malformed params, cancelled outcome/classified error → -32021 with the stable message constant
  - interact deniedByUserReason + enriched permission expiry answer (deadline expiry → deny WITH the denied-by-user reason, OQ-2 disposition)
  - acpfront front bridge (permission.go): pending map keyed by backend requestId (bind under one mutex, delete-under-lock removal), Forward round-trip with local-cancel classification, CancelPendingForTurn + Front.Cancel fan-in (pending bridges never outlive the turn)
  - harness: scripted RecordingClient.RequestPermission (selected/cancelled + Delay + Gate holds, ask capture) + fake_test permissionClient (mode/gate/awaitAsk); drivePermissionAllow / drivePermissionCancel / driveUserInputDecline drivers
  - corpus rows + goldens: permission-allow (allow round-trip, embedded response bytes verbatim), permission-cancel (-32021 error reply + stop-bypass flow), user-input-decline ({action:decline})
  - tests: TestPermissionSender, TestSinkBindIsAtomic, TestPermissionBridge (the phase tracer), TestPermissionForwardArms/CancelledArms/AskNormalization, TestPermissionCancel, TestPermissionCancelRace (TOOL-02 concurrency probe), TestPermissionExpiry
affects: [05-05 tool notes + teardown session pendings, 05-07 responder wiring in cmd main, 05-08 configOptions, 05-09 real tier]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 31576    # chars/4 over the realized diff (126303 chars incl. evidence records; plan estimated 58000)
  tasks: 3
  commits: 7       # MEASURED: git rev-list --count gsd-plan-head-before-05-02..HEAD
commits: 7
plan_head_before: 47c27e39cb10d89405276f45adcea6426a812295
plan_head_after: 634097033573cb984a8871ef8c9d1805cd02f175

# Tech tracking
tech-stack:
  added: []        # zero new dependencies (locked discipline; go list -m all unchanged)
  patterns:
    - connection-bound PermissionSender bound at construction via the sinkBinder pattern (one-shot binder hooks before any frame flows)
    - pending map with bind/remove under one mutex + delete-under-lock exactly-once (call.go pendingTable discipline); bounded by the responder's per-method deadline
    - deny-by-default translation table (unknown/missing optionId, malformed params, expiry, cancelled → deny or -32021, never allow)
    - byte-verbatim json.RawMessage passthrough for the selected option's own embedded response
    - cancel fan-in: Front.Cancel additionally cancels every in-flight bridge (T-5-05)
    - deterministic golden sequencing via causally forced frames + bounded settle windows (the race between fan-in reply and stop exchange is never byte-locked)

key-files:
  created:
    - internal/interact/permission.go
    - internal/interact/permission_test.go
    - internal/acpfront/permission.go
    - internal/acpfront/permission_test.go
    - internal/zcode/fake/testdata/scenarios/permission-allow.json
    - internal/zcode/fake/testdata/scenarios/permission-cancel.json
    - internal/zcode/fake/testdata/scenarios/user-input-decline.json
    - internal/zcode/fake/testdata/golden/permission-allow.golden
    - internal/zcode/fake/testdata/golden/permission-cancel.golden
    - internal/zcode/fake/testdata/golden/user-input-decline.golden
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-02-red-permission-sender.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-02-red-permission-bridge.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-02-red-permission-expiry.json
  modified:
    - internal/acpport/types.go
    - internal/acpport/agent.go
    - internal/acpport/agent_test.go
    - internal/acpfront/front.go
    - internal/acpfront/harness_test.go
    - internal/acpfront/loop_test.go
    - internal/interact/responder.go
    - internal/interact/responder_test.go
    - internal/zcode/fake/golden_test.go
    - cmd/zcode-acp/main.go

key-decisions:
  - "The Handlers-side binder hook is EXPORTED (acpport.PermissionBinder{BindPermissionSender}): Go qualifies unexported method names by their declaring package, so the cross-package Front can never satisfy an unexported-method interface — the adapter keeps its own unexported permissionBinder for the construction hop (Rule 3, b83cbde)"
  - "The cancel golden locks the client's cancelled-outcome arm (the ACP v1 MUST on turn cancel) gated to answer BEFORE the session cancel — byte-golden determinism forbids locking the fan-in-reply-vs-stop-exchange race; the fan-in path (-32021 while the stop rides the bypass lane) is locked semantically by TestPermissionCancel (transcript presence, reply-once, drained pending table)"
  - "The race probe locks exactly-one-reply on the backend wire (never both, never neither) and deliberately does NOT couple the reply shape to the prompt's stopReason — in the narrow SDK race the client's answer can land while the sender's context is already dead, so the single backend outcome is -32021 while the machine still completes the turn through the corpus turn.completed"
  - "Malformed asks (no usable options) deny WITHOUT forwarding — the deny-by-default posture extends to params, not just outcomes (T-5-04)"
  - "Permission expiry answers now carry the denied-by-user reason (the 05-01 bare-deny answer enriched per the OQ-2 disposition and must_haves truth)"

patterns-established:
  - "Bridge translation discipline: the ask side maps (kinds per the locked table), the sender passes through verbatim, the outcome side forwards the option's own embedded response byte-verbatim — the wrapper transports, never augments"
  - "Construction-time binding hooks: NewAgentSideConnection binds sink + sender before any frame flows; callers adapt to the 3-value return"
  - "Gated harness holds: PermissionScript.Gate/Delay give tests deterministic sequencing over in-flight asks"

requirements-completed: [TOOL-02, CORE-06]

coverage:
  - id: D1
    description: "A backend interaction/requestPermission ask surfaces as ACP session/request_permission with the backend's options mapped per the locked table (allow_once→allow_once, allow_project/allow_always→allow_always, deny→reject_once; names and optionIds verbatim) and the toolCall patch carrying the backend toolCallId — captured on the ACP wire mid-turn"
    requirement: TOOL-02
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestPermissionSender (sender passthrough + outcome union both arms)"
        status: pass
      - kind: integration
        ref: "internal/acpfront/permission_test.go#TestPermissionBridge stage a/c (captured ask verbatim, mid-turn tee order)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The client's selected option is forwarded to the backend as that option's own embedded response, byte-verbatim — the fake's expect_answer satisfied with {decision:allow, reason:Approved once} through the FULL stack (client → acpport sender → front bridge → responder → backend reply)"
    requirement: TOOL-02
    verification:
      - kind: integration
        ref: "internal/acpfront/permission_test.go#TestPermissionBridge stage b + internal/zcode/fake/golden_test.go#TestGolden/permission-allow"
        status: pass
      - kind: unit
        ref: "internal/interact/permission_test.go#TestPermissionForwardArms (embedded response verbatim, allow_project mapping)"
        status: pass
    human_judgment: false
  - id: D3
    description: "An ACP-side cancel resolves the pending bridge with the -32021 error reply carrying the bridge's stable message constant; every pending bridge of the turn is cancelled by the cancel fan-in — the pending table drains (a second prompt works) and the stop-bypass session/stop rides the bypass lane"
    requirement: TOOL-02
    verification:
      - kind: integration
        ref: "internal/acpfront/permission_test.go#TestPermissionCancel"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/permission-cancel (byte-locked -32021 reply + stop flow)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Outcomes the backend options do not define — unknown optionId, missing optionId, malformed params, the ACP cancelled outcome — resolve deny or the -32021 error respectively, never allow; deadline expiry answers deny WITH the denied-by-user reason and the turn continues (prompt returns end_turn)"
    requirement: TOOL-02
    verification:
      - kind: unit
        ref: "internal/interact/permission_test.go#TestPermissionForwardArms deny arms + TestPermissionCancelledArms"
        status: pass
      - kind: integration
        ref: "internal/acpfront/permission_test.go#TestPermissionExpiry (deny reason byte-locked on the transcript)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The cancel/decision race yields exactly one outcome for the backend — the selected response or the -32021 error, never both, never neither (reply-once under -race); the tombstone join after an answer never earns a reply"
    requirement: TOOL-02
    verification:
      - kind: integration
        ref: "internal/acpfront/permission_test.go#TestPermissionCancelRace"
        status: pass
    human_judgment: false
  - id: D6
    description: "interaction/requestUserInput is answered {action:decline} on the wire via the corpus row (elicitation stays Phase 7); unknown interaction/* keeps the -32601 floor (CORE-06, 05-01 lock re-verified green)"
    requirement: CORE-06
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/user-input-decline"
        status: pass
      - kind: unit
        ref: "internal/interact/responder_test.go#TestResponderContract (05-01 suite re-run green over the enriched expiry answer)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The connection-bound PermissionSender binds at construction before any frame flows (binder discipline preserved) and the WR-02 sink publish is atomic (atomic.Pointer with the loud nil-fallback preserved) — existing acpport suite green"
    requirement: TOOL-02
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestSinkBindIsAtomic (16 concurrent prompts ride the bound sink; unbound fallback fails loud)"
        status: pass
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestSDKImportBoundary (acpport owns the SDK usage — re-run green)"
        status: pass
    human_judgment: false

# Metrics
duration: 72min
completed: 2026-11-20
status: complete
---

# Phase 5 Plan 2: Permission Bridge Both Directions Summary

**TOOL-02 end-to-end: the permission bridge, both directions — backend interaction/requestPermission normalized and mapped per the locked table, the connection-bound acpport PermissionSender performing the ACP session/request_permission round-trip verbatim, the selected option's own embedded response forwarded byte-verbatim, cancelled → -32021, expiry → deny-with-reason, deny-by-default on every undefined outcome — proven by the orchestrator-pinned tracer over the composed harness and byte-locked by three new corpus rows, -race clean, zero new dependencies.**

## Performance

- **Duration:** 72 min
- **Started:** 2026-11-20T05:57:09Z
- **Completed:** 2026-11-20T10:09:00Z
- **Tasks:** 3 (all TDD: RED evidence records captured per task, RED_EVIDENCE_OK each)
- **Files:** 13 created / 10 modified (fixture-weighted: 3 corpus rows + 3 goldens ARE the TOOL-02 test surface)

## Accomplishments

- acpport neutral permission vocabulary (PermissionOptionDef/PermissionAsk/PermissionOutcome/PermissionSender + ACP kind constants) with the connection-bound PermissionSender over AgentSideConnection.RequestPermission — options passthrough verbatim, outcome union decoded both arms, neither-arm fails loud
- WR-02 landed with the agent.go touch: the sink is atomic.Pointer[UpdateSink] (bindSink stores, sinkForPrompt loads, loud nil-fallback preserved, 16-prompt concurrent lock)
- The backend half (interact/permission.go) implements the Responder's PermissionForwarder seam: camelCase dig-out, locked kind mapping, byte-verbatim embedded-response forwarding, deny-by-default on unknown/missing optionId and malformed params, cancelled outcome/classified error → -32021 with the stable message constant
- The front half (acpfront/permission.go): pending map keyed by backend requestId under one mutex (delete-under-lock exactly-once), Forward round-trip with local-cancel classification, CancelPendingForTurn + the Front.Cancel fan-in — pending bridges never outlive their turn (T-5-05)
- The tracer (TestPermissionBridge) replays the permission-allow corpus row through every layer — client decision → acpport sender → front bridge → responder → backend reply — with the mapped options verbatim, the toolCallId carriage, the request_permission frame mid-turn on the ACP tee, and the back-leg expect_answer satisfied
- Cancel/expiry/decline locks: permission-cancel corpus row byte-locks the -32021 reply (code + message constant) plus the stop-bypass flow; user-input-decline byte-locks {action:decline}; TestPermissionExpiry locks the expiry deny WITH the denied-by-user reason (the one production change of Task 3); TestPermissionCancelRace locks the exactly-one-reply invariant
- Three TDD RED evidence records captured inside the project evidence dir (05-02-red-permission-{sender,bridge,expiry}.json), each RED_EVIDENCE_OK with semanticAssessment recorded
- Wave gate `mise run ci` green (build → vet → lint 0 issues → test -race → test-experiments); full `go test -race ./internal/...` green; TestSDKImportBoundary green (acpport owns the SDK usage)

## Task Commits

Each task committed atomically (RED test commit precedes the GREEN feat commit per task):

1. **Task 1 RED: acpport PermissionSender contract + neutral vocabulary skeleton** - `a6a9e6a` (test)
2. **Task 1 GREEN: PermissionSender round-trip + WR-02 atomic sink** - `31ea68e` (feat)
3. **Task 2 RED: permission bridge tracer RED (corpus row + harness scripting + skeletons)** - `63eb7ae` (test)
4. **Task 2 GREEN: bridge both halves end-to-end (TRACER GREEN)** - `b08fd6e` (feat)
5. **Task 2 GREEN fix: adapter-side + Handlers-side binder split (Rule 3)** - `b83cbde` (fix)
6. **Task 3 RED: cancel + expiry + decline locks (corpus rows, drivers, race probe)** - `44ca0fa` (test)
7. **Task 3 GREEN: expiry deny reason + cancel/decline goldens** - `6340970` (feat)

**Plan metadata:** (this SUMMARY + STATE/ROADMAP/REQUIREMENTS via the gsd-tools commit verb)

## TDD Gate Compliance

- **Task 1 RED:** TestPermissionSender failed on the first staged assert (no session/request_permission frame — the stub sender never writes) in both outcome arms. `gsd_run check tdd-red-evidence` → **RED_EVIDENCE_OK** (go-test adapter, target_test_failed). GREEN satisfies the frame asserts and the outcome-union decodes.
- **Task 2 RED:** TestPermissionBridge failed on the first staged assert (client captured 0 asks — the skeletons never forward); TestGolden/permission-allow failed on the missing fixture. → **RED_EVIDENCE_OK**. GREEN passes all stages (captured ask verbatim, mapping, tee order, expect_answer satisfied, reply-once).
- **Task 3 RED:** TestPermissionExpiry failed on the planned reason assertion (the 05-01 expiry answer was bare {decision:deny}); TestPermissionCancel/CancelRace passed at RED, locking Task-2-delivered behavior (documented in the evidence's semanticAssessment); the two new corpus rows failed TestGolden on missing fixtures. → **RED_EVIDENCE_OK**. GREEN adds the denied-by-user reason to the permission expiry answer and records both goldens (deterministic across -count=3).

## Files Created/Modified

- `internal/acpport/types.go` — neutral permission vocabulary + ACP kind constants
- `internal/acpport/agent.go` — 3-value NewAgentSideConnection (sender alongside the sink), newPermissionSender translation, permissionBinder/PermissionBinder hooks, WR-02 atomic sink
- `internal/interact/permission.go` (NEW) — the backend bridge: dig-out, mapping, outcome translation, deny-by-default, -32021 classification, deniedByUserReason constant
- `internal/acpfront/permission.go` (NEW) — the front bridge: pending map, Forward, CancelPendingForTurn, PermissionForward, BindPermissionSender
- `internal/acpfront/front.go` — Front gains the perm component; Cancel fans in CancelPendingForTurn
- `internal/interact/responder.go` — expiryAnswer carries the denied-by-user reason for permission
- `internal/acpfront/harness_test.go` — scripted RecordingClient.RequestPermission (selected/cancelled + Delay/Gate, ask capture)
- `internal/zcode/fake/golden_test.go` — permissionClient (mode/gate/awaitAsk) + drivePermissionAllow/Cancel, driveUserInputDecline drivers
- corpus: `permission-allow.json`, `permission-cancel.json`, `user-input-decline.json` + their goldens (the TOOL-02 test surface)

## Decisions Made

- **Exported Handlers-side binder (Rule 3).** The plan's sinkBinder-style hook is unimplementable cross-package with an unexported method — Go qualifies unexported method names by their declaring package (verified by a minimal experiment during the tracer debugging). The exported acpport.PermissionBinder carries the Front-side opt-in; the adapter keeps its own unexported permissionBinder for the construction hop. Committed b83cbde.
- **Cancel golden sequencing.** The fan-in reply and the stop exchange are concurrent by design — a byte-golden cannot lock their relative order. The golden therefore locks the client's cancelled-outcome arm (the ACP v1 MUST on turn cancel) gated to answer before the cancel, so every frame sits in a causally forced position: reverse → -32021 reply → send ack → stop → cancelled patch → stop ack. The fan-in path (-32021 while the stop rides the bypass lane, pending table drained, second prompt works) is locked semantically by TestPermissionCancel, which asserts presence/counts, not byte order.
- **Race probe invariant.** Exactly-one-reply on the backend wire (never both, never neither), deliberately NOT coupled to the prompt's stopReason: in the narrow SDK race the client's answer can land while the sender's context is already dead, so the single backend outcome is -32021 while the machine still completes the turn through the corpus turn.completed.
- **Malformed params deny without forwarding.** An options-less ask forwards nothing — no option can be declared, so nothing may be allowed (T-5-04 extended to params).
- **Expiry answer enriched.** The 05-01 bare {decision:deny} expiry answer now carries the denied-by-user reason — the must_haves truth and Task 3's expiry lock require it; the 05-01 suite (decision-key assert) stays green.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] The unexported permissionBinder interface cannot be satisfied cross-package**
- **Found during:** Task 2 GREEN (the tracer captured 0 asks — "no permission sender bound")
- **Issue:** Go qualifies unexported method names by their declaring package, so acpfront.Front's `bindPermissionSender` can never satisfy acpport's unexported `permissionBinder` interface — the sender never reached the front bridge
- **Fix:** Exported the Handlers-side hook as `acpport.PermissionBinder{BindPermissionSender}`; the adapter-side one-shot hop keeps its own unexported interface (same-package method — satisfiable). Verified with a minimal cross-package experiment before the fix
- **Files modified:** internal/acpport/agent.go, internal/acpfront/permission.go
- **Committed in:** b83cbde

**2. [Rule 1 - Bug] drivePermissionAllow initially crossed the ACP-leg pipes and omitted the send leg**
- **Found during:** Task 2 GREEN (the recorded golden showed a 10-line preamble-only transcript; then the reverse went unanswered at its 500 ms timeout)
- **Issue:** The driver handed the agent connection the client's pipe ends (swapped peerInput/peerOutput) and skipped the h.send call — the corpus row never fired its tracer exchange
- **Fix:** Corrected pipe ownership (agent writes toClientW, reads toAgentR) and added the send; re-recorded the golden
- **Files modified:** internal/zcode/fake/golden_test.go, internal/zcode/fake/testdata/golden/permission-allow.golden
- **Committed in:** b08fd6e

**3. [Rule 2 - Missing critical functionality] The race probe's outcome coupling would flake on a conforming implementation**
- **Found during:** Task 3 RED development (the probe observed end_turn with a -32021 reply)
- **Issue:** Coupling the reply shape to the prompt's stopReason asserts a false invariant — in the narrow SDK race the client's answer lands while the sender's context is already dead, so the backend's single outcome is -32021 while the machine completes the turn through the corpus turn.completed
- **Fix:** The probe locks exactly-one-reply with a shape-validity check (allow or -32021), and the prompt must resolve cleanly — never both, never neither
- **Files modified:** internal/acpfront/permission_test.go
- **Committed in:** 44ca0fa

---

**Total deviations:** 3 auto-fixed (1 × Rule 1, 1 × Rule 2, 1 × Rule 3)
**Impact on plan:** All deviations are correctness-critical within the plan's own contracts (bind-at-construction, corpus replayability, reply-once). The 12-file fixture-weighted scope held — no task split.

## Issues Encountered

- golangci-lint (strict profile) required the usual staged-test decomposition (helpers per arm, named fixtures, lll reflows); final gate 0 issues.
- The SDK's request-id type is internal — the sender test replays the decoded raw id token verbatim into the response frame rather than assuming a numeric shape.

## Known Stubs

None blocking — the two nil-seam defaults from 05-01 remain deliberate interim postures with named resolving plans, both UNCHANGED by this plan:

| Seam | File | Interim posture (intentional) | Resolved by |
|------|------|-------------------------------|-------------|
| RuntimeHeadersKeySource | internal/interact/responder.go (handleRuntimeHeaders) | nil default answers a coded internal error naming the unconfigured key state, never headersApplied:false | Plan 05-06 (real key source) |
| requestCredentialCipher | internal/interact/responder.go (Serve) | fall-through to the conn's tracked-no-answer tracker (D-05) | Plan 05-07 (D-06 decrypt seam) |

The PermissionForwarder nil-seam default from 05-01 is RESOLVED by this plan (the real bridge; production wiring lands in 05-07).

## Verification Run

- Task 1: `go test -race -count=1 -run 'TestPermissionSender|TestSink' ./internal/acpport/` + full acpport suite — green; WR-02 atomic under -race
- Task 2: `go test -race -count=1 -run 'TestPermissionBridge|TestResponder' ./internal/acpfront/ ./internal/interact/` + `go test -race -count=1 -run 'TestGolden/permission-allow' ./internal/zcode/fake/` — green; golden deterministic across -count=3
- Task 3: `go test -race -count=1 -run 'TestPermissionCancel|TestPermissionExpiry' ./internal/acpfront/` + `TestGolden/permission-cancel|user-input-decline` + full `go test -race -count=1 ./internal/...` — green (all 8 packages + cmd)
- Wave gate: `mise run ci` — green (build, vet, lint 0 issues, test -race, test-experiments)
- SDK-import boundary: TestSDKImportBoundary green (acpport owns the RequestPermission SDK usage)

## Self-Check: PASSED

All 13 created files exist on disk; commits a6a9e6a, 31ea68e, 63eb7ae, b08fd6e, b83cbde, 44ca0fa, 6340970 verified as ancestors of HEAD (git rev-list over the plan ledger gsd-plan-head-before-05-02 = 47c27e3 → 7 commits measured).
