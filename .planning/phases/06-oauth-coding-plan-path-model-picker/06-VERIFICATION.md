---
phase: 06-oauth-coding-plan-path-model-picker
verified: 2026-11-22T05:37:58Z
status: human_needed
score: 15/16 must-haves verified
covered_files:
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-01-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-01-SUMMARY.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-02-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-02-SUMMARY.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-03-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-03-SUMMARY.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-04-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-04-SUMMARY.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-05-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-05-SUMMARY.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-06-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-06-SUMMARY.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-07-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-07-SUMMARY.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-08-PLAN.md"
  - ".planning/phases/06-oauth-coding-plan-path-model-picker/06-08-SUMMARY.md"
  - "cmd/zcode-acp/main.go"
  - "cmd/zcode-acp/main_test.go"
  - "experiments/README.md"
  - "experiments/scan-secrets.sh"
  - "internal/acpfront/auth.go"
  - "internal/acpfront/auth_test.go"
  - "internal/acpfront/codingplan_cycle_test.go"
  - "internal/acpfront/configoptions.go"
  - "internal/acpfront/configoptions_test.go"
  - "internal/acpfront/front.go"
  - "internal/acpfront/initialize_test.go"
  - "internal/acpfront/lifecycle.go"
  - "internal/acpfront/lifecycle_test.go"
  - "internal/acpfront/loop_test.go"
  - "internal/acpfront/newsession_test.go"
  - "internal/acpfront/setconfigoption_test.go"
  - "internal/acpfront/tool_stream_test.go"
  - "internal/acpport/agent.go"
  - "internal/acpport/agent_test.go"
  - "internal/acpport/types.go"
  - "internal/hostsvc/headers_pat.go"
  - "internal/hostsvc/headers_pat_test.go"
  - "internal/hostsvc/hostsvc_test.go"
  - "internal/hostsvc/patminter.go"
  - "internal/hostsvc/patminter_test.go"
  - "internal/hostsvc/snapshot.go"
  - "internal/hostsvc/snapshot_account_test.go"
  - "internal/interact/cipherdecrypt.go"
  - "internal/interact/cipherdecrypt_test.go"
  - "internal/interact/credstore.go"
  - "internal/interact/credstore_test.go"
  - "internal/interact/familysel.go"
  - "internal/interact/familysel_test.go"
  - "internal/interact/keyringprobe.go"
  - "internal/interact/keyringprobe_test.go"
  - "internal/interact/responder.go"
  - "internal/interact/responder_test.go"
  - "internal/zcode/fake/golden_test.go"
  - "internal/zcode/fake/testdata/golden/coding-plan-auth-cycle.golden"
  - "internal/zcode/fake/testdata/golden/coding-plan-probe-fail.golden"
  - "internal/zcode/fake/testdata/golden/config-option-set-model.golden"
  - "internal/zcode/fake/testdata/golden/config-option-thought.golden"
  - "internal/zcode/fake/testdata/golden/runtime-headers-pat.golden"
  - "internal/zcode/fake/testdata/scenarios/coding-plan-auth-cycle.json"
  - "internal/zcode/fake/testdata/scenarios/coding-plan-probe-fail.json"
  - "internal/zcode/fake/testdata/scenarios/config-option-set-model.json"
  - "internal/zcode/fake/testdata/scenarios/config-option-thought.json"
  - "internal/zcode/fake/testdata/scenarios/runtime-headers-pat.json"
  - "test/realdesktop/apicycle_test.go"
  - "test/realdesktop/codingplan_test.go"
