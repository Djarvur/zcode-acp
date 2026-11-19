---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 07
subsystem: supervisor
tags: [spawn-contract, process-groups, staged-shutdown, crash-window, electron-as-node, proxy-scrub, realdesktop, manual-only, tdd]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-04)
    provides: the discovery half this plan spawns from — Install (Root/Binary/Entrypoint/BuiltinFile/Layout), Probers/Discover, ErrNoInstall diagnostics, BuiltinRevision, DefaultProbers
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-06)
    provides: the reexec helper idiom and exec-tier spawn hygiene (Setpgid + piped stdio + guaranteed reap) — the test harness conventions reused here
  - phase: 01-research-protocol-documentation
    provides: the canonical spawn line (install-discovery.md :48-52), the EOF-drain facts (dialect §1 :30), the D-03 lifecycle policy, and the Phase 2 CI blocker (the real desktop is manual-only)
provides:
  - internal/supervisor spawn half: SpawnArgv(Install) — the canonical argv [binary, entrypoint, app-server, --stdio]; ChildEnv(Install, parentEnv) — the explicit allowlist (PATH+HOME copied, ELECTRON_RUN_AS_NODE=1 + ZCODE_BUILTIN_PROVIDER_CONFIG_FILE set, proxy vars scrubbed by construction); SpawnChild(ctx, argv, env, opts) — Setpgid process-group start, all-3-stdio pipes, capped (64 KiB) stderr drain goroutine, group-kill Cancel + WaitDelay, single-Wait exit monitor delivering ExitInfo + the multi-observer Done signal
  - internal/supervisor supervise half: Supervisor.Start — health watch = stdin liveness + child exit (no protocol liveness frame invented, D-03), staged shutdown (stdin close → wait within grace → UNCONDITIONAL group SIGKILL → bounded reap wait), crash-window restart (3 attempts within a sliding window, BaseBackoff << (crashes-1), stable runs longer than the window reset the counter, fail loud naming crash count + window), all knobs injectable via Policy
  - internal/supervisor/realdesktop_test.go: the real-desktop positive path (locate → spawn → create → close, no orphans) behind //go:build realdesktop AND ZCODE_ACP_REAL_E2E=1 — manual-only, never CI, canonical command in the file header
affects: [phase 4 front leg (calls Supervisor.Start and wires the Conn over Child stdout/stdin), 03-09 E2E + phase gate (consumes the spawn path end-to-end), BRIDGE-03 completion (declared by 03-04/03-07/03-09)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 14795    # 59182 diff chars / 4 over 1ac2fa4..a097da3
  tasks: 3
  commits: 5      # MEASURED: git rev-list --count 1ac2fa4..a097da3 (2 RED + 2 GREEN + 1 realdesktop, all this plan's own)

commits: 5
plan_head_before: 1ac2fa457b85ddbd10f9496e1e1ec968a8975d7b
plan_head_after: a097da38e889854b96381475b70f699dfeb60761

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: tidy-diff green, go.mod untouched)
  patterns:
    - "Spawn seam + policy injection: NewSupervisor(SpawnFunc, Policy) — the restart loop re-calls the seam per attempt and every window knob (limit/window/backoff/grace) is a Policy field the tests shrink to milliseconds"
    - "Allowlist-scrub env contract: the child env is BUILT from an explicit allowlist (never filtered), so proxy-var scrubbing holds by construction and is asserted on the child's OBSERVED environ via a reexec helper keyed on the contract env itself"
    - "Multi-observer lifecycle signals: Child.Exited (single buffered delivery) + Child.Done (close-based, consumable by any number of watchers) — shutdown and tests never race a channel receive"
    - "Unconditional group-kill backstop: staged shutdown SIGKILLs -pgid even after a clean child exit (PITFALL 6: stdin-EOF is never trusted; orphaned grandchildren die either way), ESRCH-tolerated"
    - "Double-guarded manual tier: build tag excludes from default builds/vet/CI, runtime env guard skips the body — belt and braces per T-3-10"

key-files:
  created:
    - internal/supervisor/spawn.go
    - internal/supervisor/spawn_test.go
    - internal/supervisor/supervise.go
    - internal/supervisor/supervise_test.go
    - internal/supervisor/realdesktop_test.go
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-07-red-spawn.json
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-07-red-supervise.json
  modified: []

