---
phase: 01-research-protocol-documentation
plan: "06"
subsystem: docs
tags: [reference-adapters, apache-2.0, attribution, adopt-avoid, credential-chain, porting, provenance, v0.89.0]

# Dependency graph
requires:
  - phase: 01-research-protocol-documentation (plan 01-01)
    provides: validation gates (check-docs.sh / scan-secrets.sh), provenance legend in docs/protocol/README.md
  - phase: parallel research session
    provides: ".planning/research/REFERENCE-ADAPTERS.md — the V-RPT adopt-list claims this plan re-verified live (the lead, not the truth)"
provides:
  - "docs/research/reference-adapters.md — DISC-05 deliverable: 12-row coverage map of william0wang/zcode-acp v0.89.0 verified live (v2-keyring chain, requestCredentialCipher both paths, snapshot revision formula, PAT chain, runtime-headers responder, elicitation/permission gating, requestRuntimePreferences, slash commands, resume/fork, attachments, session lifecycle, interaction robustness) with adopt/avoid verdicts, Apache-2.0 attribution obligations, weaker-adapters table, and the Start-Plan headless limitation note (DOCS-03 input)"
  - "Golden reference set for Phase 6: src/backend/credential-store.ts, src/config/account-provider.ts, src/config/project-token.ts, src/handlers/server-requests.ts, src/backend/client.ts + tests/{credential-cipher,provider-runtime-headers,project-token}.test.ts as the port acceptance oracle"
  - "Drift correction for Phase 6: the reference NEVER mints API keys headless (read-only posture; missing key = fixed project_token_key_missing error + open-the-desktop-once cue) — the research lead's 'create if absent' is wrong for v0.89.0"
  - "Differentiator claim upgraded from reported to verified: jpalmae (Rust) and Liang-HZ (Python) adapters live-confirmed and grep-verified to lack keyring/cipher/PAT coverage"
affects: [phase 6 OAuth chain porting (AUTH-02), phase 7 docs (DOCS-03 Start-Plan limitation), phase 5 interaction responder design]

# Actuals (#2632)
actuals:
  tokens: 4659
  tasks: 2
  commits: 1

