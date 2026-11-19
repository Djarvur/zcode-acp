---
phase: 02-scaffold-ci-bootstrap
plan: 01
subsystem: infra
tags: [go, mise, golangci-lint, lint-gate, toolchain-pins, branch-protection, ci]

# Dependency graph
requires:
  - phase: 01-research-protocol
    provides: experiments/ node:test suite (the test-experiments gate target), repo conventions (gate scripts, docs layout)
provides:
  - "mise.toml — single source of tool versions (go 1.27 line, golangci-lint EXACTLY 2.15.0, goreleaser 2.20.2, node 26) + gate tasks build/vet/lint/test/test-experiments/tidy-diff + ci aggregate"
  - ".golangci.yml — goswitch strict v2 config, one functional delta (goimports.local-prefixes -> github.com/Djarvur/zcode-acp)"
  - "go.mod — module github.com/Djarvur/zcode-acp, go 1.27 (adjacent to the mise pin)"
  - "cmd/zcode-acp/main.go — compiling skeleton, --version prints zcode-acp 0.0.0-dev"
  - "Proven-in-both-directions lint gate: skeleton green (0 issues), planted violation red (3 named issues), pin exactly 2.15.0"
affects: [02-04 CI workflows call these mise tasks, 02-05 protection requires the gate proven, 03+ every feature plan lives under this gate]

# Actuals (#2632) — pairs with the plan's estimate (35000) to calibrate future estimates.
actuals:
  tokens: 2366          # chars/4 over the realized diff (9465 diff bytes, 264 insertions)
  tasks: 2
  commits: 1            # MEASURED: git rev-list --count gsd-plan-head-before-02-01..HEAD
plan_head_before: 0452456c87adc492a0b4e79e05ef4dbde8441c9f
plan_head_after: 91bee67a7bf85b59bd20e973e335123817f112b4

# Tech tracking
tech-stack:
  added: [go 1.27 (mise line), golangci-lint 2.15.0 (exact pin), goreleaser 2.20.2 (inert until Phase 7), node 26 LTS]
  patterns:
    - "mise single-source-of-versions: every gate is `mise run <task>`; CI must never re-pin a tool"
    - "copied-file header provenance: adapted files carry an English header naming origin + exact deltas"
    - "Go strict-lint style: fmt.Fprint* with `_, _ =` discard (forbidigo), blank line before top-of-block return (nlreturn)"

key-files:
  created:
    - mise.toml
    - .golangci.yml
    - go.mod
    - cmd/zcode-acp/main.go
  modified:
    - .gitignore

key-decisions:
  - "golangci-lint pinned EXACTLY 2.15.0 (D-02 deliberate tightening over goswitch's floating major '2') — only an exact pin reproduces lint results; verified via `mise exec -- golangci-lint --version`"
  - ".golangci.yml copied byte-for-byte with exactly one functional change (goimports.local-prefixes, D-03); goswitch-specific inert exclusion rules kept per the D-03 lock; header provenance comment names the goswitch origin and the delta"
  - "Go strict-lint style established from the first Go file: fmt.Fprint* family with explicit error discard, blank line before every top-of-block return"
  - "[tasks.ci] depends order build -> vet -> lint -> test -> test-experiments is the canonical gate order CI (02-04) mirrors"
  - "Commits landed directly on master: sequential executor on the main working tree per orchestrator dispatch; branching_strategy=none (user config) and D-01 sequences protection LAST (02-05), matching 100% of repo history. Recommend git.allow_default_branch_commits:true in .planning/config.json to encode the pre-protection window for plans 02-02..02-04"

patterns-established:
  - "Gate order: build -> vet -> lint -> test -> test-experiments (tidy-diff in CI's sanity job); `mise run ci` is the full local gate"
  - "Version-adjacency predicate: mise [tools].go and go.mod `go` directive must agree exactly on the 1.27 line"
  - "Lint style: fmt.Fprint* + `_, _ =` discard; nlreturn blank line; multi-sentence `// Command <name>` doc comment (house style per GS:cmd/skillgen/main.go)"

