---
phase: 06-oauth-coding-plan-path-model-picker
plan: 01
subsystem: auth
tags: [oauth, credential-store, aes-gcm, keyring, enc-v1, enc-v2os, tdd]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key
    provides: the KeyLoader seam + SecretToolKeyLoader + CredentialDecrypt decrypt engine (D-06), the coded decline taxonomy (CipherCode*), the T-5-17 hygiene locks, parseEncV2OS layout parser
provides:
  - interact.NewCredentialStore / (*CredentialStore).Entries — the read-only credentials.json reader with the plaintext/enc:v1/enc:v2:os dispatch ladder, per-entry error isolation, missing-file empty-nil posture, coded undecodable-store error
  - (*CredentialStore).LoginToken(ctx, family) — the oauth:<family>:access_token accessor with exactly-one "Bearer " strip (the mint chain's step-1 credential)
  - (*CredentialStore).ZcodeJWT(ctx) — the zcodejwttoken accessor (accountScope input)
  - interact.V1FallbackKeyLoader / NewV1FallbackKeyLoader / ResolveV1CredentialSecret — the enc:v1 fallback sibling behind the SAME KeyLoader seam (D-03; pure compute, no exec)
  - the hasUnhandledEncPrefix never-plaintext rule, test-locked (T-06-01 mitigation)
affects: [06-02 patminter (LoginToken is mint step 1), 06-05 authenticate arm, 06-03 snapshot entitlement (ZcodeJWT), phase-7 godbus seam swap]

# Actuals (#2632) — pairs with the plan's estimate (38000 tokens) to calibrate.
actuals:
  tokens: 12046   # chars/4 over the realized diff (48184 diff chars, 6 files)
  tasks: 2
  commits: 4      # MEASURED: git rev-list --count edad068..HEAD

# Tech tracking
tech-stack:
  added: []       # stdlib only — zero new dependencies (T-06-SC mitigation held)
  patterns: [compile-first skeleton RED (04-06 precedent), prefix-dispatch ladder with per-entry isolation, sibling loaders behind one KeyLoader seam]

key-files:
  created:
    - internal/interact/credstore.go
    - internal/interact/credstore_test.go
  modified:
    - internal/interact/cipherdecrypt.go
    - internal/interact/cipherdecrypt_test.go

key-decisions:
  - "The v2:os arm composes the existing CredentialDecrypt service (decryptOne) instead of re-implementing parse/fp24/GCM — the plan's no-parser-fork rule taken to zero duplication; the store never bypasses the seam"
  - "The enc:v1 blob parse+open (parseEncV1/openEncV1) lives in credstore.go beside the dispatch that owns it; only the KEY loader (secret resolution + sha256) is the cipherdecrypt.go seam sibling — keeps each task's file ownership intact"
  - "The store's v1 loader is the exported NewV1FallbackKeyLoader with OS-default seams; t.Setenv(ZCODE_CREDENTIAL_SECRET) is the test determinism seam (no OS-identity injection needed at the store tier)"
  - "Unknown enc: prefixes skip with a structural log code (unhandled-enc-prefix) — the upstream v2-ENV degradation mode is locked impossible (T-06-01)"
  - "Test split into sibling functions (TestCredStore/Accessors/LogHygiene; TestV1Fallback{RoundTrip,MalformedShapes,WrongSecret,SecretResolution,LoudValidation,Digest}) to satisfy the strict golangci-lint v2 gate (gocyclo 20 / gocognit 30) — same vector families, -run regexes unchanged"

patterns-established:
  - "Compile-first skeleton RED: new-API tests commit with minimal compiling signatures (zero-value bodies) so RED fails on assertions, never build failures — 04-06 precedent reused, both REDs validated RED_EVIDENCE_OK"
  - "Per-entry error isolation as the store-reader posture: one undecryptable entry skips with key-name+shape/length+code logs; the whole store never fails on entry failures"
  - "Never-plaintext rule: any enc: value outside the handled layouts skips — encrypted bytes can never degrade to plaintext"

requirements-completed: [AUTH-02]  # copied verbatim from the plan frontmatter; AUTH-02 is phase-spanning — ready-ids reports it blocked until 06-02..06-05 land (see Decisions Made)

plan_head_before: edad0689583e5648f9f784e5136c6cc892ddb071
plan_head_after: 34788aa27b35c3ded5f12cebfc1a4401ec40fbd9

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Credential-store reader: plaintext/enc:v1/enc:v2:os dispatch ladder with per-entry isolation, fp24 verification before decrypt, unknown-enc: never-plaintext skip, missing-file empty-nil and undecodable-store coded error"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/interact/credstore_test.go#TestCredStore"
        status: pass
      - kind: unit
        ref: "internal/interact/credstore_test.go#TestCredStoreMissingAndEmpty"
        status: pass
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestV1FallbackMalformedShapes"
        status: pass
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestV1FallbackWrongSecret"
        status: pass
    human_judgment: false
  - id: D2
    description: "Chain accessors: LoginToken strips exactly one leading 'Bearer ' (unprefixed values pass unchanged, absent families answer ok=false); ZcodeJWT serves the decrypted zcodejwttoken entry"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/interact/credstore_test.go#TestCredStoreAccessors"
        status: pass
    human_judgment: false
  - id: D3
    description: "Credential hygiene (T-5-17 inheritance): store logs carry entry-key names/shapes/lengths only — no decrypted values, no key material in any encoding"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/interact/credstore_test.go#TestCredStoreLogHygiene"
        status: pass
    human_judgment: false
  - id: D4
    description: "enc:v1 fallback loader (D-03) behind the KeyLoader seam: trimmed-env-else-machine-string secret resolution, sha256 key contract, loud empty-input validation, round-trip + malformed + wrong-secret parity vectors, interface assertion, no exec"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestV1FallbackRoundTrip"
        status: pass
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestV1FallbackSecretResolution"
        status: pass
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestV1FallbackLoudValidation"
        status: pass
      - kind: unit
        ref: "internal/interact/cipherdecrypt_test.go#TestV1FallbackDigest"
        status: pass
    human_judgment: false

# Metrics
duration: 24min
completed: 2026-11-22
status: complete
---

# Phase 6 Plan 01: Credential-Store Base Summary

**Credentials.json reader with the plaintext/enc:v1/enc:v2:os dispatch ladder, the never-plaintext unknown-prefix rule, and the enc:v1 fallback loader ported behind the existing KeyLoader seam — synthetic-fixture locked, -race and lint green**

## Performance

- **Duration:** 24 min
- **Started:** 2026-11-21T23:47:31Z
- **Completed:** 2026-11-22T00:11:47Z
- **Tasks:** 2 (tracer + auto, both tdd)
- **Files modified:** 4 code files + 2 RED evidence records

## Accomplishments

- `internal/interact/credstore.go`: the read-only credentials.json reader — plaintext passthrough verbatim, `enc:v1:` decrypt (sha256(secret) key), `enc:v2:os:` decrypt composed through the existing CredentialDecrypt service (fp24 verified before any GCM open, no parser fork), unknown `enc:` prefixes skipped with a structural log (never plaintext), per-entry isolation, missing file = empty-nil, undecodable store = coded `ErrStoreUndecodable` wrap
- `LoginToken(ctx, family)` strips exactly one leading `Bearer ` ASCII prefix and answers the `oauth:<family>:access_token` entry — the mint chain's step-1 credential; `ZcodeJWT(ctx)` serves `zcodejwttoken` (the accountScope input)
- `internal/interact/cipherdecrypt.go`: `V1FallbackKeyLoader` — the enc:v1 sibling joining `SecretToolKeyLoader` behind the SAME `KeyLoader` seam (D-01: one seam, two layouts; the Phase-7 godbus swap commitment kept verbatim in the seam doc) — trimmed `ZCODE_CREDENTIAL_SECRET` env wins, else the exact machine string `zcode-credential-fallback:<GOOS>:<home>:<username>`; key = sha256(secret); pure compute, no exec; empty machine inputs fail loud
- T-5-17 hygiene inherited and test-locked at the store tier: logs carry entry-key names/shapes/lengths only

## TDD Compliance

Both tasks ran full RED→GREEN cycles with machine-validated evidence (repo convention: compile-first skeletons so RED lands on assertions, never build failures):

- **Task 1 RED** `a02aeb2`: TestCredStore + TestCredStoreMissingAndEmpty failed on their planned dispatch/posture assertions (empty skeleton result; absent coded error). Evidence `evidence/06-01-red-credstore.json` → `RED_EVIDENCE_OK` (target_test_failed). **Semantic assessment:** genuine RED — both FAIL lines are assertion fatals at the planned checks; the missing/empty subtests passing green reflects the skeleton's empty-nil posture already being the missing-file contract, implying no behavior.
- **Task 1 GREEN** `49bdde9`: the dispatch ladder. Tracer gate: `<verify>` re-run green end-to-end under auto mode (`human_verify_mode=end-of-phase`, automated-only verify) — tracer verified, expansion authorized.
- **Task 2 RED** `2e6f658`: TestV1Fallback family failed on skeleton-shaped outputs (zero key → GCM auth failure in the round-trip; empty resolution; absent loud-validation error; zero digest). Evidence `evidence/06-01-red-v1fallback.json` → `RED_EVIDENCE_OK`. **Semantic assessment:** genuine RED on planned assertions; the malformed-shapes/wrong-secret subtests passing green exercise the Task-1 store dispatch already locked — documented as such, no Task-2 behavior implied.
- **Task 2 GREEN** `34788aa`: the sibling loader + store swap. No REFACTOR commit (lint cleanup rode GREEN).
- **Gate sequence:** `test(06-01)` ×2 precede `feat(06-01)` ×2 in ancestry — compliant.

## Task Commits

1. **Task 1 RED: credential-store dispatch tests vs compiling skeleton** — `a02aeb2` (test)
2. **Task 1 GREEN: the credential-store dispatch ladder** — `49bdde9` (feat)
3. **Task 2 RED: enc:v1 fallback parity vectors vs skeleton** — `2e6f658` (test)
4. **Task 2 GREEN: enc:v1 fallback loader behind the KeyLoader seam** — `34788aa` (feat)

## Files Created/Modified

- `internal/interact/credstore.go` — NEW: CredentialStore reader (dispatch ladder, isolation, accessors, EncV1Prefix, coded store sentinels, Apache-2.0 port attribution)
- `internal/interact/credstore_test.go` — NEW: TestCredStore / TestCredStoreAccessors / TestCredStoreLogHygiene / TestCredStoreMissingAndEmpty + the in-test v1 seal helper and fixture writer (all synthetic)
- `internal/interact/cipherdecrypt.go` — MOD: V1FallbackKeyLoader + NewV1FallbackKeyLoader + ResolveV1CredentialSecret + V1SecretEnvKey + errV1SecretInputs; KeyLoader seam doc names both layouts with the Phase-7 godbus commitment verbatim; shared cipherCannotConstruct const
- `internal/interact/cipherdecrypt_test.go` — MOD: the TestV1Fallback vector family + interface assertion + in-test v1 open helper

## Decisions Made

- **AUTH-02 left unchecked in REQUIREMENTS.md.** The requirement is phase-spanning (06-01 delivers the credential-store base; 06-02 mint, 06-03 snapshot, 06-05 authenticate complete it). `requirements.ready-ids` reports AUTH-02 `blocked` — mark-complete was NOT run, matching the 05-12 precedent that the flip is a post-verification act once the whole chain passes.
- **v2 arm composes `decryptOne`** — maximal reuse of the locked parse/fp24/GCM engine; the store adds zero crypto surface of its own on the v2 path.
- **enc:v1 blob parsing lives in credstore.go** (dispatch-local), only the key loader is the seam sibling — preserves task file ownership without a move refactor.
- **Log codes for store-level failures** (`store-unreadable`, `store-undecodable`, `unhandled-enc-prefix`, `v1-fallback-unavailable`) — short coded tokens in structural logs, sentinels exported (`ErrStoreUndecodable`, `ErrStoreUnreadable`) for consumer matching.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 2's GREEN composes the exported loader in credstore.go**
- **Found during:** Task 2 (v1 fallback loader)
- **Issue:** The plan's task file lists assign credstore.go to Task 1 only, but Task 1's locked constructor signature `NewCredentialStore(path, v2 KeyLoader)` cannot reference the Task-2 loader before it exists; Task 1 GREEN necessarily shipped an internal v1 key source.
- **Fix:** Task 2's GREEN swapped the internal `storeV1Loader` for `NewV1FallbackKeyLoader(nil, nil, nil, nil)` (the plan's own end-state: "one seam, two layouts"; an unconstructable fallback skips enc:v1 entries per entry) and deleted the temporary internal type.
- **Files modified:** internal/interact/credstore.go
- **Verification:** full package gate `go test -race -count=1 ./internal/interact/...` green after the swap; zero behavior change
- **Committed in:** 34788aa (Task 2 commit)

