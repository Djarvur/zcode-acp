---
phase: 04-acp-front-minimal-chat-loop
plan: 05
subsystem: acp-front
tags: [acp, cancel-contract, exactly-once, rejection-surface, cmd-wiring, supervisor, e2e, exec-pipes, binary, fake-corpus, tdd, red-green]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: "plan 04-06: the tracer Front + front-tier harness (newTracerFront, the extended cancel-mid-turn corpus); plans 04-02/04-03/04-04: the acpport seam vocabulary (NewInvalidRequest/NewInternal/NewInvalidParams), the corpus convention (front preamble verbatim in every front-tier row), the named contract-lock tier this plan sharpens"
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: "the supervisor lifecycle this wiring composes (Probers.Override order-step-1 seam, NewSupervisor/SpawnFunc, SpawnChild + ChildEnv, the staged shutdown), cmd/fake-app-server, and the 03-09 exec-chain precedents (built-binary caching, Machine-before-frames)"
provides:
  - "CORE-05 locked as named tests: TestCancelMidTurn (cancel mid-turn resolves EXACTLY ONE success response {\"stopReason\":\"cancelled\"}, no -32800 frame anywhere on the ACP tee, updates precede the response, exactly one session/send), TestSecondPromptRejected (ONE invalid-request rejection surface for concurrent prompts — both guard layers, stable message verbatim), TestCancelAfterCompleteNoOp (cancel-after-complete: zero new agent→client frames, no session/stop on the back leg)"
  - "the acpfront invalid-request rejection surface: front.go maps the machine's ErrTurnActive onto the SAME acpport.NewInvalidRequest the front-wide slot guard answers — one -32600 class carrying turnActiveMessage ('a prompt is already running for this session') whatever guard layer fires; the machine's numeric -32010 CodeConcurrency match untouched (message text never matched); the wrapped transport noise never reaches the wire"
  - "cmd/zcode-acp first real wiring: --app-server/--model/--reasoning-level flags (+ZCODE_ACP_APP_SERVER/ZCODE_ACP_MODEL/ZCODE_ACP_REASONING_LEVEL env fallbacks, flags win), supervisor discovery through the order-step-1 Override seam, supervisor Start over the SpawnFunc seam (SpawnArgv + ChildEnv), zcode.Conn over the child exec pipes, acpport agent connection over os.Stdout/os.Stdin, staged shutdown with a dedicated spawn ctx (see Deviations), --version path unchanged, stdout protocol-exclusive"
  - "TestAgentBinaryEndToEnd — the binary-level proof: the BUILT zcode-acp binary carries initialize (agentInfo.version = the binary's version var, D-02) → session/new (setModel payload asserted on the FAKE's own transcript) → loop 1 end_turn with streamed deltas → loop 2 cancelled exactly once → clean staged shutdown exit 0, over REAL exec pipes, no desktop, green in the default suite under -race"
  - "two new corpus rows (13 → 15): send-refused-32010 (front preamble + the dialect §6 -32010 send refusal — the backend-refusal RED leg) and complete-then-cancel (loop-1 send completes times:1, loop-2 send stays active until the stop's cancelled patch — the binary E2E's two loops), each with driver + byte-deterministic golden"
  - "fake binary --transcript (fake.MainTo): the standalone hosting mode can dump the served transcript (golden dir\\tline format) after its clean EOF-drain exit — the exec tier's window into its own wire"
  - "supervisor.ExplicitRootInstall: the exported order-step-1 override Install shape (invariant spawn-surface validation, layout 'explicit') — cmd builds the canonical Install without duplicating unexported matrix constants"
affects: [phase-5-chat-cycle, phase-6-auth, phase-7-qa, /gsd:verify-work]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 19198 # chars/4 over the realized diff (76793 diff chars, 1757 insertions / 14 deletions across 14 files)
  tasks: 2
  commits: 3 # MEASURED: git rev-list --count 5ecb897..HEAD

commits: 3
plan_head_before: 5ecb897a8de955c36a42037cadeb1e7b389209f0
plan_head_after: 03534ac84c6d72015f46f5be80b1c48632d739ae

