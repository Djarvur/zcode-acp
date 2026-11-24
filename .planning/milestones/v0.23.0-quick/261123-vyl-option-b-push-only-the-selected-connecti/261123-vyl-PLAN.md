---
phase: 261123-vyl
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/interact/familysel_test.go
  - internal/hostsvc/snapshot_account_test.go
  - internal/hostsvc/snapshot_builtinfile_test.go
  - internal/acpfront/wantmodels_internal_test.go
  - internal/acpfront/auth_test.go
  - internal/zcode/fake/golden_test.go
  - internal/hostsvc/snapshot.go
  - internal/interact/familysel.go
  - internal/acpfront/auth.go
autonomous: true
must_haves:
  truths:
    - For the live desktop selection zai=individual-coding-plan and bigmodel=start-plan, the account overlay carries exactly account:zai-individual-coding-plan and account:bigmodel-start-plan; the other visible modes of each family (zai team/start, bigmodel individual/team) are absent from providers AND states.
    - EntitlementGate.Entitled answers true only when the rule's mode equals providerFamilyConnectionSelections[family].kind exactly; a rule mode that is not the selected kind (including an absent/empty mode) answers false, while the apiKey family-mode veto and the store-token requirement keep their previous semantics.
    - BuildAccount's decision seam is per (family, mode) and still only omits rules; the option-A hidden-visibility skip stays in force, runs FIRST, and stays test-locked.
    - The read-back want list mirrors the per-mode-selected overlay: a visible rule whose mode is not the family's selected connection contributes no provider/model pair, so ProbeWantModel can never target it.
    - All degradation arms remain boolean-only with the existing structural log codes; no mode, kind, or token value reaches a log line.
  artifacts:
    - internal/hostsvc/snapshot.go — builtinProviderRule.Mode captured verbatim from config.access.mode; BuildAccount gate becomes func(family, mode string) bool
    - internal/interact/familysel.go — Entitled(ctx, family, mode) compares the desktop selection kind to the rule mode; codingPlanSelectionKind deleted
    - internal/acpfront/auth.go — AuthConfig.Entitle widened to (ctx, family, mode); comments name the second omission source
    - internal/interact/familysel_test.go — per-kind arms (individual/team/start) plus mismatch, unselected-family and empty-mode arms
    - internal/hostsvc/snapshot_account_test.go — visible start-plan fixture rule and TestAccountSnapshotModeSelection
    - internal/acpfront/wantmodels_internal_test.go — non-selected-mode companion rule and want-list omission lock
  key_links:
    - builtin rule config.access.mode -> BuildAccount seam (family, mode) -> EntitlementGate kind==mode -> providers/states omission -> wantModelsFromSnapshot mirror
---

<objective>
Implement Option B for the account overlay: push only the account-provider rule whose connection mode is the one the desktop user actually selected, per provider family. Today the overlay still entitles every VISIBLE mode of an entitled family (`account:zai-individual-coding-plan`, `account:zai-team-coding-plan`, `account:zai-start-plan`), even though the live desktop selection is `{zai: individual-coding-plan, bigmodel: start-plan}`.

Purpose: the desktop's own registry decision is per (family, connection mode). The wrapper already reads the desktop's selection to decide WHETHER a family is entitled, but it decides on the hardcoded `individual-coding-plan` kind and then emits every visible rule of that family. Generalizing the gate to `kind == rule.mode` makes the wrapper's push match the desktop's own selection for individual, team, and start plans, and stops the catalog from carrying plans the user never chose.

Output: `config.access.mode` captured on the builtin rule, a per-(family, mode) entitlement seam through `BuildAccount` and `AuthConfig.Entitle`, a `kind == mode` production gate, updated doc comments, and RED/GREEN test locks at the gate, snapshot, and want-list tiers.

Non-goals (explicit): no client-side configOptions filtering, no change to `mergeCatalogEntries`, no change to the read-back watcher mechanics, no new coded error class. The option-A hidden-visibility skip must remain intact and test-locked.
</objective>

<context>
Verified against the tree at planning time on branch `fix/model-catalog-hidden-providers` (HEAD 751837b; baseline measured green before this plan):

