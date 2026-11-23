---
phase: 06-oauth-coding-plan-path-model-picker
plan: 05
subsystem: auth
tags: [keyring-probe, secret-tool, entitlement-gate, credential-store, oauth-coding-plan, pat-mint, registry-read-back, fake-corpus, golden-lock]

# Dependency graph
requires:
  - phase: 06-01
    provides: the CredentialStore reader (LoginToken/ZcodeJWT, the dispatch ladder) and the KeyLoader seam the probe composes
  - phase: 06-02
    provides: the PatMinter (Ensure single-flight cache) behind the PatEnsurer consumer seam
  - phase: 06-03
    provides: SnapshotBuilder.BuildAccount(install, entitled) — the non-empty account overlay with the caller-supplied entitlement gate
  - phase: 05-07/05-11
    provides: the api-key authenticate chain (watcher-before-push, refresh, probeCreateReadback, Await verdict) this arm clones
provides:
  - interact.KeyringProber (func-typed seam) + the production prober: LookPath → the loader's LookupCommand argv under a bounded 5s ctx → four coded arms (install hint naming libsecret-tools / no-desktop-session / log-in-via-desktop / never-hangs timeout)
  - interact.EntitlementGate: Entitled(ctx, family) over setting.json selections/modes + the 06-01 store token, read-only with honest degradation
  - acpfront.(*Front).authenticateCodingPlan: probe → login → BuildAccount overlay → snapshot-derived want list → watcher-before-push → refresh → probe-create gate → Await verdict → mint-once → success
  - acpfront.AuthConfig seams Creds/Pat (PatEnsurer)/Probe/Entitle + the D-11 auth-mode record (Front.authMode atomic + AuthMode() accessor)
  - fake corpus rows coding-plan-auth-cycle + coding-plan-probe-fail (+ goldens, drivers, post-assert, golden normalizers)
affects: [06-06, 06-07 (responder runtime-headers source reads AuthMode; cmd wires Creds/Pat/Probe/Entitle), 06-08, verify-work]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 28500        # 113995 diff chars / 4 over the realized diff
  tasks: 3
  commits: 8           # MEASURED: git rev-list --count dfde615..310a080

# Tech tracking
tech-stack:
  added: []            # zero new dependencies (exec + stdlib only — T-06-SC held)
  patterns:
    - "compiling-skeleton RED: new files land as signature-complete zero-behavior skeletons so RED fails on assertions, never build failures (06-01 precedent)"
    - "machine-derived-token golden normalization: the front-generated push revision + builtin-table hash normalize to fixture tokens (goldenNormalizers) — T-5-16 formulas stay uncached, every other byte stays locked"
    - "per-row corpus post-asserts as the semantic belt to the byte-lock braces"
    - "mode-record-once: the Authenticate dispatch is the single authMode write point; a failed/cancelled arm records nothing"

key-files:
  created:
    - internal/interact/keyringprobe.go
    - internal/interact/keyringprobe_test.go
    - internal/interact/familysel.go
    - internal/interact/familysel_test.go
    - internal/zcode/fake/testdata/scenarios/coding-plan-auth-cycle.json
    - internal/zcode/fake/testdata/golden/coding-plan-auth-cycle.golden
    - internal/zcode/fake/testdata/scenarios/coding-plan-probe-fail.json
    - internal/zcode/fake/testdata/golden/coding-plan-probe-fail.golden
  modified:
    - internal/acpfront/auth.go
    - internal/acpfront/auth_test.go
    - internal/acpfront/front.go
    - internal/zcode/fake/golden_test.go
    - experiments/scan-secrets.sh

key-decisions:
  - "AuthConfig.Entitle is a func(ctx, family) bool seam (mirroring the OQ3 prober shape) — 06-07 wires gate.Entitled as a method value; tests inject canned decisions"
  - "PatEnsurer defined at the acpfront consumer side (interface over hostsvc.Pat) — *hostsvc.PatMinter satisfies it structurally; no patminter.go edit needed"
  - "The absent modelProviderFamilyModes entry does NOT veto entitlement — the verified rule's JS strict-inequality semantics (`!== \"apiKey\"`), locked by test arm (g)"
  - "The arm's login read is a PRESENCE gate only — the mint seam re-reads its own credential (the login value never crosses into front state)"
  - "Corpus determinism via per-row golden normalizers (push revision + builtin hash → fixture tokens) instead of new AuthConfig revision seams — the plan's locked config shape stays Creds/Pat/Probe/Entitle only"
  - "The probe-fail corpus row records a scripted unknown-method canary as its only frame — a zero-frame transcript is the harness failure signature (checkExchangeRecorded)"

