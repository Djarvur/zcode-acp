---
phase: "5"
slug: "interactions-full-chat-cycle-api-key"
status: ready
nyquist_compliant: true
wave_0_complete: false
created: "2026-11-20"
finalized: "2026-11-20 (planner — per-task rows below; Wave-0 gaps absorbed into plan first-tasks)"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract. Seeded from 05-RESEARCH.md §Validation Architecture (planner finalizes per-task rows).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib testing + -race; golden corpus (fake scenarios + .golden transcripts); table-driven TestGolden; TestEOFSignature fail-loud |
| **Config file** | none — canonical gate order build→vet→lint→test→test-experiments via mise (Phase-2 lock) |
| **Quick run command** | `go test -race ./internal/interact/... ./internal/acpfront/... ./internal/acpport/... ./internal/zcode/...` |
| **Full suite command** | `mise run ci` |
| **Estimated runtime** | ~120 seconds |

---

## Sampling Rate

- **After every task commit:** quick run command (touched packages)
- **After every plan wave:** `mise run ci`
- **Phase gate:** full suite green + full-cycle fake E2E green + guarded real-tier run ledgered before verify-work
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

Planner-finalized from the 17-row RESEARCH contract. Every row's command runs < 120 s;
the guarded real-tier row is the manual-only exception (double guard, never CI).

