---
phase: 10-mount-first-appimage-backend
plan: 03
subsystem: infra
tags: [appimage, discovery, version-ranking, diagnostics, purity, go, tdd]

# Dependency graph
requires:
  - phase: 10-mount-first-appimage-backend
    provides: 10-01 Install.AppImage/Version/RequiresMount, the mount contract, PrepareMount's bundled-builtin fill
  - phase: 03-zcode-back-leg
    provides: Probers/Discover, bundleInstall, compareVersions, mountDecoy, the probe-row diagnostics contract
provides:
  - One pure-read AppImage-layout stage that evaluates every packed/unpacked/XDG-seeded candidate and ranks by version
  - Packed Install{AppImage, Version, Layout appimage} with empty Root/Binary/Entrypoint for the mount-first path
  - appimage-rank and appimage-candidate diagnostics with layout, version, resolved path, display, and failing step
  - XDG AppImage-family seeding without changing package-tree precedence
  - Packed builtin-bundled deferral naming the mount-relative path
  - Deleted Probers.ExtractAppImage/defaultExtractAppImage/dirPerm — no extraction seam remains
affects: [10-04 override completeness, 10-05 real tier, 10-06 docs]

# Actuals (#2632) — same chars/4 scale as the plan's estimate.
actuals:
  tokens: 21167
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Candidate evaluation is a pure per-path function returning a miss string; ranking sorts misses but only non-miss candidates can win"
    - "One ranked stage replaces order-coupled probes: packed files, unpacked trees, and XDG seeds share the same version/tie-break comparator"
    - "Resolved-path dedupe keeps one candidate per backend, preferring a hit, the symlink rank, then the lexicographically smaller display"
    - "Diagnostics are rows with a stable fixed order; the variable appimage-candidate block stays contiguous between appimage-rank and dpkg"

key-files:
  created:
    - .planning/tdd/10-03-task1-red.json
    - .planning/tdd/10-03-task2-red.json
  modified:
    - internal/supervisor/discover.go
    - internal/supervisor/discover_test.go

key-decisions:
  - "D-12 is binding: an unversioned packed file or a tree under a non-zcode-<ver> parent is a version-unknown miss, never a last-resort winner (the research's keep-last suggestion is not adopted)"
  - "Misses participate in ranking so candidate rows are in final rank order, but appImageWinner returns only the first non-miss candidate; a bad newer candidate never blocks a usable older one"
  - "An earlier-probe winner keeps the documented first-match order: the stage records 'not consulted; an earlier probe resolved the install' and appends no candidate rows"
  - "XDG AppImage-family targets seed the ranked stage (D-13); a package-tree entry still wins at step 2 (D-10)"
  - "The packed builtin-bundled row defers with the literal 'deferred until mount' and the mount-relative bundled path; PrepareMount fills BuiltinFile only when discovery left it empty (10-01)"

patterns-established:
  - "Per-candidate miss rows: one appimage-candidate row per evaluated path, in final rank order, naming the display, version, and failing step"
  - "Synthetic real-machine fixture: packed 4.0.0 + stale unpacked 3.18.2 + dangling ZCode.AppImage selects 4.0.0 and shows the dangling link as a miss"
  - "Pure-read proof: a directory snapshot (mode+size per path) is identical before and after Discover, and no .mount_* entry appears"

requirements-completed: [DISC-01, DISC-02, DISC-03, DISC-04, MNT-06]

