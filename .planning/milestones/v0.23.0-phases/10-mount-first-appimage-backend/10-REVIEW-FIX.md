---
phase: 10-mount-first-appimage-backend
fixed_at: 2026-11-24T04:07:05+03:00
review_path: /home/nil/DiskD/W/Djarvur/zcode-acp/.planning/phases/10-mount-first-appimage-backend/10-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 10: Code Review Fix Report

**Fixed at:** 2026-11-24T04:07:05+03:00
**Source review:** `.planning/phases/10-mount-first-appimage-backend/10-REVIEW.md`
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (fix scope `critical_warning`: CR-01, WR-01)
- Fixed: 2
- Skipped: 0

| Finding | Severity | Status | Files modified | Commit |
|---|---|---|---|---|
| CR-01 | Critical | fixed | `internal/supervisor/mount.go`, `cmd/zcode-acp/main.go`, `cmd/zcode-acp/main_test.go`, `internal/supervisor/mount_internal_test.go` | `9bd3582` |
| WR-01 | Warning | fixed | `internal/supervisor/discover.go`, `internal/supervisor/discover_internal_test.go` (new) | `2d4ab12` |

## Fixed Issues

### CR-01: Signal during mount acquisition discards a failed mount teardown (exit 0, stale mount / orphan helper)

**Files modified:** `internal/supervisor/mount.go`, `cmd/zcode-acp/main.go`, `cmd/zcode-acp/main_test.go`, `internal/supervisor/mount_internal_test.go`
**Commit:** `9bd3582`
**Applied fix:**
- Added the exported `supervisor.MountFailure(err)` predicate (`internal/supervisor/mount.go`). It reports structural failures (`errMountInstall`, `errMountBase`, `errMountSpawn`, `errMountTree`, `errMountTeardown`) directly, and reports `errMountReady` only when the error does not wrap only `context.Canceled`/`context.DeadlineExceeded`. A cancelled readiness joined with a teardown failure is therefore a mount failure (`errors.Is` sees the teardown sentinel through `errors.Join`).
- `run` (`cmd/zcode-acp/main.go`) now treats a cancelled acquisition as a clean exit only when `signalCtx.Err() != nil && !supervisor.MountFailure(err)`; otherwise it returns `fmt.Errorf("mount acquisition: %w", err)`, so the joined readiness+teardown error (with the AppImage path, failing step, observed output, and remediation) reaches stderr and the process exits 1.
- `TestMountFailure` (`mount_internal_test.go`) pins the predicate semantics: pure readiness cancellation and deadline are false; every structural sentinel and a cancellation joined with a failed teardown are true.
- New binary-tier arm `TestAppImageMountSignal/failed teardown during mount readiness outranks the signal` with the `ignore-term` fixture: the helper ignores TERM and detaches a `setsid` descendant that keeps the inherited stderr pipe open, so the readiness-time teardown cannot reap the helper within `ReapDeadline`. The test asserts exit 1 plus stderr naming the AppImage, `read the mount line`, `remediation:`, and `teardown incomplete`, and no residual private base.

**Verification evidence:** RED with the unfixed `run` — the new arm failed with `wrapper exit = <nil>, want exit code 1` after ~15 s (teardown error silently swallowed). GREEN with the fix — the arm exits 1; the existing `signal during mount readiness` arm still exits 0 (pure cancellation), and `teardown failure outranks the signal` (post-spawn path) still exits 1.

**Logic note:** this is a behavioral control-flow change, verified by the new binary-tier RED/GREEN test and the predicate unit test; a human spot-check of the exit-precedence intent is worthwhile.

### WR-01: dpkg probe can never detect an installed package (status match is off by a leading space)

**Files modified:** `internal/supervisor/discover.go`, `internal/supervisor/discover_internal_test.go` (new)
**Commit:** `2d4ab12`
**Applied fix:**
- `defaultDpkgQuery` now compares status fields instead of a space-prefixed substring: `strings.Fields` of the real `dpkg-query -W -f='${Status}'` output (`install ok installed`, no leading space) must yield `fields[0] == "install"` and `fields[2] == "installed"`; anything else (rc remnant `deinstall ok config-files`, absent package) falls through to opt-path with the empty list as documented.
- Added `internal/supervisor/discover_internal_test.go` (in-package, because the seam is unexported): fake executable `dpkg-query`/`dpkg` scripts on `PATH` cover the seam end to end. The installed arm proves the exact no-leading-space status selects the `dpkg -L` file list; the rc-remnant arm proves the empty list is returned and `dpkg -L` is never executed (marker file assertion).

**Verification evidence:** RED against the old substring — `the installed-package file list "" must carry "/opt/ZCode/resources/glm/zcode.cjs"`. GREEN after the field comparison; the discovery suite remains green.

## Verification

All gates ran in the isolated review-fix worktree
`/home/nil/DiskD/W/Djarvur/zcode-acp/.claude/worktrees/rf-10-3675141-1795482029`
(branch `gsd-reviewfix/10-3675141`), not in the main checkout:

- `mise run lint` (`golangci-lint` 2.15.0) — 0 issues
- `mise run tidy-diff` — pass (`go.mod` unchanged, exactly one `require` line pinned to `github.com/coder/acp-go-sdk v0.20.0`)
- `go test -race -count=1 ./...` — pass (all packages, including the new binary-tier and seam tests)
- `bash experiments/check-docs.sh` — pass (exit 0)

No commits used `--no-verify`.

## Out of Scope — Untouched Info Findings

Per the requested scope (`critical_warning`), the Info findings remain in `10-REVIEW.md` untouched:

| Finding | Title | Status |
|---|---|---|
| IN-01 | Teardown errors bypass the four-part mount diagnostics; `stepTeardown` is unused | untouched |
| IN-02 | vm-e2e FUSE/libfuse2 verdicts key on packed-file presence, not the ranked winner | untouched |
| IN-03 | Signal-vs-child-exit select race can report `errChildLost` (exit 1) for a signal-only shutdown | untouched |
| IN-04 | A second SIGINT/SIGTERM during the bounded teardown is swallowed (no force-quit) | untouched |
| IN-05 | Arch is not part of candidate evaluation; the documented tie-break can select a non-runnable AppImage | untouched |
| IN-06 | Every clean run logs "appimage mount stopped" twice | untouched |
| IN-07 | The "never signal an already-reaped helper" claim has a PID-reuse window | untouched |

---

_Fixed: 2026-11-24T04:07:05+03:00_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
