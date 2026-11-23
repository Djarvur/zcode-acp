---
phase: 09-user-input-bridge
verified: 2026-11-23T18:55:13Z
status: passed
score: 5/5 must-haves verified
covered_files: [".planning/phases/09-user-input-bridge/09-01-PLAN.md", ".planning/phases/09-user-input-bridge/09-01-SUMMARY.md", ".planning/phases/09-user-input-bridge/09-UAT.md", "README.md", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "docs/protocol/app-server-dialect.md", "experiments/captures/v022-ask-answered.png", "experiments/captures/v022-ask-dialog.png", "internal/acpport/userinput.go", "internal/acpport/userinput_internal_test.go", "internal/interact/responder.go", "internal/interact/userinputbridge_internal_test.go"]
covered_digest: "v4:sha256:43b0d343e520f6acb7443300486a12d289743c40d13b4763de128e77670d6c11"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 5/5
  gaps_closed:
    - "UIB-05 second pass — the user's real-machine /gsd-manager acceptance (09-UAT.md Test 2 PASSED, 2026-11-23 evening)"
  gaps_remaining: []
  regressions: []
---

# Phase 9: Ask-the-User Bridge — Verification Report

**Phase Goal:** `interaction/requestUserInput` (the AskUserQuestion shape) bridges to ONE client-rendered ACP form elicitation, and the answer maps back into the dialect's `{action, content: {answers}}` merge contract — keyed by question text with option labels as values (multi-select joined ", "); decline/cancel verbatim; an emission failure declines with the reason and never fails the turn.
**Verified:** 2026-11-23T18:55:13Z
**Status:** passed
**Re-verification:** Yes — stale fingerprint regenerated; previous human item now satisfied

