# Phase 6: OAuth Coding-Plan Path & Model Picker - Research

**Researched:** 2026-11-22
**Domain:** the official OAuth individual-coding-plan chain (keyring credential read → account-provider snapshot → credential-cipher answers → PAT mint on api.z.ai → runtime-headers PAT serving) plus the ACP v1 session config-options surface (model + thought-level picker over the live provider registry)
**Confidence:** HIGH — every load-bearing wire shape was read this session from the primary sources: the william0wang/zcode-acp **v0.89.0** golden clone (Apache-2.0 porting source), the zcode-src **3.17.1** clone (corroborating tier per dialect D-01), the coder/acp-go-sdk **v0.20.0** module source, the official ACP v1 spec pages, the in-repo Phase-5 code, the machine-local FINDINGS.md field notes, and three live probes on this machine (secret-tool failure modes, the real builtin provider table, the desktop's own setting.json/credentials.json structure — scrubbed). Residual unknowns are tagged `[ASSUMED]`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** secret-tool stays as the v1 decrypt transport **behind the existing
  KeyLoader seam**, with the explicit commitment that a Phase-7 godbus/dbus swap
  must be a local seam edit — no business logic may bypass the seam. (Phase-5
  real run already proved this transport on the real desktop; zero new deps.)
- **D-02:** AUTH-04 is **proactive**: `authenticate{zai-individual-coding-plan}`
  probes keyring availability (secret-tool binary present + expected keyring
  entry readable) BEFORE any snapshot push; failure answers a coded actionable
  error naming the missing piece (e.g. which package to install). The chain
  never hangs.
- **D-03:** The **legacy enc:v1 fallback is ported** per the reference
  (`ZCODE_CREDENTIAL_SECRET` / machine-string key derivation) alongside the
  field-verified `enc:v2:os` layout — completeness over v2-only minimalism
  (user choice against the recommended default).
- **D-04:** The PAT is **minted once at authenticate and lives in-process only**
  for the wrapper's lifetime; runtime-headers asks are answered from that cache.
  The PAT is never persisted to disk — the T-5-17 key-material hygiene extends
  to it (test-locked).
- **D-05:** Expiry is **reactive**: no TTL guessing. On staleness evidence
  (headersApplied:false, turn auth failure) — exactly one silent re-mint and
  retry of the ask; a second failure surfaces the honest coded error. The
  researcher pins the actual TTL/refresh semantics from source and field notes.
- **D-06:** Multi-project accounts: **deterministic first personal project** +
  slog warn listing all projects when several exist. The user is never blocked
  (reference personal-pick behavior).
- **D-07:** No API keys on the account → **create-if-absent via the official
  API** (reference and desktop behavior). Idempotent; no manual console step.
- **D-08:** The `configOptions` model list is a **verbatim mirror of what the
  backend announces available after auth** (`settings.model.available` /
  registry surfaces) — no wrapper-side filtering, renaming, or enrichment. The
  client can never pick a model the backend would refuse (single source of truth).
- **D-09:** Thought levels come **from per-model metadata as reported by the
  backend** — never hardcoded lists. (Researcher pins where metadata carries
  levels; fake corpus locks the shapes.)
- **D-10:** `currentValue` = **server-confirmed state only** (the created
  session's reported current + the outcome of the follow-up setModel). An
  unconfirmed target is never shown as selected.
- **D-11:** The picker works on **BOTH auth paths** (api-key + coding-plan) —
  shared machinery, parity, and preparation for the Phase-7 Orca `/model`
  contract (SESS-03).
- **D-12:** The phase bar is a **full live test, the Phase-5 standard**: login
  through the desktop's saved credentials (keyring) → real PAT mint at z.ai →
  **one real small request on the coding-plan account** (success criterion 1's
  "on a real account" proven field-first, not "should work").
- **D-13:** Spend cap per the Phase-5 D-05 precedent: one tool-less tiny
  request; every turn ledgered in experiments/README quota section; scrubbed
  captures + V-EXP ids for every real-account claim; CI never touches the
  real account or keyring.

### Claude's Discretion

- Coding-plan authenticate requires **no user-supplied wrapper config** — the
  desktop's saved login IS the credential (official-paths-only; Start Plan
  headless documented as not working per FINDINGS). Missing login = coded
  actionable error ("log in via the ZCode desktop first").
- Unknown/stale config option from the client → coded invalid-params per the
  ACP error taxonomy, never a crash path.
- Encrypt-operation cipher asks stay declined (Phase-5 posture — the chain is
  read-only); the researcher verifies from source that coding-plan auth never
  requires host-side encryption.
- Mint-chain per-call network timeouts; error wording; wave/task decomposition;
  whether `session/setThoughtLevel` is a distinct backend method or a setModel
  field (per dialect docs — researcher resolves, planner locks).

### Deferred Ideas (OUT OF SCOPE)

- godbus/dbus Secret Service transport — Phase 7 optional local seam swap (D-01)
- SESS-02 (session/delete + fork graceful no-ops) — Phase 7
- SESS-03 (Orca persona contract: `--model` at launch, `/model` slash prompt) — Phase 7
- elicitation/create preference — Phase 7 integration
- session/load/fork real support — v2
- Persistent cross-restart PAT cache — rejected for v1 (disk hygiene); revisit
  only on field demand
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AUTH-02 | OAuth individual coding plan: registry snapshot with exact `zcode-builtin:<rev>:<sha256>` revision, `interaction/requestCredentialCipher` (Secret Service key decryption), PAT mint via api.z.ai chain, `requestAuth` response — official paths only | Coding-plan snapshot carries a NON-empty account-provider overlay mirrored from the desktop's own builtin table (Pattern 2); credential store + v1/v2 decrypt ladder (Pattern 1); PAT mint chain verbatim incl. create-if-absent (Pattern 3); runtime-headers answer with accountScope (Pattern 4); verdict = registry read-back (Phase-5 machinery reused) |
| AUTH-03 | Both advertised authMethods authenticate; account models appear after auth | The api-key arm is Phase-5-landed and untouched; the coding-plan arm composes the same push → refresh → read-back spine (auth.go) with the account overlay; `account:zai-individual-coding-plan/GLM-*` models read back via the create reply's `settings.model.available` (V-EXP EXP-APIKEY-CYCLE-02/03 precedent) |
| AUTH-04 | Missing keyring (headless/CI) fails fast, never hangs | D-02 proactive probe: LookPath(secret-tool) + bounded lookup; live probed failure modes (`Cannot autolaunch D-Bus…` rc=1; empty-entry rc=1) — both fail fast (Pattern 5); every keyring/network call deadline-bounded |
| MODEL-01 | `session/new` returns `configOptions` (model + thought level) from the live registry | Create reply `settings` schema (model.available + thoughtLevel block) verified in zcode-src; SDK `NewSessionResponse.ConfigOptions` verified; capability advertisement `sessionCapabilities.options` (Pattern 6) |
| MODEL-02 | `session/set_config_option` → `session/setModel` / `session/setThoughtLevel`, re-emits full list, honest `currentValue` | SDK SetSessionConfigOption request/response verified (full-list re-emit REQUIRED); backend setModel/setThoughtLevel replies carry the full settings snapshot (`afterStateMutation`) — the D-10 mechanism (Pattern 7) |
</phase_requirements>

## Summary

Phase 6 completes the two halves the roadmap reserved for it. The **auth half**
extends the Phase-5 authenticate spine (`internal/acpfront/auth.go` →
`internal/hostsvc`) with the coding-plan arm: a proactive keyring probe (D-02),
a credential-store reader over the existing KeyLoader seam extended with the
`enc:v1` fallback ladder (D-03), an account-provider snapshot that — unlike the
api-key path's intentional empty overlay — carries the `account:zai-*`
providers mirrored from the desktop's own bundled builtin table, and a PAT
minted once at authenticate from the official api.z.ai chain (D-04/D-06/D-07),
served thereafter to `interaction/requestProviderRuntimeHeaders` from an
in-process cache with reactive re-mint on staleness (D-05). Every mechanism
this needs exists: the responder policy object, the exact-revision snapshot
formula, WatchRegistry/Await read-back, and the cipher-decrypt seam are all
Phase-5-landed and test-locked.

The **picker half** is a translation problem, not a protocol problem. The
backend already reports everything needed: the create/setModel/setThoughtLevel
replies each carry the full session `settings` snapshot (`model.available` with
per-model `reasoning.levels`, plus a `thoughtLevel` block with its own
available list, current, and default). ACP v1's config-options surface is fully
present in the pinned SDK v0.20.0 (`NewSessionResponse.ConfigOptions`,
`SetSessionConfigOptionRequest/Response`, the `config_option_update`
notification, `SessionCapabilities.Options` advertisement, and the
`model`/`thought_level` category constants). The wrapper mirrors the backend's
reported state verbatim (D-08), never invents a level (D-09), and keeps
`currentValue` server-confirmed only (D-10) — the backend mutation replies
themselves are the confirmation source.

Two findings shape planning. **(1) The coding-plan snapshot is NOT the empty
overlay** — the reference push carries `providers{...access:{type:"zhipu-account",
entitled}}` entries derived from the builtin table plus OAuth-login-derived
entitlement, with `states{id].{availability, entitled, current}}`; the
entitlement signal (`providerFamilyConnectionSelections.zai ==
"individual-coding-plan"` + `oauth:zai:access_token` in the store) was probed
live on this machine. **(2) The runtime-headers ask actually fires on this
path** — the Phase-5 research proved it is account-provider-gated
(`zhipu-account` only), so the coding-plan real run is the first mode where the
PAT answerer is load-bearing rather than defensive; the answer must carry
`accountScope` (the backend echoes it back as `expectedAccountScope` on a
rejected-token re-ask — the D-05 re-mint trigger).

**Primary recommendation:** port the reference behavior behind the existing
seams — credential store reader + v1 fallback into the KeyLoader family, an
account-snapshot builder extension in `hostsvc`, a `zcodeauth`/`hostsvc` PAT
minter on stdlib net/http, an auth-mode-aware runtime-headers source wired
through `WithRuntimeHeadersKeySource`'s successor, and a config-options
translator in `acpfront`/`acpport` — with the real-tier guarded test
(TestCodingPlanFullCycle) as the phase bar per D-12/D-13.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Keyring probe (AUTH-04) | Backend-adjacent (`internal/interact` KeyLoader family + the front's authenticate arm) | — | The probe is the credential machinery's own availability check; it must answer before any snapshot traffic |
| Credential store read + enc:v1/v2 decrypt (D-03) | `internal/interact` (KeyLoader seam extension) | `internal/hostsvc` (consumes the reader) | The seam owns the key material; hygiene locks (T-5-17) already live there |
| Coding-plan authenticate chain (AUTH-02/03) | Front tier (`internal/acpfront` authenticate arm) + `internal/hostsvc` (snapshot builder extension, read-back) | `cmd/zcode-acp` (wiring only — NO new user config per D-12) | Auth state is process-global; the chain composes the same push→refresh→read-back spine as AUTH-01 |
| PAT mint + cache (D-04/05/06/07) | `internal/hostsvc` (or a sibling exec tier) on stdlib net/http | `internal/interact` (the answerer consumes the cache) | Pure HTTPS client with no ACP or conn dependency; the responder seam stays thin |
| Runtime-headers PAT serving | `internal/interact` (policy row over the mode-aware source) | `internal/hostsvc` (cache) | The responder answers asks whenever they arrive (turn-loop and arrival-time parity per reference row 5) |
| Cipher asks during auth | `internal/interact` `CredentialDecrypt` (landed, extended loader) | — | Same responder row; the loader gains the v1 fallback and the store reader |
| configOptions in `session/new` (MODEL-01) | Front tier (`acpfront` NewSession reads the create/setModel replies' settings) + `acpport` (neutral ConfigOption vocabulary → SDK types) | `internal/zcode` (envelope decode only) | Client presentation over backend-reported state — the same tap-front ownership as tool streaming |
| `session/set_config_option` (MODEL-02) | Front tier (`acpfront` handler; backend setModel/setThoughtLevel calls) + `acpport` (adapter delegation replacing the -32601 default) | `internal/zcode` (method constants; `MethodSessionSetThoughtLevel` is NEW) | The front owns backend calls and state; the adapter owns wire translation |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`net/http`, `crypto/aes`, `crypto/sha256`, `encoding/json`, `os/exec`, `context`) | go 1.27 | PAT mint HTTPS client, credential decrypt (v1+v2), keyring probe, config-options translation | Zero-new-deps discipline (CONTEXT, STACK.md: "Three plain HTTPS calls with exact shapes documented in FINDINGS.md; an SDK would guess wrong") [VERIFIED: in-repo .claude/CLAUDE.md stack table + go.mod single-require graph] |
| internal/interact KeyLoader seam + `secret-tool` | in-repo + system binary | Credential-key loading behind the D-01 seam (v2:os), now extended with the v1 fallback and the store reader | D-01 locks secret-tool as the v1 transport; godbus explicitly deferred [VERIFIED: in-repo internal/interact/cipherdecrypt.go:112-119,315-404] |
| internal/hostsvc SnapshotBuilder + RegistryWatcher | in-repo | Exact-revision snapshot push + PITFALL-1 read-back verdict — extended with the account-provider overlay | Phase-5 landed, test-locked; the coding-plan arm reuses both unchanged [VERIFIED: in-repo internal/hostsvc/snapshot.go:83-146, readback.go:51-128] |
| coder/acp-go-sdk | v0.20.0 (pinned) | `NewSessionResponse.ConfigOptions`, `SetSessionConfigOptionRequest/Response`, `SessionConfigOption` select variant, `SessionConfigOptionCategoryModel/ThoughtLevel`, `SessionCapabilities.Options`, `SessionUpdateConfigOptionUpdate` | All config-options surface verified present in the pinned module this session [VERIFIED: acp-go-sdk v0.20.0 types_gen.go:6686-6700 (NewSessionResponse.ConfigOptions), :11492-11530 (request variants), :11582-11630 (response + required-list Validate), :8774-8960 (select option), :8960-8977 (categories), :8604-8664 (SessionCapabilities.Options), :10342-10365 (config_option_update); constants_gen.go:32] |
| internal/zcode/fake + golden corpus | in-repo | New scenarios: cipher-ask-during-auth, PAT-driven runtime-headers, configOptions/set_config_option round-trips | Reverse-request scripting with `expect_answer` proven (05 corpus rows) [VERIFIED: in-repo fake/testdata/scenarios/auth-apikey.json, runtime-headers-answer.json, credential-cipher-scripted.json] |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `secret-tool` (libsecret CLI) | system binary | Keyring probe (D-02) + v2:os key load (D-01) — real tier only | Present on this machine (`/usr/bin/secret-tool`); CI never constructs the production loader |
| godbus/dbus/v5 | v5.2.2 | **NOT in Phase 6** — the D-01 Phase-7 seam swap | Locked deferral; the KeyLoader interface is the upgrade boundary [VERIFIED: in-repo cipherdecrypt.go:112-119 seam doc] |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| stdlib net/http PAT client | zalando/go-keyring-style SDK or a z.ai SDK | No z.ai SDK exists; an HTTP lib adds deps for three calls the reference pins byte-exactly — stdlib matches STACK.md's explicit decision |
| `session/setThoughtLevel` for level switches | `session/setModel` with `options.reasoningLevel` | setModel+options is the field-proven binding path on 3.18.2 (EXP-TURN-01 refusal proved the requirement); setThoughtLevel is source-verified but never exercised live — see Open Questions (planner locks) |
| Reading the store's `account-provider:coding-plan:…:api-key` entries as the PAT source | Minting per D-04 | Desktop 3.15+ no longer stores plan api-keys as the auth source (project-token.ts header doc); the mint is the official path and D-04 locks it |

**Installation:**

```bash
# none — zero new dependencies (stdlib only); go.mod stays single-require
```

**Version verification:** `go list -m all` still prints exactly
`github.com/Djarvur/zcode-acp` + `github.com/coder/acp-go-sdk v0.20.0` (checked
indirectly this session via the module cache — the pinned SDK dir
`acp-go-sdk@v0.20.0` is the only external module). No new packages are
introduced by this phase; the legitimacy audit below is a zero-install audit.

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| coder/acp-go-sdk v0.20.0 | Go module proxy (not npm — ecosystem artifact) | since 2026-11-18 | n/a (Go) | github.com/coder/acp-go-sdk | OK | Already pinned; config-options surface read and verified in the module cache this session |
| stdlib net/http / crypto/aes | Go stdlib | — | — | go.dev | OK | No install |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none — this phase installs nothing.
The only external runtime dependency is the system `secret-tool` binary
(present, verified this session), used on the guarded real tier only, never a
Go module and never CI.

## Architecture Patterns

### System Architecture Diagram

```
 ACP client (config-options picker UI)
     │ ACP v1 (initialize / authenticate / session/new / session/set_config_option)
┌────▼────────────────────────────────────────────────────────────────────────┐
│ zcode-acp                                                                   │
│  acpport adapter                                                            │
│   • initialize: + sessionCapabilities.options (NEW advertisement)           │
│   • authenticate{zai-individual-coding-plan} → Front (NEW arm)              │
│   • session/new → response.configOptions (NEW, MODEL-01)                    │
│   • set_config_option: -32601 default → real delegation (NEW, MODEL-02)     │
│  acpfront Front                                                             │
│   • authenticate coding-plan arm (NEW):                                     │
│       1. keyring probe (D-02) ──✗→ coded actionable error, never a hang     │
│       2. credential read: store reader → oauth login token (D-03)           │
│       3. account snapshot build (builtin table mirror) → push → refresh     │
│          → read-back verdict (Phase-5 WatchRegistry — ack never trusted)    │
│       4. PAT mint once (D-04) → in-process cache (never disk)               │
│   • NewSession: create reply + setModel reply → configOptions (MODEL-01)    │
│   • SetSessionConfigOption: configId → setModel / setThoughtLevel (MODEL-02)│
│         │ replies' settings snapshot = server-confirmed currentValue (D-10) │
│  hostsvc                                                                    │
│   • KeyStore reader (credentials.json entries) + v1/v2 decrypt ladder       │
│   • PatMinter (stdlib net/http, 15 s per-call timeouts):                    │
│       GET getCustomerInfo (bare token) → personal pick (D-06) →             │
│       GET api_keys (Bearer) → POST api_keys create-if-absent (D-07) →       │
│       POST access_tokens {clientType:zcode} → {accessToken, expiresIn,…}    │
│   • PAT cache: token/apiKeyId/fingerprint/accountScope/refreshAt            │
│  interact responder (existing policy object)                                │
│   • requestProviderRuntimeHeaders → mode-aware source:                      │
│       api-key mode → static key (Phase 5, unchanged)                        │
│       coding-plan mode → PAT cache (+accountScope);                         │
│         rejectedProjectTokenFingerprint → exactly one silent re-mint (D-05) │
│   • requestCredentialCipher → CredentialDecrypt (loader gains v1 fallback)  │
│         │ answers ride the bypass lane (landed)                             │
└────────▼────────────────────────────────────────────────────────────────────┘
     zcode app-server (host-managed): asks cipher/headers mid-chain and
     mid-turn; create/setModel/setThoughtLevel replies carry the FULL
     settings snapshot (model.available + thoughtLevel) — the read-back
     and the picker's single source of truth
```

Primary use case traces:
- AUTH-02: `authenticate{zai-individual-coding-plan}` → probe ✓ → store read ✓ → snapshot push → refresh → read-back latch (account models present) → PAT mint ✓ → `{}` success → subsequent `session/prompt` fires the runtime-headers ask → PAT answer → turn completes.
- MODEL-01/02: `session/new` → create reply settings → configOptions[model, thought_level] → client `session/set_config_option {configId:"model", value:"account:zai-individual-coding-plan/GLM-5.3"}` → backend `session/setModel` (options.reasoningLevel carried) → reply settings → FULL configOptions re-emit with confirmed currentValue.

### Recommended Project Structure

```
internal/
├── interact/
│   ├── cipherdecrypt.go    # EXTEND: KeyLoader gains the v1-fallback sibling loader
│   ├── credstore.go        # NEW: credentials.json reader (entries map, mtime+size
│   │                       # cache idea per reference; per-entry error isolation)
│   └── responder.go        # EXTEND: mode-aware runtime-headers source seam
├── hostsvc/
│   ├── snapshot.go         # EXTEND: account-provider overlay builder (builtin-table
│   │                       # mirror); the empty api-key overlay stays untouched
│   ├── patminter.go        # NEW: the official api.z.ai chain + in-process cache
│   └── readback.go         # unchanged (WatchRegistry reused verbatim)
├── acpfront/
│   ├── auth.go             # EXTEND: the coding-plan arm (probe → cred → snapshot →
│   │                       # read-back → mint); api-key arm untouched
│   ├── configoptions.go    # NEW: settings-snapshot → []acpport.ConfigOption translator
│   ├── front.go            # EXTEND: NewSession returns options; SetSessionConfigOption
│   └── front.go/acpport    # METHOD constant: zcode.MethodSessionSetThoughtLevel (NEW)
├── acpport/
│   ├── agent.go            # EXTEND: SetSessionConfigOption delegation replaces -32601;
│   │                       # initializeResponse advertises sessionCapabilities.options
│   └── types.go            # EXTEND: neutral ConfigOption vocabulary (SDK types never cross)
└── cmd/zcode-acp/main.go   # EXTEND: wire the coding-plan source + cache; NO new user flags
test/realdesktop/
└── apicycle_test.go        # SIBLING: codingplan_test.go (guarded, D-12/D-13 discipline)
```

### Pattern 1: The credential store reader + the enc:v1 fallback (D-03)

**What:** a read-only reader over `~/.zcode/v2/credentials.json` (a JSON object
of `key → string value`), where values are plaintext, `enc:v1:` blobs, or
`enc:v2:os:` blobs. The D-06 `KeyLoader` seam gains a legacy sibling; the store
reader composes both.

**Wire formats (verbatim):**

- `enc:v1:<iv>.<authTag>.<ciphertext>` — all three segments base64url,
  dot-separated, exactly 3; key = `sha256(secret)`; AES-256-GCM; a value
  without the prefix passes through as plaintext [VERIFIED: ref v0.89.0
  src/backend/credential-store.ts:26,51-68 — `ENCRYPTED_PREFIX = "enc:v1:"`,
  `decryptCredential` quoted]
- v1 secret resolution: `ZCODE_CREDENTIAL_SECRET` env (trimmed) when set, else
  the machine-bound string `` `zcode-credential-fallback:${platform()}:${homedir()}:${username}` ``
  [VERIFIED: ref credential-store.ts:33-43 — `resolveCredentialSecret` quoted]
- `enc:v2:os:<fp24>:<iv>.<tag>.<ct>` — fp24 = `hex(sha256(key))[:24]`; the
  keyring key from service `ZCode Credential Encryption`, account
  `hex(sha256(path.resolve(credentials.json)))`, Linux store pinned
  `secret-service`; key shape `^[A-Za-z0-9_-]{43}$` → 32 bytes [VERIFIED: ref
  credential-store.ts:81-112,126-146; our `SecretToolKeyLoader.LookupCommand`
  implements the same formula — in-repo cipherdecrypt.go:341-357]
- **Unknown `enc:` prefixes must NEVER pass through as plaintext** — a v2-ENV
  blob served as plaintext silently degraded every credential lookup after the
  desktop 3.18.0 migration (observed live 2026-11-12 upstream) [VERIFIED: ref
  credential-store.ts:166-175 — `hasUnhandledEncPrefix` + the failure comment]

**Store entries the chain consumes** (keys read this session, values never):

- `oauth:<family>:access_token` (family `zai`) — the login token; a `Bearer `
  prefix is stripped [VERIFIED: ref project-token.ts:71-83]
- `zcodejwttoken` — the ZCode JWT folded into accountScope [VERIFIED: ref
  project-token.ts:76,113-118]
- `project-key-location:v1:…` — the desktop's persisted key-location record
  (our wrapper keeps the location in-process only; D-04 hygiene) [VERIFIED: ref
  project-access-token-client.ts:147; live: the entry exists on this machine
  with the URL-encoded payload shape `[origin, family, accountId, null, 1,
  "non-team"]` — V-EXP scrubbed probe, 2026-11-22]

**Per-entry error isolation:** one undecryptable entry never sinks the rest;
unreadable store → empty map, never a panic [VERIFIED: ref
credential-store.ts:231-297 — `readCredentialStoreSync` posture]. Our Go
reader keeps the hygiene locks: values live in reply-construction scope, never
logged (T-5-17 discipline already test-locked in the package).

### Pattern 2: The coding-plan account snapshot (NOT the empty overlay)

**What the reference pushes** (`provider/updateAccountConfig`, strict envelope):

```jsonc
// Source: ref v0.89.0 src/config/account-provider.ts buildAccountProviderConfig()
{
  "revision": "account-bridge:<timestamp>",      // any non-empty string (our form stands)
  "basedOnZCodeBuiltinRevision": "zcode-builtin:<rev>:<sha256(resolve(path))>",
  "providers": {
    "account:zai-individual-coding-plan": {
      "builtinModelIds": ["GLM-5.3", "GLM-5.3-Flash"],
      "access": { "type": "zhipu-account", "entitled": true }
    }
    // one entry per zhipu-account rule the builtin table declares
  },
  "states": {
    "account:zai-individual-coding-plan": {
      "availability": "available",   // "available" | "pending" | "unavailable" | "unknown"
      "entitled": true,
      "current": true                 // required for entitled zhipu-account entries
    }
  }
}
```

[VERIFIED: ref account-provider.ts `AccountProviderPayload` + builder; the
strict zod shape cross-checked against PITFALL 1's `providers{builtinModelIds,
access}, states{availability, entitled, current}` and 05-RESEARCH Pattern 5's
`parseProcessAccountProviderConfigSnapshot` note]

**Where the rules come from:** the desktop's OWN bundled builtin table — the
same `zcode-builtin.json` discovery already resolves (`install.BuiltinFile`,
the exact path `supervisor.ChildEnv` hands the child as
`ZCODE_BUILTIN_PROVIDER_CONFIG_FILE` — the path the snapshot formula hashes
must be the one the backend boots from) [VERIFIED: in-repo
internal/supervisor/spawn.go:33,147-148; discover.go:66,136-137]. Its
`config.providerConfigRules.providerRules[]` filtered to
`config.access.type == "zhipu-account"` yields the account providers — probed
live on this machine's revision-44 table: 10 rules, including
`account:zai-individual-coding-plan | mode: individual-coding-plan |
builtinModelIds: [GLM-5.3, GLM-5.3-Flash]` [V-EXP this session, 2026-11-22].

**Entitlement (desktop parity, read-only):** a family is entitled when
`~/.zcode/v2/setting.json` has
`providerFamilyConnectionSelections[family].kind === "individual-coding-plan"`,
`modelProviderFamilyModes[family] !== "apiKey"`, and the credential store holds
`oauth:<family>:access_token` [VERIFIED: ref account-provider.ts
`oauthSelectedCodingPlanFamilies`; live probe: this machine's setting.json
reads `selections: {zai: "individual-coding-plan", bigmodel: "start-plan"}`,
`modes: {zai: "oauth", bigmodel: "oauth"}` — V-EXP, 2026-11-22]. Reading the
desktop's setting/config files is the credential-derived parity the coding
plan REQUIRES; it does not violate the api-key path's OQ4 rule (that rule
forbids reading a user's API KEY from the desktop store — the coding-plan
arm's whole credential IS the desktop login, per D-12).

**Why push at all:** the backend builds its registry from builtin table +
personal config + the ACCOUNT snapshot; headless launches have no desktop host
to push it, and without it every `account:*` model reads unentitled —
`settings.model.available` misses the GLM models entirely and `session/setModel`
fails "Provider Registry 中不存在 Model" [VERIFIED: ref account-provider.ts:1-16
header doc — upstream verified a push turned 3 models into 21]. Phase-5's
empty overlay was correct for api-key (personal layer); the coding-plan arm
must push the real overlay. The verdict is STILL the registry read-back
(WatchRegistry/Await, never the ack — PITFALL 1 carries unchanged).

### Pattern 3: The PAT mint chain (D-04/D-06/D-07) — every call verbatim

All calls on `https://api.z.ai` (family `zai`), 15 s per-call timeout (ref
`REQUEST_TIMEOUT_MS`), business envelope `{code, data}` with success codes
{200, 0, "200", "0"} [VERIFIED: ref project-token.ts:59,188-211]:

1. **Login token:** store entry `oauth:zai:access_token`, `Bearer ` prefix
   stripped; absent → the coded "log in via the ZCode desktop first" error
   (ref `project_token_login_required`) [VERIFIED: ref project-token.ts:79-83,372]
2. **getCustomerInfo:** `GET /api/biz/customer/getCustomerInfo` with
   `Authorization: <loginToken>` — RAW, NO `Bearer` prefix (a Bearer here is
   misjudged as login-invalid by the server; zcode-src comment verbatim:
   "customerInfo 要求裸业务 token；Bearer 格式拒绝会被误判为登录失效。")
   [VERIFIED: zcode-src 3.17.1 project-access-token-client.ts:265-277; FINDINGS
   §3.3 line 171]
3. **Personal pick (D-06):** filter orgs' projects to non-team
   (`String(projectType).trim() !== "2"`); org = the one whose
   `organizationName` contains `默认机构` else the FIRST personal org; project =
   `projectName` contains `默认项目` else the first; none →
   `project_token_project_unavailable`. Multiple personal orgs/projects →
   deterministic first + slog warn listing the ids (D-06) [VERIFIED: ref
   project-token.ts:53-57,242-274; zcode-src client :291-312]
4. **api_keys list:** `GET /api/biz/v1/organization/{org}/projects/{project}/api_keys`
   with `Authorization: Bearer <loginToken>` (zai form) + Content-Type json;
   select the entry with `name === "zcode-api-key"`; `apiKeyId = entry.apiKey`
   [VERIFIED: ref project-token.ts:52,225-227,276-296; zcode-src client :313-329]
5. **Create-if-absent (D-07):** when no candidate, `POST` the SAME url with
   the SAME Bearer headers and body `{"name":"zcode-api-key","usageScene":1}`
   (`API_KEY_USAGE_SCENE.CODING_PLAN = 1`) — the desktop's own create; the
   reply is the created key entry (`apiKey` = id). The reference bridge does
   NOT do this (read-only posture, `project_token_key_missing`) — D-07 is the
   explicit user decision superseding it; the DESKTOP behavior (zcode-src +
   FINDINGS) is create-if-absent [VERIFIED: zcode-src 3.17.1
   project-access-token-client.ts:339-351 quoted `body: JSON.stringify({name,
   ...(team? {keyType:2}:{}), usageScene: API_KEY_USAGE_SCENE.CODING_PLAN})`;
   FINDINGS.md §3.3 line 179 `body: {"name":"zcode-api-key","usageScene":1}  #
   1 = CODING_PLAN`; ref read-only divergence documented at
   project-token.ts:291-293 and docs/research/reference-adapters.md row 4
   DRIFT]. Idempotency = check-then-create (the candidate search precedes every
   mint); attribution header per Apache-2.0 porting rules.
6. **access_tokens:** `POST …/api_keys/{keyId}/access_tokens` with
   `Authorization: Bearer <loginToken>`, body
   `{"clientType":"zcode","clientVersion":"<desktop version>"}`; HTTP 404 /
   `code:404` → the key moved: drop the cached location and re-resolve (ref
   `project_token_key_missing`). Reply data `{enable?, accessToken,
   tokenType:"Bearer", expiresIn, expiresAt}`; `enable === false` →
   `project_token_disabled`; `expiresAt = min(expiresAtSec*1000,
   started+expiresIn*1000)` [VERIFIED: ref project-token.ts:299-354;
   zcode-src client :214-231; FINDINGS §3.3]
7. **clientVersion:** the reference sends the DESKTOP's version
   (`installedAppVersion(appBundlePath())`, fallback "0.0.0"); zcode-src sends
   `ZCODE_VERSION` (the CLI's own). Our wrapper sends the resolved desktop
   install's version (supervisor.Install carries it via the runtime tree) —
   matching the reference's desktop-version posture [VERIFIED: ref
   project-token.ts:169-177; ASSUMED: which exact value the server validates —
   neither source documents server-side validation]

**accountScope** rides every answer: `sha256(JSON.stringify([family,
"individual-coding-plan", null, loginToken, zcodeJwt|null]))` — byte-identical
in the reference and the desktop host [VERIFIED: ref project-token.ts:113-118;
zcode-src accountProviderRequestAuthService.ts:118-135 — both quoted]. The
backend echoes it as `expectedAccountScope` on a rejected-token re-ask; a
mismatch means the sign-in changed → re-mint against the new scope, never
serve the stale token.

**Cache + reactive refresh (D-05):** cache `{token, apiKeyId,
fingerprint=sha256(token), expiresAt, refreshAt=max(now, expiresAt-120s),
accountScope}` in-process (ref `REFRESH_LEAD_MS = 120_000`). The pinned TTL
value itself is not publicly documented — the reply's expiresIn/expiresAt
drive everything, and D-05's reactive posture (re-mint on
`rejectedProjectTokenFingerprint` or staleness evidence, exactly one silent
retry, then the honest coded error) makes the TTL number immaterial
[VERIFIED: ref project-token.ts:61,126-141,164-167,346-353; ASSUMED: actual
server TTL]. Single-flight concurrent mints (one `pending` promise analog) and
the transient-failure backoff (keep serving a still-valid token, retry in 5 s)
are ref behaviors worth porting [VERIFIED: ref project-token.ts:179-186,390-415].

