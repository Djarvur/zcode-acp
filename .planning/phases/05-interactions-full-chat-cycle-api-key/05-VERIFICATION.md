---
phase: 05-interactions-full-chat-cycle-api-key
verified: 2026-11-21T10:05:00Z
status: gaps_found
score: 10/13 must-haves verified
covered_files:
  - ".golangci.yml"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-01-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-01-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-02-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-02-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-03-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-03-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-04-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-04-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-05-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-05-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-06-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-06-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-07-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-07-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-08-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-08-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-09-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-09-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-REVIEW-DISPOSITION.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-REVIEW.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-UAT.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-USER-SETUP.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-VALIDATION.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/COVERAGE.md"
  - "cmd/fake-app-server/main.go"
  - "cmd/zcode-acp/main.go"
  - "cmd/zcode-acp/main_test.go"
  - "cmd/zcode-acp/testdata/responder-e2e.json"
  - "experiments/README.md"
  - "experiments/captures/EXP-APIKEY-CYCLE-01.jsonl"
  - "experiments/scan-secrets.sh"
  - "go.mod"
  - "go.sum"
  - "internal/acpfront/auth.go"
  - "internal/acpfront/auth_test.go"
  - "internal/acpfront/cancel_test.go"
  - "internal/acpfront/core01_concurrency_test.go"
  - "internal/acpfront/doc.go"
  - "internal/acpfront/front.go"
  - "internal/acpfront/fullcycle_test.go"
  - "internal/acpfront/harness_test.go"
  - "internal/acpfront/initialize_test.go"
  - "internal/acpfront/lifecycle.go"
  - "internal/acpfront/lifecycle_test.go"
  - "internal/acpfront/loop_test.go"
  - "internal/acpfront/newsession_test.go"
  - "internal/acpfront/permission.go"
  - "internal/acpfront/permission_test.go"
  - "internal/acpfront/stream.go"
  - "internal/acpfront/stream_order_test.go"
  - "internal/acpfront/stream_test.go"
  - "internal/acpfront/tool_stream_test.go"
  - "internal/acpfront/turnctx.go"
  - "internal/acpport/agent.go"
  - "internal/acpport/agent_test.go"
  - "internal/acpport/doc.go"
  - "internal/acpport/pin_test.go"
  - "internal/acpport/sink.go"
  - "internal/acpport/types.go"
  - "internal/core/registry.go"
  - "internal/core/registry_test.go"
  - "internal/hostsvc/hostsvc_test.go"
  - "internal/hostsvc/readback.go"
  - "internal/hostsvc/snapshot.go"
  - "internal/interact/cipherdecrypt.go"
  - "internal/interact/cipherdecrypt_test.go"
  - "internal/interact/dedupe.go"
  - "internal/interact/permission.go"
  - "internal/interact/permission_test.go"
  - "internal/interact/responder.go"
  - "internal/interact/responder_test.go"
  - "internal/supervisor/discover.go"
  - "internal/supervisor/e2e_test.go"
  - "internal/turn/machine.go"
  - "internal/turn/machine_test.go"
  - "internal/zcode/conn.go"
  - "internal/zcode/conn_test.go"
  - "internal/zcode/envelope.go"
  - "internal/zcode/fake/fake.go"
  - "internal/zcode/fake/golden_test.go"
  - "internal/zcode/fake/roundtrip_test.go"
  - "internal/zcode/fake/scenario.go"
  - "internal/zcode/fake/scenario_test.go"
  - "internal/zcode/fake/stdio.go"
  - "internal/zcode/fake/testdata/golden/auth-apikey.golden"
  - "internal/zcode/fake/testdata/golden/cancel-mid-turn.golden"
  - "internal/zcode/fake/testdata/golden/complete-then-cancel.golden"
  - "internal/zcode/fake/testdata/golden/completion-quiet-window.golden"
  - "internal/zcode/fake/testdata/golden/full-cycle.golden"
  - "internal/zcode/fake/testdata/golden/newsession-orphan-close.golden"
  - "internal/zcode/fake/testdata/golden/no-reasoning.golden"
  - "internal/zcode/fake/testdata/golden/part-delta-streaming.golden"
  - "internal/zcode/fake/testdata/golden/permission-allow.golden"
  - "internal/zcode/fake/testdata/golden/permission-cancel.golden"
  - "internal/zcode/fake/testdata/golden/reannounce-dedupe.golden"
  - "internal/zcode/fake/testdata/golden/runtime-headers-answer.golden"
  - "internal/zcode/fake/testdata/golden/send-refused-32010.golden"
  - "internal/zcode/fake/testdata/golden/session-lifecycle.golden"
  - "internal/zcode/fake/testdata/golden/setmodel-followup.golden"
  - "internal/zcode/fake/testdata/golden/tool-lifecycle.golden"
  - "internal/zcode/fake/testdata/golden/user-input-decline.golden"
  - "internal/zcode/fake/testdata/scenarios/auth-apikey.json"
  - "internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json"
  - "internal/zcode/fake/testdata/scenarios/complete-then-cancel.json"
  - "internal/zcode/fake/testdata/scenarios/completion-quiet-window.json"
  - "internal/zcode/fake/testdata/scenarios/full-cycle.json"
  - "internal/zcode/fake/testdata/scenarios/newsession-orphan-close.json"
  - "internal/zcode/fake/testdata/scenarios/no-reasoning.json"
  - "internal/zcode/fake/testdata/scenarios/part-delta-streaming.json"
  - "internal/zcode/fake/testdata/scenarios/permission-allow.json"
  - "internal/zcode/fake/testdata/scenarios/permission-cancel.json"
  - "internal/zcode/fake/testdata/scenarios/reannounce-dedupe.json"
  - "internal/zcode/fake/testdata/scenarios/runtime-headers-answer.json"
  - "internal/zcode/fake/testdata/scenarios/send-refused-32010.json"
  - "internal/zcode/fake/testdata/scenarios/session-lifecycle.json"
  - "internal/zcode/fake/testdata/scenarios/setmodel-followup.json"
  - "internal/zcode/fake/testdata/scenarios/tool-lifecycle.json"
  - "internal/zcode/fake/testdata/scenarios/user-input-decline.json"
  - "mise.toml"
  - "test/realdesktop/apicycle_test.go"
  - "test/realdesktop/doc.go"
