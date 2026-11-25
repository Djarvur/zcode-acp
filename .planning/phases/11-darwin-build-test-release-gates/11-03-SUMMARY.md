---
phase: 11-darwin-build-test-release-gates
plan: "03"
subsystem: infra
tags: [bash, mise, shell-portability, node, tool-versions, github-actions, ci]
requires:
  - phase: 11-02
    provides: Portable detached-process fixtures and the static portability test seam
provides:
  - Portable find-plus-grep documentation scans with unchanged diagnostics and exit behavior
  - A PATH-level Bash 5 requirement plus named docs and secret-scan mise tasks
  - A bounded gate for four mise-managed tools and the PATH Bash line
  - Homebrew Bash bootstrap on every macOS CI and release job
affects: [11-04, 11-05, MBLD-02]
actuals:
  tokens: 10256
  tasks: 2
  commits: 3
  plan_head_before: ae80f8bd615f3214745b719d9fb3aba9438dc4c8
  plan_head_after: 3017d64e9f1c7bc816183dec3167d3205e695c3a
tech-stack:
  added:
    - Homebrew Bash 5 bootstrap in macOS CI
  patterns:
    - Four mise-managed tool pins plus one PATH-level Bash line requirement
    - mise-only probes for registry tools and a direct PATH probe for non-registry Bash
    - Explicit test-file globs across the supported Node version line
    - Static workflow contracts for bootstrap order and platform-specific Bash setup
key-files:
  created:
    - scripts/check-mise-tool-versions.mjs
    - experiments/tool-versions.test.mjs
    - .planning/phases/11-darwin-build-test-release-gates/11-03-SUMMARY.md
  modified:
    - mise.toml
    - experiments/check-docs.sh
    - experiments/portability.test.mjs
    - experiments/release.test.mjs
    - experiments/scrub-capture.test.mjs
    - experiments/vm-e2e-script.test.mjs
    - .github/workflows/pr-sanity.yml
    - .github/workflows/release.yml
    - .planning/WINDOWS.md
key-decisions:
  - Bash is not a mise tool-registry entry; the major line is declared in [env] and probed directly from PATH.
  - macOS CI installs Homebrew Bash and prepends its bin directory before mise install and the version gate.
  - Go, golangci-lint, GoReleaser, and Node pins remain in [tools] and probe only through mise exec.
  - Preserve the docs gate's diagnostics and semantics while changing only its four GNU-only file selectors.
  - Treat exact three-component pins as exact and one/two-component pins as version lines.
  - Use an explicit experiment test-file glob because the pinned Node line rejects a directory argument.
patterns-established:
  - Shell portability gate: portable find selection plus grep, statically locked against GNU include/exclude options.
  - Tool drift gate: parse mise-managed pins plus the PATH Bash line, probe each through its owning resolver, and accumulate failure.
  - Platform bootstrap gate: macOS workflows install and prepend Homebrew Bash before reading the PATH requirement.
requirements-completed: [MBLD-02]
coverage:
  - id: D1
    description: Portable documentation scans preserve diagnostics, phase-gate behavior, and exit status
    requirement: MBLD-02
    verification:
      - kind: unit
        ref: node --test experiments/portability.test.mjs
        status: pass
      - kind: other
        ref: mise run check-docs
        status: pass
    human_judgment: false
  - id: D2
    description: PATH Bash line and named docs/secret tasks are wired into the canonical CI graph
    requirement: MBLD-02
    verification:
      - kind: unit
        ref: experiments/portability.test.mjs#mise declares a PATH Bash line and exposes both shell gates
        status: pass
      - kind: other
        ref: mise run check-docs && mise run secret-scan
        status: pass
    human_judgment: false
  - id: D3
    description: Offline comparators reject wrong exact versions, wrong version lines, malformed output, and missing versions
    requirement: MBLD-02
    verification:
      - kind: unit
        ref: node --test experiments/tool-versions.test.mjs
        status: pass
      - kind: unit
        ref: experiments/portability.test.mjs#tool-version gate exports comparators and fails non-zero on drift
        status: pass
    human_judgment: false
  - id: D4
    description: Runnable gate checks four mise-resolved tools and the PATH Bash line, and exits non-zero on drift
    requirement: MBLD-02
    verification:
      - kind: other
        ref: mise exec -- node scripts/check-mise-tool-versions.mjs
        status: pass
      - kind: other
        ref: mise run ci
        status: pass
    human_judgment: false
duration: 6min plus trusted runtime correction
completed: 2026-11-25
status: complete
---

# Phase 11 Plan 03: Portable Shell Gates and Tool-Version Enforcement Summary

**Portable docs scans run through PATH Bash 5.3 after real mise execution exposed that Bash has no tool plugin; four tool pins remain mise-resolved, and macOS CI installs Homebrew Bash before every version check.**

