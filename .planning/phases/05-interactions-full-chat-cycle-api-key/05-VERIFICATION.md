---
phase: 05-interactions-full-chat-cycle-api-key
verified: 2026-11-21T21:10:00Z
status: gaps_found
score: 12/13 must-haves verified
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
  - "experiments/captures/EXP-APIKEY-CYCLE-02.jsonl"
  - "experiments/captures/EXP-APIKEY-CYCLE-03.jsonl"
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
  - "internal/zcode/fake/testdata/golden/lingering-latch-32010.golden"
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
  - "internal/zcode/fake/testdata/scenarios/lingering-latch-32010.json"
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
  - "test/realdesktop/readbackdiag_test.go"
covered_digest: "v4:sha256:9374438d93b92d052554cf4fb69e8e7805a8103dd06518742d4605a67c4dbe6b"
behavior_unverified: 0
overrides_applied: 0
unverified_prohibitions: 1
re_verification:
  previous_status: gaps_found
  previous_score: 10/13
  gaps_closed:
    - "Gap 1 — SC4's real-account half: the authenticate read-back now latches on the field-verified surface (probe-create reply's settings.model.available, plan 05-11 branch (c), EXP-READBACK-DIAG-01) and is FIELD-PROVEN: REAL RUN attempt 3 PASSED 49.43s on the real 3.18.2 account (2026-11-21, human-run per the 05-14 blocking-human checkpoint); UAT item 1 recorded pass, ledger row flipped to actuals, scrubbed capture EXP-APIKEY-CYCLE-03.jsonl committed"
    - "Gap 2 — CR-01 close-path permission fan-in: drainActiveTurn now reports sawTurn (true on both the successful-drain and deadline arms, false only on an idle close — lifecycle.go:221-260) and CloseSession fans the pendings in on exactly that signal (:170-175); locked red-first by TestCloseWithPendingPermissionCancelled (exactly-one -32021 reply) and TestCloseDeadlineArmFansPendingIn (1ns seam); delta review confirms closed with wire locks"
  gaps_remaining:
    - "Truth 7's no-dangling-bridge invariant — NEW gap 3 (WR-06): the SDK displaced-prompt arm resolves the turn but never fans its pending permission bridge in; the bridge rides the 300s responder deadline then answers deny for a cancelled turn. Deterministic, client-triggerable, and enshrined as expected behavior by committed tests (TestCancelFanInOnlyWhenSlotHeld + assertDanglingAskNote)"
  regressions: []
gaps:
  - truth: "05-02: deny-by-default AND no permission bridge outlives its turn — every pending bridge of a cancelled turn is cancelled by the cancel fan-in"
    status: partial
    reason: "The deny-by-default half holds everywhere (test-locked). The no-dangling invariant now holds on TWO of THREE cancel paths: the close path (CR-01 fixed, two wire-locked tests) and the ACP session/cancel path (slot-gated, test-locked). The third path — the SDK displaced-prompt arm (front.go:355-371, the fan-in goroutine's sdkCtx/mine arm) — fires machine.Cancel only: no cancelPendingForTurn runs on it, release() fans nothing in, so a turn cancelled by the client's second-prompt displacement leaves its in-flight interaction/requestPermission bridge dangling to the 300s responder deadline (DefaultPermissionDeadline, responder.go:41-43), then the responder answers DENY for an already-cancelled turn while the desktop's ask loop stays live (05-REVIEW.md WR-06: 'the one remaining arm that violates the contract'). Unlike CR-01 the session stays open, so the wire stays coherent — but the letter of the 05-02 truth and prohibition is violated on a deterministic, client-triggerable flow (any ACP client that double-prompts while an ask is pending), and the committed test TestCancelFanInOnlyWhenSlotHeld + assertDanglingAskNote (cancel_test.go:382-503) locks the dangling artifact in as EXPECTED behavior. Prior-pass precedent: the identical bounded-dangling shape (300s deadline then deny) failed this truth when the path was the close path; the displacement path earns the same verdict."
    artifacts:
      - path: "internal/acpfront/front.go"
        issue: "the fan-in goroutine's mine arm (:362-368) cancels the machine but not the pendings — the review's fix shape fans cancelPendingForTurn() in on the same arm (safe: the slot re-check guarantees the table belongs to this turn when mine is true)"
      - path: "internal/acpfront/cancel_test.go"
        issue: "TestCancelFanInOnlyWhenSlotHeld (:382-463) + assertDanglingAskNote (:491-503) enshrine the dangling ask as the expected wire artifact — the slot-free skip they lock stays correct, but the displaced turn's -32021 must replace the dangling note for the displacement-with-pending case"
    missing:
      - "Fan the pendings in on the mine arm (05-REVIEW.md WR-06 fix: st.Machine.Cancel() + f.perm.cancelPendingForTurn() under the same identity re-check)"
      - "Update the enshrining test: keep the slot-free-skip assertion; add a displacement-with-pending case asserting the -32021 reply for the displaced turn's bridge"
      - "Optionally batch WR-07's session-scoped pending table (cancelPendingForSession) — it kills the stale-sweep race deterministically on all three call sites (Cancel, CloseSession, the WR-06 arm) and makes the table-ownership assumption true at fire time"