coverage:
  - id: D1
    description: "Version ranking selects the newest usable backend across layouts: packed 4.0.0 beats a stale unpacked 3.18.2 tree, and an unpacked-only fixture resolves with its zcode-<ver> version"
    requirement: "DISC-01"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverRankPackedOverStaleUnpacked"
        status: pass
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverUnpackedOnly"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every candidate is evaluated independently: a dangling symlink and an invalid tree are miss rows and the scan continues to the packed winner"
    requirement: "DISC-02"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverPerCandidateMisses"
        status: pass
    human_judgment: false
  - id: D3
    description: "Diagnostics name the selected layout, version, and resolved path, one ordered candidate row per candidate, and the failing step for every miss; an unversioned candidate is a version-unknown miss"
    requirement: "DISC-03"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverDiagnostics"
        status: pass
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverVersionUnknown"
        status: pass
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverProbeOrder"
        status: pass
    human_judgment: false
  - id: D4
    description: "Equal versions break deterministically: unpacked, then packed symlink, then packed regular file, then lexicographic display; a symlink and its resolved target are one candidate"
    requirement: "DISC-04"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverTieBreak"
        status: pass
    human_judgment: false
  - id: D5
    description: "Discovery is a pure read: no mount, no directory creation, no extraction copy, and the extraction seam is gone; a packed winner is a mount-first Install"
    requirement: "MNT-06"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverPureRead"
        status: pass
      - kind: automated
        ref: "grep -c 'ExtractAppImage' internal/supervisor/discover.go == 0; grep -R 'ExtractAppImage' internal/supervisor has no match"
        status: pass
    human_judgment: false
  - id: D6
    description: "XDG AppImage-family targets join the ranking while package trees keep step-2 precedence; the packed builtin row defers to the mount and a runtime-cache hit keeps its literal"
    requirement: "DISC-01, DISC-03"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverXDGAppImageSeed"
        status: pass
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestDiscoverBuiltinDeferredForPacked"
        status: pass
    human_judgment: false

duration: 10 min
completed: 2026-11-24
plan_head_before: 4d503a2d2ff6f89d8f8019c061d1ddd9da9afddc
plan_head_after: 3ff635adec71e68a280e3b9a90998ef16238970d
status: complete
---

# Phase 10 Plan 03: Version-Ranked AppImage Discovery Summary

**One pure-read AppImage stage now selects the newest usable backend by version — a packed 4.0.0 beats a stale unpacked 3.18.2, the real machine's dangling `ZCode.AppImage` is a miss row instead of a probe abort, and every candidate is explained by an ordered diagnostic.**

## Performance

- **Duration:** 10 min
- **Started:** 2026-11-24T02:34Z (local 02:34 MSK)
- **Completed:** 2026-11-24T02:44Z (local 02:44 MSK)
- **Tasks:** 2
- **Files modified:** 2 production/test files (+2 TDD evidence records)

## Accomplishments

- `internal/supervisor/discover.go` now has a single AppImage-layout stage (`probeAppImageStage`) that collects packed `ZCode*.AppImage` files, unpacked `zcode-<ver>/squashfs-root` trees, and XDG AppImage-family seeds; evaluates each path independently; and ranks the survivors by version, then the D-08 tie-break.
- The two extraction-era probes (`appimage-glob`, `unpacked-glob`) and their order coupling are gone. `packed[0]` is never read; an unpacked tree no longer suppresses packed candidates; a dangling symlink `EvalSymlinks` failure is a miss, never a whole-probe abort.
- A packed winner is `Install{AppImage: resolved, Version: <ver>, Layout: appimage}` with empty `Root/Binary/Entrypoint` and `RequiresMount() == true`, feeding the 10-01 mount path. An unpacked winner keeps the invariant `bundleInstall` surface plus its parsed `Version`.
- Diagnostics are complete and ordered: `appimage-rank` reports `selected: <layout> version <ver> resolved <path>` (plus the symlink display when it differs) or `no AppImage-layout candidate`; one `appimage-candidate` row per evaluated candidate reports hit/miss with display, version, and the failing step. The variable block stays contiguous between `appimage-rank` and `dpkg`, and `probeCount` is the new fixed-row count (eight).
- XDG seeding closes the entry-target gap (D-13): a packed AppImage or an unpacked `squashfs-root` target joins the ranking, while a package-tree target still wins at step 2 with unchanged precedence (D-10). An earlier-probe winner leaves the stage "not consulted" with no candidate rows.
- The packed `builtin-bundled` arm now records `deferred until mount; <mount>/resources/config/provider/zcode-builtin.json is read after PrepareMount` and returns no file; the runtime-cache-hit arm keeps `not consulted; the runtime cache resolved the builtin file`. `Probers.ExtractAppImage`, `defaultExtractAppImage`, `dirPerm`, and every extraction branch are deleted.
- Discovery remains a pure read: no mount, no extraction, no directory creation. A directory snapshot (mode + size per path) is identical before and after `Discover`, and no `.mount_*` entry appears.

