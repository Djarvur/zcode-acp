# Requirements: zcode-acp — v0.23.0 Mount-First AppImage Backend

**Defined:** 2026-11-23
**Core Value:** An ACP client can run a full turn end-to-end with ZCode — from
`initialize` to a streamed reply with tool-calls and permission requests — on a real
z.ai account (api-key or OAuth coding plan), including client-rendered choice dialogs.
**Milestone goal:** The wrapper runs the ZCode app-server from a packed `.AppImage` by
managing an `--appimage-mount` helper — zero extraction copies — and discovery always
selects the newest usable backend across layouts.

## v1 Requirements

### Mount backend (MNT)

- [ ] **MNT-01**: The wrapper starts the ZCode app-server from a packed AppImage through a managed `--appimage-mount` helper, with no extraction and no on-disk duplicate
- [x] **MNT-02**: `--app-server <path.AppImage>` and `ZCODE_ACP_APP_SERVER` accept a packed AppImage; the wrapper mounts it, validates `<mount>/zcode` and `<mount>/resources/glm/zcode.cjs`, and spawns the app-server from that mount
- [x] **MNT-03**: The mount helper is a supervised child started before the app-server and torn down after it on every exit path (normal, error, signal, client disconnect) — zero orphan processes and zero stale mounts
- [x] **MNT-04**: Mount failures (missing FUSE, helper exit before the mount line, readiness timeout, unspawnable tree in the mount) fail loud with a non-zero exit and an actionable message naming the AppImage, the failing step, and remediation; never a silent extraction fallback
- [x] **MNT-05**: If the mount helper dies or the mount disappears while the app-server is alive, the supervisor terminates the session with an explicit cause instead of respawning against a vanished root
- [x] **MNT-06**: A mount-backed run creates no `squashfs-root`/`zcode-<ver>` directory and does not modify the AppImage

### Discovery (DISC)

- [x] **DISC-01**: Discovery selects the newest usable backend by version across layouts; a stale unpacked `squashfs-root` never beats a newer packed `.AppImage`
- [x] **DISC-02**: AppImage candidates are tried one by one: a dangling symlink, unresolvable path, volatile mount, or invalid tree is recorded as a miss and the probe continues with the next candidate
- [x] **DISC-03**: Discovery diagnostics report the selected backend's layout, version, and resolved path; version comes from the AppImage filename or the unpacked dir name
- [x] **DISC-04**: Selection is deterministic for a given filesystem state, with documented version ordering and tie-break (symlink before versioned file, then lexicographic)

### Verification & docs (VER)

- [x] **VER-01**: CI-safe unit tests drive mount lifecycle via a scripted fake AppImage runtime (mount line, readiness, teardown, early death, missing FUSE) without requiring FUSE
- [ ] **VER-02**: A guarded real-machine test (env-gated, never CI) mounts a real AppImage, spawns the app-server, performs `session/create`, and shuts down with zero leftovers
- [ ] **VER-03**: Docs updated coherently: `docs/protocol/install-discovery.md` (managed mount is a first-class layout; volatile-mount rule revised), README (AppImage override and behavior), and the provenance note; no documented claim contradicts behavior

## v2 Requirements

Deferred to a future milestone. Tracked but not in the v0.23.0 roadmap.

### Fallback & drift

- **MNT-V2-01**: Extraction fallback for hosts without FUSE (v1 fails loud instead)
- **DISC-V2-01**: Backend version negotiation / protocol-drift gate for major desktop upgrades (4.0.0+)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Automatic extraction when FUSE is unavailable | Duplicates ~850 MB on disk; v1 fails loud with remediation instead |
| macOS/Windows AppImage mounting | AppImage is Linux-only; outside the v1 platform scope |
| Bundling or repacking the AppImage | The wrapper never patches the desktop (honesty contract, `docs/provenance.md`) |
| Protocol coverage for ZCode 4.0.0 wire drift | Separate milestone; this one only needs the mount path to be version-agnostic |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| MNT-01 | Phase 10 | Pending |
| MNT-02 | Phase 10 | Complete |
| MNT-03 | Phase 10 | Complete |
| MNT-04 | Phase 10 | Complete |
| MNT-05 | Phase 10 | Complete |
| MNT-06 | Phase 10 | Complete |
| DISC-01 | Phase 10 | Complete |
| DISC-02 | Phase 10 | Complete |
| DISC-03 | Phase 10 | Complete |
| DISC-04 | Phase 10 | Complete |
| VER-01 | Phase 10 | Complete |
| VER-02 | Phase 10 | Pending |
| VER-03 | Phase 10 | Pending |

**Coverage:**
- v1 requirements: 13 total
- Mapped to phases: 13
- Unmapped: 0

---
*Requirements defined: 2026-11-23*
*Last updated: 2026-11-23 after milestone v0.23.0 requirements*
