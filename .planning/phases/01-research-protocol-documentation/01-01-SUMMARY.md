---
phase: 01-research-protocol-documentation
plan: 01
subsystem: docs-tooling
tags: [provenance, jsonl, app-server, secret-scanning, capture-pipeline, workspaceKey]

# Dependency graph
requires:
  - phase: research (pre-planning)
    provides: validated harness pattern (01-RESEARCH.md Pattern 2), scrub rule set (Pattern 3), source anchors for workspaceKey (3.17.1 clone), pristine 3.18.2 bundle
provides:
  - Wave-0 validation gates: experiments/check-docs.sh (structure + provenance conformance, --phase-gate) and experiments/scan-secrets.sh (secret-shape gate) — reused at every later commit and by Phase 2 CI
  - The single provenance legend (docs/protocol/README.md) indexing all six deliverable docs (D-04)
  - Canonical scrub-first capture pipeline: experiments/scrub-capture.mjs (+ 11 unit tests) and experiments/app-server-probe.mjs scenario-flow harness — reused unchanged by plans 01-02..01-06
  - DISC-04 deliverable: docs/protocol/workspace-key.md (wire schema, derivation convention, experiment record, host-freedom note) with V-SRC + V-EXP provenance
  - Scrubbed live capture experiments/captures/EXP-WORKSPACEKEY-01.jsonl (seed for Phase 3 fixtures)
  - New 3.18.2 wire facts for DISC-01: session/requestRuntimePreferences host obligation, interaction/requestOfficialMcpAuthHeaders during create, process/mcpTelemetry notifications, session/lifecycle/changed envelope
affects: [01-02 E1/E2 experiments, 01-03 app-server-dialect doc, 01-04, 01-05, 01-06, Phase 2 CI secret gate, Phase 3 golden fixtures]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff).
actuals:
  tokens: 31934        # 127737 diff bytes / 4
  tasks: 3
  commits: 6

commits: 6
plan_head_before: 3e4e6a453e15a481cab89fe975cabcb68db1f08c
plan_head_after: e7c7745f81a5efa415f354f1b574ded4694fac5c

# Tech tracking
tech-stack:
  added: []   # zero external packages (per D-02/RESEARCH: node:test + grep + jq only)
  patterns:
    - claim-ledger docs with inline V-SRC/V-EXP/V-RPT tags resolved by check-docs.sh
    - scrub-first captures: raw born outside the repo, scrubber is the only path into experiments/captures/, fixed-point idempotency
    - stable per-file UUID placeholders (nth distinct -> nth) preserving referential integrity for golden fixtures
    - capture headers pin builtinRevision + date at capture time (stale-evidence guard)
    - harness answers server->host requests and records its own replies as hostSent lines

key-files:
  created:
    - experiments/check-docs.sh
    - experiments/scan-secrets.sh
    - experiments/scrub-capture.mjs
    - experiments/scrub-capture.test.mjs
    - experiments/app-server-probe.mjs
    - experiments/README.md
    - docs/protocol/README.md
    - docs/protocol/workspace-key.md
    - experiments/captures/EXP-WORKSPACEKEY-01.jsonl
    - .planning/phases/01-research-protocol-documentation/01-01-red-evidence.json
  modified:
    - .gitignore

key-decisions:
  - "Sequential execution committed to the configured branch (master) per the orchestrator's sequential-mode directive; branching_strategy is none and the repo's full history follows this pattern"
  - "Scrubber takes private-location prefixes via repeatable --redact-path at run time instead of hardcoding them (neutral-phrasing rule compliance)"
  - "check-docs.sh skips angle-bracket placeholder arguments in V-EXP resolution (legend definition-site forms are not experiment ids)"
  - "Harness wraps each captured wire line as {ts, line} JSON — preserves wire text for Phase 3 fixture reconstruction while adding turn-timing data for E2"
  - "Host-obligation answers kept minimal (session/requestRuntimePreferences -> {nativeSearchEnhancementsEnabled:false, memoryEnabled:false}) per the 3.17.1 reference fixture; everything else -32601, mirroring the fixture host"

patterns-established:
  - "Per-task gate run: bash experiments/check-docs.sh && bash experiments/scan-secrets.sh before every commit"
  - "TDD RED evidence via TAP + check tdd-red-evidence (RED_EVIDENCE_OK) before any feat commit for a tdd=true task"
  - "Experiment records: raw outside repo -> scrub -> fixed-point check -> commit; capture header pins revision at capture time"

