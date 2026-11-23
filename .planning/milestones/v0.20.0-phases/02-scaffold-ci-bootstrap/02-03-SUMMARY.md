---
phase: 02-scaffold-ci-bootstrap
plan: 03
subsystem: infra
tags: [bash, grep, secret-scan, ci-gate, node, provenance, docs]

# Dependency graph
requires:
  - phase: 01-research-discovery
    provides: experiments/ gate scripts + scrubber (CRED_RX/TOKEN_RX proven shapes), dialect doc with [V-*] provenance grammar, 01-REVIEW findings WR-01..04/IN-01..04
  - phase: 02-scaffold-ci-bootstrap plan 02-02
    provides: scrubber WR-03/IN-04 fixes (sibling pair; file-disjoint)
provides:
  - scan-secrets.sh with CRED_RX-aligned credential class + quote-free token pattern + toplevel pin (CI-bound in plan 02-04)
  - check-docs.sh toplevel pin (runs green from any CWD inside the repo)
  - app-server-probe.mjs capture header deriving desktopVersion from --desktop-version (default unknown)
  - dialect doc re-derivable recipes (leading-dash-safe error-code grep, digit-inclusive method class, v4 extractor) + corrected singletons (9) label
affects: [02-04 (CI promotion adopts the corrected gate), 02-05 (protection contexts), phase-3+ codec work reading the dialect doc]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 3221        # 12884 diff chars / 4 over the 4 changed files
  tasks: 2
  commits: 2          # MEASURED: git rev-list --count 40efc075..527e683 (production commits)
