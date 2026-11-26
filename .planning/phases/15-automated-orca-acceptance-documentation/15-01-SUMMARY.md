---
phase: 15-automated-orca-acceptance-documentation
plan: "01"
subsystem: tests-docs
tags: [go, darwin, macos, orca, e2e, readme]
requires:
  - phase: 14-macos-authentication
    provides: Keychain auth and platform wiring
provides:
  - Fake ZCode.app text-task E2E
  - Account-provider level fallback for Orca text generators
  - English macOS README guide
affects: [15-automated-orca-acceptance-documentation, MTEST-02, MORCA-01, MDOC-01]
actuals:
  tokens: 0
  tasks: 3
  commits: 0
plan_head_before: pending
plan_head_after: pending
tech-stack:
  added: []
  patterns:
    - Fake .app shim drops the entrypoint argument before exec'ing fake-app-server
    - Account provider text selections default to `low` only when no level was configured
key-files:
  created:
    - cmd/zcode-acp/macapp_test.go
  modified:
    - cmd/zcode-acp/main.go
    - cmd/zcode-acp/text_test.go
    - README.md
key-decisions:
  - Keep the fake backend zero-spend; no real desktop or Keychain in CI.
  - Document the one-time macOS Keychain access approval.
requirements-completed: [MTEST-02, MORCA-01, MDOC-01]
coverage:
  - id: D1
    description: Fake macOS .app Orca text E2E
    requirement: MORCA-01
    verification:
      - kind: e2e
        ref: go test ./cmd/zcode-acp -run TestOmpTextMacAppEndToEnd
        status: pass
    human_judgment: false
  - id: D2
    description: Fixtures cover .app discovery/override, argv, provider cache, Keychain contract
    requirement: MTEST-02
    verification:
      - kind: unit
        ref: go test ./internal/supervisor ./internal/interact ./cmd/zcode-acp
        status: pass
    human_judgment: false
  - id: D3
    description: English macOS guide
    requirement: MDOC-01
    verification:
      - kind: other
        ref: mise run check-docs
        status: pass
    human_judgment: false
duration: 30min
completed: 2026-11-26
status: complete
---

# Phase 15 Plan 01: Automated Orca Acceptance & Documentation Summary

**The fake macOS `.app` E2E runs Orca's exact `omp --print --mode text @file` contract to exact stdout/exit 0, and the README now documents macOS build, discovery, Keychain, Orca, and limitations.**

## Verification

- `go test ./cmd/zcode-acp -run 'TestOmpTextMacAppEndToEnd|TestOmpTextSelectionRebind'` — PASS.
- Full `mise run ci` — PASS (Go race suite, 48 Node tests with 1 guarded skip, docs/secret gates, lint 0 issues).
- README `check-docs` — PASS.
