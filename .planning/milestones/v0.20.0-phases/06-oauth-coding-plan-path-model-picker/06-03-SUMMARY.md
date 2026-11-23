---
phase: 06-oauth-coding-plan-path-model-picker
plan: 03
subsystem: auth
tags: [account-snapshot, zhipu-account, builtin-table, verbatim-mirror, entitlement-gate, provider-overlay, tdd]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key
    provides: the hostsvc snapshot posture this plan extends — BuildRevision's strict envelope + empty overlay (byte-untouched here), readBuiltinRevision's file-read + typed-struct discipline, PushEnvelope/parseAck (the 06-05 consumer), supervisor.BuiltinRevision's zcode-builtin:<rev>:<sha256(path)> formula, the scripted-peer test harness
  - phase: 06-oauth-coding-plan-path-model-picker (plan 01)
    provides: the desktop builtin-table schema knowledge (revision-44 live probe — providerId / config.builtinModelIds / config.access.{type,mode,accountType})
provides:
  - (*SnapshotBuilder).BuildAccount(install supervisor.Install, entitled func(family string) bool) (Snapshot, error) — the coding-plan NON-empty provider/updateAccountConfig overlay: one providers entry per builtin-table rule with config.access.type == zhipu-account ({builtinModelIds verbatim, access {type, entitled true}}), per-provider states {availability available, entitled true, current true}
  - readBuiltinProviderRules(path) ([]builtinProviderRule, error) — the providerRules dig-out over the SAME zcode-builtin.json readBuiltinRevision opens; a malformed table (undecodable JSON, missing providerRules, a rule missing providerId/access/model-list) answers a coded build error — never a degraded overlay
  - the entitlement-gate seam: BuildAccount RECEIVES entitled(family); family = config.access.accountType; a rejected family is omitted from providers AND states entirely — the gate may only omit, never edit (D-08/T-06-09); the production gate (setting.json selections/modes + store oauth token) lands in 06-05
  - the zero-entitled-providers coded error: zero zhipu-account rules surviving the gate answers a named misconfiguration error with nil Params — never an empty overlay silently pushed on the coding-plan path (AUTH-02 empty probe resolved explicit)
  - TestPushEnvelopeCompat: BuildAccount's Snapshot feeds the EXISTING PushEnvelope unchanged — the 06-05 consumer contract proven
