---
phase: 05-interactions-full-chat-cycle-api-key
plan: 06
subsystem: auth-execution-tier
tags: [auth-01, api-key, snapshot-push, registry-readback, pitfall-1, runtime-headers, hostsvc, fake-corpus, golden]

# Dependency graph
requires:
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor
    provides: zcode.Conn Call/bypass-lane + AddObserver fan-out, supervisor.BuiltinRevision (the V-SRC revision formula), scripted fake + golden corpus machinery
  - phase: 05-interactions-full-chat-cycle-api-key (05-01)
    provides: interact Responder policy object + RuntimeHeadersKeySource seam (WithRuntimeHeadersKeySource), reannounce scripting, front-tier corpus preamble
provides:
  - internal/hostsvc SnapshotBuilder (strict {revision, basedOnZCodeBuiltinRevision, providers:{}, states:{}} envelope; supervisor.BuiltinRevision REUSED; injectable revision source for deterministic replays)
  - hostsvc.PushSnapshot / hostsvc.PushEnvelope (provider/updateAccountConfig Call + wrap-on-error; Ack parsed as receipt evidence only) / hostsvc.RefreshRegistry (provider/refreshRegistry {reason})
  - hostsvc.WatchRegistry / hostsvc.RegistryWatcher (Await poll-with-deadline, OfferCreateReply explicit surface, latched exactly-once verdict + revision echo) / hostsvc.AwaitRegistryContains one-shot — the Phase-6-reused asset
  - hostsvc.RuntimeHeadersHandler(KeySource) → interact.RuntimeHeadersKeySource (the D-02 answer over the 05-01 seam) + hostsvc.StaticKey
  - zcode method constants MethodProviderUpdateAccountConfig / MethodProviderRefreshRegistry
  - corpus rows auth-apikey + runtime-headers-answer (+ goldens): push envelope bytes with the exact formula revision string; the runtime-headers reply vector byte-locked end-to-end through the responder
affects: [05-07 front authenticate (composes these step functions + wires the responder), 05-09 real tier, phase-06 OAuth (reuses the read-back helper)]

# Actuals (#2632)
actuals:
  tokens: 16352    # chars/4 over the realized diff (65410 chars); plan estimated 42000
  tasks: 2
  commits: 4       # MEASURED: git rev-list --count gsd-plan-head-before-05-06..HEAD
commits: 4
plan_head_before: a0a8a7ca53137a1fa3943feac7dbc0c54fafa53a
plan_head_after: 4c423c086bdadaa7073c036d2c88752002c0d422

# Tech tracking
tech-stack:
  added: []        # zero new dependencies (stdlib + in-module only; module graph unchanged)
  patterns:
    - snapshot builder over a synthetic deterministic install path (the formula hashes the PATH STRING — goldens stay machine-independent while the unit tier proves path sensitivity on real files)
    - register-before-traffic read-back watcher (transient observer + remove func; latched exactly-once verdict)
    - map dig-outs for camelCase wire payloads at the source tier (tagliatelle contract)

key-files:
  created:
    - internal/hostsvc/snapshot.go
    - internal/hostsvc/readback.go
    - internal/hostsvc/hostsvc_test.go
    - internal/zcode/fake/testdata/scenarios/auth-apikey.json
    - internal/zcode/fake/testdata/golden/auth-apikey.golden
    - internal/zcode/fake/testdata/scenarios/runtime-headers-answer.json
    - internal/zcode/fake/testdata/golden/runtime-headers-answer.golden
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-06-red-auth-t1.json
    - .planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-06-red-auth-t2.json
  modified:
    - internal/zcode/envelope.go
    - internal/zcode/fake/golden_test.go

