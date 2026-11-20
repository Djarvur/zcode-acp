---
phase: 05-interactions-full-chat-cycle-api-key
verified: 2026-11-20T16:36:59Z
status: human_needed
score: 11/13 must-haves verified
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
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-USER-SETUP.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/05-VALIDATION.md"
  - ".planning/phases/05-interactions-full-chat-cycle-api-key/COVERAGE.md"
  - "cmd/fake-app-server/main.go"
  - "cmd/zcode-acp/main.go"
  - "cmd/zcode-acp/main_test.go"
  - "cmd/zcode-acp/testdata/responder-e2e.json"
  - "experiments/README.md"
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
covered_digest: "v4:sha256:7724a6a82a514c12abb844973437e5339e074a3ed5aa2d0c551c20cffd5959ad"
behavior_unverified: 2
overrides_applied: 0
unverified_prohibitions: 0
decision_coverage:
  honored: 6
  total: 6
  not_honored: []
behavior_unverified_items:
  - truth: "authenticate{zai-api-key} on a REAL account enables working turns, with success verified by reading registry state back — not by trusting the provider/updateAccountConfig ack (SC4's real-account half)"
    test: "Run the canonical guarded command from test/realdesktop/apicycle_test.go's header (ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/) with ZCODE_ACP_API_KEY exported from the local desktop config and the GNOME keyring unlocked — per 05-USER-SETUP.md"
    expected: "authenticate succeeds ONLY on the hostsvc registry read-back verdict (models present + receivedRevision echo), one tiny tool-less turn + one file-read tool turn complete, and a mid-turn requestCredentialCipher ask (D-06) is answered from the login keyring"
    why_human: "Needs a real z.ai account, real credentials, an unlocked keyring, and quota spend (EXP-APIKEY-CYCLE-01) — external-service integration the CI tiers never touch by contract; the guarded test skips in ~2 ms without the env var (verified this run), so no automated run can exercise the real path"
  - truth: "A full cycle — new → prompt with tool calls and a permission decision → resume → close — runs end-to-end ON THE REAL api-key account (SC5's live half)"
    test: "Same guarded run as above; additionally record the evidence trail: scrub the raw capture via experiments/scrub-capture.mjs to experiments/captures/EXP-APIKEY-CYCLE-01.jsonl, cite the V-EXP id, and replace the PENDING ledger row (experiments/README.md:195) with actuals"
    expected: "The cycle completes with zero orphans at staged shutdown: resume reattaches without replay, close runs the ordered teardown, post-close prompt is rejected cleanly, tool frames and the permission decision match the fake-tier golden shapes on the real wire"
    why_human: "The live wire behavior of the real 3.18.2 desktop bundle (registry surfaces, cipher ask timing, tool execution) is observable only by running against the real account; recording quota spend and handling credentials are human-held acts (D-05)"
human_verification:
  - test: "END-OF-PHASE REAL RUN (harvested from 05-09-PLAN.md <human-check>, coverage block D3 human_judgment:true): export ZCODE_ACP_API_KEY from the local ZCode desktop config (never commit), confirm the keyring unlocks and secret-tool lookup resolves the 43-char key, then run: ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/"
    expected: "PASS — authenticate{zai-api-key} proven by the registry read-back verdict (not the ack), one tiny tool-less turn, one file-read tool turn, session/resume (re-issued subscribe, no replay), session/close (clean teardown, post-close prompt rejected), staged shutdown with the zero-orphan proof; any interaction/requestProviderRuntimeHeaders ask is answered by the wired key source (CORE-06); the cipher ask is answered from the keyring (D-06 real-tier proof, discharging the 05-07 D5 username-derivation assumption)"
    why_human: "External-service integration on a real account: real credentials, unlocked GNOME keyring, ledgered quota spend (EXP-APIKEY-CYCLE-01, D-05 minimum: one read-back + two tiny turns) — the contract explicitly reserves this as the orchestrator/human act, and the double guard makes it unrunnable in CI by design"
  - test: "Evidence recording after the run: node experiments/scrub-capture.mjs <raw-capture> experiments/captures/EXP-APIKEY-CYCLE-01.jsonl --redact-path <workspace>; bash experiments/scan-secrets.sh; append the ledger actuals to experiments/README.md:195"
    expected: "Scrubbed capture lands with the V-EXP id, scan-secrets exits clean (no credential material), the ledger row flips PENDING → actuals (spend recorded, never credentials)"
    why_human: "Credential hygiene and quota accounting are judgment acts on machine-local material; the scrub-first pipeline cannot decide by itself what the raw capture contains"
---

# Phase 5: Interactions & Full Chat Cycle (api-key) — Verification Report

**Phase Goal:** On a real z.ai account via api-key, a client gets a complete chat cycle — streamed tool calls, permission decisions answered from the client, working session lifecycle — with the host-managed interaction responder always on.
**Verified:** 2026-11-20T16:36:59Z
**Status:** human_needed
**Re-verification:** No — initial verification

