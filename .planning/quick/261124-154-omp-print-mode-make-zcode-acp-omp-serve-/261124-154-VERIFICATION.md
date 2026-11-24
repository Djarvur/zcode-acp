---
phase: 261124-154
quick_task: true
verified: 2026-11-24T00:03:00Z
status: passed
score: 7/7 must-haves verified
covered_files: [".planning/quick/261124-154-omp-print-mode-make-zcode-acp-omp-serve-/261124-154-PLAN.md", ".planning/quick/261124-154-omp-print-mode-make-zcode-acp-omp-serve-/261124-154-SUMMARY.md", "README.md", "cmd/zcode-acp/main.go", "cmd/zcode-acp/main_test.go", "cmd/zcode-acp/testdata/omp-text-apikey-e2e.json", "cmd/zcode-acp/testdata/omp-text-e2e.json", "cmd/zcode-acp/text_test.go", "experiments/README.md", "internal/interact/permission.go", "internal/interact/permission_test.go", "internal/texttask/auth.go", "internal/texttask/auth_test.go", "internal/texttask/generate.go", "internal/texttask/generate_test.go", "internal/texttask/prompt.go", "internal/texttask/prompt_test.go", "internal/texttask/selection.go", "internal/texttask/selection_test.go", "internal/zcode/envelope.go", "test/realdesktop/texttask_test.go"]
covered_digest: "v4:sha256:b07f5a4d12189377b277cb9e722d8d415967977bfa8bf6ad21b8947eac498d7f"
behavior_unverified: 0
overrides_applied: 0
---

# Quick Task 261124-154: OMP `--print --mode text` Verification Report

**Goal:** Make `zcode-acp omp` serve Orca's non-interactive text tasks (chat name, branch name, commit message) by implementing the `omp --print --mode text` CLI contract and returning plain text on stdout instead of starting ACP.

