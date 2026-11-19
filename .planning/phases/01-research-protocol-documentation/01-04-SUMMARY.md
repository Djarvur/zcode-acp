---
phase: 01-research-protocol-documentation
plan: "04"
subsystem: docs
tags: [acp, protocol-documentation, omp-card, chunk-reverification, provenance, session-format]

# Dependency graph
requires:
  - phase: 01-research-protocol-documentation (plan 01-01)
    provides: scrub-first capture pipeline, validation gates (check-docs.sh / scan-secrets.sh), provenance legend in docs/protocol/README.md
provides:
  - "docs/protocol/acp-client-dialect.md — DISC-02 deliverable: omp preset-agent identity, card registry of this build, launch contract, version floor, load-bearing method sets, full method range, Hb string contracts, client capabilities, statically-resolved open items"
  - "experiments/captures/EXP-SESSIONDIR-01.jsonl — D-06 re-verification + E5 static evidence (string literals + chunk anchors only, tier-compliant)"
  - "Registry verdict: 8 ACP cards unchanged; NO native zcode chat card; 25-id scan canon delta recorded"
  - "Session-file format spec (omp/pi): --<sanitized-cwd>-- slug + <ts>_<uuid>.jsonl, Claude-style JSONL records"
affects: [phase 4 launch contract, phase 7 conformance scenario and live integration, phase 3 fixtures, SESS-03 slash contracts]

# Actuals (#2632)
actuals:
  tokens: 11169
  tasks: 2
  commits: 2

commits: 2
plan_head_before: ec7b42198401ad9c558f3f9d586b55ac38d5b2b1
plan_head_after: 6937f8a3c6149792cfc4272f7ca2c05d7605e886

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Claim-ledger doc with per-claim chunk-filename citations (V-SRC from a fresh extract, never volatile mounts)"
    - "Static chunk reads recorded as extracted string literals + anchors only (chunk contents never committed)"

key-files:
  created:
    - docs/protocol/acp-client-dialect.md
    - experiments/captures/EXP-SESSIONDIR-01.jsonl
  modified: []

key-decisions:
  - "Assumption A2 upgraded by fresh evidence: the client's --version probe under prefixed agentCmdOverrides is <program> <prefixArgs...> --version (no card args) — source-verified, not left V-RPT; the live half stays Phase 7"
  - "No native zcode chat card exists in this build (zcode is an operations-side hooks id only) — omp stays the only viable preset-agent identity"
  - "Card-registry delta clarified: the ACP card registry is unchanged at 8 cards; the ~22-25 id growth lives in the session-history scan canon and operations-only id families"
  - "E5 verdict: a Go agent CAN write omp-compatible history files (format fully specified), but v1 does not advertise loadSession — compatible files are a Phase 7 nicety, not a v1 gate"

patterns-established:
  - "D-06 re-verification pattern: fresh AppImage extract into scratch, grep anchors from 01-RESEARCH.md, cite chunk filename + anchor per claim"
  - "Probe-row resolutions documented in-doc: adjacency (startup vs per-turn method sets kept distinct), empty (-32601 no-loadSession surface, listingNamesConfiguredModel:false), encoding (byte-exact string contracts), ordering (startup order load-bearing, per-turn order user-driven), concurrency (sessionLost strings + retryBusyPrompts documented; beyond that Phase 7)"

requirements-completed: [DISC-02]

