---
phase: 14-macos-authentication
plan: "01"
subsystem: auth
tags: [go, darwin, macos, keychain, security, credentials]
requires:
  - phase: 13-desktop-launch-runtime-environment
    provides: macOS launch/env foundation
provides:
  - macOS Keychain KeyLoader over /usr/bin/security
  - Keychain-aware proactive probe with missing/locked/timeout codes
  - Platform key loader/prober factory
affects: [14-macos-authentication, MAUTH-01, MAUTH-02, MAUTH-03]
actuals:
  tokens: 0
  tasks: 3
  commits: 0
plan_head_before: pending
plan_head_after: pending
tech-stack:
  added: []
  patterns:
    - Keychain account is sha256 of the resolved credentials.json path
    - security output decoded as base64url 32-byte key; never logged
    - Coarse stderr markers distinguish missing vs locked items
key-files:
  created:
    - internal/interact/security.go
    - internal/interact/security_test.go
    - internal/interact/desktop.go
  modified:
    - internal/interact/keyringprobe.go
    - internal/interact/keyringprobe_test.go
    - cmd/zcode-acp/main.go
key-decisions:
  - Use the desktop's existing service/account identity.
  - Bound the loader to 15 s and the probe to the existing 5 s.
  - Keep Linux Secret Service constructors byte-compatible.
requirements-completed: [MAUTH-01, MAUTH-02, MAUTH-03]
coverage:
  - id: D1
    description: Api-key path remains Keychain-free
    requirement: MAUTH-01
    verification:
      - kind: unit
        ref: go test ./internal/acpfront -run TestAuthenticate
        status: pass
    human_judgment: false
  - id: D2
    description: enc:v2:os decrypts through the macOS Keychain service/account identity
    requirement: MAUTH-02
    verification:
      - kind: unit
        ref: go test ./internal/interact -run 'TestSecurityKeyLoader|TestKeychainProber'
        status: pass
      - kind: manual_procedural
        ref: real OMP text run authenticated zai individual coding plan with the real Keychain
        status: pass
    human_judgment: false
  - id: D3
    description: Missing and locked Keychain items produce distinct coded errors without secret leakage
    requirement: MAUTH-03
    verification:
      - kind: unit
        ref: go test ./internal/interact -run TestKeychainProber
        status: pass
    human_judgment: false
duration: 35min
completed: 2026-11-26
status: complete
---

# Phase 14 Plan 01: macOS Authentication Summary

**The bridge reads the desktop's `ZCode Credential Encryption` Keychain item through a bounded `/usr/bin/security` call; a real OMP text run authenticated the z.ai individual coding plan, proving the Keychain decryption path.**

## Verification

- `go test -race -count=1 ./internal/interact ./cmd/zcode-acp` — PASS.
- `go test ./... -count=1` + `mise run ci` — PASS.
- Real run evidence: text task reached `authenticate zai-individual-coding-plan: authenticated (read-back proven)` before the quota gate refused generation.

## Live limitation

The real account is currently out of quota (`[1310] Weekly/Monthly Limit Exhausted`,
reset `2026-11-27 18:05:02`), so a completed model reply cannot be observed until
the reset. This is an external account limit, not an auth or Keychain failure.
