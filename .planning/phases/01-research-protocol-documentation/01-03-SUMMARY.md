---
phase: 01-research-protocol-documentation
plan: 03
subsystem: docs-protocol
tags: [app-server, jsonl, method-map, subscribe, terminal-signals, error-codes, drift-baseline, provenance]

# Dependency graph
requires:
  - phase: 01-01
    provides: provenance legend + check-docs/scan-secrets gates, EXP-WORKSPACEKEY-01 (create-time host obligations), pristine-bundle record precedent
  - phase: 01-02
    provides: EXP-SUBSCRIBE-01 (E1 verdicts) + EXP-TURN-01 (turn-start evidence + D-05 degradation record + source-enumeration block), the two DISC-01 assumption mappings (encoding, empty/unknown-method)
  - phase: research (pre-planning)
    provides: Pattern 1 blueprint + method-map extraction technique (RESEARCH Code Examples), 3.17.x file:line citations to re-verify, ARCHITECTURE wire details (serial queue, stop bypass, EOF drain)
provides:
  - DISC-01 deliverable: docs/protocol/app-server-dialect.md — transport facts, full 110-method host→server map + 43 v4 strings (extraction commands + occurrence counts), server→host surface (7 interaction methods + requestRuntimePreferences + notification stream), E1 subscribe verdict table, turn-start signal table (V-EXP) + completion half honestly V-SRC-only, error-code table with bundle counts, session lifecycle (session/delete verifiably absent ×0), snapshot semantics (ack≠acceptance, registry read-back, revision formula), first-class 3.17-vs-3.18.2 drift baseline, Sources & provenance mapping every id
  - Open question 4 resolved: interaction/requestUserInput semantics read from 3.17.1 schemas + broker (ask-user channel, accept/decline/cancel)
  - Phase-consumable anchors: Phase 3 (sections 1/2/6 codec+errors), Phase 4 (sections 4/5 turn machine), Phase 5 (section 3 interaction shapes), Phase 6 (section 8 snapshot/revision)
  - Honest plan correction recorded in-doc: startup/storageState exists in 3.17.1 too (not a drift item) — plan draft claim fixed with evidence
affects: [Phase 3 codec/fake/supervisor, Phase 4 turn state machine, Phase 5 interaction responders, Phase 6 snapshot/revision verification, /gsd:verify-work phase gate (now all-six green)]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff).
actuals:
  tokens: 12273        # 49091 diff bytes / 4 (the one production file)
  tasks: 2
  commits: 2

commits: 2
plan_head_before: 01ce28e3d0029551df14f925dac3fb75894134ad
plan_head_after: 7bc0534d2f2a15f577aa709767c51b6a4dd3e75f

# Tech tracking
tech-stack:
  added: []   # zero external packages (grep + dd byte-offset slices + jq-free bash only)
  patterns:
    - byte-offset slice extraction (grep -bo anchor + dd skip/count) as the fast re-derivable alternative to wide grep -o windows on a 15.7 MB single-line bundle
    - drift verification loop: every bundle method string grepped against the whole 3.17.1 clone; absence recorded as a first-class diff row (4 strings)
    - plan draft claims re-verified before documenting; non-reproducing claims corrected in-doc with the evidence (startup/storageState)
    - degradation honesty carried through: completion-half rows labelled V-SRC-only with the capture's limitation attached

key-files:
  created:
    - docs/protocol/app-server-dialect.md

key-decisions:
  - "Sequential execution committed to the configured branch (master) per the orchestrator's sequential-mode directive; branching_strategy is none and the repo's full history follows this pattern (same guarded pattern as plans 01-01..01-06)"
  - "Bundle evidence taken via byte-offset slices anchored on sessionSubscribe (fast, stable) — the wide grep -o context windows from RESEARCH timed out on the 15.7 MB single line; both command forms recorded in the doc for re-derivability"
  - "startup/storageState drift claim corrected in-doc: present in both 3.18.2 bundle and 3.17.1 source (index.ts:476,4509) — 'new' only relative to prior private research docs; the correction is recorded in section 9 instead of repeating the plan draft"
  - "Transport mechanics documented from 3.17.1 source file:line with bundle-identifier corroboration (shouldBypassProcessingQueue/lastQueuedMessageStarted present in the shipped binary); EOF drain constant stays 3.17.1-only (minified away in bundle) — labeled accordingly"
  - "Error-code advice: match numeric codes, never message text (locale-mixed strings observed in the bundle registry neighborhood)"

