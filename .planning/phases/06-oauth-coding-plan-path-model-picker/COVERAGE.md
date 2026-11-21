# Phase 6 — API Coverage Matrix & Probe Ledger

**Phase:** 06-oauth-coding-plan-path-model-picker
**Declared:** 2026-11-22
**Kind:** coverage matrix (api.z.ai — the integration Phase 5's declaration deferred to this phase) + specless probe ledger (finalized at plan time; real-run dispositions updated by 06-08)

## API Coverage — api.z.ai (the official PAT-mint host)

The deterministic detector (`api-coverage.cjs`) returned `detected:false` over
the roadmap section (no verb+noun co-occurrence), but the phase genuinely
integrates an external HTTPS service — RESEARCH Pattern 3's three pinned calls
— and Phase 5's COVERAGE.md explicitly deferred this surface's coverage
treatment to Phase 6. The matrix is therefore produced at plan time, honestly.

> Full coverage by default. Opt-outs are explicit, reasoned decisions.
> Scope note: the surface enumerated is the project-token chain the official
> desktop/reference define for this integration (FINDINGS §3.3, ref
> project-token.ts) — the wrapper never calls model/chat endpoints directly
> (that is the backend's job; REQUIREMENTS Out of Scope).

| capability | decision | reason |
|---|---|---|
| getCustomerInfo (orgs/projects listing) | INTEGRATE | the personal-pick input (D-06); Pattern 3 step 2 (raw-token form) |
| api_keys list | INTEGRATE | the candidate search for name == zcode-api-key; Pattern 3 step 4 |
| api_keys create (create-if-absent) | INTEGRATE | D-07 locked user decision (desktop behavior supersedes the reference's read-only posture); Pattern 3 step 5 |
| access_tokens mint | INTEGRATE | the PAT itself; Pattern 3 step 6 incl. the 404 location-invalidation rule |
| accountScope digest | INTEGRATE | rides every mint (byte-parity with desktop+reference); Pattern 3's scope block |
| quota/usage endpoints | OPT-OUT | not retrievable per the honesty rules — REQUIREMENTS Out of Scope ("Invented quota/usage numbers"); nothing reported rather than wrong numbers |
| Start-Plan / browser-captcha auth flows | OPT-OUT | explicitly out of scope (Start Plan does not work headless — documented limitation, DOCS-03); the coding-plan credential is the desktop login per D-12 |
| bundle patching / attestation forgery for quota | OPT-OUT | ToS violation + fragile + against project honesty rules (REQUIREMENTS Out of Scope); official PAT path only |
| model/chat completion endpoints | OPT-OUT | never called by the wrapper by design (two-implementations-drift rule, REQUIREMENTS Out of Scope) |

Non-remote surfaces this phase touches (declared like Phase 5, not matrix rows):
the `secret-tool` subprocess (local keyring CLI, D-01 seam, guarded real tier
only) and the desktop's read-only state files (credentials.json, setting.json,
the builtin table). The ZCode app-server wire ops ride the existing Phase-3
codec (fake-corpus locked per plan).

## Assumption-Delta Advisory

`assumption-delta scan "6"` returned `detected:false` (no pluralization/
optional/chosen signals in the phase section). Recorded for the audit trail;
no action. (The auth-mode field is a new discriminator but not a
singular→plural migration of an existing identity model — the api-key and
coding-plan paths were designed as two arms from Phase 4's initialize onward.)

## Specless Probe Fallback — Edge Ledger (finalized)

7 applicable probe items from the coverage report: **3 resolved explicit**
(AUTH-02's categorized edges — authored as plan must_haves truths with named
tests), **4 unclassified → carried as flagged assumptions** (never
auto-resolved, never silently dropped). Equality: 7 = 3 authored + 4 flagged.

| # | Requirement | Probe (category) | Resolution | Plan/Task |
|---|-------------|------------------|------------|-----------|
| 1 | AUTH-02 | empty — empty/single/null input | Missing/empty store or missing oauth token → ok=false + the coded desktop-first error at the consumers; single-candidate pick without warnings; nil JWT folds as null into the scope array | 06-01 T1 (TestCredStoreMissingAndEmpty posture in TestCredStore), 06-02 T1/T2 (TestPatMinterLoginRequired, TestPatMinterHappyPath), 06-07 T2 (PAT-less coded arm) |
| 2 | AUTH-02 | encoding — bytes/codepoints/normalized form | v1 = exactly 3 base64url (RawURLEncoding) dot segments; fp24 = lowercase-hex sha256[:24]; scope digest = sha256 over the compact JSON array (JSON.stringify parity), lowercase hex; rejection fields validated 64-hex; rule strings carried verbatim (no normalization) | 06-01 T1/T2 (TestV1Fallback, TestCredStore), 06-02 T1 (TestAccountScopeDigest), 06-03 T1 (TestAccountSnapshotVerbatimMirror), 06-07 T1 (hex validation) |
| 3 | AUTH-02 | concurrency — interrupted/parallel guarantee | Single-flight Ensure (one mint per burst); reactive re-mint exactly once per ask, second failure = coded error; read-only stateless store reader; interrupted authenticate records no success (Phase-5 lock carried) | 06-02 T3 (TestPatMinterSingleFlight, backoff), 06-07 T2 (TestPatHeadersSourceReMint arms), 06-05 T3 arm (e) |
| 4 | AUTH-03 | unclassified | FLAGGED → A4 below (thoughtLevel live presence) | 06-05 (probe ledger), 06-08 T3 (designed confirmer) |
| 5 | AUTH-04 | unclassified | FLAGGED → A3 below (locked-keyring prompting) | 06-05 T1 (bounded-ctx containment), 06-08 (real-run observation) |
| 6 | MODEL-01 | unclassified | FLAGGED → A7 + A8 below (currentValue rendering; Orca-side rendering) | 06-04 T1/T3 (spec-safe omission, locked OQ2), 06-08 T3 |
| 7 | MODEL-02 | unclassified | FLAGGED → A5 + A9 below (setThoughtLevel liveness — retired by the OQ1 lock; seam-widening test locks) | 06-06 (OQ1 lock), 06-07 T1 (red-first lock updates) |

## Prohibition Recall/Precision (descriptor-less, flagged-unverified)

Stage-1 adversarial recall per requirement (~10 candidates) then Stage-2
precision. **Kept — 3 bespoke prohibitions** (values/honesty, project-specific,
not routine engineering), authored into the owning plans' `must_haves.prohibitions`:

1. Auth success only on the registry read-back, never the provider/updateAccountConfig ack (PITFALL 1; the schema's own 收到 comment) — authored in 06-05
2. The PAT (and the login token) is never persisted to disk or any store — in-process only for the wrapper's lifetime (D-04 hygiene, T-5-17 extension) — authored in 06-02
3. The failure-shaped runtime-headers success (headersApplied:false, or a PAT-less requestAuth) is never emitted — failures are coded error envelopes only (PITFALL 3) — authored in 06-07

**Dropped with breadcrumbs** (canon → existing tooling/threat register; routine →
edge probes/code review) — 7 candidates, 0 silent drops (10 surfaced total):

- Credential material in logs/captures/argv — canon ASVS V7 secrets-logging → threat models T-06-02/T-06-15/T-06-21/T-06-24 + scan-secrets gate (the generic form; the three KEPT prohibitions are the project-specific values)
- Read-only parity toward desktop stores (never write credentials.json/setting.json) — canon V4 → design-wide invariant; the plans contain no write paths (06-02's TestPatMemoryOnly is the nearest lock); recorded here rather than minted as a phase prohibition
- Official-paths-only / no non-official endpoints for the mint — already a REQUIREMENTS-level constraint (AUTH-02 wording "only official paths" + Out-of-Scope rows); requirement, not an additional prohibition
- Never serve a stale token after scope change — canon-adjacent but already minted as threat T-06-06/T-06-22 with mitigation vectors; duplicating it as a prohibition would double-count
- Never fall back to reading plan api-keys from the store as the auth source — superseded design state (desktop 3.15+); RESEARCH Alternatives table documents it; the mint path has no fallback branch to forbid
- Never hard-code the builtin revision string — carried from Phase 5's authored prohibition; the account-overlay plan (06-03) keeps the formula-reuse test (path-sensitivity vector) without re-minting the prohibition text
- Never invent thought levels or model entries wrapper-side — D-08/D-09 are locked decisions enforced by the verbatim-mirror vectors (06-04 T1); decision-coverage, not a new prohibition

Equality: prohibitions 3 kept (authored) + 7 dropped-with-breadcrumbs = 10
surfaced; 0 silent drops.

## Flagged Assumptions (carried, never auto-dismissed)

| # | Assumption | Where it lives | Disposition |
|---|------------|----------------|-------------|
| A1 | The PAT's actual TTL is not publicly documented; expiresIn/expiresAt + 120 s lead + the reactive posture make the number immaterial | 06-02 (Pattern 3) | Reactive semantics test-locked; a hard server cadence would surface as honest coded errors + re-mints (recoverable) |
| A2 | Create-if-absent is race-tolerant (check-then-create may duplicate under concurrent first-auths) | 06-02 | Accepted: single-user wrapper, single-flight mints, list-first self-healing (T-06-07, disposition accept) |
| A3 | A LOCKED keyring's prompting behavior was not probed (no lock state on the research machine) | 06-05 T1 | The 5 s bounded ctx contains any prompt path; the coded error still surfaces; the real run observes |
| A4 | settings.thoughtLevel is present in LIVE 3.18.2 create/setModel replies (schema-certain; live captures quoted model.available explicitly) | 06-05 probe ledger, 06-04 fallback posture | If absent, the per-model reasoning.levels fallback (also backend-reported) keeps the picker honest; 06-08's capture is the designed confirmer |
| A5 | session/setThoughtLevel behaves live per its 3.17.1 schema (never exercised on 3.18.2) | 06-06 | RETIRED by the planner-locked OQ1: the v1 wire path uses setModel + options.reasoningLevel for both switch kinds (field-proven per EXP-TURN-01); the dedicated method is not introduced |
| A6 | clientVersion in the mint body tolerates the desktop version or the "0.0.0" fallback (no documented server validation) | 06-07 (wiring seam) | A rejection surfaces as a visible mint business error, retryable with the other value; the real run confirms |
| A7 | Exact client rendering of a required currentValue when the backend reports no current is unknown; the wrapper omits the affected entry (spec-safe reading, locked OQ2) | 06-04 T1/T3 | Honest-smaller-list posture; the real run's configOptions rendering confirms or adjusts |
| A8 | Orca-side config-options rendering risk (client-side; version floor per the Phase-1 card study) | phase-level | Purely client-side; the Phase 7 live E2E covers it; our contract is spec-honest regardless |
| A9 | The responder seam widening (accountScope) extends RuntimeHeadersKeySource without breaking the 05-06/05-07 locks | 06-07 T1 | Red-first lock updates in the same red step, same assertions; the Phase-5 runtime-headers golden is the byte-compat target |

## Planner Locks (Open Questions resolved)

| OQ | Question | Lock |
|----|----------|------|
| OQ1 | Which wire method carries a level-only change | `session/setModel` with `options.reasoningLevel` for BOTH model switches and level-only changes (the field-proven 3.18.2 path); no dedicated setThoughtLevel method, no new method constant (06-06; vector + corpus locked) |
| OQ2 | Whether the model picker entry exists when no current is confirmed | Rely on the backend-reported current (settings.model.current, falling back to lastUsed); the configured setModel follow-up only when configured; OMIT the model entry when the reply carries neither (the spec-safe honest-smaller-list reading) (06-04) |
| OQ3 | Probe-seam shape for AUTH-04 at the fake tier | A `KeyringProber` func type on the auth config (default = the production prober; tests inject canned verdicts), mirroring the KeyLoader injection discipline (06-05 T1) |

## MVP Framing Note

The ROADMAP goal line is not strict user-story format; per this project's
established MVP-mode practice (Phase 4/5 plans), each PLAN frames the phase
goal as its per-plan user story ("As a … I want … so that …") derived from the
ROADMAP goal — no scope invented, no story manufactured beyond the goal's own
actors (coding-plan users; ACP client users).

## Plan Set Overview (waves)

| Plan | Wave | Slice | Requirements |
|------|------|-------|--------------|
| 06-01 | 1 | credential store + enc:v1 fallback (interact) | AUTH-02 |
| 06-02 | 1 | PAT minter + cache (hostsvc) | AUTH-02 |
| 06-03 | 1 | account overlay snapshot (hostsvc) | AUTH-02 |
| 06-04 | 1 | configOptions translator + advertisement + session/new (picker part 1) | MODEL-01, AUTH-03 |
| 06-05 | 2 | keyring probe + entitlement gate + coding-plan authenticate arm + corpus | AUTH-02, AUTH-03, AUTH-04 |
| 06-06 | 3 | set_config_option handler + corpus (picker part 2) | MODEL-02 |
| 06-07 | 4 | runtime-headers PAT serving + cmd wiring | AUTH-02 |
| 06-08 | 5 | composed fake cycle + guarded real tier + HUMAN-GATED real run + evidence | AUTH-02, AUTH-03 |

Wave conflicts resolved by the zero-overlap rule: golden_test.go ownership
(06-05 → 06-06 → 06-07 sequential), hostsvc split across plan-owned files
(patminter* vs snapshot.go + snapshot_account_test.go).
