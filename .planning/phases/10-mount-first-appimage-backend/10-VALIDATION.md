---
phase: "10"
slug: "mount-first-appimage-backend"
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-24"
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Source: `10-RESEARCH.md` §Validation Architecture; requirement IDs from
> `.planning/REQUIREMENTS.md`; task IDs from the six plans.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib), `-race`; scripted fake AppImage runtime; existing fake-app-server binary and reexec helpers |
| **Config file** | `mise.toml` (tool pins) + `.golangci.yml`; no separate test config |
| **Quick run command** | `go test -race -count=1 ./internal/supervisor/ -run 'TestMount\|TestDiscover'` |
| **Full suite command** | `mise run test` (`go test -race -count=1 ./...`) |
| **Estimated runtime** | ~60–120 s for the full Go suite on this host (CI runner class) |

---

## Sampling Rate

- **After every task commit:** `go test -race -count=1 ./internal/supervisor/` (or the narrow package for the touched area; `./cmd/zcode-acp/` for binary-level tasks).
- **After every plan wave:** `mise run test` (full Go suite with race).
- **Before `/gsd:verify-work`:** `mise run ci` green; `bash experiments/check-docs.sh`; `mise run tidy-diff` clean; one guarded real-machine run recorded.
- **Max feedback latency:** ~120 s (full suite); quick package runs < 30 s.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 10-01-01 | 10-01 | 1 | MNT-01, MNT-02, MNT-03, MNT-06, VER-01 | T-10-03 | Private base validated; helper env allowlisted; mount path trusted only after tree validation | integration (fake runtime) + guarded real | `go test -race -count=1 ./cmd/zcode-acp/ -run TestAppImageMountEndToEnd -v`<br>`go test -race -count=1 ./internal/supervisor/ -run TestMountHappyPath -v` | ❌ W0 | ⬜ pending |
| 10-01-02 | 10-01 | 1 | MNT-03, MNT-06, VER-01 | T-10-03, T-10-19 | Staged stop; no extraction; helper reaped; newline TMPDIR rejected and spaced TMPDIR accepted whole | unit | `go test -race -count=1 ./internal/supervisor/ -run 'TestMountEnv\|TestMountStop\|TestMountSpacedTmpdir\|TestMountNoExtraction\|TestMountRejectBadTmpdir' -v` | ❌ W0 | ⬜ pending |
| 10-02-01 | 10-02 | 2 | MNT-04, VER-01 | T-10-03 | Early exit / timeout / no FUSE / invalid tree fail loud with observed output; no fallback; teardown failure exits 1 | table unit + binary exit-code | `go test -race -count=1 ./internal/supervisor/ -run 'TestMountFailures\|TestMountTeardownFailure' -v`<br>`go test -race -count=1 ./cmd/zcode-acp/ -run TestAppImageMountFailures -v` | ❌ W0 | ⬜ pending |
| 10-02-02 | 10-02 | 2 | MNT-03, MNT-05, VER-01 | T-10-08, T-10-09 | Loss ends the session with cause; SIGINT/SIGTERM/SIGHUP run staged shutdown; a teardown error beats signal classification (exit 1, never a masked 0); signal during discovery exits 0 | unit + binary | `go test -race -count=1 ./internal/supervisor/ -run 'TestMountLost\|TestMountGuard' -v`<br>`go test -race -count=1 ./cmd/zcode-acp/ -run 'TestAppImageMountSignal\|TestAppImageMountLost' -v` | ❌ W0 | ⬜ pending |
| 10-03-01 | 10-03 | 2 | DISC-01, DISC-02, DISC-04 | — | Pure-read ranking; per-candidate misses; deterministic tie-break | unit | `go test -race -count=1 ./internal/supervisor/ -run 'TestDiscoverRank\|TestDiscoverPerCandidate\|TestDiscoverTieBreak\|TestDiscoverUnpackedOnly\|TestDiscoverVersionUnknown\|TestDiscoverPureRead' -v` | ❌ W0 | ⬜ pending |
| 10-03-02 | 10-03 | 2 | DISC-03, MNT-06 | — | Layout/version/path diagnostics; XDG seed; builtin row arms; extraction seam absent | unit | `go test -race -count=1 ./internal/supervisor/ -run 'TestDiscoverDiagnostics\|TestDiscoverXDG\|TestDiscoverBuiltinDeferred\|TestDiscoverProbeOrder\|TestDiscoverNoInstall\|TestDiscoverRejectsVolatileMounts' -v`<br>`go test -race -count=1 ./internal/supervisor/ -run TestDiscover -v` | ❌ W0 | ⬜ pending |
| 10-04-01 | 10-04 | 3 | MNT-02, MNT-04 | T-10-03 | Packed override mounts; literal path absolutized (never PATH-resolved); PAT version seam mount-safe; desktop version and wiring seams green | unit + binary | `go test -race -count=1 ./internal/supervisor/ -run TestOverrideInstall -v`<br>`go test -race -count=1 ./cmd/zcode-acp/ -run 'TestDesktopVersion\|TestWiringCodingPlanSeams' -v` | ❌ W0 | ⬜ pending |
| 10-04-02 | 10-04 | 3 | MNT-02, DISC-03 | T-10-03 | Env/flag precedence unchanged; selection diagnostics; directory override unchanged; no extraction | binary | `go test -race -count=1 ./cmd/zcode-acp/ -run 'TestAppImageOverride\|TestRootModeUnchanged' -v` | ❌ W0 | ⬜ pending |
| 10-05-01 | 10-05 | 4 | VER-02, MNT-01, MNT-03, MNT-06 | T-10-19 | Real mount → app-server → session/create → zero leftovers; no NEW extract entry; shared helpers preserved and the legacy real path retired | compile-only (manual real run in Manual-Only) | `go test -tags realdesktop -run '^$' -count=1 ./internal/supervisor/` | ❌ W0 | ⬜ pending |
| 10-05-02 | 10-05 | 4 | VER-02 | — | Manual-tier mirror stays coherent with production override routing | compile/manual | `go test -tags realdesktop -run '^$' -count=1 ./test/realdesktop/ ./internal/supervisor/` | ✅ | ⬜ pending |
| 10-06-01 | 10-06 | 4 | VER-03 (documents DISC-03, DISC-04) | — | Managed mount documented as first-class; volatile rule revised; version ranking and the packed builtin read documented | static doc gate + review | `bash experiments/check-docs.sh`<br>`grep -c 'Managed mount (wrapper-created)' docs/protocol/install-discovery.md` | ✅ | ⬜ pending |
| 10-06-02 | 10-06 | 4 | VER-03 (documents MNT-01, MNT-06) | — | README/provenance scope: wrapper creates no extract; pre-existing extracts untouched; FUSE prerequisites stated | static doc gate + review | `bash experiments/check-docs.sh`<br>`grep -c 'libfuse2t64' README.md`<br>`grep -c 'read-only' docs/provenance.md` | ✅ | ⬜ pending |
| 10-06-03 | 10-06 | 4 | VER-03 (documents MNT-04) | — | vm-e2e asserts FUSE/libfuse2; CI + tidy-diff gates green | static + CI | `bash -n scripts/vm-e2e.sh && bash experiments/check-docs.sh && mise run test-experiments`<br>`mise run ci`<br>`mise run tidy-diff` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/supervisor/mount.go` + `mount_test.go` — mount owner and fake-runtime harness (MNT rows)
- [ ] `internal/supervisor/discover_test.go` — packed/ranking/tie-break/per-candidate tests; update `documentedProbeNames` / `assertProbeOrder`
- [ ] `cmd/zcode-acp/main_test.go` — binary-level fake-AppImage e2e (happy path, failure exit codes, signal shutdown, no `squashfs-root`, AppImage hash unchanged)
- [ ] `internal/supervisor/realdesktop_test.go` (or `appimage_realdesktop_test.go`) — guarded real mount + `session/create`
- [ ] `test/realdesktop/apicycle_test.go` — mirror mount-backed override routing (manual tier, must not rot)
- [ ] No framework install; no new dependencies

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real packed AppImage mounts, serves `session/create`, and leaves zero processes/mounts (task 10-05-01; the automated verify is compile-only, so the real run lives here) | VER-02 | Needs FUSE, a real AppImage, and the desktop runtime; CI must never contact the real backend | On a FUSE-capable host with a real `ZCode*.AppImage`: `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealAppImageMount ./internal/supervisor/`. Pin a specific file with `ZCODE_ACP_REAL_APPIMAGE` if needed. Record: before/after parent listing (no NEW `squashfs-root`/`zcode-<ver>` entry; pre-existing extracts allowed and never deleted), AppImage hash unchanged, `session/create` reply, `kill(-pgid,0)` ESRCH for both helper and app-server, and `mount` output without the managed mount. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120 s
- [ ] `nyquist_compliant: true` set in frontmatter (at validate-phase)

**Approval:** pending
