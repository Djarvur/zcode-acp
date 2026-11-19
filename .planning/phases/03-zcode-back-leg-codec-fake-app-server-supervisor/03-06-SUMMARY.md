---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 06
subsystem: testing
tags: [fake-app-server, standalone-binary, reexec, exec-plumbing, eof-drain, process-groups, failure-signature, d-01]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-01)
    provides: the fake scripting core (Serve over any io.ReadWriteCloser, Scenario schema, EOF drain, clean-EOF classification) — the one core this plan hosts in its second mode
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (plan 03-03)
    provides: the fail-loud loader (LoadScenario), the 10-scenario corpus (happy-turn served here), and the errZeroFrameExchange zero-lines classifier the exec tier reuses
  - phase: 01-research-protocol-documentation
    provides: docs/protocol/app-server-dialect.md §1 :30 — client EOF drain facts and the exit-0-with-zero-lines failure signature
provides:
  - cmd/fake-app-server standalone binary (D-01 mode B): --scenario flag, stdio serving over the same scripting core, fail-loud usage exit 2, EOF-drain exit 0, diagnostics stderr-only — the CI child 03-07 spawns (the real Electron bundle is manual-tier only)
  - internal/zcode/fake/stdio.go: exported fake.Main hosting entry + ExitUsage/ExitServeFailed exit-code contract shared by the cmd main and the reexec helper (one scripting core, two hosting modes)
  - internal/zcode/fake/binary_test.go: the helper-process reexec integration tier — TestFakeBinarySpawnPath (frames both directions over real os.Pipe, observable drain window, clean exit 0) and TestEOFSignatureChild (real child's exit-0-with-zero-lines reported FAILURE via the shared classifier), spawned with Setpgid + stderr capture + guaranteed reap
affects: [03-07 supervisor spawn/supervise, 03-08 turn completion, 03-09 E2E + phase gate, QA-01 conformance suite (phase 7)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 4600    # 18570 added chars over this plan's diff / 4
  tasks: 2
  commits: 2     # MEASURED from the plan ledger (42d1121..2984acf): both this plan's own

commits: 2
plan_head_before: 42d1121a148f854d53368df7863db6503977a728
plan_head_after: 2984acf2608d5ad5912cc6a0abef60e30c68c8f7

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: tidy-diff green, go.mod untouched)
  patterns:
    - "Helper-process reexec idiom (GO_WANT_HELPER_PROCESS=1 + -test.run targeting the helper): the child IS the fake binary via the same fake.Main call the cmd main makes — GOROOT-verified convention, stable under -race x5"
    - "Exec-tier spawn hygiene: os.Pipe protocol pipes, Setpgid process group, stderr captured into a buffer exec drains itself, group-kill backstop, exactly one Wait per path — no orphan survives a failure path"
    - "One scripting core, two hosting modes: the standalone binary is a thin main over fake.Serve — the in-process golden tests and the exec-tier tests pin the same wire behavior"
    - "Observable-drain assertion: exit elapsed-time floor (90 ms) proves the eof_drain_ms window ran, never an instant exit masquerading as a drain"

key-files:
  created:
    - cmd/fake-app-server/main.go
    - internal/zcode/fake/stdio.go
    - internal/zcode/fake/binary_test.go

key-decisions:
  - "The shared stdio entry lives in the fake package (fake.Main in stdio.go), not in cmd: the plan's Task-2 action mandates the helper serve via 'the scripting core's stdio entry (the same call the cmd main makes)' and cmd packages cannot be imported — one entry, two hosts, exit codes exported as named constants"
  - "The exec-tier zero-lines test is named TestEOFSignatureChild: TestEOFSignature already exists in-package (03-03's in-process half of the same contract); the plan's own verify regex 'TestFakeBinary|TestEOFSignature' selects both, and the new test reuses the SAME errZeroFrameExchange classifier so there is exactly one verdict function"
  - "Raw protocol driving over the real pipes (zcode.Encode/Decode envelopes, not zcode.Conn): the exec tier's subject is the PLUMBING, not the codec — the in-process corpus already pins codec+fake semantics, and raw driving keeps this tier honest about what it proves"
  - "Drain observability asserted by elapsed-time floor (>= 90 ms between parent stdin close and child exit): the child cannot exit before its 100 ms eof_drain_ms sleep elapses, so the floor proves the drain ran without racing wall clocks"

patterns-established:
  - "Reexec helper contract: a test binary re-executed with GO_WANT_HELPER_PROCESS=1 IS the fake binary — scenario path via FAKE_SCENARIO env, exit code = fake.Main's return; 03-07's spawn tests reuse the shape"
  - "fakeChild harness: reader goroutine feeds a buffered line channel (reader never blocks a test that stops reading), stderr via exec's own copy goroutine joined by Wait, terminate() backstop kills the process group and reaps"

requirements-completed: [BRIDGE-02]  # COPIED VERBATIM from 03-06-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-02: 03-01/03-03/03-06/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Standalone fake binary (D-01 mode B): cmd/fake-app-server over fake.Serve — --scenario required, exit 2 fail-loud with one clear stderr line on missing/invalid scenario, stdout carries protocol frames only, diagnostics stderr-only via slog, exit 0 through the EOF-drain window"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "manual smoke: no flag -> rc=2 + stderr line; bad path -> rc=2; happy-turn with /dev/null stdin -> rc=0 with ZERO stdout bytes (TestFakeBinarySpawnPath#TestEOFSignatureChild cover the serving path end-to-end)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Exec-plumbing proven against a REAL child process: reexec'd test binary drives the happy-turn exchange over os.Pipe both directions in wire order (create, runtime-preferences reverse exchange, create reply, send, notifications, ack), child spawned with SysProcAttr Setpgid, stderr captured, exit reaped — no orphaned test child"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/binary_test.go#TestFakeBinarySpawnPath"
        status: pass
    human_judgment: false
  - id: D3
    description: "EOF drain contract observable at the exec tier: after stdin close the child exits 0 with elapsed time >= the 90 ms floor (the scenario's eof_drain_ms window ran — never an instant exit)"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/binary_test.go#TestFakeBinarySpawnPath (drain-elapsed assertion)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Exit-0-with-zero-lines failure signature enforced at the exec tier: a real child exiting 0 with ZERO captured frames is reported FAILURE by the shared errZeroFrameExchange classifier — never an empty success (dialect §1 :30, probe guard precedent)"
    requirement: BRIDGE-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/binary_test.go#TestEOFSignatureChild"
        status: pass
    human_judgment: false

# Metrics
duration: 7min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 06: Standalone Fake Binary + Reexec Exec-Plumbing Tier Summary

**The fake app-server now ships as a real child process: cmd/fake-app-server serves any corpus scenario over its own stdio with the EOF-drain exit contract, and a helper-reexec integration tier proves frames flow both directions over real os.Pipe pipes in a Setpgid process group — with the exit-0-with-zero-lines failure signature enforced against a live child, all under -race with zero new dependencies**

## Performance

- **Duration:** 7 min
- **Started:** 2026-11-19T19:44:09Z
- **Completed:** 2026-11-19T19:51:25Z
- **Tasks:** 2
- **Files:** 3 created (cmd main, fake stdio hosting entry, reexec integration tier)

## Accomplishments

- Task 1: cmd/fake-app-server — stdlib `--scenario` flag (required), fail-loud usage contract (missing flag or failed LoadScenario → one clear stderr line naming the problem, exit 2, the IN-04 precedent), serving over stdin/stdout via the scripting core's new exported hosting entry fake.Main (stdio.go), stdout carrying protocol frames only with every diagnostic on stderr via slog, and the dialect §1 EOF-drain exit: verified by smoke run (rc=2/rc=2/rc=0 with zero stdout bytes on stdin EOF)
- Task 2: internal/zcode/fake/binary_test.go — the reexec integration tier: TestFakeBinarySpawnPath re-executes the test binary as the fake (GO_WANT_HELPER_PROCESS=1, the same fake.Main call the cmd main makes), drives the full happy-turn exchange through the real exec pipes in wire order (create → lifecycle emissions → runtime-preferences reverse request → verbatim-string-id answer → create reply → send → patch/events → ack), asserts the frame counts both directions, closes stdin, and proves the drain window observable (exit elapsed >= 90 ms, code 0) with the child reaped; TestEOFSignatureChild runs a real zero-traffic child — exit 0, zero captured lines — and asserts the shared errZeroFrameExchange classifier reports FAILURE, never an empty success
- Spawn hygiene locked at the exec tier (T-3-07): os.Pipe stdin/stdout, child in its own process group (Setpgid), stderr captured and logged, group-kill backstop in cleanup, exactly one Wait per path — the identical plumbing 03-07's supervisor formalizes
- Full gates green: go build/vet, mise run lint (0 issues), go test -race ./internal/zcode/fake/ (corpus + reexec tier), mise run ci fully green (build → vet → lint → test → test-experiments), tidy-diff clean — go.mod untouched (T-3-SC)
- Reexec idiom stable under -race (x5 repeats) — the blessed A2 built-binary fallback was NOT needed

## Task Commits

Each task was committed atomically:

1. **Task 1: cmd/fake-app-server — scenario flag, stdio serving, EOF drain exit** - `6a88891` (feat)
2. **Task 2: reexec integration tier — spawn-path frames, EOF drain, zero-lines failure signature** - `2984acf` (test)

## Files Created/Modified

- `cmd/fake-app-server/main.go` - thin main: --scenario flag, fail-loud usage exit 2, os.Exit(fake.Main(...)) — never touches stdout except protocol frames (via the core)
- `internal/zcode/fake/stdio.go` - fake.Main hosting entry (mode B of the scripting core): LoadScenario + Serve over the raw stdio pair (Close closes stdin only, never stdout), exit-code constants ExitUsage/ExitServeFailed, slog diagnostics to stderr
- `internal/zcode/fake/binary_test.go` - the exec tier: fakeChild harness (pipes, Setpgid, stderr capture, reap backstop), TestFakeBinarySpawnPath, TestEOFSignatureChild, raw-frame driving helpers

## Decisions Made

- Shared stdio entry lives in the fake package (stdio.go: fake.Main + named exit-code constants) — the plan's Task-2 text mandates the helper use "the scripting core's stdio entry (the same call the cmd main makes)", and cmd packages are not importable; one entry guarantees binary and helper behavior are identical by construction
- Exec-tier zero-lines test named TestEOFSignatureChild — TestEOFSignature already exists in-package (03-03's in-process half of the SAME contract) and a duplicate symbol is a compile error; the plan's own verify regex 'TestFakeBinary|TestEOFSignature' selects both, and the child test reuses the same errZeroFrameExchange classifier (exactly one verdict function in the suite)
- The parent side drives RAW protocol envelopes (zcode.Encode/Decode), not zcode.Conn — the exec tier's subject is the plumbing; the in-process corpus already pins codec semantics, and the raw driving keeps this tier's proof honest about scope
- Drain observability via an elapsed-time floor (>= 90 ms from parent stdin close to child exit): the child's eof_drain_ms sleep makes an instant exit impossible, so the floor proves the drain without racing wall clocks
- The happy exchange asserts exact frame counts both directions (parent writes 3, child writes 8) — wire-order breakage fails at the specific frame via the wantMethod/wantReply helpers

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added internal/zcode/fake/stdio.go with the exported fake.Main hosting entry**
- **Found during:** Task 1 (implementing the cmd main)
- **Issue:** the plan's Task-2 action requires the helper to serve via "the scripting core's stdio entry (the same call the cmd main makes)", but the stdio wiring cannot live in package main (cmd packages cannot be imported); without a shared entry the binary and the helper would duplicate the wiring and could drift
- **Fix:** exported fake.Main(scenarioPath string) int in internal/zcode/fake/stdio.go — loads the scenario, serves over the raw stdio pair, returns the process exit code; named exit-code constants (ExitUsage=2, ExitServeFailed=1) so cmd main and helper agree by construction
- **Files modified:** internal/zcode/fake/stdio.go (new)
- **Verification:** both tasks' gates green; the reexec helper and the smoke-run binary exercise the identical call path
- **Committed in:** 6a88891 (Task 1 commit)

**2. [Rule 3 - Blocking] Exec-tier zero-lines test named TestEOFSignatureChild instead of TestEOFSignature**
- **Found during:** Task 2 (creating binary_test.go)
- **Issue:** the plan's test name TestEOFSignature collides with the in-process test of the same name landed by 03-03 in the same package (golden_test.go) — a duplicate symbol is a compile error
- **Fix:** named the child-process proof TestEOFSignatureChild; it reuses the SAME errZeroFrameExchange classifier and checkExchangeRecorded verdict function from golden_test.go, and the plan's verify regex 'TestFakeBinary|TestEOFSignature' selects both tests unchanged
- **Files modified:** internal/zcode/fake/binary_test.go
- **Verification:** go test -race -run 'TestFakeBinary|TestEOFSignature' green (all three tests: spawn path, in-process signature, child signature)
- **Committed in:** 2984acf (Task 2 commit)

**3. [Rule 3 - Blocking] Strict-gate lint findings on the new test code (4 findings across two lint passes)**
- **Found during:** Task 2 (mise run lint)
- **Issue:** errcheck flagged an error-typed channel receive (`<-done`), noctx required os/exec.CommandContext over Command, thelper required t.Helper() in terminate, and staticcheck S1005 rejected the blank-identifier receive that fixed errcheck
- **Fix:** restructured waitExit with a separate waitDone completion channel (struct{} — satisfies errcheck and S1005), switched the spawn to exec.CommandContext with a cancel stored on the harness (terminate/waitExit call it before the group kill), added t.Helper() to terminate
- **Files modified:** internal/zcode/fake/binary_test.go
- **Verification:** mise run lint — 0 issues; exec-tier tests green x3
- **Committed in:** 2984acf (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (3 Rule 3 blocking: 1 plan-mandated shared entry, 1 name collision, 1 lint-gate)
**Impact on plan:** all fixes were required by the plan's own task text (shared stdio entry), the already-landed sibling state (test-name collision), or the repo's strict lint gate. No scope creep; all four must-have truths verified.

## Issues Encountered

- None beyond the lint-gate iteration above — both exec-tier tests passed under -race on their first run and stayed green across x5 repeats, discharging RESEARCH assumption A2 (reexec-under-race flakiness) empirically; the built-binary fallback was not needed

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03-07 (spawn/supervise): the fake binary is the CI child — spawn it with the canonical argv shape, apply the env allowlist + proxy scrub, reuse the Setpgid + group-kill + reap discipline this tier already exercises; the reexec contract (GO_WANT_HELPER_PROCESS/FAKE_SCENARIO) is the template for 03-07's own child-process tests
- Plan 03-08 (turn completion): the binary serves completion-idle-patch/completion-quiet-window scenarios identically to the in-process fake — exec-tier turn proofs need no new scripting
- Plan 03-09 (E2E + phase gate): TestGolden remains the replay-all entry point; the exec tier adds the child-process layer the E2E gate can drive without the desktop bundle
- BRIDGE-02 completion remains intentionally pending: declared by 03-01/03-03/03-06/03-09, so requirements.ready-ids returns [] until 03-09's SUMMARY lands

## Self-Check: PASSED

- 03-06-SUMMARY.md exists on disk
- All claimed files exist: cmd/fake-app-server/main.go, internal/zcode/fake/stdio.go, internal/zcode/fake/binary_test.go (FOUND x3)
- All claimed commits are ancestors of HEAD: 6a88891 (Task 1), 2984acf (Task 2)
- Acceptance greps: slog in main.go = 2 (>= 1); Setpgid in binary_test.go = 3 (>= 1); GO_WANT_HELPER_PROCESS = 2 (>= 1); tidy-diff clean; mise run lint 0 issues
- Ledger-measured plan commit count at SUMMARY write: 2 (both this plan's own; plan_head_before 42d1121, plan_head_after 2984acf)
- Final gate run: mise run ci fully green (build, vet, lint 0 issues, go test -race all packages, test-experiments 22/22); exec-tier suite stable across 5 repeats

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
