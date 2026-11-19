---
schema_version: 1
open_count: 4
waived_count: 0
fixed_count: 0
total_count: 4
last_updated: 2026-11-19T13:47:11.524Z
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
  }
]
````
