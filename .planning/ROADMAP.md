# Roadmap: zcode-acp

## Overview

zcode-acp turns a Go stdio binary into an honest ACP v1 server in front of the ZCode
Desktop app-server, so any ACP client can run full turns — streamed text and thoughts,
tool calls, permission decisions, cancellation — on a real z.ai account via api-key or
the official OAuth coding-plan path. v0.24.0 ports that existing v0.23.0 behavior to
universal macOS (arm64/amd64) without adding ACP capabilities: Darwin build/test gates,
`.app` discovery and launch, Keychain-backed OAuth, safe automated acceptance, English
setup documentation, and a final human-gated live Orca run.

## Milestones

- ✅ **v0.20.0 — v1: Honest ACP Server for ZCode Desktop** — Phases 1-7 (shipped 2026-11-23)
- ✅ **v0.21.0 — Card Command & Skill Autocomplete** — Phase 8 (shipped 2026-11-23)
- ✅ **v0.22.0 — Ask-the-User Bridge** — Phase 9 (shipped 2026-11-23)
- ✅ **v0.23.0 — Mount-First AppImage Backend** — Phase 10 (shipped 2026-11-24)
- **v0.24.0 — macOS Support** — Phases 11-16 (planned)

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

<details>
<summary>✅ v0.23.0 (Phase 10) — SHIPPED 2026-11-24</summary>

- [x] Phase 10: Mount-First AppImage Backend (6/6 plans) — the wrapper runs the
  app-server from a packed `.AppImage` through a managed `--appimage-mount` helper
  (no extraction, no ~850 MB on-disk duplicate); discovery is one pure-read stage
  ranking the newest usable backend by version; real-machine acceptance
  (mount → app-server → `session/create`, zero leftovers) passed on ZCode 4.0.0.
  Delivered alongside: OMP print mode for Orca text tasks (`261124-154`) and
  `-32003 session_busy` for concurrent prompts (`261124-1ou`).

Full details: `.planning/milestones/v0.23.0-phases/`, `v0.23.0-ROADMAP.md`, `v0.23.0-REQUIREMENTS.md`.

</details>

### v0.24.0 — macOS Support (Planned)

- [ ] **Phase 11: Darwin Build, Test & Release Gates** - Both macOS architectures compile and pass portable tests, CI, and checksummed release-artifact checks.
- [ ] **Phase 12: macOS Bundle Discovery & Overrides** - Standard `.app` locations and override forms resolve without entering the Linux AppImage/FUSE path.
- [ ] **Phase 13: Desktop Launch & Runtime Environment** - The selected bundle launches the correct app-server with macOS provider paths and a usable GUI-launched tool PATH.
- [ ] **Phase 14: macOS Authentication** - Existing api-key and OAuth paths work, with Keychain-backed credential decryption and safe coded failures.
- [ ] **Phase 15: Automated Orca Acceptance & Documentation** - Fake-backend Orca coverage and Linux regressions stay zero-spend, and the English macOS guide is complete.
- [ ] **Phase 16: Live Orca Acceptance** - A human proves the existing ACP flow in Orca on the current Mac and records scrubbed evidence.

## Post-v1 Advisories (carried, non-blocking)

- 07-REVIEW-DISPOSITION.md: IN-a..IN-h + the delta-review IN-02/IN-03
- STATE.md Deferred Items: the two acknowledged phase-6 lint-debt entries
- v0.23.0 archive `10-REVIEW-DISPOSITION.md`: IN-01..IN-07 open advisories from the Phase 10 deep review
- Delivered in v0.21.0: the slash-command catalog in the card's autocomplete
- Delivered in v0.22.0: the ask-the-user bridge (choice dialogs render and feed the
  workflow); the full `/model` catalog (19 entries, provider labels, stable across switches)
- Delivered in v0.23.0: mount-first AppImage backend (read-only mount, no extraction,
  version-ranked discovery, loud FUSE failures); `omp --print --mode text` one-shot
  text tasks; `-32003 session_busy` so Orca's retry queue can engage

## Phase Details

**Milestone:** v0.24.0 — macOS Support. This is a platform port of the shipped v0.23.0
feature set. Orca's existing `omp` parsing/stdout contract is retained, not redesigned.

### Phase 11: Darwin Build, Test & Release Gates

**Goal**: Developers can build, test, and package the existing bridge for both macOS architectures without Linux-only tool assumptions.
**Depends on**: Phase 10 (shipped v0.23.0)
**Requirements**: MBLD-01, MBLD-02, MBLD-03, MTEST-01
**Success Criteria** (what must be TRUE):
  1. Both specified `CGO_ENABLED=0 GOOS=darwin` cross-builds complete successfully for arm64 and amd64.
  2. The default test suite passes on macOS arm64 without external `setsid`, GNU-only utilities, or system Bash 3.2, using a Go re-exec helper where process setup is needed.
  3. CI reports passing build and test gates for Linux and for macOS arm64/amd64 with explicit required tool versions.
  4. A release snapshot produces checksummed macOS arm64 and amd64 binary artifacts whose checksums verify.

**Plans**: 5/5 plans executed

Plans:
**Wave 1**
- [x] 11-01-PLAN.md — Normalize the Statfs width and prove both CGO-free Darwin builds.
- [x] 11-02-PLAN.md — Replace both detached shell fixtures with Go re-exec helpers.

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 11-03-PLAN.md — Port docs/secret gates to Go commands and enforce all four mise-resolved tool versions.

