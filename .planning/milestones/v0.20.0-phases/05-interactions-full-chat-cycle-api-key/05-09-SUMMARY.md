---
phase: 05-interactions-full-chat-cycle-api-key
plan: 09
subsystem: guarded-real-tier
tags: [auth-01, core-06, sess-01, tool-01, tool-02, realdesktop, double-guard, full-cycle, quota-ledger, manual-tier]

# Dependency graph
requires:
  - phase: 05-interactions-full-chat-cycle-api-key (05-07)
    provides: the composition surface — Front.Authenticate + AuthConfig, responder options (WithPermissionForwarder/WithRuntimeHeadersKeySource/WithCredentialDecrypt), SecretToolKeyLoader + ScrubbedEnv, the pre-traffic OnRequest wiring order
  - phase: 05-interactions-full-chat-cycle-api-key (05-08)
    provides: the cycle leg order this test replays live (authenticate → new → tool turn → resume → close) and the corpus-locked model selection
  - phase: 03-zcode-back-leg-codec-fake-app-server-supervisor (03-07)
    provides: the guard discipline copied verbatim (realdesktop tag + env double guard, canonical header command, discover→spawn→exchange→staged-shutdown skeleton, SpawnChild, BuiltinRevision)
provides:
  - test/realdesktop.TestAPIKeyFullCycle — the guarded real-tier full cycle (double guard, production composition via exported pieces, honest wire-tier asserts, optional wire tee for the scrub-first evidence pipeline)
  - experiments/README.md Phase-5 quota-ledger row (PENDING) — the end-of-phase real run's bookkeeping (planned spend, no-CI rule, evidence pointer format)
  - .planning/phases/05-interactions-full-chat-cycle-api-key/05-USER-SETUP.md — the human-held credential + keyring checklist for the real run
affects: [end-of-phase real run (success criteria 4+5 live half), phase-verification human-check harvest, Phase-6 OAuth real tier (same guard discipline)]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
actuals:
  tokens: 8802    # chars/4 over the realized diff (35209 chars, 3 files, +923/-0); plan estimated 24000
  tasks: 2
  commits: 2      # MEASURED: git rev-list --count gsd-plan-head-before-05-09..HEAD
commits: 2
plan_head_before: c6444fc9e39328c5b4b3fd3acfe1da6687f8d6de
plan_head_after: 92d6042220b6b8b98bc0183907d05d1333ec4a64

# Tech tracking
tech-stack:
  added: []        # zero new dependencies — syscall/syscall.Kill is stdlib; secret-tool stays a system binary (T-5-SC)
  patterns:
    - guarded-tier composition via exported pieces: the test rebuilds production run()'s stack exactly (buildFront/wireResponder mirrored) because the cmd wiring is unexported — every seam it needs is exported (acpfront options, hostsvc helpers, interact responder options)
    - the test composition IS the ACP client: the permission sender binds via the exported acpport binder hook (front.BindPermissionSender) with a first-allow-class decision stand-in, mirroring what NewAgentSideConnection does in production
    - capture tee behind an env var (ZCODE_ACP_RAW_CAPTURE): raw captures born outside the repo per Phase-1 ground rule 1; the header carries the V-EXP id + the observed builtin revision (recorded, never assumed); the scrubber remains the only path into experiments/captures/

key-files:
  created:
    - test/realdesktop/apicycle_test.go
    - test/realdesktop/doc.go
    - .planning/phases/05-interactions-full-chat-cycle-api-key/05-USER-SETUP.md
  modified:
    - experiments/README.md