**MVP-mode note.** ROADMAP marks this phase `mode: mvp`, but the phase goal is phrased in outcome form and fails the `user-story.validate` guard (`valid: false`). Verification proceeded anyway per this milestone's uniform precedent — phases 01–04 are all `mode: mvp` and all four prior verification reports carry User Flow Coverage tables built from the roadmap contract — and per the coordinator's directive to decide autonomously and emit a `status:` from the closed vocabulary. The User Flow Coverage table below is derived from the goal's client-visible flow and the five roadmap success criteria; a `/gsd:mvp-phase 5` reformat would change nothing in the evidence below.

**Security note.** 05-SECURITY.md does not exist — the secure-phase pass never ran for this phase. Known open item per the coordinator; not scored as a gap here (the load-bearing security constraints — key hygiene, deny-by-default, double guard — are test-locked and verified in this report), but it should be tracked to closure by the orchestrator.

## Goal Achievement

### User Flow Coverage (MVP)

| # | Step | Expected | Evidence in Codebase | Status |
|---|------|----------|---------------------|--------|
| 1 | Client authenticates with a z.ai api-key | `authenticate{zai-api-key}` → `{}` ONLY after the registry read-back verdict; missing key → honest coded error naming `ZCODE_ACP_API_KEY`; no fallback to the desktop config store | TestAuthenticate (5 arms incl. ack-negative + interruptible-retry, re-run green this pass); auth-apikey + runtime-headers-answer goldens; auth.go:123-165 chain read first-hand | ✓ (fake tier) |
| 2 | Client opens a session | `session/new{cwd}` → create+subscribe+setModel; Phase-4 contracts intact through the new permission/cancel additions | setmodel-followup golden replays green (zcode/fake pkg run this pass); front.go NewSession untouched in contract | ✓ |
| 3 | Agent streams tool use | `tool_call`/`tool_call_update` with title, kind, status transitions pending→in_progress→completed/failed, absolute-path diffs for Edit/Write; batch/raw/subagent log-only; foreign-turn deltas skipped | TestToolLifecycle (edit + write subtests) green; tool-lifecycle golden locks the ordered wire sequence; stream.go classification read first-hand | ✓ |
| 4 | Backend asks permission; client decides | `session/request_permission` with the backend's options (allow/reject, names+optionIds verbatim); the chosen option's own embedded response forwarded byte-verbatim; ACP-side cancel → backend `-32021`; expiry → deny | permission-allow golden line 15 (`result:{"decision":"allow","reason":"Approved once"}` — byte-verbatim); permission-cancel golden; TestPermissionBridge/Cancel/CancelRace/Expiry green | ✓ |
| 5 | Client resumes a session | `session/resume` reattaches: backend resume → re-issued subscribe, NO setModel, NO replay | lifecycle.go:55-99 read first-hand; TestResume green; session-lifecycle golden | ✓ |
| 6 | Client closes a session | `session/close` ordered teardown (cancel turn → backend close → registry Remove → observer removes); post-close prompt rejected cleanly; close-vs-prompt race resolves one outcome | lifecycle.go:103-213 read first-hand; TestCloseTeardown + TestCloseWhileTurnActiveRace green under -race | ✓ |
| 7 | Full cycle composed (fake tier) | authenticate → new → prompt(tool cards + permission decision) → end_turn → resume → close as ONE golden-locked composition; nothing hangs, drops, or fabricates | TestFullCycle re-run green under -race this pass (0.13s); full-cycle golden locks the whole-cycle wire order incl. started-card-before-ask | ✓ |
| 8 | Full cycle live on the REAL api-key account | Same cycle on the real 3.18.2 install with real key + keyring; scrubbed capture + V-EXP id + ledger actuals recorded | TestAPIKeyFullCycle exists (896 lines, double guard verified: excluded from default builds entirely; skips in ~2 ms under `-tags realdesktop` without env — run this pass, SKIP message confirmed); ledger row PENDING at experiments/README.md:195; **run NOT yet executed** | ⚠️ human (items 1-2 below) |

### Observable Truths

