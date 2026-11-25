---
phase: 11-darwin-build-test-release-gates
plan: "04"
subsystem: infra
tags: [ci, macos, darwin, cross-build, mise, github-actions, portability]
requires:
  - phase: 11-03
    provides: Managed Bash tasks, the canonical CI graph, and the five-tool mise version gate
provides:
  - Native macOS arm64 and amd64 build/test jobs with fail-closed architecture preflights
  - A separate two-target CGO-free Darwin cross-build job
  - Mise-managed secret and documentation shell gates
  - Offline static contracts for the complete PR sanity matrix
affects: [11-05, MBLD-03, MTEST-01]
actuals:
  tokens: 3725
  tasks: 2
  commits: 2
  plan_head_before: 356ba7eefc60cf5e629e5486fc4f2ef46d40ee48
  plan_head_after: 37ed03b9f5f06a25b5207fa2f2df4200eda3114d
tech-stack:
  added: []
  patterns:
    - Native architecture preflights compare both mise-resolved Go GOARCH and uname -m before gates
    - Build and test jobs run the canonical mise task graph without duplicating tool pins
    - Compile-only Darwin targets remain separate from CGO-enabled native race tests
    - Workflow contracts are enforced by offline Node text assertions
key-files:
  created:
    - .planning/phases/11-darwin-build-test-release-gates/11-04-SUMMARY.md
  modified:
    - .github/workflows/pr-sanity.yml
    - experiments/portability.test.mjs
    - .planning/WINDOWS.md
key-decisions:
  - Keep the Linux sanity job name and runner byte-for-byte unchanged for branch protection.
  - Use separate native macOS jobs rather than an architecture matrix so each required context is explicit.
  - Run the five-tool version gate without tool arguments so every native and cross-build job logs all resolved versions.
  - Keep CGO_ENABLED=0 only in the two literal Darwin cross-build commands; native jobs retain CGO-enabled race tests.
  - Route both shell gates through their existing named mise tasks so managed Bash is selected without a second version source.
patterns-established:
  - Platform gate: checkout, mise bootstrap/install, architecture preflight, version gate, canonical CI, tidy-diff.
  - Cross-build gate: mise-resolved Go with explicit CGO_ENABLED=0, GOOS=darwin, and one GOARCH per step.
  - Static workflow lock: job-scoped text extraction verifies names, runners, commands, ordering, and forbidden native CGO overrides.
requirements-completed: [MBLD-03, MTEST-01]
coverage:
  - id: D1
    description: Native macOS arm64 runs architecture and tool-version preflights before canonical CI and tidy gates
    requirement: MBLD-03
    verification:
      - kind: unit
        ref: experiments/portability.test.mjs#macOS arm64 runs architecture and version preflights before canonical gates
        status: pass
    human_judgment: false
  - id: D2
    description: Intel macOS and both literal CGO-free Darwin cross-builds are encoded as separate fail-closed jobs
    requirement: MBLD-03
    verification:
      - kind: unit
        ref: experiments/portability.test.mjs#macOS amd64 mirrors the native preflights and canonical gates
        status: pass
      - kind: unit
        ref: experiments/portability.test.mjs#Darwin cross-build checks tool pins before both literal CGO-free targets
        status: pass
    human_judgment: false
  - id: D3
    description: Secret and docs gates use managed Bash tasks while required contexts and read-only permissions remain stable
    requirement: MBLD-03
    verification:
      - kind: unit
        ref: experiments/portability.test.mjs#secret-scan runs both shell gates through mise-managed Bash tasks
        status: pass
      - kind: unit
        ref: experiments/portability.test.mjs#PR sanity preserves required security contexts and read-only concurrency
        status: pass
    human_judgment: false
  - id: D4
    description: A hosted GitHub run observes green Linux, both macOS architectures, cross-build, govulncheck, and secret-scan jobs with recorded versions
    requirement: MTEST-01
    verification:
      - kind: manual_procedural
        ref: pr-sanity.yml hosted run on the phase head SHA
        status: unknown
    human_judgment: true
    rationale: The workflow is statically locked, but no branch was pushed and no hosted runner output was available in this execution.
