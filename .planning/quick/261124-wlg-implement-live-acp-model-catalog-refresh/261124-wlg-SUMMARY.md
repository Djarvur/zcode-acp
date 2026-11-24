---
phase: 261124-wlg
plan: 01
subsystem: api
tags: [acp, model-catalog, config-options, live-refresh, tdd, go]
requires: []
provides:
  - Neutral config-option update notes and connection-bound asynchronous sink binding
  - Bounded registry-event refresh with mandatory create read-back
  - Closing-safe active-session catalog fan-out and failed-write retry eligibility
  - Exact wire, safety, teardown, retry, and two-session integration contracts
affects: [acp-client-dialect, app-server-dialect, acpfront, acpport]
actuals:
  tokens: 28065
  tasks: 3
  commits: 10
  plan_head_before: b5440a4ccfec0f26f14d120dc198aebe620c0621
  plan_head_after: f79dc30f804472b6b641ba402d5511ece6dc13cf
tech-stack:
  added: []
  patterns:
    - Construction-time asynchronous sink binding through neutral acpport interfaces
    - Event-driven single-flight worker with one bounded context-owned lifetime
    - Confirmed create read-back as catalog authority; acknowledgements are receipt only
    - Session-stripe publication with atomic closing/registry-aware settings commits
    - Context-aware bounded session stripes serialize all settings lifecycle writers
key-files:
  created:
    - internal/acpport/catalogupdate_test.go
    - internal/acpfront/catalogrefresh.go
    - internal/acpfront/catalogrefresh_internal_test.go
    - internal/acpfront/catalogrefresh_test.go
  modified:
    - internal/acpport/types.go
    - internal/acpport/sink.go
    - internal/acpport/agent.go
    - internal/acpfront/front.go
    - internal/acpfront/lifecycle.go
    - internal/acpfront/stream.go
    - internal/acpfront/turnctx.go
    - internal/acpfront/configoptions.go
    - internal/acpfront/configoptions_test.go
    - internal/acpfront/slashbusy_internal_test.go
    - docs/protocol/acp-client-dialect.md
    - docs/protocol/app-server-dialect.md
key-decisions:
  - Fresh session/create settings.model.available is the only published catalog authority.
  - Preserve a listed prior selection; otherwise apply a listed candidate with live session/setModel and require that reply's confirmation.
  - Publication holds the session mutation stripe through bounded client I/O, then commits settings atomically only if the session is still registered and not closing; closingMu is never held across I/O.
  - Final close timeout performs transactional local registry/settings/marker cleanup, and registry-aware commits prevent late replies from recreating state.
  - Standalone close marks under the stripe, releases it during drain so competing resume/mutation sees closing, then reacquires it for the final registry/settings drop.
  - Invalid or absent live fallback reasoning commits the confirmed current but omits thought-level; probe reasoning is never adopted.
  - A failed publication fails the refresh pass and clears deduplication so identical signals retry; a different pending signal survives and becomes the next pass identity.
patterns-established:
  - Neutral DTO seam: SDK config-option types remain confined to internal/acpport.
  - Session snapshots are confirmed state, never requested or acknowledgement-derived state.