key-decisions:
  - "The corpus fixture pushes against a synthetic absolute install path (/zcode-acp-corpus-fixture/...) via BuildRevision(install, 44): BuiltinRevision hashes the resolved PATH STRING — the file need not exist — so the recorded push-envelope golden bytes are deterministic across machines; path sensitivity is proven at the unit tier on real TempDir files, and Build(install) keeps the production probe-time readBuiltinRevision flow"
  - "The corpus driver injects a FIXED revision (account-bridge:corpus-fix1) via WithRevisionSource, which makes the fake's static ack a GENUINE echo — the receivedRevision pairing is byte-locked, not just shape-asserted"
  - "The read-back watcher composes register-first: WatchRegistry returns (watcher, remove) for pre-traffic registration (the refresh rule's side-effect emit precedes its reply on the wire), Await polls with a deadline (watcher deadline ∩ ctx), OfferCreateReply feeds the dialect §8 create-reply surface the notification stream does not carry; AwaitRegistryContains stays the one-shot convenience"
  - "RuntimeHeadersHandler adapts KeySource to the 05-01 seam; the api-key success arm requires a non-empty apiKey (the mode's credential — an empty-string apiKey would fail the wire schema's nonEmptyString and REJECT the pending ask, Pitfall 4), apiKeyId rides whenever supplied — the invalid shape can never be marshaled and the 05-01 responder's locked contracts stay untouched"
  - "surfaceRevision digs the matched surface's revision echo string-or-number (carrier-tolerant, the machine.go carrierKindField posture); the negative arm's error wraps the coded ErrRegistryNotConfirmed sentinel with the actionable check (verify basedOnZCodeBuiltinRevision against the active builtin — the dialect §8 silent-discard shape)"

patterns-established:
  - "hostsvc step functions compose into the front's authenticate (plan 05-07): record key → PushSnapshot → RefreshRegistry → WatchRegistry+Await → only then success"
  - "Deterministic golden fixtures over synthetic absolute paths for path-derived values, with the formula proven at the unit tier"

requirements-completed: [AUTH-01]

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "Snapshot builder emits the strict envelope {revision, basedOnZCodeBuiltinRevision, providers:{}, states:{}} with basedOn EXACTLY supervisor.BuiltinRevision's output for the same resolved path (path-sensitive, runtime-computed, never hard-coded) and the empty personal-layer overlay"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/hostsvc/hostsvc_test.go#TestSnapshotEnvelope"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/auth-apikey (push envelope bytes with the formula revision string)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Registry read-back verdict is registry-driven, never ack-driven: PushSnapshot → RefreshRegistry → await applied=true only when surfaces carry the zai-api models; the ack-mismatch negative (ack says received, registry never confirms) answers applied=false + coded ErrRegistryNotConfirmed; the create-reply settings.model.available surface is accepted via OfferCreateReply"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/hostsvc/hostsvc_test.go#TestAuthReadback (all three arms over a real Conn + scripted peer)"
        status: pass
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/auth-apikey (driver verdicts through the helper)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Runtime-headers handler answers the D-02 success shape {headersApplied:true, requestAuth:{apiKey, apiKeyId}} from the injected key (apiKeyId optional-omit) and the coded -32603 error when unconfigured — the failure-shaped headersApplied:false arm never leaves the process"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/hostsvc/hostsvc_test.go#TestRuntimeHeaders (both arms + strict-object shape)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The runtime-headers reply vector is byte-locked end-to-end through the responder: the corpus row fires the reverse (request params per the research shape, 180 s-scale window configured small, reannounce count 1) and expect_answer locks the exact strict reply over a synthetic StaticKey; an answered ask earns no re-announce envelope"
    requirement: AUTH-01
    verification:
      - kind: integration
        ref: "internal/zcode/fake/golden_test.go#TestGolden/runtime-headers-answer"
        status: pass
    human_judgment: false

# Metrics
duration: 30min
completed: 2026-11-20
status: complete
---

# Phase 5 Plan 06: API-Key Snapshot Push, Registry Read-Back & Runtime-Headers Answer Summary

**internal/hostsvc delivers AUTH-01's execution tier — the strict snapshot push reusing supervisor.BuiltinRevision verbatim, the PITFALL-1 read-back helper whose verdict is the registry's never the ack's (negative test-locked), and the D-02 runtime-headers answer over the 05-01 KeySource seam — locked at the fake tier by two corpus rows whose goldens byte-lock the push envelope and the reply vector.**

## Performance

- **Duration:** 30 min
- **Started:** 2026-11-20T13:50:52Z
- **Completed:** 2026-11-20T14:20:54Z
- **Tasks:** 2 (both TDD: RED → GREEN)
- **Files modified:** 11

## Accomplishments