### Pattern 4: Runtime-headers serving on the coding-plan path

**The ask fires on this path.** The Phase-5 source gate stands: the refresh
callback exists only for `zhipu-account` access types — so the coding-plan
real run is the FIRST live mode where the ask arrives [VERIFIED: 05-RESEARCH
Pattern 5 honesty note (runner.ts:186-221 gate), carried]. Params (strict,
3.17.1): `{requestId, sessionId, turnId?, workspace, modelSelection,
providerId, accountAccess?, expectedAccountScope?: /^[a-f0-9]{64}$/,
rejectedProjectTokenFingerprint?: /^[a-f0-9]{64}$/, reason:
"model-request"}` [VERIFIED: zcode-src 3.17.1 index.ts:2843-2864;
requestVerificationReasons = ["model-request"] quoted from
request-security-edition/verification-policy.ts:4]. Server window: 180 s
(bundle `18e4` ×5); refusal/death: -32031 [CITED:
docs/protocol/app-server-dialect.md §3].

**The success answer (verbatim schema, success arm):**

```jsonc
{ "headersApplied": true,
  "requestAuth": { "apiKey": "<PAT>", "apiKeyId": "<keyId>",
                   "accountScope": "<64-hex>" } }   // accountScope OPTIONAL but load-bearing
```

