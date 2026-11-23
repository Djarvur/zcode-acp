---
phase: 09-user-input-bridge
verified: 2026-11-23T12:35:44Z
status: human_needed
score: 5/5 must-haves verified
covered_files: [".planning/phases/09-user-input-bridge/09-01-PLAN.md", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "docs/protocol/app-server-dialect.md", "experiments/captures/v022-ask-answered.png", "experiments/captures/v022-ask-dialog.png", "internal/acpport/userinput.go", "internal/acpport/userinput_internal_test.go", "internal/interact/responder.go", "internal/interact/userinputbridge_internal_test.go"]
covered_digest: "v4:sha256:a7eeff22c8b57218c6a5021106f7a65656fe76457a302db757de74ee27515a79"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "UIB-05 second pass (real-machine acceptance): restart the Orca chat session on the installed bridge binary and re-run /gsd-manager"
    expected: "The workflow's choice dialogs RENDER in the card and answering one feeds the workflow — the field symptom 'окно в третий раз отклонено' is gone"
    why_human: "Requires the user's actual Orca install and the real gsd-manager workflow on their machine; not reproducible in CI or on the VM's fake-backend harness"
---

# Phase 9: Ask-the-User Bridge — Verification Report

**Phase Goal:** `interaction/requestUserInput` bridges to a client-rendered form elicitation (the AskUserQuestion dialogs the card never showed); the answer maps back into the dialect's `answers` merge contract
**Verified:** 2026-11-23T12:35:44Z
**Status:** human_needed (5/5 truths verified with real evidence; one outstanding human acceptance item)
**Re-verification:** No — initial verification

Evidence note: `09-01-SUMMARY.md` does not exist for this phase. Per the verification map in `09-01-PLAN.md`, the evidence base is the plan's task→requirement map, `09-UAT.md`, and the code itself — all examined directly. HEAD = `b0d03fe3a57c6e1dd5a562ddb4127fce7d407ab0` on `gsd/phase-06-oauth-coding-plan-path-model-picker`, clean tree.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | UIB-01: `interaction/requestUserInput` (AskUserQuestion shape) bridges to ONE ACP form elicitation rendered by the client | ✓ VERIFIED | `internal/acpport/userinput.go` — `NewUserInputAsker` performs exactly one `send(ctx, BuildElicitationRequest(ask))` per ask; `BuildElicitationRequest` returns one `CreateElicitationRequest{Form:{Message, Mode:"form", RequestedSchema, Session}}`. Row wired: `internal/interact/responder.go:317-323` (`MethodRequestUserInput` → `r.handleUserInput`, deadline 0 so a human answer is never declined mid-thought). Production wiring: `cmd/zcode-acp/main.go:701` `acpport.NewUserInputAsker(agentConn.CreateElicitation)` → `interact.WithUserInputAsker` (`main.go:635`). Behavioral: E2E subtest `TestAgentBinaryEndToEnd/wired_responder_answers_a_scripted_reverse` (built binary, real client conn) PASSED — the elicitation crossed to the collector client and the scripted decline rode back through the bridge with no unanswered-reverse note. Field: `v022-ask-dialog.png` shows ONE form dialog in Orca for the scripted ask |
| 2 | UIB-02: the form carries one index-keyed property per question — single-select → string with titled oneOf (const=title=label, description when present); multiSelect → array of label enums; the prompt rides the form message | ✓ VERIFIED | `TestBuildElicitationRequest` (green under -race): 2 properties for 2 questions; `q1` string variant, title = question text, oneOf[0] const=title="Verify 1" with description "verify phase one"; `q2` titled array variant with 2 anyOf options; message = prompt "Pick one", mode "form", session scope "sess-1". `TestBuildElicitationRequestFreeText` locks the no-questions arm (one prompt property). Field confirmation: the rendered dialog shows the message, question line ("Phase: Which phase?"), both option labels with the description, and the Skip button |
| 3 | UIB-03: accept maps to `{action: "accept", content: {answers: {<question text>: <option label>}}}` (multi joined ", "); decline/cancel verbatim | ✓ VERIFIED | `TestMapElicitationOutcome` (green): `answers["Which phase?"]="Exit manager"` (label by question text), `answers["Which extras?"]="lint, docs"` (joined), decline/cancel pass verbatim. Handler-level behavioral: `TestHandleUserInputBridge/the_bridged_ask_render_and_the_outcome_maps` (green) asserts the dialect result `content.answers["Which phase?"]=="Verify 1"`. Field: on the VM the fake backend's scripted expect matched the bridged answer EXACTLY — `{action: accept, content: {answers: {"Which phase?": "Verify 1"}}}` (a mismatch fails the run, per `09-UAT.md` Test 1) |
| 4 | UIB-04: without a bound elicitation sender the honest decline stays; an emission failure never fails the turn — decline with the reason, structural logs | ✓ VERIFIED | `TestHandleUserInputBridge` subtests `no_asker_—_the_legacy_decline` and `the_asker_error_declines_with_the_reason` (both green under -race): nil asker → `{action: decline}`; asker error → decline Answer, handler returns nil error. Pre-existing lock stays green: `responder_test.go:246-270` `assertDeclineStages`. Code: `responder.go:557-602` — nil-asker arm, undecodable-params arm, error arm (`declineAnswer("the ask surface failed: "+err)`), all structurally logged (id keys only, never params) |
| 5 | UIB-05 (first pass): field-verified — the choice dialog RENDERS in Orca and answering it feeds the workflow | ✓ VERIFIED | `09-UAT.md` Test 1 PASSED (2026-11-23, VM qa02, agent-executed): real Orca + omp card → wrapper → fake backend with scripted reverse. `experiments/captures/v022-ask-dialog.png` — the form dialog with message "Pick one", question "Which phase?", options "1 Verify 1 / verify phase one" and "2 Exit manager", Skip button, chat's "Awaiting user input" marker. `experiments/captures/v022-ask-answered.png` — clicking "Verify 1" resolved the dialog ("Asked … / Verify 1 / Resolved — Answered on this computer"), the turn completed, and the scripted expect matched the bridged answer exactly. The second listed pass (the user's own /gsd-manager) is the outstanding human item below |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Advisory (New Scope, Unevidenced)

