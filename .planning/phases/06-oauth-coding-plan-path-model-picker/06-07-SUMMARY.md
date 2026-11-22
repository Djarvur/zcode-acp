---
phase: 06-oauth-coding-plan-path-model-picker
plan: 07
subsystem: auth
tags: [runtime-headers, pat, account-scope, d05-reactive-remint, acp-responder, fake-corpus, golden-lock, cmd-wiring]

# Dependency graph
requires:
  - phase: 06-02
    provides: the PatMinter (Cached/Ensure/Invalidate single-flight cache + the Pat fingerprint/scope fields) the source orchestrates
  - phase: 06-05
    provides: the D-11 auth-mode record (Front.AuthMode) the source selects on, and the AuthConfig Creds/Pat/Probe/Entitle seams cmd wires
  - phase: 05-06/05-07
    provides: the RuntimeHeadersKeySource responder seam, the static RuntimeHeadersHandler sibling, and the byte-locked runtime-headers-answer corpus row
  - phase: 06-01
    provides: the CredentialStore the production minter and entitlement gate compose over
provides:
  - interact.RuntimeHeadersKeySource widened to (apiKey, apiKeyID, accountScope, err) — accountScope joins requestAuth only when supplied; the api-key two-field answer stays byte-identical (the Phase-5 goldens untouched)
  - interact.RuntimeHeadersRejection(req) — the exported D-05 dig helper (expectedAccountScope / rejectedProjectTokenFingerprint, each gated on 64-lowercase-hex, malformed/absent/undecodable → empty)
  - hostsvc.PatHeadersSource(PatHeadersSourceDeps{Minter, Mode, APIKeyFallback}) — the mode-aware source: cached answer with zero mints, exactly-one Invalidate+Ensure on the fingerprint match, ErrPatRejectionLoop on the second consecutive rejection, ErrPatScopeInvalidated refusal (outranking the re-mint), ErrPatLoginRequired PAT-less arm, byte-compatible api-key/None delegation
  - cmd wiring (D-12: no new user config): buildCodingPlanMachinery (store + v2 secret-tool loader, proactive prober, setting.json entitlement gate, PatMinter with the A6 clientVersion install dig) → buildFront's AuthConfig Creds/Pat/Probe/Entitle + wireResponder's mode-aware source; every seam lazy (D-13)
  - fake corpus row runtime-headers-pat (+ golden + corpusPatMinter driver): ask 1 plain answers the cached PAT-A three-field arm; ask 2 carries expectedAccountScope + fingerprint(PAT-A) and answers the re-minted PAT-B — the D-05 sequence byte-locked on the wire