- `internal/hostsvc` SnapshotBuilder: the strict envelope {revision, basedOnZCodeBuiltinRevision, providers:{}, states:{}} — revision `account-bridge:<unix-nano>` (reference-adapter precedent, injectable for deterministic replays), basedOn REUSING `supervisor.BuiltinRevision` over the resolved path (never re-derived, never hard-coded), the empty overlay per the field-verified personal-layer no-op
- PushSnapshot/PushEnvelope (the Call + wrap-on-error pattern) and RefreshRegistry (`{reason:"account-config-sync"}`); the Ack parsed as receipt evidence only — the schema's own 收到≠已到达 warning enforced structurally
- The read-back helper (PITFALL 1): WatchRegistry (transient observer, register-before-traffic) + Await (poll-with-deadline, injectable) + OfferCreateReply (the dialect §8 create-reply surface) + the one-shot AwaitRegistryContains; verdicts latch exactly once with the surface's revision echo; the negative arm proves an ack-only "received" never reports applied (coded ErrRegistryNotConfirmed with the actionable basedOn check)
- RuntimeHeadersHandler over the 05-01 seam: strict D-02 success shape from the injected source (in-memory only), coded error otherwise — `headersApplied:false` never leaves the process; plus StaticKey for config-sourced material
- Two corpus rows + goldens: auth-apikey locks the push envelope bytes with the exact formula revision string and drives the full push → refresh → read-back sequence through the helper; runtime-headers-answer byte-locks the reply vector end-to-end through the responder with synthetic fixture key material
- `mise run ci` green (build → vet → lint 0 issues → test -race → test-experiments); full `go test -race ./internal/...` green (9 packages); scan-secrets green; both RED phases machine-validated (RED_EVIDENCE_OK)

## Task Commits

1. **Task 1 RED: snapshot-envelope + runtime-headers contract tests** - `fc31773` (test)
2. **Task 1 GREEN: builder, refresh call, runtime-headers handler + auth-apikey corpus row** - `755faf0` (feat)
3. **Task 2 RED: read-back helper tests over a scripted peer conn** - `d166797` (test)
4. **Task 2 GREEN: awaitRegistryContains + runtime-headers-answer corpus row** - `4c423c0` (feat)

**Plan metadata:** (this SUMMARY via the gsd-tools commit verb)

_Note: both tasks are tdd="true" — each test(05-06) RED commit precedes its feat(05-06) GREEN commit; RED evidence records live at `.planning/phases/05-interactions-full-chat-cycle-api-key/evidence/05-06-red-auth-t{1,2}.json` (verdicts RED_EVIDENCE_OK; semanticAssessment recorded inline). No REFACTOR commit — GREEN landed clean under the strict linter._

## TDD Gate Compliance

