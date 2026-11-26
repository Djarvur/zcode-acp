---
phase: 11-darwin-build-test-release-gates
plan: "03"
subsystem: infra
tags: [go, docs-gate, secret-scan, mise, tool-versions, github-actions, ci]
requires:
  - phase: 11-02
    provides: Portable detached-process fixtures and the static portability test seam
provides:
  - Go documentation conformance command with phase-gate behavior
  - Go tracked-file secret scanner with redacted diagnostics
  - Four-tool mise-resolved version gate with no Bash dependency
  - Go-backed named gates in the canonical CI graph
affects: [11-04, 11-05, MBLD-02]
actuals:
  tokens: 24523
  tasks: 2
  commits: 4
  plan_head_before: ae80f8bd615f3214745b719d9fb3aba9438dc4c8
  plan_head_after: 16516c5
tech-stack:
  added: []
  patterns:
    - Go commands own standard gates through portable filesystem and process APIs
    - Four mise-managed tool pins with no Bash tool or environment requirement
    - Secret findings redact matched values while preserving file and line diagnostics
key-files:
  created:
    - cmd/check-docs/main.go
    - cmd/check-docs/main_test.go
    - cmd/scan-secrets/main.go
    - cmd/scan-secrets/main_test.go
    - scripts/check-mise-tool-versions.mjs
    - experiments/tool-versions.test.mjs
    - .planning/phases/11-darwin-build-test-release-gates/11-03-SUMMARY.md
  modified:
    - mise.toml
    - experiments/portability.test.mjs
    - experiments/scrub-capture.test.mjs
    - .github/workflows/pr-sanity.yml
    - docs/protocol/app-server-dialect.md
    - docs/protocol/install-discovery.md
    - experiments/README.md
    - scripts/vm-e2e.sh
    - .planning/WINDOWS.md
  removed:
    - experiments/check-docs.sh
    - experiments/scan-secrets.sh
key-decisions:
  - Replace the docs and secret shell gates with Go commands instead of retaining compatibility wrappers.
  - Keep the docs gate's CHECK FAIL / PENDING output and accumulated exit behavior.
  - Preserve the three credential regexes and tracked-file scope, but redact matched values from reports.
  - Keep only Go, golangci-lint, GoReleaser, and Node in the mise tool gate.
  - Run check-docs and secret-scan through `go run` from named mise tasks.
patterns-established:
  - Standard gate commands use Go stdlib filesystem APIs and project-level `go test` coverage.
  - Secret scanning uses `git ls-files` for tracked scope and never emits credential material.
  - Tool-version enforcement covers only tools the mise registry actually manages.
requirements-completed: [MBLD-02]
coverage:
  - id: D1
    description: Go documentation gate preserves conformance, phase-gate, path, tag, capture-header, and key-minting checks
    requirement: MBLD-02
    verification:
      - kind: unit
        ref: go test ./cmd/check-docs
        status: pass
      - kind: other
        ref: mise run check-docs
        status: pass
    human_judgment: false
  - id: D2
    description: Go secret gate preserves credential patterns and tracked scope while redacting findings
    requirement: MBLD-02
    verification:
      - kind: unit
        ref: go test ./cmd/scan-secrets
        status: pass
      - kind: other
        ref: mise run secret-scan
        status: pass
    human_judgment: false
  - id: D3
    description: Offline comparators reject wrong exact versions and wrong version lines
    requirement: MBLD-02
    verification:
      - kind: unit
        ref: node --test experiments/tool-versions.test.mjs
        status: pass
      - kind: unit
        ref: experiments/portability.test.mjs#tool-version gate exports comparators and fails non-zero on drift
        status: pass
    human_judgment: false
  - id: D4
    description: Four-tool runtime gate and Go-backed CI tasks execute successfully
    requirement: MBLD-02
    verification:
      - kind: other
        ref: mise exec -- node scripts/check-mise-tool-versions.mjs
        status: pass
      - kind: other
        ref: mise run ci
        status: pass
    human_judgment: false
duration: 6min plus trusted runtime corrections
completed: 2026-11-25
status: complete
---

# Phase 11 Plan 03: Go Documentation and Secret Gates Summary

**The documentation and secret gates now run as Go commands with unit coverage, and the managed toolchain contains only Go, golangci-lint, GoReleaser, and Node.**

## Performance

- **Initial duration:** 6 minutes
- **Started:** 2026-11-25T13:26:50Z
- **Initial completion:** 2026-11-25T13:32:38Z
- **Go-gate migration completed:** 2026-11-25T18:29:50Z
- **Tasks:** 2/2
- **Files modified:** 21 implementation, test, workflow, and documentation files across the final migration

## Accomplishments

- Ported all documentation checks to `cmd/check-docs`: legend uniqueness, deliverable presence, provenance tags and grammar, capture-header continuity, exact phase-gate headings, user-home path scan, and key-minting scan.
- Ported the tracked-file secret gate to `cmd/scan-secrets` with the same JWT, encrypted-credential, and token-value regexes.
- Kept the reviewer-safe contract: findings report `path:line:[REDACTED]` or `Binary file <path> matches`, never the credential value.
- Removed `experiments/check-docs.sh` and `experiments/scan-secrets.sh`; no compatibility wrappers remain.
- Removed the Bash tool/environment requirement and reduced `REQUIRED_TOOLS` to Go, golangci-lint, GoReleaser, and Node.
- Routed `check-docs` and `secret-scan` through `go run` in `mise.toml`; `mise run ci` now executes the Go commands.
- Removed every Homebrew Bash install/prepend step and all `mise exec -- bash` calls from the workflows.

