---
phase: "4"
slug: "acp-front-minimal-chat-loop"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-20"
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (house style), `-race` mandatory |
| **Config file** | none — `go test` + mise tasks (existing) |
| **Quick run command** | `go test -race -count=1 ./internal/acpport/... ./internal/acpfront/... ./internal/zcode/... ./internal/turn/...` |
| **Full suite command** | `mise run ci` (build → vet → lint → test -race → test-experiments) |
| **Estimated runtime** | ~90 seconds |

---

## Sampling Rate

- **After every task commit:** quick run command (four packages)
- **After every plan wave:** `mise run ci`
- **Phase gate:** full suite green + exec-chain E2E green (happy + cancel in default suite)
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

One row per real task (13 tasks across plans 04-01..04-06). Restructured and re-executed at the
2026-11-20 Nyquist audit — statuses below are **re-executed evidence**, not inherited from the
SUMMARYs. Multi-test commands live in the Audit Re-Execution Battery (fenced block) so the `|`
alternation stays copy-safe; table cells carry only pipe-free commands.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-T1 | 04-01 | 1 | infra: D-01 SDK pin + tidy-diff gate + subscribe/setModel constants | T-4-SC | sole require line pinned v0.20.0; unapproved requires fail the gate | gate + build | `mise run tidy-diff` | ✅ | ✅ green (re-exec) |
| 04-01-T2 | 04-01 | 1 | infra (enables CORE-01..05): conn observer fan-out | — | ordered fan-out, no hook theft, remove | unit (TDD) | `go test -race -count=1 -run TestObserverFanOut ./internal/zcode/` | ✅ | ✅ green (4/4 subtests) |
| 04-02-T1 | 04-02 | 2 | CORE-02 corpus: setmodel-followup row byte-locked | T-4-01 | setModel payload (reasoningLevel + persist:false) verbatim on the back-leg golden | integration + golden | `go test -race -count=1 -run TestGolden ./internal/zcode/fake/` | ✅ | ✅ green (15 rows) |
| 04-02-T2 | 04-02 | 2 | infra (T-4-SC) + CORE-01 adapter: acpport seam | T-4-SC | SDK behind the seam only; honest init assembly; -32601 defaults; coded errors | unit + boundary | `go test -race -count=1 ./internal/acpport/` | ✅ | ✅ green |
| 04-02-T3 | 04-02 | 2 | infra: core session registry + front-tier harness | — | mutex registry, loud ErrNoSession miss; harness compiles under -race | unit + compile | `go test -race -count=1 ./internal/core/` | ✅ | ✅ green |
| 04-06-T1 | 04-06 | 3 | CORE-02 corpus: cancel-mid-turn front preamble | — | create→subscribe→setModel preamble byte-locked in wire order | integration + golden | `go test -race -count=1 -run TestGolden ./internal/zcode/fake/` | ✅ | ✅ green |
| 04-06-T2 | 04-06 | 3 | CORE-01..05 (tracer union) | T-4-SC, T-4-02 | honest loop end-to-end (init → session/new → stream → cancel) | integration (tracer, TDD) | `go test -race -count=1 -run TestMinimalChatLoop ./internal/acpfront/` | ✅ | ✅ green (3 stages) |
| 04-03-T1 | 04-03 | 4 | CORE-01 | T-4-04 | honest caps (zero fs/terminal/loadSession fields); honest auth failure (-32603-class); -32601 defaults | integration + transcript | `go test -race -count=1 ./internal/acpfront/ ./internal/acpport/` (named tests: B7) | ✅ | ✅ green (10 tests) |
| 04-03-T2 | 04-03 | 4 | CORE-02 | T-4-01 | relative cwd rejected pre-wire (-32602); mcpServers accepted-and-ignored; setModel payload verbatim | integration + golden | `go test -race -count=1 -run TestNewSession ./internal/acpfront/` | ✅ | ✅ green |
| 04-04-T1 | 04-04 | 4 | CORE-04 | — | thought only on reasoning deltas (negative locked: never synthesized) | integration + golden | `go test -race -count=1 -run TestThoughtChunks ./internal/acpfront/` | ✅ | ✅ green |
| 04-04-T2 | 04-04 | 4 | CORE-03 | — | deltas precede response; stopReason after flush; both completion variants; zero-delta honesty | integration (proof) | `go test -race -count=1 -run TestStreamBeforeStopReason ./internal/acpfront/` | ✅ | ✅ green |
| 04-05-T1 | 04-05 | 5 | CORE-05 | T-4-05, T-4-02 | cancelled exactly once, never -32800; one -32600 rejection surface (both guard layers); cancel-after-complete silent | integration | `go test -race -count=1 ./internal/acpfront/` (named tests: B10) | ✅ | ✅ green (5 tests) |
| 04-05-T2 | 04-05 | 5 | CORE-05 (exec leg) | — | built binary loop over real exec pipes, fake backend via Override seam | exec-chain | `go test -race -count=1 ./internal/supervisor/ ./cmd/zcode-acp/` (named tests: B13) | ✅ | ✅ green (6 tests) |
| (manual) real-client spot check | 04-05 | — | phase-level | — | spawn cmd/zcode-acp under a real ACP client, fake wired via supervisor Override | manual | — | — | ⏸ DEFERRED (non-gating) |

