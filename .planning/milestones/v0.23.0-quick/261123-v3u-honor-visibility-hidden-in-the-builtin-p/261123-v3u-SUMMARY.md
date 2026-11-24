---
phase: 261123-v3u
plan: 01
subsystem: auth
tags: [hidden-providers, model-catalog, builtin-table, account-overlay, visibility, tdd]

# Dependency graph
requires:
  - phase: "06"
    provides: BuildAccount account overlay (verbatim builtin-table mirror) + entitlement gate + the coded errNoRulesToEntitle class
  - phase: "05"
    provides: the want-list / read-back verdict machinery (wantModelsFromSnapshot, ProbeWantModel)
provides:
  - 'BuildAccount omits config.visibility "hidden" zhipu-account rules from providers AND states'
  - "wantModelsFromSnapshot can never carry a hidden provider/model (absence by construction, so ProbeWantModel cannot target one)"
  - "an all-hidden builtin table still answers the existing coded errNoRulesToEntitle error — never an empty overlay"
affects: [fix/model-catalog-hidden-providers, configoptions-catalog-mirror, model-catalog-duplicate-rows]

# Actuals (#2632) — plan carried no estimate; recorded for calibration.
actuals:
  tokens: 4484
  tasks: 2
  commits: 2

# Measured commit ledger (#3968) — not narrated.
commits: 2
plan_head_before: 147c8a109d518da63f8c67b7014ac295129589f6
plan_head_after: 2513949687cb47e2e298f0d7e753b6f050714bd0

tech-stack:
  added: []
  patterns:
    - "config.visibility is captured verbatim and compared exactly to the literal \"hidden\" (absent/null/\"visible\"/other = visible), mirroring the desktop picker's strict !== \"hidden\""
    - "A hidden rule is data, not a malformed-table class — no new coded error; the existing zero-provider errNoRulesToEntitle arm covers the all-hidden table"
    - "Omission happens at snapshot-build time; no catalog-side filtering or entry renaming (D-08 intact)"

key-files:
  created:
    - internal/acpfront/wantmodels_internal_test.go
  modified:
    - internal/hostsvc/snapshot.go
    - internal/hostsvc/snapshot_account_test.go
    - internal/acpfront/auth.go

key-decisions:
  - 'Exact-string comparison: compare config.visibility to "hidden" with no trimming and no case folding; absent, null, "visible" and every other value materialize — the desktop's strict !== "hidden" discipline'
  - "Skip placement: after the zhipu-account filter and before the entitlement gate, so the filter is family-agnostic and the gate keeps its omit-only contract"
  - "No new error class: the visibility filter reuses the existing errNoRulesToEntitle arm, so an all-hidden table cannot silently push an empty overlay (T-261123-01)"
  - "No catalog filtering: the omission is at snapshot-build time, keeping the D-08 verbatim mirror and the 817f83e/91e86f3 catalog-shrink fix untouched"

requirements-completed: []

coverage:
  - id: D1
    description: "A builtin-table rule carrying config.visibility \"hidden\" is absent from BuildAccount's providers and states; visible rules (absent/\"visible\") remain mirrored verbatim"
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotVerbatimMirror"
        status: pass
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotGateOmission"
        status: pass
    human_judgment: false
  - id: D2
    description: "A table whose zhipu-account rules are all hidden answers the existing coded errNoRulesToEntitle error with a nil Snapshot (no empty overlay)"
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotAllHidden"
        status: pass
    human_judgment: false
  - id: D3
    description: "wantModelsFromSnapshot over a visible+hidden snapshot yields exactly the visible provider/model pairs and ProbeWantModel cannot return the hidden entry"
    verification:
      - kind: unit
        ref: "internal/acpfront/wantmodels_internal_test.go#TestWantModelsFromSnapshotOmitsHidden"
        status: pass
    human_judgment: false
  - id: D4
    description: "Existing verbatim-mirror, entitlement-gate, malformed-table, recorded-file, and corpus/golden locks stay green under -race"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/hostsvc/ ./internal/acpfront/"
        status: pass
    human_judgment: false

# Metrics
duration: 13min
completed: 2026-11-23
status: complete
---

# Quick Task 261123-v3u: Honor `config.visibility: "hidden"` in the builtin provider overlay — Summary

**`BuildAccount` now omits builtin-table rules carrying `config.visibility: "hidden"` from both `providers` and `states`, so the desktop's hidden idle/highspeed account providers never enter the registry and therefore never reach the verbatim-mirrored `settings.model.available` catalog**

## Performance

- **Duration:** 13 min (2026-11-23T19:35:35Z → 2026-11-23T19:48:06Z)
- **Tasks:** 2/2
- **Commits:** 2 (TDD RED → GREEN)
- **Files modified:** 4 (2 test, 2 production)

