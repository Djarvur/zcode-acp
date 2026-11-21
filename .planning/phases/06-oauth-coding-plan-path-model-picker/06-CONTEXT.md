# Phase 6: OAuth Coding-Plan Path & Model Picker - Context

**Gathered:** 2026-11-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 6 delivers the **official OAuth individual coding-plan chain end-to-end**
(AUTH-02: exact-revision registry snapshot accepted — verdict read from registry
state, never the ack; `interaction/requestCredentialCipher` answered by decrypting
the desktop's keyring-stored credential; PAT minted via the official api.z.ai chain
(getCustomerInfo → personal project → api_keys → access_tokens, clientType zcode);
`requestAuth {headersApplied:true, apiKey, apiKeyId}` returned — and subsequent
turns run), **both advertised authMethods functional** (AUTH-03), **fail-fast
without a keyring** (AUTH-04: clear actionable coded error, never a hang), and the
**registry-driven model/thought picker** (MODEL-01: `session/new` returns
`configOptions` built from the live provider registry after auth; MODEL-02:
`session/set_config_option` maps to `session/setModel` / `session/setThoughtLevel`,
re-emits the full list, keeps `currentValue` honest). Phase bar = Phase 5's live
standard: a full real run on a coding-plan account (success criterion 1).

Out of scope: SESS-02 delete/fork compat no-ops (Phase 7), SESS-03 Orca persona
contract (`--model` at launch, `/model` slash prompt Hb strings) (Phase 7),
VM/manual E2E + English docs + preset-agent launch recipe (Phase 7),
godbus Secret Service transport (deferred — seam-swappable per D-01),
elicitation/create preference (Phase 7).

</domain>

<decisions>
## Implementation Decisions

### Keyring Transport & Credentials
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

### PAT Lifecycle
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

### Model Picker (configOptions)
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

### Real-Account Evidence
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

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Planning & Research (in repo)
- `.planning/REQUIREMENTS.md` — AUTH-02, AUTH-03, AUTH-04, MODEL-01, MODEL-02
- `.planning/ROADMAP.md` — Phase 6 success criteria (5 items)
- `.planning/phases/05-interactions-full-chat-cycle-api-key/05-VERIFICATION.md`
  — the responder/auth machinery this phase extends (auth.go coding-plan arm)
- `.planning/phases/05-interactions-full-chat-cycle-api-key/05-CONTEXT.md` —
  D-05 real-account discipline, D-06 credential-decrypt seam origin
- `.planning/research/REFERENCE-ADAPTERS.md` — golden file map:
  `src/backend/credential-store.ts`, `src/config/account-provider.ts`,
  `src/config/project-token.ts`, `src/handlers/server-requests.ts`,
  `src/backend/client.ts` + their tests (the hardest chain, tested there)
- `.planning/research/PITFALLS.md` — pitfall 1 (ack ≠ acceptance), pitfall 8
  (cancel races / credential queue deadlines)
- `.planning/research/ARCHITECTURE.md` — Pattern 3 responder policy object

### Protocol & Research Docs (in repo)
- `docs/protocol/app-server-dialect.md` — provider/* + interaction/* surface,
  snapshot revision usage, error codes, §5/§8 turn-latch and read-back facts
- `docs/protocol/install-discovery.md` — the exact-revision builtin formula
  `zcode-builtin:<rev>:<sha256(path.resolve(activeFilePath))>`
- `docs/protocol/workspace-key.md` — workspaceKey semantics
- `docs/research/reference-adapters.md` — adapter comparison + porting notes

### ACP Spec (external, authoritative for the front)
- agentclientprotocol.com v1 — session-setup / config-options pages:
  `configOptions` shape, `session/set_config_option`, re-emit + `currentValue`
  contract; initialization (authMethods advertisement)

### External Ground Truth (outside the repo)
- ZCode 3.17.x local clone (provider registry service, credential flows) —
  cite-and-point only
- FINDINGS.md + field-notes oauth-coding-plan-shim.md (private, outside repo) —
  field-verified OAuth chain shapes
- github.com/william0wang/zcode-acp — golden reference implementation
  (TypeScript, Apache-2.0; Go porting allowed with attribution)
- Real coding-plan account: desktop login on the local machine (never committed)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/interact` — responder policy object, `cipherdecrypt.go` KeyLoader
  seam (secret-tool + AES-256-GCM, field-proven), dedupe: the coding-plan auth
  plugs into the responder; KeyLoader gains the v1 fallback
- `internal/acpfront/auth.go` — the `authenticate{zai-individual-coding-plan}`
  arm currently honest-declines ("lands in Phase 6"); the api-key chain
  (plan 05-07: gate → watcher → snapshot → push → refresh → read-back verdict)
  is the structural template to extend
- `internal/hostsvc` — SnapshotBuilder (exact-revision snapshot),
  WatchRegistry/Await (the PITFALL-1 verdict machinery), PushEnvelope,
  RefreshRegistry, probeCreateReadback surface — all reused
- Fake app-server + golden corpus — new scenarios: cipher ask during auth,
  PAT-mint-driven runtime-headers, configOptions/set_config_option round-trips

### Established Patterns
- TDD RED→GREEN with machine evidence; exactly-once semantics test-locked
- Honest errors: never a fabricated success (PITFALL 3), never raw transport noise
- Real-tier discipline: guarded manual tier, ledger + scrubbed captures
  (Phase-5 AR precedent, EXP-APIKEY-CYCLE-03)
- New external calls (api.z.ai HTTPS) on stdlib net/http; any new library passes
  the Legitimacy Audit (acp-go-sdk precedent, 04-CONTEXT D-01)

### Integration Points
- acpfront gains: the coding-plan authenticate chain, configOptions in the
  `session/new` reply, the `session/set_config_option` handler
- interact gains: the PAT-cache-backed `requestProviderRuntimeHeaders`
  answerer (selected by auth mode, alongside the api-key answerer)
- cmd wiring: NO new user config for coding-plan (desktop login is the
  credential); api-key config unchanged

</code_context>

<specifics>
## Specific Ideas

- The live bar is the Phase-5 standard: field-proven or the phase is not done
  (user chose the full live test explicitly, against the cheaper staged option).
- PAT-on-disk was explicitly rejected — in-process only, hygiene lock like key
  material, test-locked.
- Registry read-back helper (WatchRegistry/Await) reused per Phase-5 CONTEXT
  specifics — the verdict is always registry state, never the ack.
- The Orca `/model` slash-prompt Hb-string contract lands in Phase 7 (SESS-03);
  the picker must not preempt or half-implement it.

</specifics>

<deferred>
## Deferred Ideas

- godbus/dbus Secret Service transport — Phase 7 optional local seam swap (D-01)
- SESS-02 (session/delete + fork graceful no-ops) — Phase 7
- SESS-03 (Orca persona contract: `--model` at launch, `/model` slash prompt) — Phase 7
- elicitation/create preference — Phase 7 integration
- session/load/fork real support — v2
- Persistent cross-restart PAT cache — rejected for v1 (disk hygiene); revisit
  only on field demand

</deferred>

---

*Phase: 6-OAuth Coding-Plan Path & Model Picker*
*Context gathered: 2026-11-22*
