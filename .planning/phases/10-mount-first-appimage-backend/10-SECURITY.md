---
phase: "10"
slug: "mount-first-appimage-backend"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-11-24"
---

# Phase 10 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Register authored at plan time in the six plan `<threat_model>` blocks; verified against the
> implementation at `asvs_level: 1`, `block_on: high`.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| wrapper → AppImage runtime | The wrapper executes the user-selected/discovered AppImage as the mount helper; its stdout and exit status are untrusted. | mount path line, observed output (untrusted) |
| helper stdout → mount root | The mount path printed by the helper becomes the app-server root; it must be validated, never adopted verbatim. | filesystem path (untrusted) |
| wrapper temp filesystem | The private 0700 base and its `.mount_*` child are writable by the wrapper user only. | temp paths |
| mounted tree → app-server exec | Exactly two files are stat-validated and executed: the mount root's `zcode` and `resources/glm/zcode.cjs`. | executable path + argv |
| wrapper process → OS signals | SIGINT/SIGTERM/SIGHUP arrive from the user, terminal, or client host; children live in separate process groups. | signals |
| helper lifetime → mount lifetime | The helper can die or the mount vanish while the app-server still runs; a stale root must not be treated as valid. | process/FS liveness |
| filesystem scan → candidate list | `~/Applications` entries, symlinks, and the XDG target are untrusted path shapes. | filesystem paths (untrusted) |
| candidate list → Install | A candidate's resolved path becomes the AppImage executed later; it must be a regular executable, not a decoy or dangling link. | resolved path |
| CLI flag/env → install mapper | The user-supplied path shape decides between tree, binary, and packed mount acquisition. | user input (untrusted shape) |
| install mapper → executor | A misclassified `.AppImage` would be executed as the app-server binary (the GUI path), never serving stdio. | argv |
| mounted install → PAT mint | The client version sent to api.z.ai is derived from the install; a wrong value is wire-visible. | version string |
| diagnostics/logs | Paths, miss reasons, and observed output are user-controlled strings rendered into logs and errors. | paths, capped output |
| test host → user's desktop install | The real test executes the user's real AppImage and app-server; it must not modify either. | read-only mount, hashes |
| test teardown → host process table | A failed assertion must still reap the app-server and helper. | PIDs / mounts |
| test logs → repository | Manual-tier output may contain host paths; it must never contain credentials or capture secrets. | paths, presence booleans |
| documentation → user expectation | A doc sentence becomes a user contract; it must not promise extraction, patching, or behavior the code does not deliver. | prose |
| docs → provenance gate | V-EXP tags must resolve to real captures; V-SRC tags must carry a source argument. | provenance tags |
| vm-e2e script → host | The mirror runs read-only probes and must not mutate the desktop or install packages. | host reads |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-10-01 | Tampering | helper environment | high | mitigate | `helperEnv` builds PATH/HOME/TMPDIR only (`internal/supervisor/mount.go:777-794`); sole helper spawn `:483-486`; hostile-parent test `mount_internal_test.go:22-58` | closed |
| T-10-02 | Spoofing | helper stdout mount line | high | mitigate | Absolute, existing, non-symlink dir, `Dir == base/resolvedBase`, one `.mount_*` component (`mount.go:801-830`); both expected files validated (`:835-863`); only validated line adopted (`:946-1005`) | closed |
| T-10-03 | Elevation | private base and mount dir | high | mitigate | `os.MkdirTemp` 0700 (`mount.go:266`); NUL/LF/CR base rejected before spawn (`:272-277`, `:754-769`); scans only own base (`:630-678`); `RemoveAll` only after mount-proven-gone (`:541-546`) | closed |
| T-10-04 | Denial of service | orphaned helper/daemon and stale mounts | high | mitigate | Staged stop (`mount.go:520-549`); TERM → grace → group SIGKILL → reap with PID-reuse guard (`:555-597`); mount-root poll (`:606-650`); teardown failure exits 1 (`cmd/zcode-acp/main.go:747-759,791-797,133-137`) | closed |
| T-10-05 | Tampering | helper stdout/stderr log injection | medium | mitigate | 64 KiB caps (`mount.go:60-62`, `:881-907`); observed output rendered with `%q` (`:335-342`, `:468-473`); stderr tap cap (`internal/supervisor/spawn.go:318-351`) | closed |
| T-10-06 | Tampering | helper environment on failure paths | high | mitigate | Same single allowlist constructor used at the only helper spawn (`mount.go:777-794`, `:483-486`); no `os.Environ()` for the helper; no `APPIMAGE_EXTRACT_AND_RUN` set | closed |
| T-10-07 | Spoofing | failure-path stdout adoption | high | mitigate | Non-absolute lines are diagnostics; absolute lines validated against the base (`mount.go:977-1005`, `:801-830`); no adoption on exit/timeout (`:293-315`) | closed |
| T-10-08 | Denial of service | crash-loop respawn against vanished mount | high | mitigate | `Guard` before every `SpawnChild` (`cmd/zcode-acp/main.go:996-1003`); `ErrMountLost` (`mount.go:427-438`); run selects on `Lost()` (`main.go:872-892`) | closed |
| T-10-09 | Denial of service | orphaned children on SIGINT/SIGTERM/SIGHUP | high | mitigate | `signal.NotifyContext` INT/TERM/HUP (`main.go:688-690`); signal vs `WithoutCancel` spawn contexts (`:725-727`); app-server first, helper second (`:747-759`); exit 1 on teardown failure (`:791-827`, `:133-137`) | closed |
| T-10-10 | Information disclosure | failure diagnostics | medium | mitigate | Mount failures carry AppImage path, step, `%q`-capped observed text, remediation only (`mount.go:468-473`); startup logs paths/version/pid only (`main.go:921-923`) | closed |
| T-10-11 | Spoofing | symlink/path traversal in candidates | high | mitigate | Per-candidate `EvalSymlinks`, regular-file + executable-bit checks (`internal/supervisor/discover.go:609,624-641`); resolved-path dedupe (`:703-728`); failure is a miss (`:579-696`) | closed |
| T-10-12 | Tampering | volatile mount adopted as spawn root | high | mitigate | `mountDecoy` (`discover.go:228-234`); user override rejected (`:240-248`, `:344-361`); discovered candidates rejected (`:579-589,618-619,671-672`); managed mount created only after selection (`main.go:707,909-925`) | closed |
| T-10-13 | Denial of service | one bad candidate aborting discovery | medium | mitigate | Per-path loop (`discover.go:560-573`); miss candidate (`:579-590`); winner from hits only (`:763-771`); miss rows emitted (`:546-549`) | closed |
| T-10-14 | Tampering | TOCTOU replacement between discovery and exec | low | accept | Resolve once (`discover.go:297-340`) and exec the resolved path; mount is read-only; signature verification out of scope and never claimed (`docs/provenance.md:27-32`). See Accepted Risks Log | accepted |
| T-10-15 | Spoofing | `.AppImage` override treated as app-server binary | high | mitigate | Directory branch first (`discover.go:302-312`); literal and resolved `.AppImage` detection, absolutized (`:314-330`); `PrepareMount` owns acquisition (`main.go:909-925`); `TestOverrideInstall` arms (`discover_test.go:1206-1412`) | closed |
| T-10-16 | Tampering | shell/path injection through the override | high | mitigate | `exec.CommandContext(ctx, argv[0], argv[1:]...)` — no shell (`spawn.go:252-255`); helper argv exactly `{appImage, mountHelperFlag}` (`mount.go:483-486`); `%q` only in logs | closed |
| T-10-17 | Information disclosure | selection diagnostics | low | accept | Layout, version, root, AppImage/binary paths only (`main.go:921-923,976-978`); no env, key, or credential values. See Accepted Risks Log | accepted |
| T-10-18 | Tampering | real AppImage executed by test | high | mitigate | Mount read-only; sha256 before/after equality (`appimage_realdesktop_test.go:91,196-200`); no extraction entries added, pre-existing retained (`:203,354-385`); no prompt/turn in the real run (`:26-32`) | closed |
| T-10-19 | Denial of service | leftover app-server/helper/mount on test failure | high | mitigate | Single `sync.Once` teardown via `t.Cleanup` (`appimage_realdesktop_test.go:120-176,178`); staged supervisor shutdown first (`:129-142`); `Mount.Stop` then both-group ESRCH (`:154-171`); child ESRCH, mount-root ENOENT, base gone (`:192-194,390-406`); bounded non-fatal poll (`:411-432`) | closed |
| T-10-20 | Information disclosure | manual-tier diagnostics | medium | mitigate | Layout/version/root/binary/appimage + revision only (`test/realdesktop/apicycle_test.go:395-417`); scrub-before-commit reminder (`:645-647`); presence booleans, token material never recorded (`codingplan_test.go:583-629`); `//go:build realdesktop` + `ZCODE_ACP_REAL_E2E=1` guard | closed |
| T-10-21 | Tampering | docs claiming unimplemented behavior | medium | mitigate | Claims cross-checked against code: managed-mount/ranking/pure-read (`docs/protocol/install-discovery.md:26,36,45,113-125`); failure text (`README.md:38-52,107-125`) maps to `mount.go:90-107,468-473`; `check-docs.sh` exit 0; CI gate (`.github/workflows/pr-sanity.yml:91,137`) | closed |
| T-10-22 | Information disclosure | docs carrying user-home absolute paths | medium | mitigate | `experiments/check-docs.sh` path-shape scan passes; no `/home/` or `/Users/` in `docs/protocol/install-discovery.md`, `README.md`, `docs/provenance.md`, `scripts/vm-e2e.sh` — tilde/placeholder forms only | closed |
| T-10-23 | Repudiation | provenance contradiction | medium | mitigate | Never-patched (`docs/provenance.md:14-18`); read-only/no-extraction/preexisting-untouched (`:19-26`); no attestation/signature forgery (`:27-32`); FUSE requirement (`:92-97`); 4.0.0 `session/create` scope gap (`:98-104`) | closed |
| T-10-SC | Tampering | dependency installs (all plans) | high | mitigate | Single `require` (`go.mod:5`); `tidy-diff` assertion (`mise.toml:45-48`); no `go.mod`/`go.sum` change in `f34b7b4..HEAD`; `go mod tidy -diff` exit 0; no package installs in `scripts/vm-e2e.sh` | closed |

*Status: open · closed · open — below threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` (`high`) count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-10-01 | T-10-14 | TOCTOU replacement between discovery and exec is accepted for the user-controlled install path: the path is resolved once and that resolved path is mounted/executed; the mount is read-only; signature verification is explicitly out of scope and is never claimed in provenance (`docs/provenance.md:27-32`). Severity low — below the `high` block threshold. | Project owner (plan 10-03 threat model) | 2026-11-24 |
| AR-10-02 | T-10-17 | Selection diagnostics log layout, version, root, AppImage and binary paths only. These are not secret; no environment values, keys, or credentials are logged (`cmd/zcode-acp/main.go:921-923,976-978`). Severity low — below the `high` block threshold. | Project owner (plan 10-04 threat model) | 2026-11-24 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-11-24 | 24 | 22 (+2 accepted) | 0 | gsd-security-auditor (ASVS L1, mitigation verification) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-11-24