affects: [06-05 authenticate arm (pushes BuildAccount's Params + derives the read-back want list from the built providers), 06-07 runtime-headers]

# Actuals (#2632) — pairs with the plan's estimate (34000 tokens) to calibrate.
actuals:
  tokens: 8060    # chars/4 over the realized diff (732 added lines, ~32.2k diff chars: snapshot.go +162, test file +550, evidence 2x10)
  tasks: 2
  commits: 4      # MEASURED: git rev-list --count 38aa111c..HEAD

# Tech tracking
tech-stack:
  added: []       # stdlib only (encoding/json over the already-resolved builtin file) — zero new dependencies (T-06-SC mitigation held)
  patterns: [compile-first skeleton RED, typed-struct dig-out with coded malformed arms (readBuiltinRevision sibling), caller-supplied entitlement gate (omission-only), verbatim-mirror test discipline (every emitted string traced to the fixture table)]

key-files:
  created:
    - internal/hostsvc/snapshot_account_test.go
  modified:
    - internal/hostsvc/snapshot.go

key-decisions:
  - "The entitlement family is config.access.accountType — pinned from the live revision-44 desktop table this session (V-EXP: 10 rules, families zai/bigmodel exactly matching the setting.json selections keys and the oauth:<family>:access_token store keys); the rule's other fields (group, mode, providerName) are carried by the dig but only providerId/builtinModelIds/access.type/accountType drive the overlay"
  - "One coded error covers BOTH zero-entitled arms — a table with zero zhipu-account rules AND a gate rejecting every family answer 'no zhipu-account provider rules to entitle after the entitlement gate': pushing an empty account overlay would be the api-key shape pushed onto the coding-plan path, unentitling every account model"
  - "The malformed arms use the typed-struct dig with explicit nil checks (not the ack-style map dig) so undecodable JSON, a missing providerRules key, and a rule missing its access block answer DISTINCT coded errors — the undecodable-is-coded-error-here inversion of the ack posture is the T-06 trust-boundary mitigation"
  - "BuildAccount filters zhipu-account BEFORE the gate and validates every rule field it must mirror (providerId, access, builtinModelIds) — a rule that cannot be mirrored verbatim fails the build rather than degrading the overlay"

patterns-established:
  - "Compile-first skeleton RED (04-06/06-01/06-02 precedent): BuildAccount + readBuiltinProviderRules skeletons with zero-value bodies land in the RED commit so the vector fails on planned assertions — both REDs validated RED_EVIDENCE_OK"
  - "Verbatim-mirror test discipline: the fixture table is byte-shaped like the real desktop schema and every emitted string (provider ids, model ids incl. a mixed-case id, access type) is compared element-for-element against the fixture — T-06-09's no-renaming/no-editing lock"
  - "Lock-style subtests at RED: arms the tracer slice already satisfies (gate omission, malformed arms, PushEnvelope compat) are documented expected-green-at-RED in the evidence record (06-02 TestPatMinterBizFail precedent)"

requirements-completed: [AUTH-02]  # copied verbatim from the plan frontmatter; AUTH-02 is phase-spanning — ready-ids reports it blocked until 06-04..06-08 land (see Decisions Made)

plan_head_before: 38aa111cc1d8e063d3ecdb1b52e91848ea9fb915
plan_head_after: 0b630f256fe89d4a0b396c049293df2eebc1cda2

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "BuildAccount mirrors the fixture table's zhipu-account rules verbatim — providers EXACTLY the zhipu rules (apiKey/trial rows leak-proof), builtinModelIds element-for-element, access {type zhipu-account, entitled true}, strict states {availability available, entitled true, current true}, strict four-key envelope"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotVerbatimMirror"
        status: pass
    human_judgment: false
  - id: D2
    description: "basedOnZCodeBuiltinRevision is the exact supervisor.BuiltinRevision output over the runtime-resolved path and changes when the path changes (T-06-08 path-sensitivity)"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotVerbatimMirror"
        status: pass
    human_judgment: false
  - id: D3
    description: "Entitlement gate omission-only discipline: a gate-rejected family vanishes from providers AND states while the entitled family stays intact verbatim"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotGateOmission"
        status: pass
    human_judgment: false
  - id: D4
    description: "Coded failure arms: zero zhipu-account rules answers a named misconfiguration error with nil Params; undecodable JSON, missing providerRules, rule missing access/model-list each answer coded errors — never a partial push, never a panic"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotNoRules"
        status: pass
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestAccountSnapshotMalformedTable"
        status: pass
    human_judgment: false
  - id: D5
    description: "The 06-05 consumer contract + the Phase-5 lock: BuildAccount's Snapshot feeds the existing PushEnvelope unchanged (scripted ack echo), and the Phase-5 empty-overlay builder + its TestSnapshotEnvelope goldens stay byte-untouched (full hostsvc + acpfront suites -race green)"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/hostsvc/snapshot_account_test.go#TestPushEnvelopeCompat"
        status: pass
      - kind: unit
        ref: "internal/hostsvc/hostsvc_test.go#TestSnapshotEnvelope"
        status: pass
    human_judgment: false

# Metrics
duration: 12min
completed: 2026-11-22
status: complete
---

# Phase 6 Plan 03: Coding-Plan Account Snapshot Summary

**BuildAccount — the NON-empty zhipu-account provider overlay mirrored verbatim from the desktop's own builtin table, gated by a caller-supplied entitlement seam, with coded failure arms for zero-rules and malformed tables; the Phase-5 empty-overlay builder byte-untouched**

## Performance

- **Duration:** 12 min
- **Started:** 2026-11-22T01:06:27Z
- **Completed:** 2026-11-22T01:18:30Z
- **Tasks:** 2
- **Files modified:** 4 (2 code + 2 evidence records)

## Accomplishments

- `BuildAccount(install, entitled func(family string) bool)`: the coding-plan snapshot (RESEARCH Pattern 2) — providers carries one entry per builtin-table rule whose `config.access.type == zhipu-account`, each `{builtinModelIds: verbatim, access: {type, entitled: true}}`; states carries `{availability: "available", entitled: true, current: true}` per provider; basedOn stays the exact `supervisor.BuiltinRevision` formula over the runtime-resolved path (T-06-08)
- `readBuiltinProviderRules`: the providerRules dig-out over the SAME `install.BuiltinFile` `readBuiltinRevision` opens — capture-verbatim camelCase (`providerId` / `config.builtinModelIds` / `config.access.type` / `config.access.accountType`, shape pinned against the live revision-44 desktop table this session); a malformed table answers distinct coded errors, never a degraded overlay
- The entitlement-gate seam: the builder RECEIVES the decision and may only omit families — the production gate (setting.json `providerFamilyConnectionSelections`/`modelProviderFamilyModes` + store `oauth:<family>:access_token`) is 06-05's authored input
- Zero-entitled-providers coded error (AUTH-02 empty probe resolved explicit): an account push with nothing to entitle is a misconfiguration with nil Params — never an empty overlay silently pushed on the coding-plan path
- Edge-arm locks: gate omission is omission-only (rejected family gone from providers AND states), malformed tables (4 shapes) fail coded, `TestPushEnvelopeCompat` proves the existing `PushEnvelope` consumes the account Snapshot unchanged (the 06-05 consumer contract)
- The Phase-5 api-key empty overlay and all its locks byte-untouched — full `hostsvc` + `acpfront` suites green under `-race`

## Task Commits

Each task was committed atomically (TDD: RED → GREEN per task):

1. **Task 1: Tracer — BuildAccount overlay end-to-end** — `161601e` (test: RED, skeleton + evidence) → `27d83f4` (feat: GREEN, dig-out + builder)
2. **Task 2: Snapshot edge arms** — `cf1894a` (test: RED, zero-rules target + lock-style arms + evidence) → `0b630f2` (feat: GREEN, the coded zero-check)

**Plan metadata:** pending (docs commit follows this summary)

_Note: TDD tasks carry test → feat commit pairs; both RED records validated `RED_EVIDENCE_OK` via `gsd_run check tdd-red-evidence`._

## Files Created/Modified

- `internal/hostsvc/snapshot.go` (+162) — `BuildAccount`, `readBuiltinProviderRules`, `builtinProviderRule`, the `zhipuAccountAccess` filter constant, the zero-entitled coded error; `Build`/`BuildRevision`/`PushEnvelope` and every existing symbol untouched
- `internal/hostsvc/snapshot_account_test.go` (+550, NEW) — the fixture table in the desktop's own schema (mixed families + leak guards + no-transformation locks), the five named tests, the decode/verbatim assertion helpers
- `.planning/phases/06-oauth-coding-plan-path-model-picker/evidence/06-03-red-account.json` — Task 1 RED record (RED_EVIDENCE_OK)
- `.planning/phases/06-oauth-coding-plan-path-model-picker/evidence/06-03-red-arms.json` — Task 2 RED record (RED_EVIDENCE_OK)

## Decisions Made

- **Family = `config.access.accountType`** — pinned from the live revision-44 table (families `zai`/`bigmodel` exactly match the setting.json selections keys and the `oauth:<family>:access_token` store keys, so the 06-05 gate reads one vocabulary)
- **One coded error for both zero arms** (zero-rules table, all-rejecting gate): "no zhipu-account provider rules to entitle after the entitlement gate" — an empty account overlay would push the api-key shape onto the coding-plan path and unentitle every account model
- **Typed-struct dig with explicit nil checks** for the malformed arms (readBuiltinRevision's sibling posture) — distinct coded errors for undecodable JSON / missing providerRules / rule missing access; the undecodable-is-coded-error inversion of the ack dig-out posture is the trust-boundary mitigation
- **Validate every mirrored field**: a rule missing providerId or builtinModelIds fails the build — a rule that cannot be mirrored verbatim must never degrade the overlay
- AUTH-02 left in `requirements` blocked state — the requirement is phase-spanning; `ready-ids` reports it not ready until 06-04..06-08 land

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Missing zhipu-account filter in the first GREEN**
- **Found during:** Task 1 (GREEN implementation)
- **Issue:** The first `BuildAccount` draft gated families but did not filter `config.access.type == zhipu-account`, so the apiKey and trial fixture rules leaked into providers — caught by `TestAccountSnapshotVerbatimMirror`'s exact-key-set assertion before any commit of a passing-wrong state
- **Fix:** Added the `zhipuAccountAccess` filter constant and the `rule.AccessType != zhipuAccountAccess` guard at the head of the emit loop (zhipu filter BEFORE the gate, per the locked design)
- **Files modified:** internal/hostsvc/snapshot.go
- **Verification:** `go test -race -count=1 -run 'TestAccountSnapshot'` green; leak guards now assert the apiKey/trial rows never appear
- **Committed in:** 27d83f4 (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug, caught by the task's own RED-first vector before it could ship)
**Impact on plan:** None — the deviation was a transient defect inside Task 1's GREEN step, fixed within the task and locked by its acceptance criteria. No scope creep.

## Issues Encountered

- None — both RED→GREEN cycles landed on planned assertions; the tracer feedback gate re-ran the full `<verify>` end-to-end green (auto mode), authorizing expansion to Task 2.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 06-05 (authenticate arm) consumes: `BuildAccount`'s Params push + the read-back want list derived from the built providers (`providerId`/`modelId` of `builtinModelIds` — mirrors-what-we-pushed, never a hardcoded catalog), the entitlement-gate production implementation (setting.json selections/modes + store `oauth:<family>:access_token`), and `PushEnvelope`/`RefreshRegistry`/`WatchRegistry` reused verbatim
- 06-07 (runtime-headers source) unaffected here — the api-key `RuntimeHeadersHandler` and its locks are untouched
- No blockers; the full `hostsvc` and `acpfront` suites are `-race` green at HEAD `0b630f2`

---
*Phase: 06-oauth-coding-plan-path-model-picker*
*Completed: 2026-11-22*

## Self-Check: PASSED

- `internal/hostsvc/snapshot.go`, `internal/hostsvc/snapshot_account_test.go`, and this SUMMARY exist on disk
- All four task commits are ancestors of HEAD: `161601e`, `27d83f4`, `cf1894a`, `0b630f2` (docs commit `fa3613c` carries this summary + STATE.md + ROADMAP.md)
- Commits measured against the persisted plan ledger `38aa111c..HEAD` = 4 (frontmatter `commits: 4`, `plan_head_before`/`plan_head_after` recorded)