- `internal/hostsvc/snapshot.go:197-252` — `BuildAccount(install, entitled func(family string) bool)` filters `access.type == zhipu-account`, then `config.visibility == "hidden"` (option A, commit 2513949), then the family gate; survivors go into `providers` and `states`.
- `internal/hostsvc/snapshot.go:253-263` — `builtinProviderRule` carries ProviderID/ModelIDs/AccessType/Family/Visibility; no Mode.
- `internal/hostsvc/snapshot.go:355-402` — `digBuiltinRule` decodes `config.access.{type,accountType}`; `config.access.mode` is dropped.
- `internal/interact/familysel.go:27-33,65-99` — the gate compares the selection kind to the hardcoded `codingPlanSelectionKind = "individual-coding-plan"`; `modelProviderFamilyModes[family] != "apiKey"` and `oauth:<family>:access_token` presence complete the rule.
- `internal/acpfront/auth.go:156-160,279-283,380-404` — `AuthConfig.Entitle func(ctx, family string) bool`, the `BuildAccount` closure, and `wantModelsFromSnapshot`.
- `cmd/zcode-acp/main.go:608` and `test/realdesktop/codingplan_test.go:260` wire `Entitle: machinery.gate.Entitled`; method values adapt automatically once the method signature matches — no edit expected in either file.

Desktop evidence (two independent sources, both on this machine):

- The live `~/.zcode/v2/setting.json` carries `providerFamilyConnectionSelections` = `{zai: {kind: individual-coding-plan}, bigmodel: {kind: start-plan}}` and `modelProviderFamilyModes` = `{zai: oauth, bigmodel: oauth}`.
- The installed revision-44 builtin table (`~/.cache/zcode-appimg/squashfs-root/resources/config/provider/zcode-builtin.json`) carries six visible account rules: zai/bigmodel x individual-coding-plan / team-coding-plan / start-plan, plus the four hidden off-peak/highspeed rules. The reported five-mode symptom is exactly this table times the family-only gate.
- The desktop bundle mentions `individual-coding-plan`, `team-coding-plan`, `start-plan`, `off-peak` and `highspeed`; `off-peak`/`highspeed` occur as rule modes but never as the selection `kind`, so an exact `kind == mode` comparison can only ever entitle the three selectable plans.

Fixture/golden audit (already performed, expected to need no fixture change beyond the test files in Task 1): `codingPlanFixtureInstall` (internal/acpfront/auth_test.go) and `codingPlanCorpusInstall` (internal/zcode/fake/golden_test.go) each carry one zhipu-account rule WITHOUT an `access.mode`, so the decoded mode is empty and their family-only closures (which now ignore the mode argument) behave byte-identically. No scenario or golden file under internal/zcode/fake/testdata carries a rule `mode` (grep for `"mode"` returns nothing), so no golden bytes can shift.

Baseline measurements at planning time: `go build ./...` clean; `go vet -tags realdesktop ./test/realdesktop/` clean; `go test -race -count=1 ./internal/hostsvc/ ./internal/interact/ ./internal/acpfront/` green (hostsvc 1.4s, interact 2.4s, acpfront 102s); `golangci-lint run` reports exactly the 10 documented pre-existing findings in internal/acpfront/{catalogmerge,codingplan_probe,setconfigoption,silentauth,switchcatalog}_test.go; `golangci-lint run --new-from-rev=HEAD` reports 0 issues.

