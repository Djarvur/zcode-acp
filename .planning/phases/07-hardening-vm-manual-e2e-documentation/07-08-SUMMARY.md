---
phase: 07-hardening-vm-manual-e2e-documentation
plan: 08
subsystem: docs
tags: [readme, provenance, launch-recipe, acp, omp-card, documentation]

# Dependency graph
requires:
  - phase: 07-hardening (07-04)
    provides: the omp CLI dispatch (`zcode-acp omp`) and the 18.1.9+zacp.0 version stamp the recipe's floor claim documents
  - phase: 07-hardening (07-05)
    provides: the /model slash layer with the byte-exact refusal strings the recipe cites
  - phase: 07-hardening (07-06)
    provides: TestConformanceV1Arc — the shipped-behavior backing for the README's honesty claims
  - phase: 07-hardening (07-07)
    provides: scripts/vm-e2e.sh — the clean-machine reproduce path the README points at
  - phase: 06-oauth-coding-plan-path-model-picker
    provides: the four keyring probe sentinels, the coding-plan credential posture (the desktop login IS the credential), the PAT mint chain the provenance statement describes
  - phase: 01 (research)
    provides: docs/protocol/* (DISC-01..04) + docs/research/reference-adapters.md (DISC-05) — the provenance legend definition site and the Apache-2.0 attribution record cited
provides:
  - README.md as the English user surface (DOCS-01): what-it-is, Install, Configuration (the real flag/env table + both auth paths), Troubleshooting (the four keyring sentinels verbatim, the 18.1.9 floor, the 10s close-toast note, Start-Plan headless, scripts/vm-e2e.sh reproduce), with the CI/required-checks/branch sections preserved byte-identical
  - the README launch-recipe section (DOCS-02): the per-agent command override mechanics (tokenizer, VAR=val rejection), the version floor with a self-check command, env per auth path (agentDefaultEnv OR the two-line PATH shim, placeholder-only), launchesWithModel:true + live /model switching with the byte-exact refusal strings, PI_NO_PTY ignored, the agentCmdOverrides persistence known-unknown — QA-03-ready (ONE recipe + ONE card config per path)
  - docs/provenance.md (DOCS-03): the provenance statement (no patching, no attestation forgery, official PAT chain, clean ACP surface), the Apache-2.0 attribution naming the four ported files, and the known limitations
affects: [07-09 (the QA-03 manual run follows the recipe verbatim), verify-work (DOCS-01/02/03 routing), the repo's public front door]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 4072      # 16288 diff chars / 4 over README.md + docs/provenance.md
  tasks: 3
  commits: 3        # MEASURED: git rev-list --count dec15d96..HEAD

# Tech tracking
tech-stack:
  added: []         # documentation only — no new dependencies (T-07-SC: no package installs)
  patterns:
    - "facts-only docs rule: every README/provenance claim read back from shipped code before writing (sentinels from keyringprobe.go verbatim, refusal strings from front.go:526-527, close budget lifecycle.go:50, version stamp main.go:41)"
    - "placeholder-only credential discipline in examples: env-var NAMES in prose, <your-key> placeholders, the shim sources a user-owned env file instead of embedding a value"

key-files:
  created:
    - docs/provenance.md
  modified:
    - README.md

key-decisions:
  - "Shim example sources $HOME/.config/zcode-acp/env (a user-owned file) rather than inlining any value — the placeholder-only prohibition honored mechanically, not just by convention"
  - "Provenance statement avoids the reference's read-only PAT posture claim — this repo's patminter supersedes it (in-file header names the create-if-absent decision), so the doc claims only the official-chain fact"
  - "Recipe keeps agentDefaultEnv at mechanism level (no invented JSON schema) since the dialect doc pins the settings-key name, not the value shape"

patterns-established:
  - "Docs gate parity: every docs task commit re-ran check-docs.sh + scan-secrets.sh; the whole-docs --phase-gate ran after the new docs/ file landed"

requirements-completed: [DOCS-01, DOCS-02, DOCS-03]

# Coverage metadata (#1602) — per-deliverable traceability for verify-work routing.
coverage:
  - id: D1
    description: "README user surface: what-it-is / Install / Configuration (both auth paths) / Troubleshooting, with CI + branch sections preserved"
    requirement: DOCS-01
    verification:
      - kind: other
        ref: "grep -c '## Install' README.md (1 >= 1)"
        status: pass
      - kind: other
        ref: "grep -c 'ZCODE_ACP_API_KEY' README.md (3 >= 1)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh && bash experiments/scan-secrets.sh (both exit 0)"
        status: pass
      - kind: other
        ref: "diff of '^## CI$'..$ against pre-rewrite HEAD: byte-identical"
        status: pass
    human_judgment: false
  - id: D2
    description: "Launch recipe section: override mechanics, version floor + self-check, env per auth path (shim/agentDefaultEnv, placeholder-only), launchesWithModel + /model refusals, PI_NO_PTY, persistence known-unknown"
    requirement: DOCS-02
    verification:
      - kind: other
        ref: "grep -c 'zcode-acp omp' README.md (4 >= 1)"
        status: pass
      - kind: other
        ref: "grep -c '18.1.9' README.md (5 >= 1)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh && bash experiments/scan-secrets.sh (both exit 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "docs/provenance.md: provenance statement (no patching, no forgery, official PAT chain), Apache-2.0 attribution naming the ported surface, known limitations"
    requirement: DOCS-03
    verification:
      - kind: other
        ref: "grep -c 'Apache-2.0' docs/provenance.md (3 >= 1)"
        status: pass
      - kind: other
        ref: "grep -c '## Provenance legend' docs/provenance.md (0 == 0, single-definition-site rule)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh --phase-gate && bash experiments/scan-secrets.sh (both exit 0)"
        status: pass
    human_judgment: false

# Metrics
duration: 8min
completed: 2026-11-22
status: complete
plan_head_before: dec15d965d99717347365d8e3fb511d50bb12d19
plan_head_after: c99004e040e3bae138b47e8ad25df1f7aff23458
---

# Phase 7 Plan 08: VM & Manual E2E, Documentation Summary

**The English user surface ships: README rewritten around the real flag/env surface and both auth paths, the Orca launch recipe with verbatim card mechanics, and docs/provenance.md carrying the honesty contract and Apache-2.0 attribution — all gates green.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-11-22T14:25:07Z
- **Completed:** 2026-11-22T14:32:38Z
- **Tasks:** 3
- **Files modified:** 2 (README.md rewritten head; docs/provenance.md new)

## Accomplishments
- README is the user front door: the honest ACP v1 server intro (any ACP client, ZCode desktop the single backend, no-patching/no-forgery pointer at docs/provenance.md), Install (static binary, runtime deps scoped per auth path), Configuration (the flag/env table from cmd/zcode-acp/main.go plus BOTH auth paths), Troubleshooting (the four keyring sentinels verbatim, the 18.1.9 floor, the 10s close-toast note, Start-Plan headless, scripts/vm-e2e.sh) — CI/required-checks/branch sections preserved byte-identical
- The launch recipe is self-sufficient for preset-agent-only hosts: ONE override string (`/path/to/zcode-acp omp`) + ONE env mechanism per auth path, with the tokenizer's VAR=val rejection motivating the shim/agentDefaultEnv choice, the version floor self-checkable via `zcode-acp omp --version`, launchesWithModel:true claimable (07-04 shipped the binding), live /model switching with the byte-exact refusal strings, PI_NO_PTY ignored, and the agentCmdOverrides persistence path labeled a known-unknown — QA-03-ready
- docs/provenance.md publishes the honesty contract (bundle never patched, no attestation forgery, the quota counter simply not reported, the PAT minted via the official api.z.ai chain as the desktop's own binary signs it, the ACP surface kept clean), the Apache-2.0 attribution naming the four ported files and their in-file headers, and the known limitations — the legend stays single-definition-site (referenced, never redefined)

## Task Commits

Each task was committed atomically:

1. **Task 1: Tracer — README user surface** - `91a5920` (docs)
2. **Task 2: Launch recipe section (DOCS-02)** - `9c5b849` (docs)
3. **Task 3: docs/provenance.md (DOCS-03)** - `c99004e` (docs)

**Tracer gate:** auto mode (`auto_advance: true`) + automated-only `<verify>` → verify re-run end-to-end post-commit: all green — `⚡ Tracer verified end-to-end — expanding`.

## Files Created/Modified
- `README.md` — the English user surface (what/install/configure/troubleshoot + launch recipe), CI and branch-protection sections untouched
- `docs/provenance.md` — provenance statement, Apache-2.0 attribution, known limitations, protocol-notes index

## Decisions Made
- The shim example sources a user-owned env file (`$HOME/.config/zcode-acp/env`) instead of embedding a value — the placeholder-only prohibition (T-07-22) honored mechanically
- The provenance statement stays at "official api.z.ai chain exactly as the desktop's own binary signs it" and makes no read-only claim — this repo's patminter carries the create-if-absent supersession in its in-file header, so claiming the reference's read-only posture would be an unshipped-behavior claim (T-07-24)
- `agentDefaultEnv` documented at mechanism level only — the dialect doc pins the settings-key name, not the value shape; inventing a JSON schema would be fabrication

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None material — baselines matched the plan's acceptance anchors (both grep baselines were 0 at start; both gates were green pre-edit).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 07-09 (the final plan) can run the QA-03 manual E2E following the README recipe verbatim — ONE recipe + ONE card config per auth path, everything else pre-verified
- The phase-close gates this plan owns are all green: check-docs.sh (incl. --phase-gate), scan-secrets.sh, mise run lint (0 issues)
- No stubs, no skipped verifications, no deferred items introduced

---
*Phase: 07-hardening-vm-manual-e2e-documentation*
*Completed: 2026-11-22*
