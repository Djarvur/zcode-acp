---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
verified: 2026-11-19T22:01:45Z
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
  - internal/zcode/fake/testdata/golden/completion-idle-patch.golden
  - internal/zcode/fake/testdata/golden/completion-quiet-window.golden
  - internal/zcode/fake/testdata/golden/credential-cipher-scripted.golden
  - internal/zcode/fake/testdata/golden/envelope-negatives.golden
  - internal/zcode/fake/testdata/golden/eof-drain.golden
  - internal/zcode/fake/testdata/golden/error-32010.golden
  - internal/zcode/fake/testdata/golden/happy-turn.golden
  - internal/zcode/fake/testdata/golden/runtime-preferences.golden
  - internal/zcode/fake/testdata/golden/unknown-host-method.golden
  - internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json
  - internal/zcode/fake/testdata/scenarios/completion-idle-patch.json
  - internal/zcode/fake/testdata/scenarios/completion-quiet-window.json
  - internal/zcode/fake/testdata/scenarios/credential-cipher-scripted.json
  - internal/zcode/fake/testdata/scenarios/envelope-negatives.json
  - internal/zcode/fake/testdata/scenarios/eof-drain.json
  - internal/zcode/fake/testdata/scenarios/error-32010.json
  - internal/zcode/fake/testdata/scenarios/happy-turn.json
  - internal/zcode/fake/testdata/scenarios/runtime-preferences.json
  - internal/zcode/fake/testdata/scenarios/unknown-host-method.json
  - internal/zcode/roundtrip_test.go
  - internal/zcode/writer.go
  - internal/zcode/writer_test.go
covered_digest: "v4:sha256:00fe3d4a894305322e59cb9c2e4386830a5060784721cd91afd295f47f826317"
behavior_unverified: 0 # Count of PRESENT_BEHAVIOR_UNVERIFIED truths (present + wired, behavior not exercised); each is detailed in behavior_unverified_items below (and in human_verification when status is human_needed)
overrides_applied: 0 # Count of PASSED (override) items included in score
---

# Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor — Verification Report

**Phase Goal:** As a later-phase implementer on this repo, I want to the wrapper's back leg — a JSONL codec for the app-server dialect, a scriptable fake app-server with golden transcripts in CI, a desktop discovery + process supervisor, and a turn state machine, so that every later phase is testable without the real desktop bundle.
**Verified:** 2026-11-19T22:01:45Z
**Status:** passed
**Re-verification:** No — initial verification
**Mode:** mvp (user-story goal validated via `user-story.validate` → `true`)

## User Flow Coverage

User story: «As a later-phase implementer on this repo, I want to the wrapper's back leg — a JSONL codec for the app-server dialect, a scriptable fake app-server with golden transcripts in CI, a desktop discovery + process supervisor, and a turn state machine, so that every later phase is testable without the real desktop bundle.»

The "user" is a later-phase implementer; the outcome clause is «every later phase is testable without the real desktop bundle». Each step below is what that implementer does, with first-hand evidence.

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Run the codec against dialect frames | Frames without a `jsonrpc` key, dialect error codes, ordered single-writer output | `internal/zcode/envelope.go:86` (structural: no framing-key field), `TestEncodeOmitsJSONRPC` (`internal/zcode/roundtrip_test.go:34`, asserts no "jsonrpc" byte in outbound line), error-code constants `envelope.go:156-165` (-32700/-32600/-32601/-32002/-32010/-32020/-32021/-32022), `TestWriterOrdering` green | ✓ |
| Replay golden transcripts with no desktop present | 10 scenarios + 10 goldens replay deterministically over pipes | `TestGolden` 10/10 subtests PASS first-hand (`go test -race -count=1 -run TestGolden ./internal/zcode/fake/`); corpus at `internal/zcode/fake/testdata/{scenarios,golden}/` (10+10 files on disk) | ✓ |
| Use the standalone fake binary tier | Binary serves `--scenario` over stdio; frames proven over a real child process | `cmd/fake-app-server/main.go` (thin main, `--scenario` required, stderr-only diagnostics), `TestFakeBinarySpawnPath` PASS first-hand (helper reexec, `GO_WANT_HELPER_PROCESS`) | ✓ |
| Point discovery at an install — or at nothing | First-match-wins per the Phase-1 matrix; empty result names every probe; /tmp/.mount_* rejected | `TestDiscoverFixtureTree`, `TestDiscoverNoInstallListsEveryProbe`, `TestDiscoverOrderFirstMatchWins`, `TestDiscoverRejectsVolatileMounts`, `TestDiscoverFindsNewestBuiltin` all PASS first-hand | ✓ |
| Supervise a child; kill the whole group on exit | Zero surviving group members (grandchildren included); allowlisted+scrubbed env on the child's observed env | `TestGroupKillNoSurvivors` PASS, `TestSpawnEnv` PASS (asserts the child's OBSERVED env), `TestRestartWindow` PASS (3/60 s sliding window) | ✓ |
| Drive a synchronous turn; cancel mid-turn | Fire-and-forget `session/send` resolves exactly once; cancel resolves cancelled exactly once | `TestHappyTurnCompletedExactlyOnce`, `TestCompletionVariants` (terminal/idle-patch/quiet-window), `TestCancelMidTurnAnswersOnce`, `TestDoubleSendRejectedWrapperSide` (-32010) all PASS first-hand | ✓ |
| Compose the whole chain | Supervisor + fake binary + Conn + Machine in one test | `TestEndToEndExecChain` PASS with both subtests (`happy_turn_completes_exactly_once`, `cancel_mid-turn_resolves_cancelled_exactly_once`) | ✓ |
| Outcome: every later phase testable without the desktop bundle | Full gate green on a machine that HAS the desktop — proving nothing in the suite needs it | `mise run ci` EXIT=0 first-hand (build/vet/lint 0 issues/go test -race all packages/experiments 22/22) on this machine (ZCode 3.18.2 installed); the real-desktop path is `-tags realdesktop` + `ZCODE_ACP_REAL_E2E=1` guarded — default suite has zero skips and never compiles the file | ✓ |