## Accomplishments

- `BuildAccount` captures `config.visibility` verbatim and skips a rule whose value is exactly `"hidden"`, placing the check after the `zhipu-account` access filter and before the entitlement gate — hidden rules are omitted from **both** `providers` and `states`, while absent/`"visible"`/other values keep the byte-identical pre-change mirror.
- An all-hidden table still answers the existing coded `errNoRulesToEntitle` error with a nil Snapshot — the visibility filter opens no empty-overlay path (T-261123-01).
- `wantModelsFromSnapshot` therefore never carries a hidden provider/model: the read-back watcher and `ProbeWantModel` cannot target one, and the doc comment now states the absence-by-construction and its D-08 reasoning.
- RED/GREEN locks at two tiers: the snapshot overlay (hostsvc) and the derived want list (acpfront in-package).

## Task Commits

Each task was committed atomically (TDD sequence):

1. **Task 1 (RED): Lock hidden-visibility omission in the account overlay and the want list** — `83c3ebf` (test)
2. **Task 2 (GREEN): Capture config.visibility and skip hidden zhipu-account rules** — `2513949` (feat)

No REFACTOR commit — no cleanup was needed once GREEN passed.

## RED Evidence (observed before any production change)

Gate (plan Task 1 `<automated>`, run verbatim):

```
out=$(go test -race -count=1 ./internal/hostsvc/ -run 'TestAccountSnapshotVerbatimMirror|TestAccountSnapshotAllHidden')
out2=$(go test -race -count=1 ./internal/acpfront/ -run 'TestWantModelsFromSnapshotOmitsHidden')
→ RED confirmed at both tiers   (exit 0 from the gate)
```

hostsvc arm:

```
--- FAIL: TestAccountSnapshotVerbatimMirror (0.00s)
    snapshot_account_test.go:248: providers carries "account:zai-offpeak-idle-plan", want the rule with config.visibility "hidden" omitted (desktop-picker parity: the desktop's own model view filters the same field, and entitling a hidden account provider materializes a duplicate account row)
--- FAIL: TestAccountSnapshotAllHidden (0.00s)
    snapshot_account_test.go:587: BuildAccount answered success (providers map[account:zai-offpeak-idle-plan:map[access:map[entitled:true type:zhipu-account] builtinModelIds:[GLM-5.3]]]) for a table whose every zhipu-account rule carries config.visibility "hidden" — want the coded error, never an empty overlay
FAIL
FAIL	github.com/Djarvur/zcode-acp/internal/hostsvc	0.020s
```

acpfront arm:

```
--- FAIL: TestWantModelsFromSnapshotOmitsHidden (0.00s)
    wantmodels_internal_test.go:73: derived want list: [account:hidden/M-Hidden account:visible/M-A account:visible/M-B]
    wantmodels_internal_test.go:76: providers carries "account:hidden", want the config.visibility "hidden" rule omitted (desktop-picker parity: the desktop's own model view filters the same field)
FAIL
FAIL	github.com/Djarvur/zcode-acp/internal/acpfront	0.009s
```

**Semantic assessment:** both named target tests executed and failed on the planned assertion — the fixture's hidden provider leaked into `providers` (and therefore into the derived want list, observed directly as `[account:hidden/M-Hidden …]`), and the all-hidden table was wrongly accepted. No build/setup/zero-test failure. Full-suite RED also showed `TestAccountSnapshotGateOmission` failing for the same leak (the hidden `zai`-family idle rule survived the gate); it passes unchanged under GREEN, confirming the failure was the planned visibility behavior and not a fixture accident.

## GREEN Evidence

Task 2 focused gate (plan text, run verbatim up to lint):

```
ok  	github.com/Djarvur/zcode-acp/internal/hostsvc	1.023s   (go test -run 'TestAccountSnapshot')
ok  	github.com/Djarvur/zcode-acp/internal/acpfront	1.024s   (go test -run 'TestWantModelsFromSnapshot')
```

Full focused suites and final verification:

```
go build ./... && go vet ./...   → exit 0
ok  	github.com/Djarvur/zcode-acp/internal/hostsvc	1.392s
ok  	github.com/Djarvur/zcode-acp/internal/acpfront	102.492s
```

## Files Created/Modified

