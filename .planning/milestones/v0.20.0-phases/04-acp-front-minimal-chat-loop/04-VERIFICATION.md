---
phase: 04-acp-front-minimal-chat-loop
verified: 2026-11-21T06:12:21.375Z
status: passed
score: 11/11 must-haves verified
covered_files: [".golangci.yml", ".planning/phases/04-acp-front-minimal-chat-loop/04-01-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-01-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-02-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-02-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-03-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-03-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-04-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-04-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-05-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-05-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-06-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-06-SUMMARY.md", "cmd/fake-app-server/main.go", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "go.mod", "go.sum", "internal/acpfront/auth.go", "internal/acpfront/auth_test.go", "internal/acpfront/cancel_test.go", "internal/acpfront/core01_concurrency_test.go", "internal/acpfront/doc.go", "internal/acpfront/front.go", "internal/acpfront/fullcycle_test.go", "internal/acpfront/harness_test.go", "internal/acpfront/initialize_test.go", "internal/acpfront/lifecycle.go", "internal/acpfront/lifecycle_test.go", "internal/acpfront/loop_test.go", "internal/acpfront/newsession_test.go", "internal/acpfront/permission.go", "internal/acpfront/stream.go", "internal/acpfront/stream_order_test.go", "internal/acpfront/stream_test.go", "internal/acpfront/tool_stream_test.go", "internal/acpfront/turnctx.go", "internal/acpport/agent.go", "internal/acpport/agent_test.go", "internal/acpport/doc.go", "internal/acpport/pin_test.go", "internal/acpport/sink.go", "internal/acpport/types.go", "internal/core/registry.go", "internal/core/registry_test.go", "internal/supervisor/discover.go", "internal/supervisor/e2e_test.go", "internal/turn/machine.go", "internal/turn/machine_test.go", "internal/zcode/conn.go", "internal/zcode/conn_test.go", "internal/zcode/envelope.go", "internal/zcode/fake/fake.go", "internal/zcode/fake/golden_test.go", "internal/zcode/fake/roundtrip_test.go", "internal/zcode/fake/scenario.go", "internal/zcode/fake/stdio.go", "internal/zcode/fake/testdata/golden/cancel-mid-turn.golden", "internal/zcode/fake/testdata/golden/complete-then-cancel.golden", "internal/zcode/fake/testdata/golden/completion-quiet-window.golden", "internal/zcode/fake/testdata/golden/no-reasoning.golden", "internal/zcode/fake/testdata/golden/part-delta-streaming.golden", "internal/zcode/fake/testdata/golden/send-refused-32010.golden", "internal/zcode/fake/testdata/golden/setmodel-followup.golden", "internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json", "internal/zcode/fake/testdata/scenarios/complete-then-cancel.json", "internal/zcode/fake/testdata/scenarios/completion-quiet-window.json", "internal/zcode/fake/testdata/scenarios/no-reasoning.json", "internal/zcode/fake/testdata/scenarios/part-delta-streaming.json", "internal/zcode/fake/testdata/scenarios/send-refused-32010.json", "internal/zcode/fake/testdata/scenarios/setmodel-followup.json", "mise.toml"]
covered_digest: "v4:sha256:c48e735668ff7c367890deaed3a3a5672a36019791d1de4c77a31be1fc73441b"
behavior_unverified: 0
overrides_applied: 0
unverified_prohibitions: 0
decision_coverage:
  honored: 5
  total: 5
  not_honored: []
  note: "check.decision-coverage-verify tool skipped (reason: resolves against the state's current-phase pointer, now past Phase 4 — 'CONTEXT.md missing'); D-01..D-05 verified manually against 04-CONTEXT.md's decision list with this pass's evidence (see Decision Coverage section)"
