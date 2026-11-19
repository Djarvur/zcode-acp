# Reference Adapters & Host Detection Intel

**Compiled:** 2026-11-19
**Provenance:** Reported by a parallel source-reading research session (reads of
github.com/william0wang/zcode-acp sources and an unpacked Orca.AppImage). These
claims carry a **V-RPT** label — reported-and-plausible, not yet independently
re-verified in this repo. Phase 1 (DISC-01/02/05) must re-verify each before
relying on it; until then treat as strong leads, not wire truth.

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

## Orca preset-agent detection mechanics (chunk `agent-session-attachment-store-registry`)

All V-RPT, from reading an unpacked Orca.AppImage:

- **Version probe:** Orca runs `<command> --version`, parses semver from
  stdout/stderr, **5 s timeout**. The wrapper binary must answer `--version`
  quickly with a parseable semver ≥ the card's floor.
- **`omp` card:** requires **≥ 17.0.5**; **model-selectors** resolve from
  **≥ 18.1.9** (i.e. `--version` should report ≥ 18.1.9 if we want native model
  pickers in the card).
- **Launch line:** `omp acp [--plugin-dir] [--model]` with env `PI_NO_PTY=1`;
  sessions pass `--session-dir`.
- **omp dialect over ACP v1** includes a `/model <name>` **slash command** with
  string contracts (exact strings like `Unknown ACP model: `, `No model
  selected.`). Relevant to SESS-03 (full dialect coverage) and MODEL-01/02: the
  chosen persona may need `available_commands_update` + `/model` handling even
  though generic slash commands are EXT-01/v2.

## Impact on plan (absorbed 2026-11-19)

- Phase 1 scope shifts from *discover* to *verify + document* for: persona
  (omp), version floor, launch line, env, version-probe mechanics (DISC-02).
- Phase 6 gets a porting source: credential-store / account-provider /
  project-token / server-requests under Apache-2.0 with attribution (AUTH-02),
  plus its tests as golden references.
- MODEL-01/02: aim the reported `--version` value at ≥ 18.1.9 for
  model-selectors; decide `/model` handling during SESS-03.
- DOCS-03: document the Start-Plan-headless limitation.

---
*Intelligence fold-in after roadmap creation; sources as listed.*
