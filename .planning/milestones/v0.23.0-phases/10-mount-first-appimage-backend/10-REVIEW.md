---
phase: 10-mount-first-appimage-backend
reviewed: 2026-11-24T00:54:23Z
depth: deep
files_reviewed: 17
files_reviewed_list:
  - README.md
  - docs/protocol/install-discovery.md
  - docs/provenance.md
  - cmd/zcode-acp/main.go
  - cmd/zcode-acp/main_test.go
  - internal/supervisor/mount.go
  - internal/supervisor/mount_test.go
  - internal/supervisor/mount_internal_test.go
  - internal/supervisor/discover.go
  - internal/supervisor/discover_test.go
  - internal/supervisor/spawn.go
  - internal/supervisor/e2e_test.go
  - internal/supervisor/appimage_realdesktop_test.go
  - internal/supervisor/realdesktop_test.go
  - test/realdesktop/apicycle_test.go
  - test/realdesktop/codingplan_test.go
  - scripts/vm-e2e.sh
findings:
  critical: 1
  warning: 1
  info: 7
  total: 9
  necessity: 0
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-11-24T00:54:23Z
**Depth:** deep
**Files Reviewed:** 17
**Status:** issues_found

## Summary

Reviewed the mount-first AppImage backend end to end: the mount owner (`mount.go`), the version-ranked discovery stage and override mapper (`discover.go`), the spawn seams added for the helper (`spawn.go`), the main lifecycle/signal wiring (`main.go`), the CI fake-runtime tiers (`mount_test.go`, `main_test.go`, `discover_test.go`), the guarded real-machine tiers, the coherence docs, and the vm-e2e mirror.

The main paths match the phase decisions well: the helper env is an allowlist by construction; the mount line is validated to one non-symlink `.mount_*` component under a private 0700 base; the tree is validated before adoption; teardown is staged and the mount is verified gone before the base is removed; discovery is a pure read with per-candidate misses and a total, deterministic ranking; the packed override routes through the same mount owner; the app-server is stopped before the helper; and the docs describe the shipped behavior (no extraction, read-only mount, FUSE failures loud). The fake-runtime tests genuinely exercise the production exec/readiness/validation/teardown code.

Verification performed while reviewing: `go build ./...`, `go vet ./...` (also with `-tags realdesktop`), `golangci-lint run` (0 issues), `go test -race -count=1 ./internal/supervisor/` (including a `-count=3` mount stress), the new binary-tier mount tests, and `experiments/check-docs.sh` all pass. Two claims were verified empirically: `dpkg-query -W -f='${Status}'` output shape (see WR-01) and Go's `os/exec` duplicate-env semantics (later values win, so the test helpers' appended overrides do work — checked, not reported). One defect is significant: a signal during mount acquisition discards a failed mount teardown and exits 0 (CR-01). All other findings are diagnostics, test-mirror, or narrow-race issues.

Real-machine cleanup assumptions were traced too: `appimage_realdesktop_test.go` runs one ordered `sync.Once` teardown (supervisor shutdown, then `Mount.Stop`, then both-group ESRCH) with non-fatal assertions, which is sound for body failures; the shared `discoverRealInstall` mount cleanup is registered early so it runs last, and a fatal assertion in the manual-tier bodies is covered by the body's deferred `spawnCancel` killing the app-server group before `Mount.Stop`. No leak was found there, but note that those bodies never call `supCancel` on a failure (the supervisor's graceful staged shutdown is skipped).

No `<structural_findings>` block was supplied, so there is no structural section. The necessity rubric was read and applied: no necessity findings (`necessity: 0`).

## Critical Issues

### CR-01: Signal during mount acquisition discards a failed mount teardown (exit 0, stale mount / orphan helper)

**File:** `cmd/zcode-acp/main.go:707-716` (with `internal/supervisor/mount.go:246-255`)
**Issue:** `PrepareMount` deliberately joins the teardown error into the acquisition error so "a live mount is never swallowed by the acquisition error" (`mount.go:246-255`). `run` then discards that error whenever the signal context is cancelled:

```go
inst, mount, err := acquireMount(signalCtx, inst)
if err != nil {
    if signalCtx.Err() != nil {
        slog.Info("zcode-acp: signal received during mount acquisition; shutting down")
        return nil        // err (including errors.Join(startErr, stopErr)) is dropped, not even logged
    }
    return err
}
```