requirements-completed: []
coverage:
  - id: D1
    description: Exact config_option_update framing, async binding, and post-auth refresh
    verification:
      - kind: unit
        ref: internal/acpport/catalogupdate_test.go#TestConfigOptionUpdateWire/TestAsyncSinkBinding/TestAuthenticateRequestsCatalogRefresh
        status: pass
    human_judgment: false
  - id: D2
    description: Registry classification, safe selection merge, close gate, and retry eligibility
    verification:
      - kind: unit
        ref: internal/acpfront/catalogrefresh_internal_test.go#TestRegistryCatalogPatchClassification/TestSnapshotForLiveCatalogSafety/TestCatalogPublicationCloseGate/TestCatalogPublicationFailureThenIdenticalRefresh/TestCatalogWorkerFailureThenIdenticalRefresh
        status: pass
    human_judgment: false
  - id: D3
    description: One backend catalog change reaches two active sessions with exact wire frames
    verification:
      - kind: integration
        ref: internal/acpfront/catalogrefresh_test.go#TestLiveCatalogRefresh
        status: pass
    human_judgment: false
  - id: D5
    description: Settings lifecycle serialization, fallback reasoning omission, retry eligibility, and bounded stripe waits
    verification:
      - kind: unit
        ref: internal/acpfront/catalogrefresh_internal_test.go#TestCatalogCloseSerializesInFlightMutation/TestCatalogResumeSerializesPublication/TestPickerRefusalTeardownUnderSessionStripe/TestCatalogSessionLockCancellationDoesNotWait/TestCatalogFallbackCommitsCurrentWithoutInvalidReasoning/TestCatalogFallbackClosesOnUnreconciledReply/TestCatalogFailedFallbackRetriesAfterHealthyFanout
        status: pass
    human_judgment: false
  - id: D6
    description: Confirmed-current preservation under fresh reasoning mismatch and transactional teardown/cancellation safety
    verification:
      - kind: unit
        ref: internal/acpfront/catalogrefresh_internal_test.go#TestCatalogFallbackFreshReasoningMismatchPublishesModelOnly/TestModelSlashRechecksClosingAfterSessionStripe/TestFinalCloseStripeTimeoutCompletesLocalTeardown/TestLateSetModelReplyCannotRecreateSettingsAfterLocalTeardown/TestCatalogRefreshCancellationBoundsContendedClosingCheck
        status: pass
    human_judgment: false
  - id: D7
    description: Catalog-triggered teardown honors one shared refresh/close deadline
    verification:
      - kind: unit
        ref: internal/acpfront/catalogrefresh_internal_test.go#TestCatalogCloseLockedHonorsSingleRefreshDeadline
        status: pass
    human_judgment: false
  - id: D4
    description: Protocol documents and complete repository quality gate
    verification:
      - kind: other
        ref: bash experiments/check-docs.sh && mise run ci && mise run tidy-diff
        status: pass
    human_judgment: false
duration: 124min
completed: 2026-11-25
status: complete
---

# Quick Task 261124-wlg: Live ACP Model Catalog Refresh Summary

**Backend registry changes now publish complete, confirmed config options to every active ACP session through a bounded read-back and closing-safe fan-out path.**

## Performance

- **Duration:** 124 minutes (23 minutes initial execution + 33 minutes first review + 26 minutes second review + 25 minutes final review + 17 minutes deadline review)
- **Started:** 2026-11-24T20:49:01Z
- **Completed:** 2026-11-24T22:52:55Z
- **Tasks:** 3/3 completed
- **Files changed:** 16 (including lifecycle and busy-scope fixture hardening)
- **Commits:** 10, measured from the recorded feature-branch base

## Accomplishments

- Added neutral `NoteConfigOptions`/`ConfigOptions` transport and SDK translation with exact `config_option_update` wire framing, including `[]` rather than `null` for an empty list.
- Bound the connection sink into `Front` before traffic and requested an advisory catalog refresh only after successful authentication.
- Classified provider-registry patches off the read loop; deduplicated identical signals, coalesced pending work, and used one 15-second worker context with no polling fallback.
- Required `provider/refreshRegistry` followed by a fresh bare `session/create` read-back and best-effort probe close; discarded the acknowledgement as catalog evidence.
- Preserved still-listed selections, applied disappeared selections through live `session/setModel`, required the live reply to confirm fallback and reasoning, rejected malformed/unlisted current values, skipped unchanged lists, and fanned out deterministically to active sessions.
- Serialized publication with session close under `closingMu` and with model/config mutation under bounded per-session lock stripes; held no settings lock across client I/O and retained retry eligibility after failed writes.
- Kept closing checks off the synchronous observer path, preserved pending signals across failed passes, deduplicated remembered signals while idle, and continued healthy fan-out past closing-session skips.
- Added exact one- and two-session wire tests plus deterministic close-during-write and worker-level identical-refresh retry tests.
- Documented the client notification contract and backend refresh/read-back recipe in both protocol dialects.

## TDD Evidence and Gate Compliance

### Task 1 RED and GREEN

- RED record: `.planning/tdd/261124-wlg-task1-red.json`
- Raw output: `.planning/tdd/261124-wlg-task1-red.raw.txt`
- Command: `go test -race -count=1 -run 'TestConfigOptionUpdateWire|TestAsyncSinkBinding|TestAuthenticateRequestsCatalogRefresh' ./internal/acpport`
- Semantic assessment: **CONFIRMED RED**. All three targets compiled and failed on the intended assertions: unknown `config_options` note, missing asynchronous binding, and zero post-auth refresh requests. The primary target failed on the unknown-note guard, not compilation or fixture setup.
- Classifier: `RED_EVIDENCE_OK` (`target_test_failed`), produced by the installed `gsd-tools.cjs` fallback. No classifier verdict was fabricated.
- GREEN: focused tests, the complete `internal/acpport` race suite, SDK import-boundary coverage, and strict package lint passed.

