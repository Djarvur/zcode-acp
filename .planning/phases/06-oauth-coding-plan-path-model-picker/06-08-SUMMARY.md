---
phase: 06-oauth-coding-plan-path-model-picker
plan: 08
subsystem: auth
tags: [coding-plan, full-cycle, composed-fake-tier, guarded-real-tier, exp-codingplan-cycle-01, quota-ledger, human-gated-uat, d12-d13]

# Dependency graph
requires:
  - phase: 06-05
    provides: the composed coding-plan authenticate chain (probe -> cred -> overlay push -> read-back verdict -> mint-once), the AuthConfig Creds/Pat/Probe/Entitle seams, and the cpFix* fixture vocabulary the cycle reuses
  - phase: 06-06
    provides: the validate-first set_config_option handler with the level-carry setModel mapping and the reply-built re-emit (BuildConfigOptions/ParseSettings the cycle asserts through)
  - phase: 06-07
    provides: the mode-aware PatHeadersSource (the mid-turn ask answered from the same cache the front minted into) and the production cmd wiring shape the guarded tier mirrors
  - phase: 05-08/05-09
    provides: the composed full-cycle test structure (fullcycle_test.go) and the guarded real-tier sibling discipline (apicycle_test.go: guard triple, header contract, staged shutdown, zero-orphan proof)
provides:
  - internal/acpfront/codingplan_cycle_test.go — TestCodingPlanFullCycleFake: the zero-spend composed cycle proving every phase capability end-to-end at the fake tier (authenticate{coding-plan} with mint-once -> session/new configOptions with both reply-confirmed entries -> set_config_option model switch re-emitting the reply-built list -> one tool-less streamed turn with the runtime-headers ask answered from the PAT cache mid-turn), with the ordering locks on the tee + transcript
  - test/realdesktop/codingplan_test.go — TestCodingPlanFullCycle: the guarded real-tier sibling carrying the D-12/D-13 evidence contract (no exported key, EXP-CODINGPLAN-CYCLE-01, ONE tool-less turn cap, canonical run command, scrub gate) with the apicycle guard triple and zero-orphan proof
  - experiments/README.md quota-ledger row EXP-CODINGPLAN-CYCLE-01 — the planned-spend entry (PENDING) recorded BEFORE the run, with the A4/A6/A7 confirm-or-adjust recording contract
  - experiments/scan-secrets.sh — the gate green again on the tree (06-07 synthetic PAT markers allowlisted per the established self-declaring-marker protocol)
