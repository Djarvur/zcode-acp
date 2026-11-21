---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
verified: 2026-11-21T06:03:18Z
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
  - internal/zcode/fake/testdata/golden/auth-apikey.golden
  - internal/zcode/fake/testdata/golden/cancel-mid-turn.golden
  - internal/zcode/fake/testdata/golden/complete-then-cancel.golden
  - internal/zcode/fake/testdata/golden/completion-idle-patch.golden
  - internal/zcode/fake/testdata/golden/completion-quiet-window.golden
  - internal/zcode/fake/testdata/golden/credential-cipher-scripted.golden
  - internal/zcode/fake/testdata/golden/envelope-negatives.golden
  - internal/zcode/fake/testdata/golden/eof-drain.golden
  - internal/zcode/fake/testdata/golden/error-32010.golden
  - internal/zcode/fake/testdata/golden/full-cycle.golden
  - internal/zcode/fake/testdata/golden/happy-turn.golden
  - internal/zcode/fake/testdata/golden/newsession-orphan-close.golden
  - internal/zcode/fake/testdata/golden/no-reasoning.golden
  - internal/zcode/fake/testdata/golden/part-delta-streaming.golden
  - internal/zcode/fake/testdata/golden/permission-allow.golden
  - internal/zcode/fake/testdata/golden/permission-cancel.golden
  - internal/zcode/fake/testdata/golden/reannounce-dedupe.golden
  - internal/zcode/fake/testdata/golden/runtime-headers-answer.golden
  - internal/zcode/fake/testdata/golden/runtime-preferences.golden
  - internal/zcode/fake/testdata/golden/send-refused-32010.golden
  - internal/zcode/fake/testdata/golden/session-lifecycle.golden
  - internal/zcode/fake/testdata/golden/setmodel-followup.golden
  - internal/zcode/fake/testdata/golden/tool-lifecycle.golden
  - internal/zcode/fake/testdata/golden/unknown-host-method.golden
  - internal/zcode/fake/testdata/golden/user-input-decline.golden
  - internal/zcode/fake/testdata/scenarios/auth-apikey.json
  - internal/zcode/fake/testdata/scenarios/cancel-mid-turn.json
  - internal/zcode/fake/testdata/scenarios/complete-then-cancel.json
  - internal/zcode/fake/testdata/scenarios/completion-idle-patch.json
  - internal/zcode/fake/testdata/scenarios/completion-quiet-window.json
  - internal/zcode/fake/testdata/scenarios/credential-cipher-scripted.json
  - internal/zcode/fake/testdata/scenarios/envelope-negatives.json
  - internal/zcode/fake/testdata/scenarios/eof-drain.json
  - internal/zcode/fake/testdata/scenarios/error-32010.json
  - internal/zcode/fake/testdata/scenarios/full-cycle.json
  - internal/zcode/fake/testdata/scenarios/happy-turn.json
  - internal/zcode/fake/testdata/scenarios/newsession-orphan-close.json
  - internal/zcode/fake/testdata/scenarios/no-reasoning.json
  - internal/zcode/fake/testdata/scenarios/part-delta-streaming.json
  - internal/zcode/fake/testdata/scenarios/permission-allow.json
  - internal/zcode/fake/testdata/scenarios/permission-cancel.json
  - internal/zcode/fake/testdata/scenarios/reannounce-dedupe.json
  - internal/zcode/fake/testdata/scenarios/runtime-headers-answer.json
  - internal/zcode/fake/testdata/scenarios/runtime-preferences.json
  - internal/zcode/fake/testdata/scenarios/send-refused-32010.json
  - internal/zcode/fake/testdata/scenarios/session-lifecycle.json
  - internal/zcode/fake/testdata/scenarios/setmodel-followup.json
  - internal/zcode/fake/testdata/scenarios/tool-lifecycle.json
  - internal/zcode/fake/testdata/scenarios/unknown-host-method.json
  - internal/zcode/fake/testdata/scenarios/user-input-decline.json
  - internal/zcode/roundtrip_test.go
  - internal/zcode/writer.go
  - internal/zcode/writer_test.go
covered_digest: "v4:sha256:7d8a0a14d5ade8c075ef1de3b2b19c1c1b5c2f01bf6bb5a103d6a226bc17594f"
behavior_unverified: 0 # Count of PRESENT_BEHAVIOR_UNVERIFIED truths (present + wired, behavior not exercised); each is detailed in behavior_unverified_items below (and in human_verification when status is human_needed)
overrides_applied: 0 # Count of PASSED (override) items included in score
re_verification: # Stale-verification refresh: prior report passed 43/43 on 2026-11-20; Phase 5 commits (4c423c0, 78ab036, 8c78a0c) changed 4 covered files afterwards — all must-haves re-checked on the current tree, behavior-dependent truths re-proven by passing tests
  previous_status: passed
  previous_score: 43/43
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor — Verification Report

**Phase Goal:** As a later-phase implementer on this repo, I want to the wrapper's back leg — a JSONL codec for the app-server dialect, a scriptable fake app-server with golden transcripts in CI, a desktop discovery + process supervisor, and a turn state machine, so that every later phase is testable without the real desktop bundle.
**Verified:** 2026-11-21T06:03:18Z
**Status:** passed
**Re-verification:** Yes — stale-verification refresh: the 2026-11-20 report passed 43/43, but 4 covered files changed afterwards (Phase 5 commits). Goal-backward verification re-run over the current tree; fingerprint recomputed via the CLI.

**Mode:** mvp (user-story goal; goal text unchanged since the prior validation via `user-story.validate`)

## What changed since the verified tree (c12dd6a..HEAD)

