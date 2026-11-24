---
phase: 10-mount-first-appimage-backend
plan: 05
subsystem: infra
tags: [appimage, mount, fuse, realdesktop, session-create, e2e, go]

# Dependency graph
requires:
  - phase: 10-mount-first-appimage-backend
    provides: 10-01 PrepareMount/Mount owner (Install/Stop/HelperPID), 10-02 staged signal/exit teardown, 10-03 packed discovery + Install.Version, 10-04 supervisor.OverrideInstall + version-first desktopVersion
  - phase: 03-bridge-spawn-supervisor
    provides: SpawnArgv/ChildEnv/SpawnChild, the Supervisor staged shutdown, zcode.Conn session/create round-trip
provides:
  - TestRealAppImageMount — env-gated real packed-AppImage mount -> real app-server -> session/create -> zero-leftover teardown
  - Read-only real-tier proofs: AppImage sha256 before/after and an extraction-shaped parent-directory snapshot (no additions; pre-existing extracts retained)
  - Retired superseded TestRealDesktop path; realdesktop_test.go is now the shared real-tier helper module with the canonical mount-first command
  - Mount-aware manual tier: one shared supervisor.OverrideInstall mapper and PrepareMount for packed winners across apicycle/codingplan/readbackdiag
  - Version-first cpClientVersion mirror of main.go desktopVersion
affects: [10-06 docs, phase-10 manual VER-02 run, verify-work]

# Actuals (#2632) — same chars/4 scale as the plan's estimate.
actuals:
  tokens: 8177
  tasks: 2
  commits: 2
  plan_head_before: 4a8a724f67f05b6ee8439340a99e2f365d4abd15
  plan_head_after: f856178cf0ee95b09d6fb1fa18631635c170c5de

tech-stack:
  added: []
  patterns:
    - "Ordered sync.Once teardown closure with nil-guarded waits/closes, registered before the first spawn assertion; non-fatal assertions inside t.Cleanup"
    - "Two-context discipline: supCtx drives Supervisor.Start, spawnCtx (WithoutCancel) is the exec backstop cancelled only after Start returns, and the Conn closes last"
    - "Real-tier read-only proof: sha256 equality plus an additive parent-directory snapshot (baseline entries must survive, additions must be zero)"
    - "One shared override mapper for production and the manual tier (supervisor.OverrideInstall), with a single mount acquisition point in discoverRealInstall"

key-files:
  created:
    - internal/supervisor/appimage_realdesktop_test.go
  modified:
    - internal/supervisor/realdesktop_test.go
    - test/realdesktop/apicycle_test.go
    - test/realdesktop/codingplan_test.go

key-decisions:
  - "assertCreateRoundTrip now takes the caller-owned zcode.Conn: the test keeps stdin open until the supervisor's staged shutdown returns, preventing a second child spawned by the restart loop after an early stdin EOF"
  - "The teardown closure is registered before the spawn and asserts with t.Errorf (never t.Fatalf), so a body failure neither leaks the helper/mount nor masks the original failure"
  - "The extraction proof is additive, not absolute: pre-existing ~/Applications/zcode-3.18.2/squashfs-root is allowed and asserted still present, while zero new squashfs-root/zcode-<ver> entries may appear"
  - "Discovery selecting a non-packed install skips with every probe row logged; ZCODE_ACP_REAL_APPIMAGE maps through supervisor.OverrideInstall and must resolve to a packed AppImage"
  - "Manual-tier discoverRealInstall mounts packed winners via PrepareMount and stops them in t.Cleanup, keeping startSupervisedChild's signature and all three call sites unchanged"

patterns-established:
  - "Real mount acceptance: PrepareMount -> mounted Install -> two-context supervisor spawn -> session/create -> ordered teardown -> sha256/extraction/ESRCH proofs"
  - "Manual-tier mount-awareness: one shared override mapper plus one mount acquisition point feeding the apicycle, coding-plan, and readback cycles"

requirements-completed: [VER-02, MNT-01, MNT-03, MNT-06]