covered_digest: "v4:sha256:bb9b15797406311d9c930ba1b91af5473902f6b3021ea58f5e70110a40ec7bb1"
behavior_unverified: 1
behavior_unverified_items:
  - truth: "The real-account coding-plan run (D-12 phase bar, ROADMAP SC1's 'on a real account'): the official chain completes on the real account — registry read-back proves the account models, the real PAT mints at api.z.ai, configOptions render from the live registry, one real tool-less turn completes"
    test: "Run the canonical command from test/realdesktop/codingplan_test.go's header (desktop logged into the individual coding plan, keyring unlocked, ZCODE_ACP_RAW_CAPTURE set to a scratch path OUTSIDE the repo): export ZCODE_ACP_RAW_CAPTURE=<scratch path outside this repo>; ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/"
    expected: "authenticate{zai-individual-coding-plan} proven by the registry read-back (account models present — never the ack), the real PAT minted via api.z.ai, session/new configOptions carrying both entries from the live registry, one streamed tool-less turn with the runtime-headers ask answered (token+id+accountScope), staged shutdown with the zero-orphan proof; then scrub the raw capture into experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl, fill the ledger actuals + V-EXP id, run scan-secrets on the tree AND the staged diff, record the A4/A6/A7 confirm-or-adjust notes, and confirm the evidence commit"
    why_human: "Spends real quota on the human's coding-plan account through the desktop keyring — D-12/D-13 forbid autonomous or CI execution by design (06-08 truth 3; the 05-12/05-14 discipline). No automated tier can exercise the real keyring or the real api.z.ai chain."
overrides_applied: 0
human_verification:
  - test: "THE REAL RUN (end-of-phase UAT, harvested from 06-08 Task 3 <human-check>): desktop logged into the z.ai individual coding plan, gnome-keyring unlocked, ZCODE_ACP_RAW_CAPTURE pointed at a scratch path outside the repo; run: export ZCODE_ACP_RAW_CAPTURE=<scratch path outside this repo> && ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/"
    expected: "PASS: authenticate read-back proven from registry state (account models present), the real PAT mint at api.z.ai, session/new configOptions rendered from the live registry, one real tool-less tiny turn with the runtime-headers ask answered (token+id+accountScope), staged shutdown zero-orphan. After the run: scrub the raw capture into experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl, fill the ledger actuals + V-EXP id in experiments/README.md, scan-secrets green on the tree AND the staged diff, record A4 (thoughtLevel block) / A6 (clientVersion) / A7 (configOptions rendering) confirm-or-adjust notes, then confirm the evidence commit. A failed leg records the honest verdict with the scrubbed capture — the phase stays open (the 05-14 discipline)"
    why_human: "Spends real quota through the desktop keyring on the human's account (D-12/D-13: never autonomous, never CI). Every automated tier is proven green this pass; the handoff artifacts are complete (guarded harness race-clean per the WR-06 fix, canonical command + evidence contract in the test header, ledger row EXP-CODINGPLAN-CYCLE-01 PENDING, scrubber + scan-secrets gate green)"
---

# Phase 6: OAuth Coding-Plan Path & Model Picker Verification Report

**Phase Goal:** Users on an individual coding plan can authenticate through the official OAuth chain end-to-end, and any client can see and switch model and thought level from the live provider registry.
**Verified:** 2026-11-22T05:37:58Z
**Status:** human_needed
**Re-verification:** No — initial verification

**MVP-mode note:** ROADMAP marks this phase `mode: mvp`; the goal is outcome-phrased and fails the `user-story.validate` guard (prior pass finding, unchanged — same shape as Phases 4/5). Per the milestone's uniform precedent, the User Flow table below is derived from the roadmap contract (the 5 success criteria), and the per-plan user stories in COVERAGE.md's MVP Framing Note are honored as written.

## User Flow Coverage (MVP)

