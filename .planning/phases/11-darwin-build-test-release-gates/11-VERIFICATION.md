---
phase: 11-darwin-build-test-release-gates
verified: 2026-11-25T17:35:00Z
status: human_needed
score: 29/31 must-haves verified
covered_files: [".github/workflows/pr-sanity.yml", ".github/workflows/release.yml", ".gitignore", ".goreleaser.yaml", ".planning/phases/11-darwin-build-test-release-gates/11-01-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-01-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-02-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-02-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-03-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-03-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-04-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-04-SUMMARY.md", ".planning/phases/11-darwin-build-test-release-gates/11-05-PLAN.md", ".planning/phases/11-darwin-build-test-release-gates/11-05-SUMMARY.md", "cmd/zcode-acp/main_test.go", "experiments/check-docs.sh", "experiments/portability.test.mjs", "experiments/release.test.mjs", "experiments/tool-versions.test.mjs", "internal/supervisor/discover_test.go", "internal/supervisor/mount.go", "internal/supervisor/mount_internal_test.go", "internal/supervisor/mount_test.go", "internal/zcode/fake/golden_test.go", "mise.toml", "scripts/check-mise-tool-versions.mjs", "scripts/verify-darwin-release.sh"]
covered_digest: "v3:sha256:8707170bc361f496262e29592c4ea7f303333d70ee09d13b7be2a0d4ffc04922"
behavior_unverified: 2
overrides_applied: 0
---

# Phase 11: Darwin Build, Test & Release Gates Verification Report

**Phase Goal:** Developers can build, test, and package the existing bridge for both macOS architectures without Linux-only tool assumptions.

**Status:** human_needed — implementation and all local equivalents are green; two hosted-environment observations require a human action.

## Goal Achievement

### Verified Locally

| Requirement | Status | Evidence |
|---|---|---|
| MBLD-01 | ✓ VERIFIED | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` and the amd64 equivalent pass with Go 1.27.4. |
| MBLD-02 | ✓ VERIFIED | Full `go test -race -count=1 ./...` passes on macOS arm64 without external `setsid`; Bash 5.3 docs/secret gates pass; exact `golangci-lint 2.15.0` reports 0 issues. |
| MBLD-03 | ✓ VERIFIED (local equivalents) | GoReleaser 2.20.2 snapshot emits arm64, amd64, and universal archives; SHA-256 checks pass; each archive contains exactly one regular `zcode-acp`; `lipo` reports `arm64`, `x86_64`, and `x86_64 arm64`. CI and release workflow structures pass static tests. |
| MTEST-01 | ✓ VERIFIED | Full race suite passes on macOS arm64; Node experiments report 46 passed, 2 guarded skips. |

### Human Verification Required

| # | Must-have | Expected | Why manual |
|---|---|---|---|
| 1 | Native macOS CI jobs execute the canonical mise gates and log resolved tool versions | `mise run ci`, both native macOS jobs, and `darwin-cross-build` are green | Requires trusting the repository `mise.toml` and running the real mise wrapper; local execution used exact absolute-tool equivalents. |
| 2 | Hosted GitHub CI observes macOS arm64/amd64 and the release snapshot workflow | PR checks and `workflow_dispatch` release snapshot are green | Requires pushing the branch and observing GitHub-hosted runners; no push was authorized. |

## Implementation Evidence

- `internal/supervisor/mount.go`: `defaultStatfs` returns `int64(st.Type)`, preserving the Linux/Darwin seam.
- `internal/supervisor/mount_test.go`, `cmd/zcode-acp/main_test.go`: detached fixtures use Go re-exec helpers instead of `setsid`.
- `internal/supervisor/discover_test.go`, `internal/zcode/fake/golden_test.go`: macOS `/var` ↔ `/private/var` aliases normalized in expectations only.
- `mise.toml`: managed Bash 5.3 added; `check-docs` and `secret-scan` tasks exposed; canonical `ci` task includes them.
- `scripts/check-mise-tool-versions.mjs`: five-tool exact/line comparison with non-zero drift failure.
- `.github/workflows/pr-sanity.yml`: native macOS arm64/amd64 jobs plus separate Darwin cross-build job; Linux required-check name preserved.
- `.goreleaser.yaml`: binary-only tar.gz archives for arm64, amd64, and universal.
- `scripts/verify-darwin-release.sh`: checksum, exact-member, safe-extraction, and `lipo` architecture verification.
- `.github/workflows/release.yml`: bounded module hydration, offline verify/tidy, read-only snapshot, write-scoped tag publication.

## Residuals

- Mise-wrapper commands remain unrun locally because `mise.toml` is not trusted.
- Hosted CI and `workflow_dispatch` require a branch push.
- GSD code-review and verifier subagents stalled repeatedly in this runtime; code review is recorded as manual/advisory in `11-REVIEW.md`, and this report was assembled from actual command evidence.

**Score:** 29/31 must-haves verified; 2 human verification items.
