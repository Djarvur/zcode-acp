---
phase: 05-interactions-full-chat-cycle-api-key
plan: 11
subsystem: auth
tags: [acp, zcode, authenticate, registry-readback, jsonrpc, realdesktop-diagnostic, zero-spend]

# Dependency graph
requires:
  - phase: 05 (05-06/05-07)
    provides: the hostsvc snapshot/push/refresh/readback chain and the Front.Authenticate composition (the machinery this plan corrected)
  - phase: 05 (05-09)
    provides: the guarded realdesktop tier helpers (discovery, supervised spawn, tee capture, staged shutdown) the diagnostic reuses
  - phase: 05 (05-10)
    provides: same-wave sawTurn gap closure (no file overlap; createSessionID consumed read-only)
provides:
  - Field-verified read-back verdict for real desktop 3.18.2 (EXP-READBACK-DIAG-01, zero spend): branch (c) — the create reply IS the registry read-back surface; no state.updated patch fires for push+refresh
  - The guarded zero-spend diagnostic instrument (TestReadbackDiagnosis, realdesktop + ZCODE_ACP_READBACK_DIAG=1 double guard) with the structured DIAG verdict block
  - The corrected authenticate chain: gated probe-create read-back arm (watcher.Applied() gate), verdict still watcher.Await — ack and probe transport never trusted
  - The scrubbed diagnostic capture (experiments/captures/EXP-READBACK-DIAG-01.jsonl, scan-secrets clean) + the zero-spend ledger line under the Phase-5 row
  - UAT item 1 staged for the 05-12 guarded REAL RUN re-run (the corrected chain rides TestAPIKeyFullCycle unchanged)
affects: [05-12 REAL RUN, Phase 6 auth/registry]

# Actuals (#2632) — same estimateTokens scale (chars/4 over the realized diff).
actuals:
  tokens: 17560    # 70238 diff chars / 4 over the 8 changed files
  tasks: 3
  commits: 3       # MEASURED: git rev-list --count 28d3b82..HEAD
plan_head_before: 28d3b82323eb2e8e9c29625a2dc60b7d3ccd2757
plan_head_after: a154ff8d6664923c7b9bd9bfcc5b8962bb0aa5ad

# Tech tracking
tech-stack:
  added: []        # stdlib + the pinned module graph only (T-5-SC held)
  patterns:
    - "probe-gated read-back: the patch observer stays for patches the desktop does emit; the probe create runs only when nothing latched (the field-verified 3.18.2 shape)"
    - "capture-first diagnosis: a zero-spend guarded instrument decides the fix branch before any production change"

key-files:
  created:
    - test/realdesktop/readbackdiag_test.go
    - experiments/captures/EXP-READBACK-DIAG-01.jsonl
  modified:
    - internal/acpfront/auth.go
    - internal/hostsvc/readback.go
    - internal/hostsvc/hostsvc_test.go
    - internal/acpfront/auth_test.go
    - experiments/README.md

key-decisions:
  - "Branch (c) confirmed by field evidence (EXP-READBACK-DIAG-01, zero API spend): the probe session/create reply's settings.model.available carried zai-api/GLM-5.3-Flash + GLM-5.3 while NO state.updated fired in the 30 s window — the create reply is the registry read-back surface on desktop 3.18.2"
  - "The probe arm is GATED on watcher.Applied(): a desktop that does emit patches confirms at refresh and never probes (keeps the 05-08 teardown-wire lock of exactly one session/close green without touching files outside files_modified); on the real desktop the probe always runs (field-verified: no patch fires)"
  - "The verdict remains watcher.Await in every path — the ack and the probe's transport success are never the verdict; probe failures are recorded structurally and skipped (interruptibility contract byte-for-byte)"
  - "auth-apikey.json keeps its refreshRegistry patch side-effect: the immovable golden driver (driveAuthApikey) requires a latched surface BEFORE its first Await, and golden_test.go is outside files_modified — the corpus row stays the hostsvc-helper dual-surface rehearsal while the front-tier oracle (auth_test.go's probe arm) rehearses the field-verbatim shape"

patterns-established:
  - "Zero-spend diagnostic contract: the guarded file structurally excludes any turn-sending call (verify gate greps the absent method name; probe cycle ends at create → observe → close)"
  - "DIAG verdict block: one fact per line (ack / state-updated-in-window / create-available / watcher-verdict / branch-evidence with reason) — the instrument's product, never asserted"

requirements-completed: [AUTH-01]

