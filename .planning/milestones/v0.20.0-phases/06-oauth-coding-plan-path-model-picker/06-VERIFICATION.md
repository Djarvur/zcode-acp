---
phase: 06-oauth-coding-plan-path-model-picker
verified: 2026-11-22T09:13:37Z
status: passed
score: 16/16 must-haves verified
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
  - "docs/protocol/app-server-dialect.md"
  - "docs/protocol/install-discovery.md"
  - "experiments/README.md"
  - "experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl"
  - "experiments/scan-secrets.sh"
  - "internal/acpfront/auth.go"
  - "internal/acpfront/auth_test.go"
  - "internal/acpfront/codingplan_cycle_test.go"
  - "internal/acpfront/codingplan_probe_test.go"
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
  - "internal/hostsvc/snapshot_builtinfile_test.go"
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
  - "internal/supervisor/spawn.go"
  - "internal/supervisor/spawn_test.go"
  - "internal/zcode/fake/golden_test.go"
  - "internal/zcode/fake/testdata/golden/coding-plan-auth-cycle.golden"
  - "internal/zcode/fake/testdata/golden/coding-plan-probe-fail.golden"
  - "internal/zcode/fake/testdata/golden/coding-plan-probe-setmodel-refused.golden"
  - "internal/zcode/fake/testdata/golden/coding-plan-probe-setmodel.golden"
  - "internal/zcode/fake/testdata/golden/config-option-set-model.golden"
  - "internal/zcode/fake/testdata/golden/config-option-thought.golden"
  - "internal/zcode/fake/testdata/golden/official-mcp-auth-decline.golden"
  - "internal/zcode/fake/testdata/golden/runtime-headers-pat.golden"
  - "internal/zcode/fake/testdata/golden/unknown-host-method.golden"
  - "internal/zcode/fake/testdata/scenarios/coding-plan-auth-cycle.json"
  - "internal/zcode/fake/testdata/scenarios/coding-plan-probe-fail.json"
  - "internal/zcode/fake/testdata/scenarios/coding-plan-probe-setmodel-refused.json"
  - "internal/zcode/fake/testdata/scenarios/coding-plan-probe-setmodel.json"
  - "internal/zcode/fake/testdata/scenarios/config-option-set-model.json"
  - "internal/zcode/fake/testdata/scenarios/config-option-thought.json"
  - "internal/zcode/fake/testdata/scenarios/official-mcp-auth-decline.json"
  - "internal/zcode/fake/testdata/scenarios/runtime-headers-pat.json"
  - "internal/zcode/fake/testdata/scenarios/unknown-host-method.json"
  - "test/realdesktop/apicycle_test.go"
  - "test/realdesktop/codingplan_test.go"
covered_digest: "v4:sha256:6cb41c6b902370bce2ccb48eaddcdaa6939623b649544a8fb94bf8660d81597e"
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 15/16
  gaps_closed:
    - "SC1 real-account bar (D-12): the official chain completes ON A REAL ACCOUNT and subsequent turns run — the human-gated end-of-phase UAT (TestCodingPlanFullCycle, EXP-CODINGPLAN-CYCLE-01) recorded PASSED at attempt 8 (2026-11-22 12:02 local) with the scrubbed capture committed, the ledger actuals + spend honesty recorded, and A4/A6/A7 confirmed in the ledger"
  gaps_remaining: []
  regressions: []
---

# Phase 6: OAuth Coding-Plan Path & Model Picker Verification Report

**Phase Goal:** Users on an individual coding plan can authenticate through the official OAuth chain end-to-end, and any client can see and switch model and thought level from the live provider registry.
**Verified:** 2026-11-22T09:13:37Z
**Status:** passed
**Re-verification:** Yes — after the human-gated UAT PASSED (attempt 8) and three documented gap-cycle fix rounds

**MVP-mode note:** ROADMAP marks this phase `mode: mvp`; the goal is outcome-phrased and fails the `user-story.validate` guard (prior-pass finding, unchanged — same shape as Phases 4/5). Per the milestone's uniform precedent, the User Flow table is derived from the roadmap contract (the 5 success criteria), and the per-plan user stories in COVERAGE.md's MVP Framing Note are honored as written.

