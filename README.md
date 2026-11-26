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

**Platform support.** Linux, macOS, and Windows are supported by this tree.
Linux is the reference platform (v1 targets Ubuntu GNOME); macOS uses the
installed `ZCode.app`; Windows uses the installed desktop tree. Windows support
means: the process-management layer uses Job
Objects, the coding-plan credential key is read from Windows Credential
Manager (the same store the ZCode desktop writes), and `GOOS=windows go build
./...` passes. Windows has no AppImage mount, so the wrapper runs the installed
desktop tree it discovers natively — the per-user install under
`%LOCALAPPDATA%\Programs\ZCode` first, then the per-machine
`%ProgramFiles%\ZCode`, then the uninstall-registry records under
HKCU/HKLM. `--app-server` / `ZCODE_ACP_APP_SERVER` remains an
optional override:

```powershell
.\zcode-acp.exe --app-server "$env:LOCALAPPDATA\Programs\ZCode\ZCode.exe"
```

Discovery validates the same layout-invariant tree on every platform
(`ZCode.exe` + `resources\glm\zcode.cjs`) before it selects an install,
and a machine with none of those sources still fails loud with one
diagnostic row per source. The coding-plan path needs an interactive
user logon session: Windows Credential Manager reads fail from a network logon
(for example an SSH session) with `ERROR_NO_SUCH_LOGON_SESSION` — run the
wrapper from the desktop session, as the ZCode desktop itself does. The
unattended-VM scaffolding used to validate the port lives in
[`scripts/windows-vm/`](scripts/windows-vm/): the Autounattend answer file, the
offline Win32-OpenSSH tools ISO builder, and the real-backend
`smoke-server.ps1`.

On macOS the wrapper discovers `/Applications/ZCode.app` and
`$HOME/Applications/ZCode.app`, validates `Contents/Info.plist`, and reads the
bundle version. It launches `Contents/Resources/glm/zcode.cjs` with
`ELECTRON_RUN_AS_NODE=1`; macOS never enters the AppImage/FUSE path. The
desktop architecture must match the wrapper: `darwin-aarch64` on Apple
silicon, `darwin-x86_64` on Intel. Discovery selects the matching
runtime-provider cache under
`~/.zcode/v2/runtime/provider/<platform>/<version>/`.

To keep the Dock clean, macOS launches the app-server under the host's `node`
from the login-shell PATH (Node 22+; tested with 26) instead of the Electron
binary: Electron registers its app-server and every bundled MCP child as a
foreground LaunchServices app, producing "ZCode"/"exec" Dock icons. The same
`zcode.cjs` runs correctly under Node, and its MCP children stay
`BackgroundOnly`. If Node is absent, the wrapper falls back to the Electron
launcher (still functional, but Dock icons can appear);
`ZCODE_ACP_DISABLE_NODE_LAUNCHER=1` forces that fallback explicitly.