patterns-established:
  - "Per-claim tags at line end; V-SRC-only used as plain text (never bracketed) so tag-grammar stays machine-checkable"
  - "Phase-gate discipline: this doc completed the six-doc set — bash experiments/check-docs.sh --phase-gate exits 0 for the whole phase"

requirements-completed: [DISC-01]

coverage:
  - id: T1
    description: "Sections 1-5: transport facts, method map, server-to-host surface, subscribe verdict, turn terminal signals — claim-first with provenance tags"
    requirement: DISC-01
    verification:
      - kind: integration
        ref: "bash experiments/check-docs.sh (exit 0)"
        status: pass
      - kind: integration
        ref: "grep -c 'V-EXP EXP-SUBSCRIBE-01' docs/protocol/app-server-dialect.md = 15 (>=1)"
        status: pass
      - kind: integration
        ref: "grep -c 'V-EXP EXP-TURN-01' docs/protocol/app-server-dialect.md = 30 (>=1)"
        status: pass
      - kind: integration
        ref: "grep -c '^## ' docs/protocol/app-server-dialect.md = 5 at Task 1 (>=5); = 10 after Task 2 (>=10)"
        status: pass
      - kind: integration
        ref: "grep -c '3.18.2' docs/protocol/app-server-dialect.md = 5 (>=1); grep -c 'V-EXP' = 35+ (>=2)"
        status: pass
    human_judgment: true
    rationale: "Prose-level provenance judgment: every bundle count was re-derived live during writing (method map stability slice, error codes, turn.* literals reproduced EXP-TURN-01's block exactly), but the reading of which claims are load-bearing enough to anchor is judgment — verifier should skim sections 2/5."
  - id: T2
    description: "Sections 6-9 + Sources & provenance: error codes, session lifecycle, snapshot semantics, first-class drift baseline, id mapping"
    requirement: DISC-01
    verification:
      - kind: integration
        ref: "grep -c '^## ' docs/protocol/app-server-dialect.md = 10 (>=10 — 9 sections + Sources)"
        status: pass
      - kind: integration
        ref: "grep -c '3.17.1' docs/protocol/app-server-dialect.md = 36 (>=2 version-pinned citations)"
        status: pass
      - kind: integration
        ref: "grep -c 'requestCredentialCipher' docs/protocol/app-server-dialect.md = 6 (>=1); grep -c 'received' = 4 (>=1)"
        status: pass
      - kind: integration
        ref: "grep -c 'V-RPT' docs/protocol/app-server-dialect.md = 19 (!=0)"
        status: pass
      - kind: integration
        ref: "bash experiments/check-docs.sh --phase-gate 2>&1 | grep -c 'PENDING: docs/protocol/app-server-dialect.md' = 0; phase-gate exit 0 (all six docs green)"
        status: pass
      - kind: integration
        ref: "bash experiments/scan-secrets.sh (exit 0)"
        status: pass
      - kind: integration
        ref: "Drift loop re-derivable: all 110 host->server method strings grepped against the whole 3.17.1 clone — exactly 4 absent (requestCredentialCipher, plugins/mentionItems, plugins/readUserOptions, session/releaseUnderDirectory)"
        status: pass
    human_judgment: false

# Metrics
duration: 24min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 01-03: App-Server Dialect Doc (DISC-01) Summary

**The DISC-01 deliverable is complete: `docs/protocol/app-server-dialect.md` documents the shipped 3.18.2 JSONL surface claim-first with per-claim provenance — full method map (110 host→server strings + 43 v4 strings with re-derivable extraction commands and counts), the live E1 subscribe verdict table, the turn-start signal table from EXP-TURN-01 with the completion half honestly V-SRC-only, the bundle-count error-code table, the verifiable absence of `session/delete`, ack-not-acceptance snapshot semantics, and a first-class drift baseline whose headline is `interaction/requestCredentialCipher` (live mid-turn on 3.18.2, absent from the entire 3.17.1 tree) — and with it the six-doc phase gate goes fully green.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-11-19T07:58:49Z
- **Completed:** 2026-11-19T08:22:40Z
- **Tasks:** 2 of 2
- **Files modified:** 1 (created)

