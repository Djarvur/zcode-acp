---
phase: 06-oauth-coding-plan-path-model-picker
plan: 04
subsystem: session-config
tags: [config-options, model-picker, settings-snapshot, sessionCapabilities, select-variant, verbatim-mirror, tdd]

# Dependency graph
requires:
  - phase: 04-acp-front-minimal-chat-loop
    provides: the D-03 session/new contract (create/subscribe/setModel follow-up), the exact-shape initialize honesty locks both tiers (widened deliberately here), the composed front harness (ComposeFront + the client tee), the acpport scripted-Handlers adapter contract tests, and the SDK select vocabulary pins (v0.20.0)
  - phase: 05-interactions-full-chat-cycle-api-key
    provides: the WR-07 per-session map bounding posture (recordCwd/dropCwd and its SESS-01 teardown site), digCreateAvailable's capture-verbatim map-dig discipline (readback.go), and the reply-snapshot fact (Pattern 7) this plan's capture rides
provides:
  - acpfront.ParseSettings(raw json.RawMessage) (Settings, bool) + the Settings vocabulary (Model{Current/LastUsed/Available{Ref, Label, Description, ReasoningLevels, DefaultLevel}}, ThoughtLevel{Enabled, Current, DefaultLevel, Available}) — one typed dig-out over reply results, capture-verbatim camelCase, undecodable = absent, never a panic
  - acpfront.BuildConfigOptions(s Settings) []acpport.ConfigOptionDef — the verbatim mirror (D-08: Value = providerId + "/" + modelId, labels/descriptions as reported, the disabledReason entry present), per-model reasoning.levels thought entries with the thoughtLevel.available fallback (D-09), currentValue from current/lastUsed and enabled-only thoughtLevel.current (D-10), and the honest omission arms (locked OQ2)
  - acpport.ConfigOptionDef / SelectOptionDef + the ConfigCategoryModel/ThoughtLevel wire-string constants — the neutral config-option vocabulary (SDK types never cross the seam)
  - acpport.Handlers.NewSession widened: (ctx, NewSessionParams) (string, []ConfigOptionDef, error) with the D-10 doc contract; agent.go advertises agentCapabilities.sessionCapabilities.options UNCONDITIONALLY (D-11) and assembles the SDK select variants exclusively (the SDK injects the type discriminant; union Validate asserted)
  - the per-session confirmed-settings store: Front.recordSettings/latestSettings/dropSettings (settingsMu-guarded map, recorded at create + again at the setModel follow-up, dropped in the SESS-01 teardown beside dropCwd — WR-07)
  - tests: TestConfigOptionsTranslator, TestInitCapabilities, TestNewSessionAssemblesConfigOptions, TestNewSessionConfigOptions (harness vectors over scripted reply snapshots incl. the per-session store vector)
affects: [06-05 authenticate arm (its read-back surfaces render through this translator), 06-06 session/set_config_option (consumes the confirmed store + re-emits BuildConfigOptions output), phase-7 Orca /model contract]

# Actuals (#2632) — pairs with the plan's estimate (56000 tokens) to calibrate.
actuals:
  tokens: 19385    # chars/4 over the realized diff (77541 diff chars: configoptions.go +310, tests +788, agent.go/types.go +98, front.go +80, evidence 3 records)
  tasks: 3
  commits: 6      # MEASURED: git rev-list --count f39ce41..HEAD

# Tech tracking
tech-stack:
  added: []       # stdlib + the already-pinned acp-go-sdk v0.20.0 config-options surface — zero new dependencies (T-06-SC held)
  patterns: [compile-first skeleton RED (06-01/06-02/06-03 precedent), capture-verbatim typed dig-out (digCreateAvailable sibling), verbatim-mirror vector discipline, reply-is-the-confirmation capture (Pattern 7), per-session map bounded with the registry (WR-07), SDK-variant-only wire assembly with union Validate]

