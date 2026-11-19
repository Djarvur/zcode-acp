# Phase 3: ZCode Back Leg - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.

**Date:** 2026-11-19
**Phase:** 3-ZCode Back Leg — Codec, Fake App-Server, Supervisor
**Mode:** `--auto` (standing directive: proceed automatically, stop only when unavoidable)

---

## Fake app-server architecture

| Option | Description | Selected |
|--------|-------------|----------|
| In-process (io.Pipe) primary + standalone binary; JSON scenario files; golden transcripts in testdata | ARCHITECTURE/SUMMARY recommendation; CI-friendly (~90% of tests, race-detector-friendly) | ✓ |
| Standalone binary only | Slower CI, no race-detector reach into the fake | |
| Programmatic-only scripting | Golden transcripts harder to author/review | |

**User's choice:** *(auto)* In-process primary + binary + JSON scenarios

## Turn completion detection

| Option | Description | Selected |
|--------|-------------|----------|
| Multi-signal: terminal event primary + quiet-window fallback + bypass-lane cancel; TurnWatcher interface (polling first) | Matches dialect truth (V-SRC terminal taxonomy) and the only 3.18.2-proven path (polling) | ✓ |
| Subscribe-only | Unproven end-to-end as the sole path; slot reserved | |

**User's choice:** *(auto)* Multi-signal + interface slot for subscribe

## Supervisor policy

| Option | Description | Selected |
|--------|-------------|----------|
| Phase-1 matrix discovery; scrubbed proxy env; Setpgid group-kill + Wait; crash-window 3/60s; loud no-install diagnostics | PITFALLS 6/9 + codex-acp precedent | ✓ |
| Per-session children | Re-pushes registry, re-mints PATs N times — rejected by ARCHITECTURE | |

**User's choice:** *(auto)* Shared child + documented restart policy

## Session multiplexing / host obligations / layout

| Decision | Choice |
|----------|--------|
| Multiplexing | 1:1 sessionIds, shared child (ARCHITECTURE locked, re-affirmed) |
| requestRuntimePreferences | Static defaults answer, 15s deadline timer now; full responder Phase 5; requestCredentialCipher NOT answered (Phase 6) |
| Module layout | internal/zcode (+fake), internal/supervisor, turn machine in-package |

## Claude's Discretion
- Scenario schema, fixture granularity, queue internals, test structuring

## Deferred Ideas
- Subscribe TurnWatcher (Phase 4+); full responder policy (Phase 5); session/list+load (v2)