**Verified:** 2026-11-24T00:03:00Z
**Status:** passed
**Re-verification:** No - initial verification
**Worktree:** `/home/nil/DiskD/W/Djarvur/zcode-acp-wt-omp-text` (branch `fix/omp-print-text-mode`, HEAD `a32fb5d`)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Orca's text-task argv (`omp --print --no-session --no-tools --no-extensions --no-skills --no-rules --mode text [--model p/m] [--thinking lvl] @file...`) exits 0 and writes exactly the generated text plus at most one trailing newline to stdout, with zero ACP frames and no ACP handshake on the wire | VERIFIED | `TestOmpTextEndToEnd` (built binary): stdout exactly `"Answer: done\n"`, exit 0, exactly one inbound `workspace/generateText`, zero `initialize`/`session/*` frames, stderr carries no protocol frame. Guarded live probe (orchestrator-run, real desktop): exit 0, 3-byte stdout `ok\n`, 3.95 s, no `-32601`, no protocol frame |
| 2 | `--print` alone and `--mode text` alone both activate text mode; any other `--mode` value exits 2 without spawning the app-server; `omp acp ...` and root ACP invocations keep their current behavior | VERIFIED | `TestOmpTextArgs` activation/marker/mode subtests green; `TestOmpTextExitCodes` proves `--mode plan` exits 2 with no fake transcript (no spawn); `TestOmpFlagSet`, `TestOmpLaunchBinding`, `TestOmpRootCoexistence`, `TestRootModeUnchanged`, `TestAgentBinaryEndToEnd` green; `run()` diff preserves the ACP dispatch and logging; golden corpus diff empty |
| 3 | A prompt delivered as one or more `@file` tokens is read in argv order, one UTF-8 BOM is stripped per file, chunks concatenate with no separator, and missing/unreadable/oversize/empty prompts exit 2 before any backend spawn | VERIFIED | `internal/texttask` unit locks (`TestReadPromptArgChunks`, `TestReadPromptArgErrors`, `TestReadPromptArgEmpty`) green; `TestOmpTextEndToEnd` multi-chunk subtest proves the BOM-free concatenated prompt on the wire; `TestOmpTextExitCodes` proves missing/unreadable prompts exit 2 with no spawn; `textResolve` maps every `ReadPromptArg` sentinel (including `ErrPromptTooLarge`, `ErrPromptEmpty`) to exit 2 before `startBackend` |
| 4 | Selection resolves `--model` (or its `ZCODE_ACP_MODEL` alias) first, then `config.defaultModelSelection` from `~/.zcode/v2/provider_config.json`; a missing/invalid default exits 1 with an actionable message; `--thinking` rides `selection.options.reasoningLevel` and a level without a model exits 2 | VERIFIED | `TestResolveSelectionSources`/`TestResolveSelectionErrors` green (passthrough, default reader, missing/undecodable/empty-id sentinels, no credential echo); `TestOmpTextConfigDefault` proves the no-`--model` default ids and `reasoningLevel` reach the wire; `parseConfig` (reused by `textResolve`) keeps the model-shape and level-without-model exit-2 sentinels; `selectionParams` omits `options` for an empty level (`TestRunGenerateOptionsOmitted`) |
| 5 | Auth runs only when the resolved provider needs it (account:* and the documented zai-api arms); a personal provider such as OpenCodeGo produces exactly one `workspace/generateText` and zero `provider/*` auth frames | VERIFIED | `TestAuthMethod` table green; `TestOmpTextAuth` personal-provider arm: exactly one generateText and zero `provider/*` frames; api-key arm: one `updateAccountConfig` and one `refreshRegistry` strictly before generateText (indices asserted) |
| 6 | Failures stay honest and quiet on stdout: whitespace-only text exits 1, a wire error exits 1 naming its numeric code; one run-level 40 s deadline covers discovery, spawn (bounded by `spawnWait` 10 s), auth and generation (its own 35 s cap), so the 15 s shutdown watchdog leaves a 55 s worst case under the 60 s client kill; no prompt content or credential value appears in any log line | VERIFIED | `TestOmpTextExitCodes`: whitespace text exits 1 with empty stdout; scripted `-32031` exits 1 and stderr names `-32031`; nonexistent install exits 1. Code: `textRunBudget=40s` context created before pre-flight, passed to `startBackend`/`Authenticate`; `textGenerateBudget=35s` child; `stop()` waits `shutdownWatchdog=15s`; live probe measured 3.95 s < 55 s. Log audit: success line carries ids/finishReason/authMethod/byte count only; prompt readers/errors name paths and sizes only; deny forwarder logs method/requestId/id key only |
| 7 | The coding-plan arm sends the rebound account provider: whenever authenticate records `AuthModeCodingPlan`, `params.selection.providerId` is `account:zai-individual-coding-plan` rather than `zai-api` - asserted in the request-assembly table and on the fake wire | VERIFIED | `TestOmpTextSelectionRebind` (table over `textRequest`: coding plan rebinds; api-key/none unchanged); `TestRunGenerateCodingPlanRebindWire` (fake wire sees the account provider id). `runOmpText` assembles the request only after `Authenticate`, via `textRequest(sel, b.front.AuthMode(), ...)`; grep confirms this is the sole `texttask.Request` construction path in production code |

**Score:** 7/7 truths verified (0 present, behavior-unverified)

### Documented Deviation (approved, not a gap)