## Goal Achievement

### Observable Truths

Roadmap truths (RT) are the contract; plan truths (Pxx-y) are the plan-frontmatter detail merged per Step 2c. Evidence is first-hand re-execution unless noted.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| RT1 | The codec round-trips the app-server JSONL dialect: frames without a `jsonrpc` key, the dialect's own error codes, ordered single-writer output — a JSON-RPC-shaped frame never reaches the child | ✓ VERIFIED | `envelope.go` Envelope struct has no framing-key field (structural defense, :86-108); `TestEncodeOmitsJSONRPC` + `TestDecodeRoundTrip` + `TestDecodeEchoesIDTypes` green; `TestWriterOrdering` green; corpus `error-32010` + `envelope-negatives` replay green; lint 0 issues |
| RT2 | CI runs golden JSONL transcript tests against the scriptable fake app-server (in-process over pipes plus a standalone binary), no desktop bundle present | ✓ VERIFIED | `TestGolden` 10/10 PASS + `TestFakeBinarySpawnPath` PASS first-hand; `mise run ci` EXIT=0 first-hand; `.github/workflows/pr-sanity.yml` runs the same mise tasks (build/vet/lint/test/test-experiments/tidy-diff + govulncheck + secret-scan); realdesktop tier build-tag+env guarded |
| RT3 | The supervisor locates the install per the Phase 1 matrix, spawns the child with the correct environment, kills the whole process group on EOF/exit leaving no orphans; no-install fails with diagnostics naming what was searched | ✓ VERIFIED | 5 discovery tests PASS (incl. `TestDiscoverNoInstallListsEveryProbe`, `TestDiscoverRejectsVolatileMounts`); `TestSpawnEnv` (observed child env), `TestGroupKillNoSurvivors` (kill(-pgid,0) ESRCH), `TestRestartWindow` PASS; real-desktop positive path re-executed first-hand: `TestRealDesktop` PASS 2.04s — layout=unpacked-appimage, revision zcode-builtin:44:e9c0674a…, session/create ok, ESRCH zero orphans; `pgrep`/`ps` after: zero real app-server processes |
| RT4 | Against the fake app-server, a fire-and-forget `session/send` is lifted into a synchronous turn that completes exactly once on multi-signal turn-completion detection and honors cancel mid-turn | ✓ VERIFIED | `TestHappyTurnCompletedExactlyOnce`, `TestCompletionVariants`, `TestTerminalCarrierTolerance`, `TestCancelMidTurnAnswersOnce`, `TestCancelRacingTerminalResolvesOnce`, `TestCompletionCorpus`, `TestCredentialCipherExpiryRecorded` all PASS first-hand under -race |
| P01-1 | Conn vs in-process fake over io.Pipe completes happy-turn end-to-end (session/create with reverse request answered, session/send acked accepted:true, notifications in wire order) | ✓ VERIFIED | `TestHappyTurnRoundTrip` + `TestGoldenHappyTurn` green (fake/roundtrip_test.go, 475 lines) |
| P01-2 | Every byte to the child is a complete newline-delimited JSON envelope; no outbound frame can carry a JSON-RPC framing key (structural) | ✓ VERIFIED | `TestEncodeOmitsJSONRPC` green; `envelope.go` struct comment + `bytes.Contains(line, "jsonrpc")` negative assertion |
| P01-3 | Reverse-request replies travel a lane that never queues behind ordinary calls (two-lane writer) | ✓ VERIFIED | `writer.go:25-73` bypass channel checked before queued writes; `Conn.Reply` (`call.go:215-219`) + `Conn.Stop` enqueue bypass; `TestStopJumpsQueue` green |
| P01-4 | Golden replay byte-deterministic (fixed ids); any frame drift or missing frame fails — never normalized away | ✓ VERIFIED | `TestGolden` 10/10 green; `golden_test.go:175` — drift/missing-fixture fails; `-update` regeneration path in `golden.go` |
| P02-1 | WR-02: a numeric-id reverse request colliding with an in-flight call id routes to the router, NEVER resolves the pending call | ✓ VERIFIED | `TestDemuxReverseRequestNeverResolvesPendingCall` green |
| P02-2 | Ids correlate by exact value AND type: string 5 ≠ integer 5 on the wire | ✓ VERIFIED | `TestDemuxIdTypeNonConflation` + `TestIDKeyTypeTagging` + `TestCallReplyEchoesIDVerbatim` green |
| P02-3 | Single-writer ordering: concurrent writers produce complete, never-interleaved frames | ✓ VERIFIED | `TestWriterOrdering` green (32 concurrent writers per the validation map) |
| P02-4 | Bypass precedence: session/stop issued while the ordered queue holds pending writes is on the wire after at most the in-flight write | ✓ VERIFIED | `TestStopJumpsQueue` green |
| P02-5 | Raised-line behavior: 128 KiB line survives intact; malformed JSON line skipped with warning, stream continues | ✓ VERIFIED | `TestLineBufferLargeLineIntact` + `TestLineBufferMalformedLineBetweenValidLines` green; `conn.go:176-177` `sc.Buffer(make([]byte,0,readBufInitial), readBufMax)` |
| P02-6 | D-05 deadline honesty: runtime preferences answered within deadline; mid-turn credential-cipher known-but-NOT-answered; expiry recorded observably | ✓ VERIFIED | `TestRouterRuntimePreferencesDeadline` + `TestRouterCredentialCipherNeverAnsweredExpiryRecorded` + `TestRouterUnknownMethodAnswered32601` green |
| P03-1 | The 10-scenario corpus lives under testdata/scenarios/ and every scenario replays green against the codec Conn under -race | ✓ VERIFIED | `TestGolden` 10/10 PASS first-hand; 10 scenario files + 10 goldens on disk |
| P03-2 | Every golden replays byte-deterministically; `-update` regenerates; drift fails | ✓ VERIFIED | `TestGolden` green twice this session (ci run + named run); compare path in `golden.go` |
| P03-3 | The scenario loader fails loud: invalid file rejected with an error naming the offending field | ✓ VERIFIED | 11 `TestLoadScenarioRejectsInvalid*` tests enumerated and green (missing name, rule without method, unknown side-effect key, negative times, …) |
| P03-4 | Exit-0-with-zero-lines is a FAILURE in-process, never an empty success | ✓ VERIFIED | `TestEOFSignature` green (`internal/zcode/fake/`); exec-tier twin `TestEOFSignatureChild` PASS first-hand |
| P04-1 | Discovery implements the Phase 1 recommended order exactly; first-match-wins; probes do not merge | ✓ VERIFIED | `TestDiscoverOrderFirstMatchWins` + `TestDiscoverFixtureTree` green |
| P04-2 | Empty result names EVERY probe attempted with its result — the diagnostics are the product | ✓ VERIFIED | `TestDiscoverNoInstallListsEveryProbe` green (8 probes: override, xdg-entry, appimage-glob, unpacked-glob, dpkg, opt-path, rpm, builtin-runtime) |
| P04-3 | Volatile /tmp/.mount_* decoys recorded-and-rejected; localhost listeners never probed | ✓ VERIFIED | `TestDiscoverRejectsVolatileMounts` green |
| P04-4 | Builtin revision computed, never hardcoded: sha256 over the resolved absolute PATH STRING | ✓ VERIFIED | `discover.go:151-162` (`filepath.EvalSymlinks` → `sha256.Sum256([]byte(abs))`); `TestBuiltinRevisionFormula` + `TestDiscoverFindsNewestBuiltin` green; real-desktop run printed `zcode-builtin:44:e9c0674a…` live |
| P05-1 | Machine.Run lifts one session/send into a synchronous turn: returns StopReason exactly once, after a completion signal — never before, never twice | ✓ VERIFIED | `TestHappyTurnCompletedExactlyOnce` green; E2E subtest `happy_turn_completes_exactly_once` PASS |
| P05-2 | Completion classification multi-signal and ordered (terminal → idle patch → quiet-window); each variant resolves exactly once | ✓ VERIFIED | `TestCompletionVariants` green (three variants as subtests) |
| P05-3 | Terminal classifier matches turnId + kind across known carriers; carrier-tolerant by design | ✓ VERIFIED | `TestTerminalCarrierTolerance` green |
| P05-4 | TurnWatcher is an interface with a polling implementation; quiet-window gates with send-latency floor 3x quiet-ms after the send ack; subscribe watcher is a reserved slot, NOT implemented | ✓ VERIFIED | `watcher.go:26-27` `DefaultFloorMultiple = 3` injectable; `TestTurnWatcherDeliversUntilStopped` green; no subscribe implementation exists (grep: interface + PollWatcher only) |
| P05-5 | D-04: the machine is bound to ONE app-server sessionId on the ONE shared Conn (1:1 ACP session mapping) | ✓ VERIFIED | `machine.go:147-149` documented single-session binding; enforced in construction; E2E asserts the create reply's sessionId against the binding |
| P06-1 | Standalone fake binary serves a JSON scenario over its own stdio: --scenario required, frames on stdin/stdout, diagnostics stderr-only | ✓ VERIFIED | `cmd/fake-app-server/main.go` (usage error → `fake.ExitUsage`, stderr slog); `TestFakeBinarySpawnPath` PASS first-hand |
| P06-2 | EOF contract: on stdin EOF the fake drains ~100 ms (eofDrainMs) then exits 0; a request just before EOF is never answered; drain observable in tests | ✓ VERIFIED | `TestFakeBinarySpawnPath` + eof-drain corpus scenario + golden green; drain window asserted in `binary_test.go` |
| P06-3 | Exec-plumbing proven against a REAL child: helper reexec drives frames both directions with the child in its own process group | ✓ VERIFIED | `TestFakeBinarySpawnPath` PASS (GO_WANT_HELPER_PROCESS reexec) |
| P06-4 | Exit-0-with-zero-lines failure signature enforced at the exec tier | ✓ VERIFIED | `TestEOFSignatureChild` PASS first-hand |
| P07-1 | Child spawns with the canonical contract: argv [binary, entrypoint, app-server, --stdio]; env ELECTRON_RUN_AS_NODE=1 + ZCODE_BUILTIN_PROVIDER_CONFIG_FILE | ✓ VERIFIED | `TestSpawnArgvShape` green; `TestSpawnEnv` green |
| P07-2 | Child env is an explicit allowlist (PATH+HOME) + two ZCode vars, proxy vars scrubbed — asserted on the child's OBSERVED environment | ✓ VERIFIED | `TestSpawnEnv` asserts via the `TestSpawnEnvHelper` reexec child's own observed env |
| P07-3 | Whole process group dies on EOF/exit: staged shutdown (stdin end → wait/grace → Kill(-pgid, SIGKILL) → reap) leaves zero survivors incl. grandchildren, ESRCH-verified | ✓ VERIFIED | `TestGroupKillNoSurvivors` green; `supervise.go:168-192` staged shutdown calls `killGroup` (`spawn.go:306-308` `syscall.Kill(-pgid, SIGKILL)`, ESRCH tolerated); real-desktop run re-verified ESRCH first-hand |
| P07-4 | Crash-window restart: 3 attempts within sliding 60 s then fail loud; stable run resets the counter; injectable for tests | ✓ VERIFIED | `TestRestartWindow` green — includes the crash-loop-trip-leaves-zero-survivors case added by review fix 6794626/5bbb38f |
| P07-5 | Health watch = stdin liveness + child exit; NO protocol liveness frame is invented | ✓ VERIFIED | `supervise.go:6-8,111-112` explicit contract; grep confirms no ping/liveness frame code |
| P08-1 | Cancel mid-turn: session/stop through the bypass lane immediately; turn resolves cancelled EXACTLY once; second Cancel no-op; cancel-after-complete no-op | ✓ VERIFIED | `TestCancelMidTurnAnswersOnce` + `TestCancelRacingTerminalResolvesOnce` green; `Conn.Stop` bypass enqueue (`call.go:188-213`) |
| P08-2 | A second concurrent session/send is rejected wrapper-side with a clean error mapping the dialect code -32010 (numeric code, never message text) | ✓ VERIFIED | `TestDoubleSendRejectedWrapperSide` green; `machine.go:130-134,229` maps `zcode.CodeConcurrency` numerically |
| P08-3 | Corpus-level completion: completion-idle-patch and completion-quiet-window replay against the codec Conn, each resolving exactly once | ✓ VERIFIED | `TestCompletionCorpus` green (loads the JSON corpus via `corpusScenarioPath`) |
| P08-4 | D-05 honesty at the turn tier: scripted credential-cipher exchange ends with expiry RECORDED, turn NOT falsely resolved — degradation observable, never swallowed | ✓ VERIFIED | `TestCredentialCipherExpiryRecorded` green |
| P09-1 | The FULL chain runs in one test: supervisor spawns the standalone fake binary through the real spawn path (process group, allowlisted env, drained stderr), Conn rides the exec pipes, Machine.Run completes the happy turn exactly once | ✓ VERIFIED | `TestEndToEndExecChain` PASS first-hand; `TestMain` builds the binary via `go build` into `t.TempDir()` (`e2e_test.go:119-126`) |
| P09-2 | Cancel works over the exec chain: mid-turn cancel resolves cancelled exactly once; staged shutdown leaves zero surviving group members (ESRCH) | ✓ VERIFIED | E2E subtest `cancel_mid-turn_resolves_cancelled_exactly_once` PASS (second supervised cycle serving cancel-mid-turn.json — documented deviation, keeps every must-have) |
| P09-3 | The whole phase gate is green: mise run ci with the corpus replayed inside; go.mod/go.sum untouched (zero new dependencies) | ✓ VERIFIED | `mise run ci` EXIT=0 first-hand (lint 0 issues; experiments 22/22 pass, 0 fail/skip); `go.mod` is module-line-only (5 lines); CI runs `mise run tidy-diff` (pr-sanity.yml:89-91) |