key-files:
  created:
    - internal/acpfront/configoptions.go
    - internal/acpfront/configoptions_test.go
  modified:
    - internal/acpport/types.go
    - internal/acpport/agent.go
    - internal/acpport/agent_test.go
    - internal/acpfront/front.go
    - internal/acpfront/lifecycle.go
    - internal/acpfront/initialize_test.go
    - internal/acpfront/newsession_test.go
    - internal/acpfront/lifecycle_test.go
    - internal/acpfront/tool_stream_test.go

key-decisions:
  - "Neutral DTO naming follows the AuthMethodDef pattern — acpport.ConfigOptionDef/SelectOptionDef (the artifacts harvesting contract) — resolving the plan's Task-1 'ConfigOption{...}' shorthand in favor of the suffixed forms; the field set (ID, Name, Category, CurrentValue, Options) is the authoritative lock"
  - "The compile-first skeleton RED carries the inert acpport DTO vocabulary in the RED commit: the translator vectors reference []acpport.ConfigOptionDef, so the vocabulary must exist for RED to fail on planned assertions rather than build failures (06-01/06-02/06-03 precedent, RED_EVIDENCE_OK on all three records)"
  - "The SDK v0.20.0 carries initialize capabilities as VALUE structs — the advertisement is agentCapabilities.sessionCapabilities.options = &acp.SessionOptionsCapabilities{} (the research literal's &acp.SessionCapabilities{Options:…} adapted to the real shape; the wire bytes are identical and byte-locked both tiers)"
  - "Option-level names (Model / Thought level) are wrapper-owned identity the backend does not supply; the D-08 verbatim mirror governs the backend-reported ENTRIES (labels/descriptions/order), which the vectors lock element-for-element"
  - "Omission arms extend the identity discipline: ref-less available entries and value-less levels are skipped (nothing without a wire identity renders), and an option whose entry set comes out empty is omitted whole — a smaller honest list beats an empty or fabricated select"
  - "Front.NewSession's signature widening rode Task 2's GREEN commit with nil options (Rule 3 — the Handlers widening is compile-blocking for acpfront); Task 3 filled the capture"

patterns-established:
  - "Reply-is-the-confirmation capture: NewSession records the create reply and AGAIN the setModel follow-up (later overwrites) before rendering — the pattern every set_config_option re-render (06-06) must reuse"
  - "Scripted reply snapshots at the harness: programmatic fake.Scenario rules with fail-loud unknown_method guards render live-reply-shaped settings snapshots — the vector shape 06-05's read-back and 06-06's set_config_option harnesses reuse"
  - "Seam-scoped SDK assembly: acpport.configOptionSelects is the single translation point from the neutral vocabulary onto acp.SessionConfigOptionSelect (Ungrouped options array, SDK-injected type discriminant), Validate-asserted in the adapter test"

requirements-completed: [MODEL-01, AUTH-03]  # copied verbatim from the plan frontmatter; see Decisions — ready-ids gates the marking

