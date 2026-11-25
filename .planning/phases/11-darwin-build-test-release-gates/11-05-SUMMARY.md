---
phase: 11-darwin-build-test-release-gates
plan: "05"
subsystem: infra
tags: [go, darwin, goreleaser, macos, release, checksums, lipo, github-actions]
requires:
  - phase: 11-01
    provides: CGO-free darwin/arm64 and darwin/amd64 build compatibility
  - phase: 11-03
    provides: Go docs/secret gates and the four-tool mise-resolved version gate
  - phase: 11-04
    provides: Native macOS CI gates and exact Darwin cross-build jobs
provides:
  - GoReleaser 2.20.2 config for thin arm64/amd64 and universal binary-only archives
  - Go verifier for checksum, exact-member, safe-extraction, and exact-architecture checks
  - Offline static release contracts and Go adversarial archive coverage
  - Read-only macOS snapshot workflow and tag-only write-scoped publication
affects: [11-darwin-build-test-release-gates, MBLD-03]
actuals:
  tokens: 18638
  tasks: 3
  commits: 4
plan_head_before: fbc0dbe648b013d5d618332eac8738625b3b12e7
plan_head_after: 16516c5
tech-stack:
  added: []
  patterns:
    - GoReleaser archive ids select binaries while a non-matching extra-file glob suppresses default documentation members
    - Validate checksums and archive structure on private copies before fresh-directory extraction
    - Go archive/tar + compress/gzip + crypto/sha256 verification with lipo as the only external tool
    - Read-only snapshot job gates tag-only publication and owns no release token
key-files:
  created:
    - .goreleaser.yaml
    - cmd/verify-darwin-release/main.go
    - cmd/verify-darwin-release/main_test.go
    - experiments/release.test.mjs
    - .planning/phases/11-darwin-build-test-release-gates/11-05-SUMMARY.md
  modified:
    - .github/workflows/release.yml
    - mise.toml
    - .gitignore
    - .planning/WINDOWS.md
key-decisions:
  - Use GoReleaser 2.20.2's current archives.ids field; builds is deprecated in the exact schema.
  - Use a non-matching archives.files sentinel because files adds extra filesystem members and does not select built binaries.
  - Stage private archive copies and repeat checksum verification before validating or extracting them.
  - Implement the verifier in Go; lipo remains the only external command.
  - Keep snapshot permissions read-only and confine contents: write plus GITHUB_TOKEN to the tag-only release job.
patterns-established:
  - Release artifact trust chain: exact tool pin, frozen modules, offline build, SHA-256, exact tar member/type, fresh extraction, lipo.
  - Workflow dependency gate: version preflight, bounded proxy download, drift rejection, offline verify/tidy, then offline GoReleaser.
requirements-completed: [MBLD-03]
coverage:
  - id: D1
    description: Exact GoReleaser 2.20.2 snapshot emits checksummed binary-only arm64, amd64, and universal archives
    requirement: MBLD-03
    verification:
      - kind: e2e
        ref: /tmp/zcode-acp-release-tools.DrrlmY/bin/goreleaser release --snapshot --clean --skip=publish
        status: pass
      - kind: other
        ref: go run ./cmd/verify-darwin-release dist
        status: pass
    human_judgment: false
  - id: D2
    description: Go verifier rejects traversal, absolute paths, symlinks, extra members, missing binaries, and malformed checksums
    requirement: MBLD-03
    verification:
      - kind: unit
        ref: go test ./cmd/verify-darwin-release
        status: pass
    human_judgment: false
  - id: D3
    description: Read-only macOS snapshot workflow replays the proven dependency and artifact gates
    requirement: MBLD-03
    verification:
      - kind: unit
        ref: experiments/release.test.mjs#release workflow runs the proven snapshot with read-only rights
        status: pass
      - kind: other
        ref: direct Go 1.27.4 frozen-module download, drift, verify, and tidy sequence
        status: pass
    human_judgment: false
  - id: D4
    description: Tag publication needs the snapshot and alone owns write permission and GITHUB_TOKEN
    requirement: MBLD-03
    verification:
      - kind: unit
        ref: experiments/release.test.mjs#release workflow scopes snapshot and tag publication
        status: pass
    human_judgment: false
  - id: D5
    description: Hosted workflow_dispatch run observes read-only snapshot success and publish-job skip
    requirement: MBLD-03
    verification:
      - kind: manual_procedural
        ref: .github/workflows/release.yml hosted workflow_dispatch run on the phase head SHA
        status: unknown
    human_judgment: true
    rationale: No branch was pushed and no hosted Actions run was available; no tag or release was created.
duration: 20min
completed: 2026-11-25
status: complete
---

# Phase 11 Plan 05: Darwin Release Artifacts Summary

**GoReleaser 2.20.2 produces independently verified binary-only Darwin arm64, amd64, and universal archives; the verifier is now a Go command with no Bash dependency.**

## Performance