patterns-established:
  - "compiling-skeleton RED for new-file tasks: signatures + sentinels + stub body committed with the failing tests (06-01/06-03 precedent, reused 3×)"
  - "goldenNormalizers: per-row machine-derived-token substitution documented in-file (T-5-16 formulas are machine-derived by design; forms stay byte-locked)"
  - "assertNoCodingPlanTraffic / assertCodingPlanProbeFailTranscript: shared transcript-absence helpers as the D-02 ordering lock"

requirements-completed: [AUTH-02, AUTH-03, AUTH-04]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "Proactive keyring probe (AUTH-04/D-02): LookPath gate → the loader's LookupCommand argv under a bounded 5s ctx with the scrubbed env → coded arms (libsecret-tools install hint, no-desktop-session hint, log-in-via-desktop hint, never-hangs timeout), coarse-signal classification with no stderr verbatim and no values in logs"
    requirement: AUTH-04
    verification:
      - kind: unit
        ref: "internal/interact/keyringprobe_test.go#TestKeyringProbe"
        status: pass
    human_judgment: false
  - id: D2
    description: "Production entitlement gate (AUTH-02 Pattern 2 wrapper half): Entitled(ctx, family) over setting.json providerFamilyConnectionSelections/modelProviderFamilyModes + the 06-01 store token; read-only, honest degradation on missing/unreadable/undecodable setting.json, presence-booleans-only logging"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/interact/familysel_test.go#TestFamilySelections"
        status: pass
    human_judgment: false
  - id: D3
    description: "authenticateCodingPlan composes the official chain at the front tier (probe → login → overlay push → snapshot-derived want list → registry verdict → mint-once → auth-mode record) with coded errors per step and the ack never the verdict (PITFALL 1); six composed arms incl. the api-key regression"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "internal/acpfront/auth_test.go#TestCodingPlanAuth"
        status: pass
    human_judgment: false
  - id: D4
    description: "AUTH-03 both-methods: authenticate{zai-api-key} arms stay green unchanged with the coding-plan seams present (TestAuthenticate suite + composed arm f), and the coding-plan method authenticates end-to-end"
    requirement: AUTH-03
    verification:
      - kind: unit
        ref: "internal/acpfront/auth_test.go#TestAuthenticate"
        status: pass
      - kind: unit
        ref: "internal/acpfront/auth_test.go#TestCodingPlanAuth/api-key_arm_stays_green_with_the_coding-plan_seams_present"
        status: pass
    human_judgment: false
  - id: D5
    description: "Fake corpus rows byte-lock the coding-plan wire: the auth cycle (push with the account overlay visible in the ack's providerCount, the registry patch carrying the account models, the settings snapshot with the thoughtLevel block, the cipher + PAT reverse asks answered with expect_answer byte-compare) and the probe-fail ordering (zero provider frames on a probe failure)"
    requirement: AUTH-02
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/coding-plan-auth-cycle"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/coding-plan-probe-fail"
        status: pass
    human_judgment: false
  - id: D6
    description: "AUTH-04's field posture on a real headless/desktop machine (the probe's real secret-tool failure modes, the real keyring read end-to-end through authenticate) — the phase's real-tier bar (D-12) lands with the phase's live run, not in this plan"
    requirement: AUTH-04
    verification: []
    human_judgment: true
    rationale: "The probed failure table was V-EXP'd at research time on this machine, but the composed chain's live behavior (real keyring → real mint → real registry) is the phase-level success criterion verified at the phase's real run — automation here cannot substitute for the field check"

# Metrics
duration: 46min
completed: 2026-11-22
status: complete
---

# Phase 6 Plan 05: Coding-Plan Authenticate Arm Summary

