---
phase: 07-hardening-vm-manual-e2e-documentation
verified: 2026-11-23T09:20:37Z
status: passed
score: 36/36 must-haves verified (35 direct + 1 via user override; 0 behavior-unverified)
covered_files: [".planning/phases/07-hardening-vm-manual-e2e-documentation/07-01-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-01-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-02-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-02-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-03-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-03-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-04-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-04-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-05-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-05-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-06-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-06-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-07-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-07-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-08-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-08-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-09-PLAN.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-09-SUMMARY.md",".planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md","README.md","cmd/zcode-acp/main.go","cmd/zcode-acp/main_test.go","cmd/zcode-acp/testdata/omp-binding-e2e.json","docs/protocol/app-server-dialect.md","docs/protocol/install-discovery.md","docs/provenance.md","experiments/README.md","experiments/captures/EXP-VM-QA02.jsonl","experiments/captures/qa03-gui-vm-two-turns.png","experiments/vm-e2e-script.test.mjs","internal/acpfront/conformance_test.go","internal/acpfront/desktop_stream_test.go","internal/acpfront/failurepaths_internal_test.go","internal/acpfront/front.go","internal/acpfront/lifecycle.go","internal/acpfront/lifecycle_internal_test.go","internal/acpfront/lifecycle_test.go","internal/acpfront/pickerrefusal_internal_test.go","internal/acpfront/reconcile.go","internal/acpfront/reconcile_internal_test.go","internal/acpfront/reconcile_retry_test.go","internal/acpfront/reconcile_stream_test.go","internal/acpfront/sesscompat_test.go","internal/acpfront/slashbusy_internal_test.go","internal/acpfront/slashmodel_test.go","internal/acpfront/stream.go","internal/acpfront/terminal_signal_test.go","internal/acpfront/testdata/golden/desktop-delta-streaming.acp.golden","internal/acpfront/testdata/golden/desktop-delta-suppressed.acp.golden","internal/acpfront/testdata/golden/desktop-nodelta-reconcile.acp.golden","internal/acpfront/testdata/golden/desktop-terminal-signal.acp.golden","internal/acpfront/turnctx.go","internal/acpport/agent.go","internal/acpport/agent_test.go","internal/acpport/types.go","internal/hostsvc/snapshot.go","internal/interact/keyringprobe.go","internal/interact/keyringprobe_classify_internal_test.go","internal/supervisor/discover.go","internal/supervisor/discover_test.go","internal/turn/machine.go","internal/turn/machine_test.go","internal/turn/qa03_terminal_test.go","internal/turn/watcher.go","internal/zcode/fake/golden_test.go","internal/zcode/fake/testdata/golden/desktop-delta-messages.golden","internal/zcode/fake/testdata/golden/desktop-delta-streaming.golden","internal/zcode/fake/testdata/golden/desktop-nodelta-breach.golden","internal/zcode/fake/testdata/golden/desktop-nodelta-latedelta.golden","internal/zcode/fake/testdata/golden/desktop-nodelta-lateprojection.golden","internal/zcode/fake/testdata/golden/desktop-nodelta-messages.golden","internal/zcode/fake/testdata/golden/desktop-nodelta-noassistant.golden","internal/zcode/fake/testdata/golden/desktop-nodelta-pollerror.golden","internal/zcode/fake/testdata/golden/desktop-nodelta-slowpoll.golden","internal/zcode/fake/testdata/golden/desktop-terminal-signal.golden","internal/zcode/fake/testdata/golden/desktop-twoturn-lateprojection.golden","internal/zcode/fake/testdata/scenarios/desktop-delta-messages.json","internal/zcode/fake/testdata/scenarios/desktop-delta-streaming.json","internal/zcode/fake/testdata/scenarios/desktop-nodelta-breach.json","internal/zcode/fake/testdata/scenarios/desktop-nodelta-latedelta.json","internal/zcode/fake/testdata/scenarios/desktop-nodelta-lateprojection.json","internal/zcode/fake/testdata/scenarios/desktop-nodelta-messages.json","internal/zcode/fake/testdata/scenarios/desktop-nodelta-noassistant.json","internal/zcode/fake/testdata/scenarios/desktop-nodelta-pollerror.json","internal/zcode/fake/testdata/scenarios/desktop-nodelta-slowpoll.json","internal/zcode/fake/testdata/scenarios/desktop-terminal-signal.json","internal/zcode/fake/testdata/scenarios/desktop-twoturn-lateprojection.json","scripts/vm-e2e.sh"]
covered_digest: "v4:sha256:f1614e7bb357f9b7ea9b0621ca5bea287d5f6f4779478e0bd6e03b5b7cd03234"
behavior_unverified: 0
overrides_applied: 1
overrides:
  - must_have: "QA-03 manual E2E through BOTH auth paths — PATH 1 (zai-api-key): card opens, authenticate{zai-api-key} succeeds, models render, a tool-call + permission turn completes in real Orca on the real account"
    reason: "Accepted-closed by the user's explicit recorded decision (AskUserQuestion 2026-11-23, answer 'Считать пройденным'): the coding-plan path is the phase goal (OAuth + model picker) and the api-key surface is covered by the repository test tier (phase-5 EXP-APIKEY-CYCLE-01 guarded full-cycle PASSED on the real account). Recorded in 07-UAT.md Test 2, commit d142ed1, and now annotated in REQUIREMENTS.md's QA-03 entry."
    accepted_by: "user (recorded AskUserQuestion decision, 2026-11-23)"
    accepted_at: "2026-11-23T00:00:00Z"