**Score:** 43/43 truths verified (0 present, behavior-unverified)

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
| ----------- | ------------ | ----------- | ------ | -------- |
| BRIDGE-01 | 03-01, 03-02, 03-09 | JSONL codec for the app-server dialect (no `jsonrpc` key, its error codes, ordered single-writer output) with unit tests | ✓ SATISFIED | `internal/zcode/` (envelope/conn/writer/call, 10498 lines phase-total), 18 codec tests green |
| BRIDGE-02 | 03-01, 03-03, 03-06, 03-09 | Fake app-server (in-process over pipes + standalone binary) and golden JSONL transcript tests running in CI | ✓ SATISFIED | `internal/zcode/fake/` + `cmd/fake-app-server/`, 10+10 corpus, 20 fake tests green, CI runs them (mise tasks mirrored in pr-sanity.yml) |
| BRIDGE-03 | 03-04, 03-07, 03-09 | Desktop discovery + process supervisor: locate, spawn ELECTRON_RUN_AS_NODE=1, kill whole group on EOF/exit, fail loudly when not found | ✓ SATISFIED | `internal/supervisor/` (discover/spawn/supervise), 13 tests green + real-desktop positive path PASS first-hand |
| BRIDGE-04 | 03-05, 03-08, 03-09 | Turn state machine lifting fire-and-forget session/send into a synchronous prompt lifecycle, multi-signal completion, cancel | ✓ SATISFIED | `internal/turn/` (machine/watcher), 10 tests green incl. exactly-once + cancel + -32010 guard |

