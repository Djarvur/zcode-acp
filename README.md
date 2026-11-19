# zcode-acp

A single static Go binary that exposes ZCode Desktop to the outside world as an
honest **ACP (Agent Client Protocol) v1 server over stdio**: any ACP client can
open ZCode sessions, send prompts, and receive streamed replies (messages,
thinking, tool calls, permission requests). Feature documentation arrives with
later phases; this README currently documents the CI surface and the
branch-protection model.

## CI

Four GitHub Actions workflows run on this repository (shapes shared with the
owner's reference repo, adapted to this project):

| Workflow | Triggers | What it runs |
|----------|----------|--------------|
| `pr-sanity` | push to `master`, any pull request, manual dispatch | Three jobs: **`build / vet / lint / test / tidy`** (go build, go vet, strict pinned golangci-lint, go test with the race detector, the experiments `node --test` suite, and a go.mod/go.sum tidy-diff check), **`govulncheck`** (reachable-vulnerability scan), **`secret-scan`** (secret-shape scan over the whole tracked tree, plus the docs conformance gate) |
| `release` | `v*` tags only | goreleaser build + publish; inert until the release flow lands (no `.goreleaser.yaml` yet) |
| `e2e-matrix` | manual dispatch only | Placeholder skeleton for the future self-hosted GNOME desktop e2e runner; inert until a runner is registered (Phase 7) |
| `security-scheduled` | weekly schedule (Mondays 06:00 UTC) + manual dispatch | govulncheck over the module |

Tool versions come from `mise.toml` only. CI installs the toolchain with
`mise install` and runs the exact mise tasks the local gate loop uses — no
workflow pins a second copy of any tool version.

CI runs offline test tiers only (the Go suite, the experiments node:test
suite, static gates); it never talks to external AI services and never drives
a real desktop session.

## Required checks

Branch protection on `master` requires exactly these three check names —
character-for-character the `pr-sanity` job names (the ` / ` separators in the
first name are significant):

- `build / vet / lint / test / tidy`
- `govulncheck`
- `secret-scan`

Renaming one of these job names requires updating the protection rule in the
same change.

## Branch flow

Merges into `master` go through pull requests only. Phase work uses
`gsd/phase-{phase}-{slug}` branches forked from an up-to-date `master`
(the phase-branch naming comes from the project's planning configuration).

## Branch protection

Protection on `master` is configured fresh via the GitHub API for this
repository — nothing was copied from the reference repo (its only protection
artifact there is a disabled, misnamed ruleset targeting a branch that does
not exist). The applied setup: pull-request merges required (no approving
review count for a solo repo), admins bound by the rules, force pushes and
branch deletion rejected, linear history required, and the three required
checks listed above.

Re-application recipe (after reviewing `protection-body.json`):

```sh
gh api repos/Djarvur/zcode-acp/branches/master/protection -X PUT \
  --input .planning/phases/02-scaffold-ci-bootstrap/protection-body.json

# verify — must return the applied configuration
gh api repos/Djarvur/zcode-acp/branches/master/protection
```

---
*Created: 2026-11-19*
*CI & branch-protection section: Phase 2 — Scaffold & CI Bootstrap*
