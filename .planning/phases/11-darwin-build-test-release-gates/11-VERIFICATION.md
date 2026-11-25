---
phase: 11-darwin-build-test-release-gates
verified: 2026-11-25T19:10:00Z
status: passed
score: 31/31 must-haves verified
covered_files: [".github/workflows/pr-sanity.yml", ".github/workflows/release.yml", ".gitignore", ".goreleaser.yaml", ".planning/phases/11-darwin-build-test-release-gates/11-01-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-01-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-02-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-02-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-03-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-03-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-04-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-04-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-05-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-05-SUMMARY.md", "cmd/check-docs/main.go", "cmd/check-docs/main_test.go", "cmd/scan-secrets/main.go", "cmd/scan-secrets/main_test.go", "cmd/verify-darwin-release/main.go", "cmd/verify-darwin-release/main_test.go", "cmd/zcode-acp/main_test.go", "experiments/portability.test.mjs", "experiments/release.test.mjs", "experiments/tool-versions.test.mjs", "internal/supervisor/discover_test.go", "internal/supervisor/mount.go", "internal/supervisor/mount_internal_test.go", "internal/supervisor/mount_test.go", "internal/zcode/fake/golden_test.go", "mise.toml", "scripts/check-mise-tool-versions.mjs"]
behavior_unverified: 0
overrides_applied: 0
---

# Phase 11: Darwin Build, Test & Release Gates Verification Report

**Phase Goal:** Developers can build, test, and package the existing bridge for both macOS architectures without Linux-only tool assumptions.

**Status:** passed — implementation, local runtime gates, and hosted GitHub CI are all green.

## Goal Achievement

### Verified Locally

| Requirement | Status | Evidence |
|---|---|---|
| MBLD-01 | ✓ VERIFIED | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` and the amd64 equivalent pass with Go 1.27.4. |
| MBLD-02 | ✓ VERIFIED | Full `go test ./...` and `mise run ci` pass on macOS arm64 without external `setsid`; Go docs/secret gates pass; four-tool version gate passes; exact `golangci-lint 2.15.0` reports 0 issues. |
| MBLD-03 | ✓ VERIFIED | GoReleaser 2.20.2 snapshot emits arm64, amd64, and universal archives; Go `crypto/sha256` and `archive/tar` verification pass; each archive contains exactly one regular `zcode-acp`; `lipo` reports `arm64`, `x86_64`, and `x86_64 arm64`. |
| MTEST-01 | ✓ VERIFIED | Full race suite passes on macOS arm64; Node experiments report 47 passed, 1 guarded skip. |

### Verified Hosted

GitHub Actions run `45681033522` for PR #15 passed every required job:

- `build / vet / lint / test / tidy` — SUCCESS
- `build / vet / lint / test / tidy (macOS arm64)` — SUCCESS
- `build / vet / lint / test / tidy (macOS amd64)` — SUCCESS
- `darwin-cross-build` — SUCCESS
- `govulncheck` — SUCCESS
- `secret-scan` — SUCCESS

### Trusted mise Runtime

The repository `mise.toml` was explicitly trusted after review. The real runtime gates then passed:

- `mise install` — four tools installed: Go, golangci-lint, GoReleaser, and Node; no Bash tool.
- `mise exec -- node scripts/check-mise-tool-versions.mjs` — four mise probes, no Bash requirement.
- `mise run ci` — PASS in 110.97 s; Go docs/secret gates and Node suite included.
- `mise run check-docs && mise run secret-scan` — PASS with Go commands.
- `mise run test-experiments` — PASS, 47 passed / 1 guarded skip.
- `mise run verify-darwin-release` — PASS for all three archives.

## Implementation Evidence

- `internal/supervisor/mount.go` plus `statfs_type_linux.go`/`statfs_type_darwin.go`: platform-specific `statfsType` preserves the int64 seam and avoids Linux `unconvert`.
- `internal/supervisor/mount.go`: a denied group SIGTERM falls back to the helper's direct PID before the unconditional group SIGKILL backstop.
- `internal/supervisor/mount_test.go`, `cmd/zcode-acp/main_test.go`: detached fixtures use Go re-exec helpers instead of `setsid`.
- `internal/supervisor/discover_test.go`, `internal/zcode/fake/golden_test.go`: macOS `/var` ↔ `/private/var` aliases normalized in expectations only.
- `cmd/check-docs`: Go documentation conformance gate with unit tests and preserved `CHECK FAIL`/`PENDING` output.
- `cmd/scan-secrets`: Go tracked-file scanner with the original credential regexes, `git ls-files` scope, and redacted findings.
- `mise.toml`: four tool pins; `check-docs` and `secret-scan` tasks run Go commands; canonical `ci` includes them; no Bash pin.
- `scripts/check-mise-tool-versions.mjs`: four-tool exact/line comparison with non-zero drift failure.
- `.github/workflows/pr-sanity.yml`: native macOS arm64/amd64 jobs plus separate Darwin cross-build job; Linux required-check name preserved; no Bash setup or invocation.
- `.goreleaser.yaml`: binary-only tar.gz archives for arm64, amd64, and universal.
- `cmd/verify-darwin-release`: Go `crypto/sha256`, `archive/tar`, `compress/gzip`, and `lipo` verification with adversarial tests; no shell verifier remains.
- `.github/workflows/release.yml`: bounded module hydration, offline verify/tidy, read-only snapshot, write-scoped tag publication, Go verifier task.

## Residuals

- GSD code-review and verifier subagents stalled repeatedly in this runtime; code review is recorded as manual/advisory in `11-REVIEW.md`, and this report was assembled from actual local and hosted command evidence.

**Score:** 31/31 must-haves verified.
