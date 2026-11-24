---
phase: 10-mount-first-appimage-backend
verified: 2026-11-24T05:51:05Z
status: passed
score: 30/30 must-haves verified
covered_files: [".planning/phases/10-mount-first-appimage-backend/10-01-PLAN.md", ".planning/phases/10-mount-first-appimage-backend/10-01-SUMMARY.md", ".planning/phases/10-mount-first-appimage-backend/10-02-PLAN.md", ".planning/phases/10-mount-first-appimage-backend/10-02-SUMMARY.md", ".planning/phases/10-mount-first-appimage-backend/10-03-PLAN.md", ".planning/phases/10-mount-first-appimage-backend/10-03-SUMMARY.md", ".planning/phases/10-mount-first-appimage-backend/10-04-PLAN.md", ".planning/phases/10-mount-first-appimage-backend/10-04-SUMMARY.md", ".planning/phases/10-mount-first-appimage-backend/10-05-PLAN.md", ".planning/phases/10-mount-first-appimage-backend/10-05-SUMMARY.md", ".planning/phases/10-mount-first-appimage-backend/10-06-PLAN.md", ".planning/phases/10-mount-first-appimage-backend/10-06-SUMMARY.md", "README.md", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "docs/protocol/install-discovery.md", "docs/provenance.md", "internal/supervisor/appimage_realdesktop_test.go", "internal/supervisor/discover.go", "internal/supervisor/discover_internal_test.go", "internal/supervisor/discover_test.go", "internal/supervisor/e2e_test.go", "internal/supervisor/mount.go", "internal/supervisor/mount_internal_test.go", "internal/supervisor/mount_test.go", "internal/supervisor/realdesktop_test.go", "internal/supervisor/spawn.go", "scripts/vm-e2e.sh", "test/realdesktop/apicycle_test.go", "test/realdesktop/codingplan_test.go"]
covered_digest: "v4:sha256:0d04f7291f308c83b15bb5ff213f1f120447778f42a979fa5a85de81ba1e17f1"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 30/30
  gaps_closed: []
  gaps_remaining: []
  regressions: []
---

# Phase 10: Mount-First AppImage Backend Verification Report

**Phase Goal:** The wrapper runs the ZCode app-server directly from a packed `.AppImage`
through a managed `--appimage-mount` helper — no extraction, no ~850 MB on-disk
duplicate — and discovery always selects the newest usable backend across layouts, so
a stale unpacked `squashfs-root` can never pin an older version.
**Verified:** 2026-11-24T05:51:05Z
**Status:** passed
**Re-verification:** Yes — post-squash-merge against HEAD `3bc8a96` (PR #6 merged the
phase onto master's `startBackend`/`backend` launch abstraction; PR #7 added the
three-shape `--app-server` help wording and its regression test)

## Merge Re-Verification (post-squash)

The prior fingerprint covered the phase-branch content. Master had since refactored the
launch path into `startBackend`/`backend` (shared by the ACP and one-shot text arms), so
the squash merge composed the mount lifecycle onto that abstraction and changed
`cmd/zcode-acp/main.go` and `cmd/zcode-acp/main_test.go`. Every must-have was re-checked
against the merged code, not against the SUMMARY narrative:

- `startBackend(ctx, cfg)` (`cmd/zcode-acp/main.go:1046`) performs discovery, then
  `acquireMount` (`:1315`), then `startSupervised` (`:1395`); both `run` (`:1154`) and
  `runOmpText` (`:524`) call it, so the mount lifecycle is shared.
- The spawn context is `context.WithCancel(context.WithoutCancel(ctx))` (`main.go:1066`),
  while the supervision context derives from the caller; the spawn cancel fires only
  after the supervisor's staged shutdown returns.
- `backend.stop()` (`main.go:1132`) runs supervisor teardown → spawn backstop → mount
  stop, idempotently; `backend.finish` (`:1238`) and `classifyRunExit` (`:1275`) keep
  teardown errors winning over signal/client/child/mount-loss causes.
- `run`'s acquisition-failure branch (`main.go:1169-1175`) still uses
  `supervisor.MountFailure(err)` so a failed teardown under a signal exits 1 (CR-01),
  while a pure readiness cancellation exits 0.
- `desktopVersion` (`main.go:858-874`) keeps the version-first order: `Install.Version`
  → AppImage basename → Binary/Root/Entrypoint path scan.
- PR #7 (`3bc8a96`) is material to B5: at the squashed phase commit both `--app-server`
  help strings still read "(or app-server binary)"; HEAD now names all three shapes
  (`main.go:126-127,325-326`) and `testOmpFlagSetAppServerHelp` locks it. The truth was
  over-claimed in the previous report but is satisfied in the code actually under
  verification.

CI evidence (GitHub): PR #6 (`47e9c37`) and PR #7 (`3bc8a96`) are both MERGED with all
three checks SUCCESS — `build / vet / lint / test / tidy`, `govulncheck`, `secret-scan`.