re_verification:
  previous_status: human_needed
  previous_score: 35/36
  gaps_closed:
    - "QA-02 — one documented clean-VM field run of scripts/vm-e2e.sh: PASSED 2026-11-22 on the agent-created KVM clean Ubuntu 24.04.5 VM (user's recorded 'организуй ее сам' delegation) — 16 checks / 1 skip, exit 0; recorded in 07-UAT.md Test 1, experiments/README.md EXP-VM-QA02 row, capture EXP-VM-QA02.jsonl, commit 08307a0. Two field-run gap fixes landed and verified: discovery bundled-builtin fallback (b9a85c5) + vm-e2e.sh dead-coproc guard (5a5d618) + the no-desktop-login clean-machine arm (8da356b)."
    - "QA-03 — the Orca card E2E (the phase user gate): CLOSED 2026-11-23. PATH 2 PASSED by the user's real run on their machine (reply text renders, multi-turn with tool calls + vision, account catalog in the picker, /model switch to GLM-5.4 Max, honest resume arm) — the field symptom was fixed through 3 gap cycles (streaming translation 413dc31, terminal signal cd87945, reconcile + bounded window e30cb97) and locked by the delta-review fix cycle 8515654. PATH 1 accepted-closed by the user's recorded decision (override above). Agent-corroborating evidence: 8/8 turns terminal-verified on the fixed binary with the exact production card argv; GUI rendering proven in the VM (Orca card over CDP + fake backend, screenshot experiments/captures/qa03-gui-vm-two-turns.png)."
    - "Close-out regeneration (this run, HEAD a7fc2bb): the three non-blocking findings from the 09:14Z report are settled by commits f4e2b75/a7fc2bb — the experiments/README.md QA-03 row now reads CLOSED PASSED with the final spend actuals (~20 tiny real-account turns + zero-spend fake-tier runs), 07-UAT.md frontmatter settled to `status: passed` with the A1/A3 RESOLVED OBSERVATIONS block, REQUIREMENTS.md QA-02/QA-03 checkboxes flipped to [x] with traceability rows Verified (2026-11-23). Docs-only delta — the 67-file implementation set is byte-identical (git diff d1d541d..a7fc2bb touches no source), so the full gate evidence from the 09:14Z run carries over; the two gates that read the changed docs re-ran green (check-docs.sh exit 0, scan-secrets.sh exit 0)."
  gaps_remaining: []
  regressions: []