re_verification:
  previous_status: passed
  previous_score: 11/11
  gaps_closed: []
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "Review findings WR-01..WR-07, IN-01..IN-05 (router fan-out, atomic sink, buffered streaming, teardown, rejection wording / silent block drops, minor hardening) — routed open in 04-REVIEW-DISPOSITION.md"
    addressed_in: "Phase 5"
    evidence: "Disposition ledger routes all 12 (0 critical) to Phase 5; Phase 5 landed the majority on this branch (WR-01 AddRouterObserver, WR-02 atomic sink publish, WR-03 bounded pump ba9d362, WR-04 orphan close 541a5ab, WR-05/IN-04/IN-05 23925e7, WR-06 warn e3f748b) — routing honored, remainder in Phase 5 plans"
  - truth: "Non-SDK real-client integration (Orca chat UI driving the built binary)"
    addressed_in: "Phase 7"
    evidence: "04-UAT.md item 2 scope note: a NON-SDK real client is the Phase 7 manual-E2E done-criterion QA-03; the SDK test client drove the built binary in the discharged spot check (TestAgentBinaryEndToEnd re-run green this pass)"
  - truth: "3 flagged probe assumptions (CORE-02/04/05 edge spaces 'unclassified' by the probe classifier)"
    addressed_in: "— (carried as flagged assumptions by the plans themselves)"
    evidence: "COVERAGE.md ledger equality: 10 = 7 resolved + 3 flagged; each flagged item names its enforcement tests (e.g., CORE-05 → TestCancelMidTurn/TestSecondPromptRejected/TestCancelAfterCompleteNoOp — all re-run green this pass). Informational only"
---

# Phase 4: ACP Front — Minimal Chat Loop Verification Report

**Phase Goal:** As a v1 chat UI client, I want to open a session against ZCode and run a streamed prompt turn end-to-end, with live text and thought chunks and correct cancellation, against the app-server, so that the minimal chat loop is proven on the real ACP boundary before auth and interaction complexity accrete.
**Verified:** 2026-11-21T06:12:21.375Z
**Status:** passed
**Re-verification:** Yes — stale-covered-input refresh. The prior verdict (passed 11/11, 2026-11-20T11:38:16Z) went stale because Phase 5 work continued landing on the same stacked branch: 43 implementation files differ between the prior verification commit (c3246b8) and HEAD, including phase-4 load-bearing files (front.go, stream.go, turnctx.go, main.go, agent.go, types.go, sink.go, envelope.go, cancel_test.go, newsession_test.go, main_test.go, doc.go, conn_test.go, golden_test.go). Must-haves re-checked against the current working tree as-is.

**Mode:** mvp (user-story goal re-validated this pass via `user-story.validate` → true)
**Requirement IDs:** PLAN frontmatter and REQUIREMENTS.md both declare **CORE-01, CORE-02, CORE-03, CORE-04, CORE-05** for Phase 4 (identical lists — no conflict; frontmatter authoritative per contract). The tasking note's "ACP-01, ACP-02, ACP-03" IDs do not exist in REQUIREMENTS.md, ROADMAP.md, or any PLAN frontmatter (grep first-hand) — no such IDs are dropped; the authoritative five are each accounted for below.

## User Flow Coverage

User story: «As a v1 chat UI client, I want to open a session against ZCode and run a streamed prompt turn end-to-end, with live text and thought chunks and correct cancellation, against the app-server, so that the minimal chat loop is proven on the real ACP boundary before auth and interaction complexity accrete.» — format validated this pass.

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Client initializes | Honest reply: protocolVersion 1, real agentInfo, exactly two authMethods (zai-api-key, zai-individual-coding-plan), no fs/terminal/loadSession fields | `internal/acpfront/initialize_test.go` TestInitializeHonestCapabilities asserts the RAW response frame; green this pass (full -race suite + named run); `front.go:186-196` Initialize returns `advertisedAuthMethods()` first-hand | ✓ |
| Client opens a session | `session/new{cwd}` → backend session; create→subscribe→setModel visible on the wire with model+reasoningLevel; mcpServers accepted; relative cwd rejected pre-wire | `newsession_test.go` (TestNewSessionCreatesAndSetsModel + AcceptsMcpServers + RejectsRelativeCwd); golden `setmodel-followup.golden` byte-locks the payload; all green this pass; guard first-hand at `front.go:213-216` | ✓ |
| Client prompts | Live `agent_message_chunk` deltas stream BEFORE the response frame; stopReason only after flush; chunks mirror envelopes 1:1 | `stream_order_test.go` + `stream_test.go` + `loop_test.go` — TestStreamBeforeStopReason green this pass (15.02s incl. both completion variants + zero-delta); field switch first-hand at `stream.go:116-126` | ✓ |
| Backend emits reasoning | Client receives `agent_thought_chunk` (one per reasoning delta, interleaved); zero reasoning → zero thought chunks | `stream.go` field text→NoteMessage / reasoning→NoteThought confirmed first-hand; TestThoughtChunks positive+negative green this pass | ✓ |
| Client cancels mid-turn | Exactly ONE prompt response, `{"stopReason":"cancelled"}` as SUCCESS, no -32800, no second concurrent turn | `cancel_test.go` (TestCancelMidTurn / TestSecondPromptRejected / TestCancelAfterCompleteNoOp) — green this pass; `front.go:394-403` Cancel now also fans in `f.perm.cancelPendingForTurn()` (Phase-5 permission bridge, additive to CORE-05 semantics) | ✓ |
| Outcome: minimal loop proven on the real ACP boundary | The built binary carries initialize → session/new → streamed prompt → cancel over REAL exec pipes in CI | `cmd/zcode-acp/main_test.go` TestAgentBinaryEndToEnd — green this pass (2.05s); `main.go:303` `acpport.NewAgentSideConnection(acpport.NewAgent(front), os.Stdout, os.Stdin)` first-hand | ✓ |

