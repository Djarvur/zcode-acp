---
phase: 05-interactions-full-chat-cycle-api-key
verified: 2026-11-21T21:38:53Z
status: passed
score: 13/13 must-haves verified
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
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-10-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-10-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-11-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-11-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-12-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-12-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-13-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-13-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-14-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-14-SUMMARY.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-15-PLAN.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-15-SUMMARY.md"
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
covered_digest: "v4:sha256:eb0664d483cc2c3611a84860d851a45785e8b792c3c41562a38b4c69598f0a63"
behavior_unverified: 0
overrides_applied: 0
unverified_prohibitions: 0
re_verification:
  previous_status: gaps_found
  previous_score: 12/13
  gaps_closed:
    - "Gap 3 (WR-06) — the SDK displaced-prompt fan-in arm now fans the displaced turn's pending permission bridge in at the displacement boundary: exactly ONE -32021 cancelled reply on the back leg, no unanswered-at note, exactly-once cancelled success preserved. First-hand verified this pass: watchCancelFanIn's mine arm takes the pendings under the SAME turnMu section as the WR-04 pump-identity re-check (front.go:466-489) and fires them outside any lock; locked by the INVERTED enshrining tests (assertDanglingAskNote deleted; TestCancelFanInOnlyWhenSlotHeld rewritten with the legitimacy framing in the doc comment) — green under -race (acpfront ok 26.772s this pass); RED-first proven via gsd_run check tdd-red-evidence RED_EVIDENCE_OK on evidence/05-15-red-t1 (0 reply envelopes + unanswered note present at HEAD — the genuine missing-fan-in signal)"
    - "WR-07 batched and closed in the same cycle — the pending table is SESSION-SCOPED (map[sessionID]map[requestID]cancel, permission.go:38) with atomic takes on all three cancel arms (mine arm front.go:476-481; Front.Cancel observe+take in ONE turnMu section front.go:429-435; close path via cancelPendingForSession lifecycle.go:177 under the pre-drain closing gate); cancelPendingForTurn DELETED (promote-not-add-alongside — grep confirms no definition remains); foreign-session asks survive Front.Cancel (TestCancelFanInSkipAndScope/session scope, RED_EVIDENCE_OK on evidence/05-15-red-t2); the slot-free skip lock re-homed with full discriminating power (TestCancelFanInSkipAndScope/slot free)"
  gaps_remaining: []
  regressions: []
---

# Phase 5: Interactions & Full Chat Cycle (api-key) — Verification Report

**Phase Goal:** On a real z.ai account via api-key, a client gets a complete chat cycle — streamed tool calls, permission decisions answered from the client, working session lifecycle — with the host-managed interaction responder always on.
**Verified:** 2026-11-21T21:38:53Z
**Status:** passed
**Re-verification:** Yes — FINAL pass after the FOURTH gap-closure cycle (05-15; prior report stamped 2026-11-21T21:10:00Z with exactly one gap: WR-06 / truth 7)

## What Changed Since the Last Verdict

One gap-closure cycle landed (05-15, commits a7f67f9 RED → 7f18358 GREEN → b358cfa RED → 8add9e6 GREEN, all verified ancestors of HEAD) plus its docs commits. The production delta since the 05-14-SUMMARY baseline (924f4e7) is EXACTLY the four declared files (scope gate re-run this pass: `scope-clean`):