- **RED 1:** `go test -race -run 'TestSnapshotEnvelope|TestRuntimeHeaders' ./internal/hostsvc/` — TestSnapshotEnvelope failed on the revision-generation clause (skeleton builds nothing); both configured TestRuntimeHeaders arms failed receiving the coded unconfigured error in place of the strict success shape. Machine check: **RED_EVIDENCE_OK** (target TestSnapshotEnvelope matched).
- **GREEN 1:** all locks pass under `-race`; auth-apikey golden recorded and deterministic across `-count=3`.
- **RED 2:** `go test -race -run 'TestAuthReadback' ./internal/hostsvc/` — happy and create-reply arms failed via the stub Await (always ErrRegistryNotConfirmed); the negative arm passed as the preserved discipline posture (recorded in the evidence's semanticAssessment). Machine check: **RED_EVIDENCE_OK** (target TestAuthReadback matched).
- **GREEN 2:** all three arms pass; runtime-headers-answer golden recorded; `mise run ci` green.
- Gate validation: `test(05-06)` commits fc31773+d166797 precede `feat(05-06)` commits 755faf0+4c423c0; no REFACTOR needed.

## Files Created/Modified

- `internal/hostsvc/snapshot.go` (NEW) — KeySource/StaticKey, SnapshotBuilder (Build probe-time read + BuildRevision explicit-rev), PushSnapshot/PushEnvelope/parseAck (map dig-out), RefreshRegistry, RuntimeHeadersHandler
- `internal/hostsvc/readback.go` (NEW) — WatchRegistry/RegistryWatcher (Await, OfferCreateReply, latched verdict, ErrRegistryNotConfirmed), AwaitRegistryContains, the surface dig-outs + surfaceRevision
- `internal/hostsvc/hostsvc_test.go` (NEW) — TestSnapshotEnvelope, TestRuntimeHeaders (both arms + strict shape), TestAuthReadback (three PITFALL-1 arms over the scripted peer)
- `internal/zcode/envelope.go` — MethodProviderUpdateAccountConfig / MethodProviderRefreshRegistry constants (the dialect §2 provider/* strings)
- `internal/zcode/fake/golden_test.go` — driveAuthApikey (push → refresh → helper verdicts, watcher registered before traffic), driveRuntimeHeadersAnswer (responder + synthetic StaticKey), corpus fixtures
- `internal/zcode/fake/testdata/scenarios/auth-apikey.json` + `testdata/golden/auth-apikey.golden` — the push/read-back corpus row
- `internal/zcode/fake/testdata/scenarios/runtime-headers-answer.json` + `testdata/golden/runtime-headers-answer.golden` — the reply-vector corpus row
- `evidence/05-06-red-auth-t{1,2}.json` — machine-validated RED evidence

## Decisions Made

- **Deterministic corpus fixture over a synthetic install path.** The recorded golden carries `zcode-builtin:44:<sha256("/zcode-acp-corpus-fixture/zcode-builtin.json")>` — computed at test runtime by supervisor.BuiltinRevision over the fixed PATH STRING (the file need not exist; EvalSymlinks falls back to the input). This keeps the golden machine-independent while the formula's path sensitivity is proven at the unit tier on real TempDir files. The corpus install is synthetic exactly like `sess_fix1`.
- **Injected fixed revision = genuine ack echo.** The plan hedged that "the fake's rule reply is static, so the driver's assertion pairs the pushed revision with the echo" — with WithRevisionSource the corpus revision is fixed, so the static ack echoes it verbatim and the pairing lock is real bytes, not shape.
- **Register-first watcher composition.** The fake runs side effects BEFORE the rule's reply, so a watcher registered after RefreshRegistry returns would miss the emit; WatchRegistry therefore returns (watcher, remove) for pre-traffic registration, and AwaitRegistryContains remains the one-shot for surfaces arriving strictly later (documented on the function).
- **The api-key success arm requires a non-empty apiKey.** The wire schema's `apiKey: nonEmptyString.optional()` means an empty-string key would parse-fail and REJECT the pending server-side ask (Pitfall 4). The hostsvc handler treats empty-key as the unconfigured arm; apiKeyId rides whenever supplied. This containment keeps the 05-01 responder's locked contracts untouched (responder.go not modified).
- **Carrier-tolerant revision echo.** state.updated patches carry a numeric registry revision; create replies carry none. surfaceRevision digs string-or-number (string form), and Readback.Revision reports what the matched surface actually echoed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] provider/* method constants absent from the codec**
- **Found during:** Task 1 (GREEN implementation)
- **Issue:** `internal/zcode/envelope.go` had no constants for provider/updateAccountConfig or provider/refreshRegistry — the plan's Calls cannot be made through the house method-constant surface (the plan's files_modified did not list envelope.go)
- **Fix:** Added MethodProviderUpdateAccountConfig and MethodProviderRefreshRegistry to the dialect-§2 method-constants block (verbatim wire strings)
- **Files modified:** internal/zcode/envelope.go
- **Verification:** the auth-apikey corpus golden records the exact method strings on the wire
- **Committed in:** 755faf0

**Total deviations:** 1 auto-fixed (1 × Rule 3).
**Impact on plan:** additive constants only; no scope change — the task/verify surface is exactly as planned.

## Issues Encountered

- The strict linter required the usual iterations (funlen/gocognit splits, tagliatelle camelCase wire keys resolved by map dig-outs at the source tier — the repo's established contract), ending at 0 issues.
- Golden determinism for the path-derived formula value was solved by the synthetic-path + explicit-revision design above rather than transcript normalization.

## User Setup Required

None - no external service configuration required (fixtures carry synthetic placeholder material only; the scan-secrets gate stayed green on every commit).

## Next Phase Readiness

Ready for 05-07. The front's authenticate composes these step functions (record key → PushSnapshot → RefreshRegistry → WatchRegistry+Await → `{}` success per PITFALL 3) and wires `hostsvc.RuntimeHeadersHandler(hostsvc.NewStaticKey(...))` into `interact.WithRuntimeHeadersKeySource` at the main.go composition; the cancelled-ctx probe (AUTH-01 concurrency) composes the exported step functions. Phase 6 reuses WatchRegistry/AwaitRegistryContains as its read-back asset. AUTH-01's requirements.mark-complete is deferred until sibling plan 05-07 lands its SUMMARY (ready-ids returned empty — correct per the sibling rule).

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-20*