### Task 2 RED and GREEN

- RED record: `.planning/tdd/261124-wlg-task2-red.json`
- Raw output: `.planning/tdd/261124-wlg-task2-red.raw.txt`
- Command: `go test -race -count=1 -run 'TestRegistryCatalogPatchClassification|TestSnapshotForLiveCatalogSafety|TestCatalogPublicationCloseGate|TestCatalogPublicationFailureThenIdenticalRefresh|TestLiveCatalogRefresh' ./internal/acpfront`
- Semantic assessment: **CONFIRMED RED**. Every named target executed and failed on inert false/no-publication behavior. The primary target rejected a valid `patch.providers` payload; the integration produced no catalog updates. A pre-evidence fixture sequencing issue was corrected by adding `turn.started` before `turn.completed`, and the final evidence was recaptured.
- Classifier: `RED_EVIDENCE_OK` (`target_test_failed`), produced by the installed `gsd-tools.cjs` fallback.
- GREEN: focused tests, complete `internal/acpfront` and `internal/acpport` race suites, strict lint, exact wire assertions, and the additional worker retry regression passed.

### Gate Sequence

| Task | RED commit | GREEN commit | Result |
|------|------------|--------------|--------|
| 1 | `640bf1c` | `4be2e43` | Pass |
| 2 | `34ebd96` | `4b60a3b` | Pass |

Git ancestry was verified: each GREEN commit descends from its corresponding RED commit. No refactor commit was needed.

## Task Commits

1. **Task 1 RED:** `640bf1c` — `test(261124-wlg): add failing live config-option update contracts`
2. **Task 1 GREEN:** `4be2e43` — `feat(261124-wlg): publish config option updates through the bound sink`
3. **Task 2 RED:** `34ebd96` — `test(261124-wlg): add failing confirmed catalog refresh tests`
4. **Task 2 GREEN:** `4b60a3b` — `feat(261124-wlg): refresh and fan out confirmed model catalogs`
5. **Task 3:** `32f119b` — `docs(261124-wlg): document confirmed live model catalog updates`
6. **Final-review correction:** `fcdad69` — `fix(261124-wlg): retain retry eligibility after catalog write failure`
7. **Independent-review remediation:** `a6fa065` — `fix(261124-wlg): harden live catalog refresh races and confirmation`
8. **Second-review remediation:** `68f8be6` — `fix(261124-wlg): serialize catalog session lifecycles and bound waits`
9. **Final-review remediation:** `92c0d49` — `fix(261124-wlg): preserve confirmed catalogs and make teardown transactional`
10. **Deadline remediation:** `f79dc30` — `fix(261124-wlg): bound catalog teardown with one refresh deadline`

Workflow artifacts, including this summary and TDD evidence, remain uncommitted for the orchestrator.

## Files Created/Modified

- `internal/acpport/types.go` — neutral config-options note vocabulary and confirmed-snapshot documentation.
- `internal/acpport/sink.go` — SDK translation to `SessionConfigOptionUpdate`.
- `internal/acpport/agent.go` — `UpdateSinkBinder`, `CatalogRefresher`, construction binding, and post-auth request.
- `internal/acpport/catalogupdate_test.go` — exact wire, binding, and authentication contracts.
- `internal/acpfront/catalogrefresh.go` — classification, bounded worker, probe read-back, safe merge, fan-out, and publication gate.
- `internal/acpfront/front.go` — atomic sink and single-flight/coalescing state.
- `internal/acpfront/lifecycle.go` — context-aware resume/close lifecycle serialization and closing-window rejection.
- `internal/acpfront/stream.go` — nonblocking registry classification and refresh hand-off.
- `internal/acpfront/turnctx.go` — explicit config-option note accounting; catalog updates bypass the turn pump.
- `internal/acpfront/catalogrefresh_internal_test.go` — safety, close, write-retry, and worker-retry tests.
- `internal/acpfront/catalogrefresh_test.go` — composed backend-to-two-session exact-wire integration proof.
- `internal/acpfront/configoptions.go` — rejects partial provider/model refs before rendering.
- `internal/acpfront/configoptions_test.go` — locks partial-ref rejection for current and available entries.
- `internal/acpfront/slashbusy_internal_test.go` — registers the busy-scope fixture session for lifecycle-aware settings commits.
- `docs/protocol/acp-client-dialect.md` — client notification and selection-safety contract.
- `docs/protocol/app-server-dialect.md` — refresh/create read-back and bounded-worker recipe.

