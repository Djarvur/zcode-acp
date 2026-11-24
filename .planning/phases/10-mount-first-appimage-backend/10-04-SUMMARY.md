---
phase: 10-mount-first-appimage-backend
plan: 04
subsystem: infra
tags: [appimage, override, mount, pat-client-version, diagnostics, go, tdd]

# Dependency graph
requires:
  - phase: 10-mount-first-appimage-backend
    provides: 10-01 OverrideInstall (directory/binary/packed routing), the mount contract, PrepareMount's mounted Install; 10-03 the version-ranked AppImage stage and Install.Version
  - phase: 06-coding-plan-auth
    provides: the PAT minter's clientVersion seam (patClientVersion/desktopVersion) and the wiring fixtures
provides:
  - Complete --app-server / ZCODE_ACP_APP_SERVER surface: directory, app-server binary, literal/symlinked/dangling packed .AppImage, with flag-over-env precedence unchanged
  - Volatile /tmp/.mount_* rejection for a packed override (probe diagnostics name the path; no discovery fallthrough)
  - Version-first desktopVersion: Install.Version, then the packed AppImage basename, then the Binary/Root/Entrypoint scan
  - Startup selection diagnostics naming layout, version, root, appimage (and binary), plus a mounted-root/version line after PrepareMount
  - Both --app-server help strings naming the three accepted shapes
affects: [10-05 real tier, 10-06 docs]

# Actuals (#2632) — same chars/4 scale as the plan's estimate.
actuals:
  tokens: 8726
  tasks: 2
  commits: 5

tech-stack:
  added: []
  patterns:
    - "One override mapper owns the shape decision: directory branch first (a directory named *.AppImage stays a tree), then EvalSymlinks-resolved packed detection, then the binary shape"
    - "A packed override is always stored absolute (Abs on the resolved target or the literal), so exec.CommandContext can never PATH-resolve it"
    - "The override probe rejects any volatile root OR AppImage path through one volatileInstallPath guard"
    - "desktopVersion is version-first: Install.Version, then the AppImage basename, then the layout path scan (versionFromPath)"
    - "Selection diagnostics are one stable tuple: layout, version, root, appimage (plus binary), repeated after PrepareMount for the mounted root"

key-files:
  created:
    - .planning/tdd/10-04-task1-red.json
    - .planning/tdd/10-04-task2-red.json
  modified:
    - internal/supervisor/discover.go
    - internal/supervisor/discover_test.go
    - internal/supervisor/e2e_test.go
    - cmd/zcode-acp/main.go
    - cmd/zcode-acp/main_test.go

key-decisions:
  - "The 10-01 override mapper already implemented the directory/binary/packed routing and the absolute-path contract; this plan's production change is the volatile-AppImage probe rejection plus the version-first PAT seam, with the remaining routing arms pinned as CONFIRM-GREEN tests"
  - "A volatile AppImage path is rejected at the probe, not only at PrepareMount, so a /tmp/.mount_* override never becomes the discovery winner"
  - "desktopVersion prefers Install.Version because a mounted root carries no version segment; the AppImage basename is the second source so a pre-mount packed install still reports the true version"
  - "Startup diagnostics keep the four D-09 keys and also keep binary, so a binary-override install still names its executable while root/appimage name the resolved layout path"
  - "The mounted log repeats layout/version/root/appimage from the mounted Install after PrepareMount (the version survives the mount; the root is the mount root)"

patterns-established:
  - "Override table test: empty, valid/unvalidated directory, literal/relative/symlink/dangling AppImage, binary, and a directory named *.AppImage in one TestOverrideInstall matrix"
  - "Binary-tier override matrix: env-only packed, flag-beats-broken-env, dangling fail-loud, directory unchanged, all over the shared fake-AppImage fixture"
  - "Selection-tuple assertion: startup stderr must carry layout=, version=, root=/appimage= for the configured shape"

requirements-completed: [MNT-02, MNT-04, DISC-03]