1. **WR-06 (the last gap) is CLOSED, first-hand verified on all three levels.** The fan-in goroutine's mine arm (`watchCancelFanIn`, front.go:466-489) takes the displaced turn's session pendings under the SAME turnMu section as the WR-04 pump-identity re-check (:476-481 — check-and-take atomic by construction) and fires `machine.Cancel()` + the taken cancels outside any lock (:483-486). The enshrining tests are INVERTED, not weakened: `TestCancelFanInOnlyWhenSlotHeld` (cancel_test.go:425-488) now demands exactly ONE -32021 reply envelope for the displaced ask and NO unanswered-at note, with the rewrite's legitimacy stated in the doc comment (:397-418); `assertDanglingAskNote` is deleted (no definition remains); the dropped tail's slot-free lock is RE-HOMED with full discriminating power in `TestCancelFanInSkipAndScope`/slot free (:810-876 — a pending genuinely parked while the slot is free).
2. **WR-07 closed deterministically in the same cycle.** The pending table is session-scoped (`pending map[string]map[string]context.CancelFunc`, permission.go:38); `takePendingCancels(sessionID)` (:159-177) snapshot-and-deletes ONLY the named session's bucket under the bridge lock; `cancelPendingForTurn` is DELETED — `cancelPendingForSession` (:198-200) and `fireTakenPendings` (:183-191, the discriminator log verbatim with its non-empty gate) are the promoted surface; `Front.Cancel` folds the slot observation and the take into ONE turnMu section (:429-435), closing the deschedule window by construction; the close path fans in via `cancelPendingForSession(sessionID)` under the sawTurn gate (lifecycle.go:170-178, the closing gate predating the drain — no atomic take needed, reasoning documented in-code); the one-way turnMu → bridge.mu ordering is documented at the bridge (permission.go:26-31, T-5-38).
3. **RED-first provenance validated independently this pass:** both RED evidence records re-validated through `gsd_run check tdd-red-evidence` → `RED_EVIDENCE_OK` (reason `target_test_failed`) × 2. Task 1's RED is the genuine missing-fan-in signal (0 reply envelopes for the displaced ask within the bound + the honest unanswered-at note present at HEAD); Task 2's RED is the genuine scope-crossing signal (the FOREIGN session's round-trip read cancelled under the front-wide table). The one auto-fixed deviation is faithful: the no--32800 scan was scoped to the agent→client direction (`assertNoAgentRequestCancelled`) because the client-side -32800 answering its own cancelled permission round-trip is the honest mirror of the -32021 arm — the agent's prompt-answer shape stays fully locked (assertSingleCancelledResponse + single send).
4. **Everything else stands.** Both field truths carry their recorded live evidence (attempt 3 PASS 49.43s, read-back proven, ledger actuals, EXP-APIKEY-CYCLE-03.jsonl); all 12 previously-verified truths re-verified at regression level this pass (every owning package re-run green, wiring re-grepped, corpus replayed, hygiene gates re-run).

**Status rationale.** Step 9's decision tree: no truth FAILED, no artifact MISSING/STUB, no key link NOT_WIRED, no blocker anti-pattern, and no human verification items remain (the prior report's two items are discharged with recorded evidence; the WR-06 closure is unit-verified and needs no live spend). Status `passed`, 13/13. The prior `unverified_prohibitions: 1` flag is resolved: the 05-02 prohibition now holds on ALL THREE cancel arms, test-locked.

## Goal Achievement

### User Flow Coverage (MVP)

ROADMAP marks this phase `mode: mvp`; the goal is outcome-phrased and fails the `user-story.validate` guard (prior pass finding, unchanged) — the milestone's uniform precedent of deriving the flow table from the roadmap contract applies.

