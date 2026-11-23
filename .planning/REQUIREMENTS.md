# Requirements — v0.21.0 Card Command & Skill Autocomplete

> Derived from the user's request ("Я ХОЧУ АВТОДОПОЛНЕНИЕ ДЛЯ СКИЛОВ И КОМАНД") and the
> wire-truth read of the installed Orca build (docs/protocol/acp-client-dialect.md, the
> "slash-command catalog surface" section, 2026-11-23).

## v0.21.0 Requirements

- [x] **CAT-01**: On `session/new` the wrapper emits ONE `session/update` notification with
      `update.sessionUpdate = "available_commands_update"` carrying
      `update.availableCommands` — within the client's 9 s scan budget, for chat and
      disposable discovery sessions alike
- [x] **CAT-02**: The catalog lists the ZCode skills the wrapper can resolve — user skills
      (`~/.zcode/skills/**/SKILL.md`) and installed plugin skills
      (`~/.zcode/cli/plugins/cache/**/skills/**/SKILL.md`), name/description from the
      SKILL.md frontmatter, deterministic order, missing/broken entries skipped without
      failing the catalog
- [x] **CAT-03**: The catalog includes the wrapper's own `/model` entry (name `model`,
      description naming the model switch, `input.hint` for the selector argument) — the
      omp dialect's `modelCommand` contract (`resolvesSelectorsFrom: 18.1.9`) expects the
      selector prompt form `/model <selector>`
- [x] **CAT-04**: The catalog survives the disposal path: the discovery session's
      `session/delete` is answered by the existing SESS-02 teardown; the wrapper does NOT
      advertise `sessionCapabilities.delete` (the client's disposable-storage arm suffices,
      chat-close semantics stay untouched)
- [x] **CAT-05**: Field-verified: Orca's card composer autocomplete lists the skills and
      `/model` for the wrapper card (VM GUI pass by the agent + the user's acceptance)

## Traceability

| ID | Phase | Status |
|----|-------|--------|
| CAT-01 | Phase 8 | Verified (2026-11-23) |
| CAT-02 | Phase 8 | Verified (2026-11-23) |
| CAT-03 | Phase 8 | Verified (2026-11-23) |
| CAT-04 | Phase 8 | Verified (2026-11-23) |
| CAT-05 | Phase 8 | Verified (2026-11-23, VM GUI pass; the user's real-machine look optional) |