plan_head_before: f39ce41d336ff514fb0553f6f4835484d737328f
plan_head_after: 7d64208cd15babf9b91b0605a8c84d5c99154c38

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The translator mirrors the fixture snapshot verbatim: one entry per settings.model.available with Value = providerId + \"/\" + modelId, Name/Description as reported (the disabledReason-carrying entry PRESENT), currentValue = the confirmed model.current serialization — no filtering, renaming, enrichment (D-08, T-06-10)"
    requirement: MODEL-01
    verification:
      - kind: unit
        ref: "internal/acpfront/configoptions_test.go#TestConfigOptionsTranslator/verbatim_mirror_of_the_settings_snapshot_(D-08)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Thought-level entries come from the BOUND model's reasoning.levels; the settings.thoughtLevel.available fallback fires only when the bound model carries none (D-09 — per-model metadata as reported, never a hardcoded list)"
    requirement: MODEL-01
    verification:
      - kind: unit
        ref: "internal/acpfront/configoptions_test.go#TestConfigOptionsTranslator/thoughtLevel.available_fallback_when_the_bound_model_carries_no_reasoning_block_(D-09)"
        status: pass
    human_judgment: false
  - id: D3
    description: "currentValue is server-confirmed only: the no-current/no-lastUsed snapshot omits the model entry entirely; thoughtLevel renders only under enabled with a non-empty current; lastUsed binds when current is absent (D-10 + locked OQ2, T-06-11)"
    requirement: MODEL-01
    verification:
      - kind: unit
        ref: "internal/acpfront/configoptions_test.go#TestConfigOptionsTranslator/no_current_and_no_lastUsed_omits_the_model_entry_entirely_(D-10_+_locked_OQ2)"
        status: pass
      - kind: unit
        ref: "internal/acpfront/configoptions_test.go#TestConfigOptionsTranslator/thoughtLevel_disabled_omits_the_thought-level_entry_(D-10)"
        status: pass
      - kind: unit
        ref: "internal/acpfront/newsession_test.go#TestNewSessionConfigOptions/a_reply_with_no_current_and_no_lastUsed_omits_the_model_entry_(D-10_+_locked_OQ2)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Malformed snapshots degrade to absent fields, never a panic: garbage/empty/null bytes, wrong-typed fields, ref-less entries, level garbage all dig to zero values and render honest omissions (robustness vector)"
    requirement: MODEL-01
    verification:
      - kind: unit
        ref: "internal/acpfront/configoptions_test.go#TestConfigOptionsTranslator/malformed_snapshots_degrade_to_absent_fields,_never_a_panic"
        status: pass
    human_judgment: false
  - id: D5
    description: "initialize advertises agentCapabilities.sessionCapabilities.options (the SDK empty-struct opt-in clients gate session/set_config_option on) UNCONDITIONALLY — and every phase-4 honesty lock stays byte-asserted alongside the widened shape on both tiers (D-11, protocol version, real agentInfo, both auth methods, zero fs/terminal)"
    requirement: MODEL-01
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestInitCapabilities"
        status: pass
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestInitializeAssemblesHonestReport"
        status: pass
      - kind: unit
        ref: "internal/acpfront/initialize_test.go#TestInitializeHonestCapabilities"
        status: pass
    human_judgment: false
  - id: D6
    description: "The adapter assembles NewSessionResponse.ConfigOptions exclusively via acp.SessionConfigOptionSelect (Ungrouped options array, SDK-injected type discriminant — nothing hand-marshaled, T-06-12), the union passes its own Validate, and the byte-locked frame pins category/currentValue/entries verbatim; SDK config-option types appear ONLY inside internal/acpport (seam lock still green)"
    requirement: MODEL-01
    verification:
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestNewSessionAssemblesConfigOptions"
        status: pass
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestSDKImportBoundary"
        status: pass
    human_judgment: false
  - id: D7
    description: "session/new returns configOptions built from the live reply snapshots: the setModel follow-up reply OVERRIDES the create reply (the D-10 confirmation vector — the older current must not survive), the no-model arm renders from the create reply, and the confirmed store is per-session (two create rules render each session's own snapshot; dropSettings joined into the SESS-01 teardown beside dropCwd — WR-07)"
    requirement: MODEL-01
    verification:
      - kind: unit
        ref: "internal/acpfront/newsession_test.go#TestNewSessionConfigOptions/the_setModel_follow-up_reply_is_the_confirmation_(D-10)"
        status: pass
      - kind: unit
        ref: "internal/acpfront/newsession_test.go#TestNewSessionConfigOptions/the_no-model_arm_renders_from_the_create_reply"
        status: pass
      - kind: unit
        ref: "internal/acpfront/newsession_test.go#TestNewSessionConfigOptions/the_confirmed_store_is_per-session_—_the_second_session_renders_its_own_snapshot"
        status: pass
    human_judgment: false
  - id: D8
    description: "Both auth paths render identically through the one translator: the fixtures mix zai-api and account-prefixed provider ids and the translator consumes only reply snapshots — path-independence by construction (D-11); the coding-plan arm's live-models proof rides 06-05's read-back"
    requirement: AUTH-03
    verification:
      - kind: unit
        ref: "internal/acpfront/configoptions_test.go#TestConfigOptionsTranslator/verbatim_mirror_of_the_settings_snapshot_(D-08)"
        status: pass
      - kind: unit
        ref: "internal/acpport/agent_test.go#TestInitCapabilities"
        status: pass
    human_judgment: true
    rationale: "D-11's full claim (whatever models the coding-plan registry reports after auth actually appear in the picker) needs the 06-05 coding-plan authenticate arm and the D-12 real-account run; this plan proves the shared machinery and the path-independent advertisement by vector, not the live coding-plan render."

