# Phase 2: Scaffold & CI Bootstrap - Research

**Researched:** 2026-11-19
**Domain:** Go repository scaffold (mise + strict golangci-lint v2 + GitHub Actions) and GitHub branch protection via REST API
**Confidence:** HIGH

## Summary

Phase 2 is a mechanical scaffold phase grounded in a local, mandated template: the
owner's own goswitch repository (local clone; canonical GitHub fallback
github.com/Djarvur/goswitch). Every artifact this phase needs was read directly from
the template this session: `mise.toml` (tool pins + gate tasks), `.golangci.yml`
(strict `default: all` v2 config), four workflow files (`pr-sanity`, `release`,
`e2e-matrix`, `security-scheduled`). The critical planning facts are now empirically
verified rather than assumed: a probe skeleton (module `github.com/Djarvur/zcode-acp`,
`go 1.27`, `cmd/zcode-acp` `--version` main) with the exact goswitch linter config
changed only in `goimports.local-prefixes` runs GREEN on golangci-lint 2.15.0, turns
RED on a planted violation, and passes `govulncheck` (exit 0).

Two material discoveries change the plan's shape:

1. **goswitch itself has NO active branch protection.** Its `main` branch returns 404
   "Branch not protected"; the only artifact is a ruleset named "master" targeting
   `refs/heads/master` (a branch that does not exist there — the default branch is
   `main`) with `enforcement: "disabled"` and only two rules (`deletion`,
   `non_fast_forward`) — no required checks, no PR requirement. "The goswitch model"
   is therefore an *intent*, not a copyable API state: zcode-acp creates full classic
   branch protection fresh via `gh api`. The repo docs must describe what was actually
   applied, not claim it was "copied from goswitch".
2. **The skeleton has two strict-lint traps, verified live:** `forbidigo`'s default
   pattern `^(fmt\.Print(|f|ln)|print|println)$` bans `fmt.Println/Printf/Print` (but
   NOT the `fmt.Fprint*` family), and `nlreturn` requires a blank line before every
   `return`. A skeleton printing the version with `fmt.Println` fails lint; the
   `_, _ = fmt.Fprintln(os.Stdout, ...)` + blank-line-before-return form passes.

The 8 open Phase 1 review findings (WR-01..04, IN-01..04) all have concrete fix
designs below; one correction to the review's own fix hint is included (WR-03's
suggested regex has a direct-suffix gap — verified by execution).

**Primary recommendation:** Land in this order: (1) the 8 review fixes with their
RED→GREEN tests, (2) module skeleton + mise.toml + `.golangci.yml` (green locally),
(3) four workflows adapted to `master` + a `secret-scan` job, (4) `branching_strategy`
flip `none` → `phase` in `.planning/config.json`, (5) protection applied via `gh api`
as the LAST action, (6) README CI/protection documentation committed before step 5 so
the docs precede the gate they describe.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Branch Protection Mechanics**
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

**Toolchain Pins**
- **D-02:** Exact pins via `mise.toml` per goswitch/STACK.md: Go 1.27.x stable line
  (1.27.4), golangci-lint **2.15.0 pinned exactly** (no `latest`), goreleaser 2.20.2
  (release workflow), node (for the experiments scripts) pinned to the machine's current
  LTS major. CI reads the same pins from mise — single source of versions. go.mod
  declares the same Go version.

**Lint Config**
- **D-03:** `.golangci.yml` copied from Djarvur/goswitch (`default: any`-level strict v2
  config), changed ONLY in `goimports.local-prefixes` → `github.com/Djarvur/zcode-acp`.
  Any violation fails the lint target. No linter additions/removals in this phase.

**Workflow Set**
- **D-04:** All four goswitch workflow shapes land now, inert until triggered:
  `pr-sanity` (lint + test + build + govulncheck in its own job + secret-scan),
  `release` (goreleaser on tags), `e2e-matrix` (VM matrix, self-hosted runner labels —
  skeleton now, real scenario Phase 7), `security-scheduled` (periodic govulncheck/
  audit). Workflow versions/actions pinned by digest or exact tag per goswitch.

**Secret-Scan in CI**
- **D-05:** `experiments/scan-secrets.sh` is promoted into CI **after** fixing the open
  review findings on it (WR-01: credential-body class aligned with the scrubber's
  CRED_RX incl. `.`/`:`; IN-03: pin to repo toplevel). Trivial sibling fixes batched in
  the same touch: WR-02/IN-01/IN-02 (doc re-derivation recipes: `grep -o -e '-NNNN'`,
  digit-bearing method classes, singletons label), WR-03/IN-04 (scrubber boundary-aware
  home rewrite + header-less first line) with a RED→GREEN unit test each per TDD mode,
  and WR-04 (probe header reads desktopVersion at capture time). CI job runs the script
  over the whole tree; secret-scan stays a required check name for protection.

