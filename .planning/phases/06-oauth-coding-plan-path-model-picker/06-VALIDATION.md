---
phase: "6"
slug: "oauth-coding-plan-path-model-picker"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-11-22"
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib, -race), golangci-lint strict (goswitch-style) |
| **Config file** | `.golangci.yml` (pinned), `mise.toml` toolchain |
| **Quick run command** | `go test -race -count=1 ./... 2>&1 \| tail -8` |
| **Full suite command** | `mise run ci` (lint + test + build + govulncheck) |
| **Estimated runtime** | ~50–70 seconds (full -race suite per Phase-5 actuals) |

---

## Sampling Rate

- **After every task commit:** Run `go test -race -count=1 ./internal/... 2>&1 | tail -8`
- **After every plan wave:** Run `mise run ci`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 70 seconds

---

## Per-Task Verification Map

Filled by the planner from the 8 PLAN files (revision 2026-11-22): all 22 task
commands are automated, verbatim below, each with its plan-stated failing
direction. T-06-SC (package legitimacy / zero new dependencies) applies to
every row — all eight plans are stdlib-only.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Fails When | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|------------|-------------|--------|
| 06-01/T1 | 06-01 | 1 | AUTH-02 | T-06-01/02/03 | store prefix dispatch, per-entry isolation, never-plaintext unknown-prefix skip, Bearer-strip accessors | unit -race (RED→GREEN) | `go test -race -count=1 -run 'TestCredStore' ./internal/interact/ 2>&1 \| tail -6` | any vector wrong (decrypt, fp24 mismatch not skipping, unknown prefix served plaintext, Bearer kept, missing-file not empty-nil) or build breaks | creates | ⬜ pending |
| 06-01/T2 | 06-01 | 1 | AUTH-02 | T-06-02 | enc:v1 fallback loader behind KeyLoader seam; coded malformed-shape failures; no exec | unit -race (RED→GREEN) | `go test -race -count=1 -run 'TestV1Fallback\|TestKeyLoader' ./internal/interact/ 2>&1 \| tail -6 && go test -race -count=1 ./internal/interact/... 2>&1 \| tail -8` | v1 vector wrong, Phase-5 loader locks regress, -race trips | extends (P5 files) | ⬜ pending |
| 06-02/T1 | 06-02 | 1 | AUTH-02 | T-06-05/06 | five-call official mint chain, byte-verbatim shapes, accountScope digest pinned | httptest -race (RED→GREEN) | `go test -race -count=1 -run 'TestPatMinterHappyPath\|TestAccountScopeDigest' ./internal/hostsvc/ 2>&1 \| tail -6` | a header/body/URL/pick/expiry assertion fails or the pinned digest mismatches | creates | ⬜ pending |
| 06-02/T2 | 06-02 | 1 | AUTH-02 | T-06-05 | edge arms: login-required, pick-unavailable, create-if-absent, disabled, biz-fail, 404-only location invalidation | httptest -race | `go test -race -count=1 -run 'TestPatMinter' ./internal/hostsvc/ 2>&1 \| tail -8` | a failure arm fabricates success, create body wrong, a permission error invalidates the location, a body leaks into error text | extends | ⬜ pending |
| 06-02/T3 | 06-02 | 1 | AUTH-02 | T-06-04/07 | single-flight, reactive refresh surface, 5 s transient backoff, memory-only hygiene lock | unit -race + source assertion | `go test -race -count=1 -run 'TestPatMinterSingleFlight\|TestPatMinterReactiveSurface\|TestPatMinterTransientBackoff\|TestPatMemoryOnly' ./internal/hostsvc/ 2>&1 \| tail -8 && go test -race -count=1 ./internal/hostsvc/... 2>&1 \| tail -8` | more than one mint per burst, refreshAt ignored, backoff floor missed, a write construct or String() found on Pat | extends | ⬜ pending |
| 06-03/T1 | 06-03 | 1 | AUTH-02 | T-06-08/09 | BuildAccount verbatim zhipu-account mirror, formula reuse path-sensitive, empty overlay untouched | unit -race (RED→GREEN) | `go test -race -count=1 -run 'TestAccountSnapshot' ./internal/hostsvc/ 2>&1 \| tail -6 && go test -race -count=1 -run 'TestSnapshotEnvelope' ./internal/hostsvc/ 2>&1 \| tail -4` | a non-zhipu rule leaks in or a model id is transformed, the formula is not path-sensitive, the Phase-5 envelope lock regresses | extends snapshot.go (exists); creates test file | ⬜ pending |
| 06-03/T2 | 06-03 | 1 | AUTH-02 | T-06-09 | entitlement-gate omission-only, zero-rules + malformed-table coded errors, PushEnvelope compat | unit -race | `go test -race -count=1 -run 'TestAccountSnapshot\|TestPushEnvelopeCompat' ./internal/hostsvc/ 2>&1 \| tail -6 && go test -race -count=1 ./internal/hostsvc/... 2>&1 \| tail -8` | a gate-rejected family still present, zero-rules or a malformed table answers success, PushEnvelope rejects the Snapshot | extends | ⬜ pending |
| 06-04/T1 | 06-04 | 1 | MODEL-01 | T-06-10/11 | settings→configOptions translator: verbatim mirror, per-model thought levels, D-10 omission arms | unit -race (RED→GREEN) | `go test -race -count=1 -run 'TestConfigOptionsTranslator' ./internal/acpfront/ 2>&1 \| tail -6` | a value renamed or filtered, fallback levels unused, a currentValue guessed, the dig-out panics on malformed input | creates | ⬜ pending |
| 06-04/T2 | 06-04 | 1 | MODEL-01, AUTH-03 | T-06-12 | options capability advertisement + SDK select assembly + Validate | unit -race | `go test -race -count=1 -run 'TestInitCapabilities\|TestNewSession' ./internal/acpport/ ./internal/acpfront/ 2>&1 \| tail -6` | the advertisement is missing or misshaped, an initialize honesty lock regresses, the select discriminant is hand-marshaled | extends (P4 files) | ⬜ pending |
| 06-04/T3 | 06-04 | 1 | MODEL-01 | T-06-10/11 | session/new returns configOptions from confirmed reply snapshots; per-session store bounded (WR-07) | harness -race | `go test -race -count=1 -run 'TestNewSession' ./internal/acpfront/ 2>&1 \| tail -6 && go test -race -count=1 ./internal/acpfront/ ./internal/acpport/ 2>&1 \| tail -8` | options missing from the response, the create-reply value wins over the confirmed follow-up, the store leaks across sessions | extends (P4 files) | ⬜ pending |
| 06-05/T1 | 06-05 | 2 | AUTH-04 | T-06-14/15 | proactive keyring probe: LookPath, bounded exec, coded three-arm failures, never hangs | unit -race (RED→GREEN) | `go test -race -count=1 -run 'TestKeyringProbe' ./internal/interact/ 2>&1 \| tail -6` | an arm is misclassified, the package name is absent from the error, argv is re-derived, the hang vector exceeds its bound | creates | ⬜ pending |
| 06-05/T2 | 06-05 | 2 | AUTH-02 | T-06-15 | production entitlement gate over read-only desktop state, honest degradation | unit -race (RED→GREEN) | `go test -race -count=1 -run 'TestFamilySelections' ./internal/interact/ 2>&1 \| tail -6` | an apiKey-mode family is entitled, a missing store passes, a missing or undecodable setting.json panics | creates | ⬜ pending |
| 06-05/T3 | 06-05 | 2 | AUTH-02/03/04 | T-06-13/15/16 | coding-plan authenticate chain end-to-end; probe-before-push ordering golden-locked; api-key regression intact | composed -race + golden corpus | `go test -race -count=1 -run 'TestCodingPlanAuth\|TestAuthenticate' ./internal/acpfront/ 2>&1 \| tail -8 && go test -race -count=1 -run 'TestGolden/coding-plan' ./internal/zcode/fake/ 2>&1 \| tail -4` | probe after push, an ack-only success, a mint before the verdict, the probe-fail golden shows a push frame, the api-key arm breaks | extends (P4/P5 files); creates 2 fixture units | ⬜ pending |
| 06-06/T1 | 06-06 | 3 | MODEL-02 | T-06-17 | set_config_option validate-first; -32602 names configId + allowed set; codes intact at the adapter | unit -race (RED→GREEN) | `go test -race -count=1 -run 'TestSetConfigOption' ./internal/acpfront/ ./internal/acpport/ 2>&1 \| tail -6` | an invalid arm succeeds or emits a backend frame, an error fails to name configId or the allowed set, a seam code degrades to -32603 | extends (P4 files); creates test file | ⬜ pending |
| 06-06/T2 | 06-06 | 3 | MODEL-02 | T-06-18/19 | setModel for both switch kinds with the level carried; reply-built full-list re-emit | unit -race | `go test -race -count=1 -run 'TestSetConfigOptionModelSwitch\|TestSetConfigOptionThoughtSwitch\|TestSetConfigOptionReEmit' ./internal/acpfront/ 2>&1 \| tail -6` | the level is dropped on a switch, a non-setModel mutation frame appears, the re-emit reflects the pre-ask snapshot, a refused setModel orphans the session | extends | ⬜ pending |
| 06-06/T3 | 06-06 | 3 | MODEL-02 | T-06-18 | corpus goldens byte-lock the setModel wire (options carry/omit, persist flag) | golden corpus | `go test -race -count=1 -run 'TestGolden/config-option' ./internal/zcode/fake/ 2>&1 \| tail -4` | a golden regresses, the payload drifts from the carry/omit rules, the driver fails to complete | extends golden_test.go (exists); creates 2 fixture units | ⬜ pending |
| 06-07/T1 | 06-07 | 4 | AUTH-02 | T-06-20/21 | seam widened with accountScope; api-key reply byte-compat; 64-hex rejection-field dig | unit -race + golden byte-compat | `go test -race -count=1 ./internal/interact/... 2>&1 \| tail -8 && go test -race -count=1 -run 'TestGolden/runtime-headers-answer' ./internal/zcode/fake/ 2>&1 \| tail -4` | the api-key reply bytes drift from the Phase-5 golden, accountScope missing when supplied, a non-64-hex value accepted | extends (P5 files) | ⬜ pending |
| 06-07/T2 | 06-07 | 4 | AUTH-02 | T-06-20/22/23 | mode-aware PAT source: cache answer, exactly-one re-mint, scope refusal, PAT-less coded arm | unit -race | `go test -race -count=1 -run 'TestPatHeadersSource' ./internal/hostsvc/ 2>&1 \| tail -8` | a stale token served on scope mismatch, two re-mints in one ask, a PAT-less success arm, a malformed field triggers a re-mint | creates | ⬜ pending |
| 06-07/T3 | 06-07 | 4 | AUTH-02 | T-06-21/23 | cmd wiring, no new flags (D-12); fingerprint re-ask corpus row | wiring tests + golden + full suite | `go test -race -count=1 ./cmd/zcode-acp/ 2>&1 \| tail -6 && go test -race -count=1 -run 'TestGolden/runtime-headers-pat' ./internal/zcode/fake/ 2>&1 \| tail -4 && go test -race -count=1 ./internal/... 2>&1 \| tail -8` | a seam unwired, a new user flag appears, the golden answers with a stale PAT, any internal package regresses | extends (P4/P5 files); creates 1 fixture unit | ⬜ pending |
| 06-08/T1 | 06-08 | 5 | AUTH-02/03 | T-06-26 | zero-spend composed full cycle: auth → configOptions → set_config_option → one scripted turn | composed -race | `go test -race -count=1 -run 'TestCodingPlanFullCycleFake' ./internal/acpfront/ 2>&1 \| tail -6` | any cycle stage fails (auth not read-back-proven, an entry missing, an unconfirmed currentValue, the ask unanswered or twice, an extra mint) | creates | ⬜ pending |
| 06-08/T2 | 06-08 | 5 | AUTH-02/03 | T-06-24/26 | guarded real-tier test: compiles under the tag, skips safely, evidence contract in header | vet+build (tagged realdesktop) | `go vet -tags realdesktop ./test/realdesktop/ 2>&1 \| tail -4 && go build -tags realdesktop ./test/realdesktop/ 2>&1 \| tail -4` | the tagged file fails vet or build, or the default untagged build breaks | creates | ⬜ pending |
| 06-08/T3 | 06-08 | 5 | AUTH-02/03 | T-06-24/25 | planned-spend ledger PENDING + pre-flight green; the real run is the human-gated UAT (never executor-autonomous, never CI) | grep gate + scan-secrets | `grep -q "EXP-CODINGPLAN-CYCLE-01" experiments/README.md && grep -qi "PENDING" experiments/README.md && bash experiments/scan-secrets.sh 2>&1 \| tail -3` | no EXP-CODINGPLAN-CYCLE-01 planned-spend row with PENDING status, or scan-secrets hits the tree | extends README (exists); capture created at run time | ⬜ pending |

