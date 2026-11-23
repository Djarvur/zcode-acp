# Phase 2: Scaffold & CI Bootstrap - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-19
**Phase:** 2-Scaffold & CI Bootstrap
**Areas discussed:** Branch protection mechanics, Toolchain pins, Lint config, Workflow set, Secret-scan in CI, Module layout
**Mode:** `--auto` (manager-driven autonomous run; standing user directive: proceed automatically, stop only when unavoidable). Every choice below is the auto-selected recommended default.

---

## Branch protection mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| Full protection (require PR + required checks) + GSD branching_strategy → `phase` | Honours SCAFF-03 literally AND the owner's global git rule; applied LAST so Phase 2 lands on master first | ✓ |
| Light protection (no force-push/delete, direct pushes allowed) | Keeps branching none, but does not satisfy "direct pushes are rejected" | |
| Document-only, apply later | Does not satisfy SCAFF-03 | |

**User's choice:** *(auto)* Full protection + phase branches from Phase 3
**Notes:** `gh` session verified `admin: true` on Djarvur/zcode-acp (2026-11-19). Fallback: record refusals honestly.

---

## Toolchain pins

| Option | Description | Selected |
|--------|-------------|----------|
| Exact pins via mise (goswitch/STACK.md values) | Single source of versions; golangci-lint 2.15.0 exact | ✓ |
| Latest-of-major floating | Reproducibility risk; contradicts goswitch model | |

**User's choice:** *(auto)* Exact mise pins
**Notes:** go.mod mirrors the mise Go version.

---

## Lint config

| Option | Description | Selected |
|--------|-------------|----------|
| Copy goswitch .golangci.yml, change only goimports.local-prefixes | Recognizably the sibling repo; zero linter churn | ✓ |
| Author a custom config | Violates the "copy goswitch" mandate | |

**User's choice:** *(auto)* Copy goswitch
**Notes:** any violation fails the lint target.

---

## Workflow set

| Option | Description | Selected |
|--------|-------------|----------|
| All four goswitch shapes now (pr-sanity, release, e2e-matrix, security-scheduled) | Inert until triggered; faithful scaffold | ✓ |
| pr-sanity only, others later | Less faithful; release/e2e shapes cost nothing inert | |

**User's choice:** *(auto)* All four
**Notes:** actions pinned by digest or exact tag.

---

## Secret-scan in CI

| Option | Description | Selected |
|--------|-------------|----------|
| Fix open REVIEW findings first (WR-01/IN-03 + batched WR-02/03/04, IN-01/02/04), then wire scan-secrets into pr-sanity | CI inherits a correct gate; TDD mode honoured on scrubber fixes | ✓ |
| Wire as-is, fix later | CI would embed the known-wrong regex class | |

**User's choice:** *(auto)* Fix first, then wire
**Notes:** secret-scan becomes a required check name for protection.

---

## Module layout

| Option | Description | Selected |
|--------|-------------|----------|
| github.com/Djarvur/zcode-acp + cmd/zcode-acp skeleton (--version 0.0.0-dev) | Standard; internal/ arrives with Phases 3–4 | ✓ |
| Different module path | Nothing suggests it | |

**User's choice:** *(auto)* Standard layout
**Notes:** README stub documents protection + CI (SCAFF-03 "documented in the repo").

---

## Claude's Discretion

- Workflow YAML details within goswitch shapes; runner versions; caching
- Wave-internal ordering of gate wiring vs skeleton
- Config-flip sequencing relative to protection apply

## Deferred Ideas

- Release binaries (Phase 7); VM e2e content (Phase 7); Renovate/dependabot (PR-driven)
