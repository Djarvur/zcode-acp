---
phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
plan: 04
subsystem: supervisor
tags: [discovery, install-matrix, probers, diagnostics, tdd, supervisor]

# Dependency graph
requires:
  - phase: 01-research
    provides: install-discovery.md (probe order, locate commands, entrypoint invariant, revision formula, decoy rows — EXP-DISCOVERY-01 verified)
  - phase: 02-scaffold
    provides: strict golangci gate (mise run lint/ci/tidy-diff) + repo Go module skeleton
provides:
  - internal/supervisor discovery half: Install (Root/Binary/Entrypoint/BuiltinFile/Layout), Probe{Name,Query,Result}, Probers (Override/XDGQuery/AppDirs/DpkgQuery/RpmQuery/ExtractAppImage/Home/OptDir seams), Discover (ordered, first-match-wins, full diagnostics), ErrNoInstall sentinel, BuiltinRevision (zcode-builtin:<rev>:<sha256(resolved path string)>), DefaultProbers (production seams shell out the matrix commands)
  - Numeric version ordering helpers (3.18.2 > 3.17.1 > 0.0.0-dev) shared by the unpacked-glob and builtin-runtime probes
  - Volatile /tmp/.mount_* decoy guard applied on every probe surface; localhost listeners never probed
affects: [03-07 spawn/supervise (consumes Discover + BuiltinFile + DefaultProbers), 03-09 E2E + phase gate, phase 4/6 revision consumers]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 13240    # 52959 diff chars / 4 over 3cbd949..351fd82
  tasks: 2
  commits: 3       # MEASURED: git rev-list --count 3cbd949..HEAD (2 task commits + 1 concurrent non-plan docs commit f3c35d1)

# Tech tracking
tech-stack:
  added: []  # zero new dependencies — stdlib only (T-3-SC: go.mod still declares only the module line; tidy-diff clean)
  patterns:
    - "Seamed discovery unit: one injectable function seam per OS contact; nil seam = honest miss; DefaultProbers is the production constructor"
    - "All probes run always, rows appended hit-or-miss; winner = first hit in the documented order (probes never merge — adjacency resolved)"
    - "Empty result is a first-class outcome: ErrNoInstall wrapped with one diagnostics line per probe row (the diagnostics ARE the product)"
    - "Record-and-reject decoy discipline: any candidate matching /tmp/.mount_* is named in its probe row and never resolved"
    - "Revision formula sharpened in code: sha256 over the EvalSymlinks+Abs-resolved PATH STRING, never file contents"

key-files:
  created:
    - internal/supervisor/discover.go
    - internal/supervisor/discover_test.go
    - .planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-04-red-discovery.json
  modified: []

key-decisions:
  - "ErrNoInstall exported as the identifiable empty-result sentinel: the plan's truth makes the empty set a first-class outcome and 03-07 needs to distinguish 'nothing found' from probe failures via errors.Is"
  - "All probes run to completion on every Discover call (both attempts stay in the diagnostics; first-match-wins picks the earliest hit) — this is the adjacency resolution the plan's must_haves encode, and it keeps diagnostics complete on the success path too, not just on empty results"
  - "DpkgQuery seam contract = the dpkg -L file list only; the install-state check lives inside the production default (rc-remnant answers as an empty list, which the probe reads as the documented fall-through to opt-path, install-discovery.md :30,:42)"
  - "xdg-entry Exec resolution supports quoted Exec paths and defers packed-AppImage Exec targets to the appimage-glob probe with an honest hint row (extraction is the appimage-glob probe's route, :41)"
  - "defaultOverride (production) is a documented no-op until plan 03-07 wires the wrapper's explicit root / ZCODE_BUILTIN_PROVIDER_CONFIG_FILE configuration surface — the seam itself is implemented and test-injectable (Known Stubs)"
  - "Test fixtures stub AppDirs with t.TempDir() candidate lists rather than exercising the production glob: the whole matrix stays CI-provable with no real machine contact (plan prohibition + key_links contract); production defaultAppDirs is a two-glob pure-fs scan"