*Task IDs synthesize `PLAN/Tn` (plans do not number tasks); Wave/Requirement/Threat
refs transcribed from PLAN frontmatter + threat registers. "creates" = the task
authors the test file first (RED) before its command runs; "P4/P5" = exists since
that phase (baseline verified 2026-11-22).*

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- Existing infrastructure covers all phase requirements (go test + fake app-server corpus from Phases 3–5; no new framework).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real coding-plan authenticate end-to-end (keyring decrypt + PAT mint + registry accepted + one real turn) | AUTH-02 | Real account + desktop keyring on the local machine; CI never touches it (D-05/D-12/D-13) | Guarded manual tier per Phase-5 AR precedent: desktop logged in, run wrapper on real install, ledger spend in experiments/README, scrubbed capture + V-EXP id |
| Keyring-absent fail-fast on the real machine (AUTH-04 probe arms) | AUTH-04 | Requires an actual headless env / real keyring state | Observe coded actionable error with no hang; research already probed both arms live (rc=1 fast) |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 22/22 task commands automated (map above); zero MISSING references
- [x] Sampling continuity: no 3 consecutive tasks without automated verify — every task carries an automated command
- [x] Wave 0 covers all MISSING references — no MISSING references exist; every task authors its test file (RED-first) before its command runs
- [x] No watch-mode flags — all runs use `-count=1`; the only full-suite gate is `mise run ci`
- [x] Feedback latency < 70s — task commands are package-scoped tail-ed runs; the full suite (~50–70 s, Phase-5 actuals) runs only at wave gates
- [ ] `nyquist_compliant: true` set in frontmatter — set by /gsd:validate-phase §6 after execution (planning leaves `status: draft` / `nyquist_compliant: false`)

**Approval:** pending (/gsd:validate-phase §6)