# Metrics
duration: 24min
completed: 2026-11-22
status: complete
---

# Phase 6 Plan 04: Config-Options Translator, Advertisement & session/new Options Summary

**The settings-snapshot -> configOptions translator (verbatim mirror, D-08/D-09/D-10 omission arms), the unconditional `sessionCapabilities.options` advertisement, and `session/new` returning the full honest option list from the live create/setModel reply snapshots — both auth paths, SDK select variants only, `-race` green**

## Performance

- **Duration:** 24 min
- **Started:** 2026-11-22T01:21:23Z
- **Completed:** 2026-11-22T01:45:34Z
- **Tasks:** 3
- **Files modified:** 11 (2 created code, 7 modified code/tests, 2 test call-site fixes) + 3 evidence records

## Accomplishments

- `acpfront.ParseSettings` + `BuildConfigOptions` (MODEL-01): the digCreateAvailable dig discipline extended to the full settings snapshot — capture-verbatim camelCase, comma-ok assertions, undecodable = absent, never a panic; the translator mirrors `settings.model.available` verbatim (Value = `providerId + "/" + modelId`, labels/descriptions as reported, the disabledReason entry present), drives thought-level entries from the bound model's `reasoning.levels` with the `thoughtLevel.available` fallback (D-09), reads currentValue ONLY from `settings.model.current`/`lastUsed` and `enabled`-guarded `thoughtLevel.current` (D-10), and omits honestly (no binding / disabled / empty current / missing block / empty entry set — locked OQ2)
- initialize advertises `agentCapabilities.sessionCapabilities.options` (`&acp.SessionOptionsCapabilities{}`, the SDK empty-struct opt-in) UNCONDITIONALLY — D-11 path independence — with every phase-4 honesty byte-lock widened deliberately and still asserted on both tiers (no fs/terminal/loadSession anywhere)
- `acpport.Handlers.NewSession` widened to `(sessionID, []ConfigOptionDef, error)` with the D-10 doc contract; the adapter assembles the SDK select variants exclusively (`acp.SessionConfigOptionSelect` + the ungrouped options array; the SDK injects the `type:"select"` discriminant — T-06-12) and the assembled union passes the SDK's own Validate; the neutral DTOs (`ConfigOptionDef`/`SelectOptionDef` + category constants) never leak SDK types across the seam (D-01, seam lock green)
- `session/new` returns the full configOptions list from the CONFIRMED reply snapshots: the create reply is captured, and the `setModel` follow-up reply overwrites it (the follow-up IS the D-10 confirmation — the create reply's older current never survives); the per-session confirmed-settings store (`recordSettings`/`latestSettings`/`dropSettings`) is mutex-guarded and dropped in the SESS-01 teardown beside `dropCwd` (WR-07)
- Harness vectors over scripted reply snapshots prove the follow-up-overrides arm, the no-model arm, the unbound-session omission arm, per-session store independence, and the both-paths (zai-api + account-prefixed) fixtures — wave gate `go test -race ./internal/acpfront/... ./internal/acpport/...` green

## Task Commits

Each task was committed atomically (TDD: RED -> GREEN per task; the tracer gate re-ran the tracer `<verify>` end-to-end green before expansion):

1. **Task 1: Tracer — settings snapshot -> configOptions translator** — `3bcccda` (test: RED, skeleton + DTO vocabulary + evidence) -> `5f7b4b7` (feat: GREEN, dig-out + translator)
2. **Task 2: Capability advertisement + Handlers widen + SDK select assembly** — `fcbfafd` (test: RED, widened byte-locks + assembly vector + evidence) -> `dd2ead3` (feat: GREEN, advertisement + `configOptionSelects` + signature widening)
3. **Task 3: session/new settings capture + per-session confirmed store** — `d61d529` (test: RED, harness vectors + evidence) -> `7d64208` (feat: GREEN, capture + store + teardown join)

**Plan metadata:** pending (docs commit follows this summary)

_Note: TDD tasks carry test -> feat commit pairs; all three RED records validated `RED_EVIDENCE_OK` via `gsd_run check tdd-red-evidence`._

## Files Created/Modified

- `internal/acpfront/configoptions.go` (NEW, +310) — `ParseSettings`, `BuildConfigOptions`, the `Settings`/`ModelRef`/`ModelEntry`/`ReasoningLevel`/`LevelEntry` dig vocabulary, and the omission-arm builders
- `internal/acpfront/configoptions_test.go` (NEW, +478) — the fixture snapshots (reply-result shaped), the verbatim-mirror/fallback/omission/robustness vectors, the element-for-element assertion helper
- `internal/acpport/types.go` (+40) — `ConfigOptionDef`/`SelectOptionDef` with doc contracts + `ConfigCategoryModel`/`ConfigCategoryThoughtLevel`; the widened `Handlers.NewSession` doc contract naming D-10
- `internal/acpport/agent.go` (+66/-14) — the unconditional `sessionCapabilities.options` advertisement in `initializeResponse`; `configOptionSelects` (the single SDK select-variant translation point); the widened `NewSession` delegate
- `internal/acpfront/front.go` (+80) — the `settings` store (fields, lazy init, `recordSettings`/`latestSettings`/`dropSettings`), the create + follow-up captures, and the confirmed-snapshot render
- `internal/acpfront/lifecycle.go` (+1) — `dropSettings` joined beside `dropCwd` in the SESS-01 teardown
- `internal/acpport/agent_test.go` (+126) — `TestInitCapabilities`, `TestNewSessionAssemblesConfigOptions` + the byte-locked frame + the canned neutral options; the widened `honestInitializeFrame`; the scripted-Handlers widening
- `internal/acpfront/initialize_test.go` (+4/-3) — the widened probe shape (`"sessionCapabilities":{"options":{}}`)
- `internal/acpfront/newsession_test.go` (+305) — `TestNewSessionConfigOptions` (four vectors) + the programmatic `settingsScenario` + the verbatim current/entries dig helper
- `internal/acpfront/lifecycle_test.go`, `internal/acpfront/tool_stream_test.go` (call-site fixes) — the two direct `front.NewSession` callers follow the widened signature
- `.planning/phases/06-oauth-coding-plan-path-model-picker/evidence/06-04-red-{translator,capabilities,capture}.json` — the three RED records (all `RED_EVIDENCE_OK`)

## Decisions Made

- **DTO naming = `ConfigOptionDef`/`SelectOptionDef`** (the AuthMethodDef pattern + the artifacts contract), resolving the plan's Task-1 `ConfigOption{...}` shorthand; the field set is the authoritative lock
- **The inert acpport DTO vocabulary landed in Task 1's RED commit** — the translator vectors reference `[]acpport.ConfigOptionDef`, so compile-first skeleton RED requires it (all three RED records `RED_EVIDENCE_OK`)
- **The advertisement shape follows the real SDK v0.20.0**: capabilities are value structs, so `initializeResponse` sets `AgentCapabilities.SessionCapabilities.Options = &acp.SessionOptionsCapabilities{}` — the research literal adapted; wire bytes byte-locked both tiers
- **Option-level names are wrapper-owned** ("Model"/"Thought level") — the D-08 verbatim mirror governs the backend-reported entries, which the backend does not label; vectors lock entries element-for-element
- **Identity-first omission discipline**: ref-less available entries and value-less levels are skipped; an option whose entry set empties is omitted whole
- MODEL-01/AUTH-03 marking deferred to the `requirements.ready-ids` gate — 06-05/06-08 may declare sibling work on the same requirement ids; only what the gate reports ready gets marked

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] Front.NewSession signature widening rode Task 2's GREEN commit**
- **Found during:** Task 2 (GREEN implementation)
- **Issue:** widening `acpport.Handlers.NewSession` breaks `acpfront` compilation — the repo cannot build at Task 2 GREEN with the old front signature; Task 3's file list owns front.go, not Task 2's
- **Fix:** the minimal widening (returns `nil` options) landed in Task 2's commit with a comment pointing at Task 3's capture; Task 3 filled it
- **Files modified:** internal/acpfront/front.go (+ its two direct test callers in lifecycle_test.go / tool_stream_test.go)
- **Verification:** both suites `-race` green at Task 2 GREEN (`dd2ead3`)
- **Committed in:** dd2ead3