## Re-Verification Scope

Prior report: 15/16 verified, status human_needed — the single open item was the human-gated real-account run (D-12/D-13; autonomous/CI execution forbidden by design). This pass re-verifies that item at full depth against the CURRENT tree (the three gap-cycle fix rounds are part of the phase's source truth), and regression-checks the 15 previously verified truths.

**Gap-cycle commits verified real** (`verify.commits`: 14/14 valid, none invalid): round-A honest MCP-auth decline (`8f9d1a4`/`24063cf`/`a9000b6`), round-B revision revert + probe setModel poke + eager PAT mint (`5d2a200`/`1b1e565`/`baf3a75`/`cce3a72`/`cbb1709`/`a624fe5`), round-C access-completeness revert (`f144fa6`/`ebd8232`/`86a5e60`/`c979be3`) + the REAL fix — paired provider-config env paths in the spawn (`25b7395`) — plus the docs corrections (`b4bd308`/`aa2457d`) and the evidence commit (`18a8bf0`). All three rounds are present first-hand in the current tree:

- Round-A: `internal/interact/responder.go:543-565` — `requestOfficialMcpAuthHeaders` answered with the structured honest decline `{ok:false, reason:"official_auth_unavailable"|"official_mcp_origin_untrusted"}`, never an error envelope (the attempt-1 stall root cause); `official-mcp-auth-decline` corpus golden byte-locks it.
- Round-B: `internal/acpfront/auth.go:273` (eager `Pat.Ensure(ctx)` before the probe), `auth.go:536` (`ProbeWantModel` — the deterministic `-Flash` poke target), `auth.go:625-628` (the poke fires `session/setModel` after create-ready so the account provider materializes and publishes); `coding-plan-probe-setmodel` + `-refused` corpus rows byte-lock both poke arms.
- Round-C: `internal/supervisor/spawn.go:36-37,160` — the child env passes the personal provider config path PAIRED with the builtin one (`resolveNodeProviderRuntimePaths` throws on the unpaired form; the degraded worker never publishes account models); `spawn_test.go` locks it.

## User Flow Coverage (MVP)

| # | Flow step (from the goal) | Expected | Evidence in codebase | Status |
|---|---------------------------|----------|----------------------|--------|
| 1 | Coding-plan user runs `authenticate{zai-individual-coding-plan}` | The official chain runs: keyring probe FIRST → desktop login read → account overlay push → registry read-back verdict → PAT mint exactly once → the probe pokes the account model so it materializes | internal/acpfront/auth.go (probe→login→BuildAccount→want list→watcher-before-push→push→refresh→eager mint→probe-create→poke→Await→Ensure); corpus goldens coding-plan-auth-cycle + probe-fail + probe-setmodel arms green; the REAL RUN completed this chain on the real account (attempt 8 PASS) | ✓ VERIFIED |
| 2 | The chain's verdict and the PAT | Verdict is registry state, never the ack; PAT minted via the official api.z.ai chain with accountScope | auth.go Await is the only success path, mint strictly after; patminter.go official 6-step chain; THE REAL CAPTURE: line 35 `state.updated` patch carries `model.available` with `account:zai-individual-coding-plan/GLM-5.3-Flash` (read-back surface), line 82 the served requestAuth {accountScope 64-hex, apiKeyId, apiKey [REDACTED-JWT]} | ✓ VERIFIED |
| 3 | No keyring machine (headless/CI) | Fail fast with a coded actionable error naming the missing piece; never hangs, no push traffic | keyringprobe.go (LookPath → libsecret-tools error; 5 s bounded ctx; autolaunch + log-in-via-desktop hints); probe-fail golden proves NO provider frame after a probe failure | ✓ VERIFIED |
| 4 | Any client: `session/new` | Returns `configOptions` (model + thought level) built verbatim from the backend's confirmed settings snapshot; server-confirmed currentValue only | configoptions.go (ParseSettings/BuildConfigOptions; omission over fabrication — locked OQ2); front.go reply IS the confirmation; agent.go SDK select assembly; THE REAL RUN: configOptions rendered live from the bound account model (test-asserted on the ACP reply; ledger records model + thought_level with currentValue "high") | ✓ VERIFIED |
| 5 | Client switches model / thought level | `session/set_config_option` validates first (-32602 naming configId + allowed set), maps to `session/setModel` with the level carried (OQ1), re-emits the full reply-built list | front.go validate-first → MethodSessionSetModel → store update → rebuild; config-option corpus rows byte-lock the wire; TestSetConfigOption* green in CI | ✓ VERIFIED |
| 6 | Subsequent turns run | The mid-turn runtime-headers ask is answered from the in-process PAT cache: requestAuth {apiKey, apiKeyId, accountScope} | headers_pat.go (cache answer / exactly-once re-mint / scope refusal); THE REAL CAPTURE: exactly ONE in-turn ask (line 81, reason model-request) answered `headersApplied:true` with the three-field requestAuth; the turn completed (turn-completed surface, toolCallCount 0, modelRequestCount 1) | ✓ VERIFIED |

## Goal Achievement

### Observable Truths

Consolidated set: ROADMAP SC1–SC5 + the eight PLAN frontmatter clusters, with the three bespoke prohibitions (COVERAGE.md's kept set) as explicit rows. Truth 2 — the prior pass's single open item — is now verified by the human-gated UAT's recorded PASS plus the committed capture evidence; truths 1, 10, 11, 14, 16 re-verified at regression depth after the gap-cycle commits (they touched auth.go, responder.go, snapshot.go, spawn.go, main.go, front.go).

| #  | Truth | Status | Evidence |
|----|-------|--------|----------|
| 1  | SC1 machinery: `authenticate{zai-individual-coding-plan}` runs the composed official chain — probe-first, login read, BuildAccount overlay, watcher-before-push, registry read-back verdict (never the ack), eager PAT mint exactly once, the probe poke materializes the account provider | ✓ VERIFIED | Chain read first-hand at HEAD (auth.go incl. the round-B poke/mint); corpus rows green in CI (fake corpus package ok 12.3s incl. the new probe-setmodel arms); full `mise run ci` exit 0 at HEAD |
| 2  | SC1 real-account bar (D-12): the chain completes ON A REAL ACCOUNT and subsequent turns run | ✓ VERIFIED | THE HUMAN-GATED UAT: PASSED (2026-11-22 12:02 local, attempt 8, 15.2 s) — recorded in 06-UAT.md and the ledger. Capture `experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl` (136 lines, scrubbed, scan-secrets exit 0, header builtinRevision `zcode-builtin:44:e9c0674…` matching the pushed active-cache hash): read-back proven by registry state (line 35 `state.updated` patch carrying the account models — never the ack), PAT mint-once served mid-turn (line 81/82: one ask → `headersApplied:true` + requestAuth {accountScope, apiKey [REDACTED-JWT], apiKeyId}), configOptions rendered live from the bound account model (session/new bound `account:zai-individual-coding-plan/GLM-5.3-Flash`, reasoning `high`), ONE real tool-less turn completed (turn-completed surface, `stopReason:"stop"`, toolCallCount 0, modelRequestCount 1), staged shutdown. A4/A6/A7 CONFIRMED in the ledger |
| 3  | SC2: both advertised authMethods authenticate; account models appear after auth | ✓ VERIFIED | TestAuthenticate (api-key arm untouched) + TestCodingPlanAuth arms green in CI at HEAD; want list mirrors the BUILT snapshot; account-model appearance now field-proven on the real account by the attempt-8 read-back |
| 4  | SC3: keyring-less machine fails fast with a coded actionable error, never hangs | ✓ VERIFIED | TestKeyringProbe arms green in CI; probe-fail golden fatals on any provider frame after a probe failure (CI green) |
| 5  | SC4: `session/new` returns configOptions covering model + thought level, built from the live provider registry | ✓ VERIFIED | TestNewSessionConfigOptions + translator vectors green in CI; no hardcoded model/level lists in acpfront (prior-pass grep, files unchanged); real-run configOptions rendered live (ledger + passing test assertion) |
| 6  | SC5: config-option change maps to setModel on the wire, re-emits the full list, currentValue always real | ✓ VERIFIED | TestSetConfigOption* green in CI; corpus rows byte-lock the setModel payload incl. options.reasoningLevel; the reply rebuilds the list |
| 7  | 06-01: credential store reader — prefix dispatch, unknown-enc never passes as plaintext, isolation, missing file = empty+nil, v1 fallback behind the same KeyLoader seam | ✓ VERIFIED | Suite green in CI at HEAD (interact package ok); files unchanged since prior pass |
| 8  | 06-02: PAT minter — official 6-step api.z.ai chain verbatim, accountScope digest, cache + single-flight, reactive surface, bounded HTTPS | ✓ VERIFIED | hostsvc package green in CI (-race) at HEAD; 15 s client timeout + explicit transport unchanged |
| 9  | Prohibition (06-02, test-tier): the PAT, login token, or key material is NEVER persisted to disk or any store | ✓ VERIFIED (wired enforcement) | TestPatMemoryOnly AST-scans patminter.go for write-intent calls — green in CI at HEAD; the real run's token material never recorded ([REDACTED-JWT] in the capture; scan-secrets exit 0) |
| 10 | 06-03: BuildAccount produces the NON-empty account overlay — zhipu-account filter, verbatim builtin-table mirror, entitlement gate, exact-revision formula reused, malformed table = coded error | ✓ VERIFIED (re-verified at regression depth) | Round-B/C commits touched snapshot.go; the revision was REVERTED to the discovery-recorded builtin revision (1b1e565) and the access-completeness propagation REVERTED (c979be3 — the wire schema rejects accountType/mode); `snapshot_builtinfile_test.go` (new, 125 lines) locks the builders on the recorded builtin file revision; suite green in CI |
| 11 | 06-05: composed authenticate arm + proactive probe (D-02) + entitlement gate + api-key arm behaviorally untouched | ✓ VERIFIED (re-verified at regression depth) | Round-B commits widened auth.go (probe poke + eager mint); TestCodingPlanAuth arms + the new TestCodingPlanProbe* suite green under -race in a dedicated run this pass; api-key corpus rows + locks stayed green (CI exit 0) |
| 12 | Prohibition (06-05, test-tier): the provider/updateAccountConfig ack is NEVER treated as coding-plan auth success — the verdict is exclusively the registry read-back | ✓ VERIFIED (wired enforcement) | auth.go Await is the ONLY success path, mint strictly after; probe-fail golden's transcript assertion fatals on any provider frame; THE REAL RUN honored it (verdict from the line-35 registry patch, capture records the ack `provider/updateAccountConfig` exactly once as receipt) |
| 13 | 06-06: set_config_option validates FIRST, model change carries the current level, thought-level rides the SAME setModel path (locked OQ1), re-emits the reply-built list | ✓ VERIFIED | TestSetConfigOption* arms green in CI; corpus goldens byte-lock both switch payloads; no new method constant |
| 14 | 06-07: mode-aware runtime-headers serving — accountScope rides the widened seam, api-key path byte-compatibly identical, D-05 re-mint exactly once per ask, scope confusion refused, cmd wiring with NO new flags | ✓ VERIFIED (re-verified at regression depth) | Round-A touched responder.go (the MCP-auth decline row only — the runtime-headers path untouched by it); TestPatHeadersSource arms green in CI (-race); cmd flags unchanged (spawn env change is the supervised child's env, not a new flag); round-C spawn.go wiring locked by spawn_test.go |
| 15 | Prohibition (06-07, test-tier): the failure-shaped runtime-headers success (headersApplied:false or a PAT-less requestAuth) is NEVER emitted — failures are coded error envelopes | ✓ VERIFIED (wired enforcement) | responder.go hardcodes `"headersApplied": true`; headers_pat.go answers coded errors on the failure arms; tests assert the three-field success SHAPE and error envelopes; green in CI — and the real capture's answer carries the full three-field shape |
| 16 | 06-08: zero-spend composed fake full cycle + guarded real-tier sibling + evidence contract | ✓ VERIFIED | TestCodingPlanFullCycleFake PASS under -race (dedicated run this pass, with the Probe arms); tagged vet green; guarded skip proven this pass (`ok …/test/realdesktop 0.003s` without the env var — nothing touched); scan-secrets exit 0 at HEAD; the evidence contract CLOSED: capture committed (the PASS run), ledger actuals + V-EXP id + full gap-cycle spend honesty (attempts 1–3 zero-spend; attempts 4–8's eager-mint re-runs recorded), human confirmation preceding the evidence commit (UAT updated 12:10, commit `18a8bf0` follows the 12:02 PASS) |

**Score:** 16/16 truths verified (0 present, behavior-unverified)

### Deferred Items

None — no open item of this phase's contract is addressed in a later milestone phase. (Phase 7's scope — DOCS-01..03, QA-01..03, SESS-02/03 — is unchanged; A8's Orca-side rendering risk remains a Phase-7 live-E2E item by its own COVERAGE.md disposition, and was already advisory in the prior pass.)

### Advisory (New Scope, Unevidenced)

Re-verification ran; all new-scope findings this pass carry deterministic evidence and are recorded in Anti-Patterns/Findings below. None unevidenced.

### Required Artifacts

All 8 plans declare `must_haves.artifacts` (06-01..06-07 as bare strings — zero checkable units for the verb, supplemented by manual three-level verification; 06-08 as `{path: ...}` objects — `verify.artifacts` reported them present at HEAD). Every declared artifact exists, is substantive, and is wired.

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| internal/interact/credstore.go | store reader + accessors | ✓ VERIFIED | unchanged since prior pass; suite green in CI |
| internal/interact/cipherdecrypt.go | v1 fallback behind KeyLoader | ✓ VERIFIED | unchanged since prior pass |
| internal/hostsvc/patminter.go | official chain + cache | ✓ VERIFIED | unchanged since prior pass |
| internal/hostsvc/snapshot.go | BuildAccount overlay | ✓ VERIFIED | re-verified after round-B/C reverts; builtinfile revision lock green |
| internal/acpfront/configoptions.go | translator | ✓ VERIFIED | unchanged since prior pass |
| internal/interact/keyringprobe.go | proactive probe | ✓ VERIFIED | unchanged since prior pass |
| internal/interact/familysel.go | entitlement gate | ✓ VERIFIED | unchanged since prior pass |
| internal/hostsvc/headers_pat.go | mode-aware PAT source | ✓ VERIFIED | unchanged since prior pass |
| internal/acpfront/auth.go | coding-plan arm | ✓ VERIFIED | re-verified with the round-B poke + eager mint in the tree |
| internal/acpport/types.go, agent.go | Handlers vocabulary + SDK assembly | ✓ VERIFIED | unchanged since prior pass |
| cmd/zcode-acp/main.go | production wiring | ✓ VERIFIED | re-verified after the revision-revert wiring change; no new flags |
| internal/supervisor/spawn.go | (NEW, round-C) paired provider-config env paths | ✓ VERIFIED | the REAL fix; spawn_test.go locks the pairing |
| internal/acpfront/codingplan_cycle_test.go | composed fake cycle | ✓ VERIFIED | green under -race in a dedicated run this pass |
| internal/acpfront/codingplan_probe_test.go | (NEW, round-B) probe poke/mint arms | ✓ VERIFIED | green under -race in a dedicated run this pass |
| test/realdesktop/codingplan_test.go | guarded real tier | ✓ VERIFIED | vet green under the tag; guarded skip proven this pass (ok 0.003s, no env var) |
| experiments/README.md | quota ledger | ✓ VERIFIED | EXP-CODINGPLAN-CYCLE-01 row CLOSED: PASSED attempt 8 + full gap-cycle spend honesty |
| experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl | scrubbed real-run capture | ✓ VERIFIED | 136 lines committed at HEAD (18a8bf0); scrubbed (uuid placeholders, [REDACTED-JWT], tilde-relative paths); scan-secrets exit 0 |

### Key Link Verification

`verify.key-links` continues to report the 06-01 same-package links as "Target not referenced in source" (the verb greps literal target path strings — impossible for Go same-package links); resolved manually at symbol level in the prior pass, unchanged files. The gap-cycle links verified this pass:

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| interact/responder.go | (round-A) requestOfficialMcpAuthHeaders honest decline | structured `{ok:false, reason}` result, never an error envelope | WIRED | responder.go:543-565; `official-mcp-auth-decline` corpus golden byte-locks it; field-proven in the real run (attempt-2 capture: the session materialized past the ask) |
| acpfront/auth.go | hostsvc Pat eager mint + acpfront ProbeWantModel poke (round-B) | `Pat.Ensure(ctx)` before the probe; poke fires session/setModel on the want model | WIRED | auth.go:273 (eager mint), auth.go:536/625-628 (poke); codingplan_probe_test.go + the probe-setmodel corpus arms green under -race |
| supervisor/spawn.go | worker's provider runtime paths (round-C) | personal provider config path PAIRED with the builtin path in the child env | WIRED | spawn.go:36-37,160; spawn_test.go locks the pairing; field-proven — the account models published only after this fix (attempt 8 PASS) |
| experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl | experiments/scrub-capture.mjs | scrubber gate before the commit | WIRED | capture committed scrubbed (uuid placeholders + [REDACTED-JWT] present, no raw material); scan-secrets exit 0 at HEAD |

All previously verified links (credstore→cipherdecrypt, credstore→patminter, patminter→responder/headers_pat, patminter→auth, snapshot→supervisor, snapshot→auth, configoptions→readback, agent→SDK, front→configoptions/envelope, auth→keyringprobe/responder, cycle-test→auth-test, realdesktop→apicycle) — files unchanged or regression-green in CI; carried forward as WIRED.

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|--------------|--------|--------------------|--------|
| configoptions.go | model/thought option lists + currentValue | backend create/setModel reply settings snapshot | Yes — reply-confirmed; real-run proof: rendered from the live bound account model | ✓ FLOWING |
| headers_pat.go | apiKey/apiKeyId/accountScope | in-process PAT cache ← official chain ← keyring | Yes — REAL-RUN PROVEN: the served requestAuth rode the real mint (accountScope 64-hex, apiKeyId) | ✓ FLOWING |
| snapshot.go BuildAccount | providers/states overlay | the desktop's bundled builtin table + entitlement gate | Yes — real-run proof: the pushed overlay's account models appeared in the registry (line 35) | ✓ FLOWING |
| spawn.go child env | provider runtime paths | discovery (builtin + personal config paths, paired) | Yes — REAL-RUN PROVEN: the worker registry ran complete only in the paired form | ✓ FLOWING |
| Static/hardcoded fallback hunt | — | no static return, hardcoded model/level list, or mock feeding any rendered value | — | ✓ no disconnection found |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full CI gate at HEAD (incl. all gap-cycle commits) | `mise run ci` (build+vet+lint+`go test -race -count=1 ./...`+node experiments) | exit 0; lint 0 issues; all 10 Go packages ok (acpfront 29.9s, fake corpus 12.3s); experiments 22/22 | ✓ PASS |
| Composed full cycle + round-B probe arms + auth arms | `go test -race -count=1 -run 'TestCodingPlanFullCycleFake|TestCodingPlanProbe|TestCodingPlanAuth' ./internal/acpfront/` | package ok (2.65s) under -race | ✓ PASS |
| THE REAL RUN (human-gated UAT, D-12/D-13) | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/` (desktop logged in, keyring unlocked, raw capture outside the repo) | PASSED 2026-11-22 12:02 local, attempt 8, 15.2 s — recorded in 06-UAT.md + the ledger; scrubbed capture committed | ✓ PASS (human-gated; recorded verdict + committed evidence verified this pass) |
| Tagged real tier compiles + skips safely | `go vet -tags realdesktop ./test/realdesktop/`; `go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/` | vet exit 0; `ok … 0.003s` (guarded skip, nothing touched) | ✓ PASS |
| Secret-scan gate on the tree | `bash experiments/scan-secrets.sh` | exit 0 | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared or conventional for this phase. The phase's probe-equivalent tiers are the golden-corpus replay (green inside the CI run, now including the round-A/B rows: official-mcp-auth-decline, coding-plan-probe-setmodel, coding-plan-probe-setmodel-refused), the scan-secrets gate (exit 0 re-run this pass), and the human-gated real tier (the D-12 probe carrier — PASSED, recorded).

### Real-Run Evidence Detail (the closed gap)

The committed capture (`experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl`, the PASS run at HEAD) substantiates each claimed leg first-hand:

| UAT claim | Capture evidence |
|-----------|------------------|
| authenticate read-back proven (never the ack) | line 35: `state.updated` patch `model.available` carrying `account:zai-individual-coding-plan/GLM-5.3-Flash` + providerLabel "Z.AI Individual Co…" — registry state; the ack (`provider/updateAccountConfig`, one occurrence) is receipt only; header revision matches the pushed active-cache hash `zcode-builtin:44:e9c0674…` |
| Real PAT mint-once via api.z.ai, served mid-turn | line 81/82: exactly ONE in-turn `interaction/requestProviderRuntimeHeaders` ask (reason model-request) answered `headersApplied:true` with requestAuth {accountScope 64-hex, apiKey [REDACTED-JWT], apiKeyId 64-hex} — the mint chain succeeded upstream (A6 CONFIRMED) |
| Live configOptions | ACP-side reply (test-asserted; not on the app-server wire this capture records): session/new bound the account model, reasoning `high`; the registry-side thoughtLevel block IS in the capture (line 35: `thoughtLevel.available = [low, high, max]`) — A4 CONFIRMED; A7 CONFIRMED-SHAPE (currentValue carried the confirmed binding) |
| ONE tool-less tiny turn (D-13 cap) | turn-completed surface (computer-use/operation-event, turnId turn_<uuid-11>), `stopReason:"stop"`, `toolCallCount:0`, `modelRequestCount:1`; the only post-completion wire activity is a benign runtime-headers re-ask (line 135, same turnId, 50 ms AFTER turn-completed — outside the in-turn single-ask claim; its answer falls outside the recording window) — NOT a second turn |
| Staged shutdown | the guarded test's staged shutdown + zero-orphan proof ran in the PASS (test-asserted at runtime; the ledger records it) |
| Evidence contract | capture scrubbed (uuid placeholders, [REDACTED-JWT], tilde-relative paths); scan-secrets exit 0 at HEAD; ledger actuals + V-EXP id + spend honesty recorded; UAT updated after the PASS; evidence commit `18a8bf0` |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AUTH-02 | 06-01, 06-02, 06-03, 06-05, 06-07, 06-08 | OAuth individual coding plan: exact-revision snapshot, requestCredentialCipher, PAT mint via api.z.ai, requestAuth — only official paths | ✓ SATISFIED (field-proven) | full chain implemented + corpus-locked; the REAL RUN (attempt 8 PASS) completes the official chain end-to-end on the real account — mint, read-back, served requestAuth |
| AUTH-03 | 06-04, 06-05, 06-08 | `authenticate{methodId}` works for both advertised methods; account models appear after auth | ✓ SATISFIED (field-proven) | both arms green; account-model appearance field-proven by the attempt-8 registry read-back; api-key field-proven in Phase 5 |
| AUTH-04 | 06-05 | Missing keyring fails with a clear error and does not hang | ✓ SATISFIED | proactive probe, coded actionable errors, 5 s bound, probe-before-push golden-locked (A3's locked-keyring prompting was contained by the 5 s bound per the bounded-ctx design; the run machine was unlocked) |
| MODEL-01 | 06-04 | `session/new` returns configOptions built from the live provider registry after auth | ✓ SATISFIED (field-proven) | translator + reply-confirmed store test-locked; real-run configOptions rendered live from the bound account model |
| MODEL-02 | 06-06 | set_config_option maps to setModel/setThoughtLevel, re-emits full list, honest currentValue | ✓ SATISFIED | validate-first handler + level-carry setModel + reply-built re-emit, corpus byte-locked |

Orphan check: REQUIREMENTS.md maps exactly AUTH-02/03/04 + MODEL-01/02 to Phase 6; every ID appears in at least one plan's frontmatter `requirements`; no orphaned requirements. (DOCS-01..03, QA-01..03, SESS-02/03 are Phase 7's — unchanged.)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| COVERAGE.md (phase dir) | dispositions table (A4/A6/A7 rows) | the flagged-assumption disposition column still carries the pre-run text ("the real run confirms") — the A4/A6/A7 outcomes are recorded in the ledger (experiments/README.md attempt-8 bullet: CONFIRMED / CONFIRMED / CONFIRMED-SHAPE) but the second documentation sink named by 06-08 Task 3's action text and the probe-ledger contract was never updated (file untouched since creation `3742028`) | ⚠️ Warning | none on the must-haves: 06-08 truth 5's named sink is the ledger (satisfied); truth 4's evidence list does not include COVERAGE.md. Recommended follow-up: one-line doc sync mirroring the ledger's A4/A6/A7 outcomes into COVERAGE.md's disposition column |
| — | — | none found in phase-modified production files | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers in any gap-cycle-changed production file (grepped this pass); no empty-implementation stubs (flagged `return []…` sites are legitimate argv/method-def construction, read first-hand); full CI green at HEAD |

### Human Verification Required

None — the single human-gated item (THE REAL RUN, D-12/D-13) recorded its verdict: **PASSED** (2026-11-22 12:02 local, attempt 8), with the scrubbed capture committed and the evidence contract executed under human control. The phase's human checkpoint is closed.

### Gaps Summary

No gaps. The prior pass's single open item — the human-gated real-account run — is closed with its recorded PASS and first-hand capture evidence: the registry read-back proved the account models (never the ack), the real PAT minted once and served the mid-turn ask with the three-field requestAuth shape, configOptions rendered live from the bound account model, one real tool-less turn completed with zero tool frames, and the staged shutdown ran — all on the official chain, spending within the D-13 cap, with full gap-cycle spend honesty in the ledger.

The three gap-cycle fix rounds are part of the phase's source truth and re-verified at regression depth: the round-A honest MCP-auth decline (the attempt-1 stall fix, corpus-locked), the round-B probe poke + eager mint + revision revert (corpus-locked, -race green), and the round-C revert + paired spawn env paths (the REAL fix, spawn_test-locked, field-proven by the attempt-8 PASS). The full CI gate is green at HEAD (build, vet, lint 0 issues, the entire Go suite under -race, the experiments suite 22/22), the composed fake cycle and the round-B probe arms pass under -race in a dedicated run, all three test-tier prohibitions carry wired enforcement, and the guarded real tier compiles and skips safely without the env var.

One non-blocking warning is recorded: COVERAGE.md's flagged-assumption disposition column was not synced with the A4/A6/A7 outcomes (they live in the ledger, which is the must-have's named sink). This is a one-line documentation follow-up for the orchestrator, not a gap: no must-have truth, artifact, or link depends on it.

**Phase goal achieved.** Users on an individual coding plan can authenticate through the official OAuth chain end-to-end (field-proven on the real account), and any client can see and switch model and thought level from the live provider registry. The phase is ready to ship to Phase 7.

---

_Verified: 2026-11-22T09:13:37Z_
_Verifier: Claude (gsd-verifier)_