coverage:
  - id: D1
    description: "TestRealAppImageMount is the only real-machine test: env-gated, absent from default builds, pin or discovery of a packed AppImage, production PrepareMount/spawn to session/create, ordered teardown with helper and app-server group ESRCH, and no live prompt on ZCode 4.0.0 (D-14)"
    requirement: "VER-02"
    verification:
      - kind: automated
        ref: "go test -tags realdesktop -run '^$' -count=1 ./internal/supervisor/ (compiles the guarded file without running the body)"
        status: pass
      - kind: automated
        ref: "go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v (SKIP without ZCODE_ACP_REAL_E2E=1)"
        status: pass
      - kind: manual_procedural
        ref: "ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v (manual-only real host run; not executed by the executor)"
        status: unknown
    human_judgment: true
    rationale: "The real run requires FUSE plus the live ZCode 4.0.0 desktop runtime; the plan's automated gate is compile-only and the actual run is the phase's Manual-Only verification."
  - id: D2
    description: "The superseded TestRealDesktop path is retired; realdesktop_test.go keeps only the shared real-tier helpers (realE2EEnv, realOptDir, realShutdownWatchdog, childDuplex, readBuiltinRevision, logObservedInstall, assertCreateRoundTrip) under the mount-first canonical command"
    requirement: "MNT-01"
    verification:
      - kind: automated
        ref: "grep -c 'func TestRealDesktop' internal/supervisor/realdesktop_test.go == 0; grep -c 'go test -tags realdesktop -count=1 -run TestRealAppImageMount' >= 1; go test -tags realdesktop -run '^$' ./internal/supervisor/"
        status: pass
    human_judgment: false
  - id: D3
    description: "The manual tier discovers through supervisor.OverrideInstall and mounts packed winners with PrepareMount; cpClientVersion prefers Install.Version then the AppImage basename, mirroring main.go desktopVersion"
    requirement: "MNT-03"
    verification:
      - kind: automated
        ref: "go test -tags realdesktop -run '^$' -count=1 ./test/realdesktop/ ./internal/supervisor/; grep 'supervisor.OverrideInstall'/'PrepareMount' in apicycle_test.go; grep 'inst.Version' in codingplan_test.go"
        status: pass
    human_judgment: false
  - id: D4
    description: "The AppImage bytes and parent directory are proven unchanged: sha256 equality and no new squashfs-root/zcode-<ver> entry, with pre-existing extracts retained (never deleted)"
    requirement: "MNT-06"
    verification:
      - kind: manual_procedural
        ref: "ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v (the assertions execute only in the manual real run)"
        status: unknown
    human_judgment: true
    rationale: "The hash and parent-directory proofs execute only against a real mounted AppImage; the automated gate compiles but does not run them."

duration: 6 min
completed: 2026-11-24
status: complete
---

# Phase 10 Plan 05: Guarded Real Mount Acceptance Summary