prohibitions:
  - requirement_id: SESS-02
    statement: "MUST NOT advertise sessionCapabilities delete/fork/list/load/resume in initialize — advertising delete flips Orca into calling session/delete on its disposable listing sessions"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed re-check this run: initializeResponse (internal/acpport/agent.go:195) sets SessionCapabilities{Options:&{}} ONLY; the mutation-verified honesty guard TestSessionCapabilitiesAdvertisementStayedHonest ran green inside the full -race suite this run (acpfront ok 106.7s)."
  - requirement_id: SESS-02
    statement: "MUST NOT fake success on session/fork — the decline is the SDK's honest -32601 method-not-found shape, byte-locked, never a silent empty result"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed: agent.go's explicit UnstableForkSession arm returns unadvertised(AgentMethodSessionFork); TestSessionForkDeclineByteLocked ran green in the full -race suite this run."
  - requirement_id: SESS-03
    statement: "MUST NOT add a second require line to go.mod — stdlib flag.FlagSet dispatch, not cobra"
    status: verified
    verification: judgment
    resolution: "Machine-verified this run: `grep -c '^require' go.mod` = 1, pinned github.com/coder/acp-go-sdk; enforced by the tidy-diff gate inside `mise run ci` exit 0 (this run)."
  - requirement_id: SESS-03
    statement: "MUST NOT forward an unvalidated /model selector to the backend — validate-first against confirmed configOptions"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed: pinned refusal strings exist exactly once each this run (`Unknown ACP model: ` ×1, `No model selected.` ×1 in front.go); the validate-first intercept and TestModelSlashUnknownRefusalByteLocked ran green in the full -race suite this run."
  - requirement_id: QA-01
    statement: "MUST NOT retire, weaken, or skip-lock any existing corpus suite — the bar is additive"
    status: verified
    verification: judgment
    resolution: "Machine-verified this run: `mise run ci` exit 0 carries the whole -race suite incl. TestGolden (now 44 scenarios — 11 additive desktop-* rows) and TestConformanceV1Arc; zero t.Skip in conformance_test.go."
  - requirement_id: QA-01
    statement: "MUST NOT pass the arc via t.Skip, shortened legs, or disabled assertions"
    status: verified
    verification: judgment
    resolution: "Machine-verified this run: the arc ran green under -race inside mise run ci; the new desktop-* golden rows ADD legs (reconcile/terminal/delta), none removed."
  - requirement_id: QA-02
    statement: "MUST NOT fetch any artifact without checksum verification — no curl-pipe-to-shell, no unverified downloads in the VM script"
    status: verified
    verification: judgment
    resolution: "Machine-verified this run: scripts/vm-e2e.sh still contains zero curl/wget invocations (the header documents build-or-fetch OFF the VM; the field run transferred a locally built binary)."
  - requirement_id: QA-02
    statement: "MUST NOT echo credential values in script output or logs — env-var NAMES only"
    status: verified
    verification: judgment
    resolution: "Machine-verified this run: experiments/scan-secrets.sh exit 0 over the whole tree (re-run after the close-out docs landed); the live no-keyring stage run this run printed no credential material; the field-run capture EXP-VM-QA02.jsonl committed clean (08307a0)."
  - requirement_id: DOCS-01
    statement: "MUST NOT instruct exporting or pasting credential/key material in any doc or example — env-var NAMES and placeholder forms only"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed: check-docs.sh + scan-secrets.sh both exit 0 this run (re-run over the close-out docs); README keeps the ZCODE_ACP_API_KEY placeholder form."
  - requirement_id: DOCS-03
    statement: "MUST NOT omit the Apache-2.0 attribution for ported reference-adapter code"
    status: verified
    verification: judgment
    resolution: "Machine-verified: docs/provenance.md attribution section present; check-docs.sh --phase-gate green this run."
  - requirement_id: QA-03
    statement: "MUST NOT record a passing QA-02/QA-03 verdict without the user's actual manual run — human confirmation precedes any evidence commit"
    status: verified
    verification: judgment
    resolution: "Holds on the recorded evidence, examined per verdict: QA-03 PATH 2 — the user's actual manual run on their machine (07-UAT.md Test 2, commit d142ed1) precedes the evidence commit. QA-03 PATH 1 — no manual run; the user's EXPLICIT recorded acceptance (AskUserQuestion 'Считать пройденным') IS the human confirmation preceding commit d142ed1, carried as a formal override in this frontmatter and now annotated in REQUIREMENTS.md — never a silent pass. QA-02 — the verdict records the agent's real run on a real clean VM under the user's recorded delegation ('организуй ее сам'); the entry names the runner and the authority honestly (no fabricated user run anywhere in the tree). No assumed or anonymous pass exists."
unverified_prohibition_review: "All 11 prohibitions hold; verdicts are LLM-judge (non-authoritative) except where a wired gate ran green this run (tidy-diff, scan-secrets, check-docs, mise run ci). The end-of-phase human checkpoint the judgment items were folded into is now CLOSED by the user's own actions (the QA-03 real run + the path-1 acceptance + the QA-02 delegation), so the human review of the behavior surfaces happened through those gates."
---

# Phase 7: Hardening, VM & Manual E2E, Documentation — Verification Report (Re-verification, close-out regeneration)