coverage:
  - id: D1
    description: "Scrubbed E5 capture experiments/captures/EXP-SESSIONDIR-01.jsonl with header, chunk-anchored string literals, generalized session-dir patterns, and explicit verdicts"
    requirement: DISC-02
    verification:
      - kind: other
        ref: "grep -c '\"capture\":\"EXP-SESSIONDIR-01\"' experiments/captures/EXP-SESSIONDIR-01.jsonl -> 1 (EXPECT == 1)"
        status: pass
      - kind: other
        ref: "grep -c 'clientBuild' experiments/captures/EXP-SESSIONDIR-01.jsonl -> 1 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "grep -c '17\\.0\\.5' experiments/captures/EXP-SESSIONDIR-01.jsonl -> 3 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "bash experiments/scan-secrets.sh -> exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/protocol/acp-client-dialect.md — the DISC-02 deliverable naming the omp identity, floor 17.0.5, launch contract, method sets, string contracts, capabilities, and statically-resolved open items with per-claim provenance"
    requirement: DISC-02
    verification:
      - kind: other
        ref: "bash experiments/check-docs.sh -> exit 0, no CHECK FAIL"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh --phase-gate 2>&1 | grep -c 'PENDING: docs/protocol/acp-client-dialect.md' -> 0"
        status: pass
      - kind: other
        ref: "grep counts in docs/protocol/acp-client-dialect.md: '^## ' = 10, V-SRC = 43, V-RPT = 11, 'V-EXP EXP-SESSIONDIR-01' = 5, '17.0.5' present"
        status: pass
    human_judgment: true
    rationale: "The plan's own disposition (T-1-09, spec-less fallback): citation honesty is judgment-tier and cannot be reduced to a mechanical test — the gates prove structure, tag grammar, id resolution, and secret hygiene, but a human skim of the provenance tags (chunk citations vs claims) is the designed check at /gsd:verify-work"

# Metrics
duration: 27min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 04: ACP Client Dialect (DISC-02) Summary

**omp card contract re-verified from a fresh extract of the installed client build and documented in docs/protocol/acp-client-dialect.md — launch contract, floor 17.0.5 / selectors-from 18.1.9, method sets, verbatim Hb string contracts, and both open items answered statically via EXP-SESSIONDIR-01**

## Performance

- **Duration:** 27 min
- **Started:** 2026-11-19T06:37:40Z
- **Completed:** 2026-11-19T07:04:40Z
- **Tasks:** 2
- **Files modified:** 2 (both created)

## Accomplishments
- All five load-bearing omp contract strings (17.0.5, agentCmdOverrides, acp-command-sessions, launchesWithModel, PI_NO_PTY) re-verified in a fresh AppImage extract and upgraded V-SRC(parallel) → V-SRC with chunk-filename citations
- Launch contract documented end-to-end: `<program> [prefixArgs] acp [--plugin-dir] [--model] --session-dir <dataDir>/acp-command-sessions/omp`, env `PI_NO_PTY=1`, agent home `PI_CODING_AGENT_DIR` → `.omp/agent`, full agentCmdOverrides semantics (prefix args, shell-quoting parse, tilde expansion, metachar rejection, installed-detection satisfied)
- Card registry re-enumerated from this build: ACP card registry unchanged at 8 entries; native-chat handles (pi/claude/codex) and 25-id session-history scan canon identified; NO native zcode chat card (operations-side id only); delta vs the 8-card list recorded
- Both DISC-02 spot-check items answered statically and capture-backed: the `--session-dir` session-file format (slug algorithm verified byte-for-byte against a real local agent home; Claude-style JSONL record shapes; verdict Yes with the loadSession scope note) and omp `modelDiscovery` kind `session` (in-session listing, `listingNamesConfiguredModel:false`)
- DISC-02 deliverable written with all ten required sections, 43 V-SRC chunk citations, 11 V-RPT labels (settings-store location + live probe half stay known-unknowns), 5 V-EXP citations resolving to the committed capture

## Task Commits

Each task was committed atomically:

1. **Task 1: D-06 chunk re-verification + E5 static answers (EXP-SESSIONDIR-01)** - `c8f42a5` (feat)
2. **Task 2: Write docs/protocol/acp-client-dialect.md (DISC-02)** - `6937f8a` (feat)

## Files Created/Modified
- `experiments/captures/EXP-SESSIONDIR-01.jsonl` - scrubbed E5 capture: extracted chunk string literals + chunk-filename anchors (never chunk code), generalized session-dir patterns, verdicts for both open items
- `docs/protocol/acp-client-dialect.md` - the DISC-02 claim-ledger doc (identity, registry, launch contract, version floor, method sets, string contracts, capabilities, open items, provenance)

