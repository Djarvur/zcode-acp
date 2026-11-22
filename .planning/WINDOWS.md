---
schema_version: 1
open_count: 17
waived_count: 0
fixed_count: 1
total_count: 18
last_updated: 2026-11-22T14:45:18.399Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 01 | deviation | experiments/captures/EXP-TURN-01.jsonl | 1 | E2 turn-completion terminal signals are V-SRC-only (D-05 degradation): no completion envelope observed live; plan 01-03 must label affected claims V-SRC-only/V-RPT | open |  | 2026-11-19T06:32:58.823Z |  |
| 2 | 01 | deviation | experiments/README.md |  | Quota: 3 tiny sends issued vs 1 planned (two harness quiet-window mis-arms, fixed in-plan); 0 turns completed | open |  | 2026-11-19T06:32:59.065Z |  |
| 3 | 02 | stub | cmd/zcode-acp/main.go | 14 | var version = "0.0.0-dev" dev-fallback — stamping-contract placeholder until Phase 7 release flow stamps a real value (intentional, plan-documented) | open |  | 2026-11-19T13:47:11.362Z |  |
| 4 | 02 | stub | cmd/zcode-acp/main.go | 19 | main() implements only --version — real CLI/agent arrive with Phases 3-4 per D-06 (intentional skeleton, plan-documented) | open |  | 2026-11-19T13:47:11.524Z |  |
| 5 | 03 | stub | internal/supervisor/discover.go | 769 | defaultOverride production seam is a no-op until plan 03-07 wires the wrapper's explicit root / ZCODE_BUILTIN_PROVIDER_CONFIG_FILE config surface (seam implemented and test-injectable) | open |  | 2026-11-19T17:58:49.323Z |  |
| 6 | 03 | unrun-verify | internal/supervisor/realdesktop_test.go | 81 | Real-desktop positive path TestRealDesktop is manual-only (build tag + env guard) — scheduled END-OF-PHASE human verification, never run by CI or this plan | open |  | 2026-11-19T20:32:33.515Z |  |
| 7 | 03 | deviation | internal/supervisor/e2e_test.go |  | plan-text adaptation: the cancel turn is replayed as a SECOND supervised cycle serving the cancel-mid-turn corpus row, not a second machine on the happy child's Conn — the fake's Times=0 rules are unlimited, so the happy-turn send rule re-fires turn.completed on a second send and a same-Conn cancel would race a completed resolution; both cycles end in the zero-survivors staged shutdown | open |  | 2026-11-19T21:18:34.164Z |  |
| 8 | 04 | stub | internal/acpport/doc.go | 12 | acpport seam package pre-placed as skeleton (doc.go + pin_test.go) — adapter surface (types/agent/sink) lands with plan 04-02 | fixed |  | 2026-11-19T23:46:45.534Z | 2026-11-20T00:22:27.097Z |
| 9 | 05 | deviation | experiments/scan-secrets.sh |  | [Rule 3] scan-secrets allowlist extended with the 05-06 self-declaring synthetic fixture marker (latent PR-gate failure) | open |  | 2026-11-20T15:16:11.286Z |  |
| 10 | 05 | deviation | experiments/scan-secrets.sh |  | [Rule 3] scan-secrets allowlist extended with the 05-06 self-declaring synthetic fixture marker (latent PR-gate failure) | open |  | 2026-11-20T15:16:17.803Z |  |
| 11 | 05 | unrun-verify | test/realdesktop/apicycle_test.go |  | END-OF-PHASE REAL RUN: TestAPIKeyFullCycle against the live z.ai api-key account (guarded tier, orchestrator-run, quota-ledgered per experiments/README.md Phase-5 row + 05-USER-SETUP.md) — the D-06 live-keyring decrypt and the registry read-back verdict are proven only by this run | open |  | 2026-11-20T15:56:11.062Z |  |
| 12 | 05 | unrun-verify | internal/acpfront/auth.go | 155 | branch-(c) corrected chain real-half pending: authenticate probe-create verdict proven at the fake tier only; the REAL RUN re-run is 05-12's human checkpoint (UAT item 1) | open |  | 2026-11-21T08:45:52.672Z |  |
| 13 | 05 | unrun-verify | internal/acpfront/auth.go | 155 | branch-(c) corrected chain real-half pending: authenticate probe-create verdict proven at the fake tier only; the REAL RUN re-run is 05-12's human checkpoint (UAT item 1) | open |  | 2026-11-21T08:45:58.318Z |  |
| 14 | 05 | unmet-truth | test/realdesktop/apicycle_test.go | 208 | REAL RUN attempt 2 (EXP-APIKEY-CYCLE-02): file-read tool turn refused by the desktop turn latch ('another prompt is already running') — the real-tier full cycle remains unproven; auth read-back leg field-proven; next gap plan must add a turn-latch drain between sends | open |  | 2026-11-21T10:08:55.578Z |  |
| 15 | 6 | unrun-verify | test/realdesktop/codingplan_test.go |  | TestCodingPlanFullCycle real-account run is the human-gated end-of-phase UAT (D-12/D-13): executor prepared harness+runbook+ledger row PENDING; the run executes under human control at phase verification, never autonomously, never CI | open |  | 2026-11-22T04:52:07.817Z |  |
| 16 | 7 | deviation | .planning/tdd/07-01-task1-red.json |  | Task-1 RED landed as the plan-sanctioned CONFIRM-UNREPRODUCED branch: both WR-05 tests pass pre-fix (window cannot land), classifier verdict INVALID_RED unexpected_green persisted in the record; tasks 2-3 produced genuine RED_EVIDENCE_OK REDs | open |  | 2026-11-22T11:36:17.784Z |  |
| 17 | 7 | unrun-verify | .planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md |  | QA-02 VM field run of scripts/vm-e2e.sh is the human-gated end-of-phase UAT (D-05): executor staged the copy-pasteable runbook + PENDING ledger row; the one documented VM execution happens under user control at phase verification, never autonomously | open |  | 2026-11-22T14:45:18.280Z |  |
| 18 | 7 | unrun-verify | .planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md |  | QA-03 Orca card E2E through BOTH auth paths is THE phase user gate (D-06): executor staged per-path checklists + observation items + PENDING ledger row; the real-account run happens under user control at phase verification, never autonomously | open |  | 2026-11-22T14:45:18.399Z |  |