**Phase Goal:** The done-v1 bar is met: full CI protocol conformance, a reproducible clean-VM scenario, manual E2E confirmed in a real chat-UI client on a real account through both auth paths, and English documentation including the launch recipe for ACP hosts that only accept preset agents.
**Verified:** 2026-11-23T09:20:37Z
**Status:** passed
**Re-verification:** Yes — regeneration over the close-out commits (previous: human_needed 35/36 at 2026-11-22T15:37:05Z → passed 36/36 at 2026-11-23T09:14:08Z → this regeneration at HEAD a7fc2bb after the orchestrator's close-out docs landed: f4e2b75, a7fc2bb)

## Goal Achievement

### What changed since the 09:14Z passed report (the regeneration scope)

Exactly two commits (f4e2b75, a7fc2bb), docs-only: 07-UAT.md frontmatter settled to `status: passed` + the A1/A3 RESOLVED OBSERVATIONS block; experiments/README.md's QA-03 rows closed (CLOSED PASSED + the final spend actuals appended to the gap-cycle-3 row); REQUIREMENTS.md QA-02/QA-03 checkboxes flipped to `[x]` with the traceability rows "Verified (2026-11-23)" and the QA-03 description annotated with the path-2/path-1 disposition. `git diff d1d541d..a7fc2bb` touches **zero source files** — the 67-file phase implementation set is byte-identical, so every gate result from the 09:14Z run carries over to HEAD a7fc2bb verbatim. The two gates that read the changed docs re-ran green this run (check-docs.sh exit 0, scan-secrets.sh exit 0). The fingerprint was recomputed over the CURRENT tree.

### Observable Truths — ROADMAP Success Criteria (the contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | CI protocol conformance suite green: fake app-server drives initialize → session/new → prompt streaming → tool calls → permissions → cancel | ✓ VERIFIED | `mise run ci` exit 0 at the 09:14Z run (build + vet + lint 0 issues + 10/10 packages under `-race -count=1`, incl. acpfront ok 106.7s carrying TestConformanceV1Arc); source tree unchanged since — evidence carries to HEAD a7fc2bb. |
| 2 | Clients sending `session/delete` keep working (graceful no-op); `session/fork` handled gracefully — never a crash path | ✓ VERIFIED | sesscompat decline rows + delete teardown tests green inside the same full -race run; source unchanged. |
| 3 | Every ACP method Orca calls at card startup/per turn implemented or gracefully handled | ✓ VERIFIED | DISC-02 sweep unchanged; byte-locked decline rows + advertisement honesty lock green; the three QA-03 gap cycles extended coverage on the REAL desktop surface (model.streaming, turn.terminal) — both test-locked (delta truths below). |
| 4 | Clean Ubuntu GNOME VM scenario scripted in the repo, reproducible, incl. the no-keyring error path | ✓ VERIFIED — FIELD-PROVEN | Script live stages re-run green at the 09:14Z run (no-keyring 5/5 PASS exit 0 with the binary-missing sentinel verbatim; contract suite 3/3 incl. the guarded live stripped-PATH arm). Reproducibility field-proven 2026-11-22 on the agent-created KVM clean Ubuntu 24.04.5 VM — 16 checks / 1 skip, exit 0 (07-UAT.md Test 1; ledger EXP-VM-QA02; capture EXP-VM-QA02.jsonl; commit 08307a0). The no-keyring error path exercised BOTH live and in the field. |
| 5 | Manual E2E confirmed (Orca, real account, BOTH auth paths) + documentation published (English README, launch recipe, docs/ protocol notes with provenance + limitations) | ✓ VERIFIED (docs half direct; E2E half: path 2 direct, path 1 via user override) | Docs half: README + provenance gates green, re-run this run over the close-out docs. PATH 2: PASSED by the user's REAL run 2026-11-23 on their machine — reply text renders (field symptom fixed through 3 gap cycles), second turn ran 1m49s with tool-call rendering + vision (the model read the committed capture PNG and described it correctly), account catalog renders in the picker (GLM-5.4 / GLM-5.3 / GLM-5.3-Flash / deepseek-v4.1-flash + Effort Low/High/Max), /model switch took the chat to GLM-5.4 Max, the resume toast after the binary swap was the honest new-session arm — recorded in 07-UAT.md Test 2 (commit d142ed1). PATH 1: accepted-closed by the user's explicit recorded decision — carried as `overrides[0]`, counted in the score per the override contract. Agent-corroborating evidence for the rendering surface: 8/8 turns delivered text before end_turn on the fixed binary with the exact production card argv, and the GUI rendering was proven in the VM (Orca + omp card + fake backend over CDP; both turns render user bubble + Reasoning + reply text verbatim; clean tab-close teardown) — screenshot experiments/captures/qa03-gui-vm-two-turns.png. |

**Score:** 5/5 roadmap SCs verified (1 clause via user override)

### Observable Truths — New-Scope Delta (verified at the 09:14Z run; source unchanged — carries to HEAD a7fc2bb)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| D1 | The real desktop's delta carrier (session/event model.streaming) translates to client chunks: text_delta → agent_message_chunk, reasoning_delta → agent_thought_chunk, exactly one chunk per delta, foreign-turn envelopes skipped | ✓ VERIFIED | stream.go observeModelStreaming + decodeModelStreamingPayload read in full (payload.done deliberately not decoded — the completion machinery owns turn end); TestDesktopDeltaStreaming + golden desktop-delta-streaming green in the full -race run. Field-backed: the user's path-2 run saw rendered reply text. |
| D2 | A completed turn whose pump accepted zero message chunks reconciles the stored reply from the session/messages projection and emits it as ONE chunk BEFORE the stopReason (never a hang; honest empty on failure arms) | ✓ VERIFIED | reconcile.go read in full (reconcileNoDeltaReply hook at front.go's StopCompleted, before release's flush); TestNoDeltaReplyReconcile + golden desktop-nodelta-reconcile green; honest arms (pollerror/noassistant/slowpoll goldens) green. Field-backed: 8/8 turns on the fixed binary + the user's run. |
| D3 | The real terminal signal resolves turns (turn.terminal status / computer-use turn-completed, turnId-matched) and a start-announced turn holds the quiet settle off while the veto budget holds (~2 min total-silence bound), then the honest quiet fallback still ends it | ✓ VERIFIED | machine.go/watcher.go read (classifyTerminalLocked KindTurnTerminal arm; startProvenCarrier arming; quietVetoes budget refilled only on real wire life, poll echo excluded); qa03_terminal_test.go's 5 named tests + golden desktop-terminal-signal green in the full run. Field-backed: the user's 1m49s second turn (a silent think past the old floor) completed with its full reply. |
| D4 | The reconcile retry window is wall-clock exact — the hold never exceeds 4s regardless of poll timing (delta-review WR-02) | ✓ VERIFIED | Code read at reconcile.go:123-141 (loop-top `remaining` guard + min(2s, remaining) ctx bound per poll); reconcile_retry_test.go + golden desktop-nodelta-slowpoll green. |
| D5 | The reconcile correlates the projection to THIS turn — a prior turn's stored reply never rides as this turn's recovery (delta-review WR-01) | ✓ VERIFIED | Code read: turn.started backend stamp recorded on the slot (observeTurnStarted, stream.go; front.turnStamp), entries must be ≥ stamp−1s (reconcileTimeSkew); the stamp-less arm keeps the legacy rule only while unambiguous (≤1 assistant entry — reconcile.go:269-283, "never a guess"). Locked by TestReconcileTwoTurnStaleGuard + golden desktop-twoturn-lateprojection — RED-verified live pre-fix (turn two recovered turn ONE's text) per the disposition ledger. |
| D6 | The stored-reply emit is atomic with the message-count guard — a late delta mid-window cannot duplicate the reply (delta-review WR-03) | ✓ VERIFIED | Code read: pump.enqueueIfNoMessages decides check+enqueue under ONE lock with the errReconcileDuplicate sentinel (turnctx.go:317-356); counter bumps at the single acceptance choke point (appendLocked). TestPumpEnqueueIfNoMessagesAtomic + golden desktop-nodelta-latedelta (mutation lock) green. |
| D7 | A window breach from a late delta fails the turn loud — the prompt tail re-checks the ceiling after the reconcile (delta-review IN-01) | ✓ VERIFIED | Code read at front.go StopCompleted (turnBreach() re-check after reconcileNoDeltaReply, before the end_turn answer); TestReconcileWindowBreachFailsLoud + golden desktop-nodelta-breach green. |
| D8 | Discovery falls back to the install's BUNDLED builtin when the fresh machine has no runtime cache — never displacing a runtime-cache hit, honestly empty with neither source | ✓ VERIFIED | discover.go probeBuiltinBundled read (order, not-consulted and no-install arms); TestDiscoverBundledBuiltinFallback + TestDiscoverRuntimeCacheWinsOverBundled + TestDiscoverBuiltinStaysEmptyWithoutAnySource green; probeCount 8→9 with the row documented. FIELD-PROVEN in the QA-02 run (the bundled-builtin fallback fired on the fresh VM; the paired-env contract held end-to-end). |
| D9 | vm-e2e.sh survives a dead wrapper coproc (records FAIL diagnostics instead of crashing under set -u) and treats the no-desktop-login arm as the clean machine's designed state (hardened only under --require-auth) | ✓ VERIFIED | Code read (agent_send AGENT[1]:- guard + AGENT_DEAD; the "log in via the ZCode desktop first" arm matches the real coded error at internal/acpfront/auth.go:367,373); script live at the 09:14Z run (no-keyring stage 5/5); the field run exercised the no-desktop-login arm PASS (arc replied; builtin-only reply recorded). |

### Observable Truths — Plan Must-Haves (31 truths, per-plan, quick regression)

All 21 artifacts across the 9 plans: **exists, substantive, 21/21 PASS** (`verify.artifacts` exit 0 per plan, at the 09:14Z run). All 18 must_haves.key_links **WIRED** (`verify.key-links` exit 0 per plan, at the 09:14Z run). All previously-verified truths hold: the full -race suite (which carries every test the 31 truths were verified on) ran green; pinned facts re-grepped (refusal strings ×1 each; go.mod one require line; advertisement options-only; paired-env read-back assert at scripts/vm-e2e.sh:801; no-login coded string present). Source unchanged since — no regression possible; the close-out docs were additionally gated (check-docs/scan-secrets green).

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| DOCS-01 | 07-08 | README user surface | ✓ SATISFIED | check-docs.sh exit 0 re-run this run; previously read in full |
| DOCS-02 | 07-08 | Launch recipe for preset-agent-only hosts | ✓ SATISFIED | Recipe present; version floor live-proven at the 09:14Z run (`zcode-acp --version` / `omp --version` = 18.1.9+zacp.0 on a fresh HEAD build) |
| DOCS-03 | 07-08 | docs/ protocol notes + provenance + limitations | ✓ SATISFIED | check-docs.sh exit 0 this run; docs/protocol/app-server-dialect.md carries the live completion-signal table (1d987f4) |
| QA-01 | 07-06 | CI protocol conformance suite green | ✓ SATISFIED | mise run ci exit 0 (10/10 packages under -race); now also `[x]` in REQUIREMENTS.md |
| QA-02 | 07-07, 07-09 | VM test scenario scripted + reproducible from the repo | ✓ SATISFIED — REQUIREMENTS.md now `[x]`, traceability "Verified (2026-11-23)" | Script live-proven AND field-proven 2026-11-22 on the clean KVM Ubuntu 24.04.5 VM (16 checks/1 skip, exit 0; UAT Test 1 + EXP-VM-QA02 + capture) |
| QA-03 | 07-09 | Manual E2E in real Orca, real account, both auth paths | ✓ SATISFIED — REQUIREMENTS.md now `[x]` with the path-2/path-1 disposition annotated (path 2 real run; path 1 accepted-closed — override `overrides[0]`) | 07-UAT.md Test 2 verdict recorded 2026-11-23 (commits 0fab55b, d142ed1); ledger row CLOSED PASSED (f4e2b75); corroborating agent evidence in the ledger |
| SESS-02 | 07-01, 07-03 | session/delete graceful no-op; fork graceful | ✓ SATISFIED | Decline/teardown rows green in the full -race run |
| SESS-03 | 07-01..07-05 | Full Orca dialect coverage incl. persona CLI contract | ✓ SATISFIED | DISC-02 sweep + byte-locked rows green; /model + --model mechanisms green |

**Orphaned requirements:** none — REQUIREMENTS.md maps exactly the 8 phase IDs, every one claimed by at least one plan. The QA-02/QA-03 checkbox flips landed in the close-out commits (f4e2b75/a7fc2bb), completing the phase's designed handoff (07-09: "REQUIREMENTS.md is untouched by this plan — the checkbox flips are the orchestrator's post-verification act").

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full workspace gate (run ONCE at the 09:14Z run; source unchanged since) | `mise run ci` | exit 0 — build, vet, lint 0 issues, 10/10 packages ok under `-race -count=1`, experiments suite 24 pass / 1 guarded skip | ✓ PASS (carries to a7fc2bb) |
| Guarded real-tier compile | `go vet -tags realdesktop ./test/realdesktop/` | clean, exit 0 | ✓ PASS |
| Docs gates (re-run THIS run over the close-out docs) | `bash experiments/check-docs.sh` / `bash experiments/scan-secrets.sh` | both exit 0 | ✓ PASS |
| Version floor (fresh HEAD build at the 09:14Z run) | `/tmp/verify-p7/zcode-acp --version && … omp --version` | `18.1.9+zacp.0` ×2 | ✓ PASS |
| vm-e2e contract suite incl. live arm (09:14Z run) | `ZCODE_ACP_VM_E2E_BIN=… node --test experiments/vm-e2e-script.test.mjs` | 3 pass / 0 fail / 0 skipped (the guarded live stripped-PATH arm ran) | ✓ PASS |
| vm-e2e no-keyring stage (live, 09:14Z run) | `bash scripts/vm-e2e.sh --stage no-keyring --binary <HEAD build>` | 5/5 PASS, exit 0, binary-missing sentinel verbatim, exactly-one-table lock | ✓ PASS |
| Named delta locks (WR-01/02/03, IN-01, terminal, delta-streaming, bundled fallback) | inside the single full -race run (TestReconcileTwoTurnStaleGuard, TestReconcileLateDeltaSingleDelivery, TestReconcileWindowBreachFailsLoud, TestPumpEnqueueIfNoMessagesAtomic, TestTurnAssistantReplyText, TestNoDeltaReplyReconcile, TestDesktopDeltaStreaming, TestDesktopTerminalSignalTiming, TestRealTerminalShapesResolve, TestTerminalSignalBeatsQuietFloor, TestQuietVetoBudgetEndsSilentStartAnnouncedTurn, TestQuietWindowUnchangedWithoutStartSignal, TestDiscoverBundledBuiltinFallback, + the 11 desktop-* golden rows) | all green inside `internal/acpfront ok 106.7s`, `internal/turn ok 4.1s`, `internal/supervisor ok 1.9s`, `internal/zcode/fake ok 24.2s` | ✓ PASS |
| QA-02 field run (recorded human-tier evidence) | agent-executed on the clean KVM VM 2026-11-22 (per the user's delegation) | 16 checks / 1 skip, exit 0 — UAT Test 1 + EXP-VM-QA02 + capture committed (08307a0) | ✓ PASS (recorded) |
| QA-03 path 2 (recorded human-tier evidence) | the user's real Orca run on their machine 2026-11-23 | PASSED — reply text renders, multi-turn tool+vision live, picker catalog, /model switch, honest resume arm (UAT Test 2, d142ed1) | ✓ PASS (recorded) |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|-------|
| experiments/vm-e2e-script.test.mjs | `node --test` with ZCODE_ACP_VM_E2E_BIN (live arm unguarded), 09:14Z run | 3 pass / 0 fail / 0 skip | PASS |
| experiments/ node suite | inside `mise run ci` (test-experiments step), 09:14Z run | 24 ✔ / 0 fail / 1 guarded skip | PASS |

### Data-Flow Trace (Level 4)

No UI renders dynamic data in this phase; the trace applies to doc claims, script assertions, and the reconcile's data path:

| Artifact | Claim/Value | Source | Flows | Status |
|----------|-------------|--------|-------|--------|
| reconcile.go | stored reply text | REAL backend query (`session/messages` poll via f.conn.Call) — never a static return | Yes | ✓ FLOWING |
| stream.go | delta chunk text | the envelope's own payload delta (wire data) | Yes | ✓ FLOWING |
| vm-e2e.sh | paired-env read-back assert (script:801) | the session/new REPLY's models list (not the ack) | Yes | ✓ FLOWING |
| discover.go | bundled builtin file | real filesystem probe of `<root>/resources/config/provider/zcode-builtin.json` | Yes | ✓ FLOWING |
| docs/protocol/app-server-dialect.md §5 | terminal envelope table | live capture (raw3.out/raw5.out on 3.18.2), committed as goldens/scenarios | Yes | ✓ FLOWING |
| 07-UAT.md Test 2 | user-run verdict + resolved observations | the user's run on their machine (recorded; screenshots kept off-repo per discipline, two picker screenshots noted as recorded by the user) | Yes | ✓ FLOWING |
| experiments/README.md QA-03 rows | CLOSED PASSED + final spend actuals | the close-out commit f4e2b75, consistent with the UAT verdict | Yes | ✓ FLOWING |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| scripts/vm-e2e.sh | 3 sites | `XXXXXX` (mktemp templates, not debt markers) | ℹ️ Info (false positive) | none |
| README.md | 213 | word "Placeholder" (Phase-2 CI table text) | ℹ️ Info (false positive, pre-existing) | none |
| internal/acpfront/reconcile.go | 173/266/318 | `return ""` arms — documented honest-empty arms ("never a fabricated chunk"), real code paths | ℹ️ Info (false positive) | none |

**Debt-marker gate:** zero TBD/FIXME/XXX/unreferenced-TODO markers in any phase-modified file (scanned all 67 changed files). **Blockers:** none.

### Close-Out Findings — All Settled (recorded disposition)

The three non-blocking findings from the 09:14Z report, each settled by the close-out commits and verified in the diff:

1. **Stale ledger QA-03 rows — SETTLED (f4e2b75).** experiments/README.md's QA-03 row now reads "CLOSED PASSED (2026-11-23; path 2 by the user's real run, path 1 accepted-closed by the user's recorded decision)" and the gap-cycle-3 row carries the final spend actuals ("~20 tiny real-account turns (one-word replies, no tool calls except the /gsd-manager probe's two Read calls; no PAT re-mints beyond the phase-6 mint-once) + ZERO-spend fake-tier runs"). No contradiction remains between the ledger and the canonical UAT verdict.
2. **Observation A1 — SETTLED honestly (a7fc2bb).** 07-UAT.md's RESOLVED OBSERVATIONS block records: A3 answered by the user's real run (with configOptions and no models field, the picker starts showing the launch-bound model and the account catalog arrives after the read-back — two screenshots recorded by the user); A1 (the exact `--model` id shape from a launch log) honestly recorded as not captured — post-v1 wish, the mechanism byte-locked by TestOmpLaunchBinding. The honest no-evidence record satisfies the observation discipline (never assumed).
3. **UAT frontmatter — SETTLED (a7fc2bb).** 07-UAT.md `status:` now reads `passed`, consistent with both entries' result fields and the file's Summary block (passed 2/2).

**Post-v1 advisories carried (declared, not gaps):** the 8 code-review infos (IN-a..IN-h) + 2 delta-review infos (IN-02 projection-less backends burn the window; IN-03 fixture reindent noise) — all dispositioned post-v1 in 07-REVIEW-DISPOSITION.md (committed at d1d541d); none is a must-have failure.

### Human Verification Required

None. Both previously-human gates are closed with recorded human evidence, and the close-out paper trail is now consistent end-to-end:

- **QA-02** — closed 2026-11-22: the documented field run executed on an actual clean VM (agent-created KVM Ubuntu 24.04.5 under the user's recorded delegation; the entry names the runner and the authority), 16 checks / 1 skip, exit 0, capture committed.
- **QA-03** — closed 2026-11-23: path 2 by the user's real run on their machine; path 1 by the user's explicit recorded acceptance (formalized as the override in this report's frontmatter — a documented human decision, never a silent pass).

### Gaps Summary

**No gaps.** Everything checkable against the codebase is verified with behavioral evidence at HEAD a7fc2bb: all 21 artifacts substantive, all 18 key links wired, all 5 roadmap SCs met (SC-4 field-proven; SC-5's E2E half closed by recorded human evidence with one user-accepted override), all 9 new-scope delta truths test-locked (golden corpus 33→44 rows, every new row green under -race), the full `mise run ci` gate exit 0 at the unchanged source tree, lint 0 issues, docs/secrets gates green re-run over the close-out docs, zero debt markers, zero blockers. The delta-review fix cycle (8515654) is confirmed landed, read in source, and locked by named tests plus RED-verified mutation goldens. The two items that made the first run `human_needed` are closed by the user's own recorded actions, and the close-out commits settled every bookkeeping finding this verifier routed. REQUIREMENTS.md's QA-02/QA-03 checkboxes now carry the flipped state with the honest path-2/path-1 annotation. The phase — and the v1 milestone — close on complete, consistent, honest evidence.

---

_Verified: 2026-11-23T09:20:37Z_
_Verifier: Claude (gsd-verifier)_
