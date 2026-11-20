---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
verified: 2026-11-20T11:28:22Z
status: passed
score: 43/43 must-haves verified
covered_files:
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-01-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-01-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-02-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-02-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-03-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-03-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-04-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-04-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-05-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-05-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-06-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-06-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-07-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-07-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-08-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-08-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-09-PLAN.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/03-09-SUMMARY.md
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-01-red-happy-turn.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-02-red-router-credential-cipher.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-02-red-router-runtime-preferences.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-03-red-scenario-loader-validation.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-03-red-times-budget.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-04-red-discovery.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-05-red-completion-variants.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-05-red-turn-machine.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-07-red-spawn.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-07-red-supervise.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-08-red-cancel.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-review-red-cr01.json
  - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-review-red-wr01.json
  - cmd/fake-app-server/main.go
  - experiments/scan-secrets.sh
  - internal/supervisor/discover.go
  - internal/supervisor/discover_test.go
  - internal/supervisor/e2e_test.go
  - internal/supervisor/realdesktop_test.go
  - internal/supervisor/spawn.go
  - internal/supervisor/spawn_test.go
  - internal/supervisor/supervise.go
  - internal/supervisor/supervise_test.go
  - internal/turn/cancel_test.go
  - internal/turn/machine.go
  - internal/turn/machine_test.go
  - internal/turn/watcher.go
  - internal/zcode/call.go
  - internal/zcode/call_test.go
  - internal/zcode/conn.go
  - internal/zcode/conn_test.go
  - internal/zcode/envelope.go
  - internal/zcode/fake/binary_test.go
  - internal/zcode/fake/fake.go
  - internal/zcode/fake/golden.go
  - internal/zcode/fake/golden_test.go
  - internal/zcode/fake/roundtrip_test.go
  - internal/zcode/fake/scenario.go
  - internal/zcode/fake/scenario_test.go
  - internal/zcode/fake/stdio.go
  - internal/zcode/fake/testdata/golden/cancel-mid-turn.golden
  - internal/zcode/fake/testdata/golden/complete-then-cancel.golden
  - internal/zcode/fake/testdata/golden/completion-idle-patch.golden
  - internal/zcode/fake/testdata/golden/completion-quiet-window.golden
  - internal/zcode/fake/testdata/golden/credential-cipher-scripted.golden
  - internal/zcode/fake/testdata/golden/envelope-negatives.golden
  - internal/zcode/fake/testdata/golden/eof-drain.golden
  - internal/zcode/fake/testdata/golden/error-32010.golden
  - internal/zcode/fake/testdata/golden/happy-turn.golden
  - internal/zcode/fake/testdata/golden/no-reasoning.golden
  - internal/zcode/fake/testdata/golden/part-delta-streaming.golden
  - internal/zcode/fake/testdata/golden/permission-allow.golden
  - internal/zcode/fake/testdata/golden/permission-cancel.golden
  - internal/zcode/fake/testdata/golden/reannounce-dedupe.golden
  - internal/zcode/fake/testdata/golden/runtime-preferences.golden
  - internal/zcode/fake/testdata/golden/send-refused-32010.golden
  - internal/zcode/fake/testdata/golden/setmodel-followup.golden
  - internal/zcode/fake/testdata/golden/unknown-host-method.golden
  - internal/zcode/fake/testdata/golden/user-input-decline.golden
  - internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json
  - internal/zcode/fake/testdata/scenarios/complete-then-cancel.json
  - internal/zcode/fake/testdata/scenarios/completion-idle-patch.json
  - internal/zcode/fake/testdata/scenarios/completion-quiet-window.json
  - internal/zcode/fake/testdata/scenarios/credential-cipher-scripted.json
  - internal/zcode/fake/testdata/scenarios/envelope-negatives.json
  - internal/zcode/fake/testdata/scenarios/eof-drain.json
  - internal/zcode/fake/testdata/scenarios/error-32010.json
  - internal/zcode/fake/testdata/scenarios/happy-turn.json
  - internal/zcode/fake/testdata/scenarios/no-reasoning.json
  - internal/zcode/fake/testdata/scenarios/part-delta-streaming.json
  - internal/zcode/fake/testdata/scenarios/permission-allow.json
  - internal/zcode/fake/testdata/scenarios/permission-cancel.json
  - internal/zcode/fake/testdata/scenarios/reannounce-dedupe.json
  - internal/zcode/fake/testdata/scenarios/runtime-preferences.json
  - internal/zcode/fake/testdata/scenarios/send-refused-32010.json
  - internal/zcode/fake/testdata/scenarios/setmodel-followup.json
  - internal/zcode/fake/testdata/scenarios/unknown-host-method.json
  - internal/zcode/fake/testdata/scenarios/user-input-decline.json
  - internal/zcode/roundtrip_test.go
  - internal/zcode/writer.go
  - internal/zcode/writer_test.go
covered_digest: "v4:sha256:d48d0639791738fc036fd5c3526a08f28d0f1a016ded5313afb1b4ff239489cf"
behavior_unverified: 0 # Count of PRESENT_BEHAVIOR_UNVERIFIED truths (present + wired, behavior not exercised); each is detailed in behavior_unverified_items below (and in human_verification when status is human_needed)
overrides_applied: 0 # Count of PASSED (override) items included in score
re_verification: # Stale-digest regeneration (#4682 route): prior report passed 43/43 on 2026-11-19; Phase 4-5 stacked work modified every phase-3 implementation file since, so all must-haves were re-checked behaviorally on the current tree
  previous_status: passed
  previous_score: 43/43
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor — Verification Report