**Module Layout**
- **D-06:** Module `github.com/Djarvur/zcode-acp`; binary skeleton `cmd/zcode-acp`
  (compiles, `--version` prints 0.0.0-dev; real CLI arrives with Phases 3–4);
  `internal/` packages appear with Phases 3–4 per ARCHITECTURE.md layout. README stub
  with the honest one-liner + branch-protection/CI documentation section (DOCS groundwork
  for SCAFF-03's "documented in the repo").

### Claude's Discretion
- Exact workflow YAML details within the goswitch shapes; runner versions; cache setup
- Order of gate wiring vs. code skeleton within the wave structure
- How to sequence the config flip (`branching_strategy`) relative to protection apply

### Deferred Ideas (OUT OF SCOPE)
- Release binaries/publishing (goreleaser run) — Phase 7
- VM e2e scenario content — Phase 7 (workflow skeleton lands now per D-04)
- Renovate/dependabot config — not in goswitch model; PR-driven later
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SCAFF-01 | Go module scaffolded with strict pinned golangci-lint config (goswitch-style `default: all` v2 config, exact version pin) | Verified: probe skeleton + exact config (one-line diff) is GREEN on golangci-lint 2.15.0, RED on planted violation (lll/nlreturn/unused), `--version` prints `zcode-acp 0.0.0-dev`. go.mod shape mirrors goswitch (`module github.com/Djarvur/goswitch` + `go 1.23` → `module github.com/Djarvur/zcode-acp` + `go 1.27`). Skeleton traps (forbidigo `fmt.Print*` ban, nlreturn) documented with the passing form. |
| SCAFF-02 | GitHub Actions CI running lint + test + build (+ govulncheck), mise-driven toolchain versions per goswitch pattern | Four goswitch workflow files read in full; trigger/permission/concurrency/job shapes extracted; adaptations needed for zcode-acp enumerated (push branch `master`, node test step, secret-scan job); govulncheck verified exit-0 on skeleton; mise.run installer + `mise install` flow confirmed from goswitch YAML. |
| SCAFF-03 | `master` branch protection configured and documented (required checks, PR-only workflow) | Full API shape verified (official docs + live 422 probe listing required fields); check-run names verified live from goswitch (`build / vet / lint / test / tidy`, `govulncheck`); goswitch's own protection state discovered (disabled ruleset, 404 classic protection) — protection is created fresh; ready-to-use PUT body drafted; README documentation content outlined. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Toolchain versioning (go, golangci-lint, goreleaser, node) | Repo config (mise.toml) | CI (mise install mirrors it) | Single source of versions; CI must never re-pin (goswitch D-09 rationale) |
| Lint gate | Repo config (.golangci.yml) | Local + CI via `mise run lint` | Same config, same pinned linter, both surfaces |
| Build/vet/test gates | mise.toml tasks | pr-sanity.yml steps call them | CI calls exactly the tasks the owner runs locally |
| Vulnerability scanning | CI (govulncheck jobs) | — | Network-side tool by design; not part of the local gate (goswitch pattern) |
| Secret-shape gate | experiments/scan-secrets.sh | CI job wraps it | Script is the single definition; CI is the enforcement surface |
| Docs conformance gate | experiments/check-docs.sh | CI job wraps it | Same promotion pattern as secret-scan |
| Merge enforcement | GitHub repo settings (branch protection API) | README documents it | Settings live outside git; only their description lives in the repo |
| GSD branch flow | .planning/config.json (`branching_strategy`) | — | Local planning config; flip happens in this phase |

## Standard Stack

### Core
| Tool/Pin | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | `1.27` mise line pin (1.27.4 current stable, released 2026-11-10) | Language/toolchain | Current stable; supported lines are 1.27/1.26 [CITED: go.dev/dl via WebSearch, 2026-11-19; VERIFIED: local go 1.27.3 installed] |
| golangci-lint | `2.15.0` EXACT (no `latest`) | Strict lint gate | v2.15.0 latest (2026-10-16) [CITED: STACK.md, verified 2026-11-19]; exact pin deliberate: the strict linter set churns (goswitch config already carries wsl→wsl_v5, gomodguard→gomodguard_v2, exhaustruct→exhaustruct_v5 deprecations) and only an exact pin reproduces lint results [VERIFIED: goswitch .golangci.yml:13-21 deprecation comments; exhaustruct deprecation warning observed live in probe run] |
| goreleaser | `2.20.2` exact | Release tooling (inert until Phase 7 tags) | Latest stable 2026-11-12 [CITED: STACK.md, verified 2026-11-19]; pinned now per D-02 so release.yml's `mise install` resolves it |
| node | `26` (LTS major; machine runs v26.13.1) | experiments/ scripts + `node --test` CI job | Node 26 entered Active LTS 2026-10-28 [CITED: github.com/nodejs/release schedule via WebSearch, 2026-11-19] |
| mise | `2026.10.17` local; CI installs via `curl https://mise.run \| sh` | Tool version single-source + gate tasks | goswitch pattern; CI calls the same mise tasks as the local loop [VERIFIED: goswitch pr-sanity.yml:39-50] |
| golang.org/x/vuln (govulncheck) | `@latest` in CI only | Reachable-vulnerability gate | Go team first-party module; separate job; non-zero exit only on reachable vulns [VERIFIED: probe run — 1 module-level finding, exit 0] |
| GitHub Actions | 4 workflow files | CI surface | goswitch shapes; `actions/checkout@v4` exact tag (v5–v7 exist; v5+ = node24 runtime, runner ≥ 2.327.1 — mirror goswitch's v4 consciously) [CITED: STACK.md; VERIFIED: goswitch workflows pin @v4] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| mise-driven CI | actions/setup-go + golangci-lint-action | Rejected by goswitch on purpose: the action's `version:` input pins the linter a second time and drifts from mise.toml [VERIFIED: goswitch pr-sanity.yml:7-10 comment] |
| Classic branch protection (recommended) | Repository rulesets | goswitch's only artifact is a disabled ruleset; classic protection is the documented, simplest path for required-checks + PR-requirement on a personal repo [VERIFIED: gh api goswitch rulesets; CITED: docs.github.com REST reference] |
| `checks[]` in required_status_checks | `contexts[]` | `contexts` is deprecated ("closing down notice") per docs — use `checks: [{context, app_id?}]` [CITED: docs.github.com update-branch-protection] |
| Digest-pinned actions | Exact tag `@v4` (goswitch) | D-04 allows either "per goswitch"; goswitch uses tags; mirror for sibling recognition |

**Installation:**
```bash
# No registry packages: the toolchain installs via mise from official distributions
mise install          # reads mise.toml: go 1.27, golangci-lint 2.15.0, goreleaser 2.20.2, node 26
```

**Version verification:** go/golangci-lint/goreleaser versions verified via STACK.md
(researched today, 2026-11-19, HIGH) and re-confirmed for go + node via WebSearch this
session. golangci-lint 2.15.0 is present in the local mise install tree
[VERIFIED: ~/.local/share/mise/installs/golangci-lint/2.15.0]. goreleaser 2.20.2 is NOT
yet in the local mise tree (2.18.1 is) — `mise install` fetches it on demand; not
needed until Phase 7 [VERIFIED: local install listing].

## Package Legitimacy Audit

> No npm/PyPI/crates packages are installed by this phase — no registry legitimacy
> check applies. All tooling arrives via mise from official distribution channels:

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| go (toolchain) | official go.dev/dl | 10+ yrs | n/a | go.dev | OK | Approved — mise `go = "1.27"` |
| golangci-lint | official GitHub releases | 8+ yrs | n/a | github.com/golangci/golangci-lint | OK | Approved — exact `2.15.0` |
| goreleaser | official GitHub releases | 8+ yrs | n/a | github.com/goreleaser/goreleaser | OK | Approved — exact `2.20.2` |
| node | official nodejs.org | 15+ yrs | n/a | github.com/nodejs/node | OK | Approved — `26` LTS major |
| golang.org/x/vuln | Go module (go run @latest) | — | n/a | github.com/golang/vuln | OK | Approved — Go team first-party, CI-only |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
 developer machine                       GitHub
 ┌─────────────────────┐   push/PR    ┌──────────────────────────────────┐
 │  mise.toml (pins)   │ ──────────►  │  .github/workflows/pr-sanity.yml │
 │  ├ mise run build   │              │   jobs: sanity ── mise install   │
 │  ├ mise run vet     │              │    (build/vet/lint/test/tidy)    │
 │  ├ mise run lint    │              │         govulncheck              │
 │  ├ mise run test    │              │         secret-scan              │
 │  ├ test-experiments │              │    release.yml   (v* tags)       │
 │  └ tidy-diff        │              │    e2e-matrix.yml (dispatch)     │
 │  .golangci.yml      │              │    security-scheduled.yml (cron) │
 └─────────────────────┘              └───────────────┬──────────────────┘
        same tasks, same pins                          │ check runs:
                                                      │ "build / vet / lint / test / tidy"
                                                      │ "govulncheck"  "secret-scan"
                                                      ▼
                                       ┌──────────────────────────────────┐
                                       │ branch protection on master      │
                                       │ required checks = check names    │
                                       │ PR required, no force-push,      │
                                       │ enforce_admins                   │
                                       └───────────────┬──────────────────┘
                                                       ▼
                                                  master (protected)
                                       Phases 3+: gsd/phase-{N}-{slug} + PRs
```

Entry point: every change enters via a PR (after protection) or direct push (only
during this phase, pre-protection). Processing: pr-sanity jobs run the identical mise
tasks locally and in CI; check runs named exactly as the job `name:` fields feed the
protection gate. Decision point: all required checks green → merge allowed; anything
else → rejected. External dependency: GitHub REST API (protection apply + verification).

### Recommended Project Structure
```
zcode-acp/
├── .github/workflows/       # pr-sanity.yml, release.yml, e2e-matrix.yml, security-scheduled.yml
├── cmd/zcode-acp/           # main.go skeleton: --version prints 0.0.0-dev
├── experiments/             # Phase 1 pipeline (scan-secrets.sh, check-docs.sh, scrubber + tests, probe)
├── docs/                    # Phase 1 protocol docs (protected by check-docs.sh)
├── .planning/               # planning artifacts (committed; excluded from secret scan)
├── .golangci.yml            # exact goswitch copy, goimports.local-prefixes changed
├── mise.toml                # tool pins + gate tasks (+ node for experiments tests)
├── go.mod                   # module github.com/Djarvur/zcode-acp; go 1.27
├── README.md                # honest one-liner + CI/branch-protection documentation
└── .gitignore               # already covers Go artifacts + experiments/raw/
```

### Pattern 1: mise as the single source of tool versions
**What:** All tool pins live in `mise.toml [tools]`; CI runs `mise install` and every
gate is a mise task; CI steps call `mise run <task>` — never a second version pin.
**When to use:** Always in this repo (PROJECT.md mandates the goswitch repo model).
**Example (recommended mise.toml shape):**
```toml
[tools]
go = "1.27"              # D-02: stable line; resolves to 1.27.4+
golangci-lint = "2.15.0" # D-02: EXACT pin (goswitch floats "2" — deliberate tightening)
goreleaser = "2.20.2"    # D-02: release tool, inert until Phase 7
node = "26"              # D-02: experiments scripts + node --test CI job; Active LTS

[env]
GOFLAGS = "-trimpath"

[tasks.build]
run = "go build ./..."
[tasks.vet]
run = "go vet ./..."
[tasks.lint]
run = "golangci-lint run"
[tasks.test]
run = "go test -race -count=1 ./..."
[tasks.test-experiments]            # new vs goswitch: the Phase 1 node suite
run = "node --test experiments/"
[tasks.tidy-diff]
run = "go mod tidy && git diff --exit-code -- go.mod go.sum"
[tasks.ci]
depends = ["build", "vet", "lint", "test", "test-experiments"]
```
Source: goswitch `mise.toml` structure [VERIFIED: goswitch mise.toml:1-37] with the
D-02 pins and the node task added.

### Pattern 2: strict lint config copied with one intended change
**What:** `.golangci.yml` is copied byte-for-byte from goswitch except
`goimports.local-prefixes` → `github.com/Djarvur/zcode-acp` (the config's only repo-
specific string). Verified by diff: exactly one line differs (line 178).
**When to use:** D-03 locked. Dead exclusion rules referencing goswitch files
(`internal/engine/conn.go` G404 etc.) are inert for zcode-acp (path patterns that
match nothing) and stay per the locked "changed ONLY in local-prefixes" rule.
**Example (the passing skeleton, verified green):**
```go
// Command zcode-acp exposes ZCode Desktop as a standard ACP (Agent Client
// Protocol) v1 server over stdio. This is the Phase-2 module skeleton: it
// compiles and answers --version; the real CLI and agent implementation
// arrive with the later phases.
package main

import (
	"flag"
	"fmt"
	"os"
)

// version is reported by --version until the release flow stamps a real
// value (Phase 7).
var version = "0.0.0-dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		_, _ = fmt.Fprintln(os.Stdout, "zcode-acp "+version)

		return
	}
}
```
[VERIFIED: probe run — `golangci-lint run` exit 0, "0 issues"; `go run ./cmd/zcode-acp
--version` prints `zcode-acp 0.0.0-dev`]

### Pattern 3: CI mirrors the local loop via mise tasks
**What:** pr-sanity has one "sanity" job whose steps are `mise run build/vet/lint/
test/tidy-diff`, a separate `govulncheck` job, and a new `secret-scan` job. Tool
source is `mise install` — never the golangci-lint-action.
**When to use:** D-04 shapes. **Job `name:` fields are the required-check names** —
GitHub check runs carry the job's `name:` (verified live on goswitch), so:
- `sanity` → `name: build / vet / lint / test / tidy`
- `govulncheck` → `name: govulncheck`
- `secret-scan` → `name: secret-scan`
[VERIFIED: gh api goswitch check-runs on main — names `build / vet / lint / test / tidy`,
`govulncheck`, `Dependabot`; VERIFIED: goswitch pr-sanity.yml:33-35,69-71]

### Pattern 4: minimal e2e-matrix skeleton (goswitch discipline, no goswitch internals)
**What:** `e2e-matrix.yml` lands as a skeleton carrying goswitch's *discipline*, not
its content: `workflow_dispatch` only (never push/PR — a fork event must never be able
to drive the self-hosted VM), `runs-on: [self-hosted, gnome]`, `permissions: contents:
read`, serial concurrency group (`e2e-gnome`, cancel-in-progress false), a generous
`timeout-minutes`, and a placeholder step. The goswitch file's 15 KB of AT-SPI
preflight / matrix plumbing is NOT copied (it drives goswitch's desktop fixtures).
**When to use:** D-04 "skeleton now, real scenario Phase 7". [VERIFIED: goswitch
e2e-matrix.yml:44-77 trigger/permission/concurrency/runner/timeout shape]

### Pattern 5: release.yml copied as shape, inert by construction
**What:** `release.yml` triggers on `v*` tags with `permissions: contents: write` (the
ONLY write in the set), `fetch-depth: 0`, mise-installed goreleaser, `goreleaser
release --clean` with GITHUB_TOKEN, serial `concurrency: release`. Inert until Phase 7
because no v* tag exists and `.goreleaser.yaml` is not written yet — a premature tag
would fail the job, which is the honest outcome. Workflow registration resolves from
the default branch, so tag triggers only fire after this phase merges. [VERIFIED:
goswitch release.yml:19-64 including the registration note at lines 17-18]

### Pattern 6: protection applied LAST, verified from the API
**What:** The final task applies classic branch protection via `gh api` and verifies
by reading it back. Recommended body (solo-owner repo, zero approvals, admin bound):
```json
{
  "required_status_checks": {
    "strict": false,
    "checks": [
      { "context": "build / vet / lint / test / tidy" },
      { "context": "govulncheck" },
      { "context": "secret-scan" }
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": false,
    "require_code_owner_reviews": false,
    "required_approving_review_count": 0
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_linear_history": true,
  "required_conversation_resolution": false
}
```
Apply: `gh api repos/Djarvur/zcode-acp/branches/master/protection -X PUT --input <body>`.
Verify: `gh api repos/Djarvur/zcode-acp/branches/master/protection` (GET).
**Semantics, per official docs:** the four keys `required_status_checks`,
`enforce_admins`, `required_pull_request_reviews`, `restrictions` are the required
body members (each object-or-null); "Require a pull request before merging" IS
`required_pull_request_reviews` non-null, and `required_approving_review_count: 0`
explicitly means "no reviewers required" — this is what makes a solo repo's PR-only
flow workable. `restrictions: null` is the documented setting for personal repos
(push restrictions are org-only). `enforce_admins: true` is what actually binds the
owner (without it admins bypass). `required_linear_history: true` is feasible: the
repo currently allows squash/rebase/merge [VERIFIED: gh api repo fields —
allow_squash_merge:true, allow_rebase_merge:true, allow_merge_commit:true].
[CITED: docs.github.com/en/rest/branches/branch-protection#update-branch-protection;
VERIFIED: live 422 probe on this repo — `{"required_status_checks": null}` alone
returns `{"message":"Invalid request.\n\n\"enforce_admins\",
\"required_pull_request_reviews\", \"restrictions\" weren't supplied.","status":"422"}`,
proving (a) the required-key set and (b) that the admin token reaches the endpoint]

### Pattern 7: the eight Phase 1 review fixes (D-05), concrete designs

All findings from `01-REVIEW.md` (dispositions all "open"); batched per D-05.

| Finding | File (verbatim anchor) | Fix design |
|---------|------------------------|------------|
| WR-01 | `experiments/scan-secrets.sh:19` — pattern `'enc:v[12]:(os:)?[A-Za-z0-9+/=_-]{20,}'` | Change the body class to match the scrubber's CRED_RX exactly: `'enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}'` (CRED_RX verbatim: `/enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}/g` at `scrub-capture.mjs:48`); optionally drop the closing-quote requirement from pattern (c) to match TOKEN_RX (`/("(?:access_token\|apiKey\|api-key\|token)"\s*:\s*")[A-Za-z0-9_-]{20,}(")/g` at `scrub-capture.mjs:49`). Update the header comment to match. Verified GREEN on the current tree with both widened patterns (no matches outside `.planning`). Safe against the test file: `scrub-capture.test.mjs` builds all forbidden shapes by string concatenation by design ("IMPORTANT: forbidden shapes … are constructed here by STRING CONCATENATION so that no forbidden literal appears verbatim in this tracked file", lines 5-8). |
| IN-03 | `experiments/scan-secrets.sh:25` — CWD-relative `git grep` pathspec | Start the script with `cd "$(git rev-parse --show-toplevel)" \|\| exit 1`. Same one-liner in `check-docs.sh` for symmetry (it already fails loudly from a wrong CWD, so its fix is optional-but-cheap — the review hint names both scripts). |
| WR-02 | `docs/protocol/app-server-dialect.md:146` | Record the working recipe: `grep -o -e '-32603' <bundle>/resources/glm/zcode.cjs \| wc -l` (or `-- '-32603'`); GNU grep silently parses a bare `-32603` as numeric context options (rc=1, no output, no error — verified in review with GNU grep 3.11). |
| IN-01 | `docs/protocol/app-server-dialect.md:40-41` | Widen the recorded command's class to `'"[a-zA-Z0-9/]*"$'` and/or add the one-line v4 extractor `grep -o '"v4/[^"]*"' <bundle> \| sort -u` to the v4 paragraph (all 43 v4 strings contain digits the current class drops). |
| IN-02 | `docs/protocol/app-server-dialect.md:65` | Change the row label `singletons (8)` → `singletons (9)` (nine methods listed; the 110 headline arithmetic requires 9). |
| WR-03 | `experiments/scrub-capture.mjs:107-110` | Boundary-aware home rewrite: keep `home + '/' → '~/` unconditional (step 1), then replace bare `home` only when NOT followed by `/` or a name character: `line.replace(new RegExp(escapeRegExp(home) + '(?!/?[A-Za-z0-9._-])', 'g'), '~')` with `escapeRegExp = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')`. **Correction to the review's literal hint:** the hint's regex `(?!/[A-Za-z0-9._-])` has a direct-suffix gap — verified by execution: for home `/home/u`, input `/home/ux` → the hint form yields `~x` (still falsified) while `(?!/?[A-Za-z0-9._-])` leaves it verbatim. The RED→GREEN tests MUST cover both sibling cases (`/home/ux/otherproj/file.txt` AND `/home/ux`) plus the truth-preserving cases (`/home/u` → `~`, `/home/u/proj/e3` → `~/proj/e3`) and idempotency. Current buggy behavior confirmed live: `~x/otherproj/file.txt`. |
| IN-04 | `experiments/scrub-capture.mjs:163` | Validate line 1 parses as JSON containing a `capture` key; fail loudly (exit ≠ 0 with a clear message) otherwise, with a `--no-header` escape hatch for legitimately header-less raw captures. RED→GREEN unit test: header-less input fails without the flag, passes with it. Keep empty-input → empty-output behavior. |
| WR-04 | `experiments/app-server-probe.mjs:473` — header writes `desktopVersion: '3.18.2'` as a constant | Read at capture time like `builtinRevision` (which the probe already reads from `flags.builtin`, lines ~457-465). Two options from the review hint: (a) resolve from the bundle's version metadata near the builtin file, or (b) accept a `--desktop-version` CLI arg resolved by the runner. Recommended for Phase 2: option (b) — minimal, no new discovery logic; default to `'unknown'` when the arg is absent so the header never asserts an unread value. No unit test required (probe harness has no test suite; D-05 attaches tests only to WR-03/IN-04). |

**Doc-recipe edits (WR-02/IN-01/IN-02) have no gate impact:** `check-docs.sh
--phase-gate` pins section HEADINGS only (its `REQUIRED_SECTIONS` map), not row labels
or inline commands [VERIFIED: check-docs.sh:52-60].

### Anti-Patterns to Avoid
- **Re-pinning tool versions in CI** (golangci-lint-action, setup-go with a version):
  drifts from mise.toml; goswitch rejected it explicitly.
- **Applying protection before the phase's work is on master**: blocks the phase's own
  direct pushes; D-01 sequences protection LAST.
- **Renaming pr-sanity job `name:` fields after protection is applied** without
  updating the protection rule: a stale required context never reports, blocking all
  merges ("Expected — waiting for status" forever).
- **Copying goswitch's e2e-matrix.yml wholesale**: it drives goswitch's desktop
  fixtures (AT-SPI preflight, matrix YAML files, focus_helper.py) — none exist here.
- **`fmt.Println` in cmd code**: banned by forbidigo's default pattern under `default:
  all`; use `fmt.Fprintln` with an explicit `_, _ =` error discard.
- **Writing the review's WR-03 hint regex verbatim** `(?!/[A-Za-z0-9._-])`: it still
  falsifies direct-suffix siblings (`/home/ux` → `~x`); use `(?!/?[A-Za-z0-9._-])`.
- **Assuming goswitch's protection can be "copied"**: nothing active exists there to
  copy (disabled ruleset, wrong branch name, no required checks).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Tool version management local vs CI | Custom scripts to sync versions | mise.toml + `mise install`/`mise run` | One file; exact local/CI parity; goswitch-proven |
| Strict lint config | Hand-picking linters for a fresh repo | goswitch `.golangci.yml` verbatim copy | The config is battle-tested against `default: all` churn; D-03 locked |
| Secret-shape detection | New CI-native secret scanner | Existing `experiments/scan-secrets.sh` after WR-01/IN-03 | Already tested green; aligned with the scrubber's proven credential shapes; zero deps |
| Branch protection | Clicking through the Settings UI (undocumented) | `gh api` PUT + GET verification, documented in README | SCAFF-03 requires it "documented in the repo"; API apply is reproducible and reviewable |
| Test runner for Phase 1 suite | Custom script to discover tests | `node --test experiments/` | Built-in Node runner; suite already green (12/12 verified this session) |

**Key insight:** The template exists precisely so this phase invents nothing — every
hand-rolled alternative above already failed somewhere else or violates a locked
decision.

## Runtime State Inventory

> Omitted: Phase 2 is a greenfield scaffold, not a rename/refactor/migration phase.
> Nothing runtime-hosted carries a Phase 2 string yet: the repo has no Actions
> workflows, no protection rules (verified 404 on this repo's master probe), no
> releases, and no external services registered against it.

## Common Pitfalls

### Pitfall 1: forbidigo bans `fmt.Print*` in cmd code
**What goes wrong:** The obvious skeleton prints the version with `fmt.Println` and
fails the strict gate with ``use of `fmt.Println` forbidden by pattern
`^(fmt\.Print(|f|ln)|print|println)$` ``.
**Why it happens:** `linters.default: all` enables forbidigo with its default
patterns; the goswitch config excludes forbidigo only for `test/e2e/.*`.
**How to avoid:** Print via the `fmt.Fprint*` family (not matched by the pattern)
with an explicit `_, _ =` discard (satisfies errcheck), as in Pattern 2.
**Warning signs:** a forbidigo finding on `cmd/` during the first lint run.
[VERIFIED: probe run reproduced exactly this failure, then the Fprintln form passed]

### Pitfall 2: nlreturn style — blank line before return
**What goes wrong:** `return with no blank line before (nlreturn)` failures.
**Why it happens:** `default: all` enables nlreturn; goswitch code style already
complies so it is invisible in the template's own files.
**How to avoid:** Put a blank line before every top-of-block `return` (and match
goswitch's spacing in new code).
**Warning signs:** nlreturn appears in the first lint run alongside other issues.
[VERIFIED: probe run]

### Pitfall 3: check names vs required contexts drift
**What goes wrong:** Protection requires a context name that no job emits (typo, or a
job `name:` renamed later), so PRs hang on "Expected" forever.
**Why it happens:** Required status checks match the CHECK RUN name — the job's
`name:` field — not the workflow name and not the job id.
**How to avoid:** Copy the three names from the pr-sanity.yml `name:` fields into the
PUT body character-for-character (including the ` / ` separators in `build / vet /
lint / test / tidy`); if a job is ever renamed, update protection in the same change.
**Warning signs:** a PR whose required check never starts.
[VERIFIED: goswitch check-run names observed live via gh api]

### Pitfall 4: goswitch workflow `on.push.branches` says `main`
**What goes wrong:** A verbatim workflow copy never triggers on pushes to this repo —
its default branch is `master`.
**Why it happens:** Template and target differ in default branch name.
**How to avoid:** Adapt `branches: [main]` → `branches: [master]` in pr-sanity.yml
(within D-04's "exact workflow YAML details" discretion; record the adaptation in the
file header comment). All four workflows need a similar check for repo-specific
strings.
**Warning signs:** pr-sanity not running after merge to master.
[VERIFIED: goswitch pr-sanity.yml:16-20 `branches: [main]`; zcode-acp default branch
`master` via gh api]

### Pitfall 5: protection applied too early blocks the phase itself
**What goes wrong:** With `enforce_admins: true` and PR-only flow, the phase's own
direct pushes to master are rejected mid-phase.
**Why it happens:** Protection is a settings change that binds immediately.
**How to avoid:** D-01 sequencing — protection is the LAST task. Land everything
(including the README documentation section and the `.planning/config.json` flip)
first; the README can honestly describe the setup the phase then applies minutes
later, verified by the GET call.
**Warning signs:** a rejected push with `remote: error: GH006` during the phase.

### Pitfall 6: stale linter deprecations are warnings, not failures — but the exact pin still matters
**What goes wrong:** Nothing fails, yet `golangci-lint` prints `The linter
'exhaustruct' is deprecated (since v2.13.0) ... Replaced by exhaustruct_v5`.
**Why it happens:** The goswitch config disables exhaustruct's issues via an
exclusion rule (`linters: [exhaustruct, exhaustruct_v5], text: ".*"`) while the
linter stays enabled; future golangci-lint releases may turn deprecations into
removals and break the floating config.
**How to avoid:** The D-02 exact pin (`2.15.0`) is the mitigation — do not "clean
up" the config in this phase (D-03 forbids linter changes).
**Warning signs:** new deprecation warnings after a linter bump.
[VERIFIED: probe run output]

### Pitfall 7: `go test -race ./...` on a no-test module
**What goes wrong:** Expectation mismatch that CI "test" proves something in Phase 2.
**Reality:** With only `cmd/zcode-acp` present, `go test -race -count=1 ./...` exits 0
with "no test files" — fine (the gate is forward-looking); the Phase 1 node suite
(`node --test experiments/`, 12/12 green this session) is the real test gate until Go
tests arrive in Phase 3.
**How to avoid:** Wire both into CI (Pattern 1's `test-experiments` task) so the
suite coverage claim is honest.

## Code Examples

### pr-sanity.yml for zcode-acp (adapted shape)
```yaml
# Adapted from Djarvur/goswitch pr-sanity.yml (sibling repo). Changes: push
# branch master (default branch here), test-experiments step (node suite),
# secret-scan job (D-05), docs-gate step inside the secret-scan job.
name: pr-sanity

