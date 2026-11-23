# Phase 2: Scaffold & CI Bootstrap - Context

**Gathered:** 2026-11-19
**Status:** Ready for planning
**Mode:** `--auto` — all decisions auto-selected (recommended defaults), logged for audit; see `02-DISCUSSION-LOG.md`

<domain>
## Phase Boundary

Phase 2 delivers the **goswitch-style repository scaffold with strict quality gates,
before any feature code**: Go module skeleton (`cmd/zcode-acp`), exact-pinned toolchain
via mise, strict `default: all` golangci-lint v2 config, GitHub Actions CI (lint + test +
build + govulncheck, plus the goswitch release/e2e-matrix/security-scheduled workflow
shapes), the secret-scan gate promoted into CI, and **real `master` branch protection**
documented in the repo. Everything lands into strict gates from the first code commit.

Out of scope: any ACP/wrapper feature code (Phases 3–6), release binaries (Phase 7),
VM/e2e execution (Phase 7), agent implementation.

</domain>

<decisions>
## Implementation Decisions

### Branch Protection Mechanics
- **D-01:** Configure **full protection on `master`** per SCAFF-03 and the goswitch model
  (require PR, required status checks = the pr-sanity check names, reject direct pushes
  and force-pushes), applied via `gh api` — the local `gh` session holds `admin: true`
  on `Djarvur/zcode-acp` (verified 2026-11-19). **Sequencing:** protection is applied as
  the LAST task of the phase, after all Phase 2 work is on `master`. Consequence: GSD
  `branching_strategy` switches `none` → `phase` (`gsd/phase-{phase}-{slug}` branches
  forked from updated master) so Phases 3+ work in PRs — this also honours the owner's
  global git rule (feature branches off fresh master, never direct work on master).
  The strategy change is recorded in `.planning/config.json` and in the repo docs.
  Fallback if the API rejects a rule (org policy etc.): record exactly what was applied
  and what was refused, document the intended full setup, keep the phase honest —
  do NOT fake success. — **Reversibility:** reversible — protection rules are a settings
  change, removable via the same API; the config flip is one line.

### Toolchain Pins
- **D-02:** Exact pins via `mise.toml` per goswitch/STACK.md: Go 1.27.x stable line
  (1.27.4), golangci-lint **2.15.0 pinned exactly** (no `latest`), goreleaser 2.20.2
  (release workflow), node (for the experiments scripts) pinned to the machine's current
  LTS major. CI reads the same pins from mise — single source of versions. go.mod
  declares the same Go version.

### Lint Config
- **D-03:** `.golangci.yml` copied from Djarvur/goswitch (`default: any`-level strict v2
  config), changed ONLY in `goimports.local-prefixes` → `github.com/Djarvur/zcode-acp`.
  Any violation fails the lint target. No linter additions/removals in this phase.

### Workflow Set
- **D-04:** All four goswitch workflow shapes land now, inert until triggered:
  `pr-sanity` (lint + test + build + govulncheck in its own job + secret-scan),
  `release` (goreleaser on tags), `e2e-matrix` (VM matrix, self-hosted runner labels —
  skeleton now, real scenario Phase 7), `security-scheduled` (periodic govulncheck/
  audit). Workflow versions/actions pinned by digest or exact tag per goswitch.

### Secret-Scan in CI
- **D-05:** `experiments/scan-secrets.sh` is promoted into CI **after** fixing the open
  review findings on it (WR-01: credential-body class aligned with the scrubber's
  CRED_RX incl. `.`/`:`; IN-03: pin to repo toplevel). Trivial sibling fixes batched in
  the same touch: WR-02/IN-01/IN-02 (doc re-derivation recipes: `grep -o -e '-NNNN'`,
  digit-bearing method classes, singletons label), WR-03/IN-04 (scrubber boundary-aware
  home rewrite + header-less first line) with a RED→GREEN unit test each per TDD mode,
  and WR-04 (probe header reads desktopVersion at capture time). CI job runs the script
  over the whole tree; secret-scan stays a required check name for protection.

