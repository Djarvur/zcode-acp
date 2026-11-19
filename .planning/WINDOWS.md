---
schema_version: 1
open_count: 8
waived_count: 0
fixed_count: 0
total_count: 8
last_updated: 2026-11-19T23:46:45.534Z
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
| 8 | 04 | stub | internal/acpport/doc.go | 12 | acpport seam package pre-placed as skeleton (doc.go + pin_test.go) — adapter surface (types/agent/sink) lands with plan 04-02 | open |  | 2026-11-19T23:46:45.534Z |  |

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
    "status": "open",
    "reason": "",
    "recorded_at": "2026-11-19T23:46:45.534Z",
    "resolved_at": null,
    "milestone": "v0.20.0"
  }
]
````
