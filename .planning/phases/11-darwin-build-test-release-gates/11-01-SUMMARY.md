---
phase: 11-darwin-build-test-release-gates
plan: 01
subsystem: infra
tags: [go, darwin, cross-compilation, syscall, statfs, mount]
requires:
  - phase: 10-appimage-mount-backend
    provides: Existing supervisor mount lifecycle and injected int64 Statfs seam
provides:
  - Portable defaultStatfs conversion at the existing MountOpts.Statfs seam
  - Direct production-converter success and wrapped-error tests
  - Passing CGO-free darwin/arm64 and darwin/amd64 build evidence
  - Passing native macOS supervisor mount regression evidence under the race detector
affects: [11-darwin-build-test-release-gates, 12-macos-desktop-discovery-launch]
actuals:
  tokens: 782
  tasks: 2
  commits: 2
plan_head_before: 3b43eb0d82f51bca8c5ac4b5afe2d0ca20a1f0ec
plan_head_after: 6ed80d4aebabffc0061b57d31f0ab48c1f67d972
tech-stack:
  added: []
  patterns:
    - Normalize platform syscall field widths directly at an existing injected seam
key-files:
  created:
    - .planning/phases/11-darwin-build-test-release-gates/11-01-SUMMARY.md
  modified:
    - internal/supervisor/mount.go
    - internal/supervisor/mount_internal_test.go
key-decisions:
  - Keep one defaultStatfs implementation and cast syscall.Statfs_t.Type directly to int64.
  - Preserve the MountOpts.Statfs injection seam, %w error wrapping, and mount lifecycle unchanged.
  - Resume the evidence-only native gate after Plan 02 removes the external setsid fixture dependency; no Plan 01 production changes were needed.
patterns-established:
  - Platform-width normalization belongs at the existing syscall-to-int64 boundary, not in OS-specific mount lifecycles.
requirements-completed: [MBLD-01]
coverage:
  - id: D1
    description: Portable production filesystem-type converter with direct success and wrapped-error coverage
    requirement: MBLD-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/supervisor -run '^TestDefaultStatfs'"
        status: pass
    human_judgment: false
  - id: D2
    description: CGO-free darwin/arm64 whole-tree build
    requirement: MBLD-01
    verification:
      - kind: other
        ref: "CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./..."
        status: pass
    human_judgment: false
  - id: D3
    description: CGO-free darwin/amd64 whole-tree build
    requirement: MBLD-01
    verification:
      - kind: other
        ref: "CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./..."
        status: pass
    human_judgment: false
  - id: D4
    description: Existing native supervisor mount suite remains race-clean
    requirement: MBLD-01
    verification:
      - kind: integration
        ref: "go test -race -count=1 ./internal/supervisor -run 'TestMount|TestDefaultStatfs'"
        status: pass
    human_judgment: false
duration: 5min
completed: 2026-11-25
status: complete
---

# Phase 11 Plan 01: Darwin Statfs Portability Summary

**The `defaultStatfs` seam now compiles on both Darwin architectures with a direct `int64(st.Type)` conversion, and the native macOS mount suite passes under the race detector after Plan 02 removes the external `setsid` fixture dependency.**

## Performance

- **Duration:** 5 min active execution (4 min initial execution + 1 min continuation)
- **Started:** 2026-11-25T12:37:57Z
- **Resumed:** 2026-11-25T12:56:00Z
- **Completed:** 2026-11-25T12:56:17Z
- **Tasks:** 2/2
- **Files modified:** 2 production/test files, plus this summary

## Accomplishments

- Changed only the production converter return to `int64(st.Type), nil`, preserving the function signature, injected `MountOpts.Statfs` seam, and `%w` error wrap.
- Added `TestDefaultStatfsMatchesSyscall` and `TestDefaultStatfsWrapsError` in the in-package test file, exercising the real converter rather than an injected fake.
- Re-ran both exact CGO-free Darwin whole-tree builds successfully with Go 1.27.4 after Plan 02 landed.
- Passed the exact native macOS supervisor mount suite under `-race -count=1` after Plan 02 replaced the host `setsid` fixtures with Go re-exec helpers.
- Confirmed `go.mod` and `go.sum` remain unchanged after hydrating the frozen `acp-go-sdk v0.20.0` module from the approved Go proxy.

