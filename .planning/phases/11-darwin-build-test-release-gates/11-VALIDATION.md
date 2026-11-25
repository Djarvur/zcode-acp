---
phase: "11"
slug: "darwin-build-test-release-gates"
status: draft
nyquist_compliant: true
wave_0_complete: false
created: "2026-11-25"
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` plus Node `node:test` |
| **Config file** | `mise.toml`, `.github/workflows/pr-sanity.yml`, `.github/workflows/release.yml` |
| **Quick run command** | `go test ./internal/supervisor -run 'TestMountStop\|TestMountHappyPath' -count=1` |
| **Full suite command** | `go test -race -count=1 ./... && node --test experiments/` |
| **Tool-version gate** | `mise exec -- node scripts/check-mise-tool-versions.mjs [tool ...]` |
| **Cross-build commands** | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...` and `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...` |
| **Release verification** | In each release job: bounded `mise exec -- go mod download` from `https://proxy.golang.org`, unchanged go.mod/go.sum check, then `GOPROXY=off mise exec -- go mod verify` and `GOPROXY=off mise run tidy-diff`; followed by `GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish` and `mise exec -- bash scripts/verify-darwin-release.sh dist` |
| **Estimated runtime** | To be measured on the first native macOS run |

The tool-version gate reads required versions only from `mise.toml`, probes selected tools through `mise exec`, prints the resolved versions, and exits non-zero on a failed probe or mismatch. `experiments/tool-versions.test.mjs` proves the comparator rejects wrong exact pins and wrong required lines without invoking mise.

---

## Sampling Rate

- **After every task commit:** Run the affected Go package tests and `node --test experiments/portability.test.mjs` once it exists.
- **After the Statfs change:** Run both exact Darwin cross-build commands.
- **After every plan wave:** Run the full native `-race` suite and Node experiment suite on each supported runner.
- **Before any CI build/test job:** Run the mise-driven version gate; a mismatch must fail before compilation or tests start.
- **Before any release job's offline build:** Complete the five-minute frozen-module download and unchanged go.mod/go.sum check, then pass offline `go mod verify` and `tidy-diff`; an unavailable module fails at this dependency gate, never later in GoReleaser.
- **Before `/gsd:verify-work`:** Both Darwin cross-builds, Linux/macOS native gates, shell gates, tool-version gates, and release snapshot verification must be green.
- **Max feedback latency:** Target under 120 seconds for package-level feedback; full-suite latency is measured during Wave 0.

---

## Per-Task Verification Map

