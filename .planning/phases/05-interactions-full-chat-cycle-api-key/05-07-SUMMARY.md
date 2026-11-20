---
phase: 05-interactions-full-chat-cycle-api-key
plan: 07
subsystem: auth-composition-tier
tags: [auth-01, authenticate, credential-decrypt, aes-gcm, keyring, d-06, responder-wiring, binary-e2e, hygiene, tdd]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key (05-06)
    provides: hostsvc PushSnapshot/RefreshRegistry/WatchRegistry read-back machinery, RuntimeHeadersHandler + StaticKey, the provider/* method constants
  - phase: 05-interactions-full-chat-cycle-api-key (05-01)
    provides: interact Responder policy object (the injectable seam posture — one wiring change per D-06), WithPermissionForwarder/WithRuntimeHeadersKeySource
  - phase: 04-acp-front-minimal-chat-loop
    provides: acpport.Handlers seam + agentAdapter, the Front composition harness (ComposeFront), the Phase-4 honest-error authenticate lock (superseded)
provides:
  - interact.CredentialDecrypt — the D-06 seam (field-verified enc:v2:os layout parser, kid verification, stdlib AES-256-GCM decrypt, coded {ok:false} decline arms, encrypt decline)
  - interact.KeyLoader seam + interact.SecretToolKeyLoader (field-verified secret-tool lookup, realpath username derivation, ScrubbedEnv allowlist, captured stdout) — Phase 6's godbus vault implements the same interface
  - acpfront.Front.Authenticate — the honest chain (key gate → read-back watcher registered pre-traffic → snapshot push → registry refresh → read-back verdict → {}), per-step coded errors, the OQ4 no-key error naming ZCODE_ACP_API_KEY, clean-retry interruptibility
  - acpport.Handlers.Authenticate + adapter delegation (the Phase-4 not-available stub replaced; the honest-error lock rewritten to the delegation contract)
  - cmd wiring: ZCODE_ACP_API_KEY env + --api-key flag (flag wins), responder.OnRequest assigned pre-traffic over the full policy table (permission bridge + runtime-headers key source + cipher decrypt seam), binary E2E responder leg (differential requestUserInput discriminator)
affects: [05-08, 05-09 real tier (composes this stack live), phase-06 OAuth (godbus vault behind the KeyLoader interface)]

# Actuals (#2632)
actuals:
  tokens: 27222    # chars/4 over the realized diff (108887 chars); plan estimated 48000
  tasks: 3
  commits: 7       # MEASURED: git rev-list --count gsd-plan-head-before-05-07..HEAD
commits: 7
plan_head_before: fcf4efdfabdc2f928e59f02d266caecb3a75a22d
plan_head_after: 67efd3616d01ce4b802e9ea4154b6701e812c786

# Tech tracking
tech-stack:
  added: []        # zero new dependencies: crypto/aes + os/exec are stdlib; secret-tool is a system binary on the real tier, never a Go module (T-5-SC)
  patterns:
    - coded-decline union inside the reply RESULT (ok:false never rides the JSON-RPC error lane — the server result-schema-parses the union)
    - hygiene-by-construction: shapes/lengths in every log field, argv carries a path derivation never key bytes, caller-scrubbed subprocess env, generic decline messages (crypto/loader error text never echoed)
    - synthetic-vector discipline: runtime-generated keys + runtime-sealed blobs keep scan-secrets green with no literal credential shapes
    - differential binary-E2E discriminator: interaction/requestUserInput answered {action:decline} by the wired responder vs -32601 by the built-in router — the wiring proof needs no transcript ambiguity

key-files:
  created:
    - internal/interact/cipherdecrypt.go
    - internal/interact/cipherdecrypt_test.go
    - internal/acpfront/auth.go
    - internal/acpfront/auth_test.go
    - cmd/zcode-acp/testdata/responder-e2e.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-07-red-cipher-t1.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-07-red-auth-t2.json
  modified:
    - internal/interact/responder.go
    - internal/acpfront/front.go
    - internal/acpport/types.go
    - internal/acpport/agent.go
    - internal/acpport/agent_test.go
    - cmd/zcode-acp/main.go
    - cmd/zcode-acp/main_test.go
    - experiments/scan-secrets.sh

key-decisions:
  - "The wired cipher policy row replaces the D-05 fall-through ONLY when the seam is injected (WithCredentialDecrypt adds the row; unwired Responders keep the Phase-3 locked tracked-no-answer byte posture) — CI tiers never touch the keyring because no test path constructs the production loader AND executes it (the only test LoadKey call is the empty-path fail-loud, which returns before exec)"
  - "AuthConfig.ReadbackDeadline is an injectable knob (0 → hostsvc default 30 s): without it the read-back-negative arm is untestable — the client context dies at the same instant the server's Await gives up, and the SDK answers the client-side cancelled mapping instead of the server's coded verdict"
  - "The no-key error wraps through the seam vocabulary (acpport.NewInternal over the sentinel) so the wire MESSAGE carries ZCODE_ACP_API_KEY verbatim — the plain-error path would have shipped the text in data under the SDK's generic 'Internal error' message"
  - "The binary E2E responder leg discriminates with interaction/requestUserInput rather than runtime-preferences: the static runtime-preferences answer is byte-identical on both router paths (no proof), while requestUserInput separates the wired decline from the built-in -32601 exactly"
  - "The production loader's username derivation hashes the RESOLVED path STRING (EvalSymlinks+Abs, the supervisor.BuiltinRevision discipline) — the machine-local FINDINGS evidence is ambiguous about a trailing newline (a shell herestring artifact); flagged as an assumption the 05-09 real-tier run discharges (a wrong derivation degrades honestly per D-06, never a fabricated answer)"

requirements-completed: [AUTH-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "CredentialDecrypt seam decrypts the field-verified enc:v2:os layout with stdlib AES-256-GCM, verifies the kid, declines encrypt, and answers every failure as a coded {ok:false} arm (tamper, loader failure, kid mismatch, malformed shapes)"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestCredentialDecryptDecrypt|TestCredentialDecryptTamper|TestCredentialDecryptEncryptDeclines|TestCredentialDecryptLoaderFailure|TestCredentialDecryptKeyMismatch|TestCredentialDecryptMalformedValue (go test -race ./internal/interact/)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Key-material hygiene is test-locked: no key/plaintext on the loader argv (the sha256 realpath derivation only), the caller-scrubbed env passes through verbatim, no log line carries key bytes or plaintext, and no real credential material exists anywhere in the package"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestSecretToolLoaderHygiene|TestSecretToolLoaderEmptyPath|TestCredentialDecryptLogHygiene + bash experiments/scan-secrets.sh (exit 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Front.Authenticate{zai-api-key} runs the honest chain with per-step coded errors: read-back-proven success, the OQ4 no-key error naming ZCODE_ACP_API_KEY, unknown-methodId invalid-params, ack-without-registry negative, and clean-retry interruptibility — all over the composed SDK↔acpport↔Front↔fake harness"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/acpfront/auth_test.go#TestAuthenticate (5 arms, transcript-sequence asserts) + internal/acpport/agent_test.go#TestAuthenticateDelegation (go test -race ./internal/acpfront/ ./internal/acpport/)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The built binary wires the whole phase stack (key config, responder OnRequest before any frame flows, cipher seam injected) and the binary E2E proves the wired responder answering a scripted interaction reverse through production stdio"
    requirement: AUTH-01
    verification:
      - kind: integration
        ref: "cmd/zcode-acp/main_test.go#TestAgentBinaryEndToEnd/wired_responder_answers_a_scripted_reverse (+ the two-loop ride unchanged) — go test -race -run TestAgentBinaryEndToEnd ./cmd/zcode-acp/"
        status: pass
    human_judgment: false
  - id: D5
    description: "The secret-tool loader decrypts a REAL mid-turn cipher ask against the real login keyring on a live z.ai turn (the D-06 real-tier proof this plan's machinery exists for)"
    verification: []
    human_judgment: true
    rationale: "Coverage not determined at authoring time — the real-tier run is plan 05-09's guarded deliverable (build tag + env double lock; quota ledgered). CI tiers never touch the keyring by design, so this plan proves the machinery against synthetic vectors only; a wrong username derivation or locked keyring degrades honestly per D-06 and will surface in 05-09's run, never as a fabricated answer."

# Metrics
duration: 48min
completed: 2026-11-20
status: complete

---

# Phase 5 Plan 07: Authenticate Composition + CredentialDecrypt Seam Summary

**Front.Authenticate{zai-api-key} (read-back-proven, coded per-step failures, OQ4 no-key error), the D-06 CredentialDecrypt seam (stdlib AES-256-GCM over the field-verified enc:v2:os layout behind an injectable secret-tool loader, test-locked hygiene), and the binary wiring that stitches the whole phase stack — proven by a differential binary-E2E responder leg.**

## Performance
- **Duration:** 48min
- **Started / Completed:** 2026-11-20T14:25:21Z / 2026-11-20T15:13:56Z
- **Tasks:** 3 · **Files modified:** 15

## Accomplishments
- The CredentialDecrypt seam decrypts the field-verified `enc:v2:os:<kid>:<iv>.<tag>.<ct>` layout with stdlib AES-256-GCM (kid verified against first-24-hex of sha256(key)), answers the `{requestId, result:{ok…}}` discriminated union, declines encrypt, and degrades honestly on loader failure — never a fabricated success, never a panic, never a stall.
- The hygiene contract is test-locked, not prose: no key material on argv (the username is the sha256 realpath derivation), caller-scrubbed subprocess env (`ScrubbedEnv` allowlist), no log line carries key bytes or plaintext (shapes/lengths only), captured stdout drained, generic decline messages — synthetic runtime-sealed vectors only.
- `Front.Authenticate` runs the honest PITFALL-3 chain (record key → read-back watcher registered BEFORE the push traffic → snapshot push → registry refresh → read-back verdict → `{}`) with per-step coded errors; the no-key arm names `ZCODE_ACP_API_KEY` exactly (OQ4); interruption is a clean retry with no half-authenticated state observable.
- The binary carries the whole phase stack: `ZCODE_ACP_API_KEY`/`--api-key` (flag wins), the responder assigned to `conn.OnRequest` before the ACP connection exists, the permission bridge + runtime-headers key source + cipher seam injected — and the binary E2E proves the wired responder answering a scripted `interaction/requestUserInput` reverse through production stdio.
- Zero new dependencies: crypto/aes and os/exec are stdlib; secret-tool is a system binary consumed only on the real tier (T-5-SC).

## Task Commits

1. **Task 1: CredentialDecrypt seam (RED→GREEN)** — `b1860f4` (test) → `7bb2946` (feat); deviation fix `78ab036` (fix)
2. **Task 2: Front.Authenticate (RED→GREEN)** — `f4481d6` (test) → `a8f1b93` (feat); lint refactor `67efd36` (style)
3. **Task 3: cmd wiring + binary E2E extension** — `aa6e008` (feat)

_Note: TDD tasks carry RED evidence validated by `gsd_run check tdd-red-evidence` (both `RED_EVIDENCE_OK`) before GREEN._

## Files Created/Modified
- `internal/interact/cipherdecrypt.go` — the D-06 seam: layout parser, GCM open, coded decline arms, SecretToolKeyLoader, ScrubbedEnv
- `internal/interact/cipherdecrypt_test.go` — synthetic-vector layout/tamper/decline/hygiene locks + responder seam integration
- `internal/interact/responder.go` — WithCredentialDecrypt injection; the D-05 fall-through now conditional on the seam being unwired
- `internal/acpfront/auth.go` — Front.Authenticate + AuthConfig (Keys/Install/WantModels/ReadbackDeadline) + the honest chain
- `internal/acpfront/auth_test.go` — TestAuthenticate's five arms over the composed harness (transcript-sequence asserts)
- `internal/acpfront/front.go` — auth field + advertisedAuthMethods as the Initialize/validation single source
- `internal/acpport/types.go` — Handlers.Authenticate widened (neutral methodID vocabulary)
- `internal/acpport/agent.go` — the Authenticate delegate (wireError coding); Phase-4 stub replaced
- `internal/acpport/agent_test.go` — the honest-error lock rewritten to TestAuthenticateDelegation (plain/seam/success arms)
- `cmd/zcode-acp/main.go` — env const + flag + parseConfig key + buildFront/wireResponder (pre-traffic OnRequest)
- `cmd/zcode-acp/main_test.go` — the responder E2E leg (differential requestUserInput discriminator)
- `cmd/zcode-acp/testdata/responder-e2e.json` — the leg's scripted scenario (reverse with expect_answer mid-create)
- `experiments/scan-secrets.sh` — allowlist extension (see Deviations)

## Decisions Made
See key-decisions in frontmatter: seam injection keeps the unwired posture; the injectable ReadbackDeadline; the no-key error through the seam vocabulary; the requestUserInput E2E discriminator; the realpath (no-newline) username derivation flagged for the 05-09 real-tier discharge.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] scan-secrets gate latent failure on 05-06's synthetic fixture**
- **Found during:** Task 1 verify (`bash experiments/scan-secrets.sh`)
- **Issue:** The runtime-headers-answer corpus fixture (plan 05-06) carries the self-declaring synthetic value `zc-synthetic-fixture-not-a-real-key`, which the token-value scan pattern flags; the phase branch has not yet passed the PR secret-scan job, so the phase-level verify item "scan-secrets green" was blocked by a latent pre-existing failure (verified present at the pre-plan commit fcf4efd).
- **Fix:** Extended the script's documented exact-string allowlist with the self-declaring marker (the golden byte-locks the value; allowlisting the marker avoids corpus churn). My own new fixture was additionally shaped to not match (defense in depth).
- **Files modified:** experiments/scan-secrets.sh
- **Verification:** scan exits 0 with both fixture classes green; the patterns themselves untouched.
- **Committed in:** 78ab036

**2. [Rule 3 - Blocker] Handlers widening required the scripted test fake to gain Authenticate**
- **Found during:** Task 2 RED (interface widening broke the scriptedHandlers fake's compile)
- **Issue:** Widening `acpport.Handlers` with Authenticate leaves any test fake without the method uncompilable.
- **Fix:** scriptedHandlers gained the Authenticate double (canned authErr + call recording); totalCalls extended.
- **Files modified:** internal/acpport/agent_test.go
- **Verification:** the acpport suite compiles and stays green.
- **Committed in:** f4481d6

**3. [Rule 2 - Missing critical] AuthConfig.ReadbackDeadline injectable knob**
- **Found during:** Task 2 GREEN (the read-back-negative arm surfaced the SDK's client-side cancelled mapping, never the server's coded verdict)
- **Issue:** With the read-back window hard-wired to the 30 s default, the client context dies at the same instant the server's Await gives up — the negative arm is untestable and production cannot bound the window per deployment.
- **Fix:** `AuthConfig.ReadbackDeadline` (0 → hostsvc default) passed through to `hostsvc.WithReadbackDeadline` — the established injectable-deadline posture (WithMethodDeadline/WithReadbackDeadline precedents).
- **Files modified:** internal/acpfront/auth.go
- **Verification:** the negative arm receives the coded read-back error naming the step; all five arms green under -race.
- **Committed in:** a8f1b93

**Total deviations:** 3 auto-fixed (2 Rule 3 blockers, 1 Rule 2 missing-critical).
**Impact on plan:** none on the contracts — all three are additive or test-infrastructure fixes; the plan's own fails_when anticipated deviation 2's lock migration ("the adapter's authenticate delegation … no longer matches the new contract").

### Planned Contract Migration (not a deviation)

`TestAuthenticateHonestError` (plan 04-03) was rewritten to `TestAuthenticateDelegation` in the Task 2 RED commit: the Phase-4 not-available stub was replaced by the delegation exactly as the plan's Task 2 action prescribes, and the rewritten lock pins the new contract (plain errors code internal with detail in data, seam codes ride verbatim, success answers the empty `{}`) while keeping plan 04-03's honesty floor (never fake success, never -32601).

## Issues Encountered
- The SDK's `NewInternalError` maps a plain handler error to message "Internal error" with the detail in `data` — the no-key and read-back messages must travel through the seam vocabulary (`acpport.NewInternal`) for the client's readable surface to carry them (resolved in GREEN; the delegation lock pins both mappings).
- `mise run test-experiments` showed one flaky node --test "promise resolution" failure under concurrent load; it passes 22/22 standalone and on repeated `mise run ci` runs (CI_EXIT=0) — no action taken.

## User Setup Required
None - no external service configuration required for CI. (Real-tier users set ZCODE_ACP_API_KEY or pass --api-key; the keyring path requires libsecret's secret-tool, present on target GNOME machines — assumption A2, degraded honestly when absent.)

## Next Phase Readiness
Ready for 05-08. The composition surface 05-09's real tier consumes is complete: `Front.Authenticate` + the responder policy table + the cipher seam are wired in the production binary and proven at the binary tier; the only unproven arm is the live keyring decryption (D5, 05-09's guarded run discharges it — including the username-derivation assumption). 05-08's remaining work composes on top of an unchanged, fully green suite (`mise run ci` exit 0; 10/10 packages under -race).

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-20*