**Phase Goal:** As a later-phase implementer on this repo, I want to the wrapper's back leg — a JSONL codec for the app-server dialect, a scriptable fake app-server with golden transcripts in CI, a desktop discovery + process supervisor, and a turn state machine, so that every later phase is testable without the real desktop bundle.
**Verified:** 2026-11-20T11:28:22Z
**Status:** passed
**Re-verification:** Yes — stale-verification regeneration (#4682): the prior 2026-11-19 report (passed 43/43) went digest-stale after Phase 4-5 stacked-branch work modified every phase-3 implementation file. All 43 must-haves re-verified behaviorally against the current working tree.

**Mode:** mvp (user-story goal validated via `user-story.validate` → `true` at the prior pass; goal unchanged)

## What changed since the verified tree (66235cd..HEAD)

All net changes to phase-3 files are **additive extensions** by Phase 4-5 plans — no phase-3 invariant was removed:

- `internal/zcode/conn.go`: single-slot `OnNotification`/`OnRouterEvent` hooks replaced by ordered fan-outs (`AddObserver`/`AddRouterObserver` + remove funcs). The D-05 router policy (deadlines, credential-cipher tracked-no-answer, -32601 unknown-method) is untouched. Wire-order delivery preserved (registration-order iteration on the read goroutine).
- `internal/turn/machine.go`: registration moved to the fan-outs + `CloseObservers` teardown (WR-07). Run/Cancel resolution, -32010 numeric mapping, single-session binding unchanged.
- `internal/zcode/envelope.go`: additive method constants (`session/subscribe`, `session/setModel`) + client-request error-code block (documented namespace collision). No framing-key field added.
- `internal/zcode/fake/{fake,scenario}.go`: `reannounce` scripting (fresh wire ids, fail-loud validation) + `stdio.go`/`cmd/fake-app-server/main.go`: optional `--transcript` sink (fail-loud on write error). `--scenario` remains required.
- `internal/supervisor/discover.go`: additive `ExplicitRootInstall` (order-step-1 override helper).
- Corpus grew 10 → 19 scenario+golden pairs (Phase 4-5 rows: complete-then-cancel, no-reasoning, part-delta-streaming, permission-allow, permission-cancel, reannounce-dedupe, send-refused-32010, setmodel-followup, user-input-decline; cancel-mid-turn and completion-quiet-window extended).
- `go.mod` now carries `github.com/coder/acp-go-sdk v0.20.0` — added by Phase 4 (04-01, commit 7bad5b5) which updated the `tidy-diff` gate to assert exactly that pin (one require line). Intentional, gated, later-phase scope — not a phase-3 regression; phase 3's own work still added zero dependencies.

## User Flow Coverage

User story: «As a later-phase implementer on this repo, I want to the wrapper's back leg — a JSONL codec for the app-server dialect, a scriptable fake app-server with golden transcripts in CI, a desktop discovery + process supervisor, and a turn state machine, so that every later phase is testable without the real desktop bundle.»

The "user" is a later-phase implementer; the outcome clause is «every later phase is testable without the real desktop bundle». Each step below was re-executed first-hand on the current stacked tree.

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Run the codec against dialect frames | Frames without a `jsonrpc` key, dialect error codes, ordered single-writer output | `internal/zcode/envelope.go:86` (framing key unrepresentable by construction), `TestEncodeOmitsJSONRPC` (asserts no "jsonrpc" byte in the outbound line, `roundtrip_test.go:44`) PASS first-hand; error-code table `envelope.go:156-186`; `TestWriterOrdering` + `TestStopJumpsQueue` PASS | ✓ |
| Replay golden transcripts with no desktop present | Every corpus scenario replays byte-deterministically over pipes | `TestGolden` **19/19 subtests PASS** first-hand (the original 10 plus 9 later-phase rows — a superset of the phase-3 corpus contract); 19 scenarios + 19 goldens on disk; `TestFakeBinarySpawnPath` PASS (real child process) | ✓ |
| Use the standalone fake binary tier | Binary serves `--scenario` over stdio; frames proven over a real child | `cmd/fake-app-server/main.go` (`--scenario` required, optional `--transcript` additive), `TestFakeBinarySpawnPath` + `TestEOFSignatureChild` PASS first-hand | ✓ |
| Point discovery at an install — or at nothing | First-match-wins per the Phase-1 matrix; empty result names every probe; /tmp/.mount_* rejected | `TestDiscoverFixtureTree`, `TestDiscoverNoInstallListsEveryProbe`, `TestDiscoverOrderFirstMatchWins`, `TestDiscoverRejectsVolatileMounts`, `TestDiscoverFindsNewestBuiltin` all PASS first-hand; revision formula `discover.go:147-162` (EvalSymlinks → sha256 over the resolved absolute path STRING) | ✓ |
| Supervise a child; kill the whole group on exit | Zero surviving group members (grandchildren included); allowlisted+scrubbed env on the child's observed env | `TestGroupKillNoSurvivors` PASS, `TestSpawnEnv` PASS (child's OBSERVED env via reexec helper), `TestRestartWindow` PASS, `TestSpawnArgvShape` PASS | ✓ |
| Drive a synchronous turn; cancel mid-turn | Fire-and-forget `session/send` resolves exactly once; cancel resolves cancelled exactly once | `TestHappyTurnCompletedExactlyOnce`, `TestCompletionVariants`, `TestTerminalCarrierTolerance`, `TestCancelMidTurnAnswersOnce`, `TestCancelRacingTerminalResolvesOnce`, `TestDoubleSendRejectedWrapperSide` + `ServerSide` (-32010), `TestCompletionCorpus`, `TestCredentialCipherExpiryRecorded` all PASS first-hand | ✓ |
| Compose the whole chain | Supervisor + fake binary + Conn + Machine in one test | `TestEndToEndExecChain` PASS first-hand with both subtests (`happy_turn_completes_exactly_once`, `cancel_mid-turn_resolves_cancelled_exactly_once`) | ✓ |
| Outcome: every later phase testable without the desktop bundle | Full gate green on a machine that HAS the desktop — proving nothing in the suite needs it | Build/vet/lint (0 issues)/test-experiments (22/22)/tidy-diff all EXIT=0 first-hand this run; full `go test -race` suite green this session on this exact tree (orchestrator-stated, all packages exit 0) plus my first-hand package runs of all four phase-3 packages; the real-desktop path is `-tags realdesktop` + `ZCODE_ACP_REAL_E2E=1` guarded — default suite never compiles it | ✓ |