## Accomplishments
- **Sections 1–5 (Task 1, `dd76b97`):** transport facts (envelope grammar, `-32600`/`-32700` mechanisms, string-or-int id echoes, serial queue + stop-bypass scheduling, 100 ms EOF drain, encoding-assumption mapping stated); method map extracted from the shipped bundle with commands + stability criterion + per-string counts, including the load-bearing `session/create` drops `options.reasoningLevel` quirk (V-SRC model-mapper/model-selection + V-EXP `-32603` refusal + setModel workaround with `persistAsWorkspaceLastUsed:false` note); server→host surface (requestRuntimePreferences at create AND send, 7 interaction methods, requestUserInput semantics resolving open question 4, requestCredentialCipher mid-turn obligation); E1 verdict table (session/subscribe ACCEPTED `{sessionId, eventSeq, events}`; session/events ACCEPTED; v4/conversation/subscribe REFUSED `-32603` revealing `{topic, connectionId, clientMode}`); turn-start signal table (V-EXP) + completion half V-SRC-only with the degradation limitation attached.
- **Sections 6–9 + Sources (Task 2, `7bc0534`):** error-code table `-32700…-32042` with bundle enum names and occurrence counts + match-codes-not-strings rule; session lifecycle (`session/delete` ×0 occurrences recorded as the evidence for REQUIREMENTS.md's graceful no-op stance; stop bypass vs lifecycle-stop concurrency; create-time host obligations); snapshot semantics (ack `received`/`unchanged` schema from the bundle, ack≠acceptance silent discard at `registry-service.ts:205-212` (3.17.1), registry read-back as the Phase 6 verification, `.strict()` ×1409, revision formula cited to `zcode-builtin-provider-config-source.ts:41,241` + bundle release-store literals); first-class drift baseline (probed copy + pristine cmp re-verified; 4-method diff vs 3.17.1; `-32042`/`-32002` bundle-only; `startup/storageState` correction); Sources & provenance mapping every id with commands and counts.
- **Phase gate:** `bash experiments/check-docs.sh --phase-gate` now exits **0** — all six deliverable docs present and conforming (this doc was the last one).

## Task Commits

1. **Task 1: Sections 1–5** - `dd76b97` (feat)
2. **Task 2: Sections 6–9 + Sources** - `7bc0534` (feat)

**Plan metadata:** committed with SUMMARY/STATE/ROADMAP/REQUIREMENTS (docs).

## Files Created/Modified
- `docs/protocol/app-server-dialect.md` — created (211 lines): the DISC-01 deliverable, nine sections + Sources & provenance, every claim line tagged V-SRC / V-EXP / V-RPT per the legend in docs/protocol/README.md.

## Decisions Made
- Sequential-mode commits land on the configured branch (`master`) per the orchestrator directive — same pattern as plans 01-01…01-06 (`branching_strategy: none`; guarded per-commit: root pin, no detached HEAD, no destructive ops)
- Bundle extraction used `grep -bo` byte-offset anchors + `dd` slices (fast, stable) after the RESEARCH wide-window grep form timed out on the 15.7 MB single-line bundle; both command forms are recorded in section 2 so every count stays re-derivable
- Transport mechanics (serial queue, bypass lane, EOF drain) are documented from 3.17.1 file:line citations with bundle-identifier corroboration where the minified binary still carries the names; the drain constant (minified away) stays labeled 3.17.1-only
- 3.17.1 drift checks ran against the whole clone (not just the protocol index): `requestCredentialCipher` absence is whole-tree; three more absent strings (`plugins/mentionItems`, `plugins/readUserOptions`, `session/releaseUnderDirectory`) surfaced from the same loop
- `V-SRC-only` degradation labels are written as plain text (no brackets) so the machine tag-grammar check stays meaningful

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Plan draft claim "startup/storageState new in 3.18.2" did not re-verify**
- **Found during:** Task 2 (drift-baseline section)
- **Issue:** the plan's task action and must_haves list `startup/storageState` as new in 3.18.2; fresh verification found it present in the 3.17.1 clone too (`index.ts:476` `storageStartup: "startup/storageState"`, notification schema `:4509`) — documenting it as drift would have violated D-01/T-1-08 (evidence honesty)
- **Fix:** section 9 records the correction explicitly ("present in both — 'new' only relative to prior private research docs; not a 3.17→3.18.2 drift item") with both citations; bundle count (×3) and live observation (×5 per boot) unchanged
- **Files modified:** docs/protocol/app-server-dialect.md (section 9)
- **Committed in:** `7bc0534`

**2. [Rule 1 - Bug] RESEARCH wide-window grep form unusable on the minified bundle (timeout)**
- **Found during:** Task 1 (method-map extraction)
- **Issue:** `grep -o '.\{2500\}mcp/list.\{200\}'` on the 15.7 MB single-line bundle exceeded 120 s (bounded repetition backtracking); the documented technique needed a workable variant
- **Fix:** switched to `grep -bo` byte-offset anchor + `dd skip/count` slices (sub-second, same evidence, stability criterion kept: wider slice must add no new method strings); both command forms recorded in the doc's section 2
- **Files modified:** docs/protocol/app-server-dialect.md (section 2 commands)
- **Committed in:** `dd76b97`

**Total deviations:** 2 auto-fixed (2 × Rule 1). **Impact on plan:** neutral-to-positive — the doc is drift-honest where the plan draft was wrong, and the extraction technique is recorded in its actually-working form.

## Issues Encountered
- None blocking. The E2 degradation (terminal envelopes never observed live) was inherited from plan 01-02 by design and is handled exactly as the plan's own fallback prescribes: completion-half rows marked V-SRC-only with the capture's limitation statement attached, quiet-window fallback documented as the proven secondary.

## Authentication Gates
None — synthesis of committed evidence plus read-only bundle/source greps; no credentials read, no model turns, no quota spent (quota ledger unchanged at "3 tiny sends issued, 0 completed").

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- **Phase 3** (codec/fake/supervisor) consumes: section 1 envelope grammar + scheduling (serial queue, bypass lane, concurrency sets), section 2 method constants, section 6 error codes (match codes, not strings); golden fixtures can be seeded from the three committed captures
- **Phase 4** (turn machine) consumes: section 4 verdicts (subscribe-first viable on 3.18.2; v4 channel needs `{topic, connectionId, clientMode}`) and section 5 (turn-start signals live-verified; completion via v4 `turn.completed`/`turn.failed` remains source-level until Phase 6's credential-answering host; quiet-window fallback is the proven secondary)
- **Phase 5** (interactions) consumes: section 3 request shapes incl. the newly documented requestUserInput schema/semantics
- **Phase 6** (snapshot/revision) consumes: section 8 (registry read-back discipline, strict schema, revision formula)
- Wave-2 complete: all six deliverable docs exist; `check-docs.sh --phase-gate` exits 0 — the phase-level check at `/gsd:verify-work` starts from a green gate
- Quota ledger unchanged: "3 tiny sends issued, 0 completed" — Phase 1 spend final

---
*Phase: 01-research-protocol-documentation*
*Completed: 2026-11-19*

## Self-Check: PASSED

- Plan artifact verified present on disk: docs/protocol/app-server-dialect.md (211 lines, 49091 bytes)
- Both task commits verified as ancestors of HEAD: dd76b97 (Task 1), 7bc0534 (Task 2); measured from ledger 01ce28e = 2 production commits (plan_head_before 01ce28e3…, plan_head_after 7bc0534…)
- Gates re-verified green post-commit: check-docs.sh exit 0; check-docs.sh --phase-gate exit 0 (all six docs, 0 PENDING); scan-secrets.sh exit 0
- All plan acceptance criteria re-run and passing (headings 10; V-EXP ids cited and resolving; 3.17.1 citations 36; requestCredentialCipher 6; received 4; V-RPT 19; 3.18.2 5)
- No stub patterns in the doc (the single "placeholder" grep hit is the path-note rule text, same as install-discovery.md's footer)