## Verification Evidence

Toolchain:

```text
go version go1.27.4 darwin/arm64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1
```

The environment-required absolute Go executable
`/Users/nil/.local/share/mise/installs/go/1.27.4/bin/go` was used for every command;
`GOTOOLCHAIN=local` prevented toolchain auto-selection. No mise command or trust
change was used.

| Command | Target / scope | Exit | Result |
|---------|----------------|------|--------|
| `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` | darwin/arm64 | 0 | PASS |
| `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...` | darwin/amd64 | 0 | PASS |
| `go test -race -count=1 ./internal/supervisor -run '^TestDefaultStatfs'` | native darwin/arm64 converter tests | 0 | PASS |
| `go test -race -count=1 ./internal/supervisor -run 'TestMount\|TestDefaultStatfs'` | native darwin/arm64 mount suite | 0 | PASS |
| `git diff --exit-code HEAD -- go.mod go.sum` | frozen dependency files | 0 | PASS |
| `git diff --check` | whitespace/error markers | 0 | PASS |

The initially blocked native mount gate was retried without skips or temporary
shims after Plan 02 commit `ccb1bbb`. It passed in 9.115 seconds, including
`TestMountStop`, the existing mount lifecycle tests, and the new production
converter tests.

## Task Commits

1. **Task 1: Prove the darwin/arm64 build through the production Statfs seam** - `ba50bbd` (`fix(11-01)`)
   - `/Users/nil/DiskD/W/Djarvur/zcode-acp-go/internal/supervisor/mount.go`
   - `/Users/nil/DiskD/W/Djarvur/zcode-acp-go/internal/supervisor/mount_internal_test.go`
2. **Task 2: Prove darwin/amd64 compilation and native mount regression parity** - `6ed80d4` (`test(11-01)`)
   - Evidence-only commit: both exact Darwin builds, the native race-tested mount suite, and frozen-module checks passed.
   - No production behavior changed in Task 2.

**Plan metadata:** this completion summary commit supersedes the earlier halted
summary commit `3279f44`.

## Files Created/Modified

- `internal/supervisor/mount.go` - replaces the Linux-only comment and returns `int64(st.Type)` at the existing seam.
- `internal/supervisor/mount_internal_test.go` - adds direct production-converter success and wrapped-error tests.
- `.planning/phases/11-darwin-build-test-release-gates/11-01-SUMMARY.md` - records the completed build and native regression evidence.

## Decisions Made

- Use one portable implementation with an explicit `int64` conversion; no build tags, second implementation, or cast through `int`.
- Keep mount lifecycle, discovery, Linux behavior, and the injected Statfs seam unchanged.
- Keep Task 2 evidence-only and resume it after Plan 02 removes the fixture dependency rather than expanding Plan 01 into process-fixture work.

## Deviations from Plan

None - the production changes followed the plan exactly. Task 2 was resumed
after its declared external dependency was removed by Plan 02; no additional
Plan 01 code changes were required.

## Issues Encountered

The first native mount attempt was blocked by the then-known external `setsid`
fixture dependency. Plan 02 removed both host `setsid` fixtures, and the exact
Task 2 command passed unchanged on continuation. No issues remain.

## Authentication Gates

None.

## User Setup Required

None - no external service configuration was required or used.

## Next Phase Readiness

- Both Darwin architecture compile gates are green with CGO disabled.
- The native macOS supervisor mount suite is race-clean without external `setsid`.
- The int64 Statfs seam and existing mount lifecycle remain unchanged.
- Ready for Plan 03's managed shell/tool-version gates.
- No real ZCode desktop, Keychain, credential, or paid-service access occurred.

## Self-Check: PASSED

- Both modified Go files and this summary exist on disk.
- Task commits `ba50bbd` and `6ed80d4` are ancestors of HEAD.
- Both exact Darwin builds, the focused converter race tests, and the exact native mount race suite passed on the final verification run.
- `go.mod` and `go.sum` show no drift.
- `.planning/STATE.md` and `.planning/ROADMAP.md` retain their pre-execution content hashes.