Merged from ROADMAP Success Criteria (contract, T1-T5) + the nine PLAN frontmatter must_haves (grouped into the plan-truth clusters T6-T13), deduplicated. Score: 11/13; the two behavior-unverified truths are both the real-account halves that the plan contract explicitly reserves as the end-of-phase orchestrator/human act.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1: Every server→host `interaction/*` request is answered promptly — re-announced requests dedupe, per-method deadlines fire, unknown methods receive -32601 protocol-error replies instead of crashing the wrapper | ✓ VERIFIED | internal/interact green under -race this pass (2.06s): TestResponderContract (unknown→-32601 naming the method; deadlines; adjacency/empty probes), TestResponderDedupeRace (-race, exactly-once), dedupe table read first-hand (dedupe.go: (method, params.requestId) key, tombstone release, TTL sweep — one entry per distinct requestId, never per frame); reannounce-dedupe + unknown-host-method + user-input-decline goldens replay through a real Conn (zcode/fake green); production wiring `conn.OnRequest = responder.Serve` before traffic (main.go:252, wireResponder read first-hand) |
| 2 | SC2: Tool use streams as `tool_call`/`tool_call_update` with title, kind, status transitions, and file diffs for edit tools | ✓ VERIFIED | TestToolLifecycle (edit: ordered frames, diff at absolute path, foreign delta suppressed; write: new-file diff newText-only) green under -race; tool-lifecycle golden byte-locks the sequence; stream.go kind classification + IN-05 turn-id guard + WR-06 drop-site warn read first-hand (stream.go:117+); WR-03 pump verified under truth 9 |
| 3 | SC3: A backend permission request reaches the client as `session/request_permission` with allow/reject options; the chosen outcome is forwarded to the backend; an ACP-side cancel resolves pending requests as cancelled | ✓ VERIFIED | TestPermissionBridge + TestPermissionCancel + TestPermissionCancelRace + TestPermissionExpiry green under -race (full stack: ACP client decision → acpport sender → front bridge → responder → backend reply, fake expect_answer satisfied); permission-allow golden locks byte-verbatim forwarding (the option's own embedded response, verified in the golden text); permission-cancel golden locks the -32021 reply; option mapping table locked in interact/permission.go (allow_once→allow_once, allow_project/allow_always→allow_always, deny→reject_once — grep first-hand) |
| 4 | SC4: `authenticate{zai-api-key}` on a real account enables working turns, with success verified by reading registry state back — not by trusting the ack | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Machinery complete and behaviorally proven at the fake tier: TestAuthenticate 5 arms green (happy read-back chain; no-key honest error naming ZCODE_ACP_API_KEY; unknown methodId; **ack without registry confirmation errors, never {}**; interrupted authenticate is a clean retry); hostsvc green under -race; auth-apikey + runtime-headers-answer goldens; awaitRegistryContains read-back (readback.go:65-180, conn.AddObserver surface) read first-hand. The REAL-account half is unexercised: the guarded TestAPIKeyFullCycle skips by design (verified this pass) — the live run is the pending human act (items 1-2) |
| 5 | SC5: `session/resume` reattaches without replay and `session/close` cleans up the child-side session; the full cycle runs end-to-end on api-key | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Fake-tier cycle fully proven: TestFullCycle green under -race this pass (golden-locked whole-cycle wire order — the composed rehearsal); TestResume/TestCloseTeardown/TestCloseWhileTurnActiveRace green; session-lifecycle golden; teardown order cancel→close→Remove→CloseObservers read first-hand (lifecycle.go). The live api-key full cycle on the real account is unexercised — same guarded-run human act as truth 4 |
| 6 | 05-01: deadlines fire per policy (runtime-prefs 15 s migrated byte-identical; permission 300 s configurable, expiry → deny; runtime-headers coded error until key injected); requestUserInput answered {action:decline}; replies ride the bypass lane; expiry recorded via RouterEvent + structural log | ✓ VERIFIED | Responder policy table read first-hand (responder.go:39-52 deadlines; WithMethodDeadline knob; WithRouterEvent hook); interact tests green -race; user-input-decline + runtime-preferences goldens green; structural-log discipline asserted by tests (no params/content in log fields) |
| 7 | 05-02: deny-by-default posture — unknown/missing optionId, malformed outcome, expiry resolve deny or -32021, NEVER allow; no bridge outlives its turn (cancel fan-in) | ✓ VERIFIED | denyAnswer/cancelledWireError paths read first-hand (interact/permission.go:88-92,112-117,235-244); TestPermissionExpiry + TestPermissionCancelRace green; fan-in `f.perm.cancelPendingForTurn()` wired in Front.Cancel (front.go:403) and CloseSession (lifecycle.go:135) |
| 8 | 05-03/05-05: multi-session enablement — router-event fan-out (WR-01), exposed observer removes (WR-07), registry Remove first production consumer, WR-04 error-path orphan close, close-vs-prompt race gate (T-5-12) | ✓ VERIFIED | machine.go:200 registers via conn.AddRouterObserver; CloseObservers consumed at lifecycle.go:88-89,159; registry Remove at lifecycle.go:158; closing-gate + drainActiveTurn read first-hand; internal/zcode + internal/turn + internal/core all green (package runs this pass); newsession-orphan-close golden green (WR-04) |
| 9 | 05-04 + a4e0483: WR-03 pump — non-blocking enqueue, keep-latest coalescing (same toolCallId only), byte ceiling fails the turn loud, read goroutine never parks; flush-before-stopReason preserved; the regression-fix sites (flush-context detachment in turnctx.go; permission-ask wire barrier in permission.go) are covered by the passing locks | ✓ VERIFIED | TestStreamBackpressure 3 legs green under -race (stalled sink liveness; coalescing; ceiling fail-loud); TestStreamBeforeStopReason green (15.0s — flush-before-stopReason through release); TestFullCycle's golden locks the started-card-precedes-ask wire order (the barrier's contract); detached flushCtx + releaseFlush-after-bounded-wait read first-hand (turnctx.go:99-128,427-461); barrier() at permission.go:144-169 |
| 10 | 05-06: snapshot strict envelope with EXACT-revision builtin formula REUSED from supervisor.BuiltinRevision (never re-derived); verification is read-back, never the ack; runtime-headers answers the success shape from the injected key, never the failure-shaped arm | ✓ VERIFIED | snapshot.go:135 calls supervisor.BuiltinRevision (grep first-hand); RuntimeHeadersHandler returns interact.RuntimeHeadersKeySource (snapshot.go:258); hostsvc_test green (envelope-shape + read-back locks incl. the ack-mismatch negative); runtime-headers-answer golden green; watcher registered BEFORE push traffic (auth.go:140-143 — the conn hook contract) |
| 11 | 05-07: honest authenticate chain (record→push→refresh→read-back→{}), no-key coded error naming the env var, interruptible clean retry, CredentialDecrypt seam (AES-256-GCM over enc:v2:os, encrypt declines, hygiene locks), binary wires the whole stack before any frame flows | ✓ VERIFIED | auth.go read first-hand (chain + sentinel error + per-step coded errors); TestAuthenticate 5 arms green; cipherdecrypt_test green (layout vectors, decline arm, hygiene); cmd/zcode-acp green (2.97s) incl. the binary-level E2E exercising the wired responder through production stdio; wireResponder read first-hand (main.go:244-253: PermissionForwarder + RuntimeHeadersKeySource + CredentialDecrypt injected; OnRequest assigned before the ACP connection exists); "one key, one source" (NewStaticKey fed to front + responder + snapshot builder) |
| 12 | 05-08: the fake-tier FULL cycle — authenticate (read-back proven) → new → prompt(tool+permission) → end_turn → resume (no replay) → close (teardown) — one continuous composition, golden-locked, nothing hangs/drops/fabricates | ✓ VERIFIED | TestFullCycle green under -race this pass (0.13s); full-cycle.golden byte-locks the cycle (permission ask with toolCallId context verified in golden text); driveFullCycle registered in the driver registry (golden_test.go:269); no per-plan lock was loosened (all leg goldens replay green in the same package run) |
| 13 | 05-09: guarded real-tier test landed (build tag + env double guard; skips in ~2 ms; default suite untouched); quota ledger row exists BEFORE any run; no real credential material in the repo | ✓ VERIFIED | Ran all three checks this pass: no-tag build reports `[no test files]` (tagged file excluded entirely); `-tags realdesktop` run SKIPs in 0.005s with the honest manual-only message (output recorded); canonical header command + quota note + scrub-first evidence discipline in the file header (read first-hand); ledger row `Phase-5 guarded full-cycle run (plan 05-09, EXP-APIKEY-CYCLE-01) — PENDING` at experiments/README.md:195; test reads the key from env at run time only; scan-secrets gate intact (allowlist commit 78ab036 documented) |