Exactly 4 covered files changed, all via Phase 5 commits (4c423c0, 78ab036, 8c78a0c). Diffs read in full this run:

- `internal/zcode/envelope.go` (+26/-10): **purely additive** — 4 new method constants (`session/resume`, `session/close`, `provider/updateAccountConfig`, `provider/refreshRegistry`) plus gofmt realignment of the const block. The framing-key structural defense (Envelope has no framing field, :86) and the error-code table are untouched.
- `internal/zcode/conn_test.go` (+84 net): **test-internal refactor** — extracted `snapshotRouterOrder`/`fireCipherExpiry` helpers in `TestRouterFanout`, wire ids now fresh nonces instead of fixed strings. Every assertion preserved, including the D-05 never-reply check (now inside the helper, asserted per window) and the registration-order/[A B]→[B] delivery contract. Production code untouched.
- `internal/zcode/fake/golden_test.go` (+451): **additive** — 6 new Phase-5 corpus drivers (tool-lifecycle, session-lifecycle, newsession-orphan-close, auth-apikey, runtime-headers-answer, full-cycle) + named wire-key constants. The 19 existing rows and the compare path unchanged.
- `experiments/scan-secrets.sh` (+19/-8): the secret-scan gate's allowlist gained one exact string (`zc-synthetic-fixture-not-a-real-key`, the 05-06 self-declaring synthetic fixture marker) alongside the existing phase-3 marker; patterns untouched, all other matches still fail the gate.

Corpus grew 19 → 25 scenario+golden pairs (Phase-5 rows); all 19 phase-3-era pairs are still on disk and green — superset holds. No phase-3 invariant was weakened anywhere.

## User Flow Coverage

User story: «As a later-phase implementer on this repo, I want to the wrapper's back leg — a JSONL codec for the app-server dialect, a scriptable fake app-server with golden transcripts in CI, a desktop discovery + process supervisor, and a turn state machine, so that every later phase is testable without the real desktop bundle.»

The "user" is a later-phase implementer; the outcome clause is «every later phase is testable without the real desktop bundle». Each step below was re-executed first-hand on the current tree (2026-11-21).

| Step | Expected | Evidence | Status |
|------|----------|----------|--------|
| Run the codec against dialect frames | Frames without a `jsonrpc` key, dialect error codes, ordered single-writer output | `internal/zcode/envelope.go:86` (framing key unrepresentable by construction — re-read on current tree), `TestEncodeOmitsJSONRPC` PASS first-hand (asserts no "jsonrpc" byte in the outbound line); error-code table intact; `TestWriterOrdering` + `TestStopJumpsQueue` PASS first-hand | ✓ |
| Replay golden transcripts with no desktop present | Every corpus scenario replays byte-deterministically over pipes | `TestGolden` **25/25 subtests PASS, 0 FAIL/SKIP** first-hand (the 19 phase-3 rows plus 6 later-phase rows — a superset of the phase-3 corpus contract); 25 scenarios + 25 goldens on disk; `TestFakeBinarySpawnPath` PASS (real child process) | ✓ |
| Use the standalone fake binary tier | Binary serves `--scenario` over stdio; frames proven over a real child | `cmd/fake-app-server/main.go` unchanged (`--scenario` required, optional `--transcript`), `TestFakeBinarySpawnPath` + `TestEOFSignatureChild` PASS first-hand | ✓ |
| Point discovery at an install — or at nothing | First-match-wins per the Phase-1 matrix; empty result names every probe; /tmp/.mount_* rejected | discovery code unchanged since verified tree; `TestDiscoverOrderFirstMatchWins`, `TestDiscoverNoInstallListsEveryProbe`, `TestDiscoverRejectsVolatileMounts` PASS first-hand; revision formula `discover.go:147-162` re-read (EvalSymlinks → sha256 over the resolved absolute PATH STRING) | ✓ |
| Supervise a child; kill the whole group on exit | Zero surviving group members (grandchildren included); allowlisted+scrubbed env on the child's observed env | `TestGroupKillNoSurvivors`, `TestSpawnEnv`, `TestRestartWindow` PASS first-hand; `killGroup` wiring re-read (supervise.go:184 → spawn.go:307 `syscall.Kill(-pgid, SIGKILL)`, ESRCH tolerated) | ✓ |
| Drive a synchronous turn; cancel mid-turn | Fire-and-forget `session/send` resolves exactly once; cancel resolves cancelled exactly once | turn + supervisor + fake packages all green under `-race` first-hand; named runs: `TestHappyTurnCompletedExactlyOnce`/`TestCancelMidTurnAnswersOnce`/`TestDoubleSendRejected*`/`TestCompletionVariants`/`TestCredentialCipherExpiryRecorded` 6/6 PASS first-hand | ✓ |
| Compose the whole chain | Supervisor + fake binary + Conn + Machine in one test | `TestEndToEndExecChain` PASS first-hand (both subtests, 0.30s) | ✓ |
| Outcome: every later phase testable without the desktop bundle | Full gate green on a machine that HAS the desktop — proving nothing in the suite needs it | Build/vet EXIT=0, lint 0 issues, test-experiments EXIT=0, tidy-diff EXIT=0, secret-scan EXIT=0 — all first-hand this run; all four phase-3 packages green under `-race` first-hand; the real-desktop positive path re-executed first-hand today (PASS 2.18s, zero orphans) — see Behavioral Spot-Checks | ✓ |

## Goal Achievement

### Observable Truths

