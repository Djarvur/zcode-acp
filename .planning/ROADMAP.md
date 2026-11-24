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
- 🚧 **v0.23.0 — Mount-First AppImage Backend** — Phase 10 (in progress)

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

- [ ] **Phase 10: Mount-First AppImage Backend** — the wrapper runs the ZCode
  app-server from a packed `.AppImage` through a managed `--appimage-mount` helper
  (no extraction, no ~850 MB on-disk duplicate) and discovery ranks the newest usable
  backend across layouts

## Phase Details

### Phase 10: Mount-First AppImage Backend

**Goal**: The wrapper runs the ZCode app-server directly from a packed `.AppImage`
through a managed `--appimage-mount` helper — no extraction, no ~850 MB on-disk
duplicate — and discovery always selects the newest usable backend across layouts, so
a stale unpacked `squashfs-root` can never pin an older version.
**Depends on**: Phase 3 (supervisor, discovery, and the spawn contract — extended in place)
**Requirements**: MNT-01, MNT-02, MNT-03, MNT-04, MNT-05, MNT-06, DISC-01, DISC-02, DISC-03, DISC-04, VER-01, VER-02, VER-03
**Success Criteria** (what must be TRUE):
  1. A run backed by a mounted packed AppImage completes a session (session/create plus
     prompt) with the app-server rooted at the `--appimage-mount` path; no
     `squashfs-root`/`zcode-<ver>` directory appears and the AppImage is unmodified
  2. `--app-server <path.AppImage>` and `ZCODE_ACP_APP_SERVER` accept a packed AppImage:
     the wrapper mounts it, validates `<mount>/zcode` and
     `<mount>/resources/glm/zcode.cjs`, and spawns the app-server from that mount
  3. The mount helper is a supervised child torn down after the app-server on every exit
     path — normal, error, signal, client disconnect — with zero orphan processes and
     zero stale mounts; a mount loss while the app-server lives ends the session with an
     explicit cause instead of respawning against a vanished root
  4. Mount failures — missing FUSE, helper exit before the mount line, readiness timeout,
     unspawnable mount tree — fail loud with a non-zero exit and an actionable message
     naming the AppImage, the failing step, and the remediation; there is never a silent
     extraction fallback
  5. Discovery selects the newest usable backend by version across layouts: a stale
     unpacked `squashfs-root` loses to a newer packed AppImage, a dangling symlink or
     invalid tree is recorded as a miss and iteration continues to the next candidate,
     and selection is deterministic with documented ordering and tie-break
  6. Diagnostics report the selected backend's layout, version, and resolved path
     (version from the AppImage filename or the unpacked dir name); CI-safe scripted
     tests drive the mount lifecycle without FUSE, the real-machine test stays
     env-gated, and the install-discovery doc, README, and provenance note match behavior

**Plans**: 5/6 plans executed

Plans:
- [x] 10-01-PLAN.md — tracer: packed AppImage to managed mount to app-server to ACP session
- [x] 10-02-PLAN.md — mount failure classification, mid-session loss, OS-signal teardown
- [x] 10-03-PLAN.md — version-ranked AppImage discovery with per-candidate diagnostics
- [x] 10-04-PLAN.md — override/env routing, PAT version seam, selection diagnostics
- [x] 10-05-PLAN.md — guarded real-machine acceptance and mount-aware manual tier
- [ ] 10-06-PLAN.md — install-discovery, README, provenance, and vm-e2e coherence

## Post-v1 Advisories (carried, non-blocking)

- 07-REVIEW-DISPOSITION.md: IN-a..IN-h + the delta-review IN-02/IN-03
- STATE.md Deferred Items: the two acknowledged phase-6 lint-debt entries
- Delivered in v0.21.0: the slash-command catalog in the card's autocomplete
- Delivered in v0.22.0: the ask-the-user bridge (choice dialogs render and feed the
  workflow); the full `/model` catalog (19 entries, provider labels, stable across switches)
