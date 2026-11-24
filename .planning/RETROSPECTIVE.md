# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v0.20.0 — v1: Honest ACP Server for ZCode Desktop

**Shipped:** 2026-11-23
**Phases:** 7 | **Plans:** 58 | **Tasks:** 145

### What Was Built

- An honest ACP v1 server over stdio in front of the ZCode Desktop app-server: one
  static Go binary (`zcode-acp`), `omp` subcommand for preset-agent-only hosts (Orca).
- Verified wire truth first: `docs/protocol/*` (app-server dialect, client dialect,
  install discovery) claim-first with per-claim provenance and a 3.17↔3.18.2 drift baseline.
- The back leg: stdlib-only codec, scriptable fake app-server (`cmd/fake-app-server`) with
  a byte-deterministic golden corpus (40+ scenarios), orphan-proof supervisor, exactly-once
  turn machine with a carrier-tolerant terminal-signal union.
- The front: full chat cycle (streamed text/thoughts, tool calls, permission bridge,
  cancel, resume/close), both auth paths — api-key (REAL RUN 49.43s) and the official
  OAuth coding-plan chain (keyring → entitlement → snapshot read-back → PAT mint).
- The model picker: registry-driven configOptions, `/model` slash layer with byte-exact
  refusals, `--model` launch binding.
- Hardening: conformance arc, `scripts/vm-e2e.sh` clean-VM scenario, English README +
  Orca launch recipe + docs/provenance.md; security audits per phase (both SECURED,
  threats_open 0).

### What Worked

- **Wire-truth-first discipline** (capture-verbatim fixtures, provenance labels,
  raw-dump debugging): every field gap of QA-03 was diagnosed from captured frames, not
  guessed — and the dialect docs made the fixes provable.
- **The fake app-server as executable spec**: all 50 plans' behaviors lock in CI without
  the desktop; the QA-03 fixes each shipped with a new corpus row (RED-verified).
- **Gap cycles with honest ledgers**: every field failure (attempts 1–8 of phase 6, the
  three QA-03 cycles) produced a committed root cause + fix + live verification, with
  real-account spend recorded per run.
- **The agent-runs-it-itself contract** (user delegation): the agent fired the real runs,
  built the KVM VM, reproduced the user's symptom locally with the exact production argv,
  and verified the GUI rendering in a VM before handing anything back.
- **GSD gates**: whole-tree lint + full -race suite per wave caught cross-plan breakage
  (golden fixtures, widened signatures) before it compounded.

### What Was Inefficient

- **The stale-binary verification trap** (2026-11-23): three "successful" runs used the
  pre-fix installed binary and proved nothing — `go install` + binary mtime check must
  precede every verification series. Cost: 3 wasted real turns + a false signal.
- **Phase-6's eight real-run attempts**: the read-back stall (the MCP-auth decline) and
  the revision tie→bundled falsified hypothesis cost 4 zero-spend + 4 mint-only cycles
  before attempt 8 passed. The guard-vs-real gap is inherent, but the guarded probe
  could have poked setModel from the start.
- **Whole-tree lint debt**: the wave-1 scoped-lint masking shipped 67 issues into phase 7's
  merge; the whole-tree gate now prevents recurrence, but the debt itself became two
  acknowledged deferred items.
- **Fingerprint-vs-docs churn at close-out**: every honest docs edit after the verifier
  ran restaled the report (three regenerations). Batching close-out edits BEFORE the final
  verification run would have cost one verifier pass, not three.

### Patterns Established

- Capture-verbatim corpus rows as the ONLY fixture style (no hand-invented wire shapes).
- Honest failure arms everywhere: coded refusals, never fabricated text, never a hang.
- Memory-only credentials; scrub-first capture pipeline; scan-secrets on tree and diff.
- TDD RED evidence machine-checked (`tdd-red-evidence`) for every behavior plan.
- Turn-scoped pumps with atomic test-and-enqueue guards (the WR-03 pattern).

### Key Lessons

- The desktop's terminal signal can arrive BEFORE the reply text reaches its own
  projection — completion-side recovery must retry (bounded) and correlate the entry to
  THIS turn (backend-clock stamp), never take "last assistant entry" blind.
- Client-side greps cannot prove which signal resolved a turn: the wrapper consumes
  CU/v4 envelopes internally — wire truth needs the raw child dump, not the client view.
- "Human gate" in GSD protects against unattended real actions, not against the agent
  verifying its own work: reproduce the FULL user-visible flow (exact argv, real backend)
  before any handoff; the user does the acceptance pass only.
- A passing in-repo suite is not field proof: fake-green ≠ real-green. The QA-03 gap was
  invisible to every in-repo test until a real desktop frame capture existed.

### Cost Observations