[VERIFIED: zcode-src 3.17.1 index.ts:2888-2906 — `requestAuth` object keys
`apiKey` nonEmptyString.optional, `apiKeyId` min1.optional, `accountScope`
64-hex.optional, `headers` record.optional; the desktop host's answer shape at
accountProviderRequestAuthService.ts:111-115]. `accountScope` is what enables
the backend's one-shot rejected-PAT → refresh → retry recovery (ref comment:
"accountScope enables the backend's one-shot 'PAT rejected → refresh → retry'
recovery after a model 401") [VERIFIED: ref server-requests.ts:1448-1456].

**Mechanics:** the current seam
(`interact.RuntimeHeadersKeySource (ctx, req) → (apiKey, apiKeyID, err)`) does
not expose accountScope — Phase 6 widens the seam (or adds a mode-aware
success-shape builder in hostsvc) so the coding-plan source returns all three
fields; the api-key arm keeps its two-field answer byte-compatibly. The source
selects by auth mode recorded at authenticate (one field on the Front/hostsvc
binding; D-11 shared row). The `rejectedProjectTokenFingerprint` re-ask maps
to D-05: if it matches the cached fingerprint → drop, re-mint exactly once,
answer; a scope mismatch (`expectedAccountScope` ≠ current scope) → the
honest coded error (never serve a stale token — ref
`project_token_scope_invalidated` parity) [VERIFIED: ref
project-token.ts:367-392; zcode-src client assertCurrent :130-143].

