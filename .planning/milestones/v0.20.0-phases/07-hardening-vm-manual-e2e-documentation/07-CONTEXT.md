# Phase 7: Hardening, VM & Manual E2E, Documentation - Context

**Gathered:** 2026-11-22
**Status:** Ready for planning
**Mode:** `--auto` — decisions auto-selected (recommended defaults); see `07-DISCUSSION-LOG.md`

<domain>
## Phase Boundary

Phase 7 closes the done-v1 bar: **full CI protocol conformance** (QA-01: one scripted fake
scenario covering initialize → session/new → prompt streaming → tool calls → permissions →
cancel), **session-compat hardening** (SESS-02: `session/delete` graceful no-op, `session/fork`
graceful — never a crash path), **full Orca dialect coverage** (SESS-03: every method Orca's
client calls at card startup and during a turn per DISC-02, including the persona CLI contract —
`--model <id>` honored at launch (`launchesWithModel:true`) and the `/model <name>` slash prompt
with the exact Hb string contracts), **a reproducible clean-VM scenario** (QA-02: install,
configure, run, including the no-keyring error path — scripted in-repo), **manual E2E** (QA-03:
a working agent card in Orca on Ubuntu GNOME on a real account through BOTH auth paths — the
USER GATE), and **English documentation** (DOCS-01 README, DOCS-02 the preset-agent launch
recipe, DOCS-03 docs/ protocol notes with provenance + known limitations). The routed review
findings (WR-01/WR-04/WR-05 lifecycle consistency + IN-01..04 advisories) land here —
Hardening is their designed home per 06-REVIEW-DISPOSITION.md.

Out of scope: session/load/fork real support (v2), quota counter over OAuth (never — API does
not allow), Start Plan headless (documented limitation — browser/captcha required), anything
beyond Ubuntu GNOME v1.

</domain>

<decisions>
## Implementation Decisions

### Session-Compat Surface (SESS-02)
- **D-01:** `session/delete` answers a graceful no-op in the dialect's result shape (child-side
  cleanup follows `session/close` semantics where applicable — the runtime owns the real
  deletion); `session/fork` is handled gracefully — never a crash path. Exact result shapes are
  taken from the dialect document/bundle (researcher pins; fake corpus byte-locks both).

### Full Orca Dialect Coverage (SESS-03)
- **D-02:** the phase-1 dialect method table (the DISC-02 load-bearing set) is swept one-by-one:
  every method is implemented or gracefully handled (coded no-op / protocol-correct decline per
  the established posture) — proven by a fake-corpus conformance row per uncovered method.
- **D-03:** the persona CLI contract lands: the wrapper accepts `--model <provider/model>`
  at launch (the launch binding rides the same setModel follow-up machinery — the reasoning
  level defaults to the backend-confirmed shape) and the `/model <name>` slash prompt is
  handled with the exact Hb string contracts (`Unknown ACP model: …`, `No model selected.` —
  researcher pins the verbatim strings + expected behaviors from the omp card contract);
  `launchesWithModel:true` becomes claimable in the launch recipe (DOCS-02).

### CI Protocol Conformance (QA-01)
- **D-04:** one named CI conformance scenario composes the existing proven corpus pieces into
  the full v1 arc — initialize → session/new → prompt streaming → tool calls → permission
  decision → cancel — byte-locked, named, and green in `mise run ci`. Existing suites stay;
  this is the single named "conformance suite" the done-v1 bar names.

### Reproducible VM Scenario (QA-02)
- **D-05:** the clean-Ubuntu-GNOME scenario is scripted in-repo (`scripts/vm-*` or a documented
  equivalent): install → configure (both auth paths) → run → the no-keyring error path
  asserted. Running it on an actual VM is a MANUAL tier act (the script is the deliverable;
  one documented execution by the user is the field proof — recorded in the ledger).

### Manual E2E (QA-03 — the USER GATE)
- **D-06:** the wrapper side completes everything automatable (recipe correctness, PATH shim,
  card config docs); the actual Orca card E2E through BOTH auth paths is performed by the user
  on the real desktop and recorded in the UAT + ledger (the phase-6 UAT precedent). End-of-phase
  human gate, never autonomous.

### Documentation (DOCS-01/02/03)
- **D-07:** English README (what it is — an honest ACP server for any ACP client; install;
  configuration; both auth paths; troubleshooting), the preset-agent launch recipe (binary
  naming `zcode-acp` with the omp-compatible contract, version floor, env, PATH shim), and
  `docs/` protocol notes with the provenance statement (no patching, no attestation forgery,
  PAT signed by the official binary; Apache-2.0 attribution for ported reference-adapter code)
  and known limitations (no quota counter over OAuth; Start Plan not headless). Language: the
  repo's user-facing docs are English (PROJECT.md), planning artifacts stay as they are.