key-decisions:
  - "ChildEnv takes (Install, parentEnv []string), not the plan-artifact shorthand ChildEnv(Install): the Task-1 test mandates injecting the hostile parent env ('set them in cmd.Env input — the builder must remove them'), and a pure function over explicit input is the only race-free way to prove 'proxy vars absent even when the parent exports them' under -race (os.Setenv would mutate process-global state)"
  - "The reexec helper for TestSpawnEnv is keyed on ELECTRON_RUN_AS_NODE itself — the contract env doubles as the trigger, so the env handed to SpawnChild is ChildEnv's output VERBATIM and the child's observed key set is asserted as exactly the allowlist (nothing test-only rides along)"
  - "Child.Done (close-based) added beside the Exited delivery: shutdown, restart bookkeeping, and tests all observe reap completion without consuming the single-delivery channel — the double-receive race the staged shutdown would otherwise have"
  - "The group SIGKILL in the staged shutdown is UNCONDITIONAL (probe :667-674 precedent), not grace-expiry-only: a cleanly-exited Electron child may still leave orphaned group members, and stdin-EOF is never trusted (PITFALL 6)"
  - "Restart policy reads crashes > CrashLimit (ARCHITECTURE Pattern 4's fail-loud-past-limit), so the 3-attempt window makes 3 restarts and trips on the 4th consecutive in-window crash; the sliding reset keys on run lifetime > CrashWindow at crash time (RESEARCH Pitfall 7), pinned by the 5-vs-4-spawn trip arithmetic in TestRestartWindow"
  - "Default grace = 4 s (the probe's validated GRACE_MS), base backoff 1 s, WaitDelay 5 s — production defaults on Policy zero-values via withDefaults, so misconfigured callers still get the documented policy"

patterns-established:
  - "Contract-env-keyed reexec: a helper child triggered by the very env var under test proves builder OUTPUT (not builder intention) — reusable for any env-contract test"
  - "Poll-verified zero survivors: kill(-pgid, 0)/kill(pid, 0) ESRCH asserted by bounded polling (zombies exist until reaped), never by a single racy check"
  - "Trip-arithmetic pinning: spawn-count assertions (4 for the plain trip, 5 for the post-reset trip vs 4 without reset) make sliding-window resets observable, not narrated"

requirements-completed: [BRIDGE-03]  # COPIED VERBATIM from 03-07-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-03: 03-04/03-07/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Spawn contract proven on a real child: SpawnArgv returns exactly [Binary, Entrypoint, app-server, --stdio]; ChildEnv's output observed by the reexec child is exactly {PATH, HOME, ELECTRON_RUN_AS_NODE=1, ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=<injected builtin>} — proxy vars absent even though the hostile parent input exports all six forms, and pgid == pid (Setpgid group leader)"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/spawn_test.go#TestSpawnEnv + #TestSpawnArgvShape"
        status: pass
    human_judgment: false
  - id: D2
    description: "SpawnChild exec plumbing: Setpgid + all three stdio pipes + capped (64 KiB) stderr drain goroutine that keeps draining past the cap (PITFALL 2), group-kill Cancel + WaitDelay per stdlib semantics, and a single-Wait exit monitor gated on the stderr pump (PITFALL 3)"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/spawn_test.go#TestSpawnEnv (real child through SpawnChild) + internal/supervisor/supervise_test.go#TestGroupKillNoSurvivors (monitor reap under the full lifecycle)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Staged shutdown leaves ZERO surviving group members: stdin close → wait within grace → unconditional group SIGKILL → bounded reap wait; the helper's sleeping grandchild (same process group) and the child both report ESRCH by bounded poll, and the monitor reaps (Done closed, Exited delivered)"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/supervise_test.go#TestGroupKillNoSurvivors"
        status: pass
    human_judgment: false
  - id: D4
    description: "Crash-window restart with injectable knobs: 3 in-window attempts then fail loud naming the crash count (4) and window; a stable run longer than the window resets the sliding counter (trip at 5 spawns post-reset vs 4 without — RESEARCH Pitfall 7 encoded as arithmetic, not narrative)"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/supervise_test.go#TestRestartWindow (trips_loud_past_the_limit + stable_run_resets_the_sliding_window)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Real-desktop positive path (locate → spawn → create → close, no orphans) exists behind //go:build realdesktop AND skips without ZCODE_ACP_REAL_E2E=1; the canonical manual command is in the file header; default builds, vet, lint, and the default test suite are unaffected by the file's presence"
    requirement: BRIDGE-03
    verification:
      - kind: manual_procedural
        ref: "ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop ./internal/supervisor/ — END-OF-PHASE human verification (03-VALIDATION.md Manual-Only row); automated half proven: tagged run without the env var skips in 2 ms, go build/vet/lint/test all green with the file present"
        status: unknown
    human_judgment: true
    rationale: "the positive path contacts the REAL 3.18.2 desktop and is scheduled as the end-of-phase human check (human_verify_mode: end-of-phase, consolidated into UAT) — the executor never runs it (dispatch contract; Pitfall 10); only its CI-exclusion guards are machine-proven here"