## Decisions Made

- Used the fresh raw create reply as the sole catalog authority; `refreshRegistry` success remains transport evidence only.
- Kept the neutral DTO seam intact: no ACP SDK import was added to `internal/acpfront` production code.
- Used one worker-owned 15-second context and a 5-second publication bound; no context is stored in `Front`.
- Kept the lock order one-way: session mutation stripe, then `closingMu`, then registry/settings access; the synchronous observer takes neither `closingMu` nor a session stripe.
- Treated any failed session publication as a failed refresh pass so identical later registry signals remain retryable.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Added the new note kind to the turn-pump size switch**
- **Found during:** Task 2 GREEN verification
- **Issue:** Strict `exhaustive` lint rejected the existing `noteBytes` switch because `NoteConfigOptions` was unhandled. The plan's implementation allowlist omitted `internal/acpfront/turnctx.go`.
- **Fix:** Added an explicit `NoteConfigOptions` arm returning the fixed frame allowance, documenting that session-level catalog updates bypass the turn pump.
- **Files modified:** `internal/acpfront/turnctx.go`
- **Verification:** strict lint, focused tests, package race suites, and final full CI all pass.
- **Committed in:** `4b60a3b`
- **Scope impact:** This is the sole implementation file outside the plan's 11-file allowlist; it was required for the mandated lint gate and does not alter turn-pump behavior.

**2. [Rule 1 - Bug] Failed fan-out no longer suppresses an identical later retry**
- **Found during:** Task 3 final history review
- **Issue:** `publishFreshCatalog` initially ignored a false publication result. The worker therefore recorded a successful pass and retained the signal digest, causing an identical later event to be deduplicated despite the failed client write.
- **Fix:** Propagated publication failure out of the pass so the worker clears its remembered signal; added a worker-level failed-then-identical-refresh regression test.
- **Files modified:** `internal/acpfront/catalogrefresh.go`, `internal/acpfront/catalogrefresh_internal_test.go`
- **Verification:** the new regression, focused contracts, both race suites, strict lint, and final full CI pass.
- **Committed in:** `fcdad69`

**3. [Rule 3 - Blocking] Added an inert sink-binding seam to the Task 2 RED skeleton**
- **Found during:** Task 2 RED authoring
- **Issue:** The publication tests compile against `Front.BindUpdateSink`, but the prescribed three inert signatures did not include that method.
- **Fix:** Added a no-op `BindUpdateSink` method to the RED skeleton; GREEN replaced it with the atomic store.
- **Files modified:** `internal/acpfront/catalogrefresh.go`
- **Verification:** the final RED package compiled and failed only on behavior assertions; classifier returned `RED_EVIDENCE_OK`.
- **Committed in:** `34ebd96` (RED), `4b60a3b` (GREEN)

---

**Total deviations:** 3 (1 correctness bug, 1 missing critical lint integration, 1 compile-first seam)
**Impact on plan:** All planned behavior was delivered. The only allowlist exception is the two-line exhaustive-switch arm in `turnctx.go`; the retry correction strengthens an explicit must-have. No dependencies, Orca files, go.mod, or go.sum changed.

## Independent-Review Remediation

All seven blocking review findings were fixed in `a6fa065` with deterministic regressions:

1. **Observer lock inversion/nonblocking read loop** — synchronous registry observation now only hashes/classifies and hands off under the refresh mutex; active/closing checks run in the worker. `TestCatalogObserverDoesNotTakeClosingLock` holds `closingMu` and proves the observer returns.
2. **Stale selection overwrite** — bounded striped per-session locks serialize `SetSessionConfigOption`, `/model`, configured model binding, and catalog publication. The worker recomputes under the stripe; `TestCatalogPublicationSerializesModelMutation` blocks publication in client I/O, starts a mutation, then proves the newer confirmed selection remains recorded.
3. **False fallback confirmation** — disappeared currents, including a still-listed lastUsed, require live `session/setModel`; only that reply's listed current confirms publication, with incompatible reasoning omitted. `TestCatalogFallbackUsesLiveSetModelConfirmation`, `TestCatalogFallbackClosesOnUnreconciledReply`, the safety-table lastUsed arm, and backend transcript order assertions lock this.
4. **Reasoning corruption** — unchanged models retain a still-valid prior level and never adopt the probe default; changed models use only the live reply's level. The composed two-session test proves high survives a probe-low for both unchanged and changed sessions.
5. **Malformed partial refs** — both providerId and modelId are mandatory. `TestParseSettingsRejectsPartialModelRefs` covers partial current and available entries and prevents `/GLM-5.3` rendering.
6. **No-current probe** — probe validation now requires only a valid nonempty catalog. `TestCatalogNoCurrentProbePreservesListedSelection` proves a listed prior current still publishes.
7. **Coalescing/retry semantics** — a different pending signal survives a failed pass, becomes current before the follow-up, and is deduplicated while idle. Closing skips do not abort healthy fan-out; actual write failures remain retryable. `TestCatalogPendingSignalSurvivesFailedPass` and `TestCatalogClosingSkipDoesNotAbortHealthyFanout` lock both behaviors.

## Second-Review Remediation

All five remaining blockers were fixed in `68f8be6` with deterministic regressions:

1. **Close/resume lifecycle bypass** — new-session setup, resume, config/model mutation, publication, and final close teardown share context-aware session stripes. Standalone close marks closing under the stripe, releases it during drain so resume can reject the closing window, then reacquires it for the final registry/settings drop. Picker-refusal teardown uses the internal stripe-held close path. `TestCatalogCloseSerializesInFlightMutation`, `TestCatalogResumeSerializesPublication`, and `TestPickerRefusalTeardownUnderSessionStripe` lock the races.
2. **Successful fallback with invalid reasoning** — the confirmed current is committed and the thought-level option is omitted when live reasoning is invalid or absent. An unlisted/unparseable confirmed current closes the session rather than retaining known-stale state. `TestCatalogFallbackCommitsCurrentWithoutInvalidReasoning` and `TestCatalogFallbackClosesOnUnreconciledReply` cover both arms.
3. **Fallback retry eligibility** — transient live `setModel` failure produces a retryable pass result after healthy sessions continue. The completed signal is cleared, so an identical later event retries. `TestCatalogFailedFallbackRetriesAfterHealthyFanout` proves healthy publication, signal clearing, and a second successful refresh/setModel.
4. **Probe reasoning adoption** — snapshot construction now edits the fresh catalog directly and deletes absent/invalid thought-level state; it no longer calls a merge helper that can fall back to probe reasoning. The absent/invalid-live-reasoning subtests prove probe-low never replaces session state.
5. **Unbounded stripe wait** — mutex stripes became bounded channel semaphores with context-aware acquisition. `TestCatalogSessionLockCancellationDoesNotWait` holds a stripe, cancels a competing context, and proves the wait returns immediately with `context.Canceled`.

## Final-Review Remediation

The last two blocker classes were fixed in `92c0d49` with deterministic regressions:

1. **Fresh-catalog reasoning mismatch** — snapshot construction now validates reasoning against the fresh catalog for every selection source. A live-confirmed B/high current remains published against a fresh B/low-only catalog, with thought-level stripped. Any listed confirmed current is accepted; an unlisted confirmed current closes the session. `TestCatalogFallbackFreshReasoningMismatchPublishesModelOnly` and `TestCatalogFallbackClosesOnUnreconciledReply` lock the exact arms.
2. **`/model` close recheck** — the intercept rechecks closing immediately after acquiring the session stripe and before validation/backend mutation. `TestModelSlashRechecksClosingAfterSessionStripe` proves invalid-request rejection, zero sink writes, and zero backend setModel calls.
3. **Transactional teardown** — final-stripe timeout now performs local registry/settings/closing-marker cleanup under the closing lock. Settings commits atomically require a registered, non-closing session, and mutation replies recheck lifecycle state before commit. `TestFinalCloseStripeTimeoutCompletesLocalTeardown` proves no half-closed state; `TestLateSetModelReplyCannotRecreateSettingsAfterLocalTeardown` proves no resurrection.
4. **Bounded closing contention** — catalog publication no longer holds `closingMu` across client I/O; the session stripe excludes normal close marking. Fan-out uses a context-aware closing check. `TestCatalogPublicationCloseGate` proves `closingMu` is available during I/O while close waits on the stripe, and `TestCatalogRefreshCancellationBoundsContendedClosingCheck` proves deadline-bounded cancellation.