## Decisions Made
- Assumption A2 upgraded on fresh evidence: the plan expected the `--version` probe shape under prefixed overrides to stay V-RPT, but the chunk read verifies the probe is `prefixArgs + --version` without card args — recorded as V-SRC in doc and capture, with the live end-to-end half explicitly kept V-RPT for Phase 7
- Registry expectation corrected honestly: the plan's "~22 preset ids in the registry chunk" turned out to be the scan-canon/operations id surface in other chunks; the ACP card registry itself is unchanged at 8 — documented per-family instead of forcing the expected list
- Scope honesty preserved: compatible history files are documented as a Phase 7 nicety (v1 does not advertise `loadSession`; the client throws -32601 on the resume path), not a v1 gate

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Stale plan assumption contradicted by fresh evidence] --version probe shape under prefixed overrides upgraded V-RPT → V-SRC**
- **Found during:** Task 1 (re-verification pass, priority 2)
- **Issue:** The plan mandated recording the probe shape as a known-unknown per assumption A2; the fresh extract contains the probe implementation (`args:[...prefixArgs, '--version']`, 5 s timeout, 4096-byte cap), making the static half of A2 resolvable now
- **Fix:** Documented the verified mechanics in the doc's Version floor section and the capture, explicitly flagged as an A2 upgrade; the live-probe half remains V-RPT (Phase 7)
- **Files modified:** docs/protocol/acp-client-dialect.md, experiments/captures/EXP-SESSIONDIR-01.jsonl
- **Verification:** check-docs.sh green; tags conform to D-04 grammar
- **Committed in:** c8f42a5 / 6937f8a

**2. [Rule 1 - Registry expectation vs build reality] Card-registry delta recorded per-family**
- **Found during:** Task 1 (registry re-enumeration)
- **Issue:** Plan expected ~22 preset ids in the registry chunk (incl. hermes/devin/droid/zcode); those ids are NOT in the chat-card registry — they live in the 25-id session-history scan canon (`session-json-object` chunk) and the operations-only managed-hooks families (`managed-agent-hook-controls` chunk)
- **Fix:** Enumerated all three families separately (8 ACP cards; native-chat handles pi/claude/codex; operations-only ids), answered the native-zcode-card question negatively with evidence, and recorded the delta vs the earlier 8-card list
- **Files modified:** docs/protocol/acp-client-dialect.md, experiments/captures/EXP-SESSIONDIR-01.jsonl
- **Verification:** cross-chunk grep evidence in the capture (`command:`zcode`` occurs nowhere; zcode only in hooks ops + canon)
- **Committed in:** c8f42a5 / 6937f8a

---

**Total deviations:** 2 auto-fixed (2 × Rule 1 evidence-driven documentation corrections; no scope creep, no code)
**Impact on plan:** Both corrections increase provenance honesty — the doc now states build truth instead of the plan's stale expectations. ROADMAP success criterion 2 satisfied as written.

## Issues Encountered
- The shell's `grep` is wrapped by ugrep (complexity-limited for bounded context windows); switched to GNU `/bin/grep` for minified-chunk extraction
- Three bare tag-kind mentions in the doc header tripped check-docs.sh tag grammar on first run; reworded to bracket-free forms (verify-fix loop, same task commit)
- TDD gate checked via `task.is-behavior-adding`: `is_behavior_adding: false` (no tdd="true" tasks, no behavior blocks — documentation plan), so no RED/GREEN cycle applies

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- ROADMAP Phase 1 success criterion 2 satisfied: the dialect doc names the chosen preset-agent identity with its version floor and env quirks, and enumerates both the load-bearing method set (startup/per-turn) and the full range of ACP methods the client sends
- Phase 4/7 consumers can quote this doc: launch contract, override wiring semantics, and string contracts are chunk-cited to the build named in the header
- Phase 7 live concerns remain explicitly listed: settings-store location for `agentCmdOverrides`, live `--version` probe half, interruption semantics beyond the documented strings
- Untracked planning artifacts present in the worktree (`.planning/WINDOWS.md`, `.planning/intel/`, `.planning/state.json`, modified `.planning/config.json`) belong to the orchestrator/parallel executors and were left untouched

---
*Phase: 01-research-protocol-documentation*
*Completed: 2026-11-19*
