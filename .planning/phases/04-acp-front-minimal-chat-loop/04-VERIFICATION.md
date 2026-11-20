---
phase: 04-acp-front-minimal-chat-loop
verified: 2026-11-20T11:38:16Z
status: passed
score: 11/11 must-haves verified
covered_files: [".golangci.yml", ".planning/phases/04-acp-front-minimal-chat-loop/04-01-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-01-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-02-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-02-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-03-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-03-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-04-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-04-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-05-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-05-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-06-PLAN.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-06-SUMMARY.md", ".planning/phases/04-acp-front-minimal-chat-loop/04-UAT.md", "cmd/fake-app-server/main.go", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "go.mod", "go.sum", "internal/acpfront/cancel_test.go", "internal/acpfront/core01_concurrency_test.go", "internal/acpfront/doc.go", "internal/acpfront/front.go", "internal/acpfront/harness_test.go", "internal/acpfront/initialize_test.go", "internal/acpfront/loop_test.go", "internal/acpfront/newsession_test.go", "internal/acpfront/stream.go", "internal/acpfront/stream_order_test.go", "internal/acpfront/stream_test.go", "internal/acpfront/turnctx.go", "internal/acpport/agent.go", "internal/acpport/agent_test.go", "internal/acpport/doc.go", "internal/acpport/pin_test.go", "internal/acpport/sink.go", "internal/acpport/types.go", "internal/core/registry.go", "internal/core/registry_test.go", "internal/supervisor/discover.go", "internal/supervisor/e2e_test.go", "internal/turn/machine.go", "internal/turn/machine_test.go", "internal/zcode/conn.go", "internal/zcode/conn_test.go", "internal/zcode/envelope.go", "internal/zcode/fake/fake.go", "internal/zcode/fake/golden_test.go", "internal/zcode/fake/roundtrip_test.go", "internal/zcode/fake/scenario.go", "internal/zcode/fake/stdio.go", "internal/zcode/fake/testdata/golden/cancel-mid-turn.golden", "internal/zcode/fake/testdata/golden/complete-then-cancel.golden", "internal/zcode/fake/testdata/golden/completion-quiet-window.golden", "internal/zcode/fake/testdata/golden/no-reasoning.golden", "internal/zcode/fake/testdata/golden/part-delta-streaming.golden", "internal/zcode/fake/testdata/golden/send-refused-32010.golden", "internal/zcode/fake/testdata/golden/setmodel-followup.golden", "internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json", "internal/zcode/fake/testdata/scenarios/complete-then-cancel.json", "internal/zcode/fake/testdata/scenarios/completion-quiet-window.json", "internal/zcode/fake/testdata/scenarios/no-reasoning.json", "internal/zcode/fake/testdata/scenarios/part-delta-streaming.json", "internal/zcode/fake/testdata/scenarios/send-refused-32010.json", "internal/zcode/fake/testdata/scenarios/setmodel-followup.json", "mise.toml"]
covered_digest: "v4:sha256:da0c51359ef872513c0de6a3a3df657ad98c4ab2a7c57c3733655e3ef0fac41a"
behavior_unverified: 0
overrides_applied: 0
unverified_prohibitions: 0
decision_coverage:
  honored: 5
  total: 5
  not_honored: []
re_verification:
  previous_status: passed
  previous_score: 10/11
  gaps_closed:
    - "CORE-01 concurrency-dispatch backstop abstention (insufficient_spec) — discharged by TestConcurrentRequestDispatch (internal/acpfront/core01_concurrency_test.go, commit 19b726b), re-run green under -race this verification"
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "Review findings WR-03, WR-04/WR-07, WR-05/WR-06, IN-03..IN-05 (buffered streaming design, error-path teardown, rejection wording / silent block drops, minor hardening) — routed open in 04-REVIEW-DISPOSITION.md"
    addressed_in: "Phase 5"
    evidence: "04-REVIEW-DISPOSITION.md routes all 12 (0 critical) to Phase 5 multi-session/teardown/permission scope; Phase 5 plans cite the ledger and 5 findings already landed (WR-01 router fan-out 04ef5c7, WR-02 atomic sink publish 31ea68e, WR-07/IN-01/IN-02 in 05-03 commits); Phase 5 goal/SC cover CORE-06, SESS-01, TOOL-01/02"
  - truth: "Non-SDK real-client integration (Orca chat UI driving the built binary)"
    addressed_in: "Phase 7"
    evidence: "04-UAT.md item 2 scope note: a NON-SDK real client is the Phase 7 manual-E2E done-criterion QA-03, recorded there; the SDK test client (the reference protocol client implementation) drove the built binary in the discharged spot check"
  - truth: "3 flagged probe assumptions (CORE-02/04/05 edge spaces 'unclassified' by the probe classifier)"
    addressed_in: "— (carried as flagged assumptions by the plans themselves)"
    evidence: "COVERAGE.md ledger equality: 10 = 7 resolved + 3 flagged; each flagged item names its enforcement tests (e.g., CORE-05 → TestCancelMidTurn/TestSecondPromptRejected/TestCancelAfterCompleteNoOp). Informational only"