# Metrics
duration: 26min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 07: Supervisor Lifecycle — Spawn Contract, Staged Shutdown, Crash Window Summary

**The OS-tier half of the back leg: the canonical ELECTRON_RUN_AS_NODE spawn line with an allowlisted proxy-scrubbed child env proven on a real child's observed environ, orphan-proof staged shutdown (unconditional group SIGKILL, ESRCH-verified zero survivors including grandchildren), and a sliding-window crash policy that fails loud past 3/60s and resets on stable runs — plus the double-guarded manual-only real-desktop positive path — all green under -race with zero new dependencies**

## Performance

- **Duration:** 26 min
- **Started:** 2026-11-19T20:04:17Z
- **Completed:** 2026-11-19T20:30:40Z
- **Tasks:** 3
- **Files:** 7 created (5 code/test + 2 RED evidence records)

## Accomplishments

- Task 1 (RED `6ed130a` → GREEN `1c21305`): the spawn contract — SpawnArgv reproduces the canonical argv exactly (install-discovery.md :48-52); ChildEnv builds the child env from an explicit allowlist (PATH+HOME copied, ELECTRON_RUN_AS_NODE=1 + ZCODE_BUILTIN_PROVIDER_CONFIG_FILE set, HTTP_PROXY/HTTPS_PROXY/ALL_PROXY scrubbed by construction, T-3-08/PITFALL 9); SpawnChild starts the child in its own process group (Setpgid), wires all three stdio pipes, drains stderr capped at 64 KiB (PITFALL 2), sets the group-kill Cancel + WaitDelay, and runs a single-Wait exit monitor gated on the stderr pump (PITFALL 3). TestSpawnEnv asserts the contract on the child's OBSERVED environment — the reexec helper is keyed on ELECTRON_RUN_AS_NODE itself so the child reports ChildEnv's output verbatim, the observed key set is exactly the allowlist, and pgid == pid
- Task 2 (RED `1e92291` → GREEN `e3e5ca6`): the lifecycle machinery — Supervisor.Start's health watch is stdin liveness + child exit (no protocol liveness frame invented, D-03); staged shutdown = stdin close → wait within grace (probe GRACE_MS=4000 default) → UNCONDITIONAL group SIGKILL → bounded reap wait; the restart policy makes 3 attempts inside a sliding window with exponential backoff, resets the counter after a stable run longer than the window (RESEARCH Pitfall 7, absent from both analogs), and fails loud naming the crash count and window. TestGroupKillNoSurvivors proves zero survivors through a real grandchild: the helper spawns a sleeping `sleep 30` in its group and the poll-verified ESRCH check covers child, group, and grandchild
- Task 3 (`a097da3`): internal/supervisor/realdesktop_test.go — the real-desktop positive path (DefaultProbers.Discover with probe-row logging → production SpawnArgv/ChildEnv/SpawnChild → zcode.Conn carrying session/create through the happy-path exchange with the router answering the create-time runtime-preferences obligation → staged shutdown → ESRCH zero-orphan check), behind //go:build realdesktop AND skipped without ZCODE_ACP_REAL_E2E=1; probe-time capture-header discipline records the observed layout + builtin revision; the canonical manual command is in the file header. MANUAL-ONLY: the tagged test skips in 2 ms without the env var and CI cannot reach the desktop by construction
- Full gates green: go build/vet (default AND -tags realdesktop), mise run lint 0 issues, go test -race -count=1 ./... all packages, mise run ci fully green, tidy-diff clean (go.mod untouched — T-3-SC)
- RED→GREEN machine-checked twice: evidence/03-07-red-spawn.json and evidence/03-07-red-supervise.json both classified RED_EVIDENCE_OK before their GREEN commits; lifecycle suite stable across -count=5 repeats

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — failing spawn env + argv tests** - `6ed130a` (test)
2. **Task 1: GREEN — spawn contract with allowlisted env + process-group start** - `1c21305` (feat)
3. **Task 2: RED — failing group-kill + restart-window tests** - `1e92291` (test)
4. **Task 2: GREEN — staged shutdown, group kill, sliding-window restart** - `e3e5ca6` (feat)
5. **Task 3: env-guarded real-desktop positive path (manual-only)** - `a097da3` (feat)

