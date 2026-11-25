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
affects: [11-darwin-build-test-release-gates, 12-macos-desktop-discovery-launch]
actuals:
  tokens: 782
  tasks: 1
  commits: 1
plan_head_before: 3b43eb0d82f51bca8c5ac4b5afe2d0ca20a1f0ec
plan_head_after: ba50bbdad3106125f1042d367632e9a663439aec
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
  - Halt before claiming native mount regression parity because TestMountStop still requires external setsid, which Plan 02 owns replacing.
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
        status: fail
    human_judgment: true
    rationale: The native macOS host lacks external setsid; the unchanged TestMountStop fixture cannot complete its detached cleanup until Plan 02 replaces that dependency or the gate runs on a Linux host.
duration: 4min
completed: 2026-11-25
status: halted
---

# Phase 11 Plan 01: Darwin Statfs Portability Summary

**The `defaultStatfs` seam now compiles on both Darwin architectures with a direct `int64(st.Type)` conversion, while the full native mount gate remains blocked by the separately planned `setsid` fixture dependency.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-11-25T12:37:57Z
- **Completed:** 2026-11-25T12:42:22Z
- **Tasks:** 1/2 completed; Task 2 blocked at its native regression gate
- **Files modified:** 2 production/test files, plus this halted summary

## Accomplishments

- Changed only the production converter return to `int64(st.Type), nil`, preserving the function signature, injected `MountOpts.Statfs` seam, and `%w` error wrap.
- Added `TestDefaultStatfsMatchesSyscall` and `TestDefaultStatfsWrapsError` in the in-package test file, exercising the real converter rather than an injected fake.
- Re-ran both exact CGO-free Darwin whole-tree builds successfully with Go 1.27.4.
- Confirmed `go.mod` and `go.sum` remain unchanged after hydrating the frozen `acp-go-sdk v0.20.0` module from the approved Go proxy.

## Verification Evidence

Toolchain:

```text
go version go1.27.4 darwin/arm64
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1
```

The environment-required absolute Go executable
`/Users/nil/.local/share/mise/installs/go/1.27.4/bin/go` was used for every command;
`GOTOOLCHAIN=local` prevented toolchain auto-selection.

| Command | Target / scope | Exit | Result |
|---------|----------------|------|--------|
| `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` | darwin/arm64 | 0 | PASS |
| `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...` | darwin/amd64 | 0 | PASS |
| `go test -race -count=1 ./internal/supervisor -run '^TestDefaultStatfs'` | native darwin/arm64 converter tests | 0 | PASS |
| `go test -race -count=1 ./internal/supervisor -run 'TestMount\|TestDefaultStatfs'` | native darwin/arm64 mount suite | 1 | BLOCKED by `TestMountStop` |
| `go test -race -count=1 ./internal/supervisor -run 'TestMount\|TestDefaultStatfs' -skip '^TestMountStop$'` | remaining native mount suite | 0 | PASS |
| `git diff --exit-code HEAD -- go.mod go.sum` | frozen dependency files | 0 | PASS |

The full native mount command failed only at `internal/supervisor/mount_test.go:479`:
`TestMountStop` could not prove its mount root gone. The host reports
`setsid: not found`; the unchanged fixture at `mount_test.go:204` invokes
`setsid sh -c 'sleep 0.5; rm -rf "$1"' ...` for its detached cleaner. This is
the known Phase 11 P0 assigned to Plan 02, not a regression from the Statfs
change. Two temporary external launchers were tried for isolation and failed
identically; both temporary directories were removed and no repository
workaround was added.

## Task Commits

1. **Task 1: Prove the darwin/arm64 build through the production Statfs seam** - `ba50bbd` (`fix(11-01)`)
   - `internal/supervisor/mount.go`
   - `internal/supervisor/mount_internal_test.go`
2. **Task 2: Prove darwin/amd64 compilation and native mount regression parity** - not committed; amd64 build passed, but the exact native mount gate remains blocked by the pre-existing `setsid` fixture dependency.

**Plan metadata:** this halted summary commit.

## Files Created/Modified

- `internal/supervisor/mount.go` - replaces the Linux-only comment and returns `int64(st.Type)` at the existing seam.
- `internal/supervisor/mount_internal_test.go` - adds direct production-converter success and wrapped-error tests.
- `.planning/phases/11-darwin-build-test-release-gates/11-01-SUMMARY.md` - records passing build evidence and the precise native-test blocker.

## Decisions Made

- Use one portable implementation with an explicit `int64` conversion; no build tags, second implementation, or cast through `int`.
- Keep mount lifecycle, discovery, Linux behavior, and the injected Statfs seam unchanged.
- Do not expand Plan 01 into Plan 02's `setsid` fixture replacement merely to turn the native gate green.

## Deviations from Plan

None - the production changes followed the plan exactly. Task 2's evidence-only
verification exposed the already documented Plan 02 dependency; no out-of-scope
code fix was applied.

## Issues Encountered

- The native macOS host has no `setsid`, so the unchanged `TestMountStop` fixture
  cannot run its detached cleaner. The exact mount suite therefore remains red
  even though both Darwin builds and every other selected mount test pass.
- Plan 01 is intentionally `status: halted`: MBLD-01's two compile gates are
  proven, but the plan-level native mount regression truth is not yet proven on
  this host.

## User Setup Required

None - no external service configuration was required or used.

## Next Phase Readiness

- The Darwin `Statfs_t.Type` compile blocker is retired for both architectures.
- Execute Plan 02's Go re-exec fixture replacement, then resume Plan 01 Task 2's
  exact native mount command on macOS; alternatively, run that unchanged mount
  gate on a Linux native runner to establish regression parity.
- No real ZCode desktop, Keychain, credential, or paid-service access occurred.

## Self-Check: PASSED

- Both modified Go files exist and Task 1 commit `ba50bbd` is an ancestor of HEAD.
- Both exact Darwin builds and the focused converter race tests were re-run successfully.
- The failed native mount gate and its Plan 02 ownership are recorded without claiming completion.
- `.planning/STATE.md` and `.planning/ROADMAP.md` retain their pre-execution content hashes.