key-decisions:
  - "Missing ZCODE_ACP_API_KEY SKIPS (after the guard skip) rather than exercising the honest no-key coded error: the composed tier (05-07) locks that arm; the real tier must not burn a spawn+discovery cycle on a guaranteed coded failure — the skip message names the export"
  - "Default model selection is the corpus-locked zai-api/GLM-5.3-Flash + high (env-overridable via ZCODE_ACP_MODEL/ZCODE_ACP_REASONING_LEVEL): the D-05 minimal-turn shape field-proven on this account; the setModel follow-up carries the level because level-strict providers refuse the empty-options shape (the EXP-TURN-01 -32603 quirk)"
  - "The permission sender stand-in answers the first allow-class option (allow_once/allow_always) and logs asks structurally (tool name only, never params): a read-tool turn on a scratch file is not expected to trigger a permission ask (A1-adjacent), but an unbound sender would turn any ask into a coded deny — the bound stand-in keeps the cycle alive without fabricating anything the client tier doesn't already lock"
  - "Assertions stay at the honest wire tier exactly as planned: the authenticate read-back verdict (Front.Authenticate nil), end_turn + zero tool frames on the tool-less turn, end_turn + observed tool frames with a terminal status on the read turn, same-session resume + re-issued subscribe, clean close + post-close coded rejection, staged shutdown + polled zero-orphan — never an over-fit to exact tool titles"

requirements-completed: [AUTH-01, CORE-06, SESS-01, TOOL-01, TOOL-02]  # verbatim copy of the plan frontmatter

# Coverage metadata (#1602)
coverage:
  - id: D1
    description: "The guarded real-tier test lands with the 03-07 discipline: //go:build realdesktop + ZCODE_ACP_REAL_E2E=1 double guard (plus the api-key skip arm), the canonical manual command + quota note + evidence discipline in the file header, the production composition (discovery with the env override seam → SpawnChild + ChildEnv allowlist → zcode.Conn over exec pipes → Front with auth machinery + corpus model selection → full responder policy table incl. the D-06 secret-tool cipher seam as conn.OnRequest pre-traffic, key from the environment), the full cycle legs, and the optional UNSCRUBBED wire tee for the scrub-first pipeline"
    requirement: AUTH-01
    verification:
      - kind: unit
        ref: "go build -tags realdesktop ./... && go vet -tags realdesktop ./... (composition compiles under the gates)"
        status: pass
      - kind: unit
        ref: "go test -tags realdesktop -count=1 -v -run TestAPIKeyFullCycle ./test/realdesktop/ → --- SKIP (0.00s) without ZCODE_ACP_REAL_E2E=1; go test -count=1 ./test/realdesktop/ (no tag) → [no test files]"
        status: pass
      - kind: integration
        ref: "go test -race -count=1 ./internal/... ./cmd/... (default suite untouched) + bash experiments/scan-secrets.sh (exit 0 — zero credential material) + mise run ci (exit 0; the tagged package stays out of default gates)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The Phase-5 quota-ledger row exists PENDING in experiments/README.md: planned spend (registry read-back + two tiny turns + resume/close), the standing never-from-CI rule restated for the guarded tier, the evidence pointer format (scrubbed capture EXP-APIKEY-CYCLE-01.jsonl + V-EXP id, actuals appended by the run), and the canonical command pointer to the test file header"
    verification:
      - kind: unit
        ref: "grep -c 'Phase-5|phase 05' experiments/README.md (= 1) + bash experiments/check-docs.sh (exit 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "THE REAL RUN: the full cycle completes on the real z.ai api-key account — authenticate proven by the registry read-back verdict, the cipher ask answered from the login keyring mid-turn (the D-06 real-tier proof, discharging the 05-07 username-derivation assumption), the scrubbed capture + V-EXP id + ledger actuals recorded"
    verification: []
    human_judgment: true
    rationale: "Coverage not determined at authoring time — the REAL RUN is the end-of-phase verification act (orchestrator-run, quota-ledgered per D-05), explicitly NOT an executor step (the plan's user_setup.why and the human-check boundary). Everything the run needs is staged and verified: the guarded test, the ledger row, 05-USER-SETUP.md. The verifier must classify after the run executes."

# Metrics
duration: 16min
completed: 2026-11-20
status: complete

