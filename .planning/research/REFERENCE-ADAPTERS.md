# Reference Adapters & Host Detection Intel

**Compiled:** 2026-11-19 (updated same day with resolved Orca integration decisions)
**Provenance:** Reported by a parallel source-reading research session (reads of
github.com/william0wang/zcode-acp sources and an unpacked Orca.AppImage).
Reference-adapter claims carry a **V-RPT** label — reported-and-plausible, not
yet independently re-verified in this repo. The Orca integration section below
is upgraded to **V-SRC(parallel)** — read in Orca source chunks, not yet
exercised live. Phase 1 (DISC-01/02/05) re-verifies both before relying on them
for implementation; until then treat as strong leads, not wire truth.

## william0wang/zcode-acp — primary reference implementation

TypeScript, npm `zcode-acp-server` (~0.88.x), **Apache-2.0** → porting code to
Go is allowed **with attribution** (keep NOTICE/license header in ported files).
~91★, very active — releases track ZCode desktop migrations. Bridges the
headless ZCode app-server to ACP; prefers `elicitation/create` when the client
supports it, falls back to `session/request_permission`.

### Verified-there implementations of OUR hardest parts (OAuth coding plan outside GUI)

| Piece | What they do | Key file(s) |
|-------|--------------|-------------|
| GNOME Secret Service decryption | `~/.zcode/v2/credentials.json` (`enc:v2:os:`) via `@napi-rs/keyring`, Linux store `secret-service` (D-Bus → GNOME Keyring); service `ZCode Credential Encryption`, account = `sha256(resolve(credentials.json))` — same formula as `secret-tool lookup`. Legacy `enc:v1:` fallback (`ZCODE_CREDENTIAL_SECRET` / machine string) | `src/backend/credential-store.ts` |
| `interaction/requestCredentialCipher` | Served in both paths (turn-loop and arrival-time), requestId-correlated reply | `src/handlers/server-requests.ts` |
| Account-provider snapshot | Push via `provider/updateAccountConfig` with exact revision `zcode-builtin:<rev>:<sha256(resolve(zcode-builtin.json))>` | `src/config/account-provider.ts` |
| PAT flow | `getCustomerInfo` → personal-project pick → `api_keys` (create if absent) → `access_tokens` (`clientType:"zcode"`) → answer `interaction/requestProviderRuntimeHeaders` with `requestAuth {apiKey: PAT, apiKeyId}` | `src/config/project-token.ts` |
| Tests for all of the above | `tests/credential-cipher.test.ts`, `tests/provider-runtime-headers.test.ts`, `tests/project-token.test.ts` | also `src/backend/client.ts` |

**Known limitation (their TROUBLESHOOTING):** GLM Coding Plan works headless;
**Start Plan does not** (needs browser/captcha). → Document in `docs/` (DOCS-03).

### Other existing adapters (for the record — all worse for our purpose)

| Adapter | Language | Why not |
|---------|----------|---------|
| jpalmev/jpalmae zcode-acp | Rust | Own OAuth only against legacy v1 store; does not read keyring; does not serve `requestCredentialCipher` |
| Liang-HZ zcode-acp | Python | Only v1 store; returns `enc:v2:os:` blobs undecrypted; macOS-centric |

Confirms the differentiator claim in FEATURES.md: nobody else does the
v2-keyring + coding-plan chain. (Exact repo owner slugs V-RPT.)

## Orca integration decisions (resolved 2026-11-19)

Source-verified on an unpacked Orca.AppImage by a parallel research session —
chunks `agent-session-attachment-store-registry` and
`structured-agent-session-host-assembly`. Upgraded from V-RPT leads to
**V-SRC(parallel)**: read in source, not yet exercised live; spot-check during
Phase 1 / first integration.

### Chosen preset-agent identity: the `omp` card

Reasons: the omp dialect ("Hb") is built entirely on string contracts — trivial
to implement in Go; the card's launch command is replaceable; the `--version`
probe is simple. Alternatives ruled out: **antigravity** (only card with
`ignoreAgentCommandSetting:true` — command cannot be replaced), **copilot**
(modelDiscovery `unavailable` — dead model picker), **grok** (heavy `x.ai/*`
dialect), **opencode/opencode2** (occupied by real opencode + vendored
child-session extensions), **kimi** (session-discovery; possible conflict).
Full ACP-card list in this build: omp, opencode, opencode2, kimi,
cursor (cursor-agent), grok, copilot, antigravity.