- Real-account spend: phase-6 mint-only cycles (4×), the final full-cycle turn, ~20 tiny
  one-word QA-03 turns across the three gap cycles + verification; everything else
  fake-tier/zero-spend. All recorded in experiments/README.md.
- Sessions: continuous orchestrator session across phases 6–7 close-out (2026-11-22/23),
  executor/reviewer/verifier subagents per wave.
- Notable: the fake-tier-first architecture kept the real-account spend bounded and the
  debugging reproducible offline.

## Milestone: v0.21.0 — Card Command & Skill Autocomplete

**Shipped:** 2026-11-23
**Phases:** 1 | **Plans:** 1

### What Was Built
- The card's composer autocomplete: ONE `available_commands_update` per session/new —
  ZCode commands (`~/.zcode/commands` + compat), skills (user/compat/plugin-cache/
  workspace SKILL.md frontmatter), and `/model` with the selector hint.

### What Worked
- **Sources first** (the user's directive): the wire contract read from the installed
  chunks was cross-verified against the orca v1.4.262 clone and the zcode clone BEFORE
  implementation — the source check corrected the scanner's root set (the
  `~/.agents` compat pair), the dedupe key (lowercase), and caught the Trim-order bug
  in the manifest parse before it shipped.
- The fake-tier composition made the whole feature CI-locked and the VM GUI pass a
  20-minute affair: no real-account spend anywhere in this milestone.

### Key Lessons
- The client's slash menu distinguishes `commands` and `skills` surfaces, but the omp
  dialect maps everything to `command` — one list serves both; don't invent a second
  wire surface without reading the dialect first.
- Orca's discovery scan matches the catalog frame by SESSION ID — per-session
  emission is required, and the 9 s budget starts at connect (emit at session/new,
  never lazily).

## Milestone: v0.22.0 — Ask-the-User Bridge

**Shipped:** 2026-11-23
**Phases:** 1 | **Plans:** 1

### What Was Built
- The ask-the-user bridge: `interaction/requestUserInput` → ONE ACP form elicitation
  (message/question/option labels+descriptions/Skip); the accept maps to
  `{action, content.answers}` by question text, multi joined ", "; legacy/error
  declines stay honest and never fail the turn.
- Field-verified in a VM GUI and on the user's real Orca — the "окно в третий раз
  отклонено" symptom closed.
- The `/model` catalog root-cause fixes: the probe poke carries the model's reasoning
  level, `session/new` preserves the create snapshot's full unscoped catalog, provider
  labels distinguish same-model entries, and the switch path merges — 19 entries live,
  stable across switches and turns.

### What Worked
- Reusing the existing elicitation surface: the omp card already declared
  `clientCapabilities.elicitation.form`, so the bridge needed no new client capability.
- The field-driven fix loop: the user's report → live probe → root causes → commits with
  live verification (19-entry catalog, provider labels, switch-path merge).
- The manager's stale-verification route: `verification status` refused a report whose
  covered test changed, forcing a real re-verification instead of a checkbox flip.

### What Was Inefficient
- The milestone's ROADMAP was hand-written with only a checklist line — no
  `### Phase 9` details. `init.manager` scans headings, so the manager reported 0
  phases and could not dispatch; the repair came only at close-out. Phase-detail
  headings are load-bearing for the manager, not decoration.
- Phase 9 work landed on the phase-06 branch (the configured `gsd/phase-09-…` branch
  never existed), so phase commits sit on an unrelated branch; branch hygiene needs a
  check at phase start.
- nyquist/security/review capability steps never ran for phases 8–9; the close carried
  two advisory warnings. The advisory-vs-gate split means this can silently recur.