requirements-completed: [DISC-04]

coverage:
  - id: D1
    description: "Wave-0 validation gates: experiments/check-docs.sh (conformance + --phase-gate) and experiments/scan-secrets.sh (secret-shape gate)"
    requirement: D-02/D-04
    verification:
      - kind: integration
        ref: "bash experiments/check-docs.sh (exit 0; PENDING lines for unwritten docs)"
        status: pass
      - kind: integration
        ref: "bash experiments/scan-secrets.sh (exit 0)"
        status: pass
      - kind: integration
        ref: "bash experiments/check-docs.sh --phase-gate (exit non-zero, 4 PENDING at plan end)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Provenance legend defined exactly once in docs/protocol/README.md + index of the six deliverable docs"
    requirement: D-04
    verification:
      - kind: integration
        ref: "bash experiments/check-docs.sh check (a): legend count == 1, located in docs/protocol/README.md"
        status: pass
      - kind: integration
        ref: "grep -c '^## Provenance legend' docs/protocol/README.md == 1"
        status: pass
    human_judgment: false
  - id: D3
    description: "Canonical capture scrubber (experiments/scrub-capture.mjs) covering every D-02 scrub rule, TDD RED before GREEN"
    requirement: D-02
    verification:
      - kind: unit
        ref: "node --test experiments/ — 11 pass / 0 fail (RED evidence RED_EVIDENCE_OK: TAP 1..11, 11 ERR_ASSERTION failures before implementation)"
        status: pass
      - kind: integration
        ref: "fixed-point check: scrub(scrub(raw)) == scrub(raw) on the live EXP-WORKSPACEKEY-01 capture (cmp identical)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Scenario-flow app-server probe harness (experiments/app-server-probe.mjs) with probe + workspacekey flows and 01-02 dispatch seam"
    requirement: D-05
    verification:
      - kind: integration
        ref: "node --check experiments/app-server-probe.mjs (exit 0)"
        status: pass
      - kind: e2e
        ref: "live workspacekey flow run: 37 wire lines captured, clean exit 0, header written with builtinRevision 44"
        status: pass
      - kind: integration
        ref: "guard smoke tests: missing-flag usage exit 1, unknown-flow exit 1, raw-out-inside-repo refused"
        status: pass
    human_judgment: false
  - id: D5
    description: "Scrubbed live capture EXP-WORKSPACEKEY-01 (path-as-key accepted + arbitrary-host-string key accepted)"
    requirement: DISC-04
    verification:
      - kind: e2e
        ref: "jq over committed capture: id=1 reply result.session present with workspaceKey==workspacePath; id=2 reply result.session present with workspaceKey=probe-arbitrary-host-key-01"
        status: pass
      - kind: integration
        ref: "forbidden-shape sweep over capture: 0 JWT/credential-blob/UUID matches; header line 1 byte-intact"
        status: pass
    human_judgment: false
  - id: D6
    description: "DISC-04 deliverable docs/protocol/workspace-key.md (wire schema, derivation convention, experiment record, host-freedom note, sources & provenance)"
    requirement: DISC-04
    verification:
      - kind: integration
        ref: "bash experiments/check-docs.sh (exit 0 — V-EXP EXP-WORKSPACEKEY-01 resolves, tag grammar, legend-once, path-shape clean)"
        status: pass
      - kind: integration
        ref: "grep checks: 5 required headings; nonEmptyString cited x4; V-SRC citations verified against the 3.17.1 clone by sed before writing (index.ts:2762/:2784, official-mcp-auth.ts:149, entrypoint:219-236)"
        status: pass
    human_judgment: false
  - id: D7
    description: "experiments/README.md ops guide (E1-E5 invocation shapes, scrub-first rules, never-in-CI warning, quota ledger)"
    requirement: D-02
    verification: []
    human_judgment: true
    rationale: "Ops-guide accuracy is prose-level: flag shapes were smoke-tested against the implemented harness guards, but no automated gate enforces doc-content correctness — verifier should skim the guide against app-server-probe.mjs --help behavior."

# Metrics
duration: 29min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 01-01: Documentation-Pipeline Walking Skeleton Summary