affects: [phase end-of-phase verification (the human-gated real run harvests the Task-3 human-check), 07-hardening (the guarded tier + evidence pipeline carry forward)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 14270        # 57078 diff chars / 4 over the realized diff
  tasks: 3
  commits: 4           # MEASURED: git rev-list --count 30fdbd2..1b0fffd (the SUMMARY/docs commits follow)

# Tech tracking
tech-stack:
  added: []            # zero new dependencies (stdlib + the existing fixture corpus only — T-06-SC held)
  patterns:
    - "one-cache composition at the composed tier: the cyclePat double sits behind BOTH the front's PatEnsurer seam and the responder's PatHeadersSource (the production one-minter lifecycle); a second Ensure fails loud"
    - "runtime-headers answer asserted as a SHAPE at the real tier: field-presence booleans over requestAuth (token+id+accountScope) recorded via the OnRequest tap — never the token material itself (T-06-24)"
    - "fixture material single-sourced from the corpus: the sealed credential blob is dug from the coding-plan-auth-cycle row at run time instead of duplicating bytes the scan-secrets exact-line allowlist cannot match"

key-files:
  created:
    - internal/acpfront/codingplan_cycle_test.go
    - test/realdesktop/codingplan_test.go
  modified:
    - test/realdesktop/apicycle_test.go
    - experiments/README.md
    - experiments/scan-secrets.sh

key-decisions:
  - "The guarded real-tier cycle defaults to NO model selection: the session binds the backend's confirmed current (the D-13-minimal shape); the ZCODE_ACP_MODEL env override rides exactly as production parses it"
  - "The thought_level option is RECORDED either way at the real tier — present confirms A4 live, absent is the honest A4-adjust posture logged loudly for the ledger; only the model option (mandatory, confirmed currentValue, non-empty entries) hard-fails (D-10/locked OQ2)"
  - "The cycle scenario keeps the cipher ask at the create window (the 06-05 corpus position) and fires the runtime-headers ask MID-TURN on session/send — the ask is answered from the cache with zero extra mints (the mint-once contract)"
  - "The real run's evidence recording (scrub, ledger actuals, V-EXP id, A4/A6/A7 notes) is human-controlled per the UAT item — the executor prepared harness, runbook, and the PENDING ledger row and never touches the account (D-12/D-13, the 05-12/05-14 discipline)"

requirements-completed: [AUTH-02, AUTH-03]

# Coverage metadata (#1602) — one entry per shipped deliverable.
coverage:
  - id: D1
    description: "The zero-spend composed full cycle at the fake tier: authenticate{zai-individual-coding-plan} (probe -> login -> overlay push -> registry read-back verdict -> mint-once) -> session/new returning configOptions with BOTH entries and reply-confirmed currentValues -> set_config_option model switch (level-carry setModel, reply-built re-emit) -> one tool-less streamed turn with the runtime-headers ask answered from the scripted PAT source mid-turn, every ordering lock asserted on the tee + transcript"
    requirement: AUTH-02
    verification:
      - kind: integration
        ref: "internal/acpfront/codingplan_cycle_test.go#TestCodingPlanFullCycleFake"
        status: pass
    human_judgment: false
  - id: D2
    description: "The guarded real-tier sibling TestCodingPlanFullCycle: realdesktop build tag + ZCODE_ACP_REAL_E2E runtime guard + skip-when-unconfigured (the D-02 keyring probe over the real machine), the header evidence contract (canonical run command, D-13 spend cap, EXP-CODINGPLAN-CYCLE-01, scrub gate, no-key-material rule), the production machinery/responder mirror, staged shutdown with the zero-orphan proof — compiles clean under the tag and skips safely without the env var"
    requirement: AUTH-02
    verification:
      - kind: unit
        ref: "go vet -tags realdesktop ./test/realdesktop/ && go build -tags realdesktop ./test/realdesktop/ (both green)"
        status: pass
      - kind: unit
        ref: "go test -tags realdesktop -run TestCodingPlanFullCycle ./test/realdesktop/ without the env var -> SKIP (0.00s, nothing touched)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The HUMAN-GATED real run (the phase bar, D-12): authenticate read-back proven from registry state, the real PAT mint at api.z.ai, session/new configOptions rendered from the live registry, one real tool-less tiny turn with the runtime-headers ask answered (token+id+accountScope), staged shutdown zero-orphan — then the evidence recording (scrub to EXP-CODINGPLAN-CYCLE-01.jsonl, ledger actuals + V-EXP id, scan-secrets on tree AND staged diff, A4/A6/A7 confirm-or-adjust notes) and the evidence commit. HARVESTED by the end-of-phase verifier from Task 3's <human-check> block (human_verify_mode=end-of-phase); the executor-side preparation (ledger PENDING row, pre-flight, runbook) is complete"
    requirement: AUTH-03
    verification: []
    human_judgment: true
    rationale: "The real-account cycle is a guarded manual tier by design (D-12/D-13): it spends real quota on the human's coding-plan account through the desktop keyring and must never run autonomously or in CI — the canonical command and the evidence contract are in test/realdesktop/codingplan_test.go's header and the end-of-phase UAT item (the plan's Task-3 human-check); its verdict lands in the quota ledger and COVERAGE.md's disposition column"

# Metrics
duration: 19min
completed: 2026-11-22
status: complete
---

# Phase 6 Plan 08: Real-Run Handoff & Composed Cycles Summary

**The zero-spend composed coding-plan cycle proves every phase capability end-to-end at the fake tier, the guarded real-tier sibling carries the full D-12/D-13 evidence contract, and the human-gated real run (EXP-CODINGPLAN-CYCLE-01) is handed off with its ledger row PENDING and pre-flight green**

## Performance

- **Duration:** 19 min
- **Started:** 2026-11-22T04:31:42Z
- **Completed:** 2026-11-22T04:52:07Z
- **Tasks:** 3 (the Task-3 human-check harvested by the end-of-phase verifier per human_verify_mode=end-of-phase)
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments
- TestCodingPlanFullCycleFake (Task 1): the composed cycle over the full harness — authenticate{zai-individual-coding-plan} answers through the honest chain (probe -> fixture login -> account-overlay push with the visible overlay markers -> registry patch latching the watcher -> EXACTLY ONE mint after the verdict), session/new returns BOTH configOptions entries with the create-reply-confirmed currentValues (model = account GLM-5.3, thought = high from the bound model's reasoning.levels), the model switch maps to exactly one level-carrying setModel whose response re-emits the REPLY-BUILT list, and one tool-less turn streams to end_turn with the mid-turn runtime-headers ask answered from the PAT cache (the expect_answer semantic-compare byte-pins the three-field answer; the cyclePat double fails loud on any second Ensure)
- Ordering locks (Task 1): the tee's client story strictly ordered (authenticate -> session/new -> set_config_option -> prompt), the back leg's push/refresh before the cycle's create, the single setModel before the send, and the runtime-headers ask fired exactly once with no unanswered note
- TestCodingPlanFullCycle (Task 2): the apicycle discipline verbatim — guard triple (tag + runtime guard + skip-when-unconfigured via the D-02 keyring probe), the header evidence contract (canonical run command, spend cap, capture id, scrub gate, no-key-material rule), the production machinery/responder mirror (mode-aware PAT source over the SAME minter), live-registry configOptions assertions (model mandatory; thought_level recorded either way — A4/A7's designed confirmer), the turn leg asserting the three-field requestAuth SHAPE (values never recorded), staged shutdown + zero-orphan proof
- Task 3 executor-side preparation: the EXP-CODINGPLAN-CYCLE-01 quota-ledger row recorded PENDING before the run (planned spend: the PAT mint + registry read-back + ONE tool-less tiny turn — nothing else), pre-flight green (guard triple proven, tagged vet/build green, scan-secrets green on the tree), and the canonical command + evidence contract presented for the end-of-phase UAT
- Wave gate: `mise run ci` green (build + vet + lint + race suites + experiments), `experiments/scan-secrets.sh` exit 0

## Task Commits

Each task was committed atomically:

1. **Task 1: Zero-spend composed full cycle at the fake tier** - `aa7988e` (feat)
2. **Task 2: Guarded real-tier sibling TestCodingPlanFullCycle** - `71a0029` (feat; includes the pre-existing tagged-tier signature fix)
3. **Task 3: Real-run handoff (ledger + pre-flight)** - `626180d` (fix: gate-green fixes) + `1b0fffd` (docs: the ledger row)

**Plan metadata:** the docs commit(s) following this SUMMARY

## Files Created/Modified
- `internal/acpfront/codingplan_cycle_test.go` - the composed fake-tier cycle (5 stages + ordering locks + the cyclePat/cipher fixtures)
- `test/realdesktop/codingplan_test.go` - the guarded real-tier sibling with the full evidence contract in its header
- `test/realdesktop/apicycle_test.go` - newSessionLeg rides the 06-04 three-value NewSession signature (tagged-tier compile fix)
- `experiments/README.md` - the EXP-CODINGPLAN-CYCLE-01 planned-spend ledger row (PENDING)
- `experiments/scan-secrets.sh` - the 06-07 synthetic PAT corpus markers allowlisted (self-declaring-marker protocol)

## Decisions Made
- The guarded real-tier cycle binds NO model by default (the backend's confirmed current — the D-13-minimal shape); the production env override surface is honored verbatim
- thought_level at the real tier is the A4 designed confirmer: presence logged as CONFIRMED-SHAPE, absence logged as the ADJUST-SHAPE (honest omission posture) — never silently assumed, never a false failure of a designed fallback
- The cycle's cipher ask stays at the create window (the corpus-proven position) and its blob is dug from the corpus row at run time — one committed source for the fixture triple (scan-secrets' exact-line allowlist stays intact)
- The runtime-headers answer at the real tier is asserted as a SHAPE through the OnRequest tap (field-presence booleans) — the token material never enters a log, an assertion message, or a capture note (T-06-24)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Tagged realdesktop tier did not compile (pre-existing)**
- **Found during:** Task 2 (the plan's own `go vet -tags realdesktop` verify)
- **Issue:** 06-04 widened `front.NewSession` to return (sid, configOptions, err) but `test/realdesktop/apicycle_test.go` still consumed two values — the guarded tier (this plan's verify gate) was red on master before any 06-08 change
- **Fix:** newSessionLeg rides the three-value signature and logs the returned configOptions structurally (the api-key cycle's assertion surface stays the read-back + turn legs)
- **Files modified:** test/realdesktop/apicycle_test.go
- **Verification:** `go vet -tags realdesktop ./test/realdesktop/` + `go build -tags realdesktop ./test/realdesktop/` green
- **Committed in:** 71a0029 (Task 2 commit)

**2. [Rule 3 - Blocking] scan-secrets red on the tree (Task 3's verify gate)**
- **Found during:** Task 3 pre-flight (`bash experiments/scan-secrets.sh` exited 1)
- **Issue:** two independent hits — (a) the Task-1 blob constant, lll-rewrapped across two string literals, no longer matched the gate's exact-line allowlist entry for the corpus blob; (b) the 06-07 runtime-headers-pat corpus/golden synthetic PAT markers (`zc-synthetic-pat-{a,b}-corpus-fixture`) were never allowlisted, so the gate had been red since 06-07 landed them
- **Fix:** the cycle now digs the sealed blob from the coding-plan-auth-cycle corpus row at run time (one committed source — no duplicate fixture material); the allowlist gains the two self-declaring synthetic markers per the file's established protocol (the 05-06 posture: the golden byte-locks the literals the ask-side byte-compare needs)
- **Files modified:** internal/acpfront/codingplan_cycle_test.go, experiments/scan-secrets.sh
- **Verification:** `bash experiments/scan-secrets.sh` exit 0; the composed cycle test stays green; lint + CI green
- **Committed in:** 626180d

---

**Total deviations:** 2 auto-fixed (2 blocking)
**Impact on plan:** Both fixes gate tasks this plan owns (Task 2's verify and Task 3's verify explicitly require the tagged build and a green secret gate). No scope creep; no production code touched.

## Issues Encountered
- None beyond the two deviations above; the gsd_run windows ledger recorded the human-gated real run as an open `unrun-verify` entry (resolved by the end-of-phase UAT).

## User Setup Required

The REAL run (the human-gated end-of-phase UAT — the phase bar, success criterion 1) requires, at run time:

- The desktop logged into the z.ai individual coding plan and the session's keyring unlocked (D-12: the desktop login IS the credential — no key is exported anywhere)
- `ZCODE_ACP_RAW_CAPTURE` pointed at a scratch path OUTSIDE this repo (raw captures are born outside, then scrubbed in)

Canonical command (verbatim from test/realdesktop/codingplan_test.go's header):

```
export ZCODE_ACP_RAW_CAPTURE=<scratch path outside this repo>
ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestCodingPlanFullCycle ./test/realdesktop/
```

After the run: scrub the raw capture into `experiments/captures/EXP-CODINGPLAN-CYCLE-01.jsonl`, fill the ledger actuals + V-EXP id, run scan-secrets on the tree AND the staged diff, record the A4 (thoughtLevel block) / A6 (clientVersion) / A7 (configOptions rendering) confirm-or-adjust notes, then confirm the evidence commit. A failed leg records the honest verdict with the scrubbed capture — the phase stays open (the 05-14 discipline). Full contract: the plan's Task-3 `<human-check>` block and the guarded test's file header.

## Human-Gated Item (end-of-phase harvesting)

The real coding-plan cycle (AUTH-02/AUTH-03 field evidence, D-12) was NOT executed by the executor — by plan design and D-13 (never autonomously, never CI). It is prepared, ledgered (PENDING), pre-flight-verified, and documented for the end-of-phase verifier, which harvests Task 3's `<human-check>` block as the phase UAT (human_verify_mode=end-of-phase). No unmet precondition blocked the executor-side work: every automated Task-3 step (ledger row, guard-triple proof, tagged vet/build, scan-secrets) ran green in this session.

## Next Phase Readiness
- The phase's remaining open item is exactly the human-gated real run + its evidence recording; everything machine-verifiable is green (`mise run ci`, scan-secrets, lint, the full race suite)
- Phase 7 inherits: the guarded tier pattern (now two siblings), the quota-ledger evidence discipline, and the composed-cycle harness for any future path

---
*Phase: 06-oauth-coding-plan-path-model-picker*
*Completed: 2026-11-22*
