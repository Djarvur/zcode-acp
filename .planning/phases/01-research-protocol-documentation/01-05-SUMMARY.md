---
phase: 01-research-protocol-documentation
plan: "05"
subsystem: docs
tags: [install-discovery, appimage, deb, rpm, auto-update, builtin-revision, provenance, discovery-matrix]

# Dependency graph
requires:
  - phase: 01-research-protocol-documentation (plan 01-01)
    provides: scrub-first capture pipeline, validation gates (check-docs.sh / scan-secrets.sh), provenance legend in docs/protocol/README.md
provides:
  - "docs/protocol/install-discovery.md — DISC-03 deliverable: layout matrix (unpacked AppImage / AppImage file + symlink / deb / rpm / running-mount), per-layout entrypoint location, desktop-entry + URL-handler clash, auto-update + builtin-revision section, recommended discovery order"
  - "experiments/captures/EXP-DISCOVERY-01.jsonl — E4 scrubbed capture: all seven rows (pristine re-check, AppImage row, dpkg remnant + checksum-verified 3.18.2-10443 artifact inspection, rpm header peek, desktop entries + live xdg state, runtime tree + revision formula, auto-update facts, localhost decoys)"
  - "Revision formula sharpened V-SRC: zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))> — hash over the resolved PATH STRING, not file contents"
  - "Deb layout facts: /opt/ZCode install root, <ver>-<build> version scheme, postinst /usr/bin/zcode + apparmor profile, entrypoint path invariance across layouts"
affects: [phase 3 supervisor discovery + BRIDGE-03 diagnostics, phase 6 snapshot/revision verification, phase 7 conformance scenario]

# Actuals (#2632)
actuals:
  tokens: 8385
  tasks: 2
  commits: 2

commits: 2
plan_head_before: 3f6323f388e771549dbcb006c669138cb61177fa
plan_head_after: c766ede996f16c0727145896b107ef8d46a822d4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Layout matrix as locate-command ledger: every row is a pure-read command + provenance tag + status (stable / deb-generalized / do-not-rely)"
    - "Artifact inspection without installation: checksum-verified release artifact inspected with dpkg-deb -I/-c in scratch; CDN URLs fetched live at execution time, never hardcoded"

key-files:
  created:
    - docs/protocol/install-discovery.md
    - experiments/captures/EXP-DISCOVERY-01.jsonl
  modified: []

key-decisions:
  - "Deb artifact provenance: inspected the checksum-verified release .deb from the desktop updater's own pending cache (sha512 verified against its update-info.json) instead of a fresh 210 MB download; official CDN URLs still fetched live (HEAD 200) and recorded by pattern"
  - "Revision formula corrected on fresh bundle evidence: the third component is sha256 over path.resolve(activeFilePath) — the resolved PATH STRING, not file contents; concrete example computed (zcode-builtin:44:e9c0674a…), content-sha256 recorded separately for drift detection"
  - "rpm row recorded deb-generalized + header-peek-verified (zcode-3.18.2-10443.x86_64 from a 128 KiB range read; no rpm tooling on this machine) with the exact re-run command for an rpm-tooling host"
  - "Live auto-update stays V-RPT: observed = updater downloads deb artifacts into ~/.cache/@zcodedesktop-updater/pending/ + tree accretes per-version dirs (28→30→31→43→44); never observed = the install/apply step (pending deb not installed on this machine)"

patterns-established:
  - "Probe-row resolutions documented in-doc: empty (nothing-found is a first-class matrix outcome with BRIDGE-03 actionable-diagnostics pointer), encoding (per-layout version-string formats; equality compares on <ver> only), idempotency (all rows pure reads; only runtime tree/revision/pending cache can change under the update cycle), concurrency (record revision at probe time; never record volatile mounts as spawn root)"

requirements-completed: [DISC-03]

coverage:
  - id: D1
    description: "Scrubbed E4 capture experiments/captures/EXP-DISCOVERY-01.jsonl with header + all seven E4 rows (or explicitly-marked fallbacks), patterns only"
    requirement: DISC-03
    verification:
      - kind: other
        ref: "grep -c '\"capture\":\"EXP-DISCOVERY-01\"' experiments/captures/EXP-DISCOVERY-01.jsonl -> 1 (EXPECT == 1)"
        status: pass
      - kind: other
        ref: "grep -ciE 'pristine|identical' experiments/captures/EXP-DISCOVERY-01.jsonl -> 2 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "grep -c 'dpkg-deb' experiments/captures/EXP-DISCOVERY-01.jsonl -> 4 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh -> exit 0 (path-shape check over captures green)"
        status: pass
      - kind: other
        ref: "bash experiments/scan-secrets.sh -> exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/protocol/install-discovery.md — the DISC-03 deliverable: layout matrix, per-layout entrypoint location, desktop entry/URL handler clash, auto-update + revision section, recommended discovery order, sources & provenance"
    requirement: DISC-03
    verification:
      - kind: other
        ref: "bash experiments/check-docs.sh -> exit 0, no CHECK FAIL"
        status: pass
      - kind: other
        ref: "grep -c 'V-EXP EXP-DISCOVERY-01' docs/protocol/install-discovery.md -> 16 (EXPECT >= 3)"
        status: pass
      - kind: other
        ref: "grep -c '^## ' docs/protocol/install-discovery.md -> 6 (EXPECT >= 6)"
        status: pass
      - kind: other
        ref: "grep -ciE 'do not rely|do-not-rely' docs/protocol/install-discovery.md -> 3 (EXPECT >= 1)"
        status: pass
      - kind: other
        ref: "bash experiments/check-docs.sh --phase-gate 2>&1 | grep -c 'PENDING: docs/protocol/install-discovery.md' -> 0 (order-independent)"
        status: pass
    human_judgment: true
    rationale: "The plan's own disposition (T-1-06/T-1-11 mitigations): the gates prove structure, tag grammar, id resolution, path hygiene, and secret shapes, but the honesty of matrix rows (locate commands actually run as recorded, artifact never installed) is judgment-tier — a human skim of the capture against the doc is the designed check at /gsd:verify-work"