## Performance

- **Initial duration:** 6 minutes
- **Started:** 2026-11-25T13:26:50Z
- **Initial completion:** 2026-11-25T13:32:38Z
- **Runtime correction completed:** 2026-11-25T17:58:32Z
- **Tasks:** 2/2
- **Files modified:** 9 implementation, test, and workflow files, plus the summary and broken-windows ledger

## Accomplishments

- Replaced all four GNU-only docs/security scan selectors with portable `find` plus `grep` pipelines while leaving diagnostics, PENDING/phase-gate behavior, hit limits, path scanning, and accumulated exit status unchanged.
- Added `ZCODE_ACP_REQUIRED_BASH = "5"` under `[env]`, exposed `secret-scan` and `check-docs` as named tasks, and appended both after `test-experiments` in `tasks.ci`.
- Kept Go, golangci-lint, GoReleaser, and Node in `[tools]` and probing through bounded `mise exec --` calls.
- Probed Bash directly from PATH because mise has no Bash tool plugin; the gate accepts the 5.x line and rejects other majors.
- Added Homebrew Bash installation and PATH prepending to both macOS PR jobs and both macOS release jobs, before `mise install`.
- Kept native `go test -race -count=1 ./...` unchanged, added no global CGO setting, and kept `scripts/vm-e2e.sh` outside the default task graph.
- Switched the experiment task to `node --test experiments/*.test.mjs` because the mise-pinned Node 26.2.0 rejects a directory argument.

## Verification Evidence

Trusted runtime tools:

```text
mise 2026.11.13 macos-arm64
go 1.27.4 darwin/arm64 via mise
golangci-lint 2.15.0 via mise
goreleaser 2.20.2 via mise
node 26.2.0 via mise
GNU bash 5.3.20 from PATH
```

| Command | Result |
|---------|--------|
| `mise install` | PASS — four tools installed; no Bash registry entry |
| `mise exec -- node scripts/check-mise-tool-versions.mjs` | PASS — four mise probes plus PATH Bash line |
| `mise run ci` | PASS — 123.03 s; lint 0 issues; race suite green; shell gates green |
| `mise run check-docs && mise run secret-scan` | PASS |
| `mise run test-experiments` | PASS — 47 pass, 2 guarded skips, 0 fail |
| `/opt/homebrew/bin/node --test experiments/` | PASS — 47 pass, 2 guarded skips, 0 fail |
| `node --test experiments/tool-versions.test.mjs` | PASS — 5/5 |
| `node --test experiments/portability.test.mjs` | PASS — 14/14 |
| `node --test experiments/release.test.mjs` | PASS — 4 pass, 1 guarded skip |
| TOML parse and Node syntax checks | PASS |
| `git diff --check` | PASS |

The initial blocked checks were executed after explicit user trust was provided. Their `.planning/WINDOWS.md` entries 22–24 are now marked fixed. No real ZCode desktop, Keychain, credential, paid API, or product service was accessed.

## Task Commits

1. **Task 1: Run both shell gates with portable docs scans** — `aa9930c` (`feat(11-03)`)
   - `experiments/check-docs.sh`
   - `mise.toml`
   - `experiments/portability.test.mjs`
2. **Task 2: Enforce the tool and shell version requirements** — `3630083` (`feat(11-03)`)
   - `scripts/check-mise-tool-versions.mjs`
   - `experiments/tool-versions.test.mjs`
   - `experiments/portability.test.mjs`
3. **Trusted-runtime correction** — `3017d64` (`fix(11-03)`)
   - PATH Bash requirement, direct Bash probe, macOS Homebrew bootstrap
   - explicit Node experiment file glob
   - updated static workflow and comparator contracts

**Plan metadata:** committed separately with this summary and the broken-windows ledger.

## Files Created/Modified

- `mise.toml` — defines four tool pins, the PATH Bash line, task glob, and CI dependency order.
- `experiments/check-docs.sh` — replaces four GNU-only selectors with portable find-plus-grep scans.
- `experiments/portability.test.mjs` — locks shell semantics, version-gate shape, macOS Bash bootstrap, Linux separation, and task graph.
- `experiments/release.test.mjs` — locks Homebrew Bash bootstrap in both release jobs.
- `scripts/check-mise-tool-versions.mjs` — parses four `[tools]` pins plus `[env].ZCODE_ACP_REQUIRED_BASH`, using mise for registry tools and PATH for Bash.
- `experiments/tool-versions.test.mjs` — offline tests for extraction and exact/line matching.
- `.github/workflows/pr-sanity.yml` — installs and prepends Homebrew Bash in both macOS jobs.
- `.github/workflows/release.yml` — installs and prepends Homebrew Bash in snapshot and publish jobs.
- `.planning/WINDOWS.md` — closes the three formerly blocked runtime gates.