## Goal Achievement

### Observable Truths

Roadmap truths (RT) are the contract; plan truths (Pxx-y) are the plan-frontmatter detail merged per Step 2c. **All evidence below is first-hand re-execution on the current stacked tree (2026-11-20)** — not inherited from the prior report.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| RT1 | The codec round-trips the app-server JSONL dialect: frames without a `jsonrpc` key, the dialect's own error codes, ordered single-writer output — a JSON-RPC-shaped frame never reaches the child | ✓ VERIFIED | `envelope.go:86` framing key unrepresentable by construction; `TestEncodeOmitsJSONRPC` + `TestDecodeRoundTrip` + `TestDecodeEchoesIDTypes` + `TestWriterOrdering` PASS first-hand; corpus `error-32010` + `envelope-negatives` replay green (inside TestGolden 19/19); lint 0 issues |
| RT2 | CI runs golden JSONL transcript tests against the scriptable fake app-server (in-process over pipes plus a standalone binary), no desktop bundle present | ✓ VERIFIED | `TestGolden` 19/19 PASS + `TestFakeBinarySpawnPath` PASS first-hand; `.github/workflows/pr-sanity.yml` runs the same mise tasks (build/vet/lint/test/test-experiments/tidy-diff + govulncheck + secret-scan); realdesktop tier build-tag+env guarded (realdesktop_test.go header) |
| RT3 | The supervisor locates the install per the Phase 1 matrix, spawns the child with the correct environment, kills the whole process group on EOF/exit leaving no orphans; no-install fails with diagnostics naming what was searched | ✓ VERIFIED | 5 discovery tests PASS (incl. `TestDiscoverNoInstallListsEveryProbe`, `TestDiscoverRejectsVolatileMounts`); `TestSpawnEnv`, `TestSpawnArgvShape`, `TestGroupKillNoSurvivors`, `TestRestartWindow` PASS first-hand; real-desktop positive path re-executed first-hand on the current tree: `TestRealDesktop` PASS 2.34s — layout=unpacked-appimage, revision zcode-builtin:44:e9c0674a…, session/create ok, kill(-pgid,0) ESRCH zero orphans; `ps` after: zero app-server processes |
| RT4 | Against the fake app-server, a fire-and-forget `session/send` is lifted into a synchronous turn that completes exactly once on multi-signal turn-completion detection and honors cancel mid-turn | ✓ VERIFIED | `TestHappyTurnCompletedExactlyOnce`, `TestCompletionVariants`, `TestTerminalCarrierTolerance`, `TestCancelMidTurnAnswersOnce`, `TestCancelRacingTerminalResolvesOnce`, `TestCompletionCorpus`, `TestCredentialCipherExpiryRecorded` all PASS first-hand under -race |
| P01-1 | Conn vs in-process fake over io.Pipe completes happy-turn end-to-end (session/create with reverse request answered, session/send acked accepted:true, notifications in wire order) | ✓ VERIFIED | `TestHappyTurnRoundTrip` + `TestGoldenHappyTurn` green (inside this run's package runs); fan-out refactor preserved wire-order delivery (registration-order iteration, conn.go AddObserver doc) |
| P01-2 | Every byte to the child is a complete newline-delimited JSON envelope; no outbound frame can carry a JSON-RPC framing key (structural) | ✓ VERIFIED | `TestEncodeOmitsJSONRPC` PASS first-hand; `envelope.go:86` structural comment + `bytes.Contains(line, "jsonrpc")` negative assertion (`roundtrip_test.go:44`); envelope diff since verified tree is additive-only |
| P01-3 | Reverse-request replies travel a lane that never queues behind ordinary calls (two-lane writer) | ✓ VERIFIED | `writer.go` unchanged since verified tree (bypass channel checked before queued writes); `Conn.Reply` + `Conn.Stop` enqueue bypass (`call.go` unchanged); `TestStopJumpsQueue` PASS first-hand |
| P01-4 | Golden replay byte-deterministic (fixed ids); any frame drift or missing frame fails — never normalized away | ✓ VERIFIED | `TestGolden` 19/19 PASS first-hand including the two extended rows (cancel-mid-turn, completion-quiet-window); compare path in `golden.go` unchanged (`-update` regeneration path intact) |
| P02-1 | WR-02: a numeric-id reverse request colliding with an in-flight call id routes to the router, NEVER resolves the pending call | ✓ VERIFIED | `TestDemuxReverseRequestNeverResolvesPendingCall` PASS first-hand; WR-02 router fan-out (04ef5c7) preserved the router-first classification |
| P02-2 | Ids correlate by exact value AND type: string 5 ≠ integer 5 on the wire | ✓ VERIFIED | `TestDemuxIdTypeNonConflation` + `TestIDKeyTypeTagging` + `TestCallReplyEchoesIDVerbatim` PASS first-hand |
| P02-3 | Single-writer ordering: concurrent writers produce complete, never-interleaved frames | ✓ VERIFIED | `TestWriterOrdering` PASS first-hand; writer.go unchanged |
| P02-4 | Bypass precedence: session/stop issued while the ordered queue holds pending writes is on the wire after at most the in-flight write | ✓ VERIFIED | `TestStopJumpsQueue` PASS first-hand; bypass lane intact in call.go/writer.go |
| P02-5 | Raised-line behavior: 128 KiB line survives intact; malformed JSON line skipped with warning, stream continues | ✓ VERIFIED | `TestLineBufferLargeLineIntact` + `TestLineBufferMalformedLineBetweenValidLines` PASS first-hand; `conn.go:264` `sc.Buffer(make([]byte,0,readBufInitial), readBufMax)` (checker literal "Scanner.Buffer" is a naming false positive — var is `sc`) |
| P02-6 | D-05 deadline honesty: runtime preferences answered within deadline; mid-turn credential-cipher known-but-NOT-answered; expiry recorded observably | ✓ VERIFIED | `TestRouterRuntimePreferencesDeadline` + `TestRouterCredentialCipherNeverAnsweredExpiryRecorded` + `TestRouterUnknownMethodAnswered32601` PASS first-hand; the D-05 policy in conn.go was untouched by the fan-out refactor |
| P03-1 | The golden corpus lives under testdata/scenarios/ and every scenario replays green against the codec Conn under -race | ✓ VERIFIED | `TestGolden` 19/19 PASS first-hand (corpus grew 10 → 19 by later-phase rows; all original rows present and green — superset holds); 19+19 files on disk |
| P03-2 | Every golden replays byte-deterministically; `-update` regenerates; drift fails | ✓ VERIFIED | `TestGolden` green first-hand on the extended corpus; `golden.go` unchanged since the verified tree |
| P03-3 | The scenario loader fails loud: invalid file rejected with an error naming the offending field | ✓ VERIFIED | 15 `TestLoadScenarioRejectsInvalid*` tests PASS first-hand (the original 11 plus 3 new reannounce validations + 1) — every message names the offending field |
| P03-4 | Exit-0-with-zero-lines is a FAILURE in-process, never an empty success | ✓ VERIFIED | `TestEOFSignature` PASS first-hand; exec-tier twin `TestEOFSignatureChild` PASS first-hand |
| P04-1 | Discovery implements the Phase 1 recommended order exactly; first-match-wins; probes do not merge | ✓ VERIFIED | `TestDiscoverOrderFirstMatchWins` + `TestDiscoverFixtureTree` PASS first-hand; the +10-line discover.go change is the additive `ExplicitRootInstall` helper on the same order-step-1 path |
| P04-2 | Empty result names EVERY probe attempted with its result — the diagnostics are the product | ✓ VERIFIED | `TestDiscoverNoInstallListsEveryProbe` PASS first-hand (8 probes: override, xdg-entry, appimage-glob, unpacked-glob, dpkg, opt-path, rpm, builtin-runtime) |
| P04-3 | Volatile /tmp/.mount_* decoys recorded-and-rejected; localhost listeners never probed | ✓ VERIFIED | `TestDiscoverRejectsVolatileMounts` PASS first-hand |
| P04-4 | Builtin revision computed, never hardcoded: sha256 over the resolved absolute PATH STRING | ✓ VERIFIED | `discover.go:147-162` (`filepath.EvalSymlinks` → `sha256.Sum256([]byte(abs))`); `TestBuiltinRevisionFormula` + `TestDiscoverFindsNewestBuiltin` PASS; real-desktop run printed `zcode-builtin:44:e9c0674a…` live first-hand today |
| P05-1 | Machine.Run lifts one session/send into a synchronous turn: returns StopReason exactly once, after a completion signal — never before, never twice | ✓ VERIFIED | `TestHappyTurnCompletedExactlyOnce` PASS first-hand; E2E subtest `happy_turn_completes_exactly_once` PASS; machine.go diff since verified tree touches only observer wiring + docs, not Run's resolution logic |
| P05-2 | Completion classification multi-signal and ordered (terminal → idle patch → quiet-window); each variant resolves exactly once | ✓ VERIFIED | `TestCompletionVariants` PASS first-hand (three variants as subtests) |
| P05-3 | Terminal classifier matches turnId + kind across known carriers; carrier-tolerant by design | ✓ VERIFIED | `TestTerminalCarrierTolerance` PASS first-hand |
| P05-4 | TurnWatcher is an interface with a polling implementation; quiet-window gates with send-latency floor 3x quiet-ms after the send ack; subscribe watcher is a reserved slot, NOT implemented | ✓ VERIFIED | `watcher.go:27` `DefaultFloorMultiple = 3` injectable (set at :120 when not injected — a precondition the code establishes); `TestTurnWatcherDeliversUntilStopped` PASS; grep confirms no subscribe implementation exists (doc mentions only) |
| P05-5 | D-04: the machine is bound to ONE app-server sessionId on the ONE shared Conn (1:1 ACP session mapping) | ✓ VERIFIED | `machine.go:147-149` documented single-session binding, unchanged; E2E asserts the create reply's sessionId against the binding |
| P06-1 | Standalone fake binary serves a JSON scenario over its own stdio: --scenario required, frames on stdin/stdout, diagnostics stderr-only | ✓ VERIFIED | `cmd/fake-app-server/main.go` (usage error → `fake.ExitUsage`, stderr slog; additive optional `--transcript` flag delegates to `fake.MainTo`); `TestFakeBinarySpawnPath` PASS first-hand |
| P06-2 | EOF contract: on stdin EOF the fake drains ~100 ms (eofDrainMs) then exits 0; a request just before EOF is never answered; drain observable in tests | ✓ VERIFIED | `TestFakeBinarySpawnPath` + eof-drain corpus row + golden green; `TestEOFSignatureChild` PASS first-hand |
| P06-3 | Exec-plumbing proven against a REAL child: helper reexec drives frames both directions with the child in its own process group | ✓ VERIFIED | `TestFakeBinarySpawnPath` PASS first-hand (GO_WANT_HELPER_PROCESS reexec) |
| P06-4 | Exit-0-with-zero-lines failure signature enforced at the exec tier | ✓ VERIFIED | `TestEOFSignatureChild` PASS first-hand |
| P07-1 | Child spawns with the canonical contract: argv [binary, entrypoint, app-server, --stdio]; env ELECTRON_RUN_AS_NODE=1 + ZCODE_BUILTIN_PROVIDER_CONFIG_FILE | ✓ VERIFIED | `TestSpawnArgvShape` + `TestSpawnEnv` PASS first-hand; spawn.go unchanged since verified tree |
| P07-2 | Child env is an explicit allowlist (PATH+HOME) + two ZCode vars, proxy vars scrubbed — asserted on the child's OBSERVED environment | ✓ VERIFIED | `TestSpawnEnv` asserts via the `TestSpawnEnvHelper` reexec child's own observed env (`spawn.go:44-53` allowlist constants); PASS first-hand |
| P07-3 | Whole process group dies on EOF/exit: staged shutdown (stdin end → wait/grace → Kill(-pgid, SIGKILL) → reap) leaves zero survivors incl. grandchildren, ESRCH-verified | ✓ VERIFIED | `TestGroupKillNoSurvivors` PASS first-hand; `supervise.go:184` staged shutdown calls `killGroup` → `spawn.go:306-308` `syscall.Kill(-pgid, SIGKILL)` (ESRCH tolerated); real-desktop run re-verified ESRCH first-hand today |
| P07-4 | Crash-window restart: 3 attempts within sliding 60 s then fail loud; stable run resets the counter; injectable for tests | ✓ VERIFIED | `TestRestartWindow` PASS first-hand — includes the crash-loop-trip-leaves-zero-survivors case added by review fix 6794626/5bbb38f |
| P07-5 | Health watch = stdin liveness + child exit; NO protocol liveness frame is invented | ✓ VERIFIED | `supervise.go:6-8,109-112` explicit contract (unchanged); grep confirms no ping/liveness frame code |
| P08-1 | Cancel mid-turn: session/stop through the bypass lane immediately; turn resolves cancelled EXACTLY once; second Cancel no-op; cancel-after-complete no-op | ✓ VERIFIED | `TestCancelMidTurnAnswersOnce` + `TestCancelRacingTerminalResolvesOnce` PASS first-hand; `Conn.Stop` bypass enqueue unchanged |
| P08-2 | A second concurrent session/send is rejected wrapper-side with a clean error mapping the dialect code -32010 (numeric code, never message text) | ✓ VERIFIED | `TestDoubleSendRejectedWrapperSide` + `TestDoubleSendRejectedServerSide` PASS first-hand; `machine.go:130-134` maps `zcode.CodeConcurrency` numerically (grep-confirmed unchanged) |
| P08-3 | Corpus-level completion: completion-idle-patch and completion-quiet-window replay against the codec Conn, each resolving exactly once | ✓ VERIFIED | `TestCompletionCorpus` PASS first-hand (loads the JSON corpus via `corpusScenarioPath`) |
| P08-4 | D-05 honesty at the turn tier: scripted credential-cipher exchange ends with expiry RECORDED, turn NOT falsely resolved — degradation observable, never swallowed | ✓ VERIFIED | `TestCredentialCipherExpiryRecorded` PASS first-hand |
| P09-1 | The FULL chain runs in one test: supervisor spawns the standalone fake binary through the real spawn path (process group, allowlisted env, drained stderr), Conn rides the exec pipes, Machine.Run completes the happy turn exactly once | ✓ VERIFIED | `TestEndToEndExecChain` PASS first-hand; `TestMain` builds the binary via `go build` into `t.TempDir()` |
| P09-2 | Cancel works over the exec chain: mid-turn cancel resolves cancelled exactly once; staged shutdown leaves zero surviving group members (ESRCH) | ✓ VERIFIED | E2E subtest `cancel_mid-turn_resolves_cancelled_exactly_once` PASS first-hand (second supervised cycle serving cancel-mid-turn.json — documented deviation, keeps every must-have) |
| P09-3 | The whole phase gate is green: mise run ci with the corpus replayed inside; zero dependencies added by this phase's own work | ✓ VERIFIED | Gate legs re-run first-hand today: build EXIT=0, vet EXIT=0, lint 0 issues, test-experiments 22/22, tidy-diff EXIT=0, secret-scan EXIT=0; all four phase-3 packages green under -race first-hand; full-suite green this session (all packages exit 0). go.mod now carries acp-go-sdk v0.20.0 — added by Phase 4 (04-01, 7bad5b5) with the tidy-diff gate updated to assert exactly that pin; phase 3's own diff added zero dependencies, so the truth holds at phase scope on the cumulative tree |

**Score:** 43/43 truths verified (0 present, behavior-unverified)

### Advisory (New Scope, Unevidenced)

Re-verification ran, so this section is included. None — the anti-pattern scan produced zero findings (debt markers, placeholders, empty implementations: 0 across all 30 phase-3 files plus the 9 new corpus pairs). The go.mod dependency observation is not advisory material: it has deterministic evidence of being intentional (mise.toml `tidy-diff` asserts `require github.com/coder/acp-go-sdk v0.20.0` verbatim, added by Phase 4 commit 7bad5b5).

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Required Artifacts

All artifacts re-checked on the current tree. gsd-tools `verify.artifacts`: 6/9 plans all-pass; the 4 misses below are the known literal-pattern checker false positives, each manually verified present on the current tree.

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/zcode/envelope.go` | ID, Envelope (no framing-key field), WireError, method + error-code constants | ✓ VERIFIED | Additive constants since verified tree; framing-key structural defense intact (:86); client-request code block documented at :171-186 |
| `internal/zcode/conn.go` | Conn: read loop with raised scanner buffer, three-way demux, reverse-request router with D-05 policy | ✓ VERIFIED | `sc.Buffer` raised-line at :264 (checker "Missing pattern: Scanner.Buffer" is a var-naming false positive — direct read); D-05 policy intact; fan-out observers additive |
| `internal/zcode/writer.go` | Single writer goroutine, two lanes (ordered + bypass) | ✓ VERIFIED | Unchanged since verified tree; bypass lane verified by TestStopJumpsQueue |
| `internal/zcode/call.go` | Call/Notify/Reply/ReplyError, pending table, Stop helper | ✓ VERIFIED | Unchanged since verified tree |
| `internal/zcode/{roundtrip,conn,writer,call}_test.go` | Codec unit tests per plan | ✓ VERIFIED | conn_test.go grew +469 lines (fan-out/WR-01 tests from Phase 5); all green under -race |
| `internal/zcode/fake/fake.go` | Serve(rw, script): scripted matching, delays, events, blocking reverse requests | ✓ VERIFIED | +reannounce scripting (runReannounce, wg-tracked goroutines) — additive |
| `internal/zcode/fake/scenario.go` + `scenario_test.go` | JSON scenario schema + fail-loud load; loader validation tests | ✓ VERIFIED | +Reannounce schema with fail-loud validate (count > 0, interval >= 0); 15 rejection tests green |
| `internal/zcode/fake/golden.go` + `golden_test.go` | Transcript record/compare with -update; corpus replay + EOF signature | ✓ VERIFIED | golden.go unchanged; golden_test.go grew +662 lines (later-phase corpus rows); TestGolden 19/19 green |
| `internal/zcode/fake/testdata/scenarios/` | Golden corpus scenario files | ✓ VERIFIED | 19 JSON files on disk (checker "Not a file: directory" is a file-only checker — contents verified by listing + replay) |
| `internal/zcode/fake/testdata/golden/` | Golden transcripts | ✓ VERIFIED | 19 .golden files on disk; all replay green |
| `internal/zcode/fake/binary_test.go` | Reexec integration tests (spawn path, EOF drain, zero-lines signature) | ✓ VERIFIED | GO_WANT_HELPER_PROCESS pattern; both named tests PASS first-hand |
| `internal/zcode/fake/stdio.go` | stdio hosting of the scripting core (fake.Main/ExitUsage) | ✓ VERIFIED | +MainTo transcript sink (fail-loud ExitServeFailed on write error); used by cmd main |
| `cmd/fake-app-server/main.go` | Thin main: --scenario flag, stdio serving, stderr diagnostics, EOF drain exit | ✓ VERIFIED | Still thin (~29 lines); --scenario required; optional --transcript delegates to fake.MainTo |
| `internal/supervisor/discover.go` + `discover_test.go` | Ordered first-match discovery, full diagnostics, decoy rejection, BuiltinRevision | ✓ VERIFIED | +ExplicitRootInstall helper (order-step-1 override, additive); revision formula :147-162; 6 discovery tests green |
| `internal/supervisor/spawn.go` | SpawnArgv, ChildEnv, SpawnChild with Setpgid + 3 stdio pipes + capped stderr drain + group-kill Cancel | ✓ VERIFIED | Unchanged since verified tree; Setpgid + killGroup :306-308 verified by direct read |
| `internal/supervisor/supervise.go` | Supervisor: health watch, staged shutdown, crash-window restart | ✓ VERIFIED | Unchanged; staged shutdown :168-192 wired to killGroup at :184 (checker "Missing pattern: Kill" is a cross-file false positive — the literal lives in spawn.go, verified wired by direct read) |
| `internal/supervisor/{spawn,supervise,realdesktop,e2e}_test.go` | Env/argv/group-kill/restart/realdesktop/E2E tests | ✓ VERIFIED | All green first-hand; realdesktop behind `//go:build realdesktop` + env guard, executed green first-hand today |
| `internal/turn/machine.go` + `watcher.go` + `machine_test.go` + `cancel_test.go` | Machine (Run/Cancel, -32010 guard), TurnWatcher + PollWatcher, tests | ✓ VERIFIED | machine.go: fan-out registration + CloseObservers (additive); watcher.go unchanged; 12 turn tests green; corpus path `corpusScenarioPath` at cancel_test.go:570-572 |
| `internal/supervisor/e2e_test.go` | End-to-end exec-chain test | ✓ VERIFIED | TestEndToEndExecChain PASS first-hand (both subtests) |
| evidence/ (13 RED-proof JSONs) | TDD RED evidence for 03-01/02/03/04/05/07/08 + review CR-01/WR-01 | ✓ VERIFIED | All 13 present and unchanged since the verified tree (in the covered set) |

### Key Link Verification

gsd-tools `verify.key-links`: 8/9 plans all_verified; the single miss is the known corpus-path false positive, manually verified wired on the current tree.

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/zcode/fake/roundtrip_test.go | internal/zcode/fake/fake.go | io.Pipe pairing | ✓ WIRED | tool-verified |
| internal/zcode/conn.go | internal/zcode/call.go | demux resolves pending by canonical id | ✓ WIRED | tool-verified |
| internal/zcode/call.go | internal/zcode/writer.go | Call/Notify ordered lane; Reply/Stop bypass lane | ✓ WIRED | tool-verified |
| internal/zcode/conn_test.go | internal/zcode/conn.go | demux + router policy via public Conn surface | ✓ WIRED | tool-verified (grew +469 lines of fan-out tests, all green) |
| internal/zcode/fake/golden_test.go | testdata/scenarios/ | table-driven LoadScenario → Serve → CompareGolden | ✓ WIRED | tool-verified |
| testdata/scenarios/completion-quiet-window.json | fake/fake.go | session/messages poll answers | ✓ WIRED | tool-verified |
| internal/supervisor/discover_test.go | discover.go | Probers seams (t.TempDir fixtures) | ✓ WIRED | tool-verified |
| discover.go | docs/protocol/install-discovery.md | probe order/set/fields mirror matrix | ✓ WIRED | tool-verified |
| internal/turn/machine.go | zcode/conn.go / call.go | session/send via Call; Stop bypass; -32010 numeric mapping | ✓ WIRED | tool-verified + direct read (fan-out registration replaces hook assignment; same primitives) |
| internal/turn/cancel_test.go | testdata/scenarios/cancel-mid-turn.json | corpus scenario replay via corpusScenarioPath | ✓ WIRED | tool pattern `testdata/scenarios` missed the `filepath.Join("..","zcode","fake","testdata","scenarios",…)` construction at cancel_test.go:570-572 — verified wired by direct read; TestCompletionCorpus green |
| supervisor/spawn.go | discover.go | SpawnArgv + ChildEnv consume Install | ✓ WIRED | tool-verified |
| supervisor/supervise.go | spawn.go | staged shutdown → killGroup | ✓ WIRED | tool-verified + direct read (supervise.go:184 → spawn.go:306-308) |
| supervisor/spawn_test.go | cmd/fake-app-server/main.go | fake binary as the CI child | ✓ WIRED | tool-verified |
| supervisor/e2e_test.go | supervise.go / machine.go / cmd main | Supervisor lifecycle + Machine over exec pipes + go build child | ✓ WIRED | tool-verified |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `TestEndToEndExecChain` | StopReason / sessionId | Real exec pipes to the built fake-app-server child (go build → t.TempDir) driven through SpawnChild + Conn + Machine | Yes — real process I/O, real frames on os-level pipes; re-run first-hand today | ✓ FLOWING |
| `TestSpawnEnv` | Child's observed environment | Reexec helper child reads its own os.Environ and reports it back | Yes — asserted on the OBSERVED child env, not the builder's intention | ✓ FLOWING |
| `TestDiscover*` | Install candidates | Injected Probers seams over t.TempDir fixture trees | Yes — the seams are the documented injection point; production path consumes real probes (proven by today's real-desktop run resolving the live install) | ✓ FLOWING |
| `TestRealDesktop` | Install root / builtinRevision | The actual ZCode 3.18.2 install on this machine (layout=unpacked-appimage) | Yes — re-executed first-hand today, revision zcode-builtin:44:e9c0674a… printed | ✓ FLOWING |
| Golden corpus replay | Transcript frames | fake.Serve over io.Pipe from hand-authored scenario JSONs | Yes — the fake IS the intended data source for later phases (D-01); ids are synthetic by contract (sess_fix1/turn_fix1); the corpus is now the shared spec surface for Phases 4-5 rows too | ✓ FLOWING |

No static returns, hardcoded literals, or mocks on any asserted path. The fake app-server is the product (a scriptable stand-in), not a mock hiding a missing implementation — the E2E chain proves the composition over real exec pipes.

### Behavioral Spot-Checks

All commands below re-run first-hand on the current tree (2026-11-20). Package-scoped runs cover each phase-3 package exactly once; named runs are single named tests. The full workspace suite ran green this session per the orchestrator (all 10 packages, exit 0) and was not re-run.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Codec package (18+ tests incl. all RT1/P01/P02 invariants) | `go test -race -count=1 ./internal/zcode/` | ok 3.108s | ✓ PASS |
| Fake package (TestGolden 19/19 + loader + EOF + binary tier) | `go test -race -count=1 ./internal/zcode/fake/` | ok 7.530s | ✓ PASS |
| Golden corpus verbose (19 rows) | `go test -race -count=1 -run 'TestGolden$' -v ./internal/zcode/fake/` | 19/19 subtests PASS (3.90s) | ✓ PASS |
| Turn package (exactly-once, cancel, -32010, honesty, carriers) | `go test -race -count=1 ./internal/turn/` + named -v run | ok 3.195s; 9/9 named PASS | ✓ PASS |
| Supervisor package (discovery, spawn, group kill, restart, E2E) | `go test -race -count=1 ./internal/supervisor/` + named -v run | ok 2.418s; 10/10 named PASS | ✓ PASS |
| E2E exec chain verbose | `go test -race -count=1 -run TestEndToEndExecChain -v ./internal/supervisor/` | PASS both subtests (0.41s) | ✓ PASS |
| Fake binary tier verbose | `go test -race -count=1 -run 'TestFakeBinarySpawnPath\|TestEOFSignatureChild' -v ./internal/zcode/fake/` | both PASS (~1.1s each) | ✓ PASS |
| Real-desktop positive path | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop -v ./internal/supervisor/` | PASS 2.34s; layout=unpacked-appimage; revision zcode-builtin:44:e9c0674a8011…; session/create ok; kill(-pgid,0) ESRCH; `ps` shows zero app-server processes afterwards | ✓ PASS |
| Lint gate | `mise run lint` | 0 issues | ✓ PASS |
| Experiments gate | `mise run test-experiments` | tests 22, pass 22, fail 0 | ✓ PASS |
| Dependency gate | `mise run tidy-diff` | EXIT=0 (no drift; 1 require line pinned to acp-go-sdk v0.20.0 per the Phase-4-updated gate) | ✓ PASS |
| Build + vet | `go build ./... && go vet ./...` | both EXIT=0 | ✓ PASS |
| Secret scan | `bash experiments/scan-secrets.sh` | EXIT=0 | ✓ PASS |
| Test enumeration (no disabled tests) | `go test -list '.*'` × 4 packages | 68 tests enumerated (zcode 20, fake 23, supervisor 13, turn 12); 0 skipped/disabled in the default suite (only the 2 documented env-guard `t.Skip` in realdesktop_test.go under the tag without env) | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| (none declared) | `find scripts -path '*/tests/probe-*.sh'` → none; PLAN/SUMMARY grep for probe-*.sh → none | No conventional or declared probe scripts exist in this phase; the executable checks are the Go suite + mise tasks, all run first-hand above | N/A |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
| ----------- | ------------ | ----------- | ------ | -------- |
| BRIDGE-01 | 03-01, 03-02, 03-09 | JSONL codec for the app-server dialect (no `jsonrpc` key, its error codes, ordered single-writer output) with unit tests | ✓ SATISFIED | `internal/zcode/` green under -race first-hand; 14 named invariant tests PASS; envelope structural defense intact through the additive Phase 4-5 constants |
| BRIDGE-02 | 03-01, 03-03, 03-06, 03-09 | Fake app-server (in-process over pipes + standalone binary) and golden JSONL transcript tests running in CI | ✓ SATISFIED | `internal/zcode/fake/` + `cmd/fake-app-server/`, 19+19 corpus, TestGolden 19/19 + binary tier PASS first-hand, CI runs the same mise tasks (pr-sanity.yml) |
| BRIDGE-03 | 03-04, 03-07, 03-09 | Desktop discovery + process supervisor: locate, spawn ELECTRON_RUN_AS_NODE=1, kill whole group on EOF/exit, fail loudly when not found | ✓ SATISFIED | `internal/supervisor/` 13 tests green first-hand + real-desktop positive path PASS first-hand (2.34s, ESRCH zero orphans) |
| BRIDGE-04 | 03-05, 03-08, 03-09 | Turn state machine lifting fire-and-forget session/send into a synchronous prompt lifecycle, multi-signal completion, cancel | ✓ SATISFIED | `internal/turn/` 12 tests green first-hand incl. exactly-once + cancel + -32010 guard (both sides) |

Orphaned requirements: none — REQUIREMENTS.md maps exactly BRIDGE-01..04 to Phase 3 (traceability lines 120-123), and all four appear in the plans' `requirements` frontmatter. All four marked Complete, consistent with this verification.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/zcode/*_test.go (4 files) | BRIDGE-01 | 20 | 0 | No | Value + behavioral (verbatim id echo, type non-conflation, ordering, bypass, fan-out ordering) | STRONG |
| internal/zcode/fake/*_test.go (5 files) | BRIDGE-02 | 23 | 0 | No — goldens are hand-authored spec transcriptions (fixed synthetic ids), not captures of the SUT | Behavioral (byte-deterministic replay, EOF failure signature, fail-loud loader) | STRONG |
| internal/supervisor/*_test.go (5 files) | BRIDGE-03 | 13 | 0 default (realdesktop: 2 env-guarded skips, by design) | No | Behavioral (ESRCH no-survivors, observed child env, crash-loop trip, live-install resolve) | STRONG |
| internal/turn/*_test.go (3 files) | BRIDGE-04 | 12 | 0 | No | Behavioral (exactly-once counters, -32010 numeric mapping both sides, expiry recorded) | STRONG |

**Disabled tests on requirements:** 0 (the 2 realdesktop `t.Skip` are the documented env-guard, not requirement-linked disabled tests; the guarded run was executed green first-hand today)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (`check.decision-coverage-verify`: skipped=false, total=6, honored=6, not_honored=[] — non-blocking gate.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX debt markers: 0 across all 30 phase-3 files + 9 new corpus pairs | — | — |
| (none) | — | TODO/HACK/PLACEHOLDER: 0; not-implemented/coming-soon phrases: 0 | — | — |
| (none) | — | Empty implementations / console-log-only: 0 (`return nil` hits are normal Go error returns in substantive functions) | — | — |

Debt-marker gate: clean. The prior review disposition's CR-01 + WR-01 fixes remain in the tree (group-kill crash-loop case, stream-death call resolution — both covered by green tests). The 6 advisory review items remain routed to named later-phase seams per 03-REVIEW-DISPOSITION.md; none is a must-have gap, and the two routed to Phase 4/5 seams (WR-02 demux lock, IN-01/IN-02 observer contract) are now implemented by the stacked work with green tests.

### Human Verification Required

N/A — Infrastructure/foundation phase (internal codec/fake/supervisor/turn packages; no user-facing UI). All acceptance criteria verifiable programmatically, and every behavior-dependent truth has a passing behavioral test re-executed first-hand on the current tree.

The single harvested `<human-check>` block (03-07 real-desktop manual run) remains **discharged with machine evidence**: executed by the executor 2026-11-19, re-executed by the Nyquist audit 2026-11-20, re-executed by the prior verification (PASS 2.04s), and re-executed first-hand during this regeneration (PASS 2.34s with observed layout/revision and ESRCH zero orphans — see Behavioral Spot-Checks). The no-install diagnostics path (the other side of SC3) is covered by `TestDiscoverNoInstallListsEveryProbe`, green first-hand.

### Gaps Summary

None. The prior verification's 43/43 verdict survives the Phase 4-5 stacked-branch churn: every net change to phase-3 files is an additive extension (observer fan-outs, reannounce scripting, transcript sink, ExplicitRootInstall, corpus growth 10 → 19), no phase-3 invariant was weakened, and every behavior-dependent truth was re-proven first-hand under -race on the current tree (including the real-desktop positive path and the full E2E exec chain). All declared artifacts exist, are substantive, and are wired; the 4 artifact-checker misses and 1 key-link miss are the previously documented literal-pattern false positives, each re-verified by direct read on the current tree. Requirements BRIDGE-01..04 fully satisfied; no orphaned requirements; no debt markers; lint/experiments/tidy-diff/secret-scan/build/vet all exit 0 first-hand. The go.mod dependency (acp-go-sdk v0.20.0) is Phase-4 work asserted by the updated tidy-diff gate — intentional, outside phase-3 scope. Nothing deferred — no gaps arose for Step 9b filtering.

---

_Verified: 2026-11-20T11:28:22Z_
_Verifier: Claude (gsd-verifier)_
