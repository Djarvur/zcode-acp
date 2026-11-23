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
- 🚧 **v0.22.0 — Ask-the-User Bridge** — Phase 9 (in progress)

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

- [x] **Phase 9: Ask-the-User Bridge** — `interaction/requestUserInput` bridges to a (completed 2026-11-23)
  client-rendered form elicitation (the AskUserQuestion dialogs the card never showed);
  the answer maps back into the dialect's `answers` merge contract

## Phase Details

### Phase 9: Ask-the-User Bridge
**Goal**: `interaction/requestUserInput` (the AskUserQuestion shape) bridges to ONE
client-rendered ACP form elicitation, and the answer maps back into the dialect's
`{action, content: {answers}}` merge contract — keyed by question text with option
labels as values (multi-select joined ", "); decline/cancel verbatim; an emission
failure declines with the reason and never fails the turn.
**Depends on**: Nothing (first phase of v0.22.0)
**Requirements**: UIB-01, UIB-02, UIB-03, UIB-04, UIB-05
**Success Criteria** (what must be TRUE):
  1. The AskUserQuestion dialog renders in Orca as a form elicitation and the chosen labels ride back as `content.answers`
  2. Multi-select joins labels ", "; decline/cancel map verbatim
  3. Without a bound asker the honest decline stays; an emission failure declines with the reason and never fails the turn
  4. Field-verified on the real machine: the choice dialog renders and answering it feeds the workflow
**Plans**: 1/1 plans complete

Plans:
- [x] 09-01: The ask-the-user bridge — requestUserInput ↔ form elicitation

## Post-v1 Advisories (carried, non-blocking)

- 07-REVIEW-DISPOSITION.md: IN-a..IN-h + the delta-review IN-02/IN-03
- STATE.md Deferred Items: the two acknowledged phase-6 lint-debt entries
- Delivered in v0.21.0: the slash-command catalog in the card's autocomplete