The existing busy-scope fixture now registers its session before recording settings, matching the lifecycle-aware commit contract; its same-session and cross-session locks remain green.

## Deadline Remediation

The final verifier blocker was fixed in `f79dc30`: fallback-mismatch and other stripe-held internal teardown now compute one shared drain deadline, capped by both the caller refresh deadline and the configured close wait. Both active-turn drains use that same absolute deadline, and the backend close uses the same caller context. Standalone `CloseSession` still passes no shared deadline and retains its existing independent close-wait behavior.

`TestCatalogCloseLockedHonorsSingleRefreshDeadline` holds an active turn open, configures a 250ms close wait, and invokes the internal catalog close with a 30ms refresh context. It proves teardown finishes within 150ms (rather than two independent ~250ms drains), removes registry/settings state, and clears the closing marker. Existing standalone close, closing-window resume, and 2,000-iteration resume/close tests remain green.

## Verification Results

- Task 1 focused race tests: pass.
- Task 1 complete `internal/acpport` race suite: pass.
- Task 2 focused race tests: pass.
- Task 2 complete `internal/acpfront` and `internal/acpport` race suites: pass.
- `golangci-lint run ./internal/acpfront/... ./internal/acpport/...`: **0 issues**.
- `bash experiments/check-docs.sh`: pass.
- `mise run ci`: pass (build, vet, strict lint, full race suite, experiments; 24 experiment tests passed and 1 guarded test skipped).
- `mise run tidy-diff`: pass; `go.mod` and `go.sum` unchanged; SDK remains pinned at v0.20.0.
- Branch-base ancestry from `b5440a4ccfec0f26f14d120dc198aebe620c0621`: verified.
- Committed and uncommitted `git diff --check`: pass.
- SDK import boundary: pass; SDK imports remain outside `internal/acpfront` production code.
- Scope review: all ten commits and both committed/uncommitted lists inspected. Implementation files match the plan allowlist plus the disclosed `turnctx.go`, `configoptions.go`, `configoptions_test.go`, `lifecycle.go`, and `slashbusy_internal_test.go` review hardening; only workflow artifacts remain uncommitted.

## Known Stubs

None. No placeholder data source, TODO/FIXME implementation stub, skipped test, or unrun required verification remains. The single skipped experiment is the repository's existing guarded live stripped-PATH arm, not a test added by this task.

## Threat Flags

None beyond the plan's registered boundaries. The implementation adds no new endpoint, dependency, authentication path, or schema outside the documented catalog-update flow.

## Issues Encountered

- The first Task 2 RED attempt exposed a test-local variable declaration-order compile error; it was corrected before evidence capture and was not used as RED evidence.
- The integration fixture initially lacked a start announcement before its terminal event; `turn.started` was added and the complete RED output recaptured.
- Strict lint required fixture formatting, static test errors, checked fixture marshaling, and explicit complexity annotations for the composed proof; all were resolved locally.
- The first lifecycle run exposed `TestResumeRejectedWhileClosing`: holding the stripe through close drain made resume wait and then hit a removed backend session. Close now marks under the stripe, releases during drain for closing-aware rejection, and reacquires for the final drop; the existing test and the 2,000-iteration resume/close interleave both pass.
- The first B/high-vs-fresh-B/low fixture still listed high because JSON map keys are sorted; the fixture was corrected to a true low-only reasoning list and the exact model-only wire assertion now passes.
- Lifecycle-aware settings commits exposed an unregistered busy-scope test fixture; it now registers the session and preserves both original busy-scope assertions.

## User Setup Required

None.

## Next Phase Readiness

The live catalog path is complete and fully gated. The orchestrator can commit the workflow artifacts and update state. Polling/watch-based catalog discovery remains explicitly deferred, as planned; the shipped triggers are registry events and the post-authenticate refresh request.

## Self-Check: PASSED

- Summary file exists at the required quick-task path.
- All ten commit hashes are ancestors of HEAD.
- All sixteen implementation/documentation files exist.
- Stub-pattern scan found no TODO, FIXME, placeholder, coming-soon, or unavailable implementation in the new refresh files.
- Only workflow artifacts remain uncommitted: this summary, the quick-task plan directory, TDD evidence, and the deviation entries appended to `.planning/WINDOWS.md`.