**2. [Rule 3 - Blocker] lifecycle.go gained the one-line `dropSettings` teardown join**
- **Found during:** Task 3 (GREEN implementation)
- **Issue:** the acceptance criterion "dropSettings called in the same teardown path (source assertion)" names the SESS-01 teardown in lifecycle.go — outside the task's two-file list
- **Fix:** `f.dropSettings(sessionID)` joined beside `f.dropCwd(sessionID)` at the registry-release site (WR-07)
- **Files modified:** internal/acpfront/lifecycle.go
- **Verification:** the harness vectors green; the store drops with the registry on close
- **Committed in:** 7d64208

**3. [Rule 1 - Bug] Two blanket return-shape rewrites touched Prompt/createSessionID**
- **Found during:** Task 2 (GREEN implementation)
- **Issue:** the mechanical three-value return rewrite in front.go caught two `return "", err` / `return sid, nil` sites belonging to `Prompt` and `createSessionID` (wrong arity there)
- **Fix:** restored both to their two-value forms; build + full suites green
- **Files modified:** internal/acpfront/front.go
- **Verification:** `go build ./...` clean; both suites `-race` green
- **Committed in:** dd2ead3 (folded into the Task 2 GREEN commit)

---

**Total deviations:** 3 auto-fixed (1 signature-widening consequence, 1 acceptance-driven teardown join, 1 self-caught rewrite slip). **Impact on plan:** None — all three were mechanical consequences of the locked design, fixed within their tasks and locked by the task vectors. No scope creep.