*Status legend: ✅ green · ❌ red · ⚠️ flaky · ⏸ deferred (non-gating). All green rows re-executed 2026-11-20.*

---

## Audit Re-Execution Battery (2026-11-20)

Exact commands as executed at audit, with observed evidence. Multi-test `-run` patterns use
**bare `|` alternation** — Go's `go test -run` takes an RE2 pattern where `\|` matches a literal
pipe character (see Finding F-1).

```sh
# Gates
mise run ci                                                        # exit 0 (build, vet, lint 0 issues, test -race ./... 8 pkgs ok, experiments 22/22)
go test -race -count=1 ./internal/acpport/ ./internal/acpfront/ ./internal/zcode/ ./internal/turn/ ./internal/supervisor/   # all ok
mise run tidy-diff                                                 # exit 0 (04-01-T1)
bash experiments/scan-secrets.sh                                   # exit 0
bash experiments/check-docs.sh                                     # exit 0

# Map rows (named tests), -race -count=1, with observed RUN counts
go test -race -count=1 -run TestObserverFanOut ./internal/zcode/                        # B2: 4/4 subtests PASS
go test -race -count=1 -run TestMinimalChatLoop ./internal/acpfront/                    # B3: 3/3 stages PASS
go test -race -count=1 -run TestSDKImportBoundary ./internal/acpport/                   # B4: PASS
go test -race -count=1 -run 'TestInitializeHonestCapabilities|TestAuthenticateHonestError|TestUnadvertisedStableMethodsAnswerNotFound' ./internal/acpfront/ ./internal/acpport/   # B7: 10/10 PASS
go test -race -count=1 -run 'TestNewSession|TestGolden' ./internal/acpfront/ ./internal/zcode/fake/             # B8: 20/20 PASS (3 session/new tests + 15-row golden corpus)
go test -race -count=1 -run TestThoughtChunks ./internal/acpfront/                      # B9: 3/3 PASS (positive + negative)
go test -race -count=1 -run TestStreamBeforeStopReason ./internal/acpfront/             # B11: 4/4 PASS (terminal-event, quiet-window, zero-delta)
go test -race -count=1 -run 'TestCancelMidTurn|TestSecondPromptRejected|TestCancelAfterCompleteNoOp' ./internal/acpfront/       # B10: 5/5 PASS
go test -race -count=1 -run 'TestEndToEndExecChain|TestAgentBinaryEndToEnd' ./internal/supervisor/ ./cmd/zcode-acp/             # B13: 6/6 PASS
go test -race -count=1 ./internal/core/                                                 # 04-02-T3 registry: ok
```

---

## Audit Findings and Repairs (Nyquist, 2026-11-20)

**F-1 — trivially green map commands (REPAIRED).** Four legacy map rows (initialize/authenticate/
unadvertised; session/new+golden; cancel trio; exec-chain+binary E2E) carried `-run 'A\|B'`
patterns — grep-BRE escaping copied into `go test -run`. Go's RE2 treats `\|` as a literal pipe
character, so the pattern matched ZERO tests and go test reported `ok ... [no tests to run]` —
a command that passes without executing anything. Verified empirically both ways at audit
(`\|` → `[no tests to run]`; bare `|` → both tests run and pass). The named tests themselves
were never at fault (all pass individually and in alternation — see battery); only the recorded
commands were defective. All map rows now carry pipe-free commands, and the exact multi-test
alternations are preserved in the battery block above.