Orphaned requirements: none — REQUIREMENTS.md maps exactly BRIDGE-01..04 to Phase 3, and all four appear in the plans' `requirements` frontmatter. Traceability table marks all four Complete, consistent with this verification.

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/zcode/envelope.go` | ID, Envelope (no framing-key field), WireError, method + error-code constants | ✓ VERIFIED | 168 lines; contains session/requestRuntimePreferences + full code table |
| `internal/zcode/conn.go` | Conn: read loop with raised scanner buffer, three-way demux, reverse-request router with D-05 policy | ✓ VERIFIED | 405 lines; `sc.Buffer` raised-line at :176-177 (plan's literal pattern `Scanner.Buffer` was a naming mismatch — construct verified present by direct read) |
| `internal/zcode/writer.go` | Single writer goroutine, two lanes (ordered + bypass) | ✓ VERIFIED | 156 lines; bypass channel :31,43,63,73 |
| `internal/zcode/call.go` | Call/Notify/Reply/ReplyError, pending table, Stop helper | ✓ VERIFIED | 249 lines; Stop :188+, Reply bypass :215-219 |
| `internal/zcode/{roundtrip,conn,writer,call}_test.go` | Codec unit tests per plan | ✓ VERIFIED | 93+852+164+234 lines; all green under -race |
| `internal/zcode/fake/fake.go` | Serve(rw, script): scripted matching, delays, events, blocking reverse requests | ✓ VERIFIED | 466 lines |
| `internal/zcode/fake/scenario.go` + `scenario_test.go` | JSON scenario schema + fail-loud load; loader validation tests | ✓ VERIFIED | 289+253 lines; 11 rejection tests green |
| `internal/zcode/fake/golden.go` + `golden_test.go` | Transcript record/compare with -update; corpus replay + EOF signature | ✓ VERIFIED | 65+422 lines; TestGolden 10/10 green |
| `internal/zcode/fake/testdata/scenarios/` | 10 corpus scenario files | ✓ VERIFIED | 10 JSON files on disk (tool "Not a file: directory" is a file-only checker — contents verified by listing + replay) |
| `internal/zcode/fake/testdata/golden/` | 10 golden transcripts | ✓ VERIFIED | 10 .golden files on disk; replay green |
| `internal/zcode/fake/binary_test.go` | Reexec integration tests (spawn path, EOF drain, zero-lines signature) | ✓ VERIFIED | 488 lines; GO_WANT_HELPER_PROCESS pattern; both named tests PASS first-hand |
| `internal/zcode/fake/stdio.go` | stdio hosting of the scripting core (fake.Main/ExitUsage) | ✓ VERIFIED | 99 lines; used by cmd main |
| `cmd/fake-app-server/main.go` | Thin main: --scenario flag, stdio serving, stderr diagnostics, EOF drain exit | ✓ VERIFIED | 25 lines by design ("thin main over the same scripting core"); delegates to fake.Main |
| `internal/supervisor/discover.go` + `discover_test.go` | Ordered first-match discovery, full diagnostics, decoy rejection, BuiltinRevision | ✓ VERIFIED | 852+432 lines; 6 discovery tests green |
| `internal/supervisor/spawn.go` | SpawnArgv, ChildEnv, SpawnChild with Setpgid + 3 stdio pipes + capped stderr drain + group-kill Cancel | ✓ VERIFIED | 328 lines; Setpgid :225, killGroup :306-308 |
| `internal/supervisor/supervise.go` | Supervisor: health watch, staged shutdown, crash-window restart | ✓ VERIFIED | 193 lines; staged shutdown :168-192 wired to killGroup (plan's literal `contains: Kill` pattern lives in spawn.go — wiring verified by direct read; not an orphan) |
| `internal/supervisor/{spawn,supervise,realdesktop,e2e}_test.go` | Env/argv/group-kill/restart/realdesktop/E2E tests | ✓ VERIFIED | 295+515+249+456 lines; all green; realdesktop behind `//go:build realdesktop` + env guard |
| `internal/turn/machine.go` + `watcher.go` + `machine_test.go` | Machine (Run/Cancel, -32010 guard), TurnWatcher + PollWatcher, RED→GREEN tests | ✓ VERIFIED | 596+233+742 lines; 10 turn tests green |
| `internal/turn/cancel_test.go` | Cancel exactly-once, -32010 guard, corpus completion, cipher honesty | ✓ VERIFIED | 704 lines; all four test groups green |
| `internal/supervisor/e2e_test.go` | End-to-end exec-chain test | ✓ VERIFIED | 456 lines; TestEndToEndExecChain PASS (both subtests) |
| evidence/ (13 RED-proof JSONs) | TDD RED evidence for 03-01/02/03/04/05/07/08 + review CR-01/WR-01 | ✓ VERIFIED | All 13 tracked at HEAD; spot-checked 03-01, 03-04, 03-05, 03-review-cr01: exitCode 1 with planned assertion failures against skeletons ("not implemented (RED stub)") |