**Wave 3** *(blocked on Wave 2 completion)*
- [x] 11-04-PLAN.md — Add Linux/macOS native CI gates and both Darwin cross-builds.

**Wave 4** *(blocked on Wave 3 completion)*
- [x] 11-05-PLAN.md — Produce and verify checksummed thin and universal macOS archives.

### Phase 12: macOS Bundle Discovery & Overrides

**Goal**: Users can point the bridge at a standard or explicit ZCode Desktop `.app` installation and receive a validated, versioned backend selection.
**Depends on**: Phase 11
**Requirements**: MDISC-01, MDISC-02
**Success Criteria** (what must be TRUE):
  1. Discovery finds valid `ZCode.app` bundles in `/Applications` and `$HOME/Applications` and reports the version read from `Info.plist`.
  2. Invalid or incomplete bundles are rejected with actionable diagnostics rather than selected for launch.
  3. `--app-server` accepts a `.app`, its `Contents` directory, its `Contents/MacOS/<executable>`, and the existing Linux layouts without changing Linux override behavior.
  4. A macOS `.app` selection never invokes AppImage mounting or requires FUSE.

**Plans**: TBD

### Phase 13: Desktop Launch & Runtime Environment

**Goal**: A selected macOS bundle runs the desktop app-server with the correct host inputs, provider cache, and user tool environment.
**Depends on**: Phase 12
**Requirements**: MDISC-03, MDISC-04, MDISC-05
**Success Criteria** (what must be TRUE):
  1. A discovered `.app` launches its `Contents/MacOS/<executable>` with `Contents/Resources/glm/zcode.cjs`, `ELECTRON_RUN_AS_NODE=1`, and the bundled provider table.
  2. Runtime-provider discovery selects the macOS OS/architecture cache and the backend receives a builtin revision derived from the same provider path used at launch.
  3. A backend launched by GUI-started Orca can find the user's toolchain even when Orca inherits a launchd-minimal PATH.
  4. PATH completion is bounded and non-interactive, unions login-shell and caller paths, and preserves secret/proxy scrubbing in the child environment.

**Plans**: TBD

### Phase 14: macOS Authentication

**Goal**: Users can authenticate through either existing path on macOS, with OAuth secrets handled by Keychain and failures kept actionable and private.
**Depends on**: Phase 13
**Requirements**: MAUTH-01, MAUTH-02, MAUTH-03
**Success Criteria** (what must be TRUE):
  1. The existing api-key authentication path completes on macOS without invoking Keychain.
  2. OAuth coding-plan authentication decrypts `enc:v2:os` values through macOS Keychain using the existing service/account identity and a bounded `/usr/bin/security` invocation.
  3. Missing or locked Keychain items return actionable coded errors that distinguish the failure without exposing key material.
  4. Authentication runs write no key material to logs or stdout.

**Plans**: TBD

### Phase 15: Automated Orca Acceptance & Documentation

**Goal**: Users can validate the macOS port safely in CI and follow an English guide to build, configure, and launch it with Orca.
**Depends on**: Phase 14
**Requirements**: MTEST-02, MORCA-01, MDOC-01
**Success Criteria** (what must be TRUE):
  1. Automated fixtures cover `.app` discovery/override, exact app-server argv, provider-cache selection, and the Keychain command contract without touching a real desktop or Keychain in CI.
  2. `zcode-acp omp --print --mode text @file` completes against a fake macOS `.app` backend with the existing stdout contract and correct exit status.
  3. Linux AppImage/FUSE regression coverage remains green while macOS `.app` runs bypass that path, and all automated acceptance remains zero-spend with no real credentials.
  4. The English README documents macOS builds, ZCode discovery, Keychain prerequisites, Orca launch, and explicit platform limitations.

**Plans**: TBD

### Phase 16: Live Orca Acceptance

**Goal**: A human confirms the shipped ACP experience works in a real Orca card on the current macOS machine with an available real account.
**Depends on**: Phase 15
**Requirements**: MORCA-02
**Success Criteria** (what must be TRUE):
  1. A human-run Orca card completes the existing `initialize` → `session/new` → prompt flow against real ZCode, including streamed output, tool-calls, and permission requests.
  2. The run uses an available real account through an existing authentication path and introduces no new ACP capability.
  3. Scrubbed evidence records the macOS environment, ZCode version, launch command, authentication path, observed flow, and outcome.

**Plans**: TBD
**Acceptance gate**: Human-run only on the current Mac; this phase may use real credentials and account quota. Automated phases must remain zero-spend and must not touch the real desktop, credentials, or Keychain in CI.

## Progress

**Execution Order:** Phases execute in numeric order: 11 → 12 → 13 → 14 → 15 → 16.

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 11. Darwin Build, Test & Release Gates | 5/5 | In Progress | - |
| 12. macOS Bundle Discovery & Overrides | 0/TBD | Not started | - |
| 13. Desktop Launch & Runtime Environment | 0/TBD | Not started | - |
| 14. macOS Authentication | 0/TBD | Not started | - |
| 15. Automated Orca Acceptance & Documentation | 0/TBD | Not started | - |
| 16. Live Orca Acceptance | 0/TBD | Not started | - |