### Module Layout
- **D-06:** Module `github.com/Djarvur/zcode-acp`; binary skeleton `cmd/zcode-acp`
  (compiles, `--version` prints 0.0.0-dev; real CLI arrives with Phases 3–4);
  `internal/` packages appear with Phases 3–4 per ARCHITECTURE.md layout. README stub
  with the honest one-liner + branch-protection/CI documentation section (DOCS groundwork
  for SCAFF-03's "documented in the repo").

### Claude's Discretion
- Exact workflow YAML details within the goswitch shapes; runner versions; cache setup
- Order of gate wiring vs. code skeleton within the wave structure
- How to sequence the config flip (`branching_strategy`) relative to protection apply

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning & Research (in repo)
- `.planning/REQUIREMENTS.md` — SCAFF-01..03 definitions
- `.planning/ROADMAP.md` — Phase 2 success criteria
- `.planning/research/STACK.md` — goswitch scaffold facts (mise pins, workflow shapes,
  golangci config provenance) — HIGH confidence, read locally
- `.planning/research/SUMMARY.md` — §Recommended Stack (scaffold bullet) and phase-2
  mapping ("purely mechanical, zero domain research")
- `.planning/phases/01-research-protocol-documentation/01-CONTEXT.md` — prior-phase
  decisions that bind Phase 2 (neutral phrasing, private paths, scrub-first pipeline)
- `.planning/phases/01-research-protocol-documentation/01-VALIDATION.md` — the Wave 0
  scripts being promoted into CI (`experiments/scan-secrets.sh` reuse note)
- `.planning/phases/01-research-protocol-documentation/01-REVIEW.md` +
  `01-REVIEW-DISPOSITION.md` — the 8 open findings this phase fixes (WR-01..04,
  IN-01..04) before CI adoption
- `.planning/PROJECT.md` — constraints (goswitch repo config model, git workflow rule)

### External Ground Truth (outside the repo — real locations supplied in agent prompts)
- Djarvur/goswitch (github.com/Djarvur/goswitch; local clone — path supplied in agent
  prompts) — the scaffold template: mise.toml, .golangci.yml, four workflows, branch
  protection setup
- GitHub branch protection API docs (via `gh api` in-session)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `experiments/scan-secrets.sh` + `experiments/check-docs.sh` — become CI jobs
  (Phase 1 outputs; fixes from 01-REVIEW land first)
- `experiments/scrub-capture.mjs` tests (node --test) — the existing test suite the CI
  test job runs until Go tests appear
- Phase 1 docs tree — the repo content CI protects

### Established Patterns
- goswitch scaffold (STACK.md) — the mandated template; copy, don't invent
- Neutral phrasing + no machine paths in committed files (Phase 1 discipline)
- Scripts take machine paths as args, never hardcode

### Integration Points
- CI runs `node --test experiments/` (Phase 1 suite) and, from Phase 3, `go test ./...`
- Branch protection required-check names must match the pr-sanity job names exactly
- `mise.toml` pins must match go.mod's Go directive

</code_context>

<specifics>
## Specific Ideas

- goswitch is the owner's own repo — the scaffold should be recognizably its sibling
  (same workflow names/shapes) so maintenance habits transfer.
- Protection applies LAST; everything else lands first. The phase ends with master
  protected and Phases 3+ working via `gsd/phase-*` branches + PRs.
- govulncheck in its own job (goswitch pattern), not bundled into lint.
- The README branch-protection section documents: required checks, PR-only flow, and how
  GSD phase branches relate (`gsd/phase-{phase}-{slug}`).

</specifics>

<deferred>
## Deferred Ideas

- Release binaries/publishing (goreleaser run) — Phase 7
- VM e2e scenario content — Phase 7 (workflow skeleton lands now per D-04)
- Renovate/dependabot config — not in goswitch model; PR-driven later

</deferred>

---

*Phase: 2-Scaffold & CI Bootstrap*
*Context gathered: 2026-11-19*