If SIGINT/SIGTERM/SIGHUP arrives while `PrepareMount` is waiting for the mount line and `Stop` cannot prove the mount gone (helper not reaped within `ReapDeadline`, mount still present at `mount.go:590-596` / `599-650`), the wrapper exits 0: the 0700 base is retained by design (`mount.go:540-546`), the helper's process group may still be alive, and the teardown error plus its `fusermount -u` remediation are thrown away. This violates MNT-03 ("torn down after it on every exit path (normal, error, signal, client disconnect) — zero orphan processes and zero stale mounts"), D-02/A5, and the `10-SECURITY` mitigation text for T-10-04/T-10-09 (which cites the exit-1 teardown rule at `main.go:747-759,791-797`). The same branch also masks a genuine mount failure that races the signal (e.g. missing FUSE plus a Ctrl-C): the actionable FUSE diagnostic is lost and the exit is 0. No test covers signal + failing acquisition teardown — `e2eSignalDuringReadiness` only covers a Stop that succeeds, and `e2eTeardownFailureBeatsSignal` covers the post-spawn path.

**Fix:** Treat cancellation as clean only when nothing else failed; export the mount failure sentinels (or a predicate) and return/exit 1 for structural or teardown causes:

```go
inst, mount, err := acquireMount(signalCtx, inst)
if err != nil {
    if signalCtx.Err() != nil && !supervisor.MountFailure(err) {
        slog.Info("zcode-acp: signal received during mount acquisition; shutting down")
        return nil
    }
    return fmt.Errorf("mount acquisition: %w", err) // never silent: logs the remediation and exits 1
}
```

where `MountFailure(err)` reports `errors.Is(err, ErrMountSpawn|ErrMountReady|ErrMountTree|ErrMountTeardown)` (a pure `context.Canceled` readiness failure still exits 0). Add a binary-tier test: signal during readiness against a helper that ignores TERM + keeps its tree → exit 1 naming the mount and the manual-unmount remediation.

## Warnings

### WR-01: dpkg probe can never detect an installed package (status match is off by a leading space)