| # | Flow step (from the goal) | Expected | Evidence in codebase | Status |
|---|---------------------------|----------|----------------------|--------|
| 1 | Coding-plan user runs `authenticate{zai-individual-coding-plan}` | The official chain runs: keyring probe FIRST → desktop login read → account overlay push → registry read-back verdict → PAT mint exactly once | internal/acpfront/auth.go:227-294 (probe→login→BuildAccount→want list→watcher-before-push→push→refresh→probe-create→Await→Ensure); golden corpus coding-plan-auth-cycle + coding-plan-probe-fail green; TestCodingPlanFullCycleFake PASS under -race | ✓ VERIFIED (machinery) / real-account half → Human Verification |
| 2 | The chain's verdict and the PAT | Verdict is registry state, never the ack; PAT minted via the official api.z.ai chain with accountScope | auth.go:273-291 (Await is the only success path; Ensure only after); patminter.go (getCustomerInfo → personal pick → api_keys list/create-if-absent → access_tokens; TestPatMinterHappyPath/Digest/SingleFlight green) | ✓ VERIFIED (machinery) |
| 3 | No keyring machine (headless/CI) | Fail fast with a coded actionable error naming the missing piece; never hangs, no push traffic | keyringprobe.go (LookPath → libsecret-tools error; 5 s bounded ctx; Cannot-autolaunch + log-in-via-desktop hints); probe-fail golden proves NO provider frame on the wire after a probe failure | ✓ VERIFIED |
| 4 | Any client: `session/new` | Returns `configOptions` (model + thought level) built verbatim from the backend's confirmed settings snapshot; server-confirmed currentValue only | configoptions.go (ParseSettings/BuildConfigOptions; omission over fabrication — locked OQ2); front.go:302-314 (reply IS the confirmation); agent.go:194,250 (SDK select assembly); TestNewSessionConfigOptions + translator vectors green | ✓ VERIFIED (machinery) |
| 5 | Client switches model / thought level | `session/set_config_option` validates first (-32602 naming configId + allowed set), maps to `session/setModel` with the level carried (OQ1), re-emits the full reply-built list | front.go:503-557 (validate-first → MethodSessionSetModel → store update → rebuild); config-option-set-model/-thought corpus rows byte-lock the wire; TestSetConfigOption* green | ✓ VERIFIED |
| 6 | Subsequent turns run | The mid-turn runtime-headers ask is answered from the in-process PAT cache: requestAuth {apiKey, apiKeyId, accountScope} | headers_pat.go (cache answer / exactly-once re-mint / scope refusal); responder.go:457 (headersApplied hardcoded true); TestPatHeadersSource 9 arms + TestCodingPlanFullCycleFake turn leg green | ✓ VERIFIED (machinery) / real leg → Human Verification |

## Goal Achievement

### Observable Truths

