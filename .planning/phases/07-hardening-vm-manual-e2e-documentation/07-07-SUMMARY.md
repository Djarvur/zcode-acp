---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 07
subsystem: testing
tags: [bash, vm-e2e, acp, stdio-jsonrpc, keyring-sentinels, qa]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: the ACP stdio arc the script drives (initialize -> session/new -> prompt) and the root-mode agent serving ACP v1 over stdio
  - phase: 06-oauth-coding-plan-path-model-picker
    provides: the coding-plan authenticate chain, the four keyring probe sentinels (internal/interact/keyringprobe.go), and the paired provider-config env fact (ChildEnv)
  - phase: 07-hardening (07-04)
    provides: the omp CLI dispatch + the 18.1.9+zacp.0 version stamp the run stage's floor assert rides
provides:
  - scripts/vm-e2e.sh — the QA-02/D-05 clean-VM scenario: discovery + run + no-keyring + both auth stages, named PASS/FAIL/SKIP checks, bounded reads, re-run safe
  - experiments/vm-e2e-script.test.mjs — the script's contract suite (sentinel table verbatim, timeout bound, guarded live stripped-PATH arm)
  - the four keyring sentinels asserted byte-for-byte against runtime output; the paired-env fact asserted by read-back
affects: [07-08, 07-09 (the VM field execution human-check rides the phase close), docs (DOCS-02 references the scenario)]

# Actuals (#2632)
actuals:
  tokens: 10743    # 42973 diff chars / 4
  tasks: 3
  commits: 4       # MEASURED: git rev-list --count 611d281..HEAD (before the docs commit)

# Tech tracking
tech-stack:
  added: []        # bash + node:test only — the repo's existing gates, no new dependencies
  patterns:
    - "coproc stdio JSONL driver with per-read bounds (read -t): a hung wrapper is a FAIL, never a hang"
    - "coproc pid captured into an own variable at spawn (bash unsets <NAME>_PID after reaping)"
    - "self-invoked internal arm under timeout(1) for belt-and-braces stage bounds"

key-files:
  created:
    - scripts/vm-e2e.sh
    - experiments/vm-e2e-script.test.mjs
    - .planning/phases/07-hardening-vm-manual-e2e-documentation/07-07-red-evidence.json
  modified: []

key-decisions:
  - "Sentinel classification uses grep -F fixed strings on captured replies; the four sentences are carried character-for-character from keyringprobe.go (the no-desktop-session sentence joins its two Go source halves exactly as the runtime prints)"
  - "The stripped-PATH no-keyring arm runs via a self-invoked internal stage (nokeyring-arm) under timeout(1) so the bounded attempt never hangs the stage and the caller's env is never mutated"
  - "The coding-plan read-back asserts the session/new reply's model list for account:zai-individual-coding-plan/ entries — the reply, never any ack (the phase-6 root cause signature)"
  - "Auth-stage preconditions skip-not-fail by default (--require-auth hardens them), so a machine without a key or keyring still yields a green scenario"

patterns-established:
  - "Named-check bash gate: check <name> <pass|fail|skip> + a summary line naming every outcome and the first FAIL; exit 0/1/2"
  - "Per-stage flags (--stage discovery|run|no-keyring|auth-apikey|auth-codingplan|all) so stages are individually runnable"
  - "Env-guarded live test (ZCODE_ACP_VM_E2E_BIN) riding node --test — the test/realdesktop guarded-harness precedent"

requirements-completed: [QA-02]

coverage:
  - id: D1
    description: "scripts/vm-e2e.sh — the clean-VM scenario (discovery -> run -> no-keyring -> both auth stages) with named checks, the four keyring sentinels asserted verbatim, and the paired-env read-back assert"
    requirement: QA-02
    verification:
      - kind: unit
        ref: "experiments/vm-e2e-script.test.mjs#no-keyring stage asserts the four coded keyring sentinels verbatim"
        status: pass
      - kind: unit
        ref: "experiments/vm-e2e-script.test.mjs#no-keyring stage wraps the attempt in a bounded timeout(1)"
        status: pass
      - kind: integration
        ref: "bash scripts/vm-e2e.sh --binary /tmp/zcode-acp-vm-check (full default run: 16 PASS, 1 SKIP, exit 0 — discovery, version floor, initialize, session/new, prompt arm, stripped-PATH binary-missing sentinel, coding-plan authenticate + account-models read-back)"
        status: pass
      - kind: integration
        ref: "ZCODE_ACP_VM_E2E_BIN=<bin> node --test experiments/vm-e2e-script.test.mjs (live stripped-PATH arm surfaces the binary-missing sentinel, 3/3 pass)"
        status: pass
    human_judgment: false
  - id: D2
    description: "One documented execution of scripts/vm-e2e.sh on an actual clean Ubuntu GNOME VM (the field proof for QA-02; the script runs the dev machine green but a clean VM run is the point of the deliverable)"
    requirement: QA-02
    verification: []
    human_judgment: true
    rationale: "A clean-VM field execution is inherently manual (no toolchain on the VM, a human runs the script there); the plan assigns this human-check to the phase-close plan 07-09 with the ledger recording."