commits: 1
plan_head_before: 67ce60ba3f6ca4157aa0868b2fc7e760fc817352
plan_head_after: f53284d71ef0ba5b785cc860dca46f209aeb1111

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Live study without committing: shallow clone pinned to the release tag (v0.89.0) in a scratch dir OUTSIDE the repo; claims cited as [V-SRC <upstream-path>#<anchor>] against that tag; tracked-file gate (git ls-files --error-unmatch exit=1) proves nothing upstream leaked"
    - "Research-lead verification pattern: the lead file is the checklist, the code is the truth — every row carries a confirmed/drift/beyond-lead verdict against REFERENCE-ADAPTERS.md"

key-files:
  created:
    - docs/research/reference-adapters.md
  modified: []

key-decisions:
  - "Studied the live repo at tag v0.89.0 (release published 2026-11-19; default branch main; npm zcode-acp-server 0.89.0) instead of the lead's ~0.88.x — all upstream citations are tag-pinned with symbolic anchors"
  - "Drift recorded, not silently resolved: v0.89.0 never creates the coding-plan key (project-token.ts read-only posture) — Phase 6's Go port must fail with a fixed error + user cue, never POST api_keys"
  - "Weaker-adapter owner slugs live-confirmed (the lead's 'jpalmev/jpalmae' doubled slug resolved to jpalmae/zcode-acp; jpalmev 404s) and their Why-not gaps grep-verified in fresh clones — only the genuinely-unrechecked items (star counts, macOS-centricity) stay V-RPT"
  - "Upstream ships NO NOTICE file — attribution obligation pinned to Apache-2.0 §4(b)/(d): license header in every ported file + README/notice attribution + upstream-tag record in the porting PR"

patterns-established:
  - "Reference-study provenance: upstream citations version-pinned (tag v0.89.0), anchored on function/constant names rather than line numbers so citations survive upstream churn"

requirements-completed: [DISC-05]

coverage:
  - id: D1
    description: "Task 1: live read-only study — five key files + three test files read from a fresh v0.89.0 clone; adopt-list claims verified or drift recorded; license live-confirmed; nothing cloned committed"
    requirement: DISC-05
    verification:
      - kind: other
        ref: "gh api repos/william0wang/zcode-acp --jq '.license.spdx_id' -> Apache-2.0 (exit=0)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh -> exit 0"
        status: pass
      - kind: other
        ref: "bash experiments/scan-secrets.sh -> exit=0 (no SECRET SHAPE)"
        status: pass
      - kind: other
        ref: "git ls-files --error-unmatch -- '*credential-store*' -> exit=1 (no tracked upstream file; 0=leak and 128/129=git-broken distinguishable)"
        status: pass
      - kind: other
        ref: "Upstream state recorded: default branch main + latest release tag v0.89.0 (published 2026-11-19)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Task 2: docs/research/reference-adapters.md — seven house headings, 12-row verified coverage map with per-row lead comparison, Adopt/Avoid verdicts, Attribution section, weaker-adapters table, Start-Plan limitation, Sources & provenance"
    requirement: DISC-05
    verification:
      - kind: other
        ref: "grep -c '^## ' docs/research/reference-adapters.md -> 7 (EXPECT >= 7)"
        status: pass
      - kind: other
        ref: "grep -c 'V-SRC' docs/research/reference-adapters.md -> 17 (EXPECT >= 3)"
        status: pass
      - kind: other
        ref: "grep -c 'V-RPT' docs/research/reference-adapters.md -> 4 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "grep -c 'credential-store' docs/research/reference-adapters.md -> 3 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "grep -c 'Apache' docs/research/reference-adapters.md -> 10 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh -> exit 0, no CHECK FAIL (2 bare-tag violations found by the gate and fixed in-task)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh --phase-gate 2>&1 | grep -c 'PENDING: docs/research/reference-adapters.md' -> 0 (order-independent)"
        status: pass
    human_judgment: true
    rationale: "The plan's own disposition (T-1-10 mitigation + flagged probe assumption): the gates prove structure, tag grammar, path hygiene, and secret shapes, but the fidelity of each coverage row against the cloned code (that the cited anchor really says what the row claims) is judgment-tier — a human skim of the doc against the pinned tag is the designed check at /gsd:verify-work; the clone lives in scratch outside the repo for exactly that re-check"

# Metrics
duration: 13min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 06: Reference Adapters (DISC-05) Summary

**docs/research/reference-adapters.md written from a live study of william0wang/zcode-acp at tag v0.89.0 — 12-row coverage map (v2-keyring chain, dual-path requestCredentialCipher, exact snapshot revision formula, PAT chain, runtime-headers responder, elicitation gating, slash/resume/fork/attachments/lifecycle) with one drift correction (the reference never mints keys headless), five golden source files + three golden test files pinned for the Phase 6 port, Apache-2.0 attribution obligations named, weaker adapters live-verified, and the Start-Plan headless limitation recorded as the DOCS-03 note**

## Performance

- **Duration:** 13 min
- **Started:** 2026-11-19T07:39:29Z
- **Completed:** 2026-11-19T07:52:45Z
- **Tasks:** 2
- **Files modified:** 1 (created)

## Accomplishments

- Task 1 (read-only live study, no commit by design): shallow-cloned the reference at release tag **v0.89.0** into scratch outside the repo; read the five key files (credential-store.ts, account-provider.ts, project-token.ts, server-requests.ts, client.ts) and the three named test files; verified every adopt-list claim with provenance anchors; live-confirmed license (`gh api` → Apache-2.0, exit=0), default branch `main`, latest release `v0.89.0`; verified nothing cloned is committed (`git ls-files --error-unmatch` gate exit=1)
- Coverage map written with 12 rows, each with a confirmed/drift/beyond-lead verdict against the research lead: v2-keyring chain (service/account/pinning formula + fp24 checksum + v1 fallback ladder), requestCredentialCipher served in BOTH paths with requestId correlation, snapshot revision `zcode-builtin:<rev>:<sha256(resolved-path)>` (byte-identical to this repo's sharpened formula), PAT chain end-to-end with `{headersApplied, requestAuth:{apiKey, apiKeyId, accountScope}}` answer shapes, arrival-time runtime-headers responder (compact's internal turns), elicitation-with-permission-fallback (capability- and client-kind-gated), session/requestRuntimePreferences defaults, slash commands, resume/fork/load, attachments, session lifecycle, interaction reannounce-dedup/reconnect-resend
- **Drift recorded:** v0.89.0 NEVER creates the api key — "api_keys (create if absent)" from the lead is wrong for the studied tag; the bridge stays read-only towards the account (missing key = fixed `project_token_key_missing` + open-the-desktop-once cue), confirmed by both source and test (`tests/project-token.test.ts` "never creates the zcode-api-key")
- Adopt section names the five golden source files (one line each on what they prove) and the three golden test files as the Phase 6 acceptance oracle; Avoid section rejects the keyring module, ACP routing layer, client-kind sniffing, remote/broadcast surface, legacy config ladders, and the sync-cache shape
- Attribution section names the exact Apache-2.0 obligations (no upstream NOTICE: license header per ported file + README attribution + upstream-tag record), consistent with this repo's own LICENSE
- Weaker adapters (jpalmae/zcode-acp Rust, Liang-HZ/zcode-acp Python) live-confirmed via `gh api` and their Why-not gaps grep-verified in fresh clones — differentiator claim (nobody else does v2-keyring + coding-plan chain) upgraded from reported to verified; only genuinely-unrechecked items stay V-RPT
- Start-Plan headless limitation (OAuth JWT + Aliyun captcha session, desktop-only) recorded from upstream TROUBLESHOOTING as the DOCS-03 note, with the stored-JWT nuance found live

## Task Commits

1. **Task 1: Study the reference repo live (clone to scratch, verify adopt-list claims)** — no commit (read-only study by design: the plan's `<files>` element is deliberately empty; the only acceptance artifacts are gate exits recorded above)
2. **Task 2: Write docs/research/reference-adapters.md (DISC-05)** - `f53284d` (feat)

## Files Created/Modified

- `docs/research/reference-adapters.md` - the DISC-05 deliverable: coverage map, adopt/avoid verdicts, attribution obligations, weaker-adapters table, Start-Plan limitation, provenance

## Decisions Made

- Studied the pinned release tag v0.89.0 (not `main`) so every citation is reproducible; anchors are function/constant names, not line numbers, so they survive upstream churn
- Drift handled row-by-row rather than silently: the lead stays the checklist, the cloned code is the truth, and the one substantive drift (no key creation) is flagged in the coverage table AND as a Phase-6-affects provision
- Weak-adapter slugs were re-confirmed live where cheap (two `gh api` calls + two shallow clones + greps) — upgrading them from V-RPT to live-verified makes the differentiator claim defensible at verify-work instead of hearsay
- Task 1 committed nothing (plan-sanctioned): the study's evidence lives in the doc's V-SRC citations; the tracked-file gate distinguishes clean (1) / leak (0) / broken-git (128-129)

## Deviations from Plan

None - plan executed exactly as written. (The bare-tag CHECK FAILs during Task 2's first gate run were caught and fixed in-task by the gate itself — normal gate-driven iteration, not a deviation.)

## Issues Encountered

- `npm view zcode-acp-server version` hung on this machine's network — the version was confirmed from the clone's package.json (0.89.0) instead; the plan only requires the release tag, which `gh api` supplied
- `gh api` latency exceeded 120 s on the first batched call (moved to background); all required outputs (license, default branch, release tag) were captured before proceeding
- TDD gate: both tasks are documentation/study-type (`is_behavior_adding` = false expected, matching sibling plans 01-04/01-05) — no RED/GREEN cycle applies

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ROADMAP Phase 1 success criterion 5 satisfied: docs/research/reference-adapters.md records what the TypeScript reference covers, what to adopt, and what to avoid — verified against the live repo
- Phase 6 can begin porting with the golden-reference files, the three golden test files as the acceptance oracle, and the attribution obligation settled (license header + README attribution + tag record)
- Phase 6 must respect the recorded drift: port the read-only key posture (never mint keys; missing key = fixed error + user cue), and answer runtime headers with the `{headersApplied, requestAuth:{apiKey, apiKeyId, accountScope}}` shape including the OAuth recovery field
- Phase 7 (DOCS-03) gets the Start-Plan headless limitation with its exact upstream citation and the stored-JWT nuance
- The phase-level `bash experiments/check-docs.sh --phase-gate` all-six pass remains the `/gsd:verify-work` completion check (this plan's own doc passes its per-doc table; the remaining PENDING is app-server-dialect.md from plan 01-02)
- Untracked planning artifacts present in the worktree (`.planning/WINDOWS.md`, `.planning/intel/`, `.planning/state.json`, modified `.planning/config.json`) belong to the orchestrator/parallel executors and were left untouched

---
*Phase: 01-research-protocol-documentation*
*Completed: 2026-11-19*

## Self-Check: PASSED

- All plan artifacts verified present on disk: docs/research/reference-adapters.md, this SUMMARY
- Commit verified as ancestor of HEAD: f53284d (Task 2, the only production commit — Task 1 is read-only by design); measured from ledger 67ce60b = 1 production commit (matches actuals.commits; 2 total including the metadata commit)
- Gates re-verified green post-commit: check-docs.sh exit 0 (no CHECK FAIL), scan-secrets.sh exit 0 (no SECRET SHAPE), tracked-file gate exit=1 (nothing upstream committed)
- Phase-gate PENDING count for docs/research/reference-adapters.md = 0 (order-independent; remaining PENDING is app-server-dialect.md from plan 01-02, another wave-2 plan)
- Known stubs: none (documentation-only plan); no unrun verifies, no skipped tests
- TDD gate: documentation/study-type tasks, no tdd="true" / behavior blocks — no RED/GREEN cycle applicable
- Execution-mode note: commits landed on master per the orchestrator's sequential-execution dispatch on the main working tree (project config branching_strategy "none"; the established pattern of plans 01-01/01-02/01-04/01-05)