Build from source with a Go toolchain pinned via
[mise](https://mise.jdx.dev) (`mise.toml` carries the exact versions CI runs):

```sh
go build -o zcode-acp ./cmd/zcode-acp
```

(`mise run build` is the plain compile check.) The wrapper itself is a single
static binary with no runtime dependencies of its own; the backend it drives is
the ZCode desktop install. Alternatively, fetch a release artifact and verify
its SHA-256 checksum against the published `checksums.txt`:

- `zcode-acp_<version>_linux_amd64.tar.gz` / `..._linux_arm64.tar.gz`
- `zcode-acp_<version>_darwin_amd64.tar.gz` / `..._darwin_arm64.tar.gz`
- `zcode-acp_<version>_windows_amd64.zip`
- `zcode-acp_<version>_darwin_universal.tar.gz` (one binary for both Mac
  architectures)
- `checksums.txt` (SHA-256 for every archive above)

On the target machine the binary needs:

- a ZCode desktop install — the wrapper discovers it at startup on Linux
  (env/flag override → desktop entry → version-ranked AppImage paths → package
  paths), on Windows (env/flag override → `%LOCALAPPDATA%\Programs\ZCode` →
  `%ProgramFiles%\ZCode` → uninstall-registry entries), and on macOS
  (`/Applications` → `$HOME/Applications`). Each candidate is validated by the
  invariant tree. `--app-server`/`ZCODE_ACP_APP_SERVER` overrides the search
  on every platform, and
- on Linux, `secret-tool` (the `libsecret-tools` package) plus an unlocked
  gnome-keyring; on Windows, Windows Credential Manager; on macOS, the login
  Keychain via `/usr/bin/security` — all three **only for the coding-plan auth
  path**. The api-key path needs none of them.

**Packed-AppImage backend (Linux only).** When the selected install is a packed `.AppImage`,
the wrapper mounts it read-only at startup through the AppImage's own
`--appimage-mount` runtime and runs the app-server from that mount. It creates
no extraction copy: no `squashfs-root`/`zcode-<ver>` directory is added, and
pre-existing extracts (such as a research `squashfs-root`) are left untouched.
There is no extraction fallback — a host that cannot mount fails loud with the
AppImage path, the failing step, the observed output, and a remediation. The
packed-AppImage backend therefore needs host prerequisites the
unpacked/deb/rpm backends do not:

- a working FUSE setup — `/dev/fuse` present and unprivileged mounts allowed;
  and
- for the legacy AppImage runtime, the libfuse2 library — `libfuse2t64` on
  Ubuntu 24.04 (`sudo apt install libfuse2t64`) or `libfuse2` on older
  releases (`sudo apt install libfuse2`). Newer static type-2 AppImage runtimes
  carry their own FUSE client and do not need libfuse2.

Check the build: a source build's `./zcode-acp --version` prints
`zcode-acp 18.1.9+zacp.0`. Release artifacts stamp their tag into the build
metadata instead — v1.0.0 prints `zcode-acp 18.1.9+zacp.1.0.0` — so the
Orca compatibility floor (18.1.9) stays satisfied.

## Configuration

| Flag | Env fallback | Meaning |
|------|--------------|---------|
| `--api-key` | `ZCODE_ACP_API_KEY` | z.ai API key for the `authenticate{zai-api-key}` path — prefer the env var: the flag's value rides the process argv (world-readable via `ps`/`/proc` for the wrapper's lifetime) |
| `--app-server` | `ZCODE_ACP_APP_SERVER` | path to a ZCode install root, app-server binary, packed AppImage, macOS `.app`, its `Contents` directory, or its `Contents/MacOS/<executable>` overriding discovery |
| `--model` | `ZCODE_ACP_MODEL` | backend model selection as `providerId/modelId` |
| `--reasoning-level` | `ZCODE_ACP_REASONING_LEVEL` | reasoning level accompanying `--model` |
| `--version` | — | print the version and exit |

The flag always wins over its env var, and the override wins over on-disk
discovery (precedence unchanged); a packed `.AppImage` override routes through
the same managed mount as a discovered packed winner. A `--reasoning-level`
without a `--model` is a configuration error; malformed input exits with code
2, runtime failures exit 1. Every log line goes to stderr — stdout carries
protocol frames only.

### Authentication: two paths

**API key** — set `ZCODE_ACP_API_KEY=<your-key>` in the environment the
wrapper runs in; the env var is the recommended form (a process environment
is owner-only readable). The `--api-key` flag keeps working for
compatibility, but its value rides the process argv, which any local user on
the host can read (`ps aux`, `/proc/<pid>/cmdline`) for the wrapper's whole
lifetime — the wrapper logs a one-line warning to stderr when the flag form
is used. The value lives in your own environment; this project's
documentation never instructs exporting or pasting key material anywhere
else. When no key is configured, `authenticate{zai-api-key}` answers an
honest coded error naming `ZCODE_ACP_API_KEY` — the wrapper never falls
back to reading the desktop's own config store.

**Coding plan** — the desktop login IS the credential: nothing to export, no
env var. Requirements: the ZCode desktop installed and logged in to the
individual coding plan, and an unlocked keyring — gnome-keyring on Linux,
Windows Credential Manager on Windows, or the login Keychain on macOS. The
account's model catalog is read back from the account providers at session
creation; the wrapper keeps the app-server's private dialect off the ACP wire.

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

The four arms above are the Linux `secret-tool` probe. On Windows the cached
arms are `keyring probe: Windows Credential Manager is unavailable` (a network
logon or a locked store) and the shared `no ZCode desktop login found` /
`no Windows Credential Manager entry for this credentials file`; the fix for
the latter is to log in via the installed ZCode desktop (or run the wrapper in
the interactive desktop session where that login lives).