# Tech tracking
tech-stack:
  added: [] # zero new dependencies — acp-go-sdk v0.20.0 stays the module's sole require line
  patterns:
    - "Contract-lock plans carry their own corpus (04-04 precedent extended): the CORE-05 locks needed scenarios the corpus didn't have — the backend-refusal RED leg and the binary E2E's two-loop shape — so the rows landed before the rides with drivers and goldens, corpus 13 → 15, all replayable at both tiers"
    - "Two-lifecycle shutdown composition: the supervision ctx drives Start (and its staged stdin→grace→kill teardown) while the spawn ctx is the exec-level kill backstop cancelled only AFTER the teardown completes — a shared cancellation races SpawnChild's cmd.Cancel group SIGKILL against the graceful stdin-EOF drain and the child always dies mid-flight"
    - "The fake-scenario install shape: the Override seam accepts a bundle-shaped root whose 'zcode' binary is a sh shim exec-ing the built fake with --scenario/--transcript baked in — the production discovery order and the canonical SpawnArgv line run unchanged against it (the override seam stays a pure operator/test flag, T-4-04)"
    - "Preamble dedup by corpus convention held through two new rows: driveFrontPreamble stays the ONE source of the identical front preamble (six rows now drive it), goldens byte-stable"

key-files:
  created:
    - internal/acpfront/cancel_test.go
    - cmd/zcode-acp/main_test.go
    - internal/zcode/fake/testdata/scenarios/send-refused-32010.json
    - internal/zcode/fake/testdata/golden/send-refused-32010.golden
    - internal/zcode/fake/testdata/scenarios/complete-then-cancel.json
    - internal/zcode/fake/testdata/golden/complete-then-cancel.golden
    - .planning/phases/04-acp-front-minimal-chat-loop/evidence/04-05-red-second-prompt.json
  modified:
    - internal/acpfront/front.go
    - cmd/zcode-acp/main.go
    - cmd/fake-app-server/main.go
    - internal/zcode/fake/stdio.go
    - internal/zcode/fake/golden_test.go
    - internal/zcode/fake/roundtrip_test.go
    - internal/supervisor/discover.go

key-decisions:
  - "The rejection surface is ONE constructor + ONE message for BOTH guard layers (front-wide slot and machine ErrTurnActive): the client can never tell which layer fired, and the wrapped transport noise (the -32010 wire error verbatim) never reaches the ACP wire"
  - "The honest RED record: the plan text expected the front-slot conflict to surface as the internal class, but 04-06 had already mapped the slot to NewInvalidRequest — the genuine -32603 surface was the machine's ErrTurnActive branch (the backend -32010 refusal), so the RED rides a NEW send-refused-32010 corpus row while the slot leg and the two cancel locks are recorded as honest green-first-run locks (never manufactured RED)"
  - "The spawn ctx is deliberately NOT the supervision ctx (Rule 1): SpawnChild wires cmd.Cancel to the group SIGKILL on its ctx, so cancelling both together SIGKILLs the child the instant the teardown begins — the graceful stdin-EOF drain (the child's ~100 ms clean exit, dialect §1) would never run; spawnCancel now fires only after startDone"
  - "The binary E2E syncs loop-2's cancel on the STREAMED DELTA ('slow'), not on a sleep: the tap delivering loop-2's chunk across the real binary boundary is the wire-level proof the turn is in flight, so the cancel is mid-turn by construction"
  - "The --model flag parses providerId/modelId (single '/'); --reasoning-level without --model is a usage error (exit 2), and an empty level rides model.options.reasoningLevel verbatim — the same payload shape the corpus goldens byte-lock"

patterns-established:
  - "Fake-scenario install root (shim 'zcode' + stub zcode.cjs): the reusable shape for every future binary-tier test and the phase-end real-client spot check — fabricate the bundle-shaped root, point --app-server at it, the production path does the rest"
  - "Transcript-file golden format at the exec tier: fake.MainTo writes the SAME dir\\tjson records the goldens use, so every tier reads one transcript shape"