_Note: TDD tasks carry multiple commits (test → feat)_

**Plan metadata:** pending (docs commit lands with this SUMMARY)

## Files Created/Modified

- `internal/supervisor/spawn.go` - the spawn contract: SpawnArgv, ChildEnv (allowlist + scrub), SpawnChild (Setpgid, pipes, capped stderr drain, group-kill Cancel, exit monitor with Exited/Done signals)
- `internal/supervisor/spawn_test.go` - TestSpawnEnv (observed-env assertions through the contract-env-keyed reexec helper), TestSpawnArgvShape, TestSpawnEnvHelper
- `internal/supervisor/supervise.go` - Policy (+ withDefaults, D-03 production values), Supervisor.Start (health watch, restart loop), staged shutdown (unconditional group kill), fail-loud crash-loop error
- `internal/supervisor/supervise_test.go` - TestGroupKillNoSurvivors (grandchild included, poll-verified ESRCH), TestRestartWindow (trip + sliding reset arithmetic), the helper modes (crash/stay/grandchild)
- `internal/supervisor/realdesktop_test.go` - the double-guarded manual positive path with probe-row logging, capture-header install facts, and the canonical command
- `evidence/03-07-red-spawn.json`, `evidence/03-07-red-supervise.json` - RED records, both RED_EVIDENCE_OK

## Decisions Made

- ChildEnv signature is (Install, parentEnv []string): the plan's own test needs to inject the hostile parent env, and a pure function over explicit input is the only race-free way to prove the scrub on observed output under -race (documented as a deviation below)
- The reexec helper is keyed on the contract env itself (ELECTRON_RUN_AS_NODE=1) rather than a test-only trigger var, so the env passed to SpawnChild is the builder's output verbatim and the exact-key-set assertion has no test-only escape hatch
- Child.Done (close-based multi-observer) added beside the single-delivery Exited channel — shutdown, restart bookkeeping, and tests observe reap completion without racing a receive
- The staged shutdown's group SIGKILL is unconditional (probe :667-674 precedent): even a cleanly-exited child's orphaned group members die; ESRCH is tolerated
- Restart arithmetic: crashes > CrashLimit (ARCHITECTURE Pattern 4's fail-loud-past-limit) — 3 attempts made, the 4th consecutive in-window crash trips; the sliding reset compares run lifetime against the window at crash time
- Grace default 4 s (probe GRACE_MS), backoff 1 s, WaitDelay 5 s — all Policy-injectable; tests shrink the window to 150 ms and the grace to 100 ms

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] ChildEnv takes the parent env as an explicit parameter**
- **Found during:** Task 1 (writing the RED test the plan specifies)
- **Issue:** the plan artifact shorthand reads `ChildEnv(Install) []string`, but the Task-1 action mandates "the three proxy variables scrubbed (set them in cmd.Env input — the builder must remove them)" asserted on the child's OBSERVED environment — impossible race-free with an os.Environ()-reading builder (os.Setenv is process-global mutation under -race)
- **Fix:** ChildEnv(inst Install, parentEnv []string) []string — a pure allowlist builder over explicit input; the plan's own must-have truth ("asserted on the real child's observed environment, not on the builder's intention") is what forces the injectable input
- **Files modified:** internal/supervisor/spawn.go, internal/supervisor/spawn_test.go
- **Committed in:** 6ed130a / 1c21305

**2. [Rule 3 - Blocking] Added Child.Done (multi-observer reap signal) to the Task-1 contract shape**
- **Found during:** Task 2 (RED tests + shutdown design)
- **Issue:** Exited is a single-delivery channel; the staged shutdown's stage-2 wait and the tests' reap assertion would race each other consuming it (double-receive on a cap-1 channel stalls one side)
- **Fix:** the monitor closes Done after Wait returns (close is observable by any number of watchers); Exited stays the single delivery, received only after Done — both landed in spawn.go's monitor
- **Files modified:** internal/supervisor/spawn.go
- **Committed in:** 1e92291 (shape) / e3e5ca6 (consumers)

**3. [Rule 1 - Bug] Task-2 RED test fixes surfaced during GREEN**
- **Found during:** Task 2 (first GREEN run)
- **Issue:** (a) the readiness-line parser compared against "ready " (trailing space) while the helper's prefix was matched as a full field — the helper's own line was rejected as "unexpected"; (b) the sliding-reset spawn arithmetic was off by one: the stable run's own crash IS post-reset crash 1', so the trip lands at 5 spawns (vs 4 without reset), not 6
- **Fix:** grandchildReadyPrefix normalized to "ready" with an explicit space in the format string; resetTripSpawnCount corrected to 5 with the no-reset contrast (4) documented in the failure message
- **Files modified:** internal/supervisor/supervise_test.go
- **Verification:** green under -race x5 after the fix
- **Committed in:** e3e5ca6