Roadmap truths (RT) are the contract; plan truths (Pxx-y) are the plan-frontmatter detail merged per Step 2c. **All evidence below is first-hand re-execution on the current tree (2026-11-21)** — not inherited from the prior report. The 4 changed files (envelope.go, conn_test.go, golden_test.go, scan-secrets.sh) received full 3-level + behavioral verification; unchanged files received existence + sanity regression plus the package-level behavioral runs.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| RT1 | The codec round-trips the app-server JSONL dialect: frames without a `jsonrpc` key, the dialect's own error codes, ordered single-writer output — a JSON-RPC-shaped frame never reaches the child | ✓ VERIFIED | envelope.go diff read in full: additive constants only; framing-key structural defense intact (:86 re-read); `TestEncodeOmitsJSONRPC` + `TestWriterOrdering` + `TestStopJumpsQueue` PASS first-hand; corpus `error-32010` + `envelope-negatives` green (inside TestGolden 25/25); `go test -race ./internal/zcode/` ok 3.131s; lint 0 issues |
| RT2 | CI runs golden JSONL transcript tests against the scriptable fake app-server (in-process over pipes plus a standalone binary), no desktop bundle present | ✓ VERIFIED | `TestGolden` 25/25 PASS + `TestFakeBinarySpawnPath` PASS first-hand; `.github/workflows/pr-sanity.yml` runs mise build/vet/lint/test/test-experiments/tidy-diff (re-read, :72-91) — the `test` task replays the corpus; realdesktop tier build-tag+env guarded (realdesktop_test.go header re-read) |
| RT3 | The supervisor locates the install per the Phase 1 matrix, spawns the child with the correct environment, kills the whole process group on EOF/exit leaving no orphans; no-install fails with diagnostics naming what was searched | ✓ VERIFIED | discovery/spawn/supervise code unchanged since verified tree (git diff empty); `TestDiscoverNoInstallListsEveryProbe`, `TestDiscoverRejectsVolatileMounts`, `TestDiscoverOrderFirstMatchWins`, `TestSpawnEnv`, `TestGroupKillNoSurvivors`, `TestRestartWindow` PASS first-hand; real-desktop positive path re-executed first-hand today: `TestRealDesktop` PASS 2.18s — layout=unpacked-appimage, revision zcode-builtin:44:e9c0674a…, session/create ok, kill(-pgid,0) ESRCH zero orphans; `ps` after: zero zcode.cjs processes |
| RT4 | Against the fake app-server, a fire-and-forget `session/send` is lifted into a synchronous turn that completes exactly once on multi-signal turn-completion detection and honors cancel mid-turn | ✓ VERIFIED | `go test -race ./internal/turn/` ok 3.200s; named runs `TestHappyTurnCompletedExactlyOnce`, `TestCancelMidTurnAnswersOnce`, `TestDoubleSendRejected*`, `TestCompletionVariants`, `TestCredentialCipherExpiryRecorded` 6/6 PASS first-hand; machine.go/watcher.go unchanged since verified tree |
| P01-1 | Conn vs in-process fake over io.Pipe completes happy-turn end-to-end (session/create with reverse request answered, session/send acked accepted:true, notifications in wire order) | ✓ VERIFIED | codec + fake packages green under -race first-hand; happy-turn row green inside TestGolden 25/25 |
| P01-2 | Every byte to the child is a complete newline-delimited JSON envelope; no outbound frame can carry a JSON-RPC framing key (structural) | ✓ VERIFIED | `TestEncodeOmitsJSONRPC` PASS first-hand; envelope.go:86 structural comment + `bytes.Contains(line, "jsonrpc")` negative assertion re-read on current tree; the additive constants touch neither the struct nor Encode |
| P01-3 | Reverse-request replies travel a lane that never queues behind ordinary calls (two-lane writer) | ✓ VERIFIED | writer.go/call.go unchanged since verified tree (git diff empty); `TestStopJumpsQueue` PASS first-hand |
| P01-4 | Golden replay byte-deterministic (fixed ids); any frame drift or missing frame fails — never normalized away | ✓ VERIFIED | `TestGolden` 25/25 PASS first-hand incl. happy-turn; golden.go compare path unchanged |
| P02-1 | WR-02: a numeric-id reverse request colliding with an in-flight call id routes to the router, NEVER resolves the pending call | ✓ VERIFIED | `TestDemuxReverseRequestNeverResolvesPendingCall` PASS first-hand (named run) |
| P02-2 | Ids correlate by exact value AND type: string 5 ≠ integer 5 on the wire | ✓ VERIFIED | `TestDemuxIdTypeNonConflation` PASS first-hand (named run) |
| P02-3 | Single-writer ordering: concurrent writers produce complete, never-interleaved frames | ✓ VERIFIED | `TestWriterOrdering` PASS first-hand; writer.go unchanged |
| P02-4 | Bypass precedence: session/stop issued while the ordered queue holds pending writes is on the wire after at most the in-flight write | ✓ VERIFIED | `TestStopJumpsQueue` PASS first-hand; bypass lane intact |
| P02-5 | Raised-line behavior: 128 KiB line survives intact; malformed JSON line skipped with warning, stream continues | ✓ VERIFIED | `TestLineBufferLargeLineIntact` + `TestLineBufferMalformedLineBetweenValidLines` PASS first-hand; `conn.go:264` `sc.Buffer(make([]byte,0,readBufInitial), readBufMax)` re-read (checker literal "Scanner.Buffer" is a var-naming false positive — var is `sc`) |
| P02-6 | D-05 deadline honesty: runtime preferences answered within deadline; mid-turn credential-cipher known-but-NOT-answered; expiry recorded observably | ✓ VERIFIED | `TestRouterRuntimePreferencesDeadline` + `TestRouterCredentialCipherNeverAnsweredExpiryRecorded` + `TestRouterUnknownMethodAnswered32601` PASS first-hand (named runs); conn.go router policy untouched by the test refactor |
| P03-1 | The golden corpus lives under testdata/scenarios/ and every scenario replays green against the codec Conn under -race | ✓ VERIFIED | `TestGolden` **25/25** PASS first-hand (corpus grew 19 → 25 by Phase-5 rows; all phase-3 rows present and green — superset holds); 25+25 files on disk (listed) |
| P03-2 | Every golden replays byte-deterministically; `-update` regenerates; drift fails | ✓ VERIFIED | `TestGolden` green first-hand on the 25-row corpus; golden.go unchanged since verified tree |
| P03-3 | The scenario loader fails loud: invalid file rejected with an error naming the offending field | ✓ VERIFIED | fake package green under -race first-hand (loader rejection tests included); loader code unchanged |
| P03-4 | Exit-0-with-zero-lines is a FAILURE in-process, never an empty success | ✓ VERIFIED | `TestEOFSignature` PASS first-hand; exec-tier twin `TestEOFSignatureChild` PASS first-hand |
| P04-1 | Discovery implements the Phase 1 recommended order exactly; first-match-wins; probes do not merge | ✓ VERIFIED | `TestDiscoverOrderFirstMatchWins` PASS first-hand; discover.go unchanged |
| P04-2 | Empty result names EVERY probe attempted with its result — the diagnostics are the product | ✓ VERIFIED | `TestDiscoverNoInstallListsEveryProbe` PASS first-hand (8 probes) |
| P04-3 | Volatile /tmp/.mount_* decoys recorded-and-rejected; localhost listeners never probed | ✓ VERIFIED | `TestDiscoverRejectsVolatileMounts` PASS first-hand |
| P04-4 | Builtin revision computed, never hardcoded: sha256 over the resolved absolute PATH STRING | ✓ VERIFIED | `discover.go:147-162` re-read; real-desktop run printed `zcode-builtin:44:e9c0674a…` live first-hand today |
| P05-1 | Machine.Run lifts one session/send into a synchronous turn: returns StopReason exactly once, after a completion signal — never before, never twice | ✓ VERIFIED | turn package green under -race; `TestHappyTurnCompletedExactlyOnce` PASS first-hand; E2E subtest `happy_turn_completes_exactly_once` PASS; machine.go unchanged |
| P05-2 | Completion classification multi-signal and ordered (terminal → idle patch → quiet-window); each variant resolves exactly once | ✓ VERIFIED | `TestCompletionVariants` PASS first-hand |
| P05-3 | Terminal classifier matches turnId + kind across known carriers; carrier-tolerant by design | ✓ VERIFIED | turn package green under -race (carrier-tolerance tests included); classifier unchanged |
| P05-4 | TurnWatcher is an interface with a polling implementation; quiet-window gates with send-latency floor 3x quiet-ms after the send ack; subscribe watcher is a reserved slot, NOT implemented | ✓ VERIFIED | `watcher.go:27` `DefaultFloorMultiple = 3` re-read (set at :120 when not injected — a precondition the code establishes); grep confirms no subscribe implementation exists (doc mentions only) |
| P05-5 | D-04: the machine is bound to ONE app-server sessionId on the ONE shared Conn (1:1 ACP session mapping) | ✓ VERIFIED | `machine.go:147-149` single-session binding re-read, unchanged; E2E asserts the create reply's sessionId against the binding |
| P06-1 | Standalone fake binary serves a JSON scenario over its own stdio: --scenario required, frames on stdin/stdout, diagnostics stderr-only | ✓ VERIFIED | `cmd/fake-app-server/main.go` re-read (usage error → `fake.ExitUsage`, stderr slog; optional `--transcript` delegates to `fake.MainTo`); `TestFakeBinarySpawnPath` PASS first-hand |
| P06-2 | EOF contract: on stdin EOF the fake drains ~100 ms (eofDrainMs) then exits 0; a request just before EOF is never answered; drain observable in tests | ✓ VERIFIED | eof-drain corpus row green inside TestGolden 25/25; `TestEOFSignatureChild` PASS first-hand |
| P06-3 | Exec-plumbing proven against a REAL child: helper reexec drives frames both directions with the child in its own process group | ✓ VERIFIED | `TestFakeBinarySpawnPath` PASS first-hand (GO_WANT_HELPER_PROCESS reexec) |
| P06-4 | Exit-0-with-zero-lines failure signature enforced at the exec tier | ✓ VERIFIED | `TestEOFSignatureChild` PASS first-hand |
| P07-1 | Child spawns with the canonical contract: argv [binary, entrypoint, app-server, --stdio]; env ELECTRON_RUN_AS_NODE=1 + ZCODE_BUILTIN_PROVIDER_CONFIG_FILE | ✓ VERIFIED | `TestSpawnEnv` + spawn tests PASS first-hand (supervisor package ok 2.573s); spawn.go unchanged; `SpawnArgv` returns the populated argv literal (spawn.go:120 re-read) |
| P07-2 | Child env is an explicit allowlist (PATH+HOME) + two ZCode vars, proxy vars scrubbed — asserted on the child's OBSERVED environment | ✓ VERIFIED | `TestSpawnEnv` asserts via the reexec child's own observed env (`spawn.go` allowlist constants); PASS first-hand |
| P07-3 | Whole process group dies on EOF/exit: staged shutdown (stdin end → wait/grace → Kill(-pgid, SIGKILL) → reap) leaves zero survivors incl. grandchildren, ESRCH-verified | ✓ VERIFIED | `TestGroupKillNoSurvivors` PASS first-hand; staged shutdown re-read: `supervise.go:184` `killGroup(child.PgID)` → `spawn.go:307` `syscall.Kill(-pgid, syscall.SIGKILL)` (ESRCH tolerated); real-desktop run re-verified ESRCH first-hand today |
| P07-4 | Crash-window restart: 3 attempts within sliding 60 s then fail loud; stable run resets the counter; injectable for tests | ✓ VERIFIED | `TestRestartWindow` PASS first-hand — includes the crash-loop-trip-leaves-zero-survivors case (review fix 6794626/5bbb38f) |
| P07-5 | Health watch = stdin liveness + child exit; NO protocol liveness frame is invented | ✓ VERIFIED | supervise.go contract comments re-read (unchanged); grep confirms no ping/liveness frame code |
| P08-1 | Cancel mid-turn: session/stop through the bypass lane immediately; turn resolves cancelled EXACTLY once; second Cancel no-op; cancel-after-complete no-op | ✓ VERIFIED | `TestCancelMidTurnAnswersOnce` PASS first-hand; `Conn.Stop` bypass enqueue unchanged |
| P08-2 | A second concurrent session/send is rejected wrapper-side with a clean error mapping the dialect code -32010 (numeric code, never message text) | ✓ VERIFIED | `TestDoubleSendRejected*` PASS first-hand; `machine.go:130-134` maps `zcode.CodeConcurrency` numerically (re-read, unchanged) |
| P08-3 | Corpus-level completion: completion-idle-patch and completion-quiet-window replay against the codec Conn, each resolving exactly once | ✓ VERIFIED | turn package green under -race; corpus rows green inside TestGolden 25/25 |
| P08-4 | D-05 honesty at the turn tier: scripted credential-cipher exchange ends with expiry RECORDED, turn NOT falsely resolved — degradation observable, never swallowed | ✓ VERIFIED | `TestCredentialCipherExpiryRecorded` PASS first-hand |
| P09-1 | The FULL chain runs in one test: supervisor spawns the standalone fake binary through the real spawn path (process group, allowlisted env, drained stderr), Conn rides the exec pipes, Machine.Run completes the happy turn exactly once | ✓ VERIFIED | `TestEndToEndExecChain` PASS first-hand (0.30s); `TestMain` builds the binary via `go build` into `t.TempDir()` |
| P09-2 | Cancel works over the exec chain: mid-turn cancel resolves cancelled exactly once; staged shutdown leaves zero surviving group members (ESRCH) | ✓ VERIFIED | E2E subtest `cancel_mid-turn_resolves_cancelled_exactly_once` PASS first-hand (second supervised cycle serving cancel-mid-turn.json — documented deviation, keeps every must-have) |
| P09-3 | The whole phase gate is green: mise run ci with the corpus replayed inside; zero dependencies added by this phase's own work | ✓ VERIFIED | Gate legs re-run first-hand today: build EXIT=0, vet EXIT=0, lint 0 issues, test-experiments EXIT=0, tidy-diff EXIT=0 (1 require line pinned to acp-go-sdk v0.20.0 per the Phase-4-updated gate), secret-scan EXIT=0; all four phase-3 packages green under -race first-hand. go.mod's acp-go-sdk pin remains Phase-4 work asserted by the updated tidy-diff gate — intentional, outside phase-3 scope; phase 3's own diff added zero dependencies |