### Key Link Verification

gsd-tools `verify.key-links`: 8/9 plans all_verified; the one miss below was manually verified wired.

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/zcode/fake/roundtrip_test.go | internal/zcode/fake/fake.go | io.Pipe pairing | ✓ WIRED | tool-verified |
| internal/zcode/conn.go | internal/zcode/call.go | demux resolves pending by canonical id | ✓ WIRED | tool-verified |
| internal/zcode/call.go | internal/zcode/writer.go | Call/Notify ordered lane; Reply/Stop bypass lane | ✓ WIRED | tool-verified + direct read |
| internal/zcode/conn_test.go | internal/zcode/conn.go | demux + router policy via public Conn surface | ✓ WIRED | tool-verified |
| internal/zcode/fake/golden_test.go | testdata/scenarios/ | table-driven LoadScenario → Serve → CompareGolden | ✓ WIRED | tool-verified |
| testdata/scenarios/completion-quiet-window.json | fake/fake.go | session/messages poll answers | ✓ WIRED | tool-verified |
| internal/supervisor/discover_test.go | discover.go | Probers seams (t.TempDir fixtures) | ✓ WIRED | tool-verified |
| discover.go | docs/protocol/install-discovery.md | probe order/set/fields mirror matrix | ✓ WIRED | tool-verified |
| internal/turn/machine.go | zcode/conn.go / call.go | session/send via Call; Stop bypass; -32010 numeric mapping | ✓ WIRED | tool-verified (plans 05/08) + direct read |
| internal/turn/cancel_test.go | testdata/scenarios/cancel-mid-turn.json | corpus scenario replay via corpusScenarioPath | ✓ WIRED | tool pattern `testdata/scenarios` (slash form) missed the `filepath.Join("..","zcode","fake","testdata","scenarios",…)` construction at cancel_test.go:570-572 — link verified wired by direct read; TestCompletionCorpus green |
| supervisor/spawn.go | discover.go | SpawnArgv + ChildEnv consume Install | ✓ WIRED | tool-verified |
| supervisor/supervise.go | spawn.go | staged shutdown → killGroup | ✓ WIRED | tool-verified |
| supervisor/spawn_test.go | cmd/fake-app-server/main.go | fake binary as the CI child | ✓ WIRED | tool-verified |
| supervisor/e2e_test.go | supervise.go / machine.go / cmd main | Supervisor lifecycle + Machine over exec pipes + go build child | ✓ WIRED | tool-verified |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `TestEndToEndExecChain` | StopReason / sessionId | Real exec pipes to the built fake-app-server child (go build → t.TempDir) driven through SpawnChild + Conn + Machine | Yes — real process I/O, real frames on os-level pipes | ✓ FLOWING |
| `TestSpawnEnv` | Child's observed environment | Reexec helper child reads its own os.Environ and reports it back | Yes — asserted on the OBSERVED child env, not the builder's intention | ✓ FLOWING |
| `TestDiscover*` | Install candidates | Injected Probers seams over t.TempDir fixture trees | Yes — the seams are the documented injection point; production path consumes real probes (proven by the real-desktop run resolving the live install) | ✓ FLOWING |
| `TestRealDesktop` | Install root / builtinRevision | The actual ZCode 3.18.2 install on this machine (layout=unpacked-appimage) | Yes — re-executed first-hand, revision zcode-builtin:44:e9c0674a… printed | ✓ FLOWING |
| Golden corpus replay | Transcript frames | fake.Serve over io.Pipe from hand-authored scenario JSONs | Yes — the fake IS the intended data source for later phases (D-01); ids are synthetic by contract (sess_fix1/turn_fix1) | ✓ FLOWING |