On macOS the Keychain arms are:
`keychain probe: /usr/bin/security not found (macOS system security tool
missing)`, `keychain probe: no ZCode desktop credential found in the macOS
Keychain (log in to the ZCode desktop and retry)`, `keychain probe: the macOS
Keychain is locked or cannot prompt (unlock the login keychain and retry)`, and
`keychain probe: keychain lookup did not answer within its bound`. The first
successful read may require a one-time macOS "allow access" dialog; approve it
while the login Keychain is unlocked.

**Custom model returns 400 “maximum context length”** — a user-added
OpenAI-compatible model needs an output-token cap in
`~/.zcode/v2/provider_config.json`; without it the backend may request almost
the whole context window as completion tokens. In the model's
`modelConfigRules.providerModelRules` entry add:

```json
"config": {
  "properties": { "contextWindow": 262144 },
  "optionSpecs": { "maxOutputTokens": { "max": 32768 } }
}
```

Use a value comfortably below the model's context window; 32768 is a safe
starting point for chat and tool use.

**Provider quota / limit refusal** — ACP v1 has no failure `stopReason`, so a
provider 429/quota refusal surfaces as `-32603` with
`session/prompt: backend turn failed before completion (the model provider
refused the turn; check its quota/limits and retry)`. The exact reset time is
in ZCode's own provider response and logs; the observed 4.0.0 `turn-failed`
event carries no provider detail.

**FUSE / packed-AppImage mount** (the packed-AppImage backend). The AppImage
runtime reports these exact messages; they may arrive on stdout after a leading
empty line, so the wrapper classifies the observed text rather than trusting
the first line:

| Message | Meaning |
|---------|---------|
| `AppImages require FUSE to run.` | FUSE is not available to the runtime — check `/dev/fuse` and that unprivileged mounts are allowed |
| `Cannot mount AppImage, please check your FUSE setup.` | the mount failed after the runtime started — same FUSE remediation family |
| `dlopen(): error loading libfuse.so.2` | the legacy runtime's libfuse2 library is missing — install `libfuse2t64` on Ubuntu 24.04 or `libfuse2` on older releases |

Remediation family: ensure `/dev/fuse` exists (load the `fuse` module) and
unprivileged mounts are allowed, then install the libfuse2 package when the
runtime needs it (`sudo apt install libfuse2t64` on Ubuntu 24.04,
`sudo apt install libfuse2` on older releases). A mount failure exits non-zero
with the AppImage path and the failing step (`read the mount line`, `validate
the mount tree`, …) rather than falling back to extraction — there is no
extraction fallback by design. A mount that disappears mid-session ends the
session with an explicit cause (`zcode: appimage mount lost: <appimage> (mount
<root>)`) instead of continuing against a vanished root.

**Version floor** — a preset-agent card is probed with `<program> --version`
and requires semver ≥ 18.1.9 for model selectors to resolve (≥ 17.0.5 to
install at all). An older stamp blocks the card before any protocol flows.
Source builds print `18.1.9+zacp.0`; release artifacts print
`18.1.9+zacp.<release>` (v1.0.0 → `18.1.9+zacp.1.0.0`). Check with
`zcode-acp omp --version`.
macOS limitations in this release: Start Plan is not part of the macOS port;
the wrapper ships as a standalone binary (no `.app` packaging, signing, or
notarization); AppImage/FUSE and deb/rpm discovery are Linux-only.

**Close-timeout toast** — the wrapper's close budget and the client's
`session/close` timeout are both 10 s; a turn ending exactly at the boundary
can produce an intermittent close toast on Stop. The client tolerates it and
the session is still torn down — no action needed.