**The coding-plan authenticate chain end-to-end: proactive keyring probe (5s-bounded, three coded actionable arms), production entitlement gate over read-only desktop state, account-overlay push with a snapshot-derived want list, registry read-back verdict, one PAT mint after the verdict — plus two fake corpus rows byte-locking the happy cycle and the probe-fail ordering**

## Performance

- **Duration:** 46 min
- **Started:** 2026-11-22T01:52:51Z
- **Completed:** 2026-11-22T02:39:30Z
- **Tasks:** 3 (each TDD RED→GREEN with machine-validated RED evidence)
- **Files modified:** 16 (8 created, 8 modified)

## Accomplishments
- The `authenticate{zai-individual-coding-plan}` decline arm is REPLACED by the composed official chain (AUTH-02): probe (D-02, before ANY wire traffic) → desktop login read (D-03/D-12) → BuildAccount overlay with the production entitlement gate → want list derived from the BUILT snapshot's providers (D-08's verdict-side twin) → watcher registered before push → refresh → probe-create branch-(c) gate → Await verdict (PITFALL 1) → Pat.Ensure EXACTLY ONCE after the verdict (D-04)
- The proactive keyring probe (AUTH-04): LookPath first (the coded install hint names libsecret-tools, zero exec), the loader's exact LookupCommand argv (one argv source) under context.WithTimeout at 5s with the scrubbed env, coarse-signal classification (autolaunch marker containment → no-desktop-session hint; never stderr verbatim), and the never-hangs lock proven by a hung-seam vector returning inside its bound
- The production entitlement gate: setting.json selections `.kind == individual-coding-plan` AND modes `!= apiKey` (absent entry passes — the verified strict-inequality rule) AND the 06-01 store token; every degradation arm answers false with a structural class code, never a panic, never a value in a log
- The D-11 auth-mode record: `Front.authMode` (atomic) written ONLY by the Authenticate dispatch after a read-back-proven success — both advertised methods record their mode; a cancelled/failed authenticate records nothing (clean retry)
- Two fake corpus rows + goldens: coding-plan-auth-cycle (push with providerCount 1, the patch carrying the account models, the settings snapshot with the thoughtLevel block, the cipher + PAT reverse asks answered with expect_answer byte-compare) and coding-plan-probe-fail (ZERO provider frames — the D-02 ordering byte-locked)
- Wave gate green: `go test -race -count=1 ./internal/...` all packages; `experiments/scan-secrets.sh` green (with the corpus fixture blob allowlisted per the 03-03/05-06 convention)

## TDD Gate Compliance

Three RED→GREEN pairs, in order (git log ancestry):

| Task | RED commit | GREEN commit | RED evidence |
|------|-----------|--------------|--------------|
| 1 keyring probe | `a2f0cc3` test(06-05) | `5d89753` feat(06-05) | evidence/06-05-red-probe.json — RED_EVIDENCE_OK: all six subtests failed their errors.Is assertions against the compiling skeleton (the stub answered the not-implemented marker) |
| 2 entitlement gate | `b0e352d` test(06-05) | `33e11e0` feat(06-05) | evidence/06-05-red-famsel.json — RED_EVIDENCE_OK: the four rule-carrying arms failed on planned assertions; the three negative veto arms documented as expected-at-RED lock greens (06-02/06-03 precedent) |
| 3 composed arm + corpus | `9418de2` test(06-05) | `0339d54` feat(06-05) | evidence/06-05-red-cpauth.json — RED_EVIDENCE_OK: all six composed arms failed on planned assertions (five against the Phase-6 decline arm; arm f on the not-yet-landed mode record, a GREEN deliverable) |

Each RED's `semanticAssessment` records genuine assertion failures (no build failures, zero-test discovery, fixture crashes, or unrelated failures). Two follow-on commits: `65ab331` refactor(06-05) lint hygiene (no behavior change) and `310a080` chore(06-05) secret-gate allowlist.

## Task Commits

