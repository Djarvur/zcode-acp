---
status: testing
phase: 11-darwin-build-test-release-gates
source: [11-VERIFICATION.md]
started: 2026-11-25T17:35:00Z
updated: 2026-11-25T17:35:00Z
---

## Current Test

number: 1
name: Run the canonical mise gates on macOS
expected: |
  After trusting the repository mise.toml, `mise run ci` and the
  `mise exec -- node scripts/check-mise-tool-versions.mjs` gate pass on macOS
  arm64 with Bash 5.3 and the pinned tools.
awaiting: user response

## Tests

### 1. Run the canonical mise gates on macOS
expected: `mise run ci` passes and the version gate logs five matching pins.
result: [pending]

### 2. Observe hosted GitHub macOS and release workflows
expected: Push the phase branch and confirm macOS arm64/amd64, Darwin cross-build,
and the release snapshot workflow all pass on GitHub-hosted runners.
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
