---
phase: 10-mount-first-appimage-backend
plan: 06
subsystem: docs
tags: [appimage, fuse, mount, discovery, documentation, provenance, vm-e2e]

# Dependency graph
requires:
  - phase: 10-mount-first-appimage-backend
    provides: PrepareMount/Mount owner (10-01), failure classification and Lost watcher (10-02), version-ranked discovery (10-03), override routing and desktopVersion (10-04), guarded real-machine acceptance (10-05)
provides:
  - install-discovery managed-mount row, version ranking with the documented tie-break, revised volatile and extraction policy
  - README packed-AppImage override, read-only mount, FUSE/libfuse2 prerequisites, exact failure messages, no-extraction statement
  - provenance read-only/no-new-extraction/FUSE honesty statement and the honest 4.0.0 scope gap
  - vm-e2e mirror of the ranked discovery plus packed-AppImage FUSE dependency checks
affects: []

# Actuals (#2632) — same chars/4 scale as the plan's estimate.
actuals:
  tokens: 10615
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Docs mirror the shipped symbols: every mount/rank claim maps to PrepareMount, validateMountTree, classifyMountFailure, Guard/Lost, appImageWinner, sortAppCandidates, or compareVersions"
    - "Runtime facts tagged V-SRC at the embedded AppImageKit commit (effcebc); the 2026-11-23 field exercise is header metadata, never a hand-created V-EXP capture"
    - "vm-e2e stays a read-only mirror: presence checks for /dev/fuse and libfuse.so.2, skips when only an unpacked tree is present, no installs or mounts"

key-files:
  created: []
  modified:
    - docs/protocol/install-discovery.md
    - README.md
    - docs/provenance.md
    - scripts/vm-e2e.sh

key-decisions:
  - "Row 6 'Managed mount (wrapper-created)' is tagged [V-SRC AppImageKit src/runtime.c@effcebc]; the 2026-11-23 mount exercise is recorded in the Verified-against header as metadata so no capture is hand-created"
  - "The documented tie-break is explicit: newest {ver} wins; equal versions prefer the unpacked tree, then a packed symlink, then a packed regular file, then the lexicographic path; an unparseable version is a version-unknown miss"
  - "Both builtin-fallback sentences qualify tree layouts and state the post-PrepareMount packed read with the exact phrase 'bundled builtin is read from'; the packed arm claims no new capture"
  - "README keeps the single-static-binary claim scoped to the wrapper and presents FUSE/libfuse2 as packed-AppImage host prerequisites, with the runtime's exact failure text"
  - "docs/provenance.md adds the read-only/no-copy contract and the honest D-14 scope: real acceptance covers mount to session/create, not a live 4.0.0 prompt"
  - "vm-e2e asserts /dev/fuse and libfuse.so.2 only when a packed ZCode*.AppImage exists; an unpacked-only host records skips, and the checks never install or mount"

requirements-completed: [VER-03]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "install-discovery presents the wrapper-created managed mount as row 6, documents version ranking and the tie-break, keeps unmanaged /tmp/.mount_* do-not-rely, and removes extraction as the wrapper's route"
    requirement: VER-03
    verification:
      - kind: other
        ref: "bash experiments/check-docs.sh"
        status: pass
      - kind: other
        ref: "grep acceptance: managed-mount row >=1, V-SRC AppImageKit >=2, tie-break >=1, 'bundled builtin is read from' >=2, 'Extraction is the reproducible route' ==0"
        status: pass
      - kind: manual_procedural
        ref: "manual review: claims mapped to internal/supervisor/discover.go (appImageWinner/sortAppCandidates/compareVersions) and mount.go (PrepareMount/validateMountTree/classifyMountFailure)"
        status: pass
    human_judgment: false
  - id: D2
    description: "README documents the packed-AppImage override, the read-only mount, FUSE/libfuse2 prerequisites, the exact runtime failure messages, no extraction fallback, and the mid-session mount-loss cause"
    requirement: VER-03
    verification:
      - kind: other
        ref: "bash experiments/check-docs.sh"
        status: pass
      - kind: other
        ref: "grep acceptance: libfuse2t64 >=1, 'Cannot mount AppImage, please check your FUSE setup.' >=1, 'install root, app-server binary' >=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/provenance.md states the read-only mount, no new squashfs-root/zcode-<ver> copy, untouched pre-existing extracts, the FUSE host limitation, and the honest 4.0.0 session/create scope"
    requirement: VER-03
    verification:
      - kind: other
        ref: "bash experiments/check-docs.sh"
        status: pass
      - kind: other
        ref: "grep acceptance: read-only >=1, session/create >=1"
        status: pass
    human_judgment: false
  - id: D4
    description: "scripts/vm-e2e.sh mirrors the version-ranked discovery, reports every packed/unpacked candidate, and asserts the packed-AppImage FUSE prerequisites with pass/skip/fail statuses"
    requirement: VER-03
    verification:
      - kind: e2e
        ref: "bash -n scripts/vm-e2e.sh && bash scripts/vm-e2e.sh --stage discovery"
        status: pass
      - kind: integration
        ref: "mise run ci (build + vet + lint + test -race + test-experiments)"
        status: pass
      - kind: other
        ref: "mise run tidy-diff"
        status: pass
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-11-24
status: complete
---

