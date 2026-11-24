# Phase 10: Mount-First AppImage Backend - Context

**Gathered:** 2026-11-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 10 makes the ZCode app-server runnable **directly from a packed `.AppImage`**
through a managed `--appimage-mount` helper — no extraction, no `squashfs-root`
duplicate on disk — and makes discovery choose the **newest usable backend by
version** across layouts, so a stale unpacked research copy can never pin an older
desktop version.

Deliverables: mount-helper lifecycle + supervisor integration (helper and app-server
as supervised children, ordered teardown), `--app-server <path.AppImage>` /
`ZCODE_ACP_APP_SERVER` accepting a packed AppImage, discovery version ranking with
per-candidate failure tolerance, actionable fail-loud diagnostics, CI-safe tests over
a scripted fake AppImage runtime, one guarded real-machine test, and coherent docs
(`docs/protocol/install-discovery.md`, README, provenance note).

Out of scope: extraction fallback for FUSE-less hosts (v1 fails loud), protocol/wire
drift testing for ZCode 4.0.0, non-Linux platforms, any modification of the AppImage
or its contents.
</domain>

<decisions>
## Implementation Decisions

### Backend acquisition (user-locked)
- **D-01:** The app-server runs from a FUSE mount created by
  `<AppImage> --appimage-mount`. Extraction is **not** used for packed AppImages and
  no `squashfs-root`/`zcode-<ver>` copy is created by the wrapper.
- **D-02:** The mount helper is a **supervised child**: start it, read the mount
  path from its stdout, validate `<mount>/zcode` + `<mount>/resources/glm/zcode.cjs`,
  then spawn the app-server with that root. On every exit path (normal, protocol
  error, signal, client disconnect) stop the app-server first, then the helper, reap
  both, and verify the mount is gone. Zero orphans, zero stale mounts.
- **D-03:** **No silent extraction fallback.** Missing FUSE, helper exit before the
  mount line, readiness timeout, or an unspawnable tree inside the mount exits
  non-zero with an actionable message naming the AppImage path, the failing step, the
  observed output, and the remediation.
- **D-04:** `--app-server` / `ZCODE_ACP_APP_SERVER` accept both a packed `.AppImage`
  and an install root directory. A directory keeps today's invariant validation; a
  packed AppImage goes through the mount path. The precedence (flag > env > discovery)
  is unchanged.
- **D-05:** Mid-session mount loss (helper dies or mount disappears while the
  app-server is alive) ends the session with an explicit cause and kills the child —
  never respawn against a vanished root, never silently continue.

### Discovery ranking
- **D-06:** Discovery evaluates AppImage-layout candidates **by version** and selects
  the newest `<ver>`. A stale unpacked `~/Applications/zcode-<ver>/squashfs-root`
  must lose to a newer `ZCode-<ver>-linux-<arch>.AppImage`. Version extraction follows
  the documented formats (`install-discovery.md:33`): AppImage filename `<ver>`,
  unpacked parent dir `<ver>`; compare on `<ver>` only (existing `compareVersions`
  semantics).
- **D-07:** Candidate handling is **per-candidate**: a dangling symlink,
  `EvalSymlinks` failure, volatile `/tmp/.mount_*` path, or invalid tree is recorded
  as a miss and the probe continues with the next candidate. Today one bad first
  candidate aborts the whole probe — that is the defect this phase closes.
- **D-08:** Deterministic tie-break when versions are equal: prefer the existing
  spawnable unpacked tree (no mount needed at equal version); prefer the user-facing
  symlink over a versioned AppImage file when both are packed; then lexicographic
  path. The rule must be documented and covered by a test.
- **D-09:** Diagnostics must report the selected backend's **layout, version, and
  resolved path**, and the miss row for every skipped candidate.

### Scope & platform
- **D-10:** Linux/AppImage only. deb/rpm/opt paths and the existing discovery order
  for them stay as they are.
- **D-11:** Wire behavior is untouched: same spawn line, same env pair
  (`ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` + `ZCODE_PERSONAL_PROVIDER_CONFIG_FILE`),
  same JSONL dialect. Only the acquisition of the root changes.