coverage:
  - id: D1
    description: "Guarded zero-spend read-back diagnostic (TestReadbackDiagnosis) that runs the registry-level cycle against the real desktop and emits the structured DIAG verdict"
    requirement: AUTH-01
    verification:
      - kind: e2e
        ref: "ZCODE_ACP_READBACK_DIAG=1 go test -tags realdesktop -count=1 -run TestReadbackDiagnosis ./test/realdesktop/ (real run 2026-11-21, DIAG block + EXP-READBACK-DIAG-01 capture)"
        status: pass
      - kind: unit
        ref: "double-guard skip contract + zero-spend grep (go build/vet -tags realdesktop; grep count 0 for the turn-sending method)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Corrected authenticate chain: gated probe-create read-back arm latching the field-verified surface, ack-never-trusted negatives and interruptibility intact, red-first locked"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "internal/acpfront/auth_test.go#TestAuthenticate/probe_create_surface_latches_the_field-verified_verdict (RED evidence RED_EVIDENCE_OK, then GREEN)"
        status: pass
      - kind: unit
        ref: "internal/hostsvc/hostsvc_test.go#TestAuthReadback (all four arms incl. the Applied gate)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 ./internal/hostsvc/ ./internal/acpfront/ (all suites green)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The corrected chain proven on the REAL desktop end-to-end (authenticate via the probe-create verdict inside TestAPIKeyFullCycle)"
    requirement: AUTH-01
    verification: []
    human_judgment: true
    rationale: "Pending the 05-12 guarded REAL RUN re-run (ZCODE_ACP_REAL_E2E=1, human-gated checkpoint, quota-ledgered) — the plan's prohibitions stay flagged-unverified until that run; UAT item 1 is staged for it"
  - id: D4
    description: "Fake oracle aligned to field-verified reality; all goldens replay byte-locked; the fake-tier full cycle stays green"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "go test -race -count=1 -run TestGolden ./internal/zcode/fake/ (full corpus replay green, no byte drift)"
        status: pass
      - kind: unit
        ref: "go test -race -count=1 -run 'TestFullCycle' ./internal/acpfront/ (composed cycle green with the gated probe)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Evidence trail: scrubbed capture EXP-READBACK-DIAG-01.jsonl (scan-secrets clean) + zero-spend DIAGNOSTIC ledger line under the Phase-5 row"
    requirement: AUTH-01
    verification:
      - kind: other
        ref: "bash experiments/scan-secrets.sh (exit 0) + grep EXP-READBACK-DIAG-01 experiments/README.md (>=1)"
        status: pass
    human_judgment: false

# Metrics
duration: 23min
completed: 2026-11-21
status: complete
---

# Phase 5 Plan 11: Auth Read-Back Diagnosis & Field-Verified Fix Summary

**Field-diagnosed the SC4 auth read-back gap on real desktop 3.18.2 with a zero-spend guarded capture (branch c confirmed: the create reply IS the registry read-back surface), then fixed `Front.Authenticate` with a gated probe-create arm — the received ack still never trusted.**

## Performance

- **Duration:** 23 min
- **Started:** 2026-11-21T08:22:58Z
- **Completed:** 2026-11-21T08:45:44Z
- **Tasks:** 3 (Task 3 verification-only — see Deviations)
- **Files modified:** 8

## Accomplishments