plan_head_before: 40efc075e10f709278dfc340701b8515e36fd4bf
plan_head_after: 527e683eba5da45428ff73a767613f17073ed675

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Gate scripts pin to repo toplevel (cd \"$(git rev-parse --show-toplevel)\") — CWD-independent by contract"
    - "One definition of a credential shape: gate pattern class byte-identical to the scrubber's CRED_RX"
    - "Capture headers record values read at capture time (builtinRevision + desktopVersion) — never constants"

key-files:
  created: []
  modified:
    - experiments/scan-secrets.sh
    - experiments/check-docs.sh
    - experiments/app-server-probe.mjs
    - docs/protocol/app-server-dialect.md

key-decisions:
  - "WR-04 implemented as review option (b): --desktop-version CLI arg, default 'unknown' — no new discovery logic; empty value normalized to 'unknown' so the header never asserts an unread value"
  - "Token pattern drop of the closing quote is scan-side only — mirrors TOKEN_RX family per WR-01 sibling alignment"
  - "Singletons label arithmetic re-verified before edit: namespace rows sum to 101, +9 singletons = 110 headline"

patterns-established:
  - "Toplevel pin one-liner is the shared IN-03 fix for every bash gate entering CI"
  - "Doc recipes must be leading-dash-safe (grep -o -e '-NNNN') — bare '-NNNN' is silently parsed as GNU grep context options"

requirements-completed: [SCAFF-02]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Secret gate credential class byte-identical to the scrubber CRED_RX ([A-Za-z0-9+/=_:.-]{20,}) with quote-free keyed-token pattern and updated header comment (WR-01)"
    requirement: SCAFF-02
    verification:
      - kind: other
        ref: "grep -cF '[A-Za-z0-9+/=_:.-]{20,}' experiments/scan-secrets.sh (=1) && bash experiments/scan-secrets.sh (exit 0)"
      - kind: other
        ref: "grep -cF 'git rev-parse --show-toplevel' experiments/scan-secrets.sh experiments/check-docs.sh (=2)"
    human_judgment: false
  - id: D2
    description: "Both gate scripts pinned to repo toplevel — green from repo root and from a docs/ subdirectory (IN-03 x2)"
    requirement: SCAFF-02
    verification:
      - kind: integration
        ref: "cd docs && bash ../experiments/scan-secrets.sh && bash ../experiments/check-docs.sh (rc=0)"
    human_judgment: false
  - id: D3
    description: "Probe capture header derives desktopVersion from --desktop-version (default 'unknown'); hardcoded '3.18.2' literal removed (WR-04)"
    requirement: SCAFF-02
    verification:
      - kind: other
        ref: "node --check experiments/app-server-probe.mjs (exit 0); grep -cF -- '--desktop-version' (=2); grep -qF \"desktopVersion: '3.18.2'\" inverted (absent)"
    human_judgment: false
  - id: D4
    description: "Dialect doc recipes re-derivable: leading-dash-safe error-code grep with GNU-grep caveat, digit-inclusive method class + v4 extractor, singletons (9) label (WR-02/IN-01/IN-02)"
    requirement: SCAFF-02
    verification:
      - kind: other
        ref: "grep -cF \"grep -o -e '-32603'\" / '\"[a-zA-Z0-9/]*\"$' / '\"v4/[^\"]*\"' docs/protocol/app-server-dialect.md (all =1); grep -cF 'singletons (9)' (=1)"
      - kind: other
        ref: "bash experiments/check-docs.sh && bash experiments/check-docs.sh --phase-gate (both exit 0)"
    human_judgment: false

# Metrics
duration: 5min
completed: 2026-11-19
status: complete
---

# Phase 2 Plan 3: Review-Fix Hardening (D-05 non-TDD findings) Summary

**Secret gate aligned byte-for-byte with the scrubber's proven credential shapes and pinned CWD-independent; probe header reads desktopVersion at capture time; dialect doc recipes now re-derivable (leading-dash-safe grep, digit-inclusive classes)**

## Performance

- **Duration:** 5 min
- **Started:** 2026-11-19T14:02:41Z
- **Completed:** 2026-11-19T14:07:28Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- WR-01 closed: `scan-secrets.sh` credential-blob class widened to `[A-Za-z0-9+/=_:.-]{20,}` (byte-identical to the scrubber's CRED_RX at `scrub-capture.mjs:52`) and the keyed-token pattern drops its closing-quote requirement (TOKEN_RX family alignment) — a credential in a shape the scrubber's test suite proves real can no longer slip the CI gate plan 02-04 promotes
- IN-03 closed on both gates: `cd "$(git rev-parse --show-toplevel)" || exit 1` pins each script to the repo toplevel; both verified green from the repo root AND from `docs/` — the gate no longer depends on the invocation directory
- WR-04 closed (option b): `--desktop-version` CLI argument (default `'unknown'`, empty normalized to `'unknown'`) replaces the hardcoded `'3.18.2'` in the probe's capture header, mirroring the existing builtinRevision capture-time pattern
- WR-02/IN-01/IN-02 closed in `docs/protocol/app-server-dialect.md`: leading-dash-safe error-code recipe (`grep -o -e '-32603' …`) with the GNU-grep numeric-context caveat, method-extraction class widened to `'"[a-zA-Z0-9/]*"$'` plus the one-line v4 extractor, and the singletons row label corrected to 9 (rows sum 101 + 9 = 110 headline)

## Task Commits

Each task was committed atomically:

1. **Task 1: Gate scripts — WR-01 pattern alignment + IN-03 toplevel pin** - `8959c9d` (fix)
2. **Task 2: Probe header honesty (WR-04) + dialect doc recipes (WR-02/IN-01/IN-02)** - `527e683` (fix)

## Files Created/Modified
- `experiments/scan-secrets.sh` - CRED_RX-aligned credential class, quote-free token pattern, toplevel pin, header comment updated; rc-accumulator/SECRET SHAPE/exit contract untouched
- `experiments/check-docs.sh` - identical toplevel pin after `set -u`; fail()/rc contract, REQUIRED_SECTIONS map, check logic untouched
- `experiments/app-server-probe.mjs` - `--desktop-version` in parseArgs defaults+map+usage, empty-value normalization, header field now `flags.desktopVersion`
- `docs/protocol/app-server-dialect.md` - three recipe corrections + label fix; `[V-*]` provenance grammar and `<bundle>` placeholder preserved

## Decisions Made
- WR-04 implemented as review-disposition option (b) per plan: CLI argument with `'unknown'` default — no new bundle-discovery logic; an empty-string argument also normalizes to `'unknown'` (header-honesty contract)
- Header comment of `scan-secrets.sh` now documents the separator-inclusive body and quote-free token match while keeping the existing doc contract (shapes match VALUES, `.planning` excluded, Phase-2 CI note)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- None. All verifies passed on first run: gates green from root and `docs/`, `node --check` green, all four doc greps hit, `check-docs.sh` conformance + `--phase-gate` exit 0, `node --test experiments/` 18/18 green (no collateral from this plan).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `scan-secrets.sh` and `check-docs.sh` are promotion-ready for plan 02-04's CI `secret-scan` job (corrected gate, CWD-independent — CI runners can invoke from any directory)
- D-05 review-fix set is complete once paired with 02-02's WR-03/IN-04 TDD fixes (already landed: `b886a32`/`52f660a`)
- Plan 02-05's protection contexts can rely on the check names as documented; no rename of gate scripts occurred

## Self-Check: PASSED

- SUMMARY.md exists on disk at `.planning/phases/02-scaffold-ci-bootstrap/02-03-SUMMARY.md`
- Task commits verified as ancestors of HEAD: `8959c9d` (Task 1), `527e683` (Task 2)
- Metadata commits verified: `e28ed40` (SUMMARY), `20f6611` (tracking)
- STATE.md advanced to Plan 4 of 5; ROADMAP.md shows 02-03 complete (3/5 summaries)
- No stubs, no auth gates, no deviations; SCAFF-02 correctly left unmarked (workflows land in 02-04)

---
*Phase: 02-scaffold-ci-bootstrap*
*Completed: 2026-11-19*
