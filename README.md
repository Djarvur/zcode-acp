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

## Connecting to preset-agent-only hosts (launch recipe)

Orca's omp card is a preset agent: it is launched as `<program> omp ...`, and
a host that only accepts preset agents wires it up through a per-agent
command override. The whole integration is ONE override string plus ONE env
mechanism, per auth path — nothing else to configure.

### 1. The per-agent command override

In the client's Settings, set the per-agent command override for the `omp`
agent to (adjust the path to where you put the binary):

```
/path/to/zcode-acp omp
```

Mechanics worth knowing before you type it:

- The string is shell-quoted-tokenized by the client: the first token is the
  program, the rest are prefix arguments prepended before the card's own
  arguments. `~/`-prefixed tokens are expanded against your home directory.
- The tokenizer REJECTS env-assignment prefixes — `VAR=val cmd` is invalid,
  as are `$`, backticks and shell metacharacters — so an API key can never
  ride the override string. That rejection is why the env mechanism in step 3
  exists.
- A non-empty override satisfies the client's installed-detection check: no
  binary named `omp` needs to exist in PATH.
- The override is configured through the Settings UI; where the client
  persists `agentCmdOverrides` underneath is a labeled known-unknown (the
  manual QA-03 run confirms it) — do not rely on hand-editing a settings file.

### 2. The version floor (check before wiring)

The client probes the override as `<program> <prefix args> --version` — for
this recipe that is `zcode-acp omp --version` — with a short timeout, and
requires a semver >= 18.1.9 for the card's model selectors to resolve
(>= 17.0.5 to install the card at all). This wrapper prints
`zcode-acp 18.1.9+zacp.0` — the honest build-metadata form over the
compatibility floor. Run the check yourself first:

```
zcode-acp omp --version
```

### 3. Environment, per auth path

**API key path** — the wrapper needs `ZCODE_ACP_API_KEY` visible in its
process environment, and the override string cannot carry it (the rejection
rule above). Use ONE of these two mechanisms:

- an `agentDefaultEnv` entry in the client's settings — a sibling of the
  command-override setting — with the env name `ZCODE_ACP_API_KEY` and the
  value `<your-key>` (your real key goes only into your own client settings,
  never into any doc, script, or the override string); or
- a two-line PATH shim that the override points at instead of the binary.
  The export names the env var; the VALUE comes from a file only you own:

  ```sh
  #!/bin/sh
  # ~/bin/zcode-acp-omp — PATH shim. Never embed a key here; the sourced
  # file is yours and holds the real value: export ZCODE_ACP_API_KEY=<your-key>
  . "$HOME"/.config/zcode-acp/env
  exec /path/to/zcode-acp "$@"
  ```

  with the override set to `~/bin/zcode-acp-omp omp` — the `"$@"` already
  carries the `omp` prefix argument and everything the client appends after
  it.

**Coding-plan path** — NO env at all: the desktop login is the credential.
Requirements: the ZCode desktop installed and logged in to the individual
coding plan, and an unlocked session keyring. A missing/locked keyring
surfaces as the four coded sentinels from Troubleshooting — actionable
messages, never a hang.

### 4. Models: launch binding and live switching

- A model picked in the client UI before launch rides `--model
  providerId/modelId` and binds at session creation (launchesWithModel:true).
  A reasoning level may accompany it via `--reasoning-level` or
  `ZCODE_ACP_REASONING_LEVEL` (it requires `--model`; the level defaults to
  the backend-confirmed shape when omitted).
- `/model <name>` typed in the chat switches models live (a slash prompt the
  wrapper resolves server-side). A refused switch renders the card's own
  strings, byte-exact: `Unknown ACP model: <name>` for an unlisted model and
  `No model selected.` when nothing is selected — carried as ACP error code
  -32603 with the text in the error details, exactly the carriage the card's
  matchers read.

### 5. Client-set environment

The client sets `PI_NO_PTY=1` for the omp card; this wrapper ignores it —
ACP is a pipe protocol, no PTY is ever allocated. The card's agent-home
variable (`PI_CODING_AGENT_DIR`) is client bookkeeping and does not affect
the wrapper.

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
