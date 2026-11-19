---
phase: "3"
slug: "zcode-back-leg-codec-fake-app-server-supervisor"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-11-19"
audited: "2026-11-20"
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Finalized 2026-11-20 by the Nyquist audit: every map row re-executed at HEAD (post e7646c5).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` under `go test -race`; helper-process reexec for the standalone-binary tier |
| **Config file** | none new — existing `mise.toml` (`[tasks.test] run = "go test -race -count=1 ./..."`) |
| **Quick run command** | `go test -race ./internal/zcode/... ./internal/supervisor/... ./internal/turn/...` |
| **Full suite command** | `mise run ci` (build → vet → lint → test → test-experiments) |
| **Estimated runtime** | ~90 seconds (measured at audit: ~30s wall; test stage 5.6s) |

---

## Sampling Rate

- **After every task commit:** `go test -race ./internal/...` (packages touched)
- **After every plan wave:** `mise run ci`
- **Before `/gsd:verify-work`:** full suite green + golden corpus replayed + fake-binary reexec test green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

All 23 tasks across 9 plans carry automated verify. Every status below is from the
audit re-execution at HEAD (2026-11-20); RED-state evidence for TDD tasks lives under
`evidence/` and was classified RED_EVIDENCE_OK before the corresponding feat commits.

| Task ID | Plan | Wave | Requirement | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01 T1 | 03-01 | 1 | BRIDGE-01/02 | tracer RED: round-trip fails on assertion, not build/panic (RED_EVIDENCE_OK) | integration (-race) | `go test -race -count=1 -run TestHappyTurnRoundTrip ./internal/zcode/fake/` | ✅ | ✅ green |
| 03-01 T2 | 03-01 | 1 | BRIDGE-01 | envelope cannot carry framing key (typed struct); two-lane writer; type-tagged pending table | unit (-race) | `go test -race -count=1 -run TestHappyTurnRoundTrip ./internal/zcode/fake/` + `go build ./... && mise run vet` | ✅ | ✅ green |
| 03-01 T3 | 03-01 | 1 | BRIDGE-01/02 | demux WR-02 route; fake Serve; golden replay byte-deterministic, missing fixture fails | golden/harness | `go test -race -count=1 ./internal/zcode/...` + `mise run ci` | ✅ | ✅ green |
| 03-02 T1 | 03-02 | 2 | BRIDGE-01 | WR-02 collision never resolves pending; id value+type correlation; 3-way classification; 128 KiB line / malformed skip | unit (-race) | `go test -race -count=1 -run 'TestDemux\|TestLineBuffer' ./internal/zcode/` | ✅ | ✅ green |
| 03-02 T2 | 03-02 | 2 | BRIDGE-01 | single-writer ordering (32 concurrent, zero interleave); bypass precedence (stop ≤ 2nd frame); conn.Stop | unit (-race) | `go test -race -count=1 -run 'TestWriterOrdering\|TestStopJumpsQueue' ./internal/zcode/` | ✅ | ✅ green |
| 03-02 T3 | 03-02 | 2 | BRIDGE-01 | D-05 router: rtp answered under injectable deadline; credential-cipher tracked-no-answer (zero reply bytes, recorded expiry); unknown -32601 | unit (-race) | `go test -race -count=1 -run 'TestRouter' ./internal/zcode/` | ✅ | ✅ green |
| 03-03 T1 | 03-03 | 2 | BRIDGE-02 | fail-loud loader: field-naming rejections, strict decode, termination contract | unit (-race) | `go test -race -count=1 -run 'TestLoadScenario' ./internal/zcode/fake/` | ✅ | ✅ green |
| 03-03 T2 | 03-03 | 2 | BRIDGE-02 | 10-scenario corpus replays byte-deterministically; exit-0-zero-lines is FAILURE (in-process + exec child) | golden/integration | `go test -race -count=1 ./internal/zcode/fake/` (+ 10 scenarios / 10 goldens on disk) | ✅ | ✅ green |
| 03-04 T1 | 03-04 | 1 | BRIDGE-03 | discovery contract RED: fixtures, diagnostics, order, decoys, revision (RED_EVIDENCE_OK) | unit (-race) | `go test -race -count=1 -run 'TestDiscover\|TestBuiltinRevision' ./internal/supervisor/` | ✅ | ✅ green |
| 03-04 T2 | 03-04 | 1 | BRIDGE-03 | ordered first-match-wins probes; empty result names all 8 probes; /tmp/.mount_ rejected; sha256(path-string) revision | unit (-race) | `go test -race -count=1 ./internal/supervisor/` + `mise run lint` | ✅ | ✅ green |
| 03-05 T1 | 03-05 | 2 | BRIDGE-04 | turn machine RED: exactly-once Run contract + watcher delivery/stop (RED_EVIDENCE_OK) | unit (-race) | `go test -race -count=1 -run 'TestHappyTurn\|TestTurnWatcher' ./internal/turn/` | ✅ | ✅ green |
| 03-05 T2 | 03-05 | 2 | BRIDGE-04 | PollWatcher quiet-window gates; exactly-once completion; no synthesized resolution | unit (-race) | `go test -race -count=1 ./internal/turn/` | ✅ | ✅ green |
| 03-05 T3 | 03-05 | 2 | BRIDGE-04 | terminal / idle-patch / quiet-window each resolve exactly once; carrier-tolerant classification (turnId-matched) | unit (-race) | `go test -race -count=1 -run 'TestCompletionVariants' ./internal/turn/` | ✅ | ✅ green |
| 03-06 T1 | 03-06 | 2 | BRIDGE-02 | cmd/fake-app-server: --scenario required, fail-loud exit 2, stdout=frames only, EOF-drain exit 0 | smoke (build+lint) | `go build ./... && go vet ./...` + `mise run lint` | ✅ | ✅ green |
| 03-06 T2 | 03-06 | 2 | BRIDGE-02 | reexec child over real os.Pipe: frames both directions, Setpgid, observable drain, zero-lines = FAILURE at exec tier | integration (-race) | `go test -race -count=1 -run 'TestFakeBinary\|TestEOFSignature' ./internal/zcode/fake/` | ✅ | ✅ green |
| 03-07 T1 | 03-07 | 3 | BRIDGE-03 | spawn contract on a real child: argv [binary, entrypoint, app-server, --stdio]; allowlisted env, proxy vars scrubbed (observed, not intended); Setpgid | integration (-race) | `go test -race -count=1 -run 'TestSpawn' ./internal/supervisor/` | ✅ | ✅ green |
| 03-07 T2 | 03-07 | 3 | BRIDGE-03 | staged shutdown: group SIGKILL leaves zero survivors incl. grandchild (ESRCH poll); 3/60s sliding-window restart, reset on stable run | integration (-race) | `go test -race -count=1 -run 'TestGroupKill\|TestRestartWindow' ./internal/supervisor/` | ✅ | ✅ green |
| 03-07 T3 | 03-07 | 3 | BRIDGE-03 | realdesktop positive path behind `-tags realdesktop` + `ZCODE_ACP_REAL_E2E=1`; default build/vet/CI unaffected | build-guard + manual (DISCHARGED — see Manual-Only table) | `go build ./... && go vet ./...` (+ `-tags realdesktop` variant) | ✅ | ✅ green |
| 03-08 T1 | 03-08 | 3 | BRIDGE-04 | cancel + double-send RED: cancel resolves wrong reason / raw CallError surfaced (RED_EVIDENCE_OK) | unit (-race) | `go test -race -count=1 -run 'TestCancel\|TestDoubleSend' ./internal/turn/` | ✅ | ✅ green |
| 03-08 T2 | 03-08 | 3 | BRIDGE-04 | bypass-lane Cancel exactly-once, idempotent; -32010 mapped by numeric code (never message text) | unit (-race) | `go test -race -count=1 -run 'TestCancel\|TestDoubleSend\|TestHappyTurn\|TestCompletionVariants' ./internal/turn/` | ✅ | ✅ green |
| 03-08 T3 | 03-08 | 3 | BRIDGE-04 | corpus completion (idle-patch, quiet-window) via real Conn; credential-cipher expiry recorded, degraded StopReason, zero fabricated answers | integration (-race) | `go test -race -count=1 -run 'TestCompletionCorpus\|TestCredentialCipher' ./internal/turn/` | ✅ | ✅ green |
| 03-09 T1 | 03-09 | 4 | BRIDGE-01..04 | exec-chain E2E: supervisor spawns BUILT fake binary → Conn over exec pipes → machine; happy + cancel exactly-once; ESRCH ×2 | integration (-race) | `go test -race -count=1 -run 'TestEndToEndExecChain' ./internal/supervisor/` | ✅ | ✅ green |
| 03-09 T2 | 03-09 | 4 | all | phase gate: build/vet/lint/test/test-experiments green; zero dependency drift | gate | `mise run ci && mise run tidy-diff` | ✅ | ✅ green |

*Status: ✅ green · ❌ red · ⚠️ flaky. Seeded placeholder rows (planner guesses `TestEnvelope`, `TestBypass`) repaired to the real test names: envelope pin = `TestEncodeOmitsJSONRPC`/`TestDecodeRoundTrip`/`TestIDKeyTypeTagging` (internal/zcode/roundtrip_test.go), bypass = `TestStopJumpsQueue` (stop AND reverse-request replies share the one bypass lane — writer.go `enqueueBypass`).

---

## Wave 0 Requirements

- [x] Go package skeletons: `internal/zcode`, `internal/zcode/fake`, `internal/supervisor`, `internal/turn` (+ `-tags realdesktop` guard)
- [x] Golden-corpus scenario files under `testdata/` (10 scenarios per 03-RESEARCH corpus table — 10 JSONs + 10 goldens verified on disk)
- [x] No framework installs: stdlib `testing` only; go.mod declares only the module + `go 1.27` lines; `mise run tidy-diff` green at audit

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions | Status |
|----------|-------------|------------|-------------------|--------|
| Real-desktop positive path (locate → spawn → create → close, no orphans) | BRIDGE-03 | Requires the real desktop install; env-guarded, never CI | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop -count=1 -run TestRealDesktop ./internal/supervisor/` on a machine with ZCode installed (this machine qualifies) | ✅ EXECUTED — 2026-11-19 (executor discharge): ok 2.529s; re-executed at audit 2026-11-20: PASS 2.32s with observed `layout=unpacked-appimage root=~/Applications/zcode-3.18.2/squashfs-root`, `builtinRevision=zcode-builtin:44:e9c0674a…`, session/create ok (runtime-preferences answered by the router), `kill(-pgid, 0)` ESRCH (zero orphans). Guard verified: without the env var the tagged test skips in 0.00s; default suite has zero skips and never compiles the file. |