**2. [Rule 3 - Blocking] Test functions split to pass the strict lint gate**
- **Found during:** Task 2 (pre-commit gate run)
- **Issue:** The plan-named test functions as single bodies tripped the repo's canonical golangci-lint v2 gate (gocyclo 20 / gocognit 30 / goconst / lll / err113 / funcorder / gosec-G101 / gocritic-unnamedResult — 12 issues, all in new code).
- **Fix:** Split TestCredStore into TestCredStore/Accessors/LogHygiene and TestV1Fallback into six sibling functions (same vector families; the plan's `-run` regexes still match every name); static test sentinels; cipher-code constants in vectors; shared `cipherCannotConstruct` const (both files); nolint on the env-key name; funcorder fix. Lint now 0 issues.
- **Files modified:** internal/interact/credstore_test.go, internal/interact/cipherdecrypt_test.go, internal/interact/cipherdecrypt.go, internal/interact/credstore.go
- **Verification:** `golangci-lint run ./internal/interact/...` → 0 issues; full -race package gate green
- **Committed in:** 34788aa (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 blocking/task-ordering; zero architectural)
**Impact on plan:** Both deviations land the plan's own stated end-state; no scope creep, no behavior change. Estimate accuracy: 2 tasks / 4 commits as planned; ~12.0k tokens realized vs 38k estimated (skeleton strategy + locked Phase-5 seam reuse kept the diff small).

## Issues Encountered

- Two test-authoring compile fixes (a `keysOfCipher` type mismatch and a Fatalf verb/arg count) were corrected BEFORE the Task-1 RED evidence was captured, so the recorded RED lands only on planned assertions (documented in the evidence record).
- gosec G101 false-hit on the `ZCODE_CREDENTIAL_SECRET` env-key name — nolint with justification (the `secretToolService` precedent).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 06-02 (PAT minter) consumes `LoginToken(ctx, "zai")` as mint step 1 — the coded "log in via the ZCode desktop first" arm lives in the consumer; the reader answers ok=false on absence (locked by TestCredStoreAccessors).
- 06-03 (snapshot) folds `ZcodeJWT(ctx)` into accountScope; 06-05 (authenticate) wires the reader into the coding-plan arm.
- The KeyLoader seam is the Phase-7 godbus swap boundary — both loaders now sit behind it; no business logic bypasses the seam.

---
*Phase: 06-oauth-coding-plan-path-model-picker*
*Completed: 2026-11-22*