Fixture design decision: the extended snapshot fixture gains a visible `account:zai-start-plan` rule with mode `start-plan` and a distinct model id (`GLM-4.6-Air`) so presence/absence is unambiguous; `wantZhipuProviders()` grows to three entries, and the gate-omission arm then expects the TWO zai rules (individual + start). The want-list fixture gains a visible `account:team` rule (mode `team-coding-plan`, model `M-Team`) as the mode-mismatch companion.
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1 (RED): Lock per-(family, mode) selection at the gate, overlay and want-list tiers</name>
  <files>internal/interact/familysel_test.go, internal/hostsvc/snapshot_account_test.go, internal/hostsvc/snapshot_builtinfile_test.go, internal/acpfront/wantmodels_internal_test.go, internal/acpfront/auth_test.go, internal/zcode/fake/golden_test.go</files>
  <behavior>
    - Gate: with selections zai=individual-coding-plan and bigmodel=start-plan, Entitled(zai, "individual-coding-plan") is true while Entitled(zai, "team-coding-plan"), Entitled(zai, "start-plan") and Entitled(zai, "") are false.
    - Gate: with selections zai=team-coding-plan, Entitled(zai, "team-coding-plan") is true and Entitled(zai, "individual-coding-plan") is false; with zai=start-plan, start is true and individual is false; a family with no selection entry is entitled for no mode.
    - Gate: the apiKey veto, the missing-token veto, the missing/undecodable setting degradation and the absent-mode-entry pass keep their previous outcomes under the new signature.
    - Overlay: a per-(family, mode) callback selecting only individual modes yields exactly the individual rules of both families; a callback selecting only start-plan yields exactly account:zai-start-plan — in both cases the start-plan rule (arm 1) and the individual rules (arm 2) are absent from providers AND states, and the hidden off-peak rule never appears.
    - Want list: with a mode-selective callback for the fixture family, the want list contains only the selected-mode rule's provider/model pairs; the hidden rule and the visible team-mode rule contribute nothing.
    - The RED signal is compile-level: production is untouched, so the four test packages must fail to compile naming the new (family, mode) contract, while `go build ./...` stays green.
  </behavior>
  <action>
    In internal/interact/familysel_test.go, add test-local kind constants for the three selectable kinds (`individual-coding-plan`, `team-coding-plan`, `start-plan`) and use them in the fixture selections maps and in every assertion so the file stays goconst-clean. Change every `gate.Entitled` call to pass a rule mode as the third argument. Rewrite arm (c) — the old "a different selection kind vetoes" arm, which asserted bigmodel/start-plan false — into the new semantics: with the existing fixture selections (zai=individual, bigmodel=start), add the bigmodel store token and assert Entitled(bigmodel, "start-plan") true and Entitled(bigmodel, "individual-coding-plan") false. Add a team-kind arm (selections zai=team-coding-plan, zai token present: team true, individual false), a start-kind arm (selections zai=start-plan, zai token present: start true, individual false), an empty-mode arm (Entitled(zai, "") false) and an unselected-family arm (a family absent from the selections maps answers false for any mode). Keep arms (b) apiKey veto, (d) missing token, (e) missing setting, (f) undecodable setting and (g) absent mode entry, adapting only the call signatures; keep the existing log-code and no-material assertions. Update the TestFamilySelections doc comment to state the per-rule semantics (a family is entitled for a rule when the desktop's selected connection kind IS that rule's mode; off-peak/highspeed never appear as kinds).

    In internal/hostsvc/snapshot_account_test.go, extend accountFixtureTable with a visible `account:zai-start-plan` rule (providerName "Z.AI Start Plan", group zai-family, builtinModelIds ["GLM-4.6-Air"], access {type zhipu-account, mode start-plan, accountType zai}, no visibility field so the default-visible arm is exercised) and update the fixture doc comment. Add the rule to wantZhipuProviders(). Update TestAccountSnapshotVerbatimMirror's inline callbacks to the two-argument form and its count and failure wording to the three visible rules. Rewrite TestAccountSnapshotGateOmission to loop over the two expected zai providers (individual + start-plan) with the existing assertModelIDsVerbatim helper, and assert both appear in providers and states while the bigmodel family is omitted. Add TestAccountSnapshotModeSelection with two t.Run arms and one small shared helper that decodes the envelope and compares the provider AND state key sets: arm "individual selection" uses a callback selecting mode individual-coding-plan only and expects exactly {account:zai-individual-coding-plan, account:bigmodel-individual-coding-plan} with account:zai-start-plan absent from both maps; arm "start-plan selection" uses a callback selecting mode start-plan only and expects exactly {account:zai-start-plan} with both individual rules absent from both maps. Keep each function under the repo's funlen threshold by using the helper rather than duplicating assertions. Convert allEntitled to `func(string, string) bool` returning true and update the remaining BuildAccount callbacks (NoRules, AllHidden, Malformed arms, PushEnvelopeCompat) to the two-argument form.

    In internal/hostsvc/snapshot_builtinfile_test.go, change the TestAccountBuildUsesRecordedBuiltinFile callback to `func(string, string) bool { return true }`; no expectation changes.

    In internal/acpfront/wantmodels_internal_test.go, add a third fixture rule `account:team` to wantListFixtureTable (same family, builtinModelIds ["M-Team"], access mode team-coding-plan, visibility "visible") and update the table doc comment. Update TestWantModelsFromSnapshotOmitsHidden's BuildAccount callback to the two-argument all-pass form and its exact want list to the three sorted visible pairs including account:team/M-Team. Add TestWantModelsFromSnapshotOmitsNonSelectedMode: build with a callback that selects only the fixture family and mode individual-coding-plan, then assert the want list is exactly [account:visible/M-A, account:visible/M-B], that account:team and account:hidden are absent from both the providers map and the want list, and that ProbeWantModel of the list is one of those two entries. Keep the fixture family and mode spellings derived from the fixture table constants (no production catalog literals).

    In internal/acpfront/auth_test.go, change the codingPlanConfig Entitle closure to `func(_ context.Context, family, _ string) bool { return family == cpFixAccountFamily }`; do NOT add an access.mode to codingPlanFixtureInstall (its rule legitimately decodes as mode ""), and leave the coding-plan arm expectations, including the providerCount 1 ack and the pushed-params substring check, untouched.

    In internal/zcode/fake/golden_test.go, change all four Entitle closures to `func(_ context.Context, family, _ string) bool { return family == cpCorpusAccountFamily }`; do NOT add an access.mode to codingPlanCorpusInstall and do not regenerate or edit any scenario/golden file — behavior must stay byte-identical.

    Run `go build ./...` (must pass) and the four RED gates below; record the compiler output in the task summary. Commit only the six test files with message `test(261123-vyl): lock per-mode account selection at the gate, overlay and want list`.
  </action>
  <verify>
    <automated>go build ./... || exit 1; out1=$(go test -count=1 ./internal/interact/ 2>&1) && { echo 'EXPECTED RED: Entitled is still family-only'; exit 1; }; printf '%s\n' "$out1" | grep -q 'too many arguments' || { printf '%s\n' "$out1"; exit 1; }; out2=$(go test -count=1 ./internal/hostsvc/ 2>&1) && { echo 'EXPECTED RED: BuildAccount still takes a family-only gate'; exit 1; }; printf '%s\n' "$out2" | grep -q 'cannot use' || { printf '%s\n' "$out2"; exit 1; }; out3=$(go test -count=1 ./internal/acpfront/ 2>&1) && { echo 'EXPECTED RED: AuthConfig.Entitle is still family-only'; exit 1; }; printf '%s\n' "$out3" | grep -q 'cannot use' || { printf '%s\n' "$out3"; exit 1; }; out4=$(go test -count=1 ./internal/zcode/fake/ 2>&1) && { echo 'EXPECTED RED: golden closures are still family-only'; exit 1; }; printf '%s\n' "$out4" | grep -q 'cannot use' || { printf '%s\n' "$out4"; exit 1; }; echo 'RED confirmed: the per-(family, mode) contract is not implemented yet'</automated>
  </verify>
  <done>Production packages still build (`go build ./...` green) and the four test packages fail to compile for the intended reason: the interact arm reports three arguments against the two-argument Entitled, and the hostsvc/acpfront/fake arms report func(string, string) bool where a family-only func is still expected. The new assertions enumerate the full per-mode contract, the option-A hidden locks and all pre-existing arms are preserved, and no production or fixture-golden file is modified. The RED output is quoted in the task summary.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2 (GREEN): Capture config.access.mode and key the overlay decision by (family, mode)</name>
  <files>internal/hostsvc/snapshot.go, internal/interact/familysel.go, internal/acpfront/auth.go</files>
  <behavior>
    - Every Task 1 lock passes unchanged, including the option-A hidden-visibility arms.
    - With the real-shape selection zai=individual-coding-plan, bigmodel=start-plan, BuildAccount receives both rules and the gate decides each independently; a family whose selected kind matches a visible rule pushes only that rule.
    - A rule with an absent/empty access.mode never matches a selected kind, so it is omitted rather than entitled.
    - The apiKey veto, the store-token requirement, the structural log codes and the boolean-only degradation are unchanged; a missing or undecodable setting.json still answers false.
    - All pre-existing hostsvc, interact, acpfront and fake/golden suites stay green with unchanged bytes.
  </behavior>
  <action>
    In internal/hostsvc/snapshot.go, add a `Mode string` field to builtinProviderRule carrying `config.access.mode` verbatim, documented as the connection mode the desktop's selection must equal; it is data, never validated, and an absent mode decodes as empty and never matches a selected kind. Decode it in digBuiltinRule's existing access struct with the `json:"mode"` tag and return it alongside Type and AccountType (same nolint-annotated capture-verbatim block). Change BuildAccount's parameter to `entitled func(family, mode string) bool` and call it with `rule.Family, rule.Mode`. Keep the existing order exactly: access-type filter, then the hidden-visibility skip (option A, unchanged), then the per-rule gate. Update the comments: BuildAccount now describes one entry per VISIBLE zhipu-account rule whose access.mode is the family's selected connection, and the gate comment states the decision is keyed by (family, mode) and may only omit rules, never edit entries; decodeBuiltinTable records that visibility and access.mode are both captured verbatim as data.

    In internal/interact/familysel.go, delete the codingPlanSelectionKind constant (keep apiKeyFamilyMode) and change the method to `Entitled(ctx context.Context, family, mode string) bool`. The selection arm becomes: dig the family's kind exactly as today, then require `kind == mode` (the desktop's selected connection kind IS this rule's mode) — rename the modes-map local to familyMode so it cannot shadow the mode parameter. Keep the `modelProviderFamilyModes[family] != "apiKey"` strict-inequality arm (absent passes) and the `oauth:<family>:access_token` presence arm unchanged. Keep the read-only posture, the three structural log codes and the boolean-only log fields (family, kind_ok, mode_ok, token_present, entitled) — `kind_ok` now means exact kind==mode, and no mode or kind value is ever logged. Rewrite the doc comment: a family is entitled for a rule when the user's desktop connection selection IS that rule's mode; individual-coding-plan, team-coding-plan and start-plan are selectable kinds, while off-peak/highspeed never appear as kinds and therefore never match.

    In internal/acpfront/auth.go, change AuthConfig.Entitle to `func(ctx context.Context, family, mode string) bool` and update its comment to the per-rule decision (may only omit, never edit). Change the authenticateCodingPlan BuildAccount closure to pass ctx plus both family and mode. Extend the wantModelsFromSnapshot comment: a provider is absent from the snapshot when its rule is hidden by visibility OR when its mode is not the family's selected connection — both omission sources mean its models can never enter the want list or the ProbeWantModel target, and the verdict-side mirror still describes exactly what was pushed. Do not edit cmd/zcode-acp/main.go or test/realdesktop/codingplan_test.go: the `machinery.gate.Entitled` method value adapts to the widened seam, and both must be proven by build/vet rather than assumed.

    Run the focused packages, the fake/golden replays, the realdesktop-tagged vet, the full race suite and the lint baseline gate; commit `feat(261123-vyl): push only the selected connection mode per account family`.
  </action>
  <verify>
    <automated>go build ./... && go vet ./... && go vet -tags realdesktop ./test/realdesktop/ && go test -race -count=1 ./internal/interact/ ./internal/hostsvc/ && go test -race -count=1 ./internal/acpfront/ ./internal/zcode/fake/ && go test -race -count=1 ./... && { out=$(golangci-lint run 2>&1 || true); n=$(printf '%s\n' "$out" | grep -cE '^[a-zA-Z0-9_./-]+\.go:[0-9]+:[0-9]+:' || true); [ "$n" -le 10 ] || { printf '%s\n' "$out"; exit 1; }; if printf '%s\n' "$out" | grep -qE '^(internal/interact/familysel(_test)?\.go|internal/hostsvc/snapshot(_account|_builtinfile)?(_test)?\.go|internal/acpfront/(auth|wantmodels_internal)(_test)?\.go|internal/zcode/fake/golden_test\.go):'; then printf '%s\n' "$out"; exit 1; fi; echo "lint: only the 10 pre-existing unrelated findings"; }</automated>
  </verify>
  <done>Every Task 1 lock passes; the focused hostsvc/interact/acpfront/fake suites and the full `go test -race -count=1 ./...` gate are green; `go build ./...`, `go vet ./...` and `go vet -tags realdesktop ./test/realdesktop/` are green; `golangci-lint run` reports no finding in any of the nine touched files (only the 10 documented pre-existing acpfront test findings remain); `git diff` touches no file outside the three declared production paths for this task and preserves the hidden-visibility skip.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| desktop setting.json -> wrapper gate | The desktop's `providerFamilyConnectionSelections` / `modelProviderFamilyModes` decide which (family, mode) pair the wrapper entitles. Read-only; the file is untrusted input to the wrapper. |