affects: [06-08, verify-work, phase real-tier run (the live coding-plan cycle consumes this source)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 22500        # 90014 diff chars / 4 over the realized diff
  tasks: 3
  commits: 5           # MEASURED: git rev-list --count 8812e09..1b1499b

# Tech tracking
tech-stack:
  added: []            # zero new dependencies (stdlib only — T-06-SC held)
  patterns:
    - "per-ask re-mint bookkeeping: the D-05 drop-and-remint sequence lives INSIDE one handler invocation — never shared mutable retry state across asks"
    - "gate ordering as a locked arm: the scope refusal outranks the fingerprint re-mint; the fresh token re-checks the expectation post-mint"
    - "one-cache composition in tests: the corpus double sits behind BOTH the front's PatEnsurer and the responder's source (the production one-minter lifecycle)"
    - "trimpath-safe source parsing: runtime.Caller under GOFLAGS=-trimpath returns module-relative paths — resolve bare from the package-dir cwd"

key-files:
  created:
    - internal/hostsvc/headers_pat.go
    - internal/hostsvc/headers_pat_test.go
    - internal/zcode/fake/testdata/scenarios/runtime-headers-pat.json
    - internal/zcode/fake/testdata/golden/runtime-headers-pat.golden
  modified:
    - internal/interact/responder.go
    - internal/interact/responder_test.go
    - internal/hostsvc/snapshot.go
    - internal/hostsvc/patminter_test.go
    - cmd/zcode-acp/main.go
    - cmd/zcode-acp/main_test.go
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "The scope digest covers the login, not the token — the realistic fingerprint re-ask echoes the SAME accountScope and the re-minted token keeps it (arm (b) and the corpus ask 2 lock this shape)"
  - "The scope gate runs BEFORE the fingerprint arm and the fresh token re-checks the ask's expectation post-mint — a re-mint against a changed sign-in answers the coded refusal, never the mismatched token (T-06-22)"
  - "hostsvc cannot import acpfront (the front imports hostsvc) — the mode selector contract is the documented string value 'coding-plan' with cmd wiring front.AuthMode as the method value"
  - "The minter's coded errors pass through UNWRAPPED from the re-mint arm — the sentinel is the wire's match surface at the deliver seam (nolint:wrapcheck with the contract reason)"
  - "clientVersion seam: best-effort version-segment dig over the install paths (deepest segment first), the reference's documented '0.0.0' fallback otherwise (A6: no server-side validation documented; the real run confirms)"

patterns-established:
  - "compiling-skeleton RED for widening seams: the new signature lands with the old behavior intact (scope received-and-discarded) so RED fails on assertions while every existing call site compiles"
  - "threshold-tipping lint discipline: const-extract ONLY the new occurrences so pre-existing literals drop back under the goconst count without touching sibling files"
  - "deterministic concurrency shape: ONE trigger ask plus plain riders — counters stay exact while -race proves the riders uncorrupted"

requirements-completed: [AUTH-02]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "The widened RuntimeHeadersKeySource seam: accountScope joins requestAuth only when the source supplies it; the api-key two-field answer stays byte-compatibly identical (the Phase-5 corpus golden passes untouched) and the coded-error posture carries"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/interact/responder_test.go#TestRuntimeHeadersSeamWidening"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/runtime-headers-answer"
        status: pass
    human_judgment: false
  - id: D2
    description: "RuntimeHeadersRejection dig helper: camelCase top-level digs of expectedAccountScope / rejectedProjectTokenFingerprint, each validated as 64-lowercase-hex before return; malformed/absent/undecodable classify as empty; values never logged"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/interact/responder_test.go#TestRuntimeHeadersRejection"
        status: pass
    human_judgment: false
  - id: D3
    description: "The mode-aware PAT source: cached answer with zero mints, exactly-one Invalidate+Ensure on the fingerprint match, coded stop on the second consecutive rejection (two total mints), scope refusal before any token (outranking the re-mint), PAT-less coded arm, byte-compatible api-key/None delegation, malformed/mismatching field classifies absent, concurrent riders uncorrupted — all under -race"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/hostsvc/headers_pat_test.go#TestPatHeadersSource"
        status: pass
    human_judgment: false
  - id: D4
    description: "cmd wiring with no new user configuration (D-12/D-13): the machinery constructs lazily, buildFront wires Creds/Pat/Probe/Entitle, the responder serves through the mode-aware source, and the corpus row byte-locks the D-05 re-ask (PAT-A then the re-minted PAT-B) — mise run ci + scan-secrets green"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "cmd/zcode-acp/main_test.go#TestWiringCodingPlanSeams"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/runtime-headers-pat"
        status: pass
      - kind: e2e
        ref: "cmd/zcode-acp/main_test.go#TestAgentBinaryEndToEnd"
        status: pass
    human_judgment: false

# Metrics
duration: 30min
completed: 2026-11-22
status: complete
---

# Phase 6 Plan 07: Runtime-Headers PAT Serving Summary

**The widened runtime-headers seam carries accountScope (api-key answers byte-identical), the mode-aware PAT source answers from the in-process cache with the D-05 exactly-one re-mint and the scope refusal, and cmd wires the whole coding-plan machinery with zero new user config — the re-ask sequence byte-locked on the wire**

## Performance

- **Duration:** 30 min
- **Started:** 2026-11-22T03:53:37Z
- **Completed:** 2026-11-22T04:23:58Z
- **Tasks:** 3 (tasks 1-2 TDD RED→GREEN with machine-validated RED evidence)
- **Files modified:** 13 (4 created, 9 modified)

## Accomplishments
- The responder seam widened (Pattern 4/A9): `RuntimeHeadersKeySource` returns (apiKey, apiKeyID, accountScope, err); the success arm joins `accountScope` ONLY when supplied — the api-key two-field answer stays byte-identical and the Phase-5 corpus golden passes untouched
- `RuntimeHeadersRejection` digs the D-05 trigger fields with the strict 64-lowercase-hex gate (malformed/absent/undecodable → empty; never logged)
- The mode-aware source (Pattern 4/D-05): cached answer with zero mints; a matching `rejectedProjectTokenFingerprint` drops the cache and re-mints EXACTLY once per ask; a second consecutive rejection answers the coded `ErrPatRejectionLoop` (mints stop at two total); the scope gate refuses `ErrPatScopeInvalidated` BEFORE any token and outranks the re-mint; the PAT-less cache answers the coded unauthenticated arm; api-key/None modes delegate byte-compatibly to the static handler
- cmd wiring (D-12): `buildCodingPlanMachinery` composes the 06-01 store (with its v2 secret-tool loader), the D-02 prober, the setting.json entitlement gate, and the 06-02 minter (A6 clientVersion: install-tree dig, documented "0.0.0" fallback); buildFront wires Creds/Pat/Probe/Entitle; wireResponder serves through the mode-aware source. NO new flags/env reads; every seam executes lazily (D-13)
- The runtime-headers-pat corpus row: ask 1 plain answers the cached PAT-A three-field arm, ask 2 (expectedAccountScope + fingerprint(PAT-A)) answers the re-minted PAT-B under the unchanged scope — one `corpusPatMinter` double behind both the front's mint and the source (the production one-cache lifecycle)
- Wave gate: `mise run ci` green (build + vet + lint + race suites + experiments); `experiments/scan-secrets.sh` green

## TDD Gate Compliance

Two RED→GREEN pairs, in order (git log ancestry):

| Task | RED commit | GREEN commit | RED evidence |
|------|-----------|--------------|--------------|
| 1 seam widening + dig helper | `03bf425` test(06-07) | `3bc8dff` feat(06-07) | evidence/06-07-red-seam.json — RED_EVIDENCE_OK: the coding-plan arm failed on the missing accountScope and the well-formed dig failed on the stub's empty; the byte-compat arm and the malformed/absent dig classes were the documented expected-at-RED lock greens |
| 2 mode-aware PAT source | `6ff9bf7` test(06-07) | `75e1fd5` feat(06-07) | evidence/06-07-red-patsource.json — RED_EVIDENCE_OK: all eight coding-plan arms failed against the full-delegation skeleton (each showing the unconfigured-fallback error or the absent coded class); the api-key delegation arm was the documented expected-at-RED lock green |

Each RED's `semanticAssessment` records genuine assertion failures (no build failures, zero-test discovery, fixture crashes, or unrelated failures). Task 3 is the plan's non-TDD wiring task (single feat commit).

## Task Commits

1. **Task 1: Seam widening red-first + byte-compat regression** — `03bf425` (test RED) + `3bc8dff` (feat GREEN)
2. **Task 2: Mode-aware PAT source arms → implementation** — `6ff9bf7` (test RED) + `75e1fd5` (feat GREEN)
3. **Task 3: cmd wiring (no new flags) + runtime-headers-pat corpus row** — `1b1499b` (feat)

**Plan metadata:** pending (the docs commit follows this SUMMARY)

_Note: TDD tasks produced test→feat pairs per the RED→GREEN gate._

## Files Created/Modified
- `internal/hostsvc/headers_pat.go` — PatHeadersSource + PatHeadersSourceDeps, the patMintSource surface, ErrPatRejectionLoop + errPatSourceUnwired sentinels, the per-ask D-05 decision tree
- `internal/hostsvc/headers_pat_test.go` — TestPatHeadersSource: ten named arms (happy, fingerprint, second-rejection, ensure-failure pass-through, scope refusal, scope-first ordering, empty cache, api-key delegation, malformed field, concurrent riders) under -race
- `internal/interact/responder.go` — the widened seam + the scope-join success arm + RuntimeHeadersRejection (paramsHex64/isLowerHex64)
- `internal/interact/responder_test.go` — TestRuntimeHeadersSeamWidening (byte-compat, scope join, coded error) + TestRuntimeHeadersRejection (verbatim, absent, malformed classes, undecodable) + the widened expiry closure
- `internal/hostsvc/snapshot.go` — RuntimeHeadersHandler adapted to the widened signature (mechanical; behavior unchanged)
- `internal/hostsvc/patminter_test.go` — TestPatMemoryOnly's source path made trimpath-safe (test-only; Rule 3)
- `cmd/zcode-acp/main.go` — settingPath, desktopVersion/patClientVersion (A6), codingPlanMachinery + buildCodingPlanMachinery, buildFront's four seam fields, wireResponder's mode-aware source
- `cmd/zcode-acp/main_test.go` — TestWiringCodingPlanSeams (four named wiring vectors) beside the untouched E2E rides
- `internal/zcode/fake/golden_test.go` — the runtime-headers-pat driver + corpusPatMinter double + fixture constants + registrations; the scripted auth-cycle closure adapted (byte-compat)
- `internal/zcode/fake/testdata/scenarios/runtime-headers-pat.json` + `.golden` — the coding-plan session with the two-ask re-ask sequence byte-locked

## Decisions Made
- The realistic re-ask shape carries BOTH the echoed expectedAccountScope AND the rejected fingerprint; the re-minted token keeps the scope (the digest covers the login, not the token) — locked by arm (b) and the corpus ask 2
- Scope gate ordering: refusal BEFORE the re-mint, and the fresh token re-checks the ask's expectation post-mint (the re-mint-against-changed-signin race window closes honestly)
- The minter surface stays an unexported interface (`patMintSource`) satisfied structurally by *PatMinter — the deps struct is the only exported construction surface
- The corpus double is ONE minter behind the front's PatEnsurer AND the source (the production single-cache lifecycle), not two scripted sources
- `mise run ci` wave-gate green required a trimpath-safety fix in TestPatMemoryOnly's file resolution (see deviations)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 1's compile completion outside its declared files**
- **Found during:** Task 1 (seam widening RED)
- **Issue:** widening `RuntimeHeadersKeySource` breaks two undeclared call sites: `hostsvc.RuntimeHeadersHandler`'s returned literal (snapshot.go) and golden_test.go's scripted auth-cycle closure — the task's own verify command runs the fake package, which could not compile
- **Fix:** mechanical signature adaptation only (the adapter and the scripted closure carry an empty scope; behavior and corpus bytes unchanged — the coding-plan-auth-cycle row's expect_answer stays the two-field shape, which IS the A9 byte-compat lock)
- **Files modified:** internal/hostsvc/snapshot.go, internal/zcode/fake/golden_test.go
- **Verification:** go build ./... + the full fake suite green; the runtime-headers-answer golden passes unchanged
- **Committed in:** 03bf425 (RED commit)

**2. [Rule 1 - Bug] Fixture constant length error**
- **Found during:** Task 1 GREEN (first verify run)
- **Issue:** `seamFixtureScope` was typed with 68 hex chars (not 64) — the well-formed dig subtest failed against a correct implementation
- **Fix:** regenerated the a/b fixture constants at exactly 64 chars (scripted regex rewrite)
- **Files modified:** internal/interact/responder_test.go
- **Verification:** TestRuntimeHeadersRejection green under -race
- **Committed in:** 3bc8dff (GREEN commit)

**3. [Rule 1 - Bug] Task 2 arm-fixture design errors (production logic was right)**
- **Found during:** Task 2 GREEN (first verify run)
- **Issue:** (a) the fingerprint arm scripted the re-minted PAT with a DIFFERENT scope than the echoed expectedAccountScope — the source's honest post-mint expectation check refused it; the realistic shape is the unchanged scope (the digest covers the login). (b) the concurrency arm fired 8 fingerprint asks against one stale cache — after the first re-mint the cache holds the fresh token no fingerprint matches, so exactly 1 re-mint is CORRECT
- **Fix:** arm (b)'s fresh PAT keeps scopeA; the race arm became ONE trigger ask plus 7 plain riders (deterministic counters, shape-level answer validation) — both production behaviors unchanged and now correctly specified
- **Files modified:** internal/hostsvc/headers_pat_test.go
- **Verification:** all ten arms green under -race; lint clean
- **Committed in:** 75e1fd5 (GREEN commit)

**4. [Rule 3 - Blocking] TestPatMemoryOnly broke the repo's `-trimpath` CI posture**
- **Found during:** Task 3 verification (the plan's `mise run ci` wave gate)
- **Issue:** the 06-02 hygiene lock resolved patminter.go via `runtime.Caller`; under the repo's `GOFLAGS=-trimpath` (mise.toml) the caller path is module-relative, so the parse failed and the gate went red — pre-existing since 06-02 (06-05/06-06 verified with `go test ./internal/...`, not the mise ci gate), NOT caused by this plan's changes, but blocking this plan's mandated wave gate
- **Fix:** when the caller-derived path is not absolute, resolve the sibling source bare from the test binary's cwd (the package dir) — test-only, zero behavior change
- **Files modified:** internal/hostsvc/patminter_test.go
- **Verification:** green under both `GOFLAGS=-trimpath` and plain modes; full `mise run ci` green end-to-end
- **Committed in:** 1b1499b (Task 3 commit)

---

**Total deviations:** 4 auto-fixed (2 blocking compile/gate, 2 test-fixture bugs)
**Impact on plan:** All four necessary — the compile completion and the trimpath fix were required for the plan's own verification commands; the fixture fixes corrected the tests to the locked semantics without touching production behavior. No scope creep.

## Issues Encountered
- My new corpus literals tipped three capture-verbatim wire-key strings over the goconst package-wide threshold (flagging pre-existing literals in roundtrip_test.go too) — resolved by const-extracting ONLY my occurrences, dropping every count back under the threshold without editing sibling files
- golangci-lint's funlen/gocognit gates shaped the test layout: TestPatHeadersSource split into ten named arm funcs and the wiring test into four named vectors (the established familysel/patminter convention), and driveRuntimeHeadersPat's preamble extracted to a helper
- go test's flag ordering: `-update` must follow the package path (a leading `-update` is rejected) — recorded here for future golden recording

## Known Stubs
None — every arm is fully wired; the corpus row byte-locks real behavior; no placeholder data flows to any surface.

## Authentication Gates
None — no external service authentication was required.

## User Setup Required
None - no external service configuration required. (The phase's real-account live run remains the phase-level bar per D-12/D-13, landed by the phase's real-tier plan.)

## Next Phase Readiness
- 06-08 and the phase verification can consume: the widened seam + RuntimeHeadersRejection, PatHeadersSource (cmd-wired), the corpus row's re-ask shapes, and the green `mise run ci` posture
- The phase's real-tier coding-plan run (success criterion 1) now has the full ask-side path: authenticate's mint-once warms the cache this source serves; A6's clientVersion value gets its live confirmation there
- Deferred: none new (the pre-existing wave-1 lint debt ledger was cleared by eb49f84 before this plan; the trimpath test bug is FIXED, not deferred)

---
*Phase: 06-oauth-coding-plan-path-model-picker*
*Completed: 2026-11-22*

## Self-Check: PASSED

- All 7 key files exist on disk (4 created + the 3 primary modified files; the remaining modified files are present per git)
- All 5 plan commits are ancestors of HEAD: 03bf425, 3bc8dff, 6ff9bf7, 75e1fd5, 1b1499b
- TDD gate sequence: two test(06-07) → feat(06-07) pairs in ancestry order (each GREEN descends from its RED)
- Commits measured from the ledger: 8812e09..HEAD = 5 (plan_head_before: 8812e09484e05d60a3328083de079c047448636a, plan_head_after: 1b1499b4fbbfab7946fc4731aa45c7563179b7c2)