**F-2 — map coverage repair (REPAIRED).** The legacy 9-row requirement-centric map did not name
two task verifies: 04-01-T1 (the tidy-diff dependency-pin gate — runs in `mise run ci`'s
pr-sanity lineage and re-executed here) and 04-02-T3 (core registry unit tests + harness
compile — re-executed here). The map now carries one row per real task: 13/13 tasks covered,
each with an automated command and re-executed status.

**F-3 — no missing behavioral tests.** Every CORE-01..05 edge named in 04-RESEARCH.md's
Validation Architecture has a green named test: initialize raw-frame honesty (absence
assertions), authenticate honest error, -32601 unadvertised defaults, relative-cwd pre-wire
rejection with no backend frame, mcpServers accept-and-ignore, setModel payload verbatim
(reasoningLevel + persist:false), thought positive/negative (never synthesized), flush ordering
across both completion variants + zero-delta probe, cancel exactly-once never -32800, one
-32600 rejection surface from both guard layers, cancel-after-complete silence, fan-out
ordering/remove/two-machines, SDK seam boundary, 15-row fail-loud golden corpus, exec-chain +
binary E2E. No new tests required; existing gates extended only by the F-1/F-2 repairs.

---

## CI Observation

- The Phase-2-era `pr-sanity` workflow exists (`​.github/workflows/pr-sanity.yml`) and triggers on
  `pull_request` (plus `push: master` and `workflow_dispatch`) — the phase branch's PR will run it.
  Its sanity job runs the same mise tasks as local CI (build, vet, golangci-lint, test -race,
  experiments, tidy-diff), with separate govulncheck and secret-scan jobs.
- Branch protection on `master` unchanged (checked via `gh api .../branches/master/protection`,
  2026-11-20): required status contexts are still exactly
  `["build / vet / lint / test / tidy", "govulncheck", "secret-scan"]` — all three produced by
  pr-sanity.

---

## Wave 0 Requirements

- [x] Seam + registry packages and front-tier test scaffolding: `internal/acpport`, `internal/core`, `internal/acpfront/harness_test.go` (+ SDK dep added to go.mod per D-01 audit) — landed by plans 04-01/04-02; the acpfront front/stream/turnctx files landed with the tracer plan 04-06
- [x] Fake corpus additions: `setmodel-followup` (04-02); `cancel-mid-turn` front preamble (04-06); `part-delta-streaming`, `no-reasoning` (04-04); `send-refused-32010`, `complete-then-cancel` (04-05) (+ goldens + drivers; the `mcp-servers-ignored` behavior is locked at the FRONT tier over the setmodel-followup scenario — the back leg has no mcp wire surface to script; see plan 04-03). Corpus: 15 rows, all with drivers and byte-deterministic goldens.
- [x] Test ACP client (in-process over pipes) for the front-tier integration tests — plan 04-02 Task 3 (harness_test.go), exercised end-to-end by the 04-06 tracer and every later lock suite

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions | Status |
|----------|-------------|------------|-------------------|--------|
| Real ACP client spot check (spawn cmd/zcode-acp under a real client, fake wired via supervisor Override) | phase-level | Requires a real client binary; precedent = Phase 3 realdesktop pattern | once at phase end, non-gating (enabled by plan 04-05's --app-server/override seam; instructions in 04-05-SUMMARY "Human Check": build the binary, fabricate the shim install root, point any ACP v1 client at it with `--app-server <root> --model zai-api/GLM-5.3-Flash --reasoning-level high`) | ⏸ DEFERRED — non-gating, never blocks the phase gate |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify (13/13 tasks across plans 04-01..04-06; the only manual element is the non-gating real-client spot check, recorded deferred above)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (landed by plans 04-01/04-02/04-04/04-05/04-06)
- [x] No watch-mode flags
- [x] Feedback latency < 90s (quick run command is the four-package suite; quiet-window rides add ~4s each)
- [x] `nyquist_compliant: true` set in frontmatter
- [x] Gates re-executed at audit (2026-11-20): `mise run ci` exit 0 · five-package `-race` suite ok · `scan-secrets.sh` 0 · `check-docs.sh` 0 · `tidy-diff` 0 · all 13 map rows green (battery above)
- [x] Map repaired at audit: real task IDs (F-2), trivially-green commands replaced (F-1)

**Approval:** validated — Nyquist audit 2026-11-20 (statuses re-executed, not inherited)