Re-verification trigger: the previous report's fingerprint predated `182570a` (the goconst const-hoist in `internal/acpport/userinput_internal_test.go`). The diff was inspected line-by-line: only the shared string literals moved into a `const` block, hoisted into the existing call sites — no assertion, no case arm, and no coverage was changed. HEAD = `2592cf5b8def8645c7e55e5ebef55d8ab3364eba`; the only dirty path is an uncommitted `.planning/ROADMAP.md` phase-details block (planning bookkeeping, no source involvement). Since that prior pass, `f3d595e` also settled both deviations the stale report recorded: the README ask-the-user note landed and `TestDecodeUserInputAsk` was added.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | UIB-01: `interaction/requestUserInput` (AskUserQuestion shape) bridges to ONE ACP form elicitation rendered by the client | ✓ VERIFIED | `internal/acpport/userinput.go:59-68` — `NewUserInputAsker` performs exactly one `send(ctx, BuildElicitationRequest(ask))` per ask; `BuildElicitationRequest` returns one `CreateElicitationRequest{Form:{Message, Mode:"form", RequestedSchema, Session}}` (lines 130-189). Policy row wired: `internal/interact/responder.go:313-320` (`MethodRequestUserInput` → `r.handleUserInput`, `Deadline: 0` with the human-dialog rationale). Production wiring: `cmd/zcode-acp/main.go:701` `acpport.NewUserInputAsker(agentConn.CreateElicitation)` → `interact.WithUserInputAsker` (`main.go:635`). Behavioral: E2E subtest `TestAgentBinaryEndToEnd/wired_responder_answers_a_scripted_reverse` PASSED this run (0.21 s, built binary, real client conn) — the scripted reverse crossed to the collector's `CreateElicitation` callback and a decline rode back through production stdio with no unanswered-reverse note. Field: `v022-ask-dialog.png` shows ONE form dialog in Orca for the scripted ask |
| 2 | UIB-02: the form carries one index-keyed property per question — single-select → string with titled oneOf (const=title=label, description when present); multiSelect → array of label enums; the prompt rides the form message | ✓ VERIFIED | `TestBuildElicitationRequest` (green under `-race` this run): 2 properties for 2 questions; `q1` string variant, title = question text, oneOf[0] const=title="Verify 1" with description "verify phase one"; `q2` titled array variant with 2 anyOf options; message = prompt "Pick one", mode "form", session scope "sess-1". `TestBuildElicitationRequestFreeText` locks the no-questions arm (one prompt property). Index-keying is `elicitKey` (`q` + 1-based index), so duplicate question texts cannot collide. Field confirmation: the rendered dialog shows the message, question line ("Phase: Which phase?"), both option labels with the description, and the Skip button |
| 3 | UIB-03: accept maps to `{action: "accept", content: {answers: {<question text>: <option label>}}}` (multi joined ", "); decline/cancel verbatim | ✓ VERIFIED | `TestMapElicitationOutcome` (green): `answers["Which phase?"]="Exit manager"` (label by question text), `answers["Which extras?"]="lint, docs"` (joined), decline/cancel pass verbatim. Handler-level behavioral: `TestHandleUserInputBridge/the_bridged_ask_renders_and_the_outcome_maps` (green) asserts the dialect result `content.answers["Which phase?"]=="Verify 1"` and `action=="accept"`. Field: on the VM the fake backend's scripted expect matched the bridged answer EXACTLY — `{action: accept, content: {answers: {"Which phase?": "Verify 1"}}}` (a mismatch fails the run, per `09-UAT.md` Test 1) |
| 4 | UIB-04: without a bound elicitation sender the honest decline stays; an emission failure never fails the turn — decline with the reason, structural logs | ✓ VERIFIED | `TestHandleUserInputBridge` subtests `no_asker_—_the_legacy_decline` and `the_asker_error_declines_with_the_reason` (both green under `-race`): nil asker → `{action: decline}`; asker error → decline Answer, handler returns nil error. Pre-existing lock stays green: `TestResponderContract` run this pass, including `responder_test.go:246-270` `assertDeclineStages`. Code: `responder.go:557-602` — nil-asker arm, undecodable-params arm, error arm (`declineAnswer("the ask surface failed: "+err)`), all logged with method/requestId/id keys only (never params) |
| 5 | UIB-05: field-verified — the choice dialog RENDERS in Orca and answering it feeds the workflow (VM GUI pass + the user's /gsd-manager) | ✓ VERIFIED | Both field passes are now recorded PASSED in `09-UAT.md`: **Test 1 PASSED** (2026-11-23, VM qa02, GUI, agent-executed) — real Orca + omp card → wrapper → fake backend with scripted reverse; `experiments/captures/v022-ask-dialog.png` shows the form dialog ("Pick one", "Which phase?", "1 Verify 1 / verify phase one", "2 Exit manager", Skip, "Awaiting user input"), and `v022-ask-answered.png` shows the resolved state ("Asked … / Verify 1 / Resolved — Answered on this computer") with the turn complete and the fake's scripted expect matching the bridged answer exactly. **Test 2 PASSED** (2026-11-23 evening) — the user's real-machine live Orca acceptance: "вроде бы все работает", with the choice dialogs rendering and feeding the workflow (the "окно в третий раз отклонено" symptom closed). No outstanding human item remains |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Deferred Items

None — no failed item is scheduled to a later milestone phase.

### Advisory (New Scope, Unevidenced)

None. No new-scope blocker was raised this pass; the only re-verification-mode finding was the stale fingerprint itself, which this report regenerates.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/acpport/userinput.go` | Neutral types + `DecodeUserInputAsk` + `BuildElicitationRequest` + `MapElicitationOutcome` + `NewUserInputAsker` (the plan's Tasks 1-2, consolidated into one file — the plan's `internal/interact/userinput.go` location was superseded by this move) | ✓ VERIFIED | 259 lines, substantive, imported by `interact` and `cmd/zcode-acp` |
| `internal/interact/responder.go` (rows) | `WithUserInputAsker` seam + `handleUserInput` policy row + accept/decline answer helpers | ✓ VERIFIED | `responder.go:121-127, 185, 313-320, 545-626`; wired into `defaultPolicyTable` |
| `internal/acpport/userinput_internal_test.go` | Task 2 locks (build + map) plus the Task 1 decode arm | ✓ VERIFIED | 4 tests: `TestBuildElicitationRequest`, `TestBuildElicitationRequestFreeText`, `TestMapElicitationOutcome`, `TestDecodeUserInputAsk` — all green under `-race` this run. The previous report's warning that `TestDecodeUserInputAsk` was missing is resolved by `f3d595e`; `182570a` only const-hoisted its literals |
| `internal/interact/userinputbridge_internal_test.go` | Task 3 lock (decode → ask → map; error arm; legacy arm) | ✓ VERIFIED | 3 subtests, green under `-race -count=1` this run |
| `cmd/zcode-acp/main.go` + `main_test.go` | Task 4 production wiring + Task 5 composed binary lock | ✓ VERIFIED | `main.go:701` `NewUserInputAsker(agentConn.CreateElicitation)`; `wireResponder` injects it at `main.go:635`; E2E records the bridge's elicitation and locks the decline-through-the-bridge (transcript assert + elicitation-count assert), green this run |
| `docs/protocol/app-server-dialect.md` | Task 5: requestUserInput row gains the shipped note | ✓ VERIFIED | Line 83: **SHIPPED (v0.22.0)** note documenting the bridge, the q1..qn mapping, the answers merge contract, and the legacy-decline fallback |
| `README.md` | Task 5: README gains the shipped note | ✓ VERIFIED | Section 6 "Ask-the-user dialogs" (lines 210-216) documents the rendered dialog, the `content.answers` question-keyed merge, the decline/skip resolution, and the no-configuration posture. The stale report's PARTIAL finding is closed by `f3d595e` |
| `.planning/phases/09-user-input-bridge/09-UAT.md` | UIB-05 field evidence | ✓ VERIFIED | Test 1 PASSED (VM GUI) + Test 2 PASSED (user's real-machine acceptance); cited for Truth 5 |
| `experiments/captures/v022-ask-dialog.png`, `v022-ask-answered.png` | UIB-05 field evidence | ✓ VERIFIED | Screenshots re-examined this pass: the dialog renders with message/question/labels/descriptions/Skip, and the answered state records the choice and the resolution |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `internal/interact/responder.go` `handleUserInput` | `acpport.DecodeUserInputAsk` / asker / `acceptAnswer` | decode → `r.userInputAsk(ctx, ask)` → outcome map | WIRED | `responder.go:568, 580, 601`; the asker closure (acpport) composes build/send/map |
| policy table row | `r.handleUserInput` | `MethodRequestUserInput` entry | WIRED | `responder.go:313-320`, deadline 0 with the rationale comment |
| `cmd/zcode-acp/main.go` `run` | `acpport.NewUserInputAsker(agentConn.CreateElicitation)` | SDK method value as `ElicitationSender` | WIRED | `main.go:701`; SDK types stay inside acpport (D-01 seam) |
| `wireResponder` | `interact.WithUserInputAsker` | option injection | WIRED | `main.go:631-635` |
| `acpport.NewUserInputAsker` | `acp.CreateElicitationRequest` | `BuildElicitationRequest` → one `send` | WIRED | `userinput.go:59-68, 130-189`; one elicitation per ask |
| dialect doc row / README | shipped bridge behavior | SHIPPED notes | WIRED | `docs/protocol/app-server-dialect.md:83`; `README.md:210-216` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Orca dialog fields (message, question, option labels/descriptions) | `ask` | backend `requestUserInput` params → `DecodeUserInputAsk` → `BuildElicitationRequest` | Yes — field capture shows the scripted ask's actual values rendered | ✓ FLOWING |
| Dialect reply `content.answers` | `outcome.Answers` | client accept content `q1..qn` → `MapElicitationOutcome` → `acceptAnswer` | Yes — the fake's scripted expect matched the bridged answer exactly on the VM; the handler test asserts the same nested shape | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Decode + form build + outcome mapping locks | `go test -race -run 'TestBuildElicitationRequest\|TestMapElicitationOutcome\|TestDecodeUserInputAsk' ./internal/acpport/ -v` | 4/4 PASS (1.01 s) | ✓ PASS |
| Bridge handler: accept/error/legacy arms | `go test -race -count=1 -run 'TestHandleUserInputBridge' ./internal/interact/ -v` | 3/3 subtests PASS (1.01 s) | ✓ PASS |
| Responder contract incl. the nil-asker decline stage | `go test -race -count=1 -run 'TestResponderContract' ./internal/interact/ -v` | PASS (0.71 s) | ✓ PASS |
| Composed binary bridge (built binary, scripted fake) | `go test -race -count=1 -run 'TestAgentBinaryEndToEnd/wired_responder_answers_a_scripted_reverse' ./cmd/zcode-acp/ -v` | PASS (0.21 s) | ✓ PASS |
| Vet the phase packages | `go vet ./internal/acpport/ ./internal/interact/ ./cmd/zcode-acp/` | exit 0 | ✓ PASS |
| Lint the phase packages (strict config) | `golangci-lint run ./internal/acpport/... ./internal/interact/... ./cmd/zcode-acp/...` | `0 issues.`, exit 0 | ✓ PASS |
| Field: dialog renders and answer resolves | `09-UAT.md` Test 1 + `experiments/captures/v022-ask-dialog.png`, `v022-ask-answered.png` | Dialog renders; answer resolves the ask and completes the turn | ✓ PASS |
| Field: real-machine acceptance | `09-UAT.md` Test 2 | User-confirmed live Orca acceptance, 2026-11-23 evening | ✓ PASS |

### Probe Execution

No probes declared in the plan; conventional `scripts/*/tests/probe-*.sh` discovery returned none. The phase's runnable check surface is the Go test suite + the E2E binary lock, exercised above.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| UIB-01 | 09-01 | requestUserInput (AskUserQuestion shape) bridges to ONE client-rendered form elicitation | ✓ SATISFIED | Truth 1: code + E2E + field capture |
| UIB-02 | 09-01 | One index-keyed property per question; single-select string oneOf / multiSelect array; prompt as message | ✓ SATISFIED | Truth 2: `TestBuildElicitationRequest` + free-text lock |
| UIB-03 | 09-01 | Accept → `{action, content.answers}` merge contract; multi joined ", "; decline/cancel verbatim | ✓ SATISFIED | Truth 3: unit + handler + VM exact-match |
| UIB-04 | 09-01 | Nil-asker honest decline stays; emission failure declines with reason, never fails the turn | ✓ SATISFIED | Truth 4: bridge subtests + `assertDeclineStages` lock |
| UIB-05 | 09-01 | Field-verified: dialog renders in Orca, answering feeds the workflow | ✓ SATISFIED | Truth 5: `09-UAT.md` Test 1 (VM GUI) + Test 2 (user's real-machine acceptance) both PASSED |

No orphaned requirements: REQUIREMENTS.md maps exactly UIB-01..05 to Phase 9, all five are claimed by plan 09-01's verification table, and each has implementation + test + (for UIB-05) field evidence. The plan has no `requirements:` frontmatter key — the IDs are carried in its Verification table instead; requirements coverage was cross-checked against REQUIREMENTS.md directly. Note: REQUIREMENTS.md's traceability rows still read `Pending`; that is release bookkeeping left to the orchestrator, not an implementation gap.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/acpport/userinput_internal_test.go` | UIB-01, UIB-02, UIB-03 | 4 | 0 | no | Value (exact field/shape compares; decode arms incl. broken JSON) | PROVES |
| `internal/interact/userinputbridge_internal_test.go` | UIB-01, UIB-03, UIB-04 | 3 subtests | 0 | no | Behavioral + value (decoded ask received; dialect result shape) | PROVES |
| `internal/interact/responder_test.go` `assertDeclineStages` | UIB-04 | 1 | 0 | no | Behavioral (schema-valid decline result) | PROVES |
| `cmd/zcode-acp/main_test.go` responder-leg E2E | UIB-01 | 1 subtest | 0 | no | Behavioral (elicitation recorded on client; decline-after-ask in transcript; no unanswered note) | PROVES |

**Disabled tests on requirements:** 0 (the earlier grep hits were `e2eWaitExit(` function names, not `xit(` skip markers)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

The stale report's warning about the missing `TestDecodeUserInputAsk` is closed: the test now covers the full shape, the no-questions free-text arm, and the broken-JSON loud error, and it passed under `-race` this run.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) in phase-modified files | none | Scan clean |
| — | — | Stub/empty-implementation patterns in `userinput.go` | none | The one `return nil` is `strPtr`'s legitimate empty-string → nil-pointer helper |
| `cmd/zcode-acp/testdata/responder-e2e.json` | 16-20 | The synthetic E2E ask uses a legacy `choices` key (not the wire's `options`), so it decodes as a question with no options | ℹ️ Info | The E2E remains a valid wiring proof (one elicitation crosses, the decline returns); option mapping is proven by the unit locks and the VM field pass with the real `options` shape. No UIB requirement depends on this fixture |

**Re-verification evidence gate:** the only Step 7 candidate is the Info item above — new-scope, non-blocking, and not a 🛑 Blocker, so the gate does not apply.

### Human Verification Required

None. UIB-05's second pass — the user's real-machine `/gsd-manager` acceptance — is recorded PASSED in `09-UAT.md` Test 2 (2026-11-23 evening), alongside the VM GUI pass in Test 1. No human item remains outstanding.

### Decision Coverage

Skipped — no `*-CONTEXT.md` (no `<decisions>` block) exists in the phase directory; nothing to check.

### Gaps Summary

No gaps. Every UIB requirement verifies against real evidence: the bridge code is substantive and wired end-to-end (decode → one form elicitation → answers merge), the locks are green at HEAD under `-race` (including the newly landed decode-arm test), vet and strict-package lint are clean, and the field evidence now covers both passes — the VM GUI dialog/answer run and the user's own live Orca acceptance. Both deviations recorded by the previous report (the README shipped note, the missing `TestDecodeUserInputAsk`) were resolved in `f3d595e`; the post-verification change `182570a` was a pure goconst const-hoist with no assertion or behavior change. Phase goal achieved.

---

_Verified: 2026-11-23T18:55:13Z_
_Verifier: Claude (gsd-verifier)_
