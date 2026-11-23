# Phase 2: Scaffold & CI Bootstrap - Pattern Map

**Mapped:** 2026-11-19
**Files analyzed:** 17 (8 new, 9 modified)
**Analogs found:** 17 / 17 (external template: Djarvur/goswitch local clone; in-repo: Phase 1 scripts + docs)

All analog paths verified git-TRACKED: goswitch files via `git -C /home/nil/DiskD/W/Djarvur/goswitch ls-files`; zcode-acp files via `git -C /home/nil/DiskD/W/Djarvur/zcode-acp ls-files`.

**Analog root:** `/home/nil/DiskD/W/Djarvur/goswitch/` (below: `GS =` this root; `ZA =` `/home/nil/DiskD/W/Djarvur/zcode-acp/`). The planner reads analog files directly — pointers only, whole files not inlined here.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `mise.toml` (new) | config (toolchain pins + tasks) | transform | `GS:mise.toml:1-37` | exact |
| `.golangci.yml` (new) | config (lint gate) | transform | `GS:.golangci.yml:1-179` | exact |
| `go.mod` (new) | config (module decl) | transform | `GS:go.mod:1-3` | exact |
| `cmd/zcode-acp/main.go` (new) | entrypoint (CLI skeleton) | request-response | `GS:cmd/goswitchd/version.go:1-42` + `GS:cmd/skillgen/main.go:1-25` (house cmd style) | exact (verified green probe, RESEARCH Pattern 2) |
| `.github/workflows/pr-sanity.yml` (new) | CI workflow (CI gates) | batch | `GS:.github/workflows/pr-sanity.yml:1-93` | exact + additions (test-experiments, secret-scan job) |
| `.github/workflows/release.yml` (new) | CI workflow (release, inert) | event-driven (tags) | `GS:.github/workflows/release.yml:1-64` | exact |
| `.github/workflows/e2e-matrix.yml` (new) | CI workflow (skeleton, inert) | event-driven (dispatch) | `GS:.github/workflows/e2e-matrix.yml:44-77` (shape ONLY — not the 15KB AT-SPI body) | role-match (discipline copy, content fresh) |
| `.github/workflows/security-scheduled.yml` (new) | CI workflow (cron scan) | batch | `GS:.github/workflows/security-scheduled.yml:1-36` | exact |
| `README.md` (modify stub) | docs | transform | `ZA:docs/protocol/README.md:1-9,46-48` (header/footer style) | role-match |
| `protection-body.json` (new, phase dir) | config (API payload) | request-response | none in repo — RESEARCH Pattern 6 body is the verified source (live 422 probe) | no analog (use RESEARCH) |
| `.planning/config.json` (modify) | config (GSD branch flow) | transform | itself (`ZA:.planning/config.json:14`) | self (one-line flip) |
| `experiments/scan-secrets.sh` (modify) | gate script (bash) | transform | itself + `ZA:experiments/scrub-capture.mjs:48` (CRED_RX to align to) | self (edit) |
| `experiments/check-docs.sh` (modify) | gate script (bash) | transform | itself (`ZA:experiments/check-docs.sh:37-41`) | self (edit) |
| `experiments/scrub-capture.mjs` (modify) | utility (scrubber) | transform | itself (`ZA:experiments/scrub-capture.mjs:107-110,163`) | self (edit) |
| `experiments/scrub-capture.test.mjs` (modify) | test (node:test) | batch | itself (`ZA:experiments/scrub-capture.test.mjs:1-49` conventions) | self (extend) |
| `experiments/app-server-probe.mjs` (modify) | utility (probe harness) | request-response | itself (`ZA:experiments/app-server-probe.mjs:458-477`) | self (edit) |
| `docs/protocol/app-server-dialect.md` (modify) | docs | transform | itself (`ZA:docs/protocol/app-server-dialect.md:40-41,65,146`) | self (edit) |

## Pattern Assignments

### `mise.toml` (config)