requirements-completed: [CORE-05] # copied verbatim from plan frontmatter; CORE-05 is declared by 04-02/04-04/04-05 — requirements.ready-ids governs the mark-complete step (all declarers have SUMMARYs after this plan, so it completes)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "CORE-05 exactly-once cancel lock: cancel mid-turn resolves the prompt EXACTLY once with stopReason cancelled as a SUCCESS response, zero -32800 frames in either tee direction, zero updates after the response frame, exactly one session/send on the back leg"
    requirement: CORE-05
    verification:
      - kind: integration
        ref: "internal/acpfront/cancel_test.go#TestCancelMidTurn (composed loop over cancel-mid-turn, -race; go test -race -count=1 -run TestCancelMidTurn ./internal/acpfront/)"
        status: pass
    human_judgment: false
  - id: D2
    description: "CORE-05 one rejection surface: a second prompt answers the invalid-request class (-32600) with the stable message verbatim from BOTH guard layers — the front-wide slot (second prompt mid-turn; exactly one session/send total) and the machine's ErrTurnActive (backend -32010 refusal over send-refused-32010); cancel-after-complete silent (zero new frames, no session/stop)"
    requirement: CORE-05
    verification:
      - kind: integration
        ref: "internal/acpfront/cancel_test.go#TestSecondPromptRejected/{front_turn_slot_held,backend_-32010_refusal_mapped} + #TestCancelAfterCompleteNoOp (-race; go test -race -count=1 -run 'TestSecondPromptRejected|TestCancelAfterCompleteNoOp' ./internal/acpfront/)"
        status: pass
    human_judgment: false
  - id: D3
    description: "cmd/zcode-acp first real wiring: flags/env configuration (flags win), discovery through the order-step-1 Override seam (ExplicitRootInstall shape), supervisor Start over SpawnChild (SpawnArgv + ChildEnv), front+tap registered before any frame, acpport agent connection over os.Stdout/os.Stdin, two-lifecycle staged shutdown; --version unchanged (zcode-acp <version>); stdout protocol-exclusive (the SDK client parses it by construction)"
    requirement: CORE-05
    verification:
      - kind: integration
        ref: "cmd/zcode-acp/main_test.go#TestAgentBinaryEndToEnd/{version_flag_unchanged,two_prompt_loops_over_real_stdio} (-race; go test -race -count=1 -run TestAgentBinaryEndToEnd ./cmd/zcode-acp/)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The binary-level loop: built zcode-acp spawns the built fake through the override install, initialize asserts agentInfo.version == the version var (D-02), session/new rides --model/--reasoning-level (setModel payload verbatim on the fake's own transcript file — providerId/modelId/options.reasoningLevel + persistAsWorkspaceLastUsed:false), loop 1 end_turn, loop 2 cancelled exactly once, wrapper exits 0 on client disconnect with the graceful staged teardown"
    requirement: CORE-05
    verification:
      - kind: integration
        ref: "cmd/zcode-acp/main_test.go#TestAgentBinaryEndToEnd/two_prompt_loops_over_real_stdio (e2eAssertFakeTranscript + e2eAssertSetModelPayload, -race, green at -count=2)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Corpus grows 13 → 15 rows, both new rows replay byte-deterministically through the real conn: send-refused-32010 (preamble + dialect §6 -32010 send refusal) and complete-then-cancel (loop-1 completes times:1, loop-2 active until the stop's cancelled patch); drivers registered (driver-less fails loud); fake binary --transcript dump; TestGolden green across the whole corpus"
    requirement: CORE-05
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden (15 rows, -race; go test -race -count=1 -run TestGolden ./internal/zcode/fake/)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Zero regressions: full gate green — build, vet, lint 0 issues (all 28 findings on the new files fixed in code), full -race suite (8 packages incl. cmd/zcode-acp), experiments suite — the phase-gate rehearsal"
    verification:
      - kind: integration
        ref: "mise run ci (build + vet + lint + go test -race -count=1 ./... + node --test experiments/)"
        status: pass
    human_judgment: false

# Metrics
duration: 44min
completed: 2026-11-20
status: complete
---

# Phase 4 Plan 05: CORE-05 Cancel Locks + cmd/zcode-acp First Real Wiring Summary

**The minimal loop closes on the real binary boundary: CORE-05 locks land as named tests (cancelled exactly once and never -32800; concurrent prompts answered by ONE -32600 surface with a stable message from both guard layers, the -32010 refusal RED→GREEN'd through a new corpus row; cancel-after-complete silent), and the built zcode-acp binary runs initialize → session/new (setModel asserted on the fake's own transcript) → streamed end_turn loop → mid-turn cancelled loop → clean staged shutdown over real exec pipes — mise run ci fully green, the phase ready for /gsd:verify-work**