covered_digest: "v4:sha256:9f8123e537da01fb0fbaecaa458dce3b2227c8a04a07eae4de7fc0172788cc6e"
behavior_unverified: 1
overrides_applied: 0
unverified_prohibitions: 1
re_verification:
  previous_status: human_needed
  previous_score: 11/13
  gaps_closed: []
  gaps_remaining:
    - "REAL RUN UAT item 1 (live authenticate read-back) — attempt 1 FAILED (commit d15a745); retry gated by gap 1 (read-back surface) and gap 2 (CR-01) fixes"
    - "REAL RUN UAT item 2 (evidence recording: scrub + ledger flip) — pending the passing run"
  regressions:
    - "Truth 7 (close-path permission fan-in) — prior pass VERIFIED, now FAILED (partial): the prior evidence verified wiring PRESENCE at lifecycle.go:135, not return semantics; 05-REVIEW CR-01 (confirmed first-hand this pass) shows the fan-in is skipped on exactly the successful-drain path. Code unchanged since a4e0483 — the prior verdict was incomplete, not a code regression."
gaps:
  - truth: "SC4: authenticate{zai-api-key} on a REAL account enables working turns, with success verified by reading registry state back — the real-account half is field-FAILED: the read-back never confirms on real desktop 3.18.2"
    status: failed
    reason: "UAT attempt 1 (2026-11-21, orchestrator-run, commit d15a745) reached the real account: updateAccountConfig acked {status:received, providerCount:0}, refreshRegistry acked {refreshed:true}, the desktop applied runtime headers in its own refresh loop — but NO state.updated registry patch and no create-reply surface arrived within the 30 s window, so awaitRegistryContains timed out → ErrRegistryNotConfirmed → authenticate errored and zero turns ran (zero model spend). The fake-tier read-back surface does not exist on the real 3.18.2 wire; the machinery is honest (it failed loud instead of faking success — the 05-07 prohibitions held), but the must-have behavior is observably false on the real account."
    artifacts:
      - path: "internal/hostsvc/readback.go"
        issue: "awaitRegistryContains polls state.updated / create-reply surfaces the real desktop never emits for this push (fake-tier-oracle mismatch)"
      - path: "internal/acpfront/auth.go"
        issue: "the honest chain is correct by construction; its terminal read-back step cannot confirm against the real 3.18.2 desktop"
    missing:
      - "Diagnose which candidate cause applies (per 05-UAT.md item 1): (a) basedOnZCodeBuiltinRevision hash mismatch (runtime self-reports 0.17.1 vs 3.18.2 path-derived hash) → dialect §8 silent discard; (b) empty-providers snapshot treated as nothing-to-apply and never patched; (c) the read-back surface should be the first session/create reply (settings.model.available), not a state.updated wait — resolve via a scrubbed-capture experiment on the real desktop before changing the verdict logic"
      - "Fix the read-back chain to the field-verified surface, keep the ack-never-trusted discipline (05-06/05-07 prohibitions), update the fake oracle to the corrected wire truth, then re-run the guarded REAL RUN (UAT item 1) to green"
  - truth: "05-02: no permission bridge outlives its turn — the cancel fan-in fires on EVERY cancelled-turn path (CR-01: CloseSession skips it when the drain succeeds)"
    status: failed
    reason: "drainActiveTurn (internal/acpfront/lifecycle.go:178-213) returns FALSE after a successful drain — iteration 1 finds the active turn, cancels, waits on done; done closes only inside release() after turnActive was cleared; the loop re-enters with done==nil and returns false (line 192). The only return true is the deadline-elapsed failure path (line 205). CloseSession (lifecycle.go:131-136) gates f.perm.cancelPendingForTurn() behind that inverted return, so a close-with-active-turn cancels the turn but SKIPS the permission fan-in — the pending bridge dangles until the responder's 300 s permission deadline, then frames a deny at an already-closed backend session. Front.Cancel (front.go:396-404) performs the same fan-in unconditionally, proving the asymmetry is not deliberate. No test covers close-with-pending-permission (lifecycle_test.go has no permission-reverse close test — grep confirmed this pass). Confirmed first-hand in code this pass; matches 05-REVIEW.md CR-01 (disposition: open, gates the REAL RUN retry)."
    artifacts:
      - path: "internal/acpfront/lifecycle.go"
        issue: "drainActiveTurn reports the opposite of its documented contract on the normal path (return false at :192 after a successful drain; caller gates the fan-in on it at :131-136)"
      - path: "internal/acpfront/lifecycle_test.go"
        issue: "no close-with-pending-permission test — TestCloseTeardown/TestCloseWhileTurnActiveRace run turns with no permission reverse"
    missing:
      - "Fix drainActiveTurn to report whether an active turn of THIS session was observed and cancelled (sawTurn), not whether the drain timed out (the 05-REVIEW.md CR-01 fix shape)"
      - "Add the lock test: close a session whose active turn holds a pending permission ask; assert the pendings are cancelled (-32021 on the fake transcript or pending-count zero) — per the 05-02 prohibition 'MUST NOT let a permission bridge outlive its turn'"
      - "Batch WR-01..04 with this close-path fix per 05-REVIEW-DISPOSITION.md before the REAL RUN retry"