- **Duration:** 20 minutes
- **Started:** 2026-11-25T13:46:59Z
- **Completed:** 2026-11-25T14:07:02Z
- **Tasks:** 3/3
- **Files modified:** 5 implementation/test files, plus this summary and the broken-windows ledger

## Accomplishments

- Added two CGO-free Darwin builds, one universal binary, thin/universal tar.gz archives, and SHA-256 `checksums.txt`.
- Added a Go verifier that requires exactly one regular `zcode-acp` member, rejects unsafe member sets/types before extraction, stages private copies, and compares exact `lipo -archs` sets without executing a binary.
- Added Go unit fixtures for valid archives plus traversal, absolute-path, symlink, extra-member, missing-binary, directory, malformed-checksum, and wrong-architecture cases.
- Reworked `release.yml` into a read-only macOS snapshot job plus a tag-only publish job with independent frozen-module and offline gates.
- Kept `contents: write` and `GITHUB_TOKEN` confined to the publish job; the snapshot job uploads all three archives and checksums without a token.
- Replaced the shell verifier and `mise exec -- bash` call with `mise run verify-darwin-release`.
- Added `/dist/` to `.gitignore`.

## Verification Evidence

### Passed checks

Direct tools used:

```text
/Users/nil/.local/share/mise/installs/go/1.27.4/bin/go -> go1.27.4 darwin/arm64
/opt/homebrew/bin/node -> v26.13.1
/tmp/zcode-acp-release-tools.DrrlmY/bin/goreleaser -> GitVersion v2.20.2, GoVersion go1.27.4
```

The exact GoReleaser binary was installed through `GOPROXY=https://proxy.golang.org` into a fresh temporary GOBIN.

| Command / check | Result |
|-----------------|--------|
| `node --test experiments/release.test.mjs` | PASS — 4/4 |
| `go test ./cmd/verify-darwin-release` | PASS — valid and adversarial archive coverage |
| `go run ./cmd/verify-darwin-release dist` | PASS — checksums, exact members/types, and all architecture sets |
| `mise run verify-darwin-release` | PASS — same real `dist` snapshot |
| `mise run ci` | PASS — 110.97 s, lint 0 issues |
| `GOPROXY=off mise exec -- goreleaser check` with exact 2.20.2 | PASS |
| `GOPROXY=https://proxy.golang.org go mod download` + `git diff --exit-code HEAD -- go.mod go.sum` | PASS |
| `GOPROXY=off go mod verify` + offline `go mod tidy` + frozen-pin assertions | PASS |
| `GOPROXY=off goreleaser release --snapshot --clean --skip=publish` on final commit `84a2812` | PASS |
| `git diff --check` | PASS |

Final snapshot artifacts:

| Archive | SHA-256 | Member | `lipo -archs` |
|---------|---------|--------|---------------|
| `zcode-acp_0.23.0-SNAPSHOT-84a2812_darwin_arm64.tar.gz` | `06b3f379d64f5ee16b0031a88da4945ad36935fb9217a9906ab56a79590df223` | one regular `zcode-acp` | `arm64` |
| `zcode-acp_0.23.0-SNAPSHOT-84a2812_darwin_amd64.tar.gz` | `87a9ad043fa1d48ba3c68802ae04e80918b7c8173e6cebd87c8b5c58a17c2ab0` | one regular `zcode-acp` | `x86_64` |
| `zcode-acp_0.23.0-SNAPSHOT-84a2812_darwin_universal.tar.gz` | `bfa6f6954529663187428fffb399660230d2ee79c53f39934d141c22eeff2248` | one regular `zcode-acp` | `x86_64 arm64` |

The Go verifier independently recomputed SHA-256 over `dist/` and private staged copies for all three archives.

### Pending checks — not claimed as passed

| Check | Status / reason |
|-------|-----------------|
| Hosted `workflow_dispatch` snapshot observation | NOT RUN — no branch was pushed. Confirm read-only rights, bounded module hydration, offline checksum/tidy, three archives plus checksums, verifier success, and publish-job skip. Recorded as WINDOWS entry 29. |

No local snapshot or exact-tool check remains blocked. No tag was created or pushed, and no release was published.

## Task Commits

1. **Task 1: Build and verify one complete binary-only Darwin release slice** — `968305a` (`feat(11-05)`)
2. **Task 2: Automate the proven snapshot as a read-only macOS job** — `e02a266` (`feat(11-05)`)
3. **Task 3: Gate tag publication on the verified snapshot** — `84a2812` (`feat(11-05)`)
4. **User-directed Go verifier migration** — `16516c5` (`feat(11)`)

**Plan metadata:** committed separately with this summary and `.planning/WINDOWS.md`.

## Files Created/Modified

