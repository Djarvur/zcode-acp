# zcode-acp

A single static Go binary that exposes ZCode Desktop to the outside world as an
honest **ACP (Agent Client Protocol) v1 server over stdio**: any ACP client can
open ZCode sessions, send prompts, and receive streamed replies (messages,
thinking, tool calls, permission requests). The ZCode desktop is the single
backend — this wrapper never patches it, never forges an attestation or
signature, and never reports anything it cannot know (the honesty contract is
written down in [docs/provenance.md](docs/provenance.md)). This README is the
user surface: install, configuration for both authentication paths,
troubleshooting, and the launch recipe for preset-agent-only hosts. The CI and
branch-protection sections at the bottom document this repository's own
engineering surface.

## Install

Build from source with a Go toolchain pinned via
[mise](https://mise.jdx.dev) (`mise.toml` carries the exact versions CI runs):

```sh
go build -o zcode-acp ./cmd/zcode-acp
```

(`mise run build` is the plain compile check.) The result is a single static
binary with no runtime dependencies of its own. Alternatively, fetch a release
artifact and verify its checksum against the publisher's sidecar.

On the target machine the binary needs:

- a ZCode desktop install — the wrapper discovers it at startup
  (env/flag override → desktop entry → AppImage paths → package paths;
  `--app-server` overrides the search), and
- `secret-tool` (the `libsecret-tools` package) plus an unlocked
  gnome-keyring **only for the coding-plan auth path** — the api-key path
  needs neither.

Check the build: `./zcode-acp --version` prints `zcode-acp 18.1.9+zacp.0`.

## Configuration

| Flag | Env fallback | Meaning |
|------|--------------|---------|
| `--api-key` | `ZCODE_ACP_API_KEY` | z.ai API key for the `authenticate{zai-api-key}` path |
| `--app-server` | `ZCODE_ACP_APP_SERVER` | path to a ZCode install root (or app-server binary) overriding discovery |
| `--model` | `ZCODE_ACP_MODEL` | backend model selection as `providerId/modelId` |
| `--reasoning-level` | `ZCODE_ACP_REASONING_LEVEL` | reasoning level accompanying `--model` |
| `--version` | — | print the version and exit |

The flag always wins over its env var. A `--reasoning-level` without a
`--model` is a configuration error; malformed input exits with code 2, runtime
failures exit 1. Every log line goes to stderr — stdout carries protocol
frames only.

### Authentication: two paths

**API key** — set `ZCODE_ACP_API_KEY=<your-key>` in the environment the
wrapper runs in (or pass `--api-key`). The value lives in your own
environment; this project's documentation never instructs exporting or pasting
key material anywhere else. When no key is configured,
`authenticate{zai-api-key}` answers an honest coded error naming
`ZCODE_ACP_API_KEY` — the wrapper never falls back to reading the desktop's
own config store.

**Coding plan** — the desktop login IS the credential: nothing to export, no
env var. Requirements: the ZCode desktop installed and logged in to the
individual coding plan, and an unlocked session keyring. The account's model
catalog is read back from the account providers at session creation; the
wrapper keeps the app-server's private dialect off the ACP wire.

## Troubleshooting

**Keyring sentinels** (the coding-plan path). These four coded messages are
the expected failure text — the probe never hangs, and each message names the
fix:

| Message | Meaning |
|---------|---------|
| `keyring probe: secret-tool not found (install libsecret-tools on Ubuntu: sudo apt install libsecret-tools)` | the libsecret CLI is missing |
| `keyring probe: no desktop session available (the keyring bus cannot autolaunch headless — run inside your desktop session)` | the wrapper runs outside a desktop session; the keyring bus cannot autolaunch |
| `keyring probe: no ZCode desktop login found in the keyring (log in via the ZCode desktop first)` | no matching ZCode credential in the keyring — log in via the desktop first |
| `keyring probe: keyring lookup did not answer within its bound` | the bounded lookup (5 s ceiling) outlived its bound |

**Version floor** — a preset-agent card is probed with `<program> --version`
and requires semver ≥ 18.1.9 for model selectors to resolve (≥ 17.0.5 to
install at all). An older stamp blocks the card before any protocol flows.
This binary prints `18.1.9+zacp.0`; check with `zcode-acp omp --version`.

**Close-timeout toast** — the wrapper's close budget and the client's
`session/close` timeout are both 10 s; a turn ending exactly at the boundary
can produce an intermittent close toast on Stop. The client tolerates it and
the session is still torn down — no action needed.

**Start Plan does not work headless** — Start Plan providers authenticate
with a browser/captcha step only the desktop app can provide (the GLM Coding
Plan works headless). See
[docs/provenance.md](docs/provenance.md) for the full known-limits list.

**Reproduce on a clean machine** — `scripts/vm-e2e.sh` walks the whole v1
scenario on a fresh Ubuntu GNOME VM: desktop discovery, the version floor,
both auth stages, and the no-keyring error path asserted against the real
binary. The wire-level protocol notes live under
[docs/protocol/](docs/protocol/).

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