# Metrics
duration: 27min
completed: 2026-11-22
status: complete
---

# Phase 7 Plan 07: VM & Manual E2E Scenario Script Summary

**scripts/vm-e2e.sh — the QA-02/D-05 clean-VM scenario scripted in-repo: discovery -> run -> the four verbatim keyring sentinels -> both auth paths with the paired-env read-back, all named-check gated and bounded**

## Performance

- **Duration:** 27 min
- **Started:** 2026-11-22T13:25:15Z
- **Completed:** 2026-11-22T13:52:00Z
- **Tasks:** 3 (tracer + TDD task producing RED/GREEN + auto task)
- **Files modified:** 3 (2 created + 1 planning evidence record)

## Accomplishments

- `scripts/vm-e2e.sh` (the first script outside experiments/): the whole v1 scenario — discovery (mirrors the wrapper's probe order read-only), run (version floor >= 18.1.9, the scripted stdio ACP arc with the prompt arm RECORDED), no-keyring (the four coded sentinels, binary-missing proven live via a stripped-PATH arm), auth-apikey and auth-codingplan (both read-back asserted)
- The four `ErrKeyring*` sentences of internal/interact/keyringprobe.go:44-60 carried character-for-character and matched with grep -F; the stripped-PATH arm surfaces the exact binary-missing sentence on this machine (and deterministically on any machine — the probe's LookPath gate answers before any subprocess)
- The paired provider-config env fact asserted BY READ-BACK: after authenticate{zai-individual-coding-plan} the session/new reply must list `account:zai-individual-coding-plan/` models — proven live here; the create ack alone proves nothing (the phase-6 root cause)
- Runtime-only dependency posture (Pitfall 6): build-or-fetch documented in the header, no toolchain install, no downloads, no package installs — presence asserts with actionable hints only; env-var NAMES only in output (scan-secrets green)

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — the VM scenario skeleton** - `5e641a8` (feat)
2. **Task 2 (TDD RED): no-keyring sentinel table + bounded-attempt contract** - `80f48aa` (test)
3. **Task 2 (TDD GREEN): no-keyring stage — sentinels verbatim, binary-missing live** - `5de5b49` (feat)
4. **Task 3: both auth-path stages + the paired-env read-back** - `7e3a0b3` (feat)

**Plan metadata:** docs commit (this SUMMARY + STATE/ROADMAP/REQUIREMENTS).

## Files Created/Modified

- `scripts/vm-e2e.sh` - the clean-VM scenario script (named-check gate, five stages + the internal bounded arm, ~780 lines)
- `experiments/vm-e2e-script.test.mjs` - the script's contract suite (sentinel verbatim table, timeout bound, guarded live arm)
- `.planning/phases/07-hardening-vm-manual-e2e-documentation/07-07-red-evidence.json` - the TDD RED evidence record (classifier RED_EVIDENCE_OK)

## Decisions Made

- Sentinel classification over captured replies with grep -F fixed strings — never paraphrase, never regex; byte-identity to keyringprobe.go verified (three sentences direct, the no-desktop-session one via its two concatenated source halves)
- The bounded no-keyring attempt is a self-invoked internal stage (`--stage nokeyring-arm --internal-arm`) under timeout(1) — belt-and-braces over the probe's own 5s KeyringProbeTimeout; the stripped dir is built inside the child invocation so the caller's env is never mutated
- Auth-stage preconditions (ZCODE_ACP_API_KEY unset, secret-tool absent) SKIP with an actionable reason rather than FAIL, hardened by --require-auth — a machine without a credential still gets a green, honest scenario
- The prompt leg RECORDS the observed arm (completed vs clean auth error) instead of assuming auth state — on this logged-in desktop one minimal real turn completes; no quota is spent on an unauthenticated machine (the turn errors cleanly)

## TDD Gate Compliance

- RED: `80f48aa` `test(7-07): no-keyring sentinel table + bounded-attempt contract (RED)` precedes GREEN: `5de5b49` `feat(7-07): no-keyring stage ...` — the gate sequence holds.
- RED evidence: classifier verdict `RED_EVIDENCE_OK` (reason `target_test_failed`); command `node --test --test-reporter=tap --test-reporter-destination=stdout experiments/vm-e2e-script.test.mjs`, exit 1; target test `no-keyring stage asserts the four coded keyring sentinels verbatim` failed on its planned assertion (the binaryMissing sentence containment assert, ERR_ASSERTION at the expected sentinel loop) — not a load/syntax/fixture failure; 3 tests discovered, 2 failed on planned assertions, 1 guarded skip.
- semanticAssessment: the target executed and failed for the intended reason (the stage was a not-yet-scripted stub carrying none of the sentinel sentences); GREEN implemented the stage and the same suite passes 3/3 with the guarded live arm.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] EXIT trap read a bash-managed coproc variable that bash unsets after reaping**
- **Found during:** Task 1 (run stage)
- **Issue:** the trap's `[ -n "$AGENT_PID" ]` hit "unbound variable" under set -u after the wrapper exited — bash unsets a named coprocess's `<NAME>_PID` once it is reaped — flipping a green run's exit code to 1
- **Fix:** capture the pid into `RUN_PID` at coproc spawn and use it everywhere (expect loop, stop, trap)
- **Files modified:** scripts/vm-e2e.sh
- **Verification:** full run stage green with exit 0 (was: PASS summary then rc=1)
- **Committed in:** 5e641a8 (task 1 commit)

**2. [Rule 1 - Bug] Summary line printed lowercase statuses, violating the declared output contract**
- **Found during:** Task 1 (run stage)
- **Issue:** the summary printed `pass <name>` while the header contract (and the plan's named-check grep) requires `PASS <name>` / `FAIL <name>` / `SKIP <name>`
- **Fix:** explicit uppercase mapping in the summary loop
- **Files modified:** scripts/vm-e2e.sh
- **Verification:** summary lines read PASS/SKIP; the stage greps hold
- **Committed in:** 5e641a8 (task 1 commit)

**3. [Rule 1 - Bug] The internal arm self-invocation used an unregistered stage name and ran without the internal guard**
- **Found during:** Task 2 (no-keyring GREEN)
- **Issue:** the arm invoked `--stage no-keyring-arm` while the dispatch registers `nokeyring-arm` (usage error, caught by the bounded-attempt diagnostics), the script file is not executable so `timeout ... "$SELF"` failed with permission denied, and the summary ran inside the arm polluting the arm's output contract
- **Fix:** invoke `timeout "$TIMEOUT_ARM" bash "$SELF" --stage nokeyring-arm --internal-arm --binary "$BINARY"`
- **Files modified:** scripts/vm-e2e.sh
- **Verification:** the live stripped-PATH arm surfaces the exact binary-missing sentence; all three contract tests pass (guarded arm included)
- **Committed in:** 5de5b49 (task 2 commit)

---

**Total deviations:** 3 auto-fixed (3 Rule 1 bugs — all bash-runtime defects in the new script, caught by the script's own checks/tests)
**Impact on plan:** none on scope — all three fixes were required for the script's own declared contract (bounded, named-check, exit-code correctness).

## Issues Encountered

- The live-arm test initially asserted the stage's output contains the sentinel sentence while the PASS detail did not echo the matched sentence — the plan's Test 1 says the attempt "surfaces the exact sentence ... and the script's check PASSES on matching it", so the PASS detail now carries the matched sentence verbatim (also useful for a human reading the VM log).
- node's default test reporter is spec-style; the TDD RED evidence required TAP — rerun with `--test-reporter=tap --test-reporter-destination=stdout` per the evidence contract and captured that real run.

## User Setup Required

None - no external service configuration required. (The VM field execution itself is the user's manual tier act, recorded via 07-09.)

## Next Phase Readiness

- The scenario script is ready for the QA-02 field proof: on a clean Ubuntu GNOME VM, build-or-fetch the binary off the VM, run `bash scripts/vm-e2e.sh --binary <path>`; the one documented execution rides 07-09's human-check + the experiments ledger.
- 07-08 (docs) can reference the script as the reproducible scenario entry point; the header already documents usage, deps, and exit codes.

## Self-Check: PASSED

All created files exist on disk and all five commits (4 task + 1 docs) are ancestors of HEAD (verified 2026-11-22).

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*