coverage:
  - id: D1
    description: "--app-server / ZCODE_ACP_APP_SERVER map a directory, an app-server binary, and a literal/symlinked/dangling packed .AppImage; every packed AppImage is stored absolute (never a bare basename), a directory named *.AppImage stays a tree, and a missing AppImage stays packed so the mount step fails loud"
    requirement: "MNT-02"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestOverrideInstall"
        status: pass
    human_judgment: false
  - id: D2
    description: "The override probe names a packed AppImage and its version and rejects a volatile /tmp/.mount_* AppImage path (as well as a volatile tree root), so it never becomes a discovery winner"
    requirement: "DISC-03"
    verification:
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestOverrideInstall/probe_rejects_a_volatile_AppImage_path"
        status: pass
      - kind: unit
        ref: "internal/supervisor/discover_test.go#TestOverrideInstall/probe_row_names_the_packed_override"
        status: pass
    human_judgment: false
  - id: D3
    description: "A mounted run reports Install.Version (and a pre-mount packed install the AppImage basename) to the PAT mint instead of the documented 0.0.0 fallback; a versionless bare install still falls back to 0.0.0"
    requirement: "MNT-04"
    verification:
      - kind: unit
        ref: "cmd/zcode-acp/main_test.go#TestDesktopVersionMounted"
        status: pass
      - kind: unit
        ref: "cmd/zcode-acp/main_test.go#TestWiringCodingPlanSeams"
        status: pass
    human_judgment: false
  - id: D4
    description: "Env-only and flag-over-env packed overrides mount and serve an ACP session; a dangling override exits 1 naming the path and the mount step with no discovery fallthrough or extraction; a directory override keeps its tree behavior"
    requirement: "MNT-02, DISC-03"
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageOverride"
        status: pass
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestRootModeUnchanged"
        status: pass
    human_judgment: false
  - id: D5
    description: "Startup diagnostics name layout, version, root, and appimage for the selected install (and repeat the mounted root/version); both --app-server help strings name the install root, app-server binary, and packed .AppImage"
    requirement: "DISC-03"
    verification:
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAppImageOverride (layout=/version=/root=/appimage= stderr asserts)"
        status: pass
      - kind: automated
        ref: "grep -c 'packed .AppImage' cmd/zcode-acp/main.go == 2; grep -c '\"appimage\"' cmd/zcode-acp/main.go == 4"
        status: pass
    human_judgment: false

duration: 7 min
completed: 2026-11-24
plan_head_before: 4ad7b5b11177e3085fbef7097b2749afb4753866
plan_head_after: 707fb16b5e4dc6a55756efb6af3b869372220e19
status: complete
---

# Phase 10 Plan 04: Complete Override Routing Summary

**`--app-server` now handles all three real user shapes — install root, app-server binary, and packed `.AppImage` including symlinked and dangling values — with a mount-safe PAT client version and startup diagnostics that name the selected layout, version, and path.**

## Performance

- **Duration:** 7 min
- **Started:** 2026-11-23T23:57:23Z
- **Completed:** 2026-11-24T00:04Z
- **Tasks:** 2
- **Files modified:** 7 (3 production/test files touched by the plan, 1 shared supervisor test constant, 2 TDD evidence records)

## Accomplishments

- `TestOverrideInstall` pins the whole mapper matrix: empty, valid/unvalidated directory, literal AppImage, relative literal (asserts `filepath.IsAbs` and never a bare basename), symlink whose target is a packed AppImage, dangling AppImage (still `ok=true` so the mount step fails loud), non-AppImage binary, a directory named `*.AppImage` (stays a tree), the packed probe row, and the volatile-path rejection.
- `probeOverride` now rejects a volatile `/tmp/.mount_*` **AppImage** path via one `volatileInstallPath` guard, so a packed override from a live mount can never become the discovery winner.
- `desktopVersion` is version-first: `Install.Version`, then the packed `AppImage` basename (`versionFromPath`), then the existing Binary/Root/Entrypoint scan — a mounted root under a versionless `.mount_*` path now reports 4.0.0 to the PAT mint instead of the 0.0.0 fallback.
- `discoverInstall` logs the D-09 selection tuple (`layout`, `version`, `root`, `appimage`, plus `binary`), and `acquireMount` logs the mounted `layout`/`version`/`root`/`appimage` after `PrepareMount`.
- Both `--app-server` help strings (root arm and omp arm) now name all three accepted shapes; precedence (`firstNonEmpty(flag, env)`) and the exit-code convention are untouched.
- `TestAppImageOverride` proves the binary-tier surface: env-only packed override mounts and serves initialize/session-new, the flag beats a broken env value, a dangling override exits 1 with the path and `spawn the helper` step (no discovery fallthrough, no session, no extraction copy), and the directory override keeps `layout=explicit`/`root=` diagnostics.