**Walking skeleton proven end-to-end: secret-shape gates + provenance legend + scrub-first capture pipeline delivered, and one real registry-level experiment (E3) confirmed on the live 3.18.2 binary that `workspaceKey` accepts both the workspace path and an arbitrary host-chosen string — with the full pipeline (source → experiment → scrubbed capture → tagged doc → gates green) exercised exactly as plans 01-02…01-06 will reuse it.**

## Performance

- **Duration:** 29 min
- **Started:** 2026-11-19T05:22:09Z
- **Completed:** 2026-11-19T05:51:41Z
- **Tasks:** 3 of 3
- **Files modified:** 11 (10 created, 1 modified)

## Accomplishments
- Wave-0 validation infrastructure green: `check-docs.sh` (structure + provenance-tag conformance + `--phase-gate` honestly reporting 2/6) and `scan-secrets.sh` (value-shape gate that provably catches violations — it failed once on a real defect during development and was fixed)
- Provenance legend defined exactly once in `docs/protocol/README.md` with the D-01 ground-truth hierarchy; six-doc index naming each DISC id
- Scrub-first capture pipeline TDD'd RED→GREEN (11 rule tests, `RED_EVIDENCE_OK` machine gate) and validated on a real capture (fixed point, zero forbidden shapes)
- E3 run against the shipped 3.18.2 bundle: **path-as-key accepted and arbitrary-host-string key accepted** (host-opaque semantics) — DISC-04 satisfied with a resolving V-EXP id plus version-pinned V-SRC citations

## Task Commits

Each task was committed atomically:

1. **Task 1: Validation gates + provenance legend (tracer)** - `e08ff0a` (feat)
2. **Task 2: Capture pipeline — scrubber RED** - `7be801f` (test), **GREEN** - `b418c25` (feat), **harness** - `08381db` (feat)
3. **Task 3: E3 + workspace-key.md (DISC-04)** - `e7c7745` (feat)

**Deviation fix:** `94912e6` (fix — Rule 3, see below)
**Plan metadata:** committed with SUMMARY/STATE/ROADMAP/REQUIREMENTS (docs)

## Files Created/Modified
- `experiments/check-docs.sh` — structure + provenance conformance gate; `--phase-gate` requires all six docs (heading table hardcoded from plans 01-01…01-06)
- `experiments/scan-secrets.sh` — secret-shape gate over tracked files (JWT / credential-blob / token-keyed values; `.planning` excluded)
- `experiments/scrub-capture.mjs` — canonical scrubber; the only path into `experiments/captures/`
- `experiments/scrub-capture.test.mjs` — 11 node:test rule tests (fixtures string-concatenated so no forbidden literal is tracked)
- `experiments/app-server-probe.mjs` — scenario-flow spawn/JSONL harness (machine paths via CLI args only; stdin held open; capture header pins revision at capture time; answers host requests; `subscribe`/`turn` seam for 01-02)
- `experiments/README.md` — ops guide: E1–E5 shapes, scrub-first rules, never-in-CI warning, quota ledger (real turns spent: 0)
- `docs/protocol/README.md` — six-doc index + the single provenance legend (D-04)
- `docs/protocol/workspace-key.md` — DISC-04: wire schema, derivation convention (verbatim source quote), experiment record, host-freedom note, sources & provenance
- `experiments/captures/EXP-WORKSPACEKEY-01.jsonl` — scrubbed live capture (52 lines, header intact)
- `.gitignore` — defensive `experiments/raw/` entry (D-02)