# Metrics
duration: 20min
completed: 2026-11-19
status: complete
---

# Phase 1 Plan 05: Install Discovery Matrix (DISC-03) Summary

**docs/protocol/install-discovery.md written from a live read-only enumeration: five-layout matrix (unpacked AppImage / AppImage file + symlink / deb artifact-inspected 3.18.2-10443 / rpm header-peeked / do-not-rely running mount), the deb-vs-AppImage URL-handler clash confirmed from both desktop entries, and the builtin-revision formula V-SRC-sharpened to sha256-of-resolved-path — every row backed by recorded commands in EXP-DISCOVERY-01**

## Performance

- **Duration:** 20 min
- **Started:** 2026-11-19T07:11:50Z
- **Completed:** 2026-11-19T07:51:50Z
- **Tasks:** 2
- **Files modified:** 2 (both created)

## Accomplishments

- All seven E4 rows recorded read-only into `experiments/captures/EXP-DISCOVERY-01.jsonl` (37 lines): unpacked-AppImage pristine re-check (`cmp` byte-identical, feeds the dialect doc's provenance header per T-1-06), AppImage file + symlink, dpkg rc remnant + current-artifact deb inspection, rpm header peek, desktop entries + live xdg handler state, runtime provider tree + revision formula, auto-update facts
- Deb row is artifact-grounded: the current `3.18.2-10443` release .deb was inspected with `dpkg-deb -I`/`-c` in scratch (3,951 entries; `/opt/ZCode/` root; `resources/glm/zcode.cjs` path invariance; postinst `/usr/bin/zcode` update-alternatives + apparmor profile) — NEVER installed, per T-1-11
- Layout matrix written with all six house sections; running-mount row marked do-not-rely; empty-result case first-class with the BRIDGE-03 actionable-diagnostics requirement stated
- Revision formula V-SRC-verified from the shipped bundle and SHARPENED: `zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>` — hash over the resolved path STRING, not file contents; concrete computed example recorded
- Localhost-listener decoy (Pitfall 10) recorded live: dynamic-port `zcode-host-local-*` (404) + unrelated http-proxy on 8081 (400) — neither is protocol; port-scanning is useless
- Recommended discovery order documented (env/flag → desktop entry → AppImage paths → package paths; never volatile mounts/decoys) with idempotency + concurrency guarantees

## Task Commits

Each task was committed atomically:

1. **Task 1: E4 — install layout enumeration (EXP-DISCOVERY-01)** - `4bcdf75` (feat)
2. **Task 2: Write docs/protocol/install-discovery.md (DISC-03)** - `c766ede` (feat)

## Files Created/Modified

- `experiments/captures/EXP-DISCOVERY-01.jsonl` - scrubbed E4 capture: findings rows for all seven steps, patterns only (no user-specific absolute paths), header records builtinRevision 44 at capture time
- `docs/protocol/install-discovery.md` - the DISC-03 claim-ledger doc (matrix, entrypoint location, URL handler, auto-update/revision, discovery order, provenance)

## Decisions Made

- Deb artifact obtained from the desktop updater's own pending cache (sha512-verified against its `update-info.json`) rather than a fresh 210 MB download — same current artifact, strictly stronger provenance; official CDN URLs were additionally fetched live at execution time (HEAD 200, linux-x64 + linux-arm64) and recorded by pattern, satisfying the never-hardcode rule
- Revision formula corrected on bundle evidence: the minified release store hashes `path.resolve(activeFilePath)` — the plan's "sha256 of the resolved builtin path" shorthand now reads precisely as a path-STRING hash; the file-content sha256 was recorded separately as a drift-detection quantity
- rpm row marked deb-generalized (plan-sanctioned option): version scheme header-verified from the artifact, full `rpm -qlp` layout deferred with an exact re-run command, mirror-the-deb recorded as an assumption
- Live-update behavior honestly V-RPT: the observed pending deb was never installed on this machine (desktop still runs the AppImage layout), so the install/apply step remains unobserved

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking substitution, provenance-improving] Deb artifact sourced from the updater's checksum-verified pending cache instead of a fresh download**
- **Found during:** Task 1 (E4 step 3)
- **Issue:** The action specified downloading the current .deb from the official install page; the desktop's own updater had already downloaded the exact current artifact (`~/.cache/@zcodedesktop-updater/pending/` + `update-info.json` with sha512), making a 210 MB re-download redundant
- **Fix:** Copied the pending artifact to scratch, verified sha512 (base64 match against update-info.json), inspected with `dpkg-deb -I`/`-c`; additionally HEAD-verified the official CDN URLs live (fetched at execution time, recorded by pattern) so the never-hardcode + URL-liveness intent is fully preserved
- **Files modified:** experiments/captures/EXP-DISCOVERY-01.jsonl (provenance note in the deb row)
- **Verification:** `grep -c 'dpkg-deb'` = 4; scan-secrets green
- **Committed in:** 4bcdf75