---

# Phase 4: ACP Front — Minimal Chat Loop Verification Report

**Phase Goal:** As a v1 chat UI client, I want to open a session against ZCode and run a streamed prompt turn end-to-end, with live text and thought chunks and correct cancellation, against the app-server, so that the minimal chat loop is proven on the real ACP boundary before auth and interaction complexity accrete.
**Verified:** 2026-11-20T11:38:16Z
**Status:** passed
**Re-verification:** Yes — stale-covered-input regeneration (#4682 route). The prior verdict (passed after UAT discharge, 10/11) had gone stale because Phase 5 work landed on the same stacked branch, changing 14 files this verification covers. Must-haves re-checked against the current working tree as-is (including the one staged Phase-5 WIP change, internal/zcode/conn_test.go).
**Mode:** mvp (user-story goal validated via `user-story.validate` → true in the prior pass; goal unchanged)

## User Flow Coverage

User story: «As a v1 chat UI client, I want to open a session against ZCode and run a streamed prompt turn end-to-end, with live text and thought chunks and correct cancellation, against the app-server, so that the minimal chat loop is proven on the real ACP boundary before auth and interaction complexity accrete.»

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Client initializes | Honest reply: protocolVersion 1, real agentInfo, exactly two authMethods (zai-api-key, zai-individual-coding-plan), no fs/terminal/loadSession fields | `internal/acpfront/initialize_test.go` asserts the RAW response frame (byte-exact want + banned-substring loop); re-run green this pass | ✓ |
| Client opens a session | `session/new{cwd}` → backend session; create→subscribe→setModel visible on the wire with model+reasoningLevel; mcpServers accepted; relative cwd rejected pre-wire | `internal/acpfront/newsession_test.go` (TestNewSessionCreatesAndSetsModel + AcceptsMcpServers + RejectsRelativeCwd); golden `setmodel-followup.golden` byte-locks the payload; all re-run green | ✓ |
| Client prompts | Live `agent_message_chunk` deltas stream BEFORE the response frame; stopReason only after flush; chunks mirror envelopes 1:1 | `internal/acpfront/stream_order_test.go` + `stream_test.go` + `loop_test.go` — re-run green this pass (15.0s incl. both completion variants + zero-delta) | ✓ |
| Backend emits reasoning | Client receives `agent_thought_chunk` (one per reasoning delta, interleaved); zero reasoning → zero thought chunks | `internal/acpfront/stream.go:59-64` field switch (text→NoteMessage, reasoning→NoteThought); TestThoughtChunks positive+negative green | ✓ |
| Client cancels mid-turn | Exactly ONE prompt response, `{"stopReason":"cancelled"}` as SUCCESS, no -32800, no second concurrent turn | `internal/acpfront/cancel_test.go` (TestCancelMidTurn / TestSecondPromptRejected / TestCancelAfterCompleteNoOp) — re-run green; front.go's cancel now also fans in the Phase-5 permission bridge (`f.perm.cancelPendingForTurn()`, additive to CORE-05 semantics) | ✓ |
| Outcome: minimal loop proven on the real ACP boundary | The built binary carries initialize → session/new → streamed prompt → cancel over REAL exec pipes in CI | `cmd/zcode-acp/main_test.go` TestAgentBinaryEndToEnd — re-run green this pass (main.go's only change since the prior pass is the 3-value NewAgentSideConnection call shape, `agentConn, _, _` — wiring intact) | ✓ |

The `[outcome]` clause is observably true in the codebase at the CI-provable scope. The non-SDK real-client step (Orca chat UI) is recorded at Phase 7 QA-03 per the discharged UAT scope note (04-UAT.md item 2: pass).

## Goal Achievement

### Observable Truths

Merged from ROADMAP.md Success Criteria (contract) + the six PLAN frontmatter must_haves, deduplicated — identical set to the prior pass. Score: 11/11; the prior backstop abstention now has behavioral evidence.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `initialize` replies protocolVersion 1 with an honest capability set (nothing fs/terminal-related), real agentInfo, both authMethods | ✓ VERIFIED | TestInitializeHonestCapabilities re-run green on this tree; adapter assembles the honest shape (`internal/acpfront/front.go:133` Initialize, `internal/acpport/agent.go`); agent.go's diff since the prior pass is additive (PermissionSender), initialize untouched |
| 2 | Advertised surface exactly equals implemented surface: the 8 unadvertised stable methods answer -32601; authenticate answers an honest not-available error, never fake success | ✓ VERIFIED | TestUnadvertisedStableMethodsAnswerNotFound + TestAuthenticateHonestError re-run green; honest-error message intact at `internal/acpport/agent.go:290` ("authentication is not available in this build yet"), NewMethodNotFound default at `:402` |
| 3 | `session/new{cwd}` creates a backend session; the setModel follow-up carrying model + reasoningLevel is visible on the wire (fake transcript asserts the payload); create stays bare | ✓ VERIFIED | TestNewSession* re-run green (0.3s); relative-cwd guard intact at `front.go:163-166` (pre-wire -32602); registry.Register updated for the slimmed Session struct (dead `Subscribed` field removed by 05-03 IN-02 — Register/Get/Remove contract unchanged, TestRegistryConcurrentAccess green) |
| 4 | `mcpServers` accepted without error and never forwarded; a RELATIVE cwd is rejected (-32602) BEFORE any backend call | ✓ VERIFIED | TestNewSessionAcceptsMcpServers + TestNewSessionRejectsRelativeCwd green this pass; guard verified first-hand at `front.go:163` |
| 5 | `session/prompt` streams agent_message_chunk deltas live (1:1 envelope mirror) and returns stopReason only after all pending updates flush — across both completion variants; zero-delta turn still answers end_turn | ✓ VERIFIED | TestStreamBeforeStopReason (terminal-event + quiet-window + zero-delta) + TestThoughtChunks re-run green (15.0s) |
| 6 | Backend reasoning deltas → agent_thought_chunk (exactly one per delta, in arrival order); zero reasoning → ZERO thought chunks (never synthesized) | ✓ VERIFIED | TestThoughtChunks positive+negative green; mapping intact at `stream.go:59-64` (grep first-hand) |
| 7 | `session/cancel` mid-turn answers exactly once with stopReason "cancelled" as SUCCESS, never -32800; a second active prompt answers ONE clean invalid-request rejection (both guard layers); cancel-after-complete is silent | ✓ VERIFIED | TestCancelMidTurn / TestSecondPromptRejected / TestCancelAfterCompleteNoOp re-run green (0.7s); front.go's only handler change is the additive permission-bridge cancel fan-in at `front.go:305` — CORE-05 semantics preserved and extended (TOOL-02) |
| 8 | go.mod requires exactly one dependency — acp-go-sdk v0.20.0 exact pin — gate-locked; no non-test file outside internal/acpport imports the SDK | ✓ VERIFIED | `grep -c '^require' go.mod` = 1 (v0.20.0); `mise run tidy-diff` exit 0 this pass; independent grep: only internal/acpport/{sink,agent}.go import the SDK in non-test files — Phase 5's new packages (internal/interact, acpfront/permission.go) are SDK-free; TestSDKImportBoundary green |
| 9 | Ordered notification observer fan-out in zcode.Conn: registration-order synchronous delivery, remove deregisters, two machines on one conn each resolve interleaved turns exactly once (no hook theft) | ✓ VERIFIED | AddObserver mechanism unchanged by Phase 5's WR-01 fix (that fix replaced the machine-side single OnRouterEvent slot with AddRouterObserver fan-out — conn.go diff examined: AddObserver's entry/mutex/ordering intact, +ObserverCounts diagnostic); tap-before-machines wiring intact at `front.go:122`; TestObserverFanOut re-run green under -race |
| 10 | The built zcode-acp binary runs the full loop over real stdio (fake backend via the discovery override seam); --version unchanged; stdout protocol-exclusive | ✓ VERIFIED | TestAgentBinaryEndToEnd re-run green (1.75s) on the current main.go (single-line diff: NewAgentSideConnection 3-value call shape); production composition unchanged otherwise |
| 11 | BACKSTOP (04-03 `verification: backstop`): "concurrent ACP requests on one connection are serialized by the SDK dispatcher; front session registry is mutex-guarded" | ✓ VERIFIED | Discharged with machine evidence: TestConcurrentRequestDispatch (`internal/acpfront/concurrency_test.go` → core01_concurrency_test.go, commit 19b726b) drives session/new + initialize concurrently over ONE connection (overlap sequencer: initialize issued only after session/new's request frame is provably on the tee) and asserts 3 layers — typed caller identities, concurrent-session registry resolution via a real end_turn prompt, raw-wire 3-request/1-response pairing with each response carrying its OWN method's result shape; re-run green by this verifier under -race (1.0s); UAT item 1 recorded the -race -count=3 discharge |

**Score:** 11/11 truths verified (0 present-behavior-unverified; the prior backstop abstention upgraded to VERIFIED on a passing wired behavioral test)

### Deferred Items

Items not yet met but explicitly addressed in later milestone phases (informational only — no status impact).

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Remaining routed review findings (WR-03 buffered streaming design, WR-04/WR-07 error-path teardown, WR-05/WR-06 wording + silent block drops, IN-03..IN-05) | Phase 5 | `04-REVIEW-DISPOSITION.md` ledger (all 12 routed open, 0 critical); Phase 5 is actively honoring the routing — WR-01, WR-02, WR-07, IN-01, IN-02 already landed in 05-01..05-03 commits |
| 2 | Non-SDK real-client integration (Orca chat UI) | Phase 7 | 04-UAT.md item 2 scope note → QA-03 manual E2E done-criterion |
| 3 | 3 flagged probe assumptions (CORE-02/04/05 edge spaces) | — (flagged by the plans themselves) | COVERAGE.md ledger: 10 = 7 resolved + 3 flagged, each with named enforcement tests |

### Advisory (New Scope, Unevidenced)

None — the Step 7 scan on this re-verification pass found no new-scope concerns (debt-marker gate clean, placeholder/stub scan clean on all phase production files).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` / `go.sum` | Single approved dependency, exact v0.20.0 pin | ✓ VERIFIED | 1 require line; tidy-diff gate exit 0 this pass |
| `mise.toml` | tidy-diff gate asserts the dependency expectation | ✓ VERIFIED | count==1 + fixed-string pin (gate re-run green) |
| `internal/zcode/conn.go` | AddObserver ordered fan-out; Phase-5 adds router fan-out + removes | ✓ VERIFIED | AddObserver intact (diff examined); AddRouterObserver added (WR-01); ObserverCounts diagnostic |
| `internal/zcode/envelope.go` | MethodSessionSubscribe / MethodSessionSetModel constants | ✓ VERIFIED | :149-150 confirmed first-hand; Phase-5 added interaction client-request codes (additive) |
| `internal/turn/machine.go` | NewMachine registers via conn.AddObserver | ✓ VERIFIED | `machine.go:199` first-hand; SetOnRouterEvent superseded by AddRouterObserver (WR-01, preserves no-theft contract) |
| `internal/acpport/types.go` | Neutral Handlers vocabulary + error helpers | ✓ VERIFIED | +52 lines PermissionSender vocabulary (additive); Handlers/DTOs/StopReasons intact |
| `internal/acpport/agent.go` | acp.Agent adapter: honest initialize, delegation, -32601 defaults, honest authenticate | ✓ VERIFIED | Diff examined: atomic sink publish (WR-02) + PermissionSender added; honesty surface untouched (`:290` honest auth error, `:402` -32601) |
| `internal/acpport/sink.go` | UpdateSink → agent_message_chunk / agent_thought_chunk | ✓ VERIFIED | NoteMessage/NoteThought mapping intact (stream.go consumers green) |
| `internal/acpport/agent_test.go` | TestSDKImportBoundary + adapter contract tests | ✓ VERIFIED | Re-run green |
| `internal/core/registry.go` + test | SessionRegistry (mutex, loud miss) | ✓ VERIFIED | Slimmed by IN-02 (dead Subscribed field removed); Register/Get/Remove contract green incl. concurrency test |
| `internal/acpfront/front.go` | Front: honest init, D-03 session/new, D-04/D-05 prompt/cancel | ✓ VERIFIED | Diff examined: +perm bridge field +cancel fan-in only; all four handlers and the cwd guard intact |
| `internal/acpfront/stream.go` | Delta tap: text→message, reasoning→thought, input/output skipped | ✓ VERIFIED | Field switch at :59-64 confirmed first-hand |
| `internal/acpfront/turnctx.go` | Front-owned turn ctx + slot guard + cancel fan-in | ✓ VERIFIED | acquireTurn/release intact |
| `internal/acpfront/*_test.go` (8 files) | Named contract locks + tracer + harness + concurrency backstop | ✓ VERIFIED | All compile and pass on this tree (7 re-run by name this pass; full suite green this session across all 10 packages, exit 0) |
| `cmd/zcode-acp/main.go` + `main_test.go` | First real wiring + binary E2E | ✓ VERIFIED | One-line diff (3-value NewAgentSideConnection call); E2E green |
| fake corpus (Phase-4 rows: 7 scenarios + 7 goldens + drivers) | Byte-deterministic, fail-loud | ✓ VERIFIED | TestGolden/setmodel-followup green; corpus infra (fake.go/scenario.go) changed by Phase 5 but Phase-4 rows replay identically |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| internal/acpfront/front.go | internal/zcode/conn.go | tap registered via conn.AddObserver BEFORE any machine (registration order = delivery order) | WIRED | `front.go:122` in New() confirmed first-hand; per-session machines land after the tap |
| internal/acpport/agent.go | internal/acpport/types.go | agentAdapter translates acp.Agent ⇄ Handlers neutral DTOs | WIRED | NewAgent(h Handlers) intact; PermissionBinder opt-in added (additive); front still imports acpport not the SDK |
| internal/acpfront/turnctx.go | internal/turn/machine.go | Machine.Run on front-owned ctx; sdkCtx.Done fans in to machine.Cancel | WIRED | TestCancelMidTurn green (cancel path exercised); acquireTurn intact |
| internal/zcode/fake/golden_test.go | testdata/scenarios/setmodel-followup.json | driveSetModelFollowup replays create→subscribe→setModel→send through the real conn | WIRED | TestGolden/setmodel-followup green this pass |
| cmd/zcode-acp/main.go | internal/supervisor/discover.go | Override seam (--app-server / ZCODE_ACP_APP_SERVER) | WIRED | TestAgentBinaryEndToEnd green (drives the production discovery/spawn argv) |
| cmd/zcode-acp/main.go | internal/acpport/agent.go | acpport connection over os.Stdout/os.Stdin; version var feeds agentInfo | WIRED | Updated to the 3-value NewAgentSideConnection shape; E2E green incl. stdout protocol-exclusivity |
| internal/acpfront/front.go | fake corpus setmodel-followup | session/new transcript assertions ride the corpus row | WIRED | Same composition green under the Phase-5 harness scripting changes |

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

All commands run first-hand this pass on the current tree (not inherited from SUMMARYs or the prior report). Exit 0 = ok.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| CORE-01 backstop discharge (new) | `go test -race -count=1 -run TestConcurrentRequestDispatch ./internal/acpfront/` | ok 1.0s | ✓ PASS |
| Initialize honesty | `go test -count=1 -run TestInitializeHonestCapabilities ./internal/acpfront/` | ok | ✓ PASS |
| acpport honesty locks | `go test -count=1 -run 'TestAuthenticateHonestError\|TestUnadvertisedStableMethodsAnswerNotFound' ./internal/acpport/` | ok | ✓ PASS |
| session/new locks | `go test -count=1 -run TestNewSession ./internal/acpfront/` | ok 0.3s | ✓ PASS |
| Streaming flush + thought | `go test -count=1 -run 'TestStreamBeforeStopReason\|TestThoughtChunks' ./internal/acpfront/` | ok 15.0s | ✓ PASS |
| Cancel locks | `go test -count=1 -run 'TestCancelMidTurn\|TestSecondPromptRejected\|TestCancelAfterCompleteNoOp' ./internal/acpfront/` | ok 0.7s | ✓ PASS |
| Tracer loop | `go test -count=1 -run TestMinimalChatLoop ./internal/acpfront/` | ok | ✓ PASS |
| Binary E2E | `go test -count=1 -run TestAgentBinaryEndToEnd ./cmd/zcode-acp/` | ok 1.8s | ✓ PASS |
| Observer fan-out | `go test -race -count=1 -run TestObserverFanOut ./internal/zcode/` | ok 1.3s | ✓ PASS |
| Registry concurrency | `go test -race -count=1 -run TestRegistryConcurrentAccess ./internal/core/` | ok 1.0s | ✓ PASS |
| Golden setmodel-followup | `go test -count=1 -run TestGolden/setmodel-followup ./internal/zcode/fake/` | ok | ✓ PASS |
| SDK boundary | `go test -count=1 -run TestSDKImportBoundary ./internal/acpport/` + independent grep | ok / only acpport non-test files import SDK | ✓ PASS |
| Full workspace suite (once, this session, same tree — cited per runtime notes, not re-run) | full `go test -race ./...` | all 10 packages exit 0 | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| (none declared) | `find scripts experiments -path '*tests/probe-*.sh'` (prior pass; no probe scripts exist in the repo) | no probe scripts; the plans' "probe-ledger" is the specless probe-fallback ledger authored into PLAN must_haves, not shell probes | SKIPPED — no probes to run |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| CORE-01 | 04-01..04-06 (all declare) | initialize replies protocolVersion 1, honest capability set, real agentInfo, two authMethods | ✓ SATISFIED | Truths 1-2 (+ concurrency backstop truth 11, now test-discharged); REQUIREMENTS.md marks Complete |
| CORE-02 | 04-01..04-06 | session/new → create + setModel follow-up (reasoningLevel workaround); mcpServers accepted and ignored | ✓ SATISFIED | Truths 3-4 + golden byte-lock; REQUIREMENTS.md marks Complete |
| CORE-03 | 04-01, 04-02, 04-04, 04-05, 04-06 | prompt streams agent_message_chunk live; stopReason only after flush | ✓ SATISFIED | Truth 5; REQUIREMENTS.md marks Complete |
| CORE-04 | 04-01, 04-02, 04-04, 04-05, 04-06 | agent_thought_chunk streamed when the backend emits reasoning; never synthesized | ✓ SATISFIED | Truth 6 (positive+negative); REQUIREMENTS.md marks Complete |
| CORE-05 | 04-02, 04-04, 04-05, 04-06 | session/cancel stops the turn; stopReason "cancelled" (exactly once; never -32800) | ✓ SATISFIED | Truths 7 + 10; REQUIREMENTS.md marks Complete |

Orphan check: REQUIREMENTS.md maps exactly CORE-01..05 to Phase 4; every plan's frontmatter declares only these five (04-01/02/06: all five; 04-03: CORE-01/02; 04-04: CORE-03/04; 04-05: CORE-05); no Phase-4-mapped requirement is missing from any plan's `requirements` field. Coverage complete.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/acpfront/initialize_test.go | CORE-01 | yes | none | no | Behavioral (raw-frame + absence) | PASS |
| internal/acpport/agent_test.go | CORE-01, seam | yes | none | no | Behavioral (wire codes, shapes) | PASS |
| internal/acpfront/newsession_test.go | CORE-02 | yes | none | no (transcript asserts independent of golden) | Behavioral (wire order + payload) | PASS |
| internal/acpfront/stream_test.go / stream_order_test.go | CORE-03/04 | yes | none | no (raw tee line indexes) | Behavioral (ordering, counts) | PASS |
| internal/acpfront/cancel_test.go | CORE-05 | yes | none | no | Behavioral (exactly-once, code absence) | PASS |
| internal/acpfront/loop_test.go | CORE-01..05 | yes | none | no (staged transcript asserts) | Behavioral (end-to-end) | PASS |
| internal/acpfront/core01_concurrency_test.go | CORE-01 backstop | yes (new since prior pass) | none | no (3 independent assertion layers: typed / registry / raw-wire pairing) | Behavioral (concurrent dispatch serialization) | PASS |
| cmd/zcode-acp/main_test.go | CORE-05 (exec leg) | yes | none | no (fake's own transcript file) | Behavioral (binary E2E) | PASS |
| internal/zcode/conn_test.go (TestObserverFanOut + Phase-5 WIP staged change) | infra enabler | yes | none | no (signal assertion defeats quiet-window masking) | Behavioral (ordering/remove/theft) | PASS |
| internal/zcode/fake/golden_test.go (TestGolden corpus) | CORE-02/03/04/05 fixtures | yes | none | pinned-snapshot regression locks; contract correctness asserted independently by the front-tier want-strings | Behavioral (byte-determinism) | PASS |
| internal/supervisor/realdesktop_test.go | Phase-3 manual-only | env-gated | t.Skip (documented manual pattern, pre-existing, unrelated to CORE-01..05) | n/a | n/a | INFO |

**Disabled tests on requirements:** 0 (grep for t.Skip across all phase-4 requirement test files: no matches; the only t.Skip in the repo is the Phase-3 env-gated real-desktop manual test)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | TBD/FIXME/XXX debt markers in phase-modified production files | none | grep clean this pass |
| internal/acpport/agent.go | 290 | "not available" phrasing | ℹ️ Info | This IS the honest authenticate error message — intended behavior, not a placeholder |
| — | — | Empty implementations / console-only handlers / hardcoded empty data flowing to output / placeholders | none | grep clean across all phase production files this pass |

🛑 Blockers: 0. The debt-marker gate found nothing.

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (`check.decision-coverage-verify` re-run this pass: total 5, honored 5, not_honored [] — D-01 SDK audit choice, D-02 honesty, D-03 session/new contract, D-04 streaming, D-05 cancel semantics.)

### Security Review

`04-SECURITY.md`: all 6 threats dispositioned closed (T-4-SC pin discipline re-verified this pass via tidy-diff exit 0 + single-require grep; T-4-04 honesty re-verified via the honesty locks re-run). 2 accepted risks documented (structural queue caps; override-seam trust position).

### Code Review

`04-REVIEW.md` + `04-REVIEW-DISPOSITION.md`: 0 critical, 12 routed findings — Phase 5 is honoring the routing obligation (WR-01 router fan-out, WR-02 atomic sink publish, WR-07/IN-01/IN-02 already landed in 05-01..05-03 commits; remaining routed to Phase 5 plans 05-04..05-09). Documented routed items within the phase contract, not gaps.

### Human Verification Required

None — all items from the prior pass are discharged in the completed UAT (04-UAT.md: 3/3 passed, 0 pending, 0 issues):

1. **CORE-01 concurrency backstop** — DISCHARGED: TestConcurrentRequestDispatch written and passing (`-race -count=3` at UAT time; re-run green by this verifier), commit 19b726b.
2. **Real-client spot check** — DISCHARGED: TestAgentBinaryEndToEnd drives the built binary over real exec pipes under the SDK's ClientSideConnection (the reference protocol client implementation); the non-SDK real client (Orca) is recorded as Phase 7 QA-03.
3. **Five judgment-tier prohibitions** — DISCHARGED (user-approved per standing autonomous directive, with the verifier's evidence cited): advertise=implemented (raw-frame absence locks), no fake auth success (TestAuthenticateHonestError), never synthesize chunks (TestThoughtChunks/negative + zero-delta), stdout protocol-exclusive (E2E parse cleanliness + --version unchanged), no paths/prompt content in error surfaces. Recorded nuance: generic `NewInternal("%v")` passes underlying transport-error text through un-pinned — routed to Phase 5 error-hygiene with the WR-05/WR-06 family.

### Gaps Summary

No gaps. All 11 merged must-haves (5 ROADMAP Success Criteria + PLAN truths, deduplicated) are delivered and locked by named tests re-run green first-hand on the current tree — including the binary-boundary E2E, the byte-locked setModel payload, the flush-before-stopReason ordering across both completion variants, the cancel exactly-once contract with the -32800 absence scan, and (new since the prior pass) the CORE-01 concurrency-dispatch backstop with its 3-layer serialization proof.

The Phase 5 work that made the prior digest stale was examined at diff level: every change to a phase-4 load-bearing file is additive or a contract-preserving refactor (router-event fan-out replacing the single machine hook, atomic sink publish, permission bridge + cancel fan-in, slimmed registry struct, 3-value connection constructor) — no phase-4 contract regressed, confirmed by the targeted re-runs above and the full-suite green run on this exact tree.

Status `passed` is valid: no failed truths, no missing/stub artifacts, no unwired links, no blocker anti-patterns, and the human-verification section is empty (all prior items discharged in the completed UAT; the one backstop abstention upgraded to VERIFIED on passing behavioral evidence).

---

_Verified: 2026-11-20T11:38:16Z_
_Verifier: Claude (gsd-verifier)_