---

# Phase 5 Plan 09: Guarded Real Tier + Quota Ledger Summary

**The guarded real-tier full-cycle test lands: `TestAPIKeyFullCycle` behind the 03-07 double guard composes the entire production stack via exported pieces (discovery → SpawnChild → zcode.Conn → Front → full responder policy table with the D-06 secret-tool cipher seam, key sourced from `ZCODE_ACP_API_KEY` at run time) and drives authenticate(read-back verdict) → new → tiny tool-less turn → file-read tool turn → resume → close → staged shutdown — verified to skip honestly in ~2 ms with the default suite untouched — plus the PENDING Phase-5 quota-ledger row staging the end-of-phase real run.**

## Performance

- **Duration:** 16 min
- **Started / Completed:** 2026-11-20T15:40:26Z / 2026-11-20T15:56:23Z
- **Tasks:** 2 · **Files:** 3 created / 1 modified (+923/-0; test-weighted — the guarded test IS the deliverable)

## Accomplishments

- `test/realdesktop/apicycle_test.go` carries the 03-07 guard discipline verbatim: `//go:build realdesktop` + the `ZCODE_ACP_REAL_E2E=1` runtime skip (plus a skip arm for a missing `ZCODE_ACP_API_KEY`), the canonical manual command in the file header, the D-05 quota note, and the evidence-discipline note (scrub first, V-EXP id `EXP-APIKEY-CYCLE-01`, ledger row).
- The test composes the SAME stack production `run()` builds, through the exported seams: discovery (the `ZCODE_ACP_APP_SERVER` override seam or the default matrix, probe rows logged), `SpawnChild` + the allowlist `ChildEnv` under the two-lifecycle discipline, `zcode.Conn` over the exec pipes, the Front (auth machinery + read-back want list + the corpus-locked model selection), and the full responder policy table — permission bridge, runtime-headers key source, D-06 credential-decrypt seam — assigned `conn.OnRequest` BEFORE any traffic. The key is read from the environment at run time; never embedded, never logged.
- The cycle legs assert at the honest wire tier: the authenticate read-back verdict, `end_turn` with zero tool frames on the tool-less turn, `end_turn` with observed tool frames (start + terminal status) on the file-read turn, same-session resume with the re-issued subscribe accepted, clean close with the post-close prompt rejected coded-clean, and the polled zero-orphan proof after the staged shutdown.
- The optional wire tee records both directions (header + `{ts,dir,line}` records, the probe's capture shape plus direction) to `ZCODE_ACP_RAW_CAPTURE` — UNSCRUBBED, born outside the repo, scrubbed via `scrub-capture.mjs` before any commit (the capture is produced at real-run time, never at test-landing time).
- The Phase-5 quota-ledger row is PENDING-marked in `experiments/README.md`: planned spend (registry read-back + two tiny turns + resume/close — no model spend beyond the D-05 minimum), the never-from-CI rule restated for the guarded tier, the evidence pointer format, and the canonical command pointer.
- `mise run ci` exit 0 (lint 0 issues; the tagged package stays out of default gates — `[no test files]`), zero new dependencies.

## Task Commits

1. **Task 1: test/realdesktop/apicycle_test.go — the guarded api-key full cycle** - `f406da4` (test)
2. **Task 2: quota ledger row + real-tier run instructions** - `92d6042` (docs)

**Plan metadata:** (this SUMMARY + USER-SETUP + STATE/ROADMAP/REQUIREMENTS via the sequential-dispatch metadata commit)

## Files Created/Modified

- `test/realdesktop/apicycle_test.go` (NEW) — the guarded full-cycle test: double guard, production composition, cycle legs, honest wire-tier asserts, capture tee
- `test/realdesktop/doc.go` (NEW) — package identity so the tagged directory stays addressable by the default `./...` gates (see Deviations)
- `experiments/README.md` — the PENDING Phase-5 quota-ledger row
- `.planning/phases/05-interactions-full-chat-cycle-api-key/05-USER-SETUP.md` (NEW) — the env vars, keyring-unlock verification, and the real-run + scrub commands

## Decisions Made

See key-decisions in frontmatter: skip-on-missing-key (the composed tier owns the no-key arm), the corpus-locked default model selection with env overrides, the bound permission-sender stand-in (first allow-class option, structural logging), and the honest wire-tier assertion set.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] Added `test/realdesktop/doc.go` so the new tagged directory stays inside the `./...` package pattern**
- **Found during:** Task 1 design verification (pre-write probe)
- **Issue:** The plan's `files_modified` lists only `apicycle_test.go`; a directory whose ONLY file is build-tag-excluded falls out of the `./...` pattern entirely — `go build ./...` emits `matched no packages` noise on every default gate, and the tagged package is not addressable by wildcard builds/vets.
- **Fix:** One non-tagged `doc.go` (package clause + guard-contract comment, no code) gives the directory a buildable package identity. Default gates see an empty package (`[no test files]`); the tagged build/vet see the guarded test.
- **Files modified:** test/realdesktop/doc.go (created)
- **Verification:** `go build ./...` clean (no warning), `go vet ./...` clean, `go test ./...` reports `[no test files]`, `go build/vet -tags realdesktop ./...` compile the test file.
- **Committed in:** f406da4