**Never `headersApplied:false`** (PITFALL 3 carries): on mint failure the
wrapper answers the coded ERROR envelope (the Phase-5 responder posture
already implemented). Note the reference chooses `{headersApplied:false,
errorMessage}` so the backend's -32031 carries a friendly message — a valid
upstream choice we deliberately do NOT adopt (our posture predates and is
test-locked) [VERIFIED: ref server-requests.ts:1462-1467,1492-1502; in-repo
responder.go:421-449].

### Pattern 5: The proactive keyring probe (AUTH-04, D-02)

Probed failure modes this session (real binary, this machine — V-EXP
2026-11-22):

| Environment | Behavior | Latency |
|---|---|---|
| No D-Bus session env (scrubbed env) | stderr `secret-tool: Cannot autolaunch D-Bus without X11 $DISPLAY`, rc=1 | fast |
| D-Bus present, no matching entry | empty stdout, rc=1 | fast |

Neither hangs. The probe therefore is: (1) `exec.LookPath("secret-tool")` —
absent → the coded error naming the package (Ubuntu: `libsecret-tools`);
(2) run the exact `LookupCommand` argv under a bounded context (the loader's
scrubbed env; recommend ≤ 5 s — belt-and-braces against a locked-keyring
prompt path nobody can answer headless [ASSUMED: locked-keyring prompting
behavior not probed — no keyring-lock state available on this machine; the
bounded ctx contains it regardless]); (3) empty output / rc≠0 → the coded
actionable error naming the missing piece: no binary → install hint; no bus →
"no desktop session"; no entry → "log in via the ZCode desktop first". The
probe runs BEFORE any snapshot push (D-02) inside the authenticate arm; CI
tiers never construct the production loader (the Phase-5 posture stands — the
probe is a front-authenticate step on the real path, exercised at the fake
tier via an injected probe seam).

### Pattern 6: configOptions advertisement + MODEL-01 build

