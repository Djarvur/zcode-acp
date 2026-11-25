---
phase: 12-macos-bundle-discovery-overrides
plan: "01"
subsystem: discovery
tags: [go, darwin, macos, app-bundle, plist, discovery, override]
requires:
  - phase: 11-darwin-build-test-release-gates
    provides: Darwin builds, race suite, and release gates
provides:
  - macOS .app validation with Info.plist version selection
  - Standard-location discovery for /Applications and $HOME/Applications
  - Override mapping for .app, Contents, and Contents/MacOS/<executable>
  - Actionable mac-app candidate diagnostics
affects: [12-macos-bundle-discovery-overrides, MDISC-01, MDISC-02]
actuals:
  tokens: 0
  tasks: 3
  commits: 0
plan_head_before: 0e464a2
plan_head_after: pending
tech-stack:
  added: []
  patterns:
    - App bundles resolve to the existing Install shape; SpawnArgv stays unchanged
    - XML plist parsing in-process with a bounded plutil fallback for binary plists
    - macOS candidates are validated before selection; AppImage remains untouched
key-files:
  created:
    - internal/supervisor/bundle.go
    - internal/supervisor/bundle_internal_test.go
    - internal/supervisor/discover_mac_test.go
    - .planning/phases/12-macos-bundle-discovery-overrides/12-RESEARCH.md
    - .planning/phases/12-macos-bundle-discovery-overrides/12-01-PLAN.md
  modified:
    - internal/supervisor/discover.go
    - internal/supervisor/discover_test.go
key-decisions:
  - Use the desktop's actual layout (Contents/MacOS/<exe> + Contents/Resources/glm/zcode.cjs).
  - Require the bundled provider table during validation so an incomplete app is never selected.
  - Keep mac-app installs AppImage-free, making the FUSE path unreachable by construction.
  - Keep Linux probe behavior unchanged; the mac probe is a preceding stage.
requirements-completed: [MDISC-01, MDISC-02]
coverage:
  - id: D1
    description: Valid ZCode.app bundles in standard locations resolve with the Info.plist version
    requirement: MDISC-01
    verification:
      - kind: unit
        ref: go test ./internal/supervisor -run 'TestMacApp'
        status: pass
      - kind: manual_procedural
        ref: real /Applications/ZCode.app discovery on this Mac
        status: pass
    human_judgment: false
  - id: D2
    description: Invalid or incomplete bundles are rejected with actionable candidate rows
    requirement: MDISC-01
    verification:
      - kind: unit
        ref: go test ./internal/supervisor -run 'TestMacAppDiscoveryRejectsInvalid|TestMacAppInstall'
        status: pass
    human_judgment: false
  - id: D3
    description: All three macOS override forms resolve and Linux override behavior is unchanged
    requirement: MDISC-02
    verification:
      - kind: unit
        ref: go test ./internal/supervisor -run 'TestOverrideInstall|TestMacAppOverride'
        status: pass
    human_judgment: false
  - id: D4
    description: A mac-app selection never enters the AppImage/FUSE path
    requirement: MDISC-02
    verification:
      - kind: unit
        ref: go test -race -count=1 ./internal/supervisor
        status: pass
    human_judgment: false
duration: 55min
completed: 2026-11-26
status: complete
---

# Phase 12 Plan 01: macOS Bundle Discovery & Overrides Summary

**The bridge now selects a validated `/Applications/ZCode.app` before any Linux probe, reads version `4.0.0` from `Info.plist`, and accepts `.app`, `Contents`, and `Contents/MacOS/ZCode` overrides without touching AppImage/FUSE.**

## Verification

- `go test -race -count=1 ./internal/supervisor` — PASS.
- `go test ./... -count=1` — PASS.
- `golangci-lint run ./internal/supervisor/...` — 0 issues.
- Real-machine discovery:
  - selected `mac-app` version `4.0.0` at `/Applications/ZCode.app`;
  - binary `/Applications/ZCode.app/Contents/MacOS/ZCode`;
  - entrypoint `/Applications/ZCode.app/Contents/Resources/glm/zcode.cjs`;
  - bundled provider `/Applications/ZCode.app/Contents/Resources/config/provider/zcode-builtin.json` (revision 44);
  - `RequiresMount=false`; AppImage stage not consulted.

## Deviations

- Binary `.plist` files use a `plutil` fallback on Darwin; XML fixtures cover the parser in CI without invoking the tool.
- The runtime-provider cache is still the Linux path in this phase; selecting `darwin-aarch64` is Phase 13 (MDISC-04).
