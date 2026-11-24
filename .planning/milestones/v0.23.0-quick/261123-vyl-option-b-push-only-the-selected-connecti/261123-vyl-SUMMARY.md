---
phase: 261123-vyl
plan: 01
subsystem: hostsvc / interact / acpfront auth
tags: [account-overlay, entitlement-gate, per-mode-selection, builtin-table, want-list, tdd]

requires:
  - quick task 261123-v3u
    provides: hidden-visibility skip in BuildAccount and the want list (option A), kept intact and test-locked
provides:
  - config.access.mode captured verbatim on builtinProviderRule
  - per-(family, mode) entitlement seam through BuildAccount and AuthConfig.Entitle
  - kind == mode production gate (individual/team/start selected kinds; off-peak/highspeed never match)
  - per-mode snapshot and want-list test locks
affects: [hostsvc snapshot overlay, acpfront coding-plan authenticate, read-back want list]

actuals:
  tokens: 11650
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Gate seam keyed by (family, mode): BuildAccount receives func(family, mode string) bool and may only omit rules"
    - "RED evidence is compile-level when the seam widens: production stays buildable while the four test packages fail naming the new contract"

key-files:
  created: []
  modified:
    - internal/hostsvc/snapshot.go
    - internal/interact/familysel.go
    - internal/acpfront/auth.go
    - internal/interact/familysel_test.go
    - internal/hostsvc/snapshot_account_test.go
    - internal/hostsvc/snapshot_builtinfile_test.go
    - internal/acpfront/wantmodels_internal_test.go
    - internal/acpfront/auth_test.go
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "The gate compares the desktop's selected connection kind to the rule's access.mode exactly; off-peak/highspeed occur only as rule modes and can never match."
  - "A rule with an absent/empty access.mode never matches a selected kind (empty selection kinds are also rejected by selectionKind), so malformed/legacy rules are omitted, never fabricated as entitled."
  - "Fixture rules codingPlanFixtureInstall and codingPlanCorpusInstall deliberately carry no access.mode; their family-only test closures ignore the mode argument, keeping golden bytes byte-identical."

patterns-established:
  - "Per-mode omission is locked at three tiers: gate unit arms, snapshot key-set arms, and the want-list mirror."
  - "Test-local constants for repeated provider ids/members/kind keys keep goconst clean as arms multiply."

requirements-completed: []

coverage:
  - id: D1
    description: "config.access.mode is captured verbatim on the builtin rule and the overlay decision is keyed by (family, mode)"
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotModeSelection"
        status: pass
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotGateOmission"
        status: pass
    human_judgment: false
  - id: D2
    description: "EntitlementGate.Entitled answers true only when the desktop's selected kind equals the rule mode"
    verification:
      - kind: unit
        ref: "internal/interact/familysel_test.go#TestFamilySelections"
        status: pass
    human_judgment: false
  - id: D3
    description: "The read-back want list mirrors the per-mode overlay; non-selected-mode and hidden rules never enter it or the ProbeWantModel target"
    verification:
      - kind: unit
        ref: "internal/acpfront/wantmodels_internal_test.go#TestWantModelsFromSnapshotOmitsNonSelectedMode"
        status: pass
      - kind: unit
        ref: "internal/acpfront/wantmodels_internal_test.go#TestWantModelsFromSnapshotOmitsHidden"
        status: pass
    human_judgment: false
  - id: D4
    description: "The option-A hidden-visibility skip and all degradation arms (setting missing/unreadable/undecodable, apiKey veto, missing token) are unchanged"
    verification:
      - kind: unit
        ref: "go test -race -count=1 ./internal/interact/ ./internal/hostsvc/ ./internal/acpfront/ ./internal/zcode/fake/"
        status: pass
    human_judgment: false
  - id: D5
    description: "No file outside the nine declared paths changed; the method value machinery.gate.Entitled adapts without edits (build + realdesktop-tagged vet)"
    verification:
      - kind: other
        ref: "git diff dc1308a..HEAD --stat + go vet -tags realdesktop ./test/realdesktop/"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-11-23
status: complete
plan_head_before: dc1308a3dec00edd77bcec7986b8a5df6de99589
plan_head_after: b65901f438322198ee77ea657db6a22673190e9e
---

# Quick task 261123-vyl: push only the selected connection mode per account family

**The account overlay now carries exactly the zhipu-account rule whose access.mode equals the desktop's selected connection kind per family — with the live selection that is account:zai-individual-coding-plan and account:bigmodel-start-plan, and every other visible mode (zai team/start, bigmodel individual/team) is absent from providers AND states.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-11-23T20:0xZ (session start)
- **Completed:** 2026-11-23T20:23Z
- **Tasks:** 2 (RED, GREEN)
- **Files modified:** 9
- **Commits:** 2 (measured `dc1308a..HEAD`)

## TDD evidence

### RED (Task 1, production untouched — `go build ./...` green)

`go test -count=1 ./internal/interact/`:

```
internal/interact/familysel_test.go:162:59: too many arguments in call to gate.Entitled
	have (context.Context, string, string)
	want (context.Context, string)
```

`go test -count=1 ./internal/hostsvc/`:

```
internal/hostsvc/snapshot_account_test.go:272:37: cannot use func(string, string) bool {…} (value of type func(string, string) bool) as func(family string) bool value in argument to b.BuildAccount
internal/hostsvc/snapshot_account_test.go:609:4: cannot use func(_, mode string) bool {…} as func(family string) bool value in argument to hostsvc.NewSnapshotBuilder(nil).BuildAccount
internal/hostsvc/snapshot_account_test.go:644:36: cannot use allEntitled (value of type func(string, string) bool) as func(family string) bool value in argument to b.BuildAccount
```

`go test -count=1 ./internal/acpfront/`:

```
internal/acpfront/wantmodels_internal_test.go:83:45: cannot use func(string, string) bool {…} as func(family string) bool value in argument to hostsvc.NewSnapshotBuilder(nil).BuildAccount
internal/acpfront/wantmodels_internal_test.go:129:3: cannot use func(family, mode string) bool {…} as func(family string) bool value in argument to hostsvc.NewSnapshotBuilder(nil).BuildAccount
```

`go test -count=1 ./internal/zcode/fake/`:

```
internal/zcode/fake/golden_test.go:1547:12: cannot use func(_ context.Context, family, _ string) bool {…} as func(ctx context.Context, family string) bool value in struct literal
```

The RED gate script printed `RED confirmed: the per-(family, mode) contract is not implemented yet` — production packages built, all four test packages failed naming the intended contract.

### GREEN (Task 2, exact plan `<automated>` gate)

```
go build ./...                                            PASS
go vet ./...                                              PASS
go vet -tags realdesktop ./test/realdesktop/              PASS
go test -race -count=1 ./internal/interact/ ./internal/hostsvc/   PASS (2.40s, 1.37s)
go test -race -count=1 ./internal/acpfront/ ./internal/zcode/fake/ PASS (104.6s, 24.1s)
go test -race -count=1 ./...                              PASS (all packages)
golangci-lint run                                         exactly the 10 documented pre-existing acpfront test findings, 0 in any touched file
```

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): lock per-mode account selection at the gate, overlay and want list** — `6f6e5b9` (test)
2. **Task 2 (GREEN): push only the selected connection mode per account family** — `b65901f` (feat)

Docs artifacts (PLAN/SUMMARY/STATE) are left to the orchestrator per the execution constraints.

## Files Created/Modified

- `internal/hostsvc/snapshot.go` — `builtinProviderRule.Mode` captured verbatim from `config.access.mode`; `BuildAccount` widened to `func(family, mode string) bool`; gate call passes both; order preserved (access type → hidden skip → gate); comments record both omission sources.
- `internal/interact/familysel.go` — `Entitled(ctx, family, mode)` requires `kind == mode`; `codingPlanSelectionKind` deleted; apiKey veto, store-token requirement, log codes and boolean-only fields unchanged (`familyMode` local avoids shadowing).
- `internal/acpfront/auth.go` — `AuthConfig.Entitle` widened to `(ctx, family, mode)`; BuildAccount closure forwards both; want-list comment names both omission sources.
- `internal/interact/familysel_test.go` — per-kind arms (individual/team/start), per-kind mismatch, empty-mode, unselected-family; all calls pass a mode; existing degradation arms and no-material log locks retained.
- `internal/hostsvc/snapshot_account_test.go` — visible `account:zai-start-plan` fixture rule (mode start-plan, GLM-4.6-Air, no visibility field); `wantZhipuProviders` now three rules; `TestAccountSnapshotModeSelection` (individual vs start-plan selections) plus the gate-omission rewrite over both zai rules.
- `internal/hostsvc/snapshot_builtinfile_test.go` — callback widened to the two-argument form.
- `internal/acpfront/wantmodels_internal_test.go` — visible team-mode companion rule; all-pass want list now three sorted pairs; new `TestWantModelsFromSnapshotOmitsNonSelectedMode`.
- `internal/acpfront/auth_test.go` — Entitle closure widened; coding-plan arm expectations untouched.
- `internal/zcode/fake/golden_test.go` — all four Entitle closures widened; no scenario/golden file edited.

## Decisions Made

- Exact `kind == mode` comparison only; off-peak/highspeed occur as rule modes but never as selection kinds, so they can never be entitled.
- An absent/empty rule mode decodes as `""` and never equals a non-empty selected kind (and `selectionKind` rejects empty kinds), so legacy/partial rules are omitted rather than entitled.
- The live selection `{zai: individual-coding-plan, bigmodel: start-plan}` therefore pushes `account:bigmodel-start-plan` when the bigmodel token is present — intended desktop-parity behavior, not a defect.

## Deviations from Plan

None - plan executed exactly as written. The only adjustment was test-local constants (provider ids, envelope members, the `kind` selection key) to keep the new arms goconst-clean; they are folded into the Task 1 commit and change no assertion.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The wrapper's account overlay now matches the desktop's own per-(family, connection-mode) registry decision.
- Real-desktop coding-plan authenticate is proven at build/vet level to adapt the widened `machinery.gate.Entitled` method value with no edit to `cmd/zcode-acp/main.go` or `test/realdesktop/codingplan_test.go`.

---
*Quick task: 261123-vyl*
*Completed: 2026-11-23*

## Self-Check: PASSED

- All nine declared files exist; commits `6f6e5b9` (test) and `b65901f` (feat) are ancestors of HEAD.
- Working tree carries only the untracked SUMMARY.md (docs commit left to the orchestrator by design).