The `[outcome]` clause is observably true in the codebase at the CI-provable scope. The non-SDK real-client step (Orca chat UI) is recorded at Phase 7 QA-03 per the discharged UAT scope note (04-UAT.md item 2: pass).

## Goal Achievement

### Observable Truths

Merged from ROADMAP.md Success Criteria (contract) + the six PLAN frontmatter must_haves, deduplicated — identical set to the prior pass. Score: 11/11; every behavior-dependent truth re-exercised by its named test on the current tree this pass.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `initialize` replies protocolVersion 1 with an honest capability set (nothing fs/terminal-related), real agentInfo, both authMethods | ✓ VERIFIED | TestInitializeHonestCapabilities green this pass; `front.go:186-196` Initialize + `auth.go:170-175` advertisedAuthMethods (exactly two, contract order) read first-hand; Phase 5 edits to auth.go did not touch the honest shape |
| 2 | Advertised surface exactly equals implemented surface: the 8 unadvertised stable methods answer -32601; authenticate answers honestly, never fake success | ✓ VERIFIED | TestUnadvertisedStableMethodsAnswerNotFound green; `agent.go:456` NewMethodNotFound default first-hand. EVOLUTION since the prior pass (not a regression): Authenticate is now REALLY implemented for zai-api-key (Phase 5 05-07, read-back-proven chain — `auth.go:86-111`, TestAuthenticate arms green: transcript push→refresh ordering, no-key honest error BEFORE any backend call, interrupted = clean retry); the coding-plan method answers an honest coded "lands in Phase 6" error (`auth.go:99-106`), never fake success — the honesty contract holds and is strengthened; the phase-4 "not available in this build" wording is superseded by genuine implementation |
| 3 | `session/new{cwd}` creates a backend session; the setModel follow-up carrying model + reasoningLevel is visible on the wire (fake transcript asserts the payload); create stays bare | ✓ VERIFIED | TestNewSessionCreatesAndSetsModel green (0.10s); relative-cwd guard first-hand at `front.go:213-216` (pre-wire -32602); registry.Register green with the slimmed Session struct (TestRegistryConcurrentAccess green this pass) |
| 4 | `mcpServers` accepted without error and never forwarded; a RELATIVE cwd is rejected (-32602) BEFORE any backend call | ✓ VERIFIED | TestNewSessionAcceptsMcpServers + TestNewSessionRejectsRelativeCwd green this pass; guard read first-hand at `front.go:213` |
| 5 | `session/prompt` streams agent_message_chunk deltas live (1:1 envelope mirror) and returns stopReason only after all pending updates flush — across both completion variants; zero-delta turn still answers end_turn | ✓ VERIFIED | TestStreamBeforeStopReason (terminal-event + quiet-window + zero-delta) green this pass (15.02s) on the Phase-5-extended stream.go (bounded pump WR-03) — flush ordering preserved |
| 6 | Backend reasoning deltas → agent_thought_chunk (exactly one per delta, in arrival order); zero reasoning → ZERO thought chunks (never synthesized) | ✓ VERIFIED | TestThoughtChunks positive+negative green (0.01s); field switch first-hand at `stream.go:116-126` (text→NoteMessage, reasoning→NoteThought, input/output skipped) |
| 7 | `session/cancel` mid-turn answers exactly once with stopReason "cancelled" as SUCCESS, never -32800; a second active prompt answers ONE clean invalid-request rejection (both guard layers); cancel-after-complete is silent | ✓ VERIFIED | TestCancelMidTurn / TestSecondPromptRejected / TestCancelAfterCompleteNoOp green this pass (0.10/0.21/0.41s); `front.go:394-403` Cancel read first-hand — the additive permission-bridge fan-in (`f.perm.cancelPendingForTurn()`) extends but does not alter the exactly-once SUCCESS contract |
| 8 | go.mod requires exactly one dependency — acp-go-sdk v0.20.0 exact pin — gate-locked; no non-test file outside internal/acpport imports the SDK | ✓ VERIFIED | `grep -c '^require' go.mod` = 1 (v0.20.0 exact); `mise run tidy-diff` exit 0 this pass; independent grep: only internal/acpport/{agent,doc,sink}.go import the SDK in non-test files — all new Phase-5 packages (interact, hostsvc, acpfront/auth+lifecycle+permission) are SDK-free; TestSDKImportBoundary green |
| 9 | Ordered notification observer fan-out in zcode.Conn: registration-order synchronous delivery, remove deregisters, two machines on one conn each resolve interleaved turns exactly once (no hook theft) | ✓ VERIFIED | conn.go itself unchanged since the prior pass (only conn_test.go touched by Phase 5); TestObserverFanOut green under -race this pass (0.31s); tap-before-machines wiring intact at `front.go:174` (`conn.AddObserver(f.observeEnvelope)` BEFORE any machine) |
| 10 | The built zcode-acp binary runs the full loop over real stdio (fake backend via the discovery override seam); --version unchanged; stdout protocol-exclusive | ✓ VERIFIED | TestAgentBinaryEndToEnd green (2.05s) this pass on the Phase-5-extended main.go: `--version` → stdout `zcode-acp <version>` (`main.go:91`), protocol frames via `NewAgentSideConnection(..., os.Stdout, os.Stdin)` (`main.go:303`), logs re-pointed to stderr (`main.go:97`), override seam `ZCODE_ACP_APP_SERVER` intact (`main.go:40`) — wiring examined first-hand |
| 11 | BACKSTOP (04-03 `verification: backstop`): "concurrent ACP requests on one connection are serialized by the SDK dispatcher; front session registry is mutex-guarded" | ✓ VERIFIED | TestConcurrentRequestDispatch (`internal/acpfront/core01_concurrency_test.go`) drives session/new + initialize concurrently over ONE connection and asserts typed caller identities + registry resolution via a real end_turn prompt + raw-wire 3-request/1-response pairing; green this pass (0.01s, inside the -race full suite); UAT item 1 recorded the -race -count=3 discharge |

