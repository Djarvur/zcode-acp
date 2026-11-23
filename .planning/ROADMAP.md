# Roadmap: zcode-acp

## Overview

zcode-acp turns a Go stdio binary into an honest ACP v1 server in front of the ZCode
Desktop app-server, so any ACP client can run full turns — streamed text and thoughts,
tool calls, permission decisions, cancellation — on a real z.ai account via api-key or
the official OAuth coding-plan path. Verified wire truth from source and experiments
comes first; everything is CI-testable without the desktop bundle through a scriptable
fake app-server.

v0.20.0 shipped the v1 bar (CI conformance, a clean-VM scenario, manual E2E in Orca,
English docs). v0.21.0 adds the card's slash-command autocomplete: the wrapper serves
the `available_commands_update` catalog (ZCode skills + `/model`) the Orca composer
expects.

## Milestones

- ✅ **v0.20.0 — v1: Honest ACP Server for ZCode Desktop** — Phases 1-7 (shipped 2026-11-23)
- 🚧 **v0.21.0 — Card Command & Skill Autocomplete** — Phase 8 (in progress)

## Phases

<details>
<summary>✅ v0.20.0 (Phases 1-7) — SHIPPED 2026-11-23</summary>

- [x] Phase 1: Research & Protocol Documentation (6/6 plans) — completed 2026-11-19
- [x] Phase 2: Scaffold & CI Bootstrap (5/5 plans) — completed 2026-11-19
- [x] Phase 3: ZCode Back Leg — Codec, Fake App-Server, Supervisor (9/9 plans) — completed 2026-11-20
- [x] Phase 4: ACP Front — Minimal Chat Loop (6/6 plans) — completed 2026-11-20
- [x] Phase 5: Interactions & Full Chat Cycle (api-key) (15/15 plans) — completed 2026-11-22
- [x] Phase 6: OAuth Coding-Plan Path & Model Picker (8/8 plans) — completed 2026-11-22
- [x] Phase 7: Hardening, VM & Manual E2E, Documentation (9/9 plans) — completed 2026-11-23

Full phase details, plans and summaries: `.planning/milestones/v0.20.0-phases/`;
the milestone roadmap and requirements: `.planning/milestones/v0.20.0-ROADMAP.md`
and `.planning/milestones/v0.20.0-REQUIREMENTS.md`.

</details>

- [ ] **Phase 8: Slash Catalog — available_commands_update** — the wrapper emits the
      `available_commands_update` catalog on `session/new` (ZCode skills from
      SKILL.md frontmatter + the `/model` entry), within the client's 9 s discovery
      scan budget, chat and discovery sessions alike; verified in the Orca composer

## Post-v1 Advisories (carried, non-blocking)

- 07-REVIEW-DISPOSITION.md: IN-a..IN-h (vm-e2e nuances, doc polish, exit-code edges) + the delta-review IN-02/IN-03
- STATE.md Deferred Items: the two acknowledged phase-6 lint-debt entries (funlen/goconst/err113/tagliatelle with wire-shape reasons)
- Orca card wish: a slash-command catalog in the card's autocomplete → **BEING DELIVERED by v0.21.0 Phase 8**
