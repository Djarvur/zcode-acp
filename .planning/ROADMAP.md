# Roadmap: zcode-acp

## Overview

zcode-acp turns a Go stdio binary into an honest ACP v1 server in front of the ZCode
Desktop app-server, so any ACP client can run full turns — streamed text and thoughts,
tool calls, permission decisions, cancellation — on a real z.ai account via api-key or
the official OAuth coding-plan path. Verified wire truth from source and experiments
comes first; everything is CI-testable without the desktop bundle through a scriptable
fake app-server.

## Milestones

- ✅ **v0.20.0 — v1: Honest ACP Server for ZCode Desktop** — Phases 1-7 (shipped 2026-11-23)
- ✅ **v0.21.0 — Card Command & Skill Autocomplete** — Phase 8 (shipped 2026-11-23)
- ✅ **v0.22.0 — Ask-the-User Bridge** — Phase 9 (shipped 2026-11-23)

## Phases

<details>
<summary>✅ v0.20.0 (Phases 1-7) — SHIPPED 2026-11-23</summary>

- [x] Phase 1: Research & Protocol Documentation (6/6 plans)
- [x] Phase 2: Scaffold & CI Bootstrap (5/5 plans)
- [x] Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor (9/9 plans)
- [x] Phase 4: ACP Front — Minimal Chat Loop (6/6 plans)
- [x] Phase 5: Interactions & Full Chat Cycle (api-key) (15/15 plans)
- [x] Phase 6: OAuth Coding-Plan Path & Model Picker (8/8 plans)
- [x] Phase 7: Hardening, VM & Manual E2E, Documentation (9/9 plans)

Full details: `.planning/milestones/v0.20.0-phases/`, `v0.20.0-ROADMAP.md`, `v0.20.0-REQUIREMENTS.md`.

</details>

<details>
<summary>✅ v0.21.0 (Phase 8) — SHIPPED 2026-11-23</summary>

- [x] Phase 8: Slash Catalog — available_commands_update (1/1 plan) — the card's
  composer autocomplete lists the ZCode commands/skills and `/model`

Full details: `.planning/milestones/v0.21.0-phases/`, `v0.21.0-ROADMAP.md`, `v0.21.0-REQUIREMENTS.md`.

</details>

<details>
<summary>✅ v0.22.0 (Phase 9) — SHIPPED 2026-11-23</summary>

- [x] Phase 9: Ask-the-User Bridge (1/1 plan) — `interaction/requestUserInput` renders
  as ONE client form elicitation; the answer maps back into the dialect's
  `{action, content.answers}` merge contract (VM GUI + real-machine acceptance)

Full details: `.planning/milestones/v0.22.0-phases/`, `v0.22.0-ROADMAP.md`, `v0.22.0-REQUIREMENTS.md`.

</details>

## Post-v1 Advisories (carried, non-blocking)

- 07-REVIEW-DISPOSITION.md: IN-a..IN-h + the delta-review IN-02/IN-03
- STATE.md Deferred Items: the two acknowledged phase-6 lint-debt entries
- Delivered in v0.21.0: the slash-command catalog in the card's autocomplete
- Delivered in v0.22.0: the ask-the-user bridge (choice dialogs render and feed the
  workflow); the full `/model` catalog (19 entries, provider labels, stable across switches)
