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