## Issues Encountered

- None — all three RED->GREEN cycles landed on planned assertions; the tracer feedback gate re-ran the tracer `<verify>` end-to-end green (auto mode) before expansion to Tasks 2-3.

## Authentication Gates

None — no auth flows were exercised by this plan.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None — no placeholder data paths; every rendered option traces to a backend reply snapshot (the omission arms are the honesty mechanism, vector-locked).

## Next Phase Readiness

- 06-05 (coding-plan authenticate arm) consumes: the translator (its post-auth read-back models render through the same verbatim mirror), `PushEnvelope`/`WatchRegistry` (untouched), and the entitlement gate from 06-03
- 06-06 (`session/set_config_option`) consumes: the widened `Handlers.NewSession` shape (the delegate pattern for the set_config_option delegate), `configOptionSelects` (re-emitting the full list), the per-session confirmed store (`recordSettings` — the mutation reply overwrites), and the omission arms (the invalid-option -32602 arm names the allowed set from the rendered options)
- 06-07/06-08 unaffected here — no runtime-header or registry machinery touched
- No blockers; both suites `-race` green at HEAD `7d64208` (wave gate green; `mise run ci` runs at the wave merge per the verification contract)

---
*Phase: 06-oauth-coding-plan-path-model-picker*
*Completed: 2026-11-22*

## Self-Check: PASSED

- `internal/acpfront/configoptions.go`, `internal/acpfront/configoptions_test.go`, and this SUMMARY exist on disk
- All six task commits are ancestors of HEAD: `3bcccda`, `5f7b4b7`, `fcbfafd`, `dd2ead3`, `d61d529`, `7d64208` (docs commit `36d56a0` carries this summary + STATE.md + ROADMAP.md)
- Commits measured against the persisted plan ledger `f39ce41..HEAD` = 7 at self-check time (frontmatter `commits: 6` = the six task commits, docs commit excluded — the 06-03 recording precedent; `plan_head_before`/`plan_head_after` recorded from the ledger and HEAD at SUMMARY write)
- Wave gate green: `go test -race -count=1 ./internal/acpfront/... ./internal/acpport/...` — both packages ok