behavior_unverified_items:
  - truth: "SC5: the full cycle — new → prompt with tool calls and a permission decision → resume → close — runs end-to-end ON THE REAL api-key account (live half; fake tier fully proven)"
    test: "After gaps 1-2 close: run the canonical guarded command (ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/) per 05-USER-SETUP.md and record the evidence trail (scrub to experiments/captures/EXP-APIKEY-CYCLE-01.jsonl, V-EXP id, ledger actuals — UAT item 2)"
    expected: "The cycle completes on the real account with zero orphans: authenticate confirmed by the (corrected) read-back, two tiny turns, resume reattaches without replay, close tears down cleanly, post-close prompt rejected"
    why_human: "Real credentials, unlocked keyring, ledgered quota spend on the real 3.18.2 desktop — the guarded double guard makes the live path unrunnable in CI by contract; the fake tier cannot observe the real desktop's surfaces (that mismatch is exactly what attempt 1 proved)"
human_verification:
  - test: "END-OF-PHASE REAL RUN retry (UAT item 1, 05-UAT.md linkage preserved — currently FAIL recorded for attempt 1): FIRST land the gap-closure fixes (auth read-back surface per gap 1; CR-01 drainActiveTurn + close-with-pending-permission test per gap 2 — the disposition ledger gates the retry on CR-01), then per 05-USER-SETUP.md: export ZCODE_ACP_API_KEY from the local ZCode desktop config (never commit), confirm the keyring unlocks (path-string-hash secret-tool check, field-verified in the corrected 05-USER-SETUP.md), set ZCODE_ACP_RAW_CAPTURE outside the repo, and run ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/"
    expected: "PASS — authenticate proven by the registry read-back verdict (never the ack), one tiny tool-less turn + one file-read tool turn complete, session/resume (re-issued subscribe, no replay), session/close (clean teardown, post-close prompt rejected), staged shutdown zero orphans; runtime-headers ask answered by the wired key source (CORE-06); the mid-turn cipher ask answered from the keyring (D-06 real-tier proof)"
    why_human: "External-service integration on a real account: real credentials, unlocked GNOME keyring, ledgered quota spend (EXP-APIKEY-CYCLE-01, D-05 minimum) — the double guard keeps CI away by design, and the real desktop's surfaces are observable only by running against it"
  - test: "Evidence recording after the passing run (UAT item 2): node experiments/scrub-capture.mjs <raw-capture> experiments/captures/EXP-APIKEY-CYCLE-01.jsonl --redact-path <workspace>; bash experiments/scan-secrets.sh; flip the Phase-5 ledger row in experiments/README.md from PENDING to actuals (attempt-1 fail line already recorded; append the passing run's spend + verdict)"
    expected: "Scrubbed capture lands with the V-EXP id, scan-secrets exits clean, the ledger row records the passing run's actuals (spend, never credentials)"
    why_human: "Credential hygiene and quota accounting are human judgment acts on machine-local material; the scrub-first pipeline cannot decide by itself what the raw capture contains"
---

# Phase 5: Interactions & Full Chat Cycle (api-key) — Verification Report