**File:** `internal/supervisor/discover.go:1191`
**Issue:** `defaultDpkgQuery` runs `dpkg-query -W -f='${Status}' zcode`, which prints exactly `install ok installed` — no leading space (verified on this Ubuntu 24.04 host: `dpkg-query -W -f='${Status}' bash` → `install ok installed`). The test `strings.Contains(string(state), " install ok installed")` can therefore never be true, so the probe always takes the rc-remnant arm, returns an empty file list, reports `rc remnant or absent (empty file list)` even for a fully installed deb, and never runs `dpkg -L`. The documented row-3 file-list check (and `probeDpkg`'s `slices.Contains(files, entrypoint)` validation at `:832-837`) is dead; a deb install is only ever found by the `opt-path` fallback. The code is pre-existing at `f34b7b4` but the phase rewrote this file and its diagnostics contract (`install-discovery.md:34` — the probe rows are the product of discovery) and the documented query form is `'${Version} ${Status}'`, which would have matched the needle.

**Fix:** Compare fields instead of a space-prefixed substring:

```go
fields := strings.Fields(string(state))
if len(fields) < 3 || fields[0] != "install" || fields[2] != "installed" {
    return "", nil // rc remnant or absent
}
```

(or restore the documented `${Version} ${Status}` format and keep the substring). Add a `defaultDpkgQuery` test with a fake `dpkg-query`/`dpkg` on PATH — the seam-based discovery tests cannot catch this because they stub the query away.

## Info

### IN-01: Teardown errors bypass the four-part mount diagnostics; `stepTeardown` is unused

**File:** `internal/supervisor/mount.go:106, 599-650, 711-714`; `cmd/zcode-acp/main.go:931-943`
**Issue:** The research rule (`10-RESEARCH.md:914`) and the README diagnose mount failures with the AppImage path, the failing step, the observed output, and a remediation — `Mount.failure` implements exactly that. Teardown errors are built directly from `errMountTeardown` in `stopHelper`/`waitMountGone`/`waitBaseMountsGone`, so they name the mount path/base and remediation but not the AppImage, carry no capped observed output, and the declared `stepTeardown` constant (`:106`) is never referenced. `main.stopMount` only prepends "stop the app-server AppImage mount". A user hitting the teardown-failure arm therefore gets less actionable text than an acquisition failure.
**Fix:** Build teardown errors through `m.failure(stepTeardown, cause)` (or add `m.appImage` + `observedOutput(m)` at those sites), and drop the now-redundant unused constant.

### IN-02: vm-e2e FUSE/libfuse2 verdicts key on packed-file presence, not the ranked winner

**File:** `scripts/vm-e2e.sh:319-353` (call at `:435`)
**Issue:** `check_packed_appimage_deps` hard-FAILs when `/dev/fuse` is absent or `libfuse.so.2` is not resolvable whenever *any* `ZCode*.AppImage` exists. The wrapper selects by version across layouts (D-06/D-08), so a stale packed AppImage beside a newer unpacked tree is never mounted — yet the mirror fails the discovery stage on a host where the wrapper would run fine. Likewise a host whose AppImage embeds the new static type-2 runtime needs no libfuse2, but the arm still FAILs (its detail text admits the mirror cannot classify the runtime without mounting). Because `check` records `fail` into the stage status, these are hard failures in the QA run.
**Fix:** mirror the version ranking (newest `{ver}` across packed files and `zcode-<ver>/squashfs-root`; unpacked wins ties) and only assert FUSE when the packed candidate would be selected; make the libfuse2 arm a `skip`/note unless a classic runtime is known, or state that the verdict is advisory.

### IN-03: Signal-vs-child-exit select race can report `errChildLost` (exit 1) for a signal-only shutdown

**File:** `cmd/zcode-acp/main.go:877-900`
**Issue:** A signal closes `signalCtx.Done()`, and `supCtx` is derived from it, so the supervisor starts the staged shutdown at the same instant. If `serveACP` is descheduled past the child's ~100 ms stdin-EOF drain, both `<-signals` and `<-child.Done` are ready and `select` may pick `childLost`; `finishRun` then returns `errChildLost` and the wrapper exits 1, although A11 decided a clean signal shutdown exits 0. The mount-loss reconciliation two lines below is the same pattern.
**Fix:** pass the signal context (or its `Err`) into `serveACP` and reconcile after the select:

```go
if outcome.childLost && signalCtx.Err() != nil {
    outcome.signalled = true
}
```

### IN-04: A second SIGINT/SIGTERM during the bounded teardown is swallowed (no force-quit)

**File:** `cmd/zcode-acp/main.go:688-690`
**Issue:** `signal.NotifyContext` keeps the signals registered until the deferred `stopSignals()`; after the first signal the internal goroutine has returned and the buffered channel is never drained, so pressing Ctrl-C again during the teardown (up to StopGrace + ReapDeadline + mount-gone polling, ~25 s worst case for the mount plus the supervisor's own grace) has no effect instead of terminating immediately.
**Fix:** call `stopSignals()` as soon as the first signal is observed (restores the default disposition), or add a second-signal handler that calls `os.Exit`/`killGroup` immediately.

### IN-05: Arch is not part of candidate evaluation; the documented tie-break can select a non-runnable AppImage

**File:** `internal/supervisor/discover.go:747-759, 1052-1069`
**Issue:** `appImageVersion` accepts any `<arch>` token (`ZCode-<ver>-linux-<anything>.AppImage`), and at equal `{ver}` the D-08 tie-break falls to `strings.Compare(display)`, so `ZCode-3.18.2-linux-arm64.AppImage` outranks `...-linux-x64.AppImage` on an x64 host (pinned by `tieBreakLexicographic`). A wrong-arch pick then fails loud at the mount (Exec format error) and, per D-12, never backtracks to the usable candidate that was right there. The documented filename shape carries the arch, so the information to filter already exists.
**Fix:** treat an arch that does not correspond to `runtime.GOARCH`'s AppImage spelling (`amd64`→`x64`, `arm64`→`arm64`) as a miss with an explicit candidate row, before ranking.

### IN-06: Every clean run logs "appimage mount stopped" twice

**File:** `cmd/zcode-acp/main.go:757, 832-836, 931-943`
**Issue:** `finishRun` calls `stopMount`, which logs the success line; `run`'s deferred `stopMountBackstop` calls `stopMount` again, `Mount.Stop` returns the cached `nil`, and the same success line is emitted a second time. If the first Stop failed, the backstop emits a second warning for the same cached error.
**Fix:** log only when this call performed the teardown (e.g. have `Mount.Stop` report whether it ran, or skip/quiet the deferred backstop once the normal path succeeded).

### IN-07: The "never signal an already-reaped helper" claim has a PID-reuse window

**File:** `internal/supervisor/mount.go:555-597`
**Issue:** `Done` is closed by `monitorChild` only after `cmd.Wait` has reaped the helper and delivered `Exited` (spawn.go:305-311). Between `Process.Wait()` releasing the PID and `close(done)`, `stopHelper`'s select can still see `Done` open and send SIGTERM (or the escalation SIGKILL) to a PGID that has already been released and may have been reused. The comment's "never an already-reaped one — PID-reuse safety" overstates the guarantee (the window is tiny, and the ESRCH tolerance covers the common arm).
**Fix:** document the residual window, or handshake with the monitor (e.g. a flag set immediately before `cmd.Wait` that `stopHelper` treats as "do not signal", set/checked under the same lock that guards the kill).

---

_Reviewed: 2026-11-24T00:54:23Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