requirements-completed: [SCAFF-01]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "mise.toml with all four D-02 pins (go 1.27, golangci-lint 2.15.0 exact, goreleaser 2.20.2, node 26) and gate tasks incl. test-experiments and the ci aggregate in gate order"
    requirement: SCAFF-01
    verification:
      - kind: other
        ref: "grep -cF 'golangci-lint = \"2.15.0\"' mise.toml -> 1 (and go/goreleaser/node pins present, lines 9/13/17/19)"
        status: pass
      - kind: other
        ref: "mise run build && mise run vet && mise run lint -> rc=0, '0 issues.'"
        status: pass
      - kind: other
        ref: "mise exec -- golangci-lint --version -> 2.15.0"
        status: pass
    human_judgment: false
  - id: D2
    description: ".golangci.yml strict lint gate — goswitch config with exactly one functional delta (goimports.local-prefixes -> github.com/Djarvur/zcode-acp)"
    requirement: SCAFF-01
    verification:
      - kind: other
        ref: "diff goswitch/.golangci.yml .golangci.yml -> only the provenance header comment + line-178 local-prefixes differ; all lint semantics byte-identical"
        status: pass
      - kind: other
        ref: "grep -cF 'github.com/Djarvur/zcode-acp' .golangci.yml -> >=1"
        status: pass
      - kind: other
        ref: "mise run lint -> exit 0, '0 issues.' on the four-file skeleton"
        status: pass
    human_judgment: false
  - id: D3
    description: "go.mod declares module github.com/Djarvur/zcode-acp with go 1.27, adjacent to the mise pin (no require block; deps arrive Phases 3-4)"
    requirement: SCAFF-01
    verification:
      - kind: other
        ref: "grep -cF 'go 1.27' go.mod -> 1; mise [tools].go = \"1.27\" (adjacency predicate holds)"
        status: pass
      - kind: other
        ref: "mise run tidy-diff -> rc=0, no go.mod/go.sum drift"
        status: pass
    human_judgment: false
  - id: D4
    description: "cmd/zcode-acp/main.go compiling skeleton answering --version with the exact string zcode-acp 0.0.0-dev"
    requirement: SCAFF-01
    verification:
      - kind: other
        ref: "go run ./cmd/zcode-acp --version -> stdout exactly 'zcode-acp 0.0.0-dev', rc=0"
        status: pass
      - kind: unit
        ref: "mise run test -> rc=0 ('no test files' is not a failure — forward-looking Go gate, probe ledger 'empty' truth)"
        status: pass
    human_judgment: false
  - id: D5
    description: "SCAFF-01 negative proof: any violation fails the lint target; exact pin reproduces; clean tree restored after the probe"
    requirement: SCAFF-01
    verification:
      - kind: other
        ref: "planted 141-char line probe -> golangci-lint rc=1 with 3 named issues (lll: 1, nlreturn: 1, unused: 1) — matches 02-RESEARCH's live observation"
        status: pass
      - kind: other
        ref: "probe directory removed; clean-tree mise run lint -> exit 0, '0 issues.'"
        status: pass
      - kind: other
        ref: "mise exec -- golangci-lint --version -> contains exactly 2.15.0 (D-02 exact pin)"
        status: pass
    human_judgment: false

# Metrics
duration: 11min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 01: Strict-Lint Gate Tracer Summary

**Strict golangci-lint 2.15.0 gate proven green on the zcode-acp module skeleton and red on a planted violation — mise 4-tool pins, goswitch config one-line delta, `--version` answers `zcode-acp 0.0.0-dev`**

## Performance

- **Duration:** 11 min
- **Started:** 2026-11-19T13:32:35Z
- **Completed:** 2026-11-19T13:44:03Z
- **Tasks:** 2
- **Files modified:** 5 (4 created, 1 modified)