behavior_unverified_items: []
advisory:
  - finding: "WR-05: resume's closing gate is TOCTOU — two reachable interleavings (close completing between resume's entry gate and re-check; Register landing in close's Get→mark gap) can still leak machine observers / register a zombie (05-REVIEW.md, delta finding; supersedes prior WR-03)"
    category: architectural
    reason: "Narrow race window on files untouched since the prior verification timestamp (git: 0 commits to lifecycle.go/front.go after 2026-11-21T10:05:00Z) — new-scope review finding with no failing test; recorded in the disposition ledger (open), feeds the hardening cycle"
    evidence_status: "none provided (review-derived interleaving analysis; not reproduced by a test)"
  - finding: "WR-07: Cancel's slot-gate snapshot can sweep a later turn's pendings after a deschedule (observe-before-cancel, front.go:436-449) — a silent wrong deny wire-indistinguishable from a client decision"
    category: architectural
    reason: "Requires a deschedule spanning a full turn unwind plus a next-turn ask registration — narrow race, no failing test, same class as the prior pass's WR-04 (warning, not scored); the disposition ledger carries it open with the session-scoped-table fix shape"
    evidence_status: "none provided (review-derived interleaving analysis; not reproduced by a test)"
  - finding: "IN-04..07 (review infos): single-shot probe before Await can miss a late-applying push; fake arms the latch after the reply write (golden flake class); realdesktop ask tap logs from conn goroutines; readbackdiag capture opener repeats the IN-02 born-OUTSIDE class"
    category: other
    reason: "Hardening backlog per the disposition ledger — none observed in the field-proven attempt-3 run"
    evidence_status: "none provided"
---

# Phase 5: Interactions & Full Chat Cycle (api-key) — Verification Report

**Phase Goal:** On a real z.ai account via api-key, a client gets a complete chat cycle — streamed tool calls, permission decisions answered from the client, working session lifecycle — with the host-managed interaction responder always on.
**Verified:** 2026-11-21T21:10:00Z
**Status:** gaps_found
**Re-verification:** Yes — third gap-closure round since the last gaps_found verdict (prior report stamped 2026-11-21T10:05:00Z; this pass re-verified everything first-hand on the current tree)

## What Changed Since the Last Verdict

Three gap-closure cycles landed (05-10, 05-11, 05-13/05-14) plus a fresh delta review (05-REVIEW.md rewritten, base af5a1ed..HEAD, 23 files):