**Analog:** `GS:mise.toml:1-37`. Copy structure: `[tools]` (lines 3-9), `[env]` (11-13: `GOFLAGS = "-trimpath"`), tasks `build`/`test`/`lint`/`vet`/`tidy-diff`/`ci` (15-37, each with `description` + `run`).

**Intended changes (D-02):** pins become `go = "1.27"`, `golangci-lint = "2.15.0"` (EXACT — goswitch floats `"2"` at line 5; this is the deliberate tightening), `goreleaser = "2.20.2"`, plus new `node = "26"`. Add one new task absent from goswitch: `[tasks.test-experiments]` with `run = "node --test experiments/"`, and add it to `[tasks.ci]` `depends`. Keep the header comment style (goswitch lines 1-2 explain D-09 single-source rationale — write the zcode-acp rationale, e.g. D-02).

### `.golangci.yml` (config)

**Analog:** `GS:.golangci.yml:1-179`. Copy byte-for-byte.

**ONE intended change (D-03)** — formatter prefix, lines 176-178:
```yaml
    goimports:
      local-prefixes:
        - "github.com/Djarvur/zcode-acp"   # was: github.com/Djarvur/goswitch (line 178)
```
Do NOT touch anything else: goswitch-specific exclusion rules (`internal/engine/conn.go` G404 at 138-140, `test/e2e/.*` rules at 150-164) are inert path patterns that match nothing here and stay per D-03's "changed ONLY" lock. Keep the header comment style (lines 1-3: config provenance note) — write the goswitch provenance.

### `go.mod` (config)

**Analog shape:** `GS:go.mod:1-3` (`module github.com/Djarvur/goswitch` + `go 1.23`). New file is exactly:
```
module github.com/Djarvur/zcode-acp

go 1.27
```
No `require` block yet (deps arrive Phases 3-4). The `go` directive must equal what mise.toml resolves (integration point from CONTEXT).

### `cmd/zcode-acp/main.go` (entrypoint, request-response)

**Analog:** `GS:cmd/goswitchd/version.go:1-42` — house version-reporting style: package-level `version` var with `//nolint:gochecknoglobals` stamping-contract comment (14-19), `versionString()` renderer (24-34), `runVersion(w io.Writer)` using the `_, _ = fmt.Fprintln(w, ...)` errcheck-quiet form (39-41). Imports/doc-comment style: `GS:cmd/skillgen/main.go:1-25` (multi-sentence `// Command <name> ...` doc comment before `package main`).