| # | Step | Expected | Evidence in Codebase | Status |
|---|------|----------|---------------------|--------|
| 1 | Client authenticates with a z.ai api-key (fake tier) | `authenticate{zai-api-key}` → `{}` ONLY after the registry read-back verdict; missing key → honest coded error naming `ZCODE_ACP_API_KEY` | TestAuthenticate arms green (acpfront -race 26.77s this pass); branch-(c) lock `testAuthProbeCreateSurface` green — success ONLY via the probe create's settings.model.available | ✓ (fake tier) |
| 2 | Client authenticates on the REAL account | Read-back confirms on real desktop 3.18.2 and turns work | **FIELD-PROVEN** (unchanged from prior pass): attempt 3 PASS — authenticate read-back proven (branch-(c) surface); ledger row at actuals (re-read this pass: "PASSED 2026-11-21 (attempt 3)") | ✓ (live) |
| 3 | Client opens a session | `session/new{cwd}` → create+subscribe+setModel | setmodel-followup golden replay green (corpus ok 7.322s this pass); attempt 3 live record | ✓ |
| 4 | Agent streams tool use | `tool_call`/`tool_call_update` with title, kind, status transitions, diffs | TestToolLifecycle + stream/backpressure green under -race this pass; tool-lifecycle + full-cycle goldens green; TOOL-01 frames observed live (attempt 3) | ✓ |
| 5 | Backend asks permission; client decides | `session/request_permission` with backend options; outcome forwarded; cancel → -32021 | TestPermission* green under -race this pass; permission-allow/cancel goldens green. **All three cancel paths now sound**: close (CR-01 locks green untouched), ACP cancel (TestPermissionCancel), SDK displacement (the WR-06 fix — the rewritten lock green under -race this pass) | ✓ (on ALL paths — the prior displacement-arm caveat is resolved) |
| 6 | Client resumes a session | `session/resume` reattaches: re-issued subscribe, NO setModel, NO replay; cwd recorded | TestResume + TestResumeRecordsCwd green this pass; attempt 3 live record. WR-05 TOCTOU caveat unchanged (narrow race, advisory, recorded — the resume code section is untouched by cycle 4, git-diff verified) | ✓ |
| 7 | Client closes a session | Ordered teardown; post-close prompt rejected; close-with-pending-permission fans the bridge in (-32021) | TestCloseTeardown + TestCloseWhileTurnActiveRace + TestCloseWithPendingPermissionCancelled + TestCloseDeadlineArmFansPendingIn green under -race this pass (CR-01 locks untouched — grep + green); attempt 3 live record | ✓ |
| 8 | Full cycle composed (fake tier) | authenticate → new → prompt(tool+permission) → end_turn → resume → close, golden-locked | TestFullCycle green under -race this pass; corpus replay ok 7.322s incl. lingering-latch-32010 | ✓ |
| 9 | Full cycle live on the REAL api-key account | Same cycle on the real install; scrubbed capture + ledger actuals | **FIELD-PROVEN** (recorded, unchanged): attempt 3 PASS 49.43s — EXP-APIKEY-CYCLE-03.jsonl committed scrubbed (file present this pass); ledger at actuals; zero spend this pass by design (05-15's no-new-checkpoint decision) | ✓ (live) |

### Observable Truths

Same 13-truth set as the prior pass (ROADMAP SC1-SC5 contract + nine PLAN frontmatter clusters). Score: **13/13** — the one FAILED truth (7) now verifies with behavioral evidence.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: every server→host `interaction/*` answered promptly — dedupe, per-method deadlines, unknown → -32601, never a crash | ✓ VERIFIED (regression) | interact ok this pass (1.019s); reannounce-dedupe + goldens green (corpus ok 7.322s); production wiring re-grepped (main.go:252 `conn.OnRequest = responder.Serve`; :246-248, :294) |
| 2 | SC2: tool use streams as tool_call/tool_call_update with title, kind, status transitions, file diffs | ✓ VERIFIED (regression) | acpfront green under -race this pass (26.772s incl. tool/stream suites); goldens green |
| 3 | SC3: backend permission ask reaches the client with allow/reject options; chosen outcome forwarded; ACP-side cancel resolves pending as cancelled | ✓ VERIFIED | TestPermissionBridge/Cancel/CancelRace/Expiry green under -race this pass; permission-allow byte-verbatim + permission-cancel -32021 goldens green; the bridge's cancelled classification intact (permission.go:95-105, read first-hand this pass) |
| 4 | SC4: `authenticate{zai-api-key}` on a REAL account enables working turns, success verified by registry read-back, never the ack | ✓ VERIFIED (field-proven) | Machinery: probe-create arm unchanged since the prior pass (zero commits to auth.go/readback.go since their verifying cycle — git-verified this pass); ack-mismatch negative green (hostsvc ok 0.190s this pass). Field: attempt 3 PASS recorded (ledger actuals, scrubbed capture committed) |
| 5 | SC5: resume reattaches without replay, close cleans up; the full cycle runs end-to-end on api-key | ✓ VERIFIED (field-proven live half) | Fake tier: TestFullCycle/TestResume/TestCloseTeardown green under -race this pass; session-lifecycle golden green. Live half: attempt 3 recorded PASS (resume no replay, close clean, zero orphans) |
| 6 | 05-01: per-method deadlines (15 s runtime-prefs, 300 s permission → deny, runtime-headers coded error until key); user-input → {action:decline}; bypass lane; RouterEvent expiry records | ✓ VERIFIED (regression) | Responder policy table unchanged (zero commits to interact since the prior pass — git-verified); interact ok + goldens green this pass |
| 7 | 05-02: deny-by-default (unknown/missing optionId, malformed outcome, expiry → deny or -32021, NEVER allow); no bridge outlives its turn (cancel fan-in) | ✓ VERIFIED (fixed — behavioral evidence) | Deny-by-default half unchanged (TestPermissionExpiry/CancelRace green this pass). The no-dangling invariant now holds on ALL THREE arms, read first-hand this pass: (a) close — sawTurn gate + `cancelPendingForSession` (lifecycle.go:170-178), CR-01 locks green untouched; (b) ACP cancel — atomic observe+take (front.go:429-435), slot-free skip re-homed with discriminating power; (c) SDK displacement — the mine arm's atomic take+fan-in (front.go:476-486), locked by the inverted TestCancelFanInOnlyWhenSlotHeld: the displaced ask earns EXACTLY ONE -32021 reply and no unanswered-at note (assertCancelledFanInBackLeg, cancel_test.go:536-568), green under -race in my own run (26.772s); RED-first proven via validated RED_EVIDENCE_OK records; `assertDanglingAskNote` deleted; cancelPendingForTurn deleted |
| 8 | 05-03/05-05: router fan-out (WR-01), observer removes (WR-07), registry Remove consumer, WR-04 error-path orphan close, close-vs-prompt race gate (T-5-12) | ✓ VERIFIED (regression) | All owning packages green this pass; the claimed close-vs-PROMPT gate holds; the close-vs-RESUME TOCTOU is WR-05 (advisory, recorded — unchanged judgment; the resume section is untouched by cycle 4 per git diff) |
| 9 | 05-04 + a4e0483: WR-03 pump (non-blocking enqueue, keep-latest coalescing, byte ceiling fail-loud, flush-before-stopReason) | ✓ VERIFIED (regression) | All locked tests green in the acpfront -race run this pass; the pump-identity re-check the WR-06 fix builds on is preserved verbatim inside watchCancelFanIn |
| 10 | 05-06: strict snapshot envelope with EXACT-revision formula REUSED from supervisor; read-back never the ack; runtime-headers success shape only from the injected key | ✓ VERIFIED (regression) | hostsvc ok 0.190s this pass; snapshot reuse intact; the machinery's honesty held through all three field attempts (recorded) |
| 11 | 05-07: honest authenticate chain (record→push→refresh→read-back→{}), no-key coded error naming the env var, interruptible clean retry, CredentialDecrypt seam, hygiene locks, binary wires the stack before traffic | ✓ VERIFIED (regression) | cmd green under -race this pass (2.761s); wiring re-grepped (main.go:246-252, :294); scan-secrets exit 0 on the tree incl. all three captures (re-run this pass) |
| 12 | 05-08: fake-tier FULL cycle — one continuous composition, golden-locked, nothing hangs/drops/fabricates | ✓ VERIFIED (regression) | TestFullCycle green under -race this pass; corpus replay ok 7.322s — zero drift from the WR-06/WR-07 cycle (the fake tier was untouched; the scope gate proves it) |
| 13 | 05-09: guarded real-tier test (double guard; skips honestly; default suite untouched); quota ledger; no credential material in repo | ✓ VERIFIED (regression) | Guards re-run this pass: no-tag → `[no test files]` exit 0; tag-without-key → honest skip ok 0.005s. Ledger row at ACTUALS (re-read); EXP-APIKEY-CYCLE-03.jsonl present; scan-secrets exit 0 |

**Score:** 13/13 truths verified (0 present-behavior-unverified — truth 7's cancellation invariant now carries passing behavioral evidence from the rewritten locks; both field truths carry recorded live evidence)

**Prohibitions (25 across all 15 plans — the prior 21 + 05-15's four):** all hold.
- The 05-02 "MUST NOT let a permission bridge outlive its turn" prohibition now holds on ALL THREE cancel arms, test-locked (the prior flag is resolved).
- 05-15's "MUST NOT weaken or delete surviving contract assertions" — verified: the rewrite's git diff shows no modifications to TestCancelFanInStaleArmSkips (untouched), the surviving exactly-once/single-send/no--32800 locks remain (10 helper references), and the slot-free lock was re-homed with MORE discriminating power, not less.
- 05-15's "MUST NOT introduce a lock cycle turnMu ↔ bridge.mu" — verified by code read: every bridge method acquires b.mu only; the takes ride the callers' turnMu sections (one-way turnMu → bridge.mu, documented permission.go:26-31); fires run outside any lock.
- 05-15's "MUST NOT touch REQUIREMENTS.md or prior PLAN/SUMMARY" — verified: REQUIREMENTS.md still shows AUTH-01 `[ ]` and Phase-5 traceability "Gaps Found" (re-read this pass); restoring it is the ORCHESTRATOR's post-verification act on this verdict.
`unverified_prohibitions: 0`.

### Required Artifacts

All previously-verified artifacts carry over (zero commits to any production file outside the four cycle-4 files — git-verified). The cycle-4 artifacts:

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| internal/acpfront/permission.go | session-scoped pending table + promoted surface | ✓ VERIFIED | map[sessionID]map[requestID]cancel (:38); takePendingCancels snapshot-and-delete under lock with empty-bucket cleanup (:159-177); fireTakenPendings — log verbatim + non-empty gate (:183-191); cancelPendingForSession (:198-200); cancelPendingForTurn deleted (grep: no definition); one-way lock-ordering doc (:26-31). First-hand read this pass; green under -race |
| internal/acpfront/front.go | mine-arm fan-in + atomic Front.Cancel section | ✓ VERIFIED | watchCancelFanIn (:466-489): take under the SAME turnMu section as the pump-identity re-check, machine.Cancel + fires outside any lock; Cancel (:429-435): slotHeld + take in ONE section. First-hand read this pass; green under -race |
| internal/acpfront/lifecycle.go | close fan-in re-scoped to session sweep | ✓ VERIFIED (fix) | sawTurn gate unchanged; the fan-in swaps to cancelPendingForSession(sessionID) (:170-178) with the no-atomic-take reasoning documented (closing gate predates the drain). Git diff shows ONLY this change — the resume section untouched (WR-05 scope unchanged) |
| internal/acpfront/cancel_test.go | enshrining tests inverted + scope locks | ✓ VERIFIED | TestCancelFanInOnlyWhenSlotHeld rewritten (:425-488) — legitimacy framing in the doc comment (:397-418), exactly-one -32021 + no unanswered-note assertions; assertDanglingAskNote deleted; TestCancelFanInSkipAndScope (:698-876) — session-scope (foreign round-trip survives, drives the exported seam exactly as production does) + slot-free skip (a pending genuinely present while the slot is free); backLegSettle (:528) keyed to the corpus row's reverse deadline so the single snapshot is final either way. Green under -race |
| evidence/05-15-red-t1/t2 JSON records | RED-first provenance | ✓ VERIFIED | Both re-validated through `gsd_run check tdd-red-evidence` → RED_EVIDENCE_OK (target_test_failed) this pass; failure lines match the SUMMARY's recorded signals |
| Production scope | exactly the four declared files | ✓ VERIFIED | Scope gate re-run: baseline 924f4e7 (last commit touching 05-14-SUMMARY.md), `git diff 924f4e7..HEAD -- internal/` names exactly the four files → `scope-clean` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/acpfront/front.go | internal/acpfront/permission.go | mine-arm fan-in | WIRED (fixed — was the gap) | watchCancelFanIn takes via f.perm.takePendingCancels(sessionID) under the identity re-check section (:476-481), fires via fireTakenPendings (:485) — read first-hand; locked by the rewritten test's -32021 wire assertion |
| internal/acpfront/front.go | internal/acpfront/permission.go | Cancel fan-in | WIRED | Front.Cancel :433 take + :440 fire, atomic with the :430 slot observation |
| internal/acpfront/lifecycle.go | internal/acpfront/permission.go | close fan-in | WIRED (re-scoped) | lifecycle.go:177 cancelPendingForSession(sessionID) on the sawTurn signal; CR-01 locks green untouched |
| cmd/zcode-acp/main.go | internal/interact + internal/hostsvc | responder/key/cipher wiring | WIRED (carried) | main.go:246-252, :294 re-grepped this pass |
| internal/acpfront/auth.go | internal/hostsvc/readback.go | probe-create surface | WIRED (carried) | zero commits to either file since their verifying cycle (git-verified); hostsvc + acpfront green this pass |
| all other declared links | — | — | WIRED (carried) | Owning packages re-ran green this pass; corpus replayed green |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| displaced turn's bridge | fan-in cancels | the session-scoped pending table (real registrations via addPending on every tracked ask) | exactly one -32021 reply on the back leg; no dangling note | ✓ FLOWING (fixed — was ✗ DISCONNECTED on the displacement arm) |
| authenticate verdict | registry surfaces | probe create-reply's settings.model.available | field-proven (attempt 3, recorded) | ✓ FLOWING (both tiers) |
| permission bridge | client outcome | ACP client selection | forwarded byte-verbatim (golden); live decisions in attempt 3 | ✓ FLOWING |
| tool stream | tool frames | backend tool lifecycle events | golden-locked; live frames (attempt 3) | ✓ FLOWING |
| full-cycle frames | whole-cycle wire | scripted corpus row (fake) / real desktop (live) | golden-locked + field-proven | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| WR-06 gap closure + all front machinery incl. the rewritten locks, scope locks, CR-01 locks | `go test ./internal/acpfront/ -race -count=1` | ok 26.772s | ✓ PASS |
| Golden corpus + EOF signature through real Conn (zero drift after cycle 4) | `go test -count=1 -run 'TestGolden\|TestEOFSignature' ./internal/zcode/fake/` | ok 7.322s | ✓ PASS |
| Interact responder (SC1 machinery) | `go test ./internal/interact/ -count=1` | ok 1.019s | ✓ PASS |
| Hostsvc read-back (SC4 machinery incl. ack-mismatch negative) | `go test ./internal/hostsvc/ -count=1` | ok 0.190s | ✓ PASS |
| Turn/core/acpport/zcode/supervisor regression | `go test -count=1 ./internal/turn/ ./internal/core/ ./internal/acpport/ ./internal/zcode/ ./internal/supervisor/` | all ok | ✓ PASS |
| Binary E2E incl. wired responder | `go test ./cmd/zcode-acp/ -race -count=1` | ok 2.761s | ✓ PASS |
| Guarded real-tier test: default build exclusion | `go test ./test/realdesktop/ -count=1` | `[no test files]` exit 0 | ✓ PASS |
| Guarded real-tier test: tag without credentials | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -run TestAPIKeyFullCycle ./test/realdesktop/` | ok 0.005s — honest skip | ✓ PASS (honest skip) |
| vet + build | `go vet ./... && go build ./...` | clean | ✓ PASS |
| lint | `mise run lint` | 0 issues | ✓ PASS |
| Production scope precision | `git diff 924f4e7..HEAD -- internal/` vs the four declared files | scope-clean | ✓ PASS |
| Credential hygiene (tree incl. three captures) | `bash experiments/scan-secrets.sh` | exit 0 | ✓ PASS |
| RED evidence records | `gsd_run check tdd-red-evidence` × 2 | RED_EVIDENCE_OK × 2 | ✓ PASS |
| Cycle-4 commit ancestry | `git merge-base --is-ancestor` a7f67f9/7f18358/b358cfa/8add9e6 HEAD | all ancestors | ✓ PASS |
| Live run on the real account | guarded command, human-run per the 05-14 checkpoint | attempt 3 PASS 49.43s recorded (05-UAT.md item 1, ledger actuals, scrubbed capture) — NOT re-run this pass by design (zero-spend cycle; the recorded field evidence stands) | ✓ PASS (recorded field evidence) |

### Probe Execution

No probe scripts declared or conventional (`scripts/*/tests/probe-*.sh` absent). Step 7c: not applicable.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/acpfront/cancel_test.go | TOOL-02, CORE-06 | yes | none | none | Behavioral (wire-artifact: exactly-one -32021 reply envelope, no unanswered note, exactly-once cancelled success, single send, containment) | VALID — drives the real SDK displacement through the public surface over the real front bridge |
| internal/acpfront/lifecycle_test.go (CR-01 locks) | TOOL-02, CORE-06 | yes | none | none | Behavioral (exactly-one reply, stop-before-close order, deadline arm) | VALID — green untouched through cycle 4 |
| internal/acpfront/permission_test.go | TOOL-02 | yes | none | none | Behavioral + value-level (exactly-one reply envelope pattern) | VALID |

**Disabled tests on requirements:** 0. **Circular patterns detected:** 0. **Insufficient assertions:** 0. The rewritten locks assert wire artifacts on the back leg through the real connection — the strongest tier available in this tiering; no RED evidence was fabricated (both records re-validated through the tool independently).

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts (gate re-run this pass: 6/6 honored, 0 not honored, non-blocking).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| AUTH-01 | 05-06, 05-07, 05-09 (+05-10/05-11 gap plans) | api-key path: updateAccountConfig + requestProviderRuntimeHeaders answered correctly (registry state verified, not the ack) | ✓ SATISFIED (field-proven) | Machinery + corrected surface + REAL RUN attempt 3 PASS (recorded). REQUIREMENTS.md keeps AUTH-01 `[ ]` and Phase-5 traceability at "Gaps Found" — restoring those is the ORCHESTRATOR's post-verification act on this passed verdict (verified untouched this pass, per 05-15's prohibition) |
| CORE-06 | 05-01, 05-02, 05-08, 05-09, 05-15 (+06 others) | Host-managed interaction responder always on | ✓ SATISFIED | responder green + goldens + wired before traffic; the WR-07 session-scoping hardens the responder's permission lane (foreign asks survive other sessions' cancels — new lock green) |
| SESS-01 | 05-03, 05-05, 05-08, 05-09 | resume (no replay) and close (child-side cleanup) | ✓ SATISFIED | Both live-proven (attempt 3, recorded) + test-locked; WR-02 fixed and locked; WR-05 TOCTOU narrow race recorded (advisory — unchanged judgment) |
| TOOL-01 | 05-04, 05-08, 05-09 | tool_call/tool_call_update streamed | ✓ SATISFIED | Stream tap + pump locked green; tool frames observed live (attempt 3) |
| TOOL-02 | 05-02, 05-08, 05-09, 05-15 | permission bridged; cancelled outcome on ACP-side cancel | ✓ SATISFIED — fully | The WR-06 caveat from the prior pass is RESOLVED: permission bridged both directions, and the cancelled outcome now lands on ALL cancel paths (ACP cancel, close, SDK displacement) — each test-locked on the wire. The review's "TOOL-02 on that arm" binding is discharged |

Orphaned requirements: none — REQUIREMENTS.md maps exactly AUTH-01, CORE-06, SESS-01, TOOL-01, TOOL-02 to Phase 5, and every ID appears in plan frontmatter (re-extracted from all 15 plans this pass: AUTH-01 ×7, CORE-06 ×8, SESS-01 ×8, TOOL-01 ×6, TOOL-02 ×7).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| internal/acpfront/lifecycle_test.go | 582 | Stale doc-comment reference to the deleted `cancelPendingForTurn` (historical RED framing inside TestCloseWithPendingPermissionCancelled's provenance note) | ℹ️ Info | Doc rot only — the test's assertions are the live contract and green; rename-to-cancelPendingForSession is cosmetic |
| — | — | Debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) in the four cycle-4 files | none | Zero matches (re-grepped this pass) |
| — | — | Stub patterns (empty returns, placeholder phrases) in the modified files | none | Zero matches; every modified path carries real logic locked by behavioral tests |
| internal/acpfront/lifecycle.go | resume section | WR-05 resume closing-gate TOCTOU (carried advisory — untouched by cycle 4, git-verified) | ⚠ Warning (advisory) | Narrow race; unreachable in the recorded field run; disposition open, feeds the hardening cycle |
| test/realdesktop + cmd | IN-01..07 sites | Carried review infos (hardening backlog per the disposition ledger) | ℹ️ Info | None observed in the field-proven run |

### Advisory (New Scope, Unevidenced)

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| 1 | WR-05 resume closing-gate TOCTOU (carried from the delta review) | architectural | new-scope review interleaving analysis; the resume code section untouched by cycle 4 (git-diff verified this pass); no failing test; disposition open — feeds the hardening cycle. Honest weighing against the must-haves: it sits under truth 8's close-vs-prompt gate claim (which holds) and no scored truth asserts a close-vs-resume atomicity invariant — same classification as the prior pass, re-weighed this pass |
| 2 | IN-04..07 (carried review infos) | other | hardening backlog per the disposition ledger; none observed in the field-proven run |

(WR-07 is no longer advisory — it is CLOSED this cycle with deterministic locks.)

### Human Verification Required

None. The prior report's two human items remain DISCHARGED with recorded evidence (attempt 3 PASS 49.43s human-run per the 05-14 checkpoint; evidence recording human-confirmed — 05-UAT.md 2/2 pass). The WR-06 closure is unit-verifiable and was unit-verified this pass; per 05-15's design decision (recorded, and agreed by this verifier) no new live checkpoint is required: the displacement arm is a cancel-path invariant locked deterministically through the real SDK client, and the phase goal's live half is covered by the recorded field run. Zero real-account spend this pass — by design.

### Gaps Summary

None. The phase's single remaining gap (WR-06 — the displaced-prompt fan-in arm leaving the displaced turn's permission bridge dangling to the 300s responder deadline, enshrined by committed tests) is closed with the strongest evidence class this project has produced: an enshrining-test INVERSION (legitimacy framed in the test doc comment and the RED commit), a genuine RED at HEAD validated independently through the tool (0 reply envelopes + the honest unanswered note — exactly the defect signal), a production fix whose atomicity is by construction (check-and-take in one turnMu section), and green behavioral locks under -race asserting the wire artifact (exactly one -32021 reply, no dangling note, exactly-once cancelled success, stale-arm containment). The batched WR-07 closure removes the adjacent race class at the same time (session-scoped table, atomic takes on all three arms, foreign-session and slot-free round-trips provably spared).

All 13 truths verify. All 25 prohibitions hold. Full gates green first-hand this pass: acpfront -race 26.772s, corpus 7.322s, interact/hostsvc/turn/core/acpport/zcode/supervisor green, cmd -race green, vet/build/lint clean, scope-clean, scan-secrets 0, guards honest. Both field truths carry recorded live evidence (attempt 3, ledgered). The phase goal — a complete chat cycle on a real z.ai account via api-key, with streamed tool calls, permission decisions answered from the client on every path, working session lifecycle, and the host-managed responder always on — is achieved.

The orchestrator's post-verification acts on this verdict: restore AUTH-01 (and the other four Phase-5 requirement rows, plus the Phase-5 traceability column) in REQUIREMENTS.md.

---

_Verified: 2026-11-21T21:38:53Z_
_Verifier: Claude (gsd-verifier)_