**Score:** 43/43 truths verified (0 present, behavior-unverified)

### Advisory (New Scope, Unevidenced)

Re-verification ran, so this section is included. None — the anti-pattern scan produced zero findings across all phase-3 files including the 4 changed files (debt markers, placeholders, empty implementations: 0). The one cosmetic observation (a dangling doc comment at `conn_test.go:1305`) is recorded as Info under Anti-Patterns, not advisory material — it is deterministic, compiles clean, and gates are green.

| # | Finding | Category | Why Advisory |
|---|---------|----------|--------------|
| — | None | — | — |

### Required Artifacts

gsd-tools `verify.artifacts` re-run on all 9 plans: 6/9 all-pass; the 4 misses are the previously documented literal-pattern checker false positives, each re-confirmed present on the current tree by direct read.

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/zcode/envelope.go` | ID, Envelope (no framing-key field), WireError, method + error-code constants | ✓ VERIFIED | Additive constants since verified tree (diff read); framing-key structural defense intact (:86); error-code table untouched |
| `internal/zcode/conn.go` | Conn: read loop with raised scanner buffer, three-way demux, reverse-request router with D-05 policy | ✓ VERIFIED | `sc.Buffer` raised-line at :264 re-read (checker "Missing pattern: Scanner.Buffer" is a var-naming false positive); D-05 policy intact |
| `internal/zcode/writer.go` | Single writer goroutine, two lanes (ordered + bypass) | ✓ VERIFIED | Unchanged since verified tree; bypass lane verified by TestStopJumpsQueue |
| `internal/zcode/call.go` | Call/Notify/Reply/ReplyError, pending table, Stop helper | ✓ VERIFIED | Unchanged since verified tree |
| `internal/zcode/{roundtrip,conn,writer,call}_test.go` | Codec unit tests per plan | ✓ VERIFIED | conn_test.go refactored internally by Phase 5 (helpers extracted, assertions preserved — diff read in full); all green under -race |
| `internal/zcode/fake/fake.go` | Serve(rw, script): scripted matching, delays, events, blocking reverse requests | ✓ VERIFIED | Unchanged; fake package ok 8.252s under -race |
| `internal/zcode/fake/scenario.go` + `scenario_test.go` | JSON scenario schema + fail-loud load; loader validation tests | ✓ VERIFIED | Unchanged; loader rejection tests green inside package run |
| `internal/zcode/fake/golden.go` + `golden_test.go` | Transcript record/compare with -update; corpus replay + EOF signature | ✓ VERIFIED | golden.go unchanged; golden_test.go grew +451 lines (6 Phase-5 corpus drivers); TestGolden 25/25 green |
| `internal/zcode/fake/testdata/scenarios/` | Golden corpus scenario files | ✓ VERIFIED | 25 JSON files on disk (checker "Not a file: directory" is a file-only checker — contents verified by listing + replay) |
| `internal/zcode/fake/testdata/golden/` | Golden transcripts | ✓ VERIFIED | 25 .golden files on disk; all replay green |
| `internal/zcode/fake/binary_test.go` | Reexec integration tests (spawn path, EOF drain, zero-lines signature) | ✓ VERIFIED | GO_WANT_HELPER_PROCESS pattern; both named tests PASS first-hand |
| `internal/zcode/fake/stdio.go` | stdio hosting of the scripting core (fake.Main/ExitUsage) | ✓ VERIFIED | Unchanged; used by cmd main |
| `cmd/fake-app-server/main.go` | Thin main: --scenario flag, stdio serving, stderr diagnostics, EOF drain exit | ✓ VERIFIED | Unchanged (~29 lines); --scenario required |
| `internal/supervisor/discover.go` + `discover_test.go` | Ordered first-match discovery, full diagnostics, decoy rejection, BuiltinRevision | ✓ VERIFIED | Unchanged since verified tree (incl. the additive ExplicitRootInstall helper); discovery tests green first-hand |
| `internal/supervisor/spawn.go` | SpawnArgv, ChildEnv, SpawnChild with Setpgid + 3 stdio pipes + capped stderr drain + group-kill Cancel | ✓ VERIFIED | Unchanged; Setpgid + killGroup :307-308 re-read |
| `internal/supervisor/supervise.go` | Supervisor: health watch, staged shutdown, crash-window restart | ✓ VERIFIED | Unchanged; staged shutdown killGroup at :184 re-read (checker "Missing pattern: Kill" is a cross-file false positive — the literal lives in spawn.go, verified wired) |
| `internal/supervisor/{spawn,supervise,realdesktop,e2e}_test.go` | Env/argv/group-kill/restart/realdesktop/E2E tests | ✓ VERIFIED | All green first-hand; realdesktop behind `//go:build realdesktop` + env guard, executed green first-hand today |
| `internal/turn/machine.go` + `watcher.go` + `machine_test.go` + `cancel_test.go` | Machine (Run/Cancel, -32010 guard), TurnWatcher + PollWatcher, tests | ✓ VERIFIED | Unchanged; turn package ok 3.200s under -race; corpus path `corpusScenarioPath` re-read at cancel_test.go:570-572 |
| `internal/supervisor/e2e_test.go` | End-to-end exec-chain test | ✓ VERIFIED | TestEndToEndExecChain PASS first-hand (both subtests) |
| evidence/ (13 RED-proof JSONs) | TDD RED evidence for 03-01/02/03/04/05/07/08 + review CR-01/WR-01 | ✓ VERIFIED | All 13 present and unchanged (in the covered set) |