**Verified-green skeleton (RESEARCH Pattern 2 — the exact form that passes golangci-lint 2.15.0 `default: all`):**
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
**Load-bearing style rules (verified RED→GREEN in the research probe):**
- `fmt.Println` is FORBIDDEN (forbidigo default pattern `^(fmt\.Print(|f|ln)|print|println)$` under `default: all`); use `fmt.Fprintln` with explicit `_, _ =` discard (errcheck).
- nlreturn requires a blank line before every top-of-block `return` (see the skeleton's `return`).

### `.github/workflows/pr-sanity.yml` (CI workflow, batch)

**Analog:** `GS:.github/workflows/pr-sanity.yml:1-93`. Copy whole shape: header comment block (1-13), `on:` push/pull_request/workflow_dispatch (16-20), `permissions: contents: read` (23-24), concurrency group with event-name+ref (26-30), `sanity` job (33-67), `govulncheck` job (69-93).

**Intended changes:**
1. Line 18 `branches: [main]` → `[master]` (this repo's default branch; record the adaptation in the file header comment).
2. Add a `test (experiments suite)` step calling `mise run test-experiments` inside the sanity job (after the go test step).
3. Add a third job `secret-scan` (`name: secret-scan`, ubuntu-latest, checkout + `bash experiments/scan-secrets.sh` + `bash experiments/check-docs.sh` steps) — RESEARCH Code Example has the full adapted YAML.
4. Keep `actions/checkout@v4` and the mise.run installer flow verbatim (lines 37-50) — no setup-go, no golangci-lint-action (header comment lines 7-10 is the rationale to preserve).

**CRITICAL (Pitfall 3):** job `name:` fields ARE the protection required-check names. Must stay exactly: `build / vet / lint / test / tidy` (line 34), `govulncheck` (line 70), `secret-scan`.

### `.github/workflows/release.yml` (CI workflow, event-driven)

**Analog:** `GS:.github/workflows/release.yml:1-64`. Copy verbatim: `on.push.tags: ['v*']` (21-23), `permissions: contents: write` — the ONLY write in the set (26-27), serial `concurrency: release` cancel-in-progress false (29-33), `fetch-depth: 0` (42-44), mise install + `mise exec -- goreleaser release --clean` with `GITHUB_TOKEN` env (48-64). Preserve the registration-note comment (lines 17-18: workflow resolves from default branch, so tag triggers fire only after merge). Inert by construction: no `v*` tag exists and `.goreleaser.yaml` is not written (Phase 7). Intended change: translate/provenance the Russian header comments to this repo's decision ids (goswitch headers are Russian; zcode-acp repo docs language is English per PROJECT.md).

### `.github/workflows/e2e-matrix.yml` (CI workflow skeleton, event-driven)

**Analog (shape ONLY):** `GS:.github/workflows/e2e-matrix.yml:44-77` — `name: e2e-matrix` (44), `on: workflow_dispatch` with string/boolean inputs (46-56), `permissions: contents: read` (61-62), `concurrency: group: e2e-gnome; cancel-in-progress: false` (64-66), `runs-on: [self-hosted, gnome]` (71), generous `timeout-minutes` (77). Keep the discipline comments (33-40: fork events must never drive the self-hosted runner; one run at a time).

**DO NOT copy** lines 78-237 (idle-desktop AT-SPI preflight, matrix plumbing, artifact upload — they drive goswitch's desktop fixtures; none exist here). Skeleton body = checkout + a placeholder step (e.g. `run: echo "e2e scenario lands in Phase 7"`). Note goswitch's env-var pattern for event fields (line 94-95: `GITHUB_EVENT_NAME: ${{ github.event_name }}` via `env:`, never `${{ }}` inline in shell) — the security discipline to keep.

### `.github/workflows/security-scheduled.yml` (CI workflow, batch)

**Analog:** `GS:.github/workflows/security-scheduled.yml:1-36`. Copy verbatim: weekly cron `0 6 * * 1` (9-11) + workflow_dispatch (12), `permissions: contents: read` (15-16), single `govulncheck` job running `mise exec -- go run golang.org/x/vuln/cmd/govulncheck@latest ./...` (19-36). No repo-specific strings to change.

### `README.md` (docs, modify)

**Analog (style):** `ZA:docs/protocol/README.md:1-5` — header block format (`# Title` / `**Domain:** ...` / `**Researched:** date` / `**Confidence:** ...`) and `:46-48` footer (`---` / `*Phase: ...*` / `*Created: ...*`).

**Content** (SCAFF-03/D-06, outlined in RESEARCH): honest one-liner (existing stub `ZA:README.md` is 2 lines: `# zcode-acp` / `ACP wrapper for ZCode agent`), then a CI/branch-protection section documenting: required checks (the three exact names), PR-only flow, `gsd/phase-{phase}-{slug}` branch relation, and the honesty note that protection was created fresh (goswitch's own ruleset is disabled/misnamed — RESEARCH quotes it for this note). Neutral phrasing, no machine paths (Phase 1 discipline).

### `protection-body.json` (config) — NO ANALOG

**Source:** RESEARCH Pattern 6 (verified: live 422 probe proved the four required body keys; `checks[]` not deprecated `contexts[]`). Write the exact JSON body from RESEARCH Pattern 6 to `.planning/phases/02-scaffold-ci-bootstrap/protection-body.json`; required-check contexts must match the pr-sanity job `name:` fields character-for-character. Apply LAST via `gh api repos/Djarvur/zcode-acp/branches/master/protection -X PUT --input <body>`, verify by GET (RESEARCH Code Examples "Branch protection apply + verify").

### `.planning/config.json` (modify, self)

**Edit:** line 14 `"branching_strategy": "none"` → `"phase"` (uses existing `phase_branch_template` at line 17). One-line change; record in repo docs per D-01.

### `experiments/scan-secrets.sh` (modify, self + scrubber alignment)

**Analog for the fix:** `ZA:experiments/scrub-capture.mjs:48` — `const CRED_RX = /enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}/g;` and `:49` TOKEN_RX (no closing-quote requirement in the capture-side form).

**WR-01:** line 19 pattern `'enc:v[12]:(os:)?[A-Za-z0-9+/=_-]{20,}'` → add `.:` to the body class: `'enc:v[12]:(os:)?[A-Za-z0-9+/=_:.-]{20,}'` (align with CRED_RX verbatim). Update header comment lines 8-11 to match.
**IN-03:** after the shebang/comments, start with `cd "$(git rev-parse --show-toplevel)" || exit 1` (pin to repo toplevel; line 25's `git grep` is currently CWD-relative).
**Gate convention to preserve:** rc-accumulator + `SECRET SHAPE: <pattern>` output + `exit $rc` (lines 23-32). Verified green on current tree with widened patterns (RESEARCH WR-01).

### `experiments/check-docs.sh` (modify, self)

**IN-03 symmetry:** same one-liner `cd "$(git rev-parse --show-toplevel)" || exit 1` after line 37's `set -u` (before the docs/REQUIRED_SECTIONS block at 43-60). Optional-but-cheap; batched in the same touch per D-05. Preserve the `fail()` + rc contract (lines 40-41).

### `experiments/scrub-capture.mjs` (modify, self)

**WR-03 — home rewrite, lines 107-110.** Current:
```js
  if (home && home !== '/') {
    line = line.split(home + '/').join('~/');
    line = line.split(home).join('~');
  }
```
Fix: keep step 1 unconditional; make step 2 boundary-aware — `line.replace(new RegExp(escapeRegExp(home) + '(?!/?[A-Za-z0-9._-])', 'g'), '~')` with `escapeRegExp = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')`. Do NOT use the review hint's `(?!/[A-Za-z0-9._-])` — verified by execution it still falsifies `/home/ux` → `~x`.
**IN-04 — line 163:** `(i === 0 ? line : scrubLine(...))` currently skips validation. Add: line 1 must parse as JSON containing a `capture` key, else exit non-zero with a clear message; `--no-header` escape hatch flag (parse style follows the existing argv loop at 121-138). Preserve empty-input → empty-output (154-159).

### `experiments/scrub-capture.test.mjs` (modify, self — extend)

**Conventions to follow (lines 1-49):** forbidden literals built by STRING CONCATENATION (lines 5-8, fixtures 22-28) — new WR-03/IN-04 test inputs must not contain forbidden shapes verbatim; `runScrub(inputText, extraArgs)` spawnSync harness (34-49) for RED→GREEN cases; defensive `dataLine()` extractor. New tests: WR-03 both sibling cases (`/home/ux/otherproj/file.txt` AND `/home/ux`) plus truth-preserving cases and idempotency; IN-04 header-less input fails without flag, passes with it (RESEARCH Pattern 7 table).

### `experiments/app-server-probe.mjs` (modify, self)

**WR-04 — header constant, line 473:** `desktopVersion: '3.18.2'` → read at capture time, mirroring the in-file pattern at 458-465 (`builtinRevision` try/JSON.parse/`'unknown'` fallback). Recommended option (b): accept `--desktop-version` CLI arg (same argv parse style as the probe's existing flag loop), default `'unknown'`. No unit test required (D-05 attaches tests only to WR-03/IN-04).

### `docs/protocol/app-server-dialect.md` (modify, self)

Doc-recipe edits at the review's verbatim anchors (RESEARCH Pattern 7): line 146 (WR-02: record `grep -o -e '-32603'` recipe), lines 40-41 (IN-01: widen class to `'"[a-zA-Z0-9/]*"$'` and/or add v4 extractor one-liner), line 65 (IN-02: `singletons (8)` → `singletons (9)` — anchor verified: `| singletons (8) | bot/channel/reply, ...`). No gate impact: `check-docs.sh` pins section HEADINGS only (REQUIRED_SECTIONS at `check-docs.sh:54-60`), not row labels or inline commands. Keep the `[V-*]` provenance-tag grammar (legend at `docs/protocol/README.md:25-44`).

## Shared Patterns

### mise single-source-of-versions
**Source:** `GS:mise.toml:1-9` + `GS:pr-sanity.yml:39-50`
**Apply to:** mise.toml and ALL four workflows — CI never re-pins a tool version; every gate is `mise run <task>`; no setup-go, no golangci-lint-action (goswitch pr-sanity.yml:7-10 is the rejection rationale to carry into the header comment).

### Check-name = required-context discipline
**Source:** `GS:pr-sanity.yml:34,70` + RESEARCH Pattern 6
**Apply to:** pr-sanity.yml job `name:` fields AND protection-body.json contexts — copy `build / vet / lint / test / tidy`, `govulncheck`, `secret-scan` character-for-character (incl. ` / ` separators). Renaming one requires updating the other in the same change.

### Copied-file header provenance
**Source:** `GS:.golangci.yml:1-3` ("Adapted from the owner's reference config"), `GS:pr-sanity.yml:1-13`
**Apply to:** every copied artifact (`.golangci.yml`, four workflows, mise.toml) — an English header comment naming the goswitch origin and the exact intended deltas (master branch, node task, secret-scan job, exact linter pin).

### Minimal-permissions workflow hardening
**Source:** `GS:pr-sanity.yml:22-24`, `GS:release.yml:26-27` (only `contents: write`), `GS:e2e-matrix.yml:58-66`
**Apply to:** all four workflows — explicit `permissions:` block everywhere; `contents: write` confined to release.yml; event fields via `env:` not `${{ }}` in shell.

### Bash gate contract
**Source:** `ZA:experiments/scan-secrets.sh:23-32`, `ZA:experiments/check-docs.sh:37-41`
**Apply to:** both gate scripts' edits — `set -u` / rc accumulator / `fail()`-style labeled output / explicit `exit $rc`; toplevel pin via `cd "$(git rev-parse --show-toplevel)" || exit 1`.

### node:test conventions
**Source:** `ZA:experiments/scrub-capture.test.mjs:1-49`
**Apply to:** new RED→GREEN tests — forbidden literals via string concatenation, `runScrub` spawnSync harness, assertions (not parse errors) as the failure surface.

### Go strict-lint style (from the first Go file onward)
**Source:** `GS:cmd/goswitchd/version.go:39-41` + probe-verified skeleton
**Apply to:** `cmd/zcode-acp/main.go` — `fmt.Fprint*` family with `_, _ =` discard (never `fmt.Print*`), blank line before every top-of-block `return`, multi-sentence `// Command ...` doc comment (house style: `GS:cmd/skillgen/main.go:1-16`).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `protection-body.json` | config (API payload) | request-response | No branch-protection state exists to copy — goswitch's own is a disabled, misnamed ruleset (RESEARCH quotes it for the README honesty note). RESEARCH Pattern 6 body is the verified source (live 422 probe + docs). |
| `README.md` CI/protection section | docs | transform | No sibling README section exists (goswitch README documents a different, larger system); content outline comes from RESEARCH/CONTEXT, style from `docs/protocol/README.md` header block. |

## Metadata

**Analog search scope:** `/home/nil/DiskD/W/Djarvur/goswitch/` (mise.toml, .golangci.yml, .github/workflows/, go.mod, cmd/), zcode-acp repo (experiments/, docs/protocol/, .planning/config.json, README.md)
**Files scanned:** 15 analog files read (5 goswitch config/workflow + 2 goswitch cmd + 8 zcode-acp)
**Pattern extraction date:** 2026-11-19