| desktop builtin table -> wrapper overlay | The bundled `zcode-builtin.json` supplies each rule's `providerId`, `builtinModelIds`, `visibility` and now `access.mode`; the mode is the value the selection is compared against. |
| wrapper snapshot -> desktop registry | The pushed overlay marks providers `entitled:true, current:true`, which materializes them in the desktop model catalog. A mis-keyed entitlement surfaces plans the user did not select. |
| gate -> logs | Degradation paths log structural class codes and presence booleans only; mode/kind/token values must never appear. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-261123-03 | Tampering | `digBuiltinRule` / `BuildAccount` (untrusted `config.access.mode`) | low | mitigate | Mode is captured verbatim and never used to construct wire data — only compared in the gate. A forged or absent mode can only OMIT a rule (empty mode never equals a non-empty selected kind), and a table left with zero entitled rules still answers the existing coded `errNoRulesToEntitle` error with a nil Snapshot, never an empty overlay. |
| T-261123-04 | Information disclosure | `EntitlementGate.Entitled` log line | low | mitigate | The new `kind == mode` comparison logs only `kind_ok`, `mode_ok`, `token_present` and `entitled` booleans plus the family; the mode/kind strings and the store token are never logged, and the existing setting-missing/unreadable/undecodable codes are unchanged. |
| T-261123-05 | Denial of service (availability) | per-mode entitlement | medium | mitigate | A missing selection entry, a missing token, or an unreadable setting.json answers false rather than a fabricated true, so the existing coded authenticate failures fire instead of a degraded push; the focused suites prove each degradation arm and the full `go test -race -count=1 ./...` run proves no unrelated path regressed. |
| T-261123-SC | Tampering | npm/pip/cargo installs | high | accept | No dependency, install step or module change in this task; the tidy pin gate is unaffected. |
</threat_model>