**Advertisement:** `initializeResponse` gains
`SessionCapabilities: {Options: &acp.SessionOptionsCapabilities{}}` — the
empty-struct opt-in. Clients gate `session/set_config_option` on
`sessionCapabilities.options` (SDK doc comments: "Clients MUST only send this
when the Agent advertises 'sessionCapabilities.options'") [VERIFIED: acp-go-sdk
v0.20.0 types_gen.go:8604-8664 (SessionCapabilities.Options
*SessionOptionsCapabilities), :8990-9000 (capability doc comment); CITED:
agentclientprotocol.com/protocol/v1/session-setup — capability-gating pattern;
the config-options page itself specifies only the client-side boolean
advertisement]. Note the client side: boolean options require
`clientCapabilities.session.configOptions.boolean` — v1 ships NO boolean
options, so no client capability read is needed [CITED:
agentclientprotocol.com/protocol/v1/session-config-options — "Agents MUST NOT
include type:boolean options unless the Client advertised support"].

**The backend source (D-08/D-09):** the create/setModel/setThoughtLevel reply
`settings` snapshot (schema, strict):

```
settings.model:         { current?: {providerId, modelId, options?:{reasoningLevel?}},
                          available: [ {ref:{providerId,modelId}, label, providerLabel?,
                                        description?, contextWindow?, maxOutputTokens?,
                                        reasoning?: {levels:[{value,label,description?}],
                                                    defaultLevel?},
                                        properties, disabledReason? } ],
                          lastUsed?: … }
settings.thoughtLevel:  { enabled: bool, current?: string, defaultLevel?: string,
                          available: [{value, label, description?}] }
```

[VERIFIED: zcode-src 3.17.1 index.ts:1177-1210 (`zcodeThoughtLevelOptionSchema`,
`zcodeModelReasoningOptionsSchema`, `zcodeModelOptionSchema`), :1306-1330
(`zcodeSessionSettingsStateSchema`); the create reply carrying
`settings.model.available` is V-EXP on desktop 3.18.2 — EXP-APIKEY-CYCLE-02/03
and dialect §7. The `settings.thoughtLevel` block in the live create reply is
schema-certain but not yet capture-quoted — the Phase-6 real run confirms
[ASSUMED with schema+V-EXP corroboration]]

**The translation (prescriptive):**

| configOption | id | category | type | options (verbatim mirror) | currentValue |
|---|---|---|---|---|---|
| model | `"model"` | `acp.SessionConfigOptionCategoryModel` ("model") | select | one entry per `settings.model.available`: `Value: ref.providerId+"/"+ref.modelId`, `Name: label`, `Description: description` (D-08: no filtering/renaming — include `disabledReason`-carrying entries as reported or as the backend reports them available) | the `providerId/modelId` serialization of `settings.model.current` when bound; never a guess |
| thought level | `"thought_level"` | `acp.SessionConfigOptionCategoryThoughtLevel` ("thought_level") | select | one entry per level source: the bound model's `reasoning.levels` (D-09 — per-model metadata as reported), falling back to `settings.thoughtLevel.available` when the bound model carries none; `Value: level.value`, `Name: level.label`, `Description: level.description` | `settings.thoughtLevel.current` when set and `enabled`; absent → no currentValue arm is synthesized (see honesty note) |

`currentValue` honesty (D-10): the value comes ONLY from the server-confirmed
snapshot — the created session's reported current (after the setModel
follow-up, from the SETMODEL reply's settings, which is the confirmed state)
or the mutation reply. An unconfirmed target is never shown selected. When
`current` is absent (freshly created, unbound session), the model option's
`currentValue` may be empty-string only if the schema allows — it is REQUIRED
on the wire, so the honest mechanism is: run the configured setModel follow-up
(Phase-4 landed) or bind the backend's effective current from
`settings.model.current`/`lastUsed` — and when neither exists, the picker
still renders because `currentValue` equals one of the listed values only
after confirmation; until then the wrapper omits the model entry's `options`
confusion by leaving `currentValue` at the backend-reported current, and if
the backend reports none the wrapper answers configOptions without the model
entry (a smaller honest list beats a fabricated selection) [ASSUMED: exact
client rendering of an empty-currentValue — spec says every option's
currentValue is required, so omitting the entry until a current exists is the
spec-safe reading; planner may prefer binding-at-create instead].

**Wire assembly:** `acp.NewSessionResponse{SessionId, ConfigOptions:
[]acp.SessionConfigOption}` — the select variant
`acp.SessionConfigOptionSelect{Id, Name, Category: &cat, CurrentValue,
Options: &acp.SessionConfigSelectOptions{Ungrouped: &opts}, Type: "select"}`
(the MarshalJSON injects `"type":"select"`); options entries
`acp.SessionConfigSelectOption{Value, Name, Description}` [VERIFIED:
acp-go-sdk v0.20.0 types_gen.go:8774-8806 (select), :9078-9130 (options
union), :6686-6700 (NewSessionResponse.ConfigOptions)].

### Pattern 7: `session/set_config_option` → backend mutations (MODEL-02)

**Backend methods (the CONTEXT discretion item resolved): BOTH exist and are
distinct.**

- `session/setModel` params `{sessionId, model: {providerId, modelId,
  options?: {reasoningLevel?}}, expectedRevision?, persistAsWorkspaceLastUsed
  (default true)}` strict [VERIFIED: zcode-src 3.17.1 index.ts:2396-2404]
- `session/setThoughtLevel` params `{sessionId, thoughtLevel?: string,
  expectedRevision?, persistAsWorkspaceLastUsed (default true)}` strict; empty
  thoughtLevel → `-32602 "thoughtLevel is required"` [VERIFIED: zcode-src 3.17.1
  index.ts:2406-2414; server-operations.ts:2812-2815]
- BOTH replies = `afterStateMutation(...)` = the FULL session snapshot
  (`snapshot(context, record, undefined, {modelAvailability:"current"})`) and
  both emit a `state.updated` patch (reason `model_changed` /
  `thought_level_changed`) carrying the new settings [VERIFIED: zcode-src 3.17.1
  server-operations.ts:2807-2838 (quoted), :4212-4233 (afterStateMutation
  quoted)] — **the reply IS the D-10 confirmation source**.
- Field-proven binding path on 3.18.2: create bare → `session/setModel` WITH
  `options.reasoningLevel` (the create path drops the level; a level-strict
  model refuses without it — EXP-TURN-01). `session/setThoughtLevel` has never
  been exercised live [ASSUMED with source corroboration].

**Mapping (prescriptive):**

| ACP ask | Backend call | Wire params |
|---|---|---|
| `configId:"model"`, value `<providerId>/<modelId>` | `session/setModel` | `{sessionId, model:{providerId, modelId, options?}, persistAsWorkspaceLastUsed:false}` — `options.reasoningLevel` rides when the CURRENT confirmed level exists (switching model must not silently drop the level — Pitfall 2) |
| `configId:"model"`, value also carrying a level (see below) | `session/setModel` | same, with the level from the value |
| `configId:"thought_level"`, value `<level>` | `session/setModel` with `options.reasoningLevel` (recommended; field-proven) — or `session/setThoughtLevel` (planner locks; Open Question 2) | per the chosen method |

**Value format decision:** ACP `SessionConfigValueId` is an opaque string. The
model option's values are `providerId/modelId` (the serialization this repo
already uses for read-back want lists and the config `--model` flag). A
combined model+level value form exists in the backend's own UI picker —
`providerId/modelId$level` with `ZCODE_MODEL_REASONING_SEPARATOR = "$"`
(explicitly "display-only, not a reversible serialization") [VERIFIED:
zcode-src 3.17.1 model-selection.ts:27-40]. Recommendation: keep the two
options separate (model option values = `providerId/modelId` exactly as
mirrored; thought-level option values = the backend's own level.value strings)
— no `$` composite values; a model switch carries the current confirmed level
in `options` and the thought-level option re-renders from the reply. This
keeps D-08's verbatim-mirror rule clean.

**The handler contract:** validate first — unknown `configId` / value not in
the option's own listed set → `-32602` invalid-params naming the configId and
the allowed set (CONTEXT discretion: "coded invalid-params … never a crash
path"; the spec: the value "MUST be one of the values listed in the option's
options array" [CITED: agentclientprotocol.com/protocol/v1/session-config-options]).
Then the backend call; then rebuild configOptions FROM THE REPLY's settings
(the same translator as MODEL-01) and answer
`acp.SetSessionConfigOptionResponse{ConfigOptions: <full list>}` — the SDK
VALIDATES the response has a non-nil list (`Validate(): configOptions is
required`) and the spec: the Agent "MUST respond with the complete list of all
configuration options and their current values" [VERIFIED: acp-go-sdk v0.20.0
types_gen.go:11582-11630; CITED: session-config-options page]. An
optional-asymmetric variant: also emit the `config_option_update` session
update (full list) after applying — the spec example channel for
agent-initiated changes; the response already carries the full list, so the
notification is redundant for a client-initiated change — v1 skips it [CITED:
session-config-options — notification "also contains the complete
configuration state"].

**Adapter seam:** `agentAdapter.SetSessionConfigOption` (currently the clean
-32601 default, agent.go:388-394) delegates to the Handlers interface
exactly as Authenticate did in Phase 5; `initializeResponse` gains the
capability advertisement. The neutral vocabulary (`acpport.ConfigOption…`)
keeps SDK types out of the front (D-01 discipline). The front needs the
per-session confirmed settings state — store the latest confirmed settings
snapshot per session id (written at NewSession/setModel/mutation replies;
bounded with the registry, the WR-07 discipline).

### Pattern 8: Cipher asks during authenticate (arrival-time parity)

The reference serves `interaction/requestCredentialCipher` in BOTH paths —
the turn-loop queue AND at frame arrival (asks with no active turn) — with the
requestId-correlated `{requestId, result:{ok…}}` reply; a non-servable op
declines so the backend keeps (never deletes) the stored login [VERIFIED: ref
server-requests.ts row 2 + docs/research/reference-adapters.md row 2]. Our
responder is arrival-time by construction (the conn's OnRequest is always
wired before traffic) — the decrypt seam answers whenever the ask lands,
authenticate or mid-turn. **Encrypt ops stay declined** (Phase-5 posture).
Source verification of the discretion item "coding-plan auth never requires
host-side encryption": the desktop's own auth-path cipher use is decrypt of
its stored credentials (the store is the read source; the mint chain writes
nothing local — the location/token stay in memory); the reference's
`credentialCipherOp` encrypt arm exists only for the round-trip test parity,
not for auth [VERIFIED: ref credential-store.ts:196-219 comment + the mint
chain having no local write]. If a real run ever fires an encrypt ask
mid-auth, the honest decline surfaces as a coded auth failure — observed, not
assumed, in the capture.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PAT mint HTTPS client frameworks | An OAuth/REST client library | stdlib net/http with explicit `http.Client{Timeout}` per call | Three pinned calls; a library guesses shapes the reference pins byte-exactly (STACK.md decision) |
| Credential decrypt | Any custom cipher beyond the verified layouts | stdlib AES-256-GCM over `enc:v1:`/`enc:v2:os:` exactly as quoted (Pattern 1) | Both layouts field/source-verified; the kid/fp24 checksums are the key-selection contract |
| Registry verdict | Polling `session/messages` or trusting the ack | `hostsvc.WatchRegistry/Await` + probeCreateReadback | PITFALL 1 machinery is landed and field-proven (EXP-READBACK-DIAG-01) |
| ACP config-option wire assembly | Hand-marshaled session/update maps | SDK `SessionConfigOptionSelect`/`SessionConfigSelectOption` + the adapter translation | The SDK owns discriminant injection (`type:"select"`) and the required-list Validate |
| settings dig-out | Ad-hoc JSON path strings scattered per call | One typed dig-out helper (the `digCreateAvailable` pattern extended to `thoughtLevel`) | The readback.go dig-out discipline is capture-verbatim camelCase and test-locked |
| Model-option value parsing | Regex freestyling on `a/b$c` composites | `strings.Cut` on the first "/" only; no `$` composite values | The `$` separator is display-only (model-selection.ts comment) — adopting it invites irreversible-value bugs |

**Key insight:** Phase 6 is composition over verified primitives plus two
genuinely new external surfaces (api.z.ai HTTPS; credentials store read).
Everything protocol-shaped — snapshot push, read-back, responder rows, SDK
delegation — already exists in test-locked form from Phases 3-5.

## Runtime State Inventory

Not a rename/refactor/migration phase — omitted per template. (The phase
READS desktop runtime state — credentials.json, setting.json, the builtin
table — but writes nothing to any store; the wrapper remains read-only toward
the desktop, and the PAT stays in-process per D-04.)

## Common Pitfalls

### Pitfall 1: Trusting the `received` ack on the account overlay
**What goes wrong:** a schema-accepted snapshot with a stale
`basedOnZCodeBuiltinRevision` is silently discarded; the registry keeps
serving the previous registry; auth "succeeds" while account models never
appear. **Why:** the protocol has no rejection channel for stale snapshots
(dialect §8 warning-level fact). **How to avoid:** the Phase-5 chain verbatim —
watcher registered BEFORE push traffic, refresh, probe-create read-back, Await
verdict. **Warning signs:** authenticate ok but `session/new` shows no
`account:*` models.

### Pitfall 2: A model switch that silently drops the reasoning level
**What goes wrong:** the client picks a thinking model; the wrapper sends
`setModel` without `options.reasoningLevel`; the turn (or the switch itself,
on level-strict providers) refuses `-32603 "Reasoning level is required…"`.
**Why:** the level is not implied by the model id; the create path drops it
and so does a naive setModel. **How to avoid:** every model mutation carries
the CURRENT confirmed level in `options` (from the last server-confirmed
snapshot) unless the ask itself changes it; the thought-level picker changes
travel with `options.reasoningLevel` too (or via setThoughtLevel — planner
locks). **Warning signs:** "Reasoning level is required" in the real-run
capture; fake corpus vectors asserting the options object.

### Pitfall 3: `headersApplied:false` or a PAT-less success arm
**What goes wrong:** guaranteed `-32031` turn death for account providers.
**Why:** the protocol offers no downgrade; a wrapper that "honestly" declines
kills the turn. **How to avoid:** the coded ERROR envelope on mint failure
(landed); the success arm carries token+id+accountScope. **Warning signs:**
4001/4011 handshake codes in provider logs; instant turn failures only on
`account:*`.

### Pitfall 4: PAT or login-token material reaching disk, logs, or captures
**What goes wrong:** hygiene violation (D-04/T-5-17 extension); scrubbed
captures fail scan-secrets; credentials leak via slog fields. **Why:** the
mint chain handles four secret classes (login token, zcode JWT, PAT, keyId).
**How to avoid:** in-process only; structural logging (ids/shapes/lengths,
never values); the scrubber + scan-secrets gates run on every evidence commit
(D-13 discipline); the PAT cache struct carries no `String()`/log friendly
rendering. **Warning signs:** scan-secrets hits; capture diff shows
`eyJ…` shapes.

### Pitfall 5: The keyring probe hanging a turn (AUTH-04 regression)
**What goes wrong:** a probe (or key load) without a deadline blocks on a
prompt nobody can answer (locked keyring / headless bus autolaunch); the
client sees a dead authenticate. **How to avoid:** D-02's probe order (BEFORE
any push), LookPath first, bounded ctx on every exec + every HTTPS call;
failure = the coded actionable error naming the missing piece. **Warning
signs:** authenticate latency > probe bound; CI timeouts in auth tests.

### Pitfall 6: Building configOptions from anything but the backend's reply
**What goes wrong:** wrapper-side filtering/enrichment (D-08 violation) or a
`currentValue` that guesses (D-10 violation) — the client picks a model the
backend refuses, or sees a selection that is not real. **Why:** it is tempting
to "improve" labels or preselect the configured default. **How to avoid:** the
translator consumes ONLY `settings` snapshots from replies; the configured
`--model` default influences the setModel FOLLOW-UP (which the backend
confirms), never the rendered list. **Warning signs:** currentValue ≠ the
backend's `model.current` in any test vector.

### Pitfall 7: Mint-chain env traps (the Node lessons, Go edition)
**What goes wrong:** a dead `HTTPS_PROXY` breaks all api.z.ai calls for the
process lifetime (`ProxyFromEnvironment` caches at first use); custom dialers
can reintroduce IPv6-first hangs. **How to avoid:** explicit transport config
at minter construction; log the effective proxy once; prefer the default
Happy-Eyeballs dialer; the mint chain runs on the WRAPPER's env (the child env
scrub does not apply here — the wrapper itself calls api.z.ai) [VERIFIED:
PITFALLS.md Pitfall 9 + in-repo loader scrub precedent]. **Warning signs:**
mint timeouts while curl works.

