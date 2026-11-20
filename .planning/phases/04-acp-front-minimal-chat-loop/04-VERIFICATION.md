---
phase: 04-acp-front-minimal-chat-loop
verified: 2026-11-20T03:02:33Z
status: passed
score: 10/11 must-haves verified
covered_files: [".golangci.yml", ".planning/phases/04-acp-front-minimal-chat-loop/04-01-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-01-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-02-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-02-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-03-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-03-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-04-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-04-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-05-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-05-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-06-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-06-SUMMARY.md", "cmd/fake-app-server/main.go", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "go.mod", "go.sum", "internal/acpfront/cancel_test.go", "internal/acpfront/doc.go", "internal/acpfront/front.go", "internal/acpfront/harness_test.go", "internal/acpfront/initialize_test.go", "internal/acpfront/loop_test.go", "internal/acpfront/newsession_test.go", "internal/acpfront/stream.go", "internal/acpfront/stream_order_test.go", "internal/acpfront/stream_test.go", "internal/acpfront/turnctx.go", "internal/acpport/agent.go", "internal/acpport/agent_test.go", "internal/acpport/doc.go", "internal/acpport/pin_test.go", "internal/acpport/sink.go", "internal/acpport/types.go", "internal/core/registry.go", "internal/core/registry_test.go", "internal/supervisor/discover.go", "internal/supervisor/e2e_test.go", "internal/turn/machine.go", "internal/zcode/conn.go", "internal/zcode/conn_test.go", "internal/zcode/envelope.go", "internal/zcode/fake/golden_test.go", "internal/zcode/fake/roundtrip_test.go", "internal/zcode/fake/stdio.go", "internal/zcode/fake/testdata/golden/cancel-mid-turn.golden", "internal/zcode/fake/testdata/golden/complete-then-cancel.golden", "internal/zcode/fake/testdata/golden/completion-quiet-window.golden", "internal/zcode/fake/testdata/golden/no-reasoning.golden", "internal/zcode/fake/testdata/golden/part-delta-streaming.golden", "internal/zcode/fake/testdata/golden/send-refused-32010.golden", "internal/zcode/fake/testdata/golden/setmodel-followup.golden", "internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json", "internal/zcode/fake/testdata/scenarios/complete-then-cancel.json", "internal/zcode/fake/testdata/scenarios/completion-quiet-window.json", "internal/zcode/fake/testdata/scenarios/no-reasoning.json", "internal/zcode/fake/testdata/scenarios/part-delta-streaming.json", "internal/zcode/fake/testdata/scenarios/send-refused-32010.json", "internal/zcode/fake/testdata/scenarios/setmodel-followup.json", "mise.toml"]
covered_digest: "v4:sha256:4344e5c65ee2ce67096b3518e3fe49cc9bc0f0bbdb8ba51090c6143f5654566d"
behavior_unverified: 0
overrides_applied: 0
unverified_prohibitions: 5
decision_coverage:
  honored: 5
  total: 5
  not_honored: []
deferred:
  - truth: "Review findings WR-01..WR-07, IN-01..IN-05 (multi-session hook theft, sink publish race, tap blocking write, error-path teardown, rejection wording, silent block drops, minor hardening)"
    addressed_in: "Phase 5"
    evidence: "04-REVIEW-DISPOSITION.md routes all 12 open (0 critical) to Phase 5 multi-session/teardown/permission scope; Phase 5 goal/SC cover CORE-06 (interaction responder), SESS-01 (resume/close + child-side cleanup), TOOL-01/02"
---

# Phase 4: ACP Front — Minimal Chat Loop Verification Report

**Phase Goal:** As a v1 chat UI client, I want to open a session against ZCode and run a streamed prompt turn end-to-end, with live text and thought chunks and correct cancellation, against the app-server, so that the minimal chat loop is proven on the real ACP boundary before auth and interaction complexity accrete.
**Verified:** 2026-11-20T03:02:33Z
**Status:** human_needed (non-blocking: 1 backstop abstention + 1 deferred manual spot check + 5 flagged judgment-tier prohibitions — no gaps, no failed must-haves)
**Re-verification:** No — initial verification
**Mode:** mvp (user-story goal validated via `user-story.validate` → true)

## User Flow Coverage

