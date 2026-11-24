---
phase: 261124-wlg
verified: 2026-11-24T22:59:19Z
status: passed
score: 9/9 must-haves verified
covered_files:
  - .planning/quick/261124-wlg-implement-live-acp-model-catalog-refresh/261124-wlg-PLAN.md
  - .planning/quick/261124-wlg-implement-live-acp-model-catalog-refresh/261124-wlg-SUMMARY.md
  - docs/protocol/acp-client-dialect.md
  - docs/protocol/app-server-dialect.md
  - internal/acpfront/catalogrefresh.go
  - internal/acpfront/catalogrefresh_internal_test.go
  - internal/acpfront/catalogrefresh_test.go
  - internal/acpfront/configoptions.go
  - internal/acpfront/configoptions_test.go
  - internal/acpfront/front.go
  - internal/acpfront/lifecycle.go
  - internal/acpfront/slashbusy_internal_test.go
  - internal/acpfront/stream.go
  - internal/acpfront/turnctx.go
  - internal/acpport/agent.go
  - internal/acpport/catalogupdate_test.go
  - internal/acpport/sink.go
  - internal/acpport/types.go
covered_digest: "v4:sha256:bac367c977604c0c7fc602c3f7919c602a682daae62ad6757ae0d540c094909c"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 8/9
  gaps_closed:
    - "Internal catalog/picker teardown now shares one absolute drain deadline capped by the caller deadline and close wait; backend close uses the caller context."
    - "TestCatalogCloseLockedHonorsSingleRefreshDeadline proves the 30 ms refresh / 250 ms close-wait case completes local cleanup within 150 ms."
    - "Standalone CloseSession retains independent close-wait behavior; standalone close/resume lifecycle tests and the 2,000-iteration interleave pass."
  gaps_remaining: []
  regressions: []
advisory: []
---

# Quick Task 261124-wlg: Live ACP Model Catalog Refresh Verification

**Task Goal:** Implement live ACP model catalog refresh in zcode-acp: publish full `configOptions` through `session/update` `config_option_update` when the backend provider model catalog changes, with confirmed registry read-back, active-session fan-out, current-model safety, and wire/integration tests. Orca 1.4.271 already supports this notification; do not modify Orca.

**Verified:** 2026-11-24T22:59:19Z
**Status:** passed
**Re-verification:** Yes — after deadline remediation commit `f79dc30`

## Re-verification Summary

The final carried-forward blocker is closed:

- `closeSessionLocked` computes one shared drain deadline using the earlier of the caller deadline and `closeWait`.
- Both internal teardown drains use that same absolute deadline.
- Backend `session/close` uses the caller context.
- Standalone `CloseSession` passes a zero shared deadline and retains its existing independent close-wait behavior.
- The 30 ms refresh / 250 ms close-wait regression completes full local cleanup within 150 ms and passed 10 consecutive `-race` runs.

All must-haves, success criteria, artifacts, links, and repository gates are verified.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|------|--------|----------|
| 1 | Every active ACP session receives the complete confirmed list after a backend catalog change. | ✓ VERIFIED | Normal/no-current fan-out and fresh-reasoning-mismatch model-only publication pass. |
| 2 | A refresh acknowledgement alone never publishes; fresh `session/create` read-back is authoritative. | ✓ VERIFIED | Refresh reply is discarded; integration asserts refresh → create → close → live setModel ordering. |
| 3 | Preserve a still-listed confirmed selection; otherwise use only a listed backend-confirmed fallback. | ✓ VERIFIED | Live setModel confirmation, listed-current acceptance, unlisted-current teardown, and stale-mutation serialization pass. |
| 4 | Malformed, unconfirmed, or unlisted current values produce no fabricated selection or unsafe update. | ✓ VERIFIED | Partial refs are rejected; probe reasoning is never borrowed; fresh-catalog-invalid reasoning is stripped. |
| 5 | A closing or closed session cannot pass the final gate, receive an update, or regain settings. | ✓ VERIFIED | `/model` recheck, close/mutation serialization, transactional local teardown, late-reply rejection, and publication close gate pass. |
| 6 | A failed client write retains the prior snapshot and leaves an identical refresh eligible to retry. | ✓ VERIFIED | Write-failure and failed-fallback retry tests pass; healthy fan-out continues and failed signals clear. |
| 7 | Registry handling is off the read loop, coalesced single-flight, and bounded. | ✓ VERIFIED | Observer/coalescing/stripe/closing-contention tests pass. Internal teardown now shares one caller-capped deadline; 30 ms/250 ms regression passes. |
| 8 | The neutral `acpport` seam carries DTOs only; `acpfront` does not import the ACP SDK. | ✓ VERIFIED | No production SDK import under `internal/acpfront`; `TestSDKImportBoundary` passes. |
| 9 | Exact wire tests and a two-active-session integration test prove framing and fan-out. | ✓ VERIFIED | Exact one- and two-session JSON-RPC assertions pass, including coalescing, read-back, fallback, reasoning, and ordering. |