### Pitfall 8: Location cache invalidation missed on key 404
**What goes wrong:** the key was deleted/moved; every subsequent mint fails
against the stale `{org, project, keyId}`. **How to avoid:** port the
reference's exact rule: a `code:404`/HTTP-404 on access_tokens drops the
cached location (re-resolve next ask); permission/network errors never trigger
re-create [VERIFIED: ref project-token.ts:321-325; zcode-src client
:225-229 comment "仅明确不存在才废弃 ID"]. **Warning signs:** repeated
`project_token_key_missing` after a console-side key delete.

## Code Examples

### The PAT minter core shapes (Go, stdlib)

```go
// Source: ref v0.89.0 project-token.ts:229-296 + zcode-src 3.17.1
// project-access-token-client.ts:265-351 (both quoted in Pattern 3).
// getCustomerInfo: RAW token, NO Bearer prefix.
req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
    origin+"/api/biz/customer/getCustomerInfo", nil)
req.Header.Set("Authorization", loginToken) // raw — a Bearer is misread as login-invalid
req.Header.Set("Content-Type", "application/json")

// api_keys list + create-if-absent (D-07): Bearer form for zai.
// list:  GET  {origin}/api/biz/v1/organization/{org}/projects/{proj}/api_keys
// create: POST same URL, body {"name":"zcode-api-key","usageScene":1}
req2, _ := http.NewRequestWithContext(ctx, http.MethodPost, keyListURL,
    strings.NewReader(`{"name":"zcode-api-key","usageScene":1}`))
req2.Header.Set("Authorization", "Bearer "+loginToken)
req2.Header.Set("Content-Type", "application/json")

// access_tokens: Bearer form; body carries the DESKTOP's version.
req3, _ := http.NewRequestWithContext(ctx, http.MethodPost,
    keyListURL+"/"+url.PathEscape(keyID)+"/access_tokens",
    strings.NewReader(fmt.Sprintf(`{"clientType":"zcode","clientVersion":%q}`, desktopVersion)))
req3.Header.Set("Authorization", "Bearer "+loginToken)
req3.Header.Set("Content-Type", "application/json")
// reply data: {enable?, accessToken, tokenType:"Bearer", expiresIn, expiresAt}
```

### accountScope (byte-parity with desktop + reference)

```go
// Source: ref project-token.ts:113-118 == zcode-src accountProviderRequestAuthService.ts:118-135.
payload, _ := json.Marshal([]any{family, "individual-coding-plan", nil, loginToken, zcodeJwtOrNil})
sum := sha256.Sum256(payload)
scope := hex.EncodeToString(sum[:]) // 64-hex; rides requestAuth.accountScope
```

### enc:v1 fallback decrypt (Go)

```go
// Source: ref credential-store.ts:33-68 (resolveCredentialSecret + decryptCredential).
func v1Secret(env func(string) string, home, username, platform string) string {
    if s := strings.TrimSpace(env("ZCODE_CREDENTIAL_SECRET")); s != "" { return s }
    return "zcode-credential-fallback:" + platform + ":" + home + ":" + username
}
// value: "enc:v1:" + strings.Join([b64url(iv12), b64url(tag16), b64url(ct)], ".")
// key = sha256(secret); plaintext entries (no prefix) pass through verbatim;
// unknown "enc:" prefixes are skipped, never served as plaintext.
```

### configOptions translator (settings snapshot → SDK types)

```go
// Source: acp-go-sdk v0.20.0 types_gen.go:8774-8897 (select variant assembly);
// settings schema zcode-src index.ts:1306-1330 (Pattern 6).
func modelOption(s settingsSnapshot) acp.SessionConfigOption {
    cat := acp.SessionConfigOptionCategoryModel
    opts := make([]acp.SessionConfigSelectOption, 0, len(s.Model.Available))
    for _, m := range s.Model.Available {
        opts = append(opts, acp.SessionConfigSelectOption{
            Value: m.Ref.ProviderID + "/" + m.Ref.ModelID, Name: m.Label, // verbatim mirror, D-08
        })
    }
    cur := currentModelValue(s) // ONLY s.Model.Current serialized (D-10); "" when unbound
    return acp.SessionConfigOption{Select: &acp.SessionConfigOptionSelect{
        Id: "model", Name: "Model", Category: &cat, Type: "select",
        CurrentValue: acp.SessionConfigValueId(cur),
        Options:      acp.SessionConfigSelectOptions{Ungrouped: &opts},
    }}
}
```

### The coding-plan authenticate arm (composition sketch)

```go
// Source: composition of in-repo auth.go authenticateAPIKey (:125-178) with
// Pattern 5 (probe) + Pattern 2 (overlay snapshot) + Pattern 3 (mint).
func (f *Front) authenticateCodingPlan(ctx context.Context) error {
    if err := f.auth.Probe.Keyring(ctx); err != nil { // D-02: BEFORE any push
        return acpport.NewInternal("authenticate: keyring unavailable: %v", err)
    }
    login, err := f.auth.Creds.LoginToken(ctx, "zai") // store reader (D-03)
    if err != nil {
        return acpport.NewInternal("authenticate: no desktop login (log in via the ZCode desktop first): %v", err)
    }
    watchOpts := ... // identical read-back setup as the api-key arm (auth.go:137-140)
    watcher, remove := hostsvc.WatchRegistry(f.conn, f.auth.WantModels, watchOpts...)
    defer remove()
    snap, err := f.auth.Snapshot.BuildAccount(f.auth.Install, login) // Pattern 2 overlay
    if err != nil { return acpport.NewInternal("authenticate: build snapshot: %v", err) }
    if _, err := hostsvc.PushEnvelope(ctx, f.conn, snap.Params); err != nil { ... }
    if err := hostsvc.RefreshRegistry(ctx, f.conn); err != nil { ... }
    if !watcher.Applied() { f.probeCreateReadback(ctx, watcher) } // landed branch-(c) arm
    if _, err := watcher.Await(ctx); err != nil {                 // verdict = registry, never ack
        return acpport.NewInternal("authenticate: registry read-back: %v", err)
    }
    if _, err := f.auth.Pat.Ensure(ctx, login); err != nil {      // D-04 mint-once
        return acpport.NewInternal("authenticate: PAT mint: %v", err)
    }
    f.authMode.Store(authModeCodingPlan) // the runtime-headers source reads this (D-11)
    return nil
}
```

### Fake scenario seed (cipher ask + runtime-headers during the auth cycle)