## Task Commits

Each task was committed atomically (TDD: RED -> GREEN), plus the scheduled lint refactor:

1. **Task 1 RED — override matrix + mount-safe client version** — `659c60a` (test)
2. **Task 1 GREEN — volatile AppImage rejection + version-first desktopVersion** — `8b68742` (feat)
3. **Task 2 RED — env/flag override binary matrix + selection diagnostics** — `d2ad9cc` (test)
4. **Task 2 GREEN — full selection tuple + three-shape help strings** — `05d13ca` (feat)
5. **REFACTOR — lint-driven constant/line-wrap cleanup** — `707fb16` (refactor)

**Plan metadata:** the `docs(10-04)` commit carrying this SUMMARY, STATE.md, and ROADMAP.md.

## Files Created/Modified

- `internal/supervisor/discover.go` — `volatileInstallPath` and the probe rejection of a volatile packed AppImage path; the 10-01 `OverrideInstall` routing/absolute-path contract is verified by tests and unchanged.
- `internal/supervisor/discover_test.go` — `TestOverrideInstall` matrix and helpers (`assertPackedOverride`, `overrideProbes`), plus the shared `fixtureExplicitLayout` constant.
- `internal/supervisor/e2e_test.go` — uses the shared explicit-layout fixture constant (lint).
- `cmd/zcode-acp/main.go` — version-first `desktopVersion`/`versionFromPath`, selection-tuple startup logs, mounted-root/version log, both help strings.
- `cmd/zcode-acp/main_test.go` — `TestDesktopVersionMounted` and `TestAppImageOverride` with the shared fake-AppImage fixture, `e2eRideOverride`, and the selection-tuple assert.
- `.planning/tdd/10-04-task{1,2}-red.json` — machine-verified RED evidence (`RED_EVIDENCE_OK`).

## Decisions Made

- **Most routing was already shipped.** 10-01's `OverrideInstall` already implemented directory-first detection, resolvable-target packed detection, and the absolute-path contract; this plan closes the remaining gaps (volatile AppImage probe rejection, mount-safe version) and pins every routing arm with tests rather than rewriting working code.
- **Probe-level volatile rejection.** A volatile AppImage path is rejected in the override row instead of relying on `PrepareMount`, keeping discovery's "never resolve a volatile mount" rule symmetric for roots and AppImages.
- **Version-first with an AppImage fallback.** `Install.Version` wins because it is parsed from the AppImage name or the unpacked dir; the basename scan is the second source so an Install that carries only `AppImage` still reports the real version.
- **Keep `binary` in the diagnostics.** The plan names layout/version/root/appimage; keeping `binary` preserves the executable path for a binary override while `root`/`appimage` name the layout path D-09 requires.
- **Mounted log mirrors the selection tuple.** `acquireMount` logs the mounted Install's layout/version/root/appimage after `PrepareMount`, so the mounted root and the preserved version are visible.

## Deviations from Plan

### Ordering deviations (recorded, no behavior impact)

- **Task 1's production routing changes were already in place** from plan 10-01. `TestOverrideInstall`'s directory, literal, relative, symlink, dangling, binary, and `*.AppImage` directory arms are CONFIRM-GREEN: they lock existing behavior. The RED target that failed was the volatile-AppImage probe rejection (and the version seam in `cmd`), which this plan implemented.
- **The extra REFACTOR commit** is the scheduled lint pass: `fixtureExplicitLayout` deduplicates the `"explicit"` literal across `discover_test.go` and `e2e_test.go`, one path literal replaces `filepath.Join("/tmp", ...)`, and two over-long lines were wrapped. No behavior changed.