## Task Commits

Each task was committed atomically (TDD: RED → GREEN → REFACTOR):

1. **Task 1 RED — version-ranked selection tests** — `1bfe86d` (test)
2. **Task 1 GREEN — ranked per-candidate AppImage stage** — `a165bb6` (feat)
3. **Task 2 RED — XDG seeding and packed builtin deferral** — `2cc9a89` (test)
4. **Task 2 GREEN — XDG seeds, deferred builtin, extraction seam deleted** — `df8c997` (feat)
5. **REFACTOR — lint-driven test matrix splits** — `3ff635a` (refactor)

**Plan metadata:** the `docs(10-03)` commit carrying this SUMMARY, STATE.md, ROADMAP.md, and REQUIREMENTS.md.

## Files Created/Modified

- `internal/supervisor/discover.go` — `appCandidate` and the ranked stage; per-path evaluation (`evaluatePackedCandidate`, `evaluateUnpackedCandidate`, `evaluateAppCandidate`); resolved-path dedupe and preference; version/rank/display sorting; `appImageWinner`; `selection`/`result` diagnostics; XDG seeding through `resolveExecTarget`; the packed builtin deferral; tightened `dirVersion`/`appImageVersion`; removed extraction field/function/`dirPerm`.
- `internal/supervisor/discover_test.go` — new ranking, per-candidate miss, unpacked-only, tie-break, version-unknown, pure-read, diagnostics, XDG-seed, packed-builtin, and probe-order tests; extraction fixture removed; shared fixture constants and named subtest helpers for the strict lint gates.
- `.planning/tdd/10-03-task1-red.json`, `.planning/tdd/10-03-task2-red.json` — machine-verified RED evidence (`RED_EVIDENCE_OK`).

## Decisions Made

- **D-12 over the research's keep-last suggestion.** An unversioned packed file never wins in discovery; it is a miss with an explicit version-unknown diagnostic, and `--app-server` remains the route for a renamed file.
- **Misses are ranked but never selected.** Sorting includes misses so the candidate rows reflect the final rank order, while `appImageWinner` skips them; a newer unusable file does not hide an older usable backend, and the miss remains visible.
- **First-match order is preserved.** A prior probe's winner makes the stage record "not consulted" and append no candidate rows, so the documented precedence is unchanged and the diagnostics do not claim a misleading selection.
- **XDG precedence split by target kind.** Package trees keep step 2 (D-10); AppImage-family targets join the version ranking (D-13), so a stale entry can no longer pin an older backend.
- **Symlink dedupe keeps the user-facing display.** One backend keyed by resolved path prefers a hit, then the symlink rank, then the smaller display; the winner's `AppImage` is always the resolved target.
- **The packed builtin is deferred, not guessed.** The row names the exact mount-relative path; `PrepareMount` fills `BuiltinFile` after validation only when discovery left it empty.

## Deviations from Plan

None - plan executed exactly as written. The only additional commit is the scheduled lint-driven REFACTOR: the tie-break and XDG-seed subtests were split into named helpers, repeated test literals were replaced with the package's shared fixture constants, and candidate slices are indexed to avoid 200-byte copies. No behavior changed.

## TDD Gate Compliance