- `.goreleaser.yaml` — exact-version two-target/universal build, binary-only archives, and SHA-256 config.
- `cmd/verify-darwin-release/main.go` — Go checksum, exact-member/type, private-copy, extraction, and `lipo` verification.
- `cmd/verify-darwin-release/main_test.go` — valid and adversarial archive fixtures.
- `experiments/release.test.mjs` — offline config and workflow contracts; shell fixture arm deleted.
- `.github/workflows/release.yml` — read-only snapshot and tag-only write-scoped publication with the Go verifier task.
- `mise.toml` — `verify-darwin-release` task calls `go run ./cmd/verify-darwin-release dist`.
- `.gitignore` — ignores GoReleaser `/dist/` output.
- `.planning/WINDOWS.md` — records resolved schema adaptation and hosted-run observation.

## Decisions Made

- Used `archives.ids`, the current GoReleaser 2.20.2 field, instead of deprecated `archives.builds`.
- Used a non-matching extra-file sentinel glob so default README/LICENSE members are suppressed while archive IDs select the built binary.
- Added a second checksum pass over private staged copies to close the dist-to-extraction race.
- Made publication depend on the verified snapshot and required `github.event_name == 'push'` plus a `refs/tags/v*` ref.
- Replaced the Bash verifier with a Go command; only `lipo` remains an external executable.
- Verified archive contents from private staged copies with Go SHA-256 before `archive/tar` extraction.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected GoReleaser 2.20.2 binary-only archive selection**
- **Found during:** Task 1 real snapshot
- **Issue:** `files: [zcode-acp]` is treated as an extra-filesystem glob, so the snapshot failed after both thin builds and the universal binary were produced.
- **Fix:** Used archive `ids` to select binaries and a non-matching `files` glob to suppress default documentation members. The exact 2.20.2 source confirms `files` is additive and `builds` is deprecated.
- **Files modified:** `.goreleaser.yaml`, `experiments/release.test.mjs`
- **Verification:** Exact `goreleaser check`, repeated real snapshots, archive listings, checksums, and adversarial fixtures all pass.
- **Committed in:** `968305a`

**2. [Rule 1 - Bug] Tightened publish-step order assertion**
- **Found during:** Task 3 static test
- **Issue:** The order assertion matched the release job's display name before the dependency gate instead of the publish step.
- **Fix:** Matched the full step marker `      - name: goreleaser release`.
- **Files modified:** `experiments/release.test.mjs`
- **Verification:** Release suite passes 4/4 active tests; adversarial fixtures pass 7/7.
- **Committed in:** `84a2812`

---

### User-directed Architectural Correction

**3. [User-directed] Replace the shell release verifier with Go**
- **Found during:** User review after the trusted mise runtime correction
- **Issue:** Bash remained a managed/installed dependency for release verification.
- **Fix:** Added `cmd/verify-darwin-release` using archive/tar, compress/gzip, crypto/sha256, and lipo; deleted the shell verifier and moved workflows/tests to the Go command.
- **Verification:** Go unit tests cover valid and adversarial archives; the command passes the real `dist` snapshot and `mise run verify-darwin-release`.
- **Committed in:** `16516c5`

---

**Total deviations:** 2 auto-fixed Rule 1 bugs and 1 user-directed architectural correction.
**Impact on plan:** All release guarantees remain intact while removing the Bash toolchain dependency.

## Issues Encountered

- The verifier now relies only on Go stdlib plus Apple's `lipo`, so archive parsing and checksum verification have no shell-tool prerequisites.
- Hosted Actions observation is pending a branch push. This is recorded as an open `unrun-verify`, not represented as green.

## Authentication Gates

None.

## User Setup Required

None for implementation. No account, credential, Keychain, desktop, paid service, or `api.z.ai` access was used. The exact GoReleaser tool was acquired only through the approved Go module proxy into `/tmp/zcode-acp-release-tools.DrrlmY/bin`.

## Known Stubs

None. Adversarial archive coverage moved to `go test ./cmd/verify-darwin-release` and runs by default.

## Threat Flags

None beyond the plan's existing trust boundaries. The verifier never executes artifacts; snapshot work has read-only rights and no token; write rights and `GITHUB_TOKEN` are tag-publish-only; module access is bounded to the approved public Go proxy before offline builds.

## Next Phase Readiness

- The local release slice is complete and proven with the exact pinned GoReleaser version.
- Release verification has no Bash or shell-script dependency; it uses the Go verifier plus Apple `lipo`.
- The workflow is ready for a hosted `workflow_dispatch` observation after the phase branch is pushed.
- `.planning/STATE.md` and `.planning/ROADMAP.md` were intentionally not updated, per the execution request.

## Self-Check: PASSED

- Go verifier implementation/tests, release config, workflow contracts, ledger, and this summary exist on disk.
- Commits `968305a`, `e02a266`, `84a2812`, and `16516c5` are ancestors of HEAD.
- The real snapshot, `go run ./cmd/verify-darwin-release dist`, `mise run verify-darwin-release`, Go adversarial tests, static release tests, and `mise run ci` pass.
- The verifier uses Go stdlib for checksums and archives; `lipo` is the only external executable.
- `.planning/STATE.md` and `.planning/ROADMAP.md` are untouched by the migration.
- The pre-existing untracked `.planning/milestone.lock` was left untouched.