## Performance

- **Duration:** 44 min
- **Started:** 2026-11-20T01:42:14Z
- **Completed:** 2026-11-20T02:26Z
- **Tasks:** 2 (Task 1 via RED→GREEN TDD with classified evidence; Task 2 implementation + gate)
- **Files:** 7 created / 7 modified

## Accomplishments
- `TestCancelMidTurn` locks the exactly-once contract over the extended cancel-mid-turn corpus: exactly ONE prompt response on the tee carrying `{"stopReason":"cancelled"}` as a SUCCESS, zero -32800 frames in either direction (the SDK's ctx.Canceled→-32800 trap stays closed), zero update frames after the response (the spec ordering rule), exactly one `session/send` on the back leg
- `TestSecondPromptRejected` locks the ONE invalid-request rejection surface through BOTH guard layers: the front-wide slot (second prompt while the turn is held — exactly one session/send, the displaced prompt still resolving cancelled) and the machine's ErrTurnActive (the dialect §6 -32010 send refusal over the new send-refused-32010 corpus row) — both answer -32600 with `a prompt is already running for this session` verbatim; the wrapped transport noise never reaches the wire
- `TestCancelAfterCompleteNoOp` locks the silence: a complete prompt (end_turn), then a cancel — zero new agent→client frames, no error frame, and no `session/stop` on the back leg (the machine's idempotent Cancel writes nothing once resolved)
- front.go GREEN: the machine-error branch maps `turn.ErrTurnActive` onto the same `acpport.NewInvalidRequest` the slot guard answers, message named once (`turnActiveMessage`); the machine's numeric -32010 mapping is untouched
- cmd/zcode-acp carries its first real wiring behind the unchanged `--version` branch: flags + ZCODE_ACP_* env fallbacks (flags win), discovery via the order-step-1 Override seam, supervisor Start over the SpawnFunc seam (SpawnArgv + ChildEnv — the proxy scrub inherited), zcode.Conn over the child pipes, the Front (tap registered before any frame) wrapped via acpport over os.Stdout/os.Stdin, staged shutdown; stderr slog only, stdout protocol-exclusive
- `TestAgentBinaryEndToEnd` proves the built binary over REAL exec pipes: both binaries built once per run (03-09 precedent), the fake wired through a shim "install root" the Override accepts, initialize asserting `agentInfo.version` = the version var (D-02), the setModel payload asserted on the FAKE'S OWN TRANSCRIPT (zai-api/GLM-5.3-Flash/high + persistAsWorkspaceLastUsed:false), loop 1 end_turn with the corpus deltas, loop 2 cancelled exactly once (synced on the streamed "slow" delta), clean exit 0 — green under -race and at -count=2
- Corpus grows 13 → 15 (send-refused-32010, complete-then-cancel) with drivers and byte-locked goldens; the fake binary gained `--transcript` (MainTo) so the standalone hosting mode can dump what it actually served; supervisor gained `ExplicitRootInstall` for the override Install shape
- `mise run ci` fully green: build → vet → lint 0 issues (all 28 findings on the new code fixed in code) → full -race suite (8 packages) → experiments — the phase-gate rehearsal passed

## Task Commits

Each task committed atomically (TDD: RED commit precedes GREEN per feature):

1. **Task 1 (RED): CORE-05 cancel locks + failing second-prompt rejection** - `ac993a1` (test)
2. **Task 1 (GREEN): map the machine's ErrTurnActive onto the one invalid-request rejection** - `0d53155` (feat)
3. **Task 2: cmd/zcode-acp wiring + binary-level E2E + corpus/fake/supervisor seams** - `03534ac` (feat)

**Plan metadata:** committed in the close-out docs commit (this file + STATE/ROADMAP/REQUIREMENTS).

## TDD Record

One RED→GREEN cycle with a classifier-verified evidence record (`evidence/04-05-red-second-prompt.json`, `RED_EVIDENCE_OK`, go-test adapter):

- **RED — second-prompt rejection** (`evidence/04-05-red-second-prompt.json`): `TestSecondPromptRejected/backend_-32010_refusal_mapped` failed on its planned class/message assertion — "rejection code = -32603, want -32600 (invalid-request class)" with the message leaking the wrapped transport noise (`session/prompt: turn: session/send: zcode: wire error -32010: ... : turn: a prompt is already running for this session`) instead of the one stable surface — front.go's machine-error branch mapped EVERY machine error through `acpport.NewInternal`. **Semantic assessment:** genuine RED on the planned assertion; the failing path is the exact branch the GREEN step flips, not a build/fixture/discovery failure. One test-authoring fix happened BEFORE evidence capture (an unused `context` import — build error, removed).
- **Honest lock semantics (plan-sanctioned):** the front-slot rejection leg and the two cancel locks (TestCancelMidTurn, TestCancelAfterCompleteNoOp) PASSED on first run — the tracer built those shapes in 04-06 and had already mapped the front-wide slot conflict to `NewInvalidRequest`. The plan text anticipated the slot as the RED ("today the slot conflict surfaces as the internal-class error"); code inspection + the RED run showed the genuine internal-class surface was the machine's ErrTurnActive branch, so the RED rides a backend-refusal corpus row the slot cannot pre-empt (send-refused-32010) — exactly the plan's prescribed GREEN shape ("keep the numeric -32010 match for the backend-refusal path"). No assertion was weakened and no RED was manufactured; the locks are recorded as green-first-run.

## Files Created/Modified
- `internal/acpfront/cancel_test.go` — TestCancelMidTurn / TestSecondPromptRejected (slot + backend-refusal legs) / TestCancelAfterCompleteNoOp + the shared assert helpers
- `internal/acpfront/front.go` — the ErrTurnActive → invalid-request mapping (GREEN); turnActiveMessage const shared by both guard surfaces
- `cmd/zcode-acp/main.go` — the full wiring: flags/env, Override discovery, supervisor composition, two-lifecycle shutdown, stderr-only logging
- `cmd/zcode-acp/main_test.go` — TestAgentBinaryEndToEnd (version + two-loop ride), the shim install-root fabricator, the transcript asserts, the collector client
- `cmd/fake-app-server/main.go` — `--transcript` flag
- `internal/zcode/fake/stdio.go` — MainTo + the transcript-file writer (golden format)
- `internal/supervisor/discover.go` — ExplicitRootInstall (exported order-step-1 override shape)
- `internal/zcode/fake/testdata/scenarios/{send-refused-32010,complete-then-cancel}.json` + goldens — the two new corpus rows
- `internal/zcode/fake/golden_test.go` — driveSendRefused32010, driveCompleteThenCancel, the sendContent/wireKeyContent constants
- `internal/zcode/fake/roundtrip_test.go` — the shared sendContent/wireKeyContent constants (goconst dedup)

## Decisions Made
- The rejection surface is one constructor + one message for both guard layers — the client can never tell which layer fired, and server wire text never leaks into ACP errors
- The RED rides the backend-refusal leg (send-refused-32010) because the slot leg was already invalid-request — recorded honestly, evidence gate-passed
- The spawn ctx is separate from the supervision ctx and cancelled only after the staged teardown completes — see Deviations (Rule 1)
- Loop-2's cancel syncs on the streamed delta, not a sleep — the chunk crossing the binary boundary IS the mid-turn proof
- The binary E2E fabricates a bundle-shaped install root whose "zcode" is a sh shim exec-ing the built fake with --scenario/--transcript baked in — the production discovery order and canonical SpawnArgv run unchanged (T-4-04's accepted trust position: the override is an explicit operator/test flag)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] The staged shutdown never staged: the spawn ctx killed the child before the graceful drain**
- **Found during:** Task 2 (the E2E's transcript assert — the fake never wrote it)
- **Issue:** run() passed the supervision ctx to SpawnChild, whose exec.Cmd wires `cmd.Cancel` to the group SIGKILL on that ctx — so `supCancel` fired exec's ctx-kill the same instant the supervisor's staged teardown began: the fake was SIGKILLed before its ~100 ms stdin-EOF drain and clean exit (shutdown "completed" in ~1 ms; the fake's exit status was the kill, its transcript write lost). The graceful stdin→grace→kill staging the supervisor documents was structurally defeated on EVERY shutdown, production included
- **Fix:** the spawn rides a dedicated ctx (`spawnCtx`) cancelled only AFTER startDone (the staged teardown completed); supCancel alone now drives stdin-close → grace → kill → reap, and spawnCancel is the post-teardown backstop (ESRCH-quiet). The E2E now proves the clean path: the fake's transcript appears (written at its own EOF-drain exit) and the wrapper exits 0
- **Files modified:** cmd/zcode-acp/main.go
- **Verification:** TestAgentBinaryEndToEnd green (-race, -count=2); full -race suite green
- **Committed in:** 03534ac

**2. [Rule 3 - Blocking] Two new corpus rows + drivers + goldens (outside the task file lists)**
- **Found during:** Task 1 and Task 2 (the rides need scenarios the corpus didn't have)
- **Issue:** the backend-refusal RED leg needs a -32010 send refusal reachable THROUGH the front (the slot would pre-empt a mid-turn second prompt), and the binary E2E needs one scenario serving BOTH loops deterministically (loop 1 completes, loop 2 stays active until the stop) — no existing row does either, and a corpus row without a driver fails TestGolden loud
- **Fix:** send-refused-32010 (Task 1 RED commit) and complete-then-cancel (Task 2) with the front preamble copied verbatim (driveFrontPreamble stays the one source), registered drivers, recorded goldens — the 04-04 "contract-lock plans carry their own corpus" precedent; corpus 13 → 15, TestGolden green throughout
- **Files modified:** internal/zcode/fake/testdata/scenarios/*.json, testdata/golden/*.golden, internal/zcode/fake/golden_test.go
- **Committed in:** ac993a1, 03534ac

**3. [Rule 3 - Blocking] fake binary `--transcript` (MainTo) — the binary hosting mode had no window into its own wire**
- **Found during:** Task 2 (the acceptance criterion "setModel visible on the fake transcript of the run")
- **Issue:** the standalone fake's transcript returns to fake.Main, which logs only the frame count — the binary E2E (which must assert the setModel payload the --model/--reasoning-level flags produced) had no way to read what the fake actually served
- **Fix:** fake.MainTo(scenarioPath, transcriptPath) writes the served transcript (golden dir\tjson format) after the clean EOF-drain exit — a write failure is ExitServeFailed (evidence never silently lost); cmd/fake-app-server passes the new optional flag; fake.Main's signature and its existing callers unchanged
- **Files modified:** internal/zcode/fake/stdio.go, cmd/fake-app-server/main.go
- **Committed in:** 03534ac

**4. [Rule 3 - Blocking] supervisor.ExplicitRootInstall exported — cmd cannot build the canonical override Install otherwise**
- **Found during:** Task 2 (mapping --app-server into the Install shape)
- **Issue:** the invariant spawn-surface builder (bundleInstall) and the matrix constants (zcode, resources/glm/zcode.cjs) are unexported — main.go would have to duplicate install-discovery.md :38 constants outside the package that owns them (drift risk)
- **Fix:** exported ExplicitRootInstall(root) validating the same spawn surface with layout "explicit"; main.go's dir-override path uses it, the file-override path maps Root/Binary/Entrypoint directly with the invariant-relative entrypoint
- **Files modified:** internal/supervisor/discover.go
- **Committed in:** 03534ac

**5. [Rule 3 - Blocking] 28 strict-lint findings in the new files (mise run ci lint stage)**
- **Found during:** the Task 2 gate
- **Issue:** err113 dynamic errors (7), wrapcheck (4 — Discover/SpawnChild/childDuplex passthrough), noctx (2), lll (4), nolintlint (2 — gosec already excluded in tests), funcorder/gocognit-adjacent (gocyclo 23 on the transcript assert), goconst (the send-content literal ×3 + the "content" key ×3), mnd (os.Exit 2), gocritic (filepath.Join separator), revive context-as-argument, thelper (2), errcheck (deferred conn.Close), plus funcorder method placement
- **Fix:** static sentinels wrapped with %w at the failure sites; CommandContext everywhere; the transport passthrough carries documented //nolint:wrapcheck (the Conn owns error identity); helpers split (e2eScanTranscript/e2eWaitTranscript); corpus constants named; context-first signatures; t.Helper() added; lint 0 issues
- **Files modified:** cmd/zcode-acp/main.go, cmd/zcode-acp/main_test.go, cmd/fake-app-server/main.go, internal/zcode/fake/{golden_test,roundtrip_test,cancel_test...}.go — see the commit
- **Verification:** mise run lint 0 issues; mise run ci fully green
- **Committed in:** 03534ac

---

**Total deviations:** 5 auto-fixed (1 Rule 1, 4 Rule 3) — none architectural, no Rule 4 decisions required.
**Impact on plan:** every fix serves the plan's own gates (the binary boundary proof, the corpus convention, the lint-clean CI). Prohibitions hold: error strings name the condition and carry no paths or prompt content (turnActiveMessage is the only concurrent-prompt text; the -32010 noise is gone from the wire), stdout stays protocol-exclusive (the SDK owns every stdout byte after --version; logs are stderr slog with structural fields only), and the override seam stays an explicit flag+env with no secret amplification (T-4-04 unchanged).

## Probe Ledger (from the plan)

- CORE-05 unclassified → stays UNRESOLVED, carried as the plan's flagged assumption: "CORE-05's edge space (exactly-once, no-op repeats, rejection class) is covered by explicit tests, but the probe classifier offered no category — carried as flagged, not auto-resolved." Its enforcement evidence: TestCancelMidTurn (exactly-once, never -32800), TestCancelAfterCompleteNoOp (no-op repeats), TestSecondPromptRejected (rejection class).
- No probe truths resolve in this plan (CORE-01/03 resolutions live in 04-03/04-04). The prohibition re-stated as a binding must_haves rule (machine-path/content-free error messages) is enforced by construction: the only new error surface is turnActiveMessage plus the pre-existing coded seam messages.
- Ledger equality: phase totals (COVERAGE.md): 10 = 7 resolved (6 explicit + 1 backstop) + 3 flagged (CORE-02 → 04-03, CORE-04 → 04-04, CORE-05 → here).

## Issues Encountered
- The first E2E run failed on a wrong relative scenario path (`../zcode/...` from cmd/zcode-acp needs `../../internal/...`) — the fake exited ExitUsage at spawn and the supervisor's crash-restart churn followed; fixed before anything was committed to the plan's production files.
- The shutdown race (Deviation 1) surfaced as a missing transcript file with a ~1 ms "shutdown complete" log line — the timing signature led straight to the exec ctx-kill vs staged-teardown race.

## User Setup Required

None - no external service configuration required (fake-backed composition; no desktop, no quota cost).

## Human Check (Manual-Only, Non-Gating)

The VALIDATION Manual-Only row (spawn the built zcode-acp under a real ACP client once with the fake wired via the override, per the Phase-3 realdesktop pattern): **deferred to the phase-end check** — not run in this plan. The non-gating spot check remains available: build the binary, fabricate a shim install root (the main_test.go shape), and point any ACP v1 client at it with `--app-server <root> --model zai-api/GLM-5.3-Flash --reasoning-level high`. Never blocks the phase gate.

## Next Phase Readiness
- The phase's five VALIDATION Wave-0 coverage rows are now all landed: fan-out (04-01), seam packages + boundary test + harness (04-02), corpus (04-02/04-04/04-05/04-06), tracer front (04-06), cancel locks + cmd wiring (this plan)
- CORE-05 is complete: after this SUMMARY all its declarers (04-02/04-04/04-05) carry SUMMARYs — `requirements.ready-ids` gates the mark-complete step
- `mise run ci` green at HEAD is the phase-gate rehearsal — the phase is ready for /gsd:verify-work
- Phase 5 inherits: the wired binary (flags/env surface), the rejection surface (one invalid-request for concurrent prompts), and the exec-tier transcript window (--transcript) for its tool-surface rides
- No blockers

---
*Phase: 04-acp-front-minimal-chat-loop*
*Completed: 2026-11-20*

## Self-Check: PASSED

All created/modified files exist on disk (14/14 checked, including this SUMMARY); all 3 task commits (ac993a1, 0d53155, 03534ac) are ancestors of HEAD (03534ac). Commit count measured from the persisted plan ledger: 3 (5ecb897..HEAD), no file deletions in any task commit. The RED evidence record classified RED_EVIDENCE_OK by the gsd-tools gate; `mise run ci` green at HEAD (build, vet, lint 0 issues, full -race suite incl. cmd/zcode-acp, experiments). TestAgentBinaryEndToEnd additionally green at -race -count=2; no stray fake processes after the rides.