```json
{
  "name": "coding-plan-auth-cycle",
  "on_request": [
    {"method": "provider/updateAccountConfig",
     "reply": {"result": {"receivedRevision": "account-bridge:t1", "providerCount": 1, "status": "received"}}},
    {"method": "provider/refreshRegistry",
     "reply": {"result": {"refreshed": true}},
     "side_effects": [
       {"emit": {"method": "state.updated",
                 "params": {"scope": "registry", "reason": "registry_refresh", "revision": 3,
                            "patch": {"providers": {"account:zai-individual-coding-plan",
                                                  {"models": ["GLM-5.3", "GLM-5.3-Flash"]}}}}}}
     ]},
    {"method": "session/create",
     "reply": {"result": {"session": {"sessionId": "sess_cp1"},
                          "settings": {"model": {"current": {"providerId": "account:zai-individual-coding-plan", "modelId": "GLM-5.3"},
                                                 "available": [{"ref": {"providerId": "account:zai-individual-coding-plan", "modelId": "GLM-5.3"}, "label": "GLM-5.3",
                                                                              "reasoning": {"levels": [{"value": "high", "label": "High"}, {"value": "medium", "label": "Medium"}], "defaultLevel": "high"}}]},
                                      "thoughtLevel": {"enabled": true, "current": "high", "defaultLevel": "high",
                                                       "available": [{"value": "high", "label": "High"}, {"value": "medium", "label": "Medium"}]}}}},
     "side_effects": [
       {"reverse": {"method": "interaction/requestCredentialCipher",
                    "params": {"requestId": "cipher-1", "operation": "decrypt", "value": "enc:v2:os:kid24:iv.tag.ct"},
                    "timeout_ms": 15000,
                    "expect_answer": {"result": {"ok": true, "value": "<decrypted>"}}}},
       {"reverse": {"method": "interaction/requestProviderRuntimeHeaders",
                    "params": {"requestId": "hdr-1", "sessionId": "sess_cp1",
                               "modelSelection": {"providerId": "account:zai-individual-coding-plan", "modelId": "GLM-5.3"},
                               "providerId": "account:zai-individual-coding-plan", "reason": "model-request"},
                    "timeout_ms": 15000,
                    "expect_answer": {"result": {"headersApplied": true,
                                                 "requestAuth": {"apiKey": "<PAT>", "apiKeyId": "<kid>",
                                                                 "accountScope": "<64hex>"}}}}}
     ]},
    {"method": "session/setModel",
     "reply": {"result": {"session": {"sessionId": "sess_cp1"},
                          "settings": {"/* full snapshot after the switch */"}}}}
  ],
  "eof_drain_ms": 100
}
```

(The fake's `expect_answer` semantic compare needs literal values for the PAT
arm — the corpus injects a scripted KeySource/PathSource seam, matching the
05-07 test-injection discipline; secret shapes stay synthetic.)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Bridge reads plan api-keys from the credential store (`account-provider:coding-plan:…:api-key`) | Desktop 3.15+ stores NO plan api-key as the auth source — the host trades the OAuth login for a short-lived PAT per request | 3.15+ (ref project-token.ts header) | The mint chain is the official path; store api-key entries are legacy remnants |
| Reference v0.89.0 read-only posture (never create keys) | D-07 create-if-absent (user decision, desktop behavior) | locked 2026-11-22 | POST api_keys create arm lands; `project_token_key_missing` becomes the race/failure arm, not the primary path |
| Phase-5 empty snapshot overlay | Phase-6 account-provider overlay mirrored from the builtin table | this phase | Account models materialize headless; entitlement is wrapper-derived |
| ACP mode/mode-state selectors | v1 session config options stabilized (`configOptions`, set_config_option, config_option_update, categories incl. model/thought_level) | schema 1.25.0 in SDK v0.20.0 | The whole picker is expressible without SDK changes |
| `session/setModel` as the only binding path | setModel + setThoughtLevel both exist; level-carrying setModel is the field-proven path | EXP-TURN-01 vs source | Planner locks the level-switch wire method (Open Question 2) |

**Deprecated/outdated:** none new. The dialect drift baseline stands
(3.18.2 bundle pristine; `requestCredentialCipher` remains 3.18-only).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The z.ai PAT's actual TTL is not publicly documented; the reply's `expiresIn`/`expiresAt` + 120 s refresh lead govern refresh, and D-05's reactive re-mint makes the number immaterial | Pattern 3 | If the server enforces a hard refresh cadence our reactive posture misses, turns fail with the honest coded error and re-mint — recoverable |
| A2 | Create-if-absent is race-tolerant: check-then-create can create a duplicate `zcode-api-key` under concurrent first-auths (single-user wrapper: acceptable; the candidate search precedes every mint) | Pattern 3 | A duplicate key on the account (cosmetic; the list-first search prefers the existing name next time) |
| A3 | A LOCKED (password-prompting) keyring cannot hang the probe — bounded ctx contains it; the prompting behavior itself was not probed (no locked-keyring state on this machine) | Pattern 5 | A slow prompt path adds probe latency up to the bound; the coded error still surfaces |
| A4 | `settings.thoughtLevel` is present in the LIVE 3.18.2 create/setModel replies (schema-certain in 3.17.1; the live captures quoted `settings.model.available` explicitly) | Pattern 6 | If the live snapshot omits the block, the thought-level option falls back to per-model `reasoning.levels` (also backend-reported); the real run confirms before the phase closes |
| A5 | `session/setThoughtLevel` behaves live as its 3.17.1 schema (source-verified only; never exercised on 3.18.2) | Pattern 7 / OQ2 | If it drifts, the setModel-with-options path (field-proven) remains the fallback the planner can lock instead |
| A6 | `clientVersion` in the mint body tolerates the desktop's version string (reference sends the desktop version; zcode-src sends the CLI's own; no server-side validation documented) | Pattern 3 | A rejected clientVersion surfaces as a mint business error — visible, diagnosable, retryable with the other value |
| A7 | An empty `currentValue` cannot be answered for a required field — the wrapper omits the model entry until the backend confirms a current (or binds at create via the landed follow-up) | Pattern 6 | Clients render a model list without a preselection until the first confirmation — honest, spec-safe |
| A8 | Client-side (Orca) config-options rendering resolves model-selectors from version ≥ 18.1.9 per the Phase-1 card study; our contract is spec-honest regardless of any one client | Pattern 6 | Purely client-side rendering risk; Phase 7 live E2E covers it |
| A9 | The responder's seam widening (accountScope) can extend `RuntimeHeadersKeySource` without breaking the 05-06/05-07 test locks (the api-key two-field answer stays byte-identical) | Pattern 4 | A signature change touches responder tests — red-first update, same locks |

## Open Questions

1. **Which wire method carries a level-only change** (CONTEXT discretion;
   researcher resolves the facts, planner locks the choice)
   - What we know: both `session/setModel {model:{…, options:{reasoningLevel}}}`
     and `session/setThoughtLevel {thoughtLevel}` exist; setModel+options is
     field-proven on 3.18.2; setThoughtLevel is source-only (A5).
   - Recommendation: lock `session/setModel` with `options.reasoningLevel` for
     BOTH model switches and level-only changes in v1 (one proven path; the
     reply snapshot confirms either way); add `MethodSessionSetThoughtLevel`
     only if the planner prefers the dedicated method behind a corpus vector.
2. **Whether the model picker entry exists when no current is confirmed** (A7)
   - What we know: `currentValue` is required on the wire; the wrapper may run
     the configured setModel follow-up at create (landed) which CONFIRMS a
     current when configured; with no configured model the backend reports its
     own current.
   - Recommendation: rely on the backend-reported current (the follow-up only
     when configured); omit the model entry only in the (unlikely) no-current
     reply case; planner may instead always send a minimal bind. Needs a
     planner decision because it touches the no-`--model` launch posture.
3. **Probe-seam shape for AUTH-04 at the fake tier**
   - What we know: the probe is production-only (secret-tool); fake-tier tests
     need an injected probe result seam.
   - Recommendation: a `KeyringProber` func type on the auth config (default =
     the real prober; tests inject canned verdicts) — mirrors the KeyLoader
     injection discipline.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | everything | ✓ | 1.27.3 (mise-pinned) | — |
| mise + golangci-lint | gates | ✓ | 2.15.0 pinned | — |
| Real ZCode install | guarded real tier (D-12) | ✓ | 3.18.2 squashfs-root, pristine | none — the real run IS the phase bar |
| Desktop login (oauth:zai:access_token in the store) | coding-plan real tier | ✓ (probe verified this session — structure, values never read) | — | coded "log in via the ZCode desktop first" error (D-12 posture) |
| `secret-tool` | keyring probe + v2 key load | ✓ (/usr/bin/secret-tool) | libsecret CLI | AUTH-04 coded error naming libsecret-tools |
| gnome-keyring / D-Bus session | keyring reads | ✓ (desktop session) | — | AUTH-04 fail-fast (probed rc=1 arms) |
| api.z.ai reachability | PAT mint | ✓ (same host the desktop uses) | — | coded mint error; reactive retry per D-05 |
| godbus | deliberately absent (D-01 Phase-7 swap) | ✗ | — | out of scope |