## Verification Evidence

```text
go version go1.27.4 darwin/arm64
golangci-lint 2.15.0
goreleaser 2.20.2
node 26.2.0 via mise
```

| Command | Result |
|---------|--------|
| `go test ./...` | PASS |
| `go test ./cmd/check-docs ./cmd/scan-secrets` | PASS |
| `mise install` | PASS — four tools |
| `mise exec -- node scripts/check-mise-tool-versions.mjs` | PASS — four mise pins |
| `mise run check-docs` | PASS |
| `mise run secret-scan` | PASS |
| `mise run ci` | PASS — lint 0 issues, Go race suite green, Node suite green |
| `node --test experiments/*.test.mjs` | PASS — 47 pass, 1 guarded skip |
| `git diff --check` | PASS |

No Bash installation, Bash tool pin, Bash environment requirement, or standard-gate Bash invocation remains. No real ZCode desktop, Keychain, credential, paid API, or product service was accessed.

## Task Commits

1. **Task 1: Portable shell gates** — `aa9930c` (`feat(11-03)`)
2. **Task 2: Tool and shell version requirements** — `3630083` (`feat(11-03)`)
3. **Trusted-runtime correction** — `3017d64` (`fix(11-03)`)
4. **User-directed Go gate migration** — `16516c5` (`feat(11)`)

**Plan metadata:** committed separately with this summary and the broken-windows ledger.

## Files Created/Modified

- `cmd/check-docs/main.go` and `main_test.go` — full documentation gate plus fixtures for every check.
- `cmd/scan-secrets/main.go` and `main_test.go` — tracked-file scanner with redaction and adversarial fixtures.
- `mise.toml` — Go-backed gate tasks, four tool pins, and no Bash requirement.
- `scripts/check-mise-tool-versions.mjs` — four-tool mise-resolved comparator and runtime gate.
- `experiments/portability.test.mjs` — static absence of shell gates/Bash and Go task contracts.
- `experiments/tool-versions.test.mjs` — offline comparator coverage.
- `.github/workflows/pr-sanity.yml` — Go-backed tasks with no Bash setup.
- `experiments/README.md`, protocol docs, and `scripts/vm-e2e.sh` — updated gate commands.
- `.planning/WINDOWS.md` — updated stale gate references and fixed the executed runtime-gate entry.

## Decisions Made

- Go commands, not wrappers, are the only production gate implementations.
- The docs gate may inspect repository files only through Go stdlib filesystem APIs; it must not spawn grep or find.
- The secret gate uses `git ls-files` for tracked scope and redacts matched values unconditionally.
- Bash is not part of the managed toolchain; the version gate checks only registry-managed tools.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed a self-scan collision from the portability test** — `aa9930c`
**2. [Rule 1 - Bug] Rejected four-component version tokens** — `3630083`
**3. [Rule 1 - Bug] Replaced the unsupported mise Bash tool pin** — `3017d64` (superseded by the Go migration)
**4. [Rule 1 - Bug] Fixed the Node experiment task for the pinned line** — `3017d64`

### User-directed Architectural Correction

**5. [User-directed] Remove Bash from the standard dev/CI toolchain**
- **Found during:** User review after trusted runtime verification
- **Issue:** Bash was a managed/installed dependency even after the PATH workaround; the user rejected it as a toolchain dependency.
- **Fix:** Ported docs, secret, and release verification to Go commands, deleted all three shell scripts, removed Bash from mise/workflows, and updated static contracts and documentation.
- **Verification:** Go unit tests, full `go test ./...`, `mise run ci`, Node suite, and real release verification pass.
- **Committed in:** `16516c5`

---

**Total deviations:** 4 auto-fixed Rule 1 bugs and 1 user-directed architectural correction.
**Impact on plan:** The final migration removes the entire Bash dependency while preserving output, exit, security, and release-verification contracts.

## Issues Encountered

The original Bash design failed real execution twice before being superseded: mise had no Bash registry plugin, and the mise-selected Node line rejected a directory argument. The final Go migration eliminates both classes of toolchain assumption.

## Authentication Gates

None.

## User Setup Required

None. Repository trust was provided for verification; no Bash installation or environment variable is required.

## Known Stubs

None.

## Next Phase Readiness

- Standard gates consume only Go, Node, and mise-managed registry tools.
- The four-tool version gate, Go docs/secret commands, and canonical CI graph are green.
- Release verification is owned by `/Users/nil/DiskD/W/Djarvur/zcode-acp-go/cmd/verify-darwin-release`; Plan 05 evidence is updated separately.
- No real desktop, Keychain, credential, paid API, or product service was accessed.

## Self-Check: PASSED

- Go implementation and test files, summary, and ledger exist.
- Commits `aa9930c`, `3630083`, `3017d64`, and `16516c5` are ancestors of HEAD.
- Full Go tests, `mise install`, four-tool gate, `mise run ci`, and Node suite pass.
- No Bash tool/environment pin, Homebrew Bash step, shell gate script, or standard-gate Bash invocation remains.
- `.planning/STATE.md` and `.planning/ROADMAP.md` are untouched by this correction.
