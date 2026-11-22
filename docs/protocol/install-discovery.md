# Install Discovery — locating a ZCode desktop install

**Domain:** desktop install layouts (unpacked AppImage / AppImage file / deb / rpm / running mount), URL-handler registration, and what auto-update does to the builtin revision string (DISC-03)
**Verified against:** read-only enumeration on a live Ubuntu 24.04 GNOME machine, 2026-11-19 — every matrix row is a recorded locate/inspect command in `[V-EXP EXP-DISCOVERY-01]`; the revision formula is additionally V-SRC-verified by grep into the shipped 3.18.2 bundle (ground truth per D-01)
**Confidence:** HIGH for the unpacked-AppImage, AppImage-file, deb, desktop-entry, runtime-tree, and decoy rows (all probe- or artifact-verified); MEDIUM for the rpm row (artifact exists and its version scheme is header-verified, but the full file list is generalized from the deb — no rpm tooling on the probe machine); V-RPT for live auto-update behavior (never directly observed end-to-end)

Provenance tags (V-SRC / V-EXP / V-RPT) are used per the legend in
[README.md](README.md). This document is the Phase 3 discovery input: the
supervisor (BRIDGE-03) implements locate + diagnostics directly from the matrix
below. Layout probe facts are experiment `EXP-DISCOVERY-01`; nothing in this
document was derived by installing any package (artifact inspection only — the
URL-handler clash documented in the Desktop entry section makes installation a
real regression risk).

## Layout matrix

Every row: a locate command (recorded with its verified output in the capture), the entrypoint-relative location it yields, its provenance, and its status.

| # | Layout | Locate command (pattern form) | Entrypoint / root | Provenance | Status |
|---|--------|-------------------------------|-------------------|------------|--------|
| 1 | Unpacked AppImage (squashfs-root extract) | `ls ~/Applications/zcode-*/squashfs-root/zcode` | `<bundle-root>/resources/glm/zcode.cjs`; Electron binary at `<bundle-root>/zcode` | [V-EXP EXP-DISCOVERY-01] | STABLE — this doc's ground-truth copy; pristine (`.orig` byte-compare identical, re-run in the capture) |
| 2 | AppImage file + symlink | `ls -la ~/Applications/ZCode*.AppImage` | versioned file `~/Applications/ZCode-<ver>-linux-x64.AppImage` + unversioned symlink `~/Applications/ZCode.AppImage` (relative target) | [V-EXP EXP-DISCOVERY-01] | STABLE — prefer the symlink as the user-facing entry, resolve with `readlink -f` |
| 3 | deb package | `dpkg-query -W -f='${Version} ${Status}' zcode`; `dpkg -L zcode`; files under `/opt/ZCode/` | `<root>/resources/glm/zcode.cjs` = `/opt/ZCode/resources/glm/zcode.cjs`; plus `/usr/bin/zcode` (update-alternatives) and `/usr/share/applications/zcode.desktop` | [V-EXP EXP-DISCOVERY-01] | STABLE where installed — a removed-but-not-purged install leaves an rc-state remnant with an EMPTY file list (see below); never install to discover |
| 4 | rpm package | HEAD `https://cdn-zcode.z.ai/zcode/electron/releases/<ver>/<platform>/ZCode-<ver>-<platform>.rpm`; `rpm -qlp <artifact>` | expected `<root>/resources/glm/zcode.cjs` mirroring the deb family | [V-EXP EXP-DISCOVERY-01] | DEB-GENERALIZED — version scheme `zcode-<ver>-<build>.<arch>` header-verified (`3.18.2-10443`/`x86_64`); full file list pending an rpm-tooling host (assumption) |
| 5 | Running mount | `mount \| grep -i ZCode` | `/tmp/.mount_ZCode-<random>/` — fuse, read-only, exists only while the app runs | [V-EXP EXP-DISCOVERY-01] | **do-not-rely** — volatile name, disappears on app exit; never a discovery target |

Layout facts behind the rows [V-EXP EXP-DISCOVERY-01]:

- **Unpacked AppImage** is a `--appimage-extract`-style squashfs extract at `~/Applications/zcode-<ver>/squashfs-root/`. Its `resources/glm/zcode.cjs` ships with a `.orig` sibling **on this copy only** — packages do not contain it (verified in the deb file list). A `cmp` against `.orig` is the pristine/honesty check: byte-identical, exit 0, recorded 2026-11-19.
- **deb** installs 3,951 files: 3,913 under `/opt/ZCode/` (Electron runtime incl. the `zcode` binary, `resources/app.asar`, `resources/app-update.yml`, `resources/glm/zcode.cjs`) plus `/usr/share/applications/zcode.desktop` and icons. Current artifact `3.18.2-10443` (version scheme `<ver>-<build>`, amd64, Installed-Size ~819 MB; depends: libgtk-3-0, libnotify4, libnss3, libxss1, libxtst6, xdg-utils, libatspi2.0-0, libuuid1, libsecret-1-0). Artifact provenance: inspected from a checksum-verified copy of the release `.deb` (sha512-verified against the updater's own `update-info.json`); official CDN URLs fetched live at execution time, never hardcoded. The dpkg database on the probe machine additionally holds an rc-state remnant (`3.17.2-9661`, status `deinstall ok config-files`, `dpkg -L zcode` → "does not contain any files") — proof that a deb install previously existed and was removed; an rc remnant contributes the version scheme but **no file paths**.
- **rpm** exists on the same CDN release path (200 OK, `application/x-rpm`, ~168 MB for 3.18.2 linux-x64; linux-arm64 also present). Same `<ver>-<build>` scheme as the deb, read from the artifact header. Its layout is expected to mirror the deb (`/opt/ZCode/` + `/usr/share/...`) because both come from the same electron-builder release family — recorded as an assumption, not an observation.
- **Version-string formats per layout** (so "the same version" has a stated definition across rows): deb/rpm `zcode <ver>-<build>` (e.g. `3.18.2-10443`; `<build>` is packaging-channel-specific); AppImage filename `ZCode-<ver>-linux-<arch>.AppImage` (`<ver>` only); runtime tree directories `<ver>` (`3.18.2`), plus a `0.0.0-dev` marker dir for dev builds; the desktop's own reported version is `<ver>`. Equality checks between layouts compare on the `<ver>` component; `<build>` and packaging suffixes are ignored.

**Empty result is a first-class outcome.** When no row matches (no unpacked dir, no AppImage under `~/Applications`, no dpkg/rpm entry, nothing at `/opt/ZCode/`), the discovery result is exactly that: an empty set of layouts. It must not be reported as an error or guessed away. Phase 3's supervisor (BRIDGE-03) is required to emit actionable startup diagnostics naming **every probe attempted** — each locate command from this matrix with its empty result — so the user can see precisely what was searched (which globs, which package databases, which paths) and add the missing layout deliberately.

## Locating the entrypoint per layout

The spawn contract needs `<root>/zcode` (the Electron binary) and `<root>/resources/glm/zcode.cjs` (the glm entrypoint). The relative path `resources/glm/zcode.cjs` under the root is invariant across all layouts [V-EXP EXP-DISCOVERY-01].

- **Unpacked AppImage** — direct: `<bundle-root>/zcode` + `<bundle-root>/resources/glm/zcode.cjs`. Both verified present (Electron binary is an ELF x86-64 PIE, ~228.6 MB; entrypoint ~15.7 MB). If a `zcode.cjs.orig` sibling exists, byte-compare before trusting the copy (pristine check, recorded in the capture).
- **AppImage file / symlink** — the AppImage is a squashfs image; the entrypoint is not directly readable while packed. Two supported routes: (a) resolve the symlink and extract once to a stable location: `<resolved>.AppImage --appimage-extract` → `squashfs-root/` with layout identical to row 1; (b) let the desktop mount itself (row 5) and read from the mount **for inspection only** — never record a mount path as the spawn root (do-not-rely). Extraction is the reproducible route.
- **deb** — fixed path: `/opt/ZCode/resources/glm/zcode.cjs` (binary `/opt/ZCode/zcode`, PATH command `/usr/bin/zcode`). When the package is installed, `dpkg -L zcode` lists the tree; when only an rc remnant exists the file list is empty — check `/opt/ZCode/` directly instead.
- **rpm** — expected `/opt/ZCode/resources/glm/zcode.cjs` mirroring the deb (deb-generalized; verify with `rpm -qlp` where rpm tooling exists).
- **Running mount** — no. `/tmp/.mount_ZCode-<random>/` is volatile and its name is not stable across launches [V-EXP EXP-DISCOVERY-01].

Canonical spawn line (Phase 3 consumes this verbatim):

```
ELECTRON_RUN_AS_NODE=1 \
ZCODE_BUILTIN_PROVIDER_CONFIG_FILE=<newest zcode-builtin.json under ~/.zcode/v2/runtime/provider/linux-x86_64/> \
<root>/zcode <root>/resources/glm/zcode.cjs app-server --stdio
```

## Desktop entry and URL handler

Both packaging layouts register the same desktop entry filename for the same URL scheme with **mutually exclusive Exec targets** [V-EXP EXP-DISCOVERY-01]:

| Source | Entry file | Exec | URL handler |
|--------|-----------|------|-------------|
| deb package (read from the artifact, single-member tar read — no extraction) | `/usr/share/applications/zcode.desktop` | `/opt/ZCode/zcode %U` | `x-scheme-handler/zcode;` |
| AppImage (self-written user-level entry) | `~/.local/share/applications/zcode.desktop` | `"<path-to>/ZCode-<ver>-linux-x64.AppImage" %U` | `x-scheme-handler/zcode;` |

**Clash warning (documented, artifact-confirmed).** Installing one layout after the other can silently repoint `zcode://` URL handling, because the xdg default is a single name (`zcode.desktop`) resolved through the xdg search order (user-level `~/.local/share/applications/` wins over `/usr/share/applications/`). The deb's `postinst` additionally runs `update-desktop-database` and registers `/usr/bin/zcode → /opt/ZCode/zcode` via `update-alternatives` — so a deb install also places a PATH-level command that an AppImage-only machine does not have. On the probe machine the live state demonstrates the AppImage-pointing entry holding the default: `xdg-mime query default x-scheme-handler/zcode` → `zcode.desktop` backed by the user-level AppImage entry (`mimeapps.list` row `x-scheme-handler/zcode=zcode.desktop`). Side-effect also observed: `text/html=zcode.desktop` in `mimeapps.list` — the desktop registered itself as default browser during an install; discovery tools should record such rows, never "fix" them.

Discovery procedure for Phase 3 [V-EXP EXP-DISCOVERY-01]:

1. `xdg-mime query default x-scheme-handler/zcode` → the entry name (or empty).
2. Resolve that name through the xdg entry search order (`~/.local/share/applications/`, then `/usr/share/applications/` and distribution dirs) and read its `Exec` line — the Exec target immediately identifies which layout owns URL handling (AppImage path vs `/opt/ZCode/zcode`).
3. Record `MimeApplication` rows mentioning zcode in `~/.config/mimeapps.list` (including any `text/html` hijack) as environment state.

## Auto-update and the builtin revision

**Updater configuration** (from the unpacked bundle, `resources/app-update.yml` — [V-SRC bundle `resources/app-update.yml`], contents confirmed in the capture [V-EXP EXP-DISCOVERY-01]): `provider: generic`, `useMultipleRangeRequest: false`, `url: http://localhost:8081`, `updaterCacheDirName: '@zcodedesktop-updater'`. The updater cache directory observed on disk is `~/.cache/@zcodedesktop-updater/` with a `pending/` subdir.

**Observed update mechanics** [V-EXP EXP-DISCOVERY-01]: the desktop updater downloads new-version artifacts into `~/.cache/@zcodedesktop-updater/pending/` together with an `update-info.json` sidecar recording `{fileName, sha512, isAdminRightsRequired}`. On the probe machine the pending cache held the current release `.deb` (`ZCode-3.18.2-linux-x64.deb`, sha512-verified) — i.e. on this machine updates arrive as deb artifacts even though the running install is the AppImage layout (the downloaded deb was never installed; the desktop kept running the AppImage). Port caution: `app-update.yml` names `http://localhost:8081` as the feed URL, while an unrelated local http-proxy process currently occupies 8081 — neither 8081 nor the desktop's dynamic localhost listener is a protocol surface (see Recommended discovery order).

**Runtime provider tree** [V-EXP EXP-DISCOVERY-01]: `~/.zcode/v2/runtime/provider/linux-x86_64/<ver>/endpoint-<32-hex>/`, containing `zcode-builtin.json` and `zcode-builtin-refresh.json`. The tree accretes one `<ver>` dir per desktop version seen — 12 dirs observed (0.0.0-dev, 3.12.3, 3.14.0/3/4/5, 3.15.1/3, 3.16.3, 3.17.1, 3.17.2, 3.18.2), all sharing the SAME endpoint dir name (`endpoint-78d7c3bef4024722642626fe3669a799`). Per-version `revision` values: 3.12.3=28, 3.14.x=30, 3.15.x=31, 3.16.3–3.17.2=43, 3.18.2=44 (schemaVersion 1 everywhere). Revision is monotone with recency but **not per-version unique** (3.16.3–3.17.2 share 43).

**The builtin revision formula** — V-SRC-verified in the shipped 3.18.2 bundle [V-SRC `resources/glm/zcode.cjs`#release-store `createHash`/`nys` literals, quoted in the capture]:

```
revision = zcode-builtin:<rev>:<sha256( path.resolve(activeFilePath) )>
```

Bundle literals (minified, as shipped): the release store computes `this.#n=(0,sxr.createHash)("sha256").update((0,AJ.resolve(this.#t)).digest("hex")` and freezes `revision:`zcode-builtin:${e.revision}:${t}``. The hash input is the **resolved absolute path string** of the active `zcode-builtin.json` — **not the file contents** (this sharpens the earlier shorthand that read as "sha256 of the builtin path"). Concrete computed example from the capture: for the 3.18.2 builtin (`~/.zcode/v2/runtime/provider/linux-x86_64/3.18.2/endpoint-78d7c3bef4024722642626fe3669a799/zcode-builtin.json`, revision 44) the path-string sha256 is `e9c0674a8011ad25b1d943fdefca27211deba43bdccf50a3423de83dc28a3cd1`, giving `zcode-builtin:44:e9c0674a8011ad25b1d943fdefca27211deba43bdccf50a3423de83dc28a3cd1`. The file-content sha256 (a different quantity, useful for drift detection) was also recorded: `9cb89e6afe575281998c9dc6c6ab17b1a0dbe79e37e25b3d7a41f2896b9aa39d`. WHICH `zcode-builtin.json` the runtime itself hashes is FIELD-established: the ACTIVE-cache copy — on this machine the runtime-tree file this very example hashes (the desktop host log's own registry trace carries its hash; the subsection below records the evidence and the superseded two-candidate hypothesis).

**Live-update behavior is V-RPT** [V-RPT inference from tree accretion + pending-artifact observation; no completed update event was directly observed]: a desktop version update is expected to (a) download its package artifact into the updater pending cache (observed), (b) add a new `<ver>` dir to the runtime provider tree, and (c) move the active builtin revision (observed historically as 28 → 30 → 31 → 43 → 44 across version dirs while the endpoint dir name stayed constant). What was never observed: the install/apply step itself. Consumers must treat the revision as **probe-time state** (shared-state caveat: a live desktop can update mid-probe — record the revision in every capture header at capture time), not as a stable machine property.

### The revision the runtime's registry carries — field truth: the active-cache path hash

The formula above hashes **one** file's resolved path — and which file the runtime itself hashes is FIELD-established on the shipped 3.18.2 desktop: the **ACTIVE-cache copy** — the same `~/.zcode/v2/runtime/provider/<platform>/<appVersion>/endpoint-<32-hex>/zcode-builtin.json` the runtime tree records and discovery reads into the spawn's `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE`. NOT the install's own bundled resources copy, and NOT a computed selection between the two.

**Evidence** [V-EXP EXP-CODINGPLAN-CYCLE-01] (2026-11-22, this machine, 3.18.2 unpacked AppImage):

- **The desktop GUI host's own registry trace** (the host log's provider-runtime line, `[provider-runtime][trace:Provider Registry 已就绪]`, dated 2026-11-15 — the desktop running its OWN session on this machine) carries `configRevision` `["zcode-builtin:44:e9c0674…", "<personal-hash>"]` — the ACTIVE-cache path hash (the computed example above). On the same machine a same-revision bundled resources copy exists (`<root>/resources/config/provider/zcode-builtin.json`, revision 44, different bytes, hash `zcode-builtin:44:39ab3262…`) — and the runtime did NOT carry that hash.
- **Attempts 1-2** pushed `basedOnZCodeBuiltinRevision` = the ACTIVE-cache string exactly; the ack matched the pushed pair (`providerCount: 5`, `status: "received"`) — the revision string was never the failure. The recorded read-back gap (create replies carrying only the builtin providers, never the `account:zai-*` models) was later root-caused to the probe never materializing the account provider: session/create + close alone fire no `session/setModel`, so no model-selection materialization runs, no `interaction/requestProviderRuntimeHeaders` ask fires, no PAT answer flows, and the account providers never publish.
- **Attempt 3** pushed the bundled-copy hash under the two-candidate tie→bundled hypothesis (below) — a string the runtime's own registry trace provably does not carry on this machine; the hypothesis is withdrawn.

**The two-candidate tie→bundled selection is SUPERSEDED BY FIELD EVIDENCE** (was: V-SRC `zcode-src 3.17.1 clone`, `selectReleaseCandidate` — "only a strictly-greater ACTIVE revision replaces the BUNDLED baseline; every tie, including equal revision with different bytes, falls to BUNDLED"): the shipped 3.18.2 runtime carried the ACTIVE-cache hash through exactly such a tie (both copies revision 44, different bytes). The clone stays corroborating tier for the candidate path SHAPES only (bundled `<root>/resources/config/provider/zcode-builtin.json`; active `~/.zcode/v2/runtime/provider/<platform>/<appVersion>/endpoint-<32-hex>/zcode-builtin.json`, the endpoint dir discovered by glob — never computed) and for the registry's exact-string match gate below — never for the selection outcome.

**The registry match is exact-string, and its mismatch arm is a silent drop** [V-SRC `packages/provider/src/registry-service.ts` (the refresh composition's match gate)]: when `account.basedOnZCodeBuiltinRevision !== config.zcodeBuiltinRevision`, the registry keeps serving its previous complete snapshot — the account update is schema-accepted (`status: received`) but registry-ignored ("Built-in 已变化但 Account 仍基于旧事实时，继续服务上一份完整 Registry"). The ack is therefore never evidence of application; a consumer that hashes any file but the active cache gets a schema-clean no-op.

**Consumer rule:** the wrapper's account pushes carry the raw discovery-recorded revision — `zcode-builtin:<rev>:<sha256( path.resolve(install.BuiltinFile) )>` over the ACTIVE-cache copy discovery records (the supervisor's `BuiltinRevision`). Never a computed two-candidate selection, never the bundled resources copy's hash.

## Recommended discovery order

Phase 3's supervisor (BRIDGE-03) resolves the install in this order; each step is a pure read, and the list doubles as the exact probe set that the nothing-found diagnostics must name:

1. **Env/flag override first** — an explicit root or `ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` from the wrapper's own configuration wins over any on-disk discovery.
2. **Desktop entry** — `xdg-mime query default x-scheme-handler/zcode` → resolve the entry's `Exec` target; it instantly distinguishes the deb layout (`/opt/ZCode/zcode`) from the AppImage layout (a path under `~/Applications`).
3. **AppImage paths** — `ls ~/Applications/ZCode*.AppImage` (file + symlink), then `~/Applications/zcode-*/squashfs-root/` for an unpacked copy; extract (`--appimage-extract`) only if a packed AppImage is all that exists.
4. **Package paths** — `/opt/ZCode/resources/glm/zcode.cjs`, `dpkg-query -W zcode` + `dpkg -L zcode` (empty list = rc remnant), `rpm -q zcode` / `rpm -ql` where applicable, `/usr/bin/zcode` (deb-only PATH command).
5. **Never volatile mounts** — `/tmp/.mount_*` is do-not-rely (row 5). **Never localhost listeners** — the dynamic-port `zcode-host-local-*` listener answers 404 and an unrelated http-proxy on 8081 answers 400 [V-EXP EXP-DISCOVERY-01]; the only desktop channel is the stdio spawn line above.

Operational guarantees (documented resolutions of the probe-review rows):

- **Idempotency** — every matrix row is a pure read (locate/inspect commands); running discovery twice yields the same result modulo the desktop's own update cycle. The only rows that can legitimately change under that cycle are the runtime provider tree, the active revision, and the updater pending cache; all other rows are stable machine state.
- **Concurrency / shared state** — discovery may run while the desktop is live: auto-update can swap the runtime tree and revision mid-probe (record the revision at probe time, per the capture-header discipline), and a running desktop holds a volatile mount that must not be recorded as the spawn root.
- **Empty result** — as stated in the Layout matrix: report an empty layout set with the full probe list in the diagnostics, never a guess or an error.

## Sources & provenance

| Tag / id | Origin |
|----------|--------|
| `[V-EXP EXP-DISCOVERY-01]` | `experiments/captures/EXP-DISCOVERY-01.jsonl` — all seven E4 rows (2026-11-19): unpacked-AppImage pristine check; AppImage file + symlink listing; dpkg remnant + checksum-verified deb artifact inspection (`dpkg-deb -I`/`-c`, maintainer scripts); rpm header peek + CDN URL liveness; both desktop entries + live xdg handler state; runtime provider tree + per-version revisions + revision-formula bundle literals + computed example; `app-update.yml` + updater pending cache; localhost decoys + running-mount observation. Read-only throughout — no package installed, no machine state mutated, zero quota. |
| `[V-SRC bundle resources/app-update.yml]` | Updater configuration read from the unpacked 3.18.2 bundle copy (pristine byte-compare recorded in the capture). |
| `[V-SRC resources/glm/zcode.cjs#release-store literals]` | Revision formula (`zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>`) — grep context over the shipped 3.18.2 bundle; literals quoted verbatim in the capture. |
| `[V-SRC zcode-src 3.17.1 clone: packages/provider-node/src/zcode-builtin-provider-config-source.ts]` | The two-candidate release selection (`selectReleaseCandidate`), the construction-time `sourceKey = sha256(resolve(activeFilePath))`, and `snapshotFromRelease` (`zcode-builtin:<release.revision>:<sourceKey>`). The SELECTION OUTCOME (tie falls to bundled) is SUPERSEDED by field evidence on the shipped 3.18.2 family — the runtime carried the active-cache hash through a same-revision tie ([V-EXP EXP-CODINGPLAN-CYCLE-01] + the host-log trace); the candidate path shapes remain corroborating tier (same-family clone, not the shipped bytes). |
| `[V-SRC zcode-src 3.17.1 clone: packages/provider-node/src/zcode-builtin-cache-paths.ts]` | Active-cache path shape `<environmentConfigRoot>/runtime/provider/<platform>/<appVersion>/endpoint-<32-hex>/zcode-builtin.json`; endpoint key = sha256 of the normalized endpoint origin truncated to 32 hex; platform = `<os>-<arch>` (`linux-x86_64`). |
| `[V-SRC zcode-src 3.17.1 clone: packages/provider/src/registry-service.ts]` | The registry's exact-string match gate (`basedOnZCodeBuiltinRevision !== config.zcodeBuiltinRevision`): the mismatch arm keeps serving the previous complete registry — schema-accepted, registry-ignored. |
| `[V-EXP EXP-CODINGPLAN-CYCLE-01]` | The real-run coding-plan cycle (2026-11-22, attempts 1-3): attempts 1-2 pushed `basedOnZCodeBuiltinRevision` hashed over the runtime-tree ACTIVE-cache copy (`zcode-builtin:44:e9c0674…`) — acked (`providerCount: 5`, `status: received`), the revision-matching arm of the protocol proven; the read-back gap root-caused to the probe never materializing the account provider (no setModel poke → no runtime-headers ask → no PAT answer), NOT to the revision string. The desktop GUI host's own provider-runtime trace (host log, 2026-11-15: "Provider Registry 已就绪", `configRevision ["zcode-builtin:44:e9c0674…", "<personal-hash>"]`) confirms the shipped 3.18.2 runtime's registry revision is the ACTIVE-cache path hash even with a same-revision bundled copy present — refuting attempt 3's tie→bundled hypothesis (the bundled hash `zcode-builtin:44:39ab3262…`, withdrawn before a real run; see `experiments/README.md` attempt ledger). Read-only throughout — zero quota spend beyond the recorded attempts. |
| `[V-RPT official install docs + CDN]` | Official install page and CDN artifact URLs fetched live at execution time (2026-11-19; AppImage/deb/rpm across linux-x64 and linux-arm64); URLs are recorded by pattern, never hardcoded into tooling. |
| `[V-RPT live-update event]` | The completed install/apply step of a desktop auto-update was never observed; expected runtime-tree effects are inferred from observed artifact downloads + tree accretion. |

Path note: this document records glob/tilde/`<placeholder>` patterns only — no user-specific absolute paths (enforced by `experiments/check-docs.sh`).

---
*Phase: 1 — Research & Protocol Documentation (DISC-03)*
*Written: 2026-11-19*