## Decisions Made

- Keep the four real mise tool pins in `[tools]`; represent Bash as `[env].ZCODE_ACP_REQUIRED_BASH = "5"` because the mise registry has no Bash plugin.
- Probe Bash with `bash --version` directly from PATH; probing it through `mise exec` is invalid.
- Install Homebrew Bash only on macOS jobs and prepend `$(brew --prefix)/bin` via `GITHUB_PATH`; Linux runners already provide Bash 5.
- Use an explicit test-file glob so the task works on the whole supported Node line, including 26.2.0.
- Compare three-component pins exactly and compare one/two-component pins by their required leading version line.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed a self-scan collision from the new portability test**
- **Found during:** Task 1 direct docs gate
- **Issue:** The static test embedded the endpoint literal that `check-docs.sh` intentionally forbids in experiment scripts.
- **Fix:** Constructed the asserted pattern at runtime while preserving the scan-shape assertion.
- **Files modified:** `experiments/portability.test.mjs`
- **Verification:** Portability tests and both shell gates pass.
- **Committed in:** `aa9930c`

**2. [Rule 1 - Bug] Rejected four-component tokens in version extraction**
- **Found during:** Task 2 comparator tests
- **Issue:** The extractor could accept the final three components of a malformed four-component token.
- **Fix:** Excluded a preceding dot from the token boundary and retained trailing dotted-component rejection.
- **Files modified:** `scripts/check-mise-tool-versions.mjs`
- **Verification:** All comparator and portability tests pass.
- **Committed in:** `3630083`

**3. [Rule 1 - Bug] Replaced the unsupported mise Bash tool pin**
- **Found during:** Real `mise install` after explicit trust
- **Issue:** mise answered `bash not found in mise tool registry`; the original design could never install or probe Bash.
- **Fix:** Removed `bash` from `[tools]`, added the `[env]` major-line requirement, probed Bash directly from PATH, and added Homebrew Bash bootstrap to every macOS job.
- **Files modified:** `mise.toml`, `scripts/check-mise-tool-versions.mjs`, both workflows, and static tests.
- **Verification:** `mise install`, the five-requirement gate, mac workflow contracts, and `mise run ci` pass.
- **Committed in:** `3017d64`

**4. [Rule 1 - Bug] Fixed the Node experiment task for the pinned line**
- **Found during:** First trusted `mise run ci`
- **Issue:** Node 26.2.0 treated `experiments/` as a module path and failed before running tests.
- **Fix:** Use `node --test experiments/*.test.mjs` and lock the glob in the portability contract.
- **Files modified:** `mise.toml`, `experiments/portability.test.mjs`, test header comments.
- **Verification:** The mise task and full suite pass with 47 tests and 2 guarded skips.
- **Committed in:** `3017d64`

---

**Total deviations:** 4 auto-fixed Rule 1 bugs.
**Impact on plan:** The first two corrected the original implementation; the last two were required for the approved real mise runtime gates. No new dependency, ACP method, client feature, Linux VM behavior, or live-service access was added.

## Issues Encountered

The first trusted `mise install` invalidated the original Bash-in-`[tools]` design, and the first `mise run ci` exposed the Node directory-argument incompatibility. Both were corrected and re-run from scratch. The final full CI task passed in 123.03 seconds.

## Authentication Gates

None.

## User Setup Required

None remaining. Repository trust was explicitly provided for the runtime correction. macOS CI jobs self-install Homebrew Bash; Linux jobs use the runner Bash 5 line.

## Known Stubs

None.

## Next Phase Readiness

- Plan 04 can consume the four-pin mise source, PATH Bash line, version gate, task glob, and canonical CI dependency order.
- Native race testing remains CGO-enabled and separate from release cross-build settings.
- The Linux-only VM script remains outside the portable default gate.
- All three previously blocked runtime gates are executed and their `.planning/WINDOWS.md` entries are fixed.
- No real desktop, Keychain, credential, paid API, or product service was accessed.

## Self-Check: PASSED

- All nine implementation/test/workflow files, this summary, and `.planning/WINDOWS.md` exist on disk.
- Commits `aa9930c`, `3630083`, and `3017d64` are ancestors of HEAD.
- `mise install`, the five-requirement version gate, `mise run ci`, both named shell tasks, both Node suite forms, and `git diff --check` pass.
- `.planning/WINDOWS.md` entries 22–24 are marked fixed.
- `.planning/STATE.md` and `.planning/ROADMAP.md` are absent from the correction commit and working tree.
- The pre-existing untracked `.planning/milestone.lock` was left untouched.