Consolidated set: ROADMAP SC1–SC5 + the eight PLAN frontmatter clusters, with the three bespoke prohibitions (COVERAGE.md's kept set) as explicit rows.

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | SC1 machinery: `authenticate{zai-individual-coding-plan}` runs the composed official chain — probe-first, login read, BuildAccount overlay, watcher-before-push, registry read-back verdict (never the ack), PAT mint exactly once after the verdict | ✓ VERIFIED | Chain read first-hand (auth.go:227-294); corpus rows green; TestCodingPlanFullCycleFake PASS under -race in a dedicated run this pass; full `mise run ci` exit 0 |
| 2  | SC1 real-account bar (D-12): the chain completes ON A REAL ACCOUNT and subsequent turns run | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Machinery proven end-to-end at the fake tier; the real-keyring/real-api.z.ai leg is the human-gated end-of-phase UAT by design (06-08 truth 3) — see Human Verification |
| 3  | SC2: both advertised authMethods authenticate; account models appear after auth | ✓ VERIFIED | TestAuthenticate (api-key arm untouched) + TestCodingPlanAuth arms green in CI; want list mirrors the BUILT snapshot (auth.go:332-348, `wantModelsFromSnapshot`); account-model appearance proven by the registry, not assumed; api-key real-account field proof is Phase 5's recorded PASS |
| 4  | SC3: keyring-less machine fails fast with a coded actionable error, never hangs | ✓ VERIFIED | TestKeyringProbe arms green (libsecret-tools naming, autolaunch class, log-in hint, 5 s bound); probe-fail golden: probe failure leaves NO provider/updateAccountConfig frame (assertCodingPlanProbeFailTranscript fatals on any provider frame) |
| 5  | SC4: `session/new` returns configOptions covering model + thought level, built from the live provider registry | ✓ VERIFIED | TestNewSessionConfigOptions + TestConfigOptionsTranslator + TestNewSessionAssemblesConfigOptions green; verbatim-mirror + per-model-levels + fallback + omission vectors; no hardcoded model/level lists in acpfront (grepped this pass) |
| 6  | SC5: config-option change maps to setModel on the wire, re-emits the full list, currentValue always real | ✓ VERIFIED | TestSetConfigOptionInvalid/ModelSwitch/ThoughtSwitch/ReEmit green; config-option corpus rows byte-lock the setModel payload incl. options.reasoningLevel; the reply rebuilds the list (front.go:552-557) |
| 7  | 06-01: credential store reader — prefix dispatch (plaintext / enc:v1 / enc:v2:os), unknown-enc never passes as plaintext, per-entry isolation, missing file = empty+nil, v1 fallback behind the same KeyLoader seam, accessors + hygiene | ✓ VERIFIED | TestCredStore/Accessors/LogHygiene/MissingAndEmpty green in CI; cipherdecrypt.go carries the v1 sibling loader (590 lines; KeyLoader seam); unknown-prefix rule present |
| 8  | 06-02: PAT minter — the official 6-step api.z.ai chain verbatim, accountScope byte-parity digest, in-process cache + single-flight, reactive surface, bounded HTTPS + coded errors | ✓ VERIFIED | TestPatMinterHappyPath/Digest/LoginRequired/PickUnavailable/MultiPickWarn/CreateIfAbsent/Disabled/BizFail/Key404/SingleFlight/ReactiveSurface/TransientBackoff green in CI (-race); 15 s client timeout + explicit transport in patminter.go |
| 9  | Prohibition (06-02, test-tier): the PAT, login token, or key material is NEVER persisted to disk or any store | ✓ VERIFIED (wired enforcement) | TestPatMemoryOnly AST-scans patminter.go for write-intent calls (os.Create/WriteFile/OpenFile/Write/WriteString) and Pat rendering methods — green in CI this pass |
| 10 | 06-03: BuildAccount produces the NON-empty account overlay — zhipu-account filter, verbatim builtin-table mirror, entitlement gate RECEIVES the decision, exact-revision formula reused, malformed table = coded error | ✓ VERIFIED | TestAccountSnapshotVerbatimMirror/GateOmission/NoRules/MalformedTable green; BuildRevision's api-key empty overlay untouched (TestPushEnvelopeCompat green); supervisor.BuiltinRevision reuse read first-hand (snapshot.go:154) |
| 11 | 06-05: composed authenticate arm + proactive probe (D-02) + production entitlement gate (familysel reads ONLY desktop state, degrades honestly) + api-key arm behaviorally untouched | ✓ VERIFIED | TestCodingPlanAuth arms + TestFamilySelections green; AuthConfig seams (Creds/Pat/Probe/Entitle) read first-hand (auth.go:108-148); api-key corpus rows + locks stayed green throughout (CI exit 0) |
| 12 | Prohibition (06-05, test-tier): the provider/updateAccountConfig ack is NEVER treated as coding-plan auth success — the verdict is exclusively the registry read-back | ✓ VERIFIED (wired enforcement) | auth.go:273-282: `watcher.Applied()`/probe-create then `Await` is the ONLY success path, mint strictly after; the probe-fail golden's transcript assertion fatals on any provider frame; corpus green in CI |
| 13 | 06-06: set_config_option validates FIRST (-32602 naming configId + allowed set), model change carries the current level (Pitfall 2), thought-level rides the SAME setModel path (locked OQ1 — no dedicated method), re-emits the reply-built list | ✓ VERIFIED | TestSetConfigOptionInvalid arms green; corpus goldens byte-lock both switch payloads; no new method constant (envelope.go MethodSessionSetModel carries both — grepped) |
| 14 | 06-07: mode-aware runtime-headers serving — accountScope rides the widened seam, api-key path byte-compatibly identical, D-05 re-mint exactly once per ask, scope confusion refused, cmd wiring with NO new flags | ✓ VERIFIED | TestPatHeadersSource 9 arms enumerated and PASS under -race in a dedicated run (fingerprint-match → exactly one re-mint; second rejection → coded error; scope gate outranks fingerprint); TestRuntimeHeadersSeamWidening byte-locks the api-key reply (responder_test.go:771,789); cmd flags unchanged (5 pre-existing flags only — grepped); wiring read first-hand (main.go:295-357) |
| 15 | Prohibition (06-07, test-tier): the failure-shaped runtime-headers success (headersApplied:false or a PAT-less requestAuth) is NEVER emitted — failures are coded error envelopes | ✓ VERIFIED (wired enforcement) | responder.go:457 hardcodes `"headersApplied": true`; headers_pat.go answers ErrPatScopeInvalidated/coded errors on the failure arms (lines 117,151); tests assert the three-field success SHAPE and error envelopes (headers_pat_test.go:288,398,410; responder_test.go:758-789); green in CI |
| 16 | 06-08: zero-spend composed fake full cycle + guarded real-tier sibling (compiles under the tag, skips safely) + evidence preparation (PENDING ledger row, scrub gate, canonical command) | ✓ VERIFIED | TestCodingPlanFullCycleFake PASS under -race (dedicated run); tagged vet/build green this pass; guarded skip proven (`--- SKIP: TestCodingPlanFullCycle` with the manual-only message, 0.00s); ledger row EXP-CODINGPLAN-CYCLE-01 PENDING present (experiments/README.md:239); scan-secrets exit 0 re-run this pass; cpAskRecorder mutex-guarded (WR-06 fix d5e8c22 verified in source) |

**Score:** 15/16 truths verified (1 present, behavior-unverified — the designed human-gated real run)

**Anti-pattern scan:** clean — no TBD/FIXME/XXX markers in any phase-modified production file (grepped this pass); no TODO/HACK/PLACEHOLDER; no stub `return` shapes (all flagged sites are legitimate function tails, read first-hand). Full `mise run ci` exit 0 at HEAD (build + vet + lint 0 issues + full -race suite + experiments 22/22).

### Required Artifacts

All 8 plans declare `must_haves.artifacts` as bare strings; the verify.artifacts verb therefore had zero checkable units (it checks only `{path: ...}` objects) and was supplemented by the manual three-level verification below. Every declared artifact exists, is substantive, and is wired.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| internal/interact/credstore.go | store reader + accessors | ✓ VERIFIED | 323 lines; prefix dispatch, isolation, LoginToken/ZcodeJWT |
| internal/interact/cipherdecrypt.go | v1 fallback behind KeyLoader | ✓ VERIFIED | 590 lines; v1+v2 loaders, one seam |
| internal/hostsvc/patminter.go | official chain + cache | ✓ VERIFIED | 1220 lines; 6-step chain, single-flight, accountScope |
| internal/hostsvc/snapshot.go | BuildAccount overlay | ✓ VERIFIED | 462 lines; zhipu-account filter, formula reuse |
| internal/acpfront/configoptions.go | translator | ✓ VERIFIED | 396 lines; ParseSettings/BuildConfigOptions |
| internal/interact/keyringprobe.go | proactive probe | ✓ VERIFIED | 241 lines; LookPath, 5 s bound, coded hints |
| internal/interact/familysel.go | entitlement gate | ✓ VERIFIED | 153 lines; reads setting.json + creds, honest degradation |
| internal/hostsvc/headers_pat.go | mode-aware PAT source | ✓ VERIFIED | 214 lines; cache answer, re-mint once, scope refusal |
| internal/acpfront/auth.go | coding-plan arm | ✓ VERIFIED | full chain; AuthConfig seams; auth-mode record |
| internal/acpport/types.go, agent.go | Handlers vocabulary + SDK assembly | ✓ VERIFIED | SetSessionConfigOption delegation; SessionConfigOptionSelect via SDK |
| cmd/zcode-acp/main.go | production wiring | ✓ VERIFIED | creds/prober/gate/minter/mode-source wired; no new flags |
| internal/acpfront/codingplan_cycle_test.go | composed fake cycle | ✓ VERIFIED | 663 lines; 5 stages + ordering locks |
| test/realdesktop/codingplan_test.go | guarded real tier | ✓ VERIFIED | 662 lines; guard triple, header evidence contract |
| experiments/README.md | quota ledger | ✓ VERIFIED | EXP-CODINGPLAN-CYCLE-01 row PENDING (line 239) |
| experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl | scrubbed real-run capture | ⚠️ EXPECTED-ABSENT | born only from the human-gated run — the UAT item's deliverable, not a phase gap |

### Key Link Verification

`verify.key-links` reported 06-01's two links "Target not referenced in source": the verb greps the literal target path string inside the source file, which cannot hold for Go same-package links (no import statement exists between files of one package). Resolved manually at symbol level; every declared link verified.

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| interact/credstore.go | interact/cipherdecrypt.go | v2:os arm composes SecretToolKeyLoader; v1 loader implements the same KeyLoader | WIRED | same package; the loader seam is consumed in credstore.go's v2 arm; v1 loader implements KeyLoader (read first-hand) |
| interact/credstore.go | hostsvc/patminter.go | LoginToken(ctx,"zai") is mint step 1 | WIRED | patminter.go reads the credential-store accessor; ok=false → the coded login-required arm (TestPatMinterLoginRequired) |
| hostsvc/patminter.go | interact/responder.go | mode-aware runtime-headers source consumes Ensure/Invalidate/Cached | WIRED | headers_pat.go:55-62 declares Cached/Ensure/Invalidate over the minter (PatHeadersSourceDeps) |
| hostsvc/patminter.go | acpfront/auth.go | Ensure(ctx) once at authenticate via the PatEnsurer seam | WIRED | auth.go:96-100 (PatEnsurer over hostsvc.Pat); consumed at auth.go:289 |
| hostsvc/snapshot.go | supervisor/discover.go | supervisor.BuiltinRevision reuse + install.BuiltinFile | WIRED | snapshot.go:154 `supervisor.BuiltinRevision(install.BuiltinFile, rev)` |
| hostsvc/snapshot.go | acpfront/auth.go | BuildAccount supplies push params AND the want list | WIRED | auth.go:246 (BuildAccount call) + auth.go:253 (wantModelsFromSnapshot) |
| acpfront/configoptions.go | hostsvc/readback.go | dig-out discipline extends digCreateAvailable's pattern | WIRED | ParseSettings/typed dig helpers mirror the capture-verbatim pattern (read first-hand) |
| acpport/agent.go | acp-go-sdk v0.20.0 | SessionConfigOptionSelect assembly + initialize Options capability | WIRED | agent.go:194 (`Options: &acp.SessionOptionsCapabilities{}`), agent.go:250 (Select union — SDK injects the discriminant); tidy-diff gate pins v0.20.0 (CI green) |
| acpfront/front.go | acpfront/configoptions.go | NewSession captures + BuildConfigOptions(confirmed) | WIRED | front.go:302-314 (latestSettings → ParseSettings → BuildConfigOptions) |
| acpfront/front.go | zcode/envelope.go | MethodSessionSetModel carries both switch kinds | WIRED | front.go:542 `f.conn.Call(ctx, zcode.MethodSessionSetModel, params)` for model AND thought changes |
| acpport/agent.go | SDK SetSessionConfigOption types | response carries ConfigOptions through SDK Validate | WIRED | agent.go:461-464 delegation with acp.SetSessionConfigOptionRequest/Response |
| acpfront/auth.go | interact/keyringprobe.go + cipherdecrypt.go | probe runs the loader's LookupCommand argv, bounded, scrubbed env | WIRED | keyringprobe.go (LookupCommand source; NewKeyringProber wired with secretToolBin in main.go:300) |
| acpfront/auth.go | interact/responder.go (06-07) | recorded auth mode is the runtime-headers selector | WIRED | auth.go:53-82 (AuthMode/recordAuthMode); main.go:357 `Mode: front.AuthMode` |
| hostsvc/headers_pat.go | hostsvc/patminter.go | Cached/Ensure/Invalidate orchestration | WIRED | headers_pat.go:103-128,181 (read first-hand; TestPatHeadersSource arms) |
| interact/responder.go | testdata runtime-headers-answer.json (Phase 5) | api-key reply bytes stay identical after seam widening | WIRED | TestRuntimeHeadersSeamWidening asserts the exact byte shape (responder_test.go:771,789); green in CI |
| acpfront/codingplan_cycle_test.go | acpfront/auth_test.go | same harness composition driven through the WHOLE cycle | WIRED | TestCodingPlanFullCycleFake composes ComposeFront + the Creds/Pat/Probe seams; PASS under -race |
| test/realdesktop/codingplan_test.go | test/realdesktop/apicycle_test.go | guard triple + staged-shutdown shape copied verbatim | WIRED | both compile under -tags realdesktop (vet/build green this pass); guarded skip proven |
| experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl | experiments/scrub-capture.mjs | scrubber gate before any commit | WIRED (contract) | canonical scrub command in the test header; scrubber + node suite green (22/22 in CI); the capture itself awaits the human run |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| configoptions.go | model/thought option lists + currentValue | backend create/setModel reply settings snapshot (ParseSettings dig) | Yes — the reply IS the source; omission when unconfirmed (D-10) | ✓ FLOWING |
| headers_pat.go | apiKey/apiKeyID/accountScope | in-process PAT cache ← PatMinter official chain ← credential store ← keyring | Yes — production wiring (main.go:295-357); fake tier scripts the same surface | ✓ FLOWING |
| snapshot.go BuildAccount | providers/states overlay | the desktop's own bundled builtin table (install.BuiltinFile) + the entitlement gate over real desktop state files | Yes | ✓ FLOWING |
| familysel.go | entitlement decisions | setting.json providerFamilyConnectionSelections/modelProviderFamilyModes + store oauth token (read-only reads) | Yes | ✓ FLOWING |
| Static/hardcoded fallback hunt | — | no static return, hardcoded model/level list, or mock feeding any rendered value | — | ✓ no disconnection found |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full CI gate at HEAD | `mise run ci` (build+vet+lint+`go test -race -count=1 ./...`+node experiments) | exit 0; lint 0 issues; all 10 Go packages ok; experiments 22/22 | ✓ PASS |
| Composed full cycle (ordering/mint-once/re-emit/mid-turn ask) | `go test -race -count=1 -v -run TestCodingPlanFullCycleFake ./internal/acpfront/` | `--- PASS: TestCodingPlanFullCycleFake (0.12s)` | ✓ PASS |
| D-05 re-mint arms (exactly-once, scope refusal, ordering) | `go test -race -count=1 -v -run TestPatHeadersSource ./internal/hostsvc/` | 9 subtests run (fingerprint re-mint exactly once; second rejection coded; scope gate outranks fingerprint; api-key byte-compat; …) — package ok | ✓ PASS |
| Tagged real tier compiles + skips safely | `go vet/build -tags realdesktop ./test/realdesktop/`; `go test -tags realdesktop -run TestCodingPlanFullCycle ./test/realdesktop/` | vet/build green; `--- SKIP` with the manual-only message (0.00s, nothing touched) | ✓ PASS |
| Secret-scan gate on the tree | `bash experiments/scan-secrets.sh` | exit 0 | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared or conventional for this phase. The phase's probe-equivalent tiers are the golden-corpus replay and the scan-secrets gate — both covered above (corpus rows green inside the CI run; scan-secrets exit 0 re-run in isolation this pass). The guarded real-tier test (the D-12 probe carrier) is the human verification item below.

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AUTH-02 | 06-01, 06-02, 06-03, 06-05, 06-07, 06-08 | OAuth individual coding plan: exact-revision snapshot, requestCredentialCipher, PAT mint via api.z.ai, requestAuth — only official paths | ✓ SATISFIED (machinery; real-account field proof = the UAT item) | full chain implemented + corpus-locked + hygiene locks green; real run pending by design |
| AUTH-03 | 06-04, 06-05, 06-08 | `authenticate{methodId}` works for both advertised methods; account models appear after auth | ✓ SATISFIED (machinery; real-account confirmation rides the UAT) | dispatch + both arms green; want list mirrors the built snapshot; api-key field-proven in Phase 5 |
| AUTH-04 | 06-05 | Missing keyring fails with a clear error and does not hang | ✓ SATISFIED | proactive probe (D-02), coded actionable errors, 5 s bound, probe-before-push golden-locked |
| MODEL-01 | 06-04 | `session/new` returns configOptions built from the live provider registry after auth | ✓ SATISFIED (machinery; live-registry render = the UAT) | translator + capability advertisement + reply-confirmed store, all test-locked |
| MODEL-02 | 06-06 | set_config_option maps to setModel/setThoughtLevel, re-emits full list, honest currentValue | ✓ SATISFIED | validate-first handler + level-carry setModel + reply-built re-emit, corpus byte-locked |

Orphan check: REQUIREMENTS.md maps exactly AUTH-02/03/04 + MODEL-01/02 to Phase 6; every ID appears in at least one plan's frontmatter `requirements`; no orphaned requirements. (DOCS-01..03, QA-01..03, SESS-02/03 are Phase 7's — unchanged.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | none found | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any phase-modified production file; no empty-implementation stubs; no hardcoded data paths. Review findings WR-01/04/05 + IN-01..04 are dispositioned to Phase 7 in 06-REVIEW-DISPOSITION.md (advisory ledger items, not must-have failures; WR-02/03/06 fixed in-phase at 4dbe1aa/d5e8c22 — verified in git log and source) |

### Human Verification Required

### 1. THE REAL RUN — coding-plan full cycle on the real account (the phase bar, D-12/D-13)

**Test:** Desktop logged into the z.ai individual coding plan, gnome-keyring unlocked, `ZCODE_ACP_RAW_CAPTURE` pointed at a scratch path OUTSIDE the repo. Canonical command (verbatim from test/realdesktop/codingplan_test.go's header):
```
export ZCODE_ACP_RAW_CAPTURE=<scratch path outside this repo>
ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/
```
**Expected:** PASS — authenticate read-back proven from registry state (account models present, never the ack), the real PAT mint at api.z.ai, session/new configOptions rendered from the live registry, one real tool-less tiny turn with the runtime-headers ask answered (token+id+accountScope), staged shutdown with the zero-orphan proof. Then the evidence contract: scrub the raw capture into experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl, fill the ledger actuals + V-EXP id in experiments/README.md, scan-secrets green on the tree AND the staged diff, record A4 (thoughtLevel block) / A6 (clientVersion) / A7 (configOptions rendering) confirm-or-adjust notes, then confirm the evidence commit. A failed leg records the honest verdict with the scrubbed capture — the phase stays open (the 05-14 discipline).
**Why human:** spends real quota on the human's coding-plan account through the desktop keyring; D-12/D-13 forbid autonomous or CI execution by design. Every automated tier is proven this pass and the handoff is complete: guarded harness race-clean (cpAskRecorder mutex, full -race green), canonical command + evidence contract in the test header, ledger row PENDING, scrubber + scan-secrets green, tagged tier compiles and skips safely.

### Gaps Summary

No gaps. All 15 machine-verifiable must-haves verify with behavioral evidence: the full `mise run ci` gate is green at HEAD (build, vet, lint 0 issues, the entire Go suite under -race, the experiments suite 22/22), the composed fake full cycle and the D-05 re-mint arms pass under -race in dedicated runs, the corpus goldens byte-lock the wire contract, all three test-tier prohibitions carry wired enforcement, and the guarded real tier compiles and skips safely without the env var.

The single open item is the human-gated real-account run — the phase bar by explicit planner decision (D-12, 06-08 truth 3, human_verify_mode=end-of-phase). Its absence is the designed `human_needed` shape, not a gap: the harness, canonical command, evidence contract, and PENDING ledger row are complete, and the coordinator's phase context directs routing it to human verification. Until that run records its verdict (and the evidence commit lands), ROADMAP SC1's "on a real account" clause remains human-pending; the phase must not ship to Phase 7 as fully closed before the UAT records PASS (or an honest failure verdict per the 05-14 discipline).

---

_Verified: 2026-11-22T05:37:58Z_
_Verifier: Claude (gsd-verifier)_
