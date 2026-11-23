# Roadmap: zcode-acp

## Overview

zcode-acp turns a Go stdio binary into an honest ACP v1 server in front of the ZCode
Desktop app-server, so any ACP client can run full turns — streamed text and thoughts,
tool calls, permission decisions, cancellation — on a real z.ai account via api-key or
the official OAuth coding-plan path. Verified wire truth from source and experiments
comes first; everything is CI-testable without the desktop bundle through a scriptable
fake app-server; the done-v1 bar (CI conformance, a clean-VM scenario, manual E2E in a
real chat-UI client, English docs with a preset-agent launch recipe) is met.

## Milestones

- ✅ **v0.20.0 — v1: Honest ACP Server for ZCode Desktop** — Phases 1-7 (shipped 2026-11-23)

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

## Post-v1 Advisories (carried, non-blocking)

- 07-REVIEW-DISPOSITION.md: IN-a..IN-h (vm-e2e nuances, doc polish, exit-code edges) + the delta-review IN-02/IN-03
- STATE.md Deferred Items: the two acknowledged phase-6 lint-debt entries (funlen/goconst/err113/tagliatelle with wire-shape reasons)
- Orca card wish: a slash-command catalog in the card's autocomplete (the `/model` intercept works regardless; the catalog surface is outside the v1 dialect)