**Score:** 9/9 truths verified (0 present, behavior-unverified).

### Success Criteria Coverage

| # | Success criterion | Status | Evidence |
|-------------------|--------|----------|
| 1 | One full update per active session after a live provider-model change. | ✓ VERIFIED | Normal, no-current, and fresh-reasoning-mismatch paths publish. |
| 2 | Published catalog/current values come from confirmed backend snapshots, never an acknowledgement or requested value. | ✓ VERIFIED | Model current is live-confirmed; reasoning is validated against fresh availability. |
| 3 | A disappeared selection is replaced only by a listed confirmed fallback; otherwise no unsafe update. | ✓ VERIFIED | Listed confirmed current publishes; unlisted/malformed current closes. |
| 4 | Close safety and failed-write retry eligibility hold. | ✓ VERIFIED | Lifecycle, late-reply, write-retry, and fallback-retry regressions pass. |
| 5 | Background work is single-flight, coalesced, bounded, context-owned, and does not block the read loop. | ✓ VERIFIED | Read-loop, coalescing, stripe cancellation, closing contention, and shared teardown deadline tests pass. |
| 6 | Neutral DTO seam remains intact and SDK version is unchanged. | ✓ VERIFIED | Import-boundary test and `tidy-diff` pass; SDK remains v0.20.0. |
| 7 | Exact wire/integration tests, full race suite, lint, docs, experiments, and tidy-diff pass. | ✓ VERIFIED | Full `mise run ci`, docs check, and `tidy-diff` pass. |

### Deferred Items

None. Polling/watch fallback remains intentionally deferred by the plan.

### Advisory (New Scope, Unevidenced)

None.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/acpport/types.go` | Neutral note kind and payload | ✓ VERIFIED | Defined and used across the seam. |
| `internal/acpport/sink.go` | SDK `config_option_update` translation | ✓ VERIFIED | Exact wire test passes; empty lists render `[]`. |
| `internal/acpport/agent.go` | Async sink binding and post-auth refresh | ✓ VERIFIED | Binding/auth tests pass. |
| `internal/acpfront/catalogrefresh.go` | Refresh/read-back/safety/fan-out | ✓ VERIFIED | Substantive, wired, and behaviorally covered. |
| `internal/acpfront/catalogrefresh_internal_test.go` | Safety/lifecycle/retry/deadline regressions | ✓ VERIFIED | 24 active tests, including the shared-deadline regression. |
| `internal/acpfront/catalogrefresh_test.go` | Two-session integration and exact frames | ✓ VERIFIED | Composed proof passes. |
| `docs/protocol/acp-client-dialect.md` | Client live-update contract | ✓ VERIFIED | Documents fresh-catalog reasoning validation and unreconciled close. |
| `docs/protocol/app-server-dialect.md` | Backend read-back contract | ✓ VERIFIED | Documents no-current probe and no probe-reasoning borrowing. |

**Artifacts:** 8/8 declared artifacts present, substantive, and wired.

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/acpport/agent.go` | `internal/acpfront/front.go` | `UpdateSinkBinder.BindUpdateSink` | ✓ WIRED | Connection sink reaches `Front`. |
| `internal/acpfront/stream.go` | `internal/acpfront/catalogrefresh.go` | Registry hand-off | ✓ WIRED | Observer avoids closing/session locks. |
| `internal/acpfront/catalogrefresh.go` | `internal/hostsvc/snapshot.go` | `RefreshRegistry` | ✓ WIRED | Reply discarded; create read-back follows. |
| `internal/acpfront/catalogrefresh.go` | `internal/acpfront/configoptions.go` | Parse/build confirmed settings | ✓ WIRED | Reasoning validated against fresh availability. |
| `internal/acpfront/catalogrefresh.go` | `internal/acpfront/front.go` / `lifecycle.go` | Session stripe + transactional closing gate | ✓ WIRED | Lifecycle and shared-deadline regressions pass. |
| `internal/acpfront/catalogrefresh.go` | `internal/acpport/sink.go` | `NoteConfigOptions` | ✓ WIRED | Exact wire assertions pass. |
| `internal/acpfront/catalogrefresh_test.go` | `internal/acpfront/catalogrefresh.go` | Composed integration | ✓ WIRED | Real backend-to-ACP path passes. |

**Wiring:** 7/7 declared links verified.

### Data-Flow Trace (Level 4)

| Artifact | Data variable | Source | Produces real data | Status |
|----------|--------------|-------|-------------------|--------|
| `catalogrefresh.go` | Registry trigger | `state.updated` `patch.providers` | Yes | ✓ FLOWING |
| `catalogrefresh.go` | Authoritative model catalog | Fresh `session/create` | Yes | ✓ FLOWING |
| `catalogrefresh.go` | Live fallback current | Live `session/setModel` reply | Yes | ✓ FLOWING |
| `catalogrefresh.go` | Reasoning level | Prior/live snapshot validated against fresh catalog | Yes | ✓ FLOWING |
| `sink.go` | `configOptions` | Neutral DTO list | Yes | ✓ FLOWING |
| SDK connection | `session/update` | `SessionConfigOptionUpdate` | Yes | ✓ FLOWING |