1. **Task 1: Keyring probe (AUTH-04, D-02)** — `a2f0cc3` (test RED) + `5d89753` (feat GREEN)
2. **Task 2: Production entitlement gate (familysel.go)** — `b0e352d` (test RED) + `33e11e0` (feat GREEN)
3. **Task 3 (tracer): authenticateCodingPlan arm + auth corpus** — `9418de2` (test RED) + `0339d54` (feat GREEN) + `65ab331` (refactor lint) + `310a080` (chore gate allowlist)

**Plan metadata:** pending (the docs commit follows this SUMMARY)

_Note: TDD tasks produced test→feat pairs per the RED→GREEN gate._

## Files Created/Modified
- `internal/interact/keyringprobe.go` — the KeyringProber func seam + production prober (LookPath gate, loader argv, bounded exec, four coded arms)
- `internal/interact/keyringprobe_test.go` — six arms: missing-binary/no-exec, autolaunch coarse-classification, empty-answer, argv contract (one argv source + bounded ctx), unresolvable path, never-hangs
- `internal/interact/familysel.go` — EntitlementGate over setting.json + the 06-01 store; honest degradation classes
- `internal/interact/familysel_test.go` — seven arms incl. the absent-mode strict-inequality lock
- `internal/acpfront/auth.go` — the coding-plan arm replacing the decline; codingPlanLogin prologue; wantModelsFromSnapshot derivation; PatEnsurer interface; AuthConfig Creds/Pat/Probe/Entitle seams; the D-11 mode constants + recordAuthMode/AuthMode
- `internal/acpfront/front.go` — the Front.authMode atomic field (one struct field; the accessor lives in auth.go)
- `internal/acpfront/auth_test.go` — TestCodingPlanAuth six composed arms + the coding-plan fixture harness
- `internal/zcode/fake/golden_test.go` — the two corpus drivers, the normalizers, the probe-fail post-assert, the corpusPat/cipher fixtures
- `internal/zcode/fake/testdata/scenarios/coding-plan-auth-cycle.json` + `.golden` — the happy cycle byte-lock
- `internal/zcode/fake/testdata/scenarios/coding-plan-probe-fail.json` + `.golden` — the probe-fail ordering byte-lock
- `experiments/scan-secrets.sh` — the corpus fixture blob's exact-string allowlist entry (03-03/05-06 convention)

## Decisions Made
- `AuthConfig.Entitle` is a func seam (not the concrete gate type) — method-value wiring for 06-07, canned decisions for tests (mirrors the OQ3 prober shape)
- `PatEnsurer` declared at the acpfront consumer side — `*hostsvc.PatMinter` satisfies it structurally, keeping patminter.go untouched (stays within the plan's declared files)
- Absent modes entry passes the mode gate — the verified rule's JS `!== "apiKey"` semantics, documented and test-locked
- The arm's login read is a presence gate; the mint seam re-reads its own credential (the value never crosses into front state — hygiene)
- Corpus determinism via goldenNormalizers (push revision + builtin hash → fixture tokens) rather than new AuthConfig seams — the plan's locked config shape preserved; every non-machine-derived byte stays locked
- The probe-fail row's only frame is a scripted unknown-method canary (a zero-frame transcript is the harness failure signature); the semantic belt post-assert forbids provider frames

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] front.go gained the Front.authMode atomic field**
- **Found during:** Task 3 (the composed arm)
- **Issue:** The plan's artifacts require "the Front's auth-mode record (atomic field + accessor)" but front.go (where the Front struct lives) was not in the task's declared files
- **Fix:** One struct field (`authMode atomic.Value`) added to Front; the constants, accessor (AuthMode), and the single write point (recordAuthMode) live in auth.go per the declared file
- **Files modified:** internal/acpfront/front.go
- **Verification:** TestCodingPlanAuth arms assert mode-recorded/mode-none per arm; -race green
- **Committed in:** 9418de2 (skeleton) / 0339d54 (write point)

**2. [Rule 3 - Blocking] scan-secrets allowlist entry for the corpus fixture blob**
- **Found during:** Task 3 verification (scan-secrets gate)
- **Issue:** The corpus's fixture cipher blob is a REAL enc:v2:os layout (sealed over synthetic plaintext under a frozen fixture key) — its opaque body matches the gate's credential-shape pattern; the sealed body cannot self-declare
- **Fix:** Added the blob's exact string to the gate's allowlist (the same convention as 03-03's and 05-06's fixture markers), with an in-file comment recording the fixture key/plaintext/kid provenance
- **Files modified:** experiments/scan-secrets.sh
- **Verification:** scan-secrets green; the pattern classes untouched
- **Committed in:** 310a080