**2. [Rule 1 - Plan shorthand contradicted by bundle source] Revision-formula component identified as a path-STRING hash**
- **Found during:** Task 1 (E4 step 6 formula verification)
- **Issue:** The plan (and prior notes) recorded the formula as `zcode-builtin:<rev>:<sha256 of the resolved builtin path>` — ambiguous between hashing the path string and the file contents
- **Fix:** Grep context over the shipped 3.18.2 bundle shows `createHash("sha256").update(path.resolve(activeFilePath))` — the input is the resolved path string; doc + capture state this explicitly with a computed example
- **Files modified:** docs/protocol/install-discovery.md, experiments/captures/EXP-DISCOVERY-01.jsonl
- **Verification:** bundle literals quoted in the capture; example recomputed live
- **Committed in:** 4bcdf75 / c766ede

---

**Total deviations:** 2 auto-fixed (1 × Rule 3 method substitution with stronger provenance, 1 × Rule 1 precision correction; no scope creep, no code)
**Impact on plan:** Both increase evidence quality; ROADMAP success criterion 3 satisfied as written.

## Issues Encountered
- No rpm/rpm2cpio tooling on the probe machine — the rpm row uses the plan's deb-generalized option plus a 128 KiB header range-read for the version scheme; re-run command recorded in the doc
- The desktop was running during the capture (live decoy + running-mount evidence captured as a bonus); its `/tmp/.mount_*` paths were recorded as patterns only
- TDD gate checked via `task.is-behavior-adding`: `is_behavior_adding: false` (no tdd="true" tasks, no behavior blocks — documentation plan), so no RED/GREEN cycle applies

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ROADMAP Phase 1 success criterion 3 satisfied: the discovery matrix documents how to locate a ZCode desktop install across AppImage / deb / unpacked layouts, including what auto-update does to the builtin revision string
- Phase 3 BRIDGE-03 can implement locate + diagnostics directly from this document: matrix rows carry the exact locate commands, the discovery order is explicit, and the nothing-found diagnostics requirement is stated with the full probe list
- Phase 6 consumers get the sharpened revision formula (path-string hash) with a computed example and the per-version revision table for drift checks
- Remaining known-unknowns (V-RPT, by design): the auto-update install/apply step; full rpm file list (rpm-tooling host); both carry exact re-run instructions in the doc
- Untracked planning artifacts present in the worktree (`.planning/WINDOWS.md`, `.planning/intel/`, `.planning/state.json`, modified `.planning/config.json`) belong to the orchestrator/parallel executors and were left untouched

---
*Phase: 01-research-protocol-documentation*
*Completed: 2026-11-19*

## Self-Check: PASSED

- All plan artifacts verified present on disk: docs/protocol/install-discovery.md, experiments/captures/EXP-DISCOVERY-01.jsonl, this SUMMARY
- All commits verified as ancestors of HEAD: 4bcdf75 (Task 1), c766ede (Task 2), 81619b9 (plan metadata); measured from ledger 3f6323f = 2 production commits at SUMMARY-write time (matches actuals.commits; 3 total including the metadata commit)
- Gates re-verified green post-commit: check-docs.sh exit 0 (no CHECK FAIL), scan-secrets.sh exit 0 (no SECRET SHAPE)
- Phase-gate PENDING count for docs/protocol/install-discovery.md = 0 (order-independent per plan revision; remaining PENDING lines are the other wave-2 plans' docs: app-server-dialect.md, reference-adapters.md)
- TDD gate: task.is-behavior-adding = false (documentation plan — no tdd tasks, no behavior blocks); no RED/GREEN cycle applicable
- ROADMAP progress updated via roadmap.update-plan-progress (4/6 summaries, In Progress — no missing_phase_details decline); STATE.md at Plan 5 of 6, Stopped at Completed 01-05-PLAN.md; DISC-03 marked complete in REQUIREMENTS.md (ready-ids resolved it as ready)
- Execution-mode note: commits landed on master per the orchestrator's sequential-execution dispatch on the main working tree (project config branching_strategy "none"; the established pattern of plans 01-01/01-02/01-04)