No static/mock catalog is used in production.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Shared-deadline regression stress | `go test -race -count=10 -run '^TestCatalogCloseLockedHonorsSingleRefreshDeadline$' ./internal/acpfront` | 10/10 pass | ✓ PASS |
| Final remediation and prior contracts | `go test -race -count=1 -run 'TestCatalogCloseLockedHonorsSingleRefreshDeadline\|TestCatalogFallbackFreshReasoningMismatchPublishesModelOnly\|TestModelSlashRechecksClosingAfterSessionStripe\|TestFinalCloseStripeTimeoutCompletesLocalTeardown\|TestLateSetModelReplyCannotRecreateSettingsAfterLocalTeardown\|TestCatalogRefreshCancellationBoundsContendedClosingCheck\|TestCatalogPublicationCloseGate\|TestCatalogCloseSerializesInFlightMutation\|TestCatalogResumeSerializesPublication\|TestPickerRefusalTeardownUnderSessionStripe\|TestCatalogSessionLockCancellationDoesNotWait\|TestCatalogFallbackCommitsCurrentWithoutInvalidReasoning\|TestCatalogFallbackClosesOnUnreconciledReply\|TestCatalogFailedFallbackRetriesAfterHealthyFanout\|TestCatalogObserverDoesNotTakeClosingLock\|TestCatalogPublicationSerializesModelMutation\|TestCatalogFallbackUsesLiveSetModelConfirmation\|TestCatalogFallbackRejectsUnconfirmedReply\|TestCatalogNoCurrentProbePreservesListedSelection\|TestCatalogPendingSignalSurvivesFailedPass\|TestCatalogClosingSkipDoesNotAbortHealthyFanout\|TestParseSettingsRejectsPartialModelRefs\|TestSnapshotForLiveCatalogSafety\|TestCatalogPublicationFailureThenIdenticalRefresh\|TestCatalogWorkerFailureThenIdenticalRefresh\|TestLiveCatalogRefresh\|TestSDKImportBoundary\|TestPromptModelSlashBusyScope\|TestResumeRejectedWhileClosing\|TestResumeCloseInterleave' ./internal/acpfront ./internal/acpport` | `ok ... acpfront 11.188s`; `ok ... acpport 1.059s` | ✓ PASS |
| Standalone close/resume behavior | `go test -race -count=1 -run 'TestCloseTeardown\|TestCloseWhileTurnActiveRace\|TestCloseWithPendingPermissionCancelled\|TestCloseDeadlineArmFansPendingIn\|TestResumeRejectedWhileClosing\|TestResumeCloseInterleave' ./internal/acpfront` | Pass | ✓ PASS |
| Complete affected package race suites | `go test -race -count=1 ./internal/acpport ./internal/acpfront` | Both pass | ✓ PASS |
| Full repository gate | `mise run ci` | Exit 0; lint 0 issues; full race suite; 24 experiments pass / 1 guarded skip | ✓ PASS |
| Documentation gate | `bash experiments/check-docs.sh` | Exit 0 | ✓ PASS |
| Dependency drift gate | `mise run tidy-diff` | Exit 0; module unchanged | ✓ PASS |

### Probe Execution

Not applicable. No external probe script is declared. The production create read-back is exercised by integration and verifier tests.

### Requirements Coverage

No requirement IDs are assigned to this quick task; `.planning/REQUIREMENTS.md` has no `261124-wlg` mapping. Requirement-ID coverage is N/A.

### Test Quality Audit

| Test file | Linked truth | Active | Skipped | Circular | Assertion level | Verdict |
|-----------|--------------|--------|---------|----------|-----------------|---------|
| `internal/acpport/catalogupdate_test.go` | Framing/binding/auth | 3 | 0 | No | Exact value + behavioral | Strong |
| `internal/acpfront/catalogrefresh_internal_test.go` | Safety/lifecycle/retry/coalescing/deadline | 24 | 0 | No | Value + deterministic transitions | Strong |
| `internal/acpfront/catalogrefresh_test.go` | Two-session fan-out/order | 1 | 0 | No | Exact wire + multi-step behavioral | Strong |
| `internal/acpfront/configoptions_test.go` | Parser/partial refs | 2 | 0 | No | Value + malformed-input behavioral | Strong |

**Disabled tests:** 0. **Circular patterns:** 0. **Insufficient assertions:** 0.

### Anti-Patterns Found

No debt markers, placeholders, disabled tests, or dependency changes were found.

### Human Verification Required

None. This is an infrastructure/protocol phase; all acceptance criteria are verified programmatically.

### Gaps Summary

No gaps found. Commit `f79dc30` closes the final bounded-worker blocker by sharing one caller-capped teardown deadline across internal drains while preserving standalone close behavior. The phase goal is achieved.

---

_Verified: 2026-11-24T22:59:19Z_
_Verifier: gsd-verifier_