on:
  push:
    branches: [master]        # goswitch template says main — adapted
  pull_request:
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: pr-sanity-${{ github.event_name }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  sanity:
    name: build / vet / lint / test / tidy   # this string is a required check name
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install mise
        run: curl https://mise.run | sh
      - name: Add mise to PATH
        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"
      - name: Install tools from mise.toml
        run: mise install
      - name: build
        run: mise run build
      - name: vet
        run: mise run vet
      - name: golangci-lint
        run: mise run lint
      - name: test (go)
        run: mise run test
      - name: test (experiments suite)
        run: mise run test-experiments
      - name: modules are tidy
        run: mise run tidy-diff

  govulncheck:
    name: govulncheck                        # required check name
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install mise
        run: curl https://mise.run | sh
      - name: Add mise to PATH
        run: echo "$HOME/.local/bin" >> "$GITHUB_PATH"
      - name: Install tools from mise.toml
        run: mise install
      - name: govulncheck
        run: mise exec -- go run golang.org/x/vuln/cmd/govulncheck@latest ./...

  secret-scan:
    name: secret-scan                        # required check name (D-05)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: secret scan
        run: bash experiments/scan-secrets.sh
      - name: docs conformance gate
        run: bash experiments/check-docs.sh
```
[VERIFIED: goswitch pr-sanity.yml:14-93 job/step shapes; node/go additions per D-02/D-04/D-05]

### go.mod
```
module github.com/Djarvur/zcode-acp

go 1.27
```
[VERIFIED: probe go.mod compiles + lints green; goswitch go.mod shows the shape
`module github.com/Djarvur/goswitch` + `go 1.23`]

### Branch protection apply + verify
```bash
# apply (LAST task of the phase)
gh api repos/Djarvur/zcode-acp/branches/master/protection -X PUT \
  --input .planning/phases/02-scaffold-ci-bootstrap/protection-body.json

# verify — must return the applied JSON
gh api repos/Djarvur/zcode-acp/branches/master/protection

# negative check — a direct push to master must now be rejected (GH006)
git push origin master   # expected: ! [remote rejected]
```
[D-01 fallback honored: whatever the API refuses is recorded, not faked]

### goswitch ruleset actually found on the template repo (for the README's honesty note)
```json
{"id":26724482,"name":"master","target":"branch","enforcement":"disabled",
 "conditions":{"ref_name":{"include":["refs/heads/master"],"exclude":[]}},
 "rules":[{"type":"deletion"},{"type":"non_fast_forward"}]}
```
[VERIFIED: gh api repos/Djarvur/goswitch/rulesets/26724482 — the template's only
protection artifact: disabled, misnamed, targets a non-existent branch, no required
checks. zcode-acp's protection is a fresh implementation of the intent, and the README
should say so.]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| golangci-lint v1 config | v2 config (`version: "2"`, `linters.default: all`, `formatters:` section) | goswitch config already v2 | Copy works as-is on 2.15.0 [VERIFIED: probe] |
| `actions/checkout@v2/v3` | @v4 in goswitch (v5–v7 exist, v5+ = node24 runtime) | ongoing | Mirror @v4 per goswitch; conscious bump later [CITED: STACK.md] |
| Classic branch protection only | Rulesets API exists alongside | ~2023+ | goswitch tried a ruleset (disabled); classic protection chosen here — simplest path to required-checks + PR-requirement |
| `contexts` in required_status_checks | `checks: [{context, app_id}]` | docs mark contexts deprecated | Use `checks` in the PUT body [CITED: docs.github.com] |
| Node.js two majors/year | One major/year starting Node 27 | Oct 2026 | `node = "26"` is a long-lived LTS major [CITED: nodejs release schedule] |

**Deprecated/outdated:**
- `wsl`, `gomodguard`, `exhaustruct`, `noinlineerr` era configs: superseded inside the
  goswitch config's disable list — irrelevant here, just keep the exact linter pin.
- `grep -o '-NNNN'` recipes: silently broken for negative codes (WR-02) — use
  `grep -o -e '-NNNN'`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | goswitch's protection is intentionally dormant (owner tried a ruleset on 2026-11-06 and disabled it) rather than an oversight that hides a preferred active setup elsewhere | Pattern 6 | Low — D-01 fully specifies the target protection; goswitch's state is context, not a spec |
| A2 | `required_pull_request_reviews` with `required_approving_review_count: 0` enforces PR-only merges for a solo repo | Pattern 6 | Medium if wrong — if GitHub treats 0-reviews as "no PR requirement", direct pushes stay blocked anyway via required checks on PRs only... mitigations: D-01's fallback records exactly what was applied; the negative push test verifies behavior empirically at apply time |
| A3 | `strict: false` (no "branch up to date" requirement) is acceptable for a solo repo | Pattern 6 | Low — friction choice only; flipping it later is one API call |
| A4 | secret-scan and docs-gate can share one CI job named `secret-scan` | pr-sanity example | Low — if separate names are wanted (e.g. a distinct `docs-gate` check), add one job + one required context; planner discretion |
| A5 | Hosted `ubuntu-latest` runners satisfy all four workflows (bash, git, curl, mise-installable tools) | Standard Stack | Low — goswitch runs the identical shape on ubuntu-latest today [VERIFIED: goswitch check-runs green on main] |
| A6 | e2e-matrix skeleton's self-hosted label `[self-hosted, gnome]` — no such runner is registered yet, so dispatches queue forever (inert, as intended) | Pattern 4 | Low — skeleton is dispatch-only by design; Phase 7 registers the runner and scenario |
| A7 | mise `node = "26"` resolves to the latest 26.x on install (major-line floating) — matches D-02's "pinned to the LTS major" wording | Standard Stack | Low — if an exact node pin is wanted later, it is a one-line mise.toml change |

## Open Questions (RESOLVED 2026-11-19)

All three questions were resolved at planning time; the adopted resolutions are
implemented in the phase's executable plan content (plans 02-04 and 02-05). No
question below remains open.

1. **check-docs.sh promotion scope — RESOLVED: docs-gate runs as a step inside the
   secret-scan job; adopted by plan 02-04 (Task 1).**
   - What we know: CONTEXT.md's Reusable Assets say both gate scripts "become CI
     jobs"; D-04's locked pr-sanity list names only secret-scan.
   - Adopted resolution: run `check-docs.sh` as a step inside the secret-scan job
     (Pattern: A4) — protected via the job's required context without expanding
     D-04's list of required check names. [Implemented: 02-04 Task 1, second step
     of the `secret-scan` job.]
2. **`required_linear_history` on or off — RESOLVED: `true`; adopted by plan 02-05
   (Task 1 body, read back via GET in Task 2).**
   - What we know: feasible today (squash/rebase allowed); goswitch has no precedent
     (its ruleset lacks it); GSD squash-merges or fast-forwards PRs.
   - Adopted resolution: set `true` (clean master history; goswitch-sibling spirit);
     drop to `false` if the API or the GSD merge flow objects — D-01 fallback
     applies (refusals recorded honestly, never faked). [Implemented: 02-05 Task 1,
     `required_linear_history: true` in protection-body.json; verified by the
     Task 2 GET check `linear: .required_linear_history.enabled`.]
3. **`delete_branch_on_merge` — RESOLVED: leave untouched this phase; adopted by
   plan 02-05 by omission (no task reads or writes this repo setting).**
   - What we know: repo currently `false` [VERIFIED: gh api repo fields]; goswitch
     state unknown/not researched.
   - Adopted resolution: leave as-is this phase (out of D-01 scope); optionally
     enable later so `gsd/phase-*` branches clean up after merge. [Implemented:
     02-05's only repo-settings write is the protection PUT, whose body does not
     carry this field — the setting stays `false`.]

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| mise | all gates, CI parity | ✓ | 2026.10.17 | — |
| go | module skeleton, gates | ✓ | 1.27.3 local; mise line `1.27` resolves to 1.27.4+ | — |
| golangci-lint | SCAFF-01 gate | ✓ | 2.15.0 in local mise tree | `mise install` fetches it |
| goreleaser | release.yml (inert) | via mise on demand | 2.20.2 (not yet local; 2.18.1 is) | not needed until Phase 7 |
| node | experiments suite, CI test step | ✓ | v26.13.1 (LTS) | — |
| gh (authenticated, admin on Djarvur/zcode-acp) | protection apply/verify, workflow verification | ✓ | 2.105.0 | REST API via curl |
| git | gates, repo flow | ✓ | system | — |
| GitHub hosted runners (ubuntu-latest) | all workflows | ✓ | current | — |
| self-hosted VM runner with GNOME | e2e-matrix skeleton | ✗ (not yet registered) | — | Skeleton is dispatch-only and inert by design — Phase 7 registers it |

**Missing dependencies with no fallback:** none blocking.
**Missing dependencies with fallback:** self-hosted runner (Phase 7 concern; skeleton
stays inert); goreleaser 2.20.2 (fetched by `mise install` on first need).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | node:test (built-in runner, node 26) for the existing suite; golangci-lint 2.15.0 gate; bash gates (scan-secrets.sh, check-docs.sh); go build/vet/test via mise tasks |
| Config file | .golangci.yml (new, this phase); mise.toml (new, this phase) |
| Quick run command | `bash experiments/scan-secrets.sh && bash experiments/check-docs.sh && node --test experiments/` |
| Full suite command | `mise run ci` (build + vet + lint + test + test-experiments) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SCAFF-01 | Skeleton passes the strict pinned gate | gate (machine check) | `mise run lint` (expect exit 0 on skeleton) | ✅ verified in probe; repo files land this phase |
| SCAFF-01 | Any violation fails the lint target | gate negative test | plant a 130-char line in a scratch file → `golangci-lint run` must exit ≠ 0 (verified live: 3 issues, rc=1), then delete the file | ✅ recipe verified; run at task commit |
| SCAFF-01 | Exact pin reproduces | version check | `golangci-lint version` prints `2.15.0` | ✅ |
| SCAFF-02 | PR triggers the workflows | CI observation | open the phase-close PR / push → `gh run list --workflow=pr-sanity.yml` shows runs; `gh api repos/Djarvur/zcode-acp/commits/<sha>/check-runs` lists the three check names | ❌ Wave 0 (workflows land this phase) |
| SCAFF-02 | govulncheck green on skeleton | CI job | `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` (verified exit 0 locally in probe) | ✅ recipe verified |
| SCAFF-03 | Protection configured | API check | `gh api repos/Djarvur/zcode-acp/branches/master/protection` returns required checks + enforce_admins:true + PR requirement | ❌ applied as the phase's last task |
| SCAFF-03 | Direct pushes rejected | negative test | `git push origin master` from master after apply → remote rejected (GH006) | ❌ post-apply |
| D-05 | Scrubber boundary fix + header guard behave | unit (node:test) | `node --test experiments/` — new RED→GREEN tests for WR-03 (both sibling cases) and IN-04 land with the fixes | ❌ Wave 0 (tests written with fixes, per TDD mode) |
| D-05 | Widened secret patterns stay green | gate | `bash experiments/scan-secrets.sh` after the pattern change (verified green on current tree this session) | ✅ |

### Sampling Rate
- **Per task commit:** quick gates (`scan-secrets` + `check-docs` + `node --test`), and `mise run lint` once the config lands
- **Per wave merge:** `mise run ci`
- **Phase gate:** full suite green + workflows observed running + protection GET matches the documented body before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] WR-03 + IN-04 unit tests in `experiments/scrub-capture.test.mjs` (RED→GREEN, per TDD mode)
- [ ] `mise.toml`, `.golangci.yml`, `go.mod`, `cmd/zcode-acp/main.go`, four workflow files — created this phase (probe contents above are the verified starting shapes)
- [ ] No framework installs needed: node:test is built-in; linter arrives via mise

## Security Domain

### Applicable ASVS Categories (security_enforcement: true, level 1)

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | no auth surfaces in this phase |
| V3 Session Management | no | no sessions |
| V4 Access Control | yes (repo-level) | branch protection: PR-only merges, enforce_admins, no force-push/deletion |
| V5 Input Validation | yes (scripts) | gate scripts take no untrusted runtime input; scan patterns are fixed literals; scrubber takes machine paths as args (Phase 1 contract) |
| V6 Cryptography | no | none (secret-shape *detection* only, no crypto) |
| V14 Configuration/CI hardening | yes | `permissions: contents: read` on all workflows except release (`contents: write`); actions pinned to exact tag; secrets: only GITHUB_TOKEN, no custom secrets |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Credential material committed to the repo | Information disclosure | scan-secrets.sh in CI as a required check (D-05), aligned with the scrubber's proven shapes (WR-01) |
| Tampered master history (force-push, deletion) | Tampering | protection rules: `allow_force_pushes: false`, `allow_deletions: false`, `required_linear_history: true` |
| Unauthorized direct pushes bypassing review | Elevation of privilege | `required_pull_request_reviews` non-null + `enforce_admins: true` |
| Supply-chain via runner-side actions | Tampering | exact-tag pinning (`actions/checkout@v4`), minimal permissions, mise as the only tool source |
| Workflow injection via `${{ }}` interpolation of untrusted values | Tampering | goswitch canonical: pass event fields through `env:` (pattern visible in goswitch e2e-matrix preflight); Phase 2 workflows interpolate only non-attacker-controlled values (`github.event_name`, `github.ref`) |
| Over-privileged default token | Elevation of privilege | explicit `permissions:` block in every workflow (goswitch carries it in all four files) |

## Sources

### Primary (HIGH confidence)
- Local goswitch clone (path supplied per agent prompt; canonical: github.com/Djarvur/goswitch, public, owner's own repo) — read this session: `mise.toml` (pins/tasks), `.golangci.yml` (full strict config), `.github/workflows/{pr-sanity,release,e2e-matrix,security-scheduled}.yml` (full), `go.mod`, `cmd/skillgen/main.go` (house cmd style)
- Local zcode-acp tree — `experiments/scan-secrets.sh`, `experiments/scrub-capture.mjs`, `experiments/scrub-capture.test.mjs`, `experiments/check-docs.sh`, `experiments/app-server-probe.mjs` (header section), `.planning/config.json`, `.planning/REQUIREMENTS.md`, phase CONTEXT/REVIEW/DISPOSITION files
- gh api (live, this session): goswitch branch protection GET (404), goswitch ruleset 26724482 (disabled, quoted), goswitch check-runs on main (names), goswitch default branch, zcode-acp repo fields (default branch master, admin:true, merge settings), zcode-acp protection 422 validation probe
- Local empirical probe (/tmp/scaffold-lint-probe): golangci-lint 2.15.0 green/red runs, `go run ./cmd/zcode-acp --version` output, govulncheck exit 0, WR-03 regex variant executions

### Secondary (MEDIUM confidence)
- [CITED: docs.github.com/en/rest/branches/branch-protection#update-branch-protection] — PUT body contract, checks vs contexts, `required_approving_review_count` 0..6 semantics, restrictions-null-for-personal-repos, "require PR" mapping
- [CITED: go.dev/dl + go.dev/doc/devel/release] — go1.27.4 current stable (2026-11-10, security release) via WebSearch
- [CITED: github.com/nodejs/release schedule] — Node 26 Active LTS since 2026-10-28 via WebSearch
- `.planning/research/STACK.md` — golangci-lint 2.15.0 / goreleaser 2.20.2 release provenance (verified same day, 2026-11-19)

### Tertiary (LOW confidence)
- None — no claim in this document rests on a single unverified web source.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every pin/template file read locally this session; versions cross-verified via STACK.md (same-day) and WebSearch
- Architecture: HIGH — goswitch workflow shapes read in full; CI/protection mechanics verified via live gh api evidence and an executed lint/govulncheck probe
- Pitfalls: HIGH — the two skeleton traps were reproduced and resolved empirically; WR-03's hint gap was demonstrated by execution

**Research date:** 2026-11-19
**Valid until:** 2026-12-19 (stable domain: pins may drift only when deliberately bumped; check golangci-lint/go releases if planning slips past a month)

## RESEARCH COMPLETE