duration: 6min
completed: 2026-11-25
status: complete
---

# Phase 11 Plan 04: Native macOS and Darwin CI Gates Summary

**PR sanity now preserves the protected Linux check while adding native macOS arm64/amd64 gates, both literal CGO-free Darwin cross-builds, mise-visible tool versions, and managed Bash shell gates.**

## Performance

- **Duration:** 6 minutes
- **Started:** 2026-11-25T13:37:05Z
- **Completed:** 2026-11-25T13:43:09Z
- **Tasks:** 2/2
- **Files modified:** 2 implementation/test files, plus this summary and the broken-windows ledger

## Accomplishments

- Added `macos-arm64` on `macos-latest` with a 30-minute timeout, exact required name, checkout/mise bootstrap, architecture preflight, five-tool version gate, `mise run ci`, and `mise run tidy-diff`.
- Added `macos-amd64` on `macos-15-intel` with the same gate order and fail-closed checks for `GOARCH=amd64` and `uname -m=x86_64`.
- Added `darwin-cross-build` on Ubuntu with the version gate followed by the two exact mise-resolved, CGO-free Darwin build commands.
- Kept native macOS jobs free of `CGO_ENABLED=0`, preserving the canonical CGO-enabled race-test path.
- Added mise bootstrap/install to `secret-scan` and replaced direct system-shell calls with `mise run secret-scan` and `mise run check-docs`.
- Preserved the Linux required check name, `govulncheck`, `secret-scan`, concurrency, read-only token permissions, and `actions/checkout@v4`.
- Extended the offline portability suite with job-scoped workflow assertions for names, runners, architecture comparisons, version gates, command ordering, cross-build targets, managed shell tasks, and security invariants.

## Verification Evidence

Direct tools used:

```text
/opt/homebrew/bin/node  -> v26.13.1
/opt/homebrew/bin/bash  -> GNU bash 5.3.20
/Users/nil/.local/share/mise/installs/go/1.27.4/bin/go -> go1.27.4 darwin/arm64
```

No `mise`, `mise trust`, real ZCode, Keychain, credential, paid-service, or network API command was executed.

| Command | Result |
|---------|--------|
| `/opt/homebrew/bin/node --test experiments/portability.test.mjs` after Task 1 | PASS — 9/9 |
| Task 1 tracer re-verification after commit `ed78cb1` | PASS — 9/9 plus committed-diff whitespace check |
| `/opt/homebrew/bin/node --test experiments/portability.test.mjs` after Task 2 | PASS — 13/13 |
| `/opt/homebrew/bin/node --test experiments/` | PASS — 42 pass, 1 guarded skip, 0 fail |
| `/opt/homebrew/bin/node --check experiments/portability.test.mjs` | PASS |
| `git diff --check HEAD -- .github/workflows/pr-sanity.yml experiments/portability.test.mjs` | PASS after both tasks |
| `git diff --check 356ba7e..37ed03b -- .github/workflows/pr-sanity.yml experiments/portability.test.mjs` | PASS |

### Acceptance Criteria

- **Task 1:** PASS — arm64 uses the canonical mise tasks with CGO enabled; architecture and version gates precede build/test; Linux required name is unchanged; workflow coverage is static/offline.
- **Task 2:** PASS — both native jobs assert architecture; all three build jobs run the version gate first; both exact cross-build commands are present; shell gates use managed tasks; required contexts and read-only rights remain stable.

### Pending Hosted Observation

The plan's human-check was not run: no branch was pushed and no GitHub Actions output was available. A phase verifier must record the head SHA, actual runner architectures, all five resolved tool versions, and green conclusions for Linux, macOS arm64, macOS amd64, `darwin-cross-build`, `govulncheck`, and `secret-scan`. This is open `unrun-verify` entry 25 in `.planning/WINDOWS.md`; it is not represented as passed.

The local mise runtime gate and task graph also remain unexecuted because `mise.toml` is explicitly untrusted. This is recorded as entry 26; hosted CI owns the first runtime observation.

## Task Commits

