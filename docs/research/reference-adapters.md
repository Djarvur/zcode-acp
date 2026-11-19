# Reference Adapters — the TypeScript zcode-acp reference implementation

**Domain:** what the TypeScript reference implementation (william0wang/zcode-acp) covers for this project's hardest parts — the v2-keyring credential chain, host-managed interaction asks, and the coding-plan PAT minting — with adopt/avoid verdicts and the Apache-2.0 attribution obligations for the Phase 6 port (DISC-05)
**Verified against:** live shallow clones at tag **v0.89.0** (npm `zcode-acp-server` 0.89.0, release published 2026-11-19; default branch `main`), plus live GitHub-API checks of the two weaker adapters — 2026-11-19
**Confidence:** HIGH — every coverage row was read from the cloned source at the pinned tag; the license was live-verified via `gh api` (`Apache-2.0`); both weaker-adapter slugs and languages were live-confirmed and their key gaps grep-verified in fresh clones; only items never re-checked stay V-RPT-labeled

Provenance tags (V-SRC / V-EXP / V-RPT) are used per the legend in
[docs/protocol/README.md](../protocol/README.md). Upstream citations are
version-pinned to the release studied (tag v0.89.0); a `[V-SRC <path>#<anchor>]`
tag cites the cloned source tree of that tag, anchored on a function or constant
name. This document is the Phase 6 porting input: the OAuth chain (AUTH-02) is
implemented from the golden references named under Adopt. `.planning/research/
REFERENCE-ADAPTERS.md` was the research lead this doc verifies — where the lead
and the code disagreed, the code won and the drift is recorded row by row.

## What the reference covers

Capability map of the reference implementation, verified against the live repo at
tag v0.89.0. The "vs lead" column compares each row against
`.planning/research/REFERENCE-ADAPTERS.md` (confirmed / drift / beyond-lead).