| Task ID | Plan/Task | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|-----------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| V01 | 05-01 T2 | 1 | CORE-06 | T-5-02 | unknown interaction/* → -32601, never crash | unit (TestResponderContract) | `go test -race -count=1 ./internal/interact/` | 05-01 Task 2 creates | ⬜ |
| V02 | 05-01 T2 | 1 | CORE-06 | T-5-01 | per-method deadlines answered/recorded (expiry deny/coded error + RouterEvent) | unit (TestResponderContract staged) | `go test -race -count=1 ./internal/interact/` | 05-01 Task 2 creates | ⬜ |
| V03 | 05-01 T1+T2 | 1 | CORE-06 | T-5-01 | re-announce dedupe by requestId (new wire id per re-announce, first-reply-wins) | fake golden (TestGolden/reannounce-dedupe) | `go test -race -count=1 -run 'TestGolden/reannounce-dedupe' ./internal/zcode/fake/` | 05-01 Task 1 creates | ⬜ |
| V04 | 05-01 T2 | 1 | CORE-06 | — | runtime-prefs answer migrates byte-identical | unit staged assert (conn.go:384-387 bytes) | `go test -race -count=1 ./internal/interact/` | exists (scenario re-driven through the responder) | ⬜ |
| V05 | 05-04 T2 | 3 | TOOL-01 | T-5-10/T-5-11 | tool.updated scheduled/started/progress/result → ordered tool_call/update + fields | fake golden + composed (TestToolLifecycle) | `go test -race -count=1 -run 'TestToolLifecycle' ./internal/acpfront/ && go test -race -count=1 -run 'TestGolden/tool-lifecycle' ./internal/zcode/fake/` | 05-04 Task 2 creates | ⬜ |
| V06 | 05-04 T1/T2 | 3 | TOOL-01 | T-5-11 | edit-tool diff rendering (Edit/Write inputs; absolute paths; new-file omission) | unit + composed (ToolNote table + TestToolLifecycle Write leg) | `go test -race -count=1 ./internal/acpport/ -run 'TestToolNote' && go test -race -count=1 -run 'TestToolLifecycle' ./internal/acpfront/` | 05-04 Tasks 1-2 create | ⬜ |
| V07 | 05-02 T2 | 2 | TOOL-02 | T-5-04 | permission options map verbatim + outcome forwarded as embedded response | composed (TestPermissionBridge) + backend unit arms | `go test -race -count=1 -run 'TestPermissionBridge' ./internal/acpfront/ ./internal/interact/` | 05-02 Task 2 creates | ⬜ |
| V08 | 05-02 T3 | 2 | TOOL-02 | T-5-04/T-5-05 | ACP cancel → -32021 reply-once; deadline expiry → deny; unknown optionId → deny | composed + fake golden (TestPermissionCancel/Expiry + permission-cancel golden) | `go test -race -count=1 -run 'TestPermissionCancel|TestPermissionExpiry' ./internal/acpfront/ && go test -race -count=1 -run 'TestGolden/permission-cancel' ./internal/zcode/fake/` | 05-02 Task 3 creates | ⬜ |
| V09 | 05-06 T1+T2 | 5 | AUTH-01 | T-5-14/T-5-16 | snapshot push shape (formula reuse, empty overlay) + read-back verdict (ack negative locked) | unit (TestSnapshotEnvelope/TestAuthReadback) + fake golden | `go test -race -count=1 ./internal/hostsvc/ && go test -race -count=1 -run 'TestGolden/auth-apikey' ./internal/zcode/fake/` | 05-06 Tasks 1-2 create | ⬜ |
| V10 | 05-06 T2 | 5 | AUTH-01 | T-5-15 | runtime-headers reply schema vector (failure-shaped success never emitted) | fake golden (TestGolden/runtime-headers-answer) + unit arms | `go test -race -count=1 -run 'TestGolden/runtime-headers-answer' ./internal/zcode/fake/ && go test -race -count=1 -run 'TestRuntimeHeaders' ./internal/hostsvc/` | 05-06 Task 2 creates | ⬜ |
| V11 | 05-05 T2 | 4 | SESS-01 | T-5-12 | resume = backend resume + re-subscribe + no replay | composed (TestResume) | `go test -race -count=1 -run 'TestResume' ./internal/acpfront/` | 05-05 Task 2 creates | ⬜ |
| V12 | 05-05 T2 | 4 | SESS-01 | T-5-12/T-5-13 | close = cancel→close→Remove→removes (observer-count baseline; race probe; clean unknown-id rejection) | composed (TestCloseTeardown) | `go test -race -count=1 -run 'TestCloseTeardown' ./internal/acpfront/` | 05-05 Task 2 creates | ⬜ |
| V13 | 05-03 T1 | 2 | SESS-01 (WR-01) | T-5-07 | router-event fan-out (A then B; removal semantics; hook theft gone) | unit (TestRouterFanout, zcode+turn) | `go test -race -count=1 -run 'TestRouterFanout' ./internal/zcode/ ./internal/turn/` | 05-03 Task 1 creates | ⬜ |
| V14 | 05-02 T1 | 2 | SESS-01 (WR-02) | — | sink published atomically (atomic.Pointer) | unit (-race; existing suite re-run) | `go test -race -count=1 ./internal/acpport/` | exists (05-02 Task 1 extends) | ⬜ |
| V15 | 05-05 T3 | 4 | SESS-01 (WR-05) | — | scope-accurate concurrency rejection message (locked constant update) | composed (TestConcurrent/Cancel constants) | `go test -race -count=1 -run 'TestConcurrent|TestCancel' ./internal/acpfront/` | exists (05-05 Task 3 updates) | ⬜ |
| V16 | 05-08 T2 | 7 | CORE-06/TOOL-01/TOOL-02/SESS-01 | — | full cycle new → prompt(tool+permission) → resume → close, fake tier | composed E2E (TestFullCycle) | `go test -race -count=1 -run 'TestFullCycle' ./internal/acpfront/` | 05-08 Task 2 creates | ⬜ |
| V17 | 05-09 T1 | 8 | AUTH-01/TOOL/SESS/CORE-06 | T-5-21/T-5-22 | real tier: same cycle on api-key (double-guarded, manual) | manual + tagged test | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop ./test/realdesktop/ -run TestAPIKeyFullCycle` (+ `go test -tags realdesktop ./test/realdesktop/` must SKIP without env) | 05-09 Task 1 creates (guarded) | ⬜ |

Supplementary rows (plan-specific locks beyond the 17-row contract):

| Task ID | Plan/Task | Requirement | Automated Command | Notes |
|---------|-----------|-------------|-------------------|-------|
| S01 | 05-07 T1 | AUTH-01 (D-06 hygiene) | `go test -race -count=1 ./internal/interact/ && bash experiments/scan-secrets.sh` | cipher vectors + hygiene locks (no key material in logs/argv/env) |
| S02 | 05-07 T2 | AUTH-01 | `go test -race -count=1 -run 'TestAuthenticate' ./internal/acpfront/` | honest chain, no-key error, interruptibility probe |
| S03 | 05-07 T3 | AUTH-01 (wiring) | `go test -race -count=1 -run 'TestAgentBinaryEndToEnd' ./cmd/zcode-acp/` | binary-level responder wiring through real stdio |
| S04 | 05-04 T3 | TOOL-01 (WR-03) | `go test -race -count=1 -run 'TestStreamBackpressure' ./internal/acpfront/` | bounded pump, coalescing, ceiling fail-loud |
| S05 | 05-02 T3 | CORE-06 | `go test -race -count=1 -run 'TestGolden/user-input-decline' ./internal/zcode/fake/` | requestUserInput decline on the wire |

---

## Wave 0 Requirements

Absorbed into plan first-tasks (no separate Wave-0 plan needed — each is the opening
task of the plan that owns it):

- [x→plan] internal/interact package skeleton (responder policy object) → 05-01 Task 2 (tracer, RED-first)
- [x→plan] internal/hostsvc skeleton (api-key snapshot + registry read-back helper) → 05-06 Tasks 1-2
- [x→plan] fake re-announce reverse-request extension → 05-01 Task 1
- [x→plan] CredentialDecrypt seam interface (secret-tool transport; godbus stays Phase 6) → 05-07 Task 1

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real api-key full cycle (new → prompt with tool + permission → resume → close) | AUTH-01/TOOL/SESS criteria | Requires real account + real desktop install; double guard (tag + env), never CI | `ZCODE_ACP_REAL_E2E=1 go test -tags realdesktop ./test/realdesktop/ -run TestAPIKeyFullCycle` — quota ledgered in experiments/README (05-09 lands the test + ledger; the REAL RUN is the end-of-phase verification act, orchestrator-run) |

---

## Validation Sign-Off

- [x] All tasks have automated verify or Wave 0 deps (Wave-0 gaps absorbed into plan first-tasks; V17/S-rows carry their guards)
- [x] Sampling continuity holds (per-task quick runs → per-wave `mise run ci` → phase gate = full suite + full-cycle fake E2E + ledgered real-tier evidence)
- [x] Wave 0 covers MISSING refs (mapped above)
- [x] No watch-mode flags
- [x] Feedback latency < 120s (quick-run command scope per task; full suite ~120 s)
- [x] nyquist_compliant: true in frontmatter

**Approval:** planner-finalized 2026-11-20 (execution sign-off lands per-plan in SUMMARYs)