1. **Task 1: Add the native macOS arm64 build/test path end to end** — `ed78cb1` (`feat(11-04)`)
   - `.github/workflows/pr-sanity.yml`
   - `experiments/portability.test.mjs`
2. **Task 2: Add Intel macOS, both Darwin cross-builds, and managed shell gates** — `37ed03b` (`feat(11-04)`)
   - `.github/workflows/pr-sanity.yml`
   - `experiments/portability.test.mjs`

**Plan metadata:** committed separately with this summary and `.planning/WINDOWS.md`.

## Files Created/Modified

- `.github/workflows/pr-sanity.yml` — adds the native macOS jobs, Darwin cross-build job, version/architecture preflights, and managed shell-gate execution.
- `experiments/portability.test.mjs` — adds offline job/step extraction and static contracts for the full platform matrix.
- `.planning/WINDOWS.md` — records the unrun hosted-CI observation and untrusted local mise runtime checks.
- `.planning/phases/11-darwin-build-test-release-gates/11-04-SUMMARY.md` — records execution results and pending external evidence.

## Decisions Made

- Preserve the existing Linux job rather than converting it to a matrix, avoiding any branch-protection context change.
- Use explicit macOS jobs and runner labels, with both Go and host architecture assertions as the authority on actual execution.
- Invoke the version gate without arguments in every native/cross-build job so all five pins are printed and enforced.
- Keep cross-compilation and native race testing separate; only the two Darwin build commands disable CGO.
- Reuse named mise shell tasks instead of introducing hard-coded Bash paths or duplicate version pins.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected YAML indentation in the new arm64 contract test**
- **Found during:** Task 1 verification
- **Issue:** The test expected `EXPECTED_GOARCH` and `EXPECTED_UNAME_M` at eight spaces, but YAML nests `env` values at ten.
- **Fix:** Corrected the two assertion patterns; no workflow behavior changed.
- **Files modified:** `experiments/portability.test.mjs`
- **Verification:** Portability suite passed 9/9, then 13/13 after Task 2; full Node suite passed.
- **Committed in:** `ed78cb1`

---

**Total deviations:** 1 auto-fixed Rule 1 test bug.
**Impact on plan:** The correction was required for the static contract to match valid YAML. No architectural change, dependency, ACP method, client feature, Linux VM behavior, or live-service access was added.

## Issues Encountered

- The repository mise configuration is untrusted by explicit constraint, so mise runtime commands were not executed. Static workflow assertions and direct Node tests passed; the hosted workflow will provide runtime evidence.
- No hosted GitHub run was available. The required observation remains open in `.planning/WINDOWS.md` for end-of-phase verification.

## Authentication Gates

None.

## User Setup Required

None for implementation. The phase verifier needs a pushed phase branch/PR and a completed `pr-sanity` run to close the hosted observation.

## Known Stubs

None.

## Threat Flags

None beyond the plan's existing CI trust-boundary mitigations. The workflow retains read-only permissions, no product secrets, mise as the only tool-version source, `actions/checkout@v4`, and offline/fake test tiers.

## Next Phase Readiness

- Plan 05 can consume the native macOS jobs, exact Darwin cross-build commands, managed shell tasks, and five-tool version gate.
- The workflow matrix is statically locked and all runnable local checks pass.
- Hosted CI evidence and the first mise runtime observation remain pending; `.planning/STATE.md` and `.planning/ROADMAP.md` were intentionally not updated per the execution request.

## Self-Check: PASSED

- The summary, workflow, portability test, and broken-windows ledger exist on disk.
- Task commits `ed78cb1` and `37ed03b` are ancestors of HEAD.
- The measured task range contains two commits and passes whitespace checks; summary/ledger edits also pass whitespace checks.
- `.planning/STATE.md` and `.planning/ROADMAP.md` retain their pre-execution SHA-256 hashes and are absent from the plan changes.
- The pre-existing untracked `.planning/milestone.lock` was left untouched.
- Hosted CI and mise runtime checks remain explicitly open in `.planning/WINDOWS.md` and are not claimed as passed.