Regenerated for the final 12-task structure: 2 + 3 + 2 + 2 + 3 tasks across Plans 01-05. Every task ends with a passing automated gate; Plan 02 Task 1 is the sole RED-evidence task and ends with the machine classifier's passing `RED_EVIDENCE_OK` gate.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | Available Inputs | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|------------------|--------|
| 11-01-01 | 01 | 1 | MBLD-01 | T-11-01, T-11-SC | Normalize Darwin's filesystem-type width at the existing int64 seam and prove arm64 compilation | unit/build | `CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./...`<br>`go test -race -count=1 ./internal/supervisor -run '^TestDefaultStatfs'` | Task 1 creates the converter and direct tests; pinned Go/module are declared preconditions | ⬜ pending |
| 11-01-02 | 01 | 1 | MBLD-01 | T-11-01, T-11-SC | Prove amd64 compilation and preserve native Linux mount behavior without dependency drift | build/regression | `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...`<br>`go test -race -count=1 ./internal/supervisor -run 'TestMount\|TestDefaultStatfs'`<br>`git diff --exit-code HEAD -- go.mod go.sum` | Task 1 converter/tests; existing mount suite and frozen module files | ⬜ pending |
| 11-02-01 | 02 | 1 | MTEST-01, MBLD-02 | T-11-SC | Capture both intended fixture assertion failures and finish with a passing machine-classified RED-evidence gate | static/TDD | `gsd_run check tdd-red-evidence .planning/tdd/11-02-task1-red.json --raw` | Task action creates the static test, captures the complete failing TAP report, and records the real exit status before the classifier runs | ⬜ pending |
| 11-02-02 | 02 | 1 | MBLD-02, MTEST-01 | T-11-02, T-11-SC | Replace the stubborn mount cleaner with a bounded detached Go helper and unconditional PID cleanup | static/integration | `node --test --test-reporter=tap --test-name-pattern='mount fixture' experiments/portability.test.mjs`<br>`go test -race -count=1 ./internal/supervisor -run '^TestMountStop$'` | Task 1 static contract; existing `TestMountStop` and re-exec pattern | ⬜ pending |
| 11-02-03 | 02 | 1 | MBLD-02, MTEST-01 | T-11-03, T-11-SC | Preserve inherited wrapper stderr with nested Go helpers and bounded recorded-PID cleanup | static/integration | `node --test --test-reporter=tap experiments/portability.test.mjs`<br>`go test -race -count=1 ./cmd/zcode-acp -run '^TestAppImageMountSignal$'` | Task 1 static contract; existing readiness-teardown oracle and PID cleanup | ⬜ pending |
| 11-03-01 | 03 | 2 | MBLD-02 | T-11-05, T-11-SC | Replace GNU-only docs scans, pin managed Bash, and pass both shell gates with unchanged output/exit contracts | static/config/runtime gate | `node --test experiments/portability.test.mjs`<br>`mise exec -- bash -n experiments/check-docs.sh && mise run check-docs && mise run secret-scan` | Task 1 implements the portable scans, Bash pin/tasks, and passing static contract together; trusted mise/pinned tools are declared preconditions | ⬜ pending |
| 11-03-02 | 03 | 2 | MBLD-02 | T-11-04, T-11-SC | Check all five mise-resolved tools against single-source pins and fail non-zero on drift | unit/config/runtime gate | `node --test experiments/tool-versions.test.mjs`<br>`node --test experiments/portability.test.mjs`<br>`mise exec -- node scripts/check-mise-tool-versions.mjs` | Task 1 managed tool graph; Task 2 creates the gate, offline comparators, and passing static assertions; trusted mise/pinned tools are declared preconditions | ⬜ pending |
| 11-04-01 | 04 | 3 | MBLD-03, MTEST-01 | T-11-06, T-11-07, T-11-08, T-11-SC | Add a native arm64 CI path with architecture and mise-resolved version assertions before canonical gates | static CI contract | `node --test experiments/portability.test.mjs`<br>`git diff --check HEAD -- .github/workflows/pr-sanity.yml experiments/portability.test.mjs` | Plans 01 and 03 complete; Task 1 edits the workflow and static contract | ⬜ pending |
| 11-04-02 | 04 | 3 | MBLD-03 | T-11-06, T-11-07, T-11-08, T-11-SC | Add Intel macOS, mise-resolved Darwin cross-builds, and managed shell gates without renaming required checks | static CI contract | `node --test experiments/portability.test.mjs`<br>`git diff --check HEAD -- .github/workflows/pr-sanity.yml experiments/portability.test.mjs` | Task 1 arm64 job; Plans 01 and 03 version gate/tool graph | ⬜ pending |
| 11-05-01 | 05 | 4 | MBLD-03 | T-11-09, T-11-10, T-11-11, T-11-14, T-11-SC | Build a real binary-only arm64/amd64/universal snapshot and pass checksum, exact-member, adversarial, and architecture verification before workflow expansion | release tracer/static/negative fixtures | `node --test experiments/release.test.mjs`<br>`ZCODE_ACP_RELEASE_FIXTURES=1 mise exec -- node --test --test-name-pattern='artifact verifier' experiments/release.test.mjs`<br>`mise exec -- bash -n scripts/verify-darwin-release.sh && GOPROXY=off mise exec -- node scripts/check-mise-tool-versions.mjs go goreleaser bash && GOPROXY=off mise exec -- goreleaser check`<br>`GOPROXY=off mise exec -- goreleaser release --snapshot --clean --skip=publish && mise exec -- bash scripts/verify-darwin-release.sh dist` | Plans 01 and 03; Task 1 creates config/verifier/tests/ignore rule; frozen module and exact Apple/pinned tools are declared preconditions | ⬜ pending |
| 11-05-02 | 05 | 4 | MBLD-03 | T-11-09, T-11-12, T-11-13, T-11-14, T-11-SC | Automate the proven snapshot as a read-only macOS workflow with bounded frozen-module hydration, offline checksum/tidy verification, and version/tool preflights | static CI contract | `node --test experiments/release.test.mjs`<br>`git diff --check HEAD -- .github/workflows/release.yml experiments/release.test.mjs`<br>`GOPROXY=off mise exec -- node scripts/check-mise-tool-versions.mjs go goreleaser bash && GOPROXY=off mise exec -- goreleaser check` | Task 1 passing tracer; Task 2 adds bounded online module download plus offline verify/tidy before the snapshot workflow and its passing static assertions; no pre-populated cache is assumed | ⬜ pending |
| 11-05-03 | 05 | 4 | MBLD-03 | T-11-12, T-11-13, T-11-14, T-11-SC | Gate tag-only publication on the verified snapshot, independently hydrate and verify the frozen modules, and confine write rights/token to that job | static CI contract | `node --test experiments/release.test.mjs`<br>`git diff --check HEAD -- .github/workflows/release.yml experiments/release.test.mjs`<br>`GOPROXY=off mise exec -- node scripts/check-mise-tool-versions.mjs go goreleaser bash && GOPROXY=off mise exec -- goreleaser check` | Task 2 snapshot workflow and module gate; Task 3 repeats the bounded hydration/offline verification in the publish job and adds tag scope assertions; no tag is pushed during verification | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Add a direct production `defaultStatfs` converter test if the injected mount tests do not exercise the Darwin-width conversion.
- [ ] Replace both external `setsid` fixture call sites with re-executed Go test helpers using `syscall.SysProcAttr{Setsid: true}`.
- [ ] Add `experiments/portability.test.mjs` to reject executable `setsid` fixture calls and assert portable docs scans plus managed Bash shell gates.
- [ ] Add `scripts/check-mise-tool-versions.mjs` and offline comparator tests; every CI/release version preflight must use the non-zero-on-mismatch gate.
- [ ] Add managed Bash to `mise.toml` and route shell-gate workflow steps through it.
- [ ] Add native macOS arm64/amd64 CI jobs and the mise-resolved two-command Darwin cross-build job.
- [ ] Add `.goreleaser.yaml`, `scripts/verify-darwin-release.sh`, grouped release tests with guarded unsafe fixtures, and prove the binary-only snapshot before adding the macOS workflow.
- [ ] Resolve the pinned Go toolchain, ACP SDK module, GoReleaser, and golangci-lint versions before claiming the phase gate green.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| GitHub-hosted macOS runner labels, architecture, and cold module acquisition | MBLD-03 | Runner availability, network acquisition, and cache contents are external service state | Observe the first macOS matrix run and a release job with a cold module cache; record runner OS/arch, resolved tool versions, the bounded frozen-module download result, and the following offline checksum/tidy gate. |
| GitHub-hosted release snapshot upload and tag-publish scope | MBLD-03 | Workflow execution and GitHub artifact/release state are external service state | Dispatch the snapshot, confirm the three archives plus `checksums.txt` upload, and inspect the tag job's `contents: write` scope without publishing an unapproved tag. |

GoReleaser 2.20.2 schema compatibility and exact tool versions are automated by Tasks 11-03-02, 11-05-01, 11-05-02, and 11-05-03; they are not manual-only checks.

---

## Validation Sign-Off

- [x] All implementation tasks have runnable `<automated>` verification grounded in task outputs or declared predecessors.
- [x] Tool-version gates compare mise-resolved outputs with `mise.toml` and fail non-zero on mismatch.
- [x] Sampling continuity: no three consecutive tasks lack automated verification.
- [x] Wave 0 covers every missing artifact referenced by an implementation task.
- [x] No watch-mode flags are used.
- [ ] Feedback latency target measured and recorded during execution.
- [x] `nyquist_compliant: true` reflects the revised automated gates.

**Approval:** planning revision complete; runtime sign-off pending execution.