- The read-back surface question is answered by FIELD EVIDENCE at zero API spend: the guarded diagnostic ran the registry-level cycle (push → refresh → 30 s window → probe create → probe close) against the real 3.18.2 desktop and produced the structured DIAG verdict — **branch (c)**.
- `authenticateAPIKey` now feeds the watcher the field-verified surface: after refresh, when no patch has latched, it runs the probe `session/create` (temp-dir cwd, no model binding, no turn) and offers the reply to `watcher.OfferCreateReply`, then closes the probe (`{closed:true}` — field-verified shape). `createSessionID` consumed READ-ONLY (front.go untouched — 05-10's file).
- The ack-never-trusted discipline holds in every arm: the read-back negative, the interrupt/retry arm, and the no-key arm stay green unchanged; the verdict is `watcher.Await` in every path.
- The fake-tier full cycle (`TestFullCycle`), the whole golden corpus, and every hostsvc/acpfront suite stay green under `-race`; `mise run ci` green.
- The diagnostic's evidence is ledgered: scrubbed capture `experiments/captures/EXP-READBACK-DIAG-01.jsonl` (scan-secrets exit 0) + the zero-spend DIAGNOSTIC ledger line under the Phase-5 row; UAT item 1 is staged for the 05-12 guarded REAL RUN re-run.

## The DIAG verdict (real run 2026-11-21, EXP-READBACK-DIAG-01)

```text
DIAG install-builtin: revision=44 formula=zcode-builtin:44:e9c0674a…cd1
DIAG push: revision=account-bridge:<nanos> basedOn=zcode-builtin:44:e9c0674a… providers-empty=true states-empty=true
DIAG ack: status="received" providerCount=0 receivedRevision-echo=true
DIAG refresh: ok
DIAG state-updated-in-window: no + provider-patch: no   (startup/storageState ×5 only)
DIAG create-available: zai-api/GLM-5.3,zai-api/GLM-5.3-Flash,OpenCodeGo/deepseek-v4-flash,OpenCodeGo/deepseek-flash
DIAG probe-close: ok
DIAG watcher-verdict: confirmed
DIAG branch-evidence: c-create-reply-surface — the create reply carried the want models and no
  provider patch fired; the create surface is the read-back
```

Zero orphans after the staged shutdown; zero model spend by construction (no turn-sending call exists in the diagnostic file — grep count 0).

## Task Commits

1. **Task 1: the guarded zero-spend read-back diagnostic** - `67210f9` (test)
2. **Task 2 RED: branch-(c) corrected-surface lock (fails at HEAD)** - `f308edf` (test)
3. **Task 2 GREEN: probe-create read-back arm + evidence ledger** - `a154ff8` (feat; includes the lll lint fix amended in)
4. **Task 3: fake oracle alignment + goldens + full fake-tier green** - verification-only, no file change (see Deviations §2); gates: TestGolden full-corpus replay green, all suites green under `-race`, `TestFullCycle` green, `mise run ci` green.

**Plan metadata:** committed with the docs close-out commit (SUMMARY + STATE + ROADMAP).

## Files Created/Modified

- `test/realdesktop/readbackdiag_test.go` - the guarded diagnostic instrument (realdesktop + `ZCODE_ACP_READBACK_DIAG=1` double guard, DIAG verdict block, runtime-tree candidate-(a) facts)
- `internal/acpfront/auth.go` - `probeCreateReadback`: the gated probe arm (create → OfferCreateReply → close), every failure recorded and skipped
- `internal/hostsvc/readback.go` - `Applied()` latch gate; `notConfirmed` hint cites the field-verified create-reply surface
- `internal/hostsvc/hostsvc_test.go` - the Applied-gate lock arm (false before any surface, true once latched)
- `internal/acpfront/auth_test.go` - the branch-(c) corrected-surface lock arm (RED-first evidenced) + probe rule fixtures (`{closed:true}` field-verified)
- `experiments/captures/EXP-READBACK-DIAG-01.jsonl` - the scrubbed diagnostic capture (scan-secrets clean)
- `experiments/README.md` - the DIAGNOSTIC ledger line (zero spend, capture id, branch verdict — never credentials)

## Decisions Made

- **Branch (c) implemented, per the evidence contract.** Candidates (a)/(b) were live possibilities; the capture decided: the ack echoed `received`, no patch fired, and the probe create reply carried the want models from the desktop's own registry. No hash re-pointing (a) and no providers shape (b) was invented — the reference-adapter cite-and-point stays unused because branch (b) did not confirm.
- **Probe gate on `watcher.Applied()`** (plan's own key_links wording): the state.updated observer stays for patches the desktop does emit; the probe covers the field-verified no-patch shape. This also keeps the 05-08 teardown-wire lock (exactly one back-leg `session/close` in `TestFullCycle`) green with zero edits outside `files_modified`.
- **Corpus row kept, front-tier oracle aligned** (see Deviations §2): the auth-apikey row remains the hostsvc-helper dual-surface rehearsal; the field-verbatim no-patch + probe-create rehearsal lives in `auth_test.go`'s in-code rules.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Probe arm gated on `watcher.Applied()` instead of unconditional**
- **Found during:** Task 2 (design of the GREEN step)
- **Issue:** the plan's branch-(c) action reads as an unconditional probe after refresh; an unconditional probe adds a second back-leg `session/create` + `session/close` to every authenticate, which breaks `assertFullCycleTeardownWire` (exactly one `session/close` in `TestFullCycle`) — and `fullcycle_test.go` plus the full-cycle corpus row are outside this plan's `files_modified` ("No modifications outside files_modified" is a plan success criterion).
- **Fix:** gate the probe on the watcher's latch state (`Applied()` — new exported accessor in readback.go). A desktop that emits patches confirms at refresh and never probes; the real desktop (no patch — field-verified) always probes. The plan's key_links itself prescribes the split: "offer the probe create reply for branch (c); the state.updated observer stays for patches the desktop does emit".
- **Files modified:** internal/hostsvc/readback.go, internal/acpfront/auth.go
- **Verification:** `TestFullCycle` green; all six `TestAuthenticate` arms green; hostsvc suites green under `-race`
- **Committed in:** a154ff8 (Task 2 GREEN)

**2. [Rule 3 - Blocking] Task 3's auth-apikey.json patch-drop not landable — verification-only Task 3**
- **Found during:** Task 3
- **Issue:** dropping the refreshRegistry `state.updated` side-effect from auth-apikey.json fails the immutable golden driver: `driveAuthApikey` (golden_test.go — outside `files_modified`) awaits the read-back verdict BEFORE its create+offer step, so without the scripted patch its first `Await` times out and `TestGolden` fails.
- **Fix:** keep the corpus row as the hostsvc-helper dual-surface rehearsal (both surfaces are dialect §8 wire-real — the patch route is simply not fired by push+refresh on 3.18.2). The field-verbatim rehearsal of the corrected chain lives in the front-tier oracle (auth_test.go's probe arm: ack + NO patch + create-reply surface). "Re-record the golden" resolves to "no byte drift": the full corpus replays unchanged and green.
- **Files modified:** none (verification only)
- **Verification:** `go test -race -count=1 -run TestGolden ./internal/zcode/fake/` green; `-run 'TestFullCycle'` green; `mise run ci` green
- **Committed in:** n/a (no change)

**3. [Rule 1 - Bug] Diagnostic probe used two different temp dirs for workspacePath/workspaceKey**
- **Found during:** Task 2 (post-run capture review)
- **Issue:** the diagnostic passed `t.TempDir()` twice (001/002) — the desktop accepted it (workspaceKey is host-opaque, confirmed again live), but the split keys differ from production `NewSession`'s shape.
- **Fix:** the production probe arm in auth.go uses ONE temp dir for both keys; the diagnostic file is an instrument and its capture records the fact (line 11 of EXP-READBACK-DIAG-01.jsonl).
- **Files modified:** internal/acpfront/auth.go
- **Verification:** `TestAuthenticate` probe arm green (single-dir shape)
- **Committed in:** a154ff8

---

**Total deviations:** 3 auto-fixed (2 blocking-in-scope adaptations, 1 bug). **Impact on plan:** all three were forced or informed by field evidence and existing locks; no scope creep; no files outside `files_modified` touched.

## Issues Encountered

- `gsd_run` is not on PATH in this executor's shells — invoked via its absolute path (`$HOME/.zcode/gsd-core/bin/gsd_run`) for the RED-evidence classifier and the windows ledger. No functional impact.
- The diagnostic's runtime-tree builtin candidates produced no DIAG line on the success path (they log only on unreadable candidates or when needed for a mismatch verdict). Branch (c) confirmed, so candidate-(a) facts were not needed; the mechanism stays for future runs.

## TDD Gate Compliance

| Gate | Commit | Verdict |
|------|--------|---------|
| RED | `f308edf` (`test(05-11): branch-(c) corrected-surface lock, RED`) | target `TestAuthenticate/probe_create_surface_latches_the_field-verified_verdict` failed at HEAD on the coded `ErrRegistryNotConfirmed` read-back error; `gsd_run check tdd-red-evidence` → `RED_EVIDENCE_OK` (reason: target_test_failed); record: `.planning/tdd/05-11-red-evidence.json` |
| GREEN | `a154ff8` (`feat(05-11): probe-create read-back arm…`) | descends from RED; all arms green under `-race` |
| REFACTOR | — | no separate commit (the lll lint fix was amended into GREEN; no behavior change) |

## Known Stubs

None — the probe arm is fully wired; no placeholder data paths.

## Threat Flags

None — no security-relevant surface beyond the plan's threat model (T-5-26/27/28 dispositions held: capture born outside + scrubbed before commit + scan-secrets green; no key material in any push/log/capture; zero new dependencies; zero API spend).

## User Setup Required

None new — the existing [05-USER-SETUP.md](./05-USER-SETUP.md) steps (keyring check, `ZCODE_ACP_API_KEY` + `ZCODE_ACP_RAW_CAPTURE` exports) gate the 05-12 REAL RUN re-run unchanged; the corrected chain rides the same canonical command.

## Next Phase Readiness

- **05-12 (final plan):** UAT item 1 is staged — the guarded REAL RUN (`ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestAPIKeyFullCycle ./test/realdesktop/`) now exercises the corrected chain (probe-create verdict). The human checkpoint decides per the honest-error/verdict outcome; the ledger row EXP-APIKEY-CYCLE-01 stays PENDING for it.
- Both plan prohibitions stay **flagged-unverified until that run**: ack-never-success on the real wire, and the capture born-outside/never-credentials discipline re-proven on the re-run.
- Edge-probe carry (AUTH-01/concurrency) remains a backstop marker: the interruptibility arm stayed green and the probe arm inherits the same coded-error/no-success contract; re-checked by the verifier.

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-21*