- **`maxOutputTokens: 1024` is sent in the `workspace/generateText` params.** The plan text expected the key to be omitted. The first guarded live probe disproved that assumption on the real wire (`-32603: maxOutputTokens is outside the model option range`): the installed 3.18.2 app-server's strict model-option validator requires an explicit value for a custom `querySource` (only its special `git_commit_message` profile binds one). `internal/texttask/generate.go` now carries the top-level cap; the two tests that asserted absence assert `1024`; `tools`/`messages` remain forbidden. This is an evidence-backed in-scope correction that does not change the user-visible goal, and the orchestrator explicitly accepted it. The plan's must-have artifact wording (`no tools/messages keys`) is still satisfied.
- The other SUMMARY deviations are in-scope execution details, not scope changes: strict-linter refactors, idempotent `backend.stop` (prevents a second 15 s watchdog wait), and `main_test.go` arity/constant updates.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/texttask/prompt.go` | `@file` reading with BOM strip, ordered chunk concat, absolute-path and regular-file checks, 16 MiB cap, sentinel errors | VERIFIED | 132 lines; `ReadPromptArg` + `promptChunk`; absolute/regular/limit-reader checks; one BOM strip per chunk; total cap; errors carry path/sizes only |
| `internal/texttask/selection.go` | Configured-selection passthrough plus provider_config `defaultModelSelection` reader with actionable sentinels | VERIFIED | 86 lines; minimal decode surface, sentinels name the path and the `--model` fix, credentials ignored |
| `internal/texttask/generate.go` | Request builder, one `workspace/generateText` call over `zcode.Conn`, result decode, empty-text sentinel, no tools/messages keys | VERIFIED | 164 lines; map-assembled params, wrapper `querySource`, options only when level set, explicit `maxOutputTokens` (approved deviation), no tools/messages, `ErrEmptyText`, `WorkspaceRef` |
| `internal/texttask/auth.go` | `AuthMethod` decision and the coding-plan rebind mirroring `applyConfiguredModel` | VERIFIED | 59 lines; account/*, zai-api key/plan/none table; rebind only on `AuthModeCodingPlan` + `zai-api` |
| `internal/zcode/envelope.go` | `MethodWorkspaceGenerateText` and `MethodWorkspaceCancelGenerateText` constants (cancel unused in v1) | VERIFIED | Constants present at lines 162-163 with the v1-unused note |
| `cmd/zcode-acp/main.go` | omp text parse surface, `runOmpText` with run-level deadline, `textRequest`, backend lifecycle extraction, parameterized `wireResponder` | VERIFIED | `ompOptions` text fields, `ompFlagSet`, `ompParse`/`absorb`/`absorbPositional`, `textMode`/`validateMode`/`textLevel`, `runOmpText` branch, `textResolve`/`textWorkspace`/`authenticateText`/`generateText`/`finishText`/`textRequest`/`writeTextResult`, `providerConfigPath`, `startBackend`/`backend.stop`/`backend.Close`, forwarder-parameterized `wireResponder` |
| `internal/interact/permission.go` | Exported `DenyPermissionForwarder` | VERIFIED | Lines 239-250: structural log (method/requestId/id key) + `denyAnswer()`; `TestDenyPermissionForwarder` asserts the exact deny map |
| `cmd/zcode-acp/text_test.go` | Argv tables, prompt/selection tables, fake-transcript e2e (config-default, api-key fixture), auth arms, rebind table, exit codes, regressions | VERIFIED | 993 lines, all tests green; built-binary e2e uses a hermetic env |
| `cmd/zcode-acp/testdata/omp-text-e2e.json`, `omp-text-apikey-e2e.json` | Fake scenarios outside the golden corpus | VERIFIED | Both present under `cmd/zcode-acp/testdata`; strict-schema scenarios with one generateText reply (plus the api-key chain); no file added under `internal/zcode/fake/testdata/scenarios` |
| `test/realdesktop/texttask_test.go` | Guarded live probe using Orca's exact argv against the discovered real desktop | VERIFIED | 233 lines; `realdesktop` tag + `ZCODE_ACP_REAL_E2E=1` guard; exact Orca argv, optional `ZCODE_ACP_REAL_MODEL`/`ZCODE_ACP_REAL_THINKING`, real env, 60 s kill, <=55 s elapsed assertion, no `-32601`, no protocol frame |
| `README.md` | One troubleshooting bullet and one short recipe subsection for non-interactive text tasks | VERIFIED | Troubleshooting bullet "Empty chat or branch names from the omp card" points to "### 8. Non-interactive text tasks"; the subsection is 8 lines, documents argv, one-shot call, exit codes 0/1/2, stdout/stderr split |
| `experiments/README.md` | One quota-ledger row for the single tiny model call (flipped by the orchestrator) | VERIFIED | Row is PASSED with the exact command, exit 0, 3-byte stdout, 3.95 s, the default-path quota block, tier and no-capture note; no capture file added |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `ompParse` positional tokens | `texttask.ReadPromptArg` | `opts.prompts` -> `textResolve` | WIRED | E2E multi-chunk proves two BOM-prefixed `@file` tokens become the exact concatenated `params.prompt` |
| `--model` / config default | `texttask.ResolveSelection` -> `textRequest` -> `params.selection` | `parseConfig` + `provider_config.json` | WIRED | `TestOmpTextEndToEnd` (flag) and `TestOmpTextConfigDefault` (default ids + level) assert the wire; options omitted when level empty |
| `front.Authenticate` records `AuthModeCodingPlan` | `textRequest` rebind -> `params.selection.providerId == account:zai-individual-coding-plan` | `b.front.AuthMode()` after authenticate | WIRED | `TestOmpTextSelectionRebind` + `TestRunGenerateCodingPlanRebindWire`; `textRequest` is the sole production assembly path |
| `startBackend` conn + Front | `wireResponder` (PAT/static headers, cipher decrypt, official-MCP decline, deny permissions, nil asker) -> one generateText reply | `runOmpText` | WIRED | `wireResponder(..., interact.DenyPermissionForwarder, nil)`; e2e records exactly one reply; deny shape unit-tested |
| `result.text` | `os.Stdout` single write (+ one newline) while every log line stays on stderr | `writeTextResult` + stderr slog | WIRED | E2E asserts exact stdout and that stderr contains no protocol frame; success log carries ids/finishReason/bytes only |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `internal/texttask/generate.go` | `Result.Text` | App-server `workspace/generateText` reply over `zcode.Conn` (fake in tests, real desktop in the guarded probe) | Yes | FLOWING |
| `cmd/zcode-acp/main.go` | stdout bytes | `finishText` passes `res.Text` to `writeTextResult`; no static fallback or hardcoded text path | Yes | FLOWING |
| `internal/texttask/selection.go` | `ModelSelection` | `--model`/env or the desktop `provider_config.json` default; no hardcoded model fallback | Yes | FLOWING |
| `internal/texttask/prompt.go` | prompt string | argv literal / `@file` bytes | Yes | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build, vet, tagged vet | `go build ./... && go vet ./... && go vet -tags realdesktop ./test/realdesktop/` | all exit 0 | PASS |
| Text-task unit + e2e packages | `go test -race -count=1 ./internal/texttask/ ./internal/interact/ ./internal/zcode/fake/` | texttask 1.251 s, interact 2.393 s, fake 23.774 s, all ok | PASS |
| Focused cmd suite (text + ACP regressions) | `go test -race -count=1 ./cmd/zcode-acp/ -run 'TestOmpText\|TestOmpFlagSet\|TestOmpExitCodes\|TestOmpLaunchBinding\|TestOmpRootCoexistence\|TestRootModeUnchanged\|TestAgentBinaryEndToEnd\|TestWiringModeAwareServing' -v` | all pass; `TestOmpTextEndToEnd` 2.63 s; package ok 14.095 s | PASS |
| Whole-tree race suite | `go test -race -count=1 ./...` | every package ok (cmd 21.371 s, acpfront 107.326 s, texttask 1.389 s, fake 24.606 s; realdesktop has no default-tag test files) | PASS |
| Guarded live test skips unguarded | `env -u ZCODE_ACP_REAL_E2E go test -tags realdesktop -count=1 -run TestTextTaskLive -v ./test/realdesktop/` | `--- SKIP: TestTextTaskLive`, exit 0 | PASS |
| Pinned linter, default | `mise exec -- golangci-lint run` | version 2.15.0; rc 1 (findings), exactly 10 findings, all `internal/acpfront/*_test.go`; 0 elsewhere; no `level=error` | PASS |
| Pinned linter, realdesktop tag | `mise exec -- golangci-lint run --build-tags realdesktop` | 73 findings total, 0 in `test/realdesktop/texttask_test.go`; no `level=error` | PASS |
| Module/experiments gates | `mise run tidy-diff`; `mise run build`; `mise run vet`; `mise run test-experiments` | tidy-diff 0 (one `require` line); build/vet 0; experiments 24 pass, 1 guarded skip, 0 fail | PASS |

### Probe Execution

N/A - the project has no `scripts/*/tests/probe-*.sh` probe harness (only `scripts/vm-e2e.sh`), and the plan declares no probes. The runnable evidence is the test suite plus the guarded live probe above.

### Requirements Coverage

N/A - quick task 261124-154 declares no REQUIREMENTS.md IDs (`requirements-completed: []`). All plan success criteria are covered by the must-have truths above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| - | - | None | - | Debt-marker scan (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) over every changed file: zero hits. No empty implementations, no stub returns, no hardcoded-empty prop paths. Only stdout writes are the existing `--version` answers and the single `writeTextResult` write |

### Test Quality Audit

| Test File | Linked Requirement | Active | Skipped | Circular | Assertion Level | Verdict |
|-----------|--------------------|--------|---------|----------|-----------------|---------|
| `internal/texttask/*_test.go` | D2/D3/D4 | all | 0 | No | Value/behavioral (wire transcripts, sentinels, error non-echo) | Sufficient |
| `cmd/zcode-acp/text_test.go` | D1/D2/D3/D4/D5 | all | 0 | No | Behavioral (built binary, exact stdout/exit, transcript order) | Sufficient |
| `test/realdesktop/texttask_test.go` | D6 (guarded live) | 0 in CI | 1 by design | No | Behavioral (exit/stdout/elapsed) | Intended guard; orchestrator ran it guarded and it passed |

**Disabled tests on requirements:** 0 unexpected (the only skip is the by-design guarded live probe). **Circular patterns:** 0. **Insufficient assertions:** 0.

**Decision coverage:** N/A - no CONTEXT.md / `<decisions>` block exists for this quick task.

### Live Evidence (orchestrator-run human check)

- Command: `ZCODE_ACP_REAL_E2E=1 ZCODE_ACP_REAL_MODEL=OpenCodeGo/deepseek-v4.1-flash ZCODE_ACP_REAL_THINKING=max go test -tags realdesktop -count=1 -run TestTextTaskLive -v ./test/realdesktop/`
- Result: exit 0, 3-byte stdout (`ok\n`), 3.95 s elapsed, no `-32601`, no protocol frame, resolved install `unpacked-appimage` at `/home/nil/Applications/zcode-3.18.2/squashfs-root`; one tiny model call, no PAT mint.
- Default-selection attempt (no `--model`; `zai-api/GLM-5.3-Flash`, `reasoningLevel=max`): passed wrapper and app-server option validation and reached the provider, which returned `-32603 [1310] Weekly/Monthly Limit Exhausted` (resets 2026-11-27 18:05:02). This is an account quota block, not a wrapper failure; the wrapper's default-selection data flow is separately proven by `TestOmpTextConfigDefault`. A model-override attempt without `--thinking` was refused by the app-server's selection validator, which is a backend contract fact, not a wrapper defect; Orca's current OMP card offers only "Config default".
- The documented session-path fallback was not needed and remains a contingency requiring an explicit user decision and plan revision.

### Human Verification Required

N/A - the only human-tier check (the guarded live probe) was executed by the orchestrator and passed. The default-selection live response is currently blocked by the account quota (resets 2026-11-27), an external limitation already recorded in `experiments/README.md`; it is not a wrapper verification gap.

### Gaps Summary

None. All 7 must-have truths are verified by code inspection plus green unit/behavioral tests and the orchestrator-run live probe; all declared artifacts and key links are present and wired; the golden corpus and `go.mod` are untouched; the pinned linter reports only the 10/73 pre-existing findings; the guarded live test provably skips when unguarded. The single plan-text deviation (`maxOutputTokens: 1024`) is documented, evidence-backed, and approved as an in-scope correction.

---

_Verified: 2026-11-24T00:03:00Z_
_Verifier: Claude (gsd-verifier)_