````json
[
  {
    "id": 1,
    "kind": "deviation",
    "phase": "01",
    "file": "experiments/captures/EXP-TURN-01.jsonl",
    "line": 1,
    "description": "E2 turn-completion terminal signals are V-SRC-only (D-05 degradation): no completion envelope observed live; plan 01-03 must label affected claims V-SRC-only/V-RPT",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T06:32:58.823Z",
    "resolved_at": null,
    "milestone": null
  },
  {
    "id": 2,
    "kind": "deviation",
    "phase": "01",
    "file": "experiments/README.md",
    "line": null,
    "description": "Quota: 3 tiny sends issued vs 1 planned (two harness quiet-window mis-arms, fixed in-plan); 0 turns completed",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T06:32:59.065Z",
    "resolved_at": null,
    "milestone": null
  },
  {
    "id": 3,
    "kind": "stub",
    "phase": "02",
    "file": "cmd/zcode-acp/main.go",
    "line": 14,
    "description": "var version = \"0.0.0-dev\" dev-fallback — stamping-contract placeholder until Phase 7 release flow stamps a real value (intentional, plan-documented)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T13:47:11.362Z",
    "resolved_at": null,
    "milestone": null
  },
  {
    "id": 4,
    "kind": "stub",
    "phase": "02",
    "file": "cmd/zcode-acp/main.go",
    "line": 19,
    "description": "main() implements only --version — real CLI/agent arrive with Phases 3-4 per D-06 (intentional skeleton, plan-documented)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T13:47:11.524Z",
    "resolved_at": null,
    "milestone": null
  },
  {
    "id": 5,
    "kind": "stub",
    "phase": "03",
    "file": "internal/supervisor/discover.go",
    "line": 769,
    "description": "defaultOverride production seam is a no-op until plan 03-07 wires the wrapper's explicit root / ZCODE_BUILTIN_PROVIDER_CONFIG_FILE config surface (seam implemented and test-injectable)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T17:58:49.323Z",
    "resolved_at": null,
    "milestone": null
  },
  {
    "id": 6,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "internal/supervisor/realdesktop_test.go",
    "line": 81,
    "description": "Real-desktop positive path TestRealDesktop is manual-only (build tag + env guard) — scheduled END-OF-PHASE human verification, never run by CI or this plan",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T20:32:33.515Z",
    "resolved_at": null,
    "milestone": null
  },
  {
    "id": 7,
    "kind": "deviation",
    "phase": "03",
    "file": "internal/supervisor/e2e_test.go",
    "line": null,
    "description": "plan-text adaptation: the cancel turn is replayed as a SECOND supervised cycle serving the cancel-mid-turn corpus row, not a second machine on the happy child's Conn — the fake's Times=0 rules are unlimited, so the happy-turn send rule re-fires turn.completed on a second send and a same-Conn cancel would race a completed resolution; both cycles end in the zero-survivors staged shutdown",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T21:18:34.164Z",
    "resolved_at": null,
    "milestone": null
  },
  {
    "id": 8,
    "kind": "stub",
    "phase": "04",
    "file": "internal/acpport/doc.go",
    "line": 12,
    "description": "acpport seam package pre-placed as skeleton (doc.go + pin_test.go) — adapter surface (types/agent/sink) lands with plan 04-02",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-11-19T23:46:45.534Z",
    "resolved_at": "2026-11-20T00:22:27.097Z",
    "milestone": "v0.20.0"
  },
  {
    "id": 9,
    "kind": "deviation",
    "phase": "05",
    "file": "experiments/scan-secrets.sh",
    "line": null,
    "description": "[Rule 3] scan-secrets allowlist extended with the 05-06 self-declaring synthetic fixture marker (latent PR-gate failure)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-20T15:16:11.286Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 10,
    "kind": "deviation",
    "phase": "05",
    "file": "experiments/scan-secrets.sh",
    "line": null,
    "description": "[Rule 3] scan-secrets allowlist extended with the 05-06 self-declaring synthetic fixture marker (latent PR-gate failure)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-20T15:16:17.803Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 11,
    "kind": "unrun-verify",
    "phase": "05",
    "file": "test/realdesktop/apicycle_test.go",
    "line": null,
    "description": "END-OF-PHASE REAL RUN: TestAPIKeyFullCycle against the live z.ai api-key account (guarded tier, orchestrator-run, quota-ledgered per experiments/README.md Phase-5 row + 05-USER-SETUP.md) — the D-06 live-keyring decrypt and the registry read-back verdict are proven only by this run",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-20T15:56:11.062Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 12,
    "kind": "unrun-verify",
    "phase": "05",
    "file": "internal/acpfront/auth.go",
    "line": 155,
    "description": "branch-(c) corrected chain real-half pending: authenticate probe-create verdict proven at the fake tier only; the REAL RUN re-run is 05-12's human checkpoint (UAT item 1)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-21T08:45:52.672Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 13,
    "kind": "unrun-verify",
    "phase": "05",
    "file": "internal/acpfront/auth.go",
    "line": 155,
    "description": "branch-(c) corrected chain real-half pending: authenticate probe-create verdict proven at the fake tier only; the REAL RUN re-run is 05-12's human checkpoint (UAT item 1)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-21T08:45:58.318Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 14,
    "kind": "unmet-truth",
    "phase": "05",
    "file": "test/realdesktop/apicycle_test.go",
    "line": 208,
    "description": "REAL RUN attempt 2 (EXP-APIKEY-CYCLE-02): file-read tool turn refused by the desktop turn latch ('another prompt is already running') — the real-tier full cycle remains unproven; auth read-back leg field-proven; next gap plan must add a turn-latch drain between sends",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-21T10:08:55.578Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 15,
    "kind": "unrun-verify",
    "phase": "6",
    "file": "test/realdesktop/codingplan_test.go",
    "line": null,
    "description": "TestCodingPlanFullCycle real-account run is the human-gated end-of-phase UAT (D-12/D-13): executor prepared harness+runbook+ledger row PENDING; the run executes under human control at phase verification, never autonomously, never CI",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-22T04:52:07.817Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 16,
    "kind": "deviation",
    "phase": "7",
    "file": ".planning/tdd/07-01-task1-red.json",
    "line": null,
    "description": "Task-1 RED landed as the plan-sanctioned CONFIRM-UNREPRODUCED branch: both WR-05 tests pass pre-fix (window cannot land), classifier verdict INVALID_RED unexpected_green persisted in the record; tasks 2-3 produced genuine RED_EVIDENCE_OK REDs",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-22T11:36:17.784Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 17,
    "kind": "unrun-verify",
    "phase": "7",
    "file": ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md",
    "line": null,
    "description": "QA-02 VM field run of scripts/vm-e2e.sh is the human-gated end-of-phase UAT (D-05): executor staged the copy-pasteable runbook + PENDING ledger row; the one documented VM execution happens under user control at phase verification, never autonomously",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-22T14:45:18.280Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  },
  {
    "id": 18,
    "kind": "unrun-verify",
    "phase": "7",
    "file": ".planning/phases/07-hardening-vm-manual-e2e-documentation/07-UAT.md",
    "line": null,
    "description": "QA-03 Orca card E2E through BOTH auth paths is THE phase user gate (D-06): executor staged per-path checklists + observation items + PENDING ledger row; the real-account run happens under user control at phase verification, never autonomously",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-22T14:45:18.399Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  }
]
````