**Score:** 11/13 truths verified (2 present, behavior-unverified — both the real-account live halves; machinery for both is verified at the fake tier)

**Prohibitions (all plans, 21 items):** all hold — verified via named tests and goldens, none flagged: no auto-allow (deny paths test-locked), no synthesized frames (goldens byte-lock traceability), auth success only on read-back (ack-negative arm), no failure-shaped runtime-headers success, no hard-coded revision, no key material persisted/logged/committed (hygiene tests + scan-secrets), no CI contact with the real desktop/account (double guard re-run this pass), no loosened cycle goldens (leg goldens replay green), no desktop-config-store fallback (no-key arm), no dangling bridges/orphaned sessions (race tests + WR-04 golden), no per-re-announce replies (dedupe golden + race test), no params in logs (structural-log locks), no pre-resume replay (session-lifecycle golden). `unverified_prohibitions: 0`.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (Gate output: `total: 6, honored: 6, not_honored: []` — run this pass via check.decision-coverage-verify over 05-CONTEXT.md.)

### Required Artifacts

The `verify.artifacts` verb parsed 0/0 for every plan: the plans declare artifacts in inline format (`- path (provides)`) which the verb's frontmatter parser does not recognize, so the verb's "FAIL 0/0" is a vacuous parse, not a negative verdict (#3956 floor: zero checks is not a pass — resolved by first-hand verification below). All 27 plan-declared artifact paths verified first-hand:

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| internal/interact/responder.go | D-01 policy object | ✓ VERIFIED | 527 lines, substantive; policy table + injectable seams read first-hand |
| internal/interact/dedupe.go | bounded (method, requestId) table | ✓ VERIFIED | 140 lines; claim/release tombstone + TTL sweep read first-hand |
| internal/interact/responder_test.go | contract + race locks | ✓ VERIFIED | 706 lines; green under -race this pass |
| internal/zcode/envelope.go | client-request error codes | ✓ VERIFIED | -32020/-32021/-32022 constants present (grep line 187) |
| internal/interact/permission.go | backend bridge | ✓ VERIFIED | 245 lines; locked option table + deny-by-default read first-hand |
| internal/interact/permission_test.go | bridge locks | ✓ VERIFIED | 317 lines; green this pass |
| internal/acpport/types.go + agent.go | neutral vocabulary + delegates | ✓ VERIFIED | PermissionSender + real Resume/Close/Authenticate delegates (stub sites replaced — read first-hand) |
| internal/acpfront/permission.go | front bridge + pending table | ✓ VERIFIED | 176 lines; + a4e0483 ask barrier |
| internal/acpfront/permission_test.go | tracer + cancel/expiry locks | ✓ VERIFIED | 4 test funcs green under -race |
| internal/zcode/fake (re-announce extension) | scripted re-announce | ✓ VERIFIED | fake.go/scenario.go changed; reannounce-dedupe row+golden replay green |
| internal/acpfront/stream.go | tool.updated tap | ✓ VERIFIED | 454 lines; classification + IN-05 + WR-06 read first-hand |
| internal/acpfront/turnctx.go | WR-03 pump | ✓ VERIFIED | 474 lines; + a4e0483 flush-context detachment |
| internal/acpfront/tool_stream_test.go | lifecycle/backpressure locks | ✓ VERIFIED | TestToolLifecycle + TestStreamBackpressure green -race |
| internal/acpfront/lifecycle.go | resume/close contracts | ✓ VERIFIED | 213 lines; teardown order read first-hand |
| internal/acpfront/lifecycle_test.go | SESS-01 locks | ✓ VERIFIED | TestResume/TestCloseTeardown/TestCloseWhileTurnActiveRace green |
| internal/acpfront/auth.go | authenticate chain | ✓ VERIFIED | 181 lines; read first-hand |
| internal/acpfront/auth_test.go | honest-chain locks | ✓ VERIFIED | TestAuthenticate 5 arms green |
| internal/interact/cipherdecrypt.go + test | D-06 seam | ✓ VERIFIED | 444+583 lines; green this pass |
| cmd/zcode-acp/main.go | full-stack wiring | ✓ VERIFIED | wireResponder + key source + flag/env read first-hand |
| cmd/zcode-acp/main_test.go | binary E2E | ✓ VERIFIED | cmd package green (2.97s) incl. responder leg |
| internal/hostsvc/snapshot.go + readback.go | push + read-back | ✓ VERIFIED | 267+302 lines; green -race |
| internal/hostsvc/hostsvc_test.go | envelope + read-back locks | ✓ VERIFIED | 605 lines; green |
| 9 corpus rows + 9 goldens | fake-tier locks | ✓ VERIFIED | all present; replay green (zcode/fake 4.9s) |
| test/realdesktop/apicycle_test.go | guarded real tier | ✓ VERIFIED | 896 lines; double guard exercised this pass |
| experiments/README.md | quota ledger row | ✓ VERIFIED | Phase-5 row at :195, PENDING honestly |

### Key Link Verification

The `verify.key-links` verb reported NOT_WIRED for all 25 declared links with detail "Target not referenced in source". **Diagnosis: false negatives, not missing wiring.** The verb (bin/lib/verify.cjs:1785) does a literal substring check of the target *file path string* inside the source file — structurally inapplicable to Go, where files import packages and never contain target file-path strings. Per the agent's fallback wiring patterns, every link was verified first-hand; all 25 are wired:

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/interact/responder.go | internal/zcode/conn.go | OnRequest seam | WIRED | conn.go:95/349 seam; main.go:252 `conn.OnRequest = responder.Serve` before traffic |
| internal/zcode/fake/fake.go | internal/interact/dedupe.go | re-announce row through real Conn | WIRED | driveReannounceDedupe (golden_test.go:260,684); row replay green |
| internal/zcode/envelope.go | internal/interact/permission.go | error-code constants | WIRED | CodeHostRequestCancelled = -32021 (envelope.go:187); consumed via zcode.WireError (permission.go:77) |
| internal/interact/permission.go | internal/acpfront/permission.go | PermissionForwarder seam | WIRED | main.go:246 `NewPermissionBridge(front.PermissionForward()).Forward` |
| internal/acpfront/permission.go | internal/acpport/agent.go | connection-bound sender | WIRED | BindPermissionSender in NewAgentSideConnection (agent.go:64,85) |
| internal/acpfront/front.go | internal/acpfront/permission.go | cancel fan-in | WIRED | front.go:403 cancelPendingForTurn |
| internal/zcode/conn.go | internal/turn/machine.go | router fan-out + remove funcs | WIRED | machine.go:200 `conn.AddRouterObserver(...)` |
| internal/core/registry.go | internal/acpfront/lifecycle.go | Register/Get/Remove | WIRED | lifecycle.go:87,94,104,158 |
| internal/acpfront/stream.go | internal/acpport/sink.go | ToolNote → sink arms | WIRED | stream.go:117 SessionNote kinds → sink.go:29 noteToNotification |
| internal/acpfront/turnctx.go | internal/acpport/types.go | pump owns emitter closure | WIRED | pump writes acpport.SessionNote (turnctx.go:113-116) |
| tool-lifecycle.json | internal/acpfront/tool_stream_test.go | composed replay | WIRED | golden_test.go:264 + tool_stream_test.go:21 drive/assert the same row |
| internal/hostsvc/snapshot.go | internal/supervisor/discover.go | BuiltinRevision reuse | WIRED | snapshot.go:135 |
| internal/hostsvc/readback.go | internal/zcode/conn.go | observer-based poll | WIRED | readback.go:72 `conn.AddObserver(w.observe)` |
| internal/hostsvc/snapshot.go | internal/interact/responder.go | KeySource seam | WIRED | snapshot.go:258 RuntimeHeadersHandler → interact.RuntimeHeadersKeySource |
| internal/acpfront/auth.go | internal/hostsvc/snapshot.go | authenticate composes machinery | WIRED | auth.go:140-160 WatchRegistry/NewSnapshotBuilder/PushEnvelope/RefreshRegistry |
| internal/interact/cipherdecrypt.go | internal/interact/responder.go | D-06 policy injection | WIRED | responder.go:142-147 WithCredentialDecrypt → Handler: svc.Handle |
| cmd/zcode-acp/main.go | internal/zcode/conn.go | OnRequest assignment | WIRED | main.go:252 |
| full-cycle.json | internal/acpfront/fullcycle_test.go | composed E2E | WIRED | fullcycle_test.go:81 mustLoadScenario("full-cycle"); golden_test.go:269 |
| fullcycle_test.go | internal/acpfront/lifecycle.go | resume/close legs | WIRED | fullcycle_test.go:266,290 client ResumeSession/CloseSession |
| full-cycle.json | internal/hostsvc/readback.go | auth leg gated by read-back | WIRED | cycle's authenticate leg rides the auth-apikey push/refresh/read-back rules; TestFullCycle green proves the verdict gates the cycle |
| session-lifecycle.json | lifecycle_test | resume/close replay | WIRED | driveSessionLifecycle (golden_test.go:265,928); TestResume/TestCloseTeardown green |
| test/realdesktop/apicycle_test.go | cmd/zcode-acp/main.go | same-stack composition | WIRED | apicycle_test.go:189,635,642,657 mirrors run() via exported pieces (NewStaticKey/acpfront.New/WithAuth/NewResponder) |
| experiments/README.md | docs/protocol evidence | ledger pointer format | WIRED | README.md:195-206: EXP-APIKEY-CYCLE-01 capture + V-EXP id + scrub pipeline |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| permission bridge | client outcome | ACP client selection of a backend-declared option | forwarded byte-verbatim (golden line 15) | ✓ FLOWING |
| tool stream | tool frames | backend tool.updated envelopes | every frame traces to one envelope (goldens) | ✓ FLOWING |
| authenticate verdict | registry surfaces | state.updated / create-reply settings.model.available via conn observer (readback.go:72) | read-back drives the verdict; ack-mismatch negative arm proves the ack does NOT drive it | ✓ FLOWING |
| runtime-headers reply | apiKey material | injected KeySource (NewStaticKey from config) | success shape only with key; failures are coded errors | ✓ FLOWING |
| full-cycle frames | whole-cycle wire | scripted fake corpus row | golden-locked composition | ✓ FLOWING (fake tier) |
| real-tier wire | real backend surfaces | real 3.18.2 install + real account | guarded test exists, run pending | ⚠️ (human items) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CORE-06 responder semantics (dedupe/deadlines/unknown/user-input) | `go test ./internal/interact/ -race -count=1` | ok 2.057s | ✓ PASS |
| AUTH-01 machinery (snapshot/read-back/runtime-headers) | `go test ./internal/hostsvc/ -race -count=1` | ok 1.207s | ✓ PASS |
| Composed full cycle (SC5 fake-tier rehearsal) | `go test ./internal/acpfront/ -race -count=1 -run TestFullCycle` | --- PASS 0.13s | ✓ PASS |
| Permission bridge + cancel/expiry/race (SC3) | `go test ./internal/acpfront/ -race -count=1 -run 'TestPermission...'` | 4/4 PASS (0.11-0.63s) | ✓ PASS |
| Tool streaming + backpressure + a4e0483 sites (SC2) | `go test ./internal/acpfront/ -race -count=1 -run 'TestTool...|TestStreamBackpressure|TestStreamBeforeStopReason'` | all PASS (TestStreamBeforeStopReason 15.04s) | ✓ PASS |
| Authenticate chain (SC4 machinery) | `go test ./internal/acpfront/ -race -count=1 -run TestAuthenticate` | 5 arms PASS 1.29s | ✓ PASS |
| Resume/close lifecycle (SC5 machinery) | same run — TestResume/TestCloseTeardown/TestCloseWhileTurnActiveRace | all PASS | ✓ PASS |
| Corpus rows + goldens through real Conn | `go test ./internal/zcode/... ./internal/turn/ ./internal/core/ -count=1` | all ok (fake 4.914s) | ✓ PASS |
| Binary E2E incl. wired responder leg | `go test ./cmd/zcode-acp/ -race -count=1` | ok 2.968s | ✓ PASS |
| Guarded real-tier test: default build exclusion | `go test ./test/realdesktop/ -count=1` | `[no test files]` exit 0 | ✓ PASS |
| Guarded real-tier test: tag without env | `go test -tags realdesktop ./test/realdesktop/ -count=1 -run TestAPIKeyFullCycle -v` | --- SKIP 0.00s ("manual-only: set ZCODE_ACP_REAL_E2E=1 AND build with -tags realdesktop") | ✓ PASS (honest skip) |
| Live run on the real account | guarded command from apicycle_test.go header | not run — keyring/key/quota are human-held inputs | ? SKIP → human items 1-2 |

Full-suite context: the coordinator records the entire workspace suite green this session on this exact tree (plain + -race, `mise run ci` exit 0, lint 0 issues) after the a4e0483 regression fix; every phase-critical package was additionally re-run by this verifier (commands above).

### Probe Execution

No probe scripts declared or conventional (`scripts/*/tests/probe-*.sh` absent; plans' `<probe-ledger>` blocks are the specless edge-resolution ledger, resolved by named tests — COVERAGE.md: 8 applicable → 8 resolved explicit, 0 flagged — all 8 named tests are in the packages re-run green this pass). Step 7c: not applicable.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/interact/responder_test.go | CORE-06 | yes | none | no | behavioral (exactly-once, codes, deadlines; -race) | SUFFICIENT |
| internal/interact/permission_test.go | TOOL-02 | yes | none | no | behavioral (full-stack tracer, byte-verbatim) | SUFFICIENT |
| internal/interact/cipherdecrypt_test.go | AUTH-01 (D-06) | yes | none | no | value-level (layout vectors, decline, hygiene) | SUFFICIENT |
| internal/acpfront/permission_test.go | TOOL-02 | yes | none | no | behavioral (round-trip, -32021, race) | SUFFICIENT |
| internal/acpfront/tool_stream_test.go | TOOL-01 | yes | none | no | behavioral (ordered frames, coalescing, ceiling) | SUFFICIENT |
| internal/acpfront/lifecycle_test.go | SESS-01 | yes | none | no | behavioral (teardown order, race single-outcome) | SUFFICIENT |
| internal/acpfront/auth_test.go | AUTH-01 | yes | none | no | behavioral (5 arms incl. ack-negative) | SUFFICIENT |
| internal/acpfront/fullcycle_test.go | SC5 | yes | none | no | behavioral (golden-locked cycle) | SUFFICIENT |
| internal/hostsvc/hostsvc_test.go | AUTH-01 | yes | none | no | value-level (envelope shape, ack-mismatch negative) | SUFFICIENT |
| internal/zcode/fake/golden_test.go | all (wire order) | yes | none | no | value-level (byte-exact goldens) | SUFFICIENT |
| test/realdesktop/apicycle_test.go | SC4/SC5 live | guarded (by design) | SKIP without env+tag | no | behavioral (manual tier) | BY-DESIGN — the skip is the 03-07 contract; the fake-tier cycle (TestFullCycle) proves composition, the live run routes to human items. Not the only test proving SC4/SC5 machinery. |

**Disabled tests on requirements:** 1 (the guarded real-tier test — intentional double guard, not a hidden disable) → no blocker: the composed-cycle requirement has an active passing test at the composed tier.
**Circular patterns:** none (the scripted fake is the project's Phase-3 test oracle by design, accepted in phases 03/04; goldens are reviewed expectations with drift investigated at the owning tier).
**Insufficient assertions:** 0.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| AUTH-01 | 05-06, 05-07, 05-09 | api-key path: `provider/updateAccountConfig` + `interaction/requestProviderRuntimeHeaders` answered correctly (registry state verified, not the ack) | ✓ SATISFIED (live-account confirmation rides human items) | snapshot push + read-back verdict + runtime-headers handler; TestAuthenticate ack-negative arm; goldens; binary wiring |
| CORE-06 | 05-01, 05-02, 05-08, 05-09 | Host-managed interaction responder always on: every `interaction/*` answered promptly (dedupe, deadlines, error replies to unknown, no crash) | ✓ SATISFIED | internal/interact green -race; goldens; responder wired OnRequest before traffic in the binary |
| SESS-01 | 05-03, 05-05, 05-08, 05-09 | `session/resume` (no replay) and `session/close` (with child-side cleanup) | ✓ SATISFIED | lifecycle.go contracts + tests + session-lifecycle golden; teardown consumes WR-07 removes and Registry.Remove |
| TOOL-01 | 05-04, 05-08, 05-09 | `tool_call`/`tool_call_update` streamed with title, kind, status transitions, file diffs | ✓ SATISFIED | stream.go tap + pump; TestToolLifecycle; tool-lifecycle + full-cycle goldens |
| TOOL-02 | 05-02, 05-08, 05-09 | `session/request_permission` bridged with allow/reject options and `cancelled` outcome on ACP-side cancel | ✓ SATISFIED | both bridge halves; TestPermission*; permission-allow/cancel goldens; -32021 codes |

Orphaned requirements: none — REQUIREMENTS.md maps exactly AUTH-01, CORE-06, SESS-01, TOOL-01, TOOL-02 to Phase 5, and every one is claimed by plan frontmatter (05-09 carries all five). No Phase-5 ID appears in REQUIREMENTS.md without a claiming plan.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | Debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) across all 100 phase-modified implementation files | none | zero matches (scanned this pass) |
| — | — | Placeholder / not-yet-implemented text | none | matches are redaction documentation (experiments/README.md) and test-harness naming only — no stubs |
| — | — | Empty implementations / return-only stubs | none | all `return nil` sites are legitimate success paths in substantive functions; the former unadvertised stubs (agent.go Resume/Close/Authenticate) are replaced by real delegates |
| — | — | Key material in repo | none | test reads key from env at run time; fixtures carry synthetic placeholders; scan-secrets gate intact |

### Human Verification Required

### 1. End-of-phase REAL RUN on the real z.ai api-key account (EXP-APIKEY-CYCLE-01)

**Test:** Per 05-USER-SETUP.md and the canonical header in test/realdesktop/apicycle_test.go: export ZCODE_ACP_API_KEY from the local ZCode desktop config (never commit), confirm the GNOME keyring is unlocked and `secret-tool lookup service 'ZCode Credential Encryption'` resolves the 43-char key, then run `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/`.
**Expected:** PASS — authenticate proven by the registry read-back verdict (not the ack), one tiny tool-less turn, one file-read tool turn, session/resume (no replay), session/close (clean teardown, post-close prompt rejected), staged shutdown with zero orphans; any runtime-headers ask answered by the wired key source; the mid-turn cipher ask answered from the keyring (discharging the 05-07 D5 username-derivation assumption).
**Why human:** Real credentials, unlocked keyring, ledgered quota spend on a real account — the contract (05-09 user_setup + human-check block, coverage D3 human_judgment:true) reserves this as the orchestrator/human act; the double guard makes it unrunnable in CI by design (skip verified this pass).

### 2. Evidence recording after the run

**Test:** Scrub the raw capture (`node experiments/scrub-capture.mjs <raw> experiments/captures/EXP-APIKEY-CYCLE-01.jsonl --redact-path <workspace>`), run `bash experiments/scan-secrets.sh`, append ledger actuals to experiments/README.md:195.
**Expected:** Scrubbed capture + V-EXP id recorded; scan-secrets clean; ledger row PENDING → actuals (spend, never credentials).
**Why human:** Credential hygiene and quota accounting are human judgment acts on machine-local material.

### Gaps Summary

No gaps. Every codebase deliverable of the phase is present, substantive, wired, and behaviorally proven by passing tests this verification ran itself (11/13 truths VERIFIED under -race; 25/25 key links wired first-hand; 27/27 artifacts verified; zero debt markers; zero stubs; requirements all satisfied at the codebase tier). What remains is exactly what the phase contract reserves for the human: the live run of the guarded test against the real z.ai api-key account (SC4/SC5's real halves — truths 4 and 5 are present + wired + fake-tier-proven, with the real-account transition unexercised until that run), plus its evidence recording. The quota ledger honestly reads PENDING (experiments/README.md:195). Two process notes for the orchestrator: (1) 05-SECURITY.md was never produced — the secure-phase pass did not run for this phase; (2) the `verify.key-links` verb's filename-literal matcher yields structural false negatives on Go sources (all 25 links disproven as NOT_WIRED by direct code evidence) — worth a follow-up for Go projects.

---

_Verified: 2026-11-20T16:36:59Z_
_Verifier: Claude (gsd-verifier)_