**Missing dependencies with no fallback:** none for CI/fake tiers. The real
tier requires the desktop login + keyring + network (all present on this
machine; the phase bar runs here per D-12).

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `-race`; fake corpus (scenario JSON + `.golden` byte-lock); table-driven TestGolden; guarded real tier (`//go:build realdesktop` + `ZCODE_ACP_REAL_E2E=1`) |
| Config file | none — canonical gate order build→vet→lint→test→test-experiments via mise (Phase-2 lock) |
| Quick run command | `go test -race ./internal/interact/... ./internal/hostsvc/... ./internal/acpfront/... ./internal/acpport/... ./internal/zcode/...` |
| Full suite command | `mise run ci` |
| Guarded real tier | canonical command in the file header (apicycle_test.go:18 precedent); manual-only, human-gated checkpoint (05-14 precedent, AUTONOMOUS:false) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-04 | probe arms: no binary / no bus / no entry → coded actionable errors, fast | unit (injected prober) | `go test -race ./internal/acpfront/ -run TestCodingPlanProbe` | ❌ Wave 0 |
| AUTH-04 | probe-before-push ordering (no snapshot frame on probe failure) | fake golden | `go test -race ./internal/zcode/fake/ -run TestGolden/coding-plan-probe-fail` | ❌ Wave 0 |
| AUTH-02 | store reader: v2:os + v1 fallback + plaintext + unknown-prefix skip; per-entry isolation | unit (synthetic blobs) | `go test -race ./internal/interact/ -run TestCredStore` | ❌ Wave 0 |
| AUTH-02 | v1 decrypt parity vectors (round-trip + malformed + wrong-secret) | unit | `go test -race ./internal/interact/ -run TestV1Fallback` | ❌ Wave 0 |
| AUTH-02 | account overlay snapshot shape (providers/states entries, exact-revision formula reused) | unit | `go test -race ./internal/hostsvc/ -run TestAccountSnapshot` | ❌ Wave 0 |
| AUTH-02 | mint chain: customerInfo bare-token / personal pick / list / create-if-absent / access_tokens / scope / 404-invalidate — httptest server | unit (transport seam) | `go test -race ./internal/hostsvc/ -run TestPatMinter` | ❌ Wave 0 |
| AUTH-02 | runtime-headers PAT answer: token+id+accountScope; fingerprint re-ask → one silent re-mint; second failure → coded error | fake golden + unit | `go test -race ./internal/zcode/fake/ ./internal/interact/ -run TestRuntimeHeadersPat` | ❌ Wave 0 |
| AUTH-02/03 | full coding-plan authenticate cycle at the composed tier (probe→cred→push→read-back→mint) | front harness (fake) | `go test -race ./internal/acpfront/ -run TestCodingPlanAuth` | ❌ Wave 0 |
| AUTH-03 | api-key arm regression untouched (existing locks stay green) | existing suite | `go test -race ./internal/acpfront/ -run TestAuthenticate` | ✅ (05 locks) |
| MODEL-01 | create/setModel reply settings → configOptions (model + thought_level, verbatim mirror, honest currentValue) | unit + front harness | `go test -race ./internal/acpfront/ ./internal/acpport/ -run TestConfigOptions` | ❌ Wave 0 |
| MODEL-01 | sessionCapabilities.options advertisement in initialize | harness (tee byte-lock) | `go test -race ./internal/acpport/ -run TestInitCapabilities` | ❌ Wave 0 |
| MODEL-02 | set_config_option → setModel with options.reasoningLevel; full-list re-emit from the reply settings | fake golden + harness | `go test -race ./internal/zcode/fake/ ./internal/acpfront/ -run TestSetConfigOption` | ❌ Wave 0 |
| MODEL-02 | unknown configId / value-not-listed → -32602, never a crash | unit | `go test -race ./internal/acpfront/ -run TestSetConfigOptionInvalid` | ❌ Wave 0 |
| MODEL-02 | thought-level switch (chosen wire method) confirmed by the reply snapshot | fake golden | `go test -race ./internal/zcode/fake/ -run TestGolden/config-option-thought` | ❌ Wave 0 |
| D-04 | PAT never persisted: cache struct + wiring grep-lock / test asserting no disk writes | unit (hygiene lock) | `go test -race ./internal/hostsvc/ -run TestPatMemoryOnly` | ❌ Wave 0 |
| D-12/13 | real tier: coding-plan full cycle (auth → session/new → one tool-less tiny turn) + scrubbed capture + ledger row | guarded manual | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/` | ❌ (guarded, late-phase, human-gated) |

### Sampling Rate

- **Per task commit:** quick run command (touched packages, `-race`)
- **Per wave merge:** full suite (`mise run ci` gates)
- **Phase gate:** full suite green + the guarded real-tier evidence (D-12):
  scrubbed capture with V-EXP id, quota-ledger row in experiments/README,
  scan-secrets green on tree AND staged diff, human confirmation before the
  evidence commit (the 05-12/05-14 discipline verbatim)

### Wave 0 Gaps

- [ ] `internal/interact/credstore.go` + store/v1-fallback tests (AUTH-02 base)
- [ ] `internal/hostsvc/patminter.go` transport seam + httptest vectors (AUTH-02)
- [ ] account-snapshot builder extension + tests (AUTH-02)
- [ ] `KeyringProber` seam + probe-arm tests (AUTH-04)
- [ ] fake corpus scenarios: coding-plan-auth-cycle / coding-plan-probe-fail /
      runtime-headers-pat (incl. fingerprint re-ask) / config-option round-trips (+goldens)
- [ ] acpport neutral ConfigOption vocabulary + adapter delegation tests
- [ ] `zcode.MethodSessionSetThoughtLevel` constant (if OQ1 lands on the
      dedicated method)

## Security Domain

### Applicable ASVS Categories (security_enforcement=true, level 1)

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | yes | authenticate coding-plan arm: the desktop login IS the credential; no user-supplied secrets; honest coded errors; PAT minted via official endpoints only |
| V3 Session Management | yes | per-session confirmed-settings state bounded with the registry (WR-07 discipline); authMode recorded at authenticate, read by the headers source |
| V4 Access Control | yes | no auto-allow anywhere; encrypt cipher ops stay declined; the wrapper writes NOTHING to the desktop's stores (read-only parity) |
| V5 Input Validation | yes | set_config_option configId/value validated against the advertised option set (-32602); mint envelope codes validated (success-code set); builtin-table rules filtered by exact access.type |
| V6 Cryptography | yes | stdlib AES-256-GCM over the two verified layouts only; sha256 key derivations verbatim; kid/fp24 verification before decrypt; no hand-rolled crypto |
| V7 Secrets Logging | yes | PAT/login-token/zcodeJWT never logged or persisted (T-5-17 extension, test-locked); structural logs carry shapes/lengths; scrub-capture + scan-secrets gates on every evidence commit |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| PAT/credential leakage via logs, captures, or argv | Information Disclosure | in-process cache only (D-04); structural logging; loader argv carries the path-hash (never material); scrub gates |
| MitM / credential theft on the mint chain | Tampering / Information Disclosure | HTTPS-only official endpoints; explicit transport (Pitfall 7); no proxy env surprises; fixed error codes, response bodies never logged |
| Stale-snapshot downgrade (registry keeps old providers) | Tampering | PITFALL-1 read-back verdict; exact-revision formula computed at runtime |
| Scope confusion after re-login (stale PAT served) | Elevation of Privilege | accountScope echo check (`expectedAccountScope` mismatch → refuse + re-mint); fingerprint-matched invalidation |
| Keyring probing as a hang/DoS vector | Denial of Service | bounded ctx on every exec; probe-before-push; coded fast failures (probed rc=1 arms) |
| Duplicate key creation races on the account | Denial of Service (account hygiene) | check-then-create; single-flight mints; deterministic personal pick |

## Sources

### Primary (HIGH confidence)

- william0wang/zcode-acp **v0.89.0** clone (`/tmp/zcode-acp-ref-study/repo`,
  tag verified via git describe this session): `src/backend/credential-store.ts`
  (v1/v2 layouts, entitlement of prefixes, store reader), `src/config/project-token.ts`
  (mint chain, cache, scope, error codes, read-only divergence),
  `src/config/account-provider.ts` (snapshot payload, builtin-table mirror,
  entitlement), `src/handlers/server-requests.ts` (runtime-headers answer with
  accountScope; cipher serving both paths) — Apache-2.0, porting with
  attribution (docs/research/reference-adapters.md Attribution section:
  file headers + README notice + tag recorded in the porting PR)
- zcode-src **3.17.1** clone (`~/DiskD/W/Djarvur/zcode-src`, package.json
  version read this session; corroborating tier per dialect D-01):
  `packages/shared/src/project-access-token-client.ts` (desktop mint + CREATE
  shape + scope + location cache), `packages/shared/src/api-key-usage-scene.ts`
  (`CODING_PLAN: 1`), `packages/services/src/model-provider/accountProviderRequestAuthService.ts`
  (desktop answer material + scope), `packages/shared/src/zcode-protocol/index.ts`
  (settings/model/thoughtLevel schemas, setModel/setThoughtLevel params,
  runtime-headers params/response verbatim), `apps/zcode-cli/.../server-operations.ts`
  (setModel/setThoughtLevel replies = full snapshot via afterStateMutation),
  `packages/shared/src/model-selection.ts` (selection schema + `$` display separator)
- coder/acp-go-sdk **v0.20.0** module source (module cache, read this session):
  types_gen.go / constants_gen.go / agent_gen.go anchors quoted in Patterns 6-7
- In-repo (Read this session): internal/acpfront/{auth.go,front.go},
  internal/interact/{responder.go,cipherdecrypt.go}, internal/hostsvc/{snapshot.go,readback.go},
  internal/acpport/agent.go, cmd/zcode-acp/main.go,
  internal/zcode/fake/testdata/scenarios/auth-apikey.json, test/realdesktop/apicycle_test.go,
  experiments/README.md (quota ledger)
- Machine-local ground truth: FINDINGS.md §3.3/§3.4 (mint chain + create body +
  V4 signing stays in the official binary), the shim (secret-tool lookup shape)
- Live probes this session (V-EXP, machine-local): secret-tool failure modes
  (rc=1 fast, both arms); the revision-44 builtin table's 10 zhipu-account
  rules; setting.json `providerFamilyConnectionSelections`/`modelProviderFamilyModes`
  shapes; credentials.json entry KEY NAMES (values never read)

### Secondary (MEDIUM confidence)

- agentclientprotocol.com v1 spec: `/protocol/v1/session-config-options`
  (configOptions shape, set_config_option complete-list response,
  config_option_update, boolean client gating, currentValue semantics) and
  `/protocol/v1/session-setup` (capability-gating pattern) — cross-checked
  against the SDK source (two independent authoritative surfaces); CITED tier
  per the classify-confidence seam (webfetch LOW unverified → cited, not
  verified)

### Tertiary (LOW — flagged)

- None material; A1-A9 carry the residual uncertainty

## Metadata

**Confidence breakdown:**
- Mint chain / credential layouts / snapshot overlay: HIGH — reference v0.89.0
  + zcode-src 3.17.1 + FINDINGS all read and cross-agreeing this session;
  create-if-absent confirmed in two independent sources (desktop source + FINDINGS)
- ACP config-options surface: HIGH for the SDK tier (module source quoted);
  CITED for the spec pages (agent-side advertisement implied by SDK doc
  comments + capability-gating pattern; the config-options page text itself
  specifies the client-side gate)
- Live-desktop behavior (thoughtLevel block in replies, setThoughtLevel on
  3.18.2, PAT TTL): ASSUMED-with-corroboration — the D-12 real run is the
  designed confirmer, exactly as the Phase-5 read-back diagnosis was
- Test design / validation map: HIGH — extends proven Phase-3/4/5 machinery

**Research date:** 2026-11-22
**Valid until:** 2026-12-22 (stable domain — private dialect + pinned SDK +
pinned reference tag; re-check on any desktop version bump, SDK schema bump,
or reference-repo major bump)