**4. [Rule 3 - Blocking] Strict-gate lint findings across all three tasks**
- **Found during:** every task (mise run lint before each commit)
- **Issue:** the strict set flagged dogsled/err113/funlen/goconst/gocritic-filepathJoin/godoclint/lll/mnd/unused on the RED+GREEN code, then noctx/revive-context-as-argument/contextcheck/errcheck/tagliatelle on the tagged realdesktop file (found by an informational `golangci-lint run --build-tags realdesktop`)
- **Fix:** static sentinels instead of dynamic errors; context first in test helpers; exec.CommandContext in the grandchild helper; named constants for magic numbers; detached file comments (single package godoc); map-based session.sessionId decode (a snake_case struct tag would misname the wire contract); `_ =` on the deferred conn.Close
- **Files modified:** all five files above
- **Verification:** mise run lint 0 issues (default AND -tags realdesktop)
- **Committed in:** across 6ed130a/1c21305/1e92291/e3e5ca6/a097da3

---

**Total deviations:** 4 auto-fixed (1 Rule 1, 3 Rule 3)
**Impact on plan:** none on scope — every deviation was forced by the plan's own test contract, the channel-race reality of the shutdown design, or the repo's strict gates. All must-have truths verified.

## Issues Encountered

- None beyond the lint-gate iteration and the two RED-test bugs above — the reexec idiom stayed stable under -race (x5 repeats on both task suites), and the group-kill ESRCH poll settled within its first tick in every run

## User Setup Required

None - no external service configuration required.

## Human Verification (end-of-phase, NOT run by this plan)

- **Real-desktop positive path** (coverage D5, human_judgment: true): on this machine (ZCode 3.18.2 install present), run once after the phase's plans complete:
  `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop ./internal/supervisor/`
  Expect PASS with the observed install layout + builtin revision logged and zero orphaned processes afterwards (`pgrep -f zcode.cjs` empty). Without the env var AND the tag, the test is skipped/absent — CI never contacts the desktop.

## Next Phase Readiness

- Plan 03-08 (turn completion): unaffected — the fake binary serves the completion variants identically in-process and over exec pipes
- Plan 03-09 (E2E + phase gate): composes Discover → Supervisor.Start → Conn exactly as realdesktop_test.go demonstrates, with the fake binary as the CI child (the exec-tier harness from 03-06); the phase gate's manual row is the end-of-phase human check documented above
- Phase 4 front leg: calls Supervisor.Start(ctx) with a SpawnFunc that wires its Conn over Child.Stdout/Stdin (the childDuplex pattern in realdesktop_test.go is the reference); restart blasts live sessions by design (ARCHITECTURE Pattern 4 trade-off) — surface as failed turns, not silent replay
- The one inherited known stub is unchanged: discover.go's defaultOverride remains a documented no-op until the wrapper's config surface exists (Phase 4 scope, 03-04 Known Stubs)

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| *(none new — this plan introduces no stubs; the inherited defaultOverride no-op from 03-04 remains Phase 4's config-surface work)* | | | |

## Self-Check: PASSED

- 03-07-SUMMARY.md exists on disk
- All claimed files exist (FOUND x7): spawn.go, spawn_test.go, supervise.go, supervise_test.go, realdesktop_test.go, evidence/03-07-red-spawn.json, evidence/03-07-red-supervise.json
- All claimed commits are ancestors of HEAD: 6ed130a (T1 RED), 1c21305 (T1 GREEN), 1e92291 (T2 RED), e3e5ca6 (T2 GREEN), a097da3 (T3)
- Ledger-measured plan commit count: 5 (all this plan's own; plan_head_before 1ac2fa4, plan_head_after a097da3)
- Acceptance greps: Setpgid in spawn.go = 4 (>= 1); SIGKILL in supervise.go = 5 (>= 1); ZCODE_ACP_REAL_E2E in realdesktop_test.go = 4 (>= 1)
- TDD gate: both RED evidence records RED_EVIDENCE_OK, each captured from the final committed RED state before its feat commit
- Final gate run at close-out: go build ./... && go vet ./... green (default AND -tags realdesktop); mise run ci fully green (build → vet → lint 0 issues → test -race all packages → test-experiments); tidy-diff clean; lifecycle suite green across -count=5

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