**Total deviations:** 0 auto-fixed (no bugs/blocking issues) + 2 recorded ordering notes.
**Impact on plan:** No scope creep; the routing arms were verified rather than reimplemented, and the refactor is style-only.

## TDD Gate Compliance

| Phase | Commit | Evidence |
|-------|--------|----------|
| RED (Task 1) | `659c60a` — `test(10-04)` | `TestOverrideInstall/probe_rejects_a_volatile_AppImage_path` reported `found: packed AppImage /tmp/.mount_ZCode-...` and `TestDesktopVersionMounted` returned `""` for the mounted and packed installs (`RED_EVIDENCE_OK`, `.planning/tdd/10-04-task1-red.json`) |
| GREEN (Task 1) | `8b68742` — `feat(10-04)` | All 11 `TestOverrideInstall` arms and all 3 `TestDesktopVersionMounted` arms pass; `TestWiringCodingPlanSeams` stays green |
| RED (Task 2) | `d2ad9cc` — `test(10-04)` | The directory override startup line reported `layout=explicit binary=... appimage= version=` with no `root` attribute, so the selection-diagnostics assertion failed; the env-only, flag-beats-env, and dangling arms passed (`RED_EVIDENCE_OK`, `.planning/tdd/10-04-task2-red.json`) |
| GREEN (Task 2) | `05d13ca` — `feat(10-04)` | `TestAppImageOverride` and `TestRootModeUnchanged` pass in full |
| REFACTOR | `707fb16` — `refactor(10-04)` | Lint-driven only; full `./...` suite and `golangci-lint` stay green |

No gate violations: each `feat(10-04)` descends from a `test(10-04)` commit, and each RED target executed and failed on a planned assertion (volatile-override rejection / version resolution; missing `root` diagnostic), not on build, setup, or fixture faults.

## Verification

| Command | Result |
|---------|--------|
| `go test -race -count=1 ./internal/supervisor/ -run TestOverrideInstall -v` | PASS (11/11 arms) |
| `go test -race -count=1 ./cmd/zcode-acp/ -run 'TestDesktopVersion\|TestWiringCodingPlanSeams\|TestAppImageOverride\|TestRootModeUnchanged' -v` | PASS |
| `go test -race -count=1 ./...` (`mise run test`) | PASS (all packages) |
| `mise run lint` (golangci-lint 2.15.0) | 0 issues |
| `mise run tidy-diff` | clean (one `require` line, acp-go-sdk v0.20.0) |

Acceptance greps (all pass): `func OverrideInstall` == 1, `inst.Version` >= 1 (3), `inst.AppImage` >= 1 (2), `func TestOverrideInstall` == 1, `func overrideInstall` == 0, `func TestAppImageOverride` == 1, `packed .AppImage` == 2, `"appimage"` == 4.

## Issues Encountered

- The first RED draft of `overrideProbes` asserted `ErrNoInstall` for every override-only fixture; a packed override legitimately resolves and wins, so the helper was corrected to return rows for both outcomes before recording evidence. This was a test-harness correction, not a production defect.
- The strict lint gates flagged the new `"explicit"` literals, one `filepath.Join("/tmp", ...)`, and two >120-char lines; all were resolved in the REFACTOR commit without behavior change.

## User Setup Required

None - no external service configuration required. The AppImage override still needs FUSE plus `libfuse2` at runtime (unchanged from 10-01/10-02; 10-06 documents the prerequisites).

## Next Phase Readiness

- 10-05 can drive the guarded real tier through the same override route: a real packed `.AppImage` given via `--app-server` is absolutized, mounted, and served; a directory/binary override keeps the explicit path.
- 10-06 documents the expanded override surface, the selection diagnostics tuple, and the version-first PAT behavior; the help strings now match the three-shape wording.
- The volatile-AppImage rejection closes the last discovery/override asymmetry: neither a tree root nor a packed value can come from a live `/tmp/.mount_*` path.

## Self-Check: PASSED

All created/modified files exist; all five commits are ancestors of HEAD; no commit in the plan deleted tracked files.