No static returns, hardcoded literals, or mocks on any asserted path. The fake app-server is the product (a scriptable stand-in), not a mock hiding a missing implementation — the E2E chain proves the composition over real exec pipes.

### Behavioral Spot-Checks

All commands re-run first-hand this verification. The full workspace suite ran exactly once (`mise run ci`); named tests below are single named/package-scoped runs.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Full phase gate | `mise run ci` | EXIT=0; lint 0 issues; all Go packages ok under -race; experiments 22 pass / 0 fail / 0 skipped | ✓ PASS |
| Golden corpus (10 scenarios) | `go test -race -count=1 -run TestGolden -v ./internal/zcode/fake/` | 10/10 subtests PASS (+TestGoldenHappyTurn) | ✓ PASS |
| Standalone binary tier | `go test -race -count=1 -run 'TestFakeBinarySpawnPath|TestEOFSignatureChild' -v ./internal/zcode/fake/` | both PASS (1.12s / 1.14s) | ✓ PASS |
| Discovery (positive/negative/decoys) | `go test -race -count=1 -run TestDiscover -v ./internal/supervisor/` | 5/5 PASS | ✓ PASS |
| Spawn env / group kill / restart window | `go test -race -count=1 -run 'TestSpawnEnv|TestGroupKill|TestRestartWindow' -v ./internal/supervisor/` | 4/4 PASS (incl. TestSpawnEnvHelper, TestGroupKillNoSurvivors) | ✓ PASS |
| Turn machine (exactly-once, cancel, -32010, honesty) | `go test -race -count=1 -v ./internal/turn/` | 10/10 PASS | ✓ PASS |
| E2E exec chain | `go test -race -count=1 -run TestEndToEndExecChain -v ./internal/supervisor/` | PASS (happy + cancel subtests) | ✓ PASS |
| Real-desktop positive path | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop -v ./internal/supervisor/` | PASS 2.04s; layout=unpacked-appimage; revision zcode-builtin:44:e9c0674a8011…; session/create ok; kill(-pgid,0) ESRCH; `ps aux` shows zero app-server processes afterwards | ✓ PASS |
| Secret scan at HEAD + negative control | `bash experiments/scan-secrets.sh`; then raw pattern matches vs allowlist filter | EXIT=0; 3 raw matches all carrying the exact allowlisted marker; 0 survivors after `grep -vF` filter; a decoy body (`enc:v2:os:different-body…`) matches the regex and would fail the gate — allowlist is exact-string, regex intact | ✓ PASS |
| Test enumeration (no disabled tests) | `go test -list '.*'` all 4 packages | 61 tests enumerated; 0 skipped/disabled in default suite (only the 2 documented env-guard `t.Skip` in realdesktop_test.go under the tag without env) | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| (none declared) | `find scripts -path '*/tests/probe-*.sh'` → none; PLAN/SUMMARY grep for probe-*.sh → none | No conventional or declared probe scripts exist in this phase; the executable checks are the Go suite + mise tasks, all run first-hand above | N/A |

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/zcode/*_test.go (4 files) | BRIDGE-01 | 18 | 0 | No | Value + behavioral (verbatim id echo, type non-conflation, ordering, bypass) | STRONG |
| internal/zcode/fake/*_test.go (5 files) | BRIDGE-02 | 20 | 0 | No — goldens are hand-authored spec transcriptions (fixed synthetic ids, corpus table provenance), not captures of the SUT; `-update` exists but fixtures' provenance is the dialect docs | Behavioral (byte-deterministic replay, EOF failure signature) | STRONG |
| internal/supervisor/*_test.go (5 files) | BRIDGE-03 | 13 | 0 default (realdesktop: 2 env-guarded skips, by design) | No | Behavioral (ESRCH no-survivors, observed child env, crash-loop trip) | STRONG |
| internal/turn/*_test.go (3 files) | BRIDGE-04 | 10 | 0 | No | Behavioral (exactly-once counters, -32010 numeric mapping, expiry recorded) | STRONG |

**Disabled tests on requirements:** 0 (the 2 realdesktop `t.Skip` are the documented env-guard, not requirement-linked disabled tests; the guarded run was executed green first-hand)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (`check.decision-coverage-verify`: skipped=false, total=6, honored=6, not_honored=[] — non-blocking gate.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none) | — | TBD/FIXME/XXX debt markers: 0 across all 30 phase files | — | — |
| (none) | — | TODO/HACK/PLACEHOLDER: 0; not-implemented phrases: 0 | — | — |
| (none) | — | Empty implementations / console-log-only: 0 (`return nil` hits are normal Go error returns in substantive functions) | — | — |

Debt-marker gate: clean. Review findings CR-01 (critical) + WR-01 were fixed TDD (commits 6794626, 5bbb38f, 2e2b299, ff96773 — all verified in git log); the 6 remaining open review items are advisory, each routed to a named later-phase seam in 03-REVIEW-DISPOSITION.md (WR-02→Ph4, WR-03→Ph5, WR-04→Ph5, IN-01→Ph4, IN-02→Ph4, IN-03→Ph7). They are not must-have gaps.

### Human Verification Required

N/A — Infrastructure/foundation phase (internal codec/fake/supervisor/turn packages; no user-facing UI). All acceptance criteria verifiable programmatically, and every behavior-dependent truth has a passing behavioral test.

The single harvested `<human-check>` block (03-07 real-desktop manual run) is **discharged with machine evidence**: executed by the executor 2026-11-19 (2.529s), re-executed by the Nyquist audit 2026-11-20 (2.32s), and re-executed first-hand during this verification (PASS 2.04s with observed layout/revision and ESRCH zero orphans — see Behavioral Spot-Checks). The no-install diagnostics path (the other side of SC3) is covered by `TestDiscoverNoInstallListsEveryProbe`.

### Gaps Summary

None. All 43 must-have truths verified with first-hand machine evidence; all declared artifacts exist, are substantive, and are wired; all key links verified (4 gsd-tools literal-pattern misses were individually re-verified by direct reads and are documented above — none is a real gap); requirements BRIDGE-01..04 fully satisfied; no orphaned requirements; no debt markers; `mise run ci` exit 0; secret scan clean with the negative control proving the regex intact; real-desktop human-check discharged with machine evidence. Nothing deferred — no gaps arose for Step 9b filtering.

---

_Verified: 2026-11-19T22:01:45Z_
_Verifier: Claude (gsd-verifier)_
