# Phase 6: OAuth Coding-Plan Path & Model Picker - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-11-22
**Phase:** 6-OAuth Coding-Plan Path & Model Picker
**Areas discussed:** Keyring transport, PAT lifecycle, configOptions/picker, Real-run depth

---

## Keyring transport

| Option | Description | Selected |
|--------|-------------|----------|
| Keep secret-tool | Field-proven in Phase 5 real run; zero new deps; requires libsecret-tools documented as prerequisite | |
| godbus now | Pure Go, no external binary; new dep via Legitimacy Audit; transport loses field-proof | |
| secret-tool + swappable seam | secret-tool as v1 transport behind KeyLoader seam; Phase-7 godbus swap must be a local edit | ✓ |

**User's choice:** secret-tool + swappable seam
**Notes:** —

### AUTH-04 placement

| Option | Description | Selected |
|--------|-------------|----------|
| Probe at authenticate start | Check secret-tool + keyring entry BEFORE snapshot push; coded actionable error | ✓ |
| Lazy check | Chain starts, fails at first cipher ask | |

**User's choice:** Probe at authenticate start

### Legacy enc:v1 fallback

| Option | Description | Selected |
|--------|-------------|----------|
| v2 only | Smaller surface; v1 fallback deferred to Phase 7 if field cases appear | |
| Port v1 fallback like the reference | ZCODE_CREDENTIAL_SECRET / machine-string derivation alongside enc:v2:os | ✓ |

**User's choice:** Port v1 fallback (completeness chosen over the recommended v2-only)

---

## PAT lifecycle

| Option | Description | Selected |
|--------|-------------|----------|
| In-process, mint at authenticate | Mint once, in-memory cache, never on disk | ✓ |
| Persistent cache | Cache across restarts (keyring/state file) | |
| Mint per runtime-headers ask | Freshest PAT, several HTTPS calls mid-turn | |

**User's choice:** In-process, mint at authenticate

### Expiry handling

| Option | Description | Selected |
|--------|-------------|----------|
| Reactive: one re-mint + retry | No TTL guessing; second failure = honest coded error | ✓ |
| Proactive TTL timer | Track TTL from access_tokens reply | |

**User's choice:** Reactive

### Multi-project pick

| Option | Description | Selected |
|--------|-------------|----------|
| First personal project + warn | Deterministic, user never blocked | ✓ |
| Coded error on ambiguity | Error listing projects + config flag/env | |

**User's choice:** First personal project

### No API keys on account

| Option | Description | Selected |
|--------|-------------|----------|
| Create if absent | Official API, idempotent (reference/desktop behavior) | ✓ |
| Coded error only | User creates key manually in console | |

**User's choice:** Create if absent

---

## configOptions/picker

| Option | Description | Selected |
|--------|-------------|----------|
| Mirror the server's list verbatim | No wrapper-side filtering/enrichment; client can never pick a model the server refuses | ✓ (re-asked in plain language after user feedback "а если без жаргона и ребусов?") |
| Wrapper-curated list | Wrapper hides/renames/enriches | |

**User's choice:** Mirror verbatim
**Notes:** First phrasing was rejected as jargon-heavy; the re-ask described the mechanism in everyday words ("сервер сам объявляет список доступных моделей — показываем ровно его").

### Thought levels

| Option | Description | Selected |
|--------|-------------|----------|
| From model metadata | As reported by the backend per model; researcher pins location | ✓ |
| Static curated list | Hardcoded levels for all models | |

**User's choice:** From model metadata

### Initial currentValue

| Option | Description | Selected |
|--------|-------------|----------|
| Confirmed state only | Server-confirmed current + setModel outcome; never unconfirmed targets | ✓ |
| Target value with rollback | Show what we sent, roll back on refusal | |

**User's choice:** Confirmed state only

### Picker scope

| Option | Description | Selected |
|--------|-------------|----------|
| Both auth paths | api-key + coding-plan; prepares SESS-03 | ✓ |
| Coding-plan only | api-key picker deferred to Phase 7 | |

**User's choice:** Both auth paths

---

## Real-run depth

| Option | Description | Selected |
|--------|-------------|----------|
| Full live test | Phase-5 bar: keyring login + real PAT mint + one real request on coding-plan account | ✓ |
| Partial | Login chain live; turn stays on test double | |

**User's choice:** Full live test

### Spend cap

| Option | Description | Selected |
|--------|-------------|----------|
| Same cap as Phase 5 | One tool-less tiny request; per-turn ledger entries | ✓ |
| Pre-approved extended cap | Several attempts up front | |

**User's choice:** Same cap as Phase 5

---

## Claude's Discretion

- No user-supplied wrapper config for coding-plan auth (desktop login is the credential); missing login = coded actionable error
- Unknown/stale config option → coded invalid-params, never a crash
- Encrypt-op cipher asks stay declined; researcher verifies coding-plan auth never requires host-side encryption
- Mint-chain per-call timeouts, error wording, wave/task decomposition, setThoughtLevel-vs-setModel field resolution (researcher/planner)

## Deferred Ideas

- godbus/dbus Secret Service transport (Phase 7 optional seam swap)
- SESS-02, SESS-03 Orca persona contract, elicitation preference (Phase 7)
- session/load/fork real support (v2)
- Persistent cross-restart PAT cache (rejected for v1; revisit on field demand)