## Accomplishments
- The tracer slice is proven end-to-end: `mise run lint` exits 0 with `0 issues.` on the four-file skeleton under golangci-lint exactly 2.15.0 with the goswitch strict config (one functional delta)
- SCAFF-01's negative proof landed: a planted long-line violation makes the gate exit 1 with 3 named issues (`lll`, `nlreturn`, `unused` — identical to 02-RESEARCH's live probe observation), and the clean tree returns to green
- mise.toml is the single source of tool versions (D-02): go 1.27 line, golangci-lint EXACTLY 2.15.0, goreleaser 2.20.2 (inert until Phase 7), node 26; gate tasks build/vet/lint/test/test-experiments/tidy-diff plus the `ci` aggregate in gate order build -> vet -> lint -> test -> test-experiments
- The experiments suite is wired as a first-class gate: `mise run test-experiments` runs the Phase-1 node:test suite green (12 pass / 0 fail)
- Probe ledger truths verified: adjacency (mise go pin ↔ go.mod `go 1.27`, greps = 1/1), empty (`go test ./...` exits 0 on a no-test module), ordering (`[tasks.ci]` depends in enforcement order)

## Task Commits

1. **Task 1: Strict-lint gate green on the module skeleton, end-to-end (tracer)** - `91bee67` (feat)
2. **Task 2: Red-proof the lint gate + exact-pin verification** - no commit (evidence-only task per plan: the deliverable is the observed RED evidence recorded here; the tree was left byte-identical to Task 1's end state)

**Plan metadata:** committed after this SUMMARY (docs commit).

## Files Created/Modified
- `mise.toml` - toolchain pins (D-02) + gate tasks + ci aggregate; header cites the single-source-of-versions rationale
- `.golangci.yml` - goswitch strict v2 config; exactly one functional change (goimports.local-prefixes, D-03); provenance header names origin + delta
- `go.mod` - `module github.com/Djarvur/zcode-acp` + `go 1.27`; no require block yet
- `cmd/zcode-acp/main.go` - compiling skeleton; `--version` prints `zcode-acp 0.0.0-dev`; lint-clean house style
- `.gitignore` - added `/zcode-acp` build residue rule (see Deviations 2)

## Decisions Made
- **Exact linter pin (D-02):** `golangci-lint = "2.15.0"` — deliberate tightening over goswitch's floating `"2"`; the strict linter set churns and only an exact pin reproduces results. Verified live: `mise exec -- golangci-lint --version` reports 2.15.0 (built with go1.27.0, 2026-10-16)
- **Config copy discipline (D-03):** goswitch-specific exclusion rules (e.g. `internal/engine/conn.go` G404, `test/e2e/.*` rules) are inert path patterns here and were kept verbatim per the "changed ONLY in local-prefixes" lock; the exhaustruct deprecation warning observed at run time is a warning, not a failure — the exact pin is the mitigation (RESEARCH Pitfall 6)
- **Canonical gate order:** `[tasks.ci]` depends lists build -> vet -> lint -> test -> test-experiments; this is the order CI (02-04) mirrors
- **Commit window:** production commits landed directly on `master` — sequential executor on the main working tree per the orchestrator dispatch; `branching_strategy: "none"` is the project's committed config and D-01 sequences branch protection as the phase's LAST action (02-05), so the pre-protection window is by design (100% of repo history is direct-to-master). The commit-guard's protected-branch FATAL (#3819) was deliberately overridden by this dispatch; recommend setting `git.allow_default_branch_commits: true` in `.planning/config.json` to encode the window for plans 02-02..02-04

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Probe directory renamed `.golangci-red-probe/` -> `red-probe/`**
- **Found during:** Task 2 (red-proof)
- **Issue:** The plan's verify recipe plants the violation in a dot-prefixed directory. Directories beginning with `.` are invisible to the Go toolchain: `golangci-lint run .golangci-red-probe/...` exited 5 with `Running error: context loading failed: no go files to analyze` — non-zero, but a loading failure, NOT a violation catch. Accepting it would have faked the negative test.
- **Fix:** Re-planted the identical 130+ char code line in `red-probe/` (non-dotted, throwaway). The package loaded and the gate bit honestly: rc=1 with 3 named issues (`lll: 1, nlreturn: 1, unused: 1`) — exactly the issue set 02-RESEARCH observed live. A second probe variant (used 141-char string) confirmed `lll` itself fires under this config.
- **Files modified:** none that survive — probe created then `rm -rf`'d; never committed
- **Verification:** probe rc=1 with named issues; probe dir absent; clean-tree `mise run lint` exit 0
- **Committed in:** no commit (evidence-only)

**2. [Rule 3 - Blocking] Added `/zcode-acp` to .gitignore**
- **Found during:** Task 1 (post-build untracked-file check)
- **Issue:** `go build ./...` writes the executable to the repo root when the module has a single main package (`go help build`: a single main package's build is written out; multiple packages are discarded). The binary `zcode-acp` appeared as untracked residue after every `mise run build`, and would pollute `git status` until Phase 3 adds more packages.
- **Fix:** One .gitignore rule (`/zcode-acp`) with a comment explaining the single-main-package behavior; removed the stray binary from the working tree.
- **Files modified:** .gitignore
- **Verification:** `git status --short` shows no non-planning residue after gates run
- **Committed in:** `91bee67` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking, Rule 3) + 1 documented plan-text tension (below, no code impact)
**Impact on plan:** Both auto-fixes were required to make Task 2's negative test honest and keep the tree clean. No scope creep.

### Documented plan-text tension (no code impact)
- The plan's acceptance criterion says the .golangci.yml diff "shows one hunk", while the plan's own `<action>` (and PATTERNS' shared "Copied-file header provenance" pattern) mandates adapting the header provenance comment to name the goswitch origin. Resolved in favor of the action text: the actual diff is 2 hunks — the 3-line provenance header (comments only) and the line-178 `local-prefixes` change. The FUNCTIONAL config delta is exactly one line; every lint semantic is byte-identical to goswitch's config.

## Issues Encountered
- `mise install` fetched goreleaser 2.20.2 and node 26.13.1 (go 1.27 line and golangci-lint 2.15.0 were already cached) — no errors.
- The exhaustruct deprecation warning appears on every lint run (expected, RESEARCH Pitfall 6): it is a warning, not a failure; the D-02 exact pin is the mitigation. No config changes were made (D-03 forbids linter changes).

## Known Stubs

Intentional, plan-documented placeholders (the plan's own deliverable shape; not accidental gaps):

| File | What | Why intentional | Resolved by |
|------|------|-----------------|-------------|
| `cmd/zcode-acp/main.go:14` | `var version = "0.0.0-dev"` dev-fallback | Stamping-contract placeholder: goreleaser stamps a real value on release builds; `go install` builds honestly report the dev fallback (house style per GS:cmd/goswitchd/version.go) | Phase 7 release flow |
| `cmd/zcode-acp/main.go` | `main()` implements only `--version` | The plan's objective IS the skeleton: "the real CLI and agent implementation arrive with the later phases" (D-06) | Phase 3-4 CLI/agent plans |

## User Setup Required

None - no external service configuration required. (`mise install` fetched all four pinned tools from official channels; T-2-SC mitigation honored — no registry packages involved.)

## Next Phase Readiness
- The gate every later phase lives under is proven in both directions; plans 02-02/02-03 (gate scripts) and 02-04 (workflows calling exactly these mise tasks) can build on verified task names and the exact pin
- 02-04 must mirror the `[tasks.ci]` gate order and keep check names aligned with protection contexts (RESEARCH Pitfall 3)
- The direct-to-master commit window closes when 02-05 applies protection; plans 02-02..02-04 should keep landing there per D-01's sequencing

## Self-Check: PASSED

- `mise.toml`, `.golangci.yml`, `go.mod`, `cmd/zcode-acp/main.go` exist on disk (FOUND)
- Commit `91bee67` is an ancestor of HEAD (FOUND); ledger `gsd-plan-head-before-02-01` = `0452456` (FOUND); measured commits 0452456..HEAD = 1
- All verify commands re-run green end-to-end after Task 2 (tracer feedback gate): build/vet/lint rc=0 + `0 issues.`, version string exact, test rc=0, test-experiments 12 pass / 0 fail, tidy-diff rc=0, pin 2.15.0
- `git status --porcelain` outside `.planning/`: clean (no probe residue)

---
*Phase: 02-scaffold-ci-bootstrap*
*Completed: 2026-11-19*