**Score:** 11/11 truths verified (0 present-behavior-unverified — every behavior-dependent truth had its named behavioral test run green this pass)

### Deferred Items

Items not yet met but explicitly addressed in later milestone phases (informational only — no status impact).

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Routed review findings (WR-01..WR-07, IN-01..IN-05) | Phase 5 | `04-REVIEW-DISPOSITION.md` ledger (all 12 routed open, 0 critical); Phase 5 landed WR-01/02/03/04/05/06/07 and IN-01/02 on this branch (commits ba9d362, 541a5ab, 23925e7, e3f748b et al.) — routing actively honored |
| 2 | Non-SDK real-client integration (Orca chat UI) | Phase 7 | 04-UAT.md item 2 scope note → QA-03 manual E2E done-criterion |
| 3 | 3 flagged probe assumptions (CORE-02/04/05 edge spaces) | — (flagged by the plans themselves) | COVERAGE.md ledger: 10 = 7 resolved + 3 flagged, each with named enforcement tests (all green this pass) |

### Advisory (New Scope, Unevidenced)

None — the Step 7 scan on this refresh pass found no new-scope concerns (debt-marker gate clean, placeholder/stub scan clean on all phase production files).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` / `go.sum` | Single approved dependency, exact v0.20.0 pin | ✓ VERIFIED | 1 require line; tidy-diff gate exit 0 this pass |
| `mise.toml` | tidy-diff gate asserts the dependency expectation | ✓ VERIFIED | count==1 + fixed-string pin (gate re-run green this pass) |
| `internal/zcode/conn.go` | AddObserver ordered fan-out; router fan-out (WR-01) | ✓ VERIFIED | File unchanged since prior pass (only its test changed); AddObserver + AddRouterObserver + ObserverCounts as previously examined |
| `internal/zcode/envelope.go` | MethodSessionSubscribe / MethodSessionSetModel constants | ✓ VERIFIED | :149-150 confirmed first-hand this pass; Phase-5 additions (interaction/provider codes) additive |
| `internal/turn/machine.go` | NewMachine registers via conn.AddObserver | ✓ VERIFIED | File unchanged since prior pass; machine package green in the -race suite |
| `internal/acpport/types.go` | Neutral Handlers vocabulary + error helpers | ✓ VERIFIED | Phase-5 additions (permission/lifecycle vocabulary) additive; suite green |
| `internal/acpport/agent.go` | acp.Agent adapter: honest initialize, delegation, -32601 defaults, honest authenticate | ✓ VERIFIED | `:456` NewMethodNotFound default first-hand; adapter now delegates Authenticate to Front.Authenticate (real honest chain); no fake-success surface anywhere |
| `internal/acpport/sink.go` | UpdateSink → agent_message_chunk / agent_thought_chunk | ✓ VERIFIED | NoteMessage/NoteThought mapping intact (stream.go consumers green) |
| `internal/acpport/agent_test.go` | TestSDKImportBoundary + adapter contract tests | ✓ VERIFIED | Re-run green this pass |
| `internal/core/registry.go` + test | SessionRegistry (mutex, loud miss) | ✓ VERIFIED | Register/Get/Remove contract green incl. TestRegistryConcurrentAccess under -race |
| `internal/acpfront/front.go` | Front: honest init, D-03 session/new, D-04/D-05 prompt/cancel | ✓ VERIFIED | +294 lines of Phase-5 accretion examined at the contract points: Initialize/NewSession/Prompt/Cancel handlers, cwd guard `:213`, tap registration `:174`, cancel fan-in `:403` — all phase-4 contracts intact |
| `internal/acpfront/stream.go` | Delta tap: text→message, reasoning→thought, input/output skipped | ✓ VERIFIED | Field switch at :116-126 confirmed first-hand; Phase-5 additions (tool notes, bounded pump) additive |
| `internal/acpfront/turnctx.go` | Front-owned turn ctx + slot guard + cancel fan-in | ✓ VERIFIED | acquireTurn `:377` + SDK ctx.Done → machine.Cancel fan-in documented and green |
| `internal/acpfront/*_test.go` (now 13 files) | Named contract locks + tracer + harness + concurrency backstop | ✓ VERIFIED | All compile and pass on this tree (all named locks re-run by name this pass; full -race suite exit 0 across all 10 packages) |
| `cmd/zcode-acp/main.go` + `main_test.go` | First real wiring + binary E2E | ✓ VERIFIED | +129/+119 lines of Phase-5 accretion (auth key sourcing, responder leg) examined at the phase-4 wiring points; E2E green |
| fake corpus (Phase-4 rows: 7 scenarios + 7 goldens + drivers) | Byte-deterministic, fail-loud | ✓ VERIFIED | TestGolden green (incl. setmodel-followup); corpus infra changed by Phase 5 (new rows) but Phase-4 rows replay identically |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/acpfront/front.go | internal/zcode/conn.go | tap registered via conn.AddObserver BEFORE any machine (registration order = delivery order) | WIRED | `front.go:174` confirmed first-hand this pass |
| internal/acpport/agent.go | internal/acpport/types.go | agentAdapter translates acp.Agent ⇄ Handlers neutral DTOs | WIRED | NewAgent(h Handlers) intact; PermissionSender opt-in added (additive); front still imports acpport not the SDK |
| internal/acpfront/turnctx.go | internal/turn/machine.go | Machine.Run on front-owned ctx; sdkCtx.Done fans in to machine.Cancel | WIRED | TestCancelMidTurn green (cancel path exercised); acquireTurn at `:377` |
| internal/zcode/fake/golden_test.go | testdata/scenarios/setmodel-followup.json | driveSetModelFollowup replays create→subscribe→setModel→send through the real conn | WIRED | TestGolden green this pass |
| cmd/zcode-acp/main.go | internal/supervisor/discover.go | Override seam (--app-server / ZCODE_ACP_APP_SERVER) | WIRED | `main.go:40` env const first-hand; TestAgentBinaryEndToEnd green (drives production discovery/spawn argv) |
| cmd/zcode-acp/main.go | internal/acpport/agent.go | acpport connection over os.Stdout/os.Stdin; version var feeds agentInfo | WIRED | `main.go:303` first-hand; E2E green incl. stdout protocol-exclusivity |
| internal/acpfront/front.go | fake corpus setmodel-followup | session/new transcript assertions ride the corpus row | WIRED | Same composition green under the Phase-5 harness changes |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| initialize response | agentInfo/authMethods | Front.Initialize — version injectable via WithVersion; cmd feeds the binary version var | Yes (E2E asserts version == binary var) | ✓ FLOWING |
| session id | result.session.sessionId | Backend create reply dug fail-loud from the create response | Yes (fake transcript) | ✓ FLOWING |
| message/thought chunks | part.delta envelopes | Backend session/event stream → tap field switch → sink; never synthesized | Yes (corpus-scripted backend events; zero-delta negative proven) | ✓ FLOWING |
| stopReason | machine outcome | turn.Machine Run result mapped through the front | Yes (E2E loop 1 end_turn + loop 2 cancelled) | ✓ FLOWING |
| setModel payload | ModelSelection flags | cmd --model/--reasoning-level → D-03 follow-up → asserted verbatim on the fake's transcript | Yes (golden byte-lock) | ✓ FLOWING |

Note: the fake app-server is the declared CI backend (standing constraint: CI never contacts api.z.ai) reached through the production supervisor discovery/spawn path via the operator override seam — a documented test seam, not a stub of an unimplemented production path.

### Behavioral Spot-Checks

The full workspace suite ran ONCE this pass (`go test -race -count=1 ./...` — all 10 packages ok, exit 0, log saved), and the named phase-4 contract tests were additionally run by name in one targeted invocation. All results below are first-hand on the current tree.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full workspace suite (once) | `go test -race -count=1 ./...` | 10/10 packages ok, exit 0 | ✓ PASS |
| CORE-01 backstop | `go test -run TestConcurrentRequestDispatch` (named run) | PASS 0.01s | ✓ PASS |
| Initialize honesty | TestInitializeHonestCapabilities | PASS | ✓ PASS |
| acpport honesty locks | TestUnadvertisedStableMethodsAnswerNotFound + TestSDKImportBoundary | PASS 0.00s | ✓ PASS |
| Authenticate honesty (evolved) | TestAuthenticate arms (read-back chain, no-key, unknown-method, interrupt) | PASS (inside acpfront named run) | ✓ PASS |
| session/new locks | TestNewSessionCreatesAndSetsModel / AcceptsMcpServers / RejectsRelativeCwd (+ Phase-5 OrphanClosed) | PASS 0.10s each | ✓ PASS |
| Streaming flush + thought | TestStreamBeforeStopReason (15.02s) + TestThoughtChunks | PASS | ✓ PASS |
| Cancel locks | TestCancelMidTurn / TestSecondPromptRejected / TestCancelAfterCompleteNoOp | PASS 0.10/0.21/0.41s | ✓ PASS |
| Tracer loop | TestMinimalChatLoop | PASS 0.21s | ✓ PASS |
| Binary E2E | TestAgentBinaryEndToEnd | PASS 2.05s | ✓ PASS |
| Observer fan-out | TestObserverFanOut (-race) | PASS 0.31s | ✓ PASS |
| Registry concurrency | TestRegistryConcurrentAccess (-race) | PASS | ✓ PASS |
| Golden setmodel-followup | TestGolden (full corpus row) | PASS 0.10s | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|-------|
| (none declared) | `find scripts experiments -path '*tests/probe-*.sh'` (re-run this pass) | no probe scripts exist in the repo; the plans' "probe-ledger" is the specless probe-fallback ledger authored into PLAN must_haves, not shell probes | SKIPPED — no probes to run |

### Requirements Coverage

Tasking note resolution: the authoritative requirement list for Phase 4 is **CORE-01..CORE-05** — identical in every PLAN frontmatter and in REQUIREMENTS.md's traceability table. The "ACP-01/02/03" IDs in the tasking note appear in no authoritative source (grep across REQUIREMENTS.md, ROADMAP.md, all six PLANs first-hand); nothing is dropped — all five authoritative IDs are accounted for below.

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| CORE-01 | 04-01..04-04, 04-06 (declare) | initialize replies protocolVersion 1, honest capability set, real agentInfo, two authMethods | ✓ SATISFIED | Truths 1-2 (+ concurrency backstop truth 11); REQUIREMENTS.md marks Complete |
| CORE-02 | 04-01..04-04, 04-06 (declare) | session/new → create + setModel follow-up (reasoningLevel workaround); mcpServers accepted and ignored | ✓ SATISFIED | Truths 3-4 + golden byte-lock; REQUIREMENTS.md marks Complete |
| CORE-03 | 04-01, 04-02, 04-04..04-06 (declare) | prompt streams agent_message_chunk live; stopReason only after flush | ✓ SATISFIED | Truth 5; REQUIREMENTS.md marks Complete |
| CORE-04 | 04-01, 04-02, 04-04..04-06 (declare) | agent_thought_chunk streamed when the backend emits reasoning; never synthesized | ✓ SATISFIED | Truth 6 (positive+negative); REQUIREMENTS.md marks Complete |
| CORE-05 | 04-01, 04-02, 04-05, 04-06 (declare) | session/cancel stops the turn; stopReason "cancelled" (exactly once; never -32800) | ✓ SATISFIED | Truths 7 + 10; REQUIREMENTS.md marks Complete |

Orphan check: REQUIREMENTS.md maps exactly CORE-01..05 to Phase 4; every plan's frontmatter declares only these five; no Phase-4-mapped requirement is missing from any plan's `requirements` field. Coverage complete.

### Decision Coverage

All five 04-CONTEXT.md decisions are honored by shipped artifacts (verified manually this pass — the `check.decision-coverage-verify` tool skips because it resolves against the state's current-phase pointer, now past Phase 4; recorded in frontmatter `decision_coverage.note`):

- **D-01** (SDK base: acp-go-sdk v0.20.0 exact pin, seam package) — honored: single require, tidy-diff exit 0, SDK confined to internal/acpport non-test files (all first-hand this pass).
- **D-02** (initialize honesty: advertise exactly what exists) — honored: honest InitReport (`front.go:186-196`), honest authenticate surfaces (`auth.go:94-111`), raw-frame absence locks green.
- **D-03** (session/new contract: bare create → subscribe → setModel only when configured; mcpServers ignored) — honored: `front.go:207+` + golden byte-lock + newsession tests green.
- **D-04** (streaming: fire-and-forget send lifted to synchronous lifecycle, chunks before stopReason) — honored: TestStreamBeforeStopReason green across both completion variants.
- **D-05** (cancel contract: exactly-once cancelled SUCCESS) — honored: three cancel locks green.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/acpfront/initialize_test.go | CORE-01 | yes | none | no | Behavioral (raw-frame + absence) | PASS |
| internal/acpport/agent_test.go | CORE-01, seam | yes | none | no | Behavioral (wire codes, shapes) | PASS |
| internal/acpfront/newsession_test.go | CORE-02 | yes | none | no (transcript asserts independent of golden) | Behavioral (wire order + payload) | PASS |
| internal/acpfront/stream_test.go / stream_order_test.go | CORE-03/04 | yes | none | no (raw tee line indexes) | Behavioral (ordering, counts) | PASS |
| internal/acpfront/cancel_test.go | CORE-05 | yes | none | no | Behavioral (exactly-once, code absence) | PASS |
| internal/acpfront/loop_test.go | CORE-01..05 | yes | none | no (staged transcript asserts) | Behavioral (end-to-end) | PASS |
| internal/acpfront/core01_concurrency_test.go | CORE-01 backstop | yes | none | no (3 independent assertion layers) | Behavioral (concurrent dispatch serialization) | PASS |
| internal/acpfront/auth_test.go | AUTH-01 (Phase 5) + CORE-01 honesty carry-over | yes | none | no (transcript-sequence assert, not return-value-only) | Behavioral (read-back proven) | PASS |
| cmd/zcode-acp/main_test.go | CORE-05 (exec leg) | yes | none | no (fake's own transcript file) | Behavioral (binary E2E) | PASS |
| internal/zcode/conn_test.go (TestObserverFanOut) | infra enabler | yes | none | no (signal assertion defeats quiet-window masking) | Behavioral (ordering/remove/theft) | PASS |
| internal/zcode/fake/golden_test.go (TestGolden corpus) | CORE-02/03/04/05 fixtures | yes | none | pinned-snapshot regression locks; contract correctness asserted independently by the front-tier want-strings | Behavioral (byte-determinism) | PASS |
| internal/supervisor/realdesktop_test.go | Phase-3 manual-only | env-gated | t.Skip (documented manual pattern, pre-existing, unrelated to CORE-01..05) | n/a | n/a | INFO |

**Disabled tests on requirements:** 0 (grep for t.Skip across all phase-4 requirement test files: no matches this pass; the only t.Skip in the repo is the Phase-3 env-gated real-desktop manual test)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | TBD/FIXME/XXX debt markers in phase-modified production files | none | grep clean this pass |
| internal/acpfront/stream.go | 319 | `case "todoread", "todowrite":` | ℹ️ Info | ZCode backend tool-name literals in the tool-classification switch — not a TODO marker |
| — | — | Empty implementations / console-only handlers / hardcoded empty data flowing to output / placeholders | none | grep clean across all phase production files this pass |

🛑 Blockers: 0. The debt-marker gate found nothing.

### Prohibitions (must_haves.prohibitions, judgment + test tiers)

All five phase prohibitions carry enforcement evidence re-verified this pass; all five were previously human-approved at UAT (04-UAT.md item 3, user-approved per standing autonomous directive with cited evidence) — none unverified:

1. **MUST NOT widen the dependency set** (04-01, test-tier) — enforced by the tidy-diff gate (exit 0 this pass, count==1, exact pin).
2. **MUST NOT synthesize session/update chunks** (04-02/04-06, test-tier) — enforced by TestThoughtChunks negative + zero-delta end_turn (green this pass).
3. **MUST NOT write anything but ACP protocol frames to stdout** (04-02/04-06, test-tier) — enforced by TestAgentBinaryEndToEnd parse cleanliness (green this pass).
4. **MUST NOT advertise behavior that is not implemented** (04-03, judgment-tier, human-approved) — raw-frame absence locks green; the advertisement/implementation boundary remains exact (zai-api-key now genuinely implemented; coding-plan advertised with an honest coded decline, the same disposition UAT approved).
5. **MUST NOT fake auth success / leak machine paths or prompt content into errors** (04-03/04-05, judgment-tier, human-approved) — honest coded errors only (`auth.go:94-111`); structural slog fields only (`auth.go:179-181`); the recorded UAT nuance (generic `NewInternal("%v")` passes underlying transport-error text through un-pinned) remains routed to the Phase-5 error-hygiene family, which has since landed (commit 23925e7).

### Security Review

`04-SECURITY.md`: all 6 threats dispositioned closed; the two re-checkable locks re-verified this pass (T-4-SC pin discipline via tidy-diff exit 0 + single-require grep; T-4-04 honesty via the honesty locks re-run). 2 accepted risks documented (structural queue caps; override-seam trust position).

### Code Review

`04-REVIEW.md` + `04-REVIEW-DISPOSITION.md`: 0 critical, 12 routed findings — Phase 5 has since landed the majority on this branch (WR-01 router fan-out, WR-02 atomic sink, WR-03 bounded pump, WR-04 orphan close, WR-05/06 wording+warn, IN-03/04/05); the remainder live in Phase 5 plan scope that cites the ledger. Documented routed items within the phase contract, not gaps.

### Human Verification Required

None — all items from the prior pass remain discharged in the completed UAT (04-UAT.md: 3/3 passed, 0 pending, 0 issues), and this pass produced no new behavior-unverified truths (every behavior-dependent truth was re-exercised by its named test on the current tree):

1. **CORE-01 concurrency backstop** — DISCHARGED: TestConcurrentRequestDispatch green (re-run this pass).
2. **Real-client spot check** — DISCHARGED: TestAgentBinaryEndToEnd drives the built binary over real exec pipes; the non-SDK real client (Orca) is recorded as Phase 7 QA-03.
3. **Five judgment-tier prohibitions** — DISCHARGED (user-approved, evidence re-verified this pass — see Prohibitions section).

### Gaps Summary

No gaps. All 11 merged must-haves (5 ROADMAP Success Criteria + PLAN truths, deduplicated) are delivered and locked by named tests re-run green first-hand on the current tree — including the binary-boundary E2E, the byte-locked setModel payload, the flush-before-stopReason ordering across both completion variants, the cancel exactly-once contract with the -32800 absence scan, and the CORE-01 concurrency-dispatch backstop.

The Phase 5 accretion that made the prior digest stale was examined at the contract points (front.go, stream.go, turnctx.go, main.go, agent.go, envelope.go, auth.go): every change to a phase-4 load-bearing file is additive or a contract-preserving evolution (permission bridge + cancel fan-in, bounded pump on the tap, tool-note translation, real Authenticate implementation with honest coding-plan decline, orphan-close lifecycle, responder leg in cmd) — no phase-4 contract regressed, confirmed by the targeted named-test runs and the full -race suite green on this exact tree.

Status `passed` is valid: no failed truths, no missing/stub artifacts, no unwired links, no blocker anti-patterns, and the human-verification section is empty (all prior items discharged in the completed UAT).

---

_Verified: 2026-11-21T06:12:21.375Z_
_Verifier: Claude (gsd-verifier)_