| # | Capability | What v0.89.0 does | Provenance | vs lead |
|---|------------|-------------------|------------|---------|
| 1 | v2-keyring credential chain | Decrypts `~/.zcode/v2/credentials.json` (`enc:v2:os:`) via `@napi-rs/keyring`: service `ZCode Credential Encryption`, account = `hex(sha256(path.resolve(credentials.json)))`, Linux store pinned to `secret-service` (same vault the desktop wrote); key validated as 43-char base64url → 32 bytes; v2 wire form `enc:v2:<source>:<fp24>:<iv>.<tag>.<data>` with `fp24 = hex(sha256(key))[:24]` verified before decrypt. Legacy `enc:v1:` fallback: `ZCODE_CREDENTIAL_SECRET` env or machine-bound string `zcode-credential-fallback:<platform>:<homedir>:<username>`, sha256(secret) as the AES-256-GCM key; plaintext entries pass through; values with an unhandled `enc:` prefix are skipped, never served as if plaintext | [V-SRC src/backend/credential-store.ts#KEYRING_SERVICE] [V-SRC src/backend/credential-store.ts#v2OsKeyForCredentialsFile] [V-SRC src/backend/credential-store.ts#resolveCredentialSecret] [V-SRC src/backend/credential-store.ts#decryptV2Os] | confirmed — same service/account/pinning formula as `secret-tool lookup`; adds the fp24 checksum and the skip-unknown-prefix posture |
| 2 | `interaction/requestCredentialCipher` served in BOTH paths | Turn-loop queue path and the arrival-time path (mid-boot asks with no active turn) both serve encrypt/decrypt ops through the keyring key; the reply `{requestId, result: {ok: true, value}}` echoes the CLI's requestId (the backend's zod schema treats anything else as invalid payload); a non-servable op declines so the CLI keeps (never deletes) the stored login | [V-SRC src/handlers/server-requests.ts#requestId] [V-SRC src/handlers/server-requests.ts#credentialCipherOp] [V-SRC src/backend/credential-store.ts#credentialCipherOp] | confirmed — requestId correlation exactly as reported |
| 3 | Account-provider snapshot | Pushes the snapshot via `provider/updateAccountConfig`; the bundled-table revision is computed as `zcode-builtin:<revision>:<sha256(path.resolve(file))>` — hash over the resolved PATH STRING, byte-for-byte the formula this repo independently sharpened in `docs/protocol/install-discovery.md`; the bridge's own snapshot carries `revision: "account:bridge:<timestamp>"` plus `basedOnZCodeBuiltinRevision` | [V-SRC src/config/account-provider.ts#builtinRevision] [V-SRC src/config/account-provider.ts#AccountProviderPayload] | confirmed — exact revision formula; a mismatched `basedOn` is schema-accepted but the registry keeps serving the previous snapshot (the ack-≠-applied trap this repo documents for Phase 6) |
| 4 | Coding-plan PAT chain | `GET getCustomerInfo` → pick the personal default project (team projects and `projectType 2` excluded) → `GET .../api_keys` and select the key named `zcode-api-key` → `POST .../api_keys/{keyId}/access_tokens` with `{clientType: "zcode", clientVersion}` → answer the runtime-headers ask with `{headersApplied: true, requestAuth: {apiKey, apiKeyId, accountScope}}` (the `accountScope` field enables the backend's one-shot rejected-token → refresh → retry recovery) | [V-SRC src/config/project-token.ts#resolveLocation] [V-SRC src/config/project-token.ts#issueToken] [V-SRC src/handlers/server-requests.ts#answerProviderRuntimeHeaders] | **DRIFT**: the lead said "`api_keys` (create if absent)" — v0.89.0 NEVER creates the key: the list is read-only ("the bridge stays read-only towards the account") and a missing key is the fixed error `project_token_key_missing` with the user cue to open the desktop app once; the claim of a `requestAuth` answer shape with `apiKeyId` is confirmed (plus `accountScope`) |
| 5 | Runtime-headers responder | The backend asks `interaction/requestProviderRuntimeHeaders` before EVERY model request on a `zhipu-account` provider; answered both from the turn-loop queue AND at frame arrival (an arrival-time responder on the backend client) — an ask landing outside any turn loop (compact's internal turn) otherwise dies at the backend's 180 s cap; declining yields `{headersApplied: false, errorMessage}` and the backend fails with -32031 carrying that message | [V-SRC src/backend/client.ts#providerRuntimeHeadersResponder] [V-SRC src/handlers/server-requests.ts#answerProviderRuntimeHeaders] | beyond the lead — the arrival-time half and the 180 s / -32031 failure mode are the load-bearing detail for Phase 5's interaction responder |
| 6 | Elicitation with permission fallback | `AskUserQuestion` forwards `elicitation/create` (form) when the client declares `elicitation.form`; `ExitPlanMode` uses the form only for a client kind that renders form markdown well, every other client keeps `session/request_permission`; when the form channel fails (OR-merged capabilities can outlive the client), the form path falls back to `session/request_permission` — every client can answer that | [V-SRC src/server.ts#supportsFormElicitation] [V-SRC src/handlers/server-requests.ts#handleSinglePermission] | confirmed with nuance — "prefers elicitation, falls back to permission" is real but gated: capability-gated for AskUserQuestion, client-kind-gated for ExitPlanMode |
| 7 | `session/requestRuntimePreferences` | Newer app-servers block `session/create` until this handshake is answered; the reference auto-replies `{nativeSearchEnhancementsEnabled: false, memoryEnabled: false, askUserQuestionAutoResolutionEnabled: false}` — without a reply, create hangs | [V-SRC src/backend/client.ts#handleServerRequest] | beyond the lead — independently matches this repo's own E3 discovery (unanswered create fails -32603; see `docs/protocol/app-server-dialect.md`) |
| 8 | Slash commands | Bridge-intercepted commands (compact / goal / fork / model / mode / thought / quota / mcp …) dispatch the matching session method, emit a feedback chunk, and end the turn without the model; backend/plugin commands (skill, init, code-review) pass through to `session/send`; discovered skills and plugin commands are advertised via `available_commands_update`; TUI-only commands get a friendly error | [V-SRC src/handlers/slash.ts] [V-SRC src/utils.ts#SLASH_COMMANDS] [V-SRC src/config/plugin-commands.ts] [V-SRC src/config/skill-discovery.ts] | confirmed — a full generic slash surface exists; this repo scopes generic slash commands to v2 (EXT-01), keeping only the `/model` contract via SESS-03 |
| 9 | Resume / fork | `session/resume`, `session/load`, `session/load_earlier` (targeted replay) and `session/fork` (extension method); lazy-session adopt with "Session not found" handling; `/fork` slash command forks at the latest checkpoint | [V-SRC src/index.ts#loadSession] [V-SRC src/handlers/session.ts#resumeSession] [V-SRC src/handlers/session.ts#loadSession] | confirmed |
| 10 | Attachments | Prompt content limits `{blocks: 100, image: 10 MiB, total: 50 MiB}`; image MIME allowlist plus `pdf` and `file` kinds with MIME validation | [V-SRC src/handlers/prompt-content.ts#PROMPT_LIMITS] | confirmed |
| 11 | Session lifecycle | new / resume / resume-into / list / cancel / steering / background-tasks handlers; notification vocabulary `session/event`, `state.updated`, `session/lifecycle/changed`, `interaction/providerRuntimeHeadersCancelled` | [V-SRC src/handlers/session.ts#newSession] [V-SRC src/handlers/session.ts#cancel] [V-SRC src/backend/client.ts#handleNotification] | confirmed |
| 12 | Interaction robustness | Reannounce dedup: the app-server re-announces unanswered interaction asks every ~1 s sharing requestId/toolCallId — first forwards, reannounces join the cached answer or record their id for one unified reply; reconnect resend: asks fired while a client was offline are re-sent when its load/resume completes (first-response-wins race) | [V-SRC src/handlers/server-requests.ts#DedupEntry] | beyond the lead — required for a correct Phase 5 interaction broker |

## Adopt

Golden source references for Phase 6's OAuth chain — port the proven BEHAVIOR
(with attribution), not the TypeScript implementation:

- `src/backend/credential-store.ts` — proves the full v2-keyring read path: service/account/pinning formula, fp24 verification, the v1 fallback ladder, per-entry error isolation, and the mtime+size invalidation idea. The single most load-bearing file for the Go `internal/zcode` credentials port (godbus/dbus/v5 replaces the native keyring module; the FORMULA ports, the library does not).
- `src/config/account-provider.ts` — proves the snapshot payload shape, the exact `zcode-builtin:<rev>:<sha256(resolved-path)>` revision string, and the ack-≠-applied trap (a schema-accepted mismatch leaves the registry serving the previous snapshot — verification must be a registry read-back, never the ack).
- `src/config/project-token.ts` — proves the PAT exchange end-to-end: fixed `project_token_*` error codes, single-flight of concurrent exchanges, refresh-lead scheduling, fingerprint-based invalidation on backend rejection, and the read-only posture (never mint keys headless).
- `src/handlers/server-requests.ts` — proves both interaction-serving paths (turn-loop and arrival-time) with the exact reply envelopes (`{requestId, result: {ok, value}}` for cipher ops; `{headersApplied, requestAuth}` / `{headersApplied: false, errorMessage}` for runtime headers) and the decline shapes that keep the CLI healthy instead of hanging it.
- `src/backend/client.ts` — proves the notification vocabulary and the `session/requestRuntimePreferences` default answer (see row 7).

Golden test references — the acceptance oracle for the Phase 6 Go port; each
Go behavior should map to one of these cases:

- `tests/credential-cipher.test.ts` — upstream-parity wire vectors: encrypt/decrypt round-trip, interoperability with the desktop's own algorithm in BOTH directions, fingerprint-mismatch rejection, malformed-payload rejection, plaintext passthrough, v1 fallback derivation.
- `tests/provider-runtime-headers.test.ts` — result-shape invariants across the queue path, the arrival-time responder, turn cancellation, OAuth-exchange success/failure, static-key precedence, and Start-Plan JWT serving.
- `tests/project-token.test.ts` — exchange behaviors: personal-project pick, fingerprint-invalidation drop, single-flight, transient-refresh backoff, never-create-the-key, refuse-without-login (no network touched).

## Avoid

What NOT to adopt from the reference, and why:

- `@napi-rs/keyring` as a dependency — the Go port reaches Secret Service directly through godbus/dbus/v5 (stack decision: pure Go, static binary preserved). Adopt the service/account/pinning formula, never the native module.
- The ACP server/routing layer (`src/server.ts`, `src/index.ts`) — this project's front leg is the coder/acp-go-sdk behind an `internal/acp` port; re-porting the reference's hand-rolled routing would fight the SDK and duplicate what the honest-ACP design already owns.
- Client-kind sniffing (martty/Zed render workarounds in the ExitPlanMode path) — this project's ACP client contract is a single documented card (see `docs/protocol/acp-client-dialect.md`); port the capability-gated pattern (`elicitation.form` presence → form, else permission request), not per-client forks.
- The remote/broadcast multi-client surface (`src/remote/`) — a stdio bridge has exactly one client; the whole offline-reconnect/resend machinery is out of scope for v1.
- Legacy config ladders (pre-3.12 frozen `config.json` keys, `provider_config.json` manual-key ranking) — this project targets the current 3.18.2 layout only; the OAuth store path is the one request-auth source that must be served.
- The mtime+size sync cache as written — single-threaded Node idioms do not transfer to Go; keep the invalidation idea, reimplement the concurrency per Go practice.

## Attribution

- Upstream license: **Apache-2.0**, live-verified (`gh api repos/william0wang/zcode-acp --jq '.license.spdx_id'` → `Apache-2.0`, 2026-11-19) and confirmed by the `LICENSE` file present in the v0.89.0 clone.
- **Porting to Go is allowed WITH attribution.** Upstream ships **no separate `NOTICE` file** (verified in the v0.89.0 clone), so the exact preservation obligation is:
  1. Keep the Apache-2.0 license text/headers with ported code: per Apache-2.0 §4(b)/(d), redistributed (modified) portions must carry the license and retain all legal notices — each ported Go file gets an Apache-2.0 header naming the upstream project and tag.
  2. Attribute william0wang/zcode-acp in this repository's README/notice section for ported code, consistent with this repo's own Apache-2.0 `LICENSE`.
  3. Record the upstream tag ported from (v0.89.0) in the porting PR description, so downstream modifications are traceable to the studied release.
- Apache-2.0 is permissive: no reciprocal obligations beyond the notices above; the patent grant travels with the code.

## Weaker adapters

Other existing ACP adapters, for the record — all worse for this project's
purpose (a headless ACP server over the desktop app-server):

| Adapter | Language | Why not |
|---------|----------|---------|
| jpalmae/zcode-acp | Rust | Own OAuth only (browser/device flow) against the legacy v1 fallback cipher; no OS-keyring access (no Secret Service / keyring reference in source), no `requestCredentialCipher` serving, no coding-plan PAT chain. Slug live-confirmed 2026-11-19 (Rust, Apache-2.0, active); the research lead's "jpalmev/jpalmae" was a doubled slug — `jpalmev/zcode-acp` returns 404. Behavior gaps grep-verified in a fresh clone |
| Liang-HZ/zcode-acp | Python | Only v1 fallback decryption (`enc:v1` AES-256-GCM machine-secret path); `enc:v2:os:` blobs are returned undecrypted (treated as a login marker); no keyring, no cipher-op serving, no PAT chain. Slug live-confirmed 2026-11-19 (Python, Apache-2.0, active); macOS-centricity per the research lead only [V-RPT REFERENCE-ADAPTERS.md] |

Differentiator claim (FEATURES.md) now verified, not just reported: **nobody
else does the v2-keyring + coding-plan chain** — of the three ACP adapters
checked, only william0wang/zcode-acp combines OS-keyring v2 decryption, the
`requestCredentialCipher` service, and the coding-plan PAT exchange.

## Limitations worth documenting

- **Start Plan does not work headless** — upstream's own TROUBLESHOOTING: Start Plan providers authenticate with the provider's OAuth JWT **plus an Aliyun captcha session** that only the desktop app can provide; headless/editor sessions fail with 1113 / signing errors, while **GLM Coding Plan works headless** [V-SRC docs/TROUBLESHOOTING.md (upstream), "Start Plan (zcode-plan) providers fail headless"]. This is the note that feeds **DOCS-03** (Phase 7 README/troubleshooting): document that headless use requires a Coding Plan provider.
- Nuance found live: with a stored Start-Plan JWT the reference CAN answer the runtime-headers ask without interaction (including the no-providerId fallback ask), but a fresh login or a captcha refresh can never happen headless — the desktop-only captcha session is the irreducible limit [V-SRC src/handlers/server-requests.ts#PROVIDER_RUNTIME_HEADERS_UNAVAILABLE].

## Sources & provenance

- Upstream studied: github.com/william0wang/zcode-acp, tag **v0.89.0** (release published 2026-11-19; default branch `main`; npm package `zcode-acp-server` 0.89.0). Every V-SRC citation above was read from a fresh shallow clone of that tag; anchors are function/constant names in the cited file.
- License: `gh api repos/william0wang/zcode-acp --jq '.license.spdx_id'` → `Apache-2.0` (2026-11-19, exit 0); `LICENSE` file present in the clone; no `NOTICE` file present.
- Weaker adapters: `gh api repos/jpalmae/zcode-acp` (Rust, Apache-2.0, pushed 2026-11-13) and `gh api repos/Liang-HZ/zcode-acp` (Python, Apache-2.0, pushed 2026-10-14), both 2026-11-19; behavioral gaps grep-verified in fresh shallow clones of their default branches.
- V-RPT items: the ~91★ / release-activity trend and "releases track ZCode desktop migrations" (research lead, not re-checked); macOS-centricity of the Python adapter.
- Research lead upgraded by this doc: `.planning/research/REFERENCE-ADAPTERS.md` — its adopt-list claims are confirmed or corrected row by row above (one drift: no key creation); its Orca-integration sections are out of this document's scope and remain governed by `docs/protocol/acp-client-dialect.md`.

---
*Verified against the live reference repo at tag v0.89.0 on 2026-11-19; sources as listed.*