## Goal Achievement

### Observable Truths

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| A1 | A run started with `--app-server` and a packed `.AppImage` mounts it through the managed `--appimage-mount` helper and serves ACP from the mount | ✓ VERIFIED | Post-merge `TestAppImageMountEndToEnd` PASS (0.90s, built wrapper, production `startBackend` → `acquireMount` → `PrepareMount`); initialize + 2× session/new + 2 prompts served from the mount; fake transcript carries exactly 2 `session/send` and 1 `session/stop`; `main.go:1046,1154-1192,1315-1332` |
| A2 | The app-server is spawned with the unchanged `SpawnArgv`/`ChildEnv` contract against the mounted root | ✓ VERIFIED | `startSupervised` passes `SpawnArgv(inst)`/`ChildEnv(inst, os.Environ())` (`main.go:1410-1413`); `mount.Install()` fills Root/Binary/Entrypoint from the validated mount (`mount.go:416-426`); e2e + real run serve only through `<mount>/zcode` |
| A3 | The helper environment is an allowlist by construction (PATH/HOME/TMPDIR), so `APPIMAGE_EXTRACT_AND_RUN` and loader vars can never cross | ✓ VERIFIED | Post-merge `TestMountEnv` PASS with 8 hostile vars set; exact equality asserts only 3 vars (`mount_internal_test.go:29+`); `helperEnv` (`mount.go:806-823`) |
| A4 | The whole helper stdout line is the path; only an absolute, non-symlink `.mount_*` directory one component under the resolved private base is trusted; a spaced TMPDIR is accepted whole | ✓ VERIFIED | `validateMountLine` (`mount.go:830-859`); `TestMountSpacedTmpdir` PASS (space preserved); `TestMountFailures/bogus-line` PASS; real run accepted `/tmp/zcode-acp-mount-3221011186/.mount_ZCode-2AZ8WU` |
| A5 | A private base containing CR, LF, or NUL is rejected before the helper spawns, naming TMPDIR | ✓ VERIFIED | `basePathProblem` covers all three (`mount.go:787-798`); `TestMountRejectBadTmpdir` PASS (LF arm: error names TMPDIR and the base, rejected base removed, nothing spawned) |
| A6 | Text written by the helper immediately before exit is always in the failure error (stdout tap drained to EOF by exec's Wait, not discarded by WaitDelay) | ✓ VERIFIED | `TestMountEarlyExitOutput` PASS; `TestMountFailures/late-text` PASS; `SpawnOpts.StdoutTap` installed as `cmd.Stdout` (`spawn.go:268-272`) so Wait joins the copier |
| A7 | Teardown stops the app-server before the helper; the helper is reaped and the mount/private base are gone | ✓ VERIFIED | Post-merge `TestAppImageMountEndToEnd` asserts helper TERM record says `app-server-gone-on-term`, mount + parent base gone, both PIDs ESRCH, AppImage hash unchanged; `backend.stop` order (`main.go:1132-1147`); `TestMountStop` PASS |
| A8 | SIGINT, SIGTERM, and SIGHUP run the staged teardown with zero orphans/stale mounts; the spawn context is not signal-derived (clean stdin-EOF drain) | ✓ VERIFIED | Post-merge `TestAppImageMountSignal` 6/6 arms PASS (26.50s: SIGTERM/SIGINT/SIGHUP + readiness signal + 2 teardown-failure arms); each live arm asserts mount/base gone, both PIDs ESRCH, ordered teardown, and a served transcript — the fake writes it only after the EOF-drain exit, so a signal-derived spawn ctx could not produce it; `main.go:1066,1158-1160,1203-1219` |
| A9 | A mount that never disappears makes `Stop` fail at the deadline naming the mount and manual-unmount remediation; the private base is retained, never traversed by `RemoveAll` | ✓ VERIFIED | `TestMountTeardownFailure` PASS; binary arm `teardown failure outranks the signal` PASS (exit 1, `.mount_*` retained, base retained); `RemoveAll` gated on the gone proof (`mount.go:549-578,635-654`) |
| A10 | Helper death or tree loss while the app-server lives ends the session with `ErrMountLost` and never respawns against the vanished root | ✓ VERIFIED | Post-merge `TestMountLost` (helper-dies, tree-vanishes, stop-is-not-loss), `TestMountGuard` (spawn seam runs exactly once, returns `ErrMountLost`), and `TestAppImageMountLost` (exit 1 naming the AppImage; no completed ride after loss) all PASS; guard in `startSupervised` (`main.go:1406`), loss select (`:1198-1214`), cause (`:1258-1268`) |
| A11 | A mount-backed run creates no `squashfs-root`/`zcode-<ver>` directory and leaves the AppImage bytes unmodified; no extraction seam remains | ✓ VERIFIED | `TestMountNoExtraction` PASS (sha256 + fixture snapshot identical); e2e sha256 + `e2eAssertNoExtraction`; post-merge real run: sha256 unchanged, 0 added entries; `grep ExtractAppImage` in `mount.go`/`discover.go` → 0; `grep appimage-extract` in `internal/`/`cmd/` → 0 production hits |
| A12 | Mount failures (missing FUSE, early helper exit, readiness timeout, bogus line, unspawnable tree) exit non-zero with AppImage, failing step, observed output, and remediation; never an extraction fallback; helper reaped, base removed | ✓ VERIFIED | Post-merge `TestAppImageMountFailures` (early-exit, bogus-line, missing-tree, no-fuse) PASS: exit 1, AppImage + step + `remediation:`; `TestMountFailures` 6 arms (incl. `no-line` readiness timeout, late-text) PASS with helper ESRCH + no base; `classifyMountFailure` (`mount.go:379-399`) |
| A13 | A signal during mount acquisition that also left a failed teardown exits 1 (never a masked 0); a pure readiness cancellation exits 0 | ✓ VERIFIED | `TestMountFailure` 11 arms PASS; binary `TestAppImageMountSignal` arms `signal during mount readiness` (exit 0) and `failed teardown during mount readiness outranks the signal` (exit 1, names AppImage, step, remediation, `teardown incomplete`) PASS; `MountFailure` (`mount.go:138-156`) + `run` (`main.go:1169-1175`) |
| B1 | `--app-server` and `ZCODE_ACP_APP_SERVER` accept a packed `.AppImage`; directory and binary shapes behave as before; the packed value is mounted and both `<mount>/zcode` and `<mount>/resources/glm/zcode.cjs` are validated before spawn | ✓ VERIFIED | Post-merge `TestAppImageOverride` 4/4 arms PASS (env-only mounts/serves; flag beats broken env; dangling fails loud; directory keeps tree shape); `TestOverrideInstall` 11/11 PASS; `validateMountTree` (`mount.go:864-892`) |
| B2 | Every packed override is stored absolute (never PATH-resolved); a dangling packed override fails loud at the mount step instead of falling through to discovery | ✓ VERIFIED | `TestOverrideInstall/relative literal AppImage` and `dangling AppImage` PASS; `TestAppImageOverride/dangling packed override fails loud` PASS (exit 1, names path + mount step, no discovery fallback); `OverrideInstall` (`discover.go:297-340`) |
| B3 | A mounted run's PAT client version uses `Install.Version` (or the AppImage basename), never the `0.0.0` fallback | ✓ VERIFIED | Post-merge `TestDesktopVersionMounted` 3/3 arms PASS (mounted → 4.0.0; pre-mount packed → basename; bare → fallback); `desktopVersion` (`main.go:858-874`) |
| B4 | The override probe names a packed AppImage and its version and rejects a volatile `/tmp/.mount_*` path | ✓ VERIFIED | `TestOverrideInstall/probe row names the packed override` and `probe rejects a volatile AppImage path` PASS; `volatileInstallPath` (`discover.go:236-248,357-361`) |
| B5 | Startup diagnostics log the selected layout, version, and resolved path; both `--app-server` help strings name all three accepted shapes | ✓ VERIFIED | `discoverInstall` logs layout/version/root/appimage/binary (`main.go:1382-1384`); `acquireMount` logs the mounted tuple (`:1327-1329`); e2e asserts `layout=appimage`, `version=`, `appimage=`; help strings `main.go:126-127,325-326` name install root, app-server binary, packed AppImage; PR #7 regression test `testOmpFlagSetAppServerHelp` PASS |
| C1 | Discovery selects the newest usable backend by version: a packed 4.0.0 beats a stale unpacked `squashfs-root` 3.18.2 | ✓ VERIFIED | `TestDiscoverRankPackedOverStaleUnpacked` PASS (packed winner + `RequiresMount`, rank row names layout/version/path) |
| C2 | Every AppImage candidate is evaluated; dangling symlink, unresolvable path, volatile mount, invalid tree, non-executable file, or unparseable version is a miss row and the scan continues | ✓ VERIFIED | `TestDiscoverPerCandidateMisses` (dangling, invalid tree) PASS; `TestDiscoverRejectsVolatileMounts` PASS; `TestDiscoverVersionUnknown` PASS; `evaluateAppCandidate`/`evaluatePackedCandidate` (`discover.go:579-696`) |
| C3 | Diagnostics report the selected backend's layout, version, and resolved path, plus one hit/miss row per candidate and the failing step | ✓ VERIFIED | `TestDiscoverDiagnostics` PASS (rank row + 3 ordered candidate rows: 1 hit, 2 miss with version); `appCandidate.selection/result` (`discover.go:775-803`) |
| C4 | Equal versions break deterministically: unpacked tree, then packed symlink, then packed regular file, then lexicographic path | ✓ VERIFIED | `TestDiscoverTieBreak` 4 arms PASS; rank constants (`discover.go:66-70`); `sortAppCandidates` (`discover.go:747-759`); documented in `install-discovery.md:36` |
| C5 | Discovery stays a pure read: no mount, no extraction, no new directory | ✓ VERIFIED | `TestDiscoverPureRead` PASS (tree snapshot identical before/after; no `.mount_*`); no mount/extract call in `discover.go` |
| C6 | XDG AppImage-family targets join the version ranking; an older XDG target cannot pin an older version | ✓ VERIFIED | `TestDiscoverXDGAppImageSeed` 4/4 arms PASS incl. `unpacked seed loses to a newer packed candidate`; package tree keeps step-2 precedence (D-10) |
| D1 | CI-safe scripted fake AppImage runtime drives the mount lifecycle (mount line, readiness, teardown, early death, missing FUSE) without FUSE | ✓ VERIFIED | VER-01: `mount_test.go` + `main_test.go` fake runtimes run here with no FUSE; `go test -race -count=1 ./cmd/zcode-acp/` and the full suite PASS |
| D2 | An env-gated real-machine test mounts a real packed AppImage, spawns the app-server from the mount, completes `session/create`, and tears down with zero leftovers | ✓ VERIFIED | Re-run post-merge on this host (2026-11-24): `ZCODE_ACP_REAL_E2E=1 ZCODE_ACP_REAL_APPIMAGE=/home/nil/Applications/ZCode-4.0.0-linux-x64.AppImage go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v` → PASS (4.13s); mount `/tmp/zcode-acp-mount-3221011186/.mount_ZCode-2AZ8WU`, helper pid 64413; app-server pgid 64419 spawned from `<mount>/zcode`; `session/create ok`; both groups ESRCH; mount + base gone |
| D3 | The real test proves AppImage bytes unchanged and no new extraction entry, allowing pre-existing extracts | ✓ VERIFIED | Same post-merge run: sha256 `2cedc5fceee1ffcc74b00cca292bfa86522cf7201ba8bc93d0c5500b6fefcc47` identical before/after; `extraction check under /home/nil/Applications: 1 pre-existing entry unchanged, 0 added`; the only extraction-shaped entry is the pre-existing `zcode-3.18.2` |
| D4 | The real test never runs a live prompt on ZCode 4.0.0 (D-14) and documents the gap | ✓ VERIFIED | Test stops after `assertCreateRoundTrip`; re-run logged the scope note; file header + `docs/provenance.md` "verified 4.0.0 scope stops at session/create" (lines 98-104) |
| D5 | Manual-tier realdesktop tests mount packed installs through the shared `supervisor.OverrideInstall` mapper; the superseded `TestRealDesktop` path is retired and the canonical mount-first command is documented | ✓ VERIFIED | `test/realdesktop/apicycle_test.go:338-382` uses `supervisor.OverrideInstall` + `PrepareMount`; `codingplan_test.go` mirrors version-first; `grep 'func TestRealDesktop'` → 0; `realdesktop_test.go:18-21` canonical command; both packages compile with the tag |
| D6 | Docs state the shipped contract: install-discovery managed-mount row + version ranking/tie-break + revised volatile rule; README packed override/read-only/FUSE/exact failures/no fallback; provenance read-only/no-new-extract/FUSE/4.0.0 scope; no documented claim contradicts behavior | ✓ VERIFIED | `install-discovery.md:26,36,45,48,113-125`; `README.md:38-52,62,68-69,107-125`; `provenance.md:19-26,92-104`; `bash experiments/check-docs.sh` exit 0; `node --test experiments/` 24 pass / 1 guarded skip / 0 fail; `bash -n scripts/vm-e2e.sh` exit 0 |

**Score:** 30/30 truths verified (0 present, behavior-unverified)

Notes on scope (not failures):
- The roadmap SC1 "session/create plus prompt" leg is proved on the CI fake-runtime backend (A1) and, for `session/create`, on the real backend (D2); a live prompt on real ZCode 4.0.0 is explicitly out of scope per D-14 and tracked as DISC-V2-01. The docs state this gap honestly.
- A7/A8/A9/A10/A13 are behavior-dependent (ordering/cancellation invariants). Each is proven by a passing named behavioral test on the merged code, not by symbol presence.
- Observation (non-blocking): the one-shot text arm (`runOmpText`) now shares `startBackend`/`backend.stop` and explicitly turns a mount teardown error into exit 1, but no dedicated binary test drives a packed-AppImage mount through the text arm; the ACP arm carries the packed e2e. The ordering/cancellation invariants are exercised on the shared `backend.stop` path, and no must-have requires a text-arm mount test, so this is recorded as an observation, not a gap.

### Required Artifacts

Artifact verification: `gsd-tools query verify.artifacts` over all six plans → **23/23 passed, 0 issues**.

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `internal/supervisor/mount.go` | PrepareMount/Mount/Install/Lost/Guard/Stop/HelperPID, failure classification, Statfs seam | ✓ VERIFIED | 1047 lines; all symbols present; substantive |
| `internal/supervisor/discover.go` | `RequiresMount`, `OverrideInstall`, version-ranked stage, diagnostics, no extraction seam | ✓ VERIFIED | 1220 lines; 11 OverrideInstall test arms cannot be green with a stub |
| `internal/supervisor/spawn.go` | `StdoutTap`/`StderrTap` leased streams over the unchanged SpawnArgv/ChildEnv | ✓ VERIFIED | Taps installed; app-server path byte-identical when nil |
| `cmd/zcode-acp/main.go` | mount acquisition + ordered teardown + guard + signal ctx + override + version-first | ✓ VERIFIED | `startBackend` calls `supervisor.PrepareMount` via `acquireMount`; `MountFailure`, `signal.NotifyContext`, `mount.Guard` all wired; `desktopVersion` version-first |
| `internal/supervisor/mount_test.go` | fake AppImage runtime + lifecycle matrix | ✓ VERIFIED | 1132 lines; 11 top-level tests incl. `writeFakeAppImage` |
| `internal/supervisor/mount_internal_test.go` | env allowlist + MountFailure predicate | ✓ VERIFIED | Exact allowlist and 11 predicate arms |
| `internal/supervisor/discover_test.go` | ranking, per-candidate, tie-break, diagnostics, purity | ✓ VERIFIED | 1435 lines; all phase matrix tests |
| `internal/supervisor/discover_internal_test.go` | dpkg seam (WR-01) | ✓ VERIFIED | 81 lines; installed/rc-remnant arms |
| `cmd/zcode-acp/main_test.go` | binary-tier mount e2e, failures, signal, loss, override, version seam | ✓ VERIFIED | 2985 lines; production-built wrapper exercised through the merged `startBackend` path |
| `internal/supervisor/appimage_realdesktop_test.go` | guarded real mount acceptance | ✓ VERIFIED | 432 lines; exercised by the post-merge 2026-11-24 run (PASS, session/create, zero survivors) |
| `internal/supervisor/realdesktop_test.go` | shared real-tier helpers + canonical command; legacy retired | ✓ VERIFIED | No TestRealDesktop; helpers live |
| `test/realdesktop/apicycle_test.go` | shared mapper + mount acquisition | ✓ VERIFIED | Uses `supervisor.OverrideInstall` + `PrepareMount` |
| `test/realdesktop/codingplan_test.go` | version-first client version | ✓ VERIFIED | `cpClientVersion` reads `inst.Version` first |
| `docs/protocol/install-discovery.md` | managed-mount row, ranking, tie-break, revised volatile/extraction policy | ✓ VERIFIED | Contains "Managed mount (wrapper-created)"; V-SRC AppImageKit tags |
| `README.md` | packed override, read-only, FUSE/libfuse2, exact failures, no fallback | ✓ VERIFIED | `libfuse2t64`, exact runtime messages, mid-session loss cause |
| `docs/provenance.md` | read-only/no-new-extract/FUSE/4.0.0 scope | ✓ VERIFIED | "read-only", "never extracted", pre-existing extracts untouched |
| `scripts/vm-e2e.sh` | mirrored ranking + FUSE prerequisites | ✓ VERIFIED | `/dev/fuse`, `libfuse.so.2`, version-ranked row; `bash -n` clean |
| `internal/supervisor/e2e_test.go` | adjusted existing e2e (one-line change) | ✓ VERIFIED | Compiles and full suite passes |

Wiring (Level 3): every phase file is imported/used by production code or is the production file; no orphaned artifacts found. The real-tier test files are build-tagged by design (not orphans).

### Key Link Verification

`gsd-tools query verify.key-links` over all six plans → **18/18 verified, 0 pending**.

| From | To  | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| main.go | mount.go | `PrepareMount` + `mount.Install` in `acquireMount` (called by `startBackend`) | ✓ WIRED | `acquireMount` calls both |
| mount.go | spawn.go | `SpawnChild` starts helper in own process group | ✓ WIRED | helper argv `{AppImage, --appimage-mount}` |
| main.go | spawn.go | `SpawnArgv`/`ChildEnv` receive mounted Install | ✓ WIRED | `startSupervised` passes mounted Binary/Entrypoint/BuiltinFile |
| main.go | mount.go | `mount.Lost` / `mount.Guard` drive exit cause | ✓ WIRED | `serve` select + `startSupervised` guard |
| main.go | supervise.go | guarded SpawnFunc refuses restart against vanished root | ✓ WIRED | `Guard()` before every `SpawnChild` |
| mount.go | spawn.go | `killGroup` / `child.Done` staged helper teardown | ✓ WIRED | `stopHelper` |
| discover.go | mount.go | packed Install carries AppImage/Version/RequiresMount | ✓ WIRED | `mount.go:262-290` consumes them |
| discover.go | spawn.go | unpacked winner still feeds SpawnArgv/ChildEnv | ✓ WIRED | `bundleInstall` |
| main.go | discover.go | `DefaultProbers().Discover` returns ranked winner | ✓ WIRED | `discoverInstall` |
| main.go | discover.go | `supervisor.OverrideInstall` wired into Override seam | ✓ WIRED | `discoverInstall` |
| main.go | mount.go | packed override reaches PrepareMount via RequiresMount | ✓ WIRED | `acquireMount` |
| main.go | patminter.go | `patClientVersion` feeds clientVersion from Install.Version | ✓ WIRED | `buildCodingPlanMachinery` |
| appimage_realdesktop_test.go | mount.go | PrepareMount/Install/Stop | ✓ WIRED | guarded body; re-run PASS |
| appimage_realdesktop_test.go | spawn.go | SpawnArgv/ChildEnv/SpawnChild | ✓ WIRED | `startMountedChild`; re-run PASS |
| apicycle_test.go | discover.go | `supervisor.OverrideInstall` replaces local mapper | ✓ WIRED | `discoverRealInstall` |
| install-discovery.md | discover.go | documented ranking/diagnostics mirror implementation | ✓ WIRED | fields match |
| README.md | mount.go | documented read-only/FUSE failures mirror mount owner | ✓ WIRED | messages match `classifyMountFailure` |
| provenance.md | mount.go | no-extraction/read-only contract enforced by mount owner | ✓ WIRED | no extractor exists |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `discover.go` ranked stage | `appCandidate.install` (AppImage, Version) | `AppDirs()` glob scan + XDG seeds + per-file `Lstat`/`EvalSymlinks`/executable/version checks | Yes — filesystem-derived, fixtures prove real ranking; no static return | ✓ FLOWING |
| `mount.go` `Mount.Install()` | Root/Binary/Entrypoint/Version/BuiltinFile | validated `--appimage-mount` line + `validateMountTree` + post-mount builtin stat | Yes — mount root from the helper, tree validated; real run mounted 4.0.0 | ✓ FLOWING |
| `main.go` spawn | `inst.Binary`/`Entrypoint`/`BuiltinFile` | mounted Install (or discovered tree) | Yes — e2e + real run serve only through `<mount>/zcode`; transcript proves frames | ✓ FLOWING |
| `main.go` `desktopVersion` | `Install.Version` → AppImage basename → path scan | discovery-parsed filename/dir version | Yes — unit arms prove 4.0.0, basename, and fallback; not a stub | ✓ FLOWING |
| diagnostics | selection tuple (layout/version/root/appimage) | same Install that is spawned | Yes — e2e + real run stderr assert/log the values | ✓ FLOWING |

No `⚠️ STATIC`, `✗ DISCONNECTED`, or `✗ HOLLOW_PROP` found.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Build / vet (merged tree) | `go build ./... && go vet ./...` | both exit 0 | ✓ PASS |
| Packed-AppImage mount e2e + mid-session loss + signals + failures + override + version seam | `go test -race -count=1 ./cmd/zcode-acp/ -run 'TestAppImageMountEndToEnd\|TestAppImageMountLost\|TestAppImageMountSignal\|TestAppImageMountFailures\|TestAppImageOverride\|TestDesktopVersionMounted' -v` | PASS (30.91s; all arms incl. 6 signal arms and 4 failure arms) | ✓ PASS |
| Mount/discovery matrix | `go test -race -count=1 ./internal/supervisor/ -run 'TestMount\|TestDiscover\|TestOverrideInstall' -v` | PASS (2.73s; all arms) | ✓ PASS |
| `cmd` package under race | `go test -race -count=1 ./cmd/zcode-acp/` | `ok ... 39.380s` | ✓ PASS |
| Full Go suite | `go test -race -count=1 ./...` | all packages ok (acpfront 102.3s, cmd 40.4s, supervisor 3.6s, …) | ✓ PASS |
| Real-machine acceptance (re-run post-merge) | `ZCODE_ACP_REAL_E2E=1 ZCODE_ACP_REAL_APPIMAGE=/home/nil/Applications/ZCode-4.0.0-linux-x64.AppImage go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v` | PASS (4.13s): session/create ok, both groups ESRCH, mount/base gone, sha256 unchanged, 0 added entries | ✓ PASS |
| Docs gate | `bash experiments/check-docs.sh` | exit 0 | ✓ PASS |
| Experiments suite | `node --test experiments/` | 24 pass, 1 guarded skip, 0 fail | ✓ PASS |
| Lint | `golangci-lint run` (v2.15.0) | `0 issues.` (one deprecation notice for the `exhaustruct` option) | ✓ PASS |
| Tidy drift | `go mod tidy -diff` | exit 0, no output | ✓ PASS |
| vm-e2e syntax | `bash -n scripts/vm-e2e.sh` | exit 0 | ✓ PASS |
| CI (PR #6 / #7) | `gh pr view 6/7 --json statusCheckRollup` | both MERGED; `build / vet / lint / test / tidy`, `govulncheck`, `secret-scan` all SUCCESS | ✓ PASS |

### Manual Real-Machine Acceptance (UAT — Completed 2026-11-24; re-run post-merge)

The original manual UAT (`.planning/phases/10-mount-first-appimage-backend/10-UAT.md`, status `complete`, 2/2 passed, 0 issues, commit `3132fe6`) closed D2/D3. This re-verification re-ran the same canonical command against the merged tree on this host.

**Command (run from `/home/nil/DiskD/W/Djarvur/zcode-acp`):**
`ZCODE_ACP_REAL_E2E=1 ZCODE_ACP_REAL_APPIMAGE=/home/nil/Applications/ZCode-4.0.0-linux-x64.AppImage go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/ -v`

**Result:** PASS (4.13s).

| Check | Observed (post-merge re-run) | Status |
| ----- | -------- | ------ |
| Pinned backend | `/home/nil/Applications/ZCode-4.0.0-linux-x64.AppImage`; `layout=appimage version=4.0.0` | ✓ |
| Mount | `/tmp/zcode-acp-mount-3221011186/.mount_ZCode-2AZ8WU`, helper pid 64413 | ✓ |
| App-server | spawned from `<mount>/zcode`, pgid 64419 | ✓ |
| Session | `session/create ok: sessionId sess_6259cf58-4b8d-4ec3-8a79-d777559f5dab` | ✓ |
| Zero survivors | `kill(-64419, 0)` ESRCH; `kill(-64413, 0)` ESRCH | ✓ |
| Mount teardown | no `zcode` entry and no private base at `/tmp/zcode-acp-mount-3221011186` | ✓ |
| Read-only | AppImage sha256 `2cedc5fceee1ffcc74b00cca292bfa86522cf7201ba8bc93d0c5500b6fefcc47` unchanged before/after | ✓ |
| No extraction | `/home/nil/Applications`: 1 pre-existing entry unchanged, 0 added | ✓ |
| Scope | no live prompt on 4.0.0 logged (D-14) | ✓ |

The earlier UAT run (2026-11-24, pre-merge host state) recorded the same outcomes with mount `/tmp/zcode-acp-mount-548128178/.mount_ZCode-4mDu6E`, helper pid 3738378, app-server pgid 3738384, PASS 5.45s.

Observation (non-blocking): three empty 0700 `zcode-acp-mount-*` bases dated 2026-11-24 02:15 remain under `/tmp`; they contain no mount and back no process, they pre-date the CR-01 fix, and the re-run's own base is absent. They are not mounts, not orphans, and not a must-have failure.

### Requirements Coverage

All 13 phase requirement IDs are declared in PLAN frontmatter and satisfied in code; none is orphaned (REQUIREMENTS.md maps exactly these 13 to Phase 10).

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ----------- | ----------- | ------ | -------- |
| MNT-01 | 10-01, 10-05 | app-server from a packed AppImage via managed mount, no extraction | ✓ SATISFIED | A1/A11; fake e2e + post-merge real run (`session/create` from a 4.0.0 mount) |
| MNT-02 | 10-01, 10-04 | flag/env accept packed AppImage; validate both mount paths; spawn from mount | ✓ SATISFIED | B1, A4, `validateMountTree` |
| MNT-03 | 10-01, 10-02, 10-05 | supervised helper, torn down after app-server on every exit path; zero orphans/stale mounts | ✓ SATISFIED | A7-A9, A13; e2e + signal arms + real re-run (both groups ESRCH); shared `backend.stop` wired into both the ACP and text arms |
| MNT-04 | 10-02, 10-04 | mount failures loud (path, step, remediation), non-zero, no extraction fallback | ✓ SATISFIED | A12, B2 |
| MNT-05 | 10-02 | mount loss ends session with explicit cause, no respawn | ✓ SATISFIED | A10 |
| MNT-06 | 10-01, 10-02, 10-03, 10-05 | no squashfs-root/zcode-<ver> created; AppImage unmodified | ✓ SATISFIED | A11; fake tier + post-merge real run (sha256 unchanged, 0 added, pre-existing retained) |
| DISC-01 | 10-03 | newest usable backend by version; stale unpacked never beats newer packed | ✓ SATISFIED | C1 |
| DISC-02 | 10-03 | per-candidate misses; scan continues | ✓ SATISFIED | C2 |
| DISC-03 | 10-03, 10-04 | diagnostics: layout, version, resolved path; version from filename/dir | ✓ SATISFIED | C3, B4, B5 |
| DISC-04 | 10-03 | deterministic documented ordering and tie-break | ✓ SATISFIED | C4, C5; `install-discovery.md:36` |
| VER-01 | 10-01, 10-02 | CI-safe fake-runtime lifecycle tests without FUSE | ✓ SATISFIED | D1 |
| VER-02 | 10-05 | guarded real-machine test (env-gated, never CI) mount → app-server → `session/create` → zero leftovers | ✓ SATISFIED | Post-merge re-run PASS (4.13s), session/create ok, both groups ESRCH, mount/base gone; read-only proof 0 added entries |
| VER-03 | 10-06 | docs coherent: managed mount, revised volatile rule, README, provenance; no contradiction | ✓ SATISFIED | D6; check-docs exit 0 |

All 13 requirements are satisfied. No requirement is BLOCKED or NEEDS HUMAN.

### Anti-Patterns Found

Re-verification evidence gate applied (files changed since the prior `verified:` timestamp — `main.go`/`main_test.go` — were treated as potential regressions). No new-scope blocker was found.

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| — | — | `TBD`/`FIXME`/`XXX` debt markers | ℹ️ None | The only `XXX` hits are literal `mktemp .../XXXXXX` templates in tests/scripts, not debt markers. No unreferenced debt marker exists. |
| — | — | Stub returns / hardcoded empty props on the phase path | ℹ️ None | No `return null`-style stubs; mount/tree data comes from validated runtime paths. |
| `mount.go` | 106 | `stepTeardown` constant declared but unused (review IN-01) | ℹ️ Info | Teardown errors still name the mount + remediation, but bypass the four-part `failure()` shape. Non-blocking per review disposition. |
| `discover.go` | 240-248 | Arch not part of candidate evaluation (IN-05) | ℹ️ Info | A same-version arm64 AppImage could win the tie-break on x64. Deterministic per DISC-04; non-blocking per review. |
| `scripts/vm-e2e.sh` | 324-347 | FUSE verdicts key on packed-file presence, not the ranked winner (IN-02) | ℹ️ Info | Mirror-only; no production effect. |
| `main.go` | 1203-1219 | Signal-vs-child-exit select race (IN-03), double stop log (IN-06), PID-reuse window (IN-07) | ℹ️ Info | Narrow diagnostics/race findings recorded open in `10-REVIEW-DISPOSITION.md`; none violates a must-have. |
| `main.go` | 538-570 | Text arm's packed mount path has no dedicated binary test | ℹ️ Info | New observation for the merged composition: `runOmpText` shares `startBackend`/`backend.stop` and maps a teardown error to exit 1, but only the ACP arm has the packed e2e. No must-have requires it; the shared invariants are exercised by A7-A13. |

Review disposition accounting: CR-01 and WR-01 are `fixed` (original commits `9bd3582`, `2d4ab12`; their content is present in HEAD via the squash and covered by the passing RED/GREEN tests — `MountFailure`/`TestMountFailure`, `defaultDpkgQuery`/`TestDefaultDpkgQuery`); IN-01..IN-07 are `open` Info advisories, explicitly non-blocking for this phase. 🛑 Blockers: 0. ⚠️ Warnings: 0.

### Advisory (New Scope, Unevidenced)

None — no new-scope Step 7 finding lacked deterministic evidence. The only new observation (text-arm mount test coverage) is a test-coverage note with no failing test or reproducible defect; it does not revert a completed must-have.

### Gaps Summary

No implementation gap was found. The mount owner, the no-extraction guarantee, version-ranked discovery, per-candidate misses, deterministic tie-break, override routing, mount-loss handling, signal teardown, and the CR-01/WR-01 fixes are all present, wired, and exercised by passing behavioral tests on the merged codebase (fake-runtime tier + binary e2e + unit matrices + post-merge real-machine acceptance). All 23 declared artifacts pass the artifact query and all 18 key links are wired; the full suite, lint, vet, tidy-diff, docs gate, and experiments suite pass, and CI is green on both phase PRs.

The merge-specific composition — `startBackend` owning mount acquisition, the detached spawn context, `backend.stop`'s supervisor → spawn → mount order, and `classifyRunExit`'s teardown-error precedence — is verified against the code and exercised by the passing binary signal/loss/teardown tests. The two truths previously pending (D2/VER-02 and D3) were re-proven by a fresh real-machine run on this host. All 30 must-haves are verified. A live prompt on ZCode 4.0.0 remains explicitly out of scope (D-14, DISC-V2-01) and is documented as such — it is not a gap in this phase.

No structured `gaps:` section: status is `passed`.

---

_Verified: 2026-11-24T05:51:05Z_
_Verifier: Claude (gsd-verifier)_