<verification>
- `go build ./...`, `go vet ./...`, `go vet -tags realdesktop ./test/realdesktop/` are green (the realdesktop-tagged vet proves the `machinery.gate.Entitled` method value adapts without an edit).
- `go test -race -count=1 ./internal/interact/ ./internal/hostsvc/ ./internal/acpfront/ ./internal/zcode/fake/` is green, then `go test -race -count=1 ./...` is the full gate.
- `golangci-lint run` adds zero findings in the nine touched files; exactly the 10 documented pre-existing acpfront test findings remain.
- Manual diff inspection: only the nine declared files change; the option-A hidden-visibility skip and its tests are intact; `mergeCatalogEntries` and the configOptions translation are untouched.
- The fixture/golden corpus is unchanged apart from the test-local fixtures named in Task 1; no scenario or golden file is edited.
</verification>

<success_criteria>
- With the live desktop selection `{zai: individual-coding-plan, bigmodel: start-plan}`, the pushed overlay contains exactly `account:zai-individual-coding-plan` and `account:bigmodel-start-plan` — every other visible mode of both families is absent from providers AND states.
- `EntitlementGate.Entitled` decides per (family, mode): true only when the desktop's selected connection kind equals the rule's mode exactly; false for team/start rules under an individual selection, and for any rule whose mode is absent.
- The read-back want list contains only the selected-mode rules' provider/model pairs, so the registry watcher and ProbeWantModel can never target a non-selected-mode or hidden provider.
- The apiKey veto, store-token requirement, structural log codes and boolean-only degradation are unchanged, and the option-A hidden-visibility behavior remains test-locked.
- No wrapper-side catalog filtering, entry renaming, or new coded error class was introduced; no dependency changed.
</success_criteria>

<output>
Create `.planning/quick/261123-vyl-option-b-push-only-the-selected-connecti/261123-vyl-SUMMARY.md` when done.
</output>