**Phase Goal:** On a real z.ai account via api-key, a client gets a complete chat cycle — streamed tool calls, permission decisions answered from the client, working session lifecycle — with the host-managed interaction responder always on.
**Verified:** 2026-11-21T10:05:00Z
**Status:** gaps_found
**Re-verification:** Yes — stale-verification refresh (prior report 2026-11-20T16:36:59Z, human_needed; covered evidence changed after it was written and two new evidence sources — the failed REAL RUN and today's deep review — bear directly on must-haves)

## Why This Refresh Changed the Verdict

No implementation code changed since the prior verification (git confirms the tree is byte-identical from a4e0483 through HEAD for internal/, cmd/, test/ — only docs and the scrubbed capture landed). What changed is EVIDENCE, and two prior claims do not survive it:

1. **SC4's real half moved from unexercised to field-FAILED.** The prior pass honestly marked it PRESENT_BEHAVIOR_UNVERIFIED. The REAL RUN (UAT attempt 1, commit d15a745) then exercised it and it failed: the authenticate read-back never confirms on the real 3.18.2 desktop → ErrRegistryNotConfirmed → zero turns. The phase goal's opening clause — "On a real z.ai account via api-key" — is observably not met today.
2. **Truth 7's close-path half was falsified by review.** The prior pass verified the fan-in's wiring PRESENCE at lifecycle.go:135. 05-REVIEW.md CR-01 — confirmed first-hand this pass by reading `drainActiveTurn` (lifecycle.go:178-213) and its caller (:131-136) — shows the return value is inverted, so the fan-in is skipped exactly when a turn was actually cancelled on close. The prior verdict was incomplete; this pass corrects it.

Everything else the prior pass verified remains verified: every phase-critical package was re-run green under -race on this exact tree this pass.

**Status rationale.** The refresh directive said to mark human_needed if human items remain. Two must-have truths are now FAILED (field-proven auth gap; first-hand-confirmed CR-01), and the Step-9 decision tree puts gaps_found ahead of human_needed when FAILED truths exist — gaps_found is also what routes the CR-01/read-back fixes into a closure plan, which the disposition ledger requires before the UAT retry. The UAT linkage is preserved intact (05-UAT.md untouched; its items 1-2 are the human_verification section below). Both statuses block phase advancement; gaps_found additionally preserves the failing truths for `/gsd:plan-phase --gaps`.

## Goal Achievement

### User Flow Coverage (MVP)

ROADMAP marks this phase `mode: mvp`; the goal is outcome-phrased and fails the `user-story.validate` guard (same as phases 01-04 — this milestone's uniform precedent of deriving the flow table from the roadmap contract applies).

| # | Step | Expected | Evidence in Codebase | Status |
|---|------|----------|---------------------|--------|
| 1 | Client authenticates with a z.ai api-key (fake tier) | `authenticate{zai-api-key}` → `{}` ONLY after the registry read-back verdict; missing key → honest coded error naming `ZCODE_ACP_API_KEY`; no desktop-config fallback | TestAuthenticate 5 arms green (acpfront pkg re-run green 23.3s -race this pass); auth-apikey + runtime-headers-answer goldens replay green (zcode/fake 4.9s); auth.go chain read first-hand (prior pass + unchanged tree) | ✓ (fake tier) |
| 2 | Client authenticates on the REAL account | Read-back confirms on real desktop 3.18.2 and turns work | **FIELD-FAILED**: UAT attempt 1 — acks received, no state.updated patch in 30 s → ErrRegistryNotConfirmed, zero turns, zero spend (05-UAT.md item 1, experiments/README.md ledger) | ✗ FAILED (gap 1) |
| 3 | Client opens a session | `session/new{cwd}` → create+subscribe+setModel; Phase-4 contracts intact | setmodel-followup golden replay green this pass; front.go NewSession unchanged | ✓ |
| 4 | Agent streams tool use | `tool_call`/`tool_call_update` with title, kind, status transitions, absolute-path diffs | TestToolLifecycle + backpressure + a4e0483 sites green under -race this pass; tool-lifecycle + full-cycle goldens replay green | ✓ |
| 5 | Backend asks permission; client decides | `session/request_permission` with backend options verbatim; chosen option's response forwarded byte-verbatim; ACP-side cancel → `-32021` | TestPermission* green under -race this pass; permission-allow/cancel goldens replay green; fan-in unconditional on the Front.Cancel path (front.go:396-404 read first-hand) | ✓ (ACP-cancel path) |
| 6 | Client resumes a session | `session/resume` reattaches: re-issued subscribe, NO setModel, NO replay | lifecycle.go:55-99 read first-hand; TestResume green this pass; session-lifecycle golden green. Open review caveats: WR-02 (client cwd dropped → relative diff paths on resumed sessions), WR-03 (resume ungated on the closing set → close-vs-resume observer-leak race) — both confirmed first-hand in code this pass | ✓ (tests) / ⚠ open WR-02, WR-03 |
| 7 | Client closes a session | Ordered teardown; post-close prompt rejected; close-vs-prompt race one outcome | TestCloseTeardown + TestCloseWhileTurnActiveRace green under -race this pass; closing-gate read first-hand. **But the close-path permission fan-in is broken (CR-01)**: drainActiveTurn returns false after a successful drain (lifecycle.go:192), so `cancelPendingForTurn()` at :135 is skipped exactly when a turn was cancelled — pending bridge dangles to the 300 s deadline | ✗ FAILED (gap 2 — close-with-pending-permission) |
| 8 | Full cycle composed (fake tier) | authenticate → new → prompt(tool+permission) → end_turn → resume → close, golden-locked | TestFullCycle green under -race this pass (acpfront 23.3s); full-cycle golden replay green | ✓ |
| 9 | Full cycle live on the REAL api-key account | Same cycle on the real install; scrubbed capture + V-EXP + ledger actuals | Blocked upstream at step 2 (authenticate fails); guarded test exists and skips honestly (re-verified this pass); UAT item 2 pending | ⚠️ human (items below) |

### Observable Truths

Merged from ROADMAP SC1-SC5 (contract) + the nine PLAN frontmatter must_haves (clusters T6-T13), same structure as the prior pass. Score: 10/13 — two FAILED (SC4 real half; truth 7's close-path half), one PRESENT_BEHAVIOR_UNVERIFIED (SC5 live half).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: every server→host `interaction/*` answered promptly — dedupe, per-method deadlines, unknown → -32601, never a crash | ✓ VERIFIED | interact pkg green under -race this pass (2.05s): TestResponderContract (unknown→-32601 naming the method; deadlines; adjacency/empty probes), TestResponderDedupeRace; reannounce-dedupe + unknown-host-method + user-input-decline goldens replay through a real Conn (zcode/fake 4.9s green); production wiring `conn.OnRequest = responder.Serve` before traffic (main.go:252, re-grepped this pass) |
| 2 | SC2: tool use streams as tool_call/tool_call_update with title, kind, status transitions, file diffs | ✓ VERIFIED | acpfront pkg green under -race this pass (23.3s incl. TestToolLifecycle, TestStreamBackpressure, TestStreamBeforeStopReason); tool-lifecycle + full-cycle goldens replay green; stream.go unchanged since prior first-hand read |
| 3 | SC3: backend permission ask reaches the client with allow/reject options; chosen outcome forwarded; ACP-side cancel resolves pending as cancelled | ✓ VERIFIED | TestPermissionBridge/Cancel/CancelRace/Expiry green under -race this pass; permission-allow golden byte-verbatim + permission-cancel -32021 replay green; the ACP-cancel fan-in is unconditional in Front.Cancel (front.go:396-404, read first-hand this pass). Scope note: SC3's cancel path is Front.Cancel — the CLOSE path fan-in is truth 7's failure, not this truth's claim |
| 4 | SC4: `authenticate{zai-api-key}` on a REAL account enables working turns, success verified by registry read-back, never the ack | ✗ FAILED | Fake-tier machinery verified (TestAuthenticate ack-negative arm, hostsvc green 1.2s -race this pass). Real-account half FIELD-FAILED: UAT attempt 1 (d15a745) — updateAccountConfig acked {status:received, providerCount:0}, refreshRegistry acked {refreshed:true}, desktop applied runtime headers in its own loop, but no state.updated patch / create-reply surface within 30 s → ErrRegistryNotConfirmed → zero turns, zero spend. The fake oracle's read-back surface does not exist on the real 3.18.2 wire (gap 1) |
| 5 | SC5: resume reattaches without replay, close cleans up; the full cycle runs end-to-end on api-key | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Fake-tier cycle fully proven this pass (TestFullCycle green under -race; session-lifecycle golden; TestResume/TestCloseTeardown/TestCloseWhileTurnActiveRace green). Live half unexercised — blocked upstream at authenticate by truth 4's field failure; the guarded run is the pending human act. Review adds open resume/close-path caveats (WR-02 cwd drop, WR-03 closing-gate absence — both confirmed first-hand) that the live run will also exercise |
| 6 | 05-01: per-method deadlines (15 s runtime-prefs, 300 s permission → deny, runtime-headers coded error until key); user-input → {action:decline}; bypass lane; RouterEvent expiry records | ✓ VERIFIED | Responder policy table unchanged; interact green -race this pass; user-input-decline + runtime-preferences goldens replay green; structural-log locks green |
| 7 | 05-02: deny-by-default (unknown/missing optionId, malformed outcome, expiry → deny or -32021, NEVER allow); no bridge outlives its turn (cancel fan-in) | ✗ FAILED (partial) | Deny-by-default half verified (denyAnswer/cancelledWireError paths, TestPermissionExpiry/CancelRace green this pass). The no-dangling-bridge invariant HOLDS on the Front.Cancel path (unconditional fan-in, test-locked) but is BROKEN on the CloseSession path: drainActiveTurn's inverted return (lifecycle.go:192 vs contract at :173-175) skips `cancelPendingForTurn()` (:135) exactly when a turn was cancelled; no close-with-pending-permission test exists. Matches 05-REVIEW CR-01, disposition open, gates the REAL RUN retry (gap 2) |
| 8 | 05-03/05-05: router fan-out (WR-01), observer removes (WR-07), registry Remove consumer, WR-04 error-path orphan close, close-vs-prompt race gate (T-5-12) | ✓ VERIFIED | machine.go fan-out registration + CloseObservers consumed at lifecycle.go:88-89,159; registry Remove :158; closing-gate gates Prompt (front.go:318) and acquireTurn (turnctx.go:384); all packages green this pass; newsession-orphan-close golden green. Scope note: the claimed close-vs-PROMPT race gate holds; the close-vs-RESUME gap is review WR-03 (open warning, unproven by test either way) — recorded, not scored here |
| 9 | 05-04 + a4e0483: WR-03 pump (non-blocking enqueue, keep-latest coalescing, byte ceiling fail-loud, flush-before-stopReason; a4e0483 fix sites locked) | ✓ VERIFIED | All locked tests green in the acpfront -race run this pass (TestStreamBackpressure 3 legs; TestStreamBeforeStopReason; full-cycle golden started-card-before-ask) |
| 10 | 05-06: strict snapshot envelope with EXACT-revision formula REUSED from supervisor.BuiltinRevision; read-back never the ack; runtime-headers success shape only from the injected key | ✓ VERIFIED | hostsvc green -race this pass (1.2s) incl. the ack-mismatch negative; snapshot.go:135 reuse + :258 handler unchanged; runtime-headers-answer golden green. The FIELD gap is the surface the read-back waits on (truth 4 / gap 1) — the machinery itself behaved exactly as this truth specifies, at both tiers (on the real wire it honestly errored instead of trusting the ack) |
| 11 | 05-07: honest authenticate chain (record→push→refresh→read-back→{}), no-key coded error naming the env var, interruptible clean retry, CredentialDecrypt seam, hygiene locks, binary wires the stack before traffic | ✓ VERIFIED | acpfront + interact + cmd all green under -race this pass; wiring re-grepped (main.go:246-252, :294); hygiene held IN THE FIELD: the failed real run errored loud (ErrRegistryNotConfirmed) instead of fabricating success, and scan-secrets exits 0 on the tree incl. the new scrubbed capture (re-run this pass) |
| 12 | 05-08: fake-tier FULL cycle — one continuous composition, golden-locked, nothing hangs/drops/fabricates | ✓ VERIFIED | TestFullCycle green under -race this pass (within the 23.3s acpfront run); full-cycle golden replay green (zcode/fake 4.9s) |
| 13 | 05-09: guarded real-tier test (double guard; skips ~2 ms; default suite untouched); quota ledger row exists before any run; no credential material in repo | ✓ VERIFIED | Re-ran both guard checks this pass: no-tag build → `[no test files]` (file excluded); `-tags realdesktop` without env → honest SKIP in 0.00s. Ledger row now honestly records attempt-1 failure (experiments/README.md, commit d15a745) and stays PENDING for the passing run; scrubbed capture (10 lines) present with zero key-pattern matches; scan-secrets exit 0 this pass |

**Score:** 10/13 truths verified (1 present, behavior-unverified — SC5's live half; 2 FAILED — SC4's real half and truth 7's close-path half)

**Prohibitions (21 across all plans):** 20 hold — re-confirmed where re-runnable this pass (deny paths test-locked; goldens byte-lock traceability; auth success only on read-back — the field run PROVED this discipline by failing loud; double guard re-run; scan-secrets 0; no debt markers re-grepped). **1 FLAGGED:** 05-02's "MUST NOT let a permission bridge outlive its turn" is violated on the close-with-active-turn path (CR-01, gap 2). `unverified_prohibitions: 1`.

### Required Artifacts

The implementation tree is byte-identical to the prior pass (git: zero non-doc diffs a4e0483..HEAD across internal/, cmd/, test/), and the prior pass verified all 27 plan-declared artifact paths first-hand (the `verify.artifacts` verb's 0/0 on inline-format plan artifacts is a vacuous parse, not a negative verdict). No artifact regressed; the substantive first-hand evidence carries over and the owning packages re-ran green this pass. New/changed evidence artifacts:

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| internal/acpfront/lifecycle.go | resume/close contracts | ✗ DEFECT (CR-01) | Teardown order correct; drainActiveTurn return semantics inverted (:178-213 vs contract :173-175; caller :131-136) — gap 2 |
| internal/acpport/agent.go | resume delegate | ⚠ open WR-02 | ResumeSession forwards only the session id (:375-384); `params.Cwd` dropped → no `cwds` entry on resume → relative diff paths (Pitfall-5 shape on the resume path) |
| experiments/README.md | quota ledger | ✓ VERIFIED | Row honestly records attempt-1 failure with actuals (zero spend) and stays PENDING for the passing run — commit d15a745 |
| experiments/captures/EXP-APIKEY-CYCLE-01.jsonl | scrubbed capture | ✓ VERIFIED | 10 lines; zero key-pattern matches; scan-secrets clean (re-run this pass) |
| 05-USER-SETUP.md | human run instructions | ✓ VERIFIED | Keyring check corrected to the field-verified path-string-hash form (commit eac7365) |
| 05-REVIEW.md + 05-REVIEW-DISPOSITION.md | deep review + ledger | ✓ VERIFIED | 8 findings (1C/4W/3I), all open, CR-01 gating the retry — both re-read this pass; CR-01/WR-02/WR-03/WR-04 confirmed first-hand in code |

### Key Link Verification

The `verify.key-links` verb's filename-literal matcher yields structural false negatives on Go sources (prior pass diagnosed: it greps for the target file path string inside the source, which Go's package-import model never contains). All 25 declared links were verified first-hand in the prior pass on this identical tree; the load-bearing production wiring was re-grepped this pass:

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/interact/responder.go | internal/zcode/conn.go | OnRequest seam | WIRED | main.go:252 `conn.OnRequest = responder.Serve` before traffic (re-grepped this pass) |
| cmd/zcode-acp/main.go | internal/interact (forwarder/keys/cipher) | constructor seams | WIRED | main.go:246-248 WithPermissionForwarder / WithRuntimeHeadersKeySource / WithCredentialDecrypt; :294 NewStaticKey (re-grepped this pass) |
| internal/acpfront/front.go | internal/acpfront/permission.go | cancel fan-in | WIRED but GATED-WRONG on close | front.go:403 unconditional (correct); lifecycle.go:135 gated behind the inverted drainActiveTurn return (gap 2) |
| all other 22 declared links | — | — | WIRED (carried) | Tree identical to the pass that verified them first-hand; owning packages green this pass |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| permission bridge | client outcome | ACP client selection | forwarded byte-verbatim (golden) | ✓ FLOWING |
| tool stream | tool frames | backend tool.updated envelopes | golden-locked traceability | ✓ FLOWING |
| authenticate verdict | registry surfaces | state.updated / create-reply via conn observer | fake tier ✓; real 3.18.2 desktop never emits the awaited surface → verdict never confirms | ✗ DISCONNECTED (real tier — gap 1) |
| runtime-headers reply | apiKey material | injected KeySource | success shape only with key | ✓ FLOWING |
| full-cycle frames | whole-cycle wire | scripted corpus row | golden-locked | ✓ FLOWING (fake tier) |
| resumed-session diffs | cwd resolution | `cwds` map | resume never records cwd → resolution degrades to relative paths | ⚠ (WR-02) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CORE-06 + 05-02 bridge machinery (SC1/SC3 fake tier) | `go test ./internal/interact/ -race -count=1` | ok 2.048s | ✓ PASS |
| AUTH-01 machinery (snapshot/read-back/runtime-headers) | `go test ./internal/hostsvc/ -race -count=1` | ok 1.226s | ✓ PASS |
| Front machinery: tool/permission/auth/lifecycle/full-cycle (SC2/SC3/SC4/SC5 fake tier) | `go test ./internal/acpfront/ -race -count=1` | ok 23.343s | ✓ PASS |
| Corpus rows + goldens through real Conn | `go test ./internal/zcode/... ./internal/turn/ ./internal/core/ -count=1` | all ok (fake 4.922s) | ✓ PASS |
| Binary E2E incl. wired responder | `go test ./cmd/zcode-acp/ -race -count=1` | ok 3.112s | ✓ PASS |
| Guarded real-tier test: default build exclusion | `go test ./test/realdesktop/ -count=1` | `[no test files]` exit 0 | ✓ PASS |
| Guarded real-tier test: tag without env | `go test -tags realdesktop ./test/realdesktop/ -count=1 -run TestAPIKeyFullCycle -v` | SKIP 0.00s, honest manual-only message | ✓ PASS (honest skip) |
| Credential hygiene on the current tree (incl. new capture) | `bash experiments/scan-secrets.sh` | exit 0; capture has zero key-pattern lines | ✓ PASS |
| CR-01 first-hand code verification | read lifecycle.go:103-213 + front.go:396-404 | inverted return confirmed; fan-in skipped on successful drain; Front.Cancel unconditional | ✗ CONFIRMED DEFECT |
| Close-with-pending-permission test coverage | grep lifecycle_test.go | zero permission-reverse close tests | ✗ COVERAGE HOLE |
| Live run on the real account | guarded command | NOT re-run — attempt 1 failed (d15a745); retry gated on gaps 1-2 | ? → human item 1 |

### Probe Execution

No probe scripts declared or conventional (`scripts/*/tests/probe-*.sh` absent). Step 7c: not applicable.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| AUTH-01 | 05-06, 05-07, 05-09 | api-key path: updateAccountConfig + requestProviderRuntimeHeaders answered correctly (registry state verified, not the ack) | ✗ BLOCKED (real tier) | Machinery satisfied at fake tier; the field run proves the read-back verdict cannot confirm on real desktop 3.18.2 → api-key does not yet enable working turns. NOTE: REQUIREMENTS.md marks AUTH-01 "[x] Complete" — that checkbox predates the field evidence and should be reverted by the closure work |
| CORE-06 | 05-01, 05-02, 05-08, 05-09 | Host-managed interaction responder always on | ✓ SATISFIED | responder green -race + goldens + wired before traffic (main.go:252); held in the field (the wrapper answered/stayed honest through the failed run's window) |
| SESS-01 | 05-03, 05-05, 05-08, 05-09 | resume (no replay) and close (child-side cleanup) | ✓ SATISFIED (tests) — open WR-02/WR-03 on the resume path | lifecycle contracts + tests green this pass; live confirmation rides UAT item 1 |
| TOOL-01 | 05-04, 05-08, 05-09 | tool_call/tool_call_update streamed | ✓ SATISFIED | stream tap + pump; all locks green this pass |
| TOOL-02 | 05-02, 05-08, 05-09 | permission bridged; cancelled outcome on ACP-side cancel | ✓ SATISFIED (ACP-cancel path) | TestPermission* green; goldens green. The close-path fan-in defect (CR-01) is tracked under the 05-02 plan prohibition / truth 7, not the requirement's stated behavior |

Orphaned requirements: none — REQUIREMENTS.md maps exactly AUTH-01, CORE-06, SESS-01, TOOL-01, TOOL-02 to Phase 5, and every ID appears in plan frontmatter (05-09 carries all five; frontmatter and REQUIREMENTS.md agree — no divergence).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/acpfront/lifecycle.go | 178-213 (caller 131-136) | CR-01: drainActiveTurn inverted return → close skips permission fan-in on the path it exists for; no covering test | 🛑 Blocker | Pending permission bridge dangles to the 300 s deadline on close-with-active-turn, then denies at a closed backend session — the exact dangling-pending shape the fan-in exists to prevent (gap 2) |
| internal/interact/responder.go | 225-249 | WR-01: parked dedupe-join arm can frame a bare id-echo reply in the cancelBase→rw.Close window, contradicting the never-send invariant | ⚠ Warning (open) | Nondeterministic payload-less frame at conn close with pending joins |
| internal/acpport/agent.go + lifecycle.go | 375-384 / 55-99 | WR-02: resume drops the client cwd → relative diff/location paths on resumed sessions (Pitfall 5 on the resume path) | ⚠ Warning (open) | Broken tool cards on every resumed session's first turn with a relative backend path |
| internal/acpfront/lifecycle.go | 55-99 | WR-03: resume not gated on the closing set → close-vs-resume race leaks machine observers / leaves a zombie registration | ⚠ Warning (open) | Observer leak + silent re-closable zombie under a racy interleaving |
| internal/acpfront/front.go | 333-340 | WR-04: prompt cancel fan-in goroutine can cancel a later, unrelated turn when both ctx arms are ready | ⚠ Warning (open) | Narrow-window silent wrong-turn cancellation |
| test/realdesktop/apicycle_test.go | 239-249, 508-541 | IN-01 malformed ZCODE_ACP_MODEL silently defaulted (real spend on unintended model); IN-02 raw-capture path not enforced outside repo | ℹ️ Info | Hardening backlog per disposition ledger |
| cmd/zcode-acp/main.go | 307-314 | IN-03 shutdown select can mask a child crash with exit 0 | ℹ️ Info | Nondeterministic exit verdict only |
| — | — | Debt markers (TBD/FIXME/XXX) | none | Zero matches re-grepped this pass |

### Deferred Items

None. Checked phases 6-7 of the milestone: Phase 6 is the OAuth path (AUTH-02/03), Phase 7 is CI conformance/VM/docs — neither covers the api-key read-back surface fix (AUTH-01 is Phase 5's own requirement) nor the close-path fan-in fix. The REAL RUN is this phase's contract (05-09), not Phase 7's manual E2E. No gap is deferred.

### Human Verification Required

### 1. End-of-phase REAL RUN retry on the real z.ai api-key account (EXP-APIKEY-CYCLE-01 — UAT item 1, attempt 1 FAIL recorded)

**Test:** FIRST the gap-closure work lands (gap 1: auth read-back surface — diagnose via a scrubbed-capture experiment on the real desktop, then fix the verdict chain and the fake oracle; gap 2: CR-01 drainActiveTurn + the close-with-pending-permission lock test — the disposition ledger explicitly gates the retry on CR-01). Then per 05-USER-SETUP.md: export `ZCODE_ACP_API_KEY` from the local ZCode desktop config (never commit), confirm the keyring check resolves 43 bytes (path-string-hash form), set `ZCODE_ACP_RAW_CAPTURE` outside the repo, and run `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/`.
**Expected:** PASS — authenticate proven by the registry read-back verdict (never the ack), one tiny tool-less turn + one file-read tool turn, session/resume (no replay), session/close (clean teardown, post-close prompt rejected), staged shutdown zero orphans; runtime-headers ask answered by the wired key source; mid-turn cipher ask answered from the keyring.
**Why human:** Real credentials, unlocked keyring, ledgered quota spend on the real 3.18.2 desktop — the double guard keeps CI away by design; the fake tier structurally cannot observe the real desktop's surfaces (that mismatch is what attempt 1 proved).

### 2. Evidence recording after the passing run (UAT item 2)

**Test:** `node experiments/scrub-capture.mjs <raw> experiments/captures/EXP-APIKEY-CYCLE-01.jsonl --redact-path <workspace>`; `bash experiments/scan-secrets.sh`; flip the Phase-5 ledger row in experiments/README.md to the passing run's actuals (spend + verdict line beneath the attempt-1 record).
**Expected:** Scrubbed capture with the V-EXP id; scan-secrets clean; ledger row records the passing run's actuals (spend, never credentials).
**Why human:** Credential hygiene and quota accounting are human judgment acts on machine-local material.

### Gaps Summary

The phase's codebase deliverables remain solid — every phase-critical package re-ran green under -race on this tree this pass (interact 2.0s, hostsvc 1.2s, acpfront 23.3s, zcode/turn/core green, cmd 3.1s), the double guard holds, hygiene holds (scan-secrets 0 incl. the new scrubbed capture), zero debt markers. But the phase GOAL is not achieved: two must-have truths failed on evidence gathered since the prior report.

**Gap 1 — the real-account authenticate read-back (SC4's defining half, field-proven failing).** The fake-tier oracle promised a read-back surface (state.updated registry patches / create-reply model surfaces) that the real 3.18.2 desktop never emits for this push: attempt 1 got acks and even saw the desktop apply runtime headers in its own loop, but the 30 s read-back window expired empty → ErrRegistryNotConfirmed → zero turns. This is exactly the failure mode the honest chain was designed to surface (it refused to trust the ack — the 05-06/05-07 prohibitions held under field conditions), but the verdict chain's surface assumption is wrong for the real wire. Three candidate causes are recorded in 05-UAT.md (basedOn hash path mismatch → §8 silent discard; empty-providers nothing-to-apply; create-reply as the true surface); a capture experiment on the real desktop must resolve which before the fix.

**Gap 2 — the close-path permission fan-in (CR-01, confirmed first-hand).** `drainActiveTurn` returns false after a successful drain, so `CloseSession` skips `cancelPendingForTurn()` exactly when a turn was cancelled — a pending permission ask dangles to the 300 s deadline and then denies at a closed backend session. The same fan-in is unconditional on the Front.Cancel path, so the asymmetry is a defect, not a design. No test covers close-with-pending-permission. The disposition ledger correctly gates the REAL RUN retry on this fix (a real close-with-pending-ask mid-run would wedge the cycle).

Both gaps are in-scope for this phase (AUTH-01 and the permission/lifecycle half of the goal). The closure plan MUST cite 05-REVIEW-DISPOSITION.md (all 8 findings open; CR-01 gates the retry; WR-01..04 batch with the close-path fix). Grouped root causes: gap 1 is the auth/read-back concern (hostsvc/auth + real-desktop dialect), gap 2 is the close/permission teardown concern (acpfront lifecycle) — two focused work items, then UAT items 1-2 re-run.

---

_Verified: 2026-11-21T10:05:00Z_
_Verifier: Claude (gsd-verifier)_