### Routed Hardening (from 06-REVIEW-DISPOSITION.md)
- **D-08:** WR-01 (refused picker setModel leaves the front session alive vs dead backend),
  WR-04 (three post-create failure paths leak cwd/settings map entries), WR-05 (resume/close
  TOCTOU on isClosing + Register) — settled as one lifecycle-consistency pass in this phase.
  IN-01 (builtin-table double-read), IN-02 (adapter comment claims absent validation), IN-03
  (keyringprobe deadline-boundary classification race), IN-04 (dead `settings == nil` re-init)
  — advisory cleanups batched here. Dispositions flip to resolved as each lands.

### Claude's Discretion
- Exact delete/fork result shapes (dialect-verified); the conformance scenario's name/location;
  VM script form (bash + docs vs testfile); doc file layout inside `docs/`; whether IN-03 needs
  a behavior change or a comment (the fail-fast semantics are already correct); task/wave
  decomposition; which routed advisory lands in which wave.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning & Research (in repo)
- `.planning/REQUIREMENTS.md` — SESS-02/03, QA-01/02/03, DOCS-01/02/03
- `.planning/ROADMAP.md` — Phase 7 success criteria (5 items)
- `.planning/phases/06-oauth-coding-plan-path-model-picker/06-REVIEW-DISPOSITION.md` — the
  routed hardening items (WR-01/04/05, IN-01..04) with their evidence anchors
- `docs/protocol/acp-client-dialect.md` — Orca's client dialect: the DISC-02 load-bearing
  method table SESS-03 sweeps, the omp card contract (min 17.0.5, Hb strings, agentCmdOverrides)
- `docs/protocol/app-server-dialect.md` — the wrapper's back-leg dialect (§3 interaction
  surface incl. the MCP-auth decline row; §8 snapshot facts incl. the paired-env worker fact)
- `docs/protocol/install-discovery.md` — the discovery matrix + revision formula the VM
  scenario and the launch recipe depend on
- `.planning/research/REFERENCE-ADAPTERS.md` + `docs/research/reference-adapters.md` — the
  compiled contract table (Orca floor, agentCmdOverrides) and the Apache-2.0 attribution note
- `.planning/phases/06-oauth-coding-plan-path-model-picker/06-VERIFICATION.md` — the fresh
  phase-6 close state (16/16) and the honest-verifier posture this phase inherits
- agentclientprotocol.com v1 spec — session lifecycle (delete/fork), config options

### External Ground Truth (outside the repo)
- Orca install on this machine (the omp card contract, Hb strings — cite-and-point)
- zcode-src 3.17.1 clone + the shipped 3.18.2 bundle (cite-and-point only)
- FINDINGS.md (private) — the Start-Plan-headless limitation record
- Real Orca + real account on this machine (QA-03's manual tier — the user's gate)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The fake corpus + goldens (phases 3–6) — the conformance scenario composes proven rows
- The phase-6 configOptions/set_config_option machinery — the `/model` slash prompt and
  `--model` launch binding ride it
- The lifecycle machinery (resume/close/teardown WR-07) — WR-01/04/05 harden it
- The phase-6 spawn-env paired-paths fix — the VM scenario must carry the same contract
- experiments ledger + UAT flow (phase-6 precedent) — QA-03's recording shape

### Established Patterns
- TDD RED→GREEN machine evidence; exactly-once; honest errors; zero-dep discipline
- Real-tier: guarded manual tier, ledger + scrubbed captures (AR precedent)
- Strict lint (goswitch-style) — `mise run ci` is the done-bar gate

### Integration Points
- acpfront gains: session/delete + session/fork graceful arms; the /model slash prompt layer
- cmd/zcode-acp gains: `--model` launch binding (setModel follow-up at session/new)
- scripts/: the VM scenario; README + docs/: the English user surface
- lifecycle.go: the WR-01/04/05 consistency pass

</code_context>

<specifics>
## Specific Ideas

- The Hb string contracts are EXACT (`Unknown ACP model: …`, `No model selected.`) — byte-lock
  them in the fake corpus, never paraphrase (SESS-03's acceptance is string-exact).
- QA-03's user gate should be prepared so the user's manual run is ONE recipe + ONE card config
  per auth path — everything else pre-verified.
- The conformance scenario is the done-v1 CI bar — name it so the roadmap's success criterion
  maps to a single green suite.

</specifics>

<deferred>
## Deferred Ideas

- session/load/fork real support — v2
- elicitation/create preference — beyond v1
- Non-Ubuntu-GNOME environments — accepted PRs post-v1

</deferred>

---

*Phase: 7-Hardening, VM & Manual E2E, Documentation*
*Context gathered: 2026-11-22*
