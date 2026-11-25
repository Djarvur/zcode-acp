---
phase: 11-darwin-build-test-release-gates
plan: "05"
subsystem: infra
tags: [go, darwin, goreleaser, macos, release, checksums, lipo, github-actions]
requires:
  - phase: 11-01
    provides: CGO-free darwin/arm64 and darwin/amd64 build compatibility
  - phase: 11-03
    provides: Managed Bash and the mise-resolved tool-version gate
  - phase: 11-04
    provides: Native macOS CI gates and exact Darwin cross-build jobs
provides:
  - GoReleaser 2.20.2 config for thin arm64/amd64 and universal binary-only archives
  - Independent checksum, exact-member, safe-extraction, and exact-architecture verifier
  - Offline static release contracts and guarded adversarial archive fixtures
  - Read-only macOS snapshot workflow and tag-only write-scoped publication
affects: [11-darwin-build-test-release-gates, MBLD-03]
actuals:
  tokens: 7860
  tasks: 3
  commits: 3
plan_head_before: fbc0dbe648b013d5d618332eac8738625b3b12e7
plan_head_after: 84a2812cf1730cbbd01071f0815faeb43571b962
tech-stack:
  added: []
  patterns:
    - GoReleaser archive ids select binaries while a non-matching extra-file glob suppresses default documentation members
    - Validate checksums and archive structure on private copies before fresh-directory extraction
    - Read-only snapshot job gates tag-only publication and owns no release token
key-files:
  created:
    - .goreleaser.yaml
    - scripts/verify-darwin-release.sh
    - experiments/release.test.mjs
    - .planning/phases/11-darwin-build-test-release-gates/11-05-SUMMARY.md
  modified:
    - .github/workflows/release.yml
    - .gitignore
    - .planning/WINDOWS.md
key-decisions:
  - Use GoReleaser 2.20.2's current archives.ids field; builds is deprecated in the exact schema.
  - Use a non-matching archives.files sentinel because files adds extra filesystem members and does not select built binaries.
  - Stage private archive copies and repeat checksum verification before validating or extracting them.
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
        ref: /opt/homebrew/bin/bash scripts/verify-darwin-release.sh dist
        status: pass
    human_judgment: false
  - id: D2
    description: Verifier rejects traversal, absolute paths, symlinks, extra members, and missing binaries before extraction or lipo
    requirement: MBLD-03
    verification:
      - kind: unit
        ref: experiments/release.test.mjs#artifact verifier rejects adversarial archives before extraction
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

**GoReleaser 2.20.2 now produces independently verified binary-only Darwin arm64, amd64, and universal archives, with read-only snapshot automation and tag-only publication rights.**

## Performance

- **Duration:** 20 minutes
- **Started:** 2026-11-25T13:46:59Z
- **Completed:** 2026-11-25T14:07:02Z
- **Tasks:** 3/3
- **Files modified:** 5 implementation/test files, plus this summary and the broken-windows ledger

## Accomplishments

- Added two CGO-free Darwin builds, one universal binary, thin/universal tar.gz archives, and SHA-256 `checksums.txt`.
- Added an independent verifier that requires exactly one regular `zcode-acp` member, rejects unsafe member sets/types before extraction, stages private copies, and compares exact `lipo -archs` sets without executing a binary.
- Added offline static release contracts and guarded traversal, absolute-path, symlink, extra-member, and missing-binary fixtures.
- Reworked `release.yml` into a read-only macOS snapshot job plus a tag-only publish job with independent frozen-module and offline gates.
- Kept `contents: write` and `GITHUB_TOKEN` confined to the publish job; the snapshot job uploads all three archives and checksums without a token.
- Added `/dist/` to `.gitignore`.

## Verification Evidence

### Passed checks

Direct tools used:

```text
/Users/nil/.local/share/mise/installs/go/1.27.4/bin/go -> go1.27.4 darwin/arm64
/opt/homebrew/bin/node -> v26.13.1
/opt/homebrew/bin/bash -> GNU bash 5.3.20
/tmp/zcode-acp-release-tools.DrrlmY/bin/goreleaser -> GitVersion v2.20.2, GoVersion go1.27.4
```

The exact GoReleaser binary was installed through `GOPROXY=https://proxy.golang.org` into a fresh temporary GOBIN. No mise trust setting was changed.

| Command / check | Result |
|-----------------|--------|
| `/opt/homebrew/bin/node --test experiments/release.test.mjs` | PASS — 4 pass, 1 intentional guarded skip |
| `ZCODE_ACP_RELEASE_FIXTURES=1 ZCODE_ACP_RELEASE_BASH=/opt/homebrew/bin/bash node --test --test-name-pattern='artifact verifier' experiments/release.test.mjs` | PASS — 7/7, including all five unsafe fixtures |
| `/opt/homebrew/bin/node --test experiments/` | PASS — 46 pass, 2 intentional guarded skips, 0 fail |
| `/opt/homebrew/bin/bash -n scripts/verify-darwin-release.sh` | PASS |
| Direct Go/Node/Bash/GoReleaser pin comparisons | PASS — 1.27 / 26 / 5.3 / exact 2.20.2 |
| `GOPROXY=off goreleaser check` with exact 2.20.2 | PASS |
| `GOPROXY=https://proxy.golang.org go mod download` + `git diff --exit-code HEAD -- go.mod go.sum` | PASS |
| `GOPROXY=off go mod verify` + offline `go mod tidy` + frozen-pin assertions | PASS |
| `GOPROXY=off goreleaser release --snapshot --clean --skip=publish` on final commit `84a2812` | PASS |
| `/opt/homebrew/bin/bash scripts/verify-darwin-release.sh dist` | PASS — checksums, exact members/types, and all architecture sets |
| `git diff --check` | PASS |