| Phase | Commit | Evidence |
|-------|--------|----------|
| RED (Task 1) | `1bfe86d` — `test(10-03)` | `TestDiscoverRankPackedOverStaleUnpacked` selected the stale unpacked 3.18.2 tree; the per-candidate, tie-break, version-unknown, and pure-read tests also failed on planned assertions (`RED_EVIDENCE_OK`, `.planning/tdd/10-03-task1-red.json`) |
| GREEN (Task 1) | `a165bb6` — `feat(10-03)` | The ranked stage selects packed 4.0.0, tolerates the dangling symlink, and all six target tests pass |
| RED (Task 2) | `2cc9a89` — `test(10-03)` | The packed XDG target produced `ErrNoInstall`, the unpacked seed won without a Version, and the builtin row carried the old literal (`RED_EVIDENCE_OK`, `.planning/tdd/10-03-task2-red.json`) |
| GREEN (Task 2) | `df8c997` — `feat(10-03)` | XDG seeds join the ranking, package precedence holds, both packed builtin arms pass, and the extraction seam is gone |
| REFACTOR | `3ff635a` — `refactor(10-03)` | Lint-driven test splits/constants only; the full suite and `golangci-lint` stay green |

No gate violations: every `feat(10-03)` descends from a `test(10-03)` commit, and each RED target failed on a planned assertion (layout/version selection, XDG seeding, builtin literal), not on build or fixture faults.

## Verification

| Command | Result |
|---------|--------|
| `go test -race -count=1 ./internal/supervisor/ -run 'TestDiscoverRank\|TestDiscoverPerCandidate\|TestDiscoverTieBreak\|TestDiscoverUnpackedOnly\|TestDiscoverVersionUnknown\|TestDiscoverPureRead' -v` | PASS |
| `go test -race -count=1 ./internal/supervisor/ -run 'TestDiscoverDiagnostics\|TestDiscoverXDG\|TestDiscoverBuiltinDeferred\|TestDiscoverProbeOrder\|TestDiscoverNoInstall\|TestDiscoverRejectsVolatileMounts' -v` | PASS |
| `go test -race -count=1 ./internal/supervisor/ -run TestDiscover -v` | PASS |
| `go test -race -count=1 ./internal/supervisor/ -run TestMount -v` | PASS |
| `go test -race -count=1 ./...` | PASS (full suite) |
| `mise run lint` (golangci-lint 2.15.0) | 0 issues |
| `mise run tidy-diff` | clean (one `require` line, acp-go-sdk v0.20.0) |
| `grep -R 'ExtractAppImage' internal/supervisor` | no match |

Acceptance greps (all pass): `appCandidate` >= 1, `func appImageVersion` == 1, `packed[0]` == 0, `ExtractAppImage` == 0, `appimage-glob` == 0, `appimage-rank` >= 1, `appimage-candidate` >= 1, `deferred until mount` >= 1; `DefaultProbers` has no extractor field and no test stubs one.

## Issues Encountered

- The strict lint gates split the tie-break and XDG-seed matrices (funlen, gocognit) and required shared constants for the repeated fixture literals (goconst) plus indexed candidate loops (gocritic rangeValCopy). All resolved in the REFACTOR commit without behavior change.
- `requirements.mark-complete` may report MNT-06 already complete (10-02 recorded it); the orchestrator owns the final requirement flips if the ready check blocks sibling plans.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 10-04 can rely on a packed `Install` from discovery (empty Root/Binary/Entrypoint, `RequiresMount` true) and on the override mapper already routing `.AppImage` values through the mount.
- 10-05's guarded real tier will see 4.0.0 selected on the real-machine fixture shape; the dangling symlink is a miss row, not a probe abort.
- 10-06 documents the version ranking, the `appimage-rank`/`appimage-candidate` rows, the XDG seed, and the deletion of the extraction route.

## Self-Check: PASSED

All created/modified files exist; all five commits are ancestors of HEAD; no commit in the plan deleted tracked files.