### Wiring: official `agentCmdOverrides` setting

Per-agent setting in Orca's settings store (`record<string,string>`; siblings:
`agentDefaultArgs` / `agentDefaultEnv` / `agentPermissionOverrides`). Replaces
the launch program (**prefix args allowed**; card arguments are always appended
by Orca). An override also satisfies the "agent installed" detection —
**no binary named `omp` is needed in PATH**. Decision (2026-11-19): single
binary `zcode-acp` with an `omp` subcommand, wired as
`agentCmdOverrides: { "omp": "zcode-acp omp" }` (string parsed with
shell-quoting, `~/` expanded, shell metacharacters rejected). CLI parsing:
**spf13/cobra v1.10.2** — `FParseErrWhitelist` (proxied into pflag's
`ParseErrorsAllowlist`) gives the required liberal unknown-flag handling.
Set via Settings UI (per-agent Command setting); the key was **not** found in
`~/.config/orca/*.json` — the store is Electron storage/orcad; verify at first
integration.

### Launch contract: `zcode-acp omp`

Orca runs: `zcode-acp omp acp --plugin-dir <vis> --model <picked> --session-dir <dir>`
with env `PI_NO_PTY=1` (ignore) and agent home `$PI_CODING_AGENT_DIR/.omp/agent`.

CLI implementation (cobra): `omp` subcommand accepts the card argv contract
(`acp` positional ignored, `--plugin-dir`, `--model`, `--session-dir` captured),
unknown flags tolerated via `FParseErrWhitelist`; `--version` answers at root
AND under any subcommand, any position — the probe shape under prefixed
overrides is unverified, so be liberal.

- `--version` → semver on stdout (5 s probe); card floor **17.0.5**, but print
  **18.1.9+** — model-selectors resolve from that version
- argv: `acp` positional (ignore), `--plugin-dir <vis>` (visualsSkill — ignore
  or read), **`--model <id>` — USE**: `launchesWithModel:true`; it is the model
  picked in Orca's UI, applied at session start
- Hb dialect string contracts: model switching via `/model <name>` slash prompt;
  exact errors `Unknown ACP model: …` and `No model selected.`; sessionLost =
  `ACP cancel cleanup timed out` / `Unsupported ACP session: <id>`;
  `authenticationRequired` = `No API key found for …` (not applicable to us —
  auth lives in `~/.zcode`); Orca sets `retryBusyPrompts` for omp
- capabilities: client advertises `fs.readTextFile/writeTextFile: false`,
  `terminal: false` (never call client fs/terminal); offers `elicitation:
  {form}` — form-based questions available; support compaction +
  compactionInstructions
- no TUI needed: the ACP host spawns the subprocess with stdio pipes; Orca
  renders the whole UI

### Open questions → Phase 1 research

- **`--session-dir` format:** Orca pre-creates `<orcaData>/acp-command-sessions/omp`
  and passes it; the omp card has `sessionHistory` (WS) and `transcriptImport`
  (ZS, fork:'tip') — Orca scans that directory for history/resume. Determine the
  omp/pi session-file format and whether our Go agent can write compatible
  files (study stablyai/orca: `session-scanner-omp-subagent-transcripts`,
  `transcriptImport` in `structured-agent-session-host-assembly`; plus a real
  omp session directory).
- **modelDiscovery** for omp is kind `session`,
  `listingNamesConfiguredModel:false` — models are listed inside the session
  via the `/model` mechanics; clarify in a live test.

## Impact on plan (updated 2026-11-19)

- Phase 1 scope shifts from *discover* to *verify + document*: persona is
  chosen (omp), launch line, env, `agentCmdOverrides` wiring, version-probe
  mechanics and Hb string contracts are compiled (DISC-02); the two open
  questions above are the remaining discovery items.
- Phase 6 gets a porting source: credential-store / account-provider /
  project-token / server-requests under Apache-2.0 with attribution (AUTH-02),
  plus its tests as golden references.
- MODEL-01/02: `--version` reports ≥ 18.1.9 (model-selectors); `--model <id>`
  at launch and the `/model` slash contract are IN scope via SESS-03; generic
  slash commands remain v2 (EXT-01).
- DOCS-02: the README recipe now names `agentCmdOverrides` wiring explicitly.
- DOCS-03: document the Start-Plan-headless limitation.

---
*Intelligence fold-in after roadmap creation; sources as listed.*