**Empty chat or branch names from the omp card** — the client's own text
generators (chat name, branch name, commit message) launch the override as
`omp --print --mode text @file` and read the generated text from stdout. A
build that predates that contract answers exit 0 with empty stdout, which the
client reports as `returned an empty chat name`. Check `zcode-acp omp
--version` (≥ 18.1.9) and see [Non-interactive text
tasks](#8-non-interactive-text-tasks).

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

**Session restoration.** Orca checks top-level `loadSession` first and then
`sessionCapabilities.resume`. This wrapper advertises `sessionCapabilities.options:{}`
and `sessionCapabilities.resume:{}`, but not `loadSession`, so Orca resumes the
existing backend session with the same session id and the wrapper re-subscribes.
Restoration creates no replacement session, performs no model rebind, and replays
no history; `session/load` and history import remain unsupported.

### 1. The per-agent command override

In the client's Settings, set the per-agent command override for the `omp`
agent to (adjust the path to where you put the binary):

```
/path/to/zcode-acp omp
```

**macOS.** The same override works with a binary built on macOS. Put it
somewhere stable (for example `~/bin/zcode-acp`), install ZCode Desktop in
`/Applications`, and leave `--app-server` unset — discovery selects
`/Applications/ZCode.app` automatically. If Orca was started from the GUI, the
wrapper unions the login-shell `PATH` (mise, Homebrew, Go, cargo, orbstack, and
the bundled ZCode tools) into the backend environment, so tools resolve even
though launchd's base PATH is minimal. Only `PATH` is unioned; proxy and secret
variables remain scrubbed.

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
(>= 17.0.5 to install the card at all). Source builds print
`zcode-acp 18.1.9+zacp.0`; release artifacts stamp the tag into the build
metadata (`zcode-acp 18.1.9+zacp.1.0.0` for v1.0.0) — the honest form over
the compatibility floor. Run the check yourself first:

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

### 5. The slash-command autocomplete

The card's composer menu lists the wrapper's catalog: every ZCode command
(`~/.zcode/commands/*.md` + the `~/.agents/commands` compat root — e.g. the
whole gsd-* set with their argument hints), every skill
(`~/.zcode/skills/**/SKILL.md`, the `~/.agents/skills` compat root, and the
installed plugin caches — name/description from the SKILL.md frontmatter),
and the built-in `/model`. The catalog rides one `available_commands_update`
per session (the client's own discovery scan consumes it — nothing to
configure); workspace-level command/skill dirs (`<cwd>/.zcode[/agents]`)
join per session when present. Picking an entry sends its `/name` as the
prompt text — exactly the form the runtime already executes.

### 6. Ask-the-user dialogs

Workflow questions (an AskUserQuestion-style ask) render as a form dialog in the
chat: the question, the labeled options with their descriptions, and Skip. Your
answer feeds the workflow (`content.answers` keyed by the question text); declining
or skipping resolves the ask as declined. No configuration — the bridge is always on
when the client declares form elicitation support.

### 7. Client-set environment

The client sets `PI_NO_PTY=1` for the omp card; this wrapper ignores it —
ACP is a pipe protocol, no PTY is ever allocated. The card's agent-home
variable (`PI_CODING_AGENT_DIR`) is client bookkeeping and does not affect
the wrapper.

### 8. Non-interactive text tasks

The client's own generators — chat name, branch name, commit message —
launch the override as `omp --print --no-session --no-tools
--no-extensions --no-skills --no-rules --mode text [--model p/m]
[--thinking level] @file`. This build answers those invocations with one
one-shot model call and prints only the generated text on stdout: exit 0
when text was produced, 1 on a runtime or authentication failure, 2 on a
usage error. Stderr carries diagnostics; stdout carries only the text.

## CI

Four GitHub Actions workflows run on this repository (shapes shared with the
owner's reference repo, adapted to this project):

| Workflow | Triggers | What it runs |
|----------|----------|--------------|
| `pr-sanity` | push to `master`, any pull request, manual dispatch | Four jobs: **`build / vet / lint / test / tidy`** (go build, go vet, strict pinned golangci-lint, go test with the race detector, the experiments `node --test` suite, and a go.mod/go.sum tidy-diff check), **`govulncheck`** (reachable-vulnerability scan), **`secret-scan`** (secret-shape scan over the whole tracked tree, plus the docs conformance gate), **`build / vet / test (windows)`** (the same go build, go vet, and race-enabled Go suite on a native `windows-latest` runner, with Go installed from the same `mise.toml` pin) |
| `release` | `v*` tags only | goreleaser build + publish; inert until the release flow lands (no `.goreleaser.yaml` yet) |
| `e2e-matrix` | manual dispatch only | Placeholder skeleton for the future self-hosted GNOME desktop e2e runner; inert until a runner is registered (Phase 7) |
| `security-scheduled` | weekly schedule (Mondays 06:00 UTC) + manual dispatch | govulncheck over the module |

Tool versions come from `mise.toml` only. The Linux job installs the full tool
set with `mise install`; the Windows job installs only the Go pin with
`mise install go`. Both run the exact mise tasks the local gate loop uses, and
no workflow pins a second copy of any tool version.

The Windows job checks out `testdata/` at LF through `.gitattributes`, so the
byte-exact golden fixtures are identical on both runners.

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

The `build / vet / test (windows)` job is not part of the protection body: it
reports on every pull request, so the required set stays exactly these three
names.

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