# Phase 10 Plan 06: Documentation and vm-e2e Coherence Summary

**Mount-first documentation set: the managed mount is a first-class layout, AppImage discovery is documented as version-ranked with a deterministic tie-break, README/provenance state the read-only no-extraction contract and FUSE prerequisites, and vm-e2e mirrors it with FUSE dependency checks.**

## Performance

- **Duration:** 5 min (320 s)
- **Started:** 2026-11-24T00:21:02Z
- **Completed:** 2026-11-24T00:26:39Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

- `docs/protocol/install-discovery.md`: added row 6 `Managed mount (wrapper-created)` (read-only, lifecycle-owned, tagged V-SRC at the embedded AppImageKit commit `effcebc`), documented the AppImage-layout tie-break (newest `{ver}`; then unpacked tree, symlink, regular file, lexicographic path), scoped the volatile-mount rule to UNMANAGED mounts, replaced the extraction-as-route paragraph, rewrote the discovery order into one version-ranked stage, and added the pure-read/mount-after-selection guarantees.
- `README.md`: the static-binary claim is scoped to the wrapper; the packed-AppImage backend is documented as a read-only `--appimage-mount` with no extraction copy, FUSE (`/dev/fuse`, unprivileged mounts) and `libfuse2t64`/`libfuse2` prerequisites, the `--app-server` override accepting a packed AppImage, and the runtime's exact FUSE failure text plus the mid-session mount-loss cause.
- `docs/provenance.md`: the honesty statement now covers the mount path (read-only, no `squashfs-root`/`zcode-<ver>` copy, pre-existing extracts untouched and never deleted, no desktop-file modification); Known limitations add the FUSE host requirement and the D-14 scope gap (verified real acceptance is mount to `session/create`, not a live 4.0.0 prompt).
- `scripts/vm-e2e.sh`: the header and discovery stage mirror the version-ranked stage and report every packed candidate and unpacked tree; packed-AppImage hosts get `/dev/fuse` and `libfuse.so.2` checks with pass/skip/fail statuses and the `libfuse2t64`/`libfuse2` remediation — no installs, mounts, or new wrapper flags.

## Task Commits

Each task was committed atomically:

1. **Task 1: install-discovery managed mount, version ranking, revised volatile/extraction policy** - `96a56c2` (docs)
2. **Task 2: README and provenance packed-AppImage override, FUSE prerequisites, read-only/no-extraction contract** - `1d5a3d4` (docs)
3. **Task 3: vm-e2e ranked-discovery mirror and FUSE dependency checks** - `72f57f4` (test)

**Plan metadata:** the final `docs(10-06): complete ...` commit.

## Files Created/Modified

- `docs/protocol/install-discovery.md` - managed-mount layout row, version-ranking/tie-break semantics, unmanaged-vs-managed volatile policy, pure-read discovery order and guarantees, AppImageKit source rows
- `README.md` - packed-AppImage backend install notes, host prerequisites, override semantics, FUSE/mount troubleshooting
- `docs/provenance.md` - read-only/no-extraction mount contract, FUSE limitation, honest 4.0.0 scope
- `scripts/vm-e2e.sh` - version-ranked discovery mirror, all-candidate reporting, `/dev/fuse` and `libfuse2.so.2` checks

## Decisions Made

- Row 6 carries `[V-SRC AppImageKit src/runtime.c@effcebc]` plus the `src/notify.c@effcebc` failure-text row; the 2026-11-23 field exercise is recorded in the Verified-against header as metadata — no capture is hand-created (the scrub pipeline remains the only path into `experiments/captures/`).
- The tie-break is stated exactly as implemented (D-06/D-08), and an unparseable version is documented as a version-unknown miss (D-12), never a last-resort winner.
- Both builtin-fallback passages qualify TREE layouts and state the packed read as "bundled builtin is read from `<mount>/resources/config/provider/zcode-builtin.json` after PrepareMount"; the QA-02 capture remains the only fallback capture.
- README/Provenance present FUSE and libfuse2 as host prerequisites of the packed-AppImage backend while keeping the never-patched/never-forged/never-wrong claims intact.
- vm-e2e FUSE checks fail only when a packed AppImage is present (skip on unpacked-only hosts) and are read-only by construction.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- The `gsd_run` shell helper is not defined in the executor's shell; the GSD CLI was invoked directly as `node /home/nil/.zcode/gsd-core/bin/gsd-tools.cjs`, which is the resolver's underlying command. No plan work was affected.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- VER-03 is materially complete: docs, README, provenance, and the vm-e2e mirror are coherent with the shipped mount-first behavior, and every gate is green (`check-docs`, `test-experiments`, `mise run ci`, `mise run tidy-diff`).
- No blockers. Phase 10's remaining step is verification/close-out; the documented 4.0.0 prompt gap stays deferred to the protocol-drift milestone (D-14 / DISC-V2-01).

---
*Phase: 10-mount-first-appimage-backend*
*Completed: 2026-11-24*

## Self-Check: PASSED

- All four modified files and this SUMMARY exist on disk.
- All three task commits exist and are ancestors of HEAD (`96a56c2`, `1d5a3d4`, `72f57f4`).