1. **Gap 1 (SC4's real-account half) is CLOSED and FIELD-PROVEN.** The read-back verdict now latches on the real desktop's true surface — the first `session/create` reply's `settings.model.available`, offered through `probeCreateReadback` (auth.go:165-167, :193-229 → `watcher.OfferCreateReply`, readback.go:79-81), never the ack. REAL RUN attempt 3 (2026-11-21, human-run per the 05-14 blocking-human checkpoint) **PASSED 49.43s** on the real 3.18.2 account: authenticate read-back proven, tiny tool-less turn + file-read tool turn (TOOL-01 frames live), resume without replay, close with post-close prompt rejected, zero orphans. UAT item 1 = pass; the ledger row flipped from PENDING to actuals (first passing run); scrubbed capture EXP-APIKEY-CYCLE-03.jsonl committed (449/449 lines scrubbed, scan-secrets exit 0).
2. **Gap 2 (CR-01) is CLOSED with structural locks.** `drainActiveTurn` reports `sawTurn` — true on BOTH the successful-drain and deadline-elapsed arms, false only when no turn of the session was ever observed (lifecycle.go:221-260, read first-hand this pass) — and `CloseSession` fans the permission pendings in on exactly that signal (:170-175). Locked red-first by `TestCloseWithPendingPermissionCancelled` (exactly one reply, the -32021 cancelled error, stop-before-close order) and `TestCloseDeadlineArmFansPendingIn` (1ns seam). The delta review confirms: CR-01/WR-01/WR-02/WR-04 fixed; 0 critical.
3. **One must-have remains FAILED — and it is a NEW, named gap (WR-06).** The delta review surfaced that the SDK displaced-prompt arm of Prompt's cancel fan-in resolves the turn but never fans its pending permission bridge in — the bridge rides the 300s deadline then answers deny for a cancelled turn. Deterministic, client-triggerable, and **enshrined as expected behavior by committed tests**. Truth 7's invariant is observably false on that arm (details in the truths table and Gap Summary). Per the refresh directive, this is reported unsoftened: in my judgment the WR-06 arm violates the 05-02 truth/prohibition it sits under (the review itself binds it to "TOOL-02 on that arm"), so truth 7 fails again — this time on the displacement path, with the close and cancel paths now sound.
4. **The prior human-verification items are DISCHARGED.** 05-UAT.md records 2/2 pass (attempt 3 verdict + evidence recording with human confirmation); the guarded run's double guard re-verified this pass (no-tag → `[no test files]`; tag-without-env → honest SKIP 0.006s).

**Note on the prior report's stamp:** git shows the 05-10/05-11 fixes (e0b2c74..a154ff8, 07:39–08:41Z) predate the prior report's `verified:` 10:05:00Z timestamp — its "tree byte-identical from a4e0483" / "CR-01 confirmed first-hand this pass" narrative therefore describes the PRE-fix tree. This pass does not rely on that narrative: every truth below was re-verified against the current tree, with all owning packages re-run green under `-race` this pass (acpfront 26.2s, hostsvc 1.2s, interact 2.1s, zcode+fake+turn+core+acpport green, cmd 3.3s).

**Status rationale.** Step 9's decision tree puts gaps_found ahead of human_needed and passed when a must-have truth is FAILED. Truth 7's displacement-arm violation is exactly such a failure — observed and locked by the project's own test tier. Everything else the phase owns now verifies, including both prior gaps and the live full cycle. The remaining open warnings (WR-05 TOCTOU, WR-07 stale sweep — narrow races) and infos are recorded, consistent with the prior pass's treatment of race-window warnings; they do not independently fail a truth.

## Goal Achievement

### User Flow Coverage (MVP)

ROADMAP marks this phase `mode: mvp`; the goal is outcome-phrased and fails the `user-story.validate` guard (`valid: false`, re-run this pass) — the milestone's uniform precedent of deriving the flow table from the roadmap contract applies.

| # | Step | Expected | Evidence in Codebase | Status |
|---|------|----------|---------------------|--------|
| 1 | Client authenticates with a z.ai api-key (fake tier) | `authenticate{zai-api-key}` → `{}` ONLY after the registry read-back verdict; missing key → honest coded error naming `ZCODE_ACP_API_KEY` | TestAuthenticate arms green (acpfront -race 26.2s this pass); branch-(c) lock `testAuthProbeCreateSurface` green — acks but NO state.updated, success ONLY via the probe create's settings.model.available, probe pair order locked on the transcript (auth_test.go:274-308) | ✓ (fake tier) |
| 2 | Client authenticates on the REAL account | Read-back confirms on real desktop 3.18.2 and turns work | **FIELD-PROVEN**: attempt 3 PASS — authenticate read-back proven (branch-(c) surface); attempts 1-2 failures drove the fix; ledger row at actuals | ✓ (live) |
| 3 | Client opens a session | `session/new{cwd}` → create+subscribe+setModel | setmodel-followup golden replay green (zcode/fake 7.5s this pass); attempt 3: session/new + D-03 subscribe gate + setModel zai-api/GLM-5.3-Flash ok (live) | ✓ |
| 4 | Agent streams tool use | `tool_call`/`tool_call_update` with title, kind, status transitions, diffs | TestToolLifecycle + backpressure green under -race this pass; tool-lifecycle + full-cycle goldens green; attempt 3: file-read tool frames pending→in_progress→completed (live, TOOL-01) | ✓ |
| 5 | Backend asks permission; client decides | `session/request_permission` with backend options; outcome forwarded; cancel → -32021 | TestPermission* green under -race this pass; permission-allow/cancel goldens green; close-path fan-in now locked (CR-01 fix). Open arm: the SDK displaced-prompt path dangles the bridge (WR-06 — truth 7's gap) | ✓ (on its stated paths) / ✗ displacement arm |
| 6 | Client resumes a session | `session/resume` reattaches: re-issued subscribe, NO setModel, NO replay; cwd recorded | TestResume + TestResumeRecordsCwd green this pass (WR-02 fixed: cwd seam end-to-end); attempt 3: resume reattached, no replay (live). Open race caveat: WR-05 TOCTOU (narrow, recorded) | ✓ |
| 7 | Client closes a session | Ordered teardown; post-close prompt rejected; close-with-pending-permission fans the bridge in (-32021) | TestCloseTeardown + TestCloseWhileTurnActiveRace + **TestCloseWithPendingPermissionCancelled** + **TestCloseDeadlineArmFansPendingIn** green under -race this pass (CR-01 fixed); attempt 3: close clean, post-close prompt rejected, zero orphans (live) | ✓ |
| 8 | Full cycle composed (fake tier) | authenticate → new → prompt(tool+permission) → end_turn → resume → close, golden-locked | TestFullCycle green under -race this pass; full-cycle + lingering-latch-32010 goldens replay green (the new fake latch models the desktop's one-turn-at-a-time refusal honestly) | ✓ |
| 9 | Full cycle live on the REAL api-key account | Same cycle on the real install; scrubbed capture + ledger actuals | **FIELD-PROVEN**: attempt 3 PASS 49.43s — full cycle through the drained test; EXP-APIKEY-CYCLE-03.jsonl committed scrubbed; ledger at actuals; the desktop's true completion surfaces (turn-completed / prompt_completed ~16.5s after our end_turn) recorded as future-work data, cycle proven regardless | ✓ (live) |

### Observable Truths

Same 13-truth set as the prior pass (ROADMAP SC1-SC5 contract + nine PLAN frontmatter clusters). Score: **12/13** — both prior FAILED truths and the prior behavior-unverified truth now verify; truth 7 fails on the newly-surfaced WR-06 displacement arm.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: every server→host `interaction/*` answered promptly — dedupe, per-method deadlines, unknown → -32601, never a crash | ✓ VERIFIED | interact pkg green -race this pass (2.05s); reannounce-dedupe + unknown-host-method + user-input-decline goldens replay green (zcode/fake 7.5s); production wiring `conn.OnRequest = responder.Serve` before traffic (main.go:252, re-grepped this pass); WR-01's join-silence now structural (conn replyRequest drops nil-nil answers) |
| 2 | SC2: tool use streams as tool_call/tool_call_update with title, kind, status transitions, file diffs | ✓ VERIFIED | acpfront green -race this pass (26.2s incl. TestToolLifecycle, TestStreamBackpressure, TestStreamBeforeStopReason); goldens green; TOOL-01 frames observed live in attempt 3 |
| 3 | SC3: backend permission ask reaches the client with allow/reject options; chosen outcome forwarded; ACP-side cancel resolves pending as cancelled | ✓ VERIFIED | TestPermissionBridge/Cancel/CancelRace/Expiry green under -race this pass; permission-allow byte-verbatim + permission-cancel -32021 goldens green. Scope note: SC3's stated paths (bridge + ACP-cancel) hold; the SDK-displacement dangling is truth 7's failure, not this truth's claim |
| 4 | SC4: `authenticate{zai-api-key}` on a REAL account enables working turns, success verified by registry read-back, never the ack | ✓ VERIFIED (field-proven) | Machinery: probe-create arm read first-hand (auth.go:165-167, :193-229) — verdict remains `watcher.Await`; the probe is evidence, never the verdict; write-once latch (readback.go:146-148); ack-mismatch negative still green (hostsvc 1.2s -race). Field: attempt 3 PASS on the real account — read-back proven, turns ran (UAT item 1 pass, ledger actuals, EXP-APIKEY-CYCLE-03 committed scrubbed, scan-secrets 0) |
| 5 | SC5: resume reattaches without replay, close cleans up; the full cycle runs end-to-end on api-key | ✓ VERIFIED (field-proven live half) | Fake tier: TestFullCycle/TestResume/TestCloseTeardown green under -race; session-lifecycle golden green. Live half: attempt 3 — resume same session no replay, close clean + post-close prompt rejected, staged shutdown zero orphans (kill(-pgid,0) ESRCH). The prior PRESENT_BEHAVIOR_UNVERIFIED flag is discharged by the recorded field run |
| 6 | 05-01: per-method deadlines (15 s runtime-prefs, 300 s permission → deny, runtime-headers coded error until key); user-input → {action:decline}; bypass lane; RouterEvent expiry records | ✓ VERIFIED | Responder policy table unchanged; interact green -race this pass; user-input-decline + runtime-preferences goldens green; runtime-headers answered from the wired key source live in attempt 3 |
| 7 | 05-02: deny-by-default (unknown/missing optionId, malformed outcome, expiry → deny or -32021, NEVER allow); no bridge outlives its turn (cancel fan-in) | ✗ FAILED (partial) | Deny-by-default half verified (denyAnswer/cancelledWireError paths, TestPermissionExpiry/CancelRace green this pass). No-dangling invariant now holds on the close path (CR-01 fixed: sawTurn + two wire-locked tests, first-hand read + green -race) and the ACP-cancel path (slot-gated, TestCancelFanInOnlyWhenSlotHeld locks the gate) — but is VIOLATED on the SDK displaced-prompt arm (front.go:362-368, machine.Cancel without the pendings fan-in): the bridge dangles to the 300s deadline then answers deny for a cancelled turn; the enshrining tests assert the dangling artifact as expected (WR-06, 05-REVIEW.md :85-99) |
| 8 | 05-03/05-05: router fan-out (WR-01), observer removes (WR-07), registry Remove consumer, WR-04 error-path orphan close, close-vs-prompt race gate (T-5-12) | ✓ VERIFIED | machine.go fan-out + CloseObservers consumed at lifecycle.go:88-89,118-120,159,198; registry Remove :197; closing gate gates Prompt (front.go:334) and acquireTurn; all packages green this pass; newsession-orphan-close golden green. The claimed close-vs-PROMPT gate holds; the close-vs-RESUME TOCTOU is WR-05 (advisory, recorded — not scored here) |
| 9 | 05-04 + a4e0483: WR-03 pump (non-blocking enqueue, keep-latest coalescing, byte ceiling fail-loud, flush-before-stopReason; a4e0483 fix sites locked) | ✓ VERIFIED | All locked tests green in the acpfront -race run this pass; WR-04's pump-identity re-check sound per delta review (release nils the slot under the same lock) |
| 10 | 05-06: strict snapshot envelope with EXACT-revision formula REUSED from supervisor; read-back never the ack; runtime-headers success shape only from the injected key | ✓ VERIFIED | hostsvc green -race this pass incl. the ack-mismatch negative; snapshot reuse intact (snapshot.go:121 readBuiltinRevision / supervisor.BuiltinRevision formula); the machinery's honesty held through all three field attempts (errord loud, never trusted the ack); the 05-11 correction made the surface field-true |
| 11 | 05-07: honest authenticate chain (record→push→refresh→read-back→{}), no-key coded error naming the env var, interruptible clean retry, CredentialDecrypt seam, hygiene locks, binary wires the stack before traffic | ✓ VERIFIED | acpfront + interact + cmd green under -race this pass; wiring re-grepped (main.go:246-252, :294); hygiene held IN THE FIELD across three runs (every failure errored loud; cipher asks answered from the keyring live, value_len=815); scan-secrets exit 0 on the tree incl. all three scrubbed captures (re-run this pass) |
| 12 | 05-08: fake-tier FULL cycle — one continuous composition, golden-locked, nothing hangs/drops/fabricates | ✓ VERIFIED | TestFullCycle green under -race this pass; full-cycle golden green; the 05-13 fake latch addition is test-double-side only (zero production-contract change — git: no commits to acpfront/hostsvc/interact/acpport/turn/cmd since the prior stamp) and its lingering-latch-32010 row replays green |
| 13 | 05-09: guarded real-tier test (double guard; skips honestly; default suite untouched); quota ledger; no credential material in repo | ✓ VERIFIED | Guards re-run this pass: no-tag → `[no test files]` exit 0; tag-without-env → honest SKIP 0.006s. Ledger row at ACTUALS on the passing run (experiments/README.md:195); EXP-APIKEY-CYCLE-03.jsonl committed (449 lines, scrub markers present, zero credential shapes — my independent pattern scan hit only the public builtin-revision hash and databaseId hex, and scan-secrets exits 0); captureID discipline preserved (-01/-02 byte-intact) |

**Score:** 12/13 truths verified (0 present-behavior-unverified — both field truths now carry recorded live evidence; 1 FAILED partial — truth 7's displacement arm)

**Prohibitions (21 across all plans):** 20 hold — re-confirmed where re-runnable this pass (deny paths test-locked; goldens byte-lock traceability; auth success only on read-back — held through three field runs; double guard re-run; scan-secrets 0 incl. three captures; zero debt markers re-grepped across the phase's files). **1 FLAGGED:** 05-02's "MUST NOT let a permission bridge outlive its turn" is violated on the SDK displaced-prompt arm (WR-06) — and unlike a merely-unverified prohibition, this one is OBSERVED-violated by the project's own committed test, which locks the dangling artifact as expected. `unverified_prohibitions: 1`.

### Required Artifacts

All 27 previously-verified artifact paths carry over (the 05-10/05-11 fixes touched lifecycle.go, front.go, auth.go, readback.go, conn.go, responder.go, agent.go + their tests — all re-verified this pass, all owning packages green under -race). New/changed artifacts:

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| internal/acpfront/lifecycle.go | sawTurn drain semantics + close fan-in | ✓ VERIFIED (fix) | CR-01 fixed: sawTurn true on both arms, false only on idle close (:221-260); fan-in fires on the signal (:170-175); second drain re-check unchanged; order intact. First-hand read this pass; green under -race |
| internal/acpfront/lifecycle_test.go | close-with-pending-permission locks | ✓ VERIFIED | TestCloseWithPendingPermissionCancelled (:584) — exactly ONE reply on the pending reverse, the -32021 cancelled error, stop-before-close order; TestCloseDeadlineArmFansPendingIn (:686) — the deadline arm fans in via the WithCloseTurnWait 1ns seam. Both green under -race |
| internal/hostsvc/readback.go | field-verified read-back surface | ✓ VERIFIED (fix) | OfferCreateReply (:79-81) + registryModels digs settings.model.available[].ref (:216-253) alongside patch.providers; write-once latch; notConfirmed's coded negative names the field-verified surface and EXP-READBACK-DIAG-01 (:173-180). First-hand read this pass |
| internal/acpfront/auth.go | probe-create read-back arm | ✓ VERIFIED (fix) | WatchRegistry BEFORE traffic (:142), Applied() gates probe necessity only (:165-167), OfferCreateReply BEFORE the id dig (:214-216), best-effort probe close, every failure structurally logged and skips its step; verdict stays Await (:173). First-hand read this pass |
| internal/acpfront/auth_test.go | branch-(c) corrected-surface lock | ✓ VERIFIED | testAuthProbeCreateSurface (:274-308): fake acks but emits NO state.updated (the field-verified real-desktop shape) — authenticate succeeds ONLY via the probe create's settings.model.available; exactly-one probe pair, refresh→create→close order locked on the transcript. Red-first proven (f308edf RED → a154ff8 GREEN) |
| internal/acpfront/front.go | WR-04 fan-in identity re-check + Cancel slot gate | ✓ VERIFIED (with WR-06 open) | The re-check (:362-368) cannot cancel a later turn (TestCancelFanInStaleArmSkips locks containment); Cancel's slot gate (:439-447) locks the slot-free skip. OPEN: the mine arm still omits the pendings fan-in (WR-06 — gap 3) |
| internal/acpfront/cancel_test.go | slot-gate + (problematically) the dangling artifact | ✗ ENSHRINES GAP 3 | TestCancelFanInOnlyWhenSlotHeld (:382-463) locks the correct slot-free skip AND asserts assertDanglingAskNote (:491-503) — the displaced turn's ask riding to its deadline unanswered — as the expected wire artifact |
| internal/zcode/fake/fake.go + scenario.go | fake engine turn latch (05-13) | ✓ VERIFIED | Test-double-side only; latch schema validated fail-loud; lingering-latch-32010 corpus row + golden replay green (zcode/fake 7.5s this pass); zero production-contract change (git-verified) |
| test/realdesktop/apicycle_test.go | guarded cycle + latch drain (05-13) | ✓ VERIFIED | Double guard intact (:183-189); ask-tap settle + bounded invalid-request-only retry (8 attempts/120s, errors.As on code, never message text); exhaustion returns the loud hypothesis-(b) sentinel; stage assertions mirror the attempt-3 PASS exactly (:241-268) |
| experiments/captures/EXP-APIKEY-CYCLE-03.jsonl | scrubbed attempt-3 capture | ✓ VERIFIED | 449/449 lines; 427 scrub markers (<uuid-N>, [REDACTED-CREDENTIAL], <path>); header carries builtinRevision + capture id; zero credential shapes (independent pattern scan hit only the public revision hash/databaseId); scan-secrets exit 0 |
| experiments/README.md | quota ledger at actuals | ✓ VERIFIED | Row :195 flipped PENDING → PASSED with attempt-3 actuals (D-05 minimum spend) + verdict line; attempts 1-2 records intact beneath |
| 05-REVIEW.md + 05-REVIEW-DISPOSITION.md | rewritten delta review + ledger | ✓ VERIFIED | Delta (af5a1ed..HEAD, 23 files): 0C/3W/4I; CR-01/WR-01/02/04 fixed with locks (first-hand-verified per the review's own -race/golden/secret runs + my re-runs); WR-03 superseded by WR-05; WR-06/WR-07 + IN-04..07 open — both re-read this pass and spot-verified against the code |

### Key Link Verification

The filename-literal matcher limitation on Go sources stands (prior pass diagnosis). The load-bearing links were re-grepped/re-read first-hand this pass; the tree's previously-verified links carry over for files untouched since their verifying pass (git: zero commits to acpfront/hostsvc/interact/acpport/turn/cmd core files after the prior stamp — only fake.go/apicycle_test.go/testdata changed, in the 05-13 test-tier latch work):

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| cmd/zcode-acp/main.go | internal/interact (forwarder/keys/cipher) | constructor seams | WIRED | main.go:246-248 WithPermissionForwarder/WithRuntimeHeadersKeySource/WithCredentialDecrypt; :294 NewStaticKey; :252 conn.OnRequest = responder.Serve (re-grepped this pass) |
| internal/acpfront/auth.go | internal/hostsvc/readback.go | probe-create surface | WIRED | WatchRegistry :142 → Applied() gate :165 → OfferCreateReply :216 → Await verdict :173 (read first-hand) |
| internal/acpfront/lifecycle.go | internal/acpfront/permission.go | close fan-in | WIRED (fixed) | lifecycle.go:170-174 — gated on sawTurn (the CR-01 fix); both lock tests green |
| internal/acpfront/front.go | internal/acpfront/permission.go | cancel fan-in | WIRED, one arm INCOMPLETE | Front.Cancel :445-447 wired + slot-gated; the Prompt mine arm :366-368 cancels the machine only (WR-06 — gap 3) |
| all other declared links | — | — | WIRED (carried) | Owning packages re-ran green under -race this pass |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| authenticate verdict | registry surfaces | state.updated patches OR probe create-reply's settings.model.available | field-proven on the real desktop (attempt 3) | ✓ FLOWING (both tiers) |
| permission bridge | client outcome | ACP client selection | forwarded byte-verbatim (golden); live cipher/allow decisions in attempt 3 | ✓ FLOWING |
| displaced turn's bridge | client outcome | none — the mine arm never fans in | dangles to the 300s deadline, deny framed for a cancelled turn | ✗ DISCONNECTED on that arm (gap 3) |
| tool stream | tool frames | backend tool lifecycle events | golden-locked; live frames observed (call_3692… completed) | ✓ FLOWING |
| full-cycle frames | whole-cycle wire | scripted corpus row (fake) / real desktop (live) | golden-locked + field-proven | ✓ FLOWING |
| runtime-headers reply | apiKey material | injected KeySource | success shape only with key; live-answered in attempt 3 | ✓ FLOWING |
| resumed-session diffs | cwd resolution | `cwds` map | WR-02 fixed: resume records the client cwd (TestResumeRecordsCwd green; zero-wire-frame parity arm) | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CORE-06 + 05-02 bridge machinery (SC1/SC3 fake tier) | `go test ./internal/interact/ -race -count=1` | ok 2.053s | ✓ PASS |
| AUTH-01 machinery incl. probe-create surface | `go test ./internal/hostsvc/ -race -count=1` | ok 1.215s | ✓ PASS |
| Front machinery: tool/permission/auth/lifecycle/full-cycle + CR-01 locks + WR-04/06 tests (SC2-SC5 fake tier) | `go test ./internal/acpfront/ -race -count=1` | ok 26.150s | ✓ PASS |
| Corpus rows + goldens through real Conn (incl. lingering-latch-32010) | `go test ./internal/zcode/... ./internal/turn/ ./internal/core/ ./internal/acpport/ -count=1` | all ok (fake 7.537s) | ✓ PASS |
| Binary E2E incl. wired responder | `go test ./cmd/zcode-acp/ -race -count=1` | ok 3.300s | ✓ PASS |
| Guarded real-tier test: default build exclusion | `go test ./test/realdesktop/ -count=1` | `[no test files]` exit 0 | ✓ PASS |
| Guarded real-tier test: tag without env | `go test -tags realdesktop ./test/realdesktop/ -count=1 -run TestAPIKeyFullCycle` | ok 0.006s — honest skip | ✓ PASS (honest skip) |
| Credential hygiene on the current tree (three captures) | `bash experiments/scan-secrets.sh` | exit 0 | ✓ PASS |
| CR-01 fix first-hand verification | read lifecycle.go:140-260 + front.go Cancel | sawTurn semantics correct on all arms; fan-in fires on both | ✓ PASS |
| CR-01 lock tests exist and exercise the invariant | enumerate + green -race run | TestCloseWithPendingPermissionCancelled + TestCloseDeadlineArmFansPendingIn present, green | ✓ PASS |
| WR-06 first-hand verification | read front.go:355-371 + cancel_test.go:382-503 | mine arm cancels machine only; committed test asserts the dangling ask as expected | ✗ CONFIRMED DEFECT (gap 3) |
| Read-back surface lock red-first provenance | git f308edf (RED) → a154ff8 (GREEN) | red-first chain intact | ✓ PASS |
| Live run on the real account | guarded command, human-run per the 05-14 checkpoint | attempt 3 PASS 49.43s — recorded in 05-UAT.md + ledger + scrubbed capture (NOT re-run this pass; the guarded run is contractually manual-only) | ✓ PASS (recorded field evidence, discharged human item) |

### Probe Execution

No probe scripts declared or conventional (`scripts/*/tests/probe-*.sh` absent). Step 7c: not applicable.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| AUTH-01 | 05-06, 05-07, 05-09 (+05-10/05-11 gap plans) | api-key path: updateAccountConfig + requestProviderRuntimeHeaders answered correctly (registry state verified, not the ack) | ✓ SATISFIED (field-proven) | Machinery + corrected surface + REAL RUN attempt 3 PASS: authenticate read-back proven, turns ran, runtime-headers answered from the wired key source. REQUIREMENTS.md keeps AUTH-01 `[ ]` and Phase-5 traceability at "Gaps Found" — restoring those is the ORCHESTRATOR's post-verification act on this verdict (not this report's edit). Honest caveat: with this verdict gaps_found on truth 7, the requirement's own stated behavior is met; the traceability flip decision rests with the orchestrator |
| CORE-06 | 05-01, 05-02, 05-08, 05-09 | Host-managed interaction responder always on | ✓ SATISFIED | responder green -race + goldens + wired before traffic; held live through attempt 3 (cipher asks answered from the keyring, runtime-headers answered) |
| SESS-01 | 05-03, 05-05, 05-08, 05-09 | resume (no replay) and close (child-side cleanup) | ✓ SATISFIED | Both live-proven in attempt 3 + test-locked; WR-02 (resume cwd) fixed and locked; WR-05 TOCTOU narrow race recorded (advisory) |
| TOOL-01 | 05-04, 05-08, 05-09 | tool_call/tool_call_update streamed | ✓ SATISFIED | Stream tap + pump locked green; tool frames observed live in attempt 3 (pending→in_progress→completed) |
| TOOL-02 | 05-02, 05-08, 05-09 | permission bridged; cancelled outcome on ACP-side cancel | ✓ SATISFIED (stated behavior) — WR-06 caveat recorded | The requirement's stated behavior holds and is test-locked (bridge both directions; ACP-side cancel → -32021, now including the close path). The delta review binds WR-06's displacement arm to "TOOL-02 on that arm" — that arm's violation is carried by truth 7 / the 05-02 prohibition (same routing as the prior pass used for CR-01), not by the requirement's stated cancel path |

Orphaned requirements: none — REQUIREMENTS.md maps exactly AUTH-01, CORE-06, SESS-01, TOOL-01, TOOL-02 to Phase 5, and every ID appears in plan frontmatter (re-extracted from all 14 plans this pass: AUTH-01 ×7, CORE-06 ×7, SESS-01 ×7, TOOL-01 ×5, TOOL-02 ×5; no divergence).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/acpfront/front.go | 355-371 (mine arm :362-368) | WR-06: displaced-prompt fan-in cancels the machine but never fans the turn's pendings in — bridge dangles to the 300s deadline, deny answered for a cancelled turn; enshrined by cancel_test.go:382-503 | 🛑 Blocker (truth-7 violation, gap 3) | Deterministic on any client double-prompt displacement: the client never answers, the desktop's ask loop lives up to 300s, and a deny is framed for a turn that is already gone — the exact dangling shape the 05-02 prohibition forbids |
| internal/acpfront/cancel_test.go | 439-462, 491-503 | Tests lock the dangling artifact as the EXPECTED wire shape ("the ask was never cancelled by anything in this flow") | 🛑 Blocker (part of gap 3) | A fixer preserving the enshrined artifact would keep the defect; the test must be updated with the fix |
| internal/acpfront/lifecycle.go | 55-128 (resume) | WR-05: resume's closing gate TOCTOU — close completing between the entry gate and re-check, or Register landing in close's Get→mark gap, still leaks observers/zombie (supersedes WR-03) | ⚠ Warning (advisory) | Narrow race; unreachable in the field-proven run; disposition open |
| internal/acpfront/front.go | 436-449 | WR-07: Cancel's observe-before-cancel sweep unguarded across deschedule — a later turn's fresh ask can be swept (-32021) in a deschedule window | ⚠ Warning (advisory) | Narrow race; silent wrong deny; session-scoped pending table is the deterministic fix shape |
| test/realdesktop/apicycle_test.go | IN-01/IN-02 sites | Malformed ZCODE_ACP_MODEL silently defaulted; raw-capture born-OUTSIDE rule comment-only; IN-06 ask-tap logging goroutines; IN-07 readbackdiag opener | ℹ️ Info | Hardening backlog per disposition ledger |
| cmd/zcode-acp/main.go | 307-314 | IN-03 shutdown select can mask a child crash with exit 0 (prior info, unchanged); IN-04/IN-05 review infos | ℹ️ Info | Nondeterministic exit verdict / golden flake class |
| — | — | Debt markers (TBD/FIXME/XXX) | none | Zero matches re-grepped this pass across acpfront, hostsvc, interact, fake.go/scenario.go, realdesktop, cmd |

### Advisory (New Scope, Unevidenced)

New-scope findings from Step 7 / the delta review with no deterministic evidence — reported, not blocking (evidence gate, re-verification mode).

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | WR-05 resume closing-gate TOCTOU | architectural | new-scope review interleaving analysis; files untouched since the prior verify-ts; no failing test; disposition open |
| 2 | WR-07 Cancel stale-sweep race | architectural | new-scope review interleaving analysis; requires a deschedule spanning a full turn unwind; no failing test; disposition open (fix shape recorded) |
| 3 | IN-04..07 | other | hardening backlog per the disposition ledger; none observed in the field-proven run |

(Truth-level failures — WR-06/gap 3 — are NOT advisory: they enter through Step 3 truth verification against the carried must-have, which the evidence gate does not demote.)

### Human Verification Required

None. The prior report's two human items are DISCHARGED with recorded evidence:

1. ~~REAL RUN retry (UAT item 1)~~ — **PASS recorded**: attempt 3, 2026-11-21, human-run per the 05-14 blocking-human checkpoint, PASS 49.43s on the real 3.18.2 account (authenticate read-back proven; tool-less turn + file-read tool turn; resume no-replay; close clean; zero orphans). Evidence: 05-UAT.md item 1, experiments/README.md ledger actuals, EXP-APIKEY-CYCLE-03.jsonl (scrubbed, committed).
2. ~~Evidence recording (UAT item 2)~~ — **PASS recorded**: scrub-capture pipeline run, scan-secrets 0 (tree + staged diff), ledger row flipped to actuals, human-confirmed "evidence recorded" (05-UAT.md item 2).

The remaining work (gap 3) is code work — the WR-06 fix + test update — which is for the closure plan, not a human checkpoint.

### Deferred Items

None. Checked phases 6-7 of the milestone: Phase 6 is the OAuth path (AUTH-02/03, MODEL-01/02), Phase 7 is CI conformance/VM/docs — neither covers the displaced-prompt fan-in fix (the 05-02 truth is Phase 5's own contract). The machine-side quiet-floor-vs-patch improvement (the attempt-3 observation that the desktop's true completion surfaces arrive ~16.5s after our end_turn) is future work by the project's own record and was never a must-have of this phase — the cycle is proven with the drain in place.

### Gaps Summary

Both gaps from the last verdict are genuinely closed, and the closes are the strongest kind: code read first-hand, red-first test locks, and a passing field run on the real account with the full evidence trail. The live full cycle — the phase goal's opening clause "On a real z.ai account via api-key" — now observably works: authenticate proven by the registry read-back (never the ack), streamed tool calls live, resume without replay, clean close, zero orphans, 49.43s, ledgered spend.

**Gap 3 (the one remaining gap) — the SDK displaced-prompt arm of the cancel fan-in (WR-06).** When a client's second prompt displaces an active one while a permission ask is pending, the fan-in goroutine's mine arm fires `machine.Cancel()` only: no `cancelPendingForTurn()` runs on that arm, and `release()` fans nothing in either. The displaced turn's pending bridge therefore rides the responder's 300s permission deadline and is answered **deny** for a turn that was already cancelled — while the desktop's ask loop stays live. This is not a narrow race: it is deterministic, reachable by ordinary client behavior, and — the part that makes it a truth failure rather than a hardening note — the committed test tier asserts the dangling ask as the EXPECTED wire artifact. Truth 7's invariant ("no permission bridge outlives its turn — every pending bridge of a cancelled turn is cancelled by the fan-in") is observably false on this arm, exactly as it was on the close path before CR-01 was fixed. The delta review says it plainly: "the one remaining arm that violates the contract."

The fix is small and the review provides the shape: fan `cancelPendingForTurn()` in on the mine arm under the same identity re-check (safe — the re-check guarantees the table belongs to this turn when mine is true), and update the enshrining test: keep the correct slot-free-skip assertion, replace the displaced turn's dangling-note expectation with a -32021 assertion. WR-07's session-scoped pending table batches cleanly with it and would make the table-ownership assumption true at fire time on all three call sites.

Everything else holds: 12/13 truths verified, 20/21 prohibitions, all guards and hygiene green, both field truths carrying recorded live evidence. The phase advances to one focused closure plan (citing 05-REVIEW-DISPOSITION.md, WR-06/WR-07), then re-verification.

---

_Verified: 2026-11-21T21:10:00Z_
_Verifier: Claude (gsd-verifier)_