### Planning decisions (resolved from research open questions)
- **D-12:** Discovery stays **pure-read**. "Newest usable" means the newest
  candidate that passes file-level checks only — resolvable path, regular
  executable file, parseable `<ver>`, not a volatile `/tmp/.mount_*` path. Mount
  readiness is validated at acquisition time, not during discovery. If the newest
  candidate later fails to mount, the wrapper fails loud with diagnostics; it never
  silently falls back to an older candidate.
- **D-13:** A desktop-entry target that is an AppImage joins the AppImage version
  ranking. When the entry resolves to a package tree (`/opt/ZCode`), the existing
  package-path behavior and precedence stay unchanged.
- **D-14:** Real-machine acceptance covers mount → app-server → `session/create` →
  teardown with zero leftover processes and mounts. A live prompt on ZCode 4.0.0 is
  explicitly out of scope (protocol-drift milestone); the gap is documented, not
  papered over.

### Claude's Discretion
- Exact package layout inside `internal/supervisor`, helper-process plumbing
  (readiness timeout value, stdout scanner, stderr capture), log field names, error
  wording, test seam names, and whether the helper is a separate type or a
  supervisor child variant.
</decisions>

<specifics>
## Specific Ideas / Field Evidence

Verified on this machine on 2026-11-23 (source: this session's terminal):

- `<AppImage> --appimage-mount` prints e.g. `/tmp/.mount_ZCode-iQniYT` and stays
  alive until SIGINT; the mount is `ro,nosuid,nodev,relatime` FUSE
  (`fuse.ZCode-4.0.0-linux-x64.AppImage`).
- `ELECTRON_RUN_AS_NODE=1 /tmp/.mount_ZCode-iQniYT/zcode -e 'console.log(...)'`
  answered `node-ok 24.21.0` — the Node-mode app-server binary is executable from
  the mount (`nosuid` does not block `ELECTRON_RUN_AS_NODE`).
- SIGINT to the helper unmounted cleanly.
- The machine currently has `~/Applications/ZCode-4.0.0-linux-x64.AppImage` (packed,
  current GUI), a stale research extract `~/Applications/zcode-3.18.2/squashfs-root`,
  and a **dangling** `~/Applications/ZCode.AppImage` symlink whose target
  (`ZCode-3.18.2-linux-x64.AppImage`) was deleted. This is the exact real-world
  fixture the phase must handle: dangling symlink skipped, 4.0.0 selected by version.
- The user explicitly rejected the extraction model and the four extraction-era
  rationalizations: mount-first is the product decision, not a fallback compromise.
</specifics>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

- `docs/protocol/install-discovery.md` — layout matrix (row 1 unpacked, row 2 packed
  AppImage, row 5 volatile mount), discovery order (:110-116), version formats (:33)
- `docs/protocol/install-discovery.md:42` — extraction is currently "the reproducible
  route"; this phase replaces that policy for packed AppImages with a managed mount
- `internal/supervisor/discover.go` — probe order (:118-141), packed-AppImage probe
  and extraction seam (:387-450), unpacked probe (:469-493),
  `defaultExtractAppImage` (:899-917), `ExplicitRootInstall` (:232-238)
- `internal/supervisor/supervise.go`, `internal/supervisor/spawn.go` — supervision,
  process-group shutdown, spawn line/env contract
- `cmd/zcode-acp/main.go:740-800` — `discoverInstall`, `overrideInstall`
- `docs/provenance.md` — honesty contract (no patching, no unverifiable claims)
- `README.md` — install/config surface that must document the AppImage override
- `.planning/REQUIREMENTS.md` — MNT-01..06, DISC-01..04, VER-01..03

</canonical_refs>

<deferred>
## Deferred Ideas

- Extraction fallback for hosts without FUSE — explicitly rejected for v1; loud
  failure with remediation is the contract (MNT-V2-01).
- Backend protocol-drift gate / version negotiation for major desktop upgrades —
  separate milestone (DISC-V2-01).
- macOS/Windows backend acquisition — out of the v1 platform scope.
- Bundling/repacking the AppImage — never (honesty contract).
</deferred>