### Patterns Established
- Bridge-phrase contract: the elicitation carries the ask's own labels/descriptions;
  the answer is keyed by QUESTION TEXT (the dialect's merge contract), never by index.
- A bridged human dialog runs with no policy deadline.
- Catalog preservation across rebinds: the create snapshot owns `available`, the rebind
  owns the confirmed current.

### Key Lessons
- Verification staleness is real: a goconst hoist in a covered test file invalidated
  the digest and correctly blocked phase completion until the verifier re-ran.
- Manager visibility depends on ROADMAP phase headings; a checklist-only roadmap is
  invisible to `init.manager` (0 phases) even when the phase directory is complete.
- Final acceptance belongs to the user: the VM GUI pass caught the render path, and the
  real-machine pass confirmed the workflow actually feeds.

### Cost Observations
- No real-account spend recorded for the bridge itself; the model-catalog probes ran
  on the live account (19 entries) during the fix cycle.
- Sessions: one orchestrator session for the manager close-out; the phase execution
  itself predates this milestone-close session.

## Milestone: v0.23.0 — Mount-First AppImage Backend

**Shipped:** 2026-11-24
**Phases:** 1 | **Plans:** 6

### What Was Built
- Mount-first backend: a packed `.AppImage` is mounted by a supervised
  `--appimage-mount` helper and the app-server serves ACP from the mount — no
  extraction, no ~850 MB duplicate; private 0700 base, allowlist helper env,
  whole-line mount validation, ordered teardown (app-server → helper → mount-gone →
  base), zero orphans.
- Fail-loud lifecycle: every readiness/validation/FUSE failure exits 1 with the
  AppImage, step, observed output and remediation; a lost mount ends the session with
  an explicit cause and no respawn; SIGINT/SIGTERM/SIGHUP tear down cleanly; a failed
  teardown outranks a signal (CR-01).
- Version-ranked discovery: one pure-read stage evaluates every candidate, ranks by
  version with a documented tie-break, rejects volatile `/tmp/.mount_*`, records
  per-candidate misses, and deletes the extraction seam.
- Override completeness: `--app-server`/`ZCODE_ACP_APP_SERVER` accepts directory,
  binary, or packed `.AppImage` (literal paths absolutized); mount-safe PAT version.
- Real-machine acceptance: production mount owner → real app-server → `session/create`
  → zero leftovers, sha256 unchanged, 0 extraction entries added.
- Docs and vm-e2e mirror the shipped contract (managed-mount row, ranking, FUSE
  prerequisites, exact failure text).
- Shipped alongside (merged during the milestone window): OMP print mode for Orca
  text tasks (`261124-154`) and `-32003 session_busy` for concurrent prompts
  (`261124-1ou`).

### What Worked
- The tracer wave (10-01) established the whole contract — mount owner, spawn seam,
  binary e2e — before failure handling, discovery, and docs expanded on it.
- A scripted fake AppImage runtime exercised the real exec/readiness/teardown path in
  CI without FUSE; the real-machine tier stayed env-gated and ran once on the host.
- The deep code review earned its cost: CR-01 (a signal masking a failed teardown)
  was a real must-have violation caught before the verifier, with a RED/GREEN arm.
- Running the "human-only" real-machine item as a machine-checkable guard cleared
  `human_needed` honestly instead of asking the user to type a command.
- Security (24 threats, 0 open) and nyquist (0 gaps) gates ran before the verifier,
  so the phase closed with evidence rather than checkbox flips.

### What Was Inefficient
- The manager is cwd-bound: Phase 10 lived in a side worktree, so execution required
  moving the branch into the session workspace before any GSD skill could see it.
- The PR conflicted with master because parallel agents landed OMP text and
  `-32003` while the phase branch waited; the merge replayed an architectural
  conflict in `main.go` (mount lifecycle vs the new `startBackend`/text arm).
- `internal/acpfront` `TestFullCycle` failed twice under load at wave gates and passed
  on rerun and on the master baseline — a pre-existing ordering flake, not a phase
  regression, but it cost a diagnosis detour.
- The repo advertises merge-commit support but rejects merge commits; the first
  merge attempt failed and the close had to switch to squash.
- The verification fingerprint went stale when the PR merge changed `main.go`, so the
  milestone readiness gate correctly blocked until the verifier re-ran — an extra
  cycle caused by merging after verification.

### Patterns Established
- Manager/phase work should be moved into the session workspace (branch checked out
  there) before dispatching, or the manager will report zero phases.
- Before any PR: fetch master; if it advanced, merge it into the phase branch and run
  the full gate before pushing — the PR's own conflict state is not the place to
  discover it.
- Real-machine acceptance is a guard, not a human ritual: when the host has FUSE and a
  real AppImage, the test itself is the evidence.
- Verification is content-addressed: any post-verification merge invalidates it;
  re-run the verifier rather than hand-editing the report.
- In a squash-only repo, tags belong on the post-merge master tip, not the PR head.

### Key Lessons
- A signal must never excuse a failed teardown: exit-cause precedence needs an explicit
  predicate (`MountFailure`) and a binary test that makes teardown genuinely fail.
- Parallel work on master is the norm here; plan for a merge before PR, not a clean
  fast-forward.
- The advisory gates (code review Info findings) must be recorded, not silently
  dropped — `10-REVIEW-DISPOSITION.md` keeps IN-01..IN-07 traceable.
- Squash merges collapse phase history; keep the phase branch or a backup until the
  milestone close is merged and tagged.

### Cost Observations
- No api.z.ai spend: the real acceptance is scoped to mount → app-server →
  `session/create` (D-14) and never sends a prompt.
- Sessions: one manager session through execute/verify/close; one background verifier
  re-run after the PR merge.
- Repo state at close: 10 phases / 66 plans, 12 Go packages, ~63K lines of Go
  (tests included), lint 0 issues, CI green.