None (initial verification — no prior gap-closure contract in scope).

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/acpport/userinput.go` | Neutral types + `DecodeUserInputAsk` + `BuildElicitationRequest` + `MapElicitationOutcome` + `NewUserInputAsker` (the plan's Tasks 1-2, consolidated into one file — the plan's `internal/interact/userinput.go` location was superseded by this move) | ✓ VERIFIED | 259 lines, substantive, imported by `interact` and `cmd/zcode-acp` |
| `internal/interact/responder.go` (rows) | `WithUserInputAsker` seam + `handleUserInput` policy row + accept/decline answer helpers | ✓ VERIFIED | `responder.go:121-127, 185, 317-323, 545-620`; wired into `defaultPolicyTable` |
| `internal/acpport/userinput_internal_test.go` | Task 2 locks (build + map) | ✓ VERIFIED | 3 tests, value-level assertions, green under -race |
| `internal/interact/userinputbridge_internal_test.go` | Task 3 lock (decode → ask → map; error arm; legacy arm) | ✓ VERIFIED | 3 subtests, green under -race |
| `cmd/zcode-acp/main.go` + `main_test.go` | Task 4 production wiring + Task 5 composed binary lock | ✓ VERIFIED | `main.go:701` `NewUserInputAsker(agentConn.CreateElicitation)`; E2E records the bridge's form and locks the decline-through-the-bridge (transcript assert + elicitation count), green |
| `docs/protocol/app-server-dialect.md` | Task 5: requestUserInput row gains the shipped note | ✓ VERIFIED | Line 83: **SHIPPED (v0.22.0)** note documenting the bridge, the q1..qn mapping, the answers merge contract, and the legacy-decline fallback |
| `experiments/captures/v022-ask-dialog.png`, `v022-ask-answered.png` | UIB-05 field evidence | ✓ VERIFIED | Real VM GUI screenshots examined — dialog renders with labels/descriptions/Skip; resolved state recorded; committed at `b0d03fe` |
| README shipped note | Task 5: "README + the dialect doc … gain the shipped note" | ⚠️ PARTIAL | Only the dialect doc landed (commit `f1edb83` touches it alone); README.md contains no mention of the ask-the-user bridge. No UIB requirement depends on the README — recorded as a plan-task deviation, not a requirement gap |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `internal/interact/responder.go` `handleUserInput` | `acpport.DecodeUserInputAsk` / asker / `acceptAnswer` | decode → `r.userInputAsk(ctx, ask)` → outcome map | WIRED | `responder.go:568, 580, 601`; the asker closure (acpport) composes build/send/map |
| policy table row | `r.handleUserInput` | `MethodRequestUserInput` entry | WIRED | `responder.go:317-323`, deadline 0 with the rationale comment |
| `cmd/zcode-acp/main.go` `run` | `acpport.NewUserInputAsker(agentConn.CreateElicitation)` | SDK method value as `ElicitationSender` | WIRED | `main.go:701`; SDK types stay inside acpport (D-01 seam) |
| `wireResponder` | `interact.WithUserInputAsker` | option injection | WIRED | `main.go:631-635` |
| dialect doc row | shipped bridge behavior | SHIPPED (v0.22.0) note | WIRED | `docs/protocol/app-server-dialect.md:83` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| Orca dialog fields (message, question, option labels/descriptions) | `ask` | backend `requestUserInput` params → `DecodeUserInputAsk` → `BuildElicitationRequest` | Yes — field capture shows the scripted ask's actual values rendered | ✓ FLOWING |
| Dialect reply `content.answers` | `outcome.Answers` | client accept content `q1..qn` → `MapElicitationOutcome` → `acceptAnswer` | Yes — the fake's scripted expect matched the bridged answer exactly on the VM | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Bridge handler: accept/error/legacy arms | `go test -race -run TestHandleUserInputBridge ./internal/interact/ -v` | 3/3 subtests PASS | ✓ PASS |
| Form build + outcome mapping locks | `go test -race -run 'TestBuildElicitationRequest\|TestMapElicitationOutcome' ./internal/acpport/ -v` | 3/3 PASS | ✓ PASS |
| Composed binary bridge (built binary, scripted fake) | `go test -race -run 'TestAgentBinaryEndToEnd/wired_responder_answers_a_scripted_reverse' ./cmd/zcode-acp/ -v` | PASS (0.97s) | ✓ PASS |
| Full workspace gate at HEAD | `mise run ci` (build + vet + lint + race tests + test-experiments) | all packages ok (cmd 9.955s incl. E2E; experiments 24 pass / 0 fail / 1 guarded skip) | ✓ PASS |
| Lint at HEAD | `mise run lint` | `0 issues.`, exit 0 | ✓ PASS |

### Probe Execution

No probes declared in the plan; conventional `scripts/*/tests/probe-*.sh` discovery returned none. The phase's runnable check surface is the Go test suite + the E2E binary lock, exercised above.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| UIB-01 | 09-01 | requestUserInput (AskUserQuestion shape) bridges to ONE client-rendered form elicitation | ✓ SATISFIED | Truth 1: code + E2E + field capture |
| UIB-02 | 09-01 | One index-keyed property per question; single-select string oneOf / multiSelect array; prompt as message | ✓ SATISFIED | Truth 2: `TestBuildElicitationRequest` + free-text lock |
| UIB-03 | 09-01 | Accept → `{action, content.answers}` merge contract; multi joined ", "; decline/cancel verbatim | ✓ SATISFIED | Truth 3: unit + handler + VM exact-match |
| UIB-04 | 09-01 | Nil-asker honest decline stays; emission failure declines with reason, never fails the turn | ✓ SATISFIED | Truth 4: bridge subtests + pre-existing `assertDeclineStages` lock |
| UIB-05 | 09-01 | Field-verified: dialog renders in Orca, answering feeds the workflow | ✓ SATISFIED (VM pass) / ? human item outstanding (user's /gsd-manager) | Truth 5 + Human Verification item 1 |

No orphaned requirements: REQUIREMENTS.md maps exactly UIB-01..05 to Phase 9, and all five are claimed by plan 09-01.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| `internal/acpport/userinput_internal_test.go` | UIB-02, UIB-03 | 3 | 0 | no | Value (exact field/shape compares) | PROVES |
| `internal/interact/userinputbridge_internal_test.go` | UIB-01, UIB-03, UIB-04 | 3 subtests | 0 | no | Behavioral + value (decoded ask received; dialect result shape) | PROVES |
| `internal/interact/responder_test.go` `assertDeclineStages` | UIB-04 | 1 | 0 | no | Behavioral (schema-valid decline result) | PROVES |
| `cmd/zcode-acp/main_test.go` responder-leg E2E | UIB-01 | 1 subtest | 0 | no | Behavioral (elicitation recorded on client; decline-after-ask in transcript; no unanswered note) | PROVES |

**Disabled tests on requirements:** 0
**Circular patterns detected:** 0
**Insufficient assertions:** 0

**Warning — plan Task 1's declared RED test never landed as named.** `TestDecodeUserInputAsk` (full shape / no-questions / broken JSON) does not exist anywhere in the tree. The full-shape decode is behavior-covered indirectly (`TestHandleUserInputBridge` asserts the asker received the decoded prompt + questions), but the no-questions decode arm and the broken-JSON → decline-with-warn arm (`responder.go:568-578`) have no direct test. Neither arm is a UIB requirement (both are robustness paths whose safe default is an honest decline), so this does not fail the phase — but the plan's test contract drifted during the file consolidation into `internal/acpport`.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | Debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) in phase-modified files | none | Scan clean (the single dialect-doc "placeholder" hit is prose about path-convention documentation, not a stub marker) |
| — | — | Stub/empty-implementation patterns in `userinput.go` | none | The one `return nil` is `strPtr`'s legitimate empty-string → nil-pointer helper |

### Human Verification Required

### 1. UIB-05 second pass — the user's real-machine /gsd-manager acceptance

**Test:** Restart the Orca chat session on the machine where the bridge binary is installed and re-run `/gsd-manager`.
**Expected:** The workflow's choice dialogs now RENDER in the card, and answering one feeds the workflow — the field symptom "окно в третий раз отклонено" is gone.
**Why human:** Requires the user's actual Orca install, their real account, and the real gsd-manager workflow; the VM pass used the fake backend's scripted ask, so this is the only check that proves the real workflow's dialogs surface. (The mechanism it depends on — dialog renders and answer merges — is already field-proven on the VM; this is the acceptance ritual on the user's own setup.)

### Decision Coverage

Skipped — no `*-CONTEXT.md` (no `<decisions>` block) exists in the phase directory; nothing to check.

### Gaps Summary

No gaps. Every UIB requirement verifies against real evidence: the bridge code is substantive and wired end-to-end (decode → one form elicitation → answers merge), the locks are green at HEAD under `-race` (full `mise run ci` green, lint 0 issues), and the live dialog+answer field evidence exists (VM GUI pass with both screenshots, the answer matching the dialect's merge contract exactly).

Two honest deviations are recorded without failing any requirement:
1. The README shipped note from plan Task 5 did not land (only the dialect doc row was updated) — cosmetic doc-completeness drift.
2. Plan Task 1's `TestDecodeUserInputAsk` was never landed; two decode arms (no-questions params, broken JSON → decline) lack direct tests.

Phase status is `human_needed` solely because UIB-05's requirement text lists a second field pass — the user's own `/gsd-manager` re-run — which is still awaiting and inherently requires the user's machine.

---

_Verified: 2026-11-23T12:35:44Z_
_Verifier: Claude (gsd-verifier)_
