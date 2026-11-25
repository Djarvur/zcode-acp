# Quick Task 261125-0vv: Prevent Stale Reasoning Levels — Research

**Researched:** 2026-11-25
**Domain:** Go ACP adapter; ZCode `session/setModel`; Orca 1.4.271 config options
**Confidence:** HIGH for current code and incident; MEDIUM for client-version drift

## User Constraints (from task and CLAUDE.md)

- Preserve the previous recoverable/fatal classifier contract.
- Keep this concise and actionable; do not produce a full domain survey.
- Go; desktop ZCode app-server is the only backend.
- Research ZCode and Orca from source before experiments.
- English documentation; strict goswitch-style lint/protected-branch workflow.
- Work stays inside the invoked GSD quick-task workflow.

## Summary

The session-preservation fix is present at `7b59c64`, but the preserved snapshot is exactly what makes the next retry dangerous: `setConfigMutation` attaches `confirmedLevel(s)` to every model switch. The field journal shows Orca requesting `enabled`/`disabled` while the wrapper repeatedly sent the old `max`, and a return to DeepSeek succeeded before Orca's following thought write failed locally on `enabled`. [VERIFIED: Orca `agent-session-domain.db`, session `f27f3310-0fbe-4b4f-bd1d-4af990d1d1c0`; git HEAD `7b59c64`]

**Primary recommendation:** resolve reasoning against the target model's own `reasoning.levels`/`defaultLevel`; preserve the current level only when the target lists it. Normalize Orca's `enabled` alias through that resolver. Map `disabled` only to an advertised `off`; if the target has no `off`, reject before the wire rather than silently substituting `low` or omitting a required level. Keep reasoning as an ACP `select`; do not add a boolean thought option.

## Standard Stack

Use the existing Go 1.27.x codebase and `github.com/coder/acp-go-sdk` v0.20.0; add no packages. [VERIFIED: `.claude/CLAUDE.md:46-47`]

## Package Legitimacy Audit

Not applicable: this task installs no external packages.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|---|---|---|---|
| Normalize client aliases and choose target level | API/Backend (`acpfront`) | ACP adapter | Owns untrusted value validation and backend request construction. |
| Advertise model/thought choices | Frontend server (`acpport`) | Orca client | ACP option kind and values are the compatibility boundary. |
| Enforce supported reasoning levels | ZCode app-server | — | Backend registry is authoritative and rejects missing/unsupported levels. |
| Apply model before effort | Orca client | — | Orca explicitly writes `model`, then `thinking`, then `effort`. |

## Verified Semantics