User story: «As a v1 chat UI client, I want to open a session against ZCode and run a streamed prompt turn end-to-end, with live text and thought chunks and correct cancellation, against the app-server, so that the minimal chat loop is proven on the real ACP boundary before auth and interaction complexity accrete.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Client initializes | Honest reply: protocolVersion 1, real agentInfo, exactly two authMethods (zai-api-key, zai-individual-coding-plan), no fs/terminal/loadSession fields | `internal/acpfront/initialize_test.go:37-58` asserts the RAW response frame (byte-exact want + banned-substring loop); green under -race | ✓ |
| Client opens a session | `session/new{cwd}` → backend session; create→subscribe→setModel visible on the wire with model+reasoningLevel; mcpServers accepted; relative cwd rejected pre-wire | `internal/acpfront/newsession_test.go` (TestNewSessionCreatesAndSetsModel asserts transcript order + `assertSetModelFrame`; TestNewSessionAcceptsMcpServers; TestNewSessionRejectsRelativeCwd asserts NO create frame + -32602); golden `internal/zcode/fake/testdata/golden/setmodel-followup.golden:9` byte-locks `"options":{"reasoningLevel":"high"},"persistAsWorkspaceLastUsed":false` | ✓ |
| Client prompts | Live `agent_message_chunk` deltas stream BEFORE the response frame; stopReason only after flush; chunks mirror envelopes 1:1 | `internal/acpfront/stream_order_test.go` (TestStreamBeforeStopReason: terminal-event + quiet-window + zero-delta, raw tee line indexes); `internal/acpfront/stream_test.go` (1:1 count assertion, 3 updates for 3 envelopes); `internal/acpfront/loop_test.go` (TestMinimalChatLoop/happy) | ✓ |
| Backend emits reasoning | Client receives `agent_thought_chunk` (one per reasoning delta, interleaved); zero reasoning → zero thought chunks (never synthesized) | `internal/acpfront/stream.go:57-68` (field switch: text→NoteMessage, reasoning→NoteThought, input/output skipped); TestThoughtChunks positive+negative green over part-delta-streaming / no-reasoning corpus rows | ✓ |
| Client cancels mid-turn | Exactly ONE prompt response, `{"stopReason":"cancelled"}` as SUCCESS, no -32800 anywhere, no second concurrent turn | `internal/acpfront/cancel_test.go` (assertSingleCancelledResponse: exactly-1 response count, -32800 absence scan, assertSingleSend); TestCancelMidTurn / TestSecondPromptRejected / TestCancelAfterCompleteNoOp green | ✓ |
| Outcome: minimal loop proven on the real ACP boundary | The built binary carries initialize → session/new → streamed prompt → cancel over REAL exec pipes in CI | `cmd/zcode-acp/main_test.go` TestAgentBinaryEndToEnd (both binaries built, --app-server override shim, exec pipes, agentInfo.version = version var, setModel asserted on the fake's own transcript, loop 1 end_turn + loop 2 cancelled exactly once, exit 0) — green under -race | ✓ |

The `[outcome]` clause is observably true in the codebase at the CI-provable scope. The one remaining real-client step (a desktop ACP client driving the built binary) is the phase's own documented manual-only, non-gating spot check (VALIDATION.md Manual-Only row; harvested 04-05 `<human-check>`), listed under Human Verification.

## Goal Achievement

### Observable Truths

Merged from ROADMAP.md Success Criteria (contract) + the six PLAN frontmatter must_haves, deduplicated. Score: 10/11 (1 backstop abstention, per the honest-evidence contract).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `initialize` replies protocolVersion 1 with an honest capability set (nothing fs/terminal-related), real agentInfo, both authMethods | ✓ VERIFIED | TestInitializeHonestCapabilities green (-race): byte-exact raw-frame want (`internal/acpfront/initialize_test.go:47-48`) + absence loop for loadSession/readTextFile/writeTextFile/terminal/`"fs"` (`:58`); adapter assembles zero-value AgentCapabilities (`internal/acpport/agent.go:83-106` — the SDK struct has no fs/terminal fields) |
| 2 | Advertised surface exactly equals implemented surface: the 8 unadvertised stable methods answer -32601; authenticate answers an honest not-available error, never fake success | ✓ VERIFIED | TestUnadvertisedStableMethodsAnswerNotFound (raw frames through a real agent-side connection, each -32601 naming the method) + TestAuthenticateHonestError (-32603-class with stable message, `internal/acpport/agent.go:207-216`) — both green |
| 3 | `session/new{cwd}` creates a backend session; the setModel follow-up carrying model + reasoningLevel is visible on the wire (fake transcript asserts the payload); create stays bare | ✓ VERIFIED | TestNewSessionCreatesAndSetsModel green: transcript order create→subscribe→setModel, `assertSetModelFrame` (reasoningLevel + persist:false), create frame carries no model key; corpus golden `setmodel-followup.golden:9` byte-locks the payload; TestGolden/setmodel-followup green |
| 4 | `mcpServers` accepted without error and never forwarded; a RELATIVE cwd is rejected (-32602) BEFORE any backend call — no create frame on the back leg | ✓ VERIFIED | TestNewSessionAcceptsMcpServers (non-empty list succeeds, create IN frame carries no mcp content) + TestNewSessionRejectsRelativeCwd (no-create-frame assertion ordered first, -32602 code) — both green; guard at `internal/acpfront/front.go:157-160` |
| 5 | `session/prompt` streams agent_message_chunk deltas live (1:1 envelope mirror, never merged/split) and returns stopReason only after all pending updates flush — across both completion variants; zero-delta turn still answers end_turn | ✓ VERIFIED | TestStreamBeforeStopReason 3/3 green (16.1s, -race): terminal-event (every chunk index < response index, zero session/update after), quiet-window (response is the last protocol frame), zero-delta (0 chunks + end_turn); 1:1 mirror asserted in TestThoughtChunks/positive ("exactly 3 updates for 3 envelopes") |
| 6 | Backend reasoning deltas → agent_thought_chunk (exactly one per delta, in arrival order); zero reasoning deltas → ZERO thought chunks (never synthesized) | ✓ VERIFIED | TestThoughtChunks positive (1 thought chunk, text "Thinking about it", strictly interleaved between the two message frames, raw-tee check) + negative (zero thought chunks on collector AND raw tee, message chunks intact, end_turn) — both green; mapping at `internal/acpfront/stream.go:60-64` |
| 7 | `session/cancel` mid-turn answers exactly once with stopReason "cancelled" as SUCCESS, never -32800; a second active prompt answers ONE clean invalid-request rejection (both guard layers); cancel-after-complete is silent | ✓ VERIFIED | TestCancelMidTurn (exactly-1 response, cancelled, -32800 absence scan both directions, exactly one session/send) + TestSecondPromptRejected (front-slot leg + backend -32010 refusal leg both answer -32600 with `turnActiveMessage` verbatim) + TestCancelAfterCompleteNoOp (zero new frames, no session/stop) — all green |
| 8 | go.mod requires exactly one dependency — acp-go-sdk v0.20.0 exact pin — gate-locked; no non-test file outside internal/acpport imports the SDK | ✓ VERIFIED | `go.mod` carries exactly one `require github.com/coder/acp-go-sdk v0.20.0`; `mise run tidy-diff` exit 0 (count==1 + fixed-string pin asserted in `mise.toml`); TestSDKImportBoundary green + independent grep: only internal/acpport/{doc,sink,agent}.go import the SDK in non-test files |
| 9 | Ordered notification observer fan-out in zcode.Conn: registration-order synchronous delivery, remove deregisters, two machines on one conn each resolve interleaved turns exactly once (no hook theft) | ✓ VERIFIED | TestObserverFanOut 4/4 subtests green (-race); OnNotification slot fully removed from conn.go (grep: only AddObserver remains); `internal/zcode/conn.go:188-199` |
| 10 | The built zcode-acp binary runs the full loop over real stdio (fake backend via the discovery override seam); --version unchanged; stdout protocol-exclusive | ✓ VERIFIED | TestAgentBinaryEndToEnd green (-race, -count=2 per SUMMARY): built binaries, shim install root through supervisor ExplicitRootInstall, exec pipes, version subtest ("zcode-acp <version>"), setModel payload asserted on the fake's own transcript file, exit 0 staged shutdown |
| 11 | BACKSTOP (04-03 probe ledger / COVERAGE.md): "concurrent ACP requests on one connection are serialized by the SDK dispatcher; front session registry is mutex-guarded" | ⚠️ INSUFFICIENT_SPEC | Registry half: explicit evidence — TestRegistryConcurrentAccess hammers Register/Get/Remove from parallel goroutines under -race, green. SDK-dispatcher half: NO wired test drives two concurrent requests over one connection and asserts serialized dispatch, and no directly observed behavior was recorded; `go test -list` finds no concurrency/serialization test in acpport/acpfront. Per the honest-evidence contract (backstop without explicit evidence → human_needed, never silent pass): abstained, reason insufficient_spec → Human Verification item 1 |

**Score:** 10/11 truths verified (0 present-behavior-unverified; 1 backstop abstention not counted)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | 12 open code-review findings (7 warning / 5 info, 0 critical): OnRouterEvent slot theft at 2+ sessions, bindSink unsynchronized publish, tap's inline ACP write on the read goroutine, error-path session/observer/machine teardown, rejection-message wording, silent image/audio block drops, IN-01..05 hardening notes | Phase 5 | `04-REVIEW-DISPOSITION.md` (all 12 dispositioned open, routed with named seams: CORE-06, SESS-01 session/close + registry release, buffered streaming design); Phase 5 ROADMAP goal/SC 1/2/5 cover interactions, tool streaming, resume/close teardown. Documented routed items per the phase contract — not gaps |
| 2 | 3 flagged probe assumptions (CORE-02/04/05 edge spaces "unclassified" by the probe classifier) | — (carried as flagged assumptions by the plans themselves) | COVERAGE.md ledger equality: 10 = 7 resolved + 3 flagged; each flagged item names its enforcement tests (e.g., CORE-05 → TestCancelMidTurn/TestSecondPromptRejected/TestCancelAfterCompleteNoOp). Informational only |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` / `go.sum` | Single approved dependency, exact v0.20.0 pin | ✓ VERIFIED | 1 require line, v0.20.0; tidy-diff gate asserts count + pin (exit 0) |
| `mise.toml` | tidy-diff gate asserts the dependency expectation | ✓ VERIFIED | count==1 + fixed-string grep + Phase-5 godbus obligation comment |
| `internal/zcode/conn.go` | AddObserver ordered fan-out; OnNotification removed | ✓ VERIFIED | AddObserver at :199, pointer-identity remove, ordered delivery; zero OnNotification occurrences |
| `internal/zcode/envelope.go` | MethodSessionSubscribe / MethodSessionSetModel constants | ✓ VERIFIED | :149-150 |
| `internal/turn/machine.go` | NewMachine registers via the fan-out / SetOnRouterEvent | ✓ VERIFIED | Registers via conn (04-01); SetOnRouterEvent lock-guarded setter (04-06 race fix) |
| `internal/acpport/types.go` | Neutral Handlers vocabulary + error helpers | ✓ VERIFIED | 160 lines: Handlers, DTOs, StopReasons, NewInvalidRequest/NewInternal/NewInvalidParams |
| `internal/acpport/agent.go` | acp.Agent adapter: honest initialize, delegation, -32601 defaults, honest authenticate | ✓ VERIFIED | 314 lines; all 12 methods; wireError coding; contract-tested |
| `internal/acpport/sink.go` | UpdateSink → agent_message_chunk / agent_thought_chunk | ✓ VERIFIED | 44 lines; message/thought mapping, loud unknown kind |
| `internal/acpport/agent_test.go` | TestSDKImportBoundary + adapter contract tests | ✓ VERIFIED | Repo walk + 6 contract tests + connection-tier probes, green |
| `internal/core/registry.go` + test | SessionRegistry (mutex, loud miss) | ✓ VERIFIED | 85 lines; TestRegistryRegisterGetRemove + TestRegistryConcurrentAccess green |
| `internal/acpfront/front.go` | Front: honest init, D-03 session/new, D-04/D-05 prompt/cancel | ✓ VERIFIED | 321 lines, substantive (read in full); no stubs |
| `internal/acpfront/stream.go` | Delta tap: text→message, reasoning→thought, input/output skipped | ✓ VERIFIED | 89 lines; one chunk per envelope, arrival order |
| `internal/acpfront/turnctx.go` | Front-owned turn ctx + slot guard + cancel fan-in | ✓ VERIFIED | 63 lines; acquireTurn nil-release on held slot; release resets slot |
| `internal/acpfront/*_test.go` (7 files) | Named contract locks + tracer + harness | ✓ VERIFIED | loop/initialize/newsession/stream/stream_order/cancel/harness — all compile and pass under -race |
| `cmd/zcode-acp/main.go` + `main_test.go` | First real wiring + binary E2E | ✓ VERIFIED | 370 lines main.go; flags/env, override seam, supervisor composition, staged shutdown; E2E green |
| fake corpus (15 scenarios + 15 goldens + drivers) | Byte-deterministic, fail-loud | ✓ VERIFIED | 15/15 scenarios have goldens; all new drivers registered in goldenDrivers; TestGreen full corpus green |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/acpfront/front.go | internal/zcode/conn.go | tap registered via conn.AddObserver BEFORE any machine (registration order = delivery order) | WIRED | `front.go:116` in New(); machines constructed per NewSession (`:199`) land after the tap |
| internal/acpport/agent.go | internal/acpport/types.go | agentAdapter translates acp.Agent ⇄ Handlers neutral DTOs | WIRED | NewAgent(h Handlers); front (acpfront) implements Handlers, imports acpport not the SDK (`front.go` imports: acpport, core, turn, zcode only) |
| internal/acpfront/turnctx.go | internal/turn/machine.go | Machine.Run on front-owned ctx; sdkCtx.Done fans in to machine.Cancel | WIRED | `front.go:238-249` fan-in goroutine; `turnctx.go` acquireTurn |
| internal/zcode/fake/golden_test.go | testdata/scenarios/setmodel-followup.json | driveSetModelFollowup replays create→subscribe→setModel→send through the real conn | WIRED | Driver registered (:170); golden byte-asserts the wire incl. the setModel payload (golden line 9) |
| cmd/zcode-acp/main.go | internal/supervisor/discover.go | Override seam (--app-server / ZCODE_ACP_APP_SERVER) | WIRED | supervisor.ExplicitRootInstall exported for the override shape; TestAgentBinaryEndToEnd drives the production discovery/spawn argv through it |
| cmd/zcode-acp/main.go | internal/acpport/agent.go | acpport connection over os.Stdout/os.Stdin; version var feeds agentInfo | WIRED | E2E asserts agentInfo.version == the binary's version var |
| internal/acpfront/front.go | fake corpus setmodel-followup | session/new transcript assertions ride the corpus row | WIRED | setModelScenario constant in initialize_test.go; newsession tests ride the same composition |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|-------------------|--------|
| initialize response | agentInfo/authMethods | Front.Initialize (`front.go:127-139`) — version injectable via WithVersion; cmd feeds the binary version var | Yes (E2E asserts version == binary var) | ✓ FLOWING |
| session id | result.session.sessionId | Backend create reply dug via createSessionID map-dig (`front.go:301-321`), fail-loud on empty | Yes (fake transcript) | ✓ FLOWING |
| message/thought chunks | part.delta envelopes | Backend session/event stream → tap field switch → sink; never synthesized | Yes (corpus-scripted backend events; zero-delta turn emits zero chunks — proven negative) | ✓ FLOWING |
| stopReason | machine outcome | turn.Machine Run result mapped through the front (`front.go:263-274`) | Yes | ✓ FLOWING |
| setModel payload | ModelSelection flags | cmd --model/--reasoning-level → WithModel → D-03 follow-up → asserted verbatim on the fake's own transcript file | Yes | ✓ FLOWING |

Note: the fake app-server is the declared CI backend (STATE.md standing constraint: CI never contacts api.z.ai) reached through the production supervisor discovery/spawn path via the operator override seam — a documented test seam, not a stub of an unimplemented production path.

### Behavioral Spot-Checks

All commands re-run first-hand this session (not inherited from SUMMARYs). Exit 0 unless noted.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full phase gate | `mise run ci` | exit 0 — build, vet, lint, test -race ./... (8 packages ok), experiments 22/22 | ✓ PASS |
| Initialize honesty | `go test -race -count=1 -run 'TestInitializeHonestCapabilities\|TestAuthenticateHonestError\|TestUnadvertisedStableMethodsAnswerNotFound' ./internal/acpfront/ ./internal/acpport/` | ok (both packages) | ✓ PASS |
| session/new + golden | `go test -race -count=1 -run 'TestNewSession' ./internal/acpfront/` and `-run 'TestGolden/setmodel-followup' ./internal/zcode/fake/` | ok / ok | ✓ PASS |
| Streaming/flush | `go test -race -count=1 -run 'TestStreamBeforeStopReason\|TestThoughtChunks' ./internal/acpfront/` | ok (16.1s — both completion variants + zero-delta + thought positive/negative) | ✓ PASS |
| Cancel | `go test -race -count=1 -run 'TestCancelMidTurn\|TestSecondPrompt\|TestCancel' ./internal/acpfront/ ./internal/turn/` | ok / ok | ✓ PASS |
| Exec chain | `go test -race -count=1 -run 'TestEndToEndExecChain' ./internal/supervisor/` | ok | ✓ PASS |
| Binary E2E | `go test -race -count=1 -run TestAgentBinaryEndToEnd ./cmd/zcode-acp/` | ok (2.7s) | ✓ PASS |
| Tracer | `go test -race -count=1 -run 'TestMinimalChatLoop' ./internal/acpfront/` | ok | ✓ PASS |
| Full golden corpus | `go test -race -count=1 -run 'TestGolden' ./internal/zcode/fake/` | ok (15 rows) | ✓ PASS |
| Fan-out | `go test -race -count=1 -run TestObserverFanOut ./internal/zcode/` | ok | ✓ PASS |
| SDK seam | `go test -count=1 -run TestSDKImportBoundary ./internal/acpport/` + independent grep (only internal/acpport non-test files import the SDK) + `grep -c '^require' go.mod` = 1 | ok / clean / 1 | ✓ PASS |
| Dependency gate | `mise run tidy-diff` | exit 0 (count==1 + pinned-line grep asserted) | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| (none declared) | `find scripts experiments -path '*tests/probe-*.sh'` | no probe scripts exist in the repo; the plans' "probe-ledger" is the specless probe-fallback ledger (authored into PLAN must_haves), not shell probes | SKIPPED — no probes to run |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| CORE-01 | 04-01..04-06 (all declare) | initialize replies protocolVersion 1, honest capability set, real agentInfo, two authMethods | ✓ SATISFIED | Truths 1-2; TestInitializeHonestCapabilities, TestAuthenticateHonestError, TestUnadvertisedStableMethodsAnswerNotFound; REQUIREMENTS.md marks Complete |
| CORE-02 | 04-01..04-06 | session/new → create + setModel follow-up (reasoningLevel workaround); mcpServers accepted and ignored | ✓ SATISFIED | Truths 3-4; TestNewSessionCreatesAndSetsModel/AcceptsMcpServers/RejectsRelativeCwd + corpus golden; REQUIREMENTS.md marks Complete |
| CORE-03 | 04-01, 04-02, 04-04, 04-05, 04-06 | prompt streams agent_message_chunk live; stopReason only after flush | ✓ SATISFIED | Truth 5; TestStreamBeforeStopReason + TestMinimalChatLoop; REQUIREMENTS.md marks Complete |
| CORE-04 | 04-01, 04-02, 04-04, 04-05, 04-06 | agent_thought_chunk streamed when the backend emits reasoning; never synthesized | ✓ SATISFIED | Truth 6; TestThoughtChunks positive+negative; REQUIREMENTS.md marks Complete |
| CORE-05 | 04-02, 04-04, 04-05, 04-06 | session/cancel stops the turn; stopReason "cancelled" (exactly once; never -32800) | ✓ SATISFIED | Truth 7 + 10; TestCancelMidTurn/SecondPromptRejected/CancelAfterCompleteNoOp + TestAgentBinaryEndToEnd; REQUIREMENTS.md marks Complete |

Orphan check: REQUIREMENTS.md maps exactly CORE-01..05 to Phase 4; every plan's frontmatter declares only these five; no Phase-4-mapped requirement is missing from any plan's `requirements` field. Coverage complete.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/acpfront/initialize_test.go | CORE-01 | yes | none | no (want-string hand-written from the executed probe) | Behavioral (raw-frame + absence) | PASS |
| internal/acpport/agent_test.go | CORE-01, seam | yes | none | no | Behavioral (wire codes, shapes) | PASS |
| internal/acpfront/newsession_test.go | CORE-02 | yes | none | no (transcript asserts independent of golden) | Behavioral (wire order + payload) | PASS |
| internal/acpfront/stream_test.go / stream_order_test.go | CORE-03/04 | yes | none | no (raw tee line indexes) | Behavioral (ordering, counts) | PASS |
| internal/acpfront/cancel_test.go | CORE-05 | yes | none | no | Behavioral (exactly-once, code absence) | PASS |
| internal/acpfront/loop_test.go | CORE-01..05 | yes | none | no (staged transcript asserts) | Behavioral (end-to-end) | PASS |
| cmd/zcode-acp/main_test.go | CORE-05 (exec leg) | yes | none | no (fake's own transcript file) | Behavioral (binary E2E) | PASS |
| internal/zcode/conn_test.go (TestObserverFanOut) | infra (CORE-01..05 enabler) | yes | none | no (signal assertion defeats quiet-window masking) | Behavioral (ordering/remove/theft) | PASS |
| internal/zcode/fake/golden_test.go (TestGolden, 15 rows) | CORE-02/03/04/05 fixtures | yes | none | pinned-snapshot regression locks (recorded via -update, hand-verified at record time); contract correctness asserted independently by the front-tier want-strings | Behavioral (byte-determinism) | PASS |
| internal/supervisor/realdesktop_test.go | Phase-3 manual-only | env-gated | t.Skip (documented manual pattern, pre-existing, not a Phase-4 requirement test) | n/a | n/a | INFO |

**Disabled tests on requirements:** 0 (the only t.Skip is the Phase-3 env-gated real-desktop manual test, unrelated to CORE-01..05)
**Circular patterns detected:** 0 (goldens are pinned-snapshot regression locks; every contract-level correctness claim has an independent explicit assertion)
**Insufficient assertions:** 0 (requirement-linked tests assert at behavioral/wire level, not existence level)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | TBD/FIXME/XXX debt markers in phase-modified production files | none | grep clean (exit 1 = no matches) |
| internal/acpport/agent.go | 213 | "not available" grep hit | ℹ️ Info | This IS the honest authenticate error message — intended behavior, not a placeholder |
| — | — | Empty implementations / console-only handlers / hardcoded empty data flowing to output | none | grep clean across acpport/acpfront/core/cmd/zcode production files |

🛑 Blockers: 0. The debt-marker gate (unreferenced TBD/FIXME/XXX) found nothing.

### Advisory (New Scope, Unevidenced)

None — initial verification; the convergence-evidence gate applies only to re-verification.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (`check.decision-coverage-verify`: total 5, honored 5, not_honored [] — D-01 SDK audit choice, D-02 honesty, D-03 session/new contract, D-04 streaming, D-05 cancel semantics.)

### Security Review

`04-SECURITY.md`: threats_open: 0, status verified (frontmatter re-read this session). All 6 threats (5 unique + T-4-SC supply chain) dispositioned closed with named mitigations — the T-4-SC pin discipline (tidy-diff expectation) and T-4-04 honesty (raw-frame absence assertions + honest authenticate) were re-verified first-hand above. 2 accepted risks documented (structural queue caps; override-seam trust position).

### Code Review

`04-REVIEW.md` + `04-REVIEW-DISPOSITION.md`: 0 critical, 7 warning, 5 info — all 12 dispositioned OPEN and routed to Phase 5 multi-session/teardown/permission scope with named seams, with the explicit obligation that Phase 5 planning must cite the ledger. These are documented routed items within the phase contract (Phase 4 scope is the minimal single-turn loop), not gaps.

### Human Verification Required

**1. CORE-01 concurrency backstop (reason: insufficient_spec — backstop abstention)**

**Test:** Drive two concurrent ACP requests over ONE agent connection (e.g., a client that pipelines `session/new` for a second session while a first turn is streaming, or interleaves initialize with an in-flight prompt) against the built binary or the in-process composition.
**Expected:** The SDK dispatcher serializes the requests — each response matches its request id, no crossed or interleaved frames, and the front's session registry correctly resolves each session's machine (no stuck/hung request). Registry half already proven: TestRegistryConcurrentAccess (parallel goroutines, -race) is green.
**Why human:** The truth is tagged `verification: backstop` (non-inferable). No wired test exercises concurrent request dispatch on one connection (verified via `go test -list` — no concurrency/serialization test in acpport/acpfront), and the SDK dispatcher's serialization is an external-library property that presence checks cannot observe. Per the honest-evidence contract this abstains to human review rather than passing silently.

**2. Real-client spot check (harvested from 04-05 `<human-check>`; VALIDATION.md Manual-Only row — documented deferred, non-gating)**

**Test:** Build the zcode-acp binary; fabricate the shim install root (the main_test.go shape); point any real ACP v1 client at it with `--app-server <root> --model zai-api/GLM-5.3-Flash --reasoning-level high`; run initialize → session/new → one streamed prompt → one mid-turn cancel.
**Expected:** The real client renders the streamed reply live, sees thought chunks when the backend reasons, and the cancelled turn answers exactly once with stopReason cancelled; the binary's stdout stays protocol-clean (client parses it without noise).
**Why human:** Requires a real client binary; the plan marks it manual-only and non-gating, deferred to the phase-end checkpoint.

**3. Judgment-tier prohibitions — human review recommended (flagged, non-authoritative judged verdicts)**

Autonomous judged verdicts (non-authoritative, evidence cited; each is also enforced by the named test where one exists):

- "MUST NOT advertise any capability or authMethod behavior that is not implemented" — judged SATISFIED: zero-value AgentCapabilities by construction; raw-frame absence lock (loadSession/readTextFile/writeTextFile/terminal/fs); the only advertised-but-unimplemented method (authenticate) answers the honest error, never success.
- "MUST NOT fake success for a capability that does not exist (authenticate MUST fail honestly)" — judged SATISFIED: -32603-class RequestError with the stable not-available message; TestAuthenticateHonestError green.
- "MUST NOT synthesize session/update chunks that no backend event produced" — judged SATISFIED with enforcement test: tap emits only from received part.delta envelopes (`stream.go` switch); TestThoughtChunks/negative + zero-delta subtest green.
- "MUST NOT write anything but ACP protocol frames to stdout" — judged SATISFIED: SDK owns every stdout byte in the serving loop (a stray byte would break TestAgentBinaryEndToEnd's SDK-client parse); --version is the pre-protocol CLI path, asserted unchanged; logs stderr slog.
- "MUST NOT leak machine paths or prompt content into ACP error messages or logs" — judged SATISFIED with nuance: turnActiveMessage is the only new client-facing error text (no paths/content); the -32010 transport noise was removed from the wire (TestSecondPromptRejected pins the stable message). Residual: generic `NewInternal("session/create: %v", err)` paths pass underlying error text through — the underlying errors are transport/protocol errors, not path-bearing, but no test pins the absence of path content in every error surface. Review recommended as part of the Phase-5 error-surface work (WR-05/WR-06 family).

### Gaps Summary

No gaps. Every roadmap Success Criteria and every PLAN must-have truth is delivered and locked by named, first-hand re-run green tests — including the binary-boundary E2E (TestAgentBinaryEndToEnd over real exec pipes), the byte-locked setModel payload (golden line 9: `reasoningLevel:"high"` + `persistAsWorkspaceLastUsed:false`), the flush-before-stopReason ordering across both completion variants, and the cancel exactly-once contract with the -32800 absence scan. The SDK seam is structurally locked (boundary test + independent grep + gate-locked single pin).

The phase is NOT auto-passed for two honest reasons, both non-blocking and anticipated by the phase's own contracts: (1) the one `verification: backstop` must-have (CORE-01 concurrency: SDK dispatcher serialization) has no wired test or observed behavior proving its SDK half — under the honest-evidence contract it abstains to human verification (`insufficient_spec`); (2) the phase's own manual-only real-client spot check is documented deferred, non-gating, and is carried here so the end-of-phase checkpoint owns it. Five judgment-tier prohibitions carry non-authoritative judged verdicts (evidence cited) and are flagged for human review. The 12 open code-review findings are documented routed items for Phase 5, which owns the multi-session/teardown scope they belong to.

---

_Verified: 2026-11-20T03:02:33Z_
_Verifier: Claude (gsd-verifier)_