Final snapshot artifacts:

| Archive | SHA-256 | Member | `lipo -archs` |
|---------|---------|--------|---------------|
| `zcode-acp_0.23.0-SNAPSHOT-84a2812_darwin_arm64.tar.gz` | `06b3f379d64f5ee16b0031a88da4945ad36935fb9217a9906ab56a79590df223` | one regular `zcode-acp` | `arm64` |
| `zcode-acp_0.23.0-SNAPSHOT-84a2812_darwin_amd64.tar.gz` | `87a9ad043fa1d48ba3c68802ae04e80918b7c8173e6cebd87c8b5c58a17c2ab0` | one regular `zcode-acp` | `x86_64` |
| `zcode-acp_0.23.0-SNAPSHOT-84a2812_darwin_universal.tar.gz` | `bfa6f6954529663187428fffb399660230d2ee79c53f39934d141c22eeff2248` | one regular `zcode-acp` | `x86_64 arm64` |

`shasum -a 256 -c checksums.txt` reported `OK` for all three archives, both against `dist/` and the verifier's private staged copies.

### Pending checks — not claimed as passed

| Check | Status / reason |
|-------|-----------------|
| Release commands through `mise exec` / `mise run` | NOT RUN — repository `mise.toml` is explicitly untrusted. Exact absolute-tool equivalents above passed. Recorded as WINDOWS entry 28. |
| Hosted `workflow_dispatch` snapshot observation | NOT RUN — no branch was pushed. Confirm read-only rights, bounded module hydration, offline checksum/tidy, three archives plus checksums, verifier success, and publish-job skip. Recorded as WINDOWS entry 29. |

No exact-tool or local snapshot check remains blocked. No tag was created or pushed, and no release was published.

## Task Commits

1. **Task 1: Build and verify one complete binary-only Darwin release slice** — `968305a` (`feat(11-05)`)
   - `.goreleaser.yaml`, `scripts/verify-darwin-release.sh`, `experiments/release.test.mjs`, `.gitignore`
2. **Task 2: Automate the proven snapshot as a read-only macOS job** — `e02a266` (`feat(11-05)`)
   - `.github/workflows/release.yml`, `experiments/release.test.mjs`
3. **Task 3: Gate tag publication on the verified snapshot and write-scoped rights** — `84a2812` (`feat(11-05)`)
   - `.github/workflows/release.yml`, `experiments/release.test.mjs`

**Plan metadata:** committed separately with this summary and `.planning/WINDOWS.md`.

## Files Created/Modified

- `.goreleaser.yaml` — exact-version two-target/universal build, binary-only archives, and SHA-256 config.
- `scripts/verify-darwin-release.sh` — checksum, exact-member/type, private-copy, extraction, and `lipo` verification.
- `experiments/release.test.mjs` — offline config/verifier/workflow contracts and guarded unsafe archive fixtures.
- `.github/workflows/release.yml` — read-only snapshot and tag-only write-scoped publication.
- `.gitignore` — ignores GoReleaser `/dist/` output.
- `.planning/WINDOWS.md` — records the resolved schema adaptation and pending external observations.

## Decisions Made

- Used `archives.ids`, the current GoReleaser 2.20.2 field, instead of deprecated `archives.builds`.
- Used a non-matching extra-file sentinel glob so default README/LICENSE members are suppressed while archive IDs select the built binary.
- Added a second checksum pass over private staged copies to close the dist-to-extraction race.
- Made publication depend on the verified snapshot and required `github.event_name == 'push'` plus a `refs/tags/v*` ref.

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

**Total deviations:** 2 auto-fixed Rule 1 bugs.
**Impact on plan:** Both fixes were required for correct release behavior and test precision. No architectural change, new dependency, ACP method, client feature, Linux behavior change, or live-service access was introduced.

## Issues Encountered

- The repository mise config remains untrusted by explicit constraint. The implementation uses mise in CI as planned, while local certification used the required absolute Go/Node/Bash paths and an isolated exact GoReleaser 2.20.2 binary.
- Hosted Actions observation is pending a branch push. This is recorded as an open `unrun-verify`, not represented as green.

## Authentication Gates

None.

## User Setup Required

None for implementation. No account, credential, Keychain, desktop, paid service, or `api.z.ai` access was used. The exact GoReleaser tool was acquired only through the approved Go module proxy into `/tmp/zcode-acp-release-tools.DrrlmY/bin`.

## Known Stubs

None. The guarded fixture test is intentionally skipped by default and was explicitly executed with `ZCODE_ACP_RELEASE_FIXTURES=1`.

## Threat Flags

None beyond the plan's existing trust boundaries. The verifier never executes artifacts; snapshot work has read-only rights and no token; write rights and `GITHUB_TOKEN` are tag-publish-only; module access is bounded to the approved public Go proxy before offline builds.

## Next Phase Readiness

- The local release slice is complete and proven with the exact pinned GoReleaser version.
- The workflow is ready for a hosted `workflow_dispatch` observation after the phase branch is pushed.
- `.planning/STATE.md` and `.planning/ROADMAP.md` were intentionally not updated, per the execution request.

## Self-Check: PASSED

- All five implementation/test files, the broken-windows ledger, and this summary exist on disk.
- Task commits `968305a`, `e02a266`, and `84a2812` are ancestors of HEAD; the measured task range contains exactly three commits.
- The final exact-version snapshot, all three checksums, exact archive member/type checks, all `lipo` architecture sets, static tests, and adversarial fixtures passed.
- Summary and ledger edits pass `git diff --check`.
- `.planning/STATE.md` and `.planning/ROADMAP.md` retain their pre-execution SHA-256 hashes and are absent from the plan commit range.
- The pre-existing untracked `.planning/milestone.lock` was left untouched.
