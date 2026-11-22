---
phase: 07-hardening-vm-manual-e2e-documentation
verified: 2026-11-22T15:37:05Z
status: human_needed
score: 35/36 must-haves verified (0 behavior-unverified; 1 designed human gate = the QA-03 manual E2E clause of roadmap SC-5)
covered_files: [".planning/phases/07-hardening-vm-manual-e2e-documentation/07-01-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-01-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-02-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-02-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-03-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-03-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-04-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-04-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-05-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-05-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-06-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-06-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-07-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-07-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-08-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-08-SUMMARY.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-09-PLAN.md", ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-09-SUMMARY.md", "README.md", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "cmd/zcode-acp/testdata/omp-binding-e2e.json", "docs/provenance.md", "experiments/README.md", "experiments/vm-e2e-script.test.mjs", "internal/acpfront/conformance_test.go", "internal/acpfront/failurepaths_internal_test.go", "internal/acpfront/front.go", "internal/acpfront/lifecycle.go", "internal/acpfront/lifecycle_internal_test.go", "internal/acpfront/lifecycle_test.go", "internal/acpfront/pickerrefusal_internal_test.go", "internal/acpfront/sesscompat_test.go", "internal/acpfront/slashbusy_internal_test.go", "internal/acpfront/slashmodel_test.go", "internal/acpport/agent.go", "internal/acpport/agent_test.go", "internal/acpport/types.go", "internal/hostsvc/snapshot.go", "internal/interact/keyringprobe.go", "internal/interact/keyringprobe_classify_internal_test.go", "scripts/vm-e2e.sh"]
covered_digest: "v4:sha256:d0c2eab88809122e84d40c0021a2acb73094a243b7c769807da548eddb612f4e"
behavior_unverified: 0
overrides_applied: 0
prohibitions:
  - requirement_id: SESS-02
    statement: "MUST NOT advertise sessionCapabilities delete/fork/list/load/resume in initialize — advertising delete flips Orca into calling session/delete on its disposable listing sessions"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed: initializeResponse (internal/acpport/agent.go:186-190) sets SessionCapabilities{Options:&{}} ONLY — no delete/fork/list/load/resume field is set anywhere in the advertisement path; TestSessionCapabilitiesAdvertisementStayedHonest (sesscompat_test.go:565, mutation-verified per 07-03: a temporary Delete advertisement flips it red) ran green under -race this run."
  - requirement_id: SESS-02
    statement: "MUST NOT fake success on session/fork — the decline is the SDK's honest -32601 method-not-found shape, byte-locked, never a silent empty result"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed: agent.go:452 returns unadvertised(AgentMethodSessionFork) from the explicit UnstableForkSession arm; TestSessionForkDeclineByteLocked asserts the -32601 frame + data.method on the tee (ran green this run) and fatals on any success."
  - requirement_id: SESS-03
    statement: "MUST NOT add a second require line to go.mod — stdlib flag.FlagSet dispatch, not cobra"
    status: verified
    verification: judgment
    resolution: "Machine-verified this run: go.mod carries exactly one require line (grep -c '^require' = 1) pinned to github.com/coder/acp-go-sdk v0.20.0; the tidy-diff gate inside the green `mise run ci` run (exit 0, this run) enforces it."
  - requirement_id: SESS-03
    statement: "MUST NOT forward an unvalidated /model selector to the backend — validate-first against confirmed configOptions"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed: promptModelSlash (front.go:678) resolves the selector against latestSettings -> BuildConfigOptions listed values BEFORE any backend call and returns NewInternalWithData refusals at front.go:699-729; TestModelSlashUnknownRefusalByteLocked ran green (refusal bytes locked), and the intercept precedes acquireTurn (front.go:364-370) so nothing reaches session/send unvalidated."
  - requirement_id: QA-01
    statement: "MUST NOT retire, weaken, or skip-lock any existing corpus suite — the bar is additive"
    status: verified
    verification: judgment
    resolution: "Machine-verified this run: the full workspace gate `mise run ci` (exit 0, this run) carries the whole -race suite including TestGolden's 34 scenarios, TestFullCycle and the cancel rows; git diff 4b6d98a..HEAD touches no existing test file except additive new files (lifecycle_test.go and agent_test.go modifications are the WR-01/07-03 test additions asserted in their summaries); conformance_test.go contains zero t.Skip (grep = 0)."
  - requirement_id: QA-01
    statement: "MUST NOT pass the arc via t.Skip, shortened legs, or disabled assertions"
    status: verified
    verification: judgment
    resolution: "Machine-verified: grep -c 't.Skip' internal/acpfront/conformance_test.go = 0; the file was read in full this run — every leg (initialize/authenticate/session-new/prompt/tools/permission/cancel) carries its own Fatalf-backed asserts and the suite ran green under -race."
  - requirement_id: QA-02
    statement: "MUST NOT fetch any artifact without checksum verification — no curl-pipe-to-shell, no unverified downloads in the VM script"
    status: verified
    verification: judgment
    resolution: "Machine-verified: scripts/vm-e2e.sh contains zero curl/wget invocations (grep = 0) and performs no downloads — the header documents build-or-fetch OFF the VM; only presence asserts with actionable hints run on the VM."
  - requirement_id: QA-02
    statement: "MUST NOT echo credential values in script output or logs — env-var NAMES only"
    status: verified
    verification: judgment
    resolution: "Machine-verified: experiments/scan-secrets.sh exit 0 over the whole tree (this run); the script references ZCODE_ACP_API_KEY by name only; the live no-keyring stage run this run printed no credential material."
  - requirement_id: DOCS-01
    statement: "MUST NOT instruct exporting or pasting credential/key material in any doc or example — env-var NAMES and placeholder forms only"
    status: verified
    verification: judgment
    resolution: "Non-authoritative LLM-judge verdict, machine-backed: README.md uses ZCODE_ACP_API_KEY=<your-key> placeholder form only (read in full this run); the shim example sources a user-owned env file instead of embedding a value; check-docs.sh + scan-secrets.sh both exit 0 (this run)."
  - requirement_id: DOCS-03
    statement: "MUST NOT omit the Apache-2.0 attribution for ported reference-adapter code"
    status: verified
    verification: judgment
    resolution: "Machine-verified: docs/provenance.md 'Attribution' section (lines 40-63) names upstream william0wang/zcode-acp (Apache-2.0, tag v0.89.0), enumerates the four ported files, and cites docs/research/reference-adapters.md; check-docs.sh --phase-gate green (07-08 record; check-docs.sh green this run)."
  - requirement_id: QA-03
    statement: "MUST NOT record a passing QA-02/QA-03 verdict without the user's actual manual run — human confirmation precedes any evidence commit"
    status: verified
    verification: judgment
    resolution: "Machine-verified state: 07-UAT.md both entries read `result: awaiting (seeded 2026-11-22 …)` with no verdict recorded; experiments/README.md QA-02/QA-03 ledger rows read PENDING; REQUIREMENTS.md QA-02/QA-03 checkboxes remain unchecked — no fabricated verdict exists anywhere in the tree."
unverified_prohibition_review: "All 11 plan prohibitions hold with the listed evidence; the verdicts are LLM-judge (non-authoritative) except where a wired gate ran green (tidy-diff, scan-secrets, check-docs, mise run ci). Human review of the judgment-tier items folds into the end-of-phase checkpoint alongside the two manual gates — never a silent pass."
human_verification:
  - test: "QA-02 — one documented VM field run of scripts/vm-e2e.sh on a clean Ubuntu GNOME VM (07-UAT.md Test 1; the copy-pasteable runbook is staged there)"
    expected: "Every stage's named check prints PASS (discovery, run/arc, auth stages per VM configuration, the no-keyring stage observing exactly one coded sentinel arm verbatim); final line `vm-e2e: PASS (<n> checks, <m> skipped)`, exit 0. A failed run records its honest verdict (machine findings -> new gap plan)."
    why_human: "D-05: the script is the deliverable; one documented execution on an ACTUAL clean VM is the field proof. The executor can never provision or run the VM, and the verifier must not spend the real account's quota outside the budgeted human runs (the run/auth stages complete one minimal real turn on a logged-in machine)."
  - test: "QA-03 — the Orca card E2E through BOTH auth paths on the real account (07-UAT.md Test 2; THE phase user gate, D-06)"
    expected: "PATH 1 (api-key): card opens, authenticate{zai-api-key} succeeds, account models render, a tool-call + permission prompt turn completes. PATH 2 (coding-plan): authenticate{zai-individual-coding-plan} succeeds (read-back proven, PAT minted), models render, a streamed turn completes. BOTH: /model switch live (typo shows 'Unknown ACP model: <name>'), --model honored on a fresh card (launch log shows the binding), clean Stop, clean interrupt, no crash paths. Record verdicts PER PATH plus observations A1 (the --model id shape Orca passes) and A3 (picker behavior with configOptions but no models field)."
    why_human: "Real chat-UI client on a real account through both auth paths — never autonomous (D-06); visual/UX behavior, live model switching and the client-side observation items are unobservable to grep, and no automated test can stand in for the designed user gate."
---

# Phase 7: Hardening, VM & Manual E2E, Documentation — Verification Report

**Phase Goal:** The done-v1 bar is met: full CI protocol conformance, a reproducible clean-VM scenario, manual E2E confirmed in a real chat-UI client on a real account through both auth paths, and English documentation including the launch recipe for ACP hosts that only accept preset agents.
**Verified:** 2026-11-22T15:37:05Z
**Status:** human_needed
**Re-verification:** No — initial verification (no prior VERIFICATION.md existed; the "review-fix series" framing refers to WR-01/02/03 landing in-phase before this report — those commits are part of the verified source truth at HEAD c0207ce).

## MVP Mode Guard

`**Mode:** mvp` in ROADMAP.md, but the phase goal is NOT in user-story form: `user-story.validate` on the ROADMAP goal returns **valid: false** (re-run this session, exit 0). **This is a surfaced discrepancy** — under the MVP-mode contract the goal would be reformatted via `/gsd:mvp-phase 7`. Verification proceeded anyway per the orchestrator's explicit directive, goal-backward against the five ROADMAP success criteria (the contract), because (a) this is the milestone's final close-out phase and the goal is a done-bar statement, not a capability slice, and (b) the designed user-flow tier for this phase IS the QA-02/QA-03 manual gates — seeded as copy-pasteable user walk-throughs in 07-UAT.md before this report. No synthetic "User Flow Coverage" table was fabricated against a non-user-story goal; the human verification section below carries the two user-flow gates verbatim from the UAT.

## Goal Achievement

### Observable Truths — ROADMAP Success Criteria (the contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | CI protocol conformance suite green: fake app-server drives initialize → session/new → prompt streaming → tool calls → permissions → cancel | ✓ VERIFIED | `TestConformanceV1Arc` (internal/acpfront/conformance_test.go:77) — read in full (every leg carries Fatalf-backed frame asserts; zero t.Skip) and RUN GREEN under `-race -count=1` this run (7.9s batch); green inside `mise run ci` exit 0 (this run). Behavioral test, not just presence. |
| 2 | Clients sending `session/delete` keep working (graceful no-op); `session/fork` handled gracefully — never a crash path | ✓ VERIFIED | Front.DeleteSession (front.go:640, registry-miss silent success, registered id → CloseSession teardown); explicit fork decline arm (agent.go:452). Tests run green under -race this run: TestSessionDeleteRegisteredRunsTeardown, TestSessionDeleteUnknownSilentSuccess (unknown/empty/already-closed arms), TestSessionForkDeclineByteLocked, TestUncoveredMethodDeclines. |
| 3 | Every ACP method Orca calls at card startup/per turn (per DISC-02) implemented or gracefully handled | ✓ VERIFIED | DISC-02 table (docs/protocol/acp-client-dialect.md:220-276) swept method-by-method against the code: initialize/authenticate/session-new/prompt/cancel/set_config_option/close/resume implemented (phases 4-6); delete/fork/list/load/set_mode declined byte-locked (sesscompat_test.go:376 table — run green); set_model → SDK fallback row (raw-frame test); advertisement stays options-only so capability-gated methods never fire from Orca. |
| 4 | Clean Ubuntu GNOME VM scenario — install, configure, run — scripted in the repo and reproducible, incl. the no-keyring error path | ✓ VERIFIED | scripts/vm-e2e.sh (868 lines): five stages, named checks, exit-code contract. Live this run: `--stage no-keyring` 5/5 PASS with the binary-missing sentinel surfaced verbatim; `--stage discovery` PASS (read-only); contract suite 3/3 with the live stripped-PATH arm. Byte-identity of the four sentinels to keyringprobe.go verified by grep -F both directions. The clean-VM FIELD RUN remains the designed human gate (QA-02 below) — the scripted+reproducible clause of this SC is what is claimed here. |
| 5 | Manual E2E confirmed (Orca, real account, BOTH auth paths) + documentation published (English README, launch recipe, docs/ protocol notes with provenance + limitations) | ⏳ HUMAN GATE (docs half ✓ VERIFIED, manual-E2E half awaiting the user) | Docs half: README.md + docs/provenance.md read in full — every required section present, facts traced to code (version stamp main.go:41, sentinels keyringprobe.go:43-60, refusal strings front.go:526-527), check-docs.sh + scan-secrets.sh exit 0. Manual-E2E half: NOT performed — the designed user gate (07-UAT.md Test 2) awaits the user's runs; nothing may stand in for it (D-06). |

**Score:** 35/36 must-haves verified (0 present-behavior-unverified; the 1 unverified item is the designed QA-03 human gate clause — see human_verification)

### Observable Truths — Plan Must-Haves (31 truths, per-plan)

All 31 plan truths verified against the current tree; the review-fix series (6ffad04..a124a29) is part of the verified state.

| Plan | Truths | Status | Key evidence (this run unless noted) |
|------|--------|--------|------|
| 07-01 (D-08 lifecycle) | 4/4 | ✓ VERIFIED | registerIfOpen one closingMu section (lifecycle.go:296-324, read); TestResumeCloseInterleave (2000-iter stress) + TestRegisterIfOpen + TestCreateFailurePathsLeaveNoMapEntries + TestPickerSetModelRefusalRunsFullTeardown all green under -race; `if f.settings == nil` grep = 0 (IN-04); teardownAfterPickerRefusal at front.go:770 |
| 07-02 (D-08 advisories) | 4/4 | ✓ VERIFIED | decodeBuiltinTable single read (snapshot.go:255-266, read); classify switch order parent-cancel→success→deadline→marker→default read at keyringprobe.go (success arm before deadline); TestKeyringProbeClassify* green under -race; four sentinels byte-identical (grep -F go=1 script=1 ×3 + the joined no-desktop-session half); 'validated non-empty' grep = 0 |
| 07-03 (SESS-02 + D-02) | 5/5 | ✓ VERIFIED | five sesscompat rows run green under -race; advertisement honesty locked by mutation-verified guard (sesscompat_test.go:565); wireError/DeleteSession delegation read in agent.go |
| 07-04 (D-03 CLI) | 4/4 | ✓ VERIFIED | live `zcode-acp --version` and `zcode-acp omp --version` both print `zcode-acp 18.1.9+zacp.0` (built binary this run); TestOmpVersionProbeFloor/TestOmpLaunchBinding/TestOmpFlagSet/TestOmpRootCoexistence/TestOmpExitCodes green under -race; go.mod one require line pinned v0.20.0; scan-secrets green |
| 07-05 (D-03 slash) | 5/5 | ✓ VERIFIED | six TestModelSlash* tests green under -race incl. TestModelSlashRefusalHoldsNoTurnSlot (slot-free), TestModelSlashSelectorByteEquality, TestModelSlashCarriesConfirmedLevel; pinned strings exist exactly once each (front.go:526-527); intercept precedes acquireTurn (front.go:364-370, read); WR-02 busy gate: TestModelSlashMidTurnRejectedWhenTurnActive + TestPromptModelSlashBusyScope green (slashbusy_internal_test.go:139) |
| 07-06 (QA-01) | 4/4 | ✓ VERIFIED | conformance_test.go read in full (304 lines): one harness, per-leg asserts, exactly-once cancel resolution + blanket no--32800 scan, additive composition inside the file; TestConformanceV1Arc green; TestGolden corpus untouched (full ci green) |
| 07-07 (QA-02 script) | 4/4 | ✓ VERIFIED | script stages run live this run (no-keyring 5/5 PASS, discovery PASS); sentinel byte-identity verified; paired-env read-back assert present at scripts/vm-e2e.sh:781 (`account:zai-individual-coding-plan/` grep on the session/new REPLY); named PASS/FAIL lines + summary contract observed live; contract suite (experiments/vm-e2e-script.test.mjs) 3/3 |
| 07-08 (DOCS-01/02/03) | 4/4 | ✓ VERIFIED | README + provenance read in full; CI/required-checks/branch sections present; launch recipe complete (override mechanics, VAR=val rejection, floor self-check, env per path, launchesWithModel, byte-exact refusals); gates exit 0 |
| 07-09 (handoff) | 5/5 | ✓ VERIFIED | `mise run ci` exit 0 + `go vet -tags realdesktop` clean (both this run); 07-UAT.md + ledger rows seeded PENDING (read); no verdict recorded anywhere (the prohibition holds); REQUIREMENTS.md QA-02/QA-03 untouched (orchestrator's act); A1/A3 observation items staged in the UAT (the truth asserts the FLAGGING — the observations themselves ride the QA-03 human gate) |

### Deferred Items

None. No gap was deferred to a later phase (Phase 7 is the milestone's final phase; the two open items are end-of-phase human gates, not later-phase work). The 8 open code-review advisories (IN-a..IN-h in 07-REVIEW-DISPOSITION.md) are declared post-v1 advisories — none is a must-have failure.

### Required Artifacts

All 21 must_haves.artifacts across the 9 plans: **exists=true, substantive, 21/21 PASS** (`verify.artifacts` exit 0 per plan, run this session). Wiring: 18/18 must_haves.key_links **WIRED** (`verify.key-links` exit 0 per plan). Level-3 spot-checks beyond the verbs: ComposeFront ridden not modified; vm-e2e.sh → keyringprobe.go byte-identity; main.go → applyConfiguredModel via WithModel; UAT → README recipe (`zcode-acp omp` present).

### Data-Flow Trace (Level 4)

No UI renders dynamic data in this phase; the trace applies to doc claims and script assertions instead — each traced to its real source:

| Artifact | Claim/Value | Source | Flows | Status |
|----------|-------------|--------|-------|--------|
| README.md | version `18.1.9+zacp.0` | cmd/zcode-acp/main.go:41 `var version` | live binary prints it | ✓ FLOWING |
| README.md | four keyring sentinels | internal/interact/keyringprobe.go:43-60 | byte-identical (grep -F), live stage surfaced one verbatim | ✓ FLOWING |
| README.md | refusal strings | front.go:526-527 constants | tee-locked by TestModelSlashUnknownRefusalByteLocked | ✓ FLOWING |
| scripts/vm-e2e.sh | paired-env read-back | session/new REPLY models list | assert at script:781 on the reply, never the ack | ✓ FLOWING |
| docs/provenance.md | ported-file attribution | the four real files (credstore/cipherdecrypt/snapshot/patminter) | paths exist in tree | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Conformance arc (SC-1) | `go test -race -count=1 -run '^(TestConformanceV1Arc\|...17 named acpfront tests...)$' ./internal/acpfront/` | ok 7.9s | ✓ PASS |
| WR-02 mid-turn gate | `-run '^(TestPromptModelSlashBusyScope\|TestModelSlashMidTurnRejectedWhenTurnActive)$'` | ok 1.1s | ✓ PASS |
| WR-01 resume settings | `-run '^TestResumeRecordsConfirmedSettings$'` | ok 1.2s | ✓ PASS |
| IN-03 classify order | `-run '^TestKeyringProbeClassify' ./internal/interact/` | ok 1.2s | ✓ PASS |
| omp CLI contract | `-run '^TestOmp' ./cmd/zcode-acp/` | ok 4.2s | ✓ PASS |
| Version floor (DOCS-02) | `/tmp/verify-p7/zcode-acp --version && ... omp --version` | `18.1.9+zacp.0` ×2 | ✓ PASS |
| vm-e2e contract suite (live) | `ZCODE_ACP_VM_E2E_BIN=<bin> node --test experiments/vm-e2e-script.test.mjs` | 3/3 pass (binary-missing sentinel live) | ✓ PASS |
| vm-e2e no-keyring stage (live) | `bash scripts/vm-e2e.sh --stage no-keyring --binary <bin>` | 5 checks PASS, exit 0 | ✓ PASS |
| vm-e2e discovery stage (live, read-only) | `bash scripts/vm-e2e.sh --stage discovery` | PASS (found real desktop install) | ✓ PASS |
| vm-e2e auth-codingplan / run stages | not run — they complete a minimal REAL turn on the logged-in desktop | spends the account's quota outside the budgeted human runs; mechanics contract-locked + assert code-verified (script:781) | ? SKIP → folded into the QA-02 human gate |
| Full workspace gate (once) | `mise run ci` | exit 0 — build, vet, lint 0, 10 packages under -race, experiments suite | ✓ PASS |
| Guarded real-tier compile | `go vet -tags realdesktop ./test/realdesktop/` | clean | ✓ PASS |
| Docs gates | `check-docs.sh` / `scan-secrets.sh` | both exit 0 | ✓ PASS |

### Probe Execution

| Probe | Command | Result | Status |
|-------|---------|--------|--------|
| experiments/vm-e2e-script.test.mjs | `node --test` (guarded live arm with ZCODE_ACP_VM_E2E_BIN) | 3 pass / 0 fail / 0 skip | PASS |
| experiments/ node suite (in ci) | `mise run ci` → test-experiments step | 0 fail (1 guarded skip without the env) | PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|---------------|-------------|--------|----------|
| DOCS-01 | 07-08 | README user surface (what/install/config/both auth paths/troubleshooting) | ✓ SATISFIED | README read in full; all sections present; gates green |
| DOCS-02 | 07-08 | Launch recipe for preset-agent-only hosts | ✓ SATISFIED | Recipe section complete; version floor live-proven; env mechanisms documented |
| DOCS-03 | 07-08 | docs/ protocol notes + provenance + known limitations | ✓ SATISFIED | docs/provenance.md read in full; attribution + limitations present |
| QA-01 | 07-06 | CI protocol conformance suite green | ✓ SATISFIED | TestConformanceV1Arc green (-race + inside mise run ci) |
| QA-02 | 07-07, 07-09 | VM test scenario scripted + reproducible from the repo | ✓ SATISFIED (script) / ⏳ NEEDS HUMAN (the one documented clean-VM field run — the designed D-05 proof) | Script live-proven this run; field run = 07-UAT.md Test 1 |
| QA-03 | 07-09 | Manual E2E in real Orca, real account, both auth paths | ⏳ NEEDS HUMAN | The designed user gate (D-06); 07-UAT.md Test 2 seeded and awaiting |
| SESS-02 | 07-01, 07-03 | session/delete graceful no-op; fork graceful | ✓ SATISFIED | delete/fork rows green; full teardown on registered ids |
| SESS-03 | 07-01..07-05 | Full Orca dialect coverage incl. persona CLI contract | ✓ SATISFIED | DISC-02 sweep locked; --model binding + /model slash with byte-exact Hb strings |

**Orphaned requirements:** none — REQUIREMENTS.md maps exactly these 8 IDs to Phase 7, and every one is claimed by at least one plan. REQUIREMENTS.md state is consistent with the honest close: DOCS-01/02/03, QA-01, SESS-02/03 marked Complete; QA-02/QA-03 remain `[ ]` Pending awaiting the user's manual runs (the checkbox flips are the orchestrator's post-verification act).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| scripts/vm-e2e.sh | 249, 538, 803 | `XXXXXX` matches (mktemp templates, not debt markers) | ℹ️ Info (false positive) | none |
| README.md | 213 | word "Placeholder" (pre-existing Phase-2 CI table text describing the e2e-matrix workflow) | ℹ️ Info (false positive) | none |
| cmd/zcode-acp/main.go | 454, 465 | `return []string{...}` (real default-model code paths, not empty stubs) | ℹ️ Info (false positive) | none |

**Debt-marker gate:** zero TBD/FIXME/XXX markers in any phase-modified file. **Blockers:** none.

### Human Verification Required

### 1. QA-02 — one documented VM field run of scripts/vm-e2e.sh on a clean Ubuntu GNOME VM

**Test:** Follow the copy-pasteable runbook in 07-UAT.md Test 1: build the binary off-VM (`go build -o zcode-acp ./cmd/zcode-acp`), transfer script + binary, run `bash ~/vm-e2e.sh --binary ~/zcode-acp` (or per-stage; `--require-auth` hardens).
**Expected:** every named check PASS; `vm-e2e: PASS (<n> checks, <m> skipped)`, exit 0; exactly one keyring sentinel arm observed verbatim (on a stripped-PATH VM: the deterministic binary-missing sentence). A failed run records its honest verdict — machine findings become a new gap plan.
**Why human:** D-05 — the script is the deliverable; one documented execution on an actual clean VM is the field proof. No toolchain provisioning or VM run can be automated here, and the verifier must not spend the real account's quota outside the budgeted runs.

### 2. QA-03 — the Orca card E2E through BOTH auth paths on the real account (THE phase user gate)

**Test:** Follow the README launch recipe with Orca.AppImage + the real z.ai account, per the staged checklist in 07-UAT.md Test 2: PATH 1 (api-key via shim/agentDefaultEnv) and PATH 2 (desktop coding-plan login + unlocked keyring), each through the five checklist steps; then the BOTH-paths items (/model switch, --model launch binding via the wrapper's launch log, clean Stop, clean interrupt).
**Expected:** both paths complete full turns; close/cancel clean; no crash paths; record verdicts PER PATH plus observations A1 (the exact `--model` id shape Orca passes) and A3 (picker behavior with configOptions but no `models` field).
**Why human:** real chat-UI client on a real account — the designed never-autonomous user gate (D-06); UX behavior, live switching, and client-side variance are unobservable to grep, and no test can stand in for the gate the phase was designed around.

## Gaps Summary

**No gaps in the automated surface.** Everything checkable against the codebase is verified with behavioral evidence: all 21 artifacts substantive, all 18 key links wired, all 31 plan truths and 4 of 5 roadmap SCs verified (most backed by tests run green under -race this run, not merely present), the full `mise run ci` gate exit 0, the VM script's live stages green, docs gates green, zero debt markers, zero blockers, and the review-fix series (WR-01/02/03) confirmed landed and test-locked. The phase's own design routes exactly two items to the human tier — the QA-02 clean-VM field run and the QA-03 Orca card E2E through both auth paths — both staged as runbooks in 07-UAT.md and PENDING in the experiments ledger before this report. The status is the designed `human_needed` shape: automated checks all pass; the two manual runs await the user; on their honest completion (and evidence recording per the UAT discipline) the phase — and the v1 milestone — can close. One meta-item for the user: the MVP-mode goal-format discrepancy noted above (run `/gsd:mvp-phase 7` if MVP framing matters for the record, or accept the done-bar goal as written).

---

_Verified: 2026-11-22T15:37:05Z_
_Verifier: Claude (gsd-verifier)_