1. **Current bug.** The model arm does `if level := confirmedLevel(s); level != "" { model["options"] = map[string]any{wireKeyReasoningLevel: level} }`; the thought arm sends the requested string unchanged. [VERIFIED: `internal/acpfront/front.go:928-995`, especially `956-958` and `981-988`]
2. **Snapshot shape.** `ModelEntry` carries `ReasoningLevels []ReasoningLevel` and `DefaultLevel string`; parsing reads each available entry's `reasoning.levels` and `reasoning.defaultLevel`. [VERIFIED: `internal/acpfront/configoptions.go:32-50`, `141-144`]
3. **Rendering.** Thought options use the bound model's levels and render only when `s.ThoughtLevel.Enabled || s.ThoughtLevel.Current == ""` is false; the exact guard is `if !s.ThoughtLevel.Enabled || s.ThoughtLevel.Current == "" { return nil }`. [VERIFIED: `internal/acpfront/configoptions.go:332-375`, especially `338-340`]
4. **Omission is not a reset.** ZCode's schema makes `options.reasoningLevel` optional, but the runtime strictly validates a full selection; `EXP-TURN-01` records `-32603` with `"Reasoning level is required for zai-api/GLM-5.3-Flash"`. [VERIFIED: `experiments/captures/EXP-TURN-01.jsonl:32`; `docs/protocol/app-server-dialect.md:71`] ZCode source routes structured `session/setModel` through `resolveRegistryOwnedModelSelection` without `allowMissingReasoning`. [VERIFIED: `/home/nil/DiskD/W/Djarvur/zcode-src/apps/zcode-cli/packages/bootstrap/src/app/session-facade.ts:647-658`; `.../zcode-protocol/server-operations.ts:2807-2815`]
5. **Orca values are strings, not backend levels.** The preserved record ends with `"options":{"model":"OpenCodeGo/hungerg-free","effort":"disabled"}`. Attempts include `effort:"enabled"` and `effort:"disabled"`, but backend errors still name `"max"`; at 00:17:52 Orca returned to DeepSeek and then sent `thought_level=enabled`, rejected locally with allowed values `(low, high, max)`. [VERIFIED: Orca `agent-session-domain.db/journal_submission_attempts`, same session, ordinals 1–9]
6. **Orca ordering/retry behavior.** Orca invalidates effort when model changes, then writes keys in the order `interactionMode`, `model`, `thinking`, `effort`, `fastMode`; its ACP write type carries `value: string`. [CITED: https://github.com/stablyai/orca/blob/v1.4.271/src/main/acp/options/acp-structured-options.ts#L94-L105 ; https://github.com/stablyai/orca/blob/v1.4.271/src/main/acp/turns/acp-message-execution-options.ts#L34-L46 ; https://github.com/stablyai/orca/blob/v1.4.271/src/main/acp/options/acp-session-option-write.ts#L12-L28]
7. **Boolean ACP option is not appropriate here.** ACP supports boolean options, but they are on/off controls and capability-gated in v1; multi-state reasoning remains a select. [CITED: https://agentclientprotocol.com/protocol/v1/session-config-options.md ; https://agentclientprotocol.com/rfds/boolean-config-option.md] Orca 1.4.271's `acpEffortOption` accepts only `category === 'thought_level' && option.type === 'select'`, so a boolean thought option would be ignored as effort. [CITED: https://github.com/stablyai/orca/blob/v1.4.271/src/main/acp/options/acp-select-choices.ts#L59-L72] The adapter currently translates boolean asks to an empty string and rejects them through membership validation. [VERIFIED: `internal/acpport/agent.go:497-540`, especially `522-527`]

## Smallest Safe Design

### 1. Shared target-level resolver

Add one pure helper in `acpfront` and call it from the model arm of `setConfigMutation`; `/model` already shares that function. [VERIFIED: `internal/acpfront/front.go:840-845`, `928-995`]

For an exact target `providerId/modelId`:

1. Find the target entry in `s.Model.Available`.
2. If the entry exists and has no reasoning levels, omit `options` (the existing plain-model behavior).
3. If the confirmed level is non-empty and appears in the target's levels, preserve it.
4. Else use the target's `defaultLevel` only if it appears in the target's levels.
5. Else use the target's first listed level. This is a supported value, not a fabricated enum. [ASSUMED: first-listed fallback is acceptable product policy when `defaultLevel` is absent; confirm if the project prefers a pre-wire refusal instead.]
6. If the target entry is missing, fail closed; do not fall back to `settings.thoughtLevel.available`, which may describe the source model.

This directly fixes the observed DeepSeek `max` → hungerg-free retry loop and still preserves valid same-family choices such as `low`.

### 2. Inbound Orca alias normalization

Keep advertised options verbatim; do **not** add `enabled` or `disabled` to `BuildConfigOptions`. Before normal membership validation, special-case only `configID == optionIDThought`:

- `enabled` → run the same resolver for the currently bound model.
- `disabled` → use `off` only when the bound model explicitly lists `off`; otherwise return the existing coded `-32602` shape naming the target's allowed values and make no backend call.
- Any other value → existing listed-value validation unchanged.

The `off` value is already a legitimate backend-advertised level in fixtures (`"available": [{"value": "off", "label": "Off"}, ...]`). [VERIFIED: `internal/acpfront/setconfigoption_test.go:262-265`] There is no faithful wrapper-only representation of “disabled” for hungerg-free's observed `low, high, max`; silently mapping it to `low` would change user intent. [VERIFIED: Orca journal ordinal 3 allowed-value error]

### 3. Preserve the classifier contract

Do not modify `isRecoverableSetModelError`, either teardown call site, ACP error mapping, or settings merge behavior. The classifier currently accepts numeric `-32602`, exact structured `ModelProtocolError`/`invalid_model_request`, and two narrow data-less reasoning messages; all other failures remain fatal. [VERIFIED: `internal/acpfront/front.go:873-912`; call sites `668-677` and `845-853`]

## Validation Architecture

| Area | Existing helper/pattern | Required coverage |
|---|---|---|
| Picker wire payload | `cfgOptionScenario`, `assertSetCfgModelFrame`, `assertSingleSetModel` in `internal/acpfront/setconfigoption_test.go:33-101`, `359-425` | Source `max`, target without `max` → target default/first; target with `max` → preserve; level-less target → omit. |
| Alias handling | Same external ACP harness | `enabled` resolves to a listed level; `disabled` sends `off` when listed; `disabled` without `off` returns `-32602` and emits zero `setModel` frames. |
| Preserved retry | `newInterleaveFront`, `openPickerSession`, `assertPickerSessionPreserved` in `pickerrefusal_internal_test.go:120-179` | Rejected cross-model switch leaves bytes unchanged; retry no longer carries stale `max`; fatal classifier tests stay green. |
| `/model` parity | `TestModelSlashCarriesConfirmedLevel` in `slashmodel_test.go:622-675` | Same target-level resolution through the shared mutation function. |
| Translator honesty | `configoptions_test.go` | No synthetic aliases; disabled/current-empty omission behavior unchanged. |

Baseline verification passed: `go test -race -count=1 ./internal/acpfront` and `go test -race -count=1 ./internal/acpport`. No new dependency is needed; package legitimacy audit is not applicable.

## Common Pitfalls

- **Using global `thoughtLevel.available` for a new target:** it can be the old model's menu; use the target entry's per-model levels.
- **Omitting stale reasoning as the fix:** omission is refused for level-strict reasoning models.
- **Mapping `disabled` to `low`:** supported value, wrong semantics; reject unless `off` is advertised.
- **Adding a boolean thought option:** Orca 1.4.271 ignores it for effort, and the current adapter advertises/accepts selects only.
- **Widening the error classifier:** unrelated to this bug and would weaken the fail-closed lifecycle contract.
- **Testing only the pure mapper:** the incident depends on preserved snapshot + retry + Orca's model-then-effort sequence; keep one composed lifecycle regression.

## Security Domain

- **ASVS V5 Input Validation:** normalize only the two known aliases, validate the result against exact target metadata, and reject unknown/unrepresentable values before backend mutation.
- **Threat:** repeated stale-level requests cause backend rejection/denial-of-service; target validation removes the loop without broadening recoverable errors.
- No authentication, session ownership, cryptography, or persistence format changes.

## Environment Availability

| Dependency | Available | Version | Fallback |
|---|---:|---:|---|
| Go | ✓ | 1.27.3 | — |
| Node | ✓ | 26.13.1 | — |
| Orca installed source | ✓ | 1.4.271 | — |
| SQLite | ✓ | 3.54.1 | — |
| golangci-lint | ✓ | 2.15.0 | — |

## Assumptions Log

| # | Claim | Risk if wrong |
|---|---|---|
| A1 | First-listed target level is an acceptable fallback when `defaultLevel` is absent. | A model switch may select a higher effort than the user expected; use pre-wire refusal instead if unacceptable. |

## Open Question

If product requires a working Off toggle on models whose backend lists no `off`, that requires a backend capability change; the wrapper cannot implement it faithfully. Recommended scope for this task: support `off` where advertised and fail closed elsewhere.