requirements-completed: [BRIDGE-03]  # COPIED VERBATIM from 03-04-PLAN.md frontmatter `requirements` — completion gated by ALL declaring plans (BRIDGE-03: 03-04/03-07/03-09)

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Ordered probe execution with first-match-wins and complete diagnostics: every probe appends a Probe{Name,Query,Result} row hit or miss; the earliest spawnable install in the documented order wins; probes never merge"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/discover_test.go#TestDiscoverOrderFirstMatchWins"
        status: pass
    human_judgment: false
  - id: D2
    description: "Empty result is a first-class outcome: Discover errors with ErrNoInstall whose text names all eight documented probes (override, xdg-entry, appimage-glob, unpacked-glob, dpkg, opt-path, rpm, builtin-runtime) in the fixed documented order"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/discover_test.go#TestDiscoverNoInstallListsEveryProbe"
        status: pass
    human_judgment: false
  - id: D3
    description: "Layout fidelity per the matrix: unpacked-appimage squashfs-root, appimage file+symlink resolved through EvalSymlinks with extraction behind the seam, deb opt tree — all yielding the invariant entrypoint resources/glm/zcode.cjs under Root (:38)"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/discover_test.go#TestDiscoverFixtureTree"
        status: pass
    human_judgment: false
  - id: D4
    description: "Volatile decoy rejection: a /tmp/.mount_* candidate surfaced by AppDirs is recorded-and-rejected in the probe row and never returned as the spawn root; localhost listeners are never probed (no seam exists for them)"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/discover_test.go#TestDiscoverRejectsVolatileMounts"
        status: pass
    human_judgment: false
  - id: D5
    description: "BuiltinRevision computes zcode-builtin:<rev>:<sha256(resolved absolute path string)> — hash input is the EvalSymlinks-resolved PATH STRING, never file contents (sharpened formula :82); symlinked active files resolve"
    requirement: BRIDGE-03
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestBuiltinRevisionFormula"
        status: pass
    human_judgment: false
  - id: D6
    description: "Runtime-provider tree: newest version dir wins by numeric component compare (3.18.2 > 3.17.1 > 0.0.0-dev), shared endpoint dir walked, recorded revision surfaced in the diagnostics row, BuiltinFile attached to the winning install"
    requirement: BRIDGE-03
    verification:
      - kind: integration
        ref: "internal/supervisor/discover_test.go#TestDiscoverFindsNewestBuiltin"
        status: pass
    human_judgment: false

# Metrics
duration: 31min
completed: 2026-11-19
status: complete
---

# Phase 3 Plan 04: Seamed Install Discovery (BRIDGE-03 part 1) Summary

**The Phase 1 install-discovery matrix translated faithfully into a stub-able Probers.Discover — ordered first-match-wins probes, every-probe diagnostics on empty results, volatile-mount rejection, and the path-string builtin revision formula — fully green under -race with zero new dependencies**

## Performance

- **Duration:** 31 min
- **Started:** 2026-11-19T17:26:49Z
- **Completed:** 2026-11-19T17:57:29Z
- **Tasks:** 2
- **Files modified:** 3 (2 code/test + 1 RED evidence record)

## Accomplishments
- internal/supervisor/discover.go: the full discovery half — Install/Probe/Probers/Discover/ErrNoInstall/BuiltinRevision/DefaultProbers, with the documented probe order (override → xdg-entry → appimage-glob → unpacked-glob → dpkg → opt-path → rpm → builtin-runtime) executed first-match-wins while every probe appends its diagnostics row
- Empty-result contract honored verbatim: ErrNoInstall carries one line per probe row (name + result), making the nothing-found diagnostics actionable per install-discovery.md :34; probe row order is fixed and asserted by the negative test
- Decoy discipline test-locked: /tmp/.mount_* candidates are recorded-and-rejected on every probe surface (AppDirs candidates, resolved Exec targets, extraction results, opt roots); localhost listeners have no seam at all — they are never probed (matrix step 5)
- The builtin revision helper implements the sharpened V-SRC formula: sha256 over the EvalSymlinks+Abs-resolved absolute PATH STRING (never file contents), formatted zcode-builtin:<rev>:<hex>
- The builtin-runtime probe walks the runtime-provider tree with numeric version-dir ordering (0.0.0-dev < 3.17.1 < 3.18.2), walks the shared endpoint-* dir, reads the recorded revision into the row, and attaches the newest zcode-builtin.json to the winning install
- All gates green: go test -race -count=1 ./... (whole repo) green; mise run lint 0 issues; mise run tidy-diff clean (go.mod still declares only the module line — T-3-SC honored)
- RED→GREEN machine-checked: evidence/03-04-red-discovery.json classified RED_EVIDENCE_OK (9 tests failing on planned assertions) before the feat commit

## Task Commits

Each task was committed atomically:

1. **Task 1: RED — discovery contract tests (fixtures, diagnostics, order, decoys, revision)** - `942004d` (test)
2. **Task 2: GREEN — Probers implementation: ordered probes, diagnostics, mount guard, revision** - `351fd82` (feat)

_Note: TDD tasks may have multiple commits (test → feat → refactor)_

**Plan metadata:** pending (docs commit lands with this SUMMARY)

## Files Created/Modified
- `internal/supervisor/discover.go` - The seamed discovery unit: ordered probes, diagnostics, mount guard, revision formula, production seam defaults (xdg-mime/dpkg/rpm shell-outs with wrapped errors), numeric version ordering
- `internal/supervisor/discover_test.go` - The six RED targets: fixture trees per layout, all-probe negative diagnostics with order assertion, adjacency first-match-wins, volatile-mount rejection, revision formula (incl. content-hash anti-assertion + symlink resolution), newest-builtin pick
- `.planning/phases/03-zcode-back-leg-codec-fake-app-server-supervisor/evidence/03-04-red-discovery.json` - Persisted RED evidence, RED_EVIDENCE_OK (go-test format, 9 failing tests)