- `internal/hostsvc/snapshot.go` — `builtinVisibilityHidden` constant; `builtinProviderRule.Visibility`; `digBuiltinRule` decodes `config.visibility`; `BuildAccount` skips hidden rules before the entitlement gate; doc comments updated (`BuildAccount`, `builtinProviderRule`, `decodeBuiltinTable`).
- `internal/hostsvc/snapshot_account_test.go` — fixture gains the hidden idle-plan row and an explicit `"visible"` arm; explicit absence assertions for `providers`/`states`; new `TestAccountSnapshotAllHidden`.
- `internal/acpfront/wantmodels_internal_test.go` (new) — in-package `TestWantModelsFromSnapshotOmitsHidden` over a one-visible/one-hidden revision-44 table; exact sorted want-list comparison plus a `ProbeWantModel` target check.
- `internal/acpfront/auth.go` — `wantModelsFromSnapshot` doc comment states the hidden absence-by-construction and the D-08 verdict-side reasoning.

## Decisions Made

- Compare `config.visibility` exactly to `"hidden"` (no trim, no case folding); absent, `null`, `"visible"` or any other value is visible — the desktop picker's strict `!== "hidden"` contract.
- Skip after the `zhipu-account` filter, before the entitlement gate; reuse the existing `errNoRulesToEntitle` arm for the all-hidden case (no new coded error, T-261123-01).
- No wrapper-side catalog filtering or entry renaming (D-08 intact; the 817f83e/91e86f3 catalog-shrink fix untouched).

## Deviations from Plan

None in implementation — all four declared files and no others (`git diff --stat 147c8a1..HEAD` shows exactly those four; no deletions).

One gate outcome must be recorded honestly: **`mise run lint` exits 1 on a pre-existing lint-debt set that exists at the pre-change HEAD**. The full final command was run exactly; build, vet and both suites pass, and only the lint arm fails — for files this task never touched. Details under "Issues Encountered".

## Issues Encountered

1. **Pre-existing lint debt blocks the `mise run lint` arm (out of scope, not fixed).**
   Verified on the clean tree at HEAD `147c8a1` *before any edit*: 10 issues in 5 unrelated test files —
   `funlen` in `internal/acpfront/setconfigoption_test.go`; `goconst` ×6 in `catalogmerge_test.go`, `codingplan_probe_test.go` (×2), `silentauth_test.go`, `switchcatalog_test.go` (×2); `gofmt` + `lll` in `silentauth_test.go`; `prealloc` in `switchcatalog_test.go`.
   After the change the issue set is **byte-identical** (same 10 lines, same files; `diff` of the sorted issue lists is empty) — zero new findings in the four declared files. Per the scope-boundary rule this debt was not fixed; it should be closed separately (the STATE.md deferred-items table already tracks earlier lint-debt acknowledgements).

2. **First RED attempt did not compile** because an edit glued a comment line to the `accountTableUndecodable` const declaration (`undefined: accountTableUndecodable`). No RED evidence was accepted from that run; the syntax was fixed and the gate re-run to produce the genuine RED quoted above. No production file had been touched at that point.

3. **`gsd_run` is unavailable in this environment** (no SDK on PATH), so `gsd_run check tdd-red-evidence` and the state/roadmap/requirements update verbs could not be invoked. The plan's own RED/GREEN `<automated>` gates were run verbatim instead; TDD sequencing was checked with git (`83c3ebf` test commit is an ancestor of `2513949` feat commit). Docs/state/roadmap updates are the orchestrator's responsibility per this task's instructions.

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED | `83c3ebf` `test(261123-v3u): lock hidden-visibility omission in the account overlay` | Pass — both target tests failed on the planned assertions (quoted above) |
| GREEN | `2513949` `feat(261123-v3u): skip hidden builtin providers in the account overlay` | Pass — RED descendants green; focused and full suites pass under `-race` |
| REFACTOR | — | Not needed |

## Known Stubs

None. No placeholder values, hardcoded empty data paths, or unwired components were introduced.

## Threat Flags

None. The change removes trust-boundary surface (fewer account providers entitled from the table); no new endpoint, auth path, file-access pattern, or schema change was added. T-261123-01 (all-hidden/forged table) is closed by `TestAccountSnapshotAllHidden`; T-261123-02 (over-broad filter) is bounded by the exact-string comparison plus the visible-arm locks; T-261123-SC is unaffected (no dependency changes).

## User Setup Required

None — no external service or configuration change.

## Next Phase Readiness

- The hidden idle/highspeed providers are omitted at snapshot-build time, so the catalog mirror should stop showing the duplicate account rows without any client-side filtering.
- The remaining pre-existing lint debt (10 findings, 5 files) is unrelated to this task but will keep `mise run lint` red until addressed.

---
*Phase: 261123-v3u*
*Completed: 2026-11-23*

## Self-Check: PASSED

- All four declared files exist; the diff from `147c8a1` touches exactly those four paths.
- Commits `83c3ebf` (RED) and `2513949` (GREEN) both exist and are ancestors of HEAD; the measured ledger count is 2.
- No untracked build artifacts (only the orchestrator-owned `.planning/quick/` docs directory).