### Key Link Verification

gsd-tools `verify.key-links` re-run on all 9 plans: 8/9 all_verified; the single miss is the known corpus-path false positive, re-confirmed wired by direct read.

| From | To | Via | Status | Details |
| ---- | -- | --- | ------ | ------- |
| internal/zcode/fake/roundtrip_test.go | internal/zcode/fake/fake.go | io.Pipe pairing | ✓ WIRED | tool-verified |
| internal/zcode/conn.go | internal/zcode/call.go | demux resolves pending by canonical id | ✓ WIRED | tool-verified |
| internal/zcode/call.go | internal/zcode/writer.go | Call/Notify ordered lane; Reply/Stop bypass lane | ✓ WIRED | tool-verified |
| internal/zcode/conn_test.go | internal/zcode/conn.go | demux + router policy via public Conn surface | ✓ WIRED | tool-verified (refactored tests still assert through the public surface; all green) |
| internal/zcode/fake/golden_test.go | testdata/scenarios/ | table-driven LoadScenario → Serve → CompareGolden | ✓ WIRED | tool-verified (now 25 rows) |
| testdata/scenarios/completion-quiet-window.json | fake/fake.go | session/messages poll answers | ✓ WIRED | tool-verified |
| internal/supervisor/discover_test.go | discover.go | Probers seams (t.TempDir fixtures) | ✓ WIRED | tool-verified |
| discover.go | docs/protocol/install-discovery.md | probe order/set/fields mirror matrix | ✓ WIRED | tool-verified |
| internal/turn/machine.go | zcode/conn.go / call.go | session/send via Call; Stop bypass; -32010 numeric mapping | ✓ WIRED | tool-verified |
| internal/turn/cancel_test.go | testdata/scenarios/cancel-mid-turn.json | corpus scenario replay via corpusScenarioPath | ✓ WIRED | tool pattern missed the `filepath.Join("..","zcode","fake","testdata","scenarios",…)` construction at cancel_test.go:571 — re-confirmed wired by direct read; corpus rows green in TestGolden |
| supervisor/spawn.go | discover.go | SpawnArgv + ChildEnv consume Install | ✓ WIRED | tool-verified |
| supervisor/supervise.go | spawn.go | staged shutdown → killGroup | ✓ WIRED | tool-verified + direct read (supervise.go:184 → spawn.go:307) |
| supervisor/spawn_test.go | cmd/fake-app-server/main.go | fake binary as the CI child | ✓ WIRED | tool-verified |
| supervisor/e2e_test.go | supervise.go / machine.go / cmd main | Supervisor lifecycle + Machine over exec pipes + go build child | ✓ WIRED | tool-verified |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `TestEndToEndExecChain` | StopReason / sessionId | Real exec pipes to the built fake-app-server child (go build → t.TempDir) driven through SpawnChild + Conn + Machine | Yes — real process I/O, real frames on os-level pipes; re-run first-hand today | ✓ FLOWING |
| `TestSpawnEnv` | Child's observed environment | Reexec helper child reads its own os.Environ and reports it back | Yes — asserted on the OBSERVED child env, not the builder's intention | ✓ FLOWING |
| `TestDiscover*` | Install candidates | Injected Probers seams over t.TempDir fixture trees | Yes — the seams are the documented injection point; production path consumes real probes (proven by today's real-desktop run resolving the live install) | ✓ FLOWING |
| `TestRealDesktop` | Install root / builtinRevision | The actual ZCode 3.18.2 install on this machine (layout=unpacked-appimage) | Yes — re-executed first-hand today, revision zcode-builtin:44:e9c0674a… printed | ✓ FLOWING |
| Golden corpus replay | Transcript frames | fake.Serve over io.Pipe from hand-authored scenario JSONs | Yes — the fake IS the intended data source for later phases (D-01); ids are synthetic by contract (sess_fix1/turn_fix1); the corpus now carries Phase 4-5 rows too | ✓ FLOWING |

No static returns, hardcoded literals, or mocks on any asserted path. The fake app-server is the product (a scriptable stand-in), not a mock hiding a missing implementation — the E2E chain proves the composition over real exec pipes.

### Behavioral Spot-Checks

All commands below re-run first-hand on the current tree (2026-11-21). Package-scoped runs cover each phase-3 package exactly once; named runs are single named tests; no full-workspace-suite filter runs.

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Build + vet | `go build ./... && go vet ./...` | both EXIT=0 | ✓ PASS |
| Codec package (all RT1/P01/P02 invariants) | `go test -race -count=1 ./internal/zcode/` | ok 3.131s | ✓ PASS |
| Fake package (TestGolden + loader + EOF + binary tier) | `go test -race -count=1 ./internal/zcode/fake/` | ok 8.252s | ✓ PASS |
| Golden corpus enumeration | `go test -race -count=1 -run 'TestGolden$' -v ./internal/zcode/fake/` | 25/25 subtests PASS, 0 FAIL/SKIP | ✓ PASS |
| Turn package (exactly-once, cancel, -32010, honesty) | `go test -race -count=1 ./internal/turn/` | ok 3.200s | ✓ PASS |
| Supervisor package (discovery, spawn, group kill, restart, E2E) | `go test -race -count=1 ./internal/supervisor/` | ok 2.573s | ✓ PASS |
| Named codec/router invariants (11 tests incl. refactored TestRouterFanout) | `go test -race -run 'TestEncodeOmitsJSONRPC\|TestDemux…\|TestRouter…\|TestWriterOrdering\|TestStopJumpsQueue\|TestLineBuffer…' -v ./internal/zcode/` | 11/11 PASS | ✓ PASS |
| Named turn invariants | `go test -race -run 'TestHappyTurn…\|TestCancelMidTurn…\|TestDoubleSend…\|TestCompletionVariants\|TestCredentialCipher…' -v ./internal/turn/` | 6/6 PASS | ✓ PASS |
| Named supervisor invariants | `go test -race -run 'TestDiscover…\|TestGroupKill…\|TestRestartWindow\|TestSpawnEnv' -v ./internal/supervisor/` | 7/7 PASS | ✓ PASS |
| E2E exec chain | `go test -race -count=1 -run TestEndToEndExecChain -v ./internal/supervisor/` | PASS both subtests (0.30s) | ✓ PASS |
| Fake binary tier | `go test -race -run 'TestFakeBinarySpawnPath\|TestEOFSignatureChild' ./internal/zcode/fake/` + `TestEOFSignature` | all PASS | ✓ PASS |
| Real-desktop positive path | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop -v ./internal/supervisor/` | PASS 2.18s; layout=unpacked-appimage; revision zcode-builtin:44:e9c0674a8011…; session/create ok; kill(-pgid,0) ESRCH; `ps` shows zero zcode.cjs processes afterwards | ✓ PASS |
| Lint gate | `mise run lint` | 0 issues | ✓ PASS |
| Experiments gate | `mise run test-experiments` | EXIT=0, 0 cancelled/skipped/todo | ✓ PASS |
| Dependency gate | `mise run tidy-diff` | EXIT=0 (1 require line pinned to acp-go-sdk v0.20.0 per the Phase-4-updated gate) | ✓ PASS |
| Secret scan (changed gate script) | `bash experiments/scan-secrets.sh` | EXIT=0 with the extended 2-marker allowlist | ✓ PASS |
| Test enumeration (no disabled tests) | `go test -list '.*'` × 4 packages | 68 tests enumerated (unchanged count: zcode 20, fake 23, supervisor 13, turn 12); 0 skipped/disabled in the default suite (only the 2 documented env-guard `t.Skip` in realdesktop_test.go under the tag without env) | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| (none declared) | `find scripts -path '*/tests/probe-*.sh'` → none; PLAN/SUMMARY grep for probe-*.sh → none | No conventional or declared probe scripts exist in this phase; the executable checks are the Go suite + mise tasks, all run first-hand above | N/A |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
| ----------- | ------------ | ----------- | ------ | -------- |
| BRIDGE-01 | 03-01, 03-02, 03-09 | JSONL codec for the app-server dialect (no `jsonrpc` key, its error codes, ordered single-writer output) with unit tests | ✓ SATISFIED | `internal/zcode/` green under -race first-hand; 11 named invariant tests PASS incl. the refactored router tests; envelope structural defense intact through the additive Phase-5 constants (diff read) |
| BRIDGE-02 | 03-01, 03-03, 03-06, 03-09 | Fake app-server (in-process over pipes + standalone binary) and golden JSONL transcript tests running in CI | ✓ SATISFIED | `internal/zcode/fake/` + `cmd/fake-app-server/`, 25+25 corpus, TestGolden 25/25 + binary tier PASS first-hand, CI runs the same mise tasks (pr-sanity.yml re-read) |
| BRIDGE-03 | 03-04, 03-07, 03-09 | Desktop discovery + process supervisor: locate, spawn ELECTRON_RUN_AS_NODE=1, kill whole group on EOF/exit, fail loudly when not found | ✓ SATISFIED | `internal/supervisor/` 13 tests green first-hand + real-desktop positive path PASS first-hand (2.18s, ESRCH zero orphans) |
| BRIDGE-04 | 03-05, 03-08, 03-09 | Turn state machine lifting fire-and-forget session/send into a synchronous prompt lifecycle, multi-signal completion, cancel | ✓ SATISFIED | `internal/turn/` green under -race first-hand incl. exactly-once + cancel + -32010 guard (both sides) |

Orphaned requirements: none — REQUIREMENTS.md maps exactly BRIDGE-01..04 to Phase 3 (traceability lines 120-123), and all four appear in the plans' `requirements` frontmatter. All four marked Complete, consistent with this verification.

### Test Quality Audit

| Test File | Linked Req | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|-----------|--------|---------|----------|-----------------|---------|
| internal/zcode/*_test.go (4 files) | BRIDGE-01 | 20 | 0 | No | Value + behavioral (verbatim id echo, type non-conflation, ordering, bypass, fan-out ordering — assertions preserved through the Phase-5 refactor, diff read) | STRONG |
| internal/zcode/fake/*_test.go (5 files) | BRIDGE-02 | 23 | 0 | No — goldens are hand-authored spec transcriptions (fixed synthetic ids), not captures of the SUT | Behavioral (byte-deterministic replay, EOF failure signature, fail-loud loader) | STRONG |
| internal/supervisor/*_test.go (5 files) | BRIDGE-03 | 13 | 0 default (realdesktop: 2 env-guarded skips, by design) | No | Behavioral (ESRCH no-survivors, observed child env, crash-loop trip, live-install resolve) | STRONG |
| internal/turn/*_test.go (3 files) | BRIDGE-04 | 12 | 0 | No | Behavioral (exactly-once counters, -32010 numeric mapping both sides, expiry recorded) | STRONG |

**Disabled tests on requirements:** 0 (the 2 realdesktop `t.Skip` are the documented env-guard, not requirement-linked disabled tests; the guarded run was executed green first-hand today)
**Circular patterns detected:** 0
**Insufficient assertions:** 0

### Decision Coverage

All trackable CONTEXT.md decisions are honored by shipped artifacts. (`check.decision-coverage-verify`: skipped=false, total=6, honored=6, not_honored=[] — non-blocking gate, unchanged since the prior pass; the D-04/D-05/WR-02 code paths were re-read this run.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| internal/zcode/conn_test.go | 1305 | Dangling orphan doc comment `// snapshotRouterOrder returns a copy of the delivery-order log.` left at EOF by the Phase-5 helper-extraction refactor (the function it documents is defined at :1272 with its own doc) | ℹ️ Info | Cosmetic only — gofmt clean, vet clean, lint 0 issues, all tests green; no debt marker, no behavioral effect |
| (none) | — | TBD/FIXME/XXX debt markers: 0 across all phase-3 files incl. the 4 changed files and 25 corpus pairs | — | — |
| (none) | — | TODO/HACK/PLACEHOLDER: 0; not-implemented/coming-soon phrases: 0 | — | — |
| (none) | — | Empty implementations / console-log-only: 0 (the 3 `return []T{…}` grep hits are populated literals — SpawnArgv argv, env allowlist, a watcher envelope; verified by read) | — | — |

Debt-marker gate: clean. The scan-secrets.sh allowlist extension is deliberate, self-documenting (comment names plan 05-06 and the exact marker string), and the gate still fails on every other match (EXIT=0 with all other patterns active, verified first-hand).

### Human Verification Required

N/A — Infrastructure/foundation phase (internal codec/fake/supervisor/turn packages; no user-facing UI). All acceptance criteria verifiable programmatically, and every behavior-dependent truth has a passing behavioral test re-executed first-hand on the current tree.

The single harvested `<human-check>` block (03-07 real-desktop manual run) remains **discharged with machine evidence**: re-executed first-hand during this refresh (PASS 2.18s with observed layout/revision and ESRCH zero orphans — see Behavioral Spot-Checks). The no-install diagnostics path (the other side of SC3) is covered by `TestDiscoverNoInstallListsEveryProbe`, green first-hand.

### Gaps Summary

None. The 43/43 verdict survives the Phase 5 churn: the 4 changed covered files are additive/test-internal (envelope constants, test-helper extraction with assertions preserved, 6 new corpus drivers, one extra exact-string allowlist entry), no phase-3 invariant was weakened, and every behavior-dependent truth was re-proven first-hand under -race on the current tree (including the real-desktop positive path and the full E2E exec chain). All declared artifacts exist, are substantive, and are wired; the 4 artifact-checker misses and 1 key-link miss are the previously documented literal-pattern false positives, each re-confirmed by direct read on the current tree. Requirements BRIDGE-01..04 fully satisfied; no orphaned requirements; no debt markers; lint/experiments/tidy-diff/secret-scan/build/vet all exit 0 first-hand. Nothing deferred — no gaps arose for Step 9b filtering. One Info-level cosmetic finding recorded (dangling doc comment, conn_test.go:1305) — no action required for this phase's goal.

---

_Verified: 2026-11-21T06:03:18Z_
_Verifier: Claude (gsd-verifier)_