**TestRealAppImageMount drives a real packed AppImage through the production mount owner and spawn contract to session/create, proving sha256/no-extraction/zero-leftover teardown while the legacy real path is retired and the manual tier becomes mount-aware.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-11-24T00:12:03Z
- **Completed:** 2026-11-24T00:17:46Z
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- Added `internal/supervisor/appimage_realdesktop_test.go` (build tag `realdesktop`, guarded by `ZCODE_ACP_REAL_E2E=1`): resolves a packed AppImage by optional `ZCODE_ACP_REAL_APPIMAGE` pin through `supervisor.OverrideInstall` or production discovery (skipping with every probe row logged when discovery is not packed), records sha256 + extraction-shaped parent entries, calls `PrepareMount`, spawns the real app-server through `SpawnArgv`/`ChildEnv`/`SpawnChild` under the two-context discipline, runs `session/create`, and tears down in one `sync.Once` ordered closure.
- The teardown closure is registered before the spawn and nil-guards `startDone`, `child`, and `conn`: it cancels the supervision context, waits for the supervisor's staged shutdown, cancels the exec backstop, closes the Conn, calls `m.Stop()`, and polls `kill(-pgid, 0)` to ESRCH for both the app-server and the mount helper — with `t.Errorf`, so a body failure is neither masked nor leaked.
- Read-only proofs: AppImage sha256 is identical before/after; the parent listing gains zero `squashfs-root`/`zcode-<ver>` entries and every baseline entry survives (the host's pre-existing `zcode-3.18.2/squashfs-root` is allowed and never deleted).
- Retired `TestRealDesktop` and its private helpers (`startRealChild`, `assertRealShutdown`, `discoverRealInstall`, `realDesktopDeadline`); `realdesktop_test.go` is now the shared helper module with the canonical `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/` command.
- Manual tier made mount-aware: `discoverRealInstall` uses `supervisor.OverrideInstall` (local mirror deleted) and mounts packed winners via `PrepareMount` with a `t.Cleanup` `Stop`; `logObservedInstall` prints version/appimage; `cpClientVersion` prefers `inst.Version` then the AppImage basename, matching main.go `desktopVersion`. `startSupervisedChild` and all three call sites are unchanged.

## Task Commits

Each task was committed atomically:

1. **Task 1: Guarded real-machine acceptance and legacy real path retirement** — `42a9cb4` (test)
2. **Task 2: Mount-aware manual tier and version-first mirror** — `f856178` (test)

**Plan metadata:** the `docs(10-05)` commit carrying this SUMMARY, STATE.md, ROADMAP.md, REQUIREMENTS.md, and WINDOWS.md.

## Files Created/Modified

- `internal/supervisor/appimage_realdesktop_test.go` — `TestRealAppImageMount`, pin/discovery resolution, `startMountedChild`, ordered teardown, hash/extraction/mount-gone/ESRCH proofs.
- `internal/supervisor/realdesktop_test.go` — shared helpers only (`realE2EEnv`, `realOptDir`, `realShutdownWatchdog`, `childDuplex`, `logObservedInstall`, `readBuiltinRevision`, `assertCreateRoundTrip`); header rewritten to the mount-first scope.
- `test/realdesktop/apicycle_test.go` — shared override mapper, mount-aware `discoverRealInstall`, packed `logObservedInstall` fields; local `overrideInstall` removed.
- `test/realdesktop/codingplan_test.go` — version-first `cpClientVersion` plus `cpVersionFromPath`.

## Decisions Made

- **Conn ownership moved to the caller.** `assertCreateRoundTrip` now accepts the caller-owned `*zcode.Conn` so the test keeps stdin open until the supervisor's staged shutdown returns; the previous early `Close` is exactly the stdin-EOF race that could restart a second child before the cancel lands.
- **Teardown is non-fatal by construction.** A single `sync.Once` closure registered before the spawn performs cancel → wait Start → cancel spawn → close Conn → `Mount.Stop` → both-group ESRCH polls; every failure is `t.Errorf`, so it is safe both from `t.Cleanup` after a `t.Fatalf` and from the body's final explicit invocation.
- **Extraction proof is additive.** The parent snapshot requires baseline entries to survive and zero additions; the host's pre-existing `zcode-3.18.2/squashfs-root` is explicitly allowed and never deleted.
- **Non-packed discovery skips with diagnostics.** The real test is mount-only; if discovery selects a directory/binary layout it logs every probe row and skips, with the `ZCODE_ACP_REAL_APPIMAGE` pin as the deliberate way to force a packed file.
- **Manual tier reuses production acquisition.** One shared mapper (`supervisor.OverrideInstall`) and one mount point (`PrepareMount` inside `discoverRealInstall`) keep apicycle/codingplan/readbackdiag coherent without signature churn.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The new test's first draft declared a local `fileSHA256`, which collided with the existing helper in `mount_test.go` (same `supervisor_test` package, no build tag). The existing helper is reused instead of duplicating it, which also keeps the duplication linter quiet.

## Verification

| Command | Result |
|---------|--------|
| `go test -tags realdesktop -run '^$' -count=1 ./test/realdesktop/ ./internal/supervisor/` | PASS (compile-only, no bodies run) |
| `go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v` | PASS as SKIP without `ZCODE_ACP_REAL_E2E=1` (guard works) |
| `go test -race -count=1 ./internal/supervisor/ -run TestMount -v` | PASS (all mount lifecycle suites) |
| `go test -race -count=1 ./...` | PASS (all packages) |
| `mise run lint` (golangci-lint 2.15.0) | 0 issues |
| `go build ./... && go vet ./...` | PASS |
| `mise run tidy-diff` | clean (single `require` line, acp-go-sdk v0.20.0) |
| Acceptance greps | `//go:build realdesktop`=1, `ZCODE_ACP_REAL_E2E`=3, `ZCODE_ACP_REAL_APPIMAGE`=3, `TestRealAppImageMount`=3, `func TestRealDesktop`=0, canonical command>=1, `supervisor.OverrideInstall`=2, `func overrideInstall`=0, `PrepareMount`=2, `inst.Version`=2 |
| Manual real run | **NOT RUN** — manual-only per the plan: `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v` on a FUSE-capable host with the real AppImage |

## Manual-Only Items

- The guarded real run remains the phase's Manual-Only verification (VER-02): it must be executed once on a FUSE-capable machine with a real packed AppImage before ship. Record: probe rows, selected layout/version/AppImage, mount path, helper/app-server PIDs, `session/create` reply, both `kill(-pgid,0)` ESRCH results, unchanged sha256, and the parent listing with no new extraction entry.
- A live prompt on ZCode 4.0.0 remains explicitly out of scope (D-14, protocol-drift milestone); the test comments state the gap rather than hiding it.

## Threat Flags

None — test-only changes; no new production endpoints, auth paths, file-access patterns, or schema changes. T-10-18 (AppImage tampering) and T-10-19 (leftover processes/mounts) are mitigated by the hash/no-extraction and teardown proofs above; T-10-20 stays satisfied (no credentials or provider contents logged).

## Next Phase Readiness

- 10-06 can document the real-tier command, the `ZCODE_ACP_REAL_APPIMAGE` pin, and the D-14 scope gap; the canonical command in `realdesktop_test.go` and the manual-tier mount behavior are final.
- The phase's remaining manual gate is the real run itself; the shared helper module and both mount paths compile together under `-tags realdesktop`.

## Self-Check: PASSED