**Total deviations:** 1 auto-fixed (1 Rule 3 blocker-avoidance).
**Impact on plan:** none on contracts — additive packaging file only; the guarded test's behavior is unchanged.

## Issues Encountered

None — both tasks verified green on the first battery (gofmt clean, tagged build/vet, honest 2 ms skip, default suite green under -race, scan-secrets green, check-docs green, `mise run ci` exit 0).

## Authentication Gates

None — the executor deliberately did NOT touch credentials (the real run is the end-of-phase act). The key path is documented in 05-USER-SETUP.md; no credential material entered the repo (scan-secrets green on both commits).

## User Setup Required

**External services require manual configuration.** See [05-USER-SETUP.md](./05-USER-SETUP.md) for:
- `ZCODE_ACP_API_KEY` (export into the manual-run shell only — never commit) and `ZCODE_ACP_RAW_CAPTURE` (scratch path outside the repo)
- Keyring unlock + the `secret-tool` resolution check (the 43-char key, field-verified shape)
- The real-run command, the scrub-first step, and the ledger actuals append

## Next Phase Readiness

Phase plans complete — ready for phase verification. The real-half proof path is fully staged: the guarded test composes every phase-5 contract live (CORE-06 responder table, TOOL-01 tool frames, TOOL-02 bridge seam, SESS-01 resume/close, AUTH-01 read-back-proven authenticate), the quota ledger row caps the planned spend, and 05-USER-SETUP.md names the two human-held inputs (the key, the unlocked keyring). The end-of-phase run discharges the remaining judgment coverage (D3 here; D5 of 05-07 — the live-keyring decrypt and the username-derivation assumption). REQUIREMENTS.md's shared IDs (AUTH-01, CORE-06, SESS-01, TOOL-01, TOOL-02) become ready with this SUMMARY landing (the 05-08 shared-ID gate resolves).

---
*Phase: 05-interactions-full-chat-cycle-api-key*
*Completed: 2026-11-20*

## Self-Check: PASSED

All 5 files verified present on disk (apicycle_test.go, doc.go, experiments/README.md, this SUMMARY, 05-USER-SETUP.md); all 3 commits verified ancestors of HEAD (`f406da4` test, `92d6042` docs, `7cdf3be` docs metadata). Commit count measured from the persisted plan ledger (gsd-plan-head-before-05-09 = c6444fc): 2 production commits, matching the frontmatter actuals. No unexpected file deletions; no untracked files outside .planning/ (orchestrator-owned).
