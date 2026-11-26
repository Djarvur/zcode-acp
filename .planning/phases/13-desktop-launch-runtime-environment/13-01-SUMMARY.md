---
phase: 13-desktop-launch-runtime-environment
plan: "01"
subsystem: launch
tags: [go, darwin, macos, provider-cache, path, login-shell, launch]
requires:
  - phase: 12-macos-bundle-discovery-overrides
    provides: Validated macOS .app Install selection
provides:
  - Runtime-provider platform mapping (darwin-aarch64 / darwin-x86_64)
  - Exact app-version provider cache preference
  - Bounded non-interactive login-shell PATH union
  - Exact macOS spawn argv/env contract coverage
affects: [13-desktop-launch-runtime-environment, MDISC-03, MDISC-04, MDISC-05]
actuals:
  tokens: 0
  tasks: 3
  commits: 0
plan_head_before: pending
plan_head_after: pending
tech-stack:
  added: []
  patterns:
    - Host platform directory derived from GOOS/GOARCH, never a Linux literal
    - Login-shell probe uses a small env allowlist and returns caller env on any failure
key-files:
  created:
    - internal/supervisor/provider_platform.go
    - internal/supervisor/provider_platform_test.go
    - internal/supervisor/path.go
    - internal/supervisor/path_internal_test.go
  modified:
    - internal/supervisor/discover.go
    - internal/supervisor/discover_mac_test.go
    - cmd/zcode-acp/main.go
key-decisions:
  - Prefer the provider cache version matching the selected app over a newer entry.
  - Run zsh/bash with -lc, never -i; source the interactive rc explicitly with output discarded.
  - Only PATH is unioned; ChildEnv remains the single scrub boundary.
requirements-completed: [MDISC-03, MDISC-04, MDISC-05]
coverage:
  - id: D1
    description: Exact app-server argv and env for a macOS bundle
    requirement: MDISC-03
    verification:
      - kind: unit
        ref: go test ./internal/supervisor -run TestMacAppSpawnContract
        status: pass
    human_judgment: false
  - id: D2
    description: Darwin provider cache and exact app version selected
    requirement: MDISC-04
    verification:
      - kind: unit
        ref: go test ./internal/supervisor -run 'TestProviderPlatformDir|TestMacAppProviderCache'
        status: pass
      - kind: manual_procedural
        ref: real discovery selected darwin-aarch64/4.0.0 revision 48 on this Mac
        status: pass
    human_judgment: false
  - id: D3
    description: Bounded non-interactive login-shell PATH union with scrubbing
    requirement: MDISC-05
    verification:
      - kind: unit
        ref: go test ./internal/supervisor -run 'TestEnrichParentPATH|TestUnionPATH|TestLastNonEmptyLine'
        status: pass
      - kind: manual_procedural
        ref: real EnrichParentPATH returned the user's mise/homebrew/go PATH union
        status: pass
    human_judgment: false
duration: 40min
completed: 2026-11-26
status: complete
---

# Phase 13 Plan 01: Desktop Launch & Runtime Environment Summary

**A discovered `.app` launches with the canonical Electron-as-Node argv, the Darwin provider cache (`darwin-aarch64/4.0.0`, revision 48), and the user's login-shell PATH unioned ahead of Orca's launchd PATH.**

## Verification

- `go test -race -count=1 ./internal/supervisor` — PASS.
- `go test ./... -count=1` — PASS.
- `golangci-lint run ./internal/supervisor/... ./cmd/zcode-acp/...` — 0 issues.
- Real discovery selected:
  - layout `mac-app`, version `4.0.0`;
  - builtin `/Users/nil/.zcode/v2/runtime/provider/darwin-aarch64/4.0.0/endpoint-78d7c3bef4024722642626fe3669a799/zcode-builtin.json` (revision 48);
  - no AppImage/mount.
- Real PATH union included mise, Homebrew, Go, cargo, orbstack, and the bundled ZCode tools.

## Deviations

- The login-shell union is Darwin-only; Linux keeps the caller PATH unchanged.
- `plutil` fallback remains bounded to the Info.plist parse path.