## Decisions Made
- ErrNoInstall exported: the empty result is a first-class outcome (truth 2), and 03-07's spawn path needs to distinguish "nothing found" from probe failures
- All probes run on every Discover call: "diagnostics still record both attempts" (the plan's adjacency test) requires the losing probes to have run; first-match-wins then picks the earliest hit. This also makes diagnostics complete on the success path
- DpkgQuery seam returns the dpkg -L file list only; the production default does the dpkg-query state check first and answers rc-remnant/absent as an empty list — the probe's remnant fall-through (matrix :30, :42) stays testable with canned lists
- Test fixtures inject AppDirs candidates rather than exercising the production glob (plan's key_links contract: "Probers seams stubbed with t.TempDir() fixture trees / canned query outputs — no test touches the real machine state"); defaultAppDirs itself is a pure-fs two-glob scan
- defaultOverride is a no-op until 03-07 wires the wrapper's config surface (the matrix's step 1 override inputs are wrapper flags/env that do not exist yet); the seam is fully implemented and test-injectable

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] RED run 1: appimage fixture crashed before its planned assertion**
- **Found during:** Task 1 (first RED run)
- **Issue:** The appimage subtest wrote the fixture AppImage file before creating its Applications dir (open ...: no such file or directory) — a fixture crash, not a planned assertion failure, which the plan's fails_when forbids
- **Fix:** MkdirAll the applications dir before WriteFile; re-ran RED
- **Files modified:** internal/supervisor/discover_test.go
- **Verification:** RED run 2 — all six targets failing on planned assertions only
- **Committed in:** 942004d (final evidence captured from this state)

**2. [Rule 3 - Blocking] Strict lint findings on the new code (14 then 2 issues)**
- **Found during:** Task 1 (proactive lint before evidence capture) and Task 2 (post-implementation)
- **Issue:** gochecknoglobals (probe-name list as a global), gocognit 33>30 (three subtests inline), funlen 44>40 and 61>60 (probeAppImageGlob/probeXDGEntry), lll (6 long lines), mnd (probeCount 8, SplitN 2, MkdirAll 0o750), nilerr (dpkg/rpm defaults swallowing exec errors), wrapcheck (os.ReadDir error unwrapped), gocritic unnamedResult
- **Fix:** probe names became a function; subtests split into named helper funcs; candidate classification and Exec resolution extracted (classifyAppDirs, resolveExecTarget); named results; named constants; OS errors wrapped (dpkg-query/rpm failures now surface honestly in their probe rows instead of being silently converted to misses)
- **Files modified:** internal/supervisor/discover.go, internal/supervisor/discover_test.go
- **Verification:** mise run lint — 0 issues; package green under -race
- **Committed in:** 351fd82

---

**Total deviations:** 2 auto-fixed (1 Rule 1 fixture bug, 1 Rule 3 lint-blocking)
**Impact on plan:** None on scope — the RED evidence was re-captured from the final committed state (per the evidence contract) and lint fixes were structural only. All acceptance criteria are met.

## Issues Encountered
- A concurrent non-plan commit (f3c35d1, docs(phase-1): re-audit validation) landed on the phase branch between the two task commits — visible in the ledger-measured commit count (3 = 2 task commits + 1 external). No interaction with this plan's files
- The symlink case of the revision formula test initially asserted against the wrong expected path (the first builtin's hash instead of the resolved real file's); caught and corrected during test authoring, before any commit

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 03-07 (spawn/supervise) consumes exactly what this plan shipped: Discover (Install with Root/Binary/Entrypoint/BuiltinFile), BuiltinRevision for capture-header revision strings, and DefaultProbers(home, optDir) for the real machine; the no-install error (ErrNoInstall + rows) is its fail-loud startup diagnostic
- The ExtractAppImage production default runs `--appimage-extract` in the AppImage's dir — exercised only outside CI (Pitfall 10); the Phase-3-07 manual real-desktop test covers it
- The override seam's production wiring (explicit root / ZCODE_BUILTIN_PROVIDER_CONFIG_FILE from wrapper config) is the one known stub — 03-07 owns the config surface it plugs into

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| defaultOverride returns no-op | internal/supervisor/discover.go | 773 | The wrapper's explicit root / ZCODE_BUILTIN_PROVIDER_CONFIG_FILE configuration surface does not exist until plan 03-07 wires it; the Override seam itself is implemented and test-injectable, so the plan's goal (seamed, stub-able discovery) is fully achieved |

## Self-Check: PASSED

_(appended after close-out verification)_

---
*Phase: 03-zcode-back-leg-codec-fake-app-server-supervisor*
*Completed: 2026-11-19*