**3. [Rule 1 - Bug] Happy arm's patch assertion indexed the transcript naively**
- **Found during:** Task 3 GREEN (first test run)
- **Issue:** `tr.Lines[refreshIdx[0]-1]` assumed the emit sat immediately before the refresh request line — the emit actually rides a different direction slot
- **Fix:** Scan the transcript for the state.updated emit line carrying the account provider + both models
- **Files modified:** internal/acpfront/auth_test.go
- **Verification:** testCodingPlanHappy green
- **Committed in:** 0339d54

---

**Total deviations:** 3 auto-fixed (2 blocking/missing-piece, 1 test bug)
**Impact on plan:** All three necessary for the plan's goal (the auth-mode record and the corpus are plan artifacts; the gate green is a plan verification requirement). No scope creep.

## Issues Encountered
- `mise run lint` fails on PRE-EXISTING violations the wave-1 merge shipped (front.go NewSession funlen, snapshot.go funlen/err113/tagliatelle, acpfront_test goconst over 05-07/06-04 literals). All violations INTRODUCED by this plan were fixed (`65ab331`); the remaining ones are recorded in the phase's `deferred-items.md` per the scope boundary. The `mise run ci` lint task failure is pre-existing and NOT caused by this plan.
- familysel.go's two tagliatelle hits are the accepted capture-verbatim class (camelCase wire keys REQUIRED by the desktop-read discipline — a snake_case tag would break the read); same class as snapshot.go's four pre-existing hits.
- The composed-tier happy arm skips the probe-create read-back by design when the refresh's patch latches (the watcher's Applied() gate) — deterministic because conn observers deliver synchronously in the read loop; the corpus row relies on this for byte-determinism.

## Known Stubs
None — every arm is fully wired; the corpus rows byte-lock real behavior; no placeholder data flows to any surface. (The cmd wiring of the production seams — Creds/Pat/Probe/Entitle into AuthConfig and the PAT minter into the responder — is plan 06-07's declared scope per D-12, not a stub.)

## Authentication Gates
None — no external service authentication was required.

## User Setup Required
None - no external service configuration required. (The phase's real-account live run is the phase-level bar, landed by the phase's real-tier plan per D-12/D-13.)

## Next Phase Readiness
- 06-06/06-07 can consume: `Front.AuthMode()` (the runtime-headers source's selector), the `PatEnsurer` seam (wire `*hostsvc.PatMinter`), the AuthConfig seams Creds/Probe/Entitle (cmd wiring per D-12), and the corpus rows' shapes (the cipher/PAT reverse-ask answers the 06-07 source adapter must produce; the current two-field runtime-headers answer widens to accountScope there, updating the corpus row's expect_answer when it does)
- The registry read-back want list for the coding-plan path derives from the built overlay — config `WantModels` stays api-key-only
- Deferred: pre-existing lint failures (see `.planning/phases/06-oauth-coding-plan-path-model-picker/deferred-items.md`)

---
*Phase: 06-oauth-coding-plan-path-model-picker*
*Completed: 2026-11-22*

## Self-Check: PASSED

- All 10 created files exist on disk (keyringprobe.go/_test, familysel.go/_test, 2 scenarios, 2 goldens — plus modified auth.go/auth_test.go/front.go/golden_test.go present)
- All 8 plan commits are ancestors of HEAD: a2f0cc3, 5d89753, b0e352d, 33e11e0, 9418de2, 0339d54, 65ab331, 310a080
- TDD gate sequence: three test(06-05) → feat(06-05) pairs in ancestry order (each GREEN descends from its RED)
- Commits measured from the ledger: dfde615..HEAD = 8 (plan_head_before: dfde61554c48fa62df91206de5a593261203f1d2, plan_head_after: 310a080867489353dc78636ffc1ef28c516e13ac)