No other manual verifications remain for this phase.

---

## Audit Re-Execution Record (2026-11-20, at HEAD e7646c5)

| Gate | Result |
|------|--------|
| `mise run ci` | exit 0 — build, vet, lint 0 issues, `go test -race -count=1 ./...` all ok, test-experiments 22/22 |
| `go test -race -count=1 ./internal/zcode/fake/` | ok 5.150s |
| `go test -race -count=1 ./internal/turn/` | ok 2.729s |
| `go test -race -count=1 ./internal/supervisor/` | ok 1.839s |
| `go test -race -count=1 ./internal/zcode/` | ok 2.337s |
| All 17 per-task pinned `-run` commands (rows above) | ok, one at a time |
| `mise run tidy-diff` | exit 0 (no go.mod/go.sum drift) |
| `bash experiments/check-docs.sh` | exit 0 |
| `bash experiments/scan-secrets.sh` | **exit 1 — ESCALATED (see below)** |
| Skip inventory | 0 skips in default `go test -race -v ./internal/...`; only TestRealDesktop skips, and only under `-tags realdesktop` without the env var (by design) |

### Escalated finding (BLOCKER for CI, not a Nyquist-structure gap)

`bash experiments/scan-secrets.sh` is red at HEAD: it exits 1 on the credential-blob
value shape `enc:v2:os:…` in two Phase-3 fixtures —
`internal/zcode/fake/testdata/scenarios/credential-cipher-scripted.json` and its
recorded golden — whose synthetic marker value `enc:v2:os:fix1-synthetic-not-a-real-credential`
deliberately reproduces the capture-verbatim credential SHAPE (that shape is the fixture's
point per 03-03). The script is wired as the REQUIRED `secret-scan` check in
`.github/workflows/pr-sanity.yml:132`, so push CI fails at HEAD. The red went unnoticed
because `mise run ci` runs `node --test experiments/` (the .mjs suite), not the shell
gates, and no Phase-3 summary re-ran `scan-secrets.sh` after commit 17bf12e landed the
fixtures. Resolution is a developer call (allowlist the synthetic marker or exclude
`internal/zcode/fake/testdata/` from the value-shape scan in `scan-secrets.sh`; changing
the fixture value to dodge the regex would weaken the corpus spec) — implementation files
are outside the auditor's write scope. Not fixed here; escalated.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies (23/23 tasks, all 9 plans)
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (every task has one)
- [x] Wave 0 covers all MISSING references (all packages, fixtures, and the realdesktop guard exist)
- [x] No watch-mode flags (stdlib testing only; no watch flags in mise.toml or tests)
- [x] Feedback latency < 90s (full suite ~6s; `mise run ci` ~30s wall at audit)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated — 2026-11-20 Nyquist audit. All 23 task gates re-executed green
at HEAD; the sole open item is the escalated `scan-secrets.sh` × synthetic-fixture conflict
above (CI `secret-scan` red), which is a tooling/fixture policy decision for the developer,
not an unverified phase requirement.