## Decisions Made
- Sequential-mode commits land on the configured branch (`master`) per the orchestrator directive; `branching_strategy: none` and the repo's entire history follow this pattern (guarded per-commit: no detached HEAD, no destructive ref ops)
- Private-location redaction is supplied at run time via repeatable `--redact-path` (never hardcoded — the neutral-phrasing rule applies to scripts too)
- The harness records its own host-obligation replies in the capture as `hostSent` lines — the capture is honest about what the host did vs what the server said
- Captured wire lines are wrapped as `{"ts":…,"line":…}`: wire text stays extractable for Phase 3 fixtures, timestamps serve E2 turn-timing analysis
- check-docs treats angle-bracket V-EXP arguments as legend placeholders, not ids (definition-site forms don't need to resolve)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] E3 first run: harness did not answer server→host requests**
- **Found during:** Task 3 (E3 live run)
- **Issue:** `session/create` on 3.18.2 issues the server→host **request** `session/requestRuntimePreferences` (scope `runtime-materialization`); with no answer the internal protocol client closes and both creates fail `-32603 "ZCode Protocol client connection closed"` — the planned verdict (result.session.sessionId) was unreachable
- **Fix:** harness answers host requests — minimal preferences result for `session/requestRuntimePreferences` (per the 3.17.1 reference host fixture `mcp-apps-host-fixture/agent.ts:82-86` and the strict result schema), `-32601` for anything else; replies recorded as `hostSent` capture lines; second E3 run completed with both creates accepted
- **Files modified:** `experiments/app-server-probe.mjs`
- **Verification:** live re-run — both `session/create` replies carry `result.session` (sessionKind interactive, status starting→ready)
- **Committed in:** `94912e6`
- **Bonus wire truth (feed to plan 01-03):** the discovery also surfaced `interaction/requestOfficialMcpAuthHeaders` requests (12 during two creates, answered -32601 without breaking the flow), `process/mcpTelemetry` notifications, and the `session/lifecycle/changed` envelope — all recorded in the committed capture

**2. [Rule 3 - Blocking issue] Scrubber could not write its canonical output on first use**
- **Found during:** Task 3 (scrub step)
- **Issue:** `experiments/captures/` did not exist; `writeFileSync` failed ENOENT — the "only path into captures/" was broken on its canonical invocation
- **Fix:** `mkdirSync(path.dirname(outFile), { recursive: true })` before writing
- **Files modified:** `experiments/scrub-capture.mjs`
- **Verification:** scrub of the live capture succeeded; `node --test experiments/` 11/11
- **Committed in:** `94912e6`

**Total deviations:** 2 auto-fixed (2 x Rule 3). **Impact on plan:** positive — deviation 1 converted a would-be false verdict into the plan's first new 3.18.2 wire fact (host-obligation on create) and hardened the harness for plans 01-02…01-06.

## Issues Encountered
- The `session/requestRuntimePreferences` discovery (see deviation 1) — resolved in-plan by fixing the harness; the unanswered-request failure mode is now documented in `docs/protocol/workspace-key.md` Experiment record for reuse by DISC-01
- None otherwise — gates, TDD machine evidence, and the live experiment all behaved as designed

## Authentication Gates
None — E3 is registry-level; no credentials were read, no model turn was made, no quota was spent.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plans 01-02…01-06 reuse the pipeline unchanged: gates (`check-docs.sh`, `scan-secrets.sh`), scrubber, and harness are in place; the `subscribe`/`turn` flow seam is pre-registered (capture ids `EXP-SUBSCRIBE-01`/`EXP-TURN-01` mapped)
- `session/requestRuntimePreferences` must be answered by E1/E2 flows — the harness now does this automatically; E1 should record whether `state.updated` follows `provider/refreshRegistry` (research assumption A3)
- Quota ledger stands at "real turns spent: 0" — plan 01-02 updates it after E2's single tiny turn
- New server→host surface recorded in `EXP-WORKSPACEKEY-01` (requestRuntimePreferences, requestOfficialMcpAuthHeaders, mcpTelemetry, lifecycle/changed) is directly consumable by plan 01-03's method-map and server→host sections
- Wave-0 checklist in `01-VALIDATION.md` is fully satisfied (all five Wave-0 artifacts exist and are green)

---
*Phase: 01-research-protocol-documentation*
*Completed: 2026-11-19*

## Self-Check: PASSED

- All 10 plan artifacts verified present on disk (9 production files + this SUMMARY)
- All 7 commits verified as ancestors of HEAD: e08ff0a, 7be801f, b418c25, 08381db, 94912e6, e7c7745 (production, measured from ledger 3e4e6a4 = 6) + aaa258f (plan metadata)
- Gates re-verified green post-commit: check-docs.sh exit 0 (2/6 honest PENDING under --phase-gate), scan-secrets.sh exit 0, node --test 11/11
- TDD gate compliance: test(01-01) RED commit precedes feat(01-01) GREEN commit (7be801f → b418c25); RED evidence RED_EVIDENCE_OK; no refactor commit (optional, none needed)
