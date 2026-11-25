---
phase: "11"
slug: "darwin-build-test-release-gates"
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-25"
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` plus Node `node:test` |
| **Config file** | `mise.toml`, `.github/workflows/pr-sanity.yml`, `.github/workflows/release.yml` |
| **Quick run command** | `go test ./internal/supervisor -run 'TestMountStop|TestMountHappyPath' -count=1` |
| **Full suite command** | `go test -race -count=1 ./... && node --test experiments/` |
| **Cross-build commands** | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` and `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...` |
| **Release verification** | `goreleaser release --snapshot --clean --skip=publish`, `shasum -a 256 -c checksums.txt`, `lipo -archs` |
| **Estimated runtime** | To be measured on the first native macOS run |

---

## Sampling Rate

- **After every task commit:** Run the affected Go package tests and `node --test experiments/portability.test.mjs` once it exists.
- **After the Statfs change:** Run both exact Darwin cross-build commands.
- **After every plan wave:** Run the full native `-race` suite and Node experiment suite on each supported runner.
- **Before `/gsd:verify-work`:** Both Darwin cross-builds, Linux/macOS native gates, shell gates, and release snapshot verification must be green.
- **Max feedback latency:** Target under 120 seconds for package-level feedback; full-suite latency is measured during Wave 0.

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | MBLD-01 | T-11-01 | Darwin builds use the declared `int64` Statfs seam without CGO | unit/build | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` | ❌ W0 | ⬜ pending |
| 11-01-02 | 01 | 1 | MBLD-01 | T-11-01 | Intel Darwin build remains reproducible | build | `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...` | ❌ W0 | ⬜ pending |
| 11-02-01 | 02 | 1 | MTEST-01, MBLD-02 | T-11-02 | Process fixtures create sessions through Go, not an absent host utility | integration | `go test -race -count=1 ./internal/supervisor ./cmd/zcode-acp` | ❌ W0 | ⬜ pending |
| 11-02-02 | 02 | 1 | MBLD-02 | T-11-03 | Shell gates do not silently run under macOS Bash 3.2 | static unit | `node --test experiments/portability.test.mjs` | ❌ W0 | ⬜ pending |
| 11-03-01 | 03 | 2 | MBLD-03 | T-11-04 | Linux and both macOS architectures execute the canonical gates | CI | `mise run ci` on each matrix runner | ❌ W0 | ⬜ pending |
| 11-03-02 | 03 | 2 | MBLD-03 | T-11-05 | Snapshot artifacts match expected architectures and checksums | release smoke | `goreleaser release --snapshot --clean --skip=publish && shasum -a 256 -c checksums.txt && lipo -archs <artifact>` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Add a direct production `defaultStatfs` converter test if the injected mount tests do not exercise the Darwin-width conversion.
- [ ] Replace both external `setsid` fixture call sites with a re-executed Go test helper using `syscall.SysprocAttr{Setsid: true}`.
- [ ] Add `experiments/portability.test.mjs` to reject executable `setsid` fixture calls and assert shell gates use managed Bash.
- [ ] Add managed Bash to `mise.toml` and route shell-gate workflow steps through it.
- [ ] Add native macOS arm64/amd64 CI jobs and the two-command Darwin cross-build job.
- [ ] Add `.goreleaser.yaml` and a macOS snapshot job with checksum and architecture verification.
- [ ] Resolve the pinned Go toolchain, ACP SDK module, GoReleaser, and golangci-lint versions before claiming the phase gate green.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GitHub-hosted macOS runner labels and cache behavior | MBLD-03 | Runner availability and cache contents are external service state | Observe the first macOS matrix run; record runner OS/arch, tool versions, and cache hit/miss. |
| GoReleaser 2.20.2 universal artifact schema | MBLD-03 | The pinned binary is not installed in the initial environment | Install the exact pin in an isolated tool directory, run snapshot, inspect archive names, checksums, and `lipo -archs`. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency target measured and recorded
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
